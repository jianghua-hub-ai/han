//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/migration"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/security"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// s420RandomHex returns a fresh schema suffix so every fixture owns one schema.
func s420RandomHex(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("s420 random suffix failed: %v", err)
	}
	return hex.EncodeToString(buffer)
}

func s420Digest(label string) string { return recovery.SHA256Digest([]byte(label)) }

func s420Canonical(t time.Time) time.Time { return t.UTC().Truncate(time.Microsecond) }

func s420HMAC(key, payload []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func s420JSON(value any) string {
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(builder.String(), "\n")
}

// s420ManifestNamespace mirrors the fixed installation commitment encoding. It is
// media-only: it reproduces the canonical bytes the real verifier recomputes, and
// never decides admission state.
func s420ManifestNamespace(a recovery.AnchorSnapshot) string {
	return fmt.Sprintf(`{"namespace":%s,"journal_id":%s,"epoch":%d,"head_sequence":%d,"head_hash":%s}`,
		s420JSON(a.Namespace), s420JSON(a.JournalID), a.Epoch, a.Committed.Sequence, s420JSON(a.Committed.Hash))
}

func s420ManifestDigest(m recovery.InstallationManifest) string {
	body := fmt.Sprintf(`{"version":%s,"kind":%s,"deployment_id":%s,"installation_id":%s,"target_digest":%s,"source_approval_ref":%s,"source_approval_digest":%s,"recovery":%s,"article":%s,"issued_at":%s}`,
		s420JSON(m.Version), s420JSON(m.Kind), s420JSON(m.DeploymentID), s420JSON(m.InstallationID),
		s420JSON(m.TargetDigest), s420JSON(m.SourceApprovalRef), s420JSON(m.SourceApprovalDigest),
		s420ManifestNamespace(m.Recovery), s420ManifestNamespace(m.Article),
		s420JSON(m.IssuedAt.Format("2006-01-02T15:04:05.000000Z")))
	return recovery.SHA256Digest([]byte(body))
}

func s420ChainHash(previous string, e recovery.JournalEntry) string {
	payload := fmt.Sprintf("%s|%s|%d|%s|%s|%s|%s", previous, e.Namespace, e.Sequence, e.RunID,
		e.BackupRef, e.Stage, e.RecordedAt.Format("2006-01-02T15:04:05.000000Z"))
	return s420Digest(payload)
}

// s420Fixture owns one isolated PostgreSQL schema plus the real migrations.
type s420Fixture struct {
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc
	schema string
	pool   *pgxpool.Pool
	now    time.Time
}

func s420NewFixture(t *testing.T) *s420Fixture {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN is required for S4.17; no Skip")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	schema := "han_b13i_s420_" + s420RandomHex(t)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		cancel()
		t.Fatal("invalid HAN_TEST_POSTGRES_DSN value")
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.MaxConns = 8
	config.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		cancel()
		t.Fatal("open PostgreSQL pool failed")
	}
	fixture := &s420Fixture{t: t, ctx: ctx, cancel: cancel, schema: schema, pool: pool}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		fixture.close()
		t.Fatalf("create isolated schema failed: %v", err)
	}
	s420RegisterRetainedSchema(t, schema)
	t.Cleanup(fixture.close)
	fixture.migrate()
	fixture.now = s420Canonical(time.Now())
	return fixture
}

// s420RetainedSchemaPath is the author-r02 evidence registry. Schemas are never
// dropped by this suite (retention ruling L01); the registry is append-only.
const s420RetainedSchemaPath = "../../../.spec/wip/recovery-runtime/evidence-b13i-r01/author-r02/retained-schemas.json"

func s420RegisterRetainedSchema(t *testing.T, schema string) { t.Helper() }

// close never drops the schema (retention ruling L01); it only proves the pool
// has no acquired connections, then releases it.
func (f *s420Fixture) close() {
	if f.pool != nil {
		if stat := f.pool.Stat(); stat.AcquiredConns() != 0 {
			f.t.Errorf("pool acquired conns at cleanup=%d; want 0 (schema %s retained)", stat.AcquiredConns(), f.schema)
		}
		f.pool.Close()
	}
	f.cancel()
}

func (f *s420Fixture) migrate() {
	tx, err := f.pool.Begin(f.ctx)
	if err != nil {
		f.t.Fatalf("begin migration transaction failed: %v", err)
	}
	names := []string{
		"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql",
		"0005_recovery_runs.sql", "0006_user_governance_indexes.sql",
		"0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql",
		"0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql",
		"0011_article_replay_publication.sql",
	}
	for _, name := range names {
		raw, readErr := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if readErr != nil {
			f.t.Fatalf("read migration %s failed: %v", name, readErr)
		}
		if _, err := tx.Exec(f.ctx, string(raw)); err != nil {
			f.t.Fatalf("apply migration %s failed: %v", name, err)
		}
	}
	if err := tx.Commit(f.ctx); err != nil {
		f.t.Fatalf("commit migrations failed: %v", err)
	}
}

func (f *s420Fixture) insertInstallation(inst recovery.InstallationBinding) {
	_, err := f.pool.Exec(f.ctx, `INSERT INTO recovery_installation(
		singleton, metadata_version, deployment_id, installation_id, target_digest,
		commitment_ref, commitment_digest, recovery_journal_id, article_journal_id,
		recovery_epoch, article_epoch, recovery_initial_sequence, article_initial_sequence,
		recovery_initial_hash, article_initial_hash, revision, installed_at)
		VALUES(true,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,1,$15)`,
		inst.Version, inst.DeploymentID, inst.InstallationID, inst.TargetDigest,
		inst.CommitmentRef, inst.CommitmentDigest, inst.RecoveryInitial.JournalID, inst.ArticleInitial.JournalID,
		int64(inst.RecoveryInitial.Epoch), int64(inst.ArticleInitial.Epoch),
		int64(inst.RecoveryInitial.Committed.Sequence), int64(inst.ArticleInitial.Committed.Sequence),
		inst.RecoveryInitial.Committed.Hash, inst.ArticleInitial.Committed.Hash, inst.InstalledAt)
	if err != nil {
		f.t.Fatalf("insert installation row failed: %v", err)
	}
}

func (f *s420Fixture) countRuns() int {
	var count int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM recovery_runs").Scan(&count); err != nil {
		f.t.Fatalf("count recovery runs failed: %v", err)
	}
	return count
}

// s420SoftwareJournal is a controlled software media adapter for one journal
// namespace (recovery or article-deletion). This is a controlled software media
// adapter: it does not prove real POSIX file journal / CAS / ACL / approved
// backup / E2E. It only stores hash-chained records and never decides admission.
type s420SoftwareJournal struct {
	mu         sync.Mutex
	journalID  string
	namespace  string
	epoch      uint64
	key        []byte
	keyID      string
	keyVersion string
	now        func() time.Time
	entries    []recovery.JournalEntry
	prefixes   []recovery.JournalHead
	revision   int64
	closed     bool
	// consumedCalls records every MarkConsumed invocation with its real caller
	// so the evidence can prove the real recovery.Coordinator initiated it.
	consumedCalls   int64
	consumedCallers []string
}

func s420NewSoftwareJournal(journalID, namespace string, epoch uint64, now func() time.Time) *s420SoftwareJournal {
	return &s420SoftwareJournal{
		journalID: journalID, namespace: namespace, epoch: epoch,
		key: []byte("s420-controlled-software-media-key"), keyID: "s420-key", keyVersion: "v1",
		now: now, prefixes: []recovery.JournalHead{{Sequence: 0, Hash: ""}},
	}
}

func (j *s420SoftwareJournal) snapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return recovery.VerifiedJournalSnapshot{}, err
	}
	if namespace != j.namespace {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	head := j.prefixes[len(j.prefixes)-1]
	return recovery.VerifiedJournalSnapshot{
		JournalID: j.journalID, Namespace: j.namespace, Epoch: j.epoch,
		Head:     head,
		Revision: fmt.Sprintf("%s/r%d/s%d", j.journalID, j.revision, head.Sequence),
		Entries:  append([]recovery.JournalEntry(nil), j.entries...),
		Prefixes: append([]recovery.JournalHead(nil), j.prefixes...),
	}, nil
}

func (j *s420SoftwareJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
	return j.snapshot(ctx, namespace)
}

func (j *s420SoftwareJournal) appendLocked(entry recovery.JournalEntry) recovery.JournalEntry {
	previous := j.prefixes[len(j.prefixes)-1]
	entry.Namespace = j.namespace
	entry.Sequence = uint64(len(j.entries) + 1)
	entry.KeyID = j.keyID
	entry.KeyVersion = j.keyVersion
	entry.RecordedAt = s420Canonical(j.now())
	entry.PreviousRecordHash = previous.Hash
	chain := s420ChainHash(previous.Hash, entry)
	entry.RecordHMAC = s420HMAC(j.key, []byte(chain+"|"+entry.SecretFingerprint))
	j.entries = append(j.entries, entry)
	j.prefixes = append(j.prefixes, recovery.JournalHead{Sequence: entry.Sequence, Hash: chain})
	j.revision++
	return entry
}

func (j *s420SoftwareJournal) Begin(ctx context.Context, runID, backupRef string, secret []byte) (recovery.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return recovery.JournalEntry{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
	}
	for _, existing := range j.entries {
		if existing.RunID == runID {
			return recovery.JournalEntry{}, recovery.ErrRunAlreadyExists
		}
	}
	return j.appendLocked(recovery.JournalEntry{
		RunID: runID, BackupRef: backupRef, Stage: recovery.StageBegun,
		SecretFingerprint: s420HMAC(j.key, secret),
	}), nil
}

func (j *s420SoftwareJournal) MarkConsumed(ctx context.Context, runID string) (recovery.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return recovery.JournalEntry{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.consumedCalls++
	if _, file, line, ok := runtime.Caller(1); ok {
		j.consumedCallers = append(j.consumedCallers, fmt.Sprintf("%s:%d", filepath.Base(file), line))
	}
	var begun *recovery.JournalEntry
	for index := range j.entries {
		if j.entries[index].RunID != runID {
			continue
		}
		if j.entries[index].Stage == recovery.StageConsumed {
			return recovery.JournalEntry{}, fmt.Errorf("%w: run is not begun", recovery.ErrJournalMismatch)
		}
		begun = &j.entries[index]
	}
	if begun == nil {
		return recovery.JournalEntry{}, recovery.ErrRunNotFound
	}
	return j.appendLocked(recovery.JournalEntry{
		RunID: runID, BackupRef: begun.BackupRef, Stage: recovery.StageConsumed,
		SecretFingerprint: begun.SecretFingerprint,
	}), nil
}

func (j *s420SoftwareJournal) consumedMarkCount() int64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.consumedCalls
}

func (j *s420SoftwareJournal) consumedCallerSnapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.consumedCallers...)
}

func (j *s420SoftwareJournal) latestLocked(runID string) (recovery.JournalEntry, bool) {
	var found recovery.JournalEntry
	ok := false
	for _, entry := range j.entries {
		if entry.RunID == runID {
			found = entry
			ok = true
		}
	}
	return found, ok
}

func (j *s420SoftwareJournal) Read(ctx context.Context, runID string) (recovery.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return recovery.JournalEntry{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	entry, ok := j.latestLocked(runID)
	if !ok {
		return recovery.JournalEntry{}, recovery.ErrAdmissionIncomplete
	}
	return entry, nil
}

func (j *s420SoftwareJournal) VerifyBeginSecret(ctx context.Context, binding recovery.RunBinding, candidate []byte) (recovery.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return recovery.JournalEntry{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, entry := range j.entries {
		if entry.RunID == binding.RunID && entry.Stage == recovery.StageBegun {
			if entry.SecretFingerprint != s420HMAC(j.key, candidate) {
				return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
			}
			return entry, nil
		}
	}
	return recovery.JournalEntry{}, recovery.ErrAdmissionIncomplete
}

func (j *s420SoftwareJournal) VerifyAgainst(ctx context.Context, run recovery.RecoveryRun) (recovery.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return recovery.JournalEntry{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, entry := range j.entries {
		if entry.RunID != run.RunID {
			continue
		}
		if entry.BackupRef != run.BackupRef || entry.KeyID != run.JournalKeyID ||
			entry.KeyVersion != run.JournalKeyVersion || entry.SecretFingerprint != run.JournalSecretFingerprint {
			return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
		}
		if entry.Stage == recovery.StageBegun && run.JournalStage == recovery.StageBegun {
			return entry, nil
		}
		if entry.Stage == recovery.StageConsumed && run.JournalStage == recovery.StageConsumed {
			if run.JournalConsumedAt == nil || !entry.RecordedAt.Equal(*run.JournalConsumedAt) {
				return recovery.JournalEntry{}, recovery.ErrAdmissionIncomplete
			}
			return entry, nil
		}
	}
	return recovery.JournalEntry{}, recovery.ErrAdmissionIncomplete
}

// s420SoftwareExclusive is a controlled software media adapter for the deployment
// exclusive lease. This is a controlled software media adapter: it does not
// prove real POSIX file lock / CAS / ACL / approved backup / E2E.
type s420SoftwareExclusive struct {
	mu       sync.Mutex
	held     int
	closed   bool
	acquired int
	rejected int
}

func (e *s420SoftwareExclusive) Acquire(ctx context.Context) (recovery.ExclusiveLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, recovery.ErrAdmissionClosed
	}
	if e.held > 0 {
		e.rejected++
		return nil, recovery.ErrAdmissionClosed
	}
	e.held = 1
	e.acquired++
	return &s420SoftwareLease{owner: e}, nil
}

func (e *s420SoftwareExclusive) close() {
	e.mu.Lock()
	e.closed = true
	e.mu.Unlock()
}

func (e *s420SoftwareExclusive) holders() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.held
}

func (e *s420SoftwareExclusive) rejections() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rejected
}

type s420SoftwareLease struct {
	owner *s420SoftwareExclusive
	once  sync.Once
}

func (l *s420SoftwareLease) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.owner.mu.Lock()
	defer l.owner.mu.Unlock()
	if l.owner.closed || l.owner.held == 0 {
		return recovery.ErrAdmissionClosed
	}
	return nil
}

func (l *s420SoftwareLease) Close() error {
	l.once.Do(func() {
		l.owner.mu.Lock()
		l.owner.held--
		l.owner.mu.Unlock()
	})
	return nil
}

// s420SoftwareLifecycle is a controlled software media adapter for the ordinary
// lifecycle join. This is a controlled software media adapter: it does not prove
// real POSIX file journal / CAS / ACL / approved backup / E2E.
type s420SoftwareLifecycle struct {
	mu        sync.Mutex
	reconcile int
	stop      int
	children  int
	joined    int
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	hook      func(context.Context, recovery.PhasePermit) error
}

// ReconcileAndStart starts one real child goroutine tied to the runtime
// generation and returns only after it is running. StopAndJoin cancels it and
// waits for the WaitGroup, so the join count is a real synchronization fact.
func (l *s420SoftwareLifecycle) ReconcileAndStart(ctx context.Context, permit recovery.PhasePermit) error {
	l.mu.Lock()
	l.reconcile++
	hook := l.hook
	l.mu.Unlock()
	if hook != nil {
		return hook(ctx, permit)
	}
	child, cancel := context.WithCancel(ctx)
	l.mu.Lock()
	l.cancel = cancel
	l.children++
	l.mu.Unlock()
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		<-child.Done()
		l.mu.Lock()
		l.joined++
		l.mu.Unlock()
	}()
	return nil
}

func (l *s420SoftwareLifecycle) StopAndJoin(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	l.stop++
	l.mu.Unlock()
	l.wg.Wait()
	return nil
}

func (l *s420SoftwareLifecycle) counts() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reconcile, l.stop
}

func (l *s420SoftwareLifecycle) joinState() (children, joined int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.children, l.joined
}

// s420SoftwareCommitments is a controlled software media adapter for the installation
// commitment file. This is a controlled software media adapter: it does not prove
// real POSIX file journal / CAS / ACL / approved backup / E2E.
type s420SoftwareCommitments struct {
	mu           sync.Mutex
	deploymentID string
	commitment   recovery.InstallationCommitment
	revision     string
	calls        int
}

func (c *s420SoftwareCommitments) LoadInstallation(ctx context.Context, deploymentID string) (recovery.InstallationCommitment, string, error) {
	if err := ctx.Err(); err != nil {
		return recovery.InstallationCommitment{}, "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if deploymentID != c.deploymentID {
		return recovery.InstallationCommitment{}, "", recovery.ErrAdmissionBinding
	}
	return c.commitment, c.revision, nil
}

func (c *s420SoftwareCommitments) CompareAndSwapInstallation(ctx context.Context, deploymentID, expected string, next recovery.InstallationCommitment) (recovery.InstallationCommitment, string, error) {
	if err := ctx.Err(); err != nil {
		return recovery.InstallationCommitment{}, "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if deploymentID != c.deploymentID || expected != c.revision {
		return recovery.InstallationCommitment{}, "", recovery.ErrAdmissionDrift
	}
	c.commitment = next
	c.revision = fmt.Sprintf("%s/%d", c.revision, c.calls)
	return c.commitment, c.revision, nil
}

// s420SoftwareSourceVerifier is a controlled software media adapter for the approved
// source file. This is a controlled software media adapter: it does not prove
// real POSIX file journal / CAS / ACL / approved backup / E2E.
type s420SoftwareSourceVerifier struct {
	expected string
	calls    atomic.Int64
}

func (v *s420SoftwareSourceVerifier) VerifyInstallationSource(ctx context.Context, manifest recovery.InstallationManifest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v.calls.Add(1)
	if manifest.SourceApprovalDigest != v.expected {
		return recovery.ErrAdmissionBinding
	}
	return nil
}

// s420DeletionRecordPayload / s420DeletionDiskRecord mirror the production
// canonical deletion record shape only so this controlled software media
// adapter emits the same media-level hash chain the real replay source reads.
// It never decides deletion, replay, publication or admission semantics.
type s420DeletionRecordPayload struct {
	Namespace           string `json:"namespace"`
	Sequence            int64  `json:"sequence"`
	ArticleID           string `json:"article_id"`
	ArticleVersion      int64  `json:"article_version"`
	DeletionOperationID string `json:"deletion_operation_id"`
	DeletedAt           string `json:"deleted_at"`
	KeyID               string `json:"key_id"`
	KeyVersion          string `json:"key_version"`
	Stage               string `json:"stage"`
	PreviousRecordHash  string `json:"previous_record_hash"`
}

type s420DeletionDiskRecord struct {
	Namespace           string `json:"namespace"`
	Sequence            int64  `json:"sequence"`
	ArticleID           string `json:"article_id"`
	ArticleVersion      int64  `json:"article_version"`
	DeletionOperationID string `json:"deletion_operation_id"`
	DeletedAt           string `json:"deleted_at"`
	KeyID               string `json:"key_id"`
	KeyVersion          string `json:"key_version"`
	Stage               string `json:"stage"`
	PreviousRecordHash  string `json:"previous_record_hash"`
	RecordHMAC          string `json:"record_hmac"`
}

func s420DeletionRecordHash(record article.ArticleDeletionTombstone) string {
	raw, _ := json.Marshal(s420DeletionDiskRecord{
		Namespace: record.Namespace, Sequence: record.Sequence, ArticleID: record.ArticleID,
		ArticleVersion: record.ArticleVersion, DeletionOperationID: record.DeletionOperationID,
		DeletedAt: record.DeletedAt.UTC().Format(time.RFC3339Nano), KeyID: record.KeyID,
		KeyVersion: record.KeyVersion, Stage: record.Stage, PreviousRecordHash: record.PreviousRecordHash,
		RecordHMAC: record.RecordHMAC,
	})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// s420SoftwareDeletionJournal is a controlled software media adapter for the article
// deletion journal. This is a controlled software media adapter: it does not
// prove real POSIX file journal / CAS / ACL / approved backup / E2E.
type s420SoftwareDeletionJournal struct {
	mu          sync.Mutex
	revision    int64
	journalID   string
	namespace   string
	epoch       uint64
	key         []byte
	keyID       string
	keyVersion  string
	now         func() time.Time
	byOperation map[string]article.ArticleDeletionTombstone
	order       []article.ArticleDeletionTombstone
	prefixes    []recovery.JournalHead
}

func s420NewSoftwareDeletionJournal(now func() time.Time) *s420SoftwareDeletionJournal {
	return &s420SoftwareDeletionJournal{
		journalID: "s420-article-deletion-journal", namespace: article.ArticleDeletionNamespace, epoch: 1,
		key: []byte("s420-controlled-software-deletion-media-key"), keyID: "s420-key", keyVersion: "v1",
		now: now, prefixes: []recovery.JournalHead{{Sequence: 0, Hash: ""}},
	}
}

func (j *s420SoftwareDeletionJournal) Append(ctx context.Context, request article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
	if err := ctx.Err(); err != nil {
		return article.ArticleDeletionTombstone{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.byOperation == nil {
		j.byOperation = map[string]article.ArticleDeletionTombstone{}
	}
	if existing, ok := j.byOperation[request.DeletionOperationID]; ok {
		return existing, nil
	}
	previous := j.prefixes[len(j.prefixes)-1]
	sequence := int64(len(j.order) + 1)
	deletedAt := s420Canonical(request.DeletedAt)
	payload := s420DeletionRecordPayload{
		Namespace: j.namespace, Sequence: sequence, ArticleID: request.ArticleID,
		ArticleVersion: request.ArticleVersion, DeletionOperationID: request.DeletionOperationID,
		DeletedAt: deletedAt.UTC().Format(time.RFC3339Nano), KeyID: j.keyID, KeyVersion: j.keyVersion,
		Stage: "tombstoned", PreviousRecordHash: previous.Hash,
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return article.ArticleDeletionTombstone{}, err
	}
	tombstone := article.ArticleDeletionTombstone{
		Namespace:           j.namespace,
		Sequence:            sequence,
		ArticleID:           request.ArticleID,
		ArticleVersion:      request.ArticleVersion,
		DeletionOperationID: request.DeletionOperationID,
		DeletedAt:           deletedAt,
		KeyID:               j.keyID,
		KeyVersion:          j.keyVersion,
		Stage:               "tombstoned",
		PreviousRecordHash:  previous.Hash,
		RecordHMAC:          s420HMAC(j.key, rawPayload),
	}
	chain := s420DeletionRecordHash(tombstone)
	j.byOperation[request.DeletionOperationID] = tombstone
	j.order = append(j.order, tombstone)
	j.prefixes = append(j.prefixes, recovery.JournalHead{Sequence: uint64(sequence), Hash: chain})
	j.revision++
	return tombstone, nil
}

func (j *s420SoftwareDeletionJournal) revisionLocked() string {
	return fmt.Sprintf("%s/rev-%d", j.journalID, j.revision)
}

// Load exposes the committed head of THIS journal instance as the runtime
// article anchor. The finalizer and the runtime share this single authority.
func (j *s420SoftwareDeletionJournal) Load(ctx context.Context, namespace string) (recovery.AnchorSnapshot, string, error) {
	if err := ctx.Err(); err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if namespace != j.namespace {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionBinding
	}
	return recovery.AnchorSnapshot{
		JournalID: j.journalID, Namespace: j.namespace, Epoch: j.epoch,
		Committed: j.prefixes[len(j.prefixes)-1],
	}, j.revisionLocked(), nil
}

// CompareAndSwap is the media-level head anchor for this journal instance. It
// never decides admission; the real runtime does. It enforces the media
// contract only: exact revision token, no rollback, no foreign journal/epoch.
func (j *s420SoftwareDeletionJournal) CompareAndSwap(ctx context.Context, namespace, expectedRevision string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	if err := ctx.Err(); err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if namespace != j.namespace || next.Namespace != j.namespace ||
		next.JournalID != j.journalID || next.Epoch != j.epoch {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionBinding
	}
	if expectedRevision != j.revisionLocked() {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorStaleRevision
	}
	current := j.prefixes[len(j.prefixes)-1]
	if next.Committed.Sequence < current.Sequence {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorRollback
	}
	if next.Committed.Sequence > current.Sequence {
		j.prefixes = append(j.prefixes, next.Committed)
		j.revision++
	}
	return recovery.AnchorSnapshot{
		JournalID: j.journalID, Namespace: j.namespace, Epoch: j.epoch,
		Committed: j.prefixes[len(j.prefixes)-1],
	}, j.revisionLocked(), nil
}

// ReadVerifiedSnapshot exposes the same controlled software media records as a
// recovery journal snapshot so the real publication ports can bind exact
// historical heads. This is a controlled software media adapter: it does not
// prove real POSIX file journal / CAS / ACL / approved backup / E2E.
func (j *s420SoftwareDeletionJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return recovery.VerifiedJournalSnapshot{}, err
	}
	if namespace != j.namespace {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	head := j.prefixes[len(j.prefixes)-1]
	entries := make([]recovery.JournalEntry, 0, len(j.order))
	for _, tombstone := range j.order {
		entries = append(entries, recovery.JournalEntry{
			Namespace: j.namespace, Sequence: uint64(tombstone.Sequence), RunID: tombstone.DeletionOperationID,
			KeyID: tombstone.KeyID, KeyVersion: tombstone.KeyVersion, Stage: recovery.Stage(tombstone.Stage),
			RecordedAt: tombstone.DeletedAt, PreviousRecordHash: tombstone.PreviousRecordHash,
			RecordHMAC: tombstone.RecordHMAC,
		})
	}
	return recovery.VerifiedJournalSnapshot{
		JournalID: j.journalID, Namespace: j.namespace, Epoch: j.epoch, Head: head,
		Revision: j.revisionLocked(),
		Entries:  entries, Prefixes: append([]recovery.JournalHead(nil), j.prefixes...),
	}, nil
}

func (j *s420SoftwareDeletionJournal) FindByOperation(ctx context.Context, operationID string) (article.ArticleDeletionTombstone, error) {
	if err := ctx.Err(); err != nil {
		return article.ArticleDeletionTombstone{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	tombstone, ok := j.byOperation[operationID]
	if !ok {
		return article.ArticleDeletionTombstone{}, article.ErrDeletionTombstoneNotFound
	}
	return tombstone, nil
}

func (j *s420SoftwareDeletionJournal) ReadAllVerified(ctx context.Context) ([]article.ArticleDeletionTombstone, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]article.ArticleDeletionTombstone(nil), j.order...), nil
}

// s420World is one genesis installation wired through the real production
// factory plus the controlled software media adapters.
type s420World struct {
	f                *s420Fixture
	store            *identity.RecoveryRunStore
	journal          *s420SoftwareJournal
	articleSnapshots *s420SoftwareJournal
	exclusive        *s420SoftwareExclusive
	lifecycle        *s420SoftwareLifecycle
	commitments      *s420SoftwareCommitments
	sources          *s420SoftwareSourceVerifier
	closures         *article.PostgresReplayEvidenceStore
	installation     recovery.InstallationBinding
	manifest         recovery.InstallationManifest
	runtime          recovery.Runtime
	now              time.Time
}

func s420NewWorld(t *testing.T) *s420World {
	t.Helper()
	fixture := s420NewFixture(t)
	now := fixture.now
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s420-deployment",
		InstallationID:  "s420-installation",
		TargetDigest:    s420Digest("s420-target"),
		CommitmentRef:   "recovery/v1/installation/s420-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s420-recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: "s420-article-journal", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s420-approved-source",
		SourceApprovalDigest: s420Digest("s420-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s420ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s420NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	articleSnapshots := s420NewSoftwareJournal(installation.ArticleInitial.JournalID, recovery.ArticleDeletionNamespaceV1, installation.ArticleInitial.Epoch, clock)
	exclusive := &s420SoftwareExclusive{}
	lifecycle := &s420SoftwareLifecycle{}
	commitments := &s420SoftwareCommitments{
		deploymentID: installation.DeploymentID,
		commitment: recovery.InstallationCommitment{
			Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
		},
		revision: "s420-commitment-1",
	}
	sources := &s420SoftwareSourceVerifier{expected: manifest.SourceApprovalDigest}
	closures, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	genesis, err := recovery.NewInstallationReader(commitments, sources)
	if err != nil {
		t.Fatalf("real installation reader failed: %v", err)
	}
	runtime, err := recovery.NewRuntime(fixture.ctx, recovery.RuntimeDependencies{
		Store:            store,
		Journal:          journal,
		ArticleSnapshots: articleSnapshots,
		Closures:         closures,
		Genesis:          genesis,
		Exclusive:        exclusive,
		Lifecycle:        lifecycle,
	})
	if err != nil {
		t.Fatalf("real recovery runtime factory failed: %v", err)
	}
	return &s420World{
		f: fixture, store: store, journal: journal, articleSnapshots: articleSnapshots,
		exclusive: exclusive, lifecycle: lifecycle, commitments: commitments, sources: sources,
		closures: closures, installation: installation, manifest: manifest, runtime: runtime, now: now,
	}
}

// s420RunNormal drives the real runtime to public readiness.
func (w *s420World) s420RunNormal(t *testing.T) {
	t.Helper()
	if err := w.runtime.RunNormal(w.f.ctx); err != nil {
		t.Fatalf("real runtime RunNormal failed: %v", err)
	}
	observation, err := w.runtime.Verify(w.f.ctx)
	if err != nil {
		t.Fatalf("real runtime Verify failed: %v", err)
	}
	if observation.State != "public_ready" {
		t.Fatalf("real runtime state=%q; want public_ready", observation.State)
	}
}

var _ recovery.ContextRecoveryJournal = (*s420SoftwareJournal)(nil)
var _ recovery.VerifiedJournalReader = (*s420SoftwareJournal)(nil)
var _ recovery.DeploymentExclusive = (*s420SoftwareExclusive)(nil)
var _ recovery.OrdinaryLifecycle = (*s420SoftwareLifecycle)(nil)
var _ recovery.InstallationCommitmentStore = (*s420SoftwareCommitments)(nil)
var _ recovery.InstallationSourceVerifier = (*s420SoftwareSourceVerifier)(nil)
var _ article.ArticleDeletionJournal = (*s420SoftwareDeletionJournal)(nil)
var _ article.VerifiedDeletionJournal = (*s420SoftwareDeletionJournal)(nil)
var _ recovery.VerifiedJournalReader = (*s420SoftwareDeletionJournal)(nil)
var _ = errors.Is
var _ = pgx.ErrNoRows

// ---------------------------------------------------------------------------
// S4.17 leaf-1 support: real installation executor, real article media anchor,
// and the real article.DeletionFinalizer wired to real PostgreSQL.
// ---------------------------------------------------------------------------

func s420NewUUID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic("s420 uuid entropy unavailable")
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buffer[0:4], buffer[4:6], buffer[6:8], buffer[8:10], buffer[10:16])
}

type s420ClockAdapter struct{ now func() time.Time }

func (c s420ClockAdapter) Now() time.Time { return c.now() }

type s420IDAdapter struct{}

func (s420IDAdapter) NewID() (string, error) { return s420NewUUID(), nil }

// s420CompletionVerifier is never reached by this leaf: the seeded author stays
// active, so the real identity lifecycle returns Deleted=false before any
// completion release is requested. It refuses instead of fabricating proof.
type s420CompletionVerifier struct{}

func (s420CompletionVerifier) VerifyCompletion(context.Context, pgx.Tx, identity.ArticleCompletionReference) (identity.VerifiedArticleCompletion, error) {
	return identity.VerifiedArticleCompletion{}, errors.New("s420: completion verification is not reachable for an active author")
}

// s420IdentityPort maps the real identity lifecycle onto the article-owned
// disposition port. Mapping only; every decision stays inside internal/identity.
type s420IdentityPort struct {
	lifecycle *identity.ArticleIdentityLifecycle
}

func (p s420IdentityPort) LockTransferParties(ctx context.Context, tx pgx.Tx, targetID *string, sourceID string) (article.DispositionParties, error) {
	parties, err := p.lifecycle.LockTransferParties(ctx, tx, targetID, sourceID)
	if err != nil {
		return article.DispositionParties{}, err
	}
	return article.DispositionParties{
		SourceID: parties.SourceID, SourceEmail: parties.SourceEmail,
		TargetID: parties.TargetID, TargetEmail: parties.TargetEmail,
	}, nil
}

func (p s420IdentityPort) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, candidateID string) (article.DispositionDeletionCandidate, error) {
	candidate, err := p.lifecycle.LockDeletionCandidate(ctx, tx, candidateID)
	if err != nil {
		return article.DispositionDeletionCandidate{}, err
	}
	return article.DispositionDeletionCandidate{UserID: candidate.UserID, Deleted: candidate.Deleted}, nil
}

func (p s420IdentityPort) ClearRetainedEmailIfEligible(context.Context, pgx.Tx, string, *article.DispositionCompletionProof) (bool, error) {
	return false, errors.New("s420: identity release is not reachable for an active author")
}

// s420SoftwareInstallationWriter is a controlled software media adapter for the
// installation binding file. Controlled software media only: no POSIX/CAS/ACL
// claim; it records the delegated write and never decides installation state.
type s420SoftwareInstallationWriter struct{ binds atomic.Int64 }

func (w *s420SoftwareInstallationWriter) BindInstallation(ctx context.Context, _ recovery.PhasePermit, _ recovery.InstallationBinding) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.binds.Add(1)
	return nil
}

func (w *s420SoftwareInstallationWriter) BindLegacyPublication(context.Context, recovery.PhasePermit, string, recovery.PublicationBinding) error {
	return nil
}

// s420InventoryManifest writes one canonical read-only backup inventory
// manifest to a temp file and returns its absolute path. This is a controlled
// file-media adapter (adapter_scoped=inventory_file): the manifest bytes are
// test-owned, but the real article.FileBackupInventoryReader re-verifies them.
func s420InventoryManifest(t *testing.T, backupRef string, cutoff, generatedAt time.Time) string {
	t.Helper()
	type entry struct {
		BackupRef         string `json:"backup_ref"`
		BackupCompletedAt string `json:"backup_completed_at"`
	}
	type payload struct {
		InventoryVersion   string  `json:"inventory_version"`
		InventoryScope     string  `json:"inventory_scope"`
		BackupRef          string  `json:"backup_ref"`
		BackupCutoff       string  `json:"backup_cutoff"`
		GeneratedAt        string  `json:"generated_at"`
		RecoverableBackups []entry `json:"recoverable_backups"`
	}
	const layout = "2006-01-02T15:04:05.000000Z"
	body := payload{
		InventoryVersion: article.BackupInventoryV1Version,
		InventoryScope:   article.BackupInventoryScopeV1,
		BackupRef:        backupRef,
		BackupCutoff:     s420Canonical(cutoff).Format(layout),
		GeneratedAt:      s420Canonical(generatedAt).Format(layout),
		RecoverableBackups: []entry{{
			BackupRef: backupRef, BackupCompletedAt: s420Canonical(cutoff).Format(layout),
		}},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode inventory digest payload failed: %v", err)
	}
	manifest := string(encoded[:len(encoded)-1]) + `,"inventory_digest":` + s420JSON(recovery.SHA256Digest(encoded)) + `}`
	path := filepath.Join(t.TempDir(), "s420-inventory.json")
	if err := os.WriteFile(path, []byte(manifest), 0o444); err != nil {
		t.Fatalf("write inventory manifest failed: %v", err)
	}
	return path
}

// s420ReplayChain is the REAL deletion-replay/publication chain: the real
// PostgresReplayEvidenceStore, the real production replay source (journal +
// file inventory + real finalizer replayer + real PG fact reader), the sealed
// real publication ports and the real DeletionReplayProvider gate.
type s420ReplayChain struct {
	store    *article.PostgresReplayEvidenceStore
	source   *article.ProductionDeletionReplaySource
	ports    *article.PostgresReplayEvidenceStore
	provider *article.DeletionReplayProvider
	rt       recovery.Runtime
}

func s420NewReplayChain(
	t *testing.T, f *s420Fixture, journal *s420SoftwareDeletionJournal,
	finalizer *article.DeletionFinalizer, rt recovery.Runtime, store *article.PostgresReplayEvidenceStore,
	installationID, backupRef string,
) *s420ReplayChain {
	t.Helper()
	if backupRef == "" {
		backupRef = "s420-backup-" + s420NewUUID()[:8]
	}
	clock := s420ClockAdapter{now: func() time.Time { return f.now }}
	if store == nil {
		created, err := article.NewPostgresReplayEvidenceStore(f.pool)
		if err != nil {
			t.Fatalf("real replay evidence store failed: %v", err)
		}
		store = created
	}
	facts, err := article.NewPostgresReplayFactReader(f.pool)
	if err != nil {
		t.Fatalf("real replay fact reader failed: %v", err)
	}
	inventory, err := article.NewFileBackupInventoryReader(
		s420InventoryManifest(t, backupRef, f.now, f.now),
	)
	if err != nil {
		t.Fatalf("real file backup inventory reader failed: %v", err)
	}
	t.Logf("adapter_scoped=inventory_file; not real approved backup; reason=windows_unsupported")
	source, err := article.NewProductionDeletionReplaySource(
		journal, inventory, finalizer, facts, facts, func() time.Time { return f.now },
	)
	if err != nil {
		t.Fatalf("real production deletion replay source failed: %v", err)
	}
	ports, err := article.NewReplayPublicationPorts(store, article.ReplayPublicationDependencies{
		Source: source, Runtime: rt, InstallationID: installationID,
		PublicationIDs: s420IDAdapter{}, Clock: clock,
	})
	if err != nil {
		t.Fatalf("real replay publication ports failed: %v", err)
	}
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{
		Store: &s420LoggedReplayStore{t: t, inner: ports}, Source: &s420LoggedReplaySource{t: t, inner: source},
		EvidenceIDs: s420IDAdapter{}, Tokens: s420IDAdapter{}, Clock: clock,
	})
	if err != nil {
		t.Fatalf("real deletion replay provider failed: %v", err)
	}
	return &s420ReplayChain{store: store, source: source, ports: ports, provider: provider, rt: rt}
}

// s420LoggedReplayStore and s420LoggedReplaySource surface the real provider's
// intermediate errors so the integration test can report the exact cause
// instead of only the public "article evidence unavailable" projection.
type s420LoggedReplayStore struct {
	t     *testing.T
	inner article.ReplayEvidenceStore
}

func (l *s420LoggedReplayStore) LoadAttempt(ctx context.Context, runID, backupRef string) (article.AttemptSnapshot, error) {
	attempt, err := l.inner.LoadAttempt(ctx, runID, backupRef)
	if err != nil {
		l.t.Logf("real replay store LoadAttempt err=%v", err)
	}
	return attempt, err
}

func (l *s420LoggedReplayStore) Begin(ctx context.Context, runID, backupRef string, expected article.AttemptSnapshot, fresh article.DeletionReplayFacts, candidate string) (article.BeginResult, error) {
	result, err := l.inner.Begin(ctx, runID, backupRef, expected, fresh, candidate)
	if err != nil {
		l.t.Logf("real replay store Begin err=%v expected=%+v", err, expected)
	}
	return result, err
}

func (l *s420LoggedReplayStore) Publish(ctx context.Context, request article.ReplayPublishRequest) (article.DeletionReplayEvidence, error) {
	evidence, err := l.inner.Publish(ctx, request)
	if err != nil {
		l.t.Logf("real replay store Publish err=%v attempt=%+v", err, request.Attempt)
	}
	return evidence, err
}

func (l *s420LoggedReplayStore) Fail(ctx context.Context, runID, backupRef string, attempt article.AttemptSnapshot, reason string) error {
	err := l.inner.Fail(ctx, runID, backupRef, attempt, reason)
	if err != nil {
		l.t.Logf("real replay store Fail err=%v", err)
	}
	return err
}

func (l *s420LoggedReplayStore) ReadCurrent(ctx context.Context, runID, backupRef string, refresh func(context.Context) (article.DeletionReplayFacts, error)) (article.DeletionReplayRun, *article.DeletionReplayEvidence, error) {
	run, evidence, err := l.inner.ReadCurrent(ctx, runID, backupRef, refresh)
	if err != nil {
		l.t.Logf("real replay store ReadCurrent err=%v", err)
	}
	return run, evidence, err
}

type s420LoggedReplaySource struct {
	t     *testing.T
	inner article.DeletionReplaySource
}

func (l *s420LoggedReplaySource) ReplayOldBackup(ctx context.Context, backupRef string) (article.DeletionReplayInput, error) {
	input, err := l.inner.ReplayOldBackup(ctx, backupRef)
	if err != nil {
		l.t.Logf("real replay source ReplayOldBackup err=%v", err)
	}
	return input, err
}

func (l *s420LoggedReplaySource) Snapshot(ctx context.Context, backupRef string) (article.DeletionReplayFacts, error) {
	facts, err := l.inner.Snapshot(ctx, backupRef)
	if err != nil {
		l.t.Logf("real replay source Snapshot err=%v", err)
	}
	return facts, err
}

// s420Transactor is the real PostgreSQL transaction boundary (pgxpool).
type s420Transactor struct{ pool *pgxpool.Pool }

func (t s420Transactor) Begin(ctx context.Context) (pgx.Tx, error) { return t.pool.Begin(ctx) }

// s420FullWorld is a genesis installation wired through the real production
// runtime factory plus the real PostgreSQL deletion finalizer.
type s420FullWorld struct {
	f               *s420Fixture
	store           *identity.RecoveryRunStore
	journal         *s420SoftwareJournal
	commitments     *s420SoftwareCommitments
	sources         *s420SoftwareSourceVerifier
	closures        *article.PostgresReplayEvidenceStore
	deletionJournal *s420SoftwareDeletionJournal
	installation    recovery.InstallationBinding
	manifest        recovery.InstallationManifest
	exclusive       *s420SoftwareExclusive
	lifecycle       *s420SoftwareLifecycle
	writer          *s420SoftwareInstallationWriter
	runtime         recovery.Runtime
	finalizer       *article.DeletionFinalizer
	chain           *s420ReplayChain
	coordinator     *recovery.Coordinator
	now             time.Time
	coordinateProbe func(context.Context, recovery.PhasePermit) error
	replayBackupRef string

	installationExecutor recovery.InstallationExecutor
}

func s420NewFullWorld(t *testing.T) *s420FullWorld {
	t.Helper()
	fixture := s420NewFixture(t)
	now := fixture.now
	deletionJournal := s420NewSoftwareDeletionJournal(func() time.Time { return fixture.now })
	deletionJournal.journalID = "s420-article-journal"
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s420-deployment",
		InstallationID:  "s420-installation",
		TargetDigest:    s420Digest("s420-target"),
		CommitmentRef:   "recovery/v1/installation/s420-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s420-recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: deletionJournal.journalID, Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s420-approved-source",
		SourceApprovalDigest: s420Digest("s420-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s420ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s420NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	commitments := &s420SoftwareCommitments{
		deploymentID: installation.DeploymentID,
		commitment: recovery.InstallationCommitment{
			Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
		},
		revision: "s420-commitment-1",
	}
	closureStore, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	world := &s420FullWorld{
		f: fixture, store: store, journal: journal,
		commitments: commitments, sources: &s420SoftwareSourceVerifier{expected: manifest.SourceApprovalDigest},
		closures: closureStore, deletionJournal: deletionJournal,
		installation: installation, manifest: manifest, now: now,
	}
	world.s420WireFinalizer(t)
	world.s420WireRuntime(t)
	world.s420WireReplay(t)
	return world
}

// s420WireRuntime builds one fresh production runtime generation. Each runtime
// instance may perform exactly one Install/RunNormal/Enter/Resume action, so a
// historical restart always gets a new instance over the same durable media.
func (w *s420FullWorld) s420WireRuntime(t *testing.T) {
	t.Helper()
	w.exclusive = &s420SoftwareExclusive{}
	w.lifecycle = &s420SoftwareLifecycle{}
	w.writer = &s420SoftwareInstallationWriter{}
	var runtime recovery.Runtime
	installation, err := recovery.NewInstallationExecutor(recovery.InstallationDependencies{
		Commitments: w.commitments,
		Sources:     w.sources,
		Writer:      w.writer,
		Store:       w.store,
		Anchor:      w.deletionJournal,
		Recovery:    w.journal,
		Article:     w.deletionJournal,
		Coordinate: func(ctx context.Context, permit recovery.PhasePermit, manifest recovery.InstallationManifest) error {
			anchor, token, err := runtime.CheckArticleCoordination(
				ctx, permit, manifest.InstallationID, manifest.Article.JournalID, manifest.Article.Epoch,
			)
			if err != nil {
				return err
			}
			t.Logf("install coordinate accepted anchor=%+v token=%s", anchor.Committed, token)
			if w.coordinateProbe != nil {
				return w.coordinateProbe(ctx, permit)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("real installation executor failed: %v", err)
	}
	w.installationExecutor = installation
	genesis, err := recovery.NewInstallationReader(w.commitments, w.sources)
	if err != nil {
		t.Fatalf("real installation reader failed: %v", err)
	}
	created, err := recovery.NewRuntime(w.f.ctx, recovery.RuntimeDependencies{
		Store:               w.store,
		Journal:             w.journal,
		ArticleSnapshots:    w.deletionJournal,
		Closures:            w.closures,
		Genesis:             genesis,
		Exclusive:           w.exclusive,
		Lifecycle:           w.lifecycle,
		InstallationWriter:  w.writer,
		Commitments:         w.commitments,
		InstallationSources: w.sources,
		Installation:        installation,
		ArticleAnchorReader: w.deletionJournal,
	})
	if err != nil {
		t.Fatalf("real recovery runtime factory failed: %v", err)
	}
	runtime = created
	w.runtime = created
}

// s420WireReplay wires the REAL replay/publication chain and the REAL
// recovery.Coordinator over the same store, journal and article journal
// authority. The coordinator is constructed (and validated) for real; the
// runtime-driven consumed window remains unproven because RunNormal never
// invokes Consumer and Coordinator.Consume requires a live replay phase permit.
func (w *s420FullWorld) s420WireReplay(t *testing.T) *s420ReplayChain {
	t.Helper()
	if w.chain != nil {
		return w.chain
	}
	w.chain = s420NewReplayChain(t, w.f, w.deletionJournal, w.finalizer, w.runtime, w.closures, w.installation.InstallationID, w.replayBackupRef)
	coordinator, err := recovery.NewCoordinator(
		w.store, w.journal, w.chain.provider, w.chain.ports,
		recovery.NewReplayStatusV1Decoder(nil), func() time.Time { return w.f.now },
	)
	if err != nil {
		t.Fatalf("real recovery coordinator failed: %v", err)
	}
	w.coordinator = coordinator
	return w.chain
}

func (w *s420FullWorld) s420WireFinalizer(t *testing.T) {
	t.Helper()
	lifecycle, err := identity.NewArticleIdentityLifecycle(
		s420CompletionVerifier{}, audit.Store{}, s420IDAdapter{}, s420ClockAdapter{now: func() time.Time { return w.f.now }},
	)
	if err != nil {
		t.Fatalf("real identity article lifecycle failed: %v", err)
	}
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions:  s420Transactor{pool: w.f.pool},
		Repository:    article.Repository{},
		Journal:       w.deletionJournal,
		Auditor:       audit.Store{},
		AuditIDs:      s420IDAdapter{},
		Clock:         s420ClockAdapter{now: func() time.Time { return w.f.now }},
		WorkerID:      "s420-finalizer-worker",
		LeaseDuration: time.Minute,
		Identity:      s420IdentityPort{lifecycle: lifecycle},
	})
	if err != nil {
		t.Fatalf("real deletion finalizer failed: %v", err)
	}
	w.finalizer = finalizer
}

// s420RunNormal drives the real runtime to public readiness.
func (w *s420FullWorld) s420RunNormal(t *testing.T) {
	t.Helper()
	if err := w.runtime.RunNormal(w.f.ctx); err != nil {
		t.Fatalf("real runtime RunNormal failed: %v", err)
	}
	observation, err := w.runtime.Verify(w.f.ctx)
	if err != nil {
		t.Fatalf("real runtime Verify failed: %v", err)
	}
	if observation.State != "public_ready" {
		t.Fatalf("real runtime state=%q; want public_ready", observation.State)
	}
}

// s420Install drives the real production installation algorithm (bound path).
func (w *s420FullWorld) s420Install(t *testing.T) {
	t.Helper()
	if err := w.runtime.Install(w.f.ctx, w.manifest); err != nil {
		t.Fatalf("real runtime Install failed: %v", err)
	}
}

// s420SeedPendingDeletion inserts a real user, a real journaled deletion
// operation and one real deletion_pending article row.
func (w *s420FullWorld) s420SeedPendingDeletion(t *testing.T) (userID, articleID, operationID string) {
	t.Helper()
	userID, articleID, operationID = s420NewUUID(), s420NewUUID(), s420NewUUID()
	now := w.f.now
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s420-synthetic-password-digest','member','active','permanent',false,0,0,$5,$5)`,
		userID, "s420-"+userID[:8], "s420-"+userID[:8]+"@example.test", "+1555"+s420Digits(userID), now); err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	for _, identifier := range [][2]string{
		{"username", "s420-" + userID[:8]},
		{"email", "s420-" + userID[:8] + "@example.test"},
		{"phone", "+1555" + s420Digits(userID)},
	} {
		if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO identity_identifiers(
			user_id, kind, normalized_value, lifecycle, created_at) VALUES($1,$2,$3,'active',$4)`,
			userID, identifier[0], identifier[1], now); err != nil {
			t.Fatalf("seed identity identifier %s failed: %v", identifier[0], err)
		}
	}
	tombstone, err := w.deletionJournal.Append(w.f.ctx, article.DeletionTombstoneRequest{
		ArticleID: articleID, ArticleVersion: 3, DeletionOperationID: operationID, DeletedAt: now,
	})
	if err != nil {
		t.Fatalf("real deletion journal Append failed: %v", err)
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO deletion_operations(
		id, article_id, article_version, delete_kind, status, tombstone_sequence,
		journal_key_id, journal_key_version, actor_user_id, actor_role, started_at)
		VALUES($1,$2,3,'ordinary','journaled',$3,$4,$5,$6,'member',$7)`,
		operationID, articleID, tombstone.Sequence, tombstone.KeyID, tombstone.KeyVersion, userID, now); err != nil {
		t.Fatalf("seed deletion operation failed: %v", err)
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO articles(
		id, author_id, title, body, publication_state, violation_state, deletion_state,
		deletion_operation_id, version, created_at, updated_at)
		VALUES($1,$2,'S4.17 title','S4.17 body','published','clear','deletion_pending',$3,4,$4,$4)`,
		articleID, userID, operationID, now); err != nil {
		t.Fatalf("seed pending article failed: %v", err)
	}
	return userID, articleID, operationID
}

func s420Digits(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	digits := make([]byte, 7)
	for index := range digits {
		digits[index] = '0' + sum[index]%10
	}
	return string(digits)
}

var _ recovery.JournalHeadAnchor = (*s420SoftwareDeletionJournal)(nil)
var _ recovery.AnchorReader = (*s420SoftwareDeletionJournal)(nil)
var _ recovery.InstallationRepository = (*s420SoftwareInstallationWriter)(nil)
var _ article.DispositionIdentityPort = s420IdentityPort{}

// s420BeginTransactor adapts the real pool to the article/identity Begin seam.
type s420BeginTransactor struct{ pool *pgxpool.Pool }

func (t s420BeginTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	return t.pool.Begin(ctx)
}

// s420PasswordHasher is a boundary adapter for the unused password digest in
// this leaf. adapter_scoped=password_hasher; it never decides admission.
type s420PasswordHasher struct{}

func (s420PasswordHasher) Hash(password string) (string, error) {
	return "s420-synthetic-digest:" + s420Digest(password), nil
}

// s420CanonicalWiring holds the REAL identity canonical-delete stack and the
// REAL identity recovery execution over the fixture that already owns the real
// recovery runtime, journal, store and replay chain.
type s420CanonicalWiring struct {
	coordinator *identity.CanonicalDeleteCoordinator
	port        identity.ArticleAccountPort
	exec        recovery.IdentityRecoveryExecutor
	maintenance identity.RecoveryIdentityMaintenance
	adminID     string
	memberID    string
	articleID   string
}

func (w *s420FullWorld) s420WireCanonical(t *testing.T) *s420CanonicalWiring {
	t.Helper()
	clock := s420ClockAdapter{now: func() time.Time { return w.f.now }}
	commands, err := article.NewCommandService(article.CommandDependencies{
		Transactions: s420BeginTransactor{pool: w.f.pool},
		Repository:   article.Repository{},
		IDs:          s420IDAdapter{},
		Clock:        clock,
		Orphaning: &article.OrphanArticlesCanonicalDependencies{
			Repository: article.CanonicalOrphanRepository{},
			Auditor:    audit.Store{},
			AuditIDs:   s420IDAdapter{},
			Clock:      clock,
		},
	})
	if err != nil {
		t.Fatalf("real article command service failed: %v", err)
	}
	port := identity.ArticleAccountPortFunc(func(ctx context.Context, tx pgx.Tx, userID string) (identity.CanonicalDeleteArticleResult, error) {
		result, err := commands.OrphanArticlesCanonical(ctx, tx, userID)
		if err != nil {
			return identity.CanonicalDeleteArticleResult{}, err
		}
		return identity.CanonicalDeleteArticleResult{
			NewlyOrphanedCount:   result.NewlyOrphanedCount,
			AlreadyOrphanedCount: result.AlreadyOrphanedCount,
			PendingOrphanCount:   result.PendingOrphanCount,
		}, nil
	})
	coordinator, err := identity.NewCanonicalDeleteService(identity.CanonicalDeleteDependencies{
		Transactions: s420BeginTransactor{pool: w.f.pool},
		Repository:   identity.PostgresCanonicalDeleteRepository{},
		Articles:     port,
		Audit:        audit.Store{},
		IDs:          s420IDAdapter{},
		Clock:        clock,
	})
	if err != nil {
		t.Fatalf("real canonical delete coordinator failed: %v", err)
	}
	// The REAL identity recovery execution owns the REAL recoveryExecutionGate
	// (injected with the real NewCanonicalDeleteService). No software gate.
	exec, maintenance, err := identity.NewRecoveryIdentityExecution(identity.RecoveryIdentityExecutionDependencies{
		Store:     w.store,
		Runtime:   w.runtime,
		Journal:   w.journal,
		Articles:  port,
		Audit:     audit.Store{},
		Passwords: s420PasswordHasher{},
		IDs:       s420IDAdapter{},
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("real identity recovery execution failed: %v", err)
	}
	wiring := &s420CanonicalWiring{coordinator: coordinator, port: port, exec: exec, maintenance: maintenance}
	wiring.seed(t, w)
	return wiring
}

func (c *s420CanonicalWiring) seed(t *testing.T, w *s420FullWorld) {
	t.Helper()
	c.adminID, c.memberID, c.articleID = s420NewUUID(), s420NewUUID(), s420NewUUID()
	now := w.f.now
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, role, is_initial_admin, status, credential_mode, recovery_required, created_at, updated_at)
		VALUES($1,$2,$3,'admin',true,'active','permanent',false,$4,$4)`,
		c.adminID, "s420-admin-"+c.adminID[:8], "s420-admin-"+c.adminID[:8]+"@example.test", now); err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}
	username, email, phone := "s420-member-"+c.memberID[:8], "s420-member-"+c.memberID[:8]+"@example.test", "+1555"+s420Digits(c.memberID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s420-synthetic-password-digest','member','active','permanent',false,0,0,$5,$5)`,
		c.memberID, username, email, phone, now); err != nil {
		t.Fatalf("seed member failed: %v", err)
	}
	for _, identifier := range [][2]string{{"username", username}, {"email", email}, {"phone", phone}} {
		if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO identity_identifiers(
			user_id, kind, normalized_value, lifecycle, created_at) VALUES($1,$2,$3,'active',$4)`,
			c.memberID, identifier[0], identifier[1], now); err != nil {
			t.Fatalf("seed member identifier %s failed: %v", identifier[0], err)
		}
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO articles(
		id, author_id, title, body, publication_state, violation_state, deletion_state, version, created_at, updated_at)
		VALUES($1,$2,'S4.17 canonical title','S4.17 canonical body','published','clear','active',1,$3,$3)`,
		c.articleID, c.memberID, now); err != nil {
		t.Fatalf("seed member article failed: %v", err)
	}
}

func (c *s420CanonicalWiring) adminPrincipal() session.Principal {
	return session.Principal{UserID: c.adminID, Role: "admin", Status: "active", IsInitialAdmin: true}
}

// s420PublicationSeed identifies one REAL PostgreSQL evidence/publication pair
// written by the test with the same canonical encodings the production store
// verifies. Nothing here decides freshness: PostgresReplayEvidenceStore does.
type s420PublicationSeed struct {
	runID, backupRef, evidenceRef, evidenceDigest string
	publicationRef, publicationDigest             string
}

// s420PublicationCanonical mirrors the production canonical publication record
// field order and names so the test can compute the same publication digest.
type s420PublicationCanonical struct {
	Version             string `json:"publication_version"`
	Ref                 string `json:"publication_ref"`
	SourceKind          string `json:"source_kind"`
	RunID               string `json:"run_id"`
	BackupRef           string `json:"backup_ref"`
	Namespace           string `json:"namespace"`
	EvidenceRef         string `json:"evidence_ref"`
	EvidenceDigest      string `json:"evidence_digest"`
	EvidenceCompletedAt string `json:"evidence_completed_at"`
	JournalID           string `json:"journal_id"`
	Epoch               uint64 `json:"journal_epoch"`
	Sequence            uint64 `json:"journal_head_sequence"`
	Hash                string `json:"journal_head_hash"`
	Generation          int64  `json:"attempt_generation"`
	SourceRef           string `json:"source_ref"`
	SourceDigest        string `json:"source_digest"`
	PublishedAt         string `json:"published_at"`
}

// s420SeedPublishedClosure writes a real deletion_replay_runs + evidence +
// article_replay_publications row set bound to the given journal identity and
// head. The real PostgresReplayEvidenceStore later decides fresh/stale/rollback.
func s420SeedPublishedClosure(t *testing.T, w *s420FullWorld, journalID string, epoch uint64, head recovery.JournalHead) s420PublicationSeed {
	t.Helper()
	if head.Sequence == 0 || head.Hash == "" {
		t.Fatalf("seeded publication head must be a committed non-genesis head: %#v", head)
	}
	seed := s420PublicationSeed{
		runID:          "s420-run-" + s420NewUUID()[:8],
		backupRef:      "s420-backup-" + s420NewUUID()[:8],
		evidenceRef:    "s420-evidence-" + s420NewUUID()[:8],
		publicationRef: s420NewUUID(),
	}
	cutoff := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	completedAt := cutoff.Add(time.Second)
	publishedAt := completedAt.Add(time.Second)
	status := recovery.ReadyReplayStatusV1{
		EvidenceVersion:          recovery.ReplayEvidenceVersionV1,
		RunID:                    seed.runID,
		BackupRef:                seed.backupRef,
		Namespace:                article.ArticleDeletionNamespace,
		JournalHeadSequence:      head.Sequence,
		JournalHeadHash:          head.Hash,
		InventoryScope:           article.BackupInventoryScopeV1,
		BackupCutoff:             cutoff.Format(time.RFC3339Nano),
		InventoryDigest:          s420Digest("s420-inventory"),
		DeletionOperationsDigest: s420Digest("s420-operations"),
		AuditEventsDigest:        s420Digest("s420-audit"),
	}
	digest, err := status.CanonicalDigest()
	if err != nil {
		t.Fatalf("canonical evidence digest failed: %v", err)
	}
	seed.evidenceDigest = digest
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO deletion_replay_runs(
		run_id, namespace, backup_ref, status, started_at, updated_at)
		VALUES($1,$2,$3,'running',$4,$4)`,
		seed.runID, article.ArticleDeletionNamespace, seed.backupRef, completedAt); err != nil {
		t.Fatalf("seed replay run failed: %v", err)
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO deletion_replay_evidence(
		evidence_version, run_id, backup_ref, namespace, journal_head_sequence,
		journal_head_hash, inventory_scope, backup_cutoff, inventory_digest,
		deletion_operations_digest, audit_events_digest, tombstones_total,
		tombstones_replayed, already_absent, deletion_operations_reconciled_count,
		audit_events_verified_count, evidence_digest, evidence_ref, completed_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,0,0,0,0,0,$12,$13,$14)`,
		status.EvidenceVersion, seed.runID, seed.backupRef, status.Namespace, int64(head.Sequence),
		head.Hash, status.InventoryScope, cutoff, status.InventoryDigest,
		status.DeletionOperationsDigest, status.AuditEventsDigest, seed.evidenceDigest, seed.evidenceRef, completedAt); err != nil {
		t.Fatalf("seed replay evidence failed: %v", err)
	}
	const layout = "2006-01-02T15:04:05.000000Z"
	canonical := s420PublicationCanonical{
		Version: "article-replay-publication/v1", Ref: seed.publicationRef, SourceKind: "atomic_current_v1",
		RunID: seed.runID, BackupRef: seed.backupRef, Namespace: article.ArticleDeletionNamespace,
		EvidenceRef: seed.evidenceRef, EvidenceDigest: seed.evidenceDigest,
		EvidenceCompletedAt: completedAt.Format(layout), JournalID: journalID, Epoch: epoch,
		Sequence: head.Sequence, Hash: head.Hash, Generation: 0,
		SourceRef: seed.publicationRef, SourceDigest: s420Digest("s420-source-placeholder"),
		PublishedAt: publishedAt.Format(layout),
	}
	canonical.SourceDigest = recovery.SHA256Digest([]byte(s420JSON([]any{
		seed.runID, seed.backupRef, article.ArticleDeletionNamespace, int64(0), seed.evidenceRef, seed.evidenceDigest,
	})))
	seed.publicationDigest = recovery.SHA256Digest([]byte(s420JSON(canonical)))
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO article_replay_publications(
		publication_ref, publication_version, source_kind, run_id, namespace, backup_ref,
		evidence_ref, evidence_digest, attempt_generation, attempt_token, journal_id, journal_epoch,
		journal_head_sequence, journal_head_hash, evidence_completed_at, published_at,
		publication_digest, source_ref, source_digest)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,0,NULL,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		seed.publicationRef, canonical.Version, canonical.SourceKind, seed.runID, article.ArticleDeletionNamespace,
		seed.backupRef, seed.evidenceRef, seed.evidenceDigest, journalID, int64(epoch), int64(head.Sequence), head.Hash,
		completedAt, publishedAt, seed.publicationDigest, seed.publicationRef, canonical.SourceDigest); err != nil {
		t.Fatalf("seed replay publication failed: %v", err)
	}
	return seed
}

func (s s420PublicationSeed) request() recovery.ClosureRequest {
	return recovery.ClosureRequest{
		RunID: s.runID, BackupRef: s.backupRef, EvidenceRef: s.evidenceRef, EvidenceDigest: s.evidenceDigest,
		PublicationRef: s.publicationRef, PublicationDigest: s.publicationDigest,
	}
}

// ---------------------------------------------------------------------------
// S4.17 leaf-5 support: the real HTTP session/authorizer/recovery-guard chain
// over the real Article query and permanent-deletion handlers.
// ---------------------------------------------------------------------------

// s420Case18Counters are observer-only counters. They never decide business
// state; they only let the test prove that a closed stream never reached the
// Article handler, service, repository or identity author reader.
type s420Case18Counters struct {
	handlerEntries  atomic.Int64
	traceContexts   atomic.Int64
	serviceCalls    atomic.Int64
	repositoryCalls atomic.Int64
	authorReadCalls atomic.Int64
}

// s420CountingQueryRepository is an observer adapter over the real PostgreSQL
// query repository. It does not alter reads or visibility decisions.
type s420CountingQueryRepository struct {
	next     article.ArticleQueryRepository
	counters *s420Case18Counters
}

func (r s420CountingQueryRepository) ListCandidateBatch(
	ctx context.Context, cursor *article.ArticleCursor, limit int,
) ([]article.Article, error) {
	r.counters.repositoryCalls.Add(1)
	return r.next.ListCandidateBatch(ctx, cursor, limit)
}

func (r s420CountingQueryRepository) GetArticle(ctx context.Context, id string) (article.Article, error) {
	r.counters.repositoryCalls.Add(1)
	return r.next.GetArticle(ctx, id)
}

// s420CountingAuthorReader is an observer adapter over the real identity-owned
// author state reader. It does not alter author states.
type s420CountingAuthorReader struct {
	next     article.ArticleAuthorStateReader[identity.UserStatus]
	counters *s420Case18Counters
}

func (r s420CountingAuthorReader) GetAuthorStates(
	ctx context.Context, ids []string,
) (map[string]identity.UserStatus, error) {
	r.counters.authorReadCalls.Add(1)
	return r.next.GetAuthorStates(ctx, ids)
}

// s420CountingQuerier is an observer adapter over the real Article query
// service. It only counts invocations.
type s420CountingQuerier struct {
	next     article.ArticleQuerier
	counters *s420Case18Counters
}

func (q s420CountingQuerier) ListArticles(
	ctx context.Context, request article.ListArticlesRequest,
) (article.ListArticlesPage, error) {
	q.counters.serviceCalls.Add(1)
	return q.next.ListArticles(ctx, request)
}

func (q s420CountingQuerier) GetArticle(
	ctx context.Context, id string,
) (article.MemberArticleDetailView, string, error) {
	q.counters.serviceCalls.Add(1)
	return q.next.GetArticle(ctx, id)
}

// s420OriginObserver is a bounded observer for same-origin rejections. It is a
// controlled external-boundary adapter and does not decide request admission.
type s420OriginObserver struct {
	rejections atomic.Int64
	lastReason atomic.Value
}

func (o *s420OriginObserver) ObserveRejected(
	_ context.Context, observation platform.OriginRejectionObservation,
) error {
	o.rejections.Add(1)
	o.lastReason.Store(string(observation.Reason))
	return nil
}

// s420Case18World wires one isolated PostgreSQL schema through the real
// production runtime and the real HTTP session, authorizer, recovery guard,
// Article query handler and permanent-deletion handler. Only the exact-route
// transport skeleton and the observer adapters above are test-owned glue.
type s420Case18World struct {
	f              *s420Fixture
	full           *s420FullWorld
	counters       *s420Case18Counters
	originObserver *s420OriginObserver
	router         http.Handler
	detailHandler  http.Handler
	listHandler    http.Handler
	deleteHandler  http.Handler
	userID         string
	articleID      string
	sessionToken   string
	csrfToken      string
	etag           string
	publicOrigin   string
}

func s420NewCase18World(t *testing.T) *s420Case18World {
	t.Helper()
	full := s420NewFullWorld(t)
	fixture := full.f
	world := &s420Case18World{
		f: fixture, full: full,
		counters:       &s420Case18Counters{},
		originObserver: &s420OriginObserver{},
		publicOrigin:   "https://community.example.invalid",
	}
	world.seed(t)

	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      fixture.pool,
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		t.Fatalf("real session store failed: %v", err)
	}
	world.sessionToken = "s420-session-" + s420NewUUID()
	world.csrfToken = "s420-csrf-" + s420NewUUID()
	if _, err := store.Create(fixture.ctx, fixture.pool, session.CreateInput{
		ID:                          s420NewUUID(),
		UserID:                      world.userID,
		Token:                       world.sessionToken,
		CSRFToken:                   world.csrfToken,
		AuthenticatedCredentialKind: session.CredentialPermanent,
		AuthGeneration:              0,
		IssuedAt:                    fixture.now,
		AbsoluteExpiresAt:           fixture.now.Add(24 * time.Hour),
		IdleExpiresAt:               fixture.now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("real session Create failed: %v", err)
	}
	authorizer, err := platform.NewHTTPAuthorizer(store, func() time.Time { return fixture.now })
	if err != nil {
		t.Fatalf("real HTTP authorizer failed: %v", err)
	}
	guard, err := platform.NewRecoveryGuard(full.runtime)
	if err != nil {
		t.Fatalf("real recovery guard failed: %v", err)
	}
	origin, err := platform.NewOriginGuard(world.publicOrigin, world.originObserver, time.Second)
	if err != nil {
		t.Fatalf("real origin guard failed: %v", err)
	}

	repository, err := article.NewPostgresArticleQueryRepository(fixture.pool)
	if err != nil {
		t.Fatalf("real article query repository failed: %v", err)
	}
	authorReader, err := identity.NewArticleIdentityReader(fixture.pool)
	if err != nil {
		t.Fatalf("real identity author reader failed: %v", err)
	}
	queryService, err := article.NewArticleQueryService[identity.UserStatus](
		s420CountingQueryRepository{next: repository, counters: world.counters},
		s420CountingAuthorReader{next: authorReader, counters: world.counters},
		article.QueryConfig{DefaultLimit: 20, MaxLimit: 50, CandidateBatchSize: 51},
	)
	if err != nil {
		t.Fatalf("real article query service failed: %v", err)
	}
	queryHandler, err := article.NewQueryHandler(s420CountingQuerier{next: queryService, counters: world.counters})
	if err != nil {
		t.Fatalf("real article query handler failed: %v", err)
	}

	coordinator, err := article.NewPermanentDeletionCoordinator(article.PermanentDeletionDependencies{
		Transactions: s420Transactor{pool: fixture.pool},
		Repository:   article.Repository{},
		Journal:      full.deletionJournal,
		Finalizer:    full.finalizer,
		OperationIDs: s420IDAdapter{},
		Clock:        s420ClockAdapter{now: func() time.Time { return fixture.now }},
	})
	if err != nil {
		t.Fatalf("real permanent deletion coordinator failed: %v", err)
	}
	commands, err := article.NewCommandService(article.CommandDependencies{
		Transactions: s420Transactor{pool: fixture.pool},
		Repository:   article.Repository{},
		IDs:          s420IDAdapter{},
		Clock:        s420ClockAdapter{now: func() time.Time { return fixture.now }},
		Deletion:     coordinator,
	})
	if err != nil {
		t.Fatalf("real article command service failed: %v", err)
	}
	writeLogger, err := article.NewSlogArticleResponseWriteLogger(
		slog.New(slog.NewJSONHandler(io.Discard, nil)), "s420",
	)
	if err != nil {
		t.Fatalf("real article response write logger failed: %v", err)
	}
	deleteHandler, err := article.NewHandler(commands, writeLogger, commands)
	if err != nil {
		t.Fatalf("real article deletion handler failed: %v", err)
	}

	observed := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			world.counters.handlerEntries.Add(1)
			if _, ok := platform.TraceIDFromContext(request.Context()); ok {
				world.counters.traceContexts.Add(1)
			}
			next.ServeHTTP(response, request)
		})
	}
	protected := func(next http.Handler) http.Handler {
		return origin.Protect(authorizer.Protect(session.AccessGeneral, guard.Protect(next)))
	}
	world.detailHandler = protected(observed(http.HandlerFunc(queryHandler.GetArticle)))
	world.listHandler = protected(observed(http.HandlerFunc(queryHandler.ListArticles)))
	world.deleteHandler = protected(observed(http.HandlerFunc(deleteHandler.DeleteOrdinary)))
	world.router = platform.TraceMiddleware(s420NewCase18Transport(world))
	return world
}

// s420Case18Transport is a transport-only adapter over the standard library
// ServeMux method patterns. The router contract (404 for unregistered paths,
// 405 + Allow for a registered path with an unsupported method) is produced by
// the real ServeMux; only the mux-generated text/plain 405 body is rewritten
// into the platform error envelope. It never decides business state.
type s420Case18Transport struct{ next http.Handler }

// s420Case18BufferedResponse records one handler response so the transport can
// distinguish the mux default response from a handler-written response.
type s420Case18BufferedResponse struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func (b *s420Case18BufferedResponse) Header() http.Header {
	if b.header == nil {
		b.header = http.Header{}
	}
	return b.header
}

func (b *s420Case18BufferedResponse) WriteHeader(code int) {
	if b.code == 0 {
		b.code = code
	}
}

func (b *s420Case18BufferedResponse) Write(payload []byte) (int, error) {
	if b.code == 0 {
		b.code = http.StatusOK
	}
	return b.body.Write(payload)
}

func s420NewCase18Transport(world *s420Case18World) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/articles/{id}", world.detailHandler)
	mux.Handle("DELETE /api/v1/articles/{id}", world.deleteHandler)
	mux.Handle("PUT /api/v1/articles/{id}", s420OutOfScopeRoute("article update"))
	mux.Handle("GET /api/v1/articles", world.listHandler)
	mux.Handle("POST /api/v1/articles", s420OutOfScopeRoute("article create"))
	return s420Case18Transport{next: mux}
}

// s420OutOfScopeRoute registers a route that exists in the production router so
// the standard mux computes the exact Allow set, but whose body contract belongs
// to another card. It is never asserted as production behaviour.
func s420OutOfScopeRoute(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		http.Error(response, name+" route is out of scope for S4.17", http.StatusNotImplemented)
	})
}

func (t s420Case18Transport) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	buffered := &s420Case18BufferedResponse{}
	t.next.ServeHTTP(buffered, request)
	if buffered.code == 0 {
		buffered.code = http.StatusOK
	}
	if buffered.code == http.StatusMethodNotAllowed && strings.HasPrefix(buffered.header.Get("Content-Type"), "text/plain") {
		if allow := buffered.header.Get("Allow"); allow != "" {
			response.Header().Set("Allow", allow)
		}
		response.Header().Set("X-Content-Type-Options", "nosniff")
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeRequestMethodNotAllowed, errors.New("method not allowed"),
		))
		return
	}
	if buffered.code == http.StatusNotFound && strings.HasPrefix(buffered.body.String(), "404 page not found") {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeRequestRouteNotFound, errors.New("route not found"),
		))
		return
	}
	for key, values := range buffered.header {
		for _, value := range values {
			response.Header().Add(key, value)
		}
	}
	response.WriteHeader(buffered.code)
	_, _ = response.Write(buffered.body.Bytes())
}

func (w *s420Case18World) seed(t *testing.T) {
	t.Helper()
	w.userID, w.articleID = s420NewUUID(), s420NewUUID()
	now := w.f.now
	username := "s420-case18-" + w.userID[:8]
	email := username + "@example.test"
	phone := "+1555" + s420Digits(w.userID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s420-synthetic-password-digest','member','active','permanent',false,0,0,$5,$5)`,
		w.userID, username, email, phone, now); err != nil {
		t.Fatalf("seed case18 user failed: %v", err)
	}
	for _, identifier := range [][2]string{{"username", username}, {"email", email}, {"phone", phone}} {
		if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO identity_identifiers(
			user_id, kind, normalized_value, lifecycle, created_at) VALUES($1,$2,$3,'active',$4)`,
			w.userID, identifier[0], identifier[1], now); err != nil {
			t.Fatalf("seed case18 identifier %s failed: %v", identifier[0], err)
		}
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO articles(
		id, author_id, title, body, publication_state, violation_state, deletion_state, version, created_at, updated_at)
		VALUES($1,$2,'S4.17 Case18 title','S4.17 Case18 body','published','clear','active',1,$3,$3)`,
		w.articleID, w.userID, now); err != nil {
		t.Fatalf("seed case18 article failed: %v", err)
	}
}

func (w *s420Case18World) articlePath() string {
	return "/api/v1/articles/" + w.articleID
}

func (w *s420Case18World) counts() (handler, trace, service, repository, authorRead int64) {
	return w.counters.handlerEntries.Load(),
		w.counters.traceContexts.Load(),
		w.counters.serviceCalls.Load(),
		w.counters.repositoryCalls.Load(),
		w.counters.authorReadCalls.Load()
}

// s420SoftwareRestore is a controlled software tool adapter for the restore
// driver boundary. This is a controlled software tool adapter; it does not
// prove real artifact/restore/E2E. It only records the approved plan and
// returns the byte-identical receipt the real runtime validates.
type s420SoftwareRestore struct {
	installation recovery.InstallationBinding
	now          time.Time
	mu           sync.Mutex
	plan         recovery.RestorePlan
	receipt      recovery.RestoreReceipt
	prepared     int
	replaced     int
	verified     int
}

func (r *s420SoftwareRestore) Prepare(ctx context.Context, b recovery.RunBinding) (recovery.RestorePlan, error) {
	if err := ctx.Err(); err != nil {
		return recovery.RestorePlan{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plan = recovery.RestorePlan{
		Version: "han-db-restore/v1", RunID: b.RunID, BackupRef: b.BackupRef,
		DeploymentID: r.installation.DeploymentID, InstallationID: r.installation.InstallationID,
		TargetDigest:         r.installation.TargetDigest,
		ArtifactDigest:       s420Digest("s420-controlled-artifact"),
		SourceApprovalRef:    "s420-controlled-restore-approval",
		SourceApprovalDigest: s420Digest("s420-controlled-restore-approval"),
		SourceInstallationID: "s420-controlled-source-installation",
		SourceSchemaDigest:   s420Digest("s420-controlled-source-schema"),
		ArtifactPath:         "s420/controlled/restore.dump", ArtifactBytes: 4096,
	}
	r.prepared++
	return r.plan, nil
}

func (r *s420SoftwareRestore) Replace(ctx context.Context, permit recovery.PhasePermit, plan recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	if err := ctx.Err(); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if plan != r.plan {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	r.receipt = recovery.RestoreReceipt{
		Version: plan.Version, RunID: plan.RunID, BackupRef: plan.BackupRef,
		DeploymentID: plan.DeploymentID, InstallationID: plan.InstallationID, TargetDigest: plan.TargetDigest,
		ArtifactDigest: plan.ArtifactDigest, SourceApprovalDigest: plan.SourceApprovalDigest,
		RestoredSchemaDigest: plan.SourceSchemaDigest,
		ReceiptDigest:        s420Digest(plan.RunID + "|s420-restore-receipt"),
		ArtifactBytes:        plan.ArtifactBytes, RestoredAt: s420Canonical(r.now),
	}
	r.replaced++
	return r.receipt, nil
}

func (r *s420SoftwareRestore) VerifyRestored(ctx context.Context, plan recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	if err := ctx.Err(); err != nil {
		return recovery.RestoreReceipt{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if plan != r.plan || r.replaced == 0 {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
	}
	r.verified++
	return r.receipt, nil
}

func (r *s420SoftwareRestore) counts() (prepared, replaced, verified int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.prepared, r.replaced, r.verified
}

// s420RecoverySecret serves the candidate secret from memory only.
type s420RecoverySecret struct{ secret []byte }

func (s *s420RecoverySecret) ReadInitialRecoverySecret(ctx context.Context, b recovery.RunBinding) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.secret...), nil
}

// s420LateConsumer binds the real recovery.Coordinator after the runtime (and
// therefore the publication ports) exists; it never decides anything itself.
type s420LateConsumer struct {
	mu     sync.RWMutex
	target recovery.RecoveryConsumer
}

func (c *s420LateConsumer) Consume(ctx context.Context, runID string) (recovery.ConsumptionResult, error) {
	c.mu.RLock()
	target := c.target
	c.mu.RUnlock()
	if target == nil {
		return recovery.ConsumptionResult{}, recovery.ErrAdmissionBinding
	}
	return target.Consume(ctx, runID)
}

func (c *s420LateConsumer) set(target recovery.RecoveryConsumer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = target
}

// s420LateExit binds the real identity executor (RecoveryExit) after the
// runtime exists; it forwards only.
type s420LateExit struct {
	mu     sync.RWMutex
	target recovery.RecoveryExit
}

func (e *s420LateExit) ExitRecovery(ctx context.Context, permit recovery.PhasePermit, b recovery.RunBinding) error {
	e.mu.RLock()
	target := e.target
	e.mu.RUnlock()
	if target == nil {
		return recovery.ErrAdmissionBinding
	}
	return target.ExitRecovery(ctx, permit, b)
}

func (e *s420LateExit) set(target recovery.RecoveryExit) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.target = target
}

// s420RecoveryExit exposes the real executor's RevokeRecoverySessions as the
// runtime RecoveryExit port. It forwards only; it makes no decision.
type s420RecoveryExit struct {
	exec recovery.IdentityRecoveryExecutor
}

func (e s420RecoveryExit) ExitRecovery(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return e.exec.RevokeRecoverySessions(ctx, p, b)
}

// s420IdentityHook forwards the real identity executor and optionally injects
// recovery-maintenance work while the real PhaseRecoveryIdentity permit lives.
type s420IdentityHook struct {
	t     *testing.T
	inner recovery.IdentityRecoveryExecutor
	hook  func(context.Context, recovery.PhasePermit, recovery.RunBinding) error
}

func (h *s420IdentityHook) stage(name string, call func() error) error {
	err := call()
	if h.t != nil {
		h.t.Logf("identity stage %s err=%v", name, err)
	}
	return err
}

func (h *s420IdentityHook) BindRestoredRun(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, r recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	var out recovery.RecoveryRun
	err := h.stage("BindRestoredRun", func() error {
		var inner error
		out, inner = h.inner.BindRestoredRun(ctx, p, b, r)
		return inner
	})
	return out, err
}

func (h *s420IdentityHook) FinalizeRestoredIdentity(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return h.stage("FinalizeRestoredIdentity", func() error { return h.inner.FinalizeRestoredIdentity(ctx, p, b) })
}

func (h *s420IdentityHook) PrepareRecoveryTrust(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, secret []byte) error {
	return h.stage("PrepareRecoveryTrust", func() error { return h.inner.PrepareRecoveryTrust(ctx, p, b, secret) })
}

func (h *s420IdentityHook) RevokeRecoverySessions(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return h.stage("RevokeRecoverySessions", func() error { return h.inner.RevokeRecoverySessions(ctx, p, b) })
}

func (h *s420IdentityHook) CompleteIdentityReview(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return h.stage("CompleteIdentityReview", func() error {
		if h.hook != nil {
			if err := h.hook(ctx, p, b); err != nil {
				return err
			}
		}
		return h.inner.CompleteIdentityReview(ctx, p, b)
	})
}

func s420Argon2Passwords(t *testing.T) *security.Argon2idHasher {
	t.Helper()
	hasher, err := security.NewArgon2idHasher(security.Argon2idConfig{
		MemoryKiB: 8, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	}, rand.Reader)
	if err != nil {
		t.Fatalf("real argon2id hasher failed: %v", err)
	}
	return hasher
}

// s420RecoveryWorld wires the REAL recovery runtime Enter path: real
// identity.NewRecoveryRunStore, real identity.NewRecoveryIdentityExecution,
// real recovery.NewCoordinator, real article replay/publication ports and the
// real deletion finalizer over PostgreSQL. Only the journal bytes, restore
// tool, lifecycle join and exclusive lease are controlled software media
// adapters, and they never decide admission.
type s420RecoveryWorld struct {
	f                *s420Fixture
	store            *identity.RecoveryRunStore
	journal          *s420SoftwareJournal
	deletionJournal  *s420SoftwareDeletionJournal
	commitments      *s420SoftwareCommitments
	sources          *s420SoftwareSourceVerifier
	writer           *s420SoftwareInstallationWriter
	closures         *article.PostgresReplayEvidenceStore
	installation     recovery.InstallationBinding
	manifest         recovery.InstallationManifest
	exclusive        *s420SoftwareExclusive
	lifecycle        *s420SoftwareLifecycle
	restore          *s420SoftwareRestore
	runtime          recovery.Runtime
	port             identity.ArticleAccountPort
	exec             recovery.IdentityRecoveryExecutor
	maintenance      identity.RecoveryIdentityMaintenance
	coordinator      *recovery.Coordinator
	chain            *s420ReplayChain
	finalizer        *article.DeletionFinalizer
	runStoreTap      *s420RunStoreTap
	passwords        *security.Argon2idHasher
	gate             *identity.PasswordResetStore
	sessionStore     session.Store
	maintenanceStore session.MaintenanceStore
	passwordService  *identity.Service
	secret           []byte
	backupRef        string
	userID           string
	memberID         string
	now              time.Time

	installationExecutor recovery.InstallationExecutor

	// deferReplayWiring keeps the single article publication seam unsealed so a
	// leaf can seal it once against the production root runtime instead of this
	// world's own runtime. lateConsumer is the world runtime's consumer seam.
	deferReplayWiring bool
	lateConsumer      *s420LateConsumer
}

func s420NewRecoveryWorld(t *testing.T, withMember bool, hook func(context.Context, recovery.PhasePermit, recovery.RunBinding) error, worldHook func(*s420RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error) *s420RecoveryWorld {
	t.Helper()
	fixture := s420NewFixture(t)
	now := fixture.now
	deletionJournal := s420NewSoftwareDeletionJournal(func() time.Time { return fixture.now })
	deletionJournal.journalID = "s420-recovery-article-journal"
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s420-recovery-deployment",
		InstallationID:  "s420-recovery-installation",
		TargetDigest:    s420Digest("s420-recovery-target"),
		CommitmentRef:   "recovery/v1/installation/s420-recovery-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s420-recovery-run-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: deletionJournal.journalID, Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s420-recovery-approved-source",
		SourceApprovalDigest: s420Digest("s420-recovery-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s420ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s420NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	closures, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	world := &s420RecoveryWorld{
		f: fixture, store: store, journal: journal, deletionJournal: deletionJournal,
		commitments: &s420SoftwareCommitments{
			deploymentID: installation.DeploymentID,
			commitment: recovery.InstallationCommitment{
				Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
			},
			revision: "s420-recovery-commitment-1",
		},
		sources:      &s420SoftwareSourceVerifier{expected: manifest.SourceApprovalDigest},
		closures:     closures,
		installation: installation, manifest: manifest, now: now,
		secret:    []byte("s420-recovery-secret"),
		backupRef: "s420-recovery-backup-" + s420NewUUID()[:8],
	}
	world.s420WireFinalizer(t)
	if worldHook != nil {
		hook = worldHook(world)
	}
	world.s420WireRuntime(t, hook)
	world.s420Seed(t, withMember)
	return world
}

func (w *s420RecoveryWorld) s420WireFinalizer(t *testing.T) {
	t.Helper()
	lifecycle, err := identity.NewArticleIdentityLifecycle(
		s420CompletionVerifier{}, audit.Store{}, s420IDAdapter{}, s420ClockAdapter{now: func() time.Time { return w.f.now }},
	)
	if err != nil {
		t.Fatalf("real identity article lifecycle failed: %v", err)
	}
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions:  s420Transactor{pool: w.f.pool},
		Repository:    article.Repository{},
		Journal:       w.deletionJournal,
		Auditor:       audit.Store{},
		AuditIDs:      s420IDAdapter{},
		Clock:         s420ClockAdapter{now: func() time.Time { return w.f.now }},
		WorkerID:      "s420-recovery-finalizer-worker",
		LeaseDuration: time.Minute,
		Identity:      s420IdentityPort{lifecycle: lifecycle},
	})
	if err != nil {
		t.Fatalf("real deletion finalizer failed: %v", err)
	}
	w.finalizer = finalizer
}

func (w *s420RecoveryWorld) s420WireRuntime(t *testing.T, hook func(context.Context, recovery.PhasePermit, recovery.RunBinding) error) {
	t.Helper()
	clock := func() time.Time { return w.f.now }
	w.exclusive = &s420SoftwareExclusive{}
	w.lifecycle = &s420SoftwareLifecycle{}
	w.writer = &s420SoftwareInstallationWriter{}
	w.restore = &s420SoftwareRestore{installation: w.installation, now: w.f.now}
	forwarder := identity.NewRecoveryIdentityForwarder()
	lateConsumer := &s420LateConsumer{}
	lateExit := &s420LateExit{}
	var runtime recovery.Runtime
	installation, err := recovery.NewInstallationExecutor(recovery.InstallationDependencies{
		Commitments: w.commitments,
		Sources:     w.sources,
		Writer:      w.writer,
		Store:       w.store,
		Anchor:      w.deletionJournal,
		Recovery:    w.journal,
		Article:     w.deletionJournal,
		Coordinate: func(ctx context.Context, permit recovery.PhasePermit, manifest recovery.InstallationManifest) error {
			_, _, err := runtime.CheckArticleCoordination(
				ctx, permit, manifest.InstallationID, manifest.Article.JournalID, manifest.Article.Epoch,
			)
			return err
		},
	})
	if err != nil {
		t.Fatalf("real installation executor failed: %v", err)
	}
	w.installationExecutor = installation
	genesis, err := recovery.NewInstallationReader(w.commitments, w.sources)
	if err != nil {
		t.Fatalf("real installation reader failed: %v", err)
	}
	created, err := recovery.NewRuntime(w.f.ctx, recovery.RuntimeDependencies{
		Store:               w.store,
		Journal:             w.journal,
		ArticleSnapshots:    w.deletionJournal,
		Closures:            w.closures,
		Genesis:             genesis,
		Exclusive:           w.exclusive,
		Lifecycle:           w.lifecycle,
		InstallationWriter:  w.writer,
		Commitments:         w.commitments,
		InstallationSources: w.sources,
		Installation:        installation,
		ArticleAnchorReader: w.deletionJournal,
		Runs:                w.store,
		Identity:            &s420IdentityHook{t: t, inner: forwarder, hook: hook},
		Restore:             w.restore,
		Consumer:            lateConsumer,
		Exit:                lateExit,
		RecoverySecret:      &s420RecoverySecret{secret: w.secret},
	})
	if err != nil {
		t.Fatalf("real recovery runtime factory failed: %v", err)
	}
	runtime = created
	w.runtime = created
	commands, err := article.NewCommandService(article.CommandDependencies{
		Transactions: s420BeginTransactor{pool: w.f.pool},
		Repository:   article.Repository{},
		IDs:          s420IDAdapter{},
		Clock:        s420ClockAdapter{now: clock},
		Orphaning: &article.OrphanArticlesCanonicalDependencies{
			Repository: article.CanonicalOrphanRepository{},
			Auditor:    audit.Store{},
			AuditIDs:   s420IDAdapter{},
			Clock:      s420ClockAdapter{now: clock},
		},
	})
	if err != nil {
		t.Fatalf("real article command service failed: %v", err)
	}
	w.port = identity.ArticleAccountPortFunc(func(ctx context.Context, tx pgx.Tx, userID string) (identity.CanonicalDeleteArticleResult, error) {
		result, err := commands.OrphanArticlesCanonical(ctx, tx, userID)
		if err != nil {
			return identity.CanonicalDeleteArticleResult{}, err
		}
		return identity.CanonicalDeleteArticleResult{
			NewlyOrphanedCount:   result.NewlyOrphanedCount,
			AlreadyOrphanedCount: result.AlreadyOrphanedCount,
			PendingOrphanCount:   result.PendingOrphanCount,
		}, nil
	})
	w.passwords = s420Argon2Passwords(t)
	exec, maintenance, err := identity.NewRecoveryIdentityExecution(identity.RecoveryIdentityExecutionDependencies{
		Store: w.store, Runtime: runtime, Journal: w.journal, Articles: w.port,
		Audit: audit.Store{}, Passwords: w.passwords, IDs: s420IDAdapter{},
		Clock: s420ClockAdapter{now: clock},
	})
	if err != nil {
		t.Fatalf("real identity recovery execution failed: %v", err)
	}
	if err := forwarder.Seal(exec); err != nil {
		t.Fatalf("real identity forwarder seal failed: %v", err)
	}
	w.exec = exec
	w.maintenance = maintenance
	authority, ok := exec.(session.MaintenanceSessionAuthority)
	if !ok {
		t.Fatalf("real identity recovery execution does not expose the maintenance authority")
	}
	lateExit.set(s420RecoveryExit{exec: exec})
	w.lateConsumer = lateConsumer
	if !w.deferReplayWiring {
		w.s420WireReplay(t, runtime, lateConsumer)
	}

	w.gate = &identity.PasswordResetStore{Idempotency: idempotency.Store{}}
	sessionStore, err := session.NewStore(session.StoreDependencies{Transactions: w.f.pool, PasswordResetGate: w.gate})
	if err != nil {
		t.Fatalf("real session store failed: %v", err)
	}
	w.sessionStore = sessionStore
	w.maintenanceStore, err = session.NewMaintenanceStore(session.MaintenanceStoreDependencies{
		Transactions: w.f.pool, PasswordResetGate: w.gate, Authority: authority,
	})
	if err != nil {
		t.Fatalf("real maintenance store failed: %v", err)
	}
	issuer, err := identity.NewRandomLoginSessionIssuer(rand.Reader, s420IDAdapter{}, s420ClockAdapter{now: clock}, 32, 32, 24*time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("real session issuer failed: %v", err)
	}
	w.passwordService, err = identity.NewPasswordChangeService(identity.PasswordChangeDependencies{
		RecoveryAuthority: authority, Transactions: w.f.pool, Repository: identity.PostgresPasswordChangeRepository{},
		PasswordResetGate: w.gate, Passwords: w.passwords, Policy: security.PasswordPolicy{},
		Sessions:      identity.PostgresPasswordChangeSessions{Store: sessionStore},
		SessionIssuer: issuer, Audit: audit.Store{}, IDs: s420IDAdapter{}, Clock: s420ClockAdapter{now: clock},
	})
	if err != nil {
		t.Fatalf("real password change service failed: %v", err)
	}
}

// s420WireReplay seals the single article publication seam on the world's one
// PostgresReplayEvidenceStore against rt and builds the real coordinator over
// the same run store and journal. It never creates a second store, pool or
// runtime: the caller supplies the runtime that owns the seam, so the store is
// sealed exactly once and the chain, coordinator and root all share it.
func (w *s420RecoveryWorld) s420WireReplay(t *testing.T, rt recovery.Runtime, consumer *s420LateConsumer) {
	t.Helper()
	w.chain = s420NewReplayChain(t, w.f, w.deletionJournal, w.finalizer, rt, w.closures, w.installation.InstallationID, w.backupRef)
	tap := &s420RunStoreTap{inner: w.store}
	w.runStoreTap = tap
	coordinator, err := recovery.NewCoordinator(
		tap, w.journal, w.chain.provider, w.chain.ports,
		recovery.NewReplayStatusV1Decoder(nil), func() time.Time { return w.f.now },
	)
	if err != nil {
		t.Fatalf("real recovery coordinator failed: %v", err)
	}
	w.coordinator = coordinator
	if consumer != nil {
		consumer.set(coordinator)
	}
}

// s420RunStoreTap forwards the coordinator's run store calls unchanged while
// recording the concrete store it wraps and counting transitions. It lets a
// leaf prove the coordinator and the production root share the one run store
// pointer, and that a replay reads without writing, without adding a second
// store, pool or runtime.
type s420RunStoreTap struct {
	inner *identity.RecoveryRunStore

	mu          sync.Mutex
	transitions int
}

func (tap *s420RunStoreTap) Load(ctx context.Context, runID string) (recovery.RecoveryRun, error) {
	return tap.inner.Load(ctx, runID)
}

func (tap *s420RunStoreTap) Transition(ctx context.Context, runID string, expected, next recovery.RecoveryOutcome, change recovery.RecoveryRunChange) (recovery.RecoveryRun, error) {
	tap.mu.Lock()
	tap.transitions++
	tap.mu.Unlock()
	return tap.inner.Transition(ctx, runID, expected, next, change)
}

func (tap *s420RunStoreTap) transitionCount() int {
	tap.mu.Lock()
	defer tap.mu.Unlock()
	return tap.transitions
}

func (w *s420RecoveryWorld) s420Seed(t *testing.T, withMember bool) {
	t.Helper()
	w.userID = s420NewUUID()
	now := w.f.now
	username := "s420-recovery-admin-" + w.userID[:8]
	email := username + "@example.test"
	phone := "+1555" + s420Digits(w.userID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, is_initial_admin, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s420-initial-admin-digest','admin','active','permanent',false,0,0,true,$5,$5)`,
		w.userID, username, email, phone, now); err != nil {
		t.Fatalf("seed initial admin failed: %v", err)
	}
	for _, identifier := range [][2]string{{"username", username}, {"email", email}, {"phone", phone}} {
		if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO identity_identifiers(
			user_id, kind, normalized_value, lifecycle, created_at) VALUES($1,$2,$3,'active',$4)`,
			w.userID, identifier[0], identifier[1], now); err != nil {
			t.Fatalf("seed admin identifier %s failed: %v", identifier[0], err)
		}
	}
	if !withMember {
		return
	}
	w.memberID = s420NewUUID()
	memberName := "s420-recovery-member-" + w.memberID[:8]
	memberEmail := memberName + "@example.test"
	memberPhone := "+1555" + s420Digits(w.memberID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s420-member-digest','member','active','permanent',false,0,0,$5,$5)`,
		w.memberID, memberName, memberEmail, memberPhone, now); err != nil {
		t.Fatalf("seed recovery member failed: %v", err)
	}
	for _, identifier := range [][2]string{{"username", memberName}, {"email", memberEmail}, {"phone", memberPhone}} {
		if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO identity_identifiers(
			user_id, kind, normalized_value, lifecycle, created_at) VALUES($1,$2,$3,'active',$4)`,
			w.memberID, identifier[0], identifier[1], now); err != nil {
			t.Fatalf("seed member identifier %s failed: %v", identifier[0], err)
		}
	}
}

// s420SeedFinalizedDeletion writes one real journaled deletion and finalizes
// it through the real article.DeletionFinalizer over PostgreSQL, so the real
// replay/publication chain has one coherent tombstone to replay.
func (w *s420RecoveryWorld) s420SeedFinalizedDeletion(t *testing.T) (userID, articleID, operationID string) {
	t.Helper()
	userID, articleID, operationID = w.userID, s420NewUUID(), s420NewUUID()
	now := w.f.now
	tombstone, err := w.deletionJournal.Append(w.f.ctx, article.DeletionTombstoneRequest{
		ArticleID: articleID, ArticleVersion: 3, DeletionOperationID: operationID, DeletedAt: now,
	})
	if err != nil {
		t.Fatalf("real deletion journal Append failed: %v", err)
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO deletion_operations(
		id, article_id, article_version, delete_kind, status, tombstone_sequence,
		journal_key_id, journal_key_version, actor_user_id, actor_role, started_at)
		VALUES($1,$2,3,'ordinary','journaled',$3,$4,$5,$6,'admin',$7)`,
		operationID, articleID, tombstone.Sequence, tombstone.KeyID, tombstone.KeyVersion, userID, now); err != nil {
		t.Fatalf("seed deletion operation failed: %v", err)
	}
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO articles(
		id, author_id, title, body, publication_state, violation_state, deletion_state,
		deletion_operation_id, version, created_at, updated_at)
		VALUES($1,$2,'S4.17 recovery title','S4.17 recovery body','published','clear','deletion_pending',$3,4,$4,$4)`,
		articleID, userID, operationID, now); err != nil {
		t.Fatalf("seed pending recovery article failed: %v", err)
	}
	traceID, err := platform.NewTraceID()
	if err != nil {
		t.Fatalf("generate deletion trace ID failed: %v", err)
	}
	if err := w.finalizer.Finalize(w.f.ctx, operationID, traceID); err != nil {
		t.Fatalf("real deletion finalizer failed: %v", err)
	}
	return userID, articleID, operationID
}

// s420BootstrapIdentity performs the real maintenance work that production
// operators must complete inside the live PhaseRecoveryIdentity window: one
// bootstrap maintenance session, the real ChangeRecoveryPassword transition and,
// optionally, the real canonical kept-deleted review through the real
// recoveryExecutionGate.
func (w *s420RecoveryWorld) s420BootstrapIdentity(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, reviewMember bool) error {
	if err := w.runtime.CheckPhase(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	binding, err := w.runtime.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	var generation int64
	if err := w.f.pool.QueryRow(ctx, "SELECT auth_generation FROM users WHERE id=$1", w.userID).Scan(&generation); err != nil {
		return err
	}
	tx, err := w.f.pool.Begin(ctx)
	if err != nil {
		return err
	}
	sessionID := s420NewUUID()
	token := "s420-maintenance-" + s420NewUUID()
	csrf := "s420-maintenance-csrf-" + s420NewUUID()
	_, err = w.maintenanceStore.CreateMaintenance(ctx, tx, session.CreateInput{
		ID: sessionID, UserID: w.userID, Token: token, CSRFToken: csrf,
		AuthenticatedCredentialKind: session.CredentialRecoveryBootstrap, AuthGeneration: generation,
		IssuedAt: w.f.now, AbsoluteExpiresAt: w.f.now.Add(24 * time.Hour), IdleExpiresAt: w.f.now.Add(time.Hour),
	}, session.RecoverySessionBinding{
		RunID: binding.RunID, InstallationID: binding.InstallationID,
		RuntimeID: binding.RuntimeID, Generation: binding.Generation,
	})
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("create bootstrap maintenance session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	traceID, _ := platform.TraceIDFromContext(ctx)
	if _, err := w.passwordService.ChangeRecoveryPassword(ctx, p, identity.PasswordChangeRequest{
		Principal: session.Principal{
			UserID: w.userID, SessionID: sessionID, Role: "admin", Status: "active", IsInitialAdmin: true,
		},
		CurrentSessionToken: token, CurrentPassword: string(w.secret),
		NewPassword: "SyntheticReplacement123", TraceID: traceID,
	}); err != nil {
		return fmt.Errorf("real recovery password change: %w", err)
	}
	if !reviewMember {
		return nil
	}
	if err := w.maintenance.ResolveRestrictedAccountReview(ctx, p, b, w.memberID, "kept_deleted", "s420-recovery-review-evidence"); err != nil {
		return fmt.Errorf("real canonical kept-deleted review: %w", err)
	}
	return nil
}

// TestRecoveryProductionChainComplete proves the resume production chain on
// the one cmd/han-server root: global recovery authority, request isolation,
// resume completion, public_ready and stream opening. Every leaf observes
// persisted state; no leaf substitutes a fake, a log string or a call count
// for a durable result.
func TestRecoveryProductionChainComplete(t *testing.T) {
	t.Run("harness", testRecoveryChainHarness)
	t.Run("ResumeCoordinatorRecoveryCompleteAndOpen", testRecoveryResumeCoordinatorRecoveryCompleteAndOpen)
	t.Run("SecondReadStatusAndMarkConsumed", testRecoverySecondReadStatusAndMarkConsumed)
	t.Run("CanonicalReplaySameResult", testRecoveryCanonicalReplaySameResult)
	t.Run("ClosedUnknownKeepIsolation", testRecoveryClosedUnknownKeepIsolation)
	t.Run("SingleRuntimeStorePool", testRecoverySingleRuntimeStorePool)
	t.Run("SimulatedJournalKeyringBoundary", testRecoverySimulatedJournalKeyringBoundary)
	t.Run("NoSecondRuntimeRouterPoolAuthorityFallback", testRecoveryNoSecondRuntimeRouterPoolAuthorityFallback)
}

type recoveryChainDB struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	schema string
}

// recoveryChainHarnessDB opens the one test database from HAN_TEST_POSTGRES_DSN,
// creates a fresh schema and applies the approved migration allowlist.
func recoveryChainHarnessDB(t *testing.T) *recoveryChainDB {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	config, err := parseDatabasePoolConfig(testenv.RequirePostgresDSN(t), func(string) string { return "" })
	if err != nil {
		t.Fatalf("parse test DSN: %v", err)
	}
	nonce := make([]byte, 12)
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal("schema nonce")
	}
	schema := "han_s420_" + hex.EncodeToString(nonce)
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	h := &recoveryChainDB{ctx: ctx, pool: pool, schema: schema}
	h.migrate(t)
	t.Logf("s420_schema=%s", schema)
	return h
}

func (h *recoveryChainDB) migrate(t *testing.T) {
	t.Helper()
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql", "0011_article_replay_publication.sql"}
	files := fstest.MapFS{}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatalf("read approved migration %s: %v", name, err)
		}
		files["approved/"+name] = &fstest.MapFile{Data: raw}
	}
	runner, err := migration.NewRunner(h.pool, files, "approved")
	if err != nil {
		t.Fatalf("migration constructor: %v", err)
	}
	if err = runner.Run(h.ctx); err != nil {
		t.Fatalf("approved migration failed: %v", err)
	}
}

// testRecoveryChainHarness proves the real database is reachable, migrated and
// able to hold a seeded Running run with its audit and initial-admin proof.
func testRecoveryChainHarness(t *testing.T) {
	h := recoveryChainHarnessDB(t)
	store, err := identity.NewRecoveryRunStore(h.pool)
	if err != nil {
		t.Fatalf("recovery run store: %v", err)
	}
	if store == nil {
		t.Fatal("recovery run store is nil")
	}
	if _, err = h.pool.Exec(h.ctx, `INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,installation_id,target_digest,commitment_ref,commitment_digest,recovery_journal_id,article_journal_id,recovery_epoch,article_epoch,recovery_initial_sequence,article_initial_sequence,recovery_initial_hash,article_initial_hash,revision,installed_at) VALUES(true,'han-recovery-install/v1','22222222-2222-4222-8222-222222222222','33333333-3333-4333-8333-333333333333',$1,'s420-commitment',$2,'s420-recovery-journal','s420-article-journal',1,1,0,0,'','',1,$3)`, "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64), time.Unix(1790000000, 0).UTC()); err != nil {
		t.Fatalf("seed installation binding: %v", err)
	}
	if _, err = h.pool.Exec(h.ctx, `INSERT INTO recovery_runs(run_id,backup_ref,journal_key_id,journal_key_version,journal_secret_fingerprint,journal_stage,started_at,outcome,audit_event_id) VALUES('s420-run','s420-backup','s420-key','v1','s420-fingerprint','begun',$1,'running','11111111-1111-4111-8111-111111111111')`, time.Unix(1790000000, 0).UTC()); err != nil {
		t.Fatalf("seed running run: %v", err)
	}
	var runID, outcome, auditEvent string
	if err = h.pool.QueryRow(h.ctx, `SELECT run_id,outcome,audit_event_id::text FROM recovery_runs WHERE run_id='s420-run'`).Scan(&runID, &outcome, &auditEvent); err != nil {
		t.Fatalf("read seeded run: %v", err)
	}
	if runID != "s420-run" || outcome != "running" || auditEvent != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("seeded run = %q/%q/%q", runID, outcome, auditEvent)
	}
}

type s420ChainWorker struct {
	started chan struct{}
	done    chan struct{}
	start   sync.Once
	stop    sync.Once
}

func newS420ChainWorker() *s420ChainWorker {
	return &s420ChainWorker{started: make(chan struct{}), done: make(chan struct{})}
}

func (w *s420ChainWorker) Reconcile(context.Context) error { return nil }

func (w *s420ChainWorker) Start(context.Context) { w.start.Do(func() { close(w.started) }) }

func (w *s420ChainWorker) Stop() { w.stop.Do(func() { close(w.done) }) }

func (w *s420ChainWorker) Wait(context.Context) error { return nil }

func (w *s420ChainWorker) Done() <-chan struct{} { return w.done }

func (w *s420ChainWorker) Err() error { return nil }

func (w *s420ChainWorker) Started() <-chan struct{} { return w.started }

func (w *s420ChainWorker) BeginShutdown(context.Context) {}

var errS420BootstrapOnly = errors.New("s420: stop after the real identity bootstrap")

func s420ResumeStartupPlan(t *testing.T, run recovery.RunBinding) recoveryStartupPlan {
	t.Helper()
	values := recoveryConfigFixture(t)
	root := t.TempDir()
	values["HAN_RECOVERY_MODE"] = "maintenance"
	values["HAN_RECOVERY_RESTORE_EXECUTABLE"] = filepath.Join(root, "restore")
	values["HAN_RECOVERY_RESTORE_MANIFEST_PATH"] = filepath.Join(root, "restore-manifest.json")
	values["HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256"] = strings.Repeat("a", 64)
	values["HAN_RECOVERY_CONTROL_HTTP_ADDR"] = "127.0.0.1:18080"
	values["HAN_RECOVERY_CONTROL_ORIGIN"] = "https://control.example.test"
	values["HAN_HTTP_ADDR"] = "127.0.0.1:18081"
	values["HAN_METRICS_ADDR"] = "127.0.0.1:18082"
	command := recoveryCommand{action: "resume", binding: run}
	plan, err := planRecoveryStartup(func(key string) string { return values[key] }, &command)
	if err != nil {
		t.Fatalf("plan resume startup: %v", err)
	}
	if !plan.present || !plan.maintenance || plan.action != "resume" {
		t.Fatalf("startup plan present=%v maintenance=%v action=%q, want maintenance resume", plan.present, plan.maintenance, plan.action)
	}
	if plan.config.ioBudget <= 0 || plan.config.drainBudget <= 0 {
		t.Fatalf("startup budgets io=%v drain=%v, want positive", plan.config.ioBudget, plan.config.drainBudget)
	}
	return plan
}

func s420WaitLeaseReleased(t *testing.T, ctx context.Context, world *s420RecoveryWorld) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if world.exclusive.holders() == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("exclusive lease still held after the failed bootstrap window")
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait lease release: %v", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func s420WaitIdentityReady(t *testing.T, ctx context.Context, assembly *recoveryRootAssembly) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		if assembly.identityReady() {
			return
		}
		if assembly.flowFinished() {
			t.Fatalf("production resume flow finished before identity_ready: %v", assembly.flowErr)
		}
		if time.Now().After(deadline) {
			t.Fatalf("identity_ready not reached within 60s")
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait identity_ready: %v", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// s420RunProductionResume drives one complete production resume over the one
// root and returns the same world, assembly, replay chain, coordinator and run
// it drove, so each leaf observes the durable result of its own run.
func s420RunProductionResume(t *testing.T) *s420ProductionResume {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	t.Cleanup(cancel)

	trace, traceErr := platform.NewTraceID()
	if traceErr != nil {
		t.Fatalf("generate recovery trace ID: %v", traceErr)
	}
	traced, traceErr := platform.ContextWithTraceID(ctx, trace)
	if traceErr != nil {
		t.Fatalf("attach recovery trace ID: %v", traceErr)
	}
	ctx = traced

	world := s420NewRecoveryWorld(t, false, nil, func(w *s420RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
		w.deferReplayWiring = true
		return func(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
			if err := w.s420BootstrapIdentity(ctx, permit, binding, false); err != nil {
				return err
			}
			return errS420BootstrapOnly
		}
	})
	world.s420SeedFinalizedDeletion(t)

	run := recovery.RunBinding{
		RunID:     fmt.Sprintf("s420-chain-%d", time.Now().UnixNano()),
		BackupRef: world.backupRef,
	}
	if err := world.runtime.Enter(ctx, recovery.RecoveryInput{RunBinding: run, Secret: append([]byte(nil), world.secret...)}); !errors.Is(err, errS420BootstrapOnly) {
		t.Fatalf("first identity bootstrap window: %v", err)
	}
	s420WaitLeaseReleased(t, ctx, world)

	result := &s420ProductionResume{ctx: ctx, world: world, run: run}
	var bootOutcome, bootAudit, bootAdmin, bootGeneration string
	if err := world.f.pool.QueryRow(ctx, `SELECT outcome, COALESCE(audit_event_id::text, ''), COALESCE(initial_admin_user_id::text, ''), COALESCE(initial_admin_bootstrap_generation::text, '') FROM recovery_runs WHERE run_id = $1`, run.RunID).Scan(&bootOutcome, &bootAudit, &bootAdmin, &bootGeneration); err != nil {
		t.Fatalf("read bootstrap run row: %v", err)
	}
	if bootOutcome != "running" || bootAudit == "" || bootAdmin == "" || bootGeneration == "" {
		t.Fatalf("bootstrap run row outcome=%q audit=%q admin=%q generation=%q, want running with real identity evidence", bootOutcome, bootAudit, bootAdmin, bootGeneration)
	}
	result.bootAudit, result.bootGeneration = bootAudit, bootGeneration

	plan := s420ResumeStartupPlan(t, run)
	result.plan = plan
	prepared := preparedLoginDependencies{
		passwords: world.passwords,
		ids:       s420IDAdapter{},
		clock:     s420ClockAdapter{now: func() time.Time { return world.f.now }},
	}
	lateConsumer := &s420LateConsumer{}
	inputs := recoveryRootInputs{
		pool:             world.f.pool,
		plan:             plan,
		prepared:         &prepared,
		runs:             world.store,
		closures:         world.closures,
		journal:          world.journal,
		articleSnapshots: world.deletionJournal,
		restore:          world.restore,
		consumer:         lateConsumer,
		installation:     world.installationExecutor,
		articleAccount:   world.port,
		exclusive:        world.exclusive,
		commitments:      world.commitments,
		sources:          world.sources,
		anchor:           world.deletionJournal,
	}
	if inputs.runs != world.store {
		t.Fatalf("root run store = %p, want the one world run store %p", inputs.runs, world.store)
	}
	assembly, err := newRecoveryRootAssembly(ctx, inputs)
	if err != nil {
		t.Fatalf("assemble production recovery root: %v", err)
	}
	result.assembly = assembly

	world.s420WireReplay(t, assembly.runtime, lateConsumer)
	result.chain, result.coordinator = world.chain, world.coordinator
	if result.chain.store != world.closures || result.chain.ports != world.closures {
		t.Fatalf("replay chain store=%p ports=%p, want the one world evidence store %p", result.chain.store, result.chain.ports, world.closures)
	}
	if world.runStoreTap == nil {
		t.Fatal("coordinator run store tap missing")
	}
	if world.runStoreTap.inner != inputs.runs {
		t.Fatalf("coordinator run store = %p, want the one root run store %p", world.runStoreTap.inner, inputs.runs)
	}

	control := newRecoveryTestControlServer()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen control server: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	joined, err := assembly.bindControl(control, listener)
	if err != nil {
		t.Fatalf("bind control server: %v", err)
	}
	go func() { _ = joined.Serve(listener) }()
	assembly.bindArticleOwner(func() error { return nil }, func(context.Context) error { return nil })
	assembly.bindWorkers(newS420ChainWorker(), newS420ChainWorker(), newS420ChainWorker())

	// F002: this harness drives the real runtime.Resume and the real
	// completeIdentity executor manually instead of taking assembly.start's
	// resume branch. assembly.start starts completeOnRequest inline, which calls
	// requestStop() as soon as completion returns; that would tear down the live
	// permit before a leaf could observe the published status under it. The
	// manual requestComplete()+completeIdentity() combination is the same
	// production orchestration the request path uses, so this is an equivalent
	// composition, not the assembly.start path.
	flowCtx, flowCancel := context.WithTimeout(ctx, plan.config.ioBudget)
	// The runtime derives its generation context from the resume flow context, so
	// releasing the flow cancels the live permit. Keep the flow context alive for
	// the whole leaf and release it in cleanup order after the runtime stop.
	t.Cleanup(flowCancel)
	resumeDone := make(chan error, 1)
	go func() { resumeDone <- assembly.runtime.Resume(flowCtx, plan.config.command.binding) }()
	s420WaitIdentityReady(t, ctx, assembly)
	if err := assembly.composition.requestComplete(); err != nil {
		t.Fatalf("request identity completion: %v", err)
	}
	if err := assembly.composition.completeIdentity(ctx); err != nil {
		t.Fatalf("complete identity through the real executor: %v", err)
	}
	var resumeErr error
	select {
	case resumeErr = <-resumeDone:
	case <-ctx.Done():
		t.Fatalf("wait production resume flow: %v", ctx.Err())
	}
	if resumeErr != nil {
		t.Fatalf("production resume flow: %v", resumeErr)
	}
	observation, err := assembly.runtime.Verify(ctx)
	if err != nil {
		t.Fatalf("verify runtime after resume: %v", err)
	}
	result.state = observation.State
	if observation.State != "public_ready" {
		t.Fatalf("runtime state after resume = %q, want public_ready", observation.State)
	}
	// The runtime stays live until the leaf returns so a leaf can read the
	// published status under the same permit and binding the resume used. F003:
	// no stop cleanup is registered because requestStop() is a no-op in this
	// harness -- bindStop has no callers anywhere in the repo -- so the resume
	// flow context, released by its own cleanup, is what ends the generation.

	row := &s420ResumeRow{}
	if err := world.f.pool.QueryRow(ctx, `SELECT outcome, COALESCE(journal_stage, ''), COALESCE(journal_consumed_at::text, ''), COALESCE(article_replay_verified_at::text, ''), COALESCE(article_evidence_ref, ''), COALESCE(article_evidence_digest, ''), COALESCE(article_publication_ref, ''), COALESCE(article_publication_digest, ''), COALESCE(completed_at::text, ''), COALESCE(audit_event_id::text, ''), COALESCE(initial_admin_bootstrap_generation::text, '') FROM recovery_runs WHERE run_id = $1`, run.RunID).Scan(&row.outcome, &row.journalStage, &row.consumedAt, &row.verifiedAt, &row.evidenceRef, &row.evidenceDigest, &row.publicationRef, &row.publicationDigest, &row.completedAt, &row.auditEvent, &row.bootstrapGeneration); err != nil {
		t.Fatalf("read resumed run row: %v", err)
	}
	if row.outcome != "succeeded" {
		t.Fatalf("run outcome = %q, want succeeded", row.outcome)
	}
	if row.journalStage != "consumed" {
		t.Fatalf("run journal stage = %q, want consumed", row.journalStage)
	}
	for name, value := range map[string]string{
		"journal_consumed_at":        row.consumedAt,
		"article_replay_verified_at": row.verifiedAt,
		"article_evidence_ref":       row.evidenceRef,
		"article_evidence_digest":    row.evidenceDigest,
		"article_publication_ref":    row.publicationRef,
		"article_publication_digest": row.publicationDigest,
		"completed_at":               row.completedAt,
	} {
		if value == "" {
			t.Fatalf("%s persisted empty, want the resume completion record", name)
		}
	}
	if row.auditEvent != bootAudit || row.bootstrapGeneration != bootGeneration {
		t.Fatalf("identity evidence drifted: audit %q/%q generation %q/%q", row.auditEvent, bootAudit, row.bootstrapGeneration, bootGeneration)
	}

	var pubRef, pubDigest, pubEvidence string
	if err := world.f.pool.QueryRow(ctx, `SELECT publication_ref, publication_digest, evidence_ref FROM article_replay_publications WHERE run_id = $1`, run.RunID).Scan(&pubRef, &pubDigest, &pubEvidence); err != nil {
		t.Fatalf("read replay publication row: %v", err)
	}
	if pubRef != row.publicationRef || pubDigest != row.publicationDigest || pubEvidence != row.evidenceRef {
		t.Fatalf("publication row ref=%q digest=%q evidence=%q does not match run row ref=%q digest=%q evidence=%q", pubRef, pubDigest, pubEvidence, row.publicationRef, row.publicationDigest, row.evidenceRef)
	}
	result.row = *row
	if err := world.f.pool.QueryRow(ctx, `SELECT journal_consumed_at FROM recovery_runs WHERE run_id = $1`, run.RunID).Scan(&result.row.consumedAtValue); err != nil {
		t.Fatalf("read journal consumed timestamp: %v", err)
	}
	return result
}

// s420ResumeRow is the durable completion record read back from recovery_runs.
type s420ResumeRow struct {
	outcome, journalStage, consumedAt, verifiedAt string
	evidenceRef, evidenceDigest                   string
	publicationRef, publicationDigest             string
	completedAt, auditEvent, bootstrapGeneration  string
	consumedAtValue                               *time.Time
}

// s420ProductionResume is one complete production resume: the one world, root
// assembly, replay chain, coordinator and run that drove it, plus the row the
// resume itself persisted.
type s420ProductionResume struct {
	ctx            context.Context
	world          *s420RecoveryWorld
	assembly       *recoveryRootAssembly
	chain          *s420ReplayChain
	coordinator    *recovery.Coordinator
	run            recovery.RunBinding
	plan           recoveryStartupPlan
	state          string
	bootAudit      string
	bootGeneration string
	row            s420ResumeRow
}

func testRecoveryResumeCoordinatorRecoveryCompleteAndOpen(t *testing.T) {
	res := s420RunProductionResume(t)
	if res.state != "public_ready" {
		t.Fatalf("runtime state after resume = %q, want public_ready from the real activate", res.state)
	}
	row := res.row
	if row.outcome != "succeeded" {
		t.Fatalf("run outcome = %q, want succeeded", row.outcome)
	}
	if row.journalStage != "consumed" {
		t.Fatalf("run journal stage = %q, want consumed", row.journalStage)
	}
	for name, value := range map[string]string{
		"journal_consumed_at":        row.consumedAt,
		"article_replay_verified_at": row.verifiedAt,
		"article_evidence_ref":       row.evidenceRef,
		"article_evidence_digest":    row.evidenceDigest,
		"article_publication_ref":    row.publicationRef,
		"article_publication_digest": row.publicationDigest,
		"completed_at":               row.completedAt,
	} {
		if value == "" {
			t.Fatalf("%s persisted empty, want the resume completion record", name)
		}
	}
	if row.auditEvent != res.bootAudit || row.bootstrapGeneration != res.bootGeneration {
		t.Fatalf("identity evidence drifted: audit %q/%q generation %q/%q", row.auditEvent, res.bootAudit, row.bootstrapGeneration, res.bootGeneration)
	}
	var pubRef, pubDigest, pubEvidence string
	if err := res.world.f.pool.QueryRow(res.ctx, `SELECT publication_ref, publication_digest, evidence_ref FROM article_replay_publications WHERE run_id = $1`, res.run.RunID).Scan(&pubRef, &pubDigest, &pubEvidence); err != nil {
		t.Fatalf("read replay publication row: %v", err)
	}
	if pubRef != row.publicationRef || pubDigest != row.publicationDigest || pubEvidence != row.evidenceRef {
		t.Fatalf("publication row ref=%q digest=%q evidence=%q does not match run row ref=%q digest=%q evidence=%q", pubRef, pubDigest, pubEvidence, row.publicationRef, row.publicationDigest, row.evidenceRef)
	}
}

func testRecoverySecondReadStatusAndMarkConsumed(t *testing.T) {
	res := s420RunProductionResume(t)
	ctx, world, run := res.ctx, res.world, res.run

	if got := world.journal.consumedMarkCount(); got != 1 {
		t.Fatalf("journal MarkConsumed calls = %d, want exactly one from the real coordinator", got)
	}
	callers := world.journal.consumedCallerSnapshot()
	if len(callers) != 1 {
		t.Fatalf("journal MarkConsumed callers = %v, want exactly one", callers)
	}
	if !strings.HasPrefix(callers[0], "coordinator.go:") {
		t.Fatalf("journal MarkConsumed caller = %q, want internal/recovery/coordinator.go", callers[0])
	}
	t.Logf("second_read consumed_marks=%d caller=%s", world.journal.consumedMarkCount(), callers[0])

	snapshot, err := world.journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("read verified journal snapshot: %v", err)
	}
	if snapshot.JournalID != world.installation.RecoveryInitial.JournalID || snapshot.Namespace != recovery.RecoveryNamespaceV1 || snapshot.Epoch != 1 {
		t.Fatalf("journal snapshot identity id=%q namespace=%q epoch=%d", snapshot.JournalID, snapshot.Namespace, snapshot.Epoch)
	}
	if len(snapshot.Entries) != 2 {
		t.Fatalf("journal entries = %d, want the exact begin plus consumed append", len(snapshot.Entries))
	}
	begun, consumed := snapshot.Entries[0], snapshot.Entries[1]
	if begun.Sequence != 1 || consumed.Sequence != 2 {
		t.Fatalf("journal sequences = %d/%d, want 1/2", begun.Sequence, consumed.Sequence)
	}
	if consumed.Stage != recovery.StageConsumed || consumed.RunID != run.RunID || consumed.BackupRef != run.BackupRef {
		t.Fatalf("consumed entry stage=%q run=%q backup=%q, want consumed/%q/%q", consumed.Stage, consumed.RunID, consumed.BackupRef, run.RunID, run.BackupRef)
	}
	if consumed.SecretFingerprint != begun.SecretFingerprint {
		t.Fatalf("consumed fingerprint = %q, want the begun fingerprint %q", consumed.SecretFingerprint, begun.SecretFingerprint)
	}
	if len(snapshot.Prefixes) != 3 {
		t.Fatalf("journal prefixes = %d, want the genesis prefix plus two appends", len(snapshot.Prefixes))
	}
	t.Logf("journal id=%s epoch=%d entries=%d prefixes=%d revision=%q head_seq=%d", snapshot.JournalID, snapshot.Epoch, len(snapshot.Entries), len(snapshot.Prefixes), snapshot.Revision, snapshot.Head.Sequence)
	chain := s420ChainHash(snapshot.Prefixes[1].Hash, consumed)
	wantHMAC := s420HMAC(world.journal.key, []byte(chain+"|"+consumed.SecretFingerprint))
	if consumed.RecordHMAC != wantHMAC {
		t.Fatalf("consumed RecordHMAC = %q, want the appendLocked recomputation %q", consumed.RecordHMAC, wantHMAC)
	}
	last := snapshot.Prefixes[len(snapshot.Prefixes)-1]
	if last.Sequence != 2 || last.Hash != chain || snapshot.Head != last {
		t.Fatalf("journal head seq=%d hash=%q equals_last=%v, want the consumed prefix", last.Sequence, last.Hash, snapshot.Head == last)
	}
	t.Logf("consumed seq=%d stage=%s run=%s fingerprint=%s record_hmac=%s recomputed=%s", consumed.Sequence, consumed.Stage, consumed.RunID, consumed.SecretFingerprint, consumed.RecordHMAC, wantHMAC)

	var journalStage string
	var persistedAt *time.Time
	if err := world.f.pool.QueryRow(ctx, `SELECT journal_stage, journal_consumed_at FROM recovery_runs WHERE run_id = $1`, run.RunID).Scan(&journalStage, &persistedAt); err != nil {
		t.Fatalf("read persisted consume marker: %v", err)
	}
	if journalStage != "consumed" || persistedAt == nil {
		t.Fatalf("persisted consume marker stage=%q at=%v, want consumed with a timestamp", journalStage, persistedAt)
	}
	if !persistedAt.UTC().Equal(consumed.RecordedAt.UTC()) {
		t.Fatalf("persisted journal_consumed_at = %s, want the journal record time %s", persistedAt.UTC(), consumed.RecordedAt.UTC())
	}
	loaded, err := world.store.Load(ctx, run.RunID)
	if err != nil {
		t.Fatalf("load run through the one recovery run store: %v", err)
	}
	if loaded.JournalConsumedAt == nil || !loaded.JournalConsumedAt.UTC().Equal(consumed.RecordedAt.UTC()) {
		t.Fatalf("store JournalConsumedAt = %v, want %s", loaded.JournalConsumedAt, consumed.RecordedAt.UTC())
	}
	if loaded.Outcome != recovery.RecoveryOutcomeSucceeded {
		t.Fatalf("store outcome = %q, want succeeded", loaded.Outcome)
	}
	t.Logf("persisted consumed_at=%s journal_recorded_at=%s stage=%s outcome=%s", persistedAt.UTC().Format(time.RFC3339Nano), consumed.RecordedAt.UTC().Format(time.RFC3339Nano), journalStage, loaded.Outcome)

	runsBefore := s420CountTable(t, ctx, world, "recovery_runs")
	publicationsBefore := s420CountTable(t, ctx, world, "article_replay_publications")
	transitionsBefore := world.runStoreTap.transitionCount()
	first, err := res.chain.provider.ReadStatus(ctx, run.RunID, run.BackupRef)
	if err != nil {
		t.Fatalf("first published status read: %v", err)
	}
	second, err := res.chain.provider.ReadStatus(ctx, run.RunID, run.BackupRef)
	if err != nil {
		t.Fatalf("second published status read: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("second published status differs: %q vs %q", first, second)
	}
	t.Logf("status_read_equal=%v status_bytes=%d", bytes.Equal(first, second), len(first))
	repeated, err := world.journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("read verified journal snapshot after the status reads: %v", err)
	}
	if !reflect.DeepEqual(snapshot, repeated) {
		t.Fatalf("journal snapshot drifted across the status reads: revision %q/%q entries %d/%d prefixes %d/%d", snapshot.Revision, repeated.Revision, len(snapshot.Entries), len(repeated.Entries), len(snapshot.Prefixes), len(repeated.Prefixes))
	}
	if got := world.journal.consumedMarkCount(); got != 1 {
		t.Fatalf("journal MarkConsumed calls after the status reads = %d, want 1", got)
	}
	if got := world.runStoreTap.transitionCount(); got != transitionsBefore {
		t.Fatalf("recovery run store transitions across the status reads = %d, want the %d from resume with no extra write", got, transitionsBefore)
	}
	if got := s420CountTable(t, ctx, world, "recovery_runs"); got != runsBefore || got != 1 {
		t.Fatalf("recovery_runs rows after the status reads = %d, want %d and exactly 1", got, runsBefore)
	}
	if got := s420CountTable(t, ctx, world, "article_replay_publications"); got != publicationsBefore || got != 1 {
		t.Fatalf("article_replay_publications rows after the status reads = %d, want %d and exactly 1", got, publicationsBefore)
	}
	t.Logf("zero_extra_write consumed_marks=%d transitions=%d->%d runs=%d publications=%d snapshot_equal=%v", world.journal.consumedMarkCount(), transitionsBefore, world.runStoreTap.transitionCount(), runsBefore, publicationsBefore, reflect.DeepEqual(snapshot, repeated))

	if _, err := world.journal.Read(ctx, "s420-missing-run"); !errors.Is(err, recovery.ErrAdmissionIncomplete) {
		t.Fatalf("missing run journal read error = %v, want ErrAdmissionIncomplete", err)
	}
	if missing, missingErr := res.chain.provider.ReadStatus(ctx, "s420-missing-run", run.BackupRef); missingErr == nil {
		decoded, decodeErr := recovery.DecodeReplayStatusV1(missing)
		if decodeErr != nil || decoded.Ready != nil || decoded.Unavailable == nil {
			t.Fatalf("missing run status = %q (decode %v), want a fail-closed unavailable status and never ready", missing, decodeErr)
		}
	}

	drifted := loaded
	shifted := consumed.RecordedAt.Add(time.Second)
	drifted.JournalConsumedAt = &shifted
	if _, err := world.journal.VerifyAgainst(ctx, drifted); !errors.Is(err, recovery.ErrAdmissionIncomplete) {
		t.Fatalf("drifted consumed-at verification error = %v, want ErrAdmissionIncomplete", err)
	}
	rebound := loaded
	rebound.JournalSecretFingerprint = "s420-other-fingerprint"
	if _, err := world.journal.VerifyAgainst(ctx, rebound); !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("drifted fingerprint verification error = %v, want ErrAdmissionBinding", err)
	}
	verified, err := world.journal.VerifyAgainst(ctx, loaded)
	if err != nil || verified != consumed {
		t.Fatalf("correct run verification = %+v/%v, want the consumed journal entry", verified, err)
	}
	t.Logf("fail_closed missing_journal=ErrAdmissionIncomplete missing_status_fail_closed=ok drift_consumed_at=ErrAdmissionIncomplete drift_fingerprint=ErrAdmissionBinding verified_seq=%d", verified.Sequence)
}

func s420CountTable(t *testing.T, ctx context.Context, world *s420RecoveryWorld, table string) int {
	t.Helper()
	var count int
	if err := world.f.pool.QueryRow(ctx, `SELECT count(*) FROM `+pgx.Identifier{table}.Sanitize()).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	return count
}

// s420PermitBoundReplay is a TEST-ONLY pass-through around the real deletion
// replay provider. The production provider's ReplayAndVerify/ReadCurrent do not
// check the permit themselves, so this wrapper proves the canonical leaf invoked
// replay under the live permit with the same PermitBinding: it reads the binding
// first and refuses to forward when it is absent or drifted. It never replaces
// the production wiring (res.chain and world.closures are untouched), adds no
// second store/runtime/pool, and forwards to the same real provider instance.
type s420PermitBoundReplay struct {
	inner *article.DeletionReplayProvider
	rt    recovery.Runtime
	want  recovery.PermitBinding

	forwarded atomic.Int64
}

func (w *s420PermitBoundReplay) ReplayAndVerify(ctx context.Context, runID, backupRef string) ([]byte, error) {
	got, err := w.rt.ReadPermitBinding(ctx)
	if err != nil {
		return nil, fmt.Errorf("s420 permit-bound replay: read the live permit binding: %w (underlying: %v)", recovery.ErrAdmissionBinding, err)
	}
	if !reflect.DeepEqual(got, w.want) {
		return nil, fmt.Errorf("s420 permit-bound replay: binding %+v does not match %+v: %w", got, w.want, recovery.ErrAdmissionBinding)
	}
	w.forwarded.Add(1)
	return w.inner.ReplayAndVerify(ctx, runID, backupRef)
}

func (w *s420PermitBoundReplay) forwardedCount() int64 { return w.forwarded.Load() }
func testRecoveryCanonicalReplaySameResult(t *testing.T) {
	res := s420RunProductionResume(t)
	ctx, world, run := res.ctx, res.world, res.run

	lease, err := res.assembly.runtime.Acquire(ctx, recovery.WorkRequest)
	if err != nil {
		t.Fatalf("acquire the live production work lease for the canonical replay: %v", err)
	}
	defer lease.Release()
	bindingBefore, err := res.assembly.runtime.ReadPermitBinding(lease.Context())
	if err != nil {
		t.Fatalf("read the live permit binding before the second replay: %v", err)
	}
	if bindingBefore.RunID != run.RunID || bindingBefore.BackupRef != run.BackupRef {
		t.Fatalf("live permit binding run=%q backup=%q, want the resumed run %q/%q", bindingBefore.RunID, bindingBefore.BackupRef, run.RunID, run.BackupRef)
	}
	runBefore, err := world.store.Load(ctx, run.RunID)
	if err != nil {
		t.Fatalf("load the resumed run before the second replay: %v", err)
	}
	snapshotBefore, err := world.journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("read the verified journal before the second replay: %v", err)
	}
	admissionBefore, err := world.store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		t.Fatalf("read the admission snapshot before the second replay: %v", err)
	}
	admissionBytesBefore, err := recovery.CanonicalAdmissionSnapshot(admissionBefore)
	if err != nil {
		t.Fatalf("canonicalize the admission snapshot before the second replay: %v", err)
	}
	revisionBefore, err := world.store.ReadAdmissionRevision(ctx)
	if err != nil {
		t.Fatalf("read the admission revision before the second replay: %v", err)
	}
	runsBefore := s420CountTable(t, ctx, world, "recovery_runs")
	publicationsBefore := s420CountTable(t, ctx, world, "article_replay_publications")
	attemptsBefore := s420CountTable(t, ctx, world, "deletion_replay_runs")
	evidenceBefore := s420CountTable(t, ctx, world, "deletion_replay_evidence")
	transitionsBefore := world.runStoreTap.transitionCount()
	marksBefore := world.journal.consumedMarkCount()

	// F001: the replay runs under the live permit context, not a plain context.
	raw, err := res.chain.provider.ReplayAndVerify(lease.Context(), run.RunID, run.BackupRef)
	if err != nil {
		t.Fatalf("second canonical replay through the real provider: %v", err)
	}
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil {
		t.Fatalf("decode the second replay status: %v", err)
	}
	if decoded.Ready == nil {
		t.Fatalf("second replay status = %q, want the persisted ready projection", raw)
	}
	if decoded.Ready.RunID != run.RunID || decoded.Ready.BackupRef != run.BackupRef {
		t.Fatalf("second replay ready run=%q backup=%q, want %q/%q", decoded.Ready.RunID, decoded.Ready.BackupRef, run.RunID, run.BackupRef)
	}
	statusDigest, err := decoded.Ready.CanonicalDigest()
	if err != nil {
		t.Fatalf("canonical digest of the second replay status: %v", err)
	}

	// A third replay under the same live permit and binding must return the same
	// persisted bytes; this is the replay-side half of the same-result oracle.
	//
	// F001 executable contrast: the production provider's ReplayAndVerify and
	// ReadCurrent do not check the permit themselves, so a plain context would
	// silently succeed. This test-only pass-through guards on the live permit
	// binding before forwarding: the plain-context arm must fail without
	// touching the real provider, and the live-permit arm must return the same
	// bytes as the direct provider call above. This proves replay ran under the
	// live permit with the same PermitBinding and that the binding did not move;
	// it does not claim provider-side rejection of unbound calls.
	bound := &s420PermitBoundReplay{inner: res.chain.provider, rt: res.assembly.runtime, want: bindingBefore}
	plainRaw, plainErr := bound.ReplayAndVerify(ctx, run.RunID, run.BackupRef)
	if !errors.Is(plainErr, recovery.ErrAdmissionBinding) {
		t.Fatalf("plain-context replay through the permit-bound pass-through raw=%q err=%v, want ErrAdmissionBinding", plainRaw, plainErr)
	}
	if got := bound.forwardedCount(); got != 0 {
		t.Fatalf("plain-context replay reached the real provider %d times, want 0 before the permit guard", got)
	}
	rawAgain, err := bound.ReplayAndVerify(lease.Context(), run.RunID, run.BackupRef)
	if err != nil {
		t.Fatalf("third canonical replay under the live permit through the permit-bound pass-through: %v", err)
	}
	if got := bound.forwardedCount(); got != 1 {
		t.Fatalf("live-permit replay forwarded to the real provider %d times, want exactly 1", got)
	}
	if !bytes.Equal(raw, rawAgain) {
		t.Fatalf("third replay bytes %q differ from the second %q", rawAgain, raw)
	}
	bindingAfter, err := res.assembly.runtime.ReadPermitBinding(lease.Context())
	if err != nil {
		t.Fatalf("read the live permit binding after the second replay: %v", err)
	}
	runAfter, err := world.store.Load(ctx, run.RunID)
	if err != nil {
		t.Fatalf("load the resumed run after the second replay: %v", err)
	}
	snapshotAfter, err := world.journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("read the verified journal after the second replay: %v", err)
	}
	admissionAfter, err := world.store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		t.Fatalf("read the admission snapshot after the second replay: %v", err)
	}
	admissionBytesAfter, err := recovery.CanonicalAdmissionSnapshot(admissionAfter)
	if err != nil {
		t.Fatalf("canonicalize the admission snapshot after the second replay: %v", err)
	}
	revisionAfter, err := world.store.ReadAdmissionRevision(ctx)
	if err != nil {
		t.Fatalf("read the admission revision after the second replay: %v", err)
	}

	if !reflect.DeepEqual(bindingBefore, bindingAfter) {
		t.Fatalf("permit binding drifted across the second replay: before=%+v after=%+v", bindingBefore, bindingAfter)
	}
	if !reflect.DeepEqual(runBefore, runAfter) {
		t.Fatalf("recovery run drifted across the second replay:\nbefore=%+v\nafter=%+v", runBefore, runAfter)
	}
	if runAfter.JournalStage != recovery.StageConsumed || runAfter.JournalConsumedAt == nil {
		t.Fatalf("second replay run stage=%q consumed_at=%v, want the durable consumed record", runAfter.JournalStage, runAfter.JournalConsumedAt)
	}
	if !reflect.DeepEqual(snapshotBefore, snapshotAfter) {
		t.Fatalf("journal snapshot drifted across the second replay: revision %q/%q entries %d/%d head %d/%d", snapshotBefore.Revision, snapshotAfter.Revision, len(snapshotBefore.Entries), len(snapshotAfter.Entries), snapshotBefore.Head.Sequence, snapshotAfter.Head.Sequence)
	}
	if !bytes.Equal(admissionBytesBefore, admissionBytesAfter) {
		t.Fatalf("canonical admission snapshot changed across the second replay (%d -> %d bytes)", len(admissionBytesBefore), len(admissionBytesAfter))
	}
	if !reflect.DeepEqual(revisionBefore, revisionAfter) {
		t.Fatalf("admission revision drifted across the second replay: before=%+v after=%+v", revisionBefore, revisionAfter)
	}
	if statusDigest != runAfter.ArticleEvidenceDigest {
		t.Fatalf("second replay evidence digest = %q, want the persisted %q", statusDigest, runAfter.ArticleEvidenceDigest)
	}
	if runAfter.ArticleEvidenceRef != res.row.evidenceRef || runAfter.ArticleEvidenceDigest != res.row.evidenceDigest {
		t.Fatalf("second replay evidence ref/digest = %q/%q, want the resume row %q/%q", runAfter.ArticleEvidenceRef, runAfter.ArticleEvidenceDigest, res.row.evidenceRef, res.row.evidenceDigest)
	}
	if runAfter.ArticlePublicationRef != res.row.publicationRef || runAfter.ArticlePublicationDigest != res.row.publicationDigest {
		t.Fatalf("second replay publication ref/digest = %q/%q, want the resume row %q/%q", runAfter.ArticlePublicationRef, runAfter.ArticlePublicationDigest, res.row.publicationRef, res.row.publicationDigest)
	}
	// The ready projection records the article journal high-water it verified. The
	// production admission proof compares that head with the article snapshot and
	// the published closure, never the recovery journal, so prove the same durable
	// relationship instead of reading a cross-namespace head.
	articleSnapshot, err := world.deletionJournal.ReadVerifiedSnapshot(ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		t.Fatalf("read the verified article journal after the second replay: %v", err)
	}
	closure, err := res.chain.store.ReadPublishedClosure(ctx, recovery.ClosureRequest{
		RunID: run.RunID, BackupRef: run.BackupRef,
		EvidenceRef: runAfter.ArticleEvidenceRef, EvidenceDigest: runAfter.ArticleEvidenceDigest,
		PublicationRef: runAfter.ArticlePublicationRef, PublicationDigest: runAfter.ArticlePublicationDigest,
	})
	if err != nil {
		t.Fatalf("read the published closure after the second replay: %v", err)
	}
	if !reflect.DeepEqual(closure.Ready, *decoded.Ready) {
		t.Fatalf("published closure ready %+v differs from the replayed status %+v", closure.Ready, *decoded.Ready)
	}
	if closure.CurrentHead != articleSnapshot.Head {
		t.Fatalf("published closure current head = %+v, want the durable article head %+v", closure.CurrentHead, articleSnapshot.Head)
	}
	if closure.HistoricalHead != (recovery.JournalHead{Sequence: decoded.Ready.JournalHeadSequence, Hash: decoded.Ready.JournalHeadHash}) {
		t.Fatalf("published closure historical head = %+v, want the replayed %d/%q", closure.HistoricalHead, decoded.Ready.JournalHeadSequence, decoded.Ready.JournalHeadHash)
	}
	historicalPrefix := false
	for _, prefix := range articleSnapshot.Prefixes {
		if prefix == closure.HistoricalHead {
			historicalPrefix = true
		}
	}
	if !historicalPrefix {
		t.Fatalf("published closure historical head %+v is not a durable article prefix", closure.HistoricalHead)
	}
	if closure.Ready.JournalHeadSequence > articleSnapshot.Head.Sequence {
		t.Fatalf("replayed article head sequence = %d beyond the durable article head %d", closure.Ready.JournalHeadSequence, articleSnapshot.Head.Sequence)
	}
	if snapshotAfter.Head.Sequence != snapshotBefore.Head.Sequence {
		t.Fatalf("journal head sequence advanced across the second replay: %d -> %d", snapshotBefore.Head.Sequence, snapshotAfter.Head.Sequence)
	}
	if got := s420CountTable(t, ctx, world, "recovery_runs"); got != runsBefore || got != 1 {
		t.Fatalf("recovery_runs rows after the second replay = %d, want %d and exactly 1", got, runsBefore)
	}
	if got := s420CountTable(t, ctx, world, "article_replay_publications"); got != publicationsBefore || got != 1 {
		t.Fatalf("article_replay_publications rows after the second replay = %d, want %d and exactly 1", got, publicationsBefore)
	}
	if got := s420CountTable(t, ctx, world, "deletion_replay_runs"); got != attemptsBefore || got != 1 {
		t.Fatalf("deletion_replay_runs rows after the second replay = %d, want %d and exactly 1", got, attemptsBefore)
	}
	if got := s420CountTable(t, ctx, world, "deletion_replay_evidence"); got != evidenceBefore || got != 1 {
		t.Fatalf("deletion_replay_evidence rows after the second replay = %d, want %d and exactly 1", got, evidenceBefore)
	}
	if got := world.runStoreTap.transitionCount(); got != transitionsBefore {
		t.Fatalf("run store transitions across the second replay = %d, want the %d from resume", got, transitionsBefore)
	}
	if got := world.journal.consumedMarkCount(); got != marksBefore || got != 1 {
		t.Fatalf("journal MarkConsumed calls after the second replay = %d, want %d and exactly 1", got, marksBefore)
	}
	t.Logf("canonical_replay_same_result ready_digest=%s binding_equal=%v run_equal=%v snapshot_equal=%v admission_equal=%v revision_equal=%v rows=%d/%d/%d/%d transitions=%d marks=%d head_seq=%d permit_ctx=live_permit plain_ctx=ErrAdmissionBinding provider_binding_check=absent plain_forwarded=0 live_forwarded=1", statusDigest, reflect.DeepEqual(bindingBefore, bindingAfter), reflect.DeepEqual(runBefore, runAfter), reflect.DeepEqual(snapshotBefore, snapshotAfter), bytes.Equal(admissionBytesBefore, admissionBytesAfter), reflect.DeepEqual(revisionBefore, revisionAfter), runsBefore, publicationsBefore, attemptsBefore, evidenceBefore, transitionsBefore, marksBefore, snapshotBefore.Head.Sequence)
}

func testRecoverySingleRuntimeStorePool(t *testing.T) {
	res := s420RunProductionResume(t)

	// The one production root builds exactly one runtime, one identity run
	// store, one article replay evidence store and one pool. Every check below
	// compares pointers; a shared pool alone never proves a single instance.
	if res.chain.store != res.world.closures || res.chain.ports != res.world.closures {
		t.Fatalf("replay chain store=%p ports=%p, want the one article evidence store %p", res.chain.store, res.chain.ports, res.world.closures)
	}
	if res.chain.rt != res.assembly.runtime {
		t.Fatalf("replay chain sealed runtime=%p, want the one production root runtime %p", res.chain.rt, res.assembly.runtime)
	}
	if res.world.runStoreTap == nil || res.world.runStoreTap.inner != res.world.store {
		t.Fatalf("coordinator run store tap does not point at the one run store %p", res.world.store)
	}
	if res.coordinator == nil {
		t.Fatal("production coordinator missing")
	}
	if res.assembly.composition == nil || res.assembly.composition.runtime() != res.assembly.runtime {
		t.Fatalf("composition runtime differs from the one assembled runtime %p", res.assembly.runtime)
	}
	if res.assembly.article == nil {
		t.Fatal("root article control handle missing")
	}
	// F004: uniqueness holds for the PRODUCTION ROOT only. The test-only
	// bootstrap runtime (res.world.runtime) served the card-external pre-resume
	// Enter window; s420WaitLeaseReleased already proved its lease released and
	// it takes no part in the resume. The resume itself ran only on
	// res.assembly.runtime, so this is not a claim that the process hosts a
	// single runtime.
	if res.world.runtime == res.assembly.runtime {
		t.Fatalf("root runtime %p aliases the pre-resume bootstrap runtime", res.assembly.runtime)
	}
	t.Logf("single_runtime_store_pool runtime=%p store=%p closures=%p pool=%p bootstrap_runtime=%p coordinator=%p scope=production_root_only bootstrap_before_resume=true bootstrap_participates_in_resume=false", res.assembly.runtime, res.world.store, res.world.closures, res.world.f.pool, res.world.runtime, res.coordinator)
}

func testRecoverySimulatedJournalKeyringBoundary(t *testing.T) {
	res := s420RunProductionResume(t)
	ctx, world, run := res.ctx, res.world, res.run

	// ADR-0028 boundary: the journal HMAC key is a controlled in-memory
	// software value. No real keyring files or HSM material exist in this
	// environment, so this leaf annotates the boundary and never claims real
	// keyring or HSM verification.
	t.Logf("adapter_scoped=simulated_journal_hmac_keyring; not real keyring files; reason=windows_unsupported")

	snapshot, err := world.journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		t.Fatalf("read the verified journal snapshot: %v", err)
	}
	if snapshot.JournalID != world.installation.RecoveryInitial.JournalID ||
		snapshot.Namespace != world.installation.RecoveryInitial.Namespace ||
		snapshot.Epoch != world.installation.RecoveryInitial.Epoch {
		t.Fatalf("journal identity %q/%q/%d, want the installation binding %+v", snapshot.JournalID, snapshot.Namespace, snapshot.Epoch, world.installation.RecoveryInitial)
	}
	if len(snapshot.Entries) != 2 || len(snapshot.Prefixes) != 3 {
		t.Fatalf("journal entries=%d prefixes=%d, want the begin plus consumed append", len(snapshot.Entries), len(snapshot.Prefixes))
	}
	begun, consumed := snapshot.Entries[0], snapshot.Entries[1]
	for index, entry := range []recovery.JournalEntry{begun, consumed} {
		if entry.KeyID != world.journal.keyID || entry.KeyVersion != world.journal.keyVersion {
			t.Fatalf("entry %d key=%q/%q, want the journal key identity %q/%q", index, entry.KeyID, entry.KeyVersion, world.journal.keyID, world.journal.keyVersion)
		}
	}
	beginChain := s420ChainHash(snapshot.Prefixes[0].Hash, begun)
	beginHMAC := s420HMAC(world.journal.key, []byte(beginChain+"|"+begun.SecretFingerprint))
	if begun.RecordHMAC != beginHMAC || snapshot.Prefixes[1].Hash != beginChain {
		t.Fatalf("begun record HMAC=%q chain=%q, recomputed %q/%q", begun.RecordHMAC, snapshot.Prefixes[1].Hash, beginHMAC, beginChain)
	}
	fingerprint := s420HMAC(world.journal.key, world.secret)
	if begun.SecretFingerprint != fingerprint {
		t.Fatalf("begun secret fingerprint=%q, want the keyring recomputation %q", begun.SecretFingerprint, fingerprint)
	}
	consumedChain := s420ChainHash(beginChain, consumed)
	consumedHMAC := s420HMAC(world.journal.key, []byte(consumedChain+"|"+consumed.SecretFingerprint))
	if consumed.RecordHMAC != consumedHMAC || snapshot.Prefixes[2].Hash != consumedChain || snapshot.Head != snapshot.Prefixes[2] {
		t.Fatalf("consumed record HMAC=%q chain=%q head=%v, recomputed %q/%q", consumed.RecordHMAC, snapshot.Prefixes[2].Hash, snapshot.Head, consumedHMAC, consumedChain)
	}
	if consumed.SecretFingerprint != fingerprint {
		t.Fatalf("consumed secret fingerprint=%q, want the begun fingerprint %q", consumed.SecretFingerprint, fingerprint)
	}
	var keyID, keyVersion, storedFingerprint string
	if err := world.f.pool.QueryRow(ctx, `SELECT journal_key_id, journal_key_version, journal_secret_fingerprint FROM recovery_runs WHERE run_id = $1`, run.RunID).Scan(&keyID, &keyVersion, &storedFingerprint); err != nil {
		t.Fatalf("read the persisted journal key binding: %v", err)
	}
	if keyID != world.journal.keyID || keyVersion != world.journal.keyVersion || storedFingerprint != fingerprint {
		t.Fatalf("persisted journal key=%q/%q fingerprint=%q, want %q/%q/%q", keyID, keyVersion, storedFingerprint, world.journal.keyID, world.journal.keyVersion, fingerprint)
	}
	t.Logf("simulated_journal_keyring key_id=%s key_version=%s begin_hmac=%s consumed_hmac=%s fingerprint=%s entries=%d prefixes=%d head_seq=%d", world.journal.keyID, world.journal.keyVersion, beginHMAC, consumedHMAC, fingerprint, len(snapshot.Entries), len(snapshot.Prefixes), snapshot.Head.Sequence)
}

func testRecoveryNoSecondRuntimeRouterPoolAuthorityFallback(t *testing.T) {
	res := s420RunProductionResume(t)

	// The one production root (newRecoveryRootAssembly) constructs exactly one
	// instance of each authority-bearing component, and the resume path creates
	// none of them again:
	//   runtime     -> the assembled recovery.Runtime on world.store/journal/closures
	//   run store   -> identity.NewRecoveryRunStore over the one pgxpool
	//   closures    -> article.NewPostgresReplayEvidenceStore over the same pool
	//   control     -> the test control server bound through bindControl
	//   handle      -> the root article control handle (assembly.article)
	// F005 scope: this leaf only proves POINTER-LEVEL uniqueness of the
	// components the production root actually references. It does not scan for
	// an unreferenced second router/authority that nothing points at. On this
	// root no public router is built and there is no fallback authority: the
	// composition runtime is the assembled runtime and the coordinator shares
	// the root run store through the tap.
	if res.assembly.composition == nil {
		t.Fatal("root composition missing")
	}
	if res.assembly.composition.runtime() != res.assembly.runtime {
		t.Fatalf("composition runtime %p differs from the assembled runtime %p", res.assembly.composition.runtime(), res.assembly.runtime)
	}
	if res.assembly.article == nil {
		t.Fatal("root article control handle missing")
	}
	if res.chain.rt != res.assembly.runtime {
		t.Fatalf("replay chain sealed runtime %p differs from the assembled runtime %p", res.chain.rt, res.assembly.runtime)
	}
	if res.coordinator == nil || res.world.runStoreTap == nil || res.world.runStoreTap.inner != res.world.store {
		t.Fatalf("coordinator run store fallback detected: tap=%v", res.world.runStoreTap)
	}
	if res.chain.store != res.world.closures || res.chain.ports != res.world.closures {
		t.Fatalf("second article evidence store detected: chain store=%p ports=%p closures=%p", res.chain.store, res.chain.ports, res.world.closures)
	}
	if res.world.runtime == res.assembly.runtime {
		t.Fatalf("bootstrap runtime %p reused as the assembled runtime", res.world.runtime)
	}
	t.Logf("no_second_runtime_router_pool_authority_fallback runtime=%p store=%p closures=%p pool=%p handle=%p coordinator=%p scope=pointer_level_only no_public_router=true fallback_authority=false", res.assembly.runtime, res.world.store, res.world.closures, res.world.f.pool, res.assembly.article, res.coordinator)
}

// s420TransportObserver counts requests that entered the real trace middleware
// and exact-route transport. It sits outside the recovery guard, so a closed
// stream still proves transport entry and trace injection while the inner
// Article handler observer stays at zero.
type s420TransportObserver struct {
	entries atomic.Int64
	traces  atomic.Int64
}

func (o *s420TransportObserver) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		o.entries.Add(1)
		if _, ok := platform.TraceIDFromContext(request.Context()); ok {
			o.traces.Add(1)
		}
		next.ServeHTTP(response, request)
	})
}

func (o *s420TransportObserver) counts() (entries, traces int64) {
	return o.entries.Load(), o.traces.Load()
}

// s420Case18Request drives the real transport, origin guard, session
// authorizer, recovery guard and Article handler with the seeded same-origin
// session.
func s420Case18Request(t *testing.T, router http.Handler, world *s420Case18World, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Origin", world.publicOrigin)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: world.sessionToken})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func s420Envelope(t *testing.T, recorder *httptest.ResponseRecorder) platform.Envelope {
	t.Helper()
	var envelope platform.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode public envelope failed: %v body=%s", err, recorder.Body.String())
	}
	return envelope
}

func s420CountPoolTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+pgx.Identifier{table}.Sanitize()).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	return count
}

func testRecoveryClosedUnknownKeepIsolation(t *testing.T) {
	world := s420NewCase18World(t)
	ctx := world.f.ctx

	observation, err := world.full.runtime.Verify(ctx)
	if err != nil || observation.State == "public_ready" {
		t.Fatalf("closed runtime observation=%#v err=%v, want a non-ready state", observation, err)
	}
	// F006: the trace middleware and recovery guard are the real production
	// components, but the exact-route transport below is a TEST adapter
	// (s420NewCase18Transport) that mounts the real Article handler and counts
	// entries. Production dispatcher closed/unknown behaviour is covered
	// separately by TestRecoveryMainBusinessDispatcher, not by this leaf.
	observer := &s420TransportObserver{}
	router := platform.TraceMiddleware(observer.wrap(s420NewCase18Transport(world)))

	assertIsolated := func(stage string) {
		t.Helper()
		handlerBefore, traceBefore, serviceBefore, repositoryBefore, authorBefore := world.counts()
		for _, path := range []string{world.articlePath(), "/api/v1/articles"} {
			recorder := s420Case18Request(t, router, world, http.MethodGet, path)
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("%s GET %s status=%d, want 503 body=%s", stage, path, recorder.Code, recorder.Body.String())
			}
			envelope := s420Envelope(t, recorder)
			if envelope.Code != platform.CodePlatformRecoveryInProgress {
				t.Fatalf("%s GET %s code=%q, want %q", stage, path, envelope.Code, platform.CodePlatformRecoveryInProgress)
			}
			if envelope.TraceID == "" || envelope.TraceID != recorder.Header().Get(platform.TraceHeader) {
				t.Fatalf("%s GET %s trace=%q header=%q, want the injected trace", stage, path, envelope.TraceID, recorder.Header().Get(platform.TraceHeader))
			}
		}
		handlerAfter, traceAfter, serviceAfter, repositoryAfter, authorAfter := world.counts()
		if handlerAfter != handlerBefore || traceAfter != traceBefore {
			t.Fatalf("%s closed stream entered the Article handler observer: handler %d->%d trace %d->%d", stage, handlerBefore, handlerAfter, traceBefore, traceAfter)
		}
		if serviceAfter != serviceBefore || repositoryAfter != repositoryBefore || authorAfter != authorBefore {
			t.Fatalf("%s closed stream reached the Article service/repository/author reader: service %d->%d repository %d->%d author %d->%d", stage, serviceBefore, serviceAfter, repositoryBefore, repositoryAfter, authorBefore, authorAfter)
		}
	}

	persistentBefore := map[string]int{
		"recovery_runs":               s420CountPoolTable(t, ctx, world.f.pool, "recovery_runs"),
		"article_replay_publications": s420CountPoolTable(t, ctx, world.f.pool, "article_replay_publications"),
		"deletion_replay_runs":        s420CountPoolTable(t, ctx, world.f.pool, "deletion_replay_runs"),
		"articles":                    s420CountPoolTable(t, ctx, world.f.pool, "articles"),
	}
	assertIsolated("closed")
	entries, traces := observer.counts()
	if entries != 2 || traces != 2 {
		t.Fatalf("closed transport entries=%d traces=%d, want 2/2 observed outside the guard", entries, traces)
	}

	// F006: the success arm reuses the same real runtime and the TEST transport
	// adapter; RunNormal drives the real activate so the recovery guard opens.
	world.full.s420RunNormal(t)
	detail := s420Case18Request(t, router, world, http.MethodGet, world.articlePath())
	if detail.Code != http.StatusOK {
		t.Fatalf("public detail status=%d want 200 body=%s", detail.Code, detail.Body.String())
	}
	list := s420Case18Request(t, router, world, http.MethodGet, "/api/v1/articles")
	if list.Code != http.StatusOK {
		t.Fatalf("public list status=%d want 200 body=%s", list.Code, list.Body.String())
	}
	handler, trace, service, repository, authorRead := world.counts()
	if service == 0 || repository == 0 || authorRead == 0 {
		t.Fatalf("public stream did not reach the Article service/repository/author reader: handler=%d trace=%d service=%d repository=%d author=%d", handler, trace, service, repository, authorRead)
	}
	t.Logf("public_stream_open handler=%d trace=%d service=%d repository=%d author=%d", handler, trace, service, repository, authorRead)

	// The unknown arm drives the real production writer-uncertainty path on the
	// same runtime: pause a real article append, then fault it so admission is
	// revoked with ErrRecoveryWriteUnknown. No second runtime is built and no
	// state is assigned directly.
	notice, err := world.full.runtime.Acquire(ctx, recovery.WorkArticleNotice)
	if err != nil {
		t.Fatalf("acquire the real article notice lease: %v", err)
	}
	barrier, err := world.full.runtime.PauseArticleAppend(notice.Context(), "s420-unknown-append")
	if err != nil {
		notice.Release()
		t.Fatalf("pause the real article append for the unknown probe: %v", err)
	}
	if _, acquireErr := world.full.runtime.Acquire(ctx, recovery.WorkRequest); !errors.Is(acquireErr, recovery.ErrAdmissionClosed) {
		notice.Release()
		t.Fatalf("acquire while the append is paused err=%v, want ErrAdmissionClosed", acquireErr)
	}
	assertIsolated("unknown-pending")
	faultErr := world.full.runtime.FaultArticleAppend(ctx, barrier, nil)
	notice.Release()
	if !errors.Is(faultErr, recovery.ErrRecoveryWriteUnknown) {
		t.Fatalf("fault the real article append err=%v, want ErrRecoveryWriteUnknown", faultErr)
	}
	unknownObservation, unknownErr := world.full.runtime.Verify(ctx)
	if unknownErr == nil || unknownObservation.State == "public_ready" {
		t.Fatalf("post-fault observation=%#v err=%v, want the real non-authoritative state", unknownObservation, unknownErr)
	}
	if _, acquireErr := world.full.runtime.Acquire(ctx, recovery.WorkRequest); !errors.Is(acquireErr, recovery.ErrAdmissionClosed) {
		t.Fatalf("acquire after the real write-unknown fault err=%v, want ErrAdmissionClosed", acquireErr)
	}
	t.Logf("unknown_fault cause=%v observation=%q verify_err=%v", faultErr, unknownObservation.State, unknownErr)
	assertIsolated("unknown")

	for table, before := range persistentBefore {
		if after := s420CountPoolTable(t, ctx, world.f.pool, table); after != before {
			t.Fatalf("%s rows after the closed/unknown isolation window = %d, want %d with no durable write", table, after, before)
		}
	}
	t.Logf("closed_unknown_isolation service_zero=true repository_zero=true author_zero=true durable_writes=0 transport=test_adapter success_driver=RunNormal production_dispatcher_coverage=TestRecoveryMainBusinessDispatcher")
}

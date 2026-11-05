//go:build integration

package recovery_test

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
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// s417RandomHex returns a fresh schema suffix so every fixture owns one schema.
func s417RandomHex(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("s417 random suffix failed: %v", err)
	}
	return hex.EncodeToString(buffer)
}

func s417Digest(label string) string { return recovery.SHA256Digest([]byte(label)) }

func s417Canonical(t time.Time) time.Time { return t.UTC().Truncate(time.Microsecond) }

func s417HMAC(key, payload []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}

func s417JSON(value any) string {
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(builder.String(), "\n")
}

// s417ManifestNamespace mirrors the fixed installation commitment encoding. It is
// media-only: it reproduces the canonical bytes the real verifier recomputes, and
// never decides admission state.
func s417ManifestNamespace(a recovery.AnchorSnapshot) string {
	return fmt.Sprintf(`{"namespace":%s,"journal_id":%s,"epoch":%d,"head_sequence":%d,"head_hash":%s}`,
		s417JSON(a.Namespace), s417JSON(a.JournalID), a.Epoch, a.Committed.Sequence, s417JSON(a.Committed.Hash))
}

func s417ManifestDigest(m recovery.InstallationManifest) string {
	body := fmt.Sprintf(`{"version":%s,"kind":%s,"deployment_id":%s,"installation_id":%s,"target_digest":%s,"source_approval_ref":%s,"source_approval_digest":%s,"recovery":%s,"article":%s,"issued_at":%s}`,
		s417JSON(m.Version), s417JSON(m.Kind), s417JSON(m.DeploymentID), s417JSON(m.InstallationID),
		s417JSON(m.TargetDigest), s417JSON(m.SourceApprovalRef), s417JSON(m.SourceApprovalDigest),
		s417ManifestNamespace(m.Recovery), s417ManifestNamespace(m.Article),
		s417JSON(m.IssuedAt.Format("2006-01-02T15:04:05.000000Z")))
	return recovery.SHA256Digest([]byte(body))
}

func s417ChainHash(previous string, e recovery.JournalEntry) string {
	payload := fmt.Sprintf("%s|%s|%d|%s|%s|%s|%s", previous, e.Namespace, e.Sequence, e.RunID,
		e.BackupRef, e.Stage, e.RecordedAt.Format("2006-01-02T15:04:05.000000Z"))
	return s417Digest(payload)
}

// s417Fixture owns one isolated PostgreSQL schema plus the real migrations.
type s417Fixture struct {
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc
	schema string
	pool   *pgxpool.Pool
	now    time.Time
}

func s417NewFixture(t *testing.T) *s417Fixture {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN is required for S4.17; no Skip")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	schema := "han_b13i_s417_" + s417RandomHex(t)
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
	fixture := &s417Fixture{t: t, ctx: ctx, cancel: cancel, schema: schema, pool: pool}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		fixture.close()
		t.Fatalf("create isolated schema failed: %v", err)
	}
	s417RegisterRetainedSchema(t, schema)
	t.Cleanup(fixture.close)
	fixture.migrate()
	fixture.now = s417Canonical(time.Now())
	return fixture
}

// s417RetainedSchemaPath is the author-r02 evidence registry. Schemas are never
// dropped by this suite (retention ruling L01); the registry is append-only.
const s417RetainedSchemaPath = "../../../.spec/wip/recovery-runtime/evidence-b13i-r01/author-r02/retained-schemas.json"

func s417RegisterRetainedSchema(t *testing.T, schema string) {
	t.Helper()
	names := []string{}
	if raw, err := os.ReadFile(s417RetainedSchemaPath); err == nil {
		_ = json.Unmarshal(raw, &names)
	}
	names = append(names, schema)
	sort.Strings(names)
	unique := names[:0]
	for index, name := range names {
		if index == 0 || name != names[index-1] {
			unique = append(unique, name)
		}
	}
	encoded, err := json.MarshalIndent(unique, "", "  ")
	if err != nil {
		t.Fatalf("encode retained schema registry failed: %v", err)
	}
	if err := os.WriteFile(s417RetainedSchemaPath, append(encoded, byte(10)), 0o644); err != nil {
		t.Fatalf("write retained schema registry failed: %v", err)
	}
}

// close never drops the schema (retention ruling L01); it only proves the pool
// has no acquired connections, then releases it.
func (f *s417Fixture) close() {
	if f.pool != nil {
		if stat := f.pool.Stat(); stat.AcquiredConns() != 0 {
			f.t.Errorf("pool acquired conns at cleanup=%d; want 0 (schema %s retained)", stat.AcquiredConns(), f.schema)
		}
		f.pool.Close()
	}
	f.cancel()
}

func (f *s417Fixture) migrate() {
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
		raw, readErr := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
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

func (f *s417Fixture) insertInstallation(inst recovery.InstallationBinding) {
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

func (f *s417Fixture) countRuns() int {
	var count int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM recovery_runs").Scan(&count); err != nil {
		f.t.Fatalf("count recovery runs failed: %v", err)
	}
	return count
}

// softwareJournal is a controlled software media adapter for one journal
// namespace (recovery or article-deletion). This is a controlled software media
// adapter: it does not prove real POSIX file journal / CAS / ACL / approved
// backup / E2E. It only stores hash-chained records and never decides admission.
type softwareJournal struct {
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

func s417NewSoftwareJournal(journalID, namespace string, epoch uint64, now func() time.Time) *softwareJournal {
	return &softwareJournal{
		journalID: journalID, namespace: namespace, epoch: epoch,
		key: []byte("s417-controlled-software-media-key"), keyID: "s417-key", keyVersion: "v1",
		now: now, prefixes: []recovery.JournalHead{{Sequence: 0, Hash: ""}},
	}
}

func (j *softwareJournal) snapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
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

func (j *softwareJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
	return j.snapshot(ctx, namespace)
}

func (j *softwareJournal) appendLocked(entry recovery.JournalEntry) recovery.JournalEntry {
	previous := j.prefixes[len(j.prefixes)-1]
	entry.Namespace = j.namespace
	entry.Sequence = uint64(len(j.entries) + 1)
	entry.KeyID = j.keyID
	entry.KeyVersion = j.keyVersion
	entry.RecordedAt = s417Canonical(j.now())
	entry.PreviousRecordHash = previous.Hash
	chain := s417ChainHash(previous.Hash, entry)
	entry.RecordHMAC = s417HMAC(j.key, []byte(chain+"|"+entry.SecretFingerprint))
	j.entries = append(j.entries, entry)
	j.prefixes = append(j.prefixes, recovery.JournalHead{Sequence: entry.Sequence, Hash: chain})
	j.revision++
	return entry
}

func (j *softwareJournal) Begin(ctx context.Context, runID, backupRef string, secret []byte) (recovery.JournalEntry, error) {
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
		SecretFingerprint: s417HMAC(j.key, secret),
	}), nil
}

func (j *softwareJournal) MarkConsumed(ctx context.Context, runID string) (recovery.JournalEntry, error) {
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

func (j *softwareJournal) consumedMarkCount() int64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.consumedCalls
}

func (j *softwareJournal) consumedCallerSnapshot() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]string(nil), j.consumedCallers...)
}

func (j *softwareJournal) latestLocked(runID string) (recovery.JournalEntry, bool) {
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

func (j *softwareJournal) Read(ctx context.Context, runID string) (recovery.JournalEntry, error) {
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

func (j *softwareJournal) VerifyBeginSecret(ctx context.Context, binding recovery.RunBinding, candidate []byte) (recovery.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return recovery.JournalEntry{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, entry := range j.entries {
		if entry.RunID == binding.RunID && entry.Stage == recovery.StageBegun {
			if entry.SecretFingerprint != s417HMAC(j.key, candidate) {
				return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
			}
			return entry, nil
		}
	}
	return recovery.JournalEntry{}, recovery.ErrAdmissionIncomplete
}

func (j *softwareJournal) VerifyAgainst(ctx context.Context, run recovery.RecoveryRun) (recovery.JournalEntry, error) {
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

// softwareExclusive is a controlled software media adapter for the deployment
// exclusive lease. This is a controlled software media adapter: it does not
// prove real POSIX file lock / CAS / ACL / approved backup / E2E.
type softwareExclusive struct {
	mu       sync.Mutex
	held     int
	closed   bool
	acquired int
	rejected int
}

func (e *softwareExclusive) Acquire(ctx context.Context) (recovery.ExclusiveLease, error) {
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
	return &softwareLease{owner: e}, nil
}

func (e *softwareExclusive) close() {
	e.mu.Lock()
	e.closed = true
	e.mu.Unlock()
}

func (e *softwareExclusive) holders() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.held
}

func (e *softwareExclusive) rejections() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rejected
}

type softwareLease struct {
	owner *softwareExclusive
	once  sync.Once
}

func (l *softwareLease) Check(ctx context.Context) error {
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

func (l *softwareLease) Close() error {
	l.once.Do(func() {
		l.owner.mu.Lock()
		l.owner.held--
		l.owner.mu.Unlock()
	})
	return nil
}

// softwareLifecycle is a controlled software media adapter for the ordinary
// lifecycle join. This is a controlled software media adapter: it does not prove
// real POSIX file journal / CAS / ACL / approved backup / E2E.
type softwareLifecycle struct {
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
func (l *softwareLifecycle) ReconcileAndStart(ctx context.Context, permit recovery.PhasePermit) error {
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

func (l *softwareLifecycle) StopAndJoin(ctx context.Context) error {
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

func (l *softwareLifecycle) counts() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reconcile, l.stop
}

func (l *softwareLifecycle) joinState() (children, joined int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.children, l.joined
}

// softwareCommitments is a controlled software media adapter for the installation
// commitment file. This is a controlled software media adapter: it does not prove
// real POSIX file journal / CAS / ACL / approved backup / E2E.
type softwareCommitments struct {
	mu           sync.Mutex
	deploymentID string
	commitment   recovery.InstallationCommitment
	revision     string
	calls        int
}

func (c *softwareCommitments) LoadInstallation(ctx context.Context, deploymentID string) (recovery.InstallationCommitment, string, error) {
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

func (c *softwareCommitments) CompareAndSwapInstallation(ctx context.Context, deploymentID, expected string, next recovery.InstallationCommitment) (recovery.InstallationCommitment, string, error) {
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

// softwareSourceVerifier is a controlled software media adapter for the approved
// source file. This is a controlled software media adapter: it does not prove
// real POSIX file journal / CAS / ACL / approved backup / E2E.
type softwareSourceVerifier struct {
	expected string
	calls    atomic.Int64
}

func (v *softwareSourceVerifier) VerifyInstallationSource(ctx context.Context, manifest recovery.InstallationManifest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v.calls.Add(1)
	if manifest.SourceApprovalDigest != v.expected {
		return recovery.ErrAdmissionBinding
	}
	return nil
}

// s417DeletionRecordPayload / s417DeletionDiskRecord mirror the production
// canonical deletion record shape only so this controlled software media
// adapter emits the same media-level hash chain the real replay source reads.
// It never decides deletion, replay, publication or admission semantics.
type s417DeletionRecordPayload struct {
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

type s417DeletionDiskRecord struct {
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

func s417DeletionRecordHash(record article.ArticleDeletionTombstone) string {
	raw, _ := json.Marshal(s417DeletionDiskRecord{
		Namespace: record.Namespace, Sequence: record.Sequence, ArticleID: record.ArticleID,
		ArticleVersion: record.ArticleVersion, DeletionOperationID: record.DeletionOperationID,
		DeletedAt: record.DeletedAt.UTC().Format(time.RFC3339Nano), KeyID: record.KeyID,
		KeyVersion: record.KeyVersion, Stage: record.Stage, PreviousRecordHash: record.PreviousRecordHash,
		RecordHMAC: record.RecordHMAC,
	})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// softwareDeletionJournal is a controlled software media adapter for the article
// deletion journal. This is a controlled software media adapter: it does not
// prove real POSIX file journal / CAS / ACL / approved backup / E2E.
type softwareDeletionJournal struct {
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

func s417NewSoftwareDeletionJournal(now func() time.Time) *softwareDeletionJournal {
	return &softwareDeletionJournal{
		journalID: "s417-article-deletion-journal", namespace: article.ArticleDeletionNamespace, epoch: 1,
		key: []byte("s417-controlled-software-deletion-media-key"), keyID: "s417-key", keyVersion: "v1",
		now: now, prefixes: []recovery.JournalHead{{Sequence: 0, Hash: ""}},
	}
}

func (j *softwareDeletionJournal) Append(ctx context.Context, request article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
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
	deletedAt := s417Canonical(request.DeletedAt)
	payload := s417DeletionRecordPayload{
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
		RecordHMAC:          s417HMAC(j.key, rawPayload),
	}
	chain := s417DeletionRecordHash(tombstone)
	j.byOperation[request.DeletionOperationID] = tombstone
	j.order = append(j.order, tombstone)
	j.prefixes = append(j.prefixes, recovery.JournalHead{Sequence: uint64(sequence), Hash: chain})
	j.revision++
	return tombstone, nil
}

func (j *softwareDeletionJournal) revisionLocked() string {
	return fmt.Sprintf("%s/rev-%d", j.journalID, j.revision)
}

// Load exposes the committed head of THIS journal instance as the runtime
// article anchor. The finalizer and the runtime share this single authority.
func (j *softwareDeletionJournal) Load(ctx context.Context, namespace string) (recovery.AnchorSnapshot, string, error) {
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
func (j *softwareDeletionJournal) CompareAndSwap(ctx context.Context, namespace, expectedRevision string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
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
func (j *softwareDeletionJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
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

func (j *softwareDeletionJournal) FindByOperation(ctx context.Context, operationID string) (article.ArticleDeletionTombstone, error) {
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

func (j *softwareDeletionJournal) ReadAllVerified(ctx context.Context) ([]article.ArticleDeletionTombstone, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]article.ArticleDeletionTombstone(nil), j.order...), nil
}

// s417World is one genesis installation wired through the real production
// factory plus the controlled software media adapters.
type s417World struct {
	f                *s417Fixture
	store            *identity.RecoveryRunStore
	journal          *softwareJournal
	articleSnapshots *softwareJournal
	exclusive        *softwareExclusive
	lifecycle        *softwareLifecycle
	commitments      *softwareCommitments
	sources          *softwareSourceVerifier
	closures         *article.PostgresReplayEvidenceStore
	installation     recovery.InstallationBinding
	manifest         recovery.InstallationManifest
	runtime          recovery.Runtime
	now              time.Time
}

func s417NewWorld(t *testing.T) *s417World {
	t.Helper()
	fixture := s417NewFixture(t)
	now := fixture.now
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s417-deployment",
		InstallationID:  "s417-installation",
		TargetDigest:    s417Digest("s417-target"),
		CommitmentRef:   "recovery/v1/installation/s417-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s417-recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: "s417-article-journal", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s417-approved-source",
		SourceApprovalDigest: s417Digest("s417-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s417ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s417NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	articleSnapshots := s417NewSoftwareJournal(installation.ArticleInitial.JournalID, recovery.ArticleDeletionNamespaceV1, installation.ArticleInitial.Epoch, clock)
	exclusive := &softwareExclusive{}
	lifecycle := &softwareLifecycle{}
	commitments := &softwareCommitments{
		deploymentID: installation.DeploymentID,
		commitment: recovery.InstallationCommitment{
			Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
		},
		revision: "s417-commitment-1",
	}
	sources := &softwareSourceVerifier{expected: manifest.SourceApprovalDigest}
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
	return &s417World{
		f: fixture, store: store, journal: journal, articleSnapshots: articleSnapshots,
		exclusive: exclusive, lifecycle: lifecycle, commitments: commitments, sources: sources,
		closures: closures, installation: installation, manifest: manifest, runtime: runtime, now: now,
	}
}

// s417RunNormal drives the real runtime to public readiness.
func (w *s417World) s417RunNormal(t *testing.T) {
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

var _ recovery.ContextRecoveryJournal = (*softwareJournal)(nil)
var _ recovery.VerifiedJournalReader = (*softwareJournal)(nil)
var _ recovery.DeploymentExclusive = (*softwareExclusive)(nil)
var _ recovery.OrdinaryLifecycle = (*softwareLifecycle)(nil)
var _ recovery.InstallationCommitmentStore = (*softwareCommitments)(nil)
var _ recovery.InstallationSourceVerifier = (*softwareSourceVerifier)(nil)
var _ article.ArticleDeletionJournal = (*softwareDeletionJournal)(nil)
var _ article.VerifiedDeletionJournal = (*softwareDeletionJournal)(nil)
var _ recovery.VerifiedJournalReader = (*softwareDeletionJournal)(nil)
var _ = errors.Is
var _ = pgx.ErrNoRows

// ---------------------------------------------------------------------------
// S4.17 leaf-1 support: real installation executor, real article media anchor,
// and the real article.DeletionFinalizer wired to real PostgreSQL.
// ---------------------------------------------------------------------------

func s417NewUUID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic("s417 uuid entropy unavailable")
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buffer[0:4], buffer[4:6], buffer[6:8], buffer[8:10], buffer[10:16])
}

type s417ClockAdapter struct{ now func() time.Time }

func (c s417ClockAdapter) Now() time.Time { return c.now() }

type s417IDAdapter struct{}

func (s417IDAdapter) NewID() (string, error) { return s417NewUUID(), nil }

// s417CompletionVerifier is never reached by this leaf: the seeded author stays
// active, so the real identity lifecycle returns Deleted=false before any
// completion release is requested. It refuses instead of fabricating proof.
type s417CompletionVerifier struct{}

func (s417CompletionVerifier) VerifyCompletion(context.Context, pgx.Tx, identity.ArticleCompletionReference) (identity.VerifiedArticleCompletion, error) {
	return identity.VerifiedArticleCompletion{}, errors.New("s417: completion verification is not reachable for an active author")
}

// s417IdentityPort maps the real identity lifecycle onto the article-owned
// disposition port. Mapping only; every decision stays inside internal/identity.
type s417IdentityPort struct {
	lifecycle *identity.ArticleIdentityLifecycle
}

func (p s417IdentityPort) LockTransferParties(ctx context.Context, tx pgx.Tx, targetID *string, sourceID string) (article.DispositionParties, error) {
	parties, err := p.lifecycle.LockTransferParties(ctx, tx, targetID, sourceID)
	if err != nil {
		return article.DispositionParties{}, err
	}
	return article.DispositionParties{
		SourceID: parties.SourceID, SourceEmail: parties.SourceEmail,
		TargetID: parties.TargetID, TargetEmail: parties.TargetEmail,
	}, nil
}

func (p s417IdentityPort) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, candidateID string) (article.DispositionDeletionCandidate, error) {
	candidate, err := p.lifecycle.LockDeletionCandidate(ctx, tx, candidateID)
	if err != nil {
		return article.DispositionDeletionCandidate{}, err
	}
	return article.DispositionDeletionCandidate{UserID: candidate.UserID, Deleted: candidate.Deleted}, nil
}

func (p s417IdentityPort) ClearRetainedEmailIfEligible(context.Context, pgx.Tx, string, *article.DispositionCompletionProof) (bool, error) {
	return false, errors.New("s417: identity release is not reachable for an active author")
}

// softwareInstallationWriter is a controlled software media adapter for the
// installation binding file. Controlled software media only: no POSIX/CAS/ACL
// claim; it records the delegated write and never decides installation state.
type softwareInstallationWriter struct{ binds atomic.Int64 }

func (w *softwareInstallationWriter) BindInstallation(ctx context.Context, _ recovery.PhasePermit, _ recovery.InstallationBinding) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.binds.Add(1)
	return nil
}

func (w *softwareInstallationWriter) BindLegacyPublication(context.Context, recovery.PhasePermit, string, recovery.PublicationBinding) error {
	return nil
}

// s417InventoryManifest writes one canonical read-only backup inventory
// manifest to a temp file and returns its absolute path. This is a controlled
// file-media adapter (adapter_scoped=inventory_file): the manifest bytes are
// test-owned, but the real article.FileBackupInventoryReader re-verifies them.
func s417InventoryManifest(t *testing.T, backupRef string, cutoff, generatedAt time.Time) string {
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
		BackupCutoff:     s417Canonical(cutoff).Format(layout),
		GeneratedAt:      s417Canonical(generatedAt).Format(layout),
		RecoverableBackups: []entry{{
			BackupRef: backupRef, BackupCompletedAt: s417Canonical(cutoff).Format(layout),
		}},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode inventory digest payload failed: %v", err)
	}
	manifest := string(encoded[:len(encoded)-1]) + `,"inventory_digest":` + s417JSON(recovery.SHA256Digest(encoded)) + `}`
	path := filepath.Join(t.TempDir(), "s417-inventory.json")
	if err := os.WriteFile(path, []byte(manifest), 0o444); err != nil {
		t.Fatalf("write inventory manifest failed: %v", err)
	}
	return path
}

// s417ReplayChain is the REAL deletion-replay/publication chain: the real
// PostgresReplayEvidenceStore, the real production replay source (journal +
// file inventory + real finalizer replayer + real PG fact reader), the sealed
// real publication ports and the real DeletionReplayProvider gate.
type s417ReplayChain struct {
	store    *article.PostgresReplayEvidenceStore
	source   *article.ProductionDeletionReplaySource
	ports    *article.PostgresReplayEvidenceStore
	provider *article.DeletionReplayProvider
}

func s417NewReplayChain(
	t *testing.T, f *s417Fixture, journal *softwareDeletionJournal,
	finalizer *article.DeletionFinalizer, rt recovery.Runtime, store *article.PostgresReplayEvidenceStore,
	installationID, backupRef string,
) *s417ReplayChain {
	t.Helper()
	if backupRef == "" {
		backupRef = "s417-backup-" + s417NewUUID()[:8]
	}
	clock := s417ClockAdapter{now: func() time.Time { return f.now }}
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
		s417InventoryManifest(t, backupRef, f.now, f.now),
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
		PublicationIDs: s417IDAdapter{}, Clock: clock,
	})
	if err != nil {
		t.Fatalf("real replay publication ports failed: %v", err)
	}
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{
		Store: &s417LoggedReplayStore{t: t, inner: ports}, Source: &s417LoggedReplaySource{t: t, inner: source},
		EvidenceIDs: s417IDAdapter{}, Tokens: s417IDAdapter{}, Clock: clock,
	})
	if err != nil {
		t.Fatalf("real deletion replay provider failed: %v", err)
	}
	return &s417ReplayChain{store: store, source: source, ports: ports, provider: provider}
}

// s417LoggedReplayStore and s417LoggedReplaySource surface the real provider's
// intermediate errors so the integration test can report the exact cause
// instead of only the public "article evidence unavailable" projection.
type s417LoggedReplayStore struct {
	t     *testing.T
	inner article.ReplayEvidenceStore
}

func (l *s417LoggedReplayStore) LoadAttempt(ctx context.Context, runID, backupRef string) (article.AttemptSnapshot, error) {
	attempt, err := l.inner.LoadAttempt(ctx, runID, backupRef)
	if err != nil {
		l.t.Logf("real replay store LoadAttempt err=%v", err)
	}
	return attempt, err
}

func (l *s417LoggedReplayStore) Begin(ctx context.Context, runID, backupRef string, expected article.AttemptSnapshot, fresh article.DeletionReplayFacts, candidate string) (article.BeginResult, error) {
	result, err := l.inner.Begin(ctx, runID, backupRef, expected, fresh, candidate)
	if err != nil {
		l.t.Logf("real replay store Begin err=%v expected=%+v", err, expected)
	}
	return result, err
}

func (l *s417LoggedReplayStore) Publish(ctx context.Context, request article.ReplayPublishRequest) (article.DeletionReplayEvidence, error) {
	evidence, err := l.inner.Publish(ctx, request)
	if err != nil {
		l.t.Logf("real replay store Publish err=%v attempt=%+v", err, request.Attempt)
	}
	return evidence, err
}

func (l *s417LoggedReplayStore) Fail(ctx context.Context, runID, backupRef string, attempt article.AttemptSnapshot, reason string) error {
	err := l.inner.Fail(ctx, runID, backupRef, attempt, reason)
	if err != nil {
		l.t.Logf("real replay store Fail err=%v", err)
	}
	return err
}

func (l *s417LoggedReplayStore) ReadCurrent(ctx context.Context, runID, backupRef string, refresh func(context.Context) (article.DeletionReplayFacts, error)) (article.DeletionReplayRun, *article.DeletionReplayEvidence, error) {
	run, evidence, err := l.inner.ReadCurrent(ctx, runID, backupRef, refresh)
	if err != nil {
		l.t.Logf("real replay store ReadCurrent err=%v", err)
	}
	return run, evidence, err
}

type s417LoggedReplaySource struct {
	t     *testing.T
	inner article.DeletionReplaySource
}

func (l *s417LoggedReplaySource) ReplayOldBackup(ctx context.Context, backupRef string) (article.DeletionReplayInput, error) {
	input, err := l.inner.ReplayOldBackup(ctx, backupRef)
	if err != nil {
		l.t.Logf("real replay source ReplayOldBackup err=%v", err)
	}
	return input, err
}

func (l *s417LoggedReplaySource) Snapshot(ctx context.Context, backupRef string) (article.DeletionReplayFacts, error) {
	facts, err := l.inner.Snapshot(ctx, backupRef)
	if err != nil {
		l.t.Logf("real replay source Snapshot err=%v", err)
	}
	return facts, err
}

// s417Transactor is the real PostgreSQL transaction boundary (pgxpool).
type s417Transactor struct{ pool *pgxpool.Pool }

func (t s417Transactor) Begin(ctx context.Context) (pgx.Tx, error) { return t.pool.Begin(ctx) }

// s417FullWorld is a genesis installation wired through the real production
// runtime factory plus the real PostgreSQL deletion finalizer.
type s417FullWorld struct {
	f               *s417Fixture
	store           *identity.RecoveryRunStore
	journal         *softwareJournal
	commitments     *softwareCommitments
	sources         *softwareSourceVerifier
	closures        *article.PostgresReplayEvidenceStore
	deletionJournal *softwareDeletionJournal
	installation    recovery.InstallationBinding
	manifest        recovery.InstallationManifest
	exclusive       *softwareExclusive
	lifecycle       *softwareLifecycle
	writer          *softwareInstallationWriter
	runtime         recovery.Runtime
	finalizer       *article.DeletionFinalizer
	chain           *s417ReplayChain
	coordinator     *recovery.Coordinator
	now             time.Time
	coordinateProbe func(context.Context, recovery.PhasePermit) error
	replayBackupRef string
}

func s417NewFullWorld(t *testing.T) *s417FullWorld {
	t.Helper()
	fixture := s417NewFixture(t)
	now := fixture.now
	deletionJournal := s417NewSoftwareDeletionJournal(func() time.Time { return fixture.now })
	deletionJournal.journalID = "s417-article-journal"
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s417-deployment",
		InstallationID:  "s417-installation",
		TargetDigest:    s417Digest("s417-target"),
		CommitmentRef:   "recovery/v1/installation/s417-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s417-recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: deletionJournal.journalID, Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s417-approved-source",
		SourceApprovalDigest: s417Digest("s417-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s417ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s417NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	commitments := &softwareCommitments{
		deploymentID: installation.DeploymentID,
		commitment: recovery.InstallationCommitment{
			Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
		},
		revision: "s417-commitment-1",
	}
	closureStore, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	world := &s417FullWorld{
		f: fixture, store: store, journal: journal,
		commitments: commitments, sources: &softwareSourceVerifier{expected: manifest.SourceApprovalDigest},
		closures: closureStore, deletionJournal: deletionJournal,
		installation: installation, manifest: manifest, now: now,
	}
	world.s417WireFinalizer(t)
	world.s417WireRuntime(t)
	world.s417WireReplay(t)
	return world
}

// s417WireRuntime builds one fresh production runtime generation. Each runtime
// instance may perform exactly one Install/RunNormal/Enter/Resume action, so a
// historical restart always gets a new instance over the same durable media.
func (w *s417FullWorld) s417WireRuntime(t *testing.T) {
	t.Helper()
	w.exclusive = &softwareExclusive{}
	w.lifecycle = &softwareLifecycle{}
	w.writer = &softwareInstallationWriter{}
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

// s417WireReplay wires the REAL replay/publication chain and the REAL
// recovery.Coordinator over the same store, journal and article journal
// authority. The coordinator is constructed (and validated) for real; the
// runtime-driven consumed window remains unproven because RunNormal never
// invokes Consumer and Coordinator.Consume requires a live replay phase permit.
func (w *s417FullWorld) s417WireReplay(t *testing.T) *s417ReplayChain {
	t.Helper()
	if w.chain != nil {
		return w.chain
	}
	w.chain = s417NewReplayChain(t, w.f, w.deletionJournal, w.finalizer, w.runtime, w.closures, w.installation.InstallationID, w.replayBackupRef)
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

func (w *s417FullWorld) s417WireFinalizer(t *testing.T) {
	t.Helper()
	lifecycle, err := identity.NewArticleIdentityLifecycle(
		s417CompletionVerifier{}, audit.Store{}, s417IDAdapter{}, s417ClockAdapter{now: func() time.Time { return w.f.now }},
	)
	if err != nil {
		t.Fatalf("real identity article lifecycle failed: %v", err)
	}
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions:  s417Transactor{pool: w.f.pool},
		Repository:    article.Repository{},
		Journal:       w.deletionJournal,
		Auditor:       audit.Store{},
		AuditIDs:      s417IDAdapter{},
		Clock:         s417ClockAdapter{now: func() time.Time { return w.f.now }},
		WorkerID:      "s417-finalizer-worker",
		LeaseDuration: time.Minute,
		Identity:      s417IdentityPort{lifecycle: lifecycle},
	})
	if err != nil {
		t.Fatalf("real deletion finalizer failed: %v", err)
	}
	w.finalizer = finalizer
}

// s417RunNormal drives the real runtime to public readiness.
func (w *s417FullWorld) s417RunNormal(t *testing.T) {
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

// s417Install drives the real production installation algorithm (bound path).
func (w *s417FullWorld) s417Install(t *testing.T) {
	t.Helper()
	if err := w.runtime.Install(w.f.ctx, w.manifest); err != nil {
		t.Fatalf("real runtime Install failed: %v", err)
	}
}

// s417SeedPendingDeletion inserts a real user, a real journaled deletion
// operation and one real deletion_pending article row.
func (w *s417FullWorld) s417SeedPendingDeletion(t *testing.T) (userID, articleID, operationID string) {
	t.Helper()
	userID, articleID, operationID = s417NewUUID(), s417NewUUID(), s417NewUUID()
	now := w.f.now
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s417-synthetic-password-digest','member','active','permanent',false,0,0,$5,$5)`,
		userID, "s417-"+userID[:8], "s417-"+userID[:8]+"@example.test", "+1555"+s417Digits(userID), now); err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	for _, identifier := range [][2]string{
		{"username", "s417-" + userID[:8]},
		{"email", "s417-" + userID[:8] + "@example.test"},
		{"phone", "+1555" + s417Digits(userID)},
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

func s417Digits(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	digits := make([]byte, 7)
	for index := range digits {
		digits[index] = '0' + sum[index]%10
	}
	return string(digits)
}

var _ recovery.JournalHeadAnchor = (*softwareDeletionJournal)(nil)
var _ recovery.AnchorReader = (*softwareDeletionJournal)(nil)
var _ recovery.InstallationRepository = (*softwareInstallationWriter)(nil)
var _ article.DispositionIdentityPort = s417IdentityPort{}

// s417BeginTransactor adapts the real pool to the article/identity Begin seam.
type s417BeginTransactor struct{ pool *pgxpool.Pool }

func (t s417BeginTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	return t.pool.Begin(ctx)
}

// s417PasswordHasher is a boundary adapter for the unused password digest in
// this leaf. adapter_scoped=password_hasher; it never decides admission.
type s417PasswordHasher struct{}

func (s417PasswordHasher) Hash(password string) (string, error) {
	return "s417-synthetic-digest:" + s417Digest(password), nil
}

// s417CanonicalWiring holds the REAL identity canonical-delete stack and the
// REAL identity recovery execution over the fixture that already owns the real
// recovery runtime, journal, store and replay chain.
type s417CanonicalWiring struct {
	coordinator *identity.CanonicalDeleteCoordinator
	port        identity.ArticleAccountPort
	exec        recovery.IdentityRecoveryExecutor
	maintenance identity.RecoveryIdentityMaintenance
	adminID     string
	memberID    string
	articleID   string
}

func (w *s417FullWorld) s417WireCanonical(t *testing.T) *s417CanonicalWiring {
	t.Helper()
	clock := s417ClockAdapter{now: func() time.Time { return w.f.now }}
	commands, err := article.NewCommandService(article.CommandDependencies{
		Transactions: s417BeginTransactor{pool: w.f.pool},
		Repository:   article.Repository{},
		IDs:          s417IDAdapter{},
		Clock:        clock,
		Orphaning: &article.OrphanArticlesCanonicalDependencies{
			Repository: article.CanonicalOrphanRepository{},
			Auditor:    audit.Store{},
			AuditIDs:   s417IDAdapter{},
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
		Transactions: s417BeginTransactor{pool: w.f.pool},
		Repository:   identity.PostgresCanonicalDeleteRepository{},
		Articles:     port,
		Audit:        audit.Store{},
		IDs:          s417IDAdapter{},
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
		Passwords: s417PasswordHasher{},
		IDs:       s417IDAdapter{},
		Clock:     clock,
	})
	if err != nil {
		t.Fatalf("real identity recovery execution failed: %v", err)
	}
	wiring := &s417CanonicalWiring{coordinator: coordinator, port: port, exec: exec, maintenance: maintenance}
	wiring.seed(t, w)
	return wiring
}

func (c *s417CanonicalWiring) seed(t *testing.T, w *s417FullWorld) {
	t.Helper()
	c.adminID, c.memberID, c.articleID = s417NewUUID(), s417NewUUID(), s417NewUUID()
	now := w.f.now
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, role, is_initial_admin, status, credential_mode, recovery_required, created_at, updated_at)
		VALUES($1,$2,$3,'admin',true,'active','permanent',false,$4,$4)`,
		c.adminID, "s417-admin-"+c.adminID[:8], "s417-admin-"+c.adminID[:8]+"@example.test", now); err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}
	username, email, phone := "s417-member-"+c.memberID[:8], "s417-member-"+c.memberID[:8]+"@example.test", "+1555"+s417Digits(c.memberID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s417-synthetic-password-digest','member','active','permanent',false,0,0,$5,$5)`,
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

func (c *s417CanonicalWiring) adminPrincipal() session.Principal {
	return session.Principal{UserID: c.adminID, Role: "admin", Status: "active", IsInitialAdmin: true}
}

// s417PublicationSeed identifies one REAL PostgreSQL evidence/publication pair
// written by the test with the same canonical encodings the production store
// verifies. Nothing here decides freshness: PostgresReplayEvidenceStore does.
type s417PublicationSeed struct {
	runID, backupRef, evidenceRef, evidenceDigest string
	publicationRef, publicationDigest             string
}

// s417PublicationCanonical mirrors the production canonical publication record
// field order and names so the test can compute the same publication digest.
type s417PublicationCanonical struct {
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

// s417SeedPublishedClosure writes a real deletion_replay_runs + evidence +
// article_replay_publications row set bound to the given journal identity and
// head. The real PostgresReplayEvidenceStore later decides fresh/stale/rollback.
func s417SeedPublishedClosure(t *testing.T, w *s417FullWorld, journalID string, epoch uint64, head recovery.JournalHead) s417PublicationSeed {
	t.Helper()
	if head.Sequence == 0 || head.Hash == "" {
		t.Fatalf("seeded publication head must be a committed non-genesis head: %#v", head)
	}
	seed := s417PublicationSeed{
		runID:          "s417-run-" + s417NewUUID()[:8],
		backupRef:      "s417-backup-" + s417NewUUID()[:8],
		evidenceRef:    "s417-evidence-" + s417NewUUID()[:8],
		publicationRef: s417NewUUID(),
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
		InventoryDigest:          s417Digest("s417-inventory"),
		DeletionOperationsDigest: s417Digest("s417-operations"),
		AuditEventsDigest:        s417Digest("s417-audit"),
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
	canonical := s417PublicationCanonical{
		Version: "article-replay-publication/v1", Ref: seed.publicationRef, SourceKind: "atomic_current_v1",
		RunID: seed.runID, BackupRef: seed.backupRef, Namespace: article.ArticleDeletionNamespace,
		EvidenceRef: seed.evidenceRef, EvidenceDigest: seed.evidenceDigest,
		EvidenceCompletedAt: completedAt.Format(layout), JournalID: journalID, Epoch: epoch,
		Sequence: head.Sequence, Hash: head.Hash, Generation: 0,
		SourceRef: seed.publicationRef, SourceDigest: s417Digest("s417-source-placeholder"),
		PublishedAt: publishedAt.Format(layout),
	}
	canonical.SourceDigest = recovery.SHA256Digest([]byte(s417JSON([]any{
		seed.runID, seed.backupRef, article.ArticleDeletionNamespace, int64(0), seed.evidenceRef, seed.evidenceDigest,
	})))
	seed.publicationDigest = recovery.SHA256Digest([]byte(s417JSON(canonical)))
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

func (s s417PublicationSeed) request() recovery.ClosureRequest {
	return recovery.ClosureRequest{
		RunID: s.runID, BackupRef: s.backupRef, EvidenceRef: s.evidenceRef, EvidenceDigest: s.evidenceDigest,
		PublicationRef: s.publicationRef, PublicationDigest: s.publicationDigest,
	}
}

// ---------------------------------------------------------------------------
// S4.17 leaf-5 support: the real HTTP session/authorizer/recovery-guard chain
// over the real Article query and permanent-deletion handlers.
// ---------------------------------------------------------------------------

// s417Case18Counters are observer-only counters. They never decide business
// state; they only let the test prove that a closed stream never reached the
// Article handler, service, repository or identity author reader.
type s417Case18Counters struct {
	handlerEntries  atomic.Int64
	traceContexts   atomic.Int64
	serviceCalls    atomic.Int64
	repositoryCalls atomic.Int64
	authorReadCalls atomic.Int64
}

// s417CountingQueryRepository is an observer adapter over the real PostgreSQL
// query repository. It does not alter reads or visibility decisions.
type s417CountingQueryRepository struct {
	next     article.ArticleQueryRepository
	counters *s417Case18Counters
}

func (r s417CountingQueryRepository) ListCandidateBatch(
	ctx context.Context, cursor *article.ArticleCursor, limit int,
) ([]article.Article, error) {
	r.counters.repositoryCalls.Add(1)
	return r.next.ListCandidateBatch(ctx, cursor, limit)
}

func (r s417CountingQueryRepository) GetArticle(ctx context.Context, id string) (article.Article, error) {
	r.counters.repositoryCalls.Add(1)
	return r.next.GetArticle(ctx, id)
}

// s417CountingAuthorReader is an observer adapter over the real identity-owned
// author state reader. It does not alter author states.
type s417CountingAuthorReader struct {
	next     article.ArticleAuthorStateReader[identity.UserStatus]
	counters *s417Case18Counters
}

func (r s417CountingAuthorReader) GetAuthorStates(
	ctx context.Context, ids []string,
) (map[string]identity.UserStatus, error) {
	r.counters.authorReadCalls.Add(1)
	return r.next.GetAuthorStates(ctx, ids)
}

// s417CountingQuerier is an observer adapter over the real Article query
// service. It only counts invocations.
type s417CountingQuerier struct {
	next     article.ArticleQuerier
	counters *s417Case18Counters
}

func (q s417CountingQuerier) ListArticles(
	ctx context.Context, request article.ListArticlesRequest,
) (article.ListArticlesPage, error) {
	q.counters.serviceCalls.Add(1)
	return q.next.ListArticles(ctx, request)
}

func (q s417CountingQuerier) GetArticle(
	ctx context.Context, id string,
) (article.MemberArticleDetailView, string, error) {
	q.counters.serviceCalls.Add(1)
	return q.next.GetArticle(ctx, id)
}

// s417OriginObserver is a bounded observer for same-origin rejections. It is a
// controlled external-boundary adapter and does not decide request admission.
type s417OriginObserver struct {
	rejections atomic.Int64
	lastReason atomic.Value
}

func (o *s417OriginObserver) ObserveRejected(
	_ context.Context, observation platform.OriginRejectionObservation,
) error {
	o.rejections.Add(1)
	o.lastReason.Store(string(observation.Reason))
	return nil
}

// s417Case18World wires one isolated PostgreSQL schema through the real
// production runtime and the real HTTP session, authorizer, recovery guard,
// Article query handler and permanent-deletion handler. Only the exact-route
// transport skeleton and the observer adapters above are test-owned glue.
type s417Case18World struct {
	f              *s417Fixture
	full           *s417FullWorld
	counters       *s417Case18Counters
	originObserver *s417OriginObserver
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

func s417NewCase18World(t *testing.T) *s417Case18World {
	t.Helper()
	full := s417NewFullWorld(t)
	fixture := full.f
	world := &s417Case18World{
		f: fixture, full: full,
		counters:       &s417Case18Counters{},
		originObserver: &s417OriginObserver{},
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
	world.sessionToken = "s417-session-" + s417NewUUID()
	world.csrfToken = "s417-csrf-" + s417NewUUID()
	if _, err := store.Create(fixture.ctx, fixture.pool, session.CreateInput{
		ID:                          s417NewUUID(),
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
		s417CountingQueryRepository{next: repository, counters: world.counters},
		s417CountingAuthorReader{next: authorReader, counters: world.counters},
		article.QueryConfig{DefaultLimit: 20, MaxLimit: 50, CandidateBatchSize: 51},
	)
	if err != nil {
		t.Fatalf("real article query service failed: %v", err)
	}
	queryHandler, err := article.NewQueryHandler(s417CountingQuerier{next: queryService, counters: world.counters})
	if err != nil {
		t.Fatalf("real article query handler failed: %v", err)
	}

	coordinator, err := article.NewPermanentDeletionCoordinator(article.PermanentDeletionDependencies{
		Transactions: s417Transactor{pool: fixture.pool},
		Repository:   article.Repository{},
		Journal:      full.deletionJournal,
		Finalizer:    full.finalizer,
		OperationIDs: s417IDAdapter{},
		Clock:        s417ClockAdapter{now: func() time.Time { return fixture.now }},
	})
	if err != nil {
		t.Fatalf("real permanent deletion coordinator failed: %v", err)
	}
	commands, err := article.NewCommandService(article.CommandDependencies{
		Transactions: s417Transactor{pool: fixture.pool},
		Repository:   article.Repository{},
		IDs:          s417IDAdapter{},
		Clock:        s417ClockAdapter{now: func() time.Time { return fixture.now }},
		Deletion:     coordinator,
	})
	if err != nil {
		t.Fatalf("real article command service failed: %v", err)
	}
	writeLogger, err := article.NewSlogArticleResponseWriteLogger(
		slog.New(slog.NewJSONHandler(io.Discard, nil)), "s417",
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
	world.router = platform.TraceMiddleware(s417NewCase18Transport(world))
	return world
}

// s417Case18Transport is a transport-only adapter over the standard library
// ServeMux method patterns. The router contract (404 for unregistered paths,
// 405 + Allow for a registered path with an unsupported method) is produced by
// the real ServeMux; only the mux-generated text/plain 405 body is rewritten
// into the platform error envelope. It never decides business state.
type s417Case18Transport struct{ next http.Handler }

// s417Case18BufferedResponse records one handler response so the transport can
// distinguish the mux default response from a handler-written response.
type s417Case18BufferedResponse struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func (b *s417Case18BufferedResponse) Header() http.Header {
	if b.header == nil {
		b.header = http.Header{}
	}
	return b.header
}

func (b *s417Case18BufferedResponse) WriteHeader(code int) {
	if b.code == 0 {
		b.code = code
	}
}

func (b *s417Case18BufferedResponse) Write(payload []byte) (int, error) {
	if b.code == 0 {
		b.code = http.StatusOK
	}
	return b.body.Write(payload)
}

func s417NewCase18Transport(world *s417Case18World) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/articles/{id}", world.detailHandler)
	mux.Handle("DELETE /api/v1/articles/{id}", world.deleteHandler)
	mux.Handle("PUT /api/v1/articles/{id}", s417OutOfScopeRoute("article update"))
	mux.Handle("GET /api/v1/articles", world.listHandler)
	mux.Handle("POST /api/v1/articles", s417OutOfScopeRoute("article create"))
	return s417Case18Transport{next: mux}
}

// s417OutOfScopeRoute registers a route that exists in the production router so
// the standard mux computes the exact Allow set, but whose body contract belongs
// to another card. It is never asserted as production behaviour.
func s417OutOfScopeRoute(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Content-Type-Options", "nosniff")
		http.Error(response, name+" route is out of scope for S4.17", http.StatusNotImplemented)
	})
}

func (t s417Case18Transport) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	buffered := &s417Case18BufferedResponse{}
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

func (w *s417Case18World) seed(t *testing.T) {
	t.Helper()
	w.userID, w.articleID = s417NewUUID(), s417NewUUID()
	now := w.f.now
	username := "s417-case18-" + w.userID[:8]
	email := username + "@example.test"
	phone := "+1555" + s417Digits(w.userID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s417-synthetic-password-digest','member','active','permanent',false,0,0,$5,$5)`,
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

func (w *s417Case18World) articlePath() string {
	return "/api/v1/articles/" + w.articleID
}

func (w *s417Case18World) counts() (handler, trace, service, repository, authorRead int64) {
	return w.counters.handlerEntries.Load(),
		w.counters.traceContexts.Load(),
		w.counters.serviceCalls.Load(),
		w.counters.repositoryCalls.Load(),
		w.counters.authorReadCalls.Load()
}

// s417SoftwareRestore is a controlled software tool adapter for the restore
// driver boundary. This is a controlled software tool adapter; it does not
// prove real artifact/restore/E2E. It only records the approved plan and
// returns the byte-identical receipt the real runtime validates.
type s417SoftwareRestore struct {
	installation recovery.InstallationBinding
	now          time.Time
	mu           sync.Mutex
	plan         recovery.RestorePlan
	receipt      recovery.RestoreReceipt
	prepared     int
	replaced     int
	verified     int
}

func (r *s417SoftwareRestore) Prepare(ctx context.Context, b recovery.RunBinding) (recovery.RestorePlan, error) {
	if err := ctx.Err(); err != nil {
		return recovery.RestorePlan{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plan = recovery.RestorePlan{
		Version: "han-db-restore/v1", RunID: b.RunID, BackupRef: b.BackupRef,
		DeploymentID: r.installation.DeploymentID, InstallationID: r.installation.InstallationID,
		TargetDigest:         r.installation.TargetDigest,
		ArtifactDigest:       s417Digest("s417-controlled-artifact"),
		SourceApprovalRef:    "s417-controlled-restore-approval",
		SourceApprovalDigest: s417Digest("s417-controlled-restore-approval"),
		SourceInstallationID: "s417-controlled-source-installation",
		SourceSchemaDigest:   s417Digest("s417-controlled-source-schema"),
		ArtifactPath:         "s417/controlled/restore.dump", ArtifactBytes: 4096,
	}
	r.prepared++
	return r.plan, nil
}

func (r *s417SoftwareRestore) Replace(ctx context.Context, permit recovery.PhasePermit, plan recovery.RestorePlan) (recovery.RestoreReceipt, error) {
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
		ReceiptDigest:        s417Digest(plan.RunID + "|s417-restore-receipt"),
		ArtifactBytes:        plan.ArtifactBytes, RestoredAt: s417Canonical(r.now),
	}
	r.replaced++
	return r.receipt, nil
}

func (r *s417SoftwareRestore) VerifyRestored(ctx context.Context, plan recovery.RestorePlan) (recovery.RestoreReceipt, error) {
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

func (r *s417SoftwareRestore) counts() (prepared, replaced, verified int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.prepared, r.replaced, r.verified
}

// s417RecoverySecret serves the candidate secret from memory only.
type s417RecoverySecret struct{ secret []byte }

func (s *s417RecoverySecret) ReadInitialRecoverySecret(ctx context.Context, b recovery.RunBinding) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.secret...), nil
}

// s417LateConsumer binds the real recovery.Coordinator after the runtime (and
// therefore the publication ports) exists; it never decides anything itself.
type s417LateConsumer struct {
	mu     sync.RWMutex
	target recovery.RecoveryConsumer
}

func (c *s417LateConsumer) Consume(ctx context.Context, runID string) (recovery.ConsumptionResult, error) {
	c.mu.RLock()
	target := c.target
	c.mu.RUnlock()
	if target == nil {
		return recovery.ConsumptionResult{}, recovery.ErrAdmissionBinding
	}
	return target.Consume(ctx, runID)
}

func (c *s417LateConsumer) set(target recovery.RecoveryConsumer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = target
}

// s417LateExit binds the real identity executor (RecoveryExit) after the
// runtime exists; it forwards only.
type s417LateExit struct {
	mu     sync.RWMutex
	target recovery.RecoveryExit
}

func (e *s417LateExit) ExitRecovery(ctx context.Context, permit recovery.PhasePermit, b recovery.RunBinding) error {
	e.mu.RLock()
	target := e.target
	e.mu.RUnlock()
	if target == nil {
		return recovery.ErrAdmissionBinding
	}
	return target.ExitRecovery(ctx, permit, b)
}

func (e *s417LateExit) set(target recovery.RecoveryExit) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.target = target
}

// s417RecoveryExit exposes the real executor's RevokeRecoverySessions as the
// runtime RecoveryExit port. It forwards only; it makes no decision.
type s417RecoveryExit struct {
	exec recovery.IdentityRecoveryExecutor
}

func (e s417RecoveryExit) ExitRecovery(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return e.exec.RevokeRecoverySessions(ctx, p, b)
}

// s417IdentityHook forwards the real identity executor and optionally injects
// recovery-maintenance work while the real PhaseRecoveryIdentity permit lives.
type s417IdentityHook struct {
	t     *testing.T
	inner recovery.IdentityRecoveryExecutor
	hook  func(context.Context, recovery.PhasePermit, recovery.RunBinding) error
}

func (h *s417IdentityHook) stage(name string, call func() error) error {
	err := call()
	if h.t != nil {
		h.t.Logf("identity stage %s err=%v", name, err)
	}
	return err
}

func (h *s417IdentityHook) BindRestoredRun(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, r recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	var out recovery.RecoveryRun
	err := h.stage("BindRestoredRun", func() error {
		var inner error
		out, inner = h.inner.BindRestoredRun(ctx, p, b, r)
		return inner
	})
	return out, err
}

func (h *s417IdentityHook) FinalizeRestoredIdentity(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return h.stage("FinalizeRestoredIdentity", func() error { return h.inner.FinalizeRestoredIdentity(ctx, p, b) })
}

func (h *s417IdentityHook) PrepareRecoveryTrust(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, secret []byte) error {
	return h.stage("PrepareRecoveryTrust", func() error { return h.inner.PrepareRecoveryTrust(ctx, p, b, secret) })
}

func (h *s417IdentityHook) RevokeRecoverySessions(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return h.stage("RevokeRecoverySessions", func() error { return h.inner.RevokeRecoverySessions(ctx, p, b) })
}

func (h *s417IdentityHook) CompleteIdentityReview(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	return h.stage("CompleteIdentityReview", func() error {
		if h.hook != nil {
			if err := h.hook(ctx, p, b); err != nil {
				return err
			}
		}
		return h.inner.CompleteIdentityReview(ctx, p, b)
	})
}

func s417Argon2Passwords(t *testing.T) *security.Argon2idHasher {
	t.Helper()
	hasher, err := security.NewArgon2idHasher(security.Argon2idConfig{
		MemoryKiB: 8, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	}, rand.Reader)
	if err != nil {
		t.Fatalf("real argon2id hasher failed: %v", err)
	}
	return hasher
}

// s417RecoveryWorld wires the REAL recovery runtime Enter path: real
// identity.NewRecoveryRunStore, real identity.NewRecoveryIdentityExecution,
// real recovery.NewCoordinator, real article replay/publication ports and the
// real deletion finalizer over PostgreSQL. Only the journal bytes, restore
// tool, lifecycle join and exclusive lease are controlled software media
// adapters, and they never decide admission.
type s417RecoveryWorld struct {
	f                *s417Fixture
	store            *identity.RecoveryRunStore
	journal          *softwareJournal
	deletionJournal  *softwareDeletionJournal
	commitments      *softwareCommitments
	sources          *softwareSourceVerifier
	writer           *softwareInstallationWriter
	closures         *article.PostgresReplayEvidenceStore
	installation     recovery.InstallationBinding
	manifest         recovery.InstallationManifest
	exclusive        *softwareExclusive
	lifecycle        *softwareLifecycle
	restore          *s417SoftwareRestore
	runtime          recovery.Runtime
	port             identity.ArticleAccountPort
	exec             recovery.IdentityRecoveryExecutor
	maintenance      identity.RecoveryIdentityMaintenance
	coordinator      *recovery.Coordinator
	chain            *s417ReplayChain
	finalizer        *article.DeletionFinalizer
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
}

func s417NewRecoveryWorld(t *testing.T, withMember bool, hook func(context.Context, recovery.PhasePermit, recovery.RunBinding) error, worldHook func(*s417RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error) *s417RecoveryWorld {
	t.Helper()
	fixture := s417NewFixture(t)
	now := fixture.now
	deletionJournal := s417NewSoftwareDeletionJournal(func() time.Time { return fixture.now })
	deletionJournal.journalID = "s417-recovery-article-journal"
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s417-recovery-deployment",
		InstallationID:  "s417-recovery-installation",
		TargetDigest:    s417Digest("s417-recovery-target"),
		CommitmentRef:   "recovery/v1/installation/s417-recovery-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s417-recovery-run-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: deletionJournal.journalID, Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s417-recovery-approved-source",
		SourceApprovalDigest: s417Digest("s417-recovery-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s417ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s417NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	closures, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	world := &s417RecoveryWorld{
		f: fixture, store: store, journal: journal, deletionJournal: deletionJournal,
		commitments: &softwareCommitments{
			deploymentID: installation.DeploymentID,
			commitment: recovery.InstallationCommitment{
				Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
			},
			revision: "s417-recovery-commitment-1",
		},
		sources:      &softwareSourceVerifier{expected: manifest.SourceApprovalDigest},
		closures:     closures,
		installation: installation, manifest: manifest, now: now,
		secret:    []byte("s417-recovery-secret"),
		backupRef: "s417-recovery-backup-" + s417NewUUID()[:8],
	}
	world.s417WireFinalizer(t)
	if worldHook != nil {
		hook = worldHook(world)
	}
	world.s417WireRuntime(t, hook)
	world.s417Seed(t, withMember)
	return world
}

func (w *s417RecoveryWorld) s417WireFinalizer(t *testing.T) {
	t.Helper()
	lifecycle, err := identity.NewArticleIdentityLifecycle(
		s417CompletionVerifier{}, audit.Store{}, s417IDAdapter{}, s417ClockAdapter{now: func() time.Time { return w.f.now }},
	)
	if err != nil {
		t.Fatalf("real identity article lifecycle failed: %v", err)
	}
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions:  s417Transactor{pool: w.f.pool},
		Repository:    article.Repository{},
		Journal:       w.deletionJournal,
		Auditor:       audit.Store{},
		AuditIDs:      s417IDAdapter{},
		Clock:         s417ClockAdapter{now: func() time.Time { return w.f.now }},
		WorkerID:      "s417-recovery-finalizer-worker",
		LeaseDuration: time.Minute,
		Identity:      s417IdentityPort{lifecycle: lifecycle},
	})
	if err != nil {
		t.Fatalf("real deletion finalizer failed: %v", err)
	}
	w.finalizer = finalizer
}

func (w *s417RecoveryWorld) s417WireRuntime(t *testing.T, hook func(context.Context, recovery.PhasePermit, recovery.RunBinding) error) {
	t.Helper()
	clock := func() time.Time { return w.f.now }
	w.exclusive = &softwareExclusive{}
	w.lifecycle = &softwareLifecycle{}
	w.writer = &softwareInstallationWriter{}
	w.restore = &s417SoftwareRestore{installation: w.installation, now: w.f.now}
	forwarder := identity.NewRecoveryIdentityForwarder()
	lateConsumer := &s417LateConsumer{}
	lateExit := &s417LateExit{}
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
		Identity:            &s417IdentityHook{t: t, inner: forwarder, hook: hook},
		Restore:             w.restore,
		Consumer:            lateConsumer,
		Exit:                lateExit,
		RecoverySecret:      &s417RecoverySecret{secret: w.secret},
	})
	if err != nil {
		t.Fatalf("real recovery runtime factory failed: %v", err)
	}
	runtime = created
	w.runtime = created
	commands, err := article.NewCommandService(article.CommandDependencies{
		Transactions: s417BeginTransactor{pool: w.f.pool},
		Repository:   article.Repository{},
		IDs:          s417IDAdapter{},
		Clock:        s417ClockAdapter{now: clock},
		Orphaning: &article.OrphanArticlesCanonicalDependencies{
			Repository: article.CanonicalOrphanRepository{},
			Auditor:    audit.Store{},
			AuditIDs:   s417IDAdapter{},
			Clock:      s417ClockAdapter{now: clock},
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
	w.passwords = s417Argon2Passwords(t)
	exec, maintenance, err := identity.NewRecoveryIdentityExecution(identity.RecoveryIdentityExecutionDependencies{
		Store: w.store, Runtime: runtime, Journal: w.journal, Articles: w.port,
		Audit: audit.Store{}, Passwords: w.passwords, IDs: s417IDAdapter{},
		Clock: s417ClockAdapter{now: clock},
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
	lateExit.set(s417RecoveryExit{exec: exec})
	w.chain = s417NewReplayChain(t, w.f, w.deletionJournal, w.finalizer, runtime, w.closures, w.installation.InstallationID, w.backupRef)
	coordinator, err := recovery.NewCoordinator(
		w.store, w.journal, w.chain.provider, w.chain.ports,
		recovery.NewReplayStatusV1Decoder(nil), clock,
	)
	if err != nil {
		t.Fatalf("real recovery coordinator failed: %v", err)
	}
	w.coordinator = coordinator
	lateConsumer.set(coordinator)

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
	issuer, err := identity.NewRandomLoginSessionIssuer(rand.Reader, s417IDAdapter{}, s417ClockAdapter{now: clock}, 32, 32, 24*time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("real session issuer failed: %v", err)
	}
	w.passwordService, err = identity.NewPasswordChangeService(identity.PasswordChangeDependencies{
		RecoveryAuthority: authority, Transactions: w.f.pool, Repository: identity.PostgresPasswordChangeRepository{},
		PasswordResetGate: w.gate, Passwords: w.passwords, Policy: security.PasswordPolicy{},
		Sessions:      identity.PostgresPasswordChangeSessions{Store: sessionStore},
		SessionIssuer: issuer, Audit: audit.Store{}, IDs: s417IDAdapter{}, Clock: s417ClockAdapter{now: clock},
	})
	if err != nil {
		t.Fatalf("real password change service failed: %v", err)
	}
}

func (w *s417RecoveryWorld) s417Seed(t *testing.T, withMember bool) {
	t.Helper()
	w.userID = s417NewUUID()
	now := w.f.now
	username := "s417-recovery-admin-" + w.userID[:8]
	email := username + "@example.test"
	phone := "+1555" + s417Digits(w.userID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, is_initial_admin, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s417-initial-admin-digest','admin','active','permanent',false,0,0,true,$5,$5)`,
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
	w.memberID = s417NewUUID()
	memberName := "s417-recovery-member-" + w.memberID[:8]
	memberEmail := memberName + "@example.test"
	memberPhone := "+1555" + s417Digits(w.memberID)
	if _, err := w.f.pool.Exec(w.f.ctx, `INSERT INTO users(
		id, username, email, phone, permanent_password_digest, role, status, credential_mode,
		recovery_required, auth_generation, permanent_password_generation, created_at, updated_at)
		VALUES($1,$2,$3,$4,'s417-member-digest','member','active','permanent',false,0,0,$5,$5)`,
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

// s417SeedFinalizedDeletion writes one real journaled deletion and finalizes
// it through the real article.DeletionFinalizer over PostgreSQL, so the real
// replay/publication chain has one coherent tombstone to replay.
func (w *s417RecoveryWorld) s417SeedFinalizedDeletion(t *testing.T) (userID, articleID, operationID string) {
	t.Helper()
	userID, articleID, operationID = w.userID, s417NewUUID(), s417NewUUID()
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

// s417BootstrapIdentity performs the real maintenance work that production
// operators must complete inside the live PhaseRecoveryIdentity window: one
// bootstrap maintenance session, the real ChangeRecoveryPassword transition and,
// optionally, the real canonical kept-deleted review through the real
// recoveryExecutionGate.
func (w *s417RecoveryWorld) s417BootstrapIdentity(ctx context.Context, p recovery.PhasePermit, b recovery.RunBinding, reviewMember bool) error {
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
	sessionID := s417NewUUID()
	token := "s417-maintenance-" + s417NewUUID()
	csrf := "s417-maintenance-csrf-" + s417NewUUID()
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
	if err := w.maintenance.ResolveRestrictedAccountReview(ctx, p, b, w.memberID, "kept_deleted", "s417-recovery-review-evidence"); err != nil {
		return fmt.Errorf("real canonical kept-deleted review: %w", err)
	}
	return nil
}

//go:build integration

package recovery_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestArticleReplayPostgres exercises the bounded r22/r20 recovery matrix.
// Only the public Journal seam is controlled; physical completion, PG facts,
// evidence publication and strict status decoding use their real implementations.
const (
	s44Trace                   = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	s44Admin                   = "11000000-0000-4000-8000-000000000001"
	s44DeletedSource           = "55000000-0000-4000-8000-000000000001"
	s44Backup                  = "counter-backup"
	s44Run                     = "synthetic-old-api-replay-run"
	s44PublicationInstallPhase = "install"
	s44Inventory               = `{"inventory_version":"article-backup-inventory/v1","inventory_scope":"recoverable-backups-before-cutoff","backup_ref":"counter-backup","backup_cutoff":"2026-10-20T00:00:00.000000Z","generated_at":"2026-10-20T00:00:01.000000Z","recoverable_backups":[{"backup_ref":"counter-backup","backup_completed_at":"2026-10-20T00:00:00.000000Z"}],"inventory_digest":"sha256:ad4e519d431d58fb6a872a13fcc0ae0dcd80562d688f94a6928d726bb6bcbda2"}`
)

var s44ChildSpecPath = flag.String("s44-replay-child-spec", "", "explicit retained replay crash child input")

func TestArticleReplayPostgres(t *testing.T) {
	if *s44ChildSpecPath != "" {
		s44CrashChild(t, *s44ChildSpecPath)
		return
	}
	t.Run("CountsAndReadback", func(t *testing.T) {
		t.Run("AllAbsent0N", s44CompletedAbsent)
		t.Run("AllPendingN0", func(t *testing.T) {
			t.Run("ExistingTrace", func(t *testing.T) { s44AllPending(t, true) })
			t.Run("GeneratedTrace", func(t *testing.T) { s44AllPending(t, false) })
		})
		t.Run("Mixed2And1", s44Mixed)
		t.Run("RepeatAndRebuildZeroReplay", s44Repeat)
	})
	t.Run("InterruptedReplayAndPublish", func(t *testing.T) {
		t.Run("PartialFailureRetry", s44Partial)
		t.Run("CrashAfterReplayBeforeReady", s44Crash)
		t.Run("PublishCommitAckLoss", s44AckLoss)
	})
	t.Run("AttemptAndFreshFacts", func(t *testing.T) {
		t.Run("SameAttemptWinnerLateAbsent", s44Winner)
		t.Run("OperationAuditDriftRepublish", func(t *testing.T) {
			t.Run("Operation", func(t *testing.T) { s44Drift(t, true) })
			t.Run("Audit", func(t *testing.T) { s44Drift(t, false) })
		})
		t.Run("RefreshOutsideRunAndRecheck", func(t *testing.T) {
			t.Run("Publish", s44RefreshPublish)
			t.Run("ReadCurrent", s44RefreshRead)
		})
	})
	t.Run("HistoryAndLegacy", func(t *testing.T) {
		t.Run("FactsAToBToA", s44History)
		t.Run("LegacyGen0Mixed", s44Legacy)
		t.Run("AmbiguousHistoryRejected", s44Ambiguous)
	})
}

type s44MarkerKey struct{}
type s44Clock struct{ at time.Time }

func (clock *s44Clock) Now() time.Time { return clock.at }

type s44IDs struct {
	remaining []string
	calls     int
}

func (ids *s44IDs) NewID() (string, error) {
	ids.calls++
	if len(ids.remaining) == 0 {
		return "", errors.New("synthetic replay ID sequence exhausted")
	}
	value := ids.remaining[0]
	ids.remaining = ids.remaining[1:]
	return value, nil
}

type s44Fixture struct {
	t                             *testing.T
	ctx                           context.Context
	cancel                        context.CancelFunc
	dir, schema                   string
	created                       bool
	pool                          *pgxpool.Pool
	observer                      *pgxpool.Conn
	observerPID                   int32
	transactionPIDs               []int32
	clock                         *s44Clock
	targets                       []s44Target
	journal                       *s44Journal
	finalizer                     *article.DeletionFinalizer
	probe                         *s44ReplayProbe
	source                        *article.ProductionDeletionReplaySource
	store                         *article.PostgresReplayEvidenceStore
	provider                      *article.DeletionReplayProvider
	auditIDs, evidenceIDs, tokens *s44IDs
	proofCalls                    int
	storeProbe                    *s44StoreProbe
	publication                   *s44PublicationJournal
	evidenceClock                 *s44EvidenceClock
	observationIndex              int
}

type s44Target struct {
	row              article.Article
	operation, audit string
	kind             article.DeleteKind
}

func s44Must(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		t.Fatalf("synthetic replay dependency failed type=%T sqlstate=%s", err, pg.Code)
	}
	t.Fatalf("synthetic replay dependency failed type=%T", err)
}

func s44Suffix(t *testing.T) string {
	t.Helper()
	var random [12]byte
	_, err := rand.Read(random[:])
	s44Must(t, err)
	return fmt.Sprintf("%x", random[:])
}

func s44Address(t *testing.T, host string, port uint16) {
	t.Helper()
	address := net.ParseIP(host)
	if address == nil || !address.IsLoopback() || port != 5433 {
		t.Fatal("replay fixture requires literal loopback:5433, including every fallback")
	}
}

func s44Config(t *testing.T, schema string) *pgxpool.Config {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("selected replay top requires PostgreSQL; no Skip")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid replay PostgreSQL configuration")
	}
	s44Address(t, config.ConnConfig.Host, config.ConnConfig.Port)
	for _, fallback := range config.ConnConfig.Fallbacks {
		s44Address(t, fallback.Host, fallback.Port)
	}
	if config.ConnConfig.User == "" {
		t.Fatal("replay PostgreSQL identity missing")
	}
	config.MaxConns = 4
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	return config
}

func s44New(t *testing.T) *s44Fixture {
	t.Helper()
	f := &s44Fixture{t: t, schema: "han_s44red_" + s44Suffix(t)}
	f.ctx, f.cancel = context.WithTimeout(context.WithValue(context.Background(), s44MarkerKey{}, "original-parent"), 90*time.Second)
	t.Cleanup(f.close)
	var err error
	f.dir, err = os.MkdirTemp("", "han-s44-retained-")
	s44Must(t, err)
	t.Logf("retained_dir=%s case=%s", f.dir, t.Name())
	config := s44Config(t, f.schema)
	f.pool, err = pgxpool.NewWithConfig(f.ctx, config)
	s44Must(t, err)
	f.preflight(config.ConnConfig.User)
	_, err = f.pool.Exec(f.ctx, "CREATE SCHEMA "+pgx.Identifier{f.schema}.Sanitize())
	s44Must(t, err)
	f.created = true
	t.Logf("retained_schema=%s stage=create_committed", f.schema)
	f.save("schema-created", map[string]any{"schema": f.schema, "case": t.Name()})
	f.migrate()
	f.observer, err = f.pool.Acquire(f.ctx)
	s44Must(t, err)
	s44Must(t, f.observer.QueryRow(f.ctx, "SELECT pg_backend_pid()").Scan(&f.observerPID))
	f.seed()
	f.wire()
	f.save("seed-before", f.businessImage())
	return f
}

func (f *s44Fixture) preflight(expectedUser string) {
	var host, user string
	var port, version int
	var now time.Time
	err := f.pool.QueryRow(f.ctx, `SELECT host(inet_server_addr()),inet_server_port(),
		current_setting('server_version_num')::int,current_user,clock_timestamp()`).Scan(&host, &port, &version, &user, &now)
	s44Must(f.t, err)
	s44Address(f.t, host, uint16(port))
	if version != 180006 || user == "" || user != expectedUser {
		f.t.Fatal("replay endpoint/version/identity preflight mismatch; no CREATE attempted")
	}
	f.clock = &s44Clock{now.UTC().Truncate(time.Microsecond)}
	f.t.Log("replay_preflight=passed endpoint=literal_loopback:5433 version=180006 identity_checked=true")
}

func (f *s44Fixture) migrate() {
	tx, err := f.pool.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql",
		"0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql",
		"0008_password_reset_durable.sql", "0009_article_replay_attempt.sql"}
	for _, name := range names {
		raw, readErr := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		s44Must(f.t, readErr)
		_, err = tx.Exec(f.ctx, string(raw))
		s44Must(f.t, err)
		f.t.Logf("schema=%s migration=%s stage=applied_in_tx", f.schema, name)
	}
	s44Must(f.t, tx.Commit(f.ctx))
	f.t.Logf("schema=%s stage=migrations_committed", f.schema)
	s44ApplyPublicationMigration(f.t, f)
}

// s44ApplyPublicationMigration is the single entry point for
// migrations/0011_article_replay_publication.sql. That migration has no
// IF NOT EXISTS guards, so the shared sequence and every targeted helper must
// pass this presence check instead of executing the DDL a second time.
func s44ApplyPublicationMigration(t *testing.T, f *s44Fixture) {
	t.Helper()
	var present bool
	s44Must(t, f.pool.QueryRow(f.ctx, `SELECT to_regclass($1) IS NOT NULL`,
		pgx.Identifier{f.schema, "article_replay_publications"}.Sanitize()).Scan(&present))
	if present {
		t.Logf("schema=%s migration=0011_article_replay_publication.sql stage=already_applied", f.schema)
		return
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "0011_article_replay_publication.sql"))
	s44Must(t, err)
	tx, err := f.pool.Begin(f.ctx)
	s44Must(t, err)
	defer s44Rollback(t, tx)
	_, err = tx.Exec(f.ctx, string(raw))
	s44Must(t, err)
	s44Must(t, tx.Commit(f.ctx))
	t.Logf("schema=%s migration=0011_article_replay_publication.sql stage=applied_in_tx", f.schema)
}

func s44Rollback(t *testing.T, tx pgx.Tx) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("replay rollback failed type=%T", err)
	}
}

func (f *s44Fixture) seed() {
	tx, err := f.pool.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	started := f.clock.at.Add(-time.Minute)
	_, err = tx.Exec(f.ctx, `INSERT INTO users(id,username,role,is_initial_admin,status,credential_mode,created_at,updated_at)
		VALUES($1,'synthetic_replay_admin','admin',true,'active','bootstrap',$2,$2)`, s44Admin, started)
	s44Must(f.t, err)
	_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at)
		VALUES($1,'username','synthetic_replay_admin','active',$2)`, s44Admin, started)
	s44Must(f.t, err)
	_, err = tx.Exec(f.ctx, `INSERT INTO users(id,username,role,status,credential_mode,created_at,updated_at,deleted_at)
		VALUES($1,'synthetic_deleted_source','member','deleted','recovery_locked',$2,$2,$2)`, s44DeletedSource, started)
	s44Must(f.t, err)
	_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at)
		VALUES($1,'username','synthetic_deleted_source','reserved',$2)`, s44DeletedSource, started)
	s44Must(f.t, err)
	f.targets = s44Targets(started)
	f.journal = &s44Journal{}
	for index, target := range f.targets {
		f.seedTarget(tx, target, int64(index+1), started)
	}
	s44Must(f.t, tx.Commit(f.ctx))
}

func s44Targets(started time.Time) []s44Target {
	var targets []s44Target
	for index, kind := range []article.DeleteKind{article.DeleteOrdinary, article.DeleteViolating, article.DeleteOrphan} {
		suffix := fmt.Sprintf("%012d", index+1)
		operation := "33000000-0000-4000-8000-" + suffix
		row := article.Article{ID: "22000000-0000-4000-8000-" + suffix, Title: "synthetic replay 🙂", Body: "synthetic body",
			PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear,
			DeletionState: article.DeletionPending, DeletionOperationID: &operation,
			Version: 2, CreatedAt: started.Add(-time.Minute), UpdatedAt: started}
		if kind == article.DeleteViolating {
			category := article.ViolationSexual
			row.ViolationState, row.ViolationCategory = article.ViolationViolating, &category
		}
		if kind == article.DeleteOrphan {
			row.PublicationState, row.OrphanedFromUserID = article.PublicationHeld, s44String(s44DeletedSource)
		}
		targets = append(targets, s44Target{row: row, operation: operation, audit: "44000000-0000-4000-8000-" + suffix, kind: kind})
	}
	return targets
}

func (f *s44Fixture) seedTarget(tx pgx.Tx, target s44Target, sequence int64, started time.Time) {
	_, err := tx.Exec(f.ctx, `INSERT INTO deletion_operations(id,article_id,article_version,delete_kind,
		actor_user_id,actor_role,orphaned_from_user_id,status,tombstone_sequence,journal_key_id,journal_key_version,started_at)
		VALUES($1,$2,1,$3,$4,'admin',$5,'journaled',$6,'synthetic-key','test-v1',$7)`,
		target.operation, target.row.ID, target.kind, s44Admin, target.row.OrphanedFromUserID, sequence, started)
	s44Must(f.t, err)
	s44Must(f.t, article.ValidateArticleState(target.row))
	s44Must(f.t, (article.Repository{}).InsertArticle(f.ctx, tx, target.row))
	f.journal.records = append(f.journal.records, article.ArticleDeletionTombstone{Namespace: article.ArticleDeletionNamespace,
		Sequence: sequence, ArticleID: target.row.ID, ArticleVersion: 1, DeletionOperationID: target.operation,
		DeletedAt: started, KeyID: "synthetic-key", KeyVersion: "test-v1", Stage: "tombstoned"})
}

func s44String(value string) *string { return &value }

func (f *s44Fixture) wire() { f.wireExisting(true) }

func (f *s44Fixture) wireExisting(installInventory bool) {
	f.auditIDs = &s44IDs{}
	for _, target := range f.targets {
		f.auditIDs.remaining = append(f.auditIDs.remaining, target.audit)
	}
	f.evidenceIDs = &s44IDs{remaining: []string{"synthetic-evidence-one"}}
	f.tokens = &s44IDs{remaining: []string{"synthetic-candidate-one"}}
	var err error
	f.finalizer, err = article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{
		Transactions: &s44Transactions{f: f}, Repository: article.Repository{}, Journal: f.journal,
		Auditor: audit.Store{}, AuditIDs: f.auditIDs, Clock: f.clock,
		WorkerID: "synthetic-replay-worker", LeaseDuration: time.Minute, Identity: &s44Identity{f: f}})
	s44Must(f.t, err)
	path := filepath.Join(f.dir, "synthetic-inventory.json")
	if installInventory {
		s44Must(f.t, os.WriteFile(path, []byte(s44Inventory), 0600))
	}
	reader, err := article.NewFileBackupInventoryReader(path)
	s44Must(f.t, err)
	manifest, err := reader.Read(f.ctx)
	s44Must(f.t, err)
	if manifest.BackupRef != s44Backup || manifest.InventoryDigest != "sha256:ad4e519d431d58fb6a872a13fcc0ae0dcd80562d688f94a6928d726bb6bcbda2" {
		f.t.Fatal("independent fixed inventory golden mismatch")
	}
	f.probe = &s44ReplayProbe{real: f.finalizer}
	s44WireExistingReplay(f.t, f, path)
	f.provider, err = article.NewDeletionReplayProvider(article.DeletionReplayDependencies{
		Store: f.store, Source: f.source, EvidenceIDs: f.evidenceIDs, Tokens: f.tokens, Clock: f.clock})
	s44Must(f.t, err)
}

// These labels describe the synthetic Journal snapshot and runtime permit
// binding the production publication ports require. They are not a real
// installation, POSIX Journal or recovery runtime.
const (
	s44PublicationInstallation = "synthetic-replay-harness-installation"
	s44PublicationJournalID    = "synthetic-replay-harness-journal"
	s44PublicationRevision     = "synthetic-replay-harness-revision-0001"
)

// s44WireExistingReplay builds the real Source and Store over the synthetic
// fixture Journal and seals the production publication ports exactly once per
// Store. Initial wiring and every s44Rebuild go through this one path.
func s44WireExistingReplay(t *testing.T, f *s44Fixture, inventoryPath string) {
	t.Helper()
	journal := &s44PublicationJournal{s44Journal: f.journal}
	f.publication = journal
	inventory, err := article.NewFileBackupInventoryReader(inventoryPath)
	s44Must(t, err)
	facts, err := article.NewPostgresReplayFactReader(f.pool)
	s44Must(t, err)
	f.source, err = article.NewProductionDeletionReplaySource(journal, inventory, f.probe, facts, facts, f.clock.Now)
	s44Must(t, err)
	store, err := article.NewPostgresReplayEvidenceStore(f.pool)
	s44Must(t, err)
	f.store = s44ConfigureReplayPublication(t, f, f.source, store, &s44PublicationRuntime{})
}

// s44ConfigureReplayPublication seals one synthetic publication configuration
// onto a freshly constructed Store; NewReplayPublicationPorts seals a Store
// only once, so callers must always pass a new Store.
func s44ConfigureReplayPublication(t *testing.T, f *s44Fixture, source *article.ProductionDeletionReplaySource,
	store *article.PostgresReplayEvidenceStore, runtime *s44PublicationRuntime) *article.PostgresReplayEvidenceStore {
	t.Helper()
	publicationIDs := &s44IDs{}
	for index := 1; index <= 16; index++ {
		publicationIDs.remaining = append(publicationIDs.remaining, fmt.Sprintf("77000000-0000-4000-8000-%012d", index))
	}
	configured, err := article.NewReplayPublicationPorts(store, article.ReplayPublicationDependencies{
		Source: source, Runtime: runtime, InstallationID: s44PublicationInstallation,
		PublicationIDs: publicationIDs, Clock: f.clock})
	s44Must(t, err)
	return configured
}

// s44PublicationJournal projects the synthetic fixture Journal records through
// the verified snapshot contract used by the publication ports. It is not a
// POSIX Journal and proves no fsync/flock behavior.
type s44PublicationJournal struct {
	*s44Journal
	// snapshotHook gates one chosen verified snapshot read; production Publish
	// ignores callbacks carried on the request, so a concurrency test must hold
	// it here instead of through ReplayPublishRequest.Refresh.
	snapshotHook func(context.Context) error
}

func (journal *s44PublicationJournal) ReadVerifiedSnapshot(
	ctx context.Context, namespace string,
) (recovery.VerifiedJournalSnapshot, error) {
	if journal == nil || journal.s44Journal == nil || ctx == nil {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return recovery.VerifiedJournalSnapshot{}, err
	}
	if journal.snapshotHook != nil {
		if err := journal.snapshotHook(ctx); err != nil {
			return recovery.VerifiedJournalSnapshot{}, err
		}
	}
	if namespace != article.ArticleDeletionNamespace {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	prefixes := []recovery.JournalHead{{Sequence: 0, Hash: ""}}
	for index, record := range journal.records {
		if record.Sequence != int64(index+1) || record.Namespace != namespace {
			return recovery.VerifiedJournalSnapshot{}, article.ErrDeletionJournalCorrupt
		}
		prefixes = append(prefixes, recovery.JournalHead{
			Sequence: uint64(record.Sequence), Hash: s44PublicationRecordHash(record)})
	}
	return recovery.VerifiedJournalSnapshot{JournalID: s44PublicationJournalID, Namespace: namespace, Epoch: 1,
		Head: prefixes[len(prefixes)-1], Revision: s44PublicationRevision, Prefixes: prefixes}, nil
}

// s44PublicationRecordHash mirrors the production on-disk record projection
// over synthetic fixture records; it is not a real Journal record hash.
func s44PublicationRecordHash(record article.ArticleDeletionTombstone) string {
	raw, _ := json.Marshal(struct {
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
	}{record.Namespace, record.Sequence, record.ArticleID, record.ArticleVersion, record.DeletionOperationID,
		record.DeletedAt.UTC().Format(time.RFC3339Nano), record.KeyID, record.KeyVersion, record.Stage,
		record.PreviousRecordHash, record.RecordHMAC})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// s44PublicationRuntime supplies only the observed permit binding that the
// publication port checks. It is not a real recovery runtime.
type s44PublicationRuntime struct {
	recovery.Runtime
	backupRef string
}

func (host *s44PublicationRuntime) ReadPermitBinding(ctx context.Context) (recovery.PermitBinding, error) {
	if ctx == nil {
		return recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return recovery.PermitBinding{}, err
	}
	backupRef := host.backupRef
	if backupRef == "" {
		backupRef = s44Backup
	}
	return recovery.PermitBinding{RuntimeID: "synthetic-replay-harness-runtime", Generation: 1,
		RunID: s44Run, BackupRef: backupRef, InstallationID: s44PublicationInstallation,
		Purpose: recovery.PhaseRecoveryReplay}, nil
}

// CheckPhase mirrors the production purpose check for the synthetic harness.
// The install purpose is the unexported production constant
// replayPublicationInstall ("install") in internal/article.
func (host *s44PublicationRuntime) CheckPhase(
	ctx context.Context,
	_ recovery.PhasePermit,
	purpose string,
) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch purpose {
	case recovery.PhaseRecoveryReplay, s44PublicationInstallPhase:
		return nil
	default:
		return recovery.ErrAdmissionBinding
	}
}

type s44Transactions struct{ f *s44Fixture }

func (transactions *s44Transactions) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := transactions.f.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	var pid int32
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		s44Rollback(transactions.f.t, tx)
		return nil, err
	}
	transactions.f.transactionPIDs = append(transactions.f.transactionPIDs, pid)
	return tx, nil
}

type s44Journal struct {
	records       []article.ArticleDeletionTombstone
	reads, finds  int
	failOperation string
}

func (journal *s44Journal) ReadAllVerified(context.Context) ([]article.ArticleDeletionTombstone, error) {
	journal.reads++
	return append([]article.ArticleDeletionTombstone(nil), journal.records...), nil
}

func (journal *s44Journal) FindByOperation(_ context.Context, id string) (article.ArticleDeletionTombstone, error) {
	journal.finds++
	if id == journal.failOperation {
		journal.failOperation = ""
		return article.ArticleDeletionTombstone{}, errors.New("controlled journal lookup interruption")
	}
	for _, record := range journal.records {
		if id == record.DeletionOperationID {
			return record, nil
		}
	}
	return article.ArticleDeletionTombstone{}, errors.New("synthetic Journal binding mismatch")
}

func (journal *s44Journal) Append(context.Context, article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
	return article.ArticleDeletionTombstone{}, errors.New("unexpected Append for pre-journaled synthetic operation")
}

type s44ReplayProbe struct {
	real                *article.DeletionFinalizer
	calls               int
	result              article.DeletionReplayAllResult
	cause               error
	trace, contextTrace string
	parentMarker        any
	deadline            time.Time
	hasDeadline         bool
	done                <-chan struct{}
}

func (probe *s44ReplayProbe) ReplayAllTombstones(ctx context.Context, records []article.ArticleDeletionTombstone, trace string) (article.DeletionReplayAllResult, error) {
	probe.calls++
	probe.trace = trace
	probe.contextTrace, _ = platform.TraceIDFromContext(ctx)
	probe.parentMarker = ctx.Value(s44MarkerKey{})
	probe.deadline, probe.hasDeadline = ctx.Deadline()
	probe.done = ctx.Done()
	probe.result, probe.cause = probe.real.ReplayAllTombstones(ctx, records, trace)
	return probe.result, probe.cause
}

func (f *s44Fixture) save(name string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	s44Must(f.t, err)
	f.saveRaw(name+".json", data)
}

func (f *s44Fixture) saveRaw(name string, data []byte) {
	file, err := os.OpenFile(filepath.Join(f.dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	s44Must(f.t, err)
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	s44Must(f.t, writeErr)
	s44Must(f.t, closeErr)
}

func (f *s44Fixture) businessImage() map[string]json.RawMessage {
	image := make(map[string]json.RawMessage)
	for _, table := range []string{"articles", "deletion_operations", "audit_events", "users", "identity_identifiers"} {
		var raw []byte
		projection := "to_jsonb(t)"
		if table == "users" {
			projection += " - 'permanent_password_digest'"
		}
		query := "SELECT coalesce(jsonb_agg(entry_json ORDER BY entry_json::text),'[]'::jsonb) FROM (SELECT " + projection + " AS entry_json FROM " + pgx.Identifier{f.schema, table}.Sanitize() + " t) q"
		s44Must(f.t, f.observer.QueryRow(f.ctx, query).Scan(&raw))
		image[table] = raw
	}
	return image
}

func (f *s44Fixture) factObservation() map[string]any {
	current, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(f.t, err)
	return map[string]any{"journal_head_sequence": current.JournalHeadSequence, "journal_head_hash": current.JournalHeadHash,
		"inventory_scope": current.InventoryScope, "backup_cutoff": current.BackupCutoff, "inventory_digest": current.InventoryDigest,
		"deletion_operations_digest": current.DeletionOperationsDigest, "audit_events_digest": current.AuditEventsDigest,
		"tombstones_total": current.TombstonesTotal, "operations_reconciled": current.DeletionOperationsReconciledCount,
		"audits_verified": current.AuditEventsVerifiedCount}
}

func s44CompletedAbsent(t *testing.T) {
	f := s44New(t)
	for _, target := range f.targets {
		s44Must(t, f.finalizer.Finalize(f.ctx, target.operation, s44Trace))
	}
	f.assertCompleted(s44Trace)
	before := f.businessImage()
	facts := f.factObservation()
	f.save("precompleted-before-provider", before)
	raw, err := f.provider.ReplayAndVerify(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	f.capture(raw)
	if f.probe.calls != 1 || f.probe.cause != nil || f.probe.result.Replayed != 0 || f.probe.result.AlreadyAbsent != 3 {
		t.Error("real completed Finalizer did not return exactly 0/3")
	}
	if !reflect.DeepEqual(before, f.businessImage()) || !reflect.DeepEqual(facts, f.factObservation()) {
		t.Error("completed replay changed fixed business facts")
	}
	f.assertReady(raw, 0, 3)
}

func s44AllPending(t *testing.T, existingTrace bool) {
	f := s44New(t)
	ctx := f.ctx
	if existingTrace {
		var err error
		ctx, err = platform.ContextWithTraceID(ctx, s44Trace)
		s44Must(t, err)
	}
	raw, err := f.provider.ReplayAndVerify(ctx, s44Run, s44Backup)
	s44Must(t, err)
	f.capture(raw)
	if f.probe.calls != 1 || f.probe.parentMarker != "original-parent" {
		t.Error("production source lost the original caller context or ReplayAll invocation")
	}
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline != f.probe.hasDeadline || !deadline.Equal(f.probe.deadline) || f.probe.done != ctx.Done() {
		t.Error("production source lost the original caller deadline/cancellation channel")
	}
	if f.probe.cause != nil {
		t.Errorf("real all-pending completion failed type=%T trace_class=%s", f.probe.cause, s44TraceClass(f.probe.cause))
	}
	_, traceErr := platform.ContextWithTraceID(context.Background(), f.probe.trace)
	if traceErr != nil || f.probe.contextTrace != f.probe.trace || (existingTrace && f.probe.trace != s44Trace) {
		t.Error("production source did not forward/generate one legal consistent trace")
	}
	f.assertReady(raw, 3, 0)
}

func s44TraceClass(cause error) string {
	if cause == nil {
		return "none"
	}
	switch cause.Error() {
	case "invalid trace ID":
		return "invalid_trace_id"
	case "deletion trace context mismatch":
		return "context_trace_mismatch"
	default:
		return "other_dependency_failure"
	}
}

func (f *s44Fixture) capture(raw []byte) {
	var wire map[string]json.RawMessage
	s44Must(f.t, json.Unmarshal(raw, &wire))
	f.saveRaw("provider-wire.raw.json", raw)
	f.save("provider-wire", wire)
	f.save("after-provider-business", f.businessImage())
	var counts [3]int
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM deletion_replay_runs),
		(SELECT count(*) FROM deletion_replay_evidence),(SELECT count(*) FROM audit_events)`).Scan(&counts[0], &counts[1], &counts[2]))
	var status string
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT status FROM deletion_replay_runs WHERE run_id=$1 AND namespace=$2`, s44Run, article.ArticleDeletionNamespace).Scan(&status))
	f.save("observations", map[string]any{"schema": f.schema, "observer_pid": f.observerPID,
		"transaction_pids": f.transactionPIDs, "real_replay_calls": f.probe.calls,
		"real_result": f.probe.result, "trace_class": s44TraceClass(f.probe.cause), "incoming_trace": f.probe.trace,
		"context_trace": f.probe.contextTrace, "row_counts_run_evidence_audit": counts, "run_status": status,
		"audit_id_calls": f.auditIDs.calls, "evidence_id_calls": f.evidenceIDs.calls, "candidate_calls": f.tokens.calls,
		"journal_controlled": true, "real_pg_source_finalizer_store_provider": true})
	for _, pid := range f.transactionPIDs {
		if pid == f.observerPID {
			f.t.Error("observer and actual business transaction used the same backend")
		}
	}
	f.assertUnlocked()
}

func (f *s44Fixture) assertUnlocked() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := f.observer.Begin(ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	for _, target := range f.targets {
		var id string
		err = tx.QueryRow(ctx, `SELECT id FROM deletion_operations WHERE id=$1 FOR UPDATE NOWAIT`, target.operation).Scan(&id)
		s44Must(f.t, err)
		if id != target.operation {
			f.t.Error("independent NOWAIT operation binding mismatch")
		}
	}
	s44Must(f.t, tx.Rollback(ctx))
	f.t.Log("independent_backend_nowait=unlocked")
}

func (f *s44Fixture) assertReady(raw []byte, replayed, absent uint64) {
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	s44Must(f.t, err)
	if decoded.Ready == nil {
		if decoded.Unavailable == nil {
			f.t.Fatal("shared decoder returned no variant")
		}
		f.t.Errorf("real Source/PGStore/Provider must reach Ready; got status=%s reason=%s", decoded.Unavailable.Status, decoded.Unavailable.ReasonCode)
		return
	}
	ready := *decoded.Ready
	if ready.RunID != s44Run || ready.BackupRef != s44Backup || ready.Namespace != article.ArticleDeletionNamespace ||
		ready.TombstonesTotal != 3 || ready.TombstonesReplayed != replayed || ready.AlreadyAbsent != absent ||
		ready.DeletionOperationsReconciledCount != 3 || ready.AuditEventsVerifiedCount != 3 {
		f.t.Error("Ready binding/real execution counts differ from independent fixture facts")
	}
	f.assertReadyFields(raw)
	digest, err := ready.CanonicalDigest()
	s44Must(f.t, err)
	if digest != ready.EvidenceDigest {
		f.t.Error("real Ready does not match the sole canonical authority")
	}
	f.assertPersisted(ready)
	trace := f.probe.trace
	if replayed == 0 {
		trace = s44Trace
	}
	f.assertCompleted(trace)
	if f.probe.cause != nil || f.probe.result.Replayed != int64(replayed) || f.probe.result.AlreadyAbsent != int64(absent) {
		f.t.Error("real Finalizer result differs from the published first-execution outcome")
	}
	s44CheckReady(f, raw, replayed, absent)
}

func (f *s44Fixture) assertReadyFields(raw []byte) {
	var fields map[string]json.RawMessage
	s44Must(f.t, json.Unmarshal(raw, &fields))
	keys := []string{"evidence_version", "run_id", "backup_ref", "namespace", "journal_head_sequence", "journal_head_hash",
		"inventory_scope", "backup_cutoff", "inventory_digest", "deletion_operations_digest", "audit_events_digest",
		"tombstones_total", "tombstones_replayed", "already_absent", "deletion_operations_reconciled_count",
		"audit_events_verified_count", "status", "evidence_digest", "evidence_ref", "completed_at"}
	if len(fields) != len(keys) {
		f.t.Error("Ready must have exactly the independent20 wire keys")
	}
	for _, key := range keys {
		value, present := fields[key]
		if !present || bytes.Equal(value, []byte("null")) {
			f.t.Errorf("Ready field missing/null: %s", key)
		}
	}
}

func (f *s44Fixture) assertPersisted(ready recovery.ReadyReplayStatusV1) {
	var ref, input, status string
	var total, replayed, absent, operations, audits int64
	var completed time.Time
	err := f.observer.QueryRow(f.ctx, `SELECT r.current_evidence_ref,r.input_digest,r.status,
		e.tombstones_total,e.tombstones_replayed,e.already_absent,e.deletion_operations_reconciled_count,
		e.audit_events_verified_count,e.completed_at FROM deletion_replay_runs r JOIN deletion_replay_evidence e
		ON e.evidence_ref=r.current_evidence_ref AND e.run_id=r.run_id AND e.namespace=r.namespace
		WHERE r.run_id=$1 AND r.namespace=$2 AND r.backup_ref=$3`, s44Run, article.ArticleDeletionNamespace, s44Backup).
		Scan(&ref, &input, &status, &total, &replayed, &absent, &operations, &audits, &completed)
	s44Must(f.t, err)
	if ref != ready.EvidenceRef || input != ready.EvidenceDigest || status != "ready" || total != 3 ||
		replayed != int64(ready.TombstonesReplayed) || absent != int64(ready.AlreadyAbsent) || operations != 3 || audits != 3 ||
		completed.UTC().Format(time.RFC3339Nano) != ready.CompletedAt || completed.Nanosecond()%1000 != 0 {
		f.t.Error("independent committed current/evidence SQL oracle mismatch")
	}
	var count int
	s44Must(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM deletion_replay_evidence").Scan(&count))
	if count != 1 {
		f.t.Error("first Ready did not commit exactly one immutable evidence")
	}
}

func (f *s44Fixture) assertCompleted(trace string) {
	var articles, events int
	s44Must(f.t, f.observer.QueryRow(f.ctx, "SELECT (SELECT count(*) FROM articles),(SELECT count(*) FROM audit_events)").Scan(&articles, &events))
	if articles != 0 || events != 3 || f.auditIDs.calls != 3 || f.proofCalls != 1 {
		f.t.Fatal("real three-kind completion/finite audit IDs/public proof counts mismatch")
	}
	for _, target := range f.targets {
		f.assertCompletedTarget(target, trace)
	}
}

func (f *s44Fixture) assertCompletedTarget(target s44Target, trace string) {
	var status, auditID string
	var completed time.Time
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT status,audit_event_id,completed_at FROM deletion_operations WHERE id=$1`, target.operation).
		Scan(&status, &auditID, &completed))
	if status != "completed" || auditID != target.audit || !completed.Equal(f.clock.at) {
		f.t.Fatal("real Finalizer completion differs from the independent article/operation/audit oracle")
	}
	event, err := (audit.Store{}).Get(f.ctx, f.observer, target.audit)
	s44Must(f.t, err)
	event.OccurredAt = event.OccurredAt.UTC()
	metadata := map[string]any{"article_version": float64(1), "delete_kind": string(target.kind)}
	action := "article.delete"
	if target.kind == article.DeleteOrdinary {
		metadata["authority"] = "administrator"
	}
	if target.kind == article.DeleteViolating {
		action = "article.violation.delete"
	}
	if target.kind == article.DeleteOrphan {
		action = "article.orphan.delete"
	}
	want := audit.Event{ID: target.audit, OccurredAt: f.clock.at, ActorKind: audit.ActorUser, ActorUserID: s44String(s44Admin),
		ActorRole: s44String("admin"), Action: action, TargetType: "article", TargetID: target.row.ID,
		Outcome: "succeeded", TraceID: trace, Metadata: metadata}
	if !reflect.DeepEqual(event, want) {
		f.t.Fatal("real audit exact-value oracle or finite audit-ID consumption mismatch")
	}
}

func (f *s44Fixture) close() {
	if f.observer != nil {
		f.observer.Release()
	}
	if f.cancel != nil {
		f.cancel()
	}
	if f.probe != nil && f.probe.calls > 0 {
		select {
		case <-f.probe.done:
		default:
			f.t.Error("replay cancellation did not reach the original observed context")
		}
	}
	var acquired int32
	if f.pool != nil {
		acquired = f.pool.Stat().AcquiredConns()
		if acquired != 0 {
			f.t.Errorf("replay resources acquired=%d; expected0 before close", acquired)
		}
		f.pool.Close()
	}
	f.t.Logf("schema=%s retained=%t resources=observer_released_cancelled_pool_closed acquired=%d workers=joined_before_pool_close", f.schema, f.created, acquired)
}

var _ article.ArticleTransactor = (*s44Transactions)(nil)
var _ article.ArticleDeletionJournal = (*s44Journal)(nil)
var _ article.VerifiedDeletionJournal = (*s44Journal)(nil)
var _ article.DeletionTombstoneReplayer = (*s44ReplayProbe)(nil)

// This adapter maps the real article-owned opaque proof into the existing
// identity consumer port. It does not create a proof or decide eligibility.
type s44Identity struct{ f *s44Fixture }

func (port *s44Identity) LockTransferParties(ctx context.Context, tx pgx.Tx, target *string, source string) (article.DispositionParties, error) {
	parties, err := (&identity.ArticleIdentityLifecycle{}).LockTransferParties(ctx, tx, target, source)
	return article.DispositionParties{SourceID: parties.SourceID, SourceEmail: parties.SourceEmail,
		TargetID: parties.TargetID, TargetEmail: parties.TargetEmail}, err
}

func (port *s44Identity) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, id string) (article.DispositionDeletionCandidate, error) {
	candidate, err := (&identity.ArticleIdentityLifecycle{}).LockDeletionCandidate(ctx, tx, id)
	return article.DispositionDeletionCandidate{UserID: candidate.UserID, Deleted: candidate.Deleted}, err
}

func (port *s44Identity) ClearRetainedEmailIfEligible(ctx context.Context, tx pgx.Tx, source string, proof *article.DispositionCompletionProof) (bool, error) {
	if proof == nil || proof.Reference().Transaction != tx {
		return false, errors.New("real same-transaction completion proof required")
	}
	ids := &s44IDs{}
	verifier := &s44CompletionVerifier{f: port.f, proof: proof, tx: tx}
	lifecycle, err := identity.NewArticleIdentityLifecycle(verifier, audit.Store{}, ids, port.f.clock)
	if err != nil {
		return false, err
	}
	cleared, err := lifecycle.ClearRetainedEmailIfEligible(ctx, tx, source, s44Reference(proof.Reference()))
	if ids.calls != 0 || cleared {
		return false, errors.New("no-retained-contact fixture unexpectedly requested email-release audit")
	}
	return cleared, err
}

type s44CompletionVerifier struct {
	f     *s44Fixture
	proof *article.DispositionCompletionProof
	tx    pgx.Tx
}

func (verifier *s44CompletionVerifier) VerifyCompletion(ctx context.Context, tx pgx.Tx, reference identity.ArticleCompletionReference) (identity.VerifiedArticleCompletion, error) {
	if tx != verifier.tx || !reflect.DeepEqual(reference, s44Reference(verifier.proof.Reference())) {
		return identity.VerifiedArticleCompletion{}, errors.New("original opaque proof reference/transaction mismatch")
	}
	facts, err := verifier.proof.VerifyCompletion(ctx, tx)
	if err != nil {
		return identity.VerifiedArticleCompletion{}, err
	}
	if facts.CompletedOperation == nil || facts.CompletedOperation.AuditEventID == nil {
		return identity.VerifiedArticleCompletion{}, errors.New("real deletion completion facts missing")
	}
	verifier.f.proofCalls++
	operation := facts.CompletedOperation
	return identity.VerifiedArticleCompletion{AfterArticle: s44ArticleState(facts.AfterArticle),
		RemainingActiveCount: facts.RemainingActiveCount, RemainingPendingCount: facts.RemainingPendingCount,
		CompletedOperation: &identity.ArticleDispositionOperation{ID: operation.ID, ArticleID: operation.ArticleID,
			ArticleVersion: operation.ArticleVersion, DeleteKind: string(operation.DeleteKind), SourceID: operation.OrphanedFromUserID,
			Status: string(operation.Status), AuditEventID: *operation.AuditEventID},
		Audit: identity.ArticleDispositionAudit{ID: facts.Audit.ID, ArticleID: facts.Audit.TargetID,
			Action: facts.Audit.Action, Outcome: facts.Audit.Outcome, TraceID: facts.Audit.TraceID}}, nil
}

func s44Reference(reference article.DispositionCompletionReference) identity.ArticleCompletionReference {
	return identity.ArticleCompletionReference{Transaction: reference.Transaction, ExpectedArticleID: reference.ExpectedArticleID,
		SourceID: reference.SourceID, BeforeVersion: reference.BeforeVersion, Action: identity.ArticleDispositionAction(reference.Action),
		NoticeID: reference.NoticeID, NewAuthorID: reference.NewAuthorID, OperationID: reference.OperationID,
		AuditEventID: reference.AuditEventID, BeforeArticle: s44ArticleState(reference.BeforeArticle)}
}

func s44ArticleState(state article.DispositionArticleState) identity.ArticleDispositionState {
	var category *string
	if state.Category != nil {
		category = s44String(string(*state.Category))
	}
	return identity.ArticleDispositionState{Exists: state.Exists, ID: state.ID, Version: state.Version, AuthorID: state.AuthorID,
		SourceID: state.SourceID, Publication: string(state.Publication), Violation: string(state.Violation), Category: category,
		Deletion: string(state.Deletion), OperationID: state.OperationID}
}

var _ article.DispositionIdentityPort = (*s44Identity)(nil)
var _ identity.ArticleCompletionVerifier = (*s44CompletionVerifier)(nil)

// Everything below reuses Dalton's one fixture and opaque-proof adapter.
// These are r22 preparation algorithms, NOT a second fixture or status authority.
type s44EvidenceClock struct {
	real  *s44Clock
	calls atomic.Int64
}

func (clock *s44EvidenceClock) Now() time.Time {
	clock.calls.Add(1)
	return clock.real.Now()
}

// ReplayEvidenceStore is the concrete Store alias embedded by the probe. The
// alias preserves the historical embedded-field name while promoting the
// package-private readback method that production Publish resolves on the
// concrete Store; embedding only the public interface would hide that method and
// force the "publication not configured" fallback.
type ReplayEvidenceStore = article.PostgresReplayEvidenceStore

type s44StoreProbe struct {
	*ReplayEvidenceStore
	mu          sync.Mutex
	events      []string
	overflow    bool
	ackObserver *pgxpool.Conn
	lostACK     bool
	committed   article.DeletionReplayEvidence
}

func (probe *s44StoreProbe) record(event string) {
	probe.mu.Lock()
	defer probe.mu.Unlock()
	if len(probe.events) == 128 {
		probe.overflow = true
		return
	}
	probe.events = append(probe.events, event)
}

func (probe *s44StoreProbe) history() []string {
	probe.mu.Lock()
	defer probe.mu.Unlock()
	return append([]string(nil), probe.events...)
}

func (probe *s44StoreProbe) Begin(ctx context.Context, run, backup string, expected article.AttemptSnapshot,
	facts article.DeletionReplayFacts, candidate string) (article.BeginResult, error) {
	probe.record("Begin")
	return probe.ReplayEvidenceStore.Begin(ctx, run, backup, expected, facts, candidate)
}

func (probe *s44StoreProbe) ReadCurrent(ctx context.Context, run, backup string,
	refresh func(context.Context) (article.DeletionReplayFacts, error)) (article.DeletionReplayRun, *article.DeletionReplayEvidence, error) {
	probe.record("ReadCurrent")
	return probe.ReplayEvidenceStore.ReadCurrent(ctx, run, backup, refresh)
}

func (probe *s44StoreProbe) Fail(ctx context.Context, run, backup string, owner article.AttemptSnapshot, bucket string) error {
	probe.record("Fail")
	return probe.ReplayEvidenceStore.Fail(ctx, run, backup, owner, bucket)
}

var s44LostACK = errors.New("controlled post-commit ACK loss")

func (probe *s44StoreProbe) Publish(ctx context.Context, request article.ReplayPublishRequest) (article.DeletionReplayEvidence, error) {
	probe.record("Publish")
	evidence, err := probe.ReplayEvidenceStore.Publish(ctx, request)
	if err != nil || probe.ackObserver == nil || probe.lostACK {
		return evidence, err
	}
	var committed bool
	err = probe.ackObserver.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM deletion_replay_runs r
		JOIN deletion_replay_evidence e ON e.evidence_ref=r.current_evidence_ref
		AND e.run_id=r.run_id AND e.namespace=r.namespace
		WHERE r.run_id=$1 AND r.namespace=$2 AND r.status='ready'
		AND r.input_digest=$3 AND e.evidence_digest=$3 AND e.evidence_ref=$4)`,
		request.RunID, recovery.ArticleDeletionNamespaceV1, evidence.EvidenceDigest, evidence.EvidenceRef).Scan(&committed)
	if err != nil {
		return article.DeletionReplayEvidence{}, err
	}
	if !committed {
		return article.DeletionReplayEvidence{}, errors.New("ACK injection requires independently visible real commit")
	}
	probe.committed, probe.lostACK = evidence, true
	probe.record("CommitObservedThenAckLost")
	return article.DeletionReplayEvidence{}, s44LostACK
}

func s44Prepared(t *testing.T, completed int) *s44Fixture {
	t.Helper()
	f := s44New(t)
	var err error
	f.ctx, err = platform.ContextWithTraceID(f.ctx, s44Trace)
	s44Must(t, err)
	for index := 2; index <= 8; index++ {
		f.evidenceIDs.remaining = append(f.evidenceIDs.remaining, fmt.Sprintf("synthetic-evidence-%d", index))
		f.tokens.remaining = append(f.tokens.remaining, fmt.Sprintf("synthetic-candidate-%d", index))
	}
	f.evidenceClock = &s44EvidenceClock{real: f.clock}
	f.storeProbe = &s44StoreProbe{ReplayEvidenceStore: f.store}
	f.provider = s44Provider(t, f, f.storeProbe)
	for _, target := range f.targets[:completed] {
		s44Must(t, f.finalizer.Finalize(f.ctx, target.operation, s44Trace))
	}
	return f
}

func s44Provider(t *testing.T, f *s44Fixture, store article.ReplayEvidenceStore) *article.DeletionReplayProvider {
	t.Helper()
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{
		Store: store, Source: f.source, EvidenceIDs: f.evidenceIDs, Tokens: f.tokens, Clock: f.evidenceClock})
	s44Must(t, err)
	return provider
}

func s44Rebuild(f *s44Fixture) {
	s44WireExistingReplay(f.t, f, filepath.Join(f.dir, "synthetic-inventory.json"))
	f.storeProbe.ReplayEvidenceStore = f.store
	f.provider = s44Provider(f.t, f, f.storeProbe)
}

func s44Begin(f *s44Fixture) article.AttemptSnapshot {
	f.t.Helper()
	expected, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	facts, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(f.t, err)
	candidate, err := f.tokens.NewID()
	s44Must(f.t, err)
	result, err := f.store.Begin(f.ctx, s44Run, s44Backup, expected, facts, candidate)
	s44Must(f.t, err)
	switch result.Disposition {
	case article.AttemptCreated, article.AttemptResumedCandidate, article.AttemptJoinedCurrent,
		article.AttemptAdvancedFailed, article.AttemptAdvancedStale:
		return result.Attempt
	default:
		f.t.Fatal("Begin did not return a publishing-authorized disposition")
	}
	return article.AttemptSnapshot{}
}

func s44Request(f *s44Fixture, owner article.AttemptSnapshot, input article.DeletionReplayInput) article.ReplayPublishRequest {
	return article.ReplayPublishRequest{RunID: s44Run, BackupRef: s44Backup, Attempt: owner, Input: input,
		Refresh: func(ctx context.Context) (article.DeletionReplayFacts, error) {
			return f.source.Snapshot(ctx, s44Backup)
		},
		EvidenceIDs: f.evidenceIDs, Clock: f.evidenceClock}
}

func s44ReplayInput(f *s44Fixture, replayed, absent int64) article.DeletionReplayInput {
	f.t.Helper()
	input, err := f.source.ReplayOldBackup(f.ctx, s44Backup)
	s44Must(f.t, err)
	if input.TombstonesReplayed != replayed || input.AlreadyAbsent != absent || f.probe.cause != nil ||
		f.probe.result.Replayed != replayed || f.probe.result.AlreadyAbsent != absent {
		f.t.Fatal("real execution partitions mismatch")
	}
	f.save(fmt.Sprintf("execution-%02d", f.probe.calls), map[string]any{"actual_input": input, "real_finalizer_result": f.probe.result})
	return input
}

func s44Publish(f *s44Fixture, owner article.AttemptSnapshot, input article.DeletionReplayInput) article.DeletionReplayEvidence {
	f.t.Helper()
	evidence, err := f.store.Publish(f.ctx, s44Request(f, owner, input))
	s44Must(f.t, err)
	return evidence
}

func s44ProviderReady(f *s44Fixture, replayed, absent uint64) recovery.ReadyReplayStatusV1 {
	f.t.Helper()
	before := s44EvidenceCount(f)
	raw, err := f.provider.ReplayAndVerify(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	if before == 0 && s44EvidenceCount(f) != 1 {
		f.t.Fatal("first durable Ready did not create exactly one evidence row")
	}
	return s44CheckReady(f, raw, replayed, absent)
}

func s44Mixed(t *testing.T) {
	f := s44Prepared(t, 1)
	s44ProviderReady(f, 2, 1)
	if f.probe.calls != 1 || f.probe.result != (article.DeletionReplayAllResult{Replayed: 2, AlreadyAbsent: 1}) {
		t.Fatal("mixed fixture did not physically execute exactly two deletions and one absence")
	}
}

func s44ReadCounters(f *s44Fixture) [7]int64 {
	var begins, publishes, fails int64
	for _, event := range f.storeProbe.history() {
		switch event {
		case "Begin":
			begins++
		case "Publish":
			publishes++
		case "Fail":
			fails++
		}
	}
	return [7]int64{int64(f.probe.calls), begins, publishes, fails, int64(f.evidenceIDs.calls), int64(f.tokens.calls), f.evidenceClock.calls.Load()}
}

func s44Repeat(t *testing.T) {
	f := s44Prepared(t, 1)
	first := s44ProviderReady(f, 2, 1)
	before, counters := s44Image(f), s44ReadCounters(f)
	for iteration := 0; iteration < 2; iteration++ {
		if iteration == 1 {
			s44Rebuild(f)
		}
		if s44ProviderReady(f, 2, 1) != first {
			t.Fatal("unchanged Facts failed to reuse the entire committed Ready")
		}
		raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
		s44Must(t, err)
		if s44CheckReady(f, raw, 2, 1) != first {
			t.Fatal("rebuilt ReadStatus changed durable Ready metadata")
		}
		if s44Image(f) != before || s44ReadCounters(f) != counters {
			t.Fatal("read/reuse replayed, began, allocated ref/time or wrote rows")
		}
	}
}

func s44Partial(t *testing.T) {
	f := s44Prepared(t, 0)
	f.journal.failOperation = f.targets[1].operation
	raw, err := f.provider.ReplayAndVerify(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	s44Unavailable(t, raw, "failed", "verification_failed")
	f.saveRaw("partial-unavailable.raw.json", raw)
	s44NoEvidence(f)
	for index, target := range f.targets {
		var status string
		var present bool
		s44Must(t, f.observer.QueryRow(f.ctx, `SELECT status,EXISTS(SELECT 1 FROM articles WHERE id=$2)
			FROM deletion_operations WHERE id=$1`, target.operation, target.row.ID).Scan(&status, &present))
		if (index == 0 && (status != "completed" || present)) || (index > 0 && (status != "journaled" || !present)) {
			t.Fatal("partial completion crossed the controlled second-Journal-lookup boundary")
		}
	}
	f.save("partial-before-retry", s44Image(f))
	s44ProviderReady(f, 2, 1)
	if f.probe.calls != 2 || f.probe.result != (article.DeletionReplayAllResult{Replayed: 2, AlreadyAbsent: 1}) {
		t.Fatal("retry accumulated uncommitted physical results")
	}
}

func s44AckLoss(t *testing.T) {
	f := s44Prepared(t, 1)
	f.storeProbe.ackObserver = f.observer
	ready := s44ProviderReady(f, 2, 1)
	if !f.storeProbe.lostACK || ready != s44ReadyOf(f.storeProbe.committed) || f.probe.calls != 1 {
		t.Fatal("ACK loss failed to recover the actual committed evidence without replay")
	}
	events := f.storeProbe.history()
	var begins, publishes int
	for _, event := range events {
		if event == "Begin" {
			begins++
		}
		if event == "Publish" {
			publishes++
		}
	}
	// The concrete Store recovers the unknown ACK through its package-private
	// exact-triplet readback, which this probe cannot intercept. What remains
	// observable is that nothing retried, failed or re-read after the loss.
	if want := []string{"ReadCurrent", "Begin", "Publish", "CommitObservedThenAckLost"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("committed ACK loss must stop at the exact committed evidence, got events %v", events)
	}
	if begins != 1 || publishes != 1 || f.evidenceIDs.calls != 1 || f.evidenceClock.calls.Load() != 1 {
		t.Fatal("committed ACK loss retried Begin/Publish or allocated duplicate evidence metadata")
	}
	for _, event := range events {
		if event == "Fail" {
			t.Fatal("post-commit ACK loss incorrectly called Fail")
		}
	}
	if s44EvidenceCount(f) != 1 {
		t.Fatal("ACK loss appended duplicate evidence")
	}
}

type s44Barrier struct {
	entered, release chan struct{}
	once             sync.Once
}

func s44NewBarrier(t *testing.T) *s44Barrier {
	b := &s44Barrier{entered: make(chan struct{}, 1), release: make(chan struct{})}
	t.Cleanup(b.open)
	return b
}

func (b *s44Barrier) open() { b.once.Do(func() { close(b.release) }) }

func (b *s44Barrier) refresh(f *s44Fixture) func(context.Context) (article.DeletionReplayFacts, error) {
	return func(ctx context.Context) (article.DeletionReplayFacts, error) {
		facts, err := f.source.Snapshot(ctx, s44Backup)
		if err != nil {
			return article.DeletionReplayFacts{}, err
		}
		select {
		case b.entered <- struct{}{}:
		case <-ctx.Done():
			return article.DeletionReplayFacts{}, ctx.Err()
		}
		select {
		case <-b.release:
			return facts, nil
		case <-ctx.Done():
			return article.DeletionReplayFacts{}, ctx.Err()
		}
	}
}

// snapshot holds only the first verified publication snapshot read. Production
// Publish ignores ReplayPublishRequest.Refresh, so the fixture Journal is the
// transaction-free place to keep a Publish outside its row locks.
func (b *s44Barrier) snapshot() func(context.Context) error {
	var once sync.Once
	return func(ctx context.Context) error {
		held := false
		once.Do(func() { held = true })
		if !held {
			return nil
		}
		select {
		case b.entered <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-b.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type s44Published struct {
	evidence article.DeletionReplayEvidence
	err      error
}

func s44LaunchPublish(f *s44Fixture, request article.ReplayPublishRequest, barrier *s44Barrier) <-chan s44Published {
	ctx, cancel := context.WithTimeout(f.ctx, 20*time.Second)
	done, joined := make(chan s44Published, 1), make(chan struct{})
	go func() { defer close(joined); e, err := f.store.Publish(ctx, request); done <- s44Published{e, err} }()
	f.t.Cleanup(func() {
		barrier.open()
		cancel()
		select {
		case <-joined:
		case <-time.After(5 * time.Second):
			f.t.Error("Publish goroutine did not join after release/cancel")
		}
	})
	return done
}

func s44Entered(f *s44Fixture, b *s44Barrier) {
	f.t.Helper()
	select {
	case <-b.entered:
	case <-f.ctx.Done():
		f.t.Fatal("Refresh barrier not reached before deadline")
	}
}

func s44JoinedPublish(f *s44Fixture, done <-chan s44Published) s44Published {
	f.t.Helper()
	select {
	case result := <-done:
		return result
	case <-f.ctx.Done():
		f.t.Fatal("Publish result did not join")
	}
	return s44Published{}
}

func s44Winner(t *testing.T) {
	f := s44Prepared(t, 1)
	winnerOwner := s44Begin(f)
	winnerInput := s44ReplayInput(f, 2, 1)
	b := s44NewBarrier(t)
	request := s44Request(f, winnerOwner, winnerInput)
	request.Refresh = b.refresh(f)
	f.publication.snapshotHook = b.snapshot()
	done := s44LaunchPublish(f, request, b)
	s44Entered(f, b)
	laggerOwner := s44Begin(f)
	if winnerOwner.Generation != laggerOwner.Generation || winnerOwner.Token != laggerOwner.Token {
		t.Fatal("public Begin did not join exact winner handle")
	}
	laggerInput := s44ReplayInput(f, 0, 3)
	b.open()
	winner := s44JoinedPublish(f, done)
	s44Must(t, winner.err)
	before := s44Image(f)
	lagger := s44Publish(f, laggerOwner, laggerInput)
	if lagger != winner.evidence || s44Image(f) != before || s44EvidenceCount(f) != 1 {
		t.Fatal("late real0/3 failed to reuse immutable winner2/1 without writes")
	}
	s44CheckEvidence(f, lagger, 2, 1)
}

func s44Drift(t *testing.T, operation bool) {
	f := s44Prepared(t, 1)
	first := s44ProviderReady(f, 2, 1)
	history := s44EvidenceImage(f)
	beforeFacts, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(t, err)
	s44CommitDrift(f, operation)
	afterFacts, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(t, err)
	want := beforeFacts
	if operation {
		want.DeletionOperationsDigest = afterFacts.DeletionOperationsDigest
	} else {
		want.AuditEventsDigest, want.AuditEventsVerifiedCount = afterFacts.AuditEventsDigest, afterFacts.AuditEventsVerifiedCount
	}
	if afterFacts != want || afterFacts == beforeFacts {
		t.Fatal("fixture failed to isolate operation/audit drift from every other Fact")
	}
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	s44Unavailable(t, raw, "unknown", "evidence_stale")
	f.saveRaw("isolated-drift-stale.raw.json", raw)
	second := s44ProviderReady(f, 0, 3)
	if second.EvidenceRef == first.EvidenceRef || second.EvidenceDigest == first.EvidenceDigest || s44EvidenceCount(f) != 2 {
		t.Fatal("real isolated fact drift did not atomically publish a distinct current")
	}
	s44RequireHistoryPreserved(f, history)
	raw, err = f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	if s44CheckReady(f, raw, 0, 3) != second {
		t.Fatal("republished current did not survive public readback")
	}
}

func s44CommitDrift(f *s44Fixture, operation bool) {
	tx, err := f.pool.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	if operation {
		err = (article.Repository{}).InsertDeletionOperation(f.ctx, tx, article.DeletionOperation{
			ID: "33000000-0000-4000-8000-000000000099", ArticleID: "22000000-0000-4000-8000-000000000099",
			ArticleVersion: 1, DeleteKind: article.DeleteOrdinary, ActorUserID: s44String(s44Admin), ActorRole: s44String("admin"),
			Status: article.DeletionOperationPendingJournal, StartedAt: f.clock.at})
	} else {
		err = (audit.Store{}).Append(f.ctx, tx, audit.Event{ID: "44000000-0000-4000-8000-000000000099",
			OccurredAt: f.clock.at, ActorKind: audit.ActorSystem, ActorSystem: s44String("synthetic-replay-drift"),
			Action: "article.deletion.replay", TargetType: "article", TargetID: f.targets[0].row.ID,
			Outcome: "succeeded", TraceID: s44Trace, Metadata: map[string]any{"synthetic_fixture": true}})
	}
	s44Must(f.t, err)
	s44Must(f.t, tx.Commit(f.ctx))
	f.save("isolated-fact-drift", map[string]bool{"synthetic_operation_insert": operation, "real_audit_append": !operation})
}

func s44RunUnlocked(f *s44Fixture) {
	// Dalton's observer reserves exactly one pool connection. No other pool
	// connection may remain acquired across the real Source Refresh barrier.
	if f.pool.Stat().AcquiredConns() != 1 {
		f.t.Fatal("Refresh retained a transaction/pool connection across external I/O")
	}
	tx, err := f.observer.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	var run string
	err = tx.QueryRow(f.ctx, `SELECT run_id FROM deletion_replay_runs WHERE run_id=$1 AND namespace=$2 FOR UPDATE NOWAIT`,
		s44Run, recovery.ArticleDeletionNamespaceV1).Scan(&run)
	s44Must(f.t, err)
	if run != s44Run {
		f.t.Fatal("NOWAIT observer acquired the wrong run")
	}
	s44Must(f.t, tx.Rollback(f.ctx))
	var one int
	s44Must(f.t, f.pool.QueryRow(f.ctx, "SELECT 1").Scan(&one))
	if one != 1 || f.pool.Stat().AcquiredConns() != 1 {
		f.t.Fatal("Refresh did not release pool resources")
	}
}

func s44AdvanceWithInventoryB(f *s44Fixture, old article.AttemptSnapshot) article.DeletionReplayEvidence {
	s44InstallInventory(f, true)
	if old.Status == article.ReplayRunRunning {
		s44Must(f.t, f.store.Fail(f.ctx, s44Run, s44Backup, old, "verification_failed"))
	}
	newOwner := s44Begin(f)
	if newOwner.Generation <= old.Generation || newOwner.Token == old.Token {
		f.t.Fatal("legal Store path failed to advance exact owner")
	}
	return s44Publish(f, newOwner, s44ReplayInput(f, 0, 3))
}

func s44RefreshPublish(t *testing.T) {
	f := s44Prepared(t, 1)
	owner := s44Begin(f)
	input := s44ReplayInput(f, 2, 1)
	b := s44NewBarrier(t)
	request := s44Request(f, owner, input)
	request.Refresh = b.refresh(f)
	f.publication.snapshotHook = b.snapshot()
	f.save("publish-refresh-before", s44Image(f))
	done := s44LaunchPublish(f, request, b)
	s44Entered(f, b)
	s44RunUnlocked(f)
	winner := s44AdvanceWithInventoryB(f, owner)
	s44CheckEvidence(f, winner, 0, 3)
	if s44EvidenceCount(f) != 1 {
		t.Fatal("superseding Publish did not produce one first evidence")
	}
	after := s44Image(f)
	f.save("publish-refresh-new-owner-current", after)
	b.open()
	old := s44JoinedPublish(f, done)
	if !errors.Is(old.err, article.ErrDeletionReplayAttemptNotOwner) || s44Image(f) != after {
		t.Fatal("old Publish wrote after owner/current advanced during transaction-free Refresh")
	}
}

func s44RefreshRead(t *testing.T) {
	f := s44Prepared(t, 1)
	s44ProviderReady(f, 2, 1)
	owner, err := f.store.LoadAttempt(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	b := s44NewBarrier(t)
	f.save("read-refresh-before", s44Image(f))
	ctx, cancel := context.WithTimeout(f.ctx, 20*time.Second)
	done, joined := make(chan error, 1), make(chan struct{})
	go func() {
		defer close(joined)
		_, _, err := f.store.ReadCurrent(ctx, s44Run, s44Backup, b.refresh(f))
		done <- err
	}()
	t.Cleanup(func() {
		b.open()
		cancel()
		select {
		case <-joined:
		case <-time.After(5 * time.Second):
			t.Error("ReadCurrent goroutine did not join")
		}
	})
	s44Entered(f, b) // Captured A matches old Ready; stale must require the final recheck.
	s44RunUnlocked(f)
	winner := s44AdvanceWithInventoryB(f, owner)
	s44CheckEvidence(f, winner, 0, 3)
	if s44EvidenceCount(f) != 2 {
		t.Fatal("current replacement did not preserve old immutable evidence")
	}
	after := s44Image(f)
	f.save("read-refresh-new-owner-current", after)
	b.open()
	select {
	case err = <-done:
		if !errors.Is(err, article.ErrDeletionReplayInputChanged) {
			t.Fatal("ReadCurrent accepted a valid-but-superseded A combination")
		}
	case <-f.ctx.Done():
		t.Fatal("ReadCurrent result did not join")
	}
	if s44Image(f) != after {
		t.Fatal("ReadCurrent wrote while discarding a stale combination")
	}
}

func s44InstallInventory(f *s44Fixture, b bool) {
	raw := s44Inventory
	if b {
		raw = strings.Replace(raw, `"generated_at":"2026-10-20T00:00:01.000000Z"`, `"generated_at":"2026-10-20T00:00:02.000000Z"`, 1)
		at := strings.LastIndex(raw, `,"inventory_digest":`)
		if at < 0 {
			f.t.Fatal("fixed inventory fixture lacks its digest field")
		}
		payload := raw[:at] + "}"
		raw = raw[:at] + `,"inventory_digest":"` + recovery.SHA256Digest([]byte(payload)) + `"}`
	}
	version := s44Suffix(f.t)
	f.saveRaw("inventory-issued-"+version+".json", []byte(raw))
	staged := filepath.Join(f.dir, "inventory-staged-"+version+".json")
	s44Must(f.t, os.WriteFile(staged, []byte(raw), 0600))
	file, err := os.OpenFile(staged, os.O_RDWR, 0)
	s44Must(f.t, err)
	syncErr, closeErr := file.Sync(), file.Close()
	s44Must(f.t, syncErr)
	s44Must(f.t, closeErr)
	s44Must(f.t, os.Rename(staged, filepath.Join(f.dir, "synthetic-inventory.json")))
	reader, err := article.NewFileBackupInventoryReader(filepath.Join(f.dir, "synthetic-inventory.json"))
	s44Must(f.t, err)
	_, err = reader.Read(f.ctx)
	s44Must(f.t, err)
	f.t.Logf("inventory_install=%s synthetic=true retained_issued_bytes=true", version)
}

func s44History(t *testing.T) {
	f := s44Prepared(t, 1)
	a := s44ProviderReady(f, 2, 1)
	historyA := s44EvidenceImage(f)
	s44InstallInventory(f, true)
	b := s44ProviderReady(f, 0, 3)
	if a.EvidenceRef == b.EvidenceRef || s44EvidenceCount(f) != 2 {
		t.Fatal("inventory B did not create its own semantic evidence")
	}
	s44InstallInventory(f, false)
	chosen := s44ProviderReady(f, 2, 1)
	if chosen != a || f.probe.result != (article.DeletionReplayAllResult{Replayed: 0, AlreadyAbsent: 3}) || s44EvidenceCount(f) != 2 {
		t.Fatal("A return failed to distinguish real0/3 from unique historical2/1")
	}
	s44RequireHistoryPreserved(f, historyA)
}

func s44Legacy(t *testing.T) {
	f := s44Prepared(t, 1)
	input := s44ReplayInput(f, 2, 1)
	evidence := s44SyntheticEvidence(f, input, "synthetic-legacy-2-1")
	s44SeedHistory(f, []article.DeletionReplayEvidence{evidence}, evidence)
	history := s44EvidenceImage(f)
	before, counters := s44Image(f), s44ReadCounters(f)
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	if s44CheckReady(f, raw, 2, 1) != s44ReadyOf(evidence) || s44ProviderReady(f, 2, 1) != s44ReadyOf(evidence) ||
		s44Image(f) != before || s44ReadCounters(f) != counters {
		t.Fatal("legacy gen0 read/reuse rewrote or replayed synthetic history")
	}
	s44RequireGeneration(f, 0)
	s44InstallInventory(f, true)
	s44ProviderReady(f, 0, 3)
	s44RequireGeneration(f, 1)
	s44RequireHistoryPreserved(f, history)
}

func s44Ambiguous(t *testing.T) {
	f := s44Prepared(t, 1)
	aInput := s44ReplayInput(f, 2, 1)
	a := s44SyntheticEvidence(f, aInput, "synthetic-history-A-2-1")
	otherInput := aInput
	otherInput.TombstonesReplayed, otherInput.AlreadyAbsent = 0, 3
	other := s44SyntheticEvidence(f, otherInput, "synthetic-history-A-0-3")
	s44InstallInventory(f, true)
	b := s44SyntheticEvidence(f, s44ReplayInput(f, 0, 3), "synthetic-current-B")
	s44SeedHistory(f, []article.DeletionReplayEvidence{a, other, b}, b)
	history, replays := s44EvidenceImage(f), f.probe.calls
	s44InstallInventory(f, false)
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	s44Unavailable(t, raw, "unknown", "evidence_stale")
	f.saveRaw("ambiguous-history-stale.raw.json", raw)
	raw, err = f.provider.ReplayAndVerify(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	s44Unavailable(t, raw, "failed", "evidence_unreadable")
	f.saveRaw("ambiguous-history-rejected.raw.json", raw)
	f.save("ambiguous-history-rejected-sql", s44Image(f))
	if f.probe.calls != replays+1 || !reflect.DeepEqual(s44EvidenceImage(f), history) {
		t.Fatal("history ambiguity skipped real replay or changed immutable rows")
	}
	var current, status string
	s44Must(t, f.observer.QueryRow(f.ctx, `SELECT current_evidence_ref,status FROM deletion_replay_runs WHERE run_id=$1 AND namespace=$2`,
		s44Run, recovery.ArticleDeletionNamespaceV1).Scan(&current, &status))
	if current != b.EvidenceRef || status != "failed" {
		t.Fatal("ambiguous A histories installed a fake current Ready")
	}
}

func s44SyntheticEvidence(f *s44Fixture, input article.DeletionReplayInput, ref string) article.DeletionReplayEvidence {
	e := article.DeletionReplayEvidence{EvidenceVersion: recovery.ReplayEvidenceVersionV1, RunID: s44Run, BackupRef: s44Backup,
		Namespace: recovery.ArticleDeletionNamespaceV1, JournalHeadSequence: input.JournalHeadSequence, JournalHeadHash: input.JournalHeadHash,
		InventoryScope: input.InventoryScope, BackupCutoff: input.BackupCutoff, InventoryDigest: input.InventoryDigest,
		DeletionOperationsDigest: input.DeletionOperationsDigest, AuditEventsDigest: input.AuditEventsDigest,
		TombstonesTotal: input.TombstonesTotal, TombstonesReplayed: input.TombstonesReplayed, AlreadyAbsent: input.AlreadyAbsent,
		DeletionOperationsReconciledCount: input.DeletionOperationsReconciledCount, AuditEventsVerifiedCount: input.AuditEventsVerifiedCount,
		EvidenceRef: ref, CompletedAt: f.clock.at}
	var err error
	e.EvidenceDigest, err = s44ReadyOf(e).CanonicalDigest()
	s44Must(f.t, err)
	return e
}

func s44SeedHistory(f *s44Fixture, histories []article.DeletionReplayEvidence, current article.DeletionReplayEvidence) {
	tx, err := f.pool.Begin(f.ctx)
	s44Must(f.t, err)
	defer s44Rollback(f.t, tx)
	repository := article.Repository{}
	s44Must(f.t, repository.InsertDeletionReplayRun(f.ctx, tx, article.DeletionReplayRun{RunID: s44Run, BackupRef: s44Backup,
		Namespace: recovery.ArticleDeletionNamespaceV1, Status: article.ReplayRunRunning, StartedAt: f.clock.at, UpdatedAt: f.clock.at}))
	for _, e := range histories {
		s44Must(f.t, repository.InsertDeletionReplayEvidence(f.ctx, tx, e))
	}
	s44Must(f.t, repository.SetDeletionReplayRunCurrent(f.ctx, tx, s44Run, recovery.ArticleDeletionNamespaceV1,
		current.EvidenceDigest, current.EvidenceRef, current.CompletedAt))
	s44Must(f.t, tx.Commit(f.ctx))
	f.save("synthetic-history-provenance", map[string]any{"synthetic_compatibility_history": true, "rows": len(histories), "not_physical_provenance": true})
}

func s44RequireGeneration(f *s44Fixture, expected int64) {
	var generation int64
	var hasToken bool
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT attempt_generation,attempt_token IS NOT NULL FROM deletion_replay_runs
		WHERE run_id=$1 AND namespace=$2`, s44Run, recovery.ArticleDeletionNamespaceV1).Scan(&generation, &hasToken))
	if generation != expected || hasToken != (expected > 0) {
		f.t.Fatal("legacy generation/token transition mismatch")
	}
}

func s44ReadyOf(e article.DeletionReplayEvidence) recovery.ReadyReplayStatusV1 {
	return recovery.ReadyReplayStatusV1{EvidenceVersion: e.EvidenceVersion, RunID: e.RunID, BackupRef: e.BackupRef, Namespace: e.Namespace,
		JournalHeadSequence: uint64(e.JournalHeadSequence), JournalHeadHash: e.JournalHeadHash, InventoryScope: e.InventoryScope,
		BackupCutoff: e.BackupCutoff.UTC().Format(time.RFC3339Nano), InventoryDigest: e.InventoryDigest,
		DeletionOperationsDigest: e.DeletionOperationsDigest, AuditEventsDigest: e.AuditEventsDigest,
		TombstonesTotal: uint64(e.TombstonesTotal), TombstonesReplayed: uint64(e.TombstonesReplayed), AlreadyAbsent: uint64(e.AlreadyAbsent),
		DeletionOperationsReconciledCount: uint64(e.DeletionOperationsReconciledCount), AuditEventsVerifiedCount: uint64(e.AuditEventsVerifiedCount),
		Status: "ready", EvidenceDigest: e.EvidenceDigest, EvidenceRef: e.EvidenceRef, CompletedAt: e.CompletedAt.UTC().Format(time.RFC3339Nano)}
}

func s44ReadEvidenceSQL(f *s44Fixture, ref string) article.DeletionReplayEvidence {
	var e article.DeletionReplayEvidence
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT evidence_version,run_id,backup_ref,namespace,journal_head_sequence,journal_head_hash,
		inventory_scope,backup_cutoff,inventory_digest,deletion_operations_digest,audit_events_digest,tombstones_total,
		tombstones_replayed,already_absent,deletion_operations_reconciled_count,audit_events_verified_count,
		evidence_digest,evidence_ref,completed_at FROM deletion_replay_evidence WHERE evidence_ref=$1`, ref).Scan(
		&e.EvidenceVersion, &e.RunID, &e.BackupRef, &e.Namespace, &e.JournalHeadSequence, &e.JournalHeadHash,
		&e.InventoryScope, &e.BackupCutoff, &e.InventoryDigest, &e.DeletionOperationsDigest, &e.AuditEventsDigest, &e.TombstonesTotal,
		&e.TombstonesReplayed, &e.AlreadyAbsent, &e.DeletionOperationsReconciledCount, &e.AuditEventsVerifiedCount,
		&e.EvidenceDigest, &e.EvidenceRef, &e.CompletedAt))
	return e
}

func s44CheckEvidence(f *s44Fixture, e article.DeletionReplayEvidence, replayed, absent uint64) recovery.ReadyReplayStatusV1 {
	raw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(f.t, err)
	ready := s44CheckReady(f, raw, replayed, absent)
	if ready != s44ReadyOf(e) {
		f.t.Fatal("public Provider readback differs from the real Store result")
	}
	return ready
}

func s44CheckReady(f *s44Fixture, raw []byte, replayed, absent uint64) recovery.ReadyReplayStatusV1 {
	f.t.Helper()
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	s44Must(f.t, err)
	if decoded.Ready == nil {
		if decoded.Unavailable != nil {
			f.t.Fatalf("expected Ready; got status=%s reason=%s", decoded.Unavailable.Status, decoded.Unavailable.ReasonCode)
		}
		f.t.Fatal("shared decoder returned no Ready variant")
	}
	r := *decoded.Ready
	f.assertReadyFields(raw)
	digest, err := r.CanonicalDigest()
	s44Must(f.t, err)
	e := s44ReadEvidenceSQL(f, r.EvidenceRef)
	if r != s44ReadyOf(e) || r.TombstonesReplayed != replayed || r.AlreadyAbsent != absent || digest != r.EvidenceDigest ||
		e.BackupCutoff.Nanosecond()%1000 != 0 || e.CompletedAt.Nanosecond()%1000 != 0 {
		f.t.Fatal("all committed evidence fields/canonical/partitions mismatch")
	}
	var ref, input, status, backup string
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT current_evidence_ref,input_digest,status,backup_ref FROM deletion_replay_runs
		WHERE run_id=$1 AND namespace=$2`, s44Run, recovery.ArticleDeletionNamespaceV1).Scan(&ref, &input, &status, &backup))
	if ref != r.EvidenceRef || input != digest || status != "ready" || backup != s44Backup {
		f.t.Fatal("current/input_digest/status/backup are not atomically bound")
	}
	facts, err := f.source.Snapshot(f.ctx, s44Backup)
	s44Must(f.t, err)
	wantFacts := article.DeletionReplayFacts{JournalHeadSequence: e.JournalHeadSequence, JournalHeadHash: e.JournalHeadHash,
		InventoryScope: e.InventoryScope, BackupCutoff: e.BackupCutoff.UTC(), InventoryDigest: e.InventoryDigest,
		DeletionOperationsDigest: e.DeletionOperationsDigest, AuditEventsDigest: e.AuditEventsDigest, TombstonesTotal: e.TombstonesTotal,
		DeletionOperationsReconciledCount: e.DeletionOperationsReconciledCount, AuditEventsVerifiedCount: e.AuditEventsVerifiedCount}
	if facts != wantFacts || facts.JournalHeadSequence != 3 || facts.TombstonesTotal != 3 {
		f.t.Fatal("current evidence failed complete fresh Facts10 binding")
	}
	s44FactSQLOracle(f, facts)
	s44CompletedSQL(f)
	if f.storeProbe != nil && f.storeProbe.overflow {
		f.t.Fatal("bounded observation buffer overflow")
	}
	f.observationIndex++
	prefix := fmt.Sprintf("ready-%02d", f.observationIndex)
	f.saveRaw(prefix+".wire.json", raw)
	f.save(prefix+"-sql", s44Image(f))
	f.save(prefix+"-facts-and-last-execution", map[string]any{"fresh_facts": facts,
		"observed_replay_calls": f.probe.calls, "last_real_execution": f.probe.result, "chosen_evidence": e})
	return r
}

func s44FactSQLOracle(f *s44Fixture, facts article.DeletionReplayFacts) {
	rows, err := f.observer.Query(f.ctx, `SELECT id::text,status,article_version,completed_at FROM deletion_operations ORDER BY id`)
	s44Must(f.t, err)
	var payload strings.Builder
	var count int64
	for rows.Next() {
		var id, status, completedText string
		var version int64
		var completed *time.Time
		if err = rows.Scan(&id, &status, &version, &completed); err != nil {
			rows.Close()
			s44Must(f.t, err)
		}
		if completed != nil {
			completedText = completed.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
		}
		fmt.Fprintf(&payload, "%s\x00%s\x00%d\x00%s\n", id, status, version, completedText)
		if status == "completed" {
			count++
		}
	}
	err = rows.Err()
	rows.Close()
	s44Must(f.t, err)
	if recovery.SHA256Digest([]byte(payload.String())) != facts.DeletionOperationsDigest || count != facts.DeletionOperationsReconciledCount {
		f.t.Fatal("independent operation SQL digest/count oracle mismatch")
	}
	s44AuditFactSQLOracle(f, facts)
}

func s44AuditFactSQLOracle(f *s44Fixture, facts article.DeletionReplayFacts) {
	rows, err := f.observer.Query(f.ctx, `SELECT id::text,target_id,action,occurred_at FROM audit_events WHERE target_type='article'
		AND action IN ('article.delete','article.violation.delete','article.orphan.delete','article.deletion.replay') ORDER BY target_id,id`)
	s44Must(f.t, err)
	var payload strings.Builder
	var count int64
	for rows.Next() {
		var id, target, action string
		var at time.Time
		if err = rows.Scan(&id, &target, &action, &at); err != nil {
			rows.Close()
			s44Must(f.t, err)
		}
		fmt.Fprintf(&payload, "%s\x00%s\x00%s\x00%s\n", id, target, action, at.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano))
		count++
	}
	err = rows.Err()
	rows.Close()
	s44Must(f.t, err)
	if recovery.SHA256Digest([]byte(payload.String())) != facts.AuditEventsDigest || count != facts.AuditEventsVerifiedCount {
		f.t.Fatal("independent audit SQL digest/count oracle mismatch")
	}
}

func s44CompletedSQL(f *s44Fixture) {
	trace := s44Trace
	if f.probe != nil && f.probe.result.Replayed == 3 {
		trace = f.probe.trace
	}
	var present int
	s44Must(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM articles").Scan(&present))
	if present != 0 {
		f.t.Fatal("real replay left an Article present")
	}
	for index, target := range f.targets {
		var status, auditID, keyID, keyVersion string
		var version, sequence int64
		var completed time.Time
		var noClaim bool
		s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT status,audit_event_id::text,completed_at,article_version,tombstone_sequence,
			journal_key_id,journal_key_version,claimed_by IS NULL AND locked_until IS NULL FROM deletion_operations WHERE id=$1`, target.operation).
			Scan(&status, &auditID, &completed, &version, &sequence, &keyID, &keyVersion, &noClaim))
		if status != "completed" || auditID != target.audit || !completed.Equal(f.clock.at) || version != 1 || sequence != int64(index+1) ||
			keyID != "synthetic-key" || keyVersion != "test-v1" || !noClaim {
			f.t.Fatal("real operation completion/binding/lease SQL mismatch")
		}
		event := s44ReadAuditSQL(f, auditID)
		metadata := map[string]any{"article_version": float64(1), "delete_kind": string(target.kind)}
		action := "article.delete"
		if target.kind == article.DeleteOrdinary {
			metadata["authority"] = "administrator"
		}
		if target.kind == article.DeleteViolating {
			action = "article.violation.delete"
		}
		if target.kind == article.DeleteOrphan {
			action = "article.orphan.delete"
		}
		want := audit.Event{ID: auditID, OccurredAt: f.clock.at, ActorKind: audit.ActorUser, ActorUserID: s44String(s44Admin),
			ActorRole: s44String("admin"), Action: action, TargetType: "article", TargetID: target.row.ID, Outcome: "succeeded", TraceID: trace, Metadata: metadata}
		if !reflect.DeepEqual(event, want) {
			f.t.Fatal("independent audit full-row SQL oracle mismatch")
		}
	}
}

func s44ReadAuditSQL(f *s44Fixture, id string) audit.Event {
	var e audit.Event
	var metadata []byte
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT id::text,occurred_at,actor_kind,actor_user_id::text,actor_system,actor_role,
		action,target_type,target_id,outcome,error_code,trace_id,source_fingerprint,metadata FROM audit_events WHERE id=$1`, id).Scan(
		&e.ID, &e.OccurredAt, &e.ActorKind, &e.ActorUserID, &e.ActorSystem, &e.ActorRole, &e.Action, &e.TargetType,
		&e.TargetID, &e.Outcome, &e.ErrorCode, &e.TraceID, &e.SourceFingerprint, &metadata))
	s44Must(f.t, json.Unmarshal(metadata, &e.Metadata))
	e.OccurredAt = e.OccurredAt.UTC()
	return e
}

func s44Unavailable(t *testing.T, raw []byte, status, reason string) {
	t.Helper()
	value, err := recovery.DecodeReplayStatusV1(raw)
	s44Must(t, err)
	if value.Ready != nil || value.Unavailable == nil || value.Unavailable.Status != status || value.Unavailable.ReasonCode != reason {
		t.Fatalf("expected Unavailable status=%s reason=%s", status, reason)
	}
}

func s44EvidenceCount(f *s44Fixture) int {
	var count int
	s44Must(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM deletion_replay_evidence").Scan(&count))
	return count
}

func s44NoEvidence(f *s44Fixture) {
	var count int
	s44Must(f.t, f.observer.QueryRow(f.ctx, `SELECT count(*) FROM deletion_replay_runs
		WHERE status='ready' OR current_evidence_ref IS NOT NULL OR input_digest IS NOT NULL`).Scan(&count))
	if count != 0 || s44EvidenceCount(f) != 0 {
		f.t.Fatal("unfinished execution committed Ready/evidence/partial pointer")
	}
}

func s44EvidenceImage(f *s44Fixture) map[string]json.RawMessage {
	rows, err := f.observer.Query(f.ctx, `SELECT evidence_ref,jsonb_build_object('row',to_jsonb(e),'xmin',e.xmin::text,'ctid',e.ctid::text)
		FROM deletion_replay_evidence e ORDER BY evidence_ref`)
	s44Must(f.t, err)
	image := make(map[string]json.RawMessage)
	for rows.Next() {
		var ref string
		var raw []byte
		if err = rows.Scan(&ref, &raw); err != nil {
			rows.Close()
			s44Must(f.t, err)
		}
		image[ref] = append([]byte(nil), raw...)
	}
	err = rows.Err()
	rows.Close()
	s44Must(f.t, err)
	return image
}

func s44RequireHistoryPreserved(f *s44Fixture, before map[string]json.RawMessage) {
	after := s44EvidenceImage(f)
	for ref, row := range before {
		if !bytes.Equal(after[ref], row) {
			f.t.Fatal("immutable historical evidence row changed")
		}
	}
}

func s44Image(f *s44Fixture) string {
	image := make(map[string]json.RawMessage)
	for _, table := range []string{"articles", "deletion_operations", "audit_events", "users", "identity_identifiers", "deletion_replay_runs", "deletion_replay_evidence"} {
		projection := "to_jsonb(t)"
		if table == "users" {
			projection += " - 'permanent_password_digest'"
		}
		if table == "deletion_replay_runs" {
			projection += " - 'attempt_token'"
		}
		var raw []byte
		query := "SELECT coalesce(jsonb_agg(v ORDER BY v::text),'[]'::jsonb) FROM (SELECT jsonb_build_object('row'," + projection +
			",'xmin',t.xmin::text,'ctid',t.ctid::text) AS v FROM " + pgx.Identifier{f.schema, table}.Sanitize() + " t) q"
		s44Must(f.t, f.observer.QueryRow(f.ctx, query).Scan(&raw))
		image[table] = raw
	}
	raw, err := json.Marshal(image)
	s44Must(f.t, err)
	return string(raw)
}

type s44ChildSpec struct {
	Schema    string
	Directory string
	At        time.Time
	Records   []article.ArticleDeletionTombstone
}

func s44Crash(t *testing.T) {
	f := s44Prepared(t, 0)
	spec := s44ChildSpec{Schema: f.schema, Directory: f.dir, At: f.clock.at, Records: f.journal.records}
	f.save("crash-child-spec", spec)
	child := s44StartChild(f, filepath.Join(f.dir, "crash-child-spec.json"))
	select {
	case <-child.marker:
	case <-child.joined:
		t.Fatal("child exited before post-Replay/pre-Publish handshake")
	case <-f.ctx.Done():
		t.Fatal("child handshake deadline")
	}
	s44NoEvidence(f)
	s44CompletedSQL(f)
	f.save("crash-after-real-replay-before-publish", s44Image(f))
	_, err := io.WriteString(child.input, "exit\n")
	s44Must(t, err)
	s44Must(t, child.input.Close())
	select {
	case <-child.joined:
	case <-f.ctx.Done():
		t.Fatal("crash child did not join")
	}
	var exit *exec.ExitError
	if !errors.As(child.err, &exit) || exit.ExitCode() != 23 || child.scanErr != nil || child.output.truncated || child.stderr.truncated {
		t.Fatal("actual child did not produce the expected nonzero crash exit and fully joined bounded output")
	}
	f.saveRaw("crash-child-stdout.txt", child.output.bytes())
	f.saveRaw("crash-child-stderr.txt", child.stderr.bytes())
	var live int
	s44Must(t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name=$1", f.schema+"_crash_child").Scan(&live))
	if live != 0 {
		t.Fatal("joined crash child retained PostgreSQL backends")
	}
	s44Rebuild(f)
	s44ProviderReady(f, 0, 3)
	if f.probe.calls != 1 || f.probe.result != (article.DeletionReplayAllResult{Replayed: 0, AlreadyAbsent: 3}) {
		t.Fatal("post-crash first evidence invented lost3/0 execution history")
	}
	before := s44Image(f)
	s44ProviderReady(f, 0, 3)
	if s44Image(f) != before || f.probe.calls != 1 {
		t.Fatal("post-crash durable Ready was not stable")
	}
}

func s44CrashChild(t *testing.T, path string) {
	raw, err := os.ReadFile(path)
	s44Must(t, err)
	var spec s44ChildSpec
	s44Must(t, json.Unmarshal(raw, &spec))
	suffix := strings.TrimPrefix(spec.Schema, "han_s44red_")
	if !strings.HasPrefix(spec.Schema, "han_s44red_") || len(suffix) != 24 || strings.Trim(suffix, "0123456789abcdef") != "" || len(spec.Records) != 3 ||
		!filepath.IsAbs(spec.Directory) || filepath.Clean(spec.Directory) != spec.Directory {
		t.Fatal("invalid retained child attachment")
	}
	f := &s44Fixture{t: t, schema: spec.Schema, dir: spec.Directory, created: true}
	f.ctx, f.cancel = context.WithTimeout(context.WithValue(context.Background(), s44MarkerKey{}, "original-parent"), 20*time.Second)
	t.Cleanup(f.close)
	config := s44Config(t, f.schema)
	config.ConnConfig.RuntimeParams["application_name"] = f.schema + "_crash_child"
	f.pool, err = pgxpool.NewWithConfig(f.ctx, config)
	s44Must(t, err)
	f.preflight(config.ConnConfig.User)
	f.clock.at = spec.At
	f.ctx, err = platform.ContextWithTraceID(f.ctx, s44Trace)
	s44Must(t, err)
	f.targets, f.journal = s44Targets(spec.At.Add(-time.Minute)), &s44Journal{records: spec.Records}
	f.observer, err = f.pool.Acquire(f.ctx)
	s44Must(t, err)
	s44Must(t, f.observer.QueryRow(f.ctx, "SELECT pg_backend_pid()").Scan(&f.observerPID))
	f.wireExisting(false) // Same fixture wiring; no CREATE/migrate/seed or inventory write.
	_ = s44Begin(f)
	s44ReplayInput(f, 3, 0)
	s44NoEvidence(f)
	s44CompletedSQL(f)
	if f.auditIDs.calls != 3 || f.proofCalls != 1 {
		t.Fatal("crash child did not complete the real three-kind/proof chain")
	}
	f.close() // PostgreSQL/runtime resources are joined before deliberate exit.
	f.pool, f.observer, f.cancel, f.probe = nil, nil, nil, nil
	if t.Failed() {
		t.Fatal("crash child resources failed before handshake")
	}
	_, err = fmt.Fprintln(os.Stdout, "S44_REPLAY_DONE_BEFORE_PUBLISH_3_0")
	s44Must(t, err)
	command, err := bufio.NewReader(io.LimitReader(os.Stdin, 16)).ReadString('\n')
	s44Must(t, err)
	if command != "exit\n" {
		t.Fatal("child exit handshake mismatch")
	}
	os.Exit(23)
}

type s44LimitedBuffer struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	truncated bool
}

func (buffer *s44LimitedBuffer) Write(p []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	n := len(p)
	available := 65536 - buffer.buffer.Len()
	if len(p) > available {
		p = p[:available]
		buffer.truncated = true
	}
	_, _ = buffer.buffer.Write(p)
	return n, nil
}

func (buffer *s44LimitedBuffer) bytes() []byte {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return append([]byte(nil), buffer.buffer.Bytes()...)
}

type s44Child struct {
	input          io.WriteCloser
	marker, joined chan struct{}
	output, stderr s44LimitedBuffer
	err, scanErr   error
}

func s44StartChild(f *s44Fixture, spec string) *s44Child {
	executable, err := os.Executable()
	s44Must(f.t, err)
	ctx, cancel := context.WithTimeout(f.ctx, 30*time.Second)
	child := &s44Child{marker: make(chan struct{}), joined: make(chan struct{})}
	started := false
	f.t.Cleanup(func() {
		cancel()
		if child.input != nil {
			_ = child.input.Close()
		}
		if !started {
			return
		}
		select {
		case <-child.joined:
		case <-time.After(5 * time.Second):
			f.t.Error("child/output/process wait did not join after cancellation")
		}
	})
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestArticleReplayPostgres$", "-test.count=1", "-test.timeout=30s", "-s44-replay-child-spec="+spec)
	cmd.Env = s44ChildEnvironment(f.t)
	cmd.Stderr = &child.stderr
	child.input, err = cmd.StdinPipe()
	s44Must(f.t, err)
	output, err := cmd.StdoutPipe()
	s44Must(f.t, err)
	if err = cmd.Start(); err != nil {
		cancel()
		_ = child.input.Close()
		_ = output.Close()
		s44Must(f.t, err)
	}
	started = true
	go s44CollectChild(cmd, output, child, cancel)
	return child
}

func s44CollectChild(cmd *exec.Cmd, output io.ReadCloser, child *s44Child, cancel context.CancelFunc) {
	defer close(child.joined)
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), 4096)
	var once sync.Once
	for scanner.Scan() {
		line := scanner.Text()
		_, _ = child.output.Write(append([]byte(line), '\n'))
		if line == "S44_REPLAY_DONE_BEFORE_PUBLISH_3_0" {
			once.Do(func() { close(child.marker) })
		}
	}
	child.scanErr = scanner.Err()
	if child.scanErr != nil {
		cancel()
	}
	_ = output.Close()
	child.err = cmd.Wait() // Sole owner; output reader reaches EOF first.
}

func s44ChildEnvironment(t *testing.T) []string {
	t.Helper()
	var environment []string
	for _, key := range []string{"SystemRoot", "WINDIR", "TEMP", "TMP"} {
		if value, exists := os.LookupEnv(key); exists {
			environment = append(environment, key+"="+value)
		}
	}
	dsn, exists := os.LookupEnv("HAN_TEST_POSTGRES_DSN")
	if !exists || strings.TrimSpace(dsn) == "" {
		t.Fatal("approved PG child requires its process-scoped DSN")
	}
	return append(environment, "HAN_TEST_POSTGRES_DSN="+dsn)
}

var _ article.ReplayEvidenceStore = (*s44StoreProbe)(nil)

//go:build integration

package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/migration"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecoveryAdmissionStorePostgres(t *testing.T) {
	t.Run("CanonicalNullFieldsAndInitialProof", func(t *testing.T) {
		t.Run("CompleteProjection", testS49PGProjection)
		t.Run("RestoreAndInitialProofProjection", testS49PGProofProjection)
		t.Run("NullEmptyAndLegacyFingerprint", testS49PGLegacyFingerprint)
		t.Run("SQLNullVersusEmptyAndUTC", testS49PGNullAndTime)
	})
	t.Run("CompleteRepeatableReadAndRowsError", func(t *testing.T) {
		t.Run("FullSetStableAcrossIndependentWriter", testS49PGRR)
		t.Run("NativeRowsErrorNoPartialSet", testS49PGRowsError)
		t.Run("MissingTablesNotEmpty", testS49PGMissing)
		t.Run("CancelledReadNoWrite", testS49PGCancel)
		t.Run("ConcurrentTransitionSingleWinner", testS49PGConcurrentTransition)
		t.Run("ReadCommitAckLossNoProof", testS49PGReadAckLoss)
		t.Run("WriteCommitUnknownIndependentReadback", testS49PGWriteAckLoss)
	})
	t.Run("RevisionMutationAndOverflow", func(t *testing.T) {
		t.Run("RunRestoreInstallAndRollback", testS49PGRevision)
		t.Run("ReviewAndOperationWrites", testS49PGOperationRevision)
		t.Run("OrdinaryOperationDoesNotBump", testS49PGOrdinaryOperation)
		t.Run("OverflowRollsBackBusinessWrite", testS49PGOverflow)
	})
	t.Run("MetadataConstraintsAndLegacyUpgrade", testS49PGConstraints)
}

type s49PG struct {
	t      *testing.T
	ctx    context.Context
	pool   *pgxpool.Pool
	store  *RecoveryRunStore
	schema string
	trace  *s49SQLTrace
}

type s49SQLTrace struct {
	mu  sync.Mutex
	sql []string
}

func (x *s49SQLTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.sql = append(x.sql, d.SQL)
	return ctx
}
func (*s49SQLTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func (x *s49SQLTrace) clear()                                                        { x.mu.Lock(); defer x.mu.Unlock(); x.sql = nil }
func (x *s49SQLTrace) has(fragment string) bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	for _, s := range x.sql {
		if strings.Contains(strings.ToLower(s), fragment) {
			return true
		}
	}
	return false
}

func s49PGHarness(t *testing.T, version int) *s49PG {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	c, err := pgxpool.ParseConfig(os.Getenv("HAN_TEST_POSTGRES_DSN"))
	if err != nil || c.ConnConfig.Host != "127.0.0.1" || c.ConnConfig.User != "han_b13c" || c.ConnConfig.Database != "han_b13c_identity" || c.ConnConfig.Port == 0 {
		t.Fatal("explicit isolated identity PostgreSQL configuration required")
	}
	schema := "han_b13c_s49_" + s49Random(t)
	c.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	trace := &s49SQLTrace{}
	c.ConnConfig.Tracer = trace
	c.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		t.Fatal("open isolated identity PostgreSQL")
	}
	t.Cleanup(pool.Close)
	s49PGPreflight(t, ctx, pool, c)
	h := &s49PG{t: t, ctx: ctx, pool: pool, schema: schema, trace: trace}
	h.exec("CREATE SCHEMA " + pgx.Identifier{schema}.Sanitize())
	t.Logf("retained_schema=%s", schema)
	h.migrate(version)
	h.store, err = NewRecoveryRunStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func s49PGPreflight(t *testing.T, ctx context.Context, pool *pgxpool.Pool, c *pgxpool.Config) {
	t.Helper()
	var host, user, db string
	var port, version int
	err := pool.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int,current_user,current_database()`).Scan(&host, &port, &version, &user, &db)
	if err != nil || host != c.ConnConfig.Host || port != int(c.ConnConfig.Port) || version != 180006 || user != c.ConnConfig.User || db != c.ConnConfig.Database {
		t.Fatal("isolated PostgreSQL preflight failed before DDL")
	}
}

func s49Random(t *testing.T) string {
	t.Helper()
	var b [12]byte
	if _, e := rand.Read(b[:]); e != nil {
		t.Fatal("synthetic fixture ID")
	}
	return hex.EncodeToString(b[:])
}

func (h *s49PG) exec(sql string, args ...any) pgconn.CommandTag {
	h.t.Helper()
	tag, e := h.pool.Exec(h.ctx, sql, args...)
	if e != nil {
		h.t.Fatalf("synthetic SQL failed: %v", e)
	}
	return tag
}

func (h *s49PG) migrate(version int) {
	h.t.Helper()
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql"}
	input := fstest.MapFS{}
	for _, name := range names[:version] {
		b, e := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if e != nil {
			h.t.Fatal(e)
		}
		input["migrations/"+name] = &fstest.MapFile{Data: b}
	}
	items, e := migration.Load(input, "migrations")
	if e != nil {
		h.t.Fatal(e)
	}
	for _, item := range items {
		tx, e := h.pool.Begin(h.ctx)
		if e != nil {
			h.t.Fatal(e)
		}
		func() {
			defer finishAdmissionRead(h.ctx, tx)
			_, e = tx.Exec(h.ctx, item.SQL)
			if e == nil {
				e = tx.Commit(h.ctx)
			}
		}()
		if e != nil {
			h.t.Fatalf("migration %04d: %v", item.Version, e)
		}
	}
}

func (h *s49PG) install() {
	h.exec(`INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,installation_id,target_digest,commitment_ref,commitment_digest,
	 recovery_journal_id,recovery_epoch,recovery_initial_sequence,recovery_initial_hash,article_journal_id,article_epoch,article_initial_sequence,article_initial_hash,revision,installed_at)
	 VALUES(true,'han-recovery-install/v1','deployment','installation',$1,'commitment<&>',$1,'recovery',1,0,'','article',2,0,'',1,$2)`, s49Hash, s49Time)
}

func (h *s49PG) run(id string) recovery.RecoveryRun {
	r := recovery.RecoveryRun{RunID: id, BackupRef: "backup", JournalKeyID: "key", JournalKeyVersion: "v1", JournalSecretFingerprint: s49HMAC, JournalStage: recovery.StageBegun, StartedAt: s49Time, Outcome: recovery.RecoveryOutcomeRunning}
	if e := h.store.Create(h.ctx, r); e != nil {
		h.t.Fatal(e)
	}
	return r
}

func (h *s49PG) snapshot() recovery.AdmissionSnapshot {
	h.t.Helper()
	s, e := h.store.ReadAdmissionSnapshot(h.ctx)
	if e != nil {
		h.t.Fatal(e)
	}
	return s
}

func (h *s49PG) revision() int64 {
	h.t.Helper()
	var n int64
	if e := h.pool.QueryRow(h.ctx, "SELECT revision FROM recovery_installation").Scan(&n); e != nil {
		h.t.Fatal(e)
	}
	return n
}

func testS49PGProjection(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.exec(`UPDATE recovery_runs SET account_count=1,permanent_credential_count=2,session_count=3,temporary_credential_count=4,idempotency_count=5,pending_review_count=6 WHERE run_id='run'`)
	h.trace.clear()
	s := h.snapshot()
	want := s49Snapshot()
	if !h.trace.has("begin isolation level repeatable read read only") {
		t.Fatal("public Store did not start actual RR READ ONLY transaction")
	}
	if len(s.Runs) != 1 || !reflect.DeepEqual(s.Runs[0], want.Runs[0]) || !reflect.DeepEqual(s.Installation, want.Installation) || s.Revision.Counter != 3 {
		t.Fatalf("incomplete actual projection: %#v", s)
	}
	r, e := h.store.ReadAdmissionRevision(h.ctx)
	if e != nil || r != s.Revision || h.revision() != 3 {
		t.Fatalf("revision read mismatch/write: %v", e)
	}
	if !strings.HasPrefix(r.Digest, "sha256:") {
		t.Fatal("missing canonical digest")
	}
}

// Keep explicit cause classification available to all real-PG negative assertions.
func s49PGErrorCode(err error, code string) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && p.Code == code
}

func (h *s49PG) user(id string) {
	h.exec(`INSERT INTO users(id,username,email,phone,role,status,credential_mode,created_at,updated_at)
	 VALUES($1,$2,$3,$4,'member','suspended','recovery_locked',$5,$5)`, id, "synthetic-"+id, id+"@example.invalid", "+synthetic-"+id, s49Time)
}

func (h *s49PG) audit() {
	h.exec(`INSERT INTO audit_events(id,occurred_at,actor_kind,actor_system,action,target_type,target_id,outcome,trace_id)
	 VALUES($1,$2,'system','synthetic','recovery','run','run','succeeded','synthetic-trace')`, s49Audit, s49Time)
}

func (h *s49PG) preparedProof() {
	h.user(s49User)
	h.audit()
	h.exec(`UPDATE recovery_runs SET initial_admin_user_id=$1,initial_admin_bootstrap_generation=2 WHERE run_id='run'`, s49User)
}

func (h *s49PG) changedProof() {
	h.preparedProof()
	h.exec(`UPDATE recovery_runs SET initial_admin_changed_generation=3,initial_admin_change_audit_id=$1,initial_admin_changed_at=$2 WHERE run_id='run'`, s49Audit, s49Time)
}

func (h *s49PG) restore() {
	h.exec(`INSERT INTO recovery_restore_bindings(run_id,backup_ref,installation_id,deployment_id,target_digest,artifact_digest,
	 source_approval_digest,restored_schema_digest,receipt_digest,artifact_bytes,restored_at,receipt_version)
	 VALUES('run','backup','installation','deployment',$1,$1,$1,$1,$1,123,$2,'han-db-restore/v1')`, s49Hash, s49Time)
}

func testS49PGProofProjection(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.changedProof()
	h.restore()
	s := h.snapshot()
	r := s.Runs[0]
	if !reflect.DeepEqual(r.RestoreBinding, s49Receipt()) || r.InitialAdminUserID == nil || *r.InitialAdminUserID != s49User || *r.InitialAdminChangedGeneration != 3 || *r.InitialAdminChangeAuditID != s49Audit || !r.InitialAdminChangedAt.Equal(s49Time) {
		t.Fatalf("lost source: %#v", r)
	}
	h.exec(`UPDATE recovery_runs SET outcome='identity_ready' WHERE run_id='run'`)
	v := recovery.ReadyReplayStatusV1{EvidenceVersion: recovery.ReplayEvidenceVersionV1, RunID: "run", BackupRef: "backup", Namespace: recovery.ArticleDeletionNamespaceV1, EvidenceRef: "evidence", EvidenceDigest: s49Hash, JournalHeadSequence: 1, JournalHeadHash: s49Hash, CompletedAt: s49Time.Format("2006-01-02T15:04:05.000000Z")}
	p := recovery.PublicationBinding{Version: "article-replay-publication/v1", RunID: "run", BackupRef: "backup", Ref: "publication", Digest: s49Hash, EvidenceRef: v.EvidenceRef, EvidenceDigest: s49Hash, JournalID: "article", Epoch: 2, Head: recovery.JournalHead{Sequence: 1, Hash: s49Hash}, EvidenceCompletedAt: s49Time, PublishedAt: s49Time}
	_, e := h.store.Transition(h.ctx, "run", recovery.RecoveryOutcomeIdentityReady, recovery.RecoveryOutcomeArticleDeletionReady, recovery.RecoveryRunChange{ArticleEvidence: &v, ArticlePublication: &p, ArticleEvidenceCompletedAt: s49Time, ChangedAt: s49Time})
	if e != nil {
		t.Fatal(e)
	}
	loaded, e := h.store.Load(h.ctx, "run")
	if e != nil || loaded.ArticlePublicationRef != p.Ref || loaded.ArticlePublicationDigest != p.Digest {
		t.Fatalf("publication ACK/readback: %v", e)
	}
	r = h.snapshot().Runs[0]
	if r.ArticlePublicationRef == nil || *r.ArticlePublicationRef != p.Ref || r.ArticleEvidenceVersion == nil {
		t.Fatal("publication absent from full snapshot")
	}
}

func testS49PGLegacyFingerprint(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	r := recovery.RecoveryRun{RunID: "legacy", BackupRef: "backup", JournalKeyID: "key", JournalKeyVersion: "v1", JournalSecretFingerprint: "synthetic-legacy", JournalStage: recovery.StageBegun, StartedAt: s49Time, Outcome: recovery.RecoveryOutcomeRunning}
	if e := h.store.Create(h.ctx, r); e != nil {
		t.Fatal("old Create contract changed", e)
	}
	s, e := h.store.ReadAdmissionSnapshot(h.ctx)
	if e == nil || s.Runs != nil {
		t.Fatal("new admission accepted legacy fingerprint")
	}
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	locked, e := h.store.LockRecoveryRunForDeletion(h.ctx, tx, "legacy")
	if e != nil || locked.JournalSecretFingerprint != r.JournalSecretFingerprint {
		t.Fatalf("old lock-only scanner changed: %v", e)
	}
}

func testS49PGRR(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("z")
	tx, e := h.pool.BeginTx(h.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	a, e := readAdmissionTransaction(h.ctx, tx)
	if e != nil {
		t.Fatal(e)
	}
	h.run("a") // A second pool connection really commits while the RR transaction is alive.
	b, e := readAdmissionTransaction(h.ctx, tx)
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("RR observed partial/new set: %v", e)
	}
	if e = tx.Commit(h.ctx); e != nil {
		t.Fatal(e)
	}
	c := h.snapshot()
	if len(c.Runs) != 2 || c.Runs[0].RunID != "a" || c.Revision.Counter != a.Revision.Counter+1 || c.Revision.Digest == a.Revision.Digest {
		t.Fatal("new snapshot lost independently committed row")
	}
	var pid int32
	var isolation, readonly string
	conn, e := h.pool.Acquire(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Release()
	if e = conn.QueryRow(h.ctx, `SELECT pg_backend_pid(),current_setting('transaction_isolation'),current_setting('transaction_read_only')`).Scan(&pid, &isolation, &readonly); e != nil {
		t.Fatal(e)
	}
	t.Logf("independent_backend=%d retained_schema=%s", pid, h.schema)
}

type s49QueryErrorTx struct{ pgx.Tx }

func (x s49QueryErrorTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if sql != admissionRunsSQL {
		return nil, errors.New("unexpected query")
	}
	// Actual PostgreSQL arithmetic failure is delivered by pgx.Rows.Err, not a fake DB result.
	return x.Tx.Query(ctx, `SELECT projected.* FROM (`+sql+`) projected WHERE 1/(length(projected.journal_key_id)-length(projected.journal_key_id))=1`, args...)
}

func testS49PGRowsError(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	tx, e := h.pool.BeginTx(h.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	s, e := readAdmissionTransaction(h.ctx, s49QueryErrorTx{tx})
	if !s49PGErrorCode(e, "22012") || s.Runs != nil {
		t.Fatalf("native rows error became empty/partial success: %v", e)
	}
	if h.revision() != 2 {
		t.Fatal("read error changed revision")
	}
}

func testS49PGMissing(t *testing.T) {
	h := s49PGHarness(t, 9)
	s, e := h.store.ReadAdmissionSnapshot(h.ctx)
	if !s49PGErrorCode(e, "42P01") || !errors.Is(e, recovery.ErrAdmissionIncomplete) || s.Runs != nil {
		t.Fatalf("missing schema became empty: %v", e)
	}
}

func testS49PGCancel(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	ctx, cancel := context.WithCancel(h.ctx)
	cancel()
	_, e := h.store.ReadAdmissionSnapshot(ctx)
	if !errors.Is(e, context.Canceled) || h.revision() != 2 {
		t.Fatalf("cancel lost or wrote: %v", e)
	}
	if _, e = h.store.ReadAdmissionSnapshot(nil); !errors.Is(e, recovery.ErrAdmissionIncomplete) {
		t.Fatal("nil ctx accepted")
	}
	if h.pool.Stat().AcquiredConns() != 0 {
		t.Fatal("canceled read leaked connection")
	}
	r := recovery.RecoveryRun{RunID: "cancelled", BackupRef: "backup", JournalKeyID: "key", JournalKeyVersion: "v1", JournalSecretFingerprint: s49HMAC, JournalStage: recovery.StageBegun, StartedAt: s49Time, Outcome: recovery.RecoveryOutcomeRunning}
	if e = h.store.Create(ctx, r); !errors.Is(e, context.Canceled) {
		t.Fatal("Create lost cancellation", e)
	}
	_, e = h.store.Transition(ctx, "run", recovery.RecoveryOutcomeRunning, recovery.RecoveryOutcomeFailed, recovery.RecoveryRunChange{ChangedAt: s49Time})
	if !errors.Is(e, context.Canceled) || h.revision() != 2 || h.pool.Stat().AcquiredConns() != 0 {
		t.Fatal("cancelled writer advanced or leaked", e)
	}
}

func testS49PGRevision(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	if h.revision() != 1 {
		t.Fatal("initial revision")
	}
	h.run("run")
	if h.revision() != 2 {
		t.Fatal("run insert")
	}
	r := recovery.RecoveryRun{RunID: "run", BackupRef: "backup", JournalKeyID: "key", JournalKeyVersion: "v1", JournalSecretFingerprint: s49HMAC, JournalStage: recovery.StageBegun, StartedAt: s49Time, Outcome: recovery.RecoveryOutcomeRunning}
	if e := h.store.Create(h.ctx, r); !errors.Is(e, recovery.ErrRunAlreadyExists) || h.revision() != 2 {
		t.Fatalf("duplicate mapping/counter: %v", e)
	}
	h.exec(`UPDATE recovery_runs SET outcome='failed' WHERE run_id='run'`)
	if h.revision() != 3 {
		t.Fatal("failed update missing")
	}
	h.exec(`UPDATE recovery_runs SET account_count=account_count WHERE run_id='run'`)
	if h.revision() != 4 {
		t.Fatal("same-state actual write missing")
	}
	h.exec(`UPDATE recovery_installation SET installation_id=installation_id`)
	if h.revision() != 5 {
		t.Fatal("installation write recursive/missing")
	}
	h.restore()
	if h.revision() != 6 {
		t.Fatal("restore insert missing")
	}
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	_, e = tx.Exec(h.ctx, `UPDATE recovery_runs SET session_count=7 WHERE run_id='run'`)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Rollback(h.ctx); e != nil {
		t.Fatal(e)
	}
	if h.revision() != 6 {
		t.Fatal("rollback advanced revision")
	}
	_, e = h.store.ReadAdmissionRevision(h.ctx)
	if e != nil || h.revision() != 6 {
		t.Fatal("readonly incremented")
	}
}

const s49Operation = "33333333-3333-4333-8333-333333333333"
const s49Credential = "44444444-4444-4444-8444-444444444444"

func (h *s49PG) review() {
	h.user(s49User)
	h.exec(`INSERT INTO recovery_account_reviews(recovery_run_id,user_id,backup_role,backup_status,decision) VALUES('run',$1,'member','suspended','pending')`, s49User)
}

func (h *s49PG) operation(recoveryBound bool) {
	h.t.Helper()
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		h.t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	if e = h.seedOperation(tx, recoveryBound); e != nil {
		h.t.Fatal(e)
	}
	if e = tx.Commit(h.ctx); e != nil {
		h.t.Fatal(e)
	}
}

func (h *s49PG) seedOperation(tx pgx.Tx, bound bool) error {
	approval := []any{nil, nil, nil, nil, nil, nil, nil}
	if bound {
		approval = []any{"run", "installation", "approval", s49Hash, s49User, "admin", "active"}
	}
	return h.seedOperationWithApproval(tx, approval)
}

func (h *s49PG) seedOperationWithApproval(tx pgx.Tx, approval []any) error {
	_, e := tx.Exec(h.ctx, `UPDATE users SET password_reset_execution_generation=1,password_reset_current_operation_id=$2 WHERE id=$1`, s49User, s49Operation)
	if e != nil {
		return e
	}
	args := append([]any{s49Operation, s49User, s49Credential, s49Time}, approval...)
	_, e = tx.Exec(h.ctx, `INSERT INTO password_reset_operations(operation_id,user_id,temporary_credential_id,scope_hash,key_locator,key_digest,request_digest,digest_key_version,
	 execution_generation,request_kind,base_auth_generation,target_auth_generation,settlement_state,delivery_state,state,created_at,updated_at,
	 recovery_run_id,recovery_installation_id,recovery_evidence_ref,recovery_evidence_digest,recovery_actor_id,recovery_target_role,recovery_target_status)
	 VALUES($1,$2,$3,decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),'synthetic-v1',
	 1,'initial_reset',0,1,'pending','not_started','prepared',$4,$4,$5,$6,$7,$8,$9,$10,$11)`, args...)
	if e != nil {
		return e
	}
	_, e = tx.Exec(h.ctx, `INSERT INTO temporary_credentials(id,user_id,purpose,generation,auth_generation,source_operation_id,state,secret_digest,delivered_to_email,issued_at,expires_at)
	 VALUES($1,$2,'password_reset',1,1,$3,'pending_delivery','synthetic-digest','synthetic@example.invalid',$4,$4::timestamptz+interval '1 day')`, s49Credential, s49User, s49Operation, s49Time)
	if e != nil {
		return e
	}
	_, e = tx.Exec(h.ctx, `INSERT INTO password_reset_recovery_jobs(operation_id,status,attempt_count,next_attempt_at,created_at,updated_at) VALUES($1,'ready',0,$2,$2,$2)`, s49Operation, s49Time)
	if e != nil {
		return e
	}
	_, e = tx.Exec(h.ctx, `INSERT INTO idempotency_records(scope_hash,key_locator,key_digest,request_digest,digest_key_version,status,execution_generation,password_reset_operation_id,password_reset_user_id,locked_until,expires_at,created_at,updated_at)
	 VALUES(decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),'synthetic-v1','processing',1,$1,$2,$3::timestamptz+interval '1 hour',$3::timestamptz+interval '1 day',$3,$3)`, s49Operation, s49User, s49Time)
	if e == nil && approval[0] != nil {
		_, e = tx.Exec(h.ctx, `UPDATE recovery_account_reviews SET restoration_operation_id=$1 WHERE recovery_run_id='run' AND user_id=$2`, s49Operation, s49User)
	}
	return e
}

func testS49PGOperationRevision(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.review()
	if h.revision() != 3 {
		t.Fatal("review insert missing")
	}
	h.operation(true)
	if h.revision() != 5 {
		t.Fatal("operation/review pointer counter missing or recursive")
	}
	h.exec(`UPDATE password_reset_operations SET updated_at=updated_at+interval '1 second' WHERE operation_id=$1`, s49Operation)
	if h.revision() != 6 {
		t.Fatal("recovery operation actual write missing")
	}
	h.exec(`UPDATE recovery_account_reviews SET decision='denied',reviewed_by=$1,reviewed_at=$2 WHERE recovery_run_id='run'`, s49User, s49Time)
	if h.revision() != 7 {
		t.Fatal("review terminal write missing")
	}
	before := h.revision()
	_, e := h.pool.Exec(h.ctx, `UPDATE recovery_account_reviews SET restoration_operation_id=NULL WHERE recovery_run_id='run'`)
	if e == nil || h.revision() != before {
		t.Fatal("terminal pointer mutable")
	}
}

func testS49PGOrdinaryOperation(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.user(s49User)
	h.operation(false)
	if h.revision() != 1 {
		t.Fatal("ordinary operation bumped recovery revision")
	}
	h.exec(`UPDATE password_reset_operations SET updated_at=updated_at+interval '1 second' WHERE operation_id=$1`, s49Operation)
	if h.revision() != 1 {
		t.Fatal("ordinary operation mutation bumped recovery revision")
	}
}

func testS49PGOverflow(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	// Only synthetic fixture preparation bypasses the successor guard; it is restored before the observed write.
	h.exec(`ALTER TABLE recovery_installation DISABLE TRIGGER recovery_installation_immutable`)
	h.exec(`UPDATE recovery_installation SET revision=9223372036854775807`)
	h.exec(`ALTER TABLE recovery_installation ENABLE TRIGGER recovery_installation_immutable`)
	var enabled string
	if e := h.pool.QueryRow(h.ctx, `SELECT tgenabled::text FROM pg_trigger WHERE tgrelid='recovery_installation'::regclass AND tgname='recovery_installation_immutable'`).Scan(&enabled); e != nil || enabled != "O" {
		t.Fatal("overflow guard not enabled")
	}
	_, e := h.pool.Exec(h.ctx, `UPDATE recovery_runs SET account_count=77 WHERE run_id='run'`)
	var n int64
	_ = h.pool.QueryRow(h.ctx, `SELECT account_count FROM recovery_runs WHERE run_id='run'`).Scan(&n)
	if !s49PGErrorCode(e, "22003") || n != 0 || h.revision() != int64(9223372036854775807) {
		t.Fatalf("overflow did not rollback: %v", e)
	}
}

func testS49PGConstraints(t *testing.T) {
	t.Run("LegacyUpgradeDoesNotInventSuccess", testS49PGLegacyUpgrade)
	t.Run("ScopeFourBindingMatrix", testS49PGSessionConstraints)
	t.Run("ProofPairAndMonotonicity", testS49PGProofConstraints)
	t.Run("OperationApprovalAndCompositeForeignKeys", testS49PGOperationConstraints)
	t.Run("ImmutableInstallationRestorePublication", testS49PGImmutable)
	t.Run("InvalidApprovalAndSourceForeignKeys", testS49PGApprovalFK)
	t.Run("MigrationRollbackAtomic", testS49PGMigrationRollback)
}

func testS49PGLegacyUpgrade(t *testing.T) {
	h := s49PGHarness(t, 9)
	h.run("run")
	h.user(s49User)
	h.exec(`INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,authenticated_credential_kind,issued_at,last_seen_at,absolute_expires_at,idle_expires_at)
	 VALUES($1,$2,decode('aa','hex'),decode('bb','hex'),'permanent',$3,$3,$3::timestamptz+interval '1 day',$3::timestamptz+interval '1 hour')`, s49Credential, s49User, s49Time)
	for _, outcome := range []string{"identity_ready", "article_deletion_ready", "ready_to_consume"} {
		h.exec(`UPDATE recovery_runs SET outcome=$1 WHERE run_id='run'`, outcome)
	}
	h.exec(`UPDATE recovery_runs SET outcome='journal_consumed',journal_stage='consumed',journal_consumed_at=$1 WHERE run_id='run'`, s49Time)
	h.exec(`UPDATE recovery_runs SET outcome='succeeded',completed_at=$1 WHERE run_id='run'`, s49Time)
	h.migrateOnly0010()
	var pub, proof *string
	var scope string
	var run, installation, runtime *string
	var generation *int64
	if e := h.pool.QueryRow(h.ctx, `SELECT access_scope,recovery_run_id,recovery_installation_id,recovery_runtime_id,recovery_generation FROM sessions WHERE id=$1`, s49Credential).Scan(&scope, &run, &installation, &runtime, &generation); e != nil || scope != "ordinary" || run != nil || installation != nil || runtime != nil || generation != nil {
		t.Fatal("legacy ordinary session changed")
	}
	if e := h.pool.QueryRow(h.ctx, `SELECT article_publication_ref,initial_admin_user_id::text FROM recovery_runs WHERE run_id='run'`).Scan(&pub, &proof); e != nil || pub != nil || proof != nil {
		t.Fatal("legacy success fabricated")
	}
	_, e := h.store.ReadAdmissionSnapshot(h.ctx)
	if e == nil {
		t.Fatal("unbound legacy database admitted")
	}
	h.install()
	s := h.snapshot()
	if s.Runs[0].RestoreBinding != nil || s.Runs[0].InitialAdminUserID != nil || s.Runs[0].ArticlePublicationRef != nil {
		t.Fatal("migration fabricated provenance")
	}
}

func (h *s49PG) migrateOnly0010() {
	h.t.Helper()
	b, e := os.ReadFile(filepath.Join("..", "..", "migrations", "0010_recovery_runtime_metadata.sql"))
	if e != nil {
		h.t.Fatal(e)
	}
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		h.t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	if _, e = tx.Exec(h.ctx, string(b)); e == nil {
		e = tx.Commit(h.ctx)
	}
	if e != nil {
		h.t.Fatal(e)
	}
}

func (h *s49PG) rejected(sql string, args ...any) {
	h.t.Helper()
	before := h.revision()
	_, e := h.pool.Exec(h.ctx, sql, args...)
	if e == nil || h.revision() != before {
		h.t.Fatalf("invalid metadata write accepted or bumped counter: %v", e)
	}
}

func testS49PGSessionConstraints(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.user(s49User)
	base := `INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,authenticated_credential_kind,issued_at,last_seen_at,absolute_expires_at,idle_expires_at,
	 access_scope,recovery_run_id,recovery_installation_id,recovery_runtime_id,recovery_generation)
	 VALUES($1,$2,decode($3,'hex'),decode('bb','hex'),'permanent',$4,$4,$4::timestamptz+interval '1 day',$4::timestamptz+interval '1 hour',$5,$6,$7,$8,$9)`
	h.exec(base, s49Credential, s49User, "aa", s49Time, "ordinary", nil, nil, nil, nil)
	for i := 0; i < 4; i++ {
		a := []any{"run", "installation", "runtime", int64(1)}
		a[i] = nil
		h.rejected(base, s49Operation, s49User, "cc", s49Time, "recovery_maintenance", a[0], a[1], a[2], a[3])
	}
	h.rejected(base, s49Operation, s49User, "cc", s49Time, "ordinary", "run", nil, nil, nil)
	h.rejected(base, s49Operation, s49User, "cc", s49Time, "recovery_maintenance", "run", "installation", "runtime", int64(0))
	h.exec(base, s49Operation, s49User, "cc", s49Time, "recovery_maintenance", "run", "installation", "runtime", int64(1))
}

func testS49PGProofConstraints(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.user(s49User)
	h.audit()
	h.rejected(`UPDATE recovery_runs SET initial_admin_user_id=$1 WHERE run_id='run'`, s49User)
	h.rejected(`UPDATE recovery_runs SET initial_admin_bootstrap_generation=2 WHERE run_id='run'`)
	h.rejected(`UPDATE recovery_runs SET initial_admin_changed_generation=3 WHERE run_id='run'`)
	h.rejected(`UPDATE recovery_runs SET initial_admin_user_id=$1,initial_admin_bootstrap_generation=9223372036854775807 WHERE run_id='run'`, s49User)
	h.exec(`UPDATE recovery_runs SET initial_admin_user_id=$1,initial_admin_bootstrap_generation=2 WHERE run_id='run'`, s49User)
	h.rejected(`UPDATE recovery_runs SET initial_admin_changed_generation=2,initial_admin_change_audit_id=$1,initial_admin_changed_at=$2 WHERE run_id='run'`, s49Audit, s49Time)
	h.exec(`UPDATE recovery_runs SET initial_admin_changed_generation=3,initial_admin_change_audit_id=$1,initial_admin_changed_at=$2 WHERE run_id='run'`, s49Audit, s49Time)
	h.rejected(`UPDATE recovery_runs SET initial_admin_changed_generation=NULL,initial_admin_change_audit_id=NULL,initial_admin_changed_at=NULL WHERE run_id='run'`)
	h.rejected(`UPDATE recovery_runs SET initial_admin_bootstrap_generation=3 WHERE run_id='run'`)
	h.exec(`INSERT INTO recovery_account_reviews(recovery_run_id,user_id,backup_role,backup_status,decision,reviewed_by,reviewed_at) VALUES('run',$1,'member','suspended','denied',$1,$2)`, s49User, s49Time)
	h.rejected(`DELETE FROM recovery_account_reviews WHERE recovery_run_id='run'`)
}

func testS49PGOperationConstraints(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.review()
	h.operation(true)
	for _, field := range []string{"recovery_run_id", "recovery_installation_id", "recovery_evidence_ref", "recovery_evidence_digest", "recovery_actor_id", "recovery_target_role", "recovery_target_status"} {
		h.rejected(`UPDATE password_reset_operations SET `+pgx.Identifier{field}.Sanitize()+`=NULL WHERE operation_id=$1`, s49Operation)
		_, e := h.pool.Exec(h.ctx, `INSERT INTO password_reset_operations SELECT (jsonb_populate_record(NULL::password_reset_operations,to_jsonb(o)||jsonb_build_object($1::text,NULL))).* FROM password_reset_operations o WHERE operation_id=$2`, field, s49Operation)
		if !s49PGErrorCode(e, "23514") {
			t.Fatalf("partial approval CHECK not enforced (%s): %v", field, e)
		}
	}
	h.rejected(`UPDATE password_reset_operations SET recovery_target_role='member' WHERE operation_id=$1`, s49Operation)
	h.rejected(`UPDATE recovery_account_reviews SET restoration_operation_id=NULL WHERE recovery_run_id='run'`)
	other := "55555555-5555-4555-8555-555555555555"
	h.user(other)
	h.exec(`INSERT INTO recovery_account_reviews(recovery_run_id,user_id,backup_role,backup_status,decision) VALUES('run',$1,'member','active','pending')`, other)
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	_, e = tx.Exec(h.ctx, `UPDATE recovery_account_reviews SET restoration_operation_id=$1 WHERE user_id=$2`, s49Operation, other)
	if e == nil {
		e = tx.Commit(h.ctx)
	}
	if !s49PGErrorCode(e, "23503") {
		t.Fatalf("cross-account pointer accepted: %v", e)
	}
	h.run("other")
	h.exec(`INSERT INTO recovery_account_reviews(recovery_run_id,user_id,backup_role,backup_status,decision) VALUES('other',$1,'member','active','pending')`, s49User)
	_, e = h.pool.Exec(h.ctx, `UPDATE recovery_account_reviews SET restoration_operation_id=$1 WHERE recovery_run_id='other'`, s49Operation)
	if !s49PGErrorCode(e, "23503") {
		t.Fatalf("same-account wrong-run pointer accepted: %v", e)
	}
}

func testS49PGConcurrentTransition(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	r := h.run("run")
	h.changedProof()
	results := make(chan error, 2)
	start := make(chan struct{})
	var joined sync.WaitGroup
	defer joined.Wait()
	for i := 0; i < 2; i++ {
		joined.Add(1)
		go func() {
			defer joined.Done()
			<-start
			_, e := h.store.Transition(h.ctx, r.RunID, r.Outcome, recovery.RecoveryOutcomeIdentityReady, recovery.RecoveryRunChange{ChangedAt: s49Time})
			results <- e
		}()
	}
	close(start)
	success, conflict := 0, 0
	for i := 0; i < 2; i++ {
		select {
		case e := <-results:
			if e == nil {
				success++
			} else if errors.Is(e, recovery.ErrRecoveryRunConflict) {
				conflict++
			} else {
				t.Error(e)
			}
		case <-h.ctx.Done():
			t.Fatal("bounded SQL transition failed to join")
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("CAS successes=%d conflicts=%d", success, conflict)
	}
}

func testS49PGImmutable(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.restore()
	h.rejected(`UPDATE recovery_installation SET installation_id='other'`)
	h.rejected(`UPDATE recovery_installation SET revision=1`)
	h.rejected(`DELETE FROM recovery_installation`)
	h.rejected(`UPDATE recovery_restore_bindings SET artifact_bytes=124 WHERE run_id='run'`)
	h.rejected(`DELETE FROM recovery_restore_bindings WHERE run_id='run'`)
	h.rejected(`UPDATE recovery_runs SET article_publication_ref='p' WHERE run_id='run'`)
	h.exec(`UPDATE recovery_runs SET article_publication_ref='p',article_publication_digest=$1 WHERE run_id='run'`, s49Hash)
	h.rejected(`UPDATE recovery_runs SET article_publication_ref='other' WHERE run_id='run'`)
	h.rejected(`UPDATE recovery_runs SET article_publication_ref=NULL,article_publication_digest=NULL WHERE run_id='run'`)
	h.rejected(`DELETE FROM recovery_runs WHERE run_id='run'`)
}

func testS49PGNullAndTime(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	r := h.run("run")
	if h.snapshot().Runs[0].ArticleEvidenceRef != nil {
		t.Fatal("SQL NULL not preserved")
	}
	h.exec(`UPDATE recovery_runs SET article_evidence_version='article-deletion-replay/v1',article_evidence_digest=$1,article_evidence_ref='',article_evidence_completed_at=$2,article_namespace='article-deletion/v1',article_replay_verified_at=$2 WHERE run_id='run'`, s49Hash, s49Time)
	s, e := h.store.ReadAdmissionSnapshot(h.ctx)
	if e == nil || s.Runs != nil {
		t.Fatal("SQL empty evidence became NULL/success")
	}
	n := h.revision()
	r.RunID = "precision"
	r.StartedAt = s49Time.Add(time.Nanosecond)
	if e = h.store.Create(h.ctx, r); !errors.Is(e, recovery.ErrInvalidRecoveryRun) || h.revision() != n {
		t.Fatal("sub-microsecond Create wrote")
	}
	r.StartedAt = s49Time.In(time.FixedZone("wrong", 3600))
	if e = h.store.Create(h.ctx, r); !errors.Is(e, recovery.ErrInvalidRecoveryRun) {
		t.Fatal("non-UTC Create accepted")
	}
}

type s49ReadAckTx struct {
	pgx.Tx
	cause error
}

func (x s49ReadAckTx) Commit(ctx context.Context) error {
	if e := x.Tx.Commit(ctx); e != nil {
		return e
	}
	return x.cause
}
func testS49PGReadAckLoss(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	tx, e := h.pool.BeginTx(h.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		t.Fatal(e)
	}
	cause := errors.New("synthetic lost read-transaction commit ACK")
	s, e := readOwnedAdmissionTransaction(h.ctx, s49ReadAckTx{tx, cause})
	if !errors.Is(e, cause) || !errors.Is(e, recovery.ErrAdmissionIncomplete) || s.Runs != nil || h.revision() != 2 || h.pool.Stat().AcquiredConns() != 0 {
		t.Fatalf("commit ACK loss leaked proof/resource: %v", e)
	}
}

func testS49PGApprovalFK(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	h.review()
	for _, tc := range []struct {
		at   int
		v    any
		code string
	}{{0, "missing", "23503"}, {1, "missing", "23503"}, {2, "", "23514"}, {3, "sha256:bad", "23514"}, {4, "55555555-5555-4555-8555-555555555555", "23503"}, {5, "owner", "23514"}, {6, "deleted", "23514"}} {
		tx, e := h.pool.Begin(h.ctx)
		if e != nil {
			t.Fatal(e)
		}
		values := []any{"run", "installation", "approval", s49Hash, s49User, "admin", "active"}
		values[tc.at] = tc.v
		func() {
			defer finishAdmissionRead(h.ctx, tx)
			e = h.seedOperationWithApproval(tx, values)
			if e == nil {
				e = tx.Commit(h.ctx)
			}
		}()
		if !s49PGErrorCode(e, tc.code) || h.revision() != 3 {
			t.Fatalf("approval field %d did not atomically reject: %v", tc.at, e)
		}
	}
}

func testS49PGMigrationRollback(t *testing.T) {
	h := s49PGHarness(t, 9)
	b, e := os.ReadFile(filepath.Join("..", "..", "migrations", "0010_recovery_runtime_metadata.sql"))
	if e != nil {
		t.Fatal(e)
	}
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	if _, e = tx.Exec(h.ctx, string(b)); e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(h.ctx, `SELECT 1/0`)
	if !s49PGErrorCode(e, "22012") {
		t.Fatal(e)
	}
	if e = tx.Rollback(h.ctx); e != nil {
		t.Fatal(e)
	}
	var table *string
	var columns int
	if e = h.pool.QueryRow(h.ctx, `SELECT to_regclass('recovery_installation')::text`).Scan(&table); e != nil || table != nil {
		t.Fatal("migration table survived rollback")
	}
	if e = h.pool.QueryRow(h.ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=$1 AND table_name='recovery_runs' AND column_name='article_publication_ref'`, h.schema).Scan(&columns); e != nil || columns != 0 {
		t.Fatal("partial ALTER survived rollback")
	}
}

func testS49PGWriteAckLoss(t *testing.T) {
	h := s49PGHarness(t, 10)
	h.install()
	h.run("run")
	tx, e := h.pool.Begin(h.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finishAdmissionRead(h.ctx, tx)
	if _, e = tx.Exec(h.ctx, `UPDATE recovery_runs SET account_count=99 WHERE run_id='run'`); e != nil {
		t.Fatal(e)
	}
	cause := errors.New("synthetic lost write-commit ACK")
	e = commitRecoveryRun(h.ctx, s49ReadAckTx{tx, cause}, "commit recovery run test")
	if !errors.Is(e, cause) || !errors.Is(e, recovery.ErrRecoveryWriteUnknown) {
		t.Fatal("write unknown/cause lost", e)
	}
	s := h.snapshot()
	if s.Runs[0].AccountCount != 99 || s.Revision.Counter != 3 || h.pool.Stat().AcquiredConns() != 0 {
		t.Fatal("committed ACK-unknown not independently visible or double-written")
	}
}

//go:build integration

package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/migration"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const s410Initial = "11111111-1111-4111-8111-111111111111"
const s410ND = "22222222-2222-4222-8222-222222222222"
const s410D = "33333333-3333-4333-8333-333333333333"

func TestRecoveryIdentityExecutionPostgres(t *testing.T) {
	t.Run("FinalizeOnceAndAtomicAudit", func(t *testing.T) {
		t.Run("BindVerifiedReceiptAndAuditAtomic", s410PGBind)
		t.Run("D0ND0FullCountsAndRetryAfterTrust", s410PGFinalize)
		t.Run("AuditFailureRollsBackAllEffects", s410PGRollback)
		t.Run("TrustHashOnceAndIndependentACK", s410PGTrustACK)
		t.Run("DeletionInstantAcrossSessionTimezones", s410PGDeletionInstant)
		t.Run("TrustHashCancellationTrueJoin", s410PGHashJoin)
		t.Run("CommitPanicPreservesValueAndReleases", s410PGCommitPanic)
	})
	t.Run("UserStoreReviewLockOrder", func(t *testing.T) {
		t.Run("UserLockPrecedesStoreLock", s410PGLocks)
		t.Run("PublicTypedRootConstructionAndSameExecutor", s410PGTypedConstruction)
		t.Run("CompleteDoesNotTakeUserLocks", s410PGCompleteLocks)
	})
	t.Run("KeptDeletedCanonicalAndDenied", func(t *testing.T) {
		for _, target := range []string{s410ND, s410D} {
			for _, decision := range []string{"denied", "kept_deleted"} {
				for _, articles := range []int{0, 2} {
					target, decision, articles := target, decision, articles
					t.Run(fmt.Sprintf("%s/%s/articles%d", target[:2], decision, articles), func(t *testing.T) { s410PGRestricted(t, target, decision, articles) })
				}
			}
		}
		t.Run("ArticleFailureCanonicalRollsBack", s410PGCanonicalRollback)
	})
	t.Run("IdempotencyUnknownAndCompleteProof", func(t *testing.T) {
		t.Run("MixedLawfulTerminalComplete", s410PGComplete)
		t.Run("RestoredND0FullPersistedAggregate", s410PGRestored)
		t.Run("RestoredGenericTerminalSnapshotIsNotProof", s410PGRestoredSnapshotReject)
		t.Run("D0RestoredLabelCannotUndelete", s410PGD0RestoredReject)
		for _, bad := range []string{"pending", "missing_review", "wrong_deleted_time", "missing_review_audit", "initial_digest", "unresolved_idempotency"} {
			bad := bad
			t.Run("Reject/"+bad, func(t *testing.T) { s410PGCompleteReject(t, bad) })
		}
		t.Run("IdempotencyFullIdentityInvalidation", s410PGIdempotency)
		t.Run("ImmutableFailedSourceAndCohort", s410PGFailedCohort)
		t.Run("FailedSourceRejectAndAtomicity", s410PGFailedReject)
		t.Run("FailedResolveACKAndTerminalRetry", s410PGFailedACK)
		t.Run("CompleteOriginalCohortRejectMatrix", s410PGCohortReject)
		t.Run("LinkedResetAuthorityRemainsOwned", s410PGLinkedReset)
		t.Run("OriginalRejectedFailureResolution", s410PGDirectRejected)
	})
	t.Run("SucceededRecoverySessionExitAtomic", func(t *testing.T) {
		t.Run("ExactReceiptAndPriorRevokedHistory", s410PGExitHistory)
		t.Run("ExpiredOldGenerationAndOrdinaryUnchanged", s410PGExit)
		t.Run("FirstZeroStillAuditAndRetryReadOnly", s410PGExitZero)
		t.Run("AuditWithResidualDoesNotRepair", s410PGExitResidual)
		t.Run("ExitAuditFailureRollsBack", s410PGExitRollback)
	})
	t.Run("RecoveryExitAuditAckReadback", func(t *testing.T) {
		t.Run("CommitThenACKIndependentPhysicalReader", s410PGExitACK)
		t.Run("RollbackThenACKUnknownNoRetry", s410PGExitUnknown)
		t.Run("BindingCommitACKNoSecondBind", s410PGBindACK)
		t.Run("FinalizeCommitACKNoSecondFreeze", s410PGFinalizeACK)
		t.Run("CompleteCommitACKNoSecondTransition", s410PGCompleteACK)
		t.Run("ScopedACKConcurrentUnrelatedAndRelatedFacts", s410PGScopedACK)
	})
}

type s410Clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *s410Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *s410Clock) advance()       { c.mu.Lock(); c.now = c.now.Add(time.Second); c.mu.Unlock() }

type s410IDs struct{}

func (s410IDs) NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

type s410Passwords struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (x *s410Passwords) Hash(s string) (string, error) {
	x.calls.Add(1)
	if x.entered != nil {
		close(x.entered)
		<-x.release
	}
	if s != "synthetic-secret" {
		return "", errors.New("unexpected synthetic secret")
	}
	return "synthetic-configured-hash", nil
}

type s410Audit struct {
	mu     sync.Mutex
	fail   string
	marker error
}

func (a *s410Audit) Append(c context.Context, tx pgx.Tx, event audit.Event) error {
	a.mu.Lock()
	fail, marker := a.fail, a.marker
	a.mu.Unlock()
	if event.Action == fail {
		return marker
	}
	return (audit.Store{}).Append(c, tx, event)
}
func (a *s410Audit) set(action string) { a.mu.Lock(); defer a.mu.Unlock(); a.fail = action }

type s410PG struct {
	t            *testing.T
	ctx          context.Context
	pool         *pgxpool.Pool
	store        *RecoveryRunStore
	schema       string
	clock        *s410Clock
	passwords    *s410Passwords
	audit        *s410Audit
	j            *s410Journal
	articles     ArticleAccountPort
	trace        *s410PGTrace
	failedReader RecoveryFailedEvidenceReader
}
type s410PGTrace struct {
	mu     sync.Mutex
	active bool
	pids   map[uint32]bool
}

func (x *s410PGTrace) TraceQueryStart(c context.Context, conn *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.active {
		x.pids[conn.PgConn().PID()] = true
	}
	return c
}
func (*s410PGTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func (x *s410PGTrace) start() {
	x.mu.Lock()
	x.active = true
	x.pids = map[uint32]bool{}
	x.mu.Unlock()
}
func (x *s410PGTrace) stop() map[uint32]bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.active = false
	out := map[uint32]bool{}
	for k, v := range x.pids {
		out[k] = v
	}
	return out
}
func s410PGHarness(t *testing.T) *s410PG {
	return s410PGHarnessTimezone(t, "")
}

func s410PGHarnessTimezone(t *testing.T, zone string) *s410PG {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	c, err := pgxpool.ParseConfig(dsn)
	if dsn == "" || err != nil || c.ConnConfig.Host != "127.0.0.1" || !strings.HasPrefix(c.ConnConfig.Database, "han_b13d_") || !strings.HasSuffix(c.ConnConfig.Database, "_identity") || !strings.HasPrefix(c.ConnConfig.User, "han_b13d") || c.ConnConfig.Port == 0 {
		t.Fatal("explicit new isolated B13D identity PG configuration required")
	}
	var random [10]byte
	if _, err = rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "han_b13d_s410_" + hex.EncodeToString(random[:])
	c.MaxConns = 4
	if zone != "" {
		c.ConnConfig.RuntimeParams["timezone"] = zone
	}
	c.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	trace := &s410PGTrace{}
	c.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		t.Fatal("open isolated identity pool")
	}
	t.Cleanup(pool.Close)
	var addr, db, user string
	var port, version int
	err = pool.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),current_database(),current_user,current_setting('server_version_num')::int`).Scan(&addr, &port, &db, &user, &version)
	if err != nil || addr != "127.0.0.1" || port != int(c.ConnConfig.Port) || db != c.ConnConfig.Database || user != c.ConnConfig.User || version != 180006 {
		t.Fatal("isolated PG preflight failed before DDL")
	}
	h := &s410PG{t: t, ctx: ctx, pool: pool, schema: schema, clock: &s410Clock{now: s410Time.Add(time.Minute)}, passwords: &s410Passwords{}, audit: &s410Audit{marker: errors.New("synthetic audit failure")}, j: &s410Journal{}, trace: trace}
	h.exec(`CREATE SCHEMA ` + pgx.Identifier{schema}.Sanitize())
	t.Logf("retained_schema=%s", schema)
	// Register immediately after creation, including fixtures which later fail.
	t.Cleanup(func() { s410PhysicalRetention(t, []string{schema}) })
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql"}
	fs := fstest.MapFS{}
	for _, name := range names {
		raw, e := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if e != nil {
			t.Fatal(e)
		}
		fs["migrations/"+name] = &fstest.MapFile{Data: raw}
	}
	migrations, err := migration.Load(fs, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		func() {
			tx, e := pool.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer finishAdmissionRead(ctx, tx)
			if _, e = tx.Exec(ctx, m.SQL); e == nil {
				e = tx.Commit(ctx)
			}
			if e != nil {
				t.Fatalf("migration %d: %v", m.Version, e)
			}
		}()
	}
	h.install()
	h.users()
	h.store, err = NewRecoveryRunStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	service, err := article.NewCommandService(article.CommandDependencies{Transactions: pool, Repository: article.Repository{}, IDs: s410IDs{}, Clock: h.clock, Orphaning: &article.OrphanArticlesCanonicalDependencies{Repository: article.CanonicalOrphanRepository{}, Auditor: h.audit, AuditIDs: s410IDs{}, Clock: h.clock}})
	if err != nil {
		t.Fatal(err)
	}
	h.articles = ArticleAccountPortFunc(func(c context.Context, tx pgx.Tx, id string) (CanonicalDeleteArticleResult, error) {
		r, e := service.OrphanArticlesCanonical(c, tx, id)
		return CanonicalDeleteArticleResult{NewlyOrphanedCount: r.NewlyOrphanedCount, AlreadyOrphanedCount: r.AlreadyOrphanedCount, PendingOrphanCount: r.PendingOrphanCount}, e
	})
	return h
}
func (h *s410PG) exec(sql string, args ...any) pgconn.CommandTag {
	h.t.Helper()
	tag, e := h.pool.Exec(h.ctx, sql, args...)
	if e != nil {
		h.t.Fatalf("synthetic SQL: %v", e)
	}
	return tag
}
func (h *s410PG) count(sql string, args ...any) int64 {
	h.t.Helper()
	var n int64
	if e := h.pool.QueryRow(h.ctx, sql, args...).Scan(&n); e != nil {
		h.t.Fatal(e)
	}
	return n
}
func (h *s410PG) install() {
	i := s410Installation()
	h.exec(`INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,installation_id,target_digest,commitment_ref,commitment_digest,recovery_journal_id,recovery_epoch,recovery_initial_sequence,recovery_initial_hash,article_journal_id,article_epoch,article_initial_sequence,article_initial_hash,revision,installed_at) VALUES(true,$1,$2,$3,$4,$5,$6,$7,1,0,'',$8,2,1,$4,1,$9)`, i.Version, i.DeploymentID, i.InstallationID, i.TargetDigest, i.CommitmentRef, i.CommitmentDigest, i.RecoveryInitial.JournalID, i.ArticleInitial.JournalID, i.InstalledAt)
}
func (h *s410PG) users() {
	for _, v := range []struct {
		id, name, status string
		initial          bool
	}{{s410Initial, "initial", "active", true}, {s410ND, "member", "active", false}, {s410D, "deleted", "deleted", false}} {
		h.exec(`INSERT INTO users(id,username,email,phone,permanent_password_digest,role,is_initial_admin,status,credential_mode,recovery_required,retained_email,created_at,updated_at,deleted_at,auth_generation,permanent_password_generation) VALUES($1,$2,CASE WHEN $3='deleted' THEN NULL ELSE $2||'@example.invalid' END,CASE WHEN $3='deleted' THEN NULL ELSE '+123'||$2 END,CASE WHEN $3='deleted' THEN NULL ELSE 'synthetic-old' END,CASE WHEN $4 THEN 'admin' ELSE 'member' END,$4,$3,'permanent',false,CASE WHEN $3='deleted' THEN $2||'@example.invalid' ELSE NULL END,$5::timestamptz,$5::timestamptz,CASE WHEN $3='deleted' THEN $5::timestamptz ELSE NULL END,1,CASE WHEN $3='deleted' THEN NULL ELSE 1 END)`, v.id, v.name, v.status, v.initial, s410Time.Add(-time.Hour))
		for _, kind := range []string{"username", "email", "phone"} {
			value := v.name
			if kind == "email" {
				value += "@example.invalid"
			}
			if kind == "phone" {
				value = "+123" + v.name
			}
			life := "active"
			if v.status == "deleted" {
				life = "reserved"
			}
			h.exec(`INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at) VALUES($1,$2,$3,$4,$5)`, v.id, kind, value, life, s410Time.Add(-time.Hour))
		}
	}
}
func (h *s410PG) running() {
	h.j.append(recovery.StageBegun)
	if err := h.store.Create(h.ctx, recovery.RecoveryRun{RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, JournalKeyID: "s410-key", JournalKeyVersion: "K1", JournalSecretFingerprint: s410HMAC, JournalStage: recovery.StageBegun, StartedAt: s410Time, Outcome: recovery.RecoveryOutcomeRunning}); err != nil {
		h.t.Fatal(err)
	}
	x := s410Receipt()
	h.exec(`INSERT INTO recovery_restore_bindings(run_id,backup_ref,installation_id,deployment_id,artifact_digest,target_digest,receipt_digest,source_approval_digest,restored_schema_digest,artifact_bytes,restored_at,receipt_version) VALUES($1,$2,$3,$4,$5,$5,$5,$5,$5,1,$6,'han-db-restore/v1')`, x.RunID, x.BackupRef, x.InstallationID, x.DeploymentID, s410Hash, x.RestoredAt)
}
func (h *s410PG) phase(body func(context.Context, recovery.PhasePermit, *recoveryIdentityExecution)) {
	h.t.Helper()
	f := &recoveryExecutionForwarder{}
	rt, ctx, done := s410Runtime(h.t, h.store, h.store, h.j, f)
	defer done()
	e, err := newRecoveryIdentityExecution(recoveryExecutionDependencies{store: h.store, runtime: rt, journal: h.j, articles: h.articles, audit: h.audit, passwords: h.passwords, ids: s410IDs{}, clock: h.clock, failedEvidence: h.failedReader})
	if err != nil {
		h.t.Fatal(err)
	}
	called := 0
	err = f.seal(&s410Callbacks{bind: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding, x recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
		body(c, p, e)
		called++
		return recovery.RecoveryRun{}, s410Stop
	}, finalize: func(c context.Context, p recovery.PhasePermit, _ recovery.RunBinding) error {
		body(c, p, e)
		called++
		return s410Stop
	}})
	if err != nil {
		h.t.Fatal(err)
	}
	err = rt.Resume(ctx, s410Binding)
	if !errors.Is(err, s410Stop) || called != 1 {
		h.t.Fatalf("actual public phase producer: calls=%d err=%v", called, err)
	}
}
func (h *s410PG) finalized(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
	h.t.Helper()
	if err := e.FinalizeRestoredIdentity(c, p, s410Binding); err != nil {
		h.t.Fatal(err)
	}
}
func (h *s410PG) trusted(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
	h.finalized(c, p, e)
	if err := e.PrepareRecoveryTrust(c, p, s410Binding, []byte("synthetic-secret")); err != nil {
		h.t.Fatal(err)
	}
}

// Explicit legal persisted strong-change fixture: S4.11 HTTP/session flow is not
// implemented here. The exact existing password-change Audit/proof is consumed.
func (h *s410PG) changed() {
	h.t.Helper()
	h.clock.advance()
	now := h.clock.Now()
	id, _ := (s410IDs{}).NewID()
	tx, err := h.pool.Begin(h.ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	defer finishAdmissionRead(h.ctx, tx)
	_, err = tx.Exec(h.ctx, `UPDATE users SET credential_mode='permanent',recovery_required=false,auth_generation=auth_generation+1,permanent_password_generation=auth_generation+1,permanent_password_digest='synthetic-changed',updated_at=$2 WHERE id=$1`, s410Initial, now)
	if err == nil {
		actor, role := s410Initial, "admin"
		err = (audit.Store{}).Append(h.ctx, tx, audit.Event{ID: id, OccurredAt: now, ActorKind: audit.ActorUser, ActorUserID: &actor, ActorRole: &role, Action: passwordChangeAuditAction, TargetType: "user", TargetID: s410Initial, Outcome: "succeeded", TraceID: "00000000000000000000000001"})
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `UPDATE recovery_runs SET initial_admin_changed_generation=initial_admin_bootstrap_generation+1,initial_admin_change_audit_id=$2,initial_admin_changed_at=$3 WHERE run_id=$1`, s410Binding.RunID, id, now)
	}
	if err == nil {
		err = tx.Commit(h.ctx)
	}
	if err != nil {
		h.t.Fatal(err)
	}
	h.clock.advance()
}
func (h *s410PG) denied(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
	h.t.Helper()
	for _, id := range []string{s410ND, s410D} {
		if err := e.resolveRestricted(c, p, s410Binding, id, "denied", "synthetic-reviewed-approval"); err != nil {
			h.t.Fatal(err)
		}
	}
}
func (h *s410PG) rawUser(id string) json.RawMessage {
	var b []byte
	if err := h.pool.QueryRow(h.ctx, `SELECT to_jsonb(u)::text FROM users u WHERE id=$1`, id).Scan(&b); err != nil {
		h.t.Fatal(err)
	}
	return b
}
func (h *s410PG) auditCount(action string) int64 {
	return h.count(`SELECT count(*) FROM audit_events WHERE action=$1`, action)
}

func s410PGBind(t *testing.T) {
	h := s410PGHarness(t)
	h.j.append(recovery.StageBegun)
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		r, err := e.BindRestoredRun(c, p, s410Binding, s410Receipt())
		if err != nil || r.RunID != s410Binding.RunID {
			t.Fatalf("bind: %v", err)
		}
		if h.auditCount(recoveryBindAction) != 1 || h.count(`SELECT count(*) FROM recovery_restore_bindings`) != 1 {
			t.Fatal("bind aggregate incomplete")
		}
	})
}
func s410PGFinalize(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.oldAuthentication()
	var old recoveryExecutionUser
	if err := json.Unmarshal(h.rawUser(s410D), &old); err != nil {
		t.Fatal(err)
	}
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		before := string(h.rawUser(s410Initial))
		if err := e.FinalizeRestoredIdentity(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
		if string(h.rawUser(s410Initial)) != before {
			t.Fatal("retry overwrote prepared trust")
		}
		var got recoveryExecutionUser
		_ = json.Unmarshal(h.rawUser(s410D), &got)
		if got.Status != "deleted" || got.DeletedAt == nil || !got.DeletedAt.Equal(*old.DeletedAt) || got.Email != nil || got.Phone != nil || got.Retained == nil || *got.Retained != *old.Retained || !got.Required || got.Mode != "recovery_locked" {
			t.Fatal("D0 not preserved/frozen")
		}
		if h.count(`SELECT count(*) FROM recovery_account_reviews WHERE decision='pending'`) != 2 || h.auditCount(recoveryFinalizeAction) != 1 || h.count(`SELECT permanent_credential_count FROM recovery_runs`) != 2 || h.count(`SELECT session_count FROM recovery_runs`) != 1 || h.count(`SELECT temporary_credential_count FROM recovery_runs`) != 1 || h.count(`SELECT count(*) FROM temporary_credentials WHERE state='recovery_invalidated'`) != 1 {
			t.Fatal("full reviews/counts/audit incorrect")
		}
	})
}
func s410PGRollback(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.oldAuthentication()
	before := string(h.rawUser(s410ND))
	h.audit.set(recoveryFinalizeAction)
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		if err := e.FinalizeRestoredIdentity(c, p, s410Binding); !errors.Is(err, h.audit.marker) {
			t.Fatalf("audit cause lost: %v", err)
		}
		if h.count(`SELECT count(*) FROM recovery_account_reviews`) != 0 || string(h.rawUser(s410ND)) != before || h.count(`SELECT count(*) FROM sessions`) != 1 || h.count(`SELECT count(*) FROM temporary_credentials WHERE state='expired'`) != 1 {
			t.Fatal("partial finalization committed")
		}
	})
}
func s410PGTrustACK(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.finalized(c, p, e)
		commits := 0
		e.commit = func(c context.Context, tx pgx.Tx) error {
			commits++
			if err := tx.Commit(c); err != nil {
				return err
			}
			return errors.New("lost trust ACK")
		}
		if err := e.PrepareRecoveryTrust(c, p, s410Binding, []byte("synthetic-secret")); err != nil {
			t.Fatal(err)
		}
		if err := e.PrepareRecoveryTrust(c, p, s410Binding, nil); err != nil {
			t.Fatal(err)
		}
		if h.passwords.calls.Load() != 1 || commits != 1 {
			t.Fatal("trust retried hash or mutation")
		}
	})
}
func s410PGHashJoin(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.passwords.entered = make(chan struct{})
	h.passwords.release = make(chan struct{})
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.finalized(c, p, e)
		cc, cancel := context.WithCancel(c)
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- e.PrepareRecoveryTrust(cc, p, s410Binding, []byte("synthetic-secret")) }()
		select {
		case <-h.passwords.entered:
		case <-c.Done():
			close(h.passwords.release)
			<-done
			t.Fatal("hasher not entered")
		}
		cancel()
		select {
		case <-done:
			close(h.passwords.release)
			t.Fatal("detached hash on cancellation")
		default:
		}
		close(h.passwords.release)
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("hash cancel cause: %v", err)
		}
		if h.count(`SELECT count(*) FROM recovery_runs WHERE initial_admin_bootstrap_generation IS NOT NULL`) != 0 {
			t.Fatal("canceled hash wrote trust")
		}
	})
}

func s410PGLocks(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		block, err := h.pool.Begin(c)
		if err != nil {
			t.Fatal(err)
		}
		defer finishAdmissionRead(c, block)
		if _, err = block.Exec(c, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, s410ND); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- e.FinalizeRestoredIdentity(c, p, s410Binding) }()
		if err = s410WaitLock(c, h.pool); err != nil {
			_ = block.Rollback(c)
			<-done
			t.Fatal(err)
		}
		probe, err := h.pool.Begin(c)
		if err != nil {
			_ = block.Rollback(c)
			<-done
			t.Fatal(err)
		}
		_, err = probe.Exec(c, `SELECT run_id FROM recovery_runs WHERE run_id=$1 FOR UPDATE NOWAIT`, s410Binding.RunID)
		_ = probe.Rollback(c)
		_ = block.Rollback(c)
		joined := <-done
		if err != nil || joined != nil {
			t.Fatalf("User->Store lock order: probe=%v operation=%v", err, joined)
		}
	})
}
func s410WaitLock(ctx context.Context, pool *pgxpool.Pool) error {
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND pid<>pg_backend_pid()`).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
func s410PGCompleteLocks(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		h.denied(c, p, e)
		block, err := h.pool.Begin(c)
		if err != nil {
			t.Fatal(err)
		}
		defer finishAdmissionRead(c, block)
		if _, err = block.Exec(c, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, s410ND); err != nil {
			t.Fatal(err)
		}
		err = e.CompleteIdentityReview(c, p, s410Binding)
		_ = block.Rollback(c)
		if err != nil {
			t.Fatal(err)
		}
	})
}
func s410PGRestricted(t *testing.T, target, decision string, n int) {
	h := s410PGHarness(t)
	h.running()
	for i := 0; i < n; i++ {
		id, _ := (s410IDs{}).NewID()
		h.exec(`INSERT INTO articles(id,author_id,title,body,publication_state,violation_state,deletion_state,version,created_at,updated_at) VALUES($1,$2,'synthetic','synthetic','published','clear','active',1,$3,$3)`, id, target, s410Time)
	}
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		if err := e.resolveRestricted(c, p, s410Binding, target, decision, "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		before := h.auditCount(recoveryReviewAction)
		if err := e.resolveRestricted(c, p, s410Binding, target, decision, "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		if h.auditCount(recoveryReviewAction) != before {
			t.Fatal("terminal retry wrote audit")
		}
		if h.count(`SELECT count(*) FROM recovery_account_reviews WHERE user_id=$1 AND decision=$2 AND evidence_ref IS NOT NULL`, target, decision) != 1 {
			t.Fatal("review missing")
		}
		if decision == "kept_deleted" && h.count(`SELECT count(*) FROM articles WHERE author_id=$1`, target) != 0 {
			t.Fatal("real Article command did not orphan")
		}
		if decision == "denied" && h.auditCount(canonicalDeleteAuditAction) != 0 {
			t.Fatal("denied fabricated canonical success")
		}
	})
}
func s410PGCanonicalRollback(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.articles = ArticleAccountPortFunc(func(context.Context, pgx.Tx, string) (CanonicalDeleteArticleResult, error) {
		return CanonicalDeleteArticleResult{}, h.audit.marker
	})
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		before := string(h.rawUser(s410ND))
		if err := e.resolveRestricted(c, p, s410Binding, s410ND, "kept_deleted", "synthetic-reviewed-approval"); !errors.Is(err, h.audit.marker) {
			t.Fatal("Article failure cause missing")
		}
		if string(h.rawUser(s410ND)) != before || h.count(`SELECT count(*) FROM recovery_account_reviews WHERE decision='pending'`) != 2 {
			t.Fatal("partial canonical commit")
		}
	})
}

func s410PGComplete(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		if err := e.resolveRestricted(c, p, s410Binding, s410ND, "kept_deleted", "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		if err := e.resolveRestricted(c, p, s410Binding, s410D, "denied", "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		if err := e.CompleteIdentityReview(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
		if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='identity_ready' AND pending_review_count=0`) != 1 || h.j.consumes != 0 {
			t.Fatal("Complete did not conditionally advance or consumed")
		}
	})
}
func s410PGCompleteReject(t *testing.T, bad string) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		if bad == "missing_review_audit" {
			if err := e.resolveRestricted(c, p, s410Binding, s410ND, "denied", "synthetic-reviewed-approval"); err != nil {
				t.Fatal(err)
			}
			// Lawful schema terminal fixture with the required Audit deliberately absent.
			// No append-only event is deleted and no unrelated failure substitutes it.
			h.exec(`UPDATE recovery_account_reviews SET decision='denied',evidence_ref='synthetic-reviewed-approval',reviewed_by=$2,reviewed_at=$3 WHERE user_id=$1`, s410D, s410Initial, h.clock.Now())
		} else if bad != "pending" && bad != "missing_review" {
			h.denied(c, p, e)
		}
		switch bad {
		case "missing_review":
			h.exec(`DELETE FROM recovery_account_reviews WHERE user_id=$1`, s410D)
		case "wrong_deleted_time":
			h.exec(`UPDATE users SET deleted_at=deleted_at+interval '1 second' WHERE id=$1`, s410D)
		case "initial_digest":
			h.exec(`UPDATE users SET permanent_password_digest=NULL,permanent_password_generation=NULL WHERE id=$1`, s410Initial)
		case "unresolved_idempotency":
			h.unresolved()
		}
		if err := e.CompleteIdentityReview(c, p, s410Binding); err == nil {
			t.Fatalf("accepted incomplete proof %s", bad)
		}
		if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='running'`) != 1 {
			t.Fatal("failed Complete advanced")
		}
	})
}
func (h *s410PG) unresolved() {
	h.exec(`INSERT INTO idempotency_records(scope_hash,key_locator,key_digest,request_digest,digest_key_version,status,execution_generation,locked_until,expires_at,created_at,updated_at) VALUES(decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),decode(repeat('33',32),'hex'),decode(repeat('44',32),'hex'),'synthetic-v1','processing',1,$1::timestamptz+interval '1 hour',$1::timestamptz+interval '1 day',$1,$1)`, s410Time)
}
func s410PGIdempotency(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.unresolved()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.finalized(c, p, e)
		if h.count(`SELECT count(*) FROM idempotency_records WHERE status='recovery_review_required' AND key_digest=decode(repeat('33',32),'hex') AND request_digest=decode(repeat('44',32),'hex') AND execution_generation=1`) != 1 {
			t.Fatal("full identity transition missing")
		}
	})
}

func (h *s410PG) succeeded() {
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		h.denied(c, p, e)
		if err := e.CompleteIdentityReview(c, p, s410Binding); err != nil {
			h.t.Fatal(err)
		}
	})
	r := s410Ready()
	h.exec(`UPDATE recovery_runs SET outcome='article_deletion_ready',article_publication_ref='s410-publication',article_publication_digest=$2,article_evidence_version=$3,article_evidence_digest=$4,article_evidence_ref=$5,article_evidence_completed_at=$6,article_namespace=$7,article_replay_verified_at=$6 WHERE run_id=$1`, s410Binding.RunID, s410Hash, r.EvidenceVersion, r.EvidenceDigest, r.EvidenceRef, s410Time.Add(90*time.Second), r.Namespace)
	h.exec(`UPDATE recovery_runs SET outcome='ready_to_consume'`)
	h.j.append(recovery.StageConsumed)
	h.exec(`UPDATE recovery_runs SET outcome='journal_consumed',journal_stage='consumed',journal_consumed_at=$1`, s410Time.Add(120*time.Second))
	h.exec(`UPDATE recovery_runs SET outcome='succeeded',completed_at=$1`, s410Time.Add(121*time.Second))
	h.clock.mu.Lock()
	h.clock.now = s410Time.Add(123 * time.Second)
	h.clock.mu.Unlock()
}
func s410Ready() recovery.ReadyReplayStatusV1 {
	r := recovery.ReadyReplayStatusV1{EvidenceVersion: recovery.ReplayEvidenceVersionV1, RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, Namespace: recovery.ArticleDeletionNamespaceV1, JournalHeadSequence: 1, JournalHeadHash: s410Hash, InventoryScope: "recoverable-backups-before-cutoff", BackupCutoff: s410Time.Format(time.RFC3339Nano), InventoryDigest: s410Hash, DeletionOperationsDigest: s410Hash, AuditEventsDigest: s410Hash, Status: "ready", EvidenceRef: "s410-evidence", CompletedAt: s410Time.Add(90 * time.Second).Format(time.RFC3339Nano)}
	r.EvidenceDigest, _ = r.CanonicalDigest()
	return r
}
func (h *s410PG) exit(body func(context.Context, recovery.PhasePermit, *recoveryIdentityExecution)) {
	h.t.Helper()
	f := &recoveryExecutionForwarder{}
	ctx, cancel := context.WithTimeout(h.ctx, 20*time.Second)
	defer cancel()
	// Reuse the existing trace contract, not an arbitrary PhaseContext value.
	traced, err := platform.ContextWithTraceID(ctx, "00000000000000000000000001")
	if err != nil {
		h.t.Fatal(err)
	}
	x := &s410Exit{}
	rt, err := recovery.NewRuntime(traced, recovery.RuntimeDependencies{Store: h.store, Runs: h.store, Journal: h.j, ArticleSnapshots: h.j, Genesis: s410Genesis{}, Exclusive: &s410Exclusive{}, Lifecycle: s410Lifecycle{}, Closures: &s410Closures{ready: s410Ready()}, Identity: f, Restore: s410Restore{}, Consumer: s410Consumer{}, Exit: x, RecoverySecret: s410Secret{}})
	if err != nil {
		h.t.Fatal(err)
	}
	defer rt.Stop(traced)
	e, err := newRecoveryIdentityExecution(recoveryExecutionDependencies{store: h.store, runtime: rt, journal: h.j, articles: h.articles, audit: h.audit, passwords: h.passwords, ids: s410IDs{}, clock: h.clock})
	if err != nil {
		h.t.Fatal(err)
	}
	if err = f.seal(e); err != nil {
		h.t.Fatal(err)
	}
	called := 0
	x.fn = func(c context.Context, p recovery.PhasePermit, _ recovery.RunBinding) error {
		called++
		body(c, p, e)
		return s410Stop
	}
	if err = rt.Resume(traced, s410Binding); !errors.Is(err, s410Stop) || called != 1 {
		h.t.Fatalf("public succeeded replay: calls=%d err=%v", called, err)
	}
	if h.j.consumes != 0 {
		h.t.Fatal("Resume consumed again")
	}
}
func (h *s410PG) session(scope, runtime string, expired bool) {
	id, _ := (s410IDs{}).NewID()
	token, _ := (s410IDs{}).NewID()
	issued := h.clock.Now().Add(-2 * time.Hour)
	expires := h.clock.Now().Add(time.Hour)
	if expired {
		expires = h.clock.Now().Add(-time.Hour)
	}
	if scope == "ordinary" {
		h.exec(`INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,authenticated_credential_kind,auth_generation,issued_at,last_seen_at,absolute_expires_at,idle_expires_at) VALUES($1,$2,convert_to($3,'UTF8'),decode('bb','hex'),'permanent',3,$4,$4,$5,$5)`, id, s410Initial, token, issued, expires)
	} else {
		h.exec(`INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,authenticated_credential_kind,auth_generation,issued_at,last_seen_at,absolute_expires_at,idle_expires_at,access_scope,recovery_run_id,recovery_installation_id,recovery_runtime_id,recovery_generation) VALUES($1,$2,convert_to($3,'UTF8'),decode('bb','hex'),'permanent',3,$4,$4,$5,$5,'recovery_maintenance',$6,'s410-install',$7,1)`, id, s410Initial, token, issued, expires, s410Binding.RunID, runtime)
	}
}
func s410PGExit(t *testing.T) {
	h := s410PGHarness(t)
	h.succeeded()
	h.session("ordinary", "", false)
	h.session("maintenance", "old-runtime", true)
	h.session("maintenance", "new-runtime", false)
	before := string(h.rawUser(s410Initial))
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		if err := e.RevokeRecoverySessions(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
		if h.count(`SELECT count(*) FROM sessions WHERE access_scope='recovery_maintenance' AND revoked_at IS NULL`) != 0 || h.count(`SELECT count(*) FROM sessions WHERE access_scope='ordinary' AND revoked_at IS NULL`) != 1 || before != string(h.rawUser(s410Initial)) {
			t.Fatal("exit result or unrelated state changed")
		}
	})
}
func s410PGExitZero(t *testing.T) {
	h := s410PGHarness(t)
	h.succeeded()
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		for i := 0; i < 2; i++ {
			if err := e.RevokeRecoverySessions(c, p, s410Binding); err != nil {
				t.Fatal(err)
			}
		}
		if h.auditCount(recoveryExitAction) != 1 {
			t.Fatal("first zero/retry audit incorrect")
		}
	})
}
func s410PGExitResidual(t *testing.T) {
	h := s410PGHarness(t)
	h.succeeded()
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		if err := e.RevokeRecoverySessions(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
		h.session("maintenance", "late-invalid-issuer", false)
		if err := e.RevokeRecoverySessions(c, p, s410Binding); err == nil {
			t.Fatal("Audit+residual repaired/accepted")
		}
		if h.count(`SELECT count(*) FROM sessions WHERE revoked_at IS NULL`) != 1 {
			t.Fatal("residual was mutated")
		}
	})
}
func s410PGExitRollback(t *testing.T) {
	h := s410PGHarness(t)
	h.succeeded()
	h.session("maintenance", "old", false)
	h.audit.set(recoveryExitAction)
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		if err := e.RevokeRecoverySessions(c, p, s410Binding); !errors.Is(err, h.audit.marker) {
			t.Fatalf("exit audit cause: %v", err)
		}
		if h.count(`SELECT count(*) FROM sessions WHERE revoked_at IS NULL`) != 1 {
			t.Fatal("audit failure partially revoked")
		}
	})
}
func s410ACK(e *recoveryIdentityExecution, commit bool, calls *int) {
	e.commit = func(c context.Context, tx pgx.Tx) error {
		(*calls)++
		if commit {
			if err := tx.Commit(c); err != nil {
				return err
			}
		} else {
			if err := tx.Rollback(c); err != nil {
				return err
			}
		}
		return errors.New("synthetic lost commit ACK")
	}
}
func s410PGExitACK(t *testing.T) {
	h := s410PGHarness(t)
	h.succeeded()
	h.session("maintenance", "old", false)
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		calls := 0
		var writer uint32
		e.commit = func(c context.Context, tx pgx.Tx) error {
			calls++
			writer = tx.Conn().PgConn().PID()
			if err := tx.Commit(c); err != nil {
				return err
			}
			h.trace.start()
			return errors.New("synthetic lost exit ACK")
		}
		err := e.RevokeRecoverySessions(c, p, s410Binding)
		readers := h.trace.stop()
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 || h.auditCount(recoveryExitAction) != 1 {
			t.Fatal("ACK retried mutation or incomplete audit")
		}
		if len(readers) != 1 || readers[writer] {
			t.Fatalf("independent physical verifier not observed: writer=%d readers=%v", writer, readers)
		}
	})
}
func s410PGExitUnknown(t *testing.T) {
	h := s410PGHarness(t)
	h.succeeded()
	h.session("maintenance", "old", false)
	h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		calls := 0
		s410ACK(e, false, &calls)
		if err := e.RevokeRecoverySessions(c, p, s410Binding); !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
			t.Fatalf("rollback ACK not unknown: %v", err)
		}
		if calls != 1 || h.auditCount(recoveryExitAction) != 0 || h.count(`SELECT count(*) FROM sessions WHERE revoked_at IS NULL`) != 1 {
			t.Fatal("unknown retried/changed state")
		}
	})
}
func s410PGBindACK(t *testing.T) {
	h := s410PGHarness(t)
	h.j.append(recovery.StageBegun)
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		calls := 0
		s410ACK(e, true, &calls)
		if _, err := e.BindRestoredRun(c, p, s410Binding, s410Receipt()); err != nil {
			t.Fatal(err)
		}
		if calls != 1 || h.auditCount(recoveryBindAction) != 1 {
			t.Fatal("bind ACK not exact")
		}
	})
}
func s410PGFinalizeACK(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		calls := 0
		s410ACK(e, true, &calls)
		h.finalized(c, p, e)
		if calls != 1 || h.auditCount(recoveryFinalizeAction) != 1 {
			t.Fatal("finalize ACK replayed")
		}
	})
}
func s410PGCompleteACK(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		h.denied(c, p, e)
		calls := 0
		s410ACK(e, true, &calls)
		if err := e.CompleteIdentityReview(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
		if calls != 1 || h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='identity_ready'`) != 1 {
			t.Fatal("Complete ACK not exact")
		}
	})
}

// This is a lawful persisted F002 aggregate, NOT execution of S4.12/SMTP.
// Every original FK/CHECK/immutable trigger remains enabled during fixture setup.
func (h *s410PG) restoredFixture(target string, invalidSnapshot ...bool) {
	h.t.Helper()
	op, _ := (s410IDs{}).NewID()
	cred, _ := (s410IDs{}).NewID()
	attempt, _ := (s410IDs{}).NewID()
	auditID, _ := (s410IDs{}).NewID()
	tx, err := h.pool.Begin(h.ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	defer finishAdmissionRead(h.ctx, tx)
	now := h.clock.Now()
	result, outcome, resource := "password_reset_succeeded", "succeeded", op
	if len(invalidSnapshot) > 0 && invalidSnapshot[0] {
		result, outcome, resource = "reset", "reset", target
	}
	_, err = tx.Exec(h.ctx, `UPDATE users SET password_reset_execution_generation=1,password_reset_current_operation_id=$2,auth_generation=2,credential_mode='temporary',permanent_password_digest=NULL,permanent_password_generation=NULL,recovery_required=false,status='suspended',updated_at=$3 WHERE id=$1`, target, op, now)
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO password_reset_operations(operation_id,user_id,temporary_credential_id,scope_hash,key_locator,key_digest,request_digest,digest_key_version,execution_generation,request_kind,base_auth_generation,target_auth_generation,settlement_state,settled_auth_generation,settled_at,delivery_state,delivery_attempt_id,delivery_attempted_at,delivery_result_recorded_at,state,created_at,updated_at,terminal_at,recovery_run_id,recovery_installation_id,recovery_evidence_ref,recovery_evidence_digest,recovery_actor_id,recovery_target_role,recovery_target_status)
        VALUES($1,$2,$3,decode(repeat('55',32),'hex'),decode(repeat('66',32),'hex'),decode(repeat('77',32),'hex'),decode(repeat('88',32),'hex'),'synthetic-v1',1,'initial_reset',1,2,'temporary_active',2,$4,'accepted',$5,$4,$4,'succeeded',$4,$4,$4,$6,'s410-install','synthetic-reviewed-approval',$7,$8,'member','suspended')`, op, target, cred, now, attempt, s410Binding.RunID, s410Hash, s410Initial)
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO temporary_credentials(id,user_id,purpose,generation,auth_generation,source_operation_id,state,secret_digest,delivered_to_email,issued_at,expires_at) VALUES($1,$2,'password_reset',1,2,$3,'active','synthetic-new-temporary','member@example.invalid',$4,$4::timestamptz+interval '1 day')`, cred, target, op, now)
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO password_reset_recovery_jobs(operation_id,status,attempt_count,next_attempt_at,created_at,updated_at,completed_at) VALUES($1,'succeeded',1,$2,$2,$2,$2)`, op, now)
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO idempotency_records(scope_hash,key_locator,key_digest,request_digest,digest_key_version,status,execution_generation,password_reset_operation_id,password_reset_user_id,result_code,response_snapshot,expires_at,created_at,updated_at) VALUES(decode(repeat('55',32),'hex'),decode(repeat('66',32),'hex'),decode(repeat('77',32),'hex'),decode(repeat('88',32),'hex'),'synthetic-v1','succeeded',1,$1::uuid,$2::uuid,$4,jsonb_build_object('outcome',$5::text,'resource_id',$6::text),$3::timestamptz+interval '1 day',$3,$3)`, op, target, now, result, outcome, resource)
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `UPDATE recovery_account_reviews SET decision='restored',evidence_ref='synthetic-reviewed-approval',reviewed_by=$3,reviewed_at=$4,credential_reset_at=$4,restoration_operation_id=$2 WHERE user_id=$1 AND recovery_run_id=$5`, target, op, s410Initial, now, s410Binding.RunID)
	}
	if err == nil {
		system := "password-reset-worker"
		err = (audit.Store{}).Append(h.ctx, tx, audit.Event{ID: auditID, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &system, Action: "identity.password_reset.recovered", TargetType: "password_reset_operation", TargetID: op, Outcome: "succeeded", TraceID: "00000000000000000000000001", Metadata: map[string]any{"attempt_count": 1}})
	}
	if err == nil {
		actor, role := s410Initial, "admin"
		id, _ := (s410IDs{}).NewID()
		err = (audit.Store{}).Append(h.ctx, tx, audit.Event{ID: id, OccurredAt: now, ActorKind: audit.ActorUser, ActorUserID: &actor, ActorRole: &role, Action: recoveryReviewAction, TargetType: "user", TargetID: target, Outcome: "succeeded", TraceID: "00000000000000000000000001", Metadata: map[string]any{"run_id": s410Binding.RunID, "backup_ref": s410Binding.BackupRef, "installation_id": "s410-install", "decision": "restored", "evidence_ref": "synthetic-reviewed-approval", "actor_id": s410Initial}})
	}
	if err == nil {
		err = tx.Commit(h.ctx)
	}
	if err != nil {
		h.t.Fatalf("legal persisted F002 fixture: %v", err)
	}
}
func s410PGRestored(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		h.restoredFixture(s410ND)
		if err := e.resolveRestricted(c, p, s410Binding, s410D, "denied", "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		if err := e.CompleteIdentityReview(c, p, s410Binding); err != nil {
			t.Fatal(err)
		}
		if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='identity_ready'`) != 1 || h.count(`SELECT count(*) FROM users WHERE id=$1 AND status='suspended' AND NOT recovery_required AND credential_mode='temporary'`, s410ND) != 1 {
			t.Fatal("trusted suspended restored aggregate not accepted/preserved")
		}
	})
}

func s410PGD0RestoredReject(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		if err := e.resolveRestricted(c, p, s410Binding, s410ND, "denied", "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		// A schema-legal but domain-illegal terminal label must not be proof.
		h.exec(`UPDATE recovery_account_reviews SET decision='restored',evidence_ref='synthetic-reviewed-approval',reviewed_by=$2,reviewed_at=$3,credential_reset_at=$3 WHERE user_id=$1`, s410D, s410Initial, h.clock.Now())
		tx, err := h.pool.Begin(c)
		if err != nil {
			t.Fatal(err)
		}
		err = e.event(c, tx, recoveryReviewAction, s410D, s410Binding, "s410-install", map[string]any{"decision": "restored", "evidence_ref": "synthetic-reviewed-approval", "actor_id": s410Initial})
		if err == nil {
			err = tx.Commit(c)
		}
		_ = finishAdmissionRead(c, tx)
		if err != nil {
			t.Fatal(err)
		}
		if err = e.CompleteIdentityReview(c, p, s410Binding); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("D0 restored source not rejected: %v", err)
		}
		if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='running'`) != 1 || h.count(`SELECT count(*) FROM users WHERE id=$1 AND status='deleted'`, s410D) != 1 {
			t.Fatal("D0 label revived or opened flow")
		}
	})
}
func s410PGCommitPanic(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	marker := &struct{ message string }{"synthetic commit port panic"}
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		e.commit = func(c context.Context, tx pgx.Tx) error {
			if err := tx.Commit(c); err != nil {
				return err
			}
			panic(marker)
		}
		var got any
		func() { defer func() { got = recover() }(); _ = e.FinalizeRestoredIdentity(c, p, s410Binding) }()
		if got != marker {
			t.Fatal("original panic identity changed/swallowed")
		}
		if h.pool.Stat().AcquiredConns() != 0 {
			t.Fatal("panic leaked transaction connection")
		}
		if h.auditCount(recoveryFinalizeAction) != 1 {
			t.Fatal("commit port did not really commit once")
		}
	})
}

func (h *s410PG) oldAuthentication() {
	sessionID, _ := (s410IDs{}).NewID()
	credentialID, _ := (s410IDs{}).NewID()
	h.exec(`INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,authenticated_credential_kind,auth_generation,issued_at,last_seen_at,absolute_expires_at,idle_expires_at) VALUES($1,$2,decode('aa','hex'),decode('bb','hex'),'permanent',1,$3,$3,$3::timestamptz+interval '1 day',$3::timestamptz+interval '1 hour')`, sessionID, s410Initial, s410Time)
	// A retained expired older-generation credential is not current authority.
	h.exec(`INSERT INTO temporary_credentials(id,user_id,purpose,generation,auth_generation,state,secret_digest,delivered_to_email,issued_at,expires_at) VALUES($1,$2,'new_administrator',1,0,'expired','synthetic-old-temporary','member@example.invalid',$3::timestamptz-interval '3 hours',$3::timestamptz-interval '1 hour')`, credentialID, s410ND, s410Time)
}

// Every case below uses the original Store.Begin/Recover and Audit append in
// its actual database transaction, with an independently sealed synthetic
// checkpoint and an actual public Runtime.Resume-produced identity permit.
func (h *s410PG) seedFailure(kind string) (idempotency.BeginRequest, recoveryIdempotencyMember, RecoveryFailedEvidence) {
	h.t.Helper()
	req, m, src := s410SyntheticFailure(h.t, kind)
	tx, err := h.pool.Begin(h.ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	defer finishAdmissionRead(h.ctx, tx)
	begin, err := (idempotency.Store{}).Begin(h.ctx, tx, req)
	if err != nil || begin.Action != idempotency.BeginExecute {
		h.t.Fatalf("real original Store.Begin: %v action=%s", err, begin.Action)
	}
	actual := recoveryMember(idempotency.Transition{ScopeHash: begin.Record.ScopeHash, KeyLocator: begin.Record.KeyLocator, KeyDigest: begin.Record.KeyDigest, RequestDigest: begin.Record.RequestDigest, DigestKeyVersion: begin.Record.DigestKeyVersion, ExecutionGeneration: begin.Record.ExecutionGeneration}, begin.Record.CreatedAt, begin.Record.ExpiresAt)
	if !reflect.DeepEqual(actual, m) {
		h.t.Fatal("independent checkpoint is not bound to actual original request/six fields")
	}
	if err = tx.Commit(h.ctx); err != nil {
		h.t.Fatal(err)
	}
	return req, m, src
}
func (h *s410PG) idempotencyRows() []byte {
	h.t.Helper()
	var raw []byte
	if err := h.pool.QueryRow(h.ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(i) ORDER BY scope_hash,key_locator),'[]'::jsonb)::text FROM idempotency_records i`).Scan(&raw); err != nil {
		h.t.Fatal(err)
	}
	return raw
}
func s410PGFailedCohort(t *testing.T) {
	for _, kind := range []string{"administrator409", "administrator503", "resend"} {
		t.Run(kind, func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			_, m, src := h.seedFailure(kind)
			reader := s410CheckpointReader(src)
			h.failedReader = reader
			if kind == "resend" {
				h.exec(`UPDATE users SET role='admin' WHERE id=$1`, s410ND)
			}
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				h.changed()
				h.denied(c, p, e)
				var cohort []byte
				if err := h.pool.QueryRow(c, `SELECT metadata->'idempotency_cohort' FROM audit_events WHERE action=$1`, recoveryFinalizeAction).Scan(&cohort); err != nil {
					t.Fatal(err)
				}
				var original []recoveryIdempotencyMember
				if err := json.Unmarshal(cohort, &original); err != nil || !reflect.DeepEqual(original, []recoveryIdempotencyMember{m}) {
					t.Fatalf("exact original six-field cohort lost: %v", err)
				}
				tr := m.transition()
				if err := e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef); err != nil {
					t.Fatal(err)
				}
				before := h.idempotencyRows()
				audits := h.auditCount(recoveryIdempotencyAction)
				commits := 0
				originalCommit := e.commit
				e.commit = func(c context.Context, tx pgx.Tx) error { commits++; return originalCommit(c, tx) }
				h.clock.advance()
				if err := e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef); err != nil {
					t.Fatal(err)
				}
				if commits != 0 || audits != 1 || h.auditCount(recoveryIdempotencyAction) != 1 || !bytes.Equal(before, h.idempotencyRows()) {
					t.Fatal("terminal retry wrote, audited, or changed original actor/clock/lease/TTL")
				}
				// The real original domain consumer replays exactly this failure.
				tx, err := h.pool.Begin(c)
				if err != nil {
					t.Fatal(err)
				}
				record, err := recoveryReadIdempotency(c, tx, m, false)
				_ = finishAdmissionRead(c, tx)
				if err != nil {
					t.Fatal(err)
				}
				r := idempotency.Record{Status: record.Status, ResultCode: *record.ResultCode, ResponseSnapshot: &src.Recovery.ResponseSnapshot, CreatedAt: m.CreatedAt}
				var replay error
				if kind == "resend" {
					_, replay = replayAdministratorCredentialResend(idempotency.BeginResult{Action: idempotency.BeginReplay, Record: r}, s410ND)
				} else {
					_, replay = (&AdministratorCreateCoordinator{}).replayCreateAdministrator(r)
				}
				if platform.PublicCode(replay) != platform.Code(src.Recovery.ResultCode) {
					t.Fatalf("existing domain replay: %v", replay)
				}
				if err = e.CompleteIdentityReview(c, p, s410Binding); err != nil {
					t.Fatal(err)
				}
				if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='identity_ready'`) != 1 || h.j.consumes != 0 {
					t.Fatal("nonzero fully resolved original cohort did not Complete safely")
				}
				if reader.calls < 5 {
					t.Fatal("source was not independently re-read for commit/retry/Complete")
				}
			})
		})
	}
}
func s410PGFailedReject(t *testing.T) {
	cases := []string{"missing_provider", "wrong_actor", "wrong_install", "wrong_run", "wrong_backup", "wrong_scope", "wrong_six", "wrong_key_version", "generic_failure", "future_approval", "clock_rollback", "lost_permit", "reader_failure", "source_drift_after_commit", "audit_failure"}
	for _, bad := range cases {
		t.Run(bad, func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			_, m, src := h.seedFailure("administrator409")
			base := s410CheckpointReader(src)
			calls := 0
			h.failedReader = s410ReadFailedFunc(func(c context.Context, b recovery.PermitBinding, ref string, s, k []byte, g int64) (RecoveryFailedEvidence, error) {
				calls++
				x, err := base.ReadFailed(c, b, ref, s, k, g)
				if err != nil {
					return x, err
				}
				switch bad {
				case "wrong_actor":
					x.ActorID = s410ND
				case "wrong_install":
					x.InstallationID = "other"
				case "wrong_run":
					x.RunID = "other"
				case "wrong_backup":
					x.BackupRef = "other"
				case "wrong_scope":
					x.Scope[0] = '9'
				case "wrong_six":
					x.Recovery.RequestDigest[0] ^= 1
				case "wrong_key_version":
					x.Recovery.DigestKeyVersion = "new-active-K2"
				case "generic_failure":
					x.Recovery.ResponseSnapshot = idempotency.ResponseSnapshot{Outcome: "failed"}
				case "future_approval":
					x.ApprovedAt = s410Time.Add(24 * time.Hour)
				case "reader_failure":
					return x, errors.New("immutable source unavailable")
				case "source_drift_after_commit":
					if calls > 1 {
						x.EvidenceDigest = strings.Repeat("f", 64)
					}
				}
				return x, nil
			})
			if bad == "missing_provider" {
				h.failedReader = nil
			}
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				h.changed()
				before := h.idempotencyRows()
				tr := m.transition()
				if bad == "audit_failure" {
					h.audit.set(recoveryIdempotencyAction)
				}
				if bad == "clock_rollback" {
					h.clock.mu.Lock()
					h.clock.now = s410Time.Add(21 * time.Second)
					h.clock.mu.Unlock()
				}
				permit := p
				if bad == "lost_permit" {
					permit = recovery.PhasePermit{}
				}
				err := e.resolveIdempotency(c, permit, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef)
				if err == nil {
					t.Fatal("invalid or unavailable evidence accepted")
				}
				if bad == "source_drift_after_commit" {
					if !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || h.auditCount(recoveryIdempotencyAction) != 1 {
						t.Fatalf("late drift did not preserve committed-unknown: %v", err)
					}
				} else if !bytes.Equal(before, h.idempotencyRows()) || h.auditCount(recoveryIdempotencyAction) != 0 {
					t.Fatal("rejected input or Audit failure left a partial resolution")
				}
				if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='running'`) != 1 {
					t.Fatal("failure opened recovery")
				}
			})
		})
	}
}
func s410PGFailedACK(t *testing.T) {
	for _, applied := range []bool{true, false} {
		name := "rolled_back_unknown"
		if applied {
			name = "committed_exact"
		}
		t.Run(name, func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			_, m, src := h.seedFailure("administrator503")
			h.failedReader = s410CheckpointReader(src)
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				h.changed()
				marker := errors.New("lost failed-resolution ACK")
				calls := 0
				var writer uint32
				e.commit = func(c context.Context, tx pgx.Tx) error {
					calls++
					writer = tx.Conn().PgConn().PID()
					var err error
					if applied {
						err = tx.Commit(c)
					} else {
						err = tx.Rollback(c)
					}
					if err != nil {
						return err
					}
					h.trace.start()
					return marker
				}
				tr := m.transition()
				err := e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef)
				pids := h.trace.stop()
				if calls != 1 {
					t.Fatal("unknown ACK retried writer")
				}
				if len(pids) == 0 || pids[writer] {
					t.Fatalf("ACK not read by distinct physical same-pool connection: writer=%d readers=%v", writer, pids)
				}
				if applied {
					if err != nil || h.auditCount(recoveryIdempotencyAction) != 1 {
						t.Fatalf("exact committed postimage rejected: %v", err)
					}
					if err = e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef); err != nil || calls != 1 {
						t.Fatalf("terminal retry wrote: %v calls=%d", err, calls)
					}
				} else {
					if !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || !errors.Is(err, marker) || h.auditCount(recoveryIdempotencyAction) != 0 || h.count(`SELECT count(*) FROM idempotency_records WHERE status='recovery_review_required'`) != 1 {
						t.Fatalf("unknown lost cause or guessed commit: %v", err)
					}
				}
			})
		})
	}
}
func s410PGCohortReject(t *testing.T) {
	for _, bad := range []string{"mixed_unresolved", "missing_original", "changed_snapshot", "missing_resolution_audit", "duplicate_resolution_audit", "source_unavailable", "source_ref_conflict", "new_unresolved_record", "clock_rollback_after_resolution"} {
		t.Run(bad, func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			_, m, src := h.seedFailure("administrator409")
			base := s410CheckpointReader(src)
			var unavailable bool
			h.failedReader = s410ReadFailedFunc(func(c context.Context, b recovery.PermitBinding, ref string, s, k []byte, g int64) (RecoveryFailedEvidence, error) {
				if unavailable {
					return RecoveryFailedEvidence{}, errors.New("retained immutable source unavailable")
				}
				return base.ReadFailed(c, b, ref, s, k, g)
			})
			if bad == "mixed_unresolved" {
				h.seedFailure("administrator503")
			}
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				h.changed()
				h.denied(c, p, e)
				tr := m.transition()
				if bad == "missing_resolution_audit" || bad == "changed_snapshot" {
					// A schema-legal terminal fixture is insufficient by itself;
					// the consumer must require the immutable source AND atomic Audit.
					tx, err := h.pool.Begin(c)
					if err != nil {
						t.Fatal(err)
					}
					write := recoveryCopyFailed(src).Recovery
					write.ReviewedBy = s410Initial
					write.Now = h.clock.Now()
					if bad == "changed_snapshot" {
						write.ResponseSnapshot, err = administratorFailureSnapshot(platform.CodeIdentityEmailDeliveryFailed)
						write.ResultCode = string(platform.CodeIdentityEmailDeliveryFailed)
						if err != nil {
							t.Fatal(err)
						}
					}
					err = (idempotency.Store{}).Recover(c, tx, write)
					if err == nil && bad == "changed_snapshot" {
						err = e.appendResolution(c, tx, recoveryResolutionFor(src, m, write.Now, s410Time.Add(time.Minute)))
					}
					if err == nil {
						err = tx.Commit(c)
					}
					_ = finishAdmissionRead(c, tx)
					if err != nil {
						t.Fatal(err)
					}
				} else if bad != "missing_original" {
					if err := e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef); err != nil {
						t.Fatal(err)
					}
				}
				switch bad {
				case "missing_original":
					h.exec(`DELETE FROM idempotency_records WHERE scope_hash=$1 AND key_locator=$2`, tr.ScopeHash, tr.KeyLocator)
				case "duplicate_resolution_audit":
					tx, err := h.pool.Begin(c)
					if err != nil {
						t.Fatal(err)
					}
					a, err := recoveryReadResolution(c, tx, s410Binding, "s410-install", m)
					if err == nil {
						err = e.appendResolution(c, tx, a)
					}
					if err == nil {
						err = tx.Commit(c)
					}
					_ = finishAdmissionRead(c, tx)
					if err != nil {
						t.Fatal(err)
					}
				case "source_unavailable":
					unavailable = true
				case "source_ref_conflict":
					if err := e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef+"-different"); err == nil {
						t.Fatal("conflicting terminal intent accepted")
					}
					unavailable = true
				case "new_unresolved_record":
					h.unresolved()
				case "clock_rollback_after_resolution":
					h.clock.mu.Lock()
					h.clock.now = s410Time.Add(30 * time.Second)
					h.clock.mu.Unlock()
				}
				before := h.idempotencyRows()
				if err := e.CompleteIdentityReview(c, p, s410Binding); err == nil {
					t.Fatal("incomplete or altered original cohort accepted")
				}
				if !bytes.Equal(before, h.idempotencyRows()) || h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='running'`) != 1 {
					t.Fatal("Complete repaired evidence or advanced on missing original facts")
				}
			})
		})
	}
}

// Persisted original-domain aggregate, not execution of SMTP/recovery worker.
// The existing classifiers, deferred FKs and immutable triggers are retained.
func (h *s410PG) linkedResetFixture(state string) (recoveryIdempotencyMember, RecoveryFailedEvidence) {
	h.t.Helper()
	req, m, src := s410SyntheticFailure(h.t, "reset")
	op := src.Recovery.ResponseSnapshot.ResourceID
	cred, _ := (s410IDs{}).NewID()
	attempt, _ := (s410IDs{}).NewID()
	now := s410Time
	tx, err := h.pool.Begin(h.ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	defer finishAdmissionRead(h.ctx, tx)
	_, err = tx.Exec(h.ctx, `UPDATE users SET password_reset_execution_generation=1,password_reset_current_operation_id=$2 WHERE id=$1`, s410ND, op)
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO password_reset_operations(operation_id,user_id,temporary_credential_id,scope_hash,key_locator,key_digest,request_digest,digest_key_version,execution_generation,request_kind,base_auth_generation,target_auth_generation,settlement_state,settled_at,delivery_state,delivery_attempt_id,delivery_attempted_at,delivery_result_recorded_at,state,terminal_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,1,'initial_reset',1,2,CASE WHEN $10='invalidated' THEN 'invalidated' ELSE 'pending' END,CASE WHEN $10='invalidated' THEN $9::timestamptz ELSE NULL END,CASE WHEN $10='dispatching' THEN 'dispatching' ELSE 'unknown' END,$11,$9,CASE WHEN $10='dispatching' THEN NULL ELSE $9::timestamptz END,CASE WHEN $10='invalidated' THEN 'invalidated' WHEN $10='dispatching' THEN 'dispatching' ELSE 'commit_unknown' END,CASE WHEN $10='invalidated' THEN $9::timestamptz ELSE NULL END,$9,$9)`, op, s410ND, cred, src.Recovery.ScopeHash, src.Recovery.KeyLocator, src.Recovery.KeyDigest, src.Recovery.RequestDigest, src.Recovery.DigestKeyVersion, now, state, attempt)
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO temporary_credentials(id,user_id,purpose,generation,auth_generation,source_operation_id,state,secret_digest,delivered_to_email,issued_at,expires_at,invalidated_at) VALUES($1,$2,'password_reset',1,2,$3,CASE WHEN $5='invalidated' THEN 'recovery_invalidated' ELSE 'pending_delivery' END,'synthetic-original-digest','member@example.invalid',$4,$4::timestamptz+interval '1 day',CASE WHEN $5='invalidated' THEN $4::timestamptz ELSE NULL END)`, cred, s410ND, op, now, state)
	}
	if err == nil {
		_, err = (idempotency.Store{}).InsertPasswordResetClaim(h.ctx, tx, req, 1, op, s410ND)
	}
	if err == nil {
		_, err = tx.Exec(h.ctx, `INSERT INTO password_reset_recovery_jobs(operation_id,status,attempt_count,next_attempt_at,created_at,updated_at,completed_at) VALUES($1,CASE WHEN $3='invalidated' THEN 'succeeded' ELSE 'retry_wait' END,1,$2,$2,$2,CASE WHEN $3='invalidated' THEN $2::timestamptz ELSE NULL END)`, op, now, state)
	}
	if err == nil && state == "invalidated" {
		id, _ := (s410IDs{}).NewID()
		system := "password-reset-worker"
		err = (audit.Store{}).Append(h.ctx, tx, audit.Event{ID: id, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &system, Action: "identity.password_reset.recovered", TargetType: "password_reset_operation", TargetID: op, Outcome: "succeeded", TraceID: "00000000000000000000000001", Metadata: map[string]any{"attempt_count": 1}})
	}
	if err == nil {
		err = tx.Commit(h.ctx)
	}
	if err != nil {
		h.t.Fatal(err)
	}
	return m, src
}
func s410PGLinkedReset(t *testing.T) {
	for _, kind := range []string{"invalidated", "unknown", "hold", "dead_letter", "unelapsed_protection"} {
		t.Run(kind, func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			state := "unknown"
			if kind == "invalidated" {
				state = kind
			}
			m, src := h.linkedResetFixture(state)
			h.failedReader = s410CheckpointReader(src)
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				h.changed()
				if kind == "hold" {
					h.exec(`UPDATE users SET auth_integrity_hold=true,auth_integrity_hold_operation_id=$2,auth_integrity_hold_reason='synthetic-preserved-hold',auth_integrity_hold_at=$3 WHERE id=$1`, s410ND, src.Recovery.ResponseSnapshot.ResourceID, h.clock.Now())
				}
				if kind == "dead_letter" {
					h.exec(`UPDATE password_reset_recovery_jobs SET status='dead_letter' WHERE operation_id=$1`, src.Recovery.ResponseSnapshot.ResourceID)
				}
				if kind == "unelapsed_protection" {
					h.exec(`UPDATE password_reset_recovery_jobs SET next_attempt_at=$2 WHERE operation_id=$1`, src.Recovery.ResponseSnapshot.ResourceID, h.clock.Now().Add(time.Hour))
				}
				var before []byte
				if err := h.pool.QueryRow(c, `SELECT to_jsonb(o)::text FROM password_reset_operations o WHERE operation_id=$1`, src.Recovery.ResponseSnapshot.ResourceID).Scan(&before); err != nil {
					t.Fatal(err)
				}
				tr := m.transition()
				err := e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef)
				if kind == "invalidated" {
					if err != nil {
						t.Fatal(err)
					}
					if h.auditCount(recoveryIdempotencyAction) != 1 {
						t.Fatal("lawful original terminal projection missing")
					}
				} else if err == nil || h.auditCount(recoveryIdempotencyAction) != 0 || h.count(`SELECT count(*) FROM idempotency_records WHERE status='recovery_review_required'`) != 1 {
					t.Fatal("unknown/hold/dead-letter/protection work was settled by identity resolver")
				}
				var after []byte
				if er := h.pool.QueryRow(c, `SELECT to_jsonb(o)::text FROM password_reset_operations o WHERE operation_id=$1`, src.Recovery.ResponseSnapshot.ResourceID).Scan(&after); er != nil {
					t.Fatal(er)
				}
				if !bytes.Equal(before, after) {
					t.Fatal("resolver changed original domain operation or lineage")
				}
			})
		})
	}
}
func s410PGTypedConstruction(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	_, m, src := h.seedFailure("administrator409")
	reader := s410CheckpointReader(src)
	// These are exclusively exported identity/recovery construction surfaces:
	// cmd need not name a private type, reflect, or modify this source later.
	f := NewRecoveryIdentityForwarder()
	rt, ctx, done := s410Runtime(t, h.store, h.store, h.j, f)
	defer done()
	executor, maintenance, err := NewRecoveryIdentityExecution(RecoveryIdentityExecutionDependencies{Store: h.store, Runtime: rt, Journal: h.j, Articles: h.articles, Audit: h.audit, Passwords: h.passwords, IDs: s410IDs{}, Clock: h.clock, FailedEvidence: reader})
	if err != nil {
		t.Fatal(err)
	}
	same, ok := maintenance.(recovery.IdentityRecoveryExecutor)
	if !ok || same != executor {
		t.Fatal("root received distinct executors/Stores instead of one implementation")
	}
	called := 0
	// Only this explicit test bridge models the human maintenance interval.
	// It obtains the real public producer's raw permit; no private phase or
	// production root/HTTP/session/approval/SMTP claim is made by the fixture.
	bridge := &s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
		called++
		if err := executor.FinalizeRestoredIdentity(c, p, b); err != nil {
			return err
		}
		if err := executor.PrepareRecoveryTrust(c, p, b, []byte("synthetic-secret")); err != nil {
			return err
		}
		h.changed()
		for _, id := range []string{s410ND, s410D} {
			if err := maintenance.ResolveRestrictedAccountReview(c, p, b, id, "denied", "synthetic-reviewed-approval"); err != nil {
				return err
			}
		}
		tr := m.transition()
		if err := maintenance.ResolveRecoveryIdempotency(c, p, b, tr.ScopeHash, tr.KeyLocator, 1, "failed", src.EvidenceRef); err != nil {
			return err
		}
		if err := executor.CompleteIdentityReview(c, p, b); err != nil {
			return err
		}
		return s410Stop
	}}
	if err = f.Seal(bridge); err != nil {
		t.Fatal(err)
	}
	if err = f.Seal(executor); err == nil {
		t.Fatal("typed construction allowed replacement after Seal")
	}
	if err = rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) || called != 1 {
		t.Fatalf("public Runtime/typed identity DI: %v calls=%d", err, called)
	}
	if h.auditCount(recoveryFinalizeAction) != 1 || h.auditCount(recoveryIdempotencyAction) != 1 || h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='identity_ready'`) != 1 || h.j.consumes != 0 {
		t.Fatal("typed real-consumer positive failed or consumed/opened ordinary work")
	}
	if err = maintenance.ResolveRecoveryIdempotency(ctx, recovery.PhasePermit{}, s410Binding, m.transition().ScopeHash, m.transition().KeyLocator, 1, "failed", src.EvidenceRef); err == nil {
		t.Fatal("constructed helper minted authority without raw permit")
	}
}

// These cases are written for the future isolated PG window only. They can be
// transplanted unchanged onto the immutable pre-fix candidate: they call only
// existing public-producer/consumer methods and the original Commit seam. A
// compiled positive failure on that old candidate will be the real Red; no
// textual preparation/timeout/SQL setup error is counted as Red or Green.
func s410PGScopedACK(t *testing.T) {
	cases := []struct {
		name, action, peer string
		wantUnknown        bool
	}{
		{"ResolveIgnoresOtherReviewCommit", "resolve", "review", false},
		{"ResolveIgnoresOtherClaimAndAudit", "resolve", "claim", false},
		{"TrustIgnoresOtherClaimAndAudit", "trust", "claim", false},
		{"DeniedIgnoresOtherTargetReview", "denied", "review", false},
		{"KeptCanonicalIgnoresOtherTargetReview", "kept", "review", false},
		{"ResolveRejectsDuplicateOwnResolutionAudit", "resolve", "duplicate", true},
		{"TrustRejectsRelatedUserGenerationChange", "trust", "generation", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			var m recoveryIdempotencyMember
			var src RecoveryFailedEvidence
			if tc.action == "resolve" {
				_, m, src = h.seedFailure("administrator409")
				h.failedReader = s410CheckpointReader(src)
			}
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.finalized(c, p, e)
				if tc.action != "trust" {
					if err := e.PrepareRecoveryTrust(c, p, s410Binding, []byte("synthetic-secret")); err != nil {
						t.Fatal(err)
					}
					h.changed()
				}
				// A new non-cohort claim belongs to a different original domain
				// intent. Its real Store transition is legal data; this fixture
				// does not grant ordinary work permission during recovery.
				var unrelated recoveryIdempotencyMember
				if tc.peer == "claim" {
					_, unrelated, _ = h.seedFailure("administrator503")
				}
				committed := make(chan struct{})
				peerResult := make(chan error, 1)
				joined := make(chan struct{})
				peerCtx, cancelPeer := context.WithTimeout(c, 5*time.Second)
				defer cancelPeer()
				go func() {
					defer close(joined)
					select {
					case <-committed:
					case <-peerCtx.Done():
						peerResult <- peerCtx.Err()
						return
					}
					peerResult <- s410ScopedPeer(peerCtx, p, e, h, tc.peer, unrelated, m, src)
				}()
				marker := errors.New("synthetic ACK lost after committed scope and joined peer")
				var commits atomic.Int32
				var writer atomic.Uint32
				e.commit = func(cc context.Context, tx pgx.Tx) error {
					if err := tx.Commit(cc); err != nil {
						return err
					}
					if commits.Add(1) != 1 {
						return nil
					} // the other review uses its own real transaction
					writer.Store(tx.Conn().PgConn().PID())
					close(committed)
					select {
					case err := <-peerResult:
						if err != nil {
							return errors.Join(marker, err)
						}
					case <-peerCtx.Done():
						return errors.Join(marker, peerCtx.Err())
					}
					// The peer transaction has ended and its goroutine is joined
					// before observing which physical connection performs the ACK.
					<-joined
					h.trace.start()
					return marker
				}
				var err error
				switch tc.action {
				case "resolve":
					tr := m.transition()
					err = e.resolveIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, m.ExecutionGeneration, "failed", src.EvidenceRef)
				case "trust":
					err = e.PrepareRecoveryTrust(c, p, s410Binding, []byte("synthetic-secret"))
				case "denied":
					err = e.resolveRestricted(c, p, s410Binding, s410ND, "denied", "synthetic-reviewed-approval")
				case "kept":
					err = e.resolveRestricted(c, p, s410Binding, s410ND, "kept_deleted", "synthetic-reviewed-approval")
				}
				cancelPeer()
				<-joined // true join, including error/cancel paths; bounded peer I/O
				pids := h.trace.stop()
				// A failed setup/barrier/peer action is a diagnostic failure, never
				// the desired ACK assertion or a timeout-based behavioral Red.
				if commits.Load() == 0 {
					t.Fatalf("primary never reached actual Commit: %v", err)
				}
				if peerCtx.Err() != nil && !errors.Is(peerCtx.Err(), context.Canceled) {
					t.Fatalf("bounded peer timed out: %v (not Red/Green)", peerCtx.Err())
				}
				if len(pids) == 0 || pids[writer.Load()] {
					t.Fatalf("ACK did not use a distinct physical connection: writer=%d readers=%v result=%v", writer.Load(), pids, err)
				}
				if tc.wantUnknown {
					if !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || !errors.Is(err, marker) {
						t.Fatalf("related difference was confirmed or cause lost: %v", err)
					}
				} else if err != nil {
					t.Fatalf("unrelated lawful change invalidated exact same-intent ACK: %v", err)
				}
				if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='running'`) != 1 || h.j.consumes != 0 {
					t.Fatal("ACK comparison opened/consumed recovery")
				}
				switch tc.action {
				case "resolve":
					want := int64(1)
					if tc.peer == "duplicate" {
						want = 2
					}
					if h.auditCount(recoveryIdempotencyAction) != want || h.count(`SELECT count(*) FROM idempotency_records WHERE scope_hash=$1 AND key_locator=$2 AND status='failed' AND execution_generation=$3`, m.transition().ScopeHash, m.transition().KeyLocator, m.ExecutionGeneration) != 1 {
						t.Fatal("resolution was retried or its atomic facts are absent")
					}
				case "trust":
					if h.passwords.calls.Load() != 1 || h.auditCount(recoveryTrustAction) != 1 {
						t.Fatal("trust rehashed/retried or lost its unique Audit")
					}
				default:
					if h.count(`SELECT count(*) FROM recovery_account_reviews WHERE user_id=$1 AND decision=$2`, s410ND, map[string]string{"denied": "denied", "kept": "kept_deleted"}[tc.action]) != 1 {
						t.Fatal("target review did not really commit")
					}
				}
			})
		})
	}
}
func s410ScopedPeer(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution, h *s410PG, kind string, unrelated, m recoveryIdempotencyMember, src RecoveryFailedEvidence) (result error) {
	if kind == "review" {
		return e.resolveRestricted(c, p, s410Binding, s410D, "denied", "other-target-immutable-approval")
	}
	tx, err := h.pool.Begin(c)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(c, tx)) }()
	switch kind {
	case "claim":
		tr := unrelated.transition()
		tr.Now = h.clock.Now()
		if err = (idempotency.Store{}).MarkUncertain(c, tx, tr); err != nil {
			return err
		}
		id, err := (s410IDs{}).NewID()
		if err != nil {
			return err
		}
		system := "synthetic-independent-domain"
		trace, _ := platform.TraceIDFromContext(c)
		if err = (audit.Store{}).Append(c, tx, audit.Event{ID: id, OccurredAt: h.clock.Now(), ActorKind: audit.ActorSystem, ActorSystem: &system, Action: "synthetic.other.intent", TargetType: "user", TargetID: s410D, Outcome: "succeeded", TraceID: trace, Metadata: map[string]any{"intent": "unrelated-existing-domain"}}); err != nil {
			return err
		}
	case "duplicate":
		a, err := recoveryReadResolution(c, tx, s410Binding, "s410-install", m)
		if err != nil {
			return err
		}
		// Adversarial but schema-valid duplicate evidence; no immutable Audit
		// is changed/deleted and no successful result is forged as a fixture.
		if err = e.appendResolution(c, tx, a); err != nil {
			return err
		}
	case "generation":
		tag, err := tx.Exec(c, `UPDATE users SET auth_generation=auth_generation+1,permanent_password_generation=auth_generation+1 WHERE id=$1`, s410Initial)
		if err = recoveryExecutionOne(tag.RowsAffected(), err); err != nil {
			return err
		}
	default:
		return errors.New("unknown synthetic peer action")
	}
	return tx.Commit(c)
}

// Read-only physical retention receipt within the granted exact test top. It
// observes only schemas this fixture actually created, including failed cases;
// it never repairs, drops, cleans or treats a logged name as physical evidence.
func s410PhysicalRetention(t *testing.T, names []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	cfg, err := pgxpool.ParseConfig(dsn)
	if dsn == "" || err != nil || cfg.ConnConfig.Host != "127.0.0.1" || cfg.ConnConfig.Database != "han_b13d_identity" || cfg.ConnConfig.Port == 0 {
		t.Error("physical retention: exact fresh identity configuration missing")
		return
	}
	cfg.MaxConns = 2
	cfg.ConnConfig.RuntimeParams["search_path"] = "pg_catalog"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Error("physical retention pool failed")
		return
	}
	defer pool.Close()
	var address, database, user string
	var port, version int
	err = pool.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),current_database(),current_user,current_setting('server_version_num')::int`).Scan(&address, &port, &database, &user, &version)
	if err != nil || address != "127.0.0.1" || port != int(cfg.ConnConfig.Port) || database != cfg.ConnConfig.Database || user != cfg.ConnConfig.User || version != 180006 {
		t.Error("physical retention server preflight failed")
		return
	}
	if len(names) == 0 {
		t.Error("physical retention found no S410 schemas")
		return
	}
	for _, name := range names {
		var tables, runs, audits int64
		// Only the exact created name is observed, never a shared-prefix sweep.
		// Keep complete physical namespace/table/run/Audit evidence and all data.
		query := `SELECT (SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND c.relkind IN('r','p')), (SELECT count(*) FROM ` + pgx.Identifier{name, "recovery_runs"}.Sanitize() + `), (SELECT count(*) FROM ` + pgx.Identifier{name, "audit_events"}.Sanitize() + `)`
		if err = pool.QueryRow(ctx, query, name).Scan(&tables, &runs, &audits); err != nil {
			t.Error(err)
			return
		}
		if tables != 23 {
			t.Error("retained schema differs from the fixed 0001-0010 table set")
			return
		}
		raw, err := json.Marshal(map[string]any{"schema": name, "exists": true, "table_count": tables, "recovery_run_count": runs, "audit_count": audits, "address": address, "port": port, "database": database, "user": user, "server_version_num": version, "query": "pg_namespace/pg_class and quoted owned recovery_runs/audit_events counts"})
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("physical_retained_schema=%s", raw)
	}
}
func s410PGRestoredSnapshotReject(t *testing.T) {
	h := s410PGHarness(t)
	h.running()
	h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
		h.trusted(c, p, e)
		h.changed()
		h.restoredFixture(s410ND, true)
		if err := e.ResolveRestrictedAccountReview(c, p, s410Binding, s410D, "denied", "synthetic-reviewed-approval"); err != nil {
			t.Fatal(err)
		}
		// The full six-field aggregate is otherwise schema-legal and bound.
		// A generic terminal result is not the existing durable success wire.
		if err := e.CompleteIdentityReview(c, p, s410Binding); err == nil {
			t.Fatal("generic restored claim snapshot was accepted as original-domain success proof")
		}
		if h.count(`SELECT count(*) FROM recovery_runs WHERE outcome='running'`) != 1 {
			t.Fatal("bad restored claim opened identity")
		}
	})
}

func s410PGExitHistory(t *testing.T) {
	for _, kind := range []string{"zero_with_prior_revoked", "nonzero_retry_with_prior_revoked", "duplicate_ids", "wrong_reason", "old_exit_reason_without_audit"} {
		t.Run(kind, func(t *testing.T) {
			h := s410PGHarness(t)
			h.succeeded()
			h.session("ordinary", "", false)
			h.session("maintenance", "prior", true)
			reason := "synthetic_prior_revoke"
			if kind == "old_exit_reason_without_audit" {
				reason = recoveryExitAction
			}
			h.exec(`UPDATE sessions SET revoked_at=$1,revoke_reason=$2 WHERE recovery_runtime_id='prior'`, h.clock.Now(), reason)
			if kind == "nonzero_retry_with_prior_revoked" {
				h.session("maintenance", "current", false)
			}
			h.exit(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				if kind == "old_exit_reason_without_audit" {
					for i := 0; i < 2; i++ {
						if err := e.RevokeRecoverySessions(c, p, s410Binding); err == nil {
							t.Fatal("partial earlier exit was repaired with invented empty Audit")
						}
					}
					if h.auditCount(recoveryExitAction) != 0 {
						t.Fatal("partial exit appended an Audit")
					}
					return
				}
				if kind == "duplicate_ids" || kind == "wrong_reason" {
					var id string
					if err := h.pool.QueryRow(c, `SELECT id::text FROM sessions WHERE recovery_runtime_id='prior'`).Scan(&id); err != nil {
						t.Fatal(err)
					}
					ids := []string{id}
					if kind == "duplicate_ids" {
						ids = append(ids, id)
					}
					tx, err := h.pool.Begin(c)
					if err != nil {
						t.Fatal(err)
					}
					err = e.event(c, tx, recoveryExitAction, s410Binding.RunID, s410Binding, "s410-install", map[string]any{"revoked_ids": ids, "revoked_count": len(ids)})
					if err == nil {
						err = tx.Commit(c)
					}
					err = errors.Join(err, finishAdmissionRead(c, tx))
					if err != nil {
						t.Fatal(err)
					}
					if err = e.RevokeRecoverySessions(c, p, s410Binding); err == nil {
						t.Fatal("invented repeated/non-exit revoked result accepted")
					}
				} else {
					for i := 0; i < 2; i++ {
						if err := e.RevokeRecoverySessions(c, p, s410Binding); err != nil {
							t.Fatal(err)
						}
					}
				}
				if h.auditCount(recoveryExitAction) != 1 || h.count(`SELECT count(*) FROM sessions WHERE access_scope='ordinary' AND revoked_at IS NULL`) != 1 || h.count(`SELECT count(*) FROM sessions WHERE recovery_runtime_id='prior' AND revoke_reason='synthetic_prior_revoke'`) != 1 {
					t.Fatal("exit changed ordinary/prior history or duplicated Audit")
				}
			})
		})
	}
}

func s410PGDirectRejected(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing_source_%v", missing), func(t *testing.T) {
			h := s410PGHarness(t)
			h.running()
			m, src := h.linkedResetFixture("dispatching")
			op := src.Recovery.ResponseSnapshot.ResourceID
			// Original RecordRejectedDelivery writer: no worker recovered Audit,
			// no failed generic claim. Use that REAL method in one transaction.
			tx, err := h.pool.Begin(h.ctx)
			if err != nil {
				t.Fatal(err)
			}
			o, err := recoveryReadResetOperation(h.ctx, tx, op)
			if err == nil {
				err = (&PasswordResetStore{}).RecordRejectedDelivery(h.ctx, tx, o, o.TemporaryCredentialID, *o.DeliveryAttemptID, s410Time)
			}
			if err == nil {
				err = tx.Commit(h.ctx)
			}
			err = errors.Join(err, finishAdmissionRead(h.ctx, tx))
			if err != nil {
				t.Fatal(err)
			}
			if !missing {
				h.failedReader = s410CheckpointReader(src)
			}
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				h.changed()
				tr := m.transition()
				err := e.ResolveRecoveryIdempotency(c, p, s410Binding, tr.ScopeHash, tr.KeyLocator, m.ExecutionGeneration, "failed", src.EvidenceRef)
				if missing {
					if err == nil || h.auditCount(recoveryIdempotencyAction) != 0 {
						t.Fatal("missing original failed source accepted")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if h.auditCount("identity.password_reset.recovered") != 0 || h.auditCount(recoveryIdempotencyAction) != 1 {
					t.Fatal("direct rejection required/invented worker Audit or missed real resolution")
				}
				for _, id := range []string{s410ND, s410D} {
					if err = e.ResolveRestrictedAccountReview(c, p, s410Binding, id, "denied", "synthetic-reviewed-approval"); err != nil {
						t.Fatal(err)
					}
				}
				if err = e.CompleteIdentityReview(c, p, s410Binding); err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}

func s410PGDeletionInstant(t *testing.T) {
	for _, zone := range []string{"UTC", "+00:00", "+09:30"} {
		t.Run(zone, func(t *testing.T) {
			h := s410PGHarnessTimezone(t, zone)
			h.running()
			h.phase(func(c context.Context, p recovery.PhasePermit, e *recoveryIdentityExecution) {
				h.trusted(c, p, e)
				if h.count(`SELECT count(*) FROM recovery_runs WHERE initial_admin_bootstrap_generation IS NOT NULL`) != 1 {
					t.Fatal("same-instant deleted history prevented actual trust preparation")
				}
			})
		})
	}
}

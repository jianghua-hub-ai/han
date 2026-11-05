//go:build integration

package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type s413PG struct {
	t      *testing.T
	ctx    context.Context
	pool   *pgxpool.Pool
	store  *RecoveryRunStore
	schema string
	world  *s413PGWorld
}
type s413SQLTrace struct{ t *testing.T }

func (x s413SQLTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	x.t.Logf("fullSQL=%s args=%v", d.SQL, d.Args)
	return ctx
}
func (x s413SQLTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func s413NewPG(t *testing.T, version int) *s413PG {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	c, err := pgxpool.ParseConfig(os.Getenv("HAN_TEST_POSTGRES_DSN"))
	if err != nil || c.ConnConfig.Host != "127.0.0.1" || c.ConnConfig.Port != 51935 || c.ConnConfig.Database != "han_b13e_identity" || c.ConnConfig.User != "han_b13e" {
		t.Fatal("explicit B13E identity test PG required; no fallback")
	}
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	schema := "han_b13e_s413_" + hex.EncodeToString(b[:])
	c.MaxConns = 3
	c.ConnConfig.Tracer = s413SQLTrace{t}
	c.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	var host, db, user, data string
	var port, server int
	err = p.QueryRow(ctx, `SELECT host(inet_server_addr()),inet_server_port(),current_database(),current_user,current_setting('server_version_num')::int,current_setting('data_directory')`).Scan(&host, &port, &db, &user, &server, &data)
	if err != nil || host != "127.0.0.1" || port != 51935 || db != "han_b13e_identity" || user != "han_b13e" || server != 180006 || strings.ToLower(filepath.ToSlash(data)) != "d:/temp/han-b13e-lead-20261030-r01/postgres/data" {
		t.Fatal("physical B13E target mismatch before DDL")
	}
	h := &s413PG{t: t, ctx: ctx, pool: p, schema: schema}
	h.exec("CREATE SCHEMA " + pgx.Identifier{schema}.Sanitize())
	t.Logf("retained_schema=%s", schema)
	h.migrate(1, version)
	h.store, err = NewRecoveryRunStore(p)
	if err != nil {
		t.Fatal(err)
	}
	h.world = s413WorldFor(h.store)
	t.Cleanup(func() { h.physical() })
	return h
}
func (h *s413PG) exec(sql string, args ...any) pgconn.CommandTag {
	h.t.Helper()
	tag, err := h.pool.Exec(h.ctx, sql, args...)
	if err != nil {
		h.t.Fatalf("synthetic SQL: %v", err)
	}
	return tag
}
func (h *s413PG) migrate(from, to int) {
	h.t.Helper()
	files, err := os.ReadDir(filepath.Join("..", "..", "migrations"))
	if err != nil {
		h.t.Fatal(err)
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		var n int
		for _, ch := range f.Name()[:4] {
			n = n*10 + int(ch-'0')
		}
		if n < from || n > to {
			continue
		}
		b, err := os.ReadFile(filepath.Join("..", "..", "migrations", f.Name()))
		if err != nil {
			h.t.Fatal(err)
		}
		tx, err := h.pool.Begin(h.ctx)
		if err != nil {
			h.t.Fatal(err)
		}
		func() {
			defer finishAdmissionRead(h.ctx, tx)
			_, err = tx.Exec(h.ctx, string(b))
			if err == nil {
				err = tx.Commit(h.ctx)
			}
		}()
		if err != nil {
			h.t.Fatalf("forward migration %s: %v", f.Name(), err)
		}
	}
}
func (h *s413PG) physical() {
	h.t.Helper()
	var exists bool
	var tables int
	err := h.pool.QueryRow(h.ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1),(SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND c.relkind IN('r','p'))`, h.schema).Scan(&exists, &tables)
	if err != nil || !exists || tables == 0 || h.pool.Stat().AcquiredConns() != 0 {
		h.t.Errorf("retained schema/SQL resource not proved: %v tables=%d", err, tables)
	}
	h.t.Logf("physical_retained_schema=%s exists=%t tables=%d acquired=%d", h.schema, exists, tables, h.pool.Stat().AcquiredConns())
}
func (h *s413PG) install() error { h.t.Helper(); return h.world.runtime.Install(h.ctx, h.world.m) }
func (h *s413PG) snapshot() recovery.AdmissionSnapshot {
	h.t.Helper()
	s, err := h.store.ReadAdmissionSnapshot(h.ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	return s
}

func TestRecoveryInstallationPostgres(t *testing.T) {
	t.Run("IndependentApprovalAndPreparedBound", func(t *testing.T) {
		t.Run("RealSameStoreBindingAndBothAnchors", s413PGFresh)
		t.Run("BoundOriginRestart", s413PGBoundOrigins)
		t.Run("DatabaseFailureRollsBackAndPreparedRetries", s413PGDatabaseFailure)
		t.Run("NoApprovalNoMetadata", func(t *testing.T) {
			h := s413NewPG(t, 11)
			h.world.approvalError = recovery.ErrAdmissionIncomplete
			h.world.newRuntime(t)
			if h.install() == nil || h.world.writes != 0 {
				t.Fatal("unapproved install wrote")
			}
			var n int
			if err := h.pool.QueryRow(h.ctx, `SELECT count(*) FROM recovery_installation`).Scan(&n); err != nil || n != 0 {
				t.Fatal("unapproved DB binding")
			}
		})
	})
	t.Run("InterruptedSameManifestOnly", s413PGInterruptions)
	t.Run("DeleteThenRestartAndBeginDisablesGenesis", s413PGRestart)
	t.Run("LegacyBindingAndForwardOnly", s413PGLegacy)
	t.Run("InstallationExecutorAndBoundReentry", s413PGBound)
}
func s413PGFresh(t *testing.T) {
	h := s413NewPG(t, 11)
	h.world.newRuntime(t)
	if err := h.install(); err != nil {
		t.Fatalf("real fresh install: %v", err)
	}
	s := h.snapshot()
	if s.Revision.Counter != 1 || len(s.Runs) != 0 || h.world.commit.Phase != "bound" || len(h.world.anchors) != 2 || h.world.starts != 0 {
		t.Fatal("fresh binding incomplete")
	}
	b, err := identityInstallationBinding(h.world.m)
	if err != nil || s.Installation != b {
		t.Fatal("DB does not match original manifest")
	}
	t.Logf("installation_revision=%d same_store=true prepared_bound=true", s.Revision.Counter)
}
func s413PGDatabaseFailure(t *testing.T) {
	h := s413NewPG(t, 11)
	x := h.world
	h.exec(`CREATE TABLE s413_binding_fault(blocked boolean NOT NULL); INSERT INTO s413_binding_fault VALUES(true);
	 CREATE FUNCTION s413_binding_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF (SELECT blocked FROM s413_binding_fault) THEN RAISE EXCEPTION 'synthetic installation constraint failure' USING ERRCODE='23514'; END IF; RETURN NEW; END $$;
	 CREATE TRIGGER s413_binding_failure BEFORE INSERT ON recovery_installation FOR EACH ROW EXECUTE FUNCTION s413_binding_failure()`)
	x.newRuntime(t)
	if err := h.install(); err == nil {
		t.Fatal("real SQL failure accepted installation")
	}
	var n int
	if err := h.pool.QueryRow(h.ctx, `SELECT count(*) FROM recovery_installation`).Scan(&n); err != nil || n != 0 || x.commit.Phase != "prepared" || len(x.anchors) != 2 {
		t.Fatal("failed SQL did not retain exact prepared state")
	}
	if err := x.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	h.exec(`UPDATE s413_binding_fault SET blocked=false`)
	x.newRuntime(t)
	if err := h.install(); err != nil {
		t.Fatal(err)
	}
	if h.snapshot().Revision.Counter != 1 || x.commit.Phase != "bound" {
		t.Fatal("same prepared retry overwrote binding")
	}
}
func s413PGInterruptions(t *testing.T) {
	for _, point := range []string{"prepared", recovery.RecoveryNamespaceV1, recovery.ArticleDeletionNamespaceV1, "after-db-commit", "bound"} {
		t.Run(point, func(t *testing.T) {
			h := s413NewPG(t, 11)
			x := h.world
			x.fail = point
			x.newRuntime(t)
			if point == "after-db-commit" {
				x.coordinateHook = func(context.Context, recovery.PhasePermit) error {
					x.coordinateHook = nil
					return recovery.ErrRecoveryWriteUnknown
				}
			}
			if err := h.install(); !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
				t.Fatalf("expected retained write unknown: %v", err)
			}
			if err := x.runtime.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			x.newRuntime(t)
			if err := h.install(); err != nil {
				t.Fatalf("same exact manifest resume: %v", err)
			}
			if h.snapshot().Revision.Counter != 1 || x.commit.Phase != "bound" {
				t.Fatal("retry duplicated DB installation")
			}
			if err := x.runtime.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			x.newRuntime(t)
			x.m.TargetDigest = s413Digest("wrong-target")
			if h.install() == nil {
				t.Fatal("different manifest resumed installation")
			}
		})
	}
}
func s413PGBound(t *testing.T) {
	h := s413NewPG(t, 11)
	x := h.world
	x.newRuntime(t)
	if err := h.install(); err != nil {
		t.Fatal(err)
	}
	before := h.snapshot()
	writes, media := x.writes, x.media
	if err := x.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	x.newRuntime(t)
	if err := h.install(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, h.snapshot()) || x.writes != writes || x.media != media || x.starts != 0 {
		t.Fatal("bound reentry performed full installation")
	}
}
func s413PGRestart(t *testing.T) {
	h := s413NewPG(t, 11)
	x := h.world
	x.newRuntime(t)
	if err := h.install(); err != nil {
		t.Fatal(err)
	}
	if err := x.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	x.newRuntime(t)
	if err := x.runtime.RunNormal(h.ctx); err != nil {
		t.Fatalf("bound genesis restart: %v", err)
	}
	s413ActualDelete(h)
	if err := x.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	x.newRuntime(t)
	if err := x.runtime.RunNormal(h.ctx); err != nil {
		t.Fatalf("normal restart after actual finalizer delete: %v", err)
	}
	if x.article.Head.Sequence != 1 || h.snapshot().Revision.Counter != 1 {
		t.Fatal("normal deletion reset genesis installation")
	}
	if err := x.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	x.journal.Entries = []recovery.JournalEntry{{Namespace: recovery.RecoveryNamespaceV1, RunID: "external-begun", BackupRef: "backup", KeyID: "retained", KeyVersion: "v1", SecretFingerprint: s413HMAC(), Stage: recovery.StageBegun, Sequence: 1, RecordedAt: s413Time, RecordHMAC: s413HMAC()}}
	x.journal.Head = recovery.JournalHead{Sequence: 1, Hash: s413Digest("begin")}
	x.journal.Prefixes = append(x.journal.Prefixes, x.journal.Head)
	x.journal.Revision = "external-begin-revision"
	x.newRuntime(t)
	if err := x.runtime.RunNormal(h.ctx); err == nil {
		t.Fatal("Begin with no Store history reused genesis")
	}
}

type s413DeletionJournal struct {
	h      *s413PG
	record article.ArticleDeletionTombstone
}

func (j *s413DeletionJournal) Append(ctx context.Context, q article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
	x := j.h.world
	if err := x.runtime.CheckJournalUse(ctx, q.DeletionOperationID, recovery.JournalUseNewAppend); err != nil {
		return article.ArticleDeletionTombstone{}, err
	}
	barrier, err := x.runtime.PauseArticleAppend(ctx, q.DeletionOperationID)
	if err != nil {
		return article.ArticleDeletionTombstone{}, err
	}
	old, oldToken, err := x.Load(ctx, recovery.ArticleDeletionNamespaceV1)
	if err != nil {
		return article.ArticleDeletionTombstone{}, err
	}
	r := article.ArticleDeletionTombstone{Namespace: recovery.ArticleDeletionNamespaceV1, Sequence: 1, ArticleID: q.ArticleID, ArticleVersion: q.ArticleVersion, DeletionOperationID: q.DeletionOperationID, DeletedAt: q.DeletedAt, KeyID: "synthetic-retained-key", KeyVersion: "v1", Stage: "deleted", RecordHMAC: s413HMAC()}
	raw, err := json.Marshal(r)
	if err != nil {
		return r, err
	}
	head := recovery.JournalHead{Sequence: 1, Hash: recovery.SHA256Digest(raw)}
	path, err := s413WriteSyntheticTombstone(raw)
	if err != nil {
		return r, err
	}
	pending := old
	pending.Pending = &recovery.AnchorPending{OldHead: old.Committed, NewHead: head, OperationID: q.DeletionOperationID}
	_, pendingToken, err := x.CompareAndSwap(ctx, old.Namespace, oldToken, pending)
	if err != nil {
		return r, err
	}
	next := old
	next.Committed = head
	_, committedToken, err := x.CompareAndSwap(ctx, old.Namespace, pendingToken, next)
	if err != nil {
		return r, err
	}
	j.record = r
	x.article.Head = head
	x.article.Prefixes = append(x.article.Prefixes, head)
	if err := x.runtime.CompleteArticleAppend(ctx, barrier, recovery.VerifiedHeadAdvance{Namespace: old.Namespace, JournalID: old.JournalID, Epoch: old.Epoch, OldHead: old.Committed, NewHead: head, OldRevision: oldToken, PendingRevision: pendingToken, CommittedRevision: committedToken, OperationID: q.DeletionOperationID}); err != nil {
		return r, err
	}
	j.h.t.Logf("retained_synthetic_deletion_media=%s exact_head=%s", path, head.Hash)
	return r, nil
}
func (j *s413DeletionJournal) FindByOperation(ctx context.Context, id string) (article.ArticleDeletionTombstone, error) {
	if j.record.DeletionOperationID != id {
		return article.ArticleDeletionTombstone{}, article.ErrDeletionTombstoneNotFound
	}
	return j.record, j.h.world.runtime.CheckJournalUse(ctx, id, recovery.JournalUseCommittedRead)
}

type s413DeletionIdentity struct {
	article.DispositionIdentityPort
}

func (s413DeletionIdentity) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, id string) (article.DispositionDeletionCandidate, error) {
	v, err := (&ArticleIdentityLifecycle{}).LockDeletionCandidate(ctx, tx, id)
	return article.DispositionDeletionCandidate{UserID: v.UserID, Deleted: v.Deleted}, err
}
func s413ActualDelete(h *s413PG) {
	h.t.Helper()
	x := h.world
	const user = "44444444-4444-4444-8444-444444444444"
	const id = "55555555-5555-4555-8555-555555555555"
	const op = "66666666-6666-4666-8666-666666666666"
	h.exec(`INSERT INTO users(id,username,email,phone,role,status,credential_mode,created_at,updated_at) VALUES($1,'s413-author','s413@example.invalid','+8613900000413','member','active','permanent',$2,$2)`, user, s413Time)
	for _, pair := range [][2]string{{"username", "s413-author"}, {"email", "s413@example.invalid"}, {"phone", "+8613900000413"}} {
		h.exec(`INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at) VALUES($1,$2,$3,'active',$4)`, user, pair[0], pair[1], s413Time)
	}
	h.exec(`INSERT INTO deletion_operations(id,article_id,article_version,delete_kind,actor_user_id,actor_role,status,started_at) VALUES($1,$2,1,'ordinary',$3,'member','pending_journal',$4)`, op, id, user, s413Time)
	h.exec(`INSERT INTO articles(id,author_id,title,body,publication_state,violation_state,deletion_state,deletion_operation_id,version,created_at,updated_at) VALUES($1,$2,'synthetic-title','synthetic-body','published','clear','deletion_pending',$3,2,$4,$4)`, id, user, op, s413Time)
	l, err := x.runtime.Acquire(h.ctx, recovery.WorkRequest)
	if err != nil {
		h.t.Fatal(err)
	}
	func() {
		defer l.Release()
		journal := &s413DeletionJournal{h: h}
		f, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{Transactions: h.pool, Repository: article.Repository{}, Journal: journal, Auditor: audit.Store{}, AuditIDs: s413IDs{}, Clock: s413Clock{}, WorkerID: "s413-finalizer", LeaseDuration: time.Minute, Identity: s413DeletionIdentity{}})
		if err != nil {
			h.t.Fatal(err)
		}
		traceID, err := platform.NewTraceID()
		if err != nil {
			h.t.Fatal(err)
		}
		if err := f.ConvergeDeletionOperation(l.Context(), op, traceID); err != nil {
			h.t.Fatal(err)
		}
	}()
	var count, events int
	var status string
	if err := h.pool.QueryRow(h.ctx, `SELECT (SELECT count(*) FROM articles WHERE id=$1),(SELECT count(*) FROM audit_events WHERE target_id=$1::text),status FROM deletion_operations WHERE id=$2`, id, op).Scan(&count, &events, &status); err != nil || count != 0 || events != 1 || status != "completed" {
		h.t.Fatalf("real finalizer SQL closure: count=%d events=%d status=%s err=%v", count, events, status, err)
	}
	h.t.Log("actual Article Finalizer physically deleted row, completed operation and one Audit; original installation revision retained; normal restarted from advanced committed head")
}
func s413PGLegacy(t *testing.T) {
	for _, mode := range []string{"ExactNullOnceAndIdempotent", "MissingOriginalApproval", "WrongEvidence", "ExistingWrongBinding", "WrongStoredEvidenceVersion", "WrongStoredNamespace"} {
		t.Run(mode, func(t *testing.T) { s413LegacyCase(t, mode) })
	}
}

func s413LegacyCase(t *testing.T, mode string) {
	h := s413NewPG(t, 9)
	x := h.world
	ready := s413LegacyRows(h)
	if mode == "WrongStoredEvidenceVersion" {
		h.exec(`UPDATE recovery_runs SET article_evidence_version='unknown/v1' WHERE run_id='legacy-run'`)
	}
	if mode == "WrongStoredNamespace" {
		h.exec(`UPDATE recovery_runs SET article_namespace='other/v1' WHERE run_id='legacy-run'`)
	}
	h.migrate(10, 11)
	p := s413PublicationRows(h, ready)
	x.m.Kind = "anchored_migration"
	x.m.Recovery.Committed = x.journal.Head
	x.m.Article.Committed = x.article.Head
	x.approved = x.m
	before := s413RunImage(h)
	path := s413Checkpoint(t, p)
	s413BindLegacyCheckpoint(x, p, path, mode)
	pubStore, err := article.NewPostgresReplayEvidenceStore(h.pool)
	if err != nil {
		t.Fatal(err)
	}
	x.closures = pubStore
	if mode == "ExistingWrongBinding" {
		x.coordinateHook = func(context.Context, recovery.PhasePermit) error {
			h.exec(`UPDATE recovery_runs SET article_publication_ref='already-bound-other',article_publication_digest=$1 WHERE run_id='legacy-run'`, s413Digest("other"))
			return nil
		}
	}
	x.newRuntime(t)
	s413ConfigurePublishedReader(h, pubStore, path)
	err = h.install()
	if mode != "ExactNullOnceAndIdempotent" {
		s413LegacyRefusal(h, mode, err)
		return
	}
	s413LegacySuccess(h, p, before, err)
}

func s413RunImage(h *s413PG) map[string]any {
	h.t.Helper()
	var raw []byte
	if err := h.pool.QueryRow(h.ctx, `SELECT to_jsonb(r) FROM recovery_runs r WHERE run_id='legacy-run'`).Scan(&raw); err != nil {
		h.t.Fatal(err)
	}
	var image map[string]any
	if err := json.Unmarshal(raw, &image); err != nil {
		h.t.Fatal(err)
	}
	var auditImage []byte
	if err := h.pool.QueryRow(h.ctx, `SELECT to_jsonb(a) FROM audit_events a WHERE id='11111111-1111-4111-8111-111111111111'`).Scan(&auditImage); err != nil {
		h.t.Fatal(err)
	}
	image["original_audit_image"] = string(auditImage)
	return image
}
func s413Checkpoint(t *testing.T, p recovery.PublicationBinding) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "approved-original-publication.json")
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("synthetic_independent_checkpoint=%s sha256=%s", path, recovery.SHA256Digest(raw))
	return path
}
func s413LegacyRows(h *s413PG) recovery.ReadyReplayStatusV1 {
	x := h.world
	begun := recovery.JournalEntry{Namespace: recovery.RecoveryNamespaceV1, Sequence: 1, RunID: "legacy-run", BackupRef: "legacy-backup", KeyID: "retained-key", KeyVersion: "v1", SecretFingerprint: s413HMAC(), Stage: recovery.StageBegun, RecordedAt: s413Time, RecordHMAC: s413HMAC()}
	consumed := begun
	consumed.Sequence = 2
	consumed.Stage = recovery.StageConsumed
	consumed.RecordedAt = s413Time.Add(time.Minute)
	consumed.PreviousRecordHash = s413Digest("original-begun")
	x.journal.Entries = []recovery.JournalEntry{begun, consumed}
	x.journal.Head = recovery.JournalHead{Sequence: 2, Hash: s413Digest("original-consumed")}
	x.journal.Prefixes = []recovery.JournalHead{{}, {Sequence: 1, Hash: consumed.PreviousRecordHash}, x.journal.Head}
	x.article.Head = recovery.JournalHead{Sequence: 1, Hash: s413Digest("original-tombstone")}
	x.article.Prefixes = []recovery.JournalHead{{}, x.article.Head}
	r := recovery.ReadyReplayStatusV1{EvidenceVersion: recovery.ReplayEvidenceVersionV1, RunID: begun.RunID, BackupRef: begun.BackupRef, Namespace: recovery.ArticleDeletionNamespaceV1, JournalHeadSequence: 1, JournalHeadHash: x.article.Head.Hash, InventoryScope: "recoverable-backups-before-cutoff", BackupCutoff: s413Time.Format(time.RFC3339Nano), InventoryDigest: s413Digest("inventory"), DeletionOperationsDigest: s413Digest("operations"), AuditEventsDigest: s413Digest("audits"), TombstonesTotal: 1, TombstonesReplayed: 1, DeletionOperationsReconciledCount: 1, AuditEventsVerifiedCount: 1, Status: "ready", EvidenceRef: "legacy-evidence", CompletedAt: consumed.RecordedAt.Format("2006-01-02T15:04:05.000000Z")}
	raw, _ := json.Marshal(r)
	end := strings.Index(string(raw), `,"status":`)
	r.EvidenceDigest = recovery.SHA256Digest(append(raw[:end], '}'))
	h.exec(`INSERT INTO recovery_runs(run_id,backup_ref,journal_key_id,journal_key_version,journal_secret_fingerprint,journal_stage,started_at,outcome,account_count,session_count,audit_event_id) VALUES('legacy-run','legacy-backup','retained-key','v1',$1,'begun',$2,'running',7,13,'11111111-1111-4111-8111-111111111111')`, s413HMAC(), s413Time)
	for _, outcome := range []string{"identity_ready", "article_deletion_ready", "ready_to_consume"} {
		h.exec(`UPDATE recovery_runs SET outcome=$1 WHERE run_id='legacy-run'`, outcome)
	}
	h.exec(`UPDATE recovery_runs SET outcome='journal_consumed',journal_stage='consumed',journal_consumed_at=$1,article_evidence_version=$2,article_evidence_digest=$3,article_evidence_ref=$4,article_evidence_completed_at=$1,article_namespace=$5,article_replay_verified_at=$1 WHERE run_id='legacy-run'`, consumed.RecordedAt, r.EvidenceVersion, r.EvidenceDigest, r.EvidenceRef, r.Namespace)
	h.exec(`UPDATE recovery_runs SET outcome='succeeded',completed_at=$1 WHERE run_id='legacy-run'`, consumed.RecordedAt)
	traceID, err := platform.NewTraceID()
	if err != nil {
		h.t.Fatal(err)
	}
	h.exec(`INSERT INTO audit_events(id,occurred_at,actor_kind,actor_system,action,target_type,target_id,outcome,trace_id,metadata)
	 VALUES('11111111-1111-4111-8111-111111111111',$1,'system','synthetic-legacy-recovery','identity.recovery.finalize','recovery_run','legacy-run','succeeded',$2,'{"account_count":7,"session_count":13}'::jsonb)`, consumed.RecordedAt, traceID)
	h.exec(`INSERT INTO deletion_replay_runs(run_id,namespace,backup_ref,status,started_at,updated_at) VALUES($1,$2,$3,'running',$4,$4)`, r.RunID, r.Namespace, r.BackupRef, s413Time)
	h.exec(`INSERT INTO deletion_replay_evidence(evidence_version,run_id,backup_ref,namespace,journal_head_sequence,journal_head_hash,inventory_scope,backup_cutoff,inventory_digest,deletion_operations_digest,audit_events_digest,tombstones_total,tombstones_replayed,already_absent,deletion_operations_reconciled_count,audit_events_verified_count,evidence_digest,evidence_ref,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1,1,0,1,1,$12,$13,$14)`, r.EvidenceVersion, r.RunID, r.BackupRef, r.Namespace, r.JournalHeadSequence, r.JournalHeadHash, r.InventoryScope, s413Time, r.InventoryDigest, r.DeletionOperationsDigest, r.AuditEventsDigest, r.EvidenceDigest, r.EvidenceRef, consumed.RecordedAt)
	h.exec(`UPDATE deletion_replay_runs SET status='ready',current_evidence_ref=$1,input_digest=$2 WHERE run_id='legacy-run'`, r.EvidenceRef, s413Digest("original-input"))
	return r
}
func s413PublicationRows(h *s413PG, r recovery.ReadyReplayStatusV1) recovery.PublicationBinding {
	at := s413Time.Add(time.Minute)
	published := at.Add(time.Minute)
	p := recovery.PublicationBinding{Ref: "22222222-2222-4222-8222-222222222222", Version: "article-replay-publication/v1", RunID: r.RunID, BackupRef: r.BackupRef, EvidenceRef: r.EvidenceRef, EvidenceDigest: r.EvidenceDigest, JournalID: h.world.m.Article.JournalID, Epoch: h.world.m.Article.Epoch, Head: h.world.article.Head, EvidenceCompletedAt: at, PublishedAt: published}
	// Independent original 17-field publication oracle; no producer encoder.
	names := []string{"publication_version", "publication_ref", "source_kind", "run_id", "backup_ref", "namespace", "evidence_ref", "evidence_digest", "evidence_completed_at", "journal_id", "journal_epoch", "journal_head_sequence", "journal_head_hash", "attempt_generation", "source_ref", "source_digest", "published_at"}
	values := []any{p.Version, p.Ref, "verified_legacy_import_v1", p.RunID, p.BackupRef, recovery.ArticleDeletionNamespaceV1, p.EvidenceRef, p.EvidenceDigest, at.Format("2006-01-02T15:04:05.000000Z"), p.JournalID, p.Epoch, p.Head.Sequence, p.Head.Hash, 0, "independent-original-publication", s413Digest("approved-original-checkpoint"), published.Format("2006-01-02T15:04:05.000000Z")}
	var b strings.Builder
	b.WriteByte('{')
	for i, name := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(name)
		v, _ := json.Marshal(values[i])
		b.Write(key)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	p.Digest = recovery.SHA256Digest([]byte(b.String()))
	h.exec(`INSERT INTO article_replay_publications(publication_ref,publication_version,source_kind,run_id,namespace,backup_ref,evidence_ref,evidence_digest,attempt_generation,journal_id,journal_epoch,journal_head_sequence,journal_head_hash,evidence_completed_at,published_at,publication_digest,source_ref,source_digest) VALUES($1,$2,'verified_legacy_import_v1',$3,$4,$5,$6,$7,0,$8,$9,$10,$11,$12,$13,$14,'independent-original-publication',$15)`, p.Ref, p.Version, p.RunID, recovery.ArticleDeletionNamespaceV1, p.BackupRef, p.EvidenceRef, p.EvidenceDigest, p.JournalID, p.Epoch, p.Head.Sequence, p.Head.Hash, p.EvidenceCompletedAt, p.PublishedAt, p.Digest, s413Digest("approved-original-checkpoint"))
	return p
}

type s413ArticleSnapshot struct{ x *s413PGWorld }

func (j s413ArticleSnapshot) ReadAllVerified(context.Context) ([]article.ArticleDeletionTombstone, error) {
	return nil, errors.New("unexpected ReadAll in historical lookup")
}
func (j s413ArticleSnapshot) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	return j.x.ReadVerifiedSnapshot(ctx, ns)
}

type s413ForbiddenReplay struct{}

func (s413ForbiddenReplay) ReplayAllTombstones(context.Context, []article.ArticleDeletionTombstone, string) (article.DeletionReplayAllResult, error) {
	return article.DeletionReplayAllResult{}, errors.New("unexpected ReplayAll in history")
}

type s413IDs struct{}

func (s413IDs) NewID() (string, error) { return "33333333-3333-4333-8333-333333333333", nil }

type s413Clock struct{}

func (s413Clock) Now() time.Time { return s413Time.Add(2 * time.Minute) }
func s413ConfigurePublishedReader(h *s413PG, store *article.PostgresReplayEvidenceStore, path string) {
	h.t.Helper()
	reader, err := article.NewPostgresReplayFactReader(h.pool)
	if err != nil {
		h.t.Fatal(err)
	}
	inv, err := article.NewFileBackupInventoryReader(path)
	if err != nil {
		h.t.Fatal(err)
	}
	source, err := article.NewProductionDeletionReplaySource(s413ArticleSnapshot{h.world}, inv, s413ForbiddenReplay{}, reader, reader, func() time.Time { return s413Time })
	if err != nil {
		h.t.Fatal(err)
	}
	actual, err := article.NewReplayPublicationPorts(store, article.ReplayPublicationDependencies{Source: source, Runtime: h.world.runtime, InstallationID: h.world.m.InstallationID, PublicationIDs: s413IDs{}, Clock: s413Clock{}})
	if err != nil || actual != store {
		h.t.Fatalf("same S4.8 store reader: %v", err)
	}
}

func s413BindLegacyCheckpoint(x *s413PGWorld, p recovery.PublicationBinding, path, mode string) {
	x.legacy = func(ctx context.Context, permit recovery.PhasePermit, m recovery.InstallationManifest) ([]recovery.PublicationBinding, error) {
		if err := x.runtime.CheckPhase(ctx, permit, "install"); err != nil {
			return nil, err
		}
		if mode == "MissingOriginalApproval" {
			return nil, recovery.ErrAdmissionIncomplete
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var approved recovery.PublicationBinding
		if err := json.Unmarshal(raw, &approved); err != nil || approved != p {
			return nil, recovery.ErrAdmissionBinding
		}
		if mode == "WrongEvidence" {
			approved.EvidenceDigest = s413Digest("wrong")
		}
		return []recovery.PublicationBinding{approved, approved}, nil
	}
}

func s413LegacyRefusal(h *s413PG, mode string, err error) {
	t := h.t
	if err == nil {
		t.Fatal("untrusted or wrong legacy source bound")
	}
	var ref *string
	if e := h.pool.QueryRow(h.ctx, `SELECT article_publication_ref FROM recovery_runs WHERE run_id='legacy-run'`).Scan(&ref); e != nil {
		t.Fatal(e)
	}
	if mode == "ExistingWrongBinding" {
		if ref == nil || *ref != "already-bound-other" {
			t.Fatal("existing wrong binding overwritten")
		}
	} else if ref != nil {
		t.Fatal("rejected legacy binding wrote")
	}
}

func s413LegacySuccess(h *s413PG, p recovery.PublicationBinding, before map[string]any, err error) {
	t := h.t
	if err != nil {
		t.Fatalf("actual BindLegacyPublication: %v", err)
	}
	after := s413RunImage(h)
	delete(after, "article_publication_ref")
	delete(after, "article_publication_digest")
	delete(before, "article_publication_ref")
	delete(before, "article_publication_digest")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("binding changed original outcome/evidence/count/time/audit")
	}
	s := h.snapshot()
	if s.Revision.Counter != 2 || len(s.Runs) != 1 || s.Runs[0].ArticlePublicationRef == nil || *s.Runs[0].ArticlePublicationRef != p.Ref {
		t.Fatalf("NULL update was not once only: revision=%d", s.Revision.Counter)
	}
	var generation int64
	var source string
	if err := h.pool.QueryRow(h.ctx, `SELECT attempt_generation,source_kind FROM article_replay_publications WHERE publication_ref=$1`, p.Ref).Scan(&generation, &source); err != nil || generation != 0 || source != "verified_legacy_import_v1" {
		t.Fatal("exact S4.8 legacy publication missing")
	}
	if _, err := h.pool.Exec(h.ctx, `UPDATE article_replay_publications SET source_ref='rewrite' WHERE publication_ref=$1`, p.Ref); err == nil {
		t.Fatal("0011 allowed history rewrite")
	}
	t.Log("actual same RecoveryRunStore.BindLegacyPublication twice: original NULL once, exact nonNULL retry zero UPDATE, revision=2, original run full image unchanged")
}

func s413WriteSyntheticTombstone(raw []byte) (string, error) {
	dir, err := os.MkdirTemp(os.Getenv("TEMP"), "s413-deletion-media-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "committed-synthetic-tombstone.json")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := f.Write(raw)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return "", err
	}
	return path, nil
}

func s413PGBoundOrigins(t *testing.T) {
	for _, mode := range []string{"Genesis", "AnchoredMigration", "ApprovalRefusal", "PreparedRefusal", "WrongBinding", "MissingArticlePrefix", "AnyBegin"} {
		t.Run(mode, func(t *testing.T) { s413PGBoundOrigin(t, mode) })
	}
}

func s413PGBoundOrigin(t *testing.T, mode string) {
	h := s413NewPG(t, 11)
	x := h.world
	if mode != "Genesis" {
		x.m.Kind = "anchored_migration"
		head := recovery.JournalHead{Sequence: 1, Hash: s413Digest("independently-approved-old-article-chain")}
		x.m.Article.Committed, x.article.Head = head, head
		x.article.Prefixes = []recovery.JournalHead{{}, head}
	}
	x.approved = x.m
	x.newRuntime(t)
	if err := h.install(); err != nil {
		t.Fatalf("public Install: %v", err)
	}
	before := h.snapshot()
	if x.commit.Phase != "bound" || len(before.Runs) != 0 || len(x.journal.Entries) != 0 || x.starts != 0 {
		t.Fatal("approved bound zero-history origin not established")
	}
	if err := x.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := s413PGOriginMutation(h, mode)
	writes, media, coordinates := x.writes, x.media, x.coordinates
	commit, token := x.commit, x.commitToken
	x.newRuntime(t)
	err := x.runtime.RunNormal(h.ctx)
	if !errors.Is(err, want) {
		t.Errorf("bound %s normal restart: got %v want %v", mode, err, want)
	}
	if want == nil {
		s413PGOrdinaryLease(h)
	} else if x.starts != 0 {
		t.Error("invalid origin started ordinary lifecycle")
	}
	after := h.snapshot()
	stopErr := x.runtime.Stop(context.Background())
	if !reflect.DeepEqual(before, after) || x.writes != writes || x.media != media || x.coordinates != coordinates || x.commit != commit || x.commitToken != token || stopErr != nil {
		t.Error("normal origin verification wrote sources or failed Stop/join")
	}
	t.Logf("origin=%s public_install_bound=true zero_recovery_runs=%d normal_ok=%t ordinary_starts=%d store_unchanged=%t source_writes_unchanged=%t stop_joined=%t", mode, len(before.Runs), err == nil, x.starts, reflect.DeepEqual(before, after), x.writes == writes && x.media == media && x.coordinates == coordinates, stopErr == nil)
}

func s413PGOriginMutation(h *s413PG, mode string) error {
	x := h.world
	switch mode {
	case "ApprovalRefusal":
		x.approvalError = recovery.ErrAdmissionIncomplete
		return recovery.ErrAdmissionIncomplete
	case "PreparedRefusal":
		x.commit.Phase = "prepared"
		return recovery.ErrAdmissionBinding
	case "WrongBinding":
		x.commit.Manifest.TargetDigest = s413Digest("other-target")
		raw, err := recovery.EncodeInstallationManifest(x.commit.Manifest)
		if err != nil {
			h.t.Fatal(err)
		}
		x.commit.ManifestDigest = recovery.SHA256Digest(raw)
		x.approved = x.commit.Manifest
		return recovery.ErrAdmissionBinding
	case "MissingArticlePrefix":
		x.article.Prefixes = []recovery.JournalHead{{}}
		return recovery.ErrAdmissionIncomplete
	case "AnyBegin":
		x.journal.Entries = []recovery.JournalEntry{{Namespace: recovery.RecoveryNamespaceV1, RunID: "external-begun", BackupRef: "backup", KeyID: "retained", KeyVersion: "v1", SecretFingerprint: s413HMAC(), Stage: recovery.StageBegun, Sequence: 1, RecordedAt: s413Time, RecordHMAC: s413HMAC()}}
		x.journal.Head = recovery.JournalHead{Sequence: 1, Hash: s413Digest("external-begin")}
		x.journal.Prefixes = append(x.journal.Prefixes, x.journal.Head)
		x.journal.Revision = "external-begin-revision"
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

func s413PGOrdinaryLease(h *s413PG) {
	h.t.Helper()
	lease, err := h.world.runtime.Acquire(h.ctx, recovery.WorkRequest)
	if err != nil {
		h.t.Errorf("bound origin failed ordinary admission: %v", err)
		return
	}
	lease.Release()
}

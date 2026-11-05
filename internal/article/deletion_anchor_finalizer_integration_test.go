//go:build integration

package article

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pgA47Migrations = []struct{ name, digest string }{
	{"0001_shared.sql", "CEE72F1470B49185B3B33C7655F792ADFFCBC1E030AF6669E0902775104B7A14"},
	{"0002_identity.sql", "F4EC1005EA72B1774DF5E370C5ABA57CA3ED7A0B13099DBBE2D16AA66AEB9D50"},
	{"0003_article.sql", "FCAECE696DCE29D2F8D252EA36C4550F0C1736347E921F40703540B99E100D4A"},
	{"0004_idempotency.sql", "4794816AC2FDA4AD12522F2F324448A6F9EFB2E9E39677A595400E68148424A9"},
	{"0005_recovery_runs.sql", "66CA82109FF33BE7E4A3B349FE55A8CB88CBBF608939B8D7C5C12B06EDBDEC27"},
	{"0006_user_governance_indexes.sql", "6F93D99F6DFA0DE46DF8D891F8979039F679493932C28326EEB2AF28E2F24660"},
	{"0007_idempotency_response_snapshot_v1.sql", "9D3E642D4F5C5DE81113E23AEBD70E2E07BF583E140697D335DB4DDA35F01F9F"},
	{"0008_password_reset_durable.sql", "37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980"},
	{"0009_article_replay_attempt.sql", "4E10F50108A57D709C866044DE6678A00E11A4D6ACB589E708C7DC3779A9AA1D"},
}

type pgA47Target struct {
	Version                          int
	Database, User, Host, SearchPath string
	Port                             int
	BackendPID                       int
}

func pgA47VerifyConnection(ctx context.Context, c *pgx.Conn, path string, expectedPort uint16) (pgA47Target, error) {
	var v pgA47Target
	err := c.QueryRow(ctx, `SELECT current_setting('server_version_num')::int,current_database(),current_user,host(inet_server_addr()),inet_server_port(),current_setting('search_path'),pg_backend_pid()`).Scan(&v.Version, &v.Database, &v.User, &v.Host, &v.Port, &v.SearchPath, &v.BackendPID)
	if err != nil {
		return v, err
	}
	if v.Version != 180006 || v.Database != "han_b13c_article" || v.User != "han_b13c" || v.Host != "127.0.0.1" || v.Port != int(expectedPort) || v.SearchPath != path {
		return v, fmt.Errorf("PostgreSQL target/search_path differs from authorized isolated article fixture: actual=%+v expected_path=%q", v, path)
	}
	return v, nil
}

func pgA47Config(t *testing.T, path string) *pgxpool.Config {
	t.Helper()
	c, e := pgxpool.ParseConfig(testenv.RequirePostgresDSN(t))
	if e != nil {
		t.Fatal("parse explicitly authorized synthetic DSN")
	}
	if c.ConnConfig.Host != "127.0.0.1" || c.ConnConfig.Port == 0 || c.ConnConfig.Database != "han_b13c_article" || c.ConnConfig.User != "han_b13c" {
		t.Fatal("unauthorized PostgreSQL target")
	}
	c.ConnConfig.Fallbacks = nil
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	c.ConnConfig.RuntimeParams["search_path"] = path
	c.MaxConns = 4
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		v, e := pgA47VerifyConnection(ctx, conn, path, c.ConnConfig.Port)
		if e == nil {
			t.Logf("PG connection isolation: %+v", v)
		}
		return e
	}
	return c
}

func pgA47Pool(t *testing.T, ctx context.Context) (*pgxpool.Pool, string) {
	t.Helper()
	admin, e := pgxpool.NewWithConfig(ctx, pgA47Config(t, "pg_catalog"))
	if e != nil {
		t.Fatal("open isolated PG admin pool")
	}
	defer admin.Close()
	conn, e := admin.Acquire(ctx)
	if e != nil {
		t.Fatalf("verify isolated PG connection: %v", e)
	}
	conn.Release()
	var random [12]byte
	if _, e := rand.Read(random[:]); e != nil {
		t.Fatal(e)
	}
	schema := "han_b13c_a47_" + hex.EncodeToString(random[:])
	if _, e := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()+" AUTHORIZATION "+pgx.Identifier{"han_b13c"}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	t.Logf("RETAIN schema=%s test=%s", schema, t.Name())
	path := pgx.Identifier{schema}.Sanitize() + ", pg_catalog"
	pool, e := pgxpool.NewWithConfig(ctx, pgA47Config(t, path))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(pool.Close)
	return pool, schema
}

type pgA47Transactor struct {
	pool   *pgxpool.Pool
	schema string
	active atomic.Int32
	fault  atomic.Int32
}
type pgA47Tx struct {
	pgx.Tx
	owner      *pgA47Transactor
	once       sync.Once
	completion bool
}

func (p *pgA47Transactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, e := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if e != nil {
		return nil, e
	}
	ready := false
	defer func() {
		if !ready {
			_ = tx.Rollback(ctx)
		}
	}()
	path := pgx.Identifier{p.schema}.Sanitize() + ", pg_catalog"
	if _, e := tx.Exec(ctx, "SET LOCAL search_path TO "+path); e != nil {
		return nil, e
	}
	var actual []string
	if e := tx.QueryRow(ctx, "SELECT current_schemas(false)").Scan(&actual); e != nil {
		return nil, e
	}
	if len(actual) != 2 || actual[0] != p.schema || actual[1] != "pg_catalog" {
		return nil, fmt.Errorf("transaction schemas mismatch: %v", actual)
	}
	p.active.Add(1)
	ready = true
	return &pgA47Tx{Tx: tx, owner: p}, nil
}

func (p *pgA47Tx) finish()                            { p.once.Do(func() { p.owner.active.Add(-1) }) }
func (p *pgA47Tx) Rollback(ctx context.Context) error { defer p.finish(); return p.Tx.Rollback(ctx) }
func (p *pgA47Tx) Commit(ctx context.Context) error {
	if p.completion && p.owner.fault.CompareAndSwap(2, 0) {
		return errors.New("synthetic pre-commit refusal; real transaction will roll back")
	}
	defer p.finish()
	return p.Tx.Commit(ctx)
}
func (p *pgA47Tx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tag, e := p.Tx.Exec(ctx, sql, args...)
	if e != nil {
		return tag, e
	}
	if strings.Contains(sql, "SET status = 'completed'") {
		p.completion = true
	}
	if strings.Contains(sql, "SET status = 'journaled'") && p.owner.fault.CompareAndSwap(1, 0) {
		return tag, errors.New("synthetic binding result failure before commit")
	}
	return tag, nil
}

func pgA47Migrate(t *testing.T, ctx context.Context, txs *pgA47Transactor) {
	t.Helper()
	if _, e := txs.pool.Exec(ctx, "CREATE TABLE "+pgx.Identifier{txs.schema, "a47_migration_evidence"}.Sanitize()+" (name text PRIMARY KEY, digest text NOT NULL)"); e != nil {
		t.Fatal(e)
	}
	for _, m := range pgA47Migrations {
		raw, e := os.ReadFile(filepath.Join("..", "..", "migrations", m.name))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(raw)
		if strings.ToUpper(hex.EncodeToString(sum[:])) != m.digest {
			t.Fatalf("frozen migration drift: %s", m.name)
		}
		tx, e := txs.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		func() {
			defer tx.Rollback(ctx)
			if _, e = tx.Exec(ctx, string(raw)); e == nil {
				_, e = tx.Exec(ctx, "INSERT INTO "+pgx.Identifier{txs.schema, "a47_migration_evidence"}.Sanitize()+" VALUES ($1,$2)", m.name, m.digest)
			}
			if e == nil {
				e = tx.Commit(ctx)
			}
		}()
		if e != nil {
			t.Fatalf("apply frozen %s: %v", m.name, e)
		}
	}
	pgA47MigrationReadback(t, ctx, txs)
}

func pgA47MigrationReadback(t *testing.T, ctx context.Context, p *pgA47Transactor) {
	t.Helper()
	rows, e := p.pool.Query(ctx, "SELECT name,digest FROM "+pgx.Identifier{p.schema, "a47_migration_evidence"}.Sanitize()+" ORDER BY name")
	if rows != nil {
		defer rows.Close()
	}
	if e != nil {
		t.Fatal(e)
	}
	n := 0
	for rows.Next() {
		var name, digest string
		if e := rows.Scan(&name, &digest); e != nil {
			t.Fatal(e)
		}
		if n >= len(pgA47Migrations) || name != pgA47Migrations[n].name || digest != pgA47Migrations[n].digest {
			t.Fatal("migration readback mismatch")
		}
		n++
	}
	if e := rows.Err(); e != nil {
		t.Fatal(e)
	}
	if n != 9 {
		t.Fatalf("applied migrations=%d", n)
	}
	var foreign int
	if e := p.pool.QueryRow(ctx, `SELECT count(*) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND c.relname IN ('recovery_installations','recovery_runtime_metadata')`, p.schema).Scan(&foreign); e != nil {
		t.Fatal(e)
	}
	if foreign != 0 {
		t.Fatal("peer0010 unexpectedly present")
	}
	t.Logf("MIGRATIONS schema=%s fixed0001-0009 readback=%d", p.schema, n)
}

type pgA47Identity struct{}

func (pgA47Identity) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, id string) (DispositionDeletionCandidate, error) {
	v, e := (&identity.ArticleIdentityLifecycle{}).LockDeletionCandidate(ctx, tx, id)
	return DispositionDeletionCandidate{UserID: v.UserID, Deleted: v.Deleted}, e
}
func (pgA47Identity) LockTransferParties(context.Context, pgx.Tx, *string, string) (DispositionParties, error) {
	return DispositionParties{}, errors.New("this PG slice contains only an active ordinary author")
}
func (pgA47Identity) ClearRetainedEmailIfEligible(context.Context, pgx.Tx, string, *DispositionCompletionProof) (bool, error) {
	return false, errors.New("active-author fixture must not request identity cleanup")
}

type pgA47IDs struct{ next atomic.Int64 }

func (p *pgA47IDs) NewID() (string, error) {
	return fmt.Sprintf("66666666-6666-4666-8666-%012d", p.next.Add(1)), nil
}

type pgA47Fixture struct {
	t        *testing.T
	ctx      context.Context
	pool     *pgxpool.Pool
	observer *pgxpool.Pool
	schema   string
	txs      *pgA47Transactor
	journal  *anchoredArticleJournal
	world    *anchorRuntimeWorld
	anchor   *anchorCoreCAS
	media    deletionAnchorMedia
	handler  http.Handler
}

func newPGA47Fixture(t *testing.T) *pgA47Fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	pool, schema := pgA47Pool(t, ctx)
	f := &pgA47Fixture{t: t, ctx: ctx, pool: pool, schema: schema, txs: &pgA47Transactor{pool: pool, schema: schema}}
	config := pgA47Config(t, pgx.Identifier{schema}.Sanitize()+", pg_catalog")
	config.MaxConns = 1
	observer, e := pgxpool.NewWithConfig(ctx, config)
	if e != nil {
		t.Fatal(e)
	}
	f.observer = observer
	t.Cleanup(f.observer.Close)
	pgA47IndependentConnections(t, f)
	pgA47Migrate(t, ctx, f.txs)
	pgA47Seed(t, f)
	f.journal, f.world, f.anchor, f.media = actualAnchorCore(t, nil)
	t.Logf("RETAIN media=%s schema=%s", f.journal.options.Directory, schema)
	f.handler = f.newHandler()
	return f
}

func pgA47Seed(t *testing.T, f *pgA47Fixture) {
	t.Helper()
	tx, e := f.txs.Begin(f.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, `INSERT INTO users(id,username,email,phone,permanent_password_digest,permanent_password_generation,role,status,credential_mode,created_at,updated_at) VALUES ($1,'pg-a47-author','pg-a47@example.invalid','+15550000000','synthetic-credential-digest',0,'member','active','permanent',$2,$2)`, deletionAuthorID, deletionNow.Add(-time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	for kind, value := range map[string]string{"username": "pg-a47-author", "email": "pg-a47@example.invalid", "phone": "+15550000000"} {
		if _, e := tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at) VALUES($1,$2,$3,'active',$4)`, deletionAuthorID, kind, value, deletionNow.Add(-time.Hour)); e != nil {
			t.Fatal(e)
		}
	}
	if e := (Repository{}).InsertArticle(f.ctx, tx, deletionTestArticle()); e != nil {
		t.Fatal(e)
	}
	if e := tx.Commit(f.ctx); e != nil {
		t.Fatal(e)
	}
}

func (f *pgA47Fixture) newHandler() http.Handler {
	ids := &pgA47IDs{}
	clock := deletionClockStub{now: deletionNow}
	finalizer, e := NewDeletionFinalizer(DeletionFinalizerDependencies{Transactions: f.txs, Repository: Repository{}, Journal: f.journal, Auditor: audit.Store{}, AuditIDs: ids, Clock: clock, WorkerID: "synthetic-pg-a47", LeaseDuration: time.Minute, Identity: pgA47Identity{}})
	if e != nil {
		f.t.Fatal(e)
	}
	coordinator, e := NewPermanentDeletionCoordinator(PermanentDeletionDependencies{Transactions: f.txs, Repository: Repository{}, Journal: f.journal, Finalizer: finalizer, OperationIDs: ids, Clock: clock})
	if e != nil {
		f.t.Fatal(e)
	}
	service, e := NewCommandService(CommandDependencies{Transactions: f.txs, Repository: Repository{}, IDs: ids, Clock: clock, Deletion: coordinator})
	if e != nil {
		f.t.Fatal(e)
	}
	logger, e := NewSlogArticleResponseWriteLogger(slog.New(slog.NewJSONHandler(io.Discard, nil)), "han-pg-a47")
	if e != nil {
		f.t.Fatal(e)
	}
	handler, e := NewHandler(service, logger, service)
	if e != nil {
		f.t.Fatal(e)
	}
	guard, e := platform.NewRecoveryGuard(f.world.runtime)
	if e != nil {
		f.t.Fatal(e)
	}
	auth, e := platform.NewHTTPAuthorizer(createArticlePrincipalResolver{principal: deletionAuthorPrincipal()}, func() time.Time { return deletionNow })
	if e != nil {
		f.t.Fatal(e)
	}
	return platform.TraceMiddleware(guard.Protect(auth.Protect(session.AccessGeneral, http.HandlerFunc(handler.DeleteOrdinary))))
}

type pgA47Snapshot struct {
	Articles, Pending, Audits, Completed, BoundAudit int
	Status                                           string
	Sequence                                         int64
}

func (f *pgA47Fixture) snapshot() (pgA47Snapshot, error) {
	var s pgA47Snapshot
	tx, e := f.observer.BeginTx(f.ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if e != nil {
		return s, e
	}
	defer tx.Rollback(f.ctx)
	if _, e := tx.Exec(f.ctx, "SET LOCAL search_path TO "+pgx.Identifier{f.schema}.Sanitize()+", pg_catalog"); e != nil {
		return s, e
	}
	a := pgx.Identifier{f.schema, "articles"}.Sanitize()
	o := pgx.Identifier{f.schema, "deletion_operations"}.Sanitize()
	u := pgx.Identifier{f.schema, "audit_events"}.Sanitize()
	query := fmt.Sprintf(`SELECT (SELECT count(*) FROM %s WHERE id=$1),(SELECT count(*) FROM %s WHERE id=$1 AND deletion_state='deletion_pending'),(SELECT count(*) FROM %s WHERE target_id=$1::text),(SELECT count(*) FROM %s WHERE article_id=$1 AND status='completed'),COALESCE((SELECT status FROM %s WHERE article_id=$1),''),COALESCE((SELECT tombstone_sequence FROM %s WHERE article_id=$1),0),(SELECT count(*) FROM %s o JOIN %s a ON a.id=o.audit_event_id WHERE o.article_id=$1 AND o.status='completed' AND a.target_id=$1::text AND a.action='article.delete' AND a.outcome='succeeded')`, a, a, u, o, o, o, o, u)
	if e := tx.QueryRow(f.ctx, query, deletionArticleID).Scan(&s.Articles, &s.Pending, &s.Audits, &s.Completed, &s.Status, &s.Sequence, &s.BoundAudit); e != nil {
		return s, e
	}
	return s, tx.Commit(f.ctx)
}

type pgA47Writer struct {
	*httptest.ResponseRecorder
	fixture         *pgA47Fixture
	seen            atomic.Int32
	completionError error
}

func (w *pgA47Writer) WriteHeader(code int) {
	if code == http.StatusNoContent {
		s, e := w.fixture.snapshot()
		a, _, ae := w.fixture.anchor.Load(w.fixture.ctx, ArticleDeletionNamespace)
		if e != nil || ae != nil || s.Articles != 0 || s.Audits != 1 || s.Completed != 1 || s.BoundAudit != 1 || a.Pending != nil || a.Committed.Sequence != 1 {
			w.completionError = fmt.Errorf("premature204 sql=%+v sqlerr=%v anchor=%+v anchorerr=%v", s, e, a, ae)
		}
	}
	w.seen.Store(int32(code))
	w.ResponseRecorder.WriteHeader(code)
}

func (f *pgA47Fixture) request(ctx context.Context) *pgA47Writer {
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/articles/"+deletionArticleID, nil).WithContext(ctx)
	r.Header.Set("If-Match", deletionRequest(DeleteOrdinary).IfMatch)
	r.Header.Set(platform.CSRFHeaderName, "csrf")
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	w := &pgA47Writer{ResponseRecorder: httptest.NewRecorder(), fixture: f}
	f.handler.ServeHTTP(w, r)
	return w
}

func (f *pgA47Fixture) assertResponse(w *pgA47Writer, code int, complete bool) {
	f.t.Helper()
	if w.seen.Load() != int32(code) || w.completionError != nil {
		f.t.Fatalf("response=%d want=%d early=%v body=%s", w.seen.Load(), code, w.completionError, w.Body.String())
	}
	s, e := f.snapshot()
	if e != nil {
		f.t.Fatal(e)
	}
	if complete {
		if s.Articles != 0 || s.Audits != 1 || s.Completed != 1 || s.BoundAudit != 1 || s.Sequence != 1 {
			f.t.Fatalf("SQL completion=%+v", s)
		}
	} else if s.Articles != 1 || s.Pending != 1 || s.Audits != 0 || s.Completed != 0 {
		f.t.Fatalf("failure physically deleted or completed: %+v", s)
	}
	if f.txs.active.Load() != 0 || f.pool.Stat().AcquiredConns() != 0 || f.observer.Stat().AcquiredConns() != 0 {
		f.t.Fatal("transaction/connection not returned after actual handler join")
	}
	raw, _ := json.Marshal(s)
	f.t.Logf("SQL ORACLE schema=%s response=%d snapshot=%s", f.schema, code, raw)
}

func pgA47IndependentConnections(t *testing.T, f *pgA47Fixture) {
	t.Helper()
	one, e := f.pool.Acquire(f.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer one.Release()
	two, e := f.observer.Acquire(f.ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer two.Release()
	var a, b int
	if e := one.QueryRow(f.ctx, "SELECT pg_backend_pid()").Scan(&a); e != nil {
		t.Fatal(e)
	}
	if e := two.QueryRow(f.ctx, "SELECT pg_backend_pid()").Scan(&b); e != nil {
		t.Fatal(e)
	}
	if a == b {
		t.Fatal("SQL oracle reused writer connection")
	}
	t.Logf("INDEPENDENT schema=%s writer_pid=%d observer_pid=%d", f.schema, a, b)
}

func TestAnchoredArticleFinalizerPostgres(t *testing.T) {
	t.Run("PauseStagePendingPublishCommit", func(t *testing.T) {
		for _, name := range []string{"HealthyDualCompletion", "PendingBarrierZeroDelete", "PublishedBarrierZeroDelete", "CommittedBeforeHeadAdvanceZeroDelete", "PendingCASAckUnknownZeroDelete", "CommittedCASAckUnknownZeroDelete", "CommittedLoadUnknownZeroDelete", "HeadAdvanceRejectedZeroDelete", "BindFailureThenRetry", "AuditFailureRollsBackDelete", "CompletionFailureRollsBackDeleteAudit", "CommitFailureRollsBackDeleteAudit", "AfterAcceptedCancelKeepsJournalNoSqlCompletion"} {
			if !t.Run(name, func(t *testing.T) { pgA47Case(t, name) }) {
				return
			}
		}
	})
}

func pgA47Case(t *testing.T, name string) {
	f := newPGA47Fixture(t)
	if strings.Contains(name, "Barrier") || name == "CommittedBeforeHeadAdvanceZeroDelete" {
		pgA47Barrier(t, f, name)
		return
	}
	if name == "HealthyDualCompletion" {
		f.assertResponse(f.request(f.ctx), http.StatusNoContent, true)
		return
	}
	if name == "AfterAcceptedCancelKeepsJournalNoSqlCompletion" {
		pgA47AfterAcceptedCancel(t, f)
		return
	}
	pgA47ConfigureFailure(t, f, name)
	f.assertResponse(f.request(f.ctx), http.StatusServiceUnavailable, false)
	if strings.Contains(name, "CASAck") || strings.Contains(name, "LoadUnknown") || name == "HeadAdvanceRejectedZeroDelete" {
		if lease, e := f.world.runtime.Acquire(f.ctx, recovery.WorkRequest); e == nil {
			lease.Release()
			t.Fatal("Journal unknown admitted new request")
		}
		observed, revision, e := f.anchor.Load(f.ctx, ArticleDeletionNamespace)
		if e != nil {
			t.Fatal(e)
		}
		encoded, _ := json.Marshal(observed)
		t.Logf("UNKNOWN schema=%s phase=%s revision=%s anchor=%s", f.schema, name, revision, encoded)
	}
	if name == "BindFailureThenRetry" {
		f.assertResponse(f.request(f.ctx), http.StatusNoContent, true)
	}
}

func pgA47ConfigureFailure(t *testing.T, f *pgA47Fixture, name string) {
	t.Helper()
	switch name {
	case "PendingCASAckUnknownZeroDelete":
		f.journal.options.Anchor = &anchorFaultCAS{inner: f.anchor, target: "pending-cas", side: "after", mode: "error"}
	case "CommittedCASAckUnknownZeroDelete":
		f.journal.options.Anchor = &anchorFaultCAS{inner: f.anchor, target: "commit-cas", side: "after", mode: "error"}
	case "CommittedLoadUnknownZeroDelete":
		f.journal.options.Anchor = &anchorFaultCAS{inner: f.anchor, target: "commit-load", side: "before", mode: "error"}
	case "HeadAdvanceRejectedZeroDelete":
		f.journal.options.Runtime = pgA47Completion{Runtime: f.world.runtime, reject: true}
	case "BindFailureThenRetry":
		f.txs.fault.Store(1)
	case "CommitFailureRollsBackDeleteAudit":
		f.txs.fault.Store(2)
	case "AuditFailureRollsBackDelete":
		if _, e := f.pool.Exec(f.ctx, "ALTER TABLE "+pgx.Identifier{f.schema, "audit_events"}.Sanitize()+" ADD CONSTRAINT synthetic_reject_success CHECK (outcome <> 'succeeded')"); e != nil {
			t.Fatal(e)
		}
	case "CompletionFailureRollsBackDeleteAudit":
		if _, e := f.pool.Exec(f.ctx, "ALTER TABLE "+pgx.Identifier{f.schema, "deletion_operations"}.Sanitize()+" ADD CONSTRAINT synthetic_reject_completion CHECK (status <> 'completed')"); e != nil {
			t.Fatal(e)
		}
	default:
		t.Fatalf("unknown PG scenario %s", name)
	}
}

type pgA47Completion struct {
	recovery.Runtime
	reject  bool
	entered chan struct{}
	release <-chan struct{}
	cancel  context.CancelFunc
}

func (p pgA47Completion) CompleteArticleAppend(ctx context.Context, b recovery.AppendBarrier, a recovery.VerifiedHeadAdvance) error {
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	if p.reject {
		return errors.New("synthetic notification rejection before acceptance")
	}
	e := p.Runtime.CompleteArticleAppend(ctx, b, a)
	if e == nil && p.cancel != nil {
		p.cancel()
	}
	return e
}

type pgA47BarrierCAS struct {
	recovery.JournalHeadAnchor
	entered chan struct{}
	release <-chan struct{}
}

func (p pgA47BarrierCAS) CompareAndSwap(ctx context.Context, ns, r string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	if next.Pending == nil {
		close(p.entered)
		<-p.release
	}
	return p.JournalHeadAnchor.CompareAndSwap(ctx, ns, r, next)
}

func pgA47Barrier(t *testing.T, f *pgA47Fixture, name string) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	pgA47ConfigureBarrier(f, name, entered, release)
	done := make(chan *pgA47Writer, 1)
	var requests sync.WaitGroup
	requests.Add(1)
	go func() { defer requests.Done(); done <- f.request(f.ctx) }()
	joined := false
	defer func() {
		unblock()
		requests.Wait()
		if !joined {
			<-done
		}
	}()
	select {
	case <-entered:
	case w := <-done:
		joined = true
		requests.Wait()
		t.Fatalf("handler ended before barrier: %d", w.seen.Load())
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	pgA47ObserveBarrier(t, f, name)
	select {
	case <-done:
		joined = true
		requests.Wait()
		t.Fatal("handler returned before final acceptance")
	default:
	}
	unblock()
	w := <-done
	requests.Wait()
	joined = true
	f.assertResponse(w, http.StatusNoContent, true)
}

func pgA47AfterAcceptedCancel(t *testing.T, f *pgA47Fixture) {
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	f.journal.options.Runtime = pgA47Completion{Runtime: f.world.runtime, cancel: cancel}
	f.assertResponse(f.request(ctx), http.StatusServiceUnavailable, false)
	a, _, e := f.anchor.Load(f.ctx, ArticleDeletionNamespace)
	if e != nil || a.Committed.Sequence != 1 || a.Pending != nil {
		t.Fatalf("accepted fact changed: %+v %v", a, e)
	}
	l, e := f.world.runtime.Acquire(f.ctx, recovery.WorkRequest)
	if e != nil {
		t.Fatalf("known accepted success became unknown: %v", e)
	}
	l.Release()
}

func pgA47ObserveBarrier(t *testing.T, f *pgA47Fixture, name string) {
	s, e := f.snapshot()
	if e != nil {
		t.Fatal(e)
	}
	if s.Articles != 1 || s.Pending != 1 || s.Audits != 0 || s.Completed != 0 || s.Status != "pending_journal" {
		t.Fatalf("before final acceptance SQL=%+v", s)
	}
	a, r, e := f.anchor.Load(f.ctx, ArticleDeletionNamespace)
	if e != nil {
		t.Fatal(e)
	}
	if name == "CommittedBeforeHeadAdvanceZeroDelete" {
		if a.Committed.Sequence != 1 || a.Pending != nil {
			t.Fatal("head-advance barrier lacks committed H1")
		}
	} else if a.Committed.Sequence != 0 || a.Pending == nil || a.Pending.NewHead.Sequence != 1 {
		t.Fatal("publish/commit barrier lacks exact pending H0-to-H1")
	}
	encoded, _ := json.Marshal(a)
	t.Logf("BARRIER schema=%s phase=%s revision=%s anchor=%s sql=%+v zero204=true", f.schema, name, r, encoded, s)
}

func pgA47ConfigureBarrier(f *pgA47Fixture, name string, entered chan struct{}, release <-chan struct{}) {
	switch name {
	case "PendingBarrierZeroDelete":
		f.journal.media.link = func(ctx context.Context, a, b string) error {
			close(entered)
			<-release
			return f.media.link(ctx, a, b)
		}
	case "PublishedBarrierZeroDelete":
		f.journal.options.Anchor = pgA47BarrierCAS{JournalHeadAnchor: f.anchor, entered: entered, release: release}
	case "CommittedBeforeHeadAdvanceZeroDelete":
		f.journal.options.Runtime = pgA47Completion{Runtime: f.world.runtime, entered: entered, release: release}
	}
}

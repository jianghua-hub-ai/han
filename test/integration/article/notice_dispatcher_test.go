//go:build integration

package articleintegration

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/privacy"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const noticeTrace = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

// All schemas are retained. Closing runtime resources never drops test data.
type noticeHarness struct {
	t                 *testing.T
	ctx               context.Context
	cancel            context.CancelFunc
	pool              *pgxpool.Pool
	schema            string
	group             sync.WaitGroup
	gate              *noticeGate
	log               *noticeLog
	expectWorkerError bool
}

func noticeMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		var sql interface{ SQLState() string }
		if errors.As(err, &sql) {
			t.Fatalf("operation failed: type=%T SQLSTATE=%s", err, sql.SQLState())
		}
		t.Fatalf("operation failed: type=%T (raw configuration/protocol error withheld)", err)
	}
}

func noticeUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("test entropy unavailable")
	}
	b[6], b[8] = (b[6]&15)|64, (b[8]&63)|128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type noticeIDs struct{}

func (noticeIDs) NewID() (string, error) { return noticeUUID(), nil }

func noticePoolConfig(t *testing.T, schema string) *pgxpool.Config {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN missing; no skip")
	}
	c, err := pgxpool.ParseConfig(dsn)
	noticeMust(t, err)
	ip := net.ParseIP(c.ConnConfig.Host)
	if ip == nil || !ip.IsLoopback() || c.ConnConfig.Port != 5433 {
		t.Fatal("test DSN must target the authorized loopback:5433")
	}
	c.MaxConns = 4
	c.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	return c
}

func newNoticeHarness(t *testing.T, emptyGate ...bool) *noticeHarness {
	t.Helper()
	h := &noticeHarness{t: t, schema: "han_s72_" + strings.ReplaceAll(noticeUUID(), "-", ""), log: &noticeLog{}}
	h.ctx, h.cancel = context.WithTimeout(context.Background(), 90*time.Second)
	p, err := pgxpool.NewWithConfig(h.ctx, noticePoolConfig(t, h.schema))
	noticeMust(t, err)
	h.pool = p
	t.Cleanup(func() { h.cancel(); h.group.Wait(); h.pool.Close() })
	var addr string
	var port, version int
	noticeMust(t, p.QueryRow(h.ctx, "SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int").Scan(&addr, &port, &version))
	if !net.ParseIP(addr).IsLoopback() || port != 5433 || version != 180006 {
		t.Fatal("unexpected PostgreSQL endpoint/version")
	}
	h.migrate()
	h.gate = &noticeGate{h: h}
	if len(emptyGate) == 0 || !emptyGate[0] {
		h.gateRecord("open")
	}
	t.Logf("retained_schema=%s", h.schema)
	return h
}

func (h *noticeHarness) migrate() {
	h.t.Helper()
	tx, err := h.pool.Begin(h.ctx)
	noticeMust(h.t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(h.ctx, "CREATE SCHEMA "+pgx.Identifier{h.schema}.Sanitize())
	noticeMust(h.t, err)
	for _, name := range []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		noticeMust(h.t, err)
		_, err = tx.Exec(h.ctx, string(body))
		noticeMust(h.t, err)
	}
	noticeMust(h.t, tx.Commit(h.ctx))
}

func (h *noticeHarness) exec(sql string, args ...any) {
	h.t.Helper()
	_, err := h.pool.Exec(h.ctx, sql, args...)
	noticeMust(h.t, err)
}
func (h *noticeHarness) scalar(sql string, args ...any) int64 {
	h.t.Helper()
	var n int64
	noticeMust(h.t, h.pool.QueryRow(h.ctx, sql, args...).Scan(&n))
	return n
}
func (h *noticeHarness) now() time.Time {
	h.t.Helper()
	var n time.Time
	noticeMust(h.t, h.pool.QueryRow(h.ctx, "SELECT clock_timestamp()").Scan(&n))
	return n.UTC().Truncate(time.Microsecond)
}
func (h *noticeHarness) gateRecord(state string) {
	h.exec(`INSERT INTO processing_gate_records(region_ref,controller_ref,legal_basis_ref,smtp_processor_ref,effective_at,recorded_by) VALUES($1,'synthetic-controller','synthetic-basis','synthetic-processor',clock_timestamp()-interval '1 second','s72-fixture')`, "synthetic-region-"+state)
}

func noticeGateConfig() platform.ProcessingGateConfig {
	return platform.ProcessingGateConfig{RegionRef: "synthetic-region-open", ControllerRef: "synthetic-controller", LegalBasisRef: "synthetic-basis", SMTPProcessorRef: "synthetic-processor"}
}

type noticeGate struct {
	h      *noticeHarness
	mu     sync.Mutex
	seen   []pgx.Tx
	hook   func(context.Context, pgx.Tx, int) error
	config *platform.ProcessingGateConfig
}

func (g *noticeGate) RequireOpen(ctx context.Context) error { return g.check(ctx, g.h.pool) }
func (g *noticeGate) check(ctx context.Context, q privacy.QueryRower) error {
	s, err := privacy.NewStore(q)
	if err != nil {
		return err
	}
	cfg := noticeGateConfig()
	if g.config != nil {
		cfg = *g.config
	}
	p, err := privacy.NewProcessingGate(cfg, s)
	if err != nil {
		return err
	}
	return p.RequireOpen(ctx)
}
func (g *noticeGate) RequireOpenInTx(ctx context.Context, tx pgx.Tx) error {
	g.mu.Lock()
	g.seen = append(g.seen, tx)
	n := len(g.seen)
	g.mu.Unlock()
	if g.hook != nil {
		if err := g.hook(ctx, tx, n); err != nil {
			return err
		}
	}
	return g.check(ctx, tx)
}

func (h *noticeHarness) seed(max int64, both bool) string {
	h.t.Helper()
	tx, err := h.pool.Begin(h.ctx)
	noticeMust(h.t, err)
	defer tx.Rollback(context.Background())
	ids := [2]string{noticeUUID(), noticeUUID()}
	now := h.now()
	for i, id := range ids {
		_, err = tx.Exec(h.ctx, `INSERT INTO users(id,username,email,phone,role,status,credential_mode,created_at,updated_at) VALUES($1,$2,$3,$4,'member','active','permanent',$5,$5)`, id, "user-"+id, fmt.Sprintf("s72-%d@example.test", i), "synthetic-"+id, now)
		noticeMust(h.t, err)
		if i == 1 {
			_, err = tx.Exec(h.ctx, `UPDATE users SET status='deleted',email=NULL,phone=NULL,retained_email='former@example.test',deleted_at=$2 WHERE id=$1`, id, now)
			noticeMust(h.t, err)
		}
	}
	id := noticeUUID()
	n := article.TransferNotice{ID: id, ArticleRef: noticeUUID(), NewAuthorID: ids[0], FormerDeletedUserID: ids[1], CreatedAt: now, UpdatedAt: now}
	r := [2]article.TransferNoticeRecipient{}
	for i, kind := range []article.RecipientKind{article.RecipientNewAuthor, article.RecipientFormerDeletedAuthor} {
		r[i] = article.TransferNoticeRecipient{NoticeID: id, Kind: kind, EmailSnapshot: fmt.Sprintf("snapshot-%d@example.test", i), Status: article.RecipientPending, MaxAttempts: max, CreatedAt: now.Add(time.Duration(i) * time.Microsecond), UpdatedAt: now}
	}
	if !both {
		r[1].Status = article.RecipientFailedTerminal
	}
	noticeMust(h.t, (article.Repository{}).InsertTransferNotice(h.ctx, tx, n, r))
	noticeMust(h.t, tx.Commit(h.ctx))
	return id
}

func (h *noticeHarness) recipient(id string, kind article.RecipientKind) article.TransferNoticeRecipient {
	h.t.Helper()
	tx, err := h.pool.Begin(h.ctx)
	noticeMust(h.t, err)
	defer tx.Rollback(context.Background())
	_, r, found, err := (article.Repository{}).ReadNoticeRecipient(h.ctx, tx, id, kind)
	noticeMust(h.t, err)
	if !found {
		h.t.Fatal("recipient missing")
	}
	return r
}
func noticeConfig() article.NoticeDispatcherConfig {
	return article.NoticeDispatcherConfig{PollInterval: 10 * time.Millisecond, BatchLimit: 2, LeaseDuration: 6 * time.Second, DBOperationBudget: time.Second, BackoffInitial: 20 * time.Millisecond, BackoffMax: time.Second, SMTPOperationBudget: time.Second}
}

type noticeSendFunc func(context.Context, mailer.Message) error

func (f noticeSendFunc) Send(ctx context.Context, m mailer.Message) error { return f(ctx, m) }

type noticeLog struct {
	mu      sync.Mutex
	buckets []string
	values  []any
}

func (l *noticeLog) Log(_ context.Context, _ slog.Level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.values = append(l.values, msg)
	l.values = append(l.values, args...)
	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == "reason_bucket" {
			l.buckets = append(l.buckets, fmt.Sprint(args[i+1]))
		}
	}
}
func (l *noticeLog) has(s string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, v := range l.buckets {
		if v == s {
			return true
		}
	}
	return false
}

func (h *noticeHarness) dispatcher(m mailer.Mailer, options ...func(*article.NoticeDispatcherDependencies, *article.NoticeDispatcherConfig)) *article.TransferNoticeDispatcher {
	h.t.Helper()
	cfg := noticeConfig()
	deps := article.NoticeDispatcherDependencies{Transactions: h.pool, Repository: article.Repository{}, Gate: h.gate, Mailer: m, Audit: audit.Store{}, IDs: noticeIDs{}, NewTraceID: func() (string, error) { return noticeTrace, nil }, Logger: h.log, WorkerID: noticeUUID()}
	for _, o := range options {
		o(&deps, &cfg)
	}
	d, err := article.NewTransferNoticeDispatcher(deps, cfg)
	noticeMust(h.t, err)
	h.t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		d.BeginShutdown(c)
		d.Stop()
		err := d.Wait(c)
		if err != nil && !h.expectWorkerError {
			noticeMust(h.t, err)
		}
	})
	return d
}

func TestNoticeDispatcherPostgres(t *testing.T) {
	t.Run("HappyPath", testNoticeHappy)
	t.Run("GateClosed", testNoticeGateClosed)
	t.Run("GateSameTransaction", testNoticeGateSameTransaction)
	t.Run("MaxAttemptGateDeferred", testNoticeGateDeferred)
	t.Run("GateAbortedTransaction", testNoticeGateAborted)
	t.Run("ClaimSkipLocked", testNoticeSkipLocked)
	t.Run("MultiWorker", testNoticeMultiWorker)
	t.Run("MaintenanceDueBoundary", testNoticeMaintenance)
	t.Run("ClaimCommitUnknown", testNoticeClaimUnknown)
	t.Run("ResultAuditAtomicity", testNoticeAuditAtomicity)
	t.Run("BigIntCommitUnknown", testNoticeBigInt)
	t.Run("LateRealResult", testNoticeLateResult)
	t.Run("LeaseUTCAndABA", testNoticeABA)
	t.Run("SnapshotAfterAccountArticleChanges", testNoticeSnapshot)
	t.Run("ResultCommitRollbackAndGetFailure", testNoticeResultFaults)
	t.Run("FreshDBTimeAfterLockWait", testNoticeFreshTime)
	t.Run("ExactFencing", testNoticeExactFence)
}

func testNoticeHappy(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	var calls atomic.Int64
	d := h.dispatcher(noticeSendFunc(func(ctx context.Context, m mailer.Message) error {
		if m.Purpose != mailer.PurposeOrphanArticleTransferNotice {
			return errors.New("unexpected purpose")
		}
		calls.Add(1)
		return nil
	}))
	noticeMust(t, d.Reconcile(h.ctx))
	for _, kind := range []article.RecipientKind{article.RecipientNewAuthor, article.RecipientFormerDeletedAuthor} {
		r := h.recipient(id, kind)
		if r.Status != article.RecipientAccepted || r.AttemptCount != 1 || r.AcceptedAt == nil || r.ClaimedBy != nil || r.LockedUntil != nil {
			t.Fatal("happy path state/fencing mismatch")
		}
	}
	if calls.Load() != 2 || h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.accepted'") != 2 {
		t.Fatal("happy path send/audit count mismatch")
	}
	if h.gate.seen[0] == h.gate.seen[1] {
		t.Fatal("claim/preflight must use distinct transactions")
	}
}

// Wrappers inject transport/ack faults; all successful SQL is executed by pgx.
type noticeTransactions struct {
	pool      *pgxpool.Pool
	hook      func(context.Context, *noticeTx) error
	beginHook func(context.Context) error
}
type noticeTx struct {
	pgx.Tx
	owner *noticeTransactions
	phase string
}

func (x *noticeTransactions) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if x.beginHook != nil {
		if err := x.beginHook(ctx); err != nil {
			return nil, err
		}
	}
	if o.IsoLevel != pgx.ReadCommitted {
		return nil, errors.New("unexpected isolation")
	}
	tx, err := x.pool.BeginTx(ctx, o)
	if err != nil {
		return nil, err
	}
	return &noticeTx{Tx: tx, owner: x}, nil
}
func (tx *noticeTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	if strings.Contains(q, "SET status='processing',attempt_count") {
		tx.phase = "claim"
	}
	if strings.Contains(q, "SET status=$7") {
		tx.phase = "result"
	}
	return tx.Tx.QueryRow(ctx, q, args...)
}
func (tx *noticeTx) Commit(ctx context.Context) error {
	if tx.owner.hook != nil {
		return tx.owner.hook(ctx, tx)
	}
	return tx.Tx.Commit(ctx)
}

// Keep compile-time contracts explicit without touching production interfaces.
var _ article.NoticeTransactions = (*noticeTransactions)(nil)

func (h *noticeHarness) waitTime(deadline time.Time) {
	h.t.Helper()
	for {
		now := h.now()
		if !now.Before(deadline) {
			return
		}
		timer := time.NewTimer(deadline.Sub(now) + time.Millisecond)
		select {
		case <-timer.C:
		case <-h.ctx.Done():
			timer.Stop()
			h.t.Fatal("database time deadline exceeded")
		}
	}
}

func (h *noticeHarness) assertState(id string, status article.RecipientStatus, attempt int64, bucket string) article.TransferNoticeRecipient {
	h.t.Helper()
	r := h.recipient(id, article.RecipientNewAuthor)
	b := ""
	if r.FailureBucket != nil {
		b = *r.FailureBucket
	}
	if r.Status != status || r.AttemptCount != attempt || b != bucket {
		h.t.Fatalf("recipient mismatch: state=%s attempt=%d bucket=%s", r.Status, r.AttemptCount, b)
	}
	return r
}
func testNoticeGateClosed(t *testing.T) {
	for _, name := range []string{"missing_record", "mismatch", "region", "controller", "basis", "processor"} {
		t.Run(name, func(t *testing.T) {
			h := newNoticeHarness(t, name == "missing_record")
			id := h.seed(3, false)
			cfg := noticeGateConfig()
			switch name {
			case "mismatch":
				h.gateRecord("closed")
			case "region":
				cfg.RegionRef = ""
			case "controller":
				cfg.ControllerRef = ""
			case "basis":
				cfg.LegalBasisRef = ""
			case "processor":
				cfg.SMTPProcessorRef = ""
			}
			h.gate.config = &cfg
			var calls atomic.Int64
			d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }))
			noticeMust(t, d.Reconcile(h.ctx))
			h.assertState(id, article.RecipientPending, 0, "")
			if calls.Load() != 0 {
				t.Fatal("closed Gate connected SMTP")
			}
		})
	}
}
func testNoticeGateSameTransaction(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	h.gate.hook = func(ctx context.Context, tx pgx.Tx, n int) error {
		_, err := tx.Exec(ctx, `INSERT INTO processing_gate_records(region_ref,controller_ref,legal_basis_ref,smtp_processor_ref,effective_at,recorded_by) VALUES('private-uncommitted','synthetic-controller','synthetic-basis','synthetic-processor',now()-interval '1 second','fixture')`)
		if err != nil {
			return err
		}
		if err := h.gate.RequireOpen(ctx); err != nil {
			return err
		}
		return nil
	}
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { return errors.New("unexpected send") }))
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientPending, 0, "")
	if h.scalar("SELECT count(*) FROM processing_gate_records") != 1 {
		t.Fatal("closed claim transaction was not rolled back")
	}
}
func testNoticeGateDeferred(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(1, false)
	var calls atomic.Int64
	x := &noticeTransactions{pool: h.pool}
	x.hook = func(ctx context.Context, tx *noticeTx) error {
		err := tx.Tx.Commit(ctx)
		if err == nil && tx.phase == "claim" {
			h.gateRecord("closed")
		}
		return err
	}
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
		d.Transactions = x
		c.BackoffInitial = time.Second
	})
	noticeMust(t, d.Reconcile(h.ctx))
	r := h.assertState(id, article.RecipientRetryWait, 1, "processing_gate_closed")
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientRetryWait, 1, "processing_gate_closed")
	h.waitTime(*r.NextAttemptAt)
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientFailedTerminal, 1, "attempts_exhausted")
	if calls.Load() != 0 || h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.duplicate_risk'") != 0 {
		t.Fatal("known-unsent deferred exhausted work acquired send/risk")
	}
}
func testNoticeGateAborted(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	var calls atomic.Int64
	h.gate.hook = func(ctx context.Context, tx pgx.Tx, n int) error {
		if n == 2 {
			_, err := tx.Exec(ctx, "SELECT 1/0")
			return err
		}
		return nil
	}
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }))
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientRetryWait, 1, "processing_gate_closed")
	if calls.Load() != 0 || h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.retry_scheduled'") != 1 {
		t.Fatal("aborted preflight failed fenced deferred save")
	}
}
func testNoticeSkipLocked(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	tx, err := h.pool.Begin(h.ctx)
	noticeMust(t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(h.ctx, "SELECT 1 FROM transfer_notice_recipients WHERE notice_id=$1 AND recipient_kind='new_author' FOR UPDATE", id)
	noticeMust(t, err)
	var calls atomic.Int64
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }))
	noticeMust(t, d.Reconcile(h.ctx))
	if calls.Load() != 1 || h.recipient(id, article.RecipientFormerDeletedAuthor).Status != article.RecipientAccepted {
		t.Fatal("SKIP LOCKED did not advance independent row")
	}
	noticeMust(t, tx.Rollback(h.ctx))
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientAccepted, 1, "")
}
func testNoticeMultiWorker(t *testing.T) {
	h := newNoticeHarness(t)
	for i := 0; i < 3; i++ {
		h.seed(3, true)
	}
	var calls atomic.Int64
	start := make(chan struct{})
	out := make(chan error, 4)
	for i := 0; i < 4; i++ {
		d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }))
		h.group.Add(1)
		go func() { defer h.group.Done(); <-start; out <- d.Reconcile(h.ctx) }()
	}
	close(start)
	for i := 0; i < 4; i++ {
		noticeMust(t, <-out)
	}
	if calls.Load() != 6 || h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE status='accepted' AND attempt_count=1") != 6 {
		t.Fatal("multi-worker duplicate/missing claim")
	}
}
func testNoticeMaintenance(t *testing.T) {
	for _, state := range []string{"pending", "processing", "retry_wait", "accepted", "failed_terminal"} {
		for _, due := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_due_%t", state, due), func(t *testing.T) {
				h := newNoticeHarness(t)
				id := h.seed(1, false)
				h.gateRecord("closed")
				when := h.now().Add(time.Hour)
				if due {
					when = h.now().Add(-time.Second)
				}
				h.exec(`UPDATE transfer_notice_recipients SET status=$2,attempt_count=1,claimed_by=CASE WHEN $2='processing' THEN 'fixture-owner' END,locked_until=CASE WHEN $2='processing' THEN $3::timestamptz END,last_attempt_at=clock_timestamp()-interval '2 seconds',next_attempt_at=CASE WHEN $2='retry_wait' THEN $3::timestamptz END,accepted_at=CASE WHEN $2='accepted' THEN clock_timestamp() END WHERE notice_id=$1 AND recipient_kind='new_author'`, id, state, when)
				before := h.recipient(id, article.RecipientNewAuthor)
				d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { return errors.New("unexpected send") }))
				noticeMust(t, d.Reconcile(h.ctx))
				after := h.recipient(id, article.RecipientNewAuthor)
				finish := state == "pending" || (due && (state == "processing" || state == "retry_wait"))
				if finish {
					if after.Status != article.RecipientFailedTerminal || after.AttemptCount != 1 {
						t.Fatal("exhausted work failed to converge")
					}
				} else if !reflect.DeepEqual(before, after) {
					t.Fatal("maintenance changed not-due/terminal row")
				}
			})
		}
	}
}
func testNoticeClaimUnknown(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("committed_%t", committed), func(t *testing.T) {
			h := newNoticeHarness(t)
			id := h.seed(3, false)
			var calls atomic.Int64
			x := &noticeTransactions{pool: h.pool}
			x.hook = func(ctx context.Context, tx *noticeTx) error {
				if tx.phase == "claim" {
					if committed {
						noticeMust(t, tx.Tx.Commit(ctx))
					} else {
						noticeMust(t, tx.Tx.Rollback(ctx))
					}
					return errors.New("synthetic claim acknowledgement loss")
				}
				return tx.Tx.Commit(ctx)
			}
			d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) { d.Transactions = x })
			noticeMust(t, d.Reconcile(h.ctx))
			if calls.Load() != 0 || !h.log.has("claim_commit_unknown") {
				t.Fatal("unknown claim gained send rights")
			}
			if committed {
				h.assertState(id, article.RecipientProcessing, 1, "")
			} else {
				h.assertState(id, article.RecipientPending, 0, "")
			}
		})
	}
}

type noticeAuditor struct {
	fail    bool
	corrupt bool
	getFail bool
	appends atomic.Int64
	gets    atomic.Int64
}

func (a *noticeAuditor) Append(ctx context.Context, tx pgx.Tx, e audit.Event) error {
	a.appends.Add(1)
	if err := (audit.Store{}).Append(ctx, tx, e); err != nil {
		return err
	}
	if a.fail {
		return errors.New("synthetic audit after-insert fault")
	}
	return nil
}
func (a *noticeAuditor) Get(ctx context.Context, q audit.QueryRower, id string) (audit.Event, error) {
	a.gets.Add(1)
	e, err := (audit.Store{}).Get(ctx, q, id)
	if err == nil && a.corrupt {
		e.TargetID = noticeUUID()
	}
	if err == nil && a.getFail {
		err = errors.New("synthetic Get read failure")
	}
	return e, err
}
func testNoticeAuditAtomicity(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	a := &noticeAuditor{fail: true}
	var calls atomic.Int64
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) { d.Audit = a })
	noticeMust(t, d.Reconcile(h.ctx))
	h.assertState(id, article.RecipientProcessing, 1, "")
	if calls.Load() != 1 || h.scalar("SELECT count(*) FROM audit_events") != 0 || !h.log.has("result_persistence_failed") {
		t.Fatal("audit failure did not atomically roll back state")
	}
}
func testNoticeBigInt(t *testing.T) {
	for _, max := range []int64{9007199254740993, 9223372036854775807} {
		for _, corrupt := range []bool{false, true} {
			t.Run(fmt.Sprintf("max_%d_bad_binding_%t", max, corrupt), func(t *testing.T) {
				h := newNoticeHarness(t)
				id := h.seed(max, false)
				a := &noticeAuditor{corrupt: corrupt}
				var calls atomic.Int64
				var once atomic.Bool
				x := &noticeTransactions{pool: h.pool}
				x.hook = func(ctx context.Context, tx *noticeTx) error {
					err := tx.Tx.Commit(ctx)
					if err == nil && tx.phase == "result" && once.CompareAndSwap(false, true) {
						return errors.New("synthetic result acknowledgement loss")
					}
					return err
				}
				d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
					d.Transactions = x
					d.Audit = a
				})
				noticeMust(t, d.Reconcile(h.ctx))
				h.assertState(id, article.RecipientAccepted, 1, "")
				if h.log.has("result_commit_unknown") != corrupt {
					t.Fatal("real JSON round-trip binding did not resolve commit outcome correctly")
				}
				if calls.Load() != 1 || a.appends.Load() != 1 || a.gets.Load() != 1 {
					t.Fatal("unknown result repeated Send/Append or missed real Get")
				}
				var eid string
				noticeMust(t, h.pool.QueryRow(h.ctx, "SELECT id::text FROM audit_events WHERE action='article.transfer_notice.accepted'").Scan(&eid))
				e, err := (audit.Store{}).Get(h.ctx, h.pool, eid)
				noticeMust(t, err)
				if e.Metadata["max_attempts"] != fmt.Sprint(max) || e.Metadata["attempt_count"] != "1" {
					t.Fatal("audit bigint metadata not canonical decimal strings")
				}
			})
		}
	}
}

func testNoticeLateResult(t *testing.T) { testNoticeLate(t, false, false) }
func testNoticeABA(t *testing.T) {
	t.Run("SameWorkerID", func(t *testing.T) { testNoticeLate(t, true, false) })
	t.Run("ExpiredNoTakeover", func(t *testing.T) { testNoticeLate(t, false, true) })
	t.Run("LateAuditFallback", func(t *testing.T) { testNoticeLate(t, false, false, true) })
}
func testNoticeLate(t *testing.T, sameWorker, noTakeover bool, auditFault ...bool) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	sent := make(chan struct{})
	release := make(chan struct{})
	out := make(chan error, 1)
	worker := noticeUUID()
	// Pause an already completed send response to model a suspended caller, not a live I/O that ignores cancellation.
	m := noticeSendFunc(func(context.Context, mailer.Message) error {
		close(sent)
		select {
		case <-release:
			return nil
		case <-h.ctx.Done():
			return h.ctx.Err()
		}
	})
	a := h.dispatcher(m, func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
		d.WorkerID = worker
		c.BatchLimit = 1
		if len(auditFault) > 0 && auditFault[0] {
			d.Audit = &noticeAuditor{fail: true}
		}
	})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	h.group.Add(1)
	go func() { defer h.group.Done(); out <- a.Reconcile(h.ctx) }()
	select {
	case <-sent:
	case <-h.ctx.Done():
		t.Fatal("send barrier not reached")
	}
	r := h.recipient(id, article.RecipientNewAuthor)
	h.waitTime(*r.LockedUntil)
	if !noTakeover {
		b := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
			if sameWorker {
				d.WorkerID = worker
			}
		})
		noticeMust(t, b.Reconcile(h.ctx))
	}
	before := h.recipient(id, article.RecipientNewAuthor)
	risks := h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.duplicate_risk'")
	accepted := h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.accepted'")
	releaseOnce.Do(func() { close(release) })
	noticeMust(t, <-out)
	noticeAssertLate(h, id, before, risks, accepted, len(auditFault) > 0 && auditFault[0])
}

func noticeAssertLate(h *noticeHarness, id string, before article.TransferNoticeRecipient, risks, accepted int64, auditFault bool) {
	t := h.t
	t.Helper()
	if !reflect.DeepEqual(before, h.recipient(id, article.RecipientNewAuthor)) || accepted != h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.accepted'") {
		t.Fatal("late result changed new owner/terminal audit")
	}
	expectedRisk := risks + 1
	if auditFault {
		expectedRisk = risks
		if !h.log.has("late_result_risk_unpersisted") {
			t.Fatal("late audit fault lacked low-sensitive fallback")
		}
	}
	if h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.duplicate_risk'") != expectedRisk {
		t.Fatal("genuine issued/attempted late result lost independent risk audit")
	}
}

func testNoticeSnapshot(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, true)
	h.exec(`UPDATE users SET email='current-canary@example.test' WHERE status='active'`)
	h.exec(`UPDATE users SET retained_email=NULL WHERE status='deleted'`)
	h.exec(`UPDATE transfer_notice_recipients SET email_snapshot='shared-snapshot@example.test' WHERE notice_id=$1`, id)
	h.exec(`INSERT INTO articles(id,author_id,title,body,publication_state,violation_state,deletion_state,version,created_at,updated_at) SELECT article_ref,new_author_id,'private-title-canary','private-body-canary','published','clear','active',1,created_at,updated_at FROM transfer_notices WHERE id=$1`, id)
	h.exec(`DELETE FROM articles WHERE id=(SELECT article_ref FROM transfer_notices WHERE id=$1)`, id)
	s := newNoticeSMTP(t, noticeSMTPOptions{})
	d := h.dispatcher(s.mailer())
	noticeMust(t, d.Reconcile(h.ctx))
	if h.scalar("SELECT count(*) FROM transfer_notice_recipients WHERE notice_id=$1 AND status='accepted' AND attempt_count=1", id) != 2 {
		t.Fatal("same-address kinds were merged or lost")
	}
	s.mu.Lock()
	messages := append([]string(nil), s.messages...)
	s.mu.Unlock()
	if len(messages) != 2 {
		t.Fatal("snapshot did not independently deliver two kinds")
	}
	for _, raw := range messages {
		if !strings.Contains(raw, "To: shared-snapshot@example.test") || strings.Contains(raw, "current-canary") || strings.Contains(raw, "private-body-canary") || strings.Contains(raw, "private-title-canary") {
			t.Fatal("dispatcher queried current account/article instead of snapshot")
		}
	}
}
func testNoticeResultFaults(t *testing.T) {
	for _, mode := range []string{"rollback", "get_failure"} {
		t.Run(mode, func(t *testing.T) {
			h := newNoticeHarness(t)
			id := h.seed(3, false)
			a := &noticeAuditor{getFail: mode == "get_failure"}
			var calls atomic.Int64
			var once atomic.Bool
			x := &noticeTransactions{pool: h.pool}
			x.hook = func(ctx context.Context, tx *noticeTx) error {
				if tx.phase == "result" && once.CompareAndSwap(false, true) {
					if mode == "rollback" {
						noticeMust(t, tx.Tx.Rollback(ctx))
					} else {
						noticeMust(t, tx.Tx.Commit(ctx))
					}
					return errors.New("synthetic result commit outcome unknown")
				}
				return tx.Tx.Commit(ctx)
			}
			d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error { calls.Add(1); return nil }), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) {
				d.Transactions = x
				d.Audit = a
			})
			noticeMust(t, d.Reconcile(h.ctx))
			if !h.log.has("result_commit_unknown") || calls.Load() != 1 || a.appends.Load() != 1 {
				t.Fatal("unknown result was blindly resent/appended or reported resolved")
			}
			if mode == "rollback" {
				h.assertState(id, article.RecipientProcessing, 1, "")
				if h.scalar("SELECT count(*) FROM audit_events") != 0 {
					t.Fatal("rollback leaked audit")
				}
			} else {
				h.assertState(id, article.RecipientAccepted, 1, "")
				noticeMust(t, d.Reconcile(h.ctx))
				if calls.Load() != 1 || a.appends.Load() != 1 {
					t.Fatal("committed unknown terminal was resent")
				}
			}
		})
	}
}
func testNoticeFreshTime(t *testing.T) {
	h := newNoticeHarness(t)
	id := h.seed(3, false)
	var blocker pgx.Tx
	locked := make(chan struct{})
	out := make(chan error, 1)
	d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error {
		var err error
		blocker, err = h.pool.Begin(h.ctx)
		if err != nil {
			return err
		}
		_, err = blocker.Exec(h.ctx, "SELECT 1 FROM transfer_notice_recipients WHERE notice_id=$1 AND recipient_kind='new_author' FOR UPDATE", id)
		close(locked)
		return err
	}))
	h.group.Add(1)
	go func() { defer h.group.Done(); out <- d.Reconcile(h.ctx) }()
	<-locked
	t.Cleanup(func() { _ = blocker.Rollback(context.Background()) })
	pid := blocker.Conn().PgConn().PID()
	deadline := time.Now().Add(700 * time.Millisecond)
	for h.scalar("SELECT count(*) FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid))", int(pid)) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("real result lock wait not observed")
		}
	}
	fresh := h.now()
	noticeMust(t, blocker.Rollback(h.ctx))
	noticeMust(t, <-out)
	r := h.assertState(id, article.RecipientAccepted, 1, "")
	if r.AcceptedAt.Before(fresh) || r.UpdatedAt.Nanosecond()%1000 != 0 || r.LastAttemptAt.Nanosecond()%1000 != 0 {
		t.Fatal("result used stale/pre-lock time or non-microsecond authority")
	}
	var auditTime time.Time
	noticeMust(t, h.pool.QueryRow(h.ctx, "SELECT occurred_at FROM audit_events WHERE action='article.transfer_notice.accepted'").Scan(&auditTime))
	if !auditTime.Equal(*r.AcceptedAt) || !r.UpdatedAt.Equal(auditTime) {
		t.Fatal("result/audit did not share fresh DB microsecond time")
	}
}
func testNoticeExactFence(t *testing.T) {
	for _, mutation := range []string{"owner", "attempt", "microsecond_lease"} {
		t.Run(mutation, func(t *testing.T) {
			h := newNoticeHarness(t)
			id := h.seed(3, false)
			var before article.TransferNoticeRecipient
			d := h.dispatcher(noticeSendFunc(func(context.Context, mailer.Message) error {
				switch mutation {
				case "owner":
					h.exec("UPDATE transfer_notice_recipients SET claimed_by=$2 WHERE notice_id=$1 AND recipient_kind='new_author'", id, noticeUUID())
				case "attempt":
					h.exec("UPDATE transfer_notice_recipients SET attempt_count=attempt_count+1 WHERE notice_id=$1 AND recipient_kind='new_author'", id)
				case "microsecond_lease":
					h.exec("UPDATE transfer_notice_recipients SET locked_until=locked_until+interval '1 microsecond' WHERE notice_id=$1 AND recipient_kind='new_author'", id)
				}
				before = h.recipient(id, article.RecipientNewAuthor)
				return nil
			}), func(d *article.NoticeDispatcherDependencies, c *article.NoticeDispatcherConfig) { c.BatchLimit = 1 })
			noticeMust(t, d.Reconcile(h.ctx))
			if !reflect.DeepEqual(before, h.recipient(id, article.RecipientNewAuthor)) || h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.accepted'") != 0 {
				t.Fatal("inexact fencing updated recipient")
			}
			if h.scalar("SELECT count(*) FROM audit_events WHERE action='article.transfer_notice.duplicate_risk'") != 1 {
				t.Fatal("fenced issued response missing independent risk")
			}
		})
	}
}

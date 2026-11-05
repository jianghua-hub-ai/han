package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/mailer"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

func TestRecoveryOrdinaryWorkers(t *testing.T) {
	t.Run("InternalStartupThreeActualHandshakes", func(t *testing.T) {
		worker, err := article.NewDeletionWorker(article.DeletionWorkerDependencies{
			Discoverer: ordinaryDiscovery{}, Converger: ordinaryConverger{}, Now: time.Now,
			NewTraceID: func() (string, error) { return "ordinary-trace", nil }, Ticks: make(chan time.Time),
		}, article.DeletionWorkerConfig{PollInterval: time.Second, BatchSize: 1, OperationBudget: time.Second, BackoffInitial: time.Millisecond, BackoffMax: time.Second, MaxBatchesPerWake: 1, ShutdownBudget: time.Second})
		if err != nil {
			t.Fatal(err)
		}
		worker.Start(context.Background())
		t.Cleanup(func() {
			worker.Stop()
			if err := worker.Wait(context.Background()); err != nil {
				t.Error(err)
			}
		})
		started, ok := any(worker).(interface{ Started() <-chan struct{} })
		if !ok {
			t.Fatal("Start returned without an observable actual-loop started handshake")
		}
		select {
		case <-started.Started():
		case <-time.After(time.Second):
			t.Fatal("actual loop did not handshake")
		}
	})
	t.Run("InternalStartupThreeActualHandshakes/ThreeRealOwners", testOrdinaryStartup)
	t.Run("InternalStartupThreeActualHandshakes/EarlyDoneNeverActivates", func(t *testing.T) {
		f := newOrdinaryFixture(t)
		f.reset.Stop()
		if err := f.runtime.RunNormal(context.Background()); err == nil {
			t.Fatal("early Done accepted")
		}
		if _, err := f.runtime.Acquire(context.Background(), recovery.WorkRequest); err == nil {
			t.Fatal("early Done opened public admission")
		}
	})
	t.Run("AllScanClaimDispatchLeases", testOrdinaryBlockedScans)
	t.Run("AppendBarrierDoesNotWaitForSelf", testOrdinaryAppendBarrier)
	t.Run("SharedDeadlineNoticeFencedSave", testOrdinaryShutdownBinding)
	t.Run("RealJoinResourceOrderAndFailurePriority", testOrdinaryResourceJoin)
}

type ordinaryDiscovery struct{}

func (ordinaryDiscovery) DiscoverDeletionOperations(context.Context, time.Time, int) ([]string, error) {
	return nil, nil
}

type ordinaryConverger struct{}

func (ordinaryConverger) ConvergeDeletionOperation(context.Context, string, string) error { return nil }

type ordinaryAuthority struct {
	recovery.ContextRecoveryJournal
	snapshot                  recovery.AdmissionSnapshot
	recoveryHead, articleHead recovery.VerifiedJournalSnapshot
	closed                    atomic.Int32
}

func newOrdinaryAuthority() *ordinaryAuthority {
	r := recovery.AnchorSnapshot{JournalID: "recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1}
	a := recovery.AnchorSnapshot{JournalID: "article-journal", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 2}
	binding := recovery.InstallationBinding{Version: recovery.InstallationVersion, DeploymentID: "deployment", InstallationID: "installation", TargetDigest: ordinaryHash("target"), CommitmentRef: "commitment", CommitmentDigest: ordinaryHash("commitment"), RecoveryInitial: r, ArticleInitial: a, InstalledAt: time.Date(2026, 10, 20, 1, 0, 0, 123456000, time.UTC)}
	f := &ordinaryAuthority{snapshot: recovery.AdmissionSnapshot{Installation: binding, Revision: recovery.StoreRevision{InstallationID: "installation", Counter: 1}, Runs: []recovery.AdmissionRun{}}}
	// Independent, explicit canonical genesis fixture. No policy/reader computes this oracle.
	canonical := fmt.Sprintf(`{"version":"han-recovery-admission/v1","installation":{"metadata_version":"han-recovery-install/v1","deployment_id":"deployment","installation_id":"installation","target_digest":%q,"commitment_ref":"commitment","commitment_digest":%q,"recovery_journal_id":"recovery-journal","recovery_epoch":1,"recovery_initial_sequence":0,"recovery_initial_hash":"","article_journal_id":"article-journal","article_epoch":2,"article_initial_sequence":0,"article_initial_hash":"","revision":1,"installed_at":"2026-10-20T01:00:00.123456Z"},"runs":[]}`, binding.TargetDigest, binding.CommitmentDigest)
	f.snapshot.Revision.Digest = ordinaryHash(canonical)
	f.recoveryHead = recovery.VerifiedJournalSnapshot{JournalID: r.JournalID, Namespace: r.Namespace, Epoch: r.Epoch, Revision: "recovery-r0", Entries: []recovery.JournalEntry{}, Prefixes: []recovery.JournalHead{{}}}
	f.articleHead = recovery.VerifiedJournalSnapshot{JournalID: a.JournalID, Namespace: a.Namespace, Epoch: a.Epoch, Revision: "article-r0", Prefixes: []recovery.JournalHead{{}}}
	return f
}
func ordinaryHash(s string) string {
	b := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(b[:])
}
func (f *ordinaryAuthority) ReadAdmissionSnapshot(context.Context) (recovery.AdmissionSnapshot, error) {
	return f.snapshot, nil
}
func (f *ordinaryAuthority) ReadAdmissionRevision(context.Context) (recovery.StoreRevision, error) {
	return f.snapshot.Revision, nil
}
func (f *ordinaryAuthority) ReadVerifiedSnapshot(_ context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	if ns == recovery.RecoveryNamespaceV1 {
		return f.recoveryHead, nil
	}
	return f.articleHead, nil
}
func (f *ordinaryAuthority) ReadPublishedClosure(context.Context, recovery.ClosureRequest) (recovery.PublishedClosure, error) {
	return recovery.PublishedClosure{}, errors.New("unexpected history closure")
}
func (f *ordinaryAuthority) VerifyGenesis(context.Context, recovery.InstallationBinding) (recovery.GenesisProof, error) {
	b := f.snapshot.Installation
	return recovery.GenesisProof{Binding: b, RecoveryInitial: b.RecoveryInitial, ArticleInitial: b.ArticleInitial, CommitmentRevision: "commitment-r1", CommitmentDigest: b.CommitmentDigest}, nil
}
func (f *ordinaryAuthority) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	return ordinaryExclusive{f}, nil
}

type ordinaryExclusive struct{ f *ordinaryAuthority }

func (l ordinaryExclusive) Check(ctx context.Context) error {
	if l.f.closed.Load() != 0 {
		return recovery.ErrAdmissionClosed
	}
	return ctx.Err()
}
func (l ordinaryExclusive) Close() error { l.f.closed.Add(1); return nil }

type ordinaryFixture struct {
	lifecycle *recoveryOrdinaryLifecycle
	runtime   recovery.Runtime
	authority *ordinaryAuthority
	article   *article.DeletionWorker
	reset     *identity.PasswordResetRecoveryWorker
	notice    *article.TransferNoticeDispatcher
	mu        sync.Mutex
	stages    []string
	attempted chan recovery.WorkKind
}

func newOrdinaryFixture(t *testing.T) *ordinaryFixture {
	t.Helper()
	f := &ordinaryFixture{authority: newOrdinaryAuthority(), attempted: make(chan recovery.WorkKind, 32)}
	var err error
	f.lifecycle, err = newRecoveryOrdinaryLifecycle(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	f.runtime, err = recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: f.authority, Journal: f.authority, ArticleSnapshots: f.authority, Closures: f.authority, Genesis: f.authority, Exclusive: f.authority, Lifecycle: f.lifecycle})
	if err != nil {
		t.Fatal(err)
	}
	f.makeArticle(t)
	f.makeReset(t)
	f.makeNotice(t)
	if err = f.lifecycle.bind(f.runtime, f.article, f.reset, f.notice); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.runtime.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return f
}

func (f *ordinaryFixture) work(kind recovery.WorkKind) func(context.Context) (context.Context, func(), error) {
	return func(ctx context.Context) (context.Context, func(), error) {
		f.attempted <- kind
		return f.lifecycle.acquire(kind)(ctx)
	}
}
func (f *ordinaryFixture) record(ctx context.Context, name string) {
	b, err := f.runtime.ReadPermitBinding(ctx)
	if err != nil {
		panic(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stages = append(f.stages, name+":"+b.Purpose)
}
func (f *ordinaryFixture) makeArticle(t *testing.T) {
	var err error
	f.article, err = article.NewDeletionWorker(article.DeletionWorkerDependencies{AcquireWork: f.work(recovery.WorkArticleDeletion), Discoverer: ordinaryScan{f}, Converger: ordinaryConverger{}, Now: time.Now, NewTraceID: func() (string, error) { return "ordinary-trace", nil }, Ticks: make(chan time.Time)}, article.DeletionWorkerConfig{PollInterval: time.Hour, BatchSize: 1, OperationBudget: time.Second, BackoffInitial: time.Millisecond, BackoffMax: time.Second, MaxBatchesPerWake: 1, ShutdownBudget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
}

type ordinaryScan struct{ f *ordinaryFixture }

func (s ordinaryScan) DiscoverDeletionOperations(ctx context.Context, _ time.Time, _ int) ([]string, error) {
	s.f.record(ctx, "article")
	return nil, nil
}

func (f *ordinaryFixture) makeReset(t *testing.T) {
	permit, _ := identity.NewPasswordResetConnectionPermit(1)
	store := &identity.PasswordResetStore{}
	pool := ordinaryPool{f: f}
	operator, err := identity.NewPasswordResetOperator(identity.PasswordResetOperatorDependencies{Transactions: pool, Store: store, Audit: ordinaryAudit{}, IDs: ordinaryIDs{}, Clock: ordinaryClock{}, Permit: permit})
	if err != nil {
		t.Fatal(err)
	}
	f.reset, err = identity.NewPasswordResetRecoveryWorker(identity.PasswordResetWorkerDependencies{AcquireWork: f.work(recovery.WorkPasswordReset), Pool: pool, Store: store, Operator: operator, Permit: permit, WorkerID: "ordinary-worker", Clock: ordinaryClock{}, Ticks: make(chan time.Time)}, identity.PasswordResetWorkerConfig{LeaseDuration: time.Minute, RetryBackoff: time.Hour, MaxRetries: 3, ClaimBatchLimit: 1, ConnectionBudget: 1})
	if err != nil {
		t.Fatal(err)
	}
}

type ordinaryPool struct{ f *ordinaryFixture }

func (p ordinaryPool) Query(ctx context.Context, _ string, _ ...any) (pgx.Rows, error) {
	p.f.record(ctx, "reset")
	return &ordinaryRows{}, nil
}
func (p ordinaryPool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return ordinaryTx{f: p.f}, nil
}

type ordinaryRows struct{ pgx.Rows }

func (*ordinaryRows) Next() bool { return false }
func (*ordinaryRows) Err() error { return nil }
func (*ordinaryRows) Close()     {}

type ordinaryTx struct {
	pgx.Tx
	f *ordinaryFixture
}

func (tx ordinaryTx) QueryRow(ctx context.Context, _ string, _ ...any) pgx.Row {
	tx.f.record(ctx, "notice")
	return ordinaryNoRow{}
}
func (ordinaryTx) Rollback(context.Context) error { return nil }

type ordinaryNoRow struct{}

func (ordinaryNoRow) Scan(...any) error { return pgx.ErrNoRows }

type ordinaryClock struct{}

func (ordinaryClock) Now() time.Time { return time.Date(2026, 10, 20, 1, 0, 0, 123456000, time.UTC) }

type ordinaryIDs struct{}

func (ordinaryIDs) NewID() (string, error) { return "11111111-1111-4111-8111-111111111111", nil }

type ordinaryAudit struct{}

func (ordinaryAudit) Append(context.Context, pgx.Tx, audit.Event) error { return nil }
func (ordinaryAudit) Get(context.Context, audit.QueryRower, string) (audit.Event, error) {
	return audit.Event{}, errors.New("unexpected audit read")
}

type ordinaryMailer struct{}

func (ordinaryMailer) Send(context.Context, mailer.Message) error {
	return errors.New("unexpected send on empty queue")
}

type ordinaryGate struct{}

func (ordinaryGate) RequireOpen(context.Context) error             { return nil }
func (ordinaryGate) RequireOpenInTx(context.Context, pgx.Tx) error { return nil }

type ordinaryLogger struct{}

func (ordinaryLogger) Log(context.Context, slog.Level, string, ...any) {}

func (f *ordinaryFixture) makeNotice(t *testing.T) {
	var err error
	f.notice, err = article.NewTransferNoticeDispatcher(article.NoticeDispatcherDependencies{AcquireWork: f.work(recovery.WorkArticleNotice), Transactions: ordinaryPool{f}, Gate: ordinaryGate{}, Mailer: ordinaryMailer{}, Audit: ordinaryAudit{}, IDs: ordinaryIDs{}, NewTraceID: func() (string, error) { return "01ARZ3NDEKTSV4RRFFQ69G5FAV", nil }, Logger: ordinaryLogger{}, WorkerID: "11111111-1111-4111-8111-111111111111"}, article.NoticeDispatcherConfig{PollInterval: time.Hour, BatchLimit: 1, LeaseDuration: time.Minute, DBOperationBudget: time.Second, BackoffInitial: time.Second, BackoffMax: time.Minute, SMTPOperationBudget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
}

func testOrdinaryStartup(t *testing.T) {
	f := newOrdinaryFixture(t)
	if err := f.runtime.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, w := range []recoveryStartedWorker{f.article, f.reset, f.notice} {
		select {
		case <-w.Started():
		default:
			t.Fatal("actual loop has not entered")
		}
		select {
		case <-w.Done():
			t.Fatal("early Done")
		default:
		}
		if err := w.Err(); err != nil {
			t.Fatal(err)
		}
	}
	f.mu.Lock()
	stages := append([]string(nil), f.stages...)
	f.mu.Unlock()
	want := []string{"article:startup_reconcile", "reset:startup_reconcile", "notice:startup_reconcile", "notice:startup_reconcile"}
	if !reflect.DeepEqual(stages, want) {
		t.Fatalf("startup stages %v", stages)
	}
	if lease, err := f.runtime.Acquire(context.Background(), recovery.WorkRequest); err != nil {
		t.Fatal(err)
	} else {
		lease.Release()
	}
}

func testOrdinaryBlockedScans(t *testing.T) {
	f := newOrdinaryFixture(t)
	if err := f.runtime.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	for len(f.attempted) > 0 {
		<-f.attempted
	}
	held, err := f.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	barrier, err := f.runtime.PauseArticleAppend(held.Context(), "blocked-scans")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []lifecycleWorker{f.article, f.reset, f.notice} {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- w.Reconcile(ctx) }()
		select {
		case <-f.attempted:
		case err := <-done:
			t.Fatalf("%T ended before acquisition: %v, worker error: %v", w, err, w.Err())
		case <-time.After(3 * time.Second):
			t.Fatalf("%T never attempted acquisition; error=%v", w, w.Err())
		}
		select {
		case err := <-done:
			t.Fatalf("%T temporary closure did not wait: %v; worker error=%v", w, err, w.Err())
		default:
		}
		cancel()
		noticeAwait(t, done)
		if err := w.Err(); err != nil {
			t.Fatalf("temporary closed became terminal: %v", err)
		}
	}
	if err := f.runtime.CompleteArticleAppend(held.Context(), barrier, ordinaryAdvance("blocked-scans")); err != nil {
		t.Fatal(err)
	}
	held.Release()
}
func ordinaryAdvance(operation string) recovery.VerifiedHeadAdvance {
	return recovery.VerifiedHeadAdvance{JournalID: "article-journal", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 2, OldRevision: "article-r0", PendingRevision: "article-r1", CommittedRevision: "article-r2", OperationID: operation, NewHead: recovery.JournalHead{Sequence: 1, Hash: ordinaryHash("next")}}
}
func testOrdinaryAppendBarrier(t *testing.T) {
	f := newOrdinaryFixture(t)
	if err := f.runtime.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	work, release, err := f.lifecycle.acquire(recovery.WorkArticleDeletion)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	barrier, err := f.runtime.PauseArticleAppend(work, "self")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.runtime.CheckJournalUse(work, "self", recovery.JournalUseNewAppend); err != nil {
		t.Fatal("healthy Pause canceled its own lease", err)
	}
	fault := errors.New("append unknown")
	if err = f.runtime.FaultArticleAppend(work, barrier, fault); !errors.Is(err, fault) {
		t.Fatal(err)
	}
	noticeAwait(t, work.Done())
	if f.authority.closed.Load() != 0 {
		t.Fatal("deployment released while original work is owned")
	}
	release()
	if err := f.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.authority.closed.Load() != 1 {
		t.Fatal("deployment not joined once")
	}
}
func testOrdinaryShutdownBinding(t *testing.T) {
	f := newOrdinaryFixture(t)
	if err := f.runtime.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	lease, err := f.runtime.Acquire(context.Background(), recovery.WorkArticleNotice)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	stopped := make(chan error, 1)
	work, release := f.lifecycle.forwardCancellation(lease.Context(), context.Background())
	defer release()
	go func() { stopped <- f.runtime.Stop(context.Background()) }()
	noticeAwait(t, work.Done())
	first := f.lifecycle.shutdown
	deadline, _ := first.Deadline()
	later, cancelLater := context.WithTimeout(context.Background(), time.Hour)
	defer cancelLater()
	if f.lifecycle.bindShutdown(later) != first {
		t.Fatal("first shared deadline reset")
	}
	if d, _ := f.lifecycle.shutdown.Deadline(); !d.Equal(deadline) {
		t.Fatal("deadline extended")
	}
	release()
	lease.Release()
	if err := noticeAwait(t, stopped); err != nil {
		t.Fatal(err)
	}
}
func testOrdinaryResourceJoin(t *testing.T) {
	t.Run("FailurePriority", func(t *testing.T) { testNoticeFailurePriority(t, 1) })
	t.Run("AllOwnersBeforeResources", testOrdinaryOwnedJoins)
}
func testOrdinaryOwnedJoins(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var resources atomic.Int32
	business, metrics := newLifecycleServer("business"), newLifecycleServer("metrics")
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{businessServer: business, metricsServer: metrics, businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{}, articleWorker: newLifecycleWorkerFake(), passwordResetWorker: newLifecycleWorkerFake(), noticeWorker: newNoticeLifecycleProbe(), closeArticle: func() error {
			if resources.Add(1) != 1 {
				panic("article order")
			}
			return nil
		}, closePool: func() error {
			if resources.Add(1) != 2 {
				panic("pool order")
			}
			return nil
		}, joinWork: func(context.Context) error { close(entered); <-release; return nil }, shutdownBudget: time.Second, signals: signals})
	}()
	noticeAwait(t, business.started)
	noticeAwait(t, metrics.started)
	signals <- syscall.SIGTERM
	noticeAwait(t, entered)
	if resources.Load() != 0 {
		t.Fatal("resources closed before actual owner join")
	}
	close(release)
	if err := noticeAwait(t, result); err != nil {
		t.Fatal(err)
	}
}

type ordinaryRepairDrain struct {
	inner            *recoveryOrdinaryLifecycle
	entered, proceed chan struct{}
	once             sync.Once
}

func (d *ordinaryRepairDrain) ReconcileAndStart(c context.Context, p recovery.PhasePermit) error {
	return d.inner.ReconcileAndStart(c, p)
}
func (d *ordinaryRepairDrain) StopAndJoin(c context.Context) error {
	close(d.entered)
	<-d.proceed
	return d.inner.StopAndJoin(c)
}
func (d *ordinaryRepairDrain) release() { d.once.Do(func() { close(d.proceed) }) }

type ordinaryRepairKey struct{}
type ordinaryRepairExclusive struct {
	authority        *ordinaryAuthority
	armed            atomic.Bool
	entered, proceed chan struct{}
	once             sync.Once
}

func (e *ordinaryRepairExclusive) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	return &ordinaryRepairLease{e: e, base: ordinaryExclusive{e.authority}}, nil
}
func (e *ordinaryRepairExclusive) release() { e.once.Do(func() { close(e.proceed) }) }

type ordinaryRepairLease struct {
	e    *ordinaryRepairExclusive
	base ordinaryExclusive
}

func (l *ordinaryRepairLease) Check(c context.Context) error {
	if c.Value(ordinaryRepairKey{}) == true && l.e.armed.CompareAndSwap(true, false) {
		close(l.e.entered)
		select {
		case <-l.e.proceed:
		case <-c.Done():
			return c.Err()
		}
	}
	return l.base.Check(c)
}
func (l *ordinaryRepairLease) Close() error { return l.base.Close() }

func ordinaryRepairFixture(t *testing.T, gatedDrain bool) (*ordinaryFixture, *ordinaryRepairDrain, *ordinaryRepairExclusive) {
	return ordinaryRepairFixtureContext(t, gatedDrain, context.Background())
}

func ordinaryRepairFixtureContext(t *testing.T, gatedDrain bool, owner context.Context) (*ordinaryFixture, *ordinaryRepairDrain, *ordinaryRepairExclusive) {
	t.Helper()
	f := &ordinaryFixture{authority: newOrdinaryAuthority(), attempted: make(chan recovery.WorkKind, 64)}
	var err error
	f.lifecycle, err = newRecoveryOrdinaryLifecycle(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	d := &ordinaryRepairDrain{inner: f.lifecycle, entered: make(chan struct{}), proceed: make(chan struct{})}
	e := &ordinaryRepairExclusive{authority: f.authority, entered: make(chan struct{}), proceed: make(chan struct{})}
	var lifecycle recovery.OrdinaryLifecycle = f.lifecycle
	if gatedDrain {
		lifecycle = d
	}
	f.runtime, err = recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{
		Store: f.authority, Journal: f.authority, ArticleSnapshots: f.authority,
		Closures: f.authority, Genesis: f.authority, Exclusive: e, Lifecycle: lifecycle,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.makeArticle(t)
	f.makeReset(t)
	f.makeNotice(t)
	if err := f.lifecycle.bind(f.runtime, f.article, f.reset, f.notice); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		d.release()
		e.release()
		if err := f.runtime.Stop(context.Background()); err != nil && !errors.Is(err, owner.Err()) {
			t.Error(err)
		}
	})
	if err := f.runtime.RunNormal(owner); err != nil {
		t.Fatal(err)
	}
	return f, d, e
}

// Keep Runtime, adapter and all three real worker implementations unchanged.
// Gate only entry to the supplied lifecycle callback, after Runtime has already
// stored the caller's absolute deadline and revoked the generation.
func TestRecoveryOrdinaryWorkers_SharedDeadlineOrigin(t *testing.T) {
	f, drain, _ := ordinaryRepairFixture(t, true)
	work, release, err := f.lifecycle.acquire(recovery.WorkArticleNotice)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	defer drain.release()
	deadline := time.Now().Add(3 * time.Second)
	stopCtx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	stopped := make(chan error, 1)
	go func() { stopped <- f.runtime.Stop(stopCtx) }()
	noticeAwait(t, drain.entered)
	noticeAwait(t, work.Done())
	actual, ok := f.lifecycle.shutdown.Deadline()
	release()
	drain.release()
	if err := noticeAwait(t, stopped); err != nil {
		t.Fatal(err)
	}
	if !ok || !actual.Equal(deadline) {
		t.Errorf("FIRST_SHARED_DEADLINE_LOST: caller=%s notice=%s extension=%s; all workers/runtime Stop actually joined", deadline.Format(time.RFC3339Nano), actual.Format(time.RFC3339Nano), actual.Sub(deadline))
	}
}

// A real writer pauses between Acquire's public-ready observation and its final
// grant check. The ordinary adapter must wait until the caller's budget ends;
// a healthy pause must never surface as terminal admission closure.
func TestRecoveryOrdinaryWorkers_HealthyPauseDuringAcquire(t *testing.T) {
	f, _, gate := ordinaryRepairFixture(t, false)
	held, err := f.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	defer gate.release()
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), ordinaryRepairKey{}, true), 500*time.Millisecond)
	defer cancel()
	gate.armed.Store(true)
	result := make(chan error, 1)
	go func() {
		_, release, err := f.lifecycle.acquire(recovery.WorkPasswordReset)(ctx)
		if release != nil {
			release()
		}
		result <- err
	}()
	noticeAwait(t, gate.entered)
	barrier, err := f.runtime.PauseArticleAppend(held.Context(), "independent-healthy-pause")
	if err != nil {
		t.Fatal(err)
	}
	gate.release()
	err = noticeAwait(t, result)
	if completeErr := f.runtime.CompleteArticleAppend(held.Context(), barrier, ordinaryAdvance("independent-healthy-pause")); completeErr != nil {
		t.Fatal(completeErr)
	}
	held.Release()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("HEALTHY_PAUSE_BECAME_TERMINAL: got=%v admissionClosed=%v want caller deadline after waiting; healthy append completed and acquire goroutine joined", err, errors.Is(err, recovery.ErrAdmissionClosed))
	}
}

func TestRecoveryOrdinaryWorkers_HealthyPauseControls(t *testing.T) {
	for _, kind := range []recovery.WorkKind{recovery.WorkArticleDeletion, recovery.WorkPasswordReset, recovery.WorkArticleNotice} {
		for _, mode := range []string{"complete", "fault", "ownership"} {
			t.Run(string(kind)+"/"+mode, func(t *testing.T) { testOrdinaryRepairPause(t, kind, mode) })
		}
	}
	t.Run("request-immediate", func(t *testing.T) { testOrdinaryRepairPause(t, recovery.WorkRequest, "request") })
}

func testOrdinaryRepairPause(t *testing.T, kind recovery.WorkKind, mode string) {
	t.Helper()
	f, _, gate := ordinaryRepairFixture(t, false)
	held, err := f.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	defer gate.release()
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), ordinaryRepairKey{}, true), 3*time.Second)
	defer cancel()
	gate.armed.Store(true)
	result := make(chan error, 1)
	go func() {
		if kind == recovery.WorkRequest {
			lease, err := f.runtime.Acquire(ctx, kind)
			if lease != nil {
				lease.Release()
			}
			result <- err
			return
		}
		_, release, err := f.lifecycle.acquire(kind)(ctx)
		if release != nil {
			release()
		}
		result <- err
	}()
	noticeAwait(t, gate.entered)
	barrier, err := f.runtime.PauseArticleAppend(held.Context(), "repair-pause")
	if err != nil {
		t.Fatal(err)
	}
	completeOrdinaryRepairPause(t, f, gate, held, barrier, mode, result)
}

func completeOrdinaryRepairPause(t *testing.T, f *ordinaryFixture, gate *ordinaryRepairExclusive, held recovery.WorkLease, barrier recovery.AppendBarrier, mode string, result <-chan error) {
	t.Helper()
	if mode == "ownership" {
		f.authority.closed.Store(1)
	}
	gate.release()
	if mode == "complete" {
		if err := f.runtime.CompleteArticleAppend(held.Context(), barrier, ordinaryAdvance("repair-pause")); err != nil {
			t.Fatal(err)
		}
	} else if mode == "fault" {
		cause := errors.New("synthetic real append fault")
		if err := f.runtime.FaultArticleAppend(held.Context(), barrier, cause); !errors.Is(err, cause) {
			t.Fatal(err)
		}
		noticeAwait(t, held.Context().Done())
	}
	err := noticeAwait(t, result)
	if mode == "complete" {
		if err != nil {
			t.Fatalf("healthy append did not resume acquisition: %v", err)
		}
		if err := f.lifecycle.checkWorkers(); err != nil {
			t.Fatal(err)
		}
	} else if !errors.Is(err, recovery.ErrAdmissionClosed) && !errors.Is(err, context.Canceled) {
		t.Fatalf("terminal closure did not remain terminal: %v", err)
	}
	if mode == "request" {
		if err := f.runtime.CompleteArticleAppend(held.Context(), barrier, ordinaryAdvance("repair-pause")); err != nil {
			t.Fatal(err)
		}
	}
	held.Release()
}

func TestRecoveryOrdinaryWorkers_LocalCancellationDoesNotSelectShutdown(t *testing.T) {
	f := newOrdinaryFixture(t)
	if err := f.runtime.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	source, cancel := context.WithCancel(context.Background())
	work, release := f.lifecycle.forwardCancellation(source, context.Background())
	defer release()
	cancel()
	noticeAwait(t, work.Done())
	if c, selected := f.runtime.ShutdownContext(); c != nil || selected {
		t.Fatal("local cancellation selected runtime shutdown")
	}
	if f.lifecycle.shutdown != nil {
		t.Fatal("local cancellation invented a notice shutdown budget")
	}
	if err := f.lifecycle.checkWorkers(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryOrdinaryWorkers_ParentCancellationPublishesBeforeWork(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			owner, cancel := context.WithCancel(context.Background())
			if mode == "deadline" {
				cancel()
				owner, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
			}
			defer cancel()
			f, drain, _ := ordinaryRepairFixtureContext(t, true, owner)
			work, release, err := f.lifecycle.acquire(recovery.WorkArticleNotice)(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			defer drain.release()
			if mode == "cancel" {
				cancel()
			} else {
				noticeAwait(t, owner.Done())
			}
			noticeAwait(t, drain.entered)
			noticeAwait(t, work.Done())
			if shared, selected := f.runtime.ShutdownContext(); !selected || shared != owner {
				t.Error("parent cancellation escaped before first-context publication")
			}
			if f.lifecycle.shutdown == nil {
				t.Fatal("parent cancellation reached notice without a shared budget")
			}
			if deadline, ok := owner.Deadline(); ok {
				if actual, bounded := f.lifecycle.shutdown.Deadline(); !bounded || !actual.Equal(deadline) {
					t.Error("parent deadline was replaced")
				}
			} else if _, bounded := f.lifecycle.shutdown.Deadline(); !bounded {
				t.Error("parent cancellation lost the configured notice budget")
			}
			release()
			drain.release()
			err = f.runtime.Stop(context.Background())
			if mode == "deadline" {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Error(err)
				}
			} else if err != nil {
				t.Error(err)
			}
		})
	}
}

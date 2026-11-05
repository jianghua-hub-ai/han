package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"han/internal/article"
)

func TestRunLifecycle_AWIRELIFE03LIFE08SignalSharedDeadlineCleanExit(t *testing.T) {
	shutdownEntered := make(chan string, 2)
	releaseShutdown := make(chan struct{})
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(name string, ctx context.Context) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Errorf("%s Shutdown context has no deadline", name)
			}
			server.deadline = deadline
			shutdownEntered <- name
			<-releaseShutdown
			server.stop(http.ErrServerClosed)
			return nil
		}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: &lifecycleListenerFake{},
			metricsListener:  &lifecycleListenerFake{},
			closePool:        func() error { return nil },
			shutdownBudget:   time.Second,
			signals:          signals,
		})
	}()
	<-business.started
	<-metrics.started
	signals <- syscall.SIGTERM

	first := <-shutdownEntered
	second := <-shutdownEntered
	if first == second {
		t.Fatalf("Shutdown entries = %q/%q, want both servers", first, second)
	}
	if business.deadline.IsZero() || !business.deadline.Equal(metrics.deadline) {
		t.Fatalf("Shutdown deadlines = %v/%v, want one shared deadline", business.deadline, metrics.deadline)
	}
	close(releaseShutdown)

	if err := <-result; err != nil {
		t.Fatalf("runLifecycle() error = %v, want clean signal exit", err)
	}
	if business.closeCalls != 0 || metrics.closeCalls != 0 {
		t.Fatalf("Close calls = %d/%d, want 0/0", business.closeCalls, metrics.closeCalls)
	}
}

func TestRunLifecycle_AWIRELIFE03LIFE08ShutdownFailureClosesBoth(t *testing.T) {
	shutdownFailure := errors.New("synthetic shutdown failure")
	shutdownEntered := make(chan string, 2)
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	business.shutdown = func(name string, _ context.Context) error {
		shutdownEntered <- name
		return shutdownFailure
	}
	metrics.shutdown = func(name string, _ context.Context) error {
		shutdownEntered <- name
		return nil
	}
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.close = func(target *lifecycleServerFake) error {
			target.stop(http.ErrServerClosed)
			return nil
		}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: &lifecycleListenerFake{},
			metricsListener:  &lifecycleListenerFake{},
			closePool:        func() error { return nil },
			shutdownBudget:   time.Second,
			signals:          signals,
		})
	}()
	<-business.started
	<-metrics.started
	signals <- syscall.SIGINT
	<-shutdownEntered
	<-shutdownEntered

	err := <-result
	if !errors.Is(err, shutdownFailure) {
		t.Fatalf("runLifecycle() error = %v, want shutdown failure", err)
	}
	if business.closeCalls != 1 || metrics.closeCalls != 1 {
		t.Fatalf("Close calls = %d/%d, want 1/1", business.closeCalls, metrics.closeCalls)
	}
}

func TestRunLifecycle_AWIRELIFE05LIFE06LIFE08PreservesPrimaryAndOrder(t *testing.T) {
	serverFailure := errors.New("synthetic business serve failure")
	shutdownFailure := errors.New("synthetic metrics shutdown failure")
	listenerFailure := errors.New("synthetic listener cleanup failure")
	poolFailure := errors.New("synthetic pool cleanup failure")
	order := make(chan string, 8)
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	business.shutdown = func(_ string, _ context.Context) error { return nil }
	metrics.shutdown = func(_ string, _ context.Context) error { return shutdownFailure }
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.close = func(target *lifecycleServerFake) error {
			target.stop(http.ErrServerClosed)
			return nil
		}
	}
	businessListener := &lifecycleListenerFake{closeFn: func() error {
		order <- "business-listener"
		return listenerFailure
	}}
	metricsListener := &lifecycleListenerFake{closeFn: func() error {
		order <- "metrics-listener"
		return nil
	}}
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: businessListener,
			metricsListener:  metricsListener,
			closePool: func() error {
				order <- "pool"
				return poolFailure
			},
			shutdownBudget: time.Second,
			signals:        make(chan os.Signal),
		})
	}()
	<-business.started
	<-metrics.started
	business.stop(serverFailure)

	err := <-result
	if !errors.Is(err, serverFailure) || !errors.Is(err, shutdownFailure) ||
		!errors.Is(err, listenerFailure) || !errors.Is(err, poolFailure) {
		t.Fatalf("runLifecycle() error = %v, want joined server and cleanup failures", err)
	}
	if got := firstLifecycleError(err); !errors.Is(got, serverFailure) {
		t.Fatalf("primary error = %v, want server failure", got)
	}
	if business.closeCalls != 1 || metrics.closeCalls != 1 {
		t.Fatalf("Close calls = %d/%d, want 1/1", business.closeCalls, metrics.closeCalls)
	}
	first := <-order
	second := <-order
	third := <-order
	if first != "business-listener" || second != "metrics-listener" || third != "pool" {
		t.Fatalf("cleanup order = %q/%q/%q", first, second, third)
	}
}

func TestRunLifecycle_AWIRELIFE02LIFE03DeadlineForcesBothClose(t *testing.T) {
	deadlineReleased := make(chan struct{})
	deadlineFactory := func(_ context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-deadlineReleased
			cancel()
		}()
		return ctx, cancel
	}
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}
		server.close = func(target *lifecycleServerFake) error {
			target.stop(http.ErrServerClosed)
			return nil
		}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: &lifecycleListenerFake{},
			metricsListener:  &lifecycleListenerFake{},
			closePool:        func() error { return nil },
			shutdownBudget:   time.Second,
			signals:          signals,
			deadlineContext:  deadlineFactory,
		})
	}()
	<-business.started
	<-metrics.started
	signals <- syscall.SIGTERM
	close(deadlineReleased)

	err := <-result
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runLifecycle() error = %v, want shared deadline failure", err)
	}
	if business.closeCalls != 1 || metrics.closeCalls != 1 {
		t.Fatalf("Close calls = %d/%d, want 1/1", business.closeCalls, metrics.closeCalls)
	}
}

func TestLifecycleExitCode(t *testing.T) {
	if got := lifecycleExitCode(nil); got != 0 {
		t.Fatalf("success exit code = %d", got)
	}
	if got := lifecycleExitCode(errors.New("failure")); got == 0 {
		t.Fatal("failure exit code = 0")
	}
}

func TestRunLifecycle_AWIRELIFE01LIFE05WorkerStopsBeforeResources(t *testing.T) {
	order := make(chan string, 8)
	worker := newLifecycleWorkerFake()
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, _ context.Context) error {
			server.stop(http.ErrServerClosed)
			return nil
		}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: &lifecycleListenerFake{},
			metricsListener:  &lifecycleListenerFake{},
			articleWorker:    worker,
			closeArticle: func() error {
				order <- "article"
				return nil
			},
			closePool: func() error {
				order <- "pool"
				return nil
			},
			shutdownBudget: time.Second,
			signals:        signals,
		})
	}()
	<-business.started
	<-metrics.started
	if worker.reconcileCalls != 1 || worker.startCalls != 1 {
		t.Fatalf(
			"worker reconcile/start calls = %d/%d, want 1/1",
			worker.reconcileCalls,
			worker.startCalls,
		)
	}
	signals <- syscall.SIGTERM
	if err := <-result; err != nil {
		t.Fatalf("runLifecycle() error = %v", err)
	}
	if worker.stopCalls != 1 || worker.waitCalls != 1 {
		t.Fatalf(
			"worker stop/wait calls = %d/%d, want 1/1",
			worker.stopCalls,
			worker.waitCalls,
		)
	}
	if first, second := <-order, <-order; first != "article" || second != "pool" {
		t.Fatalf("close order = %q/%q, want article/pool", first, second)
	}
}

func TestRunLifecycle_AWIREWKR04LIFE04LIFE05InitialWorkerFailureUnwinds(t *testing.T) {
	order := make([]string, 0, 2)
	worker := newLifecycleWorkerFake()
	worker.reconcileErr = errors.New("synthetic terminal worker failure")
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	err := runLifecycle(context.Background(), lifecycleConfig{
		businessServer:   business,
		metricsServer:    metrics,
		businessListener: &lifecycleListenerFake{},
		metricsListener:  &lifecycleListenerFake{},
		articleWorker:    worker,
		closeArticle: func() error {
			order = append(order, "article")
			return nil
		},
		closePool: func() error {
			order = append(order, "pool")
			return nil
		},
		shutdownBudget: time.Second,
		signals:        make(chan os.Signal),
	})
	if err == nil || !errors.Is(err, worker.reconcileErr) {
		t.Fatalf("runLifecycle() error = %v", err)
	}
	if !reflect.DeepEqual(order, []string{"article", "pool"}) {
		t.Fatalf("cleanup order = %v, want article/pool", order)
	}
	select {
	case <-business.started:
		t.Fatal("business server started after initial worker failure")
	default:
	}
}

func TestRunLifecycle_AWIREWKR08LIFE08RuntimeWorkerFailureIsGlobal(t *testing.T) {
	worker := newLifecycleWorkerFake()
	worker.err = errors.Join(
		article.ErrDeletionWorkerTerminal,
		article.ErrDeletionJournalCorrupt,
	)
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, _ context.Context) error {
			server.stop(http.ErrServerClosed)
			return nil
		}
	}

	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: &lifecycleListenerFake{},
			metricsListener:  &lifecycleListenerFake{},
			articleWorker:    worker,
			closeArticle:     func() error { return nil },
			closePool:        func() error { return nil },
			shutdownBudget:   time.Second,
			signals:          make(chan os.Signal),
		})
	}()
	<-business.started
	<-metrics.started
	close(worker.done)
	err := <-result

	if !errors.Is(err, article.ErrDeletionWorkerTerminal) ||
		!errors.Is(err, article.ErrDeletionJournalCorrupt) {
		t.Fatalf("runLifecycle() error = %v, want terminal worker failure", err)
	}
	if business.shutdownCalls != 1 || metrics.shutdownCalls != 1 {
		t.Fatalf(
			"shutdown calls = %d/%d, want 1/1",
			business.shutdownCalls,
			metrics.shutdownCalls,
		)
	}
}

func TestRunLifecycle_AWIRELIFE07CompetingTriggersCloseResourcesOnce(t *testing.T) {
	worker := newLifecycleWorkerFake()
	worker.err = errors.New("synthetic worker failure")
	signals := make(chan os.Signal, 1)
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, _ context.Context) error {
			server.stop(http.ErrServerClosed)
			return nil
		}
	}
	var articleCloses int
	var poolCloses int

	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer:   business,
			metricsServer:    metrics,
			businessListener: &lifecycleListenerFake{},
			metricsListener:  &lifecycleListenerFake{},
			articleWorker:    worker,
			closeArticle: func() error {
				articleCloses++
				return nil
			},
			closePool: func() error {
				poolCloses++
				return nil
			},
			shutdownBudget: time.Second,
			signals:        signals,
		})
	}()
	<-business.started
	<-metrics.started
	close(worker.done)
	signals <- syscall.SIGTERM
	if err := <-result; !errors.Is(firstLifecycleError(err), worker.err) {
		t.Fatalf("primary error = %v", err)
	}

	if worker.stopCalls != 1 || worker.waitCalls != 1 ||
		articleCloses != 1 || poolCloses != 1 {
		t.Fatalf(
			"stop/wait/article/pool = %d/%d/%d/%d, want all once",
			worker.stopCalls,
			worker.waitCalls,
			articleCloses,
			poolCloses,
		)
	}
}

func TestNoticeLifecycle_PrequeuedStop_NoInitialWork(t *testing.T) {
	for _, parentCanceled := range []bool{false, true} {
		t.Run(fmt.Sprint(parentCanceled), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			signals := make(chan os.Signal, 1)
			if parentCanceled {
				cancel()
			} else {
				signals <- syscall.SIGTERM
			}
			worker := newLifecycleWorkerFake()
			business, metrics := newLifecycleServer("business"), newLifecycleServer("metrics")
			closes := 0
			err := runLifecycle(ctx, lifecycleConfig{
				businessServer: business, metricsServer: metrics,
				businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{},
				articleWorker: worker, closeArticle: func() error { closes++; return nil },
				closePool: func() error { closes++; return nil }, shutdownBudget: time.Second, signals: signals,
			})
			if worker.reconcileCalls != 0 || worker.startCalls != 0 {
				t.Errorf("reconcile/start = %d/%d, want zero", worker.reconcileCalls, worker.startCalls)
			}
			select {
			case <-business.started:
				t.Error("business Serve started after stop")
			default:
			}
			select {
			case <-metrics.started:
				t.Error("metrics Serve started after stop")
			default:
			}
			if parentCanceled != (err != nil) {
				t.Errorf("parentCanceled=%v error=%v", parentCanceled, err)
			}
			if closes != 2 {
				t.Errorf("resource closes = %d, want 2", closes)
			}
		})
	}
}

type lifecycleServerFake struct {
	name          string
	started       chan struct{}
	stopped       chan error
	stopOnce      sync.Once
	shutdown      func(string, context.Context) error
	close         func(*lifecycleServerFake) error
	deadline      time.Time
	shutdownCalls int
	closeCalls    int
}

func newLifecycleServer(name string) *lifecycleServerFake {
	return &lifecycleServerFake{
		name:    name,
		started: make(chan struct{}),
		stopped: make(chan error, 1),
	}
}

func (server *lifecycleServerFake) Serve(net.Listener) error {
	close(server.started)
	return <-server.stopped
}

func (server *lifecycleServerFake) Shutdown(ctx context.Context) error {
	server.shutdownCalls++
	if server.shutdown == nil {
		server.stop(http.ErrServerClosed)
		return nil
	}
	return server.shutdown(server.name, ctx)
}

func (server *lifecycleServerFake) Close() error {
	server.closeCalls++
	if server.close == nil {
		server.stop(http.ErrServerClosed)
		return nil
	}
	return server.close(server)
}

func (server *lifecycleServerFake) stop(err error) {
	server.stopOnce.Do(func() {
		server.stopped <- err
	})
}

type lifecycleListenerFake struct {
	closeFn func() error
}

type lifecycleWorkerFake struct {
	done           chan struct{}
	reconcileErr   error
	err            error
	reconcileCalls int
	startCalls     int
	stopCalls      int
	waitCalls      int
}

func newLifecycleWorkerFake() *lifecycleWorkerFake {
	return &lifecycleWorkerFake{done: make(chan struct{})}
}

func (worker *lifecycleWorkerFake) Reconcile(context.Context) error {
	worker.reconcileCalls++
	return worker.reconcileErr
}

func (worker *lifecycleWorkerFake) Start(context.Context) {
	worker.startCalls++
}

func (worker *lifecycleWorkerFake) Stop() {
	worker.stopCalls++
	select {
	case <-worker.done:
	default:
		close(worker.done)
	}
}

func (worker *lifecycleWorkerFake) Wait(context.Context) error {
	worker.waitCalls++
	<-worker.done
	return nil
}

func (worker *lifecycleWorkerFake) Done() <-chan struct{} {
	return worker.done
}

func (worker *lifecycleWorkerFake) Err() error {
	return worker.err
}

func (*lifecycleListenerFake) Accept() (net.Conn, error) {
	return nil, errors.New("not used")
}

func (listener *lifecycleListenerFake) Close() error {
	if listener.closeFn != nil {
		return listener.closeFn()
	}
	return nil
}

func (*lifecycleListenerFake) Addr() net.Addr {
	return &net.TCPAddr{}
}

func TestRunLifecyclePasswordResetLifecycleOrder(t *testing.T) {
	var mu sync.Mutex
	events := make([]string, 0, 8)
	record := func(event string) { mu.Lock(); events = append(events, event); mu.Unlock() }
	articleWorker := newTracingLifecycleWorker("article", record)
	passwordWorker := newTracingLifecycleWorker("password-reset", record)
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, _ context.Context) error { server.stop(http.ErrServerClosed); return nil }
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer: business, metricsServer: metrics,
			businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{},
			articleWorker: articleWorker, passwordResetWorker: passwordWorker,
			closeArticle:   func() error { record("article-close"); return nil },
			closePool:      func() error { record("pool-close"); return nil },
			shutdownBudget: time.Second, signals: signals,
		})
	}()
	<-business.started
	<-metrics.started
	mu.Lock()
	startup := append([]string(nil), events...)
	mu.Unlock()
	wantStartup := []string{"article-reconcile", "password-reset-reconcile", "article-start", "password-reset-start"}
	if !reflect.DeepEqual(startup, wantStartup) {
		t.Fatalf("startup order = %v, want %v", startup, wantStartup)
	}
	signals <- syscall.SIGTERM
	if err := <-result; err != nil {
		t.Fatalf("runLifecycle() error = %v", err)
	}
	if articleWorker.stopCalls != 1 || passwordWorker.stopCalls != 1 || articleWorker.waitCalls != 1 || passwordWorker.waitCalls != 1 {
		t.Fatalf("worker stop/wait = %d/%d %d/%d", articleWorker.stopCalls, articleWorker.waitCalls, passwordWorker.stopCalls, passwordWorker.waitCalls)
	}
	mu.Lock()
	final := append([]string(nil), events...)
	mu.Unlock()
	articleWait := indexLifecycleEvent(final, "article-wait")
	passwordWait := indexLifecycleEvent(final, "password-reset-wait")
	closeArticle := indexLifecycleEvent(final, "article-close")
	closePool := indexLifecycleEvent(final, "pool-close")
	if articleWait < 0 || passwordWait < 0 || closeArticle <= articleWait || closeArticle <= passwordWait || closePool <= closeArticle {
		t.Fatalf("cleanup order = %v", final)
	}
}

func TestRunLifecyclePasswordResetReconcileFailurePreventsStartAndServe(t *testing.T) {
	failure := errors.New("password reset reconcile failure")
	articleWorker := newLifecycleWorkerFake()
	passwordWorker := newLifecycleWorkerFake()
	passwordWorker.reconcileErr = failure
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	err := runLifecycle(context.Background(), lifecycleConfig{
		businessServer: business, metricsServer: metrics,
		businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{},
		articleWorker: articleWorker, passwordResetWorker: passwordWorker,
		closeArticle: func() error { return nil }, closePool: func() error { return nil },
		shutdownBudget: time.Second, signals: make(chan os.Signal),
	})
	if !errors.Is(err, failure) {
		t.Fatalf("runLifecycle() error = %v", err)
	}
	if articleWorker.startCalls != 0 || passwordWorker.startCalls != 0 {
		t.Fatalf("start calls = %d/%d", articleWorker.startCalls, passwordWorker.startCalls)
	}
	select {
	case <-business.started:
		t.Fatal("business server started")
	default:
	}
	select {
	case <-metrics.started:
		t.Fatal("metrics server started")
	default:
	}
}

func TestRunLifecyclePasswordResetRuntimeFailureIsPrimary(t *testing.T) {
	passwordFailure := errors.New("password reset worker failure")
	articleWorker := newLifecycleWorkerFake()
	passwordWorker := newLifecycleWorkerFake()
	passwordWorker.err = passwordFailure
	close(passwordWorker.done)
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, _ context.Context) error { server.stop(http.ErrServerClosed); return nil }
	}
	err := runLifecycle(context.Background(), lifecycleConfig{
		businessServer: business, metricsServer: metrics,
		businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{},
		articleWorker: articleWorker, passwordResetWorker: passwordWorker,
		closeArticle: func() error { return nil }, closePool: func() error { return nil },
		shutdownBudget: time.Second, signals: make(chan os.Signal),
	})
	if !errors.Is(firstLifecycleError(err), passwordFailure) {
		t.Fatalf("primary error = %v", firstLifecycleError(err))
	}
}

func TestRunLifecyclePasswordResetDeadlineWaitsForDoneBeforeClosingResources(t *testing.T) {
	articleWorker := newLifecycleWorkerFake()
	passwordWorker := newBlockedLifecycleWorker()
	business := newLifecycleServer("business")
	metrics := newLifecycleServer("metrics")
	for _, server := range []*lifecycleServerFake{business, metrics} {
		server.shutdown = func(_ string, ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}
	}
	var mu sync.Mutex
	closeArticleCalls := 0
	listenerCloseCalls := 0
	poolCloseCalls := 0
	listener := func() *lifecycleListenerFake {
		return &lifecycleListenerFake{closeFn: func() error {
			mu.Lock()
			listenerCloseCalls++
			mu.Unlock()
			return nil
		}}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer: business, metricsServer: metrics,
			businessListener: listener(), metricsListener: listener(),
			articleWorker: articleWorker, passwordResetWorker: passwordWorker,
			closeArticle: func() error {
				mu.Lock()
				closeArticleCalls++
				mu.Unlock()
				return nil
			},
			closePool: func() error {
				mu.Lock()
				poolCloseCalls++
				mu.Unlock()
				return nil
			},
			shutdownBudget: 10 * time.Millisecond, signals: signals,
		})
	}()
	<-business.started
	<-metrics.started
	signals <- syscall.SIGTERM
	<-passwordWorker.waitReturned
	<-passwordWorker.doneWaitEntered
	mu.Lock()
	beforeDone := []int{closeArticleCalls, listenerCloseCalls, poolCloseCalls}
	mu.Unlock()
	if !reflect.DeepEqual(beforeDone, []int{0, 0, 0}) {
		t.Fatalf("close calls before password worker done = %v", beforeDone)
	}
	close(passwordWorker.done)
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("runLifecycle() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runLifecycle did not finish after worker done")
	}
	mu.Lock()
	afterDone := []int{closeArticleCalls, listenerCloseCalls, poolCloseCalls}
	mu.Unlock()
	if !reflect.DeepEqual(afterDone, []int{1, 2, 1}) {
		t.Fatalf("close calls after password worker done = %v", afterDone)
	}
}

type blockedLifecycleWorker struct {
	*lifecycleWorkerFake
	waitReturned    chan struct{}
	doneWaitEntered chan struct{}
	doneWaitOnce    sync.Once
}

func newBlockedLifecycleWorker() *blockedLifecycleWorker {
	return &blockedLifecycleWorker{
		lifecycleWorkerFake: newLifecycleWorkerFake(),
		waitReturned:        make(chan struct{}),
		doneWaitEntered:     make(chan struct{}),
	}
}

func (worker *blockedLifecycleWorker) Stop() {
	worker.stopCalls++
}

func (worker *blockedLifecycleWorker) Wait(ctx context.Context) error {
	worker.waitCalls++
	<-ctx.Done()
	close(worker.waitReturned)
	return ctx.Err()
}

func (worker *blockedLifecycleWorker) Done() <-chan struct{} {
	select {
	case <-worker.waitReturned:
		worker.doneWaitOnce.Do(func() { close(worker.doneWaitEntered) })
	default:
	}
	return worker.done
}

type tracingLifecycleWorker struct {
	*lifecycleWorkerFake
	name   string
	record func(string)
}

func newTracingLifecycleWorker(name string, record func(string)) *tracingLifecycleWorker {
	return &tracingLifecycleWorker{lifecycleWorkerFake: newLifecycleWorkerFake(), name: name, record: record}
}
func (worker *tracingLifecycleWorker) Reconcile(context.Context) error {
	worker.reconcileCalls++
	worker.record(worker.name + "-reconcile")
	return worker.reconcileErr
}
func (worker *tracingLifecycleWorker) Start(context.Context) {
	worker.startCalls++
	worker.record(worker.name + "-start")
}
func (worker *tracingLifecycleWorker) Stop() {
	worker.record(worker.name + "-stop")
	worker.lifecycleWorkerFake.Stop()
}
func (worker *tracingLifecycleWorker) Wait(context.Context) error {
	worker.record(worker.name + "-wait")
	worker.waitCalls++
	<-worker.done
	return nil
}
func indexLifecycleEvent(events []string, target string) int {
	for index, event := range events {
		if event == target {
			return index
		}
	}
	return -1
}

type noticeLifecycleProbe struct {
	mu                                               sync.Mutex
	done                                             chan struct{}
	bound                                            chan context.Context
	stopped                                          chan struct{}
	reconcile                                        func(context.Context) error
	shared                                           context.Context
	work                                             context.Context
	active, stopping, closed                         bool
	retainFailure                                    bool
	stopFailure                                      error
	err                                              error
	reconcileCalls, startCalls, stopCalls, waitCalls int
}

func newNoticeLifecycleProbe() *noticeLifecycleProbe {
	return &noticeLifecycleProbe{done: make(chan struct{}), bound: make(chan context.Context, 1), stopped: make(chan struct{})}
}

func (p *noticeLifecycleProbe) Reconcile(ctx context.Context) error {
	p.mu.Lock()
	p.active = true
	p.work = ctx
	p.reconcileCalls++
	p.mu.Unlock()
	var err error
	if p.reconcile != nil {
		err = p.reconcile(ctx)
	}
	p.mu.Lock()
	if p.retainFailure {
		p.err = err
	}
	p.active = false
	p.finishLocked()
	p.mu.Unlock()
	return err
}

func (p *noticeLifecycleProbe) Start(context.Context) { p.mu.Lock(); p.startCalls++; p.mu.Unlock() }
func (p *noticeLifecycleProbe) BeginShutdown(ctx context.Context) {
	p.mu.Lock()
	p.shared = ctx
	p.mu.Unlock()
	p.bound <- ctx
}
func (p *noticeLifecycleProbe) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopCalls++
	if p.stopFailure != nil {
		p.err = p.stopFailure
	}
	if !p.stopping {
		p.stopping = true
		close(p.stopped)
	}
	p.finishLocked()
}
func (p *noticeLifecycleProbe) finishLocked() {
	if p.stopping && !p.active && !p.closed {
		p.closed = true
		close(p.done)
	}
}
func (p *noticeLifecycleProbe) Wait(ctx context.Context) error {
	p.mu.Lock()
	p.waitCalls++
	p.mu.Unlock()
	select {
	case <-p.done:
		return p.Err()
	case <-ctx.Done():
		<-p.done
		return errors.Join(p.Err(), ctx.Err())
	}
}
func (p *noticeLifecycleProbe) Done() <-chan struct{} { return p.done }
func (p *noticeLifecycleProbe) Err() error            { p.mu.Lock(); defer p.mu.Unlock(); return p.err }

func noticeAwait[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(3 * time.Second):
		t.Fatal("barrier did not complete")
		var zero T
		return zero
	}
}

func TestNoticeLifecycle_InitialStages_StopAndJoinBeforeResources(t *testing.T) {
	for _, stage := range []string{"send blocked", "accepted before save", "next recipient before claim"} {
		for _, parentStop := range []bool{false, true} {
			t.Run(stage+fmt.Sprint(parentStop), func(t *testing.T) { testNoticeInitialStop(t, stage, parentStop) })
		}
	}
}

func testNoticeInitialStop(t *testing.T, stage string, parentStop bool) {
	p := newNoticeLifecycleProbe()
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 2)
	var sharedBeforeCancel bool
	var saves int
	p.reconcile = func(work context.Context) error {
		close(entered)
		<-work.Done()
		p.mu.Lock()
		shared := p.shared
		p.mu.Unlock()
		sharedBeforeCancel = shared != nil
		if stage == "accepted before save" {
			saves++
		}
		close(canceled)
		<-release
		return work.Err()
	}
	articleWorker, passwordWorker := newLifecycleWorkerFake(), newLifecycleWorkerFake()
	var articleCloses, poolCloses, listenerCloses atomic.Int32
	business, metrics := newLifecycleServer("business"), newLifecycleServer("metrics")
	factoryCalls := 0
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(ctx, lifecycleConfig{
			businessServer: business, metricsServer: metrics,
			businessListener: &lifecycleListenerFake{closeFn: func() error { listenerCloses.Add(1); return nil }},
			metricsListener:  &lifecycleListenerFake{closeFn: func() error { listenerCloses.Add(1); return nil }},
			articleWorker:    articleWorker, passwordResetWorker: passwordWorker, noticeWorker: p,
			closeArticle: func() error { articleCloses.Add(1); return nil }, closePool: func() error { poolCloses.Add(1); return nil },
			shutdownBudget: time.Second, signals: signals, deadlineContext: func(base context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
				factoryCalls++
				return context.WithTimeout(base, budget)
			},
		})
	}()
	noticeAwait(t, entered)
	if parentStop {
		cancel()
	} else {
		signals <- syscall.SIGTERM
	}
	shared := noticeAwait(t, p.bound)
	noticeAwait(t, canceled)
	noticeAwait(t, p.stopped)
	deadline, ok := shared.Deadline()
	if !ok || deadline.IsZero() || !sharedBeforeCancel {
		t.Error("shared deadline was not bound before work cancellation")
	}
	signals <- syscall.SIGTERM
	if articleCloses.Load() != 0 || poolCloses.Load() != 0 || listenerCloses.Load() != 0 {
		t.Error("resources closed before initial call exited")
	}
	select {
	case <-p.Done():
		t.Error("Done closed before initial call exited")
	default:
	}
	close(release)
	err := noticeAwait(t, result)
	if parentStop != (err != nil) {
		t.Errorf("parent stop=%v error=%v", parentStop, err)
	}
	if factoryCalls != 1 || p.startCalls != 0 || p.reconcileCalls != 1 || p.stopCalls != 1 || p.waitCalls != 1 {
		t.Errorf("factory/reconcile/start/stop/wait=%d/%d/%d/%d/%d", factoryCalls, p.reconcileCalls, p.startCalls, p.stopCalls, p.waitCalls)
	}
	if articleWorker.waitCalls != 0 || passwordWorker.waitCalls != 0 {
		t.Error("waited for unstarted old worker")
	}
	if business.shutdownCalls != 0 || metrics.shutdownCalls != 0 {
		t.Error("shutdown called on unserved server")
	}
	select {
	case <-business.started:
		t.Error("business Serve after stop")
	default:
	}
	select {
	case <-metrics.started:
		t.Error("metrics Serve after stop")
	default:
	}
	if articleCloses.Load() != 1 || poolCloses.Load() != 1 || listenerCloses.Load() != 2 {
		t.Error("resources not closed exactly once")
	}
	if stage == "accepted before save" && saves != 1 {
		t.Errorf("save calls=%d", saves)
	}
}

func TestNoticeLifecycle_ReadyStopAndReconcile_NoNextStage(t *testing.T) {
	for _, stopAt := range []string{"article", "notice"} {
		t.Run(stopAt, func(t *testing.T) {
			signals := make(chan os.Signal, 1)
			first, notice := newNoticeLifecycleProbe(), newNoticeLifecycleProbe()
			target := notice
			if stopAt == "article" {
				target = first
			}
			target.reconcile = func(context.Context) error { signals <- syscall.SIGTERM; return nil }
			business, metrics := newLifecycleServer("business"), newLifecycleServer("metrics")
			err := runLifecycle(context.Background(), lifecycleConfig{businessServer: business, metricsServer: metrics,
				businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{}, articleWorker: first, noticeWorker: notice,
				closeArticle: func() error { return nil }, closePool: func() error { return nil }, shutdownBudget: time.Second, signals: signals})
			if err != nil || first.startCalls != 0 || notice.startCalls != 0 {
				t.Fatalf("stage advanced or clean signal failed: %v", err)
			}
			if stopAt == "article" && notice.reconcileCalls != 0 {
				t.Fatal("later initial call was dispatched")
			}
			select {
			case <-business.started:
				t.Fatal("Serve after queued stop")
			default:
			}
		})
	}
}

type noticeSaveCancellation struct{}

func (noticeSaveCancellation) Error() string { return "synthetic result persistence failure" }
func (noticeSaveCancellation) Unwrap() error { return context.Canceled }

func TestNoticeLifecycle_SaveFailureWrappingCancellation_RemainsNonzero(t *testing.T) {
	p := newNoticeLifecycleProbe()
	p.retainFailure = true
	entered := make(chan struct{})
	p.reconcile = func(ctx context.Context) error { close(entered); <-ctx.Done(); return noticeSaveCancellation{} }
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer: newLifecycleServer("business"), metricsServer: newLifecycleServer("metrics"),
			businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{}, noticeWorker: p,
			closePool: func() error { return nil }, shutdownBudget: time.Second, signals: signals})
	}()
	noticeAwait(t, entered)
	signals <- syscall.SIGTERM
	err := noticeAwait(t, result)
	var saveFailure noticeSaveCancellation
	if !errors.As(err, &saveFailure) || lifecycleExitCode(err) != 1 {
		t.Fatalf("save failure masked as signal cancellation: %v", err)
	}
}

func TestNoticeLifecycle_FiveShutdownResults_NoDeadlock(t *testing.T) {
	workers := []*noticeLifecycleProbe{newNoticeLifecycleProbe(), newNoticeLifecycleProbe(), newNoticeLifecycleProbe()}
	servers := []*lifecycleServerFake{newLifecycleServer("business"), newLifecycleServer("metrics")}
	arrived, release := make(chan struct{}, 2), make(chan struct{})
	for _, server := range servers {
		server.shutdown = func(_ string, _ context.Context) error {
			arrived <- struct{}{}
			<-release
			server.stop(http.ErrServerClosed)
			return nil
		}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	var closes atomic.Int32
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{
			businessServer: servers[0], metricsServer: servers[1], businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{},
			articleWorker: workers[0], passwordResetWorker: workers[1], noticeWorker: workers[2], closeArticle: func() error { closes.Add(1); return nil },
			closePool: func() error { closes.Add(1); return nil }, shutdownBudget: time.Second, signals: signals})
	}()
	noticeAwait(t, servers[0].started)
	noticeAwait(t, servers[1].started)
	signals <- syscall.SIGTERM
	noticeAwait(t, arrived)
	noticeAwait(t, arrived)
	close(release)
	if err := noticeAwait(t, result); err != nil {
		t.Fatal(err)
	}
	for _, worker := range workers {
		if worker.waitCalls != 1 || worker.stopCalls != 1 {
			t.Fatal("worker not stopped/joined once")
		}
	}
	if closes.Load() != 2 {
		t.Fatal("resource close count changed")
	}
}

func TestNoticeLifecycle_CompetingFailures_ErrorPriorityAndCloseOnce(t *testing.T) {
	for primary := 0; primary < 5; primary++ {
		t.Run(strconv.Itoa(primary), func(t *testing.T) { testNoticeFailurePriority(t, primary) })
	}
}

func testNoticeFailurePriority(t *testing.T, primary int) {
	failures := []error{errors.New("business fatal"), errors.New("metrics fatal"), errors.New("password reset fatal"), errors.New("deletion fatal"), errors.New("notice fatal")}
	workers := []*noticeLifecycleProbe{newNoticeLifecycleProbe(), newNoticeLifecycleProbe(), newNoticeLifecycleProbe()}
	for index, worker := range workers {
		rank := []int{3, 2, 4}[index]
		if rank >= primary {
			worker.stopFailure = failures[rank]
		}
	}
	servers := []*lifecycleServerFake{newLifecycleServer("business"), newLifecycleServer("metrics")}
	for index, server := range servers {
		server.shutdown = func(_ string, _ context.Context) error {
			if index >= primary {
				server.stop(failures[index])
			} else {
				server.stop(http.ErrServerClosed)
			}
			return nil
		}
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	var closes atomic.Int32
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{businessServer: servers[0], metricsServer: servers[1],
			businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{},
			articleWorker: workers[0], passwordResetWorker: workers[1], noticeWorker: workers[2],
			closeArticle: func() error { closes.Add(1); return nil }, closePool: func() error { closes.Add(1); return nil }, shutdownBudget: time.Second, signals: signals})
	}()
	noticeAwait(t, servers[0].started)
	noticeAwait(t, servers[1].started)
	signals <- syscall.SIGTERM
	err := noticeAwait(t, result)
	if !errors.Is(firstLifecycleError(err), failures[primary]) {
		t.Fatalf("primary=%v, want %v", firstLifecycleError(err), failures[primary])
	}
	for index := primary; index < len(failures); index++ {
		if !errors.Is(err, failures[index]) {
			t.Errorf("lost cause %v", failures[index])
		}
	}
	if closes.Load() != 2 {
		t.Fatal("resource closes changed")
	}
}

func TestNoticeLifecycle_InitialDeadline_WaitsForActualExit(t *testing.T) {
	p := newNoticeLifecycleProbe()
	entered, cancelSeen, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	p.reconcile = func(ctx context.Context) error {
		close(entered)
		<-ctx.Done()
		close(cancelSeen)
		<-release
		return ctx.Err()
	}
	signals := make(chan os.Signal, 1)
	result := make(chan error, 1)
	var closeCalls atomic.Int32
	shared, cancelShared := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	defer cancelShared()
	go func() {
		result <- runLifecycle(context.Background(), lifecycleConfig{businessServer: newLifecycleServer("business"), metricsServer: newLifecycleServer("metrics"),
			businessListener: &lifecycleListenerFake{}, metricsListener: &lifecycleListenerFake{}, noticeWorker: p,
			closePool: func() error { closeCalls.Add(1); return nil }, shutdownBudget: time.Second, signals: signals,
			deadlineContext: func(context.Context, time.Duration) (context.Context, context.CancelFunc) {
				return shared, cancelShared
			}})
	}()
	noticeAwait(t, entered)
	signals <- syscall.SIGTERM
	if got := noticeAwait(t, p.bound); got != shared {
		t.Error("notice received another shutdown context")
	}
	noticeAwait(t, cancelSeen)
	cancelShared()
	select {
	case <-p.Done():
		t.Error("Done closed while reconcile is active")
	default:
	}
	if closeCalls.Load() != 0 {
		t.Error("pool closed before actual initial exit")
	}
	close(release)
	if err := noticeAwait(t, result); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("shared timeout/cancellation suppressed: %v", err)
	}
	if closeCalls.Load() != 1 {
		t.Fatal("pool was not closed once")
	}
}

// Keep the deadline live so it cannot mask an early lost shutdown cause.
func TestRunLifecycleAdditionalServers_ShutdownCauseIsPreserved(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	defer cancel()
	failure := errors.New("maintenance shutdown failed before deadline")
	results := make(chan lifecycleOperationResult, 1)
	results <- lifecycleOperationResult{name: "maintenance", err: failure}
	close(results)
	run := lifecycleRun{shutdownCtx: ctx, priority: lifecycleUnsetPriority}
	if !run.collectShutdown(results) {
		t.Error("maintenance shutdown failure did not require Close fallback")
	}
	if !errors.Is(run.result.err(), failure) {
		t.Error("maintenance shutdown cause was lost")
	}
}

func TestRunLifecycleAdditionalServers_NilRegistrationHasNoSideEffects(t *testing.T) {
	business, metrics := newLifecycleServer("business"), newLifecycleServer("metrics")
	worker := newLifecycleWorkerFake()
	var closes atomic.Int32
	listener := &lifecycleListenerFake{closeFn: func() error { closes.Add(1); return nil }}
	config := lifecycleConfig{businessServer: business, metricsServer: metrics,
		businessListener: listener, metricsListener: listener,
		additionalServers: []*lifecycleServerState{nil},
		articleWorker:     worker, closeArticle: func() error { closes.Add(1); return nil },
		closePool:      func() error { closes.Add(1); return nil },
		shutdownBudget: time.Second, signals: make(chan os.Signal)}
	if err := runLifecycle(context.Background(), config); err == nil {
		t.Fatal("nil additional server was accepted")
	}
	for _, server := range []*lifecycleServerFake{business, metrics} {
		select {
		case <-server.started:
			t.Error("server started after invalid registration")
		default:
		}
		if server.shutdownCalls != 0 || server.closeCalls != 0 {
			t.Error("invalid registration touched server lifecycle")
		}
	}
	if worker.reconcileCalls != 0 || worker.startCalls != 0 || worker.stopCalls != 0 || worker.waitCalls != 0 {
		t.Error("invalid registration touched worker lifecycle")
	}
	if closes.Load() != 0 {
		t.Error("invalid registration closed caller-owned resources")
	}
}

func TestRunLifecycleAdditionalServers_ShutdownFailureClosesAndJoins(t *testing.T) {
	business, metrics := newLifecycleServer("business"), newLifecycleServer("metrics")
	maintenance := newLifecycleServer("maintenance")
	servers := []*lifecycleServerFake{business, metrics, maintenance}
	shutdownFailure, closeFailure := errors.New("maintenance shutdown failure"), errors.New("maintenance close failure")
	closeEntered, finished := make(chan struct{}), make(chan struct{})
	shutdownContexts := make(chan context.Context, len(servers))
	for _, server := range servers {
		server.shutdown = func(_ string, ctx context.Context) error {
			shutdownContexts <- ctx
			if server == maintenance {
				return shutdownFailure
			}
			server.stop(http.ErrServerClosed)
			return nil
		}
	}
	maintenance.close = func(*lifecycleServerFake) error { close(closeEntered); return closeFailure }
	var closes atomic.Int32
	listener := func() net.Listener {
		return &lifecycleListenerFake{closeFn: func() error { closes.Add(1); return nil }}
	}
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	var result error
	t.Cleanup(func() { joinAdditionalLifecycleTest(cancel, servers, finished) })
	go func() {
		result = runLifecycle(ctx, lifecycleConfig{businessServer: business, metricsServer: metrics,
			businessListener: listener(), metricsListener: listener(),
			additionalServers: []*lifecycleServerState{{name: "maintenance", server: maintenance, listener: listener()}},
			closePool:         func() error { closes.Add(1); return nil }, shutdownBudget: time.Hour, signals: signals})
		close(finished)
	}()
	for _, server := range servers {
		noticeAwait(t, server.started)
	}
	signals <- syscall.SIGTERM
	noticeAwait(t, closeEntered)
	assertAdditionalLifecycleJoin(t, shutdownContexts, finished, &closes, maintenance, servers)
	noticeAwait(t, finished)
	if !errors.Is(result, shutdownFailure) || !errors.Is(result, closeFailure) {
		t.Fatalf("lost shutdown/close causes: %v", result)
	}
	if closes.Load() != 4 {
		t.Fatalf("listener/pool closes = %d, want 4", closes.Load())
	}
}

func joinAdditionalLifecycleTest(cancel context.CancelFunc, servers []*lifecycleServerFake, finished <-chan struct{}) {
	cancel()
	for _, server := range servers {
		server.stop(http.ErrServerClosed)
	}
	<-finished
}

func assertAdditionalLifecycleJoin(t *testing.T, contexts <-chan context.Context, finished <-chan struct{}, closes *atomic.Int32, held *lifecycleServerFake, servers []*lifecycleServerFake) {
	t.Helper()
	first := noticeAwait(t, contexts)
	for range len(servers) - 1 {
		if got := noticeAwait(t, contexts); got != first {
			t.Error("third server received a different shutdown context")
		}
	}
	if first.Err() != nil {
		t.Error("deadline expired and masked early shutdown failure")
	}
	select {
	case <-finished:
		t.Error("coordinator returned before held Serve actually ended")
	default:
	}
	if closes.Load() != 0 {
		t.Error("listener/pool closed while maintenance Serve was still held")
	}
	held.stop(http.ErrServerClosed)
	for _, server := range servers {
		if server.closeCalls != 1 {
			t.Errorf("%s Close calls = %d, want 1", server.name, server.closeCalls)
		}
	}
}

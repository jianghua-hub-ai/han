package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/identity"
	"han/internal/recovery"
)

// recoveryTestRuntime is a package-local runtime double. It never mints a real
// capability: PhasePermit stays opaque and zero-valued, and the fake only
// replays the observe sequence a real runtime performs for the bridge.
type recoveryTestRuntime struct {
	mu          sync.Mutex
	checkErr    error
	phaseErr    error
	readErr     error
	lastPurpose string
	binding     recovery.PermitBinding
	checks      int
	reads       int
	runNormal   func(context.Context) error
}

func (r *recoveryTestRuntime) Install(context.Context, recovery.InstallationManifest) error {
	return recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) RunNormal(ctx context.Context) error {
	if r.runNormal != nil {
		return r.runNormal(ctx)
	}
	return recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) Enter(context.Context, recovery.RecoveryInput) error {
	return recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) Resume(context.Context, recovery.RunBinding) error {
	return recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) Verify(context.Context) (recovery.AdmissionObservation, error) {
	return recovery.AdmissionObservation{}, recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) Acquire(context.Context, recovery.WorkKind) (recovery.WorkLease, error) {
	return nil, recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) CheckPhase(_ context.Context, _ recovery.PhasePermit, purpose string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks++
	if r.checkErr != nil {
		return r.checkErr
	}
	r.lastPurpose = purpose
	return nil
}

func (r *recoveryTestRuntime) PhaseContext(ctx context.Context, _ recovery.PhasePermit) (context.Context, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.phaseErr != nil {
		return nil, r.phaseErr
	}
	return ctx, nil
}

func (r *recoveryTestRuntime) ReadPermitBinding(context.Context) (recovery.PermitBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	if r.readErr != nil {
		return recovery.PermitBinding{}, r.readErr
	}
	binding := r.binding
	if binding.Purpose == "" {
		binding.Purpose = r.lastPurpose
	}
	return binding, nil
}

func (r *recoveryTestRuntime) CheckJournalUse(context.Context, string, string) error {
	return nil
}

func (r *recoveryTestRuntime) CheckArticleCoordination(context.Context, recovery.PhasePermit, string, string, uint64) (recovery.AnchorSnapshot, string, error) {
	return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) PauseArticleAppend(context.Context, string) (recovery.AppendBarrier, error) {
	return recovery.AppendBarrier{}, recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) CompleteArticleAppend(context.Context, recovery.AppendBarrier, recovery.VerifiedHeadAdvance) error {
	return recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) FaultArticleAppend(context.Context, recovery.AppendBarrier, error) error {
	return recovery.ErrAdmissionUnsupported
}

func (r *recoveryTestRuntime) Stop(context.Context) error { return nil }

func (r *recoveryTestRuntime) ShutdownContext() (context.Context, bool) { return nil, false }

type recoveryTestExecutor struct {
	mu          sync.Mutex
	calls       []string
	bindErr     error
	finalizeErr error
	prepareErr  error
	completeErr error
	revokeErr   error
	onCall      func(string)
}

func (e *recoveryTestExecutor) record(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls = append(e.calls, name)
	onCall := e.onCall
	e.mu.Unlock()
	if onCall != nil {
		onCall(name)
	}
	e.mu.Lock()
}

func (e *recoveryTestExecutor) count(name string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	total := 0
	for _, call := range e.calls {
		if call == name {
			total++
		}
	}
	return total
}

func (e *recoveryTestExecutor) BindRestoredRun(context.Context, recovery.PhasePermit, recovery.RunBinding, recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	e.record("bind")
	return recovery.RecoveryRun{}, e.bindErr
}

func (e *recoveryTestExecutor) FinalizeRestoredIdentity(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	e.record("finalize")
	return e.finalizeErr
}

func (e *recoveryTestExecutor) PrepareRecoveryTrust(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte) error {
	e.record("prepare")
	return e.prepareErr
}

func (e *recoveryTestExecutor) CompleteIdentityReview(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	e.record("complete")
	return e.completeErr
}

func (e *recoveryTestExecutor) RevokeRecoverySessions(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	e.record("revoke")
	return e.revokeErr
}

type recoveryTestExit struct {
	mu     sync.Mutex
	calls  int
	err    error
	order  *[]string
	onExit func(context.Context)
}

func (x *recoveryTestExit) ExitRecovery(ctx context.Context, _ recovery.PhasePermit, _ recovery.RunBinding) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.calls++
	if x.order != nil {
		*x.order = append(*x.order, "exit")
	}
	if x.onExit != nil {
		x.onExit(ctx)
	}
	return x.err
}

type recoveryTestLifecycle struct{}

func (recoveryTestLifecycle) ReconcileAndStart(context.Context, recovery.PhasePermit) error {
	return nil
}

func (recoveryTestLifecycle) StopAndJoin(context.Context) error { return nil }

func recoveryTestPermitBinding() recovery.PermitBinding {
	return recovery.PermitBinding{
		RuntimeID:      "runtime-1",
		Generation:     3,
		RunID:          "run-1",
		BackupRef:      "backup-1",
		InstallationID: "installation-1",
	}
}

func newRecoveryTestBridge(t *testing.T) (*recoveryIdentityBridge, *recoveryTestRuntime, *recoveryTestExecutor) {
	t.Helper()
	runtime := &recoveryTestRuntime{binding: recoveryTestPermitBinding()}
	executor := &recoveryTestExecutor{}
	bridge, err := newRecoveryIdentityBridge(runtime, executor, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return bridge, runtime, executor
}

func TestRecoveryRuntimeComposition(t *testing.T) {
	t.Run("rejects_incomplete_ports", func(t *testing.T) {
		secret, err := newRecoverySecretSource(func(context.Context, recovery.RunBinding) ([]byte, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		article, err := newRecoveryControlHandle(func() error { return nil }, func(context.Context) error { return nil }, nil)
		if err != nil {
			t.Fatal(err)
		}
		for name, ports := range map[string]recoveryRuntimePorts{
			"missing_runtime":   {Identity: &recoveryTestExecutor{}, Exit: &recoveryTestExit{}, Secret: secret, Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: time.Second},
			"missing_identity":  {Runtime: &recoveryTestRuntime{}, Exit: &recoveryTestExit{}, Secret: secret, Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: time.Second},
			"missing_article":   {Runtime: &recoveryTestRuntime{}, Identity: &recoveryTestExecutor{}, Exit: &recoveryTestExit{}, Secret: secret, Lifecycle: recoveryTestLifecycle{}, JoinBudget: time.Second},
			"missing_budget":    {Runtime: &recoveryTestRuntime{}, Identity: &recoveryTestExecutor{}, Exit: &recoveryTestExit{}, Secret: secret, Lifecycle: recoveryTestLifecycle{}, Article: article},
			"missing_lifecycle": {Runtime: &recoveryTestRuntime{}, Identity: &recoveryTestExecutor{}, Exit: &recoveryTestExit{}, Secret: secret, Article: article, JoinBudget: time.Second},
		} {
			t.Run(name, func(t *testing.T) {
				if _, err := newRecoveryRuntimeComposition(ports); !errors.Is(err, recovery.ErrAdmissionBinding) {
					t.Fatalf("incomplete ports accepted: %v", err)
				}
			})
		}
	})

	t.Run("exit_joins_revokes_and_settles_once", func(t *testing.T) {
		binding := recoveryTestPermitBinding()
		runtime := &recoveryTestRuntime{binding: binding}
		executor := &recoveryTestExecutor{}
		var mu sync.Mutex
		var order []string
		record := func(name string) {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, name)
		}
		exit := &recoveryTestExit{order: &order}
		article, err := newRecoveryControlHandle(
			func() error { record("close"); return nil },
			func(context.Context) error { record("join"); return nil },
			func() string { return "installed" },
		)
		if err != nil {
			t.Fatal(err)
		}
		secret, err := newRecoverySecretSource(func(context.Context, recovery.RunBinding) ([]byte, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		composition, err := newRecoveryRuntimeComposition(recoveryRuntimePorts{
			Runtime: runtime, Identity: executor, Exit: exit, Secret: secret,
			Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: 2 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		executor.onCall = record

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		run := recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef}
		var wait sync.WaitGroup
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				results <- composition.ExitRecovery(ctx, recovery.PhasePermit{}, run)
			}()
		}
		wait.Wait()
		close(results)
		for result := range results {
			if result != nil {
				t.Fatalf("exit failed: %v", result)
			}
		}
		if exit.calls != 1 {
			t.Fatalf("exit ran %d times", exit.calls)
		}
		if executor.count("revoke") != 1 {
			t.Fatalf("revoke ran %d times", executor.count("revoke"))
		}
		mu.Lock()
		observedOrder := append([]string(nil), order...)
		mu.Unlock()
		want := []string{"exit", "revoke", "close", "join"}
		if len(observedOrder) != len(want) {
			t.Fatalf("settle order = %v", observedOrder)
		}
		for i := range want {
			if observedOrder[i] != want[i] {
				t.Fatalf("settle order = %v, want %v", observedOrder, want)
			}
		}
		if article.State() != "installed" {
			t.Fatalf("article state = %q", article.State())
		}
	})

	t.Run("exit_rejects_invalid_replay_permit", func(t *testing.T) {
		binding := recoveryTestPermitBinding()
		runtime := &recoveryTestRuntime{binding: binding, checkErr: recovery.ErrAdmissionBinding}
		executor := &recoveryTestExecutor{}
		exit := &recoveryTestExit{}
		closed := 0
		joined := 0
		article, err := newRecoveryControlHandle(
			func() error { closed++; return nil },
			func(context.Context) error { joined++; return nil },
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		secret, err := newRecoverySecretSource(func(context.Context, recovery.RunBinding) ([]byte, error) { return nil, nil })
		if err != nil {
			t.Fatal(err)
		}
		composition, err := newRecoveryRuntimeComposition(recoveryRuntimePorts{
			Runtime: runtime, Identity: executor, Exit: exit, Secret: secret,
			Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		err = composition.ExitRecovery(context.Background(), recovery.PhasePermit{}, recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef})
		if !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("invalid permit accepted: %v", err)
		}
		if exit.calls != 0 || executor.count("revoke") != 0 || closed != 0 || joined != 0 {
			t.Fatalf("invalid permit reached side effects: exit=%d revoke=%d closed=%d joined=%d", exit.calls, executor.count("revoke"), closed, joined)
		}
	})
}

// recoveryJoinWorker is a lifecycle owner double that exposes the real started
// proof the Article settle path must consult.
type recoveryJoinWorker struct {
	started chan struct{}
	done    chan struct{}
}

func newRecoveryJoinWorker(started bool) *recoveryJoinWorker {
	worker := &recoveryJoinWorker{started: make(chan struct{}), done: make(chan struct{})}
	if started {
		close(worker.started)
	}
	return worker
}

func (w *recoveryJoinWorker) Reconcile(context.Context) error { return nil }
func (w *recoveryJoinWorker) Start(context.Context)           {}
func (w *recoveryJoinWorker) Stop()                           {}
func (w *recoveryJoinWorker) Wait(context.Context) error      { return nil }
func (w *recoveryJoinWorker) Done() <-chan struct{}           { return w.done }
func (w *recoveryJoinWorker) Err() error                      { return nil }
func (w *recoveryJoinWorker) Started() <-chan struct{}        { return w.started }

// F004: a replay exits before ReconcileAndStart, so the two owners that never
// started must not hold the settle until the drain budget expires.
func TestRecoveryArticleJoinSkipsUnstartedOwners(t *testing.T) {
	unstarted := newRecoveryJoinWorker(false)
	started := newRecoveryJoinWorker(true)
	join := newRecoveryArticleJoin(&businessRuntime{articleWorker: unstarted, passwordResetWorker: started})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	time.AfterFunc(10*time.Millisecond, func() { close(started.done) })
	if err := join(ctx); err != nil {
		t.Fatalf("settle waited for an owner that never started: %v", err)
	}
}

// F005: normal startup must run under the configured io budget, not the
// caller's parent deadline.
func TestRecoveryRuntimeStartHonoursIOBudget(t *testing.T) {
	blocking := &recoveryTestRuntime{runNormal: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	assembly := &recoveryRootAssembly{
		plan:        recoveryStartupPlan{action: "normal", present: true},
		runtime:     blocking,
		composition: &recoveryRuntimeComposition{},
		flowDone:    make(chan struct{}),
	}
	parent, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	began := time.Now()
	err := assembly.start(parent, 50*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("normal startup ignored the io budget: %v", err)
	}
	if elapsed := time.Since(began); elapsed > time.Second {
		t.Fatalf("normal startup waited %v instead of the io budget", elapsed)
	}
}

// F006: the surface join on the exit chain shares the join budget instead of
// the caller's unbounded context.
func TestRecoveryRuntimeCompositionExitUsesJoinBudget(t *testing.T) {
	binding := recoveryTestPermitBinding()
	runtime := &recoveryTestRuntime{binding: binding}
	executor := &recoveryTestExecutor{}
	var observed context.Context
	exit := &recoveryTestExit{onExit: func(ctx context.Context) { observed = ctx }}
	article, err := newRecoveryControlHandle(func() error { return nil }, func(context.Context) error { return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := newRecoverySecretSource(func(context.Context, recovery.RunBinding) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	composition, err := newRecoveryRuntimeComposition(recoveryRuntimePorts{
		Runtime: runtime, Identity: executor, Exit: exit, Secret: secret,
		Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: 40 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := composition.ExitRecovery(context.Background(), recovery.PhasePermit{}, recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef}); err != nil {
		t.Fatal(err)
	}
	if observed == nil {
		t.Fatal("surface exit was never reached")
	}
	deadline, ok := observed.Deadline()
	if !ok {
		t.Fatal("surface exit ran on the unbounded parent context")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
		t.Fatalf("surface exit budget = %v", remaining)
	}
}

// F001: the single runtime observes the exit router, so a real runtime exit
// must reach the root composition (capture, surface join, same-executor
// revoke, Article settle) exactly once. Before the fix the runtime was wired
// straight to the surface exit, which only joined and never revoked or
// settled; this test cannot compile against that wiring because there is no
// exit chain to observe.
func TestRecoveryRuntimeExitRouterReachesComposition(t *testing.T) {
	binding := recoveryTestPermitBinding()
	runtime := &recoveryTestRuntime{binding: binding}
	executor := &recoveryTestExecutor{}
	var mu sync.Mutex
	var order []string
	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, name)
	}
	exit := &recoveryTestExit{order: &order}
	article, err := newRecoveryControlHandle(
		func() error { record("close"); return nil },
		func(context.Context) error { record("join"); return nil },
		func() string { return "installed" },
	)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := newRecoverySecretSource(func(context.Context, recovery.RunBinding) ([]byte, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	composition, err := newRecoveryRuntimeComposition(recoveryRuntimePorts{
		Runtime: runtime, Identity: executor, Exit: exit, Secret: secret,
		Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	executor.onCall = record
	// Production wiring: the runtime port is the router, the composition port
	// is the maintenance surface join.
	router := &recoveryExitRouter{}
	if err := router.bind(composition); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run := recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- router.ExitRecovery(ctx, recovery.PhasePermit{}, run)
		}()
	}
	wait.Wait()
	close(results)
	for result := range results {
		if result != nil {
			t.Fatalf("runtime exit failed: %v", result)
		}
	}
	if exit.calls != 1 || executor.count("revoke") != 1 {
		t.Fatalf("exit=%d revoke=%d, want exactly one each", exit.calls, executor.count("revoke"))
	}
	mu.Lock()
	observed := append([]string(nil), order...)
	mu.Unlock()
	want := []string{"exit", "revoke", "close", "join"}
	if len(observed) != len(want) {
		t.Fatalf("runtime exit order = %v, want %v", observed, want)
	}
	for i := range want {
		if observed[i] != want[i] {
			t.Fatalf("runtime exit order = %v, want %v", observed, want)
		}
	}
	if article.State() != "installed" {
		t.Fatalf("article state = %q", article.State())
	}
}

// F001: an unbound exit chain refuses; it never reports a successful exit.
func TestRecoveryRuntimeExitRouterRefusesBeforeBind(t *testing.T) {
	router := &recoveryExitRouter{}
	err := router.ExitRecovery(context.Background(), recovery.PhasePermit{}, recovery.RunBinding{RunID: "run-1", BackupRef: "backup-1"})
	if !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("unbound exit chain reported success: %v", err)
	}
}

// recoveryTestControlServer blocks in Serve until Shutdown is called, so a
// join can only complete if the wrapper's own Serve actually ran and closed
// its done channel. It mints no capability and opens no real port.
type recoveryTestControlServer struct {
	started   chan struct{}
	stop      chan struct{}
	once      sync.Once
	mu        sync.Mutex
	shutdowns int
}

func newRecoveryTestControlServer() *recoveryTestControlServer {
	return &recoveryTestControlServer{started: make(chan struct{}), stop: make(chan struct{})}
}

func (s *recoveryTestControlServer) Serve(net.Listener) error {
	close(s.started)
	<-s.stop
	return http.ErrServerClosed
}

func (s *recoveryTestControlServer) Shutdown(context.Context) error {
	s.mu.Lock()
	s.shutdowns++
	s.mu.Unlock()
	s.once.Do(func() { close(s.stop) })
	return nil
}

func (s *recoveryTestControlServer) Close() error { return s.Shutdown(context.Background()) }

func (s *recoveryTestControlServer) shutdownCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdowns
}

// F003: the lifecycle must serve the joined wrapper, not the raw server. The
// pre-fix wiring handed the raw server to the lifecycle, so the wrapper's
// Serve never ran and the recovery exit join waited on a done channel that
// could never close.
func TestRecoveryControlReleaseIsJoined(t *testing.T) {
	assembly := &recoveryRootAssembly{plan: recoveryStartupPlan{maintenance: true}, exit: &recoverySurfaceExit{}}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("open loopback listener: %v", err)
	}
	defer listener.Close()
	server := newRecoveryTestControlServer()
	joined, err := assembly.bindControl(server, listener)
	if err != nil {
		t.Fatalf("bind control surface: %v", err)
	}
	if joined == nil {
		t.Fatal("bind control returned no served wrapper")
	}
	if assembly.control != joined {
		t.Fatal("the assembly retained a surface other than the one it returned")
	}
	served := make(chan error, 1)
	go func() { served <- joined.Serve(listener) }()
	select {
	case <-server.started:
	case <-time.After(2 * time.Second):
		t.Fatal("the served wrapper never reached Serve")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := joined.join(ctx); err != nil {
		t.Fatalf("join the served control surface: %v", err)
	}
	select {
	case err := <-served:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("served wrapper returned %v, want http.ErrServerClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not return after join")
	}
	if count := server.shutdownCount(); count != 1 {
		t.Fatalf("shutdown calls = %d, want 1", count)
	}

	// Defect shape: joining a wrapper whose Serve never ran can only time out,
	// which is exactly what the raw-server wiring produced.
	rawListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("open second loopback listener: %v", err)
	}
	defer rawListener.Close()
	raw := newRecoveryTestControlServer()
	neverServed, err := newRecoveryJoinedServer(raw)
	if err != nil {
		t.Fatalf("build unserved wrapper: %v", err)
	}
	rawServed := make(chan error, 1)
	go func() { rawServed <- raw.Serve(rawListener) }()
	select {
	case <-raw.started:
	case <-time.After(2 * time.Second):
		t.Fatal("the raw server never started")
	}
	deadline, cancelDeadline := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelDeadline()
	if err := neverServed.join(deadline); err == nil {
		t.Fatal("join completed although the wrapper's Serve never ran")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unserved wrapper join = %v, want a deadline", err)
	}
	_ = fmt.Sprintf("%v", <-rawServed)
}

// TestRecoveryControlAuthorizerRequiresSealedExecutor proves the maintenance
// authorizer is bound to the sealed executor rather than falling back to the
// ordinary session store: an unsealed forwarder refuses closed.
func TestRecoveryControlAuthorizerRequiresSealedExecutor(t *testing.T) {
	forwarder := identity.NewRecoveryIdentityForwarder()
	authorizer, err := newRecoveryControlAuthorizer(forwarder, identity.PasswordResetStore{}, func() time.Time { return time.Now() })
	if !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("unsealed executor authorizer error = %v, want ErrAdmissionBinding", err)
	}
	_ = authorizer
}

// recoveryAssemblyJournal is a construction-only journal fixture. The assembly
// stores it but never calls a journal method while wiring the runtime, so the
// embedded interface stays nil on purpose.
type recoveryAssemblyJournal struct {
	recovery.ContextRecoveryJournal
}

// recoveryAssemblyHasher is a construction-only password hasher fixture. The
// assembly stores it in the sealed executor; construction never hashes.
type recoveryAssemblyHasher struct{}

func (recoveryAssemblyHasher) Hash(string) (string, error) { return "sha256:assembly", nil }

func (recoveryAssemblyHasher) Verify(string, string) (bool, bool, error) { return false, false, nil }

// recoveryAssemblyRuntimeProbe records the verify/start dispatch while the
// shared recoveryTestRuntime supplies the rest of recovery.Runtime.
type recoveryAssemblyRuntimeProbe struct {
	*recoveryTestRuntime
	verifies int
	normals  int
}

func (p *recoveryAssemblyRuntimeProbe) Verify(context.Context) (recovery.AdmissionObservation, error) {
	p.verifies++
	return recovery.AdmissionObservation{}, nil
}

func (p *recoveryAssemblyRuntimeProbe) RunNormal(context.Context) error {
	p.normals++
	return nil
}

// F009: the real constructor must hand the built Runtime to the assembly. Before
// the fix newRecoveryRootAssembly left assembly.runtime nil, so bindWorkers,
// verify, start and joinWork refused with ErrAdmissionBinding on every
// normal/verify/resume startup. This is the only test that drives the real
// constructor instead of a struct literal.
func TestRecoveryRootAssemblyWiresRuntime(t *testing.T) {
	originalJournal, originalKeyring := recoveryOpenJournal, recoveryLoadKeyring
	t.Cleanup(func() { recoveryOpenJournal, recoveryLoadKeyring = originalJournal, originalKeyring })
	recoveryLoadKeyring = func(string) (recovery.Keyring, error) { return recovery.Keyring{}, nil }
	recoveryOpenJournal = func(context.Context, recovery.JournalOptions, recovery.Keyring) (recovery.ContextRecoveryJournal, error) {
		return recoveryAssemblyJournal{}, nil
	}
	values := recoveryConfigFixture(t)
	plan, err := planRecoveryStartup(func(key string) string { return values[key] }, &recoveryCommand{action: "verify"})
	if err != nil || !plan.present || !plan.verify {
		t.Fatalf("verify plan = %+v, %v", plan, err)
	}
	assembly, err := newRecoveryRootAssembly(context.Background(), recoveryRootInputs{
		pool: &pgxpool.Pool{},
		plan: plan,
		prepared: &preparedLoginDependencies{
			clock:     mainLoginClock{now: time.Date(2026, time.October, 31, 3, 0, 0, 0, time.UTC)},
			ids:       mainLoginIDs{},
			passwords: recoveryAssemblyHasher{},
		},
	})
	if err != nil {
		t.Fatalf("production assembly refused a valid verify plan: %v", err)
	}
	if assembly.runtime == nil {
		t.Fatal("assembly.runtime is nil: bindWorkers/verify/start refuse on every startup")
	}
	if got := assembly.composition.runtime(); got == nil || got != assembly.runtime {
		t.Fatalf("assembly.runtime = %v, composition runtime = %v (must be one value)", assembly.runtime, got)
	}
	if err := assembly.bindWorkers(nil, nil, nil); err != nil {
		t.Fatalf("bindWorkers refused a verify plan after the runtime fix: %v", err)
	}
	// verify and start must dispatch through a.runtime rather than the nil guard.
	// Swap in a probe so the real recovery work is not attempted on a fixture.
	probe := &recoveryAssemblyRuntimeProbe{recoveryTestRuntime: &recoveryTestRuntime{}}
	assembly.runtime = probe
	if err := assembly.verify(context.Background(), time.Second); err != nil {
		t.Fatalf("verify refused after the runtime fix: %v", err)
	}
	if probe.verifies != 1 {
		t.Fatalf("verify calls = %d, want 1", probe.verifies)
	}
	assembly.plan.verify = false
	assembly.plan.action = "normal"
	if err := assembly.start(context.Background(), time.Second); err != nil {
		t.Fatalf("start refused after the runtime fix: %v", err)
	}
	if probe.normals != 1 {
		t.Fatalf("RunNormal calls = %d, want 1", probe.normals)
	}
}

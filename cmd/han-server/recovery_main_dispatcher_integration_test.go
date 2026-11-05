//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/article"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
)

// TestRecoveryMainBusinessDispatcher covers the production cmd/han-server
// business dispatcher, not a standalone mux or recovery fixture.
func TestRecoveryMainBusinessDispatcher(t *testing.T) {
	t.Run("ExactRouteBeforeGuard", testRecoveryMainExactRouteBeforeGuard)
	t.Run("ClosedUnknownTwoAdminGets503", testRecoveryMainClosedUnknownTwoAdminGets503)
	t.Run("LiveReadyProbes", testRecoveryMainLiveReadyProbes)
	t.Run("PublicReadyAdmitsSameSession", testRecoveryMainPublicReadyAdmitsSameSession)
	t.Run("SingleRuntimeHandle", testRecoveryMainSingleRuntimeHandle)
	t.Run("NoRecoveryConfig", testRecoveryMainNoRecoveryConfig)
}

func testRecoveryMainExactRouteBeforeGuard(t *testing.T) {
	runtime := newS419RecoveryRuntime("closed")
	fixture := newS419ProductionDispatcher(t, runtime.handle, nil)

	unknown := httptest.NewRecorder()
	fixture.handler.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/not-a-route", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown path status=%d, want 404", unknown.Code)
	}
	if got := unknown.Header().Get(platform.TraceHeader); got == "" {
		t.Fatal("unknown path lost Trace")
	}

	wrong := httptest.NewRecorder()
	fixture.handler.ServeHTTP(wrong, httptest.NewRequest(http.MethodPost, "/api/v1/admin/articles", nil))
	if wrong.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status=%d, want 405", wrong.Code)
	}
	if got := wrong.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("wrong method Allow=%q, want GET", got)
	}

	wrong = httptest.NewRecorder()
	fixture.handler.ServeHTTP(wrong, httptest.NewRequest(http.MethodDelete, "/api/v1/articles", nil))
	if wrong.Code != http.StatusMethodNotAllowed {
		t.Fatalf("collection wrong method status=%d, want 405", wrong.Code)
	}
	if got := wrong.Header().Get("Allow"); got != "GET, POST" {
		t.Fatalf("collection Allow=%q, want %q", got, "GET, POST")
	}

	if acquires, _, _ := runtime.snapshot(); acquires != 0 {
		t.Fatalf("guard acquired %d lease(s) before exact routing decisions", acquires)
	}
	s419AssertOnlyTrace(t, fixture.events)
}

func testRecoveryMainClosedUnknownTwoAdminGets503(t *testing.T) {
	for _, state := range []string{"closed", "unknown"} {
		t.Run(state, func(t *testing.T) {
			runtime := newS419RecoveryRuntime(state)
			fixture := newS419ProductionDispatcher(t, runtime.handle, nil)
			for _, path := range []string{
				"/api/v1/admin/articles",
				"/api/v1/admin/articles/" + articleAuthorRootID,
			} {
				response := fixture.serve(t, fixture.request(path), http.StatusServiceUnavailable)
				var envelope platform.Envelope
				if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
					t.Fatalf("decode %s envelope: %v", path, err)
				}
				if envelope.Code != platform.CodePlatformRecoveryInProgress {
					t.Fatalf("%s code=%q, want %q", path, envelope.Code, platform.CodePlatformRecoveryInProgress)
				}
				if envelope.TraceID == "" || envelope.TraceID != response.Header().Get(platform.TraceHeader) {
					t.Fatalf("%s envelope Trace=%q, header=%q", path, envelope.TraceID, response.Header().Get(platform.TraceHeader))
				}
				s419AssertNoRecoveryLeak(t, response.Body.Bytes())
			}
			if fixture.repository.lists+fixture.repository.gets != 0 {
				t.Fatalf("guard reached Article repository: lists=%d gets=%d", fixture.repository.lists, fixture.repository.gets)
			}
			if fixture.reader.calls != 0 {
				t.Fatalf("guard reached author reader %d time(s)", fixture.reader.calls)
			}
			if len(fixture.resolver.inputs) != 0 {
				t.Fatalf("guard reached Session %d time(s)", len(fixture.resolver.inputs))
			}
			s419AssertOnlyTrace(t, fixture.events)
			if acquires, _, releases := runtime.snapshot(); acquires != 2 || releases != 0 {
				t.Fatalf("closed acquire/release=(%d,%d), want (2,0)", acquires, releases)
			}
			wantErr := recovery.ErrAdmissionClosed
			if state == "unknown" {
				wantErr = s419ErrAdmissionUnknown
			}
			if err := runtime.acquireError(); !errors.Is(err, wantErr) {
				t.Fatalf("%s acquire error=%v, want %v", state, err, wantErr)
			}
		})
	}
}

func testRecoveryMainLiveReadyProbes(t *testing.T) {
	runtime := newS419RecoveryRuntime("closed")
	fixture := newS419ProductionDispatcher(t, runtime.handle, nil)

	s419AssertProbe(t, fixture, http.MethodGet, "/livez", http.StatusOK, "alive")
	s419AssertProbe(t, fixture, http.MethodGet, "/readyz", http.StatusServiceUnavailable, "unavailable")
	// Liveness never authorizes business work: the very next ordinary GET is
	// still blocked while the shared runtime stays closed.
	fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusServiceUnavailable)
	if acquires, _, _ := runtime.snapshot(); acquires != 1 {
		t.Fatalf("closed probe acquire=%d, want 1", acquires)
	}
	for _, state := range []string{"unknown", "maintenance", "verify"} {
		runtime.setState(state)
		s419AssertProbe(t, fixture, http.MethodGet, "/readyz", http.StatusServiceUnavailable, "unavailable")
	}
	s419AssertProbeMethod(t, fixture, http.MethodPost, "/livez", http.MethodGet)
	s419AssertProbeMethod(t, fixture, http.MethodPost, "/readyz", http.MethodGet)
	for _, path := range []string{"/livezz", "/livez/", "/readyz/"} {
		response := httptest.NewRecorder()
		fixture.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404", path, response.Code)
		}
	}

	// Probe-only boundary: this fake runtime models the state the probes observe.
	// The real admission transition is covered by PublicReadyAdmitsSameSession
	// (real recovery.NewRuntime + RunNormal) and by layer (a)
	// TestRecoveryRuntimePostgres; it is not faked here.
	runtime.setState("public_ready")
	s419AssertProbe(t, fixture, http.MethodGet, "/readyz", http.StatusOK, "ready")
	fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusOK)

	if acquires, verifies, releases := runtime.snapshot(); acquires != 2 || releases != 1 {
		t.Fatalf("probe acquire/release=(%d,%d), want (2,1)", acquires, releases)
	} else if verifies == 0 {
		t.Fatal("readyz did not consult the shared runtime")
	}

	runtime.setState("closed")
	response := fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusServiceUnavailable)
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode guarded envelope: %v", err)
	}
	if envelope.Code != platform.CodePlatformRecoveryInProgress {
		t.Fatalf("liveness authorized ordinary work: code=%q", envelope.Code)
	}
}

// testRecoveryMainPublicReadyAdmitsSameSession drives one production dispatcher
// handle and one Session across the real admission boundary. The runtime is the
// real recovery.NewRuntime; it reaches public_ready through the actual RunNormal
// startup path, not through setState or a fake state machine.
//
// Layer (a): TestRecoveryRuntimePostgres (test/integration/recovery) owns the
// complete recovery chain (Begin -> Consume -> succeeded -> Exit/join ->
// activate). This layer (b) stops at the production dispatcher and reuses the
// real runtime's ordinary startup, because rebuilding the complete restore world
// is outside this bounded write set.
func testRecoveryMainPublicReadyAdmitsSameSession(t *testing.T) {
	ctx := context.Background()
	ordinary := newOrdinaryFixture(t)
	observed := &s419ObservedRuntime{Runtime: ordinary.runtime}
	fixture := newS419ProductionDispatcher(t, func() recovery.Runtime { return observed }, nil)
	request := fixture.request("/api/v1/admin/articles")
	if cookie, err := request.Cookie(platform.SessionCookieName); err != nil || cookie.Value != "admin-session-token" {
		t.Fatalf("same Session cookie=%v err=%v, want admin-session-token", cookie, err)
	}

	// Closed: the real runtime has not started ordinary work yet. Readiness is
	// not probed first because Verify closes a temporary exclusive lease on a
	// closed runtime, which would retire this fixture's in-memory authority
	// before RunNormal can start.
	fixture.events = nil
	response := fixture.serve(t, request, http.StatusServiceUnavailable)
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode closed envelope: %v", err)
	}
	if envelope.Code != platform.CodePlatformRecoveryInProgress {
		t.Fatalf("closed code=%q, want %q", envelope.Code, platform.CodePlatformRecoveryInProgress)
	}
	if envelope.TraceID == "" || envelope.TraceID != response.Header().Get(platform.TraceHeader) {
		t.Fatalf("closed envelope Trace=%q header=%q", envelope.TraceID, response.Header().Get(platform.TraceHeader))
	}
	s419AssertNoRecoveryLeak(t, response.Body.Bytes())
	if fixture.repository.lists+fixture.repository.gets != 0 {
		t.Fatalf("closed guard reached Article repository: lists=%d gets=%d", fixture.repository.lists, fixture.repository.gets)
	}
	if fixture.reader.calls != 0 {
		t.Fatalf("closed guard reached author reader %d time(s)", fixture.reader.calls)
	}
	if len(fixture.resolver.inputs) != 0 {
		t.Fatalf("closed guard reached Session %d time(s)", len(fixture.resolver.inputs))
	}
	s419AssertOnlyTrace(t, fixture.events)
	if acquires, releases := observed.counts(); acquires != 1 || releases != 0 {
		t.Fatalf("closed acquire/release=(%d,%d), want (1,0)", acquires, releases)
	}

	if err := ordinary.runtime.RunNormal(ctx); err != nil {
		t.Fatalf("real runtime RunNormal() error = %v", err)
	}
	observation, err := ordinary.runtime.Verify(ctx)
	if err != nil {
		t.Fatalf("real runtime Verify() error = %v", err)
	}
	if observation.State != "public_ready" {
		t.Fatalf("real runtime state=%q, want public_ready", observation.State)
	}
	s419AssertProbe(t, fixture, http.MethodGet, "/readyz", http.StatusOK, "ready")

	// Same dispatcher, same handle, and the same Session after the real transition.
	fixture.events = nil
	response = fixture.serve(t, request, http.StatusOK)
	if cookie, err := request.Cookie(platform.SessionCookieName); err != nil || cookie.Value != "admin-session-token" {
		t.Fatalf("replayed Session cookie=%v err=%v, want admin-session-token", cookie, err)
	}
	if fixture.repository.lists == 0 {
		t.Fatal("public_ready did not reach the Article repository")
	}
	if fixture.reader.calls == 0 {
		t.Fatal("public_ready did not reach the author reader")
	}
	if len(fixture.resolver.inputs) != 1 {
		t.Fatalf("public_ready Session calls=%d, want 1", len(fixture.resolver.inputs))
	}
	s419AssertEvents(t, fixture.events, "trace", "session", "principal", "repository", "identity")
	if acquires, releases := observed.counts(); acquires != 2 || releases != 1 {
		t.Fatalf("public_ready acquire/release=(%d,%d), want (2,1)", acquires, releases)
	}

	// Failure negative control: a downstream repository failure must reach the
	// client as its own status/code, never as recovery_in_progress.
	beforeLists := fixture.repository.lists
	beforeInputs := len(fixture.resolver.inputs)
	beforeAcquires, beforeReleases := observed.counts()
	fixture.repository.err = errors.New("synthetic downstream repository failure")
	fixture.events = nil
	response = fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusServiceUnavailable)
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode failure envelope: %v", err)
	}
	if envelope.Code != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("downstream failure code=%q, want %q", envelope.Code, platform.CodePlatformDatabaseUnavailable)
	}
	if envelope.Code == platform.CodePlatformRecoveryInProgress || bytes.Contains(response.Body.Bytes(), []byte(platform.CodePlatformRecoveryInProgress)) {
		t.Fatal("downstream failure was masked by platform.recovery_in_progress")
	}
	if envelope.TraceID == "" || envelope.TraceID != response.Header().Get(platform.TraceHeader) {
		t.Fatalf("failure envelope Trace=%q header=%q", envelope.TraceID, response.Header().Get(platform.TraceHeader))
	}
	if fixture.repository.lists != beforeLists+1 {
		t.Fatalf("downstream repository calls=%d, want %d", fixture.repository.lists, beforeLists+1)
	}
	if len(fixture.resolver.inputs) != beforeInputs+1 {
		t.Fatalf("downstream Session calls=%d, want %d", len(fixture.resolver.inputs), beforeInputs+1)
	}
	if acquires, releases := observed.counts(); acquires != beforeAcquires+1 || releases != beforeReleases+1 {
		t.Fatalf("downstream acquire/release=(%d,%d), want (%d,%d)", acquires, releases, beforeAcquires+1, beforeReleases+1)
	}
}

func testRecoveryMainSingleRuntimeHandle(t *testing.T) {
	t.Run("LateBinding", func(t *testing.T) {
		var handle recovery.Runtime
		provider := func() recovery.Runtime { return handle }
		fixture := newS419ProductionDispatcher(t, provider, nil)

		fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusOK)
		if len(fixture.resolver.inputs) != 1 {
			t.Fatalf("unbound dispatcher Session calls=%d, want 1", len(fixture.resolver.inputs))
		}

		runtime := newS419RecoveryRuntime("closed")
		handle = runtime
		fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusServiceUnavailable)
		if acquires, _, _ := runtime.snapshot(); acquires != 1 {
			t.Fatalf("late-bound acquire count=%d, want 1", acquires)
		}

		runtime.setState("public_ready")
		fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusOK)
		if acquires, _, releases := runtime.snapshot(); acquires != 2 || releases != 1 {
			t.Fatalf("single handle acquire/release=(%d,%d), want (2,1)", acquires, releases)
		}
	})

	t.Run("ReleaseAfterPanic", func(t *testing.T) {
		runtime := newS419RecoveryRuntime("public_ready")
		fixture := newS419ProductionDispatcher(t, runtime.handle, func(seam *articleProductionTestSeam) {
			seam.adminQueryHandlers[0] = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				panic("handler panic after guard acquisition")
			})
		})
		s419ServePanicking(t, fixture.handler, fixture.request("/api/v1/admin/articles"))
		if acquires, _, releases := runtime.snapshot(); acquires != 1 || releases != 1 {
			t.Fatalf("panic acquire/release=(%d,%d), want (1,1)", acquires, releases)
		}
	})
}

func testRecoveryMainNoRecoveryConfig(t *testing.T) {
	environment := validCompositionEnvironment()
	lookup := func(key string) string { return environment[key] }
	plan, err := planRecoveryStartup(lookup, nil)
	if err != nil {
		t.Fatalf("planRecoveryStartup(no recovery configuration) error = %v", err)
	}
	if plan.present || plan.verify || plan.maintenance || plan.action != "" || recoveryRootConfigurationPresent(lookup) {
		t.Fatalf("plan present=%v action=%q verify=%v maintenance=%v, want an absent recovery plan", plan.present, plan.action, plan.verify, plan.maintenance)
	}
	var handle recovery.Runtime
	if plan.present {
		handle = newS419RecoveryRuntime("closed")
	}
	fixture := newS419ProductionDispatcher(t, func() recovery.Runtime { return handle }, nil)

	s419AssertProbe(t, fixture, http.MethodGet, "/livez", http.StatusOK, "alive")
	s419AssertProbe(t, fixture, http.MethodGet, "/readyz", http.StatusServiceUnavailable, "unavailable")

	response := fixture.serve(t, fixture.request("/api/v1/admin/articles"), http.StatusOK)
	if bytes.Contains(response.Body.Bytes(), []byte(platform.CodePlatformRecoveryInProgress)) {
		t.Fatal("ordinary route was guarded without a recovery runtime")
	}

	unknown := httptest.NewRecorder()
	fixture.handler.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/not-a-route", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown path status=%d, want 404", unknown.Code)
	}

	wrong := httptest.NewRecorder()
	fixture.handler.ServeHTTP(wrong, httptest.NewRequest(http.MethodPost, "/api/v1/admin/articles", nil))
	if wrong.Code != http.StatusMethodNotAllowed || wrong.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("wrong method status=%d Allow=%q, want 405 GET", wrong.Code, wrong.Header().Get("Allow"))
	}
	t.Logf("planRecoveryStartup present=%v action=%q verify=%v maintenance=%v handle_nil=%v", plan.present, plan.action, plan.verify, plan.maintenance, handle == nil)
}

func newS419ProductionDispatcher(
	t *testing.T,
	provider func() recovery.Runtime,
	configure func(*articleProductionTestSeam),
) *adminRootFixture {
	t.Helper()
	fixture := &adminRootFixture{}
	fixture.resolver = &articleAuthorRootResolver{
		events: &fixture.events,
		principal: session.Principal{
			SessionID: "session-admin",
			UserID:    "30000000-0000-4000-8000-000000000003",
			Role:      "admin",
			Status:    "active",
		},
	}
	seam := &articleProductionTestSeam{
		principalResolver: fixture.resolver,
		shared: func(value articleSharedDependency) {
			fixture.observations = append(fixture.observations, value)
		},
		trace: func(request *http.Request) {
			fixture.traceID, _ = platform.TraceIDFromContext(request.Context())
			fixture.events = append(fixture.events, "trace")
		},
	}
	fixture.installReaders(t, seam)
	if configure != nil {
		configure(seam)
	}
	config := mustLoadStartupConfig(t, validCompositionEnvironment())
	prepared, err := prepareProductionLoginDependencies(
		config,
		bytes.NewReader(bytes.Repeat([]byte{0x5a}, 128)),
	)
	clearStartupSecrets(&config)
	if err != nil {
		t.Fatalf("prepareProductionLoginDependencies() error = %v", err)
	}
	t.Cleanup(prepared.cleanup)
	composition := productionBusinessComposition(t, prepared)
	composition.articleJournal = mainArticleJournalStub{}
	composition.articleDeletion = &articleDeletionStartupConfig{
		workerID: "article-worker-test",
		lease:    30 * time.Second,
		worker: article.DeletionWorkerConfig{
			PollInterval:      time.Second,
			BatchSize:         10,
			OperationBudget:   10 * time.Second,
			BackoffInitial:    250 * time.Millisecond,
			BackoffMax:        time.Second,
			MaxBatchesPerWake: 3,
			ShutdownBudget:    time.Second,
		},
	}
	composition.runtime = &businessRuntime{}
	composition.articleTestSeam = seam
	composition.recoveryHandle = provider
	handler, err := newProductionBusinessHandler(new(pgxpool.Pool), composition)
	if err != nil {
		t.Fatalf("newProductionBusinessHandler() error = %v", err)
	}
	fixture.handler = handler
	return fixture
}

func s419AssertProbe(t *testing.T, fixture *adminRootFixture, method, path string, status int, body string) {
	t.Helper()
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != status {
		t.Fatalf("%s %s status=%d, want %d", method, path, response.Code, status)
	}
	var probe struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &probe); err != nil {
		t.Fatalf("%s %s decode: %v", method, path, err)
	}
	if probe.Status != body {
		t.Fatalf("%s %s status body=%q, want %q", method, path, probe.Status, body)
	}
	s419AssertNoRecoveryLeak(t, response.Body.Bytes())
}

func s419AssertNoRecoveryLeak(t *testing.T, body []byte) {
	t.Helper()
	lowered := bytes.ToLower(body)
	for _, forbidden := range []string{"run", "secret", "backup", "account", "ref"} {
		if bytes.Contains(lowered, []byte(forbidden)) {
			t.Fatalf("response leaked internal marker %q", forbidden)
		}
	}
}

func s419AssertProbeMethod(t *testing.T, fixture *adminRootFixture, method, path, allow string) {
	t.Helper()
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("%s %s status=%d, want 405", method, path, response.Code)
	}
	if got := response.Header().Get("Allow"); got != allow {
		t.Fatalf("%s %s Allow=%q, want %q", method, path, got, allow)
	}
}

func s419ServePanicking(t *testing.T, handler http.Handler, request *http.Request) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("handler did not panic")
		}
	}()
	handler.ServeHTTP(httptest.NewRecorder(), request)
}

func s419AssertOnlyTrace(t *testing.T, events []string) {
	t.Helper()
	for _, event := range events {
		if event != "trace" {
			t.Fatalf("guard boundary reached downstream event %q", event)
		}
	}
}

func s419AssertEvents(t *testing.T, events []string, want ...string) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("events=%v, want=%v", events, want)
	}
	for index := range want {
		if events[index] != want[index] {
			t.Fatalf("events=%v, want=%v", events, want)
		}
	}
}

type s419RecoveryRuntime struct {
	*recoveryTestRuntime
	mu          sync.Mutex
	observation recovery.AdmissionObservation
	acquireErr  error
	acquires    int
	verifies    int
	releases    int
}

var s419ErrAdmissionUnknown = errors.New("s419 admission unknown")

func newS419RecoveryRuntime(state string) *s419RecoveryRuntime {
	runtime := &s419RecoveryRuntime{recoveryTestRuntime: &recoveryTestRuntime{}}
	runtime.setState(state)
	return runtime
}

func (r *s419RecoveryRuntime) handle() recovery.Runtime {
	return r
}

func (r *s419RecoveryRuntime) setState(state string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observation = recovery.AdmissionObservation{State: state}
	switch state {
	case "public_ready":
		r.acquireErr = nil
	case "unknown":
		r.acquireErr = s419ErrAdmissionUnknown
	default:
		r.acquireErr = recovery.ErrAdmissionClosed
	}
}

func (r *s419RecoveryRuntime) acquireError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acquireErr
}

func (r *s419RecoveryRuntime) Verify(ctx context.Context) (recovery.AdmissionObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.verifies++
	if err := ctx.Err(); err != nil {
		return recovery.AdmissionObservation{}, err
	}
	return r.observation, nil
}

func (r *s419RecoveryRuntime) Acquire(ctx context.Context, _ recovery.WorkKind) (recovery.WorkLease, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.acquires++
	if r.acquireErr != nil {
		return nil, r.acquireErr
	}
	return &s419WorkLease{ctx: ctx, release: func() {
		r.mu.Lock()
		r.releases++
		r.mu.Unlock()
	}}, nil
}

func (r *s419RecoveryRuntime) snapshot() (acquires, verifies, releases int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acquires, r.verifies, r.releases
}

type s419WorkLease struct {
	ctx     context.Context
	release func()
}

func (lease *s419WorkLease) Context() context.Context { return lease.ctx }

func (lease *s419WorkLease) Release() { lease.release() }

// s419ObservedRuntime is a thin observation wrapper around the real
// recovery.NewRuntime instance. Every method delegates to the embedded runtime;
// only Acquire and the returned lease's Release are counted so the test can
// prove dispatcher admission without inventing a fake state machine.
type s419ObservedRuntime struct {
	recovery.Runtime
	mu       sync.Mutex
	acquires int
	releases int
}

func (r *s419ObservedRuntime) Acquire(ctx context.Context, kind recovery.WorkKind) (recovery.WorkLease, error) {
	r.mu.Lock()
	r.acquires++
	r.mu.Unlock()
	lease, err := r.Runtime.Acquire(ctx, kind)
	if err != nil || lease == nil {
		return nil, err
	}
	return &s419ObservedLease{lease: lease, released: r.countRelease}, nil
}

func (r *s419ObservedRuntime) countRelease() {
	r.mu.Lock()
	r.releases++
	r.mu.Unlock()
}

func (r *s419ObservedRuntime) counts() (acquires, releases int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acquires, r.releases
}

type s419ObservedLease struct {
	lease    recovery.WorkLease
	released func()
}

func (l *s419ObservedLease) Context() context.Context { return l.lease.Context() }

func (l *s419ObservedLease) Release() {
	l.lease.Release()
	l.released()
}

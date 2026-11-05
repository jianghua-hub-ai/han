package platform

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/recovery"
	"han/internal/session"
)

func TestRecoveryHTTPBarrier(t *testing.T) {
	t.Run("ClosedUnknownBeforeSession", testRecoveryClosed)
	t.Run("ResponseLeaseAndCancellation", testRecoveryLease)
	t.Run("ProbeDoesNotAuthorize", testRecoveryProbes)
	t.Run("ExactRouterBoundaryContract", testRecoveryExactRouter)
}

// These ports are synthetic component inputs, never production fallbacks.
type recoveryHTTPRuntime struct {
	recovery.Runtime
	acquire func(context.Context) (recovery.WorkLease, error)
	verify  func(context.Context) (recovery.AdmissionObservation, error)
	claims  atomic.Int32
	checks  atomic.Int32
}

func (r *recoveryHTTPRuntime) Acquire(ctx context.Context, kind recovery.WorkKind) (recovery.WorkLease, error) {
	r.claims.Add(1)
	if kind != recovery.WorkRequest {
		return nil, errors.New("unexpected work kind")
	}
	return r.acquire(ctx)
}

func (r *recoveryHTTPRuntime) Verify(ctx context.Context) (recovery.AdmissionObservation, error) {
	r.checks.Add(1)
	return r.verify(ctx)
}

type recoveryHTTPLease struct {
	ctx      context.Context
	released atomic.Int32
}

func (l *recoveryHTTPLease) Context() context.Context { return l.ctx }
func (l *recoveryHTTPLease) Release()                 { l.released.Add(1) }

type recoveryHTTPNilContext struct{ context.Context }

type recoveryHTTPResolver struct {
	calls atomic.Int32
	visit func(context.Context)
}

func (r *recoveryHTTPResolver) ResolvePrincipal(ctx context.Context, _ session.ResolveInput) (session.Principal, error) {
	r.calls.Add(1)
	if r.visit != nil {
		r.visit(ctx)
	}
	return session.Principal{UserID: "synthetic-admin", Role: "admin", Status: "active"}, ctx.Err()
}

func recoveryHTTPGuard(t *testing.T, runtime recovery.Runtime) RecoveryGuard {
	t.Helper()
	guard, err := NewRecoveryGuard(runtime)
	if err != nil {
		t.Fatal(err)
	}
	return guard
}

func recoveryHTTPProtected(t *testing.T, resolver *recoveryHTTPResolver, next http.Handler) http.Handler {
	t.Helper()
	authorizer, err := NewHTTPAuthorizer(resolver, func() time.Time {
		return time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	return authorizer.Protect(session.AccessGeneral, next)
}

func recoveryHTTPRequest(path string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-barrier-session"})
	return r
}

func requireRecoveryClosed(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var envelope Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, response.Body)
	}
	definition, _ := DefaultRegistry.Definition(CodePlatformRecoveryInProgress)
	if response.Code != http.StatusServiceUnavailable || envelope.Code != definition.Code || envelope.Message != definition.Message {
		t.Fatalf("response = %d %+v", response.Code, envelope)
	}
	if envelope.TraceID == "" || envelope.TraceID != response.Header().Get(TraceHeader) {
		t.Fatal("missing or mismatched response trace")
	}
	if len(envelope.FieldErrors) != 0 || strings.Contains(response.Body.String(), "private-canary") {
		t.Fatal("unexpected public error material")
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing JSON security headers")
	}
}

func testRecoveryClosed(t *testing.T) {
	t.Run("MissingRuntime", testRecoveryMissingRuntime)
	cases := []string{"ZeroGuard", "Unknown", "NilLease", "TypedNilLease", "NilContext", "TypedNilContext", "CanceledLease", "CanceledRequest"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			for _, route := range []struct{ name, path string }{{"List", "/api/v1/admin/articles"}, {"Detail", "/api/v1/admin/articles/item"}} {
				t.Run(route.name, func(t *testing.T) { testRecoveryClosedCase(t, name, route.path) })
			}
		})
	}
	t.Run("RealRuntimeDefaultsClosed", func(t *testing.T) {
		runtime, _ := newRecoveryHTTPRealRuntime(t, false)
		testRecoveryDeniedChain(t, recoveryHTTPGuard(t, runtime), recoveryHTTPRequest("/api/v1/admin/articles"))
	})
	t.Run("MissingHandler", func(t *testing.T) {
		var next http.HandlerFunc
		for _, handler := range []http.Handler{nil, next} {
			recorder := httptest.NewRecorder()
			TraceMiddleware(RecoveryGuard{}.Protect(handler)).ServeHTTP(recorder, recoveryHTTPRequest("/"))
			requireRecoveryClosed(t, recorder)
		}
	})
	t.Run("MissingRequest", func(t *testing.T) {
		port := &recoveryHTTPRuntime{}
		response := httptest.NewRecorder()
		recoveryHTTPGuard(t, port).Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("missing request reached handler")
		})).ServeHTTP(response, nil)
		requireRecoveryClosed(t, response)
		if port.claims.Load() != 0 {
			t.Fatal("missing request reached runtime")
		}
	})
}

func testRecoveryMissingRuntime(t *testing.T) {
	var typed *recoveryHTTPRuntime
	for _, runtime := range []recovery.Runtime{nil, typed} {
		guard, err := NewRecoveryGuard(runtime)
		if err == nil {
			t.Error("constructor accepted missing runtime")
		}
		testRecoveryDeniedChain(t, guard, recoveryHTTPRequest("/api/v1/admin/articles"))
	}
}

func testRecoveryClosedCase(t *testing.T, name, path string) {
	t.Helper()
	request := recoveryHTTPRequest(path)
	lease := &recoveryHTTPLease{ctx: request.Context()}
	port := &recoveryHTTPRuntime{acquire: func(context.Context) (recovery.WorkLease, error) {
		switch name {
		case "Unknown":
			return nil, errors.New("private-canary unknown admission")
		case "NilLease":
			return nil, nil
		case "TypedNilLease":
			var missing *recoveryHTTPLease
			return missing, nil
		}
		return lease, nil
	}}
	configureRecoveryCanceledCase(name, request, lease)
	guard := recoveryHTTPGuard(t, port)
	if name == "ZeroGuard" {
		guard = RecoveryGuard{}
	}
	testRecoveryDeniedChain(t, guard, request)
	if name == "NilContext" || name == "TypedNilContext" || name == "CanceledLease" || name == "CanceledRequest" {
		// An already canceled request may be refused before acquiring anything.
		if port.claims.Load() != 0 && lease.released.Load() != 1 {
			t.Fatal("acquired rejected lease not released exactly once")
		}
	}
}

func configureRecoveryCanceledCase(name string, request *http.Request, lease *recoveryHTTPLease) {
	switch name {
	case "NilContext":
		lease.ctx = nil
	case "TypedNilContext":
		var missing *recoveryHTTPNilContext
		lease.ctx = missing
	case "CanceledLease", "CanceledRequest":
		ctx, cancel := context.WithCancel(request.Context())
		cancel()
		lease.ctx = ctx
		if name == "CanceledRequest" {
			*request = *request.WithContext(ctx)
		}
	}
}

func testRecoveryDeniedChain(t *testing.T, guard RecoveryGuard, request *http.Request) {
	t.Helper()
	resolver := &recoveryHTTPResolver{}
	var service, repository, author atomic.Int32
	next := recoveryHTTPProtected(t, resolver, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		service.Add(1)
		repository.Add(1)
		author.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	TraceMiddleware(guard.Protect(next)).ServeHTTP(recorder, request)
	requireRecoveryClosed(t, recorder)
	if resolver.calls.Load()+service.Load()+repository.Load()+author.Load() != 0 {
		t.Fatal("rejected request reached Session or business consumers")
	}
}

func testRecoveryLease(t *testing.T) {
	t.Run("OriginalContextAndWriter", testRecoveryOriginalContext)
	for _, stage := range []string{"Session", "Handler", "WriteHeader", "Write", "AfterWrite"} {
		t.Run("HeldThrough"+stage, func(t *testing.T) { testRecoveryBlockedLifetime(t, stage, false) })
	}
	t.Run("RuntimeFaultWaitsForWrite", func(t *testing.T) { testRecoveryBlockedLifetime(t, "Write", true) })
	t.Run("ParentCancellationWaitsForHandler", testRecoveryParentCancellation)
	t.Run("DispatchCanceledLease", testRecoveryDispatchCancellation)
	t.Run("WriteFailure", testRecoveryWriteFailure)
	t.Run("PanicUnwindsLease", testRecoveryPanic)
	t.Run("ErrorWithLeaseStillReleases", func(t *testing.T) {
		lease := &recoveryHTTPLease{}
		port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) {
			lease.ctx = ctx
			return lease, recovery.ErrAdmissionClosed
		}}
		testRecoveryDeniedChain(t, recoveryHTTPGuard(t, port), recoveryHTTPRequest("/"))
		if lease.released.Load() != 1 {
			t.Fatal("error-returned lease leaked")
		}
	})
}

type recoveryHTTPValueKey struct{}

func testRecoveryOriginalContext(t *testing.T) {
	request := recoveryHTTPRequest("/api/v1/admin/articles")
	ctx, cancel := context.WithDeadline(context.WithValue(request.Context(), recoveryHTTPValueKey{}, "retained"), time.Now().Add(time.Minute))
	defer cancel()
	request = request.WithContext(ctx)
	lease := &recoveryHTTPLease{}
	port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) { lease.ctx = ctx; return lease, nil }}
	writer := &recoveryHTTPCapableWriter{ResponseRecorder: httptest.NewRecorder()}
	var visits int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visits++
		if w != writer || r.Context() != lease.ctx || r.Context().Value(recoveryHTTPValueKey{}) != "retained" {
			t.Error("writer identity or lease context lost")
		}
		deadline, _ := r.Context().Deadline()
		want, _ := ctx.Deadline()
		trace, _ := TraceIDFromContext(r.Context())
		if deadline != want || trace == "" || trace != w.Header().Get(TraceHeader) || lease.released.Load() != 0 {
			t.Error("deadline/trace/lifetime mismatch")
		}
		if _, ok := w.(http.Flusher); !ok {
			t.Error("Flusher lost")
		}
		if _, ok := w.(http.Pusher); !ok {
			t.Error("Pusher lost")
		}
		if _, ok := w.(io.ReaderFrom); !ok {
			t.Error("ReaderFrom lost")
		}
		if _, ok := w.(http.Hijacker); !ok {
			t.Error("Hijacker lost")
		}
		_ = WriteJSON(w, http.StatusOK, struct{ OK bool }{true})
	})
	TraceMiddleware(recoveryHTTPGuard(t, port).Protect(next)).ServeHTTP(writer, request)
	if visits != 1 || lease.released.Load() != 1 || port.claims.Load() != 1 || port.checks.Load() != 0 {
		t.Fatal("ordinary request did not acquire/release exactly once")
	}
}

type recoveryHTTPCapableWriter struct{ *httptest.ResponseRecorder }

func (*recoveryHTTPCapableWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, http.ErrNotSupported
}

func (*recoveryHTTPCapableWriter) Push(string, *http.PushOptions) error { return http.ErrNotSupported }
func (w *recoveryHTTPCapableWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(w.ResponseRecorder, r)
}

type recoveryHTTPTrackedRuntime struct {
	recovery.Runtime
	lease chan *recoveryHTTPTrackedLease
}

func (r *recoveryHTTPTrackedRuntime) Acquire(ctx context.Context, kind recovery.WorkKind) (recovery.WorkLease, error) {
	l, err := r.Runtime.Acquire(ctx, kind)
	if err != nil {
		return nil, err
	}
	tracked := &recoveryHTTPTrackedLease{WorkLease: l}
	r.lease <- tracked
	return tracked, nil
}

type recoveryHTTPTrackedLease struct {
	recovery.WorkLease
	released atomic.Int32
}

func (l *recoveryHTTPTrackedLease) Release() { l.released.Add(1); l.WorkLease.Release() }

type recoveryHTTPBlockingWriter struct {
	*httptest.ResponseRecorder
	stage string
	block func()
}

func (w *recoveryHTTPBlockingWriter) WriteHeader(status int) {
	if w.stage == "WriteHeader" {
		w.block()
	}
	w.ResponseRecorder.WriteHeader(status)
}

func (w *recoveryHTTPBlockingWriter) Write(body []byte) (int, error) {
	if w.stage == "Write" {
		w.block()
	}
	return w.ResponseRecorder.Write(body)
}

func testRecoveryBlockedLifetime(t *testing.T, stage string, fault bool) {
	real, facility := newRecoveryHTTPRealRuntime(t, true)
	tracked := &recoveryHTTPTrackedRuntime{Runtime: real, lease: make(chan *recoveryHTTPTrackedLease, 1)}
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	block := func() { close(entered); <-resume }
	t.Cleanup(func() { unblock.Do(func() { close(resume) }); recoveryHTTPWait(t, done) })
	resolver := &recoveryHTTPResolver{}
	if stage == "Session" {
		resolver.visit = func(context.Context) { block() }
	}
	next := recoveryHTTPProtected(t, resolver, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if stage == "Handler" {
			block()
		}
		_ = WriteJSON(w, http.StatusOK, struct{ Status string }{"complete"})
		if stage == "AfterWrite" {
			block()
		}
	}))
	w := &recoveryHTTPBlockingWriter{ResponseRecorder: httptest.NewRecorder(), stage: stage, block: block}
	go func() {
		defer close(done)
		TraceMiddleware(recoveryHTTPGuard(t, tracked).Protect(next)).ServeHTTP(w, recoveryHTTPRequest("/api/v1/admin/articles"))
	}()
	select {
	case <-entered:
	case <-done:
		t.Fatal("handler returned before entering the lifetime barrier")
	}
	var lease *recoveryHTTPTrackedLease
	select {
	case lease = <-tracked.lease:
	default:
		t.Fatal("handler ran without acquiring a lease")
	}
	if lease.released.Load() != 0 {
		t.Fatal("lease released before response/handler ended")
	}
	if fault {
		testRecoveryFaultWhileBlocked(t, real, facility, lease)
	}
	unblock.Do(func() { close(resume) })
	recoveryHTTPWait(t, done)
	if lease.released.Load() != 1 {
		t.Fatal("lease not released exactly once on return")
	}
}

func testRecoveryFaultWhileBlocked(t *testing.T, runtime recovery.Runtime, facility *recoveryHTTPFacility, lease *recoveryHTTPTrackedLease) {
	t.Helper()
	facility.lost.Store(true)
	if next, err := runtime.Acquire(context.Background(), recovery.WorkRequest); err == nil {
		next.Release()
		t.Fatal("fault still authorized new work")
	}
	if lease.Context().Err() == nil {
		t.Fatal("fault did not revoke the in-flight lease context")
	}
	if _, err := runtime.ReadPermitBinding(lease.Context()); err == nil {
		t.Fatal("old permit remained valid")
	}
	if lease.released.Load() != 0 || facility.closed.Load() != 0 {
		t.Fatal("fault pretended response had joined")
	}
}

func recoveryHTTPWait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("component call did not join")
	}
}

func testRecoveryParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lease := &recoveryHTTPLease{}
	port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) { lease.ctx = ctx; return lease, nil }}
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var unblock sync.Once
	t.Cleanup(func() { unblock.Do(func() { close(resume) }); recoveryHTTPWait(t, done) })
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); <-resume })
	guard := recoveryHTTPGuard(t, port)
	go func() {
		defer close(done)
		guard.Protect(next).ServeHTTP(httptest.NewRecorder(), recoveryHTTPRequest("/").WithContext(ctx))
	}()
	select {
	case <-entered:
	case <-done:
		t.Fatal("request did not enter handler")
	}
	cancel()
	if lease.ctx == nil || lease.ctx.Err() == nil || lease.released.Load() != 0 {
		t.Fatal("cancel lost context or released before actual handler exit")
	}
	unblock.Do(func() { close(resume) })
	recoveryHTTPWait(t, done)
	if lease.released.Load() != 1 {
		t.Fatal("canceled handler did not release")
	}
}

func testRecoveryDispatchCancellation(t *testing.T) {
	lease := &recoveryHTTPLease{}
	port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) {
		var cancel context.CancelFunc
		lease.ctx, cancel = context.WithCancel(ctx)
		cancel()
		return lease, nil
	}}
	testRecoveryDeniedChain(t, recoveryHTTPGuard(t, port), recoveryHTTPRequest("/"))
	if lease.released.Load() != 1 {
		t.Fatal("canceled acquired lease leaked")
	}
}

type recoveryHTTPFailedWriter struct{ *httptest.ResponseRecorder }

func (*recoveryHTTPFailedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func testRecoveryWriteFailure(t *testing.T) {
	lease := &recoveryHTTPLease{}
	port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) { lease.ctx = ctx; return lease, nil }}
	var writeErr error
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeErr = WriteJSON(w, http.StatusOK, struct{ OK bool }{true})
		if lease.released.Load() != 0 {
			t.Error("released during failing Write")
		}
	})
	recoveryHTTPGuard(t, port).Protect(next).ServeHTTP(&recoveryHTTPFailedWriter{httptest.NewRecorder()}, recoveryHTTPRequest("/"))
	if !errors.Is(writeErr, io.ErrClosedPipe) || lease.released.Load() != 1 {
		t.Fatal("write error/lifetime contract lost")
	}
}

func testRecoveryPanic(t *testing.T) {
	lease := &recoveryHTTPLease{}
	port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) { lease.ctx = ctx; return lease, nil }}
	defer func() {
		if recover() != "synthetic-handler-panic" || lease.released.Load() != 1 {
			t.Error("panic swallowed or lease leaked")
		}
	}()
	recoveryHTTPGuard(t, port).Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("synthetic-handler-panic") })).ServeHTTP(httptest.NewRecorder(), recoveryHTTPRequest("/"))
}

func testRecoveryProbes(t *testing.T) {
	for _, state := range []string{"closed", "internal_ready", "public_ready", "unknown", "faulted", "draining", "stopped", ""} {
		t.Run("State_"+state, func(t *testing.T) { testRecoveryProbeState(t, state, nil, false) })
	}
	t.Run("VerifyFailure", func(t *testing.T) { testRecoveryProbeState(t, "public_ready", errors.New("private-canary"), false) })
	t.Run("CanceledObservation", func(t *testing.T) { testRecoveryProbeState(t, "public_ready", nil, true) })
	t.Run("ZeroGuard", func(t *testing.T) {
		requireRecoveryProbe(t, RecoveryGuard{}, false, http.StatusServiceUnavailable, "unavailable")
	})
	t.Run("RealRuntimeObservations", testRecoveryRealProbes)
	t.Run("CanceledBeforeObservation", func(t *testing.T) {
		port := &recoveryHTTPRuntime{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		response := httptest.NewRecorder()
		TraceMiddleware(http.HandlerFunc(recoveryHTTPGuard(t, port).Readiness)).ServeHTTP(response, recoveryHTTPRequest("/readyz").WithContext(ctx))
		checkRecoveryProbeResponse(t, response, http.StatusServiceUnavailable, "unavailable")
		if port.claims.Load()+port.checks.Load() != 0 {
			t.Fatal("canceled probe reached runtime")
		}
	})
}

func testRecoveryProbeState(t *testing.T, state string, err error, cancelObservation bool) {
	request := recoveryHTTPRequest("/readyz")
	ctx, cancel := context.WithCancel(request.Context())
	defer cancel()
	port := &recoveryHTTPRuntime{
		acquire: func(context.Context) (recovery.WorkLease, error) { return nil, recovery.ErrAdmissionClosed },
		verify: func(context.Context) (recovery.AdmissionObservation, error) {
			if cancelObservation {
				cancel()
			}
			return recovery.AdmissionObservation{State: state, Reason: "private-canary"}, err
		},
	}
	guard := recoveryHTTPGuard(t, port)
	requireRecoveryProbe(t, guard, true, http.StatusOK, "alive")
	if port.claims.Load()+port.checks.Load() != 0 {
		t.Fatal("liveness authorized or observed runtime")
	}
	wantStatus, want := http.StatusServiceUnavailable, "unavailable"
	if state == "public_ready" && err == nil && !cancelObservation {
		wantStatus, want = http.StatusOK, "ready"
	}
	response := httptest.NewRecorder()
	TraceMiddleware(http.HandlerFunc(guard.Readiness)).ServeHTTP(response, request.WithContext(ctx))
	checkRecoveryProbeResponse(t, response, wantStatus, want)
	if port.checks.Load() != 1 || port.claims.Load() != 0 {
		t.Fatal("readiness must observe once without acquiring")
	}
	testRecoveryDeniedChain(t, guard, recoveryHTTPRequest("/api/v1/admin/articles"))
}

func requireRecoveryProbe(t *testing.T, guard RecoveryGuard, live bool, status int, want string) {
	t.Helper()
	handler := guard.Readiness
	if live {
		handler = guard.Liveness
	}
	response := httptest.NewRecorder()
	TraceMiddleware(http.HandlerFunc(handler)).ServeHTTP(response, recoveryHTTPRequest("/readyz"))
	checkRecoveryProbeResponse(t, response, status, want)
}

func checkRecoveryProbeResponse(t *testing.T, response *httptest.ResponseRecorder, status int, want string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != status || len(body) != 1 || body["status"] != want {
		t.Fatalf("probe = %d %v", response.Code, body)
	}
	if strings.Contains(response.Body.String(), "private-canary") || response.Header().Get(TraceHeader) == "" {
		t.Fatal("probe trace/privacy contract lost")
	}
}

func testRecoveryRealProbes(t *testing.T) {
	runtime, _ := newRecoveryHTTPRealRuntime(t, false)
	guard := recoveryHTTPGuard(t, runtime)
	requireRecoveryProbe(t, guard, true, http.StatusOK, "alive")
	requireRecoveryProbe(t, guard, false, http.StatusServiceUnavailable, "unavailable")
	if _, err := runtime.Acquire(context.Background(), recovery.WorkRequest); err == nil {
		t.Fatal("Verify opened a default closed runtime")
	}
	if err := runtime.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	requireRecoveryProbe(t, guard, false, http.StatusOK, "ready")
	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	requireRecoveryProbe(t, guard, false, http.StatusServiceUnavailable, "unavailable")
}

// This exact harness verifies component ordering only, NOT cmd root or Case18.
func recoveryHTTPExactComponent(guard RecoveryGuard, selected http.Handler) http.Handler {
	return TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allow := ""
		switch r.URL.Path {
		case "/api/v1/admin/articles":
			allow = http.MethodGet
		case "/api/v1/admin/articles/item":
			allow = http.MethodGet + ", " + http.MethodDelete
		case "/livez", "/readyz":
			allow = http.MethodGet
		default:
			_ = WriteError(w, r, NewPublicError(CodeRequestRouteNotFound, nil))
			return
		}
		if r.Method != http.MethodGet && !(r.Method == http.MethodDelete && r.URL.Path == "/api/v1/admin/articles/item") {
			w.Header().Set("Allow", allow)
			_ = WriteError(w, r, NewPublicError(CodeRequestMethodNotAllowed, nil))
			return
		}
		switch r.URL.Path {
		case "/livez":
			guard.Liveness(w, r)
		case "/readyz":
			guard.Readiness(w, r)
		default:
			guard.Protect(selected).ServeHTTP(w, r)
		}
	}))
}

func testRecoveryExactRouter(t *testing.T) {
	cases := []struct {
		name, method, path, allow string
		status                    int
	}{
		{"Unknown", "GET", "/unknown", "", 404},
		{"TrailingSegment", "GET", "/api/v1/admin/articles/item/extra", "", 404},
		{"TrailingSlash", "GET", "/api/v1/admin/articles/", "", 404},
		{"ListMethod", "POST", "/api/v1/admin/articles", "GET", 405},
		{"DetailMethod", "PUT", "/api/v1/admin/articles/item", "GET, DELETE", 405},
		{"LiveMethod", "HEAD", "/livez", "GET", 405},
		{"ReadyMethod", "POST", "/readyz", "GET", 405},
		{"ListClosed", "GET", "/api/v1/admin/articles", "", 503},
		{"DetailClosed", "GET", "/api/v1/admin/articles/item", "", 503},
		{"DeleteClosed", "DELETE", "/api/v1/admin/articles/item", "", 503},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { testRecoveryExactCase(t, tc.method, tc.path, tc.allow, tc.status) })
	}
	t.Run("OpenSessionAndBusinessOrder", testRecoveryOpenOrder)
}

func testRecoveryExactCase(t *testing.T, method, path, allow string, status int) {
	port := &recoveryHTTPRuntime{acquire: func(context.Context) (recovery.WorkLease, error) { return nil, recovery.ErrAdmissionClosed }}
	resolver := &recoveryHTTPResolver{}
	var business atomic.Int32
	selected := recoveryHTTPProtected(t, resolver, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { business.Add(1) }))
	response := httptest.NewRecorder()
	request := recoveryHTTPRequest(path)
	request.Method = method
	recoveryHTTPExactComponent(recoveryHTTPGuard(t, port), selected).ServeHTTP(response, request)
	if response.Code != status || response.Header().Get("Allow") != allow {
		t.Fatalf("route = %d Allow=%q", response.Code, response.Header().Get("Allow"))
	}
	if resolver.calls.Load()+business.Load() != 0 {
		t.Fatal("routing or closed guard entered Session/business")
	}
	wantClaims := int32(0)
	if status == 503 {
		wantClaims = 1
		requireRecoveryClosed(t, response)
	}
	if port.claims.Load() != wantClaims || port.checks.Load() != 0 {
		t.Fatal("exact routing did not precede guard")
	}
	if status != 503 {
		var envelope Envelope
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		want := CodeRequestRouteNotFound
		if status == 405 {
			want = CodeRequestMethodNotAllowed
		}
		if envelope.Code != want || envelope.TraceID != response.Header().Get(TraceHeader) || envelope.TraceID == "" {
			t.Fatal("route envelope/trace changed")
		}
	}
}

func testRecoveryOpenOrder(t *testing.T) {
	var events []string
	lease := &recoveryHTTPLease{}
	port := &recoveryHTTPRuntime{acquire: func(ctx context.Context) (recovery.WorkLease, error) {
		events = append(events, "guard")
		lease.ctx = ctx
		return lease, nil
	}}
	resolver := &recoveryHTTPResolver{visit: func(ctx context.Context) {
		events = append(events, "session")
		if trace, ok := TraceIDFromContext(ctx); !ok || trace == "" {
			t.Error("Session lost Trace")
		}
	}}
	selected := recoveryHTTPProtected(t, resolver, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		events = append(events, "business")
		if _, ok := PrincipalFromContext(r.Context()); !ok {
			t.Error("business lost Principal")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	recoveryHTTPExactComponent(recoveryHTTPGuard(t, port), selected).ServeHTTP(response, recoveryHTTPRequest("/api/v1/admin/articles"))
	if strings.Join(events, ",") != "guard,session,business" || response.Code != http.StatusNoContent || lease.released.Load() != 1 {
		t.Fatalf("open order/lifetime = %v, release=%d", events, lease.released.Load())
	}
}

// Explicit synthetic genesis ports exercise the REAL Runtime admission/leases.
// Embedded unused ports fail loudly if ordinary admission unexpectedly uses them.
type recoveryHTTPProof struct {
	recovery.ContextRecoveryJournal
	recovery.PublishedClosureReader
	snapshot recovery.AdmissionSnapshot
	genesis  recovery.GenesisProof
}

func (p *recoveryHTTPProof) ReadAdmissionSnapshot(context.Context) (recovery.AdmissionSnapshot, error) {
	return p.snapshot, nil
}
func (p *recoveryHTTPProof) ReadAdmissionRevision(context.Context) (recovery.StoreRevision, error) {
	return p.snapshot.Revision, nil
}
func (p *recoveryHTTPProof) VerifyGenesis(context.Context, recovery.InstallationBinding) (recovery.GenesisProof, error) {
	return p.genesis, nil
}

func (p *recoveryHTTPProof) ReadVerifiedSnapshot(_ context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
	initial := p.snapshot.Installation.RecoveryInitial
	if namespace == recovery.ArticleDeletionNamespaceV1 {
		initial = p.snapshot.Installation.ArticleInitial
	}
	return recovery.VerifiedJournalSnapshot{JournalID: initial.JournalID, Namespace: namespace, Epoch: initial.Epoch, Revision: namespace + "-synthetic-r1", Prefixes: []recovery.JournalHead{{}}, Entries: []recovery.JournalEntry{}}, nil
}

type recoveryHTTPFacility struct {
	lost   atomic.Bool
	closed atomic.Int32
}

func (f *recoveryHTTPFacility) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	return f, nil
}
func (f *recoveryHTTPFacility) Close() error { f.closed.Add(1); return nil }
func (f *recoveryHTTPFacility) ReconcileAndStart(ctx context.Context, _ recovery.PhasePermit) error {
	return ctx.Err()
}
func (*recoveryHTTPFacility) StopAndJoin(context.Context) error { return nil }
func (f *recoveryHTTPFacility) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.lost.Load() {
		return recovery.ErrAdmissionClosed
	}
	return nil
}

func newRecoveryHTTPRealRuntime(t *testing.T, start bool) (recovery.Runtime, *recoveryHTTPFacility) {
	t.Helper()
	digest := recovery.SHA256Digest([]byte("synthetic-http-admission"))
	r := recovery.AnchorSnapshot{JournalID: "synthetic-recovery", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1}
	a := recovery.AnchorSnapshot{JournalID: "synthetic-article", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 2}
	i := recovery.InstallationBinding{Version: recovery.InstallationVersion, DeploymentID: "synthetic-deployment", InstallationID: "synthetic-installation", TargetDigest: digest, CommitmentRef: "synthetic-commitment", CommitmentDigest: digest, RecoveryInitial: r, ArticleInitial: a, InstalledAt: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)}
	p := &recoveryHTTPProof{snapshot: recovery.AdmissionSnapshot{Installation: i, Revision: recovery.StoreRevision{InstallationID: i.InstallationID, Counter: 1}, Runs: []recovery.AdmissionRun{}}}
	p.genesis = recovery.GenesisProof{Binding: i, RecoveryInitial: r, ArticleInitial: a, CommitmentRevision: "synthetic-commitment-r1", CommitmentDigest: digest}
	raw, err := recovery.CanonicalAdmissionSnapshot(p.snapshot)
	if err != nil {
		t.Fatal(err)
	}
	p.snapshot.Revision.Digest = recovery.SHA256Digest(raw)
	facility := &recoveryHTTPFacility{}
	runtime, err := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: p, Journal: p, ArticleSnapshots: p, Closures: p, Genesis: p, Exclusive: facility, Lifecycle: facility})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := runtime.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if start {
		if err := runtime.RunNormal(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return runtime, facility
}

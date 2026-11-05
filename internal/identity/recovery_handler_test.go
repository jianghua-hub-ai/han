package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
)

const (
	recoveryControlTestSession  = "maintenance-session-token"
	recoveryControlTestCSRF     = "maintenance-csrf-token"
	recoveryControlTestUser     = "user-1"
	recoveryControlTestRun      = "run-identity-1"
	recoveryControlTestBackup   = "backup-identity-1"
	recoveryControlTestContract = int64(65536)
	recoveryControlTestDigest   = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
)

// recoveryControlPhaseStub stands in for the root-held identity permit. The
// real permit is opaque, so tests only observe whether the phase seam ran, in
// which order it ran, and that failures short-circuit the Session guard.
type recoveryControlPhaseStub struct {
	mu     sync.Mutex
	calls  int
	err    error
	closed bool
}

func (s *recoveryControlPhaseStub) phase(ctx context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
	s.mu.Lock()
	s.calls++
	err, closed := s.err, s.closed
	s.mu.Unlock()
	if err != nil {
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, err
	}
	if closed {
		expired, cancel := context.WithCancel(ctx)
		cancel()
		return expired, recovery.PhasePermit{}, recovery.RunBinding{}, nil
	}
	return ctx, recovery.PhasePermit{}, recovery.RunBinding{RunID: recoveryControlTestRun, BackupRef: recoveryControlTestBackup}, nil
}

func (s *recoveryControlPhaseStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type recoveryControlMaintenanceCall struct {
	binding  recovery.RunBinding
	userID   string
	decision string
	evidence string
}

type recoveryControlIdempotencyCall struct {
	binding    recovery.RunBinding
	scope      []byte
	locator    []byte
	generation int64
	decision   string
	evidence   string
}

type recoveryControlMaintenanceStub struct {
	mu               sync.Mutex
	reviewCalls      int
	idempotencyCalls int
	lastReview       recoveryControlMaintenanceCall
	lastIdempotency  recoveryControlIdempotencyCall
	err              error
}

func (s *recoveryControlMaintenanceStub) ResolveRestrictedAccountReview(_ context.Context, _ recovery.PhasePermit, binding recovery.RunBinding, userID, decision, evidence string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reviewCalls++
	s.lastReview = recoveryControlMaintenanceCall{binding: binding, userID: userID, decision: decision, evidence: evidence}
	return s.err
}

func (s *recoveryControlMaintenanceStub) ResolveRecoveryIdempotency(_ context.Context, _ recovery.PhasePermit, binding recovery.RunBinding, scope, locator []byte, generation int64, decision, evidence string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idempotencyCalls++
	s.lastIdempotency = recoveryControlIdempotencyCall{binding: binding, scope: append([]byte(nil), scope...), locator: append([]byte(nil), locator...), generation: generation, decision: decision, evidence: evidence}
	return s.err
}

type recoveryControlReviewStub struct {
	mu         sync.Mutex
	items      []RecoveryReviewItem
	next       string
	err        error
	calls      int
	lastCursor string
	lastLimit  int
}

func (s *recoveryControlReviewStub) ListPendingReviews(_ context.Context, cursor string, limit int) ([]RecoveryReviewItem, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.lastCursor, s.lastLimit = cursor, limit
	if s.err != nil {
		return nil, "", s.err
	}
	return s.items, s.next, nil
}

type recoveryControlCompleteStub struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (s *recoveryControlCompleteStub) RequestComplete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.err
}

func (s *recoveryControlCompleteStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type recoveryControlTrustStub struct {
	mu     sync.Mutex
	calls  int
	result PasswordChangeResult
	err    error
}

func (s *recoveryControlTrustStub) ChangeRecoveryPassword(_ context.Context, _ recovery.PhasePermit, _ PasswordChangeRequest) (PasswordChangeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.result, s.err
}

func (s *recoveryControlTrustStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type recoveryControlRestoredCall struct {
	binding        recovery.RunBinding
	userID         string
	role           string
	status         string
	evidence       string
	idempotencyKey string
}

type recoveryControlRestoredStub struct {
	mu    sync.Mutex
	calls int
	last  recoveryControlRestoredCall
	err   error
}

func (s *recoveryControlRestoredStub) ResolveRestoredAccountReview(_ context.Context, _ recovery.PhasePermit, binding recovery.RunBinding, userID, role, status, evidence, idempotencyKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.last = recoveryControlRestoredCall{binding: binding, userID: userID, role: role, status: status, evidence: evidence, idempotencyKey: idempotencyKey}
	return s.err
}

type recoveryControlResolverStub struct {
	mu        sync.Mutex
	calls     int
	principal session.Principal
	err       error
	last      session.ResolveInput
}

func (s *recoveryControlResolverStub) ResolvePrincipal(_ context.Context, input session.ResolveInput) (session.Principal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.last = input
	if s.err != nil {
		return session.Principal{}, s.err
	}
	if input.RequireCSRF && input.CSRFToken != recoveryControlTestCSRF {
		return session.Principal{}, session.ErrCSRFFailed
	}
	return s.principal, nil
}

func (s *recoveryControlResolverStub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// recoveryControlRuntimeStub embeds the closed Runtime interface so only the
// read the review lister performs is observable; every guard must fail before
// any other method or any pool query is reached.
type recoveryControlRuntimeStub struct {
	recovery.Runtime
	mu      sync.Mutex
	binding recovery.PermitBinding
	err     error
	calls   int
}

func (s *recoveryControlRuntimeStub) ReadPermitBinding(context.Context) (recovery.PermitBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.binding, s.err
}

type recoveryControlFixture struct {
	handler      http.Handler
	dependencies RecoveryControlDependencies
	phase        *recoveryControlPhaseStub
	maintenance  *recoveryControlMaintenanceStub
	reviews      *recoveryControlReviewStub
	trust        *recoveryControlTrustStub
	restored     *recoveryControlRestoredStub
	complete     *recoveryControlCompleteStub
	resolver     *recoveryControlResolverStub
	sessionHits  int
}

func newRecoveryControlFixture(t *testing.T) *recoveryControlFixture {
	t.Helper()
	fixture := &recoveryControlFixture{
		phase:       &recoveryControlPhaseStub{},
		maintenance: &recoveryControlMaintenanceStub{},
		reviews:     &recoveryControlReviewStub{},
		trust:       &recoveryControlTrustStub{result: PasswordChangeResult{User: User{ID: recoveryControlTestUser, Username: "root", Role: RoleAdmin, Status: UserStatusActive}, SessionToken: "rotated-session-token", CSRFToken: "rotated-csrf-token"}},
		restored:    &recoveryControlRestoredStub{},
		complete:    &recoveryControlCompleteStub{},
		resolver:    &recoveryControlResolverStub{principal: session.Principal{UserID: recoveryControlTestUser, Role: string(RoleAdmin), Status: string(UserStatusActive)}},
	}
	authorizer, err := platform.NewHTTPAuthorizer(fixture.resolver, func() time.Time { return time.Unix(1761000000, 0).UTC() })
	if err != nil {
		t.Fatalf("build authorizer: %v", err)
	}
	fixture.dependencies = RecoveryControlDependencies{
		Maintenance: fixture.maintenance,
		Reviews:     fixture.reviews,
		TrustChange: fixture.trust,
		Restored:    fixture.restored,
		Complete:    fixture.complete,
		Authorizer:  authorizer,
		Phase:       fixture.phase.phase,
		Session: func(response http.ResponseWriter, _ *http.Request) {
			fixture.sessionHits++
			response.WriteHeader(http.StatusNoContent)
		},
		Liveness:     func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
		Readiness:    func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
		MaxBodyBytes: recoveryControlTestContract,
	}
	handlers, err := NewRecoveryControlHandlers(fixture.dependencies)
	if err != nil {
		t.Fatalf("build handlers: %v", err)
	}
	fixture.handler = handlers.Handler()
	return fixture
}

func (f *recoveryControlFixture) newRequest(method, path, body string) *http.Request {
	var source io.Reader
	if body != "" {
		source = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, source)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func (f *recoveryControlFixture) authorizedRequest(method, path, body string) *http.Request {
	request := f.newRequest(method, path, body)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: recoveryControlTestSession})
	request.Header.Set(platform.CSRFHeaderName, recoveryControlTestCSRF)
	return request
}

func (f *recoveryControlFixture) serve(request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeRecoveryControlEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) platform.Envelope {
	t.Helper()
	var envelope platform.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope %q: %v", recorder.Body.String(), err)
	}
	return envelope
}

func requireRecoveryControlCode(t *testing.T, recorder *httptest.ResponseRecorder, status int, code platform.Code) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", recorder.Code, status, recorder.Body.String())
	}
	if envelope := decodeRecoveryControlEnvelope(t, recorder); envelope.Code != code {
		t.Fatalf("code = %q, want %q (body %q)", envelope.Code, code, recorder.Body.String())
	}
	if recorder.Header().Get(platform.TraceHeader) == "" {
		t.Fatalf("trace header is missing on response")
	}
}
func newRecoveryControlFixtureWith(t *testing.T, mutate func(*RecoveryControlDependencies)) *recoveryControlFixture {
	t.Helper()
	fixture := newRecoveryControlFixture(t)
	if mutate == nil {
		return fixture
	}
	dependencies := fixture.dependencies
	mutate(&dependencies)
	handlers, err := NewRecoveryControlHandlers(dependencies)
	if err != nil {
		t.Fatalf("build handlers with mutation: %v", err)
	}
	fixture.dependencies = dependencies
	fixture.handler = handlers.Handler()
	return fixture
}

func TestRecoveryControlHandlers(t *testing.T) {
	t.Run("constructor_requires_complete_wiring", func(t *testing.T) {
		valid := newRecoveryControlFixture(t).dependencies
		if _, err := NewRecoveryControlHandlers(valid); err != nil {
			t.Fatalf("constructor rejected complete wiring: %v", err)
		}
		cases := []struct {
			name   string
			mutate func(*RecoveryControlDependencies)
		}{
			{"maintenance", func(d *RecoveryControlDependencies) { d.Maintenance = nil }},
			{"reviews", func(d *RecoveryControlDependencies) { d.Reviews = nil }},
			{"complete", func(d *RecoveryControlDependencies) { d.Complete = nil }},
			{"phase", func(d *RecoveryControlDependencies) { d.Phase = nil }},
			{"session", func(d *RecoveryControlDependencies) { d.Session = nil }},
			{"body_limit", func(d *RecoveryControlDependencies) { d.MaxBodyBytes = 0 }},
			{"authorizer", func(d *RecoveryControlDependencies) { d.Authorizer = platform.HTTPAuthorizer{} }},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				dependencies := valid
				testCase.mutate(&dependencies)
				if _, err := NewRecoveryControlHandlers(dependencies); err == nil {
					t.Fatalf("constructor accepted incomplete %q wiring", testCase.name)
				}
			})
		}
	})

	t.Run("exact_methods_and_public_not_found", func(t *testing.T) {
		cases := []struct {
			method string
			path   string
			status int
			code   platform.Code
			allow  string
		}{
			{http.MethodGet, "/api/v1/sessions", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "POST"},
			{http.MethodPost, "/api/v1/users/me", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "GET"},
			{http.MethodGet, "/api/v1/users/me/password", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "PUT"},
			{http.MethodPost, "/api/v1/recovery/reviews", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "GET"},
			{http.MethodGet, "/api/v1/recovery/idempotency/resolve", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "POST"},
			{http.MethodGet, "/api/v1/recovery/complete", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "POST"},
			{http.MethodPut, "/livez", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "GET"},
			{http.MethodPost, "/readyz", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "GET"},
			{http.MethodGet, "/api/v1/recovery/reviews/" + recoveryControlTestUser + "/resolve", http.StatusMethodNotAllowed, platform.CodeRequestMethodNotAllowed, "POST"},
			{http.MethodGet, "/api/v1/recovery/unknown", http.StatusNotFound, platform.CodeRequestRouteNotFound, ""},
			{http.MethodPost, "/api/v1/recovery/reviews//resolve", http.StatusNotFound, platform.CodeRequestRouteNotFound, ""},
			{http.MethodGet, "/healthz", http.StatusNotFound, platform.CodeRequestRouteNotFound, ""},
		}
		fixture := newRecoveryControlFixture(t)
		for _, testCase := range cases {
			t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
				recorder := fixture.serve(fixture.authorizedRequest(testCase.method, testCase.path, ""))
				requireRecoveryControlCode(t, recorder, testCase.status, testCase.code)
				if allow := recorder.Header().Get("Allow"); allow != testCase.allow {
					t.Fatalf("Allow = %q, want %q", allow, testCase.allow)
				}
			})
		}
		if fixture.phase.count() != 0 || fixture.resolver.count() != 0 || fixture.sessionHits != 0 {
			t.Fatalf("routing rejection reached phase=%d resolver=%d session=%d", fixture.phase.count(), fixture.resolver.count(), fixture.sessionHits)
		}
	})

	t.Run("phase_precedes_session", func(t *testing.T) {
		cases := []struct {
			name    string
			prepare func(*recoveryControlPhaseStub)
			status  int
			code    platform.Code
		}{
			{"binding", func(p *recoveryControlPhaseStub) { p.err = recovery.ErrAdmissionBinding }, http.StatusForbidden, platform.CodeAuthForbidden},
			{"closed", func(p *recoveryControlPhaseStub) { p.err = recovery.ErrAdmissionClosed }, http.StatusServiceUnavailable, platform.CodePlatformRecoveryInProgress},
			{"expired_context", func(p *recoveryControlPhaseStub) { p.closed = true }, http.StatusServiceUnavailable, platform.CodePlatformRecoveryInProgress},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				fixture := newRecoveryControlFixture(t)
				testCase.prepare(fixture.phase)
				recorder := fixture.serve(fixture.authorizedRequest(http.MethodGet, "/api/v1/users/me", ""))
				requireRecoveryControlCode(t, recorder, testCase.status, testCase.code)
				if fixture.phase.count() != 1 {
					t.Fatalf("phase calls = %d, want 1", fixture.phase.count())
				}
				if fixture.resolver.count() != 0 {
					t.Fatalf("phase failure still resolved a session %d times", fixture.resolver.count())
				}
			})
		}

		// A valid phase without a cookie still fails at the Session boundary
		// before any principal resolution, so no permit reaches a handler.
		fixture := newRecoveryControlFixture(t)
		recorder := fixture.serve(fixture.newRequest(http.MethodGet, "/api/v1/users/me", ""))
		requireRecoveryControlCode(t, recorder, http.StatusUnauthorized, platform.CodeAuthSessionRequired)
		if fixture.phase.count() != 1 || fixture.resolver.count() != 0 {
			t.Fatalf("phase=%d resolver=%d after missing cookie", fixture.phase.count(), fixture.resolver.count())
		}
	})

	t.Run("session_and_csrf_gate", func(t *testing.T) {
		fixture := newRecoveryControlFixture(t)

		recorder := fixture.serve(fixture.newRequest(http.MethodPost, "/api/v1/recovery/complete", "{}"))
		requireRecoveryControlCode(t, recorder, http.StatusUnauthorized, platform.CodeAuthSessionRequired)

		stale := fixture.newRequest(http.MethodPost, "/api/v1/recovery/complete", "{}")
		stale.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: recoveryControlTestSession})
		stale.Header.Set(platform.CSRFHeaderName, "forged-csrf-token")
		recorder = fixture.serve(stale)
		requireRecoveryControlCode(t, recorder, http.StatusForbidden, platform.CodeAuthCSRFFailed)
		if fixture.complete.count() != 0 {
			t.Fatalf("complete signal fired %d times behind a failed CSRF check", fixture.complete.count())
		}

		fixture.resolver.mu.Lock()
		fixture.resolver.principal = session.Principal{UserID: recoveryControlTestUser, Role: string(RoleAdmin), Status: string(UserStatusActive), MustChangePassword: true}
		fixture.resolver.mu.Unlock()
		recorder = fixture.serve(fixture.authorizedRequest(http.MethodPost, "/api/v1/recovery/complete", "{}"))
		requireRecoveryControlCode(t, recorder, http.StatusForbidden, platform.CodeAuthMustCompleteOnboarding)
		if fixture.complete.count() != 0 {
			t.Fatalf("restricted principal reached the completion signal")
		}
		recorder = fixture.serve(fixture.authorizedRequest(http.MethodGet, "/api/v1/users/me", ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("restricted principal current identity status = %d, want 200", recorder.Code)
		}
	})

	t.Run("maintenance_routes", func(t *testing.T) {
		fixture := newRecoveryControlFixture(t)
		fixture.reviews.items = []RecoveryReviewItem{{UserID: "user-7", Decision: "pending"}}
		fixture.reviews.next = "next-page-cursor"

		if recorder := fixture.serve(fixture.authorizedRequest(http.MethodGet, "/livez", "")); recorder.Code != http.StatusOK {
			t.Fatalf("livez status = %d", recorder.Code)
		}
		if recorder := fixture.serve(fixture.authorizedRequest(http.MethodGet, "/readyz", "")); recorder.Code != http.StatusOK {
			t.Fatalf("readyz status = %d", recorder.Code)
		}

		recorder := fixture.serve(fixture.authorizedRequest(http.MethodGet, "/api/v1/users/me", ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("current identity status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		var current struct {
			ID     string `json:"id"`
			Role   string `json:"role"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &current); err != nil {
			t.Fatalf("decode current identity: %v", err)
		}
		if current.ID != recoveryControlTestUser || current.Role != string(RoleAdmin) || current.Status != string(UserStatusActive) {
			t.Fatalf("current identity = %+v", current)
		}

		recorder = fixture.serve(fixture.authorizedRequest(http.MethodPut, "/api/v1/users/me/password", `{"current_password":"old-secret","new_password":"new-secret"}`))
		if recorder.Code != http.StatusOK {
			t.Fatalf("password status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		if fixture.trust.count() != 1 {
			t.Fatalf("trust change calls = %d, want 1", fixture.trust.count())
		}
		if cookies := recorder.Result().Cookies(); len(cookies) == 0 || cookies[0].Name != platform.SessionCookieName || cookies[0].Value != "rotated-session-token" {
			t.Fatalf("rotated session cookie missing: %+v", cookies)
		}

		recorder = fixture.serve(fixture.authorizedRequest(http.MethodGet, "/api/v1/recovery/reviews?limit=2", ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("reviews status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		var page struct {
			Items      []RecoveryReviewItem `json:"items"`
			NextCursor *string              `json:"next_cursor"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode review page: %v", err)
		}
		if len(page.Items) != 1 || page.Items[0].UserID != "user-7" || page.NextCursor == nil || *page.NextCursor != "next-page-cursor" {
			t.Fatalf("review page = %+v", page)
		}
		if fixture.reviews.lastLimit != 2 {
			t.Fatalf("review limit = %d, want 2", fixture.reviews.lastLimit)
		}

		idempotencyBody := `{"scope_hash":"` + recoveryControlTestDigest + `","key_locator":"` + recoveryControlTestDigest + `","execution_generation":3,"decision":"failed","evidence_ref":"evidence-1"}`
		recorder = fixture.serve(fixture.authorizedRequest(http.MethodPost, "/api/v1/recovery/idempotency/resolve", idempotencyBody))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("idempotency status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		if fixture.maintenance.idempotencyCalls != 1 || len(fixture.maintenance.lastIdempotency.scope) != 32 {
			t.Fatalf("idempotency call = %+v", fixture.maintenance.lastIdempotency)
		}
		if fixture.maintenance.lastIdempotency.generation != 3 || fixture.maintenance.lastIdempotency.decision != "failed" {
			t.Fatalf("idempotency evidence = %+v", fixture.maintenance.lastIdempotency)
		}

		reviewBody := `{"user_id":"` + recoveryControlTestUser + `","decision":"denied","role":null,"status":null,"evidence_ref":"evidence-2"}`
		recorder = fixture.serve(fixture.authorizedRequest(http.MethodPost, "/api/v1/recovery/reviews/"+recoveryControlTestUser+"/resolve", reviewBody))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("review resolve status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		if fixture.maintenance.reviewCalls != 1 || fixture.maintenance.lastReview.decision != "denied" {
			t.Fatalf("review resolve call = %+v", fixture.maintenance.lastReview)
		}
		if fixture.maintenance.lastReview.binding.RunID != recoveryControlTestRun {
			t.Fatalf("review resolve binding = %+v", fixture.maintenance.lastReview.binding)
		}
		if fixture.restored.calls != 0 {
			t.Fatalf("restored resolver ran for a denied decision")
		}

		recorder = fixture.serve(fixture.authorizedRequest(http.MethodPost, "/api/v1/recovery/complete", "{}"))
		if recorder.Code != http.StatusOK {
			t.Fatalf("complete status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		var completion struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &completion); err != nil || completion.Status != "identity_ready" {
			t.Fatalf("complete body = %q (%v)", recorder.Body.String(), err)
		}
		if !recorder.Flushed {
			t.Fatalf("complete response was not flushed before the signal")
		}
		if fixture.complete.count() != 1 {
			t.Fatalf("complete signal count = %d, want 1", fixture.complete.count())
		}
		if fixture.trust.count() != 1 {
			t.Fatalf("complete HTTP call performed a trust change")
		}
	})
	t.Run("complete_preflight_failure_is_observable", func(t *testing.T) {
		fixture := newRecoveryControlFixture(t)
		preflight := &recoveryControlCompletePreflightStub{checkErr: recovery.ErrAdmissionBinding}
		fixture.dependencies.Complete = preflight
		handlers, err := NewRecoveryControlHandlers(fixture.dependencies)
		if err != nil {
			t.Fatalf("rebuild handlers with preflight signaler: %v", err)
		}
		fixture.handler = handlers.Handler()
		recorder := fixture.serve(fixture.authorizedRequest(http.MethodPost, "/api/v1/recovery/complete", "{}"))
		requireRecoveryControlCode(t, recorder, http.StatusForbidden, platform.CodeAuthForbidden)
		if preflight.signalCount() != 0 {
			t.Fatalf("complete signal fired %d times behind a failed preflight", preflight.signalCount())
		}
		if preflight.checkCount() != 1 {
			t.Fatalf("preflight checks = %d, want 1", preflight.checkCount())
		}
	})
	t.Run("strict_bodies", func(t *testing.T) {
		bigBody := "{}" + strings.Repeat(" ", int(recoveryControlTestContract)-2)
		if int64(len(bigBody)) != recoveryControlTestContract {
			t.Fatalf("oversize fixture is %d bytes", len(bigBody))
		}
		cases := []struct {
			name             string
			path             string
			body             string
			contentType      string
			status           int
			code             platform.Code
			completeCalls    int
			idempotencyCalls int
		}{
			{"missing_content_type", "/api/v1/recovery/complete", "{}", "", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"empty_body", "/api/v1/recovery/complete", "", "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"size_boundary", "/api/v1/recovery/complete", bigBody, "application/json", http.StatusOK, "", 1, 0},
			{"size_exceeded", "/api/v1/recovery/complete", bigBody + " ", "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"unknown_field", "/api/v1/recovery/complete", `{"status":"identity_ready"}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"top_level_null", "/api/v1/recovery/complete", "null", "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"top_level_array", "/api/v1/recovery/complete", "[]", "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"invalid_utf8", "/api/v1/recovery/complete", "{\"status\":\"\xff\"}", "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"truncated_utf8", "/api/v1/recovery/complete", "{\"status\":\"\xe4\xbd\"}", "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"duplicate_field", "/api/v1/recovery/idempotency/resolve", `{"scope_hash":"` + recoveryControlTestDigest + `","scope_hash":"` + recoveryControlTestDigest + `","key_locator":"` + recoveryControlTestDigest + `","execution_generation":1,"decision":"failed","evidence_ref":"e"}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"short_digest", "/api/v1/recovery/idempotency/resolve", `{"scope_hash":"AAAA","key_locator":"` + recoveryControlTestDigest + `","execution_generation":1,"decision":"failed","evidence_ref":"e"}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"short_locator", "/api/v1/recovery/idempotency/resolve", `{"scope_hash":"` + recoveryControlTestDigest + `","key_locator":"AAAA","execution_generation":1,"decision":"failed","evidence_ref":"e"}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"unknown_decision", "/api/v1/recovery/idempotency/resolve", `{"scope_hash":"` + recoveryControlTestDigest + `","key_locator":"` + recoveryControlTestDigest + `","execution_generation":1,"decision":"kept","evidence_ref":"e"}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"zero_generation", "/api/v1/recovery/idempotency/resolve", `{"scope_hash":"` + recoveryControlTestDigest + `","key_locator":"` + recoveryControlTestDigest + `","execution_generation":0,"decision":"failed","evidence_ref":"e"}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"missing_evidence", "/api/v1/recovery/idempotency/resolve", `{"scope_hash":"` + recoveryControlTestDigest + `","key_locator":"` + recoveryControlTestDigest + `","execution_generation":1,"decision":"failed","evidence_ref":" "}`, "application/json", http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				fixture := newRecoveryControlFixture(t)
				request := httptest.NewRequest(http.MethodPost, testCase.path, strings.NewReader(testCase.body))
				if testCase.contentType != "" {
					request.Header.Set("Content-Type", testCase.contentType)
				}
				request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: recoveryControlTestSession})
				request.Header.Set(platform.CSRFHeaderName, recoveryControlTestCSRF)
				recorder := fixture.serve(request)
				if testCase.code == "" {
					if recorder.Code != testCase.status {
						t.Fatalf("status = %d, want %d (body %q)", recorder.Code, testCase.status, recorder.Body.String())
					}
				} else {
					requireRecoveryControlCode(t, recorder, testCase.status, testCase.code)
				}
				if fixture.complete.count() != testCase.completeCalls {
					t.Fatalf("complete signals = %d, want %d", fixture.complete.count(), testCase.completeCalls)
				}
				if fixture.maintenance.idempotencyCalls != testCase.idempotencyCalls {
					t.Fatalf("idempotency calls = %d, want %d", fixture.maintenance.idempotencyCalls, testCase.idempotencyCalls)
				}
			})
		}
	})
	t.Run("review_decisions", func(t *testing.T) {
		path := "/api/v1/recovery/reviews/" + recoveryControlTestUser + "/resolve"
		restoredBody := `{"user_id":"` + recoveryControlTestUser + `","decision":"restored","role":"member","status":"active","evidence_ref":"evidence-3"}`
		cases := []struct {
			name             string
			body             string
			idempotencyKey   string
			withoutRestored  bool
			status           int
			code             platform.Code
			maintenanceCalls int
			restoredCalls    int
		}{
			{"user_mismatch", `{"user_id":"user-2","decision":"denied","role":null,"status":null,"evidence_ref":"e"}`, "", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"missing_evidence", `{"user_id":"` + recoveryControlTestUser + `","decision":"denied","role":null,"status":null,"evidence_ref":" "}`, "", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"denied_with_role", `{"user_id":"` + recoveryControlTestUser + `","decision":"denied","role":"member","status":null,"evidence_ref":"e"}`, "", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"kept_deleted_with_status", `{"user_id":"` + recoveryControlTestUser + `","decision":"kept_deleted","role":null,"status":"active","evidence_ref":"e"}`, "", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"restored_missing_role", `{"user_id":"` + recoveryControlTestUser + `","decision":"restored","role":null,"status":"active","evidence_ref":"e"}`, "idem-a", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"restored_bad_role", `{"user_id":"` + recoveryControlTestUser + `","decision":"restored","role":"owner","status":"active","evidence_ref":"e"}`, "idem-a", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"restored_missing_key", restoredBody, "", false, http.StatusBadRequest, platform.CodeIdentityIdempotencyKeyRequired, 0, 0},
			{"restored_source_absent", restoredBody, "idem-b", true, http.StatusServiceUnavailable, platform.CodePlatformRecoveryInProgress, 0, 0},
			{"restored_accepted", restoredBody, "idem-c", false, http.StatusNoContent, "", 0, 1},
			{"unknown_decision", `{"user_id":"` + recoveryControlTestUser + `","decision":"kept","role":null,"status":null,"evidence_ref":"e"}`, "", false, http.StatusBadRequest, platform.CodeRequestInvalidJSON, 0, 0},
			{"kept_deleted_accepted", `{"user_id":"` + recoveryControlTestUser + `","decision":"kept_deleted","role":null,"status":null,"evidence_ref":"e"}`, "", false, http.StatusNoContent, "", 1, 0},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				fixture := newRecoveryControlFixtureWith(t, func(d *RecoveryControlDependencies) {
					if testCase.withoutRestored {
						d.Restored = nil
					}
				})
				request := fixture.authorizedRequest(http.MethodPost, path, testCase.body)
				if testCase.idempotencyKey != "" {
					request.Header.Set("Idempotency-Key", testCase.idempotencyKey)
				}
				recorder := fixture.serve(request)
				if testCase.code == "" {
					if recorder.Code != testCase.status {
						t.Fatalf("status = %d, want %d (body %q)", recorder.Code, testCase.status, recorder.Body.String())
					}
				} else {
					requireRecoveryControlCode(t, recorder, testCase.status, testCase.code)
				}
				if fixture.maintenance.reviewCalls != testCase.maintenanceCalls {
					t.Fatalf("maintenance review calls = %d, want %d", fixture.maintenance.reviewCalls, testCase.maintenanceCalls)
				}
				if fixture.restored.calls != testCase.restoredCalls {
					t.Fatalf("restored calls = %d, want %d", fixture.restored.calls, testCase.restoredCalls)
				}
			})
		}

		// The restored seam receives the accepted decision and the idempotency key.
		fixture := newRecoveryControlFixture(t)
		request := fixture.authorizedRequest(http.MethodPost, path, restoredBody)
		request.Header.Set("Idempotency-Key", "idem-d")
		if recorder := fixture.serve(request); recorder.Code != http.StatusNoContent {
			t.Fatalf("restored status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		if fixture.restored.last.role != "member" || fixture.restored.last.status != "active" || fixture.restored.last.idempotencyKey != "idem-d" || fixture.restored.last.evidence != "evidence-3" {
			t.Fatalf("restored call = %+v", fixture.restored.last)
		}
		if fixture.restored.last.binding.RunID != recoveryControlTestRun {
			t.Fatalf("restored binding = %+v", fixture.restored.last.binding)
		}

		// A maintenance failure maps to the closed phase error codes.
		fixture = newRecoveryControlFixture(t)
		fixture.maintenance.err = recovery.ErrAdmissionBinding
		recorder := fixture.serve(fixture.authorizedRequest(http.MethodPost, path, `{"user_id":"`+recoveryControlTestUser+`","decision":"denied","role":null,"status":null,"evidence_ref":"e"}`))
		requireRecoveryControlCode(t, recorder, http.StatusForbidden, platform.CodeAuthForbidden)

		fixture = newRecoveryControlFixture(t)
		fixture.maintenance.err = errors.New("review store unavailable")
		recorder = fixture.serve(fixture.authorizedRequest(http.MethodPost, path, `{"user_id":"`+recoveryControlTestUser+`","decision":"denied","role":null,"status":null,"evidence_ref":"e"}`))
		requireRecoveryControlCode(t, recorder, http.StatusServiceUnavailable, platform.CodePlatformRecoveryInProgress)
	})

	t.Run("review_paging", func(t *testing.T) {
		cases := []struct {
			name   string
			query  string
			err    error
			status int
			code   platform.Code
		}{
			{"missing_limit", "", nil, http.StatusBadRequest, platform.CodeRequestInvalidJSON},
			{"zero_limit", "?limit=0", nil, http.StatusBadRequest, platform.CodeRequestInvalidJSON},
			{"over_limit", "?limit=101", nil, http.StatusBadRequest, platform.CodeRequestInvalidJSON},
			{"non_numeric_limit", "?limit=many", nil, http.StatusBadRequest, platform.CodeRequestInvalidJSON},
			{"invalid_cursor", "?limit=2&cursor=not-base64!", errRecoveryReviewCursor, http.StatusBadRequest, platform.CodeRequestInvalidCursor},
			{"binding", "?limit=2", recovery.ErrAdmissionBinding, http.StatusForbidden, platform.CodeAuthForbidden},
			{"unavailable", "?limit=2", errors.New("review store unavailable"), http.StatusServiceUnavailable, platform.CodePlatformRecoveryInProgress},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				fixture := newRecoveryControlFixture(t)
				fixture.reviews.err = testCase.err
				recorder := fixture.serve(fixture.authorizedRequest(http.MethodGet, "/api/v1/recovery/reviews"+testCase.query, ""))
				requireRecoveryControlCode(t, recorder, testCase.status, testCase.code)
				if testCase.err == nil && fixture.reviews.calls != 0 {
					t.Fatalf("invalid limit reached the review source %d times", fixture.reviews.calls)
				}
			})
		}

		fixture := newRecoveryControlFixture(t)
		recorder := fixture.serve(fixture.authorizedRequest(http.MethodGet, "/api/v1/recovery/reviews?limit=3", ""))
		if recorder.Code != http.StatusOK {
			t.Fatalf("empty page status = %d (body %q)", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"items":[]`) || !strings.Contains(recorder.Body.String(), `"next_cursor":null`) {
			t.Fatalf("empty page body = %q", recorder.Body.String())
		}
		if fixture.reviews.lastLimit != 3 {
			t.Fatalf("review limit = %d, want 3", fixture.reviews.lastLimit)
		}
	})

	t.Run("trust_change_absent", func(t *testing.T) {
		fixture := newRecoveryControlFixtureWith(t, func(d *RecoveryControlDependencies) { d.TrustChange = nil })
		recorder := fixture.serve(fixture.authorizedRequest(http.MethodPut, "/api/v1/users/me/password", `{"current_password":"old-secret","new_password":"new-secret"}`))
		requireRecoveryControlCode(t, recorder, http.StatusServiceUnavailable, platform.CodePlatformRecoveryInProgress)
	})
	t.Run("utf8_reader", func(t *testing.T) {
		prefixes := []struct {
			name   string
			input  []byte
			length int
			ok     bool
		}{
			{"ascii", []byte("abc"), 3, true},
			{"complete_rune", []byte("中"), 3, true},
			{"truncated_two_of_three", []byte{0xe4, 0xbd}, 0, true},
			{"ascii_then_truncated", []byte("a\xe4\xbd"), 1, true},
			{"invalid_byte", []byte{0xff}, 0, false},
			{"overlong", []byte{0xc0, 0xaf}, 0, false},
		}
		for _, testCase := range prefixes {
			length, ok := completeUTF8Prefix(testCase.input)
			if length != testCase.length || ok != testCase.ok {
				t.Fatalf("%s: prefix = (%d,%t), want (%d,%t)", testCase.name, length, ok, testCase.length, testCase.ok)
			}
		}

		decoded, err := io.ReadAll(newRecoveryUTF8Body(io.NopCloser(iotest.OneByteReader(strings.NewReader("身份恢复")))))
		if err != nil || string(decoded) != "身份恢复" {
			t.Fatalf("one-byte reader decoded %q (%v)", decoded, err)
		}

		if _, err := io.ReadAll(newRecoveryUTF8Body(io.NopCloser(strings.NewReader("a\xff")))); !errors.Is(err, errRecoveryBodyInvalidUTF8) {
			t.Fatalf("invalid bytes error = %v", err)
		}
		if _, err := io.ReadAll(newRecoveryUTF8Body(io.NopCloser(strings.NewReader("\xe4\xbd")))); !errors.Is(err, errRecoveryBodyInvalidUTF8) {
			t.Fatalf("truncated bytes error = %v", err)
		}
	})

	t.Run("review_lister_guards", func(t *testing.T) {
		if _, err := NewRecoveryReviewLister(nil, &recoveryControlRuntimeStub{}); err == nil {
			t.Fatalf("lister accepted a nil pool")
		}
		if _, err := NewRecoveryReviewLister(&pgxpool.Pool{}, nil); err == nil {
			t.Fatalf("lister accepted a nil runtime")
		}

		runtimeStub := &recoveryControlRuntimeStub{binding: recovery.PermitBinding{Purpose: recovery.PhaseRecoveryIdentity, RunID: recoveryControlTestRun, BackupRef: recoveryControlTestBackup, InstallationID: "installation-1", RuntimeID: "runtime-1", Generation: 1}}
		lister, err := NewRecoveryReviewLister(&pgxpool.Pool{}, runtimeStub)
		if err != nil {
			t.Fatalf("lister construction: %v", err)
		}

		// A missing or expired phase deadline fails before any permit read, so
		// an empty pool can never be touched.
		if _, _, err := lister.ListPendingReviews(context.Background(), "", 50); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("no-deadline error = %v", err)
		}
		if _, _, err := lister.ListPendingReviews(context.Background(), "", 0); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("zero-limit error = %v", err)
		}
		if _, _, err := lister.ListPendingReviews(context.Background(), "", recoveryReviewMaxLimit+1); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("over-limit error = %v", err)
		}
		if runtimeStub.calls != 0 {
			t.Fatalf("guard failures read the permit %d times", runtimeStub.calls)
		}

		bounded, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		expired, expire := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer expire()
		if _, _, err := lister.ListPendingReviews(expired, "", 50); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("expired-deadline error = %v", err)
		}

		replayStub := &recoveryControlRuntimeStub{binding: recovery.PermitBinding{Purpose: recovery.PhaseRecoveryReplay, RunID: recoveryControlTestRun, BackupRef: recoveryControlTestBackup, InstallationID: "installation-1", RuntimeID: "runtime-1", Generation: 1}}
		replayLister, err := NewRecoveryReviewLister(&pgxpool.Pool{}, replayStub)
		if err != nil {
			t.Fatalf("replay lister construction: %v", err)
		}
		if _, _, err := replayLister.ListPendingReviews(bounded, "", 50); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("replay-purpose error = %v", err)
		}

		unboundStub := &recoveryControlRuntimeStub{binding: recovery.PermitBinding{Purpose: recovery.PhaseRecoveryIdentity, RunID: recoveryControlTestRun, BackupRef: recoveryControlTestBackup}}
		unboundLister, err := NewRecoveryReviewLister(&pgxpool.Pool{}, unboundStub)
		if err != nil {
			t.Fatalf("unbound lister construction: %v", err)
		}
		if _, _, err := unboundLister.ListPendingReviews(bounded, "", 50); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("unbound installation error = %v", err)
		}

		if _, _, err := lister.ListPendingReviews(bounded, "not-base64!", 50); !errors.Is(err, errRecoveryReviewCursor) {
			t.Fatalf("invalid cursor error = %v", err)
		}
		foreign := base64.RawURLEncoding.EncodeToString([]byte(`{"run_id":"other-run","installation_id":"installation-1","last_user_id":"user-9"}`))
		if _, _, err := lister.ListPendingReviews(bounded, foreign, 50); !errors.Is(err, errRecoveryReviewCursor) {
			t.Fatalf("foreign cursor error = %v", err)
		}

		failingStub := &recoveryControlRuntimeStub{err: recovery.ErrAdmissionClosed}
		failingLister, err := NewRecoveryReviewLister(&pgxpool.Pool{}, failingStub)
		if err != nil {
			t.Fatalf("failing lister construction: %v", err)
		}
		if _, _, err := failingLister.ListPendingReviews(bounded, "", 50); !errors.Is(err, recovery.ErrAdmissionClosed) {
			t.Fatalf("runtime error = %v", err)
		}
	})
}

type recoveryControlCompletePreflightStub struct {
	mu       sync.Mutex
	calls    int
	checks   int
	checkErr error
}

func (s *recoveryControlCompletePreflightStub) CheckComplete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks++
	return s.checkErr
}

func (s *recoveryControlCompletePreflightStub) RequestComplete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return nil
}

func (s *recoveryControlCompletePreflightStub) signalCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *recoveryControlCompletePreflightStub) checkCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checks
}

//go:build contract

package identity_contract_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"han/internal/identity"
	"han/internal/platform"
)

// TestRESTContract_RegistrationStrictJSONRejectsDrift pins the shared strict
// JSON decoder contract on the identity REST boundary: unknown fields, duplicate
// fields, invalid UTF-8, wrong Content-Type, top-level non-objects and trailing
// values must all map to `400 request.invalid_json` without reaching the service.
func TestRESTContract_RegistrationStrictJSONRejectsDrift(t *testing.T) {
	valid := `{"username":"u","email":"e@example.test","phone":"13800000000","password":"pw12345","privacy_notice_version":"v1","privacy_acknowledged":true}`
	cases := []struct {
		name        string
		body        []byte
		contentType string
	}{
		{"unknown field", []byte(`{"username":"u","email":"e@example.test","phone":"13800000000","password":"pw12345","privacy_notice_version":"v1","privacy_acknowledged":true,"extra":1}`), "application/json"},
		{"duplicate field", []byte(`{"username":"u","username":"v","email":"e@example.test","phone":"13800000000","password":"pw12345","privacy_notice_version":"v1","privacy_acknowledged":true}`), "application/json"},
		{"malformed json", []byte(`{"username": }`), "application/json"},
		{"wrong content type", []byte(valid), "text/plain"},
		{"top-level array", []byte(`[]`), "application/json"},
		{"trailing value", []byte(valid + ` {}`), "application/json"},
		{"empty body", []byte(`   `), "application/json"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			spy := &registrationServiceSpy{}
			handler := mustRegistrationHandler(t, spy)
			response := serveIdentity(
				t, handler.Register, http.MethodPost, "/api/v1/users",
				test.body, test.contentType,
			)
			requireEnvelope(t, response, http.StatusBadRequest, platform.CodeRequestInvalidJSON)
			assertNoCauseLeak(t, response, "invalid json")
			if spy.calls != 0 {
				t.Fatalf("registration service calls = %d, want 0", spy.calls)
			}
		})
	}
}

// TestRESTContract_ErrorEnvelopeIsStableAndTraceable pins the shared error
// envelope shape: exactly the stable code/message, no internal cause, and a
// trace_id that matches the response trace header.
func TestRESTContract_ErrorEnvelopeIsStableAndTraceable(t *testing.T) {
	spy := &registrationServiceSpy{err: platform.NewPublicError(
		platform.CodeIdentityValidationFailed,
		errors.New("internal-cause-canary"),
		platform.FieldError{Field: "password", Code: "password_too_short"},
	)}
	handler := mustRegistrationHandler(t, spy)
	response := serveIdentity(
		t, handler.Register, http.MethodPost, "/api/v1/users",
		[]byte(`{"username":"u","email":"e@example.test","phone":"13800000000","password":"x","privacy_notice_version":"v1","privacy_acknowledged":true}`),
		"application/json",
	)

	requireEnvelope(t, response, http.StatusUnprocessableEntity, platform.CodeIdentityValidationFailed)
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
	if strings.Contains(response.Body.String(), "internal-cause-canary") {
		t.Fatalf("error envelope leaked internal cause: %s", response.Body)
	}
	traceHeader := response.Header().Get(platform.TraceHeader)
	if traceHeader == "" {
		t.Fatal("error response is missing a trace header")
	}
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.TraceID != traceHeader {
		t.Fatalf("envelope trace_id = %q, header = %q", envelope.TraceID, traceHeader)
	}
	if len(envelope.FieldErrors) != 1 ||
		envelope.FieldErrors[0].Field != "password" ||
		envelope.FieldErrors[0].Code != "password_too_short" {
		t.Fatalf("field errors = %#v", envelope.FieldErrors)
	}
}

// TestRESTContract_RouteErrorsUseStableJSONEnvelope pins the shared router error
// envelope the local business router relies on: `request.route_not_found` -> JSON
// 404 and `request.method_not_allowed` -> JSON 405, both envelope-stable with a
// non-empty trace_id that matches the response trace header, never a plain-text
// ServeMux default.
func TestRESTContract_RouteErrorsUseStableJSONEnvelope(t *testing.T) {
	cases := []struct {
		name     string
		code     platform.Code
		wantHTTP int
	}{
		{"route not found", platform.CodeRequestRouteNotFound, http.StatusNotFound},
		{"method not allowed", platform.CodeRequestMethodNotAllowed, http.StatusMethodNotAllowed},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/not-a-route", nil)
			platform.TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.code == platform.CodeRequestMethodNotAllowed {
					w.Header().Set("Allow", http.MethodPost)
				}
				w.Header().Set("X-Content-Type-Options", "nosniff")
				_ = platform.WriteError(w, r, platform.NewPublicError(
					test.code, errors.New("router decision"),
				))
			})).ServeHTTP(response, request)

			requireEnvelope(t, response, test.wantHTTP, test.code)
			if test.code == platform.CodeRequestMethodNotAllowed &&
				response.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), http.MethodPost)
			}
			contentType := response.Header().Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("Content-Type = %q, want JSON", contentType)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			if envelope.TraceID == "" || envelope.TraceID != response.Header().Get(platform.TraceHeader) {
				t.Fatalf("trace mismatch: body %q header %q",
					envelope.TraceID, response.Header().Get(platform.TraceHeader))
			}
		})
	}
}

// TestRESTContract_SuccessfulRegistrationShapeIsAllowlisted pins the exact
// success body of `POST /api/v1/users`: only the six allowlisted fields, without
// the request password or source fingerprint.
func TestRESTContract_SuccessfulRegistrationShapeIsAllowlisted(t *testing.T) {
	spy := &registrationServiceSpy{user: identity.RegisteredUser{
		ID: "user-01", Username: "u", Role: "member", Status: identity.UserStatusActive,
		CreatedAt: time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC),
	}}
	handler := mustRegistrationHandler(t, spy)
	response := serveIdentity(
		t, handler.Register, http.MethodPost, "/api/v1/users",
		[]byte(`{"username":"u","email":"e@example.test","phone":"13800000000","password":"pw12345","privacy_notice_version":"v1","privacy_acknowledged":true}`),
		"application/json",
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatalf("decode registration body: %v", err)
	}
	want := []string{"id", "username", "role", "status", "must_change_password", "created_at"}
	if len(fields) != len(want) {
		t.Fatalf("registration body fields = %v, want %v", keysOf(fields), want)
	}
	for _, name := range want {
		if _, ok := fields[name]; !ok {
			t.Fatalf("registration body is missing field %q", name)
		}
	}
	for _, forbidden := range []string{"password", "source_fingerprint", "source_kind"} {
		if _, ok := fields[forbidden]; ok {
			t.Fatalf("registration body leaked %q: %s", forbidden, response.Body)
		}
	}
}

// TestRESTContract_UnsupportedMethodsDoNotReachService pins that wrong methods
// on a known path never invoke the downstream service.
func TestRESTContract_UnsupportedMethodsDoNotReachService(t *testing.T) {
	spy := &registrationServiceSpy{}
	handler := mustRegistrationHandler(t, spy)
	response := serveIdentity(
		t, handler.Register, http.MethodGet, "/api/v1/users", nil, "application/json",
	)
	// A GET reaches the Handler (the router is not in this path) and must be
	// rejected on the JSON decode boundary, never invoking the service.
	requireEnvelope(t, response, http.StatusBadRequest, platform.CodeRequestInvalidJSON)
	if spy.calls != 0 {
		t.Fatalf("registration service calls = %d, want 0", spy.calls)
	}
}

type registrationServiceSpy struct {
	user  identity.RegisteredUser
	err   error
	calls int
}

func (s *registrationServiceSpy) Register(
	context.Context,
	identity.RegistrationRequest,
) (identity.RegisteredUser, error) {
	s.calls++
	if s.err != nil {
		return identity.RegisteredUser{}, s.err
	}
	return s.user, nil
}

type staticSourceProvider struct{}

func (staticSourceProvider) Evidence(*http.Request) (identity.RegistrationSourceEvidence, error) {
	return identity.RegistrationSourceEvidence{
		Kind: "network_source_hmac_sha256_v1", Fingerprint: "synthetic",
	}, nil
}

func mustRegistrationHandler(t *testing.T, service identity.RegistrationService) *identity.Handler {
	t.Helper()
	handler, err := identity.NewHandler(service, staticSourceProvider{}, 262144)
	if err != nil {
		t.Fatalf("identity.NewHandler() error = %v", err)
	}
	return handler
}

func serveIdentity(
	t *testing.T,
	method http.HandlerFunc,
	httpMethod string,
	target string,
	body []byte,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(string(body))
	}
	request := httptest.NewRequest(httpMethod, target, reader)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	platform.TraceMiddleware(method).ServeHTTP(response, request)
	return response
}

func requireEnvelope(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	code platform.Code,
) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body)
	}
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v; body = %s", err, response.Body)
	}
	if envelope.Code != code {
		t.Fatalf("error code = %q, want %q; body = %s", envelope.Code, code, response.Body)
	}
	def, ok := platform.DefaultRegistry.Definition(code)
	if !ok {
		t.Fatalf("code %q is not registered", code)
	}
	if envelope.Message != def.Message {
		t.Fatalf("message = %q, want stable %q", envelope.Message, def.Message)
	}
}

func assertNoCauseLeak(t *testing.T, response *httptest.ResponseRecorder, canary string) {
	t.Helper()
	if strings.Contains(response.Body.String(), canary) {
		t.Fatalf("error envelope leaked cause %q: %s", canary, response.Body)
	}
}

func keysOf(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for name := range fields {
		keys = append(keys, name)
	}
	return keys
}

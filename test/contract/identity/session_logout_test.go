//go:build contract

package identity_contract_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"han/internal/identity"
	"han/internal/platform"
)

// newLogoutRequest builds the same POST the SPA sends; an empty token models
// the unauthenticated caller.
func newLogoutRequest(token string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/logout", nil)
	if token != "" {
		request.AddCookie(&http.Cookie{
			Name:  platform.SessionCookieName,
			Value: token,
		})
	}
	return request
}

// serveLogout runs the handler behind the shared trace middleware so error
// envelopes carry a trace id exactly as in production.
func serveLogout(
	t *testing.T,
	handler *identity.SessionLogoutHandler,
	request *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	platform.TraceMiddleware(http.HandlerFunc(handler.Logout)).ServeHTTP(response, request)
	return response
}

// logoutRevokerSpy records what the handler asked it to revoke.
type logoutRevokerSpy struct {
	token  string
	at     time.Time
	reason string
	calls  int
	err    error
}

func (s *logoutRevokerSpy) RevokeCurrentSession(
	_ context.Context,
	token string,
	revokedAt time.Time,
	reason string,
) error {
	s.calls++
	s.token = token
	s.at = revokedAt
	s.reason = reason
	return s.err
}

func mustLogoutHandler(t *testing.T, revoker identity.SessionLogoutRevoker) *identity.SessionLogoutHandler {
	t.Helper()
	handler, err := identity.NewSessionLogoutHandler(
		revoker,
		func() time.Time { return time.Date(2026, time.November, 6, 8, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("identity.NewSessionLogoutHandler() error = %v", err)
	}
	return handler
}

// TestRESTContract_SessionLogoutRevokesAndClearsCookie pins the success shape of
// POST /api/v1/sessions/logout: 204, no body, one clearing Set-Cookie, and one
// revoke call carrying the presented token with the stable reason "logout".
func TestRESTContract_SessionLogoutRevokesAndClearsCookie(t *testing.T) {
	spy := &logoutRevokerSpy{}
	handler := mustLogoutHandler(t, spy)

	request := newLogoutRequest("session-token-canary")
	response := serveLogout(t, handler, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", response.Code, response.Body)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("204 response carried a body: %s", response.Body)
	}
	if spy.calls != 1 {
		t.Fatalf("revoke calls = %d, want 1", spy.calls)
	}
	if spy.token != "session-token-canary" || spy.reason != "logout" {
		t.Fatalf("revoke call = %q/%q, want token/\"logout\"", spy.token, spy.reason)
	}
	if spy.at.IsZero() {
		t.Fatal("revoke call carried a zero revocation time")
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie count = %d, want 1: %#v", len(cookies), cookies)
	}
	cleared := cookies[0]
	if cleared.Name != platform.SessionCookieName {
		t.Fatalf("cleared cookie name = %q, want %q", cleared.Name, platform.SessionCookieName)
	}
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Fatalf("cookie not cleared: value=%q maxAge=%d", cleared.Value, cleared.MaxAge)
	}
	if !cleared.HttpOnly || !cleared.Secure {
		t.Fatalf("cleared cookie lost HttpOnly/Secure: %#v", cleared)
	}
	if cleared.SameSite != http.SameSiteLaxMode || cleared.Path != "/" {
		t.Fatalf("cleared cookie attributes drifted: %#v", cleared)
	}
}

// TestRESTContract_SessionLogoutWithoutCookieIs401 pins that a cookie-less call
// fails as an authenticated-endpoint call must, using the shared 401 envelope
// and never reaching the revoker.
func TestRESTContract_SessionLogoutWithoutCookieIs401(t *testing.T) {
	spy := &logoutRevokerSpy{}
	handler := mustLogoutHandler(t, spy)

	response := serveLogout(t, handler, newLogoutRequest(""))

	requireEnvelope(t, response, http.StatusUnauthorized, platform.CodeAuthSessionRequired)
	if spy.calls != 0 {
		t.Fatalf("revoke calls = %d, want 0 for a cookie-less request", spy.calls)
	}
}

// TestRESTContract_SessionLogoutRevokeFailureHidesCause pins that a failing
// revoker degrades to the stable internal error envelope without leaking the
// internal cause or clearing the cookie as if the revoke had succeeded.
func TestRESTContract_SessionLogoutRevokeFailureHidesCause(t *testing.T) {
	spy := &logoutRevokerSpy{err: errors.New("internal-cause-canary")}
	handler := mustLogoutHandler(t, spy)

	response := serveLogout(t, handler, newLogoutRequest("session-token-canary"))

	requireEnvelope(t, response, http.StatusInternalServerError, platform.CodePlatformInternalError)
	assertNoCauseLeak(t, response, "internal-cause-canary")
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("failed revoke cleared the cookie anyway: %#v", cookies)
	}
}

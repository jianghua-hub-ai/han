package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"han/internal/platform"
)

type logoutRevokerStub struct {
	token  string
	reason string
	err    error
}

func (s *logoutRevokerStub) RevokeCurrentSession(
	_ context.Context,
	token string,
	_ time.Time,
	reason string,
) error {
	s.token = token
	s.reason = reason
	return s.err
}

func TestSessionLogout_RevokesAndClearsCookie(t *testing.T) {
	revoker := &logoutRevokerStub{}
	handler, err := NewSessionLogoutHandler(
		revoker,
		func() time.Time { return time.Unix(100, 0).UTC() },
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/logout", nil)
	request.AddCookie(&http.Cookie{
		Name:  platform.SessionCookieName,
		Value: "session-token",
	})
	response := httptest.NewRecorder()

	handler.Logout(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if revoker.token != "session-token" || revoker.reason != "logout" {
		t.Fatalf("revoke call = %q/%q", revoker.token, revoker.reason)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 ||
		cookies[0].Name != platform.SessionCookieName ||
		cookies[0].MaxAge >= 0 {
		t.Fatalf("session cookie not cleared: %#v", cookies)
	}
}

func TestSessionLogout_MissingCookieRejected(t *testing.T) {
	handler, err := NewSessionLogoutHandler(
		&logoutRevokerStub{},
		func() time.Time { return time.Unix(100, 0).UTC() },
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	response := httptest.NewRecorder()

	handler.Logout(
		response,
		httptest.NewRequest(http.MethodPost, "/api/v1/sessions/logout", nil),
	)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

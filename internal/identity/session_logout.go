package identity

import (
	"context"
	"errors"
	"net/http"
	"time"

	"han/internal/platform"
)

// SessionLogoutRevoker invalidates the session identified by one token.
type SessionLogoutRevoker interface {
	RevokeCurrentSession(
		ctx context.Context,
		token string,
		revokedAt time.Time,
		reason string,
	) error
}

// SessionLogoutHandler ends the current session and clears its cookie.
type SessionLogoutHandler struct {
	revoker SessionLogoutRevoker
	clock   func() time.Time
}

// NewSessionLogoutHandler validates dependencies for the logout endpoint.
func NewSessionLogoutHandler(
	revoker SessionLogoutRevoker,
	clock func() time.Time,
) (*SessionLogoutHandler, error) {
	if revoker == nil {
		return nil, errors.New("session logout revoker is required")
	}
	if clock == nil {
		return nil, errors.New("session logout clock is required")
	}
	return &SessionLogoutHandler{revoker: revoker, clock: clock}, nil
}

// Logout revokes the presented session; repeated calls stay side-effect free.
func (h *SessionLogoutHandler) Logout(
	response http.ResponseWriter,
	request *http.Request,
) {
	cookie, err := request.Cookie(platform.SessionCookieName)
	if err != nil || cookie.Value == "" {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("logout session cookie is missing"),
		))
		return
	}
	if err := h.revoker.RevokeCurrentSession(
		request.Context(),
		cookie.Value,
		h.clock(),
		"logout",
	); err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		))
		return
	}
	platform.ClearSessionCookie(response)
	response.WriteHeader(http.StatusNoContent)
}

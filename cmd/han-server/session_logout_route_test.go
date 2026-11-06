package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func logoutRouteHandlers() []http.Handler {
	handlers := make([]http.Handler, 23)
	handlers[22] = http.HandlerFunc(func(
		response http.ResponseWriter,
		_ *http.Request,
	) {
		response.WriteHeader(http.StatusNoContent)
	})
	return handlers
}

func TestSessionLogoutRoute_PostIsDispatched(t *testing.T) {
	router := newLoginBusinessRouter(nil, logoutRouteHandlers()...)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/sessions/logout",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestSessionLogoutRoute_OtherMethodsRejected(t *testing.T) {
	router := newLoginBusinessRouter(nil, logoutRouteHandlers()...)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sessions/logout",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow = %q, want POST", allow)
	}
}

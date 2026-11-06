package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ir02RouteHandlers registers the two routes IR-02 added on top of the 23
// existing slots.
func ir02RouteHandlers() []http.Handler {
	handlers := make([]http.Handler, 25)
	handlers[23] = http.HandlerFunc(func(
		response http.ResponseWriter,
		_ *http.Request,
	) {
		response.WriteHeader(http.StatusCreated)
	})
	handlers[24] = http.HandlerFunc(func(
		response http.ResponseWriter,
		_ *http.Request,
	) {
		response.WriteHeader(http.StatusNoContent)
	})
	return handlers
}

func TestRegistrationRoute_PostIsDispatched(t *testing.T) {
	router := newLoginBusinessRouter(nil, ir02RouteHandlers()...)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.Code)
	}
}

func TestRegistrationRoute_OtherMethodsRejected(t *testing.T) {
	router := newLoginBusinessRouter(nil, ir02RouteHandlers()...)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow = %q, want POST", allow)
	}
}

func TestMyArticlesRoute_GetIsDispatched(t *testing.T) {
	router := newLoginBusinessRouter(nil, ir02RouteHandlers()...)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me/articles",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestMyArticlesRoute_OtherMethodsRejected(t *testing.T) {
	router := newLoginBusinessRouter(nil, ir02RouteHandlers()...)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/me/articles",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", allow)
	}
}

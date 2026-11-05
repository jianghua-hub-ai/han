//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"han/internal/platform"
)

// TestRecoveryIsolationMatrix extends the production dispatcher coverage from
// two admin routes to every guarded business route in the cmd/han-server
// router. It proves that closed/unknown admission blocks each route before
// authorization, origin, session, repository or reader work, while liveness
// and readiness probes stay outside the guard. A guard-open control proves the
// authorization and origin observers are wired and reachable.
func TestRecoveryIsolationMatrix(t *testing.T) {
	routes := recoveryIsolationBusinessRoutes()
	for _, state := range []string{"closed", "unknown"} {
		t.Run(state, func(t *testing.T) {
			var (
				authorizations int
				origins        int
			)
			runtime := newS419RecoveryRuntime(state)
			fixture := newS419ProductionDispatcher(t, runtime.handle, func(seam *articleProductionTestSeam) {
				seam.authorized = func(string) { authorizations++ }
				seam.protectOrigin = func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
						origins++
						next.ServeHTTP(response, request)
					})
				}
			})

			s419AssertProbe(t, fixture, http.MethodGet, "/livez", http.StatusOK, "alive")
			s419AssertProbe(t, fixture, http.MethodGet, "/readyz", http.StatusServiceUnavailable, "unavailable")

			for _, route := range routes {
				route := route
				t.Run(route.name, func(t *testing.T) {
					request := httptest.NewRequest(route.method, route.path, nil)
					request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "admin-session-token"})
					response := httptest.NewRecorder()
					fixture.handler.ServeHTTP(response, request)

					if response.Code != http.StatusServiceUnavailable {
						t.Fatalf("%s %s status=%d, want 503; a 404 means the handler was not registered",
							route.method, route.path, response.Code)
					}
					var envelope platform.Envelope
					if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
						t.Fatalf("decode %s %s envelope: %v", route.method, route.path, err)
					}
					if envelope.Code != platform.CodePlatformRecoveryInProgress {
						t.Fatalf("%s %s code=%q, want %q", route.method, route.path, envelope.Code, platform.CodePlatformRecoveryInProgress)
					}
					if envelope.TraceID == "" || envelope.TraceID != response.Header().Get(platform.TraceHeader) {
						t.Fatalf("%s %s envelope Trace=%q, header=%q", route.method, route.path,
							envelope.TraceID, response.Header().Get(platform.TraceHeader))
					}
					s15AssertNoRecoveryLeak(t, response.Body.Bytes(), envelope.TraceID)
				})
			}

			if authorizations != 0 || origins != 0 {
				t.Fatalf("guard reached authorization=%d origin=%d before admission", authorizations, origins)
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
			if acquires, _, releases := runtime.snapshot(); acquires != len(routes) || releases != 0 {
				t.Fatalf("guarded acquire/release=(%d,%d), want (%d,0); probes must stay outside the guard",
					acquires, releases, len(routes))
			}
			if len(fixture.events) != len(routes)+2 {
				t.Fatalf("events=%d, want %d (probes plus guarded routes)", len(fixture.events), len(routes)+2)
			}
			s419AssertOnlyTrace(t, fixture.events)
		})
	}

	t.Run("guard_open_reaches_authorization_and_origin", func(t *testing.T) {
		var (
			authorizations []string
			origins        int
		)
		runtime := newS419RecoveryRuntime("public_ready")
		fixture := newS419ProductionDispatcher(t, runtime.handle, func(seam *articleProductionTestSeam) {
			seam.authorized = func(route string) { authorizations = append(authorizations, route) }
			seam.protectOrigin = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					origins++
					next.ServeHTTP(response, request)
				})
			}
			seam.createHandler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.WriteHeader(http.StatusCreated)
			})
		})

		request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://community.example.invalid")
		request.Header.Set(platform.CSRFHeaderName, "csrf-token")
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "admin-session-token"})
		response := httptest.NewRecorder()
		fixture.handler.ServeHTTP(response, request)

		if response.Code != http.StatusCreated {
			t.Fatalf("POST /api/v1/articles status=%d, want 201", response.Code)
		}
		if origins != 1 {
			t.Fatalf("origin stage entered %d time(s), want 1", origins)
		}
		if len(authorizations) != 1 || authorizations[0] != "create" {
			t.Fatalf("authorization observers=%v, want [create]", authorizations)
		}
		if acquires, _, releases := runtime.snapshot(); acquires != 1 || releases != 1 {
			t.Fatalf("open acquire/release=(%d,%d), want (1,1)", acquires, releases)
		}
	})
}

// s15AssertNoRecoveryLeak hides the validated Trace ID before applying the
// shared leak assertion. A ULID Trace can randomly contain a forbidden
// substring such as "ref", which would otherwise make the check flaky.
func s15AssertNoRecoveryLeak(t *testing.T, body []byte, traceID string) {
	t.Helper()
	masked := body
	if traceID != "" {
		masked = bytes.ReplaceAll(masked, []byte(traceID), []byte("<trace>"))
	}
	s419AssertNoRecoveryLeak(t, masked)
}

type recoveryIsolationRoute struct {
	name   string
	method string
	path   string
}

func recoveryIsolationBusinessRoutes() []recoveryIsolationRoute {
	const targetID = "30000000-0000-4000-8000-000000000003"
	return []recoveryIsolationRoute{
		{"sessions", http.MethodPost, "/api/v1/sessions"},
		{"password_change", http.MethodPut, "/api/v1/users/me/password"},
		{"administrator_notice", http.MethodGet, "/api/v1/admin/administrator-information-provision-notices/current"},
		{"administrator_create", http.MethodPost, "/api/v1/admin/administrators"},
		{"privacy_acknowledgement", http.MethodPost, "/api/v1/users/me/privacy-acknowledgements"},
		{"user_password_reset", http.MethodPost, "/api/v1/admin/users/" + targetID + "/password-reset"},
		{"administrator_password_reset", http.MethodPost, "/api/v1/admin/administrators/" + targetID + "/password-reset"},
		{"user_temporary_credential_resend", http.MethodPost, "/api/v1/admin/users/" + targetID + "/password-reset/resend"},
		{"administrator_temporary_credential_resend", http.MethodPost, "/api/v1/admin/administrators/" + targetID + "/temporary-password/resend"},
		{"article_violation", http.MethodPost, "/api/v1/admin/articles/" + articleAuthorRootID + "/violation"},
		{"article_restore", http.MethodPost, "/api/v1/admin/articles/" + articleAuthorRootID + "/restore"},
		{"orphan_list", http.MethodGet, "/api/v1/admin/orphan-articles"},
		{"orphan_publish", http.MethodPost, "/api/v1/admin/orphan-articles/" + articleAuthorRootID + "/publish"},
		{"orphan_transfer", http.MethodPost, "/api/v1/admin/orphan-articles/" + articleAuthorRootID + "/transfer"},
		{"article_list", http.MethodGet, "/api/v1/articles"},
		{"article_create", http.MethodPost, "/api/v1/articles"},
		{"admin_article_list", http.MethodGet, "/api/v1/admin/articles"},
		{"author_article_detail", http.MethodGet, "/api/v1/users/me/articles/" + articleAuthorRootID},
		{"article_detail", http.MethodGet, "/api/v1/articles/" + articleAuthorRootID},
		{"article_update", http.MethodPut, "/api/v1/articles/" + articleAuthorRootID},
		{"article_delete", http.MethodDelete, "/api/v1/articles/" + articleAuthorRootID},
		{"admin_article_detail", http.MethodGet, "/api/v1/admin/articles/" + articleAuthorRootID},
		{"admin_article_delete", http.MethodDelete, "/api/v1/admin/articles/" + articleAuthorRootID},
		{"admin_orphan_detail", http.MethodGet, "/api/v1/admin/orphan-articles/" + articleAuthorRootID},
		{"admin_orphan_delete", http.MethodDelete, "/api/v1/admin/orphan-articles/" + articleAuthorRootID},
	}
}

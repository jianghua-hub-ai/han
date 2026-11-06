//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"han/internal/identity"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ir02LogoutMigrationFiles is the forward chain this fixture needs. It is the
// same ordered list the identity integration fixtures apply.
var ir02LogoutMigrationFiles = []string{
	"0001_shared.sql",
	"0002_identity.sql",
	"0003_article.sql",
	"0004_idempotency.sql",
	"0005_recovery_runs.sql",
	"0006_user_governance_indexes.sql",
	"0007_idempotency_response_snapshot_v1.sql",
	"0008_password_reset_durable.sql",
	"0009_article_replay_attempt.sql",
	"0010_recovery_runtime_metadata.sql",
	"0011_article_replay_publication.sql",
}

// ir02LogoutFixture owns one isolated schema that is dropped on cleanup, so the
// suite leaves no residue behind.
type ir02LogoutFixture struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	schema string
}

func ir02NewLogoutFixture(t *testing.T) *ir02LogoutFixture {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("HAN_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN is required for the IR-02 logout integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid HAN_TEST_POSTGRES_DSN value")
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	schema := "ir02_logout_" + ir02RandomHex(t)
	quoted := pgx.Identifier{schema}.Sanitize()
	config.ConnConfig.RuntimeParams["search_path"] = quoted
	config.MaxConns = 4
	config.MinConns = 0

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL pool for the IR-02 logout integration test")
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE"); err != nil {
			t.Errorf("drop isolated schema %s: %v", schema, err)
		}
	})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin migration transaction: %v", err)
	}
	for _, name := range ir02LogoutMigrationFiles {
		raw, readErr := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if readErr != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("read migration %s: %v", name, readErr)
		}
		if _, err := tx.Exec(ctx, string(raw)); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit migrations: %v", err)
	}

	return &ir02LogoutFixture{ctx: ctx, pool: pool, schema: schema}
}

// ir02SeedActiveMember inserts one active member the session can belong to.
func (f *ir02LogoutFixture) ir02SeedActiveMember(t *testing.T, now time.Time) string {
	t.Helper()
	userID := ir02NewUUID(t)
	username := "ir02-logout-" + ir02RandomHex(t)
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO users (
			id, username, email, phone, role, status, credential_mode,
			permanent_password_digest, permanent_password_generation,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'member', 'active', 'permanent',
			'synthetic-ir02-password-digest', 0, $5, $5)
	`,
		userID,
		username,
		username+"@example.test",
		"+1555"+strings.ReplaceAll(userID, "-", "")[:8],
		now,
	); err != nil {
		t.Fatalf("insert active member: %v", err)
	}
	return userID
}

// TestSessionLogoutRevokesRealPostgresSession is the integration proof S2.1
// still owed: the production logout revoker commits a real revocation to
// PostgreSQL, the response clears the cookie, and the same cookie can no longer
// authorize a protected route. Only the origin guard is left out, because it is
// orthogonal CSRF hardening already covered by its own tests.
func TestSessionLogoutRevokesRealPostgresSession(t *testing.T) {
	fixture := ir02NewLogoutFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := fixture.ir02SeedActiveMember(t, now)

	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      fixture.pool,
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		t.Fatalf("real session store: %v", err)
	}

	const (
		token = "ir02-logout-session-token"
		csrf  = "ir02-logout-csrf-token"
	)
	if _, err := store.Create(fixture.ctx, fixture.pool, session.CreateInput{
		ID:                          ir02NewUUID(t),
		UserID:                      userID,
		Token:                       token,
		CSRFToken:                   csrf,
		AuthenticatedCredentialKind: session.CredentialPermanent,
		AuthGeneration:              0,
		IssuedAt:                    now,
		AbsoluteExpiresAt:           now.Add(24 * time.Hour),
		IdleExpiresAt:               now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create live session: %v", err)
	}

	// The same store the authorizer uses must accept the token before logout;
	// otherwise a later 401 would prove nothing about revocation.
	if _, err := store.ResolvePrincipal(fixture.ctx, session.ResolveInput{
		Token:       token,
		CSRFToken:   csrf,
		RequireCSRF: true,
		Now:         now,
	}); err != nil {
		t.Fatalf("live session does not resolve at the store layer: %v", err)
	}

	authorizer, err := platform.NewHTTPAuthorizer(store, func() time.Time { return now })
	if err != nil {
		t.Fatalf("real HTTP authorizer: %v", err)
	}
	probe := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	protectedProbe := authorizer.Protect(session.AccessGeneral, probe)

	// Baseline: the freshly minted cookie really does authorize the route, so a
	// later 401 cannot be explained by a broken fixture.
	if status := ir02ProbeStatus(t, protectedProbe, token); status != http.StatusNoContent {
		t.Fatalf("live session probe status = %d, want 204", status)
	}

	logout, err := identity.NewSessionLogoutHandler(
		postgresSessionLogout{pool: fixture.pool, store: &store},
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatalf("new logout handler: %v", err)
	}
	protectedLogout := authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(logout.Logout),
	)

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
	logoutRequest.Header.Set(platform.CSRFHeaderName, csrf)
	logoutResponse := httptest.NewRecorder()
	protectedLogout.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204 (body=%s)", logoutResponse.Code, logoutResponse.Body.String())
	}
	cookies := logoutResponse.Result().Cookies()
	if len(cookies) != 1 ||
		cookies[0].Name != platform.SessionCookieName ||
		cookies[0].MaxAge >= 0 {
		t.Fatalf("session cookie not cleared: %#v", cookies)
	}

	var revokedAt *time.Time
	var reason *string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT revoked_at, revoke_reason
		FROM sessions
		WHERE user_id = $1
	`, userID).Scan(&revokedAt, &reason); err != nil {
		t.Fatalf("read persisted revocation: %v", err)
	}
	if revokedAt == nil {
		t.Fatal("sessions.revoked_at is NULL after logout, want a committed revocation")
	}
	if reason == nil || *reason != "logout" {
		t.Fatalf("sessions.revoke_reason = %v, want \"logout\"", reason)
	}

	// The decisive assertion: the same cookie is no longer authorized.
	if status := ir02ProbeStatus(t, protectedProbe, token); status != http.StatusUnauthorized {
		t.Fatalf("revoked session probe status = %d, want 401", status)
	}

	// Repeating the call must not turn into a 5xx.
	repeatRequest := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/logout", nil)
	repeatRequest.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
	repeatRequest.Header.Set(platform.CSRFHeaderName, csrf)
	repeatResponse := httptest.NewRecorder()
	protectedLogout.ServeHTTP(repeatResponse, repeatRequest)
	if repeatResponse.Code != http.StatusUnauthorized {
		t.Fatalf("repeated logout status = %d, want 401", repeatResponse.Code)
	}
}

func ir02ProbeStatus(t *testing.T, handler http.Handler, token string) int {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/probe", nil)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func ir02RandomHex(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 6)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("read random bytes: %v", err)
	}
	return hex.EncodeToString(raw)
}

func ir02NewUUID(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("read random UUID bytes: %v", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return strings.Join([]string{
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	}, "-")
}

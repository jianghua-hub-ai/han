//go:build integration

package main

import (
	"testing"
	"time"

	"han/internal/identity"
	"han/internal/session"
)

// TestCase18RealSessionStoreResolveAndRevoke closes the C18-COV-F004 evidence
// gap at the store layer: it uses the real Postgres-backed session.Store (the
// same dependency set the production root wires) to resolve a live session and
// then to revoke it, proving the revoke path executes real SQL and that the
// revoked token no longer resolves. It deliberately does not replace the
// dispatcher-level checks already covered by TestRecoveryMainBusinessDispatcher.
func TestCase18RealSessionStoreResolveAndRevoke(t *testing.T) {
	world := s420NewCase18World(t)
	ctx := world.f.ctx

	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      world.f.pool,
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		t.Fatalf("real session store: %v", err)
	}

	principal, err := store.ResolvePrincipal(ctx, session.ResolveInput{
		Token:       world.sessionToken,
		CSRFToken:   world.csrfToken,
		RequireCSRF: true,
		Now:         world.f.now,
	})
	if err != nil {
		t.Fatalf("resolve live session: %v", err)
	}
	if principal.UserID != world.userID {
		t.Fatalf("principal user=%q, want %q", principal.UserID, world.userID)
	}

	if err := store.Revoke(ctx, world.f.pool, world.sessionToken, world.f.now.Add(time.Minute), "case18-f004-revoke"); err != nil {
		t.Fatalf("revoke session: %v", err)
	}

	if _, err := store.ResolvePrincipal(ctx, session.ResolveInput{
		Token:       world.sessionToken,
		CSRFToken:   world.csrfToken,
		RequireCSRF: true,
		Now:         world.f.now.Add(2 * time.Minute),
	}); err == nil {
		t.Fatal("revoked session still resolves")
	}
}

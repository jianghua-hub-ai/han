//go:build integration

package recovery_test

import "testing"

// TestRecoveryRuntimePostgres is the single top-level S4.17 acceptance test.
// Every leaf drives the real recovery runtime factory plus real PostgreSQL
// stores and the real session/authorizer/finalizer chain; only file/media ports
// use the controlled software adapters documented in runtime_fixtures_test.go.
func TestRecoveryRuntimePostgres(t *testing.T) {
	t.Run("GenesisDeletionHistoricalRestart", func(t *testing.T) {
		s417LeafGenesisDeletionHistoricalRestart(t)
	})
	t.Run("CrossNamespaceABAAndExactPublication", func(t *testing.T) {
		s417LeafCrossNamespaceABAAndExactPublication(t)
	})
	t.Run("F001F002AndCanonicalIdempotencyReview", func(t *testing.T) {
		s417LeafF001F002AndCanonicalIdempotencyReview(t)
	})
	t.Run("CrashConsumedWindowAndDrain", func(t *testing.T) {
		s417LeafCrashConsumedWindowAndDrain(t)
	})
	t.Run("Case18SameRevisionAndNoProductionClaim", func(t *testing.T) {
		s417LeafCase18SameRevisionAndNoProductionClaim(t)
	})
}

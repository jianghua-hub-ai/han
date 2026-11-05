//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/migration"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
)

// The B13H batch owns exactly one isolated root/backup pair. The harness
// refuses any other target so a stale B13C/B13D database can never pass as
// evidence for this chain.
const (
	recoveryChainRootDSN      = "postgres://han_b13h@127.0.0.1:53519/han_b13h_root?sslmode=disable"
	recoveryChainBackupDSN    = "postgres://han_b13h@127.0.0.1:53519/han_b13h_backup?sslmode=disable"
	recoveryChainMaxBody      = int64(65536)
	recoveryChainSessionToken = "b13h-chain-session"
	recoveryChainCSRFToken    = "b13h-chain-csrf"
)

type recoveryChainPG struct {
	ctx    context.Context
	config *pgxpool.Config
	pool   *pgxpool.Pool
	schema string
}

func recoveryChainOpen(t *testing.T, ctx context.Context, environment, fallback, database string) *recoveryChainPG {
	t.Helper()
	dsn := fallback
	if override := strings.TrimSpace(os.Getenv(environment)); override != "" {
		dsn = override
	}
	config, err := parseDatabasePoolConfig(dsn, func(string) string { return "" })
	if err != nil {
		t.Fatalf("parse %s DSN: %v", database, err)
	}
	if config.ConnConfig.Host != "127.0.0.1" || config.ConnConfig.Port != 53519 || config.ConnConfig.Database != database {
		t.Fatalf("explicit authorized B13H %s required, got %s:%d/%s", database, config.ConnConfig.Host, config.ConnConfig.Port, config.ConnConfig.Database)
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal("schema nonce")
	}
	schema := "han_b13h_chain_" + hex.EncodeToString(nonce)
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open %s: %v", database, err)
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create %s schema: %v", database, err)
	}
	harness := &recoveryChainPG{ctx: ctx, config: config, pool: pool, schema: schema}
	harness.migrate(t)
	t.Logf("%s_schema=%s", database, schema)
	return harness
}

func (h *recoveryChainPG) migrate(t *testing.T) {
	t.Helper()
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql"}
	files := fstest.MapFS{}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal("read approved migration")
		}
		files["approved/"+name] = &fstest.MapFile{Data: raw}
	}
	runner, err := migration.NewRunner(h.pool, files, "approved")
	if err != nil {
		t.Fatal("migration constructor")
	}
	if err := runner.Run(h.ctx); err != nil {
		t.Fatalf("approved migration failed: %v", err)
	}
}

// recoveryChainResolver is the authorization seam. The production session
// resolver is PostgreSQL-backed; this chain test isolates the ordering proof,
// so the resolver returns one active administrator and enforces CSRF exactly
// as the production guard does.
type recoveryChainResolver struct {
	principal session.Principal
}

func (r recoveryChainResolver) ResolvePrincipal(_ context.Context, input session.ResolveInput) (session.Principal, error) {
	if input.RequireCSRF && input.CSRFToken != recoveryChainCSRFToken {
		return session.Principal{}, session.ErrCSRFFailed
	}
	if r.principal.UserID == "" {
		return session.Principal{}, session.ErrSessionRequired
	}
	return r.principal, nil
}

type recoveryChainMaintenance struct{}

func (recoveryChainMaintenance) ResolveRestrictedAccountReview(context.Context, recovery.PhasePermit, recovery.RunBinding, string, string, string) error {
	return recovery.ErrAdmissionUnsupported
}

func (recoveryChainMaintenance) ResolveRecoveryIdempotency(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte, []byte, int64, string, string) error {
	return recovery.ErrAdmissionUnsupported
}

type recoveryChainTrust struct{}

func (recoveryChainTrust) ChangeRecoveryPassword(context.Context, recovery.PhasePermit, identity.PasswordChangeRequest) (identity.PasswordChangeResult, error) {
	return identity.PasswordChangeResult{}, recovery.ErrAdmissionUnsupported
}

// recoveryChainExit blocks inside the real join so the test can prove revocation
// only happens after every HTTP, password, settlement, transaction and child
// call has actually joined.
type recoveryChainExit struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	onExit  func()
}

func newRecoveryChainExit(onExit func()) *recoveryChainExit {
	return &recoveryChainExit{entered: make(chan struct{}), release: make(chan struct{}), onExit: onExit}
}

func (x *recoveryChainExit) ExitRecovery(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	x.once.Do(func() {
		if x.onExit != nil {
			x.onExit()
		}
		close(x.entered)
	})
	<-x.release
	return nil
}

func TestRecoveryRuntimeProductionChain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Open the single authorized B13H root/backup pair. The backup assertion
	// binds the harness to the isolated pair even though only root is mutated.
	root := recoveryChainOpen(t, ctx, "HAN_B13H_ROOT_DSN", recoveryChainRootDSN, "han_b13h_root")
	recoveryChainOpen(t, ctx, "HAN_B13H_BACKUP_DSN", recoveryChainBackupDSN, "han_b13h_backup")

	// Real session store and real review lister against the migrated schema.
	gate := &identity.PasswordResetStore{Idempotency: idempotency.Store{}}
	if _, err := session.NewStore(session.StoreDependencies{Transactions: root.pool, PasswordResetGate: gate}); err != nil {
		t.Fatalf("create real session store: %v", err)
	}

	binding := recoveryTestPermitBinding()
	runtime := &recoveryTestRuntime{binding: binding}
	executor := &recoveryTestExecutor{}
	lister, err := identity.NewRecoveryReviewLister(root.pool, runtime)
	if err != nil {
		t.Fatalf("create real review lister: %v", err)
	}

	var mu sync.Mutex
	var order []string
	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, name)
	}
	executor.onCall = func(name string) { record(name) }

	exit := newRecoveryChainExit(func() { record("exit") })
	article, err := newRecoveryControlHandle(
		func() error { record("close"); return nil },
		func(context.Context) error { record("join"); return nil },
		func() string { return "installed" },
	)
	if err != nil {
		t.Fatalf("build article control handle: %v", err)
	}
	secret, err := newRecoverySecretSource(func(context.Context, recovery.RunBinding) ([]byte, error) {
		return []byte("candidate-secret"), nil
	})
	if err != nil {
		t.Fatalf("build secret source: %v", err)
	}

	composition, err := newRecoveryRuntimeComposition(recoveryRuntimePorts{
		Runtime: runtime, Identity: executor, Exit: exit, Secret: secret,
		Lifecycle: recoveryTestLifecycle{}, Article: article, JoinBudget: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("build composition: %v", err)
	}
	bridge := composition.bridgeValue()
	if bridge == nil {
		t.Fatal("composition has no identity bridge")
	}

	authorizer, err := platform.NewHTTPAuthorizer(
		recoveryChainResolver{principal: session.Principal{UserID: "30000000-0000-4000-8000-000000000001", Role: string(identity.RoleAdmin), Status: string(identity.UserStatusActive)}},
		func() time.Time { return time.Unix(1790000000, 0).UTC() },
	)
	if err != nil {
		t.Fatalf("build authorizer: %v", err)
	}
	handlers, err := identity.NewRecoveryControlHandlers(identity.RecoveryControlDependencies{
		Maintenance:  recoveryChainMaintenance{},
		Reviews:      lister,
		TrustChange:  recoveryChainTrust{},
		Complete:     bridge,
		Authorizer:   authorizer,
		Phase:        composition.controlPhase,
		Session:      func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusNoContent) },
		Liveness:     func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
		Readiness:    func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
		MaxBodyBytes: recoveryChainMaxBody,
	})
	if err != nil {
		t.Fatalf("build real handler: %v", err)
	}

	run := recovery.RunBinding{RunID: binding.RunID, BackupRef: binding.BackupRef}

	// The runtime flow observes the identity permit and then blocks until the
	// root controller has delegated the real completion.
	flowDone := make(chan error, 1)
	go func() { flowDone <- bridge.CompleteIdentityReview(ctx, recovery.PhasePermit{}, run) }()
	if err := composition.waitIdentityReady(ctx); err != nil {
		t.Fatalf("identity permit was not captured: %v", err)
	}
	if executor.count("complete") != 0 {
		t.Fatal("real completion ran before the operator confirmation")
	}

	// The maintenance request derives a bounded phase context from the retained
	// identity permit; a bare runtime.PhaseContext call would carry no purpose.
	reviewCtx, _, reviewBinding, err := composition.controlPhase(ctx)
	if err != nil {
		t.Fatalf("derive review phase context: %v", err)
	}
	if reviewBinding.RunID != run.RunID || reviewBinding.BackupRef != run.BackupRef {
		t.Fatalf("review phase binding = %+v, want %+v", reviewBinding, run)
	}
	if _, _, err := lister.ListPendingReviews(reviewCtx, "", 10); err != nil {
		t.Fatalf("real review query failed: %v", err)
	}

	// Cursor hardening: a cursor that reuses the live run/installation binding
	// but carries an empty or non-UUID last_user_id must fail closed as
	// request.invalid_cursor instead of reaching the uuid column parameter.
	for _, invalidLast := range []string{"", "not-a-uuid"} {
		cursorJSON, err := json.Marshal(struct {
			RunID          string `json:"run_id"`
			InstallationID string `json:"installation_id"`
			LastUserID     string `json:"last_user_id"`
		}{RunID: binding.RunID, InstallationID: binding.InstallationID, LastUserID: invalidLast})
		if err != nil {
			t.Fatalf("encode cursor: %v", err)
		}
		cursorRequest := httptest.NewRequest(http.MethodGet, "/api/v1/recovery/reviews?limit=10&cursor="+base64.RawURLEncoding.EncodeToString(cursorJSON), nil)
		cursorRequest.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: recoveryChainSessionToken})
		cursorRequest.Header.Set(platform.CSRFHeaderName, recoveryChainCSRFToken)
		cursorRecorder := httptest.NewRecorder()
		handlers.Handler().ServeHTTP(cursorRecorder, cursorRequest)
		if cursorRecorder.Code != http.StatusBadRequest {
			t.Fatalf("cursor %q status = %d body=%q", invalidLast, cursorRecorder.Code, cursorRecorder.Body.String())
		}
		var cursorEnvelope struct {
			Code platform.Code `json:"code"`
		}
		if err := json.Unmarshal(cursorRecorder.Body.Bytes(), &cursorEnvelope); err != nil || cursorEnvelope.Code != platform.CodeRequestInvalidCursor {
			t.Fatalf("cursor %q body = %q err=%v", invalidLast, cursorRecorder.Body.String(), err)
		}
	}

	// HTTP complete: validate, write 200 identity_ready, flush, then signal.
	request := httptest.NewRequest(http.MethodPost, "/api/v1/recovery/complete", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: recoveryChainSessionToken})
	request.Header.Set(platform.CSRFHeaderName, recoveryChainCSRFToken)
	recorder := httptest.NewRecorder()
	handlers.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete status = %d body=%q", recorder.Code, recorder.Body.String())
	}
	var accepted struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &accepted); err != nil || accepted.Status != "identity_ready" {
		t.Fatalf("complete body = %q err=%v", recorder.Body.String(), err)
	}
	if executor.count("complete") != 0 {
		t.Fatal("real completion ran inside the HTTP stack")
	}

	// Root delegates the real completion only after the HTTP response returned.
	if err := composition.completeIdentity(ctx); err != nil {
		t.Fatalf("root completion failed: %v", err)
	}
	if executor.count("complete") != 1 {
		t.Fatalf("complete calls = %d", executor.count("complete"))
	}
	select {
	case err := <-flowDone:
		if err != nil {
			t.Fatalf("blocked flow returned %v", err)
		}
	case <-ctx.Done():
		t.Fatal("blocked flow did not resume after completion")
	}

	// Exit chain: revocation must wait for every real join to return.
	exitDone := make(chan error, 1)
	go func() { exitDone <- composition.ExitRecovery(ctx, recovery.PhasePermit{}, run) }()
	select {
	case <-exit.entered:
	case <-ctx.Done():
		t.Fatal("exit chain never joined")
	}
	if executor.count("revoke") != 0 {
		t.Fatal("revocation ran before the joins completed")
	}
	close(exit.release)
	select {
	case err := <-exitDone:
		if err != nil {
			t.Fatalf("exit chain failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("exit chain did not finish")
	}
	if executor.count("revoke") != 1 {
		t.Fatalf("revoke calls = %d", executor.count("revoke"))
	}

	mu.Lock()
	observed := append([]string(nil), order...)
	mu.Unlock()
	want := []string{"complete", "exit", "revoke", "close", "join"}
	if len(observed) != len(want) {
		t.Fatalf("chain order = %v, want %v", observed, want)
	}
	for i := range want {
		if observed[i] != want[i] {
			t.Fatalf("chain order = %v, want %v", observed, want)
		}
	}
	if article.State() != "installed" {
		t.Fatalf("article state = %q", article.State())
	}
}

// recoveryChainMaintenanceAuthority is a test-only session authority. Production
// supplies the sealed executor authority through
// identity.NewRecoveryMaintenanceStore; this double records the binding the
// maintenance store read so the test can prove which path resolved the session.
type recoveryChainMaintenanceAuthority struct {
	mu      sync.Mutex
	user    string
	binding session.RecoverySessionBinding
	calls   int
}

func (a *recoveryChainMaintenanceAuthority) CheckLockedMaintenanceUser(_ context.Context, _ pgx.Tx, user string, binding session.RecoverySessionBinding) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	a.user = user
	a.binding = binding
	return nil
}

func (a *recoveryChainMaintenanceAuthority) observed() (string, session.RecoverySessionBinding, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.user, a.binding, a.calls
}

// recoveryChainControlAuthorizer mirrors the production control wiring: the
// maintenance surface resolves through session.MaintenanceStore, never through
// the ordinary store whose SQL requires access_scope='ordinary'.
func recoveryChainControlAuthorizer(pool *pgxpool.Pool, gate session.PasswordResetGate, authority session.MaintenanceSessionAuthority, clock platform.Clock) (platform.HTTPAuthorizer, error) {
	store, err := session.NewMaintenanceStore(session.MaintenanceStoreDependencies{
		Transactions:      pool,
		PasswordResetGate: gate,
		Authority:         authority,
	})
	if err != nil {
		return platform.HTTPAuthorizer{}, err
	}
	return platform.NewHTTPAuthorizer(store, clock)
}

// TestRecoveryMaintenanceAuthorizerScope pins finding F002: a successful
// maintenance login session must pass the protected control surface, while the
// ordinary store (the pre-fix wiring) rejects the same recovery_maintenance
// session because its resolver SQL only accepts access_scope='ordinary'.
func TestRecoveryMaintenanceAuthorizerScope(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := recoveryChainOpen(t, ctx, "HAN_B13H_ROOT_DSN", recoveryChainRootDSN, "han_b13h_root")

	const (
		userID         = "30000000-0000-4000-8000-0000000000c1"
		sessionID      = "30000000-0000-4000-8000-0000000000c2"
		token          = "b13h-maintenance-authorizer-session"
		csrfToken      = "b13h-maintenance-authorizer-csrf"
		runID          = "b13h-authorizer-run"
		installationID = "b13h-authorizer-installation"
		runtimeID      = "b13h-authorizer-runtime"
		generation     = int64(3)
	)
	digest := func(value string) []byte {
		sum := sha256.Sum256([]byte(value))
		return sum[:]
	}
	now := time.Now().UTC()
	expiry := now.Add(time.Hour)
	statements := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO recovery_installation (singleton, metadata_version, deployment_id, installation_id,
	target_digest, commitment_ref, commitment_digest, recovery_journal_id, article_journal_id,
	recovery_epoch, article_epoch, recovery_initial_sequence, article_initial_sequence,
	recovery_initial_hash, article_initial_hash, revision, installed_at)
	VALUES (true, 'han-recovery-install/v1', 'b13h-authorizer-deployment', $1,
	'sha256:'||repeat('a', 64), 'b13h-commitment', 'sha256:'||repeat('b', 64),
	'b13h-recovery-journal', 'b13h-article-journal', 1, 1, 0, 0, '', '', 1, $2)`,
			args: []any{installationID, now},
		},
		{
			sql: `INSERT INTO users (id, username, role, is_initial_admin, status, credential_mode,
	recovery_required, permanent_password_digest, auth_generation, permanent_password_generation,
	created_at, updated_at)
	VALUES ($1, 'b13h-maintenance-authorizer', 'admin', true, 'active', 'recovery_bootstrap',
	true, 'sha256:'||repeat('c', 64), $2, $2, $3, $3)`,
			args: []any{userID, generation, now},
		},
		{
			sql: `INSERT INTO recovery_runs (run_id, backup_ref, journal_key_id, journal_key_version,
	journal_secret_fingerprint, journal_stage, started_at, outcome, initial_admin_user_id,
	initial_admin_bootstrap_generation)
	VALUES ($1, 'b13h-authorizer-backup', 'b13h-key', 'v1', 'b13h-fingerprint', 'begun', $2,
	'running', $3, $4)`,
			args: []any{runID, now, userID, generation},
		},
		{
			sql: `INSERT INTO sessions (id, user_id, token_digest, csrf_token_digest,
	authenticated_credential_kind, authenticated_credential_id, auth_generation,
	lineage_operation_id, lineage_credential_id, issued_at, last_seen_at, absolute_expires_at,
	idle_expires_at, rotated_from_id, access_scope, recovery_run_id, recovery_installation_id,
	recovery_runtime_id, recovery_generation)
	VALUES ($1, $2, $3, $4, 'recovery_bootstrap', NULL, $5, NULL, NULL, $6, $6, $7, $7, NULL,
	'recovery_maintenance', $8, $9, $10, $5)`,
			args: []any{sessionID, userID, digest(token), digest(csrfToken), generation, now, expiry, runID, installationID, runtimeID},
		},
	}
	for index, statement := range statements {
		if _, err := root.pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("seed maintenance authorizer row %d: %v", index, err)
		}
	}

	gate := identity.PasswordResetStore{}
	clock := func() time.Time { return now }
	ordinaryStore, err := session.NewStore(session.StoreDependencies{Transactions: root.pool, PasswordResetGate: gate})
	if err != nil {
		t.Fatalf("ordinary store: %v", err)
	}
	ordinaryAuthorizer, err := platform.NewHTTPAuthorizer(ordinaryStore, clock)
	if err != nil {
		t.Fatalf("ordinary authorizer: %v", err)
	}
	authority := &recoveryChainMaintenanceAuthority{}
	controlAuthorizer, err := recoveryChainControlAuthorizer(root.pool, gate, authority, clock)
	if err != nil {
		t.Fatalf("control authorizer: %v", err)
	}

	probe := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	request := func(method, csrf string) *http.Request {
		value := httptest.NewRequest(method, "/api/v1/recovery/session", nil)
		phaseCtx, phaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(phaseCancel)
		value = value.WithContext(phaseCtx)
		value.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
		if csrf != "" {
			value.Header.Set(platform.CSRFHeaderName, csrf)
		}
		return value
	}

	t.Run("ordinary_store_rejects_maintenance_scope", func(t *testing.T) {
		response := httptest.NewRecorder()
		ordinaryAuthorizer.Protect(session.AccessCurrentIdentity, probe).ServeHTTP(response, request(http.MethodGet, ""))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("ordinary authorizer status = %d, want 401", response.Code)
		}
		if _, _, calls := authority.observed(); calls != 0 {
			t.Fatalf("maintenance authority consulted on the ordinary path: %d", calls)
		}
	})

	t.Run("maintenance_store_accepts_maintenance_session", func(t *testing.T) {
		response := httptest.NewRecorder()
		controlAuthorizer.Protect(session.AccessCurrentIdentity, probe).ServeHTTP(response, request(http.MethodGet, ""))
		if response.Code != http.StatusNoContent {
			t.Fatalf("maintenance authorizer status = %d, want 204", response.Code)
		}
		user, binding, calls := authority.observed()
		if calls != 1 || user != userID {
			t.Fatalf("maintenance authority calls=%d user=%q", calls, user)
		}
		want := session.RecoverySessionBinding{RunID: runID, InstallationID: installationID, RuntimeID: runtimeID, Generation: generation}
		if binding != want {
			t.Fatalf("maintenance binding = %+v, want %+v", binding, want)
		}
	})

	t.Run("maintenance_store_requires_csrf_on_write", func(t *testing.T) {
		response := httptest.NewRecorder()
		controlAuthorizer.Protect(session.AccessCurrentIdentity, probe).ServeHTTP(response, request(http.MethodPost, ""))
		if response.Code != http.StatusForbidden {
			t.Fatalf("maintenance authorizer without CSRF status = %d, want 403", response.Code)
		}
		response = httptest.NewRecorder()
		controlAuthorizer.Protect(session.AccessCurrentIdentity, probe).ServeHTTP(response, request(http.MethodPost, csrfToken))
		if response.Code != http.StatusNoContent {
			t.Fatalf("maintenance authorizer with CSRF status = %d, want 204", response.Code)
		}
	})
}

// TestRecoveryControlAuthorizerSealedMaintenance pins F010: the production
// factory newRecoveryControlAuthorizer must unseal the forwarder into the real
// *recoveryIdentityExecution and the real CheckLockedMaintenanceUser must
// authorize a recovery_maintenance session. Unlike
// TestRecoveryMaintenanceAuthorizerScope this test substitutes no maintenance
// authority: the seeded rows are the same installation, run, restore receipt,
// finalization and trust audit the identity authority verifies in production.
func TestRecoveryControlAuthorizerSealedMaintenance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := recoveryChainOpen(t, ctx, "HAN_B13H_ROOT_DSN", recoveryChainRootDSN, "han_b13h_root")

	const (
		userID         = "40000000-0000-4000-8000-0000000000d1"
		sessionID      = "40000000-0000-4000-8000-0000000000d2"
		finalizeAudit  = "40000000-0000-4000-8000-0000000000d3"
		trustAudit     = "40000000-0000-4000-8000-0000000000d4"
		deploymentID   = "b13h-sealed-authorizer-deployment"
		installationID = "b13h-sealed-authorizer-installation"
		runtimeID      = "b13h-sealed-authorizer-runtime"
		runID          = "b13h-sealed-authorizer-run"
		backupRef      = "b13h-sealed-authorizer-backup"
		traceID        = "b13h-sealed-authorizer-trace"
		token          = "b13h-sealed-authorizer-session"
		csrfToken      = "b13h-sealed-authorizer-csrf"
		generation     = int64(3)
	)
	repeat := func(c byte) string { return strings.Repeat(string(c), 64) }
	sha := func(c byte) string { return "sha256:" + repeat(c) }
	targetDigest := sha('a')
	artifactDigest := sha('b')
	receiptDigest := sha('c')
	approvalDigest := sha('d')
	schemaDigest := sha('e')
	fingerprint := "hmac-sha256:" + repeat('f')
	now := time.Now().UTC().Truncate(time.Microsecond)
	expiry := now.Add(time.Hour)
	digest := func(value string) []byte {
		sum := sha256.Sum256([]byte(value))
		return sum[:]
	}

	// The finalization audit metadata must decode to the exact projection the
	// authority re-reads: one initial administrator, empty idempotency cohort,
	// and counts identical to the recovery_runs row.
	finalizeMetadata := map[string]any{
		"run_id":          runID,
		"backup_ref":      backupRef,
		"installation_id": installationID,
		"sources": []any{map[string]any{
			"ID": userID, "Role": "admin", "Status": "active", "DeletedAt": nil, "Initial": true,
		}},
		"idempotency_cohort":       []any{},
		"idempotency_cohort_count": "0",
		"accounts":                 1,
		"permanent":                1,
		"sessions":                 1,
		"temporary":                0,
		"idempotency":              0,
		"reviews":                  0,
	}
	trustMetadata := map[string]any{
		"run_id": runID, "backup_ref": backupRef, "installation_id": installationID,
		"initial_admin_user_id": userID, "bootstrap_generation": generation,
	}

	statements := []struct {
		sql  string
		args []any
	}{
		{
			sql: `INSERT INTO recovery_installation (singleton, metadata_version, deployment_id, installation_id,
	target_digest, commitment_ref, commitment_digest, recovery_journal_id, article_journal_id,
	recovery_epoch, article_epoch, recovery_initial_sequence, article_initial_sequence,
	recovery_initial_hash, article_initial_hash, revision, installed_at)
	VALUES (true, 'han-recovery-install/v1', $1, $2,
	$3, 'b13h-sealed-commitment', $4,
	'b13h-recovery-journal', 'b13h-article-journal', 1, 1, 0, 0, '', '', 1, $5)`,
			args: []any{deploymentID, installationID, targetDigest, sha('9'), now},
		},
		{
			sql: `INSERT INTO users (id, username, role, is_initial_admin, status, credential_mode,
	recovery_required, permanent_password_digest, auth_generation, permanent_password_generation,
	created_at, updated_at)
	VALUES ($1, 'b13h-sealed-authorizer', 'admin', true, 'active', 'recovery_bootstrap',
	true, $2, $3, $3, $4, $4)`,
			args: []any{userID, sha('7'), generation, now},
		},
		{
			sql: `INSERT INTO audit_events (id, occurred_at, actor_kind, actor_system, action, target_type,
	target_id, outcome, trace_id, metadata)
	VALUES ($1, $2, 'system', 'recovery_runtime', 'identity.recovery.finalize', 'recovery_run', $3,
	'succeeded', $4, $5)`,
			args: []any{finalizeAudit, now, runID, traceID, finalizeMetadata},
		},
		{
			sql: `INSERT INTO audit_events (id, occurred_at, actor_kind, actor_system, action, target_type,
	target_id, outcome, trace_id, metadata)
	VALUES ($1, $2, 'system', 'recovery_runtime', 'identity.recovery.trust.prepare', 'recovery_run', $3,
	'succeeded', $4, $5)`,
			args: []any{trustAudit, now, runID, traceID, trustMetadata},
		},
		{
			sql: `INSERT INTO recovery_runs (run_id, backup_ref, journal_key_id, journal_key_version,
	journal_secret_fingerprint, journal_stage, started_at, outcome, account_count,
	permanent_credential_count, session_count, temporary_credential_count, idempotency_count,
	pending_review_count, audit_event_id, initial_admin_user_id, initial_admin_bootstrap_generation)
	VALUES ($1, $2, 'b13h-key', 'v1', $3, 'begun', $4, 'running', 1, 1, 1, 0, 0, 0, $5, $6, $7)`,
			args: []any{runID, backupRef, fingerprint, now, finalizeAudit, userID, generation},
		},
		{
			sql: `INSERT INTO recovery_restore_bindings (run_id, backup_ref, installation_id, deployment_id,
	artifact_digest, target_digest, receipt_digest, source_approval_digest, restored_schema_digest,
	artifact_bytes, restored_at, receipt_version)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1024, $10, 'han-db-restore/v1')`,
			args: []any{runID, backupRef, installationID, deploymentID, artifactDigest, targetDigest, receiptDigest, approvalDigest, schemaDigest, now},
		},
		{
			sql: `INSERT INTO sessions (id, user_id, token_digest, csrf_token_digest,
	authenticated_credential_kind, authenticated_credential_id, auth_generation,
	lineage_operation_id, lineage_credential_id, issued_at, last_seen_at, absolute_expires_at,
	idle_expires_at, rotated_from_id, access_scope, recovery_run_id, recovery_installation_id,
	recovery_runtime_id, recovery_generation)
	VALUES ($1, $2, $3, $4, 'recovery_bootstrap', NULL, $5, NULL, NULL, $6, $6, $7, $7, NULL,
	'recovery_maintenance', $8, $9, $10, $5)`,
			args: []any{sessionID, userID, digest(token), digest(csrfToken), generation, now, expiry, runID, installationID, runtimeID},
		},
	}
	for index, statement := range statements {
		if _, err := root.pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatalf("seed sealed authorizer row %d: %v", index, err)
		}
	}

	store, err := identity.NewRecoveryRunStore(root.pool)
	if err != nil {
		t.Fatalf("recovery run store: %v", err)
	}
	runtime := &recoveryTestRuntime{binding: recovery.PermitBinding{
		Purpose:        recovery.PhaseRecoveryIdentity,
		RuntimeID:      runtimeID,
		Generation:     generation,
		RunID:          runID,
		BackupRef:      backupRef,
		InstallationID: installationID,
	}}
	execution, _, err := identity.NewRecoveryIdentityExecution(identity.RecoveryIdentityExecutionDependencies{
		Store:   store,
		Runtime: runtime,
		Journal: recoveryAssemblyJournal{},
		Articles: identity.ArticleAccountPortFunc(func(context.Context, pgx.Tx, string) (identity.CanonicalDeleteArticleResult, error) {
			return identity.CanonicalDeleteArticleResult{}, recovery.ErrAdmissionUnsupported
		}),
		Audit:     audit.Store{},
		Passwords: recoveryAssemblyHasher{},
		IDs:       mainLoginIDs{},
		Clock:     mainLoginClock{now: now},
	})
	if err != nil {
		t.Fatalf("sealed identity execution: %v", err)
	}
	forwarder := identity.NewRecoveryIdentityForwarder()
	if err := forwarder.Seal(execution); err != nil {
		t.Fatalf("seal identity forwarder: %v", err)
	}

	gate := &identity.PasswordResetStore{Idempotency: idempotency.Store{}}
	authorizer, err := newRecoveryControlAuthorizer(forwarder, gate, func() time.Time { return now })
	if err != nil {
		t.Fatalf("production control authorizer: %v", err)
	}

	probe := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/recovery/session", nil)
	phaseCtx, phaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(phaseCancel)
	request = request.WithContext(phaseCtx)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
	response := httptest.NewRecorder()
	authorizer.Protect(session.AccessCurrentIdentity, probe).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("sealed maintenance authorizer status = %d body=%q", response.Code, response.Body.String())
	}

	// The same production authorizer must still refuse when the live permit no
	// longer matches the session binding, so the previous 204 proves the real
	// CheckLockedMaintenanceUser ran instead of an unconditionally open path.
	runtime.mu.Lock()
	runtime.binding.Generation = generation + 1
	runtime.mu.Unlock()
	rejected := httptest.NewRecorder()
	rejectedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/recovery/session", nil)
	rejectedCtx, rejectedCancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(rejectedCancel)
	rejectedRequest = rejectedRequest.WithContext(rejectedCtx)
	rejectedRequest.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: token})
	authorizer.Protect(session.AccessCurrentIdentity, probe).ServeHTTP(rejected, rejectedRequest)
	if rejected.Code != http.StatusUnauthorized {
		t.Fatalf("mismatched permit binding status = %d, want 401", rejected.Code)
	}
	if runtime.reads == 0 {
		t.Fatal("sealed authority never re-read the live permit binding")
	}
}

//go:build integration

package identity

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/platform"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLoginCommitUnknownVerifiesActualPreparedAuditState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	adminPool := newIdentityTestPool(t, ctx)
	schema := createCommittedIdentitySchema(t, ctx, adminPool)
	pool := newLoginSchemaPool(t, ctx, schema)
	store := audit.Store{}
	auditor, err := NewPostgresLoginFailureAuditor(pool, store)
	if err != nil {
		t.Fatalf("NewPostgresLoginFailureAuditor() error = %v", err)
	}

	for _, test := range []struct {
		name            string
		commitActually  bool
		wantPersistence LoginAuditPersistence
		wantRows        int
	}{
		{
			name:            "database committed before commit error",
			commitActually:  true,
			wantPersistence: LoginAuditPersistedYes,
			wantRows:        1,
		},
		{
			name:            "database rolled back before commit error",
			wantPersistence: LoginAuditPersistedUnknown,
			wantRows:        0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			event := completeLoginAuditEvent(t)
			rawOccurredAt := time.Date(
				2026,
				time.October,
				22,
				20,
				0,
				0,
				123456789,
				time.FixedZone("synthetic-offset", 8*60*60),
			)
			coordinator, err := NewLoginAttemptCoordinator(
				&loginIDStub{ids: []string{event.ID}},
				loginClockStub{now: rawOccurredAt},
				auditor,
			)
			if err != nil {
				t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
			}
			attempt, err := coordinator.Begin()
			if err != nil {
				t.Fatalf("begin login attempt: %v", err)
			}
			if err := attempt.enterService(); err != nil {
				t.Fatalf("enter login service phase: %v", err)
			}
			wantOccurredAt := rawOccurredAt.UTC().Truncate(time.Microsecond)
			if !attempt.OccurredAt.Equal(wantOccurredAt) ||
				attempt.OccurredAt.Location() != time.UTC {
				t.Fatalf(
					"attempt occurred_at = %s, want normalized %s",
					attempt.OccurredAt,
					wantOccurredAt,
				)
			}
			event.OccurredAt = attempt.OccurredAt
			attempt.setPrepared(event)

			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin prepared audit transaction: %v", err)
			}
			if err := store.Append(ctx, tx, event); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatalf("append prepared audit event: %v", err)
			}
			fault := &commitUnknownTx{
				Tx:             tx,
				commitActually: test.commitActually,
			}
			if err := fault.Commit(ctx); !errors.Is(err, errSyntheticCommitUnknown) {
				t.Fatalf("fault Commit() error = %v, want synthetic uncertainty", err)
			}

			cause := loginPersistenceError(
				"commit login transaction",
				errSyntheticCommitUnknown,
			)
			if got := (&Service{}).finishUnknownCommit(ctx, attempt, cause); got != cause {
				t.Fatal("finishUnknownCommit() changed the original public cause")
			}
			observation := attempt.observationSnapshot()
			if observation.FailureStage != LoginFailureStageTransactionCommit ||
				observation.TransactionOutcome != LoginTransactionUnknown ||
				observation.StateChanged != LoginStateUnknown ||
				observation.AuditPersisted != test.wantPersistence ||
				observation.RollbackOutcome != LoginRollbackNotAttempted {
				t.Fatalf("observation = %#v", observation)
			}

			var rows int
			if err := pool.QueryRow(
				ctx,
				"SELECT count(*) FROM audit_events WHERE id = $1",
				event.ID,
			).Scan(&rows); err != nil {
				t.Fatalf("count prepared audit events: %v", err)
			}
			if rows != test.wantRows {
				t.Fatalf("prepared audit rows = %d, want %d", rows, test.wantRows)
			}
			if rows == 1 {
				got, err := store.Get(ctx, pool, event.ID)
				if err != nil {
					t.Fatalf("read prepared audit event: %v", err)
				}
				if !sameAuditEvent(got, event) {
					t.Fatal("commit verification mutated the stored audit event")
				}
			}
		})
	}
}

func TestLoginRollbackUnknownAfterAppendVerifiesActualPreparedAuditState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	adminPool := newIdentityTestPool(t, ctx)
	schema := createCommittedIdentitySchema(t, ctx, adminPool)
	pool := newLoginSchemaPool(t, ctx, schema)
	store := audit.Store{}
	auditor, err := NewPostgresLoginFailureAuditor(pool, store)
	if err != nil {
		t.Fatalf("NewPostgresLoginFailureAuditor() error = %v", err)
	}

	for _, test := range []struct {
		name            string
		rollbackCommits bool
		mutateStored    func(*audit.Event)
		wantPersistence LoginAuditPersistence
		wantRows        int
	}{
		{
			name:            "matching row committed before rollback error",
			rollbackCommits: true,
			wantPersistence: LoginAuditPersistedYes,
			wantRows:        1,
		},
		{
			name:            "no row after rollback error",
			wantPersistence: LoginAuditPersistedUnknown,
		},
		{
			name:            "different row committed before rollback error",
			rollbackCommits: true,
			mutateStored: func(event *audit.Event) {
				event.Outcome = loginAuditOutcomeSucceeded
			},
			wantPersistence: LoginAuditPersistedUnknown,
			wantRows:        1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			prepared := completeLoginAuditEvent(t)
			coordinator, err := NewLoginAttemptCoordinator(
				&loginIDStub{ids: []string{prepared.ID}},
				loginClockStub{now: prepared.OccurredAt},
				auditor,
			)
			if err != nil {
				t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
			}
			attempt, err := coordinator.Begin()
			if err != nil {
				t.Fatalf("begin login attempt: %v", err)
			}
			if err := attempt.enterService(); err != nil {
				t.Fatalf("enter login service phase: %v", err)
			}
			prepared.OccurredAt = attempt.OccurredAt
			attempt.setPrepared(prepared)

			stored := cloneAuditEvent(prepared)
			if test.mutateStored != nil {
				test.mutateStored(&stored)
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin prepared audit transaction: %v", err)
			}
			if err := store.Append(ctx, tx, stored); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatalf("append prepared audit event: %v", err)
			}

			cause := loginPersistenceError(
				"append login audit event",
				errors.New("synthetic append result unknown"),
			)
			fault := &commitUnknownTx{
				Tx:              tx,
				rollbackCommits: test.rollbackCommits,
			}
			if got := (&Service{}).failLoginTransaction(
				ctx,
				fault,
				attempt,
				prepared.TargetID,
				"",
				LoginFailureStageAuditAppend,
				cause,
			); got != cause {
				t.Fatal("failLoginTransaction() changed the original public cause")
			}

			observation := attempt.observationSnapshot()
			if observation.FailureStage != LoginFailureStageAuditAppend ||
				observation.TransactionOutcome != LoginTransactionUnknown ||
				observation.StateChanged != LoginStateUnknown ||
				observation.AuditPersisted != test.wantPersistence ||
				observation.RollbackOutcome != LoginRollbackUnknown {
				t.Fatalf("observation = %#v", observation)
			}

			var rows int
			if err := pool.QueryRow(
				ctx,
				"SELECT count(*) FROM audit_events WHERE id = $1",
				prepared.ID,
			).Scan(&rows); err != nil {
				t.Fatalf("count prepared audit events: %v", err)
			}
			if rows != test.wantRows {
				t.Fatalf("prepared audit rows = %d, want %d", rows, test.wantRows)
			}
			if rows == 1 {
				got, err := store.Get(ctx, pool, prepared.ID)
				if err != nil {
					t.Fatalf("read prepared audit event: %v", err)
				}
				if !sameAuditEvent(got, stored) {
					t.Fatal("verify_prepared mutated the stored audit event")
				}
			}
		})
	}
}

func TestLoginPreparedAuditPostgresRoundTripPreservesExactImmutableFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	adminPool := newIdentityTestPool(t, ctx)
	schema := createCommittedIdentitySchema(t, ctx, adminPool)
	pool := newLoginSchemaPool(t, ctx, schema)
	store := audit.Store{}

	empty := ""
	for _, test := range []struct {
		name   string
		mutate func(*audit.Event)
	}{
		{name: "all immutable fields"},
		{
			name: "nil optional fields and metadata",
			mutate: func(event *audit.Event) {
				event.ActorKind = audit.ActorAnonymous
				event.ActorUserID = nil
				event.ActorSystem = nil
				event.ActorRole = nil
				event.ErrorCode = nil
				event.SourceFingerprint = nil
				event.Metadata = nil
			},
		},
		{
			name: "empty optional text fields and metadata",
			mutate: func(event *audit.Event) {
				event.ActorRole = &empty
				event.ErrorCode = &empty
				event.SourceFingerprint = &empty
				event.Metadata = map[string]any{}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			event := completeLoginAuditEvent(t)
			event.OccurredAt = event.OccurredAt.UTC().Truncate(time.Microsecond)
			if test.mutate != nil {
				test.mutate(&event)
			}

			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin audit round-trip transaction: %v", err)
			}
			if err := store.Append(ctx, tx, event); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatalf("append audit round-trip event: %v", err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("commit audit round-trip event: %v", err)
			}

			got, err := store.Get(ctx, pool, event.ID)
			if err != nil {
				t.Fatalf("read audit round-trip event: %v", err)
			}
			if !sameAuditEvent(got, event) {
				t.Fatal("PostgreSQL round-trip changed an immutable audit field")
			}
			if got.OccurredAt.Nanosecond()%int(time.Microsecond) != 0 {
				t.Fatalf(
					"stored occurred_at nanoseconds = %d, want whole microseconds",
					got.OccurredAt.Nanosecond(),
				)
			}
		})
	}
}

func TestLoginUserRowLockSerializesConcurrentFailures(t *testing.T) {
	if !strings.Contains(findLoginUserSQL, "FOR UPDATE OF u") {
		t.Fatal("login user query does not lock the selected users row")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool := newIdentityTestPool(t, ctx)
	schema := createCommittedIdentitySchema(t, ctx, pool)
	quotedSchema := pgx.Identifier{schema}.Sanitize()

	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	userID := randomUUID(t)
	const (
		username          = "row-lock-user"
		oldPassword       = "SyntheticOldPassword123!"
		oldDigest         = "synthetic-old-password-digest"
		replacementDigest = "synthetic-replacement-password-digest"
		fakeProbe         = "synthetic-fake-credential-probe"
		fakeDigest        = "synthetic-fake-credential-digest"
		oldSessionToken   = "synthetic-old-session-token"
		revocationReason  = "credential_status_change"
	)

	passwordResetGate := &PasswordResetStore{Idempotency: idempotency.Store{}}
	sessionStore, err := session.NewStore(session.StoreDependencies{
		Transactions:      pool,
		PasswordResetGate: passwordResetGate,
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	setupTx := beginSchemaTransaction(t, ctx, pool, quotedSchema)
	insertUser(
		t,
		ctx,
		setupTx,
		userID,
		username,
		string(RoleMember),
		false,
		string(UserStatusActive),
		string(CredentialModePermanent),
		now,
	)
	if _, err := setupTx.Exec(
		ctx,
		"UPDATE users SET permanent_password_digest = $1, permanent_password_generation = 1, auth_generation = 1 WHERE id = $2",
		oldDigest,
		userID,
	); err != nil {
		t.Fatalf("set initial password digest: %v", err)
	}
	insertIdentifier(
		t,
		ctx,
		setupTx,
		userID,
		string(IdentifierUsername),
		username,
		string(IdentifierActive),
		now,
	)
	if _, err := sessionStore.Create(ctx, setupTx, session.CreateInput{
		ID:                          randomUUID(t),
		UserID:                      userID,
		Token:                       oldSessionToken,
		CSRFToken:                   "synthetic-old-csrf-token",
		AuthenticatedCredentialKind: session.CredentialPermanent,
		IssuedAt:                    now,
		AbsoluteExpiresAt:           now.Add(8 * time.Hour),
		IdleExpiresAt:               now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create initial session: %v", err)
	}
	if err := setupTx.Commit(ctx); err != nil {
		t.Fatalf("commit identity setup: %v", err)
	}

	updateTx := beginSchemaTransaction(t, ctx, pool, quotedSchema)
	updateClosed := false
	t.Cleanup(func() {
		if !updateClosed {
			_ = updateTx.Rollback(context.Background())
		}
	})
	var updatePID int
	if err := updateTx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&updatePID); err != nil {
		t.Fatalf("read concurrent update backend PID: %v", err)
	}
	if _, err := updateTx.Exec(
		ctx,
		`UPDATE users
		 SET permanent_password_digest = $1,
		     permanent_password_generation = 2,
		     auth_generation = 2,
		     status = $2,
		     updated_at = $3
		 WHERE id = $4`,
		replacementDigest,
		UserStatusSuspended,
		now.Add(time.Minute),
		userID,
	); err != nil {
		t.Fatalf("replace credential and suspend user: %v", err)
	}
	revokedSessions, err := (Repository{}).RevokeSessionsByUserID(
		ctx,
		updateTx,
		userID,
		now.Add(time.Minute),
		revocationReason,
	)
	if err != nil {
		t.Fatalf("revoke sessions during credential change: %v", err)
	}
	if revokedSessions != 1 {
		t.Fatalf("revoked sessions = %d, want 1", revokedSessions)
	}

	hasher := &loginRowLockHasher{
		oldPassword: oldPassword,
		oldDigest:   oldDigest,
		fakeProbe:   fakeProbe,
		fakeDigest:  fakeDigest,
	}
	transactor := &schemaLoginTransactor{
		pool:         pool,
		quotedSchema: quotedSchema,
		begun:        make(chan int, 1),
	}
	service, err := NewLoginService(LoginDependencies{
		Transactions:      transactor,
		Repository:        PostgresLoginRepository{},
		PasswordResetGate: passwordResetGate,
		Passwords:         hasher,
		Sessions:          sessionStore,
		SessionIssuer: loginSessionIssuerStub{
			now: now.Add(2 * time.Minute),
		},
		Abuse: &loginAbuseStub{decision: LoginAbuseDecision{
			SubjectDigest:    []byte("synthetic-subject-digest"),
			DigestKeyVersion: "synthetic-key-v1",
		}},
		RecoveryMode:         loginRecoveryModeStub{},
		Audit:                &loginAuditStub{},
		IDs:                  &loginIDStub{ids: []string{randomUUID(t)}},
		Clock:                loginClockStub{now: now.Add(2 * time.Minute)},
		FakeCredentialDigest: fakeDigest,
		FakeCredentialProbe:  fakeProbe,
	})
	if err != nil {
		t.Fatalf("NewLoginService() error = %v", err)
	}
	hasher.reset()
	coordinator, err := NewLoginAttemptCoordinator(
		&loginIDStub{ids: []string{randomUUID(t)}},
		loginClockStub{now: now.Add(2 * time.Minute)},
		&loginFailureAuditorStub{},
	)
	if err != nil {
		t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
	}
	loginAttempt, err := coordinator.Begin()
	if err != nil {
		t.Fatalf("begin login attempt: %v", err)
	}
	if err := loginAttempt.enterService(); err != nil {
		t.Fatalf("enter login service phase: %v", err)
	}

	loginCtx, err := platform.ContextWithTraceID(
		ctx,
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	type loginCall struct {
		result LoginResult
		err    error
	}
	loginCalls := make(chan loginCall, 1)
	go func() {
		result, loginErr := service.Login(loginCtx, LoginRequest{
			Username:             username,
			Password:             oldPassword,
			SourceFingerprint:    "synthetic-source-digest",
			ExistingSessionToken: oldSessionToken,
			Attempt:              loginAttempt,
		})
		loginCalls <- loginCall{result: result, err: loginErr}
	}()

	var loginPID int
	select {
	case loginPID = <-transactor.begun:
	case <-ctx.Done():
		t.Fatalf("wait for login transaction: %v", ctx.Err())
	}
	if loginPID == updatePID {
		t.Fatalf("login and concurrent update share backend PID %d", loginPID)
	}
	waitForPostgresBlocker(t, ctx, pool, updatePID, loginPID)
	if hasher.verified(oldDigest) {
		t.Fatal("login verified the stale password digest before the concurrent update committed")
	}

	if err := updateTx.Commit(ctx); err != nil {
		t.Fatalf("commit credential change and session revocation: %v", err)
	}
	updateClosed = true

	var call loginCall
	select {
	case call = <-loginCalls:
	case <-ctx.Done():
		t.Fatalf("wait for serialized login: %v", ctx.Err())
	}
	if got := platform.PublicCode(call.err); got != platform.CodeAuthLoginFailed {
		t.Fatalf("Login() public code = %q, want %q; result = %#v", got, platform.CodeAuthLoginFailed, call.result)
	}
	if hasher.verified(oldDigest) {
		t.Fatal("serialized login verified the stale password digest")
	}
	if hasher.hashCount() != 0 {
		t.Fatalf("serialized login password rehash count = %d, want 0", hasher.hashCount())
	}

	verifyTx := beginSchemaTransaction(t, ctx, pool, quotedSchema)
	defer func() { _ = verifyTx.Rollback(context.Background()) }()
	var (
		storedDigest string
		storedStatus UserStatus
	)
	if err := verifyTx.QueryRow(
		ctx,
		"SELECT permanent_password_digest, status FROM users WHERE id = $1",
		userID,
	).Scan(&storedDigest, &storedStatus); err != nil {
		t.Fatalf("read serialized user state: %v", err)
	}
	if storedDigest != replacementDigest || storedStatus != UserStatusSuspended {
		t.Fatalf(
			"serialized user state = digest %q status %q, want replacement digest and suspended",
			storedDigest,
			storedStatus,
		)
	}
	var (
		sessionCount       int
		activeSessionCount int
		storedReason       *string
	)
	if err := verifyTx.QueryRow(
		ctx,
		`SELECT
			count(*),
			count(*) FILTER (WHERE revoked_at IS NULL),
			max(revoke_reason)
		 FROM sessions
		 WHERE user_id = $1`,
		userID,
	).Scan(&sessionCount, &activeSessionCount, &storedReason); err != nil {
		t.Fatalf("read serialized session state: %v", err)
	}
	if sessionCount != 1 || activeSessionCount != 0 {
		t.Fatalf(
			"serialized sessions = total %d active %d, want total 1 active 0",
			sessionCount,
			activeSessionCount,
		)
	}
	if storedReason == nil || *storedReason != revocationReason {
		t.Fatalf("session revocation reason = %v, want %q", storedReason, revocationReason)
	}
}

var (
	errSyntheticCommitUnknown   = errors.New("synthetic commit result unknown")
	errSyntheticRollbackUnknown = errors.New("synthetic rollback result unknown")
)

type commitUnknownTx struct {
	pgx.Tx
	commitActually  bool
	rollbackCommits bool
}

func (tx *commitUnknownTx) Commit(ctx context.Context) error {
	var err error
	if tx.commitActually {
		err = tx.Tx.Commit(ctx)
	} else {
		err = tx.Tx.Rollback(ctx)
	}
	if err != nil {
		return err
	}
	return errSyntheticCommitUnknown
}

func (tx *commitUnknownTx) Rollback(ctx context.Context) error {
	var err error
	if tx.rollbackCommits {
		err = tx.Tx.Commit(ctx)
	} else {
		err = tx.Tx.Rollback(ctx)
	}
	if err != nil {
		return err
	}
	return errSyntheticRollbackUnknown
}

func newLoginSchemaPool(
	t *testing.T,
	ctx context.Context,
	schema string,
) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(testenv.RequirePostgresDSN(t))
	if err != nil {
		t.Fatal("configure PostgreSQL login audit test database")
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL login audit test database")
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("connect PostgreSQL login audit test database")
	}
	requirePostgres18(t, ctx, pool)
	return pool
}

func completeLoginAuditEvent(t *testing.T) audit.Event {
	t.Helper()
	userID := randomUUID(t)
	role := "member"
	errorCode := string(platform.CodePlatformDatabaseUnavailable)
	source := "synthetic-source-digest"
	return audit.Event{
		ID:                randomUUID(t),
		OccurredAt:        time.Date(2026, time.October, 22, 12, 0, 0, 123000, time.UTC),
		ActorKind:         audit.ActorUser,
		ActorUserID:       &userID,
		ActorSystem:       nil,
		ActorRole:         &role,
		Action:            loginAuditAction,
		TargetType:        loginAuditTargetUser,
		TargetID:          userID,
		Outcome:           loginAuditOutcomeFailed,
		ErrorCode:         &errorCode,
		TraceID:           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		SourceFingerprint: &source,
		Metadata: map[string]any{
			"alpha": "one",
			"beta":  float64(2),
		},
	}
}

func createCommittedIdentitySchema(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) string {
	t.Helper()
	schema := "identity_login_" + randomHex(t, 8)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("create identity login schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := pool.Exec(
			cleanupCtx,
			"DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE",
		); err != nil {
			t.Errorf("drop identity login schema: %v", err)
		}
	})

	tx := beginSchemaTransaction(t, ctx, pool, quotedSchema)
	chain := []string{
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
	for _, name := range chain {
		if _, err := tx.Exec(ctx, readMigration(t, "../../migrations/"+name)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit identity login schema: %v", err)
	}
	return schema
}

func beginSchemaTransaction(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	quotedSchema string,
) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin identity login transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+quotedSchema); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("set identity login search path: %v", err)
	}
	return tx
}

func waitForPostgresBlocker(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	blockerPID int,
	blockedPID int,
) {
	t.Helper()
	for {
		var blocked bool
		if err := pool.QueryRow(
			ctx,
			"SELECT $1 = ANY(pg_blocking_pids($2))",
			blockerPID,
			blockedPID,
		).Scan(&blocked); err != nil {
			t.Fatalf("read PostgreSQL blocking graph: %v", err)
		}
		if blocked {
			return
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf(
				"backend %d did not block behind backend %d: %v",
				blockedPID,
				blockerPID,
				err,
			)
		}
	}
}

type schemaLoginTransactor struct {
	pool         *pgxpool.Pool
	quotedSchema string
	begun        chan int
}

func (t *schemaLoginTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+t.quotedSchema); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	var backendPID int
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&backendPID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	select {
	case t.begun <- backendPID:
	case <-ctx.Done():
		_ = tx.Rollback(ctx)
		return nil, ctx.Err()
	}
	return tx, nil
}

type loginRowLockHasher struct {
	mu          sync.Mutex
	oldPassword string
	oldDigest   string
	fakeProbe   string
	fakeDigest  string
	digests     []string
	hashes      int
}

func (h *loginRowLockHasher) Verify(password string, digest string) (bool, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.digests = append(h.digests, digest)
	switch digest {
	case h.oldDigest:
		return password == h.oldPassword, true, nil
	case h.fakeDigest:
		return password == h.fakeProbe, false, nil
	default:
		return false, false, errors.New("unsupported synthetic credential digest")
	}
}

func (h *loginRowLockHasher) Hash(string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hashes++
	return "synthetic-stale-credential-rehash", nil
}

func (h *loginRowLockHasher) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.digests = nil
	h.hashes = 0
}

func (h *loginRowLockHasher) verified(digest string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, verifiedDigest := range h.digests {
		if verifiedDigest == digest {
			return true
		}
	}
	return false
}

func (h *loginRowLockHasher) hashCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hashes
}

package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPasswordResetGateConstruction_RequiresTransactionBoundaryAndGate(t *testing.T) {
	transactionBoundary := &storeTransactorStub{}
	gate := &passwordResetGateStub{}

	if _, err := NewStore(StoreDependencies{PasswordResetGate: gate}); err == nil {
		t.Fatal("NewStore() error = nil, want missing transaction boundary rejection")
	}
	if _, err := NewStore(StoreDependencies{Transactions: transactionBoundary}); err == nil {
		t.Fatal("NewStore() error = nil, want missing password reset Gate rejection")
	}
	if _, err := NewStore(StoreDependencies{
		Transactions:      transactionBoundary,
		PasswordResetGate: gate,
	}); err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
}

func TestPasswordResetGateConstruction_RejectsTypedNilDependencies(t *testing.T) {
	var typedNilTransactions *storeTransactorStub
	var typedNilGate *passwordResetGateStub

	tests := []struct {
		name         string
		dependencies StoreDependencies
	}{
		{
			name: "transaction boundary",
			dependencies: StoreDependencies{
				Transactions:      typedNilTransactions,
				PasswordResetGate: &passwordResetGateStub{},
			},
		},
		{
			name: "password reset Gate",
			dependencies: StoreDependencies{
				Transactions:      &storeTransactorStub{},
				PasswordResetGate: typedNilGate,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewStore(test.dependencies); err == nil {
				t.Fatal("NewStore() error = nil, want typed-nil dependency rejection")
			}
		})
	}
}

func TestSessionResolvePasswordResetGate_UserLockBeforeGateAndSessionRead(t *testing.T) {
	now := time.Date(2026, time.October, 24, 10, 0, 0, 0, time.UTC)
	store, tx, gate, _ := newResolveStore(
		t,
		rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
		nil,
		validPrincipalRow("synthetic-csrf-token"),
	)
	gate.beforeCheck = func() {
		if len(tx.queryCalls) != 1 ||
			!strings.Contains(tx.queryCalls[0].query, "FROM users") {
			t.Fatalf("queries before Gate = %#v, want only User lock", tx.queryCalls)
		}
	}

	principal, err := store.ResolvePrincipal(context.Background(), ResolveInput{
		Token: "synthetic-session-token",
		Now:   now,
	})

	if err != nil {
		t.Fatalf("ResolvePrincipal() error = %v", err)
	}
	if principal.UserID == "" || gate.calls != 1 ||
		len(tx.queryCalls) != 2 || !tx.committed {
		t.Fatalf(
			"resolve result/Gate/queries/commit = %#v/%d/%d/%t",
			principal,
			gate.calls,
			len(tx.queryCalls),
			tx.committed,
		)
	}
}

func TestSessionResolvePasswordResetGate_FailClosedWithoutPrincipal(t *testing.T) {
	now := time.Date(2026, time.October, 24, 10, 0, 0, 0, time.UTC)
	queryCause := errors.New("synthetic password reset Gate query failure")
	tests := []struct {
		name      string
		gateError error
		wantError error
	}{
		{
			name:      "closed",
			gateError: ErrPasswordResetGateClosed,
			wantError: ErrSessionRequired,
		},
		{
			name:      "query failure",
			gateError: queryCause,
			wantError: ErrStoreUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, tx, gate, _ := newResolveStore(
				t,
				rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
				test.gateError,
				validPrincipalRow("synthetic-csrf-token"),
			)

			principal, err := store.ResolvePrincipal(context.Background(), ResolveInput{
				Token: "synthetic-session-token",
				Now:   now,
			})

			if !errors.Is(err, test.wantError) {
				t.Fatalf("ResolvePrincipal() error = %v, want %v", err, test.wantError)
			}
			if test.gateError == queryCause && !errors.Is(err, queryCause) {
				t.Fatalf("ResolvePrincipal() error = %v, want query cause", err)
			}
			if principal != (Principal{}) || gate.calls != 1 ||
				len(tx.queryCalls) != 1 || len(tx.calls) != 0 ||
				tx.committed || !tx.rolledBack {
				t.Fatalf(
					"Gate failure principal/calls/queries/writes/commit/rollback = %#v/%d/%d/%d/%t/%t",
					principal,
					gate.calls,
					len(tx.queryCalls),
					len(tx.calls),
					tx.committed,
					tx.rolledBack,
				)
			}
		})
	}
}

func TestSessionResolvePasswordResetGate_TransientBoundaryFailuresReturnNoPrincipal(t *testing.T) {
	now := time.Date(2026, time.October, 24, 10, 0, 0, 0, time.UTC)
	t.Run("begin", func(t *testing.T) {
		cause := errors.New("synthetic begin failure")
		gate := &passwordResetGateStub{}
		boundary := &storeTransactorStub{
			row: rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
			err: cause,
		}
		store, err := NewStore(StoreDependencies{
			Transactions:      boundary,
			PasswordResetGate: gate,
		})
		if err != nil {
			t.Fatalf("NewStore() error = %v", err)
		}

		principal, resolveErr := store.ResolvePrincipal(context.Background(), ResolveInput{
			Token: "synthetic-session-token",
			Now:   now,
		})

		if !errors.Is(resolveErr, ErrStoreUnavailable) ||
			!errors.Is(resolveErr, cause) ||
			principal != (Principal{}) || gate.calls != 0 {
			t.Fatalf(
				"begin failure error/principal/Gate = %v/%#v/%d",
				resolveErr,
				principal,
				gate.calls,
			)
		}
	})

	t.Run("User lock", func(t *testing.T) {
		cause := errors.New("synthetic User lock failure")
		tx := newTransactionStub()
		tx.queryRows = []pgx.Row{rowStub{err: cause}}
		gate := &passwordResetGateStub{}
		boundary := &storeTransactorStub{
			row: rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
			tx:  tx,
		}
		store, err := NewStore(StoreDependencies{
			Transactions:      boundary,
			PasswordResetGate: gate,
		})
		if err != nil {
			t.Fatalf("NewStore() error = %v", err)
		}

		principal, resolveErr := store.ResolvePrincipal(context.Background(), ResolveInput{
			Token: "synthetic-session-token",
			Now:   now,
		})

		if !errors.Is(resolveErr, ErrStoreUnavailable) ||
			!errors.Is(resolveErr, cause) ||
			principal != (Principal{}) || gate.calls != 0 ||
			!tx.rolledBack || len(tx.calls) != 0 {
			t.Fatalf(
				"lock failure error/principal/Gate/rollback/writes = %v/%#v/%d/%t/%d",
				resolveErr,
				principal,
				gate.calls,
				tx.rolledBack,
				len(tx.calls),
			)
		}
	})

	t.Run("commit", func(t *testing.T) {
		cause := errors.New("synthetic commit failure")
		store, tx, gate, _ := newResolveStore(
			t,
			rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
			nil,
			validPrincipalRow("synthetic-csrf-token"),
		)
		tx.commitErr = cause

		principal, resolveErr := store.ResolvePrincipal(context.Background(), ResolveInput{
			Token: "synthetic-session-token",
			Now:   now,
		})

		if !errors.Is(resolveErr, ErrStoreUnavailable) ||
			!errors.Is(resolveErr, cause) ||
			principal != (Principal{}) || gate.calls != 1 ||
			tx.committed || tx.rolledBack || len(tx.calls) != 0 {
			t.Fatalf(
				"commit failure error/principal/Gate/commit/rollback/writes = %v/%#v/%d/%t/%t/%d",
				resolveErr,
				principal,
				gate.calls,
				tx.committed,
				tx.rolledBack,
				len(tx.calls),
			)
		}
	})
}

func TestSessionCreate_ExplicitLifetimes_PersistsOnlyDigests(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	input := validCreateInput(now)
	executor := &recordingExecutor{}
	store := newUnitStore(t)

	credentials, err := store.Create(context.Background(), executor, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if credentials.Token != input.Token || credentials.CSRFToken != input.CSRFToken {
		t.Fatalf("Create() credentials = %#v, want original boundary values", credentials)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("Exec() calls = %d, want 1", len(executor.calls))
	}

	call := executor.calls[0]
	if strings.Contains(call.query, input.Token) || strings.Contains(call.query, input.CSRFToken) {
		t.Fatal("Create() SQL contains raw token material")
	}
	tokenDigest := sha256.Sum256([]byte(input.Token))
	csrfDigest := sha256.Sum256([]byte(input.CSRFToken))
	if !bytes.Equal(call.args[2].([]byte), tokenDigest[:]) {
		t.Fatal("Create() token digest does not match SHA-256")
	}
	if !bytes.Equal(call.args[3].([]byte), csrfDigest[:]) {
		t.Fatal("Create() CSRF digest does not match SHA-256")
	}
	for _, argument := range call.args {
		if value, ok := argument.(string); ok && (value == input.Token || value == input.CSRFToken) {
			t.Fatal("Create() passed raw token material to persistence")
		}
	}
	if call.args[11] != input.AbsoluteExpiresAt || call.args[12] != input.IdleExpiresAt {
		t.Fatal("Create() did not preserve caller-injected lifetimes")
	}
}

func TestSessionCreate_MissingOrInvalidLifetime_FailsClosed(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*CreateInput)
	}{
		{name: "missing issued at", mutate: func(input *CreateInput) { input.IssuedAt = time.Time{} }},
		{name: "missing absolute expiry", mutate: func(input *CreateInput) { input.AbsoluteExpiresAt = time.Time{} }},
		{name: "missing idle expiry", mutate: func(input *CreateInput) { input.IdleExpiresAt = time.Time{} }},
		{name: "absolute expiry not after issue", mutate: func(input *CreateInput) { input.AbsoluteExpiresAt = input.IssuedAt }},
		{name: "idle expiry not after issue", mutate: func(input *CreateInput) { input.IdleExpiresAt = input.IssuedAt }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validCreateInput(now)
			test.mutate(&input)
			executor := &recordingExecutor{}
			store := newUnitStore(t)

			_, err := store.Create(context.Background(), executor, input)
			if err == nil {
				t.Fatal("Create() error = nil, want explicit lifetime rejected")
			}
			if len(executor.calls) != 0 {
				t.Fatalf("Exec() calls = %d, want 0", len(executor.calls))
			}
		})
	}
}

func TestSessionResolve_InvalidExpiredRevokedOrOldToken_FailsClosed(t *testing.T) {
	tests := []string{"missing", "expired", "revoked", "rotated old token", "inactive user", "recovery required"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			store, _, _, _ := newResolveStore(
				t,
				rowStub{err: pgx.ErrNoRows},
				nil,
				nil,
			)

			_, err := store.ResolvePrincipal(context.Background(), ResolveInput{
				Token: "synthetic-session-token",
				Now:   time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC),
			})
			if !errors.Is(err, ErrSessionRequired) {
				t.Fatalf("ResolvePrincipal() error = %v, want ErrSessionRequired", err)
			}
		})
	}
}

func TestSessionResolve_UsesDigestAndFailClosedPredicates(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	store, tx, _, boundary := newResolveStore(
		t,
		rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
		nil,
		rowStub{err: pgx.ErrNoRows},
	)

	_, err := store.ResolvePrincipal(context.Background(), ResolveInput{
		Token: "synthetic-session-token",
		Now:   now,
	})
	if !errors.Is(err, ErrSessionRequired) {
		t.Fatalf("ResolvePrincipal() error = %v, want ErrSessionRequired", err)
	}
	tokenDigest := sha256.Sum256([]byte("synthetic-session-token"))
	if len(boundary.args) != 1 || !bytes.Equal(boundary.args[0].([]byte), tokenDigest[:]) {
		t.Fatalf("pre-read args = %#v, want token digest", boundary.args)
	}
	if len(tx.queryCalls) != 2 {
		t.Fatalf("transaction query calls = %d, want User lock and Session re-read", len(tx.queryCalls))
	}
	if got := tx.queryCalls[1].args[1]; got != now {
		t.Fatalf("ResolvePrincipal() now = %v, want %v", got, now)
	}
	for _, predicate := range []string{
		"s.revoked_at IS NULL",
		"s.absolute_expires_at > $2",
		"s.idle_expires_at > $2",
		"u.status = 'active'",
		"u.recovery_required = false",
		"s.access_scope = 'ordinary'",
	} {
		if !strings.Contains(tx.queryCalls[1].query, predicate) {
			t.Fatalf("ResolvePrincipal() SQL missing fail-closed predicate %q", predicate)
		}
	}
	if strings.Contains(tx.queryCalls[1].query, "synthetic-session-token") {
		t.Fatal("ResolvePrincipal() SQL contains the raw session token")
	}
}

func TestSessionResolve_CSRFMustMatchStoredDigest(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	expectedCSRF := "synthetic-csrf-token"

	for _, test := range []struct {
		name      string
		csrfToken string
		wantErr   bool
	}{
		{name: "missing", wantErr: true},
		{name: "forged", csrfToken: "forged-csrf-token", wantErr: true},
		{name: "matching", csrfToken: expectedCSRF},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _, _, _ := newResolveStore(
				t,
				rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
				nil,
				validPrincipalRow(expectedCSRF),
			)
			_, err := store.ResolvePrincipal(context.Background(), ResolveInput{
				Token:       "synthetic-session-token",
				CSRFToken:   test.csrfToken,
				RequireCSRF: true,
				Now:         now,
			})
			if test.wantErr && !errors.Is(err, ErrCSRFFailed) {
				t.Fatalf("ResolvePrincipal() error = %v, want ErrCSRFFailed", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ResolvePrincipal() error = %v", err)
			}
		})
	}
}

func TestSessionResolve_OnboardingStateBuildsRestrictedPrincipal(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	csrfDigest := sha256.Sum256([]byte("synthetic-csrf-token"))
	acknowledged := "notice-v1"
	current := "notice-v2"
	row := rowStub{values: []any{
		"00000000-0000-4000-8000-000000000101",
		"00000000-0000-4000-8000-000000000201",
		"admin",
		false,
		"active",
		"temporary",
		&acknowledged,
		(*time.Time)(nil),
		&current,
		csrfDigest[:],
		"temporary",
		func() *string {
			value := "00000000-0000-4000-8000-000000000301"
			return &value
		}(),
		true,
	}}
	store, _, _, _ := newResolveStore(
		t,
		rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
		nil,
		row,
	)

	principal, err := store.ResolvePrincipal(context.Background(), ResolveInput{
		Token: "synthetic-session-token",
		Now:   now,
	})
	if err != nil {
		t.Fatalf("ResolvePrincipal() error = %v", err)
	}
	if !principal.MustChangePassword || !principal.MustAcknowledgePrivacy {
		t.Fatalf("ResolvePrincipal() principal = %#v, want both onboarding restrictions", principal)
	}
	if principal.Allows(AccessGeneral) {
		t.Fatal("restricted principal allowed general access")
	}
}

func TestSessionPrincipal_RestrictedAccess_OnlyAllowsOnboardingCapabilities(t *testing.T) {
	principal := Principal{
		UserID:             "00000000-0000-4000-8000-000000000201",
		Status:             "active",
		MustChangePassword: true,
	}
	tests := []struct {
		access Access
		want   bool
	}{
		{access: AccessCurrentIdentity, want: true},
		{access: AccessChangePassword, want: true},
		{access: AccessCurrentPrivacyNotice, want: true},
		{access: AccessAcknowledgePrivacy, want: true},
		{access: AccessGeneral, want: false},
	}
	for _, test := range tests {
		if got := principal.Allows(test.access); got != test.want {
			t.Fatalf("Allows(%q) = %t, want %t", test.access, got, test.want)
		}
	}

	principal.MustChangePassword = false
	if !principal.Allows(AccessGeneral) {
		t.Fatal("unrestricted principal denied general access")
	}
}

func TestSessionRotate_InvalidReplacementDoesNotRevoke(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	replacement := validCreateInput(now)
	replacement.AbsoluteExpiresAt = time.Time{}
	transaction := newTransactionStub()
	store := newUnitStore(t)

	_, err := store.Rotate(
		context.Background(),
		transaction,
		"old-synthetic-token",
		now,
		"rotated",
		replacement,
	)
	if err == nil {
		t.Fatal("Rotate() error = nil, want invalid replacement rejected")
	}
	if len(transaction.calls) != 0 {
		t.Fatalf("Exec() calls = %d, want no revoke or insert", len(transaction.calls))
	}
	if !transaction.oldSessionActive {
		t.Fatal("invalid replacement revoked the old session")
	}
}

func TestSessionRotate_InsertFailureLeavesRollbackToCaller(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	insertCause := errors.New("synthetic insert failure")
	transaction := newTransactionStub()
	transaction.insertErr = insertCause
	store := newUnitStore(t)

	_, err := store.Rotate(
		context.Background(),
		transaction,
		"old-synthetic-token",
		now,
		"rotated",
		validCreateInput(now),
	)
	if !errors.Is(err, ErrStoreUnavailable) || !errors.Is(err, insertCause) {
		t.Fatalf("Rotate() error = %v, want store classification and insert cause", err)
	}
	if transaction.rolledBack || transaction.committed {
		t.Fatalf("transaction rollback/commit = %t/%t, want caller ownership", transaction.rolledBack, transaction.committed)
	}
	if transaction.closed {
		t.Fatal("Rotate() closed the transaction after insert failure")
	}
	if err := transaction.Rollback(context.Background()); err != nil {
		t.Fatalf("caller Rollback() error = %v", err)
	}
	if !transaction.oldSessionActive {
		t.Fatal("caller rollback did not preserve the old session")
	}
}

func TestSessionRotate_SuccessLeavesTransactionOpenForAuditAndCallerCommit(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	transaction := newTransactionStub()
	store := newUnitStore(t)

	_, err := store.Rotate(
		context.Background(),
		transaction,
		"old-synthetic-token",
		now,
		"rotated",
		validCreateInput(now),
	)
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if len(transaction.calls) != 2 {
		t.Fatalf("Exec() calls = %d, want revoke and insert", len(transaction.calls))
	}
	if !strings.Contains(transaction.calls[0].query, "UPDATE sessions") ||
		!strings.Contains(transaction.calls[1].query, "INSERT INTO sessions") {
		t.Fatal("Rotate() did not revoke before inserting replacement")
	}
	oldDigest := sha256.Sum256([]byte("old-synthetic-token"))
	if !bytes.Equal(transaction.calls[0].args[0].([]byte), oldDigest[:]) {
		t.Fatal("Rotate() did not revoke by old token digest")
	}
	if transaction.committed || transaction.rolledBack || transaction.closed {
		t.Fatalf(
			"transaction commit/rollback/closed = %t/%t/%t, want caller-owned open transaction",
			transaction.committed,
			transaction.rolledBack,
			transaction.closed,
		)
	}
	if _, err := transaction.Exec(
		context.Background(),
		"INSERT INTO audit_events (action) VALUES ($1)",
		"identity.login",
	); err != nil {
		t.Fatalf("caller audit Exec() error = %v", err)
	}
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatalf("caller Commit() error = %v", err)
	}
	if transaction.oldSessionActive {
		t.Fatal("caller commit left the old session active")
	}
}

func TestSessionRotate_CallerRollbackAfterAuditFailurePreservesOldSession(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	auditCause := errors.New("synthetic audit failure")
	transaction := newTransactionStub()
	transaction.auditErr = auditCause
	store := newUnitStore(t)

	_, err := store.Rotate(
		context.Background(),
		transaction,
		"old-synthetic-token",
		now,
		"rotated",
		validCreateInput(now),
	)
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	_, err = transaction.Exec(
		context.Background(),
		"INSERT INTO audit_events (action) VALUES ($1)",
		"identity.login",
	)
	if !errors.Is(err, auditCause) {
		t.Fatalf("audit Exec() error = %v, want audit cause", err)
	}
	if err := transaction.Rollback(context.Background()); err != nil {
		t.Fatalf("caller Rollback() error = %v", err)
	}
	if !transaction.oldSessionActive {
		t.Fatal("caller rollback after audit failure did not preserve the old session")
	}
}

func TestSessionRotate_RequiresPGXTransactionBoundary(t *testing.T) {
	method, ok := reflect.TypeOf(Store{}).MethodByName("Rotate")
	if !ok {
		t.Fatal("Store.Rotate method is missing")
	}
	transactionType := reflect.TypeOf((*pgx.Tx)(nil)).Elem()
	if got := method.Type.In(2); got != transactionType {
		t.Fatalf("Rotate() transaction parameter = %v, want %v", got, transactionType)
	}
	if reflect.TypeOf((*pgxpool.Pool)(nil)).Implements(transactionType) {
		t.Fatal("auto-commit pgxpool.Pool unexpectedly satisfies pgx.Tx")
	}
}

func TestSessionStoreErrors_PreserveClassificationAndCause(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		cause *storeCause
		run   func(error) error
	}{
		{
			name:  "create",
			cause: &storeCause{operation: "create"},
			run: func(cause error) error {
				executor := &recordingExecutor{err: cause}
				store := newUnitStore(t)
				_, err := store.Create(context.Background(), executor, validCreateInput(now))
				return err
			},
		},
		{
			name:  "revoke",
			cause: &storeCause{operation: "revoke"},
			run: func(cause error) error {
				executor := &recordingExecutor{err: cause}
				store := newUnitStore(t)
				return store.Revoke(context.Background(), executor, "synthetic-session-token", now, "logout")
			},
		},
		{
			name:  "resolve",
			cause: &storeCause{operation: "resolve"},
			run: func(cause error) error {
				store, _, _, _ := newResolveStore(t, rowStub{err: cause}, nil, nil)
				_, err := store.ResolvePrincipal(context.Background(), ResolveInput{
					Token: "synthetic-session-token",
					Now:   now,
				})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.run(test.cause)
			if !errors.Is(err, ErrStoreUnavailable) {
				t.Fatalf("error = %v, want ErrStoreUnavailable classification", err)
			}
			if !errors.Is(err, test.cause) {
				t.Fatalf("error = %v, want original cause preserved", err)
			}
			var cause *storeCause
			if !errors.As(err, &cause) || cause != test.cause {
				t.Fatalf("error = %v, want typed original cause preserved", err)
			}
		})
	}
}

func validCreateInput(now time.Time) CreateInput {
	return CreateInput{
		ID:                          "00000000-0000-4000-8000-000000000101",
		UserID:                      "00000000-0000-4000-8000-000000000201",
		Token:                       "synthetic-session-token",
		CSRFToken:                   "synthetic-csrf-token",
		AuthenticatedCredentialKind: CredentialPermanent,
		IssuedAt:                    now,
		AbsoluteExpiresAt:           now.Add(2 * time.Hour),
		IdleExpiresAt:               now.Add(30 * time.Minute),
	}
}

func newUnitStore(t *testing.T) Store {
	t.Helper()
	store, err := NewStore(StoreDependencies{
		Transactions:      &storeTransactorStub{},
		PasswordResetGate: &passwordResetGateStub{},
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}

func newResolveStore(
	t *testing.T,
	preRead pgx.Row,
	gateErr error,
	resolved pgx.Row,
) (Store, *transactionStub, *passwordResetGateStub, *storeTransactorStub) {
	t.Helper()
	tx := newTransactionStub()
	tx.queryRows = []pgx.Row{
		rowStub{values: []any{"00000000-0000-4000-8000-000000000201"}},
		resolved,
	}
	gate := &passwordResetGateStub{err: gateErr}
	boundary := &storeTransactorStub{row: preRead, tx: tx}
	store, err := NewStore(StoreDependencies{
		Transactions:      boundary,
		PasswordResetGate: gate,
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store, tx, gate, boundary
}

func validPrincipalRow(csrfToken string) rowStub {
	csrfDigest := sha256.Sum256([]byte(csrfToken))
	completedAt := time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC)
	return rowStub{values: []any{
		"00000000-0000-4000-8000-000000000101",
		"00000000-0000-4000-8000-000000000201",
		"member",
		false,
		"active",
		"permanent",
		(*string)(nil),
		&completedAt,
		(*string)(nil),
		csrfDigest[:],
		"permanent",
		(*string)(nil),
		true,
	}}
}

type execCall struct {
	query string
	args  []any
}

type storeCause struct {
	operation string
}

func (e *storeCause) Error() string {
	return "synthetic " + e.operation + " cause"
}

type recordingExecutor struct {
	calls []execCall
	err   error
}

func (e *recordingExecutor) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	e.calls = append(e.calls, execCall{query: query, args: append([]any(nil), args...)})
	return pgconn.NewCommandTag("INSERT 0 1"), e.err
}

type transactionStub struct {
	calls             []execCall
	insertErr         error
	auditErr          error
	commitErr         error
	rollbackErr       error
	committed         bool
	rolledBack        bool
	closed            bool
	oldSessionActive  bool
	stagedRevocation  bool
	stagedReplacement bool
	stagedAudit       bool
	queryRows         []pgx.Row
	queryCalls        []execCall
}

func newTransactionStub() *transactionStub {
	return &transactionStub{oldSessionActive: true}
}

func (t *transactionStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported by test stub")
}

func (t *transactionStub) Commit(context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	if t.commitErr != nil {
		return t.commitErr
	}
	t.committed = true
	t.closed = true
	if t.stagedRevocation {
		t.oldSessionActive = false
	}
	return nil
}

func (t *transactionStub) Rollback(context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	t.rolledBack = true
	t.closed = true
	t.stagedRevocation = false
	t.stagedReplacement = false
	t.stagedAudit = false
	return t.rollbackErr
}

func (t *transactionStub) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("CopyFrom is not supported by test stub")
}

func (t *transactionStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (t *transactionStub) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (t *transactionStub) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("Prepare is not supported by test stub")
}

func (t *transactionStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if t.closed {
		return pgconn.CommandTag{}, pgx.ErrTxClosed
	}
	t.calls = append(t.calls, execCall{query: query, args: append([]any(nil), args...)})
	switch {
	case strings.Contains(query, "UPDATE sessions"):
		t.stagedRevocation = true
	case strings.Contains(query, "INSERT INTO sessions"):
		if t.insertErr != nil {
			return pgconn.CommandTag{}, t.insertErr
		}
		t.stagedReplacement = true
	case strings.Contains(query, "INSERT INTO audit_events"):
		if t.auditErr != nil {
			return pgconn.CommandTag{}, t.auditErr
		}
		t.stagedAudit = true
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (t *transactionStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("Query is not supported by test stub")
}

func (t *transactionStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	t.queryCalls = append(t.queryCalls, execCall{query: query, args: append([]any(nil), args...)})
	if len(t.queryRows) == 0 {
		return rowStub{err: pgx.ErrNoRows}
	}
	row := t.queryRows[0]
	t.queryRows = t.queryRows[1:]
	return row
}

func (t *transactionStub) Conn() *pgx.Conn {
	return nil
}

type storeTransactorStub struct {
	query string
	args  []any
	row   pgx.Row
	tx    pgx.Tx
	err   error
}

func (s *storeTransactorStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	s.query = query
	s.args = append([]any(nil), args...)
	if s.row == nil {
		return rowStub{err: pgx.ErrNoRows}
	}
	return s.row
}

func (s *storeTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	return s.tx, s.err
}

type passwordResetGateStub struct {
	err         error
	calls       int
	userIDs     []string
	beforeCheck func()
}

func (s *passwordResetGateStub) CheckLockedUser(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) error {
	if s.beforeCheck != nil {
		s.beforeCheck()
	}
	s.calls++
	s.userIDs = append(s.userIDs, userID)
	return s.err
}

type rowStub struct {
	values []any
	err    error
}

func (r rowStub) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(destinations) != len(r.values) {
		return errors.New("unexpected scan destination count")
	}
	for index, value := range r.values {
		switch destination := destinations[index].(type) {
		case *string:
			*destination = value.(string)
		case *bool:
			*destination = value.(bool)
		case *int64:
			*destination = value.(int64)
		case **string:
			*destination = value.(*string)
		case **time.Time:
			*destination = value.(*time.Time)
		case *[]byte:
			*destination = append([]byte(nil), value.([]byte)...)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestBootstrapInitialAdminExistingDoesNotReadOrOverwriteSecret(t *testing.T) {
	tx := &bootstrapTestTx{row: bootstrapTestRow{values: []any{int64(1), int64(1), int64(1)}}}
	passwordReads := 0
	manager, err := NewInitialAdminTrustManager(InitialAdminDependencies{
		Transactions: bootstrapTestTransactor{tx: tx},
		Password: func(context.Context) (string, error) {
			passwordReads++
			return "Bootstrap123", nil
		},
		PasswordPolicy: security.PasswordPolicy{},
		PasswordHasher: bootstrapTestPasswords{},
		Repository:     &bootstrapTestRepository{},
		Audit:          &bootstrapTestAudit{},
		IDs:            &bootstrapTestIDs{},
		Clock:          bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewInitialAdminTrustManager() error = %v", err)
	}

	if err := manager.EnsureInitialAdmin(context.Background()); err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if passwordReads != 0 {
		t.Fatalf("password reads = %d, want 0", passwordReads)
	}
	if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
		t.Fatalf("commit/rollback calls = %d/%d, want 1/0", tx.commitCalls, tx.rollbackCalls)
	}
}

func TestBootstrapInitialAdminCreatesOneProtectedBootstrapAggregate(t *testing.T) {
	tx := &bootstrapTestTx{row: bootstrapTestRow{values: []any{int64(0), int64(0), int64(0)}}}
	repository := &bootstrapTestRepository{}
	auditor := &bootstrapTestAudit{}
	ids := &bootstrapTestIDs{}
	manager, err := NewInitialAdminTrustManager(InitialAdminDependencies{
		Transactions: bootstrapTestTransactor{tx: tx},
		Password: func(context.Context) (string, error) {
			return "Bootstrap123", nil
		},
		PasswordPolicy: security.PasswordPolicy{},
		PasswordHasher: bootstrapTestPasswords{},
		Repository:     repository,
		Audit:          auditor,
		IDs:            ids,
		Clock:          bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewInitialAdminTrustManager() error = %v", err)
	}

	if err := manager.EnsureInitialAdmin(context.Background()); err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if repository.user.Username != "admin" ||
		repository.user.Role != RoleAdmin ||
		!repository.user.IsInitialAdmin ||
		repository.user.Status != UserStatusActive ||
		repository.user.CredentialMode != CredentialModeBootstrap ||
		repository.user.PermanentPasswordDigest == nil {
		t.Fatalf("bootstrap user = %#v", repository.user)
	}
	if repository.identifier.Kind != IdentifierUsername ||
		repository.identifier.NormalizedValue != "admin" ||
		repository.identifier.Lifecycle != IdentifierActive {
		t.Fatalf("bootstrap identifier = %#v", repository.identifier)
	}
	if auditor.event.ActorKind != audit.ActorSystem ||
		auditor.event.Action != "identity.initial_admin_bootstrap" ||
		auditor.event.TargetID != repository.user.ID ||
		strings.Contains(auditor.event.TraceID, "Bootstrap123") {
		t.Fatalf("bootstrap audit = %#v", auditor.event)
	}
	if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
		t.Fatalf("commit/rollback calls = %d/%d, want 1/0", tx.commitCalls, tx.rollbackCalls)
	}
}

func TestBootstrapConcurrentStartsCreateOnlyOneInitialAdmin(t *testing.T) {
	state := &bootstrapConcurrentState{}
	passwordReads := 0
	var passwordMu sync.Mutex
	manager, err := NewInitialAdminTrustManager(InitialAdminDependencies{
		Transactions: bootstrapConcurrentTransactor{state: state},
		Password: func(context.Context) (string, error) {
			passwordMu.Lock()
			passwordReads++
			passwordMu.Unlock()
			return "Bootstrap123", nil
		},
		PasswordPolicy: security.PasswordPolicy{},
		PasswordHasher: bootstrapTestPasswords{},
		Repository:     bootstrapConcurrentRepository{state: state},
		Audit:          &bootstrapTestAudit{},
		IDs:            &bootstrapConcurrentIDs{},
		Clock:          bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewInitialAdminTrustManager() error = %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- manager.EnsureInitialAdmin(context.Background())
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("EnsureInitialAdmin() error = %v", err)
		}
	}
	passwordMu.Lock()
	gotPasswordReads := passwordReads
	passwordMu.Unlock()
	state.mu.Lock()
	gotInserts := state.inserts
	state.mu.Unlock()
	if gotPasswordReads != 1 || gotInserts != 1 {
		t.Fatalf("password reads/inserts = %d/%d, want 1/1", gotPasswordReads, gotInserts)
	}
}

func TestOnboardingPasswordChangeRejectsUnauthorizedPrincipalBeforeVerifier(t *testing.T) {
	passwords := &onboardingTestPasswords{}
	tx := &bootstrapTestTx{}
	auditor := &bootstrapTestAudit{}
	service, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:      bootstrapTestTransactor{tx: tx},
		Repository:        &onboardingTestRepository{},
		PasswordResetGate: &loginPasswordResetGateStub{},
		Passwords:         passwords,
		Policy:            security.PasswordPolicy{},
		Sessions:          &onboardingTestSessions{},
		SessionIssuer:     onboardingTestIssuer{},
		Audit:             auditor,
		IDs:               &bootstrapTestIDs{},
		Clock:             bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewPasswordChangeService() error = %v", err)
	}

	_, err = service.ChangePassword(context.Background(), PasswordChangeRequest{
		Principal: session.Principal{
			SessionID: "session-1", UserID: "user-1", Status: "active",
			Role: "member", MustChangePassword: false, IsInitialAdmin: false,
		},
		CurrentSessionToken: "current-token",
		CurrentPassword:     "Current123",
		NewPassword:         "Replacement123",
		TraceID:             "trace-1",
	})
	if platform.PublicCode(err) != platform.CodeAuthForbidden {
		t.Fatalf("ChangePassword() error = %v, want auth.forbidden", err)
	}
	if passwords.verifyCalls != 0 {
		t.Fatalf("Verify() calls = %d, want 0", passwords.verifyCalls)
	}
	assertPasswordChangeAudit(
		t,
		auditor.event,
		"user-1",
		"member",
		"failed",
		platform.CodeAuthForbidden,
		"trace-1",
	)
	if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
		t.Fatalf("commit/rollback calls = %d/%d, want 1/0", tx.commitCalls, tx.rollbackCalls)
	}
}

func TestOnboardingPasswordChangePasswordResetConstruction_RequiresGate(t *testing.T) {
	_, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:  bootstrapTestTransactor{tx: &bootstrapTestTx{}},
		Repository:    &onboardingTestRepository{},
		Passwords:     &onboardingTestPasswords{},
		Policy:        security.PasswordPolicy{},
		Sessions:      &onboardingTestSessions{},
		SessionIssuer: onboardingTestIssuer{},
		Audit:         &bootstrapTestAudit{},
		IDs:           &bootstrapTestIDs{},
		Clock:         bootstrapTestClock{},
	})
	if err == nil {
		t.Fatal("NewPasswordChangeService() error = nil, want missing password reset Gate rejection")
	}
}

func TestOnboardingPasswordChangePasswordResetConstruction_RejectsTypedNilGate(t *testing.T) {
	repository := &onboardingTestRepository{}
	passwords := &onboardingTestPasswords{}
	sessions := &onboardingTestSessions{}
	var typedNilGate *loginPasswordResetGateStub

	_, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:      bootstrapTestTransactor{tx: &bootstrapTestTx{}},
		Repository:        repository,
		PasswordResetGate: typedNilGate,
		Passwords:         passwords,
		Policy:            security.PasswordPolicy{},
		Sessions:          sessions,
		SessionIssuer:     onboardingTestIssuer{},
		Audit:             &bootstrapTestAudit{},
		IDs:               &bootstrapTestIDs{},
		Clock:             bootstrapTestClock{},
	})

	if err == nil {
		t.Fatal("NewPasswordChangeService() error = nil, want typed-nil password reset Gate rejection")
	}
	if repository.findUserCalls != 0 || passwords.verifyCalls != 0 ||
		sessions.rotateCalls != 0 || sessions.revokeOtherCalls != 0 {
		t.Fatalf(
			"constructor side effects user/verify/rotate/revoke = %d/%d/%d/%d, want zero",
			repository.findUserCalls,
			passwords.verifyCalls,
			sessions.rotateCalls,
			sessions.revokeOtherCalls,
		)
	}
}

func TestOnboardingPasswordChangePasswordResetGate_FailsBeforeCredentialWrites(t *testing.T) {
	queryCause := errors.New("synthetic password reset Gate query failure")
	tests := []struct {
		name      string
		gateError error
		wantCode  platform.Code
	}{
		{
			name:      "closed",
			gateError: session.ErrPasswordResetGateClosed,
			wantCode:  platform.CodeAuthSessionRequired,
		},
		{
			name:      "query failure",
			gateError: queryCause,
			wantCode:  platform.CodePlatformDatabaseUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &bootstrapTestTx{}
			repository := &onboardingTestRepository{
				user: onboardingTestUser(CredentialModeBootstrap),
			}
			passwords := &onboardingTestPasswords{matched: true}
			sessions := &onboardingTestSessions{}
			service := newOnboardingTestService(t, tx, repository, passwords, sessions)
			gate := &loginPasswordResetGateStub{err: test.gateError}
			gate.beforeCheck = func() {
				if repository.findUserCalls != 1 {
					t.Fatalf(
						"FindUserForPasswordChange() calls before Gate = %d, want 1",
						repository.findUserCalls,
					)
				}
			}
			service.passwordChangeDependencies.PasswordResetGate = gate

			result, err := service.ChangePassword(
				context.Background(),
				passwordChangeTestRequest(repository.user, session.CredentialBootstrap),
			)

			if got := platform.PublicCode(err); got != test.wantCode {
				t.Fatalf("ChangePassword() code = %q, want %q; error = %v", got, test.wantCode, err)
			}
			if passwords.verifyCalls != 0 ||
				repository.updatedDigest != "" ||
				sessions.rotateCalls != 0 ||
				sessions.revokeOtherCalls != 0 ||
				result != (PasswordChangeResult{}) {
				t.Fatalf(
					"Gate failure effects verify/digest/rotate/revoke/result = %d/%q/%d/%d/%#v",
					passwords.verifyCalls,
					repository.updatedDigest,
					sessions.rotateCalls,
					sessions.revokeOtherCalls,
					result,
				)
			}
			if gate.calls != 1 || len(gate.userIDs) != 1 ||
				gate.userIDs[0] != repository.user.ID {
				t.Fatalf("password reset Gate calls = %#v", gate)
			}
		})
	}
}

func TestOnboardingPasswordChangePasswordResetGate_UserLockFailureSkipsGate(t *testing.T) {
	repository := &onboardingTestRepository{
		findUserErr: errors.New("synthetic User lock failure"),
	}
	passwords := &onboardingTestPasswords{matched: true}
	sessions := &onboardingTestSessions{}
	service := newOnboardingTestService(
		t,
		&bootstrapTestTx{},
		repository,
		passwords,
		sessions,
	)
	gate := &loginPasswordResetGateStub{}
	service.passwordChangeDependencies.PasswordResetGate = gate

	result, err := service.ChangePassword(
		context.Background(),
		passwordChangeTestRequest(
			onboardingTestUser(CredentialModeBootstrap),
			session.CredentialBootstrap,
		),
	)

	if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("ChangePassword() code = %q, want platform.database_unavailable; error = %v", got, err)
	}
	if gate.calls != 0 || passwords.verifyCalls != 0 ||
		repository.updatedDigest != "" || sessions.rotateCalls != 0 ||
		result != (PasswordChangeResult{}) {
		t.Fatalf(
			"User lock failure effects Gate/verify/digest/rotate/result = %d/%d/%q/%d/%#v",
			gate.calls,
			passwords.verifyCalls,
			repository.updatedDigest,
			sessions.rotateCalls,
			result,
		)
	}
}

func TestOnboardingPasswordChangePasswordResetLineage_RequiresExactSucceededAuthority(t *testing.T) {
	const (
		operationID  = "password-reset-operation-1"
		credentialID = "password-reset-credential-1"
	)
	settledGeneration := int64(2)
	newFixture := func() (
		*onboardingTestRepository,
		*onboardingTestPasswords,
		*onboardingTestSessions,
		PasswordChangeRequest,
	) {
		user := onboardingTestUser(CredentialModeTemporary)
		user.AuthGeneration = 2
		user.PasswordResetCurrentOperationID = passwordResetTestPointer(operationID)
		operation := PasswordResetOperation{
			ID:                    operationID,
			UserID:                user.ID,
			TemporaryCredentialID: credentialID,
			TargetAuthGeneration:  2,
			SettlementState:       PasswordResetSettlementTemporaryActive,
			SettledAuthGeneration: &settledGeneration,
			DeliveryState:         PasswordResetDeliveryAccepted,
			State:                 PasswordResetOperationSucceeded,
		}
		issuedAt := bootstrapTestClock{}.Now().Add(-time.Minute)
		repository := &onboardingTestRepository{
			user:      user,
			operation: &operation,
			temporary: TemporaryCredential{
				ID:                credentialID,
				UserID:            user.ID,
				Purpose:           TemporaryCredentialPasswordReset,
				Generation:        1,
				AuthGeneration:    2,
				SourceOperationID: passwordResetTestPointer(operationID),
				State:             TemporaryCredentialActive,
				SecretDigest:      testTemporaryCredentialDigest,
				DeliveredToEmail:  "member@example.test",
				IssuedAt:          issuedAt,
				ExpiresAt:         issuedAt.Add(temporaryCredentialLifetime),
			},
		}
		passwords := &onboardingTestPasswords{matched: true}
		sessions := &onboardingTestSessions{}
		request := passwordChangeTestRequest(user, session.CredentialTemporary)
		request.Principal.AuthenticatedCredentialID = passwordResetTestPointer(credentialID)
		return repository, passwords, sessions, request
	}

	t.Run("exact succeeded lineage", func(t *testing.T) {
		repository, passwords, sessions, request := newFixture()
		service := newOnboardingTestService(
			t,
			&bootstrapTestTx{},
			repository,
			passwords,
			sessions,
		)

		result, err := service.ChangePassword(context.Background(), request)

		if err != nil {
			t.Fatalf("ChangePassword() error = %v", err)
		}
		if repository.settleCalls != 1 ||
			repository.updatedExpectedGeneration != 2 ||
			repository.updatedSuccessorGeneration != 3 ||
			repository.updatedSourceOperationID == nil ||
			*repository.updatedSourceOperationID != operationID ||
			repository.updatedSourceCredentialID == nil ||
			*repository.updatedSourceCredentialID != credentialID ||
			passwords.verifyCalls != 1 ||
			sessions.rotateCalls != 1 ||
			result.SessionToken == "" {
			t.Fatalf(
				"success lineage effects repository/passwords/sessions/result = %#v/%#v/%#v/%#v",
				repository,
				passwords,
				sessions,
				result,
			)
		}
	})

	t.Run("non succeeded lineage", func(t *testing.T) {
		repository, passwords, sessions, request := newFixture()
		repository.operation.State = PasswordResetOperationDispatching
		service := newOnboardingTestService(
			t,
			&bootstrapTestTx{},
			repository,
			passwords,
			sessions,
		)

		result, err := service.ChangePassword(context.Background(), request)

		if got := platform.PublicCode(err); got != platform.CodeAuthSessionRequired {
			t.Fatalf("ChangePassword() code = %q, want auth.session_required; error = %v", got, err)
		}
		if repository.settleCalls != 0 ||
			repository.updatedDigest != "" ||
			passwords.verifyCalls != 0 ||
			sessions.rotateCalls != 0 ||
			result != (PasswordChangeResult{}) {
			t.Fatalf(
				"invalid lineage effects repository/passwords/sessions/result = %#v/%#v/%#v/%#v",
				repository,
				passwords,
				sessions,
				result,
			)
		}
	})
}

func TestOnboardingPasswordChangeStaleCredentialRevokesOnlyCurrentSession(t *testing.T) {
	tx := &bootstrapTestTx{}
	sessions := &onboardingTestSessions{}
	passwords := &onboardingTestPasswords{}
	auditor := &bootstrapTestAudit{}
	repository := &onboardingTestRepository{user: onboardingTestUser(CredentialModeBootstrap)}
	service := newOnboardingTestServiceWithAudit(t, tx, repository, passwords, sessions, auditor)

	_, err := service.ChangePassword(context.Background(), PasswordChangeRequest{
		Principal: session.Principal{
			SessionID: "session-1", UserID: repository.user.ID, Status: "active",
			Role: "admin", IsInitialAdmin: true, MustChangePassword: true,
			AuthenticatedCredential: session.CredentialPermanent,
		},
		CurrentSessionToken: "current-token",
		CurrentPassword:     "Bootstrap123",
		NewPassword:         "Replacement123",
		TraceID:             "trace-1",
	})
	if platform.PublicCode(err) != platform.CodeAuthSessionRequired {
		t.Fatalf("ChangePassword() error = %v, want auth.session_required", err)
	}
	if sessions.revokeCalls != 1 || sessions.rotateCalls != 0 || sessions.revokeOtherCalls != 0 {
		t.Fatalf(
			"session revoke/rotate/others = %d/%d/%d, want 1/0/0",
			sessions.revokeCalls,
			sessions.rotateCalls,
			sessions.revokeOtherCalls,
		)
	}
	if passwords.verifyCalls != 0 || tx.commitCalls != 1 {
		t.Fatalf("verify/commit calls = %d/%d, want 0/1", passwords.verifyCalls, tx.commitCalls)
	}
	assertPasswordChangeAudit(
		t,
		auditor.event,
		repository.user.ID,
		"admin",
		"failed",
		platform.CodeAuthSessionRequired,
		"trace-1",
	)
	if auditor.tx != tx {
		t.Fatal("stale credential audit was not appended in the revocation transaction")
	}
}

func TestOnboardingPasswordChangeSuccessPersistsPermanentAndRotatesSessions(t *testing.T) {
	tx := &bootstrapTestTx{}
	sessions := &onboardingTestSessions{}
	passwords := &onboardingTestPasswords{matched: true}
	auditor := &bootstrapTestAudit{}
	repository := &onboardingTestRepository{user: onboardingTestUser(CredentialModeBootstrap)}
	service := newOnboardingTestServiceWithAudit(t, tx, repository, passwords, sessions, auditor)

	result, err := service.ChangePassword(context.Background(), PasswordChangeRequest{
		Principal: session.Principal{
			SessionID: "session-1", UserID: repository.user.ID, Status: "active",
			Role: "admin", IsInitialAdmin: true, MustChangePassword: true,
			AuthenticatedCredential: session.CredentialBootstrap,
		},
		CurrentSessionToken: "current-token",
		CurrentPassword:     "Bootstrap123",
		NewPassword:         "Replacement123",
		TraceID:             "trace-1",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if repository.updatedDigest != "digest:Replacement123" ||
		repository.updatedMode != CredentialModePermanent {
		t.Fatalf("updated digest/mode = %q/%q", repository.updatedDigest, repository.updatedMode)
	}
	if sessions.rotateCalls != 1 || sessions.revokeOtherCalls != 1 ||
		sessions.replacementID == "" || result.SessionToken != "replacement-token" ||
		result.CSRFToken != "replacement-csrf" {
		t.Fatalf("password result/sessions = %#v / %#v", result, sessions)
	}
	if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
		t.Fatalf("commit/rollback calls = %d/%d, want 1/0", tx.commitCalls, tx.rollbackCalls)
	}
	assertPasswordChangeAudit(
		t,
		auditor.event,
		repository.user.ID,
		"admin",
		"succeeded",
		"",
		"trace-1",
	)
}

func TestOnboardingPasswordValidationReturnsExactFieldError(t *testing.T) {
	tests := []struct {
		name        string
		matched     bool
		newPassword string
		wantField   string
		wantCode    string
	}{
		{
			name: "current mismatch", newPassword: "Replacement123",
			wantField: "current_password", wantCode: "current_password_mismatch",
		},
		{
			name: "new too short", matched: true, newPassword: "Abc123",
			wantField: "new_password", wantCode: "password_too_short",
		},
		{
			name: "new missing letter", matched: true, newPassword: "12345678",
			wantField: "new_password", wantCode: "password_requires_letter",
		},
		{
			name: "new missing digit", matched: true, newPassword: "abcdefgh",
			wantField: "new_password", wantCode: "password_requires_digit",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &bootstrapTestTx{}
			passwords := &onboardingTestPasswords{matched: test.matched}
			auditor := &bootstrapTestAudit{}
			repository := &onboardingTestRepository{
				user: onboardingTestUser(CredentialModeBootstrap),
			}
			service := newOnboardingTestServiceWithAudit(
				t,
				tx,
				repository,
				passwords,
				&onboardingTestSessions{},
				auditor,
			)
			_, err := service.ChangePassword(context.Background(), PasswordChangeRequest{
				Principal: session.Principal{
					SessionID: "session-1", UserID: repository.user.ID,
					Status: "active", Role: "admin", IsInitialAdmin: true,
					MustChangePassword:      true,
					AuthenticatedCredential: session.CredentialBootstrap,
				},
				CurrentSessionToken: "current-token",
				CurrentPassword:     "Bootstrap123",
				NewPassword:         test.newPassword,
				TraceID:             "trace-1",
			})
			if platform.PublicCode(err) != platform.CodeIdentityValidationFailed {
				t.Fatalf("ChangePassword() error = %v", err)
			}
			response := httptest.NewRecorder()
			platform.TraceMiddleware(http.HandlerFunc(func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				_ = platform.WriteError(writer, request, err)
			})).ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/", nil))
			var envelope platform.Envelope
			if decodeErr := json.Unmarshal(response.Body.Bytes(), &envelope); decodeErr != nil {
				t.Fatalf("decode error envelope: %v", decodeErr)
			}
			if len(envelope.FieldErrors) != 1 ||
				envelope.FieldErrors[0].Field != test.wantField ||
				envelope.FieldErrors[0].Code != test.wantCode {
				t.Fatalf("field errors = %#v", envelope.FieldErrors)
			}
			if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
				t.Fatalf(
					"commit/rollback calls = %d/%d, want 1/0",
					tx.commitCalls,
					tx.rollbackCalls,
				)
			}
			assertPasswordChangeAudit(
				t,
				auditor.event,
				repository.user.ID,
				"admin",
				"failed",
				platform.CodeIdentityValidationFailed,
				"trace-1",
			)
		})
	}
}

func TestOnboardingHandlerRejectsUnauthorizedPrincipalBeforeReadingBody(t *testing.T) {
	service := &onboardingHandlerService{}
	handler, err := NewPasswordChangeHandler(service, 4096)
	if err != nil {
		t.Fatalf("NewPasswordChangeHandler() error = %v", err)
	}
	resolver := onboardingResolver{principal: session.Principal{
		SessionID: "session-1", UserID: "user-1", Status: "active",
		Role: "member", IsInitialAdmin: false, MustChangePassword: false,
		AuthenticatedCredential: session.CredentialPermanent,
	}}
	authorizer, err := platform.NewHTTPAuthorizer(
		resolver,
		func() time.Time { return bootstrapTestClock{}.Now() },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := platform.TraceMiddleware(authorizer.Protect(
		session.AccessChangePassword,
		http.HandlerFunc(handler.ChangePassword),
	))
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/users/me/password",
		strings.NewReader(`{"current_password":`),
	)
	request.AddCookie(&http.Cookie{
		Name: platform.SessionCookieName, Value: "current-token",
	})
	request.Header.Set(platform.CSRFHeaderName, "current-csrf")
	response := httptest.NewRecorder()

	protected.ServeHTTP(response, request)

	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if response.Code != http.StatusForbidden ||
		envelope.Code != platform.CodeAuthForbidden ||
		service.calls != 0 ||
		service.auditCalls != 1 {
		t.Fatalf(
			"status/code/service/audit calls = %d/%q/%d/%d",
			response.Code,
			envelope.Code,
			service.calls,
			service.auditCalls,
		)
	}
	if service.auditRequest.Principal.UserID != "user-1" ||
		service.auditRequest.ErrorCode != platform.CodeAuthForbidden ||
		service.auditRequest.TraceID == "" {
		t.Fatalf("failure audit request = %#v", service.auditRequest)
	}
}

func TestOnboardingPasswordChangeAuditAppendFailureRollsBack(t *testing.T) {
	tx := &bootstrapTestTx{}
	auditor := &bootstrapTestAudit{err: errors.New("audit unavailable")}
	repository := &onboardingTestRepository{user: onboardingTestUser(CredentialModeBootstrap)}
	sessions := &onboardingTestSessions{}
	service := newOnboardingTestServiceWithAudit(
		t,
		tx,
		repository,
		&onboardingTestPasswords{matched: true},
		sessions,
		auditor,
	)

	_, err := service.ChangePassword(context.Background(), passwordChangeTestRequest(
		repository.user,
		session.CredentialBootstrap,
	))

	if platform.PublicCode(err) != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("ChangePassword() error = %v, want platform.database_unavailable", err)
	}
	if tx.commitCalls != 0 || tx.rollbackCalls != 1 {
		t.Fatalf("commit/rollback calls = %d/%d, want 0/1", tx.commitCalls, tx.rollbackCalls)
	}
	if sessions.rotateCalls != 1 || sessions.revokeOtherCalls != 1 {
		t.Fatalf(
			"rotate/revoke-other calls = %d/%d, want 1/1 before rollback",
			sessions.rotateCalls,
			sessions.revokeOtherCalls,
		)
	}
}

func TestOnboardingPasswordChangePersistenceFailureAuditsAfterRollback(t *testing.T) {
	changeTx := &bootstrapTestTx{}
	auditTx := &bootstrapTestTx{}
	transactor := &onboardingSequenceTransactor{txs: []pgx.Tx{changeTx, auditTx}}
	auditor := &bootstrapTestAudit{}
	repository := &onboardingTestRepository{
		user:           onboardingTestUser(CredentialModeBootstrap),
		setPasswordErr: errors.New("write failed"),
	}
	service, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:      transactor,
		Repository:        repository,
		PasswordResetGate: &loginPasswordResetGateStub{},
		Passwords:         &onboardingTestPasswords{matched: true},
		Policy:            security.PasswordPolicy{},
		Sessions:          &onboardingTestSessions{},
		SessionIssuer:     onboardingTestIssuer{},
		Audit:             auditor,
		IDs:               &bootstrapTestIDs{},
		Clock:             bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewPasswordChangeService() error = %v", err)
	}

	_, err = service.ChangePassword(context.Background(), passwordChangeTestRequest(
		repository.user,
		session.CredentialBootstrap,
	))

	if platform.PublicCode(err) != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("ChangePassword() error = %v, want platform.database_unavailable", err)
	}
	if changeTx.rollbackCalls != 1 || changeTx.commitCalls != 0 ||
		auditTx.commitCalls != 1 || auditTx.rollbackCalls != 0 {
		t.Fatalf(
			"change/audit commit/rollback = %d/%d %d/%d",
			changeTx.commitCalls,
			changeTx.rollbackCalls,
			auditTx.commitCalls,
			auditTx.rollbackCalls,
		)
	}
	assertPasswordChangeAudit(
		t,
		auditor.event,
		repository.user.ID,
		"admin",
		"failed",
		platform.CodePlatformDatabaseUnavailable,
		"trace-1",
	)
}

func TestOnboardingConcurrentPasswordChangeStaleWaiterReturnsSessionRequired(t *testing.T) {
	state := newOnboardingConcurrentPasswordState()
	transactor := &onboardingConcurrentPasswordTransactor{state: state}
	repository := &onboardingConcurrentPasswordRepository{state: state}
	passwords := &onboardingTestPasswords{matched: true}
	sessions := &onboardingConcurrentPasswordSessions{}
	auditor := &bootstrapTestAudit{}
	service, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:      transactor,
		Repository:        repository,
		PasswordResetGate: &loginPasswordResetGateStub{},
		Passwords:         passwords,
		Policy:            security.PasswordPolicy{},
		Sessions:          sessions,
		SessionIssuer:     onboardingTestIssuer{},
		Audit:             auditor,
		IDs:               &bootstrapTestIDs{},
		Clock:             bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewPasswordChangeService() error = %v", err)
	}
	requestA := passwordChangeTestRequest(state.user, session.CredentialTemporary)
	requestA.Principal.AuthenticatedCredentialID = &state.temporary.ID
	requestA.CurrentSessionToken = "token-a"
	requestB := requestA
	requestB.Principal.SessionID = "session-b"
	requestB.CurrentSessionToken = "token-b"

	resultA := make(chan error, 1)
	go func() {
		_, changeErr := service.ChangePassword(context.Background(), requestA)
		resultA <- changeErr
	}()
	<-state.aLocked

	resultB := make(chan error, 1)
	go func() {
		_, changeErr := service.ChangePassword(context.Background(), requestB)
		resultB <- changeErr
	}()
	<-state.bWaiting
	close(state.allowA)

	if err := <-resultA; err != nil {
		t.Fatalf("request A error = %v", err)
	}
	if err := <-resultB; platform.PublicCode(err) != platform.CodeAuthSessionRequired {
		t.Fatalf("request B error = %v, want auth.session_required", err)
	}
	if passwords.verifyCalls != 1 {
		t.Fatalf("Verify() calls = %d, want 1", passwords.verifyCalls)
	}
	if sessions.revokedTokens["token-b"] != 1 ||
		sessions.revokedTokens["token-a"] != 0 ||
		sessions.rotateCalls != 1 ||
		sessions.revokeOtherCalls != 1 {
		t.Fatalf("session effects = %#v", sessions)
	}
}

func newOnboardingTestService(
	t *testing.T,
	tx *bootstrapTestTx,
	repository *onboardingTestRepository,
	passwords *onboardingTestPasswords,
	sessions *onboardingTestSessions,
) *Service {
	t.Helper()
	return newOnboardingTestServiceWithAudit(
		t,
		tx,
		repository,
		passwords,
		sessions,
		&bootstrapTestAudit{},
	)
}

func newOnboardingTestServiceWithAudit(
	t *testing.T,
	tx *bootstrapTestTx,
	repository *onboardingTestRepository,
	passwords *onboardingTestPasswords,
	sessions PasswordChangeSessionStore,
	auditor *bootstrapTestAudit,
) *Service {
	t.Helper()
	service, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:      bootstrapTestTransactor{tx: tx},
		Repository:        repository,
		PasswordResetGate: &loginPasswordResetGateStub{},
		Passwords:         passwords,
		Policy:            security.PasswordPolicy{},
		Sessions:          sessions,
		SessionIssuer:     onboardingTestIssuer{},
		Audit:             auditor,
		IDs:               &bootstrapTestIDs{},
		Clock:             bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewPasswordChangeService() error = %v", err)
	}
	return service
}

func passwordChangeTestRequest(
	user User,
	credential session.CredentialKind,
) PasswordChangeRequest {
	return PasswordChangeRequest{
		Principal: session.Principal{
			SessionID: "session-1", UserID: user.ID, Status: "active",
			Role: string(user.Role), IsInitialAdmin: user.IsInitialAdmin,
			MustChangePassword:      true,
			AuthenticatedCredential: credential,
		},
		CurrentSessionToken: "current-token",
		CurrentPassword:     "Bootstrap123",
		NewPassword:         "Replacement123",
		TraceID:             "trace-1",
	}
}

func assertPasswordChangeAudit(
	t *testing.T,
	event audit.Event,
	userID string,
	role string,
	outcome string,
	errorCode platform.Code,
	traceID string,
) {
	t.Helper()
	if event.Action != "identity.password.change" ||
		event.ActorKind != audit.ActorUser ||
		event.ActorUserID == nil || *event.ActorUserID != userID ||
		event.ActorRole == nil || *event.ActorRole != role ||
		event.TargetType != "user" || event.TargetID != userID ||
		event.Outcome != outcome ||
		event.TraceID != traceID {
		t.Fatalf("password change audit = %#v", event)
	}
	if errorCode == "" {
		if event.ErrorCode != nil {
			t.Fatalf("audit error code = %v, want nil", event.ErrorCode)
		}
	} else if event.ErrorCode == nil || *event.ErrorCode != string(errorCode) {
		t.Fatalf("audit error code = %v, want %q", event.ErrorCode, errorCode)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	for _, secret := range []string{
		"Bootstrap123",
		"Replacement123",
		"current-token",
		"current-csrf",
		"digest:",
	} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("audit event contains secret marker %q: %s", secret, encoded)
		}
	}
}

func onboardingTestUser(mode CredentialMode) User {
	digest := "digest:Bootstrap123"
	return User{
		ID: "user-1", Username: "admin", PermanentPasswordDigest: &digest,
		Role: RoleAdmin, IsInitialAdmin: true, Status: UserStatusActive,
		CredentialMode: mode,
	}
}

type bootstrapTestTransactor struct{ tx pgx.Tx }

func (t bootstrapTestTransactor) Begin(context.Context) (pgx.Tx, error) { return t.tx, nil }

type bootstrapTestTx struct {
	row           pgx.Row
	commitCalls   int
	rollbackCalls int
	closed        bool
}

func (tx *bootstrapTestTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}
func (tx *bootstrapTestTx) Commit(context.Context) error {
	tx.commitCalls++
	tx.closed = true
	return nil
}
func (tx *bootstrapTestTx) Rollback(context.Context) error {
	tx.rollbackCalls++
	tx.closed = true
	return nil
}
func (tx *bootstrapTestTx) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *bootstrapTestTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *bootstrapTestTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *bootstrapTestTx) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *bootstrapTestTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (tx *bootstrapTestTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *bootstrapTestTx) QueryRow(context.Context, string, ...any) pgx.Row {
	if tx.row == nil {
		return bootstrapTestRow{err: pgx.ErrNoRows}
	}
	return tx.row
}
func (tx *bootstrapTestTx) Conn() *pgx.Conn { return nil }

type bootstrapTestRow struct {
	values []any
	err    error
}

func (r bootstrapTestRow) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(destinations) != len(r.values) {
		return errors.New("unexpected scan count")
	}
	for index, value := range r.values {
		switch destination := destinations[index].(type) {
		case *int64:
			*destination = value.(int64)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

type bootstrapTestPasswords struct{}

func (bootstrapTestPasswords) Hash(value string) (string, error) { return "digest:" + value, nil }
func (bootstrapTestPasswords) Verify(value, digest string) (bool, bool, error) {
	return digest == "digest:"+value, false, nil
}

type bootstrapTestRepository struct {
	user       User
	identifier IdentityIdentifier
}

func (r *bootstrapTestRepository) InsertUser(_ context.Context, _ pgx.Tx, user User) error {
	r.user = user
	return nil
}
func (r *bootstrapTestRepository) InsertIdentifier(
	_ context.Context,
	_ pgx.Tx,
	identifier IdentityIdentifier,
) error {
	r.identifier = identifier
	return nil
}

type bootstrapTestAudit struct {
	event  audit.Event
	events []audit.Event
	tx     pgx.Tx
	err    error
}

func (a *bootstrapTestAudit) Append(_ context.Context, tx pgx.Tx, event audit.Event) error {
	a.event = event
	a.events = append(a.events, event)
	a.tx = tx
	return a.err
}

type bootstrapTestIDs struct{ next int }

func (g *bootstrapTestIDs) NewID() (string, error) {
	g.next++
	return "id-" + string(rune('0'+g.next)), nil
}

type bootstrapTestClock struct{}

func (bootstrapTestClock) Now() time.Time {
	return time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
}

type bootstrapConcurrentState struct {
	mu      sync.Mutex
	created bool
	inserts int
}

type bootstrapConcurrentTransactor struct {
	state *bootstrapConcurrentState
}

func (t bootstrapConcurrentTransactor) Begin(context.Context) (pgx.Tx, error) {
	return &bootstrapConcurrentTx{state: t.state}, nil
}

type bootstrapConcurrentTx struct {
	state  *bootstrapConcurrentState
	locked bool
	closed bool
}

func (tx *bootstrapConcurrentTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}
func (tx *bootstrapConcurrentTx) Commit(context.Context) error {
	tx.close()
	return nil
}
func (tx *bootstrapConcurrentTx) Rollback(context.Context) error {
	tx.close()
	return nil
}
func (tx *bootstrapConcurrentTx) close() {
	if tx.closed {
		return
	}
	tx.closed = true
	if tx.locked {
		tx.state.mu.Unlock()
	}
}
func (tx *bootstrapConcurrentTx) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *bootstrapConcurrentTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}
func (tx *bootstrapConcurrentTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }
func (tx *bootstrapConcurrentTx) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *bootstrapConcurrentTx) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	if !tx.locked {
		tx.state.mu.Lock()
		tx.locked = true
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (tx *bootstrapConcurrentTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *bootstrapConcurrentTx) QueryRow(context.Context, string, ...any) pgx.Row {
	if tx.state.created {
		return bootstrapTestRow{values: []any{int64(1), int64(1), int64(1)}}
	}
	return bootstrapTestRow{values: []any{int64(0), int64(0), int64(0)}}
}
func (tx *bootstrapConcurrentTx) Conn() *pgx.Conn { return nil }

type bootstrapConcurrentRepository struct {
	state *bootstrapConcurrentState
}

func (r bootstrapConcurrentRepository) InsertUser(
	context.Context,
	pgx.Tx,
	User,
) error {
	r.state.created = true
	r.state.inserts++
	return nil
}
func (bootstrapConcurrentRepository) InsertIdentifier(
	context.Context,
	pgx.Tx,
	IdentityIdentifier,
) error {
	return nil
}

type bootstrapConcurrentIDs struct {
	mu   sync.Mutex
	next int
}

func (g *bootstrapConcurrentIDs) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return fmt.Sprintf("concurrent-id-%d", g.next), nil
}

type onboardingTestPasswords struct {
	verifyCalls int
	matched     bool
}

func (p *onboardingTestPasswords) Hash(value string) (string, error) {
	return "digest:" + value, nil
}
func (p *onboardingTestPasswords) Verify(string, string) (bool, bool, error) {
	p.verifyCalls++
	return p.matched, false, nil
}

type onboardingTestRepository struct {
	user                       User
	findUserErr                error
	findUserCalls              int
	operation                  *PasswordResetOperation
	operationErr               error
	temporary                  TemporaryCredential
	updatedDigest              string
	updatedMode                CredentialMode
	updatedExpectedGeneration  int64
	updatedSuccessorGeneration int64
	updatedSourceOperationID   *string
	updatedSourceCredentialID  *string
	setPasswordErr             error
	settleCalls                int
}

func (r *onboardingTestRepository) FindPasswordResetOperationForPasswordChange(
	context.Context,
	pgx.Tx,
	string,
	string,
	string,
) (PasswordResetOperation, error) {
	if r.operationErr != nil {
		return PasswordResetOperation{}, r.operationErr
	}
	if r.operation == nil {
		return PasswordResetOperation{}, pgx.ErrNoRows
	}
	return *r.operation, nil
}

func (r *onboardingTestRepository) FindUserForPasswordChange(
	context.Context,
	pgx.Tx,
	string,
) (User, error) {
	r.findUserCalls++
	return r.user, r.findUserErr
}
func (r *onboardingTestRepository) FindLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (TemporaryCredential, error) {
	return r.temporary, nil
}
func (r *onboardingTestRepository) SetPermanentPassword(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	digest string,
	mode CredentialMode,
	expectedGeneration int64,
	successorGeneration int64,
	sourceOperationID *string,
	sourceCredentialID *string,
	_ time.Time,
) error {
	if r.setPasswordErr != nil {
		return r.setPasswordErr
	}
	r.updatedDigest = digest
	r.updatedMode = mode
	r.updatedExpectedGeneration = expectedGeneration
	r.updatedSuccessorGeneration = successorGeneration
	r.updatedSourceOperationID = sourceOperationID
	r.updatedSourceCredentialID = sourceCredentialID
	return nil
}
func (r *onboardingTestRepository) SettlePasswordResetSuccessor(
	context.Context,
	pgx.Tx,
	PasswordResetOperation,
	TemporaryCredential,
	int64,
	time.Time,
) error {
	r.settleCalls++
	return nil
}
func (r *onboardingTestRepository) ConsumeTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
) error {
	return nil
}

type onboardingTestSessions struct {
	revokeCalls      int
	rotateCalls      int
	revokeOtherCalls int
	replacementID    string
}

func (s *onboardingTestSessions) Revoke(
	context.Context,
	session.Executor,
	string,
	time.Time,
	string,
) error {
	s.revokeCalls++
	return nil
}
func (s *onboardingTestSessions) Rotate(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	_ time.Time,
	_ string,
	replacement session.CreateInput,
) (session.Credentials, error) {
	s.rotateCalls++
	s.replacementID = replacement.ID
	return session.Credentials{Token: "replacement-token", CSRFToken: "replacement-csrf"}, nil
}
func (s *onboardingTestSessions) RevokeOtherSessions(
	context.Context,
	pgx.Tx,
	string,
	string,
	time.Time,
	string,
) error {
	s.revokeOtherCalls++
	return nil
}

type onboardingTestIssuer struct{}

func (onboardingTestIssuer) Issue(
	user User,
	kind session.CredentialKind,
	credentialID *string,
) (session.CreateInput, error) {
	if kind != session.CredentialPermanent || credentialID != nil {
		return session.CreateInput{}, errors.New("unexpected replacement credential")
	}
	now := bootstrapTestClock{}.Now()
	return session.CreateInput{
		ID: "replacement-session", UserID: user.ID,
		Token: "replacement-token", CSRFToken: "replacement-csrf",
		AuthenticatedCredentialKind: kind,
		IssuedAt:                    now,
		AbsoluteExpiresAt:           now.Add(time.Hour),
		IdleExpiresAt:               now.Add(30 * time.Minute),
	}, nil
}

type onboardingHandlerService struct {
	calls        int
	auditCalls   int
	auditRequest PasswordChangeFailureAuditRequest
	auditErr     error
}

func (s *onboardingHandlerService) ChangePassword(
	context.Context,
	PasswordChangeRequest,
) (PasswordChangeResult, error) {
	s.calls++
	return PasswordChangeResult{}, errors.New("unexpected password change call")
}

func (s *onboardingHandlerService) AuditPasswordChangeFailure(
	_ context.Context,
	request PasswordChangeFailureAuditRequest,
) error {
	s.auditCalls++
	s.auditRequest = request
	return s.auditErr
}

type onboardingResolver struct {
	principal session.Principal
}

func (r onboardingResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return r.principal, nil
}

type onboardingSequenceTransactor struct {
	txs  []pgx.Tx
	next int
}

func (t *onboardingSequenceTransactor) Begin(context.Context) (pgx.Tx, error) {
	if t.next >= len(t.txs) {
		return nil, errors.New("unexpected transaction")
	}
	tx := t.txs[t.next]
	t.next++
	return tx, nil
}

type onboardingConcurrentPasswordState struct {
	rowLock sync.Mutex
	beginMu sync.Mutex
	begins  int

	user      User
	temporary TemporaryCredential
	operation PasswordResetOperation

	aLocked  chan struct{}
	bWaiting chan struct{}
	allowA   chan struct{}
}

func newOnboardingConcurrentPasswordState() *onboardingConcurrentPasswordState {
	now := bootstrapTestClock{}.Now()
	user := onboardingTestUser(CredentialModeTemporary)
	user.IsInitialAdmin = false
	user.Role = RoleMember
	user.AuthGeneration = 1
	operationID := "password-reset-operation-1"
	user.PasswordResetCurrentOperationID = &operationID
	settledGeneration := int64(1)
	credentialID := "temporary-1"
	return &onboardingConcurrentPasswordState{
		user: user,
		temporary: TemporaryCredential{
			ID: credentialID, UserID: user.ID,
			Purpose: TemporaryCredentialPasswordReset, Generation: 1,
			State:             TemporaryCredentialActive,
			SecretDigest:      "$argon2id$v=19$m=8,t=1,p=1$dGVtcA$dGVtcC1kaWdlc3Q",
			DeliveredToEmail:  "member@example.test",
			IssuedAt:          now.Add(-time.Minute),
			ExpiresAt:         now.Add(-time.Minute).Add(temporaryCredentialLifetime),
			AuthGeneration:    1,
			SourceOperationID: &operationID,
		},
		operation: PasswordResetOperation{
			ID:                    operationID,
			UserID:                user.ID,
			State:                 PasswordResetOperationSucceeded,
			DeliveryState:         PasswordResetDeliveryAccepted,
			SettlementState:       PasswordResetSettlementTemporaryActive,
			TemporaryCredentialID: credentialID,
			TargetAuthGeneration:  1,
			SettledAuthGeneration: &settledGeneration,
		},
		aLocked:  make(chan struct{}),
		bWaiting: make(chan struct{}),
		allowA:   make(chan struct{}),
	}
}

type onboardingConcurrentPasswordTransactor struct {
	state *onboardingConcurrentPasswordState
}

func (t *onboardingConcurrentPasswordTransactor) Begin(context.Context) (pgx.Tx, error) {
	t.state.beginMu.Lock()
	index := t.state.begins
	t.state.begins++
	t.state.beginMu.Unlock()
	return &onboardingConcurrentPasswordTx{
		bootstrapTestTx: &bootstrapTestTx{},
		state:           t.state,
		index:           index,
	}, nil
}

type onboardingConcurrentPasswordTx struct {
	*bootstrapTestTx
	state       *onboardingConcurrentPasswordState
	index       int
	locked      bool
	pendingUser *User
}

func (tx *onboardingConcurrentPasswordTx) Commit(context.Context) error {
	if tx.pendingUser != nil {
		tx.state.user = *tx.pendingUser
	}
	tx.bootstrapTestTx.commitCalls++
	tx.release()
	return nil
}

func (tx *onboardingConcurrentPasswordTx) Rollback(context.Context) error {
	tx.bootstrapTestTx.rollbackCalls++
	tx.release()
	return nil
}

func (tx *onboardingConcurrentPasswordTx) release() {
	if tx.locked {
		tx.locked = false
		tx.state.rowLock.Unlock()
	}
}

type onboardingConcurrentPasswordRepository struct {
	state *onboardingConcurrentPasswordState
}

func (r *onboardingConcurrentPasswordRepository) FindPasswordResetOperationForPasswordChange(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	_ string,
	_ string,
) (PasswordResetOperation, error) {
	return r.state.operation, nil
}

func (r *onboardingConcurrentPasswordRepository) FindUserForPasswordChange(
	_ context.Context,
	transaction pgx.Tx,
	_ string,
) (User, error) {
	tx := transaction.(*onboardingConcurrentPasswordTx)
	if tx.index == 0 {
		r.state.rowLock.Lock()
		tx.locked = true
		close(r.state.aLocked)
		<-r.state.allowA
	} else {
		close(r.state.bWaiting)
		r.state.rowLock.Lock()
		tx.locked = true
	}
	return r.state.user, nil
}

func (r *onboardingConcurrentPasswordRepository) FindLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (TemporaryCredential, error) {
	return r.state.temporary, nil
}

func (r *onboardingConcurrentPasswordRepository) SetPermanentPassword(
	_ context.Context,
	transaction pgx.Tx,
	_ string,
	digest string,
	mode CredentialMode,
	_ int64,
	successorGeneration int64,
	sourceOperationID *string,
	sourceCredentialID *string,
	at time.Time,
) error {
	tx := transaction.(*onboardingConcurrentPasswordTx)
	user := r.state.user
	user.PermanentPasswordDigest = &digest
	user.CredentialMode = mode
	user.AuthGeneration = successorGeneration
	user.PermanentPasswordGeneration = &successorGeneration
	user.PermanentPasswordSourceOperationID = sourceOperationID
	user.PermanentPasswordSourceCredentialID = sourceCredentialID
	user.UpdatedAt = at
	tx.pendingUser = &user
	return nil
}

func (*onboardingConcurrentPasswordRepository) SettlePasswordResetSuccessor(
	context.Context,
	pgx.Tx,
	PasswordResetOperation,
	TemporaryCredential,
	int64,
	time.Time,
) error {
	return nil
}

func (*onboardingConcurrentPasswordRepository) ConsumeTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
) error {
	return nil
}

type onboardingConcurrentPasswordSessions struct {
	mu               sync.Mutex
	revokedTokens    map[string]int
	rotateCalls      int
	revokeOtherCalls int
}

func (s *onboardingConcurrentPasswordSessions) Revoke(
	_ context.Context,
	_ session.Executor,
	token string,
	_ time.Time,
	_ string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revokedTokens == nil {
		s.revokedTokens = map[string]int{}
	}
	s.revokedTokens[token]++
	return nil
}

func (s *onboardingConcurrentPasswordSessions) Rotate(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
	string,
	session.CreateInput,
) (session.Credentials, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rotateCalls++
	return session.Credentials{Token: "replacement-token", CSRFToken: "replacement-csrf"}, nil
}

func (s *onboardingConcurrentPasswordSessions) RevokeOtherSessions(
	context.Context,
	pgx.Tx,
	string,
	string,
	time.Time,
	string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokeOtherCalls++
	return nil
}

func TestRecoveryPasswordChange_OrdinaryPathRejectsRequiredFlag(t *testing.T) {
	tx := &bootstrapTestTx{}
	user := onboardingTestUser(CredentialModeRecoveryBootstrap)
	user.IsInitialAdmin = true
	user.RecoveryRequired = true
	repository := &onboardingTestRepository{user: user}
	sessions := &onboardingTestSessions{}
	service := newOnboardingTestService(t, tx, repository, &onboardingTestPasswords{matched: true}, sessions)
	result, err := service.ChangePassword(context.Background(), passwordChangeTestRequest(user, session.CredentialRecoveryBootstrap))
	if err == nil || result.SessionToken != "" || repository.user.CredentialMode != CredentialModeRecoveryBootstrap {
		t.Fatal("ordinary password change bypassed recovery trust transaction")
	}
}

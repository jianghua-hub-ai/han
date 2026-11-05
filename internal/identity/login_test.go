package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	testCredentialDigest          = "$argon2id$v=19$m=8,t=1,p=1$c2FsdA$ZGlnZXN0"
	testTemporaryCredentialDigest = "$argon2id$v=19$m=8,t=1,p=1$dGVtcA$dGVtcC1kaWdlc3Q"
	testFakeCredentialProbe       = "SyntheticFakeCredentialProbe123"
)

func TestLoginPasswordResetConstruction_RequiresGateBeforeCredentialProbe(t *testing.T) {
	passwords := &loginHasherStub{matches: map[string]string{
		"fake-digest": testFakeCredentialProbe,
	}}
	dependencies := loginConstructorDependencies(
		passwords,
		"fake-digest",
		testFakeCredentialProbe,
	)
	dependencies.PasswordResetGate = nil

	if _, err := NewLoginService(dependencies); err == nil {
		t.Fatal("NewLoginService() error = nil, want missing password reset Gate rejection")
	}
	if len(passwords.verifiedDigests) != 0 {
		t.Fatalf("credential probe calls = %d, want 0", len(passwords.verifiedDigests))
	}
}

func TestLoginPasswordResetConstruction_RejectsTypedNilGateBeforeCredentialProbe(t *testing.T) {
	passwords := &loginHasherStub{matches: map[string]string{
		"fake-digest": testFakeCredentialProbe,
	}}
	dependencies := loginConstructorDependencies(
		passwords,
		"fake-digest",
		testFakeCredentialProbe,
	)
	transactions := dependencies.Transactions.(*loginTransactorStub)
	repository := dependencies.Repository.(*loginRepositoryStub)
	sessions := dependencies.Sessions.(*loginSessionStoreStub)
	var typedNilGate *loginPasswordResetGateStub
	dependencies.PasswordResetGate = typedNilGate

	if _, err := NewLoginService(dependencies); err == nil {
		t.Fatal("NewLoginService() error = nil, want typed-nil password reset Gate rejection")
	}
	if len(passwords.verifiedDigests) != 0 {
		t.Fatalf("credential probe calls = %d, want 0", len(passwords.verifiedDigests))
	}
	if transactions.beginCalls != 0 || repository.findUserCalls != 0 ||
		sessions.calls != 0 || sessions.createCalls != 0 || sessions.rotateCalls != 0 {
		t.Fatalf(
			"constructor DB/session effects begin/find/session/create/rotate = %d/%d/%d/%d/%d",
			transactions.beginCalls,
			repository.findUserCalls,
			sessions.calls,
			sessions.createCalls,
			sessions.rotateCalls,
		)
	}
}

func TestLoginPasswordResetConstruction_SecurityCriticalFunctionsFitLineLimit(t *testing.T) {
	assertFunctionBodyLineLimit(t, "login.go", map[string]bool{
		"Login": true, "newLoginExecution": true, "run": true, "begin": true,
		"rollback": true, "checkAbuse": true, "authenticate": true,
		"rejectCredentials": true, "checkPasswordResetGate": true,
		"verifyCredential": true, "prepareSession": true, "storeSession": true,
		"appendAuditAndCommit": true, "result": true, "fail": true,
	})
	assertFunctionBodyLineLimit(t, "service.go", map[string]bool{
		"ChangePassword": true, "validatePasswordChangeRequest": true,
		"newPasswordChangeExecution": true, "run": true, "rollback": true,
		"loadAuthority": true, "loadAndVerifyCredential": true,
		"verifyCurrentPassword": true, "hashReplacementPassword": true,
		"persistPermanentCredential": true, "settleCredentialLineage": true,
		"applyPermanentCredential": true, "finalizeOnboarding": true,
		"rotateSessions": true, "appendAuditAndCommit": true,
		"failInTransaction": true, "failAfterRollback": true,
	})
}

func TestLoginPasswordResetGate_FailClosedBeforeCredentialVerification(t *testing.T) {
	queryCause := errors.New("synthetic password reset Gate query failure")
	tests := []struct {
		name      string
		gateError error
		wantCode  platform.Code
	}{
		{name: "clear"},
		{
			name:      "closed",
			gateError: session.ErrPasswordResetGateClosed,
			wantCode:  platform.CodeAuthLoginFailed,
		},
		{
			name:      "query failure",
			gateError: queryCause,
			wantCode:  platform.CodeAuthLoginFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoginFixture(t)
			fixture.passwordResetGate.err = test.gateError
			fixture.passwordResetGate.beforeCheck = func() {
				if fixture.repository.findUserCalls != 1 {
					t.Fatalf(
						"FindUserByUsername() calls before Gate = %d, want 1",
						fixture.repository.findUserCalls,
					)
				}
			}

			result, err := fixture.service.Login(
				fixture.context,
				fixture.request(LoginRequest{
					Username:          "Alice",
					Password:          "Password123",
					SourceFingerprint: "source-digest",
				}),
			)

			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("Login() error = %v", err)
				}
				if len(fixture.hasher.verifiedDigests) != 1 ||
					fixture.sessions.calls != 1 ||
					result.SessionToken == "" {
					t.Fatalf(
						"clear Gate verify/session/result = %v/%d/%#v",
						fixture.hasher.verifiedDigests,
						fixture.sessions.calls,
						result,
					)
				}
			} else {
				if got := platform.PublicCode(err); got != test.wantCode {
					t.Fatalf("Login() code = %q, want %q; error = %v", got, test.wantCode, err)
				}
				if len(fixture.hasher.verifiedDigests) != 0 ||
					fixture.sessions.calls != 0 ||
					fixture.repository.permanentRehashes != 0 ||
					fixture.repository.temporaryRehashes != 0 ||
					result != (LoginResult{}) {
					t.Fatalf(
						"closed Gate effects verify/session/rehash/result = %v/%d/%d/%d/%#v",
						fixture.hasher.verifiedDigests,
						fixture.sessions.calls,
						fixture.repository.permanentRehashes,
						fixture.repository.temporaryRehashes,
						result,
					)
				}
			}
			if fixture.passwordResetGate.calls != 1 ||
				len(fixture.passwordResetGate.userIDs) != 1 ||
				fixture.passwordResetGate.userIDs[0] != fixture.repository.user.ID {
				t.Fatalf("password reset Gate calls = %#v", fixture.passwordResetGate)
			}
		})
	}
}

func TestLoginPasswordResetGate_MissingUserUsesFakeCredentialWithoutGate(t *testing.T) {
	fixture := newLoginFixture(t)
	fixture.repository.userErr = pgx.ErrNoRows

	_, err := fixture.service.Login(
		fixture.context,
		fixture.request(LoginRequest{
			Username:          "missing",
			Password:          "Password123",
			SourceFingerprint: "source-digest",
		}),
	)

	if got := platform.PublicCode(err); got != platform.CodeAuthLoginFailed {
		t.Fatalf("Login() code = %q, want auth.login_failed; error = %v", got, err)
	}
	if fixture.passwordResetGate.calls != 0 {
		t.Fatalf("password reset Gate calls = %d, want 0", fixture.passwordResetGate.calls)
	}
	if len(fixture.hasher.verifiedDigests) != 1 ||
		fixture.hasher.verifiedDigests[0] != "fake-digest" {
		t.Fatalf("fake credential verification = %v", fixture.hasher.verifiedDigests)
	}
}

func TestLogin_FailureMatrix_ReturnsUnified401AndPreservesOldSession(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		user      User
		userErr   error
		temporary *TemporaryCredential
		password  string
	}{
		{name: "blank username", username: " \t ", password: "Wrong123"},
		{name: "account missing", userErr: pgx.ErrNoRows, password: "Wrong123"},
		{name: "wrong password", user: loginTestUser(), password: "Wrong123"},
		{name: "suspended", user: loginTestUserWithStatus(UserStatusSuspended), password: "Password123"},
		{name: "deleted", user: loginTestUserWithStatus(UserStatusDeleted), password: "Password123"},
		{name: "recovery required", user: loginTestRecoveryRequiredUser(), password: "Password123"},
		{name: "recovery locked", user: loginTestUserWithMode(CredentialModeRecoveryLocked), password: "Password123"},
		{
			name: "temporary expired",
			user: loginTestUserWithMode(CredentialModeTemporary),
			temporary: loginTestTemporaryCredential(
				TemporaryCredentialExpired,
				time.Date(2026, time.October, 21, 9, 0, 0, 0, time.UTC),
			),
			password: "Temporary123",
		},
		{
			name:      "temporary superseded",
			user:      loginTestUserWithMode(CredentialModeTemporary),
			temporary: loginTestTemporaryCredential(TemporaryCredentialSuperseded, time.Time{}),
			password:  "Temporary123",
		},
		{
			name:      "temporary consumed",
			user:      loginTestUserWithMode(CredentialModeTemporary),
			temporary: loginTestTemporaryCredential(TemporaryCredentialConsumed, time.Time{}),
			password:  "Temporary123",
		},
		{
			name:      "temporary recovery invalidated",
			user:      loginTestUserWithMode(CredentialModeTemporary),
			temporary: loginTestTemporaryCredential(TemporaryCredentialRecoveryInvalidated, time.Time{}),
			password:  "Temporary123",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoginFixture(t)
			fixture.repository.user = test.user
			fixture.repository.userErr = test.userErr
			fixture.repository.temporary = test.temporary
			username := test.username
			if username == "" {
				username = "Alice"
			}

			_, err := fixture.service.Login(fixture.context, fixture.request(LoginRequest{
				Username:             username,
				Password:             test.password,
				SourceFingerprint:    "source-digest",
				ExistingSessionToken: "old-session-token",
			}))

			if got := platform.PublicCode(err); got != platform.CodeAuthLoginFailed {
				t.Fatalf("Login() public code = %q, want %q; error = %v", got, platform.CodeAuthLoginFailed, err)
			}
			if fixture.sessions.calls != 0 {
				t.Fatalf("session calls = %d, want none", fixture.sessions.calls)
			}
			if !fixture.transaction.oldSessionActive {
				t.Fatal("failed login changed the old session")
			}
			if fixture.abuse.failureCalls != 1 {
				t.Fatalf("abuse failure calls = %d, want 1", fixture.abuse.failureCalls)
			}
			if fixture.transaction.commitCalls != 1 || fixture.transaction.rollbackCalls != 0 {
				t.Fatalf(
					"transaction commit/rollback = %d/%d, want 1/0",
					fixture.transaction.commitCalls,
					fixture.transaction.rollbackCalls,
				)
			}
			if len(fixture.audit.events) != 1 ||
				fixture.audit.events[0].Outcome != loginAuditOutcomeFailed ||
				fixture.audit.events[0].ErrorCode == nil ||
				*fixture.audit.events[0].ErrorCode != string(platform.CodeAuthLoginFailed) {
				t.Fatalf("failure audit = %#v", fixture.audit.events)
			}
			for _, verifiedDigest := range fixture.hasher.verifiedDigests {
				if test.user.CredentialMode == CredentialModeTemporary &&
					test.user.PermanentPasswordDigest != nil &&
					verifiedDigest == *test.user.PermanentPasswordDigest {
					t.Fatal("temporary credential failure fell back to the permanent password")
				}
			}
		})
	}
}

func TestCredentialMode_TemporaryUsesOnlyActiveCurrentCredential(t *testing.T) {
	fixture := newLoginFixture(t)
	fixture.repository.user = loginTestUserWithMode(CredentialModeTemporary)
	fixture.repository.temporary = loginTestTemporaryCredential(
		TemporaryCredentialActive,
		time.Time{},
	)

	result, err := fixture.service.Login(fixture.context, fixture.request(LoginRequest{
		Username:             "Alice",
		Password:             "Temporary123",
		SourceFingerprint:    "source-digest",
		ExistingSessionToken: "old-session-token",
	}))
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.AuthenticatedCredential != SessionCredentialTemporary ||
		result.User.AuthenticatedCredentialID == nil ||
		*result.User.AuthenticatedCredentialID != fixture.repository.temporary.ID ||
		!result.User.MustChangePassword {
		t.Fatalf("authenticated user = %#v", result.User)
	}
	if len(fixture.hasher.verifiedDigests) != 1 ||
		fixture.hasher.verifiedDigests[0] != fixture.repository.temporary.SecretDigest {
		t.Fatalf("verified digests = %v, want only temporary digest", fixture.hasher.verifiedDigests)
	}
}

func TestLogin_SuccessRotatesSessionAndAuditFailureRollsBack(t *testing.T) {
	t.Run("success commits replacement", func(t *testing.T) {
		fixture := newLoginFixture(t)

		result, err := fixture.service.Login(fixture.context, fixture.request(LoginRequest{
			Username:             "Alice",
			Password:             "Password123",
			SourceFingerprint:    "source-digest",
			ExistingSessionToken: "old-session-token",
		}))
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if result.SessionToken != "new-session-token" || result.CSRFToken != "new-csrf-token" {
			t.Fatalf("session credentials = %q/%q", result.SessionToken, result.CSRFToken)
		}
		if fixture.sessions.rotateCalls != 1 || fixture.sessions.oldToken != "old-session-token" {
			t.Fatalf("session rotation = %d/%q", fixture.sessions.rotateCalls, fixture.sessions.oldToken)
		}
		if fixture.transaction.oldSessionActive {
			t.Fatal("successful login left the old session active")
		}
		if fixture.transaction.commitCalls != 1 || fixture.transaction.rollbackCalls != 0 {
			t.Fatalf(
				"transaction commit/rollback = %d/%d, want 1/0",
				fixture.transaction.commitCalls,
				fixture.transaction.rollbackCalls,
			)
		}
		if fixture.abuse.successCalls != 1 {
			t.Fatalf("abuse success calls = %d, want 1", fixture.abuse.successCalls)
		}
	})

	t.Run("audit failure preserves old session", func(t *testing.T) {
		fixture := newLoginFixture(t)
		fixture.audit.err = errors.New("synthetic audit failure")
		fixture.failureAudit.result = LoginAuditPersistedNo
		request := fixture.request(LoginRequest{
			Username:             "Alice",
			Password:             "Password123",
			SourceFingerprint:    "source-digest",
			ExistingSessionToken: "old-session-token",
		})

		_, err := fixture.service.Login(fixture.context, request)
		if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
			t.Fatalf("Login() public code = %q, want database unavailable; error = %v", got, err)
		}
		if !fixture.transaction.oldSessionActive {
			t.Fatal("rollback after audit failure did not preserve the old session")
		}
		if fixture.transaction.commitCalls != 0 || fixture.transaction.rollbackCalls != 1 {
			t.Fatalf(
				"transaction commit/rollback = %d/%d, want 0/1",
				fixture.transaction.commitCalls,
				fixture.transaction.rollbackCalls,
			)
		}
		if fixture.audit.calls != 1 || fixture.failureAudit.calls != 1 ||
			fixture.failureAudit.requests[0].Mode != LoginFailureAuditInsertFailure {
			t.Fatalf(
				"original/independent audit = %d/%#v",
				fixture.audit.calls,
				fixture.failureAudit.requests,
			)
		}
		observation := request.Attempt.observationSnapshot()
		if observation.FailureStage != LoginFailureStageAuditAppend ||
			observation.AuditPersisted != LoginAuditPersistedNo ||
			observation.RollbackOutcome != LoginRollbackSucceeded {
			t.Fatalf("observation = %#v", observation)
		}
	})
}

func TestLogin_InvalidSessionCookieTokenRollsBackBeforeAuditOrCommit(t *testing.T) {
	fixture := newLoginFixture(t)
	fixture.service.loginDependencies.SessionIssuer = loginSessionIssuerStub{
		now: fixture.now, token: "invalid\r\ncookie",
	}

	_, err := fixture.service.Login(fixture.context, fixture.request(LoginRequest{
		Username:             "Alice",
		Password:             "Password123",
		SourceFingerprint:    "source-digest",
		ExistingSessionToken: "old-session-token",
	}))

	if got := platform.PublicCode(err); got != platform.CodePlatformInternalError {
		t.Fatalf("Login() public code = %q, want internal error; error = %v", got, err)
	}
	if len(fixture.audit.events) != 0 {
		t.Fatalf("audit events = %#v, want none before token validation", fixture.audit.events)
	}
	if fixture.sessions.rotateCalls != 0 {
		t.Fatalf("session rotate calls = %d, want none", fixture.sessions.rotateCalls)
	}
	if fixture.transaction.commitCalls != 0 || fixture.transaction.rollbackCalls != 1 {
		t.Fatalf(
			"transaction commit/rollback = %d/%d, want 0/1",
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}
	if !fixture.transaction.oldSessionActive {
		t.Fatal("invalid replacement token revoked the old session")
	}
}

func TestLogin_AuditIncludesVersionedSubjectDigestWithoutRawUsername(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *loginFixture, *PersistentLoginAbuseProtector)
	}{
		{name: "known success"},
		{
			name: "known credential failure",
			setup: func(_ *testing.T, fixture *loginFixture, _ *PersistentLoginAbuseProtector) {
				fixture.hasher.matches[testCredentialDigest] = "Different123"
			},
		},
		{
			name: "unknown account",
			setup: func(_ *testing.T, fixture *loginFixture, _ *PersistentLoginAbuseProtector) {
				fixture.repository.userErr = pgx.ErrNoRows
			},
		},
		{
			name: "subject blocked",
			setup: func(
				t *testing.T,
				fixture *loginFixture,
				protector *PersistentLoginAbuseProtector,
			) {
				t.Helper()
				if err := protector.RecordFailure(
					fixture.context,
					fixture.transaction,
					LoginAttempt{
						Subject:           "alice",
						SourceFingerprint: "different-source-digest",
						Now:               fixture.now,
					},
				); err != nil {
					t.Fatalf("RecordFailure() error = %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoginFixture(t)
			digester, err := NewHMACLoginAbuseDigester(
				"login-key-v1",
				[]byte("synthetic-login-abuse-audit-key"),
			)
			if err != nil {
				t.Fatalf("NewHMACLoginAbuseDigester() error = %v", err)
			}
			protector, err := NewPersistentLoginAbuseProtector(
				newMemoryLoginAbuseStore(),
				digester,
				LoginAbuseConfig{
					Window:           10 * time.Minute,
					FailureThreshold: 1,
					BlockDuration:    5 * time.Minute,
					SuccessAction:    LoginAbuseSuccessClear,
				},
			)
			if err != nil {
				t.Fatalf("NewPersistentLoginAbuseProtector() error = %v", err)
			}
			fixture.service.loginDependencies.Abuse = protector
			if test.setup != nil {
				test.setup(t, fixture, protector)
			}

			_, loginErr := fixture.service.Login(fixture.context, fixture.request(LoginRequest{
				Username:          "Alice",
				Password:          "Password123",
				SourceFingerprint: "source-digest",
			}))
			if test.name == "known success" {
				if loginErr != nil {
					t.Fatalf("Login() error = %v", loginErr)
				}
			} else if got := platform.PublicCode(loginErr); got != platform.CodeAuthLoginFailed {
				t.Fatalf("Login() public code = %q, want login failed; error = %v", got, loginErr)
			}

			if len(fixture.audit.events) != 1 {
				t.Fatalf("audit events = %#v, want one", fixture.audit.events)
			}
			digest, version, err := digester.Digest(
				LoginBucketSubject,
				"alice",
				"source-digest",
			)
			if err != nil {
				t.Fatalf("Digest() error = %v", err)
			}
			assertLoginAuditSubjectEvidence(
				t,
				fixture.audit.events[0],
				hex.EncodeToString(digest),
				version,
			)
		})
	}
}

func TestLogin_MissingSubjectAuditEvidenceRollsBack(t *testing.T) {
	fixture := newLoginFixture(t)
	fixture.abuse.decision = LoginAbuseDecision{Kind: LoginAbuseAllowed}

	_, err := fixture.service.Login(fixture.context, fixture.request(LoginRequest{
		Username:             "Alice",
		Password:             "Password123",
		SourceFingerprint:    "source-digest",
		ExistingSessionToken: "old-session-token",
	}))

	if got := platform.PublicCode(err); got != platform.CodePlatformInternalError {
		t.Fatalf("Login() public code = %q, want internal error; error = %v", got, err)
	}
	if len(fixture.audit.events) != 0 {
		t.Fatalf("audit events = %#v, want none", fixture.audit.events)
	}
	if fixture.transaction.commitCalls != 0 || fixture.transaction.rollbackCalls != 1 {
		t.Fatalf(
			"transaction commit/rollback = %d/%d, want 0/1",
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}
	if !fixture.transaction.oldSessionActive {
		t.Fatal("missing audit evidence revoked the old session")
	}
}

func TestNewLoginService_FakeCredentialDigestMustMatchApprovedProbe(t *testing.T) {
	currentConfig := security.Argon2idConfig{
		MemoryKiB: 16, Iterations: 2, Parallelism: 1, SaltLength: 8, KeyLength: 16,
	}
	currentHasher := newLoginTestArgon2idHasher(t, currentConfig, 1)
	currentDigest, err := currentHasher.Hash(testFakeCredentialProbe)
	if err != nil {
		t.Fatalf("Hash(current fake credential) error = %v", err)
	}

	oldConfig := currentConfig
	oldConfig.MemoryKiB = 8
	oldConfig.Iterations = 1
	oldHasher := newLoginTestArgon2idHasher(t, oldConfig, 2)
	oldDigest, err := oldHasher.Hash(testFakeCredentialProbe)
	if err != nil {
		t.Fatalf("Hash(old fake credential) error = %v", err)
	}

	strongConfig := currentConfig
	strongConfig.MemoryKiB = 32
	strongConfig.Iterations = 3
	strongHasher := newLoginTestArgon2idHasher(t, strongConfig, 3)
	strongDigest, err := strongHasher.Hash(testFakeCredentialProbe)
	if err != nil {
		t.Fatalf("Hash(strong fake credential) error = %v", err)
	}

	tests := []struct {
		name    string
		digest  string
		probe   string
		wantErr bool
	}{
		{name: "current compatible digest", digest: currentDigest, probe: testFakeCredentialProbe},
		{name: "malformed digest", digest: "not-an-argon2id-encoding", probe: testFakeCredentialProbe, wantErr: true},
		{
			name: "unsupported algorithm",
			digest: strings.Replace(
				currentDigest,
				"$argon2id$",
				"$argon2i$",
				1,
			),
			probe: testFakeCredentialProbe, wantErr: true,
		},
		{
			name: "unsupported version",
			digest: strings.Replace(
				currentDigest,
				"$v=19$",
				"$v=16$",
				1,
			),
			probe: testFakeCredentialProbe, wantErr: true,
		},
		{name: "old parameters", digest: oldDigest, probe: testFakeCredentialProbe, wantErr: true},
		{name: "stronger parameters", digest: strongDigest, probe: testFakeCredentialProbe, wantErr: true},
		{name: "probe does not match", digest: currentDigest, probe: "DifferentSyntheticProbe123", wantErr: true},
		{name: "missing probe", digest: currentDigest, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewLoginService(loginConstructorDependencies(
				currentHasher,
				test.digest,
				test.probe,
			))
			if test.wantErr {
				if err == nil {
					t.Fatal("NewLoginService() error = nil, want fake credential rejection")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewLoginService() error = %v", err)
			}
			if service.loginDependencies.FakeCredentialProbe != "" {
				t.Fatal("NewLoginService() retained the fake credential probe")
			}
		})
	}
}

func TestNewLoginService_F005DoesNotOwnAttemptIDOrClock(t *testing.T) {
	hasher := &loginHasherStub{matches: map[string]string{
		"fake-digest": testFakeCredentialProbe,
	}}
	dependencies := loginConstructorDependencies(
		hasher,
		"fake-digest",
		testFakeCredentialProbe,
	)
	dependencies.IDs = nil
	dependencies.Clock = nil

	if _, err := NewLoginService(dependencies); err != nil {
		t.Fatalf("NewLoginService() required Handler-owned ID/clock dependencies: %v", err)
	}
}

func TestLoginAbuse_PersistentBucketsSurviveProtectorRecreation(t *testing.T) {
	now := time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC)
	store := newMemoryLoginAbuseStore()
	digester, err := NewHMACLoginAbuseDigester("key-v1", []byte("synthetic-login-abuse-key"))
	if err != nil {
		t.Fatalf("NewHMACLoginAbuseDigester() error = %v", err)
	}
	config := LoginAbuseConfig{
		Window:           10 * time.Minute,
		FailureThreshold: 2,
		BlockDuration:    5 * time.Minute,
		SuccessAction:    LoginAbuseSuccessClear,
	}
	first, err := NewPersistentLoginAbuseProtector(store, digester, config)
	if err != nil {
		t.Fatalf("NewPersistentLoginAbuseProtector(first) error = %v", err)
	}
	tx := newLoginTxStub()
	for _, source := range []string{"source-one", "source-two"} {
		if err := first.RecordFailure(context.Background(), tx, LoginAttempt{
			Subject: "alice", SourceFingerprint: source, Now: now,
		}); err != nil {
			t.Fatalf("RecordFailure(%s) error = %v", source, err)
		}
	}

	second, err := NewPersistentLoginAbuseProtector(store, digester, config)
	if err != nil {
		t.Fatalf("NewPersistentLoginAbuseProtector(second) error = %v", err)
	}
	decision, err := second.Check(context.Background(), tx, LoginAttempt{
		Subject: "alice", SourceFingerprint: "source-three", Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if decision.Kind != LoginAbuseSubjectBlocked ||
		decision.RetryAfter != 4*time.Minute {
		t.Fatalf("decision = %#v", decision)
	}
	if len(store.buckets) != 5 {
		t.Fatalf("persisted bucket count = %d, want 5", len(store.buckets))
	}
	for key := range store.buckets {
		if strings.Contains(key, "alice") || strings.Contains(key, "source-") {
			t.Fatalf("persistent bucket key leaked raw login material: %q", key)
		}
	}
}

func TestLoginAbuse_PostgresStoreLocksMissingBucketKeyBeforeRead(t *testing.T) {
	transaction := &loginBucketTxStub{loginTxStub: newLoginTxStub()}

	_, err := (PostgresLoginAbuseBucketStore{}).LoadForUpdate(
		context.Background(),
		transaction,
		LoginBucketSubject,
		[]byte("synthetic-bucket-digest"),
	)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("LoadForUpdate() error = %v, want pgx.ErrNoRows", err)
	}
	if len(transaction.operations) != 2 ||
		transaction.operations[0] != "exec" ||
		transaction.operations[1] != "query" {
		t.Fatalf("bucket lock/read operations = %v, want [exec query]", transaction.operations)
	}
	if transaction.execSQL != lockLoginAttemptBucketSQL {
		t.Fatalf("bucket lock SQL = %q", transaction.execSQL)
	}
}

func TestLoginHandler_SuccessSetsCookieAndFailureKeepsExistingCookie(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := &loginServiceStub{result: LoginResult{
			User: AuthenticatedUser{
				ID: "00000000-0000-4000-8000-000000000001", Username: "Alice",
				Role: RoleMember, Status: UserStatusActive,
				AuthenticatedCredential: SessionCredentialPermanent,
			},
			SessionToken: "new-session-token",
			CSRFToken:    "new-csrf-token",
		}}
		handler, err := NewLoginHandler(
			service,
			registrationSourceStub{evidence: RegistrationSourceEvidence{
				Kind: "trusted-source", Fingerprint: "source-digest",
			}},
			4096,
			newLoginHandlerTestConfig(t),
		)
		if err != nil {
			t.Fatalf("NewLoginHandler() error = %v", err)
		}
		request := newLoginHTTPRequest(`{"username":"Alice","password":"Password123"}`)
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "old-session-token"})
		recorder := httptest.NewRecorder()

		platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
		}
		if service.request.ExistingSessionToken != "old-session-token" ||
			service.request.SourceFingerprint != "source-digest" {
			t.Fatalf("trusted login request = %#v", service.request)
		}
		cookies := recorder.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != platform.SessionCookieName ||
			cookies[0].Value != "new-session-token" {
			t.Fatalf("cookies = %#v", cookies)
		}
		var response loginResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode login response: %v", err)
		}
		if response.CSRFToken != "new-csrf-token" ||
			response.User.ID != service.result.User.ID {
			t.Fatalf("response = %#v", response)
		}
		if strings.Contains(recorder.Body.String(), "Password123") ||
			strings.Contains(recorder.Body.String(), "new-session-token") {
			t.Fatal("login response leaked credential material")
		}
	})

	t.Run("failure", func(t *testing.T) {
		service := &loginServiceStub{err: platform.NewPublicError(
			platform.CodeAuthLoginFailed,
			errors.New("synthetic credential mismatch"),
		)}
		handler, err := NewLoginHandler(
			service,
			registrationSourceStub{evidence: RegistrationSourceEvidence{
				Kind: "trusted-source", Fingerprint: "source-digest",
			}},
			4096,
			newLoginHandlerTestConfig(t),
		)
		if err != nil {
			t.Fatalf("NewLoginHandler() error = %v", err)
		}
		request := newLoginHTTPRequest(`{"username":"Alice","password":"Wrong123"}`)
		request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "old-session-token"})
		recorder := httptest.NewRecorder()

		platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body = %s", recorder.Code, recorder.Body)
		}
		if len(recorder.Header().Values("Set-Cookie")) != 0 {
			t.Fatal("failed login replaced the existing session cookie")
		}
	})

	t.Run("source rate limit", func(t *testing.T) {
		service := &loginServiceStub{err: blockedLoginError(LoginAbuseDecision{
			Kind:       LoginAbuseSourceBlocked,
			RetryAfter: 1500 * time.Millisecond,
		})}
		handler, err := NewLoginHandler(
			service,
			registrationSourceStub{evidence: RegistrationSourceEvidence{
				Kind: "trusted-source", Fingerprint: "source-digest",
			}},
			4096,
			newLoginHandlerTestConfig(t),
		)
		if err != nil {
			t.Fatalf("NewLoginHandler() error = %v", err)
		}
		recorder := httptest.NewRecorder()

		platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
			recorder,
			newLoginHTTPRequest(`{"username":"Alice","password":"Wrong123"}`),
		)

		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429; body = %s", recorder.Code, recorder.Body)
		}
		if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != "2" {
			t.Fatalf("Retry-After = %q, want 2", retryAfter)
		}
		if len(recorder.Header().Values("Set-Cookie")) != 0 {
			t.Fatal("rate-limited login set a session cookie")
		}
	})
}

func TestLoginHandler_Argon2CapacityReturnsFixed429WithoutAbuseMutation(t *testing.T) {
	currentConfig := security.Argon2idConfig{
		MemoryKiB: 16, Iterations: 2, Parallelism: 1, SaltLength: 8, KeyLength: 16,
	}
	oldConfig := currentConfig
	oldConfig.MemoryKiB = 8
	oldConfig.Iterations = 1
	fakeDigest, err := newLoginTestArgon2idHasher(t, currentConfig, 0x31).
		Hash(testFakeCredentialProbe)
	if err != nil {
		t.Fatalf("Hash(fake) error = %v", err)
	}
	userDigest, err := newLoginTestArgon2idHasher(t, oldConfig, 0x32).
		Hash("Password123")
	if err != nil {
		t.Fatalf("Hash(user) error = %v", err)
	}
	temporaryDigest, err := newLoginTestArgon2idHasher(t, oldConfig, 0x33).
		Hash("Temporary123")
	if err != nil {
		t.Fatalf("Hash(temporary) error = %v", err)
	}
	tests := []struct {
		name          string
		password      string
		setup         func(*loginRepositoryStub)
		rehashDigest  string
		startOccupied bool
	}{
		{
			name:     "known permanent verify",
			password: "Password123",
			setup: func(repository *loginRepositoryStub) {
				user := loginTestUser()
				user.PermanentPasswordDigest = &userDigest
				repository.user = user
			},
			startOccupied: true,
		},
		{
			name:     "unknown user fake verify",
			password: "Password123",
			setup: func(repository *loginRepositoryStub) {
				repository.userErr = pgx.ErrNoRows
			},
			startOccupied: true,
		},
		{
			name:     "temporary credential rehash",
			password: "Temporary123",
			setup: func(repository *loginRepositoryStub) {
				repository.user = loginTestUserWithMode(CredentialModeTemporary)
				credential := loginTestTemporaryCredential(TemporaryCredentialActive, time.Time{})
				credential.SecretDigest = temporaryDigest
				credential.IssuedAt = time.Date(2026, time.October, 23, 8, 0, 0, 0, time.UTC)
				credential.ExpiresAt = credential.IssuedAt.Add(temporaryCredentialLifetime)
				repository.temporary = credential
			},
			rehashDigest: temporaryDigest,
		},
		{
			name:     "permanent credential rehash",
			password: "Password123",
			setup: func(repository *loginRepositoryStub) {
				user := loginTestUser()
				user.PermanentPasswordDigest = &userDigest
				repository.user = user
			},
			rehashDigest: userDigest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			base, err := security.NewArgon2idHasher(
				currentConfig,
				&loginBlockingReader{entered: entered, release: release},
			)
			if err != nil {
				t.Fatalf("NewArgon2idHasher(runtime) error = %v", err)
			}
			gate, err := security.NewArgon2idProcessGate(1)
			if err != nil {
				t.Fatalf("NewArgon2idProcessGate() error = %v", err)
			}
			passwords, err := security.NewGatedArgon2idHasher(base, gate)
			if err != nil {
				t.Fatalf("NewGatedArgon2idHasher() error = %v", err)
			}
			holderDone := make(chan error, 1)
			holderStarted := false
			startHolder := func() {
				if holderStarted {
					t.Fatal("permit holder started more than once")
				}
				holderStarted = true
				go func() {
					_, hashErr := passwords.Hash("synthetic-permit-holder")
					holderDone <- hashErr
				}()
				<-entered
			}
			defer func() {
				if holderStarted {
					close(release)
					if err := <-holderDone; err != nil {
						t.Errorf("permit holder error = %v", err)
					}
				}
			}()

			var servicePasswords LoginPasswordHasher = passwords
			if test.rehashDigest != "" {
				servicePasswords = &loginRehashBarrierHasher{
					delegate: passwords,
					digest:   test.rehashDigest,
					onRehash: startHolder,
				}
			}
			transaction := newLoginTxStub()
			repository := &loginRepositoryStub{}
			test.setup(repository)
			now := time.Date(2026, time.October, 23, 9, 0, 0, 0, time.UTC)
			abuse := &loginAbuseStub{decision: LoginAbuseDecision{
				SubjectDigest: []byte("subject-digest"), DigestKeyVersion: "v1",
			}}
			sessions := &loginSessionStoreStub{}
			auditAppender := &loginAuditStub{}
			service, err := NewLoginService(LoginDependencies{
				Transactions:         &loginTransactorStub{transaction: transaction},
				Repository:           repository,
				PasswordResetGate:    &loginPasswordResetGateStub{},
				Passwords:            servicePasswords,
				Sessions:             sessions,
				SessionIssuer:        loginSessionIssuerStub{now: now},
				Abuse:                abuse,
				RecoveryMode:         loginRecoveryModeStub{},
				Audit:                auditAppender,
				IDs:                  &loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000099"}},
				Clock:                loginClockStub{now: now},
				FakeCredentialDigest: fakeDigest,
				FakeCredentialProbe:  testFakeCredentialProbe,
			})
			if err != nil {
				t.Fatalf("NewLoginService() error = %v", err)
			}
			failureAuditor := &loginFailureAuditorStub{}
			coordinator, err := NewLoginAttemptCoordinator(
				&loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000098"}},
				loginClockStub{now: now},
				failureAuditor,
			)
			if err != nil {
				t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
			}
			observer := &loginOutcomeObserverStub{}
			handler, err := NewLoginHandler(
				service,
				registrationSourceStub{evidence: RegistrationSourceEvidence{
					Kind: "trusted-source", Fingerprint: "source-digest",
				}},
				4096,
				LoginHandlerConfig{
					Coordinator: coordinator, Observer: observer, ObservationBudget: time.Second,
				},
			)
			if err != nil {
				t.Fatalf("NewLoginHandler() error = %v", err)
			}
			if test.startOccupied {
				startHolder()
			}

			recorder := &loginResponseWriteCounter{ResponseRecorder: httptest.NewRecorder()}
			platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
				recorder,
				newLoginHTTPRequest(
					`{"username":"Alice","password":"`+test.password+`"}`,
				),
			)
			if !holderStarted {
				t.Fatal("real credential consumer did not reach the rehash barrier")
			}
			if recorder.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want 429", recorder.Code)
			}
			if got := recorder.Header().Get("Retry-After"); got != "1" {
				t.Fatalf("Retry-After = %q, want 1", got)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if envelope.Code != platform.CodeRequestRateLimited {
				t.Fatalf("response code = %q, want %q", envelope.Code, platform.CodeRequestRateLimited)
			}
			if recorder.headerWrites != 1 || recorder.bodyWrites != 1 {
				t.Fatalf(
					"response header/body writes = %d/%d, want 1/1",
					recorder.headerWrites,
					recorder.bodyWrites,
				)
			}
			if abuse.failureCalls != 0 || abuse.successCalls != 0 {
				t.Fatalf("abuse mutations = failure:%d success:%d, want 0/0", abuse.failureCalls, abuse.successCalls)
			}
			if transaction.commitCalls != 0 || transaction.rollbackCalls != 1 {
				t.Fatalf("transaction commit/rollback = %d/%d, want 0/1", transaction.commitCalls, transaction.rollbackCalls)
			}
			if auditAppender.calls != 0 || failureAuditor.calls != 1 {
				t.Fatalf(
					"transactional/independent audit calls = %d/%d, want 0/1",
					auditAppender.calls,
					failureAuditor.calls,
				)
			}
			if sessions.calls != 0 ||
				repository.permanentRehashes != 0 ||
				repository.temporaryRehashes != 0 {
				t.Fatalf(
					"session/permanent-rehash/temporary-rehash calls = %d/%d/%d, want 0/0/0",
					sessions.calls,
					repository.permanentRehashes,
					repository.temporaryRehashes,
				)
			}
			if len(observer.observed) != 1 ||
				observer.observed[0].Outcome != LoginOutcomeRateLimited ||
				observer.observed[0].ReasonBucket != "resource_capacity" {
				t.Fatalf("observations = %#v", observer.observed)
			}
			_, _, capacityErr := passwords.Verify(testFakeCredentialProbe, fakeDigest)
			var capacity *security.Argon2idCapacityError
			if !errors.As(capacityErr, &capacity) || capacity.RetryAfter() != time.Second {
				t.Fatalf("shared gate error = %v, want typed 1s capacity", capacityErr)
			}
			for _, denied := range []string{
				"Alice", test.password, "source-digest", fakeDigest, userDigest,
				temporaryDigest, "subject-digest",
			} {
				if strings.Contains(recorder.Body.String(), denied) {
					t.Fatal("capacity response exposed sensitive input")
				}
			}

			close(release)
			holderStarted = false
			if err := <-holderDone; err != nil {
				t.Fatalf("permit holder error = %v", err)
			}
			if _, _, err := passwords.Verify(testFakeCredentialProbe, fakeDigest); err != nil {
				t.Fatalf("Verify() after capacity release error = %v", err)
			}
		})
	}
}

func TestTrustedSourceEvidenceProvider_StrictProxyResolution(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	provider, err := NewTrustedSourceEvidenceProvider(
		key,
		"source_v1",
		[]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
	)
	if err != nil {
		t.Fatalf("NewTrustedSourceEvidenceProvider() error = %v", err)
	}
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  []string
		wantErr    bool
	}{
		{
			name:       "untrusted peer ignores hostile forwarded header",
			remoteAddr: "198.51.100.8:443",
			forwarded:  []string{"not-an-address"},
		},
		{
			name:       "trusted peer resolves rightmost non trusted client",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"203.0.113.9, 192.0.2.11"},
		},
		{
			name:       "multiple headers preserve wire order",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"203.0.113.9", "198.51.100.7,192.0.2.11"},
		},
		{
			name:       "sixteen forwarded hops accepted",
			remoteAddr: "192.0.2.10:443",
			forwarded: []string{
				"203.0.113.9,192.0.2.1,192.0.2.2,192.0.2.3,192.0.2.4,192.0.2.5,192.0.2.6,192.0.2.7",
				"192.0.2.8,192.0.2.9,192.0.2.11,192.0.2.12,192.0.2.13,192.0.2.14,192.0.2.15,192.0.2.16",
			},
		},
		{
			name:       "seventeen forwarded hops rejected",
			remoteAddr: "192.0.2.10:443",
			forwarded: []string{
				"203.0.113.9,192.0.2.1,192.0.2.2,192.0.2.3,192.0.2.4,192.0.2.5,192.0.2.6,192.0.2.7",
				"192.0.2.8,192.0.2.9,192.0.2.11,192.0.2.12,192.0.2.13,192.0.2.14,192.0.2.15,192.0.2.16,192.0.2.17",
			},
			wantErr: true,
		},
		{
			name:       "mapped peer is canonicalized",
			remoteAddr: "[::ffff:198.51.100.8]:443",
		},
		{name: "bare peer rejected", remoteAddr: "198.51.100.8", wantErr: true},
		{name: "zoned peer rejected", remoteAddr: "[fe80::1%eth0]:443", wantErr: true},
		{
			name:       "trusted peer without forwarded hops rejected",
			remoteAddr: "192.0.2.10:443",
			wantErr:    true,
		},
		{
			name:       "trusted peer rejects empty forwarded item",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"203.0.113.9,,192.0.2.11"},
			wantErr:    true,
		},
		{
			name:       "trusted peer rejects malformed forwarded item",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"not-an-address"},
			wantErr:    true,
		},
		{
			name:       "forwarded port rejected",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"203.0.113.9:443"},
			wantErr:    true,
		},
		{
			name:       "forwarded zone rejected",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"fe80::1%eth0"},
			wantErr:    true,
		},
		{
			name:       "forwarded brackets rejected",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"[2001:db8::1]"},
			wantErr:    true,
		},
		{
			name:       "all trusted chain rejected",
			remoteAddr: "192.0.2.10:443",
			forwarded:  []string{"192.0.2.11"},
			wantErr:    true,
		},
	}
	var directFingerprint string
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil)
			request.RemoteAddr = test.remoteAddr
			for _, value := range test.forwarded {
				request.Header.Add("X-Forwarded-For", value)
			}
			evidence, err := provider.Evidence(request)
			if test.wantErr {
				if err == nil {
					t.Fatal("Evidence() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Evidence() error = %v", err)
			}
			if evidence.Kind != trustedSourceKind ||
				!strings.HasPrefix(evidence.Fingerprint, "source_v1:") {
				t.Fatalf("Evidence() returned invalid stable shape")
			}
			if test.name == "untrusted peer ignores hostile forwarded header" {
				directFingerprint = evidence.Fingerprint
			}
			if test.name == "mapped peer is canonicalized" &&
				evidence.Fingerprint != directFingerprint {
				t.Fatal("mapped IPv6 and IPv4 did not canonicalize to the same digest")
			}
		})
	}
	direct := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil)
	direct.RemoteAddr = "198.51.100.7:443"
	directEvidence, err := provider.Evidence(direct)
	if err != nil {
		t.Fatalf("direct Evidence() error = %v", err)
	}
	wireOrdered := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil)
	wireOrdered.RemoteAddr = "192.0.2.10:443"
	wireOrdered.Header.Add("X-Forwarded-For", "203.0.113.9")
	wireOrdered.Header.Add("X-Forwarded-For", "198.51.100.7,192.0.2.11")
	wireEvidence, err := provider.Evidence(wireOrdered)
	if err != nil {
		t.Fatalf("wire-order Evidence() error = %v", err)
	}
	if wireEvidence.Fingerprint != directEvidence.Fingerprint {
		t.Fatal("multiple X-Forwarded-For headers did not preserve wire order")
	}
}

func TestTrustedSourceEvidenceProvider_TrustedPrefixBoundaries(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	sixtyFour := make([]netip.Prefix, 64)
	for index := range sixtyFour {
		sixtyFour[index] = netip.PrefixFrom(
			netip.AddrFrom4([4]byte{10, byte(index), 0, 0}),
			16,
		)
	}
	tests := []struct {
		name     string
		prefixes []netip.Prefix
		wantErr  bool
	}{
		{name: "zero prefixes"},
		{name: "one prefix", prefixes: sixtyFour[:1]},
		{name: "sixty four prefixes", prefixes: sixtyFour},
		{name: "sixty five prefixes", prefixes: append(sixtyFour, netip.MustParsePrefix("192.0.2.0/24")), wantErr: true},
		{name: "ipv4 trust all", prefixes: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}, wantErr: true},
		{name: "ipv6 trust all", prefixes: []netip.Prefix{netip.MustParsePrefix("::/0")}, wantErr: true},
		{
			name: "mapped duplicate equivalence",
			prefixes: []netip.Prefix{
				netip.MustParsePrefix("192.0.2.0/24"),
				netip.MustParsePrefix("::ffff:192.0.2.0/120"),
			},
			wantErr: true,
		},
		{name: "host bits", prefixes: []netip.Prefix{netip.MustParsePrefix("192.0.2.1/24")}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewTrustedSourceEvidenceProvider(key, "source_v1", test.prefixes)
			if test.wantErr && err == nil {
				t.Fatal("NewTrustedSourceEvidenceProvider() error = nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("NewTrustedSourceEvidenceProvider() error = %v", err)
			}
		})
	}
}

func TestRandomLoginSessionIssuer_UsesExplicitEntropyAndOneClockValue(t *testing.T) {
	now := time.Date(2026, time.October, 23, 8, 0, 0, 0, time.FixedZone("fixture", 8*60*60))
	random := bytes.NewReader(bytes.Repeat([]byte{0x5a}, 64))
	issuer, err := NewRandomLoginSessionIssuer(
		random,
		&loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000123"}},
		loginClockStub{now: now},
		32,
		32,
		24*time.Hour,
		30*time.Minute,
	)
	if err != nil {
		t.Fatalf("NewRandomLoginSessionIssuer() error = %v", err)
	}
	input, err := issuer.Issue(loginTestUser(), session.CredentialPermanent, nil)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	token, err := base64.RawURLEncoding.Strict().DecodeString(input.Token)
	if err != nil || len(token) != 32 {
		t.Fatalf("session token is not 32-byte raw base64url")
	}
	csrf, err := base64.RawURLEncoding.Strict().DecodeString(input.CSRFToken)
	if err != nil || len(csrf) != 32 {
		t.Fatalf("CSRF token is not 32-byte raw base64url")
	}
	if input.IssuedAt != now.UTC() ||
		input.AbsoluteExpiresAt != now.UTC().Add(24*time.Hour) ||
		input.IdleExpiresAt != now.UTC().Add(30*time.Minute) {
		t.Fatalf("session times = %#v", input)
	}
}

func TestLoginHandler_F005CoordinatesFailureAuditBeforeSourceAndObservesAfterWrite(t *testing.T) {
	order := make([]string, 0, 8)
	auditor := &loginFailureAuditorStub{onRecord: func(request LoginFailureAuditRequest) {
		order = append(order, "failure-audit")
		if request.Event.ID != "00000000-0000-4000-8000-000000000088" {
			t.Fatalf("failure audit ID = %q", request.Event.ID)
		}
	}}
	coordinator, err := NewLoginAttemptCoordinator(
		orderedLoginIDStub{
			id: "00000000-0000-4000-8000-000000000088",
			onNew: func() {
				order = append(order, "id")
			},
		},
		orderedLoginClockStub{
			now: time.Date(2026, time.October, 22, 11, 0, 0, 0, time.UTC),
			onNow: func() {
				order = append(order, "clock")
			},
		},
		auditor,
	)
	if err != nil {
		t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
	}
	observer := &loginOutcomeObserverStub{onObserve: func(
		ctx context.Context,
		observation LoginObservation,
	) error {
		order = append(order, "observe")
		if ctx.Err() != nil {
			t.Fatalf("observation context error = %v", ctx.Err())
		}
		if observation.Outcome != LoginOutcomeInternalFailure ||
			observation.FailureStage != LoginFailureStageSourceEvidence ||
			observation.AuditPersisted != LoginAuditPersistedNo ||
			observation.ResponseWrite != LoginResponseWriteOK {
			t.Fatalf("observation = %#v", observation)
		}
		return nil
	}}
	handler, err := NewLoginHandler(
		&loginServiceStub{},
		orderedLoginSourceStub{
			err: errors.New("synthetic source failure"),
			onEvidence: func() {
				order = append(order, "source")
			},
		},
		4096,
		LoginHandlerConfig{
			Coordinator:       coordinator,
			Observer:          observer,
			ObservationBudget: time.Second,
		},
	)
	if err != nil {
		t.Fatalf("NewLoginHandler() error = %v", err)
	}
	recorder := &orderedResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		onWrite: func() {
			order = append(order, "write")
		},
	}

	platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
		recorder,
		newLoginHTTPRequest(`{"username":"Alice","password":"Password123"}`),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	want := []string{"id", "clock", "source", "failure-audit", "write", "observe"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", order, want)
	}
	if auditor.calls != 1 || observer.calls != 1 {
		t.Fatalf("failure audit/observer calls = %d/%d, want 1/1", auditor.calls, observer.calls)
	}
}

func TestLoginAttemptCoordinator_NormalizesOccurredAtToUTCMicroseconds(t *testing.T) {
	location := time.FixedZone("synthetic-offset", 8*60*60)
	raw := time.Date(2026, time.October, 22, 19, 0, 0, 123456789, location)
	coordinator, err := NewLoginAttemptCoordinator(
		&loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000089"}},
		loginClockStub{now: raw},
		&loginFailureAuditorStub{},
	)
	if err != nil {
		t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
	}

	attempt, err := coordinator.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	want := raw.UTC().Truncate(time.Microsecond)
	if !attempt.OccurredAt.Equal(want) || attempt.OccurredAt.Location() != time.UTC {
		t.Fatalf("OccurredAt = %s, want UTC microsecond %s", attempt.OccurredAt, want)
	}
	if attempt.OccurredAt.Nanosecond()%int(time.Microsecond) != 0 {
		t.Fatalf("OccurredAt nanoseconds = %d, want whole microseconds", attempt.OccurredAt.Nanosecond())
	}
}

func TestLoginAttemptCoordinator_F005AllowsOnlyOnePhaseOwnerClaim(t *testing.T) {
	auditor := &loginFailureAuditorStub{}
	coordinator, err := NewLoginAttemptCoordinator(
		&loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000087"}},
		loginClockStub{now: time.Date(2026, time.October, 22, 11, 0, 0, 0, time.UTC)},
		auditor,
	)
	if err != nil {
		t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
	}
	attempt, err := coordinator.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	request := LoginFailureAuditRequest{
		Mode: LoginFailureAuditInsertFailure,
		Event: audit.Event{
			ID:         attempt.ID,
			OccurredAt: attempt.OccurredAt,
		},
	}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, recordErr := attempt.recordOnce(
				context.Background(),
				loginAttemptPreService,
				request,
			)
			results <- recordErr
		}()
	}
	var failures int
	for range 2 {
		if err := <-results; err != nil {
			failures++
		}
	}
	if auditor.calls != 1 || failures != 1 {
		t.Fatalf("auditor calls/guard failures = %d/%d, want 1/1", auditor.calls, failures)
	}
	if err := attempt.enterService(); err != nil {
		t.Fatalf("enterService() error = %v", err)
	}
	if _, err := attempt.recordOnce(
		context.Background(),
		loginAttemptServiceEntered,
		request,
	); err == nil {
		t.Fatal("service phase obtained a second independent audit claim")
	}
	if auditor.calls != 1 {
		t.Fatalf("auditor calls after phase transition = %d, want 1", auditor.calls)
	}
}

func TestLoginOutcomeObserverAdapter_StopsFanoutOnCancellation(t *testing.T) {
	logger := &loginEventLoggerStub{blockUntilCanceled: true}
	metrics := &loginMetricsStub{}
	observer, err := NewLoginOutcomeObserver(logger, metrics)
	if err != nil {
		t.Fatalf("NewLoginOutcomeObserver() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = observer.ObserveFinal(ctx, LoginObservation{
		Outcome: LoginOutcomeInternalFailure,
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ObserveFinal() error = %v, want context canceled", err)
	}
	if logger.calls != 0 || metrics.calls != 0 {
		t.Fatalf("logger/metrics calls = %d/%d, want 0/0", logger.calls, metrics.calls)
	}
}

func TestLogin_F005ClassifiesTransactionUncertaintyWithoutBlindAuditWrites(t *testing.T) {
	t.Run("rollback unknown before append performs no independent audit operation", func(t *testing.T) {
		fixture := newLoginFixture(t)
		fixture.abuse.checkErr = errors.New("synthetic abuse store failure")
		fixture.transaction.rollbackErr = errors.New("synthetic rollback uncertainty")
		request := fixture.request(LoginRequest{
			Username:          "Alice",
			Password:          "Password123",
			SourceFingerprint: "source-digest",
		})

		_, err := fixture.service.Login(fixture.context, request)

		if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
			t.Fatalf("Login() public code = %q, want database unavailable", got)
		}
		if fixture.failureAudit.calls != 0 {
			t.Fatalf("independent audit calls = %d, want 0", fixture.failureAudit.calls)
		}
		observation := request.Attempt.observationSnapshot()
		if observation.TransactionOutcome != LoginTransactionUnknown ||
			observation.RollbackOutcome != LoginRollbackUnknown ||
			observation.AuditPersisted != LoginAuditPersistedUnknown {
			t.Fatalf("observation = %#v", observation)
		}
	})

	t.Run("rollback unknown after append only verifies prepared event", func(t *testing.T) {
		fixture := newLoginFixture(t)
		fixture.audit.err = errors.New("synthetic append uncertainty")
		fixture.transaction.rollbackErr = errors.New("synthetic rollback uncertainty")
		fixture.failureAudit.result = LoginAuditPersistedYes
		request := fixture.request(LoginRequest{
			Username:          "Alice",
			Password:          "Password123",
			SourceFingerprint: "source-digest",
		})

		_, err := fixture.service.Login(fixture.context, request)

		if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
			t.Fatalf("Login() public code = %q, want database unavailable", got)
		}
		if fixture.failureAudit.calls != 1 ||
			fixture.failureAudit.requests[0].Mode != LoginFailureAuditVerifyPrepared {
			t.Fatalf("independent audit requests = %#v", fixture.failureAudit.requests)
		}
		observation := request.Attempt.observationSnapshot()
		if observation.TransactionOutcome != LoginTransactionUnknown ||
			observation.RollbackOutcome != LoginRollbackUnknown ||
			observation.AuditPersisted != LoginAuditPersistedYes {
			t.Fatalf("observation = %#v", observation)
		}
	})

	for _, persistence := range []LoginAuditPersistence{
		LoginAuditPersistedYes,
		LoginAuditPersistedUnknown,
	} {
		t.Run("commit unknown verifies prepared event as "+string(persistence), func(t *testing.T) {
			fixture := newLoginFixture(t)
			fixture.transaction.commitErr = errors.New("synthetic commit uncertainty")
			fixture.failureAudit.result = persistence
			request := fixture.request(LoginRequest{
				Username:          "Alice",
				Password:          "Password123",
				SourceFingerprint: "source-digest",
			})

			_, err := fixture.service.Login(fixture.context, request)

			if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
				t.Fatalf("Login() public code = %q, want database unavailable", got)
			}
			if fixture.failureAudit.calls != 1 ||
				fixture.failureAudit.requests[0].Mode != LoginFailureAuditVerifyPrepared {
				t.Fatalf("independent audit requests = %#v", fixture.failureAudit.requests)
			}
			observation := request.Attempt.observationSnapshot()
			if observation.TransactionOutcome != LoginTransactionUnknown ||
				observation.RollbackOutcome != LoginRollbackNotAttempted ||
				observation.AuditPersisted != persistence {
				t.Fatalf("observation = %#v", observation)
			}
		})
	}
}

func TestLogin_F005DatabaseBeginFailureUsesOneIndependentAuditAttempt(t *testing.T) {
	fixture := newLoginFixture(t)
	fixture.service.loginDependencies.Transactions = &loginTransactorStub{
		err: errors.New("synthetic begin failure"),
	}
	fixture.failureAudit.result = LoginAuditPersistedUnknown
	request := fixture.request(LoginRequest{
		Username:          "Alice",
		Password:          "Password123",
		SourceFingerprint: "source-digest",
	})

	_, err := fixture.service.Login(fixture.context, request)

	if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("Login() public code = %q, want database unavailable", got)
	}
	if fixture.audit.calls != 0 || fixture.failureAudit.calls != 1 ||
		fixture.failureAudit.requests[0].Mode != LoginFailureAuditInsertFailure {
		t.Fatalf(
			"original/independent audit = %d/%#v",
			fixture.audit.calls,
			fixture.failureAudit.requests,
		)
	}
	observation := request.Attempt.observationSnapshot()
	if observation.FailureStage != LoginFailureStageTransactionBegin ||
		observation.TransactionOutcome != LoginTransactionNotStarted ||
		observation.AuditPersisted != LoginAuditPersistedUnknown ||
		observation.RollbackOutcome != LoginRollbackNotAttempted {
		t.Fatalf("observation = %#v", observation)
	}
}

func TestLogin_F005ClassifiesInternalFailureStages(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*loginFixture)
		wantStage LoginFailureStage
	}{
		{
			name: "temporary credential query",
			setup: func(fixture *loginFixture) {
				fixture.repository.user = loginTestUserWithMode(CredentialModeTemporary)
				fixture.repository.temporaryErr = errors.New("synthetic temporary query failure")
			},
			wantStage: LoginFailureStageTemporaryCredentialQuery,
		},
		{
			name: "credential verify",
			setup: func(fixture *loginFixture) {
				fixture.hasher.err = errors.New("synthetic credential verify failure")
			},
			wantStage: LoginFailureStageCredentialVerify,
		},
		{
			name: "credential rehash",
			setup: func(fixture *loginFixture) {
				fixture.hasher.needsRehash = true
				fixture.hasher.hashErr = errors.New("synthetic credential rehash failure")
			},
			wantStage: LoginFailureStageCredentialRehash,
		},
		{
			name: "recovery mode",
			setup: func(fixture *loginFixture) {
				user := loginTestUserWithMode(CredentialModeRecoveryBootstrap)
				user.IsInitialAdmin = true
				user.RecoveryRequired = true
				fixture.repository.user = user
				fixture.service.loginDependencies.RecoveryMode = loginRecoveryModeStub{
					err: errors.New("synthetic recovery mode failure"),
				}
			},
			wantStage: LoginFailureStageRecoveryMode,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoginFixture(t)
			test.setup(fixture)
			request := fixture.request(LoginRequest{
				Username:          "Alice",
				Password:          "Password123",
				SourceFingerprint: "source-digest",
			})

			_, err := fixture.service.Login(fixture.context, request)

			if err == nil {
				t.Fatal("Login() error = nil, want internal failure")
			}
			if got := request.Attempt.observationSnapshot().FailureStage; got != test.wantStage {
				t.Fatalf("failure stage = %q, want %q", got, test.wantStage)
			}
		})
	}
}

func TestLoginRecoveryBootstrap_RequiresRecoveryStateBeforeAdapterOrVerify(t *testing.T) {
	fixture := newLoginFixture(t)
	user := loginTestUserWithMode(CredentialModeRecoveryBootstrap)
	user.IsInitialAdmin = true
	user.RecoveryRequired = false
	fixture.repository.user = user
	recoveryCalls := 0
	fixture.service.loginDependencies.RecoveryMode = loginRecoveryModeStub{
		allowed: true,
		calls:   &recoveryCalls,
	}

	_, err := fixture.service.Login(
		fixture.context,
		fixture.request(LoginRequest{
			Username:          "Alice",
			Password:          "Password123",
			SourceFingerprint: "source-digest",
		}),
	)
	if platform.PublicCode(err) != platform.CodeAuthLoginFailed {
		t.Fatalf("Login() public code = %q, want %q", platform.PublicCode(err), platform.CodeAuthLoginFailed)
	}
	if recoveryCalls != 0 {
		t.Fatalf("RecoveryBootstrapAllowed() calls = %d, want 0", recoveryCalls)
	}
	if len(fixture.hasher.verifiedDigests) != 0 {
		t.Fatalf("credential Verify() calls = %d, want 0", len(fixture.hasher.verifiedDigests))
	}
	if fixture.audit.calls != 1 || fixture.sessions.calls != 0 {
		t.Fatalf("audit/session calls = %d/%d, want 1/0", fixture.audit.calls, fixture.sessions.calls)
	}
	if fixture.transaction.commitCalls != 1 || fixture.transaction.rollbackCalls != 0 {
		t.Fatalf(
			"commit/rollback calls = %d/%d, want 1/0",
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}
}

func TestLoginRecoveryBootstrap_FourWayConjunctionAndAdapterResults(t *testing.T) {
	tests := []struct {
		name             string
		mutateUser       func(*User)
		allowed          bool
		adapterErr       error
		wantAdapter      int
		wantVerify       int
		wantSessions     int
		wantAudit        int
		wantFailureAudit int
		wantCommit       int
		wantRollback     int
		wantSuccess      bool
		wantPublicCode   platform.Code
	}{
		{
			name: "not initial admin",
			mutateUser: func(user *User) {
				user.IsInitialAdmin = false
			},
			wantAudit:  1,
			wantCommit: 1,
		},
		{
			name: "recovery not required",
			mutateUser: func(user *User) {
				user.RecoveryRequired = false
			},
			wantAudit:  1,
			wantCommit: 1,
		},
		{
			name:        "normal mode",
			wantAdapter: 1,
			wantAudit:   1,
			wantCommit:  1,
		},
		{
			name:        "static maintenance bool is not authority",
			allowed:     true,
			wantAdapter: 1,
			wantAudit:   1,
			wantCommit:  1,
		},
		{
			name:             "adapter error",
			adapterErr:       errors.New("synthetic recovery adapter failure"),
			wantAdapter:      1,
			wantFailureAudit: 1,
			wantRollback:     1,
			wantPublicCode:   platform.CodePlatformRecoveryInProgress,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newLoginFixture(t)
			user := loginTestUserWithMode(CredentialModeRecoveryBootstrap)
			user.IsInitialAdmin = true
			user.RecoveryRequired = true
			if test.mutateUser != nil {
				test.mutateUser(&user)
			}
			calls := 0
			fixture.service.loginDependencies.RecoveryMode = loginRecoveryModeStub{
				allowed: test.allowed, err: test.adapterErr, calls: &calls,
			}
			fixture.repository.user = user
			result, err := fixture.service.Login(
				fixture.context,
				fixture.request(LoginRequest{
					Username:          "Alice",
					Password:          "Password123",
					SourceFingerprint: "source-digest",
				}),
			)
			if test.wantSuccess {
				if err != nil ||
					result.User.AuthenticatedCredential != SessionCredentialRecoveryBootstrap {
					t.Fatalf("Login() = %#v, %v", result, err)
				}
			} else if err == nil {
				t.Fatal("Login() error = nil")
			}
			if test.wantPublicCode != "" && platform.PublicCode(err) != test.wantPublicCode {
				t.Fatalf("public code = %q, want %q", platform.PublicCode(err), test.wantPublicCode)
			}
			if calls != test.wantAdapter || len(fixture.hasher.verifiedDigests) != test.wantVerify {
				t.Fatalf(
					"adapter/Verify calls = %d/%d, want %d/%d",
					calls,
					len(fixture.hasher.verifiedDigests),
					test.wantAdapter,
					test.wantVerify,
				)
			}
			if fixture.sessions.calls != test.wantSessions ||
				fixture.audit.calls != test.wantAudit ||
				fixture.failureAudit.calls != test.wantFailureAudit {
				t.Fatalf(
					"session/audit/failure-audit calls = %d/%d/%d, want %d/%d/%d",
					fixture.sessions.calls,
					fixture.audit.calls,
					fixture.failureAudit.calls,
					test.wantSessions,
					test.wantAudit,
					test.wantFailureAudit,
				)
			}
			if fixture.transaction.commitCalls != test.wantCommit ||
				fixture.transaction.rollbackCalls != test.wantRollback {
				t.Fatalf(
					"commit/rollback calls = %d/%d, want %d/%d",
					fixture.transaction.commitCalls,
					fixture.transaction.rollbackCalls,
					test.wantCommit,
					test.wantRollback,
				)
			}
		})
	}
	for _, mode := range []string{"", "unknown"} {
		if _, err := NewConfiguredLoginRecoveryMode(mode); err == nil {
			t.Fatalf("NewConfiguredLoginRecoveryMode(%q) error = nil", mode)
		}
	}
	fixture := newLoginFixture(t)
	adapterCalls := 0
	fixture.service.loginDependencies.RecoveryMode = loginRecoveryModeStub{
		allowed: true, calls: &adapterCalls,
	}
	if _, err := fixture.service.Login(
		fixture.context,
		fixture.request(LoginRequest{
			Username:          "Alice",
			Password:          "Password123",
			SourceFingerprint: "source-digest",
		}),
	); err != nil {
		t.Fatalf("permanent Login() error = %v", err)
	}
	if adapterCalls != 0 {
		t.Fatalf("non recovery-bootstrap adapter calls = %d, want 0", adapterCalls)
	}
	if fixture.sessions.calls != 1 || fixture.audit.calls != 1 ||
		fixture.transaction.commitCalls != 1 || fixture.transaction.rollbackCalls != 0 {
		t.Fatalf(
			"session/audit/commit/rollback calls = %d/%d/%d/%d, want 1/1/1/0",
			fixture.sessions.calls,
			fixture.audit.calls,
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}
}

func TestSameAuditEvent_RequiresAllImmutableFieldsAndCanonicalMetadata(t *testing.T) {
	userID := "00000000-0000-4000-8000-000000000041"
	system := "identity"
	role := "member"
	errorCode := string(platform.CodePlatformDatabaseUnavailable)
	source := "synthetic-source-digest"
	base := audit.Event{
		ID:                "00000000-0000-4000-8000-000000000042",
		OccurredAt:        time.Date(2026, time.October, 22, 12, 0, 0, 123000, time.UTC),
		ActorKind:         audit.ActorUser,
		ActorUserID:       &userID,
		ActorSystem:       &system,
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
	equivalent := cloneAuditEvent(base)
	equivalent.Metadata = map[string]any{"beta": float64(2), "alpha": "one"}
	if !sameAuditEvent(base, equivalent) {
		t.Fatal("sameAuditEvent() rejected canonical metadata with different key order")
	}
	canonicalTimeEquivalent := cloneAuditEvent(base)
	canonicalTimeEquivalent.OccurredAt = base.OccurredAt.UTC().Truncate(time.Microsecond)
	if !sameAuditEvent(base, canonicalTimeEquivalent) {
		t.Fatal("sameAuditEvent() rejected the same normalized microsecond timestamp")
	}
	oneMicrosecondLater := cloneAuditEvent(base)
	oneMicrosecondLater.OccurredAt = base.OccurredAt.Add(time.Microsecond)
	if sameAuditEvent(base, oneMicrosecondLater) {
		t.Fatal("sameAuditEvent() accepted a timestamp differing by one microsecond")
	}
	if !sameAuditEvent(
		audit.Event{Metadata: nil},
		audit.Event{Metadata: map[string]any{}},
	) {
		t.Fatal("sameAuditEvent() rejected canonical nil/empty metadata")
	}

	tests := []struct {
		name   string
		mutate func(*audit.Event)
	}{
		{name: "id", mutate: func(event *audit.Event) { event.ID += "-changed" }},
		{name: "occurred at", mutate: func(event *audit.Event) { event.OccurredAt = event.OccurredAt.Add(time.Microsecond) }},
		{name: "actor kind", mutate: func(event *audit.Event) { event.ActorKind = audit.ActorAnonymous }},
		{name: "actor user", mutate: func(event *audit.Event) { event.ActorUserID = nil }},
		{name: "actor system", mutate: func(event *audit.Event) { event.ActorSystem = nil }},
		{name: "actor role", mutate: func(event *audit.Event) { event.ActorRole = nil }},
		{name: "action", mutate: func(event *audit.Event) { event.Action += ".changed" }},
		{name: "target type", mutate: func(event *audit.Event) { event.TargetType += "_changed" }},
		{name: "target id", mutate: func(event *audit.Event) { event.TargetID += "-changed" }},
		{name: "outcome", mutate: func(event *audit.Event) { event.Outcome = loginAuditOutcomeSucceeded }},
		{name: "error code", mutate: func(event *audit.Event) { event.ErrorCode = nil }},
		{name: "trace id", mutate: func(event *audit.Event) { event.TraceID = "01ARZ3NDEKTSV4RRFFQ69G5FAW" }},
		{name: "source fingerprint", mutate: func(event *audit.Event) { event.SourceFingerprint = nil }},
		{name: "metadata", mutate: func(event *audit.Event) { event.Metadata["alpha"] = "changed" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneAuditEvent(base)
			test.mutate(&changed)
			if sameAuditEvent(base, changed) {
				t.Fatal("sameAuditEvent() accepted changed immutable field")
			}
		})
	}
}

func TestNewLoginHandler_F005FailsClosedWithoutBoundedComposition(t *testing.T) {
	valid := newLoginHandlerTestConfig(t)
	tests := []struct {
		name   string
		config LoginHandlerConfig
	}{
		{name: "missing coordinator", config: LoginHandlerConfig{
			Observer: valid.Observer, ObservationBudget: time.Second,
		}},
		{name: "missing observer", config: LoginHandlerConfig{
			Coordinator: valid.Coordinator, ObservationBudget: time.Second,
		}},
		{name: "zero budget", config: LoginHandlerConfig{
			Coordinator: valid.Coordinator, Observer: valid.Observer,
		}},
		{name: "negative budget", config: LoginHandlerConfig{
			Coordinator: valid.Coordinator, Observer: valid.Observer,
			ObservationBudget: -time.Second,
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewLoginHandler(
				&loginServiceStub{},
				registrationSourceStub{},
				4096,
				test.config,
			); err == nil {
				t.Fatal("NewLoginHandler() error = nil, want fail-closed composition")
			}
		})
	}
	if _, err := NewLoginOutcomeObserver(nil, &loginMetricsStub{}); err == nil {
		t.Fatal("NewLoginOutcomeObserver() accepted a missing logger")
	}
	if _, err := NewLoginOutcomeObserver(&loginEventLoggerStub{}, nil); err == nil {
		t.Fatal("NewLoginOutcomeObserver() accepted missing metrics")
	}
	if _, err := NewSlogLoginEventLogger(nil, "han-server"); err == nil {
		t.Fatal("NewSlogLoginEventLogger() accepted a missing base logger")
	}
	if _, err := NewSlogLoginEventLogger(slog.Default(), " "); err == nil {
		t.Fatal("NewSlogLoginEventLogger() accepted a blank service")
	}
}

func TestLoginHandler_F005ReportsFiveOutcomesIncludingInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		service     *loginServiceStub
		wantStatus  int
		wantOutcome LoginOutcome
	}{
		{
			name: "success", body: `{"username":"Alice","password":"Password123"}`,
			service: &loginServiceStub{result: LoginResult{
				User: AuthenticatedUser{
					ID:   "00000000-0000-4000-8000-000000000001",
					Role: RoleMember, Status: UserStatusActive,
				},
				SessionToken: "new-session-token",
			}},
			wantStatus: http.StatusOK, wantOutcome: LoginOutcomeSuccess,
		},
		{
			name: "authentication failed", body: `{"username":"Alice","password":"Wrong123"}`,
			service:    &loginServiceStub{err: loginFailedError(errLoginRejected)},
			wantStatus: http.StatusUnauthorized, wantOutcome: LoginOutcomeAuthenticationFailed,
		},
		{
			name: "rate limited", body: `{"username":"Alice","password":"Wrong123"}`,
			service: &loginServiceStub{err: blockedLoginError(LoginAbuseDecision{
				Kind: LoginAbuseSourceBlocked, RetryAfter: time.Second,
			})},
			wantStatus: http.StatusTooManyRequests, wantOutcome: LoginOutcomeRateLimited,
		},
		{
			name: "invalid JSON", body: `{"username":`,
			service:    &loginServiceStub{},
			wantStatus: http.StatusBadRequest, wantOutcome: LoginOutcomeInvalidRequest,
		},
		{
			name: "internal failure", body: `{"username":"Alice","password":"Password123"}`,
			service: &loginServiceStub{err: platform.NewPublicError(
				platform.CodePlatformInternalError,
				errors.New("synthetic internal failure"),
			)},
			wantStatus: http.StatusInternalServerError, wantOutcome: LoginOutcomeInternalFailure,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observer := &loginOutcomeObserverStub{}
			config := newLoginHandlerTestConfig(t)
			config.Observer = observer
			handler, err := NewLoginHandler(
				test.service,
				registrationSourceStub{evidence: RegistrationSourceEvidence{
					Fingerprint: "source-digest",
				}},
				4096,
				config,
			)
			if err != nil {
				t.Fatalf("NewLoginHandler() error = %v", err)
			}
			response := httptest.NewRecorder()

			platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
				response,
				newLoginHTTPRequest(test.body),
			)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if observer.calls != 1 || observer.observed[0].Outcome != test.wantOutcome {
				t.Fatalf("observations = %#v", observer.observed)
			}
			if test.wantOutcome == LoginOutcomeInvalidRequest && test.service.request.Attempt != nil {
				t.Fatal("invalid JSON entered the login service")
			}
		})
	}
}

func TestLoginHandler_F005UsesEarlierDeadlineAndBoundsBlockingObserver(t *testing.T) {
	t.Run("request deadline wins", func(t *testing.T) {
		var observedDeadline time.Time
		observer := &loginOutcomeObserverStub{onObserve: func(
			ctx context.Context,
			_ LoginObservation,
		) error {
			observedDeadline, _ = ctx.Deadline()
			return nil
		}}
		config := newLoginHandlerTestConfig(t)
		config.Observer = observer
		config.ObservationBudget = time.Hour
		handler, err := NewLoginHandler(
			&loginServiceStub{err: loginFailedError(errLoginRejected)},
			registrationSourceStub{evidence: RegistrationSourceEvidence{
				Fingerprint: "source-digest",
			}},
			4096,
			config,
		)
		if err != nil {
			t.Fatalf("NewLoginHandler() error = %v", err)
		}
		requestDeadline := time.Now().Add(time.Minute)
		ctx, cancel := context.WithDeadline(
			newLoginHTTPRequest(`{"username":"Alice","password":"Wrong123"}`).Context(),
			requestDeadline,
		)
		defer cancel()
		request := newLoginHTTPRequest(`{"username":"Alice","password":"Wrong123"}`).WithContext(ctx)

		platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
			httptest.NewRecorder(),
			request,
		)

		if !observedDeadline.Equal(requestDeadline) {
			t.Fatalf("observer deadline = %v, want request deadline %v", observedDeadline, requestDeadline)
		}
	})

	t.Run("composition budget cancels blocking observer", func(t *testing.T) {
		var observedErr error
		observer := &loginOutcomeObserverStub{onObserve: func(
			ctx context.Context,
			_ LoginObservation,
		) error {
			<-ctx.Done()
			observedErr = ctx.Err()
			return observedErr
		}}
		config := newLoginHandlerTestConfig(t)
		config.Observer = observer
		config.ObservationBudget = 20 * time.Millisecond
		handler, err := NewLoginHandler(
			&loginServiceStub{err: loginFailedError(errLoginRejected)},
			registrationSourceStub{evidence: RegistrationSourceEvidence{
				Fingerprint: "source-digest",
			}},
			4096,
			config,
		)
		if err != nil {
			t.Fatalf("NewLoginHandler() error = %v", err)
		}
		returned := make(chan struct{})
		go func() {
			platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
				httptest.NewRecorder(),
				newLoginHTTPRequest(`{"username":"Alice","password":"Wrong123"}`),
			)
			close(returned)
		}()

		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("Login() did not return within the bounded observation budget")
		}
		if !errors.Is(observedErr, context.DeadlineExceeded) || observer.calls != 1 {
			t.Fatalf("observer calls/error = %d/%v", observer.calls, observedErr)
		}
	})
}

func TestLoginHandler_F005WriteFailureIsObservedWithoutSecondResponse(t *testing.T) {
	observer := &loginOutcomeObserverStub{}
	config := newLoginHandlerTestConfig(t)
	config.Observer = observer
	handler, err := NewLoginHandler(
		&loginServiceStub{result: LoginResult{
			User:         AuthenticatedUser{ID: "00000000-0000-4000-8000-000000000001"},
			SessionToken: "new-session-token",
		}, onLogin: markLoginAttemptCommitted},
		registrationSourceStub{evidence: RegistrationSourceEvidence{
			Fingerprint: "source-digest",
		}},
		4096,
		config,
	)
	if err != nil {
		t.Fatalf("NewLoginHandler() error = %v", err)
	}
	response := &loginFailingResponseWriter{header: make(http.Header)}

	platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
		response,
		newLoginHTTPRequest(`{"username":"Alice","password":"Password123"}`),
	)

	if response.writeCalls != 1 {
		t.Fatalf("response body write calls = %d, want 1", response.writeCalls)
	}
	if observer.calls != 1 ||
		observer.observed[0].FailureStage != LoginFailureStageResponseWrite ||
		observer.observed[0].ResponseWrite != LoginResponseWriteFailed ||
		observer.observed[0].TransactionOutcome != LoginTransactionCommitted ||
		observer.observed[0].StateChanged != LoginStateSessionCommitted ||
		observer.observed[0].AuditPersisted != LoginAuditPersistedYes {
		t.Fatalf("observations = %#v", observer.observed)
	}
}

func TestLoginHandler_F005CookieOutputFailurePreservesCommittedState(t *testing.T) {
	observer := &loginOutcomeObserverStub{}
	config := newLoginHandlerTestConfig(t)
	config.Observer = observer
	handler, err := NewLoginHandler(
		&loginServiceStub{result: LoginResult{
			User:         AuthenticatedUser{ID: "00000000-0000-4000-8000-000000000001"},
			SessionToken: "invalid\r\ncookie",
		}, onLogin: markLoginAttemptCommitted},
		registrationSourceStub{evidence: RegistrationSourceEvidence{
			Fingerprint: "source-digest",
		}},
		4096,
		config,
	)
	if err != nil {
		t.Fatalf("NewLoginHandler() error = %v", err)
	}
	response := httptest.NewRecorder()

	platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
		response,
		newLoginHTTPRequest(`{"username":"Alice","password":"Password123"}`),
	)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if len(response.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("invalid committed session token wrote a cookie")
	}
	if observer.calls != 1 {
		t.Fatalf("observer calls = %d, want 1", observer.calls)
	}
	observation := observer.observed[0]
	if observation.FailureStage != LoginFailureStageCookieOutput ||
		observation.ResponseWrite != LoginResponseWriteOK ||
		observation.TransactionOutcome != LoginTransactionCommitted ||
		observation.StateChanged != LoginStateSessionCommitted ||
		observation.AuditPersisted != LoginAuditPersistedYes {
		t.Fatalf("observation = %#v", observation)
	}
}

func TestLoginHandler_F005ObserverFailureDoesNotChangePublicResponse(t *testing.T) {
	observerErr := errors.New("synthetic observer failure")
	observer := &loginOutcomeObserverStub{onObserve: func(
		context.Context,
		LoginObservation,
	) error {
		return observerErr
	}}
	config := newLoginHandlerTestConfig(t)
	config.Observer = observer
	handler, err := NewLoginHandler(
		&loginServiceStub{err: loginFailedError(errLoginRejected)},
		registrationSourceStub{evidence: RegistrationSourceEvidence{
			Fingerprint: "source-digest",
		}},
		4096,
		config,
	)
	if err != nil {
		t.Fatalf("NewLoginHandler() error = %v", err)
	}
	response := httptest.NewRecorder()

	platform.TraceMiddleware(http.HandlerFunc(handler.Login)).ServeHTTP(
		response,
		newLoginHTTPRequest(`{"username":"Alice","password":"Wrong123"}`),
	)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", response.Code, response.Body)
	}
	if observer.calls != 1 {
		t.Fatalf("observer calls = %d, want 1", observer.calls)
	}
	if strings.Contains(response.Body.String(), observerErr.Error()) {
		t.Fatal("public response exposed the observer failure")
	}
}

func TestLoginOutcomeObserverAdapter_StopsFanoutAfterSinkFailure(t *testing.T) {
	loggerErr := errors.New("synthetic logger failure")
	logger := &loginEventLoggerStub{err: loggerErr}
	metrics := &loginMetricsStub{}
	observer, err := NewLoginOutcomeObserver(logger, metrics)
	if err != nil {
		t.Fatalf("NewLoginOutcomeObserver() error = %v", err)
	}

	err = observer.ObserveFinal(context.Background(), LoginObservation{
		Outcome: LoginOutcomeInternalFailure,
	})

	if !errors.Is(err, loggerErr) {
		t.Fatalf("ObserveFinal() error = %v, want logger failure", err)
	}
	if logger.calls != 1 || metrics.calls != 0 {
		t.Fatalf("logger/metrics calls = %d/%d, want 1/0", logger.calls, metrics.calls)
	}
}

func TestLoginOutcomeObserverAdapter_MetricsObeysCancellation(t *testing.T) {
	logger := &loginEventLoggerStub{}
	metrics := &loginMetricsStub{blockUntilCanceled: true}
	observer, err := NewLoginOutcomeObserver(logger, metrics)
	if err != nil {
		t.Fatalf("NewLoginOutcomeObserver() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = observer.ObserveFinal(ctx, LoginObservation{
		Outcome: LoginOutcomeAuthenticationFailed,
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ObserveFinal() error = %v, want deadline exceeded", err)
	}
	if logger.calls != 0 || metrics.calls != 1 {
		t.Fatalf("logger/metrics calls = %d/%d, want 0/1", logger.calls, metrics.calls)
	}
}

func TestLoginOutcomeObserverAdapter_RestrictsMetricsToFixedAllowlist(t *testing.T) {
	logger := &loginEventLoggerStub{}
	metrics := &loginMetricsStub{}
	observer, err := NewLoginOutcomeObserver(logger, metrics)
	if err != nil {
		t.Fatalf("NewLoginOutcomeObserver() error = %v", err)
	}
	observation := LoginObservation{
		Endpoint:           "POST /api/v1/sessions",
		Duration:           250 * time.Millisecond,
		Outcome:            LoginOutcomeAuthenticationFailed,
		ReasonBucket:       "none",
		ErrorCode:          string(platform.CodeAuthLoginFailed),
		FailureStage:       LoginFailureStageNone,
		TransactionOutcome: LoginTransactionCommitted,
		StateChanged:       LoginStateNone,
		AuditPersisted:     LoginAuditPersistedYes,
		RollbackOutcome:    LoginRollbackNotAttempted,
		ResponseWrite:      LoginResponseWriteOK,
		TraceID:            "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}

	if err := observer.ObserveFinal(context.Background(), observation); err != nil {
		t.Fatalf("ObserveFinal() error = %v", err)
	}

	if logger.calls != 0 || metrics.calls != 1 {
		t.Fatalf("logger/metrics calls = %d/%d, want 0/1", logger.calls, metrics.calls)
	}
	want := LoginMetricObservation{
		Endpoint:     observation.Endpoint,
		Duration:     observation.Duration,
		Outcome:      observation.Outcome,
		ReasonBucket: observation.ReasonBucket,
		ErrorCode:    observation.ErrorCode,
		FailureStage: observation.FailureStage,
	}
	if metrics.samples[0] != want {
		t.Fatalf("metric sample = %#v, want %#v", metrics.samples[0], want)
	}
	if loginAttemptsMetricName != "han_identity_login_attempts_total" ||
		loginFailuresMetricName != "han_identity_login_internal_failures_total" ||
		loginDurationMetricName != "han_identity_login_duration_seconds" {
		t.Fatal("login metric names changed")
	}
}

func TestSlogLoginEventLogger_EmitsOnlyStableInternalFailureFields(t *testing.T) {
	capture := &loginSlogCapture{}
	logger, err := NewSlogLoginEventLogger(
		slog.New(&loginSlogHandler{capture: capture}),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogLoginEventLogger() error = %v", err)
	}
	observation := LoginObservation{
		Endpoint:           "POST /api/v1/sessions",
		Outcome:            LoginOutcomeInternalFailure,
		ReasonBucket:       "persistence",
		ErrorCode:          string(platform.CodePlatformDatabaseUnavailable),
		FailureStage:       LoginFailureStageTransactionCommit,
		TransactionOutcome: LoginTransactionUnknown,
		StateChanged:       LoginStateUnknown,
		AuditPersisted:     LoginAuditPersistedUnknown,
		RollbackOutcome:    LoginRollbackNotAttempted,
		TraceID:            "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}

	if err := logger.LogLoginOutcome(context.Background(), observation); err != nil {
		t.Fatalf("LogLoginOutcome() error = %v", err)
	}

	if len(capture.records) != 1 {
		t.Fatalf("log records = %d, want 1", len(capture.records))
	}
	record := capture.records[0]
	if record.level != slog.LevelError {
		t.Fatalf("log level = %v, want ERROR", record.level)
	}
	want := map[string]string{
		"service":             "han-server",
		"module":              "identity",
		"trace_id":            observation.TraceID,
		"event":               "identity.login.internal_failure",
		"endpoint":            observation.Endpoint,
		"outcome":             string(observation.Outcome),
		"reason_bucket":       observation.ReasonBucket,
		"error_code":          observation.ErrorCode,
		"failure_stage":       string(observation.FailureStage),
		"transaction_outcome": string(observation.TransactionOutcome),
		"state_changed":       string(observation.StateChanged),
		"audit_persisted":     string(observation.AuditPersisted),
		"rollback_outcome":    string(observation.RollbackOutcome),
	}
	if len(record.attributes) != len(want) {
		t.Fatalf("log attribute keys = %v", record.attributes)
	}
	for key, value := range want {
		if record.attributes[key] != value {
			t.Fatalf("log attribute %q = %q, want %q", key, record.attributes[key], value)
		}
	}
	encoded, err := json.Marshal(record.attributes)
	if err != nil {
		t.Fatalf("marshal captured log fields: %v", err)
	}
	for _, forbidden := range []string{
		"Password123",
		testFakeCredentialProbe,
		"new-session-token",
		"new-csrf-token",
		"Alice",
		"source-digest",
		"synthetic internal failure",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("log fields contained forbidden login material")
		}
	}
}

func TestSlogLoginEventLogger_ObeysCancellationWithoutRetry(t *testing.T) {
	capture := &loginSlogCapture{blockUntilCanceled: true}
	logger, err := NewSlogLoginEventLogger(
		slog.New(&loginSlogHandler{capture: capture}),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogLoginEventLogger() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = logger.LogLoginOutcome(ctx, LoginObservation{
		Endpoint:           "POST /api/v1/sessions",
		Outcome:            LoginOutcomeInternalFailure,
		ReasonBucket:       "audit",
		ErrorCode:          string(platform.CodePlatformInternalError),
		FailureStage:       LoginFailureStageAuditAppend,
		TransactionOutcome: LoginTransactionUnknown,
		StateChanged:       LoginStateUnknown,
		AuditPersisted:     LoginAuditPersistedUnknown,
		RollbackOutcome:    LoginRollbackUnknown,
		TraceID:            "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("LogLoginOutcome() error = %v, want deadline exceeded", err)
	}
	if capture.handleCalls != 1 {
		t.Fatalf("slog handler calls = %d, want 1", capture.handleCalls)
	}
}

func assertLoginAuditSubjectEvidence(
	t *testing.T,
	event audit.Event,
	wantDigest string,
	wantVersion string,
) {
	t.Helper()
	if event.Metadata["login_subject_digest"] != wantDigest ||
		event.Metadata["login_subject_digest_key_version"] != wantVersion {
		t.Fatalf("login audit metadata = %#v", event.Metadata)
	}
	encoded, err := json.Marshal(event.Metadata)
	if err != nil {
		t.Fatalf("marshal login audit metadata: %v", err)
	}
	if strings.Contains(strings.ToLower(string(encoded)), "alice") {
		t.Fatalf("login audit metadata leaked raw username: %s", encoded)
	}
}

func assertFunctionBodyLineLimit(t *testing.T, source string, targets map[string]bool) {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate test file")
	}
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filepath.Join(filepath.Dir(testFile), source), nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", source, err)
	}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil || !targets[function.Name.Name] {
			continue
		}
		lines := fileSet.Position(function.Body.End()).Line -
			fileSet.Position(function.Body.Pos()).Line + 1
		t.Logf("%s body lines = %d", function.Name.Name, lines)
		if lines > 50 {
			t.Errorf("%s body lines = %d, want <= 50", function.Name.Name, lines)
		}
		delete(targets, function.Name.Name)
	}
	for name := range targets {
		t.Errorf("%s declaration not found in %s", name, source)
	}
}

func newLoginTestArgon2idHasher(
	t *testing.T,
	config security.Argon2idConfig,
	fill byte,
) *security.Argon2idHasher {
	t.Helper()
	hasher, err := security.NewArgon2idHasher(
		config,
		bytes.NewReader(bytes.Repeat([]byte{fill}, int(config.SaltLength))),
	)
	if err != nil {
		t.Fatalf("NewArgon2idHasher() error = %v", err)
	}
	return hasher
}

func loginConstructorDependencies(
	hasher LoginPasswordHasher,
	fakeDigest string,
	fakeProbe string,
) LoginDependencies {
	now := time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC)
	return LoginDependencies{
		Transactions:         &loginTransactorStub{transaction: newLoginTxStub()},
		Repository:           &loginRepositoryStub{user: loginTestUser()},
		PasswordResetGate:    &loginPasswordResetGateStub{},
		Passwords:            hasher,
		Sessions:             &loginSessionStoreStub{},
		SessionIssuer:        loginSessionIssuerStub{now: now},
		Abuse:                &loginAbuseStub{},
		RecoveryMode:         loginRecoveryModeStub{},
		Audit:                &loginAuditStub{},
		IDs:                  &loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000099"}},
		Clock:                loginClockStub{now: now},
		FakeCredentialDigest: fakeDigest,
		FakeCredentialProbe:  fakeProbe,
	}
}

func newLoginHTTPRequest(body string) *http.Request {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/sessions",
		bytes.NewBufferString(body),
	)
	request.Header.Set("Content-Type", "application/json")
	return request
}

type loginFixture struct {
	context           context.Context
	now               time.Time
	transaction       *loginTxStub
	repository        *loginRepositoryStub
	hasher            *loginHasherStub
	sessions          *loginSessionStoreStub
	abuse             *loginAbuseStub
	audit             *loginAuditStub
	failureAudit      *loginFailureAuditorStub
	passwordResetGate *loginPasswordResetGateStub
	service           *Service
}

func newLoginFixture(t *testing.T) *loginFixture {
	t.Helper()
	now := time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC)
	transaction := newLoginTxStub()
	repository := &loginRepositoryStub{user: loginTestUser()}
	hasher := &loginHasherStub{matches: map[string]string{
		testCredentialDigest:          "Password123",
		testTemporaryCredentialDigest: "Temporary123",
		"fake-digest":                 testFakeCredentialProbe,
	}}
	sessions := &loginSessionStoreStub{
		credentials: session.Credentials{
			Token: "new-session-token", CSRFToken: "new-csrf-token",
		},
	}
	abuseProtector := &loginAbuseStub{decision: LoginAbuseDecision{
		SubjectDigest:    []byte("synthetic-subject-digest"),
		DigestKeyVersion: "synthetic-key-v1",
	}}
	auditAppender := &loginAuditStub{}
	failureAuditor := &loginFailureAuditorStub{}
	passwordResetGate := &loginPasswordResetGateStub{}
	service, err := NewLoginService(LoginDependencies{
		Transactions:         &loginTransactorStub{transaction: transaction},
		Repository:           repository,
		PasswordResetGate:    passwordResetGate,
		Passwords:            hasher,
		Sessions:             sessions,
		SessionIssuer:        loginSessionIssuerStub{now: now},
		Abuse:                abuseProtector,
		RecoveryMode:         loginRecoveryModeStub{},
		Audit:                auditAppender,
		IDs:                  &loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000099"}},
		Clock:                loginClockStub{now: now},
		FakeCredentialDigest: "fake-digest",
		FakeCredentialProbe:  testFakeCredentialProbe,
	})
	if err != nil {
		t.Fatalf("NewLoginService() error = %v", err)
	}
	hasher.verifiedDigests = nil
	ctx, err := platform.ContextWithTraceID(
		context.Background(),
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	return &loginFixture{
		context: ctx, now: now, transaction: transaction, repository: repository,
		hasher: hasher, sessions: sessions, abuse: abuseProtector,
		audit: auditAppender, failureAudit: failureAuditor,
		passwordResetGate: passwordResetGate, service: service,
	}
}

func (f *loginFixture) request(request LoginRequest) LoginRequest {
	coordinator, err := NewLoginAttemptCoordinator(
		&loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000099"}},
		loginClockStub{now: f.now},
		f.failureAudit,
	)
	if err != nil {
		panic(err)
	}
	attempt, err := coordinator.Begin()
	if err != nil {
		panic(err)
	}
	if err := attempt.enterService(); err != nil {
		panic(err)
	}
	request.Attempt = attempt
	return request
}

func loginTestUser() User {
	digest := testCredentialDigest
	return User{
		ID: "00000000-0000-4000-8000-000000000001", Username: "Alice",
		PermanentPasswordDigest: &digest, Role: RoleMember, Status: UserStatusActive,
		CredentialMode: CredentialModePermanent,
	}
}

func loginTestUserWithStatus(status UserStatus) User {
	user := loginTestUser()
	user.Status = status
	if status == UserStatusDeleted {
		user.PermanentPasswordDigest = nil
	}
	return user
}

func loginTestRecoveryRequiredUser() User {
	user := loginTestUser()
	user.RecoveryRequired = true
	return user
}

func loginTestUserWithMode(mode CredentialMode) User {
	user := loginTestUser()
	user.CredentialMode = mode
	return user
}

func loginTestTemporaryCredential(
	state TemporaryCredentialState,
	expiredAt time.Time,
) *TemporaryCredential {
	issuedAt := time.Date(2026, time.October, 21, 9, 0, 0, 0, time.UTC)
	credential := &TemporaryCredential{
		ID:      "00000000-0000-4000-8000-000000000010",
		UserID:  "00000000-0000-4000-8000-000000000001",
		Purpose: TemporaryCredentialPasswordReset, Generation: 2,
		State: state, SecretDigest: testTemporaryCredentialDigest,
		DeliveredToEmail: "synthetic@example.invalid",
		IssuedAt:         issuedAt, ExpiresAt: issuedAt.Add(temporaryCredentialLifetime),
	}
	switch state {
	case TemporaryCredentialExpired:
		if expiredAt.IsZero() {
			expiredAt = credential.ExpiresAt
		}
		credential.ExpiresAt = expiredAt
		credential.IssuedAt = expiredAt.Add(-temporaryCredentialLifetime)
	case TemporaryCredentialSuperseded:
		at := issuedAt.Add(time.Hour)
		credential.SupersededAt = &at
	case TemporaryCredentialConsumed:
		at := issuedAt.Add(time.Hour)
		credential.ConsumedAt = &at
	case TemporaryCredentialRecoveryInvalidated:
		at := issuedAt.Add(time.Hour)
		credential.InvalidatedAt = &at
	}
	return credential
}

type loginTxStub struct {
	commitCalls      int
	rollbackCalls    int
	closed           bool
	oldSessionActive bool
	commitHooks      []func()
	commitErr        error
	rollbackErr      error
}

func newLoginTxStub() *loginTxStub {
	return &loginTxStub{oldSessionActive: true}
}

func (tx *loginTxStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}

func (tx *loginTxStub) Commit(context.Context) error {
	tx.commitCalls++
	if tx.closed {
		return pgx.ErrTxClosed
	}
	if tx.commitErr != nil {
		return tx.commitErr
	}
	for _, hook := range tx.commitHooks {
		hook()
	}
	tx.closed = true
	return nil
}

func (tx *loginTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	if tx.closed {
		return pgx.ErrTxClosed
	}
	if tx.rollbackErr != nil {
		return tx.rollbackErr
	}
	tx.closed = true
	return nil
}

func (tx *loginTxStub) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}

func (tx *loginTxStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *loginTxStub) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *loginTxStub) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *loginTxStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not implemented")
}
func (tx *loginTxStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (tx *loginTxStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return loginRowStub{err: errors.New("not implemented")}
}
func (tx *loginTxStub) Conn() *pgx.Conn { return nil }

type loginTransactorStub struct {
	transaction pgx.Tx
	err         error
	beginCalls  int
}

func (s *loginTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	s.beginCalls++
	return s.transaction, s.err
}

type loginRepositoryStub struct {
	user              User
	userErr           error
	findUserCalls     int
	temporary         *TemporaryCredential
	temporaryErr      error
	permanentRehashes int
	temporaryRehashes int
}

func (s *loginRepositoryStub) FindUserByUsername(
	context.Context,
	pgx.Tx,
	string,
) (User, error) {
	s.findUserCalls++
	return s.user, s.userErr
}

func (s *loginRepositoryStub) FindLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (TemporaryCredential, error) {
	if s.temporaryErr != nil {
		return TemporaryCredential{}, s.temporaryErr
	}
	if s.temporary == nil {
		return TemporaryCredential{}, pgx.ErrNoRows
	}
	return *s.temporary, nil
}

func (s *loginRepositoryStub) UpdatePermanentPasswordDigest(
	context.Context,
	pgx.Tx,
	string,
	string,
) error {
	s.permanentRehashes++
	return nil
}

func (s *loginRepositoryStub) UpdateTemporaryCredentialDigest(
	context.Context,
	pgx.Tx,
	string,
	string,
) error {
	s.temporaryRehashes++
	return nil
}

type loginPasswordResetGateStub struct {
	err         error
	calls       int
	userIDs     []string
	beforeCheck func()
}

func (s *loginPasswordResetGateStub) CheckLockedUser(
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

type loginHasherStub struct {
	matches         map[string]string
	verifiedDigests []string
	needsRehash     bool
	err             error
	hashErr         error
}

func (s *loginHasherStub) Verify(password, digest string) (bool, bool, error) {
	s.verifiedDigests = append(s.verifiedDigests, digest)
	if s.err != nil {
		return false, false, s.err
	}
	return s.matches[digest] == password, s.needsRehash, nil
}

func (s *loginHasherStub) Hash(string) (string, error) {
	if s.hashErr != nil {
		return "", s.hashErr
	}
	return testCredentialDigest, s.err
}

type loginSessionStoreStub struct {
	calls       int
	createCalls int
	rotateCalls int
	oldToken    string
	credentials session.Credentials
	err         error
}

func (s *loginSessionStoreStub) Create(
	_ context.Context,
	_ session.Executor,
	_ session.CreateInput,
) (session.Credentials, error) {
	s.calls++
	s.createCalls++
	return s.credentials, s.err
}

func (s *loginSessionStoreStub) Rotate(
	_ context.Context,
	tx pgx.Tx,
	oldToken string,
	_ time.Time,
	_ string,
	_ session.CreateInput,
) (session.Credentials, error) {
	s.calls++
	s.rotateCalls++
	s.oldToken = oldToken
	if s.err != nil {
		return session.Credentials{}, s.err
	}
	transaction := tx.(*loginTxStub)
	transaction.commitHooks = append(transaction.commitHooks, func() {
		transaction.oldSessionActive = false
	})
	return s.credentials, nil
}

type loginSessionIssuerStub struct {
	now   time.Time
	token string
	err   error
}

func (s loginSessionIssuerStub) Issue(
	user User,
	kind session.CredentialKind,
	credentialID *string,
) (session.CreateInput, error) {
	token := s.token
	if token == "" {
		token = "new-session-token"
	}
	return session.CreateInput{
		ID: "00000000-0000-4000-8000-000000000020", UserID: user.ID,
		Token: token, CSRFToken: "new-csrf-token",
		AuthenticatedCredentialKind: kind, AuthenticatedCredentialID: credentialID,
		IssuedAt: s.now, AbsoluteExpiresAt: s.now.Add(8 * time.Hour),
		IdleExpiresAt: s.now.Add(time.Hour),
	}, s.err
}

type loginAbuseStub struct {
	decision     LoginAbuseDecision
	checkErr     error
	failureErr   error
	successErr   error
	failureCalls int
	successCalls int
}

func (s *loginAbuseStub) Check(
	_ context.Context,
	_ pgx.Tx,
	attempt LoginAttempt,
) (LoginAbuseDecision, error) {
	if attempt.Subject == "" {
		return LoginAbuseDecision{}, errors.New("login abuse subject is required")
	}
	return s.decision, s.checkErr
}

func (s *loginAbuseStub) RecordFailure(context.Context, pgx.Tx, LoginAttempt) error {
	s.failureCalls++
	return s.failureErr
}

func (s *loginAbuseStub) RecordSuccess(context.Context, pgx.Tx, LoginAttempt) error {
	s.successCalls++
	return s.successErr
}

type loginRecoveryModeStub struct {
	allowed bool
	err     error
	calls   *int
}

func (s loginRecoveryModeStub) RecoveryBootstrapAllowed(context.Context) (bool, error) {
	if s.calls != nil {
		*s.calls++
	}
	return s.allowed, s.err
}

type loginAuditStub struct {
	events []audit.Event
	err    error
	calls  int
}

func (s *loginAuditStub) Append(_ context.Context, _ pgx.Tx, event audit.Event) error {
	s.calls++
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, event)
	return nil
}

type loginIDStub struct {
	ids []string
}

func (s *loginIDStub) NewID() (string, error) {
	if len(s.ids) == 0 {
		return "", errors.New("no login IDs remain")
	}
	id := s.ids[0]
	s.ids = s.ids[1:]
	return id, nil
}

type loginClockStub struct {
	now time.Time
}

func (s loginClockStub) Now() time.Time { return s.now }

type orderedLoginIDStub struct {
	id    string
	onNew func()
}

func (s orderedLoginIDStub) NewID() (string, error) {
	if s.onNew != nil {
		s.onNew()
	}
	return s.id, nil
}

type orderedLoginClockStub struct {
	now   time.Time
	onNow func()
}

func (s orderedLoginClockStub) Now() time.Time {
	if s.onNow != nil {
		s.onNow()
	}
	return s.now
}

type loginServiceStub struct {
	request LoginRequest
	result  LoginResult
	err     error
	onLogin func(LoginRequest)
}

type loginFailureAuditorStub struct {
	calls    int
	result   LoginAuditPersistence
	err      error
	requests []LoginFailureAuditRequest
	onRecord func(LoginFailureAuditRequest)
}

func (s *loginFailureAuditorStub) RecordOnce(
	_ context.Context,
	request LoginFailureAuditRequest,
) (LoginAuditPersistence, error) {
	s.calls++
	s.requests = append(s.requests, request)
	if s.onRecord != nil {
		s.onRecord(request)
	}
	result := s.result
	if result == "" {
		result = LoginAuditPersistedNo
	}
	return result, s.err
}

type loginOutcomeObserverStub struct {
	calls     int
	observed  []LoginObservation
	onObserve func(context.Context, LoginObservation) error
}

func (s *loginOutcomeObserverStub) ObserveFinal(
	ctx context.Context,
	observation LoginObservation,
) error {
	s.calls++
	s.observed = append(s.observed, observation)
	if s.onObserve != nil {
		return s.onObserve(ctx, observation)
	}
	return nil
}

type loginEventLoggerStub struct {
	calls              int
	err                error
	blockUntilCanceled bool
}

func (s *loginEventLoggerStub) LogLoginOutcome(
	ctx context.Context,
	_ LoginObservation,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.calls++
	if s.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.err
}

type loginMetricsStub struct {
	calls              int
	err                error
	blockUntilCanceled bool
	samples            []LoginMetricObservation
}

func (s *loginMetricsStub) RecordLogin(
	ctx context.Context,
	sample LoginMetricObservation,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.calls++
	s.samples = append(s.samples, sample)
	if s.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.err
}

type loginBlockingReader struct {
	entered chan<- struct{}
	release <-chan struct{}
}

func (r *loginBlockingReader) Read(target []byte) (int, error) {
	close(r.entered)
	<-r.release
	for index := range target {
		target[index] = 0x7f
	}
	return len(target), nil
}

type loginRehashBarrierHasher struct {
	delegate LoginPasswordHasher
	digest   string
	onRehash func()
}

func (h *loginRehashBarrierHasher) Verify(password, digest string) (bool, bool, error) {
	matched, needsRehash, err := h.delegate.Verify(password, digest)
	if err == nil && matched && needsRehash && digest == h.digest && h.onRehash != nil {
		h.onRehash()
	}
	return matched, needsRehash, err
}

func (h *loginRehashBarrierHasher) Hash(password string) (string, error) {
	return h.delegate.Hash(password)
}

type loginResponseWriteCounter struct {
	*httptest.ResponseRecorder
	headerWrites int
	bodyWrites   int
}

func (w *loginResponseWriteCounter) WriteHeader(statusCode int) {
	w.headerWrites++
	w.ResponseRecorder.WriteHeader(statusCode)
}

func (w *loginResponseWriteCounter) Write(body []byte) (int, error) {
	w.bodyWrites++
	return w.ResponseRecorder.Write(body)
}

type orderedLoginSourceStub struct {
	evidence   RegistrationSourceEvidence
	err        error
	onEvidence func()
}

func (s orderedLoginSourceStub) Evidence(*http.Request) (RegistrationSourceEvidence, error) {
	if s.onEvidence != nil {
		s.onEvidence()
	}
	return s.evidence, s.err
}

type orderedResponseWriter struct {
	*httptest.ResponseRecorder
	onWrite func()
}

func (w *orderedResponseWriter) WriteHeader(statusCode int) {
	if w.onWrite != nil {
		w.onWrite()
		w.onWrite = nil
	}
	w.ResponseRecorder.WriteHeader(statusCode)
}

func (w *orderedResponseWriter) Write(body []byte) (int, error) {
	if w.onWrite != nil {
		w.onWrite()
		w.onWrite = nil
	}
	return w.ResponseRecorder.Write(body)
}

type loginFailingResponseWriter struct {
	header     http.Header
	writeCalls int
}

func (w *loginFailingResponseWriter) Header() http.Header {
	return w.header
}

func (*loginFailingResponseWriter) WriteHeader(int) {}

func (w *loginFailingResponseWriter) Write([]byte) (int, error) {
	w.writeCalls++
	return 0, errors.New("synthetic response write failure")
}

type capturedLoginLog struct {
	level      slog.Level
	attributes map[string]string
}

type loginSlogCapture struct {
	handleCalls        int
	blockUntilCanceled bool
	records            []capturedLoginLog
}

type loginSlogHandler struct {
	capture    *loginSlogCapture
	attributes []slog.Attr
}

func (*loginSlogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *loginSlogHandler) Handle(ctx context.Context, record slog.Record) error {
	h.capture.handleCalls++
	if h.capture.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	attributes := make(map[string]string, len(h.attributes)+record.NumAttrs())
	for _, attribute := range h.attributes {
		attributes[attribute.Key] = attribute.Value.String()
	}
	record.Attrs(func(attribute slog.Attr) bool {
		attributes[attribute.Key] = attribute.Value.String()
		return true
	})
	h.capture.records = append(h.capture.records, capturedLoginLog{
		level:      record.Level,
		attributes: attributes,
	})
	return nil
}

func (h *loginSlogHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	combined := append([]slog.Attr(nil), h.attributes...)
	combined = append(combined, attributes...)
	return &loginSlogHandler{capture: h.capture, attributes: combined}
}

func (h *loginSlogHandler) WithGroup(string) slog.Handler {
	return h
}

func newLoginHandlerTestConfig(t *testing.T) LoginHandlerConfig {
	t.Helper()
	coordinator, err := NewLoginAttemptCoordinator(
		&loginIDStub{ids: []string{"00000000-0000-4000-8000-000000000098"}},
		loginClockStub{now: time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC)},
		&loginFailureAuditorStub{},
	)
	if err != nil {
		t.Fatalf("NewLoginAttemptCoordinator() error = %v", err)
	}
	return LoginHandlerConfig{
		Coordinator:       coordinator,
		Observer:          &loginOutcomeObserverStub{},
		ObservationBudget: time.Second,
	}
}

func (s *loginServiceStub) Login(
	_ context.Context,
	request LoginRequest,
) (LoginResult, error) {
	s.request = request
	if s.onLogin != nil {
		s.onLogin(request)
	}
	return s.result, s.err
}

func markLoginAttemptCommitted(request LoginRequest) {
	request.Attempt.updateObservation(func(observation *LoginObservation) {
		observation.TransactionOutcome = LoginTransactionCommitted
		observation.StateChanged = LoginStateSessionCommitted
		observation.AuditPersisted = LoginAuditPersistedYes
	})
}

type memoryLoginAbuseStore struct {
	buckets map[string]LoginAttemptBucket
}

func newMemoryLoginAbuseStore() *memoryLoginAbuseStore {
	return &memoryLoginAbuseStore{buckets: make(map[string]LoginAttemptBucket)}
}

func (s *memoryLoginAbuseStore) LoadForUpdate(
	_ context.Context,
	_ pgx.Tx,
	kind LoginBucketKind,
	digest []byte,
) (LoginAttemptBucket, error) {
	bucket, ok := s.buckets[loginBucketMapKey(kind, digest)]
	if !ok {
		return LoginAttemptBucket{}, pgx.ErrNoRows
	}
	return bucket, nil
}

func (s *memoryLoginAbuseStore) Save(
	_ context.Context,
	_ pgx.Tx,
	bucket LoginAttemptBucket,
) error {
	s.buckets[loginBucketMapKey(bucket.Kind, bucket.KeyDigest)] = bucket
	return nil
}

func (s *memoryLoginAbuseStore) Delete(
	_ context.Context,
	_ pgx.Tx,
	kind LoginBucketKind,
	digest []byte,
) error {
	delete(s.buckets, loginBucketMapKey(kind, digest))
	return nil
}

func loginBucketMapKey(kind LoginBucketKind, digest []byte) string {
	return string(kind) + ":" + string(digest)
}

type loginRowStub struct {
	err error
}

func (r loginRowStub) Scan(...any) error { return r.err }

type loginBucketTxStub struct {
	*loginTxStub
	operations []string
	execSQL    string
}

func (tx *loginBucketTxStub) Exec(
	_ context.Context,
	sql string,
	_ ...any,
) (pgconn.CommandTag, error) {
	tx.operations = append(tx.operations, "exec")
	tx.execSQL = sql
	return pgconn.CommandTag{}, nil
}

func (tx *loginBucketTxStub) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	tx.operations = append(tx.operations, "query")
	return loginRowStub{err: pgx.ErrNoRows}
}

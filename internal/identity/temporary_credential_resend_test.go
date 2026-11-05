package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestTemporaryCredentialResendHandlerRoutesPurpose(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantPurpose TemporaryCredentialResendPurpose
		wantRole    Role
	}{
		{
			name: "password reset",
			path: "/api/v1/admin/users/" + passwordResetTargetID +
				"/password-reset/resend",
			wantPurpose: TemporaryCredentialResendPasswordReset,
			wantRole:    RoleMember,
		},
		{
			name: "new administrator",
			path: "/api/v1/admin/administrators/" + passwordResetTargetID +
				"/temporary-password/resend",
			wantPurpose: TemporaryCredentialResendNewAdministrator,
			wantRole:    RoleAdmin,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &temporaryCredentialResendServiceStub{
				result: PasswordResetResult{
					UserID:                       passwordResetTargetID,
					TemporaryCredentialExpiresAt: passwordResetNow.Add(temporaryCredentialLifetime),
					DeliveryStatus:               passwordResetDeliveryAccepted,
				},
			}
			handler, err := NewTemporaryCredentialResendHandler(service)
			if err != nil {
				t.Fatalf("NewTemporaryCredentialResendHandler() error = %v", err)
			}
			request := passwordResetHTTPRequest(t, http.MethodPost, test.path)
			request.Header.Set("X-Temporary-Credential-Purpose", "attacker-controlled")
			request.URL.RawQuery = "purpose=attacker-controlled"
			response := httptest.NewRecorder()

			temporaryCredentialResendProtectedHandler(t, handler).
				ServeHTTP(response, request)

			if response.Code != http.StatusOK || service.calls != 1 {
				t.Fatalf("status/calls = %d/%d; body=%s",
					response.Code, service.calls, response.Body.String())
			}
			if service.request.Purpose != test.wantPurpose ||
				service.request.TargetRole != test.wantRole ||
				service.request.TargetID != passwordResetTargetID {
				t.Fatalf("resend request = %#v", service.request)
			}
			if strings.Contains(response.Body.String(), passwordResetSecret) ||
				strings.Contains(response.Body.String(), "temporary_password") {
				t.Fatalf("response leaked secret: %s", response.Body.String())
			}
		})
	}
}

func TestTemporaryCredentialResendHandlerRejectsMethodPathAndMissingKey(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		removeKey  bool
		wantStatus int
		wantCode   platform.Code
	}{
		{
			name: "method", method: http.MethodGet,
			path: "/api/v1/admin/users/" + passwordResetTargetID +
				"/password-reset/resend",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   platform.CodeRequestMethodNotAllowed,
		},
		{
			name: "path", method: http.MethodPost,
			path: "/api/v1/admin/users/" + passwordResetTargetID +
				"/password-reset/resend/extra",
			wantStatus: http.StatusNotFound,
			wantCode:   platform.CodeIdentityUserNotFound,
		},
		{
			name: "key", method: http.MethodPost,
			path: "/api/v1/admin/users/" + passwordResetTargetID +
				"/password-reset/resend",
			removeKey: true, wantStatus: http.StatusBadRequest,
			wantCode: platform.CodeIdentityIdempotencyKeyRequired,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &temporaryCredentialResendServiceStub{}
			handler, err := NewTemporaryCredentialResendHandler(service)
			if err != nil {
				t.Fatal(err)
			}
			request := passwordResetHTTPRequest(t, test.method, test.path)
			if test.removeKey {
				request.Header.Del("Idempotency-Key")
			}
			response := httptest.NewRecorder()

			temporaryCredentialResendProtectedHandler(t, handler).
				ServeHTTP(response, request)

			if response.Code != test.wantStatus || service.calls != 0 {
				t.Fatalf("status/calls = %d/%d; body=%s",
					response.Code, service.calls, response.Body.String())
			}
			if test.wantStatus == http.StatusMethodNotAllowed &&
				response.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow = %q, want POST", response.Header().Get("Allow"))
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Code != test.wantCode {
				t.Fatalf("code = %q, want %q", envelope.Code, test.wantCode)
			}
		})
	}
}

func TestTemporaryCredentialResendPasswordResetCreatesSuccessor(t *testing.T) {
	fixture := newPasswordResetFixture(
		passwordResetActor(false),
		passwordResetTarget(RoleMember),
	)
	predecessor := "password-reset-predecessor"
	fixture.repository.latest = TemporaryCredential{
		ID: "password-reset-old", UserID: passwordResetTargetID,
		Purpose: TemporaryCredentialPasswordReset, Generation: 4,
		AuthGeneration: 4, SourceOperationID: &predecessor,
		State:            TemporaryCredentialActive,
		SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
		DeliveredToEmail: "target@example.invalid",
		IssuedAt:         passwordResetNow.Add(-time.Hour),
		ExpiresAt:        passwordResetNow.Add(-time.Hour).Add(temporaryCredentialLifetime),
	}
	fixture.repository.latestErr = nil
	authority := &temporaryCredentialResendAuthorityStub{}
	fixture.service.authority = authority

	result, err := fixture.service.ResendPasswordResetCredential(
		context.Background(),
		PasswordResetRequest{
			Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
			TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
			TraceID: passwordResetTraceID,
		},
	)
	if err != nil {
		t.Fatalf("ResendPasswordResetCredential() error = %v", err)
	}
	if result.DeliveryStatus != passwordResetDeliveryAccepted ||
		len(fixture.mailer.messages) != 1 {
		t.Fatalf("result=%#v mail=%d", result, len(fixture.mailer.messages))
	}
	candidate := authority.candidate
	if candidate.Operation.RequestKind != PasswordResetResend ||
		candidate.Operation.PredecessorOperationID == nil ||
		*candidate.Operation.PredecessorOperationID != predecessor ||
		candidate.Operation.ReplacedCredentialID == nil ||
		*candidate.Operation.ReplacedCredentialID != fixture.repository.latest.ID ||
		candidate.Credential.Generation != 5 {
		t.Fatalf("successor candidate = %#v", candidate)
	}
	if !bytesContain(candidate.IdempotencyRequest.Scope, temporaryCredentialResendOperation) {
		t.Fatalf("resend scope = %q", candidate.IdempotencyRequest.Scope)
	}
}

func TestTemporaryCredentialResendPasswordResetReplayHasNoSensitivePreparation(t *testing.T) {
	tests := []struct {
		name       string
		state      PasswordResetOperationState
		claim      idempotency.Record
		wantCode   platform.Code
		wantStatus string
	}{
		{
			name:  "succeeded",
			state: PasswordResetOperationSucceeded,
			claim: passwordResetStableRecord(
				idempotency.StatusSucceeded,
				idempotency.ResponseSnapshot{
					Outcome: passwordResetOutcomeSucceeded, ResourceID: passwordResetTargetID,
				},
			),
			wantStatus: passwordResetDeliveryAccepted,
		},
		{
			name:     "failed",
			state:    PasswordResetOperationFailed,
			claim:    passwordResetRecord(idempotency.StatusFailed),
			wantCode: platform.CodeIdentityEmailDeliveryFailed,
		},
		{
			name:     "unresolved",
			state:    PasswordResetOperationCommitUnknown,
			claim:    passwordResetRecord(idempotency.StatusProcessing),
			wantCode: platform.CodeIdentityOperationUncertain,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPasswordResetFixture(
				passwordResetActor(false),
				passwordResetTarget(RoleMember),
			)
			authority := &temporaryCredentialResendAuthorityStub{
				existing: &PasswordResetCandidateResult{
					Action: PasswordResetCandidateExisting,
					Operation: PasswordResetOperation{
						ID: "resend-operation", UserID: passwordResetTargetID,
						RequestKind: PasswordResetResend,
						State:       test.state,
					},
					Claim: test.claim,
				},
			}
			fixture.service.authority = authority

			result, err := fixture.service.ResendPasswordResetCredential(
				context.Background(),
				PasswordResetRequest{
					Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
					TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
					TraceID: passwordResetTraceID,
				},
			)
			if (test.wantCode == "" && err != nil) ||
				(test.wantCode != "" && platform.PublicCode(err) != test.wantCode) ||
				result.DeliveryStatus != test.wantStatus {
				t.Fatalf("replay result=%#v error=%v", result, err)
			}
			if fixture.secrets.calls != 0 || fixture.ids.next != 0 ||
				fixture.hasher.hashCalls != 0 || fixture.repository.latestCalls != 0 ||
				len(fixture.mailer.messages) != 0 || authority.beginCalls != 0 ||
				fixture.mainTx.commitCalls != 1 {
				t.Fatalf("replay prepared sensitive state: fixture=%#v authority=%#v",
					fixture, authority)
			}
		})
	}
}

func TestTemporaryCredentialResendSucceededReplayPreservesPasswordResetExpiry(t *testing.T) {
	clock := &temporaryCredentialResendAdvancingClock{
		now:  passwordResetNow,
		step: time.Second,
	}
	fixture := newPasswordResetFixture(
		passwordResetActor(false),
		passwordResetTarget(RoleMember),
	)
	predecessor := "password-reset-predecessor"
	fixture.repository.latest = TemporaryCredential{
		ID: "password-reset-old", UserID: passwordResetTargetID,
		Purpose: TemporaryCredentialPasswordReset, Generation: 4,
		AuthGeneration: 4, SourceOperationID: &predecessor,
		State:            TemporaryCredentialActive,
		SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
		DeliveredToEmail: "target@example.invalid",
		IssuedAt:         passwordResetNow.Add(-time.Hour),
		ExpiresAt:        passwordResetNow.Add(-time.Hour).Add(temporaryCredentialLifetime),
	}
	fixture.repository.latestErr = nil
	authority := &temporaryCredentialResendAuthorityStub{}
	fixture.service.authority = authority
	fixture.service.dependencies.Clock = clock
	request := PasswordResetRequest{
		Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
		TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
		TraceID: passwordResetTraceID,
	}

	first, err := fixture.service.ResendPasswordResetCredential(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("first ResendPasswordResetCredential() error = %v", err)
	}
	sensitiveCalls := []int{
		fixture.secrets.calls,
		fixture.ids.next,
		fixture.hasher.hashCalls,
		len(fixture.mailer.messages),
	}
	claim := authority.claim
	operation := authority.candidate.Operation
	operation.State = PasswordResetOperationSucceeded
	authority.existing = &PasswordResetCandidateResult{
		Action:    PasswordResetCandidateExisting,
		Operation: operation,
		Claim:     claim,
	}
	fixture.transactor.transactions = append(
		fixture.transactor.transactions,
		&administratorTestTx{},
	)

	replay, err := fixture.service.ResendPasswordResetCredential(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("replay ResendPasswordResetCredential() error = %v", err)
	}
	if replay != first {
		t.Fatalf("replay result = %#v, want first result %#v", replay, first)
	}
	if got := []int{
		fixture.secrets.calls,
		fixture.ids.next,
		fixture.hasher.hashCalls,
		len(fixture.mailer.messages),
	}; !reflect.DeepEqual(got, sensitiveCalls) {
		t.Fatalf("replay sensitive calls = %v, want %v", got, sensitiveCalls)
	}
	if authority.beginCalls != 1 || authority.writes != 1 {
		t.Fatalf("authority begin/writes = %d/%d, want 1/1",
			authority.beginCalls, authority.writes)
	}
}

func TestTemporaryCredentialResendPasswordResetUsesDurableRecoveryChain(t *testing.T) {
	const protectionWindow = 10 * time.Minute
	tests := []struct {
		name         string
		mailErr      error
		wantAccepted bool
	}{
		{name: "accepted settlement", wantAccepted: true},
		{
			name:    "unknown remains gated until protection boundary",
			mailErr: errors.New("smtp acknowledgement unavailable"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newPasswordResetCrossCardHarness(
				t,
				passwordResetNow.Add(protectionWindow),
				test.mailErr,
			)
			harness.state.mu.Lock()
			harness.state.userID = passwordResetTargetID
			harness.state.operationID = "password-reset-id-2"
			harness.state.credentialID = "password-reset-id-1"
			harness.state.mu.Unlock()

			fixture := newPasswordResetFixture(
				passwordResetActor(false),
				passwordResetTarget(RoleMember),
			)
			predecessor := "password-reset-predecessor"
			fixture.repository.latest = TemporaryCredential{
				ID: "password-reset-old", UserID: passwordResetTargetID,
				Purpose: TemporaryCredentialPasswordReset, Generation: 4,
				AuthGeneration: 4, SourceOperationID: &predecessor,
				State:            TemporaryCredentialActive,
				SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
				DeliveredToEmail: "target@example.invalid",
				IssuedAt:         passwordResetNow.Add(-time.Hour),
				ExpiresAt: passwordResetNow.Add(-time.Hour).
					Add(temporaryCredentialLifetime),
			}
			fixture.repository.latestErr = nil
			authority := &temporaryCredentialResendAuthorityStub{}
			fixture.service.authority = authority
			fixture.service.dependencies.Transactions = harness.transactor
			fixture.service.dependencies.Mailer = harness.mail
			fixture.service.dependencies.Clock = harness.clock

			result, err := fixture.service.ResendPasswordResetCredential(
				context.Background(),
				PasswordResetRequest{
					Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
					TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
					TraceID: passwordResetTraceID,
				},
			)
			if test.wantAccepted {
				if err != nil || result.DeliveryStatus != passwordResetDeliveryAccepted {
					t.Fatalf("accepted resend result=%#v error=%v", result, err)
				}
			} else if platform.PublicCode(err) != platform.CodeIdentityOperationUncertain {
				t.Fatalf("unknown resend result=%#v error=%v", result, err)
			}
			if authority.candidate.Operation.ID != harness.state.operationID ||
				authority.candidate.Credential.ID != harness.state.credentialID ||
				authority.candidate.Operation.RequestKind != PasswordResetResend ||
				authority.candidate.Operation.PredecessorOperationID == nil ||
				*authority.candidate.Operation.PredecessorOperationID != predecessor ||
				len(harness.mail.messages) != 1 {
				t.Fatalf("durable resend candidate=%#v mail=%d",
					authority.candidate, len(harness.mail.messages))
			}

			if test.wantAccepted {
				if err := harness.worker.Reconcile(context.Background()); err != nil {
					t.Fatalf("accepted Reconcile() error = %v", err)
				}
				assertPasswordResetCrossCardPermit(t, harness.permit)
				assertPasswordResetCrossCardAccepted(t, harness.state.snapshot())
				if len(harness.mail.messages) != 1 {
					t.Fatalf("worker resent accepted mail: %d", len(harness.mail.messages))
				}
				return
			}

			if err := harness.worker.Reconcile(context.Background()); err != nil {
				t.Fatalf("before-boundary Reconcile() error = %v", err)
			}
			assertPasswordResetCrossCardUnresolved(t, harness.state.snapshot())
			assertPasswordResetCrossCardConsumersClosed(t, harness)
			harness.clock.Set(passwordResetNow.Add(protectionWindow))
			if err := harness.worker.Reconcile(context.Background()); err != nil {
				t.Fatalf("boundary Reconcile() error = %v", err)
			}
			snapshot := harness.state.snapshot()
			if !equalPasswordResetStrings(
				snapshot.workerCalls,
				[]string{"scan", "scan", "claim", "renew", "operator"},
			) {
				t.Fatalf("worker calls = %v", snapshot.workerCalls)
			}
			assertPasswordResetCrossCardInvalidated(t, snapshot)
			assertPasswordResetCrossCardOldLoginWorks(t, harness)
			if len(harness.mail.messages) != 1 {
				t.Fatalf("worker resent unknown mail: %d", len(harness.mail.messages))
			}
		})
	}
}

func TestTemporaryCredentialResendPasswordResetStalePreAcceptanceInvalidatesOnlyNew(
	t *testing.T,
) {
	const protectionWindow = 10 * time.Minute
	harness := newPasswordResetCrossCardHarness(
		t,
		passwordResetNow.Add(protectionWindow),
		nil,
	)
	harness.state.mu.Lock()
	harness.state.userID = passwordResetTargetID
	harness.state.operationID = "password-reset-resend-stale-operation"
	harness.state.credentialID = "password-reset-resend-stale-credential"
	harness.state.mu.Unlock()

	fixture := newPasswordResetFixture(
		passwordResetActor(false),
		passwordResetTarget(RoleMember),
	)
	predecessor := "password-reset-predecessor"
	fixture.repository.latest = TemporaryCredential{
		ID: "password-reset-old", UserID: passwordResetTargetID,
		Purpose: TemporaryCredentialPasswordReset, Generation: 4,
		AuthGeneration: 4, SourceOperationID: &predecessor,
		State:            TemporaryCredentialActive,
		SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
		DeliveredToEmail: "target@example.invalid",
		IssuedAt:         passwordResetNow.Add(-time.Hour),
		ExpiresAt: passwordResetNow.Add(-time.Hour).
			Add(temporaryCredentialLifetime),
	}
	fixture.repository.latestErr = nil
	authority := &temporaryCredentialResendAuthorityStub{}
	request := PasswordResetRequest{
		Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
		TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
		TraceID: passwordResetTraceID,
	}
	resend := *fixture.service
	resend.dependencies.Transactions = harness.transactor
	resend.dependencies.Clock = harness.clock
	resend.authority = passwordResetResendAuthority{
		delegate: authority,
		request:  request,
	}
	tx, err := harness.transactor.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	prepared, err := resend.prepareDurableCandidate(
		context.Background(),
		tx,
		request,
	)
	if err != nil {
		t.Fatalf("prepareDurableCandidate() error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("commit prepared resend: %v", err)
	}
	harness.state.mu.Lock()
	harness.state.operationID = prepared.operation.ID
	harness.state.credentialID = prepared.pending.ID
	harness.state.mu.Unlock()
	if !prepared.created ||
		prepared.operation.RequestKind != PasswordResetResend ||
		prepared.operation.PredecessorOperationID == nil ||
		*prepared.operation.PredecessorOperationID != predecessor ||
		authority.candidate.Operation.ID != prepared.operation.ID ||
		authority.candidate.Credential.ID != prepared.pending.ID {
		t.Fatalf("prepared stale resend = %#v candidate=%#v",
			prepared, authority.candidate)
	}

	if err := harness.worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("before-boundary Reconcile() error = %v", err)
	}
	before := harness.state.snapshot()
	if before.operationState != PasswordResetOperationPrepared ||
		before.deliveryState != PasswordResetDeliveryNotStarted ||
		before.settlementState != PasswordResetSettlementPending ||
		before.credentialState != TemporaryCredentialPendingDelivery ||
		before.idempotencyStatus != "processing" ||
		before.jobStatus != PasswordResetRecoveryJobReady ||
		before.attemptCount != 0 || before.auditCount != 0 ||
		!before.oldCapability || before.accelerationCalls != 0 ||
		!equalPasswordResetStrings(before.workerCalls, []string{"scan"}) {
		t.Fatalf("stale pre-acceptance state = %+v", before)
	}
	harness.clock.Set(passwordResetNow.Add(protectionWindow))
	if err := harness.worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("boundary Reconcile() error = %v", err)
	}
	snapshot := harness.state.snapshot()
	if !equalPasswordResetStrings(
		snapshot.workerCalls,
		[]string{"scan", "scan", "claim", "renew", "operator"},
	) {
		t.Fatalf("worker calls = %v", snapshot.workerCalls)
	}
	if snapshot.operationState != PasswordResetOperationInvalidated ||
		snapshot.deliveryState != PasswordResetDeliveryNotStarted ||
		snapshot.settlementState != PasswordResetSettlementInvalidated ||
		snapshot.credentialState != TemporaryCredentialRecoveryInvalidated ||
		snapshot.idempotencyStatus != "failed" ||
		snapshot.jobStatus != PasswordResetRecoveryJobSucceeded ||
		snapshot.attemptCount != 1 || snapshot.auditCount != 1 ||
		!snapshot.oldCapability || snapshot.activeSessions != 2 ||
		snapshot.accelerationCalls != 0 {
		t.Fatalf("stale recovery state = %+v", snapshot)
	}
	assertPasswordResetCrossCardOldLoginWorks(t, harness)
	if len(harness.mail.messages) != 0 {
		t.Fatalf("stale worker sent mail: %d", len(harness.mail.messages))
	}
}

func TestTemporaryCredentialResendPasswordResetRejectsMissingOrStaleContext(t *testing.T) {
	tests := []struct {
		name         string
		currentState TemporaryCredentialState
		withCurrent  bool
		preflightErr error
		existing     *PasswordResetCandidateResult
		wantCode     platform.Code
		wantStatus   int
	}{
		{
			name:         "missing predecessor",
			preflightErr: errTemporaryCredentialResendContextUnavailable,
			wantCode:     platform.CodeIdentityInvalidState,
			wantStatus:   http.StatusConflict,
		},
		{
			name:         "legal history stale",
			withCurrent:  true,
			currentState: TemporaryCredentialSuperseded,
			preflightErr: errTemporaryCredentialResendContextUnavailable,
			wantCode:     platform.CodeIdentityInvalidState,
			wantStatus:   http.StatusConflict,
		},
		{
			name:         "consumed predecessor",
			withCurrent:  true,
			currentState: TemporaryCredentialConsumed,
			preflightErr: errTemporaryCredentialResendContextUnavailable,
			wantCode:     platform.CodeIdentityInvalidState,
			wantStatus:   http.StatusConflict,
		},
		{
			name:         "unresolved Gate",
			withCurrent:  true,
			currentState: TemporaryCredentialActive,
			preflightErr: administratorOperationUncertain(
				ErrPasswordResetGateClosed,
			),
			wantCode:   platform.CodeIdentityOperationUncertain,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:         "new-key integrity drift",
			withCurrent:  true,
			currentState: TemporaryCredentialActive,
			preflightErr: administratorOperationUncertain(
				errors.New("password reset resend lineage integrity drift"),
			),
			wantCode:   platform.CodeIdentityOperationUncertain,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name: "same-key integrity hold",
			existing: &PasswordResetCandidateResult{
				Action: PasswordResetCandidateIntegrityHold,
			},
			wantCode:   platform.CodeIdentityOperationUncertain,
			wantStatus: http.StatusServiceUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPasswordResetFixture(
				passwordResetActor(false),
				passwordResetTarget(RoleMember),
			)
			if test.withCurrent {
				predecessor := "password-reset-predecessor"
				fixture.repository.latest = TemporaryCredential{
					ID: "password-reset-old", UserID: passwordResetTargetID,
					Purpose: TemporaryCredentialPasswordReset, Generation: 4,
					AuthGeneration: 4, SourceOperationID: &predecessor,
					State:            test.currentState,
					SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
					DeliveredToEmail: "target@example.invalid",
					IssuedAt:         passwordResetNow.Add(-time.Hour),
					ExpiresAt: passwordResetNow.Add(-time.Hour).
						Add(temporaryCredentialLifetime),
				}
				fixture.repository.latestErr = nil
			}
			authority := &temporaryCredentialResendAuthorityStub{
				existing:     test.existing,
				preflightErr: test.preflightErr,
			}
			fixture.service.authority = authority

			_, err := fixture.service.ResendPasswordResetCredential(
				context.Background(),
				PasswordResetRequest{
					Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
					TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
					TraceID: passwordResetTraceID,
				},
			)
			code := platform.PublicCode(err)
			if code != test.wantCode {
				t.Fatalf("error code = %q, want %q; err=%v", code, test.wantCode, err)
			}
			definition, ok := platform.DefaultRegistry.Definition(code)
			if !ok || definition.HTTPStatus != test.wantStatus {
				t.Fatalf(
					"HTTP status = %d/%t, want %d",
					definition.HTTPStatus,
					ok,
					test.wantStatus,
				)
			}
			if len(fixture.mailer.messages) != 0 ||
				fixture.mainTx.commitCalls != 0 ||
				authority.beginCalls != 0 ||
				authority.writes != 0 ||
				fixture.secrets.calls != 0 ||
				fixture.ids.next != 0 ||
				fixture.hasher.hashCalls != 0 {
				t.Fatalf(
					"rejected resend side effects: secret=%d ids=%d hash=%d "+
						"mail=%d commit=%d begin=%d writes=%d",
					fixture.secrets.calls, fixture.ids.next, fixture.hasher.hashCalls,
					len(fixture.mailer.messages), fixture.mainTx.commitCalls,
					authority.beginCalls, authority.writes,
				)
			}
		})
	}
}

func TestTemporaryCredentialResendPasswordResetPreflightUsesDurableAuthority(t *testing.T) {
	const (
		predecessorID = "33333333-3333-3333-3333-333333333333"
		replacedID    = "44444444-4444-4444-4444-444444444444"
	)
	tests := []struct {
		name      string
		gateClear bool
		evidence  *passwordResetResendLineageEvidence
		wantErr   error
		wantCode  platform.Code
		wantCalls int
	}{
		{
			name:      "unresolved Gate",
			gateClear: false,
			evidence: passwordResetTestPointer(
				passwordResetResendLineageEvidenceFixture(
					passwordResetTargetID, predecessorID, replacedID, 4,
				),
			),
			wantErr:  ErrPasswordResetGateClosed,
			wantCode: platform.CodeIdentityOperationUncertain,
		},
		{
			name:      "eligible current predecessor",
			gateClear: true,
			evidence: passwordResetTestPointer(
				passwordResetResendLineageEvidenceFixture(
					passwordResetTargetID, predecessorID, replacedID, 4,
				),
			),
			wantCalls: 1,
		},
		{
			name:      "consumed predecessor",
			gateClear: true,
			evidence: passwordResetTestPointer(
				passwordResetResendPermanentRootFixture(
					passwordResetTargetID, predecessorID, replacedID, 4,
				),
			),
			wantErr: errTemporaryCredentialResendContextUnavailable,
		},
		{
			name:      "legal history stale",
			gateClear: true,
			evidence: passwordResetTestPointer(
				passwordResetResendLegalSuccessorFixture(
					passwordResetTargetID, predecessorID, replacedID, 4,
				),
			),
			wantErr: errTemporaryCredentialResendContextUnavailable,
		},
		{
			name:      "integrity drift",
			gateClear: true,
			evidence: passwordResetTestPointer(
				passwordResetResendMissingCredentialFixture(
					passwordResetResendLineageEvidenceFixture(
						passwordResetTargetID, predecessorID, replacedID, 4,
					),
				),
			),
			wantCode: platform.CodeIdentityOperationUncertain,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			steps := make([]passwordResetStoreStep, 0, 2)
			if test.evidence != nil {
				steps = append(steps, passwordResetQueryStep(
					"predecessor.settlement_state",
					passwordResetResendLineageRowValues(*test.evidence)...,
				))
			}
			if test.evidence != nil &&
				classifyPasswordResetResendLineage(*test.evidence).status ==
					passwordResetResendLineageEligible {
				steps = append(steps, passwordResetQueryStep(
					"NOT current_account.auth_integrity_hold",
					test.gateClear,
				))
			}
			storeTx := &passwordResetStoreTransactionStub{steps: steps}
			tx := &temporaryCredentialResendStoreTx{
				passwordResetGateTransactionStub: &passwordResetGateTransactionStub{},
				store:                            storeTx,
			}
			delegate := &temporaryCredentialResendAuthorityStub{}
			authority := passwordResetResendAuthority{
				delegate: delegate,
				store:    &PasswordResetStore{},
			}
			candidate := PasswordResetCandidate{
				Operation: PasswordResetOperation{
					UserID:                 passwordResetTargetID,
					RequestKind:            PasswordResetResend,
					PredecessorOperationID: passwordResetTestPointer(predecessorID),
					ReplacedCredentialID:   passwordResetTestPointer(replacedID),
				},
			}

			err := authority.AuthorizeNewCandidate(
				context.Background(),
				tx,
				candidate,
			)
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("AuthorizeNewCandidate() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && test.wantCode == "" && err != nil {
				t.Fatalf("AuthorizeNewCandidate() error = %v", err)
			}
			if test.wantCode != "" && platform.PublicCode(err) != test.wantCode {
				t.Fatalf(
					"AuthorizeNewCandidate() code = %q, want %q",
					platform.PublicCode(err),
					test.wantCode,
				)
			}
			if delegate.preflightCalls != test.wantCalls {
				t.Fatalf(
					"delegate preflight calls = %d, want %d",
					delegate.preflightCalls,
					test.wantCalls,
				)
			}
			if delegate.beginCalls != 0 || delegate.writes != 0 ||
				len(storeTx.execArguments) != 0 {
				t.Fatalf(
					"preflight side effects: begin=%d writes=%d exec=%d",
					delegate.beginCalls,
					delegate.writes,
					len(storeTx.execArguments),
				)
			}
			storeTx.assertExhausted(t)
		})
	}
}

func TestTemporaryCredentialResendNewAdministratorReplacesOnlyAfterAccepted(t *testing.T) {
	mainTx := &administratorTestTx{}
	repository := newTemporaryCredentialResendAdministratorRepository()
	idempotencyStore := &administratorIdempotencyStub{
		steps: []administratorIdempotencyStep{{action: idempotency.BeginExecute}},
	}
	admin := newTemporaryCredentialResendAdministratorCoordinator(
		t, mainTx, repository, idempotencyStore, &administratorMailerStub{},
	)
	secrets := &passwordResetSecretStub{secret: passwordResetSecret}

	result, err := admin.resendNewAdministratorCredential(
		context.Background(),
		TemporaryCredentialResendRequest{
			Actor: passwordResetPrincipal(true), TargetID: passwordResetTargetID,
			TargetRole: RoleAdmin, Purpose: TemporaryCredentialResendNewAdministrator,
			IdempotencyKey: passwordResetKey, TraceID: passwordResetTraceID,
		},
		secrets,
	)
	if err != nil {
		t.Fatalf("resendNewAdministratorCredential() error = %v", err)
	}
	if result.DeliveryStatus != passwordResetDeliveryAccepted ||
		repository.updated.State != TemporaryCredentialSuperseded ||
		repository.credential.State != TemporaryCredentialActive ||
		mainTx.commitCalls != 1 {
		t.Fatalf("result=%#v updated=%#v inserted=%#v commit=%d",
			result, repository.updated, repository.credential, mainTx.commitCalls)
	}
	if len(idempotencyStore.completions) != 1 ||
		len(repository.operations) != 0 {
		t.Fatalf("idempotency completions=%d password-reset operations=%d",
			len(idempotencyStore.completions), len(repository.operations))
	}
}

func TestTemporaryCredentialResendNewAdministratorFailurePreservesOld(t *testing.T) {
	mainTx := &administratorTestTx{}
	failureTx := &administratorTestTx{}
	repository := newTemporaryCredentialResendAdministratorRepository()
	idempotencyStore := &administratorIdempotencyStub{
		steps: []administratorIdempotencyStep{
			{action: idempotency.BeginExecute},
			{action: idempotency.BeginExecute},
		},
	}
	admin := newTemporaryCredentialResendAdministratorCoordinatorWithTransactions(
		t, []pgx.Tx{mainTx, failureTx}, repository, idempotencyStore,
		&administratorMailerStub{err: errors.New("smtp rejected")},
	)

	_, err := admin.resendNewAdministratorCredential(
		context.Background(),
		TemporaryCredentialResendRequest{
			Actor: passwordResetPrincipal(true), TargetID: passwordResetTargetID,
			TargetRole: RoleAdmin, Purpose: TemporaryCredentialResendNewAdministrator,
			IdempotencyKey: passwordResetKey, TraceID: passwordResetTraceID,
		},
		&passwordResetSecretStub{secret: passwordResetSecret},
	)
	if platform.PublicCode(err) != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("error = %v", err)
	}
	if repository.current.State != TemporaryCredentialActive ||
		repository.updateCalls != 0 || repository.activateCalls != 0 ||
		mainTx.rollbackCalls != 1 || failureTx.commitCalls != 1 {
		t.Fatalf("current=%#v update=%d activate=%d rollback=%d failure commit=%d",
			repository.current, repository.updateCalls, repository.activateCalls,
			mainTx.rollbackCalls, failureTx.commitCalls)
	}
}

func TestTemporaryCredentialResendNewAdministratorReplayHasNoSensitivePreparation(
	t *testing.T,
) {
	tests := []struct {
		name       string
		existing   idempotency.BeginResult
		wantCode   platform.Code
		wantStatus string
		wantCommit int
	}{
		{
			name: "succeeded",
			existing: idempotency.BeginResult{
				Action: idempotency.BeginReplay,
				Record: passwordResetStableRecord(
					idempotency.StatusSucceeded,
					temporaryCredentialResendSuccessSnapshot(PasswordResetResult{
						UserID: passwordResetTargetID,
					}),
				),
			},
			wantStatus: passwordResetDeliveryAccepted,
			wantCommit: 1,
		},
		{
			name: "failed",
			existing: idempotency.BeginResult{
				Action: idempotency.BeginReplay,
				Record: passwordResetStableRecord(
					idempotency.StatusFailed,
					temporaryCredentialResendFailureSnapshot(),
				),
			},
			wantCode:   platform.CodeIdentityEmailDeliveryFailed,
			wantCommit: 1,
		},
		{
			name: "unresolved",
			existing: idempotency.BeginResult{
				Action: idempotency.BeginOperationUncertain,
				Record: passwordResetRecord(idempotency.StatusUncertain),
			},
			wantCode: platform.CodeIdentityOperationUncertain,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mainTx := &administratorTestTx{}
			repository := newTemporaryCredentialResendAdministratorRepository()
			idempotencyStore := &administratorIdempotencyStub{
				existing: &test.existing,
			}
			mail := &administratorMailerStub{}
			admin := newTemporaryCredentialResendAdministratorCoordinator(
				t,
				mainTx,
				repository,
				idempotencyStore,
				mail,
			)
			secrets := &passwordResetSecretStub{secret: passwordResetSecret}

			result, err := admin.resendNewAdministratorCredential(
				context.Background(),
				TemporaryCredentialResendRequest{
					Actor: passwordResetPrincipal(true), TargetID: passwordResetTargetID,
					TargetRole:     RoleAdmin,
					Purpose:        TemporaryCredentialResendNewAdministrator,
					IdempotencyKey: passwordResetKey, TraceID: passwordResetTraceID,
				},
				secrets,
			)
			if (test.wantCode == "" && err != nil) ||
				(test.wantCode != "" && platform.PublicCode(err) != test.wantCode) ||
				result.DeliveryStatus != test.wantStatus {
				t.Fatalf("replay result=%#v error=%v", result, err)
			}
			if secrets.calls != 0 || len(mail.messages) != 0 ||
				repository.updateCalls != 0 || repository.activateCalls != 0 ||
				len(repository.operations) != 0 ||
				len(idempotencyStore.beginRequests) != 0 ||
				len(idempotencyStore.completions) != 0 ||
				mainTx.commitCalls != test.wantCommit {
				t.Fatalf(
					"replay side effects: secret=%d mail=%d update=%d activate=%d "+
						"operations=%d begin=%d complete=%d commit=%d",
					secrets.calls,
					len(mail.messages),
					repository.updateCalls,
					repository.activateCalls,
					len(repository.operations),
					len(idempotencyStore.beginRequests),
					len(idempotencyStore.completions),
					mainTx.commitCalls,
				)
			}
		})
	}
}

func TestTemporaryCredentialResendSucceededReplayPreservesAdministratorExpiry(
	t *testing.T,
) {
	clock := &temporaryCredentialResendAdvancingClock{
		now:  passwordResetNow,
		step: time.Second,
	}
	idempotencyStore := &temporaryCredentialResendStatefulIdempotency{}
	repository := newTemporaryCredentialResendAdministratorRepository()
	mail := &administratorMailerStub{}
	transactions := []pgx.Tx{&administratorTestTx{}, &administratorTestTx{}}
	admin := newTemporaryCredentialResendAdministratorCoordinatorWithTransactions(
		t,
		transactions,
		repository,
		idempotencyStore,
		mail,
	)
	ids := &passwordResetIDStub{}
	hasher := &passwordResetHasherStub{}
	admin.dependencies.Clock = clock
	admin.dependencies.IDs = ids
	admin.dependencies.PasswordHasher = hasher
	secrets := &passwordResetSecretStub{secret: passwordResetSecret}
	request := TemporaryCredentialResendRequest{
		Actor: passwordResetPrincipal(true), TargetID: passwordResetTargetID,
		TargetRole: RoleAdmin, Purpose: TemporaryCredentialResendNewAdministrator,
		IdempotencyKey: passwordResetKey, TraceID: passwordResetTraceID,
	}

	first, err := admin.resendNewAdministratorCredential(
		context.Background(),
		request,
		secrets,
	)
	if err != nil {
		t.Fatalf("first resendNewAdministratorCredential() error = %v", err)
	}
	sensitiveCalls := []int{secrets.calls, ids.next, hasher.hashCalls, len(mail.messages)}

	replay, err := admin.resendNewAdministratorCredential(
		context.Background(),
		request,
		secrets,
	)
	if err != nil {
		t.Fatalf("replay resendNewAdministratorCredential() error = %v", err)
	}
	if replay != first {
		t.Fatalf("replay result = %#v, want first result %#v", replay, first)
	}
	if got := []int{
		secrets.calls, ids.next, hasher.hashCalls, len(mail.messages),
	}; !reflect.DeepEqual(got, sensitiveCalls) {
		t.Fatalf("replay sensitive calls = %v, want %v", got, sensitiveCalls)
	}
	if repository.updateCalls != 1 || repository.activateCalls != 1 {
		t.Fatalf("replay business writes changed to update/activate = %d/%d",
			repository.updateCalls, repository.activateCalls)
	}
}

type temporaryCredentialResendServiceStub struct {
	request TemporaryCredentialResendRequest
	result  PasswordResetResult
	err     error
	calls   int
}

func (s *temporaryCredentialResendServiceStub) ResendTemporaryCredential(
	_ context.Context,
	request TemporaryCredentialResendRequest,
) (PasswordResetResult, error) {
	s.calls++
	s.request = request
	return s.result, s.err
}

type temporaryCredentialResendAuthorityStub struct {
	existing       *PasswordResetCandidateResult
	beginErr       error
	preflightErr   error
	candidate      PasswordResetCandidate
	claim          idempotency.Record
	preflightCalls int
	beginCalls     int
	writes         int
}

type temporaryCredentialResendStoreTx struct {
	*passwordResetGateTransactionStub
	store *passwordResetStoreTransactionStub
}

func (tx *temporaryCredentialResendStoreTx) Exec(
	ctx context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	return tx.store.Exec(ctx, statement, arguments...)
}

func (tx *temporaryCredentialResendStoreTx) QueryRow(
	ctx context.Context,
	statement string,
	arguments ...any,
) pgx.Row {
	return tx.store.QueryRow(ctx, statement, arguments...)
}

func (s *temporaryCredentialResendAuthorityStub) ClassifyExistingCandidate(
	_ context.Context,
	_ idempotency.Transaction,
	_ string,
	_ idempotency.BeginRequest,
	_ func() (string, error),
) (PasswordResetCandidateResult, bool, error) {
	if s.existing == nil {
		return PasswordResetCandidateResult{}, false, nil
	}
	return *s.existing, true, nil
}

func (s *temporaryCredentialResendAuthorityStub) AuthorizeNewCandidate(
	_ context.Context,
	_ idempotency.Transaction,
	_ PasswordResetCandidate,
) error {
	s.preflightCalls++
	return s.preflightErr
}

func (s *temporaryCredentialResendAuthorityStub) BeginOrClassifyCandidate(
	_ context.Context,
	_ idempotency.Transaction,
	candidate PasswordResetCandidate,
) (PasswordResetCandidateResult, error) {
	s.beginCalls++
	s.candidate = candidate
	if s.beginErr != nil {
		return PasswordResetCandidateResult{}, s.beginErr
	}
	s.writes++
	s.claim = passwordResetRecord(idempotency.StatusProcessing)
	s.claim.CreatedAt = candidate.IdempotencyRequest.Now
	return PasswordResetCandidateResult{
		Action: PasswordResetCandidateCreated, Operation: candidate.Operation,
		Claim: s.claim,
	}, nil
}

type temporaryCredentialResendAdministratorRepository struct {
	administratorCreateRepositoryStub
	target        User
	current       TemporaryCredential
	updated       TemporaryCredential
	updateCalls   int
	activateCalls int
	operations    []PasswordResetOperation
}

func newTemporaryCredentialResendAdministratorRepository() *temporaryCredentialResendAdministratorRepository {
	email := "target@example.invalid"
	issuedAt := passwordResetNow.Add(-time.Hour)
	repository := &temporaryCredentialResendAdministratorRepository{
		target: User{
			ID: passwordResetTargetID, Email: &email, Role: RoleAdmin,
			Status: UserStatusActive, CredentialMode: CredentialModeTemporary,
		},
		current: TemporaryCredential{
			ID: "administrator-old", UserID: passwordResetTargetID,
			Purpose: TemporaryCredentialNewAdministrator, Generation: 1,
			State:            TemporaryCredentialActive,
			SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
			DeliveredToEmail: email, IssuedAt: issuedAt,
			ExpiresAt: issuedAt.Add(temporaryCredentialLifetime),
		},
	}
	repository.authorizationUser = User{
		ID: passwordResetActorID, Role: RoleAdmin, IsInitialAdmin: true,
		Status: UserStatusActive, CredentialMode: CredentialModePermanent,
	}
	return repository
}

func (r *temporaryCredentialResendAdministratorRepository) LockTemporaryCredentialResendTarget(
	context.Context,
	pgx.Tx,
	string,
) (User, TemporaryCredential, error) {
	return r.target, r.current, nil
}

func (r *temporaryCredentialResendAdministratorRepository) UpdateTemporaryCredential(
	_ context.Context,
	_ pgx.Tx,
	credential TemporaryCredential,
) error {
	r.updateCalls++
	r.updated = credential
	r.current = credential
	return nil
}

func (r *temporaryCredentialResendAdministratorRepository) ActivateTemporaryCredential(
	_ context.Context,
	_ pgx.Tx,
	credentialID string,
	_ time.Time,
) error {
	r.activateCalls++
	if r.credential.ID != credentialID {
		return errors.New("unexpected credential")
	}
	r.credential.State = TemporaryCredentialActive
	return nil
}

func newTemporaryCredentialResendAdministratorCoordinator(
	t *testing.T,
	tx pgx.Tx,
	repository *temporaryCredentialResendAdministratorRepository,
	idempotencyStore AdministratorIdempotencyStore,
	mail *administratorMailerStub,
) *AdministratorCreateCoordinator {
	t.Helper()
	return newTemporaryCredentialResendAdministratorCoordinatorWithTransactions(
		t, []pgx.Tx{tx}, repository, idempotencyStore, mail,
	)
}

func newTemporaryCredentialResendAdministratorCoordinatorWithTransactions(
	t *testing.T,
	transactions []pgx.Tx,
	repository *temporaryCredentialResendAdministratorRepository,
	idempotencyStore AdministratorIdempotencyStore,
	mail *administratorMailerStub,
) *AdministratorCreateCoordinator {
	t.Helper()
	service, err := NewAdministratorCreateService(AdministratorCreateDependencies{
		Transactions:       &administratorSequenceTransactor{transactions: transactions},
		Repository:         repository,
		InformationNotices: administratorInformationNoticeStub{},
		Gate:               administratorGateStub{},
		PasswordPolicy:     &passwordResetPolicyStub{},
		PasswordHasher:     administratorPasswordHasherStub{},
		Mailer:             mail,
		Idempotency: temporaryCredentialResendIdempotencyAdapter{
			delegate: idempotencyStore,
		},
		IdempotencyKeys: idempotency.DigestKeyring{
			ActiveVersion: "v1",
			Keys:          map[string][]byte{"v1": []byte("administrator-test-digest-key")},
		},
		IdempotencyLock: time.Minute, IdempotencyTTL: 24 * time.Hour,
		SMTPOperationBudget: time.Second,
		Audit:               &temporaryCredentialResendAuditStub{},
		IDs:                 &bootstrapTestIDs{},
		Clock:               passwordResetClockStub{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type temporaryCredentialResendAdvancingClock struct {
	now   time.Time
	step  time.Duration
	calls int
}

func (c *temporaryCredentialResendAdvancingClock) Now() time.Time {
	now := c.now.Add(time.Duration(c.calls) * c.step)
	c.calls++
	return now
}

type temporaryCredentialResendIdempotencyAdapter struct {
	delegate AdministratorIdempotencyStore
}

func (a temporaryCredentialResendIdempotencyAdapter) ResolveExisting(
	ctx context.Context,
	tx idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	return a.delegate.ResolveExisting(ctx, tx, request)
}

func (a temporaryCredentialResendIdempotencyAdapter) Begin(
	ctx context.Context,
	tx idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	result, err := a.delegate.Begin(ctx, tx, request)
	if err == nil && result.Record.CreatedAt.IsZero() {
		result.Record.CreatedAt = request.Now
	}
	return result, err
}

func (a temporaryCredentialResendIdempotencyAdapter) Complete(
	ctx context.Context,
	tx idempotency.Transaction,
	completion idempotency.Completion,
) error {
	return a.delegate.Complete(ctx, tx, completion)
}

func (a temporaryCredentialResendIdempotencyAdapter) MarkUncertain(
	ctx context.Context,
	tx idempotency.Transaction,
	transition idempotency.Transition,
) error {
	return a.delegate.MarkUncertain(ctx, tx, transition)
}

type temporaryCredentialResendStatefulIdempotency struct {
	record      idempotency.Record
	completions int
}

func (s *temporaryCredentialResendStatefulIdempotency) ResolveExisting(
	_ context.Context,
	_ idempotency.Transaction,
	_ idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	if s.completions == 0 {
		return idempotency.BeginResult{}, false, nil
	}
	return idempotency.BeginResult{
		Action: idempotency.BeginReplay,
		Record: s.record,
	}, true, nil
}

func (s *temporaryCredentialResendStatefulIdempotency) Begin(
	_ context.Context,
	_ idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	s.record = idempotency.Record{
		ScopeHash: []byte("scope"), KeyLocator: []byte("locator"),
		KeyDigest: []byte("key"), RequestDigest: []byte("request"),
		DigestKeyVersion: "v1", Status: idempotency.StatusProcessing,
		ExecutionGeneration: 1, CreatedAt: request.Now, UpdatedAt: request.Now,
		ExpiresAt: request.ExpiresAt,
	}
	return idempotency.BeginResult{
		Action: idempotency.BeginExecute,
		Record: s.record,
	}, nil
}

func (s *temporaryCredentialResendStatefulIdempotency) Complete(
	_ context.Context,
	_ idempotency.Transaction,
	completion idempotency.Completion,
) error {
	s.completions++
	snapshot := completion.ResponseSnapshot
	s.record.Status = completion.Status
	s.record.ResultCode = completion.ResultCode
	s.record.ResponseSnapshot = &snapshot
	s.record.UpdatedAt = completion.Now
	return nil
}

func (*temporaryCredentialResendStatefulIdempotency) MarkUncertain(
	context.Context,
	idempotency.Transaction,
	idempotency.Transition,
) error {
	return nil
}

type temporaryCredentialResendAuditStub struct{}

func (*temporaryCredentialResendAuditStub) Append(
	context.Context,
	pgx.Tx,
	audit.Event,
) error {
	return nil
}

func bytesContain(value []byte, text string) bool {
	return strings.Contains(string(value), text)
}

func temporaryCredentialResendProtectedHandler(
	t *testing.T,
	handler *Handler,
) http.Handler {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		passwordResetPrincipalResolver{principal: passwordResetPrincipal(true)},
		func() time.Time { return passwordResetNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	return authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(handler.ResendTemporaryCredential),
	)
}

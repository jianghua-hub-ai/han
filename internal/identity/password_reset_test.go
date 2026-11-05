package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	passwordResetActorID  = "11111111-1111-4111-8111-111111111111"
	passwordResetTargetID = "22222222-2222-4222-8222-222222222222"
	passwordResetTraceID  = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	passwordResetKey      = "password-reset-key-0001"
	passwordResetSecret   = "Temporary123"
)

var passwordResetNow = time.Date(2026, time.October, 24, 10, 0, 0, 0, time.UTC)

type passwordResetAuthorizationCase struct {
	name       string
	principal  session.Principal
	actor      User
	target     User
	targetRole Role
	wantCode   platform.Code
}

var passwordResetAuthorizationCases = []passwordResetAuthorizationCase{
	{
		name: "ordinary administrator resets member", principal: passwordResetPrincipal(false),
		actor: passwordResetActor(false), target: passwordResetTarget(RoleMember),
		targetRole: RoleMember,
	},
	{
		name: "initial administrator resets administrator", principal: passwordResetPrincipal(true),
		actor: passwordResetActor(true), target: passwordResetTarget(RoleAdmin),
		targetRole: RoleAdmin,
	},
	{
		name: "member actor rejected",
		principal: session.Principal{
			UserID: passwordResetActorID, Role: string(RoleMember),
			Status: string(UserStatusActive),
		},
		actor: passwordResetActor(false), target: passwordResetTarget(RoleMember),
		targetRole: RoleMember, wantCode: platform.CodeAuthForbidden,
	},
	{
		name:      "ordinary administrator cannot reset administrator",
		principal: passwordResetPrincipal(false), actor: passwordResetActor(false),
		target: passwordResetTarget(RoleAdmin), targetRole: RoleAdmin,
		wantCode: platform.CodeAuthForbidden,
	},
	{
		name: "stale principal after actor suspension", principal: passwordResetPrincipal(false),
		actor: passwordResetSuspendedActor(), target: passwordResetTarget(RoleMember),
		targetRole: RoleMember, wantCode: platform.CodeAuthForbidden,
	},
	{
		name: "stale principal after initial flag drift", principal: passwordResetPrincipal(true),
		actor: passwordResetActor(false), target: passwordResetTarget(RoleAdmin),
		targetRole: RoleAdmin, wantCode: platform.CodeAuthForbidden,
	},
	{
		name: "self target rejected", principal: passwordResetPrincipal(false),
		actor: passwordResetActor(false), target: passwordResetSelfTarget(),
		targetRole: RoleMember, wantCode: platform.CodeAuthForbidden,
	},
	{
		name: "initial administrator target protected", principal: passwordResetPrincipal(true),
		actor: passwordResetActor(true), target: passwordResetInitialTarget(),
		targetRole: RoleAdmin, wantCode: platform.CodeIdentityInitialAdminProtected,
	},
	{
		name: "deleted target rejected", principal: passwordResetPrincipal(false),
		actor: passwordResetActor(false), target: passwordResetDeletedTarget(),
		targetRole: RoleMember, wantCode: platform.CodeIdentityInvalidState,
	},
	{
		name: "recovery review target rejected", principal: passwordResetPrincipal(false),
		actor: passwordResetActor(false), target: passwordResetRecoveryTarget(),
		targetRole: RoleMember, wantCode: platform.CodeIdentityInvalidState,
	},
}

func runPasswordResetAuthorizationCase(t *testing.T, test passwordResetAuthorizationCase) {
	t.Helper()
	fixture := newPasswordResetFixture(test.actor, test.target)
	result, err := fixture.service.ResetPassword(context.Background(), PasswordResetRequest{
		Actor: test.principal, TargetID: test.target.ID,
		TargetRole: test.targetRole, IdempotencyKey: passwordResetKey,
		TraceID: passwordResetTraceID,
	})
	if test.wantCode != "" {
		if got := platform.PublicCode(err); got != test.wantCode {
			t.Fatalf("ResetPassword() code = %q, want %q, err=%v", got, test.wantCode, err)
		}
		if len(fixture.mailer.messages) != 0 || fixture.repository.insertCalls != 0 {
			t.Fatalf("forbidden reset produced side effects: mail=%d insert=%d",
				len(fixture.mailer.messages), fixture.repository.insertCalls)
		}
		return
	}
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if result.UserID != test.target.ID || result.DeliveryStatus != passwordResetDeliveryAccepted {
		t.Fatalf("ResetPassword() result = %#v", result)
	}
	if len(fixture.mailer.messages) != 1 {
		t.Fatalf("Mailer.Send() calls = %d, want 1", len(fixture.mailer.messages))
	}
}

type passwordResetIdempotencyCase struct {
	name       string
	resolve    idempotency.BeginResult
	resolveErr error
	wantCode   platform.Code
	wantReplay bool
}

var passwordResetIdempotencyCases = []passwordResetIdempotencyCase{
	{
		name: "same key same digest success replay",
		resolve: idempotency.BeginResult{
			Action: idempotency.BeginReplay,
			Record: passwordResetStableRecord(
				idempotency.StatusSucceeded,
				idempotency.ResponseSnapshot{
					Outcome: passwordResetOutcomeSucceeded, ResourceID: passwordResetTargetID,
				},
			),
		},
		wantReplay: true,
	},
	{
		name: "stable delivery failure replay",
		resolve: idempotency.BeginResult{
			Action: idempotency.BeginReplay,
			Record: passwordResetStableRecord(
				idempotency.StatusFailed,
				idempotency.ResponseSnapshot{
					Outcome:   passwordResetOutcomeFailed,
					ErrorCode: string(platform.CodeIdentityEmailDeliveryFailed),
				},
			),
		},
		wantCode: platform.CodeIdentityEmailDeliveryFailed,
	},
	{
		name: "same key different digest conflict", resolveErr: idempotency.ErrDigestConflict,
		wantCode: platform.CodeIdentityIdempotencyConflict,
	},
	{
		name: "processing is not executed",
		resolve: idempotency.BeginResult{
			Action: idempotency.BeginInProgress,
			Record: passwordResetProcessingRecord(passwordResetNow.Add(time.Minute)),
		},
		wantCode: platform.CodeIdentityOperationUncertain,
	},
	{
		name: "uncertain is not executed",
		resolve: idempotency.BeginResult{
			Action: idempotency.BeginOperationUncertain,
			Record: passwordResetRecord(idempotency.StatusUncertain),
		},
		wantCode: platform.CodeIdentityOperationUncertain,
	},
	{
		name: "recovery review required is not executed",
		resolve: idempotency.BeginResult{
			Action: idempotency.BeginOperationUncertain,
			Record: passwordResetRecord(idempotency.StatusRecoveryReviewRequired),
		},
		wantCode: platform.CodeIdentityOperationUncertain,
	},
}

func runPasswordResetIdempotencyCase(t *testing.T, test passwordResetIdempotencyCase) {
	t.Helper()
	fixture := newPasswordResetFixture(
		passwordResetActor(false),
		passwordResetTarget(RoleMember),
	)
	fixture.idempotency.resolveFound = true
	fixture.idempotency.resolve = test.resolve
	fixture.idempotency.resolveErr = test.resolveErr
	result, err := fixture.service.ResetPassword(context.Background(), PasswordResetRequest{
		Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
		TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
		TraceID: passwordResetTraceID,
	})
	if test.wantCode == "" && err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if test.wantCode != "" && platform.PublicCode(err) != test.wantCode {
		t.Fatalf("ResetPassword() code = %q, want %q, err=%v",
			platform.PublicCode(err), test.wantCode, err)
	}
	if test.wantReplay &&
		(result.UserID != passwordResetTargetID ||
			result.DeliveryStatus != passwordResetDeliveryAccepted) {
		t.Fatalf("replay result = %#v", result)
	}
	if fixture.secrets.calls != 0 || len(fixture.mailer.messages) != 0 ||
		fixture.repository.latestCalls != 0 || fixture.repository.insertCalls != 0 ||
		len(fixture.audit.recorded) != 0 || fixture.idempotency.completeCalls != 0 {
		t.Fatalf("idempotency stop/replay produced side effects: fixture=%#v", fixture)
	}
	if len(fixture.repository.lockedUserIDs) != 2 ||
		fixture.repository.lockedUserIDs[0] != passwordResetActorID ||
		fixture.repository.lockedUserIDs[1] != passwordResetTargetID {
		t.Fatalf("idempotency classification locked users = %v, want actor then target",
			fixture.repository.lockedUserIDs)
	}
}

var passwordResetPreparationFailureCases = []passwordResetFailureCase{
	{name: "secret generation", inject: func(f *passwordResetFixture) {
		f.secrets.err = errors.New("random source failed")
	}, wantCode: platform.CodePlatformInternalError},
	{name: "password policy", inject: func(f *passwordResetFixture) {
		f.policy.err = errors.New("policy failed")
	}, wantCode: platform.CodePlatformInternalError},
	{name: "credential identifier", inject: func(f *passwordResetFixture) {
		f.ids.failAt = 1
	}, wantCode: platform.CodePlatformInternalError},
	{name: "operation identifier", inject: func(f *passwordResetFixture) {
		f.ids.failAt = 2
	}, wantCode: platform.CodePlatformInternalError},
	{name: "credential hash", inject: func(f *passwordResetFixture) {
		f.hasher.hashErr = errors.New("hash failed")
	}, wantCode: platform.CodePlatformInternalError},
	{name: "current credential query", inject: func(f *passwordResetFixture) {
		f.repository.latestErr = errors.New("query failed")
	}, wantCode: platform.CodePlatformDatabaseUnavailable},
}

func runPasswordResetPreparationFailure(t *testing.T, test passwordResetFailureCase) {
	t.Helper()
	fixture := newPasswordResetFixture(
		passwordResetActor(false),
		passwordResetTarget(RoleMember),
	)
	test.inject(fixture)
	fixture.rebuildService()
	_, err := fixture.service.ResetPassword(context.Background(), PasswordResetRequest{
		Actor: passwordResetPrincipal(false), TargetID: passwordResetTargetID,
		TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
		TraceID: passwordResetTraceID,
	})
	if got := platform.PublicCode(err); got != test.wantCode {
		t.Fatalf("ResetPassword() code = %q, want %q, err=%v",
			got, test.wantCode, err)
	}
	if len(fixture.mailer.messages) != 0 || fixture.repository.writeCalls() != 0 {
		t.Fatalf("preparation failure produced mail/write side effects: mail=%d writes=%d",
			len(fixture.mailer.messages), fixture.repository.writeCalls())
	}
	if fixture.mainTx.commitCalls != 0 || fixture.mainTx.rollbackCalls != 1 {
		t.Fatalf("candidate transaction commit=%d rollback=%d, want 0/1",
			fixture.mainTx.commitCalls, fixture.mainTx.rollbackCalls)
	}
}

type passwordResetFailureCase struct {
	name     string
	inject   func(*passwordResetFixture)
	wantCode platform.Code
}

type passwordResetHandlerCase struct {
	name       string
	method     string
	path       string
	wantStatus int
	wantCode   platform.Code
	wantRole   Role
	wantCalls  int
}

var passwordResetHandlerCases = []passwordResetHandlerCase{
	{
		name: "member path", method: http.MethodPost,
		path:       "/api/v1/admin/users/" + passwordResetTargetID + "/password-reset",
		wantStatus: http.StatusOK, wantRole: RoleMember, wantCalls: 1,
	},
	{
		name: "administrator path", method: http.MethodPost,
		path:       "/api/v1/admin/administrators/" + passwordResetTargetID + "/password-reset",
		wantStatus: http.StatusOK, wantRole: RoleAdmin, wantCalls: 1,
	},
	{
		name: "malformed path", method: http.MethodPost,
		path:       "/api/v1/admin/users/" + passwordResetTargetID + "/password-reset/extra",
		wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
	},
	{
		name: "malformed uuid", method: http.MethodPost,
		path:       "/api/v1/admin/users/not-a-uuid/password-reset",
		wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
	},
	{
		name: "wrong method", method: http.MethodGet,
		path:       "/api/v1/admin/users/" + passwordResetTargetID + "/password-reset",
		wantStatus: http.StatusMethodNotAllowed,
		wantCode:   platform.CodeRequestMethodNotAllowed,
	},
}

func runPasswordResetHandlerCase(t *testing.T, test passwordResetHandlerCase) {
	t.Helper()
	service := &passwordResetServiceStub{result: PasswordResetResult{
		UserID:                       passwordResetTargetID,
		TemporaryCredentialExpiresAt: passwordResetNow.Add(temporaryCredentialLifetime),
		DeliveryStatus:               passwordResetDeliveryAccepted,
	}}
	handler, err := NewPasswordResetHandler(service)
	if err != nil {
		t.Fatalf("NewPasswordResetHandler() error = %v", err)
	}
	response := httptest.NewRecorder()
	passwordResetProtectedHandler(t, handler).ServeHTTP(
		response,
		passwordResetHTTPRequest(t, test.method, test.path),
	)
	if response.Code != test.wantStatus || service.calls != test.wantCalls {
		t.Fatalf("status=%d calls=%d, want=%d/%d, body=%s",
			response.Code, service.calls, test.wantStatus, test.wantCalls, response.Body.String())
	}
	if test.wantCalls == 1 {
		assertPasswordResetHandlerSuccess(t, service.request, response, test.wantRole)
		return
	}
	assertPasswordResetHandlerError(t, response, test)
}

func assertPasswordResetHandlerSuccess(
	t *testing.T,
	request PasswordResetRequest,
	response *httptest.ResponseRecorder,
	wantRole Role,
) {
	t.Helper()
	if request.TargetID != passwordResetTargetID || request.TargetRole != wantRole ||
		request.IdempotencyKey != passwordResetKey || request.TraceID != passwordResetTraceID {
		t.Fatalf("service request = %#v", request)
	}
	if strings.Contains(response.Body.String(), passwordResetSecret) ||
		strings.Contains(response.Body.String(), "password") {
		t.Fatalf("success response leaked secret material: %s", response.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode success response: %v", err)
	}
	if len(body) != 3 {
		t.Fatalf("success fields = %v, want 3", body)
	}
}

func assertPasswordResetHandlerError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	test passwordResetHandlerCase,
) {
	t.Helper()
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Code != test.wantCode || envelope.TraceID != passwordResetTraceID {
		t.Fatalf("error envelope = %#v", envelope)
	}
	if test.wantStatus == http.StatusMethodNotAllowed &&
		response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("Allow=%q, want POST", response.Header().Get("Allow"))
	}
}

func newPasswordResetFixture(actor, target User) *passwordResetFixture {
	mainTx := &administratorTestTx{}
	claimTx := &administratorTestTx{}
	evidenceTx := &administratorTestTx{}
	claimTx.row = passwordResetStoreRow{values: []any{target.ID}}
	evidenceTx.row = passwordResetStoreRow{values: []any{target.ID}}
	repository := &passwordResetRepositoryStub{
		users:     map[string]User{actor.ID: actor, target.ID: target},
		latestErr: pgx.ErrNoRows,
	}
	store := &passwordResetIdempotencyStub{
		begin: idempotency.BeginResult{
			Action: idempotency.BeginExecute,
			Record: passwordResetRecord(idempotency.StatusProcessing),
		},
	}
	policy := &passwordResetPolicyStub{}
	hasher := &passwordResetHasherStub{}
	ids := &passwordResetIDStub{}
	fixture := &passwordResetFixture{
		mainTx: mainTx, claimTx: claimTx,
		transactor: &passwordResetTransactorStub{
			transactions: []pgx.Tx{mainTx, claimTx, evidenceTx},
			beginErrs:    map[int]error{},
		},
		repository:  repository,
		secrets:     &passwordResetSecretStub{secret: passwordResetSecret},
		mailer:      &passwordResetMailerStub{},
		idempotency: store,
		audit:       &passwordResetAuditStub{},
		policy:      policy,
		hasher:      hasher,
		ids:         ids,
	}
	fixture.rebuildService()
	return fixture
}

func (f *passwordResetFixture) rebuildService() {
	dependencies := passwordResetTestDependencies(
		f.transactor,
		f.repository,
		f.secrets,
		f.mailer,
		f.idempotency,
		f.audit,
	)
	dependencies.PasswordPolicy = f.policy
	dependencies.PasswordHasher = f.hasher
	dependencies.IDs = f.ids
	service, err := NewPasswordResetService(dependencies)
	if err != nil {
		panic(err)
	}
	service.authority = &passwordResetFixtureAuthority{idempotency: f.idempotency}
	f.service = service
}

func passwordResetTestDependencies(
	transactor RegistrationTransactor,
	repository PasswordResetRepository,
	secrets PasswordResetSecretGenerator,
	resetMailer mailer.Mailer,
	store AdministratorIdempotencyStore,
	auditAppender RegistrationAuditAppender,
) PasswordResetDependencies {
	return PasswordResetDependencies{
		Transactions: transactor, Repository: repository,
		Store:          &PasswordResetStore{},
		PasswordPolicy: &passwordResetPolicyStub{},
		PasswordHasher: administratorPasswordHasherStub{},
		Secrets:        secrets, Mailer: resetMailer,
		IdempotencyKeys: idempotency.DigestKeyring{
			ActiveVersion: "v1",
			Keys:          map[string][]byte{"v1": []byte("password-reset-test-digest-key")},
		},
		IdempotencyLock: time.Minute, IdempotencyTTL: 24 * time.Hour,
		SMTPOperationBudget: time.Second, Audit: auditAppender,
		IDs: &passwordResetIDStub{}, Clock: passwordResetClockStub{},
	}
}

type passwordResetFixture struct {
	service     *PasswordResetCoordinator
	mainTx      *administratorTestTx
	claimTx     *administratorTestTx
	transactor  *passwordResetTransactorStub
	repository  *passwordResetRepositoryStub
	secrets     *passwordResetSecretStub
	mailer      *passwordResetMailerStub
	idempotency *passwordResetIdempotencyStub
	audit       *passwordResetAuditStub
	policy      *passwordResetPolicyStub
	hasher      *passwordResetHasherStub
	ids         *passwordResetIDStub
}

type passwordResetTransactorStub struct {
	transactions []pgx.Tx
	beginErrs    map[int]error
	next         int
}

func (t *passwordResetTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	index := t.next
	t.next++
	if err := t.beginErrs[index]; err != nil {
		return nil, err
	}
	if index >= len(t.transactions) {
		return nil, errors.New("unexpected password reset transaction")
	}
	return t.transactions[index], nil
}

type passwordResetEvidenceContextKey struct{}

type passwordResetContextObservation struct {
	err         error
	hasDeadline bool
	value       any
}

func observePasswordResetContext(ctx context.Context) passwordResetContextObservation {
	_, hasDeadline := ctx.Deadline()
	return passwordResetContextObservation{
		err:         ctx.Err(),
		hasDeadline: hasDeadline,
		value:       ctx.Value(passwordResetEvidenceContextKey{}),
	}
}

type passwordResetContextAwareTransactor struct {
	transactions []pgx.Tx
	observations []passwordResetContextObservation
	next         int
}

func (t *passwordResetContextAwareTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	observation := observePasswordResetContext(ctx)
	t.observations = append(t.observations, observation)
	if observation.err != nil {
		return nil, observation.err
	}
	if t.next >= len(t.transactions) {
		return nil, errors.New("unexpected password reset transaction")
	}
	tx := t.transactions[t.next]
	t.next++
	return tx, nil
}

type passwordResetRecordingTx struct {
	administratorTestTx
	statements    []string
	failStatement string
	execErr       error
}

func (tx *passwordResetRecordingTx) Exec(
	_ context.Context,
	statement string,
	_ ...any,
) (pgconn.CommandTag, error) {
	tx.statements = append(tx.statements, statement)
	if statement == tx.failStatement {
		return pgconn.CommandTag{}, tx.execErr
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

type passwordResetRecoveryWindowTx struct {
	passwordResetRecordingTx
	nextAttemptAt     time.Time
	accelerationCalls int
}

func (tx *passwordResetRecoveryWindowTx) Exec(
	ctx context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	tag, err := tx.passwordResetRecordingTx.Exec(ctx, statement, arguments...)
	if err != nil || statement != acceleratePasswordResetRecoveryJobSQL {
		return tag, err
	}
	if len(arguments) != 2 {
		return pgconn.CommandTag{}, errors.New("invalid recovery acceleration arguments")
	}
	nextAttemptAt, ok := arguments[0].(time.Time)
	if !ok {
		return pgconn.CommandTag{}, errors.New("invalid recovery acceleration time")
	}
	tx.nextAttemptAt = nextAttemptAt
	tx.accelerationCalls++
	return tag, nil
}

type passwordResetContextAwareTx struct {
	passwordResetRecordingTx
	observations []passwordResetContextObservation
}

func (tx *passwordResetContextAwareTx) observe(ctx context.Context) error {
	observation := observePasswordResetContext(ctx)
	tx.observations = append(tx.observations, observation)
	return observation.err
}

func (tx *passwordResetContextAwareTx) Exec(
	ctx context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	if err := tx.observe(ctx); err != nil {
		return pgconn.CommandTag{}, err
	}
	return tx.passwordResetRecordingTx.Exec(ctx, statement, arguments...)
}

func (tx *passwordResetContextAwareTx) QueryRow(
	ctx context.Context,
	statement string,
	arguments ...any,
) pgx.Row {
	if err := tx.observe(ctx); err != nil {
		return bootstrapTestRow{err: err}
	}
	return tx.passwordResetRecordingTx.QueryRow(ctx, statement, arguments...)
}

func (tx *passwordResetContextAwareTx) Commit(ctx context.Context) error {
	if err := tx.observe(ctx); err != nil {
		return err
	}
	return tx.passwordResetRecordingTx.Commit(ctx)
}

func (tx *passwordResetContextAwareTx) Rollback(ctx context.Context) error {
	if err := tx.observe(ctx); err != nil {
		return err
	}
	return tx.passwordResetRecordingTx.Rollback(ctx)
}

type passwordResetRepositoryStub struct {
	users              map[string]User
	latest             TemporaryCredential
	latestErr          error
	updated            TemporaryCredential
	inserted           TemporaryCredential
	mode               CredentialMode
	revokeReason       string
	updateTx           pgx.Tx
	insertTx           pgx.Tx
	modeTx             pgx.Tx
	revokeTx           pgx.Tx
	latestCalls        int
	updateCalls        int
	insertCalls        int
	modeCalls          int
	revokeCalls        int
	updateErr          error
	insertErr          error
	modeErr            error
	revokeErr          error
	events             *[]string
	lockedUserIDs      []string
	commitApplied      bool
	commitVerification passwordResetCommitVerification
	verifyCommitErr    error
	verifyCommitCalls  int
	invalidateCalls    int
	invalidateTx       pgx.Tx
	invalidateErr      error
	invalidateErrs     []error
	auditVerifications []passwordResetAuditVerification
	verifyAuditCalls   int
}

func (r *passwordResetRepositoryStub) LockUser(
	_ context.Context,
	_ pgx.Tx,
	id string,
) (User, error) {
	r.lockedUserIDs = append(r.lockedUserIDs, id)
	if r.events != nil {
		event := "lock-target"
		if id == passwordResetActorID {
			event = "lock-actor"
		}
		*r.events = append(*r.events, event)
	}
	user, ok := r.users[id]
	if !ok {
		return User{}, pgx.ErrNoRows
	}
	return user, nil
}

func (r *passwordResetRepositoryStub) LockLatestTemporaryCredential(
	_ context.Context,
	_ pgx.Tx,
	_ string,
) (TemporaryCredential, error) {
	r.latestCalls++
	if r.events != nil {
		*r.events = append(*r.events, "lock-credential")
	}
	return r.latest, r.latestErr
}

func (r *passwordResetRepositoryStub) UpdateTemporaryCredential(
	_ context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	r.updateCalls++
	r.updateTx, r.updated = tx, credential
	if r.events != nil {
		*r.events = append(*r.events, "supersede")
	}
	return r.updateErr
}

func (r *passwordResetRepositoryStub) InsertTemporaryCredential(
	_ context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	r.insertCalls++
	r.insertTx, r.inserted = tx, credential
	if r.events != nil {
		*r.events = append(*r.events, "insert")
	}
	return r.insertErr
}

func (r *passwordResetRepositoryStub) SetCredentialMode(
	_ context.Context,
	tx pgx.Tx,
	_ string,
	mode CredentialMode,
	_ time.Time,
) error {
	r.modeCalls++
	r.modeTx, r.mode = tx, mode
	if r.events != nil {
		*r.events = append(*r.events, "mode")
	}
	return r.modeErr
}

func (r *passwordResetRepositoryStub) RevokeSessions(
	_ context.Context,
	tx pgx.Tx,
	_ string,
	_ time.Time,
	reason string,
) error {
	r.revokeCalls++
	r.revokeTx, r.revokeReason = tx, reason
	if r.events != nil {
		*r.events = append(*r.events, "sessions")
	}
	return r.revokeErr
}

func (r *passwordResetRepositoryStub) VerifyPasswordResetCommit(
	_ context.Context,
	_ pgx.Tx,
	_ passwordResetCommitEvidence,
) (passwordResetCommitVerification, error) {
	r.verifyCommitCalls++
	if r.verifyCommitErr != nil {
		return passwordResetCommitVerification{}, r.verifyCommitErr
	}
	if r.commitApplied {
		return passwordResetCommitVerification{
			outcome: passwordResetCommitApplied,
		}, nil
	}
	if r.commitVerification != (passwordResetCommitVerification{}) {
		return r.commitVerification, nil
	}
	return passwordResetCommitVerification{
		outcome: passwordResetCommitNotApplied,
	}, nil
}

func (r *passwordResetRepositoryStub) InvalidatePasswordResetCredential(
	_ context.Context,
	tx pgx.Tx,
	_ TemporaryCredential,
	_ time.Time,
) error {
	r.invalidateCalls++
	r.invalidateTx = tx
	if len(r.invalidateErrs) > 0 {
		err := r.invalidateErrs[0]
		r.invalidateErrs = r.invalidateErrs[1:]
		return err
	}
	return r.invalidateErr
}

func (r *passwordResetRepositoryStub) VerifyPasswordResetAudit(
	_ context.Context,
	_ pgx.Tx,
	_ audit.Event,
) (passwordResetAuditVerification, error) {
	index := r.verifyAuditCalls
	r.verifyAuditCalls++
	if index >= len(r.auditVerifications) {
		return passwordResetAuditVerification{}, nil
	}
	verification := r.auditVerifications[index]
	return verification, verification.err
}

func (r *passwordResetRepositoryStub) writeCalls() int {
	return r.updateCalls + r.insertCalls + r.modeCalls + r.revokeCalls
}

type passwordResetIdempotencyStub struct {
	resolve        idempotency.BeginResult
	resolveFound   bool
	resolveErr     error
	begin          idempotency.BeginResult
	beginErr       error
	completion     idempotency.Completion
	transition     idempotency.Transition
	completeTx     idempotency.Transaction
	completeErr    error
	resolveCalls   int
	beginCalls     int
	completeCalls  int
	uncertainCalls int
	uncertainTx    idempotency.Transaction
	uncertainErr   error
	uncertainErrs  []error
	events         *[]string
}

func (s *passwordResetIdempotencyStub) ResolveExisting(
	_ context.Context,
	_ idempotency.Transaction,
	_ idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	s.resolveCalls++
	if s.events != nil {
		*s.events = append(*s.events, "idempotency-resolve")
	}
	return s.resolve, s.resolveFound, s.resolveErr
}

func (s *passwordResetIdempotencyStub) Begin(
	_ context.Context,
	_ idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	s.beginCalls++
	if s.events != nil {
		*s.events = append(*s.events, "idempotency-begin")
	}
	result := s.begin
	if result.Record.CreatedAt.IsZero() {
		result.Record.CreatedAt = request.Now
	}
	return result, s.beginErr
}

func (s *passwordResetIdempotencyStub) Complete(
	_ context.Context,
	tx idempotency.Transaction,
	completion idempotency.Completion,
) error {
	s.completeCalls++
	s.completeTx, s.completion = tx, completion
	if s.events != nil {
		*s.events = append(*s.events, "idempotency-complete")
	}
	return s.completeErr
}

func (s *passwordResetIdempotencyStub) MarkUncertain(
	_ context.Context,
	tx idempotency.Transaction,
	transition idempotency.Transition,
) error {
	s.uncertainCalls++
	s.uncertainTx = tx
	s.transition = transition
	if len(s.uncertainErrs) > 0 {
		err := s.uncertainErrs[0]
		s.uncertainErrs = s.uncertainErrs[1:]
		return err
	}
	return s.uncertainErr
}

type passwordResetSecretStub struct {
	secret string
	err    error
	calls  int
}

func (s *passwordResetSecretStub) Generate() (string, error) {
	s.calls++
	return s.secret, s.err
}

type passwordResetMailerStub struct {
	messages   []mailer.Message
	err        error
	events     *[]string
	beforeSend func()
	send       func(context.Context)
}

func (s *passwordResetMailerStub) Send(ctx context.Context, message mailer.Message) error {
	if s.beforeSend != nil {
		s.beforeSend()
	}
	s.messages = append(s.messages, message)
	if s.events != nil {
		*s.events = append(*s.events, "mail")
	}
	if s.send != nil {
		s.send(ctx)
	}
	return s.err
}

type passwordResetAuditStub struct {
	event    audit.Event
	recorded []audit.Event
	tx       pgx.Tx
	errs     []error
	events   *[]string
}

func (s *passwordResetAuditStub) Append(
	_ context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	s.tx, s.event = tx, event
	s.recorded = append(s.recorded, event)
	if s.events != nil {
		*s.events = append(*s.events, "audit")
	}
	if len(s.errs) == 0 {
		return nil
	}
	err := s.errs[0]
	s.errs = s.errs[1:]
	return err
}

type passwordResetPolicyStub struct{ err error }

func (s *passwordResetPolicyStub) Validate(value string) error {
	if s.err != nil {
		return s.err
	}
	if len(value) < 8 || !strings.ContainsAny(value, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") ||
		!strings.ContainsAny(value, "0123456789") {
		return errors.New("password policy rejected value")
	}
	return nil
}

type passwordResetHasherStub struct {
	hashErr     error
	verifyErr   error
	hashCalls   int
	verifyCalls int
}

func (s *passwordResetHasherStub) Hash(string) (string, error) {
	s.hashCalls++
	if s.hashErr != nil {
		return "", s.hashErr
	}
	return "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0", nil
}

func (s *passwordResetHasherStub) Verify(string, string) (bool, bool, error) {
	s.verifyCalls++
	if s.verifyErr != nil {
		return false, false, s.verifyErr
	}
	return true, false, nil
}

type passwordResetIDStub struct {
	next   int
	failAt int
}

func (s *passwordResetIDStub) NewID() (string, error) {
	s.next++
	if s.next == s.failAt {
		return "", errors.New("identifier generation failed")
	}
	return "password-reset-id-" + string(rune('0'+s.next)), nil
}

type passwordResetClockStub struct{}

func (passwordResetClockStub) Now() time.Time { return passwordResetNow }

type passwordResetServiceStub struct {
	result  PasswordResetResult
	err     error
	request PasswordResetRequest
	calls   int
}

func (s *passwordResetServiceStub) ResetPassword(
	_ context.Context,
	request PasswordResetRequest,
) (PasswordResetResult, error) {
	s.calls++
	s.request = request
	return s.result, s.err
}

type passwordResetPrincipalResolver struct{ principal session.Principal }

func (s passwordResetPrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return s.principal, nil
}

func passwordResetProtectedHandler(t *testing.T, handler *Handler) http.Handler {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		passwordResetPrincipalResolver{principal: passwordResetPrincipal(true)},
		func() time.Time { return passwordResetNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	return authorizer.Protect(session.AccessGeneral, http.HandlerFunc(handler.ResetPassword))
}

func passwordResetHTTPRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	ctx, err := platform.ContextWithTraceID(request.Context(), passwordResetTraceID)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request = request.WithContext(ctx)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "session-token"})
	request.Header.Set(platform.CSRFHeaderName, "csrf-token")
	request.Header.Set("Idempotency-Key", passwordResetKey)
	return request
}

func passwordResetPrincipal(initial bool) session.Principal {
	return session.Principal{
		UserID: passwordResetActorID, Role: string(RoleAdmin),
		IsInitialAdmin: initial, Status: string(UserStatusActive),
	}
}

func passwordResetActor(initial bool) User {
	completedAt := passwordResetNow.Add(-time.Hour)
	return User{
		ID: passwordResetActorID, Role: RoleAdmin, IsInitialAdmin: initial,
		Status: UserStatusActive, CredentialMode: CredentialModePermanent,
		OnboardingCompletedAt: &completedAt,
	}
}

func passwordResetSuspendedActor() User {
	user := passwordResetActor(false)
	user.Status = UserStatusSuspended
	return user
}

func passwordResetTarget(role Role) User {
	email := "target@example.invalid"
	return User{
		ID: passwordResetTargetID, Email: &email, Role: role,
		Status: UserStatusActive, CredentialMode: CredentialModePermanent,
	}
}

func passwordResetSelfTarget() User {
	user := passwordResetTarget(RoleMember)
	user.ID = passwordResetActorID
	return user
}

func passwordResetInitialTarget() User {
	user := passwordResetTarget(RoleAdmin)
	user.IsInitialAdmin = true
	return user
}

func passwordResetDeletedTarget() User {
	user := passwordResetTarget(RoleMember)
	user.Status = UserStatusDeleted
	return user
}

func passwordResetRecoveryTarget() User {
	user := passwordResetTarget(RoleMember)
	user.RecoveryRequired = true
	user.CredentialMode = CredentialModeRecoveryLocked
	return user
}

func passwordResetCurrentCredential(target User) TemporaryCredential {
	return passwordResetCurrentCredentialForPurpose(
		target,
		TemporaryCredentialPasswordReset,
	)
}

func passwordResetCurrentCredentialForPurpose(
	target User,
	purpose TemporaryCredentialPurpose,
) TemporaryCredential {
	credential, err := NewTemporaryCredential(
		administratorPasswordHasherStub{},
		TemporaryCredentialInput{
			ID: "current-credential", UserID: target.ID,
			Purpose: purpose, Generation: 1,
			Secret: "Existing123", DeliveredToEmail: *target.Email,
			IssuedAt: passwordResetNow.Add(-time.Hour),
		},
	)
	if err != nil {
		panic(err)
	}
	active, err := TransitionTemporaryCredential(
		credential,
		TemporaryCredentialActive,
		passwordResetNow.Add(-30*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	return active
}

func passwordResetRecord(status idempotency.Status) idempotency.Record {
	return idempotency.Record{
		ScopeHash: []byte("scope"), KeyLocator: []byte("locator"),
		KeyDigest: []byte("key"), RequestDigest: []byte("request"),
		DigestKeyVersion: "v1", Status: status, ExecutionGeneration: 1,
		CreatedAt: passwordResetNow,
	}
}

func passwordResetProcessingRecord(lockedUntil time.Time) idempotency.Record {
	record := passwordResetRecord(idempotency.StatusProcessing)
	record.LockedUntil = &lockedUntil
	return record
}

func passwordResetStableRecord(
	status idempotency.Status,
	snapshot idempotency.ResponseSnapshot,
) idempotency.Record {
	record := passwordResetRecord(status)
	record.ResponseSnapshot = &snapshot
	return record
}

func assertPasswordResetEventSafe(t *testing.T, event audit.Event, traceID string) {
	t.Helper()
	if event.TraceID != traceID {
		t.Fatalf("audit trace_id=%q, want=%q", event.TraceID, traceID)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	if bytes.Contains(encoded, []byte(passwordResetSecret)) {
		t.Fatalf("audit event leaked temporary password: %s", encoded)
	}
}

func assertNoPasswordResetSecret(t *testing.T, values ...any) {
	t.Helper()
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal value: %v", err)
		}
		if bytes.Contains(encoded, []byte(passwordResetSecret)) {
			t.Fatalf("value leaked temporary password: %s", encoded)
		}
	}
}

func TestPasswordResetAuthorization(t *testing.T) {
	for _, test := range passwordResetAuthorizationCases {
		t.Run(test.name, func(t *testing.T) {
			runPasswordResetAuthorizationCase(t, test)
		})
	}
}

func TestPasswordResetIdempotency(t *testing.T) {
	for _, test := range passwordResetIdempotencyCases {
		t.Run(test.name, func(t *testing.T) {
			runPasswordResetIdempotencyCase(t, test)
		})
	}
}

func TestPasswordResetPreparationFailures(t *testing.T) {
	for _, test := range passwordResetPreparationFailureCases {
		t.Run(test.name, func(t *testing.T) {
			runPasswordResetPreparationFailure(t, test)
		})
	}
}

func TestPasswordResetHandler(t *testing.T) {
	for _, test := range passwordResetHandlerCases {
		t.Run(test.name, func(t *testing.T) {
			runPasswordResetHandlerCase(t, test)
		})
	}
}

func TestPasswordResetSameKeyReplayPrecedesSensitivePreparation(t *testing.T) {
	tests := []struct {
		name     string
		state    PasswordResetOperationState
		wantCode platform.Code
	}{
		{name: "succeeded", state: PasswordResetOperationSucceeded},
		{
			name: "failed", state: PasswordResetOperationFailed,
			wantCode: platform.CodeIdentityEmailDeliveryFailed,
		},
		{
			name: "unresolved", state: PasswordResetOperationCommitUnknown,
			wantCode: platform.CodeIdentityOperationUncertain,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor, target := passwordResetActor(false), passwordResetTarget(RoleMember)
			candidateTx := &administratorTestTx{}
			repository := &passwordResetRepositoryStub{
				users: map[string]User{actor.ID: actor, target.ID: target},
			}
			secrets := &passwordResetSecretStub{secret: passwordResetSecret}
			mail := &passwordResetMailerStub{}
			hasher := &passwordResetHasherStub{}
			ids := &passwordResetIDStub{}
			dependencies := passwordResetTestDependencies(
				&passwordResetTransactorStub{
					transactions: []pgx.Tx{candidateTx}, beginErrs: map[int]error{},
				},
				repository,
				secrets,
				mail,
				&passwordResetIdempotencyStub{},
				&passwordResetAuditStub{},
			)
			dependencies.Store = &PasswordResetStore{}
			dependencies.PasswordHasher = hasher
			dependencies.IDs = ids
			service, err := NewPasswordResetService(dependencies)
			if err != nil {
				t.Fatalf("NewPasswordResetService() error = %v", err)
			}
			claim := passwordResetRecord(idempotency.StatusProcessing)
			claim.CreatedAt = passwordResetNow
			service.authority = &passwordResetCandidateAuthorityStub{
				result: PasswordResetCandidateResult{
					Action: PasswordResetCandidateExisting,
					Operation: PasswordResetOperation{
						ID: "existing-operation", UserID: target.ID, State: test.state,
					},
					Claim: claim,
				},
			}

			result, err := service.ResetPassword(context.Background(), PasswordResetRequest{
				Actor: passwordResetPrincipal(false), TargetID: target.ID,
				TargetRole: RoleMember, IdempotencyKey: passwordResetKey,
				TraceID: passwordResetTraceID,
			})
			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("ResetPassword() error = %v", err)
				}
				if result.DeliveryStatus != passwordResetDeliveryAccepted {
					t.Fatalf("ResetPassword() result = %#v", result)
				}
			} else if got := platform.PublicCode(err); got != test.wantCode {
				t.Fatalf("ResetPassword() code = %q, want %q, err=%v", got, test.wantCode, err)
			}
			if secrets.calls != 0 || ids.next != 0 || hasher.hashCalls != 0 ||
				hasher.verifyCalls != 0 || repository.latestCalls != 0 ||
				len(mail.messages) != 0 {
				t.Fatalf(
					"sensitive replay side effects: secret=%d ids=%d hash=%d verify=%d latest=%d mail=%d",
					secrets.calls, ids.next, hasher.hashCalls, hasher.verifyCalls,
					repository.latestCalls, len(mail.messages),
				)
			}
		})
	}
}

func TestPasswordResetUsesDurableCandidateBeforeDispatch(t *testing.T) {
	actor, target := passwordResetActor(false), passwordResetTarget(RoleMember)
	candidateTx := &administratorTestTx{}
	dispatchTx := &administratorTestTx{}
	evidenceTx := &administratorTestTx{}
	dispatchTx.row = passwordResetStoreRow{values: []any{target.ID}}
	evidenceTx.row = passwordResetStoreRow{values: []any{target.ID}}
	authority := &passwordResetCandidateAuthorityStub{}
	mail := &passwordResetMailerStub{}
	mail.beforeSend = func() {
		if candidateTx.commitCalls != 1 || dispatchTx.commitCalls != 1 {
			t.Fatal("SMTP ran before candidate and dispatch ownership were durable")
		}
	}
	dependencies := passwordResetTestDependencies(
		&passwordResetTransactorStub{
			transactions: []pgx.Tx{candidateTx, dispatchTx, evidenceTx},
			beginErrs:    map[int]error{},
		},
		&passwordResetRepositoryStub{
			users: map[string]User{actor.ID: actor, target.ID: target}, latestErr: pgx.ErrNoRows,
		},
		&passwordResetSecretStub{secret: passwordResetSecret},
		mail,
		&passwordResetIdempotencyStub{},
		&passwordResetAuditStub{},
	)
	dependencies.Store = &PasswordResetStore{}
	service, err := NewPasswordResetService(dependencies)
	if err != nil {
		t.Fatalf("NewPasswordResetService() error = %v", err)
	}
	service.authority = authority

	result, err := service.ResetPassword(context.Background(), PasswordResetRequest{
		Actor: passwordResetPrincipal(false), TargetID: target.ID,
		TargetRole: RoleMember, IdempotencyKey: passwordResetKey, TraceID: passwordResetTraceID,
	})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if authority.calls != 1 || authority.candidate.Operation.State != PasswordResetOperationPrepared ||
		authority.candidate.Credential.State != TemporaryCredentialPendingDelivery ||
		authority.candidate.RecoveryJob.Status != PasswordResetRecoveryJobReady {
		t.Fatalf("durable candidate call = %#v", authority)
	}
	smtpLatestEnd := passwordResetNow.Add(dependencies.SMTPOperationBudget)
	if !authority.candidate.RecoveryJob.NextAttemptAt.After(smtpLatestEnd) {
		t.Fatalf(
			"job next_attempt_at=%s, want after SMTP latest end %s",
			authority.candidate.RecoveryJob.NextAttemptAt, smtpLatestEnd,
		)
	}
	if len(mail.messages) != 1 || result.DeliveryStatus != passwordResetDeliveryAccepted {
		t.Fatalf("mail=%d result=%#v", len(mail.messages), result)
	}
}

func TestPasswordResetRequiresExistingDurableStore(t *testing.T) {
	dependencies := passwordResetTestDependencies(
		&passwordResetTransactorStub{beginErrs: map[int]error{}},
		&passwordResetRepositoryStub{},
		&passwordResetSecretStub{secret: passwordResetSecret},
		&passwordResetMailerStub{},
		&passwordResetIdempotencyStub{},
		&passwordResetAuditStub{},
	)
	dependencies.Store = nil
	if _, err := NewPasswordResetService(dependencies); err == nil {
		t.Fatal("NewPasswordResetService() accepted a nil durable Store")
	}
	dependencies.Store = &PasswordResetStore{}
	if _, err := NewPasswordResetService(dependencies); err != nil {
		t.Fatalf("NewPasswordResetService() with durable Store error = %v", err)
	}
}

func TestPasswordResetDispatchCommitUnknownDoesNotSend(t *testing.T) {
	dispatchTx := &administratorTestTx{commitErr: errors.New("ack lost")}
	dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	mail := &passwordResetMailerStub{}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions: &passwordResetTransactorStub{
			transactions: []pgx.Tx{dispatchTx}, beginErrs: map[int]error{},
		},
		Store: &PasswordResetStore{}, Mailer: mail, IDs: &passwordResetIDStub{}, Clock: passwordResetClockStub{},
		SMTPOperationBudget: time.Second,
	}}
	target := passwordResetTarget(RoleMember)
	_, err := coordinator.dispatchPasswordReset(context.Background(), passwordResetPrepared{
		target: target, secret: passwordResetSecret,
		operation: PasswordResetOperation{ID: "operation-one", UserID: passwordResetTargetID},
	})
	if platform.PublicCode(err) != platform.CodeIdentityOperationUncertain {
		t.Fatalf("dispatch error = %v", err)
	}
	if len(mail.messages) != 0 {
		t.Fatalf("mailer calls = %d, want 0", len(mail.messages))
	}
}

func TestPasswordResetAcceptedEvidenceLeavesCredentialForDurableSettlement(t *testing.T) {
	dispatchTx := &administratorTestTx{}
	evidenceTx := &administratorTestTx{}
	dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	evidenceTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	mail := &passwordResetMailerStub{}
	repository := &passwordResetRepositoryStub{}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions: &passwordResetTransactorStub{
			transactions: []pgx.Tx{dispatchTx, evidenceTx}, beginErrs: map[int]error{},
		},
		Repository: repository, Store: &PasswordResetStore{}, Mailer: mail, IDs: &passwordResetIDStub{},
		Clock: passwordResetClockStub{}, SMTPOperationBudget: time.Second,
	}}
	target := passwordResetTarget(RoleMember)
	result, err := coordinator.dispatchPasswordReset(context.Background(), passwordResetPrepared{
		target: target, secret: passwordResetSecret,
		pending:   TemporaryCredential{ID: "credential-one"},
		operation: PasswordResetOperation{ID: "operation-one", UserID: passwordResetTargetID},
		result:    PasswordResetResult{UserID: target.ID},
	})
	if err != nil || result.DeliveryStatus != passwordResetDeliveryAccepted {
		t.Fatalf("dispatch result=%#v error=%#v cause=%v", result, err, errors.Unwrap(err))
	}
	if repository.writeCalls() != 0 {
		t.Fatalf("request path changed credential/session capability: %d writes", repository.writeCalls())
	}
}

func TestPasswordResetExplicitRejectionRecordsStableFailure(t *testing.T) {
	tests := []struct {
		name string
		kind PasswordResetRequestKind
	}{
		{name: "initial reset", kind: PasswordResetInitialReset},
		{name: "ordinary resend", kind: PasswordResetResend},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatchTx := &administratorTestTx{}
			rejectedTx := &passwordResetRecordingTx{}
			dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			rejectedTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			repository := &passwordResetRepositoryStub{}
			mail := &passwordResetMailerStub{
				err: fmt.Errorf("provider response: %w", mailer.ErrMessageRejected),
			}
			transactor := &passwordResetTransactorStub{
				transactions: []pgx.Tx{dispatchTx, rejectedTx},
				beginErrs:    map[int]error{},
			}
			coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
				Transactions:        transactor,
				Repository:          repository,
				Store:               &PasswordResetStore{},
				Mailer:              mail,
				IDs:                 &passwordResetIDStub{},
				Clock:               passwordResetClockStub{},
				Audit:               &passwordResetAuditStub{},
				SMTPOperationBudget: time.Second,
			}}
			target := passwordResetTarget(RoleMember)

			_, err := coordinator.dispatchPasswordReset(
				context.Background(),
				passwordResetPrepared{
					target:  target,
					secret:  passwordResetSecret,
					pending: TemporaryCredential{ID: "credential-one"},
					operation: PasswordResetOperation{
						ID:          "operation-one",
						UserID:      target.ID,
						RequestKind: test.kind,
					},
				},
			)

			if got := platform.PublicCode(err); got != platform.CodeIdentityEmailDeliveryFailed {
				t.Fatalf("dispatch code = %q, want email delivery failed; err=%v", got, err)
			}
			if len(mail.messages) != 1 || repository.writeCalls() != 0 ||
				transactor.next != 2 {
				t.Fatalf(
					"mail/repository writes/transactions = %d/%d/%d",
					len(mail.messages),
					repository.writeCalls(),
					transactor.next,
				)
			}
			statements := strings.Join(rejectedTx.statements, "\n")
			for _, want := range []string{
				"delivery_state='rejected'",
				"state='recovery_invalidated'",
				"status='succeeded'",
			} {
				if !strings.Contains(statements, want) {
					t.Fatalf("rejected evidence statements = %q, want %q", statements, want)
				}
			}
			if strings.Contains(statements, "delivery_state='unknown'") ||
				strings.Contains(statements, "delivery_state='accepted'") {
				t.Fatalf("rejected evidence used another delivery classification: %q", statements)
			}
		})
	}
}

func TestPasswordResetUnclassifiedSMTPProtocolErrorsRemainUnknown(t *testing.T) {
	tests := []struct {
		name    string
		kind    PasswordResetRequestKind
		mailErr error
	}{
		{
			name: "initial reset greeting rejection",
			kind: PasswordResetInitialReset,
			mailErr: fmt.Errorf(
				"negotiate SMTP greeting: %w",
				&textproto.Error{Code: 550, Msg: "greeting rejected"},
			),
		},
		{
			name: "ordinary resend unexpected MAIL reply",
			kind: PasswordResetResend,
			mailErr: fmt.Errorf(
				"send SMTP MAIL command: %w",
				&textproto.Error{Code: 354, Msg: "unexpected intermediate reply"},
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatchTx := &administratorTestTx{}
			unknownTx := &passwordResetRecordingTx{}
			dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			unknownTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			repository := &passwordResetRepositoryStub{}
			mail := &passwordResetMailerStub{err: test.mailErr}
			transactor := &passwordResetTransactorStub{
				transactions: []pgx.Tx{dispatchTx, unknownTx},
				beginErrs:    map[int]error{},
			}
			coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
				Transactions:        transactor,
				Repository:          repository,
				Store:               &PasswordResetStore{},
				Mailer:              mail,
				IDs:                 &passwordResetIDStub{},
				Clock:               passwordResetClockStub{},
				Audit:               &passwordResetAuditStub{},
				SMTPOperationBudget: time.Second,
			}}
			target := passwordResetTarget(RoleMember)

			_, err := coordinator.dispatchPasswordReset(
				context.Background(),
				passwordResetPrepared{
					target:  target,
					secret:  passwordResetSecret,
					pending: TemporaryCredential{ID: "credential-one"},
					operation: PasswordResetOperation{
						ID:          "operation-one",
						UserID:      target.ID,
						RequestKind: test.kind,
					},
				},
			)

			if got := platform.PublicCode(err); got != platform.CodeIdentityOperationUncertain {
				t.Fatalf("dispatch code = %q, want operation uncertain; err=%v", got, err)
			}
			if errors.Is(test.mailErr, mailer.ErrMessageRejected) {
				t.Fatalf("test mail error unexpectedly matches rejection sentinel: %v", test.mailErr)
			}
			if len(mail.messages) != 1 || repository.writeCalls() != 0 ||
				transactor.next != 2 {
				t.Fatalf(
					"mail/repository writes/transactions = %d/%d/%d",
					len(mail.messages),
					repository.writeCalls(),
					transactor.next,
				)
			}
			statements := strings.Join(unknownTx.statements, "\n")
			if !strings.Contains(statements, "state='commit_unknown'") ||
				!strings.Contains(statements, "delivery_state='unknown'") {
				t.Fatalf("unknown evidence statements = %q", statements)
			}
			if strings.Contains(statements, "delivery_state='rejected'") ||
				strings.Contains(statements, "recovery_invalidated") {
				t.Fatalf("unclassified SMTP error used rejected evidence: %q", statements)
			}
		})
	}
}

func TestPasswordResetExplicitRejectionFlowsThroughS11ResendConsumer(t *testing.T) {
	harness := newPasswordResetCrossCardHarness(
		t,
		passwordResetNow.Add(10*time.Minute),
		fmt.Errorf("provider response: %w", mailer.ErrMessageRejected),
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

	if got := platform.PublicCode(err); got != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("resend result=%#v code=%q, want email delivery failed; err=%v", result, got, err)
	}
	if authority.candidate.Operation.ID != harness.state.operationID ||
		authority.candidate.Credential.ID != harness.state.credentialID ||
		authority.candidate.Operation.RequestKind != PasswordResetResend ||
		authority.candidate.Operation.PredecessorOperationID == nil ||
		*authority.candidate.Operation.PredecessorOperationID != predecessor ||
		len(harness.mail.messages) != 1 {
		t.Fatalf(
			"durable resend candidate=%#v mail=%d",
			authority.candidate,
			len(harness.mail.messages),
		)
	}
	snapshot := harness.state.snapshot()
	if snapshot.operationState != PasswordResetOperationFailed ||
		snapshot.deliveryState != PasswordResetDeliveryRejected ||
		snapshot.settlementState != PasswordResetSettlementInvalidated ||
		snapshot.credentialState != TemporaryCredentialRecoveryInvalidated ||
		snapshot.jobStatus != PasswordResetRecoveryJobSucceeded ||
		!snapshot.oldCapability ||
		snapshot.activeSessions != 2 {
		t.Fatalf("rejected resend state = %+v", snapshot)
	}
	if harness.gate.CheckLockedUser(
		context.Background(),
		&passwordResetCrossCardTx{state: harness.state},
		harness.state.userID,
	) != nil {
		t.Fatal("rejected resend left the shared password-reset Gate closed")
	}
}

func TestPasswordResetRejectedEvidenceFailureStaysUnresolved(t *testing.T) {
	tests := []struct {
		name          string
		beginErr      error
		failStatement string
		commitErr     error
	}{
		{
			name:     "begin failure",
			beginErr: errors.New("rejected evidence begin unavailable"),
		},
		{
			name:          "write failure",
			failStatement: recordPasswordResetRejectedSQL,
		},
		{
			name:      "commit acknowledgement lost",
			commitErr: errors.New("rejected evidence acknowledgement lost"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatchTx := &administratorTestTx{}
			dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			rejectedTx := &passwordResetRecordingTx{
				failStatement: test.failStatement,
				execErr:       errors.New("rejected evidence write unavailable"),
			}
			rejectedTx.commitErr = test.commitErr
			rejectedTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			transactions := []pgx.Tx{dispatchTx, rejectedTx}
			beginErrs := map[int]error{}
			if test.beginErr != nil {
				transactions = []pgx.Tx{dispatchTx}
				beginErrs[1] = test.beginErr
			}
			transactor := &passwordResetTransactorStub{
				transactions: transactions,
				beginErrs:    beginErrs,
			}
			repository := &passwordResetRepositoryStub{}
			coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
				Transactions: transactor,
				Repository:   repository,
				Store:        &PasswordResetStore{},
				Mailer: &passwordResetMailerStub{
					err: fmt.Errorf("provider response: %w", mailer.ErrMessageRejected),
				},
				IDs:                 &passwordResetIDStub{},
				Clock:               passwordResetClockStub{},
				Audit:               &passwordResetAuditStub{},
				SMTPOperationBudget: time.Second,
			}}
			target := passwordResetTarget(RoleMember)

			_, err := coordinator.dispatchPasswordReset(
				context.Background(),
				passwordResetPrepared{
					target:  target,
					secret:  passwordResetSecret,
					pending: TemporaryCredential{ID: "credential-one"},
					operation: PasswordResetOperation{
						ID: "operation-one", UserID: target.ID,
					},
				},
			)

			if got := platform.PublicCode(err); got != platform.CodeIdentityOperationUncertain {
				t.Fatalf("dispatch code = %q, want operation uncertain; err=%v", got, err)
			}
			if repository.writeCalls() != 0 {
				t.Fatalf("rejected evidence failure changed capability: %d writes", repository.writeCalls())
			}
		})
	}
}

func TestPasswordResetAcceptedEvidenceCommitUnknownAndCleanupUnavailableStaysUnresolved(t *testing.T) {
	dispatchTx := &administratorTestTx{}
	evidenceTx := &administratorTestTx{commitErr: errors.New("accepted evidence ack lost")}
	dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	evidenceTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	mail := &passwordResetMailerStub{}
	repository := &passwordResetRepositoryStub{}
	transactor := &passwordResetTransactorStub{
		transactions: []pgx.Tx{dispatchTx, evidenceTx}, beginErrs: map[int]error{},
	}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions: transactor,
		Repository:   repository, Store: &PasswordResetStore{}, Mailer: mail,
		IDs: &passwordResetIDStub{}, Clock: passwordResetClockStub{},
		Audit: &passwordResetAuditStub{}, SMTPOperationBudget: time.Second,
	}}
	target := passwordResetTarget(RoleMember)
	_, err := coordinator.dispatchPasswordReset(context.Background(), passwordResetPrepared{
		target: target, secret: passwordResetSecret,
		pending:   TemporaryCredential{ID: "credential-one"},
		operation: PasswordResetOperation{ID: "operation-one", UserID: target.ID},
	})
	if platform.PublicCode(err) != platform.CodeIdentityOperationUncertain {
		t.Fatalf("accepted commit-unknown error = %#v", err)
	}
	if len(mail.messages) != 1 || repository.writeCalls() != 0 {
		t.Fatalf("mail=%d direct capability writes=%d", len(mail.messages), repository.writeCalls())
	}
	if transactor.next != 2 {
		t.Fatalf("transaction count = %d, request path must not depend on finite cleanup", transactor.next)
	}
}

func TestPasswordResetDeliveryEvidenceIgnoresParentCancellation(t *testing.T) {
	tests := []struct {
		name            string
		mailErr         error
		wantCode        platform.Code
		wantState       string
		wantUnknown     bool
		wantAccelerated bool
	}{
		{
			name:            "accepted evidence",
			wantState:       "delivery_state='accepted'",
			wantAccelerated: true,
		},
		{
			name:        "unknown evidence",
			mailErr:     errors.New("smtp acknowledgement unavailable"),
			wantCode:    platform.CodeIdentityOperationUncertain,
			wantState:   "state='commit_unknown'",
			wantUnknown: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dispatchTx := &administratorTestTx{}
			evidenceTx := &passwordResetContextAwareTx{}
			dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			evidenceTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
			transactor := &passwordResetContextAwareTransactor{
				transactions: []pgx.Tx{dispatchTx, evidenceTx},
			}
			repository := &passwordResetRepositoryStub{}
			auditStore := &passwordResetAuditStub{}
			requestContext, cancelRequest := context.WithCancel(context.WithValue(
				context.Background(),
				passwordResetEvidenceContextKey{},
				"request-value",
			))
			mail := &passwordResetMailerStub{
				err: test.mailErr,
				send: func(mailContext context.Context) {
					if _, ok := mailContext.Deadline(); !ok {
						t.Fatal("mailer context has no deadline")
					}
					cancelRequest()
				},
			}
			coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
				Transactions:        transactor,
				Repository:          repository,
				Store:               &PasswordResetStore{},
				Mailer:              mail,
				IDs:                 &passwordResetIDStub{},
				Clock:               passwordResetClockStub{},
				Audit:               auditStore,
				SMTPOperationBudget: time.Second,
			}}
			target := passwordResetTarget(RoleMember)

			result, err := coordinator.dispatchPasswordReset(requestContext, passwordResetPrepared{
				target:  target,
				secret:  passwordResetSecret,
				pending: TemporaryCredential{ID: "credential-one"},
				operation: PasswordResetOperation{
					ID: "operation-one", UserID: target.ID,
				},
				result: PasswordResetResult{UserID: target.ID},
			})

			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("accepted dispatch error = %v", err)
				}
				if result.DeliveryStatus != passwordResetDeliveryAccepted {
					t.Fatalf("accepted result = %#v", result)
				}
			} else if got := platform.PublicCode(err); got != test.wantCode {
				t.Fatalf("dispatch code = %q, want %q, err=%v", got, test.wantCode, err)
			}
			if len(transactor.observations) != 2 {
				t.Fatalf("begin observations = %d, want 2", len(transactor.observations))
			}
			assertPasswordResetEvidenceContext(t, transactor.observations[1])
			if len(evidenceTx.observations) == 0 {
				t.Fatal("evidence transaction did not execute")
			}
			for _, observation := range evidenceTx.observations {
				assertPasswordResetEvidenceContext(t, observation)
			}
			if evidenceTx.commitCalls != 1 {
				t.Fatalf("evidence commits = %d, want 1", evidenceTx.commitCalls)
			}
			if len(mail.messages) != 1 || repository.writeCalls() != 0 {
				t.Fatalf(
					"mail=%d credential/session writes=%d",
					len(mail.messages), repository.writeCalls(),
				)
			}
			if len(auditStore.recorded) != 0 {
				t.Fatalf("terminal audit writes = %d, want 0", len(auditStore.recorded))
			}
			statements := strings.Join(evidenceTx.statements, "\n")
			if !strings.Contains(statements, test.wantState) {
				t.Fatalf("evidence statements = %q", statements)
			}
			if accelerated := strings.Contains(
				statements,
				"password_reset_recovery_jobs",
			); accelerated != test.wantAccelerated {
				t.Fatalf(
					"recovery job acceleration = %t, want %t; statements = %q",
					accelerated,
					test.wantAccelerated,
					statements,
				)
			}
			if test.wantUnknown &&
				(!strings.Contains(statements, "delivery_state='unknown'") ||
					strings.Contains(statements, "delivery_state='rejected'") ||
					strings.Contains(statements, "recovery_invalidated")) {
				t.Fatalf("unknown evidence statements = %q", statements)
			}
		})
	}
}

func TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow(t *testing.T) {
	const protectionWindow = 10 * time.Minute
	tests := []struct {
		name         string
		mailErr      error
		reconcileAt  time.Time
		wantSettled  bool
		wantAccepted bool
	}{
		{
			name:        "unknown before protection boundary",
			mailErr:     errors.New("smtp acknowledgement unavailable"),
			reconcileAt: passwordResetNow.Add(protectionWindow - time.Microsecond),
		},
		{
			name:        "unknown at protection boundary",
			mailErr:     errors.New("smtp acknowledgement unavailable"),
			reconcileAt: passwordResetNow.Add(protectionWindow),
			wantSettled: true,
		},
		{
			name:        "unknown after protection boundary",
			mailErr:     errors.New("smtp acknowledgement unavailable"),
			reconcileAt: passwordResetNow.Add(protectionWindow + time.Microsecond),
			wantSettled: true,
		},
		{
			name:         "accepted immediately",
			reconcileAt:  passwordResetNow,
			wantSettled:  true,
			wantAccepted: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newPasswordResetCrossCardHarness(
				t,
				passwordResetNow.Add(protectionWindow),
				test.mailErr,
			)
			harness.dispatch(t, test.wantAccepted)
			harness.clock.Set(test.reconcileAt)

			if err := harness.worker.Reconcile(context.Background()); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}

			snapshot := harness.state.snapshot()
			if !test.wantSettled {
				if got := snapshot.workerCalls; !equalPasswordResetStrings(got, []string{"scan"}) {
					t.Fatalf("worker calls = %v, want scan with no eligible candidate", got)
				}
				assertPasswordResetCrossCardUnresolved(t, snapshot)
				assertPasswordResetCrossCardConsumersClosed(t, harness)
				return
			}
			if got := snapshot.workerCalls; !equalPasswordResetStrings(
				got,
				[]string{"scan", "claim", "renew", "operator"},
			) {
				t.Fatalf("worker calls = %v", got)
			}
			assertPasswordResetCrossCardPermit(t, harness.permit)
			if test.wantAccepted {
				assertPasswordResetCrossCardAccepted(t, snapshot)
				return
			}
			assertPasswordResetCrossCardInvalidated(t, snapshot)
			assertPasswordResetCrossCardOldLoginWorks(t, harness)
		})
	}
}

func TestPasswordResetAcceptedDeliveryAcceleratesRecoveryJob(t *testing.T) {
	initialNextAttemptAt := passwordResetNow.Add(10 * time.Minute)
	dispatchTx := &administratorTestTx{}
	evidenceTx := &passwordResetRecoveryWindowTx{
		nextAttemptAt: initialNextAttemptAt,
	}
	dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	evidenceTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	repository := &passwordResetRepositoryStub{}
	mail := &passwordResetMailerStub{}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions: &passwordResetTransactorStub{
			transactions: []pgx.Tx{dispatchTx, evidenceTx},
			beginErrs:    map[int]error{},
		},
		Repository:          repository,
		Store:               &PasswordResetStore{},
		Mailer:              mail,
		IDs:                 &passwordResetIDStub{},
		Clock:               passwordResetClockStub{},
		Audit:               &passwordResetAuditStub{},
		SMTPOperationBudget: time.Second,
	}}
	target := passwordResetTarget(RoleMember)

	result, err := coordinator.dispatchPasswordReset(
		context.Background(),
		passwordResetPrepared{
			target:  target,
			secret:  passwordResetSecret,
			pending: TemporaryCredential{ID: "credential-one"},
			operation: PasswordResetOperation{
				ID: "operation-one", UserID: target.ID,
			},
			result: PasswordResetResult{UserID: target.ID},
		},
	)

	if err != nil || result.DeliveryStatus != passwordResetDeliveryAccepted {
		t.Fatalf("dispatch result=%#v error=%v", result, err)
	}
	if evidenceTx.accelerationCalls != 1 {
		t.Fatalf(
			"accepted evidence acceleration calls = %d, want 1",
			evidenceTx.accelerationCalls,
		)
	}
	if !evidenceTx.nextAttemptAt.Equal(passwordResetNow) {
		t.Fatalf(
			"next_attempt_at = %s, want immediate settlement at %s",
			evidenceTx.nextAttemptAt,
			passwordResetNow,
		)
	}
	if evidenceTx.nextAttemptAt.After(passwordResetNow) {
		t.Fatalf("accepted recovery job is not immediately claimable")
	}
	if len(mail.messages) != 1 || repository.writeCalls() != 0 {
		t.Fatalf(
			"mail=%d credential/session writes=%d",
			len(mail.messages),
			repository.writeCalls(),
		)
	}
}

type passwordResetCrossCardHarness struct {
	state       *passwordResetCrossCardState
	clock       *passwordResetMutableClock
	permit      *PasswordResetConnectionPermit
	worker      *PasswordResetRecoveryWorker
	coordinator *PasswordResetCoordinator
	gate        *passwordResetCrossCardGate
	mail        *passwordResetMailerStub
	repository  *passwordResetRepositoryStub
	transactor  *passwordResetCrossCardTransactor
}

func newPasswordResetCrossCardHarness(
	t *testing.T,
	nextAttemptAt time.Time,
	mailErr error,
) *passwordResetCrossCardHarness {
	t.Helper()
	state := newPasswordResetCrossCardState(nextAttemptAt)
	clock := &passwordResetMutableClock{now: passwordResetNow}
	store := &PasswordResetStore{}
	transactor := &passwordResetCrossCardTransactor{state: state}
	mail := &passwordResetMailerStub{err: mailErr}
	repository := &passwordResetRepositoryStub{}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions:        transactor,
		Repository:          repository,
		Store:               store,
		Mailer:              mail,
		IDs:                 &passwordResetIDStub{},
		Clock:               clock,
		Audit:               &passwordResetAuditStub{},
		SMTPOperationBudget: time.Second,
	}}
	permit, err := NewPasswordResetConnectionPermit(1)
	if err != nil {
		t.Fatalf("NewPasswordResetConnectionPermit() error = %v", err)
	}
	operator := newPasswordResetOperatorForTest(
		permit,
		func(_ context.Context, lease PasswordResetRecoveryLease) error {
			if permit.Stats().Current != 1 {
				return errors.New("operator did not inherit worker connection permit")
			}
			return state.settle(lease, clock.Now())
		},
	)
	dependencies := passwordResetWorkerTestDependencies(permit, operator)
	dependencies.Clock = clock
	pool := &passwordResetCrossCardPool{state: state}
	backend := passwordResetPostgresBackend{pool: pool, store: store, clock: clock}
	worker, err := newPasswordResetWorkerForTest(
		dependencies,
		passwordResetWorkerTestConfig(),
		backend,
	)
	if err != nil {
		t.Fatalf("newPasswordResetWorkerForTest() error = %v", err)
	}
	return &passwordResetCrossCardHarness{
		state: state, clock: clock, permit: permit, worker: worker,
		coordinator: coordinator, gate: &passwordResetCrossCardGate{state: state},
		mail: mail, repository: repository, transactor: transactor,
	}
}

func (harness *passwordResetCrossCardHarness) dispatch(
	t *testing.T,
	wantAccepted bool,
) {
	t.Helper()
	email := "alice@example.invalid"
	target := loginTestUser()
	target.Email = &email
	result, err := harness.coordinator.dispatchPasswordReset(
		context.Background(),
		passwordResetPrepared{
			target:  target,
			secret:  passwordResetSecret,
			pending: TemporaryCredential{ID: harness.state.credentialID},
			operation: PasswordResetOperation{
				ID: harness.state.operationID, UserID: harness.state.userID,
			},
			result: PasswordResetResult{UserID: target.ID},
		},
	)
	if wantAccepted {
		if err != nil || result.DeliveryStatus != passwordResetDeliveryAccepted {
			t.Fatalf("accepted dispatch result=%#v error=%v", result, err)
		}
	} else if got := platform.PublicCode(err); got != platform.CodeIdentityOperationUncertain {
		t.Fatalf("unknown dispatch code = %q, want operation uncertain; err=%v", got, err)
	}
	if len(harness.mail.messages) != 1 ||
		harness.repository.writeCalls() != 0 ||
		harness.transactor.begins != 2 {
		t.Fatalf(
			"dispatch mail/repository writes/transactions = %d/%d/%d",
			len(harness.mail.messages),
			harness.repository.writeCalls(),
			harness.transactor.begins,
		)
	}
	snapshot := harness.state.snapshot()
	if wantAccepted {
		if snapshot.operationState != PasswordResetOperationRecoveryRequired ||
			snapshot.deliveryState != PasswordResetDeliveryAccepted ||
			snapshot.accelerationCalls != 1 ||
			!snapshot.nextAttemptAt.Equal(passwordResetNow) {
			t.Fatalf("accepted evidence state = %+v", snapshot)
		}
		return
	}
	if snapshot.operationState != PasswordResetOperationCommitUnknown ||
		snapshot.deliveryState != PasswordResetDeliveryUnknown {
		t.Fatalf("unknown evidence state = %+v", snapshot)
	}
}

type passwordResetCrossCardState struct {
	mu sync.Mutex

	userID       string
	operationID  string
	credentialID string

	operationState    PasswordResetOperationState
	deliveryState     PasswordResetDeliveryState
	settlementState   PasswordResetSettlementState
	credentialState   TemporaryCredentialState
	idempotencyStatus string
	jobStatus         PasswordResetRecoveryJobStatus
	nextAttemptAt     time.Time
	attemptCount      int64
	leaseOwner        string
	leaseExpiresAt    time.Time
	auditCount        int
	activeSessions    int
	oldCapability     bool
	accelerationCalls int
	workerCalls       []string
}

type passwordResetCrossCardSnapshot struct {
	operationState    PasswordResetOperationState
	deliveryState     PasswordResetDeliveryState
	settlementState   PasswordResetSettlementState
	credentialState   TemporaryCredentialState
	idempotencyStatus string
	jobStatus         PasswordResetRecoveryJobStatus
	nextAttemptAt     time.Time
	attemptCount      int64
	auditCount        int
	activeSessions    int
	oldCapability     bool
	accelerationCalls int
	workerCalls       []string
}

func newPasswordResetCrossCardState(
	nextAttemptAt time.Time,
) *passwordResetCrossCardState {
	return &passwordResetCrossCardState{
		userID:            "00000000-0000-4000-8000-000000000001",
		operationID:       "password-reset-operation-cross-card",
		credentialID:      "password-reset-credential-cross-card",
		operationState:    PasswordResetOperationPrepared,
		deliveryState:     PasswordResetDeliveryNotStarted,
		settlementState:   PasswordResetSettlementPending,
		credentialState:   TemporaryCredentialPendingDelivery,
		idempotencyStatus: "processing",
		jobStatus:         PasswordResetRecoveryJobReady,
		nextAttemptAt:     nextAttemptAt,
		activeSessions:    2,
		oldCapability:     true,
	}
}

func (state *passwordResetCrossCardState) snapshot() passwordResetCrossCardSnapshot {
	state.mu.Lock()
	defer state.mu.Unlock()
	return passwordResetCrossCardSnapshot{
		operationState:    state.operationState,
		deliveryState:     state.deliveryState,
		settlementState:   state.settlementState,
		credentialState:   state.credentialState,
		idempotencyStatus: state.idempotencyStatus,
		jobStatus:         state.jobStatus,
		nextAttemptAt:     state.nextAttemptAt,
		attemptCount:      state.attemptCount,
		auditCount:        state.auditCount,
		activeSessions:    state.activeSessions,
		oldCapability:     state.oldCapability,
		accelerationCalls: state.accelerationCalls,
		workerCalls:       append([]string(nil), state.workerCalls...),
	}
}

func (state *passwordResetCrossCardState) recordDispatch() {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.operationState = PasswordResetOperationDispatching
	state.deliveryState = PasswordResetDeliveryDispatching
}

func (state *passwordResetCrossCardState) recordEvidence(accepted bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if accepted {
		state.operationState = PasswordResetOperationRecoveryRequired
		state.deliveryState = PasswordResetDeliveryAccepted
		return
	}
	state.operationState = PasswordResetOperationCommitUnknown
	state.deliveryState = PasswordResetDeliveryUnknown
}

func (state *passwordResetCrossCardState) recordRejectedOperation() {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.operationState = PasswordResetOperationFailed
	state.deliveryState = PasswordResetDeliveryRejected
	state.settlementState = PasswordResetSettlementInvalidated
}

func (state *passwordResetCrossCardState) invalidateRejectedCredential() {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.credentialState = TemporaryCredentialRecoveryInvalidated
}

func (state *passwordResetCrossCardState) completeRejectedJob() {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.jobStatus = PasswordResetRecoveryJobSucceeded
}

func (state *passwordResetCrossCardState) accelerate(nextAttemptAt time.Time) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.nextAttemptAt = nextAttemptAt
	state.accelerationCalls++
}

func (state *passwordResetCrossCardState) scan(
	now time.Time,
) []PasswordResetRecoveryCandidate {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.workerCalls = append(state.workerCalls, "scan")
	due := (state.jobStatus == PasswordResetRecoveryJobReady ||
		state.jobStatus == PasswordResetRecoveryJobRetryWait) &&
		!state.nextAttemptAt.After(now)
	if state.jobStatus == PasswordResetRecoveryJobLeased {
		due = !state.leaseExpiresAt.After(now)
	}
	if !due {
		return nil
	}
	return []PasswordResetRecoveryCandidate{{
		OperationID: state.operationID,
		UserID:      state.userID,
	}}
}

func (state *passwordResetCrossCardState) claim(
	owner string,
	now time.Time,
	expiresAt time.Time,
) (int64, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.workerCalls = append(state.workerCalls, "claim")
	due := (state.jobStatus == PasswordResetRecoveryJobReady ||
		state.jobStatus == PasswordResetRecoveryJobRetryWait) &&
		!state.nextAttemptAt.After(now)
	if state.jobStatus == PasswordResetRecoveryJobLeased {
		due = !state.leaseExpiresAt.After(now)
	}
	if !due {
		return 0, false
	}
	state.jobStatus = PasswordResetRecoveryJobLeased
	state.attemptCount++
	state.leaseOwner = owner
	state.leaseExpiresAt = expiresAt
	return state.attemptCount, true
}

func (state *passwordResetCrossCardState) renew(
	owner string,
	attempt int64,
	now time.Time,
	expiresAt time.Time,
) bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.workerCalls = append(state.workerCalls, "renew")
	if state.jobStatus != PasswordResetRecoveryJobLeased ||
		state.leaseOwner != owner ||
		state.attemptCount != attempt ||
		!state.leaseExpiresAt.After(now) {
		return false
	}
	state.leaseExpiresAt = expiresAt
	return true
}

func (state *passwordResetCrossCardState) settle(
	lease PasswordResetRecoveryLease,
	now time.Time,
) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.workerCalls = append(state.workerCalls, "operator")
	if state.jobStatus != PasswordResetRecoveryJobLeased ||
		state.leaseOwner != lease.Owner ||
		state.attemptCount != lease.AttemptCount ||
		!state.leaseExpiresAt.After(now) {
		return ErrPasswordResetStateConflict
	}
	if state.deliveryState == PasswordResetDeliveryAccepted {
		state.operationState = PasswordResetOperationSucceeded
		state.settlementState = PasswordResetSettlementTemporaryActive
		state.credentialState = TemporaryCredentialActive
		state.idempotencyStatus = "succeeded"
		state.activeSessions = 0
		state.oldCapability = false
	} else {
		state.operationState = PasswordResetOperationInvalidated
		state.settlementState = PasswordResetSettlementInvalidated
		state.credentialState = TemporaryCredentialRecoveryInvalidated
		state.idempotencyStatus = "failed"
	}
	state.jobStatus = PasswordResetRecoveryJobSucceeded
	state.leaseOwner = ""
	state.leaseExpiresAt = time.Time{}
	state.auditCount++
	return nil
}

func (state *passwordResetCrossCardState) gateClosed() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	switch state.operationState {
	case PasswordResetOperationSucceeded,
		PasswordResetOperationInvalidated,
		PasswordResetOperationFailed:
		return false
	default:
		return true
	}
}

type passwordResetCrossCardTransactor struct {
	state  *passwordResetCrossCardState
	begins int
}

func (transactor *passwordResetCrossCardTransactor) Begin(
	context.Context,
) (pgx.Tx, error) {
	transactor.begins++
	return &passwordResetCrossCardTx{state: transactor.state}, nil
}

type passwordResetCrossCardPool struct {
	state *passwordResetCrossCardState
}

func (pool *passwordResetCrossCardPool) Query(
	_ context.Context,
	statement string,
	arguments ...any,
) (pgx.Rows, error) {
	if compactPasswordResetSQL(statement) !=
		compactPasswordResetSQL(passwordResetRecoveryScanSQL) ||
		!strings.Contains(statement, "next_attempt_at<=$1") {
		return nil, errors.New("worker scan lost recovery time predicate")
	}
	if len(arguments) != 2 {
		return nil, errors.New("worker scan arguments are invalid")
	}
	now, ok := arguments[0].(time.Time)
	if !ok {
		return nil, errors.New("worker scan time is invalid")
	}
	candidates := pool.state.scan(now)
	values := make([][]any, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, []any{candidate.OperationID, candidate.UserID})
	}
	return &passwordResetRowsHarness{values: values}, nil
}

func (pool *passwordResetCrossCardPool) BeginTx(
	context.Context,
	pgx.TxOptions,
) (pgx.Tx, error) {
	return &passwordResetCrossCardTx{state: pool.state}, nil
}

type passwordResetCrossCardTx struct {
	pgx.Tx
	state *passwordResetCrossCardState
}

func (tx *passwordResetCrossCardTx) QueryRow(
	_ context.Context,
	statement string,
	arguments ...any,
) pgx.Row {
	switch {
	case statement == lockPasswordResetRequestUserSQL ||
		statement == passwordResetRecoveryLockUserSQL:
		return passwordResetStoreRow{values: []any{tx.state.userID}}
	case statement == passwordResetRecoveryClaimSQL:
		if !strings.Contains(statement, "next_attempt_at<=$4") ||
			len(arguments) != 5 {
			return passwordResetStoreRow{
				err: errors.New("worker claim lost recovery time predicate"),
			}
		}
		owner, ownerOK := arguments[2].(string)
		now, nowOK := arguments[3].(time.Time)
		expiresAt, expiresOK := arguments[4].(time.Time)
		if !ownerOK || !nowOK || !expiresOK {
			return passwordResetStoreRow{
				err: errors.New("worker claim arguments are invalid"),
			}
		}
		attempt, claimed := tx.state.claim(owner, now, expiresAt)
		if !claimed {
			return passwordResetStoreRow{err: pgx.ErrNoRows}
		}
		return passwordResetStoreRow{values: []any{attempt}}
	default:
		return passwordResetStoreRow{
			err: fmt.Errorf(
				"unexpected cross-card query: %s",
				compactPasswordResetSQL(statement),
			),
		}
	}
}

func (tx *passwordResetCrossCardTx) Exec(
	_ context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	switch statement {
	case claimPasswordResetDispatchSQL:
		tx.state.recordDispatch()
	case recordPasswordResetAcceptedSQL:
		tx.state.recordEvidence(true)
	case recordPasswordResetUnknownSQL:
		tx.state.recordEvidence(false)
	case recordPasswordResetRejectedSQL:
		tx.state.recordRejectedOperation()
	case invalidateRejectedPasswordResetCredentialSQL:
		tx.state.invalidateRejectedCredential()
	case completeRejectedPasswordResetJobSQL:
		tx.state.completeRejectedJob()
	case acceleratePasswordResetRecoveryJobSQL:
		if len(arguments) != 2 {
			return pgconn.CommandTag{}, errors.New("invalid recovery acceleration")
		}
		nextAttemptAt, ok := arguments[0].(time.Time)
		if !ok {
			return pgconn.CommandTag{}, errors.New("invalid recovery acceleration time")
		}
		tx.state.accelerate(nextAttemptAt)
	case passwordResetRecoveryRenewSQL:
		if !strings.Contains(statement, "lease_expires_at>$2") ||
			len(arguments) != 5 {
			return pgconn.CommandTag{}, errors.New("worker renew lost lease predicate")
		}
		expiresAt, expiresOK := arguments[0].(time.Time)
		now, nowOK := arguments[1].(time.Time)
		owner, ownerOK := arguments[3].(string)
		attempt, attemptOK := arguments[4].(int64)
		if !expiresOK || !nowOK || !ownerOK || !attemptOK {
			return pgconn.CommandTag{}, errors.New("worker renew arguments are invalid")
		}
		if !tx.state.renew(owner, attempt, now, expiresAt) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
	default:
		return pgconn.CommandTag{}, fmt.Errorf(
			"unexpected cross-card exec: %s",
			compactPasswordResetSQL(statement),
		)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (*passwordResetCrossCardTx) Commit(context.Context) error   { return nil }
func (*passwordResetCrossCardTx) Rollback(context.Context) error { return nil }

type passwordResetCrossCardGate struct {
	state *passwordResetCrossCardState
	mu    sync.Mutex
	calls int
}

func (gate *passwordResetCrossCardGate) CheckLockedUser(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) error {
	gate.mu.Lock()
	gate.calls++
	gate.mu.Unlock()
	if userID != gate.state.userID {
		return errors.New("password reset Gate user mismatch")
	}
	if gate.state.gateClosed() {
		return session.ErrPasswordResetGateClosed
	}
	return nil
}

func (gate *passwordResetCrossCardGate) callCount() int {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	return gate.calls
}

type passwordResetSessionBoundary struct {
	userID string
	tx     pgx.Tx
}

func (boundary *passwordResetSessionBoundary) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	return passwordResetStoreRow{values: []any{boundary.userID}}
}

func (boundary *passwordResetSessionBoundary) Begin(
	context.Context,
) (pgx.Tx, error) {
	return boundary.tx, nil
}

func assertPasswordResetCrossCardUnresolved(
	t *testing.T,
	snapshot passwordResetCrossCardSnapshot,
) {
	t.Helper()
	if snapshot.operationState != PasswordResetOperationCommitUnknown ||
		snapshot.deliveryState != PasswordResetDeliveryUnknown ||
		snapshot.settlementState != PasswordResetSettlementPending ||
		snapshot.credentialState != TemporaryCredentialPendingDelivery ||
		snapshot.idempotencyStatus != "processing" ||
		snapshot.jobStatus != PasswordResetRecoveryJobReady ||
		snapshot.attemptCount != 0 ||
		snapshot.auditCount != 0 ||
		!snapshot.oldCapability ||
		snapshot.accelerationCalls != 0 {
		t.Fatalf("unresolved protection state = %+v", snapshot)
	}
}

func assertPasswordResetCrossCardInvalidated(
	t *testing.T,
	snapshot passwordResetCrossCardSnapshot,
) {
	t.Helper()
	if snapshot.operationState != PasswordResetOperationInvalidated ||
		snapshot.deliveryState != PasswordResetDeliveryUnknown ||
		snapshot.settlementState != PasswordResetSettlementInvalidated ||
		snapshot.credentialState != TemporaryCredentialRecoveryInvalidated ||
		snapshot.idempotencyStatus != "failed" ||
		snapshot.jobStatus != PasswordResetRecoveryJobSucceeded ||
		snapshot.attemptCount != 1 ||
		snapshot.auditCount != 1 ||
		!snapshot.oldCapability ||
		snapshot.activeSessions != 2 ||
		snapshot.accelerationCalls != 0 {
		t.Fatalf("invalidated recovery state = %+v", snapshot)
	}
}

func assertPasswordResetCrossCardAccepted(
	t *testing.T,
	snapshot passwordResetCrossCardSnapshot,
) {
	t.Helper()
	if snapshot.operationState != PasswordResetOperationSucceeded ||
		snapshot.deliveryState != PasswordResetDeliveryAccepted ||
		snapshot.settlementState != PasswordResetSettlementTemporaryActive ||
		snapshot.credentialState != TemporaryCredentialActive ||
		snapshot.idempotencyStatus != "succeeded" ||
		snapshot.jobStatus != PasswordResetRecoveryJobSucceeded ||
		snapshot.attemptCount != 1 ||
		snapshot.auditCount != 1 ||
		snapshot.oldCapability ||
		snapshot.activeSessions != 0 ||
		snapshot.accelerationCalls != 1 ||
		!snapshot.nextAttemptAt.Equal(passwordResetNow) {
		t.Fatalf("accepted recovery state = %+v", snapshot)
	}
}

func assertPasswordResetCrossCardPermit(
	t *testing.T,
	permit *PasswordResetConnectionPermit,
) {
	t.Helper()
	stats := permit.Stats()
	if stats.Acquired != 4 || stats.Released != 4 ||
		stats.Current != 0 || stats.Peak != 1 {
		t.Fatalf("worker/operator permit stats = %+v", stats)
	}
}

func assertPasswordResetCrossCardConsumersClosed(
	t *testing.T,
	harness *passwordResetCrossCardHarness,
) {
	t.Helper()
	before := harness.gate.callCount()

	login := newLoginFixture(t)
	login.repository.user.ID = harness.state.userID
	login.service.loginDependencies.PasswordResetGate = harness.gate
	result, err := login.service.Login(
		login.context,
		login.request(LoginRequest{
			Username: "Alice", Password: "Password123",
			SourceFingerprint: "source-digest",
		}),
	)
	if platform.PublicCode(err) != platform.CodeAuthLoginFailed ||
		result != (LoginResult{}) ||
		len(login.hasher.verifiedDigests) != 0 ||
		login.sessions.calls != 0 {
		t.Fatalf(
			"Login Gate result/error/verify/session = %#v/%v/%d/%d",
			result,
			err,
			len(login.hasher.verifiedDigests),
			login.sessions.calls,
		)
	}

	passwordChangeTx := &bootstrapTestTx{}
	passwordChangeRepository := &onboardingTestRepository{
		user: onboardingTestUser(CredentialModeBootstrap),
	}
	passwordChangeRepository.user.ID = harness.state.userID
	passwords := &onboardingTestPasswords{matched: true}
	passwordChangeSessions := &onboardingTestSessions{}
	passwordChange := newOnboardingTestService(
		t,
		passwordChangeTx,
		passwordChangeRepository,
		passwords,
		passwordChangeSessions,
	)
	passwordChange.passwordChangeDependencies.PasswordResetGate = harness.gate
	changeResult, changeErr := passwordChange.ChangePassword(
		context.Background(),
		passwordChangeTestRequest(
			passwordChangeRepository.user,
			session.CredentialBootstrap,
		),
	)
	if platform.PublicCode(changeErr) != platform.CodeAuthSessionRequired ||
		changeResult != (PasswordChangeResult{}) ||
		passwords.verifyCalls != 0 ||
		passwordChangeRepository.updatedDigest != "" ||
		passwordChangeSessions.rotateCalls != 0 {
		t.Fatalf(
			"PasswordChange Gate result/error/verify/digest/rotate = %#v/%v/%d/%q/%d",
			changeResult,
			changeErr,
			passwords.verifyCalls,
			passwordChangeRepository.updatedDigest,
			passwordChangeSessions.rotateCalls,
		)
	}

	sessionTx := &administratorTestTx{}
	sessionTx.row = passwordResetStoreRow{values: []any{harness.state.userID}}
	sessionStore, storeErr := session.NewStore(session.StoreDependencies{
		Transactions: &passwordResetSessionBoundary{
			userID: harness.state.userID,
			tx:     sessionTx,
		},
		PasswordResetGate: harness.gate,
	})
	if storeErr != nil {
		t.Fatalf("session.NewStore() error = %v", storeErr)
	}
	principal, resolveErr := sessionStore.ResolvePrincipal(
		context.Background(),
		session.ResolveInput{
			Token: "existing-session-token",
			Now:   passwordResetNow,
		},
	)
	if !errors.Is(resolveErr, session.ErrSessionRequired) ||
		principal != (session.Principal{}) ||
		sessionTx.commitCalls != 0 ||
		sessionTx.rollbackCalls != 1 {
		t.Fatalf(
			"Session Resolve Gate principal/error/commit/rollback = %#v/%v/%d/%d",
			principal,
			resolveErr,
			sessionTx.commitCalls,
			sessionTx.rollbackCalls,
		)
	}
	if got := harness.gate.callCount() - before; got != 3 {
		t.Fatalf("shared Gate consumer calls = %d, want 3", got)
	}
	assertPasswordResetCrossCardUnresolved(t, harness.state.snapshot())
	stats := harness.permit.Stats()
	if stats.Acquired != 1 || stats.Released != 1 ||
		stats.Current != 0 || stats.Peak != 1 {
		t.Fatalf("before-boundary scan permit stats = %+v", stats)
	}
}

func assertPasswordResetCrossCardOldLoginWorks(
	t *testing.T,
	harness *passwordResetCrossCardHarness,
) {
	t.Helper()
	login := newLoginFixture(t)
	login.repository.user.ID = harness.state.userID
	login.service.loginDependencies.PasswordResetGate = harness.gate
	result, err := login.service.Login(
		login.context,
		login.request(LoginRequest{
			Username: "Alice", Password: "Password123",
			SourceFingerprint: "source-digest",
		}),
	)
	if err != nil || result.SessionToken == "" ||
		len(login.hasher.verifiedDigests) != 1 ||
		login.sessions.calls != 1 {
		t.Fatalf(
			"old capability Login result/error/verify/session = %#v/%v/%d/%d",
			result,
			err,
			len(login.hasher.verifiedDigests),
			login.sessions.calls,
		)
	}
}

func assertPasswordResetEvidenceContext(
	t *testing.T,
	observation passwordResetContextObservation,
) {
	t.Helper()
	if observation.err != nil {
		t.Fatalf("evidence context was canceled: %v", observation.err)
	}
	if !observation.hasDeadline {
		t.Fatal("evidence context has no deadline")
	}
	if observation.value != "request-value" {
		t.Fatalf("evidence context value = %#v, want request-value", observation.value)
	}
}

func TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability(t *testing.T) {
	dispatchTx := &administratorTestTx{}
	unknownTx := &passwordResetRecordingTx{}
	dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	unknownTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	mail := &passwordResetMailerStub{err: errors.New("smtp acknowledgement unavailable")}
	repository := &passwordResetRepositoryStub{}
	auditStore := &passwordResetAuditStub{}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions: &passwordResetTransactorStub{
			transactions: []pgx.Tx{dispatchTx, unknownTx}, beginErrs: map[int]error{},
		},
		Repository: repository, Store: &PasswordResetStore{}, Mailer: mail,
		IDs: &passwordResetIDStub{}, Clock: passwordResetClockStub{},
		Audit: auditStore, SMTPOperationBudget: time.Second,
	}}
	target := passwordResetTarget(RoleMember)
	_, err := coordinator.dispatchPasswordReset(context.Background(), passwordResetPrepared{
		target: target, secret: passwordResetSecret,
		pending: TemporaryCredential{ID: "credential-one"},
		operation: PasswordResetOperation{
			ID: "operation-one", UserID: target.ID,
			ScopeHash: []byte("scope"), KeyLocator: []byte("locator"),
		},
		claim: passwordResetRecord(idempotency.StatusProcessing),
	})
	if platform.PublicCode(err) != platform.CodeIdentityOperationUncertain {
		t.Fatalf("unknown delivery error = %#v cause=%v", err, errors.Unwrap(err))
	}
	if repository.writeCalls() != 0 {
		t.Fatalf("unknown delivery changed old credential/session capability: %d writes", repository.writeCalls())
	}
	if len(auditStore.recorded) != 0 {
		t.Fatalf("unknown delivery wrote terminal audit evidence: %d", len(auditStore.recorded))
	}
	statements := strings.Join(unknownTx.statements, "\n")
	if !strings.Contains(statements, "state='commit_unknown'") ||
		!strings.Contains(statements, "delivery_state='unknown'") {
		t.Fatalf("unknown delivery statements = %q", statements)
	}
	if strings.Contains(statements, "password_reset_recovery_jobs") {
		t.Fatalf("unknown delivery accelerated recovery job: %q", statements)
	}
	if strings.Contains(statements, "delivery_state='rejected'") ||
		strings.Contains(statements, "recovery_invalidated") {
		t.Fatalf("unknown delivery was persisted as terminal rejection: %q", statements)
	}
}

func TestPasswordResetUnknownDeliveryCommitUnknownDoesNotUseFiniteCleanup(t *testing.T) {
	dispatchTx := &administratorTestTx{}
	unknownTx := &passwordResetRecordingTx{}
	unknownTx.commitErr = errors.New("unknown evidence acknowledgement lost")
	dispatchTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	unknownTx.row = passwordResetStoreRow{values: []any{passwordResetTargetID}}
	repository := &passwordResetRepositoryStub{}
	transactor := &passwordResetTransactorStub{
		transactions: []pgx.Tx{dispatchTx, unknownTx}, beginErrs: map[int]error{},
	}
	coordinator := &PasswordResetCoordinator{dependencies: PasswordResetDependencies{
		Transactions: transactor,
		Repository:   repository,
		Store:        &PasswordResetStore{},
		Mailer: &passwordResetMailerStub{
			err: errors.New("smtp acknowledgement unavailable"),
		},
		IDs: &passwordResetIDStub{}, Clock: passwordResetClockStub{},
		Audit: &passwordResetAuditStub{}, SMTPOperationBudget: time.Second,
	}}
	target := passwordResetTarget(RoleMember)
	_, err := coordinator.dispatchPasswordReset(context.Background(), passwordResetPrepared{
		target: target, secret: passwordResetSecret,
		pending: TemporaryCredential{ID: "credential-one"},
		operation: PasswordResetOperation{
			ID: "operation-one", UserID: target.ID,
		},
	})
	if platform.PublicCode(err) != platform.CodeIdentityOperationUncertain {
		t.Fatalf("unknown evidence commit error = %#v", err)
	}
	if transactor.next != 2 {
		t.Fatalf("transaction count = %d, request path must not use finite cleanup", transactor.next)
	}
	if repository.writeCalls() != 0 {
		t.Fatalf("unknown delivery commit error changed capability: %d writes", repository.writeCalls())
	}
	statements := strings.Join(unknownTx.statements, "\n")
	if !strings.Contains(statements, "state='commit_unknown'") ||
		strings.Contains(statements, "delivery_state='rejected'") {
		t.Fatalf("unknown evidence statements = %q", statements)
	}
}

func TestPasswordResetReplayUsesOperationAuthority(t *testing.T) {
	target := passwordResetTarget(RoleMember)
	unresolved := passwordResetPrepared{
		target:    target,
		operation: PasswordResetOperation{State: PasswordResetOperationCommitUnknown},
	}
	if _, err := classifyPasswordResetReplay(unresolved); platform.PublicCode(err) != platform.CodeIdentityOperationUncertain {
		t.Fatalf("unresolved replay error = %v", err)
	}
	failed := unresolved
	failed.operation.State = PasswordResetOperationFailed
	if _, err := classifyPasswordResetReplay(failed); platform.PublicCode(err) != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("terminal failure replay error = %v", err)
	}
	invalidated := unresolved
	invalidated.operation.State = PasswordResetOperationInvalidated
	if _, err := classifyPasswordResetReplay(invalidated); platform.PublicCode(err) != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("terminal invalidated replay error = %v", err)
	}
	succeeded := unresolved
	succeeded.operation.State = PasswordResetOperationSucceeded
	succeeded.claim = passwordResetRecord(idempotency.StatusSucceeded)
	result, err := classifyPasswordResetReplay(succeeded)
	if err != nil || result.DeliveryStatus != passwordResetDeliveryAccepted ||
		result.UserID != target.ID {
		t.Fatalf("terminal success replay result=%#v error=%v", result, err)
	}
}

type passwordResetCandidateAuthorityStub struct {
	candidate PasswordResetCandidate
	result    PasswordResetCandidateResult
	err       error
	calls     int
}

func (stub *passwordResetCandidateAuthorityStub) ClassifyExistingCandidate(
	_ context.Context,
	_ idempotency.Transaction,
	_ string,
	_ idempotency.BeginRequest,
	_ func() (string, error),
) (PasswordResetCandidateResult, bool, error) {
	if stub.result.Action == PasswordResetCandidateExisting ||
		stub.result.Action == PasswordResetCandidateIntegrityHold ||
		stub.err != nil {
		return stub.result, true, stub.err
	}
	return PasswordResetCandidateResult{}, false, nil
}

func (stub *passwordResetCandidateAuthorityStub) BeginOrClassifyCandidate(
	_ context.Context,
	_ idempotency.Transaction,
	candidate PasswordResetCandidate,
) (PasswordResetCandidateResult, error) {
	stub.calls++
	stub.candidate = candidate
	if stub.result.Action != "" || stub.err != nil {
		return stub.result, stub.err
	}
	operation := candidate.Operation
	operation.UserID = candidate.Credential.UserID
	return PasswordResetCandidateResult{
		Action: PasswordResetCandidateCreated, Operation: operation,
		Claim: passwordResetRecord(idempotency.StatusProcessing),
	}, nil
}

type passwordResetFixtureAuthority struct {
	idempotency *passwordResetIdempotencyStub
}

func (authority *passwordResetFixtureAuthority) ClassifyExistingCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	_ string,
	request idempotency.BeginRequest,
	_ func() (string, error),
) (PasswordResetCandidateResult, bool, error) {
	result, found, err := authority.idempotency.ResolveExisting(ctx, tx, request)
	if err != nil || !found {
		return PasswordResetCandidateResult{}, found, err
	}
	operation := PasswordResetOperation{
		ID: "existing-operation", UserID: passwordResetTargetID,
		State: PasswordResetOperationCommitUnknown,
	}
	switch {
	case result.Action == idempotency.BeginReplay &&
		result.Record.Status == idempotency.StatusSucceeded:
		operation.State = PasswordResetOperationSucceeded
	case result.Action == idempotency.BeginReplay &&
		result.Record.Status == idempotency.StatusFailed:
		operation.State = PasswordResetOperationFailed
	}
	return PasswordResetCandidateResult{
		Action: PasswordResetCandidateExisting, Operation: operation,
		Claim: result.Record,
	}, true, nil
}

func (authority *passwordResetFixtureAuthority) BeginOrClassifyCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
) (PasswordResetCandidateResult, error) {
	result, err := authority.idempotency.Begin(ctx, tx, candidate.IdempotencyRequest)
	if err != nil {
		return PasswordResetCandidateResult{}, err
	}
	operation := candidate.Operation
	operation.UserID = candidate.Credential.UserID
	return PasswordResetCandidateResult{
		Action: PasswordResetCandidateCreated, Operation: operation,
		Claim: result.Record,
	}, nil
}

func TestPasswordResetRecoveryDispatch_OrdinaryOwner_RejectsBeforeDependencies(t *testing.T) {
	var ordinary *PasswordResetCoordinator
	prepared := passwordResetPrepared{operation: PasswordResetOperation{Recovery: &RecoveryResetMetadata{RunID: "synthetic-recovery"}}}
	if _, err := ordinary.dispatchPasswordReset(context.Background(), prepared); !errors.Is(err, errRecoveryResetOrdinaryDispatch) {
		t.Fatalf("ordinary dispatch admitted recovery owner: %v", err)
	}
}

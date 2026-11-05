package identity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

func TestCreateAdministratorRejectsNonInitialAdministrator(t *testing.T) {
	err := requireCurrentInitialAdministrator(
		session.Principal{UserID: "ordinary-admin"},
		User{
			ID: "ordinary-admin", Role: RoleAdmin,
			Status: UserStatusActive, CredentialMode: CredentialModePermanent,
		},
	)
	if code := platform.PublicCode(err); code != platform.CodeAuthForbidden {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeAuthForbidden)
	}
}

func TestCreateAdministratorReconfirmsCurrentInitialAdministratorBeforeIdempotency(t *testing.T) {
	repository := &administratorCreateRepositoryStub{
		authorizationUser: User{
			ID:             "initial-admin",
			Role:           RoleAdmin,
			IsInitialAdmin: false,
			Status:         UserStatusActive,
		},
	}
	idempotencyStore := &administratorIdempotencyStub{}
	mailerStub := &administratorMailerStub{}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{&administratorTestTx{}},
		repository,
		idempotencyStore,
		mailerStub,
		nil,
	)

	_, err := service.CreateAdministrator(
		context.Background(),
		administratorCreateTestRequest(),
	)
	if code := platform.PublicCode(err); code != platform.CodeAuthForbidden {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeAuthForbidden)
	}
	if repository.authorizationCalls != 1 {
		t.Fatalf("authorization reads = %d, want 1", repository.authorizationCalls)
	}
	if idempotencyStore.resolveCalls != 0 ||
		len(idempotencyStore.beginRequests) != 0 ||
		len(mailerStub.messages) != 0 ||
		repository.user.ID != "" ||
		len(repository.identifiers) != 0 ||
		repository.credential.ID != "" ||
		repository.declaration.ID != "" {
		t.Fatalf(
			"unauthorized side effects: resolve=%d begin=%d mail=%d user=%q identifiers=%d credential=%q declaration=%q",
			idempotencyStore.resolveCalls,
			len(idempotencyStore.beginRequests),
			len(mailerStub.messages),
			repository.user.ID,
			len(repository.identifiers),
			repository.credential.ID,
			repository.declaration.ID,
		)
	}
}

func TestAdministratorOnboardingRequiresPermanentCredentialAndCurrentNotice(t *testing.T) {
	acknowledged := "privacy-v1"
	user := User{
		Role:                                 RoleAdmin,
		CredentialMode:                       CredentialModeTemporary,
		OnboardingPrivacyAcknowledgedVersion: &acknowledged,
	}
	if administratorOnboardingComplete(user, "privacy-v1") {
		t.Fatal("temporary credential completed administrator onboarding")
	}

	user.CredentialMode = CredentialModePermanent
	if !administratorOnboardingComplete(user, "privacy-v1") {
		t.Fatal("permanent credential plus current acknowledgement did not complete onboarding")
	}
	if administratorOnboardingComplete(user, "privacy-v2") {
		t.Fatal("outdated acknowledgement completed administrator onboarding")
	}
}

func TestCreateAdministratorIdempotencyDigestIncludesDeclarationValue(t *testing.T) {
	service := newAdministratorCreateTestService(
		t,
		nil,
		&administratorCreateRepositoryStub{},
		&administratorIdempotencyStub{},
		&administratorMailerStub{},
		nil,
	)
	declared := administratorCreateTestRequest()
	notDeclared := declared
	notDeclared.InformationProvisionDeclared = false

	declaredRequest, err := service.idempotencyBeginRequest(
		declared,
		normalizeCreateAdministratorRequest(declared),
	)
	if err != nil {
		t.Fatalf("idempotencyBeginRequest(declared) error = %v", err)
	}
	notDeclaredRequest, err := service.idempotencyBeginRequest(
		notDeclared,
		normalizeCreateAdministratorRequest(notDeclared),
	)
	if err != nil {
		t.Fatalf("idempotencyBeginRequest(not declared) error = %v", err)
	}
	if string(declaredRequest.Request) == string(notDeclaredRequest.Request) {
		t.Fatalf(
			"declaration values produced identical request digest input: %s",
			declaredRequest.Request,
		)
	}
}

func TestCreateAdministratorSuccessPersistsBothPendingGatesWithoutSecretResponse(t *testing.T) {
	tx := &administratorTestTx{}
	repository := &administratorCreateRepositoryStub{}
	idempotencyStore := &administratorIdempotencyStub{
		steps: []administratorIdempotencyStep{{action: idempotency.BeginExecute}},
	}
	mailerStub := &administratorMailerStub{}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{tx},
		repository,
		idempotencyStore,
		mailerStub,
		nil,
	)

	result, err := service.CreateAdministrator(
		context.Background(),
		administratorCreateTestRequest(),
	)
	if err != nil {
		t.Fatalf("CreateAdministrator() error = %v", err)
	}
	if !repository.userCommitted() || repository.user.CredentialMode != CredentialModeTemporary ||
		repository.user.OnboardingCompletedAt != nil ||
		repository.user.OnboardingPrivacyAcknowledgedVersion != nil {
		t.Fatalf("created user = %#v, committed=%t", repository.user, repository.userCommitted())
	}
	if repository.credential.State != TemporaryCredentialPendingDelivery ||
		repository.activateCalls != 1 {
		t.Fatalf(
			"temporary credential = %#v, activate calls = %d",
			repository.credential,
			repository.activateCalls,
		)
	}
	if len(mailerStub.messages) != 1 ||
		mailerStub.messages[0].Purpose != mailer.PurposeNewAdministratorCredential ||
		!strings.Contains(mailerStub.messages[0].Body, "Temporary123") {
		t.Fatalf("mailer messages = %#v", mailerStub.messages)
	}
	if mailerStub.context == nil || !errors.Is(mailerStub.context.Err(), context.Canceled) {
		t.Fatal("mailer context was not canceled when Send returned")
	}
	if result.MustChangePassword != true || result.MustAcknowledgePrivacy != true ||
		result.DeliveryStatus != administratorDeliveryAccepted {
		t.Fatalf("CreateAdministrator() result = %#v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), "Temporary123") ||
		strings.Contains(string(encoded), repository.credential.SecretDigest) {
		t.Fatalf("safe result leaked credential: %s", encoded)
	}
	if len(idempotencyStore.completions) != 1 ||
		idempotencyStore.completions[0].Status != idempotency.StatusSucceeded ||
		idempotencyStore.completions[0].ResponseSnapshot.SchemaVersion !=
			idempotency.AdministratorCreateResponseSchemaV1 ||
		idempotencyStore.completions[0].ResponseSnapshot.Body == nil ||
		idempotencyStore.completions[0].ResponseSnapshot.Body.ID != result.ID {
		t.Fatalf("idempotency completions = %#v", idempotencyStore.completions)
	}
	if got := string(idempotencyStore.beginRequests[0].Scope); !strings.Contains(
		got,
		"initial-admin\x00"+administratorCreateOperation+"\x00ops",
	) {
		t.Fatalf("idempotency scope = %q", got)
	}
}

func TestCreateAdministratorStableReplayPrecedesDynamicPreconditions(t *testing.T) {
	request := administratorCreateTestRequest()
	result := CreatedAdministrator{
		ID: "administrator-replayed", Username: "ops", Role: RoleAdmin,
		Status: UserStatusActive, MustChangePassword: true,
		MustAcknowledgePrivacy: true,
		TemporaryCredentialExpiresAt: time.Date(
			2026, time.October, 24, 12, 0, 0, 0, time.UTC,
		),
		DeliveryStatus: administratorDeliveryAccepted,
	}
	replay := idempotency.BeginResult{
		Action: idempotency.BeginReplay,
		Record: idempotency.Record{
			Status: idempotency.StatusSucceeded,
			ResponseSnapshot: func() *idempotency.ResponseSnapshot {
				snapshot := administratorSuccessSnapshot(result)
				return &snapshot
			}(),
		},
	}
	idempotencyStore := &administratorIdempotencyStub{existing: &replay}
	mailerStub := &administratorMailerStub{}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{&administratorTestTx{}},
		&administratorCreateRepositoryStub{},
		idempotencyStore,
		mailerStub,
		errors.New("processing Gate changed after first response"),
	)
	request.InformationProvisionDeclared = false
	request.InformationProvisionNoticeVersion = "stale"
	request.TemporaryPassword = "weak"

	got, err := service.CreateAdministrator(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateAdministrator(replay) error = %v", err)
	}
	if got != result {
		t.Fatalf("CreateAdministrator(replay) = %#v, want %#v", got, result)
	}
	if len(idempotencyStore.beginRequests) != 0 || len(mailerStub.messages) != 0 {
		t.Fatalf(
			"replay produced side effects: begin=%d mail=%d",
			len(idempotencyStore.beginRequests),
			len(mailerStub.messages),
		)
	}
}

func TestCreateAdministratorReplayUsesImmutableSnapshot(t *testing.T) {
	service := &AdministratorCreateCoordinator{}
	result, err := service.replayCreateAdministrator(
		idempotency.Record{
			Status: idempotency.StatusSucceeded,
			ResponseSnapshot: &idempotency.ResponseSnapshot{
				SchemaVersion: idempotency.AdministratorCreateResponseSchemaV1,
				HTTPStatus:    201,
				ResultCode:    administratorCreateOutcome,
				Body: &idempotency.AdministratorCreateResponseBody{
					ID: "administrator-1", Username: "Ops",
					Role: "admin", Status: "active",
					MustChangePassword: true, MustAcknowledgePrivacy: true,
					TemporaryCredentialExpiresAt: "2026-10-24T08:00:00Z",
					DeliveryStatus:               "accepted",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("replayCreateAdministrator() error = %v", err)
	}
	if result.ID != "administrator-1" || result.Username != "Ops" {
		t.Fatalf("replayCreateAdministrator() result = %#v", result)
	}
}

func TestCreateAdministratorMailerUsesEarlierDeadlineAndCancels(t *testing.T) {
	t.Run("request deadline is earlier", func(t *testing.T) {
		mailerStub := &administratorMailerStub{}
		service := newAdministratorCreateTestService(
			t,
			[]pgx.Tx{&administratorTestTx{}},
			&administratorCreateRepositoryStub{},
			&administratorIdempotencyStub{
				steps: []administratorIdempotencyStep{{action: idempotency.BeginExecute}},
			},
			mailerStub,
			nil,
		)
		service.dependencies.SMTPOperationBudget = time.Second
		deadline := time.Now().Add(100 * time.Millisecond)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()
		if _, err := service.CreateAdministrator(ctx, administratorCreateTestRequest()); err != nil {
			t.Fatalf("CreateAdministrator() error = %v", err)
		}
		got, ok := mailerStub.context.Deadline()
		if !ok || !got.Equal(deadline) {
			t.Fatalf("mailer deadline = %v, %t, want %v", got, ok, deadline)
		}
		if !errors.Is(mailerStub.context.Err(), context.Canceled) {
			t.Fatal("mailer context remained active after Send")
		}
	})

	t.Run("operation budget is earlier", func(t *testing.T) {
		mailerStub := &administratorMailerStub{}
		service := newAdministratorCreateTestService(
			t,
			[]pgx.Tx{&administratorTestTx{}},
			&administratorCreateRepositoryStub{},
			&administratorIdempotencyStub{
				steps: []administratorIdempotencyStep{{action: idempotency.BeginExecute}},
			},
			mailerStub,
			nil,
		)
		service.dependencies.SMTPOperationBudget = 200 * time.Millisecond
		started := time.Now()
		if _, err := service.CreateAdministrator(
			context.Background(),
			administratorCreateTestRequest(),
		); err != nil {
			t.Fatalf("CreateAdministrator() error = %v", err)
		}
		got, ok := mailerStub.context.Deadline()
		if !ok || got.Before(started.Add(150*time.Millisecond)) ||
			got.After(started.Add(300*time.Millisecond)) {
			t.Fatalf("mailer deadline = %v, started = %v", got, started)
		}
		if !errors.Is(mailerStub.context.Err(), context.Canceled) {
			t.Fatal("mailer context remained active after Send")
		}
	})

	t.Run("already canceled request fails without usable administrator", func(t *testing.T) {
		first := &administratorTestTx{}
		failure := &administratorTestTx{}
		repository := &administratorCreateRepositoryStub{}
		mailerStub := &administratorMailerStub{waitForDone: true}
		service := newAdministratorCreateTestService(
			t,
			[]pgx.Tx{first, failure},
			repository,
			&administratorIdempotencyStub{
				steps: []administratorIdempotencyStep{
					{action: idempotency.BeginExecute},
					{action: idempotency.BeginExecute},
				},
			},
			mailerStub,
			nil,
		)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := service.CreateAdministrator(ctx, administratorCreateTestRequest())
		if code := platform.PublicCode(err); code != platform.CodeIdentityEmailDeliveryFailed {
			t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeIdentityEmailDeliveryFailed)
		}
		if repository.userCommitted() || repository.activateCalls != 0 {
			t.Fatal("canceled mail delivery left a usable administrator")
		}
	})
}

func TestCreateAdministratorSMTPFailureRollsBackAndStoresStableFailure(t *testing.T) {
	first := &administratorTestTx{}
	failure := &administratorTestTx{}
	repository := &administratorCreateRepositoryStub{}
	idempotencyStore := &administratorIdempotencyStub{
		steps: []administratorIdempotencyStep{
			{action: idempotency.BeginExecute},
			{action: idempotency.BeginExecute},
		},
	}
	mailerStub := &administratorMailerStub{err: errors.New("smtp not accepted")}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{first, failure},
		repository,
		idempotencyStore,
		mailerStub,
		nil,
	)

	_, err := service.CreateAdministrator(
		context.Background(),
		administratorCreateTestRequest(),
	)
	if code := platform.PublicCode(err); code != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeIdentityEmailDeliveryFailed)
	}
	if repository.userCommitted() || repository.activateCalls != 0 {
		t.Fatalf(
			"SMTP failure left usable administrator: committed=%t activate=%d",
			repository.userCommitted(),
			repository.activateCalls,
		)
	}
	if first.rollbackCalls != 1 {
		t.Fatalf("business rollback calls = %d, want 1", first.rollbackCalls)
	}
	if len(idempotencyStore.completions) != 1 ||
		idempotencyStore.completions[0].Status != idempotency.StatusFailed ||
		idempotencyStore.completions[0].ResponseSnapshot.ResultCode !=
			string(platform.CodeIdentityEmailDeliveryFailed) ||
		idempotencyStore.completions[0].ResponseSnapshot.Body == nil ||
		idempotencyStore.completions[0].ResponseSnapshot.Body.FieldErrors == nil {
		t.Fatalf("idempotency completions = %#v", idempotencyStore.completions)
	}
}

func TestCreateAdministratorReconcilesSMTPFailureAfterParentCancellation(t *testing.T) {
	const cleanupBudget = 2 * time.Second
	type contextKey string

	parentValue := contextKey("parent-value")
	business := &administratorContextAwareTx{}
	failure := &administratorContextAwareTx{}
	replay := &administratorContextAwareTx{}
	transactor := &administratorContextAwareTransactor{
		transactions: []pgx.Tx{business, failure, replay},
	}
	repository := &administratorCreateRepositoryStub{}
	idempotencyStore := &administratorContextAwareIdempotencyStore{
		administratorIdempotencyStub: &administratorIdempotencyStub{
			steps: []administratorIdempotencyStep{
				{action: idempotency.BeginExecute},
				{action: idempotency.BeginExecute},
			},
		},
	}
	ctx, cancelParent := context.WithCancel(
		context.WithValue(context.Background(), parentValue, "retained"),
	)
	mailerStub := &administratorCancelingMailer{
		cancel: cancelParent,
		err:    errors.New("smtp not accepted"),
	}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{business, failure, replay},
		repository,
		idempotencyStore.administratorIdempotencyStub,
		&administratorMailerStub{},
		nil,
	)
	service.dependencies.Transactions = transactor
	service.dependencies.Idempotency = idempotencyStore
	service.dependencies.Mailer = mailerStub
	service.dependencies.SMTPOperationBudget = cleanupBudget
	auditStub := &administratorContextAwareAudit{}
	service.dependencies.Audit = auditStub

	started := time.Now()
	_, err := service.CreateAdministrator(ctx, administratorCreateTestRequest())
	if code := platform.PublicCode(err); code != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeIdentityEmailDeliveryFailed)
	}
	if business.rollbackContext == nil || business.rollbackContext.Err() != context.Canceled {
		t.Fatalf("business rollback context after return = %#v", business.rollbackContext)
	}
	assertAdministratorCleanupContext(
		t,
		business.rollbackContext,
		started,
		cleanupBudget,
		parentValue,
	)
	if len(transactor.contexts) != 2 {
		t.Fatalf("transaction begin contexts = %d, want 2", len(transactor.contexts))
	}
	assertAdministratorCleanupContext(
		t,
		transactor.contexts[1],
		started,
		cleanupBudget,
		parentValue,
	)
	if failure.commitCalls != 1 {
		t.Fatalf("failure transaction commits = %d, want 1", failure.commitCalls)
	}
	if len(idempotencyStore.completions) != 1 ||
		idempotencyStore.completions[0].Status != idempotency.StatusFailed ||
		idempotencyStore.completions[0].ResponseSnapshot.ResultCode !=
			string(platform.CodeIdentityEmailDeliveryFailed) {
		t.Fatalf("idempotency completions = %#v", idempotencyStore.completions)
	}
	if len(auditStub.events) != 1 || auditStub.events[0].Outcome != "failed" {
		t.Fatalf("failure audit events = %#v", auditStub.events)
	}

	snapshot := idempotencyStore.completions[0].ResponseSnapshot
	idempotencyStore.existing = &idempotency.BeginResult{
		Action: idempotency.BeginReplay,
		Record: idempotency.Record{
			Status:           idempotency.StatusFailed,
			ResponseSnapshot: &snapshot,
		},
	}
	_, err = service.CreateAdministrator(
		context.Background(),
		administratorCreateTestRequest(),
	)
	if code := platform.PublicCode(err); code != platform.CodeIdentityEmailDeliveryFailed {
		t.Fatalf("replay PublicCode() = %q, want %q", code, platform.CodeIdentityEmailDeliveryFailed)
	}
	if mailerStub.calls != 1 {
		t.Fatalf("mailer calls after replay = %d, want 1", mailerStub.calls)
	}
}

func TestCreateAdministratorMarksUncertainAfterAcceptedMailParentCancellation(t *testing.T) {
	const cleanupBudget = 2 * time.Second
	type contextKey string

	parentValue := contextKey("parent-value")
	business := &administratorContextAwareTx{}
	uncertain := &administratorContextAwareTx{}
	retry := &administratorContextAwareTx{}
	transactor := &administratorContextAwareTransactor{
		transactions: []pgx.Tx{business, uncertain, retry},
	}
	repositoryStub := &administratorCreateRepositoryStub{}
	repository := &administratorContextAwareCreateRepository{
		administratorCreateRepositoryStub: repositoryStub,
	}
	idempotencyStore := &administratorContextAwareIdempotencyStore{
		administratorIdempotencyStub: &administratorIdempotencyStub{
			steps: []administratorIdempotencyStep{
				{action: idempotency.BeginExecute},
				{action: idempotency.BeginInProgress},
			},
		},
	}
	ctx, cancelParent := context.WithCancel(
		context.WithValue(context.Background(), parentValue, "retained"),
	)
	mailerStub := &administratorCancelingMailer{cancel: cancelParent}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{business, uncertain, retry},
		repositoryStub,
		idempotencyStore.administratorIdempotencyStub,
		&administratorMailerStub{},
		nil,
	)
	service.dependencies.Transactions = transactor
	service.dependencies.Repository = repository
	service.dependencies.Idempotency = idempotencyStore
	service.dependencies.Mailer = mailerStub
	service.dependencies.SMTPOperationBudget = cleanupBudget

	started := time.Now()
	_, err := service.CreateAdministrator(ctx, administratorCreateTestRequest())
	if code := platform.PublicCode(err); code != platform.CodeIdentityOperationUncertain {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeIdentityOperationUncertain)
	}
	if repository.activateCalls != 1 {
		t.Fatalf("temporary credential activation calls = %d, want 1", repository.activateCalls)
	}
	if business.rollbackContext == nil || business.rollbackContext.Err() != context.Canceled {
		t.Fatalf("business rollback context after return = %#v", business.rollbackContext)
	}
	assertAdministratorCleanupContext(
		t,
		business.rollbackContext,
		started,
		cleanupBudget,
		parentValue,
	)
	if len(transactor.contexts) != 2 {
		t.Fatalf("transaction begin contexts = %d, want 2", len(transactor.contexts))
	}
	assertAdministratorCleanupContext(
		t,
		transactor.contexts[1],
		started,
		cleanupBudget,
		parentValue,
	)
	if uncertain.commitCalls != 1 || len(idempotencyStore.uncertain) != 1 {
		t.Fatalf(
			"uncertain reconciliation: commits=%d transitions=%#v",
			uncertain.commitCalls,
			idempotencyStore.uncertain,
		)
	}

	idempotencyStore.existing = &idempotency.BeginResult{
		Action: idempotency.BeginOperationUncertain,
		Record: idempotency.Record{Status: idempotency.StatusUncertain},
	}
	_, err = service.CreateAdministrator(
		context.Background(),
		administratorCreateTestRequest(),
	)
	if code := platform.PublicCode(err); code != platform.CodeIdentityOperationUncertain {
		t.Fatalf("retry PublicCode() = %q, want %q", code, platform.CodeIdentityOperationUncertain)
	}
	if mailerStub.calls != 1 {
		t.Fatalf("mailer calls after retry = %d, want 1", mailerStub.calls)
	}
}

func TestCreateAdministratorFailClosedIdempotencyStates(t *testing.T) {
	tests := []struct {
		name     string
		step     administratorIdempotencyStep
		wantCode platform.Code
	}{
		{
			name:     "same key different request",
			step:     administratorIdempotencyStep{err: idempotency.ErrDigestConflict},
			wantCode: platform.CodeIdentityIdempotencyConflict,
		},
		{
			name:     "processing",
			step:     administratorIdempotencyStep{action: idempotency.BeginInProgress},
			wantCode: platform.CodeIdentityIdempotencyInProgress,
		},
		{
			name:     "uncertain or recovery review",
			step:     administratorIdempotencyStep{action: idempotency.BeginOperationUncertain},
			wantCode: platform.CodeIdentityOperationUncertain,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mailerStub := &administratorMailerStub{}
			service := newAdministratorCreateTestService(
				t,
				[]pgx.Tx{&administratorTestTx{}},
				&administratorCreateRepositoryStub{},
				&administratorIdempotencyStub{
					steps: []administratorIdempotencyStep{test.step},
				},
				mailerStub,
				nil,
			)
			_, err := service.CreateAdministrator(
				context.Background(),
				administratorCreateTestRequest(),
			)
			if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf("PublicCode() = %q, want %q", code, test.wantCode)
			}
			if len(mailerStub.messages) != 0 {
				t.Fatalf("fail-closed idempotency sent mail: %#v", mailerStub.messages)
			}
		})
	}
}

func TestCreateAdministratorAcceptedCommitFailureReturnsUncertainAndUnusable(t *testing.T) {
	business := &administratorTestTx{commitErr: errors.New("commit outcome unknown")}
	uncertain := &administratorTestTx{}
	repository := &administratorCreateRepositoryStub{}
	idempotencyStore := &administratorIdempotencyStub{
		steps: []administratorIdempotencyStep{
			{action: idempotency.BeginExecute},
			{action: idempotency.BeginInProgress},
		},
	}
	service := newAdministratorCreateTestService(
		t,
		[]pgx.Tx{business, uncertain},
		repository,
		idempotencyStore,
		&administratorMailerStub{},
		nil,
	)

	_, err := service.CreateAdministrator(
		context.Background(),
		administratorCreateTestRequest(),
	)
	if code := platform.PublicCode(err); code != platform.CodeIdentityOperationUncertain {
		t.Fatalf("PublicCode() = %q, want %q", code, platform.CodeIdentityOperationUncertain)
	}
	if repository.userCommitted() {
		t.Fatal("commit failure left test administrator usable")
	}
	if len(idempotencyStore.uncertain) != 1 {
		t.Fatalf("uncertain transitions = %#v", idempotencyStore.uncertain)
	}
}

func TestCreateAdministratorGateAndCurrentDeclarationPrecedeIdempotencyAndMail(t *testing.T) {
	tests := []struct {
		name     string
		gateErr  error
		mutate   func(*CreateAdministratorRequest)
		wantCode platform.Code
	}{
		{
			name:     "processing gate closed",
			gateErr:  errors.New("processing inputs missing"),
			wantCode: platform.CodePlatformProcessingGateClosed,
		},
		{
			name: "declaration missing",
			mutate: func(request *CreateAdministratorRequest) {
				request.InformationProvisionDeclared = false
			},
			wantCode: platform.CodePrivacyInformationDeclarationRequired,
		},
		{
			name: "declaration outdated",
			mutate: func(request *CreateAdministratorRequest) {
				request.InformationProvisionNoticeVersion = "administrator-notice-v0"
			},
			wantCode: platform.CodePrivacyNoticeOutdated,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			idempotencyStore := &administratorIdempotencyStub{}
			mailerStub := &administratorMailerStub{}
			service := newAdministratorCreateTestService(
				t,
				[]pgx.Tx{&administratorTestTx{}},
				&administratorCreateRepositoryStub{},
				idempotencyStore,
				mailerStub,
				test.gateErr,
			)
			request := administratorCreateTestRequest()
			if test.mutate != nil {
				test.mutate(&request)
			}
			_, err := service.CreateAdministrator(context.Background(), request)
			if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf("PublicCode() = %q, want %q", code, test.wantCode)
			}
			if len(idempotencyStore.beginRequests) != 0 ||
				len(mailerStub.messages) != 0 {
				t.Fatalf(
					"precondition failure reached side effects: begin=%d mail=%d",
					len(idempotencyStore.beginRequests),
					len(mailerStub.messages),
				)
			}
		})
	}
}

func TestAdministratorOnboardingAllowsEitherOrderAndTracksCurrentNotice(t *testing.T) {
	t.Run("privacy then password", func(t *testing.T) {
		repository := newAdministratorOnboardingRepository("privacy-v1")
		privacy := newAdministratorPrivacyTestService(t, repository)
		result, err := privacy.AcknowledgePrivacy(
			context.Background(),
			administratorPrivacyTestRequest(repository, session.CredentialTemporary),
		)
		if err != nil {
			t.Fatalf("AcknowledgePrivacy() error = %v", err)
		}
		if !result.MustChangePassword || !result.MustAcknowledgePrivacy ||
			repository.user.OnboardingCompletedAt != nil {
			t.Fatalf("privacy-first result = %#v user=%#v", result, repository.user)
		}

		password := newAdministratorPasswordTestService(t, repository)
		passwordResult, err := password.ChangePassword(
			context.Background(),
			administratorPasswordTestRequest(repository),
		)
		if err != nil {
			t.Fatalf("ChangePassword() error = %v", err)
		}
		if passwordResult.MustAcknowledgePrivacy ||
			repository.user.OnboardingCompletedAt == nil {
			t.Fatalf(
				"privacy-first final result = %#v user=%#v",
				passwordResult,
				repository.user,
			)
		}
	})

	t.Run("password then privacy with notice rotation", func(t *testing.T) {
		repository := newAdministratorOnboardingRepository("privacy-v1")
		password := newAdministratorPasswordTestService(t, repository)
		passwordResult, err := password.ChangePassword(
			context.Background(),
			administratorPasswordTestRequest(repository),
		)
		if err != nil {
			t.Fatalf("ChangePassword() error = %v", err)
		}
		if !passwordResult.MustAcknowledgePrivacy ||
			repository.user.OnboardingCompletedAt != nil {
			t.Fatalf(
				"password-first result = %#v user=%#v",
				passwordResult,
				repository.user,
			)
		}

		repository.currentNotice = "privacy-v2"
		privacy := newAdministratorPrivacyTestService(t, repository)
		privacyRequest := administratorPrivacyTestRequest(
			repository,
			session.CredentialPermanent,
		)
		privacyRequest.PrivacyNoticeVersion = "privacy-v1"
		_, err = privacy.AcknowledgePrivacy(context.Background(), privacyRequest)
		if code := platform.PublicCode(err); code != platform.CodePrivacyNoticeOutdated {
			t.Fatalf("outdated PublicCode() = %q, want %q", code, platform.CodePrivacyNoticeOutdated)
		}
		if repository.user.OnboardingCompletedAt != nil {
			t.Fatal("outdated acknowledgement completed onboarding")
		}

		privacyRequest.PrivacyNoticeVersion = "privacy-v2"
		privacyResult, err := privacy.AcknowledgePrivacy(
			context.Background(),
			privacyRequest,
		)
		if err != nil {
			t.Fatalf("current AcknowledgePrivacy() error = %v", err)
		}
		if privacyResult.MustChangePassword || privacyResult.MustAcknowledgePrivacy ||
			repository.user.OnboardingCompletedAt == nil {
			t.Fatalf(
				"password-first final result = %#v user=%#v",
				privacyResult,
				repository.user,
			)
		}

		repository.currentNotice = "privacy-v3"
		if repository.user.OnboardingCompletedAt == nil {
			t.Fatal("completed onboarding was relocked after notice rotation")
		}
	})
}

func administratorCreateTestRequest() CreateAdministratorRequest {
	return CreateAdministratorRequest{
		Principal: session.Principal{
			SessionID: "initial-session", UserID: "initial-admin",
			Role: string(RoleAdmin), IsInitialAdmin: true,
			Status: string(UserStatusActive),
		},
		IdempotencyKey:                    "0123456789abcdef",
		Username:                          "ops",
		Email:                             "ops@example.test",
		Phone:                             "+10000000000",
		TemporaryPassword:                 "Temporary123",
		InformationProvisionNoticeVersion: "administrator-notice-v1",
		InformationProvisionDeclared:      true,
		TraceID:                           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}
}

func newAdministratorCreateTestService(
	t *testing.T,
	transactions []pgx.Tx,
	repository *administratorCreateRepositoryStub,
	idempotencyStore *administratorIdempotencyStub,
	mailerStub *administratorMailerStub,
	gateErr error,
) *AdministratorCreateCoordinator {
	t.Helper()
	repository.transactions = transactions
	service, err := NewAdministratorCreateService(AdministratorCreateDependencies{
		Transactions: &administratorSequenceTransactor{transactions: transactions},
		Repository:   repository,
		InformationNotices: administratorInformationNoticeStub{
			notice: AdministratorInformationNotice{
				Version: "administrator-notice-v1",
				EffectiveAt: time.Date(
					2026,
					time.October,
					23,
					0,
					0,
					0,
					0,
					time.UTC,
				),
				Content:     "I am authorized to provide this administrator information.",
				ContentHash: "sha256:administrator-notice",
			},
		},
		Gate:           administratorGateStub{err: gateErr},
		PasswordPolicy: security.PasswordPolicy{},
		PasswordHasher: administratorPasswordHasherStub{},
		Mailer:         mailerStub,
		Idempotency:    idempotencyStore,
		IdempotencyKeys: idempotency.DigestKeyring{
			ActiveVersion: "v1",
			Keys:          map[string][]byte{"v1": []byte("administrator-test-digest-key")},
		},
		IdempotencyLock:     time.Minute,
		IdempotencyTTL:      24 * time.Hour,
		SMTPOperationBudget: time.Second,
		Audit:               &bootstrapTestAudit{},
		IDs:                 &bootstrapTestIDs{},
		Clock:               bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewAdministratorCreateService() error = %v", err)
	}
	return service
}

type administratorSequenceTransactor struct {
	transactions []pgx.Tx
	next         int
}

func (t *administratorSequenceTransactor) Begin(context.Context) (pgx.Tx, error) {
	if t.next >= len(t.transactions) {
		return nil, errors.New("unexpected administrator transaction")
	}
	tx := t.transactions[t.next]
	t.next++
	return tx, nil
}

type administratorTestTx struct {
	bootstrapTestTx
	commitErr error
}

func (tx *administratorTestTx) Commit(context.Context) error {
	tx.commitCalls++
	tx.closed = true
	return tx.commitErr
}

type administratorContextAwareTx struct {
	administratorTestTx
	commitContext   context.Context
	rollbackContext context.Context
}

func (tx *administratorContextAwareTx) Commit(ctx context.Context) error {
	tx.commitContext = ctx
	tx.commitCalls++
	if err := ctx.Err(); err != nil {
		return err
	}
	tx.closed = true
	return tx.commitErr
}

func (tx *administratorContextAwareTx) Rollback(ctx context.Context) error {
	tx.rollbackContext = ctx
	tx.rollbackCalls++
	if err := ctx.Err(); err != nil {
		return err
	}
	tx.closed = true
	return nil
}

type administratorContextAwareTransactor struct {
	transactions []pgx.Tx
	contexts     []context.Context
	next         int
}

func (t *administratorContextAwareTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	t.contexts = append(t.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if t.next >= len(t.transactions) {
		return nil, errors.New("unexpected administrator transaction")
	}
	tx := t.transactions[t.next]
	t.next++
	return tx, nil
}

type administratorCreateRepositoryStub struct {
	transactions       []pgx.Tx
	authorizationUser  User
	authorizationCalls int
	user               User
	identifiers        []IdentityIdentifier
	credential         TemporaryCredential
	declaration        AdministratorInformationDeclaration
	activateCalls      int
}

func (r *administratorCreateRepositoryStub) FindAdministratorCreationActor(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) (User, error) {
	r.authorizationCalls++
	if r.authorizationUser.ID != "" {
		return r.authorizationUser, nil
	}
	return User{
		ID: userID, Role: RoleAdmin, IsInitialAdmin: true, Status: UserStatusActive,
	}, nil
}

func (r *administratorCreateRepositoryStub) InsertUser(
	_ context.Context,
	_ pgx.Tx,
	user User,
) error {
	r.user = user
	return nil
}

func (r *administratorCreateRepositoryStub) InsertIdentifier(
	_ context.Context,
	_ pgx.Tx,
	identifier IdentityIdentifier,
) error {
	r.identifiers = append(r.identifiers, identifier)
	return nil
}

func (r *administratorCreateRepositoryStub) InsertTemporaryCredential(
	_ context.Context,
	_ pgx.Tx,
	credential TemporaryCredential,
) error {
	r.credential = credential
	return nil
}

func (r *administratorCreateRepositoryStub) InsertAdministratorInformationDeclaration(
	_ context.Context,
	_ pgx.Tx,
	declaration AdministratorInformationDeclaration,
) error {
	r.declaration = declaration
	return nil
}

func (r *administratorCreateRepositoryStub) ActivateTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
) error {
	r.activateCalls++
	return nil
}

func (r *administratorCreateRepositoryStub) FindCreatedAdministrator(
	context.Context,
	pgx.Tx,
	string,
) (CreatedAdministrator, error) {
	return CreatedAdministrator{
		ID: r.user.ID, Username: r.user.Username, Role: r.user.Role,
		Status: r.user.Status, MustChangePassword: true,
		MustAcknowledgePrivacy:       true,
		TemporaryCredentialExpiresAt: r.credential.ExpiresAt,
		DeliveryStatus:               administratorDeliveryAccepted,
	}, nil
}

func (r *administratorCreateRepositoryStub) userCommitted() bool {
	if len(r.transactions) == 0 {
		return false
	}
	tx, ok := r.transactions[0].(*administratorTestTx)
	return ok && tx.commitCalls == 1 && tx.commitErr == nil
}

type administratorContextAwareCreateRepository struct {
	*administratorCreateRepositoryStub
}

func (r *administratorContextAwareCreateRepository) ActivateTemporaryCredential(
	ctx context.Context,
	_ pgx.Tx,
	_ string,
	_ time.Time,
) error {
	r.activateCalls++
	return ctx.Err()
}

type administratorInformationNoticeStub struct {
	notice AdministratorInformationNotice
	err    error
}

func (s administratorInformationNoticeStub) CurrentAdministratorInformationNotice(
	context.Context,
	pgx.Tx,
) (AdministratorInformationNotice, error) {
	return s.notice, s.err
}

type administratorGateStub struct{ err error }

func (g administratorGateStub) RequireOpen(context.Context) error { return g.err }

type administratorPasswordHasherStub struct{}

func (administratorPasswordHasherStub) Hash(string) (string, error) {
	return "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0", nil
}

func (administratorPasswordHasherStub) Verify(
	string,
	string,
) (bool, bool, error) {
	return true, false, nil
}

type administratorMailerStub struct {
	messages    []mailer.Message
	err         error
	context     context.Context
	waitForDone bool
}

func (m *administratorMailerStub) Send(
	ctx context.Context,
	message mailer.Message,
) error {
	m.context = ctx
	m.messages = append(m.messages, message)
	if m.waitForDone {
		<-ctx.Done()
		return ctx.Err()
	}
	return m.err
}

type administratorCancelingMailer struct {
	cancel context.CancelFunc
	err    error
	calls  int
}

func (m *administratorCancelingMailer) Send(
	context.Context,
	mailer.Message,
) error {
	m.calls++
	m.cancel()
	return m.err
}

type administratorIdempotencyStep struct {
	action idempotency.BeginAction
	status idempotency.Status
	err    error
}

type administratorIdempotencyStub struct {
	steps           []administratorIdempotencyStep
	existing        *idempotency.BeginResult
	resolveCalls    int
	resolveRequests []idempotency.BeginRequest
	beginRequests   []idempotency.BeginRequest
	completions     []idempotency.Completion
	uncertain       []idempotency.Transition
}

func (s *administratorIdempotencyStub) ResolveExisting(
	_ context.Context,
	_ idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	s.resolveCalls++
	s.resolveRequests = append(s.resolveRequests, request)
	if s.existing == nil {
		return idempotency.BeginResult{}, false, nil
	}
	return *s.existing, true, nil
}

func (s *administratorIdempotencyStub) Begin(
	_ context.Context,
	_ idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	s.beginRequests = append(s.beginRequests, request)
	if len(s.steps) == 0 {
		return idempotency.BeginResult{}, errors.New("unexpected idempotency begin")
	}
	step := s.steps[0]
	s.steps = s.steps[1:]
	if step.err != nil {
		return idempotency.BeginResult{}, step.err
	}
	status := step.status
	if status == "" {
		status = idempotency.StatusProcessing
	}
	return idempotency.BeginResult{
		Action: step.action,
		Record: idempotency.Record{
			ScopeHash: []byte("scope"), KeyLocator: []byte("locator"),
			KeyDigest: []byte("key"), RequestDigest: []byte("request"),
			DigestKeyVersion: "v1", Status: status, ExecutionGeneration: 1,
		},
	}, nil
}

func (s *administratorIdempotencyStub) Complete(
	_ context.Context,
	_ idempotency.Transaction,
	completion idempotency.Completion,
) error {
	s.completions = append(s.completions, completion)
	return nil
}

func (s *administratorIdempotencyStub) MarkUncertain(
	_ context.Context,
	_ idempotency.Transaction,
	transition idempotency.Transition,
) error {
	s.uncertain = append(s.uncertain, transition)
	return nil
}

type administratorContextAwareIdempotencyStore struct {
	*administratorIdempotencyStub
	contexts []context.Context
}

func (s *administratorContextAwareIdempotencyStore) ResolveExisting(
	ctx context.Context,
	tx idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, bool, error) {
	s.contexts = append(s.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return idempotency.BeginResult{}, false, err
	}
	return s.administratorIdempotencyStub.ResolveExisting(ctx, tx, request)
}

func (s *administratorContextAwareIdempotencyStore) Begin(
	ctx context.Context,
	tx idempotency.Transaction,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	s.contexts = append(s.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return idempotency.BeginResult{}, err
	}
	return s.administratorIdempotencyStub.Begin(ctx, tx, request)
}

func (s *administratorContextAwareIdempotencyStore) Complete(
	ctx context.Context,
	tx idempotency.Transaction,
	completion idempotency.Completion,
) error {
	s.contexts = append(s.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.administratorIdempotencyStub.Complete(ctx, tx, completion)
}

func (s *administratorContextAwareIdempotencyStore) MarkUncertain(
	ctx context.Context,
	tx idempotency.Transaction,
	transition idempotency.Transition,
) error {
	s.contexts = append(s.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.administratorIdempotencyStub.MarkUncertain(ctx, tx, transition)
}

type administratorContextAwareAudit struct {
	bootstrapTestAudit
	contexts []context.Context
}

func (a *administratorContextAwareAudit) Append(
	ctx context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	a.contexts = append(a.contexts, ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.bootstrapTestAudit.Append(ctx, tx, event)
}

func assertAdministratorCleanupContext(
	t *testing.T,
	ctx context.Context,
	started time.Time,
	budget time.Duration,
	valueKey any,
) {
	t.Helper()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("cleanup context has no deadline")
	}
	if deadline.Before(started) || deadline.After(started.Add(budget+100*time.Millisecond)) {
		t.Fatalf(
			"cleanup deadline = %v, want after %v and within %v",
			deadline,
			started,
			budget,
		)
	}
	if got := ctx.Value(valueKey); got != "retained" {
		t.Fatalf("cleanup context value = %v, want retained", got)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("cleanup context error after return = %v, want canceled", ctx.Err())
	}
}

type administratorOnboardingRepository struct {
	user          User
	temporary     TemporaryCredential
	currentNotice string
}

func newAdministratorOnboardingRepository(
	currentNotice string,
) *administratorOnboardingRepository {
	now := bootstrapTestClock{}.Now()
	credentialID := "temporary-credential"
	return &administratorOnboardingRepository{
		user: User{
			ID: "administrator-1", Username: "ops", Role: RoleAdmin,
			Status: UserStatusActive, CredentialMode: CredentialModeTemporary,
			CreatedAt: now, UpdatedAt: now,
		},
		temporary: TemporaryCredential{
			ID: credentialID, UserID: "administrator-1",
			Purpose: TemporaryCredentialNewAdministrator, Generation: 1,
			State:            TemporaryCredentialActive,
			SecretDigest:     "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0",
			DeliveredToEmail: "ops@example.test", IssuedAt: now,
			ExpiresAt: now.Add(temporaryCredentialLifetime),
		},
		currentNotice: currentNotice,
	}
}

func (*administratorOnboardingRepository) FindPasswordResetOperationForPasswordChange(
	context.Context,
	pgx.Tx,
	string,
	string,
	string,
) (PasswordResetOperation, error) {
	return PasswordResetOperation{}, pgx.ErrNoRows
}

func (r *administratorOnboardingRepository) FindUserForPasswordChange(
	context.Context,
	pgx.Tx,
	string,
) (User, error) {
	return r.user, nil
}

func (r *administratorOnboardingRepository) FindLatestTemporaryCredential(
	context.Context,
	pgx.Tx,
	string,
) (TemporaryCredential, error) {
	return r.temporary, nil
}

func (r *administratorOnboardingRepository) SetPermanentPassword(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	digest string,
	mode CredentialMode,
	_ int64,
	successorGeneration int64,
	sourceOperationID *string,
	sourceCredentialID *string,
	updatedAt time.Time,
) error {
	r.user.PermanentPasswordDigest = &digest
	r.user.CredentialMode = mode
	r.user.AuthGeneration = successorGeneration
	r.user.PermanentPasswordGeneration = &successorGeneration
	r.user.PermanentPasswordSourceOperationID = sourceOperationID
	r.user.PermanentPasswordSourceCredentialID = sourceCredentialID
	r.user.UpdatedAt = updatedAt
	return nil
}

func (*administratorOnboardingRepository) SettlePasswordResetSuccessor(
	context.Context,
	pgx.Tx,
	PasswordResetOperation,
	TemporaryCredential,
	int64,
	time.Time,
) error {
	return nil
}

func (r *administratorOnboardingRepository) ConsumeTemporaryCredential(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	consumedAt time.Time,
) error {
	r.temporary.State = TemporaryCredentialConsumed
	r.temporary.ConsumedAt = &consumedAt
	return nil
}

func (r *administratorOnboardingRepository) CurrentNotice(
	context.Context,
	pgx.Tx,
) (PrivacyNotice, error) {
	return PrivacyNotice{Version: r.currentNotice, Status: PrivacyNoticeCurrent}, nil
}

func (r *administratorOnboardingRepository) CompleteAdministratorOnboarding(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	noticeVersion string,
	completedAt time.Time,
) error {
	if !administratorOnboardingComplete(r.user, noticeVersion) {
		return errors.New("administrator onboarding Gates are incomplete")
	}
	r.user.OnboardingCompletedAt = &completedAt
	return nil
}

func (r *administratorOnboardingRepository) FindAdministratorForOnboarding(
	context.Context,
	pgx.Tx,
	string,
) (User, error) {
	return r.user, nil
}

func (r *administratorOnboardingRepository) InsertPrivacyAcknowledgement(
	context.Context,
	pgx.Tx,
	PrivacyAcknowledgement,
) error {
	return nil
}

func (r *administratorOnboardingRepository) SetAdministratorPrivacyAcknowledgement(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	noticeVersion string,
	complete bool,
	updatedAt time.Time,
) error {
	r.user.OnboardingPrivacyAcknowledgedVersion = &noticeVersion
	r.user.UpdatedAt = updatedAt
	if complete {
		r.user.OnboardingCompletedAt = &updatedAt
	}
	return nil
}

func newAdministratorPasswordTestService(
	t *testing.T,
	repository *administratorOnboardingRepository,
) *Service {
	t.Helper()
	service, err := NewPasswordChangeService(PasswordChangeDependencies{
		Transactions:      &administratorFreshTransactor{},
		Repository:        repository,
		PasswordResetGate: &loginPasswordResetGateStub{},
		Passwords:         administratorOnboardingPasswords{},
		Policy:            security.PasswordPolicy{},
		Sessions:          &onboardingTestSessions{},
		SessionIssuer:     administratorOnboardingIssuer{},
		Audit:             &bootstrapTestAudit{},
		IDs:               &bootstrapTestIDs{},
		Clock:             bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewPasswordChangeService() error = %v", err)
	}
	return service
}

func newAdministratorPrivacyTestService(
	t *testing.T,
	repository *administratorOnboardingRepository,
) *AdministratorPrivacyCoordinator {
	t.Helper()
	service, err := NewAdministratorPrivacyService(AdministratorPrivacyDependencies{
		Transactions:  &administratorFreshTransactor{},
		Repository:    repository,
		Notices:       administratorPrivacyNoticeReader{repository: repository},
		Sessions:      &onboardingTestSessions{},
		SessionIssuer: administratorOnboardingIssuer{},
		Audit:         &bootstrapTestAudit{},
		IDs:           &bootstrapTestIDs{},
		Clock:         bootstrapTestClock{},
	})
	if err != nil {
		t.Fatalf("NewAdministratorPrivacyService() error = %v", err)
	}
	return service
}

type administratorFreshTransactor struct{}

func (*administratorFreshTransactor) Begin(context.Context) (pgx.Tx, error) {
	return &administratorTestTx{}, nil
}

type administratorPrivacyNoticeReader struct {
	repository *administratorOnboardingRepository
}

func (r administratorPrivacyNoticeReader) CurrentNotice(
	context.Context,
	pgx.Tx,
) (PrivacyNotice, error) {
	return PrivacyNotice{
		Version: r.repository.currentNotice, Status: PrivacyNoticeCurrent,
	}, nil
}

type administratorOnboardingPasswords struct{}

func (administratorOnboardingPasswords) Hash(string) (string, error) {
	return "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$bmV3ZGlnZXN0", nil
}

func (administratorOnboardingPasswords) Verify(
	value string,
	digest string,
) (bool, bool, error) {
	return value == "Temporary123" &&
		digest == "$argon2id$v=19$m=65536,t=3,p=1$c2FsdA$ZGlnZXN0", false, nil
}

type administratorOnboardingIssuer struct{}

func (administratorOnboardingIssuer) Issue(
	user User,
	kind session.CredentialKind,
	credentialID *string,
) (session.CreateInput, error) {
	now := bootstrapTestClock{}.Now()
	return session.CreateInput{
		ID: "replacement-session", UserID: user.ID,
		Token: "replacement-token", CSRFToken: "replacement-csrf",
		AuthenticatedCredentialKind: kind,
		AuthenticatedCredentialID:   credentialID,
		IssuedAt:                    now, AbsoluteExpiresAt: now.Add(time.Hour),
		IdleExpiresAt: now.Add(30 * time.Minute),
	}, nil
}

func administratorPasswordTestRequest(
	repository *administratorOnboardingRepository,
) PasswordChangeRequest {
	credentialID := repository.temporary.ID
	return PasswordChangeRequest{
		Principal: session.Principal{
			SessionID: "temporary-session", UserID: repository.user.ID,
			Role: string(RoleAdmin), Status: string(UserStatusActive),
			MustChangePassword:        true,
			AuthenticatedCredential:   session.CredentialTemporary,
			AuthenticatedCredentialID: &credentialID,
		},
		CurrentSessionToken: "temporary-token",
		CurrentPassword:     "Temporary123", NewPassword: "Permanent123",
		TraceID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}
}

func administratorPrivacyTestRequest(
	repository *administratorOnboardingRepository,
	credential session.CredentialKind,
) AdministratorPrivacyAcknowledgementRequest {
	var credentialID *string
	if credential == session.CredentialTemporary {
		id := repository.temporary.ID
		credentialID = &id
	}
	return AdministratorPrivacyAcknowledgementRequest{
		Principal: session.Principal{
			SessionID: "privacy-session", UserID: repository.user.ID,
			Role: string(RoleAdmin), Status: string(UserStatusActive),
			MustChangePassword:        credential == session.CredentialTemporary,
			MustAcknowledgePrivacy:    true,
			AuthenticatedCredential:   credential,
			AuthenticatedCredentialID: credentialID,
		},
		CurrentSessionToken:  "privacy-token",
		PrivacyNoticeVersion: repository.currentNotice, Acknowledged: true,
		SourceKind:        "network_source_hmac_sha256_v1",
		SourceFingerprint: "v1:source-digest",
		TraceID:           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}
}

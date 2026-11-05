package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegister_Success_CommitsAtomicMemberWithoutSession(t *testing.T) {
	fixture := newRegistrationFixture(t)

	got, err := fixture.service.Register(fixture.context, validRegistrationRequest())
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got.ID != "00000000-0000-4000-8000-000000000001" ||
		got.Username != "Alice" ||
		got.Role != RoleMember ||
		got.Status != UserStatusActive ||
		!got.CreatedAt.Equal(fixture.now) {
		t.Fatalf("Register() = %#v", got)
	}

	want := []string{
		"user",
		"identifier:username:alice",
		"identifier:email:alice@example.test",
		"identifier:phone:+15550000001",
		"privacy_acknowledgement",
		"audit:identity.register:succeeded",
	}
	if strings.Join(fixture.transaction.committed, "|") != strings.Join(want, "|") {
		t.Fatalf("committed operations = %v, want %v", fixture.transaction.committed, want)
	}
	for _, operation := range fixture.transaction.committed {
		if strings.Contains(operation, "session") {
			t.Fatalf("registration created a session: %v", fixture.transaction.committed)
		}
	}
	if fixture.transaction.commitCalls != 1 {
		t.Fatalf("Commit() calls = %d, want 1", fixture.transaction.commitCalls)
	}
	assertPersistedRegistration(t, fixture)
}

func TestRegister_FailureMatrix_LeavesNoPartialAccount(t *testing.T) {
	tests := append(
		registrationPreTransactionFailures(),
		registrationTransactionalFailures()...,
	)
	tests = append(tests, registrationConflictFailures()...)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertRegistrationFailure(t, test)
		})
	}
}

func TestRegister_PersistenceFailure_RollsBackBeforeIndependentFailureAudit(t *testing.T) {
	for _, test := range registrationPersistenceFailureCases() {
		t.Run(test.name, func(t *testing.T) {
			fixture := newRegistrationFixture(t)
			test.configure(fixture)

			_, err := fixture.service.Register(fixture.context, validRegistrationRequest())
			if got := platform.PublicCode(err); got != test.wantCode {
				t.Fatalf("PublicCode() = %q, want %q; error = %v", got, test.wantCode, err)
			}
			assertIndependentFailureAudit(t, fixture, test.wantCode)
		})
	}
}

func TestRegister_FailureAuditFailure_FailsClosedAndPreservesBusinessCause(t *testing.T) {
	fixture := newRegistrationFixture(t)
	businessErr := errors.New("identity store unavailable")
	auditErr := errors.New("failure audit unavailable")
	fixture.repository.failAt = "user"
	fixture.repository.err = businessErr
	fixture.audit.failOutcome = "failed"
	fixture.audit.err = auditErr

	_, err := fixture.service.Register(fixture.context, validRegistrationRequest())
	if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("PublicCode() = %q; error = %v", got, err)
	}
	if !errors.Is(err, businessErr) || !errors.Is(err, auditErr) {
		t.Fatalf("Register() error = %v, want both failure roots", err)
	}
	if len(fixture.transaction.committed) != 0 ||
		len(fixture.failureAuditTransaction.committed) != 0 {
		t.Fatalf(
			"committed business=%v audit=%v, want none",
			fixture.transaction.committed,
			fixture.failureAuditTransaction.committed,
		)
	}
}

func TestRegister_BeginFailure_AppendsAttemptScopedFailureAudit(t *testing.T) {
	fixture := newRegistrationFixture(t)
	beginErr := errors.New("business transaction unavailable")
	fixture.transactor.beginErrors = []error{beginErr}

	_, err := fixture.service.Register(fixture.context, validRegistrationRequest())
	if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("PublicCode() = %q; error = %v", got, err)
	}
	if !errors.Is(err, beginErr) {
		t.Fatalf("Register() error = %v, want begin root", err)
	}
	if fixture.transaction.rollbackCalls != 0 || len(fixture.transaction.committed) != 0 {
		t.Fatalf("business transaction was used after begin failure")
	}
	wantOperations := []string{"audit:identity.register:failed:platform.database_unavailable"}
	if got := fixture.failureAuditTransaction.committed; strings.Join(got, "|") != strings.Join(wantOperations, "|") {
		t.Fatalf("failure audit committed operations = %v, want %v", got, wantOperations)
	}
	wantTimeline := "begin:failed|begin:failure-audit|failure-audit:commit"
	if got := strings.Join(fixture.timeline.events, "|"); got != wantTimeline {
		t.Fatalf("transaction timeline = %q, want %q", got, wantTimeline)
	}
	event := failedRegistrationAuditEvent(t, fixture)
	assertMinimalFailureAuditEvent(
		t,
		event,
		platform.CodePlatformDatabaseUnavailable,
		"registration_attempt",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
}

type registrationFailureCase struct {
	name          string
	configure     func(*registrationFixture)
	request       func() RegistrationRequest
	wantCode      platform.Code
	wantBeginCall int
}

func registrationPreTransactionFailures() []registrationFailureCase {
	return []registrationFailureCase{
		{
			name: "processing gate closed",
			configure: func(f *registrationFixture) {
				f.gate.err = platform.NewPublicError(
					platform.CodePlatformProcessingGateClosed,
					errors.New("gate closed"),
				)
			},
			request:  validRegistrationRequest,
			wantCode: platform.CodePlatformProcessingGateClosed,
		},
		{
			name: "privacy not acknowledged",
			request: func() RegistrationRequest {
				request := validRegistrationRequest()
				request.PrivacyAcknowledged = false
				return request
			},
			wantCode: platform.CodePrivacyAcknowledgementRequired,
		},
	}
}

func registrationTransactionalFailures() []registrationFailureCase {
	return []registrationFailureCase{
		{
			name: "outdated notice",
			configure: func(f *registrationFixture) {
				f.notice.notice.Version = "notice-v2"
			},
			request:       validRegistrationRequest,
			wantCode:      platform.CodePrivacyNoticeOutdated,
			wantBeginCall: 1,
		},
		{
			name: "password hashing fails",
			configure: func(f *registrationFixture) {
				f.hasher.err = errors.New("hash resource unavailable")
			},
			request:       validRegistrationRequest,
			wantCode:      platform.CodePlatformInternalError,
			wantBeginCall: 1,
		},
		{
			name: "audit append fails",
			configure: func(f *registrationFixture) {
				f.audit.err = errors.New("audit unavailable")
			},
			request:       validRegistrationRequest,
			wantCode:      platform.CodePlatformDatabaseUnavailable,
			wantBeginCall: 2,
		},
	}
}

func registrationConflictFailures() []registrationFailureCase {
	uniqueViolation := &pgconn.PgError{Code: "23505"}
	return []registrationFailureCase{
		{
			name: "username conflict",
			configure: func(f *registrationFixture) {
				f.repository.failAt = "identifier:username"
				f.repository.err = uniqueViolation
			},
			request:       validRegistrationRequest,
			wantCode:      platform.CodeIdentityIdentifierConflict,
			wantBeginCall: 2,
		},
		{
			name: "email conflict",
			configure: func(f *registrationFixture) {
				f.repository.failAt = "identifier:email"
				f.repository.err = uniqueViolation
			},
			request:       validRegistrationRequest,
			wantCode:      platform.CodeIdentityIdentifierConflict,
			wantBeginCall: 2,
		},
		{
			name: "phone conflict",
			configure: func(f *registrationFixture) {
				f.repository.failAt = "identifier:phone"
				f.repository.err = uniqueViolation
			},
			request:       validRegistrationRequest,
			wantCode:      platform.CodeIdentityIdentifierConflict,
			wantBeginCall: 2,
		},
	}
}

type registrationPersistenceFailureCase struct {
	name      string
	configure func(*registrationFixture)
	wantCode  platform.Code
}

func registrationPersistenceFailureCases() []registrationPersistenceFailureCase {
	uniqueViolation := &pgconn.PgError{Code: "23505"}
	return []registrationPersistenceFailureCase{
		{
			name: "ordinary user persistence failure",
			configure: func(f *registrationFixture) {
				f.repository.failAt, f.repository.err = "user", errors.New("store unavailable")
			},
			wantCode: platform.CodePlatformDatabaseUnavailable,
		},
		{
			name: "username unique conflict",
			configure: func(f *registrationFixture) {
				f.repository.failAt, f.repository.err = "identifier:username", uniqueViolation
			},
			wantCode: platform.CodeIdentityIdentifierConflict,
		},
		{
			name: "email unique conflict",
			configure: func(f *registrationFixture) {
				f.repository.failAt, f.repository.err = "identifier:email", uniqueViolation
			},
			wantCode: platform.CodeIdentityIdentifierConflict,
		},
		{
			name: "phone unique conflict",
			configure: func(f *registrationFixture) {
				f.repository.failAt, f.repository.err = "identifier:phone", uniqueViolation
			},
			wantCode: platform.CodeIdentityIdentifierConflict,
		},
		{
			name: "commit failure",
			configure: func(f *registrationFixture) {
				f.transaction.commitErr = pgx.ErrTxCommitRollback
			},
			wantCode: platform.CodePlatformDatabaseUnavailable,
		},
	}
}

func assertIndependentFailureAudit(
	t *testing.T,
	fixture *registrationFixture,
	wantCode platform.Code,
) {
	t.Helper()
	if len(fixture.transaction.committed) != 0 {
		t.Fatalf("business committed operations = %v, want none", fixture.transaction.committed)
	}
	if !fixture.transaction.closed || len(fixture.transaction.pending) != 0 {
		t.Fatalf(
			"business transaction closed=%t pending=%v, want rolled back",
			fixture.transaction.closed,
			fixture.transaction.pending,
		)
	}
	wantOperations := []string{"audit:identity.register:failed:" + string(wantCode)}
	if strings.Join(fixture.failureAuditTransaction.committed, "|") != strings.Join(wantOperations, "|") {
		t.Fatalf(
			"failure audit committed operations = %v, want %v",
			fixture.failureAuditTransaction.committed,
			wantOperations,
		)
	}
	wantTimeline := "begin:business|business:rollback|begin:failure-audit|failure-audit:commit"
	if got := strings.Join(fixture.timeline.events, "|"); got != wantTimeline {
		t.Fatalf("transaction timeline = %q, want %q", got, wantTimeline)
	}
	event := failedRegistrationAuditEvent(t, fixture)
	assertMinimalFailureAuditEvent(
		t,
		event,
		wantCode,
		"user",
		"00000000-0000-4000-8000-000000000001",
	)
}

func failedRegistrationAuditEvent(t *testing.T, fixture *registrationFixture) audit.Event {
	t.Helper()
	for index := range fixture.audit.events {
		if fixture.audit.events[index].Outcome == "failed" {
			return fixture.audit.events[index]
		}
	}
	t.Fatalf("audit events = %#v, want failed registration event", fixture.audit.events)
	return audit.Event{}
}

func assertMinimalFailureAuditEvent(
	t *testing.T,
	event audit.Event,
	wantCode platform.Code,
	targetType string,
	targetID string,
) {
	t.Helper()
	if event.Action != registrationAuditAction ||
		event.Outcome != "failed" ||
		event.ErrorCode == nil || *event.ErrorCode != string(wantCode) ||
		event.TraceID != "01ARZ3NDEKTSV4RRFFQ69G5FAV" ||
		event.ActorKind != audit.ActorAnonymous ||
		event.TargetType != targetType ||
		event.TargetID != targetID {
		t.Fatalf("failure audit event = %#v", event)
	}
	if event.SourceFingerprint != nil || len(event.Metadata) != 0 {
		t.Fatalf("failure audit contains unnecessary context = %#v", event)
	}
}

func assertRegistrationFailure(t *testing.T, test registrationFailureCase) {
	t.Helper()
	fixture := newRegistrationFixture(t)
	if test.configure != nil {
		test.configure(fixture)
	}

	_, err := fixture.service.Register(fixture.context, test.request())
	if err == nil {
		t.Fatal("Register() error = nil")
	}
	if got := platform.PublicCode(err); got != test.wantCode {
		t.Fatalf("PublicCode() = %q, want %q; error = %v", got, test.wantCode, err)
	}
	if fixture.transactor.beginCalls != test.wantBeginCall {
		t.Fatalf("Begin() calls = %d, want %d", fixture.transactor.beginCalls, test.wantBeginCall)
	}
	if len(fixture.transaction.committed) != 0 {
		t.Fatalf("committed operations = %v, want none", fixture.transaction.committed)
	}
	if test.wantBeginCall == 1 && fixture.transaction.rollbackCalls == 0 {
		t.Fatal("Rollback() was not called")
	}
}

func TestRegister_HandlerSuccess_Returns201WithoutSessionCookie(t *testing.T) {
	handler, service, request, recorder := newRegistrationHandlerFixture(t)

	platform.TraceMiddleware(http.HandlerFunc(handler.Register)).ServeHTTP(recorder, request)

	assertRegistrationHandlerResponse(t, recorder, service)
}

func newRegistrationHandlerFixture(
	t *testing.T,
) (*Handler, *registrationServiceStub, *http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	now := time.Date(2026, time.October, 22, 8, 30, 0, 0, time.UTC)
	service := &registrationServiceStub{
		result: RegisteredUser{
			ID:        "00000000-0000-4000-8000-000000000001",
			Username:  "Alice",
			Role:      RoleMember,
			Status:    UserStatusActive,
			CreatedAt: now,
		},
	}
	handler, err := NewHandler(
		service,
		registrationSourceStub{evidence: RegistrationSourceEvidence{
			Kind:        "trusted-source",
			Fingerprint: "source-digest",
		}},
		4096,
	)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users",
		bytes.NewBufferString(`{
			"username":"Alice",
			"email":"Alice@Example.test",
			"phone":"+15550000001",
			"password":"Password123",
			"privacy_notice_version":"notice-v1",
			"privacy_acknowledged":true
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	return handler, service, request, recorder
}

func assertRegistrationHandlerResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	service *registrationServiceStub,
) {
	t.Helper()
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body)
	}
	if cookies := recorder.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("Set-Cookie = %v, want none", cookies)
	}
	if service.request.SourceKind != "trusted-source" ||
		service.request.SourceFingerprint != "source-digest" {
		t.Fatalf("source evidence = %#v", service.request)
	}
	var response registrationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != service.result.ID ||
		response.Username != service.result.Username ||
		response.Role != RoleMember ||
		response.Status != UserStatusActive ||
		response.MustChangePassword ||
		response.CreatedAt != "2026-10-22T08:30:00Z" {
		t.Fatalf("response = %#v", response)
	}
}

func assertPersistedRegistration(t *testing.T, fixture *registrationFixture) {
	t.Helper()
	user := fixture.repository.user
	if user.ID != "00000000-0000-4000-8000-000000000001" ||
		user.Username != "Alice" ||
		user.Email == nil || *user.Email != "Alice@Example.test" ||
		user.Phone == nil || *user.Phone != "+15550000001" ||
		user.PermanentPasswordDigest == nil || *user.PermanentPasswordDigest != "$argon2id$test" ||
		user.Role != RoleMember || user.Status != UserStatusActive ||
		user.CredentialMode != CredentialModePermanent ||
		user.OnboardingPrivacyAcknowledgedVersion == nil ||
		*user.OnboardingPrivacyAcknowledgedVersion != "notice-v1" ||
		user.OnboardingCompletedAt == nil || !user.OnboardingCompletedAt.Equal(fixture.now) {
		t.Fatalf("persisted user = %#v", user)
	}
	acknowledgement := fixture.repository.acknowledgement
	if acknowledgement.ID != "00000000-0000-4000-8000-000000000002" ||
		acknowledgement.UserID != user.ID ||
		acknowledgement.NoticeVersion != "notice-v1" ||
		acknowledgement.SourceKind != "trusted-source" ||
		acknowledgement.SourceFingerprint != "source-digest" {
		t.Fatalf("persisted acknowledgement = %#v", acknowledgement)
	}
	event := fixture.audit.events[0]
	if event.ID != "00000000-0000-4000-8000-000000000003" ||
		event.ActorKind != audit.ActorAnonymous ||
		event.Action != registrationAuditAction ||
		event.TargetType != "user" || event.TargetID != user.ID ||
		event.Outcome != "succeeded" ||
		event.TraceID != "01ARZ3NDEKTSV4RRFFQ69G5FAV" ||
		event.SourceFingerprint == nil || *event.SourceFingerprint != "source-digest" {
		t.Fatalf("persisted audit event = %#v", event)
	}
}

type registrationFixture struct {
	context                 context.Context
	now                     time.Time
	timeline                *registrationTimeline
	transaction             *registrationTxStub
	failureAuditTransaction *registrationTxStub
	transactor              *registrationTransactorStub
	repository              *registrationRepositoryStub
	notice                  *registrationNoticeStub
	gate                    *registrationGateStub
	policy                  registrationPasswordPolicyStub
	hasher                  *registrationHasherStub
	audit                   *registrationAuditStub
	service                 *Service
}

func newRegistrationFixture(t *testing.T) *registrationFixture {
	t.Helper()
	now := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)
	timeline := &registrationTimeline{}
	transaction := &registrationTxStub{name: "business", timeline: timeline}
	failureAuditTransaction := &registrationTxStub{name: "failure-audit", timeline: timeline}
	transactor := &registrationTransactorStub{
		transactions: []*registrationTxStub{transaction, failureAuditTransaction},
		timeline:     timeline,
	}
	repository := &registrationRepositoryStub{}
	notice := &registrationNoticeStub{notice: PrivacyNotice{
		Version:     "notice-v1",
		Content:     "current notice",
		ContentHash: []byte("notice-hash"),
		EffectiveAt: now.Add(-time.Hour),
		Status:      PrivacyNoticeCurrent,
	}}
	gate := &registrationGateStub{}
	hasher := &registrationHasherStub{digest: "$argon2id$test"}
	auditAppender := &registrationAuditStub{}
	service, err := NewService(RegistrationDependencies{
		Transactions:   transactor,
		Repository:     repository,
		Notices:        notice,
		Gate:           gate,
		PasswordPolicy: registrationPasswordPolicyStub{},
		PasswordHasher: hasher,
		Audit:          auditAppender,
		IDs: &registrationIDStub{ids: []string{
			"00000000-0000-4000-8000-000000000001",
			"00000000-0000-4000-8000-000000000002",
			"00000000-0000-4000-8000-000000000003",
			"00000000-0000-4000-8000-000000000004",
		}},
		Clock: registrationClockStub{now: now},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return &registrationFixture{
		context: newRegistrationContext(t), now: now, timeline: timeline, transaction: transaction,
		failureAuditTransaction: failureAuditTransaction, transactor: transactor,
		repository: repository, notice: notice, gate: gate, hasher: hasher,
		audit: auditAppender, service: service,
	}
}

func newRegistrationContext(t *testing.T) context.Context {
	t.Helper()
	ctx, err := platform.ContextWithTraceID(
		context.Background(),
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	return ctx
}

func validRegistrationRequest() RegistrationRequest {
	return RegistrationRequest{
		Username:             " Alice ",
		Email:                " Alice@Example.test ",
		Phone:                " +15550000001 ",
		Password:             "Password123",
		PrivacyNoticeVersion: "notice-v1",
		PrivacyAcknowledged:  true,
		SourceKind:           "trusted-source",
		SourceFingerprint:    "source-digest",
	}
}

type registrationTxStub struct {
	name          string
	timeline      *registrationTimeline
	pending       []string
	committed     []string
	commitErr     error
	commitCalls   int
	rollbackCalls int
	closed        bool
}

type registrationTimeline struct {
	events []string
}

func (tx *registrationTxStub) add(operation string) {
	tx.pending = append(tx.pending, operation)
}

func (tx *registrationTxStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}

func (tx *registrationTxStub) Commit(context.Context) error {
	tx.commitCalls++
	if tx.commitErr != nil {
		if errors.Is(tx.commitErr, pgx.ErrTxCommitRollback) {
			tx.pending = nil
			tx.closed = true
			tx.timeline.events = append(tx.timeline.events, tx.name+":rollback")
		}
		return tx.commitErr
	}
	tx.committed = append(tx.committed, tx.pending...)
	tx.pending = nil
	tx.closed = true
	tx.timeline.events = append(tx.timeline.events, tx.name+":commit")
	return nil
}

func (tx *registrationTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	tx.pending = nil
	tx.timeline.events = append(tx.timeline.events, tx.name+":rollback")
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	return nil
}

func (tx *registrationTxStub) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("not implemented")
}

func (tx *registrationTxStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *registrationTxStub) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *registrationTxStub) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}

func (tx *registrationTxStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not implemented")
}

func (tx *registrationTxStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (tx *registrationTxStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return registrationRowStub{err: errors.New("not implemented")}
}

func (tx *registrationTxStub) Conn() *pgx.Conn {
	return nil
}

type registrationTransactorStub struct {
	transactions []*registrationTxStub
	timeline     *registrationTimeline
	beginErrors  []error
	beginCalls   int
}

func (s *registrationTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	s.beginCalls++
	index := s.beginCalls - 1
	if index < len(s.beginErrors) && s.beginErrors[index] != nil {
		s.timeline.events = append(s.timeline.events, "begin:failed")
		return nil, s.beginErrors[index]
	}
	if index >= len(s.transactions) {
		return nil, errors.New("no registration transaction remains")
	}
	transaction := s.transactions[index]
	s.timeline.events = append(s.timeline.events, "begin:"+transaction.name)
	return transaction, nil
}

type registrationRepositoryStub struct {
	user            User
	acknowledgement PrivacyAcknowledgement
	failAt          string
	err             error
}

func (s *registrationRepositoryStub) InsertUser(
	_ context.Context,
	tx pgx.Tx,
	user User,
) error {
	s.user = user
	return s.record(tx, "user")
}

func (s *registrationRepositoryStub) InsertIdentifier(
	_ context.Context,
	tx pgx.Tx,
	identifier IdentityIdentifier,
) error {
	return s.record(
		tx,
		"identifier:"+string(identifier.Kind)+":"+identifier.NormalizedValue,
	)
}

func (s *registrationRepositoryStub) InsertPrivacyAcknowledgement(
	_ context.Context,
	tx pgx.Tx,
	acknowledgement PrivacyAcknowledgement,
) error {
	s.acknowledgement = acknowledgement
	return s.record(tx, "privacy_acknowledgement")
}

func (s *registrationRepositoryStub) record(tx pgx.Tx, operation string) error {
	failureKey := operation
	if strings.HasPrefix(operation, "identifier:") {
		parts := strings.SplitN(operation, ":", 3)
		failureKey = strings.Join(parts[:2], ":")
	}
	if failureKey == s.failAt {
		return s.err
	}
	tx.(*registrationTxStub).add(operation)
	return nil
}

type registrationNoticeStub struct {
	notice PrivacyNotice
	err    error
}

func (s *registrationNoticeStub) CurrentNotice(
	context.Context,
	pgx.Tx,
) (PrivacyNotice, error) {
	return s.notice, s.err
}

type registrationGateStub struct {
	err error
}

func (s *registrationGateStub) RequireOpen(context.Context) error {
	return s.err
}

type registrationPasswordPolicyStub struct {
	err error
}

func (s registrationPasswordPolicyStub) Validate(string) error {
	return s.err
}

type registrationHasherStub struct {
	digest string
	err    error
}

func (s *registrationHasherStub) Hash(string) (string, error) {
	return s.digest, s.err
}

type registrationAuditStub struct {
	events      []audit.Event
	failOutcome string
	err         error
}

func (s *registrationAuditStub) Append(
	_ context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	if s.err != nil && (s.failOutcome == "" || s.failOutcome == event.Outcome) {
		return s.err
	}
	s.events = append(s.events, event)
	errorCode := ""
	if event.ErrorCode != nil {
		errorCode = ":" + *event.ErrorCode
	}
	tx.(*registrationTxStub).add("audit:" + event.Action + ":" + event.Outcome + errorCode)
	return nil
}

type registrationIDStub struct {
	ids []string
}

func (s *registrationIDStub) NewID() (string, error) {
	if len(s.ids) == 0 {
		return "", errors.New("no registration IDs remain")
	}
	id := s.ids[0]
	s.ids = s.ids[1:]
	return id, nil
}

type registrationClockStub struct {
	now time.Time
}

func (s registrationClockStub) Now() time.Time {
	return s.now
}

type registrationServiceStub struct {
	request RegistrationRequest
	result  RegisteredUser
	err     error
}

func (s *registrationServiceStub) Register(
	_ context.Context,
	request RegistrationRequest,
) (RegisteredUser, error) {
	s.request = request
	return s.result, s.err
}

type registrationSourceStub struct {
	evidence RegistrationSourceEvidence
	err      error
}

func (s registrationSourceStub) Evidence(*http.Request) (RegistrationSourceEvidence, error) {
	return s.evidence, s.err
}

type registrationRowStub struct {
	err error
}

func (r registrationRowStub) Scan(...any) error {
	return r.err
}

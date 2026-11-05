package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	canonicalDeleteActorID   = "10000000-0000-4000-8000-000000000001"
	canonicalDeleteTargetID  = "10000000-0000-4000-8000-000000000002"
	canonicalDeleteRunID     = "recovery-run-one"
	canonicalDeleteBackupRef = "backup-one"
	canonicalDeleteTraceID   = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

var canonicalDeleteNow = time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)

func canonicalDeleteContext(t *testing.T) context.Context {
	t.Helper()
	ctx, err := platform.ContextWithTraceID(context.Background(), canonicalDeleteTraceID)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	return ctx
}

func TestCanonicalDelete_FailureAtEveryOwnedStepRollsBack(t *testing.T) {
	for _, step := range []string{
		"user", "identifier", "session", "credential", "article", "audit",
	} {
		t.Run(step, func(t *testing.T) {
			fixture := newCanonicalDeleteFixture(t, 2)
			fixture.store.failStep = step
			before := fixture.store.state.clone()

			_, err := fixture.coordinator.CanonicalDelete(
				canonicalDeleteContext(t),
				fixture.dailyRequest(),
			)
			if err == nil {
				t.Fatal("CanonicalDelete() error = nil")
			}
			if !reflect.DeepEqual(fixture.store.state, before) {
				t.Fatalf("committed state changed after %s failure", step)
			}
			if fixture.store.lastTx == nil {
				t.Fatalf("canonical deletion did not begin a transaction for %s failure", step)
			}
			if fixture.store.lastTx.commits != 0 || fixture.store.lastTx.rollbacks != 1 {
				t.Fatalf(
					"commit/rollback after %s failure = %d/%d, want 0/1",
					step,
					fixture.store.lastTx.commits,
					fixture.store.lastTx.rollbacks,
				)
			}
		})
	}
}

func TestIdentifierLifecycle_ZeroArticlesReleasesOnlyEmail(t *testing.T) {
	fixture := newCanonicalDeleteFixture(t, 0)

	result, err := fixture.coordinator.CanonicalDelete(
		canonicalDeleteContext(t),
		fixture.dailyRequest(),
	)
	if err != nil {
		t.Fatalf("CanonicalDelete() error = %v", err)
	}
	target := fixture.store.state.users[canonicalDeleteTargetID]
	if target.Status != UserStatusDeleted || target.DeletedAt == nil ||
		target.Email != nil || target.Phone != nil ||
		target.PermanentPasswordDigest != nil || target.RetainedEmail != nil {
		t.Fatalf("deleted user = %+v", target)
	}
	if _, exists := fixture.store.state.identifiers[IdentifierEmail]; exists {
		t.Fatal("email identifier remains after zero-article deletion")
	}
	if got := fixture.store.state.identifiers[IdentifierUsername]; got != IdentifierReserved {
		t.Fatalf("username lifecycle = %q, want reserved", got)
	}
	if got := fixture.store.state.identifiers[IdentifierPhone]; got != IdentifierReserved {
		t.Fatalf("phone lifecycle = %q, want reserved", got)
	}
	if !result.EmailIdentifierFreed || result.PendingOrphanCount != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestIdentifierLifecycle_MultipleAndViolatingArticlesRetainEmail(t *testing.T) {
	fixture := newCanonicalDeleteFixture(t, 2)
	fixture.store.state.articles[1].violating = true

	result, err := fixture.coordinator.CanonicalDelete(
		canonicalDeleteContext(t),
		fixture.dailyRequest(),
	)
	if err != nil {
		t.Fatalf("CanonicalDelete() error = %v", err)
	}
	target := fixture.store.state.users[canonicalDeleteTargetID]
	if target.RetainedEmail == nil || *target.RetainedEmail != "member@example.test" {
		t.Fatalf("retained email = %v", target.RetainedEmail)
	}
	if got := fixture.store.state.identifiers[IdentifierEmail]; got != IdentifierReserved {
		t.Fatalf("email lifecycle = %q, want reserved", got)
	}
	for index, row := range fixture.store.state.articles {
		if row.authorID != nil || !row.held || row.orphanedFrom != canonicalDeleteTargetID {
			t.Fatalf("article[%d] = %+v", index, row)
		}
	}
	if !fixture.store.state.articles[1].violating {
		t.Fatal("canonical deletion cleared article violation")
	}
	if result.PendingOrphanCount != 2 || result.EmailIdentifierFreed {
		t.Fatalf("result = %+v", result)
	}
}

func TestCanonicalDelete_RecoveryKeptDeletedReplaysSameCoordinatorAndCompletesReview(t *testing.T) {
	fixture := newCanonicalDeleteFixture(t, 1)
	state := &fixture.store.state
	target := state.users[canonicalDeleteTargetID]
	target.Status = UserStatusDeleted
	target.DeletedAt = canonicalDeleteTimePointer(canonicalDeleteNow.Add(-time.Hour))
	target.RetainedEmail = target.Email
	target.Email = nil
	target.Phone = nil
	target.PermanentPasswordDigest = nil
	state.users[canonicalDeleteTargetID] = target
	state.identifiers[IdentifierUsername] = IdentifierReserved
	state.identifiers[IdentifierEmail] = IdentifierReserved
	state.identifiers[IdentifierPhone] = IdentifierReserved
	state.sessions = 0
	state.credentials = 0
	state.articles[0].authorID = nil
	state.articles[0].held = true
	state.articles[0].orphanedFrom = canonicalDeleteTargetID

	request := fixture.dailyRequest()
	request.Mode = CanonicalDeleteRecoveryKept
	request.Recovery = fixture.recoveryRequest(t)

	result, err := fixture.coordinator.CanonicalDelete(canonicalDeleteContext(t), request)
	if err != nil {
		t.Fatalf("CanonicalDelete() error = %v", err)
	}
	review := fixture.store.state.review
	if review.Decision != RecoveryDecisionKeptDeleted ||
		review.ReviewedBy == nil || *review.ReviewedBy != canonicalDeleteActorID ||
		review.ReviewedAt == nil {
		t.Fatalf("recovery review = %+v", review)
	}
	if result.AlreadyOrphanedCount != 1 || result.PendingOrphanCount != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestIdentifierLifecycle_EmailReleaseFailureRollsBack(t *testing.T) {
	fixture := newCanonicalDeleteFixture(t, 0)
	fixture.store.failStep = "identifier_release"
	before := fixture.store.state.clone()

	_, err := fixture.coordinator.CanonicalDelete(
		canonicalDeleteContext(t),
		fixture.dailyRequest(),
	)
	if err == nil {
		t.Fatal("CanonicalDelete() error = nil")
	}
	if !reflect.DeepEqual(fixture.store.state, before) {
		t.Fatal("email release failure committed partial deletion")
	}
}

func TestCanonicalDelete_RecoveryReviewFailureRollsBack(t *testing.T) {
	fixture := newCanonicalDeleteFixture(t, 1)
	fixture.store.failStep = "review"
	before := fixture.store.state.clone()
	request := fixture.dailyRequest()
	request.Mode = CanonicalDeleteRecoveryKept
	request.Recovery = fixture.recoveryRequest(t)

	_, err := fixture.coordinator.CanonicalDelete(canonicalDeleteContext(t), request)
	if err == nil {
		t.Fatal("CanonicalDelete() error = nil")
	}
	if !reflect.DeepEqual(fixture.store.state, before) {
		t.Fatal("review failure committed partial deletion")
	}
}

func TestCanonicalDelete_AuthorizationAndTargetProtection(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*canonicalDeleteFixture, *CanonicalDeleteRequest)
		wantCode platform.Code
	}{
		{
			name: "ordinary administrator cannot delete administrator",
			mutate: func(f *canonicalDeleteFixture, request *CanonicalDeleteRequest) {
				actor := f.store.state.users[canonicalDeleteActorID]
				actor.IsInitialAdmin = false
				onboarded := canonicalDeleteNow.Add(-time.Hour)
				actor.OnboardingCompletedAt = &onboarded
				f.store.state.users[canonicalDeleteActorID] = actor
				target := f.store.state.users[canonicalDeleteTargetID]
				target.Role = RoleAdmin
				f.store.state.users[canonicalDeleteTargetID] = target
				request.Actor.IsInitialAdmin = false
				request.TargetRole = RoleAdmin
			},
			wantCode: platform.CodeAuthForbidden,
		},
		{
			name: "initial administrator target is protected",
			mutate: func(f *canonicalDeleteFixture, request *CanonicalDeleteRequest) {
				target := f.store.state.users[canonicalDeleteTargetID]
				target.Role = RoleAdmin
				target.IsInitialAdmin = true
				f.store.state.users[canonicalDeleteTargetID] = target
				request.TargetRole = RoleAdmin
			},
			wantCode: platform.CodeIdentityInitialAdminProtected,
		},
		{
			name: "missing target is hidden",
			mutate: func(f *canonicalDeleteFixture, _ *CanonicalDeleteRequest) {
				delete(f.store.state.users, canonicalDeleteTargetID)
			},
			wantCode: platform.CodeIdentityUserNotFound,
		},
		{
			name: "daily replay of deleted target is invalid",
			mutate: func(f *canonicalDeleteFixture, _ *CanonicalDeleteRequest) {
				target := f.store.state.users[canonicalDeleteTargetID]
				target.Status = UserStatusDeleted
				target.DeletedAt = canonicalDeleteTimePointer(canonicalDeleteNow)
				f.store.state.users[canonicalDeleteTargetID] = target
			},
			wantCode: platform.CodeIdentityInvalidState,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCanonicalDeleteFixture(t, 1)
			request := fixture.dailyRequest()
			test.mutate(&fixture, &request)
			before := fixture.store.state.clone()

			_, err := fixture.coordinator.CanonicalDelete(canonicalDeleteContext(t), request)
			if platform.PublicCode(err) != test.wantCode {
				t.Fatalf("error code = %q, want %q, err=%v", platform.PublicCode(err), test.wantCode, err)
			}
			if !reflect.DeepEqual(fixture.store.state, before) {
				t.Fatal("rejected canonical deletion changed committed state")
			}
		})
	}
}

func TestCanonicalDeleteHandler_ReasonAndNoContent(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		call       func(*Handler, http.ResponseWriter, *http.Request)
		targetRole Role
		reason     string
		wantReason AccountDeletionReason
	}{
		{
			name: "member default governance",
			path: "/api/v1/admin/users/" + canonicalDeleteTargetID,
			call: (*Handler).DeleteUser, targetRole: RoleMember,
			wantReason: AccountDeletionGovernance,
		},
		{
			name: "administrator privacy withdrawal",
			path: "/api/v1/admin/administrators/" + canonicalDeleteTargetID,
			call: (*Handler).DeleteAdministrator, targetRole: RoleAdmin,
			reason:     string(AccountDeletionPrivacyWithdrawal),
			wantReason: AccountDeletionPrivacyWithdrawal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &canonicalDeleteServiceStub{}
			handler, err := NewCanonicalDeleteHandler(service)
			if err != nil {
				t.Fatalf("NewCanonicalDeleteHandler() error = %v", err)
			}
			request := httptest.NewRequest(http.MethodDelete, test.path, nil)
			if test.reason != "" {
				request.Header.Set("X-Account-Deletion-Reason", test.reason)
			}
			response := serveCanonicalDeleteRequest(t, handler, request, test.call)
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Fatalf("status/body = %d/%q, want 204/empty", response.Code, response.Body.String())
			}
			if service.calls != 1 ||
				service.request.TargetID != canonicalDeleteTargetID ||
				service.request.TargetRole != test.targetRole ||
				service.request.Mode != CanonicalDeleteDailyGovernance ||
				service.request.Reason != test.wantReason {
				t.Fatalf("service request = %+v, calls=%d", service.request, service.calls)
			}
		})
	}
}

func TestCanonicalDeleteHandler_RejectsInvalidReasonAndMethod(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		reason     string
		wantStatus int
		wantCode   platform.Code
	}{
		{
			name: "invalid reason", method: http.MethodDelete, reason: "other",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   platform.CodeIdentityValidationFailed,
		},
		{
			name: "wrong method", method: http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   platform.CodeRequestMethodNotAllowed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &canonicalDeleteServiceStub{}
			handler, err := NewCanonicalDeleteHandler(service)
			if err != nil {
				t.Fatalf("NewCanonicalDeleteHandler() error = %v", err)
			}
			request := httptest.NewRequest(
				test.method,
				"/api/v1/admin/users/"+canonicalDeleteTargetID,
				nil,
			)
			if test.reason != "" {
				request.Header.Set("X-Account-Deletion-Reason", test.reason)
			}
			response := serveCanonicalDeleteRequest(t, handler, request, (*Handler).DeleteUser)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode || service.calls != 0 {
				t.Fatalf("envelope/service = %+v/%d", envelope, service.calls)
			}
		})
	}
}

func serveCanonicalDeleteRequest(
	t *testing.T,
	handler *Handler,
	request *http.Request,
	call func(*Handler, http.ResponseWriter, *http.Request),
) *httptest.ResponseRecorder {
	t.Helper()
	ctx, err := platform.ContextWithTraceID(
		request.Context(),
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request = request.WithContext(ctx)
	request.AddCookie(&http.Cookie{
		Name: platform.SessionCookieName, Value: "canonical-delete-session",
	})
	request.Header.Set(platform.CSRFHeaderName, "canonical-delete-csrf")
	response := httptest.NewRecorder()
	authorizer, err := platform.NewHTTPAuthorizer(
		canonicalDeletePrincipalResolver{principal: session.Principal{
			SessionID: "canonical-delete-session-id",
			UserID:    canonicalDeleteActorID, Role: string(RoleAdmin),
			IsInitialAdmin: true, Status: string(UserStatusActive),
		}},
		func() time.Time { return canonicalDeleteNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			call(handler, w, r)
		}),
	).ServeHTTP(response, request)
	return response
}

type canonicalDeleteFixture struct {
	coordinator *CanonicalDeleteCoordinator
	store       *canonicalDeleteStore
}

func newCanonicalDeleteFixture(t *testing.T, articleCount int) canonicalDeleteFixture {
	t.Helper()
	email := "member@example.test"
	phone := "+15555550100"
	digest := "permanent-password-digest"
	onboardedAt := canonicalDeleteNow.Add(-24 * time.Hour)
	store := &canonicalDeleteStore{
		state: canonicalDeleteState{
			users: map[string]User{
				canonicalDeleteActorID: {
					ID: canonicalDeleteActorID, Role: RoleAdmin, IsInitialAdmin: true,
					Status: UserStatusActive, CredentialMode: CredentialModePermanent,
				},
				canonicalDeleteTargetID: {
					ID: canonicalDeleteTargetID, Username: "member",
					Email: &email, Phone: &phone, PermanentPasswordDigest: &digest,
					Role: RoleMember, Status: UserStatusActive,
					CredentialMode:        CredentialModePermanent,
					OnboardingCompletedAt: &onboardedAt,
				},
			},
			identifiers: map[IdentifierKind]IdentifierLifecycle{
				IdentifierUsername: IdentifierActive,
				IdentifierEmail:    IdentifierActive,
				IdentifierPhone:    IdentifierActive,
			},
			identifierValues: map[IdentifierKind]string{
				IdentifierUsername: "member",
				IdentifierEmail:    email,
				IdentifierPhone:    phone,
			},
			sessions:    2,
			credentials: 2,
			review: RecoveryAccountReview{
				RecoveryRunID: canonicalDeleteRunID,
				UserID:        canonicalDeleteTargetID,
				BackupRole:    RoleMember, BackupStatus: UserStatusDeleted,
				Decision: RecoveryDecisionPending,
			},
		},
		run: recovery.RecoveryRun{
			RunID:     canonicalDeleteRunID,
			BackupRef: canonicalDeleteBackupRef,
			Outcome:   recovery.RecoveryOutcomeRunning,
		},
		gateOpen: true,
	}
	for range articleCount {
		authorID := canonicalDeleteTargetID
		store.state.articles = append(store.state.articles, canonicalDeleteArticle{
			authorID: &authorID,
		})
	}
	coordinator, err := NewCanonicalDeleteService(CanonicalDeleteDependencies{
		Transactions: store,
		Repository:   store,
		Articles:     store,
		Audit:        store,
		IDs:          &canonicalDeleteIDs{},
		Clock:        canonicalDeleteClock{},
		RecoveryRuns: store,
		RecoveryGate: store,
	})
	if err != nil {
		t.Fatalf("NewCanonicalDeleteService() error = %v", err)
	}
	return canonicalDeleteFixture{coordinator: coordinator, store: store}
}

func (f canonicalDeleteFixture) dailyRequest() CanonicalDeleteRequest {
	return CanonicalDeleteRequest{
		Actor: session.Principal{
			UserID: canonicalDeleteActorID, Role: string(RoleAdmin),
			IsInitialAdmin: true, Status: string(UserStatusActive),
		},
		TargetID: canonicalDeleteTargetID, TargetRole: RoleMember,
		Mode: CanonicalDeleteDailyGovernance, Reason: AccountDeletionGovernance,
		TraceID: canonicalDeleteTraceID,
	}
}

func (f canonicalDeleteFixture) recoveryRequest(t *testing.T) *RecoveryKeptDeletedRequest {
	t.Helper()
	capability, err := NewRecoveryDeleteCapability(
		canonicalDeleteRunID,
		canonicalDeleteBackupRef,
	)
	if err != nil {
		t.Fatalf("NewRecoveryDeleteCapability() error = %v", err)
	}
	return &RecoveryKeptDeletedRequest{Capability: capability}
}

type canonicalDeleteState struct {
	users            map[string]User
	identifiers      map[IdentifierKind]IdentifierLifecycle
	identifierValues map[IdentifierKind]string
	sessions         int
	credentials      int
	articles         []canonicalDeleteArticle
	audits           []audit.Event
	review           RecoveryAccountReview
}

func (s canonicalDeleteState) clone() canonicalDeleteState {
	clone := s
	clone.users = make(map[string]User, len(s.users))
	for id, user := range s.users {
		clone.users[id] = cloneCanonicalDeleteUser(user)
	}
	clone.identifiers = make(map[IdentifierKind]IdentifierLifecycle, len(s.identifiers))
	for kind, lifecycle := range s.identifiers {
		clone.identifiers[kind] = lifecycle
	}
	clone.identifierValues = make(map[IdentifierKind]string, len(s.identifierValues))
	for kind, value := range s.identifierValues {
		clone.identifierValues[kind] = value
	}
	clone.articles = append([]canonicalDeleteArticle(nil), s.articles...)
	for index := range clone.articles {
		if s.articles[index].authorID != nil {
			value := *s.articles[index].authorID
			clone.articles[index].authorID = &value
		}
	}
	clone.audits = append([]audit.Event(nil), s.audits...)
	clone.review = cloneCanonicalDeleteReview(s.review)
	return clone
}

func cloneCanonicalDeleteUser(user User) User {
	clone := user
	clone.Email = cloneCanonicalDeleteStringPointer(user.Email)
	clone.Phone = cloneCanonicalDeleteStringPointer(user.Phone)
	clone.PermanentPasswordDigest = cloneCanonicalDeleteStringPointer(
		user.PermanentPasswordDigest,
	)
	clone.RetainedEmail = cloneCanonicalDeleteStringPointer(user.RetainedEmail)
	if user.DeletedAt != nil {
		value := *user.DeletedAt
		clone.DeletedAt = &value
	}
	return clone
}

func cloneCanonicalDeleteReview(review RecoveryAccountReview) RecoveryAccountReview {
	clone := review
	clone.EvidenceRef = cloneCanonicalDeleteStringPointer(review.EvidenceRef)
	clone.ReviewedBy = cloneCanonicalDeleteStringPointer(review.ReviewedBy)
	if review.ReviewedAt != nil {
		value := *review.ReviewedAt
		clone.ReviewedAt = &value
	}
	return clone
}

func cloneCanonicalDeleteStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func canonicalDeleteTimePointer(value time.Time) *time.Time {
	return &value
}

type canonicalDeleteArticle struct {
	authorID      *string
	orphanedFrom  string
	held          bool
	violating     bool
	deletionState string
}

type canonicalDeleteStore struct {
	state    canonicalDeleteState
	failStep string
	lastTx   *canonicalDeleteTx
	run      recovery.RecoveryRun
	gateOpen bool
}

func (s *canonicalDeleteStore) LockRecoveryRunForDeletion(
	_ context.Context,
	tx pgx.Tx,
	runID string,
) (recovery.RecoveryRun, error) {
	if s.failStep == "recovery_lock" {
		return recovery.RecoveryRun{}, errors.New("injected recovery lock failure")
	}
	if s.run.RunID == "" || s.run.RunID != runID {
		return recovery.RecoveryRun{}, recovery.ErrRunNotFound
	}
	_ = tx
	return s.run, nil
}

func (s *canonicalDeleteStore) CheckRecoveryOpenGate(
	context.Context,
	pgx.Tx,
) error {
	if s.failStep == "recovery_gate" {
		return errors.New("injected recovery gate failure")
	}
	if !s.gateOpen {
		return errors.New("recovery readiness gate is closed")
	}
	return nil
}

func (s *canonicalDeleteStore) Begin(context.Context) (pgx.Tx, error) {
	tx := &canonicalDeleteTx{working: s.state.clone(), owner: s}
	s.lastTx = tx
	return tx, nil
}

func (s *canonicalDeleteStore) LockUser(
	_ context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	user, exists := canonicalDeleteWorking(tx).users[userID]
	if !exists {
		return User{}, pgx.ErrNoRows
	}
	return cloneCanonicalDeleteUser(user), nil
}

func (s *canonicalDeleteStore) LockIdentifiers(
	_ context.Context,
	tx pgx.Tx,
	userID string,
) ([]IdentityIdentifier, error) {
	if s.failStep == "identifier_lock" {
		return nil, errors.New("injected identifier lock failure")
	}
	state := canonicalDeleteWorking(tx)
	var identifiers []IdentityIdentifier
	for kind, lifecycle := range state.identifiers {
		identifier := IdentityIdentifier{
			UserID:          userID,
			Kind:            kind,
			Lifecycle:       lifecycle,
			NormalizedValue: state.identifierValues[kind],
		}
		identifiers = append(identifiers, identifier)
	}
	return identifiers, nil
}

func (s *canonicalDeleteStore) MarkUserDeleted(
	_ context.Context,
	tx pgx.Tx,
	userID string,
	now time.Time,
) error {
	if s.failStep == "user" {
		return errors.New("injected user failure")
	}
	state := canonicalDeleteWorking(tx)
	user := state.users[userID]
	if user.RetainedEmail == nil {
		user.RetainedEmail = cloneCanonicalDeleteStringPointer(user.Email)
	}
	user.Email = nil
	user.Phone = nil
	user.PermanentPasswordDigest = nil
	user.Status = UserStatusDeleted
	if user.DeletedAt == nil {
		user.DeletedAt = canonicalDeleteTimePointer(now)
	}
	user.UpdatedAt = now
	state.users[userID] = user
	return nil
}

func (s *canonicalDeleteStore) ReserveIdentifiers(
	_ context.Context,
	tx pgx.Tx,
	_ string,
) error {
	if s.failStep == "identifier" {
		return errors.New("injected identifier failure")
	}
	for kind := range canonicalDeleteWorking(tx).identifiers {
		canonicalDeleteWorking(tx).identifiers[kind] = IdentifierReserved
	}
	return nil
}

func (s *canonicalDeleteStore) DeleteSessions(
	_ context.Context,
	tx pgx.Tx,
	_ string,
) error {
	if s.failStep == "session" {
		return errors.New("injected session failure")
	}
	canonicalDeleteWorking(tx).sessions = 0
	return nil
}

func (s *canonicalDeleteStore) InvalidateTemporaryCredentials(
	_ context.Context,
	tx pgx.Tx,
	_ string,
	_ time.Time,
) error {
	if s.failStep == "credential" {
		return errors.New("injected credential failure")
	}
	canonicalDeleteWorking(tx).credentials = 0
	return nil
}

func (s *canonicalDeleteStore) ReleaseEmailIdentifier(
	_ context.Context,
	tx pgx.Tx,
	_ string,
	_ time.Time,
) (bool, error) {
	if s.failStep == "identifier_release" {
		return false, errors.New("injected identifier release failure")
	}
	delete(canonicalDeleteWorking(tx).identifiers, IdentifierEmail)
	user := canonicalDeleteWorking(tx).users[canonicalDeleteTargetID]
	user.RetainedEmail = nil
	canonicalDeleteWorking(tx).users[canonicalDeleteTargetID] = user
	return true, nil
}

func (s *canonicalDeleteStore) LockRecoveryAccountReview(
	_ context.Context,
	tx pgx.Tx,
	runID string,
	userID string,
) (RecoveryAccountReview, error) {
	review := canonicalDeleteWorking(tx).review
	if review.RecoveryRunID != runID || review.UserID != userID {
		return RecoveryAccountReview{}, pgx.ErrNoRows
	}
	return cloneCanonicalDeleteReview(review), nil
}

func (s *canonicalDeleteStore) CompleteRecoveryAccountReview(
	_ context.Context,
	tx pgx.Tx,
	_ string,
	_ string,
	actorID string,
	evidenceRef *string,
	now time.Time,
) error {
	if s.failStep == "review" {
		return errors.New("injected review failure")
	}
	state := canonicalDeleteWorking(tx)
	state.review.Decision = RecoveryDecisionKeptDeleted
	state.review.EvidenceRef = cloneCanonicalDeleteStringPointer(evidenceRef)
	state.review.ReviewedBy = cloneCanonicalDeleteStringPointer(&actorID)
	state.review.ReviewedAt = canonicalDeleteTimePointer(now)
	return nil
}

func (s *canonicalDeleteStore) OrphanArticlesCanonical(
	_ context.Context,
	tx pgx.Tx,
	deletedUserID string,
) (CanonicalDeleteArticleResult, error) {
	if s.failStep == "article" {
		return CanonicalDeleteArticleResult{}, errors.New("injected article failure")
	}
	state := canonicalDeleteWorking(tx)
	var result CanonicalDeleteArticleResult
	for index := range state.articles {
		row := &state.articles[index]
		if row.deletionState == "pending" {
			result.PendingOrphanCount++
			continue
		}
		if row.authorID != nil && *row.authorID == deletedUserID {
			row.authorID = nil
			row.orphanedFrom = deletedUserID
			row.held = true
			result.NewlyOrphanedCount++
			result.PendingOrphanCount++
			continue
		}
		if row.authorID == nil && row.orphanedFrom == deletedUserID && row.held {
			result.AlreadyOrphanedCount++
			result.PendingOrphanCount++
		}
	}
	return result, nil
}

func (s *canonicalDeleteStore) Append(
	_ context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	if s.failStep == "audit" {
		return errors.New("injected audit failure")
	}
	state := canonicalDeleteWorking(tx)
	state.audits = append(state.audits, event)
	return nil
}

type canonicalDeleteTx struct {
	pgx.Tx
	working   canonicalDeleteState
	owner     *canonicalDeleteStore
	commits   int
	rollbacks int
	closed    bool
}

func (tx *canonicalDeleteTx) Commit(context.Context) error {
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.owner.state = tx.working.clone()
	tx.commits++
	tx.closed = true
	return nil
}

func (tx *canonicalDeleteTx) Rollback(context.Context) error {
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.rollbacks++
	tx.closed = true
	return nil
}

func canonicalDeleteWorking(tx pgx.Tx) *canonicalDeleteState {
	return &tx.(*canonicalDeleteTx).working
}

type canonicalDeleteIDs struct {
	next int
}

func (g *canonicalDeleteIDs) NewID() (string, error) {
	g.next++
	if g.next == 1 {
		return "20000000-0000-4000-8000-000000000001", nil
	}
	return "20000000-0000-4000-8000-000000000002", nil
}

type canonicalDeleteClock struct{}

func (canonicalDeleteClock) Now() time.Time {
	return canonicalDeleteNow
}

type canonicalDeleteServiceStub struct {
	request CanonicalDeleteRequest
	result  CanonicalDeleteResult
	err     error
	calls   int
}

func (s *canonicalDeleteServiceStub) CanonicalDelete(
	_ context.Context,
	request CanonicalDeleteRequest,
) (CanonicalDeleteResult, error) {
	s.calls++
	s.request = request
	return s.result, s.err
}

type canonicalDeletePrincipalResolver struct {
	principal session.Principal
}

func (r canonicalDeletePrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return r.principal, nil
}

func TestCanonicalDeleteIdentifierTombstoneFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*canonicalDeleteStore)
	}{
		{
			name: "missing phone row",
			mutate: func(store *canonicalDeleteStore) {
				delete(store.state.identifiers, IdentifierPhone)
			},
		},
		{
			name: "phone value drift",
			mutate: func(store *canonicalDeleteStore) {
				store.state.identifierValues[IdentifierPhone] = "+15555559999"
			},
		},
		{
			name: "orphan email row",
			mutate: func(store *canonicalDeleteStore) {
				target := store.state.users[canonicalDeleteTargetID]
				target.Email = nil
				store.state.users[canonicalDeleteTargetID] = target
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCanonicalDeleteFixture(t, 1)
			test.mutate(fixture.store)
			before := fixture.store.state.clone()

			_, err := fixture.coordinator.CanonicalDelete(
				canonicalDeleteContext(t),
				fixture.dailyRequest(),
			)
			if platform.PublicCode(err) != platform.CodeIdentityInvalidState {
				t.Fatalf("error code = %q, want invalid_state, err=%v", platform.PublicCode(err), err)
			}
			if !reflect.DeepEqual(fixture.store.state, before) {
				t.Fatal("identifier tombstone failure committed partial deletion")
			}
			if fixture.store.lastTx == nil || fixture.store.lastTx.commits != 0 {
				t.Fatal("identifier tombstone failure did not roll back")
			}
		})
	}
}

func TestCanonicalDeleteRecoveryAuthorizationFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*canonicalDeleteFixture, *CanonicalDeleteRequest)
	}{
		{
			name: "missing capability",
			mutate: func(_ *canonicalDeleteFixture, request *CanonicalDeleteRequest) {
				request.Recovery = &RecoveryKeptDeletedRequest{}
			},
		},
		{
			name: "unknown recovery run",
			mutate: func(f *canonicalDeleteFixture, _ *CanonicalDeleteRequest) {
				f.store.run = recovery.RecoveryRun{}
			},
		},
		{
			name: "run not in running outcome",
			mutate: func(f *canonicalDeleteFixture, _ *CanonicalDeleteRequest) {
				f.store.run.Outcome = recovery.RecoveryOutcomeIdentityReady
			},
		},
		{
			name: "backup reference mismatch",
			mutate: func(f *canonicalDeleteFixture, _ *CanonicalDeleteRequest) {
				f.store.run.BackupRef = "other-backup"
			},
		},
		{
			name: "maintenance gate closed",
			mutate: func(f *canonicalDeleteFixture, _ *CanonicalDeleteRequest) {
				f.store.gateOpen = false
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCanonicalDeleteFixture(t, 1)
			target := fixture.store.state.users[canonicalDeleteTargetID]
			target.Status = UserStatusDeleted
			target.DeletedAt = canonicalDeleteTimePointer(canonicalDeleteNow.Add(-time.Hour))
			target.RetainedEmail = target.Email
			target.Email = nil
			target.Phone = nil
			target.PermanentPasswordDigest = nil
			fixture.store.state.users[canonicalDeleteTargetID] = target
			fixture.store.state.identifiers[IdentifierUsername] = IdentifierReserved
			fixture.store.state.identifiers[IdentifierEmail] = IdentifierReserved
			fixture.store.state.identifiers[IdentifierPhone] = IdentifierReserved
			fixture.store.state.sessions = 0
			fixture.store.state.credentials = 0

			request := fixture.dailyRequest()
			request.Mode = CanonicalDeleteRecoveryKept
			request.Recovery = fixture.recoveryRequest(t)
			test.mutate(&fixture, &request)
			before := fixture.store.state.clone()

			_, err := fixture.coordinator.CanonicalDelete(canonicalDeleteContext(t), request)
			code := platform.PublicCode(err)
			if code != platform.CodeIdentityInvalidState && code != platform.CodeAuthForbidden {
				t.Fatalf("error code = %q, want fail-closed, err=%v", code, err)
			}
			if !reflect.DeepEqual(fixture.store.state, before) {
				t.Fatal("recovery authorization failure committed partial deletion")
			}
			if fixture.store.state.review.Decision != RecoveryDecisionPending {
				t.Fatal("recovery authorization failure completed the pending review")
			}
		})
	}
}

func TestCanonicalDeleteTraceAuthorityFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		ctx    func(*testing.T) context.Context
		mutate func(*CanonicalDeleteRequest)
	}{
		{
			name: "context trace missing",
			ctx:  func(*testing.T) context.Context { return context.Background() },
		},
		{
			name: "explicit trace mismatch",
			ctx:  canonicalDeleteContext,
			mutate: func(request *CanonicalDeleteRequest) {
				request.TraceID = "01ARZ3NDEKTSV4RRFFQ69G5FAW"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCanonicalDeleteFixture(t, 1)
			request := fixture.dailyRequest()
			if test.mutate != nil {
				test.mutate(&request)
			}
			before := fixture.store.state.clone()

			_, err := fixture.coordinator.CanonicalDelete(test.ctx(t), request)
			if err == nil {
				t.Fatal("trace authority failure allowed deletion")
			}
			if !reflect.DeepEqual(fixture.store.state, before) {
				t.Fatal("trace authority failure committed partial deletion")
			}
		})
	}
}

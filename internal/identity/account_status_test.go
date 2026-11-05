package identity

import (
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
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

func TestAccountStatus_TargetStateTransitionsAndIdempotency(t *testing.T) {
	now := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		current     UserStatus
		target      UserStatus
		wantChanged bool
		wantUpdate  int
		wantRevoke  int
	}{
		{name: "suspend active", current: UserStatusActive, target: UserStatusSuspended, wantChanged: true, wantUpdate: 1, wantRevoke: 1},
		{name: "repeat suspend", current: UserStatusSuspended, target: UserStatusSuspended},
		{name: "restore suspended", current: UserStatusSuspended, target: UserStatusActive, wantChanged: true, wantUpdate: 1},
		{name: "repeat restore", current: UserStatusActive, target: UserStatusActive},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &accountStatusTxStub{}
			repository := &accountStatusRepositoryStub{
				users: []User{
					accountStatusActor(false),
					{ID: accountStatusTargetID, Role: RoleMember, Status: test.current},
				},
			}
			auditor := &accountStatusAuditStub{}
			service := newAccountStatusServiceForTest(t, tx, repository, auditor, now)

			result, err := service.SetAccountStatus(context.Background(), AccountStatusRequest{
				Actor:      accountStatusPrincipal(false),
				TargetID:   accountStatusTargetID,
				TargetRole: RoleMember,
				Status:     test.target,
				TraceID:    accountStatusTraceID,
			})
			if err != nil {
				t.Fatalf("SetAccountStatus() error = %v", err)
			}
			if result.User.ID != accountStatusTargetID ||
				result.User.Status != test.target ||
				result.Changed != test.wantChanged {
				t.Fatalf("SetAccountStatus() result = %#v", result)
			}
			if repository.lockCalls != 2 {
				t.Fatalf("LockUser() calls = %d, want 2", repository.lockCalls)
			}
			if repository.updateCalls != test.wantUpdate {
				t.Fatalf("UpdateStatus() calls = %d, want %d", repository.updateCalls, test.wantUpdate)
			}
			if repository.revokeCalls != test.wantRevoke {
				t.Fatalf("RevokeSessions() calls = %d, want %d", repository.revokeCalls, test.wantRevoke)
			}
			if auditor.calls != 1 || auditor.tx != tx {
				t.Fatalf("audit calls/tx = %d/%p, want 1/%p", auditor.calls, auditor.tx, tx)
			}
			wantOutcome := accountStatusAuditOutcomeNoChange
			if test.wantChanged {
				wantOutcome = accountStatusAuditOutcomeChanged
			}
			if auditor.event.Outcome != wantOutcome ||
				auditor.event.Action != accountStatusAuditAction(test.target) {
				t.Fatalf("audit event = %#v", auditor.event)
			}
			if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
				t.Fatalf("transaction commit/rollback = %d/%d, want 1/0", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestAccountStatus_ReauthorizesEveryRequestAndChecksTargetEligibility(t *testing.T) {
	now := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	txOne := &accountStatusTxStub{}
	txTwo := &accountStatusTxStub{}
	repository := &accountStatusRepositoryStub{
		users: []User{
			accountStatusActor(false),
			{ID: accountStatusTargetID, Role: RoleMember, Status: UserStatusSuspended},
			accountStatusActor(false),
			{ID: accountStatusTargetID, Role: RoleMember, Status: UserStatusSuspended},
		},
	}
	service, err := NewAccountStatusService(AccountStatusDependencies{
		Transactions: &accountStatusTransactorStub{transactions: []pgx.Tx{txOne, txTwo}},
		Repository:   repository,
		Audit:        &accountStatusAuditStub{},
		IDs:          &accountStatusIDStub{},
		Clock:        accountStatusClockStub{now: now},
	})
	if err != nil {
		t.Fatalf("NewAccountStatusService() error = %v", err)
	}

	for range 2 {
		_, err := service.SetAccountStatus(context.Background(), AccountStatusRequest{
			Actor: accountStatusPrincipal(false), TargetID: accountStatusTargetID,
			TargetRole: RoleMember, Status: UserStatusSuspended, TraceID: accountStatusTraceID,
		})
		if err != nil {
			t.Fatalf("SetAccountStatus() error = %v", err)
		}
	}
	if repository.lockCalls != 4 {
		t.Fatalf("LockUser() calls = %d, want actor+target for every request", repository.lockCalls)
	}
}

func TestAccountStatus_CurrentActorEligibilityMatrix(t *testing.T) {
	deletedAt := time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		principal session.Principal
		mutate    func(*User)
	}{
		{name: "role changed", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.Role = RoleMember
		}},
		{name: "status changed", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.Status = UserStatusSuspended
		}},
		{name: "initial administrator fact changed", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.IsInitialAdmin = true
		}},
		{name: "bootstrap credential", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.CredentialMode = CredentialModeBootstrap
		}},
		{name: "temporary credential", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.CredentialMode = CredentialModeTemporary
		}},
		{name: "recovery bootstrap credential", principal: accountStatusPrincipal(true), mutate: func(actor *User) {
			actor.IsInitialAdmin = true
			actor.CredentialMode = CredentialModeRecoveryBootstrap
		}},
		{name: "recovery locked credential with stale unrestricted principal", principal: accountStatusPrincipal(true), mutate: func(actor *User) {
			actor.IsInitialAdmin = true
			actor.CredentialMode = CredentialModeRecoveryLocked
		}},
		{name: "recovery required", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.RecoveryRequired = true
		}},
		{name: "onboarding incomplete", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.OnboardingCompletedAt = nil
		}},
		{name: "deleted marker", principal: accountStatusPrincipal(false), mutate: func(actor *User) {
			actor.DeletedAt = &deletedAt
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor := accountStatusActor(test.principal.IsInitialAdmin)
			test.mutate(&actor)
			tx := &accountStatusTxStub{}
			repository := &accountStatusRepositoryStub{users: []User{
				actor,
				{ID: accountStatusTargetID, Role: RoleMember, Status: UserStatusActive},
			}}
			auditor := &accountStatusAuditStub{}
			service := newAccountStatusServiceForTest(
				t,
				tx,
				repository,
				auditor,
				time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC),
			)

			_, err := service.SetAccountStatus(context.Background(), AccountStatusRequest{
				Actor: test.principal, TargetID: accountStatusTargetID,
				TargetRole: RoleMember, Status: UserStatusSuspended,
				TraceID: accountStatusTraceID,
			})
			if platform.PublicCode(err) != platform.CodeAuthForbidden {
				t.Fatalf("SetAccountStatus() code = %q, want %q (err=%v)", platform.PublicCode(err), platform.CodeAuthForbidden, err)
			}
			if repository.lockCalls != 1 {
				t.Fatalf("LockUser() calls = %d, want actor only", repository.lockCalls)
			}
			if repository.updateCalls != 0 || repository.revokeCalls != 0 || auditor.calls != 0 {
				t.Fatalf("forbidden actor side effects: update=%d revoke=%d audit=%d", repository.updateCalls, repository.revokeCalls, auditor.calls)
			}
			if tx.commitCalls != 0 || tx.rollbackCalls != 1 {
				t.Fatalf("transaction commit/rollback = %d/%d, want 0/1", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestAccountStatus_PermissionAndTerminalStateFailures(t *testing.T) {
	tests := []struct {
		name       string
		principal  session.Principal
		actor      User
		target     User
		targetRole Role
		wantCode   platform.Code
	}{
		{
			name:      "ordinary administrator cannot govern administrator",
			principal: accountStatusPrincipal(false), actor: accountStatusActor(false),
			target:     User{ID: accountStatusTargetID, Role: RoleAdmin, Status: UserStatusActive},
			targetRole: RoleAdmin, wantCode: platform.CodeAuthForbidden,
		},
		{
			name:      "administrator cannot govern self",
			principal: accountStatusPrincipal(false), actor: accountStatusActor(false),
			target:     User{ID: accountStatusActorID, Role: RoleMember, Status: UserStatusActive},
			targetRole: RoleMember, wantCode: platform.CodeAuthForbidden,
		},
		{
			name:      "initial administrator is protected",
			principal: accountStatusPrincipal(true), actor: accountStatusActor(true),
			target:     User{ID: accountStatusTargetID, Role: RoleAdmin, IsInitialAdmin: true, Status: UserStatusActive},
			targetRole: RoleAdmin, wantCode: platform.CodeIdentityInitialAdminProtected,
		},
		{
			name:      "deleted is terminal",
			principal: accountStatusPrincipal(false), actor: accountStatusActor(false),
			target:     User{ID: accountStatusTargetID, Role: RoleMember, Status: UserStatusDeleted},
			targetRole: RoleMember, wantCode: platform.CodeIdentityInvalidState,
		},
		{
			name:      "path role mismatch is not disclosed",
			principal: accountStatusPrincipal(true), actor: accountStatusActor(true),
			target:     User{ID: accountStatusTargetID, Role: RoleAdmin, Status: UserStatusActive},
			targetRole: RoleMember, wantCode: platform.CodeIdentityUserNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &accountStatusTxStub{}
			repository := &accountStatusRepositoryStub{users: []User{test.actor, test.target}}
			auditor := &accountStatusAuditStub{}
			service := newAccountStatusServiceForTest(
				t, tx, repository, auditor,
				time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC),
			)
			targetID := test.target.ID
			if test.name == "administrator cannot govern self" {
				targetID = accountStatusActorID
			}

			_, err := service.SetAccountStatus(context.Background(), AccountStatusRequest{
				Actor: test.principal, TargetID: targetID, TargetRole: test.targetRole,
				Status: UserStatusSuspended, TraceID: accountStatusTraceID,
			})
			if platform.PublicCode(err) != test.wantCode {
				t.Fatalf("SetAccountStatus() code = %q, want %q (err=%v)", platform.PublicCode(err), test.wantCode, err)
			}
			if repository.updateCalls != 0 || repository.revokeCalls != 0 || auditor.calls != 0 {
				t.Fatalf(
					"failure wrote status/session/audit: update=%d revoke=%d audit=%d",
					repository.updateCalls,
					repository.revokeCalls,
					auditor.calls,
				)
			}
			if tx.commitCalls != 0 || tx.rollbackCalls != 1 {
				t.Fatalf("transaction commit/rollback = %d/%d, want 0/1", tx.commitCalls, tx.rollbackCalls)
			}
		})
	}
}

func TestAccountStatus_DependencyFailureMatrix(t *testing.T) {
	now := time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		transactor   *accountStatusTransactorStub
		repository   *accountStatusRepositoryStub
		auditor      *accountStatusAuditStub
		id           *accountStatusIDStub
		clock        accountStatusClockStub
		wantCode     platform.Code
		wantLocks    int
		wantUpdate   int
		wantRevoke   int
		wantAudit    int
		wantCommit   int
		wantRollback int
		wantIDCalls  int
	}{
		{
			name:       "begin failure",
			transactor: &accountStatusTransactorStub{err: errors.New("synthetic begin failure")},
			repository: &accountStatusRepositoryStub{}, auditor: &accountStatusAuditStub{},
			id: &accountStatusIDStub{}, clock: accountStatusClockStub{now: now},
			wantCode: platform.CodePlatformInternalError,
		},
		{
			name:       "nil transaction",
			transactor: &accountStatusTransactorStub{returnNil: true},
			repository: &accountStatusRepositoryStub{}, auditor: &accountStatusAuditStub{},
			id: &accountStatusIDStub{}, clock: accountStatusClockStub{now: now},
			wantCode: platform.CodePlatformInternalError,
		},
		{
			name: "actor missing",
			repository: &accountStatusRepositoryStub{
				users:    []User{{}},
				lockErrs: []error{pgx.ErrNoRows},
			},
			auditor: &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodeAuthForbidden,
			wantLocks: 1, wantRollback: 1,
		},
		{
			name: "actor lock failure",
			repository: &accountStatusRepositoryStub{
				users:    []User{accountStatusActor(false)},
				lockErrs: []error{errors.New("synthetic actor lock failure")},
			},
			auditor: &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 1, wantRollback: 1,
		},
		{
			name: "target missing",
			repository: &accountStatusRepositoryStub{
				users:    []User{accountStatusActor(false), {}},
				lockErrs: []error{nil, pgx.ErrNoRows},
			},
			auditor: &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodeIdentityUserNotFound,
			wantLocks: 2, wantRollback: 1,
		},
		{
			name: "target lock failure",
			repository: &accountStatusRepositoryStub{
				users:    []User{accountStatusActor(false), {}},
				lockErrs: []error{nil, errors.New("synthetic target lock failure")},
			},
			auditor: &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantRollback: 1,
		},
		{
			name:       "zero clock",
			repository: accountStatusWritableRepository(),
			auditor:    &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantRollback: 1,
		},
		{
			name:       "ID failure",
			repository: accountStatusWritableRepository(),
			auditor:    &accountStatusAuditStub{}, id: &accountStatusIDStub{err: errors.New("synthetic ID failure")},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name:       "empty ID",
			repository: accountStatusWritableRepository(),
			auditor:    &accountStatusAuditStub{}, id: &accountStatusIDStub{useResult: true},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name: "update failure",
			repository: func() *accountStatusRepositoryStub {
				repository := accountStatusWritableRepository()
				repository.updateErr = errors.New("synthetic update failure")
				return repository
			}(),
			auditor: &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantUpdate: 1, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name: "session revoke failure",
			repository: func() *accountStatusRepositoryStub {
				repository := accountStatusWritableRepository()
				repository.revokeErr = errors.New("synthetic revoke failure")
				return repository
			}(),
			auditor: &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantUpdate: 1, wantRevoke: 1, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name:       "audit failure",
			repository: accountStatusWritableRepository(),
			auditor:    &accountStatusAuditStub{err: errors.New("synthetic audit failure")},
			id:         &accountStatusIDStub{}, clock: accountStatusClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2,
			wantUpdate: 1, wantRevoke: 1, wantAudit: 1, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name: "commit failure",
			transactor: &accountStatusTransactorStub{transactions: []pgx.Tx{
				&accountStatusTxStub{commitErr: errors.New("synthetic commit failure")},
			}},
			repository: accountStatusWritableRepository(),
			auditor:    &accountStatusAuditStub{}, id: &accountStatusIDStub{},
			clock: accountStatusClockStub{now: now}, wantCode: platform.CodePlatformInternalError,
			wantLocks: 2, wantUpdate: 1, wantRevoke: 1, wantAudit: 1,
			wantCommit: 1, wantIDCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &accountStatusTxStub{}
			transactor := test.transactor
			if transactor == nil {
				transactor = &accountStatusTransactorStub{transactions: []pgx.Tx{tx}}
			} else if len(transactor.transactions) > 0 {
				tx = transactor.transactions[0].(*accountStatusTxStub)
			}
			service, err := NewAccountStatusService(AccountStatusDependencies{
				Transactions: transactor, Repository: test.repository,
				Audit: test.auditor, IDs: test.id, Clock: test.clock,
			})
			if err != nil {
				t.Fatalf("NewAccountStatusService() error = %v", err)
			}

			_, err = service.SetAccountStatus(context.Background(), AccountStatusRequest{
				Actor: accountStatusPrincipal(false), TargetID: accountStatusTargetID,
				TargetRole: RoleMember, Status: UserStatusSuspended,
				TraceID: accountStatusTraceID,
			})
			if platform.PublicCode(err) != test.wantCode {
				t.Fatalf("SetAccountStatus() code = %q, want %q (err=%v)", platform.PublicCode(err), test.wantCode, err)
			}
			if test.repository.lockCalls != test.wantLocks ||
				test.repository.updateCalls != test.wantUpdate ||
				test.repository.revokeCalls != test.wantRevoke ||
				test.auditor.calls != test.wantAudit {
				t.Fatalf(
					"calls lock/update/revoke/audit = %d/%d/%d/%d, want %d/%d/%d/%d",
					test.repository.lockCalls, test.repository.updateCalls,
					test.repository.revokeCalls, test.auditor.calls,
					test.wantLocks, test.wantUpdate, test.wantRevoke, test.wantAudit,
				)
			}
			if tx.commitCalls != test.wantCommit || tx.rollbackCalls != test.wantRollback {
				t.Fatalf("transaction commit/rollback = %d/%d, want %d/%d", tx.commitCalls, tx.rollbackCalls, test.wantCommit, test.wantRollback)
			}
			if test.id.calls != test.wantIDCalls {
				t.Fatalf("NewID() calls = %d, want %d", test.id.calls, test.wantIDCalls)
			}
		})
	}
}

func TestAccountStatus_AuditFailureRollsBackStatusAndSessions(t *testing.T) {
	tx := &accountStatusTxStub{}
	repository := &accountStatusRepositoryStub{users: []User{
		accountStatusActor(false),
		{ID: accountStatusTargetID, Role: RoleMember, Status: UserStatusActive},
	}}
	service := newAccountStatusServiceForTest(
		t, tx, repository, &accountStatusAuditStub{err: errors.New("synthetic audit failure")},
		time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC),
	)

	_, err := service.SetAccountStatus(context.Background(), AccountStatusRequest{
		Actor: accountStatusPrincipal(false), TargetID: accountStatusTargetID,
		TargetRole: RoleMember, Status: UserStatusSuspended, TraceID: accountStatusTraceID,
	})
	if platform.PublicCode(err) != platform.CodePlatformInternalError {
		t.Fatalf("SetAccountStatus() code = %q, want %q", platform.PublicCode(err), platform.CodePlatformInternalError)
	}
	if repository.updateCalls != 1 || repository.revokeCalls != 1 {
		t.Fatalf("pre-audit transaction calls = update %d revoke %d, want 1/1", repository.updateCalls, repository.revokeCalls)
	}
	if tx.commitCalls != 0 || tx.rollbackCalls != 1 {
		t.Fatalf("transaction commit/rollback = %d/%d, want 0/1", tx.commitCalls, tx.rollbackCalls)
	}
}

func TestAccountStatusHandler_UsesUserAndAdministratorPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		call       func(*Handler, http.ResponseWriter, *http.Request)
		targetRole Role
		status     UserStatus
	}{
		{name: "suspend user", path: "/api/v1/admin/users/" + accountStatusTargetID + "/suspend", call: (*Handler).SuspendUser, targetRole: RoleMember, status: UserStatusSuspended},
		{name: "restore user", path: "/api/v1/admin/users/" + accountStatusTargetID + "/restore", call: (*Handler).RestoreUser, targetRole: RoleMember, status: UserStatusActive},
		{name: "suspend administrator", path: "/api/v1/admin/administrators/" + accountStatusTargetID + "/suspend", call: (*Handler).SuspendAdministrator, targetRole: RoleAdmin, status: UserStatusSuspended},
		{name: "restore administrator", path: "/api/v1/admin/administrators/" + accountStatusTargetID + "/restore", call: (*Handler).RestoreAdministrator, targetRole: RoleAdmin, status: UserStatusActive},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &accountStatusServiceStub{result: AccountStatusResult{
				User:    AccountStatusUser{ID: accountStatusTargetID, Status: test.status},
				Changed: true,
			}}
			handler, err := NewAccountStatusHandler(service)
			if err != nil {
				t.Fatalf("NewAccountStatusHandler() error = %v", err)
			}
			request := httptest.NewRequest(http.MethodPost, test.path, nil)
			traceContext, err := platform.ContextWithTraceID(request.Context(), accountStatusTraceID)
			if err != nil {
				t.Fatalf("ContextWithTraceID() error = %v", err)
			}
			request = request.WithContext(traceContext)
			request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
			request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
			response := httptest.NewRecorder()
			authorizer, err := platform.NewHTTPAuthorizer(
				accountStatusPrincipalResolver{principal: accountStatusPrincipal(true)},
				func() time.Time { return time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC) },
			)
			if err != nil {
				t.Fatalf("NewHTTPAuthorizer() error = %v", err)
			}
			authorizer.Protect(session.AccessGeneral, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				test.call(handler, w, r)
			})).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body struct {
				User    AccountStatusUser `json:"user"`
				Changed bool              `json:"changed"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.User.ID != accountStatusTargetID || body.User.Status != test.status || !body.Changed {
				t.Fatalf("response body = %#v", body)
			}
			if service.calls != 1 || service.request.TargetID != accountStatusTargetID ||
				service.request.TargetRole != test.targetRole ||
				service.request.Status != test.status ||
				service.request.TraceID != accountStatusTraceID {
				t.Fatalf("service request = %#v, calls=%d", service.request, service.calls)
			}
		})
	}
}

func TestAccountStatusHandler_RejectsMalformedTargetsAndWritesProtectedEnvelope(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		service     AccountStatusService
		wantStatus  int
		wantCode    platform.Code
		wantMessage string
		wantCalls   int
	}{
		{
			name:       "malformed path",
			path:       "/api/v1/admin/users/" + accountStatusTargetID + "/suspend/extra",
			service:    &accountStatusServiceStub{},
			wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
			wantMessage: "用户不存在",
		},
		{
			name: "malformed UUID",
			path: "/api/v1/admin/users/not-a-uuid/suspend",
			service: func() AccountStatusService {
				service, err := NewAccountStatusService(AccountStatusDependencies{
					Transactions: &accountStatusTransactorStub{},
					Repository:   &accountStatusRepositoryStub{}, Audit: &accountStatusAuditStub{},
					IDs: &accountStatusIDStub{}, Clock: accountStatusClockStub{now: time.Now()},
				})
				if err != nil {
					t.Fatalf("NewAccountStatusService() error = %v", err)
				}
				return service
			}(),
			wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
			wantMessage: "用户不存在",
		},
		{
			name: "initial administrator protected",
			path: "/api/v1/admin/administrators/" + accountStatusTargetID + "/suspend",
			service: &accountStatusServiceStub{err: platform.NewPublicError(
				platform.CodeIdentityInitialAdminProtected,
				errors.New("synthetic initial administrator protected"),
			)},
			wantStatus: http.StatusForbidden, wantCode: platform.CodeIdentityInitialAdminProtected,
			wantMessage: "初始管理员账号受保护", wantCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewAccountStatusHandler(test.service)
			if err != nil {
				t.Fatalf("NewAccountStatusHandler() error = %v", err)
			}
			request := httptest.NewRequest(http.MethodPost, test.path, nil)
			ctx, err := platform.ContextWithTraceID(request.Context(), accountStatusTraceID)
			if err != nil {
				t.Fatalf("ContextWithTraceID() error = %v", err)
			}
			request = request.WithContext(ctx)
			request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
			request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
			response := httptest.NewRecorder()
			authorizer, err := platform.NewHTTPAuthorizer(
				accountStatusPrincipalResolver{principal: accountStatusPrincipal(true)},
				func() time.Time { return time.Date(2026, time.October, 24, 9, 0, 0, 0, time.UTC) },
			)
			if err != nil {
				t.Fatalf("NewHTTPAuthorizer() error = %v", err)
			}
			authorizer.Protect(session.AccessGeneral, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(test.path, "/administrators/") {
					handler.SuspendAdministrator(w, r)
				} else {
					handler.SuspendUser(w, r)
				}
			})).ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode ||
				envelope.Message != test.wantMessage ||
				envelope.TraceID != accountStatusTraceID {
				t.Fatalf("error envelope = %#v", envelope)
			}
			if service, ok := test.service.(*accountStatusServiceStub); ok &&
				service.calls != test.wantCalls {
				t.Fatalf("SetAccountStatus() calls = %d, want %d", service.calls, test.wantCalls)
			}
		})
	}
}

const (
	accountStatusActorID  = "00000000-0000-0000-0000-000000000001"
	accountStatusTargetID = "00000000-0000-0000-0000-000000000002"
	accountStatusTraceID  = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

func newAccountStatusServiceForTest(
	t *testing.T,
	tx pgx.Tx,
	repository AccountStatusRepository,
	auditor AccountStatusAuditAppender,
	now time.Time,
) *AccountStatusCoordinator {
	t.Helper()
	service, err := NewAccountStatusService(AccountStatusDependencies{
		Transactions: &accountStatusTransactorStub{transactions: []pgx.Tx{tx}},
		Repository:   repository, Audit: auditor,
		IDs: &accountStatusIDStub{}, Clock: accountStatusClockStub{now: now},
	})
	if err != nil {
		t.Fatalf("NewAccountStatusService() error = %v", err)
	}
	return service
}

func accountStatusPrincipal(initial bool) session.Principal {
	return session.Principal{
		UserID: accountStatusActorID, Role: string(RoleAdmin),
		IsInitialAdmin: initial, Status: string(UserStatusActive),
	}
}

func accountStatusActor(initial bool) User {
	completedAt := time.Date(2026, time.October, 23, 9, 0, 0, 0, time.UTC)
	return User{
		ID: accountStatusActorID, Role: RoleAdmin, IsInitialAdmin: initial,
		Status: UserStatusActive, CredentialMode: CredentialModePermanent,
		OnboardingCompletedAt: &completedAt,
	}
}

type accountStatusTransactorStub struct {
	transactions []pgx.Tx
	next         int
	calls        int
	err          error
	returnNil    bool
}

func (s *accountStatusTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.returnNil {
		return nil, nil
	}
	if s.next >= len(s.transactions) {
		return nil, errors.New("unexpected account status transaction")
	}
	tx := s.transactions[s.next]
	s.next++
	return tx, nil
}

type accountStatusTxStub struct {
	pgx.Tx
	commitCalls   int
	rollbackCalls int
	commitErr     error
	rollbackErr   error
}

func (s *accountStatusTxStub) Commit(context.Context) error {
	s.commitCalls++
	return s.commitErr
}

func (s *accountStatusTxStub) Rollback(context.Context) error {
	s.rollbackCalls++
	return s.rollbackErr
}

type accountStatusRepositoryStub struct {
	users        []User
	lockErrs     []error
	lockCalls    int
	updateCalls  int
	revokeCalls  int
	updateOld    UserStatus
	updateNew    UserStatus
	revokeReason string
	updateErr    error
	revokeErr    error
}

func (s *accountStatusRepositoryStub) LockUser(context.Context, pgx.Tx, string) (User, error) {
	index := s.lockCalls
	s.lockCalls++
	if index < len(s.lockErrs) && s.lockErrs[index] != nil {
		return User{}, s.lockErrs[index]
	}
	if index >= len(s.users) {
		return User{}, pgx.ErrNoRows
	}
	return s.users[index], nil
}

func (s *accountStatusRepositoryStub) UpdateStatus(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	oldStatus UserStatus,
	newStatus UserStatus,
	_ time.Time,
) error {
	s.updateCalls++
	s.updateOld = oldStatus
	s.updateNew = newStatus
	return s.updateErr
}

func (s *accountStatusRepositoryStub) RevokeSessions(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	_ time.Time,
	reason string,
) error {
	s.revokeCalls++
	s.revokeReason = reason
	return s.revokeErr
}

type accountStatusAuditStub struct {
	event audit.Event
	tx    pgx.Tx
	calls int
	err   error
}

func (s *accountStatusAuditStub) Append(_ context.Context, tx pgx.Tx, event audit.Event) error {
	s.calls++
	s.tx = tx
	s.event = event
	return s.err
}

type accountStatusIDStub struct {
	next      int
	calls     int
	result    string
	useResult bool
	err       error
}

func (s *accountStatusIDStub) NewID() (string, error) {
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	if s.useResult {
		return s.result, nil
	}
	s.next++
	return "00000000-0000-0000-0000-00000000000" + string(rune('0'+s.next)), nil
}

type accountStatusClockStub struct{ now time.Time }

func (s accountStatusClockStub) Now() time.Time { return s.now }

type accountStatusServiceStub struct {
	result  AccountStatusResult
	err     error
	request AccountStatusRequest
	calls   int
}

func (s *accountStatusServiceStub) SetAccountStatus(
	_ context.Context,
	request AccountStatusRequest,
) (AccountStatusResult, error) {
	s.calls++
	s.request = request
	return s.result, s.err
}

type accountStatusPrincipalResolver struct{ principal session.Principal }

func (s accountStatusPrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return s.principal, nil
}

func accountStatusWritableRepository() *accountStatusRepositoryStub {
	return &accountStatusRepositoryStub{users: []User{
		accountStatusActor(false),
		{ID: accountStatusTargetID, Role: RoleMember, Status: UserStatusActive},
	}}
}

package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

var _ AdministratorGovernanceRepository = PostgresAdministratorGovernanceRepository{}

func TestAdministratorGovernanceRevokeTransitionsTargetToMember(t *testing.T) {
	now := time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC)
	tx := &administratorGovernanceTxStub{}
	repository := &administratorGovernanceRepositoryStub{users: []User{
		administratorGovernanceActor(true),
		{ID: administratorGovernanceTargetID, Role: RoleAdmin, Status: UserStatusActive},
	}}
	auditor := &administratorGovernanceAuditStub{}
	service := newAdministratorGovernanceServiceForTest(t, tx, repository, auditor, now)

	result, err := service.RevokeAdministrator(
		context.Background(),
		AdministratorRevokeRequest{
			Actor:    administratorGovernancePrincipal(true),
			TargetID: administratorGovernanceTargetID,
			TraceID:  administratorGovernanceTraceID,
		},
	)
	if err != nil {
		t.Fatalf("RevokeAdministrator() error = %v", err)
	}
	if result.User.ID != administratorGovernanceTargetID ||
		result.User.Role != RoleMember {
		t.Fatalf("RevokeAdministrator() result = %#v", result)
	}
	if repository.lockCalls != 2 {
		t.Fatalf("LockUser() calls = %d, want actor and target", repository.lockCalls)
	}
	if repository.updateRoleCalls != 1 ||
		repository.updateOldRole != RoleAdmin ||
		repository.updateNewRole != RoleMember {
		t.Fatalf(
			"UpdateRole() calls/roles = %d/%q->%q, want 1/admin->member",
			repository.updateRoleCalls,
			repository.updateOldRole,
			repository.updateNewRole,
		)
	}
	if repository.revokeCalls != 1 ||
		repository.revokeReason != administratorGovernanceSessionRevokeReason {
		t.Fatalf(
			"RevokeSessions() calls/reason = %d/%q, want 1/%q",
			repository.revokeCalls,
			repository.revokeReason,
			administratorGovernanceSessionRevokeReason,
		)
	}
	if auditor.calls != 1 || auditor.tx != tx {
		t.Fatalf("audit calls/tx = %d/%p, want 1/%p", auditor.calls, auditor.tx, tx)
	}
	if auditor.event.Action != administratorGovernanceAuditAction ||
		auditor.event.Outcome != accountStatusAuditOutcomeChanged ||
		auditor.event.TargetID != administratorGovernanceTargetID ||
		auditor.event.TraceID != administratorGovernanceTraceID {
		t.Fatalf("audit event = %#v", auditor.event)
	}
	if tx.commitCalls != 1 || tx.rollbackCalls != 0 {
		t.Fatalf(
			"transaction commit/rollback = %d/%d, want 1/0",
			tx.commitCalls,
			tx.rollbackCalls,
		)
	}
}

func TestAdministratorGovernanceRevokesSuspendedAdministrator(t *testing.T) {
	now := time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC)
	tx := &administratorGovernanceTxStub{}
	repository := &administratorGovernanceRepositoryStub{users: []User{
		administratorGovernanceActor(true),
		{ID: administratorGovernanceTargetID, Role: RoleAdmin, Status: UserStatusSuspended},
	}}
	service := newAdministratorGovernanceServiceForTest(
		t,
		tx,
		repository,
		&administratorGovernanceAuditStub{},
		now,
	)

	result, err := service.RevokeAdministrator(
		context.Background(),
		AdministratorRevokeRequest{
			Actor:    administratorGovernancePrincipal(true),
			TargetID: administratorGovernanceTargetID,
			TraceID:  administratorGovernanceTraceID,
		},
	)
	if err != nil {
		t.Fatalf("RevokeAdministrator() error = %v", err)
	}
	if result.User.Role != RoleMember || repository.updateRoleCalls != 1 ||
		repository.revokeCalls != 1 {
		t.Fatalf(
			"revoke suspended administrator = %#v, update=%d revoke=%d",
			result,
			repository.updateRoleCalls,
			repository.revokeCalls,
		)
	}
}

func TestAdministratorGovernancePermissionMatrix(t *testing.T) {
	deletedAt := time.Date(2026, time.October, 27, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		principal session.Principal
		actor     User
		target    User
		targetID  string
		lockErrs  []error
		wantCode  platform.Code
		wantRoll  int
	}{
		{
			name:      "ordinary administrator cannot revoke administrators",
			principal: administratorGovernancePrincipal(false),
			actor:     administratorGovernanceActor(false),
			target:    User{ID: administratorGovernanceTargetID, Role: RoleAdmin, Status: UserStatusActive},
			targetID:  administratorGovernanceTargetID,
			wantCode:  platform.CodeAuthForbidden,
			wantRoll:  1,
		},
		{
			name:      "initial administrator cannot revoke self",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			target:    User{ID: administratorGovernanceActorID, Role: RoleAdmin, Status: UserStatusActive},
			targetID:  administratorGovernanceActorID,
			wantCode:  platform.CodeAuthForbidden,
			wantRoll:  1,
		},
		{
			name:      "initial administrator target is protected",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			target: User{
				ID: administratorGovernanceTargetID, Role: RoleAdmin,
				IsInitialAdmin: true, Status: UserStatusActive,
			},
			targetID: administratorGovernanceTargetID,
			wantCode: platform.CodeIdentityInitialAdminProtected,
			wantRoll: 1,
		},
		{
			name:      "repeat revoke is an invalid state",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			target:    User{ID: administratorGovernanceTargetID, Role: RoleMember, Status: UserStatusActive},
			targetID:  administratorGovernanceTargetID,
			wantCode:  platform.CodeIdentityInvalidState,
			wantRoll:  1,
		},
		{
			name:      "ordinary member target is not revocable",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			target:    User{ID: administratorGovernanceTargetID, Role: RoleMember, Status: UserStatusSuspended},
			targetID:  administratorGovernanceTargetID,
			wantCode:  platform.CodeIdentityInvalidState,
			wantRoll:  1,
		},
		{
			name:      "deleted administrator is terminal",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			target: User{
				ID: administratorGovernanceTargetID, Role: RoleAdmin,
				Status: UserStatusDeleted, DeletedAt: &deletedAt,
			},
			targetID: administratorGovernanceTargetID,
			wantCode: platform.CodeIdentityInvalidState,
			wantRoll: 1,
		},
		{
			name:      "missing target is not found",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			targetID:  administratorGovernanceTargetID,
			lockErrs:  []error{nil, pgx.ErrNoRows},
			wantCode:  platform.CodeIdentityUserNotFound,
			wantRoll:  1,
		},
		{
			name:      "malformed target identifier is not found",
			principal: administratorGovernancePrincipal(true),
			actor:     administratorGovernanceActor(true),
			targetID:  "not-a-uuid",
			wantCode:  platform.CodeIdentityUserNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &administratorGovernanceTxStub{}
			repository := &administratorGovernanceRepositoryStub{
				users:    []User{test.actor, test.target},
				lockErrs: test.lockErrs,
			}
			auditor := &administratorGovernanceAuditStub{}
			service := newAdministratorGovernanceServiceForTest(
				t,
				tx,
				repository,
				auditor,
				time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC),
			)

			_, err := service.RevokeAdministrator(
				context.Background(),
				AdministratorRevokeRequest{
					Actor:    test.principal,
					TargetID: test.targetID,
					TraceID:  administratorGovernanceTraceID,
				},
			)
			if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf(
					"RevokeAdministrator() code = %q, want %q (err=%v)",
					code,
					test.wantCode,
					err,
				)
			}
			if repository.updateRoleCalls != 0 || repository.revokeCalls != 0 ||
				auditor.calls != 0 {
				t.Fatalf(
					"failure wrote role/session/audit: role=%d revoke=%d audit=%d",
					repository.updateRoleCalls,
					repository.revokeCalls,
					auditor.calls,
				)
			}
			if tx.commitCalls != 0 || tx.rollbackCalls != test.wantRoll {
				t.Fatalf(
					"transaction commit/rollback = %d/%d, want 0/%d",
					tx.commitCalls,
					tx.rollbackCalls,
					test.wantRoll,
				)
			}
		})
	}
}

func TestAdministratorGovernanceActorEligibilityMatrix(t *testing.T) {
	deletedAt := time.Date(2026, time.October, 27, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*User)
	}{
		{name: "role demoted", mutate: func(actor *User) { actor.Role = RoleMember }},
		{name: "status suspended", mutate: func(actor *User) {
			actor.Status = UserStatusSuspended
		}},
		{name: "initial administrator fact removed", mutate: func(actor *User) {
			actor.IsInitialAdmin = false
		}},
		{name: "bootstrap credential", mutate: func(actor *User) {
			actor.CredentialMode = CredentialModeBootstrap
		}},
		{name: "temporary credential", mutate: func(actor *User) {
			actor.CredentialMode = CredentialModeTemporary
		}},
		{name: "recovery locked credential", mutate: func(actor *User) {
			actor.CredentialMode = CredentialModeRecoveryLocked
		}},
		{name: "recovery required", mutate: func(actor *User) { actor.RecoveryRequired = true }},
		{name: "deleted marker", mutate: func(actor *User) { actor.DeletedAt = &deletedAt }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor := administratorGovernanceActor(true)
			test.mutate(&actor)
			tx := &administratorGovernanceTxStub{}
			repository := &administratorGovernanceRepositoryStub{users: []User{
				actor,
				{ID: administratorGovernanceTargetID, Role: RoleAdmin, Status: UserStatusActive},
			}}
			auditor := &administratorGovernanceAuditStub{}
			service := newAdministratorGovernanceServiceForTest(
				t,
				tx,
				repository,
				auditor,
				time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC),
			)

			_, err := service.RevokeAdministrator(
				context.Background(),
				AdministratorRevokeRequest{
					Actor:    administratorGovernancePrincipal(true),
					TargetID: administratorGovernanceTargetID,
					TraceID:  administratorGovernanceTraceID,
				},
			)
			if code := platform.PublicCode(err); code != platform.CodeAuthForbidden {
				t.Fatalf(
					"RevokeAdministrator() code = %q, want %q (err=%v)",
					code,
					platform.CodeAuthForbidden,
					err,
				)
			}
			if repository.lockCalls != 1 {
				t.Fatalf("LockUser() calls = %d, want actor only", repository.lockCalls)
			}
			if repository.updateRoleCalls != 0 || repository.revokeCalls != 0 ||
				auditor.calls != 0 {
				t.Fatalf(
					"forbidden actor side effects: role=%d revoke=%d audit=%d",
					repository.updateRoleCalls,
					repository.revokeCalls,
					auditor.calls,
				)
			}
			if tx.commitCalls != 0 || tx.rollbackCalls != 1 {
				t.Fatalf(
					"transaction commit/rollback = %d/%d, want 0/1",
					tx.commitCalls,
					tx.rollbackCalls,
				)
			}
		})
	}
}

func TestAdministratorGovernanceRequestValidation(t *testing.T) {
	now := time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC)
	restricted := administratorGovernancePrincipal(true)
	restricted.MustChangePassword = true
	tests := []struct {
		name      string
		principal session.Principal
		targetID  string
		traceID   string
		wantCode  platform.Code
	}{
		{
			name: "missing actor", principal: session.Principal{},
			targetID: administratorGovernanceTargetID, traceID: administratorGovernanceTraceID,
			wantCode: platform.CodeAuthForbidden,
		},
		{
			name: "restricted principal", principal: restricted,
			targetID: administratorGovernanceTargetID, traceID: administratorGovernanceTraceID,
			wantCode: platform.CodeAuthForbidden,
		},
		{
			name: "empty trace", principal: administratorGovernancePrincipal(true),
			targetID: administratorGovernanceTargetID,
			wantCode: platform.CodePlatformInternalError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &administratorGovernanceTxStub{}
			repository := &administratorGovernanceRepositoryStub{}
			service := newAdministratorGovernanceServiceForTest(
				t,
				tx,
				repository,
				&administratorGovernanceAuditStub{},
				now,
			)

			_, err := service.RevokeAdministrator(
				context.Background(),
				AdministratorRevokeRequest{
					Actor:    test.principal,
					TargetID: test.targetID,
					TraceID:  test.traceID,
				},
			)
			if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf(
					"RevokeAdministrator() code = %q, want %q (err=%v)",
					code,
					test.wantCode,
					err,
				)
			}
			if repository.lockCalls != 0 || tx.rollbackCalls != 0 {
				t.Fatalf(
					"invalid request side effects: locks=%d rollback=%d",
					repository.lockCalls,
					tx.rollbackCalls,
				)
			}
		})
	}
}

func TestAdministratorGovernanceDependencyFailureMatrix(t *testing.T) {
	now := time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		transactor   *administratorGovernanceTransactorStub
		repository   *administratorGovernanceRepositoryStub
		auditor      *administratorGovernanceAuditStub
		id           *administratorGovernanceIDStub
		clock        administratorGovernanceClockStub
		wantCode     platform.Code
		wantLocks    int
		wantRole     int
		wantRevoke   int
		wantAudit    int
		wantCommit   int
		wantRollback int
		wantIDCalls  int
	}{
		{
			name:       "begin failure",
			transactor: &administratorGovernanceTransactorStub{err: errors.New("synthetic begin failure")},
			repository: &administratorGovernanceRepositoryStub{},
			auditor:    &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError,
		},
		{
			name:       "nil transaction",
			transactor: &administratorGovernanceTransactorStub{returnNil: true},
			repository: &administratorGovernanceRepositoryStub{},
			auditor:    &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError,
		},
		{
			name: "actor missing",
			repository: &administratorGovernanceRepositoryStub{
				lockErrs: []error{pgx.ErrNoRows},
			},
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodeAuthForbidden, wantLocks: 1, wantRollback: 1,
		},
		{
			name: "actor lock failure",
			repository: &administratorGovernanceRepositoryStub{
				users:    []User{administratorGovernanceActor(true)},
				lockErrs: []error{errors.New("synthetic actor lock failure")},
			},
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 1, wantRollback: 1,
		},
		{
			name: "target missing",
			repository: &administratorGovernanceRepositoryStub{
				users:    []User{administratorGovernanceActor(true), {}},
				lockErrs: []error{nil, pgx.ErrNoRows},
			},
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodeIdentityUserNotFound, wantLocks: 2, wantRollback: 1,
		},
		{
			name: "target lock failure",
			repository: &administratorGovernanceRepositoryStub{
				users:    []User{administratorGovernanceActor(true), {}},
				lockErrs: []error{nil, errors.New("synthetic target lock failure")},
			},
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRollback: 1,
		},
		{
			name: "zero clock", repository: administratorGovernanceWritableRepository(),
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRollback: 1,
		},
		{
			name: "ID failure", repository: administratorGovernanceWritableRepository(),
			auditor:  &administratorGovernanceAuditStub{},
			id:       &administratorGovernanceIDStub{err: errors.New("synthetic ID failure")},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRollback: 1,
			wantIDCalls: 1,
		},
		{
			name: "empty ID", repository: administratorGovernanceWritableRepository(),
			auditor:  &administratorGovernanceAuditStub{},
			id:       &administratorGovernanceIDStub{useResult: true},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRollback: 1,
			wantIDCalls: 1,
		},
		{
			name: "role update failure",
			repository: func() *administratorGovernanceRepositoryStub {
				repository := administratorGovernanceWritableRepository()
				repository.updateRoleErr = errors.New("synthetic role update failure")
				return repository
			}(),
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRole: 1,
			wantRollback: 1, wantIDCalls: 1,
		},
		{
			name: "session revoke failure",
			repository: func() *administratorGovernanceRepositoryStub {
				repository := administratorGovernanceWritableRepository()
				repository.revokeErr = errors.New("synthetic revoke failure")
				return repository
			}(),
			auditor: &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRole: 1,
			wantRevoke: 1, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name: "audit failure", repository: administratorGovernanceWritableRepository(),
			auditor:  &administratorGovernanceAuditStub{err: errors.New("synthetic audit failure")},
			id:       &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRole: 1,
			wantRevoke: 1, wantAudit: 1, wantRollback: 1, wantIDCalls: 1,
		},
		{
			name: "commit failure",
			transactor: &administratorGovernanceTransactorStub{transactions: []pgx.Tx{
				&administratorGovernanceTxStub{commitErr: errors.New("synthetic commit failure")},
			}},
			repository: administratorGovernanceWritableRepository(),
			auditor:    &administratorGovernanceAuditStub{}, id: &administratorGovernanceIDStub{},
			clock:    administratorGovernanceClockStub{now: now},
			wantCode: platform.CodePlatformInternalError, wantLocks: 2, wantRole: 1,
			wantRevoke: 1, wantAudit: 1, wantCommit: 1, wantIDCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &administratorGovernanceTxStub{}
			transactor := test.transactor
			if transactor == nil {
				transactor = &administratorGovernanceTransactorStub{
					transactions: []pgx.Tx{tx},
				}
			} else if len(transactor.transactions) > 0 {
				tx = transactor.transactions[0].(*administratorGovernanceTxStub)
			}
			service, err := NewAdministratorGovernanceService(
				AdministratorGovernanceDependencies{
					Transactions: transactor,
					Repository:   test.repository,
					Audit:        test.auditor,
					IDs:          test.id,
					Clock:        test.clock,
				},
			)
			if err != nil {
				t.Fatalf("NewAdministratorGovernanceService() error = %v", err)
			}

			_, err = service.RevokeAdministrator(
				context.Background(),
				AdministratorRevokeRequest{
					Actor:    administratorGovernancePrincipal(true),
					TargetID: administratorGovernanceTargetID,
					TraceID:  administratorGovernanceTraceID,
				},
			)
			if code := platform.PublicCode(err); code != test.wantCode {
				t.Fatalf(
					"RevokeAdministrator() code = %q, want %q (err=%v)",
					code,
					test.wantCode,
					err,
				)
			}
			if test.repository.lockCalls != test.wantLocks ||
				test.repository.updateRoleCalls != test.wantRole ||
				test.repository.revokeCalls != test.wantRevoke ||
				test.auditor.calls != test.wantAudit {
				t.Fatalf(
					"calls lock/role/revoke/audit = %d/%d/%d/%d, want %d/%d/%d/%d",
					test.repository.lockCalls,
					test.repository.updateRoleCalls,
					test.repository.revokeCalls,
					test.auditor.calls,
					test.wantLocks,
					test.wantRole,
					test.wantRevoke,
					test.wantAudit,
				)
			}
			if tx.commitCalls != test.wantCommit || tx.rollbackCalls != test.wantRollback {
				t.Fatalf(
					"transaction commit/rollback = %d/%d, want %d/%d",
					tx.commitCalls,
					tx.rollbackCalls,
					test.wantCommit,
					test.wantRollback,
				)
			}
			if test.id.calls != test.wantIDCalls {
				t.Fatalf("NewID() calls = %d, want %d", test.id.calls, test.wantIDCalls)
			}
		})
	}
}

func TestAdministratorGovernanceConstructorRejectsIncompleteDependencies(t *testing.T) {
	complete := AdministratorGovernanceDependencies{
		Transactions: &administratorGovernanceTransactorStub{},
		Repository:   &administratorGovernanceRepositoryStub{},
		Audit:        &administratorGovernanceAuditStub{},
		IDs:          &administratorGovernanceIDStub{},
		Clock:        administratorGovernanceClockStub{now: time.Now()},
	}
	tests := []struct {
		name   string
		mutate func(*AdministratorGovernanceDependencies)
	}{
		{name: "missing transactions", mutate: func(d *AdministratorGovernanceDependencies) {
			d.Transactions = nil
		}},
		{name: "missing repository", mutate: func(d *AdministratorGovernanceDependencies) {
			d.Repository = nil
		}},
		{name: "typed nil repository", mutate: func(d *AdministratorGovernanceDependencies) {
			d.Repository = (*administratorGovernanceRepositoryStub)(nil)
		}},
		{name: "missing audit", mutate: func(d *AdministratorGovernanceDependencies) {
			d.Audit = nil
		}},
		{name: "missing ids", mutate: func(d *AdministratorGovernanceDependencies) {
			d.IDs = nil
		}},
		{name: "missing clock", mutate: func(d *AdministratorGovernanceDependencies) {
			d.Clock = nil
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dependencies := complete
			test.mutate(&dependencies)
			if _, err := NewAdministratorGovernanceService(dependencies); err == nil {
				t.Fatal("NewAdministratorGovernanceService() error = nil, want failure")
			}
		})
	}
}

func TestAdministratorGovernanceHandlerRevokesAdministrator(t *testing.T) {
	service := &administratorGovernanceServiceStub{result: AdministratorRevokeResult{
		User: AdministratorRevokeUser{ID: administratorGovernanceTargetID, Role: RoleMember},
	}}
	handler, err := NewAdministratorGovernanceHandler(service)
	if err != nil {
		t.Fatalf("NewAdministratorGovernanceHandler() error = %v", err)
	}
	response := serveAdministratorGovernanceRequest(
		t,
		handler,
		http.MethodPost,
		"/api/v1/admin/administrators/"+administratorGovernanceTargetID+"/revoke",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		User AdministratorRevokeUser `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.User.ID != administratorGovernanceTargetID || body.User.Role != RoleMember {
		t.Fatalf("response body = %#v", body)
	}
	if service.calls != 1 ||
		service.request.TargetID != administratorGovernanceTargetID ||
		service.request.TraceID != administratorGovernanceTraceID {
		t.Fatalf("service request = %#v, calls=%d", service.request, service.calls)
	}
}

func TestAdministratorGovernanceHandlerRejectsMalformedTargets(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		service    AdministratorGovernanceService
		wantStatus int
		wantCode   platform.Code
		wantCalls  int
	}{
		{
			name:       "trailing segment",
			path:       "/api/v1/admin/administrators/" + administratorGovernanceTargetID + "/revoke/extra",
			service:    &administratorGovernanceServiceStub{},
			wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
		},
		{
			name:       "missing target",
			path:       "/api/v1/admin/administrators/revoke",
			service:    &administratorGovernanceServiceStub{},
			wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
		},
		{
			name:       "malformed identifier",
			path:       "/api/v1/admin/administrators/not-a-uuid/revoke",
			service:    &administratorGovernanceServiceStub{},
			wantStatus: http.StatusNotFound, wantCode: platform.CodeIdentityUserNotFound,
		},
		{
			name: "protected initial administrator",
			path: "/api/v1/admin/administrators/" + administratorGovernanceTargetID + "/revoke",
			service: &administratorGovernanceServiceStub{err: platform.NewPublicError(
				platform.CodeIdentityInitialAdminProtected,
				errors.New("synthetic initial administrator protected"),
			)},
			wantStatus: http.StatusForbidden,
			wantCode:   platform.CodeIdentityInitialAdminProtected,
			wantCalls:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewAdministratorGovernanceHandler(test.service)
			if err != nil {
				t.Fatalf("NewAdministratorGovernanceHandler() error = %v", err)
			}
			response := serveAdministratorGovernanceRequest(
				t,
				handler,
				http.MethodPost,
				test.path,
			)
			if response.Code != test.wantStatus {
				t.Fatalf(
					"status = %d, want %d, body = %s",
					response.Code,
					test.wantStatus,
					response.Body.String(),
				)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode ||
				envelope.TraceID != administratorGovernanceTraceID {
				t.Fatalf("error envelope = %#v", envelope)
			}
			if service, ok := test.service.(*administratorGovernanceServiceStub); ok &&
				service.calls != test.wantCalls {
				t.Fatalf("RevokeAdministrator() calls = %d, want %d", service.calls, test.wantCalls)
			}
		})
	}
}

const (
	administratorGovernanceActorID  = "00000000-0000-0000-0000-000000000011"
	administratorGovernanceTargetID = "00000000-0000-0000-0000-000000000012"
	administratorGovernanceTraceID  = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

func newAdministratorGovernanceServiceForTest(
	t *testing.T,
	tx pgx.Tx,
	repository AdministratorGovernanceRepository,
	auditor AccountStatusAuditAppender,
	now time.Time,
) *AdministratorGovernanceCoordinator {
	t.Helper()
	service, err := NewAdministratorGovernanceService(
		AdministratorGovernanceDependencies{
			Transactions: &administratorGovernanceTransactorStub{
				transactions: []pgx.Tx{tx},
			},
			Repository: repository,
			Audit:      auditor,
			IDs:        &administratorGovernanceIDStub{},
			Clock:      administratorGovernanceClockStub{now: now},
		},
	)
	if err != nil {
		t.Fatalf("NewAdministratorGovernanceService() error = %v", err)
	}
	return service
}

func administratorGovernancePrincipal(initial bool) session.Principal {
	return session.Principal{
		UserID: administratorGovernanceActorID, Role: string(RoleAdmin),
		IsInitialAdmin: initial, Status: string(UserStatusActive),
	}
}

func administratorGovernanceActor(initial bool) User {
	completedAt := time.Date(2026, time.October, 26, 9, 0, 0, 0, time.UTC)
	return User{
		ID: administratorGovernanceActorID, Role: RoleAdmin, IsInitialAdmin: initial,
		Status: UserStatusActive, CredentialMode: CredentialModePermanent,
		OnboardingCompletedAt: &completedAt,
	}
}

func administratorGovernanceWritableRepository() *administratorGovernanceRepositoryStub {
	return &administratorGovernanceRepositoryStub{users: []User{
		administratorGovernanceActor(true),
		{ID: administratorGovernanceTargetID, Role: RoleAdmin, Status: UserStatusActive},
	}}
}

func serveAdministratorGovernanceRequest(
	t *testing.T,
	handler *Handler,
	method string,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	traceContext, err := platform.ContextWithTraceID(
		request.Context(),
		administratorGovernanceTraceID,
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request = request.WithContext(traceContext)
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	response := httptest.NewRecorder()
	authorizer, err := platform.NewHTTPAuthorizer(
		accountStatusPrincipalResolver{
			principal: administratorGovernancePrincipal(true),
		},
		func() time.Time {
			return time.Date(2026, time.October, 27, 9, 0, 0, 0, time.UTC)
		},
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.RevokeAdministrator(w, r)
		}),
	).ServeHTTP(response, request)
	return response
}

type administratorGovernanceTransactorStub struct {
	transactions []pgx.Tx
	next         int
	err          error
	returnNil    bool
}

func (s *administratorGovernanceTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.returnNil {
		return nil, nil
	}
	if s.next >= len(s.transactions) {
		return nil, errors.New("unexpected administrator governance transaction")
	}
	tx := s.transactions[s.next]
	s.next++
	return tx, nil
}

type administratorGovernanceTxStub struct {
	pgx.Tx
	commitCalls   int
	rollbackCalls int
	commitErr     error
}

func (s *administratorGovernanceTxStub) Commit(context.Context) error {
	s.commitCalls++
	return s.commitErr
}

func (s *administratorGovernanceTxStub) Rollback(context.Context) error {
	s.rollbackCalls++
	return nil
}

type administratorGovernanceRepositoryStub struct {
	users           []User
	lockErrs        []error
	lockCalls       int
	updateRoleCalls int
	updateOldRole   Role
	updateNewRole   Role
	updateRoleErr   error
	revokeCalls     int
	revokeReason    string
	revokeErr       error
}

func (s *administratorGovernanceRepositoryStub) LockUser(
	_ context.Context,
	_ pgx.Tx,
	_ string,
) (User, error) {
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

func (s *administratorGovernanceRepositoryStub) UpdateRole(
	_ context.Context,
	_ pgx.Tx,
	_ string,
	oldRole Role,
	newRole Role,
	_ time.Time,
) error {
	s.updateRoleCalls++
	s.updateOldRole = oldRole
	s.updateNewRole = newRole
	return s.updateRoleErr
}

func (s *administratorGovernanceRepositoryStub) RevokeSessions(
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

type administratorGovernanceAuditStub struct {
	event audit.Event
	tx    pgx.Tx
	calls int
	err   error
}

func (s *administratorGovernanceAuditStub) Append(
	_ context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	s.calls++
	s.tx = tx
	s.event = event
	return s.err
}

type administratorGovernanceIDStub struct {
	next      int
	calls     int
	result    string
	useResult bool
	err       error
}

func (s *administratorGovernanceIDStub) NewID() (string, error) {
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

type administratorGovernanceClockStub struct{ now time.Time }

func (s administratorGovernanceClockStub) Now() time.Time { return s.now }

type administratorGovernanceServiceStub struct {
	result  AdministratorRevokeResult
	err     error
	request AdministratorRevokeRequest
	calls   int
}

func (s *administratorGovernanceServiceStub) RevokeAdministrator(
	_ context.Context,
	request AdministratorRevokeRequest,
) (AdministratorRevokeResult, error) {
	s.calls++
	s.request = request
	return s.result, s.err
}

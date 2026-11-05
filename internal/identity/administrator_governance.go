package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	administratorGovernanceAuditAction         = "identity.role.revoke"
	administratorGovernanceSessionRevokeReason = "role_revoked"
)

// AdministratorRevokeRequest contains one authorized role-revocation operation.
type AdministratorRevokeRequest struct {
	Actor    session.Principal
	TargetID string
	TraceID  string
}

// AdministratorRevokeUser is the minimum revoked-administrator response.
type AdministratorRevokeUser struct {
	ID   string `json:"id"`
	Role Role   `json:"role"`
}

// AdministratorRevokeResult reports the persisted role after revocation.
type AdministratorRevokeResult struct {
	User AdministratorRevokeUser `json:"user"`
}

// AdministratorGovernanceRepository owns locked administrator role persistence.
type AdministratorGovernanceRepository interface {
	LockUser(context.Context, pgx.Tx, string) (User, error)
	UpdateRole(context.Context, pgx.Tx, string, Role, Role, time.Time) error
	RevokeSessions(context.Context, pgx.Tx, string, time.Time, string) error
}

// AdministratorGovernanceDependencies are the transaction-scoped collaborators.
type AdministratorGovernanceDependencies struct {
	Transactions RegistrationTransactor
	Repository   AdministratorGovernanceRepository
	Audit        AccountStatusAuditAppender
	IDs          RegistrationIDGenerator
	Clock        RegistrationClock
}

// AdministratorGovernanceCoordinator owns administrator role transitions.
type AdministratorGovernanceCoordinator struct {
	dependencies AdministratorGovernanceDependencies
}

// NewAdministratorGovernanceService creates the administrator governance coordinator.
func NewAdministratorGovernanceService(
	dependencies AdministratorGovernanceDependencies,
) (*AdministratorGovernanceCoordinator, error) {
	if nilIdentityDependency(dependencies.Transactions) ||
		nilIdentityDependency(dependencies.Repository) ||
		nilIdentityDependency(dependencies.Audit) ||
		nilIdentityDependency(dependencies.IDs) ||
		nilIdentityDependency(dependencies.Clock) {
		return nil, errors.New("administrator governance dependencies are incomplete")
	}
	return &AdministratorGovernanceCoordinator{dependencies: dependencies}, nil
}

// RevokeAdministrator converts one other administrator into an ordinary member.
func (c *AdministratorGovernanceCoordinator) RevokeAdministrator(
	ctx context.Context,
	request AdministratorRevokeRequest,
) (AdministratorRevokeResult, error) {
	if err := validateAdministratorRevokeRequest(request); err != nil {
		return AdministratorRevokeResult{}, err
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return AdministratorRevokeResult{}, administratorGovernanceInternalError(
			"begin administrator governance transaction",
			err,
		)
	}
	if tx == nil {
		return AdministratorRevokeResult{}, administratorGovernanceInternalError(
			"begin administrator governance transaction",
			errors.New("transaction is nil"),
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()

	result, err := c.revokeAdministratorInTransaction(ctx, tx, request)
	if err != nil {
		return AdministratorRevokeResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return AdministratorRevokeResult{}, administratorGovernanceInternalError(
			"commit administrator governance transaction",
			err,
		)
	}
	closed = true
	return result, nil
}

func (c *AdministratorGovernanceCoordinator) revokeAdministratorInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	request AdministratorRevokeRequest,
) (AdministratorRevokeResult, error) {
	actor, err := c.lockAdministratorGovernanceActor(ctx, tx, request)
	if err != nil {
		return AdministratorRevokeResult{}, err
	}
	target, err := c.lockAdministratorGovernanceTarget(ctx, tx, request, actor.ID)
	if err != nil {
		return AdministratorRevokeResult{}, err
	}
	now, auditID, err := c.administratorGovernanceAuditIdentity()
	if err != nil {
		return AdministratorRevokeResult{}, err
	}
	if err := c.persistAdministratorRevocation(
		ctx, tx, request, actor, target, now, auditID,
	); err != nil {
		return AdministratorRevokeResult{}, err
	}
	return AdministratorRevokeResult{
		User: AdministratorRevokeUser{ID: target.ID, Role: RoleMember},
	}, nil
}

func (c *AdministratorGovernanceCoordinator) lockAdministratorGovernanceActor(
	ctx context.Context,
	tx pgx.Tx,
	request AdministratorRevokeRequest,
) (User, error) {
	actor, err := c.dependencies.Repository.LockUser(ctx, tx, request.Actor.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, administratorGovernanceForbiddenError()
	}
	if err != nil {
		return User{}, administratorGovernanceInternalError(
			"lock administrator governance actor",
			err,
		)
	}
	// Reuse the single authorization policy: only the initial administrator may
	// govern another administrator.
	if err := authorizeAccountStatusActor(request.Actor, actor, RoleAdmin); err != nil {
		return User{}, err
	}
	return actor, nil
}

func (c *AdministratorGovernanceCoordinator) lockAdministratorGovernanceTarget(
	ctx context.Context,
	tx pgx.Tx,
	request AdministratorRevokeRequest,
	actorID string,
) (User, error) {
	if request.TargetID == actorID {
		return User{}, administratorGovernanceForbiddenError()
	}
	target, err := c.dependencies.Repository.LockUser(ctx, tx, request.TargetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, userNotFoundError()
	}
	if err != nil {
		return User{}, administratorGovernanceInternalError(
			"lock administrator governance target",
			err,
		)
	}
	if target.IsInitialAdmin {
		return User{}, accountStatusInitialAdminProtectedError()
	}
	if target.Role != RoleAdmin {
		return User{}, accountStatusInvalidStateError()
	}
	if target.Status == UserStatusDeleted || target.DeletedAt != nil {
		return User{}, accountStatusInvalidStateError()
	}
	return target, nil
}

func (c *AdministratorGovernanceCoordinator) administratorGovernanceAuditIdentity() (
	time.Time,
	string,
	error,
) {
	now := c.dependencies.Clock.Now().UTC()
	if now.IsZero() {
		return time.Time{}, "", administratorGovernanceInternalError(
			"read administrator governance time",
			errors.New("clock returned zero time"),
		)
	}
	auditID, err := c.dependencies.IDs.NewID()
	if err != nil || strings.TrimSpace(auditID) == "" {
		return time.Time{}, "", administratorGovernanceInternalError(
			"create administrator governance audit ID",
			err,
		)
	}
	return now, auditID, nil
}

func (c *AdministratorGovernanceCoordinator) persistAdministratorRevocation(
	ctx context.Context,
	tx pgx.Tx,
	request AdministratorRevokeRequest,
	actor User,
	target User,
	now time.Time,
	auditID string,
) error {
	if err := c.dependencies.Repository.UpdateRole(
		ctx, tx, target.ID, RoleAdmin, RoleMember, now,
	); err != nil {
		return administratorGovernanceInternalError("update administrator role", err)
	}
	if err := c.dependencies.Repository.RevokeSessions(
		ctx, tx, target.ID, now, administratorGovernanceSessionRevokeReason,
	); err != nil {
		return administratorGovernanceInternalError("revoke revoked administrator sessions", err)
	}
	event := administratorGovernanceAuditEvent(auditID, now, request, actor)
	if err := c.dependencies.Audit.Append(ctx, tx, event); err != nil {
		return administratorGovernanceInternalError("append administrator governance audit", err)
	}
	return nil
}

func validateAdministratorRevokeRequest(request AdministratorRevokeRequest) error {
	if request.Actor.UserID == "" ||
		request.Actor.Role != string(RoleAdmin) ||
		request.Actor.Status != string(UserStatusActive) ||
		request.Actor.Restricted() ||
		!validAccountStatusUserID(request.Actor.UserID) {
		return administratorGovernanceForbiddenError()
	}
	if !validAccountStatusUserID(request.TargetID) {
		return userNotFoundError()
	}
	if strings.TrimSpace(request.TraceID) == "" {
		return administratorGovernanceInternalError(
			"validate administrator governance trace ID",
			errors.New("trace ID is required"),
		)
	}
	return nil
}

func administratorGovernanceAuditEvent(
	id string,
	occurredAt time.Time,
	request AdministratorRevokeRequest,
	actor User,
) audit.Event {
	actorID := actor.ID
	actorRole := string(actor.Role)
	return audit.Event{
		ID: id, OccurredAt: occurredAt,
		ActorKind: audit.ActorUser, ActorUserID: &actorID, ActorRole: &actorRole,
		Action:     administratorGovernanceAuditAction,
		TargetType: accountStatusAuditTargetUser, TargetID: request.TargetID,
		Outcome: accountStatusAuditOutcomeChanged, TraceID: request.TraceID,
		Metadata: map[string]any{
			"previous_role": RoleAdmin,
			"target_role":   RoleMember,
		},
	}
}

func administratorGovernanceForbiddenError() error {
	return platform.NewPublicError(
		platform.CodeAuthForbidden,
		errors.New("administrator governance operation is forbidden"),
	)
}

func administratorGovernanceInternalError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown administrator governance failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

// PostgresAdministratorGovernanceRepository persists locked role transitions.
type PostgresAdministratorGovernanceRepository struct{}

// LockUser locks and returns the current administrator governance state.
func (PostgresAdministratorGovernanceRepository) LockUser(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	return PostgresAccountStatusRepository{}.LockUser(ctx, tx, userID)
}

// UpdateRole changes one locked administrator from the expected previous role.
func (PostgresAdministratorGovernanceRepository) UpdateRole(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	oldRole Role,
	newRole Role,
	updatedAt time.Time,
) error {
	if tx == nil {
		return errors.New("administrator governance transaction is required")
	}
	command, err := tx.Exec(
		ctx,
		updateAdministratorRoleSQL,
		userID,
		oldRole,
		newRole,
		updatedAt,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("administrator role update affected an unexpected row count")
	}
	return nil
}

// RevokeSessions revokes every still-valid session for the revoked administrator.
func (PostgresAdministratorGovernanceRepository) RevokeSessions(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	revokedAt time.Time,
	reason string,
) error {
	return PostgresAccountStatusRepository{}.RevokeSessions(
		ctx, tx, userID, revokedAt, reason,
	)
}

const updateAdministratorRoleSQL = `
	UPDATE users
	SET role = $3, updated_at = $4
	WHERE id = $1 AND role = $2 AND deleted_at IS NULL
`

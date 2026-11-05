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
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	accountStatusAuditOutcomeChanged  = "changed"
	accountStatusAuditOutcomeNoChange = "no_change"
	accountStatusAuditTargetUser      = "user"
	accountStatusSessionRevokeReason  = "account_suspended"
)

// AccountStatusRequest contains one authorized target-state operation.
type AccountStatusRequest struct {
	Actor      session.Principal
	TargetID   string
	TargetRole Role
	Status     UserStatus
	TraceID    string
}

// AccountStatusUser is the minimum account status response.
type AccountStatusUser struct {
	ID     string     `json:"id"`
	Status UserStatus `json:"status"`
}

// AccountStatusResult reports the target state and whether it changed.
type AccountStatusResult struct {
	User    AccountStatusUser `json:"user"`
	Changed bool              `json:"changed"`
}

// AccountStatusRepository owns locked user status persistence.
type AccountStatusRepository interface {
	LockUser(context.Context, pgx.Tx, string) (User, error)
	UpdateStatus(context.Context, pgx.Tx, string, UserStatus, UserStatus, time.Time) error
	RevokeSessions(context.Context, pgx.Tx, string, time.Time, string) error
}

// AccountStatusAuditAppender appends governance audit facts.
type AccountStatusAuditAppender interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

// AccountStatusDependencies are the transaction-scoped collaborators.
type AccountStatusDependencies struct {
	Transactions RegistrationTransactor
	Repository   AccountStatusRepository
	Audit        AccountStatusAuditAppender
	IDs          RegistrationIDGenerator
	Clock        RegistrationClock
}

// AccountStatusCoordinator owns account status transitions.
type AccountStatusCoordinator struct {
	dependencies AccountStatusDependencies
}

// NewAccountStatusService creates the account governance coordinator.
func NewAccountStatusService(
	dependencies AccountStatusDependencies,
) (*AccountStatusCoordinator, error) {
	if dependencies.Transactions == nil || dependencies.Repository == nil ||
		dependencies.Audit == nil || dependencies.IDs == nil || dependencies.Clock == nil {
		return nil, errors.New("account status dependencies are incomplete")
	}
	return &AccountStatusCoordinator{dependencies: dependencies}, nil
}

// SetAccountStatus applies one target-state operation.
func (c *AccountStatusCoordinator) SetAccountStatus(
	ctx context.Context,
	request AccountStatusRequest,
) (AccountStatusResult, error) {
	if err := validateAccountStatusRequest(request); err != nil {
		return AccountStatusResult{}, err
	}
	tx, err := c.beginAccountStatusTransaction(ctx)
	if err != nil {
		return AccountStatusResult{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()

	result, err := c.setAccountStatusInTransaction(ctx, tx, request)
	if err != nil {
		return AccountStatusResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return AccountStatusResult{}, accountStatusInternalError(
			"commit account status transaction",
			err,
		)
	}
	closed = true
	return result, nil
}

func (c *AccountStatusCoordinator) beginAccountStatusTransaction(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return nil, accountStatusInternalError("begin account status transaction", err)
	}
	if tx == nil {
		return nil, accountStatusInternalError(
			"begin account status transaction",
			errors.New("transaction is nil"),
		)
	}
	return tx, nil
}

func (c *AccountStatusCoordinator) setAccountStatusInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	request AccountStatusRequest,
) (AccountStatusResult, error) {
	actor, err := c.lockAccountStatusActor(ctx, tx, request)
	if err != nil {
		return AccountStatusResult{}, err
	}
	target, err := c.lockAccountStatusTarget(ctx, tx, request, actor.ID)
	if err != nil {
		return AccountStatusResult{}, err
	}
	changed, err := accountStatusTransitionRequired(target.Status, request.Status)
	if err != nil {
		return AccountStatusResult{}, err
	}
	now, auditID, err := c.accountStatusAuditIdentity()
	if err != nil {
		return AccountStatusResult{}, err
	}
	if err := c.persistAccountStatusChange(
		ctx, tx, request, actor, target, changed, now, auditID,
	); err != nil {
		return AccountStatusResult{}, err
	}
	return AccountStatusResult{
		User:    AccountStatusUser{ID: target.ID, Status: request.Status},
		Changed: changed,
	}, nil
}

func (c *AccountStatusCoordinator) lockAccountStatusActor(
	ctx context.Context,
	tx pgx.Tx,
	request AccountStatusRequest,
) (User, error) {
	actor, err := c.dependencies.Repository.LockUser(ctx, tx, request.Actor.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, accountStatusForbiddenError()
	}
	if err != nil {
		return User{}, accountStatusInternalError("lock account status actor", err)
	}
	if err := authorizeAccountStatusActor(request.Actor, actor, request.TargetRole); err != nil {
		return User{}, err
	}
	return actor, nil
}

func (c *AccountStatusCoordinator) lockAccountStatusTarget(
	ctx context.Context,
	tx pgx.Tx,
	request AccountStatusRequest,
	actorID string,
) (User, error) {
	if request.TargetID == actorID {
		return User{}, accountStatusForbiddenError()
	}
	target, err := c.dependencies.Repository.LockUser(ctx, tx, request.TargetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, userNotFoundError()
	}
	if err != nil {
		return User{}, accountStatusInternalError("lock account status target", err)
	}
	if target.Role != request.TargetRole {
		return User{}, userNotFoundError()
	}
	if target.IsInitialAdmin {
		return User{}, accountStatusInitialAdminProtectedError()
	}
	if target.Status == UserStatusDeleted || target.DeletedAt != nil {
		return User{}, accountStatusInvalidStateError()
	}
	return target, nil
}

func (c *AccountStatusCoordinator) accountStatusAuditIdentity() (time.Time, string, error) {
	now := c.dependencies.Clock.Now().UTC()
	if now.IsZero() {
		return time.Time{}, "", accountStatusInternalError(
			"read account status time",
			errors.New("clock returned zero time"),
		)
	}
	auditID, err := c.dependencies.IDs.NewID()
	if err != nil || strings.TrimSpace(auditID) == "" {
		return time.Time{}, "", accountStatusInternalError(
			"create account status audit ID",
			err,
		)
	}
	return now, auditID, nil
}

func (c *AccountStatusCoordinator) persistAccountStatusChange(
	ctx context.Context,
	tx pgx.Tx,
	request AccountStatusRequest,
	actor User,
	target User,
	changed bool,
	now time.Time,
	auditID string,
) error {
	if changed {
		if err := c.dependencies.Repository.UpdateStatus(
			ctx, tx, target.ID, target.Status, request.Status, now,
		); err != nil {
			return accountStatusInternalError("update account status", err)
		}
		if request.Status == UserStatusSuspended {
			if err := c.dependencies.Repository.RevokeSessions(
				ctx, tx, target.ID, now, accountStatusSessionRevokeReason,
			); err != nil {
				return accountStatusInternalError("revoke suspended account sessions", err)
			}
		}
	}
	event := accountStatusAuditEvent(auditID, now, request, actor, target.Status, changed)
	if err := c.dependencies.Audit.Append(ctx, tx, event); err != nil {
		return accountStatusInternalError("append account status audit", err)
	}
	return nil
}

func accountStatusAuditAction(status UserStatus) string {
	if status == UserStatusActive {
		return "identity.account.restore"
	}
	return "identity.account.suspend"
}

func validateAccountStatusRequest(request AccountStatusRequest) error {
	if request.Actor.UserID == "" ||
		request.Actor.Role != string(RoleAdmin) ||
		request.Actor.Status != string(UserStatusActive) ||
		request.Actor.Restricted() {
		return accountStatusForbiddenError()
	}
	if request.TargetRole != RoleMember && request.TargetRole != RoleAdmin {
		return accountStatusForbiddenError()
	}
	if request.Status != UserStatusActive && request.Status != UserStatusSuspended {
		return accountStatusInvalidStateError()
	}
	if strings.TrimSpace(request.TraceID) == "" {
		return accountStatusInternalError(
			"validate account status trace ID",
			errors.New("trace ID is required"),
		)
	}
	if !validAccountStatusUserID(request.Actor.UserID) {
		return accountStatusForbiddenError()
	}
	if !validAccountStatusUserID(request.TargetID) {
		return userNotFoundError()
	}
	return nil
}

func authorizeAccountStatusActor(
	principal session.Principal,
	actor User,
	targetRole Role,
) error {
	if actor.ID != principal.UserID ||
		actor.Role != RoleAdmin ||
		actor.IsInitialAdmin != principal.IsInitialAdmin ||
		actor.Status != UserStatusActive ||
		actor.DeletedAt != nil ||
		actor.RecoveryRequired ||
		!accountStatusCredentialModeAllowed(actor.CredentialMode) {
		return accountStatusForbiddenError()
	}
	if !actor.IsInitialAdmin && actor.OnboardingCompletedAt == nil {
		return accountStatusForbiddenError()
	}
	if targetRole == RoleAdmin && !actor.IsInitialAdmin {
		return accountStatusForbiddenError()
	}
	return nil
}

func accountStatusCredentialModeAllowed(mode CredentialMode) bool {
	switch mode {
	case CredentialModePermanent:
		return true
	default:
		return false
	}
}

func accountStatusTransitionRequired(
	current UserStatus,
	target UserStatus,
) (bool, error) {
	if current == UserStatusDeleted {
		return false, accountStatusInvalidStateError()
	}
	if current == target {
		return false, nil
	}
	if (current == UserStatusActive && target == UserStatusSuspended) ||
		(current == UserStatusSuspended && target == UserStatusActive) {
		return true, nil
	}
	return false, accountStatusInvalidStateError()
}

func accountStatusAuditEvent(
	id string,
	occurredAt time.Time,
	request AccountStatusRequest,
	actor User,
	previousStatus UserStatus,
	changed bool,
) audit.Event {
	actorID := actor.ID
	actorRole := string(actor.Role)
	outcome := accountStatusAuditOutcomeNoChange
	if changed {
		outcome = accountStatusAuditOutcomeChanged
	}
	return audit.Event{
		ID: id, OccurredAt: occurredAt,
		ActorKind: audit.ActorUser, ActorUserID: &actorID, ActorRole: &actorRole,
		Action:     accountStatusAuditAction(request.Status),
		TargetType: accountStatusAuditTargetUser, TargetID: request.TargetID,
		Outcome: outcome, TraceID: request.TraceID,
		Metadata: map[string]any{
			"previous_status": previousStatus,
			"target_status":   request.Status,
			"changed":         changed,
		},
	}
}

func validAccountStatusUserID(value string) bool {
	var id pgtype.UUID
	return id.Scan(value) == nil && id.Valid
}

func accountStatusForbiddenError() error {
	return platform.NewPublicError(
		platform.CodeAuthForbidden,
		errors.New("account status operation is forbidden"),
	)
}

func accountStatusInitialAdminProtectedError() error {
	return platform.NewPublicError(
		platform.CodeIdentityInitialAdminProtected,
		errors.New("initial administrator account is protected"),
	)
}

func accountStatusInvalidStateError() error {
	return platform.NewPublicError(
		platform.CodeIdentityInvalidState,
		errors.New("account status transition is invalid"),
	)
}

func accountStatusInternalError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown account status failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

// PostgresAccountStatusRepository persists locked account status transitions.
type PostgresAccountStatusRepository struct{}

// LockUser locks and returns the current user governance state.
func (PostgresAccountStatusRepository) LockUser(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	if tx == nil {
		return User{}, errors.New("account status transaction is required")
	}
	var user User
	err := tx.QueryRow(ctx, lockAccountStatusUserSQL, userID).Scan(
		&user.ID,
		&user.Role,
		&user.IsInitialAdmin,
		&user.Status,
		&user.CredentialMode,
		&user.RecoveryRequired,
		&user.OnboardingCompletedAt,
		&user.DeletedAt,
	)
	return user, err
}

// UpdateStatus changes one locked non-deleted user from the expected old state.
func (PostgresAccountStatusRepository) UpdateStatus(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	oldStatus UserStatus,
	newStatus UserStatus,
	updatedAt time.Time,
) error {
	if tx == nil {
		return errors.New("account status transaction is required")
	}
	command, err := tx.Exec(
		ctx,
		updateAccountStatusSQL,
		userID,
		oldStatus,
		newStatus,
		updatedAt,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("account status update affected an unexpected row count")
	}
	return nil
}

// RevokeSessions revokes every still-valid session for the suspended user.
func (PostgresAccountStatusRepository) RevokeSessions(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	revokedAt time.Time,
	reason string,
) error {
	if tx == nil {
		return errors.New("account status transaction is required")
	}
	_, err := tx.Exec(
		ctx,
		revokeAccountStatusSessionsSQL,
		userID,
		revokedAt,
		reason,
	)
	return err
}

const lockAccountStatusUserSQL = `
	SELECT
		id,
		role,
		is_initial_admin,
		status,
		credential_mode,
		recovery_required,
		onboarding_completed_at,
		deleted_at
	FROM users
	WHERE id = $1
	FOR UPDATE
`

const updateAccountStatusSQL = `
	UPDATE users
	SET status = $3, updated_at = $4
	WHERE id = $1 AND status = $2 AND deleted_at IS NULL
`

const revokeAccountStatusSessionsSQL = `
	UPDATE sessions
	SET revoked_at = $2, revoke_reason = $3
	WHERE user_id = $1 AND revoked_at IS NULL
`

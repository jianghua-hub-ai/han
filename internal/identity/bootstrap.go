package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	initialAdminUsername    = "admin"
	initialAdminLockScope   = "han.identity.initial-admin-bootstrap.v1"
	initialAdminAuditAction = "identity.initial_admin_bootstrap"
)

// InitialAdminPasswordProvider reads the bootstrap secret only when creation is required.
type InitialAdminPasswordProvider func(context.Context) (string, error)

// InitialAdminRepository persists the bootstrap identity aggregate.
type InitialAdminRepository interface {
	InsertUser(context.Context, pgx.Tx, User) error
	InsertIdentifier(context.Context, pgx.Tx, IdentityIdentifier) error
}

// InitialAdminAuditAppender appends the bootstrap event in the same transaction.
type InitialAdminAuditAppender interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

// InitialAdminDependencies are the explicit bootstrap collaborators.
type InitialAdminDependencies struct {
	Transactions   RegistrationTransactor
	Password       InitialAdminPasswordProvider
	PasswordPolicy RegistrationPasswordPolicy
	PasswordHasher RegistrationPasswordHasher
	Repository     InitialAdminRepository
	Audit          InitialAdminAuditAppender
	IDs            RegistrationIDGenerator
	Clock          RegistrationClock
}

// InitialAdminTrustManager creates the single fixed initial administrator.
type InitialAdminTrustManager struct {
	dependencies InitialAdminDependencies
}

// NewInitialAdminTrustManager creates a fail-closed bootstrap manager.
func NewInitialAdminTrustManager(
	dependencies InitialAdminDependencies,
) (*InitialAdminTrustManager, error) {
	if dependencies.Transactions == nil || dependencies.Password == nil ||
		dependencies.PasswordPolicy == nil || dependencies.PasswordHasher == nil ||
		dependencies.Repository == nil || dependencies.Audit == nil ||
		dependencies.IDs == nil || dependencies.Clock == nil {
		return nil, errors.New("initial admin bootstrap dependencies are incomplete")
	}
	return &InitialAdminTrustManager{dependencies: dependencies}, nil
}

// EnsureInitialAdmin creates admin exactly once and never rereads its secret afterward.
func (m *InitialAdminTrustManager) EnsureInitialAdmin(ctx context.Context) error {
	if m == nil {
		return errors.New("initial admin bootstrap manager is not initialized")
	}
	tx, err := m.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin initial admin bootstrap transaction: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err := tx.Exec(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		initialAdminLockScope,
	); err != nil {
		return fmt.Errorf("lock initial admin bootstrap: %w", err)
	}

	var initialCount, usernameCount, validCount int64
	if err := tx.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE is_initial_admin),
			count(*) FILTER (WHERE username = $1),
			count(*) FILTER (
				WHERE is_initial_admin
					AND username = $1
					AND role = 'admin'
					AND status <> 'deleted'
			)
		FROM users
		WHERE is_initial_admin OR username = $1
	`, initialAdminUsername).Scan(&initialCount, &usernameCount, &validCount); err != nil {
		return fmt.Errorf("inspect initial admin bootstrap state: %w", err)
	}

	switch {
	case initialCount == 1 && usernameCount == 1 && validCount == 1:
		if err := tx.Commit(ctx); err != nil {
			closed = true
			return fmt.Errorf("commit existing initial admin bootstrap check: %w", err)
		}
		closed = true
		return nil
	case initialCount != 0 || usernameCount != 0 || validCount != 0:
		return errors.New("initial admin bootstrap state conflicts with fixed admin identity")
	}

	password, err := m.dependencies.Password(ctx)
	if err != nil {
		return fmt.Errorf("read initial admin bootstrap password: %w", err)
	}
	if password == "" {
		return errors.New("initial admin bootstrap password is required")
	}
	if err := m.dependencies.PasswordPolicy.Validate(password); err != nil {
		return fmt.Errorf("validate initial admin bootstrap password: %w", err)
	}
	digest, err := m.dependencies.PasswordHasher.Hash(password)
	password = ""
	if err != nil {
		return fmt.Errorf("hash initial admin bootstrap password: %w", err)
	}
	if strings.TrimSpace(digest) == "" {
		return errors.New("initial admin bootstrap password digest is empty")
	}

	userID, err := m.dependencies.IDs.NewID()
	if err != nil {
		return fmt.Errorf("create initial admin identifier: %w", err)
	}
	auditID, err := m.dependencies.IDs.NewID()
	if err != nil {
		return fmt.Errorf("create initial admin audit identifier: %w", err)
	}
	now := m.dependencies.Clock.Now().UTC()
	user := User{
		ID: userID, Username: initialAdminUsername,
		PermanentPasswordDigest: &digest,
		Role:                    RoleAdmin, IsInitialAdmin: true, Status: UserStatusActive,
		CredentialMode: CredentialModeBootstrap,
		CreatedAt:      now, UpdatedAt: now,
	}
	if err := m.dependencies.Repository.InsertUser(ctx, tx, user); err != nil {
		return fmt.Errorf("insert initial admin: %w", err)
	}
	if err := m.dependencies.Repository.InsertIdentifier(ctx, tx, IdentityIdentifier{
		UserID: userID, Kind: IdentifierUsername,
		NormalizedValue: initialAdminUsername, Lifecycle: IdentifierActive,
		CreatedAt: now,
	}); err != nil {
		return fmt.Errorf("insert initial admin username: %w", err)
	}
	system := "han-server"
	if err := m.dependencies.Audit.Append(ctx, tx, audit.Event{
		ID: auditID, OccurredAt: now, ActorKind: audit.ActorSystem,
		ActorSystem: &system, Action: initialAdminAuditAction,
		TargetType: "user", TargetID: userID, Outcome: "succeeded",
		TraceID: auditID,
	}); err != nil {
		return fmt.Errorf("append initial admin bootstrap audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return fmt.Errorf("commit initial admin bootstrap: %w", err)
	}
	closed = true
	return nil
}

// PostgresPasswordChangeRepository persists the bounded password-change state.
type PostgresPasswordChangeRepository struct{}

// PostgresPasswordChangeSessions adapts the shared session store with bulk revocation.
type PostgresPasswordChangeSessions struct {
	Store session.Store
}

// Revoke delegates current-session revocation to the shared store.
func (s PostgresPasswordChangeSessions) Revoke(
	ctx context.Context,
	executor session.Executor,
	token string,
	revokedAt time.Time,
	reason string,
) error {
	return s.Store.Revoke(ctx, executor, token, revokedAt, reason)
}

// Rotate delegates atomic current-session rotation to the shared store.
func (s PostgresPasswordChangeSessions) Rotate(
	ctx context.Context,
	tx pgx.Tx,
	oldToken string,
	revokedAt time.Time,
	reason string,
	replacement session.CreateInput,
) (session.Credentials, error) {
	return s.Store.Rotate(ctx, tx, oldToken, revokedAt, reason, replacement)
}

// RevokeOtherSessions revokes every session except the replacement.
func (PostgresPasswordChangeSessions) RevokeOtherSessions(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	replacementSessionID string,
	revokedAt time.Time,
	reason string,
) error {
	if tx == nil {
		return errors.New("password change transaction is required")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = $3, revoke_reason = $4
		WHERE user_id = $1
			AND id <> $2
			AND revoked_at IS NULL
	`, userID, replacementSessionID, revokedAt, reason); err != nil {
		return fmt.Errorf("revoke other password sessions: %w", err)
	}
	return nil
}

// FindUserForPasswordChange locks the current authoritative user state.
func (PostgresPasswordChangeRepository) FindUserForPasswordChange(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	if tx == nil {
		return User{}, errors.New("password change transaction is required")
	}
	var user User
	err := tx.QueryRow(ctx, `
		SELECT
			id, username, email, phone, permanent_password_digest,
			auth_generation, permanent_password_generation,
			password_reset_execution_generation,
			password_reset_current_operation_id,
			permanent_password_source_operation_id,
			permanent_password_source_credential_id,
			auth_integrity_hold, role, is_initial_admin, status,
			credential_mode, recovery_required,
			onboarding_privacy_acknowledged_version, onboarding_completed_at,
			retained_email, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone,
		&user.PermanentPasswordDigest, &user.AuthGeneration,
		&user.PermanentPasswordGeneration, &user.PasswordResetExecutionGeneration,
		&user.PasswordResetCurrentOperationID,
		&user.PermanentPasswordSourceOperationID,
		&user.PermanentPasswordSourceCredentialID,
		&user.AuthIntegrityHold, &user.Role, &user.IsInitialAdmin,
		&user.Status, &user.CredentialMode, &user.RecoveryRequired,
		&user.OnboardingPrivacyAcknowledgedVersion, &user.OnboardingCompletedAt,
		&user.RetainedEmail, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	return user, err
}

// FindPasswordResetOperationForPasswordChange locks the exact reset authority first.
func (PostgresPasswordChangeRepository) FindPasswordResetOperationForPasswordChange(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	operationID string,
	credentialID string,
) (PasswordResetOperation, error) {
	if tx == nil {
		return PasswordResetOperation{}, errors.New("password change transaction is required")
	}
	var operation PasswordResetOperation
	err := tx.QueryRow(ctx, findPasswordChangeOperationSQL,
		operationID, userID, credentialID,
	).Scan(
		&operation.ID, &operation.UserID, &operation.TemporaryCredentialID,
		&operation.BaseAuthGeneration, &operation.TargetAuthGeneration,
		&operation.SettlementState, &operation.SettledAuthGeneration,
		&operation.SuccessorAuthGeneration, &operation.SettledAt,
		&operation.DeliveryState, &operation.State,
	)
	return operation, err
}

// FindLatestTemporaryCredential locks the latest credential generation.
func (PostgresPasswordChangeRepository) FindLatestTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (TemporaryCredential, error) {
	if tx == nil {
		return TemporaryCredential{}, errors.New("password change transaction is required")
	}
	var credential TemporaryCredential
	err := tx.QueryRow(ctx, `
		SELECT
			id, user_id, purpose, generation, auth_generation,
			source_operation_id, state, secret_digest,
			delivered_to_email, issued_at, expires_at, superseded_at,
			consumed_at, invalidated_at
		FROM temporary_credentials
		WHERE user_id = $1
		ORDER BY generation DESC
		LIMIT 1
		FOR UPDATE
	`, userID).Scan(
		&credential.ID, &credential.UserID, &credential.Purpose,
		&credential.Generation, &credential.AuthGeneration,
		&credential.SourceOperationID, &credential.State, &credential.SecretDigest,
		&credential.DeliveredToEmail, &credential.IssuedAt, &credential.ExpiresAt,
		&credential.SupersededAt, &credential.ConsumedAt, &credential.InvalidatedAt,
	)
	return credential, err
}

// SetPermanentPassword stores the replacement digest and completes the password gate.
func (PostgresPasswordChangeRepository) SetPermanentPassword(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	digest string,
	mode CredentialMode,
	expectedGeneration int64,
	successorGeneration int64,
	sourceOperationID *string,
	sourceCredentialID *string,
	updatedAt time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE users
		SET permanent_password_digest = $2,
			credential_mode = $3,
			auth_generation = $5,
			permanent_password_generation = $5,
			permanent_password_source_operation_id = $6,
			permanent_password_source_credential_id = $7,
			updated_at = $8
		WHERE id = $1
			AND status = 'active'
			AND recovery_required = false
			AND auth_generation = $4
	`, userID, digest, mode, expectedGeneration, successorGeneration,
		sourceOperationID, sourceCredentialID, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("password change user state changed")
	}
	return nil
}

// SettlePasswordResetSuccessor atomically closes exact reset lineage.
func (PostgresPasswordChangeRepository) SettlePasswordResetSuccessor(
	ctx context.Context,
	tx pgx.Tx,
	operation PasswordResetOperation,
	credential TemporaryCredential,
	successorGeneration int64,
	settledAt time.Time,
) error {
	tag, err := tx.Exec(ctx, settlePasswordChangeOperationSQL,
		operation.ID, operation.UserID, credential.ID,
		operation.TargetAuthGeneration, successorGeneration, settledAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("password reset operation settlement changed")
	}
	tag, err = tx.Exec(ctx, consumePasswordResetCredentialSQL,
		operation.ID, operation.UserID, credential.ID,
		operation.TargetAuthGeneration, settledAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("password reset credential settlement changed")
	}
	return nil
}

// ConsumeTemporaryCredential consumes the exact credential used by this session.
func (PostgresPasswordChangeRepository) ConsumeTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credentialID string,
	consumedAt time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE temporary_credentials
		SET state = 'consumed', consumed_at = $2
		WHERE id = $1
			AND state = 'active'
			AND expires_at > $2
	`, credentialID, consumedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("temporary credential state changed")
	}
	return nil
}

const findPasswordChangeOperationSQL = `
	SELECT
		operation_id,
		user_id,
		temporary_credential_id,
		base_auth_generation,
		target_auth_generation,
		settlement_state,
		settled_auth_generation,
		successor_auth_generation,
		settled_at,
		delivery_state,
		state
	FROM password_reset_operations
	WHERE operation_id = $1
		AND user_id = $2
		AND temporary_credential_id = $3
	FOR UPDATE
`

const settlePasswordChangeOperationSQL = `
	UPDATE password_reset_operations
	SET settlement_state = 'permanent_successor',
		successor_auth_generation = $5,
		settled_at = $6,
		updated_at = $6
	WHERE operation_id = $1
		AND user_id = $2
		AND temporary_credential_id = $3
		AND target_auth_generation = $4
		AND state = 'succeeded'
		AND delivery_state = 'accepted'
		AND settlement_state = 'temporary_active'
		AND settled_auth_generation = target_auth_generation
		AND successor_auth_generation IS NULL
`

const consumePasswordResetCredentialSQL = `
	UPDATE temporary_credentials
	SET state = 'consumed', consumed_at = $5
	WHERE source_operation_id = $1
		AND user_id = $2
		AND id = $3
		AND auth_generation = $4
		AND purpose = 'password_reset'
		AND state = 'active'
		AND expires_at > $5
`

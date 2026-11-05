// Package session owns server-side session persistence and trusted principals.
package session

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrSessionRequired hides missing, expired, revoked, and otherwise invalid sessions.
	ErrSessionRequired = errors.New("session required")
	// ErrCSRFFailed identifies a missing or mismatched session-bound CSRF token.
	ErrCSRFFailed = errors.New("CSRF validation failed")
	// ErrStoreUnavailable identifies a session persistence failure.
	ErrStoreUnavailable = errors.New("session store unavailable")
	// ErrPasswordResetGateClosed identifies a durable authentication barrier.
	ErrPasswordResetGateClosed = errors.New("password reset gate is closed")
)

// QueryRower executes one row-returning query.
type QueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// StoreTransactor supports token pre-read and caller-owned transactions.
type StoreTransactor interface {
	QueryRower
	Begin(context.Context) (pgx.Tx, error)
}

// PasswordResetGate checks the already locked User before authentication.
type PasswordResetGate interface {
	CheckLockedUser(context.Context, pgx.Tx, string) error
}

// StoreDependencies are the mandatory session authorization boundaries.
type StoreDependencies struct {
	Transactions      StoreTransactor
	PasswordResetGate PasswordResetGate
}

// Executor executes one statement in a caller-owned transaction.
type Executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// CredentialKind records which credential authenticated the session.
type CredentialKind string

const (
	CredentialPermanent         CredentialKind = "permanent"
	CredentialBootstrap         CredentialKind = "bootstrap"
	CredentialTemporary         CredentialKind = "temporary"
	CredentialRecoveryBootstrap CredentialKind = "recovery_bootstrap"
)

// Access identifies one protected capability at the HTTP boundary.
type Access string

const (
	AccessGeneral              Access = "general"
	AccessCurrentIdentity      Access = "current_identity"
	AccessChangePassword       Access = "change_password"
	AccessCurrentPrivacyNotice Access = "current_privacy_notice"
	AccessAcknowledgePrivacy   Access = "acknowledge_privacy"
)

// CreateInput contains all caller-selected session values, including explicit lifetimes.
type CreateInput struct {
	ID                          string
	UserID                      string
	Token                       string
	CSRFToken                   string
	AuthenticatedCredentialKind CredentialKind
	AuthenticatedCredentialID   *string
	AuthGeneration              int64
	LineageOperationID          *string
	LineageCredentialID         *string
	IssuedAt                    time.Time
	AbsoluteExpiresAt           time.Time
	IdleExpiresAt               time.Time
	RotatedFromID               *string
}

// Credentials are the raw values returned only to the required transport boundary.
type Credentials struct {
	Token     string
	CSRFToken string
}

// ResolveInput contains request-time session evidence.
type ResolveInput struct {
	Token       string
	CSRFToken   string
	RequireCSRF bool
	Now         time.Time
}

// Principal is the minimum trusted identity context exposed to business handlers.
type Principal struct {
	maintenance               bool
	SessionID                 string
	UserID                    string
	Role                      string
	IsInitialAdmin            bool
	Status                    string
	MustChangePassword        bool
	MustAcknowledgePrivacy    bool
	AuthenticatedCredential   CredentialKind
	AuthenticatedCredentialID *string
}

// Restricted reports whether onboarding limits the principal's capabilities.
func (p Principal) Restricted() bool {
	return p.MustChangePassword || p.MustAcknowledgePrivacy
}

// Allows reports whether the principal may enter one protected capability.
func (p Principal) Allows(access Access) bool {
	if p.UserID == "" || p.Status != "active" {
		return false
	}
	if p.maintenance && p.MustChangePassword {
		return access == AccessCurrentIdentity || access == AccessChangePassword
	}
	switch access {
	case AccessCurrentIdentity, AccessChangePassword, AccessCurrentPrivacyNotice,
		AccessAcknowledgePrivacy:
		return true
	case AccessGeneral:
		return !p.Restricted()
	default:
		return false
	}
}

// Store persists and resolves PostgreSQL-backed sessions.
type Store struct {
	transactions      StoreTransactor
	passwordResetGate PasswordResetGate
}

// NewStore creates the only Gate-aware SessionStore.
func NewStore(dependencies StoreDependencies) (Store, error) {
	if nilDependency(dependencies.Transactions) ||
		nilDependency(dependencies.PasswordResetGate) {
		return Store{}, errors.New("session store dependencies are incomplete")
	}
	return Store{
		transactions:      dependencies.Transactions,
		passwordResetGate: dependencies.PasswordResetGate,
	}, nil
}

func nilDependency(dependency any) bool {
	if dependency == nil {
		return true
	}
	value := reflect.ValueOf(dependency)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Create inserts one session while persisting only token and CSRF digests.
func (Store) Create(ctx context.Context, executor Executor, input CreateInput) (Credentials, error) {
	if err := validateCreateInput(executor, input); err != nil {
		return Credentials{}, err
	}

	tokenDigest := digest(input.Token)
	csrfDigest := digest(input.CSRFToken)
	if _, err := executor.Exec(
		ctx,
		insertSessionSQL,
		input.ID,
		input.UserID,
		tokenDigest,
		csrfDigest,
		input.AuthenticatedCredentialKind,
		input.AuthenticatedCredentialID,
		input.AuthGeneration,
		input.LineageOperationID,
		input.LineageCredentialID,
		input.IssuedAt,
		input.IssuedAt,
		input.AbsoluteExpiresAt,
		input.IdleExpiresAt,
		input.RotatedFromID,
	); err != nil {
		return Credentials{}, fmt.Errorf("%w: insert session: %w", ErrStoreUnavailable, err)
	}
	return Credentials{Token: input.Token, CSRFToken: input.CSRFToken}, nil
}

// Rotate revokes an old token and inserts its replacement in a caller-owned transaction.
func (s Store) Rotate(
	ctx context.Context,
	transaction pgx.Tx,
	oldToken string,
	revokedAt time.Time,
	reason string,
	replacement CreateInput,
) (Credentials, error) {
	if transaction == nil {
		return Credentials{}, errors.New("session transaction is required")
	}
	if err := validateCreateInput(transaction, replacement); err != nil {
		return Credentials{}, err
	}

	if err := s.Revoke(ctx, transaction, oldToken, revokedAt, reason); err != nil {
		return Credentials{}, err
	}
	credentials, err := s.Create(ctx, transaction, replacement)
	if err != nil {
		return Credentials{}, err
	}
	return credentials, nil
}

// Revoke invalidates one token without persisting its raw value.
func (Store) Revoke(
	ctx context.Context,
	executor Executor,
	token string,
	revokedAt time.Time,
	reason string,
) error {
	if executor == nil {
		return errors.New("session transaction is required")
	}
	if token == "" {
		return errors.New("session token is required")
	}
	if revokedAt.IsZero() {
		return errors.New("session revocation time is required")
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("session revocation reason is required")
	}

	tokenDigest := digest(token)
	if _, err := executor.Exec(ctx, revokeSessionSQL, tokenDigest, revokedAt, reason); err != nil {
		return fmt.Errorf("%w: revoke session: %w", ErrStoreUnavailable, err)
	}
	return nil
}

// ResolvePrincipal validates the current session, optional CSRF token, and user state.
func (s Store) ResolvePrincipal(ctx context.Context, input ResolveInput) (Principal, error) {
	if input.Token == "" || input.Now.IsZero() {
		return Principal{}, ErrSessionRequired
	}
	userID, err := s.preReadSessionUser(ctx, input.Token)
	if err != nil {
		return Principal{}, err
	}
	tx, err := s.transactions.Begin(ctx)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: begin resolve principal: %w", ErrStoreUnavailable, err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	record, rejected, err := s.resolveLockedPrincipal(ctx, tx, input, userID)
	if err != nil {
		return Principal{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return Principal{}, resolveCommitError(err, rejected)
	}
	closed = true
	if rejected {
		return Principal{}, ErrSessionRequired
	}
	return record.principalValue(), nil
}

func (s Store) resolveLockedPrincipal(
	ctx context.Context,
	tx pgx.Tx,
	input ResolveInput,
	userID string,
) (resolvedSession, bool, error) {
	if err := lockSessionUser(ctx, tx, userID); err != nil {
		return resolvedSession{}, false, err
	}
	if err := s.passwordResetGate.CheckLockedUser(ctx, tx, userID); err != nil {
		if errors.Is(err, ErrPasswordResetGateClosed) {
			return resolvedSession{}, false, ErrSessionRequired
		}
		return resolvedSession{}, false, fmt.Errorf(
			"%w: check password reset gate: %w",
			ErrStoreUnavailable,
			err,
		)
	}
	record, err := readResolvedSession(ctx, tx, input, userID)
	if err != nil {
		return resolvedSession{}, false, err
	}
	if !record.authorizationValid {
		err = s.Revoke(ctx, tx, input.Token, input.Now, "authorization_state_invalid")
		return resolvedSession{}, true, err
	}
	if input.RequireCSRF && !record.matchesCSRF(input.CSRFToken) {
		return resolvedSession{}, false, ErrCSRFFailed
	}
	return record, false, nil
}

func resolveCommitError(err error, rejected bool) error {
	operation := "commit resolve principal"
	if rejected {
		operation = "commit invalid session revocation"
	}
	return fmt.Errorf("%w: %s: %w", ErrStoreUnavailable, operation, err)
}

type resolvedSession struct {
	principal                   Principal
	credentialKind              string
	credentialMode              string
	acknowledgedPrivacyVersion  *string
	onboardingCompletedAt       *time.Time
	currentPrivacyNoticeVersion *string
	csrfDigest                  []byte
	authorizationValid          bool
}

func (s Store) preReadSessionUser(ctx context.Context, token string) (string, error) {
	var userID string
	err := s.transactions.QueryRow(ctx, preReadSessionUserSQL, digest(token)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSessionRequired
	}
	if err != nil {
		return "", fmt.Errorf("%w: pre-read session: %w", ErrStoreUnavailable, err)
	}
	if strings.TrimSpace(userID) == "" {
		return "", ErrSessionRequired
	}
	return userID, nil
}

func lockSessionUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var lockedID string
	err := tx.QueryRow(ctx, lockSessionUserSQL, userID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionRequired
	}
	if err != nil {
		return fmt.Errorf("%w: lock session user: %w", ErrStoreUnavailable, err)
	}
	if lockedID != userID {
		return ErrSessionRequired
	}
	return nil
}

func readResolvedSession(
	ctx context.Context,
	tx pgx.Tx,
	input ResolveInput,
	userID string,
) (resolvedSession, error) {
	var (
		record resolvedSession
	)
	err := tx.QueryRow(
		ctx,
		resolvePrincipalSQL,
		digest(input.Token),
		input.Now,
		userID,
	).Scan(
		&record.principal.SessionID,
		&record.principal.UserID,
		&record.principal.Role,
		&record.principal.IsInitialAdmin,
		&record.principal.Status,
		&record.credentialMode,
		&record.acknowledgedPrivacyVersion,
		&record.onboardingCompletedAt,
		&record.currentPrivacyNoticeVersion,
		&record.csrfDigest,
		&record.credentialKind,
		&record.principal.AuthenticatedCredentialID,
		&record.authorizationValid,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return resolvedSession{}, ErrSessionRequired
	}
	if err != nil {
		return resolvedSession{}, fmt.Errorf("%w: resolve principal: %w", ErrStoreUnavailable, err)
	}
	return record, nil
}
func (r resolvedSession) matchesCSRF(csrfToken string) bool {
	if csrfToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare(digest(csrfToken), r.csrfDigest) == 1
}

func (r resolvedSession) principalValue() Principal {
	r.principal.AuthenticatedCredential = CredentialKind(r.credentialKind)
	r.principal.MustChangePassword = mustChangePassword(r.credentialMode)
	r.principal.MustAcknowledgePrivacy = mustAcknowledgePrivacy(
		r.principal.Role,
		r.acknowledgedPrivacyVersion,
		r.onboardingCompletedAt,
		r.currentPrivacyNoticeVersion,
	)
	return r.principal
}

func validateCreateInput(executor Executor, input CreateInput) error {
	if executor == nil {
		return errors.New("session transaction is required")
	}
	if strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.UserID) == "" {
		return errors.New("session and user identifiers are required")
	}
	if input.Token == "" || input.CSRFToken == "" {
		return errors.New("session token and CSRF token are required")
	}
	if !validCredential(input.AuthenticatedCredentialKind, input.AuthenticatedCredentialID) {
		return errors.New("session authenticated credential is invalid")
	}
	if input.AuthGeneration < 0 ||
		!validSessionLineage(input) {
		return errors.New("session authentication lineage is invalid")
	}
	if input.IssuedAt.IsZero() ||
		input.AbsoluteExpiresAt.IsZero() ||
		input.IdleExpiresAt.IsZero() {
		return errors.New("session lifetimes must be explicitly provided")
	}
	if !input.AbsoluteExpiresAt.After(input.IssuedAt) ||
		!input.IdleExpiresAt.After(input.IssuedAt) {
		return errors.New("session expiry must be after issue time")
	}
	return nil
}

func validSessionLineage(input CreateInput) bool {
	if (input.LineageOperationID == nil) != (input.LineageCredentialID == nil) {
		return false
	}
	switch input.AuthenticatedCredentialKind {
	case CredentialTemporary:
		return input.AuthenticatedCredentialID != nil &&
			(input.LineageCredentialID == nil ||
				*input.AuthenticatedCredentialID == *input.LineageCredentialID)
	case CredentialPermanent:
		return input.AuthenticatedCredentialID == nil
	case CredentialBootstrap, CredentialRecoveryBootstrap:
		return input.AuthenticatedCredentialID == nil &&
			input.LineageOperationID == nil
	default:
		return false
	}
}

func validCredential(kind CredentialKind, credentialID *string) bool {
	switch kind {
	case CredentialTemporary:
		return credentialID != nil && strings.TrimSpace(*credentialID) != ""
	case CredentialPermanent, CredentialBootstrap, CredentialRecoveryBootstrap:
		return credentialID == nil
	default:
		return false
	}
}

func digest(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func mustChangePassword(credentialMode string) bool {
	switch CredentialKind(credentialMode) {
	case CredentialBootstrap, CredentialTemporary, CredentialRecoveryBootstrap:
		return true
	default:
		return credentialMode == "recovery_locked"
	}
}

func mustAcknowledgePrivacy(
	role string,
	acknowledgedVersion *string,
	onboardingCompletedAt *time.Time,
	currentVersion *string,
) bool {
	if role != "admin" || onboardingCompletedAt != nil {
		return false
	}
	if acknowledgedVersion == nil || currentVersion == nil {
		return true
	}
	return *acknowledgedVersion != *currentVersion
}

const insertSessionSQL = `
	INSERT INTO sessions (
		id, user_id, token_digest, csrf_token_digest,
		authenticated_credential_kind, authenticated_credential_id,
		auth_generation, lineage_operation_id, lineage_credential_id,
		issued_at, last_seen_at, absolute_expires_at, idle_expires_at,
		rotated_from_id, access_scope
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'ordinary')
`

const revokeSessionSQL = `
	UPDATE sessions
	SET revoked_at = $2, revoke_reason = $3
	WHERE token_digest = $1 AND revoked_at IS NULL
`

const preReadSessionUserSQL = `
	SELECT user_id
	FROM sessions
	WHERE token_digest = $1
`

const lockSessionUserSQL = `
	SELECT id
	FROM users
	WHERE id = $1
	FOR UPDATE
`

const resolvePrincipalSQL = `
	SELECT
		s.id,
		s.user_id,
		u.role,
		u.is_initial_admin,
		u.status,
		u.credential_mode,
		u.onboarding_privacy_acknowledged_version,
		u.onboarding_completed_at,
		(SELECT version FROM privacy_notices WHERE status = 'current'),
		s.csrf_token_digest,
		s.authenticated_credential_kind,
		s.authenticated_credential_id,
		(
			s.revoked_at IS NULL
			AND s.absolute_expires_at > $2
			AND s.idle_expires_at > $2
			AND u.status = 'active'
			AND u.recovery_required = false
			AND s.auth_generation = u.auth_generation
			AND CASE s.authenticated_credential_kind
				WHEN 'temporary' THEN
					s.authenticated_credential_id IS NOT NULL
					AND u.credential_mode = 'temporary'
					AND (
						(
							s.lineage_operation_id IS NULL
							AND s.lineage_credential_id IS NULL
							AND u.password_reset_current_operation_id IS NULL
							AND EXISTS (
								SELECT 1
								FROM temporary_credentials credential
								WHERE credential.id
										= s.authenticated_credential_id
									AND credential.user_id = s.user_id
									AND credential.purpose = 'new_administrator'
									AND credential.state = 'active'
									AND credential.auth_generation = s.auth_generation
									AND credential.source_operation_id IS NULL
							)
						)
						OR (
							s.lineage_operation_id IS NOT NULL
							AND s.lineage_credential_id
								= s.authenticated_credential_id
							AND u.password_reset_current_operation_id
								= s.lineage_operation_id
							AND EXISTS (
								SELECT 1
								FROM password_reset_operations operation
								JOIN temporary_credentials credential
									ON credential.id
										= operation.temporary_credential_id
								WHERE operation.operation_id
										= s.lineage_operation_id
									AND operation.user_id = s.user_id
									AND operation.state = 'succeeded'
									AND operation.delivery_state = 'accepted'
									AND operation.settlement_state
										= 'temporary_active'
									AND operation.settled_auth_generation
										= operation.target_auth_generation
									AND operation.successor_auth_generation IS NULL
									AND credential.id = s.lineage_credential_id
									AND credential.user_id = s.user_id
									AND credential.purpose = 'password_reset'
									AND credential.state = 'active'
									AND credential.auth_generation
										= operation.target_auth_generation
									AND credential.auth_generation
										= s.auth_generation
									AND credential.source_operation_id
										= operation.operation_id
							)
						)
					)
				WHEN 'permanent' THEN
					s.authenticated_credential_id IS NULL
					AND u.credential_mode = 'permanent'
					AND u.permanent_password_generation
						= u.auth_generation
					AND (
						(
							s.lineage_operation_id IS NULL
							AND s.lineage_credential_id IS NULL
							AND u.permanent_password_source_operation_id IS NULL
							AND u.permanent_password_source_credential_id IS NULL
						)
						OR (
							s.lineage_operation_id
								= u.permanent_password_source_operation_id
							AND s.lineage_credential_id
								= u.permanent_password_source_credential_id
							AND EXISTS (
								SELECT 1
								FROM password_reset_operations operation
								JOIN temporary_credentials credential
									ON credential.id
										= operation.temporary_credential_id
								WHERE operation.operation_id
										= s.lineage_operation_id
									AND operation.user_id = s.user_id
									AND operation.state = 'succeeded'
									AND operation.delivery_state = 'accepted'
									AND operation.settlement_state
										= 'permanent_successor'
									AND operation.settled_auth_generation
										= operation.target_auth_generation
									AND operation.successor_auth_generation
										<= u.permanent_password_generation
									AND credential.id = s.lineage_credential_id
									AND credential.user_id = s.user_id
									AND credential.purpose = 'password_reset'
									AND credential.state = 'consumed'
									AND credential.auth_generation
										= operation.target_auth_generation
									AND credential.source_operation_id
										= operation.operation_id
							)
						)
					)
				WHEN 'bootstrap' THEN
					s.authenticated_credential_id IS NULL
					AND s.lineage_operation_id IS NULL
					AND u.credential_mode = 'bootstrap'
					AND u.permanent_password_source_operation_id IS NULL
				WHEN 'recovery_bootstrap' THEN
					s.authenticated_credential_id IS NULL
					AND s.lineage_operation_id IS NULL
					AND u.credential_mode = 'recovery_bootstrap'
					AND u.is_initial_admin
				ELSE false
			END
		)
	FROM sessions AS s
	JOIN users AS u ON u.id = s.user_id
	WHERE
		s.token_digest = $1
		AND s.access_scope = 'ordinary'
		AND s.user_id = $3
	FOR UPDATE OF s
`

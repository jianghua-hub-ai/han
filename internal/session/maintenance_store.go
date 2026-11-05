package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// RecoverySessionBinding is an observation which the identity authority must verify.
type RecoverySessionBinding struct {
	RunID, InstallationID, RuntimeID string
	Generation                       int64
}

// MaintenanceSessionAuthority checks the already locked User and the private phase.
type MaintenanceSessionAuthority interface {
	CheckLockedMaintenanceUser(context.Context, pgx.Tx, string, RecoverySessionBinding) error
}

// MaintenanceStoreDependencies shares the ordinary store's transaction and Gate owners.
type MaintenanceStoreDependencies struct {
	Transactions      StoreTransactor
	PasswordResetGate PasswordResetGate
	Authority         MaintenanceSessionAuthority
}

// MaintenanceStore persists and resolves only recovery-maintenance sessions.
type MaintenanceStore struct{ dependencies MaintenanceStoreDependencies }

// NewMaintenanceStore validates dependencies without accessing the database.
func NewMaintenanceStore(d MaintenanceStoreDependencies) (MaintenanceStore, error) {
	if nilDependency(d.Transactions) || nilDependency(d.PasswordResetGate) || nilDependency(d.Authority) {
		return MaintenanceStore{}, errors.New("maintenance session dependencies are incomplete")
	}
	return MaintenanceStore{dependencies: d}, nil
}

func maintenanceContext(ctx context.Context) error {
	if ctx == nil {
		return ErrSessionRequired
	}
	deadline, ok := ctx.Deadline()
	if !ok || !deadline.After(time.Now()) {
		return ErrSessionRequired
	}
	return ctx.Err()
}

func validMaintenanceBinding(b RecoverySessionBinding) bool {
	for _, s := range []string{b.RunID, b.InstallationID, b.RuntimeID} {
		if s == "" || strings.TrimSpace(s) != s {
			return false
		}
	}
	return b.Generation > 0
}

func (s MaintenanceStore) check(ctx context.Context, tx pgx.Tx, user string, b RecoverySessionBinding) error {
	if err := maintenanceContext(ctx); err != nil {
		return err
	}
	if nilDependency(tx) || user == "" || !validMaintenanceBinding(b) {
		return ErrSessionRequired
	}
	if nilDependency(s.dependencies.Authority) || nilDependency(s.dependencies.PasswordResetGate) {
		return ErrStoreUnavailable
	}
	if err := s.dependencies.Authority.CheckLockedMaintenanceUser(ctx, tx, user, b); err != nil {
		return err
	}
	return s.dependencies.PasswordResetGate.CheckLockedUser(ctx, tx, user)
}

// CreateMaintenance requires the caller's User/run locks and an audited transaction.
func (s MaintenanceStore) CreateMaintenance(ctx context.Context, tx pgx.Tx, in CreateInput, b RecoverySessionBinding) (Credentials, error) {
	if err := validateCreateInput(tx, in); err != nil {
		return Credentials{}, err
	}
	if in.AuthGeneration <= 0 || in.AuthenticatedCredentialID != nil || in.LineageOperationID != nil || in.LineageCredentialID != nil {
		return Credentials{}, ErrSessionRequired
	}
	if in.AuthenticatedCredentialKind != CredentialPermanent && in.AuthenticatedCredentialKind != CredentialRecoveryBootstrap {
		return Credentials{}, ErrSessionRequired
	}
	if err := s.check(ctx, tx, in.UserID, b); err != nil {
		return Credentials{}, err
	}
	var valid bool
	err := tx.QueryRow(ctx, maintenanceCreateUserSQL, in.UserID, in.AuthGeneration, string(in.AuthenticatedCredentialKind), b.RunID).Scan(&valid)
	if err != nil {
		return Credentials{}, fmt.Errorf("%w: maintenance credential: %w", ErrStoreUnavailable, err)
	}
	if !valid {
		return Credentials{}, ErrSessionRequired
	}
	tag, err := tx.Exec(ctx, insertMaintenanceSessionSQL, in.ID, in.UserID, digest(in.Token), digest(in.CSRFToken), in.AuthenticatedCredentialKind, in.AuthGeneration, in.IssuedAt, in.AbsoluteExpiresAt, in.IdleExpiresAt, in.RotatedFromID, b.RunID, b.InstallationID, b.RuntimeID, b.Generation)
	if err != nil {
		return Credentials{}, fmt.Errorf("%w: maintenance insert: %w", ErrStoreUnavailable, err)
	}
	if tag.RowsAffected() != 1 {
		return Credentials{}, ErrStoreUnavailable
	}
	return Credentials{Token: in.Token, CSRFToken: in.CSRFToken}, nil
}

// RotateMaintenance validates the exact old token before revoking or inserting.
func (s MaintenanceStore) RotateMaintenance(ctx context.Context, tx pgx.Tx, oldToken string, at time.Time, reason string, in CreateInput, b RecoverySessionBinding) (Credentials, error) {
	if err := validateCreateInput(tx, in); err != nil {
		return Credentials{}, err
	}
	if err := s.check(ctx, tx, in.UserID, b); err != nil {
		return Credentials{}, err
	}
	if oldToken == "" || at.IsZero() || strings.TrimSpace(reason) == "" || in.RotatedFromID == nil {
		return Credentials{}, ErrSessionRequired
	}
	var id string
	err := tx.QueryRow(ctx, maintenanceRotationSQL, digest(oldToken), in.UserID, b.RunID, b.InstallationID, b.RuntimeID, b.Generation, at, in.AuthGeneration, string(in.AuthenticatedCredentialKind), reason).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrSessionRequired
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("%w: maintenance rotation: %w", ErrStoreUnavailable, err)
	}
	if id != *in.RotatedFromID {
		return Credentials{}, ErrSessionRequired
	}
	tag, err := tx.Exec(ctx, revokeSessionSQL, digest(oldToken), at, reason)
	if err != nil {
		return Credentials{}, fmt.Errorf("%w: maintenance revoke: %w", ErrStoreUnavailable, err)
	}
	if tag.RowsAffected() != 1 {
		return Credentials{}, ErrSessionRequired
	}
	return s.CreateMaintenance(ctx, tx, in, b)
}

// ResolvePrincipal never falls back to ordinary authentication.
func (s MaintenanceStore) ResolvePrincipal(ctx context.Context, in ResolveInput) (principal Principal, result error) {
	if err := maintenanceContext(ctx); err != nil {
		return Principal{}, err
	}
	if in.Token == "" || in.Now.IsZero() || nilDependency(s.dependencies.Transactions) {
		return Principal{}, ErrSessionRequired
	}
	var user string
	var b RecoverySessionBinding
	err := s.dependencies.Transactions.QueryRow(ctx, maintenancePreReadSQL, digest(in.Token)).Scan(&user, &b.RunID, &b.InstallationID, &b.RuntimeID, &b.Generation)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrSessionRequired
	}
	if err != nil {
		return Principal{}, fmt.Errorf("%w: maintenance pre-read: %w", ErrStoreUnavailable, err)
	}
	tx, err := s.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: maintenance begin: %w", ErrStoreUnavailable, err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			principal = Principal{}
			result = errors.Join(result, ErrStoreUnavailable, err)
		}
	}()
	if err = lockSessionUser(ctx, tx, user); err != nil {
		return Principal{}, err
	}
	if err = s.check(ctx, tx, user, b); err != nil {
		return Principal{}, maintenanceResolveError(err)
	}
	r, err := readMaintenanceSession(ctx, tx, in, user, b)
	if err != nil {
		return Principal{}, err
	}
	if !r.authorizationValid {
		return Principal{}, ErrSessionRequired
	}
	if in.RequireCSRF && !r.matchesCSRF(in.CSRFToken) {
		return Principal{}, ErrCSRFFailed
	}
	if err = tx.Commit(ctx); err != nil {
		return Principal{}, fmt.Errorf("%w: maintenance commit: %w", ErrStoreUnavailable, err)
	}
	p := r.principalValue()
	p.maintenance = true
	p.MustAcknowledgePrivacy = false
	return p, nil
}

func maintenanceResolveError(err error) error {
	if errors.Is(err, ErrPasswordResetGateClosed) {
		return ErrSessionRequired
	}
	if errors.Is(err, ErrSessionRequired) || errors.Is(err, ErrCSRFFailed) {
		return err
	}
	return fmt.Errorf("%w: maintenance authority/Gate: %w", ErrStoreUnavailable, err)
}

func readMaintenanceSession(ctx context.Context, tx pgx.Tx, in ResolveInput, user string, b RecoverySessionBinding) (resolvedSession, error) {
	var r resolvedSession
	err := tx.QueryRow(ctx, resolveMaintenanceSQL, digest(in.Token), in.Now, user, b.RunID, b.InstallationID, b.RuntimeID, b.Generation).Scan(
		&r.principal.SessionID, &r.principal.UserID, &r.principal.Role, &r.principal.IsInitialAdmin, &r.principal.Status,
		&r.credentialMode, &r.acknowledgedPrivacyVersion, &r.onboardingCompletedAt, &r.currentPrivacyNoticeVersion,
		&r.csrfDigest, &r.credentialKind, &r.principal.AuthenticatedCredentialID, &r.authorizationValid)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrSessionRequired
	}
	if err != nil {
		return r, fmt.Errorf("%w: maintenance resolve: %w", ErrStoreUnavailable, err)
	}
	return r, nil
}

const maintenancePreReadSQL = `SELECT user_id::text,recovery_run_id,recovery_installation_id,recovery_runtime_id,recovery_generation
 FROM sessions WHERE token_digest=$1 AND access_scope='recovery_maintenance'`

const maintenanceCreateUserSQL = `SELECT EXISTS(SELECT 1 FROM users u JOIN recovery_runs r ON r.initial_admin_user_id=u.id
 WHERE u.id=$1 AND r.run_id=$4 AND r.outcome='running' AND u.is_initial_admin AND u.role='admin' AND u.status='active'
 AND u.auth_generation=$2 AND u.permanent_password_generation=$2 AND u.credential_mode=$3
 AND u.permanent_password_source_operation_id IS NULL AND u.permanent_password_source_credential_id IS NULL
 AND ((u.recovery_required AND $3='recovery_bootstrap' AND r.initial_admin_bootstrap_generation=$2 AND r.initial_admin_changed_generation IS NULL)
 OR (NOT u.recovery_required AND $3='permanent' AND r.initial_admin_changed_generation=$2)))`

const insertMaintenanceSessionSQL = `INSERT INTO sessions(id,user_id,token_digest,csrf_token_digest,
 authenticated_credential_kind,auth_generation,issued_at,last_seen_at,absolute_expires_at,idle_expires_at,rotated_from_id,
 access_scope,recovery_run_id,recovery_installation_id,recovery_runtime_id,recovery_generation)
 VALUES($1,$2,$3,$4,$5,$6,$7,$7,$8,$9,$10,'recovery_maintenance',$11,$12,$13,$14)`

const maintenanceRotationSQL = `SELECT id::text FROM sessions WHERE token_digest=$1 AND user_id=$2
 AND access_scope='recovery_maintenance' AND recovery_run_id=$3 AND recovery_installation_id=$4
 AND recovery_runtime_id=$5 AND recovery_generation=$6 AND revoked_at IS NULL
 AND absolute_expires_at>$7 AND idle_expires_at>$7
 AND authenticated_credential_id IS NULL AND lineage_operation_id IS NULL AND lineage_credential_id IS NULL
 AND ((auth_generation=$8 AND authenticated_credential_kind=$9)
 OR ($10='password_changed' AND $9='permanent' AND auth_generation=$8-1 AND authenticated_credential_kind='recovery_bootstrap'))
 FOR UPDATE`

const resolveMaintenanceSQL = `SELECT s.id::text,s.user_id::text,u.role,u.is_initial_admin,u.status,u.credential_mode,
 u.onboarding_privacy_acknowledged_version,u.onboarding_completed_at,NULL::text,s.csrf_token_digest,
 s.authenticated_credential_kind,s.authenticated_credential_id,
 (s.revoked_at IS NULL AND s.absolute_expires_at>$2 AND s.idle_expires_at>$2
 AND u.is_initial_admin AND u.role='admin' AND u.status='active' AND s.auth_generation=u.auth_generation
 AND u.permanent_password_generation=u.auth_generation AND s.authenticated_credential_id IS NULL
 AND s.lineage_operation_id IS NULL AND s.lineage_credential_id IS NULL
 AND u.permanent_password_source_operation_id IS NULL AND u.permanent_password_source_credential_id IS NULL
 AND s.authenticated_credential_kind=u.credential_mode
 AND ((u.recovery_required AND u.credential_mode='recovery_bootstrap') OR (NOT u.recovery_required AND u.credential_mode='permanent')))
 FROM sessions s JOIN users u ON u.id=s.user_id
 WHERE s.token_digest=$1 AND s.user_id=$3 AND s.access_scope='recovery_maintenance'
 AND s.recovery_run_id=$4 AND s.recovery_installation_id=$5 AND s.recovery_runtime_id=$6 AND s.recovery_generation=$7
 FOR UPDATE OF s`

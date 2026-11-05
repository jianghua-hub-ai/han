package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const recoveryTrustChangeAction = "identity.recovery.trust.change"

// RecoveryTrustChange preserves the existing password-change body and result.
type RecoveryTrustChange interface {
	ChangeRecoveryPassword(context.Context, recovery.PhasePermit, PasswordChangeRequest) (PasswordChangeResult, error)
}

var _ session.MaintenanceSessionAuthority = (*recoveryIdentityExecution)(nil)
var _ LoginRecoveryMode = (*recoveryIdentityExecution)(nil)
var _ RecoveryTrustChange = (*Service)(nil)

func recoverySessionBinding(p recovery.PermitBinding) session.RecoverySessionBinding {
	return session.RecoverySessionBinding{RunID: p.RunID, InstallationID: p.InstallationID, RuntimeID: p.RuntimeID, Generation: p.Generation}
}

func (e *recoveryIdentityExecution) maintenanceBinding(ctx context.Context) (recovery.PermitBinding, error) {
	if e == nil || nilIdentityDependency(e.d.runtime) || ctx == nil || ctx.Err() != nil {
		return recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	deadline, ok := ctx.Deadline()
	if !ok || !deadline.After(time.Now()) {
		return recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	p, err := e.d.runtime.ReadPermitBinding(ctx)
	if err != nil {
		return p, err
	}
	if p.Purpose != recovery.PhaseRecoveryIdentity || p.RunID == "" || p.BackupRef == "" || p.InstallationID == "" || p.RuntimeID == "" || p.Generation <= 0 {
		return p, recovery.ErrAdmissionBinding
	}
	return p, nil
}

// RecoveryBootstrapAllowed checks the current private phase, never a static mode.
func (e *recoveryIdentityExecution) RecoveryBootstrapAllowed(ctx context.Context) (bool, error) {
	_, err := e.maintenanceBinding(ctx)
	return err == nil, err
}

// CheckLockedMaintenanceUser consumes the caller's User lock before locking the run.
func (e *recoveryIdentityExecution) CheckLockedMaintenanceUser(ctx context.Context, tx pgx.Tx, user string, b session.RecoverySessionBinding) (result error) {
	defer func() { result = recoveryMaintenanceAuthorityError(result) }()
	p, err := e.maintenanceBinding(ctx)
	if err != nil {
		return err
	}
	if nilIdentityDependency(tx) || !recoveryUUID.MatchString(user) || recoverySessionBinding(p) != b {
		return recovery.ErrAdmissionBinding
	}
	bind := recovery.RunBinding{RunID: p.RunID, BackupRef: p.BackupRef}
	r, _, err := e.lockedRun(ctx, tx, bind, p, recovery.RecoveryOutcomeRunning)
	if err != nil {
		return err
	}
	users, err := recoveryExecutionUsers(ctx, tx, []string{user}, false)
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return recovery.ErrAdmissionIncomplete
	}
	u := users[0]
	if err = recoveryMaintenanceUser(r, u); err != nil {
		return err
	}
	if _, err = e.finalization(ctx, tx, r, bind, p.InstallationID); err != nil {
		return err
	}
	if err = maintenanceAuditProof(ctx, tx, r, p, recoveryTrustAction); err != nil {
		return err
	}
	if r.InitialAdminChangedGeneration != nil {
		if err = e.changedInitial(ctx, tx, r, u); err != nil {
			return err
		}
		return maintenanceAuditProof(ctx, tx, r, p, recoveryTrustChangeAction)
	}
	return nil
}

func maintenanceAuditProof(ctx context.Context, tx pgx.Tx, r recovery.AdmissionRun, p recovery.PermitBinding, action string) error {
	b := recovery.RunBinding{RunID: p.RunID, BackupRef: p.BackupRef}
	_, exists, err := recoveryExecutionAudit(ctx, tx, action, r.RunID, b, p.InstallationID)
	if err != nil {
		return err
	}
	if !exists {
		return recovery.ErrAdmissionIncomplete
	}
	expected := map[string]any{"run_id": p.RunID, "backup_ref": p.BackupRef, "installation_id": p.InstallationID, "initial_admin_user_id": r.InitialAdminUserID}
	switch action {
	case recoveryTrustAction:
		expected["bootstrap_generation"] = r.InitialAdminBootstrapGeneration
	case recoveryTrustChangeAction:
		expected["changed_generation"] = r.InitialAdminChangedGeneration
		expected["change_audit_id"] = r.InitialAdminChangeAuditID
	default:
		return recovery.ErrAdmissionBinding
	}
	raw, err := json.Marshal(expected)
	if err != nil {
		return err
	}
	var exact bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE action=$1 AND target_id=$2
 AND target_type='recovery_run' AND actor_kind='system' AND actor_system='recovery_runtime'
 AND outcome='succeeded' AND occurred_at >= $3 AND metadata=$4::jsonb)`, action, r.RunID, r.StartedAt, string(raw)).Scan(&exact)
	if err != nil {
		return err
	}
	if !exact {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

func recoveryMaintenanceAuthorityError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, recovery.ErrAdmissionBinding) || errors.Is(err, session.ErrPasswordResetGateClosed) {
		return errors.Join(session.ErrSessionRequired, err)
	}
	return errors.Join(session.ErrStoreUnavailable, err)
}

func recoveryMaintenanceUser(r recovery.AdmissionRun, u recoveryExecutionUser) error {
	if r.InitialAdminUserID == nil || r.InitialAdminBootstrapGeneration == nil || *r.InitialAdminUserID != u.ID || !u.Initial || u.Role != "admin" || u.Status != "active" || u.Digest == nil || *u.Digest == "" || u.PermanentGeneration == nil || *u.PermanentGeneration != u.Auth || u.SourceOperation != nil || u.SourceCredential != nil {
		return recovery.ErrAdmissionBinding
	}
	g := *r.InitialAdminBootstrapGeneration
	if g <= 0 || g == math.MaxInt64 {
		return recovery.ErrAdmissionBinding
	}
	if r.InitialAdminChangedGeneration == nil {
		if r.InitialAdminChangeAuditID != nil || r.InitialAdminChangedAt != nil || u.Mode != "recovery_bootstrap" || !u.Required || u.Auth != g {
			return recovery.ErrAdmissionBinding
		}
		return nil
	}
	if r.InitialAdminChangeAuditID == nil || r.InitialAdminChangedAt == nil || *r.InitialAdminChangedGeneration != g+1 || u.Auth != g+1 || u.Mode != "permanent" || u.Required {
		return recovery.ErrAdmissionBinding
	}
	return nil
}

func (e *recoveryIdentityExecution) maintenanceSessions(gate session.PasswordResetGate) (session.MaintenanceStore, error) {
	if e == nil || e.d.store == nil || e.d.store.pool == nil {
		return session.MaintenanceStore{}, recovery.ErrAdmissionBinding
	}
	return session.NewMaintenanceStore(session.MaintenanceStoreDependencies{Transactions: e.d.store.pool, PasswordResetGate: gate, Authority: e})
}

func (s *Service) checkMaintenanceLogin(ctx context.Context, tx pgx.Tx, user User) error {
	e, ok := s.loginDependencies.RecoveryMode.(*recoveryIdentityExecution)
	if !ok {
		return nil
	}
	p, err := e.maintenanceBinding(ctx)
	if err != nil {
		return err
	}
	return e.CheckLockedMaintenanceUser(ctx, tx, user.ID, recoverySessionBinding(p))
}

func (s *Service) storeMaintenanceLogin(ctx context.Context, tx pgx.Tx, token string, in session.CreateInput) (session.Credentials, error) {
	e, ok := s.loginDependencies.RecoveryMode.(*recoveryIdentityExecution)
	if !ok {
		return session.Credentials{}, recovery.ErrAdmissionBinding
	}
	p, err := e.maintenanceBinding(ctx)
	if err != nil {
		return session.Credentials{}, err
	}
	store, err := e.maintenanceSessions(s.loginDependencies.PasswordResetGate)
	if err != nil {
		return session.Credentials{}, err
	}
	previous, err := maintenanceLoginPrevious(ctx, tx, token, in, recoverySessionBinding(p))
	if err != nil {
		return session.Credentials{}, err
	}
	in.RotatedFromID = previous
	if previous == nil {
		return store.CreateMaintenance(ctx, tx, in, recoverySessionBinding(p))
	}
	return store.RotateMaintenance(ctx, tx, token, in.IssuedAt, loginSessionRotateReason, in, recoverySessionBinding(p))
}

// maintenanceLoginPrevious selects an optional rotation source after password
// authentication. Absence never authorizes the supplied token; the new session
// still requires the current authority/Gate and the caller's audited transaction.
func maintenanceLoginPrevious(ctx context.Context, tx pgx.Tx, token string, in session.CreateInput, b session.RecoverySessionBinding) (*string, error) {
	if token == "" {
		return nil, nil
	}
	var id string
	err := tx.QueryRow(ctx, maintenanceLoginPreviousSQL, recoveryAuthDigest(token), in.UserID,
		b.RunID, b.InstallationID, b.RuntimeID, b.Generation, in.IssuedAt,
		in.AuthGeneration, string(in.AuthenticatedCredentialKind)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

const maintenanceLoginPreviousSQL = `SELECT id::text FROM sessions WHERE token_digest=$1 AND user_id=$2
 AND access_scope='recovery_maintenance' AND recovery_run_id=$3 AND recovery_installation_id=$4
 AND recovery_runtime_id=$5 AND recovery_generation=$6 AND revoked_at IS NULL
 AND absolute_expires_at>$7 AND idle_expires_at>$7
 AND authenticated_credential_id IS NULL AND lineage_operation_id IS NULL AND lineage_credential_id IS NULL
 AND auth_generation=$8 AND authenticated_credential_kind=$9 FOR UPDATE`

// ChangeRecoveryPassword writes only the initial administrator's recovery trust transition.
func (s *Service) ChangeRecoveryPassword(ctx context.Context, permit recovery.PhasePermit, req PasswordChangeRequest) (PasswordChangeResult, error) {
	if s == nil {
		return PasswordChangeResult{}, recovery.ErrAdmissionBinding
	}
	d := s.passwordChangeDependencies
	e, ok := d.RecoveryAuthority.(*recoveryIdentityExecution)
	if !ok || e == nil {
		return PasswordChangeResult{}, recovery.ErrAdmissionBinding
	}
	if err := e.d.runtime.CheckPhase(ctx, permit, recovery.PhaseRecoveryIdentity); err != nil {
		return PasswordChangeResult{}, err
	}
	p, err := e.maintenanceBinding(ctx)
	if err != nil {
		return PasswordChangeResult{}, err
	}
	if req.CurrentSessionToken == "" || req.TraceID == "" || !recoveryUUID.MatchString(req.Principal.UserID) {
		return PasswordChangeResult{}, session.ErrSessionRequired
	}
	if trace, ok := platform.TraceIDFromContext(ctx); !ok || trace != req.TraceID {
		return PasswordChangeResult{}, recovery.ErrAdmissionBinding
	}
	x := recoveryPasswordChange{service: s, executor: e, ctx: ctx, permit: permit, binding: p, request: req, now: d.Clock.Now().UTC().Truncate(time.Microsecond)}
	scope := recoveryExecutionScope{action: recoveryTrustAction, binding: recovery.RunBinding{RunID: p.RunID, BackupRef: p.BackupRef}, installation: p.InstallationID, actor: req.Principal.UserID}
	err = e.recoveryAuthWrite(ctx, permit, scope, x.run)
	if err != nil {
		if platform.PublicCode(err) == platform.CodeIdentityValidationFailed && !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
			if guardErr := e.guard(ctx, permit, recovery.PhaseRecoveryIdentity); guardErr != nil {
				return PasswordChangeResult{}, recoveryPasswordError(guardErr)
			}
			failed := PasswordChangeFailureAuditRequest{Principal: req.Principal, ErrorCode: platform.CodeIdentityValidationFailed, TraceID: req.TraceID}
			if auditErr := s.AuditPasswordChangeFailure(ctx, failed); auditErr != nil {
				return PasswordChangeResult{}, recoveryPasswordError(auditErr)
			}
		}
		return PasswordChangeResult{}, recoveryPasswordError(err)
	}
	return x.result, nil
}

type recoveryPasswordChange struct {
	service  *Service
	executor *recoveryIdentityExecution
	ctx      context.Context
	permit   recovery.PhasePermit
	binding  recovery.PermitBinding
	request  PasswordChangeRequest
	now      time.Time
	user     User
	result   PasswordChangeResult
}

func (x *recoveryPasswordChange) run(tx pgx.Tx) error {
	d := x.service.passwordChangeDependencies
	u, err := d.Repository.FindUserForPasswordChange(x.ctx, tx, x.request.Principal.UserID)
	if err != nil {
		return err
	}
	x.user = u
	if err = x.executor.CheckLockedMaintenanceUser(x.ctx, tx, u.ID, recoverySessionBinding(x.binding)); err != nil {
		return err
	}
	if !u.RecoveryRequired || u.CredentialMode != CredentialModeRecoveryBootstrap || u.AuthGeneration <= 0 || u.AuthGeneration == math.MaxInt64 {
		return session.ErrSessionRequired
	}
	if err = d.PasswordResetGate.CheckLockedUser(x.ctx, tx, u.ID); err != nil {
		return err
	}
	if err = x.currentSession(tx); err != nil {
		return err
	}
	digest, err := x.password()
	if err != nil {
		return err
	}
	auditID, err := d.IDs.NewID()
	if err != nil {
		return err
	}
	if err = x.persist(tx, digest, auditID); err != nil {
		return err
	}
	return x.rotate(tx)
}

func (x *recoveryPasswordChange) currentSession(tx pgx.Tx) error {
	var id string
	p := x.binding
	err := tx.QueryRow(x.ctx, recoveryChangeSessionSQL, recoveryAuthDigest(x.request.CurrentSessionToken), x.user.ID, p.RunID, p.InstallationID, p.RuntimeID, p.Generation, x.user.AuthGeneration, x.now).Scan(&id)
	if err != nil {
		return errors.Join(session.ErrSessionRequired, err)
	}
	if id != x.request.Principal.SessionID {
		return session.ErrSessionRequired
	}
	return nil
}

func (x *recoveryPasswordChange) password() (string, error) {
	d := x.service.passwordChangeDependencies
	if x.user.PermanentPasswordDigest == nil {
		return "", session.ErrSessionRequired
	}
	matched, _, err := d.Passwords.Verify(x.request.CurrentPassword, *x.user.PermanentPasswordDigest)
	if err != nil {
		return "", err
	}
	if !matched {
		return "", passwordChangeValidationError("current_password", "current_password_mismatch", session.ErrSessionRequired)
	}
	if err = d.Policy.Validate(x.request.NewPassword); err != nil {
		return "", newPasswordPolicyError(err)
	}
	digest, err := d.Passwords.Hash(x.request.NewPassword)
	if err == nil && strings.TrimSpace(digest) == "" {
		return "", recovery.ErrAdmissionBinding
	}
	return digest, err
}

func (x *recoveryPasswordChange) persist(tx pgx.Tx, digest, auditID string) error {
	if err := x.executor.guard(x.ctx, x.permit, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	tag, err := tx.Exec(x.ctx, recoveryChangeUserSQL, x.user.ID, x.user.AuthGeneration, digest, x.now, x.binding.RunID)
	if err = recoveryExecutionOne(tag.RowsAffected(), err); err != nil {
		return err
	}
	if err = x.appendChangeAudit(tx, auditID); err != nil {
		return err
	}
	tag, err = tx.Exec(x.ctx, recoveryChangeRunSQL, x.binding.RunID, x.user.ID, x.user.AuthGeneration, auditID, x.now)
	if err = recoveryExecutionOne(tag.RowsAffected(), err); err != nil {
		return err
	}
	b := recovery.RunBinding{RunID: x.binding.RunID, BackupRef: x.binding.BackupRef}
	return x.executor.event(x.ctx, tx, recoveryTrustChangeAction, b.RunID, b, x.binding.InstallationID, map[string]any{"initial_admin_user_id": x.user.ID, "changed_generation": x.user.AuthGeneration + 1, "change_audit_id": auditID})
}

func (x *recoveryPasswordChange) appendChangeAudit(tx pgx.Tx, id string) error {
	actor, role := x.user.ID, string(x.user.Role)
	return x.service.passwordChangeDependencies.Audit.Append(x.ctx, tx, audit.Event{ID: id, OccurredAt: x.now, ActorKind: audit.ActorUser, ActorUserID: &actor, ActorRole: &role, Action: passwordChangeAuditAction, TargetType: "user", TargetID: x.user.ID, Outcome: "succeeded", TraceID: x.request.TraceID})
}

func (x *recoveryPasswordChange) rotate(tx pgx.Tx) error {
	d := x.service.passwordChangeDependencies
	u, err := d.Repository.FindUserForPasswordChange(x.ctx, tx, x.user.ID)
	if err != nil {
		return err
	}
	in, err := d.SessionIssuer.Issue(u, session.CredentialPermanent, nil)
	if err != nil {
		return err
	}
	in.RotatedFromID = &x.request.Principal.SessionID
	store, err := x.executor.maintenanceSessions(d.PasswordResetGate)
	if err != nil {
		return err
	}
	creds, err := store.RotateMaintenance(x.ctx, tx, x.request.CurrentSessionToken, x.now, "password_changed", in, recoverySessionBinding(x.binding))
	if err != nil {
		return err
	}
	if err = d.Sessions.RevokeOtherSessions(x.ctx, tx, u.ID, in.ID, x.now, "password_changed"); err != nil {
		return err
	}
	x.result = PasswordChangeResult{User: u, SessionToken: creds.Token, CSRFToken: creds.CSRFToken}
	return nil
}

func recoveryPasswordError(err error) error {
	if errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
		return platform.NewPublicError(platform.CodePlatformRecoveryInProgress, err)
	}
	if errors.Is(err, session.ErrSessionRequired) || errors.Is(err, session.ErrPasswordResetGateClosed) {
		return platform.NewPublicError(platform.CodeAuthSessionRequired, err)
	}
	var public *platform.PublicError
	if errors.As(err, &public) {
		return err
	}
	return platform.NewPublicError(platform.CodePlatformRecoveryInProgress, err)
}

func recoveryAuthDigest(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }

func recoveryAuthImage(ctx context.Context, tx pgx.Tx, scope recoveryExecutionScope) ([]byte, error) {
	base, err := recoveryExecutionImage(ctx, tx, scope)
	if err != nil {
		return nil, err
	}
	extra, err := recoveryAckRows(ctx, tx, `SELECT to_jsonb(a) FROM audit_events a WHERE action=$1 AND target_id=$2`, recoveryTrustChangeAction, scope.binding.RunID)
	if err != nil {
		return nil, err
	}
	return json.Marshal([]json.RawMessage{base, extra})
}

// The writer is kept acquired until the independent physical reader is joined.
// The image includes all scoped User, run, Session and BOTH change Audit records.
func (e *recoveryIdentityExecution) recoveryAuthWrite(ctx context.Context, p recovery.PhasePermit, scope recoveryExecutionScope, work func(pgx.Tx) error) (result error) {
	if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	conn, err := e.d.store.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err := finishAdmissionRead(ctx, tx); err != nil {
			result = errors.Join(recovery.ErrRecoveryWriteUnknown, result, err)
		}
	}()
	if err = work(tx); err != nil {
		return err
	}
	expected, err := recoveryAuthImage(ctx, tx, scope)
	if err != nil {
		return err
	}
	if err = e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	ack := e.commit(ctx, tx)
	if ended := finishAdmissionRead(ctx, tx); ended != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, ack, ended)
	}
	if ack == nil {
		return e.guard(ctx, p, recovery.PhaseRecoveryIdentity)
	}
	return e.recoveryAuthReadback(ctx, p, scope, expected, conn.Conn().PgConn().PID(), ack)
}

func (e *recoveryIdentityExecution) recoveryAuthReadback(ctx context.Context, p recovery.PhasePermit, scope recoveryExecutionScope, expected []byte, writer uint32, ack error) (result error) {
	defer func() {
		if result != nil {
			result = errors.Join(recovery.ErrRecoveryWriteUnknown, ack, result)
		}
	}()
	if err := e.guard(ctx, p, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	conn, err := e.d.store.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if conn.Conn().PgConn().PID() == writer {
		return recovery.ErrAdmissionBinding
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	actual, err := recoveryAuthImage(ctx, tx, scope)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) {
		return recovery.ErrAdmissionBinding
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return e.guard(ctx, p, recovery.PhaseRecoveryIdentity)
}

const recoveryChangeSessionSQL = `SELECT id::text FROM sessions WHERE token_digest=$1 AND user_id=$2
 AND access_scope='recovery_maintenance' AND recovery_run_id=$3 AND recovery_installation_id=$4
 AND recovery_runtime_id=$5 AND recovery_generation=$6 AND auth_generation=$7
 AND authenticated_credential_kind='recovery_bootstrap' AND authenticated_credential_id IS NULL
 AND lineage_operation_id IS NULL AND lineage_credential_id IS NULL AND revoked_at IS NULL
 AND absolute_expires_at>$8 AND idle_expires_at>$8 FOR UPDATE`

const recoveryChangeUserSQL = `UPDATE users u SET permanent_password_digest=$3,auth_generation=$2::bigint+1,
 permanent_password_generation=$2::bigint+1,permanent_password_source_operation_id=NULL,permanent_password_source_credential_id=NULL,
 credential_mode='permanent',recovery_required=false,updated_at=$4
 WHERE u.id=$1 AND u.is_initial_admin AND u.role='admin' AND u.status='active'
 AND u.recovery_required AND u.credential_mode='recovery_bootstrap' AND u.auth_generation=$2
 AND EXISTS(SELECT 1 FROM recovery_runs r WHERE r.run_id=$5 AND r.outcome='running'
 AND r.initial_admin_user_id=u.id AND r.initial_admin_bootstrap_generation=$2 AND r.initial_admin_changed_generation IS NULL)`

const recoveryChangeRunSQL = `UPDATE recovery_runs SET initial_admin_changed_generation=$3::bigint+1,
 initial_admin_change_audit_id=$4,initial_admin_changed_at=$5
 WHERE run_id=$1 AND outcome='running' AND initial_admin_user_id=$2
 AND initial_admin_bootstrap_generation=$3 AND initial_admin_changed_generation IS NULL`

package identity

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const recoveryRestoreScope = "recovery_review_restore/v1"
const recoveryReviewVersion = "han-recovery-review/v1"

var recoveryRestoreDigest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var errRecoveryRestorePending = errors.New("recovery restored reset has not reached a proved terminal state")
var errRecoveryResetOrdinaryDispatch = errors.New("recovery reset requires its phase-bound dispatch owner")

// RecoveryRestoreRequest preserves the existing restored-review body and key.
type RecoveryRestoreRequest struct {
	TargetID                    string
	Role                        Role
	Status                      UserStatus
	EvidenceRef, IdempotencyKey string
}

// RecoveryRestoreReset is the design's maintenance-only identity consumer.
type RecoveryRestoreReset interface {
	ResolveRestored(context.Context, recovery.PhasePermit, RecoveryRestoreRequest) error
	SettleRestored(context.Context, recovery.PhasePermit, string, string) error
}

// RecoveryReviewApproval is returned only by an independently trusted source.
type RecoveryReviewApproval struct {
	Version, InstallationID, RunID, TargetID, ActorID string
	Role                                              Role
	Status                                            UserStatus
	EvidenceRef, EvidenceDigest                       string
}

// RecoveryReviewEvidenceReader performs bounded read-only source validation.
type RecoveryReviewEvidenceReader interface {
	ReadApprovedReview(context.Context, string, string, string, string) (RecoveryReviewApproval, error)
}

// RecoveryRestoreResetDependencies reuses the original executor and reset owner.
// Settlement dependencies below contain no Mailer; only Resolve can dispatch.
type RecoveryRestoreResetDependencies struct {
	Execution    RecoveryIdentityMaintenance
	Resets       *PasswordResetCoordinator
	Evidence     RecoveryReviewEvidenceReader
	Permit       *PasswordResetConnectionPermit
	WorkerID     string
	WorkerConfig PasswordResetWorkerConfig
}

type recoveryRestoreReset struct {
	resets     *PasswordResetCoordinator
	settlement recoveryRestoredSettlement
}

// This receiver is independently callable by the driver and has no Mailer,
// generator, hasher, or request-session dependency.
type recoveryRestoredSettlement struct {
	execution *recoveryIdentityExecution
	store     *PasswordResetStore
	evidence  RecoveryReviewEvidenceReader
	permit    *PasswordResetConnectionPermit
	workerID  string
	config    PasswordResetWorkerConfig
}

// NewRecoveryRestoreReset seals construction-time dependencies without I/O.
func NewRecoveryRestoreReset(d RecoveryRestoreResetDependencies) (RecoveryRestoreReset, error) {
	e, ok := d.Execution.(*recoveryIdentityExecution)
	if !ok || e == nil || e.d.store == nil || e.d.store.pool == nil || d.Resets == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	if nilIdentityDependency(d.Evidence) || d.Permit == nil || strings.TrimSpace(d.WorkerID) == "" {
		return nil, recovery.ErrAdmissionBinding
	}
	c := d.Resets.dependencies
	if c.Transactions != e.d.store.pool || c.Store == nil || nilIdentityDependency(c.Mailer) {
		return nil, recovery.ErrAdmissionBinding
	}
	if d.WorkerConfig.LeaseDuration <= 0 || d.WorkerConfig.RetryBackoff <= 0 || d.WorkerConfig.MaxRetries <= 0 || d.WorkerConfig.ClaimBatchLimit <= 0 || d.WorkerConfig.ConnectionBudget != d.Permit.Stats().Budget {
		return nil, ErrPasswordResetOperatorInvalid
	}
	return &recoveryRestoreReset{resets: d.Resets, settlement: recoveryRestoredSettlement{execution: e,
		store: c.Store, evidence: d.Evidence, permit: d.Permit, workerID: d.WorkerID, config: d.WorkerConfig}}, nil
}

func validRecoveryResetMetadata(m *RecoveryResetMetadata) bool {
	return m != nil && strings.TrimSpace(m.RunID) == m.RunID && m.RunID != "" &&
		m.InstallationID != "" && strings.TrimSpace(m.InstallationID) == m.InstallationID &&
		recoveryUUID.MatchString(m.ActorID) && m.EvidenceRef != "" && strings.TrimSpace(m.EvidenceRef) == m.EvidenceRef &&
		recoveryRestoreDigest.MatchString(m.EvidenceDigest) && (m.Role == RoleMember || m.Role == RoleAdmin) &&
		(m.Status == UserStatusActive || m.Status == UserStatusSuspended)
}

func recoveryRestoreApprovalValid(a RecoveryReviewApproval, p recovery.PermitBinding, q RecoveryRestoreRequest, actor string) bool {
	return a.Version == recoveryReviewVersion && a.RunID == p.RunID && a.InstallationID == p.InstallationID &&
		a.TargetID == q.TargetID && a.ActorID == actor && a.Role == q.Role && a.Status == q.Status &&
		a.EvidenceRef == q.EvidenceRef && validRecoveryResetMetadata(recoveryRestoreMetadata(a))
}

func recoveryRestoreMetadata(a RecoveryReviewApproval) *RecoveryResetMetadata {
	return &RecoveryResetMetadata{RunID: a.RunID, InstallationID: a.InstallationID, ActorID: a.ActorID,
		EvidenceRef: a.EvidenceRef, EvidenceDigest: a.EvidenceDigest, Role: a.Role, Status: a.Status}
}

func (s recoveryRestoredSettlement) phase(ctx context.Context, permit recovery.PhasePermit) (context.Context, recovery.PermitBinding, error) {
	if s.execution == nil || ctx == nil {
		return nil, recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	if err := s.execution.d.runtime.CheckPhase(ctx, permit, recovery.PhaseRecoveryIdentity); err != nil {
		return nil, recovery.PermitBinding{}, err
	}
	c, err := s.execution.d.runtime.PhaseContext(ctx, permit)
	if err != nil {
		return nil, recovery.PermitBinding{}, err
	}
	p, err := s.execution.maintenanceBinding(c)
	return c, p, err
}

func (r *recoveryRestoreReset) ResolveRestored(ctx context.Context, permit recovery.PhasePermit, q RecoveryRestoreRequest) error {
	if r == nil || r.resets == nil {
		return recovery.ErrAdmissionBinding
	}
	c, p, err := r.settlement.phase(ctx, permit)
	if err != nil {
		return err
	}
	principal, ok := platform.PrincipalFromContext(c)
	if !ok || !principal.IsInitialAdmin || principal.Role != "admin" || principal.Status != "active" || principal.Restricted() || principal.AuthenticatedCredential != session.CredentialPermanent || principal.SessionID == "" {
		return passwordResetForbiddenError()
	}
	if !recoveryUUID.MatchString(q.TargetID) || q.TargetID == principal.UserID || !validIdempotencyKey(q.IdempotencyKey) || q.EvidenceRef == "" || strings.TrimSpace(q.EvidenceRef) != q.EvidenceRef {
		return recovery.ErrAdmissionBinding
	}
	a, err := r.settlement.evidence.ReadApprovedReview(c, p.InstallationID, p.RunID, q.TargetID, q.EvidenceRef)
	if err != nil {
		return err
	}
	if !recoveryRestoreApprovalValid(a, p, q, principal.UserID) {
		return recovery.ErrAdmissionBinding
	}
	x := recoveryRestoreAttempt{owner: r, ctx: c, permit: permit, binding: p, approval: a, request: q, principal: &principal}
	if err = x.candidate(); err != nil {
		return recoveryRestoreError(err)
	}
	if x.terminal {
		return nil
	}
	if x.prepared.created {
		if err = x.dispatch(); err != nil {
			return recoveryRestoreError(err)
		}
	}
	return r.SettleRestored(c, permit, p.RunID, q.TargetID)
}

func (r *recoveryRestoreReset) SettleRestored(ctx context.Context, permit recovery.PhasePermit, runID, targetID string) error {
	if r == nil {
		return recovery.ErrAdmissionBinding
	}
	return recoveryRestoreError(r.settlement.SettleRestored(ctx, permit, runID, targetID))
}

func recoveryRestoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errRecoveryRestorePending) || errors.Is(err, ErrPasswordResetGateClosed) || errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
		return administratorOperationUncertain(err)
	}
	return err
}

type recoveryRestoreAttempt struct {
	owner     *recoveryRestoreReset
	ctx       context.Context
	permit    recovery.PhasePermit
	binding   recovery.PermitBinding
	approval  RecoveryReviewApproval
	request   RecoveryRestoreRequest
	principal *session.Principal
	prepared  passwordResetPrepared
	terminal  bool
}

func (x *recoveryRestoreAttempt) beginRequest() (idempotency.BeginRequest, error) {
	d := x.owner.resets.dependencies
	raw, err := json.Marshal(struct {
		TargetID    string     `json:"target_id"`
		Role        Role       `json:"role"`
		Status      UserStatus `json:"status"`
		EvidenceRef string     `json:"evidence_ref"`
	}{x.request.TargetID, x.request.Role, x.request.Status, x.request.EvidenceRef})
	if err != nil {
		return idempotency.BeginRequest{}, err
	}
	now := d.Clock.Now().UTC().Truncate(time.Microsecond)
	scope := []byte(strings.Join([]string{x.approval.ActorID, recoveryRestoreScope, x.request.TargetID, x.binding.RunID, x.binding.InstallationID}, "\x00"))
	return idempotency.BeginRequest{Scope: scope, Key: x.request.IdempotencyKey, Request: raw, DigestKeys: d.IdempotencyKeys,
		Now: now, LockedUntil: now.Add(d.IdempotencyLock), ExpiresAt: now.Add(d.IdempotencyTTL)}, nil
}

func (s recoveryRestoredSettlement) write(ctx context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval, work func(pgx.Tx) error) error {
	release, err := s.permit.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	scope := recoveryExecutionScope{action: recoveryReviewAction, binding: recovery.RunBinding{RunID: p.RunID, BackupRef: p.BackupRef}, installation: p.InstallationID, actor: a.ActorID, target: a.TargetID}
	var verify func() error
	if a.EvidenceRef != "" {
		verify = func() error {
			again, er := s.evidence.ReadApprovedReview(ctx, p.InstallationID, p.RunID, a.TargetID, a.EvidenceRef)
			if er != nil {
				return er
			}
			if again != a {
				return recovery.ErrAdmissionBinding
			}
			return nil
		}
	}
	return s.execution.writeVerified(ctx, permit, recovery.PhaseRecoveryIdentity, scope, work, verify)
}

func recoveryRestoreUser(ctx context.Context, tx pgx.Tx, id string) (User, error) {
	u, err := (PostgresPasswordResetRepository{}).LockUser(ctx, tx, id)
	if err != nil {
		return u, err
	}
	err = tx.QueryRow(ctx, `SELECT auth_generation,password_reset_execution_generation,password_reset_current_operation_id,auth_integrity_hold FROM users WHERE id=$1`, id).Scan(&u.AuthGeneration, &u.PasswordResetExecutionGeneration, &u.PasswordResetCurrentOperationID, &u.AuthIntegrityHold)
	return u, err
}

func (s recoveryRestoredSettlement) locked(ctx context.Context, tx pgx.Tx, p recovery.PermitBinding, a RecoveryReviewApproval) (User, recoveryExecutionReview, error) {
	actor, err := recoveryRestoreUser(ctx, tx, a.ActorID)
	if err != nil {
		return User{}, recoveryExecutionReview{}, err
	}
	target, err := recoveryRestoreUser(ctx, tx, a.TargetID)
	if err != nil {
		return target, recoveryExecutionReview{}, err
	}
	if !actor.IsInitialAdmin || actor.Status != UserStatusActive || actor.Role != RoleAdmin || actor.RecoveryRequired || actor.CredentialMode != CredentialModePermanent || actor.AuthIntegrityHold {
		return target, recoveryExecutionReview{}, recovery.ErrAdmissionBinding
	}
	if err = s.execution.CheckLockedMaintenanceUser(ctx, tx, actor.ID, recoverySessionBinding(p)); err != nil {
		return target, recoveryExecutionReview{}, err
	}
	if err = s.store.CheckLockedUser(ctx, tx, actor.ID); err != nil {
		return target, recoveryExecutionReview{}, err
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT to_jsonb(r)::text FROM recovery_account_reviews r WHERE recovery_run_id=$1 AND user_id=$2 FOR UPDATE`, p.RunID, a.TargetID).Scan(&raw)
	var review recoveryExecutionReview
	if err == nil {
		err = json.Unmarshal(raw, &review)
	}
	return target, review, err
}

func recoveryRestoreTargetEligible(u User, r recoveryExecutionReview) bool {
	return !u.IsInitialAdmin && u.Status == UserStatusSuspended && u.DeletedAt == nil && u.RecoveryRequired &&
		u.CredentialMode == CredentialModeRecoveryLocked && !u.AuthIntegrityHold && u.PermanentPasswordDigest == nil &&
		u.Email != nil && strings.TrimSpace(*u.Email) != "" && r.UserID == u.ID && r.BackupStatus != "deleted" &&
		r.BackupDeletedAt == nil && r.Decision == "pending" && r.ReviewedBy == nil && r.ReviewedAt == nil && r.ResetAt == nil &&
		u.AuthGeneration >= 0 && u.AuthGeneration < math.MaxInt64 && u.PasswordResetExecutionGeneration >= 0 && u.PasswordResetExecutionGeneration < math.MaxInt64
}

func (x *recoveryRestoreAttempt) candidate() error {
	s := x.owner.settlement
	begin, err := x.beginRequest()
	if err != nil {
		return err
	}
	return s.write(x.ctx, x.permit, x.binding, x.approval, func(tx pgx.Tx) error {
		u, review, err := s.locked(x.ctx, tx, x.binding, x.approval)
		if err != nil {
			return err
		}
		if err = x.session(tx); err != nil {
			return err
		}
		x.prepared.target = u
		if review.OperationID != nil {
			o, err := recoveryRestoreOperation(x.ctx, tx, *review.OperationID)
			if err != nil {
				return err
			}
			if o.State == PasswordResetOperationSucceeded || (o.State != PasswordResetOperationFailed && o.State != PasswordResetOperationInvalidated) {
				return x.existing(tx, u, review, o, begin)
			}
		}
		if !recoveryRestoreTargetEligible(u, review) {
			return recovery.ErrAdmissionBinding
		}
		if err = s.store.CheckLockedUser(x.ctx, tx, u.ID); err != nil {
			return err
		}
		_, found, err := s.store.ClassifyExistingCandidate(x.ctx, tx, u.ID, begin, x.owner.resets.dependencies.IDs.NewID)
		if err != nil {
			return err
		}
		if found {
			return ErrPasswordResetStateConflict
		}
		return x.create(tx, u, begin)
	})
}

func (x *recoveryRestoreAttempt) session(tx pgx.Tx) error {
	if x.principal == nil {
		return passwordResetForbiddenError()
	}
	var valid bool
	p := x.binding
	err := tx.QueryRow(x.ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE id=$1 AND user_id=$2 AND access_scope='recovery_maintenance'
 AND recovery_run_id=$3 AND recovery_installation_id=$4 AND recovery_runtime_id=$5 AND recovery_generation=$6
 AND authenticated_credential_kind='permanent' AND authenticated_credential_id IS NULL AND lineage_operation_id IS NULL
 AND lineage_credential_id IS NULL AND revoked_at IS NULL AND absolute_expires_at>$7 AND idle_expires_at>$7
 AND auth_generation=(SELECT auth_generation FROM users WHERE id=$2))`, x.principal.SessionID, x.approval.ActorID, p.RunID, p.InstallationID, p.RuntimeID, p.Generation, x.owner.resets.dependencies.Clock.Now()).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return session.ErrSessionRequired
	}
	return nil
}

func recoveryRestoreOperation(ctx context.Context, tx pgx.Tx, id string) (PasswordResetOperation, error) {
	var locked string
	if err := tx.QueryRow(ctx, `SELECT operation_id::text FROM password_reset_operations WHERE operation_id=$1 FOR UPDATE`, id).Scan(&locked); err != nil {
		return PasswordResetOperation{}, err
	}
	o, err := recoveryReadResetOperation(ctx, tx, id)
	if err != nil {
		return o, err
	}
	var run, install, ref, digest, actor, role, status *string
	err = tx.QueryRow(ctx, `SELECT recovery_run_id,recovery_installation_id,recovery_evidence_ref,recovery_evidence_digest,recovery_actor_id::text,recovery_target_role,recovery_target_status FROM password_reset_operations WHERE operation_id=$1`, id).Scan(&run, &install, &ref, &digest, &actor, &role, &status)
	if err != nil {
		return o, err
	}
	if run != nil && install != nil && ref != nil && digest != nil && actor != nil && role != nil && status != nil {
		o.Recovery = &RecoveryResetMetadata{RunID: *run, InstallationID: *install, EvidenceRef: *ref, EvidenceDigest: *digest, ActorID: *actor, Role: Role(*role), Status: UserStatus(*status)}
	} else if run != nil || install != nil || ref != nil || digest != nil || actor != nil || role != nil || status != nil {
		return o, ErrPasswordResetStateConflict
	}
	return o, nil
}

func (x *recoveryRestoreAttempt) existing(tx pgx.Tx, u User, review recoveryExecutionReview, o PasswordResetOperation, begin idempotency.BeginRequest) error {
	if o.Recovery == nil || *o.Recovery != *recoveryRestoreMetadata(x.approval) || o.UserID != u.ID || review.BackupStatus == "deleted" || u.Status == UserStatusDeleted {
		return recovery.ErrAdmissionBinding
	}
	if err := recoveryRestoreLockCredential(x.ctx, tx, o); err != nil {
		return err
	}
	result, found, err := x.owner.settlement.store.ClassifyExistingCandidate(x.ctx, tx, u.ID, begin, x.owner.resets.dependencies.IDs.NewID)
	if err != nil {
		return err
	}
	if !found || result.Action != PasswordResetCandidateExisting || result.Operation.ID != o.ID {
		return ErrPasswordResetStateConflict
	}
	x.prepared.operation = o
	if review.Decision == "restored" {
		if err = x.owner.settlement.terminal(x.ctx, tx, x.binding, x.approval, review, u, o); err != nil {
			return err
		}
		x.terminal = true
		return nil
	}
	if review.Decision != "pending" || o.State == PasswordResetOperationSucceeded {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

func recoveryRestoreLockCredential(ctx context.Context, tx pgx.Tx, o PasswordResetOperation) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM temporary_credentials WHERE id=$1 AND user_id=$2 AND source_operation_id=$3 FOR UPDATE`, o.TemporaryCredentialID, o.UserID, o.ID).Scan(&id)
	return err
}

func (x *recoveryRestoreAttempt) create(tx pgx.Tx, u User, begin idempotency.BeginRequest) error {
	d := x.owner.resets.dependencies
	var generation int64
	if err := tx.QueryRow(x.ctx, `SELECT COALESCE(MAX(generation),0) FROM temporary_credentials WHERE user_id=$1`, u.ID).Scan(&generation); err != nil {
		return err
	}
	if generation < 0 || generation == math.MaxInt64 {
		return ErrTemporaryCredentialGeneration
	}
	secret, err := d.Secrets.Generate()
	if err != nil {
		return err
	}
	if err = d.PasswordPolicy.Validate(secret); err != nil {
		return err
	}
	credentialID, err := d.IDs.NewID()
	if err != nil {
		return err
	}
	operationID, err := d.IDs.NewID()
	if err != nil {
		return err
	}
	pending, err := NewTemporaryCredential(d.PasswordHasher, TemporaryCredentialInput{ID: credentialID, UserID: u.ID, Purpose: TemporaryCredentialPasswordReset, Generation: generation + 1, Secret: secret, DeliveredToEmail: strings.TrimSpace(*u.Email), IssuedAt: begin.Now})
	if err != nil {
		return err
	}
	o := PasswordResetOperation{ID: operationID, UserID: u.ID, TemporaryCredentialID: credentialID, RequestKind: PasswordResetInitialReset,
		State: PasswordResetOperationPrepared, DeliveryState: PasswordResetDeliveryNotStarted, SettlementState: PasswordResetSettlementPending,
		Recovery: recoveryRestoreMetadata(x.approval), CreatedAt: begin.Now, UpdatedAt: begin.Now}
	job := PasswordResetRecoveryJob{OperationID: o.ID, Status: PasswordResetRecoveryJobReady, NextAttemptAt: begin.LockedUntil, CreatedAt: begin.Now, UpdatedAt: begin.Now}
	result, err := x.owner.settlement.store.BeginOrClassifyCandidate(x.ctx, tx, PasswordResetCandidate{Operation: o, Credential: pending, RecoveryJob: job, IdempotencyRequest: begin})
	if err != nil {
		return err
	}
	if result.Action != PasswordResetCandidateCreated {
		return ErrPasswordResetStateConflict
	}
	tag, err := tx.Exec(x.ctx, `UPDATE recovery_account_reviews SET restoration_operation_id=$3 WHERE recovery_run_id=$1 AND user_id=$2 AND decision='pending' AND backup_status<>'deleted'`, x.binding.RunID, u.ID, o.ID)
	if err = requirePasswordResetOneRow(tag, err); err != nil {
		return err
	}
	x.prepared = passwordResetPrepared{target: u, pending: pending, secret: secret, operation: result.Operation, claim: result.Claim, created: true}
	return nil
}

func (x *recoveryRestoreAttempt) dispatch() error {
	d := x.owner.resets.dependencies
	s := x.owner.settlement
	defer func() { x.prepared.secret = "" }()
	if err := s.execution.guard(x.ctx, x.permit, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	attemptID, err := d.IDs.NewID()
	if err != nil {
		return err
	}
	err = s.write(x.ctx, x.permit, x.binding, x.approval, func(tx pgx.Tx) error {
		_, review, err := s.locked(x.ctx, tx, x.binding, x.approval)
		if err != nil {
			return err
		}
		if review.OperationID == nil || *review.OperationID != x.prepared.operation.ID {
			return ErrPasswordResetStateConflict
		}
		return s.store.ClaimMailAttempt(x.ctx, tx, x.request.TargetID, x.prepared.operation.ID, attemptID, d.Clock.Now().UTC().Truncate(time.Microsecond))
	})
	if err != nil {
		return err
	}
	if err = s.execution.guard(x.ctx, x.permit, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	mailCtx, cancel := context.WithTimeout(x.ctx, d.SMTPOperationBudget)
	mailErr := d.Mailer.Send(mailCtx, mailer.Message{Purpose: mailer.PurposePasswordResetCredential, To: strings.TrimSpace(*x.prepared.target.Email), Subject: passwordResetSubject, Body: passwordResetSecretPrefix + x.prepared.secret})
	cancel()
	x.prepared.secret = ""
	c, done := context.WithTimeout(context.WithoutCancel(x.ctx), d.SMTPOperationBudget)
	defer done()
	return x.delivery(c, attemptID, mailErr)
}

func (x *recoveryRestoreAttempt) delivery(ctx context.Context, attemptID string, mailErr error) error {
	s := x.owner.settlement
	err := s.write(ctx, x.permit, x.binding, x.approval, func(tx pgx.Tx) error {
		_, review, err := s.locked(ctx, tx, x.binding, x.approval)
		if err != nil {
			return err
		}
		if review.OperationID == nil || *review.OperationID != x.prepared.operation.ID {
			return ErrPasswordResetStateConflict
		}
		now := s.execution.d.clock.Now().UTC().Truncate(time.Microsecond)
		o := x.prepared.operation
		if mailErr == nil {
			if err = s.store.RecordAcceptedDelivery(ctx, tx, o.UserID, o.ID, attemptID, now); err != nil {
				return err
			}
			return requirePasswordResetOneRow(tx.Exec(ctx, acceleratePasswordResetRecoveryJobSQL, now, o.ID))
		}
		if errors.Is(mailErr, mailer.ErrMessageRejected) {
			return s.store.RecordRejectedDelivery(ctx, tx, o, x.prepared.pending.ID, attemptID, now)
		}
		return s.store.RecordUnknownDelivery(ctx, tx, o.UserID, o.ID, attemptID, now)
	})
	if err != nil {
		return err
	}
	if mailErr != nil {
		if errors.Is(mailErr, mailer.ErrMessageRejected) {
			return platform.NewPublicError(platform.CodeIdentityEmailDeliveryFailed, mailErr)
		}
		return errors.Join(errRecoveryRestorePending, mailErr)
	}
	return nil
}

func (s recoveryRestoredSettlement) SettleRestored(ctx context.Context, permit recovery.PhasePermit, runID, target string) error {
	c, p, err := s.phase(ctx, permit)
	if err != nil {
		return err
	}
	if p.RunID != runID || !recoveryUUID.MatchString(target) {
		return recovery.ErrAdmissionBinding
	}
	a, operation, err := s.lookup(c, p, target)
	if err != nil {
		return err
	}
	if operation == "" {
		return s.legacy(c, permit, p, a)
	}
	approved, err := s.evidence.ReadApprovedReview(c, p.InstallationID, p.RunID, target, a.EvidenceRef)
	if err != nil {
		return err
	}
	if approved != a {
		return recovery.ErrAdmissionBinding
	}
	lease, terminal, err := s.claim(c, permit, p, a, operation)
	if err != nil {
		return err
	}
	if terminal {
		return nil
	}
	if err = s.renew(c, permit, p, a, lease); err != nil {
		return err
	}
	restored, err := s.settleClaim(c, permit, p, a, lease)
	if err != nil && !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
		return errors.Join(err, s.fail(c, permit, p, a, lease, err))
	}
	if err == nil && !restored {
		return errRecoveryRestorePending
	}
	return err
}

func (s recoveryRestoredSettlement) lookup(ctx context.Context, p recovery.PermitBinding, target string) (RecoveryReviewApproval, string, error) {
	release, err := s.permit.Acquire(ctx)
	if err != nil {
		return RecoveryReviewApproval{}, "", err
	}
	defer release()
	a := RecoveryReviewApproval{Version: recoveryReviewVersion, RunID: p.RunID, InstallationID: p.InstallationID, TargetID: target}
	var op, ref, digest, actor, role, status *string
	err = s.execution.d.store.pool.QueryRow(ctx, `SELECT v.restoration_operation_id::text,o.recovery_evidence_ref,o.recovery_evidence_digest,
 COALESCE(o.recovery_actor_id,r.initial_admin_user_id)::text,o.recovery_target_role,o.recovery_target_status
 FROM recovery_account_reviews v JOIN recovery_runs r ON r.run_id=v.recovery_run_id
 LEFT JOIN password_reset_operations o ON o.operation_id=v.restoration_operation_id
 WHERE v.recovery_run_id=$1 AND v.user_id=$2`, p.RunID, target).Scan(&op, &ref, &digest, &actor, &role, &status)
	if err != nil {
		return a, "", err
	}
	if actor == nil {
		return a, "", recovery.ErrAdmissionBinding
	}
	a.ActorID = *actor
	if op == nil {
		return a, "", nil
	}
	if ref == nil || digest == nil || role == nil || status == nil {
		return a, "", recovery.ErrAdmissionBinding
	}
	a.EvidenceRef = *ref
	a.EvidenceDigest = *digest
	a.Role = Role(*role)
	a.Status = UserStatus(*status)
	if !validRecoveryResetMetadata(recoveryRestoreMetadata(a)) {
		return a, "", recovery.ErrAdmissionBinding
	}
	return a, *op, nil
}

func (s recoveryRestoredSettlement) claim(ctx context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval, id string) (PasswordResetRecoveryLease, bool, error) {
	var lease PasswordResetRecoveryLease
	terminal := false
	err := s.write(ctx, permit, p, a, func(tx pgx.Tx) error {
		u, review, err := s.locked(ctx, tx, p, a)
		if err != nil {
			return err
		}
		if review.OperationID == nil || *review.OperationID != id {
			return recovery.ErrAdmissionBinding
		}
		o, err := recoveryRestoreOperation(ctx, tx, id)
		if err != nil {
			return err
		}
		if review.Decision == "restored" {
			if err = s.terminal(ctx, tx, p, a, review, u, o); err != nil {
				return err
			}
			terminal = true
			return nil
		}
		if !recoveryRestoreTargetEligible(u, review) || o.Recovery == nil || *o.Recovery != *recoveryRestoreMetadata(a) {
			return recovery.ErrAdmissionBinding
		}
		lease, err = claimRecoveryResetLease(ctx, tx, s, id, u.ID)
		return err
	})
	return lease, terminal, err
}

func (s recoveryRestoredSettlement) renew(ctx context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval, lease PasswordResetRecoveryLease) error {
	return s.write(ctx, permit, p, a, func(tx pgx.Tx) error {
		_, review, err := s.locked(ctx, tx, p, a)
		if err != nil {
			return err
		}
		if review.OperationID != nil && *review.OperationID != lease.OperationID {
			return ErrPasswordResetStateConflict
		}
		now := s.execution.d.clock.Now().UTC().Truncate(time.Microsecond)
		return requirePasswordResetOneRow(tx.Exec(ctx, passwordResetRecoveryRenewSQL, now.Add(s.config.LeaseDuration), now, lease.OperationID, lease.Owner, lease.AttemptCount))
	})
}

func (s recoveryRestoredSettlement) fail(ctx context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval, lease PasswordResetRecoveryLease, cause error) error {
	return s.write(ctx, permit, p, a, func(tx pgx.Tx) error {
		if _, _, err := s.locked(ctx, tx, p, a); err != nil {
			return err
		}
		now := s.execution.d.clock.Now().UTC().Truncate(time.Microsecond)
		status := PasswordResetRecoveryJobRetryWait
		next := now.Add(s.config.RetryBackoff)
		if lease.AttemptCount >= s.config.MaxRetries {
			status = PasswordResetRecoveryJobDeadLetter
			next = now
		}
		return requirePasswordResetOneRow(tx.Exec(ctx, passwordResetRecoveryFailSQL, status, next, passwordResetRecoveryFailureCode(cause), now, lease.OperationID, lease.Owner, lease.AttemptCount))
	})
}

func (s recoveryRestoredSettlement) terminal(ctx context.Context, tx pgx.Tx, p recovery.PermitBinding, a RecoveryReviewApproval, review recoveryExecutionReview, u User, o PasswordResetOperation) error {
	if o.Recovery == nil || *o.Recovery != *recoveryRestoreMetadata(a) || o.UserID != u.ID || !passwordResetOperationAuthorityValid(o) || !passwordResetDeliveryEvidenceValid(o) {
		return recovery.ErrAdmissionBinding
	}
	users, err := recoveryExecutionUsers(ctx, tx, []string{u.ID}, false)
	if err != nil {
		return err
	}
	if len(users) != 1 {
		return recovery.ErrAdmissionIncomplete
	}
	return s.execution.reviewProof(ctx, tx, recovery.RunBinding{RunID: p.RunID, BackupRef: p.BackupRef}, p.InstallationID, review, users[0])
}

func (s recoveryRestoredSettlement) legacy(ctx context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval) error {
	var lease PasswordResetRecoveryLease
	err := s.write(ctx, permit, p, a, func(tx pgx.Tx) error {
		u, review, err := s.locked(ctx, tx, p, a)
		if err != nil {
			return err
		}
		if review.Decision != "pending" || review.OperationID != nil || u.AuthIntegrityHold || u.PasswordResetCurrentOperationID == nil {
			return errRecoveryRestorePending
		}
		o, err := recoveryRestoreOperation(ctx, tx, *u.PasswordResetCurrentOperationID)
		if err != nil {
			return err
		}
		if o.Recovery != nil || o.State == PasswordResetOperationSucceeded || o.State == PasswordResetOperationInvalidated || o.State == PasswordResetOperationFailed {
			return errRecoveryRestorePending
		}
		lease, err = claimRecoveryResetLease(ctx, tx, s, o.ID, u.ID)
		return err
	})
	if err != nil {
		return err
	}
	if err = s.renew(ctx, permit, p, a, lease); err != nil {
		return err
	}
	err = s.settleLegacy(ctx, permit, p, a, lease)
	if err != nil && !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
		return errors.Join(err, s.fail(ctx, permit, p, a, lease, err))
	}
	if err == nil {
		return errRecoveryRestorePending
	}
	return err
}

var _ RecoveryRestoreReset = (*recoveryRestoreReset)(nil)

func (s recoveryRestoredSettlement) settleClaim(c context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval, lease PasswordResetRecoveryLease) (bool, error) {
	restored := false
	err := s.write(c, permit, p, a, func(tx pgx.Tx) error {
		u, review, err := s.locked(c, tx, p, a)
		if err != nil {
			return err
		}
		if !recoveryRestoreTargetEligible(u, review) || review.OperationID == nil || *review.OperationID != lease.OperationID {
			return recovery.ErrAdmissionBinding
		}
		o, err := recoveryRestoreOperation(c, tx, lease.OperationID)
		if err != nil {
			return err
		}
		if err = settleRecoveryRestoredReset(c, tx, s, lease, o, a, u, p); err != nil {
			return err
		}
		restored = o.DeliveryState == PasswordResetDeliveryAccepted
		return nil
	})
	return restored, err
}

func (s recoveryRestoredSettlement) settleLegacy(ctx context.Context, permit recovery.PhasePermit, p recovery.PermitBinding, a RecoveryReviewApproval, lease PasswordResetRecoveryLease) error {
	return s.write(ctx, permit, p, a, func(tx pgx.Tx) error {
		u, _, err := s.locked(ctx, tx, p, a)
		if err != nil {
			return err
		}
		now := s.execution.d.clock.Now().UTC().Truncate(time.Microsecond)
		o, err := lockPasswordResetSettlementOperation(ctx, tx, lease, now)
		if err != nil {
			return err
		}
		if o.delivery == PasswordResetDeliveryAccepted {
			return ErrPasswordResetGateClosed
		}
		if err = settleUnacceptedPasswordReset(ctx, tx, lease, o, now); err != nil {
			return err
		}
		if err = appendRecoveryResetSettlementAudit(ctx, tx, s, lease, now); err != nil {
			return err
		}
		if u.ID != lease.UserID {
			return ErrPasswordResetStateConflict
		}
		return completePasswordResetRecoveryJob(ctx, tx, lease, now)
	})
}

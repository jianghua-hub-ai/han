package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	canonicalDeleteAuditAction            = "identity.user.delete.canonical"
	canonicalDeleteIdentifierAuditAction  = "identity.identifier.email.release"
	canonicalDeleteAuditTargetUser        = "user"
	canonicalDeleteAuditTargetIdentifier  = "identity_identifier"
	canonicalDeleteAuditOutcomeSucceeded  = "succeeded"
	canonicalDeleteIdentifierKindMetadata = "email"
)

// CanonicalDeleteMode identifies the authorized canonical deletion caller.
type CanonicalDeleteMode string

const (
	CanonicalDeleteDailyGovernance CanonicalDeleteMode = "daily_governance"
	CanonicalDeleteRecoveryKept    CanonicalDeleteMode = "recovery_kept_deleted"
)

// AccountDeletionReason is audit-only context for a canonical deletion.
type AccountDeletionReason string

const (
	AccountDeletionGovernance        AccountDeletionReason = "governance"
	AccountDeletionPrivacyWithdrawal AccountDeletionReason = "privacy_withdrawal"
)

// RecoveryDeleteCapability authorizes one recovery_kept_deleted replay.
//
// It is the transaction-scoped authorization capability required by design r45
// §3.3.10 and §4.2 step 3.1. Only the controlled recovery consumer may mint it;
// an HTTP handler or an ordinary internal caller cannot obtain a valid one. The
// fields are unexported so no caller can construct a capability by literal, and
// minting alone never grants authority - the coordinator additionally proves the
// capability against the identity-owned recovery run authority and the
// maintenance/readiness gate before any write.
type RecoveryDeleteCapability struct {
	runID     string
	backupRef string
}

// NewRecoveryDeleteCapability mints a capability for the controlled recovery
// consumer. It MUST NOT be called from an HTTP entry point or an ordinary
// internal caller; downstream double proof fails closed if either the run
// authority or the maintenance/readiness gate does not corroborate it.
func NewRecoveryDeleteCapability(runID, backupRef string) (RecoveryDeleteCapability, error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(runID) != runID ||
		strings.TrimSpace(backupRef) == "" || strings.TrimSpace(backupRef) != backupRef {
		return RecoveryDeleteCapability{}, canonicalDeleteForbiddenError()
	}
	return RecoveryDeleteCapability{runID: runID, backupRef: backupRef}, nil
}

// RunID reports the recovery run bound to the capability.
func (c RecoveryDeleteCapability) RunID() string { return c.runID }

// BackupRef reports the immutable backup reference bound to the capability.
func (c RecoveryDeleteCapability) BackupRef() string { return c.backupRef }

func (c RecoveryDeleteCapability) valid() bool {
	return strings.TrimSpace(c.runID) == c.runID && c.runID != "" &&
		strings.TrimSpace(c.backupRef) == c.backupRef && c.backupRef != ""
}

// RecoveryKeptDeletedRequest binds one recovery replay to its authorization.
type RecoveryKeptDeletedRequest struct {
	Capability  RecoveryDeleteCapability
	EvidenceRef *string
}

// CanonicalDeleteRequest contains trusted authorization and target evidence.
type CanonicalDeleteRequest struct {
	Actor      session.Principal
	TargetID   string
	TargetRole Role
	Mode       CanonicalDeleteMode
	Reason     AccountDeletionReason
	TraceID    string
	Recovery   *RecoveryKeptDeletedRequest
}

// CanonicalDeleteResult reports the article-owned cleanup condition.
type CanonicalDeleteResult struct {
	TargetID             string
	NewlyOrphanedCount   int
	AlreadyOrphanedCount int
	PendingOrphanCount   int
	EmailIdentifierFreed bool
}

// CanonicalDeleteRepository owns identity deletion state in a caller transaction.
type CanonicalDeleteRepository interface {
	LockUser(context.Context, pgx.Tx, string) (User, error)
	LockIdentifiers(context.Context, pgx.Tx, string) ([]IdentityIdentifier, error)
	MarkUserDeleted(context.Context, pgx.Tx, string, time.Time) error
	ReserveIdentifiers(context.Context, pgx.Tx, string) error
	DeleteSessions(context.Context, pgx.Tx, string) error
	InvalidateTemporaryCredentials(context.Context, pgx.Tx, string, time.Time) error
	ReleaseEmailIdentifier(context.Context, pgx.Tx, string, time.Time) (bool, error)
	LockRecoveryAccountReview(
		context.Context,
		pgx.Tx,
		string,
		string,
	) (RecoveryAccountReview, error)
	CompleteRecoveryAccountReview(
		context.Context,
		pgx.Tx,
		string,
		string,
		string,
		*string,
		time.Time,
	) error
}

// RecoveryRunLocker locks and reads the identity-owned recovery run authority in
// the caller transaction. Its production implementation is the single
// identity-owned RecoveryRunStore; it is read-only and never an alternate
// authority, so it cannot transition run state.
type RecoveryRunLocker interface {
	LockRecoveryRunForDeletion(context.Context, pgx.Tx, string) (recovery.RecoveryRun, error)
}

// RecoveryOpenGate is the read-only maintenance/readiness gate consumed before a
// recovery_kept_deleted replay. It returns nil only while the deployment is in
// maintenance mode with public readiness closed. Its production implementation is
// delivered by the recovery mainline and later deployment cards (candidate
// identity-auth-S10.5); this card only consumes it and fails closed when absent.
type RecoveryOpenGate interface {
	CheckRecoveryOpenGate(context.Context, pgx.Tx) error
}

// CanonicalDeleteAuditAppender appends deletion facts in the caller transaction.
type CanonicalDeleteAuditAppender interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

// CanonicalDeleteDependencies are the single canonical deletion authority inputs.
type CanonicalDeleteDependencies struct {
	Transactions RegistrationTransactor
	Repository   CanonicalDeleteRepository
	Articles     ArticleAccountPort
	Audit        CanonicalDeleteAuditAppender
	IDs          RegistrationIDGenerator
	Clock        RegistrationClock
	// RecoveryRuns and RecoveryGate are required only for recovery_kept_deleted.
	// When unset, a recovery replay fails closed before any write.
	RecoveryRuns RecoveryRunLocker
	RecoveryGate RecoveryOpenGate
}

// CanonicalDeleteCoordinator owns daily and recovery-kept-deleted orchestration.
type CanonicalDeleteCoordinator struct {
	dependencies CanonicalDeleteDependencies
}

// CanonicalDelete applies all identity and article deletion effects atomically.
func (c *CanonicalDeleteCoordinator) CanonicalDelete(
	ctx context.Context,
	request CanonicalDeleteRequest,
) (CanonicalDeleteResult, error) {
	if err := validateCanonicalDeleteRequest(request); err != nil {
		return CanonicalDeleteResult{}, err
	}
	if c == nil {
		return CanonicalDeleteResult{}, canonicalDeleteInternalError(
			"validate canonical delete coordinator",
			errors.New("coordinator is nil"),
		)
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"begin canonical delete transaction",
			err,
		)
	}
	if tx == nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"begin canonical delete transaction",
			errors.New("transaction is nil"),
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()

	result, err := c.canonicalDeleteInTransaction(ctx, tx, request)
	if err != nil {
		return CanonicalDeleteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return CanonicalDeleteResult{}, platform.NewPublicError(
			platform.CodeIdentityOperationUncertain,
			fmt.Errorf("commit canonical delete transaction: %w", err),
		)
	}
	closed = true
	return result, nil
}

func (c *CanonicalDeleteCoordinator) canonicalDeleteInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
) (CanonicalDeleteResult, error) {
	plan, err := c.prepareCanonicalDelete(ctx, tx, request)
	if err != nil {
		return CanonicalDeleteResult{}, err
	}
	now, deleteAuditID, err := c.canonicalDeleteIdentity()
	if err != nil {
		return CanonicalDeleteResult{}, err
	}
	result, err := c.applyCanonicalDeleteIdentity(
		ctx,
		tx,
		request,
		plan.actor,
		plan.target,
		plan.traceID,
		now,
	)
	if err != nil {
		return CanonicalDeleteResult{}, err
	}
	if err := c.appendCanonicalDeleteAudits(
		ctx,
		tx,
		request,
		plan.actor,
		plan.target,
		result,
		deleteAuditID,
		plan.traceID,
		now,
	); err != nil {
		return CanonicalDeleteResult{}, err
	}
	if err := c.completeCanonicalDeleteRecoveryReview(ctx, tx, request, plan, now); err != nil {
		return CanonicalDeleteResult{}, err
	}
	return result, nil
}

// canonicalDeletePlan carries the locked, pre-validated deletion authority. It
// exists only to keep canonicalDeleteInTransaction within the profile function
// length limit without changing ordering or rollback semantics (F004).
type canonicalDeletePlan struct {
	actor   User
	target  User
	review  *RecoveryAccountReview
	traceID string
}

// prepareCanonicalDelete performs every fail-closed check before any write:
// trace authority (F003), actor and target authorization, the identifier
// tombstone invariant (F001), the recovery double proof (F002), and the recovery
// review lock. The order matches the original transaction body.
func (c *CanonicalDeleteCoordinator) prepareCanonicalDelete(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
) (canonicalDeletePlan, error) {
	// trace 单一权威（F003）：ctx 是 canonical 删除 trace 的唯一来源。显式
	// request.TraceID 只作一致性断言，缺失或不一致在任何写入前 fail closed，
	// 且不得回落到任何其它值。
	traceID, err := c.canonicalDeleteTrace(ctx, request)
	if err != nil {
		return canonicalDeletePlan{}, err
	}
	actor, err := c.lockCanonicalDeleteActor(ctx, tx, request)
	if err != nil {
		return canonicalDeletePlan{}, err
	}
	target, err := c.lockCanonicalDeleteTarget(ctx, tx, request, actor)
	if err != nil {
		return canonicalDeletePlan{}, err
	}
	// F001：在任何身份/标识写入之前，锁定该 User 的全部 identifier 行并完成
	// 逐 kind 存在性/kind/值/lifecycle 校验。校验必须前置于所有写入。
	identifiers, err := c.dependencies.Repository.LockIdentifiers(ctx, tx, target.ID)
	if err != nil {
		return canonicalDeletePlan{}, canonicalDeletePersistenceError(
			"lock canonical delete identifiers",
			err,
		)
	}
	if err := validateCanonicalDeleteIdentifierTombstones(target, identifiers); err != nil {
		return canonicalDeletePlan{}, err
	}
	// F002：recovery_kept_deleted 必须由受控能力驱动，并在任何写入前经同一
	// identity-owned RecoveryRunStore 与 maintenance/readiness Gate 双重证明。
	if err := c.authorizeCanonicalDeleteRecovery(ctx, tx, request, target); err != nil {
		return canonicalDeletePlan{}, err
	}
	review, err := c.lockCanonicalDeleteReview(ctx, tx, request, target)
	if err != nil {
		return canonicalDeletePlan{}, err
	}
	return canonicalDeletePlan{
		actor:   actor,
		target:  target,
		review:  review,
		traceID: traceID,
	}, nil
}

// completeCanonicalDeleteRecoveryReview writes decision=kept_deleted only after
// every other effect succeeded (F004 extraction; behavior preserved).
func (c *CanonicalDeleteCoordinator) completeCanonicalDeleteRecoveryReview(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	plan canonicalDeletePlan,
	now time.Time,
) error {
	if plan.review == nil || plan.review.Decision != RecoveryDecisionPending {
		return nil
	}
	if request.Recovery == nil {
		return canonicalDeleteForbiddenError()
	}
	if err := c.dependencies.Repository.CompleteRecoveryAccountReview(
		ctx,
		tx,
		plan.review.RecoveryRunID,
		plan.target.ID,
		plan.actor.ID,
		request.Recovery.EvidenceRef,
		now,
	); err != nil {
		return canonicalDeletePersistenceError(
			"complete kept-deleted recovery review",
			err,
		)
	}
	return nil
}

// applyCanonicalDeleteIdentity performs the identity-owned tombstone and
// credential/session effects plus the article orphaning and email retention
// decision. It is a behavior-preserving extraction of the original single
// transaction body (F004).
func (c *CanonicalDeleteCoordinator) applyCanonicalDeleteIdentity(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	actor User,
	target User,
	traceID string,
	now time.Time,
) (CanonicalDeleteResult, error) {
	if err := c.dependencies.Repository.MarkUserDeleted(
		ctx,
		tx,
		target.ID,
		now,
	); err != nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"write canonical user tombstone",
			err,
		)
	}
	if err := c.dependencies.Repository.ReserveIdentifiers(
		ctx,
		tx,
		target.ID,
	); err != nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"reserve canonical delete identifiers",
			err,
		)
	}
	if err := c.dependencies.Repository.DeleteSessions(ctx, tx, target.ID); err != nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"delete canonical user sessions",
			err,
		)
	}
	if err := c.dependencies.Repository.InvalidateTemporaryCredentials(
		ctx,
		tx,
		target.ID,
		now,
	); err != nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"invalidate canonical user credentials",
			err,
		)
	}
	articleResult, err := c.dependencies.Articles.OrphanArticlesCanonical(
		ctx,
		tx,
		target.ID,
	)
	if err != nil {
		return CanonicalDeleteResult{}, canonicalDeletePersistenceError(
			"orphan canonical user articles",
			err,
		)
	}
	result := CanonicalDeleteResult{
		TargetID:             target.ID,
		NewlyOrphanedCount:   articleResult.NewlyOrphanedCount,
		AlreadyOrphanedCount: articleResult.AlreadyOrphanedCount,
		PendingOrphanCount:   articleResult.PendingOrphanCount,
	}
	if articleResult.PendingOrphanCount == 0 {
		result.EmailIdentifierFreed, err = c.releaseCanonicalDeleteEmail(
			ctx,
			tx,
			request,
			actor,
			target,
			traceID,
			now,
		)
		if err != nil {
			return CanonicalDeleteResult{}, err
		}
	}
	return result, nil
}

// appendCanonicalDeleteAudits appends the canonical delete and (when freed)
// identifier release facts in the caller transaction (F004 extraction).
func (c *CanonicalDeleteCoordinator) appendCanonicalDeleteAudits(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	actor User,
	target User,
	result CanonicalDeleteResult,
	deleteAuditID string,
	traceID string,
	now time.Time,
) error {
	event := canonicalDeleteAuditEvent(
		deleteAuditID,
		now,
		request,
		actor,
		result,
		traceID,
	)
	if err := c.dependencies.Audit.Append(ctx, tx, event); err != nil {
		return canonicalDeletePersistenceError(
			"append canonical delete audit",
			err,
		)
	}
	return nil
}

func (c *CanonicalDeleteCoordinator) lockCanonicalDeleteActor(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
) (User, error) {
	actor, err := c.dependencies.Repository.LockUser(ctx, tx, request.Actor.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, canonicalDeleteForbiddenError()
	}
	if err != nil {
		return User{}, canonicalDeletePersistenceError("lock canonical delete actor", err)
	}
	if actor.ID != request.Actor.UserID ||
		actor.Role != RoleAdmin ||
		actor.IsInitialAdmin != request.Actor.IsInitialAdmin ||
		actor.Status != UserStatusActive ||
		actor.DeletedAt != nil ||
		actor.RecoveryRequired ||
		actor.CredentialMode != CredentialModePermanent ||
		(!actor.IsInitialAdmin && actor.OnboardingCompletedAt == nil) {
		return User{}, canonicalDeleteForbiddenError()
	}
	if request.TargetRole == RoleAdmin && !actor.IsInitialAdmin {
		return User{}, canonicalDeleteForbiddenError()
	}
	if request.Mode == CanonicalDeleteRecoveryKept && !actor.IsInitialAdmin {
		return User{}, canonicalDeleteForbiddenError()
	}
	return actor, nil
}

func (c *CanonicalDeleteCoordinator) lockCanonicalDeleteTarget(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	actor User,
) (User, error) {
	if request.TargetID == actor.ID {
		return User{}, canonicalDeleteForbiddenError()
	}
	target, err := c.dependencies.Repository.LockUser(ctx, tx, request.TargetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, userNotFoundError()
	}
	if err != nil {
		return User{}, canonicalDeletePersistenceError("lock canonical delete target", err)
	}
	if target.Role != request.TargetRole {
		return User{}, userNotFoundError()
	}
	if target.IsInitialAdmin {
		return User{}, canonicalDeleteInitialAdminProtectedError()
	}
	if request.Mode == CanonicalDeleteDailyGovernance &&
		(target.Status == UserStatusDeleted || target.DeletedAt != nil) {
		return User{}, canonicalDeleteInvalidStateError()
	}
	return target, nil
}

func (c *CanonicalDeleteCoordinator) lockCanonicalDeleteReview(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	target User,
) (*RecoveryAccountReview, error) {
	if request.Mode != CanonicalDeleteRecoveryKept {
		return nil, nil
	}
	if request.Recovery == nil {
		return nil, canonicalDeleteForbiddenError()
	}
	review, err := c.dependencies.Repository.LockRecoveryAccountReview(
		ctx,
		tx,
		request.Recovery.Capability.RunID(),
		target.ID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, canonicalDeleteInvalidStateError()
	}
	if err != nil {
		return nil, canonicalDeletePersistenceError(
			"lock kept-deleted recovery review",
			err,
		)
	}
	if review.BackupRole != request.TargetRole ||
		(review.Decision != RecoveryDecisionPending &&
			review.Decision != RecoveryDecisionKeptDeleted) ||
		review.CredentialResetAt != nil {
		return nil, canonicalDeleteInvalidStateError()
	}
	if review.Decision == RecoveryDecisionKeptDeleted &&
		request.Recovery.EvidenceRef != nil &&
		(review.EvidenceRef == nil ||
			*review.EvidenceRef != *request.Recovery.EvidenceRef) {
		return nil, canonicalDeleteInvalidStateError()
	}
	return &review, nil
}

func (c *CanonicalDeleteCoordinator) canonicalDeleteIdentity() (time.Time, string, error) {
	now := c.dependencies.Clock.Now().UTC()
	if now.IsZero() {
		return time.Time{}, "", canonicalDeleteInternalError(
			"read canonical delete time",
			errors.New("clock returned zero time"),
		)
	}
	id, err := c.dependencies.IDs.NewID()
	if err != nil || strings.TrimSpace(id) == "" {
		if err == nil {
			err = errors.New("audit ID is empty")
		}
		return time.Time{}, "", canonicalDeleteInternalError(
			"create canonical delete audit ID",
			err,
		)
	}
	return now, id, nil
}

// canonicalDeleteTrace resolves the single authoritative trace for the deletion
// transaction (F003). context.Context is the only authority; a caller-supplied
// field is accepted only as an exact consistency assertion and never as a
// fallback.
func (c *CanonicalDeleteCoordinator) canonicalDeleteTrace(
	ctx context.Context,
	request CanonicalDeleteRequest,
) (string, error) {
	ctxTrace, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return "", canonicalDeleteInternalError(
			"read canonical delete trace from context",
			errors.New("context trace ID is missing"),
		)
	}
	normalized := strings.TrimSpace(ctxTrace)
	if normalized == "" || normalized != ctxTrace {
		return "", canonicalDeleteInternalError(
			"read canonical delete trace from context",
			errors.New("context trace ID is not canonical"),
		)
	}
	requestTrace := strings.TrimSpace(request.TraceID)
	if requestTrace != "" && requestTrace != ctxTrace {
		return "", canonicalDeleteInvalidStateError()
	}
	return ctxTrace, nil
}

// validateCanonicalDeleteIdentifierTombstones enforces the F001 identifier
// invariant before any write. It branches on the locked user status: path A for
// a not-yet-deleted account (first deletion and the normal kept_deleted replay,
// where users.email/phone may still be present) and path B for a backup that is
// already deleted (where the columns are NULL and retained_email carries the
// original address). Both fail closed on missing rows, orphan rows, value drift,
// or released rows.
func validateCanonicalDeleteIdentifierTombstones(
	target User,
	identifiers []IdentityIdentifier,
) error {
	byKind := make(map[IdentifierKind]IdentityIdentifier, len(identifiers))
	for _, identifier := range identifiers {
		if identifier.UserID != target.ID {
			return canonicalDeleteInvalidStateError()
		}
		if _, duplicate := byKind[identifier.Kind]; duplicate {
			return canonicalDeleteInvalidStateError()
		}
		byKind[identifier.Kind] = identifier
	}
	if target.Status == UserStatusDeleted {
		return validateCanonicalDeleteAlreadyDeletedTombstones(byKind, target)
	}
	return validateCanonicalDeleteActiveTombstones(byKind, target)
}

// validateCanonicalDeleteActiveTombstones is path A (status <> 'deleted').
func validateCanonicalDeleteActiveTombstones(
	byKind map[IdentifierKind]IdentityIdentifier,
	target User,
) error {
	if err := validateCanonicalDeleteIdentifierRow(
		IdentifierUsername,
		target.Username,
		byKind,
	); err != nil {
		return err
	}
	if err := validateCanonicalDeleteOptionalIdentifier(
		IdentifierEmail,
		target.Email,
		byKind,
	); err != nil {
		return err
	}
	return validateCanonicalDeleteOptionalIdentifier(
		IdentifierPhone,
		target.Phone,
		byKind,
	)
}

// validateCanonicalDeleteAlreadyDeletedTombstones is path B (status = 'deleted').
// A replay against an already-deleted backup can only prove phone tombstone
// existence and non-release: the original phone is not retained anywhere, which
// is the residual limitation recorded in design r45 §3.3.10.
func validateCanonicalDeleteAlreadyDeletedTombstones(
	byKind map[IdentifierKind]IdentityIdentifier,
	target User,
) error {
	username, exists := byKind[IdentifierUsername]
	if !exists ||
		username.NormalizedValue != target.Username ||
		username.Lifecycle != IdentifierReserved ||
		username.ReleasedAt != nil {
		return canonicalDeleteInvalidStateError()
	}
	if identifier, exists := byKind[IdentifierEmail]; exists {
		if identifier.Lifecycle != IdentifierReserved ||
			identifier.ReleasedAt != nil {
			return canonicalDeleteInvalidStateError()
		}
		if target.RetainedEmail != nil &&
			identifier.NormalizedValue != *target.RetainedEmail {
			return canonicalDeleteInvalidStateError()
		}
	}
	if identifier, exists := byKind[IdentifierPhone]; exists {
		if identifier.Lifecycle != IdentifierReserved ||
			identifier.ReleasedAt != nil {
			return canonicalDeleteInvalidStateError()
		}
	}
	return nil
}

func validateCanonicalDeleteIdentifierRow(
	kind IdentifierKind,
	value string,
	byKind map[IdentifierKind]IdentityIdentifier,
) error {
	identifier, exists := byKind[kind]
	if !exists {
		return canonicalDeleteInvalidStateError()
	}
	if identifier.NormalizedValue != value ||
		identifier.ReleasedAt != nil ||
		!canonicalDeleteAdvanceableLifecycle(identifier.Lifecycle) {
		return canonicalDeleteInvalidStateError()
	}
	return nil
}

func validateCanonicalDeleteOptionalIdentifier(
	kind IdentifierKind,
	value *string,
	byKind map[IdentifierKind]IdentityIdentifier,
) error {
	_, exists := byKind[kind]
	if value == nil {
		// Orphan row: the user column is NULL but the tombstone row survives.
		if exists {
			return canonicalDeleteInvalidStateError()
		}
		return nil
	}
	return validateCanonicalDeleteIdentifierRow(kind, *value, byKind)
}

func canonicalDeleteAdvanceableLifecycle(lifecycle IdentifierLifecycle) bool {
	return lifecycle == IdentifierActive || lifecycle == IdentifierReserved
}

// authorizeCanonicalDeleteRecovery enforces the F002 double proof before any
// write: the replay must carry a minted capability, the identity-owned recovery
// run authority must corroborate the run and immutable backup reference, and the
// maintenance/readiness gate must be open. Missing authority or gate, mismatch,
// or an outcome outside {running} fails closed.
func (c *CanonicalDeleteCoordinator) authorizeCanonicalDeleteRecovery(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	target User,
) error {
	if request.Mode != CanonicalDeleteRecoveryKept {
		return nil
	}
	if request.Recovery == nil || !request.Recovery.Capability.valid() {
		return canonicalDeleteForbiddenError()
	}
	if c.dependencies.RecoveryRuns == nil || c.dependencies.RecoveryGate == nil {
		return canonicalDeleteInvalidStateError()
	}
	capability := request.Recovery.Capability
	run, err := c.dependencies.RecoveryRuns.LockRecoveryRunForDeletion(
		ctx,
		tx,
		capability.RunID(),
	)
	if errors.Is(err, recovery.ErrRunNotFound) {
		return canonicalDeleteInvalidStateError()
	}
	if err != nil {
		return canonicalDeletePersistenceError(
			"lock kept-deleted recovery run authority",
			err,
		)
	}
	if run.RunID != capability.RunID() ||
		run.BackupRef != capability.BackupRef() ||
		run.Outcome != recovery.RecoveryOutcomeRunning {
		return canonicalDeleteInvalidStateError()
	}
	if err := c.dependencies.RecoveryGate.CheckRecoveryOpenGate(ctx, tx); err != nil {
		return canonicalDeleteInvalidStateError()
	}
	_ = target
	return nil
}

func (c *CanonicalDeleteCoordinator) releaseCanonicalDeleteEmail(
	ctx context.Context,
	tx pgx.Tx,
	request CanonicalDeleteRequest,
	actor User,
	target User,
	traceID string,
	now time.Time,
) (bool, error) {
	released, err := c.dependencies.Repository.ReleaseEmailIdentifier(
		ctx,
		tx,
		target.ID,
		now,
	)
	if err != nil {
		return false, canonicalDeletePersistenceError(
			"release canonical delete email identifier",
			err,
		)
	}
	if !released {
		return false, nil
	}
	id, err := c.dependencies.IDs.NewID()
	if err != nil || strings.TrimSpace(id) == "" {
		if err == nil {
			err = errors.New("audit ID is empty")
		}
		return false, canonicalDeleteInternalError(
			"create identifier release audit ID",
			err,
		)
	}
	event := canonicalDeleteIdentifierAuditEvent(
		id,
		now,
		request,
		actor,
		target.ID,
		traceID,
	)
	if err := c.dependencies.Audit.Append(ctx, tx, event); err != nil {
		return false, canonicalDeletePersistenceError(
			"append identifier release audit",
			err,
		)
	}
	return true, nil
}

func validateCanonicalDeleteRequest(request CanonicalDeleteRequest) error {
	if request.Actor.UserID == "" ||
		request.Actor.Role != string(RoleAdmin) ||
		request.Actor.Status != string(UserStatusActive) ||
		request.Actor.Restricted() ||
		!validAccountStatusUserID(request.Actor.UserID) {
		return canonicalDeleteForbiddenError()
	}
	if request.TargetRole != RoleMember && request.TargetRole != RoleAdmin {
		return canonicalDeleteForbiddenError()
	}
	if !validAccountStatusUserID(request.TargetID) {
		return userNotFoundError()
	}
	if strings.TrimSpace(request.TraceID) == "" {
		return canonicalDeleteInternalError(
			"validate canonical delete trace ID",
			errors.New("trace ID is required"),
		)
	}
	switch request.Reason {
	case AccountDeletionGovernance, AccountDeletionPrivacyWithdrawal:
	default:
		return canonicalDeleteValidationError("reason")
	}
	switch request.Mode {
	case CanonicalDeleteDailyGovernance:
		if request.Recovery != nil {
			return canonicalDeleteInvalidStateError()
		}
	case CanonicalDeleteRecoveryKept:
		if request.Recovery == nil ||
			!request.Recovery.Capability.valid() ||
			!request.Actor.IsInitialAdmin {
			return canonicalDeleteForbiddenError()
		}
	default:
		return canonicalDeleteInvalidStateError()
	}
	return nil
}

func canonicalDeleteAuditEvent(
	id string,
	now time.Time,
	request CanonicalDeleteRequest,
	actor User,
	result CanonicalDeleteResult,
	traceID string,
) audit.Event {
	actorID := actor.ID
	actorRole := string(actor.Role)
	return audit.Event{
		ID: id, OccurredAt: now,
		ActorKind: audit.ActorUser, ActorUserID: &actorID, ActorRole: &actorRole,
		Action: canonicalDeleteAuditAction, TargetType: canonicalDeleteAuditTargetUser,
		TargetID: request.TargetID, Outcome: canonicalDeleteAuditOutcomeSucceeded,
		TraceID: traceID,
		Metadata: map[string]any{
			"mode":                      request.Mode,
			"reason":                    request.Reason,
			"newly_orphaned_count":      result.NewlyOrphanedCount,
			"already_orphaned_count":    result.AlreadyOrphanedCount,
			"pending_orphan_count":      result.PendingOrphanCount,
			"email_identifier_released": result.EmailIdentifierFreed,
		},
	}
}

func canonicalDeleteIdentifierAuditEvent(
	id string,
	now time.Time,
	request CanonicalDeleteRequest,
	actor User,
	targetID string,
	traceID string,
) audit.Event {
	actorID := actor.ID
	actorRole := string(actor.Role)
	return audit.Event{
		ID: id, OccurredAt: now,
		ActorKind: audit.ActorUser, ActorUserID: &actorID, ActorRole: &actorRole,
		Action:     canonicalDeleteIdentifierAuditAction,
		TargetType: canonicalDeleteAuditTargetIdentifier, TargetID: targetID,
		Outcome: canonicalDeleteAuditOutcomeSucceeded, TraceID: traceID,
		Metadata: map[string]any{
			"kind": canonicalDeleteIdentifierKindMetadata,
			"mode": request.Mode,
		},
	}
}

func canonicalDeleteForbiddenError() error {
	return platform.NewPublicError(
		platform.CodeAuthForbidden,
		errors.New("canonical delete operation is forbidden"),
	)
}

func canonicalDeleteInitialAdminProtectedError() error {
	return platform.NewPublicError(
		platform.CodeIdentityInitialAdminProtected,
		errors.New("initial administrator account is protected"),
	)
}

func canonicalDeleteInvalidStateError() error {
	return platform.NewPublicError(
		platform.CodeIdentityInvalidState,
		errors.New("canonical delete state is invalid"),
	)
}

func canonicalDeleteValidationError(field string) error {
	return platform.NewPublicError(
		platform.CodeIdentityValidationFailed,
		errors.New("canonical delete request is invalid"),
		platform.FieldError{Field: field, Code: "invalid"},
	)
}

func canonicalDeletePersistenceError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown persistence failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

func canonicalDeleteInternalError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown canonical delete failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

// PostgresCanonicalDeleteRepository persists identity-owned canonical deletion state.
type PostgresCanonicalDeleteRepository struct{}

// LockRecoveryRunForDeletion is the read-only, transaction-scoped locking read
// required by design r45 §4.2 step 3.1. It is declared on the single
// identity-owned RecoveryRunStore so the recovery run authority remains unique;
// it never transitions run state and must not be used as an alternate authority.
// It is defined here (a file already owned by identity-auth-S9.1) so the S9.1
// write set does not expand beyond its five files.
func (store *RecoveryRunStore) LockRecoveryRunForDeletion(
	ctx context.Context,
	tx pgx.Tx,
	runID string,
) (recovery.RecoveryRun, error) {
	if store == nil || ctx == nil || tx == nil || invalidStoreText(runID) {
		return recovery.RecoveryRun{}, recovery.ErrInvalidRecoveryRun
	}
	return scanRecoveryRun(tx.QueryRow(ctx, selectRecoveryRunForUpdateSQL, runID))
}

// LockUser locks one account for canonical deletion authorization or mutation.
func (PostgresCanonicalDeleteRepository) LockUser(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	if tx == nil {
		return User{}, errors.New("canonical delete transaction is required")
	}
	var user User
	err := tx.QueryRow(ctx, lockCanonicalDeleteUserSQL, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Phone,
		&user.PermanentPasswordDigest,
		&user.Role,
		&user.IsInitialAdmin,
		&user.Status,
		&user.CredentialMode,
		&user.RecoveryRequired,
		&user.OnboardingCompletedAt,
		&user.RetainedEmail,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	return user, err
}

// MarkUserDeleted writes the irreversible identity-owned tombstone.
func (PostgresCanonicalDeleteRepository) MarkUserDeleted(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	deletedAt time.Time,
) error {
	if tx == nil {
		return errors.New("canonical delete transaction is required")
	}
	command, err := tx.Exec(ctx, markCanonicalUserDeletedSQL, userID, deletedAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("canonical user tombstone affected an unexpected row count")
	}
	return nil
}

// LockIdentifiers locks and reads every identifier row for the target user
// before any write, including the zero-row case (F001).
func (PostgresCanonicalDeleteRepository) LockIdentifiers(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) ([]IdentityIdentifier, error) {
	if tx == nil {
		return nil, errors.New("canonical delete transaction is required")
	}
	rows, err := tx.Query(ctx, lockCanonicalDeleteIdentifiersSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var identifiers []IdentityIdentifier
	for rows.Next() {
		var identifier IdentityIdentifier
		if err := rows.Scan(
			&identifier.UserID,
			&identifier.Kind,
			&identifier.NormalizedValue,
			&identifier.Lifecycle,
			&identifier.ReleasedAt,
		); err != nil {
			return nil, err
		}
		identifiers = append(identifiers, identifier)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return identifiers, nil
}

// ReserveIdentifiers permanently reserves the remaining identity identifiers.
func (PostgresCanonicalDeleteRepository) ReserveIdentifiers(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) error {
	if tx == nil {
		return errors.New("canonical delete transaction is required")
	}
	if _, err := tx.Exec(ctx, reserveCanonicalDeleteIdentifiersSQL, userID); err != nil {
		return err
	}
	var complete bool
	if err := tx.QueryRow(
		ctx,
		verifyCanonicalDeleteIdentifiersSQL,
		userID,
	).Scan(&complete); err != nil {
		return err
	}
	if !complete {
		return errors.New("canonical delete identifier inventory is incomplete")
	}
	return nil
}

// DeleteSessions removes every server-side session for the deleted account.
func (PostgresCanonicalDeleteRepository) DeleteSessions(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) error {
	if tx == nil {
		return errors.New("canonical delete transaction is required")
	}
	_, err := tx.Exec(ctx, deleteCanonicalUserSessionsSQL, userID)
	return err
}

// InvalidateTemporaryCredentials removes all current temporary authentication paths.
func (PostgresCanonicalDeleteRepository) InvalidateTemporaryCredentials(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	invalidatedAt time.Time,
) error {
	if tx == nil {
		return errors.New("canonical delete transaction is required")
	}
	_, err := tx.Exec(
		ctx,
		invalidateCanonicalUserCredentialsSQL,
		userID,
		invalidatedAt,
	)
	return err
}

// ReleaseEmailIdentifier clears the retained contact and frees only the email identifier.
func (PostgresCanonicalDeleteRepository) ReleaseEmailIdentifier(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	releasedAt time.Time,
) (bool, error) {
	if tx == nil {
		return false, errors.New("canonical delete transaction is required")
	}
	command, err := tx.Exec(ctx, clearCanonicalRetainedEmailSQL, userID, releasedAt)
	if err != nil {
		return false, err
	}
	if command.RowsAffected() != 1 {
		return false, errors.New("retained email cleanup affected an unexpected row count")
	}
	command, err = tx.Exec(ctx, releaseCanonicalEmailIdentifierSQL, userID)
	if err != nil {
		return false, err
	}
	if command.RowsAffected() > 1 {
		return false, errors.New("email identifier release affected an unexpected row count")
	}
	return command.RowsAffected() == 1, nil
}

// LockRecoveryAccountReview locks the recovery decision completed by canonical replay.
func (PostgresCanonicalDeleteRepository) LockRecoveryAccountReview(
	ctx context.Context,
	tx pgx.Tx,
	recoveryRunID string,
	userID string,
) (RecoveryAccountReview, error) {
	if tx == nil {
		return RecoveryAccountReview{}, errors.New(
			"canonical delete transaction is required",
		)
	}
	var review RecoveryAccountReview
	err := tx.QueryRow(
		ctx,
		lockCanonicalDeleteReviewSQL,
		recoveryRunID,
		userID,
	).Scan(
		&review.RecoveryRunID,
		&review.UserID,
		&review.BackupRole,
		&review.BackupStatus,
		&review.BackupDeletedAt,
		&review.Decision,
		&review.EvidenceRef,
		&review.ReviewedBy,
		&review.ReviewedAt,
		&review.CredentialResetAt,
	)
	return review, err
}

// CompleteRecoveryAccountReview records kept_deleted only after all other effects succeed.
func (PostgresCanonicalDeleteRepository) CompleteRecoveryAccountReview(
	ctx context.Context,
	tx pgx.Tx,
	recoveryRunID string,
	userID string,
	reviewedBy string,
	evidenceRef *string,
	reviewedAt time.Time,
) error {
	if tx == nil {
		return errors.New("canonical delete transaction is required")
	}
	command, err := tx.Exec(
		ctx,
		completeCanonicalDeleteReviewSQL,
		recoveryRunID,
		userID,
		reviewedBy,
		evidenceRef,
		reviewedAt,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return errors.New("kept-deleted review affected an unexpected row count")
	}
	return nil
}

const lockCanonicalDeleteUserSQL = `
	SELECT
		id,
		username,
		email,
		phone,
		permanent_password_digest,
		role,
		is_initial_admin,
		status,
		credential_mode,
		recovery_required,
		onboarding_completed_at,
		retained_email,
		updated_at,
		deleted_at
	FROM users
	WHERE id = $1
	FOR UPDATE
`

const lockCanonicalDeleteIdentifiersSQL = `
	SELECT
		user_id,
		kind,
		normalized_value,
		lifecycle,
		released_at
	FROM identity_identifiers
	WHERE user_id = $1
	FOR UPDATE
`

const markCanonicalUserDeletedSQL = `
	UPDATE users
	SET
		status = 'deleted',
		retained_email = COALESCE(retained_email, email),
		email = NULL,
		phone = NULL,
		permanent_password_digest = NULL,
		permanent_password_generation = NULL,
		updated_at = $2,
		deleted_at = COALESCE(deleted_at, $2)
	WHERE id = $1 AND NOT is_initial_admin
`

const reserveCanonicalDeleteIdentifiersSQL = `
	UPDATE identity_identifiers
	SET lifecycle = 'reserved', released_at = NULL
	WHERE user_id = $1
`

const verifyCanonicalDeleteIdentifiersSQL = `
	SELECT
		EXISTS (
			SELECT 1
			FROM identity_identifiers
			WHERE user_id = $1
				AND kind = 'username'
				AND lifecycle = 'reserved'
				AND released_at IS NULL
		)
		AND (
			(SELECT retained_email FROM users WHERE id = $1) IS NULL
			OR EXISTS (
				SELECT 1
				FROM identity_identifiers
				WHERE user_id = $1
					AND kind = 'email'
					AND lifecycle = 'reserved'
					AND released_at IS NULL
			)
		)
`

const deleteCanonicalUserSessionsSQL = `
	DELETE FROM sessions
	WHERE user_id = $1
`

const invalidateCanonicalUserCredentialsSQL = `
	UPDATE temporary_credentials
	SET state = 'recovery_invalidated',
		invalidated_at = COALESCE(invalidated_at, $2)
	WHERE user_id = $1
		AND state IN ('pending_delivery', 'active', 'expired')
`

const clearCanonicalRetainedEmailSQL = `
	UPDATE users
	SET retained_email = NULL, updated_at = $2
	WHERE id = $1 AND status = 'deleted'
`

const releaseCanonicalEmailIdentifierSQL = `
	DELETE FROM identity_identifiers
	WHERE user_id = $1 AND kind = 'email'
`

const lockCanonicalDeleteReviewSQL = `
	SELECT
		recovery_run_id,
		user_id,
		backup_role,
		backup_status,
		backup_deleted_at,
		decision,
		evidence_ref,
		reviewed_by,
		reviewed_at,
		credential_reset_at
	FROM recovery_account_reviews
	WHERE recovery_run_id = $1 AND user_id = $2
	FOR UPDATE
`

const completeCanonicalDeleteReviewSQL = `
	UPDATE recovery_account_reviews
	SET
		decision = 'kept_deleted',
		evidence_ref = COALESCE($4, evidence_ref),
		reviewed_by = $3,
		reviewed_at = $5,
		credential_reset_at = NULL
	WHERE recovery_run_id = $1
		AND user_id = $2
		AND decision = 'pending'
`

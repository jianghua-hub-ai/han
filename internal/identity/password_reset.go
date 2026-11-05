package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	passwordResetOperation               = "identity.password.reset"
	passwordResetResidualWindowOperation = "identity.smtp.residual_window"
	passwordResetOutcomeSucceeded        = "password_reset"
	passwordResetOutcomeFailed           = "failed"
	passwordResetOutcomeUncertain        = "uncertain"
	passwordResetDeliveryAccepted        = "accepted"
	passwordResetSessionReason           = "password_reset"
	passwordResetSubject                 = "Your password reset temporary credential"
	passwordResetSecretPrefix            = "Temporary credential: "
	passwordResetGeneratorAlphabet       = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	passwordResetGeneratorLetters        = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	passwordResetGeneratorDigits         = "23456789"
)

type PasswordResetRequest struct {
	Actor          session.Principal
	TargetID       string
	TargetRole     Role
	IdempotencyKey string
	TraceID        string
}
type PasswordResetResult struct {
	UserID                       string
	TemporaryCredentialExpiresAt time.Time
	DeliveryStatus               string
}
type PasswordResetSecretGenerator interface{ Generate() (string, error) }
type PasswordResetRepository interface {
	LockUser(context.Context, pgx.Tx, string) (User, error)
	LockLatestTemporaryCredential(context.Context, pgx.Tx, string) (TemporaryCredential, error)
	UpdateTemporaryCredential(context.Context, pgx.Tx, TemporaryCredential) error
	InsertTemporaryCredential(context.Context, pgx.Tx, TemporaryCredential) error
	SetCredentialMode(context.Context, pgx.Tx, string, CredentialMode, time.Time) error
	RevokeSessions(context.Context, pgx.Tx, string, time.Time, string) error
}

type passwordResetCommitOutcome uint8

const (
	passwordResetCommitUnknown passwordResetCommitOutcome = iota
	passwordResetCommitApplied
	passwordResetCommitNotApplied
)

type passwordResetCommitEvidence struct {
	record     idempotency.Record
	credential TemporaryCredential
	auditEvent audit.Event
}
type passwordResetCommitVerification struct{ outcome passwordResetCommitOutcome }
type passwordResetAuditVerification struct {
	exact   bool
	present bool
	err     error
}
type passwordResetCommitRepository interface {
	VerifyPasswordResetCommit(context.Context, pgx.Tx, passwordResetCommitEvidence) (passwordResetCommitVerification, error)
	InvalidatePasswordResetCredential(context.Context, pgx.Tx, TemporaryCredential, time.Time) error
	VerifyPasswordResetAudit(context.Context, pgx.Tx, audit.Event) (passwordResetAuditVerification, error)
}

type PasswordResetDependencies struct {
	Transactions        RegistrationTransactor
	Repository          PasswordResetRepository
	Store               *PasswordResetStore
	PasswordPolicy      RegistrationPasswordPolicy
	PasswordHasher      TemporaryCredentialHasher
	Secrets             PasswordResetSecretGenerator
	Mailer              mailer.Mailer
	IdempotencyKeys     idempotency.DigestKeyring
	IdempotencyLock     time.Duration
	IdempotencyTTL      time.Duration
	SMTPOperationBudget time.Duration
	Audit               RegistrationAuditAppender
	IDs                 RegistrationIDGenerator
	Clock               RegistrationClock
}

type passwordResetCandidateAuthority interface {
	ClassifyExistingCandidate(
		context.Context,
		idempotency.Transaction,
		string,
		idempotency.BeginRequest,
		func() (string, error),
	) (PasswordResetCandidateResult, bool, error)
	BeginOrClassifyCandidate(context.Context, idempotency.Transaction, PasswordResetCandidate) (PasswordResetCandidateResult, error)
}

type passwordResetNewCandidateAuthorizer interface {
	AuthorizeNewCandidate(
		context.Context,
		idempotency.Transaction,
		PasswordResetCandidate,
	) error
}

type PasswordResetCoordinator struct {
	dependencies PasswordResetDependencies
	authority    passwordResetCandidateAuthority
}
type passwordResetPrepared struct {
	actor     User
	target    User
	pending   TemporaryCredential
	secret    string
	result    PasswordResetResult
	operation PasswordResetOperation
	claim     idempotency.Record
	created   bool
	traceID   string
}

func NewPasswordResetService(dependencies PasswordResetDependencies) (*PasswordResetCoordinator, error) {
	activeKey := dependencies.IdempotencyKeys.Keys[dependencies.IdempotencyKeys.ActiveVersion]
	if dependencies.Transactions == nil || dependencies.Repository == nil || dependencies.Store == nil ||
		dependencies.PasswordPolicy == nil || dependencies.PasswordHasher == nil || dependencies.Secrets == nil ||
		dependencies.Mailer == nil || dependencies.Audit == nil || dependencies.IDs == nil || dependencies.Clock == nil ||
		strings.TrimSpace(dependencies.IdempotencyKeys.ActiveVersion) == "" || len(activeKey) == 0 ||
		dependencies.IdempotencyLock <= dependencies.SMTPOperationBudget || dependencies.SMTPOperationBudget <= 0 ||
		dependencies.IdempotencyTTL <= dependencies.IdempotencyLock {
		return nil, errors.New("password reset dependencies are incomplete")
	}
	return &PasswordResetCoordinator{dependencies: dependencies, authority: dependencies.Store}, nil
}

func (c *PasswordResetCoordinator) ResetPassword(ctx context.Context, request PasswordResetRequest) (PasswordResetResult, error) {
	if c == nil {
		return PasswordResetResult{}, passwordResetInternalError("validate password reset service", errors.New("service is not initialized"))
	}
	if err := validatePasswordResetRequest(request); err != nil {
		return PasswordResetResult{}, err
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return PasswordResetResult{}, passwordResetDatabaseError("begin password reset candidate transaction", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	prepared, err := c.prepareDurableCandidate(ctx, tx, request)
	if err != nil {
		return PasswordResetResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return PasswordResetResult{}, administratorOperationUncertain(fmt.Errorf("commit durable password reset candidate: %w", err))
	}
	closed = true
	if !prepared.created {
		return classifyPasswordResetReplay(prepared)
	}
	return c.dispatchPasswordReset(ctx, prepared)
}

func (c *PasswordResetCoordinator) prepareDurableCandidate(ctx context.Context, tx pgx.Tx, request PasswordResetRequest) (passwordResetPrepared, error) {
	actor, err := c.dependencies.Repository.LockUser(ctx, tx, request.Actor.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return passwordResetPrepared{}, passwordResetForbiddenError()
	}
	if err != nil {
		return passwordResetPrepared{}, passwordResetDatabaseError("lock password reset actor", err)
	}
	if err := authorizePasswordResetActor(request.Actor, actor, request.TargetRole); err != nil {
		return passwordResetPrepared{}, err
	}
	target, err := c.dependencies.Repository.LockUser(ctx, tx, request.TargetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return passwordResetPrepared{}, userNotFoundError()
	}
	if err != nil {
		return passwordResetPrepared{}, passwordResetDatabaseError("lock password reset target", err)
	}
	if request.TargetID == actor.ID {
		return passwordResetPrepared{}, passwordResetForbiddenError()
	}
	if target.Role != request.TargetRole {
		return passwordResetPrepared{}, userNotFoundError()
	}
	if target.IsInitialAdmin {
		return passwordResetPrepared{}, accountStatusInitialAdminProtectedError()
	}
	if !passwordResetTargetEligible(target) {
		return passwordResetPrepared{}, accountStatusInvalidStateError()
	}
	begin, err := c.passwordResetBeginRequest(request)
	if err != nil {
		return passwordResetPrepared{}, err
	}
	existing, found, err := c.authority.ClassifyExistingCandidate(
		ctx, tx, target.ID, begin, c.dependencies.IDs.NewID,
	)
	if err != nil {
		return passwordResetPrepared{}, passwordResetIdempotencyError(err)
	}
	if found {
		if existing.Action == PasswordResetCandidateIntegrityHold {
			return passwordResetPrepared{}, administratorOperationUncertain(errors.New("password reset authority integrity hold"))
		}
		return passwordResetPrepared{
			target: target, operation: existing.Operation, claim: existing.Claim,
		}, nil
	}
	current, latestErr := c.dependencies.Repository.LockLatestTemporaryCredential(ctx, tx, target.ID)
	generation := int64(1)
	kind := PasswordResetInitialReset
	var predecessor, replaced *string
	if latestErr == nil {
		generation = current.Generation + 1
		if current.SourceOperationID != nil && strings.TrimSpace(*current.SourceOperationID) != "" {
			kind = PasswordResetResend
			predecessor = current.SourceOperationID
			id := current.ID
			replaced = &id
		}
	} else if !errors.Is(latestErr, pgx.ErrNoRows) {
		return passwordResetPrepared{}, passwordResetDatabaseError("lock current password reset credential", latestErr)
	}
	if authorizer, ok := c.authority.(passwordResetNewCandidateAuthorizer); ok {
		candidate := PasswordResetCandidate{
			Operation: PasswordResetOperation{
				UserID: target.ID, RequestKind: kind,
				PredecessorOperationID: predecessor, ReplacedCredentialID: replaced,
			},
			IdempotencyRequest: begin,
		}
		if err := authorizer.AuthorizeNewCandidate(ctx, tx, candidate); err != nil {
			return passwordResetPrepared{}, err
		}
	}
	secret, err := c.dependencies.Secrets.Generate()
	if err != nil {
		return passwordResetPrepared{}, passwordResetInternalError("generate password reset credential", err)
	}
	if err := c.dependencies.PasswordPolicy.Validate(secret); err != nil {
		return passwordResetPrepared{}, passwordResetInternalError("validate generated password reset credential", err)
	}
	credentialID, err := c.dependencies.IDs.NewID()
	if err != nil {
		return passwordResetPrepared{}, identifierGenerationError(err)
	}
	operationID, err := c.dependencies.IDs.NewID()
	if err != nil {
		return passwordResetPrepared{}, identifierGenerationError(err)
	}
	now := begin.Now.UTC()
	pending, err := NewTemporaryCredential(c.dependencies.PasswordHasher, TemporaryCredentialInput{ID: credentialID, UserID: target.ID, Purpose: TemporaryCredentialPasswordReset, Generation: generation, Secret: secret, DeliveredToEmail: strings.TrimSpace(*target.Email), IssuedAt: now})
	if err != nil {
		return passwordResetPrepared{}, passwordResetInternalError("create password reset credential", err)
	}
	operation := PasswordResetOperation{ID: operationID, UserID: target.ID, TemporaryCredentialID: credentialID, RequestKind: kind, PredecessorOperationID: predecessor, ReplacedCredentialID: replaced, SettlementState: PasswordResetSettlementPending, DeliveryState: PasswordResetDeliveryNotStarted, State: PasswordResetOperationPrepared, CreatedAt: now, UpdatedAt: now}
	candidate := PasswordResetCandidate{Operation: operation, Credential: pending, RecoveryJob: PasswordResetRecoveryJob{OperationID: operationID, Status: PasswordResetRecoveryJobReady, NextAttemptAt: now.Add(c.dependencies.IdempotencyLock), CreatedAt: now, UpdatedAt: now}, IdempotencyRequest: begin}
	result, err := c.authority.BeginOrClassifyCandidate(ctx, tx, candidate)
	if err != nil {
		return passwordResetPrepared{}, passwordResetIdempotencyError(err)
	}
	if result.Action == PasswordResetCandidateIntegrityHold {
		return passwordResetPrepared{}, administratorOperationUncertain(errors.New("password reset authority integrity hold"))
	}
	return passwordResetPrepared{actor: actor, target: target, pending: pending, secret: secret, operation: result.Operation, claim: result.Claim, created: result.Action == PasswordResetCandidateCreated, traceID: request.TraceID, result: PasswordResetResult{UserID: target.ID, TemporaryCredentialExpiresAt: pending.ExpiresAt}}, nil
}

func classifyPasswordResetReplay(prepared passwordResetPrepared) (PasswordResetResult, error) {
	prepared.secret = ""
	switch prepared.operation.State {
	case PasswordResetOperationSucceeded:
		if prepared.claim.CreatedAt.IsZero() {
			return PasswordResetResult{}, passwordResetInternalError(
				"replay password reset",
				errors.New("durable claim creation time is missing"),
			)
		}
		return PasswordResetResult{
			UserID: prepared.target.ID,
			TemporaryCredentialExpiresAt: prepared.claim.CreatedAt.Add(
				temporaryCredentialLifetime,
			),
			DeliveryStatus: passwordResetDeliveryAccepted,
		}, nil
	case PasswordResetOperationFailed, PasswordResetOperationInvalidated:
		return PasswordResetResult{}, platform.NewPublicError(platform.CodeIdentityEmailDeliveryFailed, errors.New("password reset replayed stable delivery failure"))
	default:
		return PasswordResetResult{}, administratorOperationUncertain(errors.New("password reset durable operation is unresolved"))
	}
}

func (c *PasswordResetCoordinator) dispatchPasswordReset(ctx context.Context, prepared passwordResetPrepared) (PasswordResetResult, error) {
	if prepared.operation.Recovery != nil {
		return PasswordResetResult{}, errRecoveryResetOrdinaryDispatch
	}
	attemptID, err := c.dependencies.IDs.NewID()
	if err != nil {
		return PasswordResetResult{}, identifierGenerationError(err)
	}
	attemptedAt := c.dependencies.Clock.Now().UTC()
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return PasswordResetResult{}, passwordResetDatabaseError("begin password reset dispatch transaction", err)
	}
	if err := c.dependencies.Store.ClaimMailAttempt(
		ctx, tx, prepared.operation.UserID, prepared.operation.ID, attemptID, attemptedAt,
	); err != nil {
		_ = tx.Rollback(ctx)
		return PasswordResetResult{}, administratorOperationUncertain(fmt.Errorf("claim password reset dispatch: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		return PasswordResetResult{}, administratorOperationUncertain(fmt.Errorf("commit password reset dispatch ownership: %w", err))
	}
	mailCtx, cancelMail := context.WithTimeout(ctx, c.dependencies.SMTPOperationBudget)
	mailErr := c.dependencies.Mailer.Send(mailCtx, mailer.Message{Purpose: mailer.PurposePasswordResetCredential, To: strings.TrimSpace(*prepared.target.Email), Subject: passwordResetSubject, Body: passwordResetSecretPrefix + prepared.secret})
	cancelMail()
	prepared.secret = ""
	evidenceCtx, cancelEvidence := c.passwordResetReconciliationContext(ctx)
	defer cancelEvidence()
	if mailErr != nil {
		if errors.Is(mailErr, mailer.ErrMessageRejected) {
			return PasswordResetResult{}, c.recordRejectedPasswordReset(
				evidenceCtx, prepared, attemptID, mailErr,
			)
		}
		return PasswordResetResult{}, c.recordUnknownPasswordReset(
			evidenceCtx, prepared, attemptID, mailErr,
		)
	}
	if err := c.recordAcceptedPasswordReset(
		evidenceCtx, prepared.operation.UserID, prepared.operation.ID, attemptID,
	); err != nil {
		return PasswordResetResult{}, administratorOperationUncertain(err)
	}
	prepared.result.DeliveryStatus = passwordResetDeliveryAccepted
	return prepared.result, nil
}

func (c *PasswordResetCoordinator) passwordResetReconciliationContext(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(ctx),
		c.dependencies.SMTPOperationBudget,
	)
}

func (c *PasswordResetCoordinator) recordAcceptedPasswordReset(
	ctx context.Context,
	userID string,
	operationID string,
	attemptID string,
) error {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return fmt.Errorf("begin accepted delivery evidence: %w", err)
	}
	now := c.dependencies.Clock.Now().UTC()
	if err := c.dependencies.Store.RecordAcceptedDelivery(
		ctx, tx, userID, operationID, attemptID, now,
	); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("record accepted delivery evidence: %w", err)
	}
	if _, err := tx.Exec(ctx, acceleratePasswordResetRecoveryJobSQL, now, operationID); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("accelerate accepted recovery job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit accepted delivery evidence: %w", err)
	}
	return nil
}

func (c *PasswordResetCoordinator) recordUnknownPasswordReset(ctx context.Context, prepared passwordResetPrepared, attemptID string, cause error) error {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	now := c.dependencies.Clock.Now().UTC()
	if err := c.dependencies.Store.RecordUnknownDelivery(
		ctx, tx, prepared.operation.UserID, prepared.operation.ID, attemptID, now,
	); err != nil {
		_ = tx.Rollback(ctx)
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	if err := tx.Commit(ctx); err != nil {
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	return administratorOperationUncertain(fmt.Errorf("password reset delivery result is unknown: %w", cause))
}

func (c *PasswordResetCoordinator) recordRejectedPasswordReset(
	ctx context.Context,
	prepared passwordResetPrepared,
	attemptID string,
	cause error,
) error {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	now := c.dependencies.Clock.Now().UTC()
	if err := c.dependencies.Store.RecordRejectedDelivery(
		ctx,
		tx,
		prepared.operation,
		prepared.pending.ID,
		attemptID,
		now,
	); err != nil {
		_ = tx.Rollback(ctx)
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	if err := tx.Commit(ctx); err != nil {
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	return platform.NewPublicError(platform.CodeIdentityEmailDeliveryFailed, cause)
}

// ClassifyExistingCandidate resolves durable authority before sensitive preparation.
func (store *PasswordResetStore) ClassifyExistingCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
	request idempotency.BeginRequest,
	newHoldID func() (string, error),
) (PasswordResetCandidateResult, bool, error) {
	if store == nil || tx == nil || strings.TrimSpace(userID) == "" || newHoldID == nil {
		return PasswordResetCandidateResult{}, false, ErrPasswordResetCandidateInvalid
	}
	user, err := lockPasswordResetCandidateUser(ctx, tx, userID)
	if err != nil {
		return PasswordResetCandidateResult{}, false, err
	}
	stored, found, err := store.Idempotency.ResolvePasswordResetClaim(ctx, tx, request)
	if err != nil {
		return PasswordResetCandidateResult{}, found, err
	}
	if !found {
		claim, err := idempotency.NewPasswordResetClaim(
			request, user.executionGeneration+1,
		)
		if err != nil {
			return PasswordResetCandidateResult{}, false, err
		}
		orphan, orphanFound, err := readPasswordResetOrphanOperation(
			ctx, tx, claim.ScopeHash, claim.KeyLocator,
		)
		if err != nil {
			return PasswordResetCandidateResult{}, false, err
		}
		if !orphanFound && !user.integrityHold {
			return PasswordResetCandidateResult{}, false, nil
		}
		holdID, err := newHoldID()
		if err != nil {
			return PasswordResetCandidateResult{}, true, err
		}
		attempt := passwordResetCandidateAttempt{
			candidate: PasswordResetCandidate{
				Operation:          PasswordResetOperation{ID: holdID, UserID: userID},
				IdempotencyRequest: request,
			},
			user: user, claim: claim, generation: user.executionGeneration + 1,
		}
		var operationID *string
		if orphanFound && orphan.UserID == userID {
			operationID = &orphan.ID
		}
		result, err := holdPasswordResetCandidate(
			ctx, tx, attempt, operationID, orphan, idempotency.Record{},
		)
		return result, true, err
	}

	candidate, err := passwordResetExistingCandidate(ctx, tx, userID, stored)
	if err != nil {
		return PasswordResetCandidateResult{}, true, err
	}
	attempt := passwordResetCandidateAttempt{
		candidate: candidate, user: user, claim: stored,
		generation: user.executionGeneration + 1,
	}
	binding, err := readPasswordResetCandidateBinding(ctx, tx, candidate, stored)
	if err != nil {
		return PasswordResetCandidateResult{}, true, err
	}
	if binding.exact && !user.integrityHold {
		return PasswordResetCandidateResult{
			Action: PasswordResetCandidateExisting, Operation: binding.operation,
			Claim: binding.claim,
		}, true, nil
	}
	holdID, err := newHoldID()
	if err != nil {
		return PasswordResetCandidateResult{}, true, err
	}
	attempt.candidate.Operation.ID = holdID
	result, err := classifyStoredPasswordResetCandidate(ctx, tx, attempt, stored)
	return result, true, err
}

func passwordResetExistingCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
	stored idempotency.Record,
) (PasswordResetCandidate, error) {
	candidate := PasswordResetCandidate{
		Operation: PasswordResetOperation{
			UserID: userID, RequestKind: PasswordResetInitialReset,
		},
	}
	row, err := scanPasswordResetCandidateBinding(ctx, tx, candidate, stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return candidate, ErrPasswordResetStateConflict
	}
	if err != nil {
		return candidate, fmt.Errorf("read password reset replay binding: %w", err)
	}
	if row.operationID != nil {
		candidate.Operation.ID = *row.operationID
	}
	if row.requestKind == nil {
		return candidate, nil
	}
	candidate.Operation.RequestKind = *row.requestKind
	if *row.requestKind != PasswordResetResend || row.operationID == nil {
		return candidate, nil
	}
	predecessor, replaced, err := passwordResetExistingResendRelation(
		ctx, tx, *row.operationID, userID,
	)
	if err != nil {
		return PasswordResetCandidate{}, err
	}
	candidate.Operation.PredecessorOperationID = predecessor
	candidate.Operation.ReplacedCredentialID = replaced
	return candidate, nil
}

func passwordResetExistingResendRelation(
	ctx context.Context,
	tx idempotency.Transaction,
	operationID string,
	userID string,
) (*string, *string, error) {
	var operation passwordResetSettlementOperation
	var attemptCount int64
	var leaseOwner *string
	var leaseExpiresAt *time.Time
	err := tx.QueryRow(ctx, passwordResetSettlementLockOperationSQL, operationID).Scan(
		&operation.state, &operation.delivery, &operation.settlement,
		&operation.credentialID, &operation.requestKind, &operation.replacedCredentialID,
		&operation.baseGeneration, &operation.targetGeneration, &operation.scopeHash,
		&operation.keyLocator, &operation.keyDigest, &operation.requestDigest,
		&operation.digestKeyVersion, &operation.executionGeneration,
		&attemptCount, &leaseOwner, &leaseExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read password reset replay operation: %w", err)
	}
	if operation.requestKind != PasswordResetResend ||
		operation.replacedCredentialID == nil {
		return nil, operation.replacedCredentialID, nil
	}
	replaced := *operation.replacedCredentialID
	var credentialUser string
	var state TemporaryCredentialState
	var predecessor *string
	var authGeneration int64
	err = tx.QueryRow(ctx, passwordResetSettlementLockCredentialSQL, replaced).Scan(
		&credentialUser, &state, &predecessor, &authGeneration,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &replaced, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read password reset replay predecessor: %w", err)
	}
	if credentialUser != userID {
		return nil, &replaced, nil
	}
	return predecessor, &replaced, nil
}

// ClaimMailAttempt grants the only SMTP send right after a User-first transition.
func (store *PasswordResetStore) ClaimMailAttempt(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	operationID string,
	attemptID string,
	now time.Time,
) error {
	if store == nil || tx == nil {
		return errors.New("password reset dispatch store is unavailable")
	}
	if err := lockPasswordResetRequestUser(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, claimPasswordResetDispatchSQL, attemptID, now, operationID, userID)
	return errors.Join(err, requireOnePasswordResetRow(tag))
}

// RecordAcceptedDelivery persists evidence only; the existing operator owns settlement.
func (store *PasswordResetStore) RecordAcceptedDelivery(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	operationID string,
	attemptID string,
	now time.Time,
) error {
	if store == nil || tx == nil {
		return errors.New("password reset delivery store is unavailable")
	}
	if err := lockPasswordResetRequestUser(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, recordPasswordResetAcceptedSQL, now, operationID, attemptID, userID)
	return errors.Join(err, requireOnePasswordResetRow(tag))
}

// RecordUnknownDelivery persists indeterminate provider evidence without settling.
func (store *PasswordResetStore) RecordUnknownDelivery(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	operationID string,
	attemptID string,
	now time.Time,
) error {
	if store == nil || tx == nil {
		return errors.New("password reset delivery store is unavailable")
	}
	if err := lockPasswordResetRequestUser(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(
		ctx, recordPasswordResetUnknownSQL, now, operationID, attemptID, userID,
	)
	return errors.Join(err, requireOnePasswordResetRow(tag))
}

// RecordRejectedDelivery records a proved non-acceptance and invalidates only this candidate.
func (store *PasswordResetStore) RecordRejectedDelivery(
	ctx context.Context,
	tx pgx.Tx,
	operation PasswordResetOperation,
	credentialID string,
	attemptID string,
	now time.Time,
) error {
	if store == nil || tx == nil {
		return errors.New("password reset delivery store is unavailable")
	}
	if err := lockPasswordResetRequestUser(ctx, tx, operation.UserID); err != nil {
		return err
	}
	tag, err := tx.Exec(
		ctx, recordPasswordResetRejectedSQL, now, operation.ID, attemptID, operation.UserID,
	)
	if err = errors.Join(err, requireOnePasswordResetRow(tag)); err != nil {
		return err
	}
	tag, err = tx.Exec(
		ctx, invalidateRejectedPasswordResetCredentialSQL, now, credentialID, operation.UserID,
	)
	if err = errors.Join(err, requireOnePasswordResetRow(tag)); err != nil {
		return err
	}
	tag, err = tx.Exec(ctx, completeRejectedPasswordResetJobSQL, now, operation.ID)
	return errors.Join(err, requireOnePasswordResetRow(tag))
}

func lockPasswordResetRequestUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var lockedID string
	if err := tx.QueryRow(ctx, lockPasswordResetRequestUserSQL, userID).Scan(&lockedID); err != nil {
		return fmt.Errorf("lock password reset request user: %w", err)
	}
	if lockedID != userID {
		return ErrPasswordResetStateConflict
	}
	return nil
}

func (c *PasswordResetCoordinator) passwordResetBeginRequest(request PasswordResetRequest) (idempotency.BeginRequest, error) {
	encoded, err := json.Marshal(struct {
		TargetID   string `json:"target_id"`
		TargetRole Role   `json:"target_role"`
	}{request.TargetID, request.TargetRole})
	if err != nil {
		return idempotency.BeginRequest{}, passwordResetInternalError("encode password reset idempotency request", err)
	}
	now := c.dependencies.Clock.Now().UTC()
	scope := []byte(request.Actor.UserID + "\x00" + passwordResetOperation + "\x00" + request.TargetID)
	return idempotency.BeginRequest{Scope: scope, Key: request.IdempotencyKey, Request: encoded, DigestKeys: c.dependencies.IdempotencyKeys, LockedUntil: now.Add(c.dependencies.IdempotencyLock), ExpiresAt: now.Add(c.dependencies.IdempotencyTTL), Now: now}, nil
}

const lockPasswordResetRequestUserSQL = `SELECT id FROM users WHERE id=$1 FOR UPDATE`
const claimPasswordResetDispatchSQL = `UPDATE password_reset_operations SET state='dispatching',delivery_state='dispatching',delivery_attempt_id=$1,delivery_attempted_at=$2,updated_at=$2 WHERE operation_id=$3 AND user_id=$4 AND state='prepared' AND delivery_state='not_started' AND settlement_state='pending' AND delivery_attempt_id IS NULL AND delivery_attempted_at IS NULL`
const recordPasswordResetAcceptedSQL = `UPDATE password_reset_operations SET state='recovery_required',delivery_state='accepted',delivery_result_recorded_at=$1,updated_at=$1 WHERE operation_id=$2 AND delivery_attempt_id=$3 AND user_id=$4 AND state='dispatching' AND delivery_state='dispatching' AND settlement_state='pending'`
const recordPasswordResetUnknownSQL = `UPDATE password_reset_operations SET state='commit_unknown',delivery_state='unknown',delivery_result_recorded_at=$1,updated_at=$1,recovery_reason='delivery_unknown' WHERE operation_id=$2 AND delivery_attempt_id=$3 AND user_id=$4 AND state='dispatching' AND delivery_state='dispatching' AND settlement_state='pending'`
const acceleratePasswordResetRecoveryJobSQL = `UPDATE password_reset_recovery_jobs SET status='ready',next_attempt_at=$1,updated_at=$1 WHERE operation_id=$2 AND status IN ('ready','retry_wait')`
const recordPasswordResetRejectedSQL = `UPDATE password_reset_operations SET state='failed',delivery_state='rejected',settlement_state='invalidated',delivery_result_recorded_at=$1,settled_at=$1,terminal_at=$1,updated_at=$1,recovery_reason='delivery_rejected' WHERE operation_id=$2 AND delivery_attempt_id=$3 AND user_id=$4 AND state='dispatching' AND delivery_state='dispatching' AND settlement_state='pending'`
const invalidateRejectedPasswordResetCredentialSQL = `UPDATE temporary_credentials SET state='recovery_invalidated',invalidated_at=$1 WHERE id=$2 AND user_id=$3 AND state='pending_delivery'`
const completeRejectedPasswordResetJobSQL = `UPDATE password_reset_recovery_jobs SET status='succeeded',updated_at=$1,completed_at=$1 WHERE operation_id=$2 AND status IN ('ready','retry_wait')`

func validatePasswordResetRequest(request PasswordResetRequest) error {
	if request.Actor.UserID == "" || request.Actor.Role != string(RoleAdmin) ||
		request.Actor.Status != string(UserStatusActive) || request.Actor.Restricted() {
		return passwordResetForbiddenError()
	}
	if request.TargetRole != RoleMember && request.TargetRole != RoleAdmin {
		return passwordResetForbiddenError()
	}
	if !validAccountStatusUserID(request.Actor.UserID) {
		return passwordResetForbiddenError()
	}
	if !validAccountStatusUserID(request.TargetID) {
		return userNotFoundError()
	}
	if strings.TrimSpace(request.TraceID) == "" {
		return passwordResetInternalError(
			"validate password reset trace ID",
			errors.New("trace ID is required"),
		)
	}
	if !validIdempotencyKey(request.IdempotencyKey) {
		return platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("password reset idempotency key is invalid"),
		)
	}
	return nil
}

func authorizePasswordResetActor(
	principal session.Principal,
	actor User,
	targetRole Role,
) error {
	if actor.ID != principal.UserID || actor.Role != RoleAdmin ||
		actor.IsInitialAdmin != principal.IsInitialAdmin ||
		actor.Status != UserStatusActive || actor.DeletedAt != nil ||
		actor.RecoveryRequired || actor.CredentialMode != CredentialModePermanent {
		return passwordResetForbiddenError()
	}
	if !actor.IsInitialAdmin && actor.OnboardingCompletedAt == nil {
		return passwordResetForbiddenError()
	}
	if targetRole == RoleAdmin && !actor.IsInitialAdmin {
		return passwordResetForbiddenError()
	}
	return nil
}

func passwordResetTargetEligible(target User) bool {
	if target.DeletedAt != nil || target.Status == UserStatusDeleted ||
		target.RecoveryRequired || target.Email == nil ||
		strings.TrimSpace(*target.Email) == "" {
		return false
	}
	switch target.CredentialMode {
	case CredentialModePermanent, CredentialModeTemporary:
		return target.Status == UserStatusActive || target.Status == UserStatusSuspended
	default:
		return false
	}
}

func replayPasswordReset(
	record idempotency.Record,
	targetID string,
) (PasswordResetResult, error) {
	if record.ResponseSnapshot == nil || record.CreatedAt.IsZero() {
		return PasswordResetResult{}, passwordResetInternalError(
			"replay password reset",
			errors.New("replay snapshot is incomplete"),
		)
	}
	snapshot := *record.ResponseSnapshot
	if record.Status == idempotency.StatusSucceeded &&
		snapshot.Outcome == passwordResetOutcomeSucceeded &&
		snapshot.ResourceID == targetID {
		return PasswordResetResult{
			UserID:                       targetID,
			TemporaryCredentialExpiresAt: record.CreatedAt.Add(temporaryCredentialLifetime),
			DeliveryStatus:               passwordResetDeliveryAccepted,
		}, nil
	}
	if record.Status == idempotency.StatusFailed &&
		snapshot.Outcome == passwordResetOutcomeFailed &&
		snapshot.ErrorCode == string(platform.CodeIdentityEmailDeliveryFailed) {
		return PasswordResetResult{}, platform.NewPublicError(
			platform.CodeIdentityEmailDeliveryFailed,
			errors.New("password reset replayed stable delivery failure"),
		)
	}
	return PasswordResetResult{}, passwordResetInternalError(
		"replay password reset",
		errors.New("replay state is invalid"),
	)
}

func passwordResetAuditEvent(
	id string,
	now time.Time,
	traceID string,
	actor User,
	target User,
) audit.Event {
	actorID, actorRole := actor.ID, string(actor.Role)
	return audit.Event{
		ID: id, OccurredAt: now, ActorKind: audit.ActorUser,
		ActorUserID: &actorID, ActorRole: &actorRole,
		Action: passwordResetOperation, TargetType: "user", TargetID: target.ID,
		Outcome:  passwordResetOutcomeSucceeded,
		TraceID:  traceID,
		Metadata: passwordResetAuditMetadata(target, true),
	}
}

func passwordResetUncertainAuditEvent(
	id string,
	now time.Time,
	traceID string,
	actor User,
	target User,
) audit.Event {
	event := passwordResetAuditEvent(id, now, traceID, actor, target)
	event.Action = passwordResetResidualWindowOperation
	event.Outcome = passwordResetOutcomeUncertain
	code := string(platform.CodeIdentityOperationUncertain)
	event.ErrorCode = &code
	event.Metadata["failure_stage"] = "accepted_persistence"
	return event
}

func passwordResetAuditMetadata(target User, accepted bool) map[string]any {
	metadata := map[string]any{
		"target_role":        target.Role,
		"credential_purpose": TemporaryCredentialPasswordReset,
	}
	if accepted {
		metadata["delivery_status"] = passwordResetDeliveryAccepted
	}
	return metadata
}

func passwordResetIdempotencyError(err error) error {
	switch {
	case errors.Is(err, idempotency.ErrDigestConflict):
		return platform.NewPublicError(platform.CodeIdentityIdempotencyConflict, err)
	case errors.Is(err, idempotency.ErrDigestKeyUnavailable):
		return administratorOperationUncertain(err)
	case errors.Is(err, ErrPasswordResetGateClosed),
		errors.Is(err, ErrPasswordResetCandidateInvalid),
		errors.Is(err, ErrPasswordResetStateConflict):
		return administratorOperationUncertain(err)
	default:
		return passwordResetDatabaseError("apply password reset idempotency", err)
	}
}

func passwordResetForbiddenError() error {
	return platform.NewPublicError(
		platform.CodeAuthForbidden,
		errors.New("password reset is forbidden"),
	)
}

func passwordResetDatabaseError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown password reset database failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

func passwordResetInternalError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown password reset failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

// CryptoTemporaryPasswordGenerator generates policy-shaped random passwords.
type CryptoTemporaryPasswordGenerator struct {
	random io.Reader
	length int
}

// NewCryptoTemporaryPasswordGenerator creates a configurable random generator.
func NewCryptoTemporaryPasswordGenerator(
	random io.Reader,
	length int,
) (*CryptoTemporaryPasswordGenerator, error) {
	if random == nil || length < 8 {
		return nil, errors.New("temporary password generator configuration is invalid")
	}
	return &CryptoTemporaryPasswordGenerator{random: random, length: length}, nil
}

// Generate returns a random password containing letters and digits.
func (g *CryptoTemporaryPasswordGenerator) Generate() (string, error) {
	if g == nil || g.random == nil || g.length < 8 {
		return "", errors.New("temporary password generator is unavailable")
	}
	value := make([]byte, g.length)
	var err error
	value[0], err = randomAlphabetByte(g.random, passwordResetGeneratorLetters)
	if err != nil {
		return "", err
	}
	value[1], err = randomAlphabetByte(g.random, passwordResetGeneratorDigits)
	if err != nil {
		return "", err
	}
	for index := 2; index < len(value); index++ {
		value[index], err = randomAlphabetByte(g.random, passwordResetGeneratorAlphabet)
		if err != nil {
			return "", err
		}
	}
	if err := shufflePassword(g.random, value); err != nil {
		return "", err
	}
	return string(value), nil
}

func randomAlphabetByte(random io.Reader, alphabet string) (byte, error) {
	var sample [1]byte
	limit := 256 - (256 % len(alphabet))
	for {
		if _, err := io.ReadFull(random, sample[:]); err != nil {
			return 0, fmt.Errorf("read temporary password randomness: %w", err)
		}
		if int(sample[0]) < limit {
			return alphabet[int(sample[0])%len(alphabet)], nil
		}
	}
}

func shufflePassword(random io.Reader, value []byte) error {
	for index := len(value) - 1; index > 0; index-- {
		var sample [1]byte
		limit := 256 - (256 % (index + 1))
		for {
			if _, err := io.ReadFull(random, sample[:]); err != nil {
				return fmt.Errorf("shuffle temporary password: %w", err)
			}
			if int(sample[0]) < limit {
				swap := int(sample[0]) % (index + 1)
				value[index], value[swap] = value[swap], value[index]
				break
			}
		}
	}
	return nil
}

// PostgresPasswordResetRepository persists password reset state.
type PostgresPasswordResetRepository struct{}

// LockUser locks one current user state.
func (PostgresPasswordResetRepository) LockUser(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	if tx == nil {
		return User{}, errors.New("password reset transaction is required")
	}
	var user User
	err := tx.QueryRow(ctx, lockPasswordResetUserSQL, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone,
		&user.PermanentPasswordDigest, &user.Role, &user.IsInitialAdmin,
		&user.Status, &user.CredentialMode, &user.RecoveryRequired,
		&user.OnboardingPrivacyAcknowledgedVersion, &user.OnboardingCompletedAt,
		&user.RetainedEmail, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	return user, err
}

// LockLatestTemporaryCredential locks the highest credential generation.
func (PostgresPasswordResetRepository) LockLatestTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (TemporaryCredential, error) {
	if tx == nil {
		return TemporaryCredential{}, errors.New("password reset transaction is required")
	}
	var credential TemporaryCredential
	err := tx.QueryRow(ctx, lockPasswordResetCredentialSQL, userID).Scan(
		&credential.ID, &credential.UserID, &credential.Purpose,
		&credential.Generation, &credential.State, &credential.SecretDigest,
		&credential.DeliveredToEmail, &credential.IssuedAt, &credential.ExpiresAt,
		&credential.SupersededAt, &credential.ConsumedAt, &credential.InvalidatedAt,
	)
	return credential, err
}

// UpdateTemporaryCredential persists a superseded current credential.
func (PostgresPasswordResetRepository) UpdateTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	if tx == nil {
		return errors.New("password reset transaction is required")
	}
	tag, err := tx.Exec(
		ctx, supersedePasswordResetCredentialSQL,
		credential.ID, credential.UserID, credential.Generation, credential.SupersededAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("password reset current credential changed")
	}
	return nil
}

// InsertTemporaryCredential inserts the new active reset credential.
func (PostgresPasswordResetRepository) InsertTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	return (Repository{}).InsertTemporaryCredential(ctx, tx, credential)
}

// SetCredentialMode selects only the temporary credential path.
func (PostgresPasswordResetRepository) SetCredentialMode(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	mode CredentialMode,
	updatedAt time.Time,
) error {
	if tx == nil {
		return errors.New("password reset transaction is required")
	}
	tag, err := tx.Exec(ctx, setPasswordResetCredentialModeSQL, userID, mode, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("password reset target state changed")
	}
	return nil
}

// RevokeSessions revokes every target session.
func (PostgresPasswordResetRepository) RevokeSessions(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	revokedAt time.Time,
	reason string,
) error {
	_, err := (Repository{}).RevokeSessionsByUserID(ctx, tx, userID, revokedAt, reason)
	return err
}

// VerifyPasswordResetCommit checks the immutable success identities after commit error.
func (PostgresPasswordResetRepository) VerifyPasswordResetCommit(
	ctx context.Context,
	tx pgx.Tx,
	evidence passwordResetCommitEvidence,
) (passwordResetCommitVerification, error) {
	if tx == nil {
		return passwordResetCommitVerification{}, errors.New("password reset transaction is required")
	}
	snapshot, err := json.Marshal(struct {
		Outcome    string `json:"outcome"`
		ResourceID string `json:"resource_id"`
	}{Outcome: passwordResetOutcomeSucceeded, ResourceID: evidence.credential.UserID})
	if err != nil {
		return passwordResetCommitVerification{}, err
	}
	metadata, err := json.Marshal(evidence.auditEvent.Metadata)
	if err != nil {
		return passwordResetCommitVerification{}, err
	}
	committedRecord, err := passwordResetRecordExists(ctx, tx, evidence.record, snapshot)
	if err != nil {
		return passwordResetCommitVerification{}, err
	}
	processingRecord, err := passwordResetProcessingRecordExists(ctx, tx, evidence.record)
	if err != nil {
		return passwordResetCommitVerification{}, err
	}
	exactCredential, credentialPresent, err := passwordResetCredentialEvidence(
		ctx, tx, evidence.credential,
	)
	if err != nil {
		return passwordResetCommitVerification{}, err
	}
	exactAudit, auditPresent, err := passwordResetAuditEvidence(
		ctx, tx, evidence.auditEvent, metadata,
	)
	if err != nil {
		return passwordResetCommitVerification{}, err
	}
	verification := passwordResetCommitVerification{}
	switch {
	case committedRecord && exactCredential && exactAudit:
		verification.outcome = passwordResetCommitApplied
	case processingRecord && !credentialPresent && !auditPresent:
		verification.outcome = passwordResetCommitNotApplied
	default:
		verification.outcome = passwordResetCommitUnknown
	}
	return verification, nil
}

// InvalidatePasswordResetCredential makes an indeterminate mailed value unusable.
func (PostgresPasswordResetRepository) InvalidatePasswordResetCredential(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
	invalidatedAt time.Time,
) error {
	if tx == nil || invalidatedAt.IsZero() {
		return errors.New("password reset invalidation transaction is required")
	}
	tag, err := tx.Exec(
		ctx, invalidatePasswordResetCredentialSQL,
		credential.ID, credential.UserID, credential.Generation, invalidatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var userID string
	var generation int64
	var state TemporaryCredentialState
	err = tx.QueryRow(ctx, readPasswordResetCredentialStateSQL, credential.ID).Scan(
		&userID, &generation, &state,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if userID == credential.UserID && generation == credential.Generation &&
		state != TemporaryCredentialActive && state != TemporaryCredentialPendingDelivery &&
		state != TemporaryCredentialExpired {
		return nil
	}
	return errors.New("password reset credential could not be invalidated")
}

// VerifyPasswordResetAudit checks one immutable audit identity before a retry.
func (PostgresPasswordResetRepository) VerifyPasswordResetAudit(
	ctx context.Context,
	tx pgx.Tx,
	event audit.Event,
) (passwordResetAuditVerification, error) {
	if tx == nil {
		return passwordResetAuditVerification{}, errors.New("password reset transaction is required")
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return passwordResetAuditVerification{}, err
	}
	exact, present, err := passwordResetAuditEvidence(ctx, tx, event, metadata)
	return passwordResetAuditVerification{
		exact: exact, present: present, err: err,
	}, err
}

func passwordResetRecordExists(
	ctx context.Context,
	tx pgx.Tx,
	record idempotency.Record,
	snapshot []byte,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, verifyPasswordResetRecordSQL,
		record.ScopeHash, record.KeyLocator, record.KeyDigest, record.RequestDigest,
		record.DigestKeyVersion, record.ExecutionGeneration, snapshot,
	).Scan(&exists)
	return exists, err
}

func passwordResetProcessingRecordExists(
	ctx context.Context,
	tx pgx.Tx,
	record idempotency.Record,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, verifyPasswordResetProcessingRecordSQL,
		record.ScopeHash, record.KeyLocator, record.KeyDigest, record.RequestDigest,
		record.DigestKeyVersion, record.ExecutionGeneration,
	).Scan(&exists)
	return exists, err
}

func passwordResetCredentialEvidence(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) (bool, bool, error) {
	var exact, present bool
	err := tx.QueryRow(ctx, verifyPasswordResetCredentialSQL,
		credential.ID, credential.UserID, credential.Purpose, credential.Generation,
		credential.SecretDigest, credential.DeliveredToEmail,
		credential.IssuedAt, credential.ExpiresAt,
	).Scan(&exact, &present)
	return exact, present, err
}

func passwordResetAuditEvidence(
	ctx context.Context,
	tx pgx.Tx,
	event audit.Event,
	metadata []byte,
) (bool, bool, error) {
	var exact, present bool
	err := tx.QueryRow(ctx, verifyPasswordResetAuditSQL,
		event.ID, event.OccurredAt, event.ActorKind, event.ActorUserID,
		event.ActorSystem, event.ActorRole, event.Action, event.TargetType,
		event.TargetID, event.Outcome, event.ErrorCode, event.TraceID,
		event.SourceFingerprint, metadata,
	).Scan(&exact, &present)
	return exact, present, err
}

const lockPasswordResetUserSQL = `
	SELECT
		id, username, email, phone, permanent_password_digest, role,
		is_initial_admin, status, credential_mode, recovery_required,
		onboarding_privacy_acknowledged_version, onboarding_completed_at,
		retained_email, created_at, updated_at, deleted_at
	FROM users
	WHERE id = $1
	FOR UPDATE
`

const lockPasswordResetCredentialSQL = `
	SELECT
		id, user_id, purpose, generation, state, secret_digest,
		delivered_to_email, issued_at, expires_at, superseded_at,
		consumed_at, invalidated_at
	FROM temporary_credentials
	WHERE user_id = $1
	ORDER BY generation DESC
	LIMIT 1
	FOR UPDATE
`

const supersedePasswordResetCredentialSQL = `
	UPDATE temporary_credentials
	SET state = 'superseded', superseded_at = $4
	WHERE id = $1 AND user_id = $2 AND generation = $3
		AND state IN ('pending_delivery', 'active', 'expired')
`

const setPasswordResetCredentialModeSQL = `
	UPDATE users
	SET credential_mode = $2, updated_at = $3
	WHERE id = $1
		AND status IN ('active', 'suspended')
		AND deleted_at IS NULL
		AND recovery_required = false
`

const verifyPasswordResetRecordSQL = `
	SELECT EXISTS (
		SELECT 1
		FROM idempotency_records
		WHERE scope_hash = $1
			AND key_locator = $2
			AND key_digest = $3
			AND request_digest = $4
			AND digest_key_version = $5
			AND execution_generation = $6
			AND status = 'succeeded'
			AND result_code = 'password_reset'
			AND response_snapshot = $7::jsonb
	)
`

const verifyPasswordResetProcessingRecordSQL = `
	SELECT EXISTS (
		SELECT 1
		FROM idempotency_records
		WHERE scope_hash = $1
			AND key_locator = $2
			AND key_digest = $3
			AND request_digest = $4
			AND digest_key_version = $5
			AND execution_generation = $6
			AND status = 'processing'
	)
`

const verifyPasswordResetCredentialSQL = `
	SELECT
		EXISTS (
			SELECT 1
			FROM temporary_credentials
			WHERE id = $1
				AND user_id = $2
				AND purpose = $3
				AND generation = $4
				AND state = 'active'
				AND secret_digest = $5
				AND delivered_to_email = $6
				AND issued_at = $7
				AND expires_at = $8
		),
		EXISTS (SELECT 1 FROM temporary_credentials WHERE id = $1)
`

const verifyPasswordResetAuditSQL = `
	SELECT
		EXISTS (
			SELECT 1
			FROM audit_events
			WHERE id = $1
				AND occurred_at = $2
				AND actor_kind = $3
				AND actor_user_id IS NOT DISTINCT FROM $4
				AND actor_system IS NOT DISTINCT FROM $5
				AND actor_role IS NOT DISTINCT FROM $6
				AND action = $7
				AND target_type = $8
				AND target_id = $9
				AND outcome = $10
				AND error_code IS NOT DISTINCT FROM $11
				AND trace_id = $12
				AND source_fingerprint IS NOT DISTINCT FROM $13
				AND metadata = $14::jsonb
		),
		EXISTS (SELECT 1 FROM audit_events WHERE id = $1)
`

const invalidatePasswordResetCredentialSQL = `
	UPDATE temporary_credentials
	SET state = 'recovery_invalidated', invalidated_at = $4
	WHERE id = $1
		AND user_id = $2
		AND generation = $3
		AND state IN ('pending_delivery', 'active', 'expired')
`

const readPasswordResetCredentialStateSQL = `
	SELECT user_id, generation, state
	FROM temporary_credentials
	WHERE id = $1
`

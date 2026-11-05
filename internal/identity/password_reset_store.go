package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/idempotency"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrPasswordResetCandidateInvalid indicates an incomplete durable candidate.
	ErrPasswordResetCandidateInvalid = errors.New("password reset candidate is invalid")
	// ErrPasswordResetGateClosed indicates unresolved or ineligible reset state.
	ErrPasswordResetGateClosed = session.ErrPasswordResetGateClosed
	// ErrPasswordResetStateConflict indicates a guarded persistence write did not apply.
	ErrPasswordResetStateConflict = errors.New("password reset state conflict")
)

// PasswordResetCandidateAction classifies one durable candidate attempt.
type PasswordResetCandidateAction string

const (
	// PasswordResetCandidateCreated means all five authoritative objects were written.
	PasswordResetCandidateCreated PasswordResetCandidateAction = "created"
	// PasswordResetCandidateExisting means an exact same-key binding already exists.
	PasswordResetCandidateExisting PasswordResetCandidateAction = "existing"
	// PasswordResetCandidateIntegrityHold means bad durable evidence was held fail closed.
	PasswordResetCandidateIntegrityHold PasswordResetCandidateAction = "integrity_hold"
)

// PasswordResetCandidate contains all caller-prepared values for one atomic write.
type PasswordResetCandidate struct {
	Operation          PasswordResetOperation
	Credential         TemporaryCredential
	RecoveryJob        PasswordResetRecoveryJob
	IdempotencyRequest idempotency.BeginRequest
}

// PasswordResetCandidateResult is the authoritative candidate classification.
type PasswordResetCandidateResult struct {
	Action    PasswordResetCandidateAction
	Operation PasswordResetOperation
	Claim     idempotency.Record
}

// PasswordResetStore persists durable reset authority in caller-owned transactions.
type PasswordResetStore struct {
	Idempotency idempotency.Store
}

// CheckLockedUser enforces the durable authentication barrier for a locked User.
func (PasswordResetStore) CheckLockedUser(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) error {
	if tx == nil || strings.TrimSpace(userID) == "" {
		return errors.New("password reset Gate input is invalid")
	}
	var clear bool
	if err := tx.QueryRow(ctx, passwordResetGateSQL, userID).Scan(&clear); err != nil {
		return fmt.Errorf("check password reset Gate: %w", err)
	}
	if !clear {
		return ErrPasswordResetGateClosed
	}
	return nil
}

// BeginOrClassifyCandidate locks User first, then creates or classifies one candidate.
func (store PasswordResetStore) BeginOrClassifyCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
) (PasswordResetCandidateResult, error) {
	attempt, err := preparePasswordResetCandidateAttempt(ctx, tx, candidate)
	if err != nil {
		return PasswordResetCandidateResult{}, err
	}
	result, handled, err := store.classifyPasswordResetCandidate(ctx, tx, attempt)
	if err != nil || handled {
		return result, err
	}
	return store.createPasswordResetCandidate(ctx, tx, attempt)
}

type passwordResetCandidateAttempt struct {
	candidate  PasswordResetCandidate
	user       passwordResetLockedUser
	claim      idempotency.Record
	generation int64
}

func preparePasswordResetCandidateAttempt(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
) (passwordResetCandidateAttempt, error) {
	if tx == nil || !validPasswordResetCandidateShape(candidate) {
		return passwordResetCandidateAttempt{}, ErrPasswordResetCandidateInvalid
	}
	user, err := lockPasswordResetCandidateUser(ctx, tx, candidate.Operation.UserID)
	if err != nil {
		return passwordResetCandidateAttempt{}, err
	}
	generation := user.executionGeneration + 1
	claim, err := idempotency.NewPasswordResetClaim(
		candidate.IdempotencyRequest,
		generation,
	)
	if err != nil {
		return passwordResetCandidateAttempt{}, fmt.Errorf(
			"%w: idempotency identity",
			ErrPasswordResetCandidateInvalid,
		)
	}
	return passwordResetCandidateAttempt{
		candidate: candidate, user: user, claim: claim, generation: generation,
	}, nil
}

func (store PasswordResetStore) classifyPasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
) (PasswordResetCandidateResult, bool, error) {
	stored, found, err := store.Idempotency.ResolvePasswordResetClaim(
		ctx, tx, attempt.candidate.IdempotencyRequest,
	)
	if err != nil {
		return PasswordResetCandidateResult{}, false, err
	}
	if found {
		result, err := classifyStoredPasswordResetCandidate(ctx, tx, attempt, stored)
		return result, true, err
	}
	return classifyOrphanPasswordResetCandidate(ctx, tx, attempt)
}

func classifyStoredPasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
	stored idempotency.Record,
) (PasswordResetCandidateResult, error) {
	existing, err := readPasswordResetCandidateBinding(
		ctx, tx, attempt.candidate, stored,
	)
	if err != nil {
		return PasswordResetCandidateResult{}, err
	}
	if existing.exact && !attempt.user.integrityHold {
		return PasswordResetCandidateResult{
			Action: PasswordResetCandidateExisting, Operation: existing.operation,
			Claim: existing.claim,
		}, nil
	}
	holdID := existing.holdOperationID(attempt.candidate.Operation.UserID)
	return holdPasswordResetCandidate(ctx, tx, attempt, holdID, existing.operation, existing.claim)
}

func classifyOrphanPasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
) (PasswordResetCandidateResult, bool, error) {
	orphan, found, err := readPasswordResetOrphanOperation(
		ctx, tx, attempt.claim.ScopeHash, attempt.claim.KeyLocator,
	)
	if err != nil {
		return PasswordResetCandidateResult{}, false, err
	}
	if found {
		var holdID *string
		if orphan.UserID == attempt.candidate.Operation.UserID {
			holdID = &orphan.ID
		}
		result, err := holdPasswordResetCandidate(
			ctx, tx, attempt, holdID, orphan, idempotency.Record{},
		)
		return result, true, err
	}
	if attempt.user.integrityHold {
		return PasswordResetCandidateResult{
			Action: PasswordResetCandidateIntegrityHold,
		}, true, nil
	}
	return PasswordResetCandidateResult{}, false, nil
}

func holdPasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
	operationID *string,
	operation PasswordResetOperation,
	claim idempotency.Record,
) (PasswordResetCandidateResult, error) {
	if err := setPasswordResetIntegrityHold(
		ctx, tx, attempt.candidate.Operation.UserID, operationID,
		attempt.candidate.Operation.ID, attempt.candidate.IdempotencyRequest.Now,
	); err != nil {
		return PasswordResetCandidateResult{}, err
	}
	return PasswordResetCandidateResult{
		Action: PasswordResetCandidateIntegrityHold, Operation: operation, Claim: claim,
	}, nil
}

func (store PasswordResetStore) createPasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
) (PasswordResetCandidateResult, error) {
	result, handled, err := authorizeNewPasswordResetCandidate(ctx, tx, attempt)
	if err != nil || handled {
		return result, err
	}
	operation, credential, job, err := buildPasswordResetCandidate(attempt)
	if err != nil {
		return PasswordResetCandidateResult{}, err
	}
	claim, err := store.writePasswordResetCandidate(
		ctx, tx, attempt, operation, credential, job,
	)
	if err != nil {
		return PasswordResetCandidateResult{}, err
	}
	return PasswordResetCandidateResult{
		Action: PasswordResetCandidateCreated, Operation: operation, Claim: claim,
	}, nil
}

func authorizeNewPasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
) (PasswordResetCandidateResult, bool, error) {
	blocked, err := passwordResetUserHasUnresolvedOperation(
		ctx, tx, attempt.candidate.Operation.UserID,
	)
	if err != nil {
		return PasswordResetCandidateResult{}, false, err
	}
	if blocked {
		return PasswordResetCandidateResult{}, false, ErrPasswordResetGateClosed
	}
	if attempt.candidate.Operation.RequestKind != PasswordResetResend {
		return PasswordResetCandidateResult{}, false, nil
	}
	lineage, err := passwordResetResendLineageStatus(ctx, tx, attempt.candidate)
	if err != nil {
		return PasswordResetCandidateResult{}, false, err
	}
	if lineage.status == passwordResetResendLineageEligible {
		return PasswordResetCandidateResult{}, false, nil
	}
	if lineage.status == passwordResetResendLineageIneligible {
		return PasswordResetCandidateResult{}, false, ErrPasswordResetGateClosed
	}
	result, err := holdPasswordResetCandidate(
		ctx, tx, attempt, lineage.holdOperationID,
		PasswordResetOperation{}, idempotency.Record{},
	)
	return result, true, err
}

func buildPasswordResetCandidate(
	attempt passwordResetCandidateAttempt,
) (PasswordResetOperation, TemporaryCredential, PasswordResetRecoveryJob, error) {
	operation := attempt.candidate.Operation
	operation.ScopeHash = attempt.claim.ScopeHash
	operation.KeyLocator = attempt.claim.KeyLocator
	operation.KeyDigest = attempt.claim.KeyDigest
	operation.RequestDigest = attempt.claim.RequestDigest
	operation.DigestKeyVersion = attempt.claim.DigestKeyVersion
	operation.ExecutionGeneration = attempt.generation
	operation.BaseAuthGeneration = attempt.user.authGeneration
	operation.TargetAuthGeneration = attempt.user.authGeneration + 1
	credential := attempt.candidate.Credential
	credential.AuthGeneration = operation.TargetAuthGeneration
	credential.SourceOperationID = &operation.ID
	if !validPasswordResetCredential(operation, credential) ||
		!validPasswordResetRecoveryJob(operation, attempt.candidate.RecoveryJob) {
		return PasswordResetOperation{}, TemporaryCredential{},
			PasswordResetRecoveryJob{}, ErrPasswordResetCandidateInvalid
	}
	return operation, credential, attempt.candidate.RecoveryJob, nil
}

func validPasswordResetCredential(
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	return ValidateTemporaryCredential(credential) == nil &&
		credential.ID == operation.TemporaryCredentialID &&
		credential.UserID == operation.UserID &&
		credential.Purpose == TemporaryCredentialPasswordReset &&
		credential.State == TemporaryCredentialPendingDelivery
}

func validPasswordResetRecoveryJob(
	operation PasswordResetOperation,
	job PasswordResetRecoveryJob,
) bool {
	return job.OperationID == operation.ID &&
		job.Status == PasswordResetRecoveryJobReady &&
		job.AttemptCount == 0 &&
		!job.NextAttemptAt.IsZero() &&
		!job.CreatedAt.IsZero() &&
		!job.UpdatedAt.IsZero() &&
		job.LeaseOwner == nil &&
		job.LeaseExpiresAt == nil &&
		job.CompletedAt == nil
}

func (store PasswordResetStore) writePasswordResetCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	attempt passwordResetCandidateAttempt,
	operation PasswordResetOperation,
	credential TemporaryCredential,
	job PasswordResetRecoveryJob,
) (idempotency.Record, error) {
	if err := advancePasswordResetExecutionGeneration(
		ctx, tx, operation.UserID, attempt.user.executionGeneration,
		attempt.generation, operation.ID, attempt.candidate.IdempotencyRequest.Now,
	); err != nil {
		return idempotency.Record{}, err
	}
	if err := insertPasswordResetOperation(ctx, tx, operation); err != nil {
		return idempotency.Record{}, err
	}
	if err := insertPasswordResetCredential(ctx, tx, credential); err != nil {
		return idempotency.Record{}, err
	}
	if err := insertPasswordResetRecoveryJob(ctx, tx, job); err != nil {
		return idempotency.Record{}, err
	}
	return store.Idempotency.InsertPasswordResetClaim(
		ctx, tx, attempt.candidate.IdempotencyRequest, attempt.generation,
		operation.ID, operation.UserID,
	)
}

type passwordResetLockedUser struct {
	authGeneration      int64
	executionGeneration int64
	integrityHold       bool
}

func lockPasswordResetCandidateUser(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
) (passwordResetLockedUser, error) {
	var user passwordResetLockedUser
	var lockedID string
	err := tx.QueryRow(ctx, lockPasswordResetCandidateUserSQL, userID).Scan(
		&lockedID,
		&user.authGeneration,
		&user.executionGeneration,
		&user.integrityHold,
	)
	if err != nil {
		return passwordResetLockedUser{}, fmt.Errorf("lock password reset user: %w", err)
	}
	if lockedID != userID || user.authGeneration < 0 || user.executionGeneration < 0 {
		return passwordResetLockedUser{}, ErrPasswordResetStateConflict
	}
	return user, nil
}

type passwordResetExistingBinding struct {
	operation     PasswordResetOperation
	operationUser *string
	exact         bool
	claim         idempotency.Record
}

func (binding passwordResetExistingBinding) holdOperationID(userID string) *string {
	if binding.operationUser == nil || *binding.operationUser != userID ||
		strings.TrimSpace(binding.operation.ID) == "" {
		return nil
	}
	value := binding.operation.ID
	return &value
}

func readPasswordResetCandidateBinding(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
	expected idempotency.Record,
) (passwordResetExistingBinding, error) {
	row, err := scanPasswordResetCandidateBinding(ctx, tx, candidate, expected)
	if errors.Is(err, pgx.ErrNoRows) {
		return passwordResetExistingBinding{}, ErrPasswordResetStateConflict
	}
	if err != nil {
		return passwordResetExistingBinding{}, fmt.Errorf(
			"read password reset candidate binding: %w",
			err,
		)
	}
	return row.toBinding(), nil
}

type passwordResetCandidateBindingRow struct {
	binding        passwordResetExistingBinding
	operationID    *string
	operationUser  *string
	requestKind    *PasswordResetRequestKind
	state          *PasswordResetOperationState
	settlement     *PasswordResetSettlementState
	delivery       *PasswordResetDeliveryState
	targetAuth     int64
	settledAuth    *int64
	successorAuth  *int64
	settledAt      *time.Time
	terminalAt     *time.Time
	execution      int64
	relationExact  bool
	authorityValid bool
}

func scanPasswordResetCandidateBinding(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
	expected idempotency.Record,
) (passwordResetCandidateBindingRow, error) {
	var row passwordResetCandidateBindingRow
	err := tx.QueryRow(ctx, readPasswordResetCandidateBindingSQL,
		expected.ScopeHash, expected.KeyLocator, expected.KeyDigest,
		expected.DigestKeyVersion, expected.RequestDigest,
		candidate.Operation.UserID, candidate.Operation.RequestKind,
		candidate.Operation.PredecessorOperationID,
		candidate.Operation.ReplacedCredentialID,
	).Scan(
		&row.binding.claim.ScopeHash, &row.binding.claim.KeyLocator,
		&row.binding.claim.KeyDigest, &row.binding.claim.DigestKeyVersion,
		&row.binding.claim.RequestDigest, &row.binding.claim.ExecutionGeneration,
		&row.operationID, &row.operationUser, &row.requestKind, &row.state,
		&row.settlement, &row.delivery, &row.targetAuth, &row.settledAuth,
		&row.successorAuth, &row.settledAt, &row.terminalAt, &row.execution,
		&row.relationExact, &row.authorityValid,
	)
	return row, err
}

func (row passwordResetCandidateBindingRow) toBinding() passwordResetExistingBinding {
	binding := row.binding
	binding.operationUser = row.operationUser
	if row.operationID != nil {
		binding.operation.ID = *row.operationID
	}
	if row.operationUser != nil {
		binding.operation.UserID = *row.operationUser
	}
	if row.requestKind != nil {
		binding.operation.RequestKind = *row.requestKind
	}
	if row.state != nil {
		binding.operation.State = *row.state
	}
	if row.settlement != nil {
		binding.operation.SettlementState = *row.settlement
	}
	if row.delivery != nil {
		binding.operation.DeliveryState = *row.delivery
	}
	binding.operation.TargetAuthGeneration = row.targetAuth
	binding.operation.SettledAuthGeneration = row.settledAuth
	binding.operation.SuccessorAuthGeneration = row.successorAuth
	binding.operation.SettledAt = row.settledAt
	binding.operation.TerminalAt = row.terminalAt
	binding.operation.ExecutionGeneration = row.execution
	binding.exact = row.relationExact && row.authorityValid &&
		passwordResetOperationAuthorityValid(binding.operation)
	return binding
}

func passwordResetOperationAuthorityValid(operation PasswordResetOperation) bool {
	if !passwordResetSettlementEvidenceValid(operation) {
		return false
	}
	terminal := operation.TerminalAt != nil
	switch operation.State {
	case PasswordResetOperationPrepared:
		return !terminal && operation.DeliveryState == PasswordResetDeliveryNotStarted &&
			operation.SettlementState == PasswordResetSettlementPending
	case PasswordResetOperationDispatching:
		return !terminal && operation.DeliveryState == PasswordResetDeliveryDispatching &&
			operation.SettlementState == PasswordResetSettlementPending
	case PasswordResetOperationCommitUnknown:
		return !terminal &&
			(operation.DeliveryState == PasswordResetDeliveryDispatching ||
				operation.DeliveryState == PasswordResetDeliveryUnknown) &&
			operation.SettlementState == PasswordResetSettlementPending
	case PasswordResetOperationRecoveryRequired, PasswordResetOperationConverging:
		return !terminal && operation.DeliveryState == PasswordResetDeliveryAccepted &&
			operation.SettlementState == PasswordResetSettlementPending
	case PasswordResetOperationSucceeded:
		return terminal && operation.DeliveryState == PasswordResetDeliveryAccepted &&
			(operation.SettlementState == PasswordResetSettlementTemporaryActive ||
				operation.SettlementState == PasswordResetSettlementPermanentSuccessor)
	case PasswordResetOperationInvalidated:
		return terminal && passwordResetInvalidatedAuthorityValid(operation)
	case PasswordResetOperationFailed:
		return terminal && operation.DeliveryState == PasswordResetDeliveryRejected &&
			operation.SettlementState == PasswordResetSettlementInvalidated
	default:
		return false
	}
}

func passwordResetSettlementEvidenceValid(operation PasswordResetOperation) bool {
	switch operation.SettlementState {
	case PasswordResetSettlementPending:
		return operation.SettledAuthGeneration == nil &&
			operation.SuccessorAuthGeneration == nil && operation.SettledAt == nil
	case PasswordResetSettlementTemporaryActive:
		return operation.SettledAuthGeneration != nil &&
			*operation.SettledAuthGeneration == operation.TargetAuthGeneration &&
			operation.SuccessorAuthGeneration == nil && operation.SettledAt != nil
	case PasswordResetSettlementPermanentSuccessor:
		return operation.SettledAuthGeneration != nil &&
			*operation.SettledAuthGeneration == operation.TargetAuthGeneration &&
			operation.SuccessorAuthGeneration != nil &&
			*operation.SuccessorAuthGeneration == operation.TargetAuthGeneration+1 &&
			operation.SettledAt != nil
	case PasswordResetSettlementInvalidated, PasswordResetSettlementIntegrityHold:
		return operation.SettledAuthGeneration == nil &&
			operation.SuccessorAuthGeneration == nil && operation.SettledAt != nil
	default:
		return false
	}
}

func passwordResetInvalidatedAuthorityValid(operation PasswordResetOperation) bool {
	deliveryValid := operation.DeliveryState == PasswordResetDeliveryNotStarted ||
		operation.DeliveryState == PasswordResetDeliveryDispatching ||
		operation.DeliveryState == PasswordResetDeliveryUnknown
	settlementValid := operation.SettlementState == PasswordResetSettlementInvalidated ||
		operation.SettlementState == PasswordResetSettlementIntegrityHold
	return deliveryValid && settlementValid
}

func readPasswordResetOrphanOperation(
	ctx context.Context,
	tx idempotency.Transaction,
	scopeHash []byte,
	keyLocator []byte,
) (PasswordResetOperation, bool, error) {
	var operation PasswordResetOperation
	err := tx.QueryRow(
		ctx,
		readPasswordResetOrphanOperationSQL,
		scopeHash,
		keyLocator,
	).Scan(&operation.ID, &operation.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PasswordResetOperation{}, false, nil
	}
	if err != nil {
		return PasswordResetOperation{}, false, fmt.Errorf(
			"read orphan password reset operation: %w",
			err,
		)
	}
	return operation, true, nil
}

func setPasswordResetIntegrityHold(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
	operationID *string,
	auditID string,
	now time.Time,
) error {
	tag, err := tx.Exec(
		ctx,
		setPasswordResetIntegrityHoldSQL,
		userID,
		operationID,
		"password_reset_binding_mismatch",
		now,
		auditID,
	)
	if err != nil {
		return fmt.Errorf("set password reset integrity hold: %w", err)
	}
	return requireOnePasswordResetRow(tag)
}

func passwordResetUserHasUnresolvedOperation(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
) (bool, error) {
	var blocked bool
	if err := tx.QueryRow(
		ctx,
		passwordResetUserHasUnresolvedOperationSQL,
		userID,
	).Scan(&blocked); err != nil {
		return false, fmt.Errorf("check unresolved password reset operation: %w", err)
	}
	return blocked, nil
}

type passwordResetResendLineageClassification string

const (
	passwordResetResendLineageEligible   passwordResetResendLineageClassification = "eligible"
	passwordResetResendLineageIneligible passwordResetResendLineageClassification = "ineligible"
	passwordResetResendLineageDrift      passwordResetResendLineageClassification = "integrity_drift"
)

type passwordResetResendLineageResult struct {
	status          passwordResetResendLineageClassification
	holdOperationID *string
}

type passwordResetResendLineageEvidence struct {
	userID, predecessorID, predecessorUserID, predecessorCredentialID *string
	credentialMode                                                    CredentialMode
	authGeneration                                                    int64
	permanentDigestPresent                                            bool
	permanentGeneration                                               *int64
	permanentSourceOperationID, permanentSourceCredentialID           *string
	permanentOperationID, permanentOperationUserID                    *string
	permanentOperationCredentialID                                    *string
	permanentOperationState                                           *PasswordResetOperationState
	permanentOperationSettlement                                      *PasswordResetSettlementState
	permanentOperationDelivery                                        *PasswordResetDeliveryState
	permanentOperationBaseAuth, permanentOperationTargetAuth          *int64
	permanentOperationSettledAuth, permanentOperationSuccessorAuth    *int64
	permanentOperationSettledAt, permanentOperationTerminalAt         *time.Time
	permanentOperationDeliveryAttemptID                               *string
	permanentOperationDeliveryAttemptedAt                             *time.Time
	permanentOperationDeliveryResultRecordedAt                        *time.Time
	permanentCredentialID, permanentCredentialUserID                  *string
	permanentCredentialPurpose                                        *TemporaryCredentialPurpose
	permanentCredentialGeneration, permanentCredentialAuth            *int64
	permanentCredentialSourceOperationID                              *string
	permanentCredentialState                                          *TemporaryCredentialState
	predecessorState                                                  *PasswordResetOperationState
	predecessorSettlement                                             *PasswordResetSettlementState
	predecessorDelivery                                               *PasswordResetDeliveryState
	targetAuth, settledAuth, successorAuth                            *int64
	settledAt, terminalAt                                             *time.Time
	deliveryAttemptID                                                 *string
	deliveryAttemptedAt, deliveryResultRecordedAt                     *time.Time
	replacedID, replacedUserID, replacedSourceOperationID             *string
	replacedPurpose                                                   *TemporaryCredentialPurpose
	replacedAuth                                                      *int64
	replacedState                                                     *TemporaryCredentialState
	currentOperationID, currentOperationUserID                        *string
	currentOperationCredentialID                                      *string
	currentRequestKind                                                *PasswordResetRequestKind
	currentPredecessorID, currentReplacedID                           *string
	currentState                                                      *PasswordResetOperationState
	currentSettlement                                                 *PasswordResetSettlementState
	currentDelivery                                                   *PasswordResetDeliveryState
	currentBaseAuth, currentTargetAuth                                *int64
	currentSettledAuth, currentSuccessorAuth                          *int64
	currentSettledAt, currentTerminalAt                               *time.Time
	currentDeliveryAttemptID                                          *string
	currentDeliveryAttemptedAt, currentDeliveryResultRecordedAt       *time.Time
	currentCredentialID, currentCredentialUserID                      *string
	currentCredentialPurpose                                          *TemporaryCredentialPurpose
	currentCredentialGeneration                                       *int64
	currentCredentialGenerationOrdinal                                *int64
	currentCredentialAuth                                             *int64
	currentCredentialSourceOperationID                                *string
	currentCredentialState                                            *TemporaryCredentialState
	currentLineageValid, currentLineageContainsPredecessor            bool
	currentLineageRootBaseAuth                                        *int64
}

func passwordResetResendLineageStatus(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
) (passwordResetResendLineageResult, error) {
	var evidence passwordResetResendLineageEvidence
	row := tx.QueryRow(
		ctx,
		passwordResetResendLineageStatusSQL,
		candidate.Operation.UserID,
		candidate.Operation.PredecessorOperationID,
		candidate.Operation.ReplacedCredentialID,
	)
	if err := scanPasswordResetResendLineageEvidence(row, &evidence); err != nil {
		return passwordResetResendLineageResult{},
			fmt.Errorf("check password reset resend lineage: %w", err)
	}
	return classifyPasswordResetResendLineage(evidence), nil
}

func scanPasswordResetResendLineageEvidence(
	row pgx.Row,
	evidence *passwordResetResendLineageEvidence,
) error {
	destinations := passwordResetPermanentEvidenceDestinations(evidence)
	destinations = append(destinations, passwordResetHistoricalEvidenceDestinations(evidence)...)
	destinations = append(destinations, passwordResetCurrentEvidenceDestinations(evidence)...)
	return row.Scan(destinations...)
}

func passwordResetPermanentEvidenceDestinations(
	evidence *passwordResetResendLineageEvidence,
) []any {
	return []any{
		&evidence.userID, &evidence.credentialMode, &evidence.authGeneration,
		&evidence.permanentDigestPresent,
		&evidence.permanentGeneration, &evidence.permanentSourceOperationID,
		&evidence.permanentSourceCredentialID,
		&evidence.permanentOperationID, &evidence.permanentOperationUserID,
		&evidence.permanentOperationCredentialID,
		&evidence.permanentOperationState, &evidence.permanentOperationSettlement,
		&evidence.permanentOperationDelivery,
		&evidence.permanentOperationBaseAuth, &evidence.permanentOperationTargetAuth,
		&evidence.permanentOperationSettledAuth,
		&evidence.permanentOperationSuccessorAuth,
		&evidence.permanentOperationSettledAt,
		&evidence.permanentOperationTerminalAt,
		&evidence.permanentOperationDeliveryAttemptID,
		&evidence.permanentOperationDeliveryAttemptedAt,
		&evidence.permanentOperationDeliveryResultRecordedAt,
		&evidence.permanentCredentialID, &evidence.permanentCredentialUserID,
		&evidence.permanentCredentialPurpose,
		&evidence.permanentCredentialGeneration,
		&evidence.permanentCredentialAuth,
		&evidence.permanentCredentialSourceOperationID,
		&evidence.permanentCredentialState,
	}
}

func passwordResetHistoricalEvidenceDestinations(
	evidence *passwordResetResendLineageEvidence,
) []any {
	return []any{
		&evidence.predecessorID, &evidence.predecessorUserID,
		&evidence.predecessorCredentialID, &evidence.predecessorState,
		&evidence.predecessorSettlement, &evidence.predecessorDelivery,
		&evidence.targetAuth, &evidence.settledAuth, &evidence.successorAuth,
		&evidence.settledAt, &evidence.terminalAt, &evidence.deliveryAttemptID,
		&evidence.deliveryAttemptedAt, &evidence.deliveryResultRecordedAt,
		&evidence.replacedID, &evidence.replacedUserID, &evidence.replacedPurpose,
		&evidence.replacedAuth, &evidence.replacedSourceOperationID,
		&evidence.replacedState,
	}
}

func passwordResetCurrentEvidenceDestinations(
	evidence *passwordResetResendLineageEvidence,
) []any {
	return []any{
		&evidence.currentOperationID, &evidence.currentOperationUserID,
		&evidence.currentOperationCredentialID, &evidence.currentRequestKind,
		&evidence.currentPredecessorID, &evidence.currentReplacedID,
		&evidence.currentState, &evidence.currentSettlement, &evidence.currentDelivery,
		&evidence.currentBaseAuth, &evidence.currentTargetAuth,
		&evidence.currentSettledAuth, &evidence.currentSuccessorAuth,
		&evidence.currentSettledAt, &evidence.currentTerminalAt,
		&evidence.currentDeliveryAttemptID, &evidence.currentDeliveryAttemptedAt,
		&evidence.currentDeliveryResultRecordedAt,
		&evidence.currentCredentialID, &evidence.currentCredentialUserID,
		&evidence.currentCredentialPurpose,
		&evidence.currentCredentialGeneration,
		&evidence.currentCredentialGenerationOrdinal,
		&evidence.currentCredentialAuth,
		&evidence.currentCredentialSourceOperationID,
		&evidence.currentCredentialState,
		&evidence.currentLineageValid,
		&evidence.currentLineageContainsPredecessor,
		&evidence.currentLineageRootBaseAuth,
	}
}

func classifyPasswordResetResendLineage(
	evidence passwordResetResendLineageEvidence,
) passwordResetResendLineageResult {
	result := passwordResetResendLineageResult{
		status:          passwordResetResendLineageDrift,
		holdOperationID: evidence.provenHoldOperationID(),
	}
	operation, operationValid := evidence.operation()
	credential, credentialValid := evidence.credential()
	if !operationValid || !credentialValid ||
		evidence.userID == nil || operation.UserID != *evidence.userID ||
		!passwordResetOperationAuthorityValid(operation) ||
		!passwordResetDeliveryEvidenceValid(operation) ||
		!passwordResetResendCredentialExact(operation, credential) {
		return result
	}
	switch {
	case passwordResetResendCurrentTemporary(evidence, operation, credential):
		result.status = passwordResetResendLineageEligible
		result.holdOperationID = nil
	case passwordResetResendCurrentPermanent(evidence, operation, credential),
		passwordResetResendPermanentLaterReset(evidence, operation, credential),
		passwordResetResendSupersededStale(evidence, operation, credential),
		passwordResetResendRecoveryInvalidatedStale(evidence, operation, credential):
		result.status = passwordResetResendLineageIneligible
		result.holdOperationID = nil
	}
	return result
}

func (evidence passwordResetResendLineageEvidence) provenHoldOperationID() *string {
	if evidence.userID == nil || evidence.predecessorID == nil ||
		evidence.predecessorUserID == nil ||
		*evidence.predecessorUserID != *evidence.userID {
		return nil
	}
	value := *evidence.predecessorID
	return &value
}

func (evidence passwordResetResendLineageEvidence) operation() (
	PasswordResetOperation,
	bool,
) {
	if evidence.predecessorID == nil || evidence.predecessorUserID == nil ||
		evidence.predecessorCredentialID == nil || evidence.predecessorState == nil ||
		evidence.predecessorSettlement == nil || evidence.predecessorDelivery == nil ||
		evidence.targetAuth == nil {
		return PasswordResetOperation{}, false
	}
	return PasswordResetOperation{
		ID: *evidence.predecessorID, UserID: *evidence.predecessorUserID,
		TemporaryCredentialID:   *evidence.predecessorCredentialID,
		State:                   *evidence.predecessorState,
		SettlementState:         *evidence.predecessorSettlement,
		DeliveryState:           *evidence.predecessorDelivery,
		TargetAuthGeneration:    *evidence.targetAuth,
		SettledAuthGeneration:   evidence.settledAuth,
		SuccessorAuthGeneration: evidence.successorAuth,
		SettledAt:               evidence.settledAt, TerminalAt: evidence.terminalAt,
		DeliveryAttemptID:        evidence.deliveryAttemptID,
		DeliveryAttemptedAt:      evidence.deliveryAttemptedAt,
		DeliveryResultRecordedAt: evidence.deliveryResultRecordedAt,
	}, true
}

func (evidence passwordResetResendLineageEvidence) credential() (
	TemporaryCredential,
	bool,
) {
	if evidence.replacedID == nil || evidence.replacedUserID == nil ||
		evidence.replacedPurpose == nil || evidence.replacedAuth == nil ||
		evidence.replacedState == nil {
		return TemporaryCredential{}, false
	}
	return TemporaryCredential{
		ID: *evidence.replacedID, UserID: *evidence.replacedUserID,
		Purpose: *evidence.replacedPurpose, AuthGeneration: *evidence.replacedAuth,
		SourceOperationID: evidence.replacedSourceOperationID,
		State:             *evidence.replacedState,
	}, true
}

func (evidence passwordResetResendLineageEvidence) currentOperation() (
	PasswordResetOperation,
	bool,
) {
	if evidence.currentOperationID == nil || evidence.currentOperationUserID == nil ||
		evidence.currentOperationCredentialID == nil ||
		evidence.currentRequestKind == nil || evidence.currentState == nil ||
		evidence.currentSettlement == nil || evidence.currentDelivery == nil ||
		evidence.currentBaseAuth == nil || evidence.currentTargetAuth == nil {
		return PasswordResetOperation{}, false
	}
	return PasswordResetOperation{
		ID: *evidence.currentOperationID, UserID: *evidence.currentOperationUserID,
		TemporaryCredentialID:  *evidence.currentOperationCredentialID,
		RequestKind:            *evidence.currentRequestKind,
		PredecessorOperationID: evidence.currentPredecessorID,
		ReplacedCredentialID:   evidence.currentReplacedID,
		BaseAuthGeneration:     *evidence.currentBaseAuth,
		TargetAuthGeneration:   *evidence.currentTargetAuth,
		State:                  *evidence.currentState, SettlementState: *evidence.currentSettlement,
		DeliveryState:           *evidence.currentDelivery,
		SettledAuthGeneration:   evidence.currentSettledAuth,
		SuccessorAuthGeneration: evidence.currentSuccessorAuth,
		SettledAt:               evidence.currentSettledAt, TerminalAt: evidence.currentTerminalAt,
		DeliveryAttemptID:        evidence.currentDeliveryAttemptID,
		DeliveryAttemptedAt:      evidence.currentDeliveryAttemptedAt,
		DeliveryResultRecordedAt: evidence.currentDeliveryResultRecordedAt,
	}, true
}

func (evidence passwordResetResendLineageEvidence) currentCredential() (
	TemporaryCredential,
	bool,
) {
	if evidence.currentCredentialID == nil || evidence.currentCredentialUserID == nil ||
		evidence.currentCredentialPurpose == nil ||
		evidence.currentCredentialGeneration == nil ||
		evidence.currentCredentialAuth == nil ||
		evidence.currentCredentialState == nil {
		return TemporaryCredential{}, false
	}
	return TemporaryCredential{
		ID: *evidence.currentCredentialID, UserID: *evidence.currentCredentialUserID,
		Purpose:           *evidence.currentCredentialPurpose,
		Generation:        *evidence.currentCredentialGeneration,
		AuthGeneration:    *evidence.currentCredentialAuth,
		SourceOperationID: evidence.currentCredentialSourceOperationID,
		State:             *evidence.currentCredentialState,
	}, true
}

func (evidence passwordResetResendLineageEvidence) permanentOperation() (
	PasswordResetOperation,
	bool,
) {
	if evidence.permanentOperationID == nil ||
		evidence.permanentOperationUserID == nil ||
		evidence.permanentOperationCredentialID == nil ||
		evidence.permanentOperationState == nil ||
		evidence.permanentOperationSettlement == nil ||
		evidence.permanentOperationDelivery == nil ||
		evidence.permanentOperationBaseAuth == nil ||
		evidence.permanentOperationTargetAuth == nil {
		return PasswordResetOperation{}, false
	}
	return PasswordResetOperation{
		ID: *evidence.permanentOperationID, UserID: *evidence.permanentOperationUserID,
		TemporaryCredentialID:    *evidence.permanentOperationCredentialID,
		BaseAuthGeneration:       *evidence.permanentOperationBaseAuth,
		TargetAuthGeneration:     *evidence.permanentOperationTargetAuth,
		State:                    *evidence.permanentOperationState,
		SettlementState:          *evidence.permanentOperationSettlement,
		DeliveryState:            *evidence.permanentOperationDelivery,
		SettledAuthGeneration:    evidence.permanentOperationSettledAuth,
		SuccessorAuthGeneration:  evidence.permanentOperationSuccessorAuth,
		SettledAt:                evidence.permanentOperationSettledAt,
		TerminalAt:               evidence.permanentOperationTerminalAt,
		DeliveryAttemptID:        evidence.permanentOperationDeliveryAttemptID,
		DeliveryAttemptedAt:      evidence.permanentOperationDeliveryAttemptedAt,
		DeliveryResultRecordedAt: evidence.permanentOperationDeliveryResultRecordedAt,
	}, true
}

func (evidence passwordResetResendLineageEvidence) permanentCredential() (
	TemporaryCredential,
	bool,
) {
	if evidence.permanentCredentialID == nil ||
		evidence.permanentCredentialUserID == nil ||
		evidence.permanentCredentialPurpose == nil ||
		evidence.permanentCredentialGeneration == nil ||
		evidence.permanentCredentialAuth == nil ||
		evidence.permanentCredentialState == nil {
		return TemporaryCredential{}, false
	}
	return TemporaryCredential{
		ID: *evidence.permanentCredentialID, UserID: *evidence.permanentCredentialUserID,
		Purpose:           *evidence.permanentCredentialPurpose,
		Generation:        *evidence.permanentCredentialGeneration,
		AuthGeneration:    *evidence.permanentCredentialAuth,
		SourceOperationID: evidence.permanentCredentialSourceOperationID,
		State:             *evidence.permanentCredentialState,
	}, true
}

func (evidence passwordResetResendLineageEvidence) currentTemporaryCapability() (
	PasswordResetOperation,
	TemporaryCredential,
	bool,
) {
	operation, operationValid := evidence.currentOperation()
	credential, credentialValid := evidence.currentCredential()
	valid := operationValid && credentialValid && evidence.currentLineageValid &&
		evidence.userID != nil && operation.UserID == *evidence.userID &&
		passwordResetOperationAuthorityValid(operation) &&
		passwordResetDeliveryEvidenceValid(operation) &&
		operation.SettlementState == PasswordResetSettlementTemporaryActive &&
		passwordResetResendCredentialExact(operation, credential) &&
		credential.State == TemporaryCredentialActive &&
		evidence.currentCredentialGenerationOrdinal != nil &&
		credential.Generation == *evidence.currentCredentialGenerationOrdinal &&
		evidence.authGeneration == operation.TargetAuthGeneration
	return operation, credential, valid
}

func (evidence passwordResetResendLineageEvidence) permanentRootAuthority() (
	PasswordResetOperation,
	TemporaryCredential,
	bool,
) {
	operation, operationValid := evidence.permanentOperation()
	credential, credentialValid := evidence.permanentCredential()
	valid := operationValid && credentialValid &&
		evidence.userID != nil && operation.UserID == *evidence.userID &&
		evidence.permanentDigestPresent && evidence.permanentGeneration != nil &&
		evidence.permanentSourceOperationID != nil &&
		*evidence.permanentSourceOperationID == operation.ID &&
		evidence.permanentSourceCredentialID != nil &&
		*evidence.permanentSourceCredentialID == credential.ID &&
		passwordResetOperationAuthorityValid(operation) &&
		passwordResetDeliveryEvidenceValid(operation) &&
		operation.SettlementState == PasswordResetSettlementPermanentSuccessor &&
		passwordResetResendCredentialExact(operation, credential) &&
		credential.Generation >= 1 &&
		credential.State == TemporaryCredentialConsumed &&
		operation.SuccessorAuthGeneration != nil &&
		*evidence.permanentGeneration >= *operation.SuccessorAuthGeneration
	return operation, credential, valid
}

func passwordResetDeliveryEvidenceValid(operation PasswordResetOperation) bool {
	switch operation.DeliveryState {
	case PasswordResetDeliveryNotStarted:
		return operation.DeliveryAttemptID == nil &&
			operation.DeliveryAttemptedAt == nil &&
			operation.DeliveryResultRecordedAt == nil
	case PasswordResetDeliveryDispatching:
		return operation.DeliveryAttemptID != nil &&
			operation.DeliveryAttemptedAt != nil &&
			operation.DeliveryResultRecordedAt == nil
	case PasswordResetDeliveryAccepted, PasswordResetDeliveryRejected,
		PasswordResetDeliveryUnknown:
		return operation.DeliveryAttemptID != nil &&
			operation.DeliveryAttemptedAt != nil &&
			operation.DeliveryResultRecordedAt != nil
	default:
		return false
	}
}

func passwordResetResendCredentialExact(
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	return operation.UserID == credential.UserID &&
		operation.TemporaryCredentialID == credential.ID &&
		credential.Purpose == TemporaryCredentialPasswordReset &&
		credential.AuthGeneration == operation.TargetAuthGeneration &&
		credential.SourceOperationID != nil &&
		*credential.SourceOperationID == operation.ID
}

func passwordResetResendCurrentTemporary(
	evidence passwordResetResendLineageEvidence,
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	currentOperation, currentCredential, currentValid :=
		evidence.currentTemporaryCapability()
	return operation.SettlementState == PasswordResetSettlementTemporaryActive &&
		credential.State == TemporaryCredentialActive &&
		evidence.credentialMode == CredentialModeTemporary &&
		evidence.authGeneration == operation.TargetAuthGeneration &&
		currentValid && currentOperation.ID == operation.ID &&
		currentCredential.ID == credential.ID
}

func passwordResetResendCurrentPermanent(
	evidence passwordResetResendLineageEvidence,
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	currentOperation, _, currentValid := evidence.permanentRootAuthority()
	return operation.SettlementState == PasswordResetSettlementPermanentSuccessor &&
		credential.State == TemporaryCredentialConsumed &&
		evidence.credentialMode == CredentialModePermanent &&
		currentValid && evidence.permanentGeneration != nil &&
		evidence.authGeneration == *evidence.permanentGeneration &&
		passwordResetPermanentHistoryCompatible(operation, currentOperation)
}

func passwordResetResendPermanentLaterReset(
	evidence passwordResetResendLineageEvidence,
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	currentOperation, _, currentValid := evidence.currentTemporaryCapability()
	permanentOperation, _, permanentValid := evidence.permanentRootAuthority()
	return operation.SettlementState == PasswordResetSettlementPermanentSuccessor &&
		credential.State == TemporaryCredentialConsumed &&
		evidence.credentialMode == CredentialModeTemporary &&
		currentValid && permanentValid &&
		passwordResetPermanentHistoryCompatible(operation, permanentOperation) &&
		evidence.currentLineageRootBaseAuth != nil &&
		evidence.permanentGeneration != nil &&
		*evidence.currentLineageRootBaseAuth == *evidence.permanentGeneration &&
		currentOperation.TargetAuthGeneration == evidence.authGeneration
}

func passwordResetPermanentHistoryCompatible(
	history PasswordResetOperation,
	current PasswordResetOperation,
) bool {
	if history.SuccessorAuthGeneration == nil {
		return false
	}
	return current.ID == history.ID ||
		(current.BaseAuthGeneration >= *history.SuccessorAuthGeneration &&
			current.TargetAuthGeneration > history.TargetAuthGeneration)
}

func passwordResetResendSupersededStale(
	evidence passwordResetResendLineageEvidence,
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	currentOperation, _, currentValid := evidence.currentTemporaryCapability()
	return operation.SettlementState == PasswordResetSettlementTemporaryActive &&
		credential.State == TemporaryCredentialSuperseded &&
		evidence.credentialMode == CredentialModeTemporary &&
		currentValid && evidence.currentLineageContainsPredecessor &&
		currentOperation.TargetAuthGeneration > operation.TargetAuthGeneration
}

func passwordResetResendRecoveryInvalidatedStale(
	evidence passwordResetResendLineageEvidence,
	operation PasswordResetOperation,
	credential TemporaryCredential,
) bool {
	return operation.SettlementState == PasswordResetSettlementTemporaryActive &&
		credential.State == TemporaryCredentialRecoveryInvalidated &&
		evidence.credentialMode == CredentialModeRecoveryLocked &&
		evidence.authGeneration == operation.TargetAuthGeneration
}

func advancePasswordResetExecutionGeneration(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
	current int64,
	next int64,
	operationID string,
	now time.Time,
) error {
	tag, err := tx.Exec(
		ctx,
		advancePasswordResetExecutionGenerationSQL,
		userID,
		current,
		next,
		operationID,
		now,
	)
	if err != nil {
		return fmt.Errorf("advance password reset execution generation: %w", err)
	}
	return requireOnePasswordResetRow(tag)
}

func insertPasswordResetOperation(
	ctx context.Context,
	tx idempotency.Transaction,
	operation PasswordResetOperation,
) error {
	if operation.Recovery != nil {
		return insertRecoveryResetOperation(ctx, tx, operation)
	}
	tag, err := tx.Exec(
		ctx,
		insertPasswordResetOperationSQL,
		operation.ID,
		operation.UserID,
		operation.TemporaryCredentialID,
		operation.ScopeHash,
		operation.KeyLocator,
		operation.KeyDigest,
		operation.DigestKeyVersion,
		operation.RequestDigest,
		operation.ExecutionGeneration,
		operation.RequestKind,
		operation.PredecessorOperationID,
		operation.ReplacedCredentialID,
		operation.BaseAuthGeneration,
		operation.TargetAuthGeneration,
		operation.SettlementState,
		operation.DeliveryState,
		operation.State,
		operation.CreatedAt,
		operation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert password reset operation: %w", err)
	}
	return requireOnePasswordResetRow(tag)
}

func insertPasswordResetCredential(
	ctx context.Context,
	tx idempotency.Transaction,
	credential TemporaryCredential,
) error {
	tag, err := tx.Exec(
		ctx,
		insertPasswordResetCredentialSQL,
		credential.ID,
		credential.UserID,
		credential.Purpose,
		credential.Generation,
		credential.AuthGeneration,
		credential.SourceOperationID,
		credential.State,
		credential.SecretDigest,
		credential.DeliveredToEmail,
		credential.IssuedAt,
		credential.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert password reset credential: %w", err)
	}
	return requireOnePasswordResetRow(tag)
}

func insertPasswordResetRecoveryJob(
	ctx context.Context,
	tx idempotency.Transaction,
	job PasswordResetRecoveryJob,
) error {
	tag, err := tx.Exec(
		ctx,
		insertPasswordResetRecoveryJobSQL,
		job.OperationID,
		job.Status,
		job.AttemptCount,
		job.NextAttemptAt,
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert password reset recovery job: %w", err)
	}
	return requireOnePasswordResetRow(tag)
}

func requireOnePasswordResetRow(tag pgconn.CommandTag) error {
	if tag.RowsAffected() != 1 {
		return ErrPasswordResetStateConflict
	}
	return nil
}

func validPasswordResetCandidateShape(candidate PasswordResetCandidate) bool {
	if candidate.Operation.Recovery != nil && !validRecoveryResetMetadata(candidate.Operation.Recovery) {
		return false
	}
	operation := candidate.Operation
	job := candidate.RecoveryJob
	if strings.TrimSpace(operation.ID) == "" ||
		strings.TrimSpace(operation.UserID) == "" ||
		strings.TrimSpace(operation.TemporaryCredentialID) == "" ||
		operation.CreatedAt.IsZero() ||
		operation.UpdatedAt.IsZero() ||
		operation.State != PasswordResetOperationPrepared ||
		operation.DeliveryState != PasswordResetDeliveryNotStarted ||
		operation.SettlementState != PasswordResetSettlementPending ||
		operation.DeliveryAttemptID != nil ||
		operation.DeliveryAttemptedAt != nil ||
		operation.DeliveryResultRecordedAt != nil ||
		operation.SettledAuthGeneration != nil ||
		operation.SuccessorAuthGeneration != nil ||
		operation.SettledAt != nil ||
		operation.TerminalAt != nil ||
		job.OperationID != operation.ID {
		return false
	}
	switch operation.RequestKind {
	case PasswordResetInitialReset:
		return operation.PredecessorOperationID == nil &&
			operation.ReplacedCredentialID == nil
	case PasswordResetResend:
		return operation.PredecessorOperationID != nil &&
			strings.TrimSpace(*operation.PredecessorOperationID) != "" &&
			operation.ReplacedCredentialID != nil &&
			strings.TrimSpace(*operation.ReplacedCredentialID) != ""
	default:
		return false
	}
}

const lockPasswordResetCandidateUserSQL = `
	SELECT id, auth_generation, password_reset_execution_generation, auth_integrity_hold
	FROM users
	WHERE id = $1
	FOR UPDATE
`

const readPasswordResetCandidateBindingSQL = `
	SELECT
		i.scope_hash,
		i.key_locator,
		i.key_digest,
		i.digest_key_version,
		i.request_digest,
		i.execution_generation,
		o.operation_id,
		o.user_id,
		o.request_kind,
		o.state,
		o.settlement_state,
		o.delivery_state,
		COALESCE(o.target_auth_generation, 0),
		o.settled_auth_generation,
		o.successor_auth_generation,
		o.settled_at,
		o.terminal_at,
		COALESCE(o.execution_generation, 0),
		COALESCE(
			i.key_digest = $3
			AND i.digest_key_version = $4
			AND i.request_digest = $5
			AND o.user_id = $6
			AND o.request_kind = $7
			AND o.predecessor_operation_id IS NOT DISTINCT FROM $8
			AND o.replaced_credential_id IS NOT DISTINCT FROM $9
			AND o.scope_hash = i.scope_hash
			AND o.key_locator = i.key_locator
			AND o.key_digest = i.key_digest
			AND o.digest_key_version = i.digest_key_version
			AND o.request_digest = i.request_digest
			AND o.execution_generation = i.execution_generation
			AND i.password_reset_operation_id = o.operation_id
			AND i.password_reset_user_id = o.user_id
			AND c.id = o.temporary_credential_id
			AND c.user_id = o.user_id
			AND c.purpose = 'password_reset'
			AND c.source_operation_id = o.operation_id
			AND c.auth_generation = o.target_auth_generation
			AND j.operation_id = o.operation_id,
			false
		),
		COALESCE(
			(
				(
					o.state = 'prepared'
					AND o.delivery_state = 'not_started'
					AND o.settlement_state = 'pending'
				)
				OR (
					o.state = 'dispatching'
					AND o.delivery_state = 'dispatching'
					AND o.settlement_state = 'pending'
				)
				OR (
					o.state = 'commit_unknown'
					AND o.delivery_state IN ('dispatching', 'unknown')
					AND o.settlement_state = 'pending'
				)
				OR (
					o.state IN ('recovery_required', 'converging')
					AND o.delivery_state = 'accepted'
					AND o.settlement_state = 'pending'
				)
				OR (
					o.state = 'succeeded'
					AND o.delivery_state = 'accepted'
					AND o.settlement_state IN (
						'temporary_active',
						'permanent_successor'
					)
				)
				OR (
					o.state = 'invalidated'
					AND o.delivery_state IN ('not_started', 'dispatching', 'unknown')
					AND o.settlement_state IN ('invalidated', 'integrity_hold')
				)
				OR (
					o.state = 'failed'
					AND o.delivery_state = 'rejected'
					AND o.settlement_state = 'invalidated'
				)
			)
			AND (
				(
					o.state IN ('succeeded', 'invalidated', 'failed')
					AND o.terminal_at IS NOT NULL
				)
				OR (
					o.state NOT IN ('succeeded', 'invalidated', 'failed')
					AND o.terminal_at IS NULL
				)
			)
			AND (
				(
					o.settlement_state = 'pending'
					AND o.settled_auth_generation IS NULL
					AND o.successor_auth_generation IS NULL
					AND o.settled_at IS NULL
				)
				OR (
					o.settlement_state = 'temporary_active'
					AND o.settled_auth_generation = o.target_auth_generation
					AND o.successor_auth_generation IS NULL
					AND o.settled_at IS NOT NULL
				)
				OR (
					o.settlement_state = 'permanent_successor'
					AND o.settled_auth_generation = o.target_auth_generation
					AND o.successor_auth_generation = o.target_auth_generation + 1
					AND o.settled_at IS NOT NULL
				)
				OR (
					o.settlement_state IN ('invalidated', 'integrity_hold')
					AND o.settled_auth_generation IS NULL
					AND o.successor_auth_generation IS NULL
					AND o.settled_at IS NOT NULL
				)
			)
			AND (
				(
					o.delivery_state = 'not_started'
					AND o.delivery_attempt_id IS NULL
					AND o.delivery_attempted_at IS NULL
					AND o.delivery_result_recorded_at IS NULL
				)
				OR (
					o.delivery_state = 'dispatching'
					AND o.delivery_attempt_id IS NOT NULL
					AND o.delivery_attempted_at IS NOT NULL
					AND o.delivery_result_recorded_at IS NULL
				)
				OR (
					o.delivery_state IN ('accepted', 'rejected', 'unknown')
					AND o.delivery_attempt_id IS NOT NULL
					AND o.delivery_attempted_at IS NOT NULL
					AND o.delivery_result_recorded_at IS NOT NULL
				)
			),
			false
		)
	FROM idempotency_records i
	LEFT JOIN password_reset_operations o
		ON o.scope_hash = i.scope_hash
		AND o.key_locator = i.key_locator
	LEFT JOIN temporary_credentials c
		ON c.id = o.temporary_credential_id
	LEFT JOIN password_reset_recovery_jobs j
		ON j.operation_id = o.operation_id
	WHERE i.scope_hash = $1
		AND i.key_locator = $2
	FOR UPDATE OF i
`

const readPasswordResetOrphanOperationSQL = `
	SELECT operation_id, user_id
	FROM password_reset_operations
	WHERE scope_hash = $1
		AND key_locator = $2
	FOR UPDATE
`

const setPasswordResetIntegrityHoldSQL = `
	WITH held_user AS (
		UPDATE users
		SET
			auth_integrity_hold = true,
			auth_integrity_hold_operation_id = COALESCE(
				auth_integrity_hold_operation_id,
				$2
			),
			auth_integrity_hold_reason = COALESCE(
				auth_integrity_hold_reason,
				$3
			),
			auth_integrity_hold_at = COALESCE(auth_integrity_hold_at, $4),
			updated_at = GREATEST(updated_at, $4)
		WHERE id = $1
		RETURNING id
	)
	INSERT INTO audit_events (
		id,
		occurred_at,
		actor_kind,
		actor_system,
		action,
		target_type,
		target_id,
		outcome,
		error_code,
		trace_id,
		metadata
	)
	SELECT
		$5,
		$4,
		'system',
		'identity-password-reset-store',
		'identity.password_reset.integrity_drift',
		'user',
		held_user.id::text,
		'held',
		$3,
		$5,
		jsonb_strip_nulls(jsonb_build_object(
			'reason', $3,
			'operation_id', $2
		))
	FROM held_user
`

const passwordResetGateSQL = `
	SELECT
		NOT current_account.auth_integrity_hold
		AND NOT EXISTS (
			SELECT 1
			FROM password_reset_operations unresolved
			WHERE unresolved.user_id = current_account.id
				AND unresolved.state NOT IN ('succeeded', 'invalidated', 'failed')
		)
		AND CASE current_account.credential_mode
			WHEN 'temporary' THEN (
				(
					current_account.password_reset_current_operation_id IS NULL
					AND current_account.password_reset_execution_generation = 0
					AND EXISTS (
						SELECT 1
						FROM temporary_credentials credential
						WHERE credential.user_id = current_account.id
							AND credential.purpose = 'new_administrator'
							AND credential.state = 'active'
							AND credential.auth_generation
								= current_account.auth_generation
							AND credential.source_operation_id IS NULL
					)
				)
				OR EXISTS (
					SELECT 1
					FROM password_reset_operations operation
					JOIN temporary_credentials credential
						ON credential.id = operation.temporary_credential_id
					WHERE operation.operation_id
							= current_account.password_reset_current_operation_id
						AND operation.user_id = current_account.id
						AND operation.state = 'succeeded'
						AND operation.delivery_state = 'accepted'
						AND operation.settlement_state = 'temporary_active'
						AND operation.settled_auth_generation
							= operation.target_auth_generation
						AND operation.successor_auth_generation IS NULL
						AND credential.user_id = current_account.id
						AND credential.purpose = 'password_reset'
						AND credential.state = 'active'
						AND credential.auth_generation
							= operation.target_auth_generation
						AND credential.source_operation_id = operation.operation_id
						AND current_account.auth_generation
							= operation.target_auth_generation
				)
			)
			WHEN 'permanent' THEN
				current_account.permanent_password_digest IS NOT NULL
				AND current_account.permanent_password_generation
					= current_account.auth_generation
				AND (
					(
						current_account.permanent_password_source_operation_id IS NULL
						AND current_account.permanent_password_source_credential_id IS NULL
					)
					OR EXISTS (
						SELECT 1
						FROM password_reset_operations operation
						JOIN temporary_credentials credential
							ON credential.id = operation.temporary_credential_id
						WHERE operation.operation_id
								= current_account.permanent_password_source_operation_id
							AND operation.user_id = current_account.id
							AND credential.id
								= current_account.permanent_password_source_credential_id
							AND credential.user_id = current_account.id
							AND credential.purpose = 'password_reset'
							AND credential.state = 'consumed'
							AND credential.auth_generation
								= operation.target_auth_generation
							AND credential.source_operation_id = operation.operation_id
							AND operation.state = 'succeeded'
							AND operation.delivery_state = 'accepted'
							AND operation.settlement_state = 'permanent_successor'
							AND operation.settled_auth_generation
								= operation.target_auth_generation
							AND operation.successor_auth_generation
								<= current_account.permanent_password_generation
					)
				)
			WHEN 'bootstrap' THEN
				current_account.permanent_password_digest IS NOT NULL
				AND current_account.permanent_password_generation
					= current_account.auth_generation
				AND current_account.permanent_password_source_operation_id IS NULL
				AND current_account.permanent_password_source_credential_id IS NULL
			WHEN 'recovery_bootstrap' THEN current_account.is_initial_admin
			WHEN 'recovery_locked' THEN current_account.recovery_required
			ELSE false
		END
	FROM users current_account
	WHERE current_account.id = $1
`

const passwordResetUserHasUnresolvedOperationSQL = `
	SELECT EXISTS (
		SELECT 1
		FROM password_reset_operations
		WHERE user_id = $1
			AND state NOT IN ('succeeded', 'invalidated', 'failed')
	)
`

const passwordResetResendLineageStatusSQL = `
	WITH RECURSIVE current_credential AS (
		SELECT
			id,
			user_id,
			purpose,
			generation,
			(
				SELECT count(*)
				FROM temporary_credentials sequence
				WHERE sequence.user_id = temporary_credentials.user_id
					AND sequence.generation
						<= temporary_credentials.generation
			) AS generation_ordinal,
			auth_generation,
			source_operation_id,
			state
		FROM temporary_credentials
		WHERE user_id = $1
			AND purpose = 'password_reset'
			AND state IN ('active', 'expired')
	),
	operation_evidence AS (
		SELECT
			operation.*,
			credential.id AS lineage_credential_id,
			credential.user_id AS lineage_credential_user_id,
			credential.purpose AS lineage_credential_purpose,
			credential.auth_generation AS lineage_credential_auth_generation,
			credential.source_operation_id AS lineage_credential_source_operation_id,
			credential.state AS lineage_credential_state,
			COALESCE(
				operation.state = 'succeeded'
				AND operation.settlement_state = 'temporary_active'
				AND operation.delivery_state = 'accepted'
				AND operation.settled_auth_generation
					= operation.target_auth_generation
				AND operation.successor_auth_generation IS NULL
				AND operation.settled_at IS NOT NULL
				AND operation.terminal_at IS NOT NULL
				AND operation.delivery_attempt_id IS NOT NULL
				AND operation.delivery_attempted_at IS NOT NULL
				AND operation.delivery_result_recorded_at IS NOT NULL
				AND credential.id = operation.temporary_credential_id
				AND credential.user_id = operation.user_id
				AND credential.purpose = 'password_reset'
				AND credential.auth_generation
					= operation.target_auth_generation
				AND credential.source_operation_id = operation.operation_id,
				false
			) AS temporary_authority_valid
		FROM password_reset_operations operation
		LEFT JOIN temporary_credentials credential
			ON credential.id = operation.temporary_credential_id
	),
	current_operation AS (
		SELECT operation_evidence.*
		FROM operation_evidence
		JOIN current_credential
			ON current_credential.source_operation_id
				= operation_evidence.operation_id
	),
	temporary_lineage (
		operation_id,
		user_id,
		temporary_credential_id,
		request_kind,
		predecessor_operation_id,
		replaced_credential_id,
		base_auth_generation,
		target_auth_generation,
		credential_id,
		credential_state,
		lineage_valid,
		depth,
		operation_path
	) AS (
		SELECT
			current_operation.operation_id,
			current_operation.user_id,
			current_operation.temporary_credential_id,
			current_operation.request_kind,
			current_operation.predecessor_operation_id,
			current_operation.replaced_credential_id,
			current_operation.base_auth_generation,
			current_operation.target_auth_generation,
			current_operation.lineage_credential_id,
			current_operation.lineage_credential_state,
			current_operation.temporary_authority_valid
				AND current_operation.user_id = $1
				AND current_operation.lineage_credential_id
					= current_credential.id
				AND current_credential.state IN ('active', 'expired'),
			0,
			ARRAY[current_operation.operation_id]
		FROM current_operation
		JOIN current_credential ON true
		UNION ALL
		SELECT
			parent.operation_id,
			parent.user_id,
			parent.temporary_credential_id,
			parent.request_kind,
			parent.predecessor_operation_id,
			parent.replaced_credential_id,
			parent.base_auth_generation,
			parent.target_auth_generation,
			parent.lineage_credential_id,
			parent.lineage_credential_state,
			child.lineage_valid
				AND child.request_kind = 'resend'
				AND child.predecessor_operation_id = parent.operation_id
				AND child.replaced_credential_id
					= parent.lineage_credential_id
				AND child.user_id = parent.user_id
				AND child.base_auth_generation
					= parent.target_auth_generation
				AND parent.temporary_authority_valid
				AND parent.lineage_credential_state = 'superseded',
			child.depth + 1,
			child.operation_path || parent.operation_id
		FROM temporary_lineage child
		JOIN operation_evidence parent
			ON parent.operation_id = child.predecessor_operation_id
		WHERE NOT parent.operation_id = ANY(child.operation_path)
	)
	SELECT
		current_account.id,
		current_account.credential_mode,
		current_account.auth_generation,
		current_account.permanent_password_digest IS NOT NULL,
		current_account.permanent_password_generation,
		current_account.permanent_password_source_operation_id,
		current_account.permanent_password_source_credential_id,
		permanent_operation.operation_id,
		permanent_operation.user_id,
		permanent_operation.temporary_credential_id,
		permanent_operation.state,
		permanent_operation.settlement_state,
		permanent_operation.delivery_state,
		permanent_operation.base_auth_generation,
		permanent_operation.target_auth_generation,
		permanent_operation.settled_auth_generation,
		permanent_operation.successor_auth_generation,
		permanent_operation.settled_at,
		permanent_operation.terminal_at,
		permanent_operation.delivery_attempt_id,
		permanent_operation.delivery_attempted_at,
		permanent_operation.delivery_result_recorded_at,
		permanent_credential.id,
		permanent_credential.user_id,
		permanent_credential.purpose,
		permanent_credential.generation,
		permanent_credential.auth_generation,
		permanent_credential.source_operation_id,
		permanent_credential.state,
		predecessor.operation_id,
		predecessor.user_id,
		predecessor.temporary_credential_id,
		predecessor.state,
		predecessor.settlement_state,
		predecessor.delivery_state,
		predecessor.target_auth_generation,
		predecessor.settled_auth_generation,
		predecessor.successor_auth_generation,
		predecessor.settled_at,
		predecessor.terminal_at,
		predecessor.delivery_attempt_id,
		predecessor.delivery_attempted_at,
		predecessor.delivery_result_recorded_at,
		replaced.id,
		replaced.user_id,
		replaced.purpose,
		replaced.auth_generation,
		replaced.source_operation_id,
		replaced.state,
		current_operation.operation_id,
		current_operation.user_id,
		current_operation.temporary_credential_id,
		current_operation.request_kind,
		current_operation.predecessor_operation_id,
		current_operation.replaced_credential_id,
		current_operation.state,
		current_operation.settlement_state,
		current_operation.delivery_state,
		current_operation.base_auth_generation,
		current_operation.target_auth_generation,
		current_operation.settled_auth_generation,
		current_operation.successor_auth_generation,
		current_operation.settled_at,
		current_operation.terminal_at,
		current_operation.delivery_attempt_id,
		current_operation.delivery_attempted_at,
		current_operation.delivery_result_recorded_at,
		current_credential.id,
		current_credential.user_id,
		current_credential.purpose,
		current_credential.generation,
		current_credential.generation_ordinal,
		current_credential.auth_generation,
		current_credential.source_operation_id,
		current_credential.state,
		COALESCE((
			SELECT lineage.lineage_valid
			FROM temporary_lineage lineage
			WHERE lineage.request_kind = 'initial_reset'
				AND lineage.predecessor_operation_id IS NULL
				AND lineage.replaced_credential_id IS NULL
			ORDER BY lineage.depth DESC
			LIMIT 1
		), false),
		COALESCE((
			SELECT bool_or(
				lineage.operation_id = $2
				AND lineage.credential_id = $3
				AND lineage.lineage_valid
			)
			FROM temporary_lineage lineage
		), false),
		(
			SELECT lineage.base_auth_generation
			FROM temporary_lineage lineage
			WHERE lineage.request_kind = 'initial_reset'
				AND lineage.predecessor_operation_id IS NULL
				AND lineage.replaced_credential_id IS NULL
				AND lineage.lineage_valid
			ORDER BY lineage.depth DESC
			LIMIT 1
		)
	FROM users current_account
	LEFT JOIN password_reset_operations predecessor
		ON predecessor.operation_id = $2
	LEFT JOIN temporary_credentials replaced
		ON replaced.id = $3
	LEFT JOIN password_reset_operations permanent_operation
		ON permanent_operation.operation_id
			= current_account.permanent_password_source_operation_id
	LEFT JOIN temporary_credentials permanent_credential
		ON permanent_credential.id
			= current_account.permanent_password_source_credential_id
	LEFT JOIN current_credential ON true
	LEFT JOIN current_operation
		ON current_operation.operation_id = current_credential.source_operation_id
	WHERE current_account.id = $1
`

const advancePasswordResetExecutionGenerationSQL = `
	UPDATE users
	SET
		password_reset_execution_generation = $3,
		password_reset_current_operation_id = $4,
		updated_at = $5
	WHERE id = $1
		AND password_reset_execution_generation = $2
		AND auth_integrity_hold = false
`

const insertPasswordResetOperationSQL = `
	INSERT INTO password_reset_operations (
		operation_id,
		user_id,
		temporary_credential_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation,
		request_kind,
		predecessor_operation_id,
		replaced_credential_id,
		base_auth_generation,
		target_auth_generation,
		settlement_state,
		delivery_state,
		state,
		created_at,
		updated_at
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19
	)
`

const insertPasswordResetCredentialSQL = `
	INSERT INTO temporary_credentials (
		id,
		user_id,
		purpose,
		generation,
		auth_generation,
		source_operation_id,
		state,
		secret_digest,
		delivered_to_email,
		issued_at,
		expires_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`

const insertPasswordResetRecoveryJobSQL = `
	INSERT INTO password_reset_recovery_jobs (
		operation_id,
		status,
		attempt_count,
		next_attempt_at,
		created_at,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6)
`

func insertRecoveryResetOperation(ctx context.Context, tx idempotency.Transaction, o PasswordResetOperation) error {
	if !validRecoveryResetMetadata(o.Recovery) {
		return ErrPasswordResetCandidateInvalid
	}
	m := o.Recovery
	tag, err := tx.Exec(ctx, insertRecoveryResetOperationSQL, o.ID, o.UserID, o.TemporaryCredentialID, o.ScopeHash, o.KeyLocator,
		o.KeyDigest, o.DigestKeyVersion, o.RequestDigest, o.ExecutionGeneration, o.RequestKind, o.PredecessorOperationID,
		o.ReplacedCredentialID, o.BaseAuthGeneration, o.TargetAuthGeneration, o.SettlementState, o.DeliveryState, o.State,
		o.CreatedAt, o.UpdatedAt, m.RunID, m.InstallationID, m.EvidenceRef, m.EvidenceDigest, m.ActorID, m.Role, m.Status)
	if err != nil {
		return err
	}
	return requireOnePasswordResetRow(tag)
}

const insertRecoveryResetOperationSQL = `INSERT INTO password_reset_operations(operation_id,user_id,temporary_credential_id,
 scope_hash,key_locator,key_digest,digest_key_version,request_digest,execution_generation,request_kind,
 predecessor_operation_id,replaced_credential_id,base_auth_generation,target_auth_generation,settlement_state,
 delivery_state,state,created_at,updated_at,recovery_run_id,recovery_installation_id,recovery_evidence_ref,
 recovery_evidence_digest,recovery_actor_id,recovery_target_role,recovery_target_status)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`

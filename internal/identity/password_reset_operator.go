package identity

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrPasswordResetOperatorInvalid = errors.New("password reset operator input is invalid")
var ErrPasswordResetSettlementCommitUnknown = errors.New("password reset settlement commit outcome is unknown")
var ErrPasswordResetSettlementRolledBack = errors.New("password reset settlement commit rolled back")

type PasswordResetPermitStats struct {
	Budget   int
	Current  int
	Peak     int
	Acquired uint64
	Released uint64
}

type PasswordResetConnectionPermit struct {
	tokens   chan struct{}
	mu       sync.Mutex
	current  int
	peak     int
	acquired uint64
	released uint64
}

func NewPasswordResetConnectionPermit(budget int) (*PasswordResetConnectionPermit, error) {
	if budget <= 0 {
		return nil, errors.New("password reset connection budget must be positive")
	}
	return &PasswordResetConnectionPermit{tokens: make(chan struct{}, budget)}, nil
}

func (permit *PasswordResetConnectionPermit) Acquire(ctx context.Context) (func(), error) {
	if permit == nil || ctx == nil {
		return nil, errors.New("password reset connection permit is required")
	}
	select {
	case permit.tokens <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	permit.mu.Lock()
	permit.current++
	permit.acquired++
	if permit.current > permit.peak {
		permit.peak = permit.current
	}
	permit.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			permit.mu.Lock()
			permit.current--
			permit.released++
			permit.mu.Unlock()
			<-permit.tokens
		})
	}, nil
}

func (permit *PasswordResetConnectionPermit) Stats() PasswordResetPermitStats {
	if permit == nil {
		return PasswordResetPermitStats{}
	}
	permit.mu.Lock()
	defer permit.mu.Unlock()
	return PasswordResetPermitStats{Budget: cap(permit.tokens), Current: permit.current, Peak: permit.peak, Acquired: permit.acquired, Released: permit.released}
}

type PasswordResetRecoveryLease struct {
	OperationID  string
	UserID       string
	Owner        string
	AttemptCount int64
	ExpiresAt    time.Time
}

type passwordResetTransactionStarter interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type PasswordResetOperatorDependencies struct {
	Transactions passwordResetTransactionStarter
	Store        *PasswordResetStore
	Audit        LoginAuditAppender
	IDs          LoginIDGenerator
	Clock        LoginClock
	Permit       *PasswordResetConnectionPermit
}

type PasswordResetOperator struct {
	permit      *PasswordResetConnectionPermit
	settle      func(context.Context, PasswordResetRecoveryLease) error
	maintenance *passwordResetMaintenanceAuthority
}

func NewPasswordResetOperator(dependencies PasswordResetOperatorDependencies) (*PasswordResetOperator, error) {
	if dependencies.Transactions == nil || dependencies.Store == nil || dependencies.Audit == nil || dependencies.IDs == nil || dependencies.Clock == nil || dependencies.Permit == nil {
		return nil, errors.New("password reset operator dependencies are incomplete")
	}
	return &PasswordResetOperator{permit: dependencies.Permit, settle: func(ctx context.Context, lease PasswordResetRecoveryLease) error {
		return dependencies.Store.settlePasswordResetRecovery(ctx, dependencies.Transactions, dependencies.Audit, dependencies.IDs, dependencies.Clock, lease)
	}}, nil
}

func newPasswordResetOperatorForTest(permit *PasswordResetConnectionPermit, settle func(context.Context, PasswordResetRecoveryLease) error) *PasswordResetOperator {
	return &PasswordResetOperator{permit: permit, settle: settle}
}

func (operator *PasswordResetOperator) Execute(ctx context.Context, lease PasswordResetRecoveryLease) error {
	if operator == nil || operator.permit == nil {
		return ErrPasswordResetOperatorInvalid
	}
	release, err := operator.permit.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return operator.executeHeld(ctx, lease)
}

func (operator *PasswordResetOperator) executeHeld(ctx context.Context, lease PasswordResetRecoveryLease) error {
	if operator == nil || operator.settle == nil || ctx == nil || strings.TrimSpace(lease.OperationID) == "" || strings.TrimSpace(lease.UserID) == "" || strings.TrimSpace(lease.Owner) == "" || lease.AttemptCount <= 0 {
		return ErrPasswordResetOperatorInvalid
	}
	return operator.settle(ctx, lease)
}

type passwordResetSettlementOperation struct {
	state                PasswordResetOperationState
	delivery             PasswordResetDeliveryState
	settlement           PasswordResetSettlementState
	credentialID         string
	requestKind          PasswordResetRequestKind
	replacedCredentialID *string
	baseGeneration       int64
	targetGeneration     int64
	scopeHash            []byte
	keyLocator           []byte
	keyDigest            []byte
	requestDigest        []byte
	digestKeyVersion     string
	executionGeneration  int64
}

func (store *PasswordResetStore) settlePasswordResetRecovery(ctx context.Context, transactions passwordResetTransactionStarter, auditor LoginAuditAppender, ids LoginIDGenerator, clock LoginClock, lease PasswordResetRecoveryLease) (err error) {
	tx, err := transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin password reset settlement: %w", err)
	}
	var operation passwordResetSettlementOperation
	defer func() {
		err = finishPasswordResetSettlementTransaction(ctx, tx, transactions, clock, lease, operation, err)
	}()
	var userGeneration int64
	var credentialMode CredentialMode
	var currentOperationID *string
	if err = tx.QueryRow(ctx, passwordResetSettlementLockUserSQL, lease.UserID).Scan(&userGeneration, &credentialMode, &currentOperationID); err != nil {
		return fmt.Errorf("lock password reset settlement user: %w", err)
	}
	now := clock.Now().UTC().Truncate(time.Microsecond)
	operation, err = lockPasswordResetSettlementOperation(ctx, tx, lease, now)
	if err != nil {
		return err
	}
	if operation.state == PasswordResetOperationSucceeded || operation.state == PasswordResetOperationInvalidated || operation.state == PasswordResetOperationFailed {
		completionNow := clock.Now().UTC().Truncate(time.Microsecond)
		if err = completePasswordResetRecoveryJob(ctx, tx, lease, completionNow); err != nil {
			return err
		}
		return nil
	}
	switch {
	case operation.delivery == PasswordResetDeliveryAccepted && (operation.state == PasswordResetOperationRecoveryRequired || operation.state == PasswordResetOperationConverging):
		err = settleAcceptedPasswordReset(ctx, tx, lease, operation, userGeneration, credentialMode, currentOperationID, now)
	case operation.delivery != PasswordResetDeliveryAccepted && (operation.state == PasswordResetOperationPrepared || operation.state == PasswordResetOperationDispatching || operation.state == PasswordResetOperationCommitUnknown):
		err = settleUnacceptedPasswordReset(ctx, tx, lease, operation, now)
	default:
		err = ErrPasswordResetStateConflict
	}
	if err != nil {
		return err
	}
	auditID, err := ids.NewID()
	if err != nil {
		return fmt.Errorf("create password reset settlement audit id: %w", err)
	}
	system := "password-reset-worker"
	if err = auditor.Append(ctx, tx, audit.Event{ID: auditID, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &system, Action: "identity.password_reset.recovered", TargetType: "password_reset_operation", TargetID: lease.OperationID, Outcome: "succeeded", TraceID: auditID, Metadata: map[string]any{"attempt_count": lease.AttemptCount}}); err != nil {
		return fmt.Errorf("append password reset settlement audit: %w", err)
	}
	completionNow := clock.Now().UTC().Truncate(time.Microsecond)
	return completePasswordResetRecoveryJob(ctx, tx, lease, completionNow)
}

func finishPasswordResetSettlementTransaction(
	ctx context.Context,
	tx pgx.Tx,
	transactions passwordResetTransactionStarter,
	clock LoginClock,
	lease PasswordResetRecoveryLease,
	operation passwordResetSettlementOperation,
	operationErr error,
) error {
	if operationErr != nil {
		if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(operationErr, fmt.Errorf("rollback password reset transaction: %w", rollbackErr))
		}
		return operationErr
	}
	commitErr := tx.Commit(ctx)
	if commitErr == nil {
		return nil
	}
	outcome, verifyErr := classifyPasswordResetSettlementCommit(
		ctx,
		transactions,
		clock,
		lease,
		operation,
	)
	switch outcome {
	case passwordResetSettlementCommitCommitted:
		return nil
	case passwordResetSettlementCommitRolledBack:
		return errors.Join(
			ErrPasswordResetSettlementRolledBack,
			fmt.Errorf("commit password reset settlement: %w", commitErr),
		)
	default:
		return errors.Join(
			ErrPasswordResetSettlementCommitUnknown,
			fmt.Errorf("commit password reset settlement: %w", commitErr),
			verifyErr,
		)
	}
}

func finishPasswordResetTransaction(ctx context.Context, tx pgx.Tx, operationErr error) error {
	if operationErr != nil {
		if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(operationErr, fmt.Errorf("rollback password reset transaction: %w", rollbackErr))
		}
		return operationErr
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit password reset transaction: %w", err)
	}
	return nil
}

type passwordResetSettlementCommitOutcome uint8

const (
	passwordResetSettlementCommitUnknown passwordResetSettlementCommitOutcome = iota
	passwordResetSettlementCommitCommitted
	passwordResetSettlementCommitRolledBack
)

type passwordResetSettlementAuthority struct {
	userGeneration     int64
	credentialMode     CredentialMode
	currentOperationID *string
	integrityHold      bool
	operationState     PasswordResetOperationState
	deliveryState      PasswordResetDeliveryState
	settlementState    PasswordResetSettlementState
	credentialID       string
	targetGeneration   int64
	jobStatus          PasswordResetRecoveryJobStatus
	leaseOwner         *string
	leaseExpiresAt     *time.Time
	attemptCount       int64
	completedAt        *time.Time
	credentialUserID   string
	credentialState    TemporaryCredentialState
	sourceOperationID  *string
	credentialGen      int64
	activeSessions     int64
	idempotencyStatus  string
	resultCode         *string
	responseSnapshot   *string
	auditCount         int64
}

func classifyPasswordResetSettlementCommit(
	ctx context.Context,
	transactions passwordResetTransactionStarter,
	clock LoginClock,
	lease PasswordResetRecoveryLease,
	operation passwordResetSettlementOperation,
) (outcome passwordResetSettlementCommitOutcome, err error) {
	tx, err := transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return passwordResetSettlementCommitUnknown, fmt.Errorf("begin password reset settlement verification: %w", err)
	}
	defer func() {
		finishErr := finishPasswordResetTransaction(ctx, tx, err)
		if finishErr != nil {
			outcome = passwordResetSettlementCommitUnknown
			err = errors.Join(err, fmt.Errorf("finish password reset settlement verification: %w", finishErr))
		}
	}()
	if err = lockPasswordResetRecoveryUser(ctx, tx, lease.UserID); err != nil {
		return passwordResetSettlementCommitUnknown, fmt.Errorf("lock password reset settlement verification user: %w", err)
	}
	_ = clock.Now().UTC().Truncate(time.Microsecond)
	var authority passwordResetSettlementAuthority
	err = tx.QueryRow(ctx, passwordResetSettlementAuthoritySQL, lease.OperationID, lease.UserID, lease.AttemptCount).Scan(
		&authority.userGeneration,
		&authority.credentialMode,
		&authority.currentOperationID,
		&authority.integrityHold,
		&authority.operationState,
		&authority.deliveryState,
		&authority.settlementState,
		&authority.credentialID,
		&authority.targetGeneration,
		&authority.jobStatus,
		&authority.leaseOwner,
		&authority.leaseExpiresAt,
		&authority.attemptCount,
		&authority.completedAt,
		&authority.credentialUserID,
		&authority.credentialState,
		&authority.sourceOperationID,
		&authority.credentialGen,
		&authority.activeSessions,
		&authority.idempotencyStatus,
		&authority.resultCode,
		&authority.responseSnapshot,
		&authority.auditCount,
	)
	if err != nil {
		return passwordResetSettlementCommitUnknown, fmt.Errorf("read password reset settlement authority: %w", err)
	}
	switch {
	case passwordResetSettlementCommitted(authority, lease, operation):
		return passwordResetSettlementCommitCommitted, nil
	case passwordResetSettlementRolledBack(authority, lease, operation):
		return passwordResetSettlementCommitRolledBack, nil
	default:
		return passwordResetSettlementCommitUnknown, nil
	}
}

func passwordResetSettlementCommitted(
	authority passwordResetSettlementAuthority,
	lease PasswordResetRecoveryLease,
	operation passwordResetSettlementOperation,
) bool {
	if authority.integrityHold ||
		authority.currentOperationID == nil ||
		*authority.currentOperationID != lease.OperationID ||
		authority.operationState == "" ||
		authority.credentialID != operation.credentialID ||
		authority.targetGeneration != operation.targetGeneration ||
		authority.jobStatus != PasswordResetRecoveryJobSucceeded ||
		authority.leaseOwner != nil ||
		authority.leaseExpiresAt != nil ||
		authority.attemptCount != lease.AttemptCount ||
		authority.completedAt == nil ||
		authority.credentialUserID != lease.UserID ||
		authority.sourceOperationID == nil ||
		*authority.sourceOperationID != lease.OperationID ||
		authority.credentialGen != operation.targetGeneration ||
		authority.auditCount != 1 {
		return false
	}
	if operation.delivery == PasswordResetDeliveryAccepted {
		return authority.userGeneration == operation.targetGeneration &&
			authority.credentialMode == CredentialModeTemporary &&
			authority.operationState == PasswordResetOperationSucceeded &&
			authority.deliveryState == PasswordResetDeliveryAccepted &&
			authority.settlementState == PasswordResetSettlementTemporaryActive &&
			authority.credentialState == TemporaryCredentialActive &&
			authority.activeSessions == 0 &&
			passwordResetIdempotencyTerminal(authority, lease.OperationID, true)
	}
	return authority.userGeneration == operation.baseGeneration &&
		authority.operationState == PasswordResetOperationInvalidated &&
		authority.deliveryState == operation.delivery &&
		authority.settlementState == PasswordResetSettlementInvalidated &&
		authority.credentialState == TemporaryCredentialRecoveryInvalidated &&
		passwordResetIdempotencyTerminal(authority, lease.OperationID, false)
}

func passwordResetSettlementRolledBack(
	authority passwordResetSettlementAuthority,
	lease PasswordResetRecoveryLease,
	operation passwordResetSettlementOperation,
) bool {
	return !authority.integrityHold &&
		authority.userGeneration == operation.baseGeneration &&
		authority.currentOperationID != nil &&
		*authority.currentOperationID == lease.OperationID &&
		authority.operationState == operation.state &&
		authority.deliveryState == operation.delivery &&
		authority.settlementState == PasswordResetSettlementPending &&
		authority.credentialID == operation.credentialID &&
		authority.targetGeneration == operation.targetGeneration &&
		authority.jobStatus == PasswordResetRecoveryJobLeased &&
		authority.leaseOwner != nil &&
		*authority.leaseOwner == lease.Owner &&
		authority.attemptCount == lease.AttemptCount &&
		authority.completedAt == nil &&
		authority.credentialUserID == lease.UserID &&
		authority.credentialState == TemporaryCredentialPendingDelivery &&
		authority.sourceOperationID != nil &&
		*authority.sourceOperationID == lease.OperationID &&
		authority.credentialGen == operation.targetGeneration &&
		(authority.idempotencyStatus == "processing" || authority.idempotencyStatus == "uncertain") &&
		authority.resultCode == nil &&
		authority.responseSnapshot == nil &&
		authority.auditCount == 0
}

func passwordResetIdempotencyTerminal(authority passwordResetSettlementAuthority, operationID string, succeeded bool) bool {
	status := "failed"
	result := "password_reset_invalidated"
	outcome := "invalidated"
	if succeeded {
		status = "succeeded"
		result = "password_reset_succeeded"
		outcome = "succeeded"
	}
	if authority.idempotencyStatus != status ||
		authority.resultCode == nil ||
		*authority.resultCode != result ||
		authority.responseSnapshot == nil {
		return false
	}
	var snapshot struct {
		Outcome    string `json:"outcome"`
		ResourceID string `json:"resource_id"`
	}
	if err := json.Unmarshal([]byte(*authority.responseSnapshot), &snapshot); err != nil {
		return false
	}
	return snapshot.Outcome == outcome && snapshot.ResourceID == operationID
}

func lockPasswordResetSettlementOperation(ctx context.Context, tx pgx.Tx, lease PasswordResetRecoveryLease, now time.Time) (passwordResetSettlementOperation, error) {
	var operation passwordResetSettlementOperation
	var attemptCount int64
	var owner *string
	var expires *time.Time
	err := tx.QueryRow(ctx, passwordResetSettlementLockOperationSQL, lease.OperationID).Scan(
		&operation.state, &operation.delivery, &operation.settlement, &operation.credentialID, &operation.requestKind, &operation.replacedCredentialID,
		&operation.baseGeneration, &operation.targetGeneration, &operation.scopeHash, &operation.keyLocator, &operation.keyDigest, &operation.requestDigest,
		&operation.digestKeyVersion, &operation.executionGeneration, &attemptCount, &owner, &expires,
	)
	if err != nil {
		return operation, fmt.Errorf("lock password reset settlement operation: %w", err)
	}
	if owner == nil || *owner != lease.Owner || expires == nil || !expires.After(now) || attemptCount != lease.AttemptCount {
		return operation, ErrPasswordResetStateConflict
	}
	return operation, nil
}

func settleAcceptedPasswordReset(ctx context.Context, tx pgx.Tx, lease PasswordResetRecoveryLease, operation passwordResetSettlementOperation, userGeneration int64, credentialMode CredentialMode, currentOperationID *string, now time.Time) error {
	if operation.settlement != PasswordResetSettlementPending || userGeneration != operation.baseGeneration || currentOperationID == nil || *currentOperationID != lease.OperationID {
		return ErrPasswordResetStateConflict
	}
	if operation.state == PasswordResetOperationRecoveryRequired {
		if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementConvergingSQL, now, lease.OperationID)); err != nil {
			return err
		}
	}
	var state TemporaryCredentialState
	var credentialUser string
	var sourceOperation *string
	var authGeneration int64
	if err := tx.QueryRow(ctx, passwordResetSettlementLockCredentialSQL, operation.credentialID).Scan(&credentialUser, &state, &sourceOperation, &authGeneration); err != nil {
		return fmt.Errorf("lock password reset credential: %w", err)
	}
	if credentialUser != lease.UserID || state != TemporaryCredentialPendingDelivery || sourceOperation == nil || *sourceOperation != lease.OperationID || authGeneration != operation.targetGeneration {
		return ErrPasswordResetStateConflict
	}
	if operation.requestKind == PasswordResetResend {
		if operation.replacedCredentialID == nil {
			return ErrPasswordResetStateConflict
		}
		if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementSupersedeSQL, now, *operation.replacedCredentialID, lease.UserID)); err != nil {
			return err
		}
	}
	if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementActivateSQL, operation.credentialID, lease.UserID)); err != nil {
		return err
	}
	if credentialMode == CredentialModeRecoveryLocked {
		return ErrPasswordResetStateConflict
	}
	if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementUserSQL, operation.targetGeneration, now, lease.UserID, operation.baseGeneration, lease.OperationID)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, passwordResetSettlementRevokeSessionsSQL, now, lease.UserID); err != nil {
		return fmt.Errorf("revoke password reset sessions: %w", err)
	}
	if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementSucceededSQL, operation.targetGeneration, now, lease.OperationID)); err != nil {
		return err
	}
	if err := completePasswordResetIdempotency(ctx, tx, operation, lease.OperationID, true, now); err != nil {
		return err
	}
	return nil
}

func settleUnacceptedPasswordReset(ctx context.Context, tx pgx.Tx, lease PasswordResetRecoveryLease, operation passwordResetSettlementOperation, now time.Time) error {
	if operation.settlement != PasswordResetSettlementPending {
		return ErrPasswordResetStateConflict
	}
	if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementInvalidateCredentialSQL, now, operation.credentialID, lease.UserID)); err != nil {
		return err
	}
	if err := requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementInvalidatedSQL, now, lease.OperationID)); err != nil {
		return err
	}
	if err := completePasswordResetIdempotency(ctx, tx, operation, lease.OperationID, false, now); err != nil {
		return err
	}
	return nil
}

func completePasswordResetIdempotency(ctx context.Context, tx pgx.Tx, operation passwordResetSettlementOperation, operationID string, succeeded bool, now time.Time) error {
	status := "failed"
	result := "password_reset_invalidated"
	outcome := "invalidated"
	if succeeded {
		status = "succeeded"
		result = "password_reset_succeeded"
		outcome = "succeeded"
	}
	snapshot := fmt.Sprintf(`{"outcome":%q,"resource_id":%q}`, outcome, operationID)
	return requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementIdempotencySQL, status, result, snapshot, now, operation.scopeHash, operation.keyLocator, operation.keyDigest, operation.requestDigest, operation.digestKeyVersion, operation.executionGeneration))
}

func completePasswordResetRecoveryJob(ctx context.Context, tx pgx.Tx, lease PasswordResetRecoveryLease, now time.Time) error {
	return requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementCompleteJobSQL, now, lease.OperationID, lease.Owner, lease.AttemptCount))
}

func requirePasswordResetOneRow(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return fmt.Errorf("persist password reset settlement: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrPasswordResetStateConflict
	}
	return nil
}

const passwordResetSettlementLockUserSQL = `SELECT auth_generation, credential_mode, password_reset_current_operation_id FROM users WHERE id = $1 FOR UPDATE`
const passwordResetSettlementLockOperationSQL = `SELECT o.state,o.delivery_state,o.settlement_state,o.temporary_credential_id,o.request_kind,o.replaced_credential_id,o.base_auth_generation,o.target_auth_generation,o.scope_hash,o.key_locator,o.key_digest,o.request_digest,o.digest_key_version,o.execution_generation,j.attempt_count,j.lease_owner,j.lease_expires_at FROM password_reset_operations o JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id WHERE o.operation_id=$1 FOR UPDATE OF o,j`
const passwordResetSettlementConvergingSQL = `UPDATE password_reset_operations SET state='converging',updated_at=$1 WHERE operation_id=$2 AND state='recovery_required'`
const passwordResetSettlementLockCredentialSQL = `SELECT user_id,state,source_operation_id,auth_generation FROM temporary_credentials WHERE id=$1 FOR UPDATE`
const passwordResetSettlementSupersedeSQL = `UPDATE temporary_credentials SET state='superseded',superseded_at=$1 WHERE id=$2 AND user_id=$3 AND state IN ('active','expired')`
const passwordResetSettlementActivateSQL = `UPDATE temporary_credentials SET state='active' WHERE id=$1 AND user_id=$2 AND state='pending_delivery'`
const passwordResetSettlementUserSQL = `UPDATE users SET auth_generation=$1,credential_mode='temporary',updated_at=$2 WHERE id=$3 AND auth_generation=$4 AND password_reset_current_operation_id=$5 AND auth_integrity_hold=false`
const passwordResetSettlementRevokeSessionsSQL = `UPDATE sessions SET revoked_at=$1,revoke_reason='password_reset_settled' WHERE user_id=$2 AND revoked_at IS NULL`
const passwordResetSettlementSucceededSQL = `UPDATE password_reset_operations SET state='succeeded',settlement_state='temporary_active',settled_auth_generation=$1,settled_at=$2,terminal_at=$2,updated_at=$2 WHERE operation_id=$3 AND state='converging' AND delivery_state='accepted' AND settlement_state='pending'`
const passwordResetSettlementInvalidateCredentialSQL = `UPDATE temporary_credentials SET state='recovery_invalidated',invalidated_at=$1 WHERE id=$2 AND user_id=$3 AND state='pending_delivery'`
const passwordResetSettlementInvalidatedSQL = `UPDATE password_reset_operations SET state='invalidated',settlement_state='invalidated',settled_at=$1,terminal_at=$1,updated_at=$1,recovery_reason='delivery_not_accepted' WHERE operation_id=$2 AND state IN ('prepared','dispatching','commit_unknown') AND delivery_state IN ('not_started','dispatching','unknown') AND settlement_state='pending'`
const passwordResetSettlementIdempotencySQL = `UPDATE idempotency_records SET status=$1,result_code=$2,response_snapshot=$3::jsonb,locked_until=NULL,updated_at=$4 WHERE scope_hash=$5 AND key_locator=$6 AND key_digest=$7 AND request_digest=$8 AND digest_key_version=$9 AND execution_generation=$10 AND status IN ('processing','uncertain')`
const passwordResetSettlementCompleteJobSQL = `UPDATE password_reset_recovery_jobs SET status='succeeded',lease_owner=NULL,lease_expires_at=NULL,last_failure_code=NULL,updated_at=$1,completed_at=$1 WHERE operation_id=$2 AND status='leased' AND lease_owner=$3 AND attempt_count=$4 AND lease_expires_at>$1`
const passwordResetSettlementAuthoritySQL = `
SELECT
	u.auth_generation,
	u.credential_mode,
	u.password_reset_current_operation_id,
	u.auth_integrity_hold,
	o.state,
	o.delivery_state,
	o.settlement_state,
	o.temporary_credential_id,
	o.target_auth_generation,
	j.status,
	j.lease_owner,
	j.lease_expires_at,
	j.attempt_count,
	j.completed_at,
	c.user_id,
	c.state,
	c.source_operation_id,
	c.auth_generation,
	(SELECT count(*) FROM sessions s WHERE s.user_id=$2 AND s.revoked_at IS NULL),
	i.status,
	i.result_code,
	i.response_snapshot::text,
	(
		SELECT count(*)
		FROM audit_events a
		WHERE a.action='identity.password_reset.recovered'
			AND a.actor_kind='system'
			AND a.actor_system='password-reset-worker'
			AND a.target_type='password_reset_operation'
			AND a.target_id=$1
			AND a.outcome='succeeded'
			AND a.metadata->>'attempt_count'=$3::text
	)
FROM users u
JOIN password_reset_operations o ON o.operation_id=$1 AND o.user_id=u.id
JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id
JOIN temporary_credentials c ON c.id=o.temporary_credential_id AND c.user_id=o.user_id
JOIN idempotency_records i
	ON i.password_reset_operation_id=o.operation_id
	AND i.password_reset_user_id=o.user_id
	AND i.scope_hash=o.scope_hash
	AND i.key_locator=o.key_locator
	AND i.key_digest=o.key_digest
	AND i.request_digest=o.request_digest
	AND i.digest_key_version=o.digest_key_version
	AND i.execution_generation=o.execution_generation
WHERE u.id=$2`

// PasswordResetMaintenanceCode is a closed maintenance-only failure category.
type PasswordResetMaintenanceCode string

const (
	PasswordResetMaintenanceInvalidRequest       PasswordResetMaintenanceCode = "maintenance.invalid_request"
	PasswordResetMaintenanceInvalidConfig        PasswordResetMaintenanceCode = "maintenance.invalid_config"
	PasswordResetMaintenanceModeConflict         PasswordResetMaintenanceCode = "maintenance.mode_conflict"
	PasswordResetMaintenanceUnsupportedPlatform  PasswordResetMaintenanceCode = "maintenance.unsupported_platform"
	PasswordResetMaintenanceAuthorityUnavailable PasswordResetMaintenanceCode = "maintenance.authority_unavailable"
	PasswordResetMaintenanceForbidden            PasswordResetMaintenanceCode = "maintenance.forbidden"
	PasswordResetMaintenanceStateConflict        PasswordResetMaintenanceCode = "maintenance.state_conflict"
	PasswordResetMaintenanceOutcomeUnknown       PasswordResetMaintenanceCode = "maintenance.outcome_unknown"
	PasswordResetMaintenanceFailed               PasswordResetMaintenanceCode = "maintenance.failed"
	passwordResetMaintenanceAction                                            = "identity.password_reset.requeue"
	passwordResetMaintenanceAuditAction                                       = "identity.password_reset.requeued"
	passwordResetMaintenanceRole                                              = "password_reset_requeue_executor"
	passwordResetMaintenancePayloadVersion                                    = "han-password-reset-requeue/v1"
	passwordResetMaintenanceTrustVersion                                      = "han-password-reset-maintenance-trust/v1"
	passwordResetMaintenanceResultVersion                                     = "han-password-reset-requeue-result/v1"
	passwordResetMaintenanceAuditVersion                                      = "identity-password-reset-requeue-audit/v1"
	passwordResetMaintenanceDomain                                            = "Han/identity-auth/S10.5/requeue-approval/v1\x00"
	passwordResetMaintenanceTimeFormat                                        = "2006-01-02T15:04:05.000000Z"
	passwordResetMaintenanceEnvelopeLimit                                     = 16 * 1024
	passwordResetMaintenancePayloadLimit                                      = 12 * 1024
	passwordResetMaintenanceMetadataLimit                                     = 64 * 1024
)

// PasswordResetMaintenanceError exposes only the stable category; causes stay private.
type PasswordResetMaintenanceError struct {
	Code  PasswordResetMaintenanceCode
	cause error
}

func (err *PasswordResetMaintenanceError) Error() string { return string(err.Code) }
func (err *PasswordResetMaintenanceError) Unwrap() error { return err.cause }

func passwordResetMaintenanceFailure(code PasswordResetMaintenanceCode, cause error) error {
	return &PasswordResetMaintenanceError{Code: code, cause: cause}
}

// PasswordResetMaintenanceConfig contains only explicit pre-DSN maintenance settings.
type PasswordResetMaintenanceConfig struct {
	TrustJSON       []byte
	MaxTTL          time.Duration
	OperationBudget time.Duration
	ShutdownBudget  time.Duration
}

// PasswordResetRequeueSnapshot is the exact ordered ADR-0026 before/after value.
type PasswordResetRequeueSnapshot struct {
	UserStatus                   string  `json:"user_status"`
	UserAuthGeneration           int64   `json:"user_auth_generation"`
	UserExecutionGeneration      int64   `json:"user_execution_generation"`
	UserCurrentOperationID       *string `json:"user_current_operation_id"`
	CredentialMode               string  `json:"credential_mode"`
	IntegrityHold                bool    `json:"integrity_hold"`
	IntegrityHoldOperationID     *string `json:"integrity_hold_operation_id"`
	IntegrityHoldReason          *string `json:"integrity_hold_reason"`
	IntegrityHoldAt              *string `json:"integrity_hold_at"`
	UserUpdatedAt                string  `json:"user_updated_at"`
	OperationState               string  `json:"operation_state"`
	DeliveryState                string  `json:"delivery_state"`
	SettlementState              string  `json:"settlement_state"`
	OperationExecutionGeneration int64   `json:"operation_execution_generation"`
	TemporaryCredentialID        string  `json:"temporary_credential_id"`
	OperationUpdatedAt           string  `json:"operation_updated_at"`
	JobStatus                    string  `json:"job_status"`
	AttemptCount                 int64   `json:"attempt_count"`
	NextAttemptAt                string  `json:"next_attempt_at"`
	LeaseOwner                   *string `json:"lease_owner"`
	LeaseExpiresAt               *string `json:"lease_expires_at"`
	LastFailureCode              *string `json:"last_failure_code"`
	JobCreatedAt                 string  `json:"job_created_at"`
	JobUpdatedAt                 string  `json:"job_updated_at"`
	CompletedAt                  *string `json:"completed_at"`
}

type passwordResetRequeuePayload struct {
	Version         string                       `json:"version"`
	DeploymentID    string                       `json:"deployment_id"`
	Epoch           string                       `json:"epoch"`
	RequestID       string                       `json:"request_id"`
	Action          string                       `json:"action"`
	UserID          string                       `json:"user_id"`
	OperationID     string                       `json:"operation_id"`
	ApproverKeyID   string                       `json:"approver_key_id"`
	ExecutorSubject string                       `json:"executor_subject"`
	IssuedAt        int64                        `json:"issued_at"`
	ExpiresAt       int64                        `json:"expires_at"`
	Reason          string                       `json:"reason"`
	EvidenceRef     string                       `json:"evidence_ref"`
	Before          PasswordResetRequeueSnapshot `json:"before"`
}

type passwordResetRequeueEnvelope struct {
	Payload           string `json:"payload"`
	ApprovalSignature string `json:"approval_signature"`
}

type passwordResetMaintenanceKey struct {
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
	Subject   string `json:"subject"`
	Action    string `json:"action"`
	NotBefore int64  `json:"not_before"`
	ExpiresAt int64  `json:"expires_at"`
}

type passwordResetMaintenanceExecutor struct {
	Subject   string `json:"subject"`
	Action    string `json:"action"`
	NotBefore int64  `json:"not_before"`
	ExpiresAt int64  `json:"expires_at"`
}

type passwordResetMaintenanceTrust struct {
	Version      string                             `json:"version"`
	DeploymentID string                             `json:"deployment_id"`
	Epoch        string                             `json:"epoch"`
	ApproverKeys []passwordResetMaintenanceKey      `json:"approver_keys"`
	Executors    []passwordResetMaintenanceExecutor `json:"executors"`
}

type passwordResetMaintenanceAuthority struct {
	trust          passwordResetMaintenanceTrust
	config         PasswordResetMaintenanceConfig
	clock          LoginClock
	subject        string
	deadline       time.Time
	clientEncoding func(pgx.Tx) bool
}

// PasswordResetRequeueRequest binds copied canonical bytes to one validated invocation.
// All fields are private so an internal caller cannot construct authorization.
type PasswordResetRequeueRequest struct {
	owner     *passwordResetMaintenanceAuthority
	payload   []byte
	signature []byte
	deadline  time.Time
}

// PasswordResetRequeueDependencies supplies the single caller-owned database authority.
type PasswordResetRequeueDependencies struct {
	Transactions passwordResetTransactionStarter
	Store        *PasswordResetStore
	Audit        audit.Store
	Permit       *PasswordResetConnectionPermit
}

// PasswordResetRequeueDisposition describes this invocation, not automatic attempts.
type PasswordResetRequeueDisposition string

const (
	PasswordResetRequeueApplied        PasswordResetRequeueDisposition = "applied"
	PasswordResetRequeueAlreadyApplied PasswordResetRequeueDisposition = "already_applied"
)

// PasswordResetRequeueReceipt preserves the original committed all-string receipt.
type PasswordResetRequeueReceipt struct {
	Version     string `json:"version"`
	RequestID   string `json:"request_id"`
	Outcome     string `json:"outcome"`
	OperationID string `json:"operation_id"`
	JobStatus   string `json:"job_status"`
	AuditID     string `json:"audit_id"`
}

// PasswordResetRequeueResult separates the current disposition from historical facts.
type PasswordResetRequeueResult struct {
	Disposition PasswordResetRequeueDisposition
	Receipt     PasswordResetRequeueReceipt
	After       PasswordResetRequeueSnapshot
}

// NewPasswordResetMaintenanceOperator freezes controlled trust and the actual Windows
// process owner before any database dependency is accepted.
func NewPasswordResetMaintenanceOperator(config PasswordResetMaintenanceConfig, clock LoginClock) (*PasswordResetOperator, error) {
	return newPasswordResetMaintenanceOperator(config, clock, runtime.GOOS, func() (string, error) {
		current, err := user.Current()
		if err != nil {
			return "", err
		}
		return "windows-sid:" + current.Uid, nil
	})
}

func newPasswordResetMaintenanceOperator(config PasswordResetMaintenanceConfig, clock LoginClock, platform string, subject func() (string, error)) (*PasswordResetOperator, error) {
	if clock == nil || config.MaxTTL <= 0 || config.OperationBudget <= 0 || config.ShutdownBudget < time.Second || config.ShutdownBudget > time.Minute {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidConfig, nil)
	}
	config.TrustJSON = bytes.Clone(config.TrustJSON)
	var trust passwordResetMaintenanceTrust
	if err := decodePasswordResetMaintenanceJSON(config.TrustJSON, &trust, passwordResetMaintenanceEnvelopeLimit, false); err != nil {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidConfig, err)
	}
	if !validPasswordResetMaintenanceTrust(trust) {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidConfig, nil)
	}
	if platform != "windows" {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceUnsupportedPlatform, nil)
	}
	actor, err := subject()
	if err != nil || !validPasswordResetMaintenanceSubject(actor) {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceAuthorityUnavailable, err)
	}
	authority := &passwordResetMaintenanceAuthority{trust: trust, config: config, clock: clock, subject: actor, deadline: time.Now().Add(config.OperationBudget), clientEncoding: passwordResetMaintenanceUTF8}
	if !authority.executorAuthorized(clock.Now().Unix()) {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceForbidden, nil)
	}
	return &PasswordResetOperator{maintenance: authority}, nil
}

func passwordResetMaintenanceUTF8(tx pgx.Tx) bool {
	conn := tx.Conn()
	return conn != nil && conn.PgConn() != nil && conn.PgConn().ParameterStatus("client_encoding") == "UTF8"
}

// ValidateRequeue returns an opaque copy of a canonical and fully authorized request.
// The caller must supply its original operation deadline before reading stdin.
func (operator *PasswordResetOperator) ValidateRequeue(ctx context.Context, envelope []byte) (*PasswordResetRequeueRequest, error) {
	if operator == nil || operator.maintenance == nil || ctx == nil {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, nil)
	}
	authority := operator.maintenance
	deadline, ok := ctx.Deadline()
	if !ok || deadline.After(authority.deadline) {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, nil)
	}
	var wire passwordResetRequeueEnvelope
	if err := decodePasswordResetMaintenanceJSON(envelope, &wire, passwordResetMaintenanceEnvelopeLimit, true); err != nil {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, err)
	}
	payload, err := decodePasswordResetMaintenanceBase64(wire.Payload, passwordResetMaintenancePayloadLimit)
	if err != nil {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, err)
	}
	signature, err := decodePasswordResetMaintenanceBase64(wire.ApprovalSignature, ed25519.SignatureSize)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, err)
	}
	request := &PasswordResetRequeueRequest{owner: authority, payload: bytes.Clone(payload), signature: bytes.Clone(signature), deadline: deadline}
	if _, err := authority.validate(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

func (authority *passwordResetMaintenanceAuthority) validate(ctx context.Context, request *PasswordResetRequeueRequest) (passwordResetRequeuePayload, error) {
	var payload passwordResetRequeuePayload
	if ctx == nil || request == nil || request.owner != authority || request.deadline.IsZero() {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, nil)
	}
	deadline, ok := ctx.Deadline()
	if !ok || !deadline.Equal(request.deadline) || request.deadline.After(authority.deadline) {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, nil)
	}
	if err := ctx.Err(); err != nil {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, err)
	}
	if !time.Now().Before(deadline) {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, context.DeadlineExceeded)
	}
	if err := decodePasswordResetMaintenanceJSON(request.payload, &payload, passwordResetMaintenancePayloadLimit, true); err != nil {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, err)
	}
	if !validPasswordResetRequeuePayload(payload) {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, nil)
	}
	if !authority.authorized(payload, request.payload, request.signature) {
		return payload, passwordResetMaintenanceFailure(PasswordResetMaintenanceForbidden, nil)
	}
	return payload, nil
}

func (authority *passwordResetMaintenanceAuthority) executorAuthorized(now int64) bool {
	for _, executor := range authority.trust.Executors {
		if executor.Subject == authority.subject && executor.NotBefore <= now && now < executor.ExpiresAt {
			return true
		}
	}
	return false
}

func (authority *passwordResetMaintenanceAuthority) authorized(payload passwordResetRequeuePayload, raw, signature []byte) bool {
	now := authority.clock.Now().Unix()
	if payload.DeploymentID != authority.trust.DeploymentID || payload.Epoch != authority.trust.Epoch || payload.ExecutorSubject != authority.subject || !authority.executorAuthorized(now) {
		return false
	}
	if payload.IssuedAt > now || now >= payload.ExpiresAt || payload.ExpiresAt-payload.IssuedAt > int64(authority.config.MaxTTL/time.Second) {
		return false
	}
	for _, key := range authority.trust.ApproverKeys {
		if key.KeyID != payload.ApproverKeyID || key.Subject == authority.subject || key.NotBefore > now || now >= key.ExpiresAt {
			continue
		}
		public, err := decodePasswordResetMaintenanceBase64(key.PublicKey, ed25519.PublicKeySize)
		return err == nil && len(public) == ed25519.PublicKeySize && len(signature) == ed25519.SignatureSize && ed25519.Verify(public, append([]byte(passwordResetMaintenanceDomain), raw...), signature)
	}
	return false
}

func validPasswordResetMaintenanceTrust(trust passwordResetMaintenanceTrust) bool {
	if trust.Version != passwordResetMaintenanceTrustVersion || !validPasswordResetMaintenanceToken(trust.DeploymentID) || !validPasswordResetMaintenanceToken(trust.Epoch) || len(trust.ApproverKeys) == 0 || len(trust.Executors) == 0 {
		return false
	}
	ids, publicKeys, subjects := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, key := range trust.ApproverKeys {
		public, err := decodePasswordResetMaintenanceBase64(key.PublicKey, ed25519.PublicKeySize)
		if !validPasswordResetMaintenanceToken(key.KeyID) || err != nil || len(public) != ed25519.PublicKeySize || !validPasswordResetMaintenanceSubject(key.Subject) || key.Action != passwordResetMaintenanceAction || key.NotBefore < 0 || key.ExpiresAt <= key.NotBefore || ids[key.KeyID] || publicKeys[key.PublicKey] || subjects[key.Subject] {
			return false
		}
		ids[key.KeyID], publicKeys[key.PublicKey], subjects[key.Subject] = true, true, true
	}
	for _, executor := range trust.Executors {
		if !validPasswordResetMaintenanceSubject(executor.Subject) || executor.Action != passwordResetMaintenanceAction || executor.NotBefore < 0 || executor.ExpiresAt <= executor.NotBefore || subjects[executor.Subject] {
			return false
		}
		subjects[executor.Subject] = true
	}
	return true
}

func validPasswordResetMaintenanceToken(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for i, ch := range []byte(value) {
		alphaNum := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
		if !alphaNum && (i == 0 || ch != '.' && ch != '_' && ch != '-') {
			return false
		}
	}
	return true
}

func validPasswordResetMaintenanceSubject(value string) bool {
	if !strings.HasPrefix(value, "windows-sid:S-1-") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(value, "windows-sid:S-1-"), "-")
	if len(parts) < 2 || len(parts) > 16 {
		return false
	}
	for i, part := range parts {
		bits := 32
		if i == 0 {
			bits = 48
		}
		number, err := strconv.ParseUint(part, 10, bits)
		if err != nil || strconv.FormatUint(number, 10) != part {
			return false
		}
	}
	return true
}

func validPasswordResetMaintenanceUUID(value string) bool {
	if len(value) != 36 || value == "00000000-0000-0000-0000-000000000000" {
		return false
	}
	for i, ch := range []byte(value) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				return false
			}
			continue
		}
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}

func validPasswordResetMaintenanceText(value string, limit int) bool {
	if len(value) == 0 || len(value) > limit || strings.TrimSpace(value) == "" || !utf8.ValidString(value) {
		return false
	}
	return !strings.ContainsFunc(value, unicode.IsControl)
}

func validPasswordResetRequeuePayload(payload passwordResetRequeuePayload) bool {
	return payload.Version == passwordResetMaintenancePayloadVersion && payload.Action == passwordResetMaintenanceAction && validPasswordResetMaintenanceToken(payload.DeploymentID) && validPasswordResetMaintenanceToken(payload.Epoch) && validPasswordResetMaintenanceUUID(payload.RequestID) && validPasswordResetMaintenanceUUID(payload.UserID) && validPasswordResetMaintenanceUUID(payload.OperationID) && payload.RequestID != payload.OperationID && validPasswordResetMaintenanceToken(payload.ApproverKeyID) && validPasswordResetMaintenanceSubject(payload.ExecutorSubject) && payload.IssuedAt >= 0 && payload.ExpiresAt > payload.IssuedAt && validPasswordResetMaintenanceText(payload.Reason, 512) && validPasswordResetMaintenanceText(payload.EvidenceRef, 256) && validPasswordResetRequeueSnapshot(payload.Before)
}

func passwordResetMaintenanceOneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func validPasswordResetRequeueSnapshot(s PasswordResetRequeueSnapshot) bool {
	if !passwordResetMaintenanceOneOf(s.UserStatus, "active", "suspended", "deleted") || !passwordResetMaintenanceOneOf(s.CredentialMode, "permanent", "bootstrap", "temporary", "recovery_locked", "recovery_bootstrap") || s.UserAuthGeneration < 0 || s.UserExecutionGeneration < 0 || s.OperationExecutionGeneration < 0 || s.AttemptCount < 0 || !validPasswordResetMaintenanceUUID(s.TemporaryCredentialID) {
		return false
	}
	if !passwordResetMaintenanceOneOf(s.OperationState, "prepared", "dispatching", "commit_unknown", "recovery_required", "converging", "succeeded", "invalidated", "failed") || !passwordResetMaintenanceOneOf(s.DeliveryState, "not_started", "dispatching", "accepted", "rejected", "unknown") || !passwordResetMaintenanceOneOf(s.SettlementState, "pending", "temporary_active", "permanent_successor", "invalidated", "integrity_hold") || !passwordResetMaintenanceOneOf(s.JobStatus, "ready", "leased", "retry_wait", "succeeded", "dead_letter") {
		return false
	}
	for _, id := range []*string{s.UserCurrentOperationID, s.IntegrityHoldOperationID} {
		if id != nil && !validPasswordResetMaintenanceUUID(*id) {
			return false
		}
	}
	for _, value := range []string{s.UserUpdatedAt, s.OperationUpdatedAt, s.NextAttemptAt, s.JobCreatedAt, s.JobUpdatedAt} {
		if !validPasswordResetMaintenanceTime(value) {
			return false
		}
	}
	for _, value := range []*string{s.IntegrityHoldAt, s.LeaseExpiresAt, s.CompletedAt} {
		if value != nil && !validPasswordResetMaintenanceTime(*value) {
			return false
		}
	}
	return true
}

func validPasswordResetMaintenanceTime(value string) bool {
	parsed, err := time.Parse(passwordResetMaintenanceTimeFormat, value)
	return err == nil && parsed.Format(passwordResetMaintenanceTimeFormat) == value
}

func decodePasswordResetMaintenanceBase64(value string, limit int) ([]byte, error) {
	if len(value) > base64.RawURLEncoding.EncodedLen(limit) {
		return nil, errors.New("encoded value exceeds bound")
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(raw) > limit || base64.RawURLEncoding.EncodeToString(raw) != value {
		return nil, errors.New("noncanonical base64url")
	}
	return raw, nil
}

func decodePasswordResetMaintenanceJSON(raw []byte, output any, limit int, canonical bool) error {
	if len(raw) == 0 || len(raw) > limit || !utf8.Valid(raw) {
		return errors.New("JSON size or UTF8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := readPasswordResetMaintenanceJSON(decoder)
	if err != nil {
		return err
	}
	if _, err = decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	if !passwordResetMaintenanceJSONShape(value, reflect.TypeOf(output).Elem()) {
		return errors.New("JSON shape")
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return err
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		return err
	}
	if canonical && !bytes.Equal(encoded, raw) {
		return errors.New("noncanonical JSON")
	}
	return nil
}

func readPasswordResetMaintenanceJSON(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		object := map[string]any{}
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, errors.New("object key")
			}
			if _, duplicate := object[name]; duplicate {
				return nil, errors.New("duplicate JSON key")
			}
			value, err := readPasswordResetMaintenanceJSON(decoder)
			if err != nil {
				return nil, err
			}
			object[name] = value
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return object, nil
	case json.Delim('['):
		values := []any{}
		for decoder.More() {
			value, err := readPasswordResetMaintenanceJSON(decoder)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return values, nil
	default:
		if _, delimiter := token.(json.Delim); delimiter {
			return nil, errors.New("unexpected JSON delimiter")
		}
		return token, nil
	}
}

func passwordResetMaintenanceJSONShape(value any, target reflect.Type) bool {
	if target.Kind() == reflect.Pointer {
		return value == nil || passwordResetMaintenanceJSONShape(value, target.Elem())
	}
	switch target.Kind() {
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok || len(object) != target.NumField() {
			return false
		}
		for i := 0; i < target.NumField(); i++ {
			field := target.Field(i)
			member, present := object[field.Tag.Get("json")]
			if !present || !passwordResetMaintenanceJSONShape(member, field.Type) {
				return false
			}
		}
		return true
	case reflect.Slice:
		values, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range values {
			if !passwordResetMaintenanceJSONShape(item, target.Elem()) {
				return false
			}
		}
		return true
	case reflect.String:
		_, ok := value.(string)
		return ok
	case reflect.Bool:
		_, ok := value.(bool)
		return ok
	case reflect.Int64:
		number, ok := value.(json.Number)
		if !ok {
			return false
		}
		_, err := strconv.ParseInt(string(number), 10, 64)
		return err == nil
	default:
		return false
	}
}

// Requeue consumes one signed request and only requeues its eligible recovery job.
// Commit errors are always unknown; this path never invokes settlement verification.
func (operator *PasswordResetOperator) Requeue(ctx context.Context, request *PasswordResetRequeueRequest, dependencies PasswordResetRequeueDependencies) (result PasswordResetRequeueResult, err error) {
	defer func() {
		if recover() != nil {
			result = PasswordResetRequeueResult{}
			err = finishPasswordResetRequeueResult(ctx, err, errors.New("maintenance invocation panic"))
		}
		err = finishPasswordResetRequeueResult(ctx, err, nil)
		if err != nil {
			result = PasswordResetRequeueResult{}
		}
	}()
	if operator == nil || operator.maintenance == nil {
		return result, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidRequest, nil)
	}
	authority := operator.maintenance
	payload, err := authority.validate(ctx, request)
	if err != nil {
		return result, err
	}
	if dependencies.Transactions == nil || dependencies.Store == nil || dependencies.Permit == nil {
		return result, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidConfig, nil)
	}
	release, err := dependencies.Permit.Acquire(ctx)
	if err != nil {
		return result, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, err)
	}
	defer release()
	return authority.requeueHeld(ctx, request, payload, dependencies)
}

func (authority *passwordResetMaintenanceAuthority) requeueHeld(ctx context.Context, request *PasswordResetRequeueRequest, payload passwordResetRequeuePayload, dependencies PasswordResetRequeueDependencies) (result PasswordResetRequeueResult, err error) {
	tx, err := dependencies.Transactions.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return result, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, err)
	}
	commitAttempted, committed := false, false
	defer func() {
		if recovered := recover(); recovered != nil {
			err = passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, errors.New("maintenance transaction panic"))
		}
		if !committed {
			err = authority.finishRequeue(ctx, request, tx, commitAttempted, err)
		}
		if err != nil {
			result = PasswordResetRequeueResult{}
		}
	}()
	if !authority.clientEncoding(tx) {
		return result, passwordResetMaintenanceFailure(PasswordResetMaintenanceInvalidConfig, nil)
	}
	result, err = authority.requeueLocked(ctx, request, payload, dependencies.Audit, tx)
	if err != nil {
		return result, err
	}
	if _, err = authority.validate(ctx, request); err != nil {
		return result, err
	}
	commitAttempted = true
	if err = tx.Commit(ctx); err != nil {
		return result, passwordResetMaintenanceFailure(PasswordResetMaintenanceOutcomeUnknown, err)
	}
	committed = true
	return result, nil
}

func (authority *passwordResetMaintenanceAuthority) finishRequeue(ctx context.Context, request *PasswordResetRequeueRequest, tx pgx.Tx, attempted bool, primary error) error {
	cutoff := request.deadline.Add(authority.config.ShutdownBudget)
	deadline := time.Now().Add(authority.config.ShutdownBudget)
	if cutoff.Before(deadline) {
		deadline = cutoff
	}
	cleanup, cancel := context.WithDeadline(context.WithoutCancel(ctx), deadline)
	defer cancel()
	rollbackErr := rollbackPasswordResetRequeue(cleanup, tx)
	// A deadline requests cancellation; Rollback must actually return before the
	// permit is released. End retains its monotonic component even if the context
	// timer callback has not yet run. Once observed, overrun stays in the cause.
	overrun := passwordResetRequeueCleanupFailure(deadline, time.Now(), cleanup.Err())
	if attempted || rollbackErr != nil {
		primary = passwordResetMaintenanceFailure(PasswordResetMaintenanceOutcomeUnknown, errors.Join(primary, rollbackErr))
	}
	return finishPasswordResetRequeueResult(ctx, primary, overrun)
}

type passwordResetRequeueCleanupOverrun struct {
	deadline time.Time
	ended    time.Time
	cause    error
}

func (failure *passwordResetRequeueCleanupOverrun) Error() string {
	return "password reset maintenance cleanup exceeded its deadline"
}

func (failure *passwordResetRequeueCleanupOverrun) Unwrap() error { return failure.cause }

func passwordResetRequeueCleanupFailure(deadline, ended time.Time, contextErr error) error {
	if contextErr == nil && !ended.After(deadline) {
		return nil
	}
	return &passwordResetRequeueCleanupOverrun{
		deadline: deadline, ended: ended, cause: errors.Join(contextErr, context.DeadlineExceeded),
	}
}

func finishPasswordResetRequeueResult(ctx context.Context, primary, cleanupErr error) error {
	var operationErr error
	if ctx != nil {
		operationErr = ctx.Err()
		if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) && operationErr == nil {
			operationErr = context.DeadlineExceeded
		}
	}
	if operationErr == nil && cleanupErr == nil {
		return primary
	}
	code := PasswordResetMaintenanceFailed
	var failure *PasswordResetMaintenanceError
	if errors.As(primary, &failure) {
		code = failure.Code
	}
	return passwordResetMaintenanceFailure(code, errors.Join(primary, operationErr, cleanupErr))
}

func rollbackPasswordResetRequeue(ctx context.Context, tx pgx.Tx) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("maintenance rollback panic")
		}
	}()
	return tx.Rollback(ctx)
}

func (authority *passwordResetMaintenanceAuthority) requeueLocked(ctx context.Context, request *PasswordResetRequeueRequest, payload passwordResetRequeuePayload, auditor audit.Store, tx pgx.Tx) (PasswordResetRequeueResult, error) {
	before, err := lockPasswordResetRequeueSnapshot(ctx, tx, payload)
	if err != nil {
		return PasswordResetRequeueResult{}, passwordResetRequeueDatabaseError(err)
	}
	if _, err := authority.validate(ctx, request); err != nil {
		return PasswordResetRequeueResult{}, err
	}
	exists, err := guardPasswordResetRequeueAudit(ctx, tx, payload)
	if err != nil {
		return PasswordResetRequeueResult{}, err
	}
	if exists {
		return authority.replayRequeue(ctx, tx, auditor, request, payload)
	}
	if !reflect.DeepEqual(before, payload.Before) || !eligiblePasswordResetRequeue(before, payload) {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, nil)
	}
	return authority.applyRequeue(ctx, tx, auditor, request, payload, before)
}

func passwordResetRequeueDatabaseError(err error) error {
	var databaseError *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, err)
	}
	return passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, err)
}

func eligiblePasswordResetRequeue(s PasswordResetRequeueSnapshot, payload passwordResetRequeuePayload) bool {
	if s.UserCurrentOperationID == nil || *s.UserCurrentOperationID != payload.OperationID || s.IntegrityHold || s.SettlementState != "pending" || s.JobStatus != "dead_letter" || s.LeaseOwner != nil || s.LeaseExpiresAt != nil || s.CompletedAt != nil || s.AttemptCount == math.MaxInt64 {
		return false
	}
	switch s.OperationState {
	case "prepared":
		return s.DeliveryState == "not_started"
	case "dispatching":
		return s.DeliveryState == "dispatching"
	case "commit_unknown":
		return s.DeliveryState == "dispatching" || s.DeliveryState == "unknown"
	case "recovery_required", "converging":
		return s.DeliveryState == "accepted"
	default:
		return false
	}
}

func lockPasswordResetRequeueSnapshot(ctx context.Context, tx pgx.Tx, payload passwordResetRequeuePayload) (s PasswordResetRequeueSnapshot, err error) {
	err = tx.QueryRow(ctx, passwordResetRequeueLockUserSQL, payload.UserID).Scan(
		&s.UserStatus, &s.UserAuthGeneration, &s.UserExecutionGeneration, &s.UserCurrentOperationID,
		&s.CredentialMode, &s.IntegrityHold, &s.IntegrityHoldOperationID, &s.IntegrityHoldReason,
		&s.IntegrityHoldAt, &s.UserUpdatedAt)
	if err != nil {
		return s, err
	}
	err = tx.QueryRow(ctx, passwordResetRequeueLockJobSQL, payload.OperationID, payload.UserID).Scan(
		&s.OperationState, &s.DeliveryState, &s.SettlementState, &s.OperationExecutionGeneration,
		&s.TemporaryCredentialID, &s.OperationUpdatedAt, &s.JobStatus, &s.AttemptCount,
		&s.NextAttemptAt, &s.LeaseOwner, &s.LeaseExpiresAt, &s.LastFailureCode,
		&s.JobCreatedAt, &s.JobUpdatedAt, &s.CompletedAt)
	return s, err
}

func guardPasswordResetRequeueAudit(ctx context.Context, tx pgx.Tx, payload passwordResetRequeuePayload) (bool, error) {
	var metadataBytes int64
	var bindingOK bool
	err := tx.QueryRow(ctx, passwordResetRequeueAuditGuardSQL, payload.RequestID, payload.ExecutorSubject, payload.OperationID).Scan(&metadataBytes, &bindingOK)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, err)
	}
	if metadataBytes < 0 || metadataBytes > passwordResetMaintenanceMetadataLimit || !bindingOK {
		return false, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, nil)
	}
	return true, nil
}

func (authority *passwordResetMaintenanceAuthority) applyRequeue(ctx context.Context, tx pgx.Tx, auditor audit.Store, request *PasswordResetRequeueRequest, payload passwordResetRequeuePayload, before PasswordResetRequeueSnapshot) (PasswordResetRequeueResult, error) {
	now := authority.clock.Now().UTC().Truncate(time.Microsecond)
	tag, err := tx.Exec(ctx, passwordResetRequeueUpdateSQL, payload.OperationID, now, before.AttemptCount)
	if err != nil {
		return PasswordResetRequeueResult{}, passwordResetRequeueDatabaseError(err)
	}
	if tag.RowsAffected() != 1 {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, nil)
	}
	after, err := lockPasswordResetRequeueSnapshot(ctx, tx, payload)
	if err != nil {
		return PasswordResetRequeueResult{}, passwordResetRequeueDatabaseError(err)
	}
	if !reflect.DeepEqual(after, passwordResetRequeueExpectedAfter(before, now)) {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, nil)
	}
	receipt := passwordResetRequeueReceipt(payload)
	metadata, err := authority.requeueMetadata(request, payload, after, receipt)
	if err != nil {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, err)
	}
	role, executor := passwordResetMaintenanceRole, authority.subject
	event := audit.Event{ID: payload.RequestID, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &executor, ActorRole: &role, Action: passwordResetMaintenanceAuditAction, TargetType: "password_reset_operation", TargetID: payload.OperationID, Outcome: "succeeded", TraceID: payload.RequestID, Metadata: metadata}
	if err := auditor.Append(ctx, tx, event); err != nil {
		return PasswordResetRequeueResult{}, passwordResetRequeueDatabaseError(err)
	}
	exists, err := guardPasswordResetRequeueAudit(ctx, tx, payload)
	if err != nil {
		return PasswordResetRequeueResult{}, err
	}
	if !exists {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceFailed, nil)
	}
	return PasswordResetRequeueResult{Disposition: PasswordResetRequeueApplied, Receipt: receipt, After: after}, nil
}

func passwordResetRequeueExpectedAfter(before PasswordResetRequeueSnapshot, now time.Time) PasswordResetRequeueSnapshot {
	before.JobStatus = "ready"
	nextAt, _ := time.Parse(passwordResetMaintenanceTimeFormat, before.NextAttemptAt)
	updatedAt, _ := time.Parse(passwordResetMaintenanceTimeFormat, before.JobUpdatedAt)
	if now.After(nextAt) {
		before.NextAttemptAt = now.Format(passwordResetMaintenanceTimeFormat)
	}
	if now.After(updatedAt) {
		before.JobUpdatedAt = now.Format(passwordResetMaintenanceTimeFormat)
	}
	return before
}

func passwordResetRequeueReceipt(payload passwordResetRequeuePayload) PasswordResetRequeueReceipt {
	return PasswordResetRequeueReceipt{Version: passwordResetMaintenanceResultVersion, RequestID: payload.RequestID, Outcome: "applied", OperationID: payload.OperationID, JobStatus: "ready", AuditID: payload.RequestID}
}

func (authority *passwordResetMaintenanceAuthority) requeueMetadata(request *PasswordResetRequeueRequest, payload passwordResetRequeuePayload, after PasswordResetRequeueSnapshot, receipt PasswordResetRequeueReceipt) (map[string]any, error) {
	afterBytes, err := json.Marshal(after)
	if err != nil || len(afterBytes) > passwordResetMaintenancePayloadLimit {
		return nil, errors.New("after exceeds bound")
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil || len(receiptBytes) > 1024 {
		return nil, errors.New("receipt exceeds bound")
	}
	subject := ""
	for _, key := range authority.trust.ApproverKeys {
		if key.KeyID == payload.ApproverKeyID {
			subject = key.Subject
		}
	}
	metadata := map[string]any{
		"schema_version": passwordResetMaintenanceAuditVersion, "deployment_id": payload.DeploymentID,
		"epoch": payload.Epoch, "approver_key_id": payload.ApproverKeyID, "approver_subject": subject,
		"executor_subject": authority.subject, "payload_b64": base64.RawURLEncoding.EncodeToString(request.payload),
		"approval_signature": base64.RawURLEncoding.EncodeToString(request.signature), "payload_sha256": passwordResetMaintenanceSHA(request.payload),
		"after_b64": base64.RawURLEncoding.EncodeToString(afterBytes), "after_sha256": passwordResetMaintenanceSHA(afterBytes),
		"result_b64": base64.RawURLEncoding.EncodeToString(receiptBytes), "result_sha256": passwordResetMaintenanceSHA(receiptBytes),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil || len(encoded) > passwordResetMaintenanceMetadataLimit {
		return nil, errors.New("metadata exceeds bound")
	}
	return metadata, nil
}

func passwordResetMaintenanceSHA(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

type passwordResetRequeueTextQuery struct{ tx pgx.Tx }

func (query passwordResetRequeueTextQuery) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	options := []any{pgx.QueryResultFormatsByOID{pgtype.JSONBOID: pgx.TextFormatCode}}
	return query.tx.QueryRow(ctx, sql, append(options, args...)...)
}

func (authority *passwordResetMaintenanceAuthority) replayRequeue(ctx context.Context, tx pgx.Tx, auditor audit.Store, request *PasswordResetRequeueRequest, payload passwordResetRequeuePayload) (PasswordResetRequeueResult, error) {
	event, err := auditor.Get(ctx, passwordResetRequeueTextQuery{tx: tx}, payload.RequestID)
	if err != nil {
		return PasswordResetRequeueResult{}, passwordResetRequeueDatabaseError(err)
	}
	if !passwordResetRequeueEventBinding(event, payload) || !validPasswordResetMaintenanceTime(event.OccurredAt.UTC().Format(passwordResetMaintenanceTimeFormat)) || !event.OccurredAt.Equal(event.OccurredAt.Truncate(time.Microsecond)) || event.OccurredAt.Unix() < payload.IssuedAt || event.OccurredAt.Unix() >= payload.ExpiresAt {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, nil)
	}
	var after PasswordResetRequeueSnapshot
	var receipt PasswordResetRequeueReceipt
	afterString, afterOK := event.Metadata["after_b64"].(string)
	resultString, resultOK := event.Metadata["result_b64"].(string)
	afterBytes, afterErr := decodePasswordResetMaintenanceBase64(afterString, passwordResetMaintenancePayloadLimit)
	receiptBytes, receiptErr := decodePasswordResetMaintenanceBase64(resultString, 1024)
	if !afterOK || !resultOK || afterErr != nil || receiptErr != nil || decodePasswordResetMaintenanceJSON(afterBytes, &after, passwordResetMaintenancePayloadLimit, true) != nil || decodePasswordResetMaintenanceJSON(receiptBytes, &receipt, 1024, true) != nil {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, nil)
	}
	if !reflect.DeepEqual(after, passwordResetRequeueExpectedAfter(payload.Before, event.OccurredAt.UTC())) || receipt != passwordResetRequeueReceipt(payload) {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, nil)
	}
	expected, err := authority.requeueMetadata(request, payload, after, receipt)
	if err != nil || !reflect.DeepEqual(event.Metadata, expected) {
		return PasswordResetRequeueResult{}, passwordResetMaintenanceFailure(PasswordResetMaintenanceStateConflict, err)
	}
	return PasswordResetRequeueResult{Disposition: PasswordResetRequeueAlreadyApplied, Receipt: receipt, After: after}, nil
}

func passwordResetRequeueEventBinding(event audit.Event, payload passwordResetRequeuePayload) bool {
	return event.ID == payload.RequestID && event.ActorKind == audit.ActorSystem && event.ActorUserID == nil && event.ActorSystem != nil && *event.ActorSystem == payload.ExecutorSubject && event.ActorRole != nil && *event.ActorRole == passwordResetMaintenanceRole && event.Action == passwordResetMaintenanceAuditAction && event.TargetType == "password_reset_operation" && event.TargetID == payload.OperationID && event.Outcome == "succeeded" && event.TraceID == payload.RequestID && event.ErrorCode == nil && event.SourceFingerprint == nil && len(event.Metadata) == 13
}

const passwordResetRequeueLockUserSQL = `SELECT status, auth_generation, password_reset_execution_generation,
 password_reset_current_operation_id::text, credential_mode, auth_integrity_hold,
 auth_integrity_hold_operation_id::text, auth_integrity_hold_reason,
 to_char(auth_integrity_hold_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 to_char(updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 FROM users WHERE id=$1 FOR UPDATE`

const passwordResetRequeueLockJobSQL = `SELECT o.state, o.delivery_state, o.settlement_state, o.execution_generation,
 o.temporary_credential_id::text, to_char(o.updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 j.status, j.attempt_count, to_char(j.next_attempt_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 j.lease_owner, to_char(j.lease_expires_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), j.last_failure_code,
 to_char(j.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 to_char(j.updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 to_char(j.completed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 FROM password_reset_operations o JOIN password_reset_recovery_jobs j ON j.operation_id=o.operation_id
 WHERE o.operation_id=$1 AND o.user_id=$2 FOR UPDATE OF o,j`

const passwordResetRequeueUpdateSQL = `UPDATE password_reset_recovery_jobs
 SET status='ready', next_attempt_at=GREATEST(next_attempt_at,$2), updated_at=GREATEST(updated_at,$2)
 WHERE operation_id=$1 AND status='dead_letter' AND attempt_count=$3
 AND lease_owner IS NULL AND lease_expires_at IS NULL AND completed_at IS NULL`

const passwordResetRequeueAuditGuardSQL = `SELECT octet_length(convert_to(metadata::text,'UTF8'))::bigint,
 (actor_kind IS NOT DISTINCT FROM 'system' AND actor_user_id IS NULL
 AND actor_system IS NOT DISTINCT FROM $2 AND actor_role IS NOT DISTINCT FROM 'password_reset_requeue_executor'
 AND action IS NOT DISTINCT FROM 'identity.password_reset.requeued'
 AND target_type IS NOT DISTINCT FROM 'password_reset_operation' AND target_id IS NOT DISTINCT FROM $3
 AND outcome IS NOT DISTINCT FROM 'succeeded' AND trace_id IS NOT DISTINCT FROM $1::uuid::text
 AND error_code IS NULL AND source_fingerprint IS NULL) IS TRUE
 FROM audit_events WHERE id=$1::uuid`

// claimRecoveryResetLease is caller-owned: the actor/target/run/review locks
// have already been acquired; no top-level operator or nested Begin is called.
func claimRecoveryResetLease(ctx context.Context, tx pgx.Tx, s recoveryRestoredSettlement, id, user string) (PasswordResetRecoveryLease, error) {
	var status PasswordResetRecoveryJobStatus
	var attempt int64
	var due time.Time
	var expires *time.Time
	err := tx.QueryRow(ctx, `SELECT status,attempt_count,next_attempt_at,lease_expires_at FROM password_reset_recovery_jobs WHERE operation_id=$1 FOR UPDATE`, id).Scan(&status, &attempt, &due, &expires)
	if err != nil {
		return PasswordResetRecoveryLease{}, err
	}
	now := s.execution.d.clock.Now().UTC().Truncate(time.Microsecond)
	if !recoveryRestoreLeaseDue(status, attempt, due, expires, now, s.config.MaxRetries) {
		return PasswordResetRecoveryLease{}, errRecoveryRestorePending
	}
	end := now.Add(s.config.LeaseDuration)
	err = tx.QueryRow(ctx, passwordResetRecoveryClaimSQL, id, user, s.workerID, now, end).Scan(&attempt)
	return PasswordResetRecoveryLease{OperationID: id, UserID: user, Owner: s.workerID, AttemptCount: attempt, ExpiresAt: end}, err
}

func recoveryRestoreLeaseDue(status PasswordResetRecoveryJobStatus, attempt int64, due time.Time, expires *time.Time, now time.Time, max int64) bool {
	if attempt < 0 || attempt >= max || attempt == math.MaxInt64 || max <= 0 {
		return false
	}
	return ((status == PasswordResetRecoveryJobReady || status == PasswordResetRecoveryJobRetryWait) && !due.After(now)) ||
		(status == PasswordResetRecoveryJobLeased && expires != nil && !expires.After(now))
}

func settleRecoveryRestoredReset(ctx context.Context, tx pgx.Tx, s recoveryRestoredSettlement, lease PasswordResetRecoveryLease, o PasswordResetOperation, a RecoveryReviewApproval, u User, p recovery.PermitBinding) error {
	now := s.execution.d.clock.Now().UTC().Truncate(time.Microsecond)
	if o.Recovery == nil || *o.Recovery != *recoveryRestoreMetadata(a) || o.UserID != u.ID || o.TargetAuthGeneration != u.AuthGeneration+1 || o.BaseAuthGeneration != u.AuthGeneration || u.PasswordResetCurrentOperationID == nil || *u.PasswordResetCurrentOperationID != o.ID || u.AuthIntegrityHold || now.Before(o.CreatedAt) || now.Before(u.UpdatedAt) {
		return ErrPasswordResetStateConflict
	}
	view, err := lockPasswordResetSettlementOperation(ctx, tx, lease, now)
	if err != nil {
		return err
	}
	if view.delivery != PasswordResetDeliveryAccepted {
		if err = settleUnacceptedPasswordReset(ctx, tx, lease, view, now); err != nil {
			return err
		}
		if err = appendRecoveryResetSettlementAudit(ctx, tx, s, lease, now); err != nil {
			return err
		}
		return completePasswordResetRecoveryJob(ctx, tx, lease, now)
	}
	return settleRecoveryRestoredAccepted(ctx, tx, s, lease, o, a, u, p, view, now)
}

func validateRecoveryResetCredential(ctx context.Context, tx pgx.Tx, o PasswordResetOperation, now time.Time) error {
	var valid bool
	err := tx.QueryRow(ctx, `SELECT user_id=$2 AND purpose='password_reset' AND state='pending_delivery'
 AND source_operation_id=$3 AND auth_generation=$4 AND expires_at>$5 AND issued_at<=$5
 AND superseded_at IS NULL AND consumed_at IS NULL AND invalidated_at IS NULL
 FROM temporary_credentials WHERE id=$1 FOR UPDATE`, o.TemporaryCredentialID, o.UserID, o.ID, o.TargetAuthGeneration, now).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return ErrPasswordResetStateConflict
	}
	return nil
}

func appendRecoveryResetSettlementAudit(ctx context.Context, tx pgx.Tx, s recoveryRestoredSettlement, lease PasswordResetRecoveryLease, now time.Time) error {
	id, err := s.execution.d.ids.NewID()
	if err != nil {
		return err
	}
	trace, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return recovery.ErrAdmissionBinding
	}
	// Keep invalidated operations consumable by the original failed-reset
	// proof. This is the durable settlement actor, not a worker Start.
	system := "password-reset-worker"
	return s.execution.d.audit.Append(ctx, tx, audit.Event{ID: id, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &system,
		Action: "identity.password_reset.recovered", TargetType: "password_reset_operation", TargetID: lease.OperationID,
		Outcome: "succeeded", TraceID: trace, Metadata: map[string]any{"attempt_count": lease.AttemptCount}})
}

func appendRecoveryRestoredReviewAudit(ctx context.Context, tx pgx.Tx, s recoveryRestoredSettlement, a RecoveryReviewApproval, p recovery.PermitBinding, now time.Time) error {
	id, err := s.execution.d.ids.NewID()
	if err != nil {
		return err
	}
	trace, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return recovery.ErrAdmissionBinding
	}
	role := "admin"
	actor := a.ActorID
	return s.execution.d.audit.Append(ctx, tx, audit.Event{ID: id, OccurredAt: now, ActorKind: audit.ActorUser, ActorUserID: &actor, ActorRole: &role,
		Action: recoveryReviewAction, TargetType: "user", TargetID: a.TargetID, Outcome: "succeeded", TraceID: trace,
		Metadata: map[string]any{"decision": "restored", "evidence_ref": a.EvidenceRef, "evidence_digest": a.EvidenceDigest,
			"actor_id": a.ActorID, "run_id": p.RunID, "backup_ref": p.BackupRef, "installation_id": p.InstallationID}})
}

const recoveryRestoreSetUserSQL = `UPDATE users SET auth_generation=$1,credential_mode='temporary',role=$6,status=$7,
 recovery_required=false,updated_at=$2 WHERE id=$3 AND NOT is_initial_admin AND status='suspended' AND deleted_at IS NULL
 AND recovery_required AND credential_mode='recovery_locked' AND auth_integrity_hold=false AND auth_generation=$4
 AND password_reset_current_operation_id=$5 AND permanent_password_digest IS NULL
 AND permanent_password_source_operation_id IS NULL AND permanent_password_source_credential_id IS NULL`

const recoveryRestoreReviewSQL = `UPDATE recovery_account_reviews SET decision='restored',evidence_ref=$4,
 reviewed_by=$5,reviewed_at=$6,credential_reset_at=$6 WHERE recovery_run_id=$1 AND user_id=$2
 AND decision='pending' AND backup_status<>'deleted' AND backup_deleted_at IS NULL AND restoration_operation_id=$3
 AND reviewed_by IS NULL AND reviewed_at IS NULL AND credential_reset_at IS NULL`

func settleRecoveryRestoredAccepted(ctx context.Context, tx pgx.Tx, s recoveryRestoredSettlement, lease PasswordResetRecoveryLease, o PasswordResetOperation, a RecoveryReviewApproval, u User, p recovery.PermitBinding, view passwordResetSettlementOperation, now time.Time) error {
	var err error
	if o.State != PasswordResetOperationRecoveryRequired && o.State != PasswordResetOperationConverging {
		return ErrPasswordResetStateConflict
	}
	if err = validateRecoveryResetCredential(ctx, tx, o, now); err != nil {
		return err
	}
	if o.State == PasswordResetOperationRecoveryRequired {
		if err = requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementConvergingSQL, now, o.ID)); err != nil {
			return err
		}
	}
	if err = requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementActivateSQL, o.TemporaryCredentialID, u.ID)); err != nil {
		return err
	}
	if err = requirePasswordResetOneRow(tx.Exec(ctx, recoveryRestoreSetUserSQL, o.TargetAuthGeneration, now, u.ID, o.BaseAuthGeneration, o.ID, a.Role, a.Status)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, passwordResetSettlementRevokeSessionsSQL, now, u.ID); err != nil {
		return err
	}
	if err = requirePasswordResetOneRow(tx.Exec(ctx, passwordResetSettlementSucceededSQL, o.TargetAuthGeneration, now, o.ID)); err != nil {
		return err
	}
	if err = completePasswordResetIdempotency(ctx, tx, view, o.ID, true, now); err != nil {
		return err
	}
	if err = requirePasswordResetOneRow(tx.Exec(ctx, recoveryRestoreReviewSQL, p.RunID, u.ID, o.ID, a.EvidenceRef, a.ActorID, now)); err != nil {
		return err
	}
	if err = appendRecoveryResetSettlementAudit(ctx, tx, s, lease, now); err != nil {
		return err
	}
	if err = appendRecoveryRestoredReviewAudit(ctx, tx, s, a, p, now); err != nil {
		return err
	}
	return completePasswordResetRecoveryJob(ctx, tx, lease, now)
}

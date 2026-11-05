package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrPasswordResetWorkerTerminal = errors.New("password reset worker terminal failure")

type PasswordResetWorkerConfig struct {
	LeaseDuration    time.Duration
	RetryBackoff     time.Duration
	MaxRetries       int64
	ClaimBatchLimit  int
	ConnectionBudget int
}

type PasswordResetRecoveryCandidate struct {
	OperationID string
	UserID      string
}

type passwordResetWorkerPool interface {
	passwordResetTransactionStarter
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type PasswordResetWorkerDependencies struct {
	// AcquireWork is distinct from the database connection permit.
	AcquireWork func(context.Context) (context.Context, func(), error)
	Pool        passwordResetWorkerPool
	Store       *PasswordResetStore
	Operator    *PasswordResetOperator
	Permit      *PasswordResetConnectionPermit
	WorkerID    string
	Clock       LoginClock
	Ticks       <-chan time.Time
}

type passwordResetWorkerBackend interface {
	Scan(context.Context, time.Time, int) ([]PasswordResetRecoveryCandidate, error)
	Claim(context.Context, PasswordResetRecoveryCandidate, string, time.Time, time.Duration) (PasswordResetRecoveryLease, bool, error)
	Renew(context.Context, PasswordResetRecoveryLease, time.Time, time.Duration) (bool, error)
	Fail(context.Context, PasswordResetRecoveryLease, time.Time, time.Duration, bool, string) (bool, error)
}

type PasswordResetRecoveryWorker struct {
	dependencies PasswordResetWorkerDependencies
	config       PasswordResetWorkerConfig
	backend      passwordResetWorkerBackend
	stop         chan struct{}
	done         chan struct{}
	loopStarted  chan struct{}
	doneOnce     sync.Once
	stopOnce     sync.Once
	startOnce    sync.Once
	mu           sync.Mutex
	started      bool
	runCancel    context.CancelFunc
	err          error
}

func NewPasswordResetRecoveryWorker(dependencies PasswordResetWorkerDependencies, config PasswordResetWorkerConfig) (*PasswordResetRecoveryWorker, error) {
	if dependencies.Pool == nil || dependencies.Store == nil {
		return nil, errors.New("password reset worker persistence dependencies are incomplete")
	}
	backend := passwordResetPostgresBackend{pool: dependencies.Pool, store: dependencies.Store, clock: dependencies.Clock}
	return newPasswordResetWorker(dependencies, config, backend)
}

func newPasswordResetWorkerForTest(dependencies PasswordResetWorkerDependencies, config PasswordResetWorkerConfig, backend passwordResetWorkerBackend) (*PasswordResetRecoveryWorker, error) {
	return newPasswordResetWorker(dependencies, config, backend)
}

func newPasswordResetWorker(dependencies PasswordResetWorkerDependencies, config PasswordResetWorkerConfig, backend passwordResetWorkerBackend) (*PasswordResetRecoveryWorker, error) {
	if dependencies.Operator == nil || dependencies.Permit == nil || dependencies.Clock == nil || strings.TrimSpace(dependencies.WorkerID) == "" || backend == nil {
		return nil, errors.New("password reset worker dependencies are incomplete")
	}
	if config.LeaseDuration <= 0 || config.RetryBackoff <= 0 || config.MaxRetries <= 0 || config.ClaimBatchLimit <= 0 || config.ConnectionBudget <= 0 || config.ConnectionBudget != dependencies.Permit.Stats().Budget {
		return nil, errors.New("password reset worker configuration is invalid")
	}
	return &PasswordResetRecoveryWorker{dependencies: dependencies, config: config, backend: backend, stop: make(chan struct{}), done: make(chan struct{}), loopStarted: make(chan struct{})}, nil
}

func (worker *PasswordResetRecoveryWorker) Reconcile(ctx context.Context) error {
	if worker == nil || ctx == nil {
		return errors.New("password reset worker context is required")
	}
	if worker.stopping() {
		return nil
	}
	now := worker.dependencies.Clock.Now().UTC().Truncate(time.Microsecond)
	var candidates []PasswordResetRecoveryCandidate
	if err := worker.withPermit(ctx, func() error {
		var err error
		candidates, err = worker.scan(ctx, now)
		return err
	}); err != nil {
		return fmt.Errorf("scan password reset recovery jobs: %w", err)
	}
	for _, candidate := range candidates {
		if worker.stopping() {
			return nil
		}
		if err := worker.reconcileCandidate(ctx, candidate); err != nil {
			return err
		}
	}
	return nil
}

func (worker *PasswordResetRecoveryWorker) settle(ctx context.Context, lease PasswordResetRecoveryLease) error {
	return worker.withPermit(ctx, func() error {
		err := worker.dependencies.Operator.executeHeld(ctx, lease)
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrPasswordResetSettlementCommitUnknown) {
			return nil
		}
		deadLetter := lease.AttemptCount >= worker.config.MaxRetries
		now := worker.dependencies.Clock.Now().UTC().Truncate(time.Microsecond)
		failureCode := passwordResetRecoveryFailureCode(err)
		failed, failErr := worker.backend.Fail(ctx, lease, now, worker.config.RetryBackoff, deadLetter, failureCode)
		if failErr != nil {
			return errors.Join(fmt.Errorf("settle password reset recovery job: %w", err), fmt.Errorf("persist password reset recovery failure: %w", failErr))
		}
		if !failed {
			return nil
		}
		return nil
	})
}

func (worker *PasswordResetRecoveryWorker) withPermit(ctx context.Context, operation func() error) error {
	release, err := worker.dependencies.Permit.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return operation()
}

func (worker *PasswordResetRecoveryWorker) Start(ctx context.Context) {
	if worker == nil {
		return
	}
	worker.startOnce.Do(func() {
		runCtx := ctx
		var cancel context.CancelFunc
		if ctx != nil {
			runCtx, cancel = context.WithCancel(ctx)
		}
		worker.mu.Lock()
		if worker.stopping() {
			worker.mu.Unlock()
			if cancel != nil {
				cancel()
			}
			worker.doneOnce.Do(func() { close(worker.done) })
			return
		}
		worker.started = true
		worker.runCancel = cancel
		stopping := worker.stopping()
		worker.mu.Unlock()
		if stopping {
			cancel()
		}
		go worker.run(runCtx)
	})
}

func (worker *PasswordResetRecoveryWorker) Stop() {
	if worker == nil {
		return
	}
	worker.stopOnce.Do(func() {
		close(worker.stop)
		worker.cancelRun()
		worker.mu.Lock()
		started := worker.started
		worker.mu.Unlock()
		if !started {
			worker.doneOnce.Do(func() { close(worker.done) })
		}
	})
}

func (worker *PasswordResetRecoveryWorker) Wait(ctx context.Context) error {
	if worker == nil || ctx == nil {
		return errors.New("password reset worker wait context is required")
	}
	select {
	case <-worker.done:
		return worker.Err()
	case <-ctx.Done():
		worker.cancelRun()
		<-worker.done
		return errors.Join(worker.Err(), ctx.Err())
	}
}

func (worker *PasswordResetRecoveryWorker) Done() <-chan struct{} {
	if worker == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return worker.done
}

func (worker *PasswordResetRecoveryWorker) Err() error {
	if worker == nil {
		return errors.New("password reset worker is required")
	}
	worker.mu.Lock()
	defer worker.mu.Unlock()
	return worker.err
}

func (worker *PasswordResetRecoveryWorker) run(parent context.Context) {
	defer worker.doneOnce.Do(func() { close(worker.done) })
	if parent == nil {
		worker.setError(errors.New("password reset worker context is required"))
		return
	}
	ticks := worker.dependencies.Ticks
	var ticker *time.Ticker
	if ticks == nil {
		ticker = time.NewTicker(worker.config.RetryBackoff)
		defer ticker.Stop()
		ticks = ticker.C
	}
	close(worker.loopStarted)
	for {
		select {
		case <-worker.stop:
			return
		case <-parent.Done():
			if worker.stopping() {
				return
			}
			worker.setError(fmt.Errorf("password reset worker context stopped: %w", parent.Err()))
			return
		case <-ticks:
		}
		if err := worker.Reconcile(parent); err != nil {
			if worker.stopping() && errors.Is(err, context.Canceled) {
				return
			}
			worker.setError(errors.Join(ErrPasswordResetWorkerTerminal, err))
			return
		}
	}
}

func (worker *PasswordResetRecoveryWorker) setError(err error) {
	worker.mu.Lock()
	worker.err = err
	worker.mu.Unlock()
}

func (worker *PasswordResetRecoveryWorker) cancelRun() {
	worker.mu.Lock()
	cancel := worker.runCancel
	worker.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (worker *PasswordResetRecoveryWorker) stopping() bool {
	select {
	case <-worker.stop:
		return true
	default:
		return false
	}
}

func passwordResetRecoveryFailureCode(error) string { return "settlement_unavailable" }

type passwordResetPostgresBackend struct {
	pool  passwordResetWorkerPool
	store *PasswordResetStore
	clock LoginClock
}

func (backend passwordResetPostgresBackend) Scan(ctx context.Context, now time.Time, limit int) ([]PasswordResetRecoveryCandidate, error) {
	rows, err := backend.pool.Query(ctx, passwordResetRecoveryScanSQL, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	candidates := make([]PasswordResetRecoveryCandidate, 0, limit)
	for rows.Next() {
		var candidate PasswordResetRecoveryCandidate
		if err := rows.Scan(&candidate.OperationID, &candidate.UserID); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (backend passwordResetPostgresBackend) Claim(ctx context.Context, candidate PasswordResetRecoveryCandidate, owner string, now time.Time, duration time.Duration) (lease PasswordResetRecoveryLease, claimed bool, err error) {
	tx, err := backend.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return lease, false, err
	}
	defer func() { err = finishPasswordResetTransaction(ctx, tx, err) }()
	if err = lockPasswordResetRecoveryUser(ctx, tx, candidate.UserID); err != nil {
		return lease, false, err
	}
	now = backend.clock.Now().UTC().Truncate(time.Microsecond)
	expires := now.Add(duration)
	var attempt int64
	err = tx.QueryRow(ctx, passwordResetRecoveryClaimSQL, candidate.OperationID, candidate.UserID, owner, now, expires).Scan(&attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return lease, false, nil
	}
	if err != nil {
		return lease, false, err
	}
	return PasswordResetRecoveryLease{OperationID: candidate.OperationID, UserID: candidate.UserID, Owner: owner, AttemptCount: attempt, ExpiresAt: expires}, true, nil
}

func (backend passwordResetPostgresBackend) Renew(ctx context.Context, lease PasswordResetRecoveryLease, now time.Time, duration time.Duration) (renewed bool, err error) {
	tx, err := backend.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { err = finishPasswordResetTransaction(ctx, tx, err) }()
	if err = lockPasswordResetRecoveryUser(ctx, tx, lease.UserID); err != nil {
		return false, err
	}
	now = backend.clock.Now().UTC().Truncate(time.Microsecond)
	tag, err := tx.Exec(ctx, passwordResetRecoveryRenewSQL, now.Add(duration), now, lease.OperationID, lease.Owner, lease.AttemptCount)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (backend passwordResetPostgresBackend) Fail(ctx context.Context, lease PasswordResetRecoveryLease, now time.Time, backoff time.Duration, deadLetter bool, failureCode string) (failed bool, err error) {
	tx, err := backend.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { err = finishPasswordResetTransaction(ctx, tx, err) }()
	if err = lockPasswordResetRecoveryUser(ctx, tx, lease.UserID); err != nil {
		return false, err
	}
	now = backend.clock.Now().UTC().Truncate(time.Microsecond)
	status := PasswordResetRecoveryJobRetryWait
	next := now.Add(backoff)
	if deadLetter {
		status = PasswordResetRecoveryJobDeadLetter
		next = now
	}
	tag, err := tx.Exec(ctx, passwordResetRecoveryFailSQL, status, next, failureCode, now, lease.OperationID, lease.Owner, lease.AttemptCount)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func lockPasswordResetRecoveryUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var locked string
	if err := tx.QueryRow(ctx, passwordResetRecoveryLockUserSQL, userID).Scan(&locked); err != nil {
		return err
	}
	if locked != userID {
		return ErrPasswordResetStateConflict
	}
	return nil
}

const passwordResetRecoveryScanSQL = `SELECT j.operation_id,o.user_id FROM password_reset_recovery_jobs j JOIN password_reset_operations o ON o.operation_id=j.operation_id WHERE ((j.status IN ('ready','retry_wait') AND j.next_attempt_at<=$1) OR (j.status='leased' AND j.lease_expires_at<=$1)) ORDER BY j.next_attempt_at,j.operation_id LIMIT $2`
const passwordResetRecoveryLockUserSQL = `SELECT id FROM users WHERE id=$1 FOR UPDATE`
const passwordResetRecoveryClaimSQL = `UPDATE password_reset_recovery_jobs j SET status='leased',attempt_count=j.attempt_count+1,lease_owner=$3,lease_expires_at=$5,updated_at=$4 FROM password_reset_operations o WHERE j.operation_id=$1 AND o.operation_id=j.operation_id AND o.user_id=$2 AND ((j.status IN ('ready','retry_wait') AND j.next_attempt_at<=$4) OR (j.status='leased' AND j.lease_expires_at<=$4)) RETURNING j.attempt_count`
const passwordResetRecoveryRenewSQL = `UPDATE password_reset_recovery_jobs SET lease_expires_at=$1,updated_at=$2 WHERE operation_id=$3 AND status='leased' AND lease_owner=$4 AND attempt_count=$5 AND lease_expires_at>$2`
const passwordResetRecoveryFailSQL = `UPDATE password_reset_recovery_jobs SET status=$1,next_attempt_at=$2,last_failure_code=$3,lease_owner=NULL,lease_expires_at=NULL,updated_at=$4 WHERE operation_id=$5 AND status='leased' AND lease_owner=$6 AND attempt_count=$7 AND lease_expires_at>$4`

// Started observes entry into the actual polling goroutine.
func (worker *PasswordResetRecoveryWorker) Started() <-chan struct{} { return worker.loopStarted }

func (worker *PasswordResetRecoveryWorker) acquireWork(ctx context.Context) (context.Context, func(), error) {
	if worker.dependencies.AcquireWork == nil {
		return ctx, func() {}, nil
	}
	return worker.dependencies.AcquireWork(ctx)
}
func (worker *PasswordResetRecoveryWorker) scan(ctx context.Context, now time.Time) ([]PasswordResetRecoveryCandidate, error) {
	work, release, err := worker.acquireWork(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return worker.backend.Scan(work, now, worker.config.ClaimBatchLimit)
}
func (worker *PasswordResetRecoveryWorker) reconcileCandidate(ctx context.Context, candidate PasswordResetRecoveryCandidate) error {
	work, release, err := worker.acquireWork(ctx)
	if err != nil {
		return err
	}
	defer release()
	ctx = work
	var lease PasswordResetRecoveryLease
	claimed := false
	claimAt := worker.dependencies.Clock.Now().UTC().Truncate(time.Microsecond)
	if err := worker.withPermit(ctx, func() error {
		var err error
		lease, claimed, err = worker.backend.Claim(ctx, candidate, worker.dependencies.WorkerID, claimAt, worker.config.LeaseDuration)
		return err
	}); err != nil {
		return fmt.Errorf("claim password reset recovery job: %w", err)
	}
	if !claimed {
		return nil
	}
	renewed := false
	renewAt := worker.dependencies.Clock.Now().UTC().Truncate(time.Microsecond)
	if err := worker.withPermit(ctx, func() error {
		var err error
		renewed, err = worker.backend.Renew(ctx, lease, renewAt, worker.config.LeaseDuration)
		return err
	}); err != nil {
		return fmt.Errorf("renew password reset recovery job: %w", err)
	}
	if !renewed {
		return nil
	}
	if err := worker.settle(ctx, lease); err != nil {
		return err
	}
	return nil
}

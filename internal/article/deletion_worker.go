package article

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"han/internal/platform"
)

var ErrDeletionWorkerTerminal = errors.New("article deletion worker terminal failure")

const (
	deletionWorkerCategoryTransient = "transient"
	deletionWorkerCategoryNoop      = "noop"
	deletionWorkerCategoryTerminal  = "terminal"

	deletionWorkerReasonDatabaseUnavailable    = "database_unavailable"
	deletionWorkerReasonJournalUnavailable     = "journal_unavailable"
	deletionWorkerReasonTraceIDUnavailable     = "trace_id_unavailable"
	deletionWorkerReasonLeaseCompetition       = "lease_competition"
	deletionWorkerReasonCompletedNotClaimed    = "completed_not_claimed"
	deletionWorkerReasonConvergenceUnavailable = "convergence_unavailable"
	deletionWorkerReasonJournalCorrupt         = "journal_corrupt"
	deletionWorkerReasonTombstoneNotFound      = "tombstone_not_found"
	deletionWorkerReasonArticleMismatch        = "article_mismatch"
	deletionWorkerReasonInvariantFailure       = "invariant_failure"
)

type deletionWorkerObservation struct {
	Category string
	Reason   string
}

type DeletionOperationDiscoverer interface {
	DiscoverDeletionOperations(context.Context, time.Time, int) ([]string, error)
}

type DeletionOperationConverger interface {
	ConvergeDeletionOperation(context.Context, string, string) error
}

type DeletionWorkerDependencies struct {
	// AcquireWork is the root-owned admission adapter; nil preserves legacy consumers.
	AcquireWork func(context.Context) (context.Context, func(), error)
	Discoverer  DeletionOperationDiscoverer
	Converger   DeletionOperationConverger
	Now         func() time.Time
	NewTraceID  func() (string, error)
	Sleep       func(context.Context, time.Duration) error
	Jitter      func(time.Duration) time.Duration
	Ticks       <-chan time.Time
}

type DeletionWorkerConfig struct {
	PollInterval      time.Duration
	BatchSize         int
	OperationBudget   time.Duration
	BackoffInitial    time.Duration
	BackoffMax        time.Duration
	MaxBatchesPerWake int
	ShutdownBudget    time.Duration
}

type DeletionWorker struct {
	dependencies DeletionWorkerDependencies
	config       DeletionWorkerConfig
	wake         chan struct{}
	stop         chan struct{}
	done         chan struct{}
	loopStarted  chan struct{}
	doneOnce     sync.Once
	stopOnce     sync.Once
	startOnce    sync.Once

	mu             sync.Mutex
	err            error
	started        bool
	opCancel       context.CancelFunc
	scheduleCancel context.CancelFunc
	observer       func(deletionWorkerObservation)
}

func NewDeletionWorker(
	dependencies DeletionWorkerDependencies,
	config DeletionWorkerConfig,
) (*DeletionWorker, error) {
	return newDeletionWorker(dependencies, config, nil)
}

func newDeletionWorker(
	dependencies DeletionWorkerDependencies,
	config DeletionWorkerConfig,
	observer func(deletionWorkerObservation),
) (*DeletionWorker, error) {
	if dependencies.Discoverer == nil || dependencies.Converger == nil ||
		dependencies.Now == nil || dependencies.NewTraceID == nil {
		return nil, errors.New("deletion worker dependencies are incomplete")
	}
	if config.PollInterval <= 0 || config.BatchSize <= 0 ||
		config.OperationBudget <= 0 || config.BackoffInitial <= 0 ||
		config.BackoffMax < config.BackoffInitial ||
		config.MaxBatchesPerWake <= 0 || config.ShutdownBudget <= 0 {
		return nil, errors.New("deletion worker configuration is invalid")
	}
	if dependencies.Sleep == nil {
		dependencies.Sleep = sleepWithContext
	}
	if dependencies.Jitter == nil {
		dependencies.Jitter = fullJitter
	}
	return &DeletionWorker{
		dependencies: dependencies,
		config:       config,
		wake:         make(chan struct{}, 1),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		loopStarted:  make(chan struct{}),
		observer:     observer,
	}, nil
}

func (worker *DeletionWorker) Reconcile(ctx context.Context) error {
	if worker == nil || ctx == nil {
		return errors.New("deletion worker context is required")
	}
	backoff := worker.config.BackoffInitial
	for batch := 0; batch < worker.config.MaxBatchesPerWake; batch++ {
		if worker.stopping() {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		discoveryCtx, cancelDiscovery := context.WithTimeout(
			ctx,
			worker.config.OperationBudget,
		)
		worker.setScheduleCancel(cancelDiscovery)
		operationIDs, err := worker.discover(
			discoveryCtx,
			worker.dependencies.Now().UTC(),
			worker.config.BatchSize,
		)
		worker.clearScheduleCancel()
		cancelDiscovery()
		if err != nil {
			worker.observe(
				deletionWorkerCategoryTransient,
				deletionWorkerReasonDatabaseUnavailable,
			)
			if worker.stopping() {
				return nil
			}
			if sleepErr := worker.backoff(ctx, backoff); sleepErr != nil {
				return sleepErr
			}
			backoff = nextBackoff(backoff, worker.config.BackoffMax)
			continue
		}
		if len(operationIDs) == 0 {
			return nil
		}
		transient := false
		for _, operationID := range operationIDs {
			if worker.stopping() {
				return nil
			}
			traceID, err := worker.dependencies.NewTraceID()
			if err != nil {
				worker.observe(
					deletionWorkerCategoryTransient,
					deletionWorkerReasonTraceIDUnavailable,
				)
				transient = true
				continue
			}
			operationCtx, cancel := context.WithTimeout(ctx, worker.config.OperationBudget)
			worker.setOperationCancel(cancel)
			if worker.stopping() {
				worker.clearOperationCancel(cancel)
				cancel()
				return nil
			}
			err = worker.converge(
				operationCtx,
				operationID,
				traceID,
			)
			worker.clearOperationCancel(cancel)
			cancel()
			if isDeletionWorkerTerminal(err) {
				worker.observe(
					deletionWorkerCategoryTerminal,
					deletionWorkerTerminalReason(err),
				)
				return errors.Join(ErrDeletionWorkerTerminal, err)
			}
			switch {
			case errors.Is(err, errDeletionLeaseCompetition):
				worker.observe(
					deletionWorkerCategoryTransient,
					deletionWorkerReasonLeaseCompetition,
				)
				transient = true
			case errors.Is(err, ErrDeletionInProgress):
				worker.observe(
					deletionWorkerCategoryNoop,
					deletionWorkerReasonCompletedNotClaimed,
				)
			case err != nil:
				worker.observe(
					deletionWorkerCategoryTransient,
					deletionWorkerTransientReason(err),
				)
				transient = true
			}
		}
		if worker.stopping() {
			return nil
		}
		if len(operationIDs) < worker.config.BatchSize && !transient {
			return nil
		}
		if transient || batch+1 < worker.config.MaxBatchesPerWake {
			if err := worker.backoff(ctx, backoff); err != nil {
				return err
			}
			backoff = nextBackoff(backoff, worker.config.BackoffMax)
		}
	}
	return nil
}

func (worker *DeletionWorker) Start(ctx context.Context) {
	if worker == nil {
		return
	}
	worker.startOnce.Do(func() {
		worker.mu.Lock()
		if worker.stopping() {
			worker.mu.Unlock()
			worker.doneOnce.Do(func() { close(worker.done) })
			return
		}
		worker.started = true
		worker.mu.Unlock()
		go worker.run(ctx)
	})
}

func (worker *DeletionWorker) Wake() {
	if worker == nil {
		return
	}
	select {
	case <-worker.stop:
		return
	default:
	}
	select {
	case worker.wake <- struct{}{}:
	default:
	}
}

func (worker *DeletionWorker) Stop() {
	if worker == nil {
		return
	}
	worker.stopOnce.Do(func() {
		close(worker.stop)
		worker.cancelSchedule()
		worker.mu.Lock()
		started := worker.started
		worker.mu.Unlock()
		if !started {
			worker.doneOnce.Do(func() { close(worker.done) })
		}
		if started {
			go worker.cancelOperationAfterShutdownBudget()
		}
	})
}

func (worker *DeletionWorker) Wait(ctx context.Context) error {
	if worker == nil || ctx == nil {
		return errors.New("deletion worker wait context is required")
	}
	select {
	case <-worker.done:
		return worker.Err()
	case <-ctx.Done():
		worker.cancelSchedule()
		worker.cancelOperation()
		<-worker.done
		return errors.Join(worker.Err(), ctx.Err())
	}
}

func (worker *DeletionWorker) Done() <-chan struct{} {
	if worker == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return worker.done
}

func (worker *DeletionWorker) Err() error {
	if worker == nil {
		return errors.New("deletion worker is required")
	}
	worker.mu.Lock()
	defer worker.mu.Unlock()
	return worker.err
}

func (worker *DeletionWorker) run(parent context.Context) {
	defer worker.doneOnce.Do(func() { close(worker.done) })
	if parent == nil {
		worker.setError(errors.New("deletion worker context is required"))
		return
	}
	var ticker *time.Ticker
	ticks := worker.dependencies.Ticks
	if ticks == nil {
		ticker = time.NewTicker(worker.config.PollInterval)
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
			worker.setError(fmt.Errorf("deletion worker context stopped: %w", parent.Err()))
			return
		case <-ticks:
		case <-worker.wake:
		}
		select {
		case <-worker.stop:
			return
		default:
		}
		err := worker.Reconcile(context.WithoutCancel(parent))
		if isDeletionWorkerTerminal(err) {
			worker.setError(err)
			return
		}
	}
}

func (worker *DeletionWorker) backoff(ctx context.Context, maximum time.Duration) error {
	delay := worker.dependencies.Jitter(maximum)
	if delay < 0 || delay > maximum {
		delay = maximum
	}
	scheduleCtx, cancelSchedule := context.WithCancel(ctx)
	worker.setScheduleCancel(cancelSchedule)
	defer func() {
		worker.clearScheduleCancel()
		cancelSchedule()
	}()
	return worker.dependencies.Sleep(scheduleCtx, delay)
}

func (worker *DeletionWorker) setOperationCancel(cancel context.CancelFunc) {
	worker.mu.Lock()
	worker.opCancel = cancel
	worker.mu.Unlock()
}

func (worker *DeletionWorker) clearOperationCancel(cancel context.CancelFunc) {
	worker.mu.Lock()
	worker.opCancel = nil
	worker.mu.Unlock()
}

func (worker *DeletionWorker) cancelOperation() {
	worker.mu.Lock()
	cancel := worker.opCancel
	worker.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (worker *DeletionWorker) setScheduleCancel(cancel context.CancelFunc) {
	worker.mu.Lock()
	worker.scheduleCancel = cancel
	worker.mu.Unlock()
	if worker.stopping() {
		cancel()
	}
}

func (worker *DeletionWorker) clearScheduleCancel() {
	worker.mu.Lock()
	worker.scheduleCancel = nil
	worker.mu.Unlock()
}

func (worker *DeletionWorker) cancelSchedule() {
	worker.mu.Lock()
	cancel := worker.scheduleCancel
	worker.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (worker *DeletionWorker) cancelOperationAfterShutdownBudget() {
	timer := time.NewTimer(worker.config.ShutdownBudget)
	defer timer.Stop()
	select {
	case <-timer.C:
		worker.cancelOperation()
	case <-worker.done:
	}
}

func (worker *DeletionWorker) setError(err error) {
	worker.mu.Lock()
	worker.err = err
	worker.mu.Unlock()
}

func (worker *DeletionWorker) stopping() bool {
	select {
	case <-worker.stop:
		return true
	default:
		return false
	}
}

func isDeletionWorkerTerminal(err error) bool {
	return errors.Is(err, ErrDeletionWorkerTerminal) ||
		errors.Is(err, ErrDeletionJournalCorrupt) ||
		errors.Is(err, ErrDeletionTombstoneNotFound) ||
		errors.Is(err, ErrDeletionArticleMismatch)
}

func deletionWorkerTransientReason(err error) string {
	switch {
	case platform.PublicCode(err) == platform.CodePlatformDatabaseUnavailable:
		return deletionWorkerReasonDatabaseUnavailable
	case errors.Is(err, ErrDeletionJournalBusy),
		errors.Is(err, errDeletionJournalFilesystem):
		return deletionWorkerReasonJournalUnavailable
	default:
		return deletionWorkerReasonConvergenceUnavailable
	}
}

func deletionWorkerTerminalReason(err error) string {
	switch {
	case errors.Is(err, ErrDeletionJournalCorrupt):
		return deletionWorkerReasonJournalCorrupt
	case errors.Is(err, ErrDeletionTombstoneNotFound):
		return deletionWorkerReasonTombstoneNotFound
	case errors.Is(err, ErrDeletionArticleMismatch):
		return deletionWorkerReasonArticleMismatch
	default:
		return deletionWorkerReasonInvariantFailure
	}
}

func (worker *DeletionWorker) observe(category, reason string) {
	if worker == nil || worker.observer == nil {
		return
	}
	worker.observer(deletionWorkerObservation{
		Category: category,
		Reason:   reason,
	})
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func fullJitter(maximum time.Duration) time.Duration {
	if maximum <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(maximum) + 1))
}

func nextBackoff(current, maximum time.Duration) time.Duration {
	if current >= maximum || current > maximum/2 {
		return maximum
	}
	return current * 2
}

// Started closes only when the actual polling goroutine has entered its loop.
func (worker *DeletionWorker) Started() <-chan struct{} { return worker.loopStarted }

func (worker *DeletionWorker) acquireWork(ctx context.Context) (context.Context, func(), error) {
	if worker.dependencies.AcquireWork == nil {
		return ctx, func() {}, nil
	}
	return worker.dependencies.AcquireWork(ctx)
}

func (worker *DeletionWorker) discover(ctx context.Context, now time.Time, limit int) ([]string, error) {
	work, release, err := worker.acquireWork(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return worker.dependencies.Discoverer.DiscoverDeletionOperations(work, now, limit)
}

func (worker *DeletionWorker) converge(ctx context.Context, id, trace string) error {
	work, release, err := worker.acquireWork(ctx)
	if err != nil {
		return err
	}
	defer release()
	return worker.dependencies.Converger.ConvergeDeletionOperation(work, id, trace)
}

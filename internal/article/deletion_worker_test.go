package article

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/platform"
)

func TestDeletionWorker_AWIREWKR06WKR08StableBatchesAndTerminal(t *testing.T) {
	terminal := errors.Join(
		ErrDeletionWorkerTerminal,
		ErrDeletionJournalCorrupt,
	)
	discoverer := &deletionWorkerDiscovererStub{
		batches: [][]string{
			{"operation-1", "operation-2"},
			{"operation-3"},
		},
	}
	converger := &deletionWorkerConvergerStub{
		errors: map[string]error{"operation-3": terminal},
	}
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		Discoverer: discoverer,
		Converger:  converger,
		Now: func() time.Time {
			return time.Date(2026, time.October, 23, 13, 0, 0, 0, time.UTC)
		},
		NewTraceID: func() (string, error) { return "trace-worker", nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
		Jitter:     func(time.Duration) time.Duration { return 0 },
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         2,
		OperationBudget:   time.Second,
		BackoffInitial:    50 * time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 3,
		ShutdownBudget:    time.Second,
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}

	err = worker.Reconcile(context.Background())

	if !errors.Is(err, ErrDeletionWorkerTerminal) ||
		!errors.Is(err, ErrDeletionJournalCorrupt) {
		t.Fatalf("Reconcile() error = %v, want terminal corruption", err)
	}
	if !reflect.DeepEqual(
		converger.calls,
		[]string{"operation-1", "operation-2", "operation-3"},
	) {
		t.Fatalf("converge calls = %v", converger.calls)
	}
	if !reflect.DeepEqual(discoverer.limits, []int{2, 2}) {
		t.Fatalf("discovery limits = %v, want [2 2]", discoverer.limits)
	}
}

func TestDeletionWorker_AWIREWKR05WKR09WakeCoalescesAndSingleConcurrency(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseOperation := func() {
		releaseOnce.Do(func() { close(release) })
	}
	discoverer := &deletionWorkerDiscovererStub{
		batches: [][]string{{}, {"operation-1"}, {}, {}},
	}
	converger := &deletionWorkerConvergerStub{
		onCall: func(ctx context.Context, operationID string) error {
			if operationID != "operation-1" {
				t.Fatalf("operation ID = %q", operationID)
			}
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	ticks := make(chan time.Time)
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		Discoverer: discoverer,
		Converger:  converger,
		Now:        time.Now,
		NewTraceID: func() (string, error) { return "trace-worker", nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
		Jitter:     func(time.Duration) time.Duration { return 0 },
		Ticks:      ticks,
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         1,
		OperationBudget:   time.Second,
		BackoffInitial:    50 * time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 1,
		ShutdownBudget:    time.Second,
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}
	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("initial Reconcile() error = %v", err)
	}
	worker.Start(context.Background())
	t.Cleanup(func() {
		worker.Stop()
		releaseOperation()
		_ = worker.Wait(context.Background())
	})

	worker.Wake()
	worker.Wake()
	worker.Wake()
	<-started
	worker.Stop()

	select {
	case <-worker.Done():
		t.Fatal("worker exited before the started operation drained")
	default:
	}
	releaseOperation()
	if err := worker.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if converger.maxConcurrent != 1 {
		t.Fatalf("max concurrent convergence = %d, want 1", converger.maxConcurrent)
	}
}

func TestDeletionWorker_AWIREWKR10LIFE02GlobalDeadlineCancelsAndJoins(t *testing.T) {
	started := make(chan struct{})
	cancelObserved := make(chan struct{})
	release := make(chan struct{})
	discoverer := &deletionWorkerDiscovererStub{
		batches: [][]string{{"operation-1"}},
	}
	converger := &deletionWorkerConvergerStub{
		onCall: func(ctx context.Context, operationID string) error {
			if operationID != "operation-1" {
				t.Fatalf("operation ID = %q", operationID)
			}
			close(started)
			<-ctx.Done()
			close(cancelObserved)
			<-release
			return ctx.Err()
		},
	}
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		Discoverer: discoverer,
		Converger:  converger,
		Now:        time.Now,
		NewTraceID: func() (string, error) { return "trace-worker", nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
		Jitter:     func(time.Duration) time.Duration { return 0 },
		Ticks:      make(chan time.Time),
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         1,
		OperationBudget:   time.Minute,
		BackoffInitial:    50 * time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 1,
		ShutdownBudget:    time.Minute,
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}
	worker.Start(context.Background())
	worker.Wake()
	<-started
	worker.Stop()

	waitCtx, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	waitResult := make(chan error, 1)
	go func() {
		waitResult <- worker.Wait(waitCtx)
	}()
	<-cancelObserved
	select {
	case err := <-waitResult:
		t.Fatalf("Wait() returned before worker joined: %v", err)
	default:
	}
	close(release)
	if err := <-waitResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait() error = %v, want context canceled", err)
	}
	select {
	case <-worker.Done():
	default:
		t.Fatal("worker was not joined before Wait returned")
	}
}

func TestDeletionWorkerConfig_RejectsInvalidBounds(t *testing.T) {
	base := DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         50,
		OperationBudget:   10 * time.Second,
		BackoffInitial:    250 * time.Millisecond,
		BackoffMax:        30 * time.Second,
		MaxBatchesPerWake: 3,
		ShutdownBudget:    10 * time.Second,
	}
	tests := []struct {
		name   string
		mutate func(*DeletionWorkerConfig)
	}{
		{"poll", func(config *DeletionWorkerConfig) { config.PollInterval = 0 }},
		{"batch", func(config *DeletionWorkerConfig) { config.BatchSize = 0 }},
		{"operation budget", func(config *DeletionWorkerConfig) { config.OperationBudget = 0 }},
		{"backoff order", func(config *DeletionWorkerConfig) {
			config.BackoffInitial = 2 * time.Second
			config.BackoffMax = time.Second
		}},
		{"max batches", func(config *DeletionWorkerConfig) { config.MaxBatchesPerWake = 0 }},
		{"shutdown", func(config *DeletionWorkerConfig) { config.ShutdownBudget = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := base
			test.mutate(&config)
			_, err := NewDeletionWorker(DeletionWorkerDependencies{
				Discoverer: &deletionWorkerDiscovererStub{},
				Converger:  &deletionWorkerConvergerStub{},
				Now:        time.Now,
				NewTraceID: func() (string, error) { return "trace", nil },
			}, config)
			if err == nil {
				t.Fatal("NewDeletionWorker() error = nil")
			}
		})
	}
}

func TestDeletionWorker_AWIREWKR07DistinguishesCompetitionFromNoOp(t *testing.T) {
	tests := []struct {
		name       string
		result     error
		wantSleeps int
	}{
		{
			name: "active lease competition backs off",
			result: errors.Join(
				ErrDeletionInProgress,
				errDeletionLeaseCompetition,
			),
			wantSleeps: 1,
		},
		{
			name:       "completed or not claimed is a no-op",
			result:     ErrDeletionInProgress,
			wantSleeps: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			discoverer := &deletionWorkerDiscovererStub{
				batches: [][]string{{"operation-1"}},
			}
			converger := &deletionWorkerConvergerStub{
				errors: map[string]error{"operation-1": test.result},
			}
			var sleeps []time.Duration
			worker, err := NewDeletionWorker(DeletionWorkerDependencies{
				Discoverer: discoverer,
				Converger:  converger,
				Now:        time.Now,
				NewTraceID: func() (string, error) { return "trace-worker", nil },
				Sleep: func(_ context.Context, delay time.Duration) error {
					sleeps = append(sleeps, delay)
					return nil
				},
				Jitter: func(maximum time.Duration) time.Duration {
					return maximum / 2
				},
			}, DeletionWorkerConfig{
				PollInterval:      time.Second,
				BatchSize:         2,
				OperationBudget:   time.Second,
				BackoffInitial:    100 * time.Millisecond,
				BackoffMax:        time.Second,
				MaxBatchesPerWake: 1,
				ShutdownBudget:    time.Second,
			})
			if err != nil {
				t.Fatalf("NewDeletionWorker() error = %v", err)
			}

			if err := worker.Reconcile(context.Background()); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if len(sleeps) != test.wantSleeps {
				t.Fatalf("sleep calls = %v, want %d", sleeps, test.wantSleeps)
			}
			if test.wantSleeps == 1 && sleeps[0] != 50*time.Millisecond {
				t.Fatalf("backoff = %v, want full-jitter seam result", sleeps[0])
			}
		})
	}
}

func TestDeletionWorker_AWIREWKR03DiscoveryStateOrderAndLimitMatrix(t *testing.T) {
	now := time.Date(2026, time.October, 23, 14, 0, 0, 0, time.UTC)
	repository := &deletionWorkerRepositoryFake{
		operations: []deletionWorkerRepositoryOperation{
			{id: "operation-journaled", status: DeletionOperationJournaled, startedAt: now.Add(-time.Minute)},
			{id: "operation-pending-b", status: DeletionOperationPendingJournal, startedAt: now.Add(-2 * time.Minute)},
			{id: "operation-pending-a", status: DeletionOperationPendingJournal, startedAt: now.Add(-2 * time.Minute)},
			{
				id: "operation-expired", status: DeletionOperationFinalizing,
				startedAt: now.Add(-3 * time.Minute), lockedUntil: now,
			},
			{
				id: "operation-active", status: DeletionOperationFinalizing,
				startedAt: now.Add(-4 * time.Minute), lockedUntil: now.Add(time.Second),
			},
			{id: "operation-completed", status: DeletionOperationCompleted, startedAt: now.Add(-5 * time.Minute)},
		},
	}
	converger := &deletionWorkerConvergerStub{}
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		Discoverer: repository,
		Converger:  converger,
		Now:        func() time.Time { return now },
		NewTraceID: func() (string, error) { return "trace-worker", nil },
		Sleep:      func(context.Context, time.Duration) error { return nil },
		Jitter:     func(time.Duration) time.Duration { return 0 },
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         3,
		OperationBudget:   time.Second,
		BackoffInitial:    time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 1,
		ShutdownBudget:    time.Second,
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}

	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	if !reflect.DeepEqual(
		repository.returned,
		[]string{"operation-expired", "operation-pending-a", "operation-pending-b"},
	) {
		t.Fatalf("discovery result = %v", repository.returned)
	}
	if !reflect.DeepEqual(converger.calls, repository.returned) {
		t.Fatalf(
			"convergence order = %v, want discovery order %v",
			converger.calls,
			repository.returned,
		)
	}
	if !reflect.DeepEqual(repository.limits, []int{3}) {
		t.Fatalf("discovery limits = %v, want [3]", repository.limits)
	}
	if !reflect.DeepEqual(repository.times, []time.Time{now}) {
		t.Fatalf("discovery times = %v, want UTC now", repository.times)
	}
}

func TestDeletionWorker_AWIREWKR06BoundsBatchesAndFullJitterBackoff(t *testing.T) {
	discoverer := &deletionWorkerDiscovererStub{
		batches: [][]string{{"operation-1"}, {"operation-2"}, {"operation-3"}},
	}
	converger := &deletionWorkerConvergerStub{}
	var jitterMaximums []time.Duration
	var sleeps []time.Duration
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		Discoverer: discoverer,
		Converger:  converger,
		Now:        time.Now,
		NewTraceID: func() (string, error) { return "trace-worker", nil },
		Jitter: func(maximum time.Duration) time.Duration {
			jitterMaximums = append(jitterMaximums, maximum)
			return maximum / 2
		},
		Sleep: func(_ context.Context, delay time.Duration) error {
			sleeps = append(sleeps, delay)
			return nil
		},
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         1,
		OperationBudget:   time.Second,
		BackoffInitial:    100 * time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 2,
		ShutdownBudget:    time.Second,
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}

	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	if !reflect.DeepEqual(converger.calls, []string{"operation-1", "operation-2"}) {
		t.Fatalf("converge calls = %v, want two bounded batches", converger.calls)
	}
	if !reflect.DeepEqual(jitterMaximums, []time.Duration{100 * time.Millisecond}) ||
		!reflect.DeepEqual(sleeps, []time.Duration{50 * time.Millisecond}) {
		t.Fatalf("jitter/sleeps = %v/%v", jitterMaximums, sleeps)
	}
}

func TestDeletionWorker_AWIREWKR07TraceFailureIsTransient(t *testing.T) {
	discoverer := &deletionWorkerDiscovererStub{
		batches: [][]string{{"operation-1"}},
	}
	var sleeps int
	var observations []deletionWorkerObservation
	worker, err := newDeletionWorker(DeletionWorkerDependencies{
		Discoverer: discoverer,
		Converger:  &deletionWorkerConvergerStub{},
		Now:        time.Now,
		NewTraceID: func() (string, error) {
			return "", errors.New("synthetic trace failure")
		},
		Jitter: func(time.Duration) time.Duration { return 0 },
		Sleep: func(context.Context, time.Duration) error {
			sleeps++
			return nil
		},
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         2,
		OperationBudget:   time.Second,
		BackoffInitial:    time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 1,
		ShutdownBudget:    time.Second,
	}, func(observation deletionWorkerObservation) {
		observations = append(observations, observation)
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}

	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if sleeps != 1 {
		t.Fatalf("backoff calls = %d, want 1", sleeps)
	}
	if !reflect.DeepEqual(observations, []deletionWorkerObservation{{
		Category: deletionWorkerCategoryTransient,
		Reason:   deletionWorkerReasonTraceIDUnavailable,
	}}) {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestDeletionWorker_AWIREWKR07CategoryReasonBucketsAreCompleteAndLowSensitivity(
	t *testing.T,
) {
	const canary = "operation-identifier-path-key-record-body-database-canary"
	tests := []struct {
		name         string
		discoveryErr error
		convergeErr  error
		wantCategory string
		wantReason   string
		wantTerminal bool
	}{
		{
			name: "database discovery unavailable", discoveryErr: errors.New(canary),
			wantCategory: deletionWorkerCategoryTransient,
			wantReason:   deletionWorkerReasonDatabaseUnavailable,
		},
		{
			name: "database convergence unavailable",
			convergeErr: platform.NewPublicError(
				platform.CodePlatformDatabaseUnavailable,
				errors.New(canary),
			),
			wantCategory: deletionWorkerCategoryTransient,
			wantReason:   deletionWorkerReasonDatabaseUnavailable,
		},
		{
			name:         "journal filesystem unavailable",
			convergeErr:  errors.Join(errDeletionJournalFilesystem, errors.New(canary)),
			wantCategory: deletionWorkerCategoryTransient,
			wantReason:   deletionWorkerReasonJournalUnavailable,
		},
		{
			name: "journal writer busy", convergeErr: ErrDeletionJournalBusy,
			wantCategory: deletionWorkerCategoryTransient,
			wantReason:   deletionWorkerReasonJournalUnavailable,
		},
		{
			name:         "lease competition",
			convergeErr:  errors.Join(ErrDeletionInProgress, errDeletionLeaseCompetition),
			wantCategory: deletionWorkerCategoryTransient,
			wantReason:   deletionWorkerReasonLeaseCompetition,
		},
		{
			name: "completed or not claimed", convergeErr: ErrDeletionInProgress,
			wantCategory: deletionWorkerCategoryNoop,
			wantReason:   deletionWorkerReasonCompletedNotClaimed,
		},
		{
			name: "journal corrupt", convergeErr: errors.Join(ErrDeletionJournalCorrupt, errors.New(canary)),
			wantCategory: deletionWorkerCategoryTerminal,
			wantReason:   deletionWorkerReasonJournalCorrupt,
			wantTerminal: true,
		},
		{
			name: "tombstone missing", convergeErr: errors.Join(ErrDeletionTombstoneNotFound, errors.New(canary)),
			wantCategory: deletionWorkerCategoryTerminal,
			wantReason:   deletionWorkerReasonTombstoneNotFound,
			wantTerminal: true,
		},
		{
			name: "article mismatch", convergeErr: errors.Join(ErrDeletionArticleMismatch, errors.New(canary)),
			wantCategory: deletionWorkerCategoryTerminal,
			wantReason:   deletionWorkerReasonArticleMismatch,
			wantTerminal: true,
		},
		{
			name: "invariant failure", convergeErr: errors.Join(ErrDeletionWorkerTerminal, errors.New(canary)),
			wantCategory: deletionWorkerCategoryTerminal,
			wantReason:   deletionWorkerReasonInvariantFailure,
			wantTerminal: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var observations []deletionWorkerObservation
			discoverer := &deletionWorkerDiscovererStub{
				batches: [][]string{{"operation-canary"}},
				err:     test.discoveryErr,
			}
			worker, err := newDeletionWorker(DeletionWorkerDependencies{
				Discoverer: discoverer,
				Converger: &deletionWorkerConvergerStub{
					errors: map[string]error{"operation-canary": test.convergeErr},
				},
				Now:        time.Now,
				NewTraceID: func() (string, error) { return "trace-worker", nil },
				Sleep:      func(context.Context, time.Duration) error { return nil },
				Jitter:     func(time.Duration) time.Duration { return 0 },
			}, DeletionWorkerConfig{
				PollInterval:      time.Second,
				BatchSize:         2,
				OperationBudget:   time.Second,
				BackoffInitial:    time.Millisecond,
				BackoffMax:        time.Second,
				MaxBatchesPerWake: 1,
				ShutdownBudget:    time.Second,
			}, func(observation deletionWorkerObservation) {
				observations = append(observations, observation)
			})
			if err != nil {
				t.Fatalf("NewDeletionWorker() error = %v", err)
			}

			reconcileErr := worker.Reconcile(context.Background())

			if test.wantTerminal != errors.Is(reconcileErr, ErrDeletionWorkerTerminal) {
				t.Fatalf("Reconcile() error = %v, terminal = %t", reconcileErr, test.wantTerminal)
			}
			if !reflect.DeepEqual(observations, []deletionWorkerObservation{{
				Category: test.wantCategory,
				Reason:   test.wantReason,
			}}) {
				t.Fatalf("observations = %#v", observations)
			}
			serialized := observations[0].Category + ":" + observations[0].Reason
			if strings.Contains(serialized, canary) ||
				strings.Contains(serialized, "operation-canary") {
				t.Fatalf("observation leaked sensitive input: %q", serialized)
			}
		})
	}
}

func TestDeletionWorker_AWIREWKR10OperationBudgetCancelsConvergence(t *testing.T) {
	canceled := make(chan struct{})
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		Discoverer: &deletionWorkerDiscovererStub{
			batches: [][]string{{"operation-1"}},
		},
		Converger: &deletionWorkerConvergerStub{
			onCall: func(ctx context.Context, _ string) error {
				<-ctx.Done()
				close(canceled)
				return ctx.Err()
			},
		},
		Now:        time.Now,
		NewTraceID: func() (string, error) { return "trace-worker", nil },
		Jitter:     func(time.Duration) time.Duration { return 0 },
		Sleep:      func(context.Context, time.Duration) error { return nil },
	}, DeletionWorkerConfig{
		PollInterval:      time.Second,
		BatchSize:         2,
		OperationBudget:   time.Millisecond,
		BackoffInitial:    time.Millisecond,
		BackoffMax:        time.Second,
		MaxBatchesPerWake: 1,
		ShutdownBudget:    time.Second,
	})
	if err != nil {
		t.Fatalf("NewDeletionWorker() error = %v", err)
	}

	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	select {
	case <-canceled:
	default:
		t.Fatal("operation budget did not cancel convergence")
	}
}

type deletionWorkerDiscovererStub struct {
	mu      sync.Mutex
	batches [][]string
	limits  []int
	err     error
}

type deletionWorkerRepositoryOperation struct {
	id          string
	status      DeletionOperationStatus
	startedAt   time.Time
	lockedUntil time.Time
}

type deletionWorkerRepositoryFake struct {
	operations []deletionWorkerRepositoryOperation
	limits     []int
	times      []time.Time
	returned   []string
}

func (repository *deletionWorkerRepositoryFake) DiscoverDeletionOperations(
	_ context.Context,
	now time.Time,
	limit int,
) ([]string, error) {
	repository.times = append(repository.times, now)
	repository.limits = append(repository.limits, limit)
	eligible := make([]deletionWorkerRepositoryOperation, 0, len(repository.operations))
	for _, operation := range repository.operations {
		switch operation.status {
		case DeletionOperationPendingJournal, DeletionOperationJournaled:
			eligible = append(eligible, operation)
		case DeletionOperationFinalizing:
			if !operation.lockedUntil.After(now) {
				eligible = append(eligible, operation)
			}
		}
	}
	sort.Slice(eligible, func(left, right int) bool {
		if eligible[left].startedAt.Equal(eligible[right].startedAt) {
			return eligible[left].id < eligible[right].id
		}
		return eligible[left].startedAt.Before(eligible[right].startedAt)
	})
	if len(eligible) > limit {
		eligible = eligible[:limit]
	}
	repository.returned = repository.returned[:0]
	for _, operation := range eligible {
		repository.returned = append(repository.returned, operation.id)
	}
	return append([]string(nil), repository.returned...), nil
}

func (stub *deletionWorkerDiscovererStub) DiscoverDeletionOperations(
	_ context.Context,
	_ time.Time,
	limit int,
) ([]string, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.limits = append(stub.limits, limit)
	if stub.err != nil {
		return nil, stub.err
	}
	if len(stub.batches) == 0 {
		return nil, nil
	}
	batch := append([]string(nil), stub.batches[0]...)
	stub.batches = stub.batches[1:]
	return batch, nil
}

type deletionWorkerConvergerStub struct {
	mu            sync.Mutex
	calls         []string
	errors        map[string]error
	onCall        func(context.Context, string) error
	concurrent    int
	maxConcurrent int
}

func (stub *deletionWorkerConvergerStub) ConvergeDeletionOperation(
	ctx context.Context,
	operationID string,
	_ string,
) error {
	stub.mu.Lock()
	stub.calls = append(stub.calls, operationID)
	stub.concurrent++
	if stub.concurrent > stub.maxConcurrent {
		stub.maxConcurrent = stub.concurrent
	}
	stub.mu.Unlock()
	defer func() {
		stub.mu.Lock()
		stub.concurrent--
		stub.mu.Unlock()
	}()
	if stub.onCall != nil {
		return stub.onCall(ctx, operationID)
	}
	return stub.errors[operationID]
}

func TestDeletionWorker_AllScanClaimDispatchLeases(t *testing.T) {
	held, acquired, released := 0, 0, 0
	entered, finish := make(chan struct{}), make(chan struct{})
	worker, err := NewDeletionWorker(DeletionWorkerDependencies{
		AcquireWork: func(ctx context.Context) (context.Context, func(), error) {
			if held != 0 {
				t.Error("nested acquisition")
			}
			held++
			acquired++
			return ctx, func() { held--; released++ }, nil
		},
		Discoverer: &deletionWorkerDiscovererStub{batches: [][]string{{"operation-1"}}},
		Converger: &deletionWorkerConvergerStub{onCall: func(context.Context, string) error {
			if held != 1 {
				t.Error("operation has no work lease")
			}
			close(entered)
			<-finish
			return nil
		}},
		Now: time.Now, NewTraceID: func() (string, error) { return "worker-trace", nil },
	}, DeletionWorkerConfig{PollInterval: time.Second, BatchSize: 1, OperationBudget: time.Second, BackoffInitial: time.Millisecond, BackoffMax: time.Second, MaxBatchesPerWake: 1, ShutdownBudget: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- worker.Reconcile(context.Background()) }()
	<-entered
	if held != 1 || released != 1 {
		t.Error("claim/convergence released work before actual completion")
	}
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if held != 0 || acquired != 2 || released != 2 {
		t.Fatalf("lease ledger %d/%d/%d", held, acquired, released)
	}
}

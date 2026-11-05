package identity

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPasswordResetWorkerPermitOwnershipPaths(t *testing.T) {
	permit, err := NewPasswordResetConnectionPermit(1)
	if err != nil {
		t.Fatal(err)
	}
	backend := &passwordResetWorkerBackendFake{permit: permit, candidates: []PasswordResetRecoveryCandidate{{OperationID: "op-1", UserID: "user-1"}}, claim: PasswordResetRecoveryLease{OperationID: "op-1", UserID: "user-1", Owner: "worker-1", AttemptCount: 1}}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error {
		backend.recordHeld("settlement")
		return nil
	})
	worker, err := newPasswordResetWorkerForTest(passwordResetWorkerTestDependencies(permit, operator), passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatalf("newPasswordResetWorkerForTest() error = %v", err)
	}
	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if backend.heldViolations != 0 {
		t.Fatalf("permit hold violations = %d", backend.heldViolations)
	}
	if got := backend.calls; !equalPasswordResetStrings(got, []string{"scan", "claim", "renew", "settlement"}) {
		t.Fatalf("calls = %v", got)
	}
	stats := permit.Stats()
	if stats.Acquired != 4 || stats.Released != 4 || stats.Current != 0 || stats.Peak != 1 {
		t.Fatalf("permit stats = %+v", stats)
	}
}

func TestPasswordResetWorkerRetryAndDeadLetterKeepAuthorityHeld(t *testing.T) {
	for _, attempt := range []int64{1, 2} {
		t.Run(time.Duration(attempt).String(), func(t *testing.T) {
			permit, _ := NewPasswordResetConnectionPermit(1)
			settlementErr := errors.New("synthetic settlement failure")
			backend := &passwordResetWorkerBackendFake{permit: permit, candidates: []PasswordResetRecoveryCandidate{{OperationID: "op-1", UserID: "user-1"}}, claim: PasswordResetRecoveryLease{OperationID: "op-1", UserID: "user-1", Owner: "worker-1", AttemptCount: attempt}}
			operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { return settlementErr })
			config := passwordResetWorkerTestConfig()
			config.MaxRetries = 2
			worker, err := newPasswordResetWorkerForTest(passwordResetWorkerTestDependencies(permit, operator), config, backend)
			if err != nil {
				t.Fatal(err)
			}
			if err := worker.Reconcile(context.Background()); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if backend.failCalls != 1 {
				t.Fatalf("fail calls = %d", backend.failCalls)
			}
			if backend.deadLetter != (attempt >= 2) {
				t.Fatalf("dead-letter = %t for attempt %d", backend.deadLetter, attempt)
			}
			if backend.releaseAuthority {
				t.Fatal("retry/dead-letter released operation Gate authority")
			}
			stats := permit.Stats()
			if stats.Acquired != 4 || stats.Released != 4 || stats.Current != 0 {
				t.Fatalf("permit stats = %+v", stats)
			}
		})
	}
}

func TestPasswordResetWorkerLeaseCompetitionDoesNotSettle(t *testing.T) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	backend := &passwordResetWorkerBackendFake{permit: permit, candidates: []PasswordResetRecoveryCandidate{{OperationID: "op-1", UserID: "user-1"}}, claimMiss: true}
	settled := 0
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { settled++; return nil })
	worker, err := newPasswordResetWorkerForTest(passwordResetWorkerTestDependencies(permit, operator), passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if settled != 0 || backend.renewCalls != 0 || backend.failCalls != 0 {
		t.Fatalf("settled/renew/fail = %d/%d/%d", settled, backend.renewCalls, backend.failCalls)
	}
	stats := permit.Stats()
	if stats.Acquired != 2 || stats.Released != 2 || stats.Current != 0 {
		t.Fatalf("permit stats = %+v", stats)
	}
}

func TestPasswordResetWorkerLifecycle(t *testing.T) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	ticks := make(chan time.Time, 1)
	backend := &passwordResetWorkerBackendFake{permit: permit, scanEvents: make(chan struct{}, 2)}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { return nil })
	dependencies := passwordResetWorkerTestDependencies(permit, operator)
	dependencies.Ticks = ticks
	worker, err := newPasswordResetWorkerForTest(dependencies, passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	worker.Start(context.Background())
	ticks <- time.Now()
	backend.waitForScans(t, 2)
	worker.Stop()
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Wait(waitCtx); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	select {
	case <-worker.Done():
	default:
		t.Fatal("Done not closed")
	}
	if err := worker.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
}

func TestPasswordResetWorkerStopCancelsActiveReconcile(t *testing.T) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	ticks := make(chan time.Time, 1)
	backend := &passwordResetWorkerBackendFake{
		permit:       permit,
		scanEntered:  make(chan struct{}),
		scanCanceled: make(chan struct{}),
		scanRelease:  make(chan struct{}),
	}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { return nil })
	dependencies := passwordResetWorkerTestDependencies(permit, operator)
	dependencies.Ticks = ticks
	worker, err := newPasswordResetWorkerForTest(dependencies, passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatal(err)
	}
	worker.Start(context.Background())
	ticks <- time.Now()
	<-backend.scanEntered
	worker.Stop()
	select {
	case <-backend.scanCanceled:
	case <-time.After(time.Second):
		close(backend.scanRelease)
		t.Fatal("active reconcile context was not canceled by Stop")
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Wait(waitCtx); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	stats := permit.Stats()
	if stats.Current != 0 || stats.Acquired != stats.Released {
		t.Fatalf("permit leaked after cancellation: %+v", stats)
	}
}

func TestPasswordResetWorkerClaimUsesClockAfterScan(t *testing.T) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	initial := time.Date(2026, 10, 25, 1, 0, 0, 0, time.UTC)
	afterScan := initial.Add(2 * time.Minute)
	clock := &passwordResetMutableClock{now: initial}
	backend := &passwordResetWorkerBackendFake{
		permit:     permit,
		candidates: []PasswordResetRecoveryCandidate{{OperationID: "op-1", UserID: "user-1"}},
		claimMiss:  true,
		afterScan:  func() { clock.Set(afterScan) },
	}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { return nil })
	dependencies := passwordResetWorkerTestDependencies(permit, operator)
	dependencies.Clock = clock
	worker, err := newPasswordResetWorkerForTest(dependencies, passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !backend.claimNow.Equal(afterScan) {
		t.Fatalf("claim time = %v, want post-scan %v", backend.claimNow, afterScan)
	}
}

func TestPasswordResetWorkerCommitUnknownDoesNotFailOrStop(t *testing.T) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	ticks := make(chan time.Time, 2)
	backend := &passwordResetWorkerBackendFake{
		permit:     permit,
		candidates: []PasswordResetRecoveryCandidate{{OperationID: "op-1", UserID: "user-1"}},
		claim:      PasswordResetRecoveryLease{OperationID: "op-1", UserID: "user-1", Owner: "worker-1", AttemptCount: 1},
		scanEvents: make(chan struct{}, 2),
	}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error {
		return ErrPasswordResetSettlementCommitUnknown
	})
	dependencies := passwordResetWorkerTestDependencies(permit, operator)
	dependencies.Ticks = ticks
	worker, err := newPasswordResetWorkerForTest(dependencies, passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatal(err)
	}
	worker.Start(context.Background())
	ticks <- time.Now()
	ticks <- time.Now()
	backend.waitForScans(t, 2)
	if backend.failCalls != 0 {
		t.Fatalf("Fail() calls = %d, want 0", backend.failCalls)
	}
	select {
	case <-worker.Done():
		t.Fatalf("worker stopped on commit unknown: %v", worker.Err())
	default:
	}
	worker.Stop()
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Wait(waitCtx); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestPasswordResetWorkerPostgresScanClosesRowsAndReturnsCandidates(t *testing.T) {
	now := time.Date(2026, 10, 25, 5, 0, 0, 0, time.UTC)
	rows := &passwordResetRowsHarness{values: [][]any{
		{"op-1", "user-1"},
		{"op-2", "user-2"},
	}}
	pool := &passwordResetWorkerPoolHarness{rows: rows}
	backend := passwordResetPostgresBackend{pool: pool, store: &PasswordResetStore{}, clock: passwordResetTestClock{now: now}}
	candidates, err := backend.Scan(context.Background(), now, 2)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	want := []PasswordResetRecoveryCandidate{{OperationID: "op-1", UserID: "user-1"}, {OperationID: "op-2", UserID: "user-2"}}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("Scan() candidates = %#v, want %#v", candidates, want)
	}
	if !rows.closed {
		t.Fatal("Scan() did not close rows")
	}
	if pool.queryCalls != 1 || !strings.Contains(pool.querySQL, "lease_expires_at<=$1") {
		t.Fatalf("query calls/SQL = %d/%q", pool.queryCalls, pool.querySQL)
	}
}

func TestPasswordResetWorkerPostgresLeaseUsesClockAfterUserLock(t *testing.T) {
	initial := time.Date(2026, 10, 25, 6, 0, 0, 0, time.UTC)
	afterLock := initial.Add(2 * time.Minute)
	duration := time.Minute
	clock := &passwordResetMutableClock{now: initial}
	candidate := PasswordResetRecoveryCandidate{OperationID: "op-1", UserID: "user-1"}
	tx := &passwordResetScriptTx{
		onQuery: func(sql string) {
			if strings.Contains(sql, passwordResetRecoveryLockUserSQL) {
				clock.Set(afterLock)
			}
		},
		steps: []passwordResetSQLStep{
			{kind: "query", marker: passwordResetRecoveryLockUserSQL, args: []any{candidate.UserID}, row: []any{candidate.UserID}},
			{kind: "query", marker: passwordResetRecoveryClaimSQL, args: []any{candidate.OperationID, candidate.UserID, "worker-1", afterLock, afterLock.Add(duration)}, row: []any{int64(1)}},
		},
	}
	pool := &passwordResetWorkerPoolHarness{passwordResetScriptStarter: passwordResetScriptStarter{transactions: []*passwordResetScriptTx{tx}}}
	backend := passwordResetPostgresBackend{pool: pool, store: &PasswordResetStore{}, clock: clock}
	lease, claimed, err := backend.Claim(context.Background(), candidate, "worker-1", initial, duration)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if !claimed || !lease.ExpiresAt.Equal(afterLock.Add(duration)) {
		t.Fatalf("Claim() lease/claimed = %+v/%t", lease, claimed)
	}
	tx.assertComplete(t)
	if tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("commit/rollback = %d/%d", tx.commits, tx.rollbacks)
	}
}

func TestPasswordResetWorkerLeaseExpiryBoundaryRejectsOldOwnerMutations(t *testing.T) {
	expiry := time.Date(2026, 10, 25, 7, 0, 0, 0, time.UTC)
	lease := passwordResetTestLease()
	lease.ExpiresAt = expiry
	for _, test := range []struct {
		name string
		now  time.Time
		rows int64
	}{
		{name: "before", now: expiry.Add(-time.Microsecond), rows: 1},
		{name: "equal", now: expiry, rows: 0},
		{name: "after", now: expiry.Add(time.Microsecond), rows: 0},
	} {
		t.Run("renew/"+test.name, func(t *testing.T) {
			clock := &passwordResetMutableClock{now: expiry.Add(-time.Minute)}
			tx := &passwordResetScriptTx{
				onQuery: func(string) { clock.Set(test.now) },
				steps: []passwordResetSQLStep{
					{kind: "query", marker: passwordResetRecoveryLockUserSQL, args: []any{lease.UserID}, row: []any{lease.UserID}},
					execStep("lease_expires_at>$2", []any{test.now.Add(time.Minute), test.now, lease.OperationID, lease.Owner, lease.AttemptCount}, test.rows),
				},
			}
			pool := &passwordResetWorkerPoolHarness{passwordResetScriptStarter: passwordResetScriptStarter{transactions: []*passwordResetScriptTx{tx}}}
			backend := passwordResetPostgresBackend{pool: pool, store: &PasswordResetStore{}, clock: clock}
			renewed, err := backend.Renew(context.Background(), lease, expiry.Add(-time.Minute), time.Minute)
			if err != nil {
				t.Fatalf("Renew() error = %v", err)
			}
			if renewed != (test.rows == 1) {
				t.Fatalf("Renew() = %t, want %t", renewed, test.rows == 1)
			}
		})
	}
	t.Run("fail/equal", func(t *testing.T) {
		clock := &passwordResetMutableClock{now: expiry.Add(-time.Minute)}
		tx := &passwordResetScriptTx{
			onQuery: func(string) { clock.Set(expiry) },
			steps: []passwordResetSQLStep{
				{kind: "query", marker: passwordResetRecoveryLockUserSQL, args: []any{lease.UserID}, row: []any{lease.UserID}},
				execStep("lease_expires_at>$4", []any{PasswordResetRecoveryJobRetryWait, expiry.Add(time.Second), "settlement_unavailable", expiry, lease.OperationID, lease.Owner, lease.AttemptCount}, 0),
			},
		}
		pool := &passwordResetWorkerPoolHarness{passwordResetScriptStarter: passwordResetScriptStarter{transactions: []*passwordResetScriptTx{tx}}}
		backend := passwordResetPostgresBackend{pool: pool, store: &PasswordResetStore{}, clock: clock}
		failed, err := backend.Fail(context.Background(), lease, expiry.Add(-time.Minute), time.Second, false, "settlement_unavailable")
		if err != nil {
			t.Fatalf("Fail() error = %v", err)
		}
		if failed {
			t.Fatal("expired owner persisted failure")
		}
	})
}

func TestPasswordResetWorkerTwoWorkerTakeoverSettlesExactlyOnce(t *testing.T) {
	now := time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)
	state := &passwordResetLeaseState{now: now}
	candidate := PasswordResetRecoveryCandidate{OperationID: "op-1", UserID: "user-1"}
	first, claimed, err := state.Claim(context.Background(), candidate, "worker-1", now, time.Minute)
	if err != nil || !claimed {
		t.Fatalf("first Claim() = %+v/%t/%v", first, claimed, err)
	}
	state.setNow(first.ExpiresAt)
	second, claimed, err := state.Claim(context.Background(), candidate, "worker-2", first.ExpiresAt, time.Minute)
	if err != nil || !claimed {
		t.Fatalf("takeover Claim() = %+v/%t/%v", second, claimed, err)
	}
	if renewed, _ := state.Renew(context.Background(), first, first.ExpiresAt, time.Minute); renewed {
		t.Fatal("expired first owner renewed lease")
	}
	if failed, _ := state.Fail(context.Background(), first, first.ExpiresAt, time.Second, false, "failure"); failed {
		t.Fatal("expired first owner persisted failure")
	}
	if err := state.Settle(first); !errors.Is(err, ErrPasswordResetStateConflict) {
		t.Fatalf("old owner settlement error = %v", err)
	}
	if err := state.Settle(second); err != nil {
		t.Fatalf("takeover settlement error = %v", err)
	}
	if err := state.Settle(second); !errors.Is(err, ErrPasswordResetStateConflict) {
		t.Fatalf("duplicate settlement error = %v", err)
	}
	if state.settlements != 1 || state.attempt != 2 || state.owner != "" {
		t.Fatalf("state = settlements:%d attempt:%d owner:%q", state.settlements, state.attempt, state.owner)
	}
}

func TestPasswordResetWorkerConfigurationRejectsNonPositiveValues(t *testing.T) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { return nil })
	base := passwordResetWorkerTestConfig()
	cases := []struct {
		name   string
		mutate func(*PasswordResetWorkerConfig)
	}{
		{"lease", func(c *PasswordResetWorkerConfig) { c.LeaseDuration = 0 }},
		{"backoff", func(c *PasswordResetWorkerConfig) { c.RetryBackoff = 0 }},
		{"retries", func(c *PasswordResetWorkerConfig) { c.MaxRetries = 0 }},
		{"batch", func(c *PasswordResetWorkerConfig) { c.ClaimBatchLimit = 0 }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			config := base
			test.mutate(&config)
			if _, err := newPasswordResetWorkerForTest(passwordResetWorkerTestDependencies(permit, operator), config, &passwordResetWorkerBackendFake{}); err == nil {
				t.Fatal("constructor succeeded")
			}
		})
	}
}

type passwordResetWorkerBackendFake struct {
	mu               sync.Mutex
	permit           *PasswordResetConnectionPermit
	candidates       []PasswordResetRecoveryCandidate
	claim            PasswordResetRecoveryLease
	claimMiss        bool
	calls            []string
	scans            int
	scanEvents       chan struct{}
	heldViolations   int
	renewCalls       int
	failCalls        int
	deadLetter       bool
	releaseAuthority bool
	afterScan        func()
	claimNow         time.Time
	scanEntered      chan struct{}
	scanCanceled     chan struct{}
	scanRelease      chan struct{}
}

func (backend *passwordResetWorkerBackendFake) recordHeld(call string) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.permit != nil && backend.permit.Stats().Current != 1 {
		backend.heldViolations++
	}
	backend.calls = append(backend.calls, call)
	if call == "scan" {
		backend.scans++
		if backend.scanEvents != nil {
			backend.scanEvents <- struct{}{}
		}
	}
}
func (backend *passwordResetWorkerBackendFake) Scan(ctx context.Context, _ time.Time, _ int) ([]PasswordResetRecoveryCandidate, error) {
	backend.recordHeld("scan")
	if backend.scanEntered != nil {
		close(backend.scanEntered)
		select {
		case <-ctx.Done():
			close(backend.scanCanceled)
			return nil, ctx.Err()
		case <-backend.scanRelease:
		}
	}
	if backend.afterScan != nil {
		backend.afterScan()
	}
	return append([]PasswordResetRecoveryCandidate(nil), backend.candidates...), nil
}
func (backend *passwordResetWorkerBackendFake) Claim(_ context.Context, candidate PasswordResetRecoveryCandidate, owner string, now time.Time, _ time.Duration) (PasswordResetRecoveryLease, bool, error) {
	backend.recordHeld("claim")
	backend.claimNow = now
	if backend.claimMiss {
		return PasswordResetRecoveryLease{}, false, nil
	}
	lease := backend.claim
	lease.OperationID = candidate.OperationID
	lease.UserID = candidate.UserID
	lease.Owner = owner
	return lease, true, nil
}
func (backend *passwordResetWorkerBackendFake) Renew(_ context.Context, _ PasswordResetRecoveryLease, _ time.Time, _ time.Duration) (bool, error) {
	backend.recordHeld("renew")
	backend.renewCalls++
	return true, nil
}
func (backend *passwordResetWorkerBackendFake) Fail(_ context.Context, _ PasswordResetRecoveryLease, _ time.Time, _ time.Duration, deadLetter bool, _ string) (bool, error) {
	backend.recordHeld("fail")
	backend.mu.Lock()
	backend.failCalls++
	backend.deadLetter = deadLetter
	backend.mu.Unlock()
	return true, nil
}
func (backend *passwordResetWorkerBackendFake) waitForScans(t *testing.T, want int) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for index := 0; index < want; index++ {
		select {
		case <-backend.scanEvents:
		case <-deadline.C:
			backend.mu.Lock()
			got := backend.scans
			backend.mu.Unlock()
			t.Fatalf("scans = %d, want %d", got, want)
		}
	}
}

func passwordResetWorkerTestDependencies(permit *PasswordResetConnectionPermit, operator *PasswordResetOperator) PasswordResetWorkerDependencies {
	return PasswordResetWorkerDependencies{Permit: permit, Operator: operator, WorkerID: "worker-1", Clock: passwordResetTestClock{now: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)}}
}
func passwordResetWorkerTestConfig() PasswordResetWorkerConfig {
	return PasswordResetWorkerConfig{LeaseDuration: time.Minute, RetryBackoff: time.Second, MaxRetries: 3, ClaimBatchLimit: 10, ConnectionBudget: 1}
}

type passwordResetTestClock struct{ now time.Time }

func (clock passwordResetTestClock) Now() time.Time { return clock.now }

type passwordResetMutableClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *passwordResetMutableClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *passwordResetMutableClock) Set(now time.Time) {
	clock.mu.Lock()
	clock.now = now
	clock.mu.Unlock()
}

func equalPasswordResetStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

type passwordResetWorkerPoolHarness struct {
	passwordResetScriptStarter
	rows       pgx.Rows
	queryCalls int
	querySQL   string
}

func (pool *passwordResetWorkerPoolHarness) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	pool.queryCalls++
	pool.querySQL = compactPasswordResetSQL(sql)
	if pool.rows == nil {
		return nil, errors.New("unexpected password reset query")
	}
	return pool.rows, nil
}

type passwordResetRowsHarness struct {
	pgx.Rows
	values [][]any
	index  int
	closed bool
	err    error
}

func (rows *passwordResetRowsHarness) Next() bool {
	return rows.index < len(rows.values)
}

func (rows *passwordResetRowsHarness) Scan(dest ...any) error {
	if rows.index >= len(rows.values) {
		return errors.New("password reset rows exhausted")
	}
	err := (passwordResetScriptRow{values: rows.values[rows.index]}).Scan(dest...)
	rows.index++
	return err
}

func (rows *passwordResetRowsHarness) Err() error { return rows.err }

func (rows *passwordResetRowsHarness) Close() { rows.closed = true }

type passwordResetLeaseState struct {
	mu          sync.Mutex
	now         time.Time
	owner       string
	expiry      time.Time
	attempt     int64
	settlements int
	settled     bool
}

func (state *passwordResetLeaseState) setNow(now time.Time) {
	state.mu.Lock()
	state.now = now
	state.mu.Unlock()
}

func (*passwordResetLeaseState) Scan(context.Context, time.Time, int) ([]PasswordResetRecoveryCandidate, error) {
	return nil, nil
}

func (state *passwordResetLeaseState) Claim(_ context.Context, candidate PasswordResetRecoveryCandidate, owner string, now time.Time, duration time.Duration) (PasswordResetRecoveryLease, bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.settled || (state.owner != "" && state.expiry.After(now)) {
		return PasswordResetRecoveryLease{}, false, nil
	}
	state.owner = owner
	state.expiry = now.Add(duration)
	state.attempt++
	return PasswordResetRecoveryLease{OperationID: candidate.OperationID, UserID: candidate.UserID, Owner: owner, AttemptCount: state.attempt, ExpiresAt: state.expiry}, true, nil
}

func (state *passwordResetLeaseState) Renew(_ context.Context, lease PasswordResetRecoveryLease, now time.Time, duration time.Duration) (bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.settled || state.owner != lease.Owner || state.attempt != lease.AttemptCount || !state.expiry.After(now) {
		return false, nil
	}
	state.expiry = now.Add(duration)
	return true, nil
}

func (state *passwordResetLeaseState) Fail(_ context.Context, lease PasswordResetRecoveryLease, now time.Time, _ time.Duration, _ bool, _ string) (bool, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.settled || state.owner != lease.Owner || state.attempt != lease.AttemptCount || !state.expiry.After(now) {
		return false, nil
	}
	state.owner = ""
	return true, nil
}

func (state *passwordResetLeaseState) Settle(lease PasswordResetRecoveryLease) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.settled || state.owner != lease.Owner || state.attempt != lease.AttemptCount || !state.expiry.After(state.now) {
		return ErrPasswordResetStateConflict
	}
	state.settled = true
	state.settlements++
	state.owner = ""
	return nil
}

var _ passwordResetWorkerBackend = (*passwordResetLeaseState)(nil)

func formatPasswordResetLeaseState(state *passwordResetLeaseState) string {
	return fmt.Sprintf("%s/%d", state.owner, state.attempt)
}

func TestPasswordResetWorker_AllScanClaimDispatchLeases(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) { testPasswordResetWorkLeases(t, fail) })
	}
}

func testPasswordResetWorkLeases(t *testing.T, fail bool) {
	permit, _ := NewPasswordResetConnectionPermit(1)
	backend := &passwordResetWorkerBackendFake{permit: permit, candidates: []PasswordResetRecoveryCandidate{{OperationID: "op", UserID: "user"}}, claim: PasswordResetRecoveryLease{OperationID: "op", UserID: "user", Owner: "worker", AttemptCount: 1}}
	held, acquired, released := 0, 0, 0
	backend.afterScan = func() {
		if held != 1 {
			t.Error("scan outside work lease")
		}
	}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error {
		if held != 1 {
			t.Error("settlement outside claim work lease")
		}
		if fail {
			return errors.New("settlement failed")
		}
		return nil
	})
	deps := passwordResetWorkerTestDependencies(permit, operator)
	deps.AcquireWork = func(ctx context.Context) (context.Context, func(), error) {
		if held != 0 {
			t.Error("nested acquisition")
		}
		held++
		acquired++
		return ctx, func() {
			if acquired == 2 && fail && backend.failCalls != 1 {
				t.Error("released before failure persistence")
			}
			held--
			released++
		}, nil
	}
	worker, err := newPasswordResetWorkerForTest(deps, passwordResetWorkerTestConfig(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err = worker.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if held != 0 || acquired != 2 || released != 2 {
		t.Fatalf("work lease ledger %d/%d/%d", held, acquired, released)
	}
	if permit.Stats().Acquired != 4 || permit.Stats().Current != 0 {
		t.Fatal("database connection permit changed")
	}
}

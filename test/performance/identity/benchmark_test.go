// Package identityperf measures in-process identity capacity baselines.
//
// The benchmarks exercise production identity, session, security, and mailer
// code with deterministic in-memory collaborators. They do not measure real
// PostgreSQL, SQL planning/indexes/locks, real SMTP/TLS networking, HTTP wire
// behavior, filesystem durability, or an actual 4 vCPU / 512 MiB cgroup.
package identityperf

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"reflect"
	"runtime"
	"runtime/metrics"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"han/internal/audit"
	"han/internal/identity"
	"han/internal/mailer"
	"han/internal/security"
	"han/internal/session"
)

const (
	argon2MemoryKiB      = 65536
	argon2Iterations     = 3
	argon2Parallelism    = 4
	argon2SaltBytes      = 16
	argon2KeyBytes       = 32
	argon2WorkerLimit    = 4
	argon2MemoryLimitKiB = 524288
	workerClaimBatch     = 100
	smtpOperationTimeout = 5 * time.Second
)

var benchmarkNow = time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)

func benchmarkUUID(value int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", value)
}

// deterministicSaltReader supplies reproducible salt bytes without crypto/rand.
type deterministicSaltReader struct {
	state uint64
}

func (reader *deterministicSaltReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		reader.state = reader.state*6364136223846793005 + 1442695040888963407
		buffer[index] = byte(reader.state >> 56)
	}
	return len(buffer), nil
}

type concurrentResult struct {
	latencies   []time.Duration
	failures    int64
	peakHeapKiB uint64
	steadyKiB   uint64
	heapSamples int64
}

func runConcurrent(
	b *testing.B,
	workers int,
	sampleHeap bool,
	operation func(worker int) error,
) concurrentResult {
	b.Helper()
	if workers < 1 {
		workers = 1
	}
	latencies := make([]time.Duration, b.N)
	var next atomic.Int64
	var failures atomic.Int64
	var peakHeap atomic.Uint64
	var samples atomic.Int64
	stopSampling := make(chan struct{})
	samplerDone := make(chan struct{})
	if sampleHeap {
		go func() {
			defer close(samplerDone)
			sampleHeapLoop(stopSampling, &peakHeap, &samples)
		}()
	} else {
		close(samplerDone)
	}

	b.ResetTimer()
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for {
				index := int(next.Add(1) - 1)
				if index >= b.N {
					return
				}
				started := time.Now()
				if err := operation(worker); err != nil {
					failures.Add(1)
				}
				latencies[index] = time.Since(started)
			}
		}(worker)
	}
	wait.Wait()
	b.StopTimer()

	if sampleHeap {
		close(stopSampling)
	}
	<-samplerDone

	runtime.GC()
	steady := readHeapBytes()
	return concurrentResult{
		latencies:   latencies,
		failures:    failures.Load(),
		peakHeapKiB: peakHeap.Load() / 1024,
		steadyKiB:   steady / 1024,
		heapSamples: samples.Load(),
	}
}

func sampleHeapLoop(stop <-chan struct{}, peak *atomic.Uint64, samples *atomic.Int64) {
	sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()
	read := func() {
		metrics.Read(sample)
		value := sample[0].Value.Uint64()
		samples.Add(1)
		for {
			current := peak.Load()
			if value <= current || peak.CompareAndSwap(current, value) {
				return
			}
		}
	}
	read()
	for {
		select {
		case <-stop:
			read()
			return
		case <-ticker.C:
			read()
		}
	}
}

func readHeapBytes() uint64 {
	sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	metrics.Read(sample)
	return sample[0].Value.Uint64()
}

func reportLatencyMetrics(b *testing.B, result concurrentResult) {
	b.Helper()
	if len(result.latencies) == 0 {
		return
	}
	sorted := append([]time.Duration(nil), result.latencies...)
	sort.Slice(sorted, func(left, right int) bool {
		return sorted[left] < sorted[right]
	})
	percentile := func(quantile float64) float64 {
		index := int(math.Ceil(quantile*float64(len(sorted)))) - 1
		if index < 0 {
			index = 0
		}
		if index >= len(sorted) {
			index = len(sorted) - 1
		}
		return float64(sorted[index]) / float64(time.Microsecond)
	}
	b.ReportMetric(percentile(0.50), "p50-us")
	b.ReportMetric(percentile(0.95), "p95-us")
	b.ReportMetric(percentile(0.99), "p99-us")
	b.ReportMetric(float64(result.failures)/float64(len(result.latencies)), "failure-rate")
}

func reportHeapMetrics(b *testing.B, result concurrentResult) {
	b.Helper()
	b.ReportMetric(float64(result.peakHeapKiB), "peak-sampled-heap-KiB")
	b.ReportMetric(float64(result.steadyKiB), "steady-heap-KiB")
	b.ReportMetric(float64(result.heapSamples), "heap-samples")
}

func BenchmarkArgon2idHash(b *testing.B) {
	for concurrency := 1; concurrency <= argon2WorkerLimit; concurrency++ {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			benchmarkArgon2id(b, concurrency, false)
		})
	}
}

func BenchmarkArgon2idVerify(b *testing.B) {
	for concurrency := 1; concurrency <= argon2WorkerLimit; concurrency++ {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			benchmarkArgon2id(b, concurrency, true)
		})
	}
}

func benchmarkArgon2id(b *testing.B, concurrency int, verify bool) {
	b.Helper()
	gate, err := security.NewArgon2idProcessGate(argon2WorkerLimit)
	if err != nil {
		b.Fatalf("create Argon2id process gate: %v", err)
	}
	config := security.Argon2idConfig{
		MemoryKiB:   argon2MemoryKiB,
		Iterations:  argon2Iterations,
		Parallelism: argon2Parallelism,
		SaltLength:  argon2SaltBytes,
		KeyLength:   argon2KeyBytes,
	}
	hashers := make([]*security.GatedArgon2idHasher, concurrency)
	for worker := range hashers {
		base, err := security.NewArgon2idHasher(
			config,
			&deterministicSaltReader{state: uint64(worker + 1)},
		)
		if err != nil {
			b.Fatalf("create Argon2id hasher: %v", err)
		}
		gated, err := security.NewGatedArgon2idHasher(base, gate)
		if err != nil {
			b.Fatalf("create gated Argon2id hasher: %v", err)
		}
		hashers[worker] = gated
	}
	encoded := ""
	if verify {
		encoded, err = hashers[0].Hash("benchmark-password")
		if err != nil {
			b.Fatalf("create verification digest: %v", err)
		}
	}

	b.ReportAllocs()
	result := runConcurrent(b, concurrency, true, func(worker int) error {
		if verify {
			matched, needsRehash, err := hashers[worker].Verify("benchmark-password", encoded)
			if err != nil {
				return err
			}
			if !matched || needsRehash {
				return fmt.Errorf("verify matched=%t needsRehash=%t", matched, needsRehash)
			}
			return nil
		}
		_, err := hashers[worker].Hash("benchmark-password")
		return err
	})
	reportLatencyMetrics(b, result)
	reportHeapMetrics(b, result)
	b.ReportMetric(float64(concurrency), "workers")
	b.ReportMetric(float64(argon2MemoryKiB), "memory-KiB")
	b.ReportMetric(float64(argon2MemoryKiB*concurrency), "memory*concurrency-KiB")
	b.ReportMetric(float64(argon2MemoryLimitKiB), "limit-KiB")
	b.ReportMetric(float64(argon2Iterations), "iterations")
	b.ReportMetric(float64(argon2Parallelism), "parallelism")
}

func scanValues(destinations []any, values []any) error {
	if len(destinations) != len(values) {
		return fmt.Errorf("scan destination count %d != value count %d", len(destinations), len(values))
	}
	for index := range destinations {
		target := reflect.ValueOf(destinations[index])
		if target.Kind() != reflect.Ptr || target.IsNil() {
			return fmt.Errorf("scan destination %d is not a pointer", index)
		}
		if err := assignValue(target.Elem(), values[index]); err != nil {
			return err
		}
	}
	return nil
}

func assignValue(target reflect.Value, value any) error {
	if value == nil {
		target.Set(reflect.Zero(target.Type()))
		return nil
	}
	if target.Kind() == reflect.Ptr {
		pointer := reflect.New(target.Type().Elem())
		if err := assignValue(pointer.Elem(), value); err != nil {
			return err
		}
		target.Set(pointer)
		return nil
	}
	source := reflect.ValueOf(value)
	if source.Type().AssignableTo(target.Type()) {
		target.Set(source)
		return nil
	}
	if source.Type().ConvertibleTo(target.Type()) {
		target.Set(source.Convert(target.Type()))
		return nil
	}
	return fmt.Errorf("cannot assign %T to %s", value, target.Type())
}

type valuesRow struct {
	values []any
	err    error
}

func (row valuesRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	return scanValues(destinations, row.values)
}

type rowsStub struct {
	pgx.Rows
	rows    [][]any
	index   int
	current []any
	err     error
}

func (rows *rowsStub) Close() {}

func (rows *rowsStub) Err() error { return rows.err }

func (rows *rowsStub) Next() bool {
	if rows.index >= len(rows.rows) {
		return false
	}
	rows.current = rows.rows[rows.index]
	rows.index++
	return true
}

func (rows *rowsStub) Scan(destinations ...any) error {
	return scanValues(destinations, rows.current)
}

type countingExecutor struct {
	executions atomic.Int64
}

func (executor *countingExecutor) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	executor.executions.Add(1)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

type sessionCounters struct {
	preReads     atomic.Int64
	lockQueries  atomic.Int64
	gateQueries  atomic.Int64
	sessionReads atomic.Int64
	begins       atomic.Int64
}

type sessionTransactor struct {
	counters *sessionCounters
	resolve  *sessionResolveTx
}

func (transactor *sessionTransactor) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	if transactor.counters != nil {
		transactor.counters.preReads.Add(1)
	}
	return valuesRow{values: []any{"user-1"}}
}

func (transactor *sessionTransactor) Begin(context.Context) (pgx.Tx, error) {
	if transactor.counters != nil {
		transactor.counters.begins.Add(1)
	}
	if transactor.resolve != nil {
		return transactor.resolve, nil
	}
	return &sessionResolveTx{}, nil
}

type sessionResolveTx struct {
	pgx.Tx
	counters *sessionCounters
}

func (*sessionResolveTx) Commit(context.Context) error { return nil }

func (*sessionResolveTx) Rollback(context.Context) error { return nil }

func (tx *sessionResolveTx) QueryRow(
	_ context.Context,
	query string,
	_ ...any,
) pgx.Row {
	switch {
	case strings.Contains(query, "current_account"):
		if tx.counters != nil {
			tx.counters.gateQueries.Add(1)
		}
		return valuesRow{values: []any{true}}
	case strings.Contains(query, "FROM users"):
		if tx.counters != nil {
			tx.counters.lockQueries.Add(1)
		}
		return valuesRow{values: []any{"user-1"}}
	case strings.Contains(query, "FROM sessions AS s"):
		if tx.counters != nil {
			tx.counters.sessionReads.Add(1)
		}
		csrfDigest := sha256.Sum256([]byte("csrf-token"))
		return valuesRow{values: []any{
			"session-1",
			"user-1",
			"admin",
			false,
			"active",
			"permanent",
			nil,
			nil,
			"v1",
			csrfDigest[:],
			"permanent",
			nil,
			true,
		}}
	default:
		return valuesRow{err: fmt.Errorf("unexpected session query")}
	}
}

func BenchmarkSessionCreate(b *testing.B) {
	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      &sessionTransactor{},
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		b.Fatalf("create session store: %v", err)
	}
	executor := &countingExecutor{}
	input := session.CreateInput{
		ID:                          "session-1",
		UserID:                      "user-1",
		Token:                       "session-token",
		CSRFToken:                   "csrf-token",
		AuthenticatedCredentialKind: session.CredentialPermanent,
		AuthGeneration:              1,
		IssuedAt:                    benchmarkNow,
		AbsoluteExpiresAt:           benchmarkNow.Add(24 * time.Hour),
		IdleExpiresAt:               benchmarkNow.Add(30 * time.Minute),
	}
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		_, err := store.Create(context.Background(), executor, input)
		return err
	})
	reportLatencyMetrics(b, result)
	b.ReportMetric(float64(executor.executions.Load())/float64(b.N), "statements/op")
}

func BenchmarkSessionRevoke(b *testing.B) {
	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      &sessionTransactor{},
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		b.Fatalf("create session store: %v", err)
	}
	executor := &countingExecutor{}
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		return store.Revoke(
			context.Background(),
			executor,
			"session-token",
			benchmarkNow,
			"benchmark",
		)
	})
	reportLatencyMetrics(b, result)
	b.ReportMetric(float64(executor.executions.Load())/float64(b.N), "statements/op")
}

func BenchmarkSessionResolvePrincipal(b *testing.B) {
	counters := &sessionCounters{}
	transactor := &sessionTransactor{
		counters: counters,
		resolve:  &sessionResolveTx{counters: counters},
	}
	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      transactor,
		PasswordResetGate: identity.PasswordResetStore{},
	})
	if err != nil {
		b.Fatalf("create session store: %v", err)
	}
	input := session.ResolveInput{
		Token:       "session-token",
		RequireCSRF: false,
		Now:         benchmarkNow,
	}
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		_, err := store.ResolvePrincipal(context.Background(), input)
		return err
	})
	reportLatencyMetrics(b, result)
	gateCalls := counters.gateQueries.Load()
	if gateCalls != int64(b.N) {
		b.Fatalf("password reset Gate calls = %d, want one per op (%d)", gateCalls, b.N)
	}
	totalQueries := counters.preReads.Load() + counters.lockQueries.Load() + gateCalls + counters.sessionReads.Load()
	if expected := int64(4) * int64(b.N); totalQueries != expected {
		b.Fatalf("session resolve queries = %d, want %d", totalQueries, expected)
	}
	b.ReportMetric(float64(counters.begins.Load())/float64(b.N), "transactions/op")
	b.ReportMetric(float64(totalQueries)/float64(b.N), "queries/op")
}

type fixedClock struct {
	now time.Time
}

func (clock fixedClock) Now() time.Time { return clock.now }

type sequentialIDGenerator struct {
	next atomic.Int64
}

func (generator *sequentialIDGenerator) NewID() (string, error) {
	return benchmarkUUID(int(generator.next.Add(1))), nil
}

type countingAudit struct {
	events atomic.Int64
}

func (appender *countingAudit) Append(context.Context, pgx.Tx, audit.Event) error {
	appender.events.Add(1)
	return nil
}

type benchTransactor struct {
	begins atomic.Int64
}

func (transactor *benchTransactor) Begin(context.Context) (pgx.Tx, error) {
	transactor.begins.Add(1)
	return benchTx{}, nil
}

type benchTx struct {
	pgx.Tx
}

func (benchTx) Commit(context.Context) error { return nil }

func (benchTx) Rollback(context.Context) error { return nil }

type governanceRepository struct {
	actor       identity.User
	target      identity.User
	locks       atomic.Int64
	updates     atomic.Int64
	revocations atomic.Int64
}

func (repository *governanceRepository) LockUser(
	_ context.Context,
	_ pgx.Tx,
	userID string,
) (identity.User, error) {
	repository.locks.Add(1)
	switch userID {
	case repository.actor.ID:
		return repository.actor, nil
	case repository.target.ID:
		return repository.target, nil
	default:
		return identity.User{}, pgx.ErrNoRows
	}
}

func (repository *governanceRepository) UpdateRole(
	context.Context,
	pgx.Tx,
	string,
	identity.Role,
	identity.Role,
	time.Time,
) error {
	repository.updates.Add(1)
	return nil
}

func (repository *governanceRepository) RevokeSessions(
	context.Context,
	pgx.Tx,
	string,
	time.Time,
	string,
) error {
	repository.revocations.Add(1)
	return nil
}

func BenchmarkGovernanceRevokeAdministrator(b *testing.B) {
	actor := identity.User{
		ID:               benchmarkUUID(1),
		Role:             identity.RoleAdmin,
		IsInitialAdmin:   true,
		Status:           identity.UserStatusActive,
		CredentialMode:   identity.CredentialModePermanent,
		RecoveryRequired: false,
	}
	target := identity.User{
		ID:             benchmarkUUID(2),
		Role:           identity.RoleAdmin,
		IsInitialAdmin: false,
		Status:         identity.UserStatusActive,
		CredentialMode: identity.CredentialModePermanent,
	}
	transactor := &benchTransactor{}
	repository := &governanceRepository{actor: actor, target: target}
	appender := &countingAudit{}
	coordinator, err := identity.NewAdministratorGovernanceService(
		identity.AdministratorGovernanceDependencies{
			Transactions: transactor,
			Repository:   repository,
			Audit:        appender,
			IDs:          &sequentialIDGenerator{},
			Clock:        fixedClock{now: benchmarkNow},
		},
	)
	if err != nil {
		b.Fatalf("create administrator governance service: %v", err)
	}
	request := identity.AdministratorRevokeRequest{
		Actor: session.Principal{
			UserID:         actor.ID,
			Role:           string(identity.RoleAdmin),
			Status:         string(identity.UserStatusActive),
			IsInitialAdmin: true,
		},
		TargetID: target.ID,
		TraceID:  "benchmark-trace",
	}
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		_, err := coordinator.RevokeAdministrator(context.Background(), request)
		return err
	})
	reportLatencyMetrics(b, result)
	b.ReportMetric(float64(transactor.begins.Load())/float64(b.N), "transactions/op")
	b.ReportMetric(float64(repository.locks.Load())/float64(b.N), "locks/op")
	b.ReportMetric(float64(repository.updates.Load())/float64(b.N), "updates/op")
	b.ReportMetric(float64(repository.revocations.Load())/float64(b.N), "session-revokes/op")
	b.ReportMetric(float64(appender.events.Load())/float64(b.N), "audit-events/op")
}

type lockedUserTx struct {
	pgx.Tx
	queries *atomic.Int64
}

func (tx lockedUserTx) QueryRow(context.Context, string, ...any) pgx.Row {
	if tx.queries != nil {
		tx.queries.Add(1)
	}
	return valuesRow{values: []any{true}}
}

func BenchmarkPasswordResetGateCheck(b *testing.B) {
	store := identity.PasswordResetStore{}
	queries := &atomic.Int64{}
	tx := lockedUserTx{queries: queries}
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		return store.CheckLockedUser(context.Background(), tx, benchmarkUUID(3))
	})
	reportLatencyMetrics(b, result)
	b.ReportMetric(float64(queries.Load())/float64(b.N), "queries/op")
}

const workerBenchmarkID = "00000000-0000-4000-8000-000000000004"

type workerCounters struct {
	scanQueries      atomic.Int64
	scanned          atomic.Int64
	begins           atomic.Int64
	locks            atomic.Int64
	claimAttempts    atomic.Int64
	claimed          atomic.Int64
	renews           atomic.Int64
	renewed          atomic.Int64
	settlementBegins atomic.Int64
	settlementExecs  atomic.Int64
	settled          atomic.Int64
	failExecs        atomic.Int64
	failed           atomic.Int64
	rollbacks        atomic.Int64
	unexpected       atomic.Int64
	lastOperation    atomic.Value
	lastFailStatus   atomic.Value
}

func (counters *workerCounters) operation() string {
	value := counters.lastOperation.Load()
	if value == nil {
		return "benchmark-operation"
	}
	operation, _ := value.(string)
	if operation == "" {
		return "benchmark-operation"
	}
	return operation
}

type workerTx struct {
	pgx.Tx
	counters          *workerCounters
	claimAvailable    bool
	settlementFailure error
}

func (*workerTx) Commit(context.Context) error { return nil }

func (tx *workerTx) Rollback(context.Context) error {
	tx.counters.rollbacks.Add(1)
	return nil
}

func (tx *workerTx) QueryRow(
	_ context.Context,
	query string,
	arguments ...any,
) pgx.Row {
	switch {
	case strings.Contains(query, "SELECT auth_generation, credential_mode, password_reset_current_operation_id FROM users"):
		tx.counters.locks.Add(1)
		return valuesRow{values: []any{int64(1), "permanent", tx.counters.operation()}}
	case strings.Contains(query, "FROM password_reset_operations o JOIN password_reset_recovery_jobs"):
		tx.counters.locks.Add(1)
		return valuesRow{values: []any{
			identity.PasswordResetOperationPrepared,
			identity.PasswordResetDeliveryNotStarted,
			identity.PasswordResetSettlementPending,
			"benchmark-credential",
			identity.PasswordResetInitialReset,
			nil,
			int64(1),
			int64(2),
			[]byte("scope"),
			[]byte("locator"),
			[]byte("key-digest"),
			[]byte("request-digest"),
			"v1",
			int64(1),
			int64(1),
			workerBenchmarkID,
			benchmarkNow.Add(time.Second),
		}}
	case strings.Contains(query, "SELECT id FROM users WHERE id=$1 FOR UPDATE"):
		tx.counters.locks.Add(1)
		if len(arguments) > 0 {
			if userID, ok := arguments[0].(string); ok {
				return valuesRow{values: []any{userID}}
			}
		}
		return valuesRow{err: pgx.ErrNoRows}
	case strings.Contains(query, "SET status='leased'"):
		tx.counters.claimAttempts.Add(1)
		if !tx.claimAvailable {
			return valuesRow{err: pgx.ErrNoRows}
		}
		if len(arguments) > 0 {
			if operation, ok := arguments[0].(string); ok {
				tx.counters.lastOperation.Store(operation)
			}
		}
		tx.counters.claimed.Add(1)
		return valuesRow{values: []any{int64(1)}}
	default:
		tx.counters.unexpected.Add(1)
		return valuesRow{err: fmt.Errorf("unexpected worker query")}
	}
}

func (tx *workerTx) Exec(
	_ context.Context,
	query string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	switch {
	case strings.Contains(query, "SET lease_expires_at=$1,updated_at=$2"):
		tx.counters.renews.Add(1)
		tx.counters.renewed.Add(1)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	case strings.Contains(query, "SET status=$1,next_attempt_at=$2,last_failure_code"):
		tx.counters.failExecs.Add(1)
		if len(arguments) > 0 {
			if status, ok := arguments[0].(identity.PasswordResetRecoveryJobStatus); ok {
				tx.counters.lastFailStatus.Store(string(status))
			}
		}
		tx.counters.failed.Add(1)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	case strings.Contains(query, "SET state='recovery_invalidated'"):
		if tx.settlementFailure != nil {
			return pgconn.CommandTag{}, tx.settlementFailure
		}
		tx.counters.settlementExecs.Add(1)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	case strings.Contains(query, "SET state='invalidated',settlement_state='invalidated'"):
		tx.counters.settlementExecs.Add(1)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	case strings.Contains(query, "UPDATE idempotency_records"):
		tx.counters.settlementExecs.Add(1)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	case strings.Contains(query, "SET status='succeeded',lease_owner=NULL"):
		tx.counters.settlementExecs.Add(1)
		tx.counters.settled.Add(1)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	default:
		tx.counters.unexpected.Add(1)
		return pgconn.CommandTag{}, fmt.Errorf("unexpected worker exec")
	}
}

type workerPool struct {
	counters       *workerCounters
	candidates     [][]any
	claimAvailable bool
}

func (pool *workerPool) Query(context.Context, string, ...any) (pgx.Rows, error) {
	pool.counters.scanQueries.Add(1)
	pool.counters.scanned.Add(int64(len(pool.candidates)))
	candidates := make([][]any, len(pool.candidates))
	copy(candidates, pool.candidates)
	return &rowsStub{rows: candidates}, nil
}

func (pool *workerPool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	pool.counters.begins.Add(1)
	return &workerTx{counters: pool.counters, claimAvailable: pool.claimAvailable}, nil
}

type workerTransactor struct {
	counters          *workerCounters
	settlementFailure error
}

func (transactor *workerTransactor) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	transactor.counters.settlementBegins.Add(1)
	return &workerTx{counters: transactor.counters, settlementFailure: transactor.settlementFailure}, nil
}

func workerCandidates() [][]any {
	candidates := make([][]any, workerClaimBatch)
	for index := range candidates {
		candidates[index] = []any{benchmarkUUID(1000 + index), benchmarkUUID(2000 + index)}
	}
	return candidates
}

func newWorkerBenchmark(
	b *testing.B,
	pool *workerPool,
	maxRetries int64,
	settlementFailure error,
) *identity.PasswordResetRecoveryWorker {
	b.Helper()
	permit, err := identity.NewPasswordResetConnectionPermit(argon2WorkerLimit)
	if err != nil {
		b.Fatalf("create password reset connection permit: %v", err)
	}
	store := &identity.PasswordResetStore{}
	operator, err := identity.NewPasswordResetOperator(
		identity.PasswordResetOperatorDependencies{
			Transactions: &workerTransactor{counters: pool.counters, settlementFailure: settlementFailure},
			Store:        store,
			Audit:        &countingAudit{},
			IDs:          &sequentialIDGenerator{},
			Clock:        fixedClock{now: benchmarkNow},
			Permit:       permit,
		},
	)
	if err != nil {
		b.Fatalf("create password reset operator: %v", err)
	}
	worker, err := identity.NewPasswordResetRecoveryWorker(
		identity.PasswordResetWorkerDependencies{
			Pool:     pool,
			Store:    store,
			Operator: operator,
			Permit:   permit,
			WorkerID: workerBenchmarkID,
			Clock:    fixedClock{now: benchmarkNow},
		},
		identity.PasswordResetWorkerConfig{
			LeaseDuration:    time.Second,
			RetryBackoff:     time.Second,
			MaxRetries:       maxRetries,
			ClaimBatchLimit:  workerClaimBatch,
			ConnectionBudget: argon2WorkerLimit,
		},
	)
	if err != nil {
		b.Fatalf("create password reset worker: %v", err)
	}
	return worker
}

func reportWorkerMetrics(b *testing.B, counters *workerCounters) {
	b.Helper()
	operations := float64(b.N)
	b.ReportMetric(float64(counters.scanQueries.Load())/operations, "scan-queries/op")
	b.ReportMetric(float64(counters.scanned.Load())/operations, "candidates/op")
	b.ReportMetric(float64(counters.locks.Load())/operations, "locks/op")
	b.ReportMetric(float64(counters.claimAttempts.Load())/operations, "claim-attempts/op")
	b.ReportMetric(float64(counters.claimed.Load())/operations, "claimed/op")
	b.ReportMetric(float64(counters.renews.Load())/operations, "renews/op")
	b.ReportMetric(float64(counters.renewed.Load())/operations, "renewed/op")
	b.ReportMetric(float64(counters.settled.Load())/operations, "settled/op")
	b.ReportMetric(float64(counters.failed.Load())/operations, "failed/op")
	b.ReportMetric(float64(counters.settlementExecs.Load())/operations, "settlement-execs/op")
	b.ReportMetric(float64(counters.begins.Load()+counters.settlementBegins.Load())/operations, "transactions/op")
	b.ReportMetric(float64(counters.rollbacks.Load())/operations, "rollbacks/op")
}

func BenchmarkPasswordResetWorkerEmptyScan(b *testing.B) {
	counters := &workerCounters{}
	pool := &workerPool{counters: counters}
	worker := newWorkerBenchmark(b, pool, 3, nil)
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		return worker.Reconcile(context.Background())
	})
	reportLatencyMetrics(b, result)
	reportWorkerMetrics(b, counters)
}

func BenchmarkPasswordResetWorkerScanCandidates100(b *testing.B) {
	counters := &workerCounters{}
	pool := &workerPool{counters: counters, candidates: workerCandidates()}
	worker := newWorkerBenchmark(b, pool, 3, nil)
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		return worker.Reconcile(context.Background())
	})
	reportLatencyMetrics(b, result)
	expectedCandidates := int64(workerClaimBatch) * int64(b.N)
	if got := counters.scanned.Load(); got != expectedCandidates {
		b.Fatalf("scanned candidates = %d, want %d", got, expectedCandidates)
	}
	if got := counters.claimed.Load(); got != 0 {
		b.Fatalf("claimed jobs = %d, want 0 for claim-miss scan", got)
	}
	if got := counters.settled.Load(); got != 0 {
		b.Fatalf("settled jobs = %d, want 0 for claim-miss scan", got)
	}
	if got := counters.unexpected.Load(); got != 0 {
		b.Fatalf("unexpected fake calls = %d", got)
	}
	reportWorkerMetrics(b, counters)
}

func BenchmarkPasswordResetWorkerClaimBatch100(b *testing.B) {
	counters := &workerCounters{}
	pool := &workerPool{counters: counters, candidates: workerCandidates(), claimAvailable: true}
	worker := newWorkerBenchmark(b, pool, 3, nil)
	b.ReportAllocs()
	result := runConcurrent(b, 1, false, func(int) error {
		return worker.Reconcile(context.Background())
	})
	reportLatencyMetrics(b, result)
	expected := int64(workerClaimBatch) * int64(b.N)
	if got := counters.claimed.Load(); got != expected {
		b.Fatalf("claimed jobs = %d, want %d", got, expected)
	}
	if got := counters.renewed.Load(); got != expected {
		b.Fatalf("renewed leases = %d, want %d", got, expected)
	}
	if got := counters.settled.Load(); got != expected {
		b.Fatalf("settled jobs = %d, want %d", got, expected)
	}
	if got := counters.failed.Load(); got != 0 {
		b.Fatalf("failed jobs = %d, want 0 on success path", got)
	}
	if got := counters.unexpected.Load(); got != 0 {
		b.Fatalf("unexpected fake calls = %d", got)
	}
	if result.failures != 0 {
		b.Fatalf("reconcile failures = %d", result.failures)
	}
	reportWorkerMetrics(b, counters)
}

func BenchmarkPasswordResetWorkerSettlementRetry100(b *testing.B) {
	for _, scenario := range []struct {
		name       string
		maxRetries int64
		status     string
	}{
		{name: "retry_wait", maxRetries: 3, status: "retry_wait"},
		{name: "dead_letter", maxRetries: 1, status: "dead_letter"},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			counters := &workerCounters{}
			pool := &workerPool{counters: counters, candidates: workerCandidates(), claimAvailable: true}
			worker := newWorkerBenchmark(b, pool, scenario.maxRetries, errors.New("synthetic settlement failure"))
			b.ReportAllocs()
			result := runConcurrent(b, 1, false, func(int) error {
				return worker.Reconcile(context.Background())
			})
			reportLatencyMetrics(b, result)
			expected := int64(workerClaimBatch) * int64(b.N)
			if got := counters.claimed.Load(); got != expected {
				b.Fatalf("claimed jobs = %d, want %d", got, expected)
			}
			if got := counters.renewed.Load(); got != expected {
				b.Fatalf("renewed leases = %d, want %d", got, expected)
			}
			if got := counters.settled.Load(); got != 0 {
				b.Fatalf("settled jobs = %d, want 0 on failure path", got)
			}
			if got := counters.failed.Load(); got != expected {
				b.Fatalf("failed jobs = %d, want %d", got, expected)
			}
			status, _ := counters.lastFailStatus.Load().(string)
			if status != scenario.status {
				b.Fatalf("failure status = %q, want %q", status, scenario.status)
			}
			if got := counters.unexpected.Load(); got != 0 {
				b.Fatalf("unexpected fake calls = %d", got)
			}
			if result.failures != 0 {
				b.Fatalf("reconcile failures = %d", result.failures)
			}
			reportWorkerMetrics(b, counters)
		})
	}
}

var (
	smtpTLSOnce  sync.Once
	smtpTLSCert  tls.Certificate
	smtpTLSRoots *x509.CertPool
	smtpTLSErr   error
)

func smtpTLSMaterial() (tls.Certificate, *x509.CertPool, error) {
	smtpTLSOnce.Do(func() {
		smtpTLSCert, smtpTLSRoots, smtpTLSErr = generateSMTPTLS()
	})
	return smtpTLSCert, smtpTLSRoots, smtpTLSErr
}

func generateSMTPTLS() (tls.Certificate, *x509.CertPool, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, roots, nil
}

type smtpPipeDialer struct {
	cert    tls.Certificate
	dials   atomic.Int64
	current atomic.Int64
	peak    atomic.Int64
}

func (dialer *smtpPipeDialer) DialContext(
	ctx context.Context,
	_ string,
	_ string,
) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client, server := net.Pipe()
	dialer.dials.Add(1)
	current := dialer.current.Add(1)
	for {
		peak := dialer.peak.Load()
		if current <= peak || dialer.peak.CompareAndSwap(peak, current) {
			break
		}
	}
	go func() {
		defer dialer.current.Add(-1)
		serveSMTPConnection(server, dialer.cert)
	}()
	return client, nil
}

func serveSMTPConnection(connection net.Conn, certificate tls.Certificate) {
	defer connection.Close()
	server := tls.Server(connection, &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
	})
	if err := server.Handshake(); err != nil {
		return
	}
	reader := bufio.NewReader(server)
	writer := bufio.NewWriter(server)
	respond := func(line string) bool {
		if _, err := writer.WriteString(line); err != nil {
			return false
		}
		return writer.Flush() == nil
	}
	if !respond("220 localhost ESMTP\r\n") {
		return
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			if !respond("250-localhost\r\n250 HELP\r\n") {
				return
			}
		case strings.HasPrefix(command, "MAIL FROM"),
			strings.HasPrefix(command, "RCPT TO"),
			strings.HasPrefix(command, "RSET"):
			if !respond("250 OK\r\n") {
				return
			}
		case command == "DATA":
			if !respond("354 End data with <CR><LF>.<CR><LF>\r\n") {
				return
			}
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
			}
			if !respond("250 OK\r\n") {
				return
			}
		case command == "QUIT":
			_ = respond("221 Bye\r\n")
			return
		default:
			if !respond("250 OK\r\n") {
				return
			}
		}
	}
}

func BenchmarkSMTPImplicitTLSInMemory(b *testing.B) {
	certificate, roots, err := smtpTLSMaterial()
	if err != nil {
		b.Fatalf("create SMTP TLS material: %v", err)
	}
	for concurrency := 1; concurrency <= argon2WorkerLimit; concurrency++ {
		b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
			benchmarkSMTP(b, concurrency, certificate, roots)
		})
	}
}

func benchmarkSMTP(
	b *testing.B,
	concurrency int,
	certificate tls.Certificate,
	roots *x509.CertPool,
) {
	b.Helper()
	dialer := &smtpPipeDialer{cert: certificate}
	client, err := mailer.NewSMTPMailer(
		mailer.SMTPConfig{
			Address: "127.0.0.1:2525",
			Host:    "localhost",
			From:    "noreply@example.test",
			Mode:    mailer.TLSModeImplicit,
			TLSConfig: &tls.Config{
				ServerName: "localhost",
				MinVersion: tls.VersionTLS12,
				RootCAs:    roots,
			},
		},
		dialer,
	)
	if err != nil {
		b.Fatalf("create SMTP mailer: %v", err)
	}
	message := mailer.Message{
		Purpose: mailer.PurposePasswordResetCredential,
		To:      "user@example.test",
		Subject: "Benchmark reset credential",
		Body:    "token=benchmark",
	}
	b.ReportAllocs()
	result := runConcurrent(b, concurrency, false, func(int) error {
		ctx, cancel := context.WithTimeout(context.Background(), smtpOperationTimeout)
		defer cancel()
		return client.Send(ctx, message)
	})
	reportLatencyMetrics(b, result)
	b.ReportMetric(float64(concurrency), "workers")
	b.ReportMetric(float64(dialer.dials.Load())/float64(b.N), "connections/op")
	b.ReportMetric(float64(dialer.peak.Load()), "peak-connections")
}

package identity

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"han/internal/audit"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestPasswordResetOperatorPermitHeldAndReleased(t *testing.T) {
	permit, err := NewPasswordResetConnectionPermit(1)
	if err != nil {
		t.Fatalf("NewPasswordResetConnectionPermit() error = %v", err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error {
		close(entered)
		<-release
		return nil
	})
	result := make(chan error, 1)
	go func() { result <- operator.Execute(context.Background(), passwordResetTestLease()) }()
	<-entered
	stats := permit.Stats()
	if stats.Current != 1 || stats.Peak != 1 || stats.Acquired != 1 || stats.Released != 0 {
		t.Fatalf("permit while operation active = %+v", stats)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	stats = permit.Stats()
	if stats.Current != 0 || stats.Acquired != 1 || stats.Released != 1 {
		t.Fatalf("permit after Execute = %+v", stats)
	}
}

func TestPasswordResetOperatorPermitReturnedOnErrorCancelAndPanic(t *testing.T) {
	cases := []struct {
		name string
		run  func(*PasswordResetOperator) error
	}{
		{name: "error", run: func(operator *PasswordResetOperator) error {
			return operator.Execute(context.Background(), passwordResetTestLease())
		}},
		{name: "cancel", run: func(operator *PasswordResetOperator) error {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return operator.Execute(ctx, passwordResetTestLease())
		}},
		{name: "panic", run: func(operator *PasswordResetOperator) (err error) {
			defer func() {
				if recover() == nil {
					t.Fatal("Execute() did not panic")
				}
			}()
			return operator.Execute(context.Background(), passwordResetTestLease())
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			permit, err := NewPasswordResetConnectionPermit(1)
			if err != nil {
				t.Fatal(err)
			}
			settleErr := errors.New("synthetic settlement failure")
			operator := newPasswordResetOperatorForTest(permit, func(ctx context.Context, _ PasswordResetRecoveryLease) error {
				if test.name == "panic" {
					panic("synthetic panic")
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				return settleErr
			})
			_ = test.run(operator)
			stats := permit.Stats()
			if stats.Current != 0 || stats.Acquired != stats.Released || stats.Peak > stats.Budget {
				t.Fatalf("permit stats = %+v", stats)
			}
		})
	}
}

func TestPasswordResetOperatorHeldHelperDoesNotAcquire(t *testing.T) {
	permit, err := NewPasswordResetConnectionPermit(1)
	if err != nil {
		t.Fatal(err)
	}
	operator := newPasswordResetOperatorForTest(permit, func(context.Context, PasswordResetRecoveryLease) error { return nil })
	release, err := permit.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := operator.executeHeld(context.Background(), passwordResetTestLease()); err != nil {
		t.Fatalf("executeHeld() error = %v", err)
	}
	release()
	stats := permit.Stats()
	if stats.Acquired != 1 || stats.Released != 1 || stats.Peak != 1 {
		t.Fatalf("nested acquire detected: %+v", stats)
	}
}

func TestPasswordResetConnectionPermitConcurrentPeakNeverExceedsBudget(t *testing.T) {
	permit, err := NewPasswordResetConnectionPermit(2)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	entered := make(chan struct{}, 4)
	releaseAll := make(chan struct{})
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			release, acquireErr := permit.Acquire(context.Background())
			if acquireErr != nil {
				t.Errorf("Acquire() error = %v", acquireErr)
				return
			}
			defer release()
			entered <- struct{}{}
			<-releaseAll
		}()
	}
	<-entered
	<-entered
	stats := permit.Stats()
	if stats.Current != 2 || stats.Peak != 2 {
		t.Fatalf("permit peak = %+v", stats)
	}
	close(releaseAll)
	group.Wait()
	stats = permit.Stats()
	if stats.Current != 0 || stats.Acquired != 4 || stats.Released != 4 || stats.Peak > stats.Budget {
		t.Fatalf("final stats = %+v", stats)
	}
}

func TestPasswordResetOperatorAcceptedSettlementExecutesCompleteTransaction(t *testing.T) {
	now := time.Date(2026, 10, 25, 2, 0, 0, 0, time.UTC)
	lease := passwordResetTestLease()
	credentialID := "00000000-0000-0000-0000-000000000103"
	owner := lease.Owner
	currentOperation := lease.OperationID
	sourceOperation := lease.OperationID
	expires := now.Add(time.Minute)
	scope := bytesOf(0x11)
	locator := bytesOf(0x22)
	keyDigest := bytesOf(0x33)
	requestDigest := bytesOf(0x44)
	state := struct {
		credentialState TemporaryCredentialState
		userGeneration  int64
		activeSessions  int
		operationState  PasswordResetOperationState
		idempotency     string
		job             PasswordResetRecoveryJobStatus
		auditCount      int
	}{
		credentialState: TemporaryCredentialPendingDelivery,
		activeSessions:  2,
		operationState:  PasswordResetOperationRecoveryRequired,
		idempotency:     "processing",
		job:             PasswordResetRecoveryJobLeased,
	}
	tx := &passwordResetScriptTx{onExec: func(sql string, _ []any) {
		switch {
		case strings.Contains(sql, "SET state='converging'"):
			state.operationState = PasswordResetOperationConverging
		case strings.Contains(sql, "SET state='active'"):
			state.credentialState = TemporaryCredentialActive
		case strings.Contains(sql, "UPDATE users SET auth_generation"):
			state.userGeneration = 1
		case strings.Contains(sql, "UPDATE sessions SET revoked_at"):
			state.activeSessions = 0
		case strings.Contains(sql, "UPDATE password_reset_operations SET state='succeeded'"):
			state.operationState = PasswordResetOperationSucceeded
		case strings.Contains(sql, "UPDATE idempotency_records SET status"):
			state.idempotency = "succeeded"
		case strings.Contains(sql, "UPDATE password_reset_recovery_jobs SET status='succeeded'"):
			state.job = PasswordResetRecoveryJobSucceeded
		case strings.Contains(sql, "INSERT INTO audit_events"):
			state.auditCount++
		}
	}, steps: []passwordResetSQLStep{
		queryStep("FROM users WHERE id = $1 FOR UPDATE", []any{int64(0), CredentialModePermanent, &currentOperation}),
		queryStep("FROM password_reset_operations o JOIN password_reset_recovery_jobs", []any{
			PasswordResetOperationRecoveryRequired, PasswordResetDeliveryAccepted, PasswordResetSettlementPending,
			credentialID, PasswordResetInitialReset, (*string)(nil), int64(0), int64(1),
			scope, locator, keyDigest, requestDigest, "v1", int64(1), int64(1), &owner, &expires,
		}),
		execStep("SET state='converging'", []any{now, lease.OperationID}, 1),
		queryStep("FROM temporary_credentials WHERE id=$1 FOR UPDATE", []any{lease.UserID, TemporaryCredentialPendingDelivery, &sourceOperation, int64(1)}),
		execStep("SET state='active'", []any{credentialID, lease.UserID}, 1),
		execStep("UPDATE users SET auth_generation", []any{int64(1), now, lease.UserID, int64(0), lease.OperationID}, 1),
		execStep("UPDATE sessions SET revoked_at", []any{now, lease.UserID}, 2),
		execStep("UPDATE password_reset_operations SET state='succeeded'", []any{int64(1), now, lease.OperationID}, 1),
		execStep("UPDATE idempotency_records SET status", nil, 1),
		execStep("INSERT INTO audit_events", nil, 1),
		execStep("UPDATE password_reset_recovery_jobs SET status='succeeded'", []any{now, lease.OperationID, lease.Owner, lease.AttemptCount}, 1),
	}}
	starter := &passwordResetScriptStarter{transactions: []*passwordResetScriptTx{tx}}
	store := &PasswordResetStore{}
	err := store.settlePasswordResetRecovery(
		context.Background(),
		starter,
		audit.Store{},
		passwordResetStaticID("00000000-0000-0000-0000-000000000104"),
		passwordResetTestClock{now: now},
		lease,
	)
	if err != nil {
		t.Fatalf("settlePasswordResetRecovery() error = %v", err)
	}
	tx.assertComplete(t)
	if tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("commit/rollback = %d/%d", tx.commits, tx.rollbacks)
	}
	if state.credentialState != TemporaryCredentialActive ||
		state.userGeneration != 1 ||
		state.activeSessions != 0 ||
		state.operationState != PasswordResetOperationSucceeded ||
		state.idempotency != "succeeded" ||
		state.job != PasswordResetRecoveryJobSucceeded ||
		state.auditCount != 1 {
		t.Fatalf("accepted transition state = %+v", state)
	}
}

func TestPasswordResetOperatorExpiredLeaseAfterUserLockWritesNothing(t *testing.T) {
	beforeLock := time.Date(2026, 10, 25, 3, 0, 0, 0, time.UTC)
	atExpiry := beforeLock.Add(time.Minute)
	clock := &passwordResetMutableClock{now: beforeLock}
	lease := passwordResetTestLease()
	currentOperation := lease.OperationID
	owner := lease.Owner
	expires := atExpiry
	tx := &passwordResetScriptTx{
		onQuery: func(marker string) {
			if strings.Contains(marker, "FROM users WHERE id = $1 FOR UPDATE") {
				clock.Set(atExpiry)
			}
		},
		steps: []passwordResetSQLStep{
			queryStep("FROM users WHERE id = $1 FOR UPDATE", []any{int64(0), CredentialModePermanent, &currentOperation}),
			queryStep("FROM password_reset_operations o JOIN password_reset_recovery_jobs", []any{
				PasswordResetOperationRecoveryRequired, PasswordResetDeliveryAccepted, PasswordResetSettlementPending,
				"credential-1", PasswordResetInitialReset, (*string)(nil), int64(0), int64(1),
				bytesOf(1), bytesOf(2), bytesOf(3), bytesOf(4), "v1", int64(1), int64(1), &owner, &expires,
			}),
		},
	}
	err := (&PasswordResetStore{}).settlePasswordResetRecovery(
		context.Background(),
		&passwordResetScriptStarter{transactions: []*passwordResetScriptTx{tx}},
		passwordResetOperatorAuditStub{},
		passwordResetStaticID("audit-1"),
		clock,
		lease,
	)
	if !errors.Is(err, ErrPasswordResetStateConflict) {
		t.Fatalf("settlePasswordResetRecovery() error = %v", err)
	}
	if tx.executes != 0 || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("exec/commit/rollback = %d/%d/%d", tx.executes, tx.commits, tx.rollbacks)
	}
}

func TestPasswordResetOperatorSettlementRechecksLeaseBeforeFinalCompletion(t *testing.T) {
	startedAt := time.Date(2026, 10, 25, 3, 30, 0, 0, time.UTC)
	expiresAt := startedAt.Add(time.Minute)
	for _, test := range []struct {
		name             string
		completionAt     time.Time
		firstCompletion  int64
		wantFirstErr     bool
		wantSecondWorker bool
	}{
		{name: "before_expiry", completionAt: expiresAt.Add(-time.Microsecond), firstCompletion: 1},
		{name: "at_expiry", completionAt: expiresAt, wantFirstErr: true, wantSecondWorker: true},
		{name: "after_expiry", completionAt: expiresAt.Add(time.Microsecond), wantFirstErr: true, wantSecondWorker: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			clock := &passwordResetMutableClock{now: startedAt}
			firstLease := passwordResetTestLease()
			auditEntered := make(chan struct{})
			releaseAudit := make(chan struct{})
			firstTx := passwordResetAcceptedSettlementBoundaryTx(
				startedAt,
				test.completionAt,
				expiresAt,
				firstLease,
				test.firstCompletion,
			)
			firstTx.onExec = func(sql string, _ []any) {
				if strings.Contains(sql, "INSERT INTO audit_events") {
					close(auditEntered)
					<-releaseAudit
				}
			}
			transactions := []*passwordResetScriptTx{firstTx}
			var secondTx *passwordResetScriptTx
			secondLease := firstLease
			if test.wantSecondWorker {
				secondLease.Owner = "worker-2"
				secondLease.AttemptCount = 2
				secondExpiry := test.completionAt.Add(time.Minute)
				secondTx = passwordResetAcceptedSettlementBoundaryTx(
					test.completionAt,
					test.completionAt,
					secondExpiry,
					secondLease,
					1,
				)
				transactions = append(transactions, secondTx)
			}
			starter := &passwordResetScriptStarter{transactions: transactions}
			firstResult := make(chan error, 1)
			go func() {
				firstResult <- (&PasswordResetStore{}).settlePasswordResetRecovery(
					context.Background(),
					starter,
					audit.Store{},
					passwordResetStaticID("00000000-0000-0000-0000-000000000104"),
					clock,
					firstLease,
				)
			}()
			select {
			case <-auditEntered:
			case firstErr := <-firstResult:
				t.Fatalf("first settlement returned before audit barrier: %v", firstErr)
			}
			clock.Set(test.completionAt)
			close(releaseAudit)
			firstErr := <-firstResult
			if test.wantFirstErr {
				if !errors.Is(firstErr, ErrPasswordResetStateConflict) {
					t.Fatalf("first settlement error = %v", firstErr)
				}
				if firstTx.commits != 0 || firstTx.rollbacks != 1 {
					t.Fatalf("expired first settlement commit/rollback = %d/%d", firstTx.commits, firstTx.rollbacks)
				}
			} else {
				if firstErr != nil {
					t.Fatalf("first settlement error = %v", firstErr)
				}
				if firstTx.commits != 1 || firstTx.rollbacks != 0 {
					t.Fatalf("valid first settlement commit/rollback = %d/%d", firstTx.commits, firstTx.rollbacks)
				}
			}
			firstTx.assertComplete(t)
			if secondTx == nil {
				if starter.begins != 1 {
					t.Fatalf("BeginTx calls = %d, want 1", starter.begins)
				}
				return
			}
			if err := (&PasswordResetStore{}).settlePasswordResetRecovery(
				context.Background(),
				starter,
				audit.Store{},
				passwordResetStaticID("00000000-0000-0000-0000-000000000105"),
				clock,
				secondLease,
			); err != nil {
				t.Fatalf("takeover settlement error = %v", err)
			}
			secondTx.assertComplete(t)
			if secondTx.commits != 1 || secondTx.rollbacks != 0 {
				t.Fatalf("takeover settlement commit/rollback = %d/%d", secondTx.commits, secondTx.rollbacks)
			}
			if starter.begins != 2 || firstTx.commits+secondTx.commits != 1 {
				t.Fatalf("transactions/commits = %d/%d, want 2/1", starter.begins, firstTx.commits+secondTx.commits)
			}
		})
	}
}

func TestPasswordResetOperatorCommitAcknowledgementClassification(t *testing.T) {
	commitErr := errors.New("commit acknowledgement lost")
	verifyErr := errors.New("authority read unavailable")
	for _, test := range []struct {
		name         string
		authority    []any
		authorityErr error
		wantErr      error
		wantTxCount  int
	}{
		{
			name:        "committed",
			authority:   passwordResetCommittedAuthorityRow(),
			wantTxCount: 2,
		},
		{
			name:        "rolled_back",
			authority:   passwordResetRolledBackAuthorityRow(),
			wantErr:     ErrPasswordResetSettlementRolledBack,
			wantTxCount: 2,
		},
		{
			name:         "unknown",
			authorityErr: verifyErr,
			wantErr:      ErrPasswordResetSettlementCommitUnknown,
			wantTxCount:  2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := passwordResetAuthorityNow()
			lease := passwordResetTestLease()
			settlement := passwordResetAcceptedSettlementTx(now, lease)
			settlement.commitErr = commitErr
			verify := passwordResetAuthorityTx(lease, test.authority, test.authorityErr)
			starter := &passwordResetScriptStarter{transactions: []*passwordResetScriptTx{settlement, verify}}
			err := (&PasswordResetStore{}).settlePasswordResetRecovery(
				context.Background(),
				starter,
				audit.Store{},
				passwordResetStaticID("00000000-0000-0000-0000-000000000104"),
				passwordResetTestClock{now: now},
				lease,
			)
			if test.wantErr == nil && err != nil {
				t.Fatalf("settlePasswordResetRecovery() error = %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("settlePasswordResetRecovery() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr != nil && !errors.Is(err, commitErr) {
				t.Fatalf("settlePasswordResetRecovery() error = %v, want commit cause", err)
			}
			if test.authorityErr != nil && !errors.Is(err, test.authorityErr) {
				t.Fatalf("settlePasswordResetRecovery() error = %v, want verification cause", err)
			}
			if starter.begins != test.wantTxCount {
				t.Fatalf("BeginTx calls = %d, want %d", starter.begins, test.wantTxCount)
			}
			settlement.assertComplete(t)
			verify.assertComplete(t)
			if settlement.commits != 1 || settlement.rollbacks != 0 {
				t.Fatalf("settlement commit/rollback = %d/%d", settlement.commits, settlement.rollbacks)
			}
			wantVerifyCommit, wantVerifyRollback := 1, 0
			if test.authorityErr != nil {
				wantVerifyCommit, wantVerifyRollback = 0, 1
			}
			if verify.commits != wantVerifyCommit || verify.rollbacks != wantVerifyRollback {
				t.Fatalf("verify commit/rollback = %d/%d, want %d/%d", verify.commits, verify.rollbacks, wantVerifyCommit, wantVerifyRollback)
			}
		})
	}
}

func passwordResetTestLease() PasswordResetRecoveryLease {
	return PasswordResetRecoveryLease{OperationID: "00000000-0000-0000-0000-000000000101", UserID: "00000000-0000-0000-0000-000000000102", Owner: "worker-test", AttemptCount: 1}
}

type passwordResetStaticID string

func (id passwordResetStaticID) NewID() (string, error) { return string(id), nil }

type passwordResetOperatorAuditStub struct{}

func (passwordResetOperatorAuditStub) Append(context.Context, pgx.Tx, audit.Event) error {
	return nil
}

type passwordResetSQLStep struct {
	kind   string
	marker string
	args   []any
	row    []any
	tag    pgconn.CommandTag
	err    error
}

func queryStep(marker string, row []any) passwordResetSQLStep {
	return passwordResetSQLStep{kind: "query", marker: marker, row: row}
}

func execStep(marker string, args []any, rows int64) passwordResetSQLStep {
	return passwordResetSQLStep{kind: "exec", marker: marker, args: args, tag: pgconn.NewCommandTag(fmt.Sprintf("UPDATE %d", rows))}
}

type passwordResetScriptStarter struct {
	transactions []*passwordResetScriptTx
	begins       int
}

func (starter *passwordResetScriptStarter) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	if starter.begins >= len(starter.transactions) {
		return nil, errors.New("unexpected password reset transaction")
	}
	tx := starter.transactions[starter.begins]
	starter.begins++
	return tx, nil
}

type passwordResetScriptTx struct {
	pgx.Tx
	steps     []passwordResetSQLStep
	index     int
	executes  int
	commits   int
	rollbacks int
	commitErr error
	onQuery   func(string)
	onExec    func(string, []any)
}

func (tx *passwordResetScriptTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	step, err := tx.next("query", sql, args)
	if tx.onQuery != nil {
		tx.onQuery(sql)
	}
	if err != nil {
		return passwordResetScriptRow{err: err}
	}
	return passwordResetScriptRow{values: step.row, err: step.err}
}

func (tx *passwordResetScriptTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.executes++
	step, err := tx.next("exec", sql, args)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	if tx.onExec != nil {
		tx.onExec(compactPasswordResetSQL(sql), args)
	}
	return step.tag, step.err
}

func (tx *passwordResetScriptTx) Commit(context.Context) error {
	tx.commits++
	return tx.commitErr
}

func (tx *passwordResetScriptTx) Rollback(context.Context) error {
	tx.rollbacks++
	return nil
}

func (tx *passwordResetScriptTx) next(kind, sql string, args []any) (passwordResetSQLStep, error) {
	if tx.index >= len(tx.steps) {
		return passwordResetSQLStep{}, fmt.Errorf("unexpected %s SQL: %s", kind, compactPasswordResetSQL(sql))
	}
	step := tx.steps[tx.index]
	tx.index++
	if step.kind != kind || !strings.Contains(compactPasswordResetSQL(sql), compactPasswordResetSQL(step.marker)) {
		return passwordResetSQLStep{}, fmt.Errorf("SQL step %d = %s %q, want %s containing %q", tx.index, kind, compactPasswordResetSQL(sql), step.kind, step.marker)
	}
	if step.args != nil && !reflect.DeepEqual(args, step.args) {
		return passwordResetSQLStep{}, fmt.Errorf("SQL step %d args = %#v, want %#v", tx.index, args, step.args)
	}
	return step, nil
}

func (tx *passwordResetScriptTx) assertComplete(t *testing.T) {
	t.Helper()
	if tx.index != len(tx.steps) {
		t.Fatalf("executed SQL steps = %d, want %d", tx.index, len(tx.steps))
	}
}

type passwordResetScriptRow struct {
	values []any
	err    error
}

func (row passwordResetScriptRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(dest) != len(row.values) {
		return fmt.Errorf("scan destinations = %d, values = %d", len(dest), len(row.values))
	}
	for index := range dest {
		target := reflect.ValueOf(dest[index])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return fmt.Errorf("scan destination %d is not a pointer", index)
		}
		value := reflect.ValueOf(row.values[index])
		if !value.IsValid() {
			target.Elem().Set(reflect.Zero(target.Elem().Type()))
			continue
		}
		if !value.Type().AssignableTo(target.Elem().Type()) {
			return fmt.Errorf("scan value %d type %s cannot assign to %s", index, value.Type(), target.Elem().Type())
		}
		target.Elem().Set(value)
	}
	return nil
}

func compactPasswordResetSQL(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func bytesOf(value byte) []byte {
	result := make([]byte, 32)
	for index := range result {
		result[index] = value
	}
	return result
}

func passwordResetAuthorityNow() time.Time {
	return time.Date(2026, 10, 25, 4, 0, 0, 0, time.UTC)
}

func passwordResetAcceptedSettlementTx(now time.Time, lease PasswordResetRecoveryLease) *passwordResetScriptTx {
	credentialID := "00000000-0000-0000-0000-000000000103"
	currentOperation := lease.OperationID
	sourceOperation := lease.OperationID
	owner := lease.Owner
	expires := now.Add(time.Minute)
	return &passwordResetScriptTx{steps: []passwordResetSQLStep{
		queryStep("FROM users WHERE id = $1 FOR UPDATE", []any{int64(0), CredentialModePermanent, &currentOperation}),
		queryStep("FROM password_reset_operations o JOIN password_reset_recovery_jobs", []any{
			PasswordResetOperationRecoveryRequired, PasswordResetDeliveryAccepted, PasswordResetSettlementPending,
			credentialID, PasswordResetInitialReset, (*string)(nil), int64(0), int64(1),
			bytesOf(0x11), bytesOf(0x22), bytesOf(0x33), bytesOf(0x44), "v1", int64(1), int64(1), &owner, &expires,
		}),
		execStep("SET state='converging'", []any{now, lease.OperationID}, 1),
		queryStep("FROM temporary_credentials WHERE id=$1 FOR UPDATE", []any{lease.UserID, TemporaryCredentialPendingDelivery, &sourceOperation, int64(1)}),
		execStep("SET state='active'", []any{credentialID, lease.UserID}, 1),
		execStep("UPDATE users SET auth_generation", []any{int64(1), now, lease.UserID, int64(0), lease.OperationID}, 1),
		execStep("UPDATE sessions SET revoked_at", []any{now, lease.UserID}, 1),
		execStep("UPDATE password_reset_operations SET state='succeeded'", []any{int64(1), now, lease.OperationID}, 1),
		execStep("UPDATE idempotency_records SET status", nil, 1),
		execStep("INSERT INTO audit_events", nil, 1),
		execStep("UPDATE password_reset_recovery_jobs SET status='succeeded'", []any{now, lease.OperationID, lease.Owner, lease.AttemptCount}, 1),
	}}
}

func passwordResetAcceptedSettlementBoundaryTx(
	startedAt time.Time,
	completionAt time.Time,
	expiresAt time.Time,
	lease PasswordResetRecoveryLease,
	completionRows int64,
) *passwordResetScriptTx {
	credentialID := "00000000-0000-0000-0000-000000000103"
	currentOperation := lease.OperationID
	sourceOperation := lease.OperationID
	owner := lease.Owner
	return &passwordResetScriptTx{steps: []passwordResetSQLStep{
		queryStep("FROM users WHERE id = $1 FOR UPDATE", []any{int64(0), CredentialModePermanent, &currentOperation}),
		queryStep("FROM password_reset_operations o JOIN password_reset_recovery_jobs", []any{
			PasswordResetOperationRecoveryRequired, PasswordResetDeliveryAccepted, PasswordResetSettlementPending,
			credentialID, PasswordResetInitialReset, (*string)(nil), int64(0), int64(1),
			bytesOf(0x11), bytesOf(0x22), bytesOf(0x33), bytesOf(0x44), "v1", int64(1), lease.AttemptCount, &owner, &expiresAt,
		}),
		execStep("SET state='converging'", []any{startedAt, lease.OperationID}, 1),
		queryStep("FROM temporary_credentials WHERE id=$1 FOR UPDATE", []any{lease.UserID, TemporaryCredentialPendingDelivery, &sourceOperation, int64(1)}),
		execStep("SET state='active'", []any{credentialID, lease.UserID}, 1),
		execStep("UPDATE users SET auth_generation", []any{int64(1), startedAt, lease.UserID, int64(0), lease.OperationID}, 1),
		execStep("UPDATE sessions SET revoked_at", []any{startedAt, lease.UserID}, 1),
		execStep("UPDATE password_reset_operations SET state='succeeded'", []any{int64(1), startedAt, lease.OperationID}, 1),
		execStep("UPDATE idempotency_records SET status", nil, 1),
		execStep("INSERT INTO audit_events", nil, 1),
		execStep("UPDATE password_reset_recovery_jobs SET status='succeeded'", []any{completionAt, lease.OperationID, lease.Owner, lease.AttemptCount}, completionRows),
	}}
}

func passwordResetAuthorityTx(lease PasswordResetRecoveryLease, row []any, rowErr error) *passwordResetScriptTx {
	return &passwordResetScriptTx{steps: []passwordResetSQLStep{
		queryStep("SELECT id FROM users WHERE id=$1 FOR UPDATE", []any{lease.UserID}),
		{kind: "query", marker: "FROM users u JOIN password_reset_operations", row: row, err: rowErr},
	}}
}

func passwordResetCommittedAuthorityRow() []any {
	now := passwordResetAuthorityNow()
	lease := passwordResetTestLease()
	operation := lease.OperationID
	credential := "00000000-0000-0000-0000-000000000103"
	source := lease.OperationID
	completed := now
	result := "password_reset_succeeded"
	snapshot := fmt.Sprintf(`{"outcome":"succeeded","resource_id":%q}`, lease.OperationID)
	return []any{
		int64(1), CredentialModeTemporary, &operation, false,
		PasswordResetOperationSucceeded, PasswordResetDeliveryAccepted, PasswordResetSettlementTemporaryActive,
		credential, int64(1),
		PasswordResetRecoveryJobSucceeded, (*string)(nil), (*time.Time)(nil), int64(1), &completed,
		lease.UserID, TemporaryCredentialActive, &source, int64(1),
		int64(0), "succeeded", &result, &snapshot, int64(1),
	}
}

func passwordResetRolledBackAuthorityRow() []any {
	now := passwordResetAuthorityNow()
	lease := passwordResetTestLease()
	operation := lease.OperationID
	credential := "00000000-0000-0000-0000-000000000103"
	source := lease.OperationID
	owner := lease.Owner
	expires := now.Add(time.Minute)
	return []any{
		int64(0), CredentialModePermanent, &operation, false,
		PasswordResetOperationRecoveryRequired, PasswordResetDeliveryAccepted, PasswordResetSettlementPending,
		credential, int64(1),
		PasswordResetRecoveryJobLeased, &owner, &expires, int64(1), (*time.Time)(nil),
		lease.UserID, TemporaryCredentialPendingDelivery, &source, int64(1),
		int64(1), "processing", (*string)(nil), (*string)(nil), int64(0),
	}
}

// TestPasswordResetMaintenanceOperator is the fixed domain acceptance entry.
func TestPasswordResetMaintenanceOperator(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"I-S105-03_04_05_trust_authority", testMaintenanceTrustAuthority},
		{"I-S105-07_09_strict_canonical", testMaintenanceCanonical},
		{"I-S105-08_all_field_signature_binding", testMaintenanceSignatureBinding},
		{"I-S105-10_time_windows", testMaintenanceTimeWindows},
		{"I-S105-11_opaque_deadline", testMaintenanceOpaqueDeadline},
		{"I-S105-12_minimal_write_and_precision", testMaintenanceApply},
		{"I-S105-13_state_matrix", testMaintenanceStates},
		{"I-S105-14_all_25_before_fields", testMaintenanceStaleBefore},
		{"I-S105-15_exact_replay", testMaintenanceReplay},
		{"I-S105-16_scalar_guard", testMaintenanceGuard},
		{"I-S105-17_20_atomic_failures", testMaintenanceTransactionFailures},
		{"I-S105-21_22_unknown_cleanup", testMaintenanceCleanup},
		{"I-S105-07_12_19_quick100", testMaintenanceProperties},
		{"I-S105-11_15_21_r49_final_cancel", testMaintenanceR49FinalCancellation},
		{"I-S105-22_r49_sticky_overrun", testMaintenanceR49CleanupClassification},
		{"I-S105-21_22_r49_actual_rollback_join", testMaintenanceR49RollbackJoin},
		{"I-S105-22_r49_priority_quick100", testMaintenanceR49PriorityProperties},
	} {
		t.Run(test.name, test.run)
	}
}

type maintenanceFixture struct {
	operator *PasswordResetOperator
	config   PasswordResetMaintenanceConfig
	trust    passwordResetMaintenanceTrust
	payload  passwordResetRequeuePayload
	private  ed25519.PrivateKey
	clock    *passwordResetMutableClock
	ctx      context.Context
}

func newMaintenanceFixture(t *testing.T) *maintenanceFixture {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 10, 0, 0, 123456000, time.UTC)
	executor, approver := "windows-sid:S-1-5-21-1", "windows-sid:S-1-5-21-2"
	trust := passwordResetMaintenanceTrust{Version: passwordResetMaintenanceTrustVersion, DeploymentID: "test-deployment", Epoch: "test-epoch", ApproverKeys: []passwordResetMaintenanceKey{{KeyID: "approver-1", PublicKey: base64.RawURLEncoding.EncodeToString(public), Subject: approver, Action: passwordResetMaintenanceAction, NotBefore: now.Unix() - 1, ExpiresAt: now.Unix() + 3600}}, Executors: []passwordResetMaintenanceExecutor{{Subject: executor, Action: passwordResetMaintenanceAction, NotBefore: now.Unix() - 1, ExpiresAt: now.Unix() + 3600}}}
	config := PasswordResetMaintenanceConfig{TrustJSON: maintenanceMarshal(t, trust), MaxTTL: 15 * time.Minute, OperationBudget: 30 * time.Second, ShutdownBudget: time.Second}
	clock := &passwordResetMutableClock{now: now}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	operator, err := newPasswordResetMaintenanceOperator(config, clock, "windows", func() (string, error) { return executor, nil })
	if err != nil {
		t.Fatal(err)
	}
	operator.maintenance.clientEncoding = func(pgx.Tx) bool { return true } // U seam, not actual OS/connection proof.
	operationID := "00000000-0000-0000-0000-000000000102"
	timestamp := now.Add(-time.Minute).Format(passwordResetMaintenanceTimeFormat)
	before := PasswordResetRequeueSnapshot{UserStatus: "active", UserAuthGeneration: 9007199254740993, UserExecutionGeneration: 1, UserCurrentOperationID: &operationID, CredentialMode: "permanent", UserUpdatedAt: timestamp, OperationState: "recovery_required", DeliveryState: "accepted", SettlementState: "pending", OperationExecutionGeneration: 1, TemporaryCredentialID: "00000000-0000-0000-0000-000000000103", OperationUpdatedAt: timestamp, JobStatus: "dead_letter", AttemptCount: 2, NextAttemptAt: timestamp, JobCreatedAt: timestamp, JobUpdatedAt: timestamp}
	payload := passwordResetRequeuePayload{Version: passwordResetMaintenancePayloadVersion, DeploymentID: trust.DeploymentID, Epoch: trust.Epoch, RequestID: "00000000-0000-0000-0000-000000000104", Action: passwordResetMaintenanceAction, UserID: "00000000-0000-0000-0000-000000000101", OperationID: operationID, ApproverKeyID: "approver-1", ExecutorSubject: executor, IssuedAt: now.Unix(), ExpiresAt: now.Unix() + 600, Reason: "test approval", EvidenceRef: "test:retained-evidence", Before: before}
	return &maintenanceFixture{operator: operator, config: config, trust: trust, payload: payload, private: private, clock: clock, ctx: ctx}
}

func maintenanceMarshal(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (fixture *maintenanceFixture) envelope(t *testing.T, payload []byte, domain string) []byte {
	t.Helper()
	signature := ed25519.Sign(fixture.private, append([]byte(domain), payload...))
	return maintenanceMarshal(t, passwordResetRequeueEnvelope{Payload: base64.RawURLEncoding.EncodeToString(payload), ApprovalSignature: base64.RawURLEncoding.EncodeToString(signature)})
}

func (fixture *maintenanceFixture) request(t *testing.T) *PasswordResetRequeueRequest {
	t.Helper()
	request, err := fixture.operator.ValidateRequeue(fixture.ctx, fixture.envelope(t, maintenanceMarshal(t, fixture.payload), passwordResetMaintenanceDomain))
	if err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	return request
}

func requireMaintenanceCode(t *testing.T, err error, code PasswordResetMaintenanceCode) {
	t.Helper()
	var failure *PasswordResetMaintenanceError
	if !errors.As(err, &failure) || failure.Code != code || failure.Error() != string(code) {
		t.Fatalf("error = %v, want %s", err, code)
	}
}

func testMaintenanceTrustAuthority(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*maintenanceFixture)
		code   PasswordResetMaintenanceCode
	}{
		{"unknown_key", func(f *maintenanceFixture) {
			f.config.TrustJSON = append([]byte(`{"extra":1,`), f.config.TrustJSON[1:]...)
		}, PasswordResetMaintenanceInvalidConfig},
		{"duplicate", func(f *maintenanceFixture) {
			f.config.TrustJSON = bytes.Replace(f.config.TrustJSON, []byte(`"epoch":"test-epoch"`), []byte(`"epoch":"test-epoch","epoch":"test-epoch"`), 1)
		}, PasswordResetMaintenanceInvalidConfig},
		{"self_approval", func(f *maintenanceFixture) {
			f.trust.ApproverKeys[0].Subject = f.trust.Executors[0].Subject
			f.config.TrustJSON = maintenanceMarshal(t, f.trust)
		}, PasswordResetMaintenanceInvalidConfig},
		{"duplicate_public", func(f *maintenanceFixture) {
			key := f.trust.ApproverKeys[0]
			key.KeyID = "other"
			key.Subject = "windows-sid:S-1-5-21-3"
			f.trust.ApproverKeys = append(f.trust.ApproverKeys, key)
			f.config.TrustJSON = maintenanceMarshal(t, f.trust)
		}, PasswordResetMaintenanceInvalidConfig},
		{"wrong_usage", func(f *maintenanceFixture) {
			f.trust.ApproverKeys[0].Action = "verify"
			f.config.TrustJSON = maintenanceMarshal(t, f.trust)
		}, PasswordResetMaintenanceInvalidConfig},
		{"expired_executor", func(f *maintenanceFixture) {
			f.trust.Executors[0].ExpiresAt = f.clock.Now().Unix()
			f.config.TrustJSON = maintenanceMarshal(t, f.trust)
		}, PasswordResetMaintenanceForbidden},
		{"no_matching_executor", func(f *maintenanceFixture) {
			f.trust.Executors[0].Subject = "windows-sid:S-1-5-21-3"
			f.config.TrustJSON = maintenanceMarshal(t, f.trust)
		}, PasswordResetMaintenanceForbidden},
		{"shutdown_zero", func(f *maintenanceFixture) { f.config.ShutdownBudget = 0 }, PasswordResetMaintenanceInvalidConfig},
		{"shutdown_over", func(f *maintenanceFixture) { f.config.ShutdownBudget = time.Minute + time.Nanosecond }, PasswordResetMaintenanceInvalidConfig},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			test.change(f)
			_, err := newPasswordResetMaintenanceOperator(f.config, f.clock, "windows", func() (string, error) { return f.payload.ExecutorSubject, nil })
			requireMaintenanceCode(t, err, test.code)
		})
	}
	testMaintenancePlatformAndSID(t)
}

func testMaintenancePlatformAndSID(t *testing.T) {
	f := newMaintenanceFixture(t)
	_, err := newPasswordResetMaintenanceOperator(f.config, f.clock, "linux", func() (string, error) { t.Fatal("non-Windows resolver called"); return "", nil })
	requireMaintenanceCode(t, err, PasswordResetMaintenanceUnsupportedPlatform)
	for _, sid := range []string{"", "USER", "windows-sid:S-1-05-21-1", "windows-sid:S-1-281474976710656-1", "windows-sid:S-1-5-4294967296", "windows-sid:S-1-5"} {
		_, err := newPasswordResetMaintenanceOperator(f.config, f.clock, "windows", func() (string, error) { return sid, nil })
		requireMaintenanceCode(t, err, PasswordResetMaintenanceAuthorityUnavailable)
	}
	for _, budget := range []time.Duration{time.Second, time.Minute} {
		config := f.config
		config.ShutdownBudget = budget
		if _, err := newPasswordResetMaintenanceOperator(config, f.clock, "windows", func() (string, error) { return f.payload.ExecutorSubject, nil }); err != nil {
			t.Fatal(err)
		}
	}
}

func testMaintenanceCanonical(t *testing.T) {
	f := newMaintenanceFixture(t)
	canonical := maintenanceMarshal(t, f.payload)
	for _, test := range []struct {
		name string
		raw  []byte
	}{
		{"whitespace", append([]byte(" "), canonical...)},
		{"duplicate", bytes.Replace(canonical, []byte(`"version":`), []byte(`"version":"han-password-reset-requeue/v1","version":`), 1)},
		{"case_alias", bytes.Replace(canonical, []byte(`"version"`), []byte(`"Version"`), 1)},
		{"missing_nullable", bytes.Replace(canonical, []byte(`"integrity_hold_at":null,`), nil, 1)},
		{"null_required", bytes.Replace(canonical, []byte(`"integrity_hold":false`), []byte(`"integrity_hold":null`), 1)},
		{"numeric_exponent", bytes.Replace(canonical, []byte(`"attempt_count":2`), []byte(`"attempt_count":2e0`), 1)},
		{"escaped_string", bytes.Replace(canonical, []byte(`"active"`), []byte(`"\u0061ctive"`), 1)},
		{"trailing", append(bytes.Clone(canonical), '\n')},
		{"unknown", bytes.Replace(canonical, []byte(`"before":{`), []byte(`"before":{"extra":1,`), 1)},
		{"invalid_utf8", append(bytes.Clone(canonical), 0xff)},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := f.operator.ValidateRequeue(f.ctx, f.envelope(t, test.raw, passwordResetMaintenanceDomain))
			requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
		})
	}
	for _, size := range []int{0, passwordResetMaintenanceEnvelopeLimit + 1} {
		_, err := f.operator.ValidateRequeue(f.ctx, bytes.Repeat([]byte{' '}, size))
		requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	}
	for _, text := range []string{"", " ", "a\n", strings.Repeat("界", 171)} {
		p := f.payload
		p.Reason = text
		_, err := f.operator.ValidateRequeue(f.ctx, f.envelope(t, maintenanceMarshal(t, p), passwordResetMaintenanceDomain))
		requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	}
	p := f.payload
	p.Reason = strings.Repeat("a", 512)
	p.EvidenceRef = strings.Repeat("b", 256)
	if _, err := f.operator.ValidateRequeue(f.ctx, f.envelope(t, maintenanceMarshal(t, p), passwordResetMaintenanceDomain)); err != nil {
		t.Fatal(err)
	}
}

func testMaintenanceSignatureBinding(t *testing.T) {
	f := newMaintenanceFixture(t)
	wire := f.envelope(t, maintenanceMarshal(t, f.payload), passwordResetMaintenanceDomain)
	var envelope passwordResetRequeueEnvelope
	if err := json.Unmarshal(wire, &envelope); err != nil {
		t.Fatal(err)
	}
	original := envelope.ApprovalSignature
	for _, domain := range []string{"", strings.TrimSuffix(passwordResetMaintenanceDomain, "\x00") + `\x00`, "other-domain\x00"} {
		_, err := f.operator.ValidateRequeue(f.ctx, f.envelope(t, maintenanceMarshal(t, f.payload), domain))
		requireMaintenanceCode(t, err, PasswordResetMaintenanceForbidden)
	}
	for _, length := range []int{63, 65} {
		envelope.ApprovalSignature = base64.RawURLEncoding.EncodeToString(make([]byte, length))
		_, err := f.operator.ValidateRequeue(f.ctx, maintenanceMarshal(t, envelope))
		requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	}
	envelope.ApprovalSignature = original
	fields := reflect.ValueOf(&f.payload).Elem()
	for i := 0; i < fields.NumField(); i++ {
		if fields.Field(i).Kind() != reflect.String {
			continue
		}
		t.Run(fields.Type().Field(i).Name, func(t *testing.T) {
			p := f.payload
			value := reflect.ValueOf(&p).Elem().Field(i)
			value.SetString(value.String() + "x")
			envelope.Payload = base64.RawURLEncoding.EncodeToString(maintenanceMarshal(t, p))
			if _, err := f.operator.ValidateRequeue(f.ctx, maintenanceMarshal(t, envelope)); err == nil {
				t.Fatal("mutated signed field accepted")
			}
		})
	}
	p := f.payload
	p.Before.UserAuthGeneration++
	envelope.Payload = base64.RawURLEncoding.EncodeToString(maintenanceMarshal(t, p))
	_, err := f.operator.ValidateRequeue(f.ctx, maintenanceMarshal(t, envelope))
	requireMaintenanceCode(t, err, PasswordResetMaintenanceForbidden)
	for _, size := range []int{31, 33} {
		trust := f.trust
		trust.ApproverKeys = append([]passwordResetMaintenanceKey(nil), f.trust.ApproverKeys...)
		trust.ApproverKeys[0].PublicKey = base64.RawURLEncoding.EncodeToString(make([]byte, size))
		if validPasswordResetMaintenanceTrust(trust) {
			t.Fatal("invalid public key length")
		}
	}
}

func testMaintenanceTimeWindows(t *testing.T) {
	for _, test := range []struct {
		name            string
		issued, expires int64
		want            bool
	}{
		{"issued_now", 0, 1, true}, {"future", 1, 2, false}, {"expired", -1, 0, false}, {"zero", 0, 0, false}, {"reverse", 1, 0, false}, {"ttl_exact", 0, 900, true}, {"ttl_over", 0, 901, false}, {"int64_max", 0, math.MaxInt64, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			now := f.clock.Now().Unix()
			f.payload.IssuedAt = now + test.issued
			f.payload.ExpiresAt = now + test.expires
			if test.name == "int64_max" {
				f.payload.ExpiresAt = math.MaxInt64
			}
			_, err := f.operator.ValidateRequeue(f.ctx, f.envelope(t, maintenanceMarshal(t, f.payload), passwordResetMaintenanceDomain))
			if (err == nil) != test.want {
				t.Fatalf("accepted=%t want=%t", err == nil, test.want)
			}
		})
	}
	f := newMaintenanceFixture(t)
	request := f.request(t)
	f.clock.Set(time.Unix(f.payload.ExpiresAt, 0))
	_, err := f.operator.Requeue(f.ctx, request, PasswordResetRequeueDependencies{})
	requireMaintenanceCode(t, err, PasswordResetMaintenanceForbidden)
}

func testMaintenanceOpaqueDeadline(t *testing.T) {
	f := newMaintenanceFixture(t)
	request := f.request(t)
	for _, ctx := range []context.Context{context.Background(), nil} {
		_, err := f.operator.ValidateRequeue(ctx, f.envelope(t, maintenanceMarshal(t, f.payload), passwordResetMaintenanceDomain))
		requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	}
	other := newMaintenanceFixture(t)
	_, err := other.operator.Requeue(f.ctx, request, PasswordResetRequeueDependencies{})
	requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	_, err = f.operator.Requeue(f.ctx, &PasswordResetRequeueRequest{}, PasswordResetRequeueDependencies{})
	requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	later, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	_, err = f.operator.Requeue(later, request, PasswordResetRequeueDependencies{})
	requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
	canceled, cancel := context.WithCancel(f.ctx)
	cancel()
	_, err = f.operator.Requeue(canceled, request, PasswordResetRequeueDependencies{})
	requireMaintenanceCode(t, err, PasswordResetMaintenanceFailed)
	copyTrust := bytes.Clone(f.config.TrustJSON)
	for i := range f.config.TrustJSON {
		f.config.TrustJSON[i] = 'x'
	}
	if !bytes.Equal(f.operator.maintenance.config.TrustJSON, copyTrust) {
		t.Fatal("trust not privately copied")
	}
	request.payload[0] = 'x'
	_, err = f.operator.Requeue(f.ctx, request, PasswordResetRequeueDependencies{})
	requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidRequest)
}

type maintenanceTestStarter struct {
	tx      pgx.Tx
	begins  int
	options pgx.TxOptions
}

func (starter *maintenanceTestStarter) BeginTx(_ context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	starter.begins++
	starter.options = options
	return starter.tx, nil
}

type maintenanceTestTx struct {
	*passwordResetScriptTx
	rollbackErr      error
	rollbackDeadline time.Time
	rollbackCanceled error
}

func (tx *maintenanceTestTx) Rollback(ctx context.Context) error {
	tx.rollbacks++
	tx.rollbackDeadline, _ = ctx.Deadline()
	tx.rollbackCanceled = ctx.Err()
	return tx.rollbackErr
}

func maintenanceSnapshotSteps(s PasswordResetRequeueSnapshot) []passwordResetSQLStep {
	return []passwordResetSQLStep{
		queryStep("FROM users", []any{s.UserStatus, s.UserAuthGeneration, s.UserExecutionGeneration, s.UserCurrentOperationID, s.CredentialMode, s.IntegrityHold, s.IntegrityHoldOperationID, s.IntegrityHoldReason, s.IntegrityHoldAt, s.UserUpdatedAt}),
		queryStep("FOR UPDATE OF o,j", []any{s.OperationState, s.DeliveryState, s.SettlementState, s.OperationExecutionGeneration, s.TemporaryCredentialID, s.OperationUpdatedAt, s.JobStatus, s.AttemptCount, s.NextAttemptAt, s.LeaseOwner, s.LeaseExpiresAt, s.LastFailureCode, s.JobCreatedAt, s.JobUpdatedAt, s.CompletedAt}),
	}
}

func maintenanceGuardStep(size int64, binding bool) passwordResetSQLStep {
	return queryStep("octet_length", []any{size, binding})
}
func maintenanceAbsentStep() passwordResetSQLStep {
	return passwordResetSQLStep{kind: "query", marker: "octet_length", err: pgx.ErrNoRows}
}

func maintenanceApplyTx(f *maintenanceFixture) *maintenanceTestTx {
	before := f.payload.Before
	after := passwordResetRequeueExpectedAfter(before, f.clock.Now())
	steps := maintenanceSnapshotSteps(before)
	steps = append(steps, maintenanceAbsentStep(), execStep("UPDATE password_reset_recovery_jobs", nil, 1))
	steps = append(steps, maintenanceSnapshotSteps(after)...)
	steps = append(steps, execStep("INSERT INTO audit_events", nil, 1), maintenanceGuardStep(4096, true))
	return &maintenanceTestTx{passwordResetScriptTx: &passwordResetScriptTx{steps: steps}}
}

func maintenanceDependencies(t *testing.T, tx pgx.Tx) (PasswordResetRequeueDependencies, *maintenanceTestStarter) {
	t.Helper()
	permit, err := NewPasswordResetConnectionPermit(1)
	if err != nil {
		t.Fatal(err)
	}
	starter := &maintenanceTestStarter{tx: tx}
	return PasswordResetRequeueDependencies{Transactions: starter, Store: &PasswordResetStore{}, Audit: audit.Store{}, Permit: permit}, starter
}

func maintenanceAssertFinished(t *testing.T, tx *maintenanceTestTx, dependencies PasswordResetRequeueDependencies, starter *maintenanceTestStarter, commit bool) {
	t.Helper()
	tx.assertComplete(t)
	stats := dependencies.Permit.Stats()
	if stats.Current != 0 || stats.Acquired != 1 || stats.Released != 1 || starter.begins != 1 || starter.options.IsoLevel != pgx.ReadCommitted {
		t.Fatalf("permit/tx: %+v begins=%d options=%+v", stats, starter.begins, starter.options)
	}
	if commit && (tx.commits != 1 || tx.rollbacks != 0) {
		t.Fatalf("commit/rollback=%d/%d", tx.commits, tx.rollbacks)
	}
	if !commit && tx.rollbacks != 1 {
		t.Fatalf("rollback=%d", tx.rollbacks)
	}
}

func testMaintenanceApply(t *testing.T) {
	for _, offset := range []time.Duration{-time.Minute, 0, time.Minute} {
		t.Run(offset.String(), func(t *testing.T) {
			f := newMaintenanceFixture(t)
			f.payload.Before.NextAttemptAt = f.clock.Now().Add(offset).Format(passwordResetMaintenanceTimeFormat)
			request := f.request(t)
			tx := maintenanceApplyTx(f)
			dependencies, starter := maintenanceDependencies(t, tx)
			metadataSeen := false
			tx.onExec = func(sql string, args []any) {
				if !strings.Contains(sql, "INSERT INTO audit_events") {
					return
				}
				metadataSeen = true
				var metadata map[string]any
				if err := json.Unmarshal(args[13].([]byte), &metadata); err != nil {
					t.Fatal(err)
				}
				if len(metadata) != 13 {
					t.Fatalf("metadata fields %d", len(metadata))
				}
				for _, value := range metadata {
					if _, ok := value.(string); !ok {
						t.Fatal("non-string metadata")
					}
				}
				raw, err := decodePasswordResetMaintenanceBase64(metadata["payload_b64"].(string), passwordResetMaintenancePayloadLimit)
				if err != nil || !bytes.Equal(raw, request.payload) {
					t.Fatal("exact P lost")
				}
				if args[0] != f.payload.RequestID || args[11] != f.payload.RequestID {
					t.Fatal("audit/trace UUID mismatch")
				}
			}
			got, err := f.operator.Requeue(f.ctx, request, dependencies)
			if err != nil {
				t.Fatal(err)
			}
			if !metadataSeen || got.Disposition != PasswordResetRequeueApplied || got.Receipt != passwordResetRequeueReceipt(f.payload) || !reflect.DeepEqual(got.After, passwordResetRequeueExpectedAfter(f.payload.Before, f.clock.Now())) || got.After.UserAuthGeneration != 9007199254740993 || got.After.AttemptCount != 2 {
				t.Fatalf("inexact result %+v", got)
			}
			maintenanceAssertFinished(t, tx, dependencies, starter, true)
		})
	}
}

func testMaintenanceStates(t *testing.T) {
	for _, state := range []struct {
		operation, delivery string
		want                bool
	}{
		{"prepared", "not_started", true}, {"dispatching", "dispatching", true}, {"commit_unknown", "dispatching", true}, {"commit_unknown", "unknown", true}, {"recovery_required", "accepted", true}, {"converging", "accepted", true}, {"succeeded", "accepted", false}, {"invalidated", "unknown", false}, {"failed", "rejected", false}, {"prepared", "accepted", false},
	} {
		t.Run(state.operation+"_"+state.delivery, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			f.payload.Before.OperationState = state.operation
			f.payload.Before.DeliveryState = state.delivery
			if eligiblePasswordResetRequeue(f.payload.Before, f.payload) != state.want {
				t.Fatal("state eligibility")
			}
			if state.want {
				tx := maintenanceApplyTx(f)
				deps, starter := maintenanceDependencies(t, tx)
				if _, err := f.operator.Requeue(f.ctx, f.request(t), deps); err != nil {
					t.Fatal(err)
				}
				maintenanceAssertFinished(t, tx, deps, starter, true)
			}
		})
	}
	for _, test := range []struct {
		name   string
		change func(*PasswordResetRequeueSnapshot)
	}{
		{"ready", func(s *PasswordResetRequeueSnapshot) { s.JobStatus = "ready" }}, {"leased", func(s *PasswordResetRequeueSnapshot) { s.JobStatus = "leased" }}, {"retry_wait", func(s *PasswordResetRequeueSnapshot) { s.JobStatus = "retry_wait" }}, {"succeeded", func(s *PasswordResetRequeueSnapshot) { s.JobStatus = "succeeded" }},
		{"terminal", func(s *PasswordResetRequeueSnapshot) { s.OperationState = "succeeded" }}, {"hold", func(s *PasswordResetRequeueSnapshot) { s.IntegrityHold = true }}, {"nonpending", func(s *PasswordResetRequeueSnapshot) { s.SettlementState = "invalidated" }},
		{"max_attempt", func(s *PasswordResetRequeueSnapshot) { s.AttemptCount = math.MaxInt64 }}, {"leased_owner", func(s *PasswordResetRequeueSnapshot) { v := "foreign"; s.LeaseOwner = &v }}, {"completed", func(s *PasswordResetRequeueSnapshot) { s.CompletedAt = &s.JobUpdatedAt }}, {"not_current", func(s *PasswordResetRequeueSnapshot) { s.UserCurrentOperationID = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			test.change(&f.payload.Before)
			steps := append(maintenanceSnapshotSteps(f.payload.Before), maintenanceAbsentStep())
			tx := &maintenanceTestTx{passwordResetScriptTx: &passwordResetScriptTx{steps: steps}}
			deps, starter := maintenanceDependencies(t, tx)
			_, err := f.operator.Requeue(f.ctx, f.request(t), deps)
			requireMaintenanceCode(t, err, PasswordResetMaintenanceStateConflict)
			maintenanceAssertFinished(t, tx, deps, starter, false)
			if tx.executes != 0 {
				t.Fatal("rejected state wrote")
			}
		})
	}
}

func testMaintenanceStaleBefore(t *testing.T) {
	count := reflect.TypeOf(PasswordResetRequeueSnapshot{}).NumField()
	if count != 25 {
		t.Fatalf("snapshot count=%d", count)
	}
	for i := 0; i < count; i++ {
		t.Run(reflect.TypeOf(PasswordResetRequeueSnapshot{}).Field(i).Name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			actual := f.payload.Before
			field := reflect.ValueOf(&actual).Elem().Field(i)
			switch field.Kind() {
			case reflect.String:
				field.SetString(field.String() + "changed")
			case reflect.Int64:
				field.SetInt(field.Int() + 1)
			case reflect.Bool:
				field.SetBool(!field.Bool())
			case reflect.Pointer:
				if field.IsNil() {
					value := "changed"
					field.Set(reflect.ValueOf(&value))
				} else {
					field.Set(reflect.Zero(field.Type()))
				}
			}
			tx := &maintenanceTestTx{passwordResetScriptTx: &passwordResetScriptTx{steps: append(maintenanceSnapshotSteps(actual), maintenanceAbsentStep())}}
			deps, starter := maintenanceDependencies(t, tx)
			_, err := f.operator.Requeue(f.ctx, f.request(t), deps)
			requireMaintenanceCode(t, err, PasswordResetMaintenanceStateConflict)
			maintenanceAssertFinished(t, tx, deps, starter, false)
			if tx.executes != 0 {
				t.Fatal("stale before wrote")
			}
		})
	}
}

func maintenanceReplayTx(t *testing.T, f *maintenanceFixture, request *PasswordResetRequeueRequest, mutate func(map[string]any)) *maintenanceTestTx {
	t.Helper()
	after := passwordResetRequeueExpectedAfter(f.payload.Before, f.clock.Now())
	metadata, err := f.operator.maintenance.requeueMetadata(request, f.payload, after, passwordResetRequeueReceipt(f.payload))
	if err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(metadata)
	}
	actor, role := f.payload.ExecutorSubject, passwordResetMaintenanceRole
	row := []any{f.payload.RequestID, f.clock.Now(), audit.ActorSystem, (*string)(nil), &actor, &role, passwordResetMaintenanceAuditAction, "password_reset_operation", f.payload.OperationID, "succeeded", (*string)(nil), f.payload.RequestID, (*string)(nil), maintenanceMarshal(t, metadata)}
	current := f.payload.Before
	current.JobStatus = "succeeded"
	current.AttemptCount = 5 // Historical after must not be replaced with current job state.
	steps := append(maintenanceSnapshotSteps(current), maintenanceGuardStep(4096, true), queryStep("FROM audit_events", row))
	return &maintenanceTestTx{passwordResetScriptTx: &passwordResetScriptTx{steps: steps}}
}

func testMaintenanceReplay(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
		valid  bool
	}{
		{"exact", nil, true}, {"digest", func(m map[string]any) { m["payload_sha256"] = "wrong" }, false}, {"extra", func(m map[string]any) { m["extra"] = "wrong" }, false}, {"missing", func(m map[string]any) { delete(m, "epoch") }, false}, {"float_carrier", func(m map[string]any) { m["epoch"] = float64(1) }, false}, {"base64_padding", func(m map[string]any) { m["after_b64"] = m["after_b64"].(string) + "=" }, false}, {"oversize", func(m map[string]any) { m["result_b64"] = strings.Repeat("a", 2000) }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			request := f.request(t)
			tx := maintenanceReplayTx(t, f, request, test.mutate)
			deps, starter := maintenanceDependencies(t, tx)
			sawText := false
			tx.onQuery = func(sql string) {
				if strings.Contains(sql, "FROM audit_events") && !strings.Contains(sql, "octet_length") {
					step := tx.steps[tx.index-1]
					_ = step
					sawText = true
				}
			}
			tx.steps[len(tx.steps)-1].args = []any{pgx.QueryResultFormatsByOID{pgtype.JSONBOID: pgx.TextFormatCode}, f.payload.RequestID}
			got, err := f.operator.Requeue(f.ctx, request, deps)
			if test.valid {
				if err != nil || got.Disposition != PasswordResetRequeueAlreadyApplied || got.Receipt.Outcome != "applied" || got.After.AttemptCount != 2 {
					t.Fatalf("replay %+v err %v", got, err)
				}
			} else {
				requireMaintenanceCode(t, err, PasswordResetMaintenanceStateConflict)
			}
			if !sawText || tx.executes != 0 {
				t.Fatal("replay did not use text read / wrote")
			}
			maintenanceAssertFinished(t, tx, deps, starter, test.valid)
		})
	}
}

func testMaintenanceGuard(t *testing.T) {
	for _, test := range []struct {
		name     string
		size     int64
		binding  bool
		queryErr error
		code     PasswordResetMaintenanceCode
	}{
		{"oversize", 65537, true, nil, PasswordResetMaintenanceStateConflict}, {"negative", -1, true, nil, PasswordResetMaintenanceStateConflict}, {"binding_false", 0, false, nil, PasswordResetMaintenanceStateConflict}, {"query_error", 0, false, errors.New("guard transport error"), PasswordResetMaintenanceFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			step := maintenanceGuardStep(test.size, test.binding)
			step.err = test.queryErr
			tx := &maintenanceTestTx{passwordResetScriptTx: &passwordResetScriptTx{steps: append(maintenanceSnapshotSteps(f.payload.Before), step)}}
			deps, starter := maintenanceDependencies(t, tx)
			_, err := f.operator.Requeue(f.ctx, f.request(t), deps)
			requireMaintenanceCode(t, err, test.code)
			maintenanceAssertFinished(t, tx, deps, starter, false)
			if tx.executes != 0 {
				t.Fatal("guard rejection wrote")
			}
		})
	}
	for _, column := range []string{"actor_kind", "actor_system", "actor_role", "action", "target_type", "target_id", "outcome", "trace_id", "error_code", "source_fingerprint", "actor_user_id"} {
		if !strings.Contains(passwordResetRequeueAuditGuardSQL, column) {
			t.Fatalf("guard missing %s", column)
		}
	}
	if strings.Contains(passwordResetRequeueAuditGuardSQL, "WHERE action") || !strings.Contains(passwordResetRequeueAuditGuardSQL, "convert_to(metadata::text,'UTF8')") {
		t.Fatal("guard did not inspect global UUID text wire length")
	}
}

func testMaintenanceTransactionFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*maintenanceTestTx)
		code   PasswordResetMaintenanceCode
	}{
		{"update_zero", func(tx *maintenanceTestTx) {
			tx.steps[3].tag = pgconn.NewCommandTag("UPDATE 0")
			tx.steps = tx.steps[:4]
		}, PasswordResetMaintenanceStateConflict},
		{"append_error", func(tx *maintenanceTestTx) { tx.steps[6].err = errors.New("append error"); tx.steps = tx.steps[:7] }, PasswordResetMaintenanceFailed},
		{"uuid_collision", func(tx *maintenanceTestTx) { tx.steps[6].err = &pgconn.PgError{Code: "23505"}; tx.steps = tx.steps[:7] }, PasswordResetMaintenanceStateConflict},
		{"postappend_oversize", func(tx *maintenanceTestTx) { tx.steps[7] = maintenanceGuardStep(65537, true) }, PasswordResetMaintenanceStateConflict},
		{"postappend_binding", func(tx *maintenanceTestTx) { tx.steps[7] = maintenanceGuardStep(2000, false) }, PasswordResetMaintenanceStateConflict},
		{"postappend_absent", func(tx *maintenanceTestTx) { tx.steps[7] = maintenanceAbsentStep() }, PasswordResetMaintenanceFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			tx := maintenanceApplyTx(f)
			test.change(tx)
			deps, starter := maintenanceDependencies(t, tx)
			_, err := f.operator.Requeue(f.ctx, f.request(t), deps)
			requireMaintenanceCode(t, err, test.code)
			maintenanceAssertFinished(t, tx, deps, starter, false)
			if tx.commits != 0 {
				t.Fatal("invalid transaction committed")
			}
		})
	}
	f := newMaintenanceFixture(t)
	tx := maintenanceApplyTx(f)
	f.operator.maintenance.clientEncoding = func(pgx.Tx) bool { return false }
	tx.steps = nil
	deps, starter := maintenanceDependencies(t, tx)
	_, err := f.operator.Requeue(f.ctx, f.request(t), deps)
	requireMaintenanceCode(t, err, PasswordResetMaintenanceInvalidConfig)
	maintenanceAssertFinished(t, tx, deps, starter, false)
}

func testMaintenanceCleanup(t *testing.T) {
	for _, commitErr := range []error{pgx.ErrTxCommitRollback, pgx.ErrTxClosed, context.DeadlineExceeded, errors.New("ack lost")} {
		t.Run(commitErr.Error(), func(t *testing.T) {
			f := newMaintenanceFixture(t)
			request := f.request(t)
			tx := maintenanceApplyTx(f)
			tx.commitErr = commitErr
			deps, starter := maintenanceDependencies(t, tx)
			got, err := f.operator.Requeue(f.ctx, request, deps)
			requireMaintenanceCode(t, err, PasswordResetMaintenanceOutcomeUnknown)
			maintenanceAssertFinished(t, tx, deps, starter, false)
			if got.Receipt.RequestID != "" || tx.commits != 1 || tx.rollbackDeadline.After(request.deadline.Add(f.config.ShutdownBudget)) || tx.rollbackCanceled != nil {
				t.Fatal("unknown result/cleanup deadline")
			}
		})
	}
	testMaintenanceCleanupFailures(t)
}

func testMaintenanceCleanupFailures(t *testing.T) {
	for _, kind := range []string{"rollback_error", "panic", "lock_expiry", "commit_expiry"} {
		t.Run(kind, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			request := f.request(t)
			tx := maintenanceApplyTx(f)
			want := PasswordResetMaintenanceFailed
			switch kind {
			case "rollback_error":
				tx.steps = tx.steps[:4]
				tx.steps[3].err = errors.New("update failed")
				tx.rollbackErr = errors.New("rollback failed")
				want = PasswordResetMaintenanceOutcomeUnknown
			case "panic":
				tx.onExec = func(string, []any) { panic("private synthetic cause") }
				tx.steps = tx.steps[:4]
			case "lock_expiry":
				tx.onQuery = func(sql string) {
					if strings.Contains(sql, "FOR UPDATE OF o,j") {
						f.clock.Set(time.Unix(f.payload.ExpiresAt, 0))
					}
				}
				tx.steps = tx.steps[:2]
				want = PasswordResetMaintenanceForbidden
			case "commit_expiry":
				guards := 0
				tx.onQuery = func(sql string) {
					if strings.Contains(sql, "octet_length") {
						guards++
						if guards == 2 {
							f.clock.Set(time.Unix(f.payload.ExpiresAt, 0))
						}
					}
				}
				want = PasswordResetMaintenanceForbidden
			}
			deps, starter := maintenanceDependencies(t, tx)
			_, err := f.operator.Requeue(f.ctx, request, deps)
			requireMaintenanceCode(t, err, want)
			maintenanceAssertFinished(t, tx, deps, starter, false)
			if tx.commits != 0 {
				t.Fatal("failed operation committed")
			}
		})
	}
}

func testMaintenanceProperties(t *testing.T) {
	f := newMaintenanceFixture(t)
	property := func(generation uint64, attempt uint64, offset int32) bool {
		before := f.payload.Before
		before.UserAuthGeneration = int64(generation & math.MaxInt64)
		before.AttemptCount = int64(attempt & math.MaxInt64)
		before.NextAttemptAt = f.clock.Now().Add(time.Duration(offset) * time.Microsecond).Format(passwordResetMaintenanceTimeFormat)
		raw, err := json.Marshal(before)
		if err != nil {
			return false
		}
		var decoded PasswordResetRequeueSnapshot
		if decodePasswordResetMaintenanceJSON(raw, &decoded, passwordResetMaintenancePayloadLimit, true) != nil || !reflect.DeepEqual(before, decoded) {
			return false
		}
		after := passwordResetRequeueExpectedAfter(before, f.clock.Now())
		again := passwordResetRequeueExpectedAfter(after, f.clock.Now())
		return reflect.DeepEqual(after, again) && after.AttemptCount == before.AttemptCount && after.UserAuthGeneration == before.UserAuthGeneration && after.NextAttemptAt >= before.NextAttemptAt && after.JobUpdatedAt >= before.JobUpdatedAt
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatal(err)
	}
}

// The wrapper changes only the existing scripted transaction's return boundary.
type maintenanceR49BoundaryTx struct {
	*maintenanceTestTx
	afterCommit    func(context.Context)
	duringRollback func(context.Context)
	acknowledged   bool
}

func (tx *maintenanceR49BoundaryTx) Commit(ctx context.Context) error {
	err := tx.maintenanceTestTx.Commit(ctx)
	tx.acknowledged = err == nil
	if tx.afterCommit != nil {
		tx.afterCommit(ctx)
	}
	return err
}
func (tx *maintenanceR49BoundaryTx) Rollback(ctx context.Context) error {
	err := tx.maintenanceTestTx.Rollback(ctx)
	if tx.duringRollback != nil {
		tx.duringRollback(ctx)
	}
	return err
}

// Models a deadline timer whose callback is delayed. The deadline itself stays
// immutable; final monotonic time, not Err alone, must reject stale success.
type maintenanceR49LateTimerContext struct {
	context.Context
	deadline time.Time
}

func (ctx maintenanceR49LateTimerContext) Deadline() (time.Time, bool) { return ctx.deadline, true }
func (maintenanceR49LateTimerContext) Done() <-chan struct{}           { return nil }
func (maintenanceR49LateTimerContext) Err() error                      { return nil }

func testMaintenanceR49FinalCancellation(t *testing.T) {
	for _, mode := range []string{"apply_success", "replay_success", "apply_cancel", "replay_cancel", "commit_error_cancel", "deadline_return", "late_timer_return"} {
		t.Run(mode, func(t *testing.T) { testMaintenanceR49CommitBoundary(t, mode) })
	}
}
func maintenanceR49CommitFixture(t *testing.T, mode string) (*maintenanceFixture, *PasswordResetRequeueRequest, *maintenanceR49BoundaryTx) {
	f := newMaintenanceFixture(t)
	if mode == "deadline_return" {
		ctx, cancel := context.WithTimeout(f.ctx, 100*time.Millisecond)
		t.Cleanup(cancel)
		f.ctx = ctx
	}
	if mode == "late_timer_return" {
		f.ctx = maintenanceR49LateTimerContext{Context: f.ctx, deadline: time.Now().Add(100 * time.Millisecond)}
	}
	ctx, cancel := context.WithCancel(f.ctx)
	t.Cleanup(cancel)
	if mode != "late_timer_return" {
		f.ctx = ctx
	}
	request := f.request(t)
	base := maintenanceApplyTx(f)
	if strings.HasPrefix(mode, "replay") {
		base = maintenanceReplayTx(t, f, request, nil)
	}
	tx := &maintenanceR49BoundaryTx{maintenanceTestTx: base}
	if strings.HasSuffix(mode, "cancel") {
		tx.afterCommit = func(context.Context) { cancel() }
	}
	if mode == "deadline_return" {
		tx.afterCommit = func(ctx context.Context) { <-ctx.Done() }
	}
	if mode == "late_timer_return" {
		tx.afterCommit = func(ctx context.Context) {
			deadline, _ := ctx.Deadline()
			timer := time.NewTimer(time.Until(deadline))
			defer timer.Stop()
			<-timer.C
		}
	}
	if mode == "commit_error_cancel" {
		base.commitErr = errors.New("synthetic acknowledged-commit failure control")
	}
	return f, request, tx
}

func testMaintenanceR49CommitBoundary(t *testing.T, mode string) {
	f, request, tx := maintenanceR49CommitFixture(t, mode)
	base := tx.maintenanceTestTx
	writes := 2
	if strings.HasPrefix(mode, "replay") {
		writes = 0
	}
	dependencies, starter := maintenanceDependencies(t, tx)
	result, err := f.operator.Requeue(f.ctx, request, dependencies)
	if strings.HasSuffix(mode, "success") {
		if err != nil || result.Receipt.Outcome != "applied" {
			t.Fatalf("timely success %+v %v", result, err)
		}
	} else {
		code := PasswordResetMaintenanceFailed
		if mode == "commit_error_cancel" {
			code = PasswordResetMaintenanceOutcomeUnknown
		}
		requireMaintenanceCode(t, err, code)
		if !reflect.DeepEqual(result, PasswordResetRequeueResult{}) {
			t.Fatal("cancellation returned a success result")
		}
		if mode == "late_timer_return" && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("final end did not detect delayed timer")
		}
	}
	rollback := 0
	if mode == "commit_error_cancel" {
		rollback = 1
	}
	if base.commits != 1 || base.rollbacks != rollback || base.executes != writes || starter.begins != 1 || base.index != len(base.steps) {
		t.Fatalf("commit/rollback/exec/begin/steps=%d/%d/%d/%d/%d", base.commits, base.rollbacks, base.executes, starter.begins, base.index)
	}
	if tx.acknowledged != (mode != "commit_error_cancel") {
		t.Fatal("clear commit fact changed")
	}
	stats := dependencies.Permit.Stats()
	if stats.Current != 0 || stats.Acquired != 1 || stats.Released != 1 {
		t.Fatalf("permit %+v", stats)
	}
}

func testMaintenanceR49CleanupClassification(t *testing.T) {
	start := time.Now()
	deadline := start.Add(time.Second)
	for _, test := range []struct {
		name    string
		end     time.Time
		expired error
		primary PasswordResetMaintenanceCode
		want    PasswordResetMaintenanceCode
	}{
		{"timely_success", start, nil, "", ""},
		{"final_end_success_overrun", deadline.Add(time.Nanosecond), nil, "", PasswordResetMaintenanceFailed},
		{"expired_context_is_sticky", start, context.DeadlineExceeded, "", PasswordResetMaintenanceFailed},
		{"unknown_priority", deadline.Add(time.Second), context.DeadlineExceeded, PasswordResetMaintenanceOutcomeUnknown, PasswordResetMaintenanceOutcomeUnknown},
		{"state_primary_preserved", deadline.Add(time.Second), nil, PasswordResetMaintenanceStateConflict, PasswordResetMaintenanceStateConflict},
		{"forbidden_primary_preserved", deadline.Add(time.Second), nil, PasswordResetMaintenanceForbidden, PasswordResetMaintenanceForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			var primary error
			if test.primary != "" {
				primary = passwordResetMaintenanceFailure(test.primary, errors.New("original private cause"))
			}
			cleanupErr := passwordResetRequeueCleanupFailure(deadline, test.end, test.expired)
			got := finishPasswordResetRequeueResult(context.Background(), primary, cleanupErr)
			if test.want == "" {
				if got != nil {
					t.Fatal(got)
				}
				return
			}
			requireMaintenanceCode(t, got, test.want)
			var overrun *passwordResetRequeueCleanupOverrun
			if !errors.As(got, &overrun) || overrun.deadline != deadline || overrun.ended != test.end || !errors.Is(got, context.DeadlineExceeded) {
				t.Fatal("sticky overrun or original monotonic value lost")
			}
		})
	}
}

func testMaintenanceR49RollbackJoin(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown_%t", unknown), func(t *testing.T) { testMaintenanceR49RollbackBoundary(t, unknown) })
	}
}
func testMaintenanceR49RollbackBoundary(t *testing.T, unknown bool) {
	f := newMaintenanceFixture(t)
	request := f.request(t)
	base := maintenanceApplyTx(f)
	code := PasswordResetMaintenanceStateConflict
	if unknown {
		base.commitErr = errors.New("synthetic commit error")
		code = PasswordResetMaintenanceOutcomeUnknown
	} else {
		base.steps = base.steps[:4]
		base.steps[3].tag = pgconn.NewCommandTag("UPDATE 0")
	}
	expired := make(chan time.Time, 1)
	release := make(chan struct{})
	var once sync.Once
	tx := &maintenanceR49BoundaryTx{maintenanceTestTx: base, duringRollback: func(ctx context.Context) { deadline, _ := ctx.Deadline(); <-ctx.Done(); expired <- deadline; <-release }}
	dependencies, starter := maintenanceDependencies(t, tx)
	done := make(chan struct{})
	var result PasswordResetRequeueResult
	var err error
	go func() { defer close(done); result, err = f.operator.Requeue(f.ctx, request, dependencies) }()
	defer func() { once.Do(func() { close(release) }); <-done }()
	deadline := <-expired
	if deadline.After(request.deadline.Add(f.config.ShutdownBudget)) {
		t.Error("rollback restarted K")
	}
	select {
	case <-done:
		t.Error("returned before actual Rollback join")
	default:
	}
	stats := dependencies.Permit.Stats()
	if stats.Current != 1 || stats.Released != 0 {
		t.Error("permit returned before rollback completion")
	}
	once.Do(func() { close(release) })
	<-done
	requireMaintenanceCode(t, err, code)
	var overrun *passwordResetRequeueCleanupOverrun
	if !errors.As(err, &overrun) || !overrun.ended.After(overrun.deadline) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("actual end/expired cleanup was not retained")
	}
	if !reflect.DeepEqual(result, PasswordResetRequeueResult{}) || base.rollbacks != 1 || starter.begins != 1 || base.index != len(base.steps) {
		t.Fatal("late cleanup changed result/transaction")
	}
	stats = dependencies.Permit.Stats()
	if stats.Current != 0 || stats.Acquired != 1 || stats.Released != 1 {
		t.Fatal("permit not actually returned")
	}
}

func testMaintenanceR49PriorityProperties(t *testing.T) {
	property := func(choice uint8, elapsed uint16, expired bool) bool {
		codes := []PasswordResetMaintenanceCode{"", PasswordResetMaintenanceOutcomeUnknown, PasswordResetMaintenanceStateConflict, PasswordResetMaintenanceForbidden, PasswordResetMaintenanceFailed}
		code := codes[int(choice)%len(codes)]
		var primary error
		if code != "" {
			primary = passwordResetMaintenanceFailure(code, nil)
		}
		deadline := time.Now().Add(time.Second)
		end := deadline.Add(time.Duration(elapsed) * time.Nanosecond)
		var ctxErr error
		if expired {
			ctxErr = context.DeadlineExceeded
		}
		cleanupErr := passwordResetRequeueCleanupFailure(deadline, end, ctxErr)
		got := finishPasswordResetRequeueResult(context.Background(), primary, cleanupErr)
		if cleanupErr == nil {
			return got == primary
		}
		var failure *PasswordResetMaintenanceError
		if !errors.As(got, &failure) || !errors.Is(got, context.DeadlineExceeded) {
			return false
		}
		if code == "" {
			code = PasswordResetMaintenanceFailed
		}
		return failure.Code == code
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordResetRecoverySettlement_AuthorityMismatch_NoSQL(t *testing.T) {
	for _, name := range []string{"missing-metadata", "wrong-actor", "base-generation", "target-generation", "current-pointer", "hold"} {
		t.Run(name, func(t *testing.T) {
			approval, binding, _ := s412ApprovalFixture()
			u, _ := s412TargetFixture()
			id := "44444444-4444-4444-8444-444444444444"
			u.PasswordResetCurrentOperationID = &id
			operation := PasswordResetOperation{ID: id, UserID: u.ID, Recovery: recoveryRestoreMetadata(approval), BaseAuthGeneration: u.AuthGeneration, TargetAuthGeneration: u.AuthGeneration + 1}
			switch name {
			case "missing-metadata":
				operation.Recovery = nil
			case "wrong-actor":
				operation.Recovery.ActorID = u.ID
			case "base-generation":
				operation.BaseAuthGeneration++
			case "target-generation":
				operation.TargetAuthGeneration++
			case "current-pointer":
				u.PasswordResetCurrentOperationID = nil
			case "hold":
				u.AuthIntegrityHold = true
			}
			owner := recoveryRestoredSettlement{execution: &recoveryIdentityExecution{d: recoveryExecutionDependencies{clock: passwordResetClockStub{}}}}
			err := settleRecoveryRestoredReset(context.Background(), nil, owner, PasswordResetRecoveryLease{}, operation, approval, u, binding)
			if !errors.Is(err, ErrPasswordResetStateConflict) {
				t.Fatalf("bad authority reached SQL: %v", err)
			}
		})
	}
}

package identity

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"han/internal/idempotency"
	"han/internal/migration"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPasswordResetGate_ClearClosedAndQueryFailure(t *testing.T) {
	queryCause := errors.New("synthetic Gate query failure")
	tests := []struct {
		name      string
		step      passwordResetStoreStep
		wantError error
	}{
		{
			name: "clear terminal authority",
			step: passwordResetQueryStep("FROM users current_account", true),
		},
		{
			name:      "closed unresolved or integrity state",
			step:      passwordResetQueryStep("FROM users current_account", false),
			wantError: ErrPasswordResetGateClosed,
		},
		{
			name:      "query failure",
			step:      passwordResetQueryErrorStep("FROM users current_account", queryCause),
			wantError: queryCause,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &passwordResetGateTransactionStub{
				row: passwordResetStoreRow{
					values: test.step.values,
					err:    test.step.err,
				},
			}

			err := (PasswordResetStore{}).CheckLockedUser(
				context.Background(),
				tx,
				"user-one",
			)

			if test.wantError == nil && err != nil {
				t.Fatalf("CheckLockedUser() error = %v", err)
			}
			if test.wantError != nil && !errors.Is(err, test.wantError) {
				t.Fatalf("CheckLockedUser() error = %v, want %v", err, test.wantError)
			}
			if tx.execCalls != 0 {
				t.Fatalf("CheckLockedUser() writes = %d, want 0", tx.execCalls)
			}
			if !strings.Contains(tx.query, test.step.contains) {
				t.Fatalf("CheckLockedUser() query = %q", tx.query)
			}
		})
	}
}

func TestPasswordResetGate_SQLCoversUnresolvedHoldAndCurrentAuthority(t *testing.T) {
	for _, predicate := range []string{
		"NOT current_account.auth_integrity_hold",
		"unresolved.state NOT IN ('succeeded', 'invalidated', 'failed')",
		"= current_account.password_reset_current_operation_id",
		"operation.settlement_state = 'temporary_active'",
		"operation.settlement_state = 'permanent_successor'",
		"credential.source_operation_id = operation.operation_id",
		"current_account.permanent_password_generation",
		"= current_account.auth_generation",
	} {
		if !strings.Contains(passwordResetGateSQL, predicate) {
			t.Fatalf("passwordResetGateSQL missing predicate %q", predicate)
		}
	}
}

type passwordResetGateTransactionStub struct {
	row       pgx.Row
	query     string
	execCalls int
}

func (*passwordResetGateTransactionStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}

func (*passwordResetGateTransactionStub) Commit(context.Context) error { return nil }

func (*passwordResetGateTransactionStub) Rollback(context.Context) error { return nil }

func (*passwordResetGateTransactionStub) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("CopyFrom is not supported")
}

func (*passwordResetGateTransactionStub) SendBatch(
	context.Context,
	*pgx.Batch,
) pgx.BatchResults {
	return nil
}

func (*passwordResetGateTransactionStub) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (*passwordResetGateTransactionStub) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("Prepare is not supported")
}

func (tx *passwordResetGateTransactionStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	tx.execCalls++
	return pgconn.CommandTag{}, errors.New("unexpected write")
}

func (*passwordResetGateTransactionStub) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	return nil, errors.New("Query is not supported")
}

func (tx *passwordResetGateTransactionStub) QueryRow(
	_ context.Context,
	query string,
	_ ...any,
) pgx.Row {
	tx.query = query
	return tx.row
}

func (*passwordResetGateTransactionStub) Conn() *pgx.Conn { return nil }

func TestPasswordResetDurableMigrationDefinesAtomicAuthority(t *testing.T) {
	t.Helper()

	root := passwordResetRepositoryRoot(t)
	migrationSQL, err := os.ReadFile(
		filepath.Join(root, "migrations", "0008_password_reset_durable.sql"),
	)
	if err != nil {
		t.Fatalf("read password reset durable migration: %v", err)
	}
	sql := string(migrationSQL)
	required := []string{
		"password_reset_execution_generation",
		"auth_generation",
		"auth_integrity_hold",
		"CREATE TABLE password_reset_operations",
		"CREATE TABLE password_reset_recovery_jobs",
		"DEFERRABLE INITIALLY DEFERRED",
		"idempotency_records_password_reset_identity_unique",
		"request_kind IN ('initial_reset', 'resend')",
		"settlement_state IN",
		"delivery_state IN",
		"status IN ('ready', 'leased', 'retry_wait', 'succeeded', 'dead_letter')",
		"temporary_credentials_password_reset_pending_idx",
		"password_reset_operations_unresolved_user_idx",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
	loaded, err := migration.Load(os.DirFS(root), "migrations")
	if err != nil {
		t.Fatalf("load migrations through runner scanner: %v", err)
	}
	// The durable password-reset migration must load through the runner scanner.
	// Later forward migrations (for example the sanctioned article replay attempt
	// migration 0009) legitimately follow it, so assert presence rather than a
	// brittle "008 is the newest version" ordering.
	var foundDurable bool
	for _, loadedMigration := range loaded {
		if loadedMigration.Version == 8 {
			foundDurable = true
			break
		}
	}
	if !foundDurable {
		t.Fatalf("loaded migrations = %#v, want version 8 present", loaded)
	}
}

func TestPasswordResetDurableMigrationEnforcesCommitTimeFiveObjectInvariant(
	t *testing.T,
) {
	sql := passwordResetMigrationSQL(t)
	required := []string{
		"password_reset_current_operation_id",
		"password_reset_operation_id",
		"password_reset_user_id",
		"password_reset_operations_current_user_unique",
		"idempotency_records_password_reset_operation_fk",
		"password_reset_operations_recovery_job_fk",
		"CREATE CONSTRAINT TRIGGER password_reset_user_counter_complete",
		"CREATE CONSTRAINT TRIGGER password_reset_operation_counter_complete",
		"DEFERRABLE INITIALLY DEFERRED",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("commit-time invariant missing %q", fragment)
		}
	}
}

func TestPasswordResetDurableMigrationPreflightFailClosedMatrix(t *testing.T) {
	sql := passwordResetMigrationSQL(t)
	createOperation := strings.Index(sql, "CREATE TABLE password_reset_operations")
	if createOperation < 0 {
		t.Fatal("password reset operation table is missing")
	}
	preflight := sql[:createOperation]
	fixtures := []passwordResetMigrationPreflightFixture{
		{name: "empty database"},
		{name: "administrator legacy succeeded", records: []passwordResetLegacyClaim{{
			status: "succeeded", resultCode: "created",
			snapshot: `{"outcome":"created","resource_id":"administrator-one"}`,
		}}},
		{name: "administrator v1 succeeded", records: []passwordResetLegacyClaim{{
			status: "succeeded", resultCode: "created",
			snapshot: `{"schema_version":"identity-administrator-create-response/v1",` +
				`"http_status":201,"result_code":"created","body":{` +
				`"id":"administrator-one","username":"ops","role":"admin",` +
				`"status":"active","must_change_password":true,` +
				`"must_acknowledge_privacy":true,` +
				`"temporary_credential_expires_at":"2026-10-24T08:00:00Z",` +
				`"delivery_status":"accepted"}}`,
		}}},
		{name: "administrator legacy extra field", blocked: true, records: []passwordResetLegacyClaim{{
			status: "succeeded", resultCode: "created",
			snapshot: `{"outcome":"created","resource_id":"administrator-one","trace_id":"old"}`,
		}}},
		{name: "administrator v1 wrong capability", blocked: true, records: []passwordResetLegacyClaim{{
			status: "succeeded", resultCode: "created",
			snapshot: `{"schema_version":"identity-administrator-create-response/v1",` +
				`"http_status":201,"result_code":"created","body":{` +
				`"id":"administrator-one","username":"ops","role":"admin",` +
				`"status":"active","must_change_password":false,` +
				`"must_acknowledge_privacy":true,` +
				`"temporary_credential_expires_at":"2026-10-24T08:00:00Z",` +
				`"delivery_status":"accepted"}}`,
		}}},
		{name: "password reset processing", blocked: true, records: []passwordResetLegacyClaim{{
			status: "processing",
		}}},
		{name: "password reset uncertain", blocked: true, records: []passwordResetLegacyClaim{{
			status: "uncertain",
		}}},
		{name: "password reset succeeded", blocked: true, records: []passwordResetLegacyClaim{{
			status: "succeeded", resultCode: "password_reset",
			snapshot: `{"outcome":"password_reset","resource_id":"user-one"}`,
		}}},
		{name: "unknown succeeded", blocked: true, records: []passwordResetLegacyClaim{{
			status: "succeeded", resultCode: "other",
			snapshot: `{"outcome":"other","resource_id":"resource-one"}`,
		}}},
		{name: "legacy current credential", blocked: true, legacyCurrentCredential: true},
		{name: "normal and abnormal mixed", blocked: true, records: []passwordResetLegacyClaim{
			{
				status: "succeeded", resultCode: "created",
				snapshot: `{"outcome":"created","resource_id":"administrator-one"}`,
			},
			{status: "uncertain"},
		}},
	}
	classifier := passwordResetMigrationPreflightClassifier(t, preflight)
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			if got := classifier(fixture); got != fixture.blocked {
				t.Fatalf("preflight blocked = %v, want %v", got, fixture.blocked)
			}
		})
	}
}

func TestPasswordResetDurableMigrationClosesAuthorityStateMatrix(t *testing.T) {
	sql := passwordResetMigrationSQL(t)
	for _, fragment := range []string{
		"password_reset_operations_authority_check",
		"state = 'prepared'",
		"state = 'dispatching'",
		"state = 'commit_unknown'",
		"state IN ('recovery_required', 'converging')",
		"state = 'succeeded'",
		"state = 'invalidated'",
		"state = 'failed'",
		"password reset terminal evidence is immutable",
		"password reset settlement cannot move backwards",
		"password reset delivery cannot move backwards",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("authority state matrix missing %q", fragment)
		}
	}
}

func TestPasswordResetDurableModelsAndStoreExist(t *testing.T) {
	t.Helper()

	root := passwordResetRepositoryRoot(t)
	model, err := os.ReadFile(filepath.Join(root, "internal", "identity", "model.go"))
	if err != nil {
		t.Fatalf("read identity model: %v", err)
	}
	store, err := os.ReadFile(
		filepath.Join(root, "internal", "identity", "password_reset_store.go"),
	)
	if err != nil {
		t.Fatalf("read password reset store: %v", err)
	}
	for _, fragment := range []string{
		"AuthGeneration",
		"PasswordResetExecutionGeneration",
		"AuthIntegrityHold",
		"type PasswordResetOperation struct",
		"type PasswordResetRecoveryJob struct",
	} {
		if !strings.Contains(string(model), fragment) {
			t.Errorf("identity model missing %q", fragment)
		}
	}
	for _, fragment := range []string{
		"BeginOrClassifyCandidate",
		"lockPasswordResetCandidateUserSQL",
		"insertPasswordResetOperationSQL",
		"insertPasswordResetRecoveryJobSQL",
	} {
		if !strings.Contains(string(store), fragment) {
			t.Errorf("password reset store missing %q", fragment)
		}
	}
}

func TestPasswordResetStoreCreatesFiveObjectsUserFirst(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				"user-one",
				int64(4),
				int64(8),
				false,
			),
			passwordResetQueryErrorStep(
				"FROM idempotency_records",
				pgx.ErrNoRows,
			),
			passwordResetQueryErrorStep(
				"FROM password_reset_operations",
				pgx.ErrNoRows,
			),
			passwordResetQueryStep(
				"state NOT IN ('succeeded', 'invalidated', 'failed')",
				false,
			),
			passwordResetExecStep(
				"password_reset_execution_generation",
				"UPDATE 1",
			),
			passwordResetExecStep(
				"INSERT INTO password_reset_operations",
				"INSERT 0 1",
			),
			passwordResetExecStep(
				"INSERT INTO temporary_credentials",
				"INSERT 0 1",
			),
			passwordResetExecStep(
				"INSERT INTO password_reset_recovery_jobs",
				"INSERT 0 1",
			),
			passwordResetExecStep(
				"INSERT INTO idempotency_records",
				"INSERT 0 1",
			),
		},
	}

	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateCreated ||
		result.Operation.ExecutionGeneration != 9 ||
		result.Operation.BaseAuthGeneration != 4 ||
		result.Operation.TargetAuthGeneration != 5 ||
		result.Claim.ExecutionGeneration != 9 {
		t.Fatalf("candidate result = %#v", result)
	}
	if len(tx.trace) != 9 || !strings.Contains(tx.trace[0], "FROM users") {
		t.Fatalf("transaction trace = %#v", tx.trace)
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreSameKeyReplayDoesNotAdvanceGeneration(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	expected, err := idempotency.NewPasswordResetClaim(
		candidate.IdempotencyRequest,
		3,
	)
	if err != nil {
		t.Fatalf("NewPasswordResetClaim() error = %v", err)
	}
	operationID := candidate.Operation.ID
	userID := candidate.Operation.UserID
	requestKind := candidate.Operation.RequestKind
	state := PasswordResetOperationSucceeded
	settlement := PasswordResetSettlementTemporaryActive
	delivery := PasswordResetDeliveryAccepted
	targetAuth := int64(7)
	settledAuth := targetAuth
	settledAt := candidate.IdempotencyRequest.Now
	terminalAt := candidate.IdempotencyRequest.Now
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				userID,
				int64(7),
				int64(11),
				false,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				passwordResetClaimRowValues(expected)...,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				expected.ScopeHash,
				expected.KeyLocator,
				expected.KeyDigest,
				expected.DigestKeyVersion,
				expected.RequestDigest,
				int64(3),
				&operationID,
				&userID,
				&requestKind,
				&state,
				&settlement,
				&delivery,
				targetAuth,
				&settledAuth,
				(*int64)(nil),
				&settledAt,
				&terminalAt,
				int64(3),
				true,
				true,
			),
		},
	}

	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateExisting ||
		result.Operation.ExecutionGeneration != 3 {
		t.Fatalf("same-key result = %#v", result)
	}
	if len(tx.trace) != 3 {
		t.Fatalf("same-key trace = %#v", tx.trace)
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreBadBindingSetsIntegrityHold(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	expected, err := idempotency.NewPasswordResetClaim(
		candidate.IdempotencyRequest,
		1,
	)
	if err != nil {
		t.Fatalf("NewPasswordResetClaim() error = %v", err)
	}
	operationID := candidate.Operation.ID
	otherUser := "user-two"
	requestKind := candidate.Operation.RequestKind
	state := PasswordResetOperationPrepared
	settlement := PasswordResetSettlementPending
	delivery := PasswordResetDeliveryNotStarted
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				candidate.Operation.UserID,
				int64(0),
				int64(0),
				false,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				passwordResetClaimRowValues(expected)...,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				expected.ScopeHash,
				expected.KeyLocator,
				expected.KeyDigest,
				expected.DigestKeyVersion,
				expected.RequestDigest,
				int64(1),
				&operationID,
				&otherUser,
				&requestKind,
				&state,
				&settlement,
				&delivery,
				int64(1),
				(*int64)(nil),
				(*int64)(nil),
				(*time.Time)(nil),
				(*time.Time)(nil),
				int64(1),
				false,
				true,
			),
			passwordResetExecStep("auth_integrity_hold = true", "UPDATE 1"),
		},
	}

	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateIntegrityHold {
		t.Fatalf("bad-binding result = %#v", result)
	}
	if operationArg := tx.execArguments[0][1]; operationArg != (*string)(nil) {
		t.Fatalf("cross-user hold operation argument = %#v", operationArg)
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreExactReplayBadAuthoritySetsIntegrityHold(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	expected, err := idempotency.NewPasswordResetClaim(
		candidate.IdempotencyRequest,
		1,
	)
	if err != nil {
		t.Fatalf("NewPasswordResetClaim() error = %v", err)
	}
	operationID := candidate.Operation.ID
	userID := candidate.Operation.UserID
	requestKind := candidate.Operation.RequestKind
	state := PasswordResetOperationSucceeded
	settlement := PasswordResetSettlementPending
	delivery := PasswordResetDeliveryAccepted
	terminalAt := candidate.IdempotencyRequest.Now
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				userID,
				int64(0),
				int64(1),
				false,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				passwordResetClaimRowValues(expected)...,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				expected.ScopeHash,
				expected.KeyLocator,
				expected.KeyDigest,
				expected.DigestKeyVersion,
				expected.RequestDigest,
				int64(1),
				&operationID,
				&userID,
				&requestKind,
				&state,
				&settlement,
				&delivery,
				int64(1),
				(*int64)(nil),
				(*int64)(nil),
				(*time.Time)(nil),
				&terminalAt,
				int64(1),
				true,
				true,
			),
			passwordResetExecStep("auth_integrity_hold = true", "UPDATE 1"),
		},
	}

	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateIntegrityHold {
		t.Fatalf("bad authority result = %#v", result)
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreSameKeyDifferentDigestIsConflictNotHold(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	original := candidate.IdempotencyRequest
	stored, err := idempotency.NewPasswordResetClaim(original, 1)
	if err != nil {
		t.Fatalf("NewPasswordResetClaim() error = %v", err)
	}
	candidate.IdempotencyRequest.Request = []byte(`{"target":"user-two"}`)
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				candidate.Operation.UserID,
				int64(0),
				int64(0),
				false,
			),
			passwordResetQueryStep(
				"FROM idempotency_records",
				passwordResetClaimRowValues(stored)...,
			),
		},
	}

	_, err = (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if !errors.Is(err, idempotency.ErrDigestConflict) {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if len(tx.execArguments) != 0 {
		t.Fatalf("digest conflict writes = %d", len(tx.execArguments))
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreStopsAtEveryAtomicWriteFailure(t *testing.T) {
	fragments := []string{
		"password_reset_execution_generation",
		"INSERT INTO password_reset_operations",
		"INSERT INTO temporary_credentials",
		"INSERT INTO password_reset_recovery_jobs",
		"INSERT INTO idempotency_records",
	}
	for failedIndex, fragment := range fragments {
		t.Run(fragment, func(t *testing.T) {
			candidate := passwordResetStoreTestCandidate(t)
			steps := []passwordResetStoreStep{
				passwordResetQueryStep(
					"FROM users",
					candidate.Operation.UserID,
					int64(0),
					int64(0),
					false,
				),
				passwordResetQueryErrorStep(
					"FROM idempotency_records",
					pgx.ErrNoRows,
				),
				passwordResetQueryErrorStep(
					"FROM password_reset_operations",
					pgx.ErrNoRows,
				),
				passwordResetQueryStep(
					"state NOT IN ('succeeded', 'invalidated', 'failed')",
					false,
				),
			}
			for index, writeFragment := range fragments {
				step := passwordResetExecStep(writeFragment, "INSERT 0 1")
				if index == 0 {
					step.tag = pgconn.NewCommandTag("UPDATE 1")
				}
				if index == failedIndex {
					if index == 0 {
						step.tag = pgconn.NewCommandTag("UPDATE 0")
					} else {
						step.tag = pgconn.NewCommandTag("INSERT 0 0")
					}
				}
				steps = append(steps, step)
			}
			tx := &passwordResetStoreTransactionStub{steps: steps}

			_, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
				context.Background(),
				tx,
				candidate,
			)
			if err == nil {
				t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
			}
			if failedIndex < len(fragments)-1 &&
				!errors.Is(err, ErrPasswordResetStateConflict) {
				t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
			}
			if got := len(tx.execArguments); got != failedIndex+1 {
				t.Fatalf("write calls = %d, want %d", got, failedIndex+1)
			}
			commitErr := tx.Commit()
			if failedIndex == 0 && commitErr != nil {
				t.Fatalf("all-none commit error = %v", commitErr)
			}
			if failedIndex > 0 && commitErr == nil {
				t.Fatal("partial authority commit succeeded")
			}
		})
	}
}

func TestPasswordResetCommitInvariantRejectsIndependentPartialShapes(t *testing.T) {
	tests := []struct {
		name    string
		applied map[string]bool
	}{
		{name: "counter-only", applied: map[string]bool{"counter": true}},
		{name: "claim-only", applied: map[string]bool{"claim": true}},
		{name: "operation-only", applied: map[string]bool{"operation": true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := &passwordResetStoreTransactionStub{applied: test.applied}
			if err := tx.Commit(); err == nil {
				t.Fatalf("%s commit succeeded", test.name)
			}
		})
	}
}

func TestPasswordResetStoreResendUsesLegalSuccessorLineage(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	predecessorID := "33333333-3333-3333-3333-333333333333"
	replacedID := "44444444-4444-4444-4444-444444444444"
	candidate.Operation.RequestKind = PasswordResetResend
	candidate.Operation.PredecessorOperationID = &predecessorID
	candidate.Operation.ReplacedCredentialID = &replacedID
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				candidate.Operation.UserID,
				int64(6),
				int64(3),
				false,
			),
			passwordResetQueryErrorStep(
				"FROM idempotency_records",
				pgx.ErrNoRows,
			),
			passwordResetQueryErrorStep(
				"FROM password_reset_operations",
				pgx.ErrNoRows,
			),
			passwordResetQueryStep(
				"state NOT IN ('succeeded', 'invalidated', 'failed')",
				false,
			),
			passwordResetQueryStep(
				"predecessor.settlement_state",
				passwordResetResendLineageRowValues(
					passwordResetResendLineageEvidenceFixture(
						candidate.Operation.UserID,
						predecessorID,
						replacedID,
						6,
					),
				)...,
			),
			passwordResetExecStep(
				"password_reset_execution_generation",
				"UPDATE 1",
			),
			passwordResetExecStep(
				"INSERT INTO password_reset_operations",
				"INSERT 0 1",
			),
			passwordResetExecStep(
				"INSERT INTO temporary_credentials",
				"INSERT 0 1",
			),
			passwordResetExecStep(
				"INSERT INTO password_reset_recovery_jobs",
				"INSERT 0 1",
			),
			passwordResetExecStep(
				"INSERT INTO idempotency_records",
				"INSERT 0 1",
			),
		},
	}

	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateCreated ||
		result.Operation.ExecutionGeneration != 4 ||
		result.Operation.BaseAuthGeneration != 6 ||
		result.Operation.TargetAuthGeneration != 7 ||
		result.Operation.PredecessorOperationID == nil ||
		result.Operation.ReplacedCredentialID == nil {
		t.Fatalf("resend successor = %#v", result.Operation)
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreResendLineageDriftSetsIntegrityHold(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	predecessorID := "33333333-3333-3333-3333-333333333333"
	replacedID := "44444444-4444-4444-4444-444444444444"
	candidate.Operation.RequestKind = PasswordResetResend
	candidate.Operation.PredecessorOperationID = &predecessorID
	candidate.Operation.ReplacedCredentialID = &replacedID
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				candidate.Operation.UserID,
				int64(6),
				int64(3),
				false,
			),
			passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
			passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
			passwordResetQueryStep(
				"state NOT IN ('succeeded', 'invalidated', 'failed')",
				false,
			),
			passwordResetQueryStep(
				"predecessor.settlement_state",
				passwordResetResendLineageRowValues(
					passwordResetResendConsumedWithoutSuccessorFixture(
						candidate.Operation.UserID,
						predecessorID,
						replacedID,
						6,
					),
				)...,
			),
			passwordResetExecStep("auth_integrity_hold = true", "UPDATE 1"),
		},
	}

	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateIntegrityHold {
		t.Fatalf("resend drift result = %#v", result)
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreResendStaleLineageIsOrdinaryRejection(t *testing.T) {
	candidate := passwordResetStoreTestCandidate(t)
	predecessorID := "33333333-3333-3333-3333-333333333333"
	replacedID := "44444444-4444-4444-4444-444444444444"
	candidate.Operation.RequestKind = PasswordResetResend
	candidate.Operation.PredecessorOperationID = &predecessorID
	candidate.Operation.ReplacedCredentialID = &replacedID
	tx := &passwordResetStoreTransactionStub{
		steps: []passwordResetStoreStep{
			passwordResetQueryStep(
				"FROM users",
				candidate.Operation.UserID,
				int64(7),
				int64(3),
				false,
			),
			passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
			passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
			passwordResetQueryStep(
				"state NOT IN ('succeeded', 'invalidated', 'failed')",
				false,
			),
			passwordResetQueryStep(
				"predecessor.settlement_state",
				passwordResetResendLineageRowValues(
					passwordResetResendLegalSuccessorFixture(
						candidate.Operation.UserID,
						predecessorID,
						replacedID,
						6,
					),
				)...,
			),
		},
	}

	_, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(),
		tx,
		candidate,
	)
	if !errors.Is(err, ErrPasswordResetGateClosed) {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if len(tx.execArguments) != 0 {
		t.Fatalf("stale lineage writes = %d", len(tx.execArguments))
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreLegalHistoryStaleAfterRootReplacement(t *testing.T) {
	const (
		userID        = "user-one"
		predecessorID = "33333333-3333-3333-3333-333333333333"
		replacedID    = "44444444-4444-4444-4444-444444444444"
	)
	tests := []struct {
		name     string
		evidence passwordResetResendLineageEvidence
	}{
		{
			name: "later password change retained historical root",
			evidence: passwordResetResendLaterPasswordChangeFixture(
				userID, predecessorID, replacedID, 4,
			),
		},
		{
			name: "later reset replaced permanent root",
			evidence: passwordResetResendLaterPermanentRootFixture(
				userID, predecessorID, replacedID, 4,
			),
		},
		{
			name: "later reset root followed by current temporary",
			evidence: passwordResetResendLaterPermanentRootTemporaryFixture(
				userID, predecessorID, replacedID, 4,
			),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPasswordResetHistoricalSameKeyReplay(
				t, test.evidence.authGeneration,
			)
			candidate := passwordResetStoreTestCandidate(t)
			candidate.Operation.RequestKind = PasswordResetResend
			candidate.Operation.PredecessorOperationID = passwordResetTestPointer(predecessorID)
			candidate.Operation.ReplacedCredentialID = passwordResetTestPointer(replacedID)
			tx := &passwordResetStoreTransactionStub{steps: []passwordResetStoreStep{
				passwordResetQueryStep(
					"FROM users", candidate.Operation.UserID,
					test.evidence.authGeneration, int64(3), false,
				),
				passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
				passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
				passwordResetQueryStep(
					"state NOT IN ('succeeded', 'invalidated', 'failed')", false,
				),
				passwordResetQueryStep(
					"predecessor.settlement_state",
					passwordResetResendLineageRowValues(test.evidence)...,
				),
			}}

			_, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
				context.Background(), tx, candidate,
			)
			if !errors.Is(err, ErrPasswordResetGateClosed) {
				t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
			}
			if len(tx.execArguments) != 0 {
				t.Fatalf("legal history stale writes = %d, want 0", len(tx.execArguments))
			}
			tx.assertExhausted(t)
		})
	}
}

func assertPasswordResetHistoricalSameKeyReplay(t *testing.T, authGeneration int64) {
	t.Helper()
	candidate := passwordResetStoreTestCandidate(t)
	expected, err := idempotency.NewPasswordResetClaim(
		candidate.IdempotencyRequest, 3,
	)
	if err != nil {
		t.Fatalf("NewPasswordResetClaim() error = %v", err)
	}
	operationID := candidate.Operation.ID
	userID := candidate.Operation.UserID
	requestKind := candidate.Operation.RequestKind
	state := PasswordResetOperationSucceeded
	settlement := PasswordResetSettlementPermanentSuccessor
	delivery := PasswordResetDeliveryAccepted
	targetAuth, successorAuth := int64(4), int64(5)
	settledAt := candidate.IdempotencyRequest.Now
	tx := &passwordResetStoreTransactionStub{steps: []passwordResetStoreStep{
		passwordResetQueryStep(
			"FROM users", userID, authGeneration, int64(11), false,
		),
		passwordResetQueryStep(
			"FROM idempotency_records", passwordResetClaimRowValues(expected)...,
		),
		passwordResetQueryStep(
			"FROM idempotency_records",
			expected.ScopeHash, expected.KeyLocator, expected.KeyDigest,
			expected.DigestKeyVersion, expected.RequestDigest, int64(3),
			&operationID, &userID, &requestKind, &state, &settlement, &delivery,
			targetAuth, &targetAuth, &successorAuth, &settledAt, &settledAt,
			int64(3), true, true,
		),
	}}
	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(), tx, candidate,
	)
	if err != nil || result.Action != PasswordResetCandidateExisting {
		t.Fatalf("same-key replay result/error = %#v/%v", result, err)
	}
	if len(tx.execArguments) != 0 {
		t.Fatalf("same-key replay writes = %d, want 0", len(tx.execArguments))
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreResendSQLRequiresCurrentLineage(t *testing.T) {
	if strings.Contains(
		passwordResetResendLineageStatusSQL,
		"WHEN replaced.state IN ('superseded', 'consumed', 'recovery_invalidated')",
	) {
		t.Fatal("resend lineage SQL blanket-classifies terminal credentials as ordinary stale")
	}
	for _, fragment := range []string{
		"current_account.credential_mode",
		"current_account.auth_generation",
		"current_account.permanent_password_digest IS NOT NULL",
		"current_account.permanent_password_generation",
		"current_account.permanent_password_source_operation_id",
		"current_account.permanent_password_source_credential_id",
		"permanent_operation.operation_id",
		"permanent_credential.source_operation_id",
		"permanent_credential.generation",
		"predecessor.temporary_credential_id",
		"predecessor.settled_auth_generation",
		"predecessor.successor_auth_generation",
		"predecessor.delivery_attempt_id",
		"replaced.auth_generation",
		"replaced.source_operation_id",
		"current_credential.generation",
		"current_credential.generation_ordinal",
		"current_credential.source_operation_id",
		"current_operation.predecessor_operation_id",
		"current_operation.replaced_credential_id",
		"child.predecessor_operation_id = parent.operation_id",
		"parent.lineage_credential_state = 'superseded'",
		"lineage.operation_id = $2",
		"lineage.credential_id = $3",
		"lineage.base_auth_generation",
	} {
		if !strings.Contains(passwordResetResendLineageStatusSQL, fragment) {
			t.Errorf("resend lineage SQL missing %q", fragment)
		}
	}
}

func TestPasswordResetStoreResendLineageClassificationMatrix(t *testing.T) {
	const (
		userID        = "user-one"
		predecessorID = "33333333-3333-3333-3333-333333333333"
		replacedID    = "44444444-4444-4444-4444-444444444444"
	)
	base := passwordResetResendLineageEvidenceFixture(
		userID, predecessorID, replacedID, 4,
	)
	tests := []struct {
		name       string
		evidence   passwordResetResendLineageEvidence
		wantStatus passwordResetResendLineageClassification
		wantHold   *string
	}{
		{name: "current active", evidence: base, wantStatus: passwordResetResendLineageEligible},
		{name: "current expired", evidence: passwordResetResendWithCredentialState(
			base, TemporaryCredentialExpired,
		), wantStatus: passwordResetResendLineageDrift,
			wantHold: passwordResetTestPointer(predecessorID)},
		{name: "consumed without permanent settlement",
			evidence: passwordResetResendConsumedWithoutSuccessorFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift, wantHold: passwordResetTestPointer(predecessorID)},
		{name: "permanent successor wrong current generation",
			evidence: passwordResetResendPermanentWrongGenerationFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift, wantHold: passwordResetTestPointer(predecessorID)},
		{name: "permanent successor wrong current mode",
			evidence: passwordResetResendPermanentWrongModeFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift, wantHold: passwordResetTestPointer(predecessorID)},
		{name: "cross user predecessor",
			evidence:   passwordResetResendCrossUserFixture(base),
			wantStatus: passwordResetResendLineageDrift},
		{name: "missing predecessor",
			evidence:   passwordResetResendMissingPredecessorFixture(base),
			wantStatus: passwordResetResendLineageDrift},
		{name: "missing exact credential",
			evidence:   passwordResetResendMissingCredentialFixture(base),
			wantStatus: passwordResetResendLineageDrift, wantHold: passwordResetTestPointer(predecessorID)},
		{name: "permanent successor without current root evidence",
			evidence: passwordResetResendCurrentPermanentFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "permanent successor wrong root operation",
			evidence: passwordResetResendPermanentWrongRootOperationFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "permanent successor wrong root credential",
			evidence: passwordResetResendPermanentWrongRootCredentialFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "permanent successor wrong permanent generation",
			evidence: passwordResetResendPermanentWrongGenerationFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "normal consumed permanent successor",
			evidence: passwordResetResendPermanentRootFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageIneligible},
		{name: "normal later password change",
			evidence: passwordResetResendLaterPasswordChangeFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageIneligible},
		{name: "normal later password reset",
			evidence: passwordResetResendLaterResetFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageIneligible},
		{name: "superseded stale without current successor evidence",
			evidence: passwordResetResendSupersededFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "superseded stale successor not succeeded",
			evidence: passwordResetResendSuccessorWrongStateFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "superseded stale successor not temporary active",
			evidence: passwordResetResendSuccessorWrongSettlementFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "superseded stale successor cross user",
			evidence: passwordResetResendSuccessorCrossUserFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "superseded stale successor credential mismatch",
			evidence: passwordResetResendSuccessorCredentialMismatchFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageDrift,
			wantHold:   passwordResetTestPointer(predecessorID)},
		{name: "normal superseded stale after legal reset",
			evidence: passwordResetResendLegalSuccessorFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageIneligible},
		{name: "normal recovery invalidated stale",
			evidence: passwordResetResendRecoveryInvalidatedFixture(
				userID, predecessorID, replacedID, 4,
			),
			wantStatus: passwordResetResendLineageIneligible},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := classifyPasswordResetResendLineage(test.evidence)
			if got.status != test.wantStatus ||
				!reflect.DeepEqual(got.holdOperationID, test.wantHold) {
				t.Fatalf("classification = %#v, want status=%q hold=%#v",
					got, test.wantStatus, test.wantHold)
			}
		})
	}
}

func TestPasswordResetStoreResendCurrentCapabilityDriftSetsDurableHold(t *testing.T) {
	const (
		userID        = "user-one"
		predecessorID = "33333333-3333-3333-3333-333333333333"
		replacedID    = "44444444-4444-4444-4444-444444444444"
	)
	tests := []struct {
		name      string
		permanent bool
		mutate    func(*passwordResetResendLineageEvidence)
	}{
		{name: "CP-01 temporary mode points at permanent", mutate: func(e *passwordResetResendLineageEvidence) {
			e.credentialMode = CredentialModePermanent
		}},
		{name: "CP-01 permanent mode points at temporary", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.credentialMode = CredentialModeTemporary
		}},
		{name: "CP-01 unknown mode", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.credentialMode = CredentialMode("unknown")
		}},
		{name: "CP-T01 missing operation credential pointer", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentOperationCredentialID = nil
		}},
		{name: "CP-T01 conflicting operation credential pointer", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentOperationCredentialID = passwordResetTestPointer("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
		}},
		{name: "CP-T02 missing current credential row", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialID = nil
		}},
		{name: "CP-T03 expired current credential", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialState = passwordResetTestPointer(TemporaryCredentialExpired)
		}},
		{name: "CP-T03 pending current credential", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialState = passwordResetTestPointer(TemporaryCredentialPendingDelivery)
		}},
		{name: "CP-T03 superseded current credential", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialState = passwordResetTestPointer(TemporaryCredentialSuperseded)
		}},
		{name: "CP-T03 consumed current credential", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialState = passwordResetTestPointer(TemporaryCredentialConsumed)
		}},
		{name: "CP-T03 recovery-invalidated current credential", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialState = passwordResetTestPointer(TemporaryCredentialRecoveryInvalidated)
		}},
		{name: "CP-T04 missing source operation", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialSourceOperationID = nil
		}},
		{name: "CP-T04 conflicting source operation", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialSourceOperationID = passwordResetTestPointer(predecessorID)
		}},
		{name: "CP-T05 invalid credential generation", mutate: func(e *passwordResetResendLineageEvidence) {
			e.currentCredentialGeneration = passwordResetTestPointer(int64(0))
		}},
		{name: "CP-T05 conflicting credential generation", mutate: func(e *passwordResetResendLineageEvidence) {
			(*e.currentCredentialGeneration)++
		}},
		{name: "CP-T05 generation sequence conflict", mutate: func(e *passwordResetResendLineageEvidence) {
			(*e.currentCredentialGenerationOrdinal)--
		}},
		{name: "CP-T05 wrong credential auth generation", mutate: func(e *passwordResetResendLineageEvidence) {
			*e.currentCredentialAuth++
		}},
		{name: "CP-T05 wrong operation target generation", mutate: func(e *passwordResetResendLineageEvidence) {
			*e.currentTargetAuth++
		}},
		{name: "CP-T05 wrong user auth generation", mutate: func(e *passwordResetResendLineageEvidence) {
			e.authGeneration++
		}},
		{name: "CP-P01 missing permanent digest", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentDigestPresent = false
		}},
		{name: "CP-P01 missing permanent generation", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentGeneration = nil
		}},
		{name: "CP-P01 wrong permanent generation", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			*e.permanentGeneration++
		}},
		{name: "CP-P02 missing permanent operation source", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentSourceOperationID = nil
		}},
		{name: "CP-P02 unknown permanent operation source", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentSourceOperationID = passwordResetTestPointer("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
		}},
		{name: "CP-P03 missing permanent credential source", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentSourceCredentialID = nil
		}},
		{name: "CP-P03 unknown permanent credential source", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentSourceCredentialID = passwordResetTestPointer("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
		}},
		{name: "CP-P04 source operation cross-user", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentOperationUserID = passwordResetTestPointer("user-two")
		}},
		{name: "CP-P04 operation exact credential mismatch", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentOperationCredentialID = passwordResetTestPointer(replacedID)
		}},
		{name: "CP-P04 credential source mismatch", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentCredentialSourceOperationID = passwordResetTestPointer(predecessorID)
		}},
		{name: "CP-P04 operation is not succeeded", permanent: true, mutate: func(e *passwordResetResendLineageEvidence) {
			e.permanentOperationState = passwordResetTestPointer(PasswordResetOperationRecoveryRequired)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evidence := passwordResetResendLaterPermanentRootTemporaryFixture(
				userID, predecessorID, replacedID, 4,
			)
			if test.permanent {
				evidence = passwordResetResendLaterPermanentRootFixture(
					userID, predecessorID, replacedID, 4,
				)
			}
			test.mutate(&evidence)
			assertPasswordResetResendDriftHolds(
				t, predecessorID, replacedID, evidence,
			)
		})
	}
}

func assertPasswordResetResendDriftHolds(
	t *testing.T,
	predecessorID string,
	replacedID string,
	evidence passwordResetResendLineageEvidence,
) {
	t.Helper()
	candidate := passwordResetStoreTestCandidate(t)
	candidate.Operation.RequestKind = PasswordResetResend
	candidate.Operation.PredecessorOperationID = &predecessorID
	candidate.Operation.ReplacedCredentialID = &replacedID
	tx := &passwordResetStoreTransactionStub{steps: []passwordResetStoreStep{
		passwordResetQueryStep(
			"FROM users", candidate.Operation.UserID,
			evidence.authGeneration, int64(3), false,
		),
		passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
		passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
		passwordResetQueryStep(
			"state NOT IN ('succeeded', 'invalidated', 'failed')", false,
		),
		passwordResetQueryStep(
			"predecessor.settlement_state",
			passwordResetResendLineageRowValues(evidence)...,
		),
		passwordResetExecStep("auth_integrity_hold = true", "UPDATE 1"),
	}}
	result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
		context.Background(), tx, candidate,
	)
	if err != nil {
		t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
	}
	if result.Action != PasswordResetCandidateIntegrityHold {
		t.Fatalf("resend drift result = %#v", result)
	}
	if len(tx.execArguments) != 1 {
		t.Fatalf("resend drift writes = %d, want one atomic hold/audit write", len(tx.execArguments))
	}
	if !strings.Contains(tx.trace[len(tx.trace)-1], "INSERT INTO audit_events") {
		t.Fatal("resend drift did not atomically append redacted audit")
	}
	for _, argument := range tx.execArguments[0] {
		if argument == candidate.Credential.SecretDigest ||
			argument == candidate.Credential.DeliveredToEmail {
			t.Fatal("resend drift audit arguments contain credential material")
		}
	}
	tx.assertExhausted(t)
}

func TestPasswordResetStoreResendDriftUsesOnlyProvenSameUserHoldPointer(t *testing.T) {
	tests := []struct {
		name            string
		predecessorUser string
		wantOperationID *string
	}{
		{name: "same user", predecessorUser: "user-one"},
		{name: "cross user", predecessorUser: "user-two"},
		{name: "missing predecessor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := passwordResetStoreTestCandidate(t)
			predecessorID := "33333333-3333-3333-3333-333333333333"
			replacedID := "44444444-4444-4444-4444-444444444444"
			candidate.Operation.RequestKind = PasswordResetResend
			candidate.Operation.PredecessorOperationID = &predecessorID
			candidate.Operation.ReplacedCredentialID = &replacedID
			if test.predecessorUser == candidate.Operation.UserID {
				test.wantOperationID = &predecessorID
			}
			evidence := passwordResetResendLineageEvidenceFixture(
				candidate.Operation.UserID,
				predecessorID,
				replacedID,
				6,
			)
			evidence = passwordResetResendConsumedWithoutSuccessor(evidence)
			switch test.predecessorUser {
			case "":
				evidence = passwordResetResendMissingPredecessorFixture(evidence)
			case candidate.Operation.UserID:
			default:
				evidence.predecessorUserID = passwordResetTestPointer(test.predecessorUser)
			}
			tx := &passwordResetStoreTransactionStub{
				steps: []passwordResetStoreStep{
					passwordResetQueryStep(
						"FROM users", candidate.Operation.UserID,
						int64(6), int64(3), false,
					),
					passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
					passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
					passwordResetQueryStep(
						"state NOT IN ('succeeded', 'invalidated', 'failed')", false,
					),
					passwordResetQueryStep(
						"predecessor.settlement_state",
						passwordResetResendLineageRowValues(evidence)...,
					),
					passwordResetExecStep("auth_integrity_hold = true", "UPDATE 1"),
				},
			}

			result, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
				context.Background(), tx, candidate,
			)
			if err != nil {
				t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
			}
			if result.Action != PasswordResetCandidateIntegrityHold {
				t.Fatalf("resend drift result = %#v", result)
			}
			if len(tx.execArguments) != 1 {
				t.Fatalf("resend drift writes = %d, want hold only", len(tx.execArguments))
			}
			if got := tx.execArguments[0][1]; !reflect.DeepEqual(got, test.wantOperationID) {
				t.Fatalf("hold operation argument = %#v, want %#v", got, test.wantOperationID)
			}
			tx.assertExhausted(t)
		})
	}
}

func TestPasswordResetStoreTransientFailuresDoNotPersistDrift(t *testing.T) {
	t.Run("user lock", func(t *testing.T) {
		candidate := passwordResetStoreTestCandidate(t)
		tx := &passwordResetStoreTransactionStub{steps: []passwordResetStoreStep{
			passwordResetQueryErrorStep("FROM users", errors.New("synthetic lock failure")),
		}}
		if _, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
			context.Background(), tx, candidate,
		); err == nil {
			t.Fatal("BeginOrClassifyCandidate() error = nil")
		}
		if len(tx.execArguments) != 0 {
			t.Fatalf("user lock failure writes = %d", len(tx.execArguments))
		}
		tx.assertExhausted(t)
	})
	t.Run("lineage query", func(t *testing.T) {
		candidate := passwordResetStoreTestCandidate(t)
		predecessorID := "33333333-3333-3333-3333-333333333333"
		replacedID := "44444444-4444-4444-4444-444444444444"
		candidate.Operation.RequestKind = PasswordResetResend
		candidate.Operation.PredecessorOperationID = &predecessorID
		candidate.Operation.ReplacedCredentialID = &replacedID
		tx := &passwordResetStoreTransactionStub{steps: []passwordResetStoreStep{
			passwordResetQueryStep("FROM users", candidate.Operation.UserID, int64(8), int64(3), false),
			passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
			passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
			passwordResetQueryStep("state NOT IN ('succeeded', 'invalidated', 'failed')", false),
			passwordResetQueryErrorStep("predecessor.settlement_state", errors.New("synthetic query failure")),
		}}
		if _, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
			context.Background(), tx, candidate,
		); err == nil {
			t.Fatal("BeginOrClassifyCandidate() error = nil")
		}
		if len(tx.execArguments) != 0 {
			t.Fatalf("lineage query failure writes = %d", len(tx.execArguments))
		}
		tx.assertExhausted(t)
	})
	t.Run("caller commit", func(t *testing.T) {
		const predecessorID = "33333333-3333-3333-3333-333333333333"
		const replacedID = "44444444-4444-4444-4444-444444444444"
		evidence := passwordResetResendLaterPermanentRootTemporaryFixture(
			"user-one", predecessorID, replacedID, 4,
		)
		candidate := passwordResetStoreTestCandidate(t)
		candidate.Operation.RequestKind = PasswordResetResend
		candidate.Operation.PredecessorOperationID = passwordResetTestPointer(predecessorID)
		candidate.Operation.ReplacedCredentialID = passwordResetTestPointer(replacedID)
		tx := &passwordResetStoreTransactionStub{
			commitErr: errors.New("synthetic commit failure"),
			steps: []passwordResetStoreStep{
				passwordResetQueryStep("FROM users", candidate.Operation.UserID, evidence.authGeneration, int64(3), false),
				passwordResetQueryErrorStep("FROM idempotency_records", pgx.ErrNoRows),
				passwordResetQueryErrorStep("FROM password_reset_operations", pgx.ErrNoRows),
				passwordResetQueryStep("state NOT IN ('succeeded', 'invalidated', 'failed')", false),
				passwordResetQueryStep("predecessor.settlement_state", passwordResetResendLineageRowValues(evidence)...),
			},
		}
		if _, err := (PasswordResetStore{}).BeginOrClassifyCandidate(
			context.Background(), tx, candidate,
		); !errors.Is(err, ErrPasswordResetGateClosed) {
			t.Fatalf("BeginOrClassifyCandidate() error = %v", err)
		}
		if err := tx.Commit(); err == nil {
			t.Fatal("Commit() error = nil")
		}
		if len(tx.execArguments) != 0 {
			t.Fatalf("commit failure writes = %d", len(tx.execArguments))
		}
		tx.assertExhausted(t)
	})
}

func TestPasswordResetOperationAuthorityMatrix(t *testing.T) {
	now := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	target := int64(4)
	successor := int64(5)
	tests := []struct {
		name      string
		operation PasswordResetOperation
		valid     bool
	}{
		{name: "prepared", operation: passwordResetAuthorityOperation(
			PasswordResetOperationPrepared, PasswordResetDeliveryNotStarted,
			PasswordResetSettlementPending, target, nil, nil, nil, nil,
		), valid: true},
		{name: "dispatching", operation: passwordResetAuthorityOperation(
			PasswordResetOperationDispatching, PasswordResetDeliveryDispatching,
			PasswordResetSettlementPending, target, nil, nil, nil, nil,
		), valid: true},
		{name: "commit unknown", operation: passwordResetAuthorityOperation(
			PasswordResetOperationCommitUnknown, PasswordResetDeliveryUnknown,
			PasswordResetSettlementPending, target, nil, nil, nil, nil,
		), valid: true},
		{name: "recovery required", operation: passwordResetAuthorityOperation(
			PasswordResetOperationRecoveryRequired, PasswordResetDeliveryAccepted,
			PasswordResetSettlementPending, target, nil, nil, nil, nil,
		), valid: true},
		{name: "succeeded temporary", operation: passwordResetAuthorityOperation(
			PasswordResetOperationSucceeded, PasswordResetDeliveryAccepted,
			PasswordResetSettlementTemporaryActive, target, &target, nil, &now, &now,
		), valid: true},
		{name: "succeeded permanent", operation: passwordResetAuthorityOperation(
			PasswordResetOperationSucceeded, PasswordResetDeliveryAccepted,
			PasswordResetSettlementPermanentSuccessor,
			target, &target, &successor, &now, &now,
		), valid: true},
		{name: "invalidated", operation: passwordResetAuthorityOperation(
			PasswordResetOperationInvalidated, PasswordResetDeliveryUnknown,
			PasswordResetSettlementInvalidated, target, nil, nil, &now, &now,
		), valid: true},
		{name: "failed", operation: passwordResetAuthorityOperation(
			PasswordResetOperationFailed, PasswordResetDeliveryRejected,
			PasswordResetSettlementInvalidated, target, nil, nil, &now, &now,
		), valid: true},
		{name: "succeeded pending", operation: passwordResetAuthorityOperation(
			PasswordResetOperationSucceeded, PasswordResetDeliveryAccepted,
			PasswordResetSettlementPending, target, nil, nil, nil, &now,
		)},
		{name: "failed accepted", operation: passwordResetAuthorityOperation(
			PasswordResetOperationFailed, PasswordResetDeliveryAccepted,
			PasswordResetSettlementInvalidated, target, nil, nil, &now, &now,
		)},
		{name: "terminal missing time", operation: passwordResetAuthorityOperation(
			PasswordResetOperationSucceeded, PasswordResetDeliveryAccepted,
			PasswordResetSettlementTemporaryActive, target, &target, nil, &now, nil,
		)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := passwordResetOperationAuthorityValid(test.operation); got != test.valid {
				t.Fatalf("authority valid = %v, want %v", got, test.valid)
			}
		})
	}
}

func TestPasswordResetStoreSecurityFunctionsStayWithinFiftyLines(t *testing.T) {
	root := passwordResetRepositoryRoot(t)
	path := filepath.Join(root, "internal", "identity", "password_reset_store.go")
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, path, nil, 0)
	if err != nil {
		t.Fatalf("parse password reset store: %v", err)
	}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		start := files.Position(function.Pos()).Line
		end := files.Position(function.End()).Line
		if lines := end - start + 1; lines > 50 {
			t.Errorf("%s spans %d lines, want <= 50", function.Name.Name, lines)
		}
	}
}

func TestPasswordResetOperationAndJobModelsExposeClosedStates(t *testing.T) {
	if PasswordResetOperationPrepared == PasswordResetOperationSucceeded ||
		PasswordResetSettlementPending == PasswordResetSettlementPermanentSuccessor ||
		PasswordResetRecoveryJobReady == PasswordResetRecoveryJobDeadLetter {
		t.Fatal("password-reset durable state constants overlap")
	}
}

func passwordResetStoreTestCandidate(t *testing.T) PasswordResetCandidate {
	t.Helper()
	now := time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC)
	credential, err := NewTemporaryCredential(
		newTemporaryCredentialTestHasher(t),
		TemporaryCredentialInput{
			ID:               "22222222-2222-2222-2222-222222222222",
			UserID:           "user-one",
			Purpose:          TemporaryCredentialPasswordReset,
			Generation:       2,
			Secret:           "replacement-temporary-secret",
			DeliveredToEmail: "recipient@example.test",
			IssuedAt:         now,
		},
	)
	if err != nil {
		t.Fatalf("NewTemporaryCredential() error = %v", err)
	}
	operationID := "11111111-1111-1111-1111-111111111111"
	return PasswordResetCandidate{
		Operation: PasswordResetOperation{
			ID:                    operationID,
			UserID:                "user-one",
			TemporaryCredentialID: credential.ID,
			RequestKind:           PasswordResetInitialReset,
			SettlementState:       PasswordResetSettlementPending,
			DeliveryState:         PasswordResetDeliveryNotStarted,
			State:                 PasswordResetOperationPrepared,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
		Credential: credential,
		RecoveryJob: PasswordResetRecoveryJob{
			OperationID:   operationID,
			Status:        PasswordResetRecoveryJobReady,
			AttemptCount:  0,
			NextAttemptAt: now.Add(2 * time.Minute),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		IdempotencyRequest: idempotency.BeginRequest{
			Scope:   []byte("actor-one\x00password-reset\x00user-one"),
			Key:     "password-reset-key",
			Request: []byte(`{"target":"user-one"}`),
			DigestKeys: idempotency.DigestKeyring{
				ActiveVersion: "v1",
				Keys: map[string][]byte{
					"v1": []byte("password-reset-test-key"),
				},
			},
			LockedUntil: now.Add(time.Minute),
			ExpiresAt:   now.Add(24 * time.Hour),
			Now:         now,
		},
	}
}

func passwordResetClaimRowValues(record idempotency.Record) []any {
	return []any{
		record.ScopeHash,
		record.KeyLocator,
		record.KeyDigest,
		record.RequestDigest,
		record.DigestKeyVersion,
		record.Status,
		record.ExecutionGeneration,
		(*string)(nil),
		[]byte(nil),
		record.LockedUntil,
		(*string)(nil),
		(*time.Time)(nil),
		record.ExpiresAt,
		record.CreatedAt,
		record.UpdatedAt,
	}
}

type passwordResetStoreStep struct {
	query    bool
	contains string
	values   []any
	tag      pgconn.CommandTag
	err      error
}

func passwordResetQueryStep(fragment string, values ...any) passwordResetStoreStep {
	return passwordResetStoreStep{
		query: true, contains: fragment, values: values,
	}
}

func passwordResetQueryErrorStep(
	fragment string,
	err error,
) passwordResetStoreStep {
	return passwordResetStoreStep{
		query: true, contains: fragment, err: err,
	}
}

func passwordResetExecStep(
	fragment string,
	commandTag string,
) passwordResetStoreStep {
	return passwordResetStoreStep{
		contains: fragment,
		tag:      pgconn.NewCommandTag(commandTag),
	}
}

type passwordResetStoreTransactionStub struct {
	steps         []passwordResetStoreStep
	trace         []string
	execArguments [][]any
	applied       map[string]bool
	commitErr     error
}

func (tx *passwordResetStoreTransactionStub) Exec(
	_ context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	step, err := tx.nextStep(false, statement)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	tx.execArguments = append(tx.execArguments, append([]any(nil), arguments...))
	if step.err == nil && step.tag.RowsAffected() == 1 {
		if tx.applied == nil {
			tx.applied = make(map[string]bool)
		}
		if kind := passwordResetWriteKind(statement); kind != "" {
			tx.applied[kind] = true
		}
	}
	return step.tag, step.err
}

func (tx *passwordResetStoreTransactionStub) QueryRow(
	_ context.Context,
	statement string,
	_ ...any,
) pgx.Row {
	step, err := tx.nextStep(true, statement)
	if err != nil {
		return passwordResetStoreRow{err: err}
	}
	return passwordResetStoreRow{values: step.values, err: step.err}
}

func (tx *passwordResetStoreTransactionStub) nextStep(
	query bool,
	statement string,
) (passwordResetStoreStep, error) {
	tx.trace = append(tx.trace, statement)
	if len(tx.steps) == 0 {
		return passwordResetStoreStep{}, errors.New("unexpected transaction call")
	}
	step := tx.steps[0]
	tx.steps = tx.steps[1:]
	if step.query != query || !strings.Contains(statement, step.contains) {
		return passwordResetStoreStep{}, errors.New("unexpected transaction statement")
	}
	return step, nil
}

func (tx *passwordResetStoreTransactionStub) assertExhausted(t *testing.T) {
	t.Helper()
	if len(tx.steps) != 0 {
		t.Fatalf("unconsumed transaction steps = %d", len(tx.steps))
	}
}

func (tx *passwordResetStoreTransactionStub) Commit() error {
	if tx.commitErr != nil {
		return tx.commitErr
	}
	if len(tx.applied) == 0 {
		return nil
	}
	for _, kind := range []string{"counter", "operation", "credential", "job", "claim"} {
		if !tx.applied[kind] {
			return errors.New("deferred password reset authority constraint")
		}
	}
	return nil
}

func passwordResetWriteKind(statement string) string {
	switch {
	case strings.Contains(statement, "password_reset_execution_generation"):
		return "counter"
	case strings.Contains(statement, "INSERT INTO password_reset_operations"):
		return "operation"
	case strings.Contains(statement, "INSERT INTO temporary_credentials"):
		return "credential"
	case strings.Contains(statement, "INSERT INTO password_reset_recovery_jobs"):
		return "job"
	case strings.Contains(statement, "INSERT INTO idempotency_records"):
		return "claim"
	default:
		return ""
	}
}

type passwordResetStoreRow struct {
	values []any
	err    error
}

func (row passwordResetStoreRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(destinations) != len(row.values) {
		return errors.New("unexpected password reset row width")
	}
	for index, destination := range destinations {
		target := reflect.ValueOf(destination)
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("invalid password reset scan destination")
		}
		value := reflect.ValueOf(row.values[index])
		if !value.IsValid() {
			target.Elem().Set(reflect.Zero(target.Elem().Type()))
			continue
		}
		if !value.Type().AssignableTo(target.Elem().Type()) {
			return errors.New("password reset scan type mismatch")
		}
		target.Elem().Set(value)
	}
	return nil
}

func passwordResetRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve password reset test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
}

func passwordResetMigrationSQL(t *testing.T) string {
	t.Helper()
	root := passwordResetRepositoryRoot(t)
	contents, err := os.ReadFile(
		filepath.Join(root, "migrations", "0008_password_reset_durable.sql"),
	)
	if err != nil {
		t.Fatalf("read password reset durable migration: %v", err)
	}
	return string(contents)
}

type passwordResetLegacyClaim struct {
	status     string
	resultCode string
	snapshot   string
}

type passwordResetMigrationPreflightFixture struct {
	name                    string
	records                 []passwordResetLegacyClaim
	legacyCurrentCredential bool
	blocked                 bool
}

func passwordResetMigrationPreflightClassifier(
	t *testing.T,
	preflight string,
) func(passwordResetMigrationPreflightFixture) bool {
	t.Helper()
	adminLegacy := passwordResetPreflightContainsAll(preflight, []string{
		"result_code = 'created'",
		"NOT response_snapshot ? 'schema_version'",
		"response_snapshot ->> 'outcome' = 'created'",
		"jsonb_typeof(response_snapshot -> 'resource_id') = 'string'",
		"btrim(response_snapshot ->> 'resource_id') <> ''",
		"NOT response_snapshot ? 'error_code'",
		"response_snapshot - ARRAY[",
		"'outcome'",
		"'resource_id'",
	})
	adminV1 := passwordResetPreflightContainsAll(preflight, []string{
		"identity-administrator-create-response/v1",
		"response_snapshot -> 'http_status' = '201'::jsonb",
		"response_snapshot ->> 'result_code' = 'created'",
		"response_snapshot -> 'body' ->> 'role' = 'admin'",
		"response_snapshot -> 'body' ->> 'status' = 'active'",
		"-> 'must_change_password' = 'true'::jsonb",
		"-> 'must_acknowledge_privacy' = 'true'::jsonb",
		"->> 'delivery_status' = 'accepted'",
		"'temporary_credential_expires_at'",
		"AND (response_snapshot -> 'body') - ARRAY[",
	})
	failClosed := passwordResetPreflightContainsAll(preflight, []string{
		"status IN ('processing', 'uncertain')",
		"status = 'succeeded'",
		") IS NOT TRUE",
		"unclassified or in-flight generic idempotency",
	})
	credentialFailClosed := passwordResetPreflightContainsAll(preflight, []string{
		"FROM temporary_credentials",
		"purpose = 'password_reset'",
		"state IN ('pending_delivery', 'active', 'expired')",
		"cannot explain a legacy current credential",
	})
	return func(fixture passwordResetMigrationPreflightFixture) bool {
		if fixture.legacyCurrentCredential {
			return credentialFailClosed
		}
		for _, record := range fixture.records {
			if record.status == "processing" || record.status == "uncertain" {
				return true
			}
			if record.status == "succeeded" &&
				!passwordResetLegacyClaimIsAdministrator(record, adminLegacy, adminV1) {
				return true
			}
		}
		return !failClosed || !credentialFailClosed || !adminLegacy || !adminV1
	}
}

func passwordResetPreflightContainsAll(contents string, fragments []string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(contents, fragment) {
			return false
		}
	}
	return true
}

func passwordResetLegacyClaimIsAdministrator(
	record passwordResetLegacyClaim,
	adminLegacy bool,
	adminV1 bool,
) bool {
	if record.resultCode != "created" {
		return false
	}
	var snapshot map[string]any
	if json.Unmarshal([]byte(record.snapshot), &snapshot) != nil {
		return false
	}
	if _, versioned := snapshot["schema_version"]; !versioned {
		return adminLegacy && reflect.DeepEqual(
			snapshot,
			map[string]any{
				"outcome": "created", "resource_id": "administrator-one",
			},
		)
	}
	return adminV1 && passwordResetAdministratorV1SnapshotExact(snapshot)
}

func passwordResetAdministratorV1SnapshotExact(snapshot map[string]any) bool {
	body, ok := snapshot["body"].(map[string]any)
	if !ok || len(snapshot) != 4 || len(body) != 8 {
		return false
	}
	return snapshot["schema_version"] == "identity-administrator-create-response/v1" &&
		snapshot["http_status"] == float64(201) &&
		snapshot["result_code"] == "created" &&
		body["id"] == "administrator-one" &&
		body["username"] == "ops" &&
		body["role"] == "admin" &&
		body["status"] == "active" &&
		body["must_change_password"] == true &&
		body["must_acknowledge_privacy"] == true &&
		body["temporary_credential_expires_at"] == "2026-10-24T08:00:00Z" &&
		body["delivery_status"] == "accepted"
}

func passwordResetResendLineageEvidenceFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	now := time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC)
	attemptID := "55555555-5555-5555-5555-555555555555"
	evidence := passwordResetResendLineageEvidence{
		userID:                    passwordResetTestPointer(userID),
		credentialMode:            CredentialModeTemporary,
		authGeneration:            target,
		predecessorID:             passwordResetTestPointer(predecessorID),
		predecessorUserID:         passwordResetTestPointer(userID),
		predecessorCredentialID:   passwordResetTestPointer(replacedID),
		predecessorState:          passwordResetTestPointer(PasswordResetOperationSucceeded),
		predecessorSettlement:     passwordResetTestPointer(PasswordResetSettlementTemporaryActive),
		predecessorDelivery:       passwordResetTestPointer(PasswordResetDeliveryAccepted),
		targetAuth:                passwordResetTestPointer(target),
		settledAuth:               passwordResetTestPointer(target),
		settledAt:                 passwordResetTestPointer(now),
		terminalAt:                passwordResetTestPointer(now),
		deliveryAttemptID:         passwordResetTestPointer(attemptID),
		deliveryAttemptedAt:       passwordResetTestPointer(now),
		deliveryResultRecordedAt:  passwordResetTestPointer(now),
		replacedID:                passwordResetTestPointer(replacedID),
		replacedUserID:            passwordResetTestPointer(userID),
		replacedPurpose:           passwordResetTestPointer(TemporaryCredentialPasswordReset),
		replacedAuth:              passwordResetTestPointer(target),
		replacedSourceOperationID: passwordResetTestPointer(predecessorID),
		replacedState:             passwordResetTestPointer(TemporaryCredentialActive),
	}
	return passwordResetResendWithCurrentTemporary(
		evidence, predecessorID, replacedID, target-1, target,
		PasswordResetInitialReset, nil, nil, true, target-1,
	)
}

func passwordResetResendLineageRowValues(
	evidence passwordResetResendLineageEvidence,
) []any {
	return []any{
		evidence.userID, evidence.credentialMode, evidence.authGeneration,
		evidence.permanentDigestPresent,
		evidence.permanentGeneration, evidence.permanentSourceOperationID,
		evidence.permanentSourceCredentialID,
		evidence.permanentOperationID, evidence.permanentOperationUserID,
		evidence.permanentOperationCredentialID,
		evidence.permanentOperationState, evidence.permanentOperationSettlement,
		evidence.permanentOperationDelivery,
		evidence.permanentOperationBaseAuth,
		evidence.permanentOperationTargetAuth,
		evidence.permanentOperationSettledAuth,
		evidence.permanentOperationSuccessorAuth,
		evidence.permanentOperationSettledAt,
		evidence.permanentOperationTerminalAt,
		evidence.permanentOperationDeliveryAttemptID,
		evidence.permanentOperationDeliveryAttemptedAt,
		evidence.permanentOperationDeliveryResultRecordedAt,
		evidence.permanentCredentialID, evidence.permanentCredentialUserID,
		evidence.permanentCredentialPurpose,
		evidence.permanentCredentialGeneration,
		evidence.permanentCredentialAuth,
		evidence.permanentCredentialSourceOperationID,
		evidence.permanentCredentialState,
		evidence.predecessorID, evidence.predecessorUserID,
		evidence.predecessorCredentialID, evidence.predecessorState,
		evidence.predecessorSettlement, evidence.predecessorDelivery,
		evidence.targetAuth, evidence.settledAuth, evidence.successorAuth,
		evidence.settledAt, evidence.terminalAt, evidence.deliveryAttemptID,
		evidence.deliveryAttemptedAt, evidence.deliveryResultRecordedAt,
		evidence.replacedID, evidence.replacedUserID, evidence.replacedPurpose,
		evidence.replacedAuth, evidence.replacedSourceOperationID,
		evidence.replacedState,
		evidence.currentOperationID, evidence.currentOperationUserID,
		evidence.currentOperationCredentialID, evidence.currentRequestKind,
		evidence.currentPredecessorID, evidence.currentReplacedID,
		evidence.currentState, evidence.currentSettlement, evidence.currentDelivery,
		evidence.currentBaseAuth, evidence.currentTargetAuth,
		evidence.currentSettledAuth, evidence.currentSuccessorAuth,
		evidence.currentSettledAt, evidence.currentTerminalAt,
		evidence.currentDeliveryAttemptID, evidence.currentDeliveryAttemptedAt,
		evidence.currentDeliveryResultRecordedAt,
		evidence.currentCredentialID, evidence.currentCredentialUserID,
		evidence.currentCredentialPurpose,
		evidence.currentCredentialGeneration,
		evidence.currentCredentialGenerationOrdinal,
		evidence.currentCredentialAuth,
		evidence.currentCredentialSourceOperationID,
		evidence.currentCredentialState,
		evidence.currentLineageValid,
		evidence.currentLineageContainsPredecessor,
		evidence.currentLineageRootBaseAuth,
	}
}

func passwordResetResendConsumedWithoutSuccessorFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	return passwordResetResendConsumedWithoutSuccessor(
		passwordResetResendLineageEvidenceFixture(
			userID, predecessorID, replacedID, target,
		),
	)
}

func passwordResetResendConsumedWithoutSuccessor(
	evidence passwordResetResendLineageEvidence,
) passwordResetResendLineageEvidence {
	evidence.replacedState = passwordResetTestPointer(TemporaryCredentialConsumed)
	evidence.credentialMode = CredentialModePermanent
	evidence.authGeneration = *evidence.targetAuth + 1
	return evidence
}

func passwordResetResendPermanentSuccessorFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLineageEvidenceFixture(
		userID, predecessorID, replacedID, target,
	)
	successor := target + 1
	evidence.predecessorSettlement = passwordResetTestPointer(
		PasswordResetSettlementPermanentSuccessor,
	)
	evidence.successorAuth = passwordResetTestPointer(successor)
	evidence.replacedState = passwordResetTestPointer(TemporaryCredentialConsumed)
	return evidence
}

func passwordResetResendCurrentPermanentFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentSuccessorFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.credentialMode = CredentialModePermanent
	evidence.authGeneration = target + 1
	return passwordResetResendWithoutCurrentTemporary(evidence)
}

func passwordResetResendPermanentRootFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendCurrentPermanentFixture(
		userID, predecessorID, replacedID, target,
	)
	generation := target + 1
	evidence.permanentGeneration = passwordResetTestPointer(generation)
	evidence.permanentSourceOperationID = passwordResetTestPointer(predecessorID)
	evidence.permanentSourceCredentialID = passwordResetTestPointer(replacedID)
	return passwordResetResendWithPermanentRoot(
		evidence, predecessorID, replacedID, target-1, target, generation,
	)
}

func passwordResetResendPermanentWrongRootOperationFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.permanentSourceOperationID = passwordResetTestPointer(
		"88888888-8888-8888-8888-888888888888",
	)
	return evidence
}

func passwordResetResendPermanentWrongRootCredentialFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.permanentSourceCredentialID = passwordResetTestPointer(
		"99999999-9999-9999-9999-999999999999",
	)
	return evidence
}

func passwordResetResendPermanentWrongGenerationFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.authGeneration = target + 2
	return evidence
}

func passwordResetResendLaterPasswordChangeFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	generation := target + 2
	evidence.authGeneration = generation
	evidence.permanentGeneration = passwordResetTestPointer(generation)
	return evidence
}

func passwordResetResendPermanentWrongModeFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.credentialMode = CredentialModeTemporary
	return evidence
}

func passwordResetResendSupersededFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLineageEvidenceFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.replacedState = passwordResetTestPointer(TemporaryCredentialSuperseded)
	evidence.authGeneration = target + 1
	return evidence
}

func passwordResetResendLegalSuccessorFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendSupersededFixture(
		userID, predecessorID, replacedID, target,
	)
	currentOperationID := "66666666-6666-6666-6666-666666666666"
	currentCredentialID := "77777777-7777-7777-7777-777777777777"
	return passwordResetResendWithCurrentTemporary(
		evidence, currentOperationID, currentCredentialID, target, target+1,
		PasswordResetResend, &predecessorID, &replacedID, true, target-1,
	)
}

func passwordResetResendSuccessorWrongStateFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLegalSuccessorFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.currentState = passwordResetTestPointer(
		PasswordResetOperationRecoveryRequired,
	)
	return evidence
}

func passwordResetResendSuccessorWrongSettlementFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLegalSuccessorFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.currentSettlement = passwordResetTestPointer(
		PasswordResetSettlementPermanentSuccessor,
	)
	return evidence
}

func passwordResetResendSuccessorCrossUserFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLegalSuccessorFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.currentOperationUserID = passwordResetTestPointer("user-two")
	return evidence
}

func passwordResetResendSuccessorCredentialMismatchFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLegalSuccessorFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.currentOperationCredentialID = passwordResetTestPointer(
		"99999999-9999-9999-9999-999999999999",
	)
	return evidence
}

func passwordResetResendLaterResetFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.credentialMode = CredentialModeTemporary
	evidence.authGeneration = target + 2
	currentOperationID := "66666666-6666-6666-6666-666666666666"
	currentCredentialID := "77777777-7777-7777-7777-777777777777"
	return passwordResetResendWithCurrentTemporary(
		evidence, currentOperationID, currentCredentialID, target+1, target+2,
		PasswordResetInitialReset, nil, nil, false, target+1,
	)
}

func passwordResetResendLaterPermanentRootFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.authGeneration = target + 3
	evidence.permanentGeneration = passwordResetTestPointer(target + 3)
	evidence.permanentSourceOperationID = passwordResetTestPointer(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)
	evidence.permanentSourceCredentialID = passwordResetTestPointer(
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	)
	return passwordResetResendWithPermanentRoot(
		evidence,
		*evidence.permanentSourceOperationID,
		*evidence.permanentSourceCredentialID,
		target+1,
		target+2,
		target+3,
	)
}

func passwordResetResendLaterPermanentRootTemporaryFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLaterPermanentRootFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.credentialMode = CredentialModeTemporary
	evidence.authGeneration = target + 4
	return passwordResetResendWithCurrentTemporary(
		evidence,
		"cccccccc-cccc-cccc-cccc-cccccccccccc",
		"dddddddd-dddd-dddd-dddd-dddddddddddd",
		target+3,
		target+4,
		PasswordResetInitialReset,
		nil,
		nil,
		false,
		target+3,
	)
}

func passwordResetResendWithCurrentTemporary(
	evidence passwordResetResendLineageEvidence,
	operationID string,
	credentialID string,
	base int64,
	target int64,
	requestKind PasswordResetRequestKind,
	predecessorID *string,
	replacedID *string,
	containsPredecessor bool,
	rootBase int64,
) passwordResetResendLineageEvidence {
	evidence.currentOperationID = passwordResetTestPointer(operationID)
	evidence.currentOperationUserID = evidence.userID
	evidence.currentOperationCredentialID = passwordResetTestPointer(credentialID)
	evidence.currentRequestKind = passwordResetTestPointer(requestKind)
	evidence.currentPredecessorID = predecessorID
	evidence.currentReplacedID = replacedID
	evidence.currentState = passwordResetTestPointer(PasswordResetOperationSucceeded)
	evidence.currentSettlement = passwordResetTestPointer(
		PasswordResetSettlementTemporaryActive,
	)
	evidence.currentDelivery = passwordResetTestPointer(PasswordResetDeliveryAccepted)
	evidence.currentBaseAuth = passwordResetTestPointer(base)
	evidence.currentTargetAuth = passwordResetTestPointer(target)
	evidence.currentSettledAuth = passwordResetTestPointer(target)
	evidence.currentSettledAt = evidence.settledAt
	evidence.currentTerminalAt = evidence.terminalAt
	evidence.currentDeliveryAttemptID = evidence.deliveryAttemptID
	evidence.currentDeliveryAttemptedAt = evidence.deliveryAttemptedAt
	evidence.currentDeliveryResultRecordedAt = evidence.deliveryResultRecordedAt
	evidence.currentCredentialID = passwordResetTestPointer(credentialID)
	evidence.currentCredentialUserID = evidence.userID
	evidence.currentCredentialPurpose = passwordResetTestPointer(
		TemporaryCredentialPasswordReset,
	)
	evidence.currentCredentialGeneration = passwordResetTestPointer(target)
	evidence.currentCredentialGenerationOrdinal = passwordResetTestPointer(target)
	evidence.currentCredentialAuth = passwordResetTestPointer(target)
	evidence.currentCredentialSourceOperationID = passwordResetTestPointer(operationID)
	evidence.currentCredentialState = passwordResetTestPointer(
		TemporaryCredentialActive,
	)
	evidence.currentLineageValid = true
	evidence.currentLineageContainsPredecessor = containsPredecessor
	evidence.currentLineageRootBaseAuth = passwordResetTestPointer(rootBase)
	return evidence
}

func passwordResetResendWithoutCurrentTemporary(
	evidence passwordResetResendLineageEvidence,
) passwordResetResendLineageEvidence {
	evidence.currentOperationID = nil
	evidence.currentOperationUserID = nil
	evidence.currentOperationCredentialID = nil
	evidence.currentRequestKind = nil
	evidence.currentPredecessorID = nil
	evidence.currentReplacedID = nil
	evidence.currentState = nil
	evidence.currentSettlement = nil
	evidence.currentDelivery = nil
	evidence.currentBaseAuth = nil
	evidence.currentTargetAuth = nil
	evidence.currentSettledAuth = nil
	evidence.currentSuccessorAuth = nil
	evidence.currentSettledAt = nil
	evidence.currentTerminalAt = nil
	evidence.currentDeliveryAttemptID = nil
	evidence.currentDeliveryAttemptedAt = nil
	evidence.currentDeliveryResultRecordedAt = nil
	evidence.currentCredentialID = nil
	evidence.currentCredentialUserID = nil
	evidence.currentCredentialPurpose = nil
	evidence.currentCredentialGeneration = nil
	evidence.currentCredentialGenerationOrdinal = nil
	evidence.currentCredentialAuth = nil
	evidence.currentCredentialSourceOperationID = nil
	evidence.currentCredentialState = nil
	evidence.currentLineageValid = false
	evidence.currentLineageContainsPredecessor = false
	evidence.currentLineageRootBaseAuth = nil
	return evidence
}

func passwordResetResendWithPermanentRoot(
	evidence passwordResetResendLineageEvidence,
	operationID string,
	credentialID string,
	base int64,
	target int64,
	successor int64,
) passwordResetResendLineageEvidence {
	now := *evidence.settledAt
	evidence.permanentDigestPresent = true
	evidence.permanentOperationID = passwordResetTestPointer(operationID)
	evidence.permanentOperationUserID = evidence.userID
	evidence.permanentOperationCredentialID = passwordResetTestPointer(credentialID)
	evidence.permanentOperationState = passwordResetTestPointer(
		PasswordResetOperationSucceeded,
	)
	evidence.permanentOperationSettlement = passwordResetTestPointer(
		PasswordResetSettlementPermanentSuccessor,
	)
	evidence.permanentOperationDelivery = passwordResetTestPointer(
		PasswordResetDeliveryAccepted,
	)
	evidence.permanentOperationBaseAuth = passwordResetTestPointer(base)
	evidence.permanentOperationTargetAuth = passwordResetTestPointer(target)
	evidence.permanentOperationSettledAuth = passwordResetTestPointer(target)
	evidence.permanentOperationSuccessorAuth = passwordResetTestPointer(successor)
	evidence.permanentOperationSettledAt = passwordResetTestPointer(now)
	evidence.permanentOperationTerminalAt = passwordResetTestPointer(now)
	evidence.permanentOperationDeliveryAttemptID = evidence.deliveryAttemptID
	evidence.permanentOperationDeliveryAttemptedAt = passwordResetTestPointer(now)
	evidence.permanentOperationDeliveryResultRecordedAt = passwordResetTestPointer(now)
	evidence.permanentCredentialID = passwordResetTestPointer(credentialID)
	evidence.permanentCredentialUserID = evidence.userID
	evidence.permanentCredentialPurpose = passwordResetTestPointer(
		TemporaryCredentialPasswordReset,
	)
	evidence.permanentCredentialGeneration = passwordResetTestPointer(target)
	evidence.permanentCredentialAuth = passwordResetTestPointer(target)
	evidence.permanentCredentialSourceOperationID = passwordResetTestPointer(operationID)
	evidence.permanentCredentialState = passwordResetTestPointer(
		TemporaryCredentialConsumed,
	)
	return evidence
}

func passwordResetResendRecoveryInvalidatedFixture(
	userID string,
	predecessorID string,
	replacedID string,
	target int64,
) passwordResetResendLineageEvidence {
	evidence := passwordResetResendLineageEvidenceFixture(
		userID, predecessorID, replacedID, target,
	)
	evidence.replacedState = passwordResetTestPointer(
		TemporaryCredentialRecoveryInvalidated,
	)
	evidence.credentialMode = CredentialModeRecoveryLocked
	return evidence
}

func passwordResetResendWithCredentialState(
	evidence passwordResetResendLineageEvidence,
	state TemporaryCredentialState,
) passwordResetResendLineageEvidence {
	evidence.replacedState = passwordResetTestPointer(state)
	return evidence
}

func passwordResetResendCrossUserFixture(
	evidence passwordResetResendLineageEvidence,
) passwordResetResendLineageEvidence {
	evidence.predecessorUserID = passwordResetTestPointer("user-two")
	evidence.replacedUserID = passwordResetTestPointer("user-two")
	return evidence
}

func passwordResetResendMissingPredecessorFixture(
	evidence passwordResetResendLineageEvidence,
) passwordResetResendLineageEvidence {
	evidence.predecessorID = nil
	evidence.predecessorUserID = nil
	evidence.predecessorCredentialID = nil
	evidence.predecessorState = nil
	evidence.predecessorSettlement = nil
	evidence.predecessorDelivery = nil
	evidence.targetAuth = nil
	return evidence
}

func passwordResetResendMissingCredentialFixture(
	evidence passwordResetResendLineageEvidence,
) passwordResetResendLineageEvidence {
	evidence.replacedID = nil
	evidence.replacedUserID = nil
	evidence.replacedPurpose = nil
	evidence.replacedAuth = nil
	evidence.replacedSourceOperationID = nil
	evidence.replacedState = nil
	return evidence
}

func passwordResetTestPointer[T any](value T) *T {
	return &value
}

func passwordResetAuthorityOperation(
	state PasswordResetOperationState,
	delivery PasswordResetDeliveryState,
	settlement PasswordResetSettlementState,
	target int64,
	settled *int64,
	successor *int64,
	settledAt *time.Time,
	terminalAt *time.Time,
) PasswordResetOperation {
	return PasswordResetOperation{
		State: state, DeliveryState: delivery, SettlementState: settlement,
		TargetAuthGeneration: target, SettledAuthGeneration: settled,
		SuccessorAuthGeneration: successor, SettledAt: settledAt, TerminalAt: terminalAt,
	}
}

func TestPasswordResetRecoveryMetadata_PartialApproval_RejectsBeforeSQL(t *testing.T) {
	for _, field := range []string{"run", "install", "actor", "ref", "digest", "role", "status"} {
		t.Run(field, func(t *testing.T) {
			candidate := passwordResetStoreTestCandidate(t)
			approval, _, _ := s412ApprovalFixture()
			candidate.Operation.Recovery = recoveryRestoreMetadata(approval)
			metadata := candidate.Operation.Recovery
			switch field {
			case "run":
				metadata.RunID = ""
			case "install":
				metadata.InstallationID = ""
			case "actor":
				metadata.ActorID = ""
			case "ref":
				metadata.EvidenceRef = ""
			case "digest":
				metadata.EvidenceDigest = ""
			case "role":
				metadata.Role = "unknown"
			case "status":
				metadata.Status = UserStatusDeleted
			}
			tx := &passwordResetStoreTransactionStub{}
			if _, err := (PasswordResetStore{}).BeginOrClassifyCandidate(context.Background(), tx, candidate); !errors.Is(err, ErrPasswordResetCandidateInvalid) {
				t.Fatalf("partial approved candidate accepted: %v", err)
			}
			tx.assertExhausted(t)
		})
	}
}

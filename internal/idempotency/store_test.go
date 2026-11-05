package idempotency

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"han/internal/migration"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdempotencyBeginClassifiesExistingRecords(t *testing.T) {
	now := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)
	request := testBeginRequest("key-one", `{"target":"user-one"}`, testDigestKeyringV1(), now)
	record := testRecordForRequest(t, request)
	tests := []struct {
		name   string
		status Status
		action BeginAction
	}{
		{name: "processing", status: StatusProcessing, action: BeginInProgress},
		{name: "succeeded", status: StatusSucceeded, action: BeginReplay},
		{name: "failed", status: StatusFailed, action: BeginReplay},
		{name: "uncertain", status: StatusUncertain, action: BeginOperationUncertain},
		{
			name:   "recovery review required",
			status: StatusRecoveryReviewRequired,
			action: BeginOperationUncertain,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := record
			candidate.Status = test.status
			if test.status == StatusSucceeded || test.status == StatusFailed {
				candidate.ResultCode = "stable-result"
				candidate.ResponseSnapshot = &ResponseSnapshot{Outcome: "stable"}
			}

			got, err := classifyExisting(request, candidate)
			if err != nil {
				t.Fatalf("classifyExisting() error = %v", err)
			}
			if got.Action != test.action {
				t.Fatalf("classifyExisting() action = %q, want %q", got.Action, test.action)
			}
		})
	}
}

func TestIdempotencyBeginUsesStoredDigestVersionAcrossKeyRotation(t *testing.T) {
	now := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)
	v1Request := testBeginRequest("same-logical-key", `{"target":"user-one"}`, testDigestKeyringV1(), now)
	record := testRecordForRequest(t, v1Request)
	record.Status = StatusSucceeded
	record.ResultCode = "created"
	record.ResponseSnapshot = &ResponseSnapshot{Outcome: "created", ResourceID: "user-one"}

	v2Request := v1Request
	v2Request.DigestKeys = testDigestKeyringV2WithV1Retained()
	got, err := classifyExisting(v2Request, record)
	if err != nil {
		t.Fatalf("classifyExisting(v1 record, active v2) error = %v", err)
	}
	if got.Action != BeginReplay {
		t.Fatalf("classifyExisting(v1 record, active v2) action = %q, want %q", got.Action, BeginReplay)
	}

	v2Request.DigestKeys = testDigestKeyringV2Only()
	_, err = classifyExisting(v2Request, record)
	if !errors.Is(err, ErrDigestKeyUnavailable) {
		t.Fatalf("classifyExisting(missing v1) error = %v, want %v", err, ErrDigestKeyUnavailable)
	}

	v2Request = v1Request
	v2Request.Request = []byte(`{"target":"user-two"}`)
	v2Request.DigestKeys = testDigestKeyringV2WithV1Retained()
	_, err = classifyExisting(v2Request, record)
	if !errors.Is(err, ErrDigestConflict) {
		t.Fatalf("classifyExisting(different request) error = %v, want %v", err, ErrDigestConflict)
	}
}

func TestIdempotencyResolveExistingIsReadOnlyAndClassifies(t *testing.T) {
	now := time.Date(2026, time.October, 23, 8, 0, 0, 0, time.UTC)
	request := testBeginRequest(
		"administrator-key",
		`{"username":"ops"}`,
		testDigestKeyringV1(),
		now,
	)
	record := testRecordForRequest(t, request)
	record.Status = StatusSucceeded
	record.ResultCode = "created"
	record.ResponseSnapshot = &ResponseSnapshot{
		SchemaVersion: AdministratorCreateResponseSchemaV1,
		HTTPStatus:    201,
		ResultCode:    "created",
		Body: &AdministratorCreateResponseBody{
			ID: "administrator-1", Username: "ops", Role: "admin", Status: "active",
			MustChangePassword: true, MustAcknowledgePrivacy: true,
			TemporaryCredentialExpiresAt: "2026-10-24T08:00:00Z",
			DeliveryStatus:               "accepted",
		},
	}

	tx := &readOnlyTransactionStub{record: &record}
	result, found, err := (Store{}).ResolveExisting(context.Background(), tx, request)
	if err != nil {
		t.Fatalf("ResolveExisting() error = %v", err)
	}
	if !found || result.Action != BeginReplay {
		t.Fatalf("ResolveExisting() = %#v, found=%v", result, found)
	}
	if tx.execCalls != 0 || tx.queryCalls != 1 {
		t.Fatalf("transaction calls: exec=%d query=%d", tx.execCalls, tx.queryCalls)
	}

	different := request
	different.Request = []byte(`{"username":"different"}`)
	if _, _, err := (Store{}).ResolveExisting(
		context.Background(),
		&readOnlyTransactionStub{record: &record},
		different,
	); !errors.Is(err, ErrDigestConflict) {
		t.Fatalf("ResolveExisting(different request) error = %v, want %v", err, ErrDigestConflict)
	}

	result, found, err = (Store{}).ResolveExisting(
		context.Background(),
		&readOnlyTransactionStub{},
		request,
	)
	if err != nil || found || result.Action != "" {
		t.Fatalf("ResolveExisting(missing) = %#v, found=%v, error=%v", result, found, err)
	}
}

func TestIdempotencyPasswordResetClaimBindsSixFieldsAndCallerGeneration(t *testing.T) {
	now := time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC)
	request := testBeginRequest(
		"password-reset-key",
		`{"operation":"password_reset","target":"user-one"}`,
		testDigestKeyringV1(),
		now,
	)
	tx := &passwordResetClaimTransactionStub{}
	record, err := (Store{}).InsertPasswordResetClaim(
		context.Background(),
		tx,
		request,
		9,
		"11111111-1111-1111-1111-111111111111",
		"user-one",
	)
	if err != nil {
		t.Fatalf("InsertPasswordResetClaim() error = %v", err)
	}
	if record.ExecutionGeneration != 9 || record.Status != StatusProcessing {
		t.Fatalf("password-reset record = %#v", record)
	}
	if tx.execCalls != 1 || !strings.Contains(tx.statement, "$6") {
		t.Fatalf("claim insert calls=%d statement=%q", tx.execCalls, tx.statement)
	}
	if len(tx.arguments) != 11 ||
		tx.arguments[5] != int64(9) ||
		tx.arguments[6] != "11111111-1111-1111-1111-111111111111" ||
		tx.arguments[7] != "user-one" {
		t.Fatalf("claim insert arguments = %#v", tx.arguments)
	}
	expected, err := NewPasswordResetClaim(request, 9)
	if err != nil {
		t.Fatalf("NewPasswordResetClaim() error = %v", err)
	}
	if !bytes.Equal(record.ScopeHash, expected.ScopeHash) ||
		!bytes.Equal(record.KeyLocator, expected.KeyLocator) ||
		!bytes.Equal(record.KeyDigest, expected.KeyDigest) ||
		!bytes.Equal(record.RequestDigest, expected.RequestDigest) ||
		record.DigestKeyVersion != expected.DigestKeyVersion {
		t.Fatalf("claim identity = %#v, want %#v", record, expected)
	}
}

func TestIdempotencyPasswordResetClaimRejectsInvalidGeneration(t *testing.T) {
	request := testBeginRequest(
		"password-reset-key",
		`{"operation":"password_reset","target":"user-one"}`,
		testDigestKeyringV1(),
		time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC),
	)
	tx := &passwordResetClaimTransactionStub{}
	if _, err := (Store{}).InsertPasswordResetClaim(
		context.Background(),
		tx,
		request,
		0,
		"11111111-1111-1111-1111-111111111111",
		"user-one",
	); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("InsertPasswordResetClaim() error = %v", err)
	}
	if tx.execCalls != 0 {
		t.Fatalf("invalid claim exec calls = %d", tx.execCalls)
	}
}

func TestIdempotencyPasswordResetClaimUsesStoredDigestVersion(t *testing.T) {
	now := time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC)
	v1Request := testBeginRequest(
		"password-reset-key",
		`{"operation":"password_reset","target":"user-one"}`,
		testDigestKeyringV1(),
		now,
	)
	record := testRecordForRequest(t, v1Request)
	record.Status = StatusSucceeded
	record.ResultCode = "password_reset"
	record.ResponseSnapshot = &ResponseSnapshot{Outcome: "accepted"}

	v2Request := v1Request
	v2Request.DigestKeys = testDigestKeyringV2WithV1Retained()
	got, found, err := (Store{}).ResolvePasswordResetClaim(
		context.Background(),
		&readOnlyTransactionStub{record: &record},
		v2Request,
	)
	if err != nil || !found || got.DigestKeyVersion != "v1" {
		t.Fatalf(
			"ResolvePasswordResetClaim() = %#v, found=%v, error=%v",
			got,
			found,
			err,
		)
	}

	different := v2Request
	different.Request = []byte(`{"operation":"password_reset","target":"user-two"}`)
	if _, _, err := (Store{}).ResolvePasswordResetClaim(
		context.Background(),
		&readOnlyTransactionStub{record: &record},
		different,
	); !errors.Is(err, ErrDigestConflict) {
		t.Fatalf("ResolvePasswordResetClaim(different) error = %v", err)
	}
}

type passwordResetClaimTransactionStub struct {
	execCalls int
	statement string
	arguments []any
}

func (t *passwordResetClaimTransactionStub) Exec(
	_ context.Context,
	statement string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	t.execCalls++
	t.statement = statement
	t.arguments = append([]any(nil), arguments...)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (*passwordResetClaimTransactionStub) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	return readOnlyRecordRow{}
}

func TestIdempotencySnapshotAllowlistRequiresExactDeclaredFields(t *testing.T) {
	validSnapshots := []json.RawMessage{
		json.RawMessage(`{"outcome":"accepted"}`),
		json.RawMessage(`{"outcome":"accepted","resource_id":"opaque","error_code":"none"}`),
	}
	for _, encoded := range validSnapshots {
		snapshot, err := decodeResponseSnapshot(encoded)
		if err != nil {
			t.Fatalf("decodeResponseSnapshot(%s) error = %v", encoded, err)
		}
		if snapshot.Outcome != "accepted" {
			t.Fatalf("decodeResponseSnapshot(%s) = %#v", encoded, snapshot)
		}
	}

	invalidSnapshots := []struct {
		name    string
		encoded json.RawMessage
	}{
		{name: "capitalized outcome", encoded: json.RawMessage(`{"Outcome":"accepted"}`)},
		{name: "uppercase outcome", encoded: json.RawMessage(`{"OUTCOME":"accepted"}`)},
		{
			name:    "capitalized optional field",
			encoded: json.RawMessage(`{"outcome":"accepted","Resource_ID":"opaque"}`),
		},
		{
			name:    "hyphenated field",
			encoded: json.RawMessage(`{"outcome":"accepted","resource-id":"opaque"}`),
		},
		{
			name:    "camel case field",
			encoded: json.RawMessage(`{"outcome":"accepted","errorCode":"none"}`),
		},
		{
			name:    "unknown authentication field",
			encoded: json.RawMessage(`{"outcome":"accepted","authorization":"Bearer secret"}`),
		},
		{
			name:    "duplicate field",
			encoded: json.RawMessage(`{"outcome":"accepted","outcome":"replaced"}`),
		},
		{
			name:    "nested object",
			encoded: json.RawMessage(`{"outcome":{"value":"accepted"}}`),
		},
		{name: "nested array", encoded: json.RawMessage(`{"outcome":["accepted"]}`)},
		{name: "null value", encoded: json.RawMessage(`{"outcome":null}`)},
		{name: "empty outcome", encoded: json.RawMessage(`{"outcome":""}`)},
		{name: "blank outcome", encoded: json.RawMessage(`{"outcome":" \t "}`)},
		{name: "top level array", encoded: json.RawMessage(`[{"outcome":"accepted"}]`)},
		{
			name:    "multiple values",
			encoded: json.RawMessage(`{"outcome":"accepted"}{"outcome":"accepted"}`),
		},
		{name: "trailing data", encoded: json.RawMessage(`{"outcome":"accepted"} trailing`)},
	}
	for _, test := range invalidSnapshots {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeResponseSnapshot(test.encoded); !errors.Is(err, ErrUnsafeSnapshot) {
				t.Fatalf(
					"decodeResponseSnapshot(%s) error = %v, want %v",
					test.encoded,
					err,
					ErrUnsafeSnapshot,
				)
			}
		})
	}
}

type readOnlyTransactionStub struct {
	record     *Record
	execCalls  int
	queryCalls int
}

func (t *readOnlyTransactionStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	t.execCalls++
	return pgconn.CommandTag{}, errors.New("unexpected Exec")
}

func (t *readOnlyTransactionStub) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	t.queryCalls++
	return readOnlyRecordRow{record: t.record}
}

type readOnlyRecordRow struct {
	record *Record
}

func (r readOnlyRecordRow) Scan(destinations ...any) error {
	if r.record == nil {
		return pgx.ErrNoRows
	}
	encoded, err := encodeResponseSnapshot(*r.record.ResponseSnapshot)
	if err != nil {
		return err
	}
	resultCode := r.record.ResultCode
	values := []any{
		r.record.ScopeHash,
		r.record.KeyLocator,
		r.record.KeyDigest,
		r.record.RequestDigest,
		r.record.DigestKeyVersion,
		r.record.Status,
		r.record.ExecutionGeneration,
		&resultCode,
		encoded,
		r.record.LockedUntil,
		r.record.ReviewedBy,
		r.record.ReviewedAt,
		r.record.ExpiresAt,
		r.record.CreatedAt,
		r.record.UpdatedAt,
	}
	for index, destination := range destinations {
		switch target := destination.(type) {
		case *[]byte:
			*target = append((*target)[:0], values[index].([]byte)...)
		case *string:
			*target = values[index].(string)
		case *Status:
			*target = values[index].(Status)
		case *int64:
			*target = values[index].(int64)
		case **string:
			*target = values[index].(*string)
		case **time.Time:
			*target = values[index].(*time.Time)
		case *time.Time:
			*target = values[index].(time.Time)
		default:
			return fmt.Errorf("unexpected scan destination %T", destination)
		}
	}
	return nil
}

func TestIdempotencySnapshotEncodingContainsOnlyDeclaredFields(t *testing.T) {
	encoded, err := encodeResponseSnapshot(ResponseSnapshot{
		Outcome:    "created",
		ResourceID: "user-one",
	})
	if err != nil {
		t.Fatalf("encodeResponseSnapshot() error = %v", err)
	}
	if string(encoded) != `{"outcome":"created","resource_id":"user-one"}` {
		t.Fatalf("encodeResponseSnapshot() = %s", encoded)
	}
}

func TestIdempotencyAdministratorSnapshotStrictLegacyV1Codec(t *testing.T) {
	fixtures := []struct {
		name     string
		encoded  json.RawMessage
		validate func(*testing.T, ResponseSnapshot)
	}{
		{
			name: "success",
			encoded: json.RawMessage(`{
				"schema_version":"identity-administrator-create-response/v1",
				"http_status":201,
				"result_code":"created",
				"body":{
					"id":"administrator-1",
					"username":"ops",
					"role":"admin",
					"status":"active",
					"must_change_password":true,
					"must_acknowledge_privacy":true,
					"temporary_credential_expires_at":"2026-10-23T10:00:00Z",
					"delivery_status":"accepted"
				}
			}`),
			validate: func(t *testing.T, snapshot ResponseSnapshot) {
				t.Helper()
				if snapshot.SchemaVersion != AdministratorCreateResponseSchemaV1 ||
					snapshot.HTTPStatus != 201 || snapshot.ResultCode != "created" ||
					snapshot.Body == nil || snapshot.Body.ID != "administrator-1" ||
					snapshot.Body.Username != "ops" {
					t.Fatalf("decoded success snapshot = %#v", snapshot)
				}
			},
		},
		{
			name: "identifier conflict",
			encoded: json.RawMessage(`{
				"schema_version":"identity-administrator-create-response/v1",
				"http_status":409,
				"result_code":"identity.identifier_conflict",
				"body":{
					"code":"identity.identifier_conflict",
					"message":"身份标识已被使用",
					"field_errors":[]
				}
			}`),
			validate: func(t *testing.T, snapshot ResponseSnapshot) {
				t.Helper()
				if snapshot.Body == nil || snapshot.Body.Code != "identity.identifier_conflict" ||
					snapshot.Body.Message != "身份标识已被使用" ||
					snapshot.Body.FieldErrors == nil || len(snapshot.Body.FieldErrors) != 0 {
					t.Fatalf("decoded conflict snapshot = %#v", snapshot)
				}
			},
		},
		{
			name: "email delivery failed",
			encoded: json.RawMessage(`{
				"schema_version":"identity-administrator-create-response/v1",
				"http_status":503,
				"result_code":"identity.email_delivery_failed",
				"body":{
					"code":"identity.email_delivery_failed",
					"message":"邮件暂未受理",
					"field_errors":[]
				}
			}`),
			validate: func(t *testing.T, snapshot ResponseSnapshot) {
				t.Helper()
				if snapshot.Body == nil || snapshot.Body.Code != "identity.email_delivery_failed" ||
					snapshot.Body.Message != "邮件暂未受理" ||
					snapshot.Body.FieldErrors == nil || len(snapshot.Body.FieldErrors) != 0 {
					t.Fatalf("decoded delivery failure snapshot = %#v", snapshot)
				}
			},
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			snapshot, err := decodeResponseSnapshot(fixture.encoded)
			if err != nil {
				t.Fatalf("decodeResponseSnapshot() error = %v", err)
			}
			fixture.validate(t, snapshot)
			encoded, err := encodeResponseSnapshot(snapshot)
			if err != nil {
				t.Fatalf("encodeResponseSnapshot() error = %v", err)
			}
			roundTrip, err := decodeResponseSnapshot(encoded)
			if err != nil {
				t.Fatalf("decodeResponseSnapshot(round trip) error = %v", err)
			}
			fixture.validate(t, roundTrip)
		})
	}
}

func TestIdempotencyAdministratorSnapshotRejectsSchemaDrift(t *testing.T) {
	validConflict := `{
		"schema_version":"identity-administrator-create-response/v1",
		"http_status":409,
		"result_code":"identity.identifier_conflict",
		"body":{
			"code":"identity.identifier_conflict",
			"message":"身份标识已被使用",
			"field_errors":[]
		}
	}`
	tests := []struct {
		name    string
		encoded string
	}{
		{name: "unknown version", encoded: strings.Replace(validConflict, AdministratorCreateResponseSchemaV1, "identity-administrator-create-response/v2", 1)},
		{name: "outer extra", encoded: strings.Replace(validConflict, `"body":`, `"trace_id":"trace-secret","body":`, 1)},
		{name: "outer missing", encoded: strings.Replace(validConflict, `"http_status":409,`, "", 1)},
		{name: "mixed legacy", encoded: strings.Replace(validConflict, `"body":`, `"outcome":"failed","body":`, 1)},
		{name: "body extra", encoded: strings.Replace(validConflict, `"field_errors":[]`, `"field_errors":[],"username":"ops"`, 1)},
		{
			name: "field errors missing",
			encoded: `{
				"schema_version":"identity-administrator-create-response/v1",
				"http_status":409,
				"result_code":"identity.identifier_conflict",
				"body":{
					"code":"identity.identifier_conflict",
					"message":"身份标识已被使用"
				}
			}`,
		},
		{name: "field errors null", encoded: strings.Replace(validConflict, `"field_errors":[]`, `"field_errors":null`, 1)},
		{name: "field errors non empty", encoded: strings.Replace(validConflict, `"field_errors":[]`, `"field_errors":[{"field":"username","code":"conflict"}]`, 1)},
		{name: "field errors wrong type", encoded: strings.Replace(validConflict, `"field_errors":[]`, `"field_errors":{}`, 1)},
		{name: "status wrong type", encoded: strings.Replace(validConflict, `"http_status":409`, `"http_status":"409"`, 1)},
		{name: "trace in body", encoded: strings.Replace(validConflict, `"field_errors":[]`, `"field_errors":[],"trace_id":"trace-secret"`, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeResponseSnapshot(json.RawMessage(test.encoded)); !errors.Is(err, ErrUnsafeSnapshot) {
				t.Fatalf("decodeResponseSnapshot() error = %v, want %v", err, ErrUnsafeSnapshot)
			}
		})
	}
}

func TestIdempotencyMigrationDefinesRotationSafePersistentStateMachine(t *testing.T) {
	sql := readIdempotencyMigration(t)
	required := []string{
		"CREATE TABLE idempotency_records",
		"key_locator bytea",
		"PRIMARY KEY (scope_hash, key_locator)",
		"key_digest",
		"request_digest",
		"digest_key_version",
		"execution_generation",
		"response_snapshot jsonb",
		"jsonb_typeof(response_snapshot)",
		"btrim(response_snapshot ->> 'outcome') <> ''",
		"'outcome'",
		"'resource_id'",
		"'error_code'",
		"locked_until",
		"reviewed_by",
		"reviewed_at",
		"expires_at",
		"'processing'",
		"'succeeded'",
		"'failed'",
		"'uncertain'",
		"'recovery_review_required'",
		"enforce_idempotency_record_transition",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}

func TestIdempotencyMigration0007OnlyWidensResponseSnapshotCheck(t *testing.T) {
	content, err := os.ReadFile(
		filepath.Join(repositoryRoot(t), "migrations", "0007_idempotency_response_snapshot_v1.sql"),
	)
	if err != nil {
		t.Fatalf("read migration 0007: %v", err)
	}
	sql := string(content)
	for _, required := range []string{
		"ALTER TABLE idempotency_records",
		"DROP CONSTRAINT",
		"ADD CONSTRAINT",
		"identity-administrator-create-response/v1",
		"schema_version",
		"http_status",
		"result_code",
		"field_errors",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration 0007 missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"ADD COLUMN",
		"ALTER COLUMN",
		"UPDATE idempotency_records",
		"DELETE FROM idempotency_records",
	} {
		if strings.Contains(strings.ToUpper(sql), forbidden) {
			t.Errorf("migration 0007 contains forbidden operation %q", forbidden)
		}
	}
}

func TestIdempotencyStoreRequiresTransactionAndValidInput(t *testing.T) {
	_, err := (Store{}).Begin(context.Background(), nil, BeginRequest{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Begin() error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestIdempotencyPostgresRotationSnapshotAndRecovery(t *testing.T) {
	dsn, exists := os.LookupEnv("HAN_TEST_POSTGRES_DSN")
	if !exists || strings.TrimSpace(dsn) == "" {
		t.Skip("HAN_TEST_POSTGRES_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	adminConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("parse PostgreSQL test configuration")
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal("create PostgreSQL test pool")
	}
	t.Cleanup(adminPool.Close)
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatal("connect PostgreSQL test pool")
	}
	var serverVersionNumber string
	if err := adminPool.QueryRow(ctx, "SHOW server_version_num").Scan(&serverVersionNumber); err != nil {
		t.Fatal("read PostgreSQL server version")
	}
	serverVersion, err := strconv.Atoi(serverVersionNumber)
	if err != nil || serverVersion/10000 != 18 {
		t.Fatalf("PostgreSQL major = %d, want 18", serverVersion/10000)
	}

	schema := "idempotency_" + randomTestHex(t, 8)
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal("create PostgreSQL idempotency test schema")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error("drop PostgreSQL idempotency test schema")
		}
	})

	testConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("parse isolated PostgreSQL test configuration")
	}
	testConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal("create isolated PostgreSQL test pool")
	}
	t.Cleanup(pool.Close)

	runner, err := migration.NewRunner(pool, os.DirFS(repositoryRoot(t)), "migrations")
	if err != nil {
		t.Fatalf("create migration runner: %v", err)
	}
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("run migrations 0001 through 0004: %v", err)
	}
	var migrationCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM schema_migrations WHERE version BETWEEN 1 AND 4",
	).Scan(&migrationCount); err != nil {
		t.Fatal("query schema migration count")
	}
	if migrationCount != 4 {
		t.Fatalf("applied migration count = %d, want 4", migrationCount)
	}

	store := Store{}
	now := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)
	v1Request := testBeginRequest(
		"rotation-key",
		`{"operation":"password_reset","target":"user-one"}`,
		testDigestKeyringV1(),
		now,
	)
	sideEffects := 0
	first := beginAndCommit(t, ctx, pool, store, v1Request)
	if first.Action != BeginExecute {
		t.Fatalf("first Begin() action = %q, want %q", first.Action, BeginExecute)
	}
	sideEffects++
	completeAndCommit(t, ctx, pool, store, first.Record, now.Add(time.Minute))

	v2Request := v1Request
	v2Request.DigestKeys = testDigestKeyringV2WithV1Retained()
	replay := beginAndRollback(t, ctx, pool, store, v2Request)
	if replay.Action != BeginReplay || replay.Record.DigestKeyVersion != "v1" {
		t.Fatalf("v2 replay = %#v", replay)
	}
	if replay.Action == BeginExecute {
		sideEffects++
	}
	if sideEffects != 1 {
		t.Fatalf("side effects = %d, want 1", sideEffects)
	}
	assertIdempotencyRowCount(t, ctx, pool, 1)

	missingOldKey := v2Request
	missingOldKey.DigestKeys = testDigestKeyringV2Only()
	assertBeginError(t, ctx, pool, store, missingOldKey, ErrDigestKeyUnavailable)
	assertIdempotencyRowCount(t, ctx, pool, 1)

	differentRequest := v2Request
	differentRequest.Request = []byte(
		`{"operation":"password_reset","target":"user-two"}`,
	)
	assertBeginError(t, ctx, pool, store, differentRequest, ErrDigestConflict)
	assertIdempotencyRowCount(t, ctx, pool, 1)

	invalidDatabaseSnapshots := []struct {
		name     string
		snapshot string
	}{
		{name: "capitalized field", snapshot: `{"Outcome":"accepted"}`},
		{name: "hyphenated field", snapshot: `{"outcome":"accepted","resource-id":"opaque"}`},
		{name: "unknown field", snapshot: `{"outcome":"accepted","authorization":"Bearer secret"}`},
		{name: "nested object", snapshot: `{"outcome":{"value":"accepted"}}`},
		{name: "array value", snapshot: `{"outcome":["accepted"]}`},
		{name: "null value", snapshot: `{"outcome":null}`},
		{name: "empty outcome", snapshot: `{"outcome":""}`},
		{name: "blank outcome", snapshot: `{"outcome":"   "}`},
		{name: "tab outcome", snapshot: `{"outcome":"\t"}`},
		{name: "top level array", snapshot: `[{"outcome":"accepted"}]`},
	}
	for _, test := range invalidDatabaseSnapshots {
		t.Run("database rejects "+test.name, func(t *testing.T) {
			_, insertErr := pool.Exec(ctx, `
				INSERT INTO idempotency_records (
					scope_hash, key_locator, key_digest, request_digest,
					digest_key_version, status, execution_generation,
					result_code, response_snapshot, expires_at, created_at, updated_at
				)
				VALUES (
					decode(repeat('01', 32), 'hex'),
					decode(repeat('02', 32), 'hex'),
					decode(repeat('03', 32), 'hex'),
					decode(repeat('04', 32), 'hex'),
					'v1', 'succeeded', 1, 'created', $1::jsonb, $2, $3, $3
				)
			`, test.snapshot, now.Add(time.Hour), now)
			if insertErr == nil {
				t.Fatal("database accepted an invalid response snapshot")
			}
		})
	}

	recoveryRequest := testBeginRequest(
		"recovery-key",
		`{"operation":"temporary_credential_resend","target":"user-one"}`,
		testDigestKeyringV2WithV1Retained(),
		now,
	)
	recoveryBegin := beginAndCommit(t, ctx, pool, store, recoveryRequest)
	transition := transitionFromRecord(recoveryBegin.Record, now.Add(time.Minute))
	transitionAndCommit(t, ctx, pool, func(tx pgx.Tx) error {
		return store.MarkUncertain(ctx, tx, transition)
	})
	uncertain := beginAndRollback(t, ctx, pool, store, recoveryRequest)
	if uncertain.Action != BeginOperationUncertain {
		t.Fatalf("uncertain Begin() action = %q", uncertain.Action)
	}

	transition.Now = now.Add(2 * time.Minute)
	transitionAndCommit(t, ctx, pool, func(tx pgx.Tx) error {
		if err := store.RequireRecoveryReview(ctx, tx, transition); err != nil {
			return err
		}
		return store.Recover(ctx, tx, Recovery{
			ScopeHash:           recoveryBegin.Record.ScopeHash,
			KeyLocator:          recoveryBegin.Record.KeyLocator,
			KeyDigest:           recoveryBegin.Record.KeyDigest,
			RequestDigest:       recoveryBegin.Record.RequestDigest,
			DigestKeyVersion:    recoveryBegin.Record.DigestKeyVersion,
			ExecutionGeneration: 1,
			Action:              RecoveryRetry,
			ReviewedBy:          "operator-ref",
			LockedUntil:         now.Add(5 * time.Minute),
			ExpiresAt:           now.Add(2 * time.Hour),
			Now:                 now.Add(3 * time.Minute),
		})
	})
	retried := beginAndRollback(t, ctx, pool, store, recoveryRequest)
	if retried.Action != BeginInProgress || retried.Record.ExecutionGeneration != 2 {
		t.Fatalf("retried Begin() = %#v", retried)
	}

	concurrentRequest := testBeginRequest(
		"concurrent-key",
		`{"operation":"password_reset","target":"user-three"}`,
		testDigestKeyringV2WithV1Retained(),
		now,
	)
	firstTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin first concurrent transaction")
	}
	firstConcurrent, err := store.Begin(ctx, firstTx, concurrentRequest)
	if err != nil || firstConcurrent.Action != BeginExecute {
		_ = firstTx.Rollback(ctx)
		t.Fatalf("first concurrent Begin() = %#v, error = %v", firstConcurrent, err)
	}
	type concurrentResult struct {
		result BeginResult
		err    error
	}
	resultChannel := make(chan concurrentResult, 1)
	go func() {
		secondTx, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			resultChannel <- concurrentResult{err: beginErr}
			return
		}
		result, beginErr := store.Begin(ctx, secondTx, concurrentRequest)
		_ = secondTx.Rollback(ctx)
		resultChannel <- concurrentResult{result: result, err: beginErr}
	}()
	if err := firstTx.Commit(ctx); err != nil {
		t.Fatal("commit first concurrent transaction")
	}
	concurrent := <-resultChannel
	if concurrent.err != nil || concurrent.result.Action != BeginInProgress {
		t.Fatalf("second concurrent Begin() = %#v, error = %v", concurrent.result, concurrent.err)
	}
}

func testDigestKeyringV1() DigestKeyring {
	return DigestKeyring{
		ActiveVersion: "v1",
		Keys: map[string][]byte{
			"v1": []byte("retained-test-key-v1"),
		},
	}
}

func testDigestKeyringV2WithV1Retained() DigestKeyring {
	return DigestKeyring{
		ActiveVersion: "v2",
		Keys: map[string][]byte{
			"v1": []byte("retained-test-key-v1"),
			"v2": []byte("active-test-key-v2"),
		},
	}
}

func testDigestKeyringV2Only() DigestKeyring {
	return DigestKeyring{
		ActiveVersion: "v2",
		Keys: map[string][]byte{
			"v2": []byte("active-test-key-v2"),
		},
	}
}

func testBeginRequest(key, request string, keyring DigestKeyring, now time.Time) BeginRequest {
	return BeginRequest{
		Scope:       []byte("actor-one\x00password-reset\x00user-one"),
		Key:         key,
		Request:     []byte(request),
		DigestKeys:  keyring,
		LockedUntil: now.Add(5 * time.Minute),
		ExpiresAt:   now.Add(time.Hour),
		Now:         now,
	}
}

func testRecordForRequest(t *testing.T, request BeginRequest) Record {
	t.Helper()
	identity, err := newRecordIdentity(request)
	if err != nil {
		t.Fatalf("newRecordIdentity() error = %v", err)
	}
	return Record{
		ScopeHash:           identity.ScopeHash,
		KeyLocator:          identity.KeyLocator,
		KeyDigest:           identity.KeyDigest,
		RequestDigest:       identity.RequestDigest,
		DigestKeyVersion:    identity.DigestKeyVersion,
		Status:              StatusProcessing,
		ExecutionGeneration: 1,
	}
}

func completeAndCommit(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	store Store,
	record Record,
	now time.Time,
) {
	t.Helper()
	transitionAndCommit(t, ctx, pool, func(tx pgx.Tx) error {
		return store.Complete(ctx, tx, Completion{
			ScopeHash:           record.ScopeHash,
			KeyLocator:          record.KeyLocator,
			KeyDigest:           record.KeyDigest,
			RequestDigest:       record.RequestDigest,
			DigestKeyVersion:    record.DigestKeyVersion,
			ExecutionGeneration: record.ExecutionGeneration,
			Status:              StatusSucceeded,
			ResultCode:          "created",
			ResponseSnapshot: ResponseSnapshot{
				Outcome:    "created",
				ResourceID: "user-one",
			},
			Now: now,
		})
	})
}

func transitionFromRecord(record Record, now time.Time) Transition {
	return Transition{
		ScopeHash:           record.ScopeHash,
		KeyLocator:          record.KeyLocator,
		KeyDigest:           record.KeyDigest,
		RequestDigest:       record.RequestDigest,
		DigestKeyVersion:    record.DigestKeyVersion,
		ExecutionGeneration: record.ExecutionGeneration,
		Now:                 now,
	}
}

func beginAndCommit(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	store Store,
	request BeginRequest,
) BeginResult {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin PostgreSQL idempotency transaction")
	}
	result, err := store.Begin(ctx, tx, request)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Begin() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal("commit PostgreSQL idempotency transaction")
	}
	return result
}

func beginAndRollback(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	store Store,
	request BeginRequest,
) BeginResult {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin PostgreSQL idempotency transaction")
	}
	result, err := store.Begin(ctx, tx, request)
	_ = tx.Rollback(ctx)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	return result
}

func assertBeginError(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	store Store,
	request BeginRequest,
	want error,
) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin PostgreSQL idempotency transaction")
	}
	_, beginErr := store.Begin(ctx, tx, request)
	_ = tx.Rollback(ctx)
	if !errors.Is(beginErr, want) {
		t.Fatalf("Begin() error = %v, want %v", beginErr, want)
	}
}

func transitionAndCommit(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	operation func(pgx.Tx) error,
) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin PostgreSQL transition transaction")
	}
	if err := operation(tx); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("idempotency transition error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal("commit PostgreSQL transition transaction")
	}
}

func assertIdempotencyRowCount(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	want int,
) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM idempotency_records").Scan(&count); err != nil {
		t.Fatal("count idempotency rows")
	}
	if count != want {
		t.Fatalf("idempotency row count = %d, want %d", count, want)
	}
}

func readIdempotencyMigration(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(
		filepath.Join(repositoryRoot(t), "migrations", "0004_idempotency.sql"),
	)
	if err != nil {
		t.Fatalf("read idempotency migration: %v", err)
	}
	return string(content)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
}

func randomTestHex(t *testing.T, byteCount int) string {
	t.Helper()
	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		t.Fatal("generate PostgreSQL test schema suffix")
	}
	return hex.EncodeToString(value)
}

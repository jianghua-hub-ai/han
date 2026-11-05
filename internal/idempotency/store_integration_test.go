//go:build integration

package idempotency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/migration"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdempotencyPostgres18AdministratorResponseSnapshotV1(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	dsn := testenv.RequirePostgresDSN(t)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("create PostgreSQL test pool")
	}
	t.Cleanup(admin.Close)
	var versionText string
	if err := admin.QueryRow(ctx, "SHOW server_version_num").Scan(&versionText); err != nil {
		t.Fatal("read PostgreSQL server version")
	}
	version, err := strconv.Atoi(versionText)
	if err != nil || version/10000 != 18 {
		t.Fatalf("PostgreSQL major = %d, want 18", version/10000)
	}

	schema := "idempotency_v1_" + randomTestHex(t, 8)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal("create isolated schema")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error("drop isolated schema")
		}
	})

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("parse isolated PostgreSQL configuration")
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("create isolated PostgreSQL pool")
	}
	t.Cleanup(pool.Close)
	runner, err := migration.NewRunner(pool, idempotencyMigrationFS(t, 6), "migrations")
	if err != nil {
		t.Fatalf("create pre-0007 migration runner: %v", err)
	}
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("run migrations through 0006: %v", err)
	}

	const legacyBeforeUpgrade = `{"outcome":"created","resource_id":"legacy-before-upgrade"}`
	if err := insertSnapshotFixture(ctx, pool, 0, legacyBeforeUpgrade); err != nil {
		t.Fatalf("insert legacy fixture before 0007: %v", err)
	}
	before := readStoredSnapshotIdentity(t, ctx, pool, 0)

	runner, err = migration.NewRunner(pool, idempotencyMigrationFS(t, 7), "migrations")
	if err != nil {
		t.Fatalf("create through-0007 migration runner: %v", err)
	}
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("run migration 0007: %v", err)
	}
	after := readStoredSnapshotIdentity(t, ctx, pool, 0)
	if after != before {
		t.Fatalf("legacy row changed during 0007: before=%#v after=%#v", before, after)
	}

	valid := []string{
		`{"outcome":"failed","error_code":"legacy.failure"}`,
		`{"outcome":"created","resource_id":"legacy-user"}`,
		`{"schema_version":"identity-administrator-create-response/v1","http_status":201,"result_code":"created","body":{"id":"administrator-1","username":"ops","role":"admin","status":"active","must_change_password":true,"must_acknowledge_privacy":true,"temporary_credential_expires_at":"2026-10-24T08:00:00Z","delivery_status":"accepted"}}`,
		`{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`,
		`{"schema_version":"identity-administrator-create-response/v1","http_status":503,"result_code":"identity.email_delivery_failed","body":{"code":"identity.email_delivery_failed","message":"邮件暂未受理","field_errors":[]}}`,
	}
	for index, fixture := range valid {
		if err := insertSnapshotFixture(ctx, pool, index+1, fixture); err != nil {
			t.Fatalf("valid fixture %d rejected: %v", index, err)
		}
	}

	invalid := []struct {
		name    string
		fixture string
	}{
		{name: "outer extra", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","trace_id":"historical","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`},
		{name: "outer missing", fixture: `{"schema_version":"identity-administrator-create-response/v1","result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`},
		{name: "outer null", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":null,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`},
		{name: "outer type drift", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":"409","result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`},
		{name: "mixed legacy v1", fixture: `{"outcome":"failed","schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`},
		{name: "unknown version", fixture: `{"schema_version":"identity-administrator-create-response/v2","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[]}}`},
		{name: "success body extra", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":201,"result_code":"created","body":{"id":"administrator-1","username":"ops","role":"admin","status":"active","must_change_password":true,"must_acknowledge_privacy":true,"temporary_credential_expires_at":"2026-10-24T08:00:00Z","delivery_status":"accepted","trace_id":"historical"}}`},
		{name: "success body missing", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":201,"result_code":"created","body":{"id":"administrator-1","username":"ops","role":"admin","status":"active","must_change_password":true,"temporary_credential_expires_at":"2026-10-24T08:00:00Z","delivery_status":"accepted"}}`},
		{name: "success body null", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":201,"result_code":"created","body":{"id":"administrator-1","username":"ops","role":"admin","status":"active","must_change_password":true,"must_acknowledge_privacy":null,"temporary_credential_expires_at":"2026-10-24T08:00:00Z","delivery_status":"accepted"}}`},
		{name: "success body type drift", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":201,"result_code":"created","body":{"id":"administrator-1","username":"ops","role":"admin","status":"active","must_change_password":"true","must_acknowledge_privacy":true,"temporary_credential_expires_at":"2026-10-24T08:00:00Z","delivery_status":"accepted"}}`},
		{name: "failed body extra", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[],"username":"ops"}}`},
		{name: "failed body missing", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用"}}`},
		{name: "email delivery failed body missing", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":503,"result_code":"identity.email_delivery_failed","body":{"code":"identity.email_delivery_failed","message":"邮件暂未受理"}}`},
		{name: "failed body null", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":null}}`},
		{name: "failed body type drift", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":{}}}`},
		{name: "identifier conflict non empty fields", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":"identity.identifier_conflict","message":"身份标识已被使用","field_errors":[{"field":"username","code":"conflict"}]}}`},
		{name: "identifier conflict code type drift", fixture: `{"schema_version":"identity-administrator-create-response/v1","http_status":409,"result_code":"identity.identifier_conflict","body":{"code":409,"message":"身份标识已被使用","field_errors":[]}}`},
	}
	for index, fixture := range invalid {
		t.Run(fixture.name, func(t *testing.T) {
			if err := insertSnapshotFixture(
				ctx,
				pool,
				index+len(valid)+1,
				fixture.fixture,
			); err == nil {
				t.Fatal("invalid fixture was accepted")
			}
		})
	}
}

type storedSnapshotIdentity struct {
	TupleID  string
	RowXID   string
	Snapshot string
}

func readStoredSnapshotIdentity(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	index int,
) storedSnapshotIdentity {
	t.Helper()
	var identity storedSnapshotIdentity
	err := pool.QueryRow(ctx, `
		SELECT ctid::text, xmin::text, response_snapshot::text
		FROM idempotency_records
		WHERE
			scope_hash = decode(md5($1) || md5($1 || 'x'), 'hex')
			AND key_locator = decode(md5($2) || md5($2 || 'x'), 'hex')
	`, "scope-"+strconv.Itoa(index), "locator-"+strconv.Itoa(index)).Scan(
		&identity.TupleID,
		&identity.RowXID,
		&identity.Snapshot,
	)
	if err != nil {
		t.Fatalf("read stored snapshot identity: %v", err)
	}
	return identity
}

func idempotencyMigrationFS(t *testing.T, through int) fstest.MapFS {
	t.Helper()
	fileSystem := make(fstest.MapFS, through)
	for version := 1; version <= through; version++ {
		matches, err := filepath.Glob(filepath.Join(
			repositoryRoot(t),
			"migrations",
			fmt.Sprintf("%04d_*.sql", version),
		))
		if err != nil {
			t.Fatalf("locate migration %04d: %v", version, err)
		}
		if len(matches) != 1 {
			t.Fatalf("migration %04d matches = %d, want 1", version, len(matches))
		}
		data, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("read migration %04d: %v", version, err)
		}
		fileSystem["migrations/"+filepath.Base(matches[0])] = &fstest.MapFile{Data: data}
	}
	return fileSystem
}

func insertSnapshotFixture(
	ctx context.Context,
	pool *pgxpool.Pool,
	index int,
	snapshot string,
) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO idempotency_records (
			scope_hash, key_locator, key_digest, request_digest,
			digest_key_version, status, execution_generation,
			result_code, response_snapshot, expires_at, created_at, updated_at
		)
		VALUES (
			decode(md5($1) || md5($1 || 'x'), 'hex'),
			decode(md5($2) || md5($2 || 'x'), 'hex'),
			decode(md5($3) || md5($3 || 'x'), 'hex'),
			decode(md5($4) || md5($4 || 'x'), 'hex'),
			'v1', 'succeeded', 1, 'fixture', $5::jsonb,
			$6, $7, $7
		)
	`, "scope-"+strconv.Itoa(index), "locator-"+strconv.Itoa(index),
		"key-"+strconv.Itoa(index), "request-"+strconv.Itoa(index),
		snapshot, time.Now().UTC().Add(time.Hour), time.Now().UTC())
	return err
}

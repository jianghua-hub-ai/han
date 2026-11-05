//go:build integration

package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"han/internal/migration"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestStore_AppendAndMutationFailure_EnforcesAtomicAppendOnly(t *testing.T) {
	ctx := context.Background()
	dsn := testenv.RequirePostgresDSN(t)
	adminPool := openPool(t, dsn, "")
	t.Cleanup(adminPool.Close)
	requirePostgres18(t, ctx, adminPool)
	verifyCleanupBeforePoolOpen(t, ctx, adminPool)

	schemasBefore := requireAuditSchemas(t, ctx, adminPool)
	t.Logf("audit schemas before main test: %v", schemasBefore)
	schema := "audit_" + randomHex(t, 8)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	registerSchemaCleanup(t, adminPool, schema, schemasBefore)

	pool := openPool(t, dsn, schema)
	t.Cleanup(pool.Close)

	runner, err := migration.NewRunner(pool, os.DirFS("../.."), "migrations")
	if err != nil {
		t.Fatalf("create migration runner: %v", err)
	}
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("apply shared migration: %v", err)
	}
	if _, err := pool.Exec(ctx, "CREATE TABLE audit_business_probe (id text PRIMARY KEY)"); err != nil {
		t.Fatalf("create business probe: %v", err)
	}

	store := Store{}
	event := validEvent(t)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin append transaction: %v", err)
	}
	if err := store.Append(ctx, tx, event); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("append audit event: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit append transaction: %v", err)
	}

	got, err := store.Get(ctx, pool, event.ID)
	if err != nil {
		t.Fatalf("get audit event: %v", err)
	}
	requireStoredAuditEventEqual(t, got, event)

	if _, err := pool.Exec(ctx, "UPDATE audit_events SET outcome = 'changed' WHERE id = $1", event.ID); err == nil {
		t.Fatal("UPDATE audit_events error = nil, want append-only rejection")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM audit_events WHERE id = $1", event.ID); err == nil {
		t.Fatal("DELETE audit_events error = nil, want append-only rejection")
	}

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO audit_business_probe (id) VALUES ('must-roll-back')"); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("insert business probe: %v", err)
	}
	if err := store.Append(ctx, tx, event); err == nil {
		_ = tx.Rollback(ctx)
		t.Fatal("duplicate audit append error = nil")
	}
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Fatalf("rollback failed append transaction: %v", err)
	}

	var probeCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_business_probe WHERE id = 'must-roll-back'").Scan(&probeCount); err != nil {
		t.Fatalf("count business probe: %v", err)
	}
	if probeCount != 0 {
		t.Fatalf("business probe count = %d, want rollback to remove it", probeCount)
	}

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin zero-time rollback transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO audit_business_probe (id) VALUES ('zero-time-must-roll-back')"); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("insert zero-time business probe: %v", err)
	}
	zeroTimeEvent := validEvent(t)
	zeroTimeEvent.OccurredAt = time.Time{}
	appendErr := store.Append(ctx, tx, zeroTimeEvent)
	if appendErr == nil {
		_ = tx.Rollback(ctx)
		t.Fatal("zero-time audit append error = nil")
	}
	if !strings.Contains(appendErr.Error(), "occurred_at") {
		_ = tx.Rollback(ctx)
		t.Fatalf("zero-time audit append error = %q, want occurred_at context", appendErr)
	}
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Fatalf("rollback zero-time audit transaction: %v", err)
	}

	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM audit_business_probe WHERE id = 'zero-time-must-roll-back'",
	).Scan(&probeCount); err != nil {
		t.Fatalf("count zero-time business probe: %v", err)
	}
	if probeCount != 0 {
		t.Fatalf("zero-time business probe count = %d, want rollback to remove it", probeCount)
	}
	var auditCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM audit_events WHERE id = $1",
		zeroTimeEvent.ID,
	).Scan(&auditCount); err != nil {
		t.Fatalf("count zero-time audit event: %v", err)
	}
	if auditCount != 0 {
		t.Fatalf("zero-time audit event count = %d, want 0", auditCount)
	}
}

func verifyCleanupBeforePoolOpen(t *testing.T, ctx context.Context, adminPool *pgxpool.Pool) {
	t.Helper()

	schemasBefore := requireAuditSchemas(t, ctx, adminPool)
	t.Logf("audit schemas before setup-failure regression: %v", schemasBefore)
	schema := "audit_" + randomHex(t, 8)

	t.Run("setup failure before pool open", func(t *testing.T) {
		quotedSchema := pgx.Identifier{schema}.Sanitize()
		if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
			t.Fatalf("create setup-failure schema: %v", err)
		}
		registerSchemaCleanup(t, adminPool, schema, schemasBefore)

		setupErr := errors.New("simulated pool setup failure")
		if setupErr == nil {
			t.Fatal("setup error = nil")
		}
	})

	schemasAfter := requireAuditSchemas(t, ctx, adminPool)
	if !slices.Equal(schemasAfter, schemasBefore) {
		t.Fatalf("audit schemas after setup failure = %v, want %v", schemasAfter, schemasBefore)
	}
}

func registerSchemaCleanup(
	t *testing.T,
	adminPool *pgxpool.Pool,
	schema string,
	schemasBefore []string,
) {
	t.Helper()
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("drop isolated schema: %v", err)
			return
		}
		var remaining *string
		if err := adminPool.QueryRow(cleanupCtx, "SELECT to_regnamespace($1)::text", schema).Scan(&remaining); err != nil {
			t.Errorf("verify isolated schema cleanup: %v", err)
			return
		}
		if remaining != nil {
			t.Errorf("isolated schema still exists after cleanup: %s", *remaining)
			return
		}

		schemasAfter := requireAuditSchemas(t, cleanupCtx, adminPool)
		t.Logf("audit schemas after cleanup: %v", schemasAfter)
		if !slices.Equal(schemasAfter, schemasBefore) {
			t.Errorf("audit schemas after cleanup = %v, want %v", schemasAfter, schemasBefore)
		}
	})
}

func requireAuditSchemas(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT nspname
		FROM pg_namespace
		WHERE left(nspname, 6) = 'audit_'
		ORDER BY nspname
	`)
	if err != nil {
		t.Fatalf("list audit schemas: %v", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			t.Fatalf("scan audit schema: %v", err)
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit schemas: %v", err)
	}
	return schemas
}

func openPool(t *testing.T, dsn, schema string) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL test configuration: %v", err)
	}
	if schema != "" {
		config.ConnConfig.RuntimeParams["search_path"] = schema
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("create PostgreSQL test pool: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping PostgreSQL test pool: %v", err)
	}
	return pool
}

func requirePostgres18(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var version int
	if err := pool.QueryRow(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&version); err != nil {
		t.Fatalf("read PostgreSQL server version: %v", err)
	}
	if major := version / 10000; major != 18 {
		t.Fatalf("PostgreSQL major = %d, want 18", major)
	}
}

func validEvent(t *testing.T) Event {
	t.Helper()
	userID := randomUUID(t)
	role := "member"
	errorCode := "synthetic.error"
	source := "synthetic-source-digest"
	return Event{
		ID:                randomUUID(t),
		OccurredAt:        time.Date(2026, time.October, 21, 10, 0, 0, 123000, time.UTC),
		ActorKind:         ActorUser,
		ActorUserID:       &userID,
		ActorRole:         &role,
		Action:            "identity.privacy.acknowledge",
		TargetType:        "privacy_notice",
		TargetID:          "notice-v1",
		Outcome:           "failed",
		ErrorCode:         &errorCode,
		TraceID:           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		SourceFingerprint: &source,
		Metadata: map[string]any{
			"notice_version": "v1",
		},
	}
}

func requireStoredAuditEventEqual(t *testing.T, got Event, want Event) {
	t.Helper()
	if got.ID != want.ID ||
		!got.OccurredAt.Equal(want.OccurredAt) ||
		got.ActorKind != want.ActorKind ||
		!equalAuditStringPointer(got.ActorUserID, want.ActorUserID) ||
		!equalAuditStringPointer(got.ActorSystem, want.ActorSystem) ||
		!equalAuditStringPointer(got.ActorRole, want.ActorRole) ||
		got.Action != want.Action ||
		got.TargetType != want.TargetType ||
		got.TargetID != want.TargetID ||
		got.Outcome != want.Outcome ||
		!equalAuditStringPointer(got.ErrorCode, want.ErrorCode) ||
		got.TraceID != want.TraceID ||
		!equalAuditStringPointer(got.SourceFingerprint, want.SourceFingerprint) ||
		!reflect.DeepEqual(got.Metadata, want.Metadata) {
		t.Fatal("stored audit event differs from the complete prepared event")
	}
}

func equalAuditStringPointer(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func randomUUID(t *testing.T) string {
	t.Helper()
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate test UUID: %v", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value[:])
	return strings.Join([]string{
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	}, "-")
}

func randomHex(t *testing.T, size int) string {
	t.Helper()
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generate random schema suffix: %v", err)
	}
	return hex.EncodeToString(value)
}

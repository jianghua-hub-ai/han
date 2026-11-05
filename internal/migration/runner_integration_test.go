//go:build integration

package migration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSetupError_MalformedAndFailedDSN_RedactsSensitiveInput(t *testing.T) {
	const (
		sentinel = "b0-cr-013-supersecret"
		dsn      = "password =" + sentinel + " invalid"
	)

	_, parseErr := pgxpool.ParseConfig(dsn)
	if parseErr == nil {
		t.Fatal("pgxpool.ParseConfig() error = nil, want malformed DSN error")
	}

	tests := []struct {
		name  string
		stage string
		err   error
	}{
		{
			name:  "parse",
			stage: "parse PostgreSQL test DSN",
			err:   parseErr,
		},
		{
			name:  "connect",
			stage: "connect PostgreSQL test database",
			err:   errors.New("connection failed for " + dsn),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgresSetupError(tt.stage, tt.err)

			if got != tt.stage {
				t.Error("postgresSetupError() returned non-generic text")
			}
			if strings.Contains(got, sentinel) {
				t.Error("postgresSetupError() leaked password sentinel")
			}
			if strings.Contains(got, dsn) {
				t.Error("postgresSetupError() leaked raw DSN")
			}
		})
	}
}

func TestRunnerConformance_EnvironmentBaseline(t *testing.T) {
	pool := newPostgres18Pool(t)

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatal(postgresSetupError("ping PostgreSQL 18 conformance pool", err))
	}
}

func TestRunnerConformance_LineCommentTerminatorsOracle(t *testing.T) {
	pool := newPostgres18Pool(t)
	ctx := context.Background()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire PostgreSQL oracle connection: %v", err)
	}
	defer conn.Release()

	tests := []struct {
		name       string
		terminator string
	}{
		{name: "LF", terminator: "\n"},
		{name: "CR", terminator: "\r"},
		{name: "CRLF", terminator: "\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commandTag, execErr := conn.Exec(
				ctx,
				"BEGIN; -- ordinary comment"+tt.terminator+"ROLLBACK;",
			)
			if conn.Conn().PgConn().TxStatus() != 'I' {
				if _, rollbackErr := conn.Exec(ctx, "ROLLBACK"); rollbackErr != nil {
					t.Fatalf("cleanup oracle transaction: %v", rollbackErr)
				}
			}
			if execErr != nil {
				t.Fatalf("execute line-comment oracle: %v", execErr)
			}
			if commandTag.String() != "ROLLBACK" {
				t.Fatalf("last command tag = %q, want ROLLBACK", commandTag.String())
			}
		})
	}
}

func TestRunnerConformance_LexemePack(t *testing.T) {
	pool := newPostgres18Pool(t)
	fsys := fstest.MapFS{
		"migrations/0001_lexeme_pack.sql": {
			Data: []byte(`
				CREATE TABLE lexeme_probe (id integer PRIMARY KEY);
				INSERT INTO lexeme_probe (id) VALUES (1);
				/* outer /* COMMIT; */ ROLLBACK; */
				SELECT 'COMMIT '' ROLLBACK';
				SELECT 'backslash\';
				SELECT E'COMMIT \' ROLLBACK';
				SELECT 1 AS "COMMIT "" ROLLBACK";
				SELECT $$ COMMIT ROLLBACK $$;
				SELECT $body$ COMMIT ROLLBACK $body$;
				SELECT $é$ COMMIT ROLLBACK $é$;
				SELECT $�$ COMMIT ROLLBACK $�$;
				SELECT 1 AS foo$guard$;
				SELECT 1 AS é$guard$;
				SELECT 1 AS COMMIＴ;
			`),
		},
	}
	runner := newRunnerForTest(t, pool, fsys, "migrations")

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() lexeme pack error = %v", err)
	}

	var probeCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM lexeme_probe").Scan(&probeCount); err != nil {
		t.Fatalf("query lexeme probe: %v", err)
	}
	if probeCount != 1 {
		t.Fatalf("lexeme_probe count = %d, want 1", probeCount)
	}
	assertMigrationVersionCount(t, pool, 1, 1)

	invalidTagFS := fstest.MapFS{
		"migrations/0002_invalid_digit_tag.sql": {
			Data: []byte(`
				CREATE TABLE invalid_digit_tag_side_effect (id integer PRIMARY KEY);
				SELECT $1tag$ COMMIT $1tag$;
			`),
		},
	}
	invalidTagRunner := newRunnerForTest(t, pool, invalidTagFS, "migrations")
	if err := invalidTagRunner.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want PostgreSQL rejection for digit-start dollar tag")
	}
	assertRelationAbsent(t, pool, "invalid_digit_tag_side_effect")
	assertMigrationVersionCount(t, pool, 2, 0)
}

func TestRunnerConformance_BeginAtomicNestedCase(t *testing.T) {
	pool := newPostgres18Pool(t)
	fsys := fstest.MapFS{
		"migrations/0001_begin_atomic.sql": {
			Data: []byte(`
				CREATE TABLE atomic_probe (value integer NOT NULL);

				CREATE FUNCTION conformance_value(input integer)
				RETURNS integer
				LANGUAGE SQL
				BEGIN ATOMIC
					SELECT CASE
						WHEN input > 0 THEN CASE WHEN input > 1 THEN input ELSE 1 END
						ELSE 0
					END;
				END;

				CREATE PROCEDURE conformance_insert(input integer)
				LANGUAGE SQL
				BEGIN ATOMIC
					INSERT INTO atomic_probe(value)
					VALUES (
						CASE
							WHEN input > 0 THEN CASE WHEN input > 1 THEN input ELSE 1 END
							ELSE 0
						END
					);
				END;
			`),
		},
	}
	runner := newRunnerForTest(t, pool, fsys, "migrations")

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() BEGIN ATOMIC migration error = %v", err)
	}

	var negativeResult, positiveResult int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT conformance_value(-1), conformance_value(2)",
	).Scan(&negativeResult, &positiveResult); err != nil {
		t.Fatalf("call conformance function: %v", err)
	}
	if negativeResult != 0 || positiveResult != 2 {
		t.Fatalf("conformance function results = (%d, %d), want (0, 2)", negativeResult, positiveResult)
	}
	if _, err := pool.Exec(context.Background(), "CALL conformance_insert(2)"); err != nil {
		t.Fatalf("call conformance procedure: %v", err)
	}
	var insertedValue int
	if err := pool.QueryRow(context.Background(), "SELECT value FROM atomic_probe").Scan(&insertedValue); err != nil {
		t.Fatalf("query atomic probe: %v", err)
	}
	if insertedValue != 2 {
		t.Fatalf("atomic_probe value = %d, want 2", insertedValue)
	}
	assertMigrationVersionCount(t, pool, 1, 1)
}

func TestRunnerConformance_RejectsBareNestedAtomic(t *testing.T) {
	pool := newPostgres18Pool(t)
	fsys := fstest.MapFS{
		"migrations/0001_invalid_nested_atomic.sql": {
			Data: []byte(`
				CREATE TABLE nested_atomic_side_effect (id integer PRIMARY KEY);
				CREATE FUNCTION invalid_nested_atomic()
				RETURNS integer
				LANGUAGE SQL
				BEGIN ATOMIC
					BEGIN ATOMIC SELECT 1; END;
				END;
			`),
		},
		"migrations/0002_after_invalid.sql": {
			Data: []byte("CREATE TABLE after_nested_atomic (id integer PRIMARY KEY);"),
		},
	}
	runner := newRunnerForTest(t, pool, fsys, "migrations")

	if err := runner.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want PostgreSQL syntax rejection for bare nested BEGIN ATOMIC")
	}

	assertRelationAbsent(t, pool, "nested_atomic_side_effect")
	assertRelationAbsent(t, pool, "after_nested_atomic")
	assertMigrationVersionCount(t, pool, 1, 0)
	assertMigrationVersionCount(t, pool, 2, 0)
}

func TestRunnerConformance_RejectsTransactionControlAfterBody(t *testing.T) {
	transactionCommands := []string{"COMMIT", "ROLLBACK"}
	bodies := []struct {
		name string
		sql  string
	}{
		{name: "line_comment_cr", sql: "SELECT 1; -- ordinary comment\r"},
		{name: "nested_block_comment", sql: "SELECT 1 /* outer /* COMMIT */ ROLLBACK */;"},
		{name: "ordinary_string", sql: "SELECT 'COMMIT '' ROLLBACK';"},
		{name: "escape_string", sql: `SELECT E'COMMIT \' ROLLBACK';`},
		{name: "quoted_identifier", sql: `SELECT 1 AS "COMMIT "" ROLLBACK";`},
		{name: "empty_dollar_tag", sql: "SELECT $$ COMMIT ROLLBACK $$;"},
		{name: "ascii_dollar_tag", sql: "SELECT $body$ COMMIT ROLLBACK $body$;"},
		{name: "high_bit_dollar_tag", sql: "SELECT $é$ COMMIT ROLLBACK $é$;"},
		{name: "replacement_character_dollar_tag", sql: "SELECT $�$ COMMIT ROLLBACK $�$;"},
		{name: "ascii_identifier_adjacent_dollar", sql: "SELECT 1 AS foo$guard$;"},
		{name: "unicode_identifier_adjacent_dollar", sql: "SELECT 1 AS é$guard$;"},
		{
			name: "function_begin_atomic",
			sql: `
				CREATE FUNCTION transaction_control_probe()
				RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC SELECT 1; END;
			`,
		},
		{
			name: "procedure_begin_atomic",
			sql: `
				CREATE PROCEDURE transaction_control_probe()
				LANGUAGE SQL
				BEGIN ATOMIC SELECT 1; END;
			`,
		},
		{
			name: "begin_atomic_nested_case",
			sql: `
				CREATE FUNCTION transaction_control_probe(input integer)
				RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC
					SELECT CASE
						WHEN input > 0 THEN CASE WHEN input > 1 THEN input ELSE 1 END
						ELSE 0
					END;
				END;
			`,
		},
	}

	for _, body := range bodies {
		body := body
		for _, command := range transactionCommands {
			command := command
			t.Run(body.name+"_then_"+strings.ToLower(command), func(t *testing.T) {
				pool := newPostgres18Pool(t)
				fsys := fstest.MapFS{
					"migrations/0001_rejected.sql": {
						Data: []byte(
							"CREATE TABLE transaction_control_side_effect (id integer PRIMARY KEY);\n" +
								body.sql + "\n" + command + ";",
						),
					},
				}
				runner := newRunnerForTest(t, pool, fsys, "migrations")

				if err := runner.Run(context.Background()); err == nil {
					t.Fatalf("Run() error = nil, want %s rejection before database side effects", command)
				}

				assertRelationAbsent(t, pool, "transaction_control_side_effect")
				assertRelationAbsent(t, pool, "schema_migrations")
			})
		}
	}
}

func TestRunner_AppliesInOrder_RecordsOnlyOnce(t *testing.T) {
	pool := newPostgres18Pool(t)
	fsys := fstest.MapFS{
		"migrations/0002_insert_probe.sql": {
			Data: []byte("INSERT INTO ordered_probe (version) VALUES (2);"),
		},
		"migrations/0001_create_probe.sql": {
			Data: []byte("CREATE TABLE ordered_probe (version integer PRIMARY KEY);"),
		},
	}
	runner := newRunnerForTest(t, pool, fsys, "migrations")

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	var probeCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM ordered_probe").Scan(&probeCount); err != nil {
		t.Fatalf("query ordered_probe: %v", err)
	}
	if probeCount != 1 {
		t.Fatalf("ordered_probe count = %d, want 1", probeCount)
	}

	var migrationCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if migrationCount != 2 {
		t.Fatalf("schema_migrations count = %d, want 2", migrationCount)
	}
}

func TestRunner_FailingMigration_RollsBackAndStops(t *testing.T) {
	pool := newPostgres18Pool(t)
	runner := newRunnerForTest(t, pool, os.DirFS("."), "testdata")

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("Run() error = nil, want migration failure")
	}

	var appliedVersions []int
	rows, queryErr := pool.Query(context.Background(), "SELECT version FROM schema_migrations ORDER BY version")
	if queryErr != nil {
		t.Fatalf("query schema_migrations: %v", queryErr)
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		if scanErr := rows.Scan(&version); scanErr != nil {
			t.Fatalf("scan version: %v", scanErr)
		}
		appliedVersions = append(appliedVersions, version)
	}
	if rows.Err() != nil {
		t.Fatalf("iterate schema_migrations: %v", rows.Err())
	}
	if len(appliedVersions) != 1 || appliedVersions[0] != 1 {
		t.Fatalf("applied versions = %v, want [1]", appliedVersions)
	}

	var rolledBackTableExists bool
	if queryErr := pool.QueryRow(
		context.Background(),
		"SELECT to_regclass('rolled_back_probe') IS NOT NULL",
	).Scan(&rolledBackTableExists); queryErr != nil {
		t.Fatalf("query rolled_back table: %v", queryErr)
	}
	if rolledBackTableExists {
		t.Fatal("rolled_back_probe exists, want transaction rollback")
	}

	var laterTableExists bool
	if queryErr := pool.QueryRow(
		context.Background(),
		"SELECT to_regclass('after_failure_probe') IS NOT NULL",
	).Scan(&laterTableExists); queryErr != nil {
		t.Fatalf("query after-failure table: %v", queryErr)
	}
	if laterTableExists {
		t.Fatal("after_failure_probe exists, want later migration not executed")
	}

	var laterVersionCount int
	if queryErr := pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM schema_migrations WHERE version = 3",
	).Scan(&laterVersionCount); queryErr != nil {
		t.Fatalf("query after-failure migration record: %v", queryErr)
	}
	if laterVersionCount != 0 {
		t.Fatalf("migration version 3 count = %d, want 0", laterVersionCount)
	}
}

func TestRunner_ConcurrentRuns_SerializesWithAdvisoryLock(t *testing.T) {
	pool := newPostgres18Pool(t)
	fsys := fstest.MapFS{
		"migrations/0001_lock_probe.sql": {
			Data: []byte(`
				CREATE TABLE lock_probe (id integer PRIMARY KEY);
				SELECT pg_sleep(0.25);
				INSERT INTO lock_probe (id) VALUES (1);
			`),
		},
	}

	const runnerCount = 2
	start := make(chan struct{})
	errs := make(chan error, runnerCount)
	var ready sync.WaitGroup
	ready.Add(runnerCount)

	for i := 0; i < runnerCount; i++ {
		runner := newRunnerForTest(t, pool, fsys, "migrations")
		go func() {
			ready.Done()
			<-start
			errs <- runner.Run(context.Background())
		}()
	}

	ready.Wait()
	close(start)

	for i := 0; i < runnerCount; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent Run() error = %v", err)
		}
	}

	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM lock_probe").Scan(&count); err != nil {
		t.Fatalf("query lock_probe: %v", err)
	}
	if count != 1 {
		t.Fatalf("lock_probe count = %d, want 1", count)
	}
}

func TestRunner_DiscardsDedicatedConnection(t *testing.T) {
	pool := newPostgres18PoolWithMaxConns(t, 1)
	fsys := fstest.MapFS{
		"migrations/0001_connection_probe.sql": {
			Data: []byte(`
				CREATE TABLE connection_probe (backend_pid integer NOT NULL);
				INSERT INTO connection_probe (backend_pid) VALUES (pg_backend_pid());
			`),
		},
	}
	runner := newRunnerForTest(t, pool, fsys, "migrations")

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var migrationPID, queryPID int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT backend_pid, pg_backend_pid() FROM connection_probe",
	).Scan(&migrationPID, &queryPID); err != nil {
		t.Fatalf("query connection probe: %v", err)
	}
	if migrationPID == queryPID {
		t.Fatalf("migration backend PID = query backend PID = %d, want discarded migration connection", migrationPID)
	}
}

func newRunnerForTest(t *testing.T, pool *pgxpool.Pool, fsys fs.FS, directory string) *Runner {
	t.Helper()

	runner, err := NewRunner(pool, fsys, directory)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	return runner
}

func newPostgres18Pool(t *testing.T) *pgxpool.Pool {
	return newPostgres18PoolWithMaxConns(t, 0)
}

func postgresSetupError(stage string, _ error) string {
	return stage
}

func newPostgres18PoolWithMaxConns(t *testing.T, maxConns int32) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	dsn := testenv.RequirePostgresDSN(t)
	adminPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(postgresSetupError("configure PostgreSQL test database", err))
	}
	t.Cleanup(adminPool.Close)

	assertPostgres18ConformanceBaseline(t, ctx, adminPool)

	schema := "han_migration_" + randomHex(t, 8)
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(postgresSetupError("create PostgreSQL test schema", err))
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error(postgresSetupError("drop PostgreSQL test schema", err))
		}
	})

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(postgresSetupError("parse PostgreSQL test DSN", err))
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	if maxConns > 0 {
		config.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(postgresSetupError("configure schema-scoped PostgreSQL test pool", err))
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(postgresSetupError("connect schema-scoped PostgreSQL test pool", err))
	}
	return pool
}

func assertPostgres18ConformanceBaseline(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	var versionNumber int
	var databaseEncoding, clientEncoding, standardConformingStrings string
	if err := pool.QueryRow(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&versionNumber); err != nil {
		t.Fatal(postgresSetupError("query PostgreSQL test server version", err))
	}
	if err := pool.QueryRow(
		ctx,
		"SELECT pg_encoding_to_char(encoding) FROM pg_database WHERE datname = current_database()",
	).Scan(&databaseEncoding); err != nil {
		t.Fatal(postgresSetupError("query PostgreSQL test database encoding", err))
	}
	if err := pool.QueryRow(ctx, "SHOW client_encoding").Scan(&clientEncoding); err != nil {
		t.Fatal(postgresSetupError("query PostgreSQL test client encoding", err))
	}
	if err := pool.QueryRow(
		ctx,
		"SHOW standard_conforming_strings",
	).Scan(&standardConformingStrings); err != nil {
		t.Fatal(postgresSetupError("query PostgreSQL test string conformance setting", err))
	}

	t.Logf(
		"PostgreSQL conformance baseline: server_version_num=%d database_encoding=%s client_encoding=%s standard_conforming_strings=%s",
		versionNumber,
		databaseEncoding,
		clientEncoding,
		standardConformingStrings,
	)
	if versionNumber/10000 != 18 {
		t.Fatalf("PostgreSQL major = %d, require 18", versionNumber/10000)
	}
	if !strings.EqualFold(databaseEncoding, "UTF8") {
		t.Fatalf("PostgreSQL database encoding = %q, require UTF8", databaseEncoding)
	}
	if !strings.EqualFold(clientEncoding, "UTF8") {
		t.Fatalf("PostgreSQL client encoding = %q, require UTF8", clientEncoding)
	}
	if !strings.EqualFold(standardConformingStrings, "on") {
		t.Fatalf("standard_conforming_strings = %q, require on", standardConformingStrings)
	}
}

func assertRelationAbsent(t *testing.T, pool *pgxpool.Pool, relation string) {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(
		context.Background(),
		"SELECT to_regclass($1) IS NOT NULL",
		relation,
	).Scan(&exists); err != nil {
		t.Fatalf("query relation %q: %v", relation, err)
	}
	if exists {
		t.Fatalf("relation %q exists, want absent", relation)
	}
}

func assertMigrationVersionCount(t *testing.T, pool *pgxpool.Pool, version, want int) {
	t.Helper()

	var count int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM schema_migrations WHERE version = $1",
		version,
	).Scan(&count); err != nil {
		t.Fatalf("query migration version %d: %v", version, err)
	}
	if count != want {
		t.Fatalf("migration version %d count = %d, want %d", version, count, want)
	}
}

func randomHex(t *testing.T, byteCount int) string {
	t.Helper()

	data := make([]byte, byteCount)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("read random bytes: %v", err)
	}
	return hex.EncodeToString(data)
}

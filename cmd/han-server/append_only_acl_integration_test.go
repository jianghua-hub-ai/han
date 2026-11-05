//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/testenv"
)

// TestAppendOnlyApplicationACL proves on a real PostgreSQL server that:
//   - a fresh install reaches the final append-only ACL from provision.sql's
//     default privileges alone, with no manual grant before the startup gate;
//   - a real non-superuser LOGIN identity can INSERT but cannot UPDATE,
//     DELETE, or TRUNCATE append-only tables;
//   - the startup gate rejects superuser sessions, SET ROLE switching, table
//     owners, and column-level UPDATE grants.
//
// Boundary: this test does not prove that search_path points at the schema
// provisioned by deploy/db/provision.sql. For the application search_path
// used here, the gate requires exactly one same-name relation in
// current_schemas(false) and exactly one pg_table_is_visible relation.
// Historical same-name schemas are never dropped or cleaned up; operators
// must keep search_path limited to the provisioned target schema.
func TestAppendOnlyApplicationACL(t *testing.T) {
	suffix := aclRandomHex(t)
	schema := "han_s15_acl_" + suffix
	migratorRole := "han_s15_mig_" + suffix
	applicationRole := "han_s15_app_" + suffix
	migratorPassword := aclRandomHex(t) + aclRandomHex(t)
	applicationPassword := aclRandomHex(t) + aclRandomHex(t)

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	superuserDSN := testenv.RequirePostgresDSN(t)
	superuserPool := aclDSNPool(t, ctx, superuserDSN)
	aclPreflight(t, ctx, superuserPool)

	// F009: only register retained resources after the superuser preflight has
	// proved the target server. Nothing is dropped; the planned and created
	// records are appended so a failed creation still leaves a conservative
	// retention list.
	aclRetain(t, schema, migratorRole, applicationRole, "planned")

	if _, err := superuserPool.Exec(ctx,
		"CREATE ROLE "+pgx.Identifier{migratorRole}.Sanitize()+
			" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD "+
			aclQuoteLiteral(migratorPassword)); err != nil {
		t.Fatalf("create migration LOGIN role: %v", err)
	}
	if _, err := superuserPool.Exec(ctx,
		"CREATE ROLE "+pgx.Identifier{applicationRole}.Sanitize()+
			" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD "+
			aclQuoteLiteral(applicationPassword)); err != nil {
		t.Fatalf("create application LOGIN role: %v", err)
	}

	// Mirrors deploy/db/provision.sql, including the explicit schema owner.
	provisionStatements := []string{
		"CREATE SCHEMA " + pgx.Identifier{schema}.Sanitize() +
			" AUTHORIZATION " + pgx.Identifier{migratorRole}.Sanitize(),
		"ALTER SCHEMA " + pgx.Identifier{schema}.Sanitize() +
			" OWNER TO " + pgx.Identifier{migratorRole}.Sanitize(),
		"REVOKE CREATE ON SCHEMA " + pgx.Identifier{schema}.Sanitize() + " FROM PUBLIC",
		"REVOKE ALL ON SCHEMA " + pgx.Identifier{schema}.Sanitize() +
			" FROM " + pgx.Identifier{applicationRole}.Sanitize(),
		"GRANT USAGE ON SCHEMA " + pgx.Identifier{schema}.Sanitize() +
			" TO " + pgx.Identifier{applicationRole}.Sanitize(),
		"ALTER DEFAULT PRIVILEGES FOR ROLE " + pgx.Identifier{migratorRole}.Sanitize() +
			" IN SCHEMA " + pgx.Identifier{schema}.Sanitize() +
			" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO " +
			pgx.Identifier{applicationRole}.Sanitize(),
		"ALTER DEFAULT PRIVILEGES FOR ROLE " + pgx.Identifier{migratorRole}.Sanitize() +
			" IN SCHEMA " + pgx.Identifier{schema}.Sanitize() +
			" GRANT USAGE, SELECT ON SEQUENCES TO " +
			pgx.Identifier{applicationRole}.Sanitize(),
	}
	for _, statement := range provisionStatements {
		if _, err := superuserPool.Exec(ctx, statement); err != nil {
			t.Fatalf("provision: %v", err)
		}
	}
	aclRetain(t, schema, migratorRole, applicationRole, "created")
	t.Logf("schema=%s stage=provisioned migrator_role=%s application_role=%s schema_owner=%s",
		schema, migratorRole, applicationRole, migratorRole)

	base := superuserPool.Config().ConnConfig
	applicationDSN := aclRoleDSN(
		base.Host, base.Port, base.Database, applicationRole, applicationPassword, schema,
	)
	migrationDSN := aclRoleDSN(
		base.Host, base.Port, base.Database, migratorRole, migratorPassword, schema,
	)

	applicationPool := aclDSNPool(t, ctx, applicationDSN)
	env := func(key string) string {
		switch key {
		case databaseDSNEnvironment:
			return applicationDSN
		case migrationDatabaseDSNEnvironment:
			return migrationDSN
		default:
			return ""
		}
	}

	// Fresh-install smoke: only provision defaults, then the production
	// migration path. No manual GRANT/REVOKE runs before the gate.
	if err := productionMigrationFrom(
		ctx, applicationPool, env, os.DirFS(filepath.Join("..", "..")),
	); err != nil {
		t.Fatalf("fresh install startup migration and gate: %v", err)
	}
	t.Logf("schema=%s stage=fresh_install_gate_passed default_privileges_only=true manual_grant=false",
		schema)

	migratorPool := aclDSNPool(t, ctx, migrationDSN)

	t.Run("fresh_install_default_privileges_gate_passed", func(t *testing.T) {
		if err := verifyAppendOnlyApplicationPrivileges(
			ctx, applicationPool, appendOnlyTables,
		); err != nil {
			t.Fatalf("gate after provision defaults and migration: %v", err)
		}
		t.Logf("schema=%s stage=gate_passed source=provision_defaults_only", schema)
	})

	t.Run("schema_owner_is_migrator", func(t *testing.T) {
		var owner string
		var ownerSuperuser bool
		if err := superuserPool.QueryRow(ctx, `
SELECT pg_catalog.pg_get_userbyid(namespace.nspowner),
       (SELECT role.rolsuper
        FROM pg_catalog.pg_roles AS role
        WHERE role.rolname = pg_catalog.pg_get_userbyid(namespace.nspowner))
FROM pg_catalog.pg_namespace AS namespace
WHERE namespace.nspname = $1`, schema).Scan(&owner, &ownerSuperuser); err != nil {
			t.Fatalf("read schema owner: %v", err)
		}
		if owner != migratorRole || ownerSuperuser {
			t.Fatalf("schema owner = %q superuser=%t; want %q superuser=false",
				owner, ownerSuperuser, migratorRole)
		}
		t.Logf("schema=%s stage=schema_owner_verified owner=%s owner_superuser=false",
			schema, owner)
	})

	t.Run("non_superuser_login_positive", func(t *testing.T) {
		var currentUser, sessionUser string
		var superuser, canLogin bool
		if err := applicationPool.QueryRow(ctx, `
SELECT current_user::text, session_user::text, role.rolsuper, role.rolcanlogin
FROM pg_catalog.pg_roles AS role
WHERE role.rolname = current_user`).Scan(
			&currentUser, &sessionUser, &superuser, &canLogin,
		); err != nil {
			t.Fatalf("read application identity: %v", err)
		}
		if currentUser != applicationRole || sessionUser != applicationRole ||
			superuser || !canLogin {
			t.Fatalf("application identity current=%q session=%q superuser=%t login=%t",
				currentUser, sessionUser, superuser, canLogin)
		}

		if err := migratorPool.QueryRow(ctx, `
SELECT current_user::text, session_user::text, role.rolsuper, role.rolcanlogin
FROM pg_catalog.pg_roles AS role
WHERE role.rolname = current_user`).Scan(
			&currentUser, &sessionUser, &superuser, &canLogin,
		); err != nil {
			t.Fatalf("read migration identity: %v", err)
		}
		if currentUser != migratorRole || sessionUser != migratorRole ||
			superuser || !canLogin {
			t.Fatalf("migration identity current=%q session=%q superuser=%t login=%t",
				currentUser, sessionUser, superuser, canLogin)
		}
		t.Logf("schema=%s stage=non_superuser_login_positive application=%s migrator=%s superuser=false",
			schema, applicationRole, migratorRole)
	})

	t.Run("insert_allowed", func(t *testing.T) {
		if _, err := applicationPool.Exec(ctx, `
INSERT INTO audit_events
 (id, occurred_at, actor_kind, actor_system, action, target_type, target_id, outcome, trace_id)
 VALUES (gen_random_uuid(), now(), 'system', 'acl-probe', 'acl.probe', 'acl_probe', '1', 'ok', 'trace-acl-probe')`,
		); err != nil {
			t.Fatalf("application INSERT into audit_events: %v", err)
		}
		if _, err := applicationPool.Exec(ctx, `
INSERT INTO processing_gate_records
 (region_ref, controller_ref, legal_basis_ref, smtp_processor_ref, effective_at, recorded_by)
 VALUES ('region', 'controller', 'legal', 'processor', now(), 'acl-probe')`,
		); err != nil {
			t.Fatalf("application INSERT into processing_gate_records: %v", err)
		}
		t.Logf("schema=%s stage=insert_allowed audit_events=true processing_gate_records=true",
			schema)
	})

	t.Run("migration_ledger_denied", func(t *testing.T) {
		_, err := applicationPool.Exec(ctx, "SELECT count(*) FROM schema_migrations")
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != "42501" {
			t.Fatalf("application SELECT schema_migrations error = %v; want SQLSTATE 42501", err)
		}
		t.Logf("schema=%s stage=migration_ledger_denied sqlstate=%s", schema, pgError.Code)
	})

	t.Run("mutation_denied", func(t *testing.T) {
		statements := map[string]string{
			"audit_events_update":      "UPDATE audit_events SET outcome = outcome",
			"audit_events_delete":      "DELETE FROM audit_events",
			"audit_events_truncate":    "TRUNCATE audit_events",
			"processing_gate_update":   "UPDATE processing_gate_records SET region_ref = region_ref",
			"processing_gate_delete":   "DELETE FROM processing_gate_records",
			"processing_gate_truncate": "TRUNCATE processing_gate_records",
		}
		for name, statement := range statements {
			_, err := applicationPool.Exec(ctx, statement)
			var pgError *pgconn.PgError
			if !errors.As(err, &pgError) || pgError.Code != "42501" {
				t.Fatalf("%s error = %v; want SQLSTATE 42501 insufficient_privilege",
					name, err)
			}
			t.Logf("schema=%s stage=mutation_denied statement=%s sqlstate=%s",
				schema, name, pgError.Code)
		}
	})

	t.Run("schema_create_denied", func(t *testing.T) {
		_, err := applicationPool.Exec(ctx,
			"CREATE TABLE "+pgx.Identifier{schema, "acl_probe_denied"}.Sanitize()+" (id int)")
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != "42501" {
			t.Fatalf("application CREATE TABLE error = %v; want SQLSTATE 42501", err)
		}
		t.Logf("schema=%s stage=schema_create_denied sqlstate=%s", schema, pgError.Code)
	})

	t.Run("superuser_session_rejected", func(t *testing.T) {
		err := verifyAppendOnlyApplicationPrivileges(ctx, superuserPool, appendOnlyTables)
		if err == nil {
			t.Fatal("append-only gate accepted a superuser session")
		}
		if !errors.Is(err, errAppendOnlyPrivilege) ||
			!strings.Contains(err.Error(), "superuser") {
			t.Fatalf("superuser rejection = %v; want superuser rejection", err)
		}
		t.Logf("schema=%s stage=superuser_session_rejected", schema)
	})

	t.Run("superuser_set_role_rejected", func(t *testing.T) {
		switchPool := aclSuperuserSetRolePool(
			t, ctx, superuserDSN, schema, applicationRole,
		)
		err := verifyAppendOnlyApplicationPrivileges(ctx, switchPool, appendOnlyTables)
		if err == nil {
			t.Fatal("append-only gate accepted superuser session with SET ROLE")
		}
		if !errors.Is(err, errAppendOnlyPrivilege) ||
			!strings.Contains(err.Error(), "role switching") {
			t.Fatalf("SET ROLE rejection = %v; want role switching rejection", err)
		}
		t.Logf("schema=%s stage=superuser_set_role_rejected", schema)
	})

	t.Run("owner_identity_rejected", func(t *testing.T) {
		err := verifyAppendOnlyApplicationPrivileges(ctx, migratorPool, appendOnlyTables)
		if err == nil {
			t.Fatal("append-only gate accepted the table owner")
		}
		if !errors.Is(err, errAppendOnlyPrivilege) ||
			!strings.Contains(err.Error(), "owner") {
			t.Fatalf("owner rejection = %v; want owner rejection", err)
		}
		t.Logf("schema=%s stage=owner_rejected identity=%s", schema, migratorRole)
	})

	t.Run("column_update_rejected", func(t *testing.T) {
		table := pgx.Identifier{schema, "audit_events"}.Sanitize()
		role := pgx.Identifier{applicationRole}.Sanitize()
		grant := "GRANT UPDATE (outcome) ON TABLE " + table + " TO " + role
		revoke := "REVOKE UPDATE (outcome) ON TABLE " + table + " FROM " + role
		if _, err := migratorPool.Exec(ctx, grant); err != nil {
			t.Fatalf("grant column UPDATE: %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := migratorPool.Exec(cleanupCtx, revoke); err != nil {
				t.Errorf("revoke column UPDATE: %v", err)
			}
		})

		err := verifyAppendOnlyApplicationPrivileges(ctx, applicationPool, appendOnlyTables)
		if err == nil {
			t.Fatal("append-only gate accepted a column-level UPDATE grant")
		}
		if !errors.Is(err, errAppendOnlyPrivilege) ||
			!strings.Contains(err.Error(), "update_column=true") {
			t.Fatalf("column UPDATE rejection = %v; want update_column=true", err)
		}

		if _, err := migratorPool.Exec(ctx, revoke); err != nil {
			t.Fatalf("revoke column UPDATE: %v", err)
		}
		if err := verifyAppendOnlyApplicationPrivileges(
			ctx, applicationPool, appendOnlyTables,
		); err != nil {
			t.Fatalf("gate after revoking column UPDATE: %v", err)
		}
		t.Logf("schema=%s stage=column_update_rejected then_revoked=true", schema)
	})

	t.Run("ambiguous_same_name_schema_rejected", func(t *testing.T) {
		decoySchema := schema + "_decoy"
		aclRetain(t, decoySchema, migratorRole, applicationRole, "planned")
		if _, err := superuserPool.Exec(ctx,
			"CREATE SCHEMA "+pgx.Identifier{decoySchema}.Sanitize()+
				" AUTHORIZATION "+pgx.Identifier{migratorRole}.Sanitize()); err != nil {
			t.Fatalf("create decoy schema: %v", err)
		}
		// current_schemas(false) is privilege-aware: an inaccessible schema is
		// not part of the effective search_path. Grant USAGE so the decoy
		// schema models a historical schema the application can still reach.
		if _, err := superuserPool.Exec(ctx,
			"GRANT USAGE ON SCHEMA "+pgx.Identifier{decoySchema}.Sanitize()+
				" TO "+pgx.Identifier{applicationRole}.Sanitize()); err != nil {
			t.Fatalf("grant decoy schema USAGE: %v", err)
		}
		if _, err := migratorPool.Exec(ctx,
			"CREATE TABLE "+pgx.Identifier{decoySchema, "audit_events"}.Sanitize()+
				" (id integer)"); err != nil {
			t.Fatalf("create decoy audit_events: %v", err)
		}
		aclRetain(t, decoySchema, migratorRole, applicationRole, "created")

		ambiguousPool := aclSearchPathPool(
			t, ctx, applicationDSN, []string{schema, decoySchema},
		)
		err := verifyAppendOnlyApplicationPrivileges(ctx, ambiguousPool, appendOnlyTables)
		if err == nil {
			t.Fatal("append-only gate accepted two same-name relations on the effective search_path")
		}
		if !errors.Is(err, errAppendOnlyTableAmbiguous) ||
			!strings.Contains(err.Error(), "same-name relations on effective search_path=2") {
			t.Fatalf("ambiguity rejection = %v; want %v", err, errAppendOnlyTableAmbiguous)
		}
		t.Logf("schema=%s stage=ambiguous_same_name_schema_rejected decoy_schema=%s",
			schema, decoySchema)
	})
}

func aclRoleDSN(
	host string,
	port uint16,
	database string,
	role string,
	password string,
	schema string,
) string {
	endpoint := net.JoinHostPort(host, strconv.Itoa(int(port)))
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(role, password),
		Host:     endpoint,
		Path:     "/" + database,
		RawQuery: "search_path=" + url.QueryEscape(schema),
	}
	return dsn.String()
}

func aclQuoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func aclDSNPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid PostgreSQL configuration")
	}
	aclEndpoint(t, config.ConnConfig.Host, config.ConnConfig.Port)
	for _, fallback := range config.ConnConfig.Fallbacks {
		aclEndpoint(t, fallback.Host, fallback.Port)
	}
	config.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL pool failed")
	}
	t.Cleanup(pool.Close)
	return pool
}

func aclSuperuserSetRolePool(
	t *testing.T,
	ctx context.Context,
	dsn string,
	schema string,
	role string,
) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid PostgreSQL configuration")
	}
	aclEndpoint(t, config.ConnConfig.Host, config.ConnConfig.Port)
	for _, fallback := range config.ConnConfig.Fallbacks {
		aclEndpoint(t, fallback.Host, fallback.Port)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.MaxConns = 2
	statement := "SET ROLE " + pgx.Identifier{role}.Sanitize()
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, statement)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL pool failed")
	}
	t.Cleanup(pool.Close)
	return pool
}

func aclSearchPathPool(
	t *testing.T,
	ctx context.Context,
	dsn string,
	schemas []string,
) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid PostgreSQL configuration")
	}
	aclEndpoint(t, config.ConnConfig.Host, config.ConnConfig.Port)
	for _, fallback := range config.ConnConfig.Fallbacks {
		aclEndpoint(t, fallback.Host, fallback.Port)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	quoted := make([]string, len(schemas))
	for index, schema := range schemas {
		quoted[index] = pgx.Identifier{schema}.Sanitize()
	}
	config.ConnConfig.RuntimeParams["search_path"] = strings.Join(quoted, ", ")
	config.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL pool failed")
	}
	t.Cleanup(pool.Close)
	return pool
}

func aclEndpoint(t *testing.T, host string, port uint16) {
	t.Helper()
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() || port != 5433 {
		t.Fatal("requires literal loopback:5433 for primary and all fallbacks")
	}
}

func aclPreflight(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var host, user, database string
	var port, version int
	var superuser bool
	if err := pool.QueryRow(ctx, `SELECT host(inet_server_addr()), inet_server_port(),
		current_setting('server_version_num')::int, current_user, current_database(),
		(SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname = current_user)`).
		Scan(&host, &port, &version, &user, &database, &superuser); err != nil {
		t.Fatalf("preflight query: %v", err)
	}
	aclEndpoint(t, host, uint16(port))
	if version != 180006 {
		t.Fatalf("server_version_num = %d; want 180006", version)
	}
	if !superuser {
		t.Fatal("provisioning requires a superuser test connection")
	}
	t.Logf("preflight=passed loopback5433=true version=%d identity=%s database=%s superuser=true",
		version, user, database)
}

// aclRetain records planned and created resources so a later cleanup decision
// has an authoritative list; nothing is dropped here. Records are append-only
// within this round; prior rounds' registries are left untouched.
func aclRetain(t *testing.T, schema, migratorRole, applicationRole, status string) {
	t.Helper()
	directory := filepath.Join(
		"..", "..", ".spec", "wip", "article-management", "evidence-s1.5", "acl-r03",
	)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatalf("create evidence directory: %v", err)
	}
	line := fmt.Sprintf(
		`{"schema":%q,"migrator_role":%q,"application_role":%q,"status":%q,"test":%q,"recorded_at":%q}`+"\n",
		schema, migratorRole, applicationRole, status, t.Name(),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	file, err := os.OpenFile(
		filepath.Join(directory, "retained-resources.ndjson"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		t.Fatalf("open retained resource registry: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString(line); err != nil {
		t.Fatalf("append retained resource registry: %v", err)
	}
	t.Logf("retained_schema=%s retained_roles=%s,%s status=%s registry=retained-resources.ndjson",
		schema, migratorRole, applicationRole, status)
}

func aclRandomHex(t *testing.T) string {
	t.Helper()
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		t.Fatalf("random suffix: %v", err)
	}
	return hex.EncodeToString(buffer[:])
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/migration"
)

// migrationDatabaseDSNEnvironment names the dedicated migration identity. It
// must differ from the application identity so that the process cannot grant
// itself mutation rights on append-only tables.
const migrationDatabaseDSNEnvironment = "HAN_MIGRATION_DATABASE_DSN"

const (
	// appendOnlyMutationPrivileges is the single source for the privileges the
	// application identity must never hold on append-only relations.
	// deploy/db/grant-application.sql mirrors this list.
	appendOnlyMutationPrivileges = "UPDATE, DELETE, TRUNCATE"
	// migrationLedgerTable is internal migration state; the application
	// identity never needs DML on it.
	migrationLedgerTable = "schema_migrations"
)

var (
	errMigrationDatabaseDSNRequired = errors.New("migration database DSN is required")
	errSharedDatabaseIdentity       = errors.New(
		"migration and application database identities must differ",
	)
	errAppendOnlyTableUnresolved = errors.New(
		"append-only table is not uniquely visible",
	)
	errAppendOnlyTableAmbiguous = errors.New(
		"append-only table is ambiguous in the effective search_path",
	)
	errAppendOnlyPrivilege = errors.New(
		"append-only table privileges are not restricted",
	)
)

// appendOnlyTables lists the append-only relations whose application-side
// privileges are verified before the service starts listening.
var appendOnlyTables = []string{"audit_events", "processing_gate_records"}

var appendOnlyTableName = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

type privilegeQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type privilegeExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// productionMigration applies migrations through a dedicated migration
// identity, restores the append-only ACL, and then verifies that the
// application identity cannot mutate append-only tables. Both steps run before
// the service opens a listener.
func productionMigration(
	ctx context.Context,
	applicationPool *pgxpool.Pool,
	getenv func(string) string,
) error {
	return productionMigrationFrom(ctx, applicationPool, getenv, os.DirFS("."))
}

// productionMigrationFrom is the testable form of productionMigration; the
// filesystem root selects the migration source.
func productionMigrationFrom(
	ctx context.Context,
	applicationPool *pgxpool.Pool,
	getenv func(string) string,
	fsys fs.FS,
) error {
	if applicationPool == nil || applicationPool.Config() == nil ||
		applicationPool.Config().ConnConfig == nil {
		return errors.New("application database pool is required")
	}
	if fsys == nil {
		return errors.New("migration filesystem is required")
	}
	migrationConfig, err := migrationPoolConfig(applicationPool.Config(), getenv)
	if err != nil {
		return err
	}

	migrationPool, err := pgxpool.NewWithConfig(ctx, migrationConfig)
	if err != nil {
		return errors.New("create migration database pool")
	}
	if err := migrationPool.Ping(ctx); err != nil {
		migrationPool.Close()
		return fmt.Errorf("validate migration database connection: %w", err)
	}
	runner, err := migration.NewRunner(migrationPool, fsys, migrationsDirectory)
	if err != nil {
		migrationPool.Close()
		return fmt.Errorf("create migration runner: %w", err)
	}
	if err := runner.Run(ctx); err != nil {
		migrationPool.Close()
		return err
	}
	if err := restrictApplicationMutations(ctx, applicationPool, migrationPool); err != nil {
		migrationPool.Close()
		return err
	}
	// The migration identity is no longer needed once the ACL is restored; the
	// gate must run only after this pool is closed.
	migrationPool.Close()
	return verifyAppendOnlyApplicationPrivileges(ctx, applicationPool, appendOnlyTables)
}

type appendOnlyIdentity struct {
	currentUser  string
	sessionUser  string
	currentSuper bool
	sessionSuper bool
}

type appendOnlyPrivileges struct {
	visible       int64
	onSearchPath  int64
	ownerMember   bool
	insertTable   bool
	updateTable   bool
	deleteTable   bool
	truncateTable bool
	insertColumn  bool
	updateColumn  bool
	deleteColumn  bool
}

// verifyAppendOnlyApplicationPrivileges proves that the application connection
// is a stable, non-superuser identity that can only insert into append-only
// relations. Any role switch, owner membership, table privilege, or column
// privilege that could mutate the relations fails closed.
func verifyAppendOnlyApplicationPrivileges(
	ctx context.Context,
	queryer privilegeQueryer,
	tables []string,
) error {
	if queryer == nil {
		return errors.New("application privilege queryer is required")
	}
	if len(tables) == 0 {
		return errors.New("append-only tables are required")
	}
	identity, err := loadAppendOnlyIdentity(ctx, queryer)
	if err != nil {
		return err
	}
	if identity.currentUser != identity.sessionUser {
		return fmt.Errorf(
			"%w: current_user=%q session_user=%q role switching is not allowed",
			errAppendOnlyPrivilege, identity.currentUser, identity.sessionUser,
		)
	}
	if identity.currentSuper || identity.sessionSuper {
		return fmt.Errorf("%w: superuser session is not allowed", errAppendOnlyPrivilege)
	}
	for _, table := range tables {
		if !appendOnlyTableName.MatchString(table) {
			return fmt.Errorf(
				"%w: invalid append-only table name %q",
				errAppendOnlyTableUnresolved, table,
			)
		}
		privileges, err := inspectAppendOnlyTable(ctx, queryer, table)
		if err != nil {
			return err
		}
		if privileges.visible != 1 {
			return fmt.Errorf(
				"%w: table=%q visible relations=%d",
				errAppendOnlyTableUnresolved, table, privileges.visible,
			)
		}
		if privileges.onSearchPath != 1 {
			return fmt.Errorf(
				"%w: table=%q same-name relations on effective search_path=%d",
				errAppendOnlyTableAmbiguous, table, privileges.onSearchPath,
			)
		}
		if privileges.ownerMember {
			return fmt.Errorf(
				"%w: table=%q application role is owner or owner member",
				errAppendOnlyPrivilege, table,
			)
		}
		if !privileges.insertTable {
			return fmt.Errorf("%w: table=%q insert=false", errAppendOnlyPrivilege, table)
		}
		if privileges.updateTable {
			return fmt.Errorf("%w: table=%q update=true", errAppendOnlyPrivilege, table)
		}
		if privileges.deleteTable {
			return fmt.Errorf("%w: table=%q delete=true", errAppendOnlyPrivilege, table)
		}
		if privileges.truncateTable {
			return fmt.Errorf("%w: table=%q truncate=true", errAppendOnlyPrivilege, table)
		}
		if !privileges.insertColumn {
			return fmt.Errorf("%w: table=%q insert_column=false", errAppendOnlyPrivilege, table)
		}
		if privileges.updateColumn {
			return fmt.Errorf("%w: table=%q update_column=true", errAppendOnlyPrivilege, table)
		}
		if privileges.deleteColumn {
			return fmt.Errorf("%w: table=%q delete_column=true", errAppendOnlyPrivilege, table)
		}
	}
	return nil
}

func loadAppendOnlyIdentity(
	ctx context.Context,
	queryer privilegeQueryer,
) (appendOnlyIdentity, error) {
	const query = `
SELECT current_user::text, session_user::text,
       effective_role.rolsuper, session_role.rolsuper
FROM pg_catalog.pg_roles AS effective_role
JOIN pg_catalog.pg_roles AS session_role
  ON session_role.rolname = session_user
WHERE effective_role.rolname = current_user`
	var identity appendOnlyIdentity
	if err := queryer.QueryRow(ctx, query).Scan(
		&identity.currentUser,
		&identity.sessionUser,
		&identity.currentSuper,
		&identity.sessionSuper,
	); err != nil {
		return appendOnlyIdentity{}, fmt.Errorf("read application session identity: %w", err)
	}
	return identity, nil
}

func inspectAppendOnlyTable(
	ctx context.Context,
	queryer privilegeQueryer,
	table string,
) (appendOnlyPrivileges, error) {
	const query = `
WITH visible AS (
	SELECT relation.oid, relation.relowner
	FROM pg_catalog.pg_class AS relation
	WHERE relation.relname = $1
	  AND relation.relkind IN ('r', 'p')
	  AND pg_catalog.pg_table_is_visible(relation.oid)
),
on_search_path AS (
	SELECT relation.oid
	FROM pg_catalog.pg_class AS relation
	JOIN pg_catalog.pg_namespace AS namespace
	  ON namespace.oid = relation.relnamespace
	WHERE relation.relname = $1
	  AND relation.relkind IN ('r', 'p')
	  AND namespace.nspname = ANY (pg_catalog.current_schemas(false))
)
SELECT
	(SELECT count(*) FROM visible),
	(SELECT count(*) FROM on_search_path),
	coalesce((SELECT pg_catalog.pg_has_role(current_user::regrole, relowner, 'MEMBER')
		FROM visible LIMIT 1), false),
	coalesce((SELECT pg_catalog.has_table_privilege(oid, 'INSERT')
		FROM visible LIMIT 1), false),
	coalesce((SELECT pg_catalog.has_table_privilege(oid, 'UPDATE')
		FROM visible LIMIT 1), false),
	coalesce((SELECT pg_catalog.has_table_privilege(oid, 'DELETE')
		FROM visible LIMIT 1), false),
	coalesce((SELECT pg_catalog.has_table_privilege(oid, 'TRUNCATE')
		FROM visible LIMIT 1), false),
	coalesce((SELECT pg_catalog.has_any_column_privilege(oid, 'INSERT')
		FROM visible LIMIT 1), false),
	coalesce((SELECT pg_catalog.has_any_column_privilege(oid, 'UPDATE')
		FROM visible LIMIT 1), false),
	coalesce((
		SELECT EXISTS (
			SELECT 1
			FROM pg_catalog.pg_attribute AS attribute
			CROSS JOIN LATERAL pg_catalog.aclexplode(attribute.attacl) AS column_acl
			WHERE attribute.attrelid = visible.oid
			  AND attribute.attnum > 0
			  AND NOT attribute.attisdropped
			  AND column_acl.privilege_type = 'DELETE'
		)
		FROM visible
		LIMIT 1
	), false)`
	var privileges appendOnlyPrivileges
	if err := queryer.QueryRow(ctx, query, table).Scan(
		&privileges.visible,
		&privileges.onSearchPath,
		&privileges.ownerMember,
		&privileges.insertTable,
		&privileges.updateTable,
		&privileges.deleteTable,
		&privileges.truncateTable,
		&privileges.insertColumn,
		&privileges.updateColumn,
		&privileges.deleteColumn,
	); err != nil {
		return appendOnlyPrivileges{}, fmt.Errorf(
			"inspect append-only table %q: %w", table, err,
		)
	}
	return privileges, nil
}

// restrictApplicationMutations restores the final append-only ACL after the
// migration identity has created or updated relations. It runs with the
// migration pool before that pool is closed, so a fresh install reaches the
// same ACL that deploy/db/grant-application.sql documents.
func restrictApplicationMutations(
	ctx context.Context,
	applicationPool *pgxpool.Pool,
	migrationPool *pgxpool.Pool,
) error {
	if applicationPool == nil || migrationPool == nil {
		return errors.New("application and migration database pools are required")
	}
	var applicationRole string
	if err := applicationPool.QueryRow(ctx, `SELECT current_user::text`).
		Scan(&applicationRole); err != nil {
		return fmt.Errorf("read application database role: %w", err)
	}
	if applicationRole == "" {
		return errors.New("application database role is required")
	}
	relationRole := pgx.Identifier{applicationRole}.Sanitize()
	for _, table := range appendOnlyTables {
		schema, err := resolveVisibleTableSchema(ctx, applicationPool, table)
		if err != nil {
			return err
		}
		statement := "REVOKE " + appendOnlyMutationPrivileges + " ON TABLE " +
			pgx.Identifier{schema, table}.Sanitize() + " FROM " + relationRole
		if _, err := migrationPool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("restore application privileges on %q: %w", table, err)
		}
	}
	schema, err := resolveVisibleTableSchema(ctx, applicationPool, migrationLedgerTable)
	if err != nil {
		return err
	}
	statement := "REVOKE ALL ON TABLE " +
		pgx.Identifier{schema, migrationLedgerTable}.Sanitize() + " FROM " + relationRole
	if _, err := migrationPool.Exec(ctx, statement); err != nil {
		return fmt.Errorf("restore application privileges on %q: %w", migrationLedgerTable, err)
	}
	return nil
}

func resolveVisibleTableSchema(
	ctx context.Context,
	queryer privilegeQueryer,
	table string,
) (string, error) {
	if !appendOnlyTableName.MatchString(table) {
		return "", fmt.Errorf(
			"%w: invalid append-only table name %q",
			errAppendOnlyTableUnresolved, table,
		)
	}
	const query = `
WITH visible_relation AS (
	SELECT relation.oid, relation.relnamespace
	FROM pg_catalog.pg_class AS relation
	WHERE relation.relname = $1
	  AND relation.relkind IN ('r', 'p')
	  AND pg_catalog.pg_table_is_visible(relation.oid)
),
on_search_path AS (
	SELECT relation.oid
	FROM pg_catalog.pg_class AS relation
	JOIN pg_catalog.pg_namespace AS namespace
	  ON namespace.oid = relation.relnamespace
	WHERE relation.relname = $1
	  AND relation.relkind IN ('r', 'p')
	  AND namespace.nspname = ANY (pg_catalog.current_schemas(false))
)
SELECT
	(SELECT count(*) FROM visible_relation),
	(SELECT count(*) FROM on_search_path),
	(SELECT coalesce(min(namespace.nspname), '')
	 FROM visible_relation
	 JOIN pg_catalog.pg_namespace AS namespace
	   ON namespace.oid = visible_relation.relnamespace)`
	var count, onSearchPath int64
	var schema string
	if err := queryer.QueryRow(ctx, query, table).Scan(
		&count, &onSearchPath, &schema,
	); err != nil {
		return "", fmt.Errorf("resolve append-only table %q: %w", table, err)
	}
	if count != 1 {
		return "", fmt.Errorf(
			"%w: table=%q visible relations=%d",
			errAppendOnlyTableUnresolved, table, count,
		)
	}
	if onSearchPath != 1 {
		return "", fmt.Errorf(
			"%w: table=%q same-name relations on effective search_path=%d",
			errAppendOnlyTableAmbiguous, table, onSearchPath,
		)
	}
	return schema, nil
}

// validateProductionMigrationConfiguration performs every identity check
// without opening a connection, so startup can fail closed before the
// application pool is created.
func validateProductionMigrationConfiguration(
	application *pgxpool.Config,
	getenv func(string) string,
) error {
	_, err := migrationPoolConfig(application, getenv)
	return err
}

func migrationPoolConfig(
	application *pgxpool.Config,
	getenv func(string) string,
) (*pgxpool.Config, error) {
	if application == nil || application.ConnConfig == nil {
		return nil, errors.New("application database configuration is required")
	}
	if getenv == nil {
		return nil, errors.New("environment reader is required")
	}
	dsn, err := migrationDatabaseDSN(getenv)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(getenv(databaseDSNEnvironment)) == dsn {
		return nil, errSharedDatabaseIdentity
	}
	config, err := parseDatabasePoolConfig(dsn, getenv)
	if err != nil {
		return nil, err
	}
	if err := rejectSharedDatabaseIdentity(application.ConnConfig, config.ConnConfig); err != nil {
		return nil, err
	}
	return config, nil
}

func migrationDatabaseDSN(getenv func(string) string) (string, error) {
	if getenv == nil {
		return "", errors.New("environment reader is required")
	}
	dsn := strings.TrimSpace(getenv(migrationDatabaseDSNEnvironment))
	if dsn == "" {
		return "", errMigrationDatabaseDSNRequired
	}
	return dsn, nil
}

type databaseIdentity struct {
	host     string
	port     uint16
	database string
	user     string
}

func databaseIdentityOf(config *pgx.ConnConfig, host string, port uint16) databaseIdentity {
	return databaseIdentity{
		host:     normalizeDatabaseHost(host),
		port:     port,
		database: config.Database,
		user:     config.User,
	}
}

func databaseEndpointsOf(config *pgx.ConnConfig) ([]databaseIdentity, error) {
	if config == nil {
		return nil, errors.New("database configuration is required")
	}
	endpoints := []databaseIdentity{
		databaseIdentityOf(config, config.Host, config.Port),
	}
	for _, fallback := range config.Fallbacks {
		if fallback == nil {
			return nil, errors.New("database fallback endpoint is required")
		}
		endpoints = append(endpoints, databaseIdentityOf(config, fallback.Host, fallback.Port))
	}
	return endpoints, nil
}

func normalizeDatabaseHost(host string) string {
	host = strings.TrimSpace(host)
	if strings.HasPrefix(host, "/") {
		return host
	}
	return strings.ToLower(host)
}

func rejectSharedDatabaseIdentity(application, migration *pgx.ConnConfig) error {
	if application == nil || migration == nil {
		return errors.New("application and migration database configurations are required")
	}
	if application.User == migration.User {
		return fmt.Errorf("%w: application and migration role names are equal", errSharedDatabaseIdentity)
	}
	applicationEndpoints, err := databaseEndpointsOf(application)
	if err != nil {
		return fmt.Errorf("resolve application database endpoints: %w", err)
	}
	migrationEndpoints, err := databaseEndpointsOf(migration)
	if err != nil {
		return fmt.Errorf("resolve migration database endpoints: %w", err)
	}
	applicationSet := make(map[databaseIdentity]struct{}, len(applicationEndpoints))
	for _, endpoint := range applicationEndpoints {
		applicationSet[endpoint] = struct{}{}
	}
	for _, endpoint := range migrationEndpoints {
		if _, shared := applicationSet[endpoint]; shared {
			return fmt.Errorf(
				"%w: migration and application share host=%q port=%d database=%q user=%q",
				errSharedDatabaseIdentity,
				endpoint.host,
				endpoint.port,
				endpoint.database,
				endpoint.user,
			)
		}
	}
	return nil
}

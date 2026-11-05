// Package migration applies the project's versioned SQL migrations.
package migration

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	advisoryLockKey        int64 = 0x48414e4d494752
	resourceCleanupTimeout       = 5 * time.Second
)

var migrationNamePattern = regexp.MustCompile(`^([0-9]{4})_([a-z0-9]+(?:_[a-z0-9]+)*)\.sql$`)

const createSchemaMigrationsSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version integer PRIMARY KEY,
	name text NOT NULL,
	applied_at timestamp with time zone NOT NULL DEFAULT now()
)`

// Migration is one versioned SQL migration.
type Migration struct {
	Version int
	Name    string
	Path    string
	SQL     string
}

// Load discovers migrations in directory.
func Load(fsys fs.FS, directory string) ([]Migration, error) {
	if fsys == nil {
		return nil, errors.New("migration filesystem is required")
	}
	if directory == "" || !fs.ValidPath(directory) {
		return nil, fmt.Errorf("invalid migration directory %q", directory)
	}

	entries, err := fs.ReadDir(fsys, directory)
	if err != nil {
		return nil, fmt.Errorf("read migration directory %q: %w", directory, err)
	}

	migrations := make([]Migration, 0, len(entries))
	versions := make(map[int]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		migration, err := loadMigration(fsys, directory, entry.Name())
		if err != nil {
			return nil, err
		}
		if previousPath, exists := versions[migration.Version]; exists {
			return nil, fmt.Errorf(
				"duplicate migration version %04d in %q and %q",
				migration.Version,
				previousPath,
				migration.Path,
			)
		}

		versions[migration.Version] = migration.Path
		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

// Runner applies migrations to PostgreSQL.
type Runner struct {
	pool      *pgxpool.Pool
	fsys      fs.FS
	directory string
}

// NewRunner creates a migration runner.
func NewRunner(pool *pgxpool.Pool, fsys fs.FS, directory string) (*Runner, error) {
	if pool == nil {
		return nil, errors.New("migration PostgreSQL pool is required")
	}
	if fsys == nil {
		return nil, errors.New("migration filesystem is required")
	}
	if directory == "" || !fs.ValidPath(directory) {
		return nil, fmt.Errorf("invalid migration directory %q", directory)
	}
	return &Runner{pool: pool, fsys: fsys, directory: directory}, nil
}

// Run applies all pending migrations.
func (r *Runner) Run(ctx context.Context) (err error) {
	migrations, err := Load(r.fsys, r.directory)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return errors.Join(
			fmt.Errorf("acquire migration advisory lock: %w", err),
			closeAcquiredConnection(conn),
		)
	}
	defer func() {
		err = errors.Join(err, unlockAndClose(conn))
	}()

	if _, err := conn.Exec(ctx, createSchemaMigrationsSQL); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	applied, err := loadAppliedVersions(ctx, conn)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if _, exists := applied[migration.Version]; exists {
			continue
		}
		if err := applyMigration(ctx, conn, migration); err != nil {
			return err
		}
	}
	return nil
}

func loadMigration(fsys fs.FS, directory, filename string) (Migration, error) {
	migrationPath := path.Join(directory, filename)
	matches := migrationNamePattern.FindStringSubmatch(filename)
	if matches == nil {
		return Migration{}, fmt.Errorf("invalid migration filename %q", migrationPath)
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return Migration{}, fmt.Errorf("parse migration version in %q: %w", migrationPath, err)
	}
	sqlBytes, err := fs.ReadFile(fsys, migrationPath)
	if err != nil {
		return Migration{}, fmt.Errorf("read migration %q: %w", migrationPath, err)
	}
	if err := rejectTransactionControl(string(sqlBytes)); err != nil {
		return Migration{}, fmt.Errorf("validate migration %q: %w", migrationPath, err)
	}

	return Migration{
		Version: version,
		Name:    matches[2],
		Path:    migrationPath,
		SQL:     string(sqlBytes),
	}, nil
}

func loadAppliedVersions(ctx context.Context, conn *pgxpool.Conn) (map[int]struct{}, error) {
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query applied migration versions: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]struct{})
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied migration version: %w", err)
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migration versions: %w", err)
	}
	return applied, nil
}

func applyMigration(ctx context.Context, conn *pgxpool.Conn, migration Migration) (err error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %04d transaction: %w", migration.Version, err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resourceCleanupTimeout)
		defer cancel()
		rollbackErr := tx.Rollback(cleanupCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback migration %04d: %w", migration.Version, rollbackErr))
		}
	}()

	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("execute migration %04d %q: %w", migration.Version, migration.Name, err)
	}
	if _, err := tx.Exec(
		ctx,
		"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
		migration.Version,
		migration.Name,
	); err != nil {
		return fmt.Errorf("record migration %04d %q: %w", migration.Version, migration.Name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %04d %q: %w", migration.Version, migration.Name, err)
	}

	committed = true
	return nil
}

func unlockAndClose(conn *pgxpool.Conn) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), resourceCleanupTimeout)
	defer cancel()

	var cleanupErr error
	var unlocked bool
	if err := conn.QueryRow(
		cleanupCtx,
		"SELECT pg_advisory_unlock($1)",
		advisoryLockKey,
	).Scan(&unlocked); err != nil {
		cleanupErr = fmt.Errorf("release migration advisory lock: %w", err)
	} else if !unlocked {
		cleanupErr = errors.New("migration advisory lock was not held during release")
	}

	return errors.Join(cleanupErr, closeAcquiredConnectionWithContext(cleanupCtx, conn))
}

func closeAcquiredConnection(conn *pgxpool.Conn) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), resourceCleanupTimeout)
	defer cancel()
	return closeAcquiredConnectionWithContext(cleanupCtx, conn)
}

func closeAcquiredConnectionWithContext(ctx context.Context, conn *pgxpool.Conn) error {
	err := conn.Hijack().Close(ctx)
	if err == nil {
		return nil
	}
	return fmt.Errorf("close migration connection: %w", err)
}

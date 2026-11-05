//go:build integration

// Package identity integration fixtures drive the real identity production
// stores against PostgreSQL 18. Every test owns an isolated schema that is
// created and dropped in its own committed transaction, so no fixture data
// leaks between cases or into another database.
package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	identitypkg "han/internal/identity"
	"han/internal/platform"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlStateUniqueViolation         = "23505"
	sqlStateForeignKeyViolation     = "23503"
	sqlStateCheckViolation          = "23514"
	sqlStateNotNullViolation        = "23502"
	sqlStateObjectStatePrerequisite = "55000"
)

// migrationFiles is the full forward chain. Order is authoritative: the
// password-reset durable migration (0008) depends on 0004/0007, the session
// writer adds auth_generation/lineage columns only from 0008, 0010 adds the
// session recovery scope, and 0011 adds replay publication metadata.
var migrationFiles = []string{
	"0001_shared.sql",
	"0002_identity.sql",
	"0003_article.sql",
	"0004_idempotency.sql",
	"0005_recovery_runs.sql",
	"0006_user_governance_indexes.sql",
	"0007_idempotency_response_snapshot_v1.sql",
	"0008_password_reset_durable.sql",
	"0009_article_replay_attempt.sql",
	"0010_recovery_runtime_metadata.sql",
	"0011_article_replay_publication.sql",
}

// schemaHarness is one isolated, migrated PostgreSQL schema.
type schemaHarness struct {
	pool   *pgxpool.Pool
	schema string
	quoted string
}

// newSchemaHarness creates a fresh schema, applies the full migration chain and
// registers cleanup that drops it with CASCADE.
func newSchemaHarness(t *testing.T) *schemaHarness {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	dsn := testenv.RequirePostgresDSN(t)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("configure PostgreSQL identity integration database")
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL identity integration database")
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("connect PostgreSQL identity integration database")
	}
	requirePostgres18(t, ctx, pool)

	schema := "identity_s16_" + randomHex(t, 8)
	quoted := pgx.Identifier{schema}.Sanitize()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			20*time.Second,
		)
		defer cleanupCancel()
		if _, err := pool.Exec(
			cleanupCtx,
			"DROP SCHEMA IF EXISTS "+quoted+" CASCADE",
		); err != nil {
			t.Errorf("drop identity integration schema: %v", err)
		}
	})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin identity integration schema transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("create identity integration schema: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+quoted); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("set identity integration search path: %v", err)
	}
	for _, file := range migrationFiles {
		sql := readMigrationFile(t, file)
		if _, err := tx.Exec(ctx, sql); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit identity integration schema: %v", err)
	}
	return &schemaHarness{pool: pool, schema: schema, quoted: quoted}
}

// begin opens a caller-owned transaction bound to the isolated schema.
func (h *schemaHarness) begin(t *testing.T, ctx context.Context) pgx.Tx {
	t.Helper()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin identity integration transaction: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+h.quoted); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("set identity integration transaction search path: %v", err)
	}
	return tx
}

// exec runs one statement inside a short committed transaction.
func (h *schemaHarness) exec(t *testing.T, ctx context.Context, sql string, args ...any) {
	t.Helper()
	tx := h.begin(t, ctx)
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("execute identity integration statement: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit identity integration statement: %v", err)
	}
}

// queryRow runs one read-only statement inside a committed transaction.
func (h *schemaHarness) queryRow(
	t *testing.T,
	ctx context.Context,
	sql string,
	args ...any,
) pgx.Row {
	t.Helper()
	tx := h.begin(t, ctx)
	row := tx.QueryRow(ctx, sql, args...)
	return &committedRow{row: row, tx: tx, ctx: ctx, t: t}
}

// committedRow releases its transaction once the single row is scanned.
type committedRow struct {
	row pgx.Row
	tx  pgx.Tx
	ctx context.Context
	t   *testing.T
}

func (r *committedRow) Scan(destinations ...any) error {
	r.t.Helper()
	err := r.row.Scan(destinations...)
	if commitErr := r.tx.Commit(r.ctx); commitErr != nil {
		return fmt.Errorf("commit identity integration read: %w", commitErr)
	}
	return err
}

// fixtureIDs are the synthetic identities every suite case references.
type fixtureIDs struct {
	member            string
	otherMember       string
	initialAdmin      string
	otherAdmin        string
	deletedMember     string
	now               time.Time
	initialDigestText string
}

// seedFixtures creates one member, a second member, the initial administrator,
// another administrator and a deleted member tombstone.
func seedFixtures(t *testing.T, h *schemaHarness, ctx context.Context) fixtureIDs {
	t.Helper()
	ids := fixtureIDs{
		member:            randomUUID(t),
		otherMember:       randomUUID(t),
		initialAdmin:      randomUUID(t),
		otherAdmin:        randomUUID(t),
		deletedMember:     randomUUID(t),
		now:               time.Date(2026, time.October, 27, 8, 0, 0, 0, time.UTC),
		initialDigestText: "synthetic-initial-password-digest",
	}
	tx := h.begin(t, ctx)
	for _, seed := range []struct {
		id        string
		username  string
		tombstone bool
	}{
		{id: ids.member, username: "s16-member"},
		{id: ids.otherMember, username: "s16-other-member"},
		{id: ids.initialAdmin, username: "s16-initial-admin"},
		{id: ids.otherAdmin, username: "s16-other-admin"},
		{id: ids.deletedMember, username: "s16-deleted-member", tombstone: true},
	} {
		email := seed.username + "@example.test"
		phone := "+1555" + strings.ReplaceAll(seed.id, "-", "")[:8]
		if seed.tombstone {
			if _, err := tx.Exec(ctx, `
				INSERT INTO users (
					id, username, email, phone, role, status, credential_mode,
					retained_email, created_at, updated_at, deleted_at
				)
				VALUES ($1, $2, NULL, NULL, 'member', 'deleted', 'permanent',
					$3, $4, $4, $4)
			`, seed.id, seed.username, email, ids.now); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatalf("insert fixture tombstone %s: %v", seed.username, err)
			}
		} else if _, err := tx.Exec(ctx, `
			INSERT INTO users (
				id, username, email, phone, role, status, credential_mode,
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, 'member', 'active', 'permanent', $5, $5)
		`,
			seed.id,
			seed.username,
			email,
			phone,
			ids.now,
		); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("insert fixture user %s: %v", seed.username, err)
		}
		lifecycle := "active"
		if seed.tombstone {
			lifecycle = "reserved"
		}
		for _, identifier := range []struct {
			kind  string
			value string
		}{
			{"username", seed.username},
			{"email", email},
			{"phone", phone},
		} {
			if _, err := tx.Exec(ctx, `
				INSERT INTO identity_identifiers (
					user_id, kind, normalized_value, lifecycle, created_at
				)
				VALUES ($1, $2, $3, $4, $5)
			`,
				seed.id,
				identifier.kind,
				identifier.value,
				lifecycle,
				ids.now,
			); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatalf(
					"insert fixture %s identifier %s: %v",
					identifier.kind, seed.username, err,
				)
			}
		}
	}
	substitutions := map[string]string{
		":member":         quoteLiteral(ids.member),
		":other_member":   quoteLiteral(ids.otherMember),
		":initial_admin":  quoteLiteral(ids.initialAdmin),
		":other_admin":    quoteLiteral(ids.otherAdmin),
		":deleted_member": quoteLiteral(ids.deletedMember),
		":initial_digest": quoteLiteral(ids.initialDigestText),
		":now":            quoteLiteral(ids.now.UTC().Format(time.RFC3339Nano)),
	}
	for _, statement := range splitSQLStatements(readFixtureFile(t, "fixtures.sql")) {
		for token, value := range substitutions {
			statement = strings.ReplaceAll(statement, token, value)
		}
		if _, err := tx.Exec(ctx, statement); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply identity integration fixtures: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit identity integration fixtures: %v", err)
	}
	return ids
}

// newGateStore creates the single production PasswordResetStore under test.
func newGateStore() *identitypkg.PasswordResetStore {
	return &identitypkg.PasswordResetStore{}
}

// newSessionStore creates the only Gate-aware session store.
func newSessionStore(
	t *testing.T,
	h *schemaHarness,
	gate *identitypkg.PasswordResetStore,
) session.Store {
	t.Helper()
	store, err := session.NewStore(session.StoreDependencies{
		Transactions:      newSchemaTransactor(h.pool, h.quoted),
		PasswordResetGate: gate,
	})
	if err != nil {
		t.Fatalf("session.NewStore() error = %v", err)
	}
	return store
}

func requirePostgres18(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	var version int
	if err := pool.QueryRow(
		ctx,
		"SELECT current_setting('server_version_num')::int / 10000",
	).Scan(&version); err != nil {
		t.Fatalf("read PostgreSQL server version: %v", err)
	}
	if version != 18 {
		t.Fatalf("PostgreSQL major version = %d, want 18", version)
	}
}

func readMigrationFile(t *testing.T, name string) string {
	t.Helper()
	return readFixtureFile(t, "../../../migrations/"+name)
}

func readFixtureFile(t *testing.T, relative string) string {
	t.Helper()
	content, err := os.ReadFile(relative)
	if err != nil {
		t.Fatalf("read fixture %s: %v", relative, err)
	}
	return string(content)
}

func randomUUID(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("read random UUID bytes: %v", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	)
}

func randomHex(t *testing.T, size int) string {
	t.Helper()
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("read random bytes: %v", err)
	}
	return hex.EncodeToString(raw)
}

// sqlState extracts the PostgreSQL SQLSTATE from an error, if present.
func sqlState(err error) string {
	var pgErr interface{ SQLState() string }
	if !asError(err, &pgErr) {
		return ""
	}
	return pgErr.SQLState()
}

func asError(err error, target *interface{ SQLState() string }) bool {
	for current := err; current != nil; {
		if candidate, ok := current.(interface{ SQLState() string }); ok {
			*target = candidate
			return true
		}
		unwrapper, ok := current.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		next := unwrapper.Unwrap()
		if next == current {
			return false
		}
		current = next
	}
	return false
}

// mustAtoi converts a fixture counter without panicking in tests.
func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		t.Fatalf("parse fixture integer %q: %v", value, err)
	}
	return parsed
}

// splitSQLStatements splits one fixture file into individually executable
// statements. The fixtures never contain semicolons inside string literals, so
// splitting on statement terminators is exact for this file.
func splitSQLStatements(script string) []string {
	statements := make([]string, 0, 16)
	var current strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			statement := strings.TrimSpace(current.String())
			statements = append(statements, strings.TrimSuffix(statement, ";"))
			current.Reset()
		}
	}
	if remainder := strings.TrimSpace(current.String()); remainder != "" {
		statements = append(statements, remainder)
	}
	return statements
}

// quoteLiteral renders one trusted fixture value as a SQL string literal. Only
// harness-generated UUIDs, fixed digests and RFC3339 timestamps reach it, and
// single quotes are doubled defensively.
func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// TestS16SchemaUniquenessAndConstraints proves the identity invariants are
// enforced by the real schema rather than by application hope.
func TestS16SchemaUniquenessAndConstraints(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	t.Run("one initial administrator", func(t *testing.T) {
		tx := harness.begin(t, ctx)
		defer func() { _ = tx.Rollback(ctx) }()
		_, err := tx.Exec(ctx, `
			INSERT INTO users (
				id, username, email, role, status, credential_mode,
				is_initial_admin, created_at, updated_at
			)
			VALUES ($1, $2, $3, 'admin', 'active', 'permanent', true, $4, $4)
		`, randomUUID(t), "s16-second-initial", "second-initial@example.test", ids.now)
		if err == nil {
			t.Fatal("second initial administrator insert succeeded, want uniqueness rejection")
		}
		if state := sqlState(err); state != sqlStateUniqueViolation {
			t.Fatalf("SQLSTATE = %q, want %q (err=%v)", state, sqlStateUniqueViolation, err)
		}
	})

	t.Run("identifier uniqueness across kinds", func(t *testing.T) {
		tx := harness.begin(t, ctx)
		defer func() { _ = tx.Rollback(ctx) }()
		_, err := tx.Exec(ctx, `
			INSERT INTO identity_identifiers (
				user_id, kind, normalized_value, lifecycle, created_at
			)
			VALUES ($1, 'username', 's16-member', 'active', $2)
		`, ids.otherMember, ids.now)
		if state := sqlState(err); state != sqlStateUniqueViolation {
			t.Fatalf("SQLSTATE = %q, want %q (err=%v)", state, sqlStateUniqueViolation, err)
		}
	})

	t.Run("illegal user enumerations rejected", func(t *testing.T) {
		for _, mutation := range []struct {
			name  string
			query string
		}{
			{"role", "UPDATE users SET role = 'owner' WHERE id = $1"},
			{"status", "UPDATE users SET status = 'archived' WHERE id = $1"},
			{"credential_mode", "UPDATE users SET credential_mode = 'plain' WHERE id = $1"},
		} {
			tx := harness.begin(t, ctx)
			_, err := tx.Exec(ctx, mutation.query, ids.member)
			_ = tx.Rollback(ctx)
			if state := sqlState(err); state != sqlStateCheckViolation {
				t.Fatalf(
					"%s SQLSTATE = %q, want %q (err=%v)",
					mutation.name, state, sqlStateCheckViolation, err,
				)
			}
		}
	})

	t.Run("deleted tombstone erases credential", func(t *testing.T) {
		tx := harness.begin(t, ctx)
		_, err := tx.Exec(ctx, `
			UPDATE users
			SET status = 'deleted',
				permanent_password_digest = 'synthetic-leftover-digest'
			WHERE id = $1
		`, ids.otherMember)
		_ = tx.Rollback(ctx)
		if err == nil {
			t.Fatal("deleted user retained a permanent digest, want rejection")
		}
		if state := sqlState(err); state != sqlStateCheckViolation {
			t.Fatalf("SQLSTATE = %q, want %q (err=%v)", state, sqlStateCheckViolation, err)
		}
	})

	t.Run("session credential binding", func(t *testing.T) {
		tx := harness.begin(t, ctx)
		_, err := tx.Exec(ctx, `
			INSERT INTO sessions (
				id, user_id, token_digest, csrf_token_digest,
				authenticated_credential_kind, authenticated_credential_id,
				auth_generation, issued_at, last_seen_at,
				absolute_expires_at, idle_expires_at
			)
			VALUES ($1, $2, $3, $4, 'permanent', $5, 0, $6, $6, $7, $7)
		`,
			randomUUID(t), ids.member, []byte("synthetic-token"),
			[]byte("synthetic-csrf"), randomUUID(t), ids.now, ids.now.Add(time.Hour),
		)
		_ = tx.Rollback(ctx)
		if state := sqlState(err); state != sqlStateCheckViolation {
			t.Fatalf("SQLSTATE = %q, want %q (err=%v)", state, sqlStateCheckViolation, err)
		}
	})

	t.Run("unknown user foreign keys rejected", func(t *testing.T) {
		for _, mutation := range []struct {
			name  string
			query string
			args  []any
		}{
			{
				name: "identifier",
				query: `INSERT INTO identity_identifiers (
					user_id, kind, normalized_value, lifecycle, created_at
				) VALUES ($1, 'username', $2, 'active', $3)`,
				args: []any{randomUUID(t), "s16-orphan-" + randomHex(t, 4), ids.now},
			},
			{
				name: "session",
				query: `INSERT INTO sessions (
					id, user_id, token_digest, csrf_token_digest,
					authenticated_credential_kind, auth_generation,
					issued_at, last_seen_at, absolute_expires_at, idle_expires_at
				) VALUES ($1, $2, $3, $4, 'permanent', 0, $5, $5, $6, $6)`,
				args: []any{
					randomUUID(t), randomUUID(t), []byte("synthetic-token-2"),
					[]byte("synthetic-csrf-2"), ids.now, ids.now.Add(time.Hour),
				},
			},
		} {
			tx := harness.begin(t, ctx)
			_, err := tx.Exec(ctx, mutation.query, mutation.args...)
			_ = tx.Rollback(ctx)
			if state := sqlState(err); state != sqlStateForeignKeyViolation {
				t.Fatalf(
					"%s SQLSTATE = %q, want %q (err=%v)",
					mutation.name, state, sqlStateForeignKeyViolation, err,
				)
			}
		}
	})
}

// TestS16UserFirstLockOrderingAndSessionRevocation proves the User row is the
// single lock anchor and that a suspension revokes every session in one
// transaction.
func TestS16UserFirstLockOrderingAndSessionRevocation(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	gate := newGateStore()
	sessions := newSessionStore(t, harness, gate)
	for index := 0; index < 3; index++ {
		tx := harness.begin(t, ctx)
		if _, err := sessions.Create(ctx, tx, session.CreateInput{
			ID:                          randomUUID(t),
			UserID:                      ids.member,
			Token:                       "s16-member-token-" + randomHex(t, 4),
			CSRFToken:                   "s16-member-csrf",
			AuthenticatedCredentialKind: session.CredentialPermanent,
			IssuedAt:                    ids.now,
			AbsoluteExpiresAt:           ids.now.Add(8 * time.Hour),
			IdleExpiresAt:               ids.now.Add(time.Hour),
		}); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("create fixture session %d: %v", index, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit fixture session %d: %v", index, err)
		}
	}

	coordinator, err := identitypkg.NewAccountStatusService(
		identitypkg.AccountStatusDependencies{
			Transactions: &schemaTransactor{pool: harness.pool, quoted: harness.quoted},
			Repository:   identitypkg.PostgresAccountStatusRepository{},
			Audit:        auditStore{},
			IDs:          &uuidSequence{},
			Clock:        fixedClock{now: ids.now.Add(time.Minute)},
		},
	)
	if err != nil {
		t.Fatalf("NewAccountStatusService() error = %v", err)
	}
	principal := session.Principal{
		UserID: ids.initialAdmin, Role: "admin",
		IsInitialAdmin: true, Status: "active",
	}
	result, err := coordinator.SetAccountStatus(ctx, identitypkg.AccountStatusRequest{
		Actor: principal, TargetID: ids.member, TargetRole: identitypkg.RoleMember,
		Status: identitypkg.UserStatusSuspended, TraceID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	if err != nil {
		t.Fatalf("SetAccountStatus() error = %v", err)
	}
	if !result.Changed || result.User.Status != identitypkg.UserStatusSuspended {
		t.Fatalf("SetAccountStatus() result = %#v", result)
	}

	var status identitypkg.UserStatus
	var activeSessions, revokedSessions int
	if err := harness.queryRow(t, ctx, `
		SELECT u.status,
			count(*) FILTER (WHERE s.revoked_at IS NULL),
			count(*) FILTER (WHERE s.revoked_at IS NOT NULL)
		FROM users AS u
		JOIN sessions AS s ON s.user_id = u.id
		WHERE u.id = $1
		GROUP BY u.status
	`, ids.member).Scan(&status, &activeSessions, &revokedSessions); err != nil {
		t.Fatalf("read post-suspension state: %v", err)
	}
	if status != identitypkg.UserStatusSuspended {
		t.Fatalf("persisted status = %q, want suspended", status)
	}
	if activeSessions != 0 || revokedSessions != 3 {
		t.Fatalf(
			"session state active/revoked = %d/%d, want 0/3",
			activeSessions, revokedSessions,
		)
	}

	// Repeating the target state must stay idempotent and write no new revoke.
	repeat, err := coordinator.SetAccountStatus(ctx, identitypkg.AccountStatusRequest{
		Actor: principal, TargetID: ids.member, TargetRole: identitypkg.RoleMember,
		Status: identitypkg.UserStatusSuspended, TraceID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	if err != nil {
		t.Fatalf("repeat SetAccountStatus() error = %v", err)
	}
	if repeat.Changed {
		t.Fatalf("repeat SetAccountStatus() changed = true, want idempotent no-op")
	}
}

// TestS16GovernanceRaceLosesSingleWriter proves the old-state predicate makes
// exactly one concurrent governance write win.
func TestS16GovernanceRaceLosesSingleWriter(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	repository := identitypkg.PostgresAccountStatusRepository{}
	// The first writer holds the row lock and changes the state.
	first := harness.begin(t, ctx)
	firstClosed := false
	t.Cleanup(func() {
		if !firstClosed {
			_ = first.Rollback(context.Background())
		}
	})

	locked, err := repository.LockUser(ctx, first, ids.member)
	if err != nil {
		t.Fatalf("first LockUser() error = %v", err)
	}
	if locked.Status != identitypkg.UserStatusActive {
		t.Fatalf("first locked status = %q, want active", locked.Status)
	}
	if err := repository.UpdateStatus(
		ctx, first, ids.member,
		identitypkg.UserStatusActive, identitypkg.UserStatusSuspended,
		ids.now.Add(time.Minute),
	); err != nil {
		t.Fatalf("first UpdateStatus() error = %v", err)
	}

	// The stale second writer still carries the old expectation. A bounded
	// lock_timeout keeps the contention deterministic instead of hanging until
	// the outer context deadline.
	second := harness.begin(t, ctx)
	if _, err := second.Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
		_ = second.Rollback(ctx)
		t.Fatalf("set second lock_timeout: %v", err)
	}
	_, lockErr := repository.LockUser(ctx, second, ids.member)
	_ = second.Rollback(ctx)
	if state := sqlState(lockErr); state != "55P03" {
		t.Fatalf("SQLSTATE = %q, want lock_not_available (err=%v)", state, lockErr)
	}
	if err := first.Commit(ctx); err != nil {
		t.Fatalf("commit first governance write: %v", err)
	}
	firstClosed = true

	// The stale update must now affect zero rows.
	stale := harness.begin(t, ctx)
	err = repository.UpdateStatus(
		ctx, stale, ids.member,
		identitypkg.UserStatusActive, identitypkg.UserStatusSuspended,
		ids.now.Add(2*time.Minute),
	)
	_ = stale.Rollback(ctx)
	if err == nil {
		t.Fatal("stale UpdateStatus() succeeded, want zero-row rejection")
	}

	var status identitypkg.UserStatus
	if err := harness.queryRow(
		t, ctx, "SELECT status FROM users WHERE id = $1", ids.member,
	).Scan(&status); err != nil {
		t.Fatalf("read governance race outcome: %v", err)
	}
	if status != identitypkg.UserStatusSuspended {
		t.Fatalf("persisted status = %q, want exactly one suspension", status)
	}
}

// TestS16AuditFailureRollsBackGovernance proves a failed audit append leaves no
// status or session change behind.
func TestS16AuditFailureRollsBackGovernance(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	coordinator, err := identitypkg.NewAccountStatusService(
		identitypkg.AccountStatusDependencies{
			Transactions: &schemaTransactor{pool: harness.pool, quoted: harness.quoted},
			Repository:   identitypkg.PostgresAccountStatusRepository{},
			Audit:        failingAudit{},
			IDs:          &uuidSequence{},
			Clock:        fixedClock{now: ids.now.Add(time.Minute)},
		},
	)
	if err != nil {
		t.Fatalf("NewAccountStatusService() error = %v", err)
	}
	_, err = coordinator.SetAccountStatus(ctx, identitypkg.AccountStatusRequest{
		Actor: session.Principal{
			UserID: ids.initialAdmin, Role: "admin",
			IsInitialAdmin: true, Status: "active",
		},
		TargetID: ids.member, TargetRole: identitypkg.RoleMember,
		Status: identitypkg.UserStatusSuspended, TraceID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	if err == nil {
		t.Fatal("SetAccountStatus() succeeded despite audit failure")
	}
	if code := publicCode(err); code != "platform.internal_error" {
		t.Fatalf("public code = %q, want platform.internal_error (err=%v)", code, err)
	}

	var status identitypkg.UserStatus
	if err := harness.queryRow(
		t, ctx, "SELECT status FROM users WHERE id = $1", ids.member,
	).Scan(&status); err != nil {
		t.Fatalf("read audited-failure state: %v", err)
	}
	if status != identitypkg.UserStatusActive {
		t.Fatalf("persisted status = %q, want unchanged active", status)
	}
	var auditCount int
	if err := harness.queryRow(
		t, ctx,
		"SELECT count(*) FROM audit_events WHERE target_id = $1",
		ids.member,
	).Scan(&auditCount); err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if auditCount != 0 {
		t.Fatalf("audit events = %d, want 0 after rollback", auditCount)
	}
}

// TestS16CommitOutcomeMatrix proves the caller-visible error and the persisted
// state can disagree, and that the matrix reads the real outcome back instead of
// trusting the adapter.
func TestS16CommitOutcomeMatrix(t *testing.T) {
	tests := []struct {
		name          string
		newTransactor func(*schemaHarness) identitypkg.RegistrationTransactor
		wantPersisted bool
		wantAnyError  bool
	}{
		{
			name: "ack lost but committed",
			newTransactor: func(h *schemaHarness) identitypkg.RegistrationTransactor {
				return &ackLossTransactor{pool: h.pool, quoted: h.quoted}
			},
			wantPersisted: true,
			wantAnyError:  true,
		},
		{
			name: "commit never reached the database",
			newTransactor: func(h *schemaHarness) identitypkg.RegistrationTransactor {
				return &rollbackTransactor{pool: h.pool, quoted: h.quoted}
			},
			wantPersisted: false,
			wantAnyError:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newSchemaHarness(t)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			t.Cleanup(cancel)
			ids := seedFixtures(t, harness, ctx)

			coordinator, err := identitypkg.NewAccountStatusService(
				identitypkg.AccountStatusDependencies{
					Transactions: test.newTransactor(harness),
					Repository:   identitypkg.PostgresAccountStatusRepository{},
					Audit:        auditStore{},
					IDs:          &uuidSequence{},
					Clock:        fixedClock{now: ids.now.Add(time.Minute)},
				},
			)
			if err != nil {
				t.Fatalf("NewAccountStatusService() error = %v", err)
			}
			_, err = coordinator.SetAccountStatus(ctx, identitypkg.AccountStatusRequest{
				Actor: session.Principal{
					UserID: ids.initialAdmin, Role: "admin",
					IsInitialAdmin: true, Status: "active",
				},
				TargetID: ids.member, TargetRole: identitypkg.RoleMember,
				Status:  identitypkg.UserStatusSuspended,
				TraceID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			})
			if gotError := err != nil; gotError != test.wantAnyError {
				t.Fatalf("error present = %t, want %t (err=%v)", gotError, test.wantAnyError, err)
			}

			var status identitypkg.UserStatus
			if err := harness.queryRow(
				t, ctx, "SELECT status FROM users WHERE id = $1", ids.member,
			).Scan(&status); err != nil {
				t.Fatalf("read commit-outcome state: %v", err)
			}
			persisted := status == identitypkg.UserStatusSuspended
			if persisted != test.wantPersisted {
				t.Fatalf(
					"persisted suspension = %t (status=%q), want %t",
					persisted, status, test.wantPersisted,
				)
			}
		})
	}
}

// TestS16CanonicalDeleteRollbackLeavesNoHalfTombstone proves the cross-module
// deletion either fully applies or leaves both identity and article state
// untouched.
func TestS16CanonicalDeleteRollbackLeavesNoHalfTombstone(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	failing := identitypkg.ArticleAccountPortFunc(func(
		context.Context,
		pgx.Tx,
		string,
	) (identitypkg.CanonicalDeleteArticleResult, error) {
		return identitypkg.CanonicalDeleteArticleResult{}, errors.New("synthetic article failure")
	})
	coordinator, err := identitypkg.NewCanonicalDeleteService(
		identitypkg.CanonicalDeleteDependencies{
			Transactions: &schemaTransactor{pool: harness.pool, quoted: harness.quoted},
			Repository:   identitypkg.PostgresCanonicalDeleteRepository{},
			Articles:     failing,
			Audit:        auditStore{},
			IDs:          &uuidSequence{},
			Clock:        fixedClock{now: ids.now.Add(time.Minute)},
		},
	)
	if err != nil {
		t.Fatalf("NewCanonicalDeleteService() error = %v", err)
	}
	_, err = coordinator.CanonicalDelete(ctx, identitypkg.CanonicalDeleteRequest{
		Actor: session.Principal{
			UserID: ids.initialAdmin, Role: "admin",
			IsInitialAdmin: true, Status: "active",
		},
		TargetID:   ids.otherMember,
		TargetRole: identitypkg.RoleMember,
		Mode:       identitypkg.CanonicalDeleteDailyGovernance,
		Reason:     identitypkg.AccountDeletionGovernance,
		TraceID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	if err == nil {
		t.Fatal("CanonicalDelete() succeeded despite article failure")
	}

	var status identitypkg.UserStatus
	var deletedAt *time.Time
	if err := harness.queryRow(t, ctx, `
		SELECT status, deleted_at FROM users WHERE id = $1
	`, ids.otherMember).Scan(&status, &deletedAt); err != nil {
		t.Fatalf("read deletion-rollback state: %v", err)
	}
	if status == identitypkg.UserStatusDeleted || deletedAt != nil {
		t.Fatalf("half tombstone after rollback: status=%q deleted_at=%v", status, deletedAt)
	}
}

// TestS16SuccessPathPersistsTombstoneAndReleasesEmail proves the happy path
// still applies fully when every collaborator succeeds.
func TestS16SuccessPathPersistsTombstoneAndReleasesEmail(t *testing.T) {
	harness := newSchemaHarness(t)
	traceCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ctx, err := platform.ContextWithTraceID(traceCtx, "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	ids := seedFixtures(t, harness, ctx)

	noArticles := identitypkg.ArticleAccountPortFunc(func(
		context.Context,
		pgx.Tx,
		string,
	) (identitypkg.CanonicalDeleteArticleResult, error) {
		return identitypkg.CanonicalDeleteArticleResult{}, nil
	})
	coordinator, err := identitypkg.NewCanonicalDeleteService(
		identitypkg.CanonicalDeleteDependencies{
			Transactions: &schemaTransactor{pool: harness.pool, quoted: harness.quoted},
			Repository:   identitypkg.PostgresCanonicalDeleteRepository{},
			Articles:     noArticles,
			Audit:        auditStore{},
			IDs:          &uuidSequence{},
			Clock:        fixedClock{now: ids.now.Add(time.Minute)},
		},
	)
	if err != nil {
		t.Fatalf("NewCanonicalDeleteService() error = %v", err)
	}
	result, err := coordinator.CanonicalDelete(ctx, identitypkg.CanonicalDeleteRequest{
		Actor: session.Principal{
			UserID: ids.initialAdmin, Role: "admin",
			IsInitialAdmin: true, Status: "active",
		},
		TargetID:   ids.otherMember,
		TargetRole: identitypkg.RoleMember,
		Mode:       identitypkg.CanonicalDeleteDailyGovernance,
		Reason:     identitypkg.AccountDeletionGovernance,
		TraceID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	if err != nil {
		t.Fatalf("CanonicalDelete() error = %v (cause=%v)", err, errors.Unwrap(err))
	}
	if result.PendingOrphanCount != 0 || !result.EmailIdentifierFreed {
		t.Fatalf("CanonicalDelete() result = %#v", result)
	}

	var status identitypkg.UserStatus
	var usernameLifecycle identitypkg.IdentifierLifecycle
	var emailRows int
	if err := harness.queryRow(
		t, ctx, "SELECT status FROM users WHERE id = $1", ids.otherMember,
	).Scan(&status); err != nil {
		t.Fatalf("read success tombstone: %v", err)
	}
	if status != identitypkg.UserStatusDeleted {
		t.Fatalf("persisted status = %q, want deleted", status)
	}
	if err := harness.queryRow(t, ctx, `
		SELECT
			max(lifecycle) FILTER (WHERE kind = 'username'),
			count(*) FILTER (WHERE kind = 'email')
		FROM identity_identifiers
		WHERE user_id = $1
	`, ids.otherMember).Scan(&usernameLifecycle, &emailRows); err != nil {
		t.Fatalf("read success identifier lifecycle: %v", err)
	}
	if usernameLifecycle != identitypkg.IdentifierReserved {
		t.Fatalf("username lifecycle = %q, want reserved", usernameLifecycle)
	}
	if emailRows != 0 {
		t.Fatalf("email identifier rows = %d, want 0 after release", emailRows)
	}
	var auditCount int
	if err := harness.queryRow(
		t, ctx,
		"SELECT count(*) FROM audit_events WHERE target_id = $1",
		ids.otherMember,
	).Scan(&auditCount); err != nil {
		t.Fatalf("count deletion audit events: %v", err)
	}
	if auditCount == 0 {
		t.Fatal("deletion audit events = 0, want at least one committed fact")
	}
}

// TestS16SessionResolveRejectsRevokedAndSuspendedUsers proves an authenticated
// principal cannot survive a revocation or suspension.
func TestS16SessionResolveRejectsRevokedAndSuspendedUsers(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	gate := newGateStore()
	sessions := newSessionStore(t, harness, gate)
	const token = "s16-resolve-token"
	tx := harness.begin(t, ctx)
	if _, err := sessions.Create(ctx, tx, session.CreateInput{
		ID:                          randomUUID(t),
		UserID:                      ids.member,
		Token:                       token,
		CSRFToken:                   "s16-resolve-csrf",
		AuthenticatedCredentialKind: session.CredentialPermanent,
		IssuedAt:                    ids.now,
		AbsoluteExpiresAt:           ids.now.Add(8 * time.Hour),
		IdleExpiresAt:               ids.now.Add(time.Hour),
	}); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("create resolve session: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit resolve session: %v", err)
	}

	resolved, err := sessions.ResolvePrincipal(ctx, session.ResolveInput{
		Token: token, CSRFToken: "s16-resolve-csrf",
		RequireCSRF: true, Now: ids.now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("ResolvePrincipal() error = %v", err)
	}
	if resolved.UserID != ids.member || resolved.Role != "member" {
		t.Fatalf("ResolvePrincipal() = %#v", resolved)
	}

	harness.exec(t, ctx, `
		UPDATE sessions SET revoked_at = $2, revoke_reason = 'synthetic_revocation'
		WHERE user_id = $1 AND revoked_at IS NULL
	`, ids.member, ids.now.Add(time.Minute))
	harness.exec(t, ctx, `
		UPDATE users SET status = 'suspended', updated_at = $2 WHERE id = $1
	`, ids.member, ids.now.Add(time.Minute))

	_, err = sessions.ResolvePrincipal(ctx, session.ResolveInput{
		Token: token, CSRFToken: "s16-resolve-csrf",
		RequireCSRF: true, Now: ids.now.Add(2 * time.Minute),
	})
	if err == nil {
		t.Fatal("ResolvePrincipal() authorized a revoked and suspended session")
	}
	if !errors.Is(err, session.ErrSessionRequired) {
		t.Fatalf("ResolvePrincipal() error = %v, want ErrSessionRequired", err)
	}
}

// TestS16MigrationPreflightBlocksUnexplainedLegacyCurrentCredential proves the
// durable migration refuses to run against state it cannot explain.
func TestS16MigrationPreflightBlocksUnexplainedLegacyCurrentCredential(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	// Rebuild the legacy shape: drop the durable columns, restore the legacy
	// current-credential index, and leave one unexplained current credential.
	harness.exec(t, ctx, `ALTER TABLE sessions DROP COLUMN lineage_operation_id`)
	harness.exec(t, ctx, `ALTER TABLE users DROP COLUMN auth_generation`)
	harness.exec(t, ctx, `
		CREATE UNIQUE INDEX temporary_credentials_current_user_idx
		ON temporary_credentials (user_id)
		WHERE state = 'active'
	`)
	harness.exec(t, ctx, `
		INSERT INTO temporary_credentials (
			id, user_id, purpose, generation, state, secret_digest,
			delivered_to_email, issued_at, expires_at
		)
		VALUES ($1, $2, 'password_reset', 1, 'active', 'synthetic-secret-digest',
			'legacy-current@example.test', $3, $4)
	`, randomUUID(t), ids.member, ids.now, ids.now.Add(time.Hour))

	tx := harness.begin(t, ctx)
	_, err := tx.Exec(ctx, readMigrationFile(t, "0008_password_reset_durable.sql"))
	_ = tx.Rollback(ctx)
	if err == nil {
		t.Fatal("0008 applied against an unexplained legacy credential, want preflight rejection")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("preflight error = %v, want *pgconn.PgError", err)
	}
	if pgErr.Code != sqlStateObjectStatePrerequisite {
		t.Fatalf("preflight SQLSTATE = %q, want %q", pgErr.Code, sqlStateObjectStatePrerequisite)
	}
	if !strings.Contains(pgErr.Message, "cannot explain a legacy current credential") {
		t.Fatalf("preflight message = %q, want the stable reason", pgErr.Message)
	}
}

// durableOperation is one synthetic password-reset operation described by the
// exact six-field identity the schema binds together.
type durableOperation struct {
	operationID   string
	credentialID  string
	scopeHash     []byte
	keyLocator    []byte
	keyDigest     []byte
	requestDigest []byte
	generation    int64
	baseAuth      int64
	targetAuth    int64
}

func newDurableOperation(t *testing.T, generation, baseAuth int64) durableOperation {
	t.Helper()
	return durableOperation{
		operationID:   randomUUID(t),
		credentialID:  randomUUID(t),
		scopeHash:     []byte(randomHex(t, 32))[:32],
		keyLocator:    []byte(randomHex(t, 32))[:32],
		keyDigest:     []byte(randomHex(t, 32))[:32],
		requestDigest: []byte(randomHex(t, 32))[:32],
		generation:    generation,
		baseAuth:      baseAuth,
		targetAuth:    baseAuth + 1,
	}
}

// writeDurableFiveObjects inserts the counter, operation, credential, recovery
// job and six-field idempotency claim inside the caller-owned transaction. All
// cross-object foreign keys are DEFERRABLE INITIALLY DEFERRED, so the atomic
// invariant is only enforced at commit time.
func writeDurableFiveObjects(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	op durableOperation,
	now time.Time,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET password_reset_execution_generation = $2,
			password_reset_current_operation_id = $3,
			updated_at = $4
		WHERE id = $1
	`, userID, op.generation, op.operationID, now); err != nil {
		t.Fatalf("advance user counter: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_reset_operations (
			operation_id, user_id, temporary_credential_id,
			scope_hash, key_locator, key_digest, digest_key_version, request_digest,
			execution_generation, request_kind,
			base_auth_generation, target_auth_generation,
			settlement_state, delivery_state, state, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, 'v1', $7,
			$8, 'initial_reset', $9, $10,
			'pending', 'not_started', 'prepared', $11, $11
		)
	`,
		op.operationID, userID, op.credentialID,
		op.scopeHash, op.keyLocator, op.keyDigest, op.requestDigest,
		op.generation, op.baseAuth, op.targetAuth, now,
	); err != nil {
		t.Fatalf("insert operation: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO temporary_credentials (
			id, user_id, purpose, generation, auth_generation,
			source_operation_id, state, secret_digest, delivered_to_email,
			issued_at, expires_at
		)
		VALUES ($1, $2, 'password_reset', $3, $4, $5,
			'pending_delivery', 'synthetic-credential-digest',
			'durable@example.test', $6, $7)
	`,
		op.credentialID, userID, op.generation, op.targetAuth,
		op.operationID, now, now.Add(time.Hour),
	); err != nil {
		t.Fatalf("insert credential: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_reset_recovery_jobs (
			operation_id, status, attempt_count, next_attempt_at,
			created_at, updated_at
		)
		VALUES ($1, 'ready', 0, $2, $3, $3)
	`, op.operationID, now.Add(time.Minute), now); err != nil {
		t.Fatalf("insert recovery job: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO idempotency_records (
			scope_hash, key_locator, key_digest, request_digest, digest_key_version,
			status, execution_generation,
			password_reset_operation_id, password_reset_user_id,
			locked_until, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'v1', 'processing', $5, $6, $7, $8, $9, $10, $10)
	`,
		op.scopeHash, op.keyLocator, op.keyDigest, op.requestDigest,
		op.generation, op.operationID, userID,
		now.Add(time.Minute), now.Add(24*time.Hour), now,
	); err != nil {
		t.Fatalf("insert idempotency claim: %v", err)
	}
}

// TestS16FiveObjectAtomicityAndCounterGap proves the deferred counter triggers
// accept a complete five-object commit and reject a counter/operation gap that
// only becomes visible at commit time.
func TestS16FiveObjectAtomicityAndCounterGap(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	t.Run("complete commit is durable", func(t *testing.T) {
		op := newDurableOperation(t, 1, 0)
		tx := harness.begin(t, ctx)
		writeDurableFiveObjects(t, ctx, tx, ids.member, op, ids.now)
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit five-object write: %v", err)
		}

		var counter int64
		var current *string
		if err := harness.queryRow(t, ctx, `
			SELECT password_reset_execution_generation, password_reset_current_operation_id
			FROM users WHERE id = $1
		`, ids.member).Scan(&counter, &current); err != nil {
			t.Fatalf("read committed counter: %v", err)
		}
		if counter != 1 || current == nil || *current != op.operationID {
			t.Fatalf("counter/current = %d/%v, want 1/%s", counter, current, op.operationID)
		}
		var objects int
		if err := harness.queryRow(t, ctx, `
			SELECT
				(SELECT count(*) FROM password_reset_operations WHERE operation_id = $1)
				+ (SELECT count(*) FROM temporary_credentials WHERE id = $2)
				+ (SELECT count(*) FROM password_reset_recovery_jobs WHERE operation_id = $1)
				+ (SELECT count(*) FROM idempotency_records
					WHERE password_reset_operation_id = $1)
		`, op.operationID, op.credentialID).Scan(&objects); err != nil {
			t.Fatalf("count committed objects: %v", err)
		}
		if objects != 4 {
			t.Fatalf("committed objects = %d, want 4", objects)
		}
	})

	t.Run("counter gap fails closed at commit", func(t *testing.T) {
		// Claim generation 2 while no generation-1 operation exists: the row-level
		// foreign key stays satisfiable, so only the deferred counter trigger can
		// reject the gap.
		op := newDurableOperation(t, 2, 0)
		tx := harness.begin(t, ctx)
		writeDurableFiveObjects(t, ctx, tx, ids.otherMember, op, ids.now)
		err := tx.Commit(ctx)
		if err == nil {
			t.Fatal("commit with a counter gap succeeded, want deferred rejection")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != sqlStateObjectStatePrerequisite {
			t.Fatalf("counter gap error = %v, want 55000", err)
		}

		var objects int
		if err := harness.queryRow(t, ctx, `
			SELECT
				(SELECT count(*) FROM password_reset_operations WHERE operation_id = $1)
				+ (SELECT count(*) FROM temporary_credentials WHERE id = $2)
				+ (SELECT count(*) FROM password_reset_recovery_jobs WHERE operation_id = $1)
				+ (SELECT count(*) FROM idempotency_records
					WHERE password_reset_operation_id = $1)
		`, op.operationID, op.credentialID).Scan(&objects); err != nil {
			t.Fatalf("count rolled-back objects: %v", err)
		}
		if objects != 0 {
			t.Fatalf("objects after failed commit = %d, want 0 (all-or-nothing)", objects)
		}
	})
}

// TestS16RecoveryLeaseSingleWinnerAndTakeover proves exactly one worker claims a
// ready job, and that an expired lease is only taken over after it lapses.
func TestS16RecoveryLeaseSingleWinnerAndTakeover(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	op := newDurableOperation(t, 1, 0)
	tx := harness.begin(t, ctx)
	writeDurableFiveObjects(t, ctx, tx, ids.member, op, ids.now)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit leased job fixture: %v", err)
	}

	claim := func(owner string, now time.Time) (int64, bool) {
		t.Helper()
		claimTx := harness.begin(t, ctx)
		defer func() { _ = claimTx.Rollback(ctx) }()
		// Mirror the production Claim statement: a single old-state-guarded
		// UPDATE that can only match while the job is ready or the lease lapsed.
		var attempt int64
		err := claimTx.QueryRow(ctx, `
			UPDATE password_reset_recovery_jobs j
			SET status = 'leased',
				attempt_count = j.attempt_count + 1,
				lease_owner = $2::text,
				lease_expires_at = $4,
				updated_at = $3
			WHERE j.operation_id = $1
				AND ((j.status IN ('ready', 'retry_wait') AND j.next_attempt_at <= $3)
					OR (j.status = 'leased' AND j.lease_expires_at <= $3))
			RETURNING j.attempt_count
		`, op.operationID, owner, now, now.Add(time.Minute)).Scan(&attempt)
		if errors.Is(err, pgx.ErrNoRows) {
			if commitErr := claimTx.Commit(ctx); commitErr != nil {
				t.Fatalf("commit no-op claim: %v", commitErr)
			}
			return 0, false
		}
		if err != nil {
			t.Fatalf("claim job: %v", err)
		}
		if commitErr := claimTx.Commit(ctx); commitErr != nil {
			t.Fatalf("commit claim: %v", commitErr)
		}
		return attempt, true
	}

	// The job became eligible at now+1m; claim after that boundary.
	firstAttempt, won := claim("worker-a", ids.now.Add(2*time.Minute))
	if !won || firstAttempt != 1 {
		t.Fatalf("first claim = (%d,%t), want (1,true)", firstAttempt, won)
	}
	// A second worker must not steal a live lease.
	if _, stolen := claim("worker-b", ids.now.Add(2*time.Minute+10*time.Second)); stolen {
		t.Fatal("second worker claimed a live lease, want single winner")
	}
	// Only after the lease lapses may another worker take it over.
	secondAttempt, took := claim("worker-b", ids.now.Add(4*time.Minute))
	if !took || secondAttempt != 2 {
		t.Fatalf("takeover claim = (%d,%t), want (2,true)", secondAttempt, took)
	}

	var owner string
	var attempts int64
	if err := harness.queryRow(t, ctx, `
		SELECT lease_owner, attempt_count FROM password_reset_recovery_jobs
		WHERE operation_id = $1
	`, op.operationID).Scan(&owner, &attempts); err != nil {
		t.Fatalf("read lease outcome: %v", err)
	}
	if owner != "worker-b" || attempts != 2 {
		t.Fatalf("final lease = %s/%d, want worker-b/2", owner, attempts)
	}
}

// TestS16OperationBindingImmutableAndLegacyRollbackBlocked proves the durable
// binding trigger rejects any attempt to re-point an operation's six-field
// identity, and that the legacy current-credential shape can no longer be
// written after the forward migration (an old binary cannot run backward).
func TestS16OperationBindingImmutableAndLegacyRollbackBlocked(t *testing.T) {
	harness := newSchemaHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	ids := seedFixtures(t, harness, ctx)

	op := newDurableOperation(t, 1, 0)
	tx := harness.begin(t, ctx)
	writeDurableFiveObjects(t, ctx, tx, ids.member, op, ids.now)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit binding fixture: %v", err)
	}

	t.Run("binding is immutable", func(t *testing.T) {
		mm := harness.begin(t, ctx)
		_, err := mm.Exec(ctx, `
			UPDATE password_reset_operations
			SET execution_generation = execution_generation + 1
			WHERE operation_id = $1
		`, op.operationID)
		_ = mm.Rollback(ctx)
		if err == nil {
			t.Fatal("binding update succeeded, want immutable rejection")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != sqlStateObjectStatePrerequisite {
			t.Fatalf("binding update error = %v, want 55000", err)
		}
	})

	t.Run("legacy current-credential index is gone", func(t *testing.T) {
		var exists bool
		if err := harness.queryRow(t, ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = current_schema()
					AND indexname = 'temporary_credentials_current_user_idx'
			)
		`).Scan(&exists); err != nil {
			t.Fatalf("probe legacy index: %v", err)
		}
		if exists {
			t.Fatal("legacy temporary_credentials_current_user_idx survived the forward migration")
		}
	})

	t.Run("generation cannot move backwards", func(t *testing.T) {
		// An old binary that never learned about the durable generations would try
		// to reset the counters. The monotonic trigger rejects any backward move,
		// which is what makes forward-only rollback blocking real.
		regress := harness.begin(t, ctx)
		_, err := regress.Exec(ctx, `
			UPDATE users
			SET auth_generation = 0,
				password_reset_execution_generation = 0
			WHERE id = $1
		`, ids.member)
		_ = regress.Rollback(ctx)
		if err == nil {
			t.Fatal("backward generation update succeeded, want monotonic rejection")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != sqlStateObjectStatePrerequisite {
			t.Fatalf("backward generation error = %v, want 55000", err)
		}
		if !strings.Contains(pgErr.Message, "cannot move backwards") {
			t.Fatalf("backward generation message = %q", pgErr.Message)
		}
	})
}

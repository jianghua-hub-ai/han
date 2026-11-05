//go:build integration

// Fault adapters for the S1.6 PostgreSQL matrix. They never fabricate a commit
// outcome: each adapter distinguishes "the caller saw an error" from "the
// database actually committed", and the assertions read the real state back.
package identity

import (
	"context"
	"errors"
	"sync"
	"time"

	"han/internal/audit"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaTransactor begins caller-owned transactions bound to one isolated
// schema and reports the backend PID of every begun transaction.
type schemaTransactor struct {
	pool   *pgxpool.Pool
	quoted string

	mu   sync.Mutex
	pids []int
}

func newSchemaTransactor(pool *pgxpool.Pool, quoted string) *schemaTransactor {
	return &schemaTransactor{pool: pool, quoted: quoted}
}

// QueryRow satisfies the session store's read-only pre-read contract.
func (s *schemaTransactor) QueryRow(
	ctx context.Context,
	sql string,
	args ...any,
) pgx.Row {
	tx, err := s.Begin(ctx)
	if err != nil {
		return errorRow{err: err}
	}
	return &transactionRow{row: tx.QueryRow(ctx, sql, args...), tx: tx, ctx: ctx}
}

// errorRow surfaces a begin failure through the pgx.Row contract.
type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

// transactionRow releases its read-only transaction after the single scan.
type transactionRow struct {
	row pgx.Row
	tx  pgx.Tx
	ctx context.Context
}

func (r *transactionRow) Scan(destinations ...any) error {
	err := r.row.Scan(destinations...)
	_ = r.tx.Rollback(r.ctx)
	return err
}

// Begin opens one transaction with the isolated search path applied.
func (s *schemaTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+s.quoted); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	var pid int
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	s.mu.Lock()
	s.pids = append(s.pids, pid)
	s.mu.Unlock()
	return tx, nil
}

func (s *schemaTransactor) backendPIDs() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int(nil), s.pids...)
}

// commitErrorTx reports a commit error while the transaction was actually
// rolled back. It is the safe direction of an unknown outcome: the caller must
// treat the operation as not committed.
type commitErrorTx struct {
	pgx.Tx
}

func (t commitErrorTx) Commit(context.Context) error {
	_ = t.Tx.Rollback(context.Background())
	return errors.New("synthetic commit failure after rollback")
}

// commitAckLossTx reports a commit error while the transaction actually
// committed. Only this adapter may be used to prove an ack-loss path, because
// the assertions can then observe the persisted rows.
type commitAckLossTx struct {
	pgx.Tx
}

func (t commitAckLossTx) Commit(ctx context.Context) error {
	if err := t.Tx.Commit(ctx); err != nil {
		return err
	}
	return errors.New("synthetic commit acknowledgement lost")
}

// ackLossTransactor commits every transaction but reports an error, modelling a
// lost acknowledgement after a real commit.
type ackLossTransactor struct {
	pool   *pgxpool.Pool
	quoted string
}

func (a *ackLossTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+a.quoted); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return commitAckLossTx{Tx: tx}, nil
}

// rollbackTransactor rolls every transaction back and reports a commit error,
// modelling a commit that never reached the database.
type rollbackTransactor struct {
	pool   *pgxpool.Pool
	quoted string
}

func (r *rollbackTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+r.quoted); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return commitErrorTx{Tx: tx}, nil
}

// fixedClock is a deterministic UTC clock for coordinator construction.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// uuidSequence emits deterministic, unique UUIDs for coordinator construction.
type uuidSequence struct {
	mu    sync.Mutex
	count int
}

func (g *uuidSequence) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.count++
	return formatSequenceUUID(g.count), nil
}

func formatSequenceUUID(value int) string {
	const alphabet = "0123456789abcdef"
	hex := make([]byte, 0, 32)
	for _, shift := range []uint{28, 24, 20, 16, 12, 8, 4, 0} {
		hex = append(hex, alphabet[(value>>shift)&0x0f])
	}
	body := make([]byte, 32)
	copy(body, "0000")
	copy(body[4:], hex)
	for index := 12; index < 32; index++ {
		body[index] = '0'
	}
	body[12] = '4'
	body[16] = '8'
	return string(body[0:8]) + "-" + string(body[8:12]) + "-" +
		string(body[12:16]) + "-" + string(body[16:20]) + "-" +
		string(body[20:32])
}

// tableExists reports whether a table exists in the named schema.
func tableExists(
	ctx context.Context,
	pool *pgxpool.Pool,
	quotedSchema, table string,
) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)
	`, trimQuotes(quotedSchema), table).Scan(&exists)
	return exists, err
}

func trimQuotes(identifier string) string {
	if len(identifier) >= 2 && identifier[0] == '"' &&
		identifier[len(identifier)-1] == '"' {
		return identifier[1 : len(identifier)-1]
	}
	return identifier
}

// auditStore adapts the real append-only audit store to the coordinator ports.
type auditStore struct{}

func (auditStore) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	return audit.Store{}.Append(ctx, tx, event)
}

// failingAudit rejects every append, proving an audit failure rolls back the
// owning business transaction.
type failingAudit struct{}

func (failingAudit) Append(context.Context, pgx.Tx, audit.Event) error {
	return errors.New("synthetic audit failure")
}

// publicCode returns the stable public error code carried by err.
func publicCode(err error) platform.Code {
	return platform.PublicCode(err)
}

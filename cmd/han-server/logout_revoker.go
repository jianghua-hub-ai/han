package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/session"
)

// postgresSessionLogout revokes one session inside its own transaction.
type postgresSessionLogout struct {
	pool  *pgxpool.Pool
	store *session.Store
}

func (p postgresSessionLogout) RevokeCurrentSession(
	ctx context.Context,
	token string,
	revokedAt time.Time,
	reason string,
) error {
	if p.pool == nil || p.store == nil {
		return errors.New("session logout dependencies are required")
	}
	transaction, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if err := p.store.Revoke(ctx, transaction, token, revokedAt, reason); err != nil {
		return err
	}
	return transaction.Commit(ctx)
}

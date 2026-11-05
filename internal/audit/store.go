// Package audit owns append-only PostgreSQL audit events.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ActorKind identifies the accountable subject category.
type ActorKind string

const (
	ActorAnonymous ActorKind = "anonymous"
	ActorUser      ActorKind = "user"
	ActorSystem    ActorKind = "system"
)

// Event is the minimum append-only audit fact.
type Event struct {
	ID                string
	OccurredAt        time.Time
	ActorKind         ActorKind
	ActorUserID       *string
	ActorSystem       *string
	ActorRole         *string
	Action            string
	TargetType        string
	TargetID          string
	Outcome           string
	ErrorCode         *string
	TraceID           string
	SourceFingerprint *string
	Metadata          map[string]any
}

// QueryRower is the minimum PostgreSQL query contract consumed by Store.
type QueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Store appends and reads audit events without update or delete capabilities.
type Store struct{}

// Append inserts an audit event into the caller-owned transaction.
func (Store) Append(ctx context.Context, tx pgx.Tx, event Event) error {
	if tx == nil {
		return errors.New("audit transaction is required")
	}
	if event.OccurredAt.IsZero() {
		return errors.New("audit event occurred_at is required")
	}
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (
			id,
			occurred_at,
			actor_kind,
			actor_user_id,
			actor_system,
			actor_role,
			action,
			target_type,
			target_id,
			outcome,
			error_code,
			trace_id,
			source_fingerprint,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`,
		event.ID,
		event.OccurredAt,
		event.ActorKind,
		event.ActorUserID,
		event.ActorSystem,
		event.ActorRole,
		event.Action,
		event.TargetType,
		event.TargetID,
		event.Outcome,
		event.ErrorCode,
		event.TraceID,
		event.SourceFingerprint,
		encodedMetadata,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// Get returns one audit event by identifier.
func (Store) Get(ctx context.Context, queryer QueryRower, id string) (Event, error) {
	if queryer == nil {
		return Event{}, errors.New("audit queryer is required")
	}

	var event Event
	var metadata []byte
	err := queryer.QueryRow(ctx, `
		SELECT
			id,
			occurred_at,
			actor_kind,
			actor_user_id,
			actor_system,
			actor_role,
			action,
			target_type,
			target_id,
			outcome,
			error_code,
			trace_id,
			source_fingerprint,
			metadata
		FROM audit_events
		WHERE id = $1
	`, id).Scan(
		&event.ID,
		&event.OccurredAt,
		&event.ActorKind,
		&event.ActorUserID,
		&event.ActorSystem,
		&event.ActorRole,
		&event.Action,
		&event.TargetType,
		&event.TargetID,
		&event.Outcome,
		&event.ErrorCode,
		&event.TraceID,
		&event.SourceFingerprint,
		&metadata,
	)
	if err != nil {
		return Event{}, fmt.Errorf("query audit event: %w", err)
	}
	event.Metadata, err = decodeMetadata(metadata)
	if err != nil {
		return Event{}, fmt.Errorf("decode audit metadata: %w", err)
	}
	return event, nil
}

func decodeMetadata(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, err
	}
	if metadata == nil {
		return map[string]any{}, nil
	}
	return metadata, nil
}

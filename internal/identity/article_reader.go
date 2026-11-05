package identity

import (
	"context"
	"errors"
	"fmt"

	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuthorState is the minimum identity fact consumed by article-management.
type AuthorState struct {
	ID     string
	Status UserStatus
}

// AuthorStateDatabase is the read-only database capability used by ArticleIdentityReader.
type AuthorStateDatabase interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// ArticleIdentityReader exposes identity-owned author states without article writes.
type ArticleIdentityReader struct {
	database AuthorStateDatabase
}

// NewArticleIdentityReader creates the read-only author state seam.
func NewArticleIdentityReader(database AuthorStateDatabase) (*ArticleIdentityReader, error) {
	if database == nil {
		return nil, errors.New("article identity database is required")
	}
	return &ArticleIdentityReader{database: database}, nil
}

// GetAuthorStates returns the authoritative status for each found author.
func (r *ArticleIdentityReader) GetAuthorStates(
	ctx context.Context,
	userIDs []string,
) (map[string]UserStatus, error) {
	if len(userIDs) == 0 {
		return map[string]UserStatus{}, nil
	}
	ids, requested, err := normalizeAuthorStateIDs(userIDs)
	if err != nil {
		return nil, err
	}
	rows, err := r.database.Query(ctx, getAuthorStatesSQL, ids)
	if err != nil {
		return nil, authorStateInternalError("query author states", err)
	}
	defer rows.Close()
	return scanAuthorStates(rows, requested)
}

func normalizeAuthorStateIDs(
	userIDs []string,
) ([]pgtype.UUID, map[string]struct{}, error) {
	ids := make([]pgtype.UUID, 0, len(userIDs))
	requested := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		var parsed pgtype.UUID
		if err := parsed.Scan(userID); err != nil || !parsed.Valid {
			return nil, nil, authorStateInternalError("validate author user ID", err)
		}
		canonical := parsed.String()
		if _, exists := requested[canonical]; exists {
			continue
		}
		requested[canonical] = struct{}{}
		ids = append(ids, parsed)
	}
	return ids, requested, nil
}

func scanAuthorStates(
	rows pgx.Rows,
	requested map[string]struct{},
) (map[string]UserStatus, error) {
	states := make(map[string]UserStatus)
	for rows.Next() {
		var state AuthorState
		if err := rows.Scan(&state.ID, &state.Status); err != nil {
			return nil, authorStateInternalError("scan author state", err)
		}
		if _, requestedID := requested[state.ID]; !requestedID ||
			!validAuthorStatus(state.Status) {
			return nil, authorStateInternalError(
				"validate author state",
				errors.New("database returned an invalid author state"),
			)
		}
		if previous, duplicate := states[state.ID]; duplicate && previous != state.Status {
			return nil, authorStateInternalError(
				"validate author state",
				errors.New("database returned conflicting author states"),
			)
		}
		states[state.ID] = state.Status
	}
	if err := rows.Err(); err != nil {
		return nil, authorStateInternalError("read author states", err)
	}
	return states, nil
}

func validAuthorStatus(status UserStatus) bool {
	return status == UserStatusActive ||
		status == UserStatusSuspended ||
		status == UserStatusDeleted
}

func authorStateInternalError(operation string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown author state failure")
	}
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

const getAuthorStatesSQL = `
	SELECT id, status
	FROM users
	WHERE id = ANY($1::uuid[])
`

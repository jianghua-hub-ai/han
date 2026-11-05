package article

import (
	"context"
	"errors"
	"fmt"
	"time"

	"han/internal/audit"

	"github.com/jackc/pgx/v5"
)

// OrphanArticlesResult reports canonical orphaning work for one deleted user.
type OrphanArticlesResult struct {
	NewlyOrphanedCount   int
	AlreadyOrphanedCount int
	PendingOrphanCount   int
}

// CanonicalOrphanUpdate is the only Article mutation permitted by canonical orphaning.
type CanonicalOrphanUpdate struct {
	ArticleID       string
	DeletedUserID   string
	ExpectedVersion int64
	UpdatedAt       time.Time
}

// OrphanArticlesCanonicalRepository owns locked canonical orphan persistence.
// Lock returns active and deletion-pending rows still associated with the
// deleted user. Set writes NULL author, held publication, deleted-user source,
// expected version + 1, and UpdatedAt without changing violation state.
type OrphanArticlesCanonicalRepository interface {
	LockCanonicalOrphanCandidates(context.Context, pgx.Tx, string) ([]Article, error)
	SetCanonicalOrphan(context.Context, pgx.Tx, CanonicalOrphanUpdate) error
}

// CanonicalOrphanRepository persists canonical orphan state in PostgreSQL.
type CanonicalOrphanRepository struct{}

var _ OrphanArticlesCanonicalRepository = CanonicalOrphanRepository{}

const lockCanonicalOrphanCandidatesSQL = `
	SELECT
		id,
		author_id,
		orphaned_from_user_id,
		title,
		body,
		publication_state,
		violation_state,
		violation_category,
		deletion_state,
		deletion_operation_id,
		version,
		created_at,
		updated_at
	FROM articles
	WHERE deletion_state IN ('active', 'deletion_pending')
		AND (author_id = $1 OR orphaned_from_user_id = $1)
	ORDER BY id
	FOR UPDATE
`

const setCanonicalOrphanSQL = `
	UPDATE articles
	SET
		author_id = NULL,
		publication_state = $1,
		orphaned_from_user_id = $2,
		version = version + 1,
		updated_at = $3
	WHERE id = $4
		AND deletion_state = $5
		AND author_id = $2
		AND version = $6
`

// LockCanonicalOrphanCandidates locks rows still associated with a deleted user.
func (CanonicalOrphanRepository) LockCanonicalOrphanCandidates(
	ctx context.Context,
	tx pgx.Tx,
	deletedUserID string,
) ([]Article, error) {
	if tx == nil {
		return nil, errors.New("canonical orphan transaction is required")
	}
	if deletedUserID == "" {
		return nil, errors.New("deleted user ID is required")
	}
	rows, err := tx.Query(ctx, lockCanonicalOrphanCandidatesSQL, deletedUserID)
	if err != nil {
		return nil, fmt.Errorf("query canonical orphan candidates: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		article, err := scanCanonicalOrphanArticle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan canonical orphan candidate: %w", err)
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read canonical orphan candidates: %w", err)
	}
	return articles, nil
}

// SetCanonicalOrphan writes the canonical active Article state transition.
func (CanonicalOrphanRepository) SetCanonicalOrphan(
	ctx context.Context,
	tx pgx.Tx,
	update CanonicalOrphanUpdate,
) error {
	if tx == nil {
		return errors.New("canonical orphan transaction is required")
	}
	if update.ArticleID == "" || update.DeletedUserID == "" {
		return errors.New("canonical orphan article and deleted user IDs are required")
	}
	if update.ExpectedVersion < 1 {
		return errors.New("canonical orphan expected version is invalid")
	}
	if update.UpdatedAt.IsZero() {
		return errors.New("canonical orphan update time is required")
	}
	tag, err := tx.Exec(
		ctx,
		setCanonicalOrphanSQL,
		PublicationHeld,
		update.DeletedUserID,
		update.UpdatedAt,
		update.ArticleID,
		DeletionActive,
		update.ExpectedVersion,
	)
	if err != nil {
		return fmt.Errorf("update canonical orphan article: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"update canonical orphan article affected %d rows",
			tag.RowsAffected(),
		)
	}
	return nil
}

func scanCanonicalOrphanArticle(rows pgx.Rows) (Article, error) {
	var article Article
	err := rows.Scan(
		&article.ID,
		&article.AuthorID,
		&article.OrphanedFromUserID,
		&article.Title,
		&article.Body,
		&article.PublicationState,
		&article.ViolationState,
		&article.ViolationCategory,
		&article.DeletionState,
		&article.DeletionOperationID,
		&article.Version,
		&article.CreatedAt,
		&article.UpdatedAt,
	)
	return article, err
}

// OrphanArticlesCanonicalAuditor appends an AuditEvent in the caller-owned transaction.
type OrphanArticlesCanonicalAuditor interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

var _ OrphanArticlesCanonicalAuditor = audit.Store{}

// OrphanArticlesCanonicalIDGenerator creates opaque AuditEvent identifiers.
type OrphanArticlesCanonicalIDGenerator interface {
	NewID() (string, error)
}

// OrphanArticlesCanonicalClock supplies canonical orphaning timestamps.
type OrphanArticlesCanonicalClock interface {
	Now() time.Time
}

// OrphanArticlesCanonicalDependencies are optional until the internal port is composed.
type OrphanArticlesCanonicalDependencies struct {
	Repository OrphanArticlesCanonicalRepository
	Auditor    OrphanArticlesCanonicalAuditor
	AuditIDs   OrphanArticlesCanonicalIDGenerator
	Clock      OrphanArticlesCanonicalClock
}

package identity

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CanonicalDeleteArticleResult is the article-owned canonical orphaning result.
type CanonicalDeleteArticleResult struct {
	NewlyOrphanedCount   int
	AlreadyOrphanedCount int
	PendingOrphanCount   int
}

// ArticleAccountPort applies article-owned canonical orphaning in the caller transaction.
type ArticleAccountPort interface {
	OrphanArticlesCanonical(
		context.Context,
		pgx.Tx,
		string,
	) (CanonicalDeleteArticleResult, error)
}

// ArticleAccountPortFunc adapts the article-owned command without owning its rules.
type ArticleAccountPortFunc func(
	context.Context,
	pgx.Tx,
	string,
) (CanonicalDeleteArticleResult, error)

// OrphanArticlesCanonical calls the configured article-owned command.
func (f ArticleAccountPortFunc) OrphanArticlesCanonical(
	ctx context.Context,
	tx pgx.Tx,
	deletedUserID string,
) (CanonicalDeleteArticleResult, error) {
	return f(ctx, tx, deletedUserID)
}

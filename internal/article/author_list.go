package article

import (
	"context"
	"errors"
	"fmt"

	"han/internal/platform"
	"han/internal/session"
)

// AuthorArticleRepository lists one author's own Articles in keyset order.
//
// It is a separate seam instead of a new method on ArticleQueryRepository so
// the member query contract and every existing test double stay untouched.
type AuthorArticleRepository interface {
	ListAuthorBatch(context.Context, string, *ArticleCursor, int) ([]Article, error)
}

// AuthorArticleLister serves GET /api/v1/users/me/articles.
type AuthorArticleLister interface {
	ListAuthorArticles(
		context.Context,
		session.Principal,
		ListArticlesRequest,
	) (ListArticlesPage, error)
}

var _ AuthorArticleRepository = (*PostgresArticleQueryRepository)(nil)
var _ AuthorArticleLister = (*ArticleQueryService)(nil)

// ListAuthorBatch returns one stable descending keyset batch for one author.
func (r *PostgresArticleQueryRepository) ListAuthorBatch(
	ctx context.Context,
	authorID string,
	after *ArticleCursor,
	limit int,
) ([]Article, error) {
	if r == nil || r.database == nil || limit <= 0 || !validUUID(authorID) {
		return nil, errors.New("author article query input is invalid")
	}
	var cursorCreatedAt any
	var cursorID any
	if after != nil {
		if !candidateCursorView(after.View) ||
			after.CreatedAt.IsZero() ||
			!validUUID(after.ID) {
			return nil, errors.New("author article cursor is invalid")
		}
		cursorCreatedAt = after.CreatedAt.UTC()
		cursorID = after.ID
	}
	rows, err := r.database.Query(
		ctx,
		listAuthorArticleCandidatesSQL,
		authorID,
		cursorCreatedAt,
		cursorID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query author article candidates: %w", err)
	}
	defer rows.Close()

	articles := make([]Article, 0, limit)
	for rows.Next() {
		article, err := scanQueryArticle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan author article candidate: %w", err)
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read author article candidates: %w", err)
	}
	return articles, nil
}

// ListAuthorArticles returns one page of the caller's own Articles.
func (s *ArticleQueryService) ListAuthorArticles(
	ctx context.Context,
	principal session.Principal,
	request ListArticlesRequest,
) (ListArticlesPage, error) {
	if s == nil || s.authorRepository == nil {
		return ListArticlesPage{}, articleQueryInternalError(
			"list author articles",
			errors.New("article author list is not configured"),
		)
	}
	if !principal.Allows(session.AccessGeneral) || principal.UserID == "" {
		return ListArticlesPage{}, articleQueryPublicError(
			platform.CodeAuthSessionRequired,
			"list author articles",
			errors.New("article principal is required"),
		)
	}
	limit, err := s.resolveLimit(request.Limit)
	if err != nil {
		return ListArticlesPage{}, err
	}
	scanAfter, err := decodeMemberCursor(request.Cursor)
	if err != nil {
		return ListArticlesPage{}, err
	}
	return s.scanAuthorArticles(ctx, principal, scanAfter, limit)
}

func (s *ArticleQueryService) scanAuthorArticles(
	ctx context.Context,
	principal session.Principal,
	scanAfter *ArticleCursor,
	limit int,
) (ListArticlesPage, error) {
	visible := make([]MemberArticleSummaryView, 0, limit+1)
	for len(visible) < limit+1 {
		candidates, exhausted, err := s.readAuthorBatch(
			ctx,
			principal.UserID,
			scanAfter,
		)
		if err != nil {
			return ListArticlesPage{}, err
		}
		if len(candidates) == 0 {
			break
		}
		for _, candidate := range candidates {
			scanAfter = articleCursor(candidate)
			item, err := (VisibilityPolicy{}).AuthorListProjection(
				candidate,
				principal,
			)
			if errors.Is(err, ErrArticleUnavailable) {
				continue
			}
			if err != nil {
				return ListArticlesPage{}, articleQueryInternalError(
					"project author article list item",
					err,
				)
			}
			visible = append(visible, item)
			if len(visible) == limit+1 {
				break
			}
		}
		if len(visible) == limit+1 || exhausted {
			break
		}
	}
	return buildListArticlesPage(visible, limit)
}

func (s *ArticleQueryService) readAuthorBatch(
	ctx context.Context,
	authorID string,
	scanAfter *ArticleCursor,
) ([]Article, bool, error) {
	candidates, err := s.authorRepository.ListAuthorBatch(
		ctx,
		authorID,
		scanAfter,
		s.config.CandidateBatchSize,
	)
	if err != nil {
		return nil, false, articleQueryDatabaseError(
			"list author article candidates",
			err,
		)
	}
	if len(candidates) > s.config.CandidateBatchSize {
		return nil, false, articleQueryInternalError(
			"list author article candidates",
			errors.New("article repository exceeded candidate batch size"),
		)
	}
	if err := validateCandidateOrder(scanAfter, candidates); err != nil {
		return nil, false, articleQueryInternalError(
			"validate author article candidate order",
			err,
		)
	}
	return candidates, len(candidates) < s.config.CandidateBatchSize, nil
}

const listAuthorArticleCandidatesSQL = `
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
	WHERE author_id = $1
		AND (
			$2::timestamptz IS NULL
			OR created_at < $2::timestamptz
			OR (created_at = $2::timestamptz AND id < $3::uuid)
		)
	ORDER BY created_at DESC, id DESC
	LIMIT $4
`

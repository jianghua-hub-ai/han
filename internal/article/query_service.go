package article

import (
	"context"
	"errors"
	"fmt"
	"time"

	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/session"
)

// QueryConfig supplies deployment-owned member pagination bounds.
type QueryConfig struct {
	DefaultLimit       int
	MaxLimit           int
	CandidateBatchSize int
}

// ListArticlesRequest contains one member list request.
type ListArticlesRequest struct {
	Cursor string
	Limit  *int
}

// ListArticlesPage is the fixed member list envelope.
type ListArticlesPage struct {
	Items      []MemberArticleSummaryView `json:"items"`
	NextCursor *string                    `json:"next_cursor"`
}

// ArticleQueryRepository owns read-only Article persistence.
type ArticleQueryRepository interface {
	ListCandidateBatch(context.Context, *ArticleCursor, int) ([]Article, error)
	GetArticle(context.Context, string) (Article, error)
}

// ArticleQueryDatabase is the read-only PostgreSQL capability used by member queries.
type ArticleQueryDatabase interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

var _ ArticleQueryDatabase = (*pgxpool.Pool)(nil)

// PostgresArticleQueryRepository reads Article query projections from PostgreSQL.
type PostgresArticleQueryRepository struct {
	database ArticleQueryDatabase
}

// NewPostgresArticleQueryRepository creates the production read-only repository.
func NewPostgresArticleQueryRepository(
	database ArticleQueryDatabase,
) (*PostgresArticleQueryRepository, error) {
	if database == nil {
		return nil, errors.New("article query database is required")
	}
	return &PostgresArticleQueryRepository{database: database}, nil
}

// ArticleAuthorStateReader reads current identity-owned author states.
type ArticleAuthorStateReader[Status ~string] interface {
	GetAuthorStates(context.Context, []string) (map[string]Status, error)
}

// ArticleQueryService serves member-visible Article reads.
type ArticleQueryService struct {
	repository       ArticleQueryRepository
	authorRepository AuthorArticleRepository
	identity         func(context.Context, []string) (map[string]string, error)
	config           QueryConfig
}

// NewArticleQueryService creates the read-only Article query service.
func NewArticleQueryService[Status ~string](
	repository ArticleQueryRepository,
	identityReader ArticleAuthorStateReader[Status],
	config QueryConfig,
) (*ArticleQueryService, error) {
	if repository == nil || identityReader == nil {
		return nil, errors.New("article query dependencies are incomplete")
	}
	if config.DefaultLimit <= 0 ||
		config.MaxLimit < config.DefaultLimit ||
		config.CandidateBatchSize <= config.MaxLimit {
		return nil, errors.New("article query pagination config is invalid")
	}
	service := &ArticleQueryService{
		repository: repository,
		identity: func(
			ctx context.Context,
			ids []string,
		) (map[string]string, error) {
			states, err := identityReader.GetAuthorStates(ctx, ids)
			if err != nil {
				return nil, err
			}
			normalized := make(map[string]string, len(states))
			for id, status := range states {
				normalized[id] = string(status)
			}
			return normalized, nil
		},
		config: config,
	}
	if authorRepository, ok := repository.(AuthorArticleRepository); ok {
		service.authorRepository = authorRepository
	}
	return service, nil
}

// ListArticles returns one stable member-visible page.
func (s *ArticleQueryService) ListArticles(
	ctx context.Context,
	request ListArticlesRequest,
) (ListArticlesPage, error) {
	if s == nil || s.repository == nil || s.identity == nil {
		return ListArticlesPage{}, articleQueryInternalError(
			"list articles",
			errors.New("article query service is not configured"),
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
	return s.scanVisibleArticles(ctx, scanAfter, limit)
}

// GetArticle returns one member-visible detail and its strong ETag.
func (s *ArticleQueryService) GetArticle(
	ctx context.Context,
	articleID string,
) (MemberArticleDetailView, string, error) {
	if s == nil || s.repository == nil || s.identity == nil {
		return MemberArticleDetailView{}, "", articleQueryInternalError(
			"get article",
			errors.New("article query service is not configured"),
		)
	}
	if !validUUID(articleID) {
		return MemberArticleDetailView{}, "", ErrArticleUnavailable
	}
	record, err := s.repository.GetArticle(ctx, articleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MemberArticleDetailView{}, "", ErrArticleUnavailable
	}
	if err != nil {
		return MemberArticleDetailView{}, "", articleQueryDatabaseError(
			"get article record",
			err,
		)
	}
	statuses, err := s.readAuthorStates(ctx, []Article{record})
	if err != nil {
		return MemberArticleDetailView{}, "", err
	}
	detail, err := (VisibilityPolicy{}).MemberDetailProjection(
		record,
		articleAuthorStatus(record, statuses),
	)
	if err != nil {
		return MemberArticleDetailView{}, "", err
	}
	etag, err := StrongETag(record)
	if err != nil {
		return MemberArticleDetailView{}, "", articleQueryInternalError(
			"create article detail ETag",
			err,
		)
	}
	return detail, etag, nil
}

// GetAuthorArticle returns the strict author-controlled projection and ETag.
//
// It deliberately ignores member visibility: the current author may read their
// own active Article even when it is violating or otherwise member-invisible,
// but only through the minimal AuthorProjection allowlist.
func (s *ArticleQueryService) GetAuthorArticle(
	ctx context.Context,
	principal session.Principal,
	articleID string,
) (AuthorArticleView, string, error) {
	if s == nil || s.repository == nil {
		return AuthorArticleView{}, "", articleQueryInternalError(
			"get author article",
			errors.New("article query service is not configured"),
		)
	}
	if !validUUID(articleID) {
		return AuthorArticleView{}, "", ErrArticleUnavailable
	}
	record, err := s.repository.GetArticle(ctx, articleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthorArticleView{}, "", ErrArticleUnavailable
	}
	if err != nil {
		return AuthorArticleView{}, "", articleQueryDatabaseError(
			"get author article record",
			err,
		)
	}
	view, err := (VisibilityPolicy{}).AuthorProjection(record, principal)
	if err != nil {
		return AuthorArticleView{}, "", err
	}
	etag, err := StrongETag(record)
	if err != nil {
		return AuthorArticleView{}, "", articleQueryInternalError(
			"create author article ETag",
			err,
		)
	}
	return view, etag, nil
}

// ListCandidateBatch returns one stable descending keyset batch.
func (r *PostgresArticleQueryRepository) ListCandidateBatch(
	ctx context.Context,
	after *ArticleCursor,
	limit int,
) ([]Article, error) {
	if r == nil || r.database == nil || limit <= 0 {
		return nil, errors.New("article candidate query input is invalid")
	}
	var cursorCreatedAt any
	var cursorID any
	if after != nil {
		if !candidateCursorView(after.View) ||
			after.CreatedAt.IsZero() ||
			!validUUID(after.ID) {
			return nil, errors.New("article candidate cursor is invalid")
		}
		cursorCreatedAt = after.CreatedAt.UTC()
		cursorID = after.ID
	}
	rows, err := r.database.Query(
		ctx,
		listArticleCandidatesSQL,
		cursorCreatedAt,
		cursorID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query article candidates: %w", err)
	}
	defer rows.Close()

	articles := make([]Article, 0, limit)
	for rows.Next() {
		article, err := scanQueryArticle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan article candidate: %w", err)
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read article candidates: %w", err)
	}
	return articles, nil
}

// GetArticle returns one Article record without applying visibility.
func (r *PostgresArticleQueryRepository) GetArticle(
	ctx context.Context,
	articleID string,
) (Article, error) {
	if r == nil || r.database == nil || !validUUID(articleID) {
		return Article{}, pgx.ErrNoRows
	}
	article, err := scanQueryArticle(r.database.QueryRow(
		ctx,
		getArticleQuerySQL,
		articleID,
	))
	if err != nil {
		return Article{}, fmt.Errorf("query article detail: %w", err)
	}
	return article, nil
}

func (s *ArticleQueryService) resolveLimit(requested *int) (int, error) {
	if requested == nil {
		return s.config.DefaultLimit, nil
	}
	if *requested <= 0 {
		return 0, articleQueryLimitValidationError(
			"must_be_positive",
			errors.New("article query limit must be positive"),
		)
	}
	if *requested > s.config.MaxLimit {
		return 0, articleQueryLimitValidationError(
			"too_large",
			errors.New("article query limit exceeds configured maximum"),
		)
	}
	return *requested, nil
}

func decodeMemberCursor(encoded string) (*ArticleCursor, error) {
	if encoded == "" {
		return nil, nil
	}
	cursor, err := (CursorCodec{}).Decode(encoded, CursorViewMember)
	if err != nil {
		return nil, err
	}
	return &cursor, nil
}

func (s *ArticleQueryService) scanVisibleArticles(
	ctx context.Context,
	scanAfter *ArticleCursor,
	limit int,
) (ListArticlesPage, error) {
	visible := make([]MemberArticleSummaryView, 0, limit+1)
	for len(visible) < limit+1 {
		candidates, exhausted, err := s.readCandidateBatch(ctx, scanAfter)
		if err != nil {
			return ListArticlesPage{}, err
		}
		if len(candidates) == 0 {
			break
		}
		statuses, err := s.readAuthorStates(ctx, candidates)
		if err != nil {
			return ListArticlesPage{}, err
		}
		items, nextScanAfter, err := projectVisibleCandidates(
			candidates,
			statuses,
			limit+1-len(visible),
		)
		if err != nil {
			return ListArticlesPage{}, err
		}
		visible = append(visible, items...)
		scanAfter = nextScanAfter
		if len(visible) == limit+1 || exhausted {
			break
		}
	}
	return buildListArticlesPage(visible, limit)
}

func (s *ArticleQueryService) readCandidateBatch(
	ctx context.Context,
	scanAfter *ArticleCursor,
) ([]Article, bool, error) {
	candidates, err := s.repository.ListCandidateBatch(
		ctx,
		scanAfter,
		s.config.CandidateBatchSize,
	)
	if err != nil {
		return nil, false, articleQueryDatabaseError(
			"list article candidates",
			err,
		)
	}
	if len(candidates) > s.config.CandidateBatchSize {
		return nil, false, articleQueryInternalError(
			"list article candidates",
			errors.New("article repository exceeded candidate batch size"),
		)
	}
	if err := validateCandidateOrder(scanAfter, candidates); err != nil {
		return nil, false, articleQueryInternalError(
			"validate article candidate order",
			err,
		)
	}
	return candidates, len(candidates) < s.config.CandidateBatchSize, nil
}

func projectVisibleCandidates(
	candidates []Article,
	statuses map[string]string,
	limit int,
) ([]MemberArticleSummaryView, *ArticleCursor, error) {
	visible := make([]MemberArticleSummaryView, 0, limit)
	var scanAfter *ArticleCursor
	for _, candidate := range candidates {
		scanAfter = articleCursor(candidate)
		item, err := (VisibilityPolicy{}).MemberListProjection(
			candidate,
			articleAuthorStatus(candidate, statuses),
		)
		if errors.Is(err, ErrArticleUnavailable) {
			continue
		}
		if err != nil {
			return nil, nil, articleQueryInternalError(
				"project article list item",
				err,
			)
		}
		visible = append(visible, item)
		if len(visible) == limit {
			break
		}
	}
	return visible, scanAfter, nil
}

func (s *ArticleQueryService) readAuthorStates(
	ctx context.Context,
	articles []Article,
) (map[string]string, error) {
	authorIDs := uniqueArticleAuthorIDs(articles)
	if len(authorIDs) == 0 {
		return map[string]string{}, nil
	}
	statuses, err := s.identity(ctx, authorIDs)
	if err != nil {
		return nil, articleQueryDependencyError("read article author states", err)
	}
	if statuses == nil {
		return map[string]string{}, nil
	}
	return statuses, nil
}

func uniqueArticleAuthorIDs(articles []Article) []string {
	seen := make(map[string]struct{}, len(articles))
	ids := make([]string, 0, len(articles))
	for _, article := range articles {
		if article.AuthorID == nil {
			continue
		}
		if _, exists := seen[*article.AuthorID]; exists {
			continue
		}
		seen[*article.AuthorID] = struct{}{}
		ids = append(ids, *article.AuthorID)
	}
	return ids
}

func articleAuthorStatus(
	article Article,
	statuses map[string]string,
) string {
	if article.AuthorID == nil {
		return ""
	}
	return statuses[*article.AuthorID]
}

func validateCandidateOrder(
	after *ArticleCursor,
	candidates []Article,
) error {
	previous := after
	for _, candidate := range candidates {
		current := articleCursorValue(candidate)
		if current.CreatedAt.IsZero() ||
			current.CreatedAt.Location() != time.UTC ||
			!validUUID(current.ID) ||
			previous != nil && !cursorStrictlyAfter(*previous, current) {
			return errors.New("article candidates are not in strict descending order")
		}
		previous = &current
	}
	return nil
}

func cursorStrictlyAfter(previous ArticleCursor, current ArticleCursor) bool {
	return current.CreatedAt.Before(previous.CreatedAt) ||
		current.CreatedAt.Equal(previous.CreatedAt) && current.ID < previous.ID
}

func articleCursor(article Article) *ArticleCursor {
	cursor := articleCursorValue(article)
	return &cursor
}

func articleCursorValue(article Article) ArticleCursor {
	return ArticleCursor{
		CreatedAt: article.CreatedAt,
		ID:        article.ID,
		View:      CursorViewMember,
	}
}

func buildListArticlesPage(
	visible []MemberArticleSummaryView,
	limit int,
) (ListArticlesPage, error) {
	hasMore := len(visible) > limit
	if hasMore {
		visible = visible[:limit]
	}
	page := ListArticlesPage{
		Items: append([]MemberArticleSummaryView(nil), visible...),
	}
	if page.Items == nil {
		page.Items = []MemberArticleSummaryView{}
	}
	if !hasMore {
		return page, nil
	}
	last := page.Items[len(page.Items)-1]
	nextCursor, err := (CursorCodec{}).Encode(ArticleCursor{
		CreatedAt: last.CreatedAt,
		ID:        last.ID,
		View:      CursorViewMember,
	})
	if err != nil {
		return ListArticlesPage{}, articleQueryInternalError(
			"encode article list cursor",
			err,
		)
	}
	page.NextCursor = &nextCursor
	return page, nil
}

func scanQueryArticle(row interface{ Scan(...any) error }) (Article, error) {
	var article Article
	err := row.Scan(
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
	if err != nil {
		return Article{}, err
	}
	article.CreatedAt = article.CreatedAt.UTC()
	article.UpdatedAt = article.UpdatedAt.UTC()
	return article, nil
}

func articleQueryLimitValidationError(code string, cause error) error {
	return platform.NewPublicError(
		CodeArticleValidationFailed,
		cause,
		platform.FieldError{Field: "limit", Code: code},
	)
}

func articleQueryInternalError(operation string, cause error) error {
	return articleQueryPublicError(
		platform.CodePlatformInternalError,
		operation,
		cause,
	)
}

func articleQueryDatabaseError(operation string, cause error) error {
	return articleQueryPublicError(
		platform.CodePlatformDatabaseUnavailable,
		operation,
		cause,
	)
}

func articleQueryDependencyError(operation string, cause error) error {
	return articleQueryPublicError(
		platform.CodePlatformDependencyUnavailable,
		operation,
		cause,
	)
}

func articleQueryPublicError(
	code platform.Code,
	operation string,
	cause error,
) error {
	if cause == nil {
		cause = errors.New("unknown article query failure")
	}
	return platform.NewPublicError(
		code,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

// OrphanArticleRepository is the administrator-only read seam.
type OrphanArticleRepository interface {
	ListOrphanBatch(context.Context, *ArticleCursor, int) ([]Article, error)
	GetArticle(context.Context, string) (Article, error)
}

// OrphanArticlesPage is the administrator orphan list envelope.
type OrphanArticlesPage struct {
	Items      []AdminArticleView `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

// OrphanQueryService reads held, active orphan articles without member filtering.
type OrphanQueryService struct {
	repository OrphanArticleRepository
	config     QueryConfig
}

// NewOrphanQueryService keeps the member query contract unchanged.
func NewOrphanQueryService(repository OrphanArticleRepository, config QueryConfig) (*OrphanQueryService, error) {
	if repository == nil || config.DefaultLimit <= 0 || config.MaxLimit < config.DefaultLimit {
		return nil, errors.New("orphan query dependencies are invalid")
	}
	return &OrphanQueryService{repository: repository, config: config}, nil
}

func requireOrphanAdministrator(principal session.Principal) error {
	if !principal.Allows(session.AccessGeneral) || principal.Role != principalRoleAdmin {
		return ErrArticleForbidden
	}
	return nil
}

// ListOrphanArticles returns only the current administrator disposition set.
func (service *OrphanQueryService) ListOrphanArticles(ctx context.Context, principal session.Principal, request ListArticlesRequest) (OrphanArticlesPage, error) {
	if err := requireOrphanAdministrator(principal); err != nil {
		return OrphanArticlesPage{}, err
	}
	if service == nil || service.repository == nil {
		return OrphanArticlesPage{}, errors.New("orphan query is not configured")
	}
	limit := service.config.DefaultLimit
	if request.Limit != nil {
		limit = *request.Limit
	}
	if limit <= 0 || limit > service.config.MaxLimit {
		return OrphanArticlesPage{}, articleQueryLimitError(errors.New("orphan limit is out of bounds"))
	}
	var after *ArticleCursor
	if request.Cursor != "" {
		cursor, err := (CursorCodec{}).Decode(request.Cursor, CursorViewAdminOrphan)
		if err != nil {
			return OrphanArticlesPage{}, err
		}
		after = &cursor
	}
	rows, err := service.repository.ListOrphanBatch(ctx, after, limit+1)
	if err != nil {
		return OrphanArticlesPage{}, articleQueryDatabaseError("list orphan articles", err)
	}
	if err := validateCandidateOrder(after, rows); err != nil {
		return OrphanArticlesPage{}, articleQueryInternalError("order orphan articles", err)
	}
	return buildOrphanArticlesPage(rows, principal, limit)
}

func buildOrphanArticlesPage(rows []Article, principal session.Principal, limit int) (OrphanArticlesPage, error) {
	page := OrphanArticlesPage{Items: make([]AdminArticleView, 0)}
	visible := rows
	if len(visible) > limit {
		visible = visible[:limit]
	}
	for _, record := range visible {
		if !isDispositionOrphan(record) {
			return OrphanArticlesPage{}, ErrArticleInvalidState
		}
		view, err := (VisibilityPolicy{}).AdminProjection(record, principal, "")
		if err != nil {
			return OrphanArticlesPage{}, err
		}
		page.Items = append(page.Items, view)
	}
	if len(rows) > limit {
		last := visible[len(visible)-1]
		cursor, err := (CursorCodec{}).Encode(ArticleCursor{ID: last.ID, CreatedAt: last.CreatedAt, View: CursorViewAdminOrphan})
		if err != nil {
			return OrphanArticlesPage{}, err
		}
		page.NextCursor = &cursor
	}
	return page, nil
}

// GetOrphanArticle returns the administrator projection and its strong ETag.
func (service *OrphanQueryService) GetOrphanArticle(ctx context.Context, principal session.Principal, articleID string) (AdminArticleView, string, error) {
	if err := requireOrphanAdministrator(principal); err != nil {
		return AdminArticleView{}, "", err
	}
	if service == nil || service.repository == nil {
		return AdminArticleView{}, "", errors.New("orphan query is not configured")
	}
	if !validUUID(articleID) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	record, err := service.repository.GetArticle(ctx, articleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	if err != nil {
		return AdminArticleView{}, "", articleQueryDatabaseError("get orphan article", err)
	}
	if !isDispositionOrphan(record) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	view, err := (VisibilityPolicy{}).AdminProjection(record, principal, "")
	if err != nil {
		return AdminArticleView{}, "", err
	}
	etag, err := StrongETag(record)
	return view, etag, err
}

func isDispositionOrphan(record Article) bool {
	return record.AuthorID == nil && record.OrphanedFromUserID != nil && record.PublicationState == PublicationHeld && record.DeletionState == DeletionActive
}

// ListOrphanBatch filters the disposition set before applying the keyset limit.
func (repository *PostgresArticleQueryRepository) ListOrphanBatch(ctx context.Context, after *ArticleCursor, limit int) ([]Article, error) {
	if repository == nil || repository.database == nil || limit <= 0 {
		return nil, errors.New("orphan list input is invalid")
	}
	var createdAt, articleID any
	if after != nil {
		if after.View != CursorViewAdminOrphan {
			return nil, ErrInvalidCursor
		}
		createdAt, articleID = after.CreatedAt, after.ID
	}
	rows, err := repository.database.Query(ctx, listOrphanArticlesSQL, createdAt, articleID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []Article
	for rows.Next() {
		record, err := scanQueryArticle(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

const listOrphanArticlesSQL = `
	SELECT id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state, deletion_operation_id, version, created_at, updated_at
	FROM articles
	WHERE author_id IS NULL AND orphaned_from_user_id IS NOT NULL
		AND publication_state = 'held' AND deletion_state = 'active'
		AND ($1::timestamptz IS NULL OR (created_at, id) < ($1::timestamptz, $2::uuid))
	ORDER BY created_at DESC, id DESC LIMIT $3
`

const listArticleCandidatesSQL = `
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
	WHERE $1::timestamptz IS NULL
		OR created_at < $1::timestamptz
		OR (created_at = $1::timestamptz AND id < $2::uuid)
	ORDER BY created_at DESC, id DESC
	LIMIT $3
`

const getArticleQuerySQL = `
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
	WHERE id = $1::uuid
`

// AdminArticleListView selects a derived administrator projection set.
type AdminArticleListView string

const (
	// AdminListViewAll includes every valid deletion-active Article.
	AdminListViewAll AdminArticleListView = "all"
	// AdminListViewViolating includes Articles with a violation constraint.
	AdminListViewViolating AdminArticleListView = "violating"
	// AdminListViewHidden includes Articles with any derived hidden reason.
	AdminListViewHidden AdminArticleListView = "hidden"
)

// ListAdminArticlesRequest contains a view-bound administrator read request.
type ListAdminArticlesRequest struct {
	View   AdminArticleListView
	Cursor string
	Limit  *int
}

// AdminArticlesPage is the complete administrator list envelope.
type AdminArticlesPage struct {
	Items      []AdminArticleView `json:"items"`
	NextCursor *string            `json:"next_cursor"`
}

func candidateCursorView(view CursorView) bool {
	return view == CursorViewMember || view == CursorViewAdminAll ||
		view == CursorViewAdminViolating || view == CursorViewAdminHidden
}

func adminCursorView(view AdminArticleListView) (CursorView, error) {
	switch view {
	case "", AdminListViewAll:
		return CursorViewAdminAll, nil
	case AdminListViewViolating:
		return CursorViewAdminViolating, nil
	case AdminListViewHidden:
		return CursorViewAdminHidden, nil
	default:
		return "", ErrArticleValidationFailed
	}
}

// ListAdminArticles reads complete governance views with strict current identity facts.
func (s *ArticleQueryService) ListAdminArticles(ctx context.Context, principal session.Principal, request ListAdminArticlesRequest) (AdminArticlesPage, error) {
	if err := requireOrphanAdministrator(principal); err != nil {
		return AdminArticlesPage{}, err
	}
	if s == nil || s.repository == nil || s.identity == nil {
		return AdminArticlesPage{}, articleQueryInternalError("list admin articles", errors.New("admin query dependencies are unavailable"))
	}
	view, err := adminCursorView(request.View)
	if err != nil {
		return AdminArticlesPage{}, err
	}
	limit, err := s.resolveLimit(request.Limit)
	if err != nil {
		return AdminArticlesPage{}, err
	}
	var after *ArticleCursor
	if request.Cursor != "" {
		decoded, err := (CursorCodec{}).Decode(request.Cursor, view)
		if err != nil {
			return AdminArticlesPage{}, err
		}
		after = &decoded
	}
	return s.scanAdminArticles(ctx, principal, view, after, limit)
}

func (s *ArticleQueryService) scanAdminArticles(ctx context.Context, principal session.Principal, view CursorView, after *ArticleCursor, limit int) (AdminArticlesPage, error) {
	items := make([]AdminArticleView, 0, limit+1)
	for len(items) < limit+1 {
		if err := ctx.Err(); err != nil {
			return AdminArticlesPage{}, articleQueryDatabaseError("cancel admin scan", err)
		}
		candidates, exhausted, err := s.readCandidateBatch(ctx, after)
		if err != nil {
			return AdminArticlesPage{}, err
		}
		active, err := activeAdminCandidates(candidates)
		if err != nil {
			return AdminArticlesPage{}, err
		}
		states, err := s.readAdminAuthorStates(ctx, active)
		if err != nil {
			return AdminArticlesPage{}, err
		}
		for _, record := range candidates {
			after = &ArticleCursor{CreatedAt: record.CreatedAt, ID: record.ID, View: view}
			if record.DeletionState == DeletionPending {
				continue
			}
			projected, err := (VisibilityPolicy{}).AdminProjection(record, principal, articleAuthorStatus(record, states))
			if err != nil {
				return AdminArticlesPage{}, adminReadStateError()
			}
			if adminViewMatches(view, projected) {
				items = append(items, projected)
			}
			if len(items) == limit+1 {
				break
			}
		}
		if exhausted || len(candidates) == 0 {
			break
		}
	}
	return buildAdminArticlesPage(items, limit, view)
}

func activeAdminCandidates(candidates []Article) ([]Article, error) {
	active := make([]Article, 0, len(candidates))
	for _, record := range candidates {
		if record.DeletionState == DeletionPending {
			continue
		}
		if ValidateArticleState(record) != nil || record.DeletionState != DeletionActive {
			return nil, adminReadStateError()
		}
		active = append(active, record)
	}
	return active, nil
}

func (s *ArticleQueryService) readAdminAuthorStates(ctx context.Context, records []Article) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, articleQueryDatabaseError("cancel admin identity read", err)
	}
	states, err := s.readAuthorStates(ctx, records)
	if err != nil {
		return nil, err
	}
	for _, id := range uniqueArticleAuthorIDs(records) {
		if states[id] != authorStatusActive && states[id] != authorStatusSuspended {
			return nil, articleQueryDependencyError("validate admin author states", errors.New("admin author state is incomplete"))
		}
	}
	return states, nil
}

func adminViewMatches(view CursorView, item AdminArticleView) bool {
	return view == CursorViewAdminAll ||
		view == CursorViewAdminViolating && item.ViolationState == ViolationViolating ||
		view == CursorViewAdminHidden && len(item.HiddenReasons) != 0
}

func buildAdminArticlesPage(items []AdminArticleView, limit int, view CursorView) (AdminArticlesPage, error) {
	page := AdminArticlesPage{Items: items}
	if len(items) <= limit {
		return page, nil
	}
	page.Items = items[:limit]
	last := page.Items[len(page.Items)-1]
	encoded, err := (CursorCodec{}).Encode(ArticleCursor{CreatedAt: last.CreatedAt, ID: last.ID, View: view})
	if err != nil {
		return AdminArticlesPage{}, articleQueryInternalError("encode admin cursor", err)
	}
	page.NextCursor = &encoded
	return page, nil
}

func adminReadStateError() error {
	// Never retain ErrArticleInvalidState: the shared mapper assigns it governance 409.
	return articleQueryInternalError("validate admin article", errors.New("admin article state is invalid"))
}

// GetAdminArticle returns one complete active Article and its same-record ETag.
func (s *ArticleQueryService) GetAdminArticle(ctx context.Context, principal session.Principal, id string) (AdminArticleView, string, error) {
	if err := requireOrphanAdministrator(principal); err != nil {
		return AdminArticleView{}, "", err
	}
	if s == nil || s.repository == nil || s.identity == nil {
		return AdminArticleView{}, "", articleQueryInternalError("get admin article", errors.New("admin query dependencies are unavailable"))
	}
	if !validUUID(id) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	record, err := s.repository.GetArticle(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	if err != nil {
		return AdminArticleView{}, "", articleQueryDatabaseError("get admin article", err)
	}
	if record.DeletionState == DeletionPending {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	if ValidateArticleState(record) != nil || record.DeletionState != DeletionActive {
		return AdminArticleView{}, "", adminReadStateError()
	}
	states, err := s.readAdminAuthorStates(ctx, []Article{record})
	if err != nil {
		return AdminArticleView{}, "", err
	}
	result, err := (VisibilityPolicy{}).AdminProjection(record, principal, articleAuthorStatus(record, states))
	if err != nil {
		return AdminArticleView{}, "", adminReadStateError()
	}
	etag, err := StrongETag(record)
	if err != nil {
		return AdminArticleView{}, "", articleQueryInternalError("encode admin detail ETag", err)
	}
	return result, etag, nil
}

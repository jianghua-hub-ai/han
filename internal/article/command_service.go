package article

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	maxArticleTitleRunes                 = 200
	maxArticleBodyRunes                  = 20000
	canonicalOrphanAuditAction           = "article.orphan.canonical"
	canonicalOrphanAuditActorSystem      = "identity-canonical-delete"
	canonicalOrphanAuditTargetType       = "article"
	canonicalOrphanAuditOutcome          = "succeeded"
	canonicalOrphanAuditDeletedUserIDKey = "deleted_user_id"
	canonicalOrphanAuditVersionKey       = "article_version"
)

var errCreateArticleCommitOutcomeUnknown = errors.New(
	"create article transaction closed with commit outcome unknown",
)

var errReplaceArticleCommitOutcomeUnknown = errors.New(
	"replace article transaction closed with commit outcome unknown",
)

// CreateArticleInput contains the only client-selected Article fields.
type CreateArticleInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ArticleTransactor begins a caller-owned Article transaction.
type ArticleTransactor interface {
	Begin(context.Context) (pgx.Tx, error)
}

// CreateArticleRepository persists a new Article in the caller-owned transaction.
type CreateArticleRepository interface {
	InsertArticle(context.Context, pgx.Tx, Article) error
}

// ArticleContentRepository owns the locked read and content-only write used by
// author content replacement. The production Repository satisfies it without
// widening CreateArticleRepository.
type ArticleContentRepository interface {
	LockArticleForUpdate(context.Context, pgx.Tx, string) (Article, error)
	UpdateArticleContent(context.Context, pgx.Tx, Article) error
}

// ArticleIDGenerator creates opaque Article identifiers.
type ArticleIDGenerator interface {
	NewID() (string, error)
}

// ArticleClock supplies Article timestamps.
type ArticleClock interface {
	Now() time.Time
}

// CommandDependencies are the explicit Article command collaborators.
type CommandDependencies struct {
	Transactions ArticleTransactor
	Repository   CreateArticleRepository
	IDs          ArticleIDGenerator
	Clock        ArticleClock
	Orphaning    *OrphanArticlesCanonicalDependencies
	Deletion     PermanentArticleDeleter
	Moderation   *ModerationDependencies
	Disposition  *OrphanDispositionDependencies
}

type ModerationDependencies struct {
	Auditor      DeletionAuditor
	AuthorStates func(context.Context, pgx.Tx, []string) (map[string]string, error)
}

type articleModerationRepository interface {
	LockArticleForUpdate(context.Context, pgx.Tx, string) (Article, error)
	UpdateArticleModeration(context.Context, pgx.Tx, Article) error
}

// CommandService applies Article command rules and transaction ownership.
type CommandService struct {
	dependencies CommandDependencies
}

// DeletePermanent executes the shared irreversible deletion protocol.
func (s *CommandService) DeletePermanent(
	ctx context.Context,
	principal session.Principal,
	request PermanentDeletionRequest,
) error {
	if s == nil || s.dependencies.Deletion == nil {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("permanent deletion is not configured"),
		)
	}
	return s.dependencies.Deletion.DeletePermanent(ctx, principal, request)
}

// NewCommandService creates an Article command service.
func NewCommandService(dependencies CommandDependencies) (*CommandService, error) {
	switch {
	case dependencies.Transactions == nil:
		return nil, errors.New("article transaction boundary is required")
	case dependencies.Repository == nil:
		return nil, errors.New("article repository is required")
	case dependencies.IDs == nil:
		return nil, errors.New("article ID generator is required")
	case dependencies.Clock == nil:
		return nil, errors.New("article clock is required")
	default:
		if dependencies.Disposition != nil && (dependencies.Disposition.Identity == nil ||
			dependencies.Disposition.Auditor == nil || dependencies.Disposition.RecipientMaxAttempts < 1) {
			return nil, errors.New("orphan disposition dependencies are incomplete")
		}
		if dependencies.Moderation != nil {
			if _, available := dependencies.Repository.(articleModerationRepository); !available ||
				dependencies.Moderation.Auditor == nil || dependencies.Moderation.AuthorStates == nil {
				return nil, errors.New("article moderation dependencies are incomplete")
			}
		}
		if err := validateOrphanArticlesCanonicalDependencies(dependencies.Orphaning); err != nil {
			return nil, err
		}
		return &CommandService{dependencies: dependencies}, nil
	}
}

func (service *CommandService) Moderate(
	ctx context.Context, principal session.Principal, articleID string,
	expectedVersion int64, category *ViolationCategory,
) (AdminArticleView, string, error) {
	if !principal.Allows(session.AccessGeneral) || principal.Role != principalRoleAdmin {
		return AdminArticleView{}, "", ErrArticleForbidden
	}
	if !validUUID(articleID) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	if category != nil && *category != ViolationViolence && *category != ViolationSexual {
		return AdminArticleView{}, "", ErrArticleValidationFailed
	}
	if service == nil || service.dependencies.Moderation == nil {
		return AdminArticleView{}, "", platform.NewPublicError(platform.CodePlatformInternalError,
			errors.New("article moderation is not configured"))
	}
	transaction, err := service.dependencies.Transactions.Begin(ctx)
	if err != nil || transaction == nil {
		if err == nil {
			err = errors.New("article moderation transaction is required")
		}
		return AdminArticleView{}, "", articleDatabaseError("begin article moderation", err)
	}
	return service.moderateTransaction(ctx, transaction, principal, articleID, expectedVersion, category)
}

func (service *CommandService) moderateTransaction(
	ctx context.Context, transaction pgx.Tx, principal session.Principal,
	articleID string, expectedVersion int64, category *ViolationCategory,
) (AdminArticleView, string, error) {
	transactionClosed := false
	defer func() {
		if !transactionClosed {
			_ = transaction.Rollback(ctx)
		}
	}()
	payload, etag, err := service.moderateLocked(ctx, transaction, principal, articleID, expectedVersion, category)
	if err != nil {
		transactionClosed = true
		if rollbackErr := transaction.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = articleDatabaseError("rollback article moderation", errors.Join(err, rollbackErr))
		}
		return AdminArticleView{}, "", err
	}
	transactionClosed = true
	if err := transaction.Commit(ctx); err != nil {
		return AdminArticleView{}, "", articleDatabaseError("commit article moderation; outcome unknown", err)
	}
	return payload, etag, nil
}

func (service *CommandService) moderateLocked(
	ctx context.Context, transaction pgx.Tx, principal session.Principal,
	articleID string, expectedVersion int64, category *ViolationCategory,
) (AdminArticleView, string, error) {
	repository := service.dependencies.Repository.(articleModerationRepository)
	existing, err := repository.LockArticleForUpdate(ctx, transaction, articleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminArticleView{}, "", ErrArticleUnavailable
	}
	if err != nil {
		return AdminArticleView{}, "", articleDatabaseError("lock article moderation", err)
	}
	if err := moderationAllowed(existing, expectedVersion, category); err != nil {
		return AdminArticleView{}, "", err
	}
	updated := existing
	updated.ViolationState, updated.ViolationCategory = ViolationClear, nil
	if category != nil {
		updated.ViolationState, updated.ViolationCategory = ViolationViolating, category
	}
	updated.Version++
	updated.UpdatedAt = service.dependencies.Clock.Now().UTC()
	if updated.UpdatedAt.IsZero() {
		return AdminArticleView{}, "", platform.NewPublicError(platform.CodePlatformInternalError,
			errors.New("article moderation clock returned zero time"))
	}
	payload, etag, err := service.moderationProjection(ctx, transaction, principal, updated)
	if err != nil {
		return AdminArticleView{}, "", err
	}
	if err := repository.UpdateArticleModeration(ctx, transaction, updated); err != nil {
		return AdminArticleView{}, "", articleDatabaseError("update article moderation", err)
	}
	if err := service.appendModerationAudit(ctx, transaction, principal, existing, updated); err != nil {
		return AdminArticleView{}, "", err
	}
	return payload, etag, nil
}

func (service *CommandService) moderationProjection(
	ctx context.Context, transaction pgx.Tx, principal session.Principal, updated Article,
) (AdminArticleView, string, error) {
	authorStatus := ""
	if updated.AuthorID != nil {
		states, err := service.dependencies.Moderation.AuthorStates(ctx, transaction, []string{*updated.AuthorID})
		if err != nil {
			return AdminArticleView{}, "", platform.NewPublicError(platform.CodePlatformDependencyUnavailable,
				fmt.Errorf("read moderation author state: %w", err))
		}
		authorStatus = states[*updated.AuthorID]
		if authorStatus != authorStatusActive && authorStatus != authorStatusSuspended {
			return AdminArticleView{}, "", platform.NewPublicError(platform.CodePlatformDependencyUnavailable,
				errors.New("moderation author state is unavailable"))
		}
	}
	payload, err := (VisibilityPolicy{}).AdminProjection(updated, principal, authorStatus)
	if err != nil {
		return AdminArticleView{}, "", err
	}
	etag, err := StrongETag(updated)
	return payload, etag, err
}

const (
	moderationAuditMarkAction  = "article.violation.mark"
	moderationAuditClearAction = "article.violation.clear"
	moderationAuditCategoryKey = "category"
)

func (service *CommandService) appendModerationAudit(
	ctx context.Context, transaction pgx.Tx, principal session.Principal, existing, updated Article,
) error {
	traceID, available := platform.TraceIDFromContext(ctx)
	if !available {
		return platform.NewPublicError(platform.CodePlatformInternalError, errors.New("moderation trace ID is required"))
	}
	auditID, err := service.dependencies.IDs.NewID()
	if err != nil || !validUUID(auditID) {
		if err == nil {
			err = errors.New("generated moderation audit ID is invalid")
		}
		return platform.NewPublicError(platform.CodePlatformInternalError,
			fmt.Errorf("generate moderation audit ID: %w", err))
	}
	action, category := moderationAuditClearAction, existing.ViolationCategory
	if updated.ViolationCategory != nil {
		action, category = moderationAuditMarkAction, updated.ViolationCategory
	}
	event := audit.Event{
		ID: auditID, OccurredAt: updated.UpdatedAt, ActorKind: audit.ActorUser,
		ActorUserID: &principal.UserID, ActorRole: &principal.Role,
		Action: action, TargetType: deletionAuditTargetType, TargetID: updated.ID,
		Outcome: deletionAuditOutcome, TraceID: traceID,
		Metadata: map[string]any{
			deletionAuditVersionKey: updated.Version, moderationAuditCategoryKey: *category,
		},
	}
	if err := service.dependencies.Moderation.Auditor.Append(ctx, transaction, event); err != nil {
		return articleDatabaseError("append article moderation audit", err)
	}
	return nil
}

func (Repository) UpdateArticleModeration(ctx context.Context, transaction pgx.Tx, updated Article) error {
	if err := requireTransaction(transaction); err != nil {
		return err
	}
	tag, err := transaction.Exec(ctx, updateArticleModerationSQL,
		updated.ID, updated.ViolationState, updated.ViolationCategory, updated.Version, updated.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update article moderation: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrPreconditionFailed
	}
	return nil
}

const updateArticleModerationSQL = `
	UPDATE articles
	SET violation_state = $2, violation_category = $3, version = $4, updated_at = $5
	WHERE id = $1::uuid AND version = $4 - 1 AND deletion_state = 'active'
`

func validateOrphanArticlesCanonicalDependencies(
	dependencies *OrphanArticlesCanonicalDependencies,
) error {
	if dependencies == nil {
		return nil
	}
	switch {
	case dependencies.Repository == nil:
		return errors.New("canonical orphan article repository is required")
	case dependencies.Auditor == nil:
		return errors.New("canonical orphan article auditor is required")
	case dependencies.AuditIDs == nil:
		return errors.New("canonical orphan audit ID generator is required")
	case dependencies.Clock == nil:
		return errors.New("canonical orphan article clock is required")
	default:
		return nil
	}
}

// OrphanArticlesCanonical applies article-owned canonical deletion state in tx.
func (s *CommandService) OrphanArticlesCanonical(
	ctx context.Context,
	tx pgx.Tx,
	deletedUserID string,
) (OrphanArticlesResult, error) {
	if s == nil {
		return OrphanArticlesResult{}, errors.New("article command service is required")
	}
	if tx == nil {
		return OrphanArticlesResult{}, errors.New("canonical orphan transaction is required")
	}
	if deletedUserID == "" {
		return OrphanArticlesResult{}, errors.New("deleted user ID is required")
	}
	dependencies := s.dependencies.Orphaning
	if dependencies == nil {
		return OrphanArticlesResult{}, errors.New("canonical orphan dependencies are required")
	}
	articles, err := dependencies.Repository.LockCanonicalOrphanCandidates(
		ctx,
		tx,
		deletedUserID,
	)
	if err != nil {
		return OrphanArticlesResult{}, fmt.Errorf(
			"lock canonical orphan candidates: %w",
			err,
		)
	}
	return s.orphanCanonicalCandidates(ctx, tx, deletedUserID, articles)
}

func (s *CommandService) orphanCanonicalCandidates(
	ctx context.Context,
	tx pgx.Tx,
	deletedUserID string,
	articles []Article,
) (OrphanArticlesResult, error) {
	var result OrphanArticlesResult
	if !hasCanonicalOrphanUpdate(articles, deletedUserID) {
		return countCanonicalOrphans(articles, deletedUserID), nil
	}
	traceID, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return result, errors.New("canonical orphan trace ID is required")
	}
	now := s.dependencies.Orphaning.Clock.Now().UTC()
	if now.IsZero() {
		return result, errors.New("canonical orphan clock returned zero time")
	}
	for _, article := range articles {
		if canonicalOrphanNeedsUpdate(article, deletedUserID) {
			if err := s.orphanCanonicalArticle(
				ctx, tx, deletedUserID, traceID, now, article,
			); err != nil {
				return OrphanArticlesResult{}, err
			}
		}
	}
	return countCanonicalOrphansAfterUpdate(articles, deletedUserID), nil
}

func (s *CommandService) orphanCanonicalArticle(
	ctx context.Context,
	tx pgx.Tx,
	deletedUserID string,
	traceID string,
	now time.Time,
	article Article,
) error {
	auditID, err := s.dependencies.Orphaning.AuditIDs.NewID()
	if err != nil || !validUUID(auditID) {
		if err == nil {
			err = errors.New("generated canonical orphan audit ID is invalid")
		}
		return fmt.Errorf("generate canonical orphan audit ID: %w", err)
	}
	update := CanonicalOrphanUpdate{
		ArticleID:       article.ID,
		DeletedUserID:   deletedUserID,
		ExpectedVersion: article.Version,
		UpdatedAt:       now,
	}
	if err := s.dependencies.Orphaning.Repository.SetCanonicalOrphan(
		ctx, tx, update,
	); err != nil {
		return fmt.Errorf("set canonical orphan article %s: %w", article.ID, err)
	}
	event := canonicalOrphanAuditEvent(auditID, traceID, deletedUserID, now, article)
	if err := s.dependencies.Orphaning.Auditor.Append(ctx, tx, event); err != nil {
		return fmt.Errorf("append canonical orphan audit for article %s: %w", article.ID, err)
	}
	return nil
}

func canonicalOrphanAuditEvent(
	id string,
	traceID string,
	deletedUserID string,
	now time.Time,
	article Article,
) audit.Event {
	system := canonicalOrphanAuditActorSystem
	return audit.Event{
		ID:          id,
		OccurredAt:  now,
		ActorKind:   audit.ActorSystem,
		ActorSystem: &system,
		Action:      canonicalOrphanAuditAction,
		TargetType:  canonicalOrphanAuditTargetType,
		TargetID:    article.ID,
		Outcome:     canonicalOrphanAuditOutcome,
		TraceID:     traceID,
		Metadata: map[string]any{
			canonicalOrphanAuditDeletedUserIDKey: deletedUserID,
			canonicalOrphanAuditVersionKey:       article.Version + 1,
		},
	}
}

func hasCanonicalOrphanUpdate(articles []Article, deletedUserID string) bool {
	for _, article := range articles {
		if canonicalOrphanNeedsUpdate(article, deletedUserID) {
			return true
		}
	}
	return false
}

func countCanonicalOrphans(
	articles []Article,
	deletedUserID string,
) OrphanArticlesResult {
	var result OrphanArticlesResult
	for _, article := range articles {
		if !canonicalOrphanAssociated(article, deletedUserID) {
			continue
		}
		if article.DeletionState == DeletionPending {
			result.PendingOrphanCount++
			continue
		}
		if canonicalOrphanAlreadyComplete(article, deletedUserID) {
			result.AlreadyOrphanedCount++
			result.PendingOrphanCount++
		}
	}
	return result
}

func countCanonicalOrphansAfterUpdate(
	articles []Article,
	deletedUserID string,
) OrphanArticlesResult {
	result := countCanonicalOrphans(articles, deletedUserID)
	for _, article := range articles {
		if canonicalOrphanNeedsUpdate(article, deletedUserID) {
			result.NewlyOrphanedCount++
			result.PendingOrphanCount++
		}
	}
	return result
}

func canonicalOrphanNeedsUpdate(article Article, deletedUserID string) bool {
	return article.DeletionState == DeletionActive &&
		article.AuthorID != nil &&
		*article.AuthorID == deletedUserID
}

func canonicalOrphanAlreadyComplete(article Article, deletedUserID string) bool {
	return article.DeletionState == DeletionActive &&
		article.AuthorID == nil &&
		article.PublicationState == PublicationHeld &&
		article.OrphanedFromUserID != nil &&
		*article.OrphanedFromUserID == deletedUserID
}

func canonicalOrphanAssociated(article Article, deletedUserID string) bool {
	return article.AuthorID != nil && *article.AuthorID == deletedUserID ||
		article.OrphanedFromUserID != nil &&
			*article.OrphanedFromUserID == deletedUserID
}

// Create persists one published Article owned by the trusted principal.
func (s *CommandService) Create(
	ctx context.Context,
	principal session.Principal,
	input CreateArticleInput,
) (Article, error) {
	if s == nil {
		return Article{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article command service is required"),
		)
	}
	if !principal.Allows(session.AccessGeneral) {
		return Article{}, ErrArticleForbidden
	}
	if err := validateArticleContent(input); err != nil {
		return Article{}, err
	}

	id, err := s.dependencies.IDs.NewID()
	if err != nil || !validUUID(id) {
		if err == nil {
			err = errors.New("generated article ID is invalid")
		}
		return Article{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("generate article ID: %w", err),
		)
	}
	now := s.dependencies.Clock.Now().UTC()
	if now.IsZero() {
		return Article{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article clock returned zero time"),
		)
	}
	authorID := principal.UserID
	article := Article{
		ID:               id,
		AuthorID:         &authorID,
		Title:            input.Title,
		Body:             input.Body,
		PublicationState: PublicationPublished,
		ViolationState:   ViolationClear,
		DeletionState:    DeletionActive,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	tx, err := s.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return Article{}, articleDatabaseError("begin create article transaction", err)
	}
	if tx == nil {
		return Article{}, articleDatabaseError(
			"begin create article transaction",
			errors.New("article transaction is required"),
		)
	}
	transactionClosed := false
	defer func() {
		if !transactionClosed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := s.dependencies.Repository.InsertArticle(ctx, tx, article); err != nil {
		transactionClosed = true
		return Article{}, rollbackCreateArticle(
			ctx,
			tx,
			articleDatabaseError("insert article", err),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		transactionClosed = true
		return Article{}, articleDatabaseError(
			"commit create article transaction",
			errors.Join(errCreateArticleCommitOutcomeUnknown, err),
		)
	}
	transactionClosed = true
	return article, nil
}

func validateArticleContent(input CreateArticleInput) error {
	if invalidArticleText(input.Title, maxArticleTitleRunes) ||
		invalidArticleText(input.Body, maxArticleBodyRunes) {
		return ErrArticleValidationFailed
	}
	return nil
}

// ReplaceContent fully replaces the title and body of an author-owned Article.
//
// It enforces the author-edit contract of design §3.3.4: only the current author
// may edit, the Article must still be active, the strong If-Match version must
// match the locked row, and only title/body/updated_at/version change. Every
// governance, publication, ownership and deletion field is preserved verbatim.
func (s *CommandService) ReplaceContent(
	ctx context.Context,
	principal session.Principal,
	articleID string,
	expectedVersion int64,
	input CreateArticleInput,
) (Article, error) {
	if s == nil {
		return Article{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article command service is required"),
		)
	}
	content, ok := s.dependencies.Repository.(ArticleContentRepository)
	if !ok || content == nil {
		return Article{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article content repository is required"),
		)
	}
	if !principal.Allows(session.AccessGeneral) {
		return Article{}, ErrArticleForbidden
	}
	if !validUUID(articleID) {
		return Article{}, ErrArticleUnavailable
	}
	if err := validateArticleContent(input); err != nil {
		return Article{}, err
	}
	now := s.dependencies.Clock.Now().UTC()
	if now.IsZero() {
		return Article{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("article clock returned zero time"),
		)
	}

	tx, err := s.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return Article{}, articleDatabaseError("begin replace article transaction", err)
	}
	if tx == nil {
		return Article{}, articleDatabaseError(
			"begin replace article transaction",
			errors.New("article transaction is required"),
		)
	}
	transactionClosed := false
	defer func() {
		if !transactionClosed {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, err := content.LockArticleForUpdate(ctx, tx, articleID)
	if err != nil {
		transactionClosed = true
		if errors.Is(err, pgx.ErrNoRows) {
			return Article{}, rollbackArticleContent(ctx, tx, ErrArticleUnavailable)
		}
		return Article{}, rollbackArticleContent(
			ctx,
			tx,
			articleDatabaseError("lock article for content replacement", err),
		)
	}
	if err := authorContentEditAllowed(existing, principal); err != nil {
		transactionClosed = true
		return Article{}, rollbackArticleContent(ctx, tx, err)
	}
	if existing.Version != expectedVersion {
		transactionClosed = true
		return Article{}, rollbackArticleContent(ctx, tx, ErrPreconditionFailed)
	}

	updated := existing
	updated.Title = input.Title
	updated.Body = input.Body
	updated.Version = existing.Version + 1
	updated.UpdatedAt = now
	if err := content.UpdateArticleContent(ctx, tx, updated); err != nil {
		transactionClosed = true
		return Article{}, rollbackArticleContent(
			ctx,
			tx,
			articleDatabaseError("update article content", err),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		transactionClosed = true
		return Article{}, articleDatabaseError(
			"commit replace article transaction",
			errors.Join(errReplaceArticleCommitOutcomeUnknown, err),
		)
	}
	transactionClosed = true
	return updated, nil
}

// authorContentEditAllowed enforces the author-only edit state policy.
func authorContentEditAllowed(article Article, principal session.Principal) error {
	if article.DeletionState != DeletionActive {
		return ErrArticleUnavailable
	}
	if article.AuthorID != nil && *article.AuthorID == principal.UserID {
		return nil
	}
	if principal.Role == principalRoleAdmin {
		return ErrArticleForbidden
	}
	return ErrArticleUnavailable
}

func rollbackArticleContent(ctx context.Context, tx pgx.Tx, cause error) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return articleDatabaseError(
			"rollback replace article transaction",
			errors.Join(cause, err),
		)
	}
	return cause
}

// LockArticleForUpdate locks one Article row for author content replacement.
func (Repository) LockArticleForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	articleID string,
) (Article, error) {
	if err := requireTransaction(tx); err != nil {
		return Article{}, err
	}
	if !validUUID(articleID) {
		return Article{}, pgx.ErrNoRows
	}
	article, err := scanQueryArticle(
		tx.QueryRow(ctx, lockArticleForContentSQL, articleID),
	)
	if err != nil {
		return Article{}, err
	}
	return article, nil
}

// UpdateArticleContent replaces only the mutable author content columns.
//
// The guarded WHERE keeps the write a compare-and-set on the expected version;
// every other column (ownership, publication, governance, deletion) is untouched.
func (Repository) UpdateArticleContent(
	ctx context.Context,
	tx pgx.Tx,
	article Article,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	tag, err := tx.Exec(
		ctx,
		updateArticleContentSQL,
		article.ID,
		article.Title,
		article.Body,
		article.Version,
		article.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update article content: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrPreconditionFailed
	}
	return nil
}

var _ ArticleContentRepository = Repository{}

const lockArticleForContentSQL = `
	SELECT id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state, deletion_operation_id,
		version, created_at, updated_at
	FROM articles
	WHERE id = $1::uuid
	FOR UPDATE
`

const updateArticleContentSQL = `
	UPDATE articles
	SET title = $2, body = $3, version = $4, updated_at = $5
	WHERE id = $1::uuid AND version = $4 - 1
`

func invalidArticleText(value string, maxRunes int) bool {
	return strings.TrimFunc(value, unicode.IsSpace) == "" ||
		utf8.RuneCountInString(value) > maxRunes
}

func rollbackCreateArticle(ctx context.Context, tx pgx.Tx, cause error) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return articleDatabaseError(
			"rollback create article transaction",
			errors.Join(cause, err),
		)
	}
	return cause
}

func articleDatabaseError(operation string, cause error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

// DispositionParties contains identity contacts held until caller commit.
type DispositionParties struct {
	SourceID    string
	SourceEmail *string
	TargetID    *string
	TargetEmail *string
}

// DispositionDeletionCandidate is the locked identity lifecycle decision.
type DispositionDeletionCandidate struct {
	UserID  string
	Deleted bool
}

// DispositionIdentityPort is adapted to identity only by the composition root.
type DispositionIdentityPort interface {
	LockTransferParties(context.Context, pgx.Tx, *string, string) (DispositionParties, error)
	LockDeletionCandidate(context.Context, pgx.Tx, string) (DispositionDeletionCandidate, error)
	ClearRetainedEmailIfEligible(context.Context, pgx.Tx, string, *DispositionCompletionProof) (bool, error)
}

// OrphanDispositionDependencies enable publish/transfer independently of SMTP.
type OrphanDispositionDependencies struct {
	Identity             DispositionIdentityPort
	Auditor              DeletionAuditor
	RecipientMaxAttempts int64
}

// OrphanDispositionResult carries no contact snapshots or delivery promises.
type OrphanDispositionResult struct {
	Article  Article
	ETag     string
	NoticeID *string
}

// DispositionAction names one completed transition.
type DispositionAction string

const (
	DispositionPublish        DispositionAction = "publish"
	DispositionTransfer       DispositionAction = "transfer"
	DispositionDeleteComplete DispositionAction = "delete_complete"
	orphanPublishAuditAction                    = "article.orphan.publish"
	orphanTransferAuditAction                   = "article.orphan.transfer"
)

// DispositionArticleState is a content-free locked preimage or SQL readback.
type DispositionArticleState struct {
	Exists      bool
	ID          string
	Version     int64
	AuthorID    *string
	SourceID    *string
	Publication PublicationState
	Violation   ViolationState
	Category    *ViolationCategory
	Deletion    DeletionState
	OperationID *string
}

// DispositionCompletionReference is a value projection, never completion authority.
type DispositionCompletionReference struct {
	Transaction       pgx.Tx
	ExpectedArticleID string
	SourceID          string
	BeforeVersion     int64
	Action            DispositionAction
	NoticeID          *string
	NewAuthorID       *string
	OperationID       *string
	AuditEventID      string
	BeforeArticle     DispositionArticleState
}

// DispositionCompletionFacts are verified current-transaction readbacks.
type DispositionCompletionFacts struct {
	AfterArticle          DispositionArticleState
	RemainingActiveCount  int64
	RemainingPendingCount int64
	Notice                *TransferNotice
	Recipients            []TransferNoticeRecipient
	CompletedOperation    *DeletionOperation
	Audit                 audit.Event
}

// DispositionCompletionProof can only be created by locked article workflows.
// Root adapters capture this object; reconstructing it from public DTOs is invalid.
type DispositionCompletionProof struct {
	reference DispositionCompletionReference
	contacts  DispositionParties
	operation *DeletionOperation
	tombstone ArticleDeletionTombstone
	traceID   string
}

// Reference returns an independent value projection for the identity DTO mapping.
func (proof *DispositionCompletionProof) Reference() DispositionCompletionReference {
	if proof == nil {
		return DispositionCompletionReference{}
	}
	reference := proof.reference
	reference.NoticeID = copyDispositionString(reference.NoticeID)
	reference.NewAuthorID = copyDispositionString(reference.NewAuthorID)
	reference.OperationID = copyDispositionString(reference.OperationID)
	reference.BeforeArticle = copyDispositionState(reference.BeforeArticle)
	return reference
}

// VerifyCompletion uses the original transaction, locked preimage and contacts.
func (proof *DispositionCompletionProof) VerifyCompletion(
	ctx context.Context, transaction pgx.Tx,
) (DispositionCompletionFacts, error) {
	if proof == nil || transaction == nil || proof.reference.Transaction != transaction {
		return DispositionCompletionFacts{}, errors.New("disposition proof transaction mismatch")
	}
	traceID, present := platform.TraceIDFromContext(ctx)
	if !present || traceID != proof.traceID || !validUUID(proof.reference.AuditEventID) {
		return DispositionCompletionFacts{}, errors.New("disposition proof trace or audit binding mismatch")
	}
	facts, err := (Repository{}).ReadDispositionCompletion(ctx, transaction, proof.reference)
	if err != nil {
		return DispositionCompletionFacts{}, err
	}
	if err := proof.validateCompletion(facts); err != nil {
		return DispositionCompletionFacts{}, err
	}
	return facts, nil
}

// PublishOrphan releases a clear held orphan as an anonymous published article.
func (service *CommandService) PublishOrphan(
	ctx context.Context, principal session.Principal, articleID string, expectedVersion int64,
) (OrphanDispositionResult, error) {
	return service.disposeOrphan(ctx, principal, articleID, expectedVersion, nil)
}

// TransferOrphan transfers ownership while retaining the violation classification.
func (service *CommandService) TransferOrphan(
	ctx context.Context, principal session.Principal, articleID string, expectedVersion int64, targetID string,
) (OrphanDispositionResult, error) {
	return service.disposeOrphan(ctx, principal, articleID, expectedVersion, &targetID)
}

func (service *CommandService) disposeOrphan(
	ctx context.Context, principal session.Principal, articleID string, expectedVersion int64, targetID *string,
) (OrphanDispositionResult, error) {
	if !principal.Allows(session.AccessGeneral) || principal.Role != principalRoleAdmin {
		return OrphanDispositionResult{}, ErrArticleForbidden
	}
	if targetID != nil && !validUUID(*targetID) {
		return OrphanDispositionResult{}, ErrTransferTargetInvalid
	}
	if !validUUID(articleID) {
		return OrphanDispositionResult{}, ErrArticleUnavailable
	}
	if service == nil || service.dependencies.Disposition == nil {
		return OrphanDispositionResult{}, errors.New("orphan disposition is not configured")
	}
	transaction, err := service.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return OrphanDispositionResult{}, articleDatabaseError("begin orphan disposition", err)
	}
	if transaction == nil {
		return OrphanDispositionResult{}, errors.New("orphan disposition transaction is required")
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if err := ensureDispositionReadCommitted(ctx, transaction); err != nil {
		return OrphanDispositionResult{}, err
	}
	result, err := service.disposeOrphanLocked(ctx, transaction, principal, articleID, expectedVersion, targetID)
	if err != nil {
		return OrphanDispositionResult{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return OrphanDispositionResult{}, articleDatabaseError("commit orphan disposition; outcome unknown", err)
	}
	return result, nil
}

func (service *CommandService) disposeOrphanLocked(
	ctx context.Context, transaction pgx.Tx, principal session.Principal,
	articleID string, expectedVersion int64, targetID *string,
) (OrphanDispositionResult, error) {
	located, err := (Repository{}).LocateDispositionArticle(ctx, transaction, articleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrphanDispositionResult{}, ErrArticleUnavailable
	}
	if err != nil {
		return OrphanDispositionResult{}, articleDatabaseError("locate orphan article", err)
	}
	if located.SourceID == nil {
		return OrphanDispositionResult{}, ErrArticleInvalidState
	}
	sourceID := *located.SourceID
	parties, err := service.dependencies.Disposition.Identity.LockTransferParties(ctx, transaction, targetID, sourceID)
	if err != nil {
		return OrphanDispositionResult{}, err
	}
	if err := validateDispositionParties(parties, sourceID, targetID); err != nil {
		return OrphanDispositionResult{}, err
	}
	before, err := (Repository{}).LockArticleForDisposition(ctx, transaction, articleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrphanDispositionResult{}, ErrArticleUnavailable
	}
	if err != nil {
		return OrphanDispositionResult{}, articleDatabaseError("lock orphan article NOWAIT", err)
	}
	if err := dispositionAllowed(before, sourceID, expectedVersion, targetID); err != nil {
		return OrphanDispositionResult{}, err
	}
	return service.applyOrphanDisposition(ctx, transaction, principal, before, parties)
}

func dispositionAllowed(before Article, sourceID string, expectedVersion int64, targetID *string) error {
	if before.DeletionState != DeletionActive {
		return ErrArticleUnavailable
	}
	if !sameDispositionString(before.OrphanedFromUserID, &sourceID) || before.AuthorID != nil ||
		before.PublicationState != PublicationHeld {
		return ErrArticleInvalidState
	}
	if before.Version != expectedVersion {
		return ErrPreconditionFailed
	}
	if before.Version == math.MaxInt64 {
		return ErrArticleInvalidState
	}
	if targetID == nil && before.ViolationState != ViolationClear {
		return ErrArticleInvalidState
	}
	return ValidateArticleState(before)
}

func validateDispositionParties(parties DispositionParties, sourceID string, targetID *string) error {
	if parties.SourceID != sourceID || !sameDispositionString(parties.TargetID, targetID) {
		return errors.New("disposition identity binding mismatch")
	}
	if targetID == nil {
		return nil
	}
	if !validDispositionEmail(parties.SourceEmail) || !validDispositionEmail(parties.TargetEmail) {
		return errors.New("disposition transfer contacts are required")
	}
	return nil
}

func validDispositionEmail(email *string) bool {
	if email == nil || strings.TrimSpace(*email) != *email {
		return false
	}
	parsed, err := mail.ParseAddress(*email)
	return err == nil && parsed.Name == "" && strings.EqualFold(parsed.Address, *email)
}

func (service *CommandService) applyOrphanDisposition(
	ctx context.Context, transaction pgx.Tx, principal session.Principal, before Article, parties DispositionParties,
) (OrphanDispositionResult, error) {
	now := service.dependencies.Clock.Now().UTC()
	traceID, present := platform.TraceIDFromContext(ctx)
	if now.IsZero() || !present {
		return OrphanDispositionResult{}, errors.New("disposition time and trace are required")
	}
	auditID, err := service.dependencies.IDs.NewID()
	if err != nil {
		return OrphanDispositionResult{}, fmt.Errorf("generate disposition audit ID: %w", err)
	}
	if !validUUID(auditID) {
		return OrphanDispositionResult{}, errors.New("disposition audit ID is invalid")
	}
	proof := newDispositionProof(transaction, before, parties, auditID, traceID)
	if parties.TargetID != nil {
		noticeID, err := service.insertDispositionNotice(ctx, transaction, before.ID, parties, now)
		if err != nil {
			return OrphanDispositionResult{}, err
		}
		proof.reference.NoticeID = &noticeID
	}
	updated := before
	updated.AuthorID, updated.OrphanedFromUserID = copyDispositionString(parties.TargetID), nil
	updated.PublicationState, updated.Version, updated.UpdatedAt = PublicationPublished, before.Version+1, now
	if err := (Repository{}).UpdateOrphanDisposition(ctx, transaction, before, updated); err != nil {
		return OrphanDispositionResult{}, articleDatabaseError("update orphan disposition", err)
	}
	event := audit.Event{ID: auditID, OccurredAt: now, ActorKind: audit.ActorUser,
		ActorUserID: &principal.UserID, ActorRole: &principal.Role, Action: orphanPublishAuditAction,
		TargetType: deletionAuditTargetType, TargetID: before.ID, Outcome: deletionAuditOutcome, TraceID: traceID,
		Metadata: map[string]any{deletionAuditVersionKey: updated.Version}}
	if parties.TargetID != nil {
		event.Action = orphanTransferAuditAction
	}
	if err := service.dependencies.Disposition.Auditor.Append(ctx, transaction, event); err != nil {
		return OrphanDispositionResult{}, fmt.Errorf("append orphan disposition audit: %w", err)
	}
	if _, err := proof.VerifyCompletion(ctx, transaction); err != nil {
		return OrphanDispositionResult{}, err
	}
	if _, err := service.dependencies.Disposition.Identity.ClearRetainedEmailIfEligible(
		ctx, transaction, parties.SourceID, proof); err != nil {
		return OrphanDispositionResult{}, err
	}
	etag, err := StrongETag(updated)
	return OrphanDispositionResult{Article: updated, ETag: etag, NoticeID: copyDispositionString(proof.reference.NoticeID)}, err
}

func (service *CommandService) insertDispositionNotice(
	ctx context.Context, transaction pgx.Tx, articleID string, parties DispositionParties, now time.Time,
) (string, error) {
	noticeID, err := service.dependencies.IDs.NewID()
	if err != nil {
		return "", fmt.Errorf("generate transfer notice ID: %w", err)
	}
	if !validUUID(noticeID) {
		return "", errors.New("transfer notice ID is invalid")
	}
	notice := TransferNotice{ID: noticeID, ArticleRef: articleID, NewAuthorID: *parties.TargetID,
		FormerDeletedUserID: parties.SourceID, CreatedAt: now, UpdatedAt: now}
	recipients := [2]TransferNoticeRecipient{
		{NoticeID: noticeID, Kind: RecipientNewAuthor, EmailSnapshot: *parties.TargetEmail,
			Status: RecipientPending, MaxAttempts: service.dependencies.Disposition.RecipientMaxAttempts, CreatedAt: now, UpdatedAt: now},
		{NoticeID: noticeID, Kind: RecipientFormerDeletedAuthor, EmailSnapshot: *parties.SourceEmail,
			Status: RecipientPending, MaxAttempts: service.dependencies.Disposition.RecipientMaxAttempts, CreatedAt: now, UpdatedAt: now},
	}
	return noticeID, (Repository{}).InsertTransferNotice(ctx, transaction, notice, recipients)
}

func newDispositionProof(
	transaction pgx.Tx, before Article, parties DispositionParties, auditID, traceID string,
) *DispositionCompletionProof {
	proof := &DispositionCompletionProof{traceID: traceID, contacts: DispositionParties{SourceID: parties.SourceID,
		SourceEmail: copyDispositionString(parties.SourceEmail), TargetID: copyDispositionString(parties.TargetID),
		TargetEmail: copyDispositionString(parties.TargetEmail)}}
	proof.reference = DispositionCompletionReference{Transaction: transaction, ExpectedArticleID: before.ID,
		SourceID: parties.SourceID, BeforeVersion: before.Version, Action: DispositionPublish,
		NewAuthorID: copyDispositionString(parties.TargetID), AuditEventID: auditID, BeforeArticle: dispositionState(before)}
	if parties.TargetID != nil {
		proof.reference.Action = DispositionTransfer
	}
	return proof
}

func dispositionState(article Article) DispositionArticleState {
	state := DispositionArticleState{Exists: true, ID: article.ID, Version: article.Version,
		AuthorID: article.AuthorID, SourceID: article.OrphanedFromUserID, Publication: article.PublicationState,
		Violation: article.ViolationState, Category: article.ViolationCategory, Deletion: article.DeletionState,
		OperationID: article.DeletionOperationID}
	return copyDispositionState(state)
}

func copyDispositionState(state DispositionArticleState) DispositionArticleState {
	state.AuthorID, state.SourceID = copyDispositionString(state.AuthorID), copyDispositionString(state.SourceID)
	state.OperationID = copyDispositionString(state.OperationID)
	if state.Category != nil {
		category := *state.Category
		state.Category = &category
	}
	return state
}

func copyDispositionString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func sameDispositionString(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func (proof *DispositionCompletionProof) validateCompletion(facts DispositionCompletionFacts) error {
	reference := proof.reference
	if facts.RemainingActiveCount < 0 || facts.RemainingPendingCount < 0 ||
		facts.Audit.ID != reference.AuditEventID || facts.Audit.TargetID != reference.ExpectedArticleID ||
		facts.Audit.TargetType != deletionAuditTargetType || facts.Audit.Outcome != deletionAuditOutcome ||
		facts.Audit.TraceID != proof.traceID || facts.Audit.OccurredAt.IsZero() {
		return errors.New("disposition completion audit or count mismatch")
	}
	if reference.Action == DispositionDeleteComplete {
		return proof.validateDeletionFacts(facts)
	}
	before, after := reference.BeforeArticle, facts.AfterArticle
	if !before.Exists || before.ID != reference.ExpectedArticleID || before.Version != reference.BeforeVersion ||
		before.AuthorID != nil || !sameDispositionString(before.SourceID, &reference.SourceID) ||
		before.Publication != PublicationHeld || before.Deletion != DeletionActive || before.OperationID != nil ||
		!after.Exists || after.ID != before.ID || after.Version != before.Version+1 || after.SourceID != nil ||
		after.Publication != PublicationPublished || after.Deletion != DeletionActive || after.OperationID != nil ||
		facts.CompletedOperation != nil || !validDispositionState(before) || !validDispositionState(after) {
		return errors.New("disposition completion article mismatch")
	}
	if reference.Action == DispositionTransfer {
		return proof.validateTransferFacts(facts)
	}
	if reference.Action != DispositionPublish || reference.NewAuthorID != nil || reference.NoticeID != nil ||
		after.AuthorID != nil || before.Violation != ViolationClear || after.Violation != ViolationClear ||
		facts.Notice != nil || len(facts.Recipients) != 0 || facts.Audit.Action != orphanPublishAuditAction {
		return errors.New("disposition publish completion mismatch")
	}
	return nil
}

func validDispositionState(state DispositionArticleState) bool {
	return ValidateArticleState(Article{AuthorID: state.AuthorID, OrphanedFromUserID: state.SourceID,
		PublicationState: state.Publication, ViolationState: state.Violation, ViolationCategory: state.Category,
		DeletionState: state.Deletion, DeletionOperationID: state.OperationID, Version: state.Version}) == nil
}

func (proof *DispositionCompletionProof) validateTransferFacts(facts DispositionCompletionFacts) error {
	reference, notice := proof.reference, facts.Notice
	if reference.NoticeID == nil || reference.NewAuthorID == nil || notice == nil ||
		notice.ID != *reference.NoticeID || notice.ArticleRef != reference.ExpectedArticleID ||
		notice.NewAuthorID != *reference.NewAuthorID || notice.FormerDeletedUserID != reference.SourceID ||
		!sameDispositionString(facts.AfterArticle.AuthorID, reference.NewAuthorID) ||
		facts.AfterArticle.Violation != reference.BeforeArticle.Violation ||
		!sameDispositionCategory(facts.AfterArticle.Category, reference.BeforeArticle.Category) ||
		facts.Audit.Action != orphanTransferAuditAction || len(facts.Recipients) != 2 {
		return errors.New("disposition transfer associations mismatch")
	}
	contacts := map[RecipientKind]*string{RecipientNewAuthor: proof.contacts.TargetEmail,
		RecipientFormerDeletedAuthor: proof.contacts.SourceEmail}
	for _, recipient := range facts.Recipients {
		expected, present := contacts[recipient.Kind]
		if !present || !validDispositionEmail(expected) || recipient.EmailSnapshot != *expected ||
			recipient.NoticeID != notice.ID || recipient.Status != RecipientPending {
			return errors.New("disposition transfer contact snapshot mismatch")
		}
		delete(contacts, recipient.Kind)
	}
	return nil
}

func sameDispositionCategory(left, right *ViolationCategory) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func (proof *DispositionCompletionProof) validateDeletionFacts(facts DispositionCompletionFacts) error {
	reference, operation := proof.reference, facts.CompletedOperation
	if proof.operation == nil || reference.OperationID == nil || operation == nil || facts.AfterArticle.Exists ||
		facts.Notice != nil || len(facts.Recipients) != 0 || operation.ID != *reference.OperationID ||
		operation.ArticleID != reference.ExpectedArticleID || operation.ArticleVersion != reference.BeforeVersion ||
		operation.DeleteKind != proof.operation.DeleteKind ||
		!sameDispositionString(operation.OrphanedFromUserID, proof.operation.OrphanedFromUserID) ||
		operation.Status != DeletionOperationCompleted || operation.CompletedAt == nil ||
		!sameDispositionString(operation.AuditEventID, &reference.AuditEventID) ||
		operation.ClaimedBy != nil || operation.LockedUntil != nil ||
		facts.Audit.Action != deletionAuditAction(operation.DeleteKind) {
		return errors.New("disposition deletion completion mismatch")
	}
	if err := requireTombstoneBinding(*operation, proof.tombstone); err != nil {
		return err
	}
	before := reference.BeforeArticle
	if before.Exists && (before.ID != operation.ArticleID || before.Version != operation.ArticleVersion+1 ||
		before.Deletion != DeletionPending || !sameDispositionString(before.OperationID, &operation.ID) ||
		!validDispositionState(before)) {
		return ErrDeletionArticleMismatch
	}
	if operation.DeleteKind == DeleteOrphan {
		if !sameDispositionString(operation.OrphanedFromUserID, &reference.SourceID) || before.Exists &&
			(before.AuthorID != nil || !sameDispositionString(before.SourceID, &reference.SourceID) ||
				before.Publication != PublicationHeld) {
			return ErrDeletionArticleMismatch
		}
	} else if (operation.DeleteKind != DeleteOrdinary && operation.DeleteKind != DeleteViolating) ||
		!before.Exists || operation.OrphanedFromUserID != nil || before.SourceID != nil ||
		!sameDispositionString(before.AuthorID, &reference.SourceID) {
		return ErrDeletionArticleMismatch
	}
	return nil
}

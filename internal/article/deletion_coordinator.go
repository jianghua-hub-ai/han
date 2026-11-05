package article

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

// PermanentDeletionRequest identifies one authorized deletion attempt.
type PermanentDeletionRequest struct {
	ArticleID  string
	IfMatch    string
	DeleteKind DeleteKind
	TraceID    string
}

// PermanentArticleDeleter executes the shared irreversible deletion protocol.
type PermanentArticleDeleter interface {
	DeletePermanent(context.Context, session.Principal, PermanentDeletionRequest) error
}

// PermanentDeletionRepository owns T1 and journal-binding persistence.
type PermanentDeletionRepository interface {
	LockArticleForDeletion(context.Context, pgx.Tx, string) (Article, *DeletionOperation, error)
	InsertDeletionOperation(context.Context, pgx.Tx, DeletionOperation) error
	SetArticleDeletionPending(
		context.Context, pgx.Tx, string, int64, string, time.Time,
	) error
	MarkDeletionOperationJournaled(
		context.Context, pgx.Tx, string, ArticleDeletionTombstone,
	) error
}

// DeletionOperationFinalizer completes one journaled operation.
type DeletionOperationFinalizer interface {
	Finalize(context.Context, string, string) error
}

// PermanentDeletionDependencies are the coordinator collaborators.
type PermanentDeletionDependencies struct {
	Transactions ArticleTransactor
	Repository   PermanentDeletionRepository
	Journal      ArticleDeletionJournal
	Finalizer    DeletionOperationFinalizer
	OperationIDs DeletionIDGenerator
	Clock        DeletionClock
}

// PermanentDeletionCoordinator coordinates T1, durable journal, and T2.
type PermanentDeletionCoordinator struct {
	dependencies PermanentDeletionDependencies
}

// NewPermanentDeletionCoordinator creates the shared deletion coordinator.
func NewPermanentDeletionCoordinator(
	dependencies PermanentDeletionDependencies,
) (*PermanentDeletionCoordinator, error) {
	switch {
	case dependencies.Transactions == nil:
		return nil, errors.New("permanent deletion transaction boundary is required")
	case dependencies.Repository == nil:
		return nil, errors.New("permanent deletion repository is required")
	case dependencies.Journal == nil:
		return nil, errors.New("permanent deletion journal is required")
	case dependencies.Finalizer == nil:
		return nil, errors.New("permanent deletion finalizer is required")
	case dependencies.OperationIDs == nil:
		return nil, errors.New("permanent deletion operation ID generator is required")
	case dependencies.Clock == nil:
		return nil, errors.New("permanent deletion clock is required")
	default:
		return &PermanentDeletionCoordinator{dependencies: dependencies}, nil
	}
}

// DeletePermanent returns nil only after the journal and physical deletion complete.
func (coordinator *PermanentDeletionCoordinator) DeletePermanent(
	ctx context.Context,
	principal session.Principal,
	request PermanentDeletionRequest,
) error {
	if coordinator == nil {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("permanent deletion coordinator is required"),
		)
	}
	if !validUUID(request.ArticleID) || strings.TrimSpace(request.TraceID) == "" {
		return ErrArticleUnavailable
	}
	operation, err := coordinator.beginOrLoad(ctx, principal, request)
	if err != nil {
		return err
	}
	tombstone, err := coordinator.dependencies.Journal.Append(
		ctx,
		DeletionTombstoneRequest{
			ArticleID: operation.ArticleID, ArticleVersion: operation.ArticleVersion,
			DeletionOperationID: operation.ID, DeletedAt: operation.StartedAt,
		},
	)
	if err != nil {
		return errors.Join(ErrDeletionInProgress, fmt.Errorf("append deletion tombstone: %w", err))
	}
	if operation.Status == DeletionOperationPendingJournal {
		if err := coordinator.markJournaled(ctx, operation.ID, tombstone); err != nil {
			return errors.Join(ErrDeletionInProgress, err)
		}
	}
	if err := coordinator.dependencies.Finalizer.Finalize(
		ctx, operation.ID, request.TraceID,
	); err != nil {
		return errors.Join(ErrDeletionInProgress, err)
	}
	return nil
}

func (coordinator *PermanentDeletionCoordinator) beginOrLoad(
	ctx context.Context,
	principal session.Principal,
	request PermanentDeletionRequest,
) (DeletionOperation, error) {
	tx, err := coordinator.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return DeletionOperation{}, articleDatabaseError("begin permanent deletion", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	article, existing, err := coordinator.dependencies.Repository.LockArticleForDeletion(
		ctx, tx, request.ArticleID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeletionOperation{}, ErrArticleUnavailable
	}
	if err != nil {
		return DeletionOperation{}, articleDatabaseError("lock article for deletion", err)
	}
	if article.DeletionState == DeletionPending {
		if existing == nil {
			return DeletionOperation{}, ErrArticleInvalidState
		}
		if err := authorizePendingDeletion(principal, request, *existing); err != nil {
			return DeletionOperation{}, err
		}
		return *existing, nil
	}
	if err := authorizeActiveDeletion(principal, request.DeleteKind, article); err != nil {
		return DeletionOperation{}, err
	}
	if err := RequireMatchingETag(request.IfMatch, article); err != nil {
		return DeletionOperation{}, err
	}
	operationID, err := coordinator.dependencies.OperationIDs.NewID()
	if err != nil || !validUUID(operationID) {
		if err == nil {
			err = errors.New("generated deletion operation ID is invalid")
		}
		return DeletionOperation{}, fmt.Errorf("generate deletion operation ID: %w", err)
	}
	now := coordinator.dependencies.Clock.Now().UTC().Truncate(time.Microsecond)
	if now.IsZero() {
		return DeletionOperation{}, errors.New("permanent deletion clock returned zero time")
	}
	actorID, actorRole := principal.UserID, principal.Role
	operation := DeletionOperation{
		ID: operationID, ArticleID: article.ID, ArticleVersion: article.Version,
		DeleteKind: request.DeleteKind, ActorUserID: &actorID, ActorRole: &actorRole,
		OrphanedFromUserID: article.OrphanedFromUserID,
		Status:             DeletionOperationPendingJournal, StartedAt: now,
	}
	if err := coordinator.dependencies.Repository.InsertDeletionOperation(
		ctx, tx, operation,
	); err != nil {
		return DeletionOperation{}, articleDatabaseError("insert deletion operation", err)
	}
	if err := coordinator.dependencies.Repository.SetArticleDeletionPending(
		ctx, tx, article.ID, article.Version, operation.ID, now,
	); err != nil {
		return DeletionOperation{}, articleDatabaseError("hide article for deletion", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return DeletionOperation{}, articleDatabaseError("commit deletion pending", err)
	}
	return operation, nil
}

func (coordinator *PermanentDeletionCoordinator) markJournaled(
	ctx context.Context,
	operationID string,
	tombstone ArticleDeletionTombstone,
) error {
	tx, err := coordinator.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin journal binding: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := coordinator.dependencies.Repository.MarkDeletionOperationJournaled(
		ctx, tx, operationID, tombstone,
	); err != nil {
		return fmt.Errorf("bind deletion tombstone: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit deletion tombstone binding: %w", err)
	}
	return nil
}

func authorizeActiveDeletion(
	principal session.Principal,
	kind DeleteKind,
	article Article,
) error {
	if !principal.Allows(session.AccessGeneral) {
		return ErrArticleForbidden
	}
	if ValidateArticleState(article) != nil || article.DeletionState != DeletionActive {
		return ErrArticleInvalidState
	}
	switch kind {
	case DeleteOrdinary:
		ordinary := article.AuthorID != nil &&
			article.PublicationState == PublicationPublished &&
			article.ViolationState == ViolationClear
		if !ordinary {
			return ErrArticleInvalidState
		}
		if *article.AuthorID == principal.UserID || principal.Role == principalRoleAdmin {
			return nil
		}
		return ErrArticleUnavailable
	case DeleteViolating:
		if principal.Role != principalRoleAdmin {
			return ErrArticleForbidden
		}
		if article.PublicationState != PublicationPublished || article.OrphanedFromUserID != nil ||
			article.ViolationState != ViolationViolating {
			return ErrArticleInvalidState
		}
		return nil
	case DeleteOrphan:
		if principal.Role != principalRoleAdmin {
			return ErrArticleForbidden
		}
		if article.AuthorID != nil || article.OrphanedFromUserID == nil ||
			article.PublicationState != PublicationHeld {
			return ErrArticleInvalidState
		}
		return nil
	default:
		return ErrArticleInvalidState
	}
}

func authorizePendingDeletion(
	principal session.Principal,
	request PermanentDeletionRequest,
	operation DeletionOperation,
) error {
	if operation.ActorUserID == nil || operation.ActorRole == nil ||
		*operation.ActorUserID != principal.UserID ||
		*operation.ActorRole != principal.Role ||
		operation.DeleteKind != request.DeleteKind {
		return ErrArticleUnavailable
	}
	id, version, err := parseStrongETag(request.IfMatch)
	if err != nil {
		return err
	}
	if id != operation.ArticleID || version != operation.ArticleVersion {
		return ErrPreconditionFailed
	}
	return nil
}

const lockArticleForDeletionSQL = `
	SELECT id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state, deletion_operation_id,
		version, created_at, updated_at
	FROM articles
	WHERE id = $1
	FOR UPDATE
`

// LockArticleForDeletion locks an Article and its existing operation when pending.
func (Repository) LockArticleForDeletion(
	ctx context.Context,
	tx pgx.Tx,
	articleID string,
) (Article, *DeletionOperation, error) {
	if err := requireTransaction(tx); err != nil {
		return Article{}, nil, err
	}
	var article Article
	err := tx.QueryRow(ctx, lockArticleForDeletionSQL, articleID).Scan(
		&article.ID, &article.AuthorID, &article.OrphanedFromUserID,
		&article.Title, &article.Body, &article.PublicationState,
		&article.ViolationState, &article.ViolationCategory,
		&article.DeletionState, &article.DeletionOperationID,
		&article.Version, &article.CreatedAt, &article.UpdatedAt,
	)
	if err != nil || article.DeletionOperationID == nil {
		return article, nil, err
	}
	operation, err := (Repository{}).LoadDeletionOperationForUpdate(
		ctx, tx, *article.DeletionOperationID,
	)
	if err != nil {
		return Article{}, nil, err
	}
	return article, &operation, nil
}

// SetArticleDeletionPending performs the T1 visibility transition.
func (Repository) SetArticleDeletionPending(
	ctx context.Context,
	tx pgx.Tx,
	articleID string,
	expectedVersion int64,
	operationID string,
	updatedAt time.Time,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE articles
		SET deletion_state = 'deletion_pending', deletion_operation_id = $3,
			version = version + 1, updated_at = $4
		WHERE id = $1 AND version = $2 AND deletion_state = 'active'
	`, articleID, expectedVersion, operationID, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrPreconditionFailed
	}
	return nil
}

// MarkDeletionOperationJournaled binds the durable tombstone to its operation.
func (Repository) MarkDeletionOperationJournaled(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
	tombstone ArticleDeletionTombstone,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE deletion_operations
		SET status = 'journaled', tombstone_sequence = $2,
			journal_key_id = $3, journal_key_version = $4
		WHERE id = $1 AND status = 'pending_journal'
	`, operationID, tombstone.Sequence, tombstone.KeyID, tombstone.KeyVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("deletion operation journal binding was not pending")
	}
	return nil
}

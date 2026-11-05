package article

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
)

const (
	deletionAuditTargetType        = "article"
	deletionAuditOutcome           = "succeeded"
	deletionAuditVersionKey        = "article_version"
	deletionAuditKindKey           = "delete_kind"
	deletionAuditAuthorityKey      = "authority"
	deletionAuditAdministrator     = "administrator"
	deletionReplayAuditAction      = "article.deletion.replay"
	deletionReplayAuditActorSystem = "article-deletion-replay"
)

var (
	// ErrDeletionArticleMismatch marks a present Article that is not bound to
	// the finalizing operation and therefore cannot be completed safely.
	ErrDeletionArticleMismatch = errors.New("article deletion target mismatch")
)

var errDeletionLeaseCompetition = errors.New("article deletion lease competition")

// PendingArticleDeletionResult distinguishes idempotent absence from a
// present Article that did not match the operation-bound delete predicate.
type PendingArticleDeletionResult string

const (
	PendingArticleDeleted         PendingArticleDeletionResult = "deleted"
	PendingArticleAlreadyAbsent   PendingArticleDeletionResult = "already_absent"
	PendingArticlePresentMismatch PendingArticleDeletionResult = "present_mismatch"
)

// DeletionFinalizerRepository owns finalizer lease and completion persistence.
type DeletionFinalizerRepository interface {
	LockArticleForDisposition(context.Context, pgx.Tx, string) (Article, error)
	LoadDeletionOperationForUpdate(context.Context, pgx.Tx, string) (DeletionOperation, error)
	MarkDeletionOperationJournaled(
		context.Context, pgx.Tx, string, ArticleDeletionTombstone,
	) error
	ClaimDeletionOperation(
		context.Context, pgx.Tx, string, string, time.Time, time.Time,
	) (DeletionOperation, bool, error)
	DeletePendingArticle(
		context.Context, pgx.Tx, string, int64, string,
	) (PendingArticleDeletionResult, error)
	CompleteDeletionOperation(
		context.Context, pgx.Tx, string, string, string, time.Time,
	) error
	InsertReplayedDeletionOperation(context.Context, pgx.Tx, DeletionOperation) error
}

// DeletionAuditor appends the final deletion audit in the caller-owned transaction.
type DeletionAuditor interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

// DeletionIDGenerator creates opaque operation or audit identifiers.
type DeletionIDGenerator interface {
	NewID() (string, error)
}

// DeletionClock supplies permanent-deletion timestamps.
type DeletionClock interface {
	Now() time.Time
}

// DeletionFinalizerDependencies are the explicit finalizer collaborators.
type DeletionFinalizerDependencies struct {
	Transactions  ArticleTransactor
	Repository    DeletionFinalizerRepository
	Journal       ArticleDeletionJournal
	Auditor       DeletionAuditor
	AuditIDs      DeletionIDGenerator
	Clock         DeletionClock
	WorkerID      string
	LeaseDuration time.Duration
	Identity      DispositionIdentityPort
}

// DeletionFinalizer physically deletes one pending Article and completes its operation.
type DeletionFinalizer struct {
	dependencies DeletionFinalizerDependencies
}

// NewDeletionFinalizer creates an idempotent deletion finalizer.
func NewDeletionFinalizer(
	dependencies DeletionFinalizerDependencies,
) (*DeletionFinalizer, error) {
	switch {
	case dependencies.Transactions == nil:
		return nil, errors.New("deletion finalizer transaction boundary is required")
	case dependencies.Repository == nil:
		return nil, errors.New("deletion finalizer repository is required")
	case dependencies.Journal == nil:
		return nil, errors.New("deletion finalizer journal is required")
	case dependencies.Auditor == nil:
		return nil, errors.New("deletion finalizer auditor is required")
	case dependencies.AuditIDs == nil:
		return nil, errors.New("deletion finalizer audit ID generator is required")
	case dependencies.Clock == nil:
		return nil, errors.New("deletion finalizer clock is required")
	case strings.TrimSpace(dependencies.WorkerID) == "":
		return nil, errors.New("deletion finalizer worker ID is required")
	case dependencies.LeaseDuration <= 0:
		return nil, errors.New("deletion finalizer lease duration is required")
	default:
		return &DeletionFinalizer{dependencies: dependencies}, nil
	}
}

// Finalize converges one journaled operation; nil means completion is durable.
func (finalizer *DeletionFinalizer) Finalize(
	ctx context.Context,
	operationID string,
	traceID string,
) error {
	_, err := finalizer.finalizeWithResult(ctx, operationID, ArticleDeletionTombstone{}, traceID)
	return err
}

// ConvergeDeletionOperation repairs a pending journal binding and finalizes it.
func (finalizer *DeletionFinalizer) ConvergeDeletionOperation(
	ctx context.Context,
	operationID string,
	traceID string,
) error {
	if finalizer == nil || !validUUID(operationID) || strings.TrimSpace(traceID) == "" {
		return errors.New("deletion finalizer input is invalid")
	}
	operation, err := finalizer.loadOperation(ctx, operationID)
	if err != nil {
		return err
	}
	switch operation.Status {
	case DeletionOperationCompleted:
		return nil
	case DeletionOperationPendingJournal:
		tombstone, appendErr := finalizer.dependencies.Journal.Append(
			ctx,
			DeletionTombstoneRequest{
				ArticleID:           operation.ArticleID,
				ArticleVersion:      operation.ArticleVersion,
				DeletionOperationID: operation.ID,
				DeletedAt:           operation.StartedAt,
			},
		)
		if appendErr != nil {
			return fmt.Errorf("append deletion tombstone during convergence: %w", appendErr)
		}
		if err := finalizer.bindTombstone(ctx, operation, tombstone); err != nil {
			return err
		}
	case DeletionOperationJournaled, DeletionOperationFinalizing:
	default:
		return errors.Join(
			ErrDeletionWorkerTerminal,
			errors.New("deletion operation has an invalid status"),
		)
	}
	return finalizer.Finalize(ctx, operationID, traceID)
}

// DeletionReplayAllResult reports the idempotent replay outcome counts.
type DeletionReplayAllResult struct {
	Replayed      int64
	AlreadyAbsent int64
}

// ReplayAllTombstones idempotently converges every supplied tombstone.
//
// It is the source-side ReplayAll entry: each tombstone is reconciled through
// the existing journaled/finalizer path, so a replayed row is physically
// deleted once and an absent row is counted idempotently.
func (finalizer *DeletionFinalizer) ReplayAllTombstones(
	ctx context.Context,
	tombstones []ArticleDeletionTombstone,
	traceID string,
) (DeletionReplayAllResult, error) {
	var result DeletionReplayAllResult
	for _, tombstone := range tombstones {
		deletionResult, err := finalizer.convergeAndCount(ctx, tombstone, traceID)
		if err != nil {
			return DeletionReplayAllResult{}, err
		}
		switch deletionResult {
		case PendingArticleDeleted:
			result.Replayed++
		case PendingArticleAlreadyAbsent:
			result.AlreadyAbsent++
		}
	}
	return result, nil
}

// convergeAndCount converges one tombstone and reports its physical outcome.
func (finalizer *DeletionFinalizer) convergeAndCount(
	ctx context.Context,
	tombstone ArticleDeletionTombstone,
	traceID string,
) (PendingArticleDeletionResult, error) {
	operationID := tombstone.DeletionOperationID
	operation, err := finalizer.loadOperation(ctx, operationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			operation, err = finalizer.rebuildOperationFromTombstone(ctx, tombstone)
		}
	}
	if err != nil {
		return "", err
	}
	switch operation.Status {
	case DeletionOperationCompleted:
		return PendingArticleAlreadyAbsent, nil
	case DeletionOperationPendingJournal:
		appended, appendErr := finalizer.dependencies.Journal.Append(
			ctx,
			DeletionTombstoneRequest{
				ArticleID:           operation.ArticleID,
				ArticleVersion:      operation.ArticleVersion,
				DeletionOperationID: operation.ID,
				DeletedAt:           operation.StartedAt,
			},
		)
		if appendErr != nil {
			return "", appendErr
		}
		if err := finalizer.bindTombstone(ctx, operation, appended); err != nil {
			return "", err
		}
	case DeletionOperationJournaled, DeletionOperationFinalizing:
	default:
		return "", ErrDeletionJournalCorrupt
	}
	return finalizer.finalizeWithResult(ctx, operationID, tombstone, traceID)
}

// finalizeWithResult runs the normal finalize path and returns the physical outcome.
func (finalizer *DeletionFinalizer) finalizeWithResult(
	ctx context.Context,
	operationID string,
	tombstone ArticleDeletionTombstone,
	traceID string,
) (PendingArticleDeletionResult, error) {
	tombstone, err := finalizer.dependencies.Journal.FindByOperation(ctx, operationID)
	if err != nil {
		return "", fmt.Errorf("verify deletion tombstone: %w", err)
	}
	if tombstone.DeletionOperationID != operationID {
		return "", ErrDeletionJournalCorrupt
	}
	operation, claimed, err := finalizer.claim(ctx, operationID, tombstone)
	if err != nil {
		return "", err
	}
	if operation.Status == DeletionOperationCompleted {
		return PendingArticleAlreadyAbsent, nil
	}
	if !claimed {
		return "", ErrDeletionInProgress
	}
	return finalizer.completeWithResult(ctx, operation, tombstone, traceID)
}

// rebuildOperationFromTombstone inserts the minimal journaled operation that a
// restored backup is missing, per the ReplayAll contract.
func (finalizer *DeletionFinalizer) rebuildOperationFromTombstone(
	ctx context.Context,
	tombstone ArticleDeletionTombstone,
) (DeletionOperation, error) {
	tx, err := finalizer.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return DeletionOperation{}, fmt.Errorf("begin deletion replay rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	operation, err := finalizer.dependencies.Repository.LoadDeletionOperationForUpdate(
		ctx, tx, tombstone.DeletionOperationID,
	)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return DeletionOperation{}, fmt.Errorf("commit deletion replay rebuild read: %w", err)
		}
		return operation, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DeletionOperation{}, err
	}
	operation = DeletionOperation{
		ID:                tombstone.DeletionOperationID,
		ArticleID:         tombstone.ArticleID,
		ArticleVersion:    tombstone.ArticleVersion,
		DeleteKind:        DeleteOrdinary,
		Status:            DeletionOperationJournaled,
		TombstoneSequence: &tombstone.Sequence,
		JournalKeyID:      &tombstone.KeyID,
		JournalKeyVersion: &tombstone.KeyVersion,
		StartedAt:         tombstone.DeletedAt,
	}
	if err := finalizer.dependencies.Repository.InsertReplayedDeletionOperation(
		ctx, tx, operation,
	); err != nil {
		return DeletionOperation{}, fmt.Errorf("insert replayed deletion operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return DeletionOperation{}, fmt.Errorf("commit deletion replay rebuild: %w", err)
	}
	return operation, nil
}

func (finalizer *DeletionFinalizer) loadOperation(
	ctx context.Context,
	operationID string,
) (DeletionOperation, error) {
	tx, err := finalizer.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return DeletionOperation{}, fmt.Errorf("begin deletion convergence: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	operation, err := finalizer.dependencies.Repository.LoadDeletionOperationForUpdate(
		ctx,
		tx,
		operationID,
	)
	if err != nil {
		return DeletionOperation{}, fmt.Errorf("load deletion convergence: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return DeletionOperation{}, fmt.Errorf("commit deletion convergence read: %w", err)
	}
	return operation, nil
}

func (finalizer *DeletionFinalizer) bindTombstone(
	ctx context.Context,
	operation DeletionOperation,
	tombstone ArticleDeletionTombstone,
) error {
	if err := requireTombstoneBinding(operation, tombstone); err != nil {
		return err
	}
	tx, err := finalizer.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deletion convergence binding: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	locked, err := finalizer.dependencies.Repository.LoadDeletionOperationForUpdate(
		ctx,
		tx,
		operation.ID,
	)
	if err != nil {
		return fmt.Errorf("lock deletion convergence binding: %w", err)
	}
	if locked.Status != DeletionOperationPendingJournal {
		if err := requireTombstoneBinding(locked, tombstone); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if err := requireTombstoneBinding(locked, tombstone); err != nil {
		return err
	}
	if err := finalizer.dependencies.Repository.MarkDeletionOperationJournaled(
		ctx,
		tx,
		operation.ID,
		tombstone,
	); err != nil {
		return fmt.Errorf("bind deletion convergence tombstone: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit deletion convergence binding: %w", err)
	}
	return nil
}

func (finalizer *DeletionFinalizer) claim(
	ctx context.Context,
	operationID string,
	tombstone ArticleDeletionTombstone,
) (DeletionOperation, bool, error) {
	tx, err := finalizer.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return DeletionOperation{}, false, fmt.Errorf("begin finalizer claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	operation, err := finalizer.dependencies.Repository.LoadDeletionOperationForUpdate(
		ctx, tx, operationID,
	)
	if err != nil {
		return DeletionOperation{}, false, fmt.Errorf("load deletion operation: %w", err)
	}
	if err := requireTombstoneBinding(operation, tombstone); err != nil {
		return DeletionOperation{}, false, err
	}
	if operation.Status == DeletionOperationCompleted {
		return operation, false, nil
	}
	if operation.Status == DeletionOperationPendingJournal {
		if err := finalizer.dependencies.Repository.MarkDeletionOperationJournaled(
			ctx,
			tx,
			operationID,
			tombstone,
		); err != nil {
			return DeletionOperation{}, false, fmt.Errorf(
				"reconcile deletion tombstone binding: %w",
				err,
			)
		}
		operation.Status = DeletionOperationJournaled
		operation.TombstoneSequence = &tombstone.Sequence
		operation.JournalKeyID = &tombstone.KeyID
		operation.JournalKeyVersion = &tombstone.KeyVersion
	}
	now := finalizer.dependencies.Clock.Now().UTC()
	if now.IsZero() {
		return DeletionOperation{}, false, errors.New("deletion finalizer clock returned zero time")
	}
	operation, claimed, err := finalizer.dependencies.Repository.ClaimDeletionOperation(
		ctx, tx, operationID, finalizer.dependencies.WorkerID,
		now, now.Add(finalizer.dependencies.LeaseDuration),
	)
	if err != nil {
		return DeletionOperation{}, false, fmt.Errorf("claim deletion operation: %w", err)
	}
	leaseCompetition := !claimed &&
		operation.Status == DeletionOperationFinalizing &&
		operation.LockedUntil != nil &&
		operation.LockedUntil.After(now)
	if err := tx.Commit(ctx); err != nil {
		return DeletionOperation{}, false, fmt.Errorf("commit finalizer claim: %w", err)
	}
	if leaseCompetition {
		return operation, false, errors.Join(
			ErrDeletionInProgress,
			errDeletionLeaseCompetition,
		)
	}
	return operation, claimed, nil
}

func (finalizer *DeletionFinalizer) complete(
	ctx context.Context,
	operation DeletionOperation,
	tombstone ArticleDeletionTombstone,
	traceID string,
) error {
	_, err := finalizer.completeWithResult(ctx, operation, tombstone, traceID)
	return err
}

func (finalizer *DeletionFinalizer) completeWithResult(
	ctx context.Context,
	operation DeletionOperation,
	tombstone ArticleDeletionTombstone,
	traceID string,
) (PendingArticleDeletionResult, error) {
	ctx, err := deletionCompletionContext(ctx, traceID)
	if err != nil {
		return "", err
	}
	tx, err := finalizer.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin deletion completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ensureDispositionReadCommitted(ctx, tx); err != nil {
		return "", err
	}
	locked, err := finalizer.dependencies.Repository.LoadDeletionOperationForUpdate(
		ctx, tx, operation.ID,
	)
	if err != nil {
		return "", fmt.Errorf("lock deletion completion: %w", err)
	}
	if locked.Status == DeletionOperationCompleted {
		return PendingArticleAlreadyAbsent, nil
	}
	if err := finalizer.validateCompletionLease(locked, operation); err != nil {
		return "", err
	}
	if err := requireTombstoneBinding(locked, tombstone); err != nil {
		return "", err
	}
	proof, err := finalizer.lockDeletionDisposition(ctx, tx, locked, tombstone, traceID)
	if err != nil {
		return "", err
	}
	deletionResult, err := finalizer.applyDeletionCompletion(ctx, tx, locked, proof, traceID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit deletion completion; outcome unknown: %w", err)
	}
	return deletionResult, nil
}

func deletionCompletionContext(ctx context.Context, traceID string) (context.Context, error) {
	if existing, present := platform.TraceIDFromContext(ctx); present {
		if existing != traceID {
			return nil, errors.New("deletion trace context mismatch")
		}
		return ctx, nil
	}
	return platform.ContextWithTraceID(ctx, traceID)
}

func (finalizer *DeletionFinalizer) validateCompletionLease(locked, claimed DeletionOperation) error {
	now := finalizer.dependencies.Clock.Now().UTC()
	if now.IsZero() || locked.Status != DeletionOperationFinalizing || claimed.Status != DeletionOperationFinalizing ||
		locked.ClaimedBy == nil || *locked.ClaimedBy != finalizer.dependencies.WorkerID ||
		!sameDispositionString(locked.ClaimedBy, claimed.ClaimedBy) ||
		locked.LockedUntil == nil || claimed.LockedUntil == nil ||
		!locked.LockedUntil.UTC().Truncate(time.Microsecond).Equal(claimed.LockedUntil.UTC().Truncate(time.Microsecond)) ||
		!locked.LockedUntil.After(now) {
		return ErrDeletionInProgress
	}
	return nil
}

func (finalizer *DeletionFinalizer) lockDeletionDisposition(
	ctx context.Context, transaction pgx.Tx, operation DeletionOperation,
	tombstone ArticleDeletionTombstone, traceID string,
) (*DispositionCompletionProof, error) {
	var parties DispositionParties
	if operation.DeleteKind == DeleteOrphan {
		if operation.OrphanedFromUserID == nil || finalizer.dependencies.Identity == nil {
			return nil, errors.New("orphan finalization requires identity lifecycle")
		}
		var err error
		parties, err = finalizer.dependencies.Identity.LockTransferParties(ctx, transaction, nil, *operation.OrphanedFromUserID)
		if err != nil {
			return nil, fmt.Errorf("lock orphan finalization identity NOWAIT: %w", err)
		}
		if err := validateDispositionParties(parties, *operation.OrphanedFromUserID, nil); err != nil {
			return nil, err
		}
	}
	before, err := finalizer.dependencies.Repository.LockArticleForDisposition(ctx, transaction, operation.ArticleID)
	present := !errors.Is(err, pgx.ErrNoRows)
	if err != nil && present {
		return nil, fmt.Errorf("lock deletion article NOWAIT: %w", err)
	}
	if present {
		if err := validatePendingDisposition(before, operation); err != nil {
			return nil, err
		}
	}
	if operation.DeleteKind != DeleteOrphan {
		if !present || before.AuthorID == nil {
			return nil, nil
		}
		var deleted bool
		parties, deleted, err = finalizer.lockAuthoredDeletionCandidate(ctx, transaction, *before.AuthorID)
		if err != nil || !deleted {
			return nil, err
		}
	}
	proof := newDispositionProof(transaction, before, parties, "", traceID)
	proof.reference.Action, proof.reference.BeforeVersion = DispositionDeleteComplete, operation.ArticleVersion
	proof.reference.ExpectedArticleID, proof.reference.OperationID = operation.ArticleID, copyDispositionString(&operation.ID)
	if !present {
		proof.reference.BeforeArticle = DispositionArticleState{}
	}
	operation.OrphanedFromUserID = copyDispositionString(operation.OrphanedFromUserID)
	proof.operation, proof.tombstone = &operation, tombstone
	return proof, nil
}

func (finalizer *DeletionFinalizer) lockAuthoredDeletionCandidate(
	ctx context.Context, transaction pgx.Tx, authorID string,
) (DispositionParties, bool, error) {
	if finalizer.dependencies.Identity == nil {
		return DispositionParties{}, false, errors.New("authored finalization requires identity lifecycle")
	}
	candidate, err := finalizer.dependencies.Identity.LockDeletionCandidate(ctx, transaction, authorID)
	if err != nil {
		return DispositionParties{}, false, fmt.Errorf("lock deletion candidate NOWAIT: %w", err)
	}
	if candidate.UserID != authorID {
		return DispositionParties{}, false, errors.New("deletion candidate identity mismatch")
	}
	return DispositionParties{SourceID: candidate.UserID}, candidate.Deleted, nil
}

func validatePendingDisposition(before Article, operation DeletionOperation) error {
	if before.ID != operation.ArticleID || before.Version != operation.ArticleVersion+1 ||
		before.DeletionState != DeletionPending || !sameDispositionString(before.DeletionOperationID, &operation.ID) ||
		ValidateArticleState(before) != nil {
		return ErrDeletionArticleMismatch
	}
	if operation.DeleteKind == DeleteOrphan {
		if before.AuthorID != nil || !sameDispositionString(before.OrphanedFromUserID, operation.OrphanedFromUserID) ||
			before.PublicationState != PublicationHeld {
			return ErrDeletionArticleMismatch
		}
	} else if (operation.DeleteKind != DeleteOrdinary && operation.DeleteKind != DeleteViolating) ||
		operation.OrphanedFromUserID != nil || before.OrphanedFromUserID != nil {
		return ErrDeletionArticleMismatch
	}
	return nil
}

func (finalizer *DeletionFinalizer) applyDeletionCompletion(
	ctx context.Context, tx pgx.Tx, locked DeletionOperation, proof *DispositionCompletionProof, traceID string,
) (PendingArticleDeletionResult, error) {
	deletionResult, err := finalizer.dependencies.Repository.DeletePendingArticle(
		ctx, tx, locked.ArticleID, locked.ArticleVersion+1, locked.ID,
	)
	if err != nil {
		return "", fmt.Errorf("delete pending article: %w", err)
	}
	if deletionResult == PendingArticlePresentMismatch {
		return "", ErrDeletionArticleMismatch
	}
	if deletionResult != PendingArticleDeleted &&
		deletionResult != PendingArticleAlreadyAbsent {
		return "", fmt.Errorf("delete pending article returned invalid result %q", deletionResult)
	}
	auditID, err := finalizer.dependencies.AuditIDs.NewID()
	if err != nil || !validUUID(auditID) {
		if err == nil {
			err = errors.New("generated deletion audit ID is invalid")
		}
		return "", fmt.Errorf("generate deletion audit ID: %w", err)
	}
	completedAt := finalizer.dependencies.Clock.Now().UTC()
	if completedAt.IsZero() {
		return "", errors.New("deletion finalizer clock returned zero completion time")
	}
	event := deletionAuditEvent(auditID, traceID, completedAt, locked, deletionResult)
	if err := finalizer.dependencies.Auditor.Append(ctx, tx, event); err != nil {
		return "", fmt.Errorf("append deletion audit: %w", err)
	}
	if err := finalizer.dependencies.Repository.CompleteDeletionOperation(
		ctx, tx, locked.ID, finalizer.dependencies.WorkerID, auditID, completedAt,
	); err != nil {
		return "", fmt.Errorf("complete deletion operation: %w", err)
	}
	if proof != nil {
		proof.reference.AuditEventID = auditID
		if _, err := proof.VerifyCompletion(ctx, tx); err != nil {
			return "", err
		}
		if _, err := finalizer.dependencies.Identity.ClearRetainedEmailIfEligible(
			ctx, tx, proof.reference.SourceID, proof); err != nil {
			return "", fmt.Errorf("clear finalization identity: %w", err)
		}
	}
	return deletionResult, nil
}

func requireTombstoneBinding(
	operation DeletionOperation,
	tombstone ArticleDeletionTombstone,
) error {
	if operation.ID != tombstone.DeletionOperationID ||
		operation.ArticleID != tombstone.ArticleID ||
		operation.ArticleVersion != tombstone.ArticleVersion ||
		!operation.StartedAt.UTC().Equal(tombstone.DeletedAt.UTC()) {
		return ErrDeletionJournalCorrupt
	}
	if operation.Status != DeletionOperationPendingJournal {
		if operation.TombstoneSequence == nil ||
			*operation.TombstoneSequence != tombstone.Sequence ||
			operation.JournalKeyID == nil || *operation.JournalKeyID != tombstone.KeyID ||
			operation.JournalKeyVersion == nil ||
			*operation.JournalKeyVersion != tombstone.KeyVersion {
			return ErrDeletionJournalCorrupt
		}
	}
	return nil
}

func deletionAuditEvent(
	id string,
	traceID string,
	completedAt time.Time,
	operation DeletionOperation,
	deletionResult PendingArticleDeletionResult,
) audit.Event {
	metadata := map[string]any{
		deletionAuditVersionKey: operation.ArticleVersion,
		deletionAuditKindKey:    operation.DeleteKind,
	}
	if operation.DeleteKind == DeleteOrdinary &&
		operation.ActorRole != nil && *operation.ActorRole == principalRoleAdmin {
		metadata[deletionAuditAuthorityKey] = deletionAuditAdministrator
	}
	event := audit.Event{
		ID: id, OccurredAt: completedAt, ActorKind: audit.ActorUser,
		ActorUserID: operation.ActorUserID, ActorRole: operation.ActorRole,
		Action:     deletionAuditAction(operation.DeleteKind),
		TargetType: deletionAuditTargetType, TargetID: operation.ArticleID,
		Outcome: deletionAuditOutcome, TraceID: traceID,
		Metadata: metadata,
	}
	if deletionResult == PendingArticleAlreadyAbsent && operation.DeleteKind == DeleteOrdinary &&
		operation.ActorUserID == nil && operation.ActorRole == nil && operation.OrphanedFromUserID == nil {
		actorSystem := deletionReplayAuditActorSystem
		event.ActorKind, event.ActorSystem, event.Action = audit.ActorSystem, &actorSystem, deletionReplayAuditAction
	}
	return event
}

func deletionAuditAction(kind DeleteKind) string {
	switch kind {
	case DeleteViolating:
		return "article.violation.delete"
	case DeleteOrphan:
		return "article.orphan.delete"
	default:
		return "article.delete"
	}
}

const loadDeletionOperationForUpdateSQL = `
	SELECT id, article_id, article_version, delete_kind, actor_user_id, actor_role,
		orphaned_from_user_id, status, tombstone_sequence, journal_key_id,
		journal_key_version, claimed_by, locked_until, audit_event_id,
		started_at, completed_at
	FROM deletion_operations
	WHERE id = $1
	FOR UPDATE
`

// LoadDeletionOperationForUpdate locks one operation.
func (Repository) LoadDeletionOperationForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
) (DeletionOperation, error) {
	if err := requireTransaction(tx); err != nil {
		return DeletionOperation{}, err
	}
	return scanDispositionOperation(tx.QueryRow(ctx, loadDeletionOperationForUpdateSQL, operationID))
}

func scanDispositionOperation(row pgx.Row) (DeletionOperation, error) {
	var operation DeletionOperation
	err := row.Scan(
		&operation.ID, &operation.ArticleID, &operation.ArticleVersion,
		&operation.DeleteKind, &operation.ActorUserID, &operation.ActorRole,
		&operation.OrphanedFromUserID, &operation.Status,
		&operation.TombstoneSequence, &operation.JournalKeyID,
		&operation.JournalKeyVersion, &operation.ClaimedBy, &operation.LockedUntil,
		&operation.AuditEventID, &operation.StartedAt, &operation.CompletedAt,
	)
	return operation, err
}

// ClaimDeletionOperation acquires or reclaims one finalizer lease.
func (Repository) ClaimDeletionOperation(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
	workerID string,
	now time.Time,
	lockedUntil time.Time,
) (DeletionOperation, bool, error) {
	operation, err := (Repository{}).LoadDeletionOperationForUpdate(ctx, tx, operationID)
	if err != nil {
		return DeletionOperation{}, false, err
	}
	if operation.Status == DeletionOperationCompleted {
		return operation, false, nil
	}
	if operation.Status == DeletionOperationFinalizing &&
		operation.LockedUntil != nil && operation.LockedUntil.After(now) {
		return operation, false, nil
	}
	claimed, err := scanDispositionOperation(tx.QueryRow(ctx, `
		UPDATE deletion_operations
		SET status = 'finalizing', claimed_by = $2, locked_until = $3
		WHERE id = $1 AND status IN ('journaled', 'finalizing')
		RETURNING id, article_id, article_version, delete_kind, actor_user_id, actor_role,
			orphaned_from_user_id, status, tombstone_sequence, journal_key_id,
			journal_key_version, claimed_by, locked_until, audit_event_id, started_at, completed_at
	`, operationID, workerID, lockedUntil))
	if errors.Is(err, pgx.ErrNoRows) {
		return operation, false, nil
	}
	if err != nil {
		return DeletionOperation{}, false, err
	}
	if claimed.LockedUntil == nil || claimed.ClaimedBy == nil || *claimed.ClaimedBy != workerID ||
		claimed.Status != DeletionOperationFinalizing {
		return DeletionOperation{}, false, errors.New("claimed deletion lease readback is invalid")
	}
	canonicalUntil := claimed.LockedUntil.UTC().Truncate(time.Microsecond)
	claimed.LockedUntil = &canonicalUntil
	return claimed, true, nil
}

// DeletePendingArticle physically deletes only the operation-bound pending Article.
func (Repository) DeletePendingArticle(
	ctx context.Context,
	tx pgx.Tx,
	articleID string,
	pendingVersion int64,
	operationID string,
) (PendingArticleDeletionResult, error) {
	if err := requireTransaction(tx); err != nil {
		return "", err
	}
	tag, err := tx.Exec(ctx, `
		DELETE FROM articles
		WHERE id = $1 AND version = $2
			AND deletion_state = 'deletion_pending'
			AND deletion_operation_id = $3
	`, articleID, pendingVersion, operationID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 1 {
		return PendingArticleDeleted, nil
	}
	var present bool
	if err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM articles WHERE id = $1)`,
		articleID,
	).Scan(&present); err != nil {
		return "", err
	}
	if present {
		return PendingArticlePresentMismatch, nil
	}
	return PendingArticleAlreadyAbsent, nil
}

// CompleteDeletionOperation records the audit binding and immutable completion.
func (Repository) CompleteDeletionOperation(
	ctx context.Context,
	tx pgx.Tx,
	operationID string,
	workerID string,
	auditEventID string,
	completedAt time.Time,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE deletion_operations
		SET status = 'completed', claimed_by = NULL, locked_until = NULL,
			audit_event_id = $3, completed_at = $4
		WHERE id = $1 AND status = 'finalizing' AND claimed_by = $2
	`, operationID, workerID, auditEventID, completedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("deletion operation completion lost its lease")
	}
	return nil
}

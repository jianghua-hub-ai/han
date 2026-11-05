package article

import (
	"context"
	"errors"
	"fmt"
	"time"

	"han/internal/audit"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists article-owned models in a caller-owned transaction.
type Repository struct{}

// PostgresDeletionOperationDiscoverer reads the durable deletion queue.
type PostgresDeletionOperationDiscoverer struct {
	pool *pgxpool.Pool
}

// NewPostgresDeletionOperationDiscoverer creates the production discovery adapter.
func NewPostgresDeletionOperationDiscoverer(
	pool *pgxpool.Pool,
) (*PostgresDeletionOperationDiscoverer, error) {
	if pool == nil {
		return nil, errors.New("article deletion discovery pool is required")
	}
	return &PostgresDeletionOperationDiscoverer{pool: pool}, nil
}

// DiscoverDeletionOperations returns one stable bounded batch of due operations.
func (discoverer *PostgresDeletionOperationDiscoverer) DiscoverDeletionOperations(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]string, error) {
	if discoverer == nil || discoverer.pool == nil || now.IsZero() || limit <= 0 {
		return nil, errors.New("article deletion discovery input is invalid")
	}
	rows, err := discoverer.pool.Query(ctx, discoverDeletionOperationsSQL, now.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("discover article deletion operations: %w", err)
	}
	defer rows.Close()
	operationIDs := make([]string, 0, limit)
	for rows.Next() {
		var operationID string
		if err := rows.Scan(&operationID); err != nil {
			return nil, fmt.Errorf("scan article deletion operation: %w", err)
		}
		operationIDs = append(operationIDs, operationID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read article deletion operations: %w", err)
	}
	return operationIDs, nil
}

// InsertArticle inserts one article with authoritative orthogonal state.
func (Repository) InsertArticle(ctx context.Context, tx pgx.Tx, article Article) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertArticleRecordSQL,
		article.ID, article.AuthorID, article.OrphanedFromUserID, article.Title,
		article.Body, article.PublicationState, article.ViolationState,
		article.ViolationCategory, article.DeletionState,
		article.DeletionOperationID, article.Version, article.CreatedAt,
		article.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert article: %w", err)
	}
	return nil
}

// InsertTransferNotice inserts one notice and its two independent recipients.
func (Repository) InsertTransferNotice(
	ctx context.Context,
	tx pgx.Tx,
	notice TransferNotice,
	recipients [2]TransferNoticeRecipient,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if err := validateTransferNoticeRecipients(notice.ID, recipients); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertTransferNoticeRecordSQL,
		notice.ID, notice.ArticleRef, notice.NewAuthorID,
		notice.FormerDeletedUserID, notice.CreatedAt, notice.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert article transfer notice: %w", err)
	}
	for _, recipient := range recipients {
		if err := insertTransferNoticeRecipient(ctx, tx, recipient); err != nil {
			return err
		}
	}
	return nil
}

func validateTransferNoticeRecipients(
	noticeID string,
	recipients [2]TransferNoticeRecipient,
) error {
	var hasNewAuthor bool
	var hasFormerDeletedAuthor bool
	for _, recipient := range recipients {
		if recipient.NoticeID != noticeID {
			return errors.New("article transfer notice recipient is bound to another notice")
		}
		switch recipient.Kind {
		case RecipientNewAuthor:
			if hasNewAuthor {
				return errors.New("article transfer notice recipient kinds must be unique")
			}
			hasNewAuthor = true
		case RecipientFormerDeletedAuthor:
			if hasFormerDeletedAuthor {
				return errors.New("article transfer notice recipient kinds must be unique")
			}
			hasFormerDeletedAuthor = true
		default:
			return errors.New("article transfer notice recipient kind is invalid")
		}
	}
	if !hasNewAuthor || !hasFormerDeletedAuthor {
		return errors.New("article transfer notice requires both recipient kinds")
	}
	return nil
}

// InsertDeletionOperation inserts one permanent-deletion coordination record.
func (Repository) InsertDeletionOperation(
	ctx context.Context,
	tx pgx.Tx,
	operation DeletionOperation,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertDeletionOperationRecordSQL,
		operation.ID, operation.ArticleID, operation.ArticleVersion,
		operation.DeleteKind, operation.ActorUserID, operation.ActorRole,
		operation.OrphanedFromUserID, operation.Status,
		operation.TombstoneSequence, operation.JournalKeyID,
		operation.JournalKeyVersion, operation.ClaimedBy, operation.LockedUntil,
		operation.AuditEventID, operation.StartedAt, operation.CompletedAt,
	); err != nil {
		return fmt.Errorf("insert article deletion operation: %w", err)
	}
	return nil
}

// InsertDeletionReplayRun inserts the first immutable backup binding for a run.
func (Repository) InsertDeletionReplayRun(
	ctx context.Context,
	tx pgx.Tx,
	run DeletionReplayRun,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertDeletionReplayRunRecordSQL,
		run.RunID, run.Namespace, run.BackupRef, run.InputDigest,
		run.CurrentEvidenceRef, run.Status, run.LastErrorBucket,
		run.StartedAt, run.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert article deletion replay run: %w", err)
	}
	return nil
}

// InsertDeletionReplayEvidence inserts one immutable replay result.
func (Repository) InsertDeletionReplayEvidence(
	ctx context.Context,
	tx pgx.Tx,
	evidence DeletionReplayEvidence,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertDeletionReplayEvidenceRecordSQL,
		evidence.EvidenceVersion, evidence.RunID, evidence.BackupRef,
		evidence.Namespace, evidence.JournalHeadSequence,
		evidence.JournalHeadHash, evidence.InventoryScope,
		evidence.BackupCutoff, evidence.InventoryDigest,
		evidence.DeletionOperationsDigest, evidence.AuditEventsDigest,
		evidence.TombstonesTotal, evidence.TombstonesReplayed,
		evidence.AlreadyAbsent, evidence.DeletionOperationsReconciledCount,
		evidence.AuditEventsVerifiedCount, evidence.EvidenceDigest,
		evidence.EvidenceRef, evidence.CompletedAt,
	); err != nil {
		return fmt.Errorf("insert article deletion replay evidence: %w", err)
	}
	return nil
}

// SetDeletionReplayRunCurrent atomically marks immutable evidence current and ready.
func (Repository) SetDeletionReplayRunCurrent(
	ctx context.Context,
	tx pgx.Tx,
	runID string,
	namespace string,
	inputDigest string,
	evidenceRef string,
	updatedAt time.Time,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	tag, err := tx.Exec(
		ctx, setDeletionReplayRunCurrentSQL,
		inputDigest, evidenceRef, updatedAt, runID, namespace,
	)
	if err != nil {
		return fmt.Errorf("set current article deletion replay evidence: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("article deletion replay run was not found")
	}
	return nil
}

// InsertReplayedDeletionOperation inserts the minimal journaled operation that a
// restored backup is missing during ReplayAll reconciliation.
func (Repository) InsertReplayedDeletionOperation(
	ctx context.Context,
	tx pgx.Tx,
	operation DeletionOperation,
) error {
	return (Repository{}).InsertDeletionOperation(ctx, tx, operation)
}

// PostgresReplayFactReader reads the durable operation and audit facts that the
// production replay source consumes. It is read-only.
type PostgresReplayFactReader struct {
	pool *pgxpool.Pool
}

// NewPostgresReplayFactReader creates the read-only production fact reader.
func NewPostgresReplayFactReader(
	pool *pgxpool.Pool,
) (*PostgresReplayFactReader, error) {
	if pool == nil {
		return nil, errors.New("article replay fact reader pool is required")
	}
	return &PostgresReplayFactReader{pool: pool}, nil
}

// ReadDeletionOperations returns every durable operation completion fact.
func (reader *PostgresReplayFactReader) ReadDeletionOperations(
	ctx context.Context,
) ([]DeletionOperationReplayFact, error) {
	if reader == nil || reader.pool == nil {
		return nil, ErrDeletionReplayInvalidRequest
	}
	rows, err := reader.pool.Query(ctx, readDeletionOperationsSQL)
	if err != nil {
		return nil, fmt.Errorf("read deletion operations for replay: %w", err)
	}
	defer rows.Close()
	var facts []DeletionOperationReplayFact
	for rows.Next() {
		var fact DeletionOperationReplayFact
		if err := rows.Scan(
			&fact.OperationID, &fact.Status, &fact.ArticleVersion, &fact.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan deletion operation replay fact: %w", err)
		}
		if fact.CompletedAt != nil {
			completedAt := fact.CompletedAt.UTC()
			fact.CompletedAt = &completedAt
		}
		facts = append(facts, fact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deletion operation replay facts: %w", err)
	}
	return facts, nil
}

// ReadDeletionAuditEvents returns every durable deletion-audit fact.
func (reader *PostgresReplayFactReader) ReadDeletionAuditEvents(
	ctx context.Context,
) ([]DeletionReplayAuditFact, error) {
	if reader == nil || reader.pool == nil {
		return nil, ErrDeletionReplayInvalidRequest
	}
	rows, err := reader.pool.Query(ctx, readDeletionAuditEventsSQL)
	if err != nil {
		return nil, fmt.Errorf("read deletion audit events for replay: %w", err)
	}
	defer rows.Close()
	var facts []DeletionReplayAuditFact
	for rows.Next() {
		var fact DeletionReplayAuditFact
		if err := rows.Scan(
			&fact.EventID, &fact.TargetID, &fact.OccurredAt, &fact.Action,
		); err != nil {
			return nil, fmt.Errorf("scan deletion audit replay fact: %w", err)
		}
		fact.OccurredAt = fact.OccurredAt.UTC()
		facts = append(facts, fact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deletion audit replay facts: %w", err)
	}
	return facts, nil
}

const readDeletionOperationsSQL = `
	SELECT id, status, article_version, completed_at
	FROM deletion_operations
	ORDER BY id ASC
`

const readDeletionAuditEventsSQL = `
	SELECT id, target_id, occurred_at, action
	FROM audit_events
	WHERE target_type = 'article'
		AND action IN (
			'article.delete',
			'article.violation.delete',
			'article.orphan.delete',
			'article.deletion.replay'
		)
	ORDER BY target_id ASC, id ASC
`

var _ ReplayOperationReader = (*PostgresReplayFactReader)(nil)
var _ ReplayAuditReader = (*PostgresReplayFactReader)(nil)

func insertTransferNoticeRecipient(
	ctx context.Context,
	tx pgx.Tx,
	recipient TransferNoticeRecipient,
) error {
	if _, err := tx.Exec(ctx, insertTransferNoticeRecipientRecordSQL,
		recipient.NoticeID, recipient.Kind, recipient.EmailSnapshot,
		recipient.Status, recipient.ClaimedBy, recipient.LockedUntil,
		recipient.NextAttemptAt, recipient.AttemptCount, recipient.MaxAttempts,
		recipient.LastAttemptAt, recipient.AcceptedAt, recipient.FailureBucket,
		recipient.CreatedAt, recipient.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert article transfer notice recipient: %w", err)
	}
	return nil
}

func requireTransaction(tx pgx.Tx) error {
	if tx == nil {
		return errors.New("article transaction is required")
	}
	return nil
}

const insertArticleRecordSQL = `
	INSERT INTO articles (
		id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state,
		deletion_operation_id, version, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const insertTransferNoticeRecordSQL = `
	INSERT INTO transfer_notices (
		id, article_ref, new_author_id, former_deleted_user_id, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6)
`

const insertTransferNoticeRecipientRecordSQL = `
	INSERT INTO transfer_notice_recipients (
		notice_id, recipient_kind, email_snapshot, status, claimed_by,
		locked_until, next_attempt_at, attempt_count, max_attempts,
		last_attempt_at, accepted_at, failure_bucket, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

const insertDeletionOperationRecordSQL = `
	INSERT INTO deletion_operations (
		id, article_id, article_version, delete_kind, actor_user_id, actor_role,
		orphaned_from_user_id, status, tombstone_sequence, journal_key_id,
		journal_key_version, claimed_by, locked_until, audit_event_id,
		started_at, completed_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15, $16
	)
`

const insertDeletionReplayRunRecordSQL = `
	INSERT INTO deletion_replay_runs (
		run_id, namespace, backup_ref, input_digest, current_evidence_ref,
		status, last_error_bucket, started_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

const insertDeletionReplayEvidenceRecordSQL = `
	INSERT INTO deletion_replay_evidence (
		evidence_version, run_id, backup_ref, namespace, journal_head_sequence,
		journal_head_hash, inventory_scope, backup_cutoff, inventory_digest,
		deletion_operations_digest, audit_events_digest, tombstones_total,
		tombstones_replayed, already_absent,
		deletion_operations_reconciled_count, audit_events_verified_count,
		evidence_digest, evidence_ref, completed_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19
	)
`

const setDeletionReplayRunCurrentSQL = `
	UPDATE deletion_replay_runs
	SET input_digest = $1, current_evidence_ref = $2, status = 'ready',
		last_error_bucket = NULL, updated_at = $3
	WHERE run_id = $4 AND namespace = $5
`

const discoverDeletionOperationsSQL = `
	SELECT id
	FROM deletion_operations
	WHERE status IN ('pending_journal', 'journaled')
		OR (status = 'finalizing' AND locked_until <= $1)
	ORDER BY started_at ASC, id ASC
	LIMIT $2
`

// LocateDispositionArticle reads only a location, never a completion preimage.
func (Repository) LocateDispositionArticle(ctx context.Context, transaction pgx.Tx, articleID string) (DispositionArticleState, error) {
	if err := requireTransaction(transaction); err != nil {
		return DispositionArticleState{}, err
	}
	var state DispositionArticleState
	err := transaction.QueryRow(ctx, locateDispositionArticleSQL, articleID).Scan(&state.ID, &state.SourceID, &state.Version)
	state.Exists = err == nil
	return state, err
}

// LockArticleForDisposition takes the explicit operation-compatible NOWAIT lock.
func (Repository) LockArticleForDisposition(ctx context.Context, transaction pgx.Tx, articleID string) (Article, error) {
	if err := requireTransaction(transaction); err != nil {
		return Article{}, err
	}
	return scanQueryArticle(transaction.QueryRow(ctx, lockDispositionArticleSQL, articleID))
}

// UpdateOrphanDisposition changes only ownership/publication and version/time.
func (Repository) UpdateOrphanDisposition(ctx context.Context, transaction pgx.Tx, before, after Article) error {
	if err := requireTransaction(transaction); err != nil {
		return err
	}
	command, err := transaction.Exec(ctx, updateOrphanDispositionSQL,
		before.ID, before.Version, before.OrphanedFromUserID, after.AuthorID, after.UpdatedAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrPreconditionFailed
	}
	return nil
}

// ReadDispositionCompletion reads authoritative facts exclusively on caller tx.
func (Repository) ReadDispositionCompletion(
	ctx context.Context, transaction pgx.Tx, reference DispositionCompletionReference,
) (DispositionCompletionFacts, error) {
	if err := requireTransaction(transaction); err != nil {
		return DispositionCompletionFacts{}, err
	}
	state, err := readDispositionState(ctx, transaction, reference.ExpectedArticleID)
	if err != nil {
		return DispositionCompletionFacts{}, fmt.Errorf("read disposition article: %w", err)
	}
	facts := DispositionCompletionFacts{AfterArticle: state}
	facts.Audit, err = (audit.Store{}).Get(ctx, transaction, reference.AuditEventID)
	if err != nil {
		return DispositionCompletionFacts{}, fmt.Errorf("read disposition audit: %w", err)
	}
	if reference.Action == DispositionTransfer {
		if reference.NoticeID == nil {
			return DispositionCompletionFacts{}, errors.New("disposition notice binding is required")
		}
		notice, recipients, err := readDispositionNotice(ctx, transaction, *reference.NoticeID)
		if err != nil {
			return DispositionCompletionFacts{}, err
		}
		facts.Notice, facts.Recipients = &notice, recipients
	}
	if reference.Action == DispositionDeleteComplete {
		if reference.OperationID == nil {
			return DispositionCompletionFacts{}, errors.New("disposition operation binding is required")
		}
		operation, err := scanDispositionOperation(transaction.QueryRow(ctx, readDispositionOperationSQL, *reference.OperationID))
		if err != nil {
			return DispositionCompletionFacts{}, fmt.Errorf("read completed disposition operation: %w", err)
		}
		facts.CompletedOperation = &operation
	}
	err = transaction.QueryRow(ctx, countDispositionAssociationsSQL, reference.SourceID).Scan(
		&facts.RemainingActiveCount, &facts.RemainingPendingCount)
	if err != nil {
		return DispositionCompletionFacts{}, fmt.Errorf("count disposition associations: %w", err)
	}
	return facts, nil
}

func readDispositionState(ctx context.Context, transaction pgx.Tx, articleID string) (DispositionArticleState, error) {
	var state DispositionArticleState
	err := transaction.QueryRow(ctx, readDispositionStateSQL, articleID).Scan(&state.ID, &state.Version,
		&state.AuthorID, &state.SourceID, &state.Publication, &state.Violation, &state.Category,
		&state.Deletion, &state.OperationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return DispositionArticleState{}, nil
	}
	state.Exists = err == nil
	return state, err
}

func readDispositionNotice(
	ctx context.Context, transaction pgx.Tx, noticeID string,
) (TransferNotice, []TransferNoticeRecipient, error) {
	var notice TransferNotice
	err := transaction.QueryRow(ctx, readDispositionNoticeSQL, noticeID).Scan(
		&notice.ID, &notice.ArticleRef, &notice.NewAuthorID, &notice.FormerDeletedUserID)
	if err != nil {
		return TransferNotice{}, nil, fmt.Errorf("read disposition notice: %w", err)
	}
	rows, err := transaction.Query(ctx, readDispositionRecipientsSQL, noticeID)
	if err != nil {
		return TransferNotice{}, nil, fmt.Errorf("read disposition recipients: %w", err)
	}
	defer rows.Close()
	var recipients []TransferNoticeRecipient
	for rows.Next() {
		var recipient TransferNoticeRecipient
		if err := rows.Scan(&recipient.NoticeID, &recipient.Kind, &recipient.EmailSnapshot, &recipient.Status); err != nil {
			return TransferNotice{}, nil, fmt.Errorf("scan disposition recipient: %w", err)
		}
		recipients = append(recipients, recipient)
	}
	if err := rows.Err(); err != nil {
		return TransferNotice{}, nil, fmt.Errorf("iterate disposition recipients: %w", err)
	}
	return notice, recipients, nil
}

const locateDispositionArticleSQL = `
	SELECT id, orphaned_from_user_id, version FROM articles WHERE id = $1::uuid
`

func ensureDispositionReadCommitted(ctx context.Context, transaction pgx.Tx) error {
	if err := requireTransaction(transaction); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, dispositionReadCommittedSQL); err != nil {
		return fmt.Errorf("set disposition transaction read committed: %w", err)
	}
	return nil
}

const dispositionReadCommittedSQL = "SET TRANSACTION ISOLATION LEVEL READ COMMITTED"

const lockDispositionArticleSQL = `
	SELECT id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state, deletion_operation_id,
		version, created_at, updated_at
	FROM articles WHERE id = $1::uuid FOR UPDATE NOWAIT
`

const updateOrphanDispositionSQL = `
	UPDATE articles
	SET author_id = $4::uuid, orphaned_from_user_id = NULL,
		publication_state = 'published', version = version + 1, updated_at = $5
	WHERE id = $1::uuid AND version = $2 AND orphaned_from_user_id = $3::uuid
		AND author_id IS NULL AND publication_state = 'held' AND deletion_state = 'active'
`

const readDispositionStateSQL = `
	SELECT id, version, author_id, orphaned_from_user_id, publication_state,
		violation_state, violation_category, deletion_state, deletion_operation_id
	FROM articles WHERE id = $1::uuid
`

const readDispositionNoticeSQL = `
	SELECT id, article_ref, new_author_id, former_deleted_user_id FROM transfer_notices WHERE id = $1::uuid
`

const readDispositionRecipientsSQL = `
	SELECT notice_id, recipient_kind, email_snapshot, status FROM transfer_notice_recipients
	WHERE notice_id = $1::uuid ORDER BY recipient_kind
`

const readDispositionOperationSQL = `
	SELECT id, article_id, article_version, delete_kind, actor_user_id, actor_role,
		orphaned_from_user_id, status, tombstone_sequence, journal_key_id,
		journal_key_version, claimed_by, locked_until, audit_event_id, started_at, completed_at
	FROM deletion_operations WHERE id = $1::uuid
`

const countDispositionAssociationsSQL = `
	SELECT count(*) FILTER (WHERE deletion_state = 'active'),
		count(*) FILTER (WHERE deletion_state = 'deletion_pending')
	FROM articles WHERE author_id = $1::uuid OR orphaned_from_user_id = $1::uuid
`

// LockDueNoticeRecipient locks at most one eligible recipient in queue order.
func (Repository) LockDueNoticeRecipient(ctx context.Context, tx pgx.Tx) (TransferNotice, TransferNoticeRecipient, bool, error) {
	return readNoticeRow(ctx, tx, lockDueNoticeRecipientSQL)
}

// LockExhaustedNoticeRecipient locks due exhausted work independently of Gate.
func (Repository) LockExhaustedNoticeRecipient(ctx context.Context, tx pgx.Tx) (TransferNotice, TransferNoticeRecipient, bool, error) {
	return readNoticeRow(ctx, tx, lockExhaustedNoticeRecipientSQL)
}

// LockNoticeRecipient locks one exact recipient for preflight or result fencing.
func (Repository) LockNoticeRecipient(ctx context.Context, tx pgx.Tx, id string, kind RecipientKind) (TransferNotice, TransferNoticeRecipient, bool, error) {
	return readNoticeRow(ctx, tx, readNoticeRecipientSQL+" FOR UPDATE OF recipient", id, kind)
}

// ReadNoticeRecipient reads committed result evidence without acquiring send rights.
func (Repository) ReadNoticeRecipient(ctx context.Context, tx pgx.Tx, id string, kind RecipientKind) (TransferNotice, TransferNoticeRecipient, bool, error) {
	return readNoticeRow(ctx, tx, readNoticeRecipientSQL, id, kind)
}

func readNoticeRow(ctx context.Context, tx pgx.Tx, query string, args ...any) (TransferNotice, TransferNoticeRecipient, bool, error) {
	if noticeNil(tx) {
		return TransferNotice{}, TransferNoticeRecipient{}, false, noticeFailure("transaction_required", nil, true)
	}
	var notice TransferNotice
	var r TransferNoticeRecipient
	err := tx.QueryRow(ctx, query, args...).Scan(&notice.ArticleRef, &r.NoticeID, &r.Kind, &r.EmailSnapshot, &r.Status,
		&r.ClaimedBy, &r.LockedUntil, &r.NextAttemptAt, &r.AttemptCount, &r.MaxAttempts, &r.LastAttemptAt,
		&r.AcceptedAt, &r.FailureBucket, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notice, r, false, nil
	}
	if err != nil {
		return notice, r, false, noticeDBFailure("recipient_read", err)
	}
	notice.ID = r.NoticeID
	return notice, normalizeNoticeRecipient(r), true, nil
}

func normalizeNoticeRecipient(r TransferNoticeRecipient) TransferNoticeRecipient {
	r.CreatedAt = r.CreatedAt.UTC().Truncate(time.Microsecond)
	r.UpdatedAt = r.UpdatedAt.UTC().Truncate(time.Microsecond)
	for _, value := range []**time.Time{&r.LockedUntil, &r.NextAttemptAt, &r.LastAttemptAt, &r.AcceptedAt} {
		if *value != nil {
			normalized := (**value).UTC().Truncate(time.Microsecond)
			*value = &normalized
		}
	}
	return r
}

// ClaimNoticeRecipient updates exactly the locked preimage and returns DB ownership.
func (Repository) ClaimNoticeRecipient(ctx context.Context, tx pgx.Tx, before TransferNoticeRecipient, worker string, lease time.Duration, now time.Time) (TransferNoticeRecipient, bool, error) {
	until, err := noticeAdd(now, lease)
	if err != nil {
		return TransferNoticeRecipient{}, false, err
	}
	if !noticeDue(before, now) || before.AttemptCount >= before.MaxAttempts {
		return TransferNoticeRecipient{}, false, nil
	}
	return updateNoticeRow(ctx, tx, claimNoticeRecipientSQL, before.NoticeID, before.Kind, before.Status,
		before.AttemptCount, before.ClaimedBy, before.LockedUntil, before.NextAttemptAt, worker, now, until)
}

// SaveNoticeResult conditionally writes a classified result under full live fencing.
func (Repository) SaveNoticeResult(ctx context.Context, tx pgx.Tx, claim noticeClaim, result noticeResult, now time.Time) (TransferNoticeRecipient, bool, error) {
	status := noticeResultState(claim, result)
	var next, accepted *time.Time
	var bucket *string
	if status == string(RecipientRetryWait) {
		value, err := noticeAdd(now, result.retryDelay)
		if err != nil {
			return TransferNoticeRecipient{}, false, err
		}
		next = &value
	}
	if status == string(RecipientAccepted) {
		accepted = &now
	} else {
		bucket = &result.bucket
	}
	return updateNoticeRow(ctx, tx, saveNoticeResultSQL, claim.NoticeID, claim.Kind, claim.ClaimedBy,
		claim.AttemptCount, claim.LockedUntil, now, status, next, accepted, bucket)
}

// FinishExhaustedNoticeRecipient terminates only a still-due exhausted preimage.
func (Repository) FinishExhaustedNoticeRecipient(ctx context.Context, tx pgx.Tx, before TransferNoticeRecipient, now time.Time) (TransferNoticeRecipient, bool, error) {
	bucket := "attempts_exhausted"
	if before.Status == RecipientProcessing {
		bucket = "attempts_exhausted_unknown"
	}
	return updateNoticeRow(ctx, tx, finishExhaustedNoticeRecipientSQL, before.NoticeID, before.Kind, before.Status,
		before.AttemptCount, before.ClaimedBy, before.LockedUntil, before.NextAttemptAt, now, bucket)
}

func updateNoticeRow(ctx context.Context, tx pgx.Tx, query string, args ...any) (TransferNoticeRecipient, bool, error) {
	if noticeNil(tx) {
		return TransferNoticeRecipient{}, false, noticeFailure("transaction_required", nil, true)
	}
	var r TransferNoticeRecipient
	err := tx.QueryRow(ctx, query, args...).Scan(&r.NoticeID, &r.Kind, &r.EmailSnapshot, &r.Status,
		&r.ClaimedBy, &r.LockedUntil, &r.NextAttemptAt, &r.AttemptCount, &r.MaxAttempts, &r.LastAttemptAt,
		&r.AcceptedAt, &r.FailureBucket, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, noticeDBFailure("recipient_update", err)
	}
	return normalizeNoticeRecipient(r), true, nil
}

const noticeRecipientColumns = `recipient.notice_id, recipient.recipient_kind,
	recipient.email_snapshot, recipient.status, recipient.claimed_by, recipient.locked_until,
	recipient.next_attempt_at, recipient.attempt_count, recipient.max_attempts,
	recipient.last_attempt_at, recipient.accepted_at, recipient.failure_bucket,
	recipient.created_at, recipient.updated_at`

const noticeSelect = `SELECT notice.article_ref, ` + noticeRecipientColumns + `
	FROM transfer_notice_recipients AS recipient
	JOIN transfer_notices AS notice ON notice.id=recipient.notice_id `

const noticeDuePredicate = `(recipient.status='pending'
	OR (recipient.status='retry_wait' AND recipient.next_attempt_at<=clock_timestamp())
	OR (recipient.status='processing' AND recipient.locked_until<=clock_timestamp()))`

const lockDueNoticeRecipientSQL = noticeSelect + `
	WHERE recipient.attempt_count<recipient.max_attempts AND ` + noticeDuePredicate + `
	ORDER BY recipient.created_at,recipient.notice_id,recipient.recipient_kind
	LIMIT 1 FOR UPDATE OF recipient SKIP LOCKED`

const lockExhaustedNoticeRecipientSQL = noticeSelect + `
	WHERE recipient.attempt_count=recipient.max_attempts AND ` + noticeDuePredicate + `
	ORDER BY recipient.created_at,recipient.notice_id,recipient.recipient_kind
	LIMIT 1 FOR UPDATE OF recipient SKIP LOCKED`

const readNoticeRecipientSQL = noticeSelect + ` WHERE recipient.notice_id=$1::uuid AND recipient.recipient_kind=$2`

const noticePreimagePredicate = `recipient.notice_id=$1::uuid AND recipient.recipient_kind=$2
	AND recipient.status=$3 AND recipient.attempt_count=$4
	AND recipient.claimed_by IS NOT DISTINCT FROM $5::text
	AND recipient.locked_until IS NOT DISTINCT FROM $6::timestamptz
	AND recipient.next_attempt_at IS NOT DISTINCT FROM $7::timestamptz`

const claimNoticeRecipientSQL = `UPDATE transfer_notice_recipients AS recipient
	SET status='processing',attempt_count=attempt_count+1,claimed_by=$8,
		last_attempt_at=$9,locked_until=$10,next_attempt_at=NULL,failure_bucket=NULL,accepted_at=NULL,updated_at=$9
	WHERE ` + noticePreimagePredicate + ` AND recipient.attempt_count<recipient.max_attempts
	AND (recipient.status='pending' OR (recipient.status='retry_wait' AND recipient.next_attempt_at<=$9)
		OR (recipient.status='processing' AND recipient.locked_until<=$9))
	RETURNING ` + noticeRecipientColumns

const saveNoticeResultSQL = `UPDATE transfer_notice_recipients AS recipient
	SET status=$7,next_attempt_at=$8,accepted_at=$9,failure_bucket=$10,
		claimed_by=NULL,locked_until=NULL,updated_at=$6
	WHERE recipient.notice_id=$1::uuid AND recipient.recipient_kind=$2
		AND recipient.status='processing' AND recipient.claimed_by=$3
		AND recipient.attempt_count=$4 AND recipient.locked_until=$5 AND recipient.locked_until>$6
	RETURNING ` + noticeRecipientColumns

const finishExhaustedNoticeRecipientSQL = `UPDATE transfer_notice_recipients AS recipient
	SET status='failed_terminal',claimed_by=NULL,locked_until=NULL,next_attempt_at=NULL,
		accepted_at=NULL,failure_bucket=$9,updated_at=$8
	WHERE ` + noticePreimagePredicate + ` AND recipient.attempt_count=recipient.max_attempts
	AND (recipient.status='pending' OR (recipient.status='retry_wait' AND recipient.next_attempt_at<=$8)
		OR (recipient.status='processing' AND recipient.locked_until<=$8))
	RETURNING ` + noticeRecipientColumns

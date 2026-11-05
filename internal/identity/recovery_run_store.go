package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecoveryRunStore struct {
	pool *pgxpool.Pool
}

func NewRecoveryRunStore(pool *pgxpool.Pool) (*RecoveryRunStore, error) {
	if pool == nil {
		return nil, errors.New("recovery run store pool is required")
	}
	return &RecoveryRunStore{pool: pool}, nil
}

func (store *RecoveryRunStore) Create(ctx context.Context, run recovery.RecoveryRun) (result error) {
	if store == nil || store.pool == nil || ctx == nil || !validNewRecoveryRun(run) {
		return recovery.ErrInvalidRecoveryRun
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin recovery run create: %w", err)
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	_, err = tx.Exec(ctx, insertRecoveryRunSQL,
		run.RunID, run.BackupRef, run.JournalKeyID, run.JournalKeyVersion,
		run.JournalSecretFingerprint, run.JournalStage, run.StartedAt, run.Outcome,
	)
	if err != nil {
		return mapRecoveryRunCreateError(err)
	}
	if err := commitRecoveryRun(ctx, tx, "commit recovery run create"); err != nil {
		return err
	}
	return nil
}

func mapRecoveryRunCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "recovery_runs_pkey" {
		return recovery.ErrRunAlreadyExists
	}
	return fmt.Errorf("insert recovery run: %w", err)
}

func (store *RecoveryRunStore) Load(ctx context.Context, runID string) (recovery.RecoveryRun, error) {
	if store == nil || store.pool == nil || ctx == nil || invalidStoreText(runID) {
		return recovery.RecoveryRun{}, recovery.ErrInvalidRecoveryRun
	}
	return scanPublishedRecoveryRun(store.pool.QueryRow(ctx, selectPublishedRecoveryRunSQL, runID))
}

func (store *RecoveryRunStore) Transition(
	ctx context.Context,
	runID string,
	expected recovery.RecoveryOutcome,
	next recovery.RecoveryOutcome,
	change recovery.RecoveryRunChange,
) (run recovery.RecoveryRun, result error) {
	if store == nil || store.pool == nil || ctx == nil || invalidStoreText(runID) ||
		!canonicalStoreTime(change.ChangedAt) || !validRecoveryTransition(expected, next, change) {
		return recovery.RecoveryRun{}, recovery.ErrInvalidRecoveryRun
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return recovery.RecoveryRun{}, fmt.Errorf("begin recovery run transition: %w", err)
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	current, err := scanPublishedRecoveryRun(tx.QueryRow(ctx, selectPublishedRecoveryRunSQL+` FOR UPDATE`, runID))
	if err != nil {
		return recovery.RecoveryRun{}, err
	}
	if current.Outcome != expected {
		return recovery.RecoveryRun{}, recovery.ErrRecoveryRunConflict
	}
	if !validRecoveryEvidence(current, change) {
		return recovery.RecoveryRun{}, recovery.ErrInvalidRecoveryRun
	}
	applyRecoveryChange(&current, next, change)
	tag, err := tx.Exec(ctx, updateRecoveryRunSQL,
		current.Outcome, current.JournalStage, current.JournalConsumedAt,
		nullText(current.ArticleEvidenceVersion), nullText(current.ArticleEvidenceDigest),
		nullText(current.ArticleEvidenceRef), nullTime(current.ArticleEvidenceCompletedAt),
		nullText(current.ArticleNamespace), nullTime(current.ArticleReplayVerifiedAt),
		nullTime(current.CompletedAt), nullText(current.ArticlePublicationRef),
		nullText(current.ArticlePublicationDigest), runID, expected,
	)
	if err != nil {
		return recovery.RecoveryRun{}, fmt.Errorf("update recovery run: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return recovery.RecoveryRun{}, recovery.ErrRecoveryRunConflict
	}
	if err := commitRecoveryRun(ctx, tx, "commit recovery run transition"); err != nil {
		return recovery.RecoveryRun{}, err
	}
	return current, nil
}

func validNewRecoveryRun(run recovery.RecoveryRun) bool {
	return !invalidStoreText(run.RunID) && !invalidStoreText(run.BackupRef) &&
		!invalidStoreText(run.JournalKeyID) && !invalidStoreText(run.JournalKeyVersion) &&
		!invalidStoreText(run.JournalSecretFingerprint) && run.JournalStage == recovery.StageBegun &&
		canonicalStoreTime(run.StartedAt) && run.Outcome == recovery.RecoveryOutcomeRunning
}

func validRecoveryTransition(
	expected recovery.RecoveryOutcome,
	next recovery.RecoveryOutcome,
	change recovery.RecoveryRunChange,
) bool {
	if change.ArticlePublication != nil && !(expected == recovery.RecoveryOutcomeIdentityReady && next == recovery.RecoveryOutcomeArticleDeletionReady) {
		return false
	}
	if next == recovery.RecoveryOutcomeFailed {
		return (expected == recovery.RecoveryOutcomeRunning ||
			expected == recovery.RecoveryOutcomeIdentityReady ||
			expected == recovery.RecoveryOutcomeArticleDeletionReady ||
			expected == recovery.RecoveryOutcomeReadyToConsume ||
			expected == recovery.RecoveryOutcomeJournalConsumed) &&
			change.ArticleEvidence == nil && change.JournalConsumedAt == nil
	}
	switch {
	case expected == recovery.RecoveryOutcomeRunning &&
		next == recovery.RecoveryOutcomeIdentityReady:
		return change.ArticleEvidence == nil && change.JournalConsumedAt == nil
	case expected == recovery.RecoveryOutcomeIdentityReady &&
		next == recovery.RecoveryOutcomeArticleDeletionReady:
		return change.ArticleEvidence != nil &&
			change.ArticlePublication != nil &&
			change.ArticleEvidence.RunID != "" &&
			change.ArticleEvidence.BackupRef != "" &&
			canonicalStoreTime(change.ArticleEvidenceCompletedAt) &&
			change.JournalConsumedAt == nil
	case expected == recovery.RecoveryOutcomeArticleDeletionReady &&
		next == recovery.RecoveryOutcomeReadyToConsume:
		return change.ArticleEvidence == nil && change.JournalConsumedAt == nil
	case expected == recovery.RecoveryOutcomeReadyToConsume &&
		next == recovery.RecoveryOutcomeJournalConsumed:
		return change.ArticleEvidence == nil &&
			change.JournalConsumedAt != nil && canonicalStoreTime(*change.JournalConsumedAt)
	case expected == recovery.RecoveryOutcomeJournalConsumed &&
		next == recovery.RecoveryOutcomeSucceeded:
		return change.ArticleEvidence == nil && change.JournalConsumedAt == nil
	default:
		return false
	}
}

func applyRecoveryChange(
	run *recovery.RecoveryRun,
	next recovery.RecoveryOutcome,
	change recovery.RecoveryRunChange,
) {
	run.Outcome = next
	if change.ArticlePublication != nil {
		run.ArticlePublicationRef = change.ArticlePublication.Ref
		run.ArticlePublicationDigest = change.ArticlePublication.Digest
	}
	if change.ArticleEvidence != nil {
		run.ArticleEvidenceVersion = change.ArticleEvidence.EvidenceVersion
		run.ArticleEvidenceDigest = change.ArticleEvidence.EvidenceDigest
		run.ArticleEvidenceRef = change.ArticleEvidence.EvidenceRef
		run.ArticleEvidenceCompletedAt = change.ArticleEvidenceCompletedAt
		run.ArticleNamespace = change.ArticleEvidence.Namespace
		run.ArticleReplayVerifiedAt = change.ChangedAt
	}
	if change.JournalConsumedAt != nil {
		run.JournalStage = recovery.StageConsumed
		run.JournalConsumedAt = change.JournalConsumedAt
	}
	if next == recovery.RecoveryOutcomeSucceeded {
		run.CompletedAt = change.ChangedAt
	}
}

type rowScanner interface{ Scan(...any) error }

func scanRecoveryRun(row rowScanner) (recovery.RecoveryRun, error) {
	var run recovery.RecoveryRun
	var articleVersion, articleDigest, articleRef, articleNamespace *string
	var articleCompletedAt, articleVerifiedAt, journalConsumedAt, completedAt *time.Time
	err := row.Scan(
		&run.RunID, &run.BackupRef, &run.JournalKeyID, &run.JournalKeyVersion,
		&run.JournalSecretFingerprint, &run.JournalStage, &journalConsumedAt,
		&articleVersion, &articleDigest, &articleRef, &articleCompletedAt,
		&articleNamespace, &articleVerifiedAt, &run.StartedAt, &completedAt, &run.Outcome,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return recovery.RecoveryRun{}, recovery.ErrRunNotFound
	}
	if err != nil {
		return recovery.RecoveryRun{}, fmt.Errorf("load recovery run: %w", err)
	}
	if !canonicalStoreTime(run.StartedAt) ||
		!canonicalStoreTimeOrNil(journalConsumedAt) ||
		!canonicalStoreTimeOrNil(articleCompletedAt) ||
		!canonicalStoreTimeOrNil(articleVerifiedAt) ||
		!canonicalStoreTimeOrNil(completedAt) {
		return recovery.RecoveryRun{}, fmt.Errorf(
			"%w: persisted time is not UTC microsecond precision",
			recovery.ErrInvalidRecoveryRun,
		)
	}
	run.JournalConsumedAt = journalConsumedAt
	if articleVersion != nil {
		run.ArticleEvidenceVersion = *articleVersion
	}
	if articleDigest != nil {
		run.ArticleEvidenceDigest = *articleDigest
	}
	if articleRef != nil {
		run.ArticleEvidenceRef = *articleRef
	}
	if articleCompletedAt != nil {
		run.ArticleEvidenceCompletedAt = *articleCompletedAt
	}
	if articleNamespace != nil {
		run.ArticleNamespace = *articleNamespace
	}
	if articleVerifiedAt != nil {
		run.ArticleReplayVerifiedAt = *articleVerifiedAt
	}
	if completedAt != nil {
		run.CompletedAt = *completedAt
	}
	return run, nil
}

func invalidStoreText(value string) bool { return value == "" || strings.TrimSpace(value) != value }
func canonicalStoreTime(value time.Time) bool {
	return value.Location() == time.UTC && value.Nanosecond()%1000 == 0
}
func canonicalStoreTimeOrNil(value *time.Time) bool {
	return value == nil || canonicalStoreTime(*value)
}
func nullText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

const insertRecoveryRunSQL = `
	INSERT INTO recovery_runs (
		run_id, backup_ref, journal_key_id, journal_key_version,
		journal_secret_fingerprint, journal_stage, started_at, outcome
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
const selectRecoveryRunSQL = `
	SELECT run_id, backup_ref, journal_key_id, journal_key_version,
		journal_secret_fingerprint, journal_stage,
		journal_consumed_at AT TIME ZONE 'UTC',
		article_evidence_version, article_evidence_digest, article_evidence_ref,
		article_evidence_completed_at AT TIME ZONE 'UTC',
		article_namespace, article_replay_verified_at AT TIME ZONE 'UTC',
		started_at AT TIME ZONE 'UTC', completed_at AT TIME ZONE 'UTC', outcome
	FROM recovery_runs WHERE run_id = $1
`
const selectRecoveryRunForUpdateSQL = selectRecoveryRunSQL + ` FOR UPDATE`
const updateRecoveryRunSQL = `
	UPDATE recovery_runs SET
		outcome = $1, journal_stage = $2, journal_consumed_at = $3,
		article_evidence_version = $4, article_evidence_digest = $5,
		article_evidence_ref = $6, article_evidence_completed_at = $7,
		article_namespace = $8, article_replay_verified_at = $9, completed_at = $10,
		article_publication_ref = $11, article_publication_digest = $12
	WHERE run_id = $13 AND outcome = $14
`

func validRecoveryEvidence(run recovery.RecoveryRun, change recovery.RecoveryRunChange) bool {
	v := change.ArticleEvidence
	if v == nil {
		return change.ArticlePublication == nil
	}
	return v.RunID == run.RunID && v.BackupRef == run.BackupRef &&
		v.EvidenceVersion == recovery.ReplayEvidenceVersionV1 && v.Namespace == recovery.ArticleDeletionNamespaceV1 &&
		!invalidStoreText(v.EvidenceDigest) && !invalidStoreText(v.EvidenceRef) && validStorePublication(run, change)
}

// The old 16-column scanner/query remain the lock-only deletion contract.
type publishedRunScanner struct {
	row         rowScanner
	ref, digest *string
}

func (s *publishedRunScanner) Scan(dest ...any) error {
	return s.row.Scan(append(dest, &s.ref, &s.digest)...)
}

func scanPublishedRecoveryRun(row rowScanner) (recovery.RecoveryRun, error) {
	s := &publishedRunScanner{row: row}
	r, err := scanRecoveryRun(s)
	if err != nil {
		return recovery.RecoveryRun{}, err
	}
	if (s.ref == nil) != (s.digest == nil) {
		return recovery.RecoveryRun{}, recovery.ErrInvalidRecoveryRun
	}
	if s.ref != nil {
		if invalidStoreText(*s.ref) || !recoveryMetadataDigest.MatchString(*s.digest) {
			return recovery.RecoveryRun{}, recovery.ErrInvalidRecoveryRun
		}
		r.ArticlePublicationRef, r.ArticlePublicationDigest = *s.ref, *s.digest
	}
	return r, nil
}

const selectPublishedRecoveryRunSQL = `
	SELECT run_id, backup_ref, journal_key_id, journal_key_version,
		journal_secret_fingerprint, journal_stage, journal_consumed_at AT TIME ZONE 'UTC',
		article_evidence_version, article_evidence_digest, article_evidence_ref,
		article_evidence_completed_at AT TIME ZONE 'UTC', article_namespace,
		article_replay_verified_at AT TIME ZONE 'UTC', started_at AT TIME ZONE 'UTC',
		completed_at AT TIME ZONE 'UTC', outcome, article_publication_ref, article_publication_digest
	FROM recovery_runs WHERE run_id = $1
`

func commitRecoveryRun(ctx context.Context, tx pgx.Tx, operation string) error {
	if err := tx.Commit(ctx); err != nil {
		return errors.Join(recovery.ErrRecoveryWriteUnknown, fmt.Errorf("%s: %w", operation, err))
	}
	return nil
}

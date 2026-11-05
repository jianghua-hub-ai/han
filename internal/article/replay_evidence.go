package article

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDeletionReplayRunNotFound       = errors.New("article deletion replay run not found")
	ErrDeletionReplayBackupMismatch    = errors.New("article deletion replay backup binding mismatch")
	ErrDeletionReplayInputChanged      = errors.New("article deletion replay input changed")
	ErrDeletionReplayEvidenceBad       = errors.New("article deletion replay evidence unreadable")
	ErrDeletionReplayUnsupported       = errors.New("article deletion replay evidence version unsupported")
	ErrDeletionReplayInvalidRequest    = errors.New("invalid article deletion replay request")
	ErrDeletionReplayAttemptNotOwner   = errors.New("article deletion replay attempt is not owned")
	ErrDeletionReplayAttemptSuperseded = errors.New("article deletion replay attempt superseded")
	ErrDeletionReplayAlreadyReady      = errors.New("article deletion replay attempt already ready")
)

// DeletionReplayFacts contains only current, independently refreshable facts.
// Execution outcomes are bound by the first committed immutable evidence.
type DeletionReplayFacts struct {
	JournalHeadSequence               int64
	JournalHeadHash                   string
	InventoryScope                    string
	BackupCutoff                      time.Time
	InventoryDigest                   string
	DeletionOperationsDigest          string
	AuditEventsDigest                 string
	TombstonesTotal                   int64
	DeletionOperationsReconciledCount int64
	AuditEventsVerifiedCount          int64
}

// DeletionReplayInput adds one complete successful replay's physical outcomes.
type DeletionReplayInput struct {
	DeletionReplayFacts
	TombstonesReplayed int64
	AlreadyAbsent      int64
}

type DeletionReplaySource interface {
	ReplayOldBackup(context.Context, string) (DeletionReplayInput, error)
	Snapshot(context.Context, string) (DeletionReplayFacts, error)
}

type ReplayEvidenceIDGenerator interface {
	NewID() (string, error)
}

type ReplayEvidenceClock interface {
	Now() time.Time
}

// ReplayEvidenceStore owns the article-internal attempt state machine. The
// generation/token handle returned by Begin is the only write authority for
// Ready/Fail; the public ReplayStatus DTO never carries it.
type ReplayEvidenceStore interface {
	LoadAttempt(context.Context, string, string) (AttemptSnapshot, error)
	Begin(
		context.Context, string, string, AttemptSnapshot, DeletionReplayFacts, string,
	) (BeginResult, error)
	Publish(context.Context, ReplayPublishRequest) (DeletionReplayEvidence, error)
	Fail(context.Context, string, string, AttemptSnapshot, string) error
	ReadCurrent(
		context.Context,
		string,
		string,
		func(context.Context) (DeletionReplayFacts, error),
	) (DeletionReplayRun, *DeletionReplayEvidence, error)
}

type ReplayPublishRequest struct {
	RunID       string
	BackupRef   string
	Attempt     AttemptSnapshot
	Input       DeletionReplayInput
	Refresh     func(context.Context) (DeletionReplayFacts, error)
	EvidenceIDs ReplayEvidenceIDGenerator
	Clock       ReplayEvidenceClock
}

type PostgresReplayEvidenceStore struct {
	pool                    *pgxpool.Pool
	publication             replayPublicationConstruction
	publicationTransactions replayPublicationBeginner
}

func NewPostgresReplayEvidenceStore(
	pool *pgxpool.Pool,
) (*PostgresReplayEvidenceStore, error) {
	if pool == nil {
		return nil, errors.New("article deletion replay evidence pool is required")
	}
	return &PostgresReplayEvidenceStore{pool: pool}, nil
}

// LoadAttempt performs one read-only snapshot of the run row (no lock, no
// source I/O) and never writes.
func (store *PostgresReplayEvidenceStore) LoadAttempt(
	ctx context.Context,
	runID string,
	backupRef string,
) (AttemptSnapshot, error) {
	store.publicationEntry()
	if err := store.validateRequest(ctx, runID, backupRef); err != nil {
		return AttemptSnapshot{}, err
	}
	attempt, err := scanReplayAttempt(store.pool.QueryRow(
		ctx,
		selectReplayAttemptSQL,
		runID,
		ArticleDeletionNamespace,
	))
	if errors.Is(err, ErrDeletionReplayRunNotFound) {
		return AttemptSnapshot{Exists: false}, nil
	}
	if err != nil {
		return AttemptSnapshot{}, err
	}
	return attempt, nil
}

// Begin uses current Facts without inventing pre-replay outcomes.
func (store *PostgresReplayEvidenceStore) Begin(
	ctx context.Context, runID, backupRef string, expected AttemptSnapshot,
	fresh DeletionReplayFacts, candidateToken string,
) (BeginResult, error) {
	configuration := store.publicationEntry()
	if err := store.validateRequest(ctx, runID, backupRef); err != nil {
		return BeginResult{}, err
	}
	fresh, err := normalizeReplayFacts(fresh)
	if err != nil || strings.TrimSpace(candidateToken) == "" {
		return BeginResult{}, ErrDeletionReplayInvalidRequest
	}
	if err := configuration.checkReplay(ctx, runID, backupRef); err != nil {
		return BeginResult{}, err
	}
	tx, err := configuration.transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return BeginResult{}, fmt.Errorf("begin article deletion replay run: %w", err)
	}
	defer tx.Rollback(ctx)
	attempt, err := insertAndLockReplayAttempt(ctx, tx, runID, backupRef, expected)
	if err != nil {
		return BeginResult{}, err
	}
	result, err := store.decideBegin(ctx, tx, runID, expected, fresh, candidateToken, attempt)
	if err != nil {
		return BeginResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BeginResult{}, fmt.Errorf("commit article deletion replay begin: %w", err)
	}
	return result, nil
}

func insertAndLockReplayAttempt(ctx context.Context, tx pgx.Tx, runID, backupRef string, expected AttemptSnapshot) (AttemptSnapshot, error) {
	tag, err := tx.Exec(ctx, insertReplayRunSQL, runID, ArticleDeletionNamespace, backupRef)
	if err != nil {
		return AttemptSnapshot{}, fmt.Errorf("insert article deletion replay run: %w", err)
	}
	if tag.RowsAffected() == 1 && expected.Exists {
		return AttemptSnapshot{}, ErrDeletionReplayAttemptSuperseded
	}
	attempt, err := scanReplayAttempt(tx.QueryRow(ctx, selectReplayRunForUpdateSQL, runID, ArticleDeletionNamespace))
	if err != nil {
		return AttemptSnapshot{}, err
	}
	if attempt.BackupRef != backupRef {
		return AttemptSnapshot{}, ErrDeletionReplayBackupMismatch
	}
	return attempt, nil
}

func (store *PostgresReplayEvidenceStore) decideBegin(
	ctx context.Context, tx pgx.Tx, runID string, expected AttemptSnapshot,
	fresh DeletionReplayFacts, candidate string, attempt AttemptSnapshot,
) (BeginResult, error) {
	if attempt.Status == ReplayRunReady {
		return store.beginFromReady(ctx, tx, runID, expected, fresh, candidate, attempt)
	}
	if attempt.Token != "" && attempt.Token == candidate {
		return BeginResult{Attempt: attempt, Disposition: AttemptResumedCandidate}, nil
	}
	switch {
	case attempt.Status == ReplayRunRunning && attempt.Generation == 0:
		return store.advanceAttempt(ctx, tx, runID, candidate, AttemptCreated)
	case attempt.Status == ReplayRunRunning:
		return BeginResult{Attempt: attempt, Disposition: AttemptJoinedCurrent}, nil
	case attempt.Status == ReplayRunFailed && matchesAttemptSnapshot(expected, attempt):
		return store.advanceAttempt(ctx, tx, runID, candidate, AttemptAdvancedFailed)
	default:
		return BeginResult{Attempt: attempt, Disposition: AttemptSuperseded}, nil
	}
}

func (store *PostgresReplayEvidenceStore) beginFromReady(
	ctx context.Context, tx pgx.Tx, runID string, expected AttemptSnapshot,
	fresh DeletionReplayFacts, candidate string, attempt AttemptSnapshot,
) (BeginResult, error) {
	evidence, err := readBoundReplayEvidence(ctx, tx, runID, attempt.BackupRef, attempt)
	if err != nil {
		return BeginResult{}, err
	}
	err = checkReplayCurrentFacts(runID, attempt.BackupRef, attempt, evidence, fresh)
	if err == nil {
		return BeginResult{Attempt: attempt, Disposition: AttemptAlreadyReady}, nil
	}
	if !errors.Is(err, ErrDeletionReplayInputChanged) {
		return BeginResult{}, err
	}
	if attempt.Token == candidate {
		return BeginResult{}, err
	}
	if !matchesAttemptSnapshot(expected, attempt) {
		return BeginResult{Attempt: attempt, Disposition: AttemptSuperseded}, nil
	}
	return store.advanceAttempt(ctx, tx, runID, candidate, AttemptAdvancedStale)
}

// advanceAttempt increments the generation and installs the candidate token.
func (store *PostgresReplayEvidenceStore) advanceAttempt(
	ctx context.Context,
	tx pgx.Tx,
	runID string,
	candidateToken string,
	disposition AttemptDisposition,
) (BeginResult, error) {
	var generation int64
	if err := tx.QueryRow(
		ctx,
		markReplayRunRunningSQL,
		runID,
		ArticleDeletionNamespace,
		candidateToken,
	).Scan(&generation); err != nil {
		return BeginResult{}, fmt.Errorf("mark article deletion replay running: %w", err)
	}
	attempt, err := scanReplayAttempt(tx.QueryRow(
		ctx,
		selectReplayRunForUpdateSQL,
		runID,
		ArticleDeletionNamespace,
	))
	if err != nil {
		return BeginResult{}, err
	}
	if attempt.Generation != generation || attempt.Token != candidateToken ||
		attempt.Status != ReplayRunRunning {
		return BeginResult{}, ErrDeletionReplayAttemptSuperseded
	}
	return BeginResult{Attempt: attempt, Disposition: disposition}, nil
}

// Fail writes a failure only for the exact current running owner.
func (store *PostgresReplayEvidenceStore) Fail(
	ctx context.Context,
	runID string,
	backupRef string,
	attempt AttemptSnapshot,
	reason string,
) error {
	configuration := store.publicationEntry()
	if err := store.validateRequest(ctx, runID, backupRef); err != nil {
		return err
	}
	if _, ok := replayUnavailableReasons[reason]; !ok {
		return ErrDeletionReplayInvalidRequest
	}
	if err := configuration.checkReplay(ctx, runID, backupRef); err != nil {
		return err
	}
	tx, err := configuration.transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin article deletion replay failure: %w", err)
	}
	defer tx.Rollback(ctx)
	current, err := scanReplayAttempt(tx.QueryRow(
		ctx,
		selectReplayRunForUpdateSQL,
		runID,
		ArticleDeletionNamespace,
	))
	if err != nil {
		return err
	}
	return failReplayCurrent(ctx, tx, runID, backupRef, current, attempt, reason)
}

// Publish refreshes outside the run transaction and then locks the exact owner.
func (store *PostgresReplayEvidenceStore) Publish(ctx context.Context, request ReplayPublishRequest) (evidence DeletionReplayEvidence, result error) {
	configuration := store.publicationEntry()
	if err := store.validatePublish(ctx, request); err != nil {
		return evidence, err
	}
	fresh, snapshot, err := prepareReplayPublicationPublish(ctx, configuration, request)
	if err != nil {
		return evidence, err
	}
	return store.publishPublicationRows(ctx, configuration, request, fresh, snapshot)
}

func sameReplayOwner(current, attempt AttemptSnapshot) bool {
	return current.Generation == attempt.Generation && current.Token != "" && current.Token == attempt.Token
}

func (store *PostgresReplayEvidenceStore) selectPublishEvidence(
	ctx context.Context, tx pgx.Tx, request ReplayPublishRequest, fresh DeletionReplayFacts, current AttemptSnapshot,
) (DeletionReplayEvidence, error) {
	if current.Status == ReplayRunReady {
		evidence, err := readBoundReplayEvidence(ctx, tx, request.RunID, request.BackupRef, current)
		if err != nil {
			return DeletionReplayEvidence{}, err
		}
		return evidence, checkReplayCurrentFacts(request.RunID, request.BackupRef, current, evidence, fresh)
	}
	if current.Status != ReplayRunRunning && current.Status != ReplayRunFailed {
		return DeletionReplayEvidence{}, ErrDeletionReplayAttemptNotOwner
	}
	if err := validateReplayCandidate(request, fresh); err != nil {
		return DeletionReplayEvidence{}, err
	}
	return store.reuseOrInsertEvidence(ctx, tx, request, fresh)
}

func validateReplayCandidate(request ReplayPublishRequest, fresh DeletionReplayFacts) error {
	if !sameReplayFacts(request.Input.DeletionReplayFacts, fresh) {
		return ErrDeletionReplayInputChanged
	}
	digest, err := replayInputDigest(request.RunID, request.BackupRef, request.Input)
	if err != nil {
		return errors.Join(ErrDeletionReplayInvalidRequest, err)
	}
	refreshed := request.Input
	refreshed.DeletionReplayFacts = fresh
	refreshedDigest, err := replayInputDigest(request.RunID, request.BackupRef, refreshed)
	if err != nil {
		return errors.Join(ErrDeletionReplayInvalidRequest, err)
	}
	if digest != refreshedDigest {
		return ErrDeletionReplayInputChanged
	}
	return nil
}

func setReplayCurrent(ctx context.Context, tx pgx.Tx, request ReplayPublishRequest, evidence DeletionReplayEvidence) error {
	tag, err := tx.Exec(ctx, setReplayRunCurrentSQL, evidence.EvidenceDigest, evidence.EvidenceRef,
		request.RunID, ArticleDeletionNamespace, request.Attempt.Generation, request.Attempt.Token)
	if err != nil {
		return fmt.Errorf("set current article replay evidence: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrDeletionReplayAttemptNotOwner
	}
	return nil
}

func (store *PostgresReplayEvidenceStore) reuseOrInsertEvidence(
	ctx context.Context, tx pgx.Tx, request ReplayPublishRequest, fresh DeletionReplayFacts,
) (DeletionReplayEvidence, error) {
	evidence, err := findReplayHistory(ctx, tx, request, fresh)
	if err != nil || evidence.EvidenceRef != "" {
		return evidence, err
	}
	candidate, err := newReplayEvidence(request, request.Input)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	inserted, err := insertReplayEvidenceRecord(ctx, tx, candidate)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	if inserted {
		return candidate, nil
	}
	evidence, err = loadReplayEvidenceByDigest(ctx, tx, request.RunID, ArticleDeletionNamespace,
		DeletionReplayEvidenceVersion, candidate.EvidenceDigest)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	if evidence.BackupRef != request.BackupRef || !sameReplayFacts(replayFactsFromEvidence(evidence), fresh) {
		return DeletionReplayEvidence{}, ErrDeletionReplayEvidenceBad
	}
	return evidence, nil
}

func findReplayHistory(ctx context.Context, tx pgx.Tx, request ReplayPublishRequest, fresh DeletionReplayFacts) (DeletionReplayEvidence, error) {
	rows, err := tx.Query(ctx, selectReplayHistorySQL, request.RunID, ArticleDeletionNamespace, request.BackupRef)
	if err != nil {
		return DeletionReplayEvidence{}, fmt.Errorf("read article replay history: %w", err)
	}
	defer rows.Close()
	var selected DeletionReplayEvidence
	for rows.Next() {
		evidence, err := scanReplayEvidenceFields(rows)
		if err != nil {
			return DeletionReplayEvidence{}, errors.Join(ErrDeletionReplayEvidenceBad, err)
		}
		if !sameReplayFacts(replayFactsFromEvidence(evidence), fresh) {
			continue
		}
		if err := selectReplayHistoryEvidence(&selected, evidence); err != nil {
			return DeletionReplayEvidence{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return DeletionReplayEvidence{}, fmt.Errorf("iterate article replay history: %w", err)
	}
	return selected, nil
}

func selectReplayHistoryEvidence(selected *DeletionReplayEvidence, evidence DeletionReplayEvidence) error {
	if err := validateStoredReplayEvidence(evidence); err != nil {
		return err
	}
	if selected.EvidenceRef != "" {
		return ErrDeletionReplayEvidenceBad
	}
	*selected = evidence
	return nil
}

// ReadCurrent checks only current evidence, refreshes without a run connection,
// and rechecks the complete owner/current observation before returning Ready.
func (store *PostgresReplayEvidenceStore) ReadCurrent(
	ctx context.Context, runID, backupRef string, refresh func(context.Context) (DeletionReplayFacts, error),
) (DeletionReplayRun, *DeletionReplayEvidence, error) {
	store.publicationEntry()
	if err := store.validateRequest(ctx, runID, backupRef); err != nil {
		return DeletionReplayRun{}, nil, err
	}
	if refresh == nil {
		return DeletionReplayRun{}, nil, ErrDeletionReplayInvalidRequest
	}
	before, err := store.observeReplayRun(ctx, runID)
	if err != nil {
		return DeletionReplayRun{}, nil, err
	}
	if before.run.BackupRef != backupRef {
		return DeletionReplayRun{}, nil, ErrDeletionReplayBackupMismatch
	}
	if before.run.Status != ReplayRunReady {
		return before.run, nil, nil
	}
	evidence, err := readBoundReplayEvidence(ctx, store.pool, runID, backupRef, before.attempt)
	if err != nil {
		return DeletionReplayRun{}, nil, err
	}
	fresh, err := refresh(ctx)
	if err != nil {
		return DeletionReplayRun{}, nil, err
	}
	fresh, err = normalizeReplayFacts(fresh)
	if err != nil {
		return DeletionReplayRun{}, nil, err
	}
	if err := checkReplayCurrentFacts(runID, backupRef, before.attempt, evidence, fresh); err != nil {
		return DeletionReplayRun{}, nil, err
	}
	after, err := store.observeReplayRun(ctx, runID)
	if err != nil {
		return DeletionReplayRun{}, nil, err
	}
	if !matchesAttemptSnapshot(before.attempt, after.attempt) {
		return DeletionReplayRun{}, nil, ErrDeletionReplayInputChanged
	}
	return after.run, &evidence, nil
}

type replayRunObservation struct {
	run     DeletionReplayRun
	attempt AttemptSnapshot
}

func (store *PostgresReplayEvidenceStore) observeReplayRun(ctx context.Context, runID string) (replayRunObservation, error) {
	var observed replayRunObservation
	var token *string
	run, attempt := &observed.run, &observed.attempt
	err := store.pool.QueryRow(ctx, selectReplayRunObservationSQL, runID, ArticleDeletionNamespace).Scan(
		&run.RunID, &run.Namespace, &run.BackupRef, &run.InputDigest, &run.CurrentEvidenceRef,
		&run.Status, &run.LastErrorBucket, &run.StartedAt, &run.UpdatedAt, &attempt.Generation, &token)
	if errors.Is(err, pgx.ErrNoRows) {
		return replayRunObservation{}, ErrDeletionReplayRunNotFound
	}
	if err != nil {
		return replayRunObservation{}, fmt.Errorf("read article replay observation: %w", err)
	}
	run.StartedAt, run.UpdatedAt = run.StartedAt.UTC(), run.UpdatedAt.UTC()
	attempt.Exists, attempt.Status, attempt.BackupRef = true, run.Status, run.BackupRef
	attempt.InputDigest, attempt.CurrentEvidenceRef, attempt.LastErrorBucket = run.InputDigest, run.CurrentEvidenceRef, run.LastErrorBucket
	if token != nil {
		attempt.Token = *token
	}
	return observed, nil
}

func readBoundReplayEvidence(ctx context.Context, queryer replayPoolQueryRower, runID, backupRef string, attempt AttemptSnapshot) (DeletionReplayEvidence, error) {
	if attempt.InputDigest == nil || attempt.CurrentEvidenceRef == nil {
		return DeletionReplayEvidence{}, ErrDeletionReplayEvidenceBad
	}
	evidence, err := loadReplayEvidenceByRefPool(ctx, queryer, *attempt.CurrentEvidenceRef, runID, ArticleDeletionNamespace)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	return evidence, validateReplayEvidenceBinding(runID, backupRef, attempt, evidence)
}

func validateReplayEvidenceBinding(runID, backupRef string, attempt AttemptSnapshot, evidence DeletionReplayEvidence) error {
	if err := validateStoredReplayEvidence(evidence); err != nil {
		return err
	}
	if attempt.InputDigest == nil || attempt.CurrentEvidenceRef == nil || evidence.RunID != runID ||
		evidence.Namespace != ArticleDeletionNamespace || evidence.BackupRef != backupRef ||
		evidence.EvidenceRef != *attempt.CurrentEvidenceRef || evidence.EvidenceDigest != *attempt.InputDigest {
		return ErrDeletionReplayEvidenceBad
	}
	return nil
}

func checkReplayCurrentFacts(runID, backupRef string, attempt AttemptSnapshot, evidence DeletionReplayEvidence, fresh DeletionReplayFacts) error {
	if err := validateReplayEvidenceBinding(runID, backupRef, attempt, evidence); err != nil {
		return err
	}
	if !sameReplayFacts(replayFactsFromEvidence(evidence), fresh) {
		return ErrDeletionReplayInputChanged
	}
	input := replayInputFromEvidence(evidence)
	input.DeletionReplayFacts = fresh
	digest, err := replayInputDigest(runID, backupRef, input)
	if err != nil {
		return errors.Join(ErrDeletionReplayEvidenceBad, err)
	}
	if digest != evidence.EvidenceDigest || digest != *attempt.InputDigest {
		return ErrDeletionReplayInputChanged
	}
	return nil
}

func normalizeReplayFacts(facts DeletionReplayFacts) (DeletionReplayFacts, error) {
	facts.BackupCutoff = canonicalReplayTime(facts.BackupCutoff)
	if facts.JournalHeadSequence < 0 || facts.TombstonesTotal < 0 || facts.DeletionOperationsReconciledCount < 0 ||
		facts.AuditEventsVerifiedCount < 0 || facts.BackupCutoff.IsZero() || facts.InventoryScope != BackupInventoryScopeV1 ||
		strings.TrimSpace(facts.InventoryDigest) == "" || strings.TrimSpace(facts.DeletionOperationsDigest) == "" ||
		strings.TrimSpace(facts.AuditEventsDigest) == "" || (facts.JournalHeadSequence > 0 && strings.TrimSpace(facts.JournalHeadHash) == "") {
		return DeletionReplayFacts{}, ErrDeletionReplayInvalidRequest
	}
	return facts, nil
}

func sameReplayFacts(left, right DeletionReplayFacts) bool {
	left.BackupCutoff, right.BackupCutoff = canonicalReplayTime(left.BackupCutoff), canonicalReplayTime(right.BackupCutoff)
	return left == right
}

func replayFactsFromEvidence(evidence DeletionReplayEvidence) DeletionReplayFacts {
	return DeletionReplayFacts{JournalHeadSequence: evidence.JournalHeadSequence, JournalHeadHash: evidence.JournalHeadHash,
		InventoryScope: evidence.InventoryScope, BackupCutoff: evidence.BackupCutoff, InventoryDigest: evidence.InventoryDigest,
		DeletionOperationsDigest: evidence.DeletionOperationsDigest, AuditEventsDigest: evidence.AuditEventsDigest,
		TombstonesTotal: evidence.TombstonesTotal, DeletionOperationsReconciledCount: evidence.DeletionOperationsReconciledCount,
		AuditEventsVerifiedCount: evidence.AuditEventsVerifiedCount}
}

func replayInputFromEvidence(evidence DeletionReplayEvidence) DeletionReplayInput {
	return DeletionReplayInput{DeletionReplayFacts: replayFactsFromEvidence(evidence),
		TombstonesReplayed: evidence.TombstonesReplayed, AlreadyAbsent: evidence.AlreadyAbsent}
}

func validateStoredReplayEvidence(evidence DeletionReplayEvidence) error {
	if evidence.EvidenceVersion != DeletionReplayEvidenceVersion {
		return ErrDeletionReplayUnsupported
	}
	if evidence.BackupCutoff.Nanosecond()%1000 != 0 || evidence.CompletedAt.Nanosecond()%1000 != 0 {
		return ErrDeletionReplayEvidenceBad
	}
	if _, err := encodeReadyReplayStatus(evidence); err != nil {
		return errors.Join(ErrDeletionReplayEvidenceBad, err)
	}
	return nil
}

func (store *PostgresReplayEvidenceStore) validateRequest(
	ctx context.Context,
	runID string,
	backupRef string,
) error {
	if store == nil ||
		store.pool == nil ||
		ctx == nil ||
		strings.TrimSpace(runID) == "" ||
		strings.TrimSpace(backupRef) == "" {
		return ErrDeletionReplayInvalidRequest
	}
	return nil
}

func (store *PostgresReplayEvidenceStore) validateAttemptRequest(
	ctx context.Context,
	runID string,
	backupRef string,
	attempt AttemptSnapshot,
) error {
	if err := store.validateRequest(ctx, runID, backupRef); err != nil {
		return err
	}
	if !attempt.Exists || attempt.Generation <= 0 || strings.TrimSpace(attempt.Token) == "" {
		return ErrDeletionReplayAttemptNotOwner
	}
	return nil
}

func (store *PostgresReplayEvidenceStore) validatePublish(
	ctx context.Context,
	request ReplayPublishRequest,
) error {
	if err := store.validateAttemptRequest(ctx, request.RunID, request.BackupRef, request.Attempt); err != nil {
		return err
	}
	if request.Refresh == nil || request.EvidenceIDs == nil || request.Clock == nil {
		return ErrDeletionReplayInvalidRequest
	}
	return nil
}

func matchesAttemptSnapshot(expected AttemptSnapshot, current AttemptSnapshot) bool {
	return expected.Exists &&
		expected.Status == current.Status &&
		expected.Generation == current.Generation &&
		expected.Token == current.Token &&
		expected.BackupRef == current.BackupRef &&
		equalOptionalString(expected.InputDigest, current.InputDigest) &&
		equalOptionalString(expected.CurrentEvidenceRef, current.CurrentEvidenceRef) &&
		equalOptionalString(expected.LastErrorBucket, current.LastErrorBucket)
}

func equalOptionalString(left *string, right *string) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func replayInputDigest(
	runID string,
	backupRef string,
	input DeletionReplayInput,
) (string, error) {
	status, err := readyReplayCanonical(runID, backupRef, input)
	if err != nil {
		return "", err
	}
	return status.CanonicalDigest()
}

func evidenceFromReplayInput(
	runID string,
	backupRef string,
	input DeletionReplayInput,
	evidenceRef string,
	completedAt time.Time,
) (DeletionReplayEvidence, error) {
	if strings.TrimSpace(evidenceRef) == "" || completedAt.IsZero() {
		return DeletionReplayEvidence{}, ErrDeletionReplayInvalidRequest
	}
	// F003：所有 cutoff/completed 时间在计算 digest、持久化与首次响应前统一为
	// UTC 微秒精度，避免内存纳秒值与 PostgreSQL timestamptz 微秒回读不一致，
	// 破坏 completed_at 稳定性或损坏 canonical digest。
	input.BackupCutoff = canonicalReplayTime(input.BackupCutoff)
	completedAt = canonicalReplayTime(completedAt)
	digest, err := replayInputDigest(runID, backupRef, input)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	return DeletionReplayEvidence{
		EvidenceVersion:                   DeletionReplayEvidenceVersion,
		RunID:                             runID,
		BackupRef:                         backupRef,
		Namespace:                         ArticleDeletionNamespace,
		JournalHeadSequence:               input.JournalHeadSequence,
		JournalHeadHash:                   input.JournalHeadHash,
		InventoryScope:                    input.InventoryScope,
		BackupCutoff:                      input.BackupCutoff.UTC(),
		InventoryDigest:                   input.InventoryDigest,
		DeletionOperationsDigest:          input.DeletionOperationsDigest,
		AuditEventsDigest:                 input.AuditEventsDigest,
		TombstonesTotal:                   input.TombstonesTotal,
		TombstonesReplayed:                input.TombstonesReplayed,
		AlreadyAbsent:                     input.AlreadyAbsent,
		DeletionOperationsReconciledCount: input.DeletionOperationsReconciledCount,
		AuditEventsVerifiedCount:          input.AuditEventsVerifiedCount,
		EvidenceDigest:                    digest,
		EvidenceRef:                       evidenceRef,
		CompletedAt:                       completedAt.UTC(),
	}, nil
}

// canonicalReplayTime normalizes a replay time to canonical UTC microsecond
// precision. It mirrors the recovery package's canonical time contract (F003).
func canonicalReplayTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}

func newReplayEvidence(
	request ReplayPublishRequest,
	input DeletionReplayInput,
) (DeletionReplayEvidence, error) {
	evidenceRef, err := request.EvidenceIDs.NewID()
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	return evidenceFromReplayInput(
		request.RunID,
		request.BackupRef,
		input,
		evidenceRef,
		request.Clock.Now(),
	)
}

func insertReplayEvidenceRecord(
	ctx context.Context,
	tx pgx.Tx,
	evidence DeletionReplayEvidence,
) (bool, error) {
	tag, err := tx.Exec(
		ctx,
		insertReplayEvidenceSQL,
		evidence.EvidenceVersion,
		evidence.RunID,
		evidence.BackupRef,
		evidence.Namespace,
		evidence.JournalHeadSequence,
		evidence.JournalHeadHash,
		evidence.InventoryScope,
		evidence.BackupCutoff,
		evidence.InventoryDigest,
		evidence.DeletionOperationsDigest,
		evidence.AuditEventsDigest,
		evidence.TombstonesTotal,
		evidence.TombstonesReplayed,
		evidence.AlreadyAbsent,
		evidence.DeletionOperationsReconciledCount,
		evidence.AuditEventsVerifiedCount,
		evidence.EvidenceDigest,
		evidence.EvidenceRef,
		evidence.CompletedAt,
	)
	if err != nil {
		return false, fmt.Errorf("insert article deletion replay evidence: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

type replayRowScanner interface {
	Scan(...any) error
}

func scanReplayAttempt(row replayRowScanner) (AttemptSnapshot, error) {
	var (
		attempt AttemptSnapshot
		token   *string
	)
	if err := row.Scan(
		&attempt.Status,
		&attempt.Generation,
		&token,
		&attempt.BackupRef,
		&attempt.InputDigest,
		&attempt.CurrentEvidenceRef,
		&attempt.LastErrorBucket,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AttemptSnapshot{}, ErrDeletionReplayRunNotFound
		}
		return AttemptSnapshot{}, fmt.Errorf("read article deletion replay attempt: %w", err)
	}
	if token != nil {
		attempt.Token = *token
	}
	attempt.Exists = true
	return attempt, nil
}

func scanReplayEvidence(row replayRowScanner) (DeletionReplayEvidence, error) {
	evidence, err := scanReplayEvidenceFields(row)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	return evidence, validateStoredReplayEvidence(evidence)
}

func scanReplayEvidenceFields(row replayRowScanner) (DeletionReplayEvidence, error) {
	var evidence DeletionReplayEvidence
	if err := row.Scan(
		&evidence.EvidenceVersion,
		&evidence.RunID,
		&evidence.BackupRef,
		&evidence.Namespace,
		&evidence.JournalHeadSequence,
		&evidence.JournalHeadHash,
		&evidence.InventoryScope,
		&evidence.BackupCutoff,
		&evidence.InventoryDigest,
		&evidence.DeletionOperationsDigest,
		&evidence.AuditEventsDigest,
		&evidence.TombstonesTotal,
		&evidence.TombstonesReplayed,
		&evidence.AlreadyAbsent,
		&evidence.DeletionOperationsReconciledCount,
		&evidence.AuditEventsVerifiedCount,
		&evidence.EvidenceDigest,
		&evidence.EvidenceRef,
		&evidence.CompletedAt,
	); err != nil {
		return DeletionReplayEvidence{}, err
	}
	evidence.BackupCutoff = evidence.BackupCutoff.UTC()
	evidence.CompletedAt = evidence.CompletedAt.UTC()
	return evidence, nil
}

type replayPoolQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadReplayEvidenceByRefPool(
	ctx context.Context,
	queryer replayPoolQueryRower,
	evidenceRef string,
	runID string,
	namespace string,
) (DeletionReplayEvidence, error) {
	evidence, err := scanReplayEvidence(queryer.QueryRow(
		ctx,
		selectReplayEvidenceByRefSQL,
		evidenceRef,
		runID,
		namespace,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return DeletionReplayEvidence{}, ErrDeletionReplayEvidenceBad
	}
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	return evidence, nil
}

func loadReplayEvidenceByDigest(
	ctx context.Context,
	tx pgx.Tx,
	runID string,
	namespace string,
	version string,
	digest string,
) (DeletionReplayEvidence, error) {
	return scanReplayEvidence(tx.QueryRow(
		ctx,
		selectReplayEvidenceByDigestSQL,
		runID,
		namespace,
		version,
		digest,
	))
}

const insertReplayRunSQL = `
	INSERT INTO deletion_replay_runs (
		run_id, namespace, backup_ref, status, started_at, updated_at
	) VALUES ($1, $2, $3, 'running', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	ON CONFLICT (run_id, namespace) DO NOTHING
`

const selectReplayAttemptColumns = `
	status, attempt_generation, attempt_token, backup_ref, input_digest,
	current_evidence_ref, last_error_bucket
`

const selectReplayAttemptSQL = `
	SELECT ` + selectReplayAttemptColumns + `
	FROM deletion_replay_runs
	WHERE run_id = $1 AND namespace = $2
`

const selectReplayRunForUpdateSQL = `
	SELECT ` + selectReplayAttemptColumns + `
	FROM deletion_replay_runs
	WHERE run_id = $1 AND namespace = $2
	FOR UPDATE
`

const markReplayRunRunningSQL = `
	UPDATE deletion_replay_runs
	SET status = 'running',
		last_error_bucket = NULL,
		attempt_generation = attempt_generation + 1,
		attempt_token = $3,
		updated_at = CURRENT_TIMESTAMP
	WHERE run_id = $1 AND namespace = $2
	RETURNING attempt_generation
`

const markReplayRunFailedSQL = `
	UPDATE deletion_replay_runs
	SET status = 'failed', last_error_bucket = $1, updated_at = CURRENT_TIMESTAMP
	WHERE run_id = $2 AND namespace = $3
		AND status = 'running'
		AND attempt_generation = $4
		AND attempt_token = $5
`

const replayEvidenceColumns = `
	evidence_version, run_id, backup_ref, namespace, journal_head_sequence,
	journal_head_hash, inventory_scope, backup_cutoff, inventory_digest,
	deletion_operations_digest, audit_events_digest, tombstones_total,
	tombstones_replayed, already_absent,
	deletion_operations_reconciled_count, audit_events_verified_count,
	evidence_digest, evidence_ref, completed_at
`

const selectReplayEvidenceByRefSQL = `
	SELECT ` + replayEvidenceColumns + `
	FROM deletion_replay_evidence
	WHERE evidence_ref = $1 AND run_id = $2 AND namespace = $3
`

const selectReplayEvidenceByDigestSQL = `
	SELECT ` + replayEvidenceColumns + `
	FROM deletion_replay_evidence
	WHERE run_id = $1 AND namespace = $2
		AND evidence_version = $3 AND evidence_digest = $4
`

const insertReplayEvidenceSQL = `
	INSERT INTO deletion_replay_evidence (
		` + replayEvidenceColumns + `
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19
	)
	ON CONFLICT (run_id, namespace, evidence_version, evidence_digest)
	DO NOTHING
`

const setReplayRunCurrentSQL = `
	UPDATE deletion_replay_runs
	SET input_digest = $1, current_evidence_ref = $2, status = 'ready',
		last_error_bucket = NULL, updated_at = CURRENT_TIMESTAMP
	WHERE run_id = $3 AND namespace = $4
		AND status IN ('running', 'failed')
		AND attempt_generation = $5
		AND attempt_token = $6
`

var _ ReplayEvidenceStore = (*PostgresReplayEvidenceStore)(nil)

const selectReplayRunObservationSQL = `
    SELECT run_id, namespace, backup_ref, input_digest, current_evidence_ref,
        status, last_error_bucket, started_at, updated_at, attempt_generation, attempt_token
    FROM deletion_replay_runs WHERE run_id = $1 AND namespace = $2
`

const selectReplayHistorySQL = `
    SELECT ` + replayEvidenceColumns + ` FROM deletion_replay_evidence
    WHERE run_id = $1 AND namespace = $2 AND backup_ref = $3
    ORDER BY completed_at, evidence_ref
`

func failReplayCurrent(ctx context.Context, tx pgx.Tx, runID, backupRef string, current, attempt AttemptSnapshot, reason string) error {
	if current.BackupRef != backupRef {
		return ErrDeletionReplayBackupMismatch
	}
	if current.Status == ReplayRunReady {
		return nil
	}
	if current.Status != ReplayRunRunning ||
		current.Generation != attempt.Generation ||
		current.Token != attempt.Token {
		return ErrDeletionReplayAttemptNotOwner
	}
	tag, err := tx.Exec(
		ctx,
		markReplayRunFailedSQL,
		reason,
		runID,
		ArticleDeletionNamespace,
		attempt.Generation,
		attempt.Token,
	)
	if err != nil {
		return fmt.Errorf("mark article deletion replay failed: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrDeletionReplayAttemptNotOwner
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit article deletion replay failure: %w", err)
	}
	return nil
}

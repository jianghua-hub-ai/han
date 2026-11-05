package article

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

const (
	replayPublicationVersion = "article-replay-publication/v1"
	replayPublicationAtomic  = "atomic_current_v1"
	replayPublicationLegacy  = "verified_legacy_import_v1"
	replayPublicationInstall = "install"
	replayPublicationTime    = "2006-01-02T15:04:05.000000Z"
)

var (
	errReplayPublicationUnconfigured = errors.New("article replay publication is not configured")
	errReplayPublicationSealed       = errors.New("article replay publication construction is sealed")
	errReplayPublicationUnknown      = errors.New("article replay publication outcome unknown")
)

type LegacyPublicationProof struct {
	SourceRef, SourceDigest, RunID, BackupRef, EvidenceRef, EvidenceDigest, JournalID string
	Epoch                                                                             uint64
	Head                                                                              recovery.JournalHead
	EvidenceCompletedAt, OriginalPublishedAt                                          time.Time
}

type LegacyPublicationVerifier interface {
	VerifyLegacyPublication(context.Context, LegacyPublicationProof) error
}

type PublishedReplayReader interface {
	ReadPublishedClosure(context.Context, recovery.ClosureRequest) (recovery.PublishedClosure, error)
}
type CurrentPublicationReader interface {
	ReadCurrentPublication(context.Context, string, string, string, string) (recovery.PublicationBinding, error)
}
type PublicationPublisher interface {
	EnsureCurrentPublication(context.Context, recovery.PhasePermit, string, string, string, string) (recovery.PublicationBinding, error)
	ImportLegacyPublication(context.Context, recovery.PhasePermit, LegacyPublicationProof) (recovery.PublicationBinding, error)
}

type ReplayPublicationDependencies struct {
	Source         *ProductionDeletionReplaySource
	Runtime        recovery.Runtime
	InstallationID string
	PublicationIDs ReplayEvidenceIDGenerator
	Clock          ReplayEvidenceClock
	LegacySources  LegacyPublicationVerifier
}

type replayPublicationBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type replayPublicationConfiguration struct {
	ReplayPublicationDependencies
	journal      recovery.VerifiedJournalReader
	transactions replayPublicationBeginner
}

type replayPublicationConstruction struct {
	mu            sync.Mutex
	used          bool
	configuration *replayPublicationConfiguration
}

// NewReplayPublicationPorts only seals references; it never invokes a port.
func NewReplayPublicationPorts(store *PostgresReplayEvidenceStore, d ReplayPublicationDependencies) (*PostgresReplayEvidenceStore, error) {
	if store == nil || store.pool == nil || d.Source == nil || missingReplayPublicationPort(d.Runtime) ||
		!replayPublicationText(d.InstallationID) || missingReplayPublicationPort(d.PublicationIDs) || missingReplayPublicationPort(d.Clock) {
		return nil, ErrDeletionReplayInvalidRequest
	}
	journal, err := d.Source.publicationJournal()
	if err != nil {
		return nil, err
	}
	if missingReplayPublicationPort(d.LegacySources) {
		d.LegacySources = nil
	}
	beginner := store.publicationTransactions
	if missingReplayPublicationPort(beginner) {
		beginner = store.pool
	}
	store.publication.mu.Lock()
	defer store.publication.mu.Unlock()
	if store.publication.used || store.publication.configuration != nil {
		return nil, errReplayPublicationSealed
	}
	store.publication.configuration = &replayPublicationConfiguration{ReplayPublicationDependencies: d, journal: journal, transactions: beginner}
	return store, nil
}

func missingReplayPublicationPort(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}

// The entry is deliberately before even context/argument validation.
func (store *PostgresReplayEvidenceStore) publicationEntry() *replayPublicationConfiguration {
	if store == nil {
		return nil
	}
	store.publication.mu.Lock()
	defer store.publication.mu.Unlock()
	store.publication.used = true
	return store.publication.configuration
}

func (c *replayPublicationConfiguration) checkReplay(ctx context.Context, run, backup string) error {
	if c == nil {
		return errReplayPublicationUnconfigured
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return err
	}
	b, err := c.Runtime.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	if b.Purpose != recovery.PhaseRecoveryReplay || b.InstallationID != c.InstallationID || b.RunID != run || b.BackupRef != backup {
		return recovery.ErrAdmissionBinding
	}
	return deletionAnchorContext(ctx)
}

func (c *replayPublicationConfiguration) snapshot(ctx context.Context) (recovery.VerifiedJournalSnapshot, error) {
	if j, ok := c.Source.journal.(*anchoredArticleJournal); ok && j.options.InstallationID != c.InstallationID {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	s, err := c.journal.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace)
	if err != nil {
		return s, err
	}
	if err := validateReplayPublicationSnapshot(s); err != nil {
		return recovery.VerifiedJournalSnapshot{}, err
	}
	return s, deletionAnchorContext(ctx)
}

func validateReplayPublicationSnapshot(s recovery.VerifiedJournalSnapshot) error {
	if s.Namespace != ArticleDeletionNamespace || !replayPublicationText(s.JournalID) || s.Epoch == 0 ||
		s.Epoch > math.MaxInt64 || s.Head.Sequence > math.MaxInt64 || !replayPublicationText(s.Revision) || s.Pending != nil {
		return recovery.ErrAdmissionBinding
	}
	if uint64(len(s.Prefixes)) != s.Head.Sequence+1 || len(s.Prefixes) == 0 {
		return recovery.ErrAnchorRollback
	}
	for index, head := range s.Prefixes {
		if head.Sequence != uint64(index) || !replayPublicationHead(head) {
			return recovery.ErrAnchorRollback
		}
	}
	if s.Prefixes[len(s.Prefixes)-1] != s.Head {
		return recovery.ErrAnchorRollback
	}
	return nil
}

func replayPublicationHead(head recovery.JournalHead) bool {
	if head.Sequence == 0 {
		return head.Hash == ""
	}
	return head.Sequence <= math.MaxInt64 && replayPublicationSHA(head.Hash)
}

func replayPublicationSHA(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && strings.ToLower(value) == value
}

func replayPublicationText(value string) bool {
	return value != "" && utf8.ValidString(value) && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}

type replayPublicationRecord struct {
	Ref, Version, SourceKind, RunID, Namespace, BackupRef, EvidenceRef, EvidenceDigest string
	Generation                                                                         int64
	Token                                                                              *string
	JournalID                                                                          string
	Epoch                                                                              uint64
	Head                                                                               recovery.JournalHead
	EvidenceCompletedAt, PublishedAt                                                   time.Time
	Digest, SourceRef, SourceDigest                                                    string
}

type replayPublicationCanonical struct {
	Version             string `json:"publication_version"`
	Ref                 string `json:"publication_ref"`
	SourceKind          string `json:"source_kind"`
	RunID               string `json:"run_id"`
	BackupRef           string `json:"backup_ref"`
	Namespace           string `json:"namespace"`
	EvidenceRef         string `json:"evidence_ref"`
	EvidenceDigest      string `json:"evidence_digest"`
	EvidenceCompletedAt string `json:"evidence_completed_at"`
	JournalID           string `json:"journal_id"`
	Epoch               uint64 `json:"journal_epoch"`
	Sequence            uint64 `json:"journal_head_sequence"`
	Hash                string `json:"journal_head_hash"`
	Generation          int64  `json:"attempt_generation"`
	SourceRef           string `json:"source_ref"`
	SourceDigest        string `json:"source_digest"`
	PublishedAt         string `json:"published_at"`
}

func canonicalReplayPublication(p replayPublicationRecord) ([]byte, error) {
	if err := validateReplayPublicationFields(p); err != nil {
		return nil, err
	}
	value := replayPublicationCanonical{p.Version, p.Ref, p.SourceKind, p.RunID, p.BackupRef, p.Namespace,
		p.EvidenceRef, p.EvidenceDigest, p.EvidenceCompletedAt.UTC().Format(replayPublicationTime), p.JournalID,
		p.Epoch, p.Head.Sequence, p.Head.Hash, p.Generation, p.SourceRef, p.SourceDigest, p.PublishedAt.UTC().Format(replayPublicationTime)}
	return replayPublicationJSON(value)
}

func replayPublicationJSON(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}

func replayPublicationSourceDigest(p replayPublicationRecord) (string, error) {
	raw, err := replayPublicationJSON([]any{p.RunID, p.BackupRef, p.Namespace, p.Generation, p.EvidenceRef, p.EvidenceDigest})
	if err != nil {
		return "", err
	}
	return recovery.SHA256Digest(raw), nil
}

func validateReplayPublicationFields(p replayPublicationRecord) error {
	if p.Version != replayPublicationVersion {
		return ErrDeletionReplayUnsupported
	}
	for _, text := range []string{p.Ref, p.RunID, p.BackupRef, p.EvidenceRef, p.JournalID, p.SourceRef} {
		if !replayPublicationText(text) {
			return ErrDeletionReplayEvidenceBad
		}
	}
	if !validUUID(p.Ref) || p.Namespace != ArticleDeletionNamespace || p.Generation < 0 || p.Epoch == 0 ||
		p.Epoch > math.MaxInt64 || !replayPublicationHead(p.Head) || !replayPublicationSHA(p.EvidenceDigest) || !replayPublicationSHA(p.SourceDigest) {
		return ErrDeletionReplayEvidenceBad
	}
	if (p.Generation == 0 && p.Token != nil) || (p.Generation > 0 && (p.Token == nil || strings.TrimSpace(*p.Token) == "")) {
		return ErrDeletionReplayEvidenceBad
	}
	if !replayPublicationTimeValid(p.EvidenceCompletedAt) || !replayPublicationTimeValid(p.PublishedAt) || p.PublishedAt.Before(p.EvidenceCompletedAt) {
		return ErrDeletionReplayEvidenceBad
	}
	switch p.SourceKind {
	case replayPublicationAtomic:
		digest, err := replayPublicationSourceDigest(p)
		if err != nil || p.SourceRef != p.Ref || p.SourceDigest != digest {
			return errors.Join(ErrDeletionReplayEvidenceBad, err)
		}
	case replayPublicationLegacy:
	default:
		return ErrDeletionReplayUnsupported
	}
	return nil
}

func replayPublicationTimeValid(value time.Time) bool {
	return !value.IsZero() && value.Equal(value.UTC().Truncate(time.Microsecond)) && value.Year() >= 0 && value.Year() <= 9999
}

func validateReplayPublication(p replayPublicationRecord, evidence DeletionReplayEvidence) error {
	raw, err := canonicalReplayPublication(p)
	if err != nil {
		return err
	}
	if p.Digest != recovery.SHA256Digest(raw) || p.RunID != evidence.RunID || p.BackupRef != evidence.BackupRef ||
		p.EvidenceRef != evidence.EvidenceRef || p.EvidenceDigest != evidence.EvidenceDigest || !p.EvidenceCompletedAt.Equal(evidence.CompletedAt) ||
		p.Head != (recovery.JournalHead{Sequence: uint64(evidence.JournalHeadSequence), Hash: evidence.JournalHeadHash}) {
		return ErrDeletionReplayEvidenceBad
	}
	return validateStoredReplayEvidence(evidence)
}

func (p replayPublicationRecord) binding() recovery.PublicationBinding {
	return recovery.PublicationBinding{Ref: p.Ref, Digest: p.Digest, Version: p.Version, RunID: p.RunID, BackupRef: p.BackupRef,
		EvidenceRef: p.EvidenceRef, EvidenceDigest: p.EvidenceDigest, JournalID: p.JournalID, Epoch: p.Epoch, Head: p.Head,
		EvidenceCompletedAt: p.EvidenceCompletedAt, PublishedAt: p.PublishedAt}
}

func publicationMatchesSnapshot(p replayPublicationRecord, s recovery.VerifiedJournalSnapshot, exact bool) error {
	if s.Namespace != ArticleDeletionNamespace || s.Pending != nil || p.JournalID != s.JournalID || p.Epoch != s.Epoch || (exact && p.Head != s.Head) {
		return recovery.ErrAdmissionBinding
	}
	if p.Head.Sequence >= uint64(len(s.Prefixes)) || s.Prefixes[p.Head.Sequence] != p.Head {
		return recovery.ErrAnchorRollback
	}
	return nil
}

const replayPublicationColumns = `publication_ref, publication_version, source_kind, run_id, namespace, backup_ref,
 evidence_ref, evidence_digest, attempt_generation, attempt_token, journal_id, journal_epoch,
 journal_head_sequence, journal_head_hash, evidence_completed_at, published_at, publication_digest, source_ref, source_digest`

const selectReplayPublicationCurrent = `SELECT ` + replayPublicationColumns + ` FROM article_replay_publications
 WHERE run_id=$1 AND namespace=$2 AND attempt_generation=$3 AND evidence_ref=$4 AND source_kind=$5`

const selectReplayPublicationExact = `SELECT ` + replayPublicationColumns + ` FROM article_replay_publications
 WHERE run_id=$1 AND backup_ref=$2 AND evidence_ref=$3 AND evidence_digest=$4
 AND publication_ref=$5 AND publication_digest=$6 AND namespace=$7`

const insertReplayPublication = `INSERT INTO article_replay_publications (` + replayPublicationColumns + `)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
 ON CONFLICT (run_id,namespace,attempt_generation,evidence_ref,source_kind) DO NOTHING`

func scanReplayPublication(row replayRowScanner) (replayPublicationRecord, error) {
	var p replayPublicationRecord
	var epoch, sequence int64
	err := row.Scan(&p.Ref, &p.Version, &p.SourceKind, &p.RunID, &p.Namespace, &p.BackupRef, &p.EvidenceRef,
		&p.EvidenceDigest, &p.Generation, &p.Token, &p.JournalID, &epoch, &sequence, &p.Head.Hash,
		&p.EvidenceCompletedAt, &p.PublishedAt, &p.Digest, &p.SourceRef, &p.SourceDigest)
	if err != nil {
		return p, err
	}
	if epoch <= 0 || sequence < 0 {
		return p, ErrDeletionReplayEvidenceBad
	}
	p.Epoch, p.Head.Sequence = uint64(epoch), uint64(sequence)
	p.EvidenceCompletedAt, p.PublishedAt = p.EvidenceCompletedAt.UTC(), p.PublishedAt.UTC()
	return p, validateReplayPublicationFields(p)
}

func readGenerationPublication(ctx context.Context, q replayPoolQueryRower, run string, a AttemptSnapshot, ref, kind string) (replayPublicationRecord, error) {
	return scanReplayPublication(q.QueryRow(ctx, selectReplayPublicationCurrent, run, ArticleDeletionNamespace, a.Generation, ref, kind))
}

func samePublicationAttempt(p replayPublicationRecord, a AttemptSnapshot) bool {
	if p.Generation != a.Generation {
		return false
	}
	if a.Generation == 0 {
		return p.Token == nil && a.Token == ""
	}
	return p.Token != nil && *p.Token == a.Token
}

// Reuse is resolved before IDs/Clock are called, under the existing run lock.
func appendReplayPublication(ctx context.Context, tx pgx.Tx, c *replayPublicationConfiguration, a AttemptSnapshot,
	evidence DeletionReplayEvidence, snapshot recovery.VerifiedJournalSnapshot, proof *LegacyPublicationProof,
) (replayPublicationRecord, error) {
	kind := replayPublicationAtomic
	if proof != nil {
		kind = replayPublicationLegacy
	}
	p, err := readGenerationPublication(ctx, tx, evidence.RunID, a, evidence.EvidenceRef, kind)
	if err == nil {
		return p, validateExistingPublication(p, a, evidence, snapshot, proof)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return p, err
	}
	p, err = newReplayPublication(c, a, evidence, snapshot, proof)
	if err != nil {
		return p, err
	}
	if err := deletionAnchorContext(ctx); err != nil {
		return p, err
	}
	tag, err := tx.Exec(ctx, insertReplayPublication, p.Ref, p.Version, p.SourceKind, p.RunID, p.Namespace, p.BackupRef,
		p.EvidenceRef, p.EvidenceDigest, p.Generation, p.Token, p.JournalID, int64(p.Epoch), int64(p.Head.Sequence), p.Head.Hash,
		p.EvidenceCompletedAt, p.PublishedAt, p.Digest, p.SourceRef, p.SourceDigest)
	if err != nil || tag.RowsAffected() == 1 {
		return p, err
	}
	winner, err := readGenerationPublication(ctx, tx, evidence.RunID, a, evidence.EvidenceRef, kind)
	if err != nil {
		return winner, err
	}
	return winner, validateExistingPublication(winner, a, evidence, snapshot, proof)
}

func validateExistingPublication(p replayPublicationRecord, a AttemptSnapshot, evidence DeletionReplayEvidence,
	snapshot recovery.VerifiedJournalSnapshot, proof *LegacyPublicationProof,
) error {
	if !samePublicationAttempt(p, a) {
		return ErrDeletionReplayAttemptNotOwner
	}
	if proof != nil && (p.SourceRef != proof.SourceRef || p.SourceDigest != proof.SourceDigest) {
		return ErrDeletionReplayEvidenceBad
	}
	if err := validateReplayPublication(p, evidence); err != nil {
		return err
	}
	return publicationMatchesSnapshot(p, snapshot, proof == nil)
}

func newReplayPublication(c *replayPublicationConfiguration, a AttemptSnapshot, e DeletionReplayEvidence,
	s recovery.VerifiedJournalSnapshot, proof *LegacyPublicationProof,
) (replayPublicationRecord, error) {
	ref, err := c.PublicationIDs.NewID()
	if err != nil {
		return replayPublicationRecord{}, err
	}
	p := replayPublicationRecord{Ref: ref, Version: replayPublicationVersion, SourceKind: replayPublicationAtomic,
		RunID: e.RunID, Namespace: ArticleDeletionNamespace, BackupRef: e.BackupRef, EvidenceRef: e.EvidenceRef, EvidenceDigest: e.EvidenceDigest,
		Generation: a.Generation, JournalID: s.JournalID, Epoch: s.Epoch,
		Head:                recovery.JournalHead{Sequence: uint64(e.JournalHeadSequence), Hash: e.JournalHeadHash},
		EvidenceCompletedAt: e.CompletedAt, PublishedAt: canonicalReplayTime(c.Clock.Now())}
	if a.Generation > 0 {
		token := a.Token
		p.Token = &token
	}
	p.SourceRef = p.Ref
	p.SourceDigest, err = replayPublicationSourceDigest(p)
	if proof != nil {
		p.SourceKind, p.SourceRef, p.SourceDigest = replayPublicationLegacy, proof.SourceRef, proof.SourceDigest
		if p.PublishedAt.Before(proof.OriginalPublishedAt) {
			return p, ErrDeletionReplayEvidenceBad
		}
	}
	if err != nil {
		return p, err
	}
	raw, err := canonicalReplayPublication(p)
	if err == nil {
		p.Digest = recovery.SHA256Digest(raw)
	}
	return p, err
}

func finishReplayPublicationTx(tx pgx.Tx, result *error) {
	err := tx.Rollback(context.Background())
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		*result = errors.Join(*result, err)
	}
}

// readPublicationTriplet is private so the old base Store interface stays stable.
// A nil evidence with nil error means an independently observed non-Ready owner.
func (store *PostgresReplayEvidenceStore) readPublicationTriplet(ctx context.Context, run, backup string) (evidence *DeletionReplayEvidence, result error) {
	c := store.publicationEntry()
	if err := store.validateRequest(ctx, run, backup); err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errReplayPublicationUnconfigured
	}
	before, err := c.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	a, e, p, err := store.readCurrentTripletRows(ctx, run, backup)
	if err != nil {
		return nil, err
	}
	if e == nil {
		after, err := c.snapshot(ctx)
		if err != nil || !samePublicationSnapshot(before, after) {
			return nil, errors.Join(recovery.ErrAdmissionDrift, err)
		}
		return nil, deletionAnchorContext(ctx)
	}
	fresh, err := c.Source.Snapshot(ctx, backup)
	if err != nil {
		return nil, err
	}
	if err := checkReplayCurrentFacts(run, backup, a, *e, fresh); err != nil {
		return nil, err
	}
	if err := publicationMatchesSnapshot(p, before, true); err != nil {
		return nil, err
	}
	after, err := c.snapshot(ctx)
	if err != nil || !samePublicationSnapshot(before, after) {
		return nil, errors.Join(recovery.ErrAdmissionDrift, err)
	}
	observed, err := store.observeReplayRun(ctx, run)
	if err != nil || !matchesAttemptSnapshot(a, observed.attempt) {
		return nil, errors.Join(ErrDeletionReplayInputChanged, err)
	}
	return e, nil
}

func samePublicationSnapshot(a, b recovery.VerifiedJournalSnapshot) bool {
	return a.Namespace == b.Namespace && a.JournalID == b.JournalID && a.Epoch == b.Epoch && a.Revision == b.Revision && a.Head == b.Head
}

func (store *PostgresReplayEvidenceStore) readCurrentTripletRows(ctx context.Context, run, backup string) (a AttemptSnapshot, evidence *DeletionReplayEvidence, p replayPublicationRecord, result error) {
	tx, err := store.beginPublicationTransaction(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return a, nil, p, err
	}
	defer finishReplayPublicationTx(tx, &result)
	a, err = scanReplayAttempt(tx.QueryRow(ctx, selectReplayAttemptSQL, run, ArticleDeletionNamespace))
	if errors.Is(err, ErrDeletionReplayRunNotFound) {
		// Exact run absence is definite, not a receipt-read network failure.
		var evidenceCount, publicationCount int
		if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM deletion_replay_evidence WHERE run_id=$1 AND namespace=$2),(SELECT count(*) FROM article_replay_publications WHERE run_id=$1 AND namespace=$2)`, run, ArticleDeletionNamespace).Scan(&evidenceCount, &publicationCount); err != nil {
			return a, nil, p, err
		}
		if evidenceCount != 0 || publicationCount != 0 {
			return a, nil, p, ErrDeletionReplayEvidenceBad
		}
		return a, nil, p, tx.Commit(ctx)
	}
	if err != nil {
		return a, nil, p, err
	}
	if a.BackupRef != backup {
		return a, nil, p, ErrDeletionReplayBackupMismatch
	}
	if a.Status != ReplayRunReady {
		if err := checkUnreadyPublicationRows(ctx, tx, run, a); err != nil {
			return a, nil, p, err
		}
		return a, nil, p, tx.Commit(ctx)
	}
	e, err := readBoundReplayEvidence(ctx, tx, run, backup, a)
	if err != nil {
		return a, nil, p, err
	}
	p, err = readGenerationPublication(ctx, tx, run, a, e.EvidenceRef, replayPublicationAtomic)
	if err != nil {
		return a, nil, p, errors.Join(errReplayPublicationUnknown, err)
	}
	if err := validateReplayPublication(p, e); err != nil {
		return a, nil, p, err
	}
	if !samePublicationAttempt(p, a) {
		return a, nil, p, ErrDeletionReplayEvidenceBad
	}
	return a, &e, p, tx.Commit(ctx)
}

func (store *PostgresReplayEvidenceStore) ReadCurrentPublication(ctx context.Context, run, backup, ref, digest string) (binding recovery.PublicationBinding, result error) {
	c := store.publicationEntry()
	if err := store.validateRequest(ctx, run, backup); err != nil {
		return binding, err
	}
	if c == nil {
		return binding, errReplayPublicationUnconfigured
	}
	_, e, err := store.ReadCurrent(ctx, run, backup, func(ctx context.Context) (DeletionReplayFacts, error) { return c.Source.Snapshot(ctx, backup) })
	if err != nil {
		return binding, err
	}
	if e == nil || e.EvidenceRef != ref || e.EvidenceDigest != digest {
		return binding, recovery.ErrPublicationMissing
	}
	a, current, p, err := store.readCurrentTripletRows(ctx, run, backup)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errReplayPublicationUnknown) && errors.Is(err, pgx.ErrNoRows) {
		return binding, recovery.ErrPublicationMissing
	}
	if err != nil {
		return binding, err
	}
	if current == nil || current.EvidenceRef != ref || current.EvidenceDigest != digest {
		return binding, recovery.ErrAdmissionDrift
	}
	s, err := c.snapshot(ctx)
	if err != nil {
		return binding, err
	}
	if err := publicationMatchesSnapshot(p, s, true); err != nil {
		return binding, err
	}
	after, err := store.observeReplayRun(ctx, run)
	if err != nil || !matchesAttemptSnapshot(a, after.attempt) {
		return binding, errors.Join(ErrDeletionReplayInputChanged, err)
	}
	return p.binding(), deletionAnchorContext(ctx)
}

func (store *PostgresReplayEvidenceStore) ReadPublishedClosure(ctx context.Context, q recovery.ClosureRequest) (closure recovery.PublishedClosure, result error) {
	c := store.publicationEntry()
	if err := store.validateRequest(ctx, q.RunID, q.BackupRef); err != nil {
		return closure, err
	}
	if c == nil {
		return closure, errReplayPublicationUnconfigured
	}
	for _, value := range []string{q.EvidenceRef, q.EvidenceDigest, q.PublicationRef, q.PublicationDigest} {
		if !replayPublicationText(value) {
			return closure, ErrDeletionReplayInvalidRequest
		}
	}
	before, err := c.snapshot(ctx)
	if err != nil {
		return closure, err
	}
	p, evidence, err := store.readHistoricalRows(ctx, q)
	if err != nil {
		return closure, err
	}
	after, err := c.snapshot(ctx)
	if err != nil || !samePublicationSnapshot(before, after) {
		return closure, errors.Join(recovery.ErrAdmissionDrift, err)
	}
	if err := publicationMatchesSnapshot(p, after, false); err != nil {
		return closure, err
	}
	ready, err := readyReplayStatus(evidence)
	if err != nil {
		return closure, err
	}
	closure = recovery.PublishedClosure{Ready: ready, PublicationRef: p.Ref, PublicationDigest: p.Digest, JournalID: p.JournalID,
		Epoch: p.Epoch, PublishedAt: p.PublishedAt, VerifiedArticleRevision: after.Revision, CurrentHead: after.Head, HistoricalHead: p.Head}
	return closure, deletionAnchorContext(ctx)
}

func (store *PostgresReplayEvidenceStore) readHistoricalRows(ctx context.Context, q recovery.ClosureRequest) (p replayPublicationRecord, e DeletionReplayEvidence, result error) {
	tx, err := store.beginPublicationTransaction(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return p, e, err
	}
	defer finishReplayPublicationTx(tx, &result)
	p, err = scanReplayPublication(tx.QueryRow(ctx, selectReplayPublicationExact, q.RunID, q.BackupRef, q.EvidenceRef, q.EvidenceDigest, q.PublicationRef, q.PublicationDigest, ArticleDeletionNamespace))
	if err != nil {
		return p, e, err
	}
	e, err = loadReplayEvidenceByRefPool(ctx, tx, q.EvidenceRef, q.RunID, ArticleDeletionNamespace)
	if err != nil {
		return p, e, err
	}
	if err := validateReplayPublication(p, e); err != nil {
		return p, e, err
	}
	return p, e, tx.Commit(ctx)
}

func (store *PostgresReplayEvidenceStore) EnsureCurrentPublication(ctx context.Context, permit recovery.PhasePermit, run, backup, ref, digest string) (binding recovery.PublicationBinding, result error) {
	c := store.publicationEntry()
	if err := store.validateRequest(ctx, run, backup); err != nil {
		return binding, err
	}
	if c == nil {
		return binding, errReplayPublicationUnconfigured
	}
	if err := c.Runtime.CheckPhase(ctx, permit, recovery.PhaseRecoveryReplay); err != nil {
		return binding, err
	}
	if err := c.checkReplay(ctx, run, backup); err != nil {
		return binding, err
	}
	_, evidence, err := store.ReadCurrent(ctx, run, backup, func(ctx context.Context) (DeletionReplayFacts, error) { return c.Source.Snapshot(ctx, backup) })
	if err != nil || evidence == nil {
		return binding, errors.Join(ErrDeletionReplayEvidenceBad, err)
	}
	if evidence.EvidenceRef != ref || evidence.EvidenceDigest != digest {
		return binding, ErrDeletionReplayEvidenceBad
	}
	s, err := c.snapshot(ctx)
	if err != nil {
		return binding, err
	}
	if s.Head != (recovery.JournalHead{Sequence: uint64(evidence.JournalHeadSequence), Hash: evidence.JournalHeadHash}) {
		return binding, recovery.ErrAdmissionDrift
	}
	return store.ensurePublicationRows(ctx, c, permit, *evidence, s)
}

func (store *PostgresReplayEvidenceStore) ensurePublicationRows(ctx context.Context, c *replayPublicationConfiguration, permit recovery.PhasePermit, e DeletionReplayEvidence, s recovery.VerifiedJournalSnapshot) (binding recovery.PublicationBinding, result error) {
	if err := c.checkReplay(ctx, e.RunID, e.BackupRef); err != nil {
		return binding, err
	}
	tx, err := c.transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return binding, err
	}
	defer finishReplayPublicationTx(tx, &result)
	a, err := scanReplayAttempt(tx.QueryRow(ctx, selectReplayRunForUpdateSQL, e.RunID, ArticleDeletionNamespace))
	if err != nil {
		return binding, err
	}
	if err := validateReplayEvidenceBinding(e.RunID, e.BackupRef, a, e); err != nil {
		return binding, err
	}
	if a.Status != ReplayRunReady {
		return binding, ErrDeletionReplayAttemptNotOwner
	}
	p, err := appendReplayPublication(ctx, tx, c, a, e, s, nil)
	if err != nil {
		return binding, err
	}
	if err := c.Runtime.CheckPhase(ctx, permit, recovery.PhaseRecoveryReplay); err != nil {
		return binding, err
	}
	if err := c.checkReplay(ctx, e.RunID, e.BackupRef); err != nil {
		return binding, err
	}
	if err := tx.Commit(ctx); err != nil {
		return binding, errors.Join(errReplayPublicationUnknown, err)
	}
	return p.binding(), nil
}

func (store *PostgresReplayEvidenceStore) ImportLegacyPublication(ctx context.Context, permit recovery.PhasePermit, proof LegacyPublicationProof) (binding recovery.PublicationBinding, result error) {
	c := store.publicationEntry()
	if err := store.validateRequest(ctx, proof.RunID, proof.BackupRef); err != nil {
		return binding, err
	}
	if c == nil || c.LegacySources == nil {
		return binding, errReplayPublicationUnconfigured
	}
	j, ok := c.Source.journal.(*anchoredArticleJournal)
	if !ok || j.control == nil {
		return binding, recovery.ErrAdmissionBinding
	}
	// Reuse the real held namespace; no nested acquire/read call under this gate.
	result = j.withLock(ctx, false, func() error {
		if err := checkReplayImport(ctx, c, j, permit, proof); err != nil {
			return err
		}
		s, _, err := j.snapshot(ctx)
		if err != nil {
			return err
		}
		if err := validateReplayLegacyProof(proof, s); err != nil {
			return err
		}
		if err := c.LegacySources.VerifyLegacyPublication(ctx, proof); err != nil {
			return err
		}
		if err := checkReplayImport(ctx, c, j, permit, proof); err != nil {
			return err
		}
		again, _, err := j.snapshot(ctx)
		if err != nil || !samePublicationSnapshot(s, again) {
			return errors.Join(recovery.ErrAdmissionDrift, err)
		}
		binding, err = store.importPublicationRows(ctx, c, j, permit, proof, again)
		return err
	})
	return binding, result
}

func checkReplayImport(ctx context.Context, c *replayPublicationConfiguration, j *anchoredArticleJournal, permit recovery.PhasePermit, proof LegacyPublicationProof) error {
	if err := c.Runtime.CheckPhase(ctx, permit, replayPublicationInstall); err != nil {
		return err
	}
	b, err := c.Runtime.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	if b.Purpose != replayPublicationInstall || b.InstallationID != c.InstallationID || j.options.InstallationID != c.InstallationID ||
		j.options.JournalID != proof.JournalID || j.options.Epoch != proof.Epoch {
		return recovery.ErrAdmissionBinding
	}
	if _, _, err := c.Runtime.CheckArticleCoordination(ctx, permit, c.InstallationID, proof.JournalID, proof.Epoch); err != nil {
		return err
	}
	return j.checkControl(ctx)
}

func validateReplayLegacyProof(p LegacyPublicationProof, s recovery.VerifiedJournalSnapshot) error {
	if !replayPublicationText(p.SourceRef) || !replayPublicationSHA(p.SourceDigest) || !replayPublicationSHA(p.EvidenceDigest) ||
		!replayPublicationText(p.EvidenceRef) || !replayPublicationTimeValid(p.EvidenceCompletedAt) ||
		!replayPublicationTimeValid(p.OriginalPublishedAt) || p.OriginalPublishedAt.Before(p.EvidenceCompletedAt) {
		return ErrDeletionReplayEvidenceBad
	}
	if p.JournalID != s.JournalID || p.Epoch != s.Epoch || !replayPublicationHead(p.Head) ||
		p.Head.Sequence >= uint64(len(s.Prefixes)) || s.Prefixes[p.Head.Sequence] != p.Head {
		return recovery.ErrAnchorRollback
	}
	return nil
}

func (store *PostgresReplayEvidenceStore) importPublicationRows(ctx context.Context, c *replayPublicationConfiguration, j *anchoredArticleJournal,
	permit recovery.PhasePermit, proof LegacyPublicationProof, s recovery.VerifiedJournalSnapshot,
) (binding recovery.PublicationBinding, result error) {
	tx, err := c.transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return binding, err
	}
	defer finishReplayPublicationTx(tx, &result)
	a, err := scanReplayAttempt(tx.QueryRow(ctx, selectReplayRunForUpdateSQL, proof.RunID, ArticleDeletionNamespace))
	if err != nil {
		return binding, err
	}
	if a.BackupRef != proof.BackupRef {
		return binding, ErrDeletionReplayBackupMismatch
	}
	e, err := loadReplayEvidenceByRefPool(ctx, tx, proof.EvidenceRef, proof.RunID, ArticleDeletionNamespace)
	if err != nil {
		return binding, err
	}
	if e.BackupRef != proof.BackupRef || e.EvidenceDigest != proof.EvidenceDigest || !e.CompletedAt.Equal(proof.EvidenceCompletedAt) ||
		(recovery.JournalHead{Sequence: uint64(e.JournalHeadSequence), Hash: e.JournalHeadHash}) != proof.Head {
		return binding, ErrDeletionReplayEvidenceBad
	}
	if err := validateStoredReplayEvidence(e); err != nil {
		return binding, err
	}
	if err := checkReplayImport(ctx, c, j, permit, proof); err != nil {
		return binding, err
	}
	p, err := appendReplayPublication(ctx, tx, c, a, e, s, &proof)
	if err != nil {
		return binding, err
	}
	if err := checkReplayImport(ctx, c, j, permit, proof); err != nil {
		return binding, err
	}
	if err := tx.Commit(ctx); err != nil {
		return binding, errors.Join(errReplayPublicationUnknown, err)
	}
	return p.binding(), nil
}

var _ recovery.PublicationPort = (*PostgresReplayEvidenceStore)(nil)
var _ recovery.PublishedClosureReader = (*PostgresReplayEvidenceStore)(nil)

func (store *PostgresReplayEvidenceStore) beginPublicationTransaction(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	c := store.publicationEntry()
	if c == nil {
		return nil, errReplayPublicationUnconfigured
	}
	return c.transactions.BeginTx(ctx, options)
}

func prepareReplayPublicationPublish(ctx context.Context, c *replayPublicationConfiguration, request ReplayPublishRequest) (fresh DeletionReplayFacts, snapshot recovery.VerifiedJournalSnapshot, result error) {
	if err := c.checkReplay(ctx, request.RunID, request.BackupRef); err != nil {
		return fresh, snapshot, err
	}
	before, err := c.snapshot(ctx)
	if err != nil {
		return fresh, snapshot, err
	}
	// The configured same-source reader is trusted; caller callbacks are not metadata authority.
	fresh, err = c.Source.Snapshot(ctx, request.BackupRef)
	if err != nil {
		return fresh, snapshot, err
	}
	fresh, err = normalizeReplayFacts(fresh)
	if err != nil {
		return fresh, snapshot, err
	}
	snapshot, err = c.snapshot(ctx)
	if err != nil || !samePublicationSnapshot(before, snapshot) || fresh.JournalHeadSequence < 0 ||
		(recovery.JournalHead{Sequence: uint64(fresh.JournalHeadSequence), Hash: fresh.JournalHeadHash}) != snapshot.Head {
		return fresh, snapshot, errors.Join(ErrDeletionReplayInputChanged, err)
	}
	return fresh, snapshot, c.checkReplay(ctx, request.RunID, request.BackupRef)
}

func (store *PostgresReplayEvidenceStore) publishPublicationRows(ctx context.Context, c *replayPublicationConfiguration, request ReplayPublishRequest,
	fresh DeletionReplayFacts, snapshot recovery.VerifiedJournalSnapshot,
) (evidence DeletionReplayEvidence, result error) {
	defer clearFailedPublicationEvidence(&evidence, &result)
	tx, err := c.transactions.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return evidence, err
	}
	defer finishReplayPublicationTx(tx, &result)
	current, err := scanReplayAttempt(tx.QueryRow(ctx, selectReplayRunForUpdateSQL, request.RunID, ArticleDeletionNamespace))
	if err != nil {
		return evidence, err
	}
	if current.BackupRef != request.BackupRef {
		return evidence, ErrDeletionReplayBackupMismatch
	}
	if !sameReplayOwner(current, request.Attempt) {
		return evidence, ErrDeletionReplayAttemptNotOwner
	}
	evidence, err = store.selectPublishEvidence(ctx, tx, request, fresh, current)
	if err != nil {
		return evidence, err
	}
	if current.Status == ReplayRunReady {
		p, err := readGenerationPublication(ctx, tx, request.RunID, current, evidence.EvidenceRef, replayPublicationAtomic)
		if err != nil {
			// Legacy reassertion belongs to Ensure, not an incidental late Publish.
			return evidence, errors.Join(errReplayPublicationUnknown, err)
		}
		if err := validateExistingPublication(p, current, evidence, snapshot, nil); err != nil {
			return evidence, err
		}
	} else {
		if err := setReplayCurrent(ctx, tx, request, evidence); err != nil {
			return evidence, err
		}
		if _, err := appendReplayPublication(ctx, tx, c, current, evidence, snapshot, nil); err != nil {
			return evidence, err
		}
	}
	if err := c.checkReplay(ctx, request.RunID, request.BackupRef); err != nil {
		return evidence, err
	}
	if err := tx.Commit(ctx); err != nil {
		return evidence, errors.Join(errReplayPublicationUnknown, err)
	}
	return evidence, nil
}

// Clear after transaction finalization, including any joined rollback error.
// Unknown commits remain errors and require independent durable readback.
func clearFailedPublicationEvidence(evidence *DeletionReplayEvidence, result *error) {
	if *result != nil {
		*evidence = DeletionReplayEvidence{}
	}
}

func checkUnreadyPublicationRows(ctx context.Context, tx pgx.Tx, run string, a AttemptSnapshot) error {
	// Observe the same generation's receipt table even when current is pending.
	var count, evidenceCount int
	err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM article_replay_publications WHERE run_id=$1 AND namespace=$2 AND attempt_generation=$3 AND source_kind=$4),(SELECT count(*) FROM deletion_replay_evidence WHERE run_id=$1 AND namespace=$2)`, run, ArticleDeletionNamespace, a.Generation, replayPublicationAtomic).Scan(&count, &evidenceCount)
	if err != nil || count != 0 {
		return errors.Join(ErrDeletionReplayEvidenceBad, err)
	}
	if a.CurrentEvidenceRef != nil {
		if _, err := loadReplayEvidenceByRefPool(ctx, tx, *a.CurrentEvidenceRef, run, ArticleDeletionNamespace); err != nil {
			return err
		}
	}
	return nil
}

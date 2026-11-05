package recovery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const RecoveryNamespaceV1 = "recovery/v1"

var (
	ErrJournalCorrupt      = errors.New("recovery journal corrupt")
	ErrJournalBusy         = errors.New("recovery journal busy")
	ErrRunNotFound         = errors.New("recovery run not found")
	ErrRunAlreadyExists    = errors.New("recovery run already exists")
	ErrSecretReused        = errors.New("recovery secret already used")
	ErrJournalMismatch     = errors.New("recovery journal mismatch")
	ErrUnsupportedPlatform = errors.New("recovery journal unsupported platform")
	ErrAnchorConflict      = errors.New("journal head anchor conflict")
	ErrAnchorStaleRevision = errors.New("journal head anchor stale revision")
	ErrAnchorMissing       = errors.New("journal head anchor missing")
	ErrAnchorRollback      = errors.New("journal head anchor rollback")
	ErrAnchorIndeterminate = errors.New("journal head anchor result indeterminate")
	ErrInvalidRecoveryRun  = errors.New("invalid recovery run")
	ErrRecoveryRunConflict = errors.New("recovery run state conflict")
)

type Stage string

const (
	StageBegun    Stage = "begun"
	StageConsumed Stage = "consumed"
)

type RecoveryOutcome string

const (
	RecoveryOutcomeRunning              RecoveryOutcome = "running"
	RecoveryOutcomeFailed               RecoveryOutcome = "failed"
	RecoveryOutcomeIdentityReady        RecoveryOutcome = "identity_ready"
	RecoveryOutcomeArticleDeletionReady RecoveryOutcome = "article_deletion_ready"
	RecoveryOutcomeReadyToConsume       RecoveryOutcome = "ready_to_consume"
	RecoveryOutcomeJournalConsumed      RecoveryOutcome = "journal_consumed"
	RecoveryOutcomeSucceeded            RecoveryOutcome = "succeeded"
)

type JournalHead struct {
	Sequence uint64 `json:"sequence"`
	Hash     string `json:"hash"`
}

type AnchorPending struct {
	OldHead     JournalHead `json:"old_head"`
	NewHead     JournalHead `json:"new_head"`
	OperationID string      `json:"operation_id"`
}

type AnchorSnapshot struct {
	JournalID string         `json:"journal_id"`
	Namespace string         `json:"namespace"`
	Epoch     uint64         `json:"epoch"`
	Committed JournalHead    `json:"committed"`
	Pending   *AnchorPending `json:"pending,omitempty"`
}

type JournalHeadAnchor interface {
	Load(ctx context.Context, namespace string) (AnchorSnapshot, string, error)
	CompareAndSwap(
		ctx context.Context,
		namespace string,
		expectedRevision string,
		next AnchorSnapshot,
	) (AnchorSnapshot, string, error)
}

// VerifiedJournalSnapshot is a complete, side-effect-free committed observation.
// Revisions are opaque, non-reused tokens belonging to one namespace only.
type VerifiedJournalSnapshot struct {
	JournalID string
	Namespace string
	Epoch     uint64
	Head      JournalHead
	Revision  string
	Pending   *AnchorPending
	Entries   []JournalEntry
	Prefixes  []JournalHead
}

// VerifiedJournalReader never initializes media or repairs pending writes.
type VerifiedJournalReader interface {
	ReadVerifiedSnapshot(context.Context, string) (VerifiedJournalSnapshot, error)
}

// ContextRecoveryJournal is the contextual port; S4.8 owns its I/O implementation.
// VerifyBeginSecret uses the retained Begin key and never mutates Journal state.
type ContextRecoveryJournal interface {
	VerifiedJournalReader
	Begin(context.Context, string, string, []byte) (JournalEntry, error)
	VerifyBeginSecret(context.Context, RunBinding, []byte) (JournalEntry, error)
	MarkConsumed(context.Context, string) (JournalEntry, error)
	Read(context.Context, string) (JournalEntry, error)
	VerifyAgainst(context.Context, RecoveryRun) (JournalEntry, error)
}

// PublicationBinding identifies an immutable article publication.
type PublicationBinding struct {
	Ref, Digest, Version, RunID, BackupRef string
	EvidenceRef, EvidenceDigest, JournalID string
	Epoch                                  uint64
	Head                                   JournalHead
	EvidenceCompletedAt, PublishedAt       time.Time
}

// ErrPublicationMissing is the definite absence of the exact current receipt.
var ErrPublicationMissing = errors.New("recovery publication missing")

// PublicationPort confirms a fresh exact source; Ensure requires a replay grant.
type PublicationPort interface {
	ReadCurrentPublication(context.Context, string, string, string, string) (PublicationBinding, error)
	EnsureCurrentPublication(context.Context, PhasePermit, string, string, string, string) (PublicationBinding, error)
}

// ClosureRequest selects the exact persisted historical publication.
type ClosureRequest struct {
	RunID, BackupRef, EvidenceRef, EvidenceDigest string
	PublicationRef, PublicationDigest             string
}

// PublishedClosure contains independently verified historical article evidence.
type PublishedClosure struct {
	Ready                                        ReadyReplayStatusV1
	PublicationRef, PublicationDigest, JournalID string
	Epoch                                        uint64
	PublishedAt                                  time.Time
	VerifiedArticleRevision                      string
	CurrentHead, HistoricalHead                  JournalHead
}

// PublishedClosureReader reads history without refreshing current Facts.
type PublishedClosureReader interface {
	ReadPublishedClosure(context.Context, ClosureRequest) (PublishedClosure, error)
}

type JournalEntry struct {
	Namespace          string
	Sequence           uint64
	RunID              string
	BackupRef          string
	KeyID              string
	KeyVersion         string
	SecretFingerprint  string
	Stage              Stage
	RecordedAt         time.Time
	PreviousRecordHash string
	RecordHMAC         string
}

type RecoveryRun struct {
	RunID                      string
	BackupRef                  string
	JournalKeyID               string
	JournalKeyVersion          string
	JournalSecretFingerprint   string
	JournalStage               Stage
	StartedAt                  time.Time
	JournalConsumedAt          *time.Time
	ArticleEvidenceVersion     string
	ArticleEvidenceDigest      string
	ArticleEvidenceRef         string
	ArticleEvidenceCompletedAt time.Time
	ArticleNamespace           string
	ArticleReplayVerifiedAt    time.Time
	CompletedAt                time.Time
	Outcome                    RecoveryOutcome
	ArticlePublicationRef      string
	ArticlePublicationDigest   string
}

type RecoveryRunChange struct {
	ArticlePublication         *PublicationBinding
	ArticleEvidence            *ReadyReplayStatusV1
	ArticleEvidenceCompletedAt time.Time
	JournalConsumedAt          *time.Time
	ChangedAt                  time.Time
}

type RecoveryRunStore interface {
	Load(ctx context.Context, runID string) (RecoveryRun, error)
	Transition(
		ctx context.Context,
		runID string,
		expected RecoveryOutcome,
		next RecoveryOutcome,
		change RecoveryRunChange,
	) (RecoveryRun, error)
}

type RecoveryJournal = ContextRecoveryJournal

type JournalOptions struct {
	ReadMaxBytes, ReadMaxRecords int64
	Path                         string
	FileMode                     fs.FileMode
	Now                          func() time.Time
	JournalID                    string
	Epoch                        uint64
	Anchor                       JournalHeadAnchor
	Bootstrap                    bool
	KeyRetirementVerifier        KeyRetirementCapabilityVerifier
}

type FileRecoveryJournal struct {
	readOnly                     bool
	readMaxBytes, readMaxRecords int64
	path                         string
	fileMode                     fs.FileMode
	now                          func() time.Time
	journalID                    string
	epoch                        uint64
	anchor                       JournalHeadAnchor
	keyring                      Keyring

	keyRetirementVerifier KeyRetirementCapabilityVerifier
	acquireLock           func(context.Context, string, fs.FileMode) (*journalFileLock, error)
	syncDirectory         func(string) error
	bootstrapPending      bool
}

func NewFileRecoveryJournal(ctx context.Context, options JournalOptions, keyring Keyring) (*FileRecoveryJournal, error) {
	if err := checkedContext(ctx); err != nil {
		return nil, err
	}
	if !fileJournalSupported() {
		return nil, ErrUnsupportedPlatform
	}
	if err := validateJournalOptions(options); err != nil {
		return nil, err
	}
	if _, err := keyring.activeReference(); err != nil {
		return nil, err
	}
	journal := &FileRecoveryJournal{
		path: options.Path, fileMode: options.FileMode, now: options.Now,
		readMaxBytes: options.ReadMaxBytes, readMaxRecords: options.ReadMaxRecords,
		journalID: options.JournalID, epoch: options.Epoch, anchor: options.Anchor,
		keyring: keyring, keyRetirementVerifier: options.KeyRetirementVerifier,
		acquireLock: acquireJournalFileLock, syncDirectory: syncDirectoryPath,
	}
	if err := journal.validatePath(); err != nil {
		return nil, err
	}
	if err := journal.withLock(ctx, func(ctx context.Context) error {
		bootstrapped, err := journal.bootstrapAnchor(ctx, options.Bootstrap)
		if err != nil {
			return err
		}
		journal.bootstrapPending = bootstrapped
		_, err = journal.loadRecords(ctx, bootstrapped)
		return err
	}); err != nil {
		return nil, err
	}
	return journal, nil
}

func validateJournalOptions(options JournalOptions) error {
	if options.Path == "" || strings.TrimSpace(options.Path) != options.Path ||
		!admissionText(options.JournalID) ||
		options.Epoch == 0 || missingRuntimePort(options.Anchor) || options.Now == nil || options.ReadMaxBytes <= 0 || options.ReadMaxRecords <= 0 {
		return fmt.Errorf("%w: invalid journal options", ErrJournalCorrupt)
	}
	if options.FileMode.Perm() == 0 || options.FileMode.Perm()&0o077 != 0 {
		return fmt.Errorf("%w: file mode is not owner-only", ErrJournalCorrupt)
	}
	return nil
}

func (keyring Keyring) activeReference() (Key, error) {
	if len(keyring.keys) == 0 || len(keyring.active.Material) == 0 {
		return Key{}, fmt.Errorf("%w: uninitialized keyring", ErrInvalidKeyring)
	}
	return keyring.activeKey(), nil
}

func (journal *FileRecoveryJournal) validatePath() error {
	parent := filepath.Dir(journal.path)
	info, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("%w: journal parent unavailable: %v", ErrJournalCorrupt, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: journal parent is not an owner-only directory", ErrJournalCorrupt)
	}
	info, err = os.Lstat(journal.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: journal unavailable: %v", ErrJournalCorrupt, err)
	}
	return journal.validateFileInfo(info)
}

func (journal *FileRecoveryJournal) validateFileInfo(info fs.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: journal is not an owner-only regular file", ErrJournalCorrupt)
	}
	return nil
}

func (journal *FileRecoveryJournal) bootstrapAnchor(ctx context.Context, allowed bool) (bool, error) {
	snapshot, revision, err := journal.readWriterAnchor(ctx)
	if err == nil {
		return false, journal.validateAnchorSnapshot(snapshot, revision)
	}
	if !errors.Is(err, ErrAnchorMissing) || !allowed {
		return false, err
	}
	if _, statErr := os.Lstat(journal.path); !errors.Is(statErr, os.ErrNotExist) {
		return false, fmt.Errorf("%w: bootstrap requires missing journal", ErrJournalCorrupt)
	}
	artifacts, artifactErr := journal.tempArtifactPaths(ctx)
	if artifactErr != nil {
		return false, artifactErr
	}
	if len(artifacts) != 0 {
		return false, fmt.Errorf("%w: bootstrap requires no journal artifacts", ErrJournalCorrupt)
	}
	initial := AnchorSnapshot{
		JournalID: journal.journalID, Namespace: RecoveryNamespaceV1, Epoch: journal.epoch,
		Committed: JournalHead{},
	}
	if _, _, err = journal.compareAndSwapAnchor(ctx, "", initial); err != nil {
		return false, err
	}
	final, finalRevision, err := journal.readWriterAnchor(ctx)
	if err != nil {
		return false, err
	}
	if err := journal.validateAnchorSnapshot(final, finalRevision); err != nil {
		return false, err
	}
	if !anchorSnapshotsEqual(final, initial) {
		return false, ErrAnchorIndeterminate
	}
	return true, nil
}

func (journal *FileRecoveryJournal) validateAnchorSnapshot(
	snapshot AnchorSnapshot,
	revision string,
) error {
	if revision == "" || snapshot.JournalID != journal.journalID ||
		snapshot.Namespace != RecoveryNamespaceV1 || snapshot.Epoch != journal.epoch {
		return ErrAnchorRollback
	}
	if !validJournalHead(snapshot.Committed) {
		return ErrAnchorRollback
	}
	if snapshot.Pending != nil {
		pending := snapshot.Pending
		if !validJournalHead(pending.OldHead) ||
			!validJournalHead(pending.NewHead) ||
			snapshot.Committed != pending.OldHead ||
			pending.NewHead.Sequence != pending.OldHead.Sequence+1 ||
			pending.OperationID != journalOperationID(journal.journalID, pending.NewHead) {
			return ErrAnchorRollback
		}
	}
	return nil
}

func validJournalHead(head JournalHead) bool {
	return (head.Sequence == 0 && head.Hash == "") ||
		(head.Sequence > 0 && sha256DigestPattern.MatchString(head.Hash))
}

func journalOperationID(journalID string, head JournalHead) string {
	return fmt.Sprintf("%s:%d:%s", journalID, head.Sequence, head.Hash)
}

func anchorSnapshotsEqual(left, right AnchorSnapshot) bool {
	if left.JournalID != right.JournalID ||
		left.Namespace != right.Namespace ||
		left.Epoch != right.Epoch ||
		left.Committed != right.Committed ||
		(left.Pending == nil) != (right.Pending == nil) {
		return false
	}
	if left.Pending == nil {
		return true
	}
	return *left.Pending == *right.Pending
}

func (journal *FileRecoveryJournal) compareAndSwapAnchor(ctx context.Context, expectedRevision string, next AnchorSnapshot) (AnchorSnapshot, string, error) {
	if e := journal.checkWriter(ctx); e != nil {
		return AnchorSnapshot{}, "", e
	}
	op, e := journal.writerOperation(ctx)
	if e != nil {
		return AnchorSnapshot{}, "", e
	}
	if op.revision != expectedRevision {
		return AnchorSnapshot{}, "", ErrAnchorStaleRevision
	}
	if e = journal.startWriterStep(ctx); e != nil {
		return AnchorSnapshot{}, "", e
	}
	a, token, writeErr := journal.anchor.CompareAndSwap(ctx, RecoveryNamespaceV1, expectedRevision, next)
	return journal.confirmWriterCAS(ctx, expectedRevision, next, a, token, writeErr)
}
func (journal *FileRecoveryJournal) confirmWriterCAS(ctx context.Context, old string, next, ack AnchorSnapshot, ackToken string, writeErr error) (AnchorSnapshot, string, error) {
	op, e := journal.writerOperation(ctx)
	if e != nil {
		return AnchorSnapshot{}, "", errors.Join(ErrRecoveryWriteUnknown, writeErr, e)
	}
	if writeErr == nil && (ackToken == old || journal.validateAnchorSnapshot(ack, ackToken) != nil || !anchorSnapshotsEqual(ack, next)) {
		return AnchorSnapshot{}, "", errors.Join(ErrRecoveryWriteUnknown, ErrAnchorIndeterminate)
	}
	// The operation still owns its original lock. Only this exact attempted CAS
	// may explain a successor; an arbitrary fresh revision cannot reset authority.
	a, token, readErr := journal.anchor.Load(ctx, RecoveryNamespaceV1)
	_, e = journal.writerOperation(ctx)
	if readErr != nil || e != nil {
		return AnchorSnapshot{}, "", errors.Join(ErrRecoveryWriteUnknown, ErrAnchorIndeterminate, writeErr, readErr, e)
	}
	if token == old || journal.validateAnchorSnapshot(a, token) != nil || !anchorSnapshotsEqual(a, next) || writeErr == nil && token != ackToken {
		return AnchorSnapshot{}, "", errors.Join(ErrRecoveryWriteUnknown, ErrAnchorIndeterminate, writeErr)
	}
	op.observed = true
	op.missing = false
	op.snapshot = copyWriterAnchor(a)
	op.revision = token
	return a, token, nil
}

func (journal *FileRecoveryJournal) writable(ctx context.Context) error {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if journal == nil || journal.readOnly {
		return ErrAdmissionClosed
	}
	return nil
}

func (journal *FileRecoveryJournal) Begin(ctx context.Context, runID, backupRef string, recoverySecret []byte) (JournalEntry, error) {
	if e := journal.writable(ctx); e != nil {
		return JournalEntry{}, e
	}
	if invalidText(runID) || invalidText(backupRef) || len(recoverySecret) == 0 {
		return JournalEntry{}, fmt.Errorf("%w: invalid begin input", ErrJournalMismatch)
	}
	mutated := false
	var result JournalEntry
	err := journal.withLock(ctx, func(ctx context.Context) error {
		records, err := journal.loadRecords(ctx, false)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.RunID == runID {
				return ErrRunAlreadyExists
			}
		}
		for _, key := range journal.keyring.retained() {
			candidate := fingerprintSecret(key, recoverySecret)
			for _, record := range records {
				if record.KeyID == key.ID && record.KeyVersion == key.Version &&
					hmac.Equal([]byte(candidate), []byte(record.SecretFingerprint)) {
					return ErrSecretReused
				}
			}
		}
		active := journal.keyring.activeKey()
		record := diskRecord{
			Namespace: RecoveryNamespaceV1, Sequence: uint64(len(records) + 1),
			RunID: runID, BackupRef: backupRef, KeyID: active.ID, KeyVersion: active.Version,
			SecretFingerprint: fingerprintSecret(active, recoverySecret), Stage: StageBegun,
			RecordedAt:         canonicalTime(journal.now()).Format(time.RFC3339Nano),
			PreviousRecordHash: previousRecordHash(records),
		}
		record.RecordHMAC = computeRecordHMAC(record, active.Material)
		mutated = true
		if err := journal.persistRecords(ctx, records, append(records, record)); err != nil {
			return err
		}
		result = record.toEntry()
		return nil
	})
	if mutated && err != nil {
		err = errors.Join(ErrRecoveryWriteUnknown, err)
	}
	return result, err
}

func (journal *FileRecoveryJournal) MarkConsumed(ctx context.Context, runID string) (JournalEntry, error) {
	if e := journal.writable(ctx); e != nil {
		return JournalEntry{}, e
	}
	if invalidText(runID) {
		return JournalEntry{}, fmt.Errorf("%w: invalid run id", ErrJournalMismatch)
	}
	mutated := false
	var result JournalEntry
	err := journal.withLock(ctx, func(ctx context.Context) error {
		records, err := journal.loadRecords(ctx, false)
		if err != nil {
			return err
		}
		begun, latest, err := recordsForRun(records, runID)
		if err != nil {
			return err
		}
		if latest.Stage != StageBegun {
			return fmt.Errorf("%w: run is not begun", ErrJournalMismatch)
		}
		key, ok := journal.keyring.lookup(begun.KeyID, begun.KeyVersion)
		if !ok {
			return fmt.Errorf("%w: referenced key unavailable", ErrJournalCorrupt)
		}
		recordedAt := canonicalTime(journal.now())
		begunAt, _ := parseCanonicalTime(begun.RecordedAt)
		if recordedAt.Before(begunAt) {
			recordedAt = begunAt
		}
		record := diskRecord{
			Namespace: RecoveryNamespaceV1, Sequence: uint64(len(records) + 1),
			RunID: begun.RunID, BackupRef: begun.BackupRef, KeyID: begun.KeyID,
			KeyVersion: begun.KeyVersion, SecretFingerprint: begun.SecretFingerprint,
			Stage: StageConsumed, RecordedAt: recordedAt.Format(time.RFC3339Nano),
			PreviousRecordHash: previousRecordHash(records),
		}
		record.RecordHMAC = computeRecordHMAC(record, key.Material)
		mutated = true
		if err := journal.persistRecords(ctx, records, append(records, record)); err != nil {
			return err
		}
		result = record.toEntry()
		return nil
	})
	if mutated && err != nil {
		err = errors.Join(ErrRecoveryWriteUnknown, err)
	}
	return result, err
}

func (journal *FileRecoveryJournal) Read(ctx context.Context, runID string) (JournalEntry, error) {
	return readerForJournal(journal).Read(ctx, runID)
}
func (journal *FileRecoveryJournal) VerifyAgainst(ctx context.Context, run RecoveryRun) (JournalEntry, error) {
	return readerForJournal(journal).VerifyAgainst(ctx, run)
}
func (journal *FileRecoveryJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (VerifiedJournalSnapshot, error) {
	return readerForJournal(journal).ReadVerifiedSnapshot(ctx, namespace)
}
func (journal *FileRecoveryJournal) VerifyBeginSecret(ctx context.Context, binding RunBinding, candidate []byte) (JournalEntry, error) {
	return readerForJournal(journal).VerifyBeginSecret(ctx, binding, candidate)
}

// ValidateKeyRetirement derives both static prerequisites from the authenticated
// journal before delegating opaque capability verification and consumption.
func (journal *FileRecoveryJournal) ValidateKeyRetirement(
	ctx context.Context,
	keyID string,
	keyVersion string,
	capability KeyRetirementCapability,
) error {
	key, err := journal.retirementKey(ctx, keyID, keyVersion)
	if err != nil {
		return err
	}
	return journal.withLock(ctx, func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		records, err := journal.loadRecords(ctx, false)
		if err != nil {
			return err
		}
		latestByRun := make(map[string]Stage)
		hasRetainedRecords := false
		for _, record := range records {
			if record.KeyID != key.ID || record.KeyVersion != key.Version {
				continue
			}
			hasRetainedRecords = true
			latestByRun[record.RunID] = record.Stage
		}
		for _, stage := range latestByRun {
			if stage == StageBegun {
				return ErrKeyRetirementBlocked
			}
		}
		if hasRetainedRecords {
			return ErrKeyRetirementBlocked
		}
		return journal.verifyAndConsumeKeyRetirementCapability(
			ctx,
			KeyRetirementRequest{KeyID: key.ID, KeyVersion: key.Version},
			capability,
		)
	})
}

func (journal *FileRecoveryJournal) retirementKey(ctx context.Context, keyID, keyVersion string) (Key, error) {
	if journal.readOnly {
		return Key{}, ErrAdmissionClosed
	}
	if ctx == nil || invalidKeyReferencePart(keyID) || invalidKeyReferencePart(keyVersion) {
		return Key{}, fmt.Errorf("%w: invalid retirement request", ErrInvalidKeyring)
	}
	if e := checkedContext(ctx); e != nil {
		return Key{}, e
	}
	key, ok := journal.keyring.lookup(keyID, keyVersion)
	if !ok {
		return Key{}, fmt.Errorf("%w: unknown key", ErrInvalidKeyring)
	}
	if key.State != KeyStateVerifyOnly {
		return Key{}, ErrKeyRetirementBlocked
	}
	return key, nil
}

func (journal *FileRecoveryJournal) verifyAndConsumeKeyRetirementCapability(
	ctx context.Context,
	request KeyRetirementRequest,
	capability KeyRetirementCapability,
) error {
	if journal.keyRetirementVerifier == nil {
		return errors.Join(
			ErrKeyRetirementBlocked,
			ErrKeyRetirementVerifierUnavailable,
		)
	}
	if e := journal.startWriterStep(ctx); e != nil {
		return e
	}
	decision, err := journal.keyRetirementVerifier.VerifyAndConsume(
		ctx,
		request,
		capability,
	)
	if e := journal.checkWriter(ctx); e != nil {
		return errors.Join(ErrRecoveryWriteUnknown, err, e)
	}
	if err != nil {
		return errors.Join(
			ErrKeyRetirementBlocked,
			ErrKeyRetirementVerifierUnavailable,
			err,
		)
	}
	switch decision {
	case KeyRetirementVerificationApproved:
		return nil
	case KeyRetirementVerificationRejected:
		return errors.Join(
			ErrKeyRetirementBlocked,
			ErrKeyRetirementCapabilityRejected,
		)
	default:
		return errors.Join(
			ErrKeyRetirementBlocked,
			ErrKeyRetirementCapabilityIndeterminate,
		)
	}
}

func validateRecoveryRun(run RecoveryRun) error {
	if invalidText(run.RunID) || invalidText(run.BackupRef) || invalidText(run.JournalKeyID) ||
		invalidText(run.JournalKeyVersion) || invalidText(run.JournalSecretFingerprint) ||
		!isCanonicalTime(run.StartedAt) || (run.JournalConsumedAt != nil && !isCanonicalTime(*run.JournalConsumedAt)) {
		return ErrInvalidRecoveryRun
	}
	return nil
}

// writerOperation is owned by one withLock invocation, never by a receiver.
// Its context is required by private mutating helpers and expires before release.
type writerOperationKey struct{}
type writerOperation struct {
	owner                     *FileRecoveryJournal
	lock                      *journalFileLock
	binding                   string
	closed                    atomic.Bool
	observed, missing, writes bool
	snapshot                  AnchorSnapshot
	revision                  string
}

func (journal *FileRecoveryJournal) writerOperation(ctx context.Context) (*writerOperation, error) {
	if e := checkedContext(ctx); e != nil {
		return nil, e
	}
	op, ok := ctx.Value(writerOperationKey{}).(*writerOperation)
	if !ok || op.owner != journal || op.closed.Load() {
		return nil, ErrAdmissionBinding
	}
	if e := op.lock.Check(ctx); e != nil {
		return nil, e
	}
	if op.binding != readerForJournal(journal).sourceBinding() {
		return nil, ErrAdmissionDrift
	}
	return op, checkedContext(ctx)
}
func (journal *FileRecoveryJournal) readWriterAnchor(ctx context.Context) (AnchorSnapshot, string, error) {
	op, e := journal.writerOperation(ctx)
	if e != nil {
		return AnchorSnapshot{}, "", e
	}
	a, token, readErr := journal.anchor.Load(ctx, RecoveryNamespaceV1)
	if _, e = journal.writerOperation(ctx); e != nil {
		return AnchorSnapshot{}, "", errors.Join(readErr, e)
	}
	missing := errors.Is(readErr, ErrAnchorMissing)
	if readErr != nil && !missing {
		return AnchorSnapshot{}, "", readErr
	}
	if !op.observed && !missing {
		if e = validateConsumptionAnchor(ctx, a, token); e != nil {
			return AnchorSnapshot{}, "", e
		}
	}
	if !missing {
		if e = journal.validateAnchorSnapshot(a, token); e != nil {
			return AnchorSnapshot{}, "", e
		}
	}
	if !op.observed {
		op.observed = true
		op.missing = missing
		op.snapshot = copyWriterAnchor(a)
		op.revision = token
	} else if missing != op.missing || !missing && (token != op.revision || !anchorSnapshotsEqual(a, op.snapshot)) {
		return AnchorSnapshot{}, "", errors.Join(ErrAdmissionDrift, ErrAnchorRollback)
	}
	if missing {
		return a, token, ErrAnchorMissing
	}
	return a, token, nil
}
func copyWriterAnchor(a AnchorSnapshot) AnchorSnapshot {
	if a.Pending != nil {
		p := *a.Pending
		a.Pending = &p
	}
	return a
}
func (journal *FileRecoveryJournal) checkWriter(ctx context.Context) error {
	_, _, e := journal.readWriterAnchor(ctx)
	if e == ErrAnchorMissing {
		return nil
	}
	return e
}
func (journal *FileRecoveryJournal) startWriterStep(ctx context.Context) error {
	if e := journal.checkWriter(ctx); e != nil {
		return e
	}
	op, e := journal.writerOperation(ctx)
	if e != nil {
		return e
	}
	op.writes = true
	return nil
}
func (journal *FileRecoveryJournal) withLock(ctx context.Context, operation func(context.Context) error) (result error) {
	if journal.acquireLock == nil {
		return fmt.Errorf("%w: lock provider is unavailable", ErrJournalCorrupt)
	}
	if e := checkedContext(ctx); e != nil {
		return e
	}
	lock, e := journal.acquireLock(ctx, journal.path, journal.fileMode)
	if e != nil {
		return e
	}
	if lock == nil {
		return ErrAdmissionIncomplete
	}
	op := &writerOperation{owner: journal, lock: lock}
	// Register before binding, checks or ports can panic. Natural unwinding keeps
	// the original panic; ownership expires before mandatory release.
	defer func() { op.closed.Store(true); result = errors.Join(result, lock.Close()) }()
	op.binding = readerForJournal(journal).sourceBinding()
	owned := context.WithValue(ctx, writerOperationKey{}, op)
	_, operationErr := journal.writerOperation(owned)
	if operationErr == nil {
		operationErr = operation(owned)
	}
	if op.observed {
		operationErr = errors.Join(operationErr, journal.checkWriter(owned))
	} else {
		_, e = journal.writerOperation(owned)
		operationErr = errors.Join(operationErr, e)
	}
	if operationErr != nil && op.writes {
		operationErr = errors.Join(ErrRecoveryWriteUnknown, operationErr)
	}
	return operationErr
}

func validateConsumptionAnchor(ctx context.Context, a AnchorSnapshot, revision string) error {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if sources, ok := ctx.Value(consumptionSourceKey{}).(*consumptionSources); ok {
		s := sources.recovery
		if a.Pending != nil || a.JournalID != s.JournalID || a.Namespace != s.Namespace || a.Epoch != s.Epoch || a.Committed != s.Head || revision != s.Revision {
			return ErrAdmissionDrift
		}
	}
	return nil
}

func (journal *FileRecoveryJournal) loadRecords(ctx context.Context, allowMissingBootstrap bool) ([]diskRecord, error) {
	snapshot, revision, err := journal.readWriterAnchor(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateConsumptionAnchor(ctx, snapshot, revision); err != nil {
		return nil, err
	}
	if err := journal.validateAnchorSnapshot(snapshot, revision); err != nil {
		return nil, err
	}
	artifacts, err := journal.tempArtifactPaths(ctx)
	if err != nil {
		return nil, err
	}
	records, exists, err := journal.readRecordsAt(ctx, journal.path, true)
	if err != nil {
		return nil, err
	}
	if snapshot.Pending == nil {
		if len(artifacts) != 0 {
			return nil, fmt.Errorf("%w: orphan journal artifact requires review", ErrJournalCorrupt)
		}
		if !exists {
			if (allowMissingBootstrap || journal.bootstrapPending) && snapshot.Committed == (JournalHead{}) {
				return nil, nil
			}
			return nil, ErrAnchorRollback
		}
		if headForRecords(records) != snapshot.Committed {
			return nil, ErrAnchorRollback
		}
		journal.bootstrapPending = false
		return records, checkedContext(ctx)
	}
	return journal.reconcilePendingRecords(ctx, snapshot, revision, records, exists, artifacts)
}

func (journal *FileRecoveryJournal) reconcilePendingRecords(ctx context.Context, snapshot AnchorSnapshot, revision string, records []diskRecord, exists bool, artifacts []string) ([]diskRecord, error) {
	if len(artifacts) > 1 {
		return nil, fmt.Errorf("%w: multiple journal artifacts", ErrJournalCorrupt)
	}
	if len(artifacts) == 1 {
		var err error
		records, err = journal.reconcileArtifact(ctx, snapshot, records, artifacts[0])
		if err != nil {
			return nil, err
		}
		exists = true
	}
	finalHead, err := journal.commitPendingRecovery(ctx, revision, snapshot, headForRecords(records), len(artifacts) == 0)
	if err != nil {
		return nil, err
	}
	if err := journal.verifyCommittedHead(ctx, finalHead, finalHead == (JournalHead{})); err != nil {
		return nil, err
	}
	if !exists && finalHead != (JournalHead{}) {
		return nil, ErrAnchorRollback
	}
	journal.bootstrapPending = finalHead == (JournalHead{})
	return records, checkedContext(ctx)
}

func (journal *FileRecoveryJournal) reconcileArtifact(ctx context.Context, snapshot AnchorSnapshot, records []diskRecord, path string) ([]diskRecord, error) {
	temp, _, err := journal.readRecordsAt(ctx, path, false)
	if err != nil {
		return nil, err
	}
	pending := snapshot.Pending
	if headForRecords(temp) != pending.NewHead {
		return nil, ErrAnchorRollback
	}
	if err := journal.startWriterStep(ctx); err != nil {
		return nil, err
	}
	switch headForRecords(records) {
	case pending.OldHead:
		if err := os.Rename(path, journal.path); err != nil {
			return nil, fmt.Errorf("%w: pending journal replace failed: %w", ErrJournalCorrupt, err)
		}
		records = temp
	case pending.NewHead:
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("%w: remove redundant journal artifact: %w", ErrJournalCorrupt, err)
		}
	default:
		return nil, ErrAnchorRollback
	}
	if err := journal.syncJournalDirectory(ctx); err != nil {
		return nil, errors.Join(ErrRecoveryWriteUnknown, ErrJournalCorrupt, err)
	}
	return records, journal.checkWriter(ctx)
}

func (journal *FileRecoveryJournal) commitPendingRecovery(
	ctx context.Context,
	revision string,
	snapshot AnchorSnapshot,
	head JournalHead,
	directorySyncRequired bool,
) (JournalHead, error) {
	pending := *snapshot.Pending
	var finalHead JournalHead
	switch head {
	case pending.NewHead:
		if directorySyncRequired {
			if e := journal.syncJournalDirectory(ctx); e != nil {
				return JournalHead{}, errors.Join(ErrRecoveryWriteUnknown, ErrJournalCorrupt, e)
			}
		}
		snapshot.Committed = pending.NewHead
		finalHead = pending.NewHead
	case pending.OldHead:
		finalHead = pending.OldHead
	default:
		return JournalHead{}, ErrAnchorRollback
	}
	snapshot.Pending = nil
	if _, _, err := journal.compareAndSwapAnchor(ctx, revision, snapshot); err != nil {
		return JournalHead{}, err
	}
	return finalHead, nil
}

func (journal *FileRecoveryJournal) verifyCommittedHead(
	ctx context.Context,
	expected JournalHead,
	allowMissingZero bool,
) error {
	snapshot, revision, err := journal.readWriterAnchor(ctx)
	if err != nil {
		return err
	}
	if err := journal.validateAnchorSnapshot(snapshot, revision); err != nil {
		return err
	}
	if snapshot.Pending != nil || snapshot.Committed != expected {
		return ErrAnchorIndeterminate
	}
	artifacts, err := journal.tempArtifactPaths(ctx)
	if err != nil {
		return err
	}
	if len(artifacts) != 0 {
		return ErrAnchorIndeterminate
	}
	records, exists, err := journal.readRecordsAt(ctx, journal.path, true)
	if err != nil {
		return err
	}
	if !exists {
		if allowMissingZero && expected == (JournalHead{}) {
			return nil
		}
		return ErrAnchorIndeterminate
	}
	if headForRecords(records) != expected {
		return ErrAnchorIndeterminate
	}
	return nil
}

func (journal *FileRecoveryJournal) readRecordsAt(ctx context.Context, path string, allowMissing bool) ([]diskRecord, bool, error) {
	if err := checkedContext(ctx); err != nil {
		return nil, false, err
	}
	file, err := openJournalReadFile(ctx, path)
	if errors.Is(err, os.ErrNotExist) && allowMissing {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	records, err := journal.readOpenedRecords(ctx, file)
	return records, true, errors.Join(err, file.Close())
}

func (journal *FileRecoveryJournal) readOpenedRecords(ctx context.Context, file *os.File) ([]diskRecord, error) {
	before, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if err := journal.validateFileInfo(before); err != nil {
		return nil, err
	}
	if journal.readMaxBytes <= 0 || journal.readMaxRecords <= 0 || before.Size() > journal.readMaxBytes {
		return nil, ErrAdmissionIncomplete
	}
	raw, err := io.ReadAll(io.LimitReader(file, journal.readMaxBytes))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	current, err := os.Lstat(file.Name())
	if err != nil {
		return nil, err
	}
	if !os.SameFile(after, current) {
		return nil, ErrAdmissionDrift
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || int64(len(raw)) != after.Size() {
		return nil, ErrAdmissionDrift
	}
	return journal.decodeReadRecords(ctx, raw)
}

func (journal *FileRecoveryJournal) decodeReadRecords(ctx context.Context, raw []byte) ([]diskRecord, error) {
	if len(raw) == 0 {
		return []diskRecord{}, checkedContext(ctx)
	}
	if raw[len(raw)-1] != '\n' {
		return nil, ErrJournalCorrupt
	}
	lines := bytes.Split(raw[:len(raw)-1], []byte{'\n'})
	if int64(len(lines)) > journal.readMaxRecords {
		return nil, ErrAdmissionIncomplete
	}
	records := make([]diskRecord, 0, len(lines))
	for _, line := range lines {
		if err := checkedContext(ctx); err != nil {
			return nil, err
		}
		record, err := decodeDiskRecord(line)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid record: %w", ErrJournalCorrupt, err)
		}
		records = append(records, record)
	}
	if err := journal.validateRecords(records); err != nil {
		return nil, err
	}
	return records, checkedContext(ctx)
}

func (journal *FileRecoveryJournal) tempArtifactPaths(ctx context.Context) ([]string, error) {
	if e := checkedContext(ctx); e != nil {
		return nil, e
	}
	entries, err := os.ReadDir(filepath.Dir(journal.path))
	if err != nil {
		return nil, fmt.Errorf("%w: inspect journal directory: %v", ErrJournalCorrupt, err)
	}
	prefix := "." + filepath.Base(journal.path) + ".tmp-"
	paths := make([]string, 0)
	for _, entry := range entries {
		if e := checkedContext(ctx); e != nil {
			return nil, e
		}
		if strings.HasPrefix(entry.Name(), prefix) {
			paths = append(paths, filepath.Join(filepath.Dir(journal.path), entry.Name()))
		}
	}
	return paths, nil
}

func (journal *FileRecoveryJournal) validateRecords(records []diskRecord) error {
	previousHash := ""
	runs := make(map[string]diskRecord)
	for index, record := range records {
		if record.Namespace != RecoveryNamespaceV1 || record.Sequence != uint64(index+1) ||
			invalidText(record.RunID) || invalidText(record.BackupRef) ||
			record.PreviousRecordHash != previousHash {
			return fmt.Errorf("%w: sequence, namespace, binding, or chain mismatch", ErrJournalCorrupt)
		}
		if _, err := parseCanonicalTime(record.RecordedAt); err != nil {
			return fmt.Errorf("%w: noncanonical time", ErrJournalCorrupt)
		}
		key, ok := journal.keyring.lookup(record.KeyID, record.KeyVersion)
		if !ok {
			return fmt.Errorf("%w: referenced key unavailable", ErrJournalCorrupt)
		}
		if !hmac.Equal([]byte(record.RecordHMAC), []byte(computeRecordHMAC(record, key.Material))) {
			return fmt.Errorf("%w: record authentication failed", ErrJournalCorrupt)
		}
		if previous, exists := runs[record.RunID]; !exists {
			if record.Stage != StageBegun {
				return fmt.Errorf("%w: consumed without begin", ErrJournalCorrupt)
			}
			runs[record.RunID] = record
		} else {
			if previous.Stage != StageBegun || record.Stage != StageConsumed ||
				previous.BackupRef != record.BackupRef || previous.KeyID != record.KeyID ||
				previous.KeyVersion != record.KeyVersion ||
				previous.SecretFingerprint != record.SecretFingerprint {
				return fmt.Errorf("%w: consumed binding mismatch", ErrJournalCorrupt)
			}
			runs[record.RunID] = record
		}
		canonical, _ := encodeDiskRecord(record)
		previousHash = SHA256Digest(canonical)
	}
	return nil
}

func recordsForRun(records []diskRecord, runID string) (diskRecord, diskRecord, error) {
	var begun, latest diskRecord
	found := false
	for _, record := range records {
		if record.RunID != runID {
			continue
		}
		if !found {
			begun = record
			found = true
		}
		latest = record
	}
	if !found {
		return diskRecord{}, diskRecord{}, ErrRunNotFound
	}
	return begun, latest, nil
}

func (journal *FileRecoveryJournal) persistRecords(ctx context.Context, oldRecords, newRecords []diskRecord) error {
	if err := checkedContext(ctx); err != nil {
		return err
	}
	snapshot, revision, err := journal.readWriterAnchor(ctx)
	if err != nil {
		return err
	}
	if err := validateConsumptionAnchor(ctx, snapshot, revision); err != nil {
		return err
	}
	if err := journal.validateAnchorSnapshot(snapshot, revision); err != nil {
		return err
	}
	if snapshot.Pending != nil || snapshot.Committed != headForRecords(oldRecords) {
		return ErrAnchorRollback
	}
	oldHead, newHead := snapshot.Committed, headForRecords(newRecords)
	if newHead.Sequence != oldHead.Sequence+1 {
		return ErrAnchorRollback
	}
	tempPath, err := journal.stageRecords(ctx, newRecords)
	if err != nil {
		return err
	}
	snapshot.Pending = &AnchorPending{
		OldHead: oldHead, NewHead: newHead,
		OperationID: journalOperationID(journal.journalID, newHead),
	}
	pending, pendingRevision, err := journal.compareAndSwapAnchor(ctx, revision, snapshot)
	if err != nil {
		return err
	}
	if err := journal.installStagedRecords(ctx, tempPath); err != nil {
		return errors.Join(ErrRecoveryWriteUnknown, err)
	}
	pending.Committed, pending.Pending = newHead, nil
	if _, _, err = journal.compareAndSwapAnchor(ctx, pendingRevision, pending); err != nil {
		return err
	}
	if err := journal.verifyCommittedHead(ctx, newHead, false); err != nil {
		return err
	}
	journal.bootstrapPending = false
	return nil
}

func (journal *FileRecoveryJournal) stageRecords(ctx context.Context, newRecords []diskRecord) (path string, err error) {
	if err = journal.checkWriter(ctx); err != nil {
		return "", err
	}
	raw, err := encodeDiskRecords(newRecords)
	if err != nil {
		return "", fmt.Errorf("%w: encode failed: %w", ErrJournalCorrupt, err)
	}
	if int64(len(raw)) > journal.readMaxBytes || int64(len(newRecords)) > journal.readMaxRecords {
		return "", ErrAdmissionIncomplete
	}
	if err = journal.checkWriter(ctx); err != nil {
		return "", err
	}
	if err = journal.startWriterStep(ctx); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(filepath.Dir(journal.path), "."+filepath.Base(journal.path)+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("%w: create temporary file: %w", ErrJournalCorrupt, err)
	}
	path = temp.Name()
	// Keep uncertain staging evidence. A started syscall always returns before
	// close; cancellation prevents the next persistent action, never detaches it.
	defer func() {
		err = errors.Join(err, temp.Close(), journal.checkWriter(ctx))
		if err != nil {
			err = errors.Join(ErrRecoveryWriteUnknown, err)
		}
	}()
	if err = journal.checkWriter(ctx); err != nil {
		return path, err
	}
	if err = temp.Chmod(journal.fileMode); err != nil {
		return path, err
	}
	if err = journal.checkWriter(ctx); err != nil {
		return path, err
	}
	if _, err = temp.Write(raw); err != nil {
		return path, err
	}
	if err = journal.checkWriter(ctx); err != nil {
		return path, err
	}
	if err = temp.Sync(); err != nil {
		return path, err
	}
	return path, journal.checkWriter(ctx)
}

func (journal *FileRecoveryJournal) syncJournalDirectory(ctx context.Context) error {
	if e := journal.startWriterStep(ctx); e != nil {
		return e
	}
	if journal.syncDirectory == nil {
		return fmt.Errorf("%w: directory sync provider is unavailable", ErrJournalCorrupt)
	}
	e := journal.syncDirectory(filepath.Dir(journal.path))
	return errors.Join(e, journal.checkWriter(ctx))
}
func (journal *FileRecoveryJournal) installStagedRecords(ctx context.Context, path string) error {
	if e := journal.startWriterStep(ctx); e != nil {
		return e
	}
	if e := os.Rename(path, journal.path); e != nil {
		return errors.Join(ErrRecoveryWriteUnknown, ErrJournalCorrupt, e, journal.checkWriter(ctx))
	}
	return journal.syncJournalDirectory(ctx)
}

func encodeDiskRecords(records []diskRecord) ([]byte, error) {
	var buffer bytes.Buffer
	for _, record := range records {
		line, err := encodeDiskRecord(record)
		if err != nil {
			return nil, err
		}
		buffer.Write(line)
		buffer.WriteByte('\n')
	}
	return buffer.Bytes(), nil
}

func previousRecordHash(records []diskRecord) string {
	if len(records) == 0 {
		return ""
	}
	raw, _ := encodeDiskRecord(records[len(records)-1])
	return SHA256Digest(raw)
}

func headForRecords(records []diskRecord) JournalHead {
	if len(records) == 0 {
		return JournalHead{}
	}
	return JournalHead{Sequence: uint64(len(records)), Hash: previousRecordHash(records)}
}

func syncDirectoryPath(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

type diskRecord struct {
	Namespace          string `json:"namespace"`
	Sequence           uint64 `json:"sequence"`
	RunID              string `json:"run_id"`
	BackupRef          string `json:"backup_ref"`
	KeyID              string `json:"key_id"`
	KeyVersion         string `json:"key_version"`
	SecretFingerprint  string `json:"secret_fingerprint"`
	Stage              Stage  `json:"stage"`
	RecordedAt         string `json:"recorded_at"`
	PreviousRecordHash string `json:"previous_record_hash"`
	RecordHMAC         string `json:"record_hmac"`
}

type diskRecordMACPayload struct {
	Namespace          string `json:"namespace"`
	Sequence           uint64 `json:"sequence"`
	RunID              string `json:"run_id"`
	BackupRef          string `json:"backup_ref"`
	KeyID              string `json:"key_id"`
	KeyVersion         string `json:"key_version"`
	SecretFingerprint  string `json:"secret_fingerprint"`
	Stage              Stage  `json:"stage"`
	RecordedAt         string `json:"recorded_at"`
	PreviousRecordHash string `json:"previous_record_hash"`
}

func computeRecordHMAC(record diskRecord, key []byte) string {
	raw, _ := json.Marshal(record.macPayload())
	return hmacDigest(key, raw)
}

func (record diskRecord) macPayload() diskRecordMACPayload {
	return diskRecordMACPayload{
		Namespace: record.Namespace, Sequence: record.Sequence, RunID: record.RunID,
		BackupRef: record.BackupRef, KeyID: record.KeyID, KeyVersion: record.KeyVersion,
		SecretFingerprint: record.SecretFingerprint, Stage: record.Stage,
		RecordedAt: record.RecordedAt, PreviousRecordHash: record.PreviousRecordHash,
	}
}

func encodeDiskRecord(record diskRecord) ([]byte, error) { return json.Marshal(record) }

func decodeDiskRecord(raw []byte) (diskRecord, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var record diskRecord
	if err := decoder.Decode(&record); err != nil {
		return diskRecord{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return diskRecord{}, errors.New("trailing record data")
	}
	return record, nil
}

func (record diskRecord) toEntry() JournalEntry {
	recordedAt, _ := parseCanonicalTime(record.RecordedAt)
	return JournalEntry{
		Namespace: record.Namespace, Sequence: record.Sequence, RunID: record.RunID,
		BackupRef: record.BackupRef, KeyID: record.KeyID, KeyVersion: record.KeyVersion,
		SecretFingerprint: record.SecretFingerprint, Stage: record.Stage, RecordedAt: recordedAt,
		PreviousRecordHash: record.PreviousRecordHash, RecordHMAC: record.RecordHMAC,
	}
}

func canonicalTime(value time.Time) time.Time { return value.UTC().Truncate(time.Microsecond) }

func isCanonicalTime(value time.Time) bool {
	return value.Location() == time.UTC && value.Nanosecond()%1000 == 0
}

func parseCanonicalTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || !strings.HasSuffix(value, "Z") || !isCanonicalTime(parsed) ||
		parsed.Format(time.RFC3339Nano) != value {
		return time.Time{}, errors.New("noncanonical UTC microsecond time")
	}
	return parsed, nil
}

func invalidText(value string) bool { return value == "" || strings.TrimSpace(value) != value }

func (lock *journalFileLock) Check(ctx context.Context) error {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if lock == nil || lock.check == nil {
		return ErrAdmissionClosed
	}
	return lock.check(ctx)
}
func (lock *journalFileLock) Close() error {
	if lock == nil {
		return nil
	}
	var unlockErr, closeErr error
	if lock.unlock != nil {
		unlockErr = lock.unlock()
		lock.unlock = nil
	}
	if lock.close != nil {
		closeErr = lock.close()
		lock.close = nil
	}
	lock.check = nil
	return errors.Join(unlockErr, closeErr)
}

package recovery

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"strings"
)

// ReadOptions binds a pure reader to installed media and explicit budgets.
type ReadOptions struct {
	Path, JournalID      string
	FileMode             fs.FileMode
	Epoch                uint64
	Anchor               JournalHeadAnchor
	Keyring              Keyring
	MaxBytes, MaxRecords int64
}

// verifiedRecoveryJournal only exposes committed reads.
type verifiedRecoveryJournal struct {
	journal              *FileRecoveryJournal
	lock                 func(context.Context) (ExclusiveLease, error)
	records              func(context.Context) ([]diskRecord, error)
	artifacts            func(context.Context) ([]string, error)
	maxBytes, maxRecords int64
}

// OpenVerifiedRecoveryJournal opens installed media for committed reads only.
// The returned contextual Journal rejects every mutating method.
func OpenVerifiedRecoveryJournal(ctx context.Context, options ReadOptions) (*FileRecoveryJournal, error) {
	if e := checkedContext(ctx); e != nil {
		return nil, e
	}
	if !fileJournalSupported() {
		return nil, errors.Join(ErrAdmissionUnsupported, ErrUnsupportedPlatform)
	}
	if options.MaxBytes <= 0 || options.MaxBytes > int64(math.MaxInt) || options.MaxRecords <= 0 {
		return nil, ErrAdmissionBinding
	}
	keyring, e := NewKeyring(options.Keyring.retained())
	if e != nil {
		return nil, errors.Join(ErrAdmissionIncomplete, e)
	}
	j := &FileRecoveryJournal{readOnly: true, path: options.Path, fileMode: options.FileMode, journalID: options.JournalID,
		epoch: options.Epoch, anchor: options.Anchor, keyring: keyring, readMaxBytes: options.MaxBytes, readMaxRecords: options.MaxRecords}
	if !filepath.IsAbs(j.path) || filepath.Clean(j.path) != j.path || invalidText(j.path) || !admissionText(j.journalID) || j.epoch == 0 || missingRuntimePort(j.anchor) || j.fileMode.Perm() == 0 || j.fileMode.Perm()&0o077 != 0 {
		return nil, ErrAdmissionBinding
	}
	r := readerForJournal(j)
	if _, e := r.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1); e != nil {
		return nil, e
	}
	return j, nil
}

func readerForJournal(j *FileRecoveryJournal) *verifiedRecoveryJournal {
	return &verifiedRecoveryJournal{journal: j, maxBytes: j.readMaxBytes, maxRecords: j.readMaxRecords,
		lock: func(ctx context.Context) (ExclusiveLease, error) {
			return acquireJournalReadLock(ctx, j.path, j.fileMode)
		},
		records: func(ctx context.Context) ([]diskRecord, error) {
			x, _, e := j.readRecordsAt(ctx, j.path, false)
			return x, e
		},
		artifacts: func(ctx context.Context) ([]string, error) {
			if e := checkedContext(ctx); e != nil {
				return nil, e
			}
			return j.tempArtifactPaths(ctx)
		}}
}

func (reader *verifiedRecoveryJournal) withLock(ctx context.Context, fn func(ExclusiveLease) error) (result error) {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if reader == nil || reader.journal == nil || reader.lock == nil || reader.records == nil || reader.artifacts == nil || missingRuntimePort(reader.journal.anchor) || reader.maxBytes <= 0 || reader.maxRecords <= 0 {
		return ErrAdmissionBinding
	}
	l, e := reader.lock(ctx)
	if e != nil {
		if errors.Is(e, ErrUnsupportedPlatform) {
			return errors.Join(ErrAdmissionUnsupported, e)
		}
		return errors.Join(ErrAdmissionBusy, e)
	}
	if missingRuntimePort(l) {
		return ErrAdmissionIncomplete
	}
	// This invocation owns the acquired lease even while a port panics.
	defer func() { result = errors.Join(result, l.Close()) }()
	return fn(l)
}

func (reader *verifiedRecoveryJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (s VerifiedJournalSnapshot, err error) {
	if e := checkedContext(ctx); e != nil {
		return s, e
	}
	if namespace != RecoveryNamespaceV1 {
		return s, ErrAdmissionUnsupported
	}
	err = reader.withLock(ctx, func(l ExclusiveLease) error { var e error; s, e = reader.snapshot(ctx, l); return e })
	if err != nil {
		return VerifiedJournalSnapshot{}, err
	}
	return s, nil
}

func (reader *verifiedRecoveryJournal) sourceBinding() string {
	j := reader.journal
	raw, _ := json.Marshal(struct {
		Path, ID, Anchor string
		Epoch            uint64
		Mode             fs.FileMode
		Keys             []Key
	}{j.path, j.journalID, fmt.Sprintf("%T/%p", j.anchor, j.anchor), j.epoch, j.fileMode, j.keyring.retained()})
	defer clear(raw)
	return SHA256Digest(raw)
}

func (reader *verifiedRecoveryJournal) snapshot(ctx context.Context, l ExclusiveLease) (VerifiedJournalSnapshot, error) {
	if e := checkExclusive(ctx, l); e != nil {
		return VerifiedJournalSnapshot{}, e
	}
	before := reader.sourceBinding()
	j := reader.journal
	a, revision, e := j.anchor.Load(ctx, RecoveryNamespaceV1)
	if e != nil {
		return VerifiedJournalSnapshot{}, errors.Join(ErrAdmissionIncomplete, e)
	}
	if e := j.validateAnchorSnapshot(a, revision); e != nil {
		return VerifiedJournalSnapshot{}, errors.Join(ErrAdmissionBinding, e)
	}
	if a.Pending != nil {
		return VerifiedJournalSnapshot{}, ErrAdmissionIncomplete
	}
	if e := reader.noArtifacts(ctx); e != nil {
		return VerifiedJournalSnapshot{}, e
	}
	records, e := reader.records(ctx)
	if e != nil {
		return VerifiedJournalSnapshot{}, errors.Join(ErrAdmissionIncomplete, e)
	}
	if e := reader.validateReadRecords(ctx, records); e != nil {
		return VerifiedJournalSnapshot{}, e
	}
	s := VerifiedJournalSnapshot{JournalID: a.JournalID, Namespace: a.Namespace, Epoch: a.Epoch, Head: headForRecords(records), Revision: revision, Entries: []JournalEntry{}, Prefixes: []JournalHead{{}}}
	if s.Head != a.Committed {
		return VerifiedJournalSnapshot{}, ErrAdmissionDrift
	}
	for _, record := range records {
		s.Entries = append(s.Entries, record.toEntry())
		raw, _ := encodeDiskRecord(record)
		s.Prefixes = append(s.Prefixes, JournalHead{Sequence: record.Sequence, Hash: SHA256Digest(raw)})
	}
	after, token, e := j.anchor.Load(ctx, RecoveryNamespaceV1)
	if e != nil {
		return VerifiedJournalSnapshot{}, errors.Join(ErrAdmissionIncomplete, e)
	}
	if token != revision || !anchorSnapshotsEqual(a, after) || before != reader.sourceBinding() {
		return VerifiedJournalSnapshot{}, ErrAdmissionDrift
	}
	if e := reader.noArtifacts(ctx); e != nil {
		return VerifiedJournalSnapshot{}, e
	}
	return s, checkExclusive(ctx, l)
}

func (reader *verifiedRecoveryJournal) noArtifacts(ctx context.Context) error {
	x, e := reader.artifacts(ctx)
	if e != nil {
		return errors.Join(ErrAdmissionIncomplete, e)
	}
	if len(x) != 0 {
		return ErrAdmissionIncomplete
	}
	return checkedContext(ctx)
}

func (reader *verifiedRecoveryJournal) validateReadRecords(ctx context.Context, records []diskRecord) error {
	if int64(len(records)) > reader.maxRecords {
		return ErrAdmissionIncomplete
	}
	raw, e := encodeDiskRecords(records)
	if e != nil || int64(len(raw)) > reader.maxBytes {
		return errors.Join(ErrAdmissionIncomplete, e)
	}
	for _, record := range records {
		for _, value := range []string{record.RunID, record.BackupRef, record.KeyID, record.KeyVersion} {
			if !admissionText(value) {
				return ErrAdmissionBinding
			}
		}
		if !journalHMACDigestPattern.MatchString(record.SecretFingerprint) || !journalHMACDigestPattern.MatchString(record.RecordHMAC) {
			return ErrAdmissionIncomplete
		}
		if e := checkedContext(ctx); e != nil {
			return e
		}
		if record.Namespace != RecoveryNamespaceV1 {
			if strings.HasPrefix(record.Namespace, "recovery/") {
				return ErrAdmissionUnsupported
			}
			return ErrAdmissionBinding
		}
		if record.Stage != StageBegun && record.Stage != StageConsumed {
			return ErrAdmissionUnsupported
		}
		if _, ok := reader.journal.keyring.lookup(record.KeyID, record.KeyVersion); !ok {
			return ErrAdmissionIncomplete
		}
	}
	if e := reader.journal.validateRecords(records); e != nil {
		return errors.Join(ErrAdmissionIncomplete, e)
	}
	return checkedContext(ctx)
}

func (reader *verifiedRecoveryJournal) VerifyBeginSecret(ctx context.Context, binding RunBinding, candidate []byte) (entry JournalEntry, err error) {
	if e := checkedContext(ctx); e != nil {
		return entry, e
	}
	if !admissionText(binding.RunID) || !admissionText(binding.BackupRef) || len(candidate) == 0 {
		return entry, ErrAdmissionBinding
	}
	err = reader.withLock(ctx, func(l ExclusiveLease) error {
		before := reader.sourceBinding()
		s, e := reader.snapshot(ctx, l)
		if e != nil {
			return e
		}
		b, _, e := snapshotRun(s, binding.RunID)
		if e != nil {
			return errors.Join(ErrAdmissionBinding, e)
		}
		if b.BackupRef != binding.BackupRef {
			return ErrAdmissionBinding
		}
		key, ok := reader.journal.keyring.lookup(b.KeyID, b.KeyVersion)
		if !ok {
			return ErrAdmissionIncomplete
		}
		defer clear(key.Material)
		if !hmac.Equal([]byte(fingerprintSecret(key, candidate)), []byte(b.SecretFingerprint)) {
			return ErrAdmissionBinding
		}
		after, e := reader.snapshot(ctx, l)
		if e != nil {
			return e
		}
		if !sameControlledJournal(s, after) || before != reader.sourceBinding() {
			return ErrAdmissionDrift
		}
		if e := checkExclusive(ctx, l); e != nil {
			return e
		}
		entry = b
		return nil
	})
	if err != nil {
		return JournalEntry{}, err
	}
	return entry, nil
}

func snapshotRun(s VerifiedJournalSnapshot, run string) (JournalEntry, JournalEntry, error) {
	var begin, latest JournalEntry
	for _, e := range s.Entries {
		if e.RunID == run {
			if begin.RunID == "" {
				begin = e
			}
			latest = e
		}
	}
	if begin.RunID == "" {
		return begin, latest, ErrRunNotFound
	}
	return begin, latest, nil
}

func (reader *verifiedRecoveryJournal) Read(ctx context.Context, run string) (JournalEntry, error) {
	if e := checkedContext(ctx); e != nil {
		return JournalEntry{}, e
	}
	if !admissionText(run) {
		return JournalEntry{}, ErrAdmissionBinding
	}
	s, e := reader.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil {
		return JournalEntry{}, e
	}
	_, latest, e := snapshotRun(s, run)
	return latest, e
}

func (reader *verifiedRecoveryJournal) VerifyAgainst(ctx context.Context, run RecoveryRun) (JournalEntry, error) {
	if e := checkedContext(ctx); e != nil {
		return JournalEntry{}, e
	}
	if e := validateRecoveryRun(run); e != nil {
		return JournalEntry{}, e
	}
	s, e := reader.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil {
		return JournalEntry{}, e
	}
	b, latest, e := snapshotRun(s, run.RunID)
	if e != nil {
		return JournalEntry{}, e
	}
	if run.BackupRef != b.BackupRef || run.JournalKeyID != b.KeyID || run.JournalKeyVersion != b.KeyVersion || run.JournalSecretFingerprint != b.SecretFingerprint || !run.StartedAt.Equal(b.RecordedAt) || run.JournalStage != latest.Stage {
		return JournalEntry{}, ErrJournalMismatch
	}
	if latest.Stage == StageBegun && run.JournalConsumedAt != nil {
		return JournalEntry{}, ErrJournalMismatch
	}
	if latest.Stage == StageConsumed && (run.JournalConsumedAt == nil || !run.JournalConsumedAt.Equal(latest.RecordedAt)) {
		return JournalEntry{}, ErrJournalMismatch
	}
	return latest, nil
}

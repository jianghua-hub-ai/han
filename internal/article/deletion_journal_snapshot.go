package article

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"han/internal/recovery"
)

func (j *anchoredArticleJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (snapshot recovery.VerifiedJournalSnapshot, result error) {
	if namespace != ArticleDeletionNamespace {
		return snapshot, recovery.ErrAdmissionBinding
	}
	// Admission itself consumes this port before it can issue any WorkLease.
	result = j.withLock(ctx, false, func() error {
		var err error
		snapshot, _, err = j.snapshot(ctx)
		return err
	})
	if result != nil {
		return recovery.VerifiedJournalSnapshot{}, result
	}
	return snapshot, nil
}

func filepathForDeletionAnchor(j *anchoredArticleJournal, name string) string {
	return filepath.Join(j.options.Directory, name)
}

func (j *anchoredArticleJournal) ReadAllVerified(ctx context.Context) (records []ArticleDeletionTombstone, result error) {
	result = j.withLock(ctx, false, func() error {
		var err error
		_, records, err = j.snapshot(ctx)
		return err
	})
	if result != nil {
		return nil, result
	}
	return records, nil
}

func (j *anchoredArticleJournal) FindByOperation(ctx context.Context, operation string) (ArticleDeletionTombstone, error) {
	if !validUUID(operation) {
		return ArticleDeletionTombstone{}, ErrDeletionTombstoneNotFound
	}
	records, err := j.ReadAllVerified(ctx)
	if err != nil {
		return ArticleDeletionTombstone{}, err
	}
	for _, record := range records {
		if record.DeletionOperationID == operation {
			return record, nil
		}
	}
	return ArticleDeletionTombstone{}, ErrDeletionTombstoneNotFound
}

func (j *anchoredArticleJournal) snapshot(ctx context.Context) (recovery.VerifiedJournalSnapshot, []ArticleDeletionTombstone, error) {
	before, revision, err := j.loadCommitted(ctx)
	if err != nil {
		return recovery.VerifiedJournalSnapshot{}, nil, err
	}
	records, err := j.readRecords(ctx)
	if err != nil {
		return recovery.VerifiedJournalSnapshot{}, nil, err
	}
	head := deletionAnchorHead(records)
	if head != before.Committed {
		return recovery.VerifiedJournalSnapshot{}, nil, recovery.ErrAnchorRollback
	}
	after, token, err := j.loadCommitted(ctx)
	if err != nil || revision != token || !sameDeletionAnchor(before, after) {
		return recovery.VerifiedJournalSnapshot{}, nil, errors.Join(recovery.ErrAdmissionDrift, err)
	}
	s := recovery.VerifiedJournalSnapshot{JournalID: before.JournalID, Namespace: before.Namespace, Epoch: before.Epoch,
		Head: head, Revision: revision, Entries: []recovery.JournalEntry{}, Prefixes: []recovery.JournalHead{{}}}
	for _, record := range records {
		s.Prefixes = append(s.Prefixes, recovery.JournalHead{Sequence: uint64(record.Sequence), Hash: deletionRecordHash(record)})
	}
	return s, records, nil
}

func sameDeletionAnchor(a, b recovery.AnchorSnapshot) bool {
	if a.JournalID != b.JournalID || a.Namespace != b.Namespace || a.Epoch != b.Epoch || a.Committed != b.Committed || (a.Pending == nil) != (b.Pending == nil) {
		return false
	}
	return a.Pending == nil || *a.Pending == *b.Pending
}

func (j *anchoredArticleJournal) loadCommitted(ctx context.Context) (recovery.AnchorSnapshot, string, error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	a, revision, err := j.options.Anchor.Load(ctx, ArticleDeletionNamespace)
	if err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	if a.JournalID != j.options.JournalID || a.Namespace != ArticleDeletionNamespace || a.Epoch != j.options.Epoch || strings.TrimSpace(revision) != revision || revision == "" {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionBinding
	}
	if a.Pending != nil || !validDeletionAnchorHead(a.Committed) {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionIncomplete
	}
	return a, revision, deletionAnchorContext(ctx)
}

func validDeletionAnchorHead(h recovery.JournalHead) bool {
	if h.Sequence == 0 {
		return h.Hash == ""
	}
	if h.Sequence > math.MaxInt64 || !strings.HasPrefix(h.Hash, "sha256:") || len(h.Hash) != len("sha256:")+64 || strings.ToLower(h.Hash) != h.Hash {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(h.Hash, "sha256:"))
	return err == nil
}

func deletionAnchorHead(records []ArticleDeletionTombstone) recovery.JournalHead {
	if len(records) == 0 {
		return recovery.JournalHead{}
	}
	last := records[len(records)-1]
	return recovery.JournalHead{Sequence: uint64(last.Sequence), Hash: deletionRecordHash(last)}
}

func (j *anchoredArticleJournal) readRecords(ctx context.Context) ([]ArticleDeletionTombstone, error) {
	names, err := j.media.names(ctx)
	if err != nil || int64(len(names)) > j.options.ReadMaxRecords {
		return nil, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	sort.Strings(names)
	records, err := j.readNamedRecords(ctx, names)
	if err != nil {
		return nil, err
	}
	after, err := j.media.names(ctx)
	sort.Strings(after)
	if err != nil || strings.Join(names, "\x00") != strings.Join(after, "\x00") {
		return nil, errors.Join(recovery.ErrAdmissionDrift, err)
	}
	return records, deletionAnchorContext(ctx)
}

func (j *anchoredArticleJournal) readNamedRecords(ctx context.Context, names []string) ([]ArticleDeletionTombstone, error) {
	if int64(len(names)) > j.options.ReadMaxRecords {
		return nil, recovery.ErrAdmissionIncomplete
	}
	records := make([]ArticleDeletionTombstone, 0, len(names))
	operations := make(map[string]bool, len(names))
	remaining := j.options.ReadMaxBytes
	for _, name := range names {
		if name != deletionRecordName(int64(len(records)+1)) {
			return nil, ErrDeletionJournalCorrupt
		}
		raw, err := j.readRecordBytes(ctx, name, remaining)
		if err != nil {
			return nil, err
		}
		remaining -= int64(len(raw))
		record, err := j.decodeAnchorRecord(raw, records)
		if err != nil || operations[record.DeletionOperationID] {
			return nil, errors.Join(ErrDeletionJournalCorrupt, err)
		}
		operations[record.DeletionOperationID] = true
		records = append(records, record)
	}
	return records, deletionAnchorContext(ctx)
}

type deletionCoordinationInventory struct {
	stage, target string
	candidate     []byte
	published     bool
}

func (j *anchoredArticleJournal) coordinationInventory(ctx context.Context) (deletionCoordinationInventory, error) {
	p := j.control.current.Pending
	parts := strings.Split(p.OperationID, ":")
	expected := ".article-deletion.stage-" + parts[1]
	names, err := j.media.names(ctx)
	if err != nil || int64(len(names)) > j.options.ReadMaxRecords+1 {
		return deletionCoordinationInventory{}, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	sort.Strings(names)
	v := deletionCoordinationInventory{target: deletionRecordName(int64(p.NewHead.Sequence))}
	final := make([]string, 0, len(names))
	for _, name := range names {
		if name == expected {
			v.stage = name
		} else {
			final = append(final, name)
		}
	}
	records, err := j.readNamedRecords(ctx, final)
	if err != nil {
		return v, err
	}
	head := deletionAnchorHead(records)
	if head != p.OldHead && head != p.NewHead {
		return v, recovery.ErrAnchorRollback
	}
	v.published = head == p.NewHead
	v, err = j.readCoordinationCandidate(ctx, v, records)
	if err != nil {
		return v, err
	}
	if err := j.validateCoordinationCandidate(v, records, parts[2]); err != nil {
		return v, err
	}
	after, err := j.media.names(ctx)
	sort.Strings(after)
	if err != nil || strings.Join(names, "\x00") != strings.Join(after, "\x00") {
		return v, errors.Join(recovery.ErrAdmissionDrift, err)
	}
	return v, deletionAnchorContext(ctx)
}

func (j *anchoredArticleJournal) readCoordinationCandidate(ctx context.Context, v deletionCoordinationInventory, records []ArticleDeletionTombstone) (deletionCoordinationInventory, error) {
	var err error
	if v.published {
		if deletionAnchorHead(records[:len(records)-1]) != j.control.current.Pending.OldHead {
			return v, recovery.ErrAdmissionDrift
		}
		v.candidate, err = j.readRecordBytes(ctx, v.target, j.options.ReadMaxBytes)
		if err != nil {
			return v, err
		}
	}
	if v.stage != "" {
		raw, err := j.readRecordBytes(ctx, v.stage, j.options.ReadMaxBytes)
		if err != nil {
			return v, err
		}
		if v.published && (!bytes.Equal(v.candidate, raw) || !j.sameCoordinationAlias(v.stage, v.target)) {
			return v, ErrDeletionJournalCorrupt
		}
		v.candidate = raw
	}
	return v, deletionAnchorContext(ctx)
}

func (j *anchoredArticleJournal) sameCoordinationAlias(stage, target string) bool {
	a, err := os.Lstat(filepathForDeletionAnchor(j, stage))
	if err != nil || !a.Mode().IsRegular() {
		return false
	}
	b, err := os.Lstat(filepathForDeletionAnchor(j, target))
	return err == nil && b.Mode().IsRegular() && os.SameFile(a, b)
}

func (j *anchoredArticleJournal) validateCoordinationCandidate(v deletionCoordinationInventory, records []ArticleDeletionTombstone, digest string) error {
	if v.candidate == nil {
		return nil
	}
	if recovery.SHA256Digest(v.candidate) != "sha256:"+digest {
		return ErrDeletionJournalCorrupt
	}
	if v.published {
		records = records[:len(records)-1]
	}
	remaining := j.options.ReadMaxBytes
	operations := make(map[string]bool, len(records))
	for _, r := range records {
		raw, err := json.Marshal(deletionDiskRecordFromModel(r))
		if err != nil {
			return err
		}
		remaining -= int64(len(raw))
		operations[r.DeletionOperationID] = true
	}
	if int64(len(v.candidate)) > remaining || int64(len(records)) >= j.options.ReadMaxRecords {
		return recovery.ErrAdmissionIncomplete
	}
	r, err := j.decodeAnchorRecord(v.candidate, records)
	if err != nil || operations[r.DeletionOperationID] {
		return errors.Join(ErrDeletionJournalCorrupt, err)
	}
	if (recovery.JournalHead{Sequence: uint64(r.Sequence), Hash: deletionRecordHash(r)}) != j.control.current.Pending.NewHead {
		return recovery.ErrAdmissionDrift
	}
	return nil
}

func (j *anchoredArticleJournal) readRecordBytes(ctx context.Context, name string, budget int64) (raw []byte, result error) {
	if err := deletionAnchorContext(ctx); err != nil {
		return nil, err
	}
	file, err := j.media.open(ctx, name)
	if file != nil {
		defer func() { result = errors.Join(result, file.Close()) }()
	}
	if err != nil || file == nil {
		return nil, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > budget {
		return nil, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	raw, err = io.ReadAll(io.LimitReader(file, budget+1))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	current, pathErr := os.Lstat(filepathForDeletionAnchor(j, name))
	if err != nil || pathErr != nil || !os.SameFile(before, after) || !os.SameFile(after, current) ||
		before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || int64(len(raw)) != after.Size() {
		return nil, errors.Join(recovery.ErrAdmissionDrift, err, pathErr)
	}
	return raw, deletionAnchorContext(ctx)
}

func (j *anchoredArticleJournal) decodeAnchorRecord(raw []byte, records []ArticleDeletionTombstone) (ArticleDeletionTombstone, error) {
	var disk deletionDiskRecord
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&disk); err != nil {
		return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
	}
	record, err := disk.toModel()
	if err != nil || record.Sequence != int64(len(records)+1) {
		return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
	}
	canonical, err := json.Marshal(deletionDiskRecordFromModel(record))
	if err != nil || !bytes.Equal(canonical, raw) {
		return ArticleDeletionTombstone{}, ErrDeletionJournalCorrupt
	}
	return record, j.records.verifyRecord(record, records)
}

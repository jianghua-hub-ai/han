package article

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"han/internal/platform"
)

func TestProductionDeletionReplaySource(t *testing.T) {
	f := newSourceDependencyFixture(t)
	t.Run("requires every real dependency", f.requireDependencies)
	t.Run("replayall then snapshot", f.replayThenSnapshot)
	t.Run("wrong backup fails closed", f.wrongBackup)
	t.Run("FactsOnlySnapshot", testSourceFactsOnlySnapshot)
	t.Run("ActualOutcomePartitions", testSourceActualOutcomePartitions)
	t.Run("TraceContext", testSourceTraceContext)
}

type sourceDependencyFixture struct {
	journal    VerifiedDeletionJournal
	reader     *FileBackupInventoryReader
	replayer   *replayRecorderSourceStub
	operations *staticReplayOperationReader
	audits     *staticReplayAuditReader
}

func newSourceDependencyFixture(t *testing.T) *sourceDependencyFixture {
	t.Helper()
	dir := t.TempDir()
	var journal VerifiedDeletionJournal = &stubVerifiedJournal{}
	if articleDeletionJournalPlatformSupported() {
		journalRoot := filepath.Join(dir, "journal")
		if err := os.Mkdir(journalRoot, 0o700); err != nil {
			t.Fatalf("create journal root: %v", err)
		}
		concrete, err := NewProductionFileArticleDeletionJournal(
			journalRoot,
			0o600,
			DeletionJournalKeyring{Active: DeletionJournalKey{
				ID: "article-deletion-hmac", Version: "v1",
				Material: []byte("0123456789abcdef0123456789abcdef"),
			}},
		)
		if err != nil {
			t.Fatalf("NewProductionFileArticleDeletionJournal() error = %v", err)
		}
		t.Cleanup(func() { _ = concrete.Close() })
		journal = concrete
	}

	inventoryPath := filepath.Join(dir, "inventory.json")
	writeInventory(t, inventoryPath, validInventoryManifest(t, "backup-current"))
	reader, err := NewFileBackupInventoryReader(inventoryPath)
	if err != nil {
		t.Fatalf("NewFileBackupInventoryReader() error = %v", err)
	}

	replayer := &replayRecorderSourceStub{}
	operations := &staticReplayOperationReader{facts: []DeletionOperationReplayFact{
		{OperationID: "op-1", Status: DeletionOperationCompleted, ArticleVersion: 2},
	}}
	audits := &staticReplayAuditReader{facts: []DeletionReplayAuditFact{
		{EventID: "ev-1", TargetID: "target-1", Action: "article.delete",
			OccurredAt: time.Date(2026, 10, 20, 0, 0, 0, 500, time.UTC)},
	}}

	return &sourceDependencyFixture{journal, reader, replayer, operations, audits}
}

func (f *sourceDependencyFixture) requireDependencies(t *testing.T) {
	journal, reader, replayer, operations, audits := f.journal, f.reader, f.replayer, f.operations, f.audits
	cases := []struct {
		name string
		call func() error
	}{
		{"nil journal", func() error {
			_, err := NewProductionDeletionReplaySource(journal, reader, replayer, operations, audits, time.Now)
			return err
		}},
	}
	for _, tc := range cases {
		if err := tc.call(); err != nil {
			t.Fatalf("%s error = %v, want nil", tc.name, err)
		}
	}
	if _, err := NewProductionDeletionReplaySource(
		nil, reader, replayer, operations, audits, time.Now,
	); err == nil {
		t.Fatal("nil journal accepted, want fail closed")
	}
	if _, err := NewProductionDeletionReplaySource(
		journal, nil, replayer, operations, audits, time.Now,
	); err == nil {
		t.Fatal("nil inventory accepted, want fail closed")
	}
	if _, err := NewProductionDeletionReplaySource(
		journal, reader, nil, operations, audits, time.Now,
	); err == nil {
		t.Fatal("nil replayer accepted, want fail closed")
	}
	if _, err := NewProductionDeletionReplaySource(
		journal, reader, replayer, nil, audits, time.Now,
	); err == nil {
		t.Fatal("nil operation reader accepted, want fail closed")
	}
	if _, err := NewProductionDeletionReplaySource(
		journal, reader, replayer, operations, nil, time.Now,
	); err == nil {
		t.Fatal("nil audit reader accepted, want fail closed")
	}
}

func (f *sourceDependencyFixture) replayThenSnapshot(t *testing.T) {
	journal, reader, replayer, operations, audits := f.journal, f.reader, f.replayer, f.operations, f.audits
	source, err := NewProductionDeletionReplaySource(
		journal, reader, replayer, operations, audits,
		func() time.Time { return time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("NewProductionDeletionReplaySource() error = %v", err)
	}
	input, err := source.ReplayOldBackup(context.Background(), "backup-current")
	if err != nil {
		t.Fatalf("ReplayOldBackup() error = %v", err)
	}
	if replayer.calls != 1 {
		t.Fatalf("ReplayAll calls = %d, want 1", replayer.calls)
	}
	if input.InventoryDigest == "" || input.DeletionOperationsDigest == "" ||
		input.AuditEventsDigest == "" || input.InventoryScope != BackupInventoryScopeV1 {
		t.Fatalf("input = %#v", input)
	}
	if input.AuditEventsVerifiedCount != 1 ||
		input.DeletionOperationsReconciledCount != 1 {
		t.Fatalf("counts = %#v", input)
	}
	snapshot, err := source.Snapshot(context.Background(), "backup-current")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if replayer.calls != 1 {
		t.Fatalf("Snapshot() must not run ReplayAll; calls = %d", replayer.calls)
	}
	if snapshot.InventoryDigest != input.InventoryDigest {
		t.Fatal("snapshot digest drifted without a source change")
	}
}

func (f *sourceDependencyFixture) wrongBackup(t *testing.T) {
	journal, reader, replayer, operations, audits := f.journal, f.reader, f.replayer, f.operations, f.audits
	source, _ := NewProductionDeletionReplaySource(
		journal, reader, replayer, operations, audits, time.Now,
	)
	if _, err := source.Snapshot(context.Background(), "backup-other"); !errors.Is(
		err, ErrDeletionReplayBackupMismatch,
	) {
		t.Fatalf("Snapshot() error = %v, want backup mismatch", err)
	}
}

type replayRecorderSourceStub struct {
	calls int
}

func (stub *replayRecorderSourceStub) ReplayAllTombstones(
	context.Context,
	[]ArticleDeletionTombstone,
	string,
) (DeletionReplayAllResult, error) {
	stub.calls++
	return DeletionReplayAllResult{Replayed: 0, AlreadyAbsent: 0}, nil
}

type staticReplayOperationReader struct {
	facts []DeletionOperationReplayFact
}

func (reader *staticReplayOperationReader) ReadDeletionOperations(
	context.Context,
) ([]DeletionOperationReplayFact, error) {
	return reader.facts, nil
}

type staticReplayAuditReader struct {
	facts []DeletionReplayAuditFact
}

func (reader *staticReplayAuditReader) ReadDeletionAuditEvents(
	context.Context,
) ([]DeletionReplayAuditFact, error) {
	return reader.facts, nil
}

func validInventoryManifest(t *testing.T, backupRef string) string {
	t.Helper()
	cutoff := "2026-10-20T00:00:00.000000Z"
	manifest := map[string]any{
		"inventory_version": BackupInventoryV1Version,
		"inventory_scope":   BackupInventoryScopeV1,
		"backup_ref":        backupRef,
		"backup_cutoff":     cutoff,
		"generated_at":      "2026-10-20T00:00:01.000000Z",
		"recoverable_backups": []map[string]string{
			{"backup_ref": backupRef, "backup_completed_at": cutoff},
		},
	}
	digest, err := canonicalInventoryDigest(t, manifest)
	if err != nil {
		t.Fatalf("canonicalInventoryDigest() error = %v", err)
	}
	manifest["inventory_digest"] = digest
	return marshalOrdered(t, manifest)
}

var _ = os.Getenv

type stubVerifiedJournal struct{}

func (*stubVerifiedJournal) ReadAllVerified(
	context.Context,
) ([]ArticleDeletionTombstone, error) {
	return nil, nil
}

// These component seams report explicit outcomes; physical SQL is the PG layer.
type sourceOutcomeJournal struct{ records []ArticleDeletionTombstone }

func (journal *sourceOutcomeJournal) ReadAllVerified(context.Context) ([]ArticleDeletionTombstone, error) {
	return append([]ArticleDeletionTombstone(nil), journal.records...), nil
}

type sourceOutcomeReplayer struct {
	result  DeletionReplayAllResult
	failure error
	calls   int
	ctx     context.Context
	trace   string
}

func (replayer *sourceOutcomeReplayer) ReplayAllTombstones(ctx context.Context, _ []ArticleDeletionTombstone, trace string) (DeletionReplayAllResult, error) {
	replayer.calls++
	replayer.ctx, replayer.trace = ctx, trace
	if err := ctx.Err(); err != nil {
		return DeletionReplayAllResult{}, err
	}
	return replayer.result, replayer.failure
}

func newSourceOutcomeFixture(t *testing.T, result DeletionReplayAllResult) (*ProductionDeletionReplaySource, *sourceOutcomeReplayer) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inventory.json")
	writeInventory(t, path, validInventoryManifest(t, "backup-current"))
	reader, err := NewFileBackupInventoryReader(path)
	if err != nil {
		t.Fatal(err)
	}
	journal := &sourceOutcomeJournal{}
	operations := &staticReplayOperationReader{}
	audits := &staticReplayAuditReader{}
	clock := time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC)
	for index := range 3 {
		id := fmt.Sprintf("synthetic-%d", index)
		journal.records = append(journal.records, ArticleDeletionTombstone{Namespace: ArticleDeletionNamespace,
			Sequence: int64(index + 1), ArticleID: id, ArticleVersion: 1, DeletionOperationID: id,
			DeletedAt: clock, KeyID: "synthetic", KeyVersion: "v1", Stage: "tombstoned"})
		operations.facts = append(operations.facts, DeletionOperationReplayFact{OperationID: id,
			Status: DeletionOperationCompleted, ArticleVersion: 1, CompletedAt: &clock})
		audits.facts = append(audits.facts, DeletionReplayAuditFact{EventID: id, TargetID: id, Action: "article.delete", OccurredAt: clock})
	}
	replayer := &sourceOutcomeReplayer{result: result}
	source, err := NewProductionDeletionReplaySource(journal, reader, replayer, operations, audits, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	return source, replayer
}

func testSourceFactsOnlySnapshot(t *testing.T) {
	source, replayer := newSourceOutcomeFixture(t, DeletionReplayAllResult{})
	operations := source.operations.(*staticReplayOperationReader)
	for index := range operations.facts {
		operations.facts[index].Status = DeletionOperationJournaled
		operations.facts[index].CompletedAt = nil
	}
	source.audits.(*staticReplayAuditReader).facts = nil
	facts, err := source.Snapshot(context.Background(), "backup-current")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(facts).NumField() != 10 || facts.TombstonesTotal != 3 ||
		facts.DeletionOperationsReconciledCount != 0 || facts.AuditEventsVerifiedCount != 0 || replayer.calls != 0 {
		t.Fatalf("pending Facts must have10 fields and zero execution: %#v", facts)
	}
	if _, err := normalizeReplayFacts(facts); err != nil {
		t.Fatalf("legal pre-replay pending Facts rejected: %v", err)
	}
	if facts.JournalHeadSequence != 3 || facts.JournalHeadHash == "" || facts.InventoryDigest == "" ||
		facts.DeletionOperationsDigest == "" || facts.AuditEventsDigest == "" || facts.InventoryScope != BackupInventoryScopeV1 {
		t.Fatal("snapshot omitted independently supplied facts")
	}
}

func testSourceActualOutcomePartitions(t *testing.T) {
	for _, test := range []struct {
		name    string
		result  DeletionReplayAllResult
		invalid bool
	}{
		{"AllPending3And0", DeletionReplayAllResult{Replayed: 3}, false},
		{"AllAbsent0And3", DeletionReplayAllResult{AlreadyAbsent: 3}, false},
		{"Mixed2And1", DeletionReplayAllResult{Replayed: 2, AlreadyAbsent: 1}, false},
		{"Negative", DeletionReplayAllResult{Replayed: -1, AlreadyAbsent: 4}, true},
		{"WrongSum", DeletionReplayAllResult{Replayed: 2}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, _ := newSourceOutcomeFixture(t, test.result)
			input, err := source.ReplayOldBackup(context.Background(), "backup-current")
			if test.invalid {
				if !errors.Is(err, ErrDeletionReplayInvalidRequest) || input != (DeletionReplayInput{}) {
					t.Fatal("invalid execution outcome accepted")
				}
				return
			}
			if err != nil || input.TombstonesReplayed != test.result.Replayed || input.AlreadyAbsent != test.result.AlreadyAbsent {
				t.Fatalf("real returned partitions changed: %#v, %v", input, err)
			}
			facts, err := source.Snapshot(context.Background(), "backup-current")
			if err != nil || !sameReplayFacts(input.DeletionReplayFacts, facts) {
				t.Fatal("fixed source facts changed")
			}
		})
	}
	t.Run("PartialFailure", func(t *testing.T) {
		source, replayer := newSourceOutcomeFixture(t, DeletionReplayAllResult{Replayed: 2})
		replayer.failure = errors.New("controlled replay dependency failure")
		input, err := source.ReplayOldBackup(context.Background(), "backup-current")
		if !errors.Is(err, replayer.failure) || input != (DeletionReplayInput{}) {
			t.Fatal("partial execution became successful input")
		}
	})
}

type replaySourceMarker struct{}

func testSourceTraceContext(t *testing.T) {
	for _, existing := range []bool{true, false} {
		t.Run(fmt.Sprintf("ExistingTrace_%t", existing), func(t *testing.T) {
			source, recorder := newSourceOutcomeFixture(t, DeletionReplayAllResult{Replayed: 3})
			parent := context.WithValue(context.Background(), replaySourceMarker{}, "original")
			if existing {
				var err error
				parent, err = platform.ContextWithTraceID(parent, "01ARZ3NDEKTSV4RRFFQ69G5FAV")
				if err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(parent, time.Minute)
			defer cancel()
			if _, err := source.ReplayOldBackup(ctx, "backup-current"); err != nil {
				t.Fatal(err)
			}
			trace, ok := platform.TraceIDFromContext(recorder.ctx)
			deadline, _ := ctx.Deadline()
			observed, _ := recorder.ctx.Deadline()
			if !ok || trace != recorder.trace || recorder.ctx.Value(replaySourceMarker{}) != "original" || !deadline.Equal(observed) {
				t.Fatal("trace/parent/deadline lost at real Source boundary")
			}
			if existing && trace != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
				t.Fatal("existing trusted trace replaced")
			}
			cancel()
			select {
			case <-recorder.ctx.Done():
			default:
				t.Fatal("parent cancellation lost")
			}
		})
	}
	t.Run("Cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		testSourceEndedContext(t, ctx, context.Canceled)
	})
	t.Run("Deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
		defer cancel()
		testSourceEndedContext(t, ctx, context.DeadlineExceeded)
	})
}

func testSourceEndedContext(t *testing.T, ctx context.Context, expected error) {
	source, recorder := newSourceOutcomeFixture(t, DeletionReplayAllResult{Replayed: 3})
	input, err := source.ReplayOldBackup(ctx, "backup-current")
	if !errors.Is(err, expected) || input != (DeletionReplayInput{}) || recorder.ctx.Err() != expected {
		t.Fatal("ended parent was masked or yielded successful execution")
	}
}

func TestProductionDeletionReplaySource_PublicationMetadataUsesSameObject(t *testing.T) {
	f := newSourceDependencyFixture(t)
	journal := &publicationConstructorJournal{}
	source, err := NewProductionDeletionReplaySource(journal, f.reader, f.replayer, f.operations, f.audits, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := source.publicationJournal()
	if err != nil || reader != journal {
		t.Fatal("metadata did not project same source journal")
	}
	if journal.calls.Load() != 0 {
		t.Fatal("projection performed port I/O")
	}
	legacy, err := NewProductionDeletionReplaySource(&stubVerifiedJournal{}, f.reader, f.replayer, f.operations, f.audits, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.publicationJournal(); err == nil {
		t.Fatal("legacy journal without anchored metadata accepted")
	}
	var absent *publicationConstructorJournal
	if _, err := NewProductionDeletionReplaySource(absent, f.reader, f.replayer, f.operations, f.audits, time.Now); err == nil {
		t.Fatal("typed nil journal accepted")
	}
}

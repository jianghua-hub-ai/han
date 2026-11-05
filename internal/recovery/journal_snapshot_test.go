package recovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type snapshotTestLease struct {
	lost   bool
	closes int
}

func (l *snapshotTestLease) Check(ctx context.Context) error {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if l.lost {
		return ErrAdmissionClosed
	}
	return nil
}
func (l *snapshotTestLease) Close() error { l.closes++; return nil }

func snapshotReaderFixture(t *testing.T) (*verifiedRecoveryJournal, []diskRecord, *deterministicAnchor, *snapshotTestLease) {
	t.Helper()
	key := Key{ID: "retained", Version: "K1", State: KeyStateVerifyOnly, Material: []byte("synthetic-retained-key")}
	keys, e := NewKeyring([]Key{key, {ID: "active", Version: "K2", State: KeyStateActive, Material: []byte("synthetic-active-key")}})
	if e != nil {
		t.Fatal(e)
	}
	b := diskRecord{Namespace: RecoveryNamespaceV1, Sequence: 1, RunID: "run", BackupRef: "backup", KeyID: key.ID, KeyVersion: key.Version,
		SecretFingerprint: fingerprintSecret(key, []byte("synthetic-candidate")), Stage: StageBegun, RecordedAt: time.Date(2026, 10, 29, 1, 0, 0, 123456000, time.UTC).Format(time.RFC3339Nano)}
	b.RecordHMAC = computeRecordHMAC(b, key.Material)
	// Independent Python stdlib HMAC oracle over the exact synthetic wire fields.
	if b.SecretFingerprint != "hmac-sha256:806ff252248ce95ee6d69baa4e2891ed2a571e7fd18a64618ec1e1da94a519ad" || b.RecordHMAC != "hmac-sha256:7537b61529ce87dbb78b8d7e6f6a39373cdd3da98bf92b8f73b6c3e5de4b6533" {
		t.Fatal("synthetic HMAC producer differs from independent oracle")
	}
	records := []diskRecord{b}
	a := newDeterministicAnchor()
	a.forceSnapshot(AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1, Committed: headForRecords(records)})
	j := &FileRecoveryJournal{journalID: "journal", epoch: 1, anchor: a, keyring: keys}
	l := &snapshotTestLease{}
	r := &verifiedRecoveryJournal{journal: j, maxBytes: 8 << 20, maxRecords: 100000,
		lock: func(context.Context) (ExclusiveLease, error) { return l, nil }, records: func(context.Context) ([]diskRecord, error) { return records, nil },
		artifacts: func(context.Context) ([]string, error) { return nil, nil }}
	return r, records, a, l
}

func TestRecoveryJournalReadOnlyContext(t *testing.T) {
	t.Run("PendingStageMissingAndBudget", testSnapshotFailures)
	t.Run("VerifierSourceDriftAndZeroEffects", testVerifierSourceFences)
	t.Run("ContextLockAndWriteUnknown", testContextJournalOperations)
	t.Run("CommittedReadOnly", func(t *testing.T) {
		r, records, _, l := snapshotReaderFixture(t)
		s, e := r.ReadVerifiedSnapshot(context.Background(), RecoveryNamespaceV1)
		if e != nil || len(s.Entries) != 1 || s.Entries[0] != records[0].toEntry() {
			t.Fatalf("committed read: %+v %v", s, e)
		}
		if l.closes != 1 {
			t.Fatal("read lease not closed")
		}
	})
	t.Run("RetainedBeginSecretReadOnlyVerifier", func(t *testing.T) {
		r, records, _, _ := snapshotReaderFixture(t)
		e, err := r.VerifyBeginSecret(context.Background(), RunBinding{RunID: "run", BackupRef: "backup"}, []byte("synthetic-candidate"))
		if err != nil || e != records[0].toEntry() {
			t.Fatalf("retained K1 verification with active K2: %+v %v", e, err)
		}
		for _, bad := range []string{"secret", "run", "backup"} {
			t.Run(bad, func(t *testing.T) {
				b := RunBinding{RunID: "run", BackupRef: "backup"}
				secret := []byte("synthetic-candidate")
				if bad == "secret" {
					secret = []byte("synthetic-wrong")
				}
				if bad == "run" {
					b.RunID = "wrong"
				}
				if bad == "backup" {
					b.BackupRef = "wrong"
				}
				if _, e := r.VerifyBeginSecret(context.Background(), b, secret); !errors.Is(e, ErrAdmissionBinding) {
					t.Fatalf("bad candidate binding: %v", e)
				}
			})
		}
	})
}

func testSnapshotFailures(t *testing.T) {
	for _, kind := range []string{"nilctx", "cancel", "pending", "orphan", "missing", "bytes", "records", "unknown-version", "unknown-stage", "missing-key", "tamper", "UTC", "lock-loss"} {
		t.Run(kind, func(t *testing.T) {
			r, records, a, l := snapshotReaderFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "nilctx":
				ctx = nil
			case "cancel":
				cancel()
			case "pending":
				s := a.snapshot
				next := JournalHead{Sequence: 2, Hash: fixtureHash("pending")}
				s.Pending = &AnchorPending{OldHead: s.Committed, NewHead: next, OperationID: journalOperationID("journal", next)}
				a.forceSnapshot(s)
			case "orphan":
				r.artifacts = func(context.Context) ([]string, error) { return []string{"orphan"}, nil }
			case "missing":
				r.records = func(context.Context) ([]diskRecord, error) { return nil, os.ErrNotExist }
			case "bytes":
				raw, _ := encodeDiskRecords(records)
				r.maxBytes = int64(len(raw) - 1)
			case "records":
				r.maxRecords = 1
				r.records = func(context.Context) ([]diskRecord, error) { return append(records, records[0]), nil }
			case "unknown-version":
				records[0].Namespace = "recovery/v2"
			case "unknown-stage":
				records[0].Stage = "unsupported"
			case "missing-key":
				r.journal.keyring, _ = NewKeyring([]Key{{ID: "active", Version: "K2", State: KeyStateActive, Material: []byte("synthetic-active-key")}})
			case "tamper":
				records[0].RecordHMAC = "hmac-sha256:" + strings.Repeat("0", 64)
			case "UTC":
				records[0].RecordedAt = "2026-10-29T01:00:00.123456001Z"
			case "lock-loss":
				l.lost = true
			}
			if _, e := r.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1); e == nil {
				t.Fatal("incomplete source accepted")
			}
			if l.closes > 1 {
				t.Fatal("lease closed twice")
			}
		})
	}
	t.Run("NativeSupportMatrix", testReaderNativeSupport)
}

func testReaderNativeSupport(t *testing.T) {
	r, records, base, _ := snapshotReaderFixture(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "recovery.journal")
	a := &snapshotAnchorProbe{deterministicAnchor: base}
	options := ReadOptions{Path: path, FileMode: 0o600, JournalID: "journal", Epoch: 1, Anchor: a, Keyring: r.journal.keyring, MaxBytes: 8 << 20, MaxRecords: 100000}
	if runtime.GOOS == "windows" {
		_, e := OpenVerifiedRecoveryJournal(context.Background(), options)
		if !errors.Is(e, ErrAdmissionUnsupported) || !errors.Is(e, ErrUnsupportedPlatform) {
			t.Fatalf("platform: %v", e)
		}
		x, e := os.ReadDir(dir)
		if e != nil || len(x) != 0 || a.loads != 0 || a.cas != 0 {
			t.Fatal("unsupported constructor touched external media")
		}
		return
	}
	if e := os.Chmod(dir, 0o700); e != nil {
		t.Fatal(e)
	}
	raw, _ := encodeDiskRecords(records)
	if e := os.WriteFile(path, raw, 0o600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path+".lock", nil, 0o600); e != nil {
		t.Fatal(e)
	}
	reader, e := OpenVerifiedRecoveryJournal(context.Background(), options)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := reader.VerifyBeginSecret(context.Background(), RunBinding{RunID: "run", BackupRef: "backup"}, []byte("synthetic-candidate")); e != nil {
		t.Fatal(e)
	}
	if _, e := reader.Begin(context.Background(), "other", "backup", []byte("synthetic")); !errors.Is(e, ErrAdmissionClosed) {
		t.Fatalf("read-only handle wrote: %v", e)
	}
	after, e := os.ReadFile(path)
	if e != nil || string(after) != string(raw) || a.cas != 0 {
		t.Fatal("pure native read changed source")
	}
}

type snapshotAnchorProbe struct {
	*deterministicAnchor
	loads, cas int
	hook       func(int)
}

func (a *snapshotAnchorProbe) Load(ctx context.Context, ns string) (AnchorSnapshot, string, error) {
	a.loads++
	if a.hook != nil {
		a.hook(a.loads)
	}
	if e := checkedContext(ctx); e != nil {
		return AnchorSnapshot{}, "", e
	}
	return a.deterministicAnchor.Load(ctx, ns)
}
func (a *snapshotAnchorProbe) CompareAndSwap(context.Context, string, string, AnchorSnapshot) (AnchorSnapshot, string, error) {
	a.cas++
	return AnchorSnapshot{}, "", errors.New("reader attempted write")
}

func testVerifierSourceFences(t *testing.T) {
	for _, kind := range []string{"ABA", "journal-id", "namespace", "epoch", "cancel", "lost-lock", "keyring", "key-version"} {
		t.Run(kind, func(t *testing.T) {
			r, records, base, l := snapshotReaderFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a := &snapshotAnchorProbe{deterministicAnchor: base}
			r.journal.anchor = a
			a.hook = func(n int) {
				if n != 3 {
					return
				}
				switch kind {
				case "ABA":
					base.revision += "-unexplained-ABA"
				case "journal-id":
					base.snapshot.JournalID = "other"
				case "namespace":
					base.snapshot.Namespace = ArticleDeletionNamespaceV1
				case "epoch":
					base.snapshot.Epoch++
				case "cancel":
					cancel()
				case "lost-lock":
					l.lost = true
				case "keyring":
					keys := r.journal.keyring.retained()
					keys = append(keys, Key{ID: "retained-extra", Version: "v0", State: KeyStateVerifyOnly, Material: []byte("synthetic-extra")})
					r.journal.keyring, _ = NewKeyring(keys)
				case "key-version":
					records[0].KeyVersion = "unknown"
				}
			}
			if _, e := r.VerifyBeginSecret(ctx, RunBinding{RunID: "run", BackupRef: "backup"}, []byte("synthetic-candidate")); e == nil {
				t.Fatal("source drift accepted")
			}
			if a.loads < 3 || a.cas != 0 || l.closes != 1 {
				t.Fatalf("injection/write/join: loads=%d CAS=%d close=%d", a.loads, a.cas, l.closes)
			}
		})
	}
}

type contextCASProbe struct {
	*deterministicAnchor
	ctx              context.Context
	calls            int
	commit           bool
	entered, release chan struct{}
}

func (a *contextCASProbe) CompareAndSwap(ctx context.Context, ns, token string, next AnchorSnapshot) (AnchorSnapshot, string, error) {
	a.ctx = ctx
	a.calls++
	if a.entered != nil {
		close(a.entered)
		<-a.release
	}
	if a.commit {
		_, _, _ = a.deterministicAnchor.CompareAndSwap(ctx, ns, token, next)
	}
	return AnchorSnapshot{}, "", errors.New("synthetic lost ACK")
}

func testContextJournalOperations(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "ACKUnknown", true: "ACKCommittedReadback"}[commit], func(t *testing.T) { testContextACK(t, commit) })
	}
	t.Run("CancelledWriteActuallyJoins", testJournalWriteJoin)
	t.Run("HeldWriterContinuity", testHeldWriterContinuity)
	t.Run("PanicFinalization", testWriterPanicFinalization)
	t.Run("ReaderLeaseFinalization", testReaderLeaseFinalization)
	t.Run("DirectorySyncCancellationFence", testDirectorySyncFence)
	t.Run("WriterRetainsOriginalAnchor", testWriterSourceAnchor)
	t.Run("StagingStopsBetweenSynchronousActions", testStagingCancellationFence)
	t.Run("CancelledLockWait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		done := make(chan error, 1)
		j := &FileRecoveryJournal{acquireLock: func(ctx context.Context, _ string, _ fs.FileMode) (*journalFileLock, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}}
		go func() {
			done <- j.withLock(ctx, func(ctx context.Context) error { t.Error("cancelled wait ran operation"); return nil })
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("lock wait did not start")
		}
		cancel()
		select {
		case e := <-done:
			if !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
		case <-time.After(time.Second):
			t.Fatal("lock wait ignored cancellation")
		}
	})
}

func testJournalWriteJoin(t *testing.T) {
	a := &contextCASProbe{deterministicAnchor: newDeterministicAnchor(), entered: make(chan struct{}), release: make(chan struct{}), commit: true}
	j := &FileRecoveryJournal{journalID: "journal", epoch: 1, anchor: a}
	closed := 0
	j.acquireLock = func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
		return &journalFileLock{check: checkedContext, close: func() error { closed++; return nil }}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- j.withLock(ctx, func(ctx context.Context) error {
			_, _, e := j.compareAndSwapAnchor(ctx, "", AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1})
			return e
		})
	}()
	select {
	case <-a.entered:
	case <-time.After(time.Second):
		t.Fatal("write did not start")
	}
	cancel()
	select {
	case <-done:
		t.Error("cancel detached write")
	default:
	}
	if closed != 0 {
		t.Error("lease released before true write completion")
	}
	close(a.release)
	select {
	case e := <-done:
		if !errors.Is(e, ErrRecoveryWriteUnknown) || !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("write did not join")
	}
	if closed != 1 {
		t.Fatal("write lease close count")
	}
}

type stagingObservationContext struct {
	context.Context
	cancel     context.CancelFunc
	dir        string
	afterWrite bool
}

func (c stagingObservationContext) Err() error {
	entries, _ := os.ReadDir(c.dir)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			info, e := entry.Info()
			if e == nil && (!c.afterWrite || info.Size() > 0) {
				c.cancel()
			}
		}
	}
	return c.Context.Err()
}
func testStagingCancellationFence(t *testing.T) {
	for _, afterWrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "AfterCreate", true: "AfterWrite"}[afterWrite], func(t *testing.T) {
			r, records, _, _ := snapshotReaderFixture(t)
			dir := t.TempDir()
			j := r.journal
			j.path = filepath.Join(dir, "recovery.journal")
			j.fileMode = 0o600
			j.readMaxBytes = 8 << 20
			j.readMaxRecords = 100000
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx := stagingObservationContext{Context: base, cancel: cancel, dir: dir, afterWrite: afterWrite}
			installSyntheticWriterLock(j)
			e := j.withLock(ctx, func(ctx context.Context) error { _, e := j.stageRecords(ctx, records); return e })
			if !errors.Is(e, context.Canceled) {
				t.Fatalf("stage cancellation cause lost: %v", e)
			}
			files, e := os.ReadDir(dir)
			if e != nil || len(files) != 1 {
				t.Fatalf("staged evidence not retained: %v", e)
			}
			info, e := files[0].Info()
			if e != nil {
				t.Fatal(e)
			}
			if !afterWrite && info.Size() != 0 {
				t.Fatal("write started after CreateTemp returned cancelled")
			}
			if afterWrite && info.Size() == 0 {
				t.Fatal("write boundary was not reached")
			}
			f, e := os.OpenFile(filepath.Join(dir, files[0].Name()), os.O_WRONLY, 0)
			if e != nil {
				t.Fatalf("staging handle not closed: %v", e)
			}
			if e = f.Close(); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func testDirectorySyncFence(t *testing.T) {
	for _, before := range []bool{true, false} {
		t.Run(map[bool]string{true: "BeforeStart", false: "AfterReturn"}[before], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			cause := errors.New("synthetic sync ACK unknown")
			j := &FileRecoveryJournal{path: filepath.Join(t.TempDir(), "journal"), syncDirectory: func(string) error { calls++; cancel(); return cause }}
			if before {
				cancel()
			}
			fixture, _, _, _ := snapshotReaderFixture(t)
			j.anchor = fixture.journal.anchor
			j.journalID = "journal"
			j.epoch = 1
			installSyntheticWriterLock(j)
			e := j.withLock(ctx, func(ctx context.Context) error { return j.syncJournalDirectory(ctx) })
			if !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
			if before && calls != 0 || !before && (calls != 1 || !errors.Is(e, cause)) {
				t.Fatalf("sync calls/cause: %d %v", calls, e)
			}
		})
	}
}
func testWriterSourceAnchor(t *testing.T) {
	for _, kind := range []string{"ABA", "head", "id", "namespace", "epoch"} {
		t.Run(kind, func(t *testing.T) {
			r, _, base, _ := snapshotReaderFixture(t)
			expected := VerifiedJournalSnapshot{JournalID: base.snapshot.JournalID, Namespace: base.snapshot.Namespace, Epoch: base.snapshot.Epoch, Head: base.snapshot.Committed, Revision: base.revision}
			ctx := context.WithValue(context.Background(), consumptionSourceKey{}, &consumptionSources{recovery: expected})
			switch kind {
			case "ABA":
				base.revision += "-changed"
			case "head":
				base.snapshot.Committed = JournalHead{}
			case "id":
				base.snapshot.JournalID = "different"
			case "namespace":
				base.snapshot.Namespace = ArticleDeletionNamespaceV1
			case "epoch":
				base.snapshot.Epoch++
			}
			installSyntheticWriterLock(r.journal)
			if _, e := lockedTestRecords(ctx, r.journal, false); !errors.Is(e, ErrAdmissionDrift) {
				t.Fatalf("writer reset original authority: %v", e)
			}
		})
	}
}

// Explicit portable unit-test lease. Native tests keep their real OS lock.
func installSyntheticWriterLock(j *FileRecoveryJournal) {
	j.acquireLock = func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
		return &journalFileLock{check: checkedContext, close: func() error { return nil }}, nil
	}
}
func lockedTestCAS(ctx context.Context, j *FileRecoveryJournal, old string, next AnchorSnapshot) (a AnchorSnapshot, token string, e error) {
	e = j.withLock(ctx, func(ctx context.Context) error {
		var err error
		a, token, err = j.compareAndSwapAnchor(ctx, old, next)
		return err
	})
	return
}
func lockedTestRecords(ctx context.Context, j *FileRecoveryJournal, allow bool) (records []diskRecord, e error) {
	e = j.withLock(ctx, func(ctx context.Context) error { var err error; records, err = j.loadRecords(ctx, allow); return err })
	return
}
func lockedTestPending(ctx context.Context, j *FileRecoveryJournal, token string, a AnchorSnapshot, head JournalHead, syncRequired bool) (h JournalHead, e error) {
	e = j.withLock(ctx, func(ctx context.Context) error {
		var err error
		h, err = j.commitPendingRecovery(ctx, token, a, head, syncRequired)
		return err
	})
	return
}

type writerBoundaryProbe struct {
	*deterministicAnchor
	cause, ackErr                                   error
	dir, path, after                                string
	lost, changed                                   bool
	loseAtCAS, calls, syncs, closes, readsAfterLoss int
}

func (p *writerBoundaryProbe) Load(ctx context.Context, ns string) (AnchorSnapshot, string, error) {
	if p.lost {
		p.readsAfterLoss++
	}
	return p.deterministicAnchor.Load(ctx, ns)
}
func (p *writerBoundaryProbe) CompareAndSwap(ctx context.Context, ns, old string, next AnchorSnapshot) (AnchorSnapshot, string, error) {
	p.calls++
	a, token, e := p.deterministicAnchor.CompareAndSwap(ctx, ns, old, next)
	if p.calls == p.loseAtCAS {
		p.lost = true
	}
	if p.ackErr != nil {
		return AnchorSnapshot{}, "", p.ackErr
	}
	return a, token, e
}

type writerBoundaryContext struct {
	context.Context
	p *writerBoundaryProbe
}

func (c writerBoundaryContext) Err() error { c.p.observe(); return c.Context.Err() }
func (p *writerBoundaryProbe) observe() {
	if p.changed {
		return
	}
	files, _ := os.ReadDir(p.dir)
	if p.after == "ArtifactRename" {
		hasStage, hasMain := false, false
		for _, f := range files {
			hasStage = hasStage || strings.Contains(f.Name(), ".tmp-")
			hasMain = hasMain || filepath.Join(p.dir, f.Name()) == p.path
		}
		if !hasStage && hasMain {
			p.changed = true
			p.lost = true
		}
		return
	}
	for _, f := range files {
		info, e := f.Info()
		if e != nil {
			continue
		}
		stage := strings.Contains(f.Name(), ".tmp-")
		ready := stage && (p.after == "Create" || p.after == "ABA" || p.after == "Chmod" && info.Mode().Perm()&0o200 == 0 || p.after == "Write" && info.Size() > 0) || p.after == "Rename" && filepath.Join(p.dir, f.Name()) == p.path
		if !ready {
			continue
		}
		p.changed = true
		if p.after == "ABA" {
			p.revision += "-unexplained"
		} else {
			p.lost = true
		}
		return
	}
}
func newBoundaryWriter(t *testing.T) (*FileRecoveryJournal, *writerBoundaryProbe, context.Context, []diskRecord) {
	t.Helper()
	reader, records, a, _ := snapshotReaderFixture(t)
	a.forceSnapshot(AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1})
	dir := t.TempDir()
	p := &writerBoundaryProbe{deterministicAnchor: a, dir: dir, path: filepath.Join(dir, "journal"), cause: errors.New("synthetic operation lock identity lost")}
	j := reader.journal
	j.path = p.path
	j.fileMode = 0o600
	j.readMaxBytes = 8 << 20
	j.readMaxRecords = 100000
	j.anchor = p
	j.acquireLock = func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
		return &journalFileLock{check: func(ctx context.Context) error {
			if e := checkedContext(ctx); e != nil {
				return e
			}
			if p.lost {
				return p.cause
			}
			return nil
		}, close: func() error { p.closes++; return nil }}, nil
	}
	j.syncDirectory = func(string) error {
		p.syncs++
		if p.after == "DirectorySync" {
			p.lost = true
		}
		return nil
	}
	base, cancel := context.WithTimeout(context.WithValue(context.Background(), "synthetic-trace", "trace"), 30*time.Second)
	t.Cleanup(cancel)
	return j, p, writerBoundaryContext{Context: base, p: p}, records
}
func testHeldWriterContinuity(t *testing.T) {
	t.Run("PositiveStaging", testPositiveGuardedStaging)
	for _, boundary := range []string{"Create", "Chmod", "Write", "PendingCAS", "Rename", "DirectorySync", "CommitCAS", "ABA"} {
		t.Run("NestedPersist/"+boundary, func(t *testing.T) { testNestedWriterLoss(t, boundary) })
	}
	t.Run("FirstCASLossStopsSecond", testCASLossStopsSecond)
	t.Run("PendingRepair", testPendingWriterLoss)
	t.Run("InstanceAndExpiredContext", testWriterContextScope)
}
func testPositiveGuardedStaging(t *testing.T) {
	j, p, ctx, records := newBoundaryWriter(t)
	var path string
	e := j.withLock(ctx, func(ctx context.Context) error { var e error; path, e = j.stageRecords(ctx, records); return e })
	raw, readErr := os.ReadFile(path)
	expected, _ := encodeDiskRecords(records)
	if e != nil || readErr != nil || string(raw) != string(expected) || p.closes != 1 || p.calls != 0 {
		t.Fatalf("positive staging failed: %v %v closes=%d CAS=%d", e, readErr, p.closes, p.calls)
	}
}
func testNestedWriterLoss(t *testing.T, boundary string) {
	j, p, ctx, records := newBoundaryWriter(t)
	p.after = boundary
	if boundary == "Chmod" {
		j.fileMode = 0o400
	}
	if boundary == "PendingCAS" {
		p.loseAtCAS = 1
		p.ackErr = errors.New("synthetic CAS ACK lost")
	}
	if boundary == "CommitCAS" {
		p.loseAtCAS = 2
	}
	e := j.withLock(ctx, func(ctx context.Context) error { return j.persistRecords(ctx, nil, records) })
	want := p.cause
	if boundary == "ABA" {
		want = ErrAdmissionDrift
	}
	if !errors.Is(e, want) || !errors.Is(e, ErrRecoveryWriteUnknown) {
		t.Fatalf("loss/unknown cause: %v", e)
	}
	if p.ackErr != nil && !errors.Is(e, p.ackErr) {
		t.Fatal("original provider ACK cause lost")
	}
	expectedCAS := 0
	if boundary == "PendingCAS" || boundary == "Rename" || boundary == "DirectorySync" {
		expectedCAS = 1
	}
	if boundary == "CommitCAS" {
		expectedCAS = 2
	}
	expectedSync := 0
	if boundary == "DirectorySync" || boundary == "CommitCAS" {
		expectedSync = 1
	}
	if p.calls != expectedCAS || p.syncs != expectedSync || p.closes != 1 || p.readsAfterLoss != 0 {
		t.Fatalf("later operation after loss: CAS=%d sync=%d close=%d readsAfterLoss=%d", p.calls, p.syncs, p.closes, p.readsAfterLoss)
	}
	assertNestedWriterEvidence(t, j, p, boundary)
	t.Logf("boundary=%s CAS=%d sync=%d closes=%d err=%v", boundary, p.calls, p.syncs, p.closes, e)
}
func testCASLossStopsSecond(t *testing.T) {
	j, p, ctx, _ := newBoundaryWriter(t)
	p.loseAtCAS = 1
	e := j.withLock(ctx, func(ctx context.Context) error {
		initial := AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1}
		_, token, e := j.compareAndSwapAnchor(ctx, "forced", initial)
		if e != nil {
			return e
		}
		_, _, e = j.compareAndSwapAnchor(ctx, token, initial)
		return e
	})
	if !errors.Is(e, p.cause) || !errors.Is(e, ErrRecoveryWriteUnknown) || p.calls != 1 || p.closes != 1 || p.readsAfterLoss != 0 {
		t.Fatalf("lost lock allowed next CAS/read or early release: calls=%d reads=%d closes=%d err=%v", p.calls, p.readsAfterLoss, p.closes, e)
	}
}
func testPendingWriterLoss(t *testing.T) {
	for _, boundary := range []string{"BeforeSync", "AfterSync", "AfterCAS", "Positive"} {
		t.Run(boundary, func(t *testing.T) {
			j, p, ctx, records := newBoundaryWriter(t)
			old := JournalHead{}
			next := headForRecords(records)
			a := AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1, Pending: &AnchorPending{OldHead: old, NewHead: next, OperationID: journalOperationID("journal", next)}}
			p.forceSnapshot(a)
			if boundary == "AfterSync" {
				p.after = "DirectorySync"
			}
			if boundary == "AfterCAS" {
				p.loseAtCAS = 1
			}
			e := j.withLock(ctx, func(ctx context.Context) error {
				if boundary == "BeforeSync" {
					p.lost = true
				}
				_, e := j.commitPendingRecovery(ctx, "forced", a, next, true)
				return e
			})
			expectedCAS := 0
			if boundary == "Positive" || boundary == "AfterCAS" {
				expectedCAS = 1
			}
			expectedSync := 1
			if boundary == "BeforeSync" {
				expectedSync = 0
			}
			if p.calls != expectedCAS || p.syncs != expectedSync || p.closes != 1 {
				t.Fatalf("pending action continued: CAS=%d sync=%d closes=%d", p.calls, p.syncs, p.closes)
			}
			if boundary == "Positive" {
				if e != nil || p.snapshot.Pending != nil || p.snapshot.Committed != next {
					t.Fatalf("pending positive: %v", e)
				}
			} else if !errors.Is(e, p.cause) {
				t.Fatalf("pending loss cause: %v", e)
			}
			if boundary == "AfterCAS" && !errors.Is(e, ErrRecoveryWriteUnknown) {
				t.Fatal("attempted CAS lost unknown")
			}
		})
	}
}
func testWriterContextScope(t *testing.T) {
	j, p, ctx, records := newBoundaryWriter(t)
	other, otherProbe, _, _ := newBoundaryWriter(t)
	var expired context.Context
	e := j.withLock(ctx, func(owned context.Context) error {
		expired = owned
		if _, e := other.stageRecords(owned, records); !errors.Is(e, ErrAdmissionBinding) {
			t.Fatalf("borrowed another operation's lock: %v", e)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e := j.stageRecords(expired, records); !errors.Is(e, ErrAdmissionBinding) {
		t.Fatalf("expired operation accepted: %v", e)
	}
	files, _ := os.ReadDir(p.dir)
	otherFiles, _ := os.ReadDir(otherProbe.dir)
	if len(files) != 0 || len(otherFiles) != 0 || p.closes != 1 || otherProbe.closes != 0 {
		t.Fatal("invalid operation acquired/wrote/released another lease")
	}
}

func testContextACK(t *testing.T, commit bool) {
	a := &contextCASProbe{deterministicAnchor: newDeterministicAnchor(), commit: commit}
	j := &FileRecoveryJournal{journalID: "journal", epoch: 1, anchor: a}
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), "synthetic-trace", "trace"), 30*time.Second)
	defer cancel()
	next := AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1}
	installSyntheticWriterLock(j)
	original := ctx
	e := j.withLock(ctx, func(ctx context.Context) error {
		_, _, e := j.compareAndSwapAnchor(ctx, "", next)
		if a.calls != 1 || a.ctx != ctx {
			t.Fatal("context changed or CAS repeated")
		}
		deadline, _ := ctx.Deadline()
		want, _ := original.Deadline()
		if deadline != want || ctx.Value("synthetic-trace") != original.Value("synthetic-trace") {
			t.Fatal("context values/deadline changed")
		}
		return e
	})
	if commit && e != nil || !commit && !errors.Is(e, ErrRecoveryWriteUnknown) {
		t.Fatalf("ACK classification: %v", e)
	}

}

func assertNestedWriterEvidence(t *testing.T, j *FileRecoveryJournal, p *writerBoundaryProbe, boundary string) {
	t.Helper()
	files, _ := os.ReadDir(p.dir)
	var bytes int64
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			t.Fatal(err)
		}
		bytes += info.Size()
	}
	if (boundary == "Create" || boundary == "Chmod" || boundary == "ABA") && bytes != 0 {
		t.Fatalf("write after lost authority: %d bytes", bytes)
	}
	if _, err := os.Stat(j.path); boundary == "PendingCAS" && !errors.Is(err, os.ErrNotExist) {
		t.Fatal("rename ran after pending CAS lost lock")
	}

}

type panicWriterAnchor struct {
	JournalHeadAnchor
	sentinel any
	seen     context.Context
	calls    int
}

func (p *panicWriterAnchor) CompareAndSwap(ctx context.Context, _ string, _ string, _ AnchorSnapshot) (AnchorSnapshot, string, error) {
	p.calls++
	p.seen = ctx
	panic(p.sentinel)
}

type panicWriterRetirement struct {
	sentinel any
	seen     context.Context
	calls    int
}

func (p *panicWriterRetirement) VerifyAndConsume(ctx context.Context, _ KeyRetirementRequest, _ KeyRetirementCapability) (KeyRetirementVerification, error) {
	p.calls++
	p.seen = ctx
	panic(p.sentinel)
}
func testWriterPanicFinalization(t *testing.T) {
	for _, port := range []string{"Anchor", "Retirement", "InitialCheck"} {
		t.Run(port, func(t *testing.T) { testWriterPortPanic(t, port) })
	}
	t.Run("NormalReturn", func(t *testing.T) { testWriterReturnFinalization(t, false) })
	t.Run("ErrorAndCloseJoin", func(t *testing.T) { testWriterReturnFinalization(t, true) })
}
func catchWriterPanic(fn func()) (p any) { defer func() { p = recover() }(); fn(); return }
func testWriterPortPanic(t *testing.T, port string) {
	j, p, ctx, _ := newBoundaryWriter(t)
	sentinel := &struct{ port string }{port}
	anchor := &panicWriterAnchor{JournalHeadAnchor: p, sentinel: sentinel}
	retirement := &panicWriterRetirement{sentinel: sentinel}
	if port == "Anchor" {
		j.anchor = anchor
	}
	if port == "Retirement" {
		j.keyRetirementVerifier = retirement
	}
	lease := &panicWriterLease{port: port, sentinel: sentinel, anchor: anchor, retirement: retirement}
	j.acquireLock = func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
		return &journalFileLock{check: lease.check, close: lease.close}, nil
	}
	got := catchWriterPanic(func() {
		_ = j.withLock(ctx, func(owned context.Context) error {
			if port == "Anchor" {
				_, _, e := j.compareAndSwapAnchor(owned, p.revision, p.snapshot)
				return e
			}
			if port == "Retirement" {
				return j.verifyAndConsumeKeyRetirementCapability(owned, KeyRetirementRequest{KeyID: "synthetic", KeyVersion: "v1"}, KeyRetirementCapability{})
			}
			t.Error("initial Check panic failed to stop callback")
			return nil
		})
	})
	seen, calls, closes, closedBeforeClose := lease.seen, lease.calls, lease.closes, lease.closedBeforeClose
	if got != sentinel || calls != 1 || seen == nil {
		t.Fatalf("actual %s panic identity/call missing: got=%v calls=%d", port, got, calls)
	}
	op := seen.Value(writerOperationKey{}).(*writerOperation)
	_, e := j.writerOperation(seen)
	if closes != 1 || !closedBeforeClose || !op.closed.Load() || !errors.Is(e, ErrAdmissionBinding) {
		t.Fatalf("panic cleanup failed: closes=%d closedBeforeClose=%t opClosed=%t escapedGuard=%v", closes, closedBeforeClose, op.closed.Load(), e)
	}
	t.Logf("port=%s original_panic_preserved=true calls=%d releases=%d closedBeforeClose=%t escapedGuard=%v", port, calls, closes, closedBeforeClose, e)
}
func testWriterReturnFinalization(t *testing.T, fail bool) {
	j, _, ctx, _ := newBoundaryWriter(t)
	var escaped context.Context
	closes := 0
	closedBeforeClose := false
	operationErr, closeErr := errors.New("synthetic operation error"), errors.New("synthetic close error")
	j.acquireLock = func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
		return &journalFileLock{check: checkedContext, close: func() error {
			closes++
			closedBeforeClose = escaped.Value(writerOperationKey{}).(*writerOperation).closed.Load()
			if fail {
				return closeErr
			}
			return nil
		}}, nil
	}
	e := j.withLock(ctx, func(owned context.Context) error {
		escaped = owned
		if fail {
			return operationErr
		}
		return nil
	})
	if fail && (!errors.Is(e, operationErr) || !errors.Is(e, closeErr)) || !fail && e != nil {
		t.Fatalf("normal/error aggregation: %v", e)
	}
	_, guardErr := j.writerOperation(escaped)
	if closes != 1 || !closedBeforeClose || !errors.Is(guardErr, ErrAdmissionBinding) {
		t.Fatalf("normal/error cleanup: closes=%d closedBeforeClose=%t escapedGuard=%v", closes, closedBeforeClose, guardErr)
	}
}

type panicWriterLease struct {
	port              string
	sentinel          any
	anchor            *panicWriterAnchor
	retirement        *panicWriterRetirement
	seen              context.Context
	calls, closes     int
	closedBeforeClose bool
}

func (p *panicWriterLease) check(ctx context.Context) error {
	if p.port == "InitialCheck" {
		p.seen = ctx
		p.calls++
		panic(p.sentinel)
	}
	return checkedContext(ctx)
}
func (p *panicWriterLease) close() error {
	p.closes++
	if p.port == "Anchor" {
		p.seen = p.anchor.seen
		p.calls = p.anchor.calls
	}
	if p.port == "Retirement" {
		p.seen = p.retirement.seen
		p.calls = p.retirement.calls
	}
	if p.seen != nil {
		p.closedBeforeClose = p.seen.Value(writerOperationKey{}).(*writerOperation).closed.Load()
	}
	return errors.New("synthetic Close error while unwinding")
}

type readerFinalizationLease struct {
	checks, closes int
	marker         any
	closeErr       error
}

func (l *readerFinalizationLease) Check(ctx context.Context) error {
	l.checks++
	if l.marker != nil {
		panic(l.marker)
	}
	return checkedContext(ctx)
}
func (l *readerFinalizationLease) Close() error { l.closes++; return l.closeErr }

type readerPanicAnchor struct {
	JournalHeadAnchor
	marker any
	calls  int
}

func (p *readerPanicAnchor) Load(context.Context, string) (AnchorSnapshot, string, error) {
	p.calls++
	panic(p.marker)
}
func readerLifecycleFixture(t *testing.T) (*verifiedRecoveryJournal, *snapshotAnchorProbe, *readerFinalizationLease) {
	t.Helper()
	r, _, base, _ := snapshotReaderFixture(t)
	a := &snapshotAnchorProbe{deterministicAnchor: base}
	r.journal.anchor = a
	lease := &readerFinalizationLease{}
	r.lock = func(context.Context) (ExclusiveLease, error) { return lease, nil }
	return r, a, lease
}
func callReaderLifecycle(t *testing.T, r *verifiedRecoveryJournal, ctx context.Context, method string) error {
	t.Helper()
	if method == "Snapshot" {
		s, e := r.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
		if e != nil && (s.JournalID != "" || len(s.Entries) != 0) {
			t.Error("failed read exposed partial proof")
		}
		return e
	}
	entry, e := r.VerifyBeginSecret(ctx, RunBinding{RunID: "run", BackupRef: "backup"}, []byte("synthetic-candidate"))
	if e != nil && entry != (JournalEntry{}) {
		t.Error("failed verifier exposed a successful entry")
	}
	return e
}
func testReaderLeaseFinalization(t *testing.T) {
	for _, method := range []string{"Snapshot", "VerifyBeginSecret"} {
		t.Run(method, func(t *testing.T) {
			for _, site := range []string{"Anchor", "Check", "Records", "Artifacts"} {
				t.Run("Panic/"+site, func(t *testing.T) { testReadPortPanic(t, method, site) })
			}
			for _, mode := range []string{"Normal", "ReadError", "CloseError", "ReadAndCloseError"} {
				t.Run(mode, func(t *testing.T) { testReadErrorJoin(t, method, mode) })
			}
			t.Run("CancelledPortActuallyJoins", func(t *testing.T) { testReadPortJoin(t, method) })
		})
	}
}
func catchReadPanic(fn func()) (p any) { defer func() { p = recover() }(); fn(); return }
func testReadPortPanic(t *testing.T, method, site string) {
	r, a, lease := readerLifecycleFixture(t)
	marker := &struct{ site string }{site}
	calls := 0
	lease.closeErr = errors.New("synthetic reader Close error during unwind")
	switch site {
	case "Anchor":
		r.journal.anchor = &readerPanicAnchor{JournalHeadAnchor: a, marker: marker}
	case "Check":
		lease.marker = marker
	case "Records":
		r.records = func(context.Context) ([]diskRecord, error) { calls++; panic(marker) }
	case "Artifacts":
		r.artifacts = func(context.Context) ([]string, error) { calls++; panic(marker) }
	}
	got := catchReadPanic(func() { _ = callReaderLifecycle(t, r, context.Background(), method) })
	if site == "Anchor" {
		calls = r.journal.anchor.(*readerPanicAnchor).calls
	}
	if site == "Check" {
		calls = lease.checks
	}
	if got != marker || calls != 1 || lease.closes != 1 || a.cas != 0 {
		t.Fatalf("reader panic/lease/zero-write: identity=%t calls=%d closes=%d CAS=%d", got == marker, calls, lease.closes, a.cas)
	}
	t.Logf("method=%s site=%s original_panic_preserved=true calls=%d lease_closes=%d CAS=%d", method, site, calls, lease.closes, a.cas)
}
func testReadErrorJoin(t *testing.T, method, mode string) {
	r, a, lease := readerLifecycleFixture(t)
	cause := errors.New("synthetic record read error")
	closeCause := errors.New("synthetic reader Close error")
	readFailure := mode == "ReadError" || mode == "ReadAndCloseError"
	closeFailure := mode == "CloseError" || mode == "ReadAndCloseError"
	if readFailure {
		r.records = func(context.Context) ([]diskRecord, error) { return nil, cause }
	}
	if closeFailure {
		lease.closeErr = closeCause
	}
	e := callReaderLifecycle(t, r, context.Background(), method)
	if readFailure && (!errors.Is(e, cause) || !errors.Is(e, ErrAdmissionIncomplete)) {
		t.Fatalf("read cause lost: %v", e)
	}
	if closeFailure && !errors.Is(e, closeCause) {
		t.Fatalf("Close cause lost: %v", e)
	}
	if !readFailure && !closeFailure && e != nil {
		t.Fatal(e)
	}
	if lease.closes != 1 || a.cas != 0 {
		t.Fatalf("reader release/zero effects: close=%d CAS=%d", lease.closes, a.cas)
	}
}
func testReadPortJoin(t *testing.T, method string) {
	r, a, lease := readerLifecycleFixture(t)
	read := r.records
	entered, release := make(chan struct{}), make(chan struct{})
	r.records = func(ctx context.Context) ([]diskRecord, error) { close(entered); <-release; return read(ctx) }
	lease.closeErr = errors.New("synthetic Close error after cancelled read")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- callReaderLifecycle(t, r, ctx, method) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("reader port not entered")
	}
	cancel()
	select {
	case e := <-done:
		t.Errorf("read detached before actual port return: %v", e)
	default:
	}
	if lease.closes != 0 {
		t.Error("read lease closed before actual port join")
	}
	close(release)
	select {
	case e := <-done:
		if !errors.Is(e, context.Canceled) || !errors.Is(e, lease.closeErr) {
			t.Fatalf("cancel/Close causes lost: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("read did not actually join")
	}
	if lease.closes != 1 || a.cas != 0 {
		t.Fatalf("joined read release/effects: closes=%d CAS=%d", lease.closes, a.cas)
	}
}

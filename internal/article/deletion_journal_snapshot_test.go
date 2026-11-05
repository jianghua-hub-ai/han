package article

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"han/internal/recovery"
)

type anchorCoreCAS struct {
	mu       sync.Mutex
	value    recovery.AnchorSnapshot
	revision int
	calls    []string
}

func testAnchorC1Snapshots(t *testing.T) {
	t.Run("GenerationContextAndLocalSupply", testAnchorC1GenerationSnapshot)
	t.Run("ColdNormalNoRepairAndNewRuntimeRebuild", testAnchorC1NormalRebuild)
	t.Run("SingleBindAndUnboundRefuse", testAnchorC1SingleBind)
	t.Run("IndependentPostproof", testAnchorC1Postproof)
	t.Run("PostproofReadCancelTwoStopJoin", testAnchorC1PostproofJoin)
}

func testAnchorC1PostproofJoin(t *testing.T) {
	w := newAnchorC1World(t, "committed")
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	defer w.h.releaseNow()
	returned := false
	w.snapshots.hook = func(ctx context.Context) { close(entered); <-release; returned = true }
	install := startAnchorJoinedCall(t, w.h, w.run)
	select {
	case <-entered:
	case <-install.done:
		t.Fatal("Install did not reach real postproof reader")
	case <-time.After(3 * time.Second):
		t.Fatal("postproof read not reached")
	}
	stop1 := startAnchorJoinedCall(t, w.h, func() error { return w.runtime.Stop(context.Background()) })
	stop2 := startAnchorJoinedCall(t, w.h, func() error { return w.runtime.Stop(context.Background()) })
	<-w.pc.Done()
	assertAnchorPending(t, install, stop1, stop2)
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 0)
	once.Do(func() { close(release) })
	install.join()
	if !returned || install.err == nil {
		t.Fatal("postproof read did not join/cancel")
	}
	// The action is now over. This explicit component owner closes its idle handle;
	// production idle-handle/deployment/pool ordering remains S4.16, not this test.
	if e := w.control.Close(); e != nil {
		t.Fatal(e)
	}
	stop1.join()
	stop2.join()
	assertAnchorTerminal(t, w.h)
}

func testAnchorC1GenerationSnapshot(t *testing.T) {
	w := newAnchorC1World(t, "stage")
	w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
		if w.snapshots.reads != 0 {
			return errors.New("preproof used unavailable snapshot")
		}
		if e := w.coordinate(ctx, p); e != nil {
			return e
		}
		n := anchorC1CASCount(w.a)
		syncs := w.syncs
		s, e := w.control.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
		if e != nil || s.Head != w.local.Head || s.Revision != w.local.Revision {
			return errors.Join(errors.New("local pure observation before acceptance"), e)
		}
		r, e := w.control.FindByOperation(context.Background(), anchorRequest(1).DeletionOperationID)
		if e != nil || r.DeletionOperationID != anchorRequest(1).DeletionOperationID {
			return errors.Join(errors.New("pure committed operation read"), e)
		}
		if anchorC1CASCount(w.a) != n || w.syncs != syncs {
			return errors.New("pure local read wrote")
		}
		return nil
	}
	post := 0
	w.snapshots.hook = func(ctx context.Context) {
		post++
		if ctx == w.pc {
			t.Error("postproof was passed executor pc")
		}
		if _, _, e := w.runtime.CheckArticleCoordination(ctx, w.permit, w.o.InstallationID, w.o.JournalID, w.o.Epoch); e == nil {
			t.Error("generation ctx gained coordination")
		}
	}
	if e := w.run(); e != nil {
		t.Fatal(e)
	}
	if post != 1 || w.locks != 1 {
		t.Fatal("postproof not actual same-handle read")
	}
}

func testAnchorC1NormalRebuild(t *testing.T) {
	w := newAnchorC1World(t, "stage")
	if j, e := newAnchoredDeletionJournal(context.Background(), w.o, w.k, w.normalMedia); e == nil {
		_ = j.Close()
		t.Fatal("cold normal accepted pending")
	}
	if w.syncs != 0 || anchorC1CASCount(w.a) != 0 || w.locks != 0 {
		t.Fatal("cold normal repaired")
	}
	if e := w.run(); e != nil {
		t.Fatal(e)
	}
	if e := w.control.Close(); e != nil {
		t.Fatal(e)
	}
	assertAnchorTerminal(t, w.h)
	if e := w.runtime.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 1)
	j, r := nextAnchorC1Normal(t, w)
	n := anchorC1CASCount(w.a)
	syncs := w.syncs
	if e := r.RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
	lease, e := r.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	s, e := j.ReadVerifiedSnapshot(lease.Context(), ArticleDeletionNamespace)
	if e != nil || s.Head != w.local.Head || len(s.Prefixes) != 2 {
		t.Fatalf("new runtime proof: %+v %v", s, e)
	}
	if n != anchorC1CASCount(w.a) || syncs != w.syncs {
		t.Fatal("normal reconstruction repaired")
	}
}

func nextAnchorC1Normal(t *testing.T, w *anchorC1World) (*anchoredArticleJournal, recovery.Runtime) {
	t.Helper()
	next := &anchorRuntimeWorld{installation: w.installation, join: make(chan struct{})}
	r, e := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: next, Journal: next, ArticleSnapshots: next, Closures: next, Genesis: next, Exclusive: next, Lifecycle: next})
	if e != nil {
		t.Fatal(e)
	}
	o := w.o
	o.Runtime = r
	j, e := newAnchoredDeletionJournal(context.Background(), o, w.k, w.normalMedia)
	if e != nil {
		t.Fatal(e)
	}
	next.journal = j
	t.Cleanup(func() {
		if e := r.Stop(context.Background()); e != nil {
			t.Error(e)
		}
		if e := j.Close(); e != nil {
			t.Error(e)
		}
	})
	return j, r
}

func testAnchorC1SingleBind(t *testing.T) {
	w := newAnchorC1World(t, "committed")
	empty := &anchorC1RuntimeForward{}
	if e := empty.CheckJournalUse(context.Background(), anchorRequest(1).DeletionOperationID, recovery.JournalUseNewAppend); e == nil {
		t.Fatal("unbound runtime opened")
	}
	if _, e := w.snapshots.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace); e == nil {
		t.Fatal("unbound snapshot opened")
	}
	w.snapshots.reads = 0
	if e := w.forward.bind(w.runtime); e == nil {
		t.Fatal("runtime rebind")
	}
	w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
		if e := w.coordinate(ctx, p); e != nil {
			return e
		}
		if e := w.snapshots.bind(w.control); e == nil {
			return errors.New("snapshot rebind")
		}
		return nil
	}
	if e := w.run(); e != nil {
		t.Fatal(e)
	}
}

func testAnchorC1Postproof(t *testing.T) {
	for _, kind := range []string{"no-supply", "different-backend", "tampered-after-local", "store-after-local", "commitment-after-local", "approval-after-local"} {
		t.Run(kind, func(t *testing.T) {
			state := "stage"
			if kind == "no-supply" {
				state = "committed"
			}
			w := newAnchorC1World(t, state)
			if kind == "different-backend" {
				w.o.Anchor = &anchorCoreCAS{value: copyDeletionAnchor(w.a.value), revision: w.a.revision}
			}
			if kind == "no-supply" {
				w.execute = func(context.Context, recovery.PhasePermit) error { return nil }
			}
			w.snapshots.hook = func(context.Context) {
				switch kind {
				case "tampered-after-local":
					raw, e := os.ReadFile(filepath.Join(w.o.Directory, deletionRecordName(1)))
					if e != nil {
						t.Error(e)
					}
					if e := os.WriteFile(filepath.Join(w.o.Directory, deletionRecordName(1)), append(raw, ' '), w.o.FileMode); e != nil {
						t.Error(e)
					}
				case "store-after-local":
					w.installation.TargetDigest = recovery.SHA256Digest([]byte("drifted-store"))
				case "commitment-after-local":
					w.commitment.ManifestDigest = recovery.SHA256Digest([]byte("drifted-commitment"))
				case "approval-after-local":
					w.approved.SourceApprovalRef += "-drift"
				}
			}
			if e := w.run(); e == nil {
				t.Fatal("executor success mistaken for independent proof")
			}
			if kind == "no-supply" && w.snapshots.reads != 1 {
				t.Fatal("no-supply did not reach actual independent snapshot postproof")
			}
			if w.control != nil {
				if e := w.control.Close(); e != nil {
					t.Fatal(e)
				}
			}
			if lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
				lease.Release()
				t.Fatal("postproof error opened public work")
			}
		})
	}
}

func (a *anchorCoreCAS) Load(ctx context.Context, namespace string) (recovery.AnchorSnapshot, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "load")
	if err := ctx.Err(); err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	if namespace != ArticleDeletionNamespace {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionBinding
	}
	v := a.value
	if v.Pending != nil {
		p := *v.Pending
		v.Pending = &p
	}
	return v, fmt.Sprintf("revision-%d", a.revision), nil
}

func (a *anchorCoreCAS) CompareAndSwap(ctx context.Context, namespace, revision string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, "cas")
	if err := ctx.Err(); err != nil {
		return recovery.AnchorSnapshot{}, "", err
	}
	if namespace != ArticleDeletionNamespace || revision != fmt.Sprintf("revision-%d", a.revision) {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorConflict
	}
	a.value = next
	a.revision++
	return next, fmt.Sprintf("revision-%d", a.revision), nil
}

type anchorCoreLease struct{ closes int }

func (l *anchorCoreLease) Check(ctx context.Context) error { return ctx.Err() }
func (l *anchorCoreLease) Close() error                    { l.closes++; return nil }
func (l *anchorCoreLease) acquire(context.Context, bool) (deletionNamespaceLease, error) {
	return newAnchorCoreNamespaceLease(l), nil
}

// Legacy mechanics fixture: no namespace file is owned. Each acquisition still
// has its own Close attempt and synchronous completion event.
type anchorCoreNamespaceLease struct {
	counter *anchorCoreLease
	once    sync.Once
	done    chan struct{}
}

func newAnchorCoreNamespaceLease(counter *anchorCoreLease) *anchorCoreNamespaceLease {
	return &anchorCoreNamespaceLease{counter: counter, done: make(chan struct{})}
}
func (l *anchorCoreNamespaceLease) Check(ctx context.Context) error { return ctx.Err() }
func (l *anchorCoreNamespaceLease) Close() error {
	l.once.Do(func() { l.counter.Close(); close(l.done) })
	return nil
}
func (l *anchorCoreNamespaceLease) WaitReleased(ctx context.Context) error {
	<-l.done
	return ctx.Err()
}

func anchorCoreFixture(t *testing.T) (AnchoredJournalOptions, DeletionJournalKeyring, deletionAnchorMedia, *anchorCoreCAS, *anchorCoreLease) {
	t.Helper()
	dir, err := os.MkdirTemp("", "han-article-anchor-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	a := &anchorCoreCAS{value: recovery.AnchorSnapshot{JournalID: "article-core", Namespace: ArticleDeletionNamespace, Epoch: 2}, revision: 1}
	l := &anchorCoreLease{}
	m := deletionAnchorMedia{
		lock:  l.acquire,
		check: func(ctx context.Context) error { return ctx.Err() },
		names: func(ctx context.Context) ([]string, error) {
			entries, e := os.ReadDir(dir)
			var names []string
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			return names, e
		},
		open: func(ctx context.Context, name string) (*os.File, error) { return os.Open(filepath.Join(dir, name)) },
		create: func(ctx context.Context, name string, mode fs.FileMode) (*os.File, error) {
			return os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		},
		write:    writeDeletionAnchorBytes,
		syncFile: func(ctx context.Context, f *os.File) error { return f.Sync() },
		syncDir:  func(ctx context.Context) error { return ctx.Err() },
		link: func(ctx context.Context, from, to string) error {
			return os.Link(filepath.Join(dir, from), filepath.Join(dir, to))
		},
		unlink: func(ctx context.Context, from, to string, raw []byte) error {
			a, e := os.Stat(filepath.Join(dir, from))
			if e != nil {
				return e
			}
			b, e := os.Stat(filepath.Join(dir, to))
			if e != nil {
				return e
			}
			got, e := os.ReadFile(filepath.Join(dir, from))
			if e != nil || !os.SameFile(a, b) || !bytes.Equal(got, raw) {
				return errors.Join(ErrDeletionJournalCorrupt, e)
			}
			return os.Remove(filepath.Join(dir, from))
		},
	}
	k := DeletionJournalKeyring{Active: DeletionJournalKey{ID: "synthetic-key", Version: "v1", Material: []byte("synthetic-article-core-key-32bytes")}}
	o := AnchoredJournalOptions{Directory: dir, FileMode: 0o600, InstallationID: "synthetic-installation", JournalID: a.value.JournalID, Epoch: a.value.Epoch, Anchor: a, Runtime: &anchorCoreRuntime{}, ReadMaxBytes: 1 << 20, ReadMaxRecords: 100}
	return o, k, m, a, l
}

func TestAnchoredArticleSnapshot(t *testing.T) {
	t.Run("RealInstallColdOpenAndPostproof", testAnchorC1Snapshots)
	t.Run("ReaderFaultStates", testAnchorReaderFaultStates)
	t.Run("ResourceFinalization", testAnchorReaderFinalization)
	t.Run("ContextLockAndBudget", testAnchorReaderContext)
	t.Run("PureReaderNoRepairAndPrefix", func(t *testing.T) {
		t.Run("EachNormalReaderZeroWrites", testAnchorPureReaders)
		t.Run("CommittedNewHeadWithLegalOldPrefixRejected", testAnchorPrefixRollback)
		t.Run("CommittedBeforeRuntimeAdmission", func(t *testing.T) {
			o, k, m, a, l := anchorCoreFixture(t)
			j, e := newAnchoredDeletionJournal(context.Background(), o, k, m)
			if e != nil {
				t.Fatalf("installed committed constructor must verify without work lease: %v", e)
			}
			defer j.Close()
			s, e := j.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
			if e != nil || s.Head != (recovery.JournalHead{}) || len(s.Prefixes) != 1 || s.Revision != "revision-1" {
				t.Fatalf("committed snapshot: %+v %v", s, e)
			}
			for _, call := range a.calls {
				if call == "cas" {
					t.Fatal("pure reader wrote anchor")
				}
			}
			if l.closes != 2 {
				t.Fatalf("constructor and reader lease closes=%d", l.closes)
			}
		})
	})
}

// Fixture bytes deliberately encode the original wire order independently of
// the journal's serializer, hasher and HMAC helpers.
func anchorFixtureBytes(r DeletionTombstoneRequest, sequence int, previous string, key DeletionJournalKey) []byte {
	payload := fmt.Sprintf(`{"namespace":"article-deletion/v1","sequence":%d,"article_id":"%s","article_version":%d,"deletion_operation_id":"%s","deleted_at":"%s","key_id":"%s","key_version":"%s","stage":"tombstoned","previous_record_hash":"%s"}`,
		sequence, r.ArticleID, r.ArticleVersion, r.DeletionOperationID, r.DeletedAt.Format(time.RFC3339Nano), key.ID, key.Version, previous)
	mac := hmac.New(sha256.New, key.Material)
	mac.Write([]byte("han/article-deletion/v1/tombstone\x00"))
	mac.Write([]byte(payload))
	return []byte(payload[:len(payload)-1] + `,"record_hmac":"hmac-sha256:` + hex.EncodeToString(mac.Sum(nil)) + `"}`)
}

func anchorInstallFixture(t *testing.T, o AnchoredJournalOptions, k DeletionJournalKeyring, a *anchorCoreCAS, count int) {
	t.Helper()
	previous := ""
	for n := 1; n <= count; n++ {
		raw := anchorFixtureBytes(anchorRequest(n), n, previous, k.Active)
		if e := os.WriteFile(filepath.Join(o.Directory, fmt.Sprintf("%020d.json", n)), raw, o.FileMode); e != nil {
			t.Fatal(e)
		}
		previous = recovery.SHA256Digest(raw)
	}
	a.value.Committed = recovery.JournalHead{Sequence: uint64(count), Hash: previous}
}

func testAnchorPureReaders(t *testing.T) {
	for _, reader := range []string{"constructor", "snapshot", "all", "find", "repeat"} {
		t.Run(reader, func(t *testing.T) {
			o, k, m, a, _ := anchorCoreFixture(t)
			anchorInstallFixture(t, o, k, a, 1)
			writes := 0
			fail := func() error { writes++; return errors.New("forbidden reader write") }
			m.create = func(context.Context, string, fs.FileMode) (*os.File, error) { return nil, fail() }
			m.write = func(context.Context, *os.File, []byte) error { return fail() }
			m.syncFile = func(context.Context, *os.File) error { return fail() }
			m.syncDir = func(context.Context) error { return fail() }
			m.link = func(context.Context, string, string) error { return fail() }
			m.unlink = func(context.Context, string, string, []byte) error { return fail() }
			j, e := newAnchoredDeletionJournal(context.Background(), o, k, m)
			if e != nil {
				t.Fatal(e)
			}
			defer j.Close()
			switch reader {
			case "snapshot":
				s, err := j.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
				e = err
				if s.Head != a.value.Committed || len(s.Prefixes) != 2 || s.Prefixes[1] != a.value.Committed || len(s.Entries) != 0 {
					t.Errorf("snapshot=%+v", s)
				}
			case "all":
				_, e = j.ReadAllVerified(context.Background())
			case "find":
				_, e = j.FindByOperation(context.Background(), anchorRequest(1).DeletionOperationID)
			case "repeat":
				_, e = j.Append(context.Background(), anchorRequest(1))
			}
			if e != nil {
				t.Fatal(e)
			}
			if writes != 0 {
				t.Fatalf("writes=%d", writes)
			}
			for _, call := range a.calls {
				if call == "cas" {
					t.Fatal("reader CAS")
				}
			}
		})
	}
}

func testAnchorPrefixRollback(t *testing.T) {
	o, k, m, a, _ := anchorCoreFixture(t)
	anchorInstallFixture(t, o, k, a, 1)
	old := a.value.Committed
	raw := anchorFixtureBytes(anchorRequest(2), 2, old.Hash, k.Active)
	a.value.Committed = recovery.JournalHead{Sequence: 2, Hash: recovery.SHA256Digest(raw)}
	if _, e := newAnchoredDeletionJournal(context.Background(), o, k, m); !errors.Is(e, recovery.ErrAnchorRollback) {
		t.Fatalf("legal old prefix accepted: %v", e)
	}
}

func testAnchorReaderFaultStates(t *testing.T) {
	for _, kind := range []string{"pending", "orphan-stage", "uncommitted-tail", "wrong-namespace", "wrong-id", "wrong-epoch", "bytes-budget", "record-budget", "missing-retained-key", "tampered", "trailing-bytes", "noncanonical-bytes", "directory-record", "file-identity-drift", "revision-ABA"} {
		t.Run(kind, func(t *testing.T) {
			o, k, m, a, _ := anchorCoreFixture(t)
			anchorInstallFixture(t, o, k, a, 1)
			configureAnchorReaderFault(t, kind, &o, &k, &m, a)
			if _, e := newAnchoredDeletionJournal(context.Background(), o, k, m); e == nil {
				t.Fatalf("reader accepted %s", kind)
			}
			for _, call := range a.calls {
				if call == "cas" {
					t.Fatal("reader repaired anchor")
				}
			}
		})
	}
}

type anchorLoadPort struct {
	inner recovery.JournalHeadAnchor
	load  func(context.Context, string) (recovery.AnchorSnapshot, string, error)
}

func (p anchorLoadPort) Load(ctx context.Context, ns string) (recovery.AnchorSnapshot, string, error) {
	return p.load(ctx, ns)
}
func (p anchorLoadPort) CompareAndSwap(ctx context.Context, ns, r string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	return p.inner.CompareAndSwap(ctx, ns, r, next)
}

func configureAnchorReaderFault(t *testing.T, kind string, o *AnchoredJournalOptions, k *DeletionJournalKeyring, m *deletionAnchorMedia, a *anchorCoreCAS) {
	t.Helper()
	switch kind {
	case "pending":
		a.value.Pending = &recovery.AnchorPending{OldHead: a.value.Committed, NewHead: recovery.JournalHead{Sequence: 2, Hash: recovery.SHA256Digest([]byte("pending"))}, OperationID: "pending"}
	case "orphan-stage":
		if e := os.WriteFile(filepath.Join(o.Directory, ".article-deletion.stage-synthetic"), []byte("candidate"), o.FileMode); e != nil {
			t.Fatal(e)
		}
	case "uncommitted-tail":
		a.value.Committed = recovery.JournalHead{}
	case "wrong-namespace":
		a.value.Namespace = recovery.RecoveryNamespaceV1
	case "wrong-id":
		a.value.JournalID = "other"
	case "wrong-epoch":
		a.value.Epoch++
	case "bytes-budget":
		o.ReadMaxBytes = 1
	case "record-budget":
		anchorInstallFixture(t, *o, *k, a, 2)
		o.ReadMaxRecords = 1
	case "missing-retained-key":
		k.Active = DeletionJournalKey{ID: "K2", Version: "v2", Material: []byte("synthetic-other-key")}
	case "tampered", "trailing-bytes", "noncanonical-bytes":
		mutateAnchorFixtureRecord(t, kind, o)
	case "directory-record":
		m.open = func(context.Context, string) (*os.File, error) { return os.Open(o.Directory) }
	case "file-identity-drift":
		alternate, e := os.CreateTemp("", "han-article-alternate-")
		if e != nil {
			t.Fatal(e)
		}
		if _, e := alternate.Write(anchorFixtureBytes(anchorRequest(1), 1, "", k.Active)); e != nil {
			t.Fatal(e)
		}
		if e := alternate.Close(); e != nil {
			t.Fatal(e)
		}
		m.open = func(context.Context, string) (*os.File, error) { return os.Open(alternate.Name()) }
	case "revision-ABA":
		n := 0
		o.Anchor = anchorLoadPort{inner: a, load: func(ctx context.Context, ns string) (recovery.AnchorSnapshot, string, error) {
			s, r, e := a.Load(ctx, ns)
			n++
			return s, fmt.Sprintf("%s-%d", r, n), e
		}}
	}
}

func testAnchorReaderFinalization(t *testing.T) {
	t.Run("CloseCauseJoinsOriginalError", testAnchorReaderCloseCause)
	for _, point := range []string{"anchor-load", "namespace-check", "file-open"} {
		t.Run("Panic/"+point, func(t *testing.T) { testAnchorReaderPanic(t, point) })
	}
	t.Run("FileAcquiredWithErrorClosed", testAnchorOpenWithError)
}

func testAnchorReaderCloseCause(t *testing.T) {
	o, k, m, _, _ := anchorCoreFixture(t)
	j, e := newAnchoredDeletionJournal(context.Background(), o, k, m)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	loadErr := errors.New("synthetic load error")
	closeErr := errors.New("synthetic close error")
	j.options.Anchor = anchorLoadPort{load: func(context.Context, string) (recovery.AnchorSnapshot, string, error) {
		return recovery.AnchorSnapshot{}, "", loadErr
	}}
	closed := 0
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) {
		return anchorLeaseFuncs{close: func() error { closed++; return closeErr }}, nil
	}
	s, e := j.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
	if !errors.Is(e, loadErr) || !errors.Is(e, closeErr) || closed != 1 || s.Revision != "" {
		t.Fatalf("close join result=%+v %v closed=%d", s, e, closed)
	}
}

func testAnchorOpenWithError(t *testing.T) {
	o, k, m, a, _ := anchorCoreFixture(t)
	anchorInstallFixture(t, o, k, a, 1)
	var opened *os.File
	m.open = func(ctx context.Context, n string) (*os.File, error) {
		f, e := os.Open(filepath.Join(o.Directory, n))
		opened = f
		return f, errors.Join(e, errors.New("synthetic open ACK"))
	}
	if _, e := newAnchoredDeletionJournal(context.Background(), o, k, m); e == nil {
		t.Fatal("open error ignored")
	}
	if opened.Fd() != ^uintptr(0) {
		t.Fatal("open error leaked handle")
	}
}

func testAnchorReaderPanic(t *testing.T, point string) {
	o, k, m, _, l := anchorCoreFixture(t)
	j, e := newAnchoredDeletionJournal(context.Background(), o, k, m)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	closes := l.closes
	sentinel := &struct{ point string }{point}
	switch point {
	case "anchor-load":
		j.options.Anchor = anchorLoadPort{load: func(context.Context, string) (recovery.AnchorSnapshot, string, error) { panic(sentinel) }}
	case "namespace-check":
		j.media.check = func(context.Context) error { panic(sentinel) }
	case "file-open":
		a := o.Anchor.(*anchorCoreCAS)
		anchorInstallFixture(t, o, k, a, 1)
		j.media.open = func(context.Context, string) (*os.File, error) { panic(sentinel) }
	}
	func() {
		defer func() {
			if got := recover(); got != sentinel {
				t.Errorf("panic replaced: %v", got)
			}
		}()
		j.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
	}()
	if l.closes != closes+1 {
		t.Fatalf("namespace lease not closed on panic: %d -> %d", closes, l.closes)
	}
	select {
	case j.gate <- struct{}{}:
		<-j.gate
	default:
		t.Fatal("mutex retained on panic")
	}
}

type anchorLeaseFuncs struct {
	check func(context.Context) error
	close func() error
	wait  func(context.Context) error
}

func (l anchorLeaseFuncs) Check(ctx context.Context) error {
	if l.check != nil {
		return l.check(ctx)
	}
	return ctx.Err()
}
func (l anchorLeaseFuncs) Close() error { return l.close() }
func (l anchorLeaseFuncs) WaitReleased(ctx context.Context) error {
	if l.wait != nil {
		return l.wait(ctx)
	}
	// Explicit resource-free fault fixture; real-resource wrappers supply wait.
	return ctx.Err()
}

func testAnchorReaderContext(t *testing.T) {
	o, k, m, _, _ := anchorCoreFixture(t)
	j, e := newAnchoredDeletionJournal(context.Background(), o, k, m)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	if _, e := j.ReadVerifiedSnapshot(nil, ArticleDeletionNamespace); e == nil {
		t.Fatal("nil ctx accepted")
	}
	if _, e := j.ReadVerifiedSnapshot(context.Background(), recovery.RecoveryNamespaceV1); e == nil {
		t.Fatal("foreign namespace accepted")
	}
	j.gate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, err := j.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace)
	<-j.gate
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("mutex wait ignored ctx: %v", err)
	}
}

func mutateAnchorFixtureRecord(t *testing.T, kind string, o *AnchoredJournalOptions) {
	p := filepath.Join(o.Directory, "00000000000000000001.json")
	raw, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	if kind == "tampered" {
		raw[len(raw)-3] = '0'
	} else if kind == "trailing-bytes" {
		raw = append(raw, []byte("{}")...)
	} else {
		raw = append(raw, '\n')
	}
	if e := os.WriteFile(p, raw, o.FileMode); e != nil {
		t.Fatal(e)
	}
}

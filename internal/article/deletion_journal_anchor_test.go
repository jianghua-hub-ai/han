package article

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/recovery"
)

// These forwards bind once to actual objects; they do not mint permissions.
type anchorC1RuntimeForward struct {
	mu     sync.Mutex
	target ArticleAppendRuntime
}

func (f *anchorC1RuntimeForward) bind(r ArticleAppendRuntime) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.target != nil || r == nil {
		return recovery.ErrAdmissionBinding
	}
	f.target = r
	return nil
}
func (f *anchorC1RuntimeForward) get() ArticleAppendRuntime {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.target
}
func (f *anchorC1RuntimeForward) CheckJournalUse(ctx context.Context, op, use string) error {
	if r := f.get(); r != nil {
		return r.CheckJournalUse(ctx, op, use)
	}
	return recovery.ErrAdmissionClosed
}
func (f *anchorC1RuntimeForward) CheckArticleCoordination(ctx context.Context, p recovery.PhasePermit, i, j string, e uint64) (recovery.AnchorSnapshot, string, error) {
	if r := f.get(); r != nil {
		return r.CheckArticleCoordination(ctx, p, i, j, e)
	}
	return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionClosed
}
func (f *anchorC1RuntimeForward) PauseArticleAppend(ctx context.Context, op string) (recovery.AppendBarrier, error) {
	if r := f.get(); r != nil {
		return r.PauseArticleAppend(ctx, op)
	}
	return recovery.AppendBarrier{}, recovery.ErrAdmissionClosed
}
func (f *anchorC1RuntimeForward) CompleteArticleAppend(ctx context.Context, b recovery.AppendBarrier, a recovery.VerifiedHeadAdvance) error {
	if r := f.get(); r != nil {
		return r.CompleteArticleAppend(ctx, b, a)
	}
	return recovery.ErrAdmissionClosed
}
func (f *anchorC1RuntimeForward) FaultArticleAppend(ctx context.Context, b recovery.AppendBarrier, e error) error {
	if r := f.get(); r != nil {
		return r.FaultArticleAppend(ctx, b, e)
	}
	return recovery.ErrAdmissionClosed
}

type anchorC1Snapshots struct {
	mu     sync.Mutex
	target *anchoredArticleJournal
	reads  int
	hook   func(context.Context)
}

func (s *anchorC1Snapshots) bind(j *anchoredArticleJournal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j == nil || s.target != nil {
		return recovery.ErrAdmissionBinding
	}
	s.target = j
	return nil
}
func (s *anchorC1Snapshots) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	s.mu.Lock()
	j := s.target
	s.reads++
	s.mu.Unlock()
	if j == nil {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionClosed
	}
	if s.hook != nil {
		s.hook(ctx)
	}
	return j.ReadVerifiedSnapshot(ctx, ns)
}

// Outside ports below are explicitly synthetic. The producer/checker, article
// control core, file parser, CAS progression and postproof are the actual code.
type anchorC1World struct {
	*anchorRuntimeWorld
	o                  AnchoredJournalOptions
	k                  DeletionJournalKeyring
	m                  deletionAnchorMedia
	normalMedia        deletionAnchorMedia
	a                  *anchorCoreCAS
	h                  *anchorHeldNamespace
	manifest, approved recovery.InstallationManifest
	commitment         recovery.InstallationCommitment
	snapshots          *anchorC1Snapshots
	forward            *anchorC1RuntimeForward
	control            *anchoredArticleJournal
	execute            func(context.Context, recovery.PhasePermit) error
	pc                 context.Context
	permit             recovery.PhasePermit
	local              recovery.VerifiedJournalSnapshot
	locks, syncs       int
	expectedClose      error
}

func anchorC1ManifestDigest(m recovery.InstallationManifest) string {
	object := func(names string, values ...any) json.RawMessage {
		var b bytes.Buffer
		b.WriteByte('{')
		for n, k := range strings.Fields(names) {
			if n != 0 {
				b.WriteByte(',')
			}
			key, _ := json.Marshal(k)
			b.Write(key)
			b.WriteByte(':')
			v := values[n]
			if tm, ok := v.(time.Time); ok {
				v = tm.UTC().Format("2006-01-02T15:04:05.000000Z")
			}
			raw, _ := json.Marshal(v)
			b.Write(raw)
		}
		b.WriteByte('}')
		return b.Bytes()
	}
	ns := func(a recovery.AnchorSnapshot) json.RawMessage {
		return object("namespace journal_id epoch head_sequence head_hash", a.Namespace, a.JournalID, a.Epoch, a.Committed.Sequence, a.Committed.Hash)
	}
	return recovery.SHA256Digest(object("version kind deployment_id installation_id target_digest source_approval_ref source_approval_digest recovery article issued_at", m.Version, m.Kind, m.DeploymentID, m.InstallationID, m.TargetDigest, m.SourceApprovalRef, m.SourceApprovalDigest, ns(m.Recovery), ns(m.Article), m.IssuedAt))
}

func newAnchorC1World(t *testing.T, state string) *anchorC1World {
	t.Helper()
	o, k, m, a, _ := anchorCoreFixture(t)
	h := newAnchorHeldNamespace(t)
	h.closeErr = nil
	w := &anchorC1World{o: o, k: k, m: m, normalMedia: m, a: a, h: h, snapshots: &anchorC1Snapshots{}, forward: &anchorC1RuntimeForward{}}
	w.anchorRuntimeWorld = &anchorRuntimeWorld{join: make(chan struct{})}
	initAnchorC1Binding(w)
	w.seed(t, state)
	configureAnchorC1Media(w)
	bindAnchorC1Runtime(t, w)
	return w
}

func initAnchorC1Binding(w *anchorC1World) {
	w.manifest = recovery.InstallationManifest{Version: recovery.InstallationVersion, Kind: "genesis", DeploymentID: "synthetic-deployment", InstallationID: w.o.InstallationID,
		TargetDigest: recovery.SHA256Digest([]byte("synthetic-target")), SourceApprovalRef: "synthetic-approved-source", SourceApprovalDigest: recovery.SHA256Digest([]byte("synthetic-independent-approval")),
		Article: w.a.value, Recovery: recovery.AnchorSnapshot{JournalID: "synthetic-recovery", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1}, IssuedAt: anchorRequest(1).DeletedAt}
	w.approved = w.manifest
	w.commitment = recovery.InstallationCommitment{Manifest: w.manifest, ManifestDigest: anchorC1ManifestDigest(w.manifest), Phase: "bound"}
	w.installation = recovery.InstallationBinding{Version: w.manifest.Version, DeploymentID: w.manifest.DeploymentID, InstallationID: w.manifest.InstallationID, TargetDigest: w.manifest.TargetDigest,
		CommitmentRef: "recovery/v1/installation/" + w.manifest.DeploymentID, CommitmentDigest: w.commitment.ManifestDigest, InstalledAt: w.manifest.IssuedAt, RecoveryInitial: w.manifest.Recovery, ArticleInitial: w.manifest.Article}
}

func configureAnchorC1Media(w *anchorC1World) {
	w.m.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
		w.locks++
		if !write || w.locks != 1 {
			return nil, errors.New("nested namespace acquire")
		}
		return w.h, nil
	}
	w.m.syncDir = func(ctx context.Context) error { w.syncs++; return ctx.Err() }
	// Portable synthetic media needs a writable handle for Windows FlushFileBuffers.
	// The production native wrapper stays unsupported here; there is no fallback.
	w.m.open = func(ctx context.Context, name string) (*os.File, error) {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		return os.OpenFile(filepath.Join(w.o.Directory, name), os.O_RDWR, 0)
	}
}

func bindAnchorC1Runtime(t *testing.T, w *anchorC1World) {
	t.Helper()
	w.o.Runtime = w.forward
	r, e := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: w.anchorRuntimeWorld, Journal: w.anchorRuntimeWorld, ArticleSnapshots: w.snapshots, Closures: w.anchorRuntimeWorld, Genesis: w.anchorRuntimeWorld, Exclusive: w.anchorRuntimeWorld, Lifecycle: w.anchorRuntimeWorld, Commitments: w, InstallationSources: w, Installation: w, ArticleAnchorReader: w.a})
	if e != nil {
		t.Fatal(e)
	}
	w.runtime = r
	if e := w.forward.bind(r); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if w.control != nil {
			if e := w.control.Close(); e != nil && !errors.Is(e, w.expectedClose) {
				t.Errorf("control close/join: %v", e)
			}
		}
		if e := r.Stop(context.Background()); e != nil {
			t.Error(e)
		}
	})
}

func (w *anchorC1World) seed(t *testing.T, state string) {
	t.Helper()
	if state == "committed" {
		return
	}
	j, e := newAnchoredDeletionJournal(context.Background(), w.o, w.k, w.m)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	a, e := j.prepareAppend(anchorRequest(1), recovery.VerifiedJournalSnapshot{Head: w.a.value.Committed, Revision: "revision-1"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	w.a.value = copyDeletionAnchor(a.pending)
	if state == "absent" {
		return
	}
	if e := os.WriteFile(filepath.Join(w.o.Directory, a.stage), a.raw, w.o.FileMode); e != nil {
		t.Fatal(e)
	}
	if state == "published" || state == "published-alias" || state == "committed-advanced" {
		if e := w.m.link(context.Background(), a.stage, deletionRecordName(1)); e != nil {
			t.Fatal(e)
		}
		if state == "published" || state == "committed-advanced" {
			if e := w.m.unlink(context.Background(), a.stage, deletionRecordName(1), a.raw); e != nil {
				t.Fatal(e)
			}
		}
	}
	if state == "committed-advanced" {
		w.a.value.Committed = w.a.value.Pending.NewHead
		w.a.value.Pending = nil
	}
}
func (w *anchorC1World) LoadInstallation(context.Context, string) (recovery.InstallationCommitment, string, error) {
	return w.commitment, "synthetic-bound-1", nil
}
func (*anchorC1World) CompareAndSwapInstallation(context.Context, string, string, recovery.InstallationCommitment) (recovery.InstallationCommitment, string, error) {
	return recovery.InstallationCommitment{}, "", errors.New("unexpected installation write")
}
func (w *anchorC1World) VerifyInstallationSource(ctx context.Context, m recovery.InstallationManifest) error {
	if m != w.approved {
		return recovery.ErrAdmissionBinding
	}
	return ctx.Err()
}
func (w *anchorC1World) Install(ctx context.Context, p recovery.PhasePermit, m recovery.InstallationManifest) (result error) {
	defer func() {
		v := recover()
		if (result != nil || v != nil) && w.control != nil {
			e, cv := closeDeletionControl(w.control)
			result = errors.Join(result, e, deletionNamespacePanicCause(cv))
			if v == nil {
				v = cv
			}
		}
		if v != nil {
			panic(v)
		}
	}()
	if m != w.manifest {
		return recovery.ErrAdmissionBinding
	}
	w.pc, w.permit = ctx, p
	if w.execute != nil {
		return w.execute(ctx, p)
	}
	return w.coordinate(ctx, p)
}
func (w *anchorC1World) coordinate(ctx context.Context, p recovery.PhasePermit) (result error) {
	j, e := newAnchoredDeletionJournalForCoordinate(ctx, p, w.o, w.k, w.m)
	if e != nil {
		return e
	}
	w.control = j
	if e := j.CoordinateAnchor(ctx, p); e != nil {
		return e
	}
	s, e := j.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace)
	if e != nil {
		return e
	}
	w.local = s
	return w.snapshots.bind(j)
}
func (w *anchorC1World) run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return w.runtime.Install(ctx, w.manifest)
}

func testAnchorC1Install(t *testing.T) {
	t.Run("CandidateOpenBoundary", testAnchorC1CandidateOpenBoundary)
	t.Run("ScopeAndOrigin", testAnchorC1Scope)
	t.Run("InvalidMediaRetained", testAnchorC1InvalidMedia)
	t.Run("OriginProgressionAndCopies", testAnchorC1Progression)
	t.Run("UnknownNoReplay", testAnchorC1Unknown)
	t.Run("PublicUnsupported", testAnchorC1PublicPlatform)
	t.Run("InFlightPortsAndTerminalJoin", testAnchorC1Join)
	t.Run("DurableBoundaryCauses", testAnchorC1Boundaries)
	for _, state := range []string{"committed", "committed-advanced", "stage", "published", "published-alias", "absent"} {
		t.Run(state, func(t *testing.T) {
			w := newAnchorC1World(t, state)
			if e := w.run(); e != nil {
				t.Fatalf("actual Install/control core/Coordinate/local supply/postproof: %v", e)
			}
			want := uint64(1)
			if state == "committed" || state == "absent" {
				want = 0
			}
			if w.local.Head.Sequence != want || w.a.value.Pending != nil || w.local.Revision != fmt.Sprintf("revision-%d", w.a.revision) || w.snapshots.reads != 1 || w.locks != 1 {
				t.Fatalf("actual control proof: %+v anchor=%+v reads=%d locks=%d", w.local, w.a.value, w.snapshots.reads, w.locks)
			}
			if strings.HasPrefix(state, "committed") && (w.a.revision != 1 || w.syncs != 0) {
				t.Fatal("committed changed")
			}
			if state == "absent" && w.a.revision == 1 {
				t.Fatal("H0 withdrawal reused revision")
			}
			if e := w.control.CoordinateAnchor(w.pc, w.permit); e == nil {
				t.Fatal("expired grant survived Install")
			}
			for _, kind := range []recovery.WorkKind{recovery.WorkRequest, recovery.WorkArticleDeletion, recovery.WorkPasswordReset, recovery.WorkArticleNotice} {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				lease, e := w.runtime.Acquire(ctx, kind)
				cancel()
				if e == nil {
					lease.Release()
					t.Fatalf("Install opened %s", kind)
				}
			}
		})
	}
}

type anchorC1CandidateOpenProbe struct {
	mode                                string
	opens, syncs, cancelledSyncs, links int
	candidate                           *os.File
	lost                                bool
	cause                               error
}

func testAnchorC1CandidateOpenBoundary(t *testing.T) {
	for _, mode := range []string{"CancelAfterOpen", "NamespaceLossAfterOpen", "Healthy"} {
		t.Run(mode, func(t *testing.T) { testAnchorC1CandidateOpenCase(t, mode) })
	}
}

func configureAnchorC1CandidateOpen(w *anchorC1World, p *anchorC1CandidateOpenProbe) {
	baseLock, baseLink := w.m.lock, w.m.link
	w.m.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
		lease, err := baseLock(ctx, write)
		if lease == nil {
			return nil, err
		}
		return anchorLeaseFuncs{check: func(ctx context.Context) error {
			if p.lost {
				return p.cause
			}
			return lease.Check(ctx)
		}, close: lease.Close, wait: lease.WaitReleased}, err
	}
	w.m.link = func(ctx context.Context, from, to string) error { p.links++; return baseLink(ctx, from, to) }
	w.execute = func(ctx context.Context, permit recovery.PhasePermit) error {
		child := ctx
		var cancel context.CancelFunc
		if p.mode == "CancelAfterOpen" {
			child, cancel = context.WithCancel(ctx)
			defer cancel()
		}
		p.wrapCandidateMedia(w, cancel)
		return w.coordinate(child, permit)
	}
}

func (p *anchorC1CandidateOpenProbe) wrapCandidateMedia(w *anchorC1World, cancel context.CancelFunc) {
	baseOpen, baseSync := w.m.open, w.m.syncFile
	w.m.open = func(ctx context.Context, name string) (*os.File, error) {
		file, err := baseOpen(ctx, name)
		p.opens++
		if err == nil && file != nil && p.opens == 2 {
			p.candidate = file
			if p.mode == "CancelAfterOpen" {
				cancel()
			}
			if p.mode == "NamespaceLossAfterOpen" {
				p.lost = true
			}
		}
		return file, err
	}
	w.m.syncFile = func(ctx context.Context, file *os.File) error {
		p.syncs++
		if ctx.Err() != nil {
			p.cancelledSyncs++
		}
		return baseSync(ctx, file)
	}
}

func testAnchorC1CandidateOpenCase(t *testing.T, mode string) {
	w := newAnchorC1World(t, "stage")
	p := &anchorC1CandidateOpenProbe{mode: mode, cause: errors.New("synthetic namespace validity lost after candidate Open")}
	if mode == "CancelAfterOpen" {
		p.cause = context.Canceled
	}
	configureAnchorC1CandidateOpen(w, p)
	origin := copyDeletionAnchor(w.a.value)
	before := anchorDirectoryImage(t, w.o.Directory)
	err := w.run()
	// Always finish actual ownership and join before reporting any failed oracle.
	var closeErr error
	if w.control != nil {
		closeErr = w.control.Close()
	}
	stopErr := w.runtime.Stop(context.Background())
	w.h.driver.Wait()
	if closeErr != nil || stopErr != nil {
		t.Errorf("owner cleanup: %v / %v", closeErr, stopErr)
	}
	if p.candidate == nil {
		t.Error("real sync candidate Open was not reached")
	}
	if mode == "Healthy" {
		if err != nil || p.opens < 2 || p.syncs != 1 || p.cancelledSyncs != 0 || p.links != 1 || w.a.value.Pending != nil || w.a.value.Committed.Sequence != 1 || w.snapshots.reads != 1 || w.locks != 1 {
			t.Errorf("healthy path: err=%v opens=%d syncs=%d cancelled=%d links=%d anchor=%+v reads=%d locks=%d", err, p.opens, p.syncs, p.cancelledSyncs, p.links, w.a.value, w.snapshots.reads, w.locks)
		}
	} else {
		if !errors.Is(err, p.cause) {
			t.Errorf("original cause lost: %v", err)
		}
		if p.opens != 2 || p.syncs != 0 || p.cancelledSyncs != 0 || p.links != 0 || w.syncs != 0 || anchorC1CASCount(w.a) != 0 {
			t.Errorf("new durable operation after Open boundary: opens=%d Sync=%d syncAfterCancel=%d Link=%d dirSync=%d CAS=%d", p.opens, p.syncs, p.cancelledSyncs, p.links, w.syncs, anchorC1CASCount(w.a))
		}
		if !sameDeletionAnchor(origin, w.a.value) || fmt.Sprint(before) != fmt.Sprint(anchorDirectoryImage(t, w.o.Directory)) {
			t.Error("failed candidate boundary repaired or removed pending/stage")
		}
	}
	assertAnchorTerminal(t, w.h)
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 1)
}

func testAnchorC1Boundaries(t *testing.T) {
	for _, point := range []string{"file-sync", "link", "dir-before-alias", "alias", "dir-after-alias", "media-check", "namespace-check"} {
		t.Run(point, func(t *testing.T) {
			w := newAnchorC1World(t, "stage")
			sentinel := errors.New("synthetic controlled " + point)
			configureAnchorC1Boundary(w, point, sentinel)
			if e := w.run(); !errors.Is(e, sentinel) {
				t.Fatalf("boundary cause lost: %v", e)
			}
			if anchorC1CASCount(w.a) != 0 || w.a.value.Pending == nil {
				t.Fatal("failed boundary committed or replayed")
			}
			assertAnchorTerminal(t, w.h)
		})
	}
}

func configureAnchorC1Boundary(w *anchorC1World, point string, sentinel error) {
	switch point {
	case "file-sync":
		w.m.syncFile = func(context.Context, *os.File) error { return sentinel }
	case "link":
		w.m.link = func(context.Context, string, string) error { return sentinel }
	case "alias":
		w.m.unlink = func(context.Context, string, string, []byte) error { return sentinel }
	case "dir-before-alias", "dir-after-alias":
		base := w.m.syncDir
		n := 0
		target := 1
		if point == "dir-after-alias" {
			target = 2
		}
		w.m.syncDir = func(ctx context.Context) error {
			n++
			if n == target {
				return sentinel
			}
			return base(ctx)
		}
	case "media-check":
		n := 0
		w.m.check = func(ctx context.Context) error {
			n++
			if n == 3 {
				return sentinel
			}
			return ctx.Err()
		}
	case "namespace-check":
		base := w.m.lock
		n := 0
		w.m.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
			l, e := base(ctx, write)
			return anchorLeaseFuncs{check: func(ctx context.Context) error {
				n++
				if n == 3 {
					return sentinel
				}
				return l.Check(ctx)
			}, close: l.Close, wait: l.WaitReleased}, e
		}
	}
}

type anchorC1BlockedCAS struct {
	recovery.JournalHeadAnchor
	entered  chan struct{}
	release  <-chan struct{}
	returned atomic.Bool
}

func (a *anchorC1BlockedCAS) CompareAndSwap(ctx context.Context, ns, token string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	close(a.entered)
	<-a.release
	defer a.returned.Store(true)
	return a.JournalHeadAnchor.CompareAndSwap(ctx, ns, token, next)
}

func testAnchorC1Join(t *testing.T) {
	t.Run("CASCancelTwoStopAndConcurrentClose", testAnchorC1CASJoin)
	t.Run("AcquireOwnedError", func(t *testing.T) { testAnchorC1OpenTerminal(t, false) })
	t.Run("OpenCheckPanic", func(t *testing.T) { testAnchorC1OpenTerminal(t, true) })
	t.Run("ReaderPanic", testAnchorC1ReaderPanic)
}

func testAnchorC1CASJoin(t *testing.T) {
	w := newAnchorC1World(t, "stage")
	w.h.closeErr = errors.New("synthetic controlled Close unknown")
	w.expectedClose = w.h.closeErr
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	defer w.h.releaseNow()
	port := &anchorC1BlockedCAS{JournalHeadAnchor: w.a, entered: make(chan struct{}), release: release}
	w.o.Anchor = port
	install := startAnchorJoinedCall(t, w.h, w.run)
	select {
	case <-port.entered:
	case <-install.done:
		t.Fatal("Install ended before CAS")
	case <-time.After(3 * time.Second):
		t.Fatal("CAS not reached")
	}
	closing := startAnchorJoinedCall(t, w.h, func() error { return w.control.Close() })
	stops := []*anchorJoinedCall{startAnchorJoinedCall(t, w.h, func() error { return w.runtime.Stop(context.Background()) }), startAnchorJoinedCall(t, w.h, func() error { return w.runtime.Stop(context.Background()) })}
	<-w.pc.Done()
	assertAnchorPending(t, install, closing, stops[0], stops[1])
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 0)
	once.Do(func() { close(release) })
	awaitAnchorNamespaceWait(t, w.h, install)
	if !port.returned.Load() {
		t.Fatal("port not joined before terminal wait")
	}
	if len(w.control.records.active.Material) == 0 {
		t.Fatal("keys erased before namespace terminal")
	}
	assertAnchorC1ClosingRead(t, w)
	assertAnchorPending(t, install, closing, stops[0], stops[1])
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 0)
	w.h.releaseNow()
	install.join()
	closing.join()
	stops[0].join()
	stops[1].join()
	if install.err == nil || !errors.Is(closing.err, w.expectedClose) {
		t.Fatalf("cancel/close causes: %v / %v", install.err, closing.err)
	}
	if stops[0].err != nil || stops[1].err != nil {
		t.Fatal("Stop did not join", stops[0].err, stops[1].err)
	}
	assertAnchorC1Released(t, w)
}

func assertAnchorC1Released(t *testing.T, w *anchorC1World) {
	t.Helper()
	assertAnchorTerminal(t, w.h)
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 1)
	if len(w.control.records.active.Material) != 0 {
		t.Fatal("keys retained after terminal Close")
	}
}

func assertAnchorC1ClosingRead(t *testing.T, w *anchorC1World) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, e := w.control.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace); !errors.Is(e, recovery.ErrAdmissionClosed) {
		t.Fatalf("closing handle admitted a new read: %v", e)
	}
}

func testAnchorC1OpenTerminal(t *testing.T, panics bool) {
	w := newAnchorC1World(t, "committed")
	marker := errors.New("synthetic owned acquisition/check failure")
	w.h.closeErr = errors.New("synthetic Close unknown")
	w.expectedClose = w.h.closeErr
	w.m.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return w.h, marker }
	if panics {
		w.m.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return w.h, nil }
		w.m.check = func(context.Context) error { panic(marker) }
	}
	defer w.h.releaseNow()
	install := startAnchorJoinedCall(t, w.h, w.run)
	awaitAnchorNamespaceWait(t, w.h, install)
	stop := startAnchorJoinedCall(t, w.h, func() error { return w.runtime.Stop(context.Background()) })
	assertAnchorPending(t, install, stop)
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 0)
	w.h.releaseNow()
	install.join()
	stop.join()
	if panics {
		if install.panicValue != marker {
			t.Fatalf("original panic lost: %v", install.panicValue)
		}
	} else if !errors.Is(install.err, marker) {
		t.Fatalf("owned error lost: %v", install.err)
	}
	assertAnchorTerminal(t, w.h)
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 1)
}

func testAnchorC1ReaderPanic(t *testing.T) {
	w := newAnchorC1World(t, "committed")
	marker := errors.New("synthetic control parser panic")
	w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
		if e := w.open(ctx, p); e != nil {
			return e
		}
		w.control.media.names = func(context.Context) ([]string, error) { panic(marker) }
		_, e := w.control.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace)
		return e
	}
	func() {
		defer func() {
			if v := recover(); v != marker {
				t.Fatalf("reader panic lost: %v", v)
			}
		}()
		_ = w.run()
		t.Error("reader panic swallowed")
	}()
	assertAnchorTerminal(t, w.h)
	if e := w.runtime.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 1)
}

func anchorC1CASCount(a *anchorCoreCAS) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for _, c := range a.calls {
		if c == "cas" {
			n++
		}
	}
	return n
}

func (w *anchorC1World) open(ctx context.Context, p recovery.PhasePermit) error {
	j, e := newAnchoredDeletionJournalForCoordinate(ctx, p, w.o, w.k, w.m)
	if e == nil {
		w.control = j
	}
	return e
}

func testAnchorC1Scope(t *testing.T) {
	for _, kind := range []string{"zero-permit", "plain-ctx", "no-budget", "cancelled", "installation", "journal", "epoch", "other-runtime"} {
		t.Run(kind, func(t *testing.T) {
			w := newAnchorC1World(t, "stage")
			w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
				switch kind {
				case "zero-permit":
					p = recovery.PhasePermit{}
				case "plain-ctx":
					c, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					ctx = c
				case "no-budget":
					ctx = context.WithoutCancel(ctx)
				case "cancelled":
					c, cancel := context.WithCancel(ctx)
					cancel()
					ctx = c
				case "installation":
					w.o.InstallationID += "-wrong"
				case "journal":
					w.o.JournalID += "-wrong"
				case "epoch":
					w.o.Epoch++
				case "other-runtime":
					_, other, _, _ := closedActualAnchorCore(t)
					w.o.Runtime = other.runtime
				}
				return w.open(ctx, p)
			}
			if e := w.run(); e == nil {
				t.Fatal("unauthorized open accepted")
			}
			if w.locks != 0 || w.syncs != 0 || anchorC1CASCount(w.a) != 0 {
				t.Fatal("scope rejection performed media I/O")
			}
		})
	}
}

func testAnchorC1InvalidMedia(t *testing.T) {
	for _, kind := range []string{"foreign-stage", "multiple-stage", "malformed", "wrong-hash", "wrong-head", "H2", "committed-orphan", "committed-missing", "different-inode-alias"} {
		t.Run(kind, func(t *testing.T) {
			state := "stage"
			if kind == "different-inode-alias" {
				state = "published"
			}
			if kind == "foreign-stage" {
				state = "absent"
			}
			w := newAnchorC1World(t, state)
			corruptAnchorC1Media(t, w, kind)
			before, e := os.ReadDir(w.o.Directory)
			if e != nil {
				t.Fatal(e)
			}
			if e := w.run(); e == nil {
				t.Fatal("invalid inventory accepted")
			}
			after, e := os.ReadDir(w.o.Directory)
			if e != nil {
				t.Fatal(e)
			}
			if len(before) != len(after) || anchorC1CASCount(w.a) != 0 || w.syncs != 0 {
				t.Fatal("invalid inventory repaired/deleted")
			}
		})
	}
}

func corruptAnchorC1Media(t *testing.T, w *anchorC1World, kind string) {
	t.Helper()
	p := w.a.value.Pending
	stage := ".article-deletion.stage-" + strings.Split(p.OperationID, ":")[1]
	path := filepath.Join(w.o.Directory, stage)
	switch kind {
	case "foreign-stage", "multiple-stage":
		if e := os.WriteFile(filepath.Join(w.o.Directory, ".article-deletion.stage-33333333-3333-4333-8333-333333333333"), []byte("foreign synthetic"), w.o.FileMode); e != nil {
			t.Fatal(e)
		}
	case "malformed":
		if e := os.WriteFile(path, []byte("malformed"), w.o.FileMode); e != nil {
			t.Fatal(e)
		}
	case "wrong-hash":
		parts := strings.Split(p.OperationID, ":")
		p.OperationID = parts[0] + ":" + parts[1] + ":" + strings.Repeat("0", 64)
	case "wrong-head":
		p.NewHead.Hash = recovery.SHA256Digest([]byte("wrong head"))
	case "H2":
		if e := os.WriteFile(filepath.Join(w.o.Directory, deletionRecordName(2)), []byte("unauthorized tail"), w.o.FileMode); e != nil {
			t.Fatal(e)
		}
	case "committed-orphan":
		w.a.value.Pending = nil
	case "committed-missing":
		w.a.value.Committed = p.NewHead
		w.a.value.Pending = nil
	case "different-inode-alias":
		raw, e := os.ReadFile(filepath.Join(w.o.Directory, deletionRecordName(1)))
		if e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(path, raw, w.o.FileMode); e != nil {
			t.Fatal(e)
		}
	}
}

func testAnchorC1Progression(t *testing.T) {
	for _, kind := range []string{"pending-copy", "idempotent", "open-ABA", "after-open-ABA", "after-open-H2"} {
		t.Run(kind, func(t *testing.T) {
			w := newAnchorC1World(t, "stage")
			configureAnchorC1Progression(w, kind)
			e := w.run()
			positive := kind == "pending-copy" || kind == "idempotent"
			if (e == nil) != positive {
				t.Fatalf("progression %s: %v", kind, e)
			}
			if !positive && anchorC1CASCount(w.a) != 0 {
				t.Fatal("rebased origin")
			}
		})
	}
}

func configureAnchorC1Progression(w *anchorC1World, kind string) {
	w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
		if kind == "pending-copy" {
			a, _, e := w.runtime.CheckArticleCoordination(ctx, p, w.o.InstallationID, w.o.JournalID, w.o.Epoch)
			if e != nil {
				return e
			}
			a.Pending.NewHead.Hash = recovery.SHA256Digest([]byte("mutated caller copy"))
			return w.coordinate(ctx, p)
		}
		if kind == "open-ABA" {
			w.a.revision++
		}
		if e := w.open(ctx, p); e != nil {
			return e
		}
		if kind == "after-open-ABA" {
			w.a.revision++
		}
		if kind == "after-open-H2" {
			w.a.value.Pending.NewHead.Sequence++
		}
		if e := w.control.CoordinateAnchor(ctx, p); e != nil {
			return e
		}
		n := anchorC1CASCount(w.a)
		syncs := w.syncs
		if e := w.control.CoordinateAnchor(ctx, p); e != nil {
			return e
		}
		if _, e := w.control.Append(ctx, anchorRequest(2)); e == nil {
			return errors.New("business append in control mode")
		}
		if n != anchorC1CASCount(w.a) || syncs != w.syncs {
			return errors.New("idempotent coordination wrote")
		}
		s, e := w.control.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespace)
		w.local = s
		if e != nil {
			return e
		}
		return w.snapshots.bind(w.control)
	}
}

func testAnchorC1Unknown(t *testing.T) {
	t.Run("AppliedThenCancel", func(t *testing.T) {
		w := newAnchorC1World(t, "stage")
		w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
			child, cancel := context.WithCancel(ctx)
			defer cancel()
			w.o.Anchor = &anchorFaultCAS{inner: w.a, target: "commit-cas", side: "after", mode: "cancel", cancel: cancel}
			return w.coordinate(child, p)
		}
		if e := w.run(); !errors.Is(e, recovery.ErrAnchorIndeterminate) || !errors.Is(e, context.Canceled) {
			t.Fatalf("applied cancellation: %v", e)
		}
		if w.a.value.Pending != nil || w.a.value.Committed.Sequence != 1 || anchorC1CASCount(w.a) != 1 {
			t.Fatal("applied committed fact rolled back/retried")
		}
		assertAnchorTerminal(t, w.h)
	})
	for _, point := range []string{"commit-cas", "commit-load"} {
		for _, side := range []string{"before", "after"} {
			t.Run(point+"/"+side, func(t *testing.T) {
				w := newAnchorC1World(t, "stage")
				w.o.Anchor = &anchorFaultCAS{inner: w.a, target: point, side: side, mode: "error"}
				if e := w.run(); !errors.Is(e, recovery.ErrAnchorIndeterminate) {
					t.Fatalf("unknown lost: %v", e)
				}
				if anchorC1CASCount(w.a) > 1 {
					t.Fatal("blind CAS replay")
				}
				if _, e := os.Stat(filepath.Join(w.o.Directory, deletionRecordName(1))); e != nil {
					t.Fatal("published H1 removed", e)
				}
			})
		}
	}
}

func testAnchorC1PublicPlatform(t *testing.T) {
	if articleDeletionJournalPlatformSupported() {
		t.Log("native public-wrapper positive is outside this portable component suite")
		return
	}
	w := newAnchorC1World(t, "committed")
	w.execute = func(ctx context.Context, p recovery.PhasePermit) error {
		j, e := OpenAnchoredDeletionJournalForCoordinate(ctx, p, w.o, w.k)
		if j != nil {
			_ = j.Close()
			return errors.New("unsupported public wrapper returned handle")
		}
		return e
	}
	if e := w.run(); !errors.Is(e, ErrDeletionJournalUnsupportedPlatform) {
		t.Fatalf("unsupported wrapper: %v", e)
	}
	if w.locks != 0 || w.syncs != 0 || anchorC1CASCount(w.a) != 0 {
		t.Fatal("native fallback")
	}
}

// This adapter exercises portable media mechanics, never controlled-coordinate authority.
type anchorCoreRuntime struct{}

func (*anchorCoreRuntime) CheckArticleCoordination(context.Context, recovery.PhasePermit, string, string, uint64) (recovery.AnchorSnapshot, string, error) {
	return recovery.AnchorSnapshot{}, "", recovery.ErrAdmissionClosed
}

func (*anchorCoreRuntime) CheckJournalUse(context.Context, string, string) error { return nil }
func (*anchorCoreRuntime) PauseArticleAppend(context.Context, string) (recovery.AppendBarrier, error) {
	return recovery.AppendBarrier{}, nil
}
func (*anchorCoreRuntime) CompleteArticleAppend(context.Context, recovery.AppendBarrier, recovery.VerifiedHeadAdvance) error {
	return nil
}
func (*anchorCoreRuntime) FaultArticleAppend(context.Context, recovery.AppendBarrier, error) error {
	return nil
}

type anchorRuntimeWorld struct {
	mu           sync.Mutex
	runtime      recovery.Runtime
	journal      *anchoredArticleJournal
	installation recovery.InstallationBinding
	startup      func(context.Context, recovery.PhasePermit) error
	join         chan struct{}
	once         sync.Once
	deployment   *anchorWorldLease
	stops        atomic.Int32
}

func (w *anchorRuntimeWorld) ReadAdmissionSnapshot(context.Context) (recovery.AdmissionSnapshot, error) {
	s := recovery.AdmissionSnapshot{Installation: w.installation, Revision: recovery.StoreRevision{InstallationID: w.installation.InstallationID, Counter: 1}, Runs: []recovery.AdmissionRun{}}
	raw, e := recovery.CanonicalAdmissionSnapshot(s)
	s.Revision.Digest = recovery.SHA256Digest(raw)
	return s, e
}
func (w *anchorRuntimeWorld) ReadAdmissionRevision(ctx context.Context) (recovery.StoreRevision, error) {
	s, e := w.ReadAdmissionSnapshot(ctx)
	return s.Revision, e
}
func (w *anchorRuntimeWorld) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	if ns == ArticleDeletionNamespace {
		return w.journal.ReadVerifiedSnapshot(ctx, ns)
	}
	if ns != recovery.RecoveryNamespaceV1 {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	i := w.installation.RecoveryInitial
	return recovery.VerifiedJournalSnapshot{JournalID: i.JournalID, Namespace: i.Namespace, Epoch: i.Epoch, Head: i.Committed, Revision: "recovery-independent-r1", Entries: []recovery.JournalEntry{}, Prefixes: []recovery.JournalHead{{}}}, nil
}
func (*anchorRuntimeWorld) Begin(context.Context, string, string, []byte) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (*anchorRuntimeWorld) VerifyBeginSecret(context.Context, recovery.RunBinding, []byte) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (*anchorRuntimeWorld) MarkConsumed(context.Context, string) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (*anchorRuntimeWorld) Read(context.Context, string) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (*anchorRuntimeWorld) VerifyAgainst(context.Context, recovery.RecoveryRun) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (w *anchorRuntimeWorld) VerifyGenesis(context.Context, recovery.InstallationBinding) (recovery.GenesisProof, error) {
	i := w.installation
	return recovery.GenesisProof{Binding: i, RecoveryInitial: i.RecoveryInitial, ArticleInitial: i.ArticleInitial, CommitmentRevision: "genesis-independent-r1", CommitmentDigest: i.CommitmentDigest}, nil
}
func (*anchorRuntimeWorld) ReadPublishedClosure(context.Context, recovery.ClosureRequest) (recovery.PublishedClosure, error) {
	return recovery.PublishedClosure{}, recovery.ErrPublicationMissing
}
func (w *anchorRuntimeWorld) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	l := &anchorWorldLease{}
	w.mu.Lock()
	w.deployment = l
	w.mu.Unlock()
	return l, nil
}
func (w *anchorRuntimeWorld) ReconcileAndStart(ctx context.Context, p recovery.PhasePermit) error {
	if w.startup != nil {
		return w.startup(ctx, p)
	}
	return nil
}
func (w *anchorRuntimeWorld) StopAndJoin(context.Context) error {
	w.stops.Add(1)
	w.once.Do(func() { close(w.join) })
	return nil
}

type anchorWorldLease struct {
	mu     sync.Mutex
	closed bool
	closes int
}

func (l *anchorWorldLease) Check(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return recovery.ErrAdmissionClosed
	}
	return ctx.Err()
}
func (l *anchorWorldLease) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	l.closes++
	return nil
}

func actualAnchorCore(t *testing.T, startup func(*anchorRuntimeWorld, context.Context, recovery.PhasePermit) error) (*anchoredArticleJournal, *anchorRuntimeWorld, *anchorCoreCAS, deletionAnchorMedia) {
	t.Helper()
	j, w, a, m := closedActualAnchorCore(t)
	if startup != nil {
		w.startup = func(ctx context.Context, p recovery.PhasePermit) error { return startup(w, ctx, p) }
	}
	if e := w.runtime.RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
	return j, w, a, m
}

func closedActualAnchorCore(t *testing.T) (*anchoredArticleJournal, *anchorRuntimeWorld, *anchorCoreCAS, deletionAnchorMedia) {
	t.Helper()
	o, k, m, a, _ := anchorCoreFixture(t)
	w := &anchorRuntimeWorld{join: make(chan struct{})}
	w.installation = recovery.InstallationBinding{Version: recovery.InstallationVersion, DeploymentID: "synthetic-deployment", InstallationID: "synthetic-installation",
		TargetDigest: recovery.SHA256Digest([]byte("synthetic-target")), CommitmentRef: "synthetic-genesis", CommitmentDigest: recovery.SHA256Digest([]byte("synthetic-genesis")),
		InstalledAt: time.Date(2026, 10, 20, 1, 0, 0, 123456000, time.UTC), ArticleInitial: a.value,
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "synthetic-recovery", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1}}
	r, e := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: w, Journal: w, ArticleSnapshots: w, Closures: w, Genesis: w, Exclusive: w, Lifecycle: w})
	if e != nil {
		t.Fatal(e)
	}
	w.runtime = r
	o.Runtime = r
	j, e := newAnchoredDeletionJournal(context.Background(), o, k, m)
	if e != nil {
		t.Fatal(e)
	}
	w.journal = j
	t.Cleanup(func() {
		if e := r.Stop(context.Background()); e != nil {
			t.Errorf("actual runtime stop/join: %v", e)
		}
		if e := j.Close(); e != nil {
			t.Error(e)
		}
	})
	return j, w, a, m
}

func anchorRequest(n int) DeletionTombstoneRequest {
	return DeletionTombstoneRequest{ArticleID: fmt.Sprintf("11111111-1111-4111-8111-%012d", n), ArticleVersion: 3,
		DeletionOperationID: fmt.Sprintf("22222222-2222-4222-8222-%012d", n), DeletedAt: time.Date(2026, 10, 20, 1, 0, 0, 123456000, time.UTC)}
}

func TestAnchoredArticleJournal(t *testing.T) {
	t.Run("RealInstallColdOpenAndPostproof", testAnchorC1Install)
	t.Run("CommittedInstanceAndBranchPermission", testAnchorActualBranches)
	t.Run("PauseStagePendingPublishCommit", func(t *testing.T) {
		t.Run("FourWorkKindBarrier", testAnchorActualBarrier)
		t.Run("ProtocolOrderAndOriginalBytes", testAnchorProtocolOrder)
	})
	t.Run("EveryDurableBoundaryUnknown", testAnchorBoundaryFailures)
	t.Run("OwnCauseAndLateCompletion", testAnchorFaultAndJoin)
	t.Run("CompleteLinearizationAndCloseOrder", testAnchorCompleteAndCloseOrder)
	t.Run("NamespaceTerminalAndSingleClose", testAnchorNamespaceTerminal)
	t.Run("NamespaceCloseUnknownRetainsDeploymentOwnership", testAnchorUnknownCloseOwnership)
	t.Run("ReaderStartupPanicAndFaultRace", testAnchorReaderStartupFaultRace)
	t.Run("ControlledCoordinateAndABA", func(t *testing.T) {
		t.Run("NoAuthorityNoRepair", func(t *testing.T) {
			j, _, a, _ := actualAnchorCore(t, nil)
			n := len(a.calls)
			if e := j.CoordinateAnchor(context.Background(), recovery.PhasePermit{}); e == nil {
				t.Fatal("coordinate authorized")
			}
			if len(a.calls) != n {
				t.Fatal("coordinate performed I/O")
			}
		})
	})
}

type anchorTraceRuntime struct {
	recovery.Runtime
	events *[]string
}

func (p anchorTraceRuntime) PauseArticleAppend(ctx context.Context, operation string) (recovery.AppendBarrier, error) {
	*p.events = append(*p.events, "pause")
	return p.Runtime.PauseArticleAppend(ctx, operation)
}
func (p anchorTraceRuntime) CompleteArticleAppend(ctx context.Context, b recovery.AppendBarrier, a recovery.VerifiedHeadAdvance) error {
	*p.events = append(*p.events, "head-advance")
	return p.Runtime.CompleteArticleAppend(ctx, b, a)
}

type anchorTraceCAS struct {
	inner            *anchorCoreCAS
	events           *[]string
	last             string
	pendingOperation string
}

func (p *anchorTraceCAS) Load(ctx context.Context, ns string) (recovery.AnchorSnapshot, string, error) {
	if p.last != "" {
		*p.events = append(*p.events, p.last)
		p.last = ""
	}
	return p.inner.Load(ctx, ns)
}
func (p *anchorTraceCAS) CompareAndSwap(ctx context.Context, ns, r string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	point := "commit-cas"
	p.last = "commit-load"
	if next.Pending != nil {
		point = "pending-cas"
		p.last = "pending-load"
		p.pendingOperation = next.Pending.OperationID
	}
	*p.events = append(*p.events, point)
	return p.inner.CompareAndSwap(ctx, ns, r, next)
}

func testAnchorProtocolOrder(t *testing.T) {
	j, w, a, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	events := []string{}
	port := &anchorTraceCAS{inner: a, events: &events}
	j.options.Anchor = port
	j.options.Runtime = anchorTraceRuntime{Runtime: w.runtime, events: &events}
	j.media = traceAnchorMedia(m, &events)
	r, e := j.Append(lease.Context(), anchorRequest(1))
	if e != nil {
		t.Fatal(e)
	}
	want := []string{"pause", "stage", "bytes", "file-sync", "dir-1", "pending-cas", "pending-load", "link", "dir-2", "alias", "dir-3", "commit-cas", "commit-load", "head-advance"}
	if fmt.Sprint(events) != fmt.Sprint(want) {
		t.Fatalf("protocol order=%v want=%v", events, want)
	}
	raw, e := os.ReadFile(filepath.Join(j.options.Directory, deletionRecordName(1)))
	if e != nil {
		t.Fatal(e)
	}
	expected := anchorFixtureBytes(anchorRequest(1), 1, "", j.records.active)
	if string(raw) != string(expected) || r.RecordHMAC == "" {
		t.Fatal("original canonical bytes/HMAC changed")
	}
	parts := strings.Split(port.pendingOperation, ":")
	if len(parts) != 3 || parts[0] != "article-append/v1" || !validUUID(parts[1]) || "sha256:"+parts[2] != recovery.SHA256Digest(raw) {
		t.Fatalf("candidate not bound to exact bytes: %s", port.pendingOperation)
	}
}

func traceAnchorMedia(m deletionAnchorMedia, events *[]string) deletionAnchorMedia {
	base := m
	m.create = func(ctx context.Context, n string, mode fs.FileMode) (*os.File, error) {
		*events = append(*events, "stage")
		return base.create(ctx, n, mode)
	}
	m.write = func(ctx context.Context, f *os.File, raw []byte) error {
		*events = append(*events, "bytes")
		return base.write(ctx, f, raw)
	}
	m.syncFile = func(ctx context.Context, f *os.File) error {
		*events = append(*events, "file-sync")
		return base.syncFile(ctx, f)
	}
	n := 0
	m.syncDir = func(ctx context.Context) error {
		n++
		*events = append(*events, fmt.Sprintf("dir-%d", n))
		return base.syncDir(ctx)
	}
	m.link = func(ctx context.Context, a, b string) error {
		*events = append(*events, "link")
		return base.link(ctx, a, b)
	}
	m.unlink = func(ctx context.Context, a, b string, raw []byte) error {
		*events = append(*events, "alias")
		return base.unlink(ctx, a, b, raw)
	}
	return m
}

func testAnchorActualBranches(t *testing.T) {
	t.Run("PublicResumeReplayOnlyCommitted", testAnchorPublicReplay)
	t.Run("OrdinaryAndExactCommittedRepeat", testAnchorOrdinaryRepeat)
	t.Run("PublicStartupProducer", testAnchorStartupProducer)
	t.Run("ReleasedLeaseAndWrongRuntime", testAnchorWrongRuntime)
}

func testAnchorOrdinaryRepeat(t *testing.T) {
	j, w, a, _ := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	r, e := j.Append(lease.Context(), anchorRequest(1))
	if e != nil {
		t.Fatal(e)
	}
	before := len(a.calls)
	got, e := j.Append(lease.Context(), anchorRequest(1))
	if e != nil || got != r {
		t.Fatalf("repeat=%+v %v", got, e)
	}
	for _, call := range a.calls[before:] {
		if call == "cas" {
			t.Fatal("repeat wrote anchor")
		}
	}
	bad := anchorRequest(1)
	bad.ArticleVersion++
	if _, e := j.Append(lease.Context(), bad); e == nil {
		t.Fatal("same op mismatched request accepted")
	}
	if _, e := j.Append(context.Background(), anchorRequest(2)); e == nil {
		t.Fatal("caller ctx granted append")
	}
	if _, e := j.FindByOperation(context.Background(), r.DeletionOperationID); e != nil {
		t.Fatal(e)
	}
}

func testAnchorStartupProducer(t *testing.T) {
	j, w, _, _ := actualAnchorCore(t, func(w *anchorRuntimeWorld, ctx context.Context, p recovery.PhasePermit) error {
		pc, e := w.runtime.PhaseContext(ctx, p)
		if e != nil {
			return e
		}
		if _, _, e := w.runtime.CheckArticleCoordination(pc, p, w.journal.options.InstallationID, w.journal.options.JournalID, w.journal.options.Epoch); e == nil {
			return errors.New("startup obtained coordinate authority")
		}
		if lease, e := w.runtime.Acquire(ctx, recovery.WorkRequest); e == nil {
			lease.Release()
			return errors.New("startup prematurely public")
		}
		_, e = w.journal.Append(pc, anchorRequest(1))
		return e
	})
	if _, e := j.FindByOperation(context.Background(), anchorRequest(1).DeletionOperationID); e != nil {
		t.Fatal(e)
	}
	if o, e := w.runtime.Verify(context.Background()); e != nil || o.State != "public_ready" {
		t.Fatalf("startup complete: %+v %v", o, e)
	}
}

func testAnchorWrongRuntime(t *testing.T) {
	j, w, _, _ := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	lease.Release()
	if _, e := j.Append(lease.Context(), anchorRequest(1)); e == nil {
		t.Fatal("released lease accepted")
	}
	_, peer, _, _ := actualAnchorCore(t, nil)
	other, e := peer.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Release()
	if _, e := j.Append(other.Context(), anchorRequest(1)); e == nil {
		t.Fatal("other runtime accepted")
	}
}

// Neutral persistent projections are synthetic fixtures. Resume itself, permit
// production, journal parsing and the article committed branch are real code.
type anchorReplayWorld struct {
	*anchorRuntimeWorld
	row     recovery.AdmissionRun
	begin   recovery.JournalEntry
	plan    recovery.RestorePlan
	receipt recovery.RestoreReceipt
	stop    error
	checked bool
}

func (w *anchorReplayWorld) ReadAdmissionSnapshot(ctx context.Context) (recovery.AdmissionSnapshot, error) {
	s, e := w.anchorRuntimeWorld.ReadAdmissionSnapshot(ctx)
	if e != nil {
		return s, e
	}
	s.Runs = []recovery.AdmissionRun{w.row}
	raw, e := recovery.CanonicalAdmissionSnapshot(s)
	s.Revision.Digest = recovery.SHA256Digest(raw)
	return s, e
}
func (w *anchorReplayWorld) ReadAdmissionRevision(ctx context.Context) (recovery.StoreRevision, error) {
	s, e := w.ReadAdmissionSnapshot(ctx)
	return s.Revision, e
}
func (w *anchorReplayWorld) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	if ns != recovery.RecoveryNamespaceV1 {
		return w.anchorRuntimeWorld.ReadVerifiedSnapshot(ctx, ns)
	}
	i := w.installation.RecoveryInitial
	head := recovery.JournalHead{Sequence: 1, Hash: recovery.SHA256Digest([]byte("synthetic-committed-Begin"))}
	return recovery.VerifiedJournalSnapshot{JournalID: i.JournalID, Namespace: ns, Epoch: i.Epoch, Head: head, Revision: "recovery-begin-r1", Entries: []recovery.JournalEntry{w.begin}, Prefixes: []recovery.JournalHead{{}, head}}, nil
}
func (w *anchorReplayWorld) Load(context.Context, string) (recovery.RecoveryRun, error) {
	r := w.row
	return recovery.RecoveryRun{RunID: r.RunID, BackupRef: r.BackupRef, JournalKeyID: r.JournalKeyID, JournalKeyVersion: r.JournalKeyVersion, JournalSecretFingerprint: r.JournalSecretFingerprint, StartedAt: r.StartedAt, JournalStage: r.JournalStage, Outcome: r.Outcome}, nil
}
func (*anchorReplayWorld) Transition(context.Context, string, recovery.RecoveryOutcome, recovery.RecoveryOutcome, recovery.RecoveryRunChange) (recovery.RecoveryRun, error) {
	return recovery.RecoveryRun{}, recovery.ErrAdmissionClosed
}
func (w *anchorReplayWorld) Prepare(context.Context, recovery.RunBinding) (recovery.RestorePlan, error) {
	return w.plan, nil
}
func (*anchorReplayWorld) Replace(context.Context, recovery.PhasePermit, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return recovery.RestoreReceipt{}, recovery.ErrAdmissionClosed
}
func (w *anchorReplayWorld) VerifyRestored(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return w.receipt, nil
}
func (*anchorReplayWorld) BindRestoredRun(context.Context, recovery.PhasePermit, recovery.RunBinding, recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	return recovery.RecoveryRun{}, recovery.ErrAdmissionClosed
}
func (*anchorReplayWorld) FinalizeRestoredIdentity(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (*anchorReplayWorld) PrepareRecoveryTrust(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte) error {
	return recovery.ErrAdmissionClosed
}
func (*anchorReplayWorld) CompleteIdentityReview(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (*anchorReplayWorld) RevokeRecoverySessions(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (*anchorReplayWorld) ExitRecovery(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (*anchorReplayWorld) ReadInitialRecoverySecret(context.Context, recovery.RunBinding) ([]byte, error) {
	return nil, recovery.ErrAdmissionClosed
}

func (w *anchorReplayWorld) Consume(ctx context.Context, run string) (recovery.ConsumptionResult, error) {
	if run != w.row.RunID {
		return recovery.ConsumptionResult{}, recovery.ErrAdmissionBinding
	}
	if _, _, e := w.runtime.CheckArticleCoordination(ctx, recovery.PhasePermit{}, w.journal.options.InstallationID, w.journal.options.JournalID, w.journal.options.Epoch); e == nil {
		return recovery.ConsumptionResult{}, errors.New("replay obtained coordinate authority")
	}
	if _, e := w.journal.Append(ctx, anchorRequest(1)); e != nil {
		return recovery.ConsumptionResult{}, e
	}
	if _, e := w.journal.Append(ctx, anchorRequest(2)); e == nil {
		return recovery.ConsumptionResult{}, errors.New("replay appended new record")
	}
	bad := anchorRequest(1)
	bad.ArticleVersion++
	if _, e := w.journal.Append(ctx, bad); e == nil {
		return recovery.ConsumptionResult{}, errors.New("replay accepted wrong committed request")
	}
	w.checked = true
	return recovery.ConsumptionResult{}, w.stop
}

func testAnchorPublicReplay(t *testing.T) {
	j, normal, a, _ := actualAnchorCore(t, nil)
	lease, e := normal.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	_, e = j.Append(lease.Context(), anchorRequest(1))
	lease.Release()
	if e != nil {
		t.Fatal(e)
	}
	if e := normal.runtime.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	w := newAnchorReplayWorld(normal)
	r, e := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: w, Journal: w, ArticleSnapshots: w, Closures: w, Genesis: w, Exclusive: w, Lifecycle: w, Runs: w, Restore: w, Identity: w, Consumer: w, Exit: w, RecoverySecret: w})
	if e != nil {
		t.Fatal(e)
	}
	w.runtime = r
	j.options.Runtime = r
	t.Cleanup(func() {
		if e := r.Stop(context.Background()); e != nil {
			t.Error(e)
		}
	})
	before := anchorDirectoryImage(t, j.options.Directory)
	n := len(a.calls)
	e = r.Resume(context.Background(), recovery.RunBinding{RunID: w.row.RunID, BackupRef: w.row.BackupRef})
	if !errors.Is(e, w.stop) || !w.checked {
		t.Fatalf("public replay branch not reached: %v", e)
	}
	if fmt.Sprint(before) != fmt.Sprint(anchorDirectoryImage(t, j.options.Directory)) {
		t.Fatal("replay mutated media")
	}
	for _, call := range a.calls[n:] {
		if call == "cas" {
			t.Fatal("replay wrote anchor")
		}
	}
}

func newAnchorReplayWorld(base *anchorRuntimeWorld) *anchorReplayWorld {
	now := base.installation.InstalledAt
	w := &anchorReplayWorld{anchorRuntimeWorld: base, stop: errors.New("bounded replay branch observed; full recovery intentionally not executed")}
	digest := func(s string) string { return recovery.SHA256Digest([]byte(s)) }
	w.begin = recovery.JournalEntry{Namespace: recovery.RecoveryNamespaceV1, Sequence: 1, RunID: "synthetic-run", BackupRef: "synthetic-backup", KeyID: "synthetic-recovery-key", KeyVersion: "v1", SecretFingerprint: "hmac-sha256:" + strings.Repeat("a", 64), Stage: recovery.StageBegun, RecordedAt: now, RecordHMAC: "hmac-sha256:" + strings.Repeat("b", 64)}
	i := base.installation
	w.plan = recovery.RestorePlan{Version: "han-db-restore/v1", RunID: w.begin.RunID, BackupRef: w.begin.BackupRef, DeploymentID: i.DeploymentID, InstallationID: i.InstallationID, TargetDigest: i.TargetDigest, ArtifactDigest: digest("artifact"), ArtifactBytes: 100, SourceApprovalRef: "synthetic-approval", SourceApprovalDigest: digest("approval"), SourceInstallationID: "synthetic-source", SourceSchemaDigest: digest("schema"), ArtifactPath: "synthetic-artifact"}
	w.receipt = recovery.RestoreReceipt{Version: w.plan.Version, RunID: w.plan.RunID, BackupRef: w.plan.BackupRef, DeploymentID: i.DeploymentID, InstallationID: i.InstallationID, TargetDigest: i.TargetDigest, ArtifactDigest: w.plan.ArtifactDigest, ArtifactBytes: 100, SourceApprovalDigest: w.plan.SourceApprovalDigest, RestoredSchemaDigest: w.plan.SourceSchemaDigest, ReceiptDigest: digest("receipt"), RestoredAt: now}
	user, audit, change := "33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444", "55555555-5555-4555-8555-555555555555"
	bootstrap, changed := int64(2), int64(3)
	at := now.Add(time.Minute)
	w.row = recovery.AdmissionRun{RunID: w.begin.RunID, BackupRef: w.begin.BackupRef, JournalKeyID: w.begin.KeyID, JournalKeyVersion: w.begin.KeyVersion, JournalSecretFingerprint: w.begin.SecretFingerprint, StartedAt: now, JournalStage: recovery.StageBegun, Outcome: recovery.RecoveryOutcomeIdentityReady, RestoreBinding: &w.receipt, AuditEventID: &audit, InitialAdminUserID: &user, InitialAdminBootstrapGeneration: &bootstrap, InitialAdminChangedGeneration: &changed, InitialAdminChangeAuditID: &change, InitialAdminChangedAt: &at}
	return w
}

func testAnchorActualBarrier(t *testing.T) {
	j, w, a, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	staged := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer func() { unblock(); workers.Wait() }()
	done := make(chan error, 1)
	j.media.create = func(ctx context.Context, name string, mode fs.FileMode) (*os.File, error) {
		close(staged)
		<-release
		return m.create(ctx, name, mode)
	}
	anchorTestWorker(&workers, func() { _, e := j.Append(lease.Context(), anchorRequest(1)); done <- e })
	<-staged
	if l, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
		l.Release()
		t.Error("request admitted before pending")
	}
	for _, kind := range []recovery.WorkKind{recovery.WorkArticleDeletion, recovery.WorkPasswordReset, recovery.WorkArticleNotice} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		if l, e := w.runtime.Acquire(ctx, kind); !errors.Is(e, context.DeadlineExceeded) {
			if l != nil {
				l.Release()
			}
			t.Errorf("worker %s did not wait behind live barrier: %v", kind, e)
		}
		cancel()
	}
	if a.value.Pending != nil || a.value.Committed.Sequence != 0 {
		t.Error("anchor advanced before stage")
	}
	unblock()
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if a.value.Pending != nil || a.value.Committed.Sequence != 1 {
		t.Fatalf("not committed: %+v", a.value)
	}
	assertAnchorAllWorkKinds(t, w.runtime)
}

func assertAnchorAllWorkKinds(t *testing.T, runtime recovery.Runtime) {
	t.Helper()
	for _, kind := range []recovery.WorkKind{recovery.WorkRequest, recovery.WorkArticleDeletion, recovery.WorkPasswordReset, recovery.WorkArticleNotice} {
		l, e := runtime.Acquire(context.Background(), kind)
		if e != nil {
			t.Fatal(e)
		}
		l.Release()
	}
}

func testAnchorFaultAndJoin(t *testing.T) {
	t.Run("HealthyClearsOnlyOwnCause", testAnchorOwnCause)
	t.Run("UnknownRevokesAndWaitsActualLease", func(t *testing.T) {
		j, w, _, _ := actualAnchorCore(t, nil)
		lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
		if e != nil {
			t.Fatal(e)
		}
		defer lease.Release()
		j.media.syncFile = func(context.Context, *os.File) error { return errors.New("synthetic unknown sync") }
		if _, e := j.Append(lease.Context(), anchorRequest(1)); !errors.Is(e, recovery.ErrRecoveryWriteUnknown) {
			t.Fatalf("unknown=%v", e)
		}
		if l, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
			l.Release()
			t.Error("unknown admitted request")
		}
		<-w.join
		done := make(chan error, 1)
		var workers sync.WaitGroup
		defer func() { lease.Release(); workers.Wait() }()
		anchorTestWorker(&workers, func() { done <- w.runtime.Stop(context.Background()) })
		select {
		case <-done:
			t.Fatal("Stop returned before actual lease release")
		default:
		}
		lease.Release()
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	})
	t.Run("WindowsProductionUnsupported", func(t *testing.T) {
		if articleDeletionJournalPlatformSupported() {
			return
		}
		o, k, _, _, _ := anchorCoreFixture(t)
		if _, e := OpenAnchoredDeletionJournal(context.Background(), o, k); !errors.Is(e, ErrDeletionJournalUnsupportedPlatform) {
			t.Fatalf("production platform=%v", e)
		}
	})
}

func testAnchorCompleteAndCloseOrder(t *testing.T) {
	t.Run("NamespaceCloseUnknownRetainsDeploymentOwnership", testAnchorUnknownCloseOwnership)
	t.Run("NamespaceCloseBeforeFinalNotification", testAnchorLateClose)
	t.Run("AfterAcceptCancellationReturnsCommitted", testAnchorLateCancellation)
	t.Run("BeforeAcceptFailureCloses", testAnchorBeforeAcceptFailures)
	t.Run("BlockedPortsStopReallyJoins", testAnchorStopJoinsPorts)
	for _, point := range []string{"ownership-check", "complete"} {
		t.Run("CompletionPortPanicAfterTerminal/"+point, func(t *testing.T) { testAnchorCompletionPortPanic(t, point) })
	}
}

type anchorCompletionPanic struct {
	recovery.Runtime
	point  string
	value  any
	checks int
}

func (p *anchorCompletionPanic) CheckJournalUse(ctx context.Context, operation, use string) error {
	p.checks++
	if p.point == "ownership-check" && p.checks == 2 {
		panic(p.value)
	}
	return p.Runtime.CheckJournalUse(ctx, operation, use)
}
func (p *anchorCompletionPanic) CompleteArticleAppend(ctx context.Context, b recovery.AppendBarrier, a recovery.VerifiedHeadAdvance) error {
	panic(p.value)
}
func testAnchorCompletionPortPanic(t *testing.T, point string) {
	j, w, _, _ := actualAnchorCore(t, nil)
	h := newAnchorHeldNamespace(t)
	h.closeErr = nil
	marker := &struct{ point string }{point}
	port := &anchorCompletionPanic{Runtime: w.runtime, point: point, value: marker}
	p := &anchorC2Probe{Runtime: port}
	j.options.Runtime = p
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return h, nil }
	lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	func() {
		defer func() {
			if value := recover(); value != marker {
				t.Errorf("completion panic replaced: %v", value)
			}
		}()
		j.Append(lease.Context(), anchorRequest(1))
	}()
	assertAnchorTerminal(t, h)
	if p.fault.Load() != 1 || p.accepted.Load() != 0 {
		t.Errorf("completion panic did not Fault original token: fault=%d accepted=%d", p.fault.Load(), p.accepted.Load())
	}
	lease.Release()
	if err := w.runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertAnchorDeploymentCloses(t, w, 1)
}

func testAnchorUnknownCloseOwnership(t *testing.T) {
	j, w, _, m := actualAnchorCore(t, nil)
	held := newAnchorHeldNamespace(t)
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	held.probe = p
	j.media.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
		if !write {
			return m.lock(ctx, write)
		}
		return held, nil
	}
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	var released atomic.Bool
	call := startAnchorJoinedCall(t, held, func() error {
		defer func() { lease.Release(); released.Store(true) }()
		_, err := j.Append(lease.Context(), anchorRequest(1))
		return err
	})
	awaitAnchorNamespaceWait(t, held, call)
	assertAnchorHeld(t, held, w, p, call)
	first := startAnchorJoinedCall(t, held, func() error { return w.runtime.Stop(context.Background()) })
	second := startAnchorJoinedCall(t, held, func() error { return w.runtime.Stop(context.Background()) })
	<-w.join
	assertAnchorPending(t, first, second)
	if released.Load() {
		t.Fatal("caller released original work while namespace held")
	}
	assertAnchorDeploymentCloses(t, w, 0)
	held.releaseNow()
	call.join()
	first.join()
	second.join()
	if !errors.Is(call.err, held.closeErr) || call.panicValue != nil || !released.Load() {
		t.Fatalf("original result/work lost: %v panic=%v released=%v", call.err, call.panicValue, released.Load())
	}
	assertAnchorTerminal(t, held)
	assertAnchorDeploymentCloses(t, w, 1)
	if first.err != nil || second.err != nil || w.stops.Load() != 1 {
		t.Fatalf("repeated drain: %v %v stops=%d", first.err, second.err, w.stops.Load())
	}
}

func testAnchorLateClose(t *testing.T) {
	j, w, a, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	failure := errors.New("synthetic namespace Close result unknown")
	j.media.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
		if !write {
			return m.lock(ctx, write)
		}
		return anchorLeaseFuncs{close: func() error { return failure }}, nil
	}
	_, err := j.Append(lease.Context(), anchorRequest(1))
	if !errors.Is(err, failure) || a.value.Committed.Sequence != 1 {
		t.Fatalf("late Close=%v anchor=%+v", err, a.value)
	}
	next, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e == nil {
		next.Release()
		t.Fatalf("namespace Close failed after notification but admission reopened; append=%v", err)
	}
}

func testAnchorOwnCause(t *testing.T) {
	j, w, a, _ := actualAnchorCore(t, nil)
	one, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer one.Release()
	two, e := w.runtime.Acquire(context.Background(), recovery.WorkArticleNotice)
	if e != nil {
		t.Fatal(e)
	}
	defer two.Release()
	other, e := w.runtime.PauseArticleAppend(two.Context(), anchorRequest(2).DeletionOperationID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := j.Append(one.Context(), anchorRequest(1)); e != nil {
		t.Fatal(e)
	}
	if a.value.Committed.Sequence != 1 {
		t.Fatal("healthy durable fact lost")
	}
	if lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
		lease.Release()
		t.Fatal("healthy completion cleared another token")
	}
	if e := w.runtime.FaultArticleAppend(two.Context(), other, errors.New("synthetic other cause")); !errors.Is(e, recovery.ErrRecoveryWriteUnknown) {
		t.Fatal(e)
	}
}

type anchorCompletionCancel struct {
	recovery.Runtime
	cancel context.CancelFunc
}

func (p anchorCompletionCancel) CompleteArticleAppend(ctx context.Context, b recovery.AppendBarrier, a recovery.VerifiedHeadAdvance) error {
	err := p.Runtime.CompleteArticleAppend(ctx, b, a)
	if err == nil {
		p.cancel()
	}
	return err
}

func testAnchorLateCancellation(t *testing.T) {
	j, w, a, _ := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	ctx, cancel := context.WithCancel(lease.Context())
	defer cancel()
	j.options.Runtime = anchorCompletionCancel{Runtime: w.runtime, cancel: cancel}
	got, err := j.Append(ctx, anchorRequest(1))
	if err != nil || got.DeletionOperationID != anchorRequest(1).DeletionOperationID {
		t.Fatalf("accepted result changed by late cancellation: %+v %v", got, err)
	}
	if a.value.Committed.Sequence != 1 || a.value.Pending != nil {
		t.Fatal("durable fact rolled back")
	}
	next, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatalf("known successful append faulted runtime: %v", e)
	}
	if next != nil {
		next.Release()
	}
}

type anchorC2Probe struct {
	recovery.Runtime
	complete, accepted, fault atomic.Int32
}

func (p *anchorC2Probe) CompleteArticleAppend(ctx context.Context, b recovery.AppendBarrier, a recovery.VerifiedHeadAdvance) error {
	p.complete.Add(1)
	e := p.Runtime.CompleteArticleAppend(ctx, b, a)
	if e == nil {
		p.accepted.Add(1)
	}
	return e
}
func (p *anchorC2Probe) FaultArticleAppend(ctx context.Context, b recovery.AppendBarrier, e error) error {
	p.fault.Add(1)
	return p.Runtime.FaultArticleAppend(ctx, b, e)
}

func testAnchorBeforeAcceptFailures(t *testing.T) {
	for _, point := range []string{"media-check", "namespace-check", "namespace-close", "cancel-during-close", "panic-during-close"} {
		t.Run(point, func(t *testing.T) { testAnchorBeforeAcceptFailure(t, point) })
	}
}

func testAnchorBeforeAcceptFailure(t *testing.T, point string) {
	j, w, a, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	ctx, cancel := context.WithCancel(lease.Context())
	defer cancel()
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	sentinel := errors.New("synthetic finalization error")
	fault := configureAnchorFinalizationFault(j, m, point, sentinel, cancel)
	var result error
	func() {
		defer func() {
			r := recover()
			if point == "panic-during-close" {
				if r != sentinel {
					t.Errorf("close panic replaced: %v", r)
				}
			} else if r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		_, result = j.Append(ctx, anchorRequest(1))
	}()
	if point != "panic-during-close" && !errors.Is(result, recovery.ErrRecoveryWriteUnknown) {
		t.Fatalf("failure=%v", result)
	}
	if p.complete.Load() != 0 || p.accepted.Load() != 0 || p.fault.Load() != 1 || fault.closed != 1 {
		t.Fatalf("completion=%d accepted=%d fault=%d closed=%d", p.complete.Load(), p.accepted.Load(), p.fault.Load(), fault.closed)
	}
	if a.value.Committed.Sequence != 1 {
		t.Fatal("H1 rolled back")
	}
	if next, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
		next.Release()
		t.Fatal("pre-accept failure reopened")
	}
}

type anchorC2BlockingCAS struct {
	recovery.JournalHeadAnchor
	entered chan struct{}
	release <-chan struct{}
}

func (p anchorC2BlockingCAS) CompareAndSwap(ctx context.Context, ns, r string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	s, token, e := p.JournalHeadAnchor.CompareAndSwap(ctx, ns, r, next)
	close(p.entered)
	<-p.release
	return s, token, e
}

func testAnchorStopJoinsPorts(t *testing.T) {
	for _, point := range []string{"Close", "CAS"} {
		t.Run(point, func(t *testing.T) { testAnchorStopJoinsPort(t, point) })
	}
}

func testAnchorStopJoinsPort(t *testing.T, point string) {
	j, w, a, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer func() { unblock(); workers.Wait() }()
	configureAnchorBlockedPort(j, a, m, point, entered, release)
	done := make(chan error, 1)
	anchorTestWorker(&workers, func() { defer lease.Release(); _, e := j.Append(lease.Context(), anchorRequest(1)); done <- e })
	<-entered
	stop := make(chan error, 1)
	anchorTestWorker(&workers, func() { stop <- w.runtime.Stop(context.Background()) })
	<-w.join
	select {
	case <-stop:
		t.Fatal("Stop returned while port was still executing")
	default:
	}
	if p.complete.Load() != 0 {
		t.Error("Complete before port join")
	}
	unblock()
	if e := <-done; e == nil {
		t.Error("canceled append succeeded before acceptance")
	}
	if e := <-stop; e != nil {
		t.Error(e)
	}
	if p.accepted.Load() != 0 {
		t.Error("canceled stopped append accepted")
	}
}

func configureAnchorBlockedPort(j *anchoredArticleJournal, a *anchorCoreCAS, m deletionAnchorMedia, point string, entered chan struct{}, release <-chan struct{}) {
	if point == "CAS" {
		j.options.Anchor = anchorC2BlockingCAS{JournalHeadAnchor: a, entered: entered, release: release}
	} else {
		j.media.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
			l, e := m.lock(ctx, write)
			if e != nil || !write {
				return l, e
			}
			return anchorLeaseFuncs{check: l.Check, close: func() error { close(entered); <-release; return l.Close() }, wait: l.WaitReleased}, nil
		}
	}
}

func anchorTestWorker(workers *sync.WaitGroup, run func()) {
	workers.Add(1)
	go func() { defer workers.Done(); run() }()
}

func testAnchorBoundaryFailures(t *testing.T) {
	t.Run("AnchorCASAndIndependentLoad", testAnchorCASBoundaries)
	t.Run("PanicFinalizesNamespaceAndStage", testAnchorWriterPanic)
	for _, boundary := range []string{"create", "write", "file-sync", "stage-dir", "link", "final-dir", "alias", "alias-dir", "verify-chain"} {
		for _, side := range []string{"before", "after"} {
			for _, mode := range []string{"error", "cancel", "panic"} {
				t.Run(boundary+"/"+side+"/"+mode, func(t *testing.T) { testAnchorMediaBoundary(t, boundary, side, mode) })
			}
		}
	}
}

func testAnchorMediaBoundary(t *testing.T, boundary, side, mode string) {
	j, w, _, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	ctx, cancel := context.WithCancel(lease.Context())
	defer cancel()
	marker := &struct{ boundary string }{boundary}
	fault := func() error {
		if mode == "panic" {
			panic(marker)
		}
		if mode == "cancel" {
			cancel()
			return context.Canceled
		}
		return errors.New("synthetic boundary failure")
	}
	injectAnchorMedia(j, m, boundary, side, fault)
	var result error
	func() {
		defer func() {
			p := recover()
			if mode == "panic" {
				if p != marker {
					t.Errorf("panic replaced: %v", p)
				}
			} else if p != nil {
				t.Errorf("unexpected panic: %v", p)
			}
		}()
		_, result = j.Append(ctx, anchorRequest(1))
	}()
	if mode != "panic" && !errors.Is(result, recovery.ErrRecoveryWriteUnknown) {
		t.Fatalf("boundary result %v", result)
	}
	if l, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
		l.Release()
		t.Fatal("fault reopened")
	}
	entries, e := os.ReadDir(j.options.Directory)
	if e != nil {
		t.Fatal(e)
	}
	if (boundary != "create" || side == "after") && len(entries) == 0 {
		t.Fatal("failure artifacts lost")
	}
}

func injectAnchorMedia(j *anchoredArticleJournal, m deletionAnchorMedia, boundary, side string, fault func() error) {
	call := func(action func() error) error {
		if side == "before" {
			return fault()
		}
		return errors.Join(action(), fault())
	}
	switch boundary {
	case "create":
		j.media.create = anchorCreateFault(m, side, fault)
	case "file-sync":
		j.media.syncFile = func(ctx context.Context, f *os.File) error { return call(func() error { return m.syncFile(ctx, f) }) }
	case "link":
		j.media.link = func(ctx context.Context, a, b string) error { return call(func() error { return m.link(ctx, a, b) }) }
	case "alias":
		j.media.unlink = func(ctx context.Context, a, b string, raw []byte) error {
			return call(func() error { return m.unlink(ctx, a, b, raw) })
		}
	case "verify-chain":
		n := 0
		j.media.names = func(ctx context.Context) ([]string, error) {
			n++
			if n != 3 {
				return m.names(ctx)
			}
			if side == "before" {
				return nil, fault()
			}
			names, e := m.names(ctx)
			return names, errors.Join(e, fault())
		}
	case "write":
		j.media.write = func(ctx context.Context, f *os.File, raw []byte) error {
			return call(func() error { return m.write(ctx, f, raw) })
		}
	default:
		n := 0
		want := map[string]int{"stage-dir": 1, "final-dir": 2, "alias-dir": 3}[boundary]
		j.media.syncDir = func(ctx context.Context) error {
			n++
			if n == want {
				return call(func() error { return m.syncDir(ctx) })
			}
			return m.syncDir(ctx)
		}
	}
}

func anchorCreateFault(m deletionAnchorMedia, side string, fault func() error) func(context.Context, string, fs.FileMode) (*os.File, error) {
	return func(ctx context.Context, n string, mode fs.FileMode) (*os.File, error) {
		if side == "before" {
			return nil, fault()
		}
		file, err := m.create(ctx, n, mode)
		returned := false
		if file != nil {
			defer func() {
				if !returned {
					_ = file.Close()
				}
			}()
		}
		injected := fault()
		returned = true
		return file, errors.Join(err, injected)
	}
}

type anchorFaultCAS struct {
	inner                    *anchorCoreCAS
	target, side, mode, last string
	cancel                   context.CancelFunc
	marker                   any
}

func (p *anchorFaultCAS) fault() error {
	switch p.mode {
	case "panic":
		panic(p.marker)
	case "cancel":
		p.cancel()
		return context.Canceled
	default:
		return errors.New("synthetic Anchor result unknown")
	}
}

func (p *anchorFaultCAS) Load(ctx context.Context, ns string) (recovery.AnchorSnapshot, string, error) {
	point := p.last
	p.last = ""
	if point == p.target && p.side == "before" {
		return recovery.AnchorSnapshot{}, "", p.fault()
	}
	s, r, e := p.inner.Load(ctx, ns)
	if point == p.target && p.side == "after" {
		return s, r, errors.Join(e, p.fault())
	}
	return s, r, e
}

func (p *anchorFaultCAS) CompareAndSwap(ctx context.Context, ns, r string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	point := "commit-cas"
	p.last = "commit-load"
	if next.Pending != nil {
		point = "pending-cas"
		p.last = "pending-load"
	}
	if point == p.target && p.side == "before" {
		return recovery.AnchorSnapshot{}, "", p.fault()
	}
	s, revision, e := p.inner.CompareAndSwap(ctx, ns, r, next)
	if point == p.target && p.side == "after" {
		return s, revision, errors.Join(e, p.fault())
	}
	return s, revision, e
}

func testAnchorCASBoundaries(t *testing.T) {
	for _, point := range []string{"pending-cas", "pending-load", "commit-cas", "commit-load"} {
		for _, side := range []string{"before", "after"} {
			for _, mode := range []string{"error", "cancel", "panic"} {
				t.Run(point+"/"+side+"/"+mode, func(t *testing.T) { testAnchorCASBoundary(t, point, side, mode) })
			}
		}
	}
}

func testAnchorCASBoundary(t *testing.T, point, side, mode string) {
	j, w, a, _ := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	ctx, cancel := context.WithCancel(lease.Context())
	defer cancel()
	marker := &struct{ phase string }{point}
	j.options.Anchor = &anchorFaultCAS{inner: a, target: point, side: side, mode: mode, cancel: cancel, marker: marker}
	var result error
	func() {
		defer func() {
			got := recover()
			if mode == "panic" {
				if got != marker {
					t.Errorf("panic replaced: %v", got)
				}
			} else if got != nil {
				t.Errorf("unexpected panic: %v", got)
			}
		}()
		_, result = j.Append(ctx, anchorRequest(1))
	}()
	if mode != "panic" && !errors.Is(result, recovery.ErrRecoveryWriteUnknown) {
		t.Fatalf("unknown result=%v", result)
	}
	if l, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
		l.Release()
		t.Fatal("fault admitted work")
	}
	before := anchorDirectoryImage(t, j.options.Directory)
	n := len(a.calls)
	_, _ = j.ReadVerifiedSnapshot(context.Background(), ArticleDeletionNamespace)
	after := anchorDirectoryImage(t, j.options.Directory)
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatal("normal reader changed failed media")
	}
	for _, call := range a.calls[n:] {
		if call == "cas" {
			t.Fatal("normal reader repaired failed CAS")
		}
	}
}

func anchorDirectoryImage(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, e := os.ReadDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	image := make(map[string]string, len(entries))
	for _, entry := range entries {
		raw, e := os.ReadFile(filepath.Join(dir, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		image[entry.Name()] = recovery.SHA256Digest(raw)
	}
	return image
}

func testAnchorWriterPanic(t *testing.T) {
	j, w, _, m := actualAnchorCore(t, nil)
	lease, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Release()
	namespace := &anchorCoreLease{}
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) {
		return newAnchorCoreNamespaceLease(namespace), nil
	}
	var opened *os.File
	j.media.create = func(ctx context.Context, n string, mode fs.FileMode) (*os.File, error) {
		f, e := m.create(ctx, n, mode)
		opened = f
		return f, e
	}
	marker := &struct{ label string }{"file-sync-panic"}
	j.media.syncFile = func(context.Context, *os.File) error { panic(marker) }
	func() {
		defer func() {
			if got := recover(); got != marker {
				t.Errorf("panic replaced=%v", got)
			}
		}()
		j.Append(lease.Context(), anchorRequest(1))
	}()
	if namespace.closes != 1 || opened.Fd() != ^uintptr(0) {
		t.Fatal("panic leaked namespace/file resource")
	}
	if l, e := w.runtime.Acquire(context.Background(), recovery.WorkRequest); e == nil {
		l.Release()
		t.Fatal("panic reopened work")
	}
}

type anchorC2FinalizationFault struct {
	point          string
	sentinel       error
	cancel         context.CancelFunc
	checks, closed int
}

func configureAnchorFinalizationFault(j *anchoredArticleJournal, m deletionAnchorMedia, point string, sentinel error, cancel context.CancelFunc) *anchorC2FinalizationFault {
	fault := &anchorC2FinalizationFault{point: point, sentinel: sentinel, cancel: cancel}
	if point == "media-check" {
		j.media.check = func(context.Context) error {
			fault.checks++
			if fault.checks == 2 {
				return sentinel
			}
			return nil
		}
	}
	j.media.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
		lease, err := m.lock(ctx, write)
		if err != nil || !write {
			return lease, err
		}
		return anchorLeaseFuncs{check: func(ctx context.Context) error { return fault.check(ctx, lease) }, close: func() error { return fault.close(lease) }, wait: lease.WaitReleased}, nil
	}
	return fault
}
func (f *anchorC2FinalizationFault) check(ctx context.Context, lease recovery.ExclusiveLease) error {
	if f.point == "namespace-check" {
		f.checks++
		if f.checks == 2 {
			return f.sentinel
		}
	}
	return lease.Check(ctx)
}
func (f *anchorC2FinalizationFault) close(lease recovery.ExclusiveLease) error {
	defer func() { f.closed++ }()
	err := lease.Close()
	switch f.point {
	case "namespace-close":
		return errors.Join(err, f.sentinel)
	case "cancel-during-close":
		f.cancel()
	case "panic-during-close":
		panic(f.sentinel)
	}
	return err
}

// This fixture's terminal event follows an actual close of its one held file.
// The injected Close result precedes that raw close; it never retries a raw FD.
type anchorHeldNamespace struct {
	synchronous                                   bool
	file                                          *os.File
	closeErr, waitErr, releaseErr                 error
	closePanic                                    any
	probe                                         *anchorC2Probe
	requested, release, terminal, waiting         chan struct{}
	faultRelease                                  chan struct{}
	requestOnce, releaseOnce, waitOnce, faultOnce sync.Once
	driver                                        sync.WaitGroup
	closeAttempts, rawCloses, waits               atomic.Int32
	faultAtWait                                   atomic.Bool
}

func newAnchorHeldNamespace(t *testing.T) *anchorHeldNamespace {
	t.Helper()
	file, err := os.CreateTemp("", "han-a47-namespace-terminal-")
	if err != nil {
		t.Fatal(err)
	}
	h := &anchorHeldNamespace{file: file, closeErr: errors.New("synthetic namespace Close unknown"),
		requested: make(chan struct{}), release: make(chan struct{}), terminal: make(chan struct{}), waiting: make(chan struct{})}
	h.driver.Add(1)
	go func() {
		defer h.driver.Done()
		<-h.release
		h.rawCloses.Add(1)
		h.releaseErr = h.file.Close()
		if h.releaseErr == nil {
			close(h.terminal)
		}
	}()
	t.Cleanup(func() {
		h.releaseNow()
		h.driver.Wait()
		if h.releaseErr != nil {
			t.Errorf("actual fixture file release: %v", h.releaseErr)
		}
	})
	return h
}

func (h *anchorHeldNamespace) Check(ctx context.Context) error {
	_, err := h.file.Stat()
	return errors.Join(ctx.Err(), err)
}
func (h *anchorHeldNamespace) Close() error {
	h.requestOnce.Do(func() { h.closeAttempts.Add(1); close(h.requested) })
	if h.closePanic != nil {
		panic(h.closePanic)
	}
	if h.closeErr == nil || h.synchronous {
		h.releaseNow()
		<-h.terminal
		h.driver.Wait()
	}
	return h.closeErr
}
func (h *anchorHeldNamespace) WaitReleased(ctx context.Context) error {
	h.waits.Add(1)
	if h.probe != nil {
		h.faultAtWait.Store(h.probe.fault.Load() != 0)
	}
	h.waitOnce.Do(func() { close(h.waiting) })
	<-h.terminal
	h.driver.Wait()
	return errors.Join(h.closeErr, h.waitErr, h.releaseErr, ctx.Err())
}
func (h *anchorHeldNamespace) releaseFault() {
	if h.faultRelease != nil {
		h.faultOnce.Do(func() { close(h.faultRelease) })
	}
}
func (h *anchorHeldNamespace) releaseNow() {
	h.releaseFault()
	h.releaseOnce.Do(func() { close(h.release) })
}

type anchorJoinedCall struct {
	done       chan struct{}
	worker     sync.WaitGroup
	err        error
	panicValue any
}

func startAnchorJoinedCall(t *testing.T, h *anchorHeldNamespace, fn func() error) *anchorJoinedCall {
	t.Helper()
	c := &anchorJoinedCall{done: make(chan struct{})}
	c.worker.Add(1)
	go func() {
		defer c.worker.Done()
		defer close(c.done)
		defer func() { c.panicValue = recover() }()
		c.err = fn()
	}()
	t.Cleanup(func() { h.releaseNow(); c.join() })
	return c
}
func (c *anchorJoinedCall) join() { <-c.done; c.worker.Wait() }
func assertAnchorPending(t *testing.T, calls ...*anchorJoinedCall) {
	t.Helper()
	for _, c := range calls {
		select {
		case <-c.done:
			t.Fatal("original call/Stop returned before terminal")
		default:
		}
	}
}
func awaitAnchorNamespaceWait(t *testing.T, h *anchorHeldNamespace, call *anchorJoinedCall) {
	t.Helper()
	select {
	case <-h.waiting:
	case <-call.done:
		call.join()
		t.Fatalf("original call ended without terminal resource join: err=%v panic=%v", call.err, call.panicValue)
	}
}
func assertAnchorDeploymentCloses(t *testing.T, w *anchorRuntimeWorld, want int) {
	t.Helper()
	w.mu.Lock()
	deployment := w.deployment
	w.mu.Unlock()
	if deployment == nil {
		t.Fatal("actual deployment lease not acquired")
	}
	deployment.mu.Lock()
	got := deployment.closes
	deployment.mu.Unlock()
	if got != want {
		t.Fatalf("actual deployment Close=%d want=%d", got, want)
	}
}
func assertAnchorTerminal(t *testing.T, h *anchorHeldNamespace) {
	t.Helper()
	h.driver.Wait()
	if h.rawCloses.Load() != 1 || h.closeAttempts.Load() != 1 || h.releaseErr != nil {
		t.Fatalf("release proof: raw=%d attempts=%d err=%v", h.rawCloses.Load(), h.closeAttempts.Load(), h.releaseErr)
	}
	select {
	case <-h.terminal:
	default:
		t.Fatal("no physical terminal event")
	}
}
func assertAnchorHeld(t *testing.T, h *anchorHeldNamespace, w *anchorRuntimeWorld, p *anchorC2Probe, call *anchorJoinedCall) {
	t.Helper()
	assertAnchorPending(t, call)
	if _, err := h.file.Stat(); err != nil {
		t.Fatalf("synthetic file no longer held: %v", err)
	}
	if h.rawCloses.Load() != 0 || p.complete.Load() != 0 || !h.faultAtWait.Load() {
		t.Fatalf("premature close/Complete or late Fault: raw=%d complete=%d faultBeforeWait=%v", h.rawCloses.Load(), p.complete.Load(), h.faultAtWait.Load())
	}
	a, _, err := w.journal.options.Anchor.Load(context.Background(), ArticleDeletionNamespace)
	if err != nil || a.Committed.Sequence != 1 || a.Pending != nil {
		t.Fatalf("H1 lost: %+v %v", a, err)
	}
	if lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest); err == nil {
		lease.Release()
		t.Fatal("unknown namespace admitted work")
	}
	assertAnchorDeploymentCloses(t, w, 0)
}

func testAnchorNamespaceTerminal(t *testing.T) {
	for _, mode := range []string{"nil", "close-error", "join-error", "cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) { testAnchorNamespaceTerminalCase(t, mode) })
	}
}
func testAnchorNamespaceTerminalCase(t *testing.T, mode string) {
	h := newAnchorHeldNamespace(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if mode == "nil" {
		h.closeErr = nil
	}
	if mode == "close-error" {
		h.synchronous = true
	}
	if mode == "join-error" {
		h.waitErr = errors.New("synthetic terminal join cause")
	}
	if mode == "deadline" {
		var stop context.CancelFunc
		ctx, stop = context.WithDeadline(ctx, time.Now().Add(-time.Second))
		defer stop()
	}
	first, second := h.Close(), h.Close()
	if first != h.closeErr || second != h.closeErr {
		t.Fatal("Close result changed")
	}
	if mode == "nil" || mode == "close-error" {
		assertAnchorTerminal(t, h)
		assertAnchorTerminalReturn(t, h, h.WaitReleased(ctx), mode)
		return
	}
	call := startAnchorJoinedCall(t, h, func() error { return h.WaitReleased(ctx) })
	awaitAnchorNamespaceWait(t, h, call)
	if mode == "cancel" {
		cancel()
	}
	assertAnchorPending(t, call)
	if h.rawCloses.Load() != 0 {
		t.Fatal("raw close before driver release")
	}
	h.releaseNow()
	call.join()
	assertAnchorTerminal(t, h)
	assertAnchorTerminalReturn(t, h, call.err, mode)
}

func assertAnchorTerminalReturn(t *testing.T, h *anchorHeldNamespace, result error, mode string) {
	t.Helper()
	if h.closeErr != nil && !errors.Is(result, h.closeErr) {
		t.Fatalf("Close cause lost: %v", result)
	}
	if h.waitErr != nil && !errors.Is(result, h.waitErr) {
		t.Fatalf("join cause lost: %v", result)
	}
	if mode == "cancel" && !errors.Is(result, context.Canceled) {
		t.Fatal(result)
	}
	if mode == "deadline" && !errors.Is(result, context.DeadlineExceeded) {
		t.Fatal(result)
	}
	if mode == "nil" && result != nil {
		t.Fatal(result)
	}
	err := h.WaitReleased(context.Background())
	if h.closeErr != nil && !errors.Is(err, h.closeErr) {
		t.Fatalf("Close cause lost on rejoin: %v", err)
	}
	if h.waitErr != nil && !errors.Is(err, h.waitErr) {
		t.Fatalf("terminal rejoin=%v", err)
	}
}

func testAnchorReaderStartupFaultRace(t *testing.T) {
	t.Run("AcquireOwnedWithError", testAnchorAcquireOwnedError)
	t.Run("PureReaderWorkLifetime", testAnchorReaderWorkLifetime)
	t.Run("PureReaderPanicWaitsForFile", func(t *testing.T) { testAnchorReaderLifetime(t, &struct{ label string }{"reader-panic"}) })
	t.Run("AdmissionReaderWithoutWorkLease", func(t *testing.T) { testAnchorActionLifetime(t, false) })
	t.Run("StartupAppendAction", func(t *testing.T) { testAnchorActionLifetime(t, true) })
	t.Run("AdmissionReaderPanic", func(t *testing.T) { testAnchorActionLifetime(t, false, &struct{ label string }{"readProof-panic"}) })
	t.Run("StartupAppendPanic", func(t *testing.T) { testAnchorActionLifetime(t, true, &struct{ label string }{"startup-panic"}) })
	for _, point := range []string{"body", "close"} {
		t.Run("Panic/"+point, func(t *testing.T) { testAnchorHeldPanic(t, point) })
	}
	t.Run("FaultRejectedAfterStop", testAnchorFaultStopRace)
}

func testAnchorAcquireOwnedError(t *testing.T) {
	j, w, _, _ := actualAnchorCore(t, nil)
	h := newAnchorHeldNamespace(t)
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	acquireErr := errors.New("synthetic acquisition ACK with owned file")
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return h, acquireErr }
	lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	call := startAnchorJoinedCall(t, h, func() error { defer lease.Release(); _, err := j.Append(lease.Context(), anchorRequest(1)); return err })
	awaitAnchorNamespaceWait(t, h, call)
	if p.fault.Load() != 0 || p.complete.Load() != 0 {
		t.Fatal("unarmed acquisition invented a barrier")
	}
	stop := startAnchorJoinedCall(t, h, func() error { return w.runtime.Stop(context.Background()) })
	<-w.join
	assertAnchorPending(t, call, stop)
	assertAnchorDeploymentCloses(t, w, 0)
	h.releaseNow()
	call.join()
	stop.join()
	assertAnchorTerminal(t, h)
	if !errors.Is(call.err, acquireErr) || !errors.Is(call.err, h.closeErr) {
		t.Fatalf("causes lost: %v", call.err)
	}
	assertAnchorDeploymentCloses(t, w, 1)
}

func testAnchorReaderWorkLifetime(t *testing.T) {
	testAnchorReaderLifetime(t, nil)
}

func testAnchorReaderLifetime(t *testing.T, panicValue any) {
	j, w, a, _ := actualAnchorCore(t, nil)
	h := newAnchorHeldNamespace(t)
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return h, nil }
	if panicValue != nil {
		j.options.Anchor = anchorLoadPort{load: func(context.Context, string) (recovery.AnchorSnapshot, string, error) { panic(panicValue) }}
	}
	lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	call := startAnchorJoinedCall(t, h, func() error {
		defer lease.Release()
		_, err := j.ReadVerifiedSnapshot(lease.Context(), ArticleDeletionNamespace)
		return err
	})
	awaitAnchorNamespaceWait(t, h, call)
	stop := startAnchorJoinedCall(t, h, func() error { return w.runtime.Stop(context.Background()) })
	<-w.join
	assertAnchorPending(t, call, stop)
	assertAnchorDeploymentCloses(t, w, 0)
	if p.fault.Load() != 0 || p.complete.Load() != 0 {
		t.Fatal("reader invented a writer barrier")
	}
	h.releaseNow()
	call.join()
	stop.join()
	assertAnchorTerminal(t, h)
	if panicValue == nil && !errors.Is(call.err, h.closeErr) {
		t.Fatal(call.err)
	}
	if call.panicValue != panicValue {
		t.Fatalf("reader panic replaced: %v", call.panicValue)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, event := range a.calls {
		if event == "cas" {
			t.Fatal("reader wrote anchor")
		}
	}
	assertAnchorDeploymentCloses(t, w, 1)
}

func testAnchorActionLifetime(t *testing.T, startup bool, panicValues ...any) {
	j, w, _, m := closedActualAnchorCore(t)
	h := newAnchorHeldNamespace(t)
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	var panicValue any
	if len(panicValues) != 0 {
		panicValue = panicValues[0]
	}
	if panicValue != nil && startup {
		j.media.syncFile = func(context.Context, *os.File) error { panic(panicValue) }
	}
	if panicValue != nil && !startup {
		j.options.Anchor = anchorLoadPort{load: func(context.Context, string) (recovery.AnchorSnapshot, string, error) { panic(panicValue) }}
	}
	j.media.lock = func(ctx context.Context, write bool) (deletionNamespaceLease, error) {
		if write == startup {
			return h, nil
		}
		return m.lock(ctx, write)
	}
	if startup {
		configureAnchorStartupAction(w, h, p)
	}
	call := startAnchorJoinedCall(t, h, func() error { return w.runtime.RunNormal(context.Background()) })
	awaitAnchorNamespaceWait(t, h, call)
	first := startAnchorJoinedCall(t, h, func() error { return w.runtime.Stop(context.Background()) })
	second := startAnchorJoinedCall(t, h, func() error { return w.runtime.Stop(context.Background()) })
	assertAnchorPending(t, call, first, second)
	assertAnchorDeploymentCloses(t, w, 0)
	if !startup && p.fault.Load() != 0 {
		t.Fatal("admission reader forged Fault")
	}
	if startup && !h.faultAtWait.Load() {
		t.Fatal("startup joined before Fault")
	}
	h.releaseNow()
	call.join()
	first.join()
	second.join()
	assertAnchorTerminal(t, h)
	if (panicValue == nil && !errors.Is(call.err, h.closeErr)) || call.panicValue != panicValue {
		t.Fatalf("action result=%v panic=%v", call.err, call.panicValue)
	}
	assertAnchorDeploymentCloses(t, w, 1)
}

func configureAnchorStartupAction(w *anchorRuntimeWorld, h *anchorHeldNamespace, p *anchorC2Probe) {
	h.probe = p
	w.startup = func(ctx context.Context, permit recovery.PhasePermit) error {
		pc, err := w.runtime.PhaseContext(ctx, permit)
		if err != nil {
			return err
		}
		_, err = w.journal.Append(pc, anchorRequest(1))
		return err
	}
}

func testAnchorHeldPanic(t *testing.T, point string) {
	j, w, _, m := actualAnchorCore(t, nil)
	h := newAnchorHeldNamespace(t)
	p := &anchorC2Probe{Runtime: w.runtime}
	j.options.Runtime = p
	h.probe = p
	marker := errors.New("synthetic original panic")
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return h, nil }
	if point == "body" {
		j.media.syncFile = func(context.Context, *os.File) error { panic(marker) }
	} else {
		h.closePanic = marker
	}
	lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	call := startAnchorJoinedCall(t, h, func() error { defer lease.Release(); _, err := j.Append(lease.Context(), anchorRequest(1)); return err })
	awaitAnchorNamespaceWait(t, h, call)
	stop := startAnchorJoinedCall(t, h, func() error { return w.runtime.Stop(context.Background()) })
	assertAnchorPending(t, call, stop)
	assertAnchorDeploymentCloses(t, w, 0)
	if !h.faultAtWait.Load() || p.complete.Load() != 0 {
		t.Fatal("panic joined before revocation")
	}
	h.releaseNow()
	call.join()
	stop.join()
	assertAnchorTerminal(t, h)
	if call.panicValue != marker {
		t.Fatalf("original panic replaced: %v", call.panicValue)
	}
	j.media = m
	assertAnchorDeploymentCloses(t, w, 1)
}

type anchorFaultGate struct {
	recovery.Runtime
	entered chan struct{}
	release <-chan struct{}
	result  chan error
}

func (g anchorFaultGate) FaultArticleAppend(ctx context.Context, b recovery.AppendBarrier, cause error) error {
	close(g.entered)
	<-g.release
	err := g.Runtime.FaultArticleAppend(ctx, b, cause)
	g.result <- err
	return err
}
func testAnchorFaultStopRace(t *testing.T) {
	j, w, _, _ := actualAnchorCore(t, nil)
	h := newAnchorHeldNamespace(t)
	h.faultRelease = make(chan struct{})
	gate := anchorFaultGate{Runtime: w.runtime, entered: make(chan struct{}), release: h.faultRelease, result: make(chan error, 1)}
	p := &anchorC2Probe{Runtime: gate}
	j.options.Runtime = p
	h.probe = p
	j.media.lock = func(context.Context, bool) (deletionNamespaceLease, error) { return h, nil }
	lease, err := w.runtime.Acquire(context.Background(), recovery.WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	call := startAnchorJoinedCall(t, h, func() error { defer lease.Release(); _, err := j.Append(lease.Context(), anchorRequest(1)); return err })
	<-gate.entered
	stopCtx, cancel := context.WithCancel(context.Background())
	stop := startAnchorJoinedCall(t, h, func() error { return w.runtime.Stop(stopCtx) })
	<-w.join
	cancel()
	h.releaseFault()
	awaitAnchorNamespaceWait(t, h, call)
	assertAnchorHeld(t, h, w, p, call)
	assertAnchorPending(t, stop)
	if err := <-gate.result; !errors.Is(err, recovery.ErrAdmissionBinding) {
		t.Fatalf("generation race=%v", err)
	}
	heldClose := startAnchorJoinedCall(t, h, j.Close)
	assertAnchorPending(t, heldClose)
	h.releaseNow()
	call.join()
	stop.join()
	heldClose.join()
	assertAnchorTerminal(t, h)
	if !errors.Is(call.err, recovery.ErrAdmissionBinding) || !errors.Is(call.err, h.closeErr) {
		t.Fatalf("race causes lost: %v", call.err)
	}
	if !errors.Is(stop.err, context.Canceled) {
		t.Fatalf("Stop result=%v", stop.err)
	}
	assertAnchorDeploymentCloses(t, w, 1)
}

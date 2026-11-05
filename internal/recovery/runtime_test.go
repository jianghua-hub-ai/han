package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecoveryRuntimeInstall(t *testing.T) {
	t.Run("ModeDependenciesAndDispatch", func(t *testing.T) {
		t.Run("PublicBoundProducer", testInstallPublicBound)
		testInstallationModes(t)
	})
	t.Run("BoundSourcePrePostProof", testInstallationSources)
	t.Run("ExactCurrentGrantAndPendingCopy", testInstallationGrant)
	t.Run("ExecutorReturnIsNotProof", testInstallationPostProof)
	t.Run("StopCancelsAndJoinsInstall", testInstallationJoin)
	t.Run("InstallNeverOpensAndKeepsOwnCause", testInstallationClosed)
}

type installationFixture struct {
	f                               *admissionFixture
	r                               Runtime
	m                               InstallationManifest
	approved                        InstallationManifest
	commitment                      InstallationCommitment
	commitRevision                  string
	anchor                          AnchorSnapshot
	revision                        string
	calls                           int
	execute                         func(context.Context, PhasePermit) error
	hook                            func(string)
	sourceErr, commitErr, anchorErr error
	loadCount                       int
	saved                           PhasePermit
	savedContext                    context.Context
	lastContext                     context.Context
	aliasAnchor                     bool
	snapshotReady                   bool
	badStoreDigest                  bool
}

func newInstallationFixture(t *testing.T, pending bool) *installationFixture {
	t.Helper()
	f := newAdmissionFixture(false)
	m := InstallationManifest{Version: InstallationVersion, Kind: "genesis", DeploymentID: "deployment", InstallationID: "installation", TargetDigest: fixtureHash("target"), SourceApprovalRef: "approved-source", SourceApprovalDigest: fixtureHash("approval"), Recovery: f.snapshot.Installation.RecoveryInitial, Article: f.snapshot.Installation.ArticleInitial, IssuedAt: f.snapshot.Installation.InstalledAt}
	w := &installationFixture{f: f, m: m, approved: m, commitRevision: "commitment-1", anchor: m.Article, revision: "article-1"}
	w.commitment = InstallationCommitment{Manifest: m, ManifestDigest: installationOracleDigest(m), Phase: "bound"}
	f.snapshot.Installation.CommitmentDigest = w.commitment.ManifestDigest
	f.snapshot.Installation.CommitmentRef = "recovery/v1/installation/" + m.DeploymentID
	if pending {
		w.anchor.Pending = &AnchorPending{OldHead: w.anchor.Committed, NewHead: JournalHead{Sequence: 1, Hash: fixtureHash("next")}, OperationID: "article-append/v1:11111111-1111-4111-8111-111111111111:" + fixtureHash("next")[7:]}
	}
	w.r = installationRuntime(t, w)
	return w
}

func installationRuntime(t *testing.T, w *installationFixture) Runtime {
	t.Helper()
	w.f.stop = func(ctx context.Context) error { w.eventContext("drain", ctx); return nil }
	r, e := NewRuntime(context.Background(), RuntimeDependencies{Store: installationStorePort{w}, Journal: installationJournalPort{w.f, w}, ArticleSnapshots: installationSnapshotPort{w}, Closures: w.f, Genesis: w.f, Exclusive: installationExclusivePort{w}, Lifecycle: w.f, Installation: w, Commitments: w, InstallationSources: w, ArticleAnchorReader: w, InstallationWriter: w})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := r.Stop(context.Background()); e != nil {
			t.Error(e)
		}
	})
	return r
}

func installationOracleDigest(m InstallationManifest) string {
	ns := func(a AnchorSnapshot) []byte {
		return oracleObject("namespace journal_id epoch head_sequence head_hash", a.Namespace, a.JournalID, a.Epoch, a.Committed.Sequence, a.Committed.Hash)
	}
	return fixtureHash(string(oracleObject("version kind deployment_id installation_id target_digest source_approval_ref source_approval_digest recovery article issued_at", m.Version, m.Kind, m.DeploymentID, m.InstallationID, m.TargetDigest, m.SourceApprovalRef, m.SourceApprovalDigest, json.RawMessage(ns(m.Recovery)), json.RawMessage(ns(m.Article)), m.IssuedAt)))
}

func (w *installationFixture) event(name string) {
	if w.hook != nil {
		w.hook(name)
	}
}
func (w *installationFixture) eventContext(name string, ctx context.Context) {
	w.lastContext = ctx
	w.event(name)
}
func (w *installationFixture) LoadInstallation(ctx context.Context, _ string) (InstallationCommitment, string, error) {
	w.loadCount++
	w.eventContext("commitment", ctx)
	return w.commitment, w.commitRevision, w.commitErr
}
func (w *installationFixture) CompareAndSwapInstallation(context.Context, string, string, InstallationCommitment) (InstallationCommitment, string, error) {
	return InstallationCommitment{}, "", errors.New("unexpected commitment write")
}
func (w *installationFixture) VerifyInstallationSource(ctx context.Context, m InstallationManifest) error {
	w.eventContext("approval", ctx)
	if m != w.approved {
		return ErrAdmissionBinding
	}
	return w.sourceErr
}
func (w *installationFixture) Load(ctx context.Context, _ string) (AnchorSnapshot, string, error) {
	w.eventContext("anchor", ctx)
	if w.aliasAnchor {
		return w.anchor, w.revision, w.anchorErr
	}
	return copyWriterAnchor(w.anchor), w.revision, w.anchorErr
}
func (w *installationFixture) BindInstallation(context.Context, PhasePermit, InstallationBinding) error {
	return errors.New("unexpected binding write")
}
func (w *installationFixture) BindLegacyPublication(context.Context, PhasePermit, string, PublicationBinding) error {
	return errors.New("unexpected publication write")
}
func (w *installationFixture) Install(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	w.calls++
	w.saved, w.savedContext = p, ctx
	w.eventContext("executor", ctx)
	if w.execute != nil {
		return w.execute(ctx, p)
	}
	return w.complete(ctx, p, m)
}

func (w *installationFixture) complete(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	a, token, e := w.r.CheckArticleCoordination(ctx, p, m.InstallationID, m.Article.JournalID, m.Article.Epoch)
	if e != nil || token != w.revision {
		return errors.Join(ErrAdmissionBinding, e)
	}
	if a.Pending != nil {
		w.anchor.Committed = a.Pending.NewHead
		w.anchor.Pending = nil
		w.revision = "article-2"
	}
	w.f.article.Head = w.anchor.Committed
	w.f.article.Revision = w.revision
	w.f.article.Prefixes = []JournalHead{{}}
	if w.anchor.Committed.Sequence > 0 {
		w.f.article.Prefixes = append(w.f.article.Prefixes, w.anchor.Committed)
	}
	w.snapshotReady = true
	return nil
}

func testInstallPublicBound(t *testing.T) {
	w := newInstallationFixture(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e := w.r.Install(ctx, w.m); e != nil {
		t.Fatalf("public Install must reach exact bound executor: %v", e)
	}
	if w.calls != 1 || w.f.stops != 1 || w.f.starts != 0 {
		t.Fatalf("producer calls=%d drain=%d start=%d", w.calls, w.f.stops, w.f.starts)
	}
}

func installationCall(t *testing.T, w *installationFixture) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return w.r.Install(ctx, w.m)
}

func testInstallationModes(t *testing.T) {
	for _, field := range []string{"Installation", "ArticleAnchorReader", "Lifecycle", "Commitments", "InstallationSources"} {
		t.Run("MissingOrTypedNil/"+field, func(t *testing.T) {
			for _, typed := range []bool{false, true} {
				installationMissingPort(t, field, typed)
			}
		})
	}
	for _, phase := range []string{"fresh", "prepared", "bound"} {
		t.Run("Dispatch/"+phase, func(t *testing.T) { installationDispatch(t, phase) })
	}
	t.Run("NormalVerifyWithoutInstallationPorts", testInstallationNotRequired)
	t.Run("V1FlatNamespaceEncoding", func(t *testing.T) {
		w := newInstallationFixture(t, false)
		if installationManifestDigest(w.m) != installationOracleDigest(w.m) {
			t.Fatal("manifest wire changed")
		}
	})
	for _, bad := range []string{"version", "kind", "empty", "space", "utf8", "length", "target", "approval", "time", "epoch", "head", "namespace", "pending"} {
		t.Run("ManifestRejects/"+bad, func(t *testing.T) { installationInvalidManifest(t, bad) })
	}
	for _, bad := range []string{"nil", "unbounded", "cancelled", "expired"} {
		t.Run("Context/"+bad, func(t *testing.T) { installationInvalidContext(t, bad) })
	}
}

func installationMissingPort(t *testing.T, field string, typed bool) {
	t.Helper()
	w := newInstallationFixture(t, false)
	d := w.r.(*admissionRuntime).deps
	v := reflect.ValueOf(&d).Elem().FieldByName(field)
	v.SetZero()
	if typed {
		if field == "Lifecycle" {
			v.Set(reflect.ValueOf((*admissionFixture)(nil)))
		} else {
			v.Set(reflect.ValueOf((*installationFixture)(nil)))
		}
	}
	r, e := NewRuntime(context.Background(), d)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Stop(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := r.Install(ctx, w.m); !errors.Is(e, ErrAdmissionBinding) {
		t.Fatalf("missing %s=%v", field, e)
	}
	if w.f.stops != 0 || w.calls != 0 || w.loadCount != 0 {
		t.Fatal("dependency rejection started work")
	}
}

func installationDispatch(t *testing.T, phase string) {
	w := newInstallationFixture(t, false)
	if phase != "bound" {
		w.commitment.Phase = "prepared"
		if phase == "fresh" {
			w.commitErr = ErrAnchorMissing
		}
		d := w.r.(*admissionRuntime).deps
		d.InstallationWriter = nil
		r, e := NewRuntime(context.Background(), d)
		if e != nil {
			t.Fatal(e)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		e = r.Install(ctx, w.m)
		cancel()
		if !errors.Is(e, ErrAdmissionBinding) || w.f.stops != 0 || w.calls != 0 {
			t.Fatalf("writer preflight=%v", e)
		}
		if e := r.Stop(context.Background()); e != nil {
			t.Fatal(e)
		}
		w.execute = func(ctx context.Context, p PhasePermit) error {
			if e := w.r.CheckPhase(ctx, p, phaseInstall); e != nil {
				return e
			}
			w.commitErr = nil
			w.commitment.Phase = "bound"
			w.commitRevision = "bound-new"
			w.f.article.Revision = w.revision
			w.snapshotReady = true
			return nil
		}
	} else {
		w.r.(*admissionRuntime).deps.InstallationWriter = nil
	}
	if e := installationCall(t, w); e != nil || w.calls != 1 {
		t.Fatalf("%s dispatch: %v calls=%d", phase, e, w.calls)
	}
}

func testInstallationNotRequired(t *testing.T) {
	f := newAdmissionFixture(false)
	r := fixtureRuntime(t, f)
	if _, e := r.Verify(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := r.RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
}

func installationInvalidManifest(t *testing.T, bad string) {
	w := newInstallationFixture(t, false)
	switch bad {
	case "version":
		w.m.Version = "v2"
	case "kind":
		w.m.Kind = "repair"
	case "empty":
		w.m.DeploymentID = ""
	case "space":
		w.m.InstallationID = " installation"
	case "utf8":
		w.m.SourceApprovalRef = "\xff"
	case "length":
		w.m.SourceApprovalRef = strings.Repeat("x", 1025)
	case "target":
		w.m.TargetDigest = "sha256:bad"
	case "approval":
		w.m.SourceApprovalDigest = "hmac-" + w.m.SourceApprovalDigest
	case "time":
		w.m.IssuedAt = w.m.IssuedAt.Add(time.Nanosecond)
	case "epoch":
		w.m.Article.Epoch = 0
	case "head":
		w.m.Article.Committed.Hash = fixtureHash("invalid-empty-head")
	case "namespace":
		w.m.Article.Namespace = RecoveryNamespaceV1
	case "pending":
		w.m.Article.Pending = &AnchorPending{}
	}
	if e := installationCall(t, w); !errors.Is(e, ErrAdmissionBinding) {
		t.Fatalf("bad %s: %v", bad, e)
	}
	if w.calls != 0 || w.f.stops != 0 || w.loadCount != 0 {
		t.Fatal("invalid manifest caused I/O")
	}
}

func installationInvalidContext(t *testing.T, bad string) {
	w := newInstallationFixture(t, false)
	var ctx context.Context
	var cancel context.CancelFunc
	switch bad {
	case "nil":
	case "unbounded":
		ctx = context.Background()
	case "cancelled":
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		cancel()
	case "expired":
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
	}
	if e := w.r.Install(ctx, w.m); e == nil {
		t.Fatal("invalid budget accepted")
	}
	if w.calls != 0 || w.f.stops != 0 || w.loadCount != 0 {
		t.Fatal("invalid context caused I/O")
	}
}

func testInstallationSources(t *testing.T) {
	for _, next := range []string{"complete", "rollback", "already-file-H1", "committed"} {
		t.Run("ExactSuccessor/"+next, func(t *testing.T) { installationSuccess(t, next) })
	}
	for _, bad := range []string{"approval", "phase", "manifest", "digest", "commit-token", "store-digest", "store-counter", "missing-run", "recovery-pending", "recovery-ABA", "article-scope", "article-epoch", "pending-old", "pending-new", "pending-operation", "missing-anchor"} {
		t.Run("BeforeGrant/"+bad, func(t *testing.T) { installationSourceRejects(t, bad) })
	}
	t.Run("FullFailedRunIsNotIgnoredOrMutated", testInstallationFailedRun)
	t.Run("OriginalPrefixMayPrecedeCurrentHead", testInstallationExtendedHead)
}

func installationSuccess(t *testing.T, next string) {
	w := newInstallationFixture(t, next != "committed")
	if next == "already-file-H1" {
		w.f.article.Head = w.anchor.Pending.NewHead
		w.f.article.Prefixes = append(w.f.article.Prefixes, w.anchor.Pending.NewHead)
	}
	if next == "rollback" {
		w.execute = func(ctx context.Context, p PhasePermit) error {
			if _, _, e := w.r.CheckArticleCoordination(ctx, p, w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch); e != nil {
				return e
			}
			w.anchor.Pending = nil
			w.revision = "rollback-new"
			w.f.article.Revision = w.revision
			w.snapshotReady = true
			return nil
		}
	}
	// The neutral executor owns file verification; this label exercises its H1 observation.
	if e := installationCall(t, w); e != nil {
		t.Fatal(e)
	}
	if w.calls != 1 || w.f.starts != 0 {
		t.Fatal("wrong completion effects")
	}
}

func installationSourceRejects(t *testing.T, bad string) {
	w := newInstallationFixture(t, true)
	installationSourceMutation(w, bad)
	if e := installationCall(t, w); e == nil {
		t.Fatalf("bad source %s admitted", bad)
	}
	if w.calls != 0 {
		t.Fatalf("bad source %s reached executor", bad)
	}
}

func installationSourceMutation(w *installationFixture, bad string) {
	switch bad {
	case "approval":
		w.sourceErr = errors.New("unapproved source")
	case "phase":
		w.commitment.Phase = "other"
	case "manifest":
		w.commitment.Manifest.TargetDigest = fixtureHash("other")
	case "digest":
		w.commitment.ManifestDigest = fixtureHash("other")
	case "commit-token":
		w.hook = func(s string) {
			if s == "commitment" {
				w.commitRevision += "-changed"
			}
		}
	case "store-digest":
		w.badStoreDigest = true
	case "store-counter":
		w.f.hook = func(s string) {
			if s == "store" {
				w.f.snapshot.Revision.Counter++
			}
		}
	case "missing-run":
		w.f.addHistory(w.m.IssuedAt)
		w.f.snapshot.Runs = []AdmissionRun{}
	case "recovery-pending":
		w.f.recovery.Pending = &AnchorPending{}
	case "recovery-ABA":
		w.f.hook = func(s string) {
			if s == RecoveryNamespaceV1 {
				w.f.recovery.Revision += "-ABA"
			}
		}
	case "article-scope":
		w.anchor.Namespace = RecoveryNamespaceV1
	case "article-epoch":
		w.anchor.Epoch++
	case "pending-old":
		w.anchor.Pending.OldHead.Hash = fixtureHash("wrong")
	case "pending-new":
		w.anchor.Pending.NewHead.Sequence++
	case "pending-operation":
		w.anchor.Pending.OperationID = "caller-operation"
	case "missing-anchor":
		w.anchorErr = ErrAnchorMissing
	}
}

func testInstallationFailedRun(t *testing.T) {
	w := newInstallationFixture(t, false)
	w.f.addHistory(w.m.IssuedAt)
	w.f.snapshot.Runs[0].Outcome = RecoveryOutcomeFailed
	w.anchor.Committed = w.f.article.Head
	w.revision = w.f.article.Revision
	before := string(oracleSnapshot(w.f.snapshot))
	if e := installationCall(t, w); e != nil {
		t.Fatal(e)
	}
	if before != string(oracleSnapshot(w.f.snapshot)) || w.f.starts != 0 {
		t.Fatal("coordination ignored/mutated failed run")
	}
}

func testInstallationExtendedHead(t *testing.T) {
	w := newInstallationFixture(t, false)
	w.anchor.Committed = JournalHead{Sequence: 1, Hash: fixtureHash("existing-head")}
	w.f.article.Head = w.anchor.Committed
	w.f.article.Prefixes = append(w.f.article.Prefixes, w.anchor.Committed)
	if e := installationCall(t, w); e != nil {
		t.Fatal(e)
	}
}

type installationStorePort struct{ w *installationFixture }

func (p installationStorePort) ReadAdmissionSnapshot(ctx context.Context) (AdmissionSnapshot, error) {
	p.w.eventContext("store", ctx)
	s, e := p.w.f.ReadAdmissionSnapshot(ctx)
	if p.w.badStoreDigest {
		s.Revision.Digest = fixtureHash("not this full snapshot")
	}
	return s, e
}
func (p installationStorePort) ReadAdmissionRevision(ctx context.Context) (StoreRevision, error) {
	return p.w.f.ReadAdmissionRevision(ctx)
}

type installationJournalPort struct {
	ContextRecoveryJournal
	w *installationFixture
}

func (p installationJournalPort) ReadVerifiedSnapshot(ctx context.Context, ns string) (VerifiedJournalSnapshot, error) {
	p.w.eventContext("recovery", ctx)
	return p.ContextRecoveryJournal.ReadVerifiedSnapshot(ctx, ns)
}

type installationSnapshotPort struct{ w *installationFixture }

func (p installationSnapshotPort) ReadVerifiedSnapshot(ctx context.Context, ns string) (VerifiedJournalSnapshot, error) {
	p.w.eventContext("snapshot", ctx)
	if !p.w.snapshotReady {
		return VerifiedJournalSnapshot{}, ErrAdmissionClosed
	}
	return p.w.f.ReadVerifiedSnapshot(ctx, ns)
}

type installationExclusivePort struct{ w *installationFixture }

func (p installationExclusivePort) Acquire(ctx context.Context) (ExclusiveLease, error) {
	p.w.eventContext("exclusive", ctx)
	return p.w.f.Acquire(ctx)
}

func testInstallationGrant(t *testing.T) {
	for _, bad := range []string{"zero", "copied", "ordinary-context", "raw-context-mismatch", "wrong-instance", "purpose-startup", "purpose-replay", "generation", "installation", "journal", "epoch", "lost-lock"} {
		t.Run("Rejects/"+bad, func(t *testing.T) { installationBadGrant(t, bad) })
	}
	t.Run("CaptureAndReturnedPendingAreCopied", testInstallationPendingCopy)
	t.Run("TraceAndOriginalDeadline", testInstallationContextValues)
	t.Run("RepeatedCheckerReturnsOriginalStart", func(t *testing.T) {
		w := newInstallationFixture(t, true)
		start, rev := copyWriterAnchor(w.anchor), w.revision
		w.execute = func(ctx context.Context, p PhasePermit) error {
			if e := w.complete(ctx, p, w.m); e != nil {
				return e
			}
			a, v, e := w.r.CheckArticleCoordination(ctx, p, w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch)
			if e != nil || v != rev || !anchorSnapshotsEqual(a, start) {
				return ErrAdmissionDrift
			}
			return nil
		}
		if e := installationCall(t, w); e != nil {
			t.Fatal(e)
		}
	})
}

func installationBadGrant(t *testing.T, bad string) {
	w := newInstallationFixture(t, true)
	w.execute = func(ctx context.Context, p PhasePermit) error {
		target := w.r
		raw := p
		scope, id, epoch := w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch
		switch bad {
		case "zero":
			raw = PhasePermit{}
		case "copied":
			g := *p.grant
			raw = PhasePermit{grant: &g}
		case "ordinary-context":
			ctx = context.WithValue(ctx, runtimeContextKey{}, runtimeAuthorization{})
		case "raw-context-mismatch":
			ctx = context.WithValue(ctx, runtimeContextKey{}, runtimeAuthorization{phase: &phaseGrant{}})
		case "wrong-instance":
			target = newInstallationFixture(t, false).r
		case "purpose-startup", "purpose-replay":
			g := *p.grant
			g.binding.Purpose = PhaseStartupReconcile
			if bad == "purpose-replay" {
				g.binding.Purpose = PhaseRecoveryReplay
			}
			raw = PhasePermit{grant: &g}
		case "generation":
			g := *p.grant
			g.generation++
			raw = PhasePermit{grant: &g}
		case "installation":
			scope = "different"
		case "journal":
			id = "different"
		case "epoch":
			epoch++
		case "lost-lock":
			w.f.mu.Lock()
			w.f.lockErr = errors.New("lost deployment")
			w.f.mu.Unlock()
		}
		if _, _, e := target.CheckArticleCoordination(ctx, raw, scope, id, epoch); e == nil {
			t.Errorf("%s authorized", bad)
		}
		return errors.New("negative grant probe completed")
	}
	if e := installationCall(t, w); e == nil || w.calls != 1 {
		t.Fatalf("negative callback not executed: %v", e)
	}
}

func testInstallationPendingCopy(t *testing.T) {
	w := newInstallationFixture(t, true)
	w.aliasAnchor = true
	w.execute = func(ctx context.Context, p PhasePermit) error {
		original := *w.anchor.Pending
		w.anchor.Pending.NewHead.Hash = fixtureHash("aliased port mutation")
		a, _, e := w.r.CheckArticleCoordination(ctx, p, w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch)
		if e != nil || a.Pending == nil || *a.Pending != original {
			return ErrAdmissionDrift
		}
		a.Pending.OperationID = "caller mutation"
		b, _, e := w.r.CheckArticleCoordination(ctx, p, w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch)
		if e != nil || b.Pending == nil || *b.Pending != original {
			return ErrAdmissionDrift
		}
		*w.anchor.Pending = original
		return w.complete(ctx, p, w.m)
	}
	if e := installationCall(t, w); e != nil {
		t.Fatal(e)
	}
}

func testInstallationContextValues(t *testing.T) {
	w := newInstallationFixture(t, false)
	type key struct{}
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), key{}, "trace"), 5*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	w.execute = func(ctx context.Context, p PhasePermit) error {
		d, ok := ctx.Deadline()
		if !ok || d != deadline || ctx.Value(key{}) != "trace" {
			return ErrAdmissionBinding
		}
		return w.complete(ctx, p, w.m)
	}
	if e := w.r.Install(ctx, w.m); e != nil {
		t.Fatal(e)
	}
}

func testInstallationPostProof(t *testing.T) {
	for _, bad := range []string{"fake-nil", "committed-fake-nil", "H2", "reused-token", "committed-new-token", "snapshot-head", "snapshot-token", "snapshot-prefix", "store-change", "recovery-change", "commitment-change", "approval-change", "torn-readback", "unknown", "cause"} {
		t.Run(bad, func(t *testing.T) { installationBadPostProof(t, bad) })
	}
}

func installationBadPostProof(t *testing.T, bad string) {
	w := newInstallationFixture(t, bad != "committed-new-token" && bad != "committed-fake-nil")
	originalToken := w.revision
	cause := errors.New("explicit executor cause")
	w.execute = func(ctx context.Context, p PhasePermit) error {
		if bad == "fake-nil" || bad == "committed-fake-nil" {
			return nil
		}
		if bad == "unknown" {
			return errors.Join(ErrRecoveryWriteUnknown, cause)
		}
		if bad == "cause" {
			return cause
		}
		if e := w.complete(ctx, p, w.m); e != nil {
			return e
		}
		installationPostMutation(w, bad, originalToken)
		return nil
	}
	e := installationCall(t, w)
	if e == nil || w.calls != 1 {
		t.Fatalf("bad post proof %s accepted %v", bad, e)
	}
	if (bad == "cause" || bad == "unknown") && !errors.Is(e, cause) {
		t.Fatal("executor cause lost")
	}
	if bad == "unknown" && !errors.Is(e, ErrRecoveryWriteUnknown) {
		t.Fatal("write unknown lost")
	}
	if w.f.starts != 0 {
		t.Fatal("bad post proof started workers")
	}
}

func installationPostMutation(w *installationFixture, bad, old string) {
	switch bad {
	case "H2":
		w.anchor.Committed = JournalHead{Sequence: 2, Hash: fixtureHash("third-head")}
		w.f.article.Head = w.anchor.Committed
		w.f.article.Prefixes = append(w.f.article.Prefixes, w.anchor.Committed)
	case "reused-token":
		w.revision = old
		w.f.article.Revision = old
	case "committed-new-token":
		w.revision = "unexplained-new"
		w.f.article.Revision = w.revision
	case "snapshot-head":
		w.f.article.Head = JournalHead{Sequence: 1, Hash: fixtureHash("different")}
	case "snapshot-token":
		w.f.article.Revision = "different-token"
	case "snapshot-prefix":
		w.f.article.Prefixes = w.f.article.Prefixes[1:]
	case "store-change":
		w.f.snapshot.Revision.Counter++
	case "recovery-change":
		w.f.recovery.Revision += "-new"
	case "commitment-change":
		w.commitRevision += "-new"
	case "approval-change":
		w.sourceErr = errors.New("approval changed")
	case "torn-readback":
		w.hook = func(s string) {
			if s == "snapshot" {
				w.revision += "-late"
			}
		}
	}
}

func testInstallationJoin(t *testing.T) {
	for _, boundary := range []string{"approval", "commitment", "drain", "exclusive", "store", "recovery", "anchor", "executor", "snapshot"} {
		for _, stop := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stop=%t", boundary, stop), func(t *testing.T) { installationBlocked(t, boundary, stop) })
		}
	}
	for _, boundary := range []string{"approval", "anchor", "executor"} {
		t.Run("Panic/"+boundary, func(t *testing.T) { installationPanic(t, boundary) })
	}
	t.Run("NoConcurrentActionOrVerify", testInstallationConcurrent)
}

func installationBlocked(t *testing.T, boundary string, stop bool) {
	w := newInstallationFixture(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entered, release := make(chan context.Context, 1), make(chan struct{})
	var once sync.Once
	w.hook = func(s string) {
		if s == boundary {
			once.Do(func() { entered <- w.lastContext; <-release })
		}
	}
	done := make(chan error, 1)
	go func() { done <- w.r.Install(ctx, w.m) }()
	given := installationWaitContext(t, entered)
	var stopped chan error
	if stop {
		stopped = make(chan error, 1)
		go func() { stopped <- w.r.Stop(ctx) }()
	} else {
		cancel()
	}
	installationAwaitRevocation(t, w, boundary, given, done, stopped)
	close(release)
	if e := <-done; e == nil {
		t.Error("late callback returned success")
	}
	if stop {
		if e := <-stopped; e != nil {
			t.Error(e)
		}
	}
}

func installationAwaitRevocation(t *testing.T, w *installationFixture, boundary string, given context.Context, done, stopped <-chan error) {
	t.Helper()
	select {
	case <-given.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("port context not cancelled")
	}
	w.f.mu.Lock()
	closes := w.f.closes
	w.f.mu.Unlock()
	if closes != 0 {
		t.Error("deployment closed before actual port joined")
	}
	if boundary == "executor" || boundary == "snapshot" {
		if _, _, e := w.r.CheckArticleCoordination(given, w.saved, w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch); e == nil {
			t.Error("grant survived cancellation")
		}
	}
	select {
	case e := <-done:
		t.Errorf("abandoned port: %v", e)
	default:
	}
	if stopped != nil {
		select {
		case e := <-stopped:
			t.Errorf("Stop abandoned port: %v", e)
		default:
		}
	}
}

func installationWaitContext(t *testing.T, c <-chan context.Context) context.Context {
	t.Helper()
	select {
	case ctx := <-c:
		return ctx
	case <-time.After(2 * time.Second):
		t.Fatal("expected port not reached")
		return nil
	}
}

func installationPanic(t *testing.T, boundary string) {
	w := newInstallationFixture(t, true)
	panicValue := &struct{ label string }{boundary}
	w.hook = func(s string) {
		if s == boundary {
			panic(panicValue)
		}
	}
	var caught any
	func() { defer func() { caught = recover() }(); _ = installationCall(t, w) }()
	if caught != panicValue {
		t.Fatalf("panic identity changed: %v", caught)
	}
	if e := w.r.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	w.f.mu.Lock()
	closes := w.f.closes
	w.f.mu.Unlock()
	want := 1
	if boundary == "approval" {
		want = 0
	}
	if closes != want {
		t.Fatalf("panic cleanup closes=%d want=%d", closes, want)
	}
	if w.savedContext != nil && w.savedContext.Err() == nil {
		t.Fatal("panic escaped live context")
	}
}

func testInstallationConcurrent(t *testing.T) {
	w := newInstallationFixture(t, true)
	entered, release := make(chan context.Context, 1), make(chan struct{})
	w.execute = func(ctx context.Context, p PhasePermit) error {
		entered <- ctx
		<-release
		return w.complete(ctx, p, w.m)
	}
	done := make(chan error, 1)
	go func() { done <- installationCall(t, w) }()
	ctx := installationWaitContext(t, entered)
	if e := w.r.Install(ctx, w.m); !errors.Is(e, ErrAdmissionBusy) {
		t.Errorf("parallel Install=%v", e)
	}
	if e := w.r.RunNormal(ctx); !errors.Is(e, ErrAdmissionBusy) {
		t.Errorf("parallel normal=%v", e)
	}
	if _, e := w.r.Verify(ctx); e == nil {
		t.Error("Verify entered Install")
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}

func testInstallationClosed(t *testing.T) {
	t.Run("SuccessNeverOpensOrRestarts", testInstallationNeverOpens)
	t.Run("PreservesUnrelatedCloseCause", func(t *testing.T) {
		w := newInstallationFixture(t, true)
		r := w.r.(*admissionRuntime)
		// An explicit closed-state oracle, not a capability producer.
		token := &appendToken{}
		cause := CloseCause{Code: "unrelated-close", OperationID: "unrelated"}
		r.causes[token] = cause
		if e := installationCall(t, w); e != nil {
			t.Fatal(e)
		}
		r.mu.Lock()
		same := r.causes[token] == cause
		r.mu.Unlock()
		if !same {
			t.Fatal("Install cleared unrelated close cause")
		}
	})
	t.Run("DrainFailureBeforeExclusiveOrExecutor", func(t *testing.T) {
		w := newInstallationFixture(t, true)
		cause := errors.New("maintenance child not joined")
		w.f.stop = func(context.Context) error { return cause }
		if e := installationCall(t, w); !errors.Is(e, cause) {
			t.Fatal("drain cause lost")
		}
		if w.calls != 0 {
			t.Fatal("unjoined drain reached executor")
		}
	})
}

func testInstallationNeverOpens(t *testing.T) {
	w := newInstallationFixture(t, true)
	if e := installationCall(t, w); e != nil {
		t.Fatal(e)
	}
	if w.f.starts != 0 || w.f.stops != 1 {
		t.Fatal("Install incorrectly started workers or skipped cold drain")
	}
	if _, e := w.r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("Install opened public work")
	}
	if _, _, e := w.r.CheckArticleCoordination(w.savedContext, w.saved, w.m.InstallationID, w.m.Article.JournalID, w.m.Article.Epoch); e == nil {
		t.Fatal("completed grant still valid")
	}
	if e := w.r.RunNormal(context.Background()); !errors.Is(e, ErrAdmissionBusy) {
		t.Fatalf("one-shot reset=%v", e)
	}
	w.f.mu.Lock()
	before := w.f.closes
	w.f.mu.Unlock()
	if before != 0 {
		t.Fatal("Install prematurely released deployment")
	}
	if e := w.r.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	w.f.mu.Lock()
	after := w.f.closes
	w.f.mu.Unlock()
	if after != 1 {
		t.Fatalf("Stop closes=%d", after)
	}
}

func TestRecoveryRuntimeAdmission(t *testing.T) {
	t.Run("PublicEntryProducerAndModeDependencies", func(t *testing.T) {
		t.Run("Enter", testControlledEnter)
		t.Run("Resume", testControlledResume)
		t.Run("ModeDependencies", testModeDependencies)
		t.Run("ConsumedMissingStoreRejectsBind", regressionR02ConsumedMissingStoreRejectsBind)
		t.Run("BindHistoryCrossProduct", testR03BindHistory)
		t.Run("IndependentPublicEntryPositive", regressionR02IndependentPublicEntryPositive)
		t.Run("MissingActionPorts", testMissingActionPorts)
	})
	t.Run("ResumeBeginSecretWriteFence", testResumeSecret)
	t.Run("RecoveryExitBeforeStartup", testControlledExit)
	t.Run("FullHistoryPriority", testFullHistory)
	t.Run("GenesisAndExactHistory", testGenesisHistory)
	t.Run("IndependentRevisionABA", func(t *testing.T) {
		testRevisionABA(t)
		t.Run("RestoreRevisionABARejectsBeforeBind", regressionR02RestoreRevisionABARejectsBeforeBind)
		t.Run("ControlledSourceFences", testR03SourceFences)
	})
	t.Run("PermitGenerationAndOwnCause", testPermitCauses)
	t.Run("DriftCancellationAndNoCachedOpen", testRuntimeRevocation)
}

func testRevisionABA(t *testing.T) {
	t.Run("IndependentNamespaceTokens", testIndependentNamespaceTokens)
	for _, name := range []string{"RecoveryABA", "ArticleABA", "StoreCounter", "StoreDigest"} {
		t.Run(name, func(t *testing.T) {
			f := newAdmissionFixture(true)
			f.hook = func(event string) {
				if event != "revision" {
					return
				}
				f.mu.Lock()
				defer f.mu.Unlock()
				switch name {
				case "RecoveryABA":
					f.recovery.Revision += "-new"
				case "ArticleABA":
					f.article.Revision += "-new"
				case "StoreCounter":
					f.snapshot.Revision.Counter++
				case "StoreDigest":
					f.snapshot.Runs[0].AccountCount++
				}
			}
			r := fixtureRuntime(t, f)
			if e := r.RunNormal(context.Background()); !errors.Is(e, ErrAdmissionDrift) {
				t.Fatalf("drift = %v", e)
			}
			if f.snapshots != 2 {
				t.Fatalf("attempts=%d, want 2", f.snapshots)
			}
		})
	}
	t.Run("OneCompleteRetry", testOneCompleteRetry)
}

func normalFixture(t *testing.T) (*admissionFixture, Runtime) {
	t.Helper()
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	if e := r.RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
	return f, r
}

func fixtureAdvance(f *admissionFixture, operation string) VerifiedHeadAdvance {
	return VerifiedHeadAdvance{JournalID: f.article.JournalID, Namespace: ArticleDeletionNamespaceV1, Epoch: f.article.Epoch, OldHead: f.article.Head, NewHead: JournalHead{Sequence: f.article.Head.Sequence + 1, Hash: fixtureHash(operation)}, OldRevision: f.article.Revision, PendingRevision: "pending-" + operation, CommittedRevision: "committed-" + operation, OperationID: operation}
}

func testPermitCauses(t *testing.T) {
	t.Run("DefaultClosedAndConstructorPure", testDefaultClosedAndConstructorPure)
	t.Run("PhaseScopeAndTrace", testPhaseScopeAndTrace)
	t.Run("LeaseReleaseRevokesContext", testLeaseReleaseRevokesContext)
	t.Run("HealthyAppendOnlyOwnCause", testHealthyAppendOnlyOwnCause)
	t.Run("AppendDoesNotWaitForSelf", testAppendDoesNotWaitForSelf)
	t.Run("WrongBarrierOrHead", testWrongBarrierOrHead)
	t.Run("PhaseJournalMatrix", testPhaseJournalMatrix)
	t.Run("WorkerWaitsTemporaryBarrier", testWorkerBarrier)
}

func testControlledEnter(t *testing.T) {
	c, r := newControlledFixture(t, "")
	secret := []byte("synthetic-candidate")
	if e := r.Enter(context.Background(), RecoveryInput{RunBinding: c.binding(), Secret: secret}); e != nil {
		t.Fatal(e)
	}
	if c.begins != 1 || c.replaces != 1 || c.consumes != 1 || c.revokes != 1 {
		t.Fatalf("incomplete public path %v", c.steps)
	}
	want := []string{"prepare", "begin", "replace", "verify-restored", "bind", "finalize", "trust", "identity-ready", "consume", "exit", "revoke"}
	if !reflect.DeepEqual(c.steps, want) {
		t.Fatalf("order %v", c.steps)
	}
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	l.Release()
}

func testControlledResume(t *testing.T) {
	for _, stage := range []RecoveryOutcome{RecoveryOutcomeRunning, RecoveryOutcomeIdentityReady, RecoveryOutcomeSucceeded} {
		t.Run(string(stage), func(t *testing.T) {
			c, r := newControlledFixture(t, stage)
			if e := r.Resume(context.Background(), c.binding()); e != nil {
				t.Fatal(e)
			}
			if c.begins != 0 || c.replaces != 0 {
				t.Fatal("Resume repeated Begin/Replace")
			}
			if stage == RecoveryOutcomeSucceeded && c.consumes != 0 {
				t.Fatal("succeeded Resume consumed again")
			}
			if stage != RecoveryOutcomeRunning && (c.secretReads != 0 || c.secretVerifies != 0) {
				t.Fatal("prepared Resume read/verified secret")
			}
		})
	}
}

func testModeDependencies(t *testing.T) {
	f := newAdmissionFixture(false)
	d := RuntimeDependencies{Store: f, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: f}
	r, e := NewRuntime(context.Background(), d)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Stop(context.Background())
	if _, e := r.Verify(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := r.RunNormal(context.Background()); !errors.Is(e, ErrAdmissionBinding) {
		t.Fatalf("missing Lifecycle=%v", e)
	}
	if e := r.Enter(context.Background(), RecoveryInput{RunBinding: RunBinding{RunID: "run", BackupRef: "backup"}, Secret: []byte("synthetic")}); !errors.Is(e, ErrAdmissionBinding) {
		t.Fatalf("missing Enter ports=%v", e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, step := range f.ledger {
		if step == "begin" {
			t.Fatal("missing ports allowed Begin")
		}
	}
}

func testControlledExit(t *testing.T) {
	c, r := newControlledFixture(t, RecoveryOutcomeIdentityReady)
	c.exitEntered = make(chan struct{})
	c.exitRelease = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- r.Resume(context.Background(), c.binding()) }()
	select {
	case <-c.exitEntered:
	case e := <-done:
		t.Fatalf("public Resume did not reach Exit: %v", e)
	}
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Error("Exit opened before join")
	}
	c.f.mu.Lock()
	started := c.f.starts
	c.f.mu.Unlock()
	if started != 0 {
		t.Error("workers started before Exit")
	}
	close(c.exitRelease)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	t.Run("UnknownExitStaysClosed", func(t *testing.T) {
		c, r := newControlledFixture(t, RecoveryOutcomeIdentityReady)
		c.exitErr = ErrRecoveryWriteUnknown
		if e := r.Resume(context.Background(), c.binding()); e == nil {
			t.Fatal("unknown exit accepted")
		}
		if c.f.starts != 0 {
			t.Fatal("unknown exit started workers")
		}
	})
}

func testResumeSecret(t *testing.T) {
	t.Run("PreparedOrChangedDoesNotReadSecret", testPreparedResume)
	t.Run("RetainedBeginEntry", func(t *testing.T) {
		c, r := newControlledFixture(t, RecoveryOutcomeRunning)
		if e := r.Resume(context.Background(), c.binding()); e != nil {
			t.Fatal(e)
		}
		if c.secretReads != 1 || c.secretVerifies != 1 {
			t.Fatal("secret gate bypassed")
		}
	})
	for _, kind := range []string{"missing-key", "wrong-entry", "wrong-run", "wrong-backup", "wrong-time", "wrong-secret", "cancel", "lost-lock", "ABA"} {
		t.Run(kind, func(t *testing.T) {
			c, r := newControlledFixture(t, RecoveryOutcomeRunning)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mutateVerifier(c, kind, cancel)
			if e := r.Resume(ctx, c.binding()); e == nil {
				t.Fatal("bad verifier accepted")
			}
			for _, b := range c.lastCandidate {
				if b != 0 {
					t.Fatal("candidate not cleared")
				}
			}
			if c.begins != 0 || c.replaces != 0 {
				t.Fatal("verifier failure reissued Begin/Replace")
			}
			for _, step := range c.steps {
				if step == "trust" {
					t.Fatal("invalid verifier reached trust write")
				}
			}
		})
	}
}

func mutateVerifier(c *controlledFixture, kind string, cancel context.CancelFunc) {
	switch kind {
	case "missing-key":
		c.verifyErr = ErrAdmissionIncomplete
	case "wrong-entry":
		c.verifierEntry = func(e JournalEntry) JournalEntry { e.KeyVersion = "new-active"; return e }
	case "wrong-run":
		c.verifierEntry = func(e JournalEntry) JournalEntry { e.RunID = "wrong"; return e }
	case "wrong-backup":
		c.verifierEntry = func(e JournalEntry) JournalEntry { e.BackupRef = "wrong"; return e }
	case "wrong-time":
		c.verifierEntry = func(e JournalEntry) JournalEntry { e.RecordedAt = e.RecordedAt.Add(time.Microsecond); return e }
	case "wrong-secret":
		c.secretCandidate = []byte("synthetic-wrong")
	case "cancel":
		c.verifierEntry = func(e JournalEntry) JournalEntry { cancel(); return e }
	case "lost-lock":
		c.verifierEntry = func(e JournalEntry) JournalEntry { c.f.lockErr = errors.New("synthetic lost lock"); return e }
	case "ABA":
		c.verifierEntry = func(e JournalEntry) JournalEntry { c.f.recovery.Revision += "-ABA"; return e }
	}
}

func testPreparedResume(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprint(changed), func(t *testing.T) {
			c, r := newControlledFixture(t, RecoveryOutcomeRunning)
			for i := range c.f.snapshot.Runs {
				x := &c.f.snapshot.Runs[i]
				if x.RunID == c.complete.RunID {
					x.InitialAdminUserID = c.complete.InitialAdminUserID
					x.InitialAdminBootstrapGeneration = c.complete.InitialAdminBootstrapGeneration
					if changed {
						x.InitialAdminChangedGeneration = c.complete.InitialAdminChangedGeneration
						x.InitialAdminChangeAuditID = c.complete.InitialAdminChangeAuditID
						x.InitialAdminChangedAt = c.complete.InitialAdminChangedAt
					}
				}
			}
			if e := r.Resume(context.Background(), c.binding()); e != nil {
				t.Fatal(e)
			}
			if c.secretReads != 0 || c.secretVerifies != 0 {
				t.Fatal("prepared secret was read/verified")
			}
			for _, s := range c.steps {
				if s == "trust" {
					t.Fatal("prepared password overwritten")
				}
			}
		})
	}
}

func testMissingActionPorts(t *testing.T) {
	for _, field := range []string{"Runs", "Identity", "Restore", "Consumer", "Exit", "Lifecycle", "RecoverySecret"} {
		t.Run(field, func(t *testing.T) {
			c, api := newControlledFixture(t, RecoveryOutcomeRunning)
			d := api.(*admissionRuntime).deps
			reflect.ValueOf(&d).Elem().FieldByName(field).SetZero()
			r, e := NewRuntime(context.Background(), d)
			if e != nil {
				t.Fatal(e)
			}
			if e := r.Resume(context.Background(), c.binding()); !errors.Is(e, ErrAdmissionBinding) {
				t.Fatalf("preflight=%v", e)
			}
			if len(c.steps) != 0 {
				t.Fatalf("missing port performed work: %v", c.steps)
			}
			if e := r.Stop(context.Background()); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func testRuntimeRevocation(t *testing.T) {
	t.Run("R01TemporaryVerifyAcquire", regressionTemporaryAcquire)
	t.Run("R01VerifyDrainBoundaries", regressionVerifyDrainBoundaries)
	t.Run("R01TemporaryVerifyLease", regressionTemporaryVerifyLease)
	t.Run("R01VerifyFaultDoesNotJoinItself", regressionVerifyOwnFault)
	t.Run("R01KnownAppendRetryBound", regressionKnownAppendBound)
	t.Run("R01UnnotifiedAdvance", regressionUnnotifiedAdvance)
	t.Run("R01StopCancelsStartupPort", regressionStopCancelsStartupPort)
	t.Run("R01StopJoinsVerify", regressionStopJoinsVerify)
	t.Run("R01VerifyAcceptsConcurrentKnownAppend", regressionVerifyAcceptsConcurrentKnownAppend)
	t.Run("R01StartupAppendPositiveControl", regressionStartupAppendPositiveControl)
	t.Run("R01StopCancelsProofPort", regressionStopCancelsProofPort)
	t.Run("LostExclusiveRevokesGeneration", testLostExclusiveRevokesGeneration)
	t.Run("LateSuccessCannotOpen", testLateSuccessCannotOpen)
	t.Run("CancelledStartupCannotOpen", testCancelledStartupCannotOpen)
	t.Run("RestartDoesNotCacheOpen", testRestartDoesNotCacheOpen)
	t.Run("DependencyFailureAfterOpen", testDependencyFailureAfterOpen)
	t.Run("SourceChangeRevokesInsteadOfReprovingOpen", testOpenSourceDrift)
	t.Run("DrainWaitsForActualLeaseRelease", testActualDrain)
	t.Run("CancelledStopStillJoins", testCancelledStop)
	t.Run("StoppedStartupCannotReturnLateOpen", testStoppedStartup)
	t.Run("LifecycleFailureCloses", testLifecycleFailureCloses)
}

type heldExclusive struct {
	*admissionFixture
	entered chan context.Context
	release chan struct{}
}

func (e *heldExclusive) Acquire(ctx context.Context) (ExclusiveLease, error) {
	e.entered <- ctx
	<-e.release
	return fixtureExclusive{e.admissionFixture}, nil
}

func regressionTemporaryAcquire(t *testing.T) {
	f := newAdmissionFixture(true)
	exclusive := &heldExclusive{admissionFixture: f, entered: make(chan context.Context, 1), release: make(chan struct{})}
	r, e := NewRuntime(context.Background(), RuntimeDependencies{Store: f, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: exclusive, Lifecycle: f})
	if e != nil {
		t.Fatal(e)
	}
	verified := make(chan error, 1)
	go func() { _, err := r.Verify(context.Background()); verified <- err }()
	given := <-exclusive.entered
	if e := r.RunNormal(context.Background()); !errors.Is(e, ErrAdmissionBusy) {
		t.Errorf("mode started during Verify acquire: %v", e)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	<-given.Done()
	select {
	case <-stopped:
		t.Error("Stop abandoned temporary acquire")
	default:
	}
	close(exclusive.release)
	if e := <-verified; e == nil {
		t.Error("cancelled temporary acquire admitted")
	}
	if e := <-stopped; e != nil {
		t.Error(e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closes != 1 {
		t.Errorf("late-acquired lease closes=%d", f.closes)
	}
}

func regressionVerifyDrainBoundaries(t *testing.T) {
	for _, boundary := range []string{RecoveryNamespaceV1, ArticleDeletionNamespaceV1, "store", "closure", "revision"} {
		t.Run(boundary, func(t *testing.T) {
			f, r := normalFixture(t)
			entered, release, stopping := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var once sync.Once
			f.mu.Lock()
			f.hook = func(event string) {
				if event == boundary {
					once.Do(func() { close(entered); <-release })
				}
			}
			f.stop = func(context.Context) error { close(stopping); return nil }
			f.mu.Unlock()
			verified := make(chan error, 1)
			go func() { _, err := r.Verify(context.Background()); verified <- err }()
			<-entered
			stopped := make(chan error, 1)
			go func() { stopped <- r.Stop(context.Background()) }()
			<-stopping
			if _, e := r.Verify(context.Background()); !errors.Is(e, ErrAdmissionClosed) {
				t.Errorf("new Verify during drain: %v", e)
			}
			select {
			case <-stopped:
				t.Error("Stop abandoned proof")
			default:
			}
			f.mu.Lock()
			closes := f.closes
			f.mu.Unlock()
			if closes != 0 {
				t.Error("lease closed before proof joined")
			}
			close(release)
			if e := <-verified; e == nil {
				t.Error("stopped Verify succeeded")
			}
			if e := <-stopped; e != nil {
				t.Error(e)
			}
		})
	}
}

type heldAdmissionStore struct {
	*admissionFixture
	entered chan context.Context
	release chan struct{}
}

func (s *heldAdmissionStore) ReadAdmissionSnapshot(ctx context.Context) (AdmissionSnapshot, error) {
	s.entered <- ctx
	<-s.release
	return s.admissionFixture.ReadAdmissionSnapshot(ctx)
}

func regressionTemporaryVerifyLease(t *testing.T) {
	f := newAdmissionFixture(true)
	store := &heldAdmissionStore{admissionFixture: f, entered: make(chan context.Context, 1), release: make(chan struct{})}
	r, e := NewRuntime(context.Background(), RuntimeDependencies{Store: store, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: f, Lifecycle: f})
	if e != nil {
		t.Fatal(e)
	}
	verified := make(chan error, 1)
	go func() { _, err := r.Verify(context.Background()); verified <- err }()
	given := <-store.entered
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	<-given.Done()
	if _, e := r.Verify(context.Background()); !errors.Is(e, ErrAdmissionClosed) {
		t.Errorf("temporary drain admitted Verify: %v", e)
	}
	f.mu.Lock()
	closes := f.closes
	f.mu.Unlock()
	if closes != 0 {
		t.Error("temporary lease closed before port returned")
	}
	select {
	case <-stopped:
		t.Error("temporary Verify was not joined")
	default:
	}
	close(store.release)
	if e := <-verified; e == nil {
		t.Error("cancelled proof accepted")
	}
	if e := <-stopped; e != nil {
		t.Error(e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closes != 1 {
		t.Errorf("temporary lease closes=%d", f.closes)
	}
}

func regressionVerifyOwnFault(t *testing.T) {
	f, r := normalFixture(t)
	f.mu.Lock()
	f.readErr = errors.New("synthetic proof fault")
	f.mu.Unlock()
	if _, e := r.Verify(context.Background()); e == nil {
		t.Fatal("missing proof fault")
	}
	if e := r.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closes != 1 {
		t.Fatal("self-fault did not join/release")
	}
}

func regressionKnownAppendBound(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkArticleDeletion)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	f.mu.Lock()
	before := f.snapshots
	f.mu.Unlock()
	attempt := 0
	f.hook = func(event string) {
		if event == "revision" {
			attempt++
			op := fmt.Sprintf("known-%d", attempt)
			reviewerCommitAppend(t, f, r, l.Context(), op)
		}
	}
	if _, e := r.Verify(context.Background()); !errors.Is(e, ErrAdmissionDrift) {
		t.Fatalf("bound not enforced: %v", e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshots-before != 2 {
		t.Fatalf("Verify complete attempts=%d", f.snapshots-before)
	}
}

func regressionUnnotifiedAdvance(t *testing.T) {
	f, r := normalFixture(t)
	a := fixtureAdvance(f, "unnotified")
	f.mu.Lock()
	f.article.Head = a.NewHead
	f.article.Revision = a.CommittedRevision
	f.article.Prefixes = append(f.article.Prefixes, a.NewHead)
	f.closure.CurrentHead = a.NewHead
	f.closure.VerifiedArticleRevision = a.CommittedRevision
	f.mu.Unlock()
	if _, e := r.Verify(context.Background()); !errors.Is(e, ErrAdmissionDrift) {
		t.Fatalf("unknown head accepted: %v", e)
	}
}

func testCancelledStop(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{})
	f.mu.Lock()
	f.stop = func(context.Context) error { close(entered); return nil }
	f.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- r.Stop(ctx) }()
	select {
	case <-entered:
	case err := <-done:
		l.Release()
		t.Fatalf("cancelled Stop returned without initiating real join: %v", err)
	}
	select {
	case <-done:
		l.Release()
		t.Fatal("cancelled Stop abandoned work")
	default:
	}
	l.Release()
	if e := <-done; !errors.Is(e, context.Canceled) {
		t.Fatalf("cancellation not preserved: %v", e)
	}
}

func testPhaseJournalMatrix(t *testing.T) {
	for _, purpose := range []string{"startup_reconcile", "recovery_replay", "recovery_identity", "coordinate"} {
		t.Run(purpose, func(t *testing.T) {
			_, api := normalFixture(t)
			r := api.(*admissionRuntime)
			// Exercise the private phase predicate after a complete real admission proof.
			// This fixture is not evidence of the later recovery execution workflow.
			r.mu.Lock()
			p := r.phaseLocked(purpose)
			r.mu.Unlock()
			ctx, e := r.PhaseContext(context.Background(), p)
			if e != nil {
				t.Fatal(e)
			}
			appendAllowed := r.CheckJournalUse(ctx, "op", "new_append") == nil
			if appendAllowed != (purpose == "startup_reconcile") {
				t.Fatal("wrong append purpose")
			}
			coordinateAllowed := r.CheckJournalUse(ctx, "op", "coordinate") == nil
			if coordinateAllowed != (purpose == "coordinate") {
				t.Fatal("wrong coordinate purpose")
			}
			if e := r.CheckJournalUse(ctx, "op", "committed_read"); e != nil {
				t.Fatal(e)
			}
			forged := *p.grant
			forged.binding.RunID = "cross-run"
			if _, e := r.PhaseContext(context.Background(), PhasePermit{grant: &forged}); e == nil {
				t.Fatal("cross-run permit accepted")
			}
		})
	}
}

func testWorkerBarrier(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	b, e := r.PauseArticleAppend(l.Context(), "worker-op")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		lease WorkLease
		err   error
	}
	done := make(chan result, 1)
	go func() { w, e := r.Acquire(ctx, WorkPasswordReset); done <- result{w, e} }()
	select {
	case got := <-done:
		if got.lease != nil {
			got.lease.Release()
		}
		t.Fatalf("temporary barrier returned terminal result: %v", got.err)
	default:
	}
	if e := r.CompleteArticleAppend(l.Context(), b, fixtureAdvance(f, "worker-op")); e != nil {
		t.Fatal(e)
	}
	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	got.lease.Release()
}

func testOpenSourceDrift(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	f.mu.Lock()
	f.snapshot.Revision.Counter++
	f.mu.Unlock()
	if obs, e := r.Verify(context.Background()); !errors.Is(e, ErrAdmissionDrift) || obs.State != "closed" {
		t.Fatalf("unexpected drift result %+v %v", obs, e)
	}
	if _, e := r.ReadPermitBinding(l.Context()); e == nil {
		t.Fatal("old capability survived source drift")
	}
}

func testActualDrain(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	stopping := make(chan struct{})
	f.mu.Lock()
	f.stop = func(context.Context) error { close(stopping); return nil }
	f.mu.Unlock()
	done := make(chan error, 1)
	go func() { done <- r.Stop(context.Background()) }()
	<-stopping
	<-l.Context().Done()
	select {
	case <-done:
		t.Fatal("Stop returned before lease release")
	default:
	}
	f.mu.Lock()
	closed := f.closes
	f.mu.Unlock()
	if closed != 0 {
		t.Fatal("exclusive released before join")
	}
	l.Release()
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stops != 1 || f.closes != 1 {
		t.Fatalf("stops/closes=%d/%d", f.stops, f.closes)
	}
}

func testStoppedStartup(t *testing.T) {
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	started, release := make(chan struct{}), make(chan struct{})
	f.start = func(context.Context, PhasePermit) error { close(started); <-release; return nil }
	normal := make(chan error, 1)
	go func() { normal <- r.RunNormal(context.Background()) }()
	<-started
	stopEntered := make(chan struct{})
	f.mu.Lock()
	f.stop = func(context.Context) error { close(stopEntered); return nil }
	f.mu.Unlock()
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	<-stopEntered
	select {
	case <-stopped:
		t.Fatal("Stop did not join startup")
	default:
	}
	close(release)
	if e := <-normal; e == nil {
		t.Fatal("late startup returned open")
	}
	if e := <-stopped; e != nil {
		t.Fatal(e)
	}
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("late startup reopened")
	}
}

func testIndependentNamespaceTokens(t *testing.T) {
	f := newAdmissionFixture(true)
	if e := fixtureRuntime(t, f).RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
}

func testOneCompleteRetry(t *testing.T) {
	f := newAdmissionFixture(false)
	once := sync.Once{}
	f.hook = func(event string) {
		if event == "revision" {
			once.Do(func() { f.mu.Lock(); f.snapshot.Revision.Counter++; f.mu.Unlock() })
		}
	}
	r := fixtureRuntime(t, f)
	if e := r.RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
	if f.snapshots != 2 {
		t.Fatalf("attempts=%d", f.snapshots)
	}
}

func testDefaultClosedAndConstructorPure(t *testing.T) {
	f := newAdmissionFixture(false)
	r := fixtureRuntime(t, f)
	if len(f.ledger) != 0 {
		t.Fatal("constructor accessed ports")
	}
	if l, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		l.Release()
		t.Fatal("new runtime open")
	}
	if _, e := NewRuntime(nil, RuntimeDependencies{}); e == nil {
		t.Fatal("nil constructor accepted")
	}
}

func testPhaseScopeAndTrace(t *testing.T) {
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	other := fixtureRuntime(t, newAdmissionFixture(true))
	type traceKey struct{}
	ctx := context.WithValue(context.Background(), traceKey{}, "trace")
	var saved PhasePermit
	f.start = func(c context.Context, p PhasePermit) error {
		saved = p
		if e := r.CheckPhase(c, p, "startup_reconcile"); e != nil {
			return e
		}
		pc, e := r.PhaseContext(c, p)
		if e != nil {
			return e
		}
		if pc.Value(traceKey{}) != "trace" {
			t.Error("trace lost")
		}
		if e := r.CheckJournalUse(pc, "operation", "new_append"); e != nil {
			t.Error(e)
		}
		if e := r.CheckPhase(c, p, "recovery_identity"); e == nil {
			t.Error("wrong phase")
		}
		if e := other.CheckPhase(c, p, "startup_reconcile"); e == nil {
			t.Error("wrong instance")
		}
		return nil
	}
	if e := r.RunNormal(ctx); e != nil {
		t.Fatal(e)
	}
	if e := r.CheckPhase(ctx, saved, "startup_reconcile"); e == nil {
		t.Fatal("expired phase accepted")
	}
	if _, e := r.PhaseContext(ctx, PhasePermit{}); e == nil {
		t.Fatal("zero permit accepted")
	}
}

func testLeaseReleaseRevokesContext(t *testing.T) {
	_, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	if e := r.CheckJournalUse(l.Context(), "operation", "new_append"); e != nil {
		t.Error(e)
	}
	b, e := r.ReadPermitBinding(l.Context())
	if e != nil || b.RuntimeID == "" || b.Generation <= 0 || b.RunID != "run-a" {
		t.Errorf("binding=%+v %v", b, e)
	}
	l.Release()
	l.Release()
	if e := r.CheckJournalUse(l.Context(), "operation", "new_append"); e == nil {
		t.Fatal("released lease authorized")
	}
}

func testHealthyAppendOnlyOwnCause(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkArticleDeletion)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	b, e := r.PauseArticleAppend(l.Context(), "op-a")
	if e != nil {
		t.Fatal(e)
	}
	other, e := r.PauseArticleAppend(l.Context(), "op-b")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("barrier admitted request")
	}
	if e := r.CompleteArticleAppend(l.Context(), b, fixtureAdvance(f, "op-a")); e != nil {
		t.Fatal(e)
	}
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("cleared other cause")
	}
	if e := r.CompleteArticleAppend(l.Context(), b, fixtureAdvance(f, "op-a")); e == nil {
		t.Fatal("duplicate completion")
	}
	if e := r.FaultArticleAppend(l.Context(), other, errors.New("synthetic unknown")); e == nil {
		t.Fatal("fault must retain error")
	}
}

func testAppendDoesNotWaitForSelf(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkArticleDeletion)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	b, e := r.PauseArticleAppend(l.Context(), "op")
	if e != nil {
		t.Fatal(e)
	}
	if e := r.CompleteArticleAppend(l.Context(), b, fixtureAdvance(f, "op")); e != nil {
		t.Fatal(e)
	}
	next, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	next.Release()
}

func testWrongBarrierOrHead(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	b, e := r.PauseArticleAppend(l.Context(), "op")
	if e != nil {
		t.Fatal(e)
	}
	a := fixtureAdvance(f, "op")
	a.OldRevision = "wrong"
	if e := r.CompleteArticleAppend(l.Context(), b, a); e == nil {
		t.Fatal("wrong revision accepted")
	}
	if e := r.CompleteArticleAppend(l.Context(), AppendBarrier{}, a); e == nil {
		t.Fatal("zero barrier accepted")
	}
}

func testLostExclusiveRevokesGeneration(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	f.mu.Lock()
	f.lockErr = errors.New("synthetic lost lock")
	f.mu.Unlock()
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("lost lock admitted")
	}
	select {
	case <-l.Context().Done():
	default:
		t.Fatal("old work not cancelled")
	}
	if e := r.CheckJournalUse(l.Context(), "op", "new_append"); e == nil {
		t.Fatal("old generation authorized")
	}
}

func testLateSuccessCannotOpen(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	b, e := r.PauseArticleAppend(l.Context(), "op")
	if e != nil {
		t.Fatal(e)
	}
	_ = r.FaultArticleAppend(l.Context(), b, errors.New("synthetic unknown"))
	if e := r.CompleteArticleAppend(context.Background(), b, fixtureAdvance(f, "op")); e == nil {
		t.Fatal("late success accepted")
	}
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("fault reopened")
	}
}

func testCancelledStartupCannotOpen(t *testing.T) {
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	f.start = func(context.Context, PhasePermit) error { cancel(); return nil }
	if e := r.RunNormal(ctx); e == nil {
		t.Fatal("cancelled startup admitted")
	}
}

func testRestartDoesNotCacheOpen(t *testing.T) {
	_, r := normalFixture(t)
	fresh := fixtureRuntime(t, newAdmissionFixture(true))
	if _, e := fresh.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("cached admission crossed restart")
	}
	if e := r.CheckJournalUse(context.Background(), "op", "new_append"); e == nil {
		t.Fatal("arbitrary ctx authorized")
	}
}

func testDependencyFailureAfterOpen(t *testing.T) {
	f, r := normalFixture(t)
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	f.mu.Lock()
	f.readErr = errors.New("synthetic unavailable")
	f.mu.Unlock()
	if _, e := r.Verify(context.Background()); e == nil {
		t.Fatal("dependency failure ignored")
	}
	select {
	case <-l.Context().Done():
	default:
		t.Fatal("fault did not cancel work")
	}
}

func testLifecycleFailureCloses(t *testing.T) {
	f := newAdmissionFixture(true)
	f.startErr = errors.New("synthetic startup error")
	r := fixtureRuntime(t, f)
	if e := r.RunNormal(context.Background()); !errors.Is(e, f.startErr) {
		t.Fatalf("cause lost: %v", e)
	}
	if _, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		t.Fatal("failed lifecycle open")
	}
}

// Reviewer probes exercise public operations only; private fixture state is an explicit neutral port oracle.
func regressionStopCancelsStartupPort(t *testing.T) {
	f := newAdmissionFixture(true)
	entered, release, stopping := make(chan context.Context, 1), make(chan struct{}), make(chan struct{})
	f.start = func(c context.Context, _ PhasePermit) error { entered <- c; <-release; return c.Err() }
	f.stop = func(context.Context) error { close(stopping); return nil }
	r := fixtureRuntime(t, f)
	normal := make(chan error, 1)
	go func() { normal <- r.RunNormal(context.Background()) }()
	given := <-entered
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	<-stopping
	if given.Err() == nil {
		t.Error("runtime Stop revoked generation but startup port context is still live; a cooperative startup blocked on ctx.Done cannot join")
	}
	close(release)
	if err := <-normal; err == nil {
		t.Error("stopped startup returned success")
	}
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
}

func regressionStopJoinsVerify(t *testing.T) {
	f, r := normalFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.mu.Lock()
	f.hook = func(event string) {
		if event == "closure" {
			once.Do(func() { close(entered); <-release })
		}
	}
	f.mu.Unlock()
	verified := make(chan error, 1)
	go func() { _, err := r.Verify(context.Background()); verified <- err }()
	<-entered
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	early := false
	timer := time.NewTimer(time.Second)
	select {
	case err := <-stopped:
		early = true
		if err != nil {
			t.Error(err)
		}
		f.mu.Lock()
		closes := f.closes
		f.mu.Unlock()
		t.Errorf("Stop returned and exclusive closes=%d while Verify is blocked in the closure port", closes)
	case <-timer.C:
	}
	timer.Stop()
	close(release)
	<-verified
	if !early {
		if err := <-stopped; err != nil {
			t.Error(err)
		}
	}
}

func reviewerCommitAppend(t *testing.T, f *admissionFixture, r Runtime, ctx context.Context, op string) {
	t.Helper()
	b, err := r.PauseArticleAppend(ctx, op)
	if err != nil {
		t.Fatal(err)
	}
	a := fixtureAdvance(f, op)
	f.mu.Lock()
	f.article.Head = a.NewHead
	f.article.Revision = a.CommittedRevision
	f.article.Prefixes = append(f.article.Prefixes, a.NewHead)
	f.closure.CurrentHead = a.NewHead
	f.closure.VerifiedArticleRevision = a.CommittedRevision
	f.mu.Unlock()
	if err := r.CompleteArticleAppend(ctx, b, a); err != nil {
		t.Fatal(err)
	}
}

func regressionVerifyAcceptsConcurrentKnownAppend(t *testing.T) {
	f, r := normalFixture(t)
	l, err := r.Acquire(context.Background(), WorkArticleDeletion)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.mu.Lock()
	f.hook = func(event string) {
		if event == "revision" {
			once.Do(func() { close(entered); <-release })
		}
	}
	f.mu.Unlock()
	verified := make(chan error, 1)
	go func() { _, err := r.Verify(context.Background()); verified <- err }()
	<-entered
	reviewerCommitAppend(t, f, r, l.Context(), "known-concurrent")
	close(release)
	if err := <-verified; err != nil {
		t.Errorf("runtime faults a valid append whose exact HeadAdvance it already accepted: %v", err)
	}
	if _, err := r.ReadPermitBinding(l.Context()); err != nil {
		t.Errorf("accepted append revoked its still-owned work: %v", err)
	}
}

func regressionStartupAppendPositiveControl(t *testing.T) {
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	f.start = func(c context.Context, p PhasePermit) error {
		pc, err := r.PhaseContext(c, p)
		if err != nil {
			return err
		}
		reviewerCommitAppend(t, f, r, pc, "startup-append")
		if _, err := r.Acquire(c, WorkRequest); err == nil {
			t.Error("startup append opened public work")
		}
		return nil
	}
	if err := r.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
	l, err := r.Acquire(context.Background(), WorkRequest)
	if err != nil {
		t.Fatal(err)
	}
	l.Release()
}

type reviewerBlockedAdmissionStore struct {
	*admissionFixture
	entered chan context.Context
	release chan struct{}
}

func (s *reviewerBlockedAdmissionStore) ReadAdmissionSnapshot(c context.Context) (AdmissionSnapshot, error) {
	s.entered <- c
	select {
	case <-c.Done():
		return AdmissionSnapshot{}, c.Err()
	case <-s.release:
	}
	return s.admissionFixture.ReadAdmissionSnapshot(c)
}
func regressionStopCancelsProofPort(t *testing.T) {
	f := newAdmissionFixture(true)
	stopping := make(chan struct{})
	f.stop = func(context.Context) error { close(stopping); return nil }
	store := &reviewerBlockedAdmissionStore{admissionFixture: f, entered: make(chan context.Context, 1), release: make(chan struct{})}
	r, err := NewRuntime(context.Background(), RuntimeDependencies{Store: store, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: f, Lifecycle: f})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	normal := make(chan error, 1)
	go func() { normal <- r.RunNormal(context.Background()) }()
	given := <-store.entered
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	<-stopping
	if given.Err() == nil {
		t.Error("Stop revoked generation but in-flight pure Store proof read context is still live; read has no PhasePermit from which to derive cancellation")
	}
	close(store.release)
	if err := <-normal; err == nil {
		t.Error("stopped proof admitted")
	}
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
}

type r02BindObserver struct {
	*controlledFixture
	binds int
}

func (w *r02BindObserver) BindRestoredRun(ctx context.Context, p PhasePermit, b RunBinding, x RestoreReceipt) (RecoveryRun, error) {
	w.binds++
	return w.controlledFixture.BindRestoredRun(ctx, p, b, x)
}
func regressionR02ConsumedMissingStoreRejectsBind(t *testing.T) {
	c, old := newControlledFixture(t, RecoveryOutcomeSucceeded)
	c.f.snapshot.Runs = c.f.snapshot.Runs[:1]
	c.f.snapshot.Revision.Counter++
	w := &r02BindObserver{controlledFixture: c}
	deps := old.(*admissionRuntime).deps
	deps.Identity = w
	r, e := NewRuntime(context.Background(), deps)
	if e != nil {
		t.Fatal(e)
	}
	c.r = r
	defer r.Stop(context.Background())
	err := r.Resume(context.Background(), c.binding())
	if err == nil {
		t.Error("consumed Journal with missing current Store run admitted")
	}
	if w.binds != 0 {
		t.Errorf("invalid consumed/missing-row state received recovery_bind and reached %d BindRestoredRun write calls before rejection: %v", w.binds, err)
	}
}

type r02RestoreDrift struct{ *controlledFixture }

func (w *r02RestoreDrift) VerifyRestored(ctx context.Context, p RestorePlan) (RestoreReceipt, error) {
	x, e := w.controlledFixture.VerifyRestored(ctx, p)
	w.f.recovery.Revision += "-unexplained-ABA"
	return x, e
}
func regressionR02RestoreRevisionABARejectsBeforeBind(t *testing.T) {
	c, old := newControlledFixture(t, "")
	w := &r02BindObserver{controlledFixture: c}
	deps := old.(*admissionRuntime).deps
	deps.Identity = w
	deps.Restore = &r02RestoreDrift{c}
	r, e := NewRuntime(context.Background(), deps)
	if e != nil {
		t.Fatal(e)
	}
	c.r = r
	defer r.Stop(context.Background())
	err := r.Enter(context.Background(), RecoveryInput{RunBinding: c.binding(), Secret: []byte("synthetic-candidate")})
	if err == nil {
		t.Error("unexplained recovery namespace revision ABA during restore was accepted and admission opened")
	}
	if w.binds != 0 {
		t.Errorf("unexplained revision drift reached %d BindRestoredRun calls; steps=%v", w.binds, c.steps)
	}
}

func regressionR02IndependentPublicEntryPositive(t *testing.T) {
	t.Run("EnterCurrentRunNotHistoryLast", func(t *testing.T) {
		c, r := newControlledFixture(t, "")
		if err := r.Enter(context.Background(), RecoveryInput{RunBinding: c.binding(), Secret: []byte("synthetic-candidate")}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.steps, []string{"prepare", "begin", "replace", "verify-restored", "bind", "finalize", "trust", "identity-ready", "consume", "exit", "revoke"}) {
			t.Fatalf("wrong public phase/owner chain: %v", c.steps)
		}
		l, err := r.Acquire(context.Background(), WorkRequest)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Release()
		binding, err := r.ReadPermitBinding(l.Context())
		if err != nil || binding.RunID != c.complete.RunID || binding.BackupRef != c.complete.BackupRef {
			t.Fatalf("public binding=%+v err=%v", binding, err)
		}
	})
	t.Run("ResumeSucceededOnlyExit", func(t *testing.T) {
		c, r := newControlledFixture(t, RecoveryOutcomeSucceeded)
		if err := r.Resume(context.Background(), c.binding()); err != nil {
			t.Fatal(err)
		}
		if c.begins != 0 || c.replaces != 0 || c.consumes != 0 || c.secretReads != 0 || c.secretVerifies != 0 || c.revokes != 1 {
			t.Fatalf("succeeded resumed wrong effects: %v", c.steps)
		}
	})
}

func testR03SourceFences(t *testing.T) {
	t.Run("BeginExactSuccessor", testR03BeginFence)
	boundaries := []struct {
		at, next string
		resume   bool
	}{
		{"prepare", "replace", false}, {"replace", "verify-restored", false},
		{"verify-restored", "bind", false}, {"bind", "finalize", false},
		{"finalize", "trust", false}, {"trust", "identity-ready", false},
		{"identity-ready", "consume", false}, {"exit", "startup", false},
		{"prepare", "verify-restored", true}, {"verify-restored", "trust", true},
		{"secret-read", "trust", true},
	}
	for _, b := range boundaries {
		for _, namespace := range []string{"recovery", "article"} {
			t.Run(fmt.Sprintf("%s/resume=%t/%s/revision", b.at, b.resume, namespace), func(t *testing.T) {
				r03SourceFence(t, b.at, b.next, b.resume, namespace, "revision")
			})
		}
	}
	for _, kind := range []string{"scope", "epoch", "head", "entries"} {
		for _, at := range []string{"verify-restored", "identity-ready", "exit"} {
			for _, namespace := range []string{"recovery", "article"} {
				t.Run(at+"/"+namespace+"/"+kind, func(t *testing.T) {
					r03SourceFence(t, at, "", false, namespace, kind)
				})
			}
		}
	}
	t.Run("ReadOnlyVerifierRejectsStoreRevision", func(t *testing.T) { r03SourceFence(t, "verify-restored", "bind", false, "recovery", "store-revision") })
	t.Run("ConsumedReconcileKeepsJournal", func(t *testing.T) { r03ConsumedBoundary(t, "") })
	for _, ns := range []string{"recovery", "article"} {
		t.Run("ConsumedReconcileRejectsABA/"+ns, func(t *testing.T) { r03ConsumedBoundary(t, ns) })
	}
	t.Run("ConsumeRejectsArticleABA", func(t *testing.T) { r03SourceFence(t, "consume", "exit", false, "article", "revision") })
	t.Run("ConsumeRejectsExtraRecoveryHead", func(t *testing.T) { r03SourceFence(t, "consume", "exit", false, "recovery", "head") })
	testR03StoreFences(t)
}

func testR03StoreFences(t *testing.T) {
	t.Run("RestoreMayChangeTargetStore", func(t *testing.T) {
		c, old := newControlledFixture(t, "")
		w, r := r03Runtime(t, c, old)
		w.hook = func(name string) {
			if name == "replace" {
				c.f.snapshot.Revision.Counter++
			}
		}
		if e := r.Enter(context.Background(), RecoveryInput{RunBinding: c.binding(), Secret: []byte("synthetic-candidate")}); e != nil {
			t.Fatal(e)
		}
		if w.calls["bind"] != 1 || c.revokes != 1 {
			t.Fatalf("owned Store changes lost: %v", c.steps)
		}
	})
	for _, at := range []string{"bind", "finalize", "trust", "identity-ready", "consume", "exit"} {
		t.Run("StoreRollback/"+at, func(t *testing.T) { r03SourceFence(t, at, "", false, "recovery", "store-rollback") })
	}
	t.Run("BoundRunCannotDisappear", func(t *testing.T) { r03SourceFence(t, "finalize", "trust", false, "recovery", "missing-row") })
}

func testR03BeginFence(t *testing.T) {
	for _, kind := range []string{"article-ABA", "reused-token", "wrong-scope", "wrong-epoch", "extra-head", "changed-prefix"} {
		t.Run(kind, func(t *testing.T) {
			c, old := newControlledFixture(t, "")
			w, r := r03Runtime(t, c, old)
			base := c.f.recovery.Revision
			rt := r.(*admissionRuntime)
			rt.deps.Journal = r03BeginJournal{ContextRecoveryJournal: rt.deps.Journal, after: func() {
				switch kind {
				case "article-ABA":
					c.f.article.Revision += "-unexplained"
				case "reused-token":
					c.f.recovery.Revision = base
				case "wrong-scope":
					c.f.recovery.JournalID = "other-journal"
				case "wrong-epoch":
					c.f.recovery.Epoch++
				case "extra-head":
					r03Mutate(c, "recovery", "head")
				case "changed-prefix":
					c.f.recovery.Entries[0].RecordHMAC = fixtureHash("rewritten-history")
				}
			}}
			if e := r.Enter(context.Background(), RecoveryInput{RunBinding: c.binding(), Secret: []byte("synthetic-candidate")}); e == nil {
				t.Fatal("unaccounted Begin successor accepted")
			}
			if c.begins != 1 || w.calls["replace"] != 0 {
				t.Fatalf("Begin fence order: %v", c.steps)
			}
			r03AssertClosed(t, c, r)
		})
	}
}

func r03Mutate(c *controlledFixture, namespace, kind string) {
	s := &c.f.recovery
	if namespace == "article" {
		s = &c.f.article
	}
	switch kind {
	case "revision":
		s.Revision += "-unexplained-ABA"
	case "scope":
		s.Namespace = "wrong-namespace"
	case "epoch":
		s.Epoch++
	case "head":
		s.Head = JournalHead{Sequence: s.Head.Sequence + 1, Hash: fixtureHash("unaccounted-head")}
		s.Prefixes = append(s.Prefixes, s.Head)
		s.Revision += "-unaccounted-head"
	case "entries":
		if len(s.Entries) > 0 {
			s.Entries[0].RecordHMAC = fixtureHash("changed-history")
		} else {
			s.Prefixes[0].Hash = fixtureHash("changed-prefix")
		}
	case "store-revision":
		c.f.snapshot.Revision.Counter++
	case "store-rollback":
		c.f.snapshot.Revision.Counter = 1
	case "missing-row":
		c.f.snapshot.Runs = c.f.snapshot.Runs[:1]
		c.f.snapshot.Revision.Counter++
	}
}

func r03SourceFence(t *testing.T, at, next string, resume bool, namespace, kind string) {
	t.Helper()
	stage := RecoveryOutcome("")
	if resume {
		stage = RecoveryOutcomeRunning
	}
	c, old := newControlledFixture(t, stage)
	w, r := r03Runtime(t, c, old)
	w.hook = func(name string) {
		if name == at {
			r03Mutate(c, namespace, kind)
		}
	}
	var e error
	if resume {
		e = r.Resume(context.Background(), c.binding())
	} else {
		e = r.Enter(context.Background(), RecoveryInput{RunBinding: c.binding(), Secret: []byte("synthetic-candidate")})
	}
	if e == nil {
		t.Fatalf("%s/%s at %s opened; steps=%v", namespace, kind, at, c.steps)
	}
	if kind == "revision" && !errors.Is(e, ErrAdmissionDrift) {
		t.Fatalf("ABA classification: %v", e)
	}
	if w.calls[at] != 1 {
		t.Fatalf("injection not reached: %s, calls=%v", at, w.calls)
	}
	if next != "" && w.calls[next] != 0 {
		t.Fatalf("drift reached %s: %v", next, w.calls)
	}
	if at == "verify-restored" && w.calls["bind"] != 0 {
		t.Fatal("verifier drift authorized bind")
	}
	if at == "identity-ready" && w.calls["consume"] != 0 {
		t.Fatal("identity drift authorized Consume")
	}
	r03AssertClosed(t, c, r)
}

func r03AssertClosed(t *testing.T, c *controlledFixture, r Runtime) {
	t.Helper()
	if e := r.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
	if c.f.starts != 0 {
		t.Fatal("invalid source reached startup")
	}
	if l, e := r.Acquire(context.Background(), WorkRequest); e == nil {
		l.Release()
		t.Fatal("invalid source admitted work")
	}
	for _, p := range c.saved {
		if e := r.CheckPhase(context.Background(), p, p.grant.binding.Purpose); e == nil {
			t.Fatal("old grant survived")
		}
	}
}

func r03Reconcile(c *controlledFixture, ctx context.Context, id string) (ConsumptionResult, error) {
	b, e := c.r.ReadPermitBinding(ctx)
	if e != nil || b.Purpose != PhaseRecoveryReplay || id != c.complete.RunID {
		return ConsumptionResult{}, ErrAdmissionBinding
	}
	c.steps = append(c.steps, "consume")
	c.consumes++
	c.putStage(RecoveryOutcomeSucceeded)
	x, e := c.f.Load(ctx, id)
	return ConsumptionResult{Run: x, Evidence: c.closure.Ready, Journal: c.f.recovery.Entries[len(c.f.recovery.Entries)-1]}, e
}

func r03ConsumedBoundary(t *testing.T, namespace string) {
	c, old := newControlledFixture(t, RecoveryOutcomeSucceeded)
	c.f.snapshot.Runs[1].Outcome = RecoveryOutcomeReadyToConsume
	c.f.snapshot.Runs[1].CompletedAt = nil
	c.f.snapshot.Revision.Counter++
	w, r := r03Runtime(t, c, old)
	w.consume = func(ctx context.Context, id string) (ConsumptionResult, error) { return r03Reconcile(c, ctx, id) }
	if namespace != "" {
		w.hook = func(name string) {
			if name == "consume" {
				r03Mutate(c, namespace, "revision")
			}
		}
	}
	e := r.Resume(context.Background(), c.binding())
	if namespace == "" {
		if e != nil || c.consumes != 1 || c.revokes != 1 {
			t.Fatalf("exact consumed reconciliation: %v, %v", e, c.steps)
		}
		return
	}
	if !errors.Is(e, ErrAdmissionDrift) || w.calls["exit"] != 0 {
		t.Fatalf("consumed ABA: %v, calls=%v", e, w.calls)
	}
	r03AssertClosed(t, c, r)
}

func testR03BindHistory(t *testing.T) {
	for _, present := range []bool{false, true} {
		for _, consumed := range []bool{false, true} {
			outcomes := []RecoveryOutcome{RecoveryOutcomeRunning, RecoveryOutcomeIdentityReady, RecoveryOutcomeArticleDeletionReady, RecoveryOutcomeReadyToConsume, RecoveryOutcomeJournalConsumed, RecoveryOutcomeSucceeded, RecoveryOutcomeFailed}
			if !present {
				outcomes = []RecoveryOutcome{"no-row"}
			}
			for _, outcome := range outcomes {
				t.Run(fmt.Sprintf("present=%t/consumed=%t/%s", present, consumed, outcome), func(t *testing.T) { r03BindHistoryCase(t, present, consumed, outcome) })
			}
		}
	}
	for _, bad := range []string{"run", "backup", "installation", "receipt", "cancel"} {
		t.Run("UnboundReceiptRejects/"+bad, func(t *testing.T) { r03UnboundRejects(t, bad) })
	}
}

func r03BindHistoryCase(t *testing.T, present, consumed bool, outcome RecoveryOutcome) {
	c, old := newControlledFixture(t, RecoveryOutcomeSucceeded)
	x := c.f.snapshot.Runs[1]
	x.Outcome = outcome
	if !consumed {
		c.f.recovery.Entries = c.f.recovery.Entries[:len(c.f.recovery.Entries)-1]
		c.f.recovery.Prefixes = c.f.recovery.Prefixes[:len(c.f.recovery.Prefixes)-1]
		c.f.recovery.Head = c.f.recovery.Prefixes[len(c.f.recovery.Prefixes)-1]
		c.f.recovery.Revision += "-fixture-begun"
		x.JournalStage, x.JournalConsumedAt = StageBegun, nil
	}
	if outcome != RecoveryOutcomeSucceeded {
		x.CompletedAt = nil
	}
	c.f.snapshot.Runs[1] = x
	if !present {
		c.f.snapshot.Runs = c.f.snapshot.Runs[:1]
	}
	c.f.snapshot.Revision.Counter++
	w, r := r03Runtime(t, c, old)
	if consumed {
		w.consume = func(ctx context.Context, id string) (ConsumptionResult, error) { return r03Reconcile(c, ctx, id) }
	}
	e := r.Resume(context.Background(), c.binding())
	if r03ValidHistory(present, consumed, outcome) {
		if e != nil {
			t.Fatalf("valid receipt/stage rejected: %v, steps=%v", e, c.steps)
		}
		wantBind := 0
		if !present {
			wantBind = 1
		}
		if w.calls["bind"] != wantBind {
			t.Fatalf("bind=%d want %d", w.calls["bind"], wantBind)
		}
		if c.begins != 0 || c.replaces != 0 {
			t.Fatal("Resume repeated Begin/Replace")
		}
		return
	}
	if e == nil {
		t.Fatal("invalid history accepted")
	}
	for _, name := range []string{"bind", "finalize", "trust", "consume", "exit"} {
		if w.calls[name] != 0 {
			t.Fatalf("invalid history invoked %s", name)
		}
	}
	r03AssertClosed(t, c, r)
}

func r03ValidHistory(present, consumed bool, outcome RecoveryOutcome) bool {
	if !present {
		return !consumed
	}
	if consumed {
		return outcome == RecoveryOutcomeReadyToConsume || outcome == RecoveryOutcomeJournalConsumed || outcome == RecoveryOutcomeSucceeded
	}
	return outcome == RecoveryOutcomeRunning || outcome == RecoveryOutcomeIdentityReady || outcome == RecoveryOutcomeArticleDeletionReady || outcome == RecoveryOutcomeReadyToConsume
}

func r03UnboundRejects(t *testing.T, bad string) {
	c, old := newControlledFixture(t, RecoveryOutcomeRunning)
	c.f.snapshot.Runs = c.f.snapshot.Runs[:1]
	c.f.snapshot.Revision.Counter++
	w, r := r03Runtime(t, c, old)
	b := c.binding()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	switch bad {
	case "run":
		b.RunID = "wrong-run"
	case "backup":
		b.BackupRef = "wrong-backup"
	case "installation":
		c.plan.InstallationID = "wrong-installation"
	case "receipt":
		c.complete.RestoreBinding.RunID = "wrong-run"
	case "cancel":
		w.hook = func(name string) {
			if name == "verify-restored" {
				cancel()
			}
		}
	}
	if e := r.Resume(ctx, b); e == nil {
		t.Fatal("invalid receipt/cancellation accepted")
	}
	for _, name := range []string{"bind", "finalize", "trust", "consume", "exit"} {
		if w.calls[name] != 0 {
			t.Fatalf("invalid binding invoked %s", name)
		}
	}
	r03AssertClosed(t, c, r)
}

func TestRecoveryRuntime_ShutdownContext_PureAndFirst(t *testing.T) {
	for _, mode := range []string{"stop", "cancelled-stop", "fault"} {
		t.Run(mode, func(t *testing.T) { testShutdownFirstContext(t, mode) })
	}
}

func testShutdownFirstContext(t *testing.T, mode string) {
	t.Helper()
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	assertShutdownUnselected(t, r)
	owner := context.WithValue(context.Background(), shutdownQueryKey{}, "owner")
	if err := r.RunNormal(owner); err != nil {
		t.Fatal(err)
	}
	work, err := r.Acquire(context.Background(), WorkArticleNotice)
	if err != nil {
		t.Fatal(err)
	}
	defer work.Release()
	entered, proceed := make(chan context.Context, 1), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(proceed) }) }
	defer unblock()
	f.mu.Lock()
	f.stop = func(c context.Context) error { entered <- c; <-proceed; return nil }
	f.mu.Unlock()
	first, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if mode == "cancelled-stop" {
		cancel()
	}
	stopped := make(chan error, 1)
	if mode == "fault" {
		first = owner
		barrier, err := r.PauseArticleAppend(work.Context(), "shutdown-query-fault")
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("synthetic append failure")
		if err := r.FaultArticleAppend(work.Context(), barrier, cause); !errors.Is(err, cause) {
			t.Fatal(err)
		}
	} else {
		go func() { stopped <- r.Stop(first) }()
	}
	<-work.Context().Done()
	assertShutdownSelected(t, r, first)
	if got := <-entered; got != first {
		t.Error("drain did not receive the selected context")
	}
	joinShutdownQuery(t, r, first, work, unblock, stopped, mode)
}

func assertShutdownSelected(t *testing.T, r Runtime, first context.Context) {
	t.Helper()
	for range 3 {
		if c, ok := r.ShutdownContext(); !ok || c != first {
			t.Error("revoked work cannot read the exact first shutdown context")
		}
	}
}

type shutdownQueryKey struct{}

func assertShutdownUnselected(t *testing.T, r Runtime) {
	t.Helper()
	for range 3 {
		if c, ok := r.ShutdownContext(); c != nil || ok {
			t.Fatal("query selected shutdown on a closed runtime")
		}
	}
}

func joinShutdownQuery(t *testing.T, r Runtime, first context.Context, work WorkLease, unblock func(), stopped chan error, mode string) {
	t.Helper()
	later, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	repeated := make(chan error, 1)
	go func() { repeated <- r.Stop(later) }()
	readers := make(chan struct{}, 8)
	for range 8 {
		go func() {
			defer func() { readers <- struct{}{} }()
			for range 8 {
				if c, ok := r.ShutdownContext(); !ok || c != first {
					t.Error("concurrent query changed shutdown context")
				}
			}
		}()
	}
	for range 8 {
		<-readers
	}
	select {
	case err := <-repeated:
		t.Errorf("repeated Stop abandoned actual work: %v", err)
	default:
	}
	work.Release()
	unblock()
	if mode != "fault" {
		if err := <-stopped; !errors.Is(err, first.Err()) {
			t.Error(err)
		}
	}
	if err := <-repeated; err != nil {
		t.Error(err)
	}
	if c, ok := r.ShutdownContext(); !ok || c != first {
		t.Error("Stop replaced the selected context")
	}
}

type shutdownCloseFailure struct {
	f   *admissionFixture
	err error
}

func (e shutdownCloseFailure) Acquire(context.Context) (ExclusiveLease, error) {
	return shutdownCloseFailureLease{e}, nil
}

type shutdownCloseFailureLease struct{ shutdownCloseFailure }

func (e shutdownCloseFailureLease) Check(c context.Context) error {
	return (fixtureExclusive{e.f}).Check(c)
}

func (e shutdownCloseFailureLease) Close() error {
	return errors.Join((fixtureExclusive{e.f}).Close(), e.err)
}

func TestRecoveryRuntime_ShutdownContext_ColdProofCleanupFault(t *testing.T) {
	f := newAdmissionFixture(true)
	owner := context.WithValue(context.Background(), shutdownQueryKey{}, "constructor-owner")
	cause := errors.New("synthetic temporary ownership close failure")
	r, err := NewRuntime(owner, RuntimeDependencies{Store: f, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: shutdownCloseFailure{f, cause}, Lifecycle: f})
	if err != nil {
		t.Fatal(err)
	}
	if c, selected := r.ShutdownContext(); c != nil || selected {
		t.Fatal("constructor selected shutdown")
	}
	if _, err := r.Verify(context.Background()); !errors.Is(err, cause) {
		t.Error(err)
	}
	if c, selected := r.ShutdownContext(); !selected || c != owner {
		t.Error("cold cleanup fault lost its existing constructor owner")
	}
	if err := r.Stop(context.Background()); !errors.Is(err, cause) {
		t.Error(err)
	}
}

func TestRecoveryRuntime_AcquireReady_ExplicitHealthyPause(t *testing.T) {
	for _, kind := range []WorkKind{WorkArticleDeletion, WorkPasswordReset, WorkArticleNotice, WorkRequest} {
		t.Run(string(kind), func(t *testing.T) {
			f, api := normalFixture(t)
			r := api.(*admissionRuntime)
			held, err := api.Acquire(context.Background(), WorkRequest)
			if err != nil {
				t.Fatal(err)
			}
			defer held.Release()
			barrier, err := api.PauseArticleAppend(held.Context(), "explicit-pause")
			if err != nil {
				t.Fatal(err)
			}
			lease, paused, err := r.acquireReady(context.Background(), kind)
			if lease != nil {
				lease.Release()
				t.Error("healthy pause granted work")
			}
			if kind == WorkRequest {
				if paused != nil || !errors.Is(err, ErrAdmissionClosed) {
					t.Errorf("request pause=%v error=%v", paused, err)
				}
			} else if paused == nil || err != nil {
				t.Errorf("ordinary pause=%v error=%v", paused, err)
			}
			if err := api.CompleteArticleAppend(held.Context(), barrier, fixtureAdvance(f, "explicit-pause")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRecoveryRuntime_ShutdownContext_ParentStartupAndWatchJoin(t *testing.T) {
	f := newAdmissionFixture(true)
	entered := make(chan context.Context, 1)
	f.start = func(c context.Context, _ PhasePermit) error {
		entered <- c
		<-c.Done()
		return c.Err()
	}
	r := fixtureRuntime(t, f)
	owner, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan error, 1)
	go func() { started <- r.RunNormal(owner) }()
	phase := <-entered
	cancel()
	<-phase.Done()
	if shared, selected := r.ShutdownContext(); !selected || shared != owner {
		t.Error("startup phase canceled before parent context was published")
	}
	if err := <-started; !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-r.(*admissionRuntime).ownerWatch.done:
	default:
		t.Error("Stop returned before its owner cancellation callback actually joined")
	}
}

func TestRecoveryRuntime_ShutdownContext_SynchronousOwnerCheck(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			owner, cancel := context.WithCancel(context.Background())
			if mode == "deadline" {
				cancel()
				owner, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
			}
			defer cancel()
			f := newAdmissionFixture(true)
			api := fixtureRuntime(t, f)
			if err := api.RunNormal(owner); err != nil {
				t.Fatal(err)
			}
			r := api.(*admissionRuntime)
			r.mu.Lock() // Prevent the asynchronous owner watcher from doing the work.
			if mode == "cancel" {
				cancel()
			} else {
				<-owner.Done()
			}
			if err := r.generationCtx.Err(); !errors.Is(err, owner.Err()) {
				t.Error("synchronous cancellation check accepted a canceled owner", err)
			}
			select {
			case <-r.generationCtx.Done():
			default:
				t.Error("Err returned cancellation before Done closed")
			}
			if c, selected := r.ShutdownContext(); !selected || c != owner {
				t.Error("synchronous cancellation did not publish its exact owner first")
			}
			r.mu.Unlock()
			if err := api.Stop(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestRecoveryRuntime_ShutdownContext_CancellationErrorStaysFirst(t *testing.T) {
	f := newAdmissionFixture(true)
	api := fixtureRuntime(t, f)
	owner, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := api.RunNormal(owner); err != nil {
		t.Fatal(err)
	}
	first := context.Background()
	if err := api.Stop(first); err != nil {
		t.Fatal(err)
	}
	generation := api.(*admissionRuntime).generationCtx
	if err := generation.Err(); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	<-owner.Done()
	if err := generation.Err(); !errors.Is(err, context.Canceled) {
		t.Error("later owner deadline changed the first generation cancellation error", err)
	}
	if selected, ok := api.ShutdownContext(); !ok || selected != first {
		t.Error("later owner deadline changed first shutdown context")
	}
}

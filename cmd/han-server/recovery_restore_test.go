package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/recovery"
)

var s414Boundary = errors.New("driver component boundary reached")

const s414Canary = "synthetic-s414-private-canary"

const s414EvidenceDirEnv = "HAN_S414_TEST_EVIDENCE_DIR"

// This dispatch exists only in the current test binary, never the server image.
func TestMain(m *testing.M) {
	runtime.GOMAXPROCS(2)
	if len(os.Args) == 2 && (os.Args[1] == "restore-v1" || os.Args[1] == "verify-v1") {
		os.Exit(s414Child())
	}
	os.Exit(m.Run())
}

func s414Child() int {
	var input struct {
		Plan   restorePlanWire `json:"plan"`
		Target struct {
			DSN string `json:"dsn"`
		} `json:"target_connection"`
	}
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, restoreReceiptLimit+1))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || input.Target.DSN == "" {
		return 21
	}
	for _, entry := range os.Environ() {
		if strings.Contains(entry, s414Canary) {
			return 22
		}
	}
	p := s414PlanFromWire(input.Plan)
	raw := s414ReceiptJSON(p)
	os.Stderr.WriteString(s414Canary)
	switch p.BackupRef {
	case "child-oversize":
		raw = append(raw, bytes.Repeat([]byte(" "), restoreReceiptLimit+1-len(raw))...)
	case "child-invalid":
		raw = []byte("null")
	case "child-exit":
		return 23
	}
	if _, err := os.Stdout.Write(raw); err != nil {
		return 24
	}
	// The hold case deliberately emits a receipt before EOF; a receipt is not exit.
	if p.BackupRef == "child-hold" {
		_, _ = io.Copy(io.Discard, decoder.Buffered())
		_, _ = io.Copy(io.Discard, os.Stdin)
	}
	return 0
}

func s414PlanFromWire(w restorePlanWire) recovery.RestorePlan {
	return recovery.RestorePlan{Version: w.Version, RunID: w.RunID, BackupRef: w.BackupRef, DeploymentID: w.DeploymentID, InstallationID: w.InstallationID, TargetDigest: w.TargetDigest, ArtifactDigest: w.ArtifactDigest, ArtifactBytes: w.ArtifactBytes, SourceApprovalRef: w.SourceApprovalRef, SourceApprovalDigest: w.SourceApprovalDigest, SourceInstallationID: w.SourceInstallationID, SourceSchemaDigest: w.SourceSchemaDigest, ArtifactPath: w.ArtifactPath}
}

func s414ReceiptJSON(p recovery.RestorePlan) []byte {
	// Independent oracle with the contract's explicit ordered fields, not driver encoder.
	fields := []struct {
		key   string
		value any
	}{
		{"version", p.Version}, {"run_id", p.RunID}, {"backup_ref", p.BackupRef}, {"deployment_id", p.DeploymentID}, {"installation_id", p.InstallationID},
		{"target_digest", p.TargetDigest}, {"artifact_digest", p.ArtifactDigest}, {"artifact_bytes", p.ArtifactBytes}, {"source_approval_digest", p.SourceApprovalDigest},
		{"restored_schema_digest", p.SourceSchemaDigest}, {"restored_at", "2026-10-29T00:00:00.123456Z"},
	}
	raw := []byte("{")
	for i, f := range fields {
		if i > 0 {
			raw = append(raw, ',')
		}
		k, _ := json.Marshal(f.key)
		v, _ := json.Marshal(f.value)
		raw = append(raw, k...)
		raw = append(raw, ':')
		raw = append(raw, v...)
	}
	canonical := append(append([]byte{}, raw...), '}')
	digest, _ := json.Marshal(recovery.SHA256Digest(canonical))
	raw = append(raw, []byte(",\"receipt_digest\":")...)
	raw = append(raw, digest...)
	return append(raw, '}')
}

func s414Directory(t *testing.T) string {
	t.Helper()
	root := os.Getenv(s414EvidenceDirEnv)
	if root == "" {
		return t.TempDir()
	}
	dir, err := os.MkdirTemp(root, "han-s414-retain-")
	if err != nil {
		t.Fatalf("create retained fixture under root %q from %s: %v", root, s414EvidenceDirEnv, err)
	}
	t.Logf("retained_fixture=%s", dir)
	return dir
}

type s414Source struct {
	plan    recovery.RestorePlan
	handle  *os.File
	checks  int
	failure error
}

func (s *s414Source) OpenApproved(ctx context.Context, b recovery.RunBinding) (recovery.RestorePlan, *os.File, error) {
	if s.failure != nil {
		return recovery.RestorePlan{}, nil, s.failure
	}
	if b.RunID != s.plan.RunID || b.BackupRef != s.plan.BackupRef {
		return recovery.RestorePlan{}, nil, recovery.ErrAdmissionBinding
	}
	f, err := os.Open(s.plan.ArtifactPath)
	s.handle = f
	return s.plan, f, err
}
func (s *s414Source) CheckApproved(ctx context.Context, p recovery.RestorePlan, f *os.File) error {
	s.checks++
	if p != s.plan || f != s.handle || s.failure != nil {
		return recovery.ErrAdmissionBinding
	}
	return ctx.Err()
}

type s414Target struct {
	receipt recovery.RestoreReceipt
	calls   int
	read    func(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error)
}

func (s *s414Target) ReadRestored(ctx context.Context, p recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	s.calls++
	if s.read != nil {
		return s.read(ctx, p)
	}
	return s.receipt, nil
}

type s414Fixture struct {
	source       *s414Source
	target       *s414Target
	tool         *restoreProcessTool
	driver       *controlledRestoreDriver
	world        *s414World
	observations []restoreProcessObservation
	uses         []*os.File
}

func newS414Fixture(t *testing.T, backup string) *s414Fixture {
	t.Helper()
	dir := s414Directory(t)
	artifact := filepath.Join(dir, "artifact")
	content := []byte("approved synthetic artifact bytes")
	if os.WriteFile(artifact, content, 0600) != nil {
		t.Fatal("create fixture artifact")
	}
	p := recovery.RestorePlan{Version: restoreVersion, RunID: "restore-run", BackupRef: backup, DeploymentID: "deployment", InstallationID: "installation", TargetDigest: recovery.SHA256Digest([]byte("target")), ArtifactDigest: recovery.SHA256Digest(content), ArtifactBytes: int64(len(content)), SourceApprovalRef: "independent-fixture-approval", SourceApprovalDigest: recovery.SHA256Digest([]byte("approval")), SourceInstallationID: "source-installation", SourceSchemaDigest: recovery.SHA256Digest([]byte("schema")), ArtifactPath: artifact}
	f := &s414Fixture{source: &s414Source{plan: p}}
	f.target = &s414Target{}
	f.target.receipt, _ = decodeRestoreReceipt(s414ReceiptJSON(p), p)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal("read test image")
	}
	f.tool = &restoreProcessTool{executable: exe, digest: recovery.SHA256Digest(image), environment: []string{"SystemRoot=" + os.Getenv("SystemRoot")}, approveImage: func(context.Context, string, *os.File) error { return nil }}
	f.tool.acquireImage = s414AcquireImage(dir, t)
	f.tool.useSource = func(ctx context.Context, verb string, handle *os.File) error {
		f.uses = append(f.uses, handle)
		raw, err := io.ReadAll(handle)
		if err != nil || !bytes.Equal(raw, content) {
			return recovery.ErrAdmissionBinding
		}
		return ctx.Err()
	}
	f.tool.observed = func(o restoreProcessObservation) {
		f.observations = append(f.observations, o)
		t.Logf("child_pid=%d waited=%t pipes_joined=%t wait_returned_utc=%s", o.PID, o.Waited, o.PipesJoined, time.Now().UTC().Format(time.RFC3339Nano))
	}
	f.driver, err = newControlledRestoreDriver(f.source, f.target, f.tool, s414Canary)
	if err != nil {
		t.Fatal(err)
	}
	f.world = newS414World(p)
	f.driver.ownership = f.world.lease
	t.Cleanup(func() {
		if err := f.driver.Close(); err != nil {
			t.Error(err)
		}
	})
	return f
}

func (f *s414Fixture) runtime(t *testing.T) recovery.Runtime {
	t.Helper()
	if f.world == nil {
		f.world = newS414World(f.source.plan)
	}
	w := f.world
	store := recovery.AdmissionStore(w)
	if w.actualStore != nil {
		store = w.actualStore
	}
	r, err := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: store, Journal: w, ArticleSnapshots: w, Closures: w, Genesis: w, Exclusive: w, Lifecycle: w, Runs: w, Identity: w, Restore: f.driver, Consumer: w, Exit: w, RecoverySecret: w})
	if err != nil {
		t.Fatal(err)
	}
	f.driver.authority = r
	f.driver.ownership = w.lease
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return r
}

func s414Context(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	ctx, err := platform.ContextWithTraceID(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}
func s414Enter(t *testing.T, f *s414Fixture, r recovery.Runtime) error {
	t.Helper()
	return r.Enter(s414Context(t), recovery.RecoveryInput{RunBinding: recovery.RunBinding{RunID: f.source.plan.RunID, BackupRef: f.source.plan.BackupRef}, Secret: []byte("synthetic-independent-secret")})
}

func TestRecoveryRestoreDriver(t *testing.T) {
	t.Run("ApprovedImageLifetimeAndOwnership", testS414LaunchControls)
	t.Run("ImageOwnerLifecycle", testS414ImageLifecycle)
	t.Run("ImageAcquisitionUnwind", testS414ImageUnwind)
	t.Run("CompoundPanicCause", testS414CompoundCauses)
	t.Run("StableArtifactHandleAndApprovedTarget", func(t *testing.T) { testS414Stable(t); t.Run("SameBytesWrongHandlePath", testS414WrongHandlePath) })
	t.Run("PrivateStdinStrictReceiptAndCanary", func(t *testing.T) { testS414Codec(t); t.Run("AllBindings", testS414AllReceiptBindings) })
	t.Run("IndependentTargetAndNativeReceiptReadback", func(t *testing.T) { testS414Independent(t); t.Run("ProviderErrorRedacted", testS414PrivateError) })
	t.Run("BeginAckAndRestoreUnknownNoReplay", testS414Begin)
	t.Run("ExclusiveAndRealChildJoin", testS414ChildJoin)
}

func testS414Stable(t *testing.T) {
	t.Run("PublicEnterSameHandle", s414StableCase1)
	for _, name := range []string{"MissingApproval", "BadToolDigest", "MissingNativeHandoff", "BadArtifact", "PathSwap", "WrongBinding"} {
		t.Run(name, func(t *testing.T) {
			f := newS414Fixture(t, "backup")
			switch name {
			case "MissingApproval":
				f.source.failure = recovery.ErrAdmissionUnsupported
			case "BadToolDigest":
				f.tool.digest = recovery.SHA256Digest([]byte("wrong"))
			case "MissingNativeHandoff":
				f.tool.useSource = nil
			case "BadArtifact":
				f.source.plan.ArtifactDigest = recovery.SHA256Digest([]byte("wrong"))
			case "WrongBinding":
				f.source.plan.SourceSchemaDigest = "bad"
			case "PathSwap":
				if _, err := f.driver.Prepare(context.Background(), recovery.RunBinding{RunID: f.source.plan.RunID, BackupRef: f.source.plan.BackupRef}); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(f.source.plan.ArtifactPath, f.source.plan.ArtifactPath+".retained"); err != nil {
					t.Log("native platform denied rename of held artifact; separately test handle/path mismatch")
					f.source.failure = recovery.ErrAdmissionDrift
				}
				if err := os.WriteFile(f.source.plan.ArtifactPath, []byte("replacement"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			r := f.runtime(t)
			if err := s414Enter(t, f, r); err == nil || errors.Is(err, s414Boundary) {
				t.Fatal("invalid source passed")
			}
			if f.world.begins != 0 || len(f.observations) != 0 {
				t.Fatal("invalid preflight reached Begin/tool")
			}
		})
	}
	t.Run("ZeroPermitCannotLaunch", s414StableCase2)
}

func testS414Codec(t *testing.T) {
	f := newS414Fixture(t, "backup")
	p := f.source.plan
	valid := s414ReceiptJSON(p)
	t.Run("ExactLimit", s414CodecCase1)
	testS414ReceiptVariants(t, p, valid)
	t.Run("PrivateWire", s414CodecCase2)
	for _, backup := range []string{"backup", "child-invalid", "child-oversize", "child-exit"} {
		t.Run("ActualChild_"+backup, func(t *testing.T) {
			f := newS414Fixture(t, backup)
			r := f.runtime(t)
			err := s414Enter(t, f, r)
			if backup == "backup" {
				if !errors.Is(err, s414Boundary) {
					t.Fatal(err)
				}
			} else if err == nil || f.world.binds != 0 {
				t.Fatal("bad child accepted")
			}
			if strings.Contains(err.Error(), s414Canary) {
				t.Fatal("canary leaked")
			}
			for _, o := range f.observations {
				if !o.Waited || !o.PipesJoined {
					t.Fatal("unjoined child")
				}
				t.Logf("child_pid=%d waited=true pipes_joined=true", o.PID)
			}
		})
	}
}

func testS414Independent(t *testing.T) {
	for _, name := range []string{"TargetMismatch", "Unavailable", "NativeReceiptAbsent"} {
		t.Run(name, func(t *testing.T) {
			f := newS414Fixture(t, "backup")
			switch name {
			case "TargetMismatch":
				f.target.receipt.TargetDigest = recovery.SHA256Digest([]byte("other"))
			case "Unavailable":
				f.target.read = func(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
					return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
				}
			case "NativeReceiptAbsent":
				f.target.receipt = recovery.RestoreReceipt{}
			}
			r := f.runtime(t)
			if err := s414Enter(t, f, r); !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
				t.Fatal("exit0 became proof", err)
			}
			if f.world.binds != 0 {
				t.Fatal("unproved bind")
			}
		})
	}
}

func testS414Begin(t *testing.T) {
	for _, mode := range []string{"ack", "pending", "wrong-backup", "uncommitted"} {
		t.Run(mode, func(t *testing.T) {
			f := newS414Fixture(t, "backup")
			f.world = newS414World(f.source.plan)
			f.world.beginMode = mode
			r := f.runtime(t)
			if err := s414Enter(t, f, r); err == nil {
				t.Fatal("Begin uncertainty passed")
			}
			if len(f.observations) != 0 || f.world.binds != 0 {
				t.Fatal("unknown Begin replaced")
			}
		})
	}
	t.Run("CrashBeforeStoreResumeReadOnly", func(t *testing.T) {
		f := newS414Fixture(t, "backup")
		r := f.runtime(t)
		if err := s414Enter(t, f, r); !errors.Is(err, s414Boundary) {
			t.Fatal(err)
		}
		if err := r.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		f.driver.Close()
		next := newS414Fixture(t, "backup")
		next.source.plan = f.source.plan
		next.target.receipt = f.target.receipt
		next.world = f.world
		next.world.binds = 0
		next.world.lease = &s414Lease{}
		rr := next.runtime(t)
		if err := rr.Resume(s414Context(t), recovery.RunBinding{RunID: f.source.plan.RunID, BackupRef: f.source.plan.BackupRef}); !errors.Is(err, s414Boundary) {
			t.Fatal(err)
		}
		if next.world.begins != 1 || len(next.observations) != 1 {
			t.Fatal("Resume Begin/Replace repeated")
		}
	})
}

func testS414ChildJoin(t *testing.T) {
	f := newS414Fixture(t, "child-hold")
	_, seen, live, release := s414HoldChild(f)
	defer release()
	r := f.runtime(t)
	action := make(chan error, 1)
	go func() { action <- s414Enter(t, f, r) }()
	select {
	case <-seen:
	case e := <-action:
		release()
		t.Fatal("child did not enter", e)
	case <-time.After(20 * time.Second):
		release()
		t.Fatal("child start timeout")
	}
	stop := make(chan error, 1)
	go func() { stop <- r.Stop(context.Background()) }()
	operationContext := <-live
	<-operationContext.Done()
	select {
	case <-action:
		t.Fatal("returned before writer joined")
	default:
	}
	select {
	case <-stop:
		t.Fatal("Stop returned before child/pipe join")
	default:
	}
	if f.world.lease.closes.Load() != 0 {
		t.Fatal("deployment released while pipe owned")
	}
	release()
	if err := <-action; err == nil {
		t.Fatal("cancellation lost")
	}
	if err := <-stop; err != nil {
		t.Fatal(err)
	}
	if f.world.lease.closes.Load() != 1 || len(f.observations) != 1 || !f.observations[0].Waited || !f.observations[0].PipesJoined {
		t.Fatal("join ledger")
	}
	t.Logf("child_pid=%d waited=true pipes_joined=true close=1", f.observations[0].PID)
}

// s414World supplies only external software authorities. Runtime is always real.
type s414World struct {
	mu            sync.Mutex
	plan          recovery.RestorePlan
	installation  recovery.InstallationBinding
	entries       []recovery.JournalEntry
	beginMode     string
	begins, binds int
	lease         *s414Lease
	actualStore   recovery.AdmissionStore
}
type s414Lease struct {
	lost   atomic.Bool
	closes atomic.Int32
}

func (l *s414Lease) Check(ctx context.Context) error {
	if l.lost.Load() || l.closes.Load() != 0 {
		return recovery.ErrAdmissionClosed
	}
	return ctx.Err()
}
func (l *s414Lease) Close() error { l.closes.Add(1); return nil }
func newS414World(p recovery.RestorePlan) *s414World {
	i := recovery.InstallationBinding{Version: recovery.InstallationVersion, DeploymentID: p.DeploymentID, InstallationID: p.InstallationID, TargetDigest: p.TargetDigest, CommitmentRef: "commitment", CommitmentDigest: recovery.SHA256Digest([]byte("commitment")), InstalledAt: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC), RecoveryInitial: recovery.AnchorSnapshot{JournalID: "recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1}, ArticleInitial: recovery.AnchorSnapshot{JournalID: "article-journal", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 2}}
	return &s414World{plan: p, installation: i, lease: &s414Lease{}}
}
func (w *s414World) ReadAdmissionSnapshot(ctx context.Context) (recovery.AdmissionSnapshot, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := recovery.AdmissionSnapshot{Installation: w.installation, Revision: recovery.StoreRevision{InstallationID: w.installation.InstallationID, Counter: 1}, Runs: []recovery.AdmissionRun{}}
	raw, e := recovery.CanonicalAdmissionSnapshot(s)
	s.Revision.Digest = recovery.SHA256Digest(raw)
	return s, e
}
func (w *s414World) ReadAdmissionRevision(ctx context.Context) (recovery.StoreRevision, error) {
	s, e := w.ReadAdmissionSnapshot(ctx)
	return s.Revision, e
}
func (w *s414World) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	a := w.installation.ArticleInitial
	if ns == recovery.RecoveryNamespaceV1 {
		a = w.installation.RecoveryInitial
	}
	s := recovery.VerifiedJournalSnapshot{JournalID: a.JournalID, Namespace: ns, Epoch: a.Epoch, Revision: ns + "-r0", Entries: []recovery.JournalEntry{}, Prefixes: []recovery.JournalHead{{}}}
	if ns == recovery.RecoveryNamespaceV1 && len(w.entries) > 0 {
		s.Entries = append([]recovery.JournalEntry{}, w.entries...)
		s.Head = recovery.JournalHead{Sequence: 1, Hash: recovery.SHA256Digest([]byte("begin-head"))}
		s.Prefixes = append(s.Prefixes, s.Head)
		s.Revision = "recovery-r1"
		if w.beginMode == "pending" {
			s.Pending = &recovery.AnchorPending{NewHead: s.Head}
		}
	}
	return s, nil
}
func (w *s414World) Begin(ctx context.Context, run, backup string, secret []byte) (recovery.JournalEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.begins++
	e := recovery.JournalEntry{Namespace: recovery.RecoveryNamespaceV1, Sequence: 1, RunID: run, BackupRef: backup, KeyID: "retained-key", KeyVersion: "v1", SecretFingerprint: "hmac-sha256:" + strings.Repeat("a", 64), RecordHMAC: "hmac-sha256:" + strings.Repeat("b", 64), Stage: recovery.StageBegun, RecordedAt: w.installation.InstalledAt}
	if w.beginMode != "uncommitted" {
		w.entries = []recovery.JournalEntry{e}
		if w.beginMode == "wrong-backup" {
			w.entries[0].BackupRef = "wrong"
		}
	}
	if w.beginMode == "ack" {
		return e, recovery.ErrRecoveryWriteUnknown
	}
	return e, nil
}
func (w *s414World) VerifyGenesis(context.Context, recovery.InstallationBinding) (recovery.GenesisProof, error) {
	return recovery.GenesisProof{Binding: w.installation, RecoveryInitial: w.installation.RecoveryInitial, ArticleInitial: w.installation.ArticleInitial, CommitmentRevision: "commitment-r1", CommitmentDigest: w.installation.CommitmentDigest}, nil
}
func (w *s414World) Acquire(context.Context) (recovery.ExclusiveLease, error) { return w.lease, nil }
func (w *s414World) StopAndJoin(context.Context) error                        { return nil }
func (w *s414World) ReconcileAndStart(context.Context, recovery.PhasePermit) error {
	return recovery.ErrAdmissionClosed
}
func (w *s414World) ReadPublishedClosure(context.Context, recovery.ClosureRequest) (recovery.PublishedClosure, error) {
	return recovery.PublishedClosure{}, recovery.ErrAdmissionClosed
}
func (w *s414World) BindRestoredRun(context.Context, recovery.PhasePermit, recovery.RunBinding, recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	w.binds++
	return recovery.RecoveryRun{}, s414Boundary
}
func (w *s414World) FinalizeRestoredIdentity(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (w *s414World) PrepareRecoveryTrust(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte) error {
	return recovery.ErrAdmissionClosed
}
func (w *s414World) CompleteIdentityReview(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (w *s414World) RevokeRecoverySessions(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (w *s414World) Consume(context.Context, string) (recovery.ConsumptionResult, error) {
	return recovery.ConsumptionResult{}, recovery.ErrAdmissionClosed
}
func (w *s414World) ExitRecovery(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
	return recovery.ErrAdmissionClosed
}
func (w *s414World) ReadInitialRecoverySecret(context.Context, recovery.RunBinding) ([]byte, error) {
	return nil, recovery.ErrAdmissionClosed
}
func (w *s414World) VerifyBeginSecret(context.Context, recovery.RunBinding, []byte) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (w *s414World) MarkConsumed(context.Context, string) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (w *s414World) Read(context.Context, string) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (w *s414World) VerifyAgainst(context.Context, recovery.RecoveryRun) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, recovery.ErrAdmissionClosed
}
func (w *s414World) Load(context.Context, string) (recovery.RecoveryRun, error) {
	return recovery.RecoveryRun{}, recovery.ErrRunNotFound
}
func (w *s414World) Transition(context.Context, string, recovery.RecoveryOutcome, recovery.RecoveryOutcome, recovery.RecoveryRunChange) (recovery.RecoveryRun, error) {
	return recovery.RecoveryRun{}, recovery.ErrAdmissionClosed
}

func s414StableCase1(t *testing.T) {
	f := newS414Fixture(t, "backup")
	r := f.runtime(t)
	if err := s414Enter(t, f, r); !errors.Is(err, s414Boundary) {
		t.Fatalf("public producer did not reach bind: %v", err)
	}
	if f.world.begins != 1 || f.world.binds != 1 || len(f.uses) != 2 || f.uses[0] != f.uses[1] {
		t.Fatal("wrong Begin/handle/bind ledger")
	}
	for _, o := range f.observations {
		if o.PID == 0 || !o.Waited || !o.PipesJoined {
			t.Fatal("child not joined")
		}
		t.Logf("child_pid=%d waited=true pipes_joined=true", o.PID)
	}
}

func s414StableCase2(t *testing.T) {
	f := newS414Fixture(t, "backup")
	f.runtime(t)
	p, _ := f.driver.Prepare(context.Background(), recovery.RunBinding{RunID: f.source.plan.RunID, BackupRef: f.source.plan.BackupRef})
	if _, err := f.driver.Replace(context.Background(), recovery.PhasePermit{}, p); err == nil {
		t.Fatal("zero permit")
	}
	if len(f.observations) != 0 {
		t.Fatal("tool launched")
	}
}

func testS414ReceiptVariants(t *testing.T, p recovery.RestorePlan, valid []byte) {
	cases := map[string][]byte{"Null": []byte("null"), "Empty": []byte("{}"), "InvalidUTF8": append([]byte{0xff}, valid...), "BOM": append([]byte{239, 187, 191}, valid...), "Trailing": append(append([]byte{}, valid...), []byte("{}")...), "Duplicate": bytes.Replace(valid, []byte("{\"version\":"), []byte("{\"version\":\"han-db-restore/v1\",\"version\":"), 1), "EscapedDuplicate": bytes.Replace(valid, []byte("{\"version\":"), []byte("{\"ver\\u0073ion\":\"han-db-restore/v1\",\"version\":"), 1)}
	for _, key := range receiptKeys() {
		var values map[string]any
		json.Unmarshal(valid, &values)
		delete(values, key)
		cases["Missing_"+key], _ = json.Marshal(values)
		json.Unmarshal(valid, &values)
		values[key] = nil
		cases["Null_"+key], _ = json.Marshal(values)
	}
	for _, name := range []string{"Extra", "FloatBytes", "WrongBinding", "TimePrecision", "WrongDigest"} {
		raw := append([]byte{}, valid...)
		switch name {
		case "Extra":
			raw = bytes.Replace(raw, []byte("{\"version"), []byte("{\"extra\":1,\"version"), 1)
		case "FloatBytes":
			raw = bytes.Replace(raw, []byte("\"artifact_bytes\":33"), []byte("\"artifact_bytes\":33.5"), 1)
		case "WrongBinding":
			raw = bytes.Replace(raw, []byte("restore-run"), []byte("another-run"), 1)
		case "TimePrecision":
			raw = bytes.Replace(raw, []byte(".123456Z"), []byte(".123Z"), 1)
		case "WrongDigest":
			raw = bytes.Replace(raw, []byte("sha256:"), []byte("sha255:"), 1)
		}
		cases[name] = raw
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeRestoreReceipt(raw, p); err == nil {
				t.Fatal("malformed receipt accepted")
			}
		})
	}

}

func s414CodecCase1(t *testing.T) {
	f := newS414Fixture(t, "backup")
	p := f.source.plan
	valid := s414ReceiptJSON(p)
	raw := append(append([]byte{}, valid...), bytes.Repeat([]byte(" "), restoreReceiptLimit-len(valid))...)
	if _, e := decodeRestoreReceipt(raw, p); e != nil {
		t.Fatal(e)
	}
	if _, e := decodeRestoreReceipt(append(raw, ' '), p); e == nil {
		t.Fatal("65537 accepted")
	}
}

func s414CodecCase2(t *testing.T) {
	f := newS414Fixture(t, "backup")
	p := f.source.plan
	raw, _ := privateRestoreInput(p, s414Canary)
	if strictRestoreObject(raw, []string{"plan", "target_connection"}) != nil {
		t.Fatal("envelope")
	}
	var top map[string]json.RawMessage
	json.Unmarshal(raw, &top)
	if strictRestoreObject(top["plan"], []string{"version", "run_id", "backup_ref", "deployment_id", "installation_id", "target_digest", "artifact_digest", "artifact_bytes", "source_approval_ref", "source_approval_digest", "source_installation_id", "source_schema_digest", "artifact_path"}) != nil {
		t.Fatal("plan keys")
	}
	if strictRestoreObject(top["target_connection"], []string{"dsn"}) != nil {
		t.Fatal("target keys")
	}
}

func s414HoldChild(f *s414Fixture) (chan struct{}, chan struct{}, chan context.Context, func()) {
	gate := make(chan struct{})
	seen := make(chan struct{})
	var once sync.Once
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(gate) }) }
	live := make(chan context.Context, 1)
	originalUse := f.tool.useSource
	f.tool.useSource = func(ctx context.Context, verb string, file *os.File) error {
		live <- ctx
		return originalUse(ctx, verb, file)
	}
	f.tool.inputClosed = func() { <-gate }
	f.tool.outputSeen = func() { once.Do(func() { close(seen) }) }

	return gate, seen, live, release
}

func testS414AllReceiptBindings(t *testing.T) {
	f := newS414Fixture(t, "backup")
	p := f.source.plan
	cases := map[string]func(*recovery.RestorePlan){"version": func(p *recovery.RestorePlan) { p.Version = "unknown" }, "run": func(p *recovery.RestorePlan) { p.RunID = "other" }, "backup": func(p *recovery.RestorePlan) { p.BackupRef = "other" }, "deployment": func(p *recovery.RestorePlan) { p.DeploymentID = "other" }, "installation": func(p *recovery.RestorePlan) { p.InstallationID = "other" }, "target": func(p *recovery.RestorePlan) { p.TargetDigest = recovery.SHA256Digest([]byte("other")) }, "artifact": func(p *recovery.RestorePlan) { p.ArtifactDigest = recovery.SHA256Digest([]byte("other")) }, "bytes": func(p *recovery.RestorePlan) { p.ArtifactBytes++ }, "approval": func(p *recovery.RestorePlan) { p.SourceApprovalDigest = recovery.SHA256Digest([]byte("other")) }, "schema": func(p *recovery.RestorePlan) { p.SourceSchemaDigest = recovery.SHA256Digest([]byte("other")) }}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			other := p
			change(&other)
			if _, err := decodeRestoreReceipt(s414ReceiptJSON(other), p); err == nil {
				t.Fatal("self-consistent wrong binding accepted")
			}
		})
	}
}

func testS414WrongHandlePath(t *testing.T) {
	f := newS414Fixture(t, "backup")
	p := f.source.plan
	handle, err := os.Open(p.ArtifactPath)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	raw, err := io.ReadAll(handle)
	if err != nil {
		t.Fatal(err)
	}
	p.ArtifactPath = filepath.Join(filepath.Dir(p.ArtifactPath), "same-bytes-different-file")
	if err = os.WriteFile(p.ArtifactPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err = checkRestoreArtifact(context.Background(), p, handle); !errors.Is(err, recovery.ErrAdmissionDrift) {
		t.Fatal("reopened path substituted for retained handle", err)
	}
}

func testS414PrivateError(t *testing.T) {
	f := newS414Fixture(t, "backup")
	cause := errors.New(s414Canary)
	f.target.read = func(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
		return recovery.RestoreReceipt{}, cause
	}
	r := f.runtime(t)
	err := s414Enter(t, f, r)
	if !errors.Is(err, cause) || !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || strings.Contains(err.Error(), s414Canary) {
		t.Fatal("private cause presentation or identity invalid")
	}
}

// Owned CURRENT testbinary copy only; unchanged code and fixed verbs.
func testS414LaunchControls(t *testing.T) {
	for _, mode := range []string{"healthy", "image", "lease"} {
		t.Run(mode, func(t *testing.T) { testS414LaunchCase(t, mode) })
	}
}

type s414LaunchProbe struct {
	approvals, approvalAt  int
	mutated, digestChanged bool
}

func testS414LaunchCase(t *testing.T, mode string) {
	f := newS414Fixture(t, "backup")
	probe := s414ConfigureLaunchProbe(t, f, mode)
	r := f.runtime(t)
	err := s414Enter(t, f, r)
	stopErr := r.Stop(context.Background())
	closeErr := f.driver.Close()
	t.Logf("mode=%s approvals=%d approval_at_mutation=%d mutated=%t digest_changed=%t begin=%d bind=%d children=%d target_reads=%d stop_ok=%t close_ok=%t deployment_close=%d", mode, probe.approvals, probe.approvalAt, probe.mutated, probe.digestChanged, f.world.begins, f.world.binds, len(f.observations), f.target.calls, stopErr == nil, closeErr == nil, f.world.lease.closes.Load())
	for _, child := range f.observations {
		t.Logf("child_pid=%d waited=%t pipes_joined=%t", child.PID, child.Waited, child.PipesJoined)
		if child.PID <= 0 || !child.Waited || !child.PipesJoined {
			t.Error("unjoined real child/pipe")
		}
	}
	if stopErr != nil || closeErr != nil || f.world.lease.closes.Load() != 1 {
		t.Error("cleanup/join incomplete")
	}
	if mode == "healthy" {
		if !errors.Is(err, s414Boundary) || f.world.begins != 1 || f.world.binds != 1 || len(f.observations) != 2 || probe.approvals < 3 {
			t.Error("healthy public path not reached")
		}
	} else {
		if !probe.mutated || probe.approvalAt < 2 || f.world.begins != 1 {
			t.Error("actual postpreflight seam not reached")
		}
		if len(f.observations) != 0 {
			t.Errorf("NEW CHILD AFTER %s INVALIDATION: children=%d; all waited; rejection came later", mode, len(f.observations))
		}
		if f.world.binds != 0 {
			t.Error("invalid launch reached bind")
		}
		if mode == "lease" && !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
			t.Error("unknown cause lost")
		}
	}
}

func s414ConfigureLaunchProbe(t *testing.T, f *s414Fixture, mode string) *s414LaunchProbe {
	t.Helper()
	current, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(current)
	if err != nil {
		t.Fatal("read current own testbinary")
	}
	name := filepath.Join(s414Directory(t), "owned-restore-test.exe")
	if os.WriteFile(name, image, 0700) != nil {
		t.Fatal("write owned copy")
	}
	digest := recovery.SHA256Digest(image)
	f.tool.executable, f.tool.digest = name, digest
	probe := &s414LaunchProbe{}
	f.tool.approveImage = func(ctx context.Context, path string, file *os.File) error {
		probe.approvals++
		raw, err := io.ReadAll(file)
		defer clear(raw)
		if err != nil || path != name || recovery.SHA256Digest(raw) != digest {
			return recovery.ErrAdmissionBinding
		}
		_, err = file.Seek(0, io.SeekStart)
		return errors.Join(err, ctx.Err())
	}
	base := f.tool.useSource
	f.tool.useSource = func(ctx context.Context, verb string, file *os.File) error {
		if err := base(ctx, verb, file); err != nil {
			return err
		}
		return probe.mutate(ctx, mode, name, image, f)
	}
	return probe
}

func (p *s414LaunchProbe) mutate(ctx context.Context, mode, name string, image []byte, f *s414Fixture) error {
	if p.mutated || mode == "healthy" {
		return ctx.Err()
	}
	p.mutated = true
	p.approvalAt = p.approvals
	if mode == "image" {
		replacement := append(append([]byte{}, image...), []byte("S414-REVIEWER-NONCODE-TRAILER")...)
		if err := os.WriteFile(name, replacement, 0700); err != nil {
			return err
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		p.digestChanged = recovery.SHA256Digest(raw) != recovery.SHA256Digest(image)
		if !p.digestChanged {
			return errors.New("probe did not change digest")
		}
	} else {
		f.world.lease.lost.Store(true)
	}
	return ctx.Err()
}

// Explicit software image facility: captures approved bytes before handoff and
// starts a retained OWN COPY of that image, not a later reopening of source path.
// This test facility proves its software lifetime protocol, not native ACL/locking.
type s414ImageLease struct {
	mu                         sync.Mutex
	path, dir, digest          string
	original                   *os.File // borrowed checked handle, retained by restoreCheckedImage
	bytes                      []byte   // sole owner until terminal CloseAndJoin; never exposed to handoff
	executable                 *os.File
	closed                     bool
	closeEntered, closeRelease chan struct{}
	closeFailure               error
	started                    func(int)
}

func s414AcquireImage(dir string, t *testing.T) func(context.Context, string, string, *os.File) (restoreImageLease, error) {
	return func(ctx context.Context, name, digest string, checked *os.File) (restoreImageLease, error) {
		raw, err := io.ReadAll(checked)
		if err != nil || recovery.SHA256Digest(raw) != digest {
			clear(raw)
			return nil, recovery.ErrAdmissionBinding
		}
		lease := &s414ImageLease{path: name, dir: dir, digest: digest, original: checked, bytes: raw, started: func(pid int) {
			t.Logf("child_start_pid=%d start_returned_utc=%s", pid, time.Now().UTC().Format(time.RFC3339Nano))
		}}
		return lease, ctx.Err()
	}
}

func (i *s414ImageLease) checkOriginal(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	opened, err := i.original.Stat()
	if err != nil {
		return recovery.ErrAdmissionBinding
	}
	current, err := os.Lstat(i.path)
	if err != nil || !os.SameFile(opened, current) || !opened.Mode().IsRegular() {
		return recovery.ErrAdmissionDrift
	}
	if _, err = i.original.Seek(0, io.SeekStart); err != nil {
		return err
	}
	raw, err := io.ReadAll(i.original)
	defer clear(raw)
	if err != nil || recovery.SHA256Digest(raw) != i.digest || !bytes.Equal(raw, i.bytes) {
		return recovery.ErrAdmissionBinding
	}
	return ctx.Err()
}

func (i *s414ImageLease) ownExecutable() (string, error) {
	writer, err := os.CreateTemp(i.dir, "owned-approved-image-*.exe")
	if err != nil {
		return "", err
	}
	_, writeErr := writer.Write(i.bytes)
	closeErr := writer.Close()
	if writeErr != nil || closeErr != nil {
		return "", errors.Join(writeErr, closeErr)
	}
	if err = os.Chmod(writer.Name(), 0700); err != nil {
		return "", err
	}
	i.executable, err = os.Open(writer.Name())
	if err != nil {
		return "", err
	}
	raw, err := io.ReadAll(i.executable)
	defer clear(raw)
	if err != nil || recovery.SHA256Digest(raw) != i.digest {
		return "", recovery.ErrAdmissionBinding
	}
	return writer.Name(), nil
}

func (i *s414ImageLease) Start(ctx context.Context, verb string, env []string, pipes *restoreChildPipes, check restoreStartCheck) (*exec.Cmd, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed || check == nil || (verb != "restore-v1" && verb != "verify-v1") {
		return nil, recovery.ErrAdmissionClosed
	}
	if err := i.checkOriginal(ctx); err != nil {
		return nil, err
	}
	name, err := i.ownExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(name, verb)
	cmd.Env = append([]string{}, env...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pipes.inR, pipes.outW, pipes.errW
	// Approval is bound to the owner-captured image used by this exact Start.
	// The live real Runtime/lease check follows all handoff/image preparation I/O.
	if err = check(ctx); err != nil {
		return nil, err
	}
	err = cmd.Start()
	if cmd.Process != nil && i.started != nil {
		i.started(cmd.Process.Pid)
	}
	return cmd, err
}

func (i *s414ImageLease) CloseAndJoin() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return i.closeFailure
	}
	if i.closeEntered != nil {
		close(i.closeEntered)
		<-i.closeRelease
	}
	var err error
	if i.executable != nil {
		err = i.executable.Close()
	}
	clear(i.bytes)
	i.closed = true
	return errors.Join(err, i.closeFailure)
}

func testS414ImageLifecycle(t *testing.T) {
	t.Run("MissingFacility", func(t *testing.T) {
		f := newS414Fixture(t, "backup")
		f.tool.acquireImage = nil
		r := f.runtime(t)
		err := s414Enter(t, f, r)
		if !errors.Is(err, recovery.ErrAdmissionUnsupported) || f.world.begins != 0 || len(f.observations) != 0 {
			t.Fatal("missing launch facility did not fail before Begin")
		}
	})
	t.Run("NilLease", func(t *testing.T) {
		f := newS414Fixture(t, "backup")
		f.tool.acquireImage = func(context.Context, string, string, *os.File) (restoreImageLease, error) { return nil, nil }
		r := f.runtime(t)
		if err := s414Enter(t, f, r); err == nil || f.world.begins != 0 {
			t.Fatal("nil image lease accepted")
		}
	})
	t.Run("ReturnedOwnerErrorJoin", testS414ImageAcquireErrorJoin)
	t.Run("ImageCloseJoin", testS414ImageCloseJoin)
	t.Run("LateLeaseBeforeConcreteStart", testS414LateLeaseStart)
	t.Run("StartedCommandErrorStillWaits", testS414PartialStart)
}

type s414StartWrapper struct {
	restoreImageLease
	before  func()
	failure error
}

func (i *s414StartWrapper) Start(ctx context.Context, verb string, env []string, pipes *restoreChildPipes, check restoreStartCheck) (*exec.Cmd, error) {
	if i.before != nil {
		i.before()
	}
	cmd, err := i.restoreImageLease.Start(ctx, verb, env, pipes, check)
	if err == nil && cmd != nil && cmd.Process != nil {
		err = i.failure
	}
	return cmd, err
}

func testS414LateLeaseStart(t *testing.T) {
	f := newS414Fixture(t, "backup")
	base := f.tool.acquireImage
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		l, err := base(ctx, name, digest, file)
		if err != nil {
			return l, err
		}
		return &s414StartWrapper{restoreImageLease: l, before: func() { f.world.lease.lost.Store(true) }}, nil
	}
	r := f.runtime(t)
	err := s414Enter(t, f, r)
	if !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || f.world.begins != 1 || f.world.binds != 0 || len(f.observations) != 0 || f.target.calls != 0 {
		t.Fatal("concrete Start skipped real lease check")
	}
}

func testS414PartialStart(t *testing.T) {
	f := newS414Fixture(t, "backup")
	base := f.tool.acquireImage
	cause := errors.New("synthetic started-command failure")
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		l, err := base(ctx, name, digest, file)
		if err != nil {
			return l, err
		}
		return &s414StartWrapper{restoreImageLease: l, failure: cause}, nil
	}
	r := f.runtime(t)
	err := s414Enter(t, f, r)
	if !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || !errors.Is(err, cause) || len(f.observations) != 1 || !f.observations[0].Waited || !f.observations[0].PipesJoined || f.target.calls != 0 {
		t.Fatal("partially started child not joined/classified")
	}
}

func testS414ImageAcquireErrorJoin(t *testing.T) {
	f := newS414Fixture(t, "backup")
	base := f.tool.acquireImage
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var image *s414ImageLease
	live := make(chan context.Context, 1)
	cause := errors.New("synthetic acquired-image failure")
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		l, err := base(ctx, name, digest, file)
		if err != nil {
			return l, err
		}
		image = l.(*s414ImageLease)
		image.closeEntered = entered
		image.closeRelease = release
		live <- ctx
		return image, cause
	}
	r := f.runtime(t)
	action := make(chan error, 1)
	go func() { action <- s414Enter(t, f, r) }()
	<-entered
	if _, err := image.original.Stat(); err != nil {
		t.Fatal("image handle closed before terminal owner")
	}
	stop := s414StopWhileImageOwned(t, r, live, action, f.world.lease)
	once.Do(func() { close(release) })
	if err := <-action; !errors.Is(err, cause) {
		t.Fatal("acquire cause lost")
	}
	if err := <-stop; err != nil {
		t.Fatal(err)
	}
	if _, err := image.original.Stat(); err == nil {
		t.Fatal("checked image descriptor leaked")
	}
	if f.world.begins != 0 || !image.closed || len(f.observations) != 0 || f.world.lease.closes.Load() != 1 {
		t.Fatal("acquire terminal ledger")
	}
}

func testS414ImageCloseJoin(t *testing.T) {
	f := newS414Fixture(t, "backup")
	base := f.tool.acquireImage
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var image *s414ImageLease
	var calls int
	terminalFailure := errors.New("synthetic terminal image cleanup failure")
	live := make(chan context.Context, 1)
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		l, err := base(ctx, name, digest, file)
		calls++
		if err == nil && calls == 2 {
			image = l.(*s414ImageLease)
			image.closeEntered = entered
			image.closeRelease = release
			image.closeFailure = terminalFailure
			live <- ctx
		}
		return l, err
	}
	r := f.runtime(t)
	action := make(chan error, 1)
	go func() { action <- s414Enter(t, f, r) }()
	<-entered
	if len(f.observations) != 1 || !f.observations[0].Waited || !f.observations[0].PipesJoined {
		t.Fatal("image owner closed before child/pipe join")
	}
	if _, err := image.executable.Stat(); err != nil {
		t.Fatal("image facility no longer owns resource")
	}
	stop := s414StopWhileImageOwned(t, r, live, action, f.world.lease)
	once.Do(func() { close(release) })
	if err := <-action; !errors.Is(err, recovery.ErrRecoveryWriteUnknown) || !errors.Is(err, context.Canceled) || !errors.Is(err, terminalFailure) {
		t.Fatal("unknown/cancel lost")
	}
	if err := <-stop; err != nil {
		t.Fatal(err)
	}
	if _, err := image.executable.Stat(); err == nil {
		t.Fatal("owned executable not closed")
	}
	if !image.closed || f.world.lease.closes.Load() != 1 || f.world.binds != 0 || f.target.calls != 0 {
		t.Fatal("image terminal ledger")
	}
}

func s414StopWhileImageOwned(t *testing.T, r recovery.Runtime, live chan context.Context, action chan error, lease *s414Lease) chan error {
	t.Helper()
	stop := make(chan error, 1)
	go func() { stop <- r.Stop(context.Background()) }()
	ctx := <-live
	<-ctx.Done()
	select {
	case <-action:
		t.Fatal("image owner abandoned join")
	default:
	}
	if lease.closes.Load() != 0 {
		t.Fatal("deployment returned before image terminal")
	}
	return stop
}

func testS414ImageUnwind(t *testing.T) {
	for _, mode := range []string{"ApprovePanic", "AcquirePanicRun", "HandoffPanic"} {
		t.Run(mode, func(t *testing.T) { testS414AcquisitionPanic(t, mode) })
	}
	for _, mode := range []string{"ReturnedOwnerError", "ReturnedOwnerErrorClosePanic", "SuccessfulOwnerClosePanic"} {
		t.Run(mode, func(t *testing.T) { testS414CleanupOutcome(t, mode) })
	}
	t.Run("SuccessfulTransfer", testS414ImageTransfer)
}

func s414CaughtPanic(action func()) (caught any) {
	defer func() { caught = recover() }()
	action()
	return nil
}

func testS414AcquisitionPanic(t *testing.T, mode string) {
	f := newS414Fixture(t, "backup")
	marker := errors.New("synthetic acquisition panic")
	var checked *os.File
	base, calls := f.tool.acquireImage, 0
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		calls++
		if calls == 2 {
			checked = file
			if mode == "AcquirePanicRun" {
				panic(marker)
			}
		}
		return base(ctx, name, digest, file)
	}
	if mode == "ApprovePanic" {
		f.tool.approveImage = func(_ context.Context, _ string, file *os.File) error { checked = file; panic(marker) }
	} else if mode == "HandoffPanic" {
		f.tool.useSource = func(context.Context, string, *os.File) error { panic(marker) }
	}
	r := f.runtime(t)
	caught := s414CaughtPanic(func() { _ = s414Enter(t, f, r) })
	stopErr, closeErr := r.Stop(context.Background()), f.driver.Close()
	if caught != marker || checked == nil || stopErr != nil || closeErr != nil {
		t.Fatal("panic identity or public Stop/Close lost")
	}
	s414AssertImageClosed(t, checked)
	wantBegin := 1
	if mode == "ApprovePanic" {
		wantBegin = 0
	}
	if f.world.begins != wantBegin || len(f.observations) != 0 || f.world.binds != 0 || f.world.lease.closes.Load() != 1 {
		t.Fatal("panic crossed Begin/child/lease boundary")
	}
	t.Logf("original_panic=true image_closed=true Begin=%d children=0 deployment_close=1", wantBegin)
}

// This wrapper panics only AFTER the existing software lease actually joins.
// It does not claim that an arbitrary provider panic proves terminal ownership.
type s414CleanupProbe struct {
	*s414ImageLease
	calls       int
	borrowedErr error
	failure     error
	marker      any
}

func (p *s414CleanupProbe) CloseAndJoin() error {
	p.calls++
	_, p.borrowedErr = p.original.ReadAt(make([]byte, 1), 0)
	err := p.s414ImageLease.CloseAndJoin()
	if p.marker != nil {
		panic(p.marker)
	}
	return errors.Join(err, p.failure)
}

func testS414CleanupOutcome(t *testing.T, mode string) {
	f := newS414Fixture(t, "backup")
	base := f.tool.acquireImage
	acquireErr, cleanupErr := errors.New("synthetic returned owner error"), errors.New("synthetic terminal cleanup error")
	marker := errors.New("synthetic terminal cleanup panic")
	var probe *s414CleanupProbe
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		lease, err := base(ctx, name, digest, file)
		if err != nil {
			return lease, err
		}
		probe = &s414CleanupProbe{s414ImageLease: lease.(*s414ImageLease), failure: cleanupErr}
		if mode != "ReturnedOwnerError" {
			probe.marker = marker
		}
		if mode == "SuccessfulOwnerClosePanic" {
			return probe, nil
		}
		return probe, acquireErr
	}
	r := f.runtime(t)
	var result error
	caught := s414CaughtPanic(func() { result = s414Enter(t, f, r) })
	if mode == "ReturnedOwnerError" {
		if caught != nil || !errors.Is(result, acquireErr) || !errors.Is(result, cleanupErr) {
			t.Fatal("returned acquisition/cleanup error cause lost")
		}
	} else if caught != marker {
		t.Fatal("cleanup panic identity swallowed or replaced")
	}
	if probe == nil || probe.calls != 1 || probe.borrowedErr != nil || !probe.closed {
		t.Fatal("returned owner not joined exactly once with borrowed handle live")
	}
	s414AssertImageClosed(t, probe.original)
	if r.Stop(context.Background()) != nil || f.driver.Close() != nil || probe.calls != 1 || f.world.lease.closes.Load() != 1 {
		t.Fatal("Stop/Close repeated image cleanup or failed to join")
	}
	if f.world.begins != 0 || len(f.observations) != 0 {
		t.Fatal("cleanup failure passed preflight")
	}
	t.Log("image_owner_close=1 actual_terminal=true borrowed_handle_live_during_join=true checked_handle_closed=true")
}

func testS414ImageTransfer(t *testing.T) {
	f := newS414Fixture(t, "backup")
	image, err := f.tool.openImage(context.Background())
	if err != nil || image == nil {
		t.Fatal("image acquisition failed")
	}
	lease := image.lease.(*s414ImageLease)
	if lease.closed {
		t.Fatal("owner closed before transfer")
	}
	if _, err = image.file.ReadAt(make([]byte, 1), 0); err != nil {
		t.Fatal("checked handle closed before transfer")
	}
	if err = image.closeAndJoin(); err != nil || !lease.closed {
		t.Fatal("transferred owner did not join")
	}
	s414AssertImageClosed(t, image.file)
}

func s414AssertImageClosed(t *testing.T, file *os.File) {
	t.Helper()
	_, err := file.ReadAt(make([]byte, 1), 0)
	if !errors.Is(err, os.ErrClosed) {
		// Retire a measured test leak only after recording the component failure.
		t.Error("component image descriptor remained open")
		if err := file.Close(); err != nil {
			t.Error("test leak retirement failed")
		}
	}
}

func testS414CompoundCauses(t *testing.T) {
	for _, primary := range []string{"Error", "OpaqueSlice"} {
		for _, cleanup := range []string{"Error", "PanicError", "PanicSlice"} {
			t.Run(primary+"_"+cleanup, func(t *testing.T) { testS414CompoundCase(t, primary, cleanup) })
		}
	}
	t.Run("ReturnedErrorAndCleanupError", testS414CompoundReturnedErrors)
	t.Run("CleanupOnlyPanicAfterChild", testS414CleanupOnlyRunPanic)
}

type s414CompoundFixture struct {
	*s414Fixture
	image *s414CleanupProbe
}

func newS414CompoundFixture(t *testing.T, failure error, marker any) *s414CompoundFixture {
	t.Helper()
	f := &s414CompoundFixture{s414Fixture: newS414Fixture(t, "backup")}
	base, calls := f.tool.acquireImage, 0
	f.tool.acquireImage = func(ctx context.Context, name, digest string, file *os.File) (restoreImageLease, error) {
		lease, err := base(ctx, name, digest, file)
		calls++
		if err == nil && calls == 2 {
			f.image = &s414CleanupProbe{s414ImageLease: lease.(*s414ImageLease), failure: failure, marker: marker}
			return f.image, nil
		}
		return lease, err
	}
	return f
}

func s414OpaquePanic(kind string) any {
	if kind == "Error" || kind == "PanicError" {
		return errors.New(s414Canary)
	}
	return []byte(s414Canary)
}

func s414SameOpaquePanic(got, want any) bool {
	if expected, ok := want.([]byte); ok {
		actual, ok := got.([]byte)
		return ok && len(actual) == len(expected) && len(actual) > 0 && &actual[0] == &expected[0]
	}
	return got == want
}

func testS414CompoundCase(t *testing.T, primaryKind, cleanupKind string) {
	primary, secondary := s414OpaquePanic(primaryKind), s414OpaquePanic(cleanupKind)
	var failure error
	var marker any
	if cleanupKind == "Error" {
		failure = secondary.(error)
	} else {
		marker = secondary
	}
	f := newS414CompoundFixture(t, failure, marker)
	f.tool.useSource = func(context.Context, string, *os.File) error { panic(primary) }
	r := f.runtime(t)
	caught := s414CaughtPanic(func() { _ = s414Enter(t, f.s414Fixture, r) })
	f.assertTerminal(t, r, 0)
	if !s414SameOpaquePanic(caught, primary) {
		t.Error("active main panic identity replaced")
	}
	s414AssertSecondaryCause(t, f.tool.takeCleanupFailure(), secondary, cleanupKind != "Error")
	if f.tool.takeCleanupFailure() != nil {
		t.Error("secondary drain retained an already consumed cause")
	}
}

func s414AssertSecondaryCause(t *testing.T, cause error, secondary any, raised bool) {
	t.Helper()
	if cause == nil || strings.Contains(cause.Error(), s414Canary) {
		t.Fatal("private secondary missing or unsafe presentation")
	}
	if expected, ok := secondary.(error); ok && !errors.Is(cause, expected) {
		t.Error("secondary error identity lost")
	}
	if raised {
		var captured restoreCleanupPanic
		if !errors.As(cause, &captured) || !s414SameOpaquePanic(captured.value, secondary) {
			t.Error("opaque secondary panic value lost")
		}
	}
}

func (f *s414CompoundFixture) assertTerminal(t *testing.T, r recovery.Runtime, children int) {
	t.Helper()
	if r.Stop(context.Background()) != nil || f.driver.Close() != nil {
		t.Fatal("public Stop/driver Close failed")
	}
	if f.image == nil || f.image.calls != 1 || !f.image.closed || f.image.borrowedErr != nil {
		t.Fatal("actual image join or borrowed handle lifetime failed")
	}
	s414AssertImageClosed(t, f.image.original)
	if f.world.begins != 1 || f.world.binds != 0 || f.target.calls != 0 || f.world.lease.closes.Load() != 1 || len(f.observations) != children {
		t.Fatal("public producer or actual terminal controls changed")
	}
	for _, child := range f.observations {
		if child.PID <= 0 || !child.Waited || !child.PipesJoined {
			t.Error("actual child or pipe not joined")
		}
	}
	t.Logf("image_owner_close=1 actual_terminal=true checked_fd_closed=true Begin=1 children=%d deployment_close=1", children)
}

func testS414CompoundReturnedErrors(t *testing.T) {
	primary, secondary := errors.New(s414Canary), errors.New(s414Canary)
	f := newS414CompoundFixture(t, secondary, nil)
	f.tool.useSource = func(context.Context, string, *os.File) error { return primary }
	r := f.runtime(t)
	result := s414Enter(t, f.s414Fixture, r)
	f.assertTerminal(t, r, 0)
	if !errors.Is(result, primary) || !errors.Is(result, secondary) || !errors.Is(result, recovery.ErrRecoveryWriteUnknown) || strings.Contains(result.Error(), s414Canary) {
		t.Error("returned action/cleanup causes or private error boundary changed")
	}
	if f.tool.takeCleanupFailure() != nil {
		t.Error("normal returned errors incorrectly stored as panic diagnostics")
	}
}

func testS414CleanupOnlyRunPanic(t *testing.T) {
	marker := errors.New(s414Canary)
	f := newS414CompoundFixture(t, nil, marker)
	r := f.runtime(t)
	caught := s414CaughtPanic(func() { _ = s414Enter(t, f.s414Fixture, r) })
	f.assertTerminal(t, r, 1)
	if caught != marker || f.tool.takeCleanupFailure() != nil {
		t.Error("cleanup-only panic identity replaced or swallowed")
	}
}

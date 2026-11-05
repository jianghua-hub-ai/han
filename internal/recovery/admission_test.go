package recovery

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

type admissionFixture struct {
	mu                               sync.Mutex
	snapshot                         AdmissionSnapshot
	recovery, article                VerifiedJournalSnapshot
	closure                          PublishedClosure
	closures                         map[string]PublishedClosure
	genesis                          GenesisProof
	readErr, lockErr, startErr       error
	hook                             func(string)
	start                            func(context.Context, PhasePermit) error
	stop                             func(context.Context) error
	ledger                           []string
	starts, stops, closes, snapshots int
	control                          *controlledFixture
}

type controlledFixture struct {
	secretCandidate, lastCandidate                                   []byte
	f                                                                *admissionFixture
	r                                                                Runtime
	complete                                                         AdmissionRun
	closure                                                          PublishedClosure
	begin                                                            JournalEntry
	plan                                                             RestorePlan
	steps                                                            []string
	secretReads, secretVerifies, begins, replaces, consumes, revokes int
	verifyErr, exitErr                                               error
	verifierEntry                                                    func(JournalEntry) JournalEntry
	exitEntered, exitRelease                                         chan struct{}
	saved                                                            []PhasePermit
}

// r03BoundaryPorts counts invocation before the delegate checks its capability.
// Its hook changes independent source facts only after the selected owner returns.
type r03BoundaryPorts struct {
	*controlledFixture
	calls   map[string]int
	hook    func(string)
	consume func(context.Context, string) (ConsumptionResult, error)
}

func (w *r03BoundaryPorts) called(name string) {
	w.calls[name]++
	if w.hook != nil {
		w.hook(name)
	}
}

func (w *r03BoundaryPorts) Prepare(ctx context.Context, b RunBinding) (RestorePlan, error) {
	x, e := w.controlledFixture.Prepare(ctx, b)
	w.called("prepare")
	return x, e
}
func (w *r03BoundaryPorts) Replace(ctx context.Context, p PhasePermit, plan RestorePlan) (RestoreReceipt, error) {
	x, e := w.controlledFixture.Replace(ctx, p, plan)
	w.called("replace")
	return x, e
}
func (w *r03BoundaryPorts) VerifyRestored(ctx context.Context, plan RestorePlan) (RestoreReceipt, error) {
	x, e := w.controlledFixture.VerifyRestored(ctx, plan)
	w.called("verify-restored")
	return x, e
}
func (w *r03BoundaryPorts) BindRestoredRun(ctx context.Context, p PhasePermit, b RunBinding, x RestoreReceipt) (RecoveryRun, error) {
	w.calls["bind"]++
	r, e := w.controlledFixture.BindRestoredRun(ctx, p, b, x)
	if w.hook != nil {
		w.hook("bind")
	}
	return r, e
}
func (w *r03BoundaryPorts) FinalizeRestoredIdentity(ctx context.Context, p PhasePermit, b RunBinding) error {
	w.calls["finalize"]++
	e := w.controlledFixture.FinalizeRestoredIdentity(ctx, p, b)
	if w.hook != nil {
		w.hook("finalize")
	}
	return e
}
func (w *r03BoundaryPorts) PrepareRecoveryTrust(ctx context.Context, p PhasePermit, b RunBinding, secret []byte) error {
	w.calls["trust"]++
	e := w.controlledFixture.PrepareRecoveryTrust(ctx, p, b, secret)
	if w.hook != nil {
		w.hook("trust")
	}
	return e
}
func (w *r03BoundaryPorts) CompleteIdentityReview(ctx context.Context, p PhasePermit, b RunBinding) error {
	w.calls["identity-ready"]++
	e := w.controlledFixture.CompleteIdentityReview(ctx, p, b)
	if w.hook != nil {
		w.hook("identity-ready")
	}
	return e
}
func (w *r03BoundaryPorts) Consume(ctx context.Context, id string) (ConsumptionResult, error) {
	w.calls["consume"]++
	consume := w.consume
	if consume == nil {
		consume = w.controlledFixture.Consume
	}
	x, e := consume(ctx, id)
	if w.hook != nil {
		w.hook("consume")
	}
	return x, e
}
func (w *r03BoundaryPorts) ExitRecovery(ctx context.Context, p PhasePermit, b RunBinding) error {
	w.calls["exit"]++
	e := w.controlledFixture.ExitRecovery(ctx, p, b)
	if w.hook != nil {
		w.hook("exit")
	}
	return e
}
func (w *r03BoundaryPorts) ReadInitialRecoverySecret(ctx context.Context, b RunBinding) ([]byte, error) {
	x, e := w.controlledFixture.ReadInitialRecoverySecret(ctx, b)
	w.called("secret-read")
	return x, e
}

func r03Runtime(t *testing.T, c *controlledFixture, old Runtime) (*r03BoundaryPorts, Runtime) {
	t.Helper()
	w := &r03BoundaryPorts{controlledFixture: c, calls: make(map[string]int)}
	d := old.(*admissionRuntime).deps
	d.Restore, d.Identity, d.Consumer, d.Exit, d.RecoverySecret = w, w, w, w, w
	r, e := NewRuntime(context.Background(), d)
	if e != nil {
		t.Fatal(e)
	}
	c.r = r
	t.Cleanup(func() {
		if e := r.Stop(context.Background()); e != nil {
			t.Error(e)
		}
	})
	return w, r
}

type r03BeginJournal struct {
	ContextRecoveryJournal
	after func()
}

func (w r03BeginJournal) Begin(ctx context.Context, run, backup string, secret []byte) (JournalEntry, error) {
	x, e := w.ContextRecoveryJournal.Begin(ctx, run, backup, secret)
	w.after()
	return x, e
}

func newControlledFixture(t *testing.T, stage RecoveryOutcome) (*controlledFixture, Runtime) {
	t.Helper()
	f := newAdmissionFixture(true)
	source := newAdmissionFixture(true)
	x := source.snapshot.Runs[0]
	x.RunID = "run-current"
	x.BackupRef = "backup-current"
	receipt := *x.RestoreBinding
	receipt.RunID = x.RunID
	receipt.BackupRef = x.BackupRef
	x.RestoreBinding = &receipt
	cl := source.closure
	cl.Ready.RunID = x.RunID
	cl.Ready.BackupRef = x.BackupRef
	cl.Ready.EvidenceDigest = oracleReadyDigest(cl.Ready)
	x.ArticleEvidenceDigest = fixtureText(cl.Ready.EvidenceDigest)
	c := &controlledFixture{f: f, complete: x, closure: cl}
	f.control = c
	c.plan = RestorePlan{Version: "han-db-restore/v1", RunID: x.RunID, BackupRef: x.BackupRef, DeploymentID: "deployment", InstallationID: "installation", TargetDigest: receipt.TargetDigest, ArtifactDigest: receipt.ArtifactDigest, SourceApprovalRef: "approval", SourceApprovalDigest: receipt.SourceApprovalDigest, SourceInstallationID: "source-installation", SourceSchemaDigest: receipt.RestoredSchemaDigest, ArtifactPath: "synthetic-artifact", ArtifactBytes: receipt.ArtifactBytes}
	if stage != "" {
		c.appendBegin()
		c.putStage(stage)
		if stage == RecoveryOutcomeSucceeded {
			c.publishConsumed()
		}
	}
	r, e := NewRuntime(context.Background(), RuntimeDependencies{Store: f, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: f, Lifecycle: f, Runs: f, Identity: c, Restore: c, Consumer: c, Exit: c, RecoverySecret: c})
	if e != nil {
		t.Fatal(e)
	}
	c.r = r
	t.Cleanup(func() {
		if e := r.Stop(context.Background()); e != nil {
			t.Error(e)
		}
	})
	return c, r
}

func (c *controlledFixture) binding() RunBinding {
	return RunBinding{RunID: c.complete.RunID, BackupRef: c.complete.BackupRef}
}
func (c *controlledFixture) appendBegin() {
	f := c.f
	seq := f.recovery.Head.Sequence + 1
	c.begin = JournalEntry{Namespace: RecoveryNamespaceV1, Sequence: seq, RunID: c.complete.RunID, BackupRef: c.complete.BackupRef, KeyID: c.complete.JournalKeyID, KeyVersion: c.complete.JournalKeyVersion, SecretFingerprint: c.complete.JournalSecretFingerprint, Stage: StageBegun, RecordedAt: c.complete.StartedAt, PreviousRecordHash: f.recovery.Head.Hash}
	c.begin = fixtureSignEntry(c.begin)
	f.recovery.Entries = append(f.recovery.Entries, c.begin)
	f.recovery.Head = JournalHead{Sequence: seq, Hash: fixtureHash("current-begin-record")}
	f.recovery.Prefixes = append(f.recovery.Prefixes, f.recovery.Head)
	f.recovery.Revision += "-begin"
}

func (c *controlledFixture) putStage(stage RecoveryOutcome) {
	x := c.complete
	x.Outcome = stage
	if stage != RecoveryOutcomeSucceeded {
		x.JournalStage = StageBegun
		x.JournalConsumedAt = nil
		x.CompletedAt = nil
		x.ArticleEvidenceVersion = nil
		x.ArticleEvidenceDigest = nil
		x.ArticleEvidenceRef = nil
		x.ArticleEvidenceCompletedAt = nil
		x.ArticleNamespace = nil
		x.ArticleReplayVerifiedAt = nil
		x.ArticlePublicationRef = nil
		x.ArticlePublicationDigest = nil
	}
	if stage == RecoveryOutcomeRunning {
		x.AuditEventID = nil
		x.InitialAdminUserID = nil
		x.InitialAdminBootstrapGeneration = nil
		x.InitialAdminChangedGeneration = nil
		x.InitialAdminChangeAuditID = nil
		x.InitialAdminChangedAt = nil
	}
	f := c.f
	found := false
	for i, r := range f.snapshot.Runs {
		if r.RunID == x.RunID {
			f.snapshot.Runs[i] = x
			found = true
		}
	}
	if !found {
		f.snapshot.Runs = append(f.snapshot.Runs, x)
	}
	f.snapshot.Revision.Counter++
}

func (c *controlledFixture) publishConsumed() {
	f := c.f
	x := c.begin
	x.Sequence = f.recovery.Head.Sequence + 1
	x.Stage = StageConsumed
	x.RecordedAt = *c.complete.JournalConsumedAt
	x.PreviousRecordHash = f.recovery.Head.Hash
	x = fixtureSignEntry(x)
	f.recovery.Entries = append(f.recovery.Entries, x)
	f.recovery.Head = JournalHead{Sequence: x.Sequence, Hash: fixtureHash("current-consumed")}
	f.recovery.Prefixes = append(f.recovery.Prefixes, f.recovery.Head)
	f.recovery.Revision += "-consumed"
	if f.closures == nil {
		f.closures = make(map[string]PublishedClosure)
	}
	f.closures[c.complete.RunID] = c.closure
}

func (c *controlledFixture) step(ctx context.Context, p PhasePermit, b RunBinding, purpose, name string) error {
	if b != c.binding() {
		return ErrAdmissionBinding
	}
	if e := c.r.CheckPhase(ctx, p, purpose); e != nil {
		return e
	}
	x, e := c.r.ReadPermitBinding(ctx)
	if e != nil || x.RunID != b.RunID || x.BackupRef != b.BackupRef || x.InstallationID != "installation" {
		return ErrAdmissionBinding
	}
	if purpose == PhaseRecoveryRestore || purpose == PhaseRecoveryBind {
		for _, use := range []string{JournalUseCommittedRead, JournalUseNewAppend, JournalUseCoordinate} {
			if c.r.CheckJournalUse(ctx, "op", use) == nil {
				return errors.New("pre-purpose authorized Journal use")
			}
		}
	}
	for _, old := range c.saved {
		if c.r.CheckPhase(ctx, old, old.grant.binding.Purpose) == nil && old.grant != p.grant {
			return errors.New("previous grant survived")
		}
	}
	c.steps = append(c.steps, name)
	c.saved = append(c.saved, p)
	return nil
}

func (c *controlledFixture) Prepare(ctx context.Context, b RunBinding) (RestorePlan, error) {
	c.steps = append(c.steps, "prepare")
	if b != c.binding() {
		return RestorePlan{}, ErrAdmissionBinding
	}
	return c.plan, ctx.Err()
}
func (c *controlledFixture) Replace(ctx context.Context, p PhasePermit, plan RestorePlan) (RestoreReceipt, error) {
	if e := c.step(ctx, p, c.binding(), PhaseRecoveryRestore, "replace"); e != nil {
		return RestoreReceipt{}, e
	}
	c.replaces++
	return *c.complete.RestoreBinding, nil
}
func (c *controlledFixture) VerifyRestored(ctx context.Context, plan RestorePlan) (RestoreReceipt, error) {
	c.steps = append(c.steps, "verify-restored")
	return *c.complete.RestoreBinding, ctx.Err()
}
func (c *controlledFixture) BindRestoredRun(ctx context.Context, p PhasePermit, b RunBinding, receipt RestoreReceipt) (RecoveryRun, error) {
	if e := c.step(ctx, p, b, PhaseRecoveryBind, "bind"); e != nil {
		return RecoveryRun{}, e
	}
	c.putStage(RecoveryOutcomeRunning)
	return c.f.Load(ctx, b.RunID)
}
func (c *controlledFixture) FinalizeRestoredIdentity(ctx context.Context, p PhasePermit, b RunBinding) error {
	if e := c.step(ctx, p, b, PhaseRecoveryIdentity, "finalize"); e != nil {
		return e
	}
	for i := range c.f.snapshot.Runs {
		if c.f.snapshot.Runs[i].RunID == b.RunID {
			c.f.snapshot.Runs[i].AuditEventID = c.complete.AuditEventID
		}
	}
	c.f.snapshot.Revision.Counter++
	return nil
}
func (c *controlledFixture) PrepareRecoveryTrust(ctx context.Context, p PhasePermit, b RunBinding, secret []byte) error {
	if e := c.step(ctx, p, b, PhaseRecoveryIdentity, "trust"); e != nil {
		return e
	}
	if !bytes.Equal(secret, []byte("synthetic-candidate")) {
		return ErrAdmissionBinding
	}
	for i := range c.f.snapshot.Runs {
		r := &c.f.snapshot.Runs[i]
		if r.RunID == b.RunID {
			r.InitialAdminUserID = c.complete.InitialAdminUserID
			r.InitialAdminBootstrapGeneration = c.complete.InitialAdminBootstrapGeneration
		}
	}
	c.f.snapshot.Revision.Counter++
	return nil
}
func (c *controlledFixture) CompleteIdentityReview(ctx context.Context, p PhasePermit, b RunBinding) error {
	if e := c.step(ctx, p, b, PhaseRecoveryIdentity, "identity-ready"); e != nil {
		return e
	}
	c.putStage(RecoveryOutcomeIdentityReady)
	return nil
}
func (c *controlledFixture) Consume(ctx context.Context, id string) (ConsumptionResult, error) {
	b, e := c.r.ReadPermitBinding(ctx)
	if e != nil || b.Purpose != PhaseRecoveryReplay || id != c.complete.RunID {
		return ConsumptionResult{}, ErrAdmissionBinding
	}
	c.steps = append(c.steps, "consume")
	c.consumes++
	c.putStage(RecoveryOutcomeSucceeded)
	c.publishConsumed()
	r, e := c.f.Load(ctx, id)
	return ConsumptionResult{Run: r, Evidence: c.closure.Ready, Journal: c.f.recovery.Entries[len(c.f.recovery.Entries)-1]}, e
}
func (c *controlledFixture) ExitRecovery(ctx context.Context, p PhasePermit, b RunBinding) error {
	if e := c.step(ctx, p, b, PhaseRecoveryReplay, "exit"); e != nil {
		return e
	}
	if c.exitErr != nil {
		return c.exitErr
	}
	if c.exitEntered != nil {
		close(c.exitEntered)
		<-c.exitRelease
	}
	return c.RevokeRecoverySessions(ctx, p, b)
}
func (c *controlledFixture) RevokeRecoverySessions(ctx context.Context, p PhasePermit, b RunBinding) error {
	if e := c.step(ctx, p, b, PhaseRecoveryReplay, "revoke"); e != nil {
		return e
	}
	r, e := c.f.Load(ctx, b.RunID)
	if e != nil || r.Outcome != RecoveryOutcomeSucceeded {
		return ErrAdmissionBinding
	}
	c.revokes++
	return nil
}
func (c *controlledFixture) ReadInitialRecoverySecret(context.Context, RunBinding) ([]byte, error) {
	c.secretReads++
	c.lastCandidate = []byte("synthetic-candidate")
	if c.secretCandidate != nil {
		c.lastCandidate = append([]byte(nil), c.secretCandidate...)
	}
	return c.lastCandidate, nil
}

func (f *admissionFixture) Begin(ctx context.Context, run, backup string, secret []byte) (JournalEntry, error) {
	if f.control == nil {
		return JournalEntry{}, ErrAdmissionBinding
	}
	c := f.control
	if c.binding() != (RunBinding{RunID: run, BackupRef: backup}) || !bytes.Equal(secret, []byte("synthetic-candidate")) {
		return JournalEntry{}, ErrAdmissionBinding
	}
	c.begins++
	c.steps = append(c.steps, "begin")
	c.appendBegin()
	return c.begin, ctx.Err()
}
func (f *admissionFixture) VerifyBeginSecret(ctx context.Context, b RunBinding, secret []byte) (JournalEntry, error) {
	if f.control == nil {
		return JournalEntry{}, ErrAdmissionBinding
	}
	c := f.control
	c.secretVerifies++
	c.steps = append(c.steps, "verify-secret")
	if c.verifyErr != nil {
		return JournalEntry{}, c.verifyErr
	}
	if b != c.binding() || !bytes.Equal(secret, []byte("synthetic-candidate")) {
		return JournalEntry{}, ErrAdmissionBinding
	}
	entry := c.begin
	if c.verifierEntry != nil {
		entry = c.verifierEntry(entry)
	}
	return entry, ctx.Err()
}
func (f *admissionFixture) MarkConsumed(context.Context, string) (JournalEntry, error) {
	return JournalEntry{}, ErrAdmissionBinding
}
func (f *admissionFixture) Read(context.Context, string) (JournalEntry, error) {
	return JournalEntry{}, ErrAdmissionBinding
}
func (f *admissionFixture) VerifyAgainst(context.Context, RecoveryRun) (JournalEntry, error) {
	return JournalEntry{}, ErrAdmissionBinding
}
func (f *admissionFixture) Load(ctx context.Context, id string) (RecoveryRun, error) {
	for _, x := range f.snapshot.Runs {
		if x.RunID == id {
			return fixtureRecoveryRun(x), ctx.Err()
		}
	}
	return RecoveryRun{}, ErrRunNotFound
}
func (f *admissionFixture) Transition(context.Context, string, RecoveryOutcome, RecoveryOutcome, RecoveryRunChange) (RecoveryRun, error) {
	return RecoveryRun{}, ErrAdmissionBinding
}
func fixtureRecoveryRun(x AdmissionRun) RecoveryRun {
	r := RecoveryRun{RunID: x.RunID, BackupRef: x.BackupRef, JournalKeyID: x.JournalKeyID, JournalKeyVersion: x.JournalKeyVersion, JournalSecretFingerprint: x.JournalSecretFingerprint, JournalStage: x.JournalStage, StartedAt: x.StartedAt, JournalConsumedAt: x.JournalConsumedAt, Outcome: x.Outcome}
	if x.ArticleEvidenceVersion != nil {
		r.ArticleEvidenceVersion = *x.ArticleEvidenceVersion
	}
	if x.ArticleEvidenceDigest != nil {
		r.ArticleEvidenceDigest = *x.ArticleEvidenceDigest
	}
	if x.ArticleEvidenceRef != nil {
		r.ArticleEvidenceRef = *x.ArticleEvidenceRef
	}
	if x.ArticleEvidenceCompletedAt != nil {
		r.ArticleEvidenceCompletedAt = *x.ArticleEvidenceCompletedAt
	}
	if x.ArticleNamespace != nil {
		r.ArticleNamespace = *x.ArticleNamespace
	}
	if x.ArticlePublicationRef != nil {
		r.ArticlePublicationRef = *x.ArticlePublicationRef
	}
	if x.ArticlePublicationDigest != nil {
		r.ArticlePublicationDigest = *x.ArticlePublicationDigest
	}
	return r
}

func fixtureText(s string) *string       { return &s }
func fixtureTime(t time.Time) *time.Time { return &t }
func fixtureInt(n int64) *int64          { return &n }
func fixtureHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func newAdmissionFixture(history bool) *admissionFixture {
	now := time.Date(2026, 10, 20, 1, 0, 0, 123456000, time.UTC)
	r := AnchorSnapshot{JournalID: "recovery-journal", Namespace: RecoveryNamespaceV1, Epoch: 1}
	a := AnchorSnapshot{JournalID: "article-journal", Namespace: ArticleDeletionNamespaceV1, Epoch: 2}
	i := InstallationBinding{Version: InstallationVersion, DeploymentID: "deployment", InstallationID: "installation", TargetDigest: fixtureHash("target"), CommitmentRef: "commitment", CommitmentDigest: fixtureHash("commitment"), RecoveryInitial: r, ArticleInitial: a, InstalledAt: now}
	f := &admissionFixture{snapshot: AdmissionSnapshot{Installation: i, Revision: StoreRevision{InstallationID: i.InstallationID, Counter: 1}, Runs: []AdmissionRun{}}}
	f.recovery = VerifiedJournalSnapshot{JournalID: r.JournalID, Namespace: r.Namespace, Epoch: r.Epoch, Revision: "recovery-r0", Entries: []JournalEntry{}, Prefixes: []JournalHead{{}}}
	f.article = VerifiedJournalSnapshot{JournalID: a.JournalID, Namespace: a.Namespace, Epoch: a.Epoch, Revision: "article-r0", Prefixes: []JournalHead{{}}}
	f.genesis = GenesisProof{Binding: i, RecoveryInitial: r, ArticleInitial: a, CommitmentRevision: "commitment-r1", CommitmentDigest: i.CommitmentDigest}
	if history {
		f.addHistory(now)
	}
	return f
}

func (f *admissionFixture) addHistory(now time.Time) {
	consumed := now.Add(time.Minute)
	h := JournalHead{Sequence: 1, Hash: fixtureHash("article-head")}
	f.article.Head = h
	f.article.Prefixes = append(f.article.Prefixes, h)
	b := JournalEntry{Namespace: RecoveryNamespaceV1, Sequence: 1, RunID: "run-a", BackupRef: "backup-a", KeyID: "key", KeyVersion: "v1", SecretFingerprint: fingerprintSecret(fixtureJournalKey(), []byte("synthetic-candidate")), Stage: StageBegun, RecordedAt: now}
	b = fixtureSignEntry(b)
	c := b
	c.Sequence = 2
	c.Stage = StageConsumed
	c.RecordedAt = consumed
	c.PreviousRecordHash = fixtureHash("begin-record")
	c = fixtureSignEntry(c)
	f.recovery.Entries = []JournalEntry{b, c}
	f.recovery.Head = JournalHead{Sequence: 2, Hash: fixtureHash("recovery-head")}
	f.recovery.Prefixes = append(f.recovery.Prefixes, JournalHead{Sequence: 1, Hash: c.PreviousRecordHash}, f.recovery.Head)
	r := AdmissionRun{RunID: b.RunID, BackupRef: b.BackupRef, JournalKeyID: b.KeyID, JournalKeyVersion: b.KeyVersion, JournalSecretFingerprint: b.SecretFingerprint, JournalStage: StageConsumed, StartedAt: now, JournalConsumedAt: &consumed, ArticleEvidenceVersion: fixtureText(ReplayEvidenceVersionV1), ArticleEvidenceRef: fixtureText("evidence-a"), ArticleNamespace: fixtureText(ArticleDeletionNamespaceV1), ArticleEvidenceCompletedAt: &consumed, ArticleReplayVerifiedAt: &consumed, CompletedAt: &consumed, ArticlePublicationRef: fixtureText("publication-a"), ArticlePublicationDigest: fixtureText(fixtureHash("publication")), AuditEventID: fixtureText("11111111-1111-4111-8111-111111111111"), Outcome: RecoveryOutcomeSucceeded, AccountCount: 2, InitialAdminUserID: fixtureText("22222222-2222-4222-8222-222222222222"), InitialAdminBootstrapGeneration: fixtureInt(2), InitialAdminChangedGeneration: fixtureInt(3), InitialAdminChangeAuditID: fixtureText("33333333-3333-4333-8333-333333333333"), InitialAdminChangedAt: &consumed}
	r.RestoreBinding = &RestoreReceipt{Version: "han-db-restore/v1", RunID: r.RunID, BackupRef: r.BackupRef, DeploymentID: "deployment", InstallationID: "installation", TargetDigest: fixtureHash("target"), ArtifactDigest: fixtureHash("artifact"), SourceApprovalDigest: fixtureHash("approval"), RestoredSchemaDigest: fixtureHash("schema"), ReceiptDigest: fixtureHash("receipt"), ArtifactBytes: 100, RestoredAt: now}
	ready := ReadyReplayStatusV1{EvidenceVersion: ReplayEvidenceVersionV1, RunID: r.RunID, BackupRef: r.BackupRef, Namespace: ArticleDeletionNamespaceV1, JournalHeadSequence: h.Sequence, JournalHeadHash: h.Hash, InventoryScope: "recoverable-backups-before-cutoff", BackupCutoff: "2026-10-20T00:00:00Z", InventoryDigest: fixtureHash("inventory"), DeletionOperationsDigest: fixtureHash("operations"), AuditEventsDigest: fixtureHash("audits"), Status: "ready", EvidenceRef: *r.ArticleEvidenceRef, CompletedAt: consumed.Format(time.RFC3339Nano)}
	ready.EvidenceDigest = oracleReadyDigest(ready)
	r.ArticleEvidenceDigest = fixtureText(ready.EvidenceDigest)
	f.snapshot.Runs = []AdmissionRun{r}
	f.closure = PublishedClosure{Ready: ready, PublicationRef: *r.ArticlePublicationRef, PublicationDigest: *r.ArticlePublicationDigest, JournalID: f.article.JournalID, Epoch: f.article.Epoch, PublishedAt: consumed, VerifiedArticleRevision: f.article.Revision, CurrentHead: h, HistoricalHead: h}
}

// These independent wire oracles enumerate the design's order explicitly.
func oracleJSON(v any) []byte {
	if t, ok := v.(time.Time); ok {
		v = t.Format("2006-01-02T15:04:05.000000Z")
	}
	if t, ok := v.(*time.Time); ok && t != nil {
		v = t.Format("2006-01-02T15:04:05.000000Z")
	}
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		panic(err)
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n"))
}

func oracleObject(names string, values ...any) []byte {
	keys := strings.Fields(names)
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(oracleJSON(k))
		b.WriteByte(':')
		b.Write(oracleJSON(values[i]))
	}
	b.WriteByte('}')
	return b.Bytes()
}

func oracleRun(r AdmissionRun) json.RawMessage {
	var receipt json.RawMessage = []byte("null")
	if x := r.RestoreBinding; x != nil {
		receipt = oracleObject("receipt_version deployment_id installation_id backup_ref artifact_digest artifact_bytes target_digest source_approval_digest restored_schema_digest receipt_digest restored_at", x.Version, x.DeploymentID, x.InstallationID, x.BackupRef, x.ArtifactDigest, x.ArtifactBytes, x.TargetDigest, x.SourceApprovalDigest, x.RestoredSchemaDigest, x.ReceiptDigest, x.RestoredAt)
	}
	return oracleObject("run_id backup_ref journal_key_id journal_key_version journal_secret_fingerprint journal_stage started_at journal_consumed_at article_evidence_version article_evidence_digest article_evidence_ref article_evidence_completed_at article_namespace article_replay_verified_at article_publication_ref article_publication_digest completed_at outcome account_count permanent_credential_count session_count temporary_credential_count idempotency_count pending_review_count audit_event_id restore_binding initial_admin_user_id initial_admin_bootstrap_generation initial_admin_changed_generation initial_admin_change_audit_id initial_admin_changed_at", r.RunID, r.BackupRef, r.JournalKeyID, r.JournalKeyVersion, r.JournalSecretFingerprint, r.JournalStage, r.StartedAt, r.JournalConsumedAt, r.ArticleEvidenceVersion, r.ArticleEvidenceDigest, r.ArticleEvidenceRef, r.ArticleEvidenceCompletedAt, r.ArticleNamespace, r.ArticleReplayVerifiedAt, r.ArticlePublicationRef, r.ArticlePublicationDigest, r.CompletedAt, r.Outcome, r.AccountCount, r.PermanentCredentialCount, r.SessionCount, r.TemporaryCredentialCount, r.IdempotencyCount, r.PendingReviewCount, r.AuditEventID, receipt, r.InitialAdminUserID, r.InitialAdminBootstrapGeneration, r.InitialAdminChangedGeneration, r.InitialAdminChangeAuditID, r.InitialAdminChangedAt)
}

func oracleSnapshot(s AdmissionSnapshot) []byte {
	i := s.Installation
	r, a := i.RecoveryInitial, i.ArticleInitial
	installation := oracleObject("metadata_version deployment_id installation_id target_digest commitment_ref commitment_digest recovery_journal_id recovery_epoch recovery_initial_sequence recovery_initial_hash article_journal_id article_epoch article_initial_sequence article_initial_hash revision installed_at", i.Version, i.DeploymentID, i.InstallationID, i.TargetDigest, i.CommitmentRef, i.CommitmentDigest, r.JournalID, r.Epoch, r.Committed.Sequence, r.Committed.Hash, a.JournalID, a.Epoch, a.Committed.Sequence, a.Committed.Hash, s.Revision.Counter, i.InstalledAt)
	ordered := append([]AdmissionRun(nil), s.Runs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].RunID < ordered[j].RunID })
	runs := make([]json.RawMessage, 0, len(ordered))
	for _, r := range ordered {
		runs = append(runs, oracleRun(r))
	}
	return oracleObject("version installation runs", admissionVersion, json.RawMessage(installation), runs)
}

func oracleReadyDigest(r ReadyReplayStatusV1) string {
	b := oracleObject("evidence_version run_id backup_ref namespace journal_head_sequence journal_head_hash inventory_scope backup_cutoff inventory_digest deletion_operations_digest audit_events_digest tombstones_total tombstones_replayed already_absent deletion_operations_reconciled_count audit_events_verified_count", r.EvidenceVersion, r.RunID, r.BackupRef, r.Namespace, r.JournalHeadSequence, r.JournalHeadHash, r.InventoryScope, r.BackupCutoff, r.InventoryDigest, r.DeletionOperationsDigest, r.AuditEventsDigest, r.TombstonesTotal, r.TombstonesReplayed, r.AlreadyAbsent, r.DeletionOperationsReconciledCount, r.AuditEventsVerifiedCount)
	return fixtureHash(string(b))
}

func (f *admissionFixture) event(name string) {
	f.mu.Lock()
	f.ledger = append(f.ledger, name)
	hook := f.hook
	f.mu.Unlock()
	if hook != nil {
		hook(name)
	}
}
func (f *admissionFixture) ReadAdmissionSnapshot(context.Context) (AdmissionSnapshot, error) {
	f.event("store")
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshots++
	s := f.snapshot
	s.Revision.Digest = fixtureHash(string(oracleSnapshot(s)))
	return s, f.readErr
}
func (f *admissionFixture) ReadAdmissionRevision(context.Context) (StoreRevision, error) {
	f.event("revision")
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.snapshot.Revision
	r.Digest = fixtureHash(string(oracleSnapshot(f.snapshot)))
	return r, f.readErr
}
func (f *admissionFixture) ReadVerifiedSnapshot(_ context.Context, namespace string) (VerifiedJournalSnapshot, error) {
	f.event(namespace)
	f.mu.Lock()
	defer f.mu.Unlock()
	if namespace == RecoveryNamespaceV1 {
		return f.recovery, f.readErr
	}
	return f.article, f.readErr
}
func (f *admissionFixture) ReadPublishedClosure(_ context.Context, q ClosureRequest) (PublishedClosure, error) {
	f.event("closure")
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.closure
	if alternative, ok := f.closures[q.RunID]; ok {
		c = alternative
	}
	if q.RunID != c.Ready.RunID || q.BackupRef != c.Ready.BackupRef || q.EvidenceRef != c.Ready.EvidenceRef || q.EvidenceDigest != c.Ready.EvidenceDigest || q.PublicationRef != c.PublicationRef || q.PublicationDigest != c.PublicationDigest {
		return PublishedClosure{}, ErrAdmissionBinding
	}
	return c, f.readErr
}
func (f *admissionFixture) VerifyGenesis(context.Context, InstallationBinding) (GenesisProof, error) {
	f.event("genesis")
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.genesis, f.readErr
}
func (f *admissionFixture) Acquire(context.Context) (ExclusiveLease, error) {
	f.event("exclusive")
	return fixtureExclusive{f}, nil
}

type fixtureExclusive struct{ f *admissionFixture }

func (l fixtureExclusive) Check(ctx context.Context) error {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return l.f.lockErr
}
func (l fixtureExclusive) Close() error {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	l.f.closes++
	return nil
}
func (f *admissionFixture) ReconcileAndStart(ctx context.Context, p PhasePermit) error {
	f.mu.Lock()
	f.starts++
	fn, err := f.start, f.startErr
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx, p)
	}
	return err
}
func (f *admissionFixture) StopAndJoin(ctx context.Context) error {
	f.mu.Lock()
	f.stops++
	fn := f.stop
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx)
	}
	return nil
}

func fixtureRuntime(t *testing.T, f *admissionFixture) Runtime {
	t.Helper()
	r, err := NewRuntime(context.Background(), RuntimeDependencies{Store: f, Journal: f, ArticleSnapshots: f, Closures: f, Genesis: f, Exclusive: f, Lifecycle: f})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return r
}

func testFullHistory(t *testing.T) {
	t.Run("JournalHMACCompatibility", testJournalHMACCompatibility)
	t.Run("CompleteHistory", testCompleteHistory)
	cases := []struct {
		name   string
		change func(*admissionFixture)
	}{
		{"EarlierFailed", func(f *admissionFixture) {
			x := f.snapshot.Runs[0]
			x.RunID = "earlier"
			x.Outcome = RecoveryOutcomeFailed
			f.snapshot.Runs = append([]AdmissionRun{x}, f.snapshot.Runs...)
		}},
		{"EarlierUnfinished", func(f *admissionFixture) { f.snapshot.Runs[0].Outcome = RecoveryOutcomeRunning }},
		{"ExternalRunMissingDB", func(f *admissionFixture) { f.snapshot.Runs = []AdmissionRun{} }},
		{"DBRunMissingJournal", func(f *admissionFixture) { f.recovery.Entries = nil }},
		{"MissingConsumed", func(f *admissionFixture) { f.recovery.Entries = f.recovery.Entries[:1] }},
		{"DuplicateRun", func(f *admissionFixture) { f.snapshot.Runs = append(f.snapshot.Runs, f.snapshot.Runs[0]) }},
		{"ReadErrorNotEmpty", func(f *admissionFixture) { f.readErr = errors.New("synthetic rows failure") }},
		{"MissingChangeProof", func(f *admissionFixture) { f.snapshot.Runs[0].InitialAdminChangedAt = nil }},
		{"PendingReview", func(f *admissionFixture) { f.snapshot.Runs[0].PendingReviewCount = 1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newAdmissionFixture(true)
			tc.change(f)
			r := fixtureRuntime(t, f)
			if err := r.RunNormal(context.Background()); err == nil {
				t.Fatal("incomplete history admitted")
			}
		})
	}
	t.Run("EveryProjectionField", testProjectionFields)
	t.Run("CanonicalValidation", testCanonicalValidation)
	t.Run("CountEncodingProperty", testCountEncodingProperty)
}

func fixtureJournalKey() Key {
	return Key{ID: "key", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-admission-key")}
}

func fixtureDiskEntry(e JournalEntry) diskRecord {
	return diskRecord{Namespace: e.Namespace, Sequence: e.Sequence, RunID: e.RunID, BackupRef: e.BackupRef,
		KeyID: e.KeyID, KeyVersion: e.KeyVersion, SecretFingerprint: e.SecretFingerprint, Stage: e.Stage,
		RecordedAt: e.RecordedAt.Format(time.RFC3339Nano), PreviousRecordHash: e.PreviousRecordHash, RecordHMAC: e.RecordHMAC}
}

func fixtureSignEntry(e JournalEntry) JournalEntry {
	e.RecordHMAC = computeRecordHMAC(fixtureDiskEntry(e), fixtureJournalKey().Material)
	return e
}

func newHMACAdmissionFixture() *admissionFixture {
	f := newAdmissionFixture(true)
	fingerprint := fingerprintSecret(fixtureJournalKey(), []byte("synthetic-candidate"))
	f.snapshot.Runs[0].JournalSecretFingerprint = fingerprint
	f.recovery.Prefixes = []JournalHead{{}}
	previous := ""
	for i, e := range f.recovery.Entries {
		e.SecretFingerprint, e.PreviousRecordHash = fingerprint, previous
		e = fixtureSignEntry(e)
		f.recovery.Entries[i] = e
		raw, _ := encodeDiskRecord(fixtureDiskEntry(e))
		previous = SHA256Digest(raw)
		f.recovery.Prefixes = append(f.recovery.Prefixes, JournalHead{Sequence: e.Sequence, Hash: previous})
	}
	f.recovery.Head = f.recovery.Prefixes[len(f.recovery.Prefixes)-1]
	return f
}

func testJournalHMACCompatibility(t *testing.T) {
	t.Run("RealProducerStoreProjection", func(t *testing.T) {
		f := newHMACAdmissionFixture()
		// Independent Python stdlib HMAC oracle, fixed synthetic key/candidate.
		const fingerprint = "hmac-sha256:711bd631cb953d0d0fc89e53ff33b096e0d3a4c3a1a200f4c84b2d6e11e44794"
		if f.snapshot.Runs[0].JournalSecretFingerprint != fingerprint {
			t.Fatal("producer differs from independent HMAC oracle")
		}
		if _, e := CanonicalAdmissionSnapshot(f.snapshot); e != nil {
			t.Fatalf("real fingerprint rejected: %v", e)
		}
	})
	t.Run("RealProducerVerifiedEntries", func(t *testing.T) {
		f := newHMACAdmissionFixture()
		keyring, e := NewKeyring([]Key{fixtureJournalKey()})
		if e != nil {
			t.Fatal(e)
		}
		journal := &FileRecoveryJournal{keyring: keyring}
		records := make([]diskRecord, 0, len(f.recovery.Entries))
		for _, entry := range f.recovery.Entries {
			records = append(records, fixtureDiskEntry(entry))
		}
		if e := journal.validateRecords(records); e != nil {
			t.Fatalf("producer chain invalid: %v", e)
		}
		if e := validateRecoveryEntries(f.recovery); e != nil {
			t.Fatalf("real committed HMAC entries rejected: %v", e)
		}
	})
	t.Run("RealProducerCompleteHistory", func(t *testing.T) {
		f := newHMACAdmissionFixture()
		r := fixtureRuntime(t, f)
		if e := r.RunNormal(context.Background()); e != nil {
			t.Fatalf("real producer history closed: %v", e)
		}
		l, e := r.Acquire(context.Background(), WorkRequest)
		if e != nil {
			t.Fatal(e)
		}
		l.Release()
	})
	t.Run("RejectNonHMAC", testRejectNonHMACAdmission)
	t.Run("TrueSHAFieldsStayStrict", testHMACDoesNotWidenSHA)
}

func testRejectNonHMACAdmission(t *testing.T) {
	valid := fingerprintSecret(fixtureJournalKey(), []byte("synthetic-candidate"))
	digest := strings.TrimPrefix(valid, "hmac-sha256:")
	values := map[string]string{"plainSHA": "sha256:" + digest, "wrongPrefix": "hmac-sha512:" + digest,
		"short": "hmac-sha256:" + digest[:63], "long": valid + "0", "uppercaseHex": "hmac-sha256:" + strings.ToUpper(digest),
		"uppercasePrefix": "HMAC-SHA256:" + digest, "badHex": valid[:len(valid)-1] + "g", "empty": "", "whitespace": " " + valid}
	for name, value := range values {
		for _, field := range []string{"StoreFingerprint", "EntryFingerprint", "RecordHMAC"} {
			t.Run(field+"/"+name, func(t *testing.T) {
				f := newHMACAdmissionFixture()
				if field == "StoreFingerprint" {
					f.snapshot.Runs[0].JournalSecretFingerprint = value
					if _, e := CanonicalAdmissionSnapshot(f.snapshot); e == nil {
						t.Fatal("invalid HMAC projection accepted")
					}
				} else {
					if field == "EntryFingerprint" {
						f.recovery.Entries[0].SecretFingerprint = value
					} else {
						f.recovery.Entries[0].RecordHMAC = value
					}
					if e := validateRecoveryEntries(f.recovery); e == nil {
						t.Fatal("invalid HMAC entry accepted")
					}
				}
			})
		}
	}
}

func testHMACDoesNotWidenSHA(t *testing.T) {
	mac := fingerprintSecret(fixtureJournalKey(), []byte("synthetic-candidate"))
	mutations := map[string]func(*AdmissionSnapshot){
		"TargetDigest":      func(s *AdmissionSnapshot) { s.Installation.TargetDigest = mac },
		"CommitmentDigest":  func(s *AdmissionSnapshot) { s.Installation.CommitmentDigest = mac },
		"EvidenceDigest":    func(s *AdmissionSnapshot) { s.Runs[0].ArticleEvidenceDigest = fixtureText(mac) },
		"PublicationDigest": func(s *AdmissionSnapshot) { s.Runs[0].ArticlePublicationDigest = fixtureText(mac) },
		"RestoreTarget":     func(s *AdmissionSnapshot) { s.Runs[0].RestoreBinding.TargetDigest = mac },
		"ArtifactDigest":    func(s *AdmissionSnapshot) { s.Runs[0].RestoreBinding.ArtifactDigest = mac },
		"ApprovalDigest":    func(s *AdmissionSnapshot) { s.Runs[0].RestoreBinding.SourceApprovalDigest = mac },
		"SchemaDigest":      func(s *AdmissionSnapshot) { s.Runs[0].RestoreBinding.RestoredSchemaDigest = mac },
		"ReceiptDigest":     func(s *AdmissionSnapshot) { s.Runs[0].RestoreBinding.ReceiptDigest = mac },
		"InitialHead": func(s *AdmissionSnapshot) {
			s.Installation.RecoveryInitial.Committed = JournalHead{Sequence: 1, Hash: mac}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			f := newHMACAdmissionFixture()
			mutate(&f.snapshot)
			if _, e := CanonicalAdmissionSnapshot(f.snapshot); e == nil {
				t.Fatal("HMAC accepted in SHA field")
			}
		})
	}
	t.Run("SHAHeadAndRegex", func(t *testing.T) {
		if sha256DigestPattern.MatchString(mac) {
			t.Fatal("global SHA regex widened")
		}
		f := newHMACAdmissionFixture()
		f.recovery.Head.Hash = mac
		if e := validateVerifiedJournal(f.recovery, f.snapshot.Installation.RecoveryInitial); e == nil {
			t.Fatal("HMAC head accepted")
		}
	})
}

func testCanonicalValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*AdmissionSnapshot)
	}{
		{"NilRuns", func(s *AdmissionSnapshot) { s.Runs = nil }},
		{"CounterZero", func(s *AdmissionSnapshot) { s.Revision.Counter = 0 }},
		{"InvalidUTF8", func(s *AdmissionSnapshot) { s.Runs[0].BackupRef = string([]byte{255}) }},
		{"ControlText", func(s *AdmissionSnapshot) { s.Runs[0].BackupRef = "backup\nref" }},
		{"NonUTC", func(s *AdmissionSnapshot) {
			s.Runs[0].StartedAt = s.Runs[0].StartedAt.In(time.FixedZone("synthetic", 3600))
		}},
		{"NonMicrosecond", func(s *AdmissionSnapshot) { s.Runs[0].StartedAt = s.Runs[0].StartedAt.Add(time.Nanosecond) }},
		{"WrongReceipt", func(s *AdmissionSnapshot) {
			x := *s.Runs[0].RestoreBinding
			x.RunID = "wrong"
			s.Runs[0].RestoreBinding = &x
		}},
		{"OverflowEpoch", func(s *AdmissionSnapshot) { s.Installation.ArticleInitial.Epoch = 1 << 63 }},
		{"EmptyHeadMismatch", func(s *AdmissionSnapshot) { s.Installation.ArticleInitial.Committed.Hash = fixtureHash("unexpected") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newAdmissionFixture(true).snapshot
			tc.change(&s)
			if _, e := CanonicalAdmissionSnapshot(s); e == nil {
				t.Fatal("malformed projection encoded")
			}
		})
	}
	for _, name := range []string{"AccountCount", "PermanentCredentialCount", "SessionCount", "TemporaryCredentialCount", "IdempotencyCount", "PendingReviewCount"} {
		t.Run("Negative"+name, func(t *testing.T) {
			s := newAdmissionFixture(true).snapshot
			reflect.ValueOf(&s.Runs[0]).Elem().FieldByName(name).SetInt(-1)
			if _, e := CanonicalAdmissionSnapshot(s); e == nil {
				t.Fatal("negative count accepted")
			}
		})
	}
}

func testProjectionFields(t *testing.T) {
	f := newAdmissionFixture(true)
	s := f.snapshot
	b, err := CanonicalAdmissionSnapshot(s)
	if err != nil || !bytes.Equal(b, oracleSnapshot(s)) {
		t.Fatalf("canonical mismatch: %s / %v", b, err)
	}
	value := reflect.ValueOf(s.Runs[0])
	typ := value.Type()
	for n := 0; n < value.NumField(); n++ {
		t.Run(typ.Field(n).Name, func(t *testing.T) {
			copy := s
			copy.Runs = append([]AdmissionRun(nil), s.Runs...)
			v := reflect.ValueOf(&copy.Runs[0]).Elem().Field(n)
			switch v.Kind() {
			case reflect.String:
				v.SetString(v.String() + "x")
			case reflect.Int64:
				v.SetInt(v.Int() + 1)
			case reflect.Pointer:
				v.SetZero()
			case reflect.Struct:
				v.Set(reflect.ValueOf(s.Runs[0].StartedAt.Add(time.Microsecond)))
			}
			changed, e := CanonicalAdmissionSnapshot(copy)
			if e == nil && bytes.Equal(changed, b) {
				t.Fatal("field omitted from digest")
			}
		})
	}
	t.Run("NullVersusEmpty", func(t *testing.T) {
		s.Runs[0].ArticleEvidenceRef = fixtureText("")
		if _, e := CanonicalAdmissionSnapshot(s); e == nil {
			t.Fatal("empty accepted as SQL NULL")
		}
	})
}

func testGenesisHistory(t *testing.T) {
	t.Run("ExplicitGenesisAndObservationOnly", func(t *testing.T) {
		f := newAdmissionFixture(false)
		r := fixtureRuntime(t, f)
		obs, e := r.Verify(context.Background())
		if e != nil || obs.State != "internal_ready" {
			t.Fatalf("observation: %+v %v", obs, e)
		}
		if l, e := r.Acquire(context.Background(), WorkRequest); e == nil {
			l.Release()
			t.Fatal("Verify opened admission")
		}
		if e := r.RunNormal(context.Background()); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("DeleteThenRestart", func(t *testing.T) {
		f := newAdmissionFixture(false)
		h := JournalHead{Sequence: 1, Hash: fixtureHash("ordinary")}
		f.article.Head = h
		f.article.Prefixes = append(f.article.Prefixes, h)
		if e := fixtureRuntime(t, f).RunNormal(context.Background()); e != nil {
			t.Fatal(e)
		}
	})
	cases := []struct {
		name   string
		change func(*admissionFixture)
	}{
		{"WrongCommitment", func(f *admissionFixture) { f.genesis.CommitmentDigest = fixtureHash("wrong") }},
		{"WrongInstallation", func(f *admissionFixture) { f.genesis.Binding.InstallationID = "other" }},
		{"BeginDisablesGenesis", func(f *admissionFixture) { f.recovery.Entries = []JournalEntry{{RunID: "external", Stage: StageBegun}} }},
		{"Pending", func(f *admissionFixture) { f.article.Pending = &AnchorPending{} }},
		{"MissingPrefix", func(f *admissionFixture) { f.article.Prefixes = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newAdmissionFixture(false)
			tc.change(f)
			if e := fixtureRuntime(t, f).RunNormal(context.Background()); e == nil {
				t.Fatal("invalid genesis admitted")
			}
		})
	}
	t.Run("ExactHistoricalClosure", testClosureMutations)
	t.Run("BeginOrderNotDatabaseTime", testBeginOrder)
}

func fixtureSecondHistory(f *admissionFixture) AdmissionRun {
	a := f.snapshot.Runs[0]
	b := a
	b.RunID = "run-b"
	b.BackupRef = "backup-b"
	b.StartedAt = a.StartedAt.Add(-time.Hour)
	end := b.StartedAt.Add(time.Minute)
	b.JournalConsumedAt = &end
	b.ArticleEvidenceCompletedAt = &end
	b.ArticleReplayVerifiedAt = &end
	b.CompletedAt = &end
	b.InitialAdminChangedAt = &end
	b.ArticleEvidenceRef = fixtureText("evidence-b")
	b.ArticlePublicationRef = fixtureText("publication-b")
	b.ArticlePublicationDigest = fixtureText(fixtureHash("publication-b"))
	receipt := *a.RestoreBinding
	receipt.RunID = b.RunID
	receipt.BackupRef = b.BackupRef
	receipt.RestoredAt = b.StartedAt
	b.RestoreBinding = &receipt
	c := f.closure
	c.Ready.RunID = b.RunID
	c.Ready.BackupRef = b.BackupRef
	c.Ready.EvidenceRef = *b.ArticleEvidenceRef
	c.Ready.CompletedAt = end.Format(time.RFC3339Nano)
	c.Ready.EvidenceDigest = oracleReadyDigest(c.Ready)
	b.ArticleEvidenceDigest = fixtureText(c.Ready.EvidenceDigest)
	c.PublicationRef = *b.ArticlePublicationRef
	c.PublicationDigest = *b.ArticlePublicationDigest
	c.PublishedAt = end
	f.closures = map[string]PublishedClosure{b.RunID: c}
	f.snapshot.Runs = []AdmissionRun{b, a}
	begin := f.recovery.Entries[0]
	begin.Sequence = 3
	begin.RunID = b.RunID
	begin.BackupRef = b.BackupRef
	begin.RecordedAt = b.StartedAt
	begin.PreviousRecordHash = f.recovery.Head.Hash
	begin = fixtureSignEntry(begin)
	consumed := begin
	consumed.Sequence = 4
	consumed.Stage = StageConsumed
	consumed.RecordedAt = end
	consumed.PreviousRecordHash = fixtureHash("begin-b")
	consumed = fixtureSignEntry(consumed)
	f.recovery.Entries = append(f.recovery.Entries, begin, consumed)
	f.recovery.Head = JournalHead{Sequence: 4, Hash: fixtureHash("consumed-b")}
	f.recovery.Prefixes = append(f.recovery.Prefixes, JournalHead{Sequence: 3, Hash: consumed.PreviousRecordHash}, f.recovery.Head)
	return b
}

func testBeginOrder(t *testing.T) {
	f := newAdmissionFixture(true)
	b := fixtureSecondHistory(f)
	r := fixtureRuntime(t, f)
	if e := r.RunNormal(context.Background()); e != nil {
		t.Fatal(e)
	}
	l, e := r.Acquire(context.Background(), WorkRequest)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Release()
	binding, e := r.ReadPermitBinding(l.Context())
	if e != nil || binding.RunID != b.RunID {
		t.Fatalf("did not use verified Begin order: %+v %v", binding, e)
	}
}

func testClosureMutations(t *testing.T) {
	cases := []struct {
		name   string
		change func(*admissionFixture)
	}{
		{"Ref", func(f *admissionFixture) { f.closure.PublicationRef = "other" }},
		{"Digest", func(f *admissionFixture) { f.closure.Ready.EvidenceDigest = fixtureHash("other") }},
		{"Time", func(f *admissionFixture) { f.closure.Ready.CompletedAt = "2026-10-21T00:00:00Z" }},
		{"Revision", func(f *admissionFixture) { f.closure.VerifiedArticleRevision = "other" }},
		{"Prefix", func(f *admissionFixture) { f.closure.HistoricalHead.Hash = fixtureHash("other") }},
		{"RestoreBinding", func(f *admissionFixture) { f.snapshot.Runs[0].RestoreBinding = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newAdmissionFixture(true)
			tc.change(f)
			if e := fixtureRuntime(t, f).RunNormal(context.Background()); e == nil {
				t.Fatal("invalid history admitted")
			}
		})
	}
}

func testCompleteHistory(t *testing.T) {
	f := newAdmissionFixture(true)
	r := fixtureRuntime(t, f)
	if err := r.RunNormal(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func testCountEncodingProperty(t *testing.T) {
	err := quick.Check(func(n uint32) bool {
		f := newAdmissionFixture(true)
		f.snapshot.Runs[0].AccountCount = int64(n)
		b, e := CanonicalAdmissionSnapshot(f.snapshot)
		if e != nil || !bytes.Equal(b, oracleSnapshot(f.snapshot)) {
			return false
		}
		f.snapshot.Runs[0].AccountCount++
		c, e := CanonicalAdmissionSnapshot(f.snapshot)
		return e == nil && !bytes.Equal(b, c)
	}, &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(7))})
	if err != nil {
		t.Fatal(err)
	}
}

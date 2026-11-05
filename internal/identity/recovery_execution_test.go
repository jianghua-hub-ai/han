package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/idempotency"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var s410Time = time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
var s410Hash = "sha256:" + strings.Repeat("a", 64)
var s410HMAC = "hmac-sha256:" + strings.Repeat("b", 64)
var s410Stop = errors.New("synthetic callback completed without ordinary activation")
var s410Binding = recovery.RunBinding{RunID: "s410-run", BackupRef: "s410-backup"}

func TestRecoveryIdentityExecution(t *testing.T) {
	t.Run("FinalizeOnceAndAtomicAudit", func(t *testing.T) {
		s410FinalizeUnit(t)
		t.Run("OptionalDeletionInstantProof", s410DeletionInstantProof)
	})
	t.Run("KeptDeletedCanonicalAndDenied", s410CanonicalUnit)
	t.Run("IdempotencyUnknownAndCompleteProof", s410FailedUnit)
	t.Run("UserStoreReviewLockOrder", func(t *testing.T) {
		t.Run("TypedConstructionAndR56Metadata", s410ConstructionSurface)
		t.Run("MissingDependenciesNoIO", func(t *testing.T) {
			if _, err := newRecoveryIdentityExecution(recoveryExecutionDependencies{}); err == nil {
				t.Fatal("missing dependencies accepted")
			}
		})
		t.Run("PublicIdentityProducerAndExactBinding", func(t *testing.T) {
			s, j := s410MemoryInput()
			f := &recoveryExecutionForwarder{}
			rt, ctx, done := s410Runtime(t, s, s, j, f)
			defer done()
			e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt}}
			called := 0
			target := &s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
				called++
				pc, x, err := e.phase(c, p, b, recovery.PhaseRecoveryIdentity)
				if err != nil || pc == nil || x.RunID != b.RunID || x.BackupRef != b.BackupRef || x.InstallationID != s.snapshot.Installation.InstallationID || x.RuntimeID == "" || x.Generation <= 0 {
					t.Fatalf("public phase: %v", err)
				}
				for _, bad := range []struct {
					name, purpose string
					b             recovery.RunBinding
				}{{"wrongpurpose", recovery.PhaseRecoveryReplay, b}, {"wrongrun", recovery.PhaseRecoveryIdentity, recovery.RunBinding{RunID: "other", BackupRef: b.BackupRef}}, {"wrongbackup", recovery.PhaseRecoveryIdentity, recovery.RunBinding{RunID: b.RunID, BackupRef: "other"}}} {
					t.Run(bad.name, func(t *testing.T) {
						if _, _, err := e.phase(c, p, bad.b, bad.purpose); err == nil {
							t.Fatal("invalid binding/purpose accepted")
						}
					})
				}
				if _, _, err := e.phase(c, recovery.PhasePermit{}, b, recovery.PhaseRecoveryIdentity); err == nil {
					t.Fatal("zero permit accepted")
				}
				return s410Stop
			}}
			if err := f.seal(target); err != nil {
				t.Fatal(err)
			}
			if err := rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) {
				t.Fatalf("public Resume result: %v", err)
			}
			if called != 1 {
				t.Fatalf("producer calls=%d", called)
			}
		})
		t.Run("ForwarderSealsOnce", func(t *testing.T) {
			f := &recoveryExecutionForwarder{}
			if _, err := f.get(); err == nil {
				t.Fatal("unsealed usable")
			}
			if err := f.seal(&s410Callbacks{}); err != nil {
				t.Fatal(err)
			}
			if err := f.seal(&s410Callbacks{}); err == nil {
				t.Fatal("delegate replaced")
			}
		})
		t.Run("RowsErrorClosesWithoutPartialProof", func(t *testing.T) {
			marker := errors.New("synthetic trailing Rows error")
			rows := &s410ErrorRows{marker: marker}
			_, err := recoveryExecutionReviews(context.Background(), &s410ReadTx{rows: rows}, s410Binding.RunID)
			if !errors.Is(err, marker) || rows.closes != 1 {
				t.Fatalf("Rows error/Close: err=%v closes=%d", err, rows.closes)
			}
		})
	})
	t.Run("SucceededRecoverySessionExitAtomic", func(t *testing.T) {
		t.Run("CompleteReceiptStructure", s410ExitReceiptUnit)
		t.Run("IdentityCannotAuthorizeSessionExit", func(t *testing.T) {
			s, j := s410MemoryInput()
			f := &recoveryExecutionForwarder{}
			rt, ctx, done := s410Runtime(t, s, s, j, f)
			defer done()
			e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt}}
			calls := 0
			_ = f.seal(&s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
				calls++
				if err := e.RevokeRecoverySessions(c, p, b); err == nil {
					t.Fatal("identity grant authorized exit")
				}
				return s410Stop
			}})
			if err := rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) || calls != 1 {
				t.Fatalf("producer/callback: %v calls=%d", err, calls)
			}
		})
		t.Run("StopCancelsAndJoinsCallback", func(t *testing.T) {
			s, j := s410MemoryInput()
			f := &recoveryExecutionForwarder{}
			rt, ctx, done := s410Runtime(t, s, s, j, f)
			defer done()
			entered, release, canceled := make(chan struct{}), make(chan struct{}), make(chan struct{})
			_ = f.seal(&s410Callbacks{finalize: func(c context.Context, _ recovery.PhasePermit, _ recovery.RunBinding) error {
				close(entered)
				<-c.Done()
				close(canceled)
				<-release
				return c.Err()
			}})
			resume := make(chan error, 1)
			go func() { resume <- rt.Resume(ctx, s410Binding) }()
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("producer never entered callback")
			}
			stopped := make(chan error, 1)
			go func() { stopped <- rt.Stop(ctx) }()
			select {
			case <-canceled:
			case <-ctx.Done():
				close(release)
				<-resume
				<-stopped
				t.Fatal("Stop did not cancel")
			}
			select {
			case <-stopped:
				close(release)
				<-resume
				t.Fatal("Stop returned before callback joined")
			default:
			}
			close(release)
			if err := <-resume; !errors.Is(err, context.Canceled) {
				t.Fatalf("resume cancellation cause: %v", err)
			}
			<-stopped
		})
	})
	t.Run("RecoveryExitAuditAckReadback", func(t *testing.T) {
		t.Run("CallerBindingIsNotCapability", func(t *testing.T) {
			e := &recoveryIdentityExecution{}
			if _, _, err := e.phase(nil, recovery.PhasePermit{}, s410Binding, recovery.PhaseRecoveryReplay); err == nil {
				t.Fatal("nil context accepted")
			}
			if _, _, err := e.phase(context.Background(), recovery.PhasePermit{}, s410Binding, recovery.PhaseRecoveryReplay); err == nil {
				t.Fatal("unbounded context accepted")
			}
		})
		t.Run("EscapedPermitInvalidAfterProducerReturns", func(t *testing.T) {
			s, j := s410MemoryInput()
			f := &recoveryExecutionForwarder{}
			rt, ctx, done := s410Runtime(t, s, s, j, f)
			defer done()
			var escaped recovery.PhasePermit
			_ = f.seal(&s410Callbacks{finalize: func(_ context.Context, p recovery.PhasePermit, _ recovery.RunBinding) error {
				escaped = p
				return s410Stop
			}})
			_ = rt.Resume(ctx, s410Binding)
			if err := rt.CheckPhase(ctx, escaped, recovery.PhaseRecoveryIdentity); err == nil {
				t.Fatal("escaped grant survived closed producer")
			}
		})
	})
}

type s410ReadTx struct {
	s410CohortTx
	rows *s410ErrorRows
}

func (x s410ReadTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return x.rows, nil }

type s410ErrorRows struct {
	s410CohortRows
	n, closes int
	marker    error
}

func (r *s410ErrorRows) Next() bool {
	r.n++
	if r.n == 1 {
		return true
	}
	r.Close()
	return false
}
func (r *s410ErrorRows) Scan(dest ...any) error {
	*(dest[0].(*[]byte)) = []byte(`{"user_id":"22222222-2222-4222-8222-222222222222","backup_role":"member","backup_status":"active","decision":"pending"}`)
	return nil
}
func (r *s410ErrorRows) Err() error { return r.marker }
func (r *s410ErrorRows) Close() {
	if !r.closed {
		r.closes++
		r.s410CohortRows.Close()
	}
}

type s410Callbacks struct {
	bind     func(context.Context, recovery.PhasePermit, recovery.RunBinding, recovery.RestoreReceipt) (recovery.RecoveryRun, error)
	finalize func(context.Context, recovery.PhasePermit, recovery.RunBinding) error
	trust    func(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte) error
	complete func(context.Context, recovery.PhasePermit, recovery.RunBinding) error
	exit     func(context.Context, recovery.PhasePermit, recovery.RunBinding) error
}

func (c *s410Callbacks) BindRestoredRun(x context.Context, p recovery.PhasePermit, b recovery.RunBinding, r recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	if c.bind == nil {
		return recovery.RecoveryRun{}, errors.New("unexpected fixture bind")
	}
	return c.bind(x, p, b, r)
}
func (c *s410Callbacks) FinalizeRestoredIdentity(x context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	if c.finalize == nil {
		return errors.New("unexpected fixture finalization")
	}
	return c.finalize(x, p, b)
}
func (c *s410Callbacks) PrepareRecoveryTrust(x context.Context, p recovery.PhasePermit, b recovery.RunBinding, s []byte) error {
	if c.trust == nil {
		return errors.New("unexpected fixture trust")
	}
	return c.trust(x, p, b, s)
}
func (c *s410Callbacks) CompleteIdentityReview(x context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	if c.complete == nil {
		return errors.New("unexpected fixture Complete")
	}
	return c.complete(x, p, b)
}
func (c *s410Callbacks) RevokeRecoverySessions(x context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	if c.exit == nil {
		return errors.New("unexpected fixture Session exit")
	}
	return c.exit(x, p, b)
}

type s410Memory struct {
	snapshot recovery.AdmissionSnapshot
	run      recovery.RecoveryRun
}

func (s *s410Memory) ReadAdmissionSnapshot(context.Context) (recovery.AdmissionSnapshot, error) {
	x := s.snapshot
	raw, err := recovery.CanonicalAdmissionSnapshot(x)
	if err != nil {
		return x, err
	}
	x.Revision.Digest = recovery.SHA256Digest(raw)
	return x, nil
}
func (s *s410Memory) ReadAdmissionRevision(c context.Context) (recovery.StoreRevision, error) {
	x, e := s.ReadAdmissionSnapshot(c)
	return x.Revision, e
}
func (s *s410Memory) Load(context.Context, string) (recovery.RecoveryRun, error) { return s.run, nil }
func (s *s410Memory) Transition(context.Context, string, recovery.RecoveryOutcome, recovery.RecoveryOutcome, recovery.RecoveryRunChange) (recovery.RecoveryRun, error) {
	return recovery.RecoveryRun{}, errors.New("unexpected memory Store mutation")
}

func s410Installation() recovery.InstallationBinding {
	return recovery.InstallationBinding{Version: recovery.InstallationVersion, DeploymentID: "s410-deploy", InstallationID: "s410-install", TargetDigest: s410Hash, CommitmentRef: "s410-commitment", CommitmentDigest: s410Hash, RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s410-recovery", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1}, ArticleInitial: recovery.AnchorSnapshot{JournalID: "s410-article", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 2, Committed: recovery.JournalHead{Sequence: 1, Hash: s410Hash}}, InstalledAt: s410Time.Add(-time.Hour)}
}
func s410Receipt() recovery.RestoreReceipt {
	return recovery.RestoreReceipt{Version: "han-db-restore/v1", RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, DeploymentID: "s410-deploy", InstallationID: "s410-install", TargetDigest: s410Hash, ArtifactDigest: s410Hash, SourceApprovalDigest: s410Hash, RestoredSchemaDigest: s410Hash, ReceiptDigest: s410Hash, ArtifactBytes: 1, RestoredAt: s410Time}
}
func s410MemoryInput() (*s410Memory, *s410Journal) {
	x := s410Receipt()
	j := &s410Journal{}
	j.append(recovery.StageBegun)
	a := recovery.AdmissionRun{RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, JournalKeyID: "s410-key", JournalKeyVersion: "K1", JournalSecretFingerprint: s410HMAC, JournalStage: recovery.StageBegun, StartedAt: s410Time, Outcome: recovery.RecoveryOutcomeRunning, RestoreBinding: &x}
	s := &s410Memory{snapshot: recovery.AdmissionSnapshot{Installation: s410Installation(), Revision: recovery.StoreRevision{InstallationID: "s410-install", Counter: 1}, Runs: []recovery.AdmissionRun{a}}, run: recovery.RecoveryRun{RunID: a.RunID, BackupRef: a.BackupRef, JournalKeyID: a.JournalKeyID, JournalKeyVersion: a.JournalKeyVersion, JournalSecretFingerprint: a.JournalSecretFingerprint, JournalStage: a.JournalStage, StartedAt: a.StartedAt, Outcome: a.Outcome}}
	return s, j
}

type s410Journal struct {
	mu       sync.Mutex
	entries  []recovery.JournalEntry
	consumes int
}

func (j *s410Journal) append(stage recovery.Stage) {
	n := len(j.entries)
	old := ""
	if n > 0 {
		old = recovery.SHA256Digest([]byte(fmt.Sprint(n)))
	}
	recorded := s410Time.Add(time.Duration(n) * time.Second)
	if stage == recovery.StageConsumed {
		recorded = s410Time.Add(120 * time.Second)
	}
	j.entries = append(j.entries, recovery.JournalEntry{Namespace: recovery.RecoveryNamespaceV1, Sequence: uint64(n + 1), RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, KeyID: "s410-key", KeyVersion: "K1", SecretFingerprint: s410HMAC, Stage: stage, RecordedAt: recorded, PreviousRecordHash: old, RecordHMAC: s410HMAC})
}
func (j *s410Journal) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return recovery.VerifiedJournalSnapshot{}, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if ns == recovery.ArticleDeletionNamespaceV1 {
		return recovery.VerifiedJournalSnapshot{JournalID: "s410-article", Namespace: ns, Epoch: 2, Head: recovery.JournalHead{Sequence: 1, Hash: s410Hash}, Revision: "article-r1", Prefixes: []recovery.JournalHead{{Sequence: 1, Hash: s410Hash}}}, nil
	}
	s := recovery.VerifiedJournalSnapshot{JournalID: "s410-recovery", Namespace: ns, Epoch: 1, Revision: fmt.Sprint("recovery-r", len(j.entries)), Entries: append([]recovery.JournalEntry{}, j.entries...), Prefixes: []recovery.JournalHead{{}}}
	for i := range j.entries {
		s.Head = recovery.JournalHead{Sequence: uint64(i + 1), Hash: recovery.SHA256Digest([]byte(fmt.Sprint(i + 1)))}
		s.Prefixes = append(s.Prefixes, s.Head)
	}
	return s, nil
}
func (j *s410Journal) Begin(c context.Context, run, backup string, _ []byte) (recovery.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if run != s410Binding.RunID || backup != s410Binding.BackupRef || len(j.entries) != 0 {
		return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
	}
	j.append(recovery.StageBegun)
	return j.entries[0], c.Err()
}
func (j *s410Journal) VerifyBeginSecret(c context.Context, b recovery.RunBinding, s []byte) (recovery.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if b != s410Binding || string(s) != "synthetic-secret" || len(j.entries) == 0 {
		return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
	}
	return j.entries[0], c.Err()
}
func (j *s410Journal) MarkConsumed(c context.Context, run string) (recovery.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if run != s410Binding.RunID {
		return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
	}
	j.append(recovery.StageConsumed)
	j.consumes++
	return j.entries[len(j.entries)-1], c.Err()
}
func (j *s410Journal) Read(c context.Context, run string) (recovery.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if len(j.entries) == 0 || run != s410Binding.RunID {
		return recovery.JournalEntry{}, recovery.ErrRunNotFound
	}
	return j.entries[len(j.entries)-1], c.Err()
}
func (j *s410Journal) VerifyAgainst(c context.Context, r recovery.RecoveryRun) (recovery.JournalEntry, error) {
	x, err := j.Read(c, r.RunID)
	if err != nil {
		return x, err
	}
	if x.BackupRef != r.BackupRef || x.KeyID != r.JournalKeyID || x.KeyVersion != r.JournalKeyVersion || x.SecretFingerprint != r.JournalSecretFingerprint {
		return x, recovery.ErrAdmissionBinding
	}
	return x, nil
}

type s410Restore struct{}

func (s410Restore) Prepare(c context.Context, b recovery.RunBinding) (recovery.RestorePlan, error) {
	x := s410Receipt()
	return recovery.RestorePlan{Version: x.Version, RunID: b.RunID, BackupRef: b.BackupRef, DeploymentID: x.DeploymentID, InstallationID: x.InstallationID, TargetDigest: x.TargetDigest, ArtifactDigest: x.ArtifactDigest, SourceApprovalDigest: x.SourceApprovalDigest, SourceSchemaDigest: x.RestoredSchemaDigest, ArtifactBytes: 1, ArtifactPath: "synthetic.fixture", SourceApprovalRef: "synthetic-approved", SourceInstallationID: "synthetic-source"}, c.Err()
}
func (s410Restore) Replace(context.Context, recovery.PhasePermit, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return s410Receipt(), nil
}
func (s410Restore) VerifyRestored(c context.Context, _ recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return s410Receipt(), c.Err()
}

type s410Genesis struct{}

func (s410Genesis) VerifyGenesis(c context.Context, i recovery.InstallationBinding) (recovery.GenesisProof, error) {
	return recovery.GenesisProof{Binding: i, RecoveryInitial: i.RecoveryInitial, ArticleInitial: i.ArticleInitial, CommitmentRevision: "synthetic-genesis-r1", CommitmentDigest: i.CommitmentDigest}, c.Err()
}

type s410Exclusive struct {
	mu   sync.Mutex
	held bool
}

func (x *s410Exclusive) Acquire(c context.Context) (recovery.ExclusiveLease, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.held {
		return nil, errors.New("synthetic deployment busy")
	}
	x.held = true
	return &s410Lease{x: x}, c.Err()
}

type s410Lease struct {
	x      *s410Exclusive
	closed bool
}

func (l *s410Lease) Check(c context.Context) error {
	l.x.mu.Lock()
	defer l.x.mu.Unlock()
	if l.closed || !l.x.held {
		return recovery.ErrAdmissionClosed
	}
	return c.Err()
}
func (l *s410Lease) Close() error {
	l.x.mu.Lock()
	defer l.x.mu.Unlock()
	if l.closed {
		return errors.New("double synthetic lease Close")
	}
	l.closed = true
	l.x.held = false
	return nil
}

type s410Lifecycle struct{}

func (s410Lifecycle) StopAndJoin(c context.Context) error { return c.Err() }
func (s410Lifecycle) ReconcileAndStart(context.Context, recovery.PhasePermit) error {
	return errors.New("fixture must not open ordinary work")
}

type s410Secret struct{}

func (s410Secret) ReadInitialRecoverySecret(c context.Context, _ recovery.RunBinding) ([]byte, error) {
	return []byte("synthetic-secret"), c.Err()
}

type s410Consumer struct{}

func (s410Consumer) Consume(context.Context, string) (recovery.ConsumptionResult, error) {
	return recovery.ConsumptionResult{}, errors.New("unexpected fixture Consume")
}

type s410Exit struct {
	fn func(context.Context, recovery.PhasePermit, recovery.RunBinding) error
}

func (x *s410Exit) ExitRecovery(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
	if x.fn == nil {
		return errors.New("unexpected fixture exit")
	}
	return x.fn(c, p, b)
}

type s410Closures struct{ ready recovery.ReadyReplayStatusV1 }

func (x *s410Closures) ReadPublishedClosure(c context.Context, _ recovery.ClosureRequest) (recovery.PublishedClosure, error) {
	if x.ready.RunID == "" {
		return recovery.PublishedClosure{}, errors.New("unexpected fixture closure")
	}
	return recovery.PublishedClosure{Ready: x.ready, PublicationRef: "s410-publication", PublicationDigest: s410Hash, JournalID: "s410-article", Epoch: 2, PublishedAt: s410Time.Add(91 * time.Second), VerifiedArticleRevision: "article-r1", CurrentHead: recovery.JournalHead{Sequence: 1, Hash: s410Hash}, HistoricalHead: recovery.JournalHead{Sequence: 1, Hash: s410Hash}}, c.Err()
}

func s410Runtime(t *testing.T, s recovery.AdmissionStore, runs recovery.RecoveryRunStore, j *s410Journal, f recovery.IdentityRecoveryExecutor) (recovery.Runtime, context.Context, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx, err := platform.ContextWithTraceID(ctx, "00000000000000000000000001")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	rt, err := recovery.NewRuntime(ctx, recovery.RuntimeDependencies{Store: s, Runs: runs, Journal: j, ArticleSnapshots: j, Genesis: s410Genesis{}, Exclusive: &s410Exclusive{}, Lifecycle: s410Lifecycle{}, Closures: &s410Closures{}, Identity: f, Restore: s410Restore{}, Consumer: s410Consumer{}, Exit: &s410Exit{}, RecoverySecret: s410Secret{}})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	return rt, ctx, func() { _ = rt.Stop(ctx); cancel() }
}

// All approval material below is synthetic immutable checkpoint data, never a
// production provider/ACL claim. It is fixed BEFORE ReadFailed and cannot approve
// arbitrary caller lookup values. Source constructors own every nested byte.
type s410Checkpoint struct {
	mu    sync.Mutex
	data  RecoveryFailedEvidence
	calls int
}

func s410CheckpointReader(x RecoveryFailedEvidence) *s410Checkpoint {
	x = recoveryCopyFailed(x)
	x.EvidenceDigest = ""
	raw, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(raw)
	x.EvidenceDigest = hex.EncodeToString(digest[:])
	return &s410Checkpoint{data: x}
}
func (s *s410Checkpoint) ReadFailed(c context.Context, b recovery.PermitBinding, ref string, h, l []byte, g int64) (RecoveryFailedEvidence, error) {
	if c == nil {
		return RecoveryFailedEvidence{}, recovery.ErrAdmissionBinding
	}
	if _, ok := c.Deadline(); !ok {
		return RecoveryFailedEvidence{}, recovery.ErrAdmissionBinding
	}
	if err := c.Err(); err != nil {
		return RecoveryFailedEvidence{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	x := s.data
	r := x.Recovery
	if b.RuntimeID == "" || b.Generation <= 0 || b.Purpose != recovery.PhaseRecoveryIdentity || b.InstallationID != x.InstallationID || b.RunID != x.RunID || b.BackupRef != x.BackupRef || ref != x.EvidenceRef || !bytes.Equal(h, r.ScopeHash) || !bytes.Equal(l, r.KeyLocator) || g != r.ExecutionGeneration {
		return RecoveryFailedEvidence{}, recovery.ErrAdmissionBinding
	}
	return recoveryCopyFailed(x), nil
}

type s410ReadFailedFunc func(context.Context, recovery.PermitBinding, string, []byte, []byte, int64) (RecoveryFailedEvidence, error)

func (f s410ReadFailedFunc) ReadFailed(c context.Context, b recovery.PermitBinding, r string, s, k []byte, g int64) (RecoveryFailedEvidence, error) {
	return f(c, b, r, s, k, g)
}
func s410SyntheticFailure(t *testing.T, kind string) (idempotency.BeginRequest, recoveryIdempotencyMember, RecoveryFailedEvidence) {
	t.Helper()
	actor := "11111111-1111-4111-8111-111111111111"
	target := "22222222-2222-4222-8222-222222222222"
	scope := []byte(actor + "\x00" + administratorCreateOperation + "\x00original-name")
	code := platform.CodeIdentityIdentifierConflict
	if kind == "administrator503" {
		code = platform.CodeIdentityEmailDeliveryFailed
	}
	snapshot, err := administratorFailureSnapshot(code)
	if err != nil {
		t.Fatal(err)
	}
	if kind == "resend" {
		scope = []byte(actor + "\x00" + temporaryCredentialResendOperation + "\x00" + target + "\x00new_administrator")
		snapshot = temporaryCredentialResendFailureSnapshot()
		code = platform.CodeIdentityEmailDeliveryFailed
	}
	if kind == "reset" {
		scope = []byte(actor + "\x00" + passwordResetOperation + "\x00" + target)
		snapshot = idempotency.ResponseSnapshot{Outcome: "invalidated", ResourceID: "44444444-4444-4444-8444-444444444444"}
		code = platform.Code("password_reset_invalidated")
	}
	original := []byte(`{"username":"original-name","email":"original@example.invalid","phone":"+123synthetic","temporary_password":"synthetic-only-password","information_provision_notice_version":"fixture-v1","information_provision_declared":true}`)
	if kind == "resend" {
		original = []byte(`{"target_id":"22222222-2222-4222-8222-222222222222","purpose":"new_administrator"}`)
	}
	if kind == "reset" {
		original = []byte(`{"target_id":"22222222-2222-4222-8222-222222222222","target_role":"member"}`)
	}
	request := idempotency.BeginRequest{Scope: scope, Key: "synthetic-original-key-" + kind, Request: original, DigestKeys: idempotency.DigestKeyring{ActiveVersion: "retained-fixture-K1", Keys: map[string][]byte{"retained-fixture-K1": []byte("explicit-synthetic-fixture-HMAC-key")}}, LockedUntil: s410Time.Add(time.Hour), ExpiresAt: s410Time.Add(24 * time.Hour), Now: s410Time}
	claim, err := idempotency.NewPasswordResetClaim(request, 1)
	if err != nil {
		t.Fatal(err)
	}
	member := recoveryMember(idempotency.Transition{ScopeHash: claim.ScopeHash, KeyLocator: claim.KeyLocator, KeyDigest: claim.KeyDigest, RequestDigest: claim.RequestDigest, DigestKeyVersion: claim.DigestKeyVersion, ExecutionGeneration: claim.ExecutionGeneration}, claim.CreatedAt, claim.ExpiresAt)
	src := RecoveryFailedEvidence{Version: recoveryFailedEvidenceVersion, EvidenceRef: "immutable-original-" + kind, InstallationID: "s410-install", RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, ActorID: actor, Scope: scope, ApprovedAt: s410Time.Add(20 * time.Second), Recovery: idempotency.Recovery{ScopeHash: claim.ScopeHash, KeyLocator: claim.KeyLocator, KeyDigest: claim.KeyDigest, RequestDigest: claim.RequestDigest, DigestKeyVersion: claim.DigestKeyVersion, ExecutionGeneration: claim.ExecutionGeneration, Action: idempotency.RecoveryFail, ResultCode: string(code), ResponseSnapshot: snapshot}}
	checkpoint := s410CheckpointReader(src)
	return request, member, recoveryCopyFailed(checkpoint.data)
}
func s410FailedUnit(t *testing.T) {
	t.Run("RealScopeAlgorithmAndDomainFailureProducers", func(t *testing.T) {
		for _, kind := range []string{"administrator409", "administrator503", "resend", "reset"} {
			t.Run(kind, func(t *testing.T) {
				_, m, x := s410SyntheticFailure(t, kind)
				v := recovery.PermitBinding{InstallationID: x.InstallationID}
				if err := recoveryFailedValid(x, m, s410Binding, v, x.ActorID, x.EvidenceRef, s410Time.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
				wire, err := recoveryFailureWire(x)
				if err != nil || !json.Valid(wire) {
					t.Fatalf("real producer not accepted: %v", err)
				}
				record := idempotency.Record{Status: idempotency.StatusFailed, ResultCode: x.Recovery.ResultCode, ResponseSnapshot: &x.Recovery.ResponseSnapshot, CreatedAt: m.CreatedAt}
				var replay error
				if strings.HasPrefix(kind, "administrator") {
					_, replay = (&AdministratorCreateCoordinator{}).replayCreateAdministrator(record)
				} else if kind == "reset" {
					wire, _ := recoveryFailureWire(x)
					raw := string(wire)
					code := x.Recovery.ResultCode
					if !passwordResetIdempotencyTerminal(passwordResetSettlementAuthority{idempotencyStatus: "failed", resultCode: &code, responseSnapshot: &raw}, x.Recovery.ResponseSnapshot.ResourceID, false) {
						t.Fatal("original durable invalidation classifier rejected")
					}
					return
				} else {
					_, replay = replayPasswordReset(record, "22222222-2222-4222-8222-222222222222")
				}
				if platform.PublicCode(replay) != platform.Code(x.Recovery.ResultCode) {
					t.Fatalf("original domain replay: %v", replay)
				}
			})
		}
	})
	t.Run("SourceAndOriginalIdentityRejectMatrix", func(t *testing.T) {
		cases := []struct {
			name   string
			change func(*RecoveryFailedEvidence)
		}{
			{"version", func(x *RecoveryFailedEvidence) { x.Version = "unreviewed" }},
			{"reference", func(x *RecoveryFailedEvidence) { x.EvidenceRef += "-other" }},
			{"digest_prefix", func(x *RecoveryFailedEvidence) { x.EvidenceDigest = "sha256:" + x.EvidenceDigest }},
			{"digest_case", func(x *RecoveryFailedEvidence) { x.EvidenceDigest = strings.ToUpper(x.EvidenceDigest) }},
			{"installation", func(x *RecoveryFailedEvidence) { x.InstallationID = "other" }},
			{"run", func(x *RecoveryFailedEvidence) { x.RunID = "other" }},
			{"backup", func(x *RecoveryFailedEvidence) { x.BackupRef = "other" }},
			{"actor", func(x *RecoveryFailedEvidence) { x.ActorID = "22222222-2222-4222-8222-222222222222" }},
			{"scope", func(x *RecoveryFailedEvidence) { x.Scope[0] = '9' }},
			{"scope_hash", func(x *RecoveryFailedEvidence) { x.Recovery.ScopeHash[0] ^= 1 }},
			{"key_locator", func(x *RecoveryFailedEvidence) { x.Recovery.KeyLocator[0] ^= 1 }},
			{"key_digest", func(x *RecoveryFailedEvidence) { x.Recovery.KeyDigest[0] ^= 1 }},
			{"request_digest", func(x *RecoveryFailedEvidence) { x.Recovery.RequestDigest[0] ^= 1 }},
			{"retained_version", func(x *RecoveryFailedEvidence) { x.Recovery.DigestKeyVersion = "new-active-K2" }},
			{"generation", func(x *RecoveryFailedEvidence) { x.Recovery.ExecutionGeneration++ }},
			{"retry", func(x *RecoveryFailedEvidence) { x.Recovery.Action = idempotency.RecoveryRetry }},
			{"source_owned_actor", func(x *RecoveryFailedEvidence) { x.Recovery.ReviewedBy = x.ActorID }},
			{"source_owned_clock", func(x *RecoveryFailedEvidence) { x.Recovery.Now = s410Time }},
			{"source_ttl", func(x *RecoveryFailedEvidence) { x.Recovery.ExpiresAt = s410Time.Add(48 * time.Hour) }},
			{"source_lease", func(x *RecoveryFailedEvidence) { x.Recovery.LockedUntil = s410Time }},
			{"approval_future", func(x *RecoveryFailedEvidence) { x.ApprovedAt = s410Time.Add(time.Hour) }},
			{"approval_submicrosecond", func(x *RecoveryFailedEvidence) { x.ApprovedAt = x.ApprovedAt.Add(time.Nanosecond) }},
			{"generic_failure", func(x *RecoveryFailedEvidence) {
				x.Recovery.ResponseSnapshot = idempotency.ResponseSnapshot{Outcome: "failed"}
			}},
			{"success_code", func(x *RecoveryFailedEvidence) { x.Recovery.ResultCode = "created" }},
			{"null_field_errors", func(x *RecoveryFailedEvidence) { x.Recovery.ResponseSnapshot.Body.FieldErrors = nil }},
			{"extra_field_errors", func(x *RecoveryFailedEvidence) {
				x.Recovery.ResponseSnapshot.Body.FieldErrors = []idempotency.ResponseFieldError{{Field: "secret", Code: "leak"}}
			}},
			{"extra_success_field", func(x *RecoveryFailedEvidence) { x.Recovery.ResponseSnapshot.Body.Username = "must-not-replay" }},
			{"wrong_message", func(x *RecoveryFailedEvidence) { x.Recovery.ResponseSnapshot.Body.Message = "generic failed" }},
			{"mixed_legacy", func(x *RecoveryFailedEvidence) { x.Recovery.ResponseSnapshot.Outcome = "failed" }},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, m, x := s410SyntheticFailure(t, "administrator409")
				ref, actor := x.EvidenceRef, x.ActorID
				tc.change(&x)
				if err := recoveryFailedValid(x, m, s410Binding, recovery.PermitBinding{InstallationID: "s410-install"}, actor, ref, s410Time.Add(time.Minute)); err == nil {
					t.Fatal("invalid source/identity/snapshot accepted")
				}
			})
		}
	})
	t.Run("ImmutableCheckpointAndOwnedNestedCopies", func(t *testing.T) {
		_, m, x := s410SyntheticFailure(t, "administrator409")
		reader := s410CheckpointReader(x)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		v := recovery.PermitBinding{RuntimeID: "synthetic-lookup-only", Generation: 1, Purpose: recovery.PhaseRecoveryIdentity, InstallationID: x.InstallationID, RunID: x.RunID, BackupRef: x.BackupRef}
		tr := m.transition()
		a, err := reader.ReadFailed(ctx, v, x.EvidenceRef, tr.ScopeHash, tr.KeyLocator, 1)
		if err != nil {
			t.Fatal(err)
		}
		a.Scope[0] = '9'
		a.Recovery.KeyDigest[0] ^= 1
		a.Recovery.ResponseSnapshot.Body.Message = "mutated"
		a.Recovery.ResponseSnapshot.Body.FieldErrors = append(a.Recovery.ResponseSnapshot.Body.FieldErrors, idempotency.ResponseFieldError{Field: "x", Code: "y"})
		b, err := reader.ReadFailed(ctx, v, x.EvidenceRef, tr.ScopeHash, tr.KeyLocator, 1)
		if err != nil || !reflect.DeepEqual(b, reader.data) || b.Recovery.ResponseSnapshot.Body.FieldErrors == nil {
			t.Fatalf("source copy escaped: %v", err)
		}
	})
	t.Run("OriginalCohortCannotBeInferredFromCounts", func(t *testing.T) {
		_, m, _ := s410SyntheticFailure(t, "resend")
		if err := recoveryCohortValid([]recoveryIdempotencyMember{m}, 1); err != nil {
			t.Fatal(err)
		}
		if err := recoveryCohortValid([]recoveryIdempotencyMember{}, 0); err != nil {
			t.Fatal(err)
		}
		for _, bad := range []struct {
			name  string
			items []recoveryIdempotencyMember
			n     int64
		}{{"absent_empty", nil, 0}, {"count_only", nil, 1}, {"missing_member", []recoveryIdempotencyMember{}, 1}, {"duplicate", []recoveryIdempotencyMember{m, m}, 2}} {
			t.Run(bad.name, func(t *testing.T) {
				if err := recoveryCohortValid(bad.items, bad.n); err == nil {
					t.Fatal("incomplete cohort accepted")
				}
			})
		}
	})
	t.Run("PublicProducerReadsSourceAndRejectsMissingProvider", func(t *testing.T) {
		s, j := s410MemoryInput()
		f := &recoveryExecutionForwarder{}
		rt, ctx, done := s410Runtime(t, s, s, j, f)
		defer done()
		_, m, x := s410SyntheticFailure(t, "administrator409")
		reader := s410CheckpointReader(x)
		e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt, failedEvidence: reader}}
		called := 0
		_ = f.seal(&s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
			called++
			pc, v, err := e.phase(c, p, b, recovery.PhaseRecoveryIdentity)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = e.readFailure(pc, p, b, v, m, x.ActorID, x.EvidenceRef, s410Time.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			missing := *e
			missing.d.failedEvidence = nil
			if _, err = missing.readFailure(pc, p, b, v, m, x.ActorID, x.EvidenceRef, s410Time.Add(time.Minute)); !errors.Is(err, recovery.ErrAdmissionIncomplete) {
				t.Fatalf("missing provider: %v", err)
			}
			return s410Stop
		}})
		if err := rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) || called != 1 || reader.calls != 1 {
			t.Fatalf("real public producer: %v callback=%d reader=%d", err, called, reader.calls)
		}
	})
	t.Run("ReaderCancellationTrueJoin", s410ReaderJoin)
}
func s410ReaderJoin(t *testing.T) {
	s, j := s410MemoryInput()
	f := &recoveryExecutionForwarder{}
	rt, ctx, done := s410Runtime(t, s, s, j, f)
	defer done()
	_, m, x := s410SyntheticFailure(t, "administrator409")
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	reader := s410ReadFailedFunc(func(c context.Context, _ recovery.PermitBinding, _ string, _, _ []byte, _ int64) (RecoveryFailedEvidence, error) {
		close(entered)
		<-c.Done()
		close(canceled)
		<-release
		return RecoveryFailedEvidence{}, c.Err()
	})
	e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt, failedEvidence: reader}}
	_ = f.seal(&s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
		pc, v, err := e.phase(c, p, b, recovery.PhaseRecoveryIdentity)
		if err != nil {
			return err
		}
		_, err = e.readFailure(pc, p, b, v, m, x.ActorID, x.EvidenceRef, s410Time.Add(time.Minute))
		return err
	}})
	finished := make(chan error, 1)
	go func() { finished <- rt.Resume(ctx, s410Binding) }()
	select {
	case <-entered:
	case <-ctx.Done():
		close(release)
		<-finished
		t.Fatal("reader did not start")
	}
	stopped := make(chan error, 1)
	go func() { stopped <- rt.Stop(ctx) }()
	select {
	case <-canceled:
	case <-ctx.Done():
		close(release)
		<-finished
		<-stopped
		t.Fatal("reader not canceled")
	}
	select {
	case <-stopped:
		close(release)
		<-finished
		t.Fatal("Stop detached reader")
	default:
	}
	close(release)
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("reader cause lost: %v", err)
	}
	<-stopped
}
func s410ConstructionSurface(t *testing.T) {
	t.Run("TypedCrossPackageDependenciesNoIO", func(t *testing.T) {
		executor, maintenance, err := NewRecoveryIdentityExecution(RecoveryIdentityExecutionDependencies{})
		if err == nil || executor != nil || maintenance != nil {
			t.Fatal("invalid public construction succeeded")
		}
	})
	t.Run("ConstructionOnlyOriginalFiveMethodForwarder", func(t *testing.T) {
		f := NewRecoveryIdentityForwarder()
		var executor recovery.IdentityRecoveryExecutor = f
		if err := executor.FinalizeRestoredIdentity(context.Background(), recovery.PhasePermit{}, s410Binding); err == nil {
			t.Fatal("unsealed forwarder dispatched")
		}
		if err := f.Seal(f); err == nil {
			t.Fatal("self delegation accepted")
		}
		if err := f.Seal(nil); err == nil {
			t.Fatal("nil delegation accepted")
		}
		calls := 0
		target := &s410Callbacks{finalize: func(context.Context, recovery.PhasePermit, recovery.RunBinding) error { calls++; return s410Stop }}
		if err := f.Seal(target); err != nil {
			t.Fatal(err)
		}
		if err := f.Seal(&s410Callbacks{}); err == nil {
			t.Fatal("sealed delegate replaced")
		}
		if err := executor.FinalizeRestoredIdentity(context.Background(), recovery.PhasePermit{}, s410Binding); !errors.Is(err, s410Stop) || calls != 1 {
			t.Fatalf("original interface not forwarded: %v calls=%d", err, calls)
		}
	})
	t.Run("R56DecimalGenerationRetainsMaxInt64", func(t *testing.T) {
		_, m, _ := s410SyntheticFailure(t, "administrator409")
		m.ExecutionGeneration = 9223372036854775807
		raw, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(raw, []byte(`"execution_generation":"9223372036854775807"`)) {
			t.Fatal("new Audit generation is not a decimal string")
		}
		var got recoveryIdempotencyMember
		if err = json.Unmarshal(raw, &got); err != nil || !reflect.DeepEqual(got, m) {
			t.Fatalf("metadata int64 lost precision: %v", err)
		}
		for _, bad := range []string{`"01"`, `"+1"`, `"1.0"`, `"9223372036854775808"`, `1`, `null`} {
			t.Run(bad, func(t *testing.T) {
				altered := bytes.Replace(raw, []byte(`"9223372036854775807"`), []byte(bad), 1)
				if err := json.Unmarshal(altered, &got); err == nil {
					t.Fatal("noncanonical generation accepted")
				}
			})
		}
	})
}
func s410FinalizeUnit(t *testing.T) {
	for _, trailing := range []bool{false, true} {
		name := "FullOriginalSixFieldsAndOwnedCohort"
		if trailing {
			name = "TrailingRowsErrorNoPartialTransition"
		}
		t.Run(name, func(t *testing.T) {
			s, j := s410MemoryInput()
			f := NewRecoveryIdentityForwarder()
			rt, ctx, done := s410Runtime(t, s, s, j, f)
			defer done()
			_, member, _ := s410SyntheticFailure(t, "administrator409")
			tr := member.transition()
			marker := errors.New("synthetic trailing invalidation Rows error")
			rows := &s410CohortRows{values: []any{tr.ScopeHash, tr.KeyLocator, tr.KeyDigest, tr.RequestDigest, tr.DigestKeyVersion, tr.ExecutionGeneration, member.CreatedAt, member.ExpiresAt}}
			if trailing {
				rows.err = marker
			}
			tx := &s410CohortTx{rows: rows}
			e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt, clock: s410UnitClock{}}}
			calls := 0
			if err := f.Seal(&s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
				calls++
				cohort, err := e.invalidateIdempotency(c, tx, p)
				if trailing {
					if !errors.Is(err, marker) || tx.writes != 0 || cohort != nil {
						t.Fatalf("partial cohort or write escaped trailing Rows failure: %v writes=%d", err, tx.writes)
					}
				} else {
					if err != nil || len(cohort) != 1 || !reflect.DeepEqual(cohort[0], member) || tx.writes != 1 {
						t.Fatalf("real invalidation did not preserve original six fields: %v writes=%d", err, tx.writes)
					}
					if len(tx.args) != 8 || tx.args[0] != idempotency.StatusRecoveryReviewRequired || tx.args[7] != member.ExecutionGeneration || tx.args[6] != member.DigestKeyVersion {
						t.Fatal("Store transition lost original generation/retained key version")
					}
					for i, value := range [][]byte{tr.ScopeHash, tr.KeyLocator, tr.KeyDigest, tr.RequestDigest} {
						if !bytes.Equal(tx.args[i+2].([]byte), value) {
							t.Fatal("Store transition changed original execution identity")
						}
					}
					tr.KeyDigest[0] ^= 1
					if cohort[0].KeyDigest != member.KeyDigest {
						t.Fatal("cohort retained borrowed mutable row bytes")
					}
				}
				if !rows.closed || rows.releases != 1 {
					t.Fatal("invalidation did not close Rows before transition")
				}
				return s410Stop
			}}); err != nil {
				t.Fatal(err)
			}
			if err := rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) || calls != 1 {
				t.Fatalf("public producer: %v calls=%d", err, calls)
			}
		})
	}
}

type s410UnitClock struct{}

func (s410UnitClock) Now() time.Time { return s410Time.Add(time.Minute) }

// Explicit complete pgx surfaces: unexpected operations fail immediately; these
// unit ports do not emulate a commit, connection, lease, or a positive SQL proof.
type s410CohortTx struct {
	rows   *s410CohortRows
	writes int
	args   []any
}

func (x *s410CohortTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected unit Begin")
}
func (x *s410CohortTx) Commit(context.Context) error   { return errors.New("unexpected unit Commit") }
func (x *s410CohortTx) Rollback(context.Context) error { return errors.New("unexpected unit Rollback") }
func (x *s410CohortTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("unexpected unit CopyFrom")
}
func (x *s410CohortTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	panic("unexpected unit SendBatch")
}
func (x *s410CohortTx) LargeObjects() pgx.LargeObjects { panic("unexpected unit LargeObjects") }
func (x *s410CohortTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("unexpected unit Prepare")
}
func (x *s410CohortTx) Conn() *pgx.Conn { panic("unexpected unit Conn") }
func (x *s410CohortTx) Query(c context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return x.rows, c.Err()
}
func (x *s410CohortTx) QueryRow(context.Context, string, ...any) pgx.Row { return s410UnitErrorRow{} }
func (x *s410CohortTx) Exec(c context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	if !x.rows.closed || x.rows.err != nil {
		return pgconn.CommandTag{}, errors.New("write before full Rows join")
	}
	if err := c.Err(); err != nil {
		return pgconn.CommandTag{}, err
	}
	x.writes++
	x.args = append([]any{}, args...)
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

type s410UnitErrorRow struct{}

func (s410UnitErrorRow) Scan(...any) error { return errors.New("unexpected unit QueryRow") }

type s410CohortRows struct {
	values   []any
	next     int
	closed   bool
	releases int
	err      error
}

func (r *s410CohortRows) Close() {
	if !r.closed {
		r.closed = true
		r.releases++
	}
}
func (r *s410CohortRows) Err() error { return r.err }
func (r *s410CohortRows) Next() bool {
	r.next++
	if r.next == 1 {
		return true
	}
	r.Close()
	return false
}
func (r *s410CohortRows) Scan(dest ...any) error {
	if len(dest) != len(r.values) {
		return errors.New("wrong explicit cohort row width")
	}
	for i, d := range dest {
		switch p := d.(type) {
		case *[]byte:
			*p = r.values[i].([]byte)
		case *string:
			*p = r.values[i].(string)
		case *int64:
			*p = r.values[i].(int64)
		case *time.Time:
			*p = r.values[i].(time.Time)
		default:
			return errors.New("unexpected cohort scan type")
		}
	}
	return nil
}
func (r *s410CohortRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT 1") }
func (r *s410CohortRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *s410CohortRows) Values() ([]any, error)                       { return append([]any{}, r.values...), r.err }
func (r *s410CohortRows) RawValues() [][]byte                          { panic("unexpected unit RawValues") }
func (r *s410CohortRows) Conn() *pgx.Conn                              { return nil }
func (r *s410CohortRows) TypeMap() *pgtype.Map                         { return pgtype.NewMap() }
func s410CanonicalUnit(t *testing.T) {
	t.Run("RealCanonicalAAndDeletedBIdentifierContracts", func(t *testing.T) {
		email, phone := "old@example.invalid", "+123synthetic"
		u := User{ID: "22222222-2222-4222-8222-222222222222", Username: "original", Status: UserStatusSuspended, Email: &email, Phone: &phone}
		ids := []IdentityIdentifier{{UserID: u.ID, Kind: IdentifierUsername, NormalizedValue: u.Username, Lifecycle: IdentifierActive}, {UserID: u.ID, Kind: IdentifierEmail, NormalizedValue: email, Lifecycle: IdentifierActive}, {UserID: u.ID, Kind: IdentifierPhone, NormalizedValue: phone, Lifecycle: IdentifierActive}}
		if err := validateCanonicalDeleteIdentifierTombstones(u, ids); err != nil {
			t.Fatal(err)
		}
		u.Status = UserStatusDeleted
		u.Email = nil
		u.Phone = nil
		u.RetainedEmail = &email
		for i := range ids {
			ids[i].Lifecycle = IdentifierReserved
		}
		if err := validateCanonicalDeleteIdentifierTombstones(u, ids); err != nil {
			t.Fatal(err)
		}
		for _, kind := range []IdentifierKind{IdentifierUsername, IdentifierEmail, IdentifierPhone} {
			t.Run(string(kind), func(t *testing.T) {
				bad := append([]IdentityIdentifier{}, ids...)
				for i := range bad {
					if bad[i].Kind == kind {
						bad[i].Lifecycle = IdentifierActive
					}
				}
				if err := validateCanonicalDeleteIdentifierTombstones(u, bad); err == nil {
					t.Fatal("deleted B accepted released/active tombstone")
				}
			})
		}
	})
	t.Run("RealPublicIdentityPermitCannotInventReviewDecision", func(t *testing.T) {
		s, j := s410MemoryInput()
		f := NewRecoveryIdentityForwarder()
		rt, ctx, done := s410Runtime(t, s, s, j, f)
		defer done()
		e := &recoveryIdentityExecution{d: recoveryExecutionDependencies{runtime: rt}}
		_ = f.Seal(&s410Callbacks{finalize: func(c context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
			for _, tc := range []struct{ decision, evidence string }{{"restored", "source"}, {"accepted", "source"}, {"denied", ""}, {"kept_deleted", ""}} {
				if err := e.ResolveRestrictedAccountReview(c, p, b, "22222222-2222-4222-8222-222222222222", tc.decision, tc.evidence); err == nil {
					t.Fatal("invalid decision/evidence reached Store or minted authority")
				}
			}
			return s410Stop
		}})
		if err := rt.Resume(ctx, s410Binding); !errors.Is(err, s410Stop) {
			t.Fatal(err)
		}
	})
}

func s410ExitReceiptUnit(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	for _, tc := range []struct {
		name, raw string
		good      bool
	}{
		{"empty", `{"revoked_ids":[],"revoked_count":0}`, true},
		{"one", `{"revoked_ids":["` + id + `"],"revoked_count":1}`, true},
		{"missing_ids", `{"revoked_count":0}`, false},
		{"missing_count", `{"revoked_ids":[]}`, false},
		{"null_ids", `{"revoked_ids":null,"revoked_count":0}`, false},
		{"null_count", `{"revoked_ids":[],"revoked_count":null}`, false},
		{"non_uuid", `{"revoked_ids":["invented"],"revoked_count":1}`, false},
		{"duplicate", `{"revoked_ids":["` + id + `","` + id + `"],"revoked_count":2}`, false},
		{"mismatched_count", `{"revoked_ids":[],"revoked_count":1}`, false},
		{"negative_count", `{"revoked_ids":[],"revoked_count":-1}`, false},
		{"fractional_count", `{"revoked_ids":[],"revoked_count":0.5}`, false},
		{"string_count", `{"revoked_ids":[],"revoked_count":"0"}`, false},
		{"null_element", `{"revoked_ids":[null],"revoked_count":1}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ids, err := recoveryExitReceiptIDs([]byte(tc.raw))
			if (err == nil) != tc.good {
				t.Fatalf("receipt structure: ids=%v err=%v", ids, err)
			}
		})
	}
}

// Exercise the real finalization consumer with independent full read projections.
func s410DeletionInstantProof(t *testing.T) {
	base, err := time.Parse(time.RFC3339Nano, "2026-10-29T07:00:00.123456Z")
	if err != nil {
		t.Fatal(err)
	}
	zero, _ := time.Parse(time.RFC3339Nano, "2026-10-29T07:00:00.123456+00:00")
	offset, _ := time.Parse(time.RFC3339Nano, "2026-10-29T16:30:00.123456+09:30")
	later := base.Add(time.Microsecond)
	for _, name := range []string{"both_nil", "UTC_Z", "zero_offset", "nonzero_offset", "microsecond", "nil_source", "nil_review", "role", "status", "member", "missing_account"} {
		t.Run(name, func(t *testing.T) {
			sourceTime, reviewTime := &base, &base
			role, status, user := "member", "deleted", s410TimeUser
			good := name == "both_nil" || name == "UTC_Z" || name == "zero_offset" || name == "nonzero_offset"
			switch name {
			case "both_nil":
				sourceTime, reviewTime = nil, nil
				status = "suspended"
			case "zero_offset":
				reviewTime = &zero
			case "nonzero_offset":
				reviewTime = &offset
			case "microsecond":
				reviewTime = &later
			case "nil_source":
				sourceTime = nil
			case "nil_review":
				reviewTime = nil
			case "role":
				role = "admin"
			case "status":
				status = "suspended"
			case "member":
				user = "44444444-4444-4444-8444-444444444444"
			}
			f := recoveryExecutionFinalization{RunID: s410Binding.RunID, BackupRef: s410Binding.BackupRef, Install: "s410-install", Accounts: 1, Reviews: 1, CohortCount: "0", Cohort: []recoveryIdempotencyMember{}, Sources: []recoveryExecutionSource{{ID: s410TimeUser, Role: "member", Status: func() string {
				if name == "both_nil" {
					return "suspended"
				}
				return "deleted"
			}(), DeletedAt: sourceTime}}}
			auditRaw, _ := json.Marshal(f)
			userRaw, _ := json.Marshal(recoveryExecutionUser{ID: user})
			reviewRaw, _ := json.Marshal(recoveryExecutionReview{UserID: s410TimeUser, BackupRole: role, BackupStatus: status, BackupDeletedAt: reviewTime})
			if name == "missing_account" {
				userRaw = nil
			}
			tx := &s410TimeProofTx{audit: auditRaw, user: userRaw, review: reviewRaw}
			auditID := "55555555-5555-4555-8555-555555555555"
			_, got := (&recoveryIdentityExecution{}).finalization(context.Background(), tx, recovery.AdmissionRun{AuditEventID: &auditID, AccountCount: 1}, s410Binding, "s410-install")
			if (got == nil) != good {
				t.Fatalf("real finalization instant proof: expected success=%t got=%v", good, got)
			}
			for _, rows := range tx.reads {
				if rows.releases != 1 {
					t.Fatal("finalization did not close an owned projection")
				}
			}
		})
	}
}

const s410TimeUser = "33333333-3333-4333-8333-333333333333"

type s410TimeProofTx struct {
	s410CohortTx
	audit, user, review []byte
	reads               []*s410TimeProofRows
}

func (x *s410TimeProofTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	var raw []byte
	switch {
	case strings.Contains(sql, "SELECT metadata::text FROM audit_events"):
		raw = x.audit
	case strings.Contains(sql, "to_jsonb(u)"):
		raw = x.user
	case strings.Contains(sql, "to_jsonb(r)"):
		raw = x.review
	default:
		return nil, errors.New("unexpected time-proof query")
	}
	rows := &s410TimeProofRows{raw: raw}
	x.reads = append(x.reads, rows)
	return rows, nil
}
func (x *s410TimeProofTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.HasPrefix(sql, "SELECT EXISTS(SELECT 1 FROM audit_events") {
		return s410TimeBoolRow{}
	}
	return s410UnitErrorRow{}
}
func (x *s410TimeProofTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("time proof attempted a write")
}

type s410TimeBoolRow struct{}

func (s410TimeBoolRow) Scan(dst ...any) error {
	if len(dst) != 1 {
		return errors.New("unexpected audit proof width")
	}
	p, ok := dst[0].(*bool)
	if !ok {
		return errors.New("unexpected audit proof type")
	}
	*p = true
	return nil
}

type s410TimeProofRows struct {
	s410CohortRows
	raw  []byte
	sent bool
}

func (r *s410TimeProofRows) Next() bool {
	if r.sent || r.raw == nil {
		r.Close()
		return false
	}
	r.sent = true
	return true
}
func (r *s410TimeProofRows) Scan(dst ...any) error {
	if len(dst) != 1 {
		return errors.New("unexpected time proof width")
	}
	p, ok := dst[0].(*[]byte)
	if !ok {
		return errors.New("unexpected time proof type")
	}
	*p = append([]byte{}, r.raw...)
	return nil
}

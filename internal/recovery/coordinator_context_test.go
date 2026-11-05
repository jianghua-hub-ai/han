package recovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRecoveryCoordinatorPublication(t *testing.T) {
	t.Run("ModeDependencies", testCoordinatorDependencies)
	t.Run("FreshPublicationAndConsumedWindow", func(t *testing.T) { testPublicCoordinator(t); testConsumerSourceContinuity(t) })
	t.Run("ContextLockAndWriteUnknown", testCoordinatorUnknown)
	t.Run("RejectCallerContext", func(t *testing.T) {
		run, j := finalStateFixture(mustReadyFixture(t), RecoveryOutcomeSucceeded)
		g := &stubArticleDeletionGate{read: []byte(readyFixtureV1)}
		c, e := NewCoordinator(newMemoryRunStore(run), j, g, publicationForReady(mustReadyFixture(t)), NewReplayStatusV1Decoder(nil), time.Now)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := c.Consume(context.Background(), run.RunID); e == nil {
			t.Fatal("caller context consumed without a Runtime replay capability")
		}
		if g.readCalls != 0 {
			t.Fatal("unpermitted consumer read article source")
		}
	})
}

func testCoordinatorDependencies(t *testing.T) {
	for _, name := range []string{"store", "journal", "gate", "publications", "decoder", "clock"} {
		t.Run(name, func(t *testing.T) {
			var store RecoveryRunStore = newMemoryRunStore(RecoveryRun{})
			var journal ContextRecoveryJournal = &memoryJournal{}
			var gate ArticleDeletionGate = &stubArticleDeletionGate{}
			var pubs PublicationPort = publicationForReady(mustReadyFixture(t))
			decoder := NewReplayStatusV1Decoder(nil)
			clock := time.Now
			switch name {
			case "store":
				var missing *memoryRunStore
				store = missing
			case "journal":
				var missing *memoryJournal
				journal = missing
			case "gate":
				var missing *stubArticleDeletionGate
				gate = missing
			case "publications":
				var missing *testPublication
				pubs = missing
			case "decoder":
				decoder = ReplayStatusV1Decoder{}
			case "clock":
				clock = nil
			}
			if _, e := NewCoordinator(store, journal, gate, pubs, decoder, clock); !errors.Is(e, ErrAdmissionBinding) {
				t.Fatalf("missing dependency accepted: %v", e)
			}
		})
	}
}

type componentRunStore struct {
	afterTransition func()
	c               *controlledFixture
	transitions     int
	failAt          RecoveryOutcome
	commit          bool
}

func (s *componentRunStore) Load(ctx context.Context, id string) (RecoveryRun, error) {
	if e := checkedContext(ctx); e != nil {
		return RecoveryRun{}, e
	}
	for _, x := range s.c.f.snapshot.Runs {
		if x.RunID == id {
			r := fixtureRecoveryRun(x)
			if x.CompletedAt != nil {
				r.CompletedAt = *x.CompletedAt
			}
			if x.ArticleReplayVerifiedAt != nil {
				r.ArticleReplayVerifiedAt = *x.ArticleReplayVerifiedAt
			}
			return r, nil
		}
	}
	return RecoveryRun{}, ErrRunNotFound
}
func (s *componentRunStore) Transition(ctx context.Context, id string, old, next RecoveryOutcome, ch RecoveryRunChange) (RecoveryRun, error) {
	if e := checkedContext(ctx); e != nil {
		return RecoveryRun{}, e
	}
	s.transitions++
	if s.failAt == next && !s.commit {
		return RecoveryRun{}, errors.New("synthetic Store ACK unknown")
	}
	x := &s.c.f.snapshot.Runs[1]
	if x.RunID != id || x.Outcome != old {
		return RecoveryRun{}, ErrRecoveryRunConflict
	}
	x.Outcome = next
	if ch.ArticleEvidence != nil {
		v := ch.ArticleEvidence
		x.ArticleEvidenceVersion = fixtureText(v.EvidenceVersion)
		x.ArticleEvidenceDigest = fixtureText(v.EvidenceDigest)
		x.ArticleEvidenceRef = fixtureText(v.EvidenceRef)
		x.ArticleNamespace = fixtureText(v.Namespace)
		x.ArticleEvidenceCompletedAt = fixtureTime(ch.ArticleEvidenceCompletedAt)
		x.ArticleReplayVerifiedAt = fixtureTime(ch.ChangedAt)
	}
	if ch.ArticlePublication != nil {
		x.ArticlePublicationRef = fixtureText(ch.ArticlePublication.Ref)
		x.ArticlePublicationDigest = fixtureText(ch.ArticlePublication.Digest)
	}
	if ch.JournalConsumedAt != nil {
		x.JournalStage = StageConsumed
		x.JournalConsumedAt = fixtureTime(*ch.JournalConsumedAt)
	}
	if next == RecoveryOutcomeSucceeded {
		x.CompletedAt = fixtureTime(ch.ChangedAt)
	}
	s.c.f.snapshot.Revision.Counter++
	if s.afterTransition != nil {
		s.afterTransition()
	}
	if s.failAt == next {
		return RecoveryRun{}, errors.New("synthetic committed Store ACK lost")
	}
	return s.Load(ctx, id)
}

type componentJournal struct {
	beforeMark func()
	*admissionFixture
	c       *controlledFixture
	marks   int
	ackLost bool
}

func (j *componentJournal) Read(ctx context.Context, id string) (JournalEntry, error) {
	if e := checkedContext(ctx); e != nil {
		return JournalEntry{}, e
	}
	_, x, e := snapshotRun(j.fSnapshot(), id)
	return x, e
}
func (j *componentJournal) fSnapshot() VerifiedJournalSnapshot { return j.c.f.recovery }
func (j *componentJournal) VerifyAgainst(ctx context.Context, x RecoveryRun) (JournalEntry, error) {
	if e := checkedContext(ctx); e != nil {
		return JournalEntry{}, e
	}
	b, last, e := snapshotRun(j.fSnapshot(), x.RunID)
	if e != nil {
		return JournalEntry{}, e
	}
	if b.BackupRef != x.BackupRef || b.KeyID != x.JournalKeyID || b.KeyVersion != x.JournalKeyVersion || b.SecretFingerprint != x.JournalSecretFingerprint || !b.RecordedAt.Equal(x.StartedAt) || last.Stage != x.JournalStage {
		return JournalEntry{}, ErrJournalMismatch
	}
	if last.Stage == StageConsumed && (x.JournalConsumedAt == nil || !last.RecordedAt.Equal(*x.JournalConsumedAt)) {
		return JournalEntry{}, ErrJournalMismatch
	}
	return last, nil
}
func (j *componentJournal) MarkConsumed(ctx context.Context, id string) (JournalEntry, error) {
	if e := checkedContext(ctx); e != nil {
		return JournalEntry{}, e
	}
	j.marks++
	if id != j.c.binding().RunID {
		return JournalEntry{}, ErrAdmissionBinding
	}
	if j.beforeMark != nil {
		j.beforeMark()
	}
	source := j.c.f.recovery
	if e := validateConsumptionAnchor(ctx, AnchorSnapshot{JournalID: source.JournalID, Namespace: source.Namespace, Epoch: source.Epoch, Committed: source.Head}, source.Revision); e != nil {
		return JournalEntry{}, e
	}
	j.c.publishConsumed()
	if j.ackLost {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, errors.New("synthetic Journal ACK lost"))
	}
	return j.Read(ctx, id)
}

type componentGate struct {
	raw            []byte
	replays, reads int
	readHook       func(context.Context)
}

func (g *componentGate) ReplayAndVerify(ctx context.Context, _, _ string) ([]byte, error) {
	g.replays++
	return g.raw, checkedContext(ctx)
}
func (g *componentGate) ReadStatus(ctx context.Context, _, _ string) ([]byte, error) {
	g.reads++
	if g.readHook != nil {
		g.readHook(ctx)
	}
	return g.raw, checkedContext(ctx)
}

func publicCoordinatorFixture(t *testing.T, stage RecoveryOutcome) (*controlledFixture, Runtime, *componentRunStore, *componentJournal, *testPublication, *componentGate) {
	t.Helper()
	c, r := newControlledFixture(t, stage)
	p := publicationForReady(c.closure.Ready)
	p.binding.Ref = c.closure.PublicationRef
	p.binding.Digest = c.closure.PublicationDigest
	s := &componentRunStore{c: c}
	j := &componentJournal{admissionFixture: c.f, c: c}
	raw, e := json.Marshal(c.closure.Ready)
	if e != nil {
		t.Fatal(e)
	}
	g := &componentGate{raw: raw}
	coordinator, e := NewCoordinator(s, j, g, p, NewReplayStatusV1Decoder(nil), func() time.Time { return *c.complete.CompletedAt })
	if e != nil {
		t.Fatal(e)
	}
	r.(*admissionRuntime).deps.Consumer = coordinator
	if stage != RecoveryOutcomeIdentityReady && stage != RecoveryOutcomeSucceeded {
		x := c.complete
		x.Outcome = stage
		x.CompletedAt = nil
		if stage == RecoveryOutcomeJournalConsumed {
			c.publishConsumed()
		} else {
			x.JournalStage = StageBegun
			x.JournalConsumedAt = nil
		}
		c.f.snapshot.Runs[1] = x
		c.f.snapshot.Revision.Counter++
	}
	return c, r, s, j, p, g
}

func testPublicCoordinator(t *testing.T) {
	for _, stage := range []RecoveryOutcome{RecoveryOutcomeIdentityReady, RecoveryOutcomeArticleDeletionReady, RecoveryOutcomeReadyToConsume, RecoveryOutcomeJournalConsumed, RecoveryOutcomeSucceeded} {
		t.Run(string(stage), func(t *testing.T) {
			c, r, s, j, p, g := publicCoordinatorFixture(t, stage)
			if e := r.Resume(context.Background(), c.binding()); e != nil {
				t.Fatalf("public producer/Coordinator: %v steps=%v", e, c.steps)
			}
			if c.revokes != 1 || c.f.starts != 1 {
				t.Fatal("Consumer did not lead to exit/startup")
			}
			if stage == RecoveryOutcomeSucceeded && s.transitions != 0 {
				t.Fatal("succeeded resumed Consume")
			}
			if stage == RecoveryOutcomeIdentityReady && (g.replays != 1 || p.readCalls < 2) {
				t.Fatal("fresh source/publication not checked")
			}
			if stage == RecoveryOutcomeJournalConsumed && j.marks != 0 {
				t.Fatal("consumed appended again")
			}
		})
	}
	t.Run("MissingExactPublicationEnsure", func(t *testing.T) {
		c, r, _, _, p, _ := publicCoordinatorFixture(t, RecoveryOutcomeIdentityReady)
		p.missing = true
		if e := r.Resume(context.Background(), c.binding()); e != nil {
			t.Fatal(e)
		}
		if p.ensureCalls != 1 || p.readCalls < 3 {
			t.Fatal("Ensure not independently reread")
		}
	})
	t.Run("ExactConsumedWindow", func(t *testing.T) {
		c, r, _, j, _, _ := publicCoordinatorFixture(t, RecoveryOutcomeReadyToConsume)
		c.publishConsumed()
		if e := r.Resume(context.Background(), c.binding()); e != nil {
			t.Fatal(e)
		}
		if j.marks != 0 {
			t.Fatal("reconciliation called MarkConsumed")
		}
	})
	for _, bad := range []string{"stale", "wrong-publication", "wrong-journal", "wrong-epoch", "publication-drift", "missing-later-publication", "cancel-before-entry", "cancel"} {
		t.Run(bad, func(t *testing.T) { testBadCoordinatorSource(t, bad) })
	}
}

func testBadCoordinatorSource(t *testing.T, bad string) {
	c, r, s, j, p, g := publicCoordinatorFixture(t, RecoveryOutcomeIdentityReady)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	switch bad {
	case "stale":
		g.readHook = func(context.Context) { g.raw = []byte(unavailableFixtureV1) }
	case "wrong-publication":
		p.binding.BackupRef = "other"
	case "wrong-journal":
		p.binding.JournalID = "wrong"
	case "wrong-epoch":
		p.binding.Epoch++
	case "publication-drift":
		p.hook = func() {
			if p.readCalls == 2 {
				p.binding.Ref = "changed"
			}
		}
	case "missing-later-publication":
		c, r, s, j, p, g = publicCoordinatorFixture(t, RecoveryOutcomeArticleDeletionReady)
		p.missing = true
	case "cancel-before-entry":
		cancel()
	case "cancel":
		p.hook = cancel
	}
	if e := r.Resume(ctx, c.binding()); e == nil {
		t.Fatal("invalid fresh source admitted")
	}
	if s.transitions != 0 || j.marks != 0 || c.revokes != 0 || c.f.starts != 0 {
		t.Fatalf("failed fence wrote/started: transitions=%d marks=%d", s.transitions, j.marks)
	}
	if p.ensureCalls != 0 {
		t.Fatal("unexpected Ensure")
	}
}

func testCoordinatorUnknown(t *testing.T) {
	t.Run("EnsureCancellationActuallyJoins", testEnsureCancellationJoin)
	for _, stage := range []RecoveryOutcome{RecoveryOutcomeArticleDeletionReady, RecoveryOutcomeReadyToConsume, RecoveryOutcomeJournalConsumed, RecoveryOutcomeSucceeded} {
		for _, commit := range []bool{false, true} {
			t.Run(string(stage)+map[bool]string{false: "/Unknown", true: "/CommittedACK"}[commit], func(t *testing.T) {
				c, r, s, _, _, _ := publicCoordinatorFixture(t, RecoveryOutcomeIdentityReady)
				s.failAt = stage
				s.commit = commit
				e := r.Resume(context.Background(), c.binding())
				if commit && e != nil {
					t.Fatalf("exact committed ACK readback: %v", e)
				}
				if !commit && (!errors.Is(e, ErrRecoveryWriteUnknown) || c.f.starts != 0) {
					t.Fatalf("unknown advanced: %v", e)
				}
			})
		}
	}
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "EnsureUnknown", true: "EnsureCommittedACK"}[commit], func(t *testing.T) {
			c, r, _, _, p, _ := publicCoordinatorFixture(t, RecoveryOutcomeIdentityReady)
			p.missing = true
			cause := errors.New("synthetic Ensure ACK lost")
			p.ensureErr = cause
			p.rejectCommit = !commit
			e := r.Resume(context.Background(), c.binding())
			if commit && e != nil {
				t.Fatalf("committed Ensure not independently confirmed: %v", e)
			}
			if !commit && (!errors.Is(e, ErrRecoveryWriteUnknown) || !errors.Is(e, cause) || c.f.starts != 0) {
				t.Fatalf("Ensure unknown/cause: %v", e)
			}
			if p.ensureCalls != 1 {
				t.Fatal("Ensure repeated")
			}
		})
	}
	t.Run("JournalCommittedACK", func(t *testing.T) {
		c, r, _, j, _, _ := publicCoordinatorFixture(t, RecoveryOutcomeReadyToConsume)
		j.ackLost = true
		if e := r.Resume(context.Background(), c.binding()); e != nil {
			t.Fatal(e)
		}
		if j.marks != 1 {
			t.Fatal("Journal write repeated")
		}
	})
}

func testEnsureCancellationJoin(t *testing.T) {
	c, r, _, _, p, _ := publicCoordinatorFixture(t, RecoveryOutcomeIdentityReady)
	p.missing = true
	p.entered = make(chan struct{})
	p.release = make(chan struct{})
	cause := errors.New("synthetic Ensure ACK unknown")
	p.ensureErr = cause
	done := make(chan error, 1)
	go func() { done <- r.Resume(context.Background(), c.binding()) }()
	select {
	case <-p.entered:
	case <-time.After(time.Second):
		t.Fatal("Ensure boundary not reached")
	}
	reads := p.readCalls
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	select {
	case <-r.(*admissionRuntime).generationCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel owned generation")
	}
	select {
	case <-stopped:
		t.Error("Stop released before actual Ensure completion")
	default:
	}
	close(p.release)
	select {
	case e := <-done:
		if !errors.Is(e, ErrRecoveryWriteUnknown) || !errors.Is(e, cause) || !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("Ensure did not join")
	}
	select {
	case e := <-stopped:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not finish")
	}
	if p.readCalls != reads || p.ensureCalls != 1 || c.f.starts != 0 {
		t.Fatal("cancelled Ensure performed readback/retry/startup")
	}
}

type testPublication struct {
	ensureErr              error
	rejectCommit           bool
	entered, release       chan struct{}
	binding                PublicationBinding
	missing                bool
	readCalls, ensureCalls int
	hook                   func()
}

func publicationForReady(v ReadyReplayStatusV1) *testPublication {
	completed, _ := time.Parse(time.RFC3339Nano, v.CompletedAt)
	return &testPublication{binding: PublicationBinding{Ref: "publication-test", Digest: fixtureHash("publication-test"), Version: "article-replay-publication/v1", RunID: v.RunID, BackupRef: v.BackupRef, EvidenceRef: v.EvidenceRef, EvidenceDigest: v.EvidenceDigest, JournalID: "article-journal", Epoch: 2, Head: JournalHead{Sequence: v.JournalHeadSequence, Hash: v.JournalHeadHash}, EvidenceCompletedAt: completed, PublishedAt: completed}}
}
func (p *testPublication) ReadCurrentPublication(ctx context.Context, run, backup, ref, digest string) (PublicationBinding, error) {
	p.readCalls++
	if e := checkedContext(ctx); e != nil {
		return PublicationBinding{}, e
	}
	if p.hook != nil {
		p.hook()
	}
	if p.missing {
		return PublicationBinding{}, ErrPublicationMissing
	}
	x := p.binding
	if x.RunID != run || x.BackupRef != backup || x.EvidenceRef != ref || x.EvidenceDigest != digest {
		return PublicationBinding{}, ErrAdmissionBinding
	}
	return x, nil
}
func (p *testPublication) EnsureCurrentPublication(ctx context.Context, permit PhasePermit, run, backup, ref, digest string) (PublicationBinding, error) {
	p.ensureCalls++
	if permit.grant == nil || permit.grant.owner.CheckPhase(ctx, permit, PhaseRecoveryReplay) != nil {
		return PublicationBinding{}, ErrAdmissionBinding
	}
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	if !p.rejectCommit {
		p.missing = false
	}
	return p.binding, p.ensureErr
}

func (j *memoryJournal) ReadVerifiedSnapshot(ctx context.Context, ns string) (VerifiedJournalSnapshot, error) {
	if j.source == nil {
		return VerifiedJournalSnapshot{}, ErrAdmissionIncomplete
	}
	return j.source.f.ReadVerifiedSnapshot(ctx, ns)
}
func (j *memoryJournal) VerifyBeginSecret(context.Context, RunBinding, []byte) (JournalEntry, error) {
	return JournalEntry{}, ErrAdmissionBinding
}

type heldReplayConsumer struct {
	entered chan context.Context
	release chan struct{}
}

func (h heldReplayConsumer) Consume(ctx context.Context, _ string) (ConsumptionResult, error) {
	h.entered <- ctx
	<-h.release
	return ConsumptionResult{}, ErrAdmissionClosed
}

// The legacy behavior tests retain their assertions while using a real public
// Resume-produced replay capability; this adapter is test-only.
type legacyContextCoordinator struct {
	*Coordinator
	t       *testing.T
	store   *memoryRunStore
	journal *memoryJournal
}

func (c *legacyContextCoordinator) Consume(ctx context.Context, run string) (ConsumptionResult, error) {
	r, p := publicReplayPermit(c.t, c.store, c.journal)
	pc, e := r.PhaseContext(ctx, p)
	if e != nil {
		return ConsumptionResult{}, e
	}
	return c.Coordinator.Consume(pc, run)
}
func newLegacyCoordinator(t *testing.T, store RecoveryRunStore, journal ContextRecoveryJournal, gate ArticleDeletionGate, decoder ReplayStatusV1Decoder, now func() time.Time) (*legacyContextCoordinator, error) {
	t.Helper()
	c, e := NewCoordinator(store, journal, gate, publicationForReady(mustReadyFixture(t)), decoder, now)
	return &legacyContextCoordinator{Coordinator: c, t: t, store: store.(*memoryRunStore), journal: journal.(*memoryJournal)}, e
}
func publicReplayPermit(t *testing.T, store *memoryRunStore, journal *memoryJournal) (Runtime, PhasePermit) {
	t.Helper()
	c, r := legacySourceFixture(t, store, journal)
	h := heldReplayConsumer{entered: make(chan context.Context, 1), release: make(chan struct{})}
	r.(*admissionRuntime).deps.Consumer = h
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	joined := make(chan struct{})
	go func() { defer close(joined); done <- r.Resume(ctx, c.binding()) }()
	t.Cleanup(func() {
		close(h.release)
		select {
		case <-joined:
		case <-time.After(30 * time.Second):
			t.Error("held public Resume did not join")
		}
		cancel()
	})
	select {
	case pc := <-h.entered:
		auth := pc.Value(runtimeContextKey{}).(runtimeAuthorization)
		return r, PhasePermit{grant: auth.phase}
	case e := <-done:
		t.Fatalf("public replay producer failed before consumer: %v", e)
	case <-ctx.Done():
		t.Fatal("public replay producer timed out")
	}
	return nil, PhasePermit{}
}

// A coherent synthetic source is shared with the real public producer. Each
// invocation represents a fresh recovery process, including crash convergence.
func legacySourceFixture(t *testing.T, store *memoryRunStore, journal *memoryJournal) (*controlledFixture, Runtime) {
	t.Helper()
	c, r := newControlledFixture(t, RecoveryOutcomeIdentityReady)
	x := &store.run
	x.JournalKeyID = fixtureJournalKey().ID
	x.JournalKeyVersion = fixtureJournalKey().Version
	x.JournalSecretFingerprint = fingerprintSecret(fixtureJournalKey(), []byte("synthetic-candidate"))
	c.complete.RunID = x.RunID
	c.complete.BackupRef = x.BackupRef
	c.complete.StartedAt = x.StartedAt
	c.complete.JournalKeyID = x.JournalKeyID
	c.complete.JournalKeyVersion = x.JournalKeyVersion
	c.complete.JournalSecretFingerprint = x.JournalSecretFingerprint
	receipt := *c.complete.RestoreBinding
	receipt.RunID = x.RunID
	receipt.BackupRef = x.BackupRef
	receipt.RestoredAt = x.StartedAt
	c.complete.RestoreBinding = &receipt
	c.plan.RunID = x.RunID
	c.plan.BackupRef = x.BackupRef
	c.begin.RunID = x.RunID
	c.begin.BackupRef = x.BackupRef
	c.begin.RecordedAt = x.StartedAt
	c.begin.KeyID = x.JournalKeyID
	c.begin.KeyVersion = x.JournalKeyVersion
	c.begin.SecretFingerprint = x.JournalSecretFingerprint
	c.begin = fixtureSignEntry(c.begin)
	c.f.recovery.Entries[len(c.f.recovery.Entries)-1] = c.begin
	c.f.snapshot.Runs = c.f.snapshot.Runs[:1]
	c.putStage(RecoveryOutcomeIdentityReady)
	if journal.entry.Stage == StageConsumed {
		c.complete.JournalConsumedAt = fixtureTime(journal.entry.RecordedAt)
		c.publishConsumed()
		c.f.snapshot.Runs[1].Outcome = RecoveryOutcomeReadyToConsume
		ready := c.complete
		ready.Outcome = RecoveryOutcomeReadyToConsume
		ready.JournalStage = StageBegun
		ready.JournalConsumedAt = nil
		ready.CompletedAt = nil
		c.f.snapshot.Runs[1] = ready
	}
	legacyArticleSource(t, c)
	journal.source = c
	journal.begunAt = x.StartedAt
	_, journal.entry, _ = snapshotRun(c.f.recovery, x.RunID)
	return c, r
}

func testConsumerSourceContinuity(t *testing.T) {
	for _, boundary := range []string{"Gate", "Publication", "Store", "Mark"} {
		for _, ns := range []string{RecoveryNamespaceV1, ArticleDeletionNamespaceV1} {
			t.Run(boundary+"/"+ns, func(t *testing.T) {
				c, r, s, j, p, g := publicCoordinatorFixture(t, RecoveryOutcomeIdentityReady)
				drift := func() {
					if ns == RecoveryNamespaceV1 {
						c.f.recovery.Revision += "-unexplained"
					} else {
						c.f.article.Revision += "-unexplained"
					}
				}
				switch boundary {
				case "Gate":
					g.readHook = func(context.Context) { drift() }
				case "Publication":
					p.hook = drift
				case "Store":
					s.afterTransition = drift
				case "Mark":
					j.beforeMark = drift
				}
				if e := r.Resume(context.Background(), c.binding()); !errors.Is(e, ErrAdmissionDrift) {
					t.Fatalf("namespace ABA not rejected: %v", e)
				}
				limit := 0
				if boundary == "Store" {
					limit = 1
				}
				if boundary == "Mark" {
					limit = 2
				}
				if s.transitions > limit || c.revokes != 0 || c.f.starts != 0 {
					t.Fatalf("unexplained source progressed: transitions=%d", s.transitions)
				}
				expectedMarks := 0
				if boundary == "Mark" {
					expectedMarks = 1
				}
				if j.marks != expectedMarks {
					t.Fatalf("mark calls=%d want=%d", j.marks, expectedMarks)
				}
			})
		}
	}
}

func legacyReadyRaw(t *testing.T) []byte {
	t.Helper()
	raw, e := json.Marshal(mustReadyFixture(t))
	if e != nil {
		t.Fatal(e)
	}
	return raw
}

func legacyArticleSource(t *testing.T, c *controlledFixture) {
	t.Helper()
	ready := mustReadyFixture(t)
	head := JournalHead{Sequence: ready.JournalHeadSequence, Hash: ready.JournalHeadHash}
	for seq := c.f.article.Head.Sequence + 1; seq <= head.Sequence; seq++ {
		next := JournalHead{Sequence: seq, Hash: fixtureHash(fmt.Sprint(seq))}
		if seq == head.Sequence {
			next = head
		}
		c.f.article.Prefixes = append(c.f.article.Prefixes, next)
	}
	c.f.article.Head = head
	c.f.article.Revision += "-legacy-fixture"
	c.f.closure.CurrentHead = head
	c.f.closure.VerifiedArticleRevision = c.f.article.Revision

}

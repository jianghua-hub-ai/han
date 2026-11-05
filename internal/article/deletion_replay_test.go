package article

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"han/internal/recovery"
)

func TestDeletionReplay_SameInputReusesEvidenceRefDigestAndTime(t *testing.T) {
	fixture := newDeletionReplayFixture(t)

	first, err := fixture.provider.ReplayAndVerify(context.Background(), "run-01", "backup-01")
	if err != nil {
		t.Fatalf("first ReplayAndVerify() error = %v", err)
	}
	second, err := fixture.provider.ReplayAndVerify(context.Background(), "run-01", "backup-01")
	if err != nil {
		t.Fatalf("second ReplayAndVerify() error = %v", err)
	}
	assertSameReadyStatus(t, first, second)
	if fixture.store.evidenceCount() != 1 {
		t.Fatalf("evidence count = %d, want 1", fixture.store.evidenceCount())
	}
	if fixture.ids.calls != 1 || fixture.clock.calls != 1 {
		t.Fatalf("metadata calls ids/clock = %d/%d, want 1/1", fixture.ids.calls, fixture.clock.calls)
	}
}

func TestDeletionReplay_ConcurrentSameBackupConvergesToOneEvidence(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	const workers = 12
	results := make(chan []byte, workers)
	errorsSeen := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			raw, err := fixture.provider.ReplayAndVerify(
				context.Background(), "run-concurrent", "backup-one",
			)
			results <- raw
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatalf("ReplayAndVerify() error = %v", err)
		}
	}
	var first []byte
	for raw := range results {
		if first == nil {
			first = raw
			continue
		}
		assertSameReadyStatus(t, first, raw)
	}
	if fixture.store.evidenceCount() != 1 {
		t.Fatalf("evidence count = %d, want 1", fixture.store.evidenceCount())
	}
}

func TestDeletionReplay_ConcurrentDifferentBackupsKeepFirstBinding(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	start := make(chan struct{})
	type result struct {
		backup string
		raw    []byte
		err    error
	}
	results := make(chan result, 2)
	for _, backup := range []string{"backup-a", "backup-b"} {
		backup := backup
		go func() {
			<-start
			raw, err := fixture.provider.ReplayAndVerify(
				context.Background(), "run-binding", backup,
			)
			results <- result{backup: backup, raw: raw, err: err}
		}()
	}
	close(start)
	first := <-results
	second := <-results
	for _, item := range []result{first, second} {
		if item.err != nil {
			t.Fatalf("%s ReplayAndVerify() error = %v", item.backup, item.err)
		}
	}
	statuses := []recovery.ReplayStatusV1{
		decodeReplayStatus(t, first.raw),
		decodeReplayStatus(t, second.raw),
	}
	var readyCount, failedCount int
	for _, status := range statuses {
		if status.Ready != nil {
			readyCount++
		}
		if status.Unavailable != nil && status.Unavailable.Status == replayStatusFailed &&
			status.Unavailable.ReasonCode == replayReasonBackupMismatch {
			failedCount++
		}
	}
	if readyCount != 1 || failedCount != 1 {
		t.Fatalf("ready/failed = %d/%d, want 1/1", readyCount, failedCount)
	}
	if fixture.store.runCount() != 1 {
		t.Fatalf("run count = %d, want 1", fixture.store.runCount())
	}
}

func TestDeletionReplay_InputDriftInvalidatesOldReady(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	if _, err := fixture.provider.ReplayAndVerify(
		context.Background(), "run-drift", "backup-01",
	); err != nil {
		t.Fatalf("ReplayAndVerify() error = %v", err)
	}
	fixture.source.mutate(func(input *DeletionReplayInput) {
		input.InventoryDigest = "sha256:eeee"
	})

	raw, err := fixture.provider.ReadStatus(
		context.Background(), "run-drift", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReadStatus() error = %v", err)
	}
	status := decodeReplayStatus(t, raw)
	if status.Unavailable == nil ||
		status.Unavailable.Status != replayStatusUnknown ||
		status.Unavailable.ReasonCode != replayReasonEvidenceStale {
		t.Fatalf("status = %#v, want stale unavailable", status)
	}
}

func TestDeletionReplay_InputDriftThenReplayPublishesNewCurrentEvidence(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	first, err := fixture.provider.ReplayAndVerify(
		context.Background(), "run-redrift", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReplayAndVerify() error = %v", err)
	}
	fixture.source.mutate(func(input *DeletionReplayInput) {
		input.InventoryDigest = "sha256:deadbeef"
	})
	second, err := fixture.provider.ReplayAndVerify(
		context.Background(), "run-redrift", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReplayAndVerify() after drift error = %v", err)
	}
	firstReady := decodeReplayStatus(t, first).Ready
	secondReady := decodeReplayStatus(t, second).Ready
	if firstReady == nil || secondReady == nil {
		t.Fatalf("expected both replays to reach Ready, got %#v / %#v", firstReady, secondReady)
	}
	if firstReady.EvidenceRef == secondReady.EvidenceRef ||
		firstReady.EvidenceDigest == secondReady.EvidenceDigest {
		t.Fatal("drift must publish new immutable evidence, not reuse the stale ref/digest")
	}
	if fixture.store.evidenceCount() != 2 {
		t.Fatalf("evidence count = %d, want 2", fixture.store.evidenceCount())
	}
	raw, err := fixture.provider.ReadStatus(
		context.Background(), "run-redrift", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReadStatus() error = %v", err)
	}
	current := decodeReplayStatus(t, raw).Ready
	if current == nil || current.EvidenceRef != secondReady.EvidenceRef {
		t.Fatalf("current = %#v, want the post-drift evidence", current)
	}
}

func TestDeletionReplay_StaleFailCannotOverrideNewReady(t *testing.T) {
	ctx := context.Background()
	fixture := newDeletionReplayFixture(t)
	if _, err := fixture.provider.ReplayAndVerify(ctx, "run-stale", "backup-01"); err != nil {
		t.Fatalf("ReplayAndVerify() error = %v", err)
	}
	first, err := fixture.store.LoadAttempt(ctx, "run-stale", "backup-01")
	if err != nil {
		t.Fatalf("LoadAttempt() error = %v", err)
	}

	// A Ready attempt is a Fail no-op: Ready can never be downgraded, even by a
	// forged token.
	if err := fixture.store.Fail(ctx, "run-stale", "backup-01",
		AttemptSnapshot{Exists: true, Generation: first.Generation, Token: "forged"},
		replayReasonVerificationFailed,
	); err != nil {
		t.Fatalf("Fail() after Ready error = %v, want no-op", err)
	}

	// Publish a newer Ready generation via drift, then prove the stale handle
	// cannot Fail it.
	fixture.source.mutate(func(input *DeletionReplayInput) {
		input.InventoryDigest = "sha256:ffff"
	})
	if _, err := fixture.provider.ReplayAndVerify(ctx, "run-stale", "backup-01"); err != nil {
		t.Fatalf("ReplayAndVerify() after drift error = %v", err)
	}
	current, err := fixture.store.LoadAttempt(ctx, "run-stale", "backup-01")
	if err != nil {
		t.Fatalf("LoadAttempt() error = %v", err)
	}
	if current.Generation == first.Generation {
		t.Fatalf("generation did not advance: %d", current.Generation)
	}
	if err := fixture.store.Fail(ctx, "run-stale", "backup-01", first,
		replayReasonVerificationFailed,
	); err != nil {
		t.Fatalf("stale Fail() error = %v, want no-op on Ready", err)
	}
	// A candidate-only / empty-token handle against a running generation is
	// rejected as not-owner.
	running := fixture.store.beginFresh(t, "run-running", "backup-01", fixture.tokens.next())
	if err := fixture.store.Fail(ctx, "run-running", "backup-01",
		AttemptSnapshot{Exists: true, Generation: running.Generation, Token: ""},
		replayReasonVerificationFailed,
	); !errors.Is(err, ErrDeletionReplayAttemptNotOwner) {
		t.Fatalf("Fail() with empty token error = %v, want not-owner", err)
	}
	raw, err := fixture.provider.ReadStatus(ctx, "run-stale", "backup-01")
	if err != nil {
		t.Fatalf("ReadStatus() error = %v", err)
	}
	ready := decodeReplayStatus(t, raw).Ready
	if ready == nil {
		t.Fatal("stale Fail degraded the published Ready attempt")
	}
}
func TestDeletionReplay_CrashPointsResumeWithoutDuplicateEvidence(t *testing.T) {
	t.Run("run inserted before replay", testReplayRunInsertedBeforeReplay)
	t.Run("commit completed before response", testReplayCommitCompletedBeforeResponse)
}

func testReplayRunInsertedBeforeReplay(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	input, err := fixture.source.Snapshot(context.Background(), "backup-01")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	result, err := fixture.store.Begin(
		context.Background(), "run-crash-before-replay", "backup-01",
		AttemptSnapshot{}, input, fixture.tokens.next(),
	)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if result.Disposition != AttemptCreated {
		t.Fatalf("disposition = %q, want created", result.Disposition)
	}
	pending, err := fixture.provider.ReadStatus(
		context.Background(), "run-crash-before-replay", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReadStatus() error = %v", err)
	}
	if got := decodeReplayStatus(t, pending).Unavailable; got == nil ||
		got.Status != replayStatusPending {
		t.Fatalf("pending status = %#v", got)
	}
	if _, err := fixture.provider.ReplayAndVerify(
		context.Background(), "run-crash-before-replay", "backup-01",
	); err != nil {
		t.Fatalf("resume ReplayAndVerify() error = %v", err)
	}
	if fixture.store.evidenceCount() != 1 {
		t.Fatalf("evidence count = %d, want 1", fixture.store.evidenceCount())
	}
}

func testReplayCommitCompletedBeforeResponse(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	first, err := fixture.provider.ReplayAndVerify(
		context.Background(), "run-crash-after-commit", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReplayAndVerify() error = %v", err)
	}
	raw, err := fixture.provider.ReadStatus(
		context.Background(), "run-crash-after-commit", "backup-01",
	)
	if err != nil {
		t.Fatalf("ReadStatus() error = %v", err)
	}
	assertSameReadyStatus(t, first, raw)
	if fixture.store.evidenceCount() != 1 {
		t.Fatalf("evidence count = %d, want 1", fixture.store.evidenceCount())
	}
}

func TestDeletionReplay_BeginCommitUnknownRejoinsSameCandidate(t *testing.T) {
	fixture := newDeletionReplayFixture(t)
	input, err := fixture.source.Snapshot(context.Background(), "backup-01")
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	candidate := fixture.tokens.next()
	first, err := fixture.store.Begin(
		context.Background(), "run-unknown", "backup-01",
		AttemptSnapshot{}, input, candidate,
	)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	second, err := fixture.store.Begin(
		context.Background(), "run-unknown", "backup-01",
		AttemptSnapshot{}, input, candidate,
	)
	if err != nil {
		t.Fatalf("retry Begin() error = %v", err)
	}
	if second.Disposition != AttemptResumedCandidate ||
		second.Attempt.Token != first.Attempt.Token ||
		second.Attempt.Generation != first.Attempt.Generation {
		t.Fatalf("retry Begin() = %#v, want resumed candidate of %#v", second, first)
	}
	if fixture.store.runCount() != 1 {
		t.Fatalf("run count = %d, want 1", fixture.store.runCount())
	}
}

type deletionReplayFixture struct {
	provider *DeletionReplayProvider
	store    *memoryReplayEvidenceStore
	source   *replaySourceStub
	ids      *replayIDStub
	tokens   *replayIDStub
	clock    *replayClockStub
}

func newDeletionReplayFixture(t *testing.T) deletionReplayFixture {
	t.Helper()
	store := newMemoryReplayEvidenceStore()
	source := &replaySourceStub{input: replayFixtureInput()}
	ids := &replayIDStub{}
	tokens := &replayIDStub{}
	clock := &replayClockStub{
		now: time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC),
	}
	provider, err := NewDeletionReplayProvider(DeletionReplayDependencies{
		Store: store, Source: source, EvidenceIDs: ids, Tokens: tokens, Clock: clock,
	})
	if err != nil {
		t.Fatalf("NewDeletionReplayProvider() error = %v", err)
	}
	return deletionReplayFixture{
		provider: provider, store: store, source: source,
		ids: ids, tokens: tokens, clock: clock,
	}
}

type replaySourceStub struct {
	mu          sync.Mutex
	input       DeletionReplayInput
	replayCalls int
	failure     error
}

func (source *replaySourceStub) ReplayOldBackup(
	context.Context,
	string,
) (DeletionReplayInput, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.replayCalls++
	return source.input, source.failure
}

func (source *replaySourceStub) Snapshot(
	context.Context,
	string,
) (DeletionReplayFacts, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.input.DeletionReplayFacts, nil
}

func (source *replaySourceStub) mutate(change func(*DeletionReplayInput)) {
	source.mu.Lock()
	defer source.mu.Unlock()
	change(&source.input)
}

type replayIDStub struct {
	mu    sync.Mutex
	calls int
}

func (generator *replayIDStub) NewID() (string, error) {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	generator.calls++
	return fmt.Sprintf("id-%02d", generator.calls), nil
}

func (generator *replayIDStub) next() string {
	value, _ := generator.NewID()
	return value
}

type replayClockStub struct {
	mu    sync.Mutex
	now   time.Time
	calls int
}

func (clock *replayClockStub) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.calls++
	return clock.now
}

type memoryRun struct {
	run        DeletionReplayRun
	generation int64
	token      string
}

type memoryReplayEvidenceStore struct {
	mu       sync.Mutex
	runs     map[string]memoryRun
	evidence map[string]DeletionReplayEvidence
	byDigest map[string]string
}

func newMemoryReplayEvidenceStore() *memoryReplayEvidenceStore {
	return &memoryReplayEvidenceStore{
		runs:     make(map[string]memoryRun),
		evidence: make(map[string]DeletionReplayEvidence),
		byDigest: make(map[string]string),
	}
}

func (store *memoryReplayEvidenceStore) LoadAttempt(
	_ context.Context,
	runID string,
	backupRef string,
) (AttemptSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.runs[runID]
	if !ok {
		return AttemptSnapshot{}, nil
	}
	if entry.run.BackupRef != backupRef {
		return AttemptSnapshot{}, ErrDeletionReplayBackupMismatch
	}
	return snapshotFromRun(entry), nil
}

func (store *memoryReplayEvidenceStore) Begin(
	_ context.Context, runID, backupRef string, expected AttemptSnapshot,
	fresh DeletionReplayFacts, candidateToken string,
) (BeginResult, error) {
	if _, err := normalizeReplayFacts(fresh); err != nil {
		return BeginResult{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.runs[runID]
	if ok && entry.run.BackupRef != backupRef {
		return BeginResult{}, ErrDeletionReplayBackupMismatch
	}
	if !ok {
		entry = memoryRun{run: DeletionReplayRun{RunID: runID, BackupRef: backupRef, Namespace: ArticleDeletionNamespace, Status: ReplayRunRunning}}
	}
	current := snapshotFromRun(entry)
	if entry.run.Status == ReplayRunReady {
		return store.beginReady(runID, entry, expected, fresh, candidateToken)
	}
	if current.Token != "" && current.Token == candidateToken {
		return BeginResult{Attempt: current, Disposition: AttemptResumedCandidate}, nil
	}
	switch {
	case current.Status == ReplayRunRunning && current.Generation == 0:
		return store.advance(runID, entry, candidateToken, AttemptCreated)
	case current.Status == ReplayRunRunning:
		return BeginResult{Attempt: current, Disposition: AttemptJoinedCurrent}, nil
	case current.Status == ReplayRunFailed && matchesAttemptSnapshot(expected, current):
		return store.advance(runID, entry, candidateToken, AttemptAdvancedFailed)
	default:
		return BeginResult{Attempt: current, Disposition: AttemptSuperseded}, nil
	}
}

func (store *memoryReplayEvidenceStore) beginReady(runID string, entry memoryRun, expected AttemptSnapshot, fresh DeletionReplayFacts, candidate string) (BeginResult, error) {
	current := snapshotFromRun(entry)
	if current.CurrentEvidenceRef == nil {
		return BeginResult{}, ErrDeletionReplayEvidenceBad
	}
	evidence, ok := store.evidence[*current.CurrentEvidenceRef]
	if !ok {
		return BeginResult{}, ErrDeletionReplayEvidenceBad
	}
	err := checkReplayCurrentFacts(runID, entry.run.BackupRef, current, evidence, fresh)
	if err == nil {
		return BeginResult{Attempt: current, Disposition: AttemptAlreadyReady}, nil
	}
	if !errors.Is(err, ErrDeletionReplayInputChanged) || current.Token == candidate {
		return BeginResult{}, err
	}
	if !matchesAttemptSnapshot(expected, current) {
		return BeginResult{Attempt: current, Disposition: AttemptSuperseded}, nil
	}
	return store.advance(runID, entry, candidate, AttemptAdvancedStale)
}

func (store *memoryReplayEvidenceStore) advance(
	runID string,
	entry memoryRun,
	token string,
	disposition AttemptDisposition,
) (BeginResult, error) {
	entry.run.Status = ReplayRunRunning
	entry.run.LastErrorBucket = nil
	entry.generation++
	entry.token = token
	store.runs[runID] = entry
	return BeginResult{Attempt: snapshotFromRun(entry), Disposition: disposition}, nil
}

func (store *memoryReplayEvidenceStore) Fail(
	_ context.Context,
	runID string,
	backupRef string,
	attempt AttemptSnapshot,
	reason string,
) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.runs[runID]
	if !ok {
		return ErrDeletionReplayRunNotFound
	}
	if entry.run.BackupRef != backupRef {
		return ErrDeletionReplayBackupMismatch
	}
	if entry.run.Status == ReplayRunReady {
		return nil
	}
	if entry.run.Status != ReplayRunRunning || entry.generation != attempt.Generation ||
		attempt.Token == "" || entry.token != attempt.Token {
		return ErrDeletionReplayAttemptNotOwner
	}
	entry.run.Status = ReplayRunFailed
	entry.run.LastErrorBucket = &reason
	store.runs[runID] = entry
	return nil
}

func (store *memoryReplayEvidenceStore) Publish(ctx context.Context, request ReplayPublishRequest) (DeletionReplayEvidence, error) {
	fresh, err := request.Refresh(ctx)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	entry, ok := store.runs[request.RunID]
	if !ok {
		return DeletionReplayEvidence{}, ErrDeletionReplayRunNotFound
	}
	if entry.run.BackupRef != request.BackupRef {
		return DeletionReplayEvidence{}, ErrDeletionReplayBackupMismatch
	}
	if !sameReplayOwner(snapshotFromRun(entry), request.Attempt) {
		return DeletionReplayEvidence{}, ErrDeletionReplayAttemptNotOwner
	}
	if entry.run.Status == ReplayRunReady {
		if entry.run.CurrentEvidenceRef == nil {
			return DeletionReplayEvidence{}, ErrDeletionReplayEvidenceBad
		}
		evidence, ok := store.evidence[*entry.run.CurrentEvidenceRef]
		if !ok {
			return DeletionReplayEvidence{}, ErrDeletionReplayEvidenceBad
		}
		return evidence, checkReplayCurrentFacts(request.RunID, request.BackupRef, snapshotFromRun(entry), evidence, fresh)
	}
	if err := validateReplayCandidate(request, fresh); err != nil {
		return DeletionReplayEvidence{}, err
	}
	evidence, err := store.selectEvidence(request, fresh)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	store.setCurrent(&entry, evidence.EvidenceDigest, evidence.EvidenceRef)
	return evidence, nil
}

func (store *memoryReplayEvidenceStore) selectEvidence(request ReplayPublishRequest, fresh DeletionReplayFacts) (DeletionReplayEvidence, error) {
	var selected DeletionReplayEvidence
	for _, evidence := range store.evidence {
		if evidence.RunID != request.RunID || evidence.Namespace != ArticleDeletionNamespace || evidence.BackupRef != request.BackupRef ||
			!sameReplayFacts(replayFactsFromEvidence(evidence), fresh) {
			continue
		}
		if err := selectReplayHistoryEvidence(&selected, evidence); err != nil {
			return DeletionReplayEvidence{}, err
		}
	}
	if selected.EvidenceRef != "" {
		return selected, nil
	}
	evidence, err := newReplayEvidence(request, request.Input)
	if err != nil {
		return DeletionReplayEvidence{}, err
	}
	store.evidence[evidence.EvidenceRef] = evidence
	store.byDigest[evidence.EvidenceDigest] = evidence.EvidenceRef
	return evidence, nil
}

func (store *memoryReplayEvidenceStore) setCurrent(
	entry *memoryRun,
	digest string,
	ref string,
) {
	entry.run.InputDigest = &digest
	entry.run.CurrentEvidenceRef = &ref
	entry.run.Status = ReplayRunReady
	entry.run.LastErrorBucket = nil
	store.runs[entry.run.RunID] = *entry
}

func (store *memoryReplayEvidenceStore) ReadCurrent(
	ctx context.Context, runID, backupRef string, refresh func(context.Context) (DeletionReplayFacts, error),
) (DeletionReplayRun, *DeletionReplayEvidence, error) {
	store.mu.Lock()
	entry, ok := store.runs[runID]
	var evidence DeletionReplayEvidence
	var foundEvidence bool
	if entry.run.CurrentEvidenceRef != nil {
		evidence, foundEvidence = store.evidence[*entry.run.CurrentEvidenceRef]
	}
	store.mu.Unlock()
	if !ok {
		return DeletionReplayRun{}, nil, ErrDeletionReplayRunNotFound
	}
	if entry.run.BackupRef != backupRef {
		return entry.run, nil, ErrDeletionReplayBackupMismatch
	}
	if entry.run.Status != ReplayRunReady {
		return entry.run, nil, nil
	}
	if !foundEvidence {
		return entry.run, nil, ErrDeletionReplayEvidenceBad
	}
	current := snapshotFromRun(entry)
	if err := validateReplayEvidenceBinding(runID, backupRef, current, evidence); err != nil {
		return entry.run, nil, err
	}
	fresh, err := refresh(ctx)
	if err != nil {
		return entry.run, nil, err
	}
	if err := checkReplayCurrentFacts(runID, backupRef, current, evidence, fresh); err != nil {
		return entry.run, nil, err
	}
	store.mu.Lock()
	after := snapshotFromRun(store.runs[runID])
	store.mu.Unlock()
	if !matchesAttemptSnapshot(current, after) {
		return entry.run, nil, ErrDeletionReplayInputChanged
	}
	return entry.run, &evidence, nil
}

func (store *memoryReplayEvidenceStore) evidenceCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.evidence)
}

func (store *memoryReplayEvidenceStore) runCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.runs)
}

func snapshotFromRun(entry memoryRun) AttemptSnapshot {
	return AttemptSnapshot{
		Exists:             true,
		Status:             entry.run.Status,
		Generation:         entry.generation,
		Token:              entry.token,
		BackupRef:          entry.run.BackupRef,
		InputDigest:        entry.run.InputDigest,
		CurrentEvidenceRef: entry.run.CurrentEvidenceRef,
		LastErrorBucket:    entry.run.LastErrorBucket,
	}
}
func (store *memoryReplayEvidenceStore) beginFresh(
	t *testing.T,
	runID string,
	backupRef string,
	token string,
) AttemptSnapshot {
	t.Helper()
	input := replayFixtureInput()
	result, err := store.Begin(
		context.Background(), runID, backupRef, AttemptSnapshot{}, input.DeletionReplayFacts, token,
	)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	return result.Attempt
}
func decodeReplayStatus(t *testing.T, raw []byte) recovery.ReplayStatusV1 {
	t.Helper()
	status, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil {
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		t.Fatalf("DecodeReplayStatusV1(%#v) error = %v", body, err)
	}
	return status
}

var _ ReplayEvidenceStore = (*memoryReplayEvidenceStore)(nil)

func TestDeletionReplay_ReusesPublishedOutcomeAfterExecutionWouldChange(t *testing.T) {
	f := newDeletionReplayFixture(t)
	first, err := f.provider.ReplayAndVerify(context.Background(), "run-outcome", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	f.source.mutate(func(input *DeletionReplayInput) { input.TombstonesReplayed, input.AlreadyAbsent = 0, 3 })
	read, err := f.provider.ReadStatus(context.Background(), "run-outcome", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.provider.ReplayAndVerify(context.Background(), "run-outcome", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	assertSameReadyStatus(t, first, read)
	assertSameReadyStatus(t, first, again)
	if f.source.replayCalls != 1 || f.ids.calls != 1 || f.clock.calls != 1 {
		t.Fatal("persisted Ready caused extra execution or metadata")
	}
}

type replayAckStore struct {
	ReplayEvidenceStore
	mode                                string
	events                              []string
	beginCalls, publishCalls, failCalls int
}

func (s *replayAckStore) ReadCurrent(ctx context.Context, run, backup string, refresh func(context.Context) (DeletionReplayFacts, error)) (DeletionReplayRun, *DeletionReplayEvidence, error) {
	s.events = append(s.events, "ReadCurrent")
	return s.ReplayEvidenceStore.ReadCurrent(ctx, run, backup, refresh)
}

func (s *replayAckStore) LoadAttempt(ctx context.Context, run, backup string) (AttemptSnapshot, error) {
	s.events = append(s.events, "LoadAttempt")
	return s.ReplayEvidenceStore.LoadAttempt(ctx, run, backup)
}

func (s *replayAckStore) Begin(ctx context.Context, run, backup string, expected AttemptSnapshot, fresh DeletionReplayFacts, candidate string) (BeginResult, error) {
	s.beginCalls++
	s.events = append(s.events, "Begin")
	if s.mode == "RejectBegin" {
		return BeginResult{}, ErrDeletionReplayEvidenceBad
	}
	result, err := s.ReplayEvidenceStore.Begin(ctx, run, backup, expected, fresh, candidate)
	if err == nil && s.mode == "Begin" && s.beginCalls == 1 {
		return BeginResult{}, errors.New("controlled Begin acknowledgement loss")
	}
	return result, err
}

func (s *replayAckStore) Publish(ctx context.Context, request ReplayPublishRequest) (DeletionReplayEvidence, error) {
	s.publishCalls++
	s.events = append(s.events, "Publish")
	if s.mode == "UnreadablePublish" {
		return DeletionReplayEvidence{}, nil
	}
	result, err := s.ReplayEvidenceStore.Publish(ctx, request)
	if err == nil && s.mode == "Publish" && s.publishCalls == 1 {
		return DeletionReplayEvidence{}, errors.New("controlled Publish acknowledgement loss")
	}
	return result, err
}

func (s *replayAckStore) Fail(ctx context.Context, run, backup string, attempt AttemptSnapshot, reason string) error {
	s.failCalls++
	s.events = append(s.events, "Fail")
	err := s.ReplayEvidenceStore.Fail(ctx, run, backup, attempt, reason)
	if err == nil && s.mode == "Fail" && s.failCalls == 1 {
		return errors.New("controlled Fail acknowledgement loss")
	}
	return err
}

func TestDeletionReplay_UnknownCommitReadsCurrentBeforeObservingOrWriting(t *testing.T) {
	for _, mode := range []string{"Begin", "Publish", "Fail"} {
		t.Run(mode, func(t *testing.T) { testReplayUnknownAck(t, mode) })
	}
}

func testReplayUnknownAck(t *testing.T, mode string) {
	f := newDeletionReplayFixture(t)
	spy := &replayAckStore{ReplayEvidenceStore: f.store, mode: mode}
	f.provider.dependencies.Store = spy
	if mode == "Fail" {
		f.source.failure = errors.New("controlled source failure")
	}
	raw, err := f.provider.ReplayAndVerify(context.Background(), "run-ack", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	status := decodeReplayStatus(t, raw)
	if mode == "Fail" {
		if status.Unavailable == nil || status.Unavailable.ReasonCode != replayReasonVerificationFailed || spy.failCalls != 1 || f.store.evidenceCount() != 0 {
			t.Fatal("committed Fail did not converge without another write")
		}
	} else {
		if status.Ready == nil || f.store.evidenceCount() != 1 || spy.failCalls != 0 || f.ids.calls != 1 || f.clock.calls != 1 {
			t.Fatal("committed Begin/Publish did not converge to one evidence")
		}
	}
	if f.source.replayCalls != 1 || f.tokens.calls != 1 {
		t.Fatal("unknown commit minted another candidate or re-executed source")
	}
	for index, event := range spy.events {
		if event == mode {
			if index+1 >= len(spy.events) || spy.events[index+1] != "ReadCurrent" {
				t.Fatalf("unknown %s did not read current first: %v", mode, spy.events)
			}
			break
		}
	}
}

func TestDeletionReplay_ObservationDoesNotAuthorizeFail(t *testing.T) {
	f := newDeletionReplayFixture(t)
	owner := f.store.beginFresh(t, "run-observed", "backup-01", "existing-owner")
	spy := &replayAckStore{ReplayEvidenceStore: f.store, mode: "RejectBegin"}
	f.provider.dependencies.Store = spy
	raw, err := f.provider.ReplayAndVerify(context.Background(), "run-observed", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	status := decodeReplayStatus(t, raw)
	after, err := f.store.LoadAttempt(context.Background(), "run-observed", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	if spy.failCalls != 0 || f.source.replayCalls != 0 || !matchesAttemptSnapshot(owner, after) ||
		status.Unavailable == nil || status.Unavailable.ReasonCode != replayReasonEvidenceUnreadable {
		t.Fatal("mere observation became write authority")
	}
}

func TestDeletionReplay_UnreadablePublishReturnsUnavailable(t *testing.T) {
	f := newDeletionReplayFixture(t)
	spy := &replayAckStore{ReplayEvidenceStore: f.store, mode: "UnreadablePublish"}
	f.provider.dependencies.Store = spy
	raw, err := f.provider.ReplayAndVerify(context.Background(), "run-bad-publish", "backup-01")
	if err != nil {
		t.Fatalf("invalid port result escaped the public variant: %v", err)
	}
	status := decodeReplayStatus(t, raw)
	if status.Unavailable == nil || status.Ready != nil || status.Unavailable.ReasonCode != replayReasonEvidenceUnreadable ||
		spy.failCalls != 1 || f.store.evidenceCount() != 0 {
		t.Fatal("unreadable uncommitted result did not fail closed")
	}
}

type publicationUnreadableAckStore struct {
	ReplayEvidenceStore
	cause            error
	readbacks, fails int
}

func (s *publicationUnreadableAckStore) Publish(ctx context.Context, q ReplayPublishRequest) (DeletionReplayEvidence, error) {
	evidence, err := s.ReplayEvidenceStore.Publish(ctx, q)
	if err != nil {
		return evidence, err
	}
	return evidence, errReplayPublicationUnknown
}
func (s *publicationUnreadableAckStore) readPublicationTriplet(context.Context, string, string) (*DeletionReplayEvidence, error) {
	s.readbacks++
	return nil, s.cause
}
func (s *publicationUnreadableAckStore) Fail(ctx context.Context, run, backup string, a AttemptSnapshot, reason string) error {
	s.fails++
	return s.ReplayEvidenceStore.Fail(ctx, run, backup, a, reason)
}
func TestDeletionReplay_PublicationUnknownReadIsNotAbsence(t *testing.T) {
	f := newDeletionReplayFixture(t)
	store := &publicationUnreadableAckStore{ReplayEvidenceStore: f.store, cause: errors.New("independent receipt read unavailable")}
	f.provider.dependencies.Store = store
	raw, err := f.provider.ReplayAndVerify(context.Background(), "run-unknown-publication", "backup-01")
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeReplayStatus(t, raw)
	if decoded.Ready != nil || decoded.Unavailable == nil || store.readbacks != 1 || store.fails != 0 || f.source.replayCalls != 1 || f.tokens.calls != 1 {
		t.Fatal("unknown triple read became Ready/absence/replay/Fail")
	}
	if f.store.evidenceCount() != 1 {
		t.Fatal("committed immutable evidence was lost")
	}
}

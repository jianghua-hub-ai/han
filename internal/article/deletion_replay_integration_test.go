//go:build integration

package article

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDeletionReplayPostgres exercises the generation/token CAS, UTC microsecond
// stability, input drift and commit-outcome-unknown retries against real
// PostgreSQL with multiple connections and transactions (A-REC-04/06/07).
func TestDeletionReplayPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("A-REC-04 stable ref digest and microsecond time", func(t *testing.T) {
		fixture := newReplayIntegrationFixture(t, newReplayIntegrationPool(t, ctx), ctx)
		ctx := fixture.ctx
		input := replayFixtureInput()
		input.BackupCutoff = input.BackupCutoff.Add(999 * time.Nanosecond)

		first := fixture.replayAndVerify(t, ctx, input)
		second := fixture.replayAndVerify(t, ctx, input)
		assertSameReadyStatus(t, first, second)
		ready := decodeReplayStatus(t, first).Ready
		if ready == nil {
			t.Fatalf("first replay did not reach Ready: %s", first)
		}
		if _, err := time.Parse(time.RFC3339, ready.CompletedAt); err != nil {
			t.Fatalf("completed_at = %q is not second-precision RFC3339", ready.CompletedAt)
		}
		if fixture.countEvidence(t, ctx) != 1 {
			t.Fatalf("evidence rows = %d, want 1", fixture.countEvidence(t, ctx))
		}
	})

	t.Run("A-REC-06 concurrent same backup converges and stale fail cannot override", func(t *testing.T) {
		fixture := newReplayIntegrationFixture(t, newReplayIntegrationPool(t, ctx), ctx)
		ctx := fixture.ctx
		input := replayFixtureInput()

		const workers = 8
		var wait sync.WaitGroup
		results := make(chan []byte, workers)
		for range workers {
			wait.Add(1)
			go func() {
				defer wait.Done()
				raw, err := fixture.provider.ReplayAndVerify(ctx, fixture.runID, fixture.backupRef)
				if err != nil {
					t.Errorf("ReplayAndVerify() error = %v", err)
					return
				}
				results <- raw
			}()
		}
		wait.Wait()
		close(results)
		var first []byte
		for raw := range results {
			if first == nil {
				first = raw
				continue
			}
			assertSameReadyStatus(t, first, raw)
		}
		if first == nil {
			t.Fatal("no concurrent replay result")
		}
		if fixture.countEvidence(t, ctx) != 1 {
			t.Fatalf("evidence rows = %d, want 1", fixture.countEvidence(t, ctx))
		}
		current, err := fixture.store.LoadAttempt(ctx, fixture.runID, fixture.backupRef)
		if err != nil {
			t.Fatalf("LoadAttempt() error = %v", err)
		}
		if current.Status != ReplayRunReady {
			t.Fatalf("status = %q, want ready", current.Status)
		}
		// A stale handle (generation-1) must not Fail the current Ready attempt.
		if err := fixture.store.Fail(ctx, fixture.runID, fixture.backupRef,
			AttemptSnapshot{Exists: true, Generation: current.Generation - 1, Token: current.Token},
			replayReasonVerificationFailed,
		); err != nil {
			t.Fatalf("stale Fail() error = %v, want no-op", err)
		}
		after, err := fixture.store.LoadAttempt(ctx, fixture.runID, fixture.backupRef)
		if err != nil {
			t.Fatalf("LoadAttempt() error = %v", err)
		}
		if after.Status != ReplayRunReady || after.Generation != current.Generation ||
			after.Token != current.Token {
			t.Fatalf("stale Fail changed the attempt: %#v -> %#v", current, after)
		}
		_ = input
	})

	t.Run("A-REC-07 begin commit unknown rejoins same candidate", func(t *testing.T) {
		fixture := newReplayIntegrationFixture(t, newReplayIntegrationPool(t, ctx), ctx)
		ctx := fixture.ctx
		input := replayFixtureInput()
		candidate := "candidate-token-fixed"
		first, err := fixture.store.Begin(
			ctx, fixture.runID, fixture.backupRef, AttemptSnapshot{}, input.DeletionReplayFacts, candidate,
		)
		if err != nil {
			t.Fatalf("Begin() error = %v", err)
		}
		retry, err := fixture.store.Begin(
			ctx, fixture.runID, fixture.backupRef, AttemptSnapshot{}, input.DeletionReplayFacts, candidate,
		)
		if err != nil {
			t.Fatalf("retry Begin() error = %v", err)
		}
		if retry.Disposition != AttemptResumedCandidate ||
			retry.Attempt.Token != first.Attempt.Token ||
			retry.Attempt.Generation != first.Attempt.Generation {
			t.Fatalf("retry Begin() = %#v, want resumed candidate %#v", retry, first)
		}
		var runCount int
		if err := fixture.store.pool.QueryRow(ctx,
			`SELECT count(*) FROM deletion_replay_runs`,
		).Scan(&runCount); err != nil {
			t.Fatalf("count runs: %v", err)
		}
		if runCount != 1 {
			t.Fatalf("run rows = %d, want 1", runCount)
		}
	})

	t.Run("A-REC-07 join existing running returns authoritative handle", func(t *testing.T) {
		fixture := newReplayIntegrationFixture(t, newReplayIntegrationPool(t, ctx), ctx)
		ctx := fixture.ctx
		input := replayFixtureInput()
		owner, err := fixture.store.Begin(
			ctx, fixture.runID, fixture.backupRef, AttemptSnapshot{}, input.DeletionReplayFacts, "owner-token",
		)
		if err != nil {
			t.Fatalf("owner Begin() error = %v", err)
		}
		joiner, err := fixture.store.Begin(
			ctx, fixture.runID, fixture.backupRef, AttemptSnapshot{}, input.DeletionReplayFacts, "joiner-token",
		)
		if err != nil {
			t.Fatalf("joiner Begin() error = %v", err)
		}
		if joiner.Disposition != AttemptJoinedCurrent ||
			joiner.Attempt.Token != owner.Attempt.Token {
			t.Fatalf("joiner Begin() = %#v, want joined current of %#v", joiner, owner)
		}
		// The candidate-only joiner token must not Fail the owner.
		if err := fixture.store.Fail(ctx, fixture.runID, fixture.backupRef,
			AttemptSnapshot{Exists: true, Generation: owner.Attempt.Generation, Token: "joiner-token"},
			replayReasonVerificationFailed,
		); !errors.Is(err, ErrDeletionReplayAttemptNotOwner) {
			t.Fatalf("candidate-only Fail() error = %v, want not-owner", err)
		}
	})

	t.Run("A-REC-07 stale ready advances generation and republishes", func(t *testing.T) {
		fixture := newReplayIntegrationFixture(t, newReplayIntegrationPool(t, ctx), ctx)
		ctx := fixture.ctx
		first := replayFixtureInput()
		firstRaw := fixture.replayAndVerify(t, ctx, first)
		drifted := replayFixtureInput()
		drifted.InventoryDigest = "sha256:feedface"
		secondRaw := fixture.replayAndVerify(t, ctx, drifted)
		firstReady := decodeReplayStatus(t, firstRaw).Ready
		secondReady := decodeReplayStatus(t, secondRaw).Ready
		if firstReady == nil || secondReady == nil {
			t.Fatal("both replays should reach Ready")
		}
		if firstReady.EvidenceRef == secondReady.EvidenceRef {
			t.Fatal("drift must publish new evidence")
		}
		ready := secondReady
		if ready == nil {
			t.Fatal("post-drift replay did not reach Ready")
		}
		if fixture.countEvidence(t, ctx) != 2 {
			t.Fatalf("evidence rows = %d, want 2", fixture.countEvidence(t, ctx))
		}
	})

	// A-REC-05: drive the real FileBackupInventoryReader through the unique
	// production source against real PostgreSQL, atomically replacing the
	// inventory between reads. The stale current must fail closed to
	// unknown/evidence_stale, a subsequent Replay must install a new unique
	// current, and the next read must return the new Ready.
	t.Run("A-REC-05 atomic inventory drift returns stale then republishes", func(t *testing.T) {
		fixture := newInventoryDriftFixture(t, newReplayIntegrationPool(t, ctx))
		ctx := fixture.ctx
		firstRaw := fixture.replayAndVerify(t, ctx)
		firstReady := decodeReplayStatus(t, firstRaw).Ready
		if firstReady == nil {
			t.Fatalf("first replay did not reach Ready: %s", firstRaw)
		}

		fixture.replaceInventory(t, "2026-10-20T00:00:02.000000Z")
		staleRaw, err := fixture.provider.ReadStatus(ctx, fixture.runID, fixture.backupRef)
		if err != nil {
			t.Fatalf("ReadStatus() after drift error = %v", err)
		}
		stale := decodeReplayStatus(t, staleRaw)
		if stale.Unavailable == nil ||
			stale.Unavailable.Status != replayStatusUnknown ||
			stale.Unavailable.ReasonCode != replayReasonEvidenceStale {
			t.Fatalf("post-drift status = %#v, want unknown/evidence_stale", stale)
		}

		secondRaw := fixture.replayAndVerify(t, ctx)
		secondReady := decodeReplayStatus(t, secondRaw).Ready
		if secondReady == nil {
			t.Fatalf("post-drift replay did not reach Ready: %s", secondRaw)
		}
		if firstReady.EvidenceRef == secondReady.EvidenceRef ||
			firstReady.EvidenceDigest == secondReady.EvidenceDigest {
			t.Fatal("drift must publish new immutable evidence, not reuse the stale ref/digest")
		}
		if fixture.countEvidence(t, ctx) != 2 {
			t.Fatalf("evidence rows = %d, want 2", fixture.countEvidence(t, ctx))
		}
		currentRaw, err := fixture.provider.ReadStatus(ctx, fixture.runID, fixture.backupRef)
		if err != nil {
			t.Fatalf("ReadStatus() after republish error = %v", err)
		}
		current := decodeReplayStatus(t, currentRaw).Ready
		if current == nil || current.EvidenceRef != secondReady.EvidenceRef {
			t.Fatalf("current = %#v, want the post-drift evidence %s", current, secondReady.EvidenceRef)
		}
	})
}

type replayIntegrationFixture struct {
	store     *PostgresReplayEvidenceStore
	provider  *DeletionReplayProvider
	source    *ProductionDeletionReplaySource
	ctx       context.Context
	actual    *publicationPGFixture
	runID     string
	backupRef string
}

// inventoryDriftFixture wires the real FileBackupInventoryReader and the unique
// ProductionDeletionReplaySource against real PostgreSQL so A-REC-05 exercises
// the production read path rather than a fixed input stub.
type inventoryDriftFixture struct {
	store     *PostgresReplayEvidenceStore
	provider  *DeletionReplayProvider
	runID     string
	backupRef string
	path      string
	ctx       context.Context
}

func newInventoryDriftFixture(t *testing.T, pool *pgxpool.Pool) inventoryDriftFixture {
	t.Helper()
	parent, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	f := newPublicationPGFixtureOnPool(t, pool, parent)
	return inventoryDriftFixture{store: f.store, provider: f.provider, runID: f.run, backupRef: f.backup, path: f.path, ctx: f.ctx}
}

func (fixture inventoryDriftFixture) replayAndVerify(t *testing.T, ctx context.Context) []byte {
	t.Helper()
	raw, err := fixture.provider.ReplayAndVerify(ctx, fixture.runID, fixture.backupRef)
	if err != nil {
		t.Fatalf("ReplayAndVerify() error = %v", err)
	}
	return raw
}

// replaceInventory atomically swaps the on-disk manifest via rename so the real
// reader's lstat/open/fstat/rename checks observe a genuine replacement.
func (fixture inventoryDriftFixture) replaceInventory(t *testing.T, cutoff string) {
	t.Helper()
	replacement := filepath.Join(filepath.Dir(fixture.path), "backup-inventory.next.json")
	writeInventory(t, replacement, inventoryManifestWithCutoff(t, fixture.backupRef, cutoff))
	if err := os.Rename(replacement, fixture.path); err != nil {
		t.Fatalf("atomically replace inventory: %v", err)
	}
}

func (fixture inventoryDriftFixture) countEvidence(t *testing.T, ctx context.Context) int {
	t.Helper()
	var count int
	if err := fixture.store.pool.QueryRow(
		ctx, `SELECT count(*) FROM deletion_replay_evidence`,
	).Scan(&count); err != nil {
		t.Fatalf("count evidence: %v", err)
	}
	return count
}

// driftVerifiedJournal returns one verified tombstone so the replay snapshot has
// a non-empty journal head (matching the deletion_replay_evidence schema) while
// remaining fully deterministic.
type driftVerifiedJournal struct{}

func (*driftVerifiedJournal) ReadAllVerified(
	context.Context,
) ([]ArticleDeletionTombstone, error) {
	return []ArticleDeletionTombstone{{
		Namespace:      ArticleDeletionNamespace,
		Sequence:       1,
		ArticleID:      "article-drift",
		ArticleVersion: 1,
		DeletedAt:      time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		KeyID:          "article-deletion-hmac",
		KeyVersion:     "v1",
		Stage:          "deleted",
	}}, nil
}

// driftReplayer reports the single tombstone as replayed so the evidence
// tombstones_total = tombstones_replayed + already_absent invariant holds.
type driftReplayer struct{}

func (*driftReplayer) ReplayAllTombstones(
	context.Context, []ArticleDeletionTombstone, string,
) (DeletionReplayAllResult, error) {
	return DeletionReplayAllResult{Replayed: 1, AlreadyAbsent: 0}, nil
}

// inventoryManifestWithCutoff builds a canonical manifest whose matching entry
// completion equals the requested cutoff.
func inventoryManifestWithCutoff(t *testing.T, backupRef string, cutoff string) string {
	t.Helper()
	manifest := map[string]any{
		"inventory_version": BackupInventoryV1Version,
		"inventory_scope":   BackupInventoryScopeV1,
		"backup_ref":        backupRef,
		"backup_cutoff":     cutoff,
		"generated_at":      mustInventoryTime(t, cutoff).Add(time.Second).Format("2006-01-02T15:04:05.000000Z"),
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

func newReplayIntegrationFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) replayIntegrationFixture {
	t.Helper()
	f := newPublicationPGFixtureOnPool(t, pool, ctx)
	return replayIntegrationFixture{store: f.store, provider: f.provider, source: f.source, runID: f.run, backupRef: f.backup, ctx: f.ctx, actual: f}
}

func (fixture replayIntegrationFixture) replayAndVerify(t *testing.T, ctx context.Context, input DeletionReplayInput) []byte {
	t.Helper()
	if input.InventoryDigest != replayFixtureInput().InventoryDigest {
		// Preserve the old drift oracle using a genuine canonical inventory replacement.
		writeInventory(t, fixture.actual.path, inventoryManifestWithCutoff(t, fixture.backupRef, "2026-10-20T00:00:02.000000Z"))
	}
	if input.BackupCutoff.Nanosecond()%1000 != 0 {
		return publicationNoisyCandidate(t, fixture.actual, input.BackupCutoff.Nanosecond()%1000)
	}
	raw, err := fixture.provider.ReplayAndVerify(ctx, fixture.runID, fixture.backupRef)
	pubMust(t, err)
	return raw
}

func (fixture replayIntegrationFixture) countEvidence(t *testing.T, ctx context.Context) int {
	t.Helper()
	var count int
	if err := fixture.store.pool.QueryRow(
		ctx, `SELECT count(*) FROM deletion_replay_evidence`,
	).Scan(&count); err != nil {
		t.Fatalf("count evidence: %v", err)
	}
	return count
}

type integrationReplaySource struct {
	input DeletionReplayInput
}

func (source *integrationReplaySource) ReplayOldBackup(
	context.Context, string,
) (DeletionReplayInput, error) {
	return source.input, nil
}

func (source *integrationReplaySource) Snapshot(
	context.Context, string,
) (DeletionReplayFacts, error) {
	return source.input.DeletionReplayFacts, nil
}

type integrationClock struct{}

func (integrationClock) Now() time.Time {
	return time.Date(2026, 10, 20, 1, 0, 0, 250500000, time.UTC)
}

func newReplayIntegrationPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	return newReplayPublicationPool(t, ctx)
}

func randomIntegrationHex(t *testing.T, size int) string {
	t.Helper()
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("generate random hex: %v", err)
	}
	return hex.EncodeToString(buffer)
}

var _ = strings.TrimSpace

// This keeps the old nanosecond-input Store oracle on real physical replay;
// only the candidate cutoff receives the deliberate noise, never the facts source.
func publicationNoisyCandidate(t *testing.T, f *publicationPGFixture, noise int) []byte {
	t.Helper()
	raw, err := f.provider.ReadStatus(f.ctx, f.run, f.backup)
	pubMust(t, err)
	status, err := recovery.DecodeReplayStatusV1(raw)
	pubMust(t, err)
	if status.Ready != nil {
		return raw
	}
	request, _, _ := publicationPrepareRequest(t, f)
	request.Input.BackupCutoff = request.Input.BackupCutoff.Add(time.Duration(noise))
	evidence, err := f.store.Publish(f.ctx, request)
	pubMust(t, err)
	raw, err = encodeReadyReplayStatus(evidence)
	pubMust(t, err)
	return raw
}

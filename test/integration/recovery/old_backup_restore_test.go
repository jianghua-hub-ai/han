//go:build integration

package recovery_test

// TestOldBackupRestore is the article-management-S4.4 third-path, test-only
// addition. Every manifest, inventory, keyring material, HMAC vector, Journal
// snapshot, publication ID and runtime binding used here is synthetic data.
//
// This test does NOT prove that a real old backup can be restored, that real
// POSIX Journal fsync/flock behavior holds, that a real keyring/HMAC runtime
// is wired, that identity-side dual namespace/MarkConsumed/joint stream
// behavior works, or any RPO/RTO target. It retains every schema it creates
// and never drops or deletes schemas, roles or files.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"han/internal/article"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

const (
	s44OldBackupInstallation = "synthetic-old-backup-installation"
	s44OldBackupJournalID    = "synthetic-old-backup-journal"
	s44OldBackupRevision     = "synthetic-old-backup-revision-0001"
)

func TestOldBackupRestore(t *testing.T) {
	t.Run("SyntheticManifestIsNotRealBackup", s44OldBackupSyntheticManifest)
	t.Run("RestoreAndReplayThreeTombstones", s44OldBackupRestoreAndReplay)
	t.Run("ReadyDriftInvalidatesEvidence", s44OldBackupReadyDrift)
	t.Run("InventoryUnavailable", s44OldBackupInventoryUnavailable)
}

func s44OldBackupSyntheticManifest(t *testing.T) {
	manifest := readFixture[backupManifest](t, "backup-manifest.json")
	keyring := readFixture[keyringDocument](t, "keyring-fixtures.json")

	if !manifest.SyntheticOnly || manifest.ProductionUse != "forbidden" {
		t.Fatalf("manifest guards = synthetic_only:%t production_use:%q", manifest.SyntheticOnly, manifest.ProductionUse)
	}
	if manifest.Backup.Artifact.Status != "not_included" {
		t.Fatalf("backup artifact status = %q, want not_included", manifest.Backup.Artifact.Status)
	}
	if manifest.EntryGate.RealExerciseAllowed || manifest.EntryGate.UnknownPolicy != "block" {
		t.Fatalf("entry gate = %#v, want real exercise blocked", manifest.EntryGate)
	}
	if manifest.Backup.BackupRef != manifest.Run.BackupRef {
		t.Fatalf("backup.backup_ref=%q run.backup_ref=%q", manifest.Backup.BackupRef, manifest.Run.BackupRef)
	}
	if manifest.KeyringFixture.ActiveKeyID != "recovery-fixture" || manifest.KeyringFixture.ActiveKeyVersion != "v3" {
		t.Fatalf("active key binding = %s/%s, want recovery-fixture/v3",
			manifest.KeyringFixture.ActiveKeyID, manifest.KeyringFixture.ActiveKeyVersion)
	}

	var active fixtureKey
	var vector testVector
	keyFound, vectorFound := false, false
	for _, fixture := range keyring.Fixtures {
		if fixture.Name != manifest.KeyringFixture.Name {
			continue
		}
		for _, candidate := range fixture.Keys {
			if candidate.ID == manifest.KeyringFixture.ActiveKeyID &&
				candidate.Version == manifest.KeyringFixture.ActiveKeyVersion && candidate.State == "active" {
				active, keyFound = candidate, true
			}
		}
		for _, candidate := range fixture.TestVectors {
			if candidate.Name == manifest.KeyringFixture.TestVector {
				vector, vectorFound = candidate, true
			}
		}
	}
	if !keyFound || !vectorFound {
		t.Fatal("active key or test vector named by the manifest is absent from the keyring fixture")
	}

	mac := hmac.New(sha256.New, []byte(active.Material))
	_, _ = mac.Write([]byte(vector.SyntheticRecoveryInput))
	actual := hex.EncodeToString(mac.Sum(nil))
	if !strings.HasPrefix(manifest.Run.JournalSecretFingerprint, "hmac-sha256:") {
		t.Fatalf("journal fingerprint lacks hmac-sha256 prefix: %q", manifest.Run.JournalSecretFingerprint)
	}
	want := strings.TrimPrefix(manifest.Run.JournalSecretFingerprint, "hmac-sha256:")
	if actual != want {
		t.Fatalf("synthetic HMAC fingerprint = %s, want %s", actual, want)
	}

	t.Log("synthetic manifest only: this fixture must never be used for real recovery")
	t.Log("not verified: real old backup restore, real POSIX Journal/fsync/flock, real keyring/HMAC runtime, identity dual namespace/MarkConsumed/joint stream, RPO/RTO")
}

func s44OldBackupRestoreAndReplay(t *testing.T) {
	f := s44Prepared(t, 0)
	s44OldBackupConfigurePublication(t, f)
	for _, target := range f.targets {
		s44Must(t, f.finalizer.Finalize(f.ctx, target.operation, s44Trace))
	}
	s44OldBackupRequireCompleted(t, f)

	input, err := f.source.ReplayOldBackup(f.ctx, s44Backup)
	s44Must(t, err)
	if input.TombstonesReplayed != 0 || input.AlreadyAbsent != 3 || f.probe.cause != nil {
		t.Fatalf("real ReplayOldBackup = %d/%d cause=%v, want 0/3 without error",
			input.TombstonesReplayed, input.AlreadyAbsent, f.probe.cause)
	}
	ready := s44ProviderReady(f, 0, 3)

	replays, evidenceCount, image := f.probe.calls, s44EvidenceCount(f), s44Image(f)
	firstRaw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	secondRaw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	first, second := s44OldBackupDecodeReady(t, firstRaw), s44OldBackupDecodeReady(t, secondRaw)
	if !bytes.Equal(firstRaw, secondRaw) || first != second || first != ready {
		t.Fatal("consecutive ReadStatus calls did not return the identical Ready")
	}
	if f.probe.calls != replays || s44EvidenceCount(f) != evidenceCount || s44Image(f) != image {
		t.Fatal("ReadStatus rewrote evidence, replayed tombstones or changed the business image")
	}
	t.Logf("restore_and_replay schema=%s replayed=%d already_absent=%d evidence=%d probes=%d",
		f.schema, ready.TombstonesReplayed, ready.AlreadyAbsent, evidenceCount, replays)
}

func s44OldBackupRequireCompleted(t *testing.T, f *s44Fixture) {
	t.Helper()
	tx, err := f.pool.Begin(f.ctx)
	s44Must(t, err)
	defer s44Rollback(t, tx)
	repository := article.Repository{}
	for _, target := range f.targets {
		var present int
		s44Must(t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM articles WHERE id=$1", target.row.ID).Scan(&present))
		if present != 0 {
			t.Fatalf("article %s is still readable", target.row.ID)
		}
		state, readErr := repository.LocateDispositionArticle(f.ctx, tx, target.row.ID)
		if !errors.Is(readErr, pgx.ErrNoRows) || state.Exists {
			t.Fatalf("Repository read of %s = exists:%t err:%v, want absent", target.row.ID, state.Exists, readErr)
		}
		var status string
		s44Must(t, f.observer.QueryRow(f.ctx, "SELECT status FROM deletion_operations WHERE id=$1", target.operation).Scan(&status))
		if status != "completed" {
			t.Fatalf("deletion operation %s status = %q, want completed", target.operation, status)
		}
		var events int
		s44Must(t, f.observer.QueryRow(f.ctx, `SELECT count(*) FROM audit_events WHERE id=$1 AND target_type='article'
			AND target_id=$2 AND outcome='succeeded'`, target.audit, target.row.ID).Scan(&events))
		if events != 1 {
			t.Fatalf("audit event %s for article %s count = %d, want 1", target.audit, target.row.ID, events)
		}
	}
}

func s44OldBackupReadyDrift(t *testing.T) {
	f := s44Prepared(t, 3)
	s44OldBackupConfigurePublication(t, f)
	first := s44ProviderReady(f, 0, 3)
	history := s44EvidenceImage(f)
	s44CommitDrift(f, true)

	stale, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	s44Unavailable(t, stale, "unknown", "evidence_stale")

	second := s44ProviderReady(f, 0, 3)
	if second.EvidenceRef == first.EvidenceRef || second.EvidenceDigest == first.EvidenceDigest {
		t.Fatal("drift republish reused the stale evidence ref or digest")
	}
	if count := s44EvidenceCount(f); count != 2 {
		t.Fatalf("evidence rows = %d, want 2", count)
	}
	s44RequireHistoryPreserved(f, history)

	image := s44Image(f)
	firstRaw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	secondRaw, err := f.provider.ReadStatus(f.ctx, s44Run, s44Backup)
	s44Must(t, err)
	if !bytes.Equal(firstRaw, secondRaw) || s44OldBackupDecodeReady(t, firstRaw) != second || s44Image(f) != image {
		t.Fatal("republished Ready was not stable across consecutive ReadStatus calls")
	}
	t.Logf("ready_drift schema=%s first_ref=%s second_ref=%s evidence=2 stale=unknown/evidence_stale",
		f.schema, first.EvidenceRef, second.EvidenceRef)
}

func s44OldBackupInventoryUnavailable(t *testing.T) {
	f := s44Prepared(t, 0)
	missing := filepath.Join(f.dir, "synthetic-inventory-missing.json")
	reader, err := article.NewFileBackupInventoryReader(missing)
	s44Must(t, err)
	facts, err := article.NewPostgresReplayFactReader(f.pool)
	s44Must(t, err)
	source, err := article.NewProductionDeletionReplaySource(f.journal, reader, f.probe, facts, facts, f.clock.Now)
	s44Must(t, err)

	before := s44Image(f)
	_, replayErr := source.ReplayOldBackup(f.ctx, s44Backup)
	if !errors.Is(replayErr, article.ErrBackupInventoryUnavailable) {
		t.Fatalf("ReplayOldBackup error = %v, want ErrBackupInventoryUnavailable", replayErr)
	}
	var articles, operations, journaled, audits, runs, evidence int
	s44Must(t, f.observer.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM articles),
		(SELECT count(*) FROM deletion_operations),
		(SELECT count(*) FROM deletion_operations WHERE status='journaled'),
		(SELECT count(*) FROM audit_events),
		(SELECT count(*) FROM deletion_replay_runs),
		(SELECT count(*) FROM deletion_replay_evidence)`).Scan(
		&articles, &operations, &journaled, &audits, &runs, &evidence))
	if f.probe.calls != 0 || f.journal.reads != 0 || f.journal.finds != 0 {
		t.Fatalf("unavailable inventory reached replay/journal: probes=%d reads=%d finds=%d",
			f.probe.calls, f.journal.reads, f.journal.finds)
	}
	if s44Image(f) != before || articles != 3 || operations != 3 || journaled != 3 ||
		audits != 0 || runs != 0 || evidence != 0 {
		t.Fatalf("unavailable inventory wrote state: articles=%d operations=%d journaled=%d audits=%d runs=%d evidence=%d",
			articles, operations, journaled, audits, runs, evidence)
	}
	t.Logf("inventory_unavailable schema=%s probes=0 evidence=0 tombstone_writes=0", f.schema)
}

func s44OldBackupDecodeReady(t *testing.T, raw []byte) recovery.ReadyReplayStatusV1 {
	t.Helper()
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	s44Must(t, err)
	if decoded.Ready == nil {
		t.Fatal("expected Ready replay status")
	}
	return *decoded.Ready
}

// s44OldBackupConfigurePublication rebuilds the Store/Source/Provider on top of
// the shared fixture so that the production publication ports are configured.
// This path keeps its own assembler instead of the shared
// s44ConfigureReplayPublication; the two are equivalent today (registered as
// R02-N02 in review-code-s4.4-old-backup-r02.md). Without sealing, the provider
// only returns unavailable and never writes evidence. The Journal
// snapshot and runtime binding below are synthetic stand-ins for the real
// POSIX Journal and recovery runtime.
func s44OldBackupConfigurePublication(t *testing.T, f *s44Fixture) {
	t.Helper()
	if f == nil || f.probe == nil || f.journal == nil || f.evidenceClock == nil {
		t.Fatal("publication configuration requires the prepared shared fixture")
	}
	s44OldBackupApplyPublicationMigration(t, f)

	journal := &s44OldBackupJournal{s44Journal: f.journal}
	inventory, err := article.NewFileBackupInventoryReader(filepath.Join(f.dir, "synthetic-inventory.json"))
	s44OldBackupMust(t, "inventory reader", err)
	facts, err := article.NewPostgresReplayFactReader(f.pool)
	s44OldBackupMust(t, "fact reader", err)
	source, err := article.NewProductionDeletionReplaySource(journal, inventory, f.probe, facts, facts, f.clock.Now)
	s44OldBackupMust(t, "replay source", err)

	store, err := article.NewPostgresReplayEvidenceStore(f.pool)
	s44OldBackupMust(t, "evidence store", err)
	publicationIDs := &s44IDs{}
	for index := 1; index <= 16; index++ {
		publicationIDs.remaining = append(publicationIDs.remaining,
			fmt.Sprintf("66000000-0000-4000-8000-%012d", index))
	}
	store, err = article.NewReplayPublicationPorts(store, article.ReplayPublicationDependencies{
		Source: source, Runtime: &s44OldBackupRuntime{}, InstallationID: s44OldBackupInstallation,
		PublicationIDs: publicationIDs, Clock: f.evidenceClock,
	})
	s44OldBackupMust(t, "replay publication ports", err)

	f.storeProbe.ReplayEvidenceStore = store
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{
		Store: f.storeProbe, Source: source, EvidenceIDs: f.evidenceIDs, Tokens: f.tokens, Clock: f.evidenceClock,
	})
	s44OldBackupMust(t, "replay provider", err)
	f.source, f.store, f.provider = source, store, provider
	t.Logf("publication_configured schema=%s migration=0011 journal_id=%s epoch=1 synthetic=true",
		f.schema, s44OldBackupJournalID)
}

func s44OldBackupApplyPublicationMigration(t *testing.T, f *s44Fixture) {
	t.Helper()
	s44ApplyPublicationMigration(t, f)
}

func s44OldBackupMust(t *testing.T, stage string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s failed: %v", stage, err)
	}
}

// s44OldBackupJournal projects the shared synthetic Journal records through the
// recovery publication reader contract. It is not a POSIX Journal.
type s44OldBackupJournal struct {
	*s44Journal
}

func (journal *s44OldBackupJournal) ReadVerifiedSnapshot(
	ctx context.Context, namespace string,
) (recovery.VerifiedJournalSnapshot, error) {
	if journal == nil || journal.s44Journal == nil || ctx == nil {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return recovery.VerifiedJournalSnapshot{}, err
	}
	if namespace != article.ArticleDeletionNamespace {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	prefixes := []recovery.JournalHead{{Sequence: 0, Hash: ""}}
	for index, record := range journal.records {
		if record.Sequence != int64(index+1) || record.Namespace != namespace {
			return recovery.VerifiedJournalSnapshot{}, article.ErrDeletionJournalCorrupt
		}
		prefixes = append(prefixes, recovery.JournalHead{
			Sequence: uint64(record.Sequence), Hash: s44OldBackupRecordHash(record)})
	}
	return recovery.VerifiedJournalSnapshot{
		JournalID: s44OldBackupJournalID,
		Namespace: namespace,
		Epoch:     1,
		Head:      prefixes[len(prefixes)-1],
		Revision:  s44OldBackupRevision,
		Prefixes:  prefixes,
	}, nil
}

// s44OldBackupRecordHash delegates to the single synthetic projection so the
// third path cannot drift from the shared replay harness.
func s44OldBackupRecordHash(record article.ArticleDeletionTombstone) string {
	return s44PublicationRecordHash(record)
}

// s44OldBackupRuntime supplies only the observed permit binding that the
// publication port checks. It is not a real recovery runtime.
type s44OldBackupRuntime struct {
	recovery.Runtime
}

func (host *s44OldBackupRuntime) ReadPermitBinding(ctx context.Context) (recovery.PermitBinding, error) {
	if ctx == nil {
		return recovery.PermitBinding{}, recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return recovery.PermitBinding{}, err
	}
	return recovery.PermitBinding{
		RuntimeID:      "synthetic-old-backup-runtime",
		Generation:     1,
		RunID:          s44Run,
		BackupRef:      s44Backup,
		InstallationID: s44OldBackupInstallation,
		Purpose:        recovery.PhaseRecoveryReplay,
	}, nil
}

// CheckPhase mirrors the production purpose check for the synthetic harness.
func (host *s44OldBackupRuntime) CheckPhase(
	ctx context.Context,
	_ recovery.PhasePermit,
	purpose string,
) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch purpose {
	case recovery.PhaseRecoveryReplay, s44PublicationInstallPhase:
		return nil
	default:
		return recovery.ErrAdmissionBinding
	}
}

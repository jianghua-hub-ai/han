//go:build integration

package recovery_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/recovery"
)

func TestJournalFaults_CorruptionAndConsumption_FailClosed(t *testing.T) {
	t.Run("MarkConsumedDuplicateAndConcurrent", func(t *testing.T) {
		ctx := context.Background()
		journal := s417NewSoftwareJournal("s44-mark-journal", recovery.RecoveryNamespaceV1, 1, func() time.Time {
			return s417Canonical(time.Now())
		})
		if _, err := journal.MarkConsumed(ctx, "s44-run-a"); !errors.Is(err, recovery.ErrRunNotFound) {
			t.Fatalf("MarkConsumed before Begin err=%v; want ErrRunNotFound", err)
		}
		if _, err := journal.Begin(ctx, "s44-run-a", "s44-backup-a", []byte("s44-secret-a")); err != nil {
			t.Fatalf("Begin run-a failed: %v", err)
		}
		if _, err := journal.Begin(ctx, "s44-run-a", "s44-backup-a", []byte("s44-secret-a")); !errors.Is(err, recovery.ErrRunAlreadyExists) {
			t.Fatalf("duplicate Begin err=%v; want ErrRunAlreadyExists", err)
		}
		if _, err := journal.MarkConsumed(ctx, "s44-run-a"); err != nil {
			t.Fatalf("first MarkConsumed failed: %v", err)
		}
		if _, err := journal.MarkConsumed(ctx, "s44-run-a"); !errors.Is(err, recovery.ErrJournalMismatch) {
			t.Fatalf("duplicate MarkConsumed err=%v; want ErrJournalMismatch", err)
		}
		if _, err := journal.Begin(ctx, "s44-run-b", "s44-backup-b", []byte("s44-secret-b")); err != nil {
			t.Fatalf("Begin run-b failed: %v", err)
		}

		const writers = 16
		var successes int64
		var mismatches int64
		failures := make(chan error, writers)
		var wait sync.WaitGroup
		for index := 0; index < writers; index++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				_, err := journal.MarkConsumed(ctx, "s44-run-b")
				switch {
				case err == nil:
					atomic.AddInt64(&successes, 1)
				case errors.Is(err, recovery.ErrJournalMismatch):
					atomic.AddInt64(&mismatches, 1)
				default:
					failures <- err
				}
			}()
		}
		wait.Wait()
		close(failures)
		for err := range failures {
			t.Fatalf("concurrent MarkConsumed unexpected err=%v", err)
		}
		if successes != 1 || mismatches != writers-1 {
			t.Fatalf("concurrent MarkConsumed successes=%d mismatches=%d; want 1/%d", successes, mismatches, writers-1)
		}
		if calls := journal.consumedMarkCount(); calls != 3+writers {
			t.Fatalf("consumed invocation count=%d; want %d (every accepted and rejected call)", calls, 3+writers)
		}
		if callers := journal.consumedCallerSnapshot(); len(callers) != 3+writers {
			t.Fatalf("consumed caller snapshot=%d; want %d", len(callers), 3+writers)
		}

		snapshot, err := journal.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
		if err != nil {
			t.Fatalf("read verified snapshot failed: %v", err)
		}
		if len(snapshot.Entries) != 4 {
			t.Fatalf("journal entries=%d; want 4 (two stages for each of two runs)", len(snapshot.Entries))
		}
		stages := map[string]map[recovery.Stage]int{}
		previous := ""
		for index, entry := range snapshot.Entries {
			if entry.Sequence != uint64(index+1) {
				t.Fatalf("entry[%d] sequence=%d; want %d", index, entry.Sequence, index+1)
			}
			if entry.Namespace != recovery.RecoveryNamespaceV1 {
				t.Fatalf("entry[%d] namespace=%s; want %s", index, entry.Namespace, recovery.RecoveryNamespaceV1)
			}
			if entry.PreviousRecordHash != previous {
				t.Fatalf("entry[%d] previous hash=%s; want %s", index, entry.PreviousRecordHash, previous)
			}
			if stages[entry.RunID] == nil {
				stages[entry.RunID] = map[recovery.Stage]int{}
			}
			stages[entry.RunID][entry.Stage]++
			previous = s417ChainHash(previous, entry)
		}
		if snapshot.Head.Sequence != 4 || snapshot.Head.Hash != previous {
			t.Fatalf("head=%+v; want sequence 4 and recomputed chain hash %s", snapshot.Head, previous)
		}
		for _, runID := range []string{"s44-run-a", "s44-run-b"} {
			if stages[runID][recovery.StageBegun] != 1 || stages[runID][recovery.StageConsumed] != 1 {
				t.Fatalf("run %s stages=%v; want exactly one begun and one consumed", runID, stages[runID])
			}
		}
		t.Logf("MarkConsumed ordering: unbegun=ErrRunNotFound duplicate-begin=ErrRunAlreadyExists duplicate-consumed=ErrJournalMismatch concurrent=1/%d mismatch calls=%d chain=contiguous", writers, journal.consumedMarkCount())
	})

	t.Run("ReplayAfterControlledRestart_SoftwareJournalOnly", func(t *testing.T) {
		ctx := context.Background()
		journal := s417NewSoftwareJournal("s44-restart-journal", recovery.RecoveryNamespaceV1, 1, func() time.Time {
			return s417Canonical(time.Now())
		})
		secret := []byte("s44-restart-secret")
		begun, err := journal.Begin(ctx, "s44-restart-run", "s44-restart-backup", secret)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		consumed, err := journal.MarkConsumed(ctx, "s44-restart-run")
		if err != nil {
			t.Fatalf("MarkConsumed failed: %v", err)
		}
		restarted := &s44TamperedJournal{softwareJournal: journal}
		snapshot, err := restarted.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
		if err != nil {
			t.Fatalf("restarted reader failed: %v", err)
		}
		if len(snapshot.Entries) != 2 || snapshot.Head.Sequence != 2 {
			t.Fatalf("restart snapshot entries=%d head=%d; want 2/2", len(snapshot.Entries), snapshot.Head.Sequence)
		}
		if snapshot.Entries[0].Stage != recovery.StageBegun || snapshot.Entries[1].Stage != recovery.StageConsumed {
			t.Fatalf("restart stages=%s,%s; want begun,consumed", snapshot.Entries[0].Stage, snapshot.Entries[1].Stage)
		}
		if snapshot.Entries[0].SecretFingerprint != begun.SecretFingerprint || snapshot.Entries[1].SecretFingerprint != consumed.SecretFingerprint {
			t.Fatalf("restart secret fingerprint drifted")
		}
		if snapshot.Entries[1].PreviousRecordHash == "" || snapshot.Head.Hash == "" {
			t.Fatalf("restart chain hashes missing")
		}
		if _, err := restarted.ReadVerifiedSnapshot(ctx, recovery.ArticleDeletionNamespaceV1); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("restart reader accepted article namespace: %v", err)
		}
		t.Logf("controlled restart replay consistent: entries=2 head=2 stages=begun,consumed duplicate_namespace_rejected=true")
		t.Log("NOT RUN: real process crash / transaction interruption durability requires the persistent-volume environment; the controlled software journal cannot prove fsync or on-disk replay")
	})

	t.Run("CorruptRecoverySnapshotRejectedByAdmission", func(t *testing.T) {
		var tampered *s44TamperedJournal
		world := s44NewWorldWith(t, s44WorldOptions{
			RecoveryJournal: func(j *softwareJournal) recovery.ContextRecoveryJournal {
				tampered = &s44TamperedJournal{softwareJournal: j}
				return tampered
			},
		})

		tampered.mutate = func(s *recovery.VerifiedJournalSnapshot) error {
			s44InjectRecoveryEntry(s, func(e *recovery.JournalEntry) {
				e.RecordHMAC = "hmac-sha256:corrupt"
			})
			return nil
		}
		observation, err := world.runtime.Verify(world.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionIncomplete) {
			t.Fatalf("invalid RecordHMAC err=%v; want ErrAdmissionIncomplete", err)
		}

		tampered.mutate = func(s *recovery.VerifiedJournalSnapshot) error {
			s44InjectRecoveryEntry(s, func(e *recovery.JournalEntry) {
				e.PreviousRecordHash = s417Digest("s44-non-empty-first-previous-hash")
			})
			return nil
		}
		observation, err = world.runtime.Verify(world.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("first PreviousRecordHash err=%v; want ErrAdmissionBinding", err)
		}

		tampered.mutate = func(s *recovery.VerifiedJournalSnapshot) error {
			s44InjectRecoveryEntry(s, nil)
			s.Head = recovery.JournalHead{Sequence: 0, Hash: ""}
			s.Prefixes = []recovery.JournalHead{{Sequence: 0, Hash: ""}}
			return nil
		}
		observation, err = world.runtime.Verify(world.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionIncomplete) {
			t.Fatalf("entry without head advance err=%v; want ErrAdmissionIncomplete", err)
		}

		tampered.mutate = func(s *recovery.VerifiedJournalSnapshot) error {
			s44InjectRecoveryEntry(s, nil)
			return nil
		}
		observation, err = world.runtime.Verify(world.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionIncomplete) {
			t.Fatalf("journal entries without RecoveryRun err=%v; want ErrAdmissionIncomplete", err)
		}
		t.Logf("corrupt recovery snapshots rejected fail-closed: invalid-hmac=incomplete first-previous-hash=binding head-not-advanced=incomplete entries-without-run=incomplete runtime=closed")
	})

	t.Run("FileJournalCrashCorruption_PersistentVolume", func(t *testing.T) {
		if !s44FileJournalSupported() {
			ctx := context.Background()
			_, err := recovery.NewFileRecoveryJournal(ctx, recovery.JournalOptions{}, recovery.Keyring{})
			if !errors.Is(err, recovery.ErrUnsupportedPlatform) {
				t.Fatalf("NewFileRecoveryJournal err=%v; want ErrUnsupportedPlatform", err)
			}
			_, err = recovery.OpenVerifiedRecoveryJournal(ctx, recovery.ReadOptions{})
			if !errors.Is(err, recovery.ErrAdmissionUnsupported) || !errors.Is(err, recovery.ErrUnsupportedPlatform) {
				t.Fatalf("OpenVerifiedRecoveryJournal err=%v; want ErrAdmissionUnsupported+ErrUnsupportedPlatform", err)
			}
			t.Logf("branch=non-posix runtime.GOOS=%s result=NewFileRecoveryJournal=ErrUnsupportedPlatform and OpenVerifiedRecoveryJournal=ErrAdmissionUnsupported+ErrUnsupportedPlatform asserted; file-journal crash/corruption injection NOT RUN: %s", runtime.GOOS, s44NotRunReason())
			return
		}
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod journal dir: %v", err)
		}
		anchor := newS44MemoryAnchor()
		now := s417Canonical(time.Now())
		journal := s44NewFileJournal(t, dir, anchor, now, true, []recovery.Key{
			{ID: "s44-recovery", Version: "v1", State: recovery.KeyStateActive, Material: []byte("s44-material-v1")},
		})
		if _, err := journal.Begin(context.Background(), "s44-fault-run", "s44-fault-backup", []byte("s44-fault-secret")); err != nil {
			t.Fatalf("Begin error=%v", err)
		}
		if _, err := journal.MarkConsumed(context.Background(), "s44-fault-run"); err != nil {
			t.Fatalf("MarkConsumed error=%v", err)
		}
		file, err := os.OpenFile(filepath.Join(dir, "recovery.journal"), os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatalf("open journal for partial write failed: %v", err)
		}
		if _, err := file.WriteString(`{"namespace":"recovery/v1"`); err != nil {
			_ = file.Close()
			t.Fatalf("partial trailing write failed: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close journal after partial write failed: %v", err)
		}
		if _, err := journal.Read(context.Background(), "s44-fault-run"); !errors.Is(err, recovery.ErrJournalCorrupt) {
			t.Fatalf("file journal partial trailing write err=%v; want ErrJournalCorrupt", err)
		}
		t.Logf("branch=posix runtime.GOOS=%s result=partial trailing write rejected with ErrJournalCorrupt (real file journal under temp dir; anchor=s44MemoryAnchor test double)", runtime.GOOS)
	})
}

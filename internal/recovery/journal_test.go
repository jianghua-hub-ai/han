package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

const testJournalMode fs.FileMode = 0o600

func TestKeyRetirementSeparatorCollisionCannotBypassRetainedRecordGate(t *testing.T) {
	verifier := newDeterministicRetirementVerifier()
	references := map[keyReference]struct{}{
		{id: "recovery\x00v1", version: "archive"}: {},
		{id: "recovery", version: "v1\x00archive"}: {},
	}
	if len(references) != 2 {
		t.Fatal("composite key references collided")
	}
	keyring, err := NewKeyring([]Key{
		{ID: "active", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
		{
			ID: "recovery\x00v1", Version: "archive", State: KeyStateVerifyOnly,
			Material: []byte("synthetic-key-v1"),
		},
	})
	if !errors.Is(err, ErrInvalidKeyring) {
		_, collided := keyring.lookup("recovery", "v1\x00archive")
		t.Fatalf(
			"NewKeyring() error = %v, collided lookup = %v; control-bearing key must be rejected",
			err,
			collided,
		)
	}
	if calls := verifier.callCount(); calls != 0 {
		t.Fatalf("verifier calls = %d, want 0", calls)
	}
}

func TestKeyRetirementControlCharactersRejectedBeforeVerifier(t *testing.T) {
	keyring := mustKeyring(t, []Key{
		{ID: "active", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
		{ID: "recovery", Version: "v1", State: KeyStateVerifyOnly, Material: []byte("synthetic-key-v1")},
	})
	tests := []struct {
		name       string
		control    string
		requestKey bool
	}{
		{name: "NUL in key", control: "recovery\x00archive", requestKey: true},
		{name: "newline in key", control: "recovery\narchive", requestKey: true},
		{name: "DEL in version", control: "v1\u007farchive"},
		{name: "C1 control in version", control: "v1\u0085archive"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidateID, candidateVersion := "retained", "v0"
			if test.requestKey {
				candidateID = test.control
			} else {
				candidateVersion = test.control
			}
			_, err := NewKeyring([]Key{
				{ID: "active", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
				{
					ID: candidateID, Version: candidateVersion, State: KeyStateVerifyOnly,
					Material: []byte("synthetic-key-v1"),
				},
			})
			if !errors.Is(err, ErrInvalidKeyring) {
				t.Fatalf("NewKeyring() error = %v", err)
			}

			verifier := newDeterministicRetirementVerifier()
			journal := &FileRecoveryJournal{
				keyring: keyring, keyRetirementVerifier: verifier,
			}
			keyID, keyVersion := "recovery", "v1"
			if test.requestKey {
				keyID = test.control
			} else {
				keyVersion = test.control
			}
			err = journal.ValidateKeyRetirement(
				context.Background(), keyID, keyVersion, KeyRetirementCapability{},
			)
			if !errors.Is(err, ErrInvalidKeyring) {
				t.Fatalf("ValidateKeyRetirement() error = %v", err)
			}
			if calls := verifier.callCount(); calls != 0 {
				t.Fatalf("verifier calls = %d, want 0", calls)
			}
		})
	}
}

func TestKeyRetirementUsesResolvedKeyForRetainedRecordGate(t *testing.T) {
	requirePOSIXJournal(t)
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	now := time.Date(2026, 10, 22, 0, 25, 0, 0, time.UTC)
	v1 := newTestJournal(t, path, anchor, now, true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	if _, err := v1.Begin(context.Background(), "run-collision", "backup-collision", []byte("synthetic-secret")); err != nil {
		t.Fatal(err)
	}
	verifier := newDeterministicRetirementVerifier()
	aliasRequest := KeyRetirementRequest{KeyID: "alias", KeyVersion: "legacy"}
	capability := verifier.issue(
		"alias-capability",
		aliasRequest,
		KeyRetirementVerificationApproved,
	)
	v2 := newTestJournalWithRetirementVerifier(
		t,
		path,
		anchor,
		now.Add(time.Minute),
		false,
		[]Key{
			{ID: "recovery", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
			{ID: "recovery", Version: "v1", State: KeyStateVerifyOnly, Material: []byte("synthetic-key-v1")},
		},
		verifier,
	)
	resolved, ok := v2.keyring.lookup("recovery", "v1")
	if !ok {
		t.Fatal("retained key is unavailable")
	}
	v2.keyring.byRef[keyReference{id: aliasRequest.KeyID, version: aliasRequest.KeyVersion}] = resolved
	err := v2.ValidateKeyRetirement(
		context.Background(),
		aliasRequest.KeyID,
		aliasRequest.KeyVersion,
		capability,
	)
	if !errors.Is(err, ErrKeyRetirementBlocked) {
		t.Fatalf("ValidateKeyRetirement() error = %v", err)
	}
	if calls := verifier.callCount(); calls != 0 {
		t.Fatalf("verifier calls = %d, want 0", calls)
	}
}

func TestKeyRetirementStaticPrerequisitesCannotBeBypassed(t *testing.T) {
	requirePOSIXJournal(t)
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	now := time.Date(2026, 10, 22, 0, 30, 0, 0, time.UTC)
	v1 := newTestJournal(t, path, anchor, now, true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	if _, err := v1.Begin(context.Background(), "run-01", "backup-01", []byte("synthetic-secret")); err != nil {
		t.Fatal(err)
	}
	keys := []Key{
		{ID: "recovery", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
		{ID: "recovery", Version: "v1", State: KeyStateVerifyOnly, Material: []byte("synthetic-key-v1")},
	}
	verifier := newDeterministicRetirementVerifier()
	capability := verifier.issue(
		"approved-v1",
		KeyRetirementRequest{KeyID: "recovery", KeyVersion: "v1"},
		KeyRetirementVerificationApproved,
	)
	current := now.Add(time.Minute)
	v2 := newTestJournalWithRetirementVerifier(
		t, path, anchor, current, false, keys, verifier,
	)
	if err := v2.ValidateKeyRetirement(
		context.Background(), "recovery", "v1", capability,
	); !errors.Is(err, ErrKeyRetirementBlocked) {
		t.Fatalf("ValidateKeyRetirement(unconsumed) error = %v", err)
	}
	if calls := verifier.callCount(); calls != 0 {
		t.Fatalf("verifier calls with begun run = %d, want 0", calls)
	}
	if _, err := v2.MarkConsumed(context.Background(), "run-01"); err != nil {
		t.Fatal(err)
	}
	if err := v2.ValidateKeyRetirement(
		context.Background(), "recovery", "v1", capability,
	); !errors.Is(err, ErrKeyRetirementBlocked) {
		t.Fatalf("ValidateKeyRetirement(retained records) error = %v", err)
	}
	if calls := verifier.callCount(); calls != 0 {
		t.Fatalf("verifier calls with retained records = %d, want 0", calls)
	}
	records := readAllRecords(t, path)
	records[0].RecordHMAC = "hmac-sha256:00"
	writeAllRecords(t, path, records)
	if err := v2.ValidateKeyRetirement(
		context.Background(), "recovery", "v1", capability,
	); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("ValidateKeyRetirement(tampered journal) error = %v", err)
	}
}

func TestKeyRetirementVerifierFailsClosed(t *testing.T) {
	request := KeyRetirementRequest{KeyID: "recovery", KeyVersion: "v1"}
	unavailable := errors.New("synthetic verifier unavailable")
	tests := []struct {
		name         string
		configure    func(*deterministicRetirementVerifier) KeyRetirementCapability
		withoutPort  bool
		wantSpecific error
	}{
		{name: "unconfigured", withoutPort: true, wantSpecific: ErrKeyRetirementVerifierUnavailable},
		{name: "missing capability", wantSpecific: ErrKeyRetirementCapabilityRejected},
		{name: "rejected", configure: func(verifier *deterministicRetirementVerifier) KeyRetirementCapability {
			return verifier.issue("rejected", request, KeyRetirementVerificationRejected)
		}, wantSpecific: ErrKeyRetirementCapabilityRejected},
		{name: "unavailable", configure: func(verifier *deterministicRetirementVerifier) KeyRetirementCapability {
			verifier.verifyErr = unavailable
			return verifier.issue("unavailable", request, KeyRetirementVerificationApproved)
		}, wantSpecific: ErrKeyRetirementVerifierUnavailable},
		{name: "indeterminate", configure: func(verifier *deterministicRetirementVerifier) KeyRetirementCapability {
			return verifier.issue("indeterminate", request, KeyRetirementVerificationIndeterminate)
		}, wantSpecific: ErrKeyRetirementCapabilityIndeterminate},
		{name: "wrong key", configure: func(verifier *deterministicRetirementVerifier) KeyRetirementCapability {
			return verifier.issue(
				"wrong-key",
				KeyRetirementRequest{KeyID: "other", KeyVersion: "v1"},
				KeyRetirementVerificationApproved,
			)
		}, wantSpecific: ErrKeyRetirementCapabilityRejected},
		{name: "wrong version", configure: func(verifier *deterministicRetirementVerifier) KeyRetirementCapability {
			return verifier.issue(
				"wrong-version",
				KeyRetirementRequest{KeyID: "recovery", KeyVersion: "v0"},
				KeyRetirementVerificationApproved,
			)
		}, wantSpecific: ErrKeyRetirementCapabilityRejected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := newDeterministicRetirementVerifier()
			var capability KeyRetirementCapability
			if test.configure != nil {
				capability = test.configure(verifier)
			}
			var port KeyRetirementCapabilityVerifier = verifier
			if test.withoutPort {
				port = nil
			}
			journal := newPortableRetirementJournal(port)
			err := lockedTestRetirement(
				context.Background(), journal, request, capability,
			)
			if !errors.Is(err, ErrKeyRetirementBlocked) ||
				!errors.Is(err, test.wantSpecific) {
				t.Fatalf("ValidateKeyRetirement() error = %v", err)
			}
		})
	}
}

func TestKeyRetirementCapabilityIsConsumedOnce(t *testing.T) {
	request := KeyRetirementRequest{KeyID: "recovery", KeyVersion: "v1"}
	verifier := newDeterministicRetirementVerifier()
	capability := verifier.issue(
		"single-use",
		request,
		KeyRetirementVerificationApproved,
	)
	journal := newPortableRetirementJournal(verifier)
	if err := lockedTestRetirement(
		context.Background(), journal, request, capability,
	); err != nil {
		t.Fatalf("ValidateKeyRetirement(first) error = %v", err)
	}
	copied := capability
	for attempt, candidate := range []KeyRetirementCapability{capability, copied} {
		err := lockedTestRetirement(
			context.Background(), journal, request, candidate,
		)
		if !errors.Is(err, ErrKeyRetirementBlocked) ||
			!errors.Is(err, ErrKeyRetirementCapabilityRejected) {
			t.Fatalf("ValidateKeyRetirement(replay %d) error = %v", attempt+1, err)
		}
	}

	concurrent := verifier.issue(
		"concurrent-single-use",
		request,
		KeyRetirementVerificationApproved,
	)
	const attempts = 8
	start := make(chan struct{})
	results := make(chan error, attempts)
	var workers sync.WaitGroup
	for range attempts {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			results <- lockedTestRetirement(
				context.Background(), journal, request, concurrent,
			)
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	approved := 0
	rejected := 0
	for err := range results {
		switch {
		case err == nil:
			approved++
		case errors.Is(err, ErrKeyRetirementBlocked) &&
			errors.Is(err, ErrKeyRetirementCapabilityRejected):
			rejected++
		default:
			t.Fatalf("concurrent VerifyAndConsume() error = %v", err)
		}
	}
	if approved != 1 || rejected != attempts-1 {
		t.Fatalf("concurrent results approved=%d rejected=%d", approved, rejected)
	}
}

func TestKeyRetirementValidateUsesOpaqueVerifierBoundary(t *testing.T) {
	requirePOSIXJournal(t)
	keys := []Key{
		{ID: "recovery", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
		{ID: "recovery", Version: "v1", State: KeyStateVerifyOnly, Material: []byte("synthetic-key-v1")},
	}
	request := KeyRetirementRequest{KeyID: "recovery", KeyVersion: "v1"}
	t.Run("unconfigured", func(t *testing.T) {
		journal := newTestJournal(
			t,
			filepath.Join(newJournalTestDir(t), "recovery.journal"),
			newDeterministicAnchor(),
			time.Date(2026, 10, 22, 0, 45, 0, 0, time.UTC),
			true,
			keys,
		)
		err := journal.ValidateKeyRetirement(
			context.Background(), request.KeyID, request.KeyVersion, KeyRetirementCapability{},
		)
		if !errors.Is(err, ErrKeyRetirementBlocked) ||
			!errors.Is(err, ErrKeyRetirementVerifierUnavailable) {
			t.Fatalf("ValidateKeyRetirement() error = %v", err)
		}
	})
	t.Run("approved then replayed", func(t *testing.T) {
		verifier := newDeterministicRetirementVerifier()
		capability := verifier.issue(
			"public-boundary-single-use",
			request,
			KeyRetirementVerificationApproved,
		)
		journal := newTestJournalWithRetirementVerifier(
			t,
			filepath.Join(newJournalTestDir(t), "recovery.journal"),
			newDeterministicAnchor(),
			time.Date(2026, 10, 22, 0, 46, 0, 0, time.UTC),
			true,
			keys,
			verifier,
		)
		if err := journal.ValidateKeyRetirement(
			context.Background(), request.KeyID, request.KeyVersion, capability,
		); err != nil {
			t.Fatalf("ValidateKeyRetirement(first) error = %v", err)
		}
		err := journal.ValidateKeyRetirement(
			context.Background(), request.KeyID, request.KeyVersion, capability,
		)
		if !errors.Is(err, ErrKeyRetirementBlocked) ||
			!errors.Is(err, ErrKeyRetirementCapabilityRejected) {
			t.Fatalf("ValidateKeyRetirement(replay) error = %v", err)
		}
	})
}

func TestJournalLockInstanceSeamsRemainIsolatedInParallel(t *testing.T) {
	firstErr := errors.New("first acquire failed")
	secondErr := errors.New("second acquire failed")
	newJournal := func(acquireErr error) *FileRecoveryJournal {
		return &FileRecoveryJournal{
			acquireLock: func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
				return nil, acquireErr
			},
		}
	}
	first := newJournal(firstErr)
	second := newJournal(secondErr)
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, journal := range []*FileRecoveryJournal{first, second} {
		workers.Add(1)
		go func(journal *FileRecoveryJournal) {
			defer workers.Done()
			<-start
			results <- journal.withLock(context.Background(), func(ctx context.Context) error { return nil })
		}(journal)
	}
	close(start)
	workers.Wait()
	close(results)
	seen := map[error]int{}
	for err := range results {
		switch {
		case errors.Is(err, firstErr) && !errors.Is(err, secondErr):
			seen[firstErr]++
		case errors.Is(err, secondErr) && !errors.Is(err, firstErr):
			seen[secondErr]++
		default:
			t.Fatalf("withLock() error crossed instance seams: %v", err)
		}
	}
	if seen[firstErr] != 1 || seen[secondErr] != 1 {
		t.Fatalf("acquire errors = %#v", seen)
	}
}

func TestFileRecoveryJournalBindsBackupAndBeginKeyAcrossRotation(t *testing.T) {
	requirePOSIXJournal(t)
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	now := time.Date(2026, 10, 22, 1, 2, 3, 123456789, time.FixedZone("offset", 3600))
	v1 := newTestJournal(t, path, anchor, now, true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	begun, err := v1.Begin(context.Background(), "run-01", "backup-01", []byte("synthetic-secret"))
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	wantTime := now.UTC().Truncate(time.Microsecond)
	if begun.BackupRef != "backup-01" || !begun.RecordedAt.Equal(wantTime) {
		t.Fatalf("begun binding = backup:%q time:%s", begun.BackupRef, begun.RecordedAt)
	}
	v2 := newTestJournal(t, path, anchor, now.Add(time.Second), false, []Key{
		{ID: "recovery", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
		{ID: "recovery", Version: "v1", State: KeyStateVerifyOnly, Material: []byte("synthetic-key-v1")},
	})
	consumed, err := v2.MarkConsumed(context.Background(), "run-01")
	if err != nil {
		t.Fatalf("MarkConsumed() error = %v", err)
	}
	if consumed.KeyVersion != "v1" || consumed.BackupRef != "backup-01" {
		t.Fatalf("consumed binding = key:%q backup:%q", consumed.KeyVersion, consumed.BackupRef)
	}
	run := RecoveryRun{
		RunID: "run-01", BackupRef: "backup-01", JournalKeyID: "recovery",
		JournalKeyVersion: "v1", JournalSecretFingerprint: begun.SecretFingerprint,
		JournalStage: StageConsumed, StartedAt: begun.RecordedAt, JournalConsumedAt: &consumed.RecordedAt,
	}
	if _, err := v2.VerifyAgainst(context.Background(), run); err != nil {
		t.Fatalf("VerifyAgainst() error = %v", err)
	}
	run.BackupRef = "backup-other"
	if _, err := v2.VerifyAgainst(context.Background(), run); !errors.Is(err, ErrJournalMismatch) {
		t.Fatalf("VerifyAgainst(wrong backup) error = %v", err)
	}
}

func TestFileRecoveryJournalRejectsTamperedBackupAndNonCanonicalTime(t *testing.T) {
	requirePOSIXJournal(t)
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	journal := newTestJournal(t, path, anchor, time.Date(2026, 10, 22, 2, 0, 0, 0, time.UTC), true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	if _, err := journal.Begin(context.Background(), "run-01", "backup-01", []byte("synthetic-secret")); err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	record := readSingleRecord(t, path)
	record.BackupRef = "backup-other"
	writeSingleRecord(t, path, record)
	if _, err := journal.Read(context.Background(), "run-01"); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("Read(tampered backup) error = %v", err)
	}
	record.BackupRef = "backup-01"
	record.RecordedAt = "2026-10-22T02:00:00.000000001Z"
	record.RecordHMAC = computeRecordHMAC(record, []byte("synthetic-key-v1"))
	writeSingleRecord(t, path, record)
	anchor.forceCommitted(headForRecords([]diskRecord{record}))
	if _, err := journal.Read(context.Background(), "run-01"); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("Read(noncanonical time) error = %v", err)
	}
}

func TestFileRecoveryJournalRetainedReplayAndMissingKeyFailClosed(t *testing.T) {
	requirePOSIXJournal(t)
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	now := time.Date(2026, 10, 22, 2, 30, 0, 0, time.UTC)
	secret := []byte("synthetic-secret")
	v1 := newTestJournal(t, path, anchor, now, true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	if _, err := v1.Begin(context.Background(), "run-v1", "backup-v1", secret); err != nil {
		t.Fatal(err)
	}
	if _, err := v1.MarkConsumed(context.Background(), "run-v1"); err != nil {
		t.Fatal(err)
	}
	v2 := newTestJournal(t, path, anchor, now.Add(time.Minute), false, []Key{
		{ID: "recovery", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
		{ID: "recovery", Version: "v1", State: KeyStateVerifyOnly, Material: []byte("synthetic-key-v1")},
	})
	if _, err := v2.Begin(context.Background(), "run-replay", "backup-v2", secret); !errors.Is(err, ErrSecretReused) {
		t.Fatalf("Begin(replay) error = %v", err)
	}
	keyring := mustKeyring(t, []Key{
		{ID: "recovery", Version: "v2", State: KeyStateActive, Material: []byte("synthetic-key-v2")},
	})
	_, err := NewFileRecoveryJournal(context.Background(), JournalOptions{ReadMaxBytes: 8 << 20, ReadMaxRecords: 100000,
		Path: path, FileMode: testJournalMode, Now: func() time.Time { return now },
		JournalID: "synthetic-journal", Epoch: 1, Anchor: anchor,
	}, keyring)
	if !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("NewFileRecoveryJournal(missing key) error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, secret) {
		t.Fatal("journal contains recovery secret")
	}
}

func TestFileRecoveryJournalCorruptionMatrixFailsClosed(t *testing.T) {
	requirePOSIXJournal(t)
	tests := []struct {
		name   string
		mutate func([]diskRecord) []diskRecord
		raw    bool
	}{
		{name: "partial write", raw: true},
		{name: "hmac", mutate: func(records []diskRecord) []diskRecord {
			records[1].RecordHMAC = "hmac-sha256:00"
			return records
		}},
		{name: "chain", mutate: func(records []diskRecord) []diskRecord {
			records[1].PreviousRecordHash = "sha256:00"
			records[1].RecordHMAC = computeRecordHMAC(records[1], []byte("synthetic-key-v1"))
			return records
		}},
		{name: "sequence", mutate: func(records []diskRecord) []diskRecord {
			records[1].Sequence = 1
			records[1].RecordHMAC = computeRecordHMAC(records[1], []byte("synthetic-key-v1"))
			return records
		}},
		{name: "namespace", mutate: func(records []diskRecord) []diskRecord {
			records[1].Namespace = ArticleDeletionNamespaceV1
			records[1].RecordHMAC = computeRecordHMAC(records[1], []byte("synthetic-key-v1"))
			return records
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(newJournalTestDir(t), "recovery.journal")
			anchor := newDeterministicAnchor()
			journal := newTestJournal(t, path, anchor, time.Date(2026, 10, 22, 2, 45, 0, 0, time.UTC), true, []Key{
				{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
			})
			if _, err := journal.Begin(context.Background(), "run-01", "backup-01", []byte("synthetic-secret")); err != nil {
				t.Fatal(err)
			}
			if _, err := journal.MarkConsumed(context.Background(), "run-01"); err != nil {
				t.Fatal(err)
			}
			if test.raw {
				file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, testJournalMode)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = file.WriteString(`{"namespace":"recovery/v1"`)
				_ = file.Close()
			} else {
				records := test.mutate(readAllRecords(t, path))
				writeAllRecords(t, path, records)
			}
			if _, err := journal.Read(context.Background(), "run-01"); !errors.Is(err, ErrJournalCorrupt) {
				t.Fatalf("Read() error = %v", err)
			}
		})
	}
}

func TestJournalHeadAnchorFailuresFailClosed(t *testing.T) {
	requirePOSIXJournal(t)
	for _, injected := range []error{
		ErrAnchorConflict, ErrAnchorStaleRevision, ErrAnchorMissing, ErrAnchorRollback, ErrAnchorIndeterminate,
	} {
		t.Run(injected.Error(), func(t *testing.T) {
			path := filepath.Join(newJournalTestDir(t), "recovery.journal")
			anchor := newDeterministicAnchor()
			journal := newTestJournal(t, path, anchor, time.Now(), true, []Key{
				{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
			})
			anchor.nextCASErr = injected
			if _, err := journal.Begin(context.Background(), "run-01", "backup-01", []byte("synthetic-secret")); !errors.Is(err, injected) {
				t.Fatalf("Begin() error = %v, want %v", err, injected)
			}
		})
	}
}

func TestJournalHeadAnchorCASReloadDriftDoesNotPublishJournal(t *testing.T) {
	requirePOSIXJournal(t)
	tests := []struct {
		name     string
		override func(AnchorSnapshot, string) anchorLoadOverride
	}{
		{name: "old snapshot", override: func(old AnchorSnapshot, revision string) anchorLoadOverride {
			return anchorLoadOverride{snapshot: old, revision: revision}
		}},
		{name: "third snapshot", override: func(old AnchorSnapshot, revision string) anchorLoadOverride {
			old.Committed = JournalHead{Sequence: 1, Hash: SHA256Digest([]byte("third"))}
			return anchorLoadOverride{snapshot: old, revision: revision}
		}},
		{name: "wrong revision", override: func(snapshot AnchorSnapshot, _ string) anchorLoadOverride {
			return anchorLoadOverride{snapshot: snapshot, revision: "drifted-revision"}
		}},
		{name: "load error", override: func(AnchorSnapshot, string) anchorLoadOverride {
			return anchorLoadOverride{err: errors.New("anchor reload failed")}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(newJournalTestDir(t), "recovery.journal")
			anchor := newDeterministicAnchor()
			journal := newTestJournal(t, path, anchor, time.Now(), true, []Key{
				{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
			})
			old, revision, err := anchor.Load(context.Background(), RecoveryNamespaceV1)
			if err != nil {
				t.Fatal(err)
			}
			anchor.overrideLoadAfterNextCAS(test.override(old, revision))
			if _, err := journal.Begin(context.Background(),
				"run-01", "backup-01", []byte("synthetic-secret"),
			); !errors.Is(err, ErrAnchorIndeterminate) {
				t.Fatalf("Begin() error = %v", err)
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("journal advanced before authoritative reload: %v", err)
			}
		})
	}
}

func TestJournalHeadAnchorCASReloadDriftFailsClosed(t *testing.T) {
	base := AnchorSnapshot{
		JournalID: "synthetic-journal", Namespace: RecoveryNamespaceV1, Epoch: 1,
	}
	tests := []struct {
		name     string
		override anchorLoadOverride
	}{
		{name: "old snapshot", override: anchorLoadOverride{
			snapshot: base, revision: "old-revision",
		}},
		{name: "third snapshot", override: anchorLoadOverride{
			snapshot: AnchorSnapshot{
				JournalID: "synthetic-journal", Namespace: RecoveryNamespaceV1, Epoch: 1,
				Committed: JournalHead{Sequence: 1, Hash: SHA256Digest([]byte("third"))},
			},
			revision: "next-revision",
		}},
		{name: "wrong revision", override: anchorLoadOverride{
			snapshot: base, revision: "wrong-revision",
		}},
		{name: "load error", override: anchorLoadOverride{
			err: errors.New("anchor reload failed"),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			anchor := newDeterministicAnchor()
			anchor.forceSnapshot(base)
			anchor.overrideLoadAfterNextCAS(test.override)
			journal := &FileRecoveryJournal{
				journalID: "synthetic-journal", epoch: 1, anchor: anchor,
			}
			installSyntheticWriterLock(journal)
			if _, _, err := lockedTestCAS(context.Background(), journal, "forced", base); !errors.Is(
				err, ErrAnchorIndeterminate,
			) {
				t.Fatalf("compareAndSwapAnchor() error = %v", err)
			}
		})
	}
}

func TestJournalHeadAnchorPendingInvariantRejectsMalformedState(t *testing.T) {
	t.Parallel()
	journal := &FileRecoveryJournal{journalID: "synthetic-journal", epoch: 1}
	oldHead := JournalHead{Sequence: 1, Hash: SHA256Digest([]byte("old"))}
	newHead := JournalHead{Sequence: 2, Hash: SHA256Digest([]byte("new"))}
	base := AnchorSnapshot{
		JournalID: "synthetic-journal", Namespace: RecoveryNamespaceV1, Epoch: 1,
		Committed: oldHead,
		Pending: &AnchorPending{
			OldHead: oldHead, NewHead: newHead,
			OperationID: journalOperationID("synthetic-journal", newHead),
		},
	}
	tests := []struct {
		name   string
		mutate func(*AnchorSnapshot)
	}{
		{name: "committed differs from pending old", mutate: func(snapshot *AnchorSnapshot) {
			snapshot.Committed = JournalHead{Sequence: 1, Hash: SHA256Digest([]byte("third"))}
		}},
		{name: "new sequence does not follow old", mutate: func(snapshot *AnchorSnapshot) {
			snapshot.Pending.NewHead.Sequence = snapshot.Pending.OldHead.Sequence + 2
			snapshot.Pending.OperationID = journalOperationID("synthetic-journal", snapshot.Pending.NewHead)
		}},
		{name: "operation id does not bind new head", mutate: func(snapshot *AnchorSnapshot) {
			snapshot.Pending.OperationID = "synthetic-journal:2:" + SHA256Digest([]byte("other"))
		}},
		{name: "zero sequence has hash", mutate: func(snapshot *AnchorSnapshot) {
			snapshot.Committed = JournalHead{Hash: "sha256:invalid"}
			snapshot.Pending.OldHead = snapshot.Committed
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := base
			pending := *base.Pending
			snapshot.Pending = &pending
			test.mutate(&snapshot)
			if err := journal.validateAnchorSnapshot(snapshot, "revision"); !errors.Is(err, ErrAnchorRollback) {
				t.Fatalf("validateAnchorSnapshot() error = %v", err)
			}
		})
	}
}

func TestFileRecoveryJournalRejectsCommittedPendingOldMismatchForEveryFileHead(t *testing.T) {
	requirePOSIXJournal(t)
	for _, fileState := range []string{"old", "new", "third"} {
		t.Run(fileState, func(t *testing.T) {
			journal, anchor, path, oldRecords, newRecords, thirdRecords := anchorCrashFixture(t)
			oldHead := headForRecords(oldRecords)
			newHead := headForRecords(newRecords)
			anchor.forceSnapshot(AnchorSnapshot{
				JournalID: journal.journalID, Namespace: RecoveryNamespaceV1, Epoch: journal.epoch,
				Committed: headForRecords(thirdRecords),
				Pending: &AnchorPending{
					OldHead: oldHead, NewHead: newHead,
					OperationID: journalOperationID(journal.journalID, newHead),
				},
			})
			switch fileState {
			case "old":
				writeAllRecords(t, path, oldRecords)
			case "new":
				writeAllRecords(t, path, newRecords)
			case "third":
				writeAllRecords(t, path, thirdRecords)
			}
			if _, err := lockedTestRecords(context.Background(), journal, false); !errors.Is(err, ErrAnchorRollback) {
				t.Fatalf("loadRecords() error = %v", err)
			}
		})
	}
}

func TestFileRecoveryJournalExistingZeroHeadMissingFileRequiresFirstBootstrap(t *testing.T) {
	requirePOSIXJournal(t)
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	anchor.forceSnapshot(AnchorSnapshot{
		JournalID: "synthetic-journal", Namespace: RecoveryNamespaceV1, Epoch: 1,
	})
	_, err := NewFileRecoveryJournal(context.Background(), JournalOptions{ReadMaxBytes: 8 << 20, ReadMaxRecords: 100000,
		Path: path, FileMode: testJournalMode, Now: time.Now,
		JournalID: "synthetic-journal", Epoch: 1, Anchor: anchor, Bootstrap: false,
	}, mustKeyring(t, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	}))
	if !errors.Is(err, ErrAnchorRollback) {
		t.Fatalf("NewFileRecoveryJournal() error = %v", err)
	}
	firstPath := filepath.Join(newJournalTestDir(t), "recovery.journal")
	first := newTestJournal(t, firstPath, newDeterministicAnchor(), time.Now(), true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	if _, err := first.Begin(context.Background(),
		"run-first", "backup-first", []byte("synthetic-secret"),
	); err != nil {
		t.Fatalf("Begin(first bootstrap) error = %v", err)
	}
}

func TestFileRecoveryJournalPendingNewHeadWithoutTempSyncFailureRecovers(t *testing.T) {
	requirePOSIXJournal(t)

	journal, anchor, path, oldRecords, newRecords, _ := anchorCrashFixture(t)
	oldHead := headForRecords(oldRecords)
	newHead := headForRecords(newRecords)
	anchor.forceSnapshot(AnchorSnapshot{
		JournalID: journal.journalID, Namespace: RecoveryNamespaceV1, Epoch: journal.epoch,
		Committed: oldHead,
		Pending: &AnchorPending{
			OldHead: oldHead, NewHead: newHead,
			OperationID: journalOperationID(journal.journalID, newHead),
		},
	})
	writeAllRecords(t, path, newRecords)

	syncFailure := errors.New("synthetic directory sync failure")
	syncCalls := 0
	journal.syncDirectory = func(directory string) error {
		syncCalls++
		if directory != filepath.Dir(path) {
			t.Fatalf("sync directory = %q, want %q", directory, filepath.Dir(path))
		}
		return syncFailure
	}
	if _, err := lockedTestRecords(context.Background(), journal, false); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("loadRecords(sync failure) error = %v, want ErrJournalCorrupt", err)
	}
	if syncCalls != 1 {
		t.Fatalf("sync calls = %d, want 1", syncCalls)
	}
	snapshot, _, err := anchor.Load(context.Background(), RecoveryNamespaceV1)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Committed != oldHead || snapshot.Pending == nil {
		t.Fatalf("anchor committed despite sync failure: %#v", snapshot)
	}

	journal.syncDirectory = syncDirectoryPath
	records, err := lockedTestRecords(context.Background(), journal, false)
	if err != nil {
		t.Fatalf("loadRecords(recovery) error = %v", err)
	}
	if headForRecords(records) != newHead {
		t.Fatalf("records head = %#v, want %#v", headForRecords(records), newHead)
	}
	snapshot, _, err = anchor.Load(context.Background(), RecoveryNamespaceV1)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Pending != nil || snapshot.Committed != newHead {
		t.Fatalf("final anchor = %#v", snapshot)
	}
}

func TestFileRecoveryJournalCommitPendingNewHeadSyncFailurePreservesAnchor(t *testing.T) {
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	oldHead := JournalHead{Sequence: 1, Hash: SHA256Digest([]byte("old"))}
	newHead := JournalHead{Sequence: 2, Hash: SHA256Digest([]byte("new"))}
	snapshot := AnchorSnapshot{
		JournalID: "synthetic-journal", Namespace: RecoveryNamespaceV1, Epoch: 1,
		Committed: oldHead,
		Pending: &AnchorPending{
			OldHead: oldHead, NewHead: newHead,
			OperationID: journalOperationID("synthetic-journal", newHead),
		},
	}
	anchor := newDeterministicAnchor()
	anchor.forceSnapshot(snapshot)
	_, revision, err := anchor.Load(context.Background(), RecoveryNamespaceV1)
	if err != nil {
		t.Fatal(err)
	}
	journal := &FileRecoveryJournal{
		path: path, journalID: snapshot.JournalID, epoch: snapshot.Epoch, anchor: anchor,
	}
	installSyntheticWriterLock(journal)

	syncCalls := 0
	journal.syncDirectory = func(directory string) error {
		syncCalls++
		if directory != filepath.Dir(path) {
			t.Fatalf("sync directory = %q, want %q", directory, filepath.Dir(path))
		}
		return errors.New("synthetic directory sync failure")
	}
	if _, err := lockedTestPending(context.Background(), journal,
		revision, snapshot, newHead, true,
	); !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("commitPendingRecovery(sync failure) error = %v, want ErrJournalCorrupt", err)
	}
	if syncCalls != 1 {
		t.Fatalf("sync calls = %d, want 1", syncCalls)
	}
	current, currentRevision, err := anchor.Load(context.Background(), RecoveryNamespaceV1)
	if err != nil {
		t.Fatal(err)
	}
	if currentRevision != revision || !anchorSnapshotsEqual(current, snapshot) {
		t.Fatalf("anchor changed despite sync failure: %#v revision=%q", current, currentRevision)
	}

	journal.syncDirectory = func(string) error { return nil }
	finalHead, err := lockedTestPending(context.Background(), journal,
		revision, snapshot, newHead, true,
	)
	if err != nil {
		t.Fatalf("commitPendingRecovery(recovery) error = %v", err)
	}
	if finalHead != newHead {
		t.Fatalf("final head = %#v, want %#v", finalHead, newHead)
	}
	current, _, err = anchor.Load(context.Background(), RecoveryNamespaceV1)
	if err != nil {
		t.Fatal(err)
	}
	if current.Pending != nil || current.Committed != newHead {
		t.Fatalf("final anchor = %#v", current)
	}
}

func TestFileRecoveryJournalReconcilesValidPendingTempCrashMatrix(t *testing.T) {
	requirePOSIXJournal(t)
	for _, fileState := range []string{"old", "new", "third"} {
		t.Run(fileState, func(t *testing.T) {
			journal, anchor, path, oldRecords, newRecords, thirdRecords := anchorCrashFixture(t)
			oldHead := headForRecords(oldRecords)
			newHead := headForRecords(newRecords)
			anchor.forceSnapshot(AnchorSnapshot{
				JournalID: journal.journalID, Namespace: RecoveryNamespaceV1, Epoch: journal.epoch,
				Committed: oldHead,
				Pending: &AnchorPending{
					OldHead: oldHead, NewHead: newHead,
					OperationID: journalOperationID(journal.journalID, newHead),
				},
			})
			switch fileState {
			case "old":
				writeAllRecords(t, path, oldRecords)
			case "new":
				writeAllRecords(t, path, newRecords)
			case "third":
				writeAllRecords(t, path, thirdRecords)
			}
			tempPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp-crash")
			writeAllRecords(t, tempPath, newRecords)
			records, err := lockedTestRecords(context.Background(), journal, false)
			if fileState == "third" {
				if !errors.Is(err, ErrAnchorRollback) {
					t.Fatalf("loadRecords() error = %v", err)
				}
				if _, statErr := os.Lstat(tempPath); statErr != nil {
					t.Fatalf("third-head temp changed: %v", statErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadRecords() error = %v", err)
			}
			if headForRecords(records) != newHead {
				t.Fatalf("records head = %#v, want %#v", headForRecords(records), newHead)
			}
			snapshot, _, loadErr := anchor.Load(context.Background(), RecoveryNamespaceV1)
			if loadErr != nil || snapshot.Pending != nil || snapshot.Committed != newHead {
				t.Fatalf("final anchor = %#v, error = %v", snapshot, loadErr)
			}
			if _, statErr := os.Lstat(tempPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("temp remains after convergence: %v", statErr)
			}
		})
	}
}

func TestCoordinatorArticleGateErrorsPreserveSentinelAndCause(t *testing.T) {
	t.Parallel()

	source := errors.New("synthetic article gate source")
	cause := &articleGateTestError{cause: source}
	ready := mustReadyFixture(t)

	tests := []struct {
		name  string
		run   RecoveryRun
		gate  *stubArticleDeletionGate
		entry JournalEntry
	}{
		{
			name: "replay and verify",
			run: RecoveryRun{
				RunID: ready.RunID, BackupRef: ready.BackupRef,
				JournalKeyID: "recovery", JournalKeyVersion: "v1",
				JournalSecretFingerprint: "hmac-sha256:synthetic",
				JournalStage:             StageBegun,
				StartedAt:                time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC),
				Outcome:                  RecoveryOutcomeIdentityReady,
			},
			gate: &stubArticleDeletionGate{replayErr: cause},
			entry: JournalEntry{
				Namespace: RecoveryNamespaceV1, Sequence: 1,
				RunID: ready.RunID, BackupRef: ready.BackupRef,
				KeyID: "recovery", KeyVersion: "v1",
				SecretFingerprint: "hmac-sha256:synthetic", Stage: StageBegun,
				RecordedAt: time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "read status",
			run: func() RecoveryRun {
				run, _ := finalStateFixture(ready, RecoveryOutcomeSucceeded)
				return run
			}(),
			gate: &stubArticleDeletionGate{readErr: cause},
			entry: func() JournalEntry {
				_, journal := finalStateFixture(ready, RecoveryOutcomeSucceeded)
				return journal.entry
			}(),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			coordinator, err := newLegacyCoordinator(t,
				newMemoryRunStore(test.run),
				&memoryJournal{entry: test.entry},
				test.gate,
				NewReplayStatusV1Decoder(nil),
				time.Now,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = coordinator.Consume(context.Background(), test.run.RunID)
			if !errors.Is(err, ErrRecoveryNotReady) {
				t.Fatalf("Consume() error = %v, want ErrRecoveryNotReady", err)
			}
			if !errors.Is(err, source) {
				t.Fatalf("Consume() error = %v, want original source", err)
			}
			var typedCause *articleGateTestError
			if !errors.As(err, &typedCause) || typedCause != cause {
				t.Fatalf("Consume() error = %v, want typed original cause", err)
			}
		})
	}
}

func TestCoordinatorConsumesByRunIDAndConvergesOnlyReadyToConsume(t *testing.T) {
	t.Parallel()
	begunAt := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	run := RecoveryRun{
		RunID: "run-01", BackupRef: "backup-2026-10-20T00:00:00Z",
		JournalKeyID: "recovery", JournalKeyVersion: "v1",
		JournalSecretFingerprint: "hmac-sha256:synthetic", JournalStage: StageBegun,
		StartedAt: begunAt, Outcome: RecoveryOutcomeIdentityReady,
	}
	store := newMemoryRunStore(run)
	journal := &memoryJournal{entry: JournalEntry{
		Namespace: RecoveryNamespaceV1, Sequence: 1, RunID: run.RunID, BackupRef: run.BackupRef,
		KeyID: run.JournalKeyID, KeyVersion: run.JournalKeyVersion,
		SecretFingerprint: run.JournalSecretFingerprint, Stage: StageBegun, RecordedAt: begunAt,
	}}
	gate := &stubArticleDeletionGate{replay: legacyReadyRaw(t), read: legacyReadyRaw(t)}
	coordinator, err := newLegacyCoordinator(t, store, journal, gate, NewReplayStatusV1Decoder(nil), func() time.Time {
		return time.Date(2026, 10, 22, 4, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("newLegacyCoordinator(t, ) error = %v", err)
	}
	result, err := coordinator.Consume(context.Background(), "run-01")
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if result.Run.Outcome != RecoveryOutcomeSucceeded || journal.markCalls != 1 {
		t.Fatalf("result=%#v markCalls=%d", result, journal.markCalls)
	}

	store.run.Outcome = RecoveryOutcomeReadyToConsume
	store.run.JournalStage = StageBegun
	journal.entry.Stage = StageConsumed
	journal.entry.RecordedAt = begunAt.Add(time.Minute)
	store.run.JournalConsumedAt = nil
	journal.markCalls = 0
	if _, err := coordinator.Consume(context.Background(), "run-01"); err != nil {
		t.Fatalf("Consume(converge) error = %v", err)
	}
	if journal.markCalls != 0 {
		t.Fatalf("convergence called MarkConsumed %d times", journal.markCalls)
	}
}

func TestCoordinatorFinalStatesRevalidateArticleEvidenceFailureMatrix(t *testing.T) {
	t.Parallel()
	original := mustReadyFixture(t)
	for _, outcome := range []RecoveryOutcome{
		RecoveryOutcomeJournalConsumed,
		RecoveryOutcomeSucceeded,
	} {
		for _, test := range []struct {
			name string
			raw  []byte
		}{
			{name: "unavailable", raw: []byte(unavailableFixtureV1)},
			{name: "wrong backup", raw: mutateReady(t, func(ready *ReadyReplayStatusV1) {
				ready.BackupRef = "backup-other"
			})},
			{name: "wrong ref", raw: mutateReady(t, func(ready *ReadyReplayStatusV1) {
				ready.EvidenceRef = "evidence-other"
			})},
			{name: "digest drift", raw: mutateReady(t, func(ready *ReadyReplayStatusV1) {
				ready.JournalHeadSequence++
			})},
		} {
			t.Run(string(outcome)+"/"+test.name, func(t *testing.T) {
				run, journal := finalStateFixture(original, outcome)
				store := newMemoryRunStore(run)
				gate := &stubArticleDeletionGate{read: test.raw}
				coordinator, err := newLegacyCoordinator(t,
					store, journal, gate, NewReplayStatusV1Decoder(nil), time.Now,
				)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := coordinator.Consume(
					context.Background(), run.RunID,
				); !errors.Is(err, ErrRecoveryNotReady) {
					t.Fatalf("Consume() error = %v", err)
				}
				if store.run.Outcome != outcome {
					t.Fatalf("outcome changed to %q", store.run.Outcome)
				}
			})
		}
	}
}

func TestCoordinatorFinalStatesReturnRevalidatedCompleteEvidenceAndJournal(t *testing.T) {
	t.Parallel()
	original := mustReadyFixture(t)
	for _, outcome := range []RecoveryOutcome{
		RecoveryOutcomeJournalConsumed,
		RecoveryOutcomeSucceeded,
	} {
		t.Run(string(outcome), func(t *testing.T) {
			run, journal := finalStateFixture(original, outcome)
			gate := &stubArticleDeletionGate{read: legacyReadyRaw(t)}
			coordinator, err := newLegacyCoordinator(t,
				newMemoryRunStore(run), journal, gate, NewReplayStatusV1Decoder(nil), time.Now,
			)
			if err != nil {
				t.Fatal(err)
			}
			result, err := coordinator.Consume(context.Background(), run.RunID)
			if err != nil {
				t.Fatalf("Consume() error = %v", err)
			}
			if result.Evidence.InventoryScope != original.InventoryScope ||
				result.Evidence.JournalHeadSequence != original.JournalHeadSequence ||
				result.Journal.RunID != run.RunID || result.Journal.Stage != StageConsumed {
				t.Fatalf("result = %#v", result)
			}
			if gate.readCalls != 1 || journal.verifyCalls == 0 {
				t.Fatalf("read calls = %d, verify calls = %d", gate.readCalls, journal.verifyCalls)
			}
		})
	}
}

func TestCoordinatorRejectsEvidenceReferenceOrDigestDrift(t *testing.T) {
	t.Parallel()
	for _, mutate := range []func(*ReadyReplayStatusV1){
		func(ready *ReadyReplayStatusV1) { ready.EvidenceRef = "evidence-other" },
		func(ready *ReadyReplayStatusV1) { ready.JournalHeadSequence++ },
	} {
		run := RecoveryRun{
			RunID: "run-01", BackupRef: "backup-2026-10-20T00:00:00Z",
			JournalKeyID: "recovery", JournalKeyVersion: "v1",
			JournalSecretFingerprint: "hmac-sha256:synthetic", JournalStage: StageBegun,
			StartedAt: time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC),
			Outcome:   RecoveryOutcomeIdentityReady,
		}
		store := newMemoryRunStore(run)
		journal := &memoryJournal{entry: JournalEntry{
			RunID: run.RunID, BackupRef: run.BackupRef, KeyID: run.JournalKeyID,
			KeyVersion: run.JournalKeyVersion, SecretFingerprint: run.JournalSecretFingerprint,
			Stage: StageBegun, RecordedAt: run.StartedAt,
		}}
		gate := &stubArticleDeletionGate{
			replay: legacyReadyRaw(t),
			read:   mutateReady(t, mutate),
		}
		coordinator, err := newLegacyCoordinator(t, store, journal, gate, NewReplayStatusV1Decoder(nil), time.Now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := coordinator.Consume(context.Background(), run.RunID); !errors.Is(err, ErrRecoveryNotReady) {
			t.Fatalf("Consume(drift) error = %v", err)
		}
		if journal.markCalls != 0 {
			t.Fatalf("MarkConsumed calls = %d", journal.markCalls)
		}
	}
}

type deterministicAnchor struct {
	mu           sync.Mutex
	snapshot     AnchorSnapshot
	revision     string
	missing      bool
	nextCASErr   error
	nextLoad     *anchorLoadOverride
	loadAfterCAS *anchorLoadOverride
	revisionSeq  int
}

type anchorLoadOverride struct {
	snapshot AnchorSnapshot
	revision string
	err      error
}

func newDeterministicAnchor() *deterministicAnchor { return &deterministicAnchor{missing: true} }
func (a *deterministicAnchor) Load(context.Context, string) (AnchorSnapshot, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.nextLoad != nil {
		override := *a.nextLoad
		a.nextLoad = nil
		return override.snapshot, override.revision, override.err
	}
	if a.missing {
		return AnchorSnapshot{}, "", ErrAnchorMissing
	}
	return a.snapshot, a.revision, nil
}
func (a *deterministicAnchor) CompareAndSwap(_ context.Context, _ string, expected string, next AnchorSnapshot) (AnchorSnapshot, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.nextCASErr != nil {
		err := a.nextCASErr
		a.nextCASErr = nil
		return AnchorSnapshot{}, "", err
	}
	if (!a.missing && expected != a.revision) || (a.missing && expected != "") {
		return AnchorSnapshot{}, "", ErrAnchorConflict
	}
	a.missing = false
	a.revisionSeq++
	a.revision = string(rune('a' + a.revisionSeq))
	a.snapshot = next
	if a.loadAfterCAS != nil {
		a.nextLoad = a.loadAfterCAS
		a.loadAfterCAS = nil
	}
	return next, a.revision, nil
}
func (a *deterministicAnchor) overrideLoadAfterNextCAS(override anchorLoadOverride) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.loadAfterCAS = &override
}
func (a *deterministicAnchor) forceCommitted(head JournalHead) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.snapshot.Committed = head
	a.snapshot.Pending = nil
}

func (a *deterministicAnchor) forceSnapshot(snapshot AnchorSnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.missing = false
	if a.revision == "" {
		a.revision = "forced"
	}
	a.snapshot = snapshot
}

type stubArticleDeletionGate struct {
	replay, read           []byte
	replayErr, readErr     error
	replayCalls, readCalls int
}

type articleGateTestError struct {
	cause error
}

func (err *articleGateTestError) Error() string {
	return "article gate test error: " + err.cause.Error()
}

func (err *articleGateTestError) Unwrap() error {
	return err.cause
}

func (g *stubArticleDeletionGate) ReplayAndVerify(context.Context, string, string) ([]byte, error) {
	g.replayCalls++
	if g.replayErr != nil {
		return nil, g.replayErr
	}
	return append([]byte(nil), g.replay...), nil
}
func (g *stubArticleDeletionGate) ReadStatus(context.Context, string, string) ([]byte, error) {
	g.readCalls++
	if g.readErr != nil {
		return nil, g.readErr
	}
	return append([]byte(nil), g.read...), nil
}

type memoryJournal struct {
	source      *controlledFixture
	entry       JournalEntry
	begunAt     time.Time
	markCalls   int
	verifyCalls int
}

func (j *memoryJournal) Begin(context.Context, string, string, []byte) (JournalEntry, error) {
	return JournalEntry{}, errors.New("unused")
}
func (j *memoryJournal) Read(context.Context, string) (JournalEntry, error) { return j.entry, nil }
func (j *memoryJournal) VerifyAgainst(_ context.Context, run RecoveryRun) (JournalEntry, error) {
	j.verifyCalls++
	begunAt := j.begunAt
	if begunAt.IsZero() {
		begunAt = j.entry.RecordedAt
		j.begunAt = begunAt
	}
	if j.entry.RunID != run.RunID ||
		j.entry.BackupRef != run.BackupRef ||
		j.entry.KeyID != run.JournalKeyID ||
		j.entry.KeyVersion != run.JournalKeyVersion ||
		j.entry.SecretFingerprint != run.JournalSecretFingerprint ||
		j.entry.Stage != run.JournalStage ||
		!begunAt.Equal(run.StartedAt) {
		return JournalEntry{}, ErrJournalMismatch
	}
	if run.JournalStage == StageConsumed &&
		(run.JournalConsumedAt == nil ||
			!j.entry.RecordedAt.Equal(*run.JournalConsumedAt)) {
		return JournalEntry{}, ErrJournalMismatch
	}
	return j.entry, nil
}
func (j *memoryJournal) MarkConsumed(context.Context, string) (JournalEntry, error) {
	j.markCalls++

	if j.source != nil {
		j.source.complete.JournalConsumedAt = fixtureTime(j.entry.RecordedAt.Add(time.Minute))
		j.source.publishConsumed()
		_, j.entry, _ = snapshotRun(j.source.f.recovery, j.entry.RunID)
	} else {
		j.entry.Stage = StageConsumed
		j.entry.RecordedAt = j.entry.RecordedAt.Add(time.Minute)
	}
	return j.entry, nil
}

type memoryRunStore struct {
	mu  sync.Mutex
	run RecoveryRun
}

func newMemoryRunStore(run RecoveryRun) *memoryRunStore { return &memoryRunStore{run: run} }
func (s *memoryRunStore) Load(context.Context, string) (RecoveryRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.run, nil
}
func (s *memoryRunStore) Transition(_ context.Context, runID string, expected, next RecoveryOutcome, change RecoveryRunChange) (RecoveryRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run.RunID != runID || s.run.Outcome != expected {
		return RecoveryRun{}, ErrRecoveryRunConflict
	}
	s.run.Outcome = next
	if change.ArticlePublication != nil {
		s.run.ArticlePublicationRef = change.ArticlePublication.Ref
		s.run.ArticlePublicationDigest = change.ArticlePublication.Digest
	}
	if change.ArticleEvidence != nil {
		s.run.ArticleEvidenceVersion = change.ArticleEvidence.EvidenceVersion
		s.run.ArticleEvidenceDigest = change.ArticleEvidence.EvidenceDigest
		s.run.ArticleEvidenceRef = change.ArticleEvidence.EvidenceRef
		s.run.ArticleEvidenceCompletedAt = change.ArticleEvidenceCompletedAt
		s.run.ArticleNamespace = change.ArticleEvidence.Namespace
		s.run.ArticleReplayVerifiedAt = change.ChangedAt
	}
	if change.JournalConsumedAt != nil {
		s.run.JournalStage = StageConsumed
		s.run.JournalConsumedAt = change.JournalConsumedAt
	}
	if next == RecoveryOutcomeSucceeded {
		s.run.CompletedAt = change.ChangedAt
	}
	return s.run, nil
}

func newTestJournal(t *testing.T, path string, anchor JournalHeadAnchor, now time.Time, bootstrap bool, keys []Key) *FileRecoveryJournal {
	t.Helper()
	return newTestJournalWithRetirementVerifier(
		t, path, anchor, now, bootstrap, keys, nil,
	)
}
func newTestJournalWithRetirementVerifier(
	t *testing.T,
	path string,
	anchor JournalHeadAnchor,
	now time.Time,
	bootstrap bool,
	keys []Key,
	verifier KeyRetirementCapabilityVerifier,
) *FileRecoveryJournal {
	t.Helper()
	journal, err := NewFileRecoveryJournal(context.Background(), JournalOptions{ReadMaxBytes: 8 << 20, ReadMaxRecords: 100000,
		Path: path, FileMode: testJournalMode, Now: func() time.Time { return now },
		JournalID: "synthetic-journal", Epoch: 1, Anchor: anchor, Bootstrap: bootstrap,
		KeyRetirementVerifier: verifier,
	}, mustKeyring(t, keys))
	if err != nil {
		t.Fatalf("NewFileRecoveryJournal() error = %v", err)
	}
	return journal
}
func mustKeyring(t *testing.T, keys []Key) Keyring {
	t.Helper()
	keyring, err := NewKeyring(keys)
	if err != nil {
		t.Fatalf("NewKeyring() error = %v", err)
	}
	return keyring
}

type deterministicRetirementGrant struct {
	request  KeyRetirementRequest
	decision KeyRetirementVerification
	consumed bool
}

type deterministicRetirementVerifier struct {
	mu        sync.Mutex
	grants    map[string]deterministicRetirementGrant
	verifyErr error
	calls     int
}

func newDeterministicRetirementVerifier() *deterministicRetirementVerifier {
	return &deterministicRetirementVerifier{
		grants: make(map[string]deterministicRetirementGrant),
	}
}

func (verifier *deterministicRetirementVerifier) issue(
	opaque string,
	request KeyRetirementRequest,
	decision KeyRetirementVerification,
) KeyRetirementCapability {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	verifier.grants[opaque] = deterministicRetirementGrant{
		request: request, decision: decision,
	}
	return KeyRetirementCapability{opaque: opaque}
}

func (verifier *deterministicRetirementVerifier) VerifyAndConsume(
	_ context.Context,
	request KeyRetirementRequest,
	capability KeyRetirementCapability,
) (KeyRetirementVerification, error) {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	verifier.calls++
	if verifier.verifyErr != nil {
		return KeyRetirementVerificationIndeterminate, verifier.verifyErr
	}
	grant, ok := verifier.grants[capability.opaque]
	if !ok || grant.consumed || grant.request != request {
		return KeyRetirementVerificationRejected, nil
	}
	if grant.decision == KeyRetirementVerificationApproved {
		grant.consumed = true
		verifier.grants[capability.opaque] = grant
	}
	return grant.decision, nil
}

func (verifier *deterministicRetirementVerifier) callCount() int {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	return verifier.calls
}
func requirePOSIXJournal(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("FileRecoveryJournal is unsupported on Windows")
	}
}

func newJournalTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod journal dir: %v", err)
	}
	return dir
}
func readSingleRecord(t *testing.T, path string) diskRecord {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	record, err := decodeDiskRecord(raw[:len(raw)-1])
	if err != nil {
		t.Fatal(err)
	}
	return record
}
func writeSingleRecord(t *testing.T, path string, record diskRecord) {
	t.Helper()
	raw, err := encodeDiskRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, testJournalMode); err != nil {
		t.Fatal(err)
	}
}

func readAllRecords(t *testing.T, path string) []diskRecord {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSuffix(raw, []byte{'\n'}), []byte{'\n'})
	records := make([]diskRecord, 0, len(lines))
	for _, line := range lines {
		record, err := decodeDiskRecord(line)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func writeAllRecords(t *testing.T, path string, records []diskRecord) {
	t.Helper()
	raw, err := encodeDiskRecords(records)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, testJournalMode); err != nil {
		t.Fatal(err)
	}
}

func mutateReady(t *testing.T, mutate func(*ReadyReplayStatusV1)) []byte {
	t.Helper()
	var ready ReadyReplayStatusV1
	if err := json.Unmarshal(legacyReadyRaw(t), &ready); err != nil {
		t.Fatal(err)
	}
	mutate(&ready)
	digest, err := ready.CanonicalDigest()
	if err != nil {
		t.Fatal(err)
	}
	ready.EvidenceDigest = digest
	raw, err := json.Marshal(ready)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func anchorCrashFixture(
	t *testing.T,
) (*FileRecoveryJournal, *deterministicAnchor, string, []diskRecord, []diskRecord, []diskRecord) {
	t.Helper()
	path := filepath.Join(newJournalTestDir(t), "recovery.journal")
	anchor := newDeterministicAnchor()
	journal := newTestJournal(t, path, anchor, time.Date(2026, 10, 22, 2, 50, 0, 0, time.UTC), true, []Key{
		{ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key-v1")},
	})
	if _, err := journal.Begin(context.Background(), "run-old", "backup-old", []byte("synthetic-secret-old")); err != nil {
		t.Fatal(err)
	}
	oldRecords := readAllRecords(t, path)
	if _, err := journal.Begin(context.Background(), "run-new", "backup-new", []byte("synthetic-secret-new")); err != nil {
		t.Fatal(err)
	}
	newRecords := readAllRecords(t, path)
	thirdRecords := append([]diskRecord(nil), newRecords...)
	thirdRecords[1].RunID = "run-third"
	thirdRecords[1].BackupRef = "backup-third"
	thirdRecords[1].SecretFingerprint = fingerprintSecret(
		Key{Material: []byte("synthetic-key-v1")}, []byte("synthetic-secret-third"),
	)
	thirdRecords[1].RecordHMAC = computeRecordHMAC(
		thirdRecords[1], []byte("synthetic-key-v1"),
	)
	return journal, anchor, path, oldRecords, newRecords, thirdRecords
}

func mustReadyFixture(t *testing.T) ReadyReplayStatusV1 {
	t.Helper()
	status, err := DecodeReplayStatusV1([]byte(readyFixtureV1))
	if err != nil {
		t.Fatal(err)
	}
	v := *status.Ready
	v.JournalHeadHash = fixtureHash("legacy-test-head")
	v.EvidenceDigest = oracleReadyDigest(v)
	return v
}

func finalStateFixture(
	evidence ReadyReplayStatusV1,
	outcome RecoveryOutcome,
) (RecoveryRun, *memoryJournal) {
	startedAt := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)
	consumedAt := startedAt.Add(time.Minute)
	completedAt, _ := time.Parse(time.RFC3339, evidence.CompletedAt)
	run := RecoveryRun{
		RunID: evidence.RunID, BackupRef: evidence.BackupRef,
		JournalKeyID: "recovery", JournalKeyVersion: "v1",
		ArticlePublicationRef: "publication-test", ArticlePublicationDigest: fixtureHash("publication-test"),
		JournalSecretFingerprint: "hmac-sha256:synthetic",
		JournalStage:             StageConsumed, StartedAt: startedAt, JournalConsumedAt: &consumedAt,
		ArticleEvidenceVersion:     evidence.EvidenceVersion,
		ArticleEvidenceDigest:      evidence.EvidenceDigest,
		ArticleEvidenceRef:         evidence.EvidenceRef,
		ArticleEvidenceCompletedAt: canonicalTime(completedAt),
		ArticleNamespace:           evidence.Namespace,
		ArticleReplayVerifiedAt:    startedAt,
		Outcome:                    outcome,
	}
	if outcome == RecoveryOutcomeSucceeded {
		run.CompletedAt = startedAt.Add(2 * time.Minute)
	}
	journal := &memoryJournal{
		begunAt: startedAt,
		entry: JournalEntry{
			Namespace: RecoveryNamespaceV1, Sequence: 2,
			RunID: run.RunID, BackupRef: run.BackupRef,
			KeyID: run.JournalKeyID, KeyVersion: run.JournalKeyVersion,
			SecretFingerprint: run.JournalSecretFingerprint,
			Stage:             StageConsumed, RecordedAt: consumedAt,
		},
	}
	return run, journal
}

func newPortableRetirementJournal(port KeyRetirementCapabilityVerifier) *FileRecoveryJournal {
	anchor := newDeterministicAnchor()
	anchor.forceSnapshot(AnchorSnapshot{JournalID: "synthetic-retirement-journal", Namespace: RecoveryNamespaceV1, Epoch: 1})
	j := &FileRecoveryJournal{journalID: "synthetic-retirement-journal", epoch: 1, anchor: anchor, keyRetirementVerifier: port}
	installSyntheticWriterLock(j)
	return j
}
func lockedTestRetirement(ctx context.Context, j *FileRecoveryJournal, request KeyRetirementRequest, capability KeyRetirementCapability) error {
	return j.withLock(ctx, func(ctx context.Context) error {
		return j.verifyAndConsumeKeyRetirementCapability(ctx, request, capability)
	})
}

//go:build integration

package recovery_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"han/internal/recovery"
)

// s44FileJournalSupported mirrors the build constraint of
// internal/recovery/filelock_flock.go:
//
//	//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
//
// Go also satisfies the linux tag on GOOS=android, the darwin tag on
// GOOS=ios, and the solaris tag on GOOS=illumos, so those GOOS values use
// the same flock backend and must be listed here too.
func s44FileJournalSupported() bool {
	switch runtime.GOOS {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "linux", "netbsd", "openbsd", "solaris":
		return true
	default:
		return false
	}
}

// s44NotRunReason reports the CURRENT runtime reason a file-journal assertion
// was not executed on this non-flock platform, instead of a hard-coded claim.
func s44NotRunReason() string {
	return "file journal backend not available on runtime.GOOS=" + runtime.GOOS + " (requires flock-backed POSIX platform)"
}

// s44MemoryAnchor is a local in-memory JournalHeadAnchor test double. It is NOT
// the production anchor (the recovery store). In the POSIX branches only the
// file journal IO, chain, and HMAC validation are real; the anchor revisioning
// is simulated here so the real file journal can bootstrap and commit records.
type s44MemoryAnchor struct {
	mu       sync.Mutex
	snapshot recovery.AnchorSnapshot
	revision string
	missing  bool
	seq      int
}

func newS44MemoryAnchor() *s44MemoryAnchor { return &s44MemoryAnchor{missing: true} }

func (a *s44MemoryAnchor) Load(_ context.Context, _ string) (recovery.AnchorSnapshot, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.missing {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorMissing
	}
	return a.snapshot, a.revision, nil
}

func (a *s44MemoryAnchor) CompareAndSwap(_ context.Context, _ string, expected string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if (!a.missing && expected != a.revision) || (a.missing && expected != "") {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorConflict
	}
	a.missing = false
	a.seq++
	a.revision = "s44-anchor-r" + strconv.Itoa(a.seq)
	a.snapshot = next
	return next, a.revision, nil
}

func mustS44Keyring(t *testing.T, keys []recovery.Key) recovery.Keyring {
	t.Helper()
	keyring, err := recovery.NewKeyring(keys)
	if err != nil {
		t.Fatalf("NewKeyring error=%v", err)
	}
	return keyring
}

// s44NewFileJournal opens the REAL file-backed recovery journal in an
// owner-only temporary directory. It is only reached on POSIX platforms, and
// its anchor argument is normally the s44MemoryAnchor test double described
// above; only the journal IO/chain/HMAC behavior is production code.
func s44NewFileJournal(t *testing.T, dir string, anchor recovery.JournalHeadAnchor, now time.Time, bootstrap bool, keys []recovery.Key) *recovery.FileRecoveryJournal {
	t.Helper()
	journal, err := recovery.NewFileRecoveryJournal(context.Background(), recovery.JournalOptions{
		ReadMaxBytes: 8 << 20, ReadMaxRecords: 100000,
		Path: filepath.Join(dir, "recovery.journal"), FileMode: 0o600,
		Now: func() time.Time { return now }, JournalID: "s44-file-journal", Epoch: 1,
		Anchor: anchor, Bootstrap: bootstrap,
	}, mustS44Keyring(t, keys))
	if err != nil {
		t.Fatalf("NewFileRecoveryJournal error=%v", err)
	}
	return journal
}

func s44FixtureKeys(t *testing.T) (keyringFixture, []recovery.Key) {
	t.Helper()
	document := readFixture[keyringDocument](t, "keyring-fixtures.json")
	if len(document.Fixtures) != 1 || document.Fixtures[0].Name != "rotated_complete" || len(document.Fixtures[0].Keys) != 3 {
		t.Fatalf("keyring fixture shape mismatch: fixtures=%d", len(document.Fixtures))
	}
	fixture := document.Fixtures[0]
	keys := make([]recovery.Key, 0, len(fixture.Keys))
	for _, key := range fixture.Keys {
		keys = append(keys, recovery.Key{
			ID: key.ID, Version: key.Version, State: recovery.KeyState(key.State), Material: []byte(key.Material),
		})
	}
	return fixture, keys
}

func TestIdentityRecovery_KeyringAndJournalBindings_FailClosed(t *testing.T) {
	t.Run("KeyringRotationFingerprintMatrix", func(t *testing.T) {
		fixture, keys := s44FixtureKeys(t)
		if _, err := recovery.NewKeyring(keys); err != nil {
			t.Fatalf("rotated_complete keyring rejected: %v", err)
		}
		seen := map[string]bool{}
		for _, vector := range fixture.TestVectors {
			var material string
			for _, key := range fixture.Keys {
				if key.Version == vector.ExpectedKeyVersion {
					material = key.Material
				}
			}
			if material == "" {
				t.Fatalf("fixture vector %s references unknown version %s", vector.Name, vector.ExpectedKeyVersion)
			}
			fingerprint := syntheticFingerprint(material, vector.SyntheticRecoveryInput)
			if fingerprint != vector.ExpectedFingerprint {
				t.Fatalf("vector %s fingerprint=%s; want %s", vector.Name, fingerprint, vector.ExpectedFingerprint)
			}
			if seen[fingerprint] {
				t.Fatalf("cross-version fingerprints collided for %s", vector.Name)
			}
			seen[fingerprint] = true
		}
		if len(seen) != 3 {
			t.Fatalf("distinct cross-version fingerprints=%d; want 3", len(seen))
		}

		active := func() recovery.Key {
			return recovery.Key{ID: "recovery-fixture", Version: "v3", State: recovery.KeyStateActive, Material: []byte("material-v3")}
		}
		verifyOnly := func() recovery.Key {
			return recovery.Key{ID: "recovery-fixture", Version: "v2", State: recovery.KeyStateVerifyOnly, Material: []byte("material-v2")}
		}
		cases := []struct {
			name string
			keys []recovery.Key
		}{
			{"no_keys", nil},
			{"two_active", []recovery.Key{active(), {ID: "second", Version: "v3", State: recovery.KeyStateActive, Material: []byte("material")}}},
			{"no_active", []recovery.Key{verifyOnly()}},
			{"empty_material", []recovery.Key{{ID: "recovery-fixture", Version: "v3", State: recovery.KeyStateActive}}},
			{"duplicate_ref", []recovery.Key{active(), {ID: "recovery-fixture", Version: "v3", State: recovery.KeyStateVerifyOnly, Material: []byte("material")}}},
			{"invalid_state", []recovery.Key{{ID: "recovery-fixture", Version: "v3", State: recovery.KeyState("retired"), Material: []byte("material")}}},
			{"blank_id", []recovery.Key{{ID: " ", Version: "v3", State: recovery.KeyStateActive, Material: []byte("material")}}},
			{"control_version", []recovery.Key{{ID: "recovery-fixture", Version: "v3\n", State: recovery.KeyStateActive, Material: []byte("material")}}},
		}
		for _, tc := range cases {
			if _, err := recovery.NewKeyring(tc.keys); !errors.Is(err, recovery.ErrInvalidKeyring) {
				t.Fatalf("%s err=%v; want ErrInvalidKeyring", tc.name, err)
			}
		}
		t.Logf("keyring matrix: active=v3 verify_only=v2,v1 fingerprints=3 distinct negatives=%d all ErrInvalidKeyring", len(cases))
	})

	t.Run("SoftwareJournalSecretFingerprintAndStages", func(t *testing.T) {
		now := func() time.Time { return s417Canonical(time.Now()) }
		journal := s417NewSoftwareJournal("s44-key-fingerprint-journal", recovery.RecoveryNamespaceV1, 1, now)
		secret := []byte("s44-controlled-secret")
		entry, err := journal.Begin(context.Background(), "s44-key-run", "s44-key-backup", secret)
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		if entry.SecretFingerprint != s417HMAC(journal.key, secret) {
			t.Fatalf("secret fingerprint=%s; want active-key HMAC", entry.SecretFingerprint)
		}
		if entry.RecordHMAC != s417HMAC(journal.key, []byte(s417ChainHash("", entry)+"|"+entry.SecretFingerprint)) {
			t.Fatalf("record HMAC does not bind chain hash and secret fingerprint")
		}
		if _, err := journal.Begin(context.Background(), "s44-key-run", "s44-key-backup", secret); !errors.Is(err, recovery.ErrRunAlreadyExists) {
			t.Fatalf("duplicate Begin err=%v; want ErrRunAlreadyExists", err)
		}
		binding := recovery.RunBinding{RunID: "s44-key-run", BackupRef: "s44-key-backup"}
		verified, err := journal.VerifyBeginSecret(context.Background(), binding, secret)
		if err != nil || verified.Stage != recovery.StageBegun {
			t.Fatalf("VerifyBeginSecret correct err=%v stage=%s; want begun", err, verified.Stage)
		}
		if _, err := journal.VerifyBeginSecret(context.Background(), binding, []byte("s44-wrong-secret")); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("VerifyBeginSecret wrong secret err=%v; want ErrAdmissionBinding", err)
		}
		if _, err := journal.VerifyBeginSecret(context.Background(), recovery.RunBinding{RunID: "s44-missing-run"}, secret); !errors.Is(err, recovery.ErrAdmissionIncomplete) {
			t.Fatalf("VerifyBeginSecret missing run err=%v; want ErrAdmissionIncomplete", err)
		}
		consumed, err := journal.MarkConsumed(context.Background(), "s44-key-run")
		if err != nil || consumed.Stage != recovery.StageConsumed {
			t.Fatalf("MarkConsumed err=%v stage=%s; want consumed", err, consumed.Stage)
		}
		t.Logf("software journal fingerprint binds active key; wrong secret and missing run fail closed")
	})

	t.Run("CrossVersionReplay_PersistentVolume", func(t *testing.T) {
		_, keys := s44FixtureKeys(t)
		if _, err := recovery.NewKeyring(keys); err != nil {
			t.Fatalf("keyring failed: %v", err)
		}
		if !s44FileJournalSupported() {
			_, err := recovery.NewFileRecoveryJournal(context.Background(), recovery.JournalOptions{
				Path: "/s44/unavailable/journal", FileMode: 0o600, JournalID: "s44-file-journal", Epoch: 1,
				Now: func() time.Time { return s417Canonical(time.Now()) }, ReadMaxBytes: 1 << 20, ReadMaxRecords: 1024,
			}, mustS44Keyring(t, keys))
			if !errors.Is(err, recovery.ErrUnsupportedPlatform) {
				t.Fatalf("NewFileRecoveryJournal err=%v; want ErrUnsupportedPlatform", err)
			}
			t.Logf("branch=non-posix runtime.GOOS=%s result=NewFileRecoveryJournal ErrUnsupportedPlatform asserted; retained cross-version replay NOT RUN: %s", runtime.GOOS, s44NotRunReason())
			return
		}
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod journal dir: %v", err)
		}
		anchor := newS44MemoryAnchor()
		now := s417Canonical(time.Now())
		secret := []byte("s44-cross-version-secret")
		v1 := s44NewFileJournal(t, dir, anchor, now, true, []recovery.Key{
			{ID: "s44-recovery", Version: "v1", State: recovery.KeyStateActive, Material: []byte("s44-material-v1")},
		})
		if _, err := v1.Begin(context.Background(), "s44-cross-run-v1", "s44-cross-backup-v1", secret); err != nil {
			t.Fatalf("v1 Begin error=%v", err)
		}
		if _, err := v1.MarkConsumed(context.Background(), "s44-cross-run-v1"); err != nil {
			t.Fatalf("v1 MarkConsumed error=%v", err)
		}
		v2 := s44NewFileJournal(t, dir, anchor, now.Add(time.Minute), false, []recovery.Key{
			{ID: "s44-recovery", Version: "v2", State: recovery.KeyStateActive, Material: []byte("s44-material-v2")},
			{ID: "s44-recovery", Version: "v1", State: recovery.KeyStateVerifyOnly, Material: []byte("s44-material-v1")},
		})
		if _, err := v2.Begin(context.Background(), "s44-cross-run-replay", "s44-cross-backup-v2", secret); !errors.Is(err, recovery.ErrSecretReused) {
			t.Fatalf("cross-version replay Begin err=%v; want ErrSecretReused", err)
		}
		t.Logf("branch=posix runtime.GOOS=%s result=v1-secret/v2-keyring replay rejected with ErrSecretReused (real file journal under temp dir; anchor=s44MemoryAnchor test double)", runtime.GOOS)
	})

	t.Run("VerifyOnlyKeyRetention_PerPlatform", func(t *testing.T) {
		_, keys := s44FixtureKeys(t)
		keyring, err := recovery.NewKeyring(keys)
		if err != nil {
			t.Fatalf("keyring failed: %v", err)
		}
		verifyOnlyOnly := []recovery.Key{keys[1], keys[2]}
		if _, err := recovery.NewKeyring(verifyOnlyOnly); !errors.Is(err, recovery.ErrInvalidKeyring) {
			t.Fatalf("verify-only-only keyring err=%v; want ErrInvalidKeyring", err)
		}
		if !s44FileJournalSupported() {
			_, err = recovery.OpenVerifiedRecoveryJournal(context.Background(), recovery.ReadOptions{Keyring: keyring})
			if !errors.Is(err, recovery.ErrAdmissionUnsupported) || !errors.Is(err, recovery.ErrUnsupportedPlatform) {
				t.Fatalf("OpenVerifiedRecoveryJournal err=%v; want ErrAdmissionUnsupported+ErrUnsupportedPlatform", err)
			}
			t.Logf("branch=non-posix runtime.GOOS=%s result=verify-only keyring ErrInvalidKeyring and OpenVerifiedRecoveryJournal ErrAdmissionUnsupported+ErrUnsupportedPlatform asserted; production key retirement NOT RUN (identity-auth-S4.6 not implemented): %s", runtime.GOOS, s44NotRunReason())
			return
		}
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod journal dir: %v", err)
		}
		anchor := newS44MemoryAnchor()
		now := s417Canonical(time.Now())
		writer := s44NewFileJournal(t, dir, anchor, now, true, []recovery.Key{
			{ID: "s44-recovery", Version: "v1", State: recovery.KeyStateActive, Material: []byte("s44-material-v1")},
		})
		if _, err := writer.Begin(context.Background(), "s44-retire-run", "s44-retire-backup", []byte("s44-retire-secret")); err != nil {
			t.Fatalf("Begin error=%v", err)
		}
		if _, err := writer.MarkConsumed(context.Background(), "s44-retire-run"); err != nil {
			t.Fatalf("MarkConsumed error=%v", err)
		}
		reader, err := recovery.OpenVerifiedRecoveryJournal(context.Background(), recovery.ReadOptions{
			Path: filepath.Join(dir, "recovery.journal"), JournalID: "s44-file-journal", FileMode: 0o600,
			Epoch: 1, Anchor: anchor, Keyring: mustS44Keyring(t, []recovery.Key{
				{ID: "s44-recovery", Version: "v2", State: recovery.KeyStateActive, Material: []byte("s44-material-v2")},
				{ID: "s44-recovery", Version: "v1", State: recovery.KeyStateVerifyOnly, Material: []byte("s44-material-v1")},
			}), MaxBytes: 8 << 20, MaxRecords: 100000,
		})
		if err != nil {
			t.Fatalf("OpenVerifiedRecoveryJournal(rotated keyring) error=%v", err)
		}
		snapshot, err := reader.ReadVerifiedSnapshot(context.Background(), recovery.RecoveryNamespaceV1)
		if err != nil {
			t.Fatalf("verified read with v1 verify-only key error=%v", err)
		}
		if snapshot.Head.Sequence != 2 {
			t.Fatalf("verified read head=%d; want 2", snapshot.Head.Sequence)
		}
		if _, err := reader.Begin(context.Background(), "s44-retire-replay", "s44-retire-backup", []byte("s44-retire-secret")); !errors.Is(err, recovery.ErrAdmissionClosed) {
			t.Fatalf("verified reader Begin err=%v; want ErrAdmissionClosed", err)
		}
		t.Logf("branch=posix runtime.GOOS=%s result=verify-only read head=%d and Begin=ErrAdmissionClosed asserted (real file journal; anchor=s44MemoryAnchor test double); production key retirement NOT RUN (identity-auth-S4.6 not implemented)", runtime.GOOS, snapshot.Head.Sequence)
	})

	t.Run("ConstructorValidation_PerPlatform", func(t *testing.T) {
		if s44FileJournalSupported() {
			_, err := recovery.NewFileRecoveryJournal(context.Background(), recovery.JournalOptions{}, recovery.Keyring{})
			if !errors.Is(err, recovery.ErrJournalCorrupt) {
				t.Fatalf("POSIX NewFileRecoveryJournal(empty options) err=%v; want ErrJournalCorrupt", err)
			}
			_, err = recovery.OpenVerifiedRecoveryJournal(context.Background(), recovery.ReadOptions{})
			if !errors.Is(err, recovery.ErrAdmissionBinding) {
				t.Fatalf("POSIX OpenVerifiedRecoveryJournal(empty options) err=%v; want ErrAdmissionBinding", err)
			}
			t.Logf("branch=posix runtime.GOOS=%s result=NewFileRecoveryJournal(empty)=ErrJournalCorrupt and OpenVerifiedRecoveryJournal(empty)=ErrAdmissionBinding asserted; non-POSIX ErrUnsupportedPlatform branch not applicable", runtime.GOOS)
			return
		}
		_, err := recovery.NewFileRecoveryJournal(context.Background(), recovery.JournalOptions{}, recovery.Keyring{})
		if !errors.Is(err, recovery.ErrUnsupportedPlatform) {
			t.Fatalf("NewFileRecoveryJournal err=%v; want ErrUnsupportedPlatform", err)
		}
		_, err = recovery.OpenVerifiedRecoveryJournal(context.Background(), recovery.ReadOptions{})
		if !errors.Is(err, recovery.ErrAdmissionUnsupported) || !errors.Is(err, recovery.ErrUnsupportedPlatform) {
			t.Fatalf("OpenVerifiedRecoveryJournal err=%v; want ErrAdmissionUnsupported+ErrUnsupportedPlatform", err)
		}
		t.Logf("branch=non-posix runtime.GOOS=%s result=NewFileRecoveryJournal=ErrUnsupportedPlatform and OpenVerifiedRecoveryJournal=ErrAdmissionUnsupported+ErrUnsupportedPlatform asserted; POSIX constructor validation NOT RUN: %s", runtime.GOOS, s44NotRunReason())
	})
}

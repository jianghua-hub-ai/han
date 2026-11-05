//go:build windows

package recovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileRecoveryJournalWindowsUnsupportedHasZeroExternalChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recovery.journal")
	anchor := &countingAnchor{}
	keyring, err := NewKeyring([]Key{{
		ID: "recovery", Version: "v1", State: KeyStateActive, Material: []byte("synthetic-key"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewFileRecoveryJournal(context.Background(), JournalOptions{ReadMaxBytes: 8 << 20, ReadMaxRecords: 100000,
		Path: path, FileMode: 0o600, Now: time.Now,
		JournalID: "synthetic-journal", Epoch: 1, Anchor: anchor, Bootstrap: true,
	}, keyring)
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("error = %v", err)
	}
	if anchor.calls != 0 {
		t.Fatalf("anchor calls = %d", anchor.calls)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("external files = %v", entries)
	}
}

type countingAnchor struct{ calls int }

func (a *countingAnchor) Load(context.Context, string) (AnchorSnapshot, string, error) {
	a.calls++
	return AnchorSnapshot{}, "", nil
}
func (a *countingAnchor) CompareAndSwap(context.Context, string, string, AnchorSnapshot) (AnchorSnapshot, string, error) {
	a.calls++
	return AnchorSnapshot{}, "", nil
}

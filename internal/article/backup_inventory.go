package article

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupInventoryV1Version is the only supported inventory manifest version.
const BackupInventoryV1Version = "article-backup-inventory/v1"

// BackupInventoryScopeV1 is the only supported inventory scope.
const BackupInventoryScopeV1 = "recoverable-backups-before-cutoff"

// backupInventoryLimitBytes bounds a single manifest read.
const backupInventoryLimitBytes = 1 << 20

var (
	// ErrBackupInventoryUnavailable marks any unreadable or unsafe manifest.
	ErrBackupInventoryUnavailable = errors.New("article backup inventory unavailable")
	// ErrBackupInventoryInvalid marks a syntactically or semantically invalid manifest.
	ErrBackupInventoryInvalid = errors.New("article backup inventory invalid")
)

// BackupInventoryEntry is one recoverable backup fact.
type BackupInventoryEntry struct {
	BackupRef         string
	BackupCompletedAt time.Time
}

// BackupInventory is one verified production inventory snapshot.
type BackupInventory struct {
	Version         string
	Scope           string
	BackupRef       string
	BackupCutoff    time.Time
	GeneratedAt     time.Time
	Recoverable     []BackupInventoryEntry
	InventoryDigest string
}

// FileBackupInventoryReader reads one controlled read-only manifest.
type FileBackupInventoryReader struct {
	path string
}

// NewFileBackupInventoryReader validates the mandatory absolute clean path.
func NewFileBackupInventoryReader(path string) (*FileBackupInventoryReader, error) {
	if path == "" || path != strings.TrimSpace(path) || path != filepath.Clean(path) ||
		(!filepath.IsAbs(path) && !strings.HasPrefix(path, string(filepath.Separator))) {
		return nil, ErrBackupInventoryInvalid
	}
	return &FileBackupInventoryReader{path: path}, nil
}

// Read reopens and re-verifies the manifest on every call (no caching).
func (reader *FileBackupInventoryReader) Read(_ context.Context) (BackupInventory, error) {
	if reader == nil || reader.path == "" {
		return BackupInventory{}, ErrBackupInventoryUnavailable
	}
	raw, err := reader.readSecureBytes()
	if err != nil {
		return BackupInventory{}, err
	}
	return parseBackupInventory(raw)
}

// readSecureBytes performs lstat -> no-follow open -> fstat -> bounded read ->
// fstat -> lstat and rejects any identity/permission/size drift.
func (reader *FileBackupInventoryReader) readSecureBytes() ([]byte, error) {
	before, err := os.Lstat(reader.path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, ErrBackupInventoryUnavailable
	}
	if !backupInventoryModeSafe(before.Mode()) {
		return nil, ErrBackupInventoryUnavailable
	}
	file, err := os.Open(reader.path)
	if err != nil {
		return nil, ErrBackupInventoryUnavailable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !backupInventoryModeSafe(opened.Mode()) {
		return nil, ErrBackupInventoryUnavailable
	}
	if !os.SameFile(before, opened) {
		return nil, ErrBackupInventoryUnavailable
	}
	if opened.Size() > backupInventoryLimitBytes {
		return nil, ErrBackupInventoryUnavailable
	}
	raw, err := io.ReadAll(io.LimitReader(file, backupInventoryLimitBytes+1))
	if err != nil {
		return nil, ErrBackupInventoryUnavailable
	}
	if int64(len(raw)) > backupInventoryLimitBytes {
		return nil, ErrBackupInventoryUnavailable
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(opened, after) || after.Size() != opened.Size() {
		return nil, ErrBackupInventoryUnavailable
	}
	final, err := os.Lstat(reader.path)
	if err != nil || !os.SameFile(after, final) {
		return nil, ErrBackupInventoryUnavailable
	}
	return raw, nil
}

// backupInventoryModeSafe rejects any write mode bit. On platforms where the
// standard library cannot prove POSIX owner/ACL semantics the check is skipped
// to match the existing Article deletion journal platform contract.
func backupInventoryModeSafe(mode fs.FileMode) bool {
	if !articleDeletionJournalPlatformSupported() {
		return true
	}
	return mode.Perm()&0o222 == 0
}

type backupInventoryManifest struct {
	InventoryVersion   string                    `json:"inventory_version"`
	InventoryScope     string                    `json:"inventory_scope"`
	BackupRef          string                    `json:"backup_ref"`
	BackupCutoff       string                    `json:"backup_cutoff"`
	GeneratedAt        string                    `json:"generated_at"`
	RecoverableBackups []backupInventoryEntryRaw `json:"recoverable_backups"`
	InventoryDigest    string                    `json:"inventory_digest"`
}

type backupInventoryEntryRaw struct {
	BackupRef         string `json:"backup_ref"`
	BackupCompletedAt string `json:"backup_completed_at"`
}

func parseBackupInventory(raw []byte) (BackupInventory, error) {
	if len(raw) == 0 || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	var manifest backupInventoryManifest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	canonical, err := json.Marshal(manifest)
	if err != nil || !bytes.Equal(canonical, raw) {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	if manifest.InventoryVersion != BackupInventoryV1Version ||
		manifest.InventoryScope != BackupInventoryScopeV1 {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	cutoff, err := parseInventoryTime(manifest.BackupCutoff)
	if err != nil {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	generatedAt, err := parseInventoryTime(manifest.GeneratedAt)
	if err != nil || generatedAt.Before(cutoff) {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	entries, err := parseBackupInventoryEntries(manifest.RecoverableBackups, manifest.BackupRef, cutoff)
	if err != nil {
		return BackupInventory{}, err
	}
	expected, err := backupInventoryDigest(manifest)
	if err != nil {
		return BackupInventory{}, err
	}
	if manifest.InventoryDigest != expected {
		return BackupInventory{}, ErrBackupInventoryInvalid
	}
	return BackupInventory{
		Version:         manifest.InventoryVersion,
		Scope:           manifest.InventoryScope,
		BackupRef:       manifest.BackupRef,
		BackupCutoff:    cutoff,
		GeneratedAt:     generatedAt,
		Recoverable:     entries,
		InventoryDigest: manifest.InventoryDigest,
	}, nil
}

func parseBackupInventoryEntries(
	rawEntries []backupInventoryEntryRaw,
	backupRef string,
	cutoff time.Time,
) ([]BackupInventoryEntry, error) {
	if len(rawEntries) == 0 {
		return nil, ErrBackupInventoryInvalid
	}
	entries := make([]BackupInventoryEntry, 0, len(rawEntries))
	seen := make(map[string]struct{}, len(rawEntries))
	matched := 0
	for _, raw := range rawEntries {
		if raw.BackupRef == "" || raw.BackupRef != strings.TrimSpace(raw.BackupRef) {
			return nil, ErrBackupInventoryInvalid
		}
		completedAt, err := parseInventoryTime(raw.BackupCompletedAt)
		if err != nil {
			return nil, ErrBackupInventoryInvalid
		}
		if completedAt.After(cutoff) {
			return nil, ErrBackupInventoryInvalid
		}
		if raw.BackupRef == backupRef {
			matched++
			if !completedAt.Equal(cutoff) {
				return nil, ErrBackupInventoryInvalid
			}
		}
		if _, ok := seen[raw.BackupRef]; ok {
			return nil, ErrBackupInventoryInvalid
		}
		seen[raw.BackupRef] = struct{}{}
		entries = append(entries, BackupInventoryEntry{
			BackupRef:         raw.BackupRef,
			BackupCompletedAt: completedAt,
		})
	}
	if matched != 1 {
		return nil, ErrBackupInventoryInvalid
	}
	if !sort.SliceIsSorted(entries, func(i, j int) bool {
		if entries[i].BackupCompletedAt.Equal(entries[j].BackupCompletedAt) {
			return entries[i].BackupRef < entries[j].BackupRef
		}
		return entries[i].BackupCompletedAt.Before(entries[j].BackupCompletedAt)
	}) {
		return nil, ErrBackupInventoryInvalid
	}
	return entries, nil
}

// parseInventoryTime requires canonical UTC with exactly six fractional digits.
func parseInventoryTime(raw string) (time.Time, error) {
	const layout = "2006-01-02T15:04:05.000000Z"
	parsed, err := time.Parse(layout, raw)
	if err != nil || parsed.Location() != time.UTC {
		return time.Time{}, ErrBackupInventoryInvalid
	}
	if parsed.Format(layout) != raw {
		return time.Time{}, ErrBackupInventoryInvalid
	}
	return parsed, nil
}

// backupInventoryDigest hashes the canonical top-level object minus the digest.
func backupInventoryDigest(manifest backupInventoryManifest) (string, error) {
	payload := struct {
		InventoryVersion   string                    `json:"inventory_version"`
		InventoryScope     string                    `json:"inventory_scope"`
		BackupRef          string                    `json:"backup_ref"`
		BackupCutoff       string                    `json:"backup_cutoff"`
		GeneratedAt        string                    `json:"generated_at"`
		RecoverableBackups []backupInventoryEntryRaw `json:"recoverable_backups"`
	}{
		InventoryVersion:   manifest.InventoryVersion,
		InventoryScope:     manifest.InventoryScope,
		BackupRef:          manifest.BackupRef,
		BackupCutoff:       manifest.BackupCutoff,
		GeneratedAt:        manifest.GeneratedAt,
		RecoverableBackups: manifest.RecoverableBackups,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode backup inventory digest payload: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

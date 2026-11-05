package article

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileBackupInventoryReader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	cutoff := "2026-10-20T00:00:00.000000Z"
	generatedAt := "2026-10-20T00:00:01.000000Z"
	entry := func(ref string, at string) map[string]string {
		return map[string]string{"backup_ref": ref, "backup_completed_at": at}
	}
	build := func(entries []map[string]string, ref string, cutoffValue string) string {
		manifest := map[string]any{
			"inventory_version":   BackupInventoryV1Version,
			"inventory_scope":     BackupInventoryScopeV1,
			"backup_ref":          ref,
			"backup_cutoff":       cutoffValue,
			"generated_at":        generatedAt,
			"recoverable_backups": entries,
		}
		digest, err := canonicalInventoryDigest(t, manifest)
		if err != nil {
			t.Fatalf("canonicalInventoryDigest() error = %v", err)
		}
		manifest["inventory_digest"] = digest
		return marshalOrdered(t, manifest)
	}
	valid := build(
		[]map[string]string{
			entry("backup-old", "2026-10-19T00:00:00.000000Z"),
			entry("backup-current", cutoff),
		},
		"backup-current",
		cutoff,
	)

	t.Run("valid", func(t *testing.T) {
		writeInventory(t, path, valid)
		reader, err := NewFileBackupInventoryReader(path)
		if err != nil {
			t.Fatalf("NewFileBackupInventoryReader() error = %v", err)
		}
		inventory, err := reader.Read(context.Background())
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		if inventory.BackupRef != "backup-current" ||
			inventory.Scope != BackupInventoryScopeV1 ||
			!inventory.BackupCutoff.Equal(mustInventoryTime(t, cutoff)) ||
			len(inventory.Recoverable) != 2 {
			t.Fatalf("inventory = %#v", inventory)
		}
	})

	t.Run("missing file fails closed", func(t *testing.T) {
		reader, err := NewFileBackupInventoryReader(filepath.Join(dir, "absent.json"))
		if err != nil {
			t.Fatalf("NewFileBackupInventoryReader() error = %v", err)
		}
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want unavailable")
		}
	})

	t.Run("relative path rejected", func(t *testing.T) {
		if _, err := NewFileBackupInventoryReader("relative.json"); err == nil {
			t.Fatal("NewFileBackupInventoryReader() error = nil, want invalid")
		}
	})

	t.Run("symlink rejected", func(t *testing.T) {
		target := filepath.Join(dir, "target.json")
		writeInventory(t, target, valid)
		link := filepath.Join(dir, "link.json")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unsupported: %v", err)
		}
		reader, err := NewFileBackupInventoryReader(link)
		if err != nil {
			t.Fatalf("NewFileBackupInventoryReader() error = %v", err)
		}
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want symlink rejection")
		}
	})

	t.Run("world writable rejected", func(t *testing.T) {
		if !articleDeletionJournalPlatformSupported() {
			t.Skip("POSIX write-mode semantics unavailable on this platform")
		}
		writeInventory(t, path, valid)
		if err := os.Chmod(path, 0o666); err != nil {
			t.Fatalf("Chmod() error = %v", err)
		}
		reader, _ := NewFileBackupInventoryReader(path)
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want write-mode rejection")
		}
	})

	t.Run("digest mismatch rejected", func(t *testing.T) {
		bad := build(
			[]map[string]string{
				entry("backup-old", "2026-10-19T00:00:00.000000Z"),
				entry("backup-current", cutoff),
			},
			"backup-current",
			cutoff,
		)
		bad = replaceDigest(t, bad, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
		writeInventory(t, path, bad)
		reader, _ := NewFileBackupInventoryReader(path)
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want digest mismatch")
		}
	})

	t.Run("non canonical time rejected", func(t *testing.T) {
		bad := build(
			[]map[string]string{
				entry("backup-current", "2026-10-20T00:00:00Z"),
			},
			"backup-current",
			"2026-10-20T00:00:00Z",
		)
		writeInventory(t, path, bad)
		reader, _ := NewFileBackupInventoryReader(path)
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want non-canonical time rejection")
		}
	})

	t.Run("entry after cutoff rejected", func(t *testing.T) {
		bad := build(
			[]map[string]string{
				entry("backup-current", cutoff),
				entry("backup-future", "2026-10-21T00:00:00.000000Z"),
			},
			"backup-current",
			cutoff,
		)
		writeInventory(t, path, bad)
		reader, _ := NewFileBackupInventoryReader(path)
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want post-cutoff rejection")
		}
	})

	t.Run("wrong version rejected", func(t *testing.T) {
		bad := build(
			[]map[string]string{entry("backup-current", cutoff)},
			"backup-current",
			cutoff,
		)
		bad = replaceField(t, bad, "inventory_version", "article-backup-inventory/v2")
		writeInventory(t, path, bad)
		reader, _ := NewFileBackupInventoryReader(path)
		if _, err := reader.Read(context.Background()); err == nil {
			t.Fatal("Read() error = nil, want version rejection")
		}
	})

	t.Run("freshness reopens snapshot", func(t *testing.T) {
		writeInventory(t, path, valid)
		reader, _ := NewFileBackupInventoryReader(path)
		if _, err := reader.Read(context.Background()); err != nil {
			t.Fatalf("first Read() error = %v", err)
		}
		replaced := build(
			[]map[string]string{entry("backup-current", cutoff)},
			"backup-current",
			cutoff,
		)
		writeInventory(t, path, replaced)
		inventory, err := reader.Read(context.Background())
		if err != nil {
			t.Fatalf("second Read() error = %v", err)
		}
		if len(inventory.Recoverable) != 1 {
			t.Fatalf("reader cached a stale snapshot: %#v", inventory)
		}
	})
}

func mustInventoryTime(t *testing.T, raw string) time.Time {
	t.Helper()
	parsed, err := parseInventoryTime(raw)
	if err != nil {
		t.Fatalf("parseInventoryTime(%q) error = %v", raw, err)
	}
	return parsed
}

var _ = os.Getenv

// marshalOrdered encodes the manifest in the fixed field order with no spaces.
func marshalOrdered(t *testing.T, manifest map[string]any) string {
	t.Helper()
	entriesRaw, _ := manifest["recoverable_backups"].([]map[string]string)
	var builder strings.Builder
	builder.WriteString("{")
	builder.WriteString(jsonString("inventory_version") + ":" + jsonString(manifest["inventory_version"].(string)) + ",")
	builder.WriteString(jsonString("inventory_scope") + ":" + jsonString(manifest["inventory_scope"].(string)) + ",")
	builder.WriteString(jsonString("backup_ref") + ":" + jsonString(manifest["backup_ref"].(string)) + ",")
	builder.WriteString(jsonString("backup_cutoff") + ":" + jsonString(manifest["backup_cutoff"].(string)) + ",")
	builder.WriteString(jsonString("generated_at") + ":" + jsonString(manifest["generated_at"].(string)) + ",")
	builder.WriteString(jsonString("recoverable_backups") + ":[")
	for index, entry := range entriesRaw {
		if index > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("{")
		builder.WriteString(jsonString("backup_ref") + ":" + jsonString(entry["backup_ref"]) + ",")
		builder.WriteString(jsonString("backup_completed_at") + ":" + jsonString(entry["backup_completed_at"]))
		builder.WriteString("}")
	}
	builder.WriteString("],")
	builder.WriteString(jsonString("inventory_digest") + ":" + jsonString(manifest["inventory_digest"].(string)))
	builder.WriteString("}")
	return builder.String()
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// canonicalInventoryDigest recomputes the digest over the canonical payload.
func canonicalInventoryDigest(t *testing.T, manifest map[string]any) (string, error) {
	t.Helper()
	entriesRaw, _ := manifest["recoverable_backups"].([]map[string]string)
	entries := make([]backupInventoryEntryRaw, 0, len(entriesRaw))
	for _, entry := range entriesRaw {
		entries = append(entries, backupInventoryEntryRaw{
			BackupRef:         entry["backup_ref"],
			BackupCompletedAt: entry["backup_completed_at"],
		})
	}
	return backupInventoryDigest(backupInventoryManifest{
		InventoryVersion:   manifest["inventory_version"].(string),
		InventoryScope:     manifest["inventory_scope"].(string),
		BackupRef:          manifest["backup_ref"].(string),
		BackupCutoff:       manifest["backup_cutoff"].(string),
		GeneratedAt:        manifest["generated_at"].(string),
		RecoverableBackups: entries,
	})
}

func writeInventory(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		t.Fatalf("chmod inventory before write: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write inventory: %v", err)
	}
	perm := os.FileMode(0o600)
	if articleDeletionJournalPlatformSupported() {
		perm = 0o400
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatalf("chmod inventory: %v", err)
	}
}

func replaceDigest(t *testing.T, contents string, digest string) string {
	t.Helper()
	return replaceField(t, contents, "inventory_digest", digest)
}

func replaceField(t *testing.T, contents string, field string, value string) string {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(contents), &decoded); err != nil {
		t.Fatalf("unmarshal inventory: %v", err)
	}
	if field != "inventory_digest" {
		decoded[field] = value
		entries, _ := decoded["recoverable_backups"].([]any)
		normalized := make([]map[string]string, 0, len(entries))
		for _, raw := range entries {
			item := raw.(map[string]any)
			normalized = append(normalized, map[string]string{
				"backup_ref":          item["backup_ref"].(string),
				"backup_completed_at": item["backup_completed_at"].(string),
			})
		}
		decoded["recoverable_backups"] = normalized
		digest, err := canonicalInventoryDigest(t, decoded)
		if err != nil {
			t.Fatalf("canonicalInventoryDigest() error = %v", err)
		}
		decoded["inventory_digest"] = digest
		return marshalOrdered(t, decoded)
	}
	decoded[field] = value
	entries, _ := decoded["recoverable_backups"].([]any)
	normalized := make([]map[string]string, 0, len(entries))
	for _, raw := range entries {
		item := raw.(map[string]any)
		normalized = append(normalized, map[string]string{
			"backup_ref":          item["backup_ref"].(string),
			"backup_completed_at": item["backup_completed_at"].(string),
		})
	}
	decoded["recoverable_backups"] = normalized
	return marshalOrdered(t, decoded)
}

//go:build contract

package identity_contract_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"han/internal/recovery"
)

const (
	readyFixtureName       = "ready-v1.json"
	unavailableFixtureName = "unavailable-v1.json"
	readyFixtureDigest     = "sha256:169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7"
)

// TestReplayStatusConsumer_Ready20FixtureMatchesAuthority pins the identity-side
// consumer view of the shared v1 Ready20 fixture to the unique `internal/recovery`
// authority: exactly 20 ordered fields, canonical digest over the first 16, and a
// digest equal to both the fixture's declared value and the package constant.
func TestReplayStatusConsumer_Ready20FixtureMatchesAuthority(t *testing.T) {
	raw := readContractFixture(t, readyFixtureName)

	decoded, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil {
		t.Fatalf("DecodeReplayStatusV1(ready) error = %v", err)
	}
	if decoded.Ready == nil || decoded.Unavailable != nil {
		t.Fatalf("decoded variant = %#v, want ready only", decoded)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("Unmarshal(ready fixture) error = %v", err)
	}
	expected := []string{
		"evidence_version", "run_id", "backup_ref", "namespace",
		"journal_head_sequence", "journal_head_hash", "inventory_scope",
		"backup_cutoff", "inventory_digest", "deletion_operations_digest",
		"audit_events_digest", "tombstones_total", "tombstones_replayed",
		"already_absent", "deletion_operations_reconciled_count",
		"audit_events_verified_count", "status", "evidence_digest",
		"evidence_ref", "completed_at",
	}
	if len(fields) != len(expected) {
		t.Fatalf("ready fixture field count = %d, want %d", len(fields), len(expected))
	}
	for _, name := range expected {
		if _, ok := fields[name]; !ok {
			t.Fatalf("ready fixture is missing field %q", name)
		}
	}

	if decoded.Ready.EvidenceVersion != recovery.ReplayEvidenceVersionV1 {
		t.Fatalf("evidence_version = %q, want %q",
			decoded.Ready.EvidenceVersion, recovery.ReplayEvidenceVersionV1)
	}
	if decoded.Ready.Namespace != recovery.ArticleDeletionNamespaceV1 {
		t.Fatalf("namespace = %q, want %q",
			decoded.Ready.Namespace, recovery.ArticleDeletionNamespaceV1)
	}
	if decoded.Ready.EvidenceDigest != readyFixtureDigest {
		t.Fatalf("declared digest = %q, want %q",
			decoded.Ready.EvidenceDigest, readyFixtureDigest)
	}
	if recovery.ReadyFixtureDigestV1 != readyFixtureDigest {
		t.Fatalf("authority constant = %q, want %q",
			recovery.ReadyFixtureDigestV1, readyFixtureDigest)
	}

	digest, err := decoded.Ready.CanonicalDigest()
	if err != nil {
		t.Fatalf("CanonicalDigest() error = %v", err)
	}
	if digest != readyFixtureDigest {
		t.Fatalf("recomputed digest = %q, want %q", digest, readyFixtureDigest)
	}
}

// TestReplayStatusConsumer_CanonicalDigestIgnoresMetadataFields proves the
// consumer contract that only the first 16 fields enter the canonical digest;
// the four metadata fields (status, evidence_digest, evidence_ref, completed_at)
// must not change it.
func TestReplayStatusConsumer_CanonicalDigestIgnoresMetadataFields(t *testing.T) {
	raw := readContractFixture(t, readyFixtureName)
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil {
		t.Fatalf("DecodeReplayStatusV1() error = %v", err)
	}
	base := *decoded.Ready

	mutations := []struct {
		name   string
		mutate func(*recovery.ReadyReplayStatusV1)
	}{
		{"status", func(r *recovery.ReadyReplayStatusV1) { r.Status = "different" }},
		{"evidence_digest", func(r *recovery.ReadyReplayStatusV1) { r.EvidenceDigest = "sha256:00" }},
		{"evidence_ref", func(r *recovery.ReadyReplayStatusV1) { r.EvidenceRef = "different" }},
		{"completed_at", func(r *recovery.ReadyReplayStatusV1) { r.CompletedAt = "2026-10-22T00:00:00Z" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			candidate := base
			mutation.mutate(&candidate)
			digest, err := candidate.CanonicalDigest()
			if err != nil {
				t.Fatalf("CanonicalDigest() error = %v", err)
			}
			if digest != readyFixtureDigest {
				t.Fatalf("digest = %q, want unchanged %q", digest, readyFixtureDigest)
			}
		})
	}

	canonicalMutation := base
	canonicalMutation.JournalHeadSequence++
	changed, err := canonicalMutation.CanonicalDigest()
	if err != nil {
		t.Fatalf("CanonicalDigest(canonical mutation) error = %v", err)
	}
	if changed == readyFixtureDigest {
		t.Fatalf("digest = %q, want change after canonical-field mutation", changed)
	}
}

// TestReplayStatusConsumer_Unavailable6StopsBeforeDigest proves the six-field
// unavailable variant is accepted and that no canonical digest is computed.
func TestReplayStatusConsumer_Unavailable6StopsBeforeDigest(t *testing.T) {
	raw := readContractFixture(t, unavailableFixtureName)
	digestCalls := 0
	decoder := recovery.NewReplayStatusV1Decoder(func([]byte) string {
		digestCalls++
		return "sha256:unexpected"
	})

	decoded, err := decoder.Decode(raw)
	if err != nil {
		t.Fatalf("Decode(unavailable) error = %v", err)
	}
	if decoded.Unavailable == nil || decoded.Ready != nil {
		t.Fatalf("decoded variant = %#v, want unavailable only", decoded)
	}
	if digestCalls != 0 {
		t.Fatalf("digest calls = %d, want 0", digestCalls)
	}
	for _, status := range []string{"pending", "unknown", "failed"} {
		candidate := strings.Replace(
			string(raw),
			`"status": "pending"`,
			`"status": "`+status+`"`,
			1,
		)
		if decodedStatus, err := recovery.DecodeReplayStatusV1([]byte(candidate)); err != nil {
			t.Fatalf("Decode(%s) error = %v", status, err)
		} else if decodedStatus.Unavailable == nil ||
			decodedStatus.Unavailable.Status != status {
			t.Fatalf("status %q decoded = %#v", status, decodedStatus)
		}
	}
}

// TestReplayStatusConsumer_FailsClosedOnExtraMissingNullAndTypeDrift proves the
// closed-schema contract for both variants: extra/missing/null/wrong-type and
// unknown-version inputs are rejected with the single stable error.
func TestReplayStatusConsumer_FailsClosedOnExtraMissingNullAndTypeDrift(t *testing.T) {
	readyRaw := readContractFixture(t, readyFixtureName)
	unavailableRaw := readContractFixture(t, unavailableFixtureName)

	cases := []struct {
		name   string
		source []byte
		mutate func(map[string]any)
	}{
		{"ready extra field", readyRaw, func(v map[string]any) { v["extra"] = true }},
		{"ready missing field", readyRaw, func(v map[string]any) { delete(v, "inventory_digest") }},
		{"ready null field", readyRaw, func(v map[string]any) { v["evidence_ref"] = nil }},
		{"ready wrong int type", readyRaw, func(v map[string]any) { v["tombstones_total"] = "3" }},
		{"ready wrong string type", readyRaw, func(v map[string]any) { v["inventory_scope"] = map[string]any{"a": "b"} }},
		{"ready unknown version", readyRaw, func(v map[string]any) { v["evidence_version"] = "article-deletion-replay/v2" }},
		{"ready count mismatch", readyRaw, func(v map[string]any) { v["already_absent"] = float64(2) }},
		{"unavailable extra field", unavailableRaw, func(v map[string]any) { v["journal_head_hash"] = "sha256:aaaa" }},
		{"unavailable null reason", unavailableRaw, func(v map[string]any) { v["reason_code"] = nil }},
		{"unavailable empty reason", unavailableRaw, func(v map[string]any) { v["reason_code"] = "" }},
		{"unavailable unknown status", unavailableRaw, func(v map[string]any) { v["status"] = "ready_later" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var value map[string]any
			if err := json.Unmarshal(test.source, &value); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			test.mutate(value)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if _, err := recovery.DecodeReplayStatusV1(raw); !errors.Is(err, recovery.ErrInvalidReplayStatus) {
				t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
			}
		})
	}
}

// TestReplayStatusConsumer_CrossRunAndNamespaceBindingsFailClosed proves the
// backup-A-for-run-B, wrong-namespace and version-binding failures that the
// recovery coordinator relies on remain rejected.
func TestReplayStatusConsumer_CrossRunAndNamespaceBindingsFailClosed(t *testing.T) {
	readyRaw := string(readContractFixture(t, readyFixtureName))
	mutations := []struct {
		name     string
		oldValue string
		newValue string
	}{
		{"wrong namespace", `"namespace": "article-deletion/v1"`, `"namespace": "recovery/v1"`},
		{"wrong evidence version", `"evidence_version": "article-deletion-replay/v1"`, `"evidence_version": "recovery/v1"`},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			mutated := strings.Replace(readyRaw, mutation.oldValue, mutation.newValue, 1)
			if mutated == readyRaw {
				t.Fatalf("mutation %q did not change fixture", mutation.name)
			}
			if _, err := recovery.DecodeReplayStatusV1([]byte(mutated)); !errors.Is(err, recovery.ErrInvalidReplayStatus) {
				t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
			}
		})
	}
}

func readContractFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(contractRepoRoot(t), "testdata", "contracts", "article-deletion-replay", name))
	if err != nil {
		t.Fatalf("read contract fixture %s: %v", name, err)
	}
	return raw
}

func contractRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

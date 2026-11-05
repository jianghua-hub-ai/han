package recovery

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const readyFixtureV1 = `{
  "evidence_version": "article-deletion-replay/v1",
  "run_id": "run-01",
  "backup_ref": "backup-2026-10-20T00:00:00Z",
  "namespace": "article-deletion/v1",
  "journal_head_sequence": 7,
  "journal_head_hash": "sha256:aaaa",
  "inventory_scope": "recoverable-backups-before-cutoff",
  "backup_cutoff": "2026-10-20T00:00:00Z",
  "inventory_digest": "sha256:bbbb",
  "deletion_operations_digest": "sha256:cccc",
  "audit_events_digest": "sha256:dddd",
  "tombstones_total": 3,
  "tombstones_replayed": 2,
  "already_absent": 1,
  "deletion_operations_reconciled_count": 3,
  "audit_events_verified_count": 3,
  "status": "ready",
  "evidence_digest": "sha256:169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7",
  "evidence_ref": "evidence-01",
  "completed_at": "2026-10-20T01:00:00Z"
}`

const readyCanonicalFixtureV1 = `{"evidence_version":"article-deletion-replay/v1","run_id":"run-01","backup_ref":"backup-2026-10-20T00:00:00Z","namespace":"article-deletion/v1","journal_head_sequence":7,"journal_head_hash":"sha256:aaaa","inventory_scope":"recoverable-backups-before-cutoff","backup_cutoff":"2026-10-20T00:00:00Z","inventory_digest":"sha256:bbbb","deletion_operations_digest":"sha256:cccc","audit_events_digest":"sha256:dddd","tombstones_total":3,"tombstones_replayed":2,"already_absent":1,"deletion_operations_reconciled_count":3,"audit_events_verified_count":3}`

const unavailableFixtureV1 = `{
  "evidence_version": "article-deletion-replay/v1",
  "run_id": "run-01",
  "backup_ref": "backup-2026-10-20T00:00:00Z",
  "namespace": "article-deletion/v1",
  "status": "pending",
  "reason_code": "replay_in_progress"
}`

func TestReplayStatusV1Decoder_Ready20Fixture_RecomputesFixedDigest(t *testing.T) {
	t.Parallel()

	status, err := DecodeReplayStatusV1([]byte(readyFixtureV1))
	if err != nil {
		t.Fatalf("DecodeReplayStatusV1() error = %v", err)
	}
	if status.Ready == nil || status.Unavailable != nil {
		t.Fatalf("decoded variant = %#v, want ready only", status)
	}
	canonical, err := status.Ready.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON() error = %v", err)
	}
	if string(canonical) != readyCanonicalFixtureV1 {
		t.Fatalf("canonical JSON = %s", canonical)
	}
	if status.Ready.EvidenceDigest != ReadyFixtureDigestV1 {
		t.Fatalf("evidence digest = %q, want %q", status.Ready.EvidenceDigest, ReadyFixtureDigestV1)
	}
}

func TestReplayStatusV1Decoder_ReadyRejectsWrongInventoryScopeString(t *testing.T) {
	t.Parallel()

	const wrongScope = "all-backups-before-cutoff"
	raw := strings.Replace(
		readyFixtureV1,
		"recoverable-backups-before-cutoff",
		wrongScope,
		1,
	)
	wrongCanonical := strings.Replace(
		readyCanonicalFixtureV1,
		"recoverable-backups-before-cutoff",
		wrongScope,
		1,
	)
	raw = strings.Replace(raw, ReadyFixtureDigestV1, SHA256Digest([]byte(wrongCanonical)), 1)

	if _, err := DecodeReplayStatusV1([]byte(raw)); !errors.Is(err, ErrInvalidReplayStatus) {
		t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
	}
}

func TestReplayStatusV1Decoder_ReadyRejectsReorderedFields(t *testing.T) {
	t.Parallel()

	reordered := strings.Replace(
		readyFixtureV1,
		"  \"evidence_version\": \"article-deletion-replay/v1\",\n  \"run_id\": \"run-01\",",
		"  \"run_id\": \"run-01\",\n  \"evidence_version\": \"article-deletion-replay/v1\",",
		1,
	)
	if reordered == readyFixtureV1 {
		t.Fatal("ready fixture reorder did not change input")
	}
	if _, err := DecodeReplayStatusV1([]byte(reordered)); !errors.Is(err, ErrInvalidReplayStatus) {
		t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
	}
}

func TestReplayStatusV1Decoder_Unavailable6_DoesNotComputeCanonicalDigest(t *testing.T) {
	t.Parallel()

	digestCalls := 0
	decoder := NewReplayStatusV1Decoder(func([]byte) string {
		digestCalls++
		return "sha256:unexpected"
	})
	status, err := decoder.Decode([]byte(unavailableFixtureV1))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if status.Unavailable == nil || status.Ready != nil {
		t.Fatalf("decoded variant = %#v, want unavailable only", status)
	}
	if digestCalls != 0 {
		t.Fatalf("digest calls = %d, want 0", digestCalls)
	}
}

func TestReplayStatusV1Decoder_BadTaggedUnionFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing ready field", mutate: func(value map[string]any) { delete(value, "inventory_digest") }},
		{name: "extra ready field", mutate: func(value map[string]any) { value["extra"] = true }},
		{name: "null ready field", mutate: func(value map[string]any) { value["evidence_ref"] = nil }},
		{name: "wrong inventory type", mutate: func(value map[string]any) { value["inventory_scope"] = map[string]any{"scope": "all"} }},
		{name: "wrong integer type", mutate: func(value map[string]any) { value["tombstones_total"] = "3" }},
		{name: "unknown version", mutate: func(value map[string]any) { value["evidence_version"] = "article-deletion-replay/v2" }},
		{name: "wrong digest", mutate: func(value map[string]any) { value["evidence_digest"] = "sha256:00" }},
		{name: "count mismatch", mutate: func(value map[string]any) { value["already_absent"] = float64(2) }},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var value map[string]any
			if err := json.Unmarshal([]byte(readyFixtureV1), &value); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			test.mutate(value)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if _, err := DecodeReplayStatusV1(raw); !errors.Is(err, ErrInvalidReplayStatus) {
				t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
			}
		})
	}
}

func TestReplayStatusV1Decoder_UnavailableRejectsMixedOrInvalidFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "canonical field", mutate: func(value map[string]any) { value["journal_head_sequence"] = float64(0) }},
		{name: "digest field", mutate: func(value map[string]any) { value["evidence_digest"] = "" }},
		{name: "ref field", mutate: func(value map[string]any) { value["evidence_ref"] = nil }},
		{name: "time field", mutate: func(value map[string]any) { value["completed_at"] = "" }},
		{name: "unknown status", mutate: func(value map[string]any) { value["status"] = "ready_later" }},
		{name: "empty reason", mutate: func(value map[string]any) { value["reason_code"] = "" }},
		{name: "extra field", mutate: func(value map[string]any) { value["extra"] = true }},
		{name: "null reason", mutate: func(value map[string]any) { value["reason_code"] = nil }},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var value map[string]any
			if err := json.Unmarshal([]byte(unavailableFixtureV1), &value); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			test.mutate(value)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if _, err := DecodeReplayStatusV1(raw); !errors.Is(err, ErrInvalidReplayStatus) {
				t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
			}
		})
	}
}

func TestReplayStatusV1Decoder_DuplicateOrTrailingJSONFailsClosed(t *testing.T) {
	t.Parallel()

	duplicate := strings.Replace(readyFixtureV1, `"status": "ready",`, `"status": "ready", "status": "ready",`, 1)
	for name, raw := range map[string][]byte{
		"duplicate field": []byte(duplicate),
		"trailing value":  append([]byte(readyFixtureV1), []byte(` {}`)...),
		"array":           []byte(`[]`),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeReplayStatusV1(raw); !errors.Is(err, ErrInvalidReplayStatus) {
				t.Fatalf("DecodeReplayStatusV1() error = %v, want ErrInvalidReplayStatus", err)
			}
		})
	}
}

func TestReadyReplayStatusV1_CanonicalDigest_ExcludesMetadataFields(t *testing.T) {
	t.Parallel()

	status, err := DecodeReplayStatusV1([]byte(readyFixtureV1))
	if err != nil {
		t.Fatalf("DecodeReplayStatusV1() error = %v", err)
	}
	original := *status.Ready

	tests := []struct {
		name   string
		mutate func(*ReadyReplayStatusV1)
	}{
		{name: "status", mutate: func(ready *ReadyReplayStatusV1) { ready.Status = "different" }},
		{name: "evidence digest", mutate: func(ready *ReadyReplayStatusV1) { ready.EvidenceDigest = "sha256:different" }},
		{name: "evidence ref", mutate: func(ready *ReadyReplayStatusV1) { ready.EvidenceRef = "different" }},
		{name: "completed at", mutate: func(ready *ReadyReplayStatusV1) { ready.CompletedAt = "2026-10-22T00:00:00Z" }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			candidate := original
			test.mutate(&candidate)
			digest, err := candidate.CanonicalDigest()
			if err != nil {
				t.Fatalf("CanonicalDigest() error = %v", err)
			}
			if digest != ReadyFixtureDigestV1 {
				t.Fatalf("digest = %q, want %q", digest, ReadyFixtureDigestV1)
			}
		})
	}
}

func TestReadyReplayStatusV1_CanonicalDigest_ChangesWhenCanonicalFieldChanges(t *testing.T) {
	t.Parallel()

	status, err := DecodeReplayStatusV1([]byte(readyFixtureV1))
	if err != nil {
		t.Fatalf("DecodeReplayStatusV1() error = %v", err)
	}
	candidate := *status.Ready
	candidate.JournalHeadSequence++
	digest, err := candidate.CanonicalDigest()
	if err != nil {
		t.Fatalf("CanonicalDigest() error = %v", err)
	}
	if digest == ReadyFixtureDigestV1 {
		t.Fatalf("digest = %q, want change", digest)
	}
}

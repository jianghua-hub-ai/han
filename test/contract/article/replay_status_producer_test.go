//go:build contract

package articlecontract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/recovery"
)

type producerStore struct {
	t     *testing.T
	read  func(context.Context, string, string, func(context.Context) (article.DeletionReplayFacts, error)) (article.DeletionReplayRun, *article.DeletionReplayEvidence, error)
	calls int
}

func (s *producerStore) ReadCurrent(ctx context.Context, run, backup string, refresh func(context.Context) (article.DeletionReplayFacts, error)) (article.DeletionReplayRun, *article.DeletionReplayEvidence, error) {
	s.calls++
	return s.read(ctx, run, backup, refresh)
}
func (s *producerStore) LoadAttempt(context.Context, string, string) (article.AttemptSnapshot, error) {
	s.t.Fatal("unexpected LoadAttempt")
	return article.AttemptSnapshot{}, nil
}
func (s *producerStore) Begin(context.Context, string, string, article.AttemptSnapshot, article.DeletionReplayFacts, string) (article.BeginResult, error) {
	s.t.Fatal("unexpected Begin")
	return article.BeginResult{}, nil
}
func (s *producerStore) Publish(context.Context, article.ReplayPublishRequest) (article.DeletionReplayEvidence, error) {
	s.t.Fatal("unexpected Publish")
	return article.DeletionReplayEvidence{}, nil
}
func (s *producerStore) Fail(context.Context, string, string, article.AttemptSnapshot, string) error {
	s.t.Fatal("unexpected Fail")
	return nil
}

type producerSource struct {
	t      *testing.T
	calls  int
	backup string
	ctx    context.Context
}

func (s *producerSource) Snapshot(ctx context.Context, backup string) (article.DeletionReplayFacts, error) {
	s.calls++
	s.backup = backup
	s.ctx = ctx
	return article.DeletionReplayFacts{}, nil
}
func (s *producerSource) ReplayOldBackup(context.Context, string) (article.DeletionReplayInput, error) {
	s.t.Fatal("unexpected ReplayOldBackup")
	return article.DeletionReplayInput{}, nil
}

type fixedIDs struct{}

func (fixedIDs) NewID() (string, error) { return auditID, nil }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return fixtureTime }

func TestArticleReplayStatusProducerContract(t *testing.T) {
	t.Run("ReadStatusReady20", producerReady)
	t.Run("ReadStatusUnavailable6", producerUnavailable)
	t.Run("InvalidEvidenceFailsClosed", producerBadEvidence)
	t.Run("SharedAuthorityAndConsumerMutations", producerMutations)
}

func readyFixture(t *testing.T) ([]byte, recovery.ReadyReplayStatusV1) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "contracts", "article-deletion-replay", "ready-v1.json"))
	must(t, err)
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	must(t, err)
	if decoded.Ready == nil || decoded.Unavailable != nil {
		t.Fatal("shared Ready fixture changed variant")
	}
	return raw, *decoded.Ready
}

func fixtureEvidence(t *testing.T) article.DeletionReplayEvidence {
	t.Helper()
	_, r := readyFixture(t)
	cutoff, err := time.Parse(time.RFC3339Nano, r.BackupCutoff)
	must(t, err)
	completed, err := time.Parse(time.RFC3339Nano, r.CompletedAt)
	must(t, err)
	return article.DeletionReplayEvidence{EvidenceVersion: r.EvidenceVersion, RunID: r.RunID, BackupRef: r.BackupRef, Namespace: r.Namespace,
		JournalHeadSequence: int64(r.JournalHeadSequence), JournalHeadHash: r.JournalHeadHash, InventoryScope: r.InventoryScope, BackupCutoff: cutoff,
		InventoryDigest: r.InventoryDigest, DeletionOperationsDigest: r.DeletionOperationsDigest, AuditEventsDigest: r.AuditEventsDigest,
		TombstonesTotal: int64(r.TombstonesTotal), TombstonesReplayed: int64(r.TombstonesReplayed), AlreadyAbsent: int64(r.AlreadyAbsent),
		DeletionOperationsReconciledCount: int64(r.DeletionOperationsReconciledCount), AuditEventsVerifiedCount: int64(r.AuditEventsVerifiedCount),
		EvidenceDigest: r.EvidenceDigest, EvidenceRef: r.EvidenceRef, CompletedAt: completed}
}

func callProducer(t *testing.T, evidence *article.DeletionReplayEvidence, run article.DeletionReplayRun, cause error) []byte {
	t.Helper()
	_, fixture := readyFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := &producerSource{t: t}
	store := &producerStore{t: t}
	store.read = func(got context.Context, runID, backup string, refresh func(context.Context) (article.DeletionReplayFacts, error)) (article.DeletionReplayRun, *article.DeletionReplayEvidence, error) {
		if got != ctx || runID != fixture.RunID || backup != fixture.BackupRef || refresh == nil {
			t.Fatal("ReadCurrent binding/callback changed")
		}
		if evidence != nil {
			_, err := refresh(got)
			must(t, err)
		}
		return run, evidence, cause
	}
	provider, err := article.NewDeletionReplayProvider(article.DeletionReplayDependencies{Store: store, Source: source, EvidenceIDs: fixedIDs{}, Tokens: fixedIDs{}, Clock: fixedClock{}})
	must(t, err)
	raw, err := provider.ReadStatus(ctx, fixture.RunID, fixture.BackupRef)
	must(t, err)
	if store.calls != 1 {
		t.Fatal("ReadStatus did not read exactly once")
	}
	if evidence != nil && (source.calls != 1 || source.ctx != ctx || source.backup != fixture.BackupRef) {
		t.Fatal("Refresh did not forward original context/backup")
	}
	if evidence == nil && source.calls != 0 {
		t.Fatal("unrequested source read")
	}
	return raw
}

func readyKeys() []string {
	return []string{"evidence_version", "run_id", "backup_ref", "namespace", "journal_head_sequence", "journal_head_hash", "inventory_scope", "backup_cutoff", "inventory_digest", "deletion_operations_digest", "audit_events_digest", "tombstones_total", "tombstones_replayed", "already_absent", "deletion_operations_reconciled_count", "audit_events_verified_count", "status", "evidence_digest", "evidence_ref", "completed_at"}
}

func producerReady(t *testing.T) {
	e := fixtureEvidence(t)
	raw := callProducer(t, &e, article.DeletionReplayRun{}, nil)
	fixture, want := readyFixture(t)
	got := jsonObject(t, raw, readyKeys()...)
	var expected map[string]json.RawMessage
	must(t, json.Unmarshal(fixture, &expected))
	for key, value := range expected {
		if bytes.Equal(got[key], []byte("null")) {
			t.Fatalf("Ready null field %s", key)
		}
		var left, right any
		must(t, json.Unmarshal(got[key], &left))
		must(t, json.Unmarshal(value, &right))
		if !equalJSON(t, left, right) {
			t.Fatalf("producer field %s differs from shared golden", key)
		}
	}
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	must(t, err)
	digest, err := decoded.Ready.CanonicalDigest()
	must(t, err)
	if digest != recovery.ReadyFixtureDigestV1 || digest != want.EvidenceDigest {
		t.Fatal("producer canonical differs from fixed golden")
	}
	canonical, err := decoded.Ready.CanonicalJSON()
	must(t, err)
	if strings.Contains(string(canonical), "evidence_ref") || strings.Contains(string(canonical), "completed_at") {
		t.Fatal("metadata entered canonical16")
	}
	e.EvidenceRef = "different-evidence"
	e.CompletedAt = e.CompletedAt.Add(time.Minute)
	changed, err := recovery.DecodeReplayStatusV1(callProducer(t, &e, article.DeletionReplayRun{}, nil))
	must(t, err)
	metadataDigest, err := changed.Ready.CanonicalDigest()
	must(t, err)
	if metadataDigest != digest {
		t.Fatal("metadata changed canonical digest")
	}
}

func equalJSON(t *testing.T, left, right any) bool {
	t.Helper()
	return bytes.Equal(marshal(t, left), marshal(t, right))
}

func producerUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name           string
		run            article.DeletionReplayRun
		err            error
		status, reason string
	}{
		{"pending", article.DeletionReplayRun{Status: article.ReplayRunRunning}, nil, "pending", "replay_in_progress"},
		{"missing", article.DeletionReplayRun{}, article.ErrDeletionReplayRunNotFound, "unknown", "evidence_missing"},
		{"failed", article.DeletionReplayRun{Status: article.ReplayRunFailed}, nil, "failed", "dependency_unavailable"},
		{"stale", article.DeletionReplayRun{}, article.ErrDeletionReplayInputChanged, "unknown", "evidence_stale"},
		{"binding", article.DeletionReplayRun{}, article.ErrDeletionReplayBackupMismatch, "failed", "backup_binding_mismatch"},
		{"unsupported", article.DeletionReplayRun{}, article.ErrDeletionReplayUnsupported, "unknown", "unsupported_evidence_version"},
		{"cause", article.DeletionReplayRun{}, errors.New(privateCanary), "failed", "dependency_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := callProducer(t, nil, tc.run, tc.err)
			obj := jsonObject(t, raw, "evidence_version", "run_id", "backup_ref", "namespace", "status", "reason_code")
			_, fixture := readyFixture(t)
			for key, value := range map[string]any{"evidence_version": fixture.EvidenceVersion, "run_id": fixture.RunID, "backup_ref": fixture.BackupRef, "namespace": fixture.Namespace, "status": tc.status, "reason_code": tc.reason} {
				fieldValue(t, obj, key, value)
			}
			calls := 0
			decoder := recovery.NewReplayStatusV1Decoder(func([]byte) string { calls++; return "unexpected" })
			decoded, err := decoder.Decode(raw)
			must(t, err)
			if calls != 0 || decoded.Ready != nil || decoded.Unavailable == nil || strings.Contains(string(raw), privateCanary) {
				t.Fatal("Unavailable computed digest/leaked cause/changed variant")
			}
		})
	}
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "contracts", "article-deletion-replay", "unavailable-v1.json"))
	must(t, err)
	actual := callProducer(t, nil, article.DeletionReplayRun{Status: article.ReplayRunRunning}, nil)
	var a, b any
	must(t, json.Unmarshal(fixture, &a))
	must(t, json.Unmarshal(actual, &b))
	if !equalJSON(t, a, b) {
		t.Fatal("pending producer differs from shared Unavailable fixture")
	}
}

func producerBadEvidence(t *testing.T) {
	for _, mode := range []string{"version", "namespace", "digest", "ref", "time", "negative_count"} {
		t.Run(mode, func(t *testing.T) {
			e := fixtureEvidence(t)
			switch mode {
			case "version":
				e.EvidenceVersion = "unsupported"
			case "namespace":
				e.Namespace = "wrong"
			case "digest":
				e.EvidenceDigest = "sha256:bad"
			case "ref":
				e.EvidenceRef = ""
			case "time":
				e.CompletedAt = time.Time{}
			case "negative_count":
				e.TombstonesTotal = -1
			}
			raw := callProducer(t, &e, article.DeletionReplayRun{}, nil)
			obj := jsonObject(t, raw, "evidence_version", "run_id", "backup_ref", "namespace", "status", "reason_code")
			fieldValue(t, obj, "status", "failed")
			fieldValue(t, obj, "reason_code", "evidence_unreadable")
		})
	}
}

func producerMutations(t *testing.T) {
	e := fixtureEvidence(t)
	raw := callProducer(t, &e, article.DeletionReplayRun{}, nil)
	for _, tc := range []struct{ name, old, new string }{
		{"extra", `"status":"ready"`, `"unexpected":true,"status":"ready"`},
		{"missing", `"evidence_ref":"evidence-01",`, ``},
		{"null", `"evidence_ref":"evidence-01"`, `"evidence_ref":null`},
		{"type", `"tombstones_total":3`, `"tombstones_total":"3"`},
		{"version", `"article-deletion-replay/v1"`, `"article-deletion-replay/v2"`},
		{"count", `"tombstones_total":3`, `"tombstones_total":4`},
		{"binding", `"run_id":"run-01"`, `"run_id":"another-run"`},
		{"mixed", `"status":"ready"`, `"reason_code":"replay_in_progress","status":"ready"`},
		{"order", `"evidence_version":"article-deletion-replay/v1","run_id":"run-01"`, `"run_id":"run-01","evidence_version":"article-deletion-replay/v1"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(string(raw), tc.old, tc.new, 1)
			if changed == string(raw) {
				t.Fatal("mutation did not target actual producer bytes")
			}
			if _, err := recovery.DecodeReplayStatusV1([]byte(changed)); !errors.Is(err, recovery.ErrInvalidReplayStatus) {
				t.Fatal("strict consumer accepted mutated producer bytes")
			}
		})
	}
}

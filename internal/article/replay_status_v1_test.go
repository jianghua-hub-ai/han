package article

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"han/internal/recovery"
)

func TestReplayStatus_ReadyFixtureMatchesConsumerContract(t *testing.T) {
	evidence := replayFixtureEvidence()

	raw, err := encodeReadyReplayStatus(evidence)
	if err != nil {
		t.Fatalf("encodeReadyReplayStatus() error = %v", err)
	}
	status, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil {
		t.Fatalf("DecodeReplayStatusV1() error = %v", err)
	}
	if status.Ready == nil || status.Unavailable != nil {
		t.Fatalf("decoded variant = %#v, want ready only", status)
	}
	if status.Ready.EvidenceDigest != recovery.ReadyFixtureDigestV1 {
		t.Fatalf("digest = %q, want %q", status.Ready.EvidenceDigest, recovery.ReadyFixtureDigestV1)
	}

	var ordered map[string]json.RawMessage
	if err := json.Unmarshal(raw, &ordered); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(ordered) != 20 {
		t.Fatalf("ready fields = %d, want 20", len(ordered))
	}
}

func TestReplayStatus_UnavailableVariantsRemainStrict(t *testing.T) {
	for _, statusName := range []string{
		replayStatusPending,
		replayStatusUnknown,
		replayStatusFailed,
	} {
		t.Run(statusName, func(t *testing.T) {
			raw, err := encodeUnavailableReplayStatus(
				"run-01",
				"backup-2026-10-20T00:00:00Z",
				statusName,
				replayReasonEvidenceMissing,
			)
			if err != nil {
				t.Fatalf("encodeUnavailableReplayStatus() error = %v", err)
			}
			decoded, err := recovery.DecodeReplayStatusV1(raw)
			if err != nil {
				t.Fatalf("DecodeReplayStatusV1() error = %v", err)
			}
			if decoded.Unavailable == nil || decoded.Ready != nil {
				t.Fatalf("decoded variant = %#v, want unavailable only", decoded)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if len(fields) != 6 {
				t.Fatalf("unavailable fields = %d, want 6", len(fields))
			}
		})
	}
}

func TestReplayEvidence_InputDigestTracksEveryCoveredInput(t *testing.T) {
	base := replayFixtureInput()
	original, err := replayInputDigest("run-01", "backup-01", base)
	if err != nil {
		t.Fatalf("replayInputDigest() error = %v", err)
	}
	mutations := []struct {
		name   string
		change func(*DeletionReplayInput)
	}{
		{name: "journal sequence", change: func(value *DeletionReplayInput) {
			value.JournalHeadSequence++
		}},
		{name: "journal hash", change: func(value *DeletionReplayInput) {
			value.JournalHeadHash = "sha256:eeee"
		}},
		{name: "backup cutoff", change: func(value *DeletionReplayInput) {
			value.BackupCutoff = value.BackupCutoff.Add(time.Second)
		}},
		{name: "inventory digest", change: func(value *DeletionReplayInput) {
			value.InventoryDigest = "sha256:eeee"
		}},
		{name: "deletion operations digest", change: func(value *DeletionReplayInput) {
			value.DeletionOperationsDigest = "sha256:ffff"
		}},
		{name: "audit events digest", change: func(value *DeletionReplayInput) {
			value.AuditEventsDigest = "sha256:1111"
		}},
		{name: "tombstones total", change: func(value *DeletionReplayInput) {
			value.TombstonesTotal++
			value.AlreadyAbsent++
		}},
		{name: "tombstones replayed", change: func(value *DeletionReplayInput) {
			value.TombstonesTotal++
			value.TombstonesReplayed++
		}},
		{name: "already absent", change: func(value *DeletionReplayInput) {
			value.TombstonesTotal++
			value.AlreadyAbsent++
		}},
		{name: "operations reconciled", change: func(value *DeletionReplayInput) {
			value.DeletionOperationsReconciledCount++
		}},
		{name: "audit events verified", change: func(value *DeletionReplayInput) {
			value.AuditEventsVerifiedCount++
		}},
	}
	for _, mutation := range mutations {
		candidate := base
		mutation.change(&candidate)
		digest, err := replayInputDigest("run-01", "backup-01", candidate)
		if err != nil {
			t.Fatalf("%s replayInputDigest() error = %v", mutation.name, err)
		}
		if digest == original {
			t.Fatalf("%s digest did not change", mutation.name)
		}
	}
	for _, binding := range []struct {
		runID     string
		backupRef string
	}{
		{runID: "run-02", backupRef: "backup-01"},
		{runID: "run-01", backupRef: "backup-02"},
	} {
		digest, err := replayInputDigest(binding.runID, binding.backupRef, base)
		if err != nil {
			t.Fatalf("binding replayInputDigest() error = %v", err)
		}
		if digest == original {
			t.Fatalf("binding %q/%q digest did not change", binding.runID, binding.backupRef)
		}
	}
	repeated, err := replayInputDigest("run-01", "backup-01", base)
	if err != nil || repeated != original {
		t.Fatalf("stable digest = %q, %v; want %q", repeated, err, original)
	}
}

func replayFixtureEvidence() DeletionReplayEvidence {
	return DeletionReplayEvidence{
		EvidenceVersion: DeletionReplayEvidenceVersion,
		RunID:           "run-01", BackupRef: "backup-2026-10-20T00:00:00Z",
		Namespace: ArticleDeletionNamespace, JournalHeadSequence: 7,
		JournalHeadHash:          "sha256:aaaa",
		InventoryScope:           "recoverable-backups-before-cutoff",
		BackupCutoff:             time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		InventoryDigest:          "sha256:bbbb",
		DeletionOperationsDigest: "sha256:cccc",
		AuditEventsDigest:        "sha256:dddd",
		TombstonesTotal:          3, TombstonesReplayed: 2, AlreadyAbsent: 1,
		DeletionOperationsReconciledCount: 3, AuditEventsVerifiedCount: 3,
		EvidenceDigest: recovery.ReadyFixtureDigestV1, EvidenceRef: "evidence-01",
		CompletedAt: time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC),
	}
}

// TestReplayEvidence_SubMicrosecondTimesNormalizeToStableUTC proves F003: a
// sub-microsecond backup cutoff or clock time must not change the persisted
// evidence digest or completed_at versus the microsecond-precision value that
// PostgreSQL timestamptz would round-trip.
func TestReplayEvidence_SubMicrosecondTimesNormalizeToStableUTC(t *testing.T) {
	input := replayFixtureInput()
	base := input.BackupCutoff.Truncate(time.Microsecond)
	input.BackupCutoff = base
	completed := base.Add(time.Hour)

	clean, err := evidenceFromReplayInput("run-01", "backup-01", input, "ref-01", completed)
	if err != nil {
		t.Fatalf("evidenceFromReplayInput() error = %v", err)
	}
	noisy := input
	noisy.BackupCutoff = base.Add(500 * time.Nanosecond)
	noisyEvidence, err := evidenceFromReplayInput(
		"run-01",
		"backup-01",
		noisy,
		"ref-01",
		completed.Add(500*time.Nanosecond),
	)
	if err != nil {
		t.Fatalf("evidenceFromReplayInput() error = %v", err)
	}
	if clean.EvidenceDigest != noisyEvidence.EvidenceDigest {
		t.Fatalf(
			"digest drift: clean=%s noisy=%s",
			clean.EvidenceDigest,
			noisyEvidence.EvidenceDigest,
		)
	}
	if !clean.CompletedAt.Equal(noisyEvidence.CompletedAt) {
		t.Fatalf("completed_at drift: %v vs %v", clean.CompletedAt, noisyEvidence.CompletedAt)
	}
	if noisyEvidence.CompletedAt.Nanosecond()%1000 != 0 ||
		noisyEvidence.BackupCutoff.Nanosecond()%1000 != 0 {
		t.Fatalf(
			"non-microsecond persisted time: cutoff=%v completed=%v",
			noisyEvidence.BackupCutoff,
			noisyEvidence.CompletedAt,
		)
	}
}

func replayFixtureInput() DeletionReplayInput {
	evidence := replayFixtureEvidence()
	return DeletionReplayInput{
		DeletionReplayFacts: DeletionReplayFacts{
			JournalHeadSequence:               evidence.JournalHeadSequence,
			JournalHeadHash:                   evidence.JournalHeadHash,
			InventoryScope:                    evidence.InventoryScope,
			BackupCutoff:                      evidence.BackupCutoff,
			InventoryDigest:                   evidence.InventoryDigest,
			DeletionOperationsDigest:          evidence.DeletionOperationsDigest,
			AuditEventsDigest:                 evidence.AuditEventsDigest,
			TombstonesTotal:                   evidence.TombstonesTotal,
			DeletionOperationsReconciledCount: evidence.DeletionOperationsReconciledCount,
			AuditEventsVerifiedCount:          evidence.AuditEventsVerifiedCount,
		},
		TombstonesReplayed: evidence.TombstonesReplayed,
		AlreadyAbsent:      evidence.AlreadyAbsent,
	}
}

func assertSameReadyStatus(t *testing.T, left, right []byte) {
	t.Helper()
	var leftValue recovery.ReadyReplayStatusV1
	var rightValue recovery.ReadyReplayStatusV1
	if err := json.Unmarshal(left, &leftValue); err != nil {
		t.Fatalf("left Unmarshal() error = %v", err)
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		t.Fatalf("right Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(leftValue, rightValue) {
		t.Fatalf("ready status changed:\nleft=%#v\nright=%#v", leftValue, rightValue)
	}
}

func TestReplayEvidence_FreshFactsPreserveBoundOutcomes(t *testing.T) {
	evidence := replayFixtureEvidence()
	fresh := replayFixtureInput().DeletionReplayFacts
	attempt := replayEvidenceTestAttempt(evidence)
	if err := checkReplayCurrentFacts(evidence.RunID, evidence.BackupRef, attempt, evidence, fresh); err != nil {
		t.Fatalf("valid legacy gen0/2And1 evidence rejected: %v", err)
	}
	for _, mutation := range replayFactMutations() {
		t.Run(mutation.name, func(t *testing.T) {
			changed := fresh
			mutation.change(&changed)
			if err := checkReplayCurrentFacts(evidence.RunID, evidence.BackupRef, attempt, evidence, changed); !errors.Is(err, ErrDeletionReplayInputChanged) {
				t.Fatalf("changed fact accepted: %v", err)
			}
		})
	}
}

func replayEvidenceTestAttempt(evidence DeletionReplayEvidence) AttemptSnapshot {
	return AttemptSnapshot{Exists: true, Status: ReplayRunReady, BackupRef: evidence.BackupRef,
		InputDigest: &evidence.EvidenceDigest, CurrentEvidenceRef: &evidence.EvidenceRef}
}

type replayFactMutation struct {
	name   string
	change func(*DeletionReplayFacts)
}

func replayFactMutations() []replayFactMutation {
	return []replayFactMutation{
		{"JournalHeadSequence", func(f *DeletionReplayFacts) { f.JournalHeadSequence++ }},
		{"JournalHeadHash", func(f *DeletionReplayFacts) { f.JournalHeadHash = "sha256:changed" }},
		{"InventoryScope", func(f *DeletionReplayFacts) { f.InventoryScope = "changed" }},
		{"BackupCutoff", func(f *DeletionReplayFacts) { f.BackupCutoff = f.BackupCutoff.Add(time.Second) }},
		{"InventoryDigest", func(f *DeletionReplayFacts) { f.InventoryDigest = "sha256:changed" }},
		{"DeletionOperationsDigest", func(f *DeletionReplayFacts) { f.DeletionOperationsDigest = "sha256:changed" }},
		{"AuditEventsDigest", func(f *DeletionReplayFacts) { f.AuditEventsDigest = "sha256:changed" }},
		{"TombstonesTotal", func(f *DeletionReplayFacts) { f.TombstonesTotal++ }},
		{"OperationsReconciled", func(f *DeletionReplayFacts) { f.DeletionOperationsReconciledCount++ }},
		{"AuditsVerified", func(f *DeletionReplayFacts) { f.AuditEventsVerifiedCount++ }},
	}
}

func TestReplayEvidence_CurrentMustBeCompleteAndBound(t *testing.T) {
	for _, mode := range []string{"version", "namespace", "run", "backup", "negative", "sum", "digest", "ref", "completed", "cutoff_nanos", "completed_nanos", "missing_current", "missing_input"} {
		t.Run(mode, func(t *testing.T) { testReplayBadCurrent(t, mode) })
	}
}

func testReplayBadCurrent(t *testing.T, mode string) {
	evidence := replayFixtureEvidence()
	fresh := replayFixtureInput().DeletionReplayFacts
	attempt := replayEvidenceTestAttempt(evidence)
	switch mode {
	case "version":
		evidence.EvidenceVersion = "unsupported"
	case "namespace":
		evidence.Namespace = "wrong"
	case "run":
		evidence.RunID = "wrong"
	case "backup":
		evidence.BackupRef = "wrong"
	case "negative":
		evidence.TombstonesReplayed = -1
	case "sum":
		evidence.AlreadyAbsent++
	case "digest":
		evidence.EvidenceDigest = "sha256:wrong"
	case "ref":
		evidence.EvidenceRef = ""
	case "completed":
		evidence.CompletedAt = time.Time{}
	case "cutoff_nanos":
		evidence.BackupCutoff = evidence.BackupCutoff.Add(time.Nanosecond)
	case "completed_nanos":
		evidence.CompletedAt = evidence.CompletedAt.Add(time.Nanosecond)
	case "missing_current":
		attempt.CurrentEvidenceRef = nil
	case "missing_input":
		attempt.InputDigest = nil
	}
	err := checkReplayCurrentFacts("run-01", "backup-2026-10-20T00:00:00Z", attempt, evidence, fresh)
	want := ErrDeletionReplayEvidenceBad
	if mode == "version" {
		want = ErrDeletionReplayUnsupported
	}
	if !errors.Is(err, want) {
		t.Fatalf("bad current error=%v, want %v", err, want)
	}
}

func TestReplayEvidence_UniqueHistoryNeverChoosesAmbiguousOutcome(t *testing.T) {
	first := replayFixtureEvidence()
	input := replayFixtureInput()
	input.TombstonesReplayed, input.AlreadyAbsent = 0, 3
	other, err := evidenceFromReplayInput(first.RunID, first.BackupRef, input, "other-history", first.CompletedAt)
	if err != nil {
		t.Fatal(err)
	}
	var selected DeletionReplayEvidence
	if err := selectReplayHistoryEvidence(&selected, first); err != nil {
		t.Fatal(err)
	}
	if err := selectReplayHistoryEvidence(&selected, other); !errors.Is(err, ErrDeletionReplayEvidenceBad) {
		t.Fatal("ambiguous outcomes selected")
	}
	if selected.EvidenceRef != first.EvidenceRef {
		t.Fatal("ambiguity changed the first immutable record")
	}
	selected = DeletionReplayEvidence{}
	other.EvidenceDigest = "sha256:broken"
	if err := selectReplayHistoryEvidence(&selected, other); !errors.Is(err, ErrDeletionReplayEvidenceBad) {
		t.Fatal("bad history accepted")
	}
}

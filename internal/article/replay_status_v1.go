package article

import (
	"encoding/json"
	"errors"
	"time"

	"han/internal/recovery"
)

const (
	replayStatusReady   = "ready"
	replayStatusPending = "pending"
	replayStatusUnknown = "unknown"
	replayStatusFailed  = "failed"

	replayReasonInProgress            = "replay_in_progress"
	replayReasonEvidenceMissing       = "evidence_missing"
	replayReasonEvidenceStale         = "evidence_stale"
	replayReasonUnsupportedVersion    = "unsupported_evidence_version"
	replayReasonBackupMismatch        = "backup_binding_mismatch"
	replayReasonEvidenceUnreadable    = "evidence_unreadable"
	replayReasonVerificationFailed    = "verification_failed"
	replayReasonDependencyUnavailable = "dependency_unavailable"
)

var replayUnavailableReasons = map[string]struct{}{
	replayReasonInProgress:            {},
	replayReasonEvidenceMissing:       {},
	replayReasonEvidenceStale:         {},
	replayReasonUnsupportedVersion:    {},
	replayReasonBackupMismatch:        {},
	replayReasonEvidenceUnreadable:    {},
	replayReasonVerificationFailed:    {},
	replayReasonDependencyUnavailable: {},
}

func encodeReadyReplayStatus(evidence DeletionReplayEvidence) ([]byte, error) {
	status, err := readyReplayStatus(evidence)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(status)
	if err != nil {
		return nil, err
	}
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil || decoded.Ready == nil || decoded.Unavailable != nil {
		if err == nil {
			err = errors.New("ready replay status encoded the wrong variant")
		}
		return nil, err
	}
	return raw, nil
}

func encodeUnavailableReplayStatus(
	runID string,
	backupRef string,
	statusName string,
	reason string,
) ([]byte, error) {
	if _, ok := replayUnavailableReasons[reason]; !ok {
		return nil, errors.New("invalid replay status reason")
	}
	status := recovery.UnavailableReplayStatusV1{
		EvidenceVersion: DeletionReplayEvidenceVersion,
		RunID:           runID,
		BackupRef:       backupRef,
		Namespace:       ArticleDeletionNamespace,
		Status:          statusName,
		ReasonCode:      reason,
	}
	raw, err := json.Marshal(status)
	if err != nil {
		return nil, err
	}
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	if err != nil || decoded.Unavailable == nil || decoded.Ready != nil {
		if err == nil {
			err = errors.New("unavailable replay status encoded the wrong variant")
		}
		return nil, err
	}
	return raw, nil
}

func readyReplayStatus(evidence DeletionReplayEvidence) (recovery.ReadyReplayStatusV1, error) {
	status, err := readyReplayCanonical(
		evidence.RunID,
		evidence.BackupRef,
		replayInputFromEvidence(evidence),
	)
	if err != nil {
		return recovery.ReadyReplayStatusV1{}, err
	}
	if evidence.EvidenceVersion != DeletionReplayEvidenceVersion ||
		evidence.Namespace != ArticleDeletionNamespace ||
		evidence.EvidenceRef == "" ||
		evidence.CompletedAt.IsZero() {
		return recovery.ReadyReplayStatusV1{}, errors.New("invalid replay evidence metadata")
	}
	status.Status = replayStatusReady
	status.EvidenceDigest = evidence.EvidenceDigest
	status.EvidenceRef = evidence.EvidenceRef
	status.CompletedAt = evidence.CompletedAt.UTC().Format(time.RFC3339Nano)
	digest, err := status.CanonicalDigest()
	if err != nil {
		return recovery.ReadyReplayStatusV1{}, err
	}
	if digest != status.EvidenceDigest {
		return recovery.ReadyReplayStatusV1{}, errors.New("replay evidence digest mismatch")
	}
	return status, nil
}

func readyReplayCanonical(
	runID string,
	backupRef string,
	input DeletionReplayInput,
) (recovery.ReadyReplayStatusV1, error) {
	// Canonicalize every participating time before computing the digest so a
	// sub-microsecond source clock cannot produce a digest that disagrees with
	// the persisted PostgreSQL microsecond value (F003).
	input.BackupCutoff = canonicalReplayTime(input.BackupCutoff)
	if input.JournalHeadSequence < 0 ||
		input.TombstonesTotal < 0 ||
		input.TombstonesReplayed < 0 ||
		input.AlreadyAbsent < 0 ||
		input.DeletionOperationsReconciledCount < 0 ||
		input.AuditEventsVerifiedCount < 0 ||
		input.BackupCutoff.IsZero() {
		return recovery.ReadyReplayStatusV1{}, errors.New("invalid replay evidence input")
	}
	status := recovery.ReadyReplayStatusV1{
		EvidenceVersion:                   DeletionReplayEvidenceVersion,
		RunID:                             runID,
		BackupRef:                         backupRef,
		Namespace:                         ArticleDeletionNamespace,
		JournalHeadSequence:               uint64(input.JournalHeadSequence),
		JournalHeadHash:                   input.JournalHeadHash,
		InventoryScope:                    input.InventoryScope,
		BackupCutoff:                      input.BackupCutoff.UTC().Format(time.RFC3339Nano),
		InventoryDigest:                   input.InventoryDigest,
		DeletionOperationsDigest:          input.DeletionOperationsDigest,
		AuditEventsDigest:                 input.AuditEventsDigest,
		TombstonesTotal:                   uint64(input.TombstonesTotal),
		TombstonesReplayed:                uint64(input.TombstonesReplayed),
		AlreadyAbsent:                     uint64(input.AlreadyAbsent),
		DeletionOperationsReconciledCount: uint64(input.DeletionOperationsReconciledCount),
		AuditEventsVerifiedCount:          uint64(input.AuditEventsVerifiedCount),
	}
	if _, err := status.CanonicalDigest(); err != nil {
		return recovery.ReadyReplayStatusV1{}, err
	}
	return status, nil
}

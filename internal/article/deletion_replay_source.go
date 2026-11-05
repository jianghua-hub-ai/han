package article

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"han/internal/platform"
	"han/internal/recovery"
)

// DeletionOperationReplayFact is the read-only operation completion fact that
// the source consumes. It is never synthesized by a caller.
type DeletionOperationReplayFact struct {
	OperationID    string
	Status         DeletionOperationStatus
	ArticleVersion int64
	CompletedAt    *time.Time
}

// ReplayOperationReader reads the durable DeletionOperation facts.
type ReplayOperationReader interface {
	ReadDeletionOperations(context.Context) ([]DeletionOperationReplayFact, error)
}

// ReplayAuditReader reads the durable deletion-audit facts.
type ReplayAuditReader interface {
	ReadDeletionAuditEvents(context.Context) ([]DeletionReplayAuditFact, error)
}

// DeletionReplayAuditFact is the read-only audit fact that the source consumes.
type DeletionReplayAuditFact struct {
	EventID    string
	TargetID   string
	OccurredAt time.Time
	Action     string
}

// DeletionReplayKeyringView exposes the active+retained journal keyring view.
type DeletionReplayKeyringView interface {
	KeyringReferenceIDs() []string
}

// DeletionTombstoneReplayer runs the idempotent ReplayAll convergence.
type DeletionTombstoneReplayer interface {
	ReplayAllTombstones(
		context.Context, []ArticleDeletionTombstone, string,
	) (DeletionReplayAllResult, error)
}

// ProductionDeletionReplaySource is the only production source. It derives every
// input from the real Journal/keyring, file inventory, ReplayAll, and the
// durable operation/audit facts; it never accepts caller-supplied counts,
// digests, or "verified" markers.
type ProductionDeletionReplaySource struct {
	journal    VerifiedDeletionJournal
	inventory  *FileBackupInventoryReader
	replayer   DeletionTombstoneReplayer
	operations ReplayOperationReader
	audits     ReplayAuditReader
	now        func() time.Time
}

// NewProductionDeletionReplaySource wires the explicit production collaborators.
func NewProductionDeletionReplaySource(
	journal VerifiedDeletionJournal,
	inventory *FileBackupInventoryReader,
	replayer DeletionTombstoneReplayer,
	operations ReplayOperationReader,
	audits ReplayAuditReader,
	now func() time.Time,
) (*ProductionDeletionReplaySource, error) {
	switch {
	case missingReplayPublicationPort(journal):
		return nil, errors.New("production replay source journal is required")
	case inventory == nil:
		return nil, errors.New("production replay source inventory reader is required")
	case missingReplayPublicationPort(replayer):
		return nil, errors.New("production replay source replayer is required")
	case missingReplayPublicationPort(operations):
		return nil, errors.New("production replay source operation reader is required")
	case missingReplayPublicationPort(audits):
		return nil, errors.New("production replay source audit reader is required")
	case now == nil:
		return nil, errors.New("production replay source clock is required")
	default:
		return &ProductionDeletionReplaySource{
			journal: journal, inventory: inventory, replayer: replayer,
			operations: operations, audits: audits, now: now,
		}, nil
	}
}

// ReplayOldBackup recomputes the full snapshot after executing ReplayAll.
func (source *ProductionDeletionReplaySource) ReplayOldBackup(
	ctx context.Context,
	backupRef string,
) (DeletionReplayInput, error) {
	if source == nil || ctx == nil || strings.TrimSpace(backupRef) == "" {
		return DeletionReplayInput{}, ErrDeletionReplayInvalidRequest
	}
	inventory, err := source.inventory.Read(ctx)
	if err != nil {
		return DeletionReplayInput{}, err
	}
	if err := validateInventoryBinding(inventory, backupRef); err != nil {
		return DeletionReplayInput{}, err
	}
	tombstones, err := source.journal.ReadAllVerified(ctx)
	if err != nil {
		return DeletionReplayInput{}, err
	}
	ctx, traceID, err := deletionReplayContext(ctx)
	if err != nil {
		return DeletionReplayInput{}, err
	}
	result, err := source.replayer.ReplayAllTombstones(ctx, tombstones, traceID)
	if err != nil {
		return DeletionReplayInput{}, err
	}
	facts, err := source.snapshot(ctx, inventory, tombstones)
	if err != nil {
		return DeletionReplayInput{}, err
	}
	if result.Replayed < 0 || result.AlreadyAbsent < 0 ||
		result.Replayed > facts.TombstonesTotal || facts.TombstonesTotal-result.Replayed != result.AlreadyAbsent {
		return DeletionReplayInput{}, ErrDeletionReplayInvalidRequest
	}
	return DeletionReplayInput{DeletionReplayFacts: facts,
		TombstonesReplayed: result.Replayed, AlreadyAbsent: result.AlreadyAbsent}, nil
}

func deletionReplayContext(ctx context.Context) (context.Context, string, error) {
	if traceID, ok := platform.TraceIDFromContext(ctx); ok {
		return ctx, traceID, nil
	}
	traceID, err := platform.NewTraceID()
	if err != nil {
		return nil, "", err
	}
	traced, err := platform.ContextWithTraceID(ctx, traceID)
	return traced, traceID, err
}

// Snapshot refreshes Facts without ReplayAll or invented physical outcomes.
func (source *ProductionDeletionReplaySource) Snapshot(
	ctx context.Context,
	backupRef string,
) (DeletionReplayFacts, error) {
	if source == nil || ctx == nil || strings.TrimSpace(backupRef) == "" {
		return DeletionReplayFacts{}, ErrDeletionReplayInvalidRequest
	}
	inventory, err := source.inventory.Read(ctx)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	if err := validateInventoryBinding(inventory, backupRef); err != nil {
		return DeletionReplayFacts{}, err
	}
	tombstones, err := source.journal.ReadAllVerified(ctx)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	return source.snapshot(ctx, inventory, tombstones)
}

func (source *ProductionDeletionReplaySource) snapshot(
	ctx context.Context,
	inventory BackupInventory,
	tombstones []ArticleDeletionTombstone,
) (DeletionReplayFacts, error) {
	operations, err := source.operations.ReadDeletionOperations(ctx)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	audits, err := source.audits.ReadDeletionAuditEvents(ctx)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	headSequence, headHash, err := journalHead(tombstones)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	operationsDigest, reconciled, err := deletionOperationsDigest(operations)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	auditDigest, verified, err := deletionAuditsDigest(audits)
	if err != nil {
		return DeletionReplayFacts{}, err
	}
	return DeletionReplayFacts{
		JournalHeadSequence:               headSequence,
		JournalHeadHash:                   headHash,
		InventoryScope:                    inventory.Scope,
		BackupCutoff:                      canonicalReplayTime(inventory.BackupCutoff),
		InventoryDigest:                   inventory.InventoryDigest,
		DeletionOperationsDigest:          operationsDigest,
		AuditEventsDigest:                 auditDigest,
		TombstonesTotal:                   int64(len(tombstones)),
		DeletionOperationsReconciledCount: reconciled,
		AuditEventsVerifiedCount:          verified,
	}, nil
}

func validateInventoryBinding(inventory BackupInventory, backupRef string) error {
	if inventory.BackupRef != backupRef ||
		inventory.Scope != BackupInventoryScopeV1 ||
		inventory.BackupCutoff.IsZero() {
		return ErrDeletionReplayBackupMismatch
	}
	return nil
}

func journalHead(tombstones []ArticleDeletionTombstone) (int64, string, error) {
	if len(tombstones) == 0 {
		return 0, "", nil
	}
	last := tombstones[len(tombstones)-1]
	if last.Sequence != int64(len(tombstones)) {
		return 0, "", ErrDeletionJournalCorrupt
	}
	return last.Sequence, deletionRecordHash(last), nil
}

func deletionOperationsDigest(
	operations []DeletionOperationReplayFact,
) (string, int64, error) {
	ordered := make([]DeletionOperationReplayFact, len(operations))
	copy(ordered, operations)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].OperationID < ordered[j].OperationID
	})
	var builder strings.Builder
	var reconciled int64
	for _, operation := range ordered {
		if operation.OperationID == "" {
			return "", 0, ErrDeletionReplayInvalidRequest
		}
		completedAt := ""
		if operation.CompletedAt != nil {
			completedAt = canonicalReplayTime(*operation.CompletedAt).Format(time.RFC3339Nano)
		}
		fmt.Fprintf(
			&builder, "%s\x00%s\x00%d\x00%s\n",
			operation.OperationID, operation.Status, operation.ArticleVersion, completedAt,
		)
		if operation.Status == DeletionOperationCompleted {
			reconciled++
		}
	}
	return digestOf(builder.String()), reconciled, nil
}

func deletionAuditsDigest(
	audits []DeletionReplayAuditFact,
) (string, int64, error) {
	ordered := make([]DeletionReplayAuditFact, len(audits))
	copy(ordered, audits)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].TargetID == ordered[j].TargetID {
			return ordered[i].EventID < ordered[j].EventID
		}
		return ordered[i].TargetID < ordered[j].TargetID
	})
	var builder strings.Builder
	for _, event := range ordered {
		if event.EventID == "" {
			return "", 0, ErrDeletionReplayInvalidRequest
		}
		fmt.Fprintf(
			&builder, "%s\x00%s\x00%s\x00%s\n",
			event.EventID, event.TargetID, event.Action,
			canonicalReplayTime(event.OccurredAt).Format(time.RFC3339Nano),
		)
	}
	return digestOf(builder.String()), int64(len(ordered)), nil
}

func digestOf(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}

var _ DeletionReplaySource = (*ProductionDeletionReplaySource)(nil)

// publicationJournal projects metadata only from the source's identical journal.
func (source *ProductionDeletionReplaySource) publicationJournal() (recovery.VerifiedJournalReader, error) {
	if source == nil || missingReplayPublicationPort(source.journal) || source.inventory == nil ||
		missingReplayPublicationPort(source.replayer) || missingReplayPublicationPort(source.operations) ||
		missingReplayPublicationPort(source.audits) || source.now == nil {
		return nil, ErrDeletionReplayInvalidRequest
	}
	journal, ok := source.journal.(recovery.VerifiedJournalReader)
	if !ok || missingReplayPublicationPort(journal) {
		return nil, errReplayPublicationUnconfigured
	}
	return journal, nil
}

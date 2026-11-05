package recovery

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrRecoveryNotReady = errors.New("recovery is not ready to consume")

type ArticleDeletionGate interface {
	ReplayAndVerify(context.Context, string, string) ([]byte, error)
	ReadStatus(context.Context, string, string) ([]byte, error)
}

type ConsumptionResult struct {
	Run      RecoveryRun
	Evidence ReadyReplayStatusV1
	Journal  JournalEntry
}

type Coordinator struct {
	store        RecoveryRunStore
	journal      ContextRecoveryJournal
	gate         ArticleDeletionGate
	publications PublicationPort
	decoder      ReplayStatusV1Decoder
	now          func() time.Time
}

func NewCoordinator(store RecoveryRunStore, journal ContextRecoveryJournal, gate ArticleDeletionGate, publications PublicationPort, decoder ReplayStatusV1Decoder, now func() time.Time) (*Coordinator, error) {
	for _, port := range []any{store, journal, gate, publications, now} {
		if missingRuntimePort(port) {
			return nil, ErrAdmissionBinding
		}
	}
	if decoder.digest == nil {
		return nil, ErrAdmissionBinding
	}
	return &Coordinator{store: store, journal: journal, gate: gate, publications: publications, decoder: decoder, now: now}, nil
}

func (c *Coordinator) phaseBinding(ctx context.Context, runID string) (PhasePermit, PermitBinding, error) {
	if e := checkedContext(ctx); e != nil {
		return PhasePermit{}, PermitBinding{}, e
	}
	a, ok := ctx.Value(runtimeContextKey{}).(runtimeAuthorization)
	if !ok || a.phase == nil || a.phase.owner == nil {
		return PhasePermit{}, PermitBinding{}, ErrAdmissionBinding
	}
	p := PhasePermit{grant: a.phase}
	if e := a.phase.owner.CheckPhase(ctx, p, PhaseRecoveryReplay); e != nil {
		return PhasePermit{}, PermitBinding{}, e
	}
	b, e := a.phase.owner.ReadPermitBinding(ctx)
	if e != nil {
		return PhasePermit{}, PermitBinding{}, e
	}
	if b.RunID != runID || !admissionText(b.BackupRef) || !admissionText(b.InstallationID) {
		return PhasePermit{}, PermitBinding{}, ErrAdmissionBinding
	}
	return p, b, nil
}

// These private observations only restrict an already-owned replay operation;
// they grant no capability and never replace Runtime's independent proof.
type consumptionSourceKey struct{}
type consumptionSources struct{ recovery, article VerifiedJournalSnapshot }

func (c *Coordinator) authorize(ctx context.Context, runID string) (PhasePermit, PermitBinding, error) {
	p, b, e := c.phaseBinding(ctx, runID)
	if e != nil {
		return p, b, e
	}
	if sources, ok := ctx.Value(consumptionSourceKey{}).(*consumptionSources); ok {
		if e := c.checkSources(ctx, p.grant.owner, sources); e != nil {
			return PhasePermit{}, PermitBinding{}, e
		}
	}
	return p, b, nil
}

func (c *Coordinator) checkSources(ctx context.Context, owner *admissionRuntime, sources *consumptionSources) error {
	recovery, e := c.journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil {
		return e
	}
	article, e := owner.deps.ArticleSnapshots.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if e != nil {
		return e
	}
	if !sameControlledJournal(sources.recovery, recovery) || !sameControlledJournal(sources.article, article) {
		return ErrAdmissionDrift
	}
	return owner.checkOwnership(ctx)
}

func (c *Coordinator) consumptionContext(ctx context.Context, runID string) (context.Context, error) {
	p, _, e := c.phaseBinding(ctx, runID)
	if e != nil {
		return nil, e
	}
	owner := p.grant.owner
	owner.mu.Lock()
	proof := owner.proof
	owner.mu.Unlock()
	sources := &consumptionSources{recovery: copyControlledJournal(proof.recovery), article: copyControlledJournal(proof.article)}
	if e := c.checkSources(ctx, owner, sources); e != nil {
		return nil, e
	}
	return context.WithValue(ctx, consumptionSourceKey{}, sources), nil
}

func (c *Coordinator) load(ctx context.Context, runID string) (RecoveryRun, error) {
	_, b, e := c.authorize(ctx, runID)
	if e != nil {
		return RecoveryRun{}, e
	}
	x, e := c.store.Load(ctx, runID)
	if e != nil {
		return RecoveryRun{}, e
	}
	if x.RunID != runID || x.BackupRef != b.BackupRef {
		return RecoveryRun{}, ErrAdmissionBinding
	}
	if _, _, e := c.authorize(ctx, runID); e != nil {
		return RecoveryRun{}, e
	}
	return x, nil
}

func (c *Coordinator) Consume(ctx context.Context, runID string) (ConsumptionResult, error) {
	var e error
	ctx, e = c.consumptionContext(ctx, runID)
	if e != nil {
		return ConsumptionResult{}, e
	}
	for steps := 0; steps < 5; steps++ {
		run, e := c.load(ctx, runID)
		if e != nil {
			return ConsumptionResult{}, e
		}
		switch run.Outcome {
		case RecoveryOutcomeIdentityReady:
			e = c.prepareArticle(ctx, run)
		case RecoveryOutcomeArticleDeletionReady:
			e = c.prepareConsume(ctx, run)
		case RecoveryOutcomeReadyToConsume:
			e = c.consumeJournal(ctx, run)
		case RecoveryOutcomeJournalConsumed:
			return c.finishConsumed(ctx, run)
		case RecoveryOutcomeSucceeded:
			evidence, e := c.revalidateEvidence(ctx, run)
			if e != nil {
				return ConsumptionResult{}, e
			}
			entry, e := c.journal.VerifyAgainst(ctx, run)
			if e != nil {
				return ConsumptionResult{}, e
			}
			if _, _, e := c.authorize(ctx, runID); e != nil {
				return ConsumptionResult{}, e
			}
			return ConsumptionResult{Run: run, Evidence: evidence, Journal: entry}, nil
		default:
			return ConsumptionResult{}, fmt.Errorf("%w: unsupported outcome", ErrRecoveryNotReady)
		}
		if e != nil {
			return ConsumptionResult{}, e
		}
	}
	return ConsumptionResult{}, ErrRecoveryNotReady
}

func (c *Coordinator) finishConsumed(ctx context.Context, run RecoveryRun) (ConsumptionResult, error) {
	evidence, e := c.revalidateEvidence(ctx, run)
	if e != nil {
		return ConsumptionResult{}, e
	}
	entry, e := c.journal.VerifyAgainst(ctx, run)
	if e != nil {
		return ConsumptionResult{}, e
	}
	final, e := c.transition(ctx, run, RecoveryOutcomeSucceeded, RecoveryRunChange{})
	if e != nil {
		return ConsumptionResult{}, e
	}
	if _, _, e := c.authorize(ctx, run.RunID); e != nil {
		return ConsumptionResult{}, e
	}
	return ConsumptionResult{Run: final, Evidence: evidence, Journal: entry}, nil
}

func (c *Coordinator) prepareArticle(ctx context.Context, run RecoveryRun) error {
	if _, e := c.journal.VerifyAgainst(ctx, run); e != nil {
		return e
	}
	evidence, e := c.replayAndDecode(ctx, run)
	if e != nil {
		return e
	}
	fresh, e := c.readAndDecode(ctx, run)
	if e != nil {
		return e
	}
	if fresh != evidence {
		return ErrRecoveryNotReady
	}
	publication, e := c.currentPublication(ctx, run, evidence, true)
	if e != nil {
		return e
	}
	final, e := c.readAndDecode(ctx, run)
	if e != nil {
		return e
	}
	if final != evidence {
		return ErrRecoveryNotReady
	}
	completed, e := time.Parse(time.RFC3339Nano, evidence.CompletedAt)
	if e != nil {
		return ErrRecoveryNotReady
	}
	_, e = c.transition(ctx, run, RecoveryOutcomeArticleDeletionReady, RecoveryRunChange{ArticleEvidence: &evidence, ArticleEvidenceCompletedAt: completed, ArticlePublication: &publication})
	return e
}

func (c *Coordinator) prepareConsume(ctx context.Context, run RecoveryRun) error {
	if _, e := c.revalidateEvidence(ctx, run); e != nil {
		return e
	}
	if _, e := c.journal.VerifyAgainst(ctx, run); e != nil {
		return e
	}
	_, e := c.transition(ctx, run, RecoveryOutcomeReadyToConsume, RecoveryRunChange{})
	return e
}

func (c *Coordinator) consumeJournal(ctx context.Context, run RecoveryRun) error {
	if _, e := c.revalidateEvidence(ctx, run); e != nil {
		return e
	}
	entry, e := c.journal.Read(ctx, run.RunID)
	if e != nil {
		return e
	}
	if entry.Stage == StageConsumed {
		exact := run
		exact.JournalStage = StageConsumed
		exact.JournalConsumedAt = &entry.RecordedAt
		if _, e := c.journal.VerifyAgainst(ctx, exact); e != nil {
			return e
		}
	} else {
		if _, e := c.journal.VerifyAgainst(ctx, run); e != nil {
			return e
		}
		if _, _, e := c.authorize(ctx, run.RunID); e != nil {
			return e
		}
		entry, e = c.markAndRead(ctx, run)
		if e != nil {
			return e
		}
	}
	if entry.Stage != StageConsumed || entry.RecordedAt.Before(run.StartedAt) {
		return ErrJournalMismatch
	}
	_, e = c.transition(ctx, run, RecoveryOutcomeJournalConsumed, RecoveryRunChange{JournalConsumedAt: &entry.RecordedAt})
	return e
}

func (c *Coordinator) markAndRead(ctx context.Context, run RecoveryRun) (JournalEntry, error) {
	returned, writeErr := c.journal.MarkConsumed(ctx, run.RunID)
	p, _, e := c.phaseBinding(ctx, run.RunID)
	if e != nil {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e)
	}
	observed, e := c.journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e)
	}
	_, entry, e := snapshotRun(observed, run.RunID)
	if e != nil || entry.Stage != StageConsumed {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e, ErrJournalMismatch)
	}
	sources := ctx.Value(consumptionSourceKey{}).(*consumptionSources)
	if !exactJournalAppend(sources.recovery, observed, entry) {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, ErrAdmissionDrift)
	}
	if writeErr == nil && returned != entry {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, ErrAdmissionDrift)
	}
	exact := run
	exact.JournalStage = StageConsumed
	exact.JournalConsumedAt = &entry.RecordedAt
	verified, e := c.journal.VerifyAgainst(ctx, exact)
	if e != nil || verified != entry {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e, ErrJournalMismatch)
	}
	final, e := c.journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil || !sameControlledJournal(observed, final) {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e, ErrAdmissionDrift)
	}
	article, e := p.grant.owner.deps.ArticleSnapshots.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if e != nil || !sameControlledJournal(sources.article, article) {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e, ErrAdmissionDrift)
	}
	sources.recovery = copyControlledJournal(observed)
	if _, _, e := c.authorize(ctx, run.RunID); e != nil {
		return JournalEntry{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e)
	}
	return entry, nil
}

func (c *Coordinator) transition(ctx context.Context, before RecoveryRun, next RecoveryOutcome, change RecoveryRunChange) (RecoveryRun, error) {
	if _, _, e := c.authorize(ctx, before.RunID); e != nil {
		return RecoveryRun{}, e
	}
	change.ChangedAt = canonicalTime(c.now())
	_, writeErr := c.store.Transition(ctx, before.RunID, before.Outcome, next, change)
	observed, e := c.load(ctx, before.RunID)
	expected := before
	applyExpectedChange(&expected, next, change)
	if e == nil && sameRecoveryRun(expected, observed) {
		return observed, nil
	}
	if writeErr != nil {
		return RecoveryRun{}, errors.Join(ErrRecoveryWriteUnknown, writeErr, e)
	}
	return RecoveryRun{}, errors.Join(ErrRecoveryWriteUnknown, ErrAdmissionDrift, e)
}

func applyExpectedChange(x *RecoveryRun, next RecoveryOutcome, change RecoveryRunChange) {
	x.Outcome = next
	if change.ArticleEvidence != nil {
		v := change.ArticleEvidence
		x.ArticleEvidenceVersion = v.EvidenceVersion
		x.ArticleEvidenceDigest = v.EvidenceDigest
		x.ArticleEvidenceRef = v.EvidenceRef
		x.ArticleNamespace = v.Namespace
		x.ArticleEvidenceCompletedAt = change.ArticleEvidenceCompletedAt
		x.ArticleReplayVerifiedAt = change.ChangedAt
	}
	if change.ArticlePublication != nil {
		x.ArticlePublicationRef = change.ArticlePublication.Ref
		x.ArticlePublicationDigest = change.ArticlePublication.Digest
	}
	if change.JournalConsumedAt != nil {
		x.JournalStage = StageConsumed
		t := *change.JournalConsumedAt
		x.JournalConsumedAt = &t
	}
	if next == RecoveryOutcomeSucceeded {
		x.CompletedAt = change.ChangedAt
	}
}

func sameRecoveryRun(a, b RecoveryRun) bool {
	if (a.JournalConsumedAt == nil) != (b.JournalConsumedAt == nil) {
		return false
	}
	if a.JournalConsumedAt != nil && !a.JournalConsumedAt.Equal(*b.JournalConsumedAt) {
		return false
	}
	a.JournalConsumedAt = nil
	b.JournalConsumedAt = nil
	return a == b
}

func (c *Coordinator) readPublication(ctx context.Context, run RecoveryRun, v ReadyReplayStatusV1) (PublicationBinding, error) {
	if _, _, e := c.authorize(ctx, run.RunID); e != nil {
		return PublicationBinding{}, e
	}
	return c.publications.ReadCurrentPublication(ctx, run.RunID, run.BackupRef, v.EvidenceRef, v.EvidenceDigest)
}

func publicationReadbackError(attempted bool, writeErr, cause error) error {
	if attempted {
		return errors.Join(ErrRecoveryWriteUnknown, writeErr, cause)
	}
	return cause
}

func (c *Coordinator) currentPublication(ctx context.Context, run RecoveryRun, v ReadyReplayStatusV1, allowEnsure bool) (PublicationBinding, error) {
	p, _, e := c.authorize(ctx, run.RunID)
	if e != nil {
		return PublicationBinding{}, e
	}
	x, e := c.readPublication(ctx, run, v)
	var writeErr error
	attempted := false
	if errors.Is(e, ErrPublicationMissing) && allowEnsure {
		if _, _, e := c.authorize(ctx, run.RunID); e != nil {
			return PublicationBinding{}, e
		}
		attempted = true
		_, writeErr = c.publications.EnsureCurrentPublication(ctx, p, run.RunID, run.BackupRef, v.EvidenceRef, v.EvidenceDigest)
		// The port's ACK is not proof. Read only under the same still-valid ctx/grant.
		x, e = c.readPublication(ctx, run, v)
	}
	if e != nil {
		return PublicationBinding{}, publicationReadbackError(attempted, writeErr, e)
	}
	if e := validateCurrentPublication(x, v); e != nil {
		return PublicationBinding{}, publicationReadbackError(attempted, writeErr, e)
	}
	sources := ctx.Value(consumptionSourceKey{}).(*consumptionSources)
	if x.JournalID != sources.article.JournalID || x.Epoch != sources.article.Epoch || x.Head != sources.article.Head {
		return PublicationBinding{}, publicationReadbackError(attempted, writeErr, ErrAdmissionBinding)
	}
	again, e := c.readPublication(ctx, run, v)
	if e != nil {
		return PublicationBinding{}, publicationReadbackError(attempted, writeErr, e)
	}
	if again != x {
		return PublicationBinding{}, publicationReadbackError(attempted, writeErr, ErrAdmissionDrift)
	}
	if _, _, e := c.authorize(ctx, run.RunID); e != nil {
		return PublicationBinding{}, publicationReadbackError(attempted, writeErr, e)
	}
	return x, nil
}

func validateCurrentPublication(x PublicationBinding, v ReadyReplayStatusV1) error {
	completed, e := time.Parse(time.RFC3339Nano, v.CompletedAt)
	if e != nil || x.Version != "article-replay-publication/v1" || x.RunID != v.RunID || x.BackupRef != v.BackupRef || x.EvidenceRef != v.EvidenceRef || x.EvidenceDigest != v.EvidenceDigest || !admissionText(x.Ref) || !sha256DigestPattern.MatchString(x.Digest) || !admissionText(x.JournalID) || x.Epoch == 0 || x.Head != (JournalHead{Sequence: v.JournalHeadSequence, Hash: v.JournalHeadHash}) || !x.EvidenceCompletedAt.Equal(completed) || !admissionTime(x.PublishedAt) || x.PublishedAt.Before(completed) {
		return ErrAdmissionBinding
	}
	return nil
}

func (c *Coordinator) replayAndDecode(ctx context.Context, run RecoveryRun) (ReadyReplayStatusV1, error) {
	if _, _, e := c.authorize(ctx, run.RunID); e != nil {
		return ReadyReplayStatusV1{}, e
	}
	raw, e := c.gate.ReplayAndVerify(ctx, run.RunID, run.BackupRef)
	if e != nil {
		return ReadyReplayStatusV1{}, fmt.Errorf("%w: replay failed: %w", ErrRecoveryNotReady, e)
	}
	return c.decodeReadyForRun(raw, run)
}
func (c *Coordinator) readAndDecode(ctx context.Context, run RecoveryRun) (ReadyReplayStatusV1, error) {
	if _, _, e := c.authorize(ctx, run.RunID); e != nil {
		return ReadyReplayStatusV1{}, e
	}
	raw, e := c.gate.ReadStatus(ctx, run.RunID, run.BackupRef)
	if e != nil {
		return ReadyReplayStatusV1{}, fmt.Errorf("%w: status read failed: %w", ErrRecoveryNotReady, e)
	}
	return c.decodeReadyForRun(raw, run)
}
func (c *Coordinator) revalidateEvidence(ctx context.Context, run RecoveryRun) (ReadyReplayStatusV1, error) {
	v, e := c.readAndDecode(ctx, run)
	if e != nil {
		return ReadyReplayStatusV1{}, e
	}
	completed, e := time.Parse(time.RFC3339Nano, v.CompletedAt)
	if e != nil || v.EvidenceVersion != run.ArticleEvidenceVersion || v.EvidenceDigest != run.ArticleEvidenceDigest || v.EvidenceRef != run.ArticleEvidenceRef || v.Namespace != run.ArticleNamespace || !completed.Equal(run.ArticleEvidenceCompletedAt) {
		return ReadyReplayStatusV1{}, ErrRecoveryNotReady
	}
	x, e := c.currentPublication(ctx, run, v, false)
	if e != nil {
		return ReadyReplayStatusV1{}, e
	}
	if x.Ref != run.ArticlePublicationRef || x.Digest != run.ArticlePublicationDigest {
		return ReadyReplayStatusV1{}, ErrAdmissionBinding
	}
	return v, nil
}
func (c *Coordinator) decodeReadyForRun(raw []byte, run RecoveryRun) (ReadyReplayStatusV1, error) {
	status, e := c.decoder.Decode(raw)
	if e != nil {
		return ReadyReplayStatusV1{}, e
	}
	if status.Ready == nil {
		return ReadyReplayStatusV1{}, fmt.Errorf("%w: article evidence unavailable", ErrRecoveryNotReady)
	}
	v := *status.Ready
	if v.RunID != run.RunID || v.BackupRef != run.BackupRef || v.Namespace != ArticleDeletionNamespaceV1 {
		return ReadyReplayStatusV1{}, fmt.Errorf("%w: article evidence binding mismatch", ErrRecoveryNotReady)
	}
	return v, nil
}

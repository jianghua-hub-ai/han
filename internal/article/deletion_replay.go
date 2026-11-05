package article

import (
	"context"
	"errors"
	"strings"

	"han/internal/recovery"
)

// deletionReplayMaxBeginAttempts bounds the commit-outcome-unknown retry loop.
const deletionReplayMaxBeginAttempts = 4

type DeletionReplayDependencies struct {
	Store       ReplayEvidenceStore
	Source      DeletionReplaySource
	EvidenceIDs ReplayEvidenceIDGenerator
	Tokens      ReplayEvidenceIDGenerator
	Clock       ReplayEvidenceClock
}

type DeletionReplayProvider struct {
	dependencies DeletionReplayDependencies
}

func NewDeletionReplayProvider(
	dependencies DeletionReplayDependencies,
) (*DeletionReplayProvider, error) {
	if dependencies.Store == nil || dependencies.Source == nil ||
		dependencies.EvidenceIDs == nil || dependencies.Tokens == nil ||
		dependencies.Clock == nil {
		return nil, errors.New("article deletion replay dependencies are required")
	}
	return &DeletionReplayProvider{dependencies: dependencies}, nil
}

// ReplayAndVerify runs the fixed attempt order: reuse a matching Ready, then
// converge through LoadAttempt/Begin, run the production source outside any
// transaction, and promote only the exact authoritative handle.
func (provider *DeletionReplayProvider) ReplayAndVerify(
	ctx context.Context,
	runID string,
	backupRef string,
) ([]byte, error) {
	if err := provider.validateRequest(ctx, runID, backupRef); err != nil {
		return nil, err
	}
	if raw, reuse := provider.tryReuseReady(ctx, runID, backupRef); reuse {
		return raw, nil
	}
	return provider.convergeAttempt(ctx, runID, backupRef)
}

// convergeAttempt acquires current Facts before minting one invocation candidate.
func (provider *DeletionReplayProvider) convergeAttempt(ctx context.Context, runID, backupRef string) ([]byte, error) {
	expected, fresh, err := provider.loadAttemptAndSnapshot(ctx, runID, backupRef)
	if err != nil {
		return provider.publishUnavailable(ctx, runID, backupRef, err)
	}
	candidate, err := provider.dependencies.Tokens.NewID()
	if err != nil || strings.TrimSpace(candidate) == "" {
		return encodeUnavailableReplayStatus(runID, backupRef, replayStatusFailed, replayReasonDependencyUnavailable)
	}
	attempt, err := provider.beginAttempt(ctx, runID, backupRef, expected, fresh, candidate)
	if err != nil {
		return provider.publishUnavailable(ctx, runID, backupRef, err)
	}
	return provider.publish(ctx, runID, backupRef, attempt, candidate)
}

// beginAttempt retries only through Begin; observing an owner never grants it.
func (provider *DeletionReplayProvider) beginAttempt(
	ctx context.Context, runID, backupRef string, expected AttemptSnapshot, fresh DeletionReplayFacts, candidate string,
) (AttemptSnapshot, error) {
	for attemptNumber := 0; attemptNumber < deletionReplayMaxBeginAttempts; attemptNumber++ {
		result, err := provider.dependencies.Store.Begin(ctx, runID, backupRef, expected, fresh, candidate)
		if err != nil {
			if definitiveReplayFailure(err) {
				return AttemptSnapshot{}, err
			}
			if _, ready, readErr := provider.tryReuseAfterUnknown(ctx, runID, backupRef); readErr != nil {
				return AttemptSnapshot{}, errors.Join(errReplayPublicationUnknown, err, readErr)
			} else if ready {
				return AttemptSnapshot{}, ErrDeletionReplayAlreadyReady
			}
			// Preserve the original expected snapshot on an unknown commit.
			if _, err := provider.dependencies.Store.LoadAttempt(ctx, runID, backupRef); err != nil {
				return AttemptSnapshot{}, err
			}
			fresh, err = provider.dependencies.Source.Snapshot(ctx, backupRef)
			if err != nil {
				return AttemptSnapshot{}, err
			}
			continue
		}
		switch result.Disposition {
		case AttemptAlreadyReady:
			return AttemptSnapshot{}, ErrDeletionReplayAlreadyReady
		case AttemptSuperseded:
			expected, fresh, err = provider.loadAttemptAndSnapshot(ctx, runID, backupRef)
			if err != nil {
				return AttemptSnapshot{}, err
			}
		default:
			return result.Attempt, nil
		}
	}
	return AttemptSnapshot{}, ErrDeletionReplayAttemptSuperseded
}

func definitiveReplayFailure(err error) bool {
	return errors.Is(err, ErrDeletionReplayBackupMismatch) || errors.Is(err, ErrDeletionReplayInvalidRequest) ||
		errors.Is(err, ErrDeletionReplayEvidenceBad) || errors.Is(err, ErrDeletionReplayUnsupported) ||
		errors.Is(err, ErrDeletionReplayInputChanged) || errors.Is(err, ErrDeletionReplayAttemptNotOwner)
}

// publish preserves the actual successful candidate across an unknown commit.
func (provider *DeletionReplayProvider) publish(ctx context.Context, runID, backupRef string, attempt AttemptSnapshot, candidate string) ([]byte, error) {
	input, err := provider.dependencies.Source.ReplayOldBackup(ctx, backupRef)
	if err != nil {
		return provider.failAttempt(ctx, runID, backupRef, attempt, replayReasonVerificationFailed)
	}
	request := ReplayPublishRequest{RunID: runID, BackupRef: backupRef, Attempt: attempt, Input: input,
		Refresh: provider.snapshotRefresh(backupRef), EvidenceIDs: provider.dependencies.EvidenceIDs, Clock: provider.dependencies.Clock}
	for number := 0; number < deletionReplayMaxBeginAttempts; number++ {
		evidence, publishErr := provider.dependencies.Store.Publish(ctx, request)
		if publishErr == nil {
			raw, encodeErr := encodeReadyReplayStatus(evidence)
			if encodeErr == nil {
				return raw, nil
			}
			return provider.failAttempt(ctx, runID, backupRef, request.Attempt, replayReasonEvidenceUnreadable)
		}
		err = publishErr
		// Exact triple readback precedes Fail, replay or retry after an unknown ACK.
		if raw, ready, readErr := provider.tryReuseAfterUnknown(ctx, runID, backupRef); readErr != nil {
			return provider.publicationUnknownStatus(runID, backupRef, errors.Join(publishErr, readErr))
		} else if ready {
			return raw, nil
		}
		if errors.Is(err, ErrDeletionReplayInputChanged) || errors.Is(err, ErrDeletionReplayAttemptNotOwner) ||
			errors.Is(err, ErrDeletionReplayAttemptSuperseded) {
			return provider.publishUnavailable(ctx, runID, backupRef, err)
		}
		if definitiveReplayFailure(err) {
			_, reason := replayFailureStatus(err)
			return provider.failAttempt(ctx, runID, backupRef, request.Attempt, reason)
		}
		request.Attempt, err = provider.resumeReplayPublish(ctx, request, candidate)
		if err != nil {
			if raw, ready, readErr := provider.tryReuseAfterUnknown(ctx, runID, backupRef); readErr != nil {
				return provider.publicationUnknownStatus(runID, backupRef, errors.Join(err, readErr))
			} else if ready {
				return raw, nil
			}
			return provider.publicationUnknownStatus(runID, backupRef, errors.Join(errReplayPublicationUnknown, err))
		}
	}
	if _, exact := provider.dependencies.Store.(replayPublicationReadback); exact {
		return provider.publicationUnknownStatus(runID, backupRef, errors.Join(errReplayPublicationUnknown, err))
	}
	return provider.publishUnavailable(ctx, runID, backupRef, err)
}

func (provider *DeletionReplayProvider) resumeReplayPublish(ctx context.Context, request ReplayPublishRequest, candidate string) (AttemptSnapshot, error) {
	if _, err := provider.dependencies.Store.LoadAttempt(ctx, request.RunID, request.BackupRef); err != nil {
		return AttemptSnapshot{}, err
	}
	fresh, err := provider.dependencies.Source.Snapshot(ctx, request.BackupRef)
	if err != nil {
		return AttemptSnapshot{}, err
	}
	if !sameReplayFacts(request.Input.DeletionReplayFacts, fresh) {
		return AttemptSnapshot{}, ErrDeletionReplayInputChanged
	}
	return provider.beginAttempt(ctx, request.RunID, request.BackupRef, request.Attempt, fresh, candidate)
}

func (provider *DeletionReplayProvider) failAttempt(ctx context.Context, runID, backupRef string, attempt AttemptSnapshot, reason string) ([]byte, error) {
	if _, ok := replayUnavailableReasons[reason]; !ok {
		reason = replayReasonDependencyUnavailable
	}
	for number := 0; number < deletionReplayMaxBeginAttempts; number++ {
		if raw, ready := provider.tryReuseReady(ctx, runID, backupRef); ready {
			return raw, nil
		}
		err := provider.dependencies.Store.Fail(ctx, runID, backupRef, attempt, reason)
		if err == nil {
			return unavailableReplayFailure(runID, backupRef, reason)
		}
		if raw, ready, readErr := provider.tryReuseAfterUnknown(ctx, runID, backupRef); readErr != nil {
			return provider.publicationUnknownStatus(runID, backupRef, errors.Join(err, readErr))
		} else if ready {
			return raw, nil
		}
		observed, observeErr := provider.dependencies.Store.LoadAttempt(ctx, runID, backupRef)
		if observeErr != nil {
			return provider.publishUnavailable(ctx, runID, backupRef, observeErr)
		}
		if !sameReplayOwner(observed, attempt) {
			return provider.publishUnavailable(ctx, runID, backupRef, ErrDeletionReplayAttemptSuperseded)
		}
		if observed.Status == ReplayRunFailed {
			return unavailableReplayFailure(runID, backupRef, reason)
		}
		if observed.Status != ReplayRunRunning {
			return provider.ReadStatus(ctx, runID, backupRef)
		}
		// Retry only the exact handle originally returned by Begin, not the observation.
	}
	return unavailableReplayFailure(runID, backupRef, replayReasonDependencyUnavailable)
}

func unavailableReplayFailure(runID, backupRef, reason string) ([]byte, error) {
	statusName := replayStatusFailed
	switch reason {
	case replayReasonEvidenceMissing, replayReasonEvidenceStale, replayReasonUnsupportedVersion:
		statusName = replayStatusUnknown
	}
	return encodeUnavailableReplayStatus(runID, backupRef, statusName, reason)
}

func (provider *DeletionReplayProvider) publishUnavailable(ctx context.Context, runID, backupRef string, err error) ([]byte, error) {
	if errors.Is(err, errReplayPublicationUnknown) {
		return provider.publicationUnknownStatus(runID, backupRef, err)
	}

	if raw, ready := provider.tryReuseReady(ctx, runID, backupRef); ready {
		return raw, nil
	}
	if errors.Is(err, ErrDeletionReplayAlreadyReady) {
		return provider.ReadStatus(ctx, runID, backupRef)
	}
	if errors.Is(err, ErrDeletionReplayInputChanged) || errors.Is(err, ErrDeletionReplayAttemptSuperseded) ||
		errors.Is(err, ErrDeletionReplayAttemptNotOwner) {
		return encodeUnavailableReplayStatus(runID, backupRef, replayStatusUnknown, replayReasonEvidenceStale)
	}
	statusName, reason := replayFailureStatus(err)
	// No authoritative Begin handle here: never Fail using a mere LoadAttempt.
	return encodeUnavailableReplayStatus(runID, backupRef, statusName, reason)
}

// ReadStatus returns the public projection after re-deriving the current input.
func (provider *DeletionReplayProvider) ReadStatus(
	ctx context.Context,
	runID string,
	backupRef string,
) ([]byte, error) {
	if err := provider.validateRequest(ctx, runID, backupRef); err != nil {
		return nil, err
	}
	run, evidence, err := provider.dependencies.Store.ReadCurrent(
		ctx, runID, backupRef, provider.snapshotRefresh(backupRef),
	)
	if err != nil {
		statusName, reason := replayReadFailureStatus(err)
		return encodeUnavailableReplayStatus(runID, backupRef, statusName, reason)
	}
	if evidence == nil {
		statusName, reason := unavailableForRun(run)
		return encodeUnavailableReplayStatus(runID, backupRef, statusName, reason)
	}
	raw, err := encodeReadyReplayStatus(*evidence)
	if err == nil {
		return raw, nil
	}
	return encodeUnavailableReplayStatus(
		runID, backupRef, replayStatusFailed, replayReasonEvidenceUnreadable,
	)
}

// tryReuseReady returns encoded Ready bytes when the persisted current evidence
// still matches the freshly derived input. Any other outcome returns reuse=false
// so the caller converges through LoadAttempt/Begin.
func (provider *DeletionReplayProvider) tryReuseReady(
	ctx context.Context,
	runID string,
	backupRef string,
) ([]byte, bool) {
	_, evidence, err := provider.dependencies.Store.ReadCurrent(
		ctx, runID, backupRef, provider.snapshotRefresh(backupRef),
	)
	if err != nil || evidence == nil {
		return nil, false
	}
	raw, err := encodeReadyReplayStatus(*evidence)
	if err != nil {
		return nil, false
	}
	return raw, true
}

func (provider *DeletionReplayProvider) loadAttemptAndSnapshot(
	ctx context.Context,
	runID string,
	backupRef string,
) (AttemptSnapshot, DeletionReplayFacts, error) {
	expected, err := provider.dependencies.Store.LoadAttempt(ctx, runID, backupRef)
	if err != nil {
		return AttemptSnapshot{}, DeletionReplayFacts{}, err
	}
	input, err := provider.dependencies.Source.Snapshot(ctx, backupRef)
	if err != nil {
		return AttemptSnapshot{}, DeletionReplayFacts{}, err
	}
	return normalizeAttemptBackup(expected, backupRef), input, nil
}

func normalizeAttemptBackup(attempt AttemptSnapshot, backupRef string) AttemptSnapshot {
	if !attempt.Exists {
		return AttemptSnapshot{}
	}
	attempt.BackupRef = backupRef
	return attempt
}

func (provider *DeletionReplayProvider) snapshotRefresh(
	backupRef string,
) func(context.Context) (DeletionReplayFacts, error) {
	return func(refreshContext context.Context) (DeletionReplayFacts, error) {
		return provider.dependencies.Source.Snapshot(refreshContext, backupRef)
	}
}

func (provider *DeletionReplayProvider) validateRequest(
	ctx context.Context,
	runID string,
	backupRef string,
) error {
	if provider == nil ||
		provider.dependencies.Store == nil ||
		provider.dependencies.Source == nil ||
		provider.dependencies.EvidenceIDs == nil ||
		provider.dependencies.Tokens == nil ||
		provider.dependencies.Clock == nil ||
		ctx == nil ||
		strings.TrimSpace(runID) == "" ||
		strings.TrimSpace(backupRef) == "" {
		return ErrDeletionReplayInvalidRequest
	}
	return nil
}

func replayFailureStatus(err error) (string, string) {
	switch {
	case errors.Is(err, ErrDeletionReplayBackupMismatch):
		return replayStatusFailed, replayReasonBackupMismatch
	case errors.Is(err, ErrDeletionReplayInputChanged):
		return replayStatusUnknown, replayReasonEvidenceStale
	case errors.Is(err, ErrDeletionReplayUnsupported):
		return replayStatusUnknown, replayReasonUnsupportedVersion
	case errors.Is(err, ErrDeletionReplayEvidenceBad):
		return replayStatusFailed, replayReasonEvidenceUnreadable
	case errors.Is(err, ErrDeletionReplayRunNotFound):
		return replayStatusUnknown, replayReasonEvidenceMissing
	default:
		return replayStatusFailed, replayReasonDependencyUnavailable
	}
}

func replayReadFailureStatus(err error) (string, string) {
	return replayFailureStatus(err)
}

func unavailableForRun(run DeletionReplayRun) (string, string) {
	if run.Status == ReplayRunRunning {
		return replayStatusPending, replayReasonInProgress
	}
	if run.LastErrorBucket != nil {
		reason := *run.LastErrorBucket
		switch reason {
		case replayReasonEvidenceMissing,
			replayReasonEvidenceStale,
			replayReasonUnsupportedVersion:
			return replayStatusUnknown, reason
		case replayReasonBackupMismatch,
			replayReasonEvidenceUnreadable,
			replayReasonVerificationFailed,
			replayReasonDependencyUnavailable:
			return replayStatusFailed, reason
		}
	}
	if run.Status == ReplayRunFailed {
		return replayStatusFailed, replayReasonDependencyUnavailable
	}
	return replayStatusUnknown, replayReasonEvidenceMissing
}

var _ recovery.ArticleDeletionGate = (*DeletionReplayProvider)(nil)

type replayPublicationReadback interface {
	readPublicationTriplet(context.Context, string, string) (*DeletionReplayEvidence, error)
}

func (provider *DeletionReplayProvider) tryReuseAfterUnknown(ctx context.Context, run, backup string) ([]byte, bool, error) {
	if reader, ok := provider.dependencies.Store.(replayPublicationReadback); ok {
		evidence, err := reader.readPublicationTriplet(ctx, run, backup)
		if err != nil {
			return nil, false, err
		}
		if evidence == nil {
			return nil, false, nil
		}
		raw, err := encodeReadyReplayStatus(*evidence)
		return raw, err == nil, err
	}
	// A production source cannot lose exact readback through an opaque Store wrapper.
	if _, production := provider.dependencies.Source.(*ProductionDeletionReplaySource); production {
		return nil, false, errReplayPublicationUnconfigured
	}
	// Preserve the pre-publication base-contract tests/adapters; no PostgreSQL fallback.
	raw, ready := provider.tryReuseReady(ctx, run, backup)
	return raw, ready, nil
}

func (provider *DeletionReplayProvider) publicationUnknownStatus(run, backup string, cause error) ([]byte, error) {
	status, reason := replayReadFailureStatus(cause)
	return encodeUnavailableReplayStatus(run, backup, status, reason)
}

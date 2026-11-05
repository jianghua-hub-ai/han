package main

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"han/internal/identity"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

// recoveryIdentityBridge captures the opaque phase permits that only the
// recovery runtime can mint. Package main cannot construct a PhasePermit, so
// the bridge observes them at the exact executor call sites, validates the
// observed binding, retains the permit privately, and then delegates under a
// derived phase context.
type recoveryIdentityBridge struct {
	runtime     recovery.Runtime
	executor    recovery.IdentityRecoveryExecutor
	phaseBudget time.Duration

	mu            sync.Mutex
	identity      recovery.PhasePermit
	identityBound recovery.PermitBinding
	identityValid bool
	replay        recovery.PhasePermit
	replayBound   recovery.PermitBinding
	replayValid   bool

	ready     chan struct{}
	readyOnce sync.Once

	complete     chan struct{}
	completeOnce sync.Once
	completeUsed bool

	requestFail     chan struct{}
	requestFailOnce sync.Once
	requestErr      error

	completeDone     chan struct{}
	completeDoneOnce sync.Once
	completeErr      error
	completeStarted  bool
}

func newRecoveryIdentityBridge(runtime recovery.Runtime, executor recovery.IdentityRecoveryExecutor, phaseBudget time.Duration) (*recoveryIdentityBridge, error) {
	if missingRecoveryBridgeValue(executor) || phaseBudget <= 0 {
		return nil, recovery.ErrAdmissionBinding
	}
	return &recoveryIdentityBridge{
		runtime:      runtime,
		executor:     executor,
		phaseBudget:  phaseBudget,
		ready:        make(chan struct{}),
		complete:     make(chan struct{}),
		completeDone: make(chan struct{}),
		requestFail:  make(chan struct{}),
	}, nil
}

func missingRecoveryBridgeValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

// bindRuntime closes the construction cycle: the runtime depends on this
// bridge as its identity executor, so the bridge receives the runtime only
// after the single runtime instance exists.
func (b *recoveryIdentityBridge) bindRuntime(runtime recovery.Runtime) error {
	if b == nil || missingRecoveryBridgeValue(runtime) {
		return recovery.ErrAdmissionBinding
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.runtime != nil {
		return recovery.ErrAdmissionBusy
	}
	b.runtime = runtime
	return nil
}

func (b *recoveryIdentityBridge) boundRuntime() recovery.Runtime {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.runtime
}

// capture validates that the observed permit really is the requested purpose
// bound to the requested run and backup, then retains it privately. The
// returned binding is a value copy; it never authorizes a caller.
func (b *recoveryIdentityBridge) capture(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding, purpose string) (recovery.RunBinding, error) {
	if b == nil || ctx == nil || binding.RunID == "" || binding.BackupRef == "" {
		return recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	if purpose != recovery.PhaseRecoveryIdentity && purpose != recovery.PhaseRecoveryReplay {
		return recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	runtime := b.boundRuntime()
	if runtime == nil {
		return recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	if err := runtime.CheckPhase(ctx, permit, purpose); err != nil {
		return recovery.RunBinding{}, err
	}
	phaseCtx, err := runtime.PhaseContext(ctx, permit)
	if err != nil {
		return recovery.RunBinding{}, err
	}
	observed, err := runtime.ReadPermitBinding(phaseCtx)
	if err != nil {
		return recovery.RunBinding{}, err
	}
	if observed.Purpose != purpose || observed.RunID != binding.RunID || observed.BackupRef != binding.BackupRef ||
		observed.InstallationID == "" || observed.RuntimeID == "" || observed.Generation <= 0 {
		return recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	bound := recovery.RunBinding{RunID: observed.RunID, BackupRef: observed.BackupRef}
	b.mu.Lock()
	switch purpose {
	case recovery.PhaseRecoveryIdentity:
		b.identity, b.identityBound, b.identityValid = permit, observed, true
	default:
		b.replay, b.replayBound, b.replayValid = permit, observed, true
	}
	b.mu.Unlock()
	return bound, nil
}

func (b *recoveryIdentityBridge) markReady() {
	b.readyOnce.Do(func() { close(b.ready) })
}

// waitIdentityReady blocks until the identity permit is captured or the
// context ends. The maintenance surface uses it before accepting traffic.
func (b *recoveryIdentityBridge) waitIdentityReady(ctx context.Context) error {
	if b == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	select {
	case <-b.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// waitCompleteRequest blocks until the operator confirmed completion over the
// maintenance HTTP surface and that accepted response was flushed.
func (b *recoveryIdentityBridge) waitCompleteRequest(ctx context.Context) error {
	if b == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	b.mu.Lock()
	used := b.completeUsed
	b.mu.Unlock()
	if used {
		return nil
	}
	select {
	case <-b.requestFail:
		return b.requestFailure()
	case <-b.complete:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RequestComplete implements identity.RecoveryCompleteSignaler. The HTTP
// handler calls it only after the accepted response is written and flushed.
func (b *recoveryIdentityBridge) RequestComplete() error {
	if b == nil {
		return recovery.ErrAdmissionBinding
	}
	b.mu.Lock()
	valid := b.identityValid
	failed := b.requestErr
	b.mu.Unlock()
	if !valid {
		err := incompleteCompleteSignal()
		b.recordCompleteFailure(err)
		return err
	}
	if failed != nil {
		return failed
	}
	b.completeOnce.Do(func() {
		b.mu.Lock()
		b.completeUsed = true
		b.mu.Unlock()
		close(b.complete)
	})
	return nil
}

// CheckComplete is the deterministic check the maintenance handler runs
// before it writes the accepted response. A missing identity permit closes
// the completion path and is published to the root controller, so the flow
// fails instead of waiting for a signal that can never arrive.
func (b *recoveryIdentityBridge) CheckComplete() error {
	if b == nil {
		return recovery.ErrAdmissionBinding
	}
	b.mu.Lock()
	valid := b.identityValid
	b.mu.Unlock()
	if valid {
		return nil
	}
	err := incompleteCompleteSignal()
	b.recordCompleteFailure(err)
	return err
}

// recordCompleteFailure wakes the root controller with the exact failure a
// post-flush signal cannot report over HTTP. The first failure wins.
func (b *recoveryIdentityBridge) recordCompleteFailure(err error) {
	if b == nil || err == nil {
		return
	}
	b.mu.Lock()
	if b.requestErr == nil {
		b.requestErr = err
	}
	b.mu.Unlock()
	b.requestFailOnce.Do(func() { close(b.requestFail) })
}

func (b *recoveryIdentityBridge) requestFailure() error {
	if b == nil {
		return recovery.ErrAdmissionBinding
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.requestErr == nil {
		return recovery.ErrAdmissionBinding
	}
	return b.requestErr
}

func incompleteCompleteSignal() error {
	return fmt.Errorf("%w: the identity permit was not captured", recovery.ErrAdmissionBinding)
}

// authorization re-reads the retained permit binding under a bounded phase
// context. A stale run, backup, purpose or generation fails closed.
func (b *recoveryIdentityBridge) authorization(ctx context.Context, purpose string) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
	if b == nil || ctx == nil {
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	runtime := b.boundRuntime()
	if runtime == nil {
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	b.mu.Lock()
	permit, expected, valid := b.identity, b.identityBound, b.identityValid
	if purpose == recovery.PhaseRecoveryReplay {
		permit, expected, valid = b.replay, b.replayBound, b.replayValid
	}
	b.mu.Unlock()
	if !valid {
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	bounded, cancel := context.WithTimeout(ctx, b.phaseBudget)
	phaseCtx, err := runtime.PhaseContext(bounded, permit)
	if err != nil {
		cancel()
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, err
	}
	observed, err := runtime.ReadPermitBinding(phaseCtx)
	if err != nil {
		cancel()
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, err
	}
	if observed.Purpose != purpose || observed.RunID != expected.RunID || observed.BackupRef != expected.BackupRef ||
		observed.InstallationID != expected.InstallationID || observed.RuntimeID != expected.RuntimeID ||
		observed.Generation != expected.Generation || observed.InstallationID == "" || observed.RuntimeID == "" || observed.Generation <= 0 {
		cancel()
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	context.AfterFunc(phaseCtx, cancel)
	return phaseCtx, permit, recovery.RunBinding{RunID: observed.RunID, BackupRef: observed.BackupRef}, nil
}

func (b *recoveryIdentityBridge) identityAuthorization(ctx context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
	return b.authorization(ctx, recovery.PhaseRecoveryIdentity)
}

func (b *recoveryIdentityBridge) replayAuthorization(ctx context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
	return b.authorization(ctx, recovery.PhaseRecoveryReplay)
}

// CompleteIdentityReview never invokes the real executor on this call stack.
// It validates and retains the identity permit, announces that the
// maintenance surface is ready, and then blocks until the root control flow
// has delegated the real completion after the operator's HTTP confirmation
// actually returned. The retained error is returned to the runtime flow.
func (b *recoveryIdentityBridge) CompleteIdentityReview(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
	if _, err := b.capture(ctx, permit, binding, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	b.markReady()
	select {
	case <-b.completeDone:
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.completeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// completeIdentity is the root control step. It waits for the flushed HTTP
// confirmation, then delegates the real executor exactly once with the
// retained identity permit and releases the runtime flow.
func (b *recoveryIdentityBridge) completeIdentity(ctx context.Context) error {
	if b == nil || ctx == nil || missingRecoveryBridgeValue(b.executor) {
		return recovery.ErrAdmissionBinding
	}
	if err := b.waitCompleteRequest(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	if b.completeStarted {
		b.mu.Unlock()
		return recovery.ErrAdmissionBusy
	}
	b.completeStarted = true
	b.mu.Unlock()
	err := func() error {
		phaseCtx, permit, bound, err := b.identityAuthorization(ctx)
		if err != nil {
			return err
		}
		return b.executor.CompleteIdentityReview(phaseCtx, permit, bound)
	}()
	b.mu.Lock()
	b.completeErr = err
	b.mu.Unlock()
	b.completeDoneOnce.Do(func() { close(b.completeDone) })
	return err
}

func (b *recoveryIdentityBridge) BindRestoredRun(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding, receipt recovery.RestoreReceipt) (recovery.RecoveryRun, error) {
	if b == nil || missingRecoveryBridgeValue(b.executor) {
		return recovery.RecoveryRun{}, recovery.ErrAdmissionBinding
	}
	return b.executor.BindRestoredRun(ctx, permit, binding, receipt)
}

func (b *recoveryIdentityBridge) FinalizeRestoredIdentity(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
	if _, err := b.capture(ctx, permit, binding, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	b.markReady()
	return b.executor.FinalizeRestoredIdentity(ctx, permit, binding)
}

func (b *recoveryIdentityBridge) PrepareRecoveryTrust(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding, secret []byte) error {
	if _, err := b.capture(ctx, permit, binding, recovery.PhaseRecoveryIdentity); err != nil {
		return err
	}
	return b.executor.PrepareRecoveryTrust(ctx, permit, binding, secret)
}

// RevokeRecoverySessions only captures the replay permit. Revocation happens
// in the root exit chain after every HTTP, password, settlement, transaction
// and child call has truly joined; it never runs here.
func (b *recoveryIdentityBridge) RevokeRecoverySessions(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
	_, err := b.capture(ctx, permit, binding, recovery.PhaseRecoveryReplay)
	return err
}

func (b *recoveryIdentityBridge) revokeSessions(ctx context.Context) error {
	if b == nil || missingRecoveryBridgeValue(b.executor) {
		return recovery.ErrAdmissionBinding
	}
	phaseCtx, permit, bound, err := b.replayAuthorization(ctx)
	if err != nil {
		return err
	}
	return b.executor.RevokeRecoverySessions(phaseCtx, permit, bound)
}

// recoverySecretSource bridges a read-only candidate provider. The bridge
// never retains the material and never verifies it; the Journal verifies the
// candidate against the retained Begin key.
type recoverySecretSource struct {
	read func(context.Context, recovery.RunBinding) ([]byte, error)
}

func newRecoverySecretSource(read func(context.Context, recovery.RunBinding) ([]byte, error)) (*recoverySecretSource, error) {
	if read == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	return &recoverySecretSource{read: read}, nil
}

func (s *recoverySecretSource) ReadInitialRecoverySecret(ctx context.Context, binding recovery.RunBinding) ([]byte, error) {
	if s == nil || s.read == nil || ctx == nil || binding.RunID == "" || binding.BackupRef == "" {
		return nil, recovery.ErrAdmissionBinding
	}
	return s.read(ctx, binding)
}

// recoveryAnchorAdapter is one explicitly registered production anchor adapter
// bundle. Production recovery state changes require a delivered and assembled
// CAS Anchor provider, so the registry stays empty until such a product exists.
type recoveryAnchorAdapter struct {
	Commitments recovery.InstallationCommitmentStore
	Sources     recovery.InstallationSourceVerifier
	Anchor      recovery.AnchorReader
	Exclusive   recovery.DeploymentExclusive
}

// recoveryAnchorRegistry is the config-gated adapter registry. Resolution is
// exact: HAN_RECOVERY_ANCHOR_ADAPTER must name a registered ID and the
// protected HAN_RECOVERY_ANCHOR_CONFIG_PATH must be available.
//
// The closed ports returned for an unregistered or mismatched adapter are an
// EXPLICIT REFUSAL, not a no-op success: every call fails with a Closed/Binding
// error and never fabricates a snapshot, commitment or lease, and never falls
// back to a file or SQL source. Normal and verify startup therefore stay
// available while the affected recovery action refuses to begin.
type recoveryAnchorRegistry struct {
	adapters map[string]recoveryAnchorAdapter
}

// recoveryAnchorPorts is the resolved provider set handed to the single root
// runtime assembly. It is either the registered adapter or four closed ports.
type recoveryAnchorPorts struct {
	AdapterID   string
	Commitments recovery.InstallationCommitmentStore
	Sources     recovery.InstallationSourceVerifier
	Anchor      recovery.AnchorReader
	Exclusive   recovery.DeploymentExclusive
}

func newRecoveryAnchorRegistry(adapters map[string]recoveryAnchorAdapter) (*recoveryAnchorRegistry, error) {
	copied := make(map[string]recoveryAnchorAdapter, len(adapters))
	for id, adapter := range adapters {
		if !recoveryExactText(id) || missingRecoveryBridgeValue(adapter.Commitments) ||
			missingRecoveryBridgeValue(adapter.Sources) || missingRecoveryBridgeValue(adapter.Anchor) ||
			missingRecoveryBridgeValue(adapter.Exclusive) {
			return nil, recovery.ErrAdmissionBinding
		}
		copied[id] = adapter
	}
	return &recoveryAnchorRegistry{adapters: copied}, nil
}

// resolve returns the registered adapter only for an exact ID with an available
// protected configuration path. Every other case closes all four ports.
func (r *recoveryAnchorRegistry) resolve(adapterID, configPath string) recoveryAnchorPorts {
	closed := func(reason string) recoveryAnchorPorts {
		port := recoveryClosedPort{reason: reason}
		return recoveryAnchorPorts{Commitments: port, Sources: port, Anchor: port, Exclusive: port}
	}
	if r == nil || !recoveryExactText(adapterID) {
		return closed("recovery anchor adapter id is not registered")
	}
	if !recoveryAbsolutePath(configPath) {
		return closed("recovery anchor adapter configuration is unavailable")
	}
	adapter, ok := r.adapters[adapterID]
	if !ok {
		return closed("recovery anchor adapter id is not registered")
	}
	return recoveryAnchorPorts{
		AdapterID:   adapterID,
		Commitments: adapter.Commitments,
		Sources:     adapter.Sources,
		Anchor:      adapter.Anchor,
		Exclusive:   adapter.Exclusive,
	}
}

// recoveryClosedPort refuses every operation of the four provider ports.
type recoveryClosedPort struct {
	reason string
}

func (p recoveryClosedPort) closed() error {
	reason := p.reason
	if reason == "" {
		reason = "recovery provider is not configured"
	}
	return fmt.Errorf("%w: %s", recovery.ErrAdmissionClosed, reason)
}

func (p recoveryClosedPort) LoadInstallation(context.Context, string) (recovery.InstallationCommitment, string, error) {
	return recovery.InstallationCommitment{}, "", p.closed()
}

func (p recoveryClosedPort) CompareAndSwapInstallation(context.Context, string, string, recovery.InstallationCommitment) (recovery.InstallationCommitment, string, error) {
	return recovery.InstallationCommitment{}, "", p.closed()
}

func (p recoveryClosedPort) VerifyInstallationSource(context.Context, recovery.InstallationManifest) error {
	return p.closed()
}

func (p recoveryClosedPort) Load(context.Context, string) (recovery.AnchorSnapshot, string, error) {
	return recovery.AnchorSnapshot{}, "", p.closed()
}

func (p recoveryClosedPort) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	return nil, p.closed()
}

var (
	_ recovery.InstallationCommitmentStore = recoveryClosedPort{}
	_ recovery.InstallationSourceVerifier  = recoveryClosedPort{}
	_ recovery.AnchorReader                = recoveryClosedPort{}
	_ recovery.DeploymentExclusive         = recoveryClosedPort{}
)

// The adapters below are EXPLICIT REFUSALS for provider products that are not
// yet delivered and assembled. Every call fails with a Closed/Binding error;
// none of them fabricates a snapshot, run, lease or success. This is
// deliberately different from a no-op that returns nil: normal and verify
// startup stay available while the affected recovery action refuses to begin.

// recoveryClosedVerifiedJournalReader refuses the Article journal snapshot
// port. The anchored Article handle needs a delivered keyring and anchor
// namespace; until then no read can be proven.
type recoveryClosedVerifiedJournalReader struct{ reason string }

func (r recoveryClosedVerifiedJournalReader) refused() error {
	reason := r.reason
	if reason == "" {
		reason = "Article journal snapshot port is not configured"
	}
	return fmt.Errorf("%w: %s", recovery.ErrAdmissionClosed, reason)
}

func (r recoveryClosedVerifiedJournalReader) ReadVerifiedSnapshot(context.Context, string) (recovery.VerifiedJournalSnapshot, error) {
	return recovery.VerifiedJournalSnapshot{}, r.refused()
}

// recoveryClosedRestoreDriver refuses the restore driver port.
type recoveryClosedRestoreDriver struct{}

func (recoveryClosedRestoreDriver) refused() error {
	return fmt.Errorf("%w: restore driver is not delivered", recovery.ErrAdmissionClosed)
}

func (d recoveryClosedRestoreDriver) Prepare(context.Context, recovery.RunBinding) (recovery.RestorePlan, error) {
	return recovery.RestorePlan{}, d.refused()
}

func (d recoveryClosedRestoreDriver) Replace(context.Context, recovery.PhasePermit, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return recovery.RestoreReceipt{}, d.refused()
}

func (d recoveryClosedRestoreDriver) VerifyRestored(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return recovery.RestoreReceipt{}, d.refused()
}

// recoveryClosedConsumer refuses the contextual Coordinator port.
type recoveryClosedConsumer struct{}

func (recoveryClosedConsumer) Consume(context.Context, string) (recovery.ConsumptionResult, error) {
	return recovery.ConsumptionResult{}, fmt.Errorf("%w: recovery coordinator is not delivered", recovery.ErrAdmissionClosed)
}

// recoveryClosedInstallationExecutor refuses the installation executor port.
type recoveryClosedInstallationExecutor struct{}

func (recoveryClosedInstallationExecutor) Install(context.Context, recovery.PhasePermit, recovery.InstallationManifest) error {
	return fmt.Errorf("%w: installation executor is not delivered", recovery.ErrAdmissionClosed)
}

// recoveryClosedLifecycle refuses both ordinary lifecycle steps. It never
// reports a fake start or join.
type recoveryClosedLifecycle struct{}

func (recoveryClosedLifecycle) ReconcileAndStart(context.Context, recovery.PhasePermit) error {
	return fmt.Errorf("%w: ordinary lifecycle owners are not bound", recovery.ErrAdmissionClosed)
}

func (recoveryClosedLifecycle) StopAndJoin(context.Context) error {
	return fmt.Errorf("%w: ordinary lifecycle owners are not bound", recovery.ErrAdmissionClosed)
}

// recoveryClosedSecretSource refuses the candidate secret port.
type recoveryClosedSecretSource struct{}

func (recoveryClosedSecretSource) ReadInitialRecoverySecret(context.Context, recovery.RunBinding) ([]byte, error) {
	return nil, fmt.Errorf("%w: recovery secret candidate source is not delivered", recovery.ErrAdmissionClosed)
}

var (
	_ recovery.VerifiedJournalReader = recoveryClosedVerifiedJournalReader{}
	_ recovery.RestoreDriver         = recoveryClosedRestoreDriver{}
	_ recovery.RecoveryConsumer      = recoveryClosedConsumer{}
	_ recovery.InstallationExecutor  = recoveryClosedInstallationExecutor{}
	_ recovery.OrdinaryLifecycle     = recoveryClosedLifecycle{}
	_ recovery.RecoverySecretSource  = recoveryClosedSecretSource{}
)

// recoveryClosedJournalAnchor refuses the JournalHeadAnchor port. The CAS anchor
// provider is a separate delivered product; until an adapter is registered the
// journal can neither load nor advance its head. This is an explicit refusal,
// not a no-op success: no snapshot is fabricated and nothing falls back to a
// file or SQL source.
type recoveryClosedJournalAnchor struct{ reason string }

func (a recoveryClosedJournalAnchor) refused() error {
	reason := a.reason
	if reason == "" {
		reason = "recovery journal anchor provider is not configured"
	}
	return fmt.Errorf("%w: %s", recovery.ErrAdmissionClosed, reason)
}

func (a recoveryClosedJournalAnchor) Load(context.Context, string) (recovery.AnchorSnapshot, string, error) {
	return recovery.AnchorSnapshot{}, "", a.refused()
}

func (a recoveryClosedJournalAnchor) CompareAndSwap(context.Context, string, string, recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	return recovery.AnchorSnapshot{}, "", a.refused()
}

// recoveryClosedArticleAccount refuses the identity ArticleAccountPort. The
// canonical orphaning command belongs to the Article service; the recovery
// identity executor must not own or duplicate it. Refusal keeps the affected
// action closed instead of deleting accounts without an owner.
type recoveryClosedArticleAccount struct{ reason string }

func (a recoveryClosedArticleAccount) refused() error {
	reason := a.reason
	if reason == "" {
		reason = "canonical Article account port is not bound"
	}
	return fmt.Errorf("%w: %s", recovery.ErrAdmissionClosed, reason)
}

func (a recoveryClosedArticleAccount) OrphanArticlesCanonical(context.Context, pgx.Tx, string) (identity.CanonicalDeleteArticleResult, error) {
	return identity.CanonicalDeleteArticleResult{}, a.refused()
}

var (
	_ recovery.JournalHeadAnchor  = recoveryClosedJournalAnchor{}
	_ identity.ArticleAccountPort = recoveryClosedArticleAccount{}
)

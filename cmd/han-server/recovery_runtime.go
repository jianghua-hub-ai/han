package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"

	"github.com/jackc/pgx/v5/pgxpool"
)

// recoveryRuntimePorts is the single-instance port set assembled by root.
// Every field must be the same object already owned by the process; the
// composition never constructs a second pool, Store, Runtime or registry.
type recoveryRuntimePorts struct {
	Runtime    recovery.Runtime
	Identity   recovery.IdentityRecoveryExecutor
	Exit       recovery.RecoveryExit
	Secret     recovery.RecoverySecretSource
	Lifecycle  recovery.OrdinaryLifecycle
	Article    *recoveryControlHandle
	JoinBudget time.Duration
}

// recoveryRuntimeComposition owns the recovery-specific lifecycle decisions
// that must not live in the public runtime package. It is the root-side
// RecoveryExit: it joins the maintenance surface, then confirms revocation
// through the same executor, and only then settles the long-lived Article
// control handle.
type recoveryRuntimeComposition struct {
	mu       sync.Mutex
	ports    recoveryRuntimePorts
	bridge   *recoveryIdentityBridge
	exitOnce sync.Once
	exitErr  error
}

var _ recovery.RecoveryExit = (*recoveryRuntimeComposition)(nil)
var _ recovery.IdentityRecoveryExecutor = (*recoveryIdentityBridge)(nil)

func newRecoveryRuntimeComposition(ports recoveryRuntimePorts) (*recoveryRuntimeComposition, error) {
	if missingRecoveryBridgeValue(ports.Runtime) || missingRecoveryBridgeValue(ports.Identity) ||
		missingRecoveryBridgeValue(ports.Exit) || missingRecoveryBridgeValue(ports.Secret) ||
		missingRecoveryBridgeValue(ports.Lifecycle) || ports.Article == nil || ports.JoinBudget <= 0 {
		return nil, recovery.ErrAdmissionBinding
	}
	// The runtime Identity port and this composition must be the same bridge:
	// the runtime captures the opaque permits there and the root delegates the
	// deferred calls through it. A second bridge would split capture and
	// deadlock the real flow.
	bridge, ok := ports.Identity.(*recoveryIdentityBridge)
	if !ok {
		var err error
		bridge, err = newRecoveryIdentityBridge(ports.Runtime, ports.Identity, ports.JoinBudget)
		if err != nil {
			return nil, err
		}
	}
	if bridge.boundRuntime() == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	return &recoveryRuntimeComposition{ports: ports, bridge: bridge}, nil
}

// ExitRecovery runs once per recovery flow. A second concurrent Stop observes
// the same settled result instead of joining or revoking twice.
func (c *recoveryRuntimeComposition) ExitRecovery(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
	if c == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	c.exitOnce.Do(func() {
		c.exitErr = c.exitRecovery(ctx, permit, binding)
	})
	return c.exitErr
}

func (c *recoveryRuntimeComposition) exitRecovery(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
	c.mu.Lock()
	ports, bridge := c.ports, c.bridge
	c.mu.Unlock()
	if bridge == nil {
		return recovery.ErrAdmissionBinding
	}
	// The replay permit is captured and re-verified before any join begins.
	if _, err := bridge.capture(ctx, permit, binding, recovery.PhaseRecoveryReplay); err != nil {
		return err
	}
	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), ports.JoinBudget)
	defer cancel()
	// The real maintenance HTTP, password change, settlement, transaction and
	// child work must be truly joined before the same executor revokes.
	if err := ports.Exit.ExitRecovery(bounded, permit, binding); err != nil {
		return err
	}
	if err := bridge.revokeSessions(bounded); err != nil {
		return err
	}
	return ports.Article.settle(bounded)
}

func (c *recoveryRuntimeComposition) runtime() recovery.Runtime {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ports.Runtime
}

func (c *recoveryRuntimeComposition) bridgeValue() *recoveryIdentityBridge {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bridge
}

// waitIdentityReady blocks until the identity permit has been captured. Root
// starts the maintenance surface and its Session gate only after this returns.
func (c *recoveryRuntimeComposition) waitIdentityReady(ctx context.Context) error {
	bridge := c.bridgeValue()
	if bridge == nil {
		return recovery.ErrAdmissionBinding
	}
	return bridge.waitIdentityReady(ctx)
}

// waitCompleteRequest blocks until the operator confirmed completion over the
// maintenance HTTP surface after the accepted response was flushed.
func (c *recoveryRuntimeComposition) waitCompleteRequest(ctx context.Context) error {
	bridge := c.bridgeValue()
	if bridge == nil {
		return recovery.ErrAdmissionBinding
	}
	return bridge.waitCompleteRequest(ctx)
}

// requestComplete is the only signal path the HTTP handler uses.
func (c *recoveryRuntimeComposition) requestComplete() error {
	bridge := c.bridgeValue()
	if bridge == nil {
		return recovery.ErrAdmissionBinding
	}
	return bridge.RequestComplete()
}

// completeIdentity is the root control step that delegates the real executor
// after the HTTP response has actually returned.
func (c *recoveryRuntimeComposition) completeIdentity(ctx context.Context) error {
	bridge := c.bridgeValue()
	if bridge == nil {
		return recovery.ErrAdmissionBinding
	}
	return bridge.completeIdentity(ctx)
}

// controlPhase derives a bounded phase context from the retained identity
// permit for one maintenance request. A missing or stale permit fails closed.
func (c *recoveryRuntimeComposition) controlPhase(ctx context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
	bridge := c.bridgeValue()
	if bridge == nil {
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	return bridge.identityAuthorization(ctx)
}

func (c *recoveryRuntimeComposition) replayPhase(ctx context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error) {
	bridge := c.bridgeValue()
	if bridge == nil {
		return nil, recovery.PhasePermit{}, recovery.RunBinding{}, recovery.ErrAdmissionBinding
	}
	return bridge.replayAuthorization(ctx)
}

// recoveryControlHandle is the root-owned long-lived Article control handle.
// Install returning nil never means the handle is closed, and the handle never
// becomes an ordinary WorkLease. Every consumer must finish; then Close joins
// the actual namespace owner exactly once before deployment ownership returns.
type recoveryControlHandle struct {
	closeFn func() error
	joinFn  func(context.Context) error
	stateFn func() string

	closeOnce  sync.Once
	joinOnce   sync.Once
	settleOnce sync.Once
	mu         sync.Mutex
	closeErr   error
	joinErr    error
	settleErr  error
}

func newRecoveryControlHandle(closeFn func() error, joinFn func(context.Context) error, stateFn func() string) (*recoveryControlHandle, error) {
	if closeFn == nil || joinFn == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	return &recoveryControlHandle{closeFn: closeFn, joinFn: joinFn, stateFn: stateFn}, nil
}

func (h *recoveryControlHandle) Close() error {
	if h == nil {
		return recovery.ErrAdmissionBinding
	}
	h.closeOnce.Do(func() {
		h.closeErr = panicRecoveryError(h.closeFn)
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeErr
}

func (h *recoveryControlHandle) Join(ctx context.Context) error {
	if h == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	h.joinOnce.Do(func() {
		h.joinErr = panicRecoveryValue(func() error { return h.joinFn(ctx) })
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.joinErr
}

// settle is the single concurrent-Stop settlement: the same owner closes and
// joins once, and every caller observes the same retained cause.
func (h *recoveryControlHandle) settle(ctx context.Context) error {
	if h == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	h.settleOnce.Do(func() {
		h.settleErr = errors.Join(h.Close(), h.Join(ctx))
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.settleErr
}

func (h *recoveryControlHandle) State() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	stateFn := h.stateFn
	h.mu.Unlock()
	if stateFn == nil {
		return ""
	}
	return stateFn()
}

func panicRecoveryError(fn func() error) (err error) {
	panicRecoveryValue(func() error {
		err = fn()
		return nil
	})
	return err
}

func panicRecoveryValue(fn func() error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = errors.Join(err, fmt.Errorf("recovery control settle panic: %v", value))
		}
	}()
	return fn()
}

const recoveryControlMaxBodyBytes int64 = 65536

// recoveryOpenJournal is the production journal seam. The default is the real
// file journal constructor; any failure, including the honest unsupported
// platform result, is a closed startup. Tests may substitute a fixture here.
// Production never registers a no-op or fake fallback.
var recoveryOpenJournal = func(ctx context.Context, options recovery.JournalOptions, keyring recovery.Keyring) (recovery.ContextRecoveryJournal, error) {
	return recovery.NewFileRecoveryJournal(ctx, options, keyring)
}

// recoveryLoadKeyring has no delivered protected loader. The default refuses
// instead of reading ambient credentials; the Journal verifies any candidate
// against the retained Begin key.
var recoveryLoadKeyring = func(string) (recovery.Keyring, error) {
	return recovery.Keyring{}, fmt.Errorf("%w: the recovery keyring material loader is not delivered", recovery.ErrAdmissionClosed)
}

// recoveryControlWiring is the exact maintenance surface seam. Every field is
// the single instance the root assembly already holds.
type recoveryControlWiring struct {
	Maintenance identity.RecoveryControlMaintenance
	Executor    recovery.IdentityRecoveryExecutor
	Reviews     identity.RecoveryControlReviewSource
	Complete    identity.RecoveryCompleteSignaler
	Phase       identity.RecoveryControlPhase
	Mode        identity.LoginRecoveryMode
	MaxBody     int64
}

// recoverySurfaceExit is the root RecoveryExit. It joins the real maintenance
// HTTP surface; a surface that never started refuses instead of reporting a
// successful exit.
type recoverySurfaceExit struct {
	mu   sync.Mutex
	join func(context.Context) error
}

var _ recovery.RecoveryExit = (*recoverySurfaceExit)(nil)

func (e *recoverySurfaceExit) bind(join func(context.Context) error) error {
	if e == nil || join == nil {
		return recovery.ErrAdmissionBinding
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.join != nil {
		return recovery.ErrAdmissionBusy
	}
	e.join = join
	return nil
}

// ExitRecovery revokes the maintenance surface on the exit chain. The
// permit and binding were already verified by the composition.
func (e *recoverySurfaceExit) ExitRecovery(ctx context.Context, _ recovery.PhasePermit, _ recovery.RunBinding) error {
	return e.joinConfigured(ctx)
}

// joinConfigured joins the maintenance surface without a phase permit so the
// ordinary lifecycle can reuse the same join owner. A surface that never
// started is a closed refusal, not a silent success.
func (e *recoverySurfaceExit) joinConfigured(ctx context.Context) error {
	if e == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	e.mu.Lock()
	join := e.join
	e.mu.Unlock()
	if join == nil {
		return fmt.Errorf("%w: the maintenance surface never started", recovery.ErrAdmissionClosed)
	}
	return join(ctx)
}

// recoveryExitRouter is the exit chain the single runtime observes. It
// forwards to the root composition only once that composition exists, so a
// real Runtime exit always performs capture, maintenance-surface join, the
// same-executor revoke and the Article settle. It never fakes a success:
// an unbound router is a closed refusal.
type recoveryExitRouter struct {
	mu     sync.Mutex
	target recovery.RecoveryExit
	bound  bool
}

var _ recovery.RecoveryExit = (*recoveryExitRouter)(nil)

func (r *recoveryExitRouter) bind(target recovery.RecoveryExit) error {
	if r == nil || missingRecoveryBridgeValue(target) {
		return recovery.ErrAdmissionBinding
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bound {
		return recovery.ErrAdmissionBusy
	}
	r.target, r.bound = target, true
	return nil
}

func (r *recoveryExitRouter) ExitRecovery(ctx context.Context, permit recovery.PhasePermit, binding recovery.RunBinding) error {
	if r == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	r.mu.Lock()
	target := r.target
	r.mu.Unlock()
	if target == nil {
		return fmt.Errorf("%w: the recovery exit chain was never bound", recovery.ErrAdmissionBinding)
	}
	return target.ExitRecovery(ctx, permit, binding)
}

// recoveryJoinedServer is the one control surface the lifecycle owns. join
// shuts it down and waits for the actual Serve call to return.
type recoveryJoinedServer struct {
	lifecycleServer
	done chan struct{}
}

func newRecoveryJoinedServer(server lifecycleServer) (*recoveryJoinedServer, error) {
	if server == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	return &recoveryJoinedServer{lifecycleServer: server, done: make(chan struct{})}, nil
}

func (s *recoveryJoinedServer) Serve(listener net.Listener) error {
	defer close(s.done)
	return s.lifecycleServer.Serve(listener)
}

func (s *recoveryJoinedServer) join(ctx context.Context) error {
	if s == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	shutdownErr := s.Shutdown(ctx)
	select {
	case <-s.done:
		return shutdownErr
	case <-ctx.Done():
		return errors.Join(shutdownErr, ctx.Err())
	}
}

// recoveryRootInputs are the single-instance inputs the root already owns.
// The optional seams below exist so the integration harness can hand the one
// root the same Store, Journal and ports the process would own. A nil field
// keeps the production constructor or the explicit closed placeholder at its
// original site; a seam never relaxes a closed default on its own.
type recoveryRootInputs struct {
	pool     *pgxpool.Pool
	plan     recoveryStartupPlan
	prepared *preparedLoginDependencies

	runs             *identity.RecoveryRunStore
	closures         *article.PostgresReplayEvidenceStore
	journal          recovery.ContextRecoveryJournal
	articleSnapshots recovery.VerifiedJournalReader
	restore          recovery.RestoreDriver
	consumer         recovery.RecoveryConsumer
	installation     recovery.InstallationExecutor
	articleAccount   identity.ArticleAccountPort
	exclusive        recovery.DeploymentExclusive
	commitments      recovery.InstallationCommitmentStore
	sources          recovery.InstallationSourceVerifier
	anchor           recovery.AnchorReader
	secretCandidate  func(context.Context, recovery.RunBinding) ([]byte, error)
	seal             func(recovery.Runtime) error
}

// recoveryEnterInstallMissingPorts reports every closed enter/install port in
// the fixed delivery order. A nil seam keeps its closed placeholder; the list
// is evidence of what is still missing, never a capability claim.
func recoveryEnterInstallMissingPorts(inputs recoveryRootInputs) []string {
	missing := make([]string, 0, 11)
	if inputs.journal == nil {
		missing = append(missing, "journal")
	}
	if inputs.articleSnapshots == nil {
		missing = append(missing, "articleSnapshots")
	}
	if inputs.restore == nil {
		missing = append(missing, "restore")
	}
	if inputs.consumer == nil {
		missing = append(missing, "consumer")
	}
	if inputs.installation == nil {
		missing = append(missing, "installation")
	}
	if inputs.articleAccount == nil {
		missing = append(missing, "articleAccount")
	}
	if inputs.commitments == nil {
		missing = append(missing, "commitments")
	}
	if inputs.sources == nil {
		missing = append(missing, "sources")
	}
	if inputs.anchor == nil {
		missing = append(missing, "anchor")
	}
	if inputs.exclusive == nil {
		missing = append(missing, "exclusive")
	}
	if inputs.secretCandidate == nil {
		missing = append(missing, "secretCandidate")
	}
	return missing
}

func recoveryEnterInstallClosedError(missing []string) error {
	return fmt.Errorf("%w: missing recovery enter/install ports: %s", recovery.ErrAdmissionClosed, strings.Join(missing, ", "))
}

// recoveryRootAssembly owns the one recovery composition: one pool, one Store,
// one Journal, one Runtime, one registry and one control surface.
type recoveryRootAssembly struct {
	plan        recoveryStartupPlan
	runtime     recovery.Runtime
	bridge      *recoveryIdentityBridge
	maintenance identity.RecoveryControlMaintenance
	executor    recovery.IdentityRecoveryExecutor
	reviews     identity.RecoveryControlReviewSource
	composition *recoveryRuntimeComposition
	exit        *recoverySurfaceExit
	ordinary    *recoveryOrdinaryLifecycle
	article     *recoveryControlHandle

	mu           sync.Mutex
	control      *recoveryJoinedServer
	closeArticle func() error
	joinArticle  func(context.Context) error
	stop         func()
	flowOnce     sync.Once
	flowDone     chan struct{}
	flowErr      error
}

func newRecoveryRootAssembly(ctx context.Context, inputs recoveryRootInputs) (*recoveryRootAssembly, error) {
	if ctx == nil || inputs.pool == nil || inputs.prepared == nil || inputs.prepared.clock == nil || !inputs.plan.present {
		return nil, recovery.ErrAdmissionBinding
	}
	cfg := inputs.plan.config
	switch inputs.plan.action {
	case "normal", "verify", "resume":
	case "enter", "install":
		if missing := recoveryEnterInstallMissingPorts(inputs); len(missing) > 0 {
			return nil, recoveryEnterInstallClosedError(missing)
		}
	default:
		return nil, recovery.ErrAdmissionBinding
	}
	var err error
	runs := inputs.runs
	if runs == nil {
		runs, err = identity.NewRecoveryRunStore(inputs.pool)
		if err != nil {
			return nil, err
		}
	}
	closures := inputs.closures
	if closures == nil {
		closures, err = article.NewPostgresReplayEvidenceStore(inputs.pool)
		if err != nil {
			return nil, err
		}
	}
	anchorRegistry, err := newRecoveryAnchorRegistry(nil)
	if err != nil {
		return nil, err
	}
	anchors := anchorRegistry.resolve(cfg.anchorAdapter, cfg.anchorConfig)
	if inputs.commitments != nil {
		anchors.Commitments = inputs.commitments
	}
	if inputs.sources != nil {
		anchors.Sources = inputs.sources
	}
	if inputs.anchor != nil {
		anchors.Anchor = inputs.anchor
	}
	if inputs.exclusive != nil {
		anchors.Exclusive = inputs.exclusive
	}
	genesis, err := recovery.NewInstallationReader(anchors.Commitments, anchors.Sources)
	if err != nil {
		return nil, err
	}
	journal := inputs.journal
	if journal == nil {
		keyring, keyringErr := recoveryLoadKeyring(cfg.keyringPath)
		if keyringErr != nil {
			return nil, keyringErr
		}
		journal, err = recoveryOpenJournal(ctx, recovery.JournalOptions{
			ReadMaxBytes:   cfg.maxBytes,
			ReadMaxRecords: cfg.maxRecords,
			Path:           cfg.journalPath,
			FileMode:       cfg.fileMode,
			Now:            inputs.prepared.clock.Now,
			JournalID:      cfg.journalID,
			Epoch:          cfg.epoch,
			Anchor:         recoveryClosedJournalAnchor{reason: "the recovery journal head anchor provider is not delivered"},
		}, keyring)
		if err != nil {
			return nil, errors.Join(recovery.ErrAdmissionClosed, err)
		}
	}
	articleSnapshots := inputs.articleSnapshots
	if articleSnapshots == nil {
		articleSnapshots = recoveryClosedVerifiedJournalReader{reason: "the verified journal reader is not delivered"}
	}
	restore := inputs.restore
	if restore == nil {
		restore = recoveryClosedRestoreDriver{}
	}
	consumer := inputs.consumer
	if consumer == nil {
		consumer = recoveryClosedConsumer{}
	}
	installation := inputs.installation
	if installation == nil {
		installation = recoveryClosedInstallationExecutor{}
	}
	articleAccount := inputs.articleAccount
	if articleAccount == nil {
		articleAccount = recoveryClosedArticleAccount{reason: "the Article deletion account port is not delivered"}
	}
	// The runtime's Exit port and the root composition form a construction
	// cycle: the composition needs the built Runtime, while the Runtime needs
	// the exit chain. The router closes it by binding the composition right
	// after construction; an unbound router refuses instead of succeeding.
	exitRouter := &recoveryExitRouter{}
	assembly := &recoveryRootAssembly{plan: inputs.plan, exit: &recoverySurfaceExit{}, flowDone: make(chan struct{})}
	handle, err := newRecoveryControlHandle(assembly.closeArticleOwner, assembly.joinArticleOwner, func() string { return "article" })
	if err != nil {
		return nil, err
	}
	ordinary, err := newRecoveryOrdinaryLifecycle(cfg.drainBudget, assembly.exit.joinConfigured)
	if err != nil {
		return nil, err
	}
	forwarder := identity.NewRecoveryIdentityForwarder()
	bridge, err := newRecoveryIdentityBridge(nil, forwarder, cfg.ioBudget)
	if err != nil {
		return nil, err
	}
	readCandidate := func(context.Context, recovery.RunBinding) ([]byte, error) {
		return nil, fmt.Errorf("%w: the recovery secret candidate source is not delivered", recovery.ErrAdmissionClosed)
	}
	if inputs.secretCandidate != nil {
		readCandidate = inputs.secretCandidate
	}
	secret, err := newRecoverySecretSource(readCandidate)
	if err != nil {
		return nil, err
	}
	runtimeValue, err := recovery.NewRuntime(ctx, recovery.RuntimeDependencies{
		Store:               runs,
		Journal:             journal,
		ArticleSnapshots:    articleSnapshots,
		Closures:            closures,
		Genesis:             genesis,
		Exclusive:           anchors.Exclusive,
		Lifecycle:           ordinary,
		Runs:                runs,
		Identity:            bridge,
		Restore:             restore,
		Consumer:            consumer,
		Exit:                exitRouter,
		RecoverySecret:      secret,
		InstallationWriter:  runs,
		Commitments:         anchors.Commitments,
		InstallationSources: anchors.Sources,
		Installation:        installation,
		ArticleAnchorReader: anchors.Anchor,
	})
	if err != nil {
		return nil, err
	}
	if inputs.seal != nil {
		if err := inputs.seal(runtimeValue); err != nil {
			return nil, fmt.Errorf("seal recovery publication ports: %w", err)
		}
	}
	if err := bridge.bindRuntime(runtimeValue); err != nil {
		return nil, err
	}
	executor, maintenance, err := identity.NewRecoveryIdentityExecution(identity.RecoveryIdentityExecutionDependencies{
		Store:     runs,
		Runtime:   runtimeValue,
		Journal:   journal,
		Articles:  articleAccount,
		Audit:     audit.Store{},
		Passwords: inputs.prepared.passwords,
		IDs:       inputs.prepared.ids,
		Clock:     inputs.prepared.clock,
	})
	if err != nil {
		return nil, err
	}
	if err := forwarder.Seal(executor); err != nil {
		return nil, err
	}
	reviews, err := identity.NewRecoveryReviewLister(inputs.pool, runtimeValue)
	if err != nil {
		return nil, err
	}
	composition, err := newRecoveryRuntimeComposition(recoveryRuntimePorts{
		Runtime:    runtimeValue,
		Identity:   bridge,
		Exit:       assembly.exit,
		Secret:     secret,
		Lifecycle:  ordinary,
		Article:    handle,
		JoinBudget: cfg.drainBudget,
	})
	if err != nil {
		return nil, err
	}
	if err := exitRouter.bind(composition); err != nil {
		return nil, err
	}
	assembly.bridge = bridge
	assembly.maintenance = maintenance
	assembly.executor = forwarder
	assembly.reviews = reviews
	assembly.composition = composition
	assembly.ordinary = ordinary
	assembly.article = handle
	assembly.runtime = runtimeValue
	return assembly, nil
}

// wiring is the maintenance control surface seam. A missing instance refuses.
func (a *recoveryRootAssembly) wiring() (*recoveryControlWiring, error) {
	if a == nil || a.bridge == nil || a.reviews == nil || a.composition == nil || missingRecoveryBridgeValue(a.maintenance) || missingRecoveryBridgeValue(a.executor) {
		return nil, recovery.ErrAdmissionBinding
	}
	mode, ok := a.maintenance.(identity.LoginRecoveryMode)
	if !ok {
		return nil, recovery.ErrAdmissionBinding
	}
	return &recoveryControlWiring{
		Maintenance: a.maintenance,
		Executor:    a.executor,
		Reviews:     a.reviews,
		Complete:    a.bridge,
		Phase:       a.composition.controlPhase,
		Mode:        mode,
		MaxBody:     recoveryControlMaxBodyBytes,
	}, nil
}

func (a *recoveryRootAssembly) maintenanceMode() bool {
	return a != nil && a.plan.maintenance
}

func (a *recoveryRootAssembly) controlAddress() string {
	if a == nil {
		return ""
	}
	return a.plan.config.controlAddress
}

// bindWorkers binds the three real owners. Verify binds none: it leaves zero
// workers and zero repair behind.
func (a *recoveryRootAssembly) bindWorkers(article, reset, notice recoveryStartedWorker) error {
	if a == nil || a.ordinary == nil || a.runtime == nil {
		return recovery.ErrAdmissionBinding
	}
	if a.plan.verify {
		return nil
	}
	return a.ordinary.bind(a.runtime, article, reset, notice)
}

// bindArticleOwner hands the long-lived Article control handle its real close
// and join owners. Install returning nil never means the handle is released.
func (a *recoveryRootAssembly) bindArticleOwner(closeArticle func() error, joinArticle func(context.Context) error) error {
	if a == nil || closeArticle == nil || joinArticle == nil {
		return recovery.ErrAdmissionBinding
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closeArticle != nil || a.joinArticle != nil {
		return recovery.ErrAdmissionBusy
	}
	a.closeArticle, a.joinArticle = closeArticle, joinArticle
	return nil
}

func (a *recoveryRootAssembly) closeArticleOwner() error {
	if a == nil {
		return recovery.ErrAdmissionBinding
	}
	a.mu.Lock()
	closeFn := a.closeArticle
	a.mu.Unlock()
	if closeFn == nil {
		return recovery.ErrAdmissionBinding
	}
	return closeFn()
}

func (a *recoveryRootAssembly) joinArticleOwner(ctx context.Context) error {
	if a == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	a.mu.Lock()
	joinFn := a.joinArticle
	a.mu.Unlock()
	if joinFn == nil {
		return recovery.ErrAdmissionBinding
	}
	return joinFn(ctx)
}

// bindControl binds the one started control surface so RecoveryExit can join it.
func (a *recoveryRootAssembly) bindControl(server lifecycleServer, listener net.Listener) (*recoveryJoinedServer, error) {
	// The lifecycle must serve the joined wrapper itself: binding the raw
	// server would leave the wrapper's Serve unexecuted and its done channel
	// open, so RecoveryExit would join a surface that never started.
	if a == nil || !a.plan.maintenance || server == nil || listener == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	joined, err := newRecoveryJoinedServer(server)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if a.control != nil {
		a.mu.Unlock()
		return nil, recovery.ErrAdmissionBusy
	}
	a.control = joined
	a.mu.Unlock()
	if bindErr := a.exit.bind(joined.join); bindErr != nil {
		return nil, bindErr
	}
	return joined, nil
}

func (a *recoveryRootAssembly) bindStop(stop func()) error {
	if a == nil || stop == nil {
		return recovery.ErrAdmissionBinding
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stop != nil {
		return recovery.ErrAdmissionBusy
	}
	a.stop = stop
	return nil
}

func (a *recoveryRootAssembly) requestStop() {
	if a == nil {
		return
	}
	a.mu.Lock()
	stop := a.stop
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// waitIdentityReady blocks until the identity permit was captured. The control
// surface must not accept traffic before that.
func (a *recoveryRootAssembly) waitIdentityReady(ctx context.Context) error {
	if a == nil || a.composition == nil {
		return recovery.ErrAdmissionBinding
	}
	return a.composition.waitIdentityReady(ctx)
}

// verify is the read-only proof. It starts no listener, no worker and no repair.
func (a *recoveryRootAssembly) verify(ctx context.Context, budget time.Duration) error {
	if a == nil || ctx == nil || a.runtime == nil || !a.plan.verify || budget <= 0 {
		return recovery.ErrAdmissionBinding
	}
	bounded, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	_, err := a.runtime.Verify(bounded)
	return err
}

func (a *recoveryRootAssembly) start(ctx context.Context, budget time.Duration) error {
	if a == nil || ctx == nil || a.runtime == nil || a.composition == nil || budget <= 0 {
		return recovery.ErrAdmissionBinding
	}
	switch a.plan.action {
	case "normal":
		bounded, cancel := context.WithTimeout(ctx, budget)
		defer cancel()
		err := a.runtime.RunNormal(bounded)
		a.setFlowResult(err)
		return err
	case "resume":
		bounded, cancel := context.WithTimeout(ctx, budget)
		go func() {
			defer cancel()
			a.setFlowResult(a.runtime.Resume(bounded, a.plan.config.command.binding))
		}()
		go func() {
			defer cancel()
			a.completeOnRequest(bounded)
		}()
		return nil
	}
	return recovery.ErrAdmissionBinding
}

// completeOnRequest is the root completion controller. The HTTP surface only
// signals after its accepted response was flushed; the real executor call runs
// here, off the HTTP stack, and only then does the recovery flow continue.
func (a *recoveryRootAssembly) completeOnRequest(ctx context.Context) {
	if err := a.composition.waitCompleteRequest(ctx); err != nil {
		a.setFlowResult(err)
		a.requestStop()
		return
	}
	if err := a.composition.completeIdentity(ctx); err != nil {
		a.setFlowResult(err)
		a.requestStop()
		return
	}
	a.setFlowResult(a.waitFlow(ctx))
	a.requestStop()
}

func (a *recoveryRootAssembly) setFlowResult(err error) {
	a.flowOnce.Do(func() {
		a.flowErr = err
		close(a.flowDone)
	})
}

func (a *recoveryRootAssembly) flowFinished() bool {
	select {
	case <-a.flowDone:
		return true
	default:
		return false
	}
}

func (a *recoveryRootAssembly) waitFlow(ctx context.Context) error {
	select {
	case <-a.flowDone:
		return a.flowErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// joinWork is the lifecycle join: it never leaves a flow, worker or call behind.
func (a *recoveryRootAssembly) joinWork(ctx context.Context) error {
	if a == nil || ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	if a.runtime == nil {
		return nil
	}
	switch a.plan.action {
	case "normal":
		if !a.flowFinished() {
			return recovery.ErrAdmissionBinding
		}
		return a.waitFlow(ctx)
	case "resume":
	default:
		return recovery.ErrAdmissionBinding
	}
	if a.flowFinished() {
		return a.waitFlow(ctx)
	}
	stopErr := a.runtime.Stop(ctx)
	return errors.Join(stopErr, a.waitFlow(ctx))
}

// recoveryControlOptions carries the existing identity observation seams into
// the maintenance surface. Nothing here is a second authority: the same pool,
// the same prepared identity inputs and the same runtime are reused.
type recoveryControlOptions struct {
	prepared     *preparedLoginDependencies
	logger       *slog.Logger
	observer     identity.LoginOutcomeObserver
	auditor      identity.LoginFailureAuditor
	auditFor     time.Duration
	observeFor   time.Duration
	maxBodyBytes int64
	ready        func() bool
}

// identityReady reports whether the identity permit was already captured. It
// never blocks and never mints a permit.
func (a *recoveryRootAssembly) identityReady() bool {
	if a == nil || a.composition == nil {
		return false
	}
	bridge := a.composition.bridgeValue()
	if bridge == nil {
		return false
	}
	select {
	case <-bridge.ready:
		return true
	default:
		return false
	}
}

// awaitControlReady waits for the retained identity permit or for the flow to
// fail before one existed. The control surface accepts no traffic first.
func (a *recoveryRootAssembly) awaitControlReady(ctx context.Context) error {
	if a == nil || ctx == nil || a.composition == nil {
		return recovery.ErrAdmissionBinding
	}
	ready := make(chan error, 1)
	go func() { ready <- a.composition.waitIdentityReady(ctx) }()
	select {
	case err := <-ready:
		return err
	case <-a.flowDone:
	}
	if a.flowErr != nil {
		return a.flowErr
	}
	select {
	case err := <-ready:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// newRecoveryArticleJoin waits for every bound ordinary owner to actually
// finish before the Article control handle returns deployment ownership.
func newRecoveryArticleJoin(runtime *businessRuntime) func(context.Context) error {
	return func(ctx context.Context) error {
		if ctx == nil || runtime == nil {
			return recovery.ErrAdmissionBinding
		}
		for _, worker := range []lifecycleWorker{runtime.articleWorker, runtime.passwordResetWorker, runtime.noticeWorker} {
			if worker == nil {
				continue
			}
			// A replay exits before ReconcileAndStart, so an owner that never
			// started can never close Done. Only a proved started owner is
			// joined; waiting for an unstarted owner would burn the drain
			// budget and skip the executor revoke.
			started, ok := worker.(recoveryStartedWorker)
			if !ok {
				continue
			}
			select {
			case <-started.Started():
			default:
				continue
			}
			done := workerDone(worker)
			if done == nil {
				continue
			}
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}

func recoveryControlProbe(status int, ready func() bool) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		if ready != nil && !ready() {
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		response.WriteHeader(status)
	}
}

// newRecoveryControlAuthorizer binds the maintenance authorizer to the same
// sealed executor the control routes drive, through the maintenance session
// store. It never falls back to the ordinary session store: ordinary resolution
// requires access_scope='ordinary' and would reject every maintenance session.
func newRecoveryControlAuthorizer(executor recovery.IdentityRecoveryExecutor, gate session.PasswordResetGate, clock func() time.Time) (platform.HTTPAuthorizer, error) {
	if missingRecoveryBridgeValue(executor) || clock == nil {
		return platform.HTTPAuthorizer{}, recovery.ErrAdmissionBinding
	}
	store, err := identity.NewRecoveryMaintenanceStore(executor, gate)
	if err != nil {
		return platform.HTTPAuthorizer{}, err
	}
	authorizer, err := platform.NewHTTPAuthorizer(store, clock)
	if err != nil {
		return platform.HTTPAuthorizer{}, err
	}
	return authorizer, nil
}

// newProductionRecoveryControlHandler composes the maintenance surface from the
// same pool and prepared identity inputs the business surface already uses.
func newProductionRecoveryControlHandler(pool *pgxpool.Pool, options recoveryControlOptions, wiring *recoveryControlWiring) (http.Handler, error) {
	if pool == nil || wiring == nil || options.prepared == nil ||
		options.observer == nil || options.auditor == nil ||
		options.auditFor <= 0 || options.observeFor <= 0 || options.maxBodyBytes <= 0 {
		return nil, recovery.ErrAdmissionBinding
	}
	gate := &identity.PasswordResetStore{Idempotency: idempotency.Store{}}
	value, err := session.NewStore(session.StoreDependencies{Transactions: pool, PasswordResetGate: gate})
	if err != nil {
		return nil, errors.New("create recovery maintenance session store")
	}
	store := &value
	abuse, err := identity.NewPersistentLoginAbuseProtector(identity.PostgresLoginAbuseBucketStore{}, options.prepared.abuseDigester, options.prepared.abuseConfig)
	if err != nil {
		return nil, errors.New("create recovery maintenance abuse protector")
	}
	service, err := identity.NewLoginService(identity.LoginDependencies{
		Transactions: pool, Repository: identity.PostgresLoginRepository{},
		PasswordResetGate: gate, Passwords: options.prepared.passwords,
		Sessions: store, SessionIssuer: options.prepared.sessionIssuer,
		Abuse: abuse, RecoveryMode: wiring.Mode, Audit: &audit.Store{},
		IDs: options.prepared.ids, Clock: options.prepared.clock,
		FakeCredentialDigest: options.prepared.fakeDigest,
		FakeCredentialProbe:  options.prepared.fakeProbe,
	})
	if err != nil {
		return nil, errors.New("create recovery maintenance login service")
	}
	options.prepared.fakeProbe = ""
	coordinator, err := identity.NewLoginAttemptCoordinator(options.prepared.ids, options.prepared.clock, options.auditor)
	if err != nil {
		return nil, errors.New("create recovery maintenance login coordinator")
	}
	login, err := identity.NewLoginHandler(service, options.prepared.source, options.maxBodyBytes, identity.LoginHandlerConfig{
		Coordinator: coordinator, Observer: options.observer, ObservationBudget: options.observeFor,
	})
	if err != nil {
		return nil, errors.New("create recovery maintenance login handler")
	}
	authorizer, err := newRecoveryControlAuthorizer(wiring.Executor, gate, func() time.Time { return options.prepared.clock.Now() })
	if err != nil {
		return nil, fmt.Errorf("create recovery maintenance authorizer: %w", err)
	}
	handlers, err := identity.NewRecoveryControlHandlers(identity.RecoveryControlDependencies{
		Maintenance:  wiring.Maintenance,
		Reviews:      wiring.Reviews,
		TrustChange:  service,
		Complete:     wiring.Complete,
		Authorizer:   authorizer,
		Phase:        wiring.Phase,
		Session:      login.Login,
		Liveness:     recoveryControlProbe(http.StatusOK, nil),
		Readiness:    recoveryControlProbe(http.StatusOK, options.ready),
		MaxBodyBytes: wiring.MaxBody,
	})
	if err != nil {
		return nil, err
	}
	return handlers.Handler(), nil
}

// recoveryStartedWorkerFrom requires the real started proof. A worker without
// it cannot be reconciled by the ordinary lifecycle.
func recoveryStartedWorkerFrom(worker lifecycleWorker) (recoveryStartedWorker, error) {
	if missingRecoveryBridgeValue(worker) {
		return nil, recovery.ErrAdmissionBinding
	}
	started, ok := worker.(recoveryStartedWorker)
	if !ok {
		return nil, fmt.Errorf("%w: the ordinary worker exposes no started proof", recovery.ErrAdmissionBinding)
	}
	return started, nil
}

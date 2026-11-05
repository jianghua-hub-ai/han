package recovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Phase purposes and Journal uses are stable consumer-neutral capability vocabulary.
const (
	PhaseStartupReconcile   = "startup_reconcile"
	PhaseRecoveryReplay     = "recovery_replay"
	PhaseRecoveryIdentity   = "recovery_identity"
	PhaseRecoveryRestore    = "recovery_restore"
	PhaseRecoveryBind       = "recovery_bind"
	JournalUseCommittedRead = "committed_read"
	JournalUseNewAppend     = "new_append"
	JournalUseCoordinate    = "coordinate"
	phaseInstall            = "install"
	runtimeClosed           = "closed"
	runtimeChecking         = "checking"
	runtimePublicReady      = "public_ready"
	runtimeFaulted          = "faulted"
	runtimeDraining         = "draining"
	runtimeStopped          = "stopped"
	causeArticleAppend      = "article_append"
	causeArticleAppendFault = "article_append_fault"
)

// RunBinding is an exact recovery run and backup pair.
type RunBinding struct{ RunID, BackupRef string }

// RecoveryInput belongs to the root; its candidate secret is never serialized.
type RecoveryInput struct {
	RunBinding
	Secret []byte `json:"-"`
}

// RestorePlan contains approved source and target descriptors, never credentials.
type RestorePlan struct {
	Version, RunID, BackupRef, DeploymentID, InstallationID, TargetDigest         string
	ArtifactDigest, SourceApprovalRef, SourceApprovalDigest, SourceInstallationID string
	SourceSchemaDigest, ArtifactPath                                              string
	ArtifactBytes                                                                 int64
}

// RestoreDriver owns real artifact/target verification and joins all tool work.
type RestoreDriver interface {
	Prepare(context.Context, RunBinding) (RestorePlan, error)
	Replace(context.Context, PhasePermit, RestorePlan) (RestoreReceipt, error)
	VerifyRestored(context.Context, RestorePlan) (RestoreReceipt, error)
}

// IdentityRecoveryExecutor delegates identity transitions to the unique Store owner.
type IdentityRecoveryExecutor interface {
	BindRestoredRun(context.Context, PhasePermit, RunBinding, RestoreReceipt) (RecoveryRun, error)
	FinalizeRestoredIdentity(context.Context, PhasePermit, RunBinding) error
	PrepareRecoveryTrust(context.Context, PhasePermit, RunBinding, []byte) error
	CompleteIdentityReview(context.Context, PhasePermit, RunBinding) error
	RevokeRecoverySessions(context.Context, PhasePermit, RunBinding) error
}

// RecoveryConsumer is implemented by the unique contextual Coordinator.
type RecoveryConsumer interface {
	Consume(context.Context, string) (ConsumptionResult, error)
}

// RecoveryExit joins the root-owned maintenance surface, then confirms same-Store revocation.
type RecoveryExit interface {
	ExitRecovery(context.Context, PhasePermit, RunBinding) error
}

// RecoverySecretSource only reads a candidate; the Journal separately verifies it.
type RecoverySecretSource interface {
	ReadInitialRecoverySecret(context.Context, RunBinding) ([]byte, error)
}

// WorkKind identifies ordinary work covered until actual completion.
type WorkKind string

const (
	WorkRequest         WorkKind = "request"
	WorkArticleDeletion WorkKind = "article_deletion"
	WorkPasswordReset   WorkKind = "password_reset"
	WorkArticleNotice   WorkKind = "article_notice"
)

// PhasePermit is an opaque, process-local purpose capability.
type PhasePermit struct{ grant *phaseGrant }
type phaseGrant struct {
	owner        *admissionRuntime
	generation   int64
	binding      PermitBinding
	ctx          context.Context
	cancel       context.CancelFunc
	installation *installationObservation
}

// WorkLease covers the original context and the complete work lifetime.
type WorkLease interface {
	Context() context.Context
	Release()
}

// CloseCause describes a private runtime-owned close reason.
type CloseCause struct {
	Code        string
	OperationID string
}

// AppendBarrier is an opaque one-operation close-cause capability.
type AppendBarrier struct{ token *appendToken }
type appendToken struct {
	owner      *admissionRuntime
	generation int64
	operation  string
	head       JournalHead
	revision   string
}

// VerifiedHeadAdvance is the writer's independently confirmed CAS progression.
type VerifiedHeadAdvance struct {
	JournalID, Namespace                                         string
	Epoch                                                        uint64
	OldHead, NewHead                                             JournalHead
	OldRevision, PendingRevision, CommittedRevision, OperationID string
}

// PermitBinding is observation only; it cannot be submitted as authorization.
type PermitBinding struct {
	RuntimeID                                 string
	Generation                                int64
	RunID, BackupRef, InstallationID, Purpose string
}

// AdmissionObservation is a sanitized classification, never a capability.
type AdmissionObservation struct{ State, Reason string }

// DeploymentExclusive owns a deployment-wide lease outside the restore domain.
type DeploymentExclusive interface {
	Acquire(context.Context) (ExclusiveLease, error)
}

// ExclusiveLease checks the actual exclusive facility and releases its handle.
type ExclusiveLease interface {
	Check(context.Context) error
	Close() error
}

// OrdinaryLifecycle returns only after actual startup or actual stop/join.
type OrdinaryLifecycle interface {
	ReconcileAndStart(context.Context, PhasePermit) error
	StopAndJoin(context.Context) error
}

// InstallationManifest preserves the approved v1 installation source values.
type InstallationManifest struct {
	Version, Kind, DeploymentID, InstallationID, TargetDigest string
	SourceApprovalRef, SourceApprovalDigest                   string
	Recovery, Article                                         AnchorSnapshot
	IssuedAt                                                  time.Time
}

// InstallationCommitment is the independent prepared or bound installation key.
type InstallationCommitment struct {
	Manifest              InstallationManifest
	ManifestDigest, Phase string
}

// InstallationRepository is implemented by the unique identity Store.
type InstallationRepository interface {
	BindInstallation(context.Context, PhasePermit, InstallationBinding) error
	BindLegacyPublication(context.Context, PhasePermit, string, PublicationBinding) error
}

// InstallationCommitmentStore owns the independent installation CAS key.
type InstallationCommitmentStore interface {
	LoadInstallation(context.Context, string) (InstallationCommitment, string, error)
	CompareAndSwapInstallation(context.Context, string, string, InstallationCommitment) (InstallationCommitment, string, error)
}

// InstallationSourceVerifier verifies independent approval, never current media.
type InstallationSourceVerifier interface {
	VerifyInstallationSource(context.Context, InstallationManifest) error
}

// InstallationExecutor returns only after installation and its child I/O finish.
type InstallationExecutor interface {
	Install(context.Context, PhasePermit, InstallationManifest) error
}

// AnchorReader exposes only the existing article backend's actual namespace key.
type AnchorReader interface {
	Load(context.Context, string) (AnchorSnapshot, string, error)
}

// RuntimeDependencies contains the neutral ports used by admission and permits.
type RuntimeDependencies struct {
	Store               AdmissionStore
	Journal             ContextRecoveryJournal
	ArticleSnapshots    VerifiedJournalReader
	Closures            PublishedClosureReader
	Genesis             InstallationReader
	Exclusive           DeploymentExclusive
	Lifecycle           OrdinaryLifecycle
	Runs                RecoveryRunStore
	Identity            IdentityRecoveryExecutor
	Restore             RestoreDriver
	Consumer            RecoveryConsumer
	Exit                RecoveryExit
	RecoverySecret      RecoverySecretSource
	InstallationWriter  InstallationRepository
	Commitments         InstallationCommitmentStore
	InstallationSources InstallationSourceVerifier
	Installation        InstallationExecutor
	ArticleAnchorReader AnchorReader
}

// Runtime provides admission and scoped capabilities; it never persists ready.
type Runtime interface {
	Install(context.Context, InstallationManifest) error
	RunNormal(context.Context) error
	Enter(context.Context, RecoveryInput) error
	Resume(context.Context, RunBinding) error
	Verify(context.Context) (AdmissionObservation, error)
	Acquire(context.Context, WorkKind) (WorkLease, error)
	CheckPhase(context.Context, PhasePermit, string) error
	PhaseContext(context.Context, PhasePermit) (context.Context, error)
	ReadPermitBinding(context.Context) (PermitBinding, error)
	CheckJournalUse(context.Context, string, string) error
	CheckArticleCoordination(context.Context, PhasePermit, string, string, uint64) (AnchorSnapshot, string, error)
	PauseArticleAppend(context.Context, string) (AppendBarrier, error)
	CompleteArticleAppend(context.Context, AppendBarrier, VerifiedHeadAdvance) error
	FaultArticleAppend(context.Context, AppendBarrier, error) error
	Stop(context.Context) error
	// ShutdownContext returns the first selected shutdown context, without selecting one.
	ShutdownContext() (context.Context, bool)
}

// Install is the closed installation producer; concrete installation is delegated.
func (r *admissionRuntime) Install(ctx context.Context, m InstallationManifest) (result error) {
	if e := r.installPreflight(ctx, m); e != nil {
		return e
	}
	if e := r.beginAction(ctx, false); e != nil {
		return e
	}
	finished := false
	defer func() {
		if finished && result == nil {
			result = r.acceptInstallation(ctx)
		}
		if !finished {
			r.fail(ErrRecoveryWriteUnknown)
		} else if result != nil {
			r.fail(result)
		}
		r.finishInstallation()
		close(r.normalDone)
	}()
	f := &installationFlow{r: r, manifest: m, digest: installationManifestDigest(m)}
	result = f.run(r.generationCtx)
	finished = true
	return result
}

func (r *admissionRuntime) acceptInstallation(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if r.phase == nil || !r.validPhaseLocked(PhasePermit{grant: r.phase}, phaseInstall) {
		return ErrAdmissionClosed
	}
	r.phase.cancel()
	r.phase = nil
	return nil
}

// CheckArticleCoordination validates only the current exact installation grant.
func (r *admissionRuntime) CheckArticleCoordination(ctx context.Context, p PhasePermit, installation, journal string, epoch uint64) (AnchorSnapshot, string, error) {
	if e := installationContext(ctx); e != nil {
		return AnchorSnapshot{}, "", e
	}
	if e := r.checkOwnership(ctx); e != nil {
		return AnchorSnapshot{}, "", e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := ctx.Value(runtimeContextKey{}).(runtimeAuthorization)
	if !ok || a.phase != p.grant || !r.validPhaseLocked(p, phaseInstall) || p.grant.installation == nil {
		return AnchorSnapshot{}, "", ErrAdmissionBinding
	}
	o := p.grant.installation
	if installation != o.installation.InstallationID || journal != o.article.JournalID || epoch != o.article.Epoch {
		return AnchorSnapshot{}, "", ErrAdmissionBinding
	}
	if e := checkedContext(ctx); e != nil {
		return AnchorSnapshot{}, "", e
	}
	return copyWriterAnchor(o.article), o.articleRevision, nil
}

func installationContext(ctx context.Context) error {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if _, ok := ctx.Deadline(); !ok {
		return ErrAdmissionBinding
	}
	return nil
}

func (r *admissionRuntime) installPreflight(ctx context.Context, m InstallationManifest) error {
	if e := installationContext(ctx); e != nil {
		return e
	}
	if e := validateInstallationManifest(m); e != nil {
		return e
	}
	for _, p := range []any{r.deps.Installation, r.deps.ArticleAnchorReader, r.deps.Lifecycle, r.deps.Commitments, r.deps.InstallationSources} {
		if missingRuntimePort(p) {
			return ErrAdmissionBinding
		}
	}
	return nil
}

func validateInstallationManifest(m InstallationManifest) error {
	if m.Version != InstallationVersion || m.Kind != "genesis" && m.Kind != "anchored_migration" {
		return ErrAdmissionBinding
	}
	for _, s := range []string{m.DeploymentID, m.InstallationID, m.SourceApprovalRef} {
		if !admissionText(s) {
			return ErrAdmissionBinding
		}
	}
	if !sha256DigestPattern.MatchString(m.TargetDigest) || !sha256DigestPattern.MatchString(m.SourceApprovalDigest) || !admissionTime(m.IssuedAt) {
		return ErrAdmissionBinding
	}
	if !validateInitial(m.Recovery, RecoveryNamespaceV1) || !validateInitial(m.Article, ArticleDeletionNamespaceV1) {
		return ErrAdmissionBinding
	}
	if m.Kind == "genesis" && (m.Recovery.Committed.Sequence != 0 || m.Article.Committed.Sequence != 0) {
		return ErrAdmissionBinding
	}
	return nil
}

func encodeManifestNamespace(a AnchorSnapshot) []byte {
	return encodeAdmissionObject([]admissionField{{"namespace", a.Namespace}, {"journal_id", a.JournalID}, {"epoch", a.Epoch}, {"head_sequence", a.Committed.Sequence}, {"head_hash", a.Committed.Hash}})
}

func installationManifestDigest(m InstallationManifest) string {
	return SHA256Digest(encodeAdmissionObject([]admissionField{
		{"version", m.Version}, {"kind", m.Kind}, {"deployment_id", m.DeploymentID}, {"installation_id", m.InstallationID}, {"target_digest", m.TargetDigest},
		{"source_approval_ref", m.SourceApprovalRef}, {"source_approval_digest", m.SourceApprovalDigest},
		{"recovery", json.RawMessage(encodeManifestNamespace(m.Recovery))}, {"article", json.RawMessage(encodeManifestNamespace(m.Article))}, {"issued_at", m.IssuedAt},
	}))
}

type installationObservation struct {
	installation    InstallationBinding
	store           StoreRevision
	storeBytes      string
	recovery        VerifiedJournalSnapshot
	article         AnchorSnapshot
	articleRevision string
	commitment      InstallationCommitment
	commitRevision  string
}

type installationFlow struct {
	r                      *admissionRuntime
	manifest               InstallationManifest
	digest                 string
	classificationRevision string
}

func (f *installationFlow) alive(ctx context.Context) error {
	if e := installationContext(ctx); e != nil {
		return e
	}
	f.r.mu.Lock()
	defer f.r.mu.Unlock()
	if f.r.generationCtx.Err() != nil || !f.r.starting || f.r.state != runtimeChecking && f.r.state != phaseInstall {
		return ErrAdmissionClosed
	}
	return nil
}

func (f *installationFlow) guard(ctx context.Context) error {
	if e := f.alive(ctx); e != nil {
		return e
	}
	return f.r.checkOwnership(ctx)
}

func (f *installationFlow) commitment(ctx context.Context) (InstallationCommitment, string, bool, error) {
	if e := f.alive(ctx); e != nil {
		return InstallationCommitment{}, "", false, e
	}
	if e := f.r.deps.InstallationSources.VerifyInstallationSource(ctx, f.manifest); e != nil {
		return InstallationCommitment{}, "", false, e
	}
	if e := f.alive(ctx); e != nil {
		return InstallationCommitment{}, "", false, e
	}
	c, v, e := f.r.deps.Commitments.LoadInstallation(ctx, f.manifest.DeploymentID)
	if ce := f.alive(ctx); ce != nil {
		return InstallationCommitment{}, "", false, errors.Join(e, ce)
	}
	if errors.Is(e, ErrAnchorMissing) {
		return InstallationCommitment{}, "", true, nil
	}
	if e != nil {
		return InstallationCommitment{}, "", false, e
	}
	if c.Manifest != f.manifest || c.ManifestDigest != f.digest || !admissionText(v) || c.Phase != "prepared" && c.Phase != "bound" {
		return InstallationCommitment{}, "", false, ErrAdmissionBinding
	}
	return c, v, false, nil
}

func (f *installationFlow) run(ctx context.Context) error {
	c, v, fresh, e := f.commitment(ctx)
	if e != nil {
		return e
	}
	f.classificationRevision = v
	if (fresh || c.Phase == "prepared") && missingRuntimePort(f.r.deps.InstallationWriter) {
		return ErrAdmissionBinding
	}
	if e := f.acquire(ctx); e != nil {
		return e
	}
	c2, v2, fresh2, e := f.commitment(ctx)
	if e != nil {
		return e
	}
	if c2 != c || v2 != v || fresh2 != fresh {
		return ErrAdmissionDrift
	}
	var before *installationObservation
	if !fresh && c.Phase == "bound" {
		o, e := f.stableObservation(ctx)
		if e != nil {
			return e
		}
		if o.commitment != c || o.commitRevision != v {
			return ErrAdmissionDrift
		}
		before = &o
	}
	p, pc, e := f.grant(ctx, before)
	if e != nil {
		return e
	}
	if e := f.r.deps.Installation.Install(pc, p, f.manifest); e != nil {
		return e
	}
	if e := f.guard(ctx); e != nil {
		return e
	}
	return f.verifyResult(ctx, before)
}

func (f *installationFlow) acquire(ctx context.Context) error {
	if e := f.alive(ctx); e != nil {
		return e
	}
	if e := f.r.deps.Lifecycle.StopAndJoin(ctx); e != nil {
		return e
	}
	if e := f.alive(ctx); e != nil {
		return e
	}
	l, e := f.r.deps.Exclusive.Acquire(ctx)
	f.r.mu.Lock()
	f.r.exclusive = l
	f.r.mu.Unlock()
	if e != nil {
		return e
	}
	return f.guard(ctx)
}

func (f *installationFlow) grant(ctx context.Context, o *installationObservation) (PhasePermit, context.Context, error) {
	if e := f.guard(ctx); e != nil {
		return PhasePermit{}, nil, e
	}
	r := f.r
	r.mu.Lock()
	if r.generationCtx.Err() != nil || r.state != runtimeChecking {
		r.mu.Unlock()
		return PhasePermit{}, nil, ErrAdmissionClosed
	}
	r.proof.installation = InstallationBinding{Version: f.manifest.Version, DeploymentID: f.manifest.DeploymentID, InstallationID: f.manifest.InstallationID, TargetDigest: f.manifest.TargetDigest, CommitmentRef: "recovery/v1/installation/" + f.manifest.DeploymentID, CommitmentDigest: f.digest, RecoveryInitial: f.manifest.Recovery, ArticleInitial: f.manifest.Article}
	if o != nil {
		r.proof.installation = o.installation
		sealed := *o
		sealed.article = copyWriterAnchor(o.article)
		o = &sealed
	}
	p := r.phaseLocked(phaseInstall)
	p.grant.installation = o
	r.mu.Unlock()
	pc, e := r.PhaseContext(ctx, p)
	return p, pc, e
}

func (f *installationFlow) observe(ctx context.Context) (installationObservation, error) {
	var o installationObservation
	if e := f.guard(ctx); e != nil {
		return o, e
	}
	c, v, fresh, e := f.commitment(ctx)
	if e != nil {
		return o, e
	}
	if fresh || c.Phase != "bound" {
		return o, ErrAdmissionIncomplete
	}
	o.commitment, o.commitRevision = c, v
	s, e := f.observeStore(ctx, &o)
	if e != nil {
		return o, e
	}
	j, e := f.r.deps.Journal.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if e != nil {
		return o, admissionReadError(e)
	}
	if e := f.guard(ctx); e != nil {
		return o, e
	}
	if e := validateVerifiedJournal(j, f.manifest.Recovery); e != nil {
		return o, e
	}
	if e := validateInstallationRuns(s, j); e != nil {
		return o, e
	}
	o.recovery = copyControlledJournal(j)
	a, token, e := f.r.deps.ArticleAnchorReader.Load(ctx, ArticleDeletionNamespaceV1)
	if e != nil {
		return o, admissionReadError(e)
	}
	if e := f.guard(ctx); e != nil {
		return o, e
	}
	if e := f.validArticleAnchor(a, token); e != nil {
		return o, e
	}
	o.article, o.articleRevision = copyWriterAnchor(a), token
	return o, nil
}

func (f *installationFlow) observeStore(ctx context.Context, o *installationObservation) (AdmissionSnapshot, error) {
	s, e := f.r.deps.Store.ReadAdmissionSnapshot(ctx)
	if e != nil {
		return s, admissionReadError(e)
	}
	if e := f.guard(ctx); e != nil {
		return s, e
	}
	raw, e := CanonicalAdmissionSnapshot(s)
	if e != nil {
		return s, e
	}
	if s.Revision.Digest != SHA256Digest(raw) {
		return s, ErrAdmissionDrift
	}
	if !f.matchesInstallation(s.Installation) {
		return s, ErrAdmissionBinding
	}
	o.installation, o.store, o.storeBytes = s.Installation, s.Revision, string(raw)
	return s, nil
}

func (f *installationFlow) matchesInstallation(i InstallationBinding) bool {
	m := f.manifest
	return i.Version == m.Version && i.DeploymentID == m.DeploymentID && i.InstallationID == m.InstallationID && i.TargetDigest == m.TargetDigest &&
		i.CommitmentDigest == f.digest && i.CommitmentRef == "recovery/v1/installation/"+m.DeploymentID && i.RecoveryInitial == m.Recovery && i.ArticleInitial == m.Article
}

func validateInstallationRuns(s AdmissionSnapshot, j VerifiedJournalSnapshot) error {
	runs := make(map[string]AdmissionRun, len(s.Runs))
	for _, x := range s.Runs {
		runs[x.RunID] = x
	}
	begun, consumed := make(map[string]JournalEntry), make(map[string]bool)
	for _, e := range j.Entries {
		x, ok := runs[e.RunID]
		if !ok || e.Stage == StageConsumed && x.JournalConsumedAt == nil {
			return ErrAdmissionIncomplete
		}
		if e := matchHistoryEntry(e, x, begun, consumed); e != nil {
			return e
		}
	}
	for _, x := range s.Runs {
		if _, ok := begun[x.RunID]; !ok || (x.JournalStage == StageConsumed) != consumed[x.RunID] {
			return ErrAdmissionIncomplete
		}
	}
	return nil
}

func (f *installationFlow) validArticleAnchor(a AnchorSnapshot, token string) error {
	i := f.manifest.Article
	if a.Namespace != i.Namespace || a.JournalID != i.JournalID || a.Epoch != i.Epoch || !admissionText(token) || !admissionHead(a.Committed) {
		return ErrAdmissionBinding
	}
	if a.Committed.Sequence < i.Committed.Sequence || a.Committed.Sequence == i.Committed.Sequence && a.Committed != i.Committed {
		return ErrAdmissionDrift
	}
	if a.Pending == nil {
		return nil
	}
	p := a.Pending
	parts := strings.Split(p.OperationID, ":")
	if p.OldHead != a.Committed || p.OldHead.Sequence >= math.MaxInt64 || p.NewHead.Sequence != p.OldHead.Sequence+1 || !admissionHead(p.NewHead) ||
		len(parts) != 3 || parts[0] != "article-append/v1" || !admissionUUID(parts[1]) || !sha256DigestPattern.MatchString("sha256:"+parts[2]) {
		return ErrAdmissionBinding
	}
	return nil
}

func sameInstallationObservation(a, b installationObservation) bool {
	return a.installation == b.installation && a.store == b.store && a.storeBytes == b.storeBytes && sameControlledJournal(a.recovery, b.recovery) &&
		anchorSnapshotsEqual(a.article, b.article) && a.articleRevision == b.articleRevision && a.commitment == b.commitment && a.commitRevision == b.commitRevision
}

func (f *installationFlow) stableObservation(ctx context.Context) (installationObservation, error) {
	a, e := f.observe(ctx)
	if e != nil {
		return a, e
	}
	b, e := f.observe(ctx)
	if e != nil {
		return a, e
	}
	if !sameInstallationObservation(a, b) {
		return a, ErrAdmissionDrift
	}
	return a, nil
}

func (f *installationFlow) verifyResult(ctx context.Context, before *installationObservation) error {
	after, e := f.stableObservation(ctx)
	if e != nil {
		return e
	}
	if after.article.Pending != nil {
		return ErrAdmissionIncomplete
	}
	if before != nil {
		unchanged := after
		unchanged.article, unchanged.articleRevision = before.article, before.articleRevision
		if !sameInstallationObservation(*before, unchanged) || !installationSuccessor(*before, after) {
			return ErrAdmissionDrift
		}
	} else {
		if after.commitRevision == f.classificationRevision || after.recovery.Head != f.manifest.Recovery.Committed || after.article.Committed != f.manifest.Article.Committed {
			return ErrAdmissionDrift
		}
		if f.manifest.Kind == "genesis" && len(after.recovery.Entries) != 0 {
			return ErrAdmissionBinding
		}
	}
	s, e := f.r.deps.ArticleSnapshots.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if e != nil {
		return admissionReadError(e)
	}
	if e := f.guard(ctx); e != nil {
		return e
	}
	if e := validateVerifiedJournal(s, f.manifest.Article); e != nil {
		return e
	}
	if s.Head != after.article.Committed || s.Revision != after.articleRevision {
		return ErrAdmissionDrift
	}
	last, e := f.observe(ctx)
	if e != nil {
		return e
	}
	if !sameInstallationObservation(after, last) {
		return ErrAdmissionDrift
	}
	return f.guard(ctx)
}

func installationSuccessor(before, after installationObservation) bool {
	p := before.article.Pending
	if p == nil {
		return before.article.Committed == after.article.Committed && before.articleRevision == after.articleRevision
	}
	return before.articleRevision != after.articleRevision && (after.article.Committed == p.OldHead || after.article.Committed == p.NewHead)
}

func (r *admissionRuntime) finishInstallation() {
	r.mu.Lock()
	if r.phase != nil {
		r.phase.cancel()
		r.phase = nil
	}
	if r.cancelGeneration != nil {
		r.cancelGeneration()
	}
	r.starting = false
	if r.state == runtimeChecking || r.state == phaseInstall {
		r.state = phaseInstall
	}
	r.signalLocked()
	watch := r.ownerWatch
	r.mu.Unlock()
	watch.join()
}

// NewRuntime constructs a closed runtime without accessing any port.
func NewRuntime(ctx context.Context, d RuntimeDependencies) (Runtime, error) {
	if err := checkedContext(ctx); err != nil {
		return nil, err
	}
	for _, port := range []any{d.Store, d.Journal, d.ArticleSnapshots, d.Closures, d.Genesis, d.Exclusive} {
		if missingRuntimePort(port) {
			return nil, ErrAdmissionBinding
		}
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, errors.Join(ErrAdmissionClosed, err)
	}
	return &admissionRuntime{deps: d, id: hex.EncodeToString(id[:]), ownerCtx: ctx, state: runtimeClosed, changed: make(chan struct{}), work: make(map[*runtimeWorkLease]bool), causes: make(map[*appendToken]CloseCause), operations: make(map[*runtimeOperation]bool)}, nil
}

func missingRuntimePort(port any) bool {
	if port == nil {
		return true
	}
	v := reflect.ValueOf(port)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Func, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}

type admissionRuntime struct {
	mu                      sync.Mutex
	deps                    RuntimeDependencies
	id, state               string
	generation              int64
	starting, started       bool
	ownerCtx, generationCtx context.Context
	shutdownCtx             context.Context
	shutdownMu              sync.Mutex
	cancelGeneration        context.CancelFunc
	ownerWatch              *runtimeOwnerWatch
	exclusive               ExclusiveLease
	proof                   admissionProof
	phase                   *phaseGrant
	work                    map[*runtimeWorkLease]bool
	causes                  map[*appendToken]CloseCause
	changed                 chan struct{}
	drainDone               chan struct{}
	drainErr                error
	cleanupErr              error
	normalDone              chan struct{}
	operations              map[*runtimeOperation]bool
	observationVersion      uint64
	observationCleanupErr   error
	current                 RunBinding
	lifecycleActive         bool
}

// runtimeOwnerWatch is joined outside the runtime lock on every completion path.
type runtimeOwnerWatch struct {
	stop func() bool
	done chan struct{}
	once sync.Once
}

func (w *runtimeOwnerWatch) join() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		if w.stop() {
			close(w.done)
		}
		<-w.done
	})
}

// The owner cannot bypass shutdown publication by canceling the generation directly.
// Values and deadline remain those of the owner; cancellation is runtime ordered.
type runtimeGenerationContext struct {
	context.Context
	owner  context.Context
	cancel context.CancelFunc
}

func (c runtimeGenerationContext) Deadline() (time.Time, bool) { return c.owner.Deadline() }
func (c runtimeGenerationContext) Err() error {
	if c.Context.Err() == nil && c.owner.Err() != nil {
		c.cancel()
	}
	if c.Context.Err() != nil {
		return context.Cause(c.Context)
	}
	return nil
}

func (r *admissionRuntime) signalLocked() { close(r.changed); r.changed = make(chan struct{}) }

func (r *admissionRuntime) beginNormal(ctx context.Context) error {
	return r.beginAction(ctx, true)
}

func (r *admissionRuntime) beginAction(ctx context.Context, lifecycleActive bool) error {
	if err := checkedContext(ctx); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.starting || r.started || r.state != runtimeClosed || len(r.operations) != 0 {
		return ErrAdmissionBusy
	}
	r.starting = true
	r.started = true
	r.lifecycleActive = lifecycleActive
	r.state = runtimeChecking
	r.generation++
	r.normalDone = make(chan struct{})
	r.ownerCtx = ctx
	generation, cancel := context.WithCancelCause(context.WithoutCancel(ctx))
	r.generationCtx = runtimeGenerationContext{Context: generation, owner: ctx, cancel: func() {
		r.selectShutdown(ctx)
		cancel(ctx.Err())
	}}
	r.cancelGeneration = func() { cancel(ctx.Err()) }
	r.ownerWatch = &runtimeOwnerWatch{done: make(chan struct{})}
	watch := r.ownerWatch
	watch.stop = context.AfterFunc(ctx, func() {
		defer close(watch.done)
		r.fail(ctx.Err())
	})
	r.signalLocked()
	return nil
}

func (r *admissionRuntime) RunNormal(ctx context.Context) error {
	if missingRuntimePort(r.deps.Lifecycle) {
		return ErrAdmissionBinding
	}
	if err := r.beginNormal(ctx); err != nil {
		return err
	}
	defer close(r.normalDone)
	ctx = r.generationCtx
	lease, err := r.deps.Exclusive.Acquire(ctx)
	if err != nil {
		r.fail(err)
		return errors.Join(ErrAdmissionClosed, err)
	}
	r.mu.Lock()
	r.exclusive = lease
	r.mu.Unlock()
	p, err := readProof(ctx, r.deps, lease)
	if err != nil {
		r.fail(err)
		return err
	}
	permit, err := r.startupPhase(p)
	if err != nil {
		r.fail(err)
		return err
	}
	if err := r.deps.Lifecycle.ReconcileAndStart(ctx, permit); err != nil {
		r.fail(err)
		return err
	}
	return r.activate(ctx)
}

func (r *admissionRuntime) startupPhase(p admissionProof) (PhasePermit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state != runtimeChecking || r.generationCtx.Err() != nil {
		return PhasePermit{}, ErrAdmissionClosed
	}
	r.proof = p
	r.lifecycleActive = true
	return r.phaseLocked(PhaseStartupReconcile), nil
}

func (r *admissionRuntime) phaseLocked(purpose string) PhasePermit {
	if r.phase != nil {
		r.phase.cancel()
	}
	r.state = purpose
	ctx, cancel := context.WithCancel(r.generationCtx)
	r.phase = &phaseGrant{owner: r, generation: r.generation, ctx: ctx, cancel: cancel, binding: r.bindingLocked(purpose)}
	r.signalLocked()
	return PhasePermit{grant: r.phase}
}

func (r *admissionRuntime) activate(ctx context.Context) error {
	r.mu.Lock()
	p, lease, generation := r.proof, r.exclusive, r.generation
	r.mu.Unlock()
	if err := recheckProof(ctx, r.deps, lease, p); err != nil {
		r.fail(err)
		return err
	}
	r.mu.Lock()
	if r.state != PhaseStartupReconcile || r.generation != generation || len(r.causes) != 0 || r.generationCtx.Err() != nil {
		r.mu.Unlock()
		r.fail(ErrAdmissionClosed)
		return ErrAdmissionClosed
	}
	if r.phase != nil {
		r.phase.cancel()
		r.phase = nil
	}
	r.starting = false
	r.state = runtimePublicReady
	r.signalLocked()
	r.mu.Unlock()
	return nil
}

type runtimeOperation struct {
	ctx            context.Context
	cancel         context.CancelFunc
	stopGeneration func() bool
}

func (r *admissionRuntime) beginObservation(ctx context.Context) (*runtimeOperation, ExclusiveLease, error) {
	if err := checkedContext(ctx); err != nil {
		return nil, nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.starting {
		return nil, nil, ErrAdmissionBusy
	}
	if r.state != runtimeClosed && r.state != runtimePublicReady {
		return nil, nil, ErrAdmissionClosed
	}
	c, cancel := context.WithCancel(ctx)
	op := &runtimeOperation{ctx: c, cancel: cancel}
	if r.generationCtx != nil {
		op.stopGeneration = context.AfterFunc(r.generationCtx, cancel)
	}
	r.operations[op] = true
	return op, r.exclusive, nil
}

func (r *admissionRuntime) endObservation(op *runtimeOperation) {
	op.cancel()
	if op.stopGeneration != nil {
		op.stopGeneration()
	}
	r.mu.Lock()
	delete(r.operations, op)
	r.signalLocked()
	r.mu.Unlock()
}

func (r *admissionRuntime) Verify(ctx context.Context) (observation AdmissionObservation, result error) {
	op, lease, err := r.beginObservation(ctx)
	if err != nil {
		return closedObservation(err)
	}
	defer r.endObservation(op)
	temporary := missingRuntimePort(lease)
	if temporary {
		lease, err = r.deps.Exclusive.Acquire(op.ctx)
		if !missingRuntimePort(lease) {
			defer func() {
				result = errors.Join(result, r.closeObservationLease(lease))
				if result != nil {
					observation = AdmissionObservation{State: runtimeClosed, Reason: "proof_unavailable"}
				}
			}()
		}
		if err != nil {
			return closedObservation(err)
		}
		_, err = readProof(op.ctx, r.deps, lease)
	} else {
		err = r.verifyLiveProof(op.ctx, lease)
	}
	if err != nil {
		if !temporary {
			r.fail(err)
		}
		return closedObservation(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if op.ctx.Err() != nil || r.state != runtimeClosed && (r.state != runtimePublicReady || len(r.causes) != 0 || r.generationCtx.Err() != nil) {
		return closedObservation(ErrAdmissionClosed)
	}
	state := "internal_ready"
	if !temporary {
		state = runtimePublicReady
	}
	return AdmissionObservation{State: state, Reason: "verified"}, nil
}

func (r *admissionRuntime) closeObservationLease(lease ExclusiveLease) error {
	err := lease.Close()
	if err == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observationCleanupErr = errors.Join(r.observationCleanupErr, err)
	r.revokeLocked(err)
	return err
}

func (r *admissionRuntime) verifyLiveProof(ctx context.Context, lease ExclusiveLease) error {
	for attempt := 0; attempt < admissionProofAttempts; attempt++ {
		r.mu.Lock()
		expected, version := r.proof, r.observationVersion
		r.mu.Unlock()
		p, err := readProofAttempt(ctx, r.deps, lease)
		r.mu.Lock()
		current, known := r.proof, r.observationVersion != version
		live := r.state == runtimePublicReady && len(r.causes) == 0
		r.mu.Unlock()
		if e := checkedContext(ctx); e != nil {
			return e
		}
		if !live {
			return ErrAdmissionClosed
		}
		if err == nil {
			if p.store != current.store || !sameJournalObservation(p.recovery, current.recovery) {
				return ErrAdmissionDrift
			}
			if sameJournalObservation(p.article, current.article) {
				return nil
			}
			err = ErrAdmissionDrift
		}
		if !errors.Is(err, ErrAdmissionDrift) || !known || expected.store != current.store || !sameJournalObservation(expected.recovery, current.recovery) {
			return err
		}
	}
	return ErrAdmissionDrift
}

func closedObservation(err error) (AdmissionObservation, error) {
	return AdmissionObservation{State: runtimeClosed, Reason: "proof_unavailable"}, err
}

func validWorkKind(k WorkKind) bool {
	return k == WorkRequest || k == WorkArticleDeletion || k == WorkPasswordReset || k == WorkArticleNotice
}

func (r *admissionRuntime) Acquire(ctx context.Context, kind WorkKind) (WorkLease, error) {
	if err := checkedContext(ctx); err != nil {
		return nil, err
	}
	if !validWorkKind(kind) {
		return nil, ErrAdmissionBinding
	}
	for {
		r.mu.Lock()
		ready := r.state == runtimePublicReady && len(r.causes) == 0
		var changed <-chan struct{} = r.changed
		state := r.state
		r.mu.Unlock()
		if ready {
			lease, paused, err := r.acquireReady(ctx, kind)
			if paused == nil {
				return lease, err
			}
			changed = paused
		}
		if !ready && (kind == WorkRequest || state == runtimeFaulted || state == runtimeDraining || state == runtimeStopped) {
			return nil, ErrAdmissionClosed
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

type runtimeWorkLease struct {
	owner          *admissionRuntime
	ctx            context.Context
	cancel         context.CancelFunc
	stopGeneration func() bool
	generation     int64
	kind           WorkKind
	once           sync.Once
}

type runtimeContextKey struct{}
type runtimeAuthorization struct {
	phase *phaseGrant
	work  *runtimeWorkLease
}

// acquireReady reports only a healthy ordinary pause as a retry of Acquire's wait loop.
func (r *admissionRuntime) acquireReady(ctx context.Context, kind WorkKind) (WorkLease, <-chan struct{}, error) {
	if err := r.checkOwnership(ctx); err != nil {
		return nil, nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state != runtimePublicReady || r.generationCtx.Err() != nil {
		return nil, nil, ErrAdmissionClosed
	}
	if len(r.causes) != 0 {
		if kind != WorkRequest {
			return nil, r.changed, nil
		}
		return nil, nil, ErrAdmissionClosed
	}
	if err := checkedContext(ctx); err != nil {
		return nil, nil, err
	}
	workCtx, cancel := context.WithCancel(ctx)
	l := &runtimeWorkLease{owner: r, cancel: cancel, generation: r.generation, kind: kind}
	l.ctx = context.WithValue(workCtx, runtimeContextKey{}, runtimeAuthorization{work: l})
	l.stopGeneration = context.AfterFunc(r.generationCtx, cancel)
	r.work[l] = true
	return l, nil, nil
}

func (l *runtimeWorkLease) Context() context.Context { return l.ctx }
func (l *runtimeWorkLease) Release() {
	l.once.Do(func() {
		l.owner.mu.Lock()
		delete(l.owner.work, l)
		l.owner.signalLocked()
		l.owner.mu.Unlock()
		l.stopGeneration()
		l.cancel()
	})
}

func (r *admissionRuntime) checkOwnership(ctx context.Context) error {
	r.mu.Lock()
	lease := r.exclusive
	generationCtx := r.generationCtx
	r.mu.Unlock()
	if generationCtx == nil {
		return ErrAdmissionClosed
	}
	if err := generationCtx.Err(); err != nil {
		r.fail(err)
		return err
	}
	if err := checkExclusive(ctx, lease); err != nil {
		r.fail(err)
		return err
	}
	return nil
}

func (r *admissionRuntime) bindingLocked(purpose string) PermitBinding {
	b := r.proof.last
	if r.current.RunID != "" {
		b = r.current
	}
	return PermitBinding{RuntimeID: r.id, Generation: r.generation, RunID: b.RunID, BackupRef: b.BackupRef, InstallationID: r.proof.installation.InstallationID, Purpose: purpose}
}

func (r *admissionRuntime) validPhaseLocked(p PhasePermit, purpose string) bool {
	g := p.grant
	return g != nil && g == r.phase && g.owner == r && g.generation == r.generation && g.binding == r.bindingLocked(purpose) && g.ctx.Err() == nil && r.state == purpose
}

func (r *admissionRuntime) CheckPhase(ctx context.Context, p PhasePermit, purpose string) error {
	if err := checkedContext(ctx); err != nil {
		return err
	}
	if err := r.checkOwnership(ctx); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.validPhaseLocked(p, purpose) {
		return ErrAdmissionBinding
	}
	return nil
}

func (r *admissionRuntime) PhaseContext(ctx context.Context, p PhasePermit) (context.Context, error) {
	if p.grant == nil {
		return nil, ErrAdmissionBinding
	}
	if err := r.CheckPhase(ctx, p, p.grant.binding.Purpose); err != nil {
		return nil, err
	}
	c, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(p.grant.ctx, cancel)
	context.AfterFunc(c, func() { stop() })
	return context.WithValue(c, runtimeContextKey{}, runtimeAuthorization{phase: p.grant}), nil
}

func (r *admissionRuntime) contextBindingLocked(ctx context.Context) (PermitBinding, error) {
	if r.generationCtx == nil || r.generationCtx.Err() != nil {
		return PermitBinding{}, ErrAdmissionClosed
	}
	a, ok := ctx.Value(runtimeContextKey{}).(runtimeAuthorization)
	if !ok {
		return PermitBinding{}, ErrAdmissionBinding
	}
	if a.phase != nil {
		if !r.validPhaseLocked(PhasePermit{grant: a.phase}, a.phase.binding.Purpose) {
			return PermitBinding{}, ErrAdmissionBinding
		}
		return a.phase.binding, nil
	}
	l := a.work
	if l == nil || l.owner != r || l.generation != r.generation || !r.work[l] || l.ctx.Err() != nil || r.state != runtimePublicReady {
		return PermitBinding{}, ErrAdmissionBinding
	}
	return r.bindingLocked(string(l.kind)), nil
}

func (r *admissionRuntime) ReadPermitBinding(ctx context.Context) (PermitBinding, error) {
	if err := checkedContext(ctx); err != nil {
		return PermitBinding{}, err
	}
	if err := r.checkOwnership(ctx); err != nil {
		return PermitBinding{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.contextBindingLocked(ctx)
}

func (r *admissionRuntime) CheckJournalUse(ctx context.Context, operation, use string) error {
	if !admissionText(operation) {
		return ErrAdmissionBinding
	}
	b, err := r.ReadPermitBinding(ctx)
	if err != nil {
		return err
	}
	if b.Purpose == PhaseRecoveryRestore || b.Purpose == PhaseRecoveryBind {
		return ErrAdmissionBinding
	}
	switch use {
	case JournalUseCommittedRead:
		return nil
	case JournalUseNewAppend:
		if b.Purpose == PhaseStartupReconcile || validWorkKind(WorkKind(b.Purpose)) {
			return nil
		}
	case JournalUseCoordinate:
		if b.Purpose == JournalUseCoordinate || b.Purpose == phaseInstall {
			return nil
		}
	}
	return ErrAdmissionBinding
}

func (r *admissionRuntime) PauseArticleAppend(ctx context.Context, operation string) (AppendBarrier, error) {
	if err := r.CheckJournalUse(ctx, operation, JournalUseNewAppend); err != nil {
		return AppendBarrier{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.contextBindingLocked(ctx); err != nil {
		return AppendBarrier{}, err
	}
	for _, c := range r.causes {
		if c.OperationID == operation {
			return AppendBarrier{}, ErrAdmissionBusy
		}
	}
	t := &appendToken{owner: r, generation: r.generation, operation: operation, head: r.proof.article.Head, revision: r.proof.article.Revision}
	r.causes[t] = CloseCause{Code: causeArticleAppend, OperationID: operation}
	r.signalLocked()
	return AppendBarrier{token: t}, nil
}

func (r *admissionRuntime) validBarrierLocked(b AppendBarrier) bool {
	if b.token == nil || b.token.owner != r || b.token.generation != r.generation {
		return false
	}
	c, ok := r.causes[b.token]
	return ok && c.Code == causeArticleAppend && c.OperationID == b.token.operation
}

func (r *admissionRuntime) CompleteArticleAppend(ctx context.Context, b AppendBarrier, a VerifiedHeadAdvance) error {
	if err := checkedContext(ctx); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.validBarrierLocked(b) {
		return ErrAdmissionBinding
	}
	if _, err := r.contextBindingLocked(ctx); err != nil {
		return err
	}
	if !r.validAdvanceLocked(b.token, a) {
		r.revokeLocked(ErrAdmissionDrift)
		return ErrAdmissionDrift
	}
	r.proof.article.Head = a.NewHead
	r.proof.article.Revision = a.CommittedRevision
	r.proof.article.Prefixes = append(append([]JournalHead(nil), r.proof.article.Prefixes...), a.NewHead)
	r.observationVersion++
	delete(r.causes, b.token)
	r.signalLocked()
	return nil
}

func (r *admissionRuntime) validAdvanceLocked(t *appendToken, a VerifiedHeadAdvance) bool {
	s := r.proof.article
	return a.Namespace == ArticleDeletionNamespaceV1 && a.JournalID == s.JournalID && a.Epoch == s.Epoch &&
		a.OperationID == t.operation && a.OldHead == t.head && a.OldHead == s.Head && a.OldRevision == t.revision && a.OldRevision == s.Revision &&
		a.OldHead.Sequence < math.MaxInt64 && a.NewHead.Sequence == a.OldHead.Sequence+1 && admissionHead(a.NewHead) &&
		admissionText(a.PendingRevision) && admissionText(a.CommittedRevision) && a.PendingRevision != a.OldRevision &&
		a.CommittedRevision != a.OldRevision && a.CommittedRevision != a.PendingRevision
}

func (r *admissionRuntime) FaultArticleAppend(ctx context.Context, b AppendBarrier, cause error) error {
	if ctx == nil {
		return ErrAdmissionBinding
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.validBarrierLocked(b) {
		return ErrAdmissionBinding
	}
	if cause == nil {
		cause = ErrRecoveryWriteUnknown
	}
	r.causes[b.token] = CloseCause{Code: causeArticleAppendFault, OperationID: b.token.operation}
	r.revokeLocked(cause)
	return errors.Join(ErrRecoveryWriteUnknown, cause)
}

func (r *admissionRuntime) fail(cause error) { r.mu.Lock(); defer r.mu.Unlock(); r.revokeLocked(cause) }

func (r *admissionRuntime) revokeLocked(cause error) {
	if r.state == runtimeStopped || r.state == runtimeFaulted || r.state == runtimeDraining {
		return
	}
	r.selectShutdown(r.ownerCtx)
	r.state = runtimeFaulted
	r.starting = false
	if r.generation < math.MaxInt64 {
		r.generation++
	}
	if r.phase != nil {
		r.phase.cancel()
		r.phase = nil
	}
	if r.cancelGeneration != nil {
		r.cancelGeneration()
	}
	for w := range r.work {
		w.cancel()
	}
	for op := range r.operations {
		op.cancel()
	}
	r.drainErr = cause
	r.signalLocked()
	r.startDrainLocked()
}

func (r *admissionRuntime) startDrainLocked() {
	if r.drainDone != nil {
		return
	}
	r.drainDone = make(chan struct{})
	go r.drain()
}

func (r *admissionRuntime) drain() {
	r.mu.Lock()
	started, normalDone, watch := r.lifecycleActive, r.normalDone, r.ownerWatch
	r.mu.Unlock()
	ctx, _ := r.ShutdownContext()
	watch.join()
	var err error
	if started {
		err = r.deps.Lifecycle.StopAndJoin(ctx)
	}
	if normalDone != nil {
		<-normalDone
	}
	for {
		r.mu.Lock()
		remaining, changed := len(r.work)+len(r.operations), r.changed
		r.mu.Unlock()
		if remaining == 0 {
			break
		}
		<-changed
	}
	r.mu.Lock()
	lease := r.exclusive
	r.mu.Unlock()
	err = errors.Join(err, r.observationCleanupErr)
	// An adapter unable to prove join must not release deployment ownership.
	if err == nil && lease != nil {
		err = lease.Close()
	}
	r.mu.Lock()
	r.cleanupErr = err
	r.drainErr = errors.Join(r.drainErr, err)
	r.state = runtimeStopped
	r.signalLocked()
	close(r.drainDone)
	r.mu.Unlock()
}

// selectShutdown can also publish during a synchronous cancellation check while
// admission is locked. It never calls an external port or cancels a context.
func (r *admissionRuntime) selectShutdown(ctx context.Context) {
	r.shutdownMu.Lock()
	defer r.shutdownMu.Unlock()
	if r.shutdownCtx == nil {
		r.shutdownCtx = ctx
	}
}

func (r *admissionRuntime) ShutdownContext() (context.Context, bool) {
	r.shutdownMu.Lock()
	defer r.shutdownMu.Unlock()
	return r.shutdownCtx, r.shutdownCtx != nil
}

func (r *admissionRuntime) Stop(ctx context.Context) error {
	if ctx == nil {
		return ErrAdmissionBinding
	}
	r.mu.Lock()
	if r.drainDone == nil {
		r.selectShutdown(ctx)
		r.state = runtimeDraining
		r.starting = false
		if r.generation < math.MaxInt64 {
			r.generation++
		}
		if r.phase != nil {
			r.phase.cancel()
			r.phase = nil
		}
		if r.cancelGeneration != nil {
			r.cancelGeneration()
		}
		for w := range r.work {
			w.cancel()
		}
		for op := range r.operations {
			op.cancel()
		}
		r.signalLocked()
		r.startDrainLocked()
	}
	done := r.drainDone
	r.mu.Unlock()
	<-done
	r.mu.Lock()
	defer r.mu.Unlock()
	// Fault causes are returned by the operation that raised them; Stop reports cleanup.
	return errors.Join(ctx.Err(), r.cleanupErr)
}

type recoveryFlow struct {
	r       *admissionRuntime
	binding RunBinding
	plan    RestorePlan
	receipt RestoreReceipt
	view    controlledView
	others  map[string]string
}

func (r *admissionRuntime) recoveryPreflight(ctx context.Context, b RunBinding, resume bool) error {
	if e := checkedContext(ctx); e != nil {
		return e
	}
	if !admissionText(b.RunID) || !admissionText(b.BackupRef) {
		return ErrAdmissionBinding
	}
	ports := []any{r.deps.Runs, r.deps.Identity, r.deps.Restore, r.deps.Consumer, r.deps.Exit, r.deps.Lifecycle}
	if resume {
		ports = append(ports, r.deps.RecoverySecret)
	}
	for _, p := range ports {
		if missingRuntimePort(p) {
			return ErrAdmissionBinding
		}
	}
	return nil
}

// Enter is the controlled, closed/drained/exclusive producer of recovery permits.
func (r *admissionRuntime) Enter(ctx context.Context, in RecoveryInput) error {
	defer clear(in.Secret)
	if e := r.recoveryPreflight(ctx, in.RunBinding, false); e != nil {
		return e
	}
	if len(in.Secret) == 0 {
		return ErrAdmissionBinding
	}
	if e := r.beginAction(ctx, false); e != nil {
		return e
	}
	defer close(r.normalDone)
	f := &recoveryFlow{r: r, binding: in.RunBinding}
	e := f.enter(r.generationCtx, in.Secret)
	if e != nil {
		r.fail(e)
	}
	return e
}

// Resume continues only independently verified persistent progress, never Begin/Replace.
func (r *admissionRuntime) Resume(ctx context.Context, b RunBinding) error {
	if e := r.recoveryPreflight(ctx, b, true); e != nil {
		return e
	}
	if e := r.beginAction(ctx, false); e != nil {
		return e
	}
	defer close(r.normalDone)
	f := &recoveryFlow{r: r, binding: b}
	e := f.resume(r.generationCtx)
	if e != nil {
		r.fail(e)
	}
	return e
}

func (f *recoveryFlow) acquire(ctx context.Context) error {
	// Even a cold root must actually confirm its ordinary owners have stopped.
	if e := f.r.deps.Lifecycle.StopAndJoin(ctx); e != nil {
		return e
	}
	if e := checkedContext(ctx); e != nil {
		return e
	}
	l, e := f.r.deps.Exclusive.Acquire(ctx)
	f.r.mu.Lock()
	f.r.exclusive = l
	f.r.current = f.binding
	f.r.mu.Unlock()
	if e != nil {
		return e
	}
	return checkExclusive(ctx, l)
}

func (f *recoveryFlow) enter(ctx context.Context, secret []byte) error {
	if e := f.acquire(ctx); e != nil {
		return e
	}
	if e := f.begin(ctx, secret); e != nil {
		return e
	}
	p, pc, e := f.phase(ctx, PhaseRecoveryRestore)
	if e != nil {
		return e
	}
	f.receipt, e = f.r.deps.Restore.Replace(pc, p, f.plan)
	if e != nil {
		return e
	}
	if e := f.refreshAfter(ctx, nil, storeRestored); e != nil {
		return e
	}
	if e := f.verifyReplacement(ctx); e != nil {
		return e
	}
	if e := f.bind(ctx); e != nil {
		return e
	}
	return f.identity(ctx, secret)
}

func (f *recoveryFlow) begin(ctx context.Context, secret []byte) error {
	initial, e := readProof(ctx, f.r.deps, f.r.exclusive)
	if e != nil {
		return e
	}
	initial.recovery = copyControlledJournal(initial.recovery)
	initial.article = copyControlledJournal(initial.article)
	f.plan, e = f.r.deps.Restore.Prepare(ctx, f.binding)
	if e != nil {
		return e
	}
	if e := validateRestorePlan(f.plan, initial.installation, f.binding); e != nil {
		return e
	}
	if e := recheckProof(ctx, f.r.deps, f.r.exclusive, initial); e != nil {
		return e
	}
	entry, e := f.r.deps.Journal.Begin(ctx, f.binding.RunID, f.binding.BackupRef, secret)
	if e != nil {
		return e
	}
	v, e := f.r.readControlled(ctx, f.binding)
	if e != nil {
		return e
	}
	if v.run != nil || !sameBegin(entry, v.begin) || v.proof.store != initial.store || v.proof.installation != initial.installation ||
		!sameControlledJournal(initial.article, v.proof.article) || !exactJournalAppend(initial.recovery, v.proof.recovery, entry) {
		return ErrAdmissionBinding
	}
	f.seed(v)
	return nil
}

func (f *recoveryFlow) verifyReplacement(ctx context.Context) error {
	verified, e := f.r.deps.Restore.VerifyRestored(ctx, f.plan)
	if e != nil {
		return e
	}
	if e := validatePlanReceipt(f.plan, verified); e != nil {
		return e
	}
	if f.receipt != verified {
		return ErrAdmissionBinding
	}
	return nil
}

func (f *recoveryFlow) resume(ctx context.Context) error {
	if e := f.acquire(ctx); e != nil {
		return e
	}
	v, e := f.r.readControlled(ctx, f.binding)
	if e != nil {
		return e
	}
	f.seed(v)
	f.plan, e = f.r.deps.Restore.Prepare(ctx, f.binding)
	if e != nil {
		return e
	}
	if e := validateRestorePlan(f.plan, v.proof.installation, f.binding); e != nil {
		return e
	}
	if e := f.refresh(ctx); e != nil {
		return e
	}
	f.receipt, e = f.r.deps.Restore.VerifyRestored(ctx, f.plan)
	if e != nil {
		return e
	}
	if e := validatePlanReceipt(f.plan, f.receipt); e != nil {
		return e
	}
	if v.run == nil {
		if e := f.bind(ctx); e != nil {
			return e
		}
	} else if e := f.refresh(ctx); e != nil {
		return e
	}
	if f.view.run.Outcome == RecoveryOutcomeRunning {
		return f.identity(ctx, nil)
	}
	return f.replay(ctx)
}

func (f *recoveryFlow) seed(v controlledView) {
	f.view = v
	f.others = make(map[string]string)
	for _, x := range v.snapshot.Runs {
		if x.RunID != f.binding.RunID {
			f.others[x.RunID] = string(encodeAdmissionRun(x))
		}
	}
}

type controlledStoreEffect uint8

const (
	storeUnchanged controlledStoreEffect = iota
	storeRestored
	storeWritten
)

func effectForConsumption(fresh *ConsumptionResult) controlledStoreEffect {
	if fresh != nil {
		return storeWritten
	}
	return storeUnchanged
}

func (f *recoveryFlow) acceptStoreObservation(v controlledView, effect controlledStoreEffect) bool {
	before, after := f.view.proof.store, v.proof.store
	if f.view.run != nil && v.run == nil {
		return false
	}
	if effect == storeUnchanged {
		return before == after
	}
	// A verified replacement may contain an older target counter; a transaction
	// on the bound target must preserve or advance its independent revision.
	if effect == storeRestored {
		return f.view.run == nil && v.run == nil
	}
	return before == after || after.Counter > before.Counter
}

func (f *recoveryFlow) refresh(ctx context.Context) error {
	return f.refreshAfter(ctx, nil, storeUnchanged)
}

// Only the surrounding owned write may explain a Store change. Only a fresh
// Consume result may explain the exact consumed append; all other reads are stable.
func (f *recoveryFlow) refreshAfter(ctx context.Context, consumed *ConsumptionResult, storeEffect controlledStoreEffect) error {
	v, e := f.r.readControlled(ctx, f.binding)
	if e != nil {
		return e
	}
	if v.proof.installation != f.view.proof.installation || !sameBegin(v.begin, f.view.begin) {
		return ErrAdmissionDrift
	}
	if !sameControlledJournal(v.proof.article, f.view.proof.article) || !f.acceptRecoveryObservation(v, consumed) ||
		!f.acceptStoreObservation(v, storeEffect) {
		return ErrAdmissionDrift
	}
	count := 0
	for _, x := range v.snapshot.Runs {
		if x.RunID != f.binding.RunID {
			count++
			if f.others[x.RunID] != string(encodeAdmissionRun(x)) {
				return ErrAdmissionDrift
			}
		}
	}
	if count != len(f.others) {
		return ErrAdmissionDrift
	}
	if v.run != nil && f.receipt.Version != "" && *v.run.RestoreBinding != f.receipt {
		return ErrAdmissionBinding
	}
	f.view = v
	return nil
}

func (f *recoveryFlow) acceptRecoveryObservation(v controlledView, consumed *ConsumptionResult) bool {
	before, after := f.view.proof.recovery, v.proof.recovery
	if sameControlledJournal(before, after) {
		return true
	}
	return consumed != nil && f.view.latest.Stage == StageBegun && v.latest.Stage == StageConsumed &&
		consumed.Journal == v.latest && exactJournalAppend(before, after, consumed.Journal)
}

func (f *recoveryFlow) phase(ctx context.Context, purpose string) (PhasePermit, context.Context, error) {
	if e := f.refresh(ctx); e != nil {
		return PhasePermit{}, nil, e
	}
	if purpose == PhaseRecoveryBind && (f.view.run != nil || f.view.latest.Stage != StageBegun || f.receipt.Version == "") {
		return PhasePermit{}, nil, ErrAdmissionBinding
	}
	r := f.r
	r.mu.Lock()
	if r.generationCtx.Err() != nil || r.state == runtimeFaulted || r.state == runtimeDraining || r.state == runtimeStopped {
		r.mu.Unlock()
		return PhasePermit{}, nil, ErrAdmissionClosed
	}
	r.proof = f.view.proof
	p := r.phaseLocked(purpose)
	r.mu.Unlock()
	pc, e := r.PhaseContext(ctx, p)
	return p, pc, e
}

func (f *recoveryFlow) bind(ctx context.Context) error {
	p, pc, e := f.phase(ctx, PhaseRecoveryBind)
	if e != nil {
		return e
	}
	_, e = f.r.deps.Identity.BindRestoredRun(pc, p, f.binding, f.receipt)
	if e != nil {
		return e
	}
	if e := f.refreshAfter(ctx, nil, storeWritten); e != nil {
		return e
	}
	if f.view.run == nil || f.view.run.Outcome != RecoveryOutcomeRunning {
		return ErrAdmissionBinding
	}
	return nil
}

func (f *recoveryFlow) identity(ctx context.Context, enterSecret []byte) error {
	p, pc, e := f.phase(ctx, PhaseRecoveryIdentity)
	if e != nil {
		return e
	}
	if f.view.run == nil || f.view.run.Outcome != RecoveryOutcomeRunning {
		return ErrAdmissionBinding
	}
	if e := f.r.deps.Identity.FinalizeRestoredIdentity(pc, p, f.binding); e != nil {
		return e
	}
	if e := f.refreshAfter(pc, nil, storeWritten); e != nil {
		return e
	}
	if f.view.run.AuditEventID == nil {
		return ErrAdmissionIncomplete
	}
	if f.view.run.InitialAdminBootstrapGeneration == nil {
		if e := f.prepareTrust(pc, p, enterSecret); e != nil {
			return e
		}
	}
	if f.view.run.InitialAdminBootstrapGeneration == nil || f.view.run.InitialAdminUserID == nil {
		return ErrAdmissionIncomplete
	}
	if e := f.r.CheckPhase(pc, p, PhaseRecoveryIdentity); e != nil {
		return e
	}
	if e := f.r.deps.Identity.CompleteIdentityReview(pc, p, f.binding); e != nil {
		return e
	}
	if e := f.refreshAfter(pc, nil, storeWritten); e != nil {
		return e
	}
	if f.view.run.Outcome != RecoveryOutcomeIdentityReady {
		return ErrAdmissionBinding
	}
	if !changedRecoveryTrust(*f.view.run) {
		return ErrAdmissionIncomplete
	}
	return f.replay(ctx)
}

func (f *recoveryFlow) prepareTrust(ctx context.Context, p PhasePermit, secret []byte) error {
	if secret == nil {
		var e error
		secret, e = f.r.deps.RecoverySecret.ReadInitialRecoverySecret(ctx, f.binding)
		defer clear(secret)
		if e != nil {
			return e
		}
		if len(secret) == 0 {
			return ErrAdmissionBinding
		}
		if e := f.refresh(ctx); e != nil {
			return e
		}
		before := f.view.proof
		entry, e := f.r.deps.Journal.VerifyBeginSecret(ctx, f.binding, secret)
		if e != nil {
			return e
		}
		if !sameBegin(entry, f.view.begin) {
			return ErrAdmissionBinding
		}
		if e := f.refresh(ctx); e != nil {
			return e
		}
		if before.store != f.view.proof.store || !sameJournalObservation(before.recovery, f.view.proof.recovery) || !sameJournalObservation(before.article, f.view.proof.article) {
			return ErrAdmissionDrift
		}
	}
	if f.view.run == nil || f.view.run.InitialAdminBootstrapGeneration != nil || f.view.run.Outcome != RecoveryOutcomeRunning {
		return ErrAdmissionBinding
	}
	if e := f.r.CheckPhase(ctx, p, PhaseRecoveryIdentity); e != nil {
		return e
	}
	if e := f.r.deps.Identity.PrepareRecoveryTrust(ctx, p, f.binding, secret); e != nil {
		return e
	}
	return f.refreshAfter(ctx, nil, storeWritten)
}

func (f *recoveryFlow) replay(ctx context.Context) error {
	p, pc, e := f.phase(ctx, PhaseRecoveryReplay)
	if e != nil {
		return e
	}
	var fresh *ConsumptionResult
	if f.view.run == nil {
		return ErrAdmissionIncomplete
	}
	if f.view.run.Outcome != RecoveryOutcomeSucceeded {
		result, e := f.r.deps.Consumer.Consume(pc, f.binding.RunID)
		if e != nil {
			return e
		}
		fresh = &result
	}
	if e := f.confirmSucceeded(pc, fresh); e != nil {
		return e
	}
	if e := f.r.CheckPhase(pc, p, PhaseRecoveryReplay); e != nil {
		return e
	}
	if e := f.r.deps.Exit.ExitRecovery(pc, p, f.binding); e != nil {
		return e
	}
	if e := f.refreshAfter(pc, nil, storeWritten); e != nil {
		return e
	}
	if e := f.confirmSucceeded(pc, nil); e != nil {
		return e
	}
	r := f.r
	r.mu.Lock()
	if r.phase != p.grant || r.generationCtx.Err() != nil {
		r.mu.Unlock()
		return ErrAdmissionClosed
	}
	r.phase.cancel()
	r.phase = nil
	r.state = runtimeChecking
	r.mu.Unlock()
	startup, e := r.startupPhase(f.view.proof)
	if e != nil {
		return e
	}
	if e := r.deps.Lifecycle.ReconcileAndStart(ctx, startup); e != nil {
		return e
	}
	return r.activate(ctx)
}

func (f *recoveryFlow) confirmSucceeded(ctx context.Context, fresh *ConsumptionResult) error {
	if e := f.refreshAfter(ctx, fresh, effectForConsumption(fresh)); e != nil {
		return e
	}
	x := f.view.run
	if x == nil || !completeAdmissionRun(*x) {
		return ErrAdmissionIncomplete
	}
	if e := proveClosure(ctx, f.r.deps.Closures, *x, f.view.proof.article); e != nil {
		return e
	}
	if fresh != nil {
		// Exact immutable publication corroborates, never replaces, the fresh Consume result.
		q := ClosureRequest{RunID: x.RunID, BackupRef: x.BackupRef, EvidenceRef: *x.ArticleEvidenceRef, EvidenceDigest: *x.ArticleEvidenceDigest, PublicationRef: *x.ArticlePublicationRef, PublicationDigest: *x.ArticlePublicationDigest}
		closure, e := f.r.deps.Closures.ReadPublishedClosure(ctx, q)
		if e != nil {
			return e
		}
		if fresh.Evidence != closure.Ready || fresh.Journal != f.view.latest || fresh.Run.RunID != x.RunID || fresh.Run.BackupRef != x.BackupRef || fresh.Run.Outcome != RecoveryOutcomeSucceeded || fresh.Run.ArticlePublicationRef != *x.ArticlePublicationRef || fresh.Run.ArticlePublicationDigest != *x.ArticlePublicationDigest {
			return ErrAdmissionBinding
		}
	}
	return recheckProof(ctx, f.r.deps, f.r.exclusive, f.view.proof)
}

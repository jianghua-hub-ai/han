package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"
)

// InstallationDependencies binds the sole Store and the two namespace owners.
// Media verifies/installs journal metadata under the current install permit;
// Coordinate opens, coordinates and binds the same Article control handle.
// Every callback must synchronously join its I/O, including on error.
type InstallationDependencies struct {
	Commitments       InstallationCommitmentStore
	Sources           InstallationSourceVerifier
	Writer            InstallationRepository
	Store             AdmissionStore
	Anchor            JournalHeadAnchor
	Recovery, Article VerifiedJournalReader
	Media             func(context.Context, PhasePermit, InstallationManifest) error
	Coordinate        func(context.Context, PhasePermit, InstallationManifest) error
	Legacy            func(context.Context, PhasePermit, InstallationManifest) ([]PublicationBinding, error)
}

type installationExecutor struct{ d InstallationDependencies }

// NewInstallationExecutor constructs the closed installation algorithm without I/O.
func NewInstallationExecutor(d InstallationDependencies) (InstallationExecutor, error) {
	for _, port := range []any{d.Commitments, d.Sources, d.Store, d.Anchor, d.Recovery, d.Article, d.Coordinate} {
		if missingRuntimePort(port) {
			return nil, ErrAdmissionBinding
		}
	}
	return &installationExecutor{d: d}, nil
}

func (e *installationExecutor) Install(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	if err := e.guard(ctx, p, m); err != nil {
		return err
	}
	c, revision, fresh, err := e.readInstallation(ctx, p, m)
	if err != nil && !fresh {
		return err
	}
	if err == nil && c.Phase == "bound" {
		return e.d.Coordinate(ctx, p, m)
	}
	if missingRuntimePort(e.d.Writer) || e.d.Media == nil || !sameInstallationPort(e.d.Writer, p.grant.owner.deps.InstallationWriter) {
		return ErrAdmissionBinding
	}
	if fresh {
		c = InstallationCommitment{Manifest: m, ManifestDigest: installationManifestDigest(m), Phase: "prepared"}
		revision, err = e.swap(ctx, p, c, "", m)
		if err != nil {
			return err
		}
	}
	return e.prepare(ctx, p, m, c, revision)
}

func (e *installationExecutor) guard(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	if e == nil || p.grant == nil || p.grant.owner == nil {
		return ErrAdmissionBinding
	}
	if err := installationContext(ctx); err != nil {
		return err
	}
	if err := validateInstallationManifest(m); err != nil {
		return err
	}
	r := p.grant.owner
	if err := r.CheckPhase(ctx, p, phaseInstall); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := ctx.Value(runtimeContextKey{}).(runtimeAuthorization)
	if !ok || a.phase != p.grant || !r.validPhaseLocked(p, phaseInstall) {
		return ErrAdmissionBinding
	}
	b := installationBindingFor(m)
	i := r.proof.installation
	if i.Version != b.Version || i.DeploymentID != b.DeploymentID || i.InstallationID != b.InstallationID || i.TargetDigest != b.TargetDigest || i.CommitmentRef != b.CommitmentRef || i.CommitmentDigest != b.CommitmentDigest || i.RecoveryInitial != b.RecoveryInitial || i.ArticleInitial != b.ArticleInitial {
		return ErrAdmissionBinding
	}
	for _, pair := range [][2]any{{e.d.Store, r.deps.Store}, {e.d.Commitments, r.deps.Commitments}, {e.d.Sources, r.deps.InstallationSources}, {e.d.Anchor, r.deps.ArticleAnchorReader}, {e.d.Recovery, r.deps.Journal}, {e.d.Article, r.deps.ArticleSnapshots}} {
		if !sameInstallationPort(pair[0], pair[1]) {
			return ErrAdmissionBinding
		}
	}
	return nil
}

func sameInstallationPort(a, b any) bool {
	return a != nil && b != nil && reflect.TypeOf(a) == reflect.TypeOf(b) && reflect.TypeOf(a).Comparable() && a == b
}

func (e *installationExecutor) load(ctx context.Context, p PhasePermit, m InstallationManifest) (InstallationCommitment, string, error) {
	c, revision, _, err := e.readInstallation(ctx, p, m)
	return c, revision, err
}

// Only a missing commitment key classifies a fresh installation. Approval
// lookup errors retain their causes without becoming permission to create it.
func (e *installationExecutor) readInstallation(ctx context.Context, p PhasePermit, m InstallationManifest) (InstallationCommitment, string, bool, error) {
	if err := e.guard(ctx, p, m); err != nil {
		return InstallationCommitment{}, "", false, err
	}
	if err := e.d.Sources.VerifyInstallationSource(ctx, m); err != nil {
		return InstallationCommitment{}, "", false, err
	}
	if err := e.guard(ctx, p, m); err != nil {
		return InstallationCommitment{}, "", false, err
	}
	c, v, err := e.d.Commitments.LoadInstallation(ctx, m.DeploymentID)
	if err != nil {
		return InstallationCommitment{}, "", errors.Is(err, ErrAnchorMissing), err
	}
	if c.Manifest != m || c.ManifestDigest != installationManifestDigest(m) || !admissionText(v) || c.Phase != "prepared" && c.Phase != "bound" {
		return InstallationCommitment{}, "", false, ErrAdmissionBinding
	}
	return c, v, false, e.guard(ctx, p, m)
}

func (e *installationExecutor) swap(ctx context.Context, p PhasePermit, c InstallationCommitment, revision string, m InstallationManifest) (string, error) {
	if err := e.guard(ctx, p, m); err != nil {
		return "", err
	}
	written, token, err := e.d.Commitments.CompareAndSwapInstallation(ctx, m.DeploymentID, revision, c)
	if err != nil {
		return "", errors.Join(ErrRecoveryWriteUnknown, err)
	}
	if written != c || !admissionText(token) || token == revision {
		return "", ErrRecoveryWriteUnknown
	}
	actual, v, err := e.load(ctx, p, m)
	if err != nil || actual != c || v != token {
		return "", errors.Join(ErrRecoveryWriteUnknown, ErrAdmissionDrift, err)
	}
	return token, nil
}

func installationBindingFor(m InstallationManifest) InstallationBinding {
	return InstallationBinding{Version: m.Version, DeploymentID: m.DeploymentID, InstallationID: m.InstallationID, TargetDigest: m.TargetDigest,
		CommitmentRef: "recovery/v1/installation/" + m.DeploymentID, CommitmentDigest: installationManifestDigest(m),
		RecoveryInitial: m.Recovery, ArticleInitial: m.Article, InstalledAt: m.IssuedAt}
}

func (e *installationExecutor) prepare(ctx context.Context, p PhasePermit, m InstallationManifest, c InstallationCommitment, revision string) error {
	if err := e.d.Media(ctx, p, m); err != nil {
		return err
	}
	for _, initial := range []AnchorSnapshot{m.Recovery, m.Article} {
		if err := e.preparedCurrent(ctx, p, m, c, revision); err != nil {
			return err
		}
		if err := e.initialAnchor(ctx, p, m, initial); err != nil {
			return fmt.Errorf("install initial %s: %w", initial.Namespace, err)
		}
	}
	if err := e.preparedCurrent(ctx, p, m, c, revision); err != nil {
		return err
	}
	if !sameInstallationPort(e.d.Writer, p.grant.owner.deps.InstallationWriter) {
		return ErrAdmissionBinding
	}
	if err := e.d.Writer.BindInstallation(ctx, p, installationBindingFor(m)); err != nil {
		return err
	}
	if err := e.sealPrepared(ctx, p, m); err != nil {
		return fmt.Errorf("verify prepared origin: %w", err)
	}
	if err := e.d.Coordinate(ctx, p, m); err != nil {
		return err
	}
	if err := e.legacy(ctx, p, m); err != nil {
		return err
	}
	if err := e.verifyPrepared(ctx, p, m); err != nil {
		return err
	}
	actual, token, err := e.load(ctx, p, m)
	if err != nil || actual != c || token != revision {
		return errors.Join(ErrAdmissionDrift, err)
	}
	c.Phase = "bound"
	_, err = e.swap(ctx, p, c, revision, m)
	return err
}

func (e *installationExecutor) preparedCurrent(ctx context.Context, p PhasePermit, m InstallationManifest, c InstallationCommitment, revision string) error {
	actual, token, err := e.load(ctx, p, m)
	if err != nil || actual != c || token != revision {
		return errors.Join(ErrAdmissionDrift, err)
	}
	return nil
}

func (e *installationExecutor) initialAnchor(ctx context.Context, p PhasePermit, m InstallationManifest, initial AnchorSnapshot) error {
	if err := e.guard(ctx, p, m); err != nil {
		return err
	}
	a, v, err := e.d.Anchor.Load(ctx, initial.Namespace)
	if errors.Is(err, ErrAnchorMissing) {
		if err := e.guard(ctx, p, m); err != nil {
			return err
		}
		var written AnchorSnapshot
		var token string
		written, token, err = e.d.Anchor.CompareAndSwap(ctx, initial.Namespace, "", initial)
		if err != nil {
			return errors.Join(ErrRecoveryWriteUnknown, err)
		}
		if written != initial || !admissionText(token) {
			return ErrRecoveryWriteUnknown
		}
		a, v, err = e.d.Anchor.Load(ctx, initial.Namespace)
		if err == nil && (a != written || v != token) {
			return ErrAdmissionDrift
		}
	}
	if err != nil {
		return err
	}
	if a != initial || !admissionText(v) {
		return ErrAdmissionBinding
	}
	return e.guard(ctx, p, m)
}

func (e *installationExecutor) preparedObservation(ctx context.Context, p PhasePermit, m InstallationManifest) (installationObservation, error) {
	var o installationObservation
	if err := e.guard(ctx, p, m); err != nil {
		return o, err
	}
	s, err := e.d.Store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return o, err
	}
	raw, err := CanonicalAdmissionSnapshot(s)
	if err != nil {
		return o, err
	}
	if s.Revision.Digest != SHA256Digest(raw) || s.Installation != installationBindingFor(m) {
		return o, ErrAdmissionBinding
	}
	j, err := e.d.Recovery.ReadVerifiedSnapshot(ctx, RecoveryNamespaceV1)
	if err != nil {
		return o, err
	}
	if err := validateVerifiedJournal(j, m.Recovery); err != nil {
		return o, err
	}
	if j.Head != m.Recovery.Committed {
		return o, ErrAdmissionDrift
	}
	if m.Kind == "genesis" && (len(s.Runs) != 0 || len(j.Entries) != 0) {
		return o, ErrAdmissionBinding
	}
	if err := validateInstallationRuns(s, j); err != nil {
		return o, err
	}
	a, token, err := e.d.Anchor.Load(ctx, ArticleDeletionNamespaceV1)
	if err != nil {
		return o, err
	}
	if a != m.Article || !admissionText(token) {
		return o, ErrAdmissionBinding
	}
	o.installation, o.store, o.storeBytes, o.recovery, o.article, o.articleRevision = s.Installation, s.Revision, string(raw), copyControlledJournal(j), a, token
	o.commitment, o.commitRevision, err = e.load(ctx, p, m)
	return o, err
}

func (e *installationExecutor) sealPrepared(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	a, err := e.preparedObservation(ctx, p, m)
	if err != nil {
		return err
	}
	b, err := e.preparedObservation(ctx, p, m)
	if err != nil || !sameInstallationObservation(a, b) {
		return errors.Join(ErrAdmissionDrift, err)
	}
	r := p.grant.owner
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.validPhaseLocked(p, phaseInstall) || ctx.Err() != nil || p.grant.installation != nil {
		return ErrAdmissionBinding
	}
	// Complete the original fresh/prepared grant with the independently read
	// installed origin. This never creates, renews or transfers a capability.
	p.grant.installation = &b
	return nil
}

func (e *installationExecutor) verifyPrepared(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	o, err := e.preparedObservation(ctx, p, m)
	if err != nil {
		return err
	}
	a, err := e.d.Article.ReadVerifiedSnapshot(ctx, ArticleDeletionNamespaceV1)
	if err != nil {
		return err
	}
	if err := validateVerifiedJournal(a, m.Article); err != nil {
		return err
	}
	if a.Head != m.Article.Committed || a.Revision != o.articleRevision {
		return ErrAdmissionDrift
	}
	last, err := e.preparedObservation(ctx, p, m)
	if err != nil || !sameInstallationObservation(o, last) {
		return errors.Join(ErrAdmissionDrift, err)
	}
	return e.guard(ctx, p, m)
}

func (e *installationExecutor) legacy(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	if m.Kind != "anchored_migration" {
		return nil
	}
	if e.d.Legacy != nil {
		bindings, err := e.d.Legacy(ctx, p, m)
		if err != nil {
			return err
		}
		for _, b := range bindings {
			if err := e.guard(ctx, p, m); err != nil {
				return err
			}
			if err := e.d.Writer.BindLegacyPublication(ctx, p, b.RunID, b); err != nil {
				return err
			}
		}
	}
	s, err := e.d.Store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return err
	}
	for _, r := range s.Runs {
		if r.Outcome == RecoveryOutcomeSucceeded && (r.ArticlePublicationRef == nil || r.ArticlePublicationDigest == nil) {
			return ErrAdmissionIncomplete
		}
	}
	return e.guard(ctx, p, m)
}

// EncodeInstallationManifest preserves the original v1 flat namespace encoding.
func EncodeInstallationManifest(m InstallationManifest) ([]byte, error) {
	if err := validateInstallationManifest(m); err != nil {
		return nil, err
	}
	return encodeAdmissionObject([]admissionField{{"version", m.Version}, {"kind", m.Kind}, {"deployment_id", m.DeploymentID}, {"installation_id", m.InstallationID}, {"target_digest", m.TargetDigest}, {"source_approval_ref", m.SourceApprovalRef}, {"source_approval_digest", m.SourceApprovalDigest}, {"recovery", json.RawMessage(encodeManifestNamespace(m.Recovery))}, {"article", json.RawMessage(encodeManifestNamespace(m.Article))}, {"issued_at", m.IssuedAt}}), nil
}

func installationFields(raw []byte, names []string) (decodedJSONObject, error) {
	o, err := decodeJSONObject(raw)
	if err != nil || len(o.fields) != len(names) {
		return o, ErrAdmissionBinding
	}
	for _, name := range names {
		if v, ok := o.fields[name]; !ok || bytes.Equal(v, []byte("null")) {
			return o, ErrAdmissionBinding
		}
	}
	return o, nil
}

// DecodeInstallationManifest rejects incomplete, duplicate and non-v1 fields.
func DecodeInstallationManifest(raw []byte) (InstallationManifest, error) {
	var m InstallationManifest
	if len(raw) > 65536 || !utf8.Valid(raw) {
		return m, ErrAdmissionBinding
	}
	names := []string{"version", "kind", "deployment_id", "installation_id", "target_digest", "source_approval_ref", "source_approval_digest", "recovery", "article", "issued_at"}
	o, err := installationFields(raw, names)
	if err != nil {
		return m, err
	}
	dest := []any{&m.Version, &m.Kind, &m.DeploymentID, &m.InstallationID, &m.TargetDigest, &m.SourceApprovalRef, &m.SourceApprovalDigest}
	for i, v := range dest {
		if err := json.Unmarshal(o.fields[names[i]], v); err != nil {
			return m, ErrAdmissionBinding
		}
	}
	if err := json.Unmarshal(o.fields["issued_at"], &m.IssuedAt); err != nil {
		return m, ErrAdmissionBinding
	}
	for name, dst := range map[string]*AnchorSnapshot{"recovery": &m.Recovery, "article": &m.Article} {
		if err := decodeInstallationNamespace(o.fields[name], dst); err != nil {
			return m, err
		}
	}
	return m, validateInstallationManifest(m)
}

func decodeInstallationNamespace(raw []byte, a *AnchorSnapshot) error {
	names := []string{"namespace", "journal_id", "epoch", "head_sequence", "head_hash"}
	o, err := installationFields(raw, names)
	if err != nil {
		return err
	}
	for i, v := range []any{&a.Namespace, &a.JournalID, &a.Epoch, &a.Committed.Sequence, &a.Committed.Hash} {
		if err := json.Unmarshal(o.fields[names[i]], v); err != nil {
			return ErrAdmissionBinding
		}
	}
	return nil
}

type installationReader struct {
	commitments InstallationCommitmentStore
	sources     InstallationSourceVerifier
}

// NewInstallationReader constructs the pure bound commitment verifier.
func NewInstallationReader(c InstallationCommitmentStore, s InstallationSourceVerifier) (InstallationReader, error) {
	if missingRuntimePort(c) || missingRuntimePort(s) {
		return nil, ErrAdmissionBinding
	}
	return &installationReader{c, s}, nil
}

func (v *installationReader) VerifyGenesis(ctx context.Context, b InstallationBinding) (GenesisProof, error) {
	if err := installationContext(ctx); err != nil {
		return GenesisProof{}, err
	}
	c, revision, err := v.commitments.LoadInstallation(ctx, b.DeploymentID)
	if err != nil {
		return GenesisProof{}, err
	}
	if err := validateInstallationManifest(c.Manifest); err != nil {
		return GenesisProof{}, err
	}
	// Both approved manifest kinds can establish an origin before any Recovery
	// history. Admission separately requires zero runs/entries and continuity
	// from both initial heads; an existing Begin always uses full history.
	if c.Phase != "bound" || c.Manifest.Recovery.Committed != (JournalHead{}) || !admissionText(revision) || c.ManifestDigest != installationManifestDigest(c.Manifest) || b != installationBindingFor(c.Manifest) {
		return GenesisProof{}, ErrAdmissionBinding
	}
	if err := v.sources.VerifyInstallationSource(ctx, c.Manifest); err != nil {
		return GenesisProof{}, err
	}
	after, token, err := v.commitments.LoadInstallation(ctx, b.DeploymentID)
	if err != nil || after != c || token != revision {
		return GenesisProof{}, errors.Join(ErrAdmissionDrift, err)
	}
	if err := ctx.Err(); err != nil {
		return GenesisProof{}, err
	}
	return GenesisProof{Binding: b, RecoveryInitial: b.RecoveryInitial, ArticleInitial: b.ArticleInitial, CommitmentRevision: revision, CommitmentDigest: c.ManifestDigest}, nil
}

package identity

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

// NewRecoveryInstallationExecutor keeps installation writes on the same Store.
func NewRecoveryInstallationExecutor(store *RecoveryRunStore, rt recovery.Runtime, d recovery.InstallationDependencies, closures recovery.PublishedClosureReader) (recovery.InstallationExecutor, error) {
	if store == nil || store.pool == nil || missingIdentityInstallationPort(rt) || d.Writer != store || d.Store != store {
		return nil, recovery.ErrAdmissionBinding
	}
	x := &recoveryInstallationExecutor{store: store, runtime: rt, d: d, closures: closures}
	if d.Legacy != nil {
		d.Legacy = x.importLegacy
	}
	inner, err := recovery.NewInstallationExecutor(d)
	if err != nil {
		return nil, err
	}
	x.inner = inner
	return x, nil
}

type recoveryInstallationExecutor struct {
	store    *RecoveryRunStore
	runtime  recovery.Runtime
	d        recovery.InstallationDependencies
	closures recovery.PublishedClosureReader
	inner    recovery.InstallationExecutor
}
type installationAuthorityKey struct{}
type installationAuthority struct {
	executor *recoveryInstallationExecutor
	manifest recovery.InstallationManifest
	mu       sync.Mutex
	imported map[string]recovery.PublicationBinding
}

func missingIdentityInstallationPort(v any) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Pointer && reflect.ValueOf(v).IsNil())
}
func (x *recoveryInstallationExecutor) Install(ctx context.Context, p recovery.PhasePermit, m recovery.InstallationManifest) error {
	if ctx == nil {
		return recovery.ErrAdmissionBinding
	}
	if err := x.runtime.CheckPhase(ctx, p, "install"); err != nil {
		return err
	}
	a := &installationAuthority{executor: x, manifest: m, imported: map[string]recovery.PublicationBinding{}}
	return x.inner.Install(context.WithValue(ctx, installationAuthorityKey{}, a), p, m)
}
func (x *recoveryInstallationExecutor) importLegacy(ctx context.Context, p recovery.PhasePermit, m recovery.InstallationManifest) ([]recovery.PublicationBinding, error) {
	a, err := x.store.installationAuthority(ctx, p)
	if err != nil || a.manifest != m {
		return nil, errors.Join(recovery.ErrAdmissionBinding, err)
	}
	items, err := x.d.Legacy(ctx, p, m)
	if err != nil {
		return nil, err
	}
	if _, err := x.store.installationAuthority(ctx, p); err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, b := range items {
		if previous, ok := a.imported[b.RunID]; ok && previous != b {
			return nil, recovery.ErrAdmissionBinding
		}
		a.imported[b.RunID] = b
	}
	return items, nil
}
func (store *RecoveryRunStore) installationAuthority(ctx context.Context, p recovery.PhasePermit) (*installationAuthority, error) {
	if store == nil || store.pool == nil || ctx == nil {
		return nil, recovery.ErrAdmissionBinding
	}
	a, ok := ctx.Value(installationAuthorityKey{}).(*installationAuthority)
	if !ok || a == nil || a.executor == nil || a.executor.store != store {
		return nil, recovery.ErrAdmissionBinding
	}
	if err := a.executor.runtime.CheckPhase(ctx, p, "install"); err != nil {
		return nil, err
	}
	b, err := a.executor.runtime.ReadPermitBinding(ctx)
	if err != nil {
		return nil, err
	}
	if b.Purpose != "install" || b.InstallationID != a.manifest.InstallationID {
		return nil, recovery.ErrAdmissionBinding
	}
	return a, nil
}
func identityInstallationBinding(m recovery.InstallationManifest) (recovery.InstallationBinding, error) {
	raw, err := recovery.EncodeInstallationManifest(m)
	if err != nil {
		return recovery.InstallationBinding{}, err
	}
	return recovery.InstallationBinding{Version: m.Version, DeploymentID: m.DeploymentID, InstallationID: m.InstallationID, TargetDigest: m.TargetDigest,
		CommitmentRef: "recovery/v1/installation/" + m.DeploymentID, CommitmentDigest: recovery.SHA256Digest(raw), RecoveryInitial: m.Recovery, ArticleInitial: m.Article, InstalledAt: m.IssuedAt}, nil
}

// BindInstallation creates the immutable installation or verifies an exact retry.
func (store *RecoveryRunStore) BindInstallation(ctx context.Context, p recovery.PhasePermit, b recovery.InstallationBinding) (result error) {
	a, err := store.installationAuthority(ctx, p)
	if err != nil {
		return err
	}
	expected, err := identityInstallationBinding(a.manifest)
	if err != nil || b != expected {
		return errors.Join(recovery.ErrAdmissionBinding, err)
	}
	if err := a.executor.d.Sources.VerifyInstallationSource(ctx, a.manifest); err != nil {
		return err
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	if a.manifest.Kind == "genesis" {
		var n int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM recovery_runs`).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return recovery.ErrAdmissionBinding
		}
	}
	if _, err := store.installationAuthority(ctx, p); err != nil {
		return err
	}
	if err := insertInstallationBinding(ctx, tx, b); err != nil {
		return err
	}
	var s recovery.AdmissionSnapshot
	if err := scanAdmissionInstallation(tx.QueryRow(ctx, admissionInstallationSQL+` FOR UPDATE`), &s); err != nil {
		return err
	}
	if s.Installation != expected {
		return recovery.ErrAdmissionBinding
	}
	if _, err := store.installationAuthority(ctx, p); err != nil {
		return err
	}
	return commitRecoveryRun(ctx, tx, "commit installation binding")
}

func insertInstallationBinding(ctx context.Context, tx pgx.Tx, b recovery.InstallationBinding) error {
	r, a := b.RecoveryInitial, b.ArticleInitial
	_, err := tx.Exec(ctx, `INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,installation_id,target_digest,commitment_ref,commitment_digest,
	 recovery_journal_id,recovery_epoch,recovery_initial_sequence,recovery_initial_hash,article_journal_id,article_epoch,article_initial_sequence,article_initial_hash,revision,installed_at)
	 VALUES(true,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,1,$15) ON CONFLICT(singleton) DO NOTHING`,
		b.Version, b.DeploymentID, b.InstallationID, b.TargetDigest, b.CommitmentRef, b.CommitmentDigest, r.JournalID, r.Epoch, r.Committed.Sequence, r.Committed.Hash, a.JournalID, a.Epoch, a.Committed.Sequence, a.Committed.Hash, b.InstalledAt)
	return err
}

// BindLegacyPublication fills an original NULL pair from a verified old publication.
func (store *RecoveryRunStore) BindLegacyPublication(ctx context.Context, p recovery.PhasePermit, runID string, b recovery.PublicationBinding) (result error) {
	a, err := store.installationAuthority(ctx, p)
	if err != nil {
		return err
	}
	if a.manifest.Kind != "anchored_migration" || b.RunID != runID || missingIdentityInstallationPort(a.executor.closures) {
		return recovery.ErrAdmissionBinding
	}
	a.mu.Lock()
	imported, ok := a.imported[runID]
	a.mu.Unlock()
	if !ok || imported != b {
		return recovery.ErrAdmissionBinding
	}
	before, err := store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return err
	}
	var run recovery.AdmissionRun
	for _, r := range before.Runs {
		if r.RunID == runID {
			run = r
		}
	}
	if err := verifyLegacyInstallationRun(ctx, a, run, b); err != nil {
		return err
	}
	return store.writeLegacyInstallation(ctx, p, before, run, b)
}

func (store *RecoveryRunStore) writeLegacyInstallation(ctx context.Context, p recovery.PhasePermit, before recovery.AdmissionSnapshot, run recovery.AdmissionRun, b recovery.PublicationBinding) (result error) {
	runID := run.RunID
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM recovery_runs WHERE run_id=$1 FOR UPDATE`, runID); err != nil {
		return err
	}
	current, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, current) {
		return recovery.ErrAdmissionDrift
	}
	if run.ArticlePublicationRef != nil || run.ArticlePublicationDigest != nil {
		if run.ArticlePublicationRef == nil || run.ArticlePublicationDigest == nil || *run.ArticlePublicationRef != b.Ref || *run.ArticlePublicationDigest != b.Digest {
			return recovery.ErrAdmissionBinding
		}
	} else {
		if _, err := store.installationAuthority(ctx, p); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE recovery_runs SET article_publication_ref=$2,article_publication_digest=$3 WHERE run_id=$1 AND outcome='succeeded' AND article_publication_ref IS NULL AND article_publication_digest IS NULL`, runID, b.Ref, b.Digest)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return recovery.ErrAdmissionDrift
		}
	}
	if _, err := store.installationAuthority(ctx, p); err != nil {
		return err
	}
	return commitRecoveryRun(ctx, tx, "commit legacy publication binding")
}

func verifyLegacyInstallationRun(ctx context.Context, a *installationAuthority, r recovery.AdmissionRun, b recovery.PublicationBinding) error {
	if r.RunID != b.RunID || r.BackupRef != b.BackupRef || r.Outcome != recovery.RecoveryOutcomeSucceeded || r.JournalStage != recovery.StageConsumed || r.JournalConsumedAt == nil || r.CompletedAt == nil || r.ArticleEvidenceRef == nil || r.ArticleEvidenceDigest == nil || r.ArticleEvidenceCompletedAt == nil || r.ArticleEvidenceVersion == nil || r.ArticleNamespace == nil {
		return recovery.ErrAdmissionIncomplete
	}
	if *r.ArticleEvidenceVersion != recovery.ReplayEvidenceVersionV1 || *r.ArticleNamespace != recovery.ArticleDeletionNamespaceV1 {
		return recovery.ErrAdmissionBinding
	}
	if b.Version != "article-replay-publication/v1" || b.EvidenceRef != *r.ArticleEvidenceRef || b.EvidenceDigest != *r.ArticleEvidenceDigest || b.EvidenceCompletedAt != *r.ArticleEvidenceCompletedAt || b.JournalID != a.manifest.Article.JournalID || b.Epoch != a.manifest.Article.Epoch || !recoveryMetadataDigest.MatchString(b.Digest) || invalidStoreText(b.Ref) || b.PublishedAt.Before(b.EvidenceCompletedAt) {
		return recovery.ErrAdmissionBinding
	}
	j, err := a.executor.d.Recovery.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil {
		return err
	}
	if err := legacyJournalMatches(r, j, a.manifest.Recovery); err != nil {
		return err
	}
	q := recovery.ClosureRequest{RunID: r.RunID, BackupRef: r.BackupRef, EvidenceRef: b.EvidenceRef, EvidenceDigest: b.EvidenceDigest, PublicationRef: b.Ref, PublicationDigest: b.Digest}
	c, err := a.executor.closures.ReadPublishedClosure(ctx, q)
	if err != nil {
		return err
	}
	if c.PublicationRef != b.Ref || c.PublicationDigest != b.Digest || c.JournalID != b.JournalID || c.Epoch != b.Epoch || c.PublishedAt != b.PublishedAt || c.HistoricalHead != b.Head || c.Ready.RunID != r.RunID || c.Ready.BackupRef != r.BackupRef || c.Ready.EvidenceRef != b.EvidenceRef || c.Ready.EvidenceDigest != b.EvidenceDigest || c.Ready.EvidenceVersion != *r.ArticleEvidenceVersion || c.Ready.Namespace != *r.ArticleNamespace {
		return recovery.ErrAdmissionBinding
	}
	completed, err := time.Parse(time.RFC3339Nano, c.Ready.CompletedAt)
	if err != nil || !completed.Equal(b.EvidenceCompletedAt) || completed.Nanosecond()%1000 != 0 {
		return recovery.ErrAdmissionBinding
	}
	after, err := a.executor.d.Recovery.ReadVerifiedSnapshot(ctx, recovery.RecoveryNamespaceV1)
	if err != nil || !reflect.DeepEqual(j, after) {
		return errors.Join(recovery.ErrAdmissionDrift, err)
	}
	return ctx.Err()
}

func legacyJournalMatches(r recovery.AdmissionRun, j recovery.VerifiedJournalSnapshot, initial recovery.AnchorSnapshot) error {
	if j.Namespace != recovery.RecoveryNamespaceV1 || j.JournalID != initial.JournalID || j.Epoch != initial.Epoch || j.Pending != nil || invalidStoreText(j.Revision) || uint64(len(j.Entries)) != j.Head.Sequence {
		return recovery.ErrAdmissionBinding
	}
	begun, consumed := 0, 0
	for _, e := range j.Entries {
		if e.RunID != r.RunID {
			continue
		}
		if e.BackupRef != r.BackupRef || e.KeyID != r.JournalKeyID || e.KeyVersion != r.JournalKeyVersion || e.SecretFingerprint != r.JournalSecretFingerprint {
			return recovery.ErrAdmissionBinding
		}
		switch e.Stage {
		case recovery.StageBegun:
			if e.RecordedAt != r.StartedAt || begun != 0 || consumed != 0 {
				return recovery.ErrAdmissionBinding
			}
			begun++
		case recovery.StageConsumed:
			if r.JournalConsumedAt == nil || e.RecordedAt != *r.JournalConsumedAt || begun != 1 || consumed != 0 {
				return recovery.ErrAdmissionBinding
			}
			consumed++
		default:
			return recovery.ErrAdmissionBinding
		}
	}
	if begun != 1 || consumed != 1 {
		return recovery.ErrAdmissionIncomplete
	}
	return nil
}

var _ recovery.InstallationRepository = (*RecoveryRunStore)(nil)

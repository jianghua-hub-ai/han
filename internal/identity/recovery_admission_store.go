package identity

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
)

var _ recovery.AdmissionStore = (*RecoveryRunStore)(nil)
var recoveryMetadataDigest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ReadAdmissionSnapshot reads the complete installation/run projection in one short read-only snapshot.
func (store *RecoveryRunStore) ReadAdmissionSnapshot(ctx context.Context) (snapshot recovery.AdmissionSnapshot, result error) {
	if store == nil || store.pool == nil || ctx == nil {
		return snapshot, recovery.ErrAdmissionIncomplete
	}
	if err := ctx.Err(); err != nil {
		return snapshot, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return snapshot, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	return readOwnedAdmissionTransaction(ctx, tx)
}

func readOwnedAdmissionTransaction(ctx context.Context, tx pgx.Tx) (snapshot recovery.AdmissionSnapshot, result error) {
	defer func() { result = errors.Join(result, finishAdmissionRead(ctx, tx)) }()
	snapshot, err := readAdmissionTransaction(ctx, tx)
	if err != nil {
		return recovery.AdmissionSnapshot{}, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return recovery.AdmissionSnapshot{}, errors.Join(recovery.ErrAdmissionIncomplete, err)
	}
	return snapshot, nil
}

func finishAdmissionRead(ctx context.Context, tx pgx.Tx) error {
	err := tx.Rollback(ctx)
	if errors.Is(err, pgx.ErrTxClosed) {
		return nil
	}
	return err
}

// ReadAdmissionRevision recomputes the full digest; the counter alone is not a proof.
func (store *RecoveryRunStore) ReadAdmissionRevision(ctx context.Context) (recovery.StoreRevision, error) {
	s, err := store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return recovery.StoreRevision{}, err
	}
	return s.Revision, nil
}

func readAdmissionTransaction(ctx context.Context, tx pgx.Tx) (recovery.AdmissionSnapshot, error) {
	var s recovery.AdmissionSnapshot
	if err := scanAdmissionInstallation(tx.QueryRow(ctx, admissionInstallationSQL), &s); err != nil {
		return s, err
	}
	rows, err := tx.Query(ctx, admissionRunsSQL)
	if err != nil {
		return s, err
	}
	s.Runs, err = scanAdmissionRows(ctx, rows)
	if err != nil {
		return recovery.AdmissionSnapshot{}, err
	}
	raw, err := recovery.CanonicalAdmissionSnapshot(s)
	if err != nil {
		return recovery.AdmissionSnapshot{}, err
	}
	s.Revision.Digest = recovery.SHA256Digest(raw)
	return s, nil
}

func scanAdmissionInstallation(row rowScanner, s *recovery.AdmissionSnapshot) error {
	i := &s.Installation
	r, a := &i.RecoveryInitial, &i.ArticleInitial
	err := row.Scan(&i.Version, &i.DeploymentID, &i.InstallationID, &i.TargetDigest,
		&i.CommitmentRef, &i.CommitmentDigest, &r.JournalID, &r.Epoch, &r.Committed.Sequence, &r.Committed.Hash,
		&a.JournalID, &a.Epoch, &a.Committed.Sequence, &a.Committed.Hash, &s.Revision.Counter, &i.InstalledAt)
	if err != nil {
		return fmt.Errorf("read recovery installation: %w", err)
	}
	r.Namespace, a.Namespace = recovery.RecoveryNamespaceV1, recovery.ArticleDeletionNamespaceV1
	s.Revision.InstallationID = i.InstallationID
	return nil
}

func scanAdmissionRows(ctx context.Context, rows pgx.Rows) ([]recovery.AdmissionRun, error) {
	defer rows.Close()
	runs := make([]recovery.AdmissionRun, 0)
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		run, err := scanAdmissionRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := errors.Join(rows.Err(), ctx.Err()); err != nil {
		return nil, err
	}
	return runs, nil
}

type admissionRestoreColumns struct {
	version, runID, backup, deployment, installation, target *string
	artifact, approval, schema, receipt                      *string
	bytes                                                    *int64
	at                                                       *time.Time
}

func scanAdmissionRun(row rowScanner) (recovery.AdmissionRun, error) {
	var r recovery.AdmissionRun
	var x admissionRestoreColumns
	err := row.Scan(&r.RunID, &r.BackupRef, &r.JournalKeyID, &r.JournalKeyVersion, &r.JournalSecretFingerprint,
		&r.JournalStage, &r.StartedAt, &r.JournalConsumedAt, &r.ArticleEvidenceVersion, &r.ArticleEvidenceDigest,
		&r.ArticleEvidenceRef, &r.ArticleEvidenceCompletedAt, &r.ArticleNamespace, &r.ArticleReplayVerifiedAt,
		&r.ArticlePublicationRef, &r.ArticlePublicationDigest, &r.CompletedAt, &r.Outcome,
		&r.AccountCount, &r.PermanentCredentialCount, &r.SessionCount, &r.TemporaryCredentialCount,
		&r.IdempotencyCount, &r.PendingReviewCount, &r.AuditEventID,
		&x.version, &x.runID, &x.backup, &x.deployment, &x.installation, &x.target,
		&x.artifact, &x.approval, &x.schema, &x.receipt, &x.bytes, &x.at,
		&r.InitialAdminUserID, &r.InitialAdminBootstrapGeneration, &r.InitialAdminChangedGeneration,
		&r.InitialAdminChangeAuditID, &r.InitialAdminChangedAt)
	if err != nil {
		return recovery.AdmissionRun{}, fmt.Errorf("scan complete recovery run: %w", err)
	}
	r.RestoreBinding, err = x.project()
	return r, err
}

func (x admissionRestoreColumns) project() (*recovery.RestoreReceipt, error) {
	fields := []*string{x.version, x.runID, x.backup, x.deployment, x.installation, x.target, x.artifact, x.approval, x.schema, x.receipt}
	present := 0
	for _, p := range fields {
		if p != nil {
			present++
		}
	}
	if present == 0 && x.bytes == nil && x.at == nil {
		return nil, nil
	}
	if present != len(fields) || x.bytes == nil || x.at == nil {
		return nil, recovery.ErrAdmissionIncomplete
	}
	return &recovery.RestoreReceipt{Version: *x.version, RunID: *x.runID, BackupRef: *x.backup,
		DeploymentID: *x.deployment, InstallationID: *x.installation, TargetDigest: *x.target,
		ArtifactDigest: *x.artifact, SourceApprovalDigest: *x.approval, RestoredSchemaDigest: *x.schema,
		ReceiptDigest: *x.receipt, ArtifactBytes: *x.bytes, RestoredAt: *x.at}, nil
}

func validStorePublication(run recovery.RecoveryRun, c recovery.RecoveryRunChange) bool {
	p, v := c.ArticlePublication, c.ArticleEvidence
	if p == nil || v == nil {
		return false
	}
	completed, err := time.Parse(time.RFC3339Nano, v.CompletedAt)
	if err != nil || !completed.Equal(c.ArticleEvidenceCompletedAt) || completed.IsZero() {
		return false
	}
	return p.Version == "article-replay-publication/v1" && p.RunID == run.RunID && p.BackupRef == run.BackupRef &&
		p.EvidenceRef == v.EvidenceRef && p.EvidenceDigest == v.EvidenceDigest &&
		recoveryMetadataDigest.MatchString(v.EvidenceDigest) && canonicalStoreTime(c.ArticleEvidenceCompletedAt) &&
		!invalidStoreText(p.Ref) && recoveryMetadataDigest.MatchString(p.Digest) &&
		!invalidStoreText(p.JournalID) && p.Epoch > 0 && p.Epoch <= uint64(^uint64(0)>>1) &&
		p.Head == (recovery.JournalHead{Sequence: v.JournalHeadSequence, Hash: v.JournalHeadHash}) &&
		p.EvidenceCompletedAt.Equal(c.ArticleEvidenceCompletedAt) &&
		!p.PublishedAt.IsZero() && canonicalStoreTime(p.PublishedAt) && !p.PublishedAt.Before(p.EvidenceCompletedAt)
}

const admissionInstallationSQL = `SELECT metadata_version,deployment_id,installation_id,target_digest,
	commitment_ref,commitment_digest,recovery_journal_id,recovery_epoch,recovery_initial_sequence,recovery_initial_hash,
	article_journal_id,article_epoch,article_initial_sequence,article_initial_hash,revision,installed_at AT TIME ZONE 'UTC'
	FROM recovery_installation WHERE singleton`

const admissionRunsSQL = `SELECT r.run_id,r.backup_ref,r.journal_key_id,r.journal_key_version,r.journal_secret_fingerprint,
	r.journal_stage,r.started_at AT TIME ZONE 'UTC',r.journal_consumed_at AT TIME ZONE 'UTC',
	r.article_evidence_version,r.article_evidence_digest,r.article_evidence_ref,r.article_evidence_completed_at AT TIME ZONE 'UTC',
	r.article_namespace,r.article_replay_verified_at AT TIME ZONE 'UTC',r.article_publication_ref,r.article_publication_digest,
	r.completed_at AT TIME ZONE 'UTC',r.outcome,r.account_count,r.permanent_credential_count,r.session_count,
	r.temporary_credential_count,r.idempotency_count,r.pending_review_count,r.audit_event_id::text,
	x.receipt_version,x.run_id,x.backup_ref,x.deployment_id,x.installation_id,x.target_digest,
	x.artifact_digest,x.source_approval_digest,x.restored_schema_digest,x.receipt_digest,x.artifact_bytes,x.restored_at AT TIME ZONE 'UTC',
	r.initial_admin_user_id::text,r.initial_admin_bootstrap_generation,r.initial_admin_changed_generation,
	r.initial_admin_change_audit_id::text,r.initial_admin_changed_at AT TIME ZONE 'UTC'
	FROM recovery_runs r LEFT JOIN recovery_restore_bindings x ON x.run_id=r.run_id ORDER BY r.run_id COLLATE "C"`

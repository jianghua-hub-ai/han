package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Repository persists identity-owned models in a caller-owned transaction.
type Repository struct{}

// InsertUser inserts one account or tombstone.
func (Repository) InsertUser(ctx context.Context, tx pgx.Tx, user User) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	var err error
	if user.PermanentPasswordGeneration == nil {
		_, err = tx.Exec(ctx, insertUserSQL,
			user.ID, user.Username, user.Email, user.Phone, user.PermanentPasswordDigest,
			user.Role, user.IsInitialAdmin, user.Status, user.CredentialMode,
			user.RecoveryRequired, user.OnboardingPrivacyAcknowledgedVersion,
			user.OnboardingCompletedAt, user.RetainedEmail, user.CreatedAt,
			user.UpdatedAt, user.DeletedAt,
		)
	} else {
		// The generation column only exists once a permanent credential does,
		// mirroring users_permanent_password_generation_check.
		_, err = tx.Exec(ctx, insertUserWithPermanentPasswordGenerationSQL,
			user.ID, user.Username, user.Email, user.Phone, user.PermanentPasswordDigest,
			user.Role, user.IsInitialAdmin, user.Status, user.CredentialMode,
			user.RecoveryRequired, user.OnboardingPrivacyAcknowledgedVersion,
			user.OnboardingCompletedAt, user.RetainedEmail, user.CreatedAt,
			user.UpdatedAt, user.DeletedAt, *user.PermanentPasswordGeneration,
		)
	}
	if err != nil {
		return fmt.Errorf("insert identity user: %w", err)
	}
	return nil
}

// InsertIdentifier inserts one normalized identity identifier.
func (Repository) InsertIdentifier(
	ctx context.Context,
	tx pgx.Tx,
	identifier IdentityIdentifier,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertIdentifierSQL,
		identifier.UserID, identifier.Kind, identifier.NormalizedValue,
		identifier.Lifecycle, identifier.CreatedAt, identifier.ReleasedAt,
	); err != nil {
		return fmt.Errorf("insert identity identifier: %w", err)
	}
	return nil
}

// InsertSession inserts one server-side session record.
func (Repository) InsertSession(ctx context.Context, tx pgx.Tx, session Session) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertSessionSQL,
		session.ID, session.UserID, session.TokenDigest, session.CSRFTokenDigest,
		session.AuthenticatedCredentialKind, session.AuthenticatedCredentialID,
		session.IssuedAt, session.LastSeenAt, session.AbsoluteExpiresAt,
		session.IdleExpiresAt, session.RotatedFromID, session.RevokedAt,
		session.RevokeReason,
	); err != nil {
		return fmt.Errorf("insert identity session: %w", err)
	}
	return nil
}

// RevokeSessionsByUserID revokes every active session for one user.
func (Repository) RevokeSessionsByUserID(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	revokedAt time.Time,
	reason string,
) (int64, error) {
	if err := requireTransaction(tx); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, revokeSessionsByUserIDSQL, revokedAt, reason, userID)
	if err != nil {
		return 0, fmt.Errorf("revoke identity sessions by user: %w", err)
	}
	return tag.RowsAffected(), nil
}

// InsertTemporaryCredential inserts one temporary credential lifecycle record.
func (Repository) InsertTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertTemporaryCredentialSQL,
		credential.ID, credential.UserID, credential.Purpose, credential.Generation,
		credential.State, credential.SecretDigest, credential.DeliveredToEmail,
		credential.IssuedAt, credential.ExpiresAt, credential.SupersededAt,
		credential.ConsumedAt, credential.InvalidatedAt,
	); err != nil {
		return fmt.Errorf("insert identity temporary credential: %w", err)
	}
	return nil
}

// InsertPrivacyNotice inserts one versioned notice.
func (Repository) InsertPrivacyNotice(
	ctx context.Context,
	tx pgx.Tx,
	notice PrivacyNotice,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertPrivacyNoticeSQL,
		notice.Version, notice.Content, notice.ContentHash, notice.EffectiveAt, notice.Status,
	); err != nil {
		return fmt.Errorf("insert privacy notice: %w", err)
	}
	return nil
}

// InsertPrivacyAcknowledgement inserts append-only confirmation evidence.
func (Repository) InsertPrivacyAcknowledgement(
	ctx context.Context,
	tx pgx.Tx,
	acknowledgement PrivacyAcknowledgement,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertPrivacyAcknowledgementSQL,
		acknowledgement.ID, acknowledgement.UserID, acknowledgement.NoticeVersion,
		acknowledgement.AcknowledgedAt, acknowledgement.SourceKind,
		acknowledgement.SourceFingerprint,
	); err != nil {
		return fmt.Errorf("insert privacy acknowledgement: %w", err)
	}
	return nil
}

// InsertAdministratorInformationDeclaration inserts append-only declaration evidence.
func (Repository) InsertAdministratorInformationDeclaration(
	ctx context.Context,
	tx pgx.Tx,
	declaration AdministratorInformationDeclaration,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertAdministratorInformationDeclarationSQL,
		declaration.ID, declaration.ActorUserID, declaration.TargetLocatorDigest,
		declaration.StatementVersion, declaration.DeclaredAt, declaration.Purpose,
	); err != nil {
		return fmt.Errorf("insert administrator information declaration: %w", err)
	}
	return nil
}

// InsertLoginAttemptBucket inserts one persistent login attempt bucket.
func (Repository) InsertLoginAttemptBucket(
	ctx context.Context,
	tx pgx.Tx,
	bucket LoginAttemptBucket,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertLoginAttemptBucketSQL,
		bucket.Kind, bucket.KeyDigest, bucket.DigestKeyVersion,
		bucket.WindowStartedAt, bucket.FailureCount, bucket.BlockedUntil,
		bucket.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert login attempt bucket: %w", err)
	}
	return nil
}

// InsertRecoveryAccountReview inserts one recovered account review record.
func (Repository) InsertRecoveryAccountReview(
	ctx context.Context,
	tx pgx.Tx,
	review RecoveryAccountReview,
) error {
	if err := requireTransaction(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, insertRecoveryAccountReviewSQL,
		review.RecoveryRunID, review.UserID, review.BackupRole, review.BackupStatus,
		review.BackupDeletedAt, review.Decision, review.EvidenceRef, review.ReviewedBy,
		review.ReviewedAt, review.CredentialResetAt,
	); err != nil {
		return fmt.Errorf("insert recovery account review: %w", err)
	}
	return nil
}

func requireTransaction(tx pgx.Tx) error {
	if tx == nil {
		return errors.New("identity transaction is required")
	}
	return nil
}

const insertUserSQL = `
	INSERT INTO users (
		id, username, email, phone, permanent_password_digest, role,
		is_initial_admin, status, credential_mode, recovery_required,
		onboarding_privacy_acknowledged_version, onboarding_completed_at,
		retained_email, created_at, updated_at, deleted_at
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15, $16
	)
`

const insertUserWithPermanentPasswordGenerationSQL = `
	INSERT INTO users (
		id, username, email, phone, permanent_password_digest, role,
		is_initial_admin, status, credential_mode, recovery_required,
		onboarding_privacy_acknowledged_version, onboarding_completed_at,
		retained_email, created_at, updated_at, deleted_at,
		permanent_password_generation
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15, $16, $17
	)
`

const insertIdentifierSQL = `
	INSERT INTO identity_identifiers (
		user_id, kind, normalized_value, lifecycle, created_at, released_at
	)
	VALUES ($1, $2, $3, $4, $5, $6)
`

const insertSessionSQL = `
	INSERT INTO sessions (
		id, user_id, token_digest, csrf_token_digest,
		authenticated_credential_kind, authenticated_credential_id,
		issued_at, last_seen_at, absolute_expires_at, idle_expires_at,
		rotated_from_id, revoked_at, revoke_reason
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const revokeSessionsByUserIDSQL = `
	UPDATE sessions
	SET revoked_at = $1, revoke_reason = $2
	WHERE user_id = $3 AND revoked_at IS NULL
`

const insertTemporaryCredentialSQL = `
	INSERT INTO temporary_credentials (
		id, user_id, purpose, generation, state, secret_digest,
		delivered_to_email, issued_at, expires_at, superseded_at,
		consumed_at, invalidated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const insertPrivacyNoticeSQL = `
	INSERT INTO privacy_notices (
		version, content, content_hash, effective_at, status
	)
	VALUES ($1, $2, $3, $4, $5)
`

const insertPrivacyAcknowledgementSQL = `
	INSERT INTO privacy_acknowledgements (
		id, user_id, notice_version, acknowledged_at, source_kind, source_fingerprint
	)
	VALUES ($1, $2, $3, $4, $5, $6)
`

const insertAdministratorInformationDeclarationSQL = `
	INSERT INTO administrator_information_declarations (
		id, actor_user_id, target_locator_digest, statement_version, declared_at, purpose
	)
	VALUES ($1, $2, $3, $4, $5, $6)
`

const insertLoginAttemptBucketSQL = `
	INSERT INTO login_attempt_buckets (
		bucket_kind, bucket_key_digest, digest_key_version,
		window_started_at, failure_count, blocked_until, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
`

const insertRecoveryAccountReviewSQL = `
	INSERT INTO recovery_account_reviews (
		recovery_run_id, user_id, backup_role, backup_status, backup_deleted_at,
		decision, evidence_ref, reviewed_by, reviewed_at, credential_reset_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

CREATE TABLE users (
	id uuid PRIMARY KEY,
	username text NOT NULL CHECK (btrim(username) <> ''),
	email text CHECK (email IS NULL OR btrim(email) <> ''),
	phone text CHECK (phone IS NULL OR btrim(phone) <> ''),
	permanent_password_digest text
		CHECK (permanent_password_digest IS NULL OR btrim(permanent_password_digest) <> ''),
	role text NOT NULL CHECK (role IN ('member', 'admin')),
	is_initial_admin boolean NOT NULL DEFAULT false,
	status text NOT NULL CHECK (status IN ('active', 'suspended', 'deleted')),
	credential_mode text NOT NULL CHECK (
		credential_mode IN (
			'permanent',
			'bootstrap',
			'temporary',
			'recovery_locked',
			'recovery_bootstrap'
		)
	),
	recovery_required boolean NOT NULL DEFAULT false,
	onboarding_privacy_acknowledged_version text
		CHECK (
			onboarding_privacy_acknowledged_version IS NULL
			OR btrim(onboarding_privacy_acknowledged_version) <> ''
		),
	onboarding_completed_at timestamp with time zone,
	retained_email text CHECK (retained_email IS NULL OR btrim(retained_email) <> ''),
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	deleted_at timestamp with time zone,
	CHECK (NOT is_initial_admin OR role = 'admin'),
	CHECK (
		status = 'deleted'
		OR is_initial_admin
		OR (email IS NOT NULL AND phone IS NOT NULL)
	),
	CHECK (
		status <> 'deleted'
		OR (
			email IS NULL
			AND phone IS NULL
			AND permanent_password_digest IS NULL
		)
	),
	CHECK (retained_email IS NULL OR status = 'deleted'),
	CHECK ((status = 'deleted') = (deleted_at IS NOT NULL))
);

CREATE UNIQUE INDEX users_single_initial_admin_idx
	ON users ((is_initial_admin))
	WHERE is_initial_admin;

CREATE TABLE identity_identifiers (
	user_id uuid NOT NULL REFERENCES users (id),
	kind text NOT NULL CHECK (kind IN ('username', 'email', 'phone')),
	normalized_value text NOT NULL CHECK (btrim(normalized_value) <> ''),
	lifecycle text NOT NULL CHECK (lifecycle IN ('active', 'reserved')),
	created_at timestamp with time zone NOT NULL,
	released_at timestamp with time zone,
	PRIMARY KEY (user_id, kind),
	UNIQUE (kind, normalized_value)
);

CREATE TABLE sessions (
	id uuid PRIMARY KEY,
	user_id uuid NOT NULL REFERENCES users (id),
	token_digest bytea NOT NULL UNIQUE CHECK (octet_length(token_digest) > 0),
	csrf_token_digest bytea NOT NULL CHECK (octet_length(csrf_token_digest) > 0),
	authenticated_credential_kind text NOT NULL CHECK (
		authenticated_credential_kind IN (
			'permanent',
			'bootstrap',
			'temporary',
			'recovery_bootstrap'
		)
	),
	authenticated_credential_id uuid,
	issued_at timestamp with time zone NOT NULL,
	last_seen_at timestamp with time zone NOT NULL,
	absolute_expires_at timestamp with time zone NOT NULL,
	idle_expires_at timestamp with time zone NOT NULL,
	rotated_from_id uuid REFERENCES sessions (id) ON DELETE SET NULL,
	revoked_at timestamp with time zone,
	revoke_reason text CHECK (revoke_reason IS NULL OR btrim(revoke_reason) <> ''),
	CHECK (last_seen_at >= issued_at),
	CHECK (absolute_expires_at > issued_at),
	CHECK (idle_expires_at > issued_at),
	CHECK (
		(
			authenticated_credential_kind = 'temporary'
			AND authenticated_credential_id IS NOT NULL
		)
		OR (
			authenticated_credential_kind <> 'temporary'
			AND authenticated_credential_id IS NULL
		)
	)
);

CREATE INDEX sessions_user_id_idx
	ON sessions (user_id);

CREATE TABLE temporary_credentials (
	id uuid PRIMARY KEY,
	user_id uuid NOT NULL REFERENCES users (id),
	purpose text NOT NULL CHECK (purpose IN ('new_administrator', 'password_reset')),
	generation bigint NOT NULL CHECK (generation >= 1),
	state text NOT NULL CHECK (
		state IN (
			'pending_delivery',
			'active',
			'expired',
			'superseded',
			'consumed',
			'recovery_invalidated'
		)
	),
	secret_digest text NOT NULL CHECK (btrim(secret_digest) <> ''),
	delivered_to_email text NOT NULL CHECK (btrim(delivered_to_email) <> ''),
	issued_at timestamp with time zone NOT NULL,
	expires_at timestamp with time zone NOT NULL,
	superseded_at timestamp with time zone,
	consumed_at timestamp with time zone,
	invalidated_at timestamp with time zone,
	UNIQUE (user_id, generation),
	UNIQUE (id, user_id),
	CHECK (expires_at > issued_at)
);

CREATE UNIQUE INDEX temporary_credentials_current_user_idx
	ON temporary_credentials (user_id)
	WHERE state IN ('pending_delivery', 'active', 'expired');

ALTER TABLE sessions
	ADD CONSTRAINT sessions_temporary_credential_user_fk
	FOREIGN KEY (authenticated_credential_id, user_id)
	REFERENCES temporary_credentials (id, user_id);

CREATE TABLE privacy_notices (
	version text PRIMARY KEY CHECK (btrim(version) <> ''),
	content text NOT NULL CHECK (btrim(content) <> ''),
	content_hash bytea NOT NULL CHECK (octet_length(content_hash) > 0),
	effective_at timestamp with time zone NOT NULL,
	status text NOT NULL CHECK (btrim(status) <> '')
);

CREATE UNIQUE INDEX privacy_notices_single_current_idx
	ON privacy_notices ((status))
	WHERE status = 'current';

CREATE FUNCTION enforce_privacy_notice_version_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF NEW.version IS DISTINCT FROM OLD.version
		OR NEW.content IS DISTINCT FROM OLD.content
		OR NEW.content_hash IS DISTINCT FROM OLD.content_hash
		OR NEW.effective_at IS DISTINCT FROM OLD.effective_at THEN
		RAISE EXCEPTION 'privacy notice version content is immutable'
			USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;

CREATE TRIGGER privacy_notices_immutable_version
	BEFORE UPDATE ON privacy_notices
	FOR EACH ROW
	EXECUTE FUNCTION enforce_privacy_notice_version_immutable();

CREATE TABLE privacy_acknowledgements (
	id uuid PRIMARY KEY,
	user_id uuid NOT NULL REFERENCES users (id),
	notice_version text NOT NULL REFERENCES privacy_notices (version),
	acknowledged_at timestamp with time zone NOT NULL,
	source_kind text NOT NULL CHECK (btrim(source_kind) <> ''),
	source_fingerprint text NOT NULL CHECK (btrim(source_fingerprint) <> ''),
	UNIQUE (user_id, notice_version)
);

CREATE TRIGGER privacy_acknowledgements_append_only
	BEFORE UPDATE OR DELETE ON privacy_acknowledgements
	FOR EACH STATEMENT
	EXECUTE FUNCTION reject_append_only_mutation();

CREATE TABLE administrator_information_declarations (
	id uuid PRIMARY KEY,
	actor_user_id uuid NOT NULL REFERENCES users (id),
	target_locator_digest bytea NOT NULL CHECK (octet_length(target_locator_digest) > 0),
	statement_version text NOT NULL CHECK (btrim(statement_version) <> ''),
	declared_at timestamp with time zone NOT NULL,
	purpose text NOT NULL CHECK (btrim(purpose) <> '')
);

CREATE TRIGGER administrator_information_declarations_append_only
	BEFORE UPDATE OR DELETE ON administrator_information_declarations
	FOR EACH STATEMENT
	EXECUTE FUNCTION reject_append_only_mutation();

CREATE TABLE login_attempt_buckets (
	bucket_kind text NOT NULL CHECK (bucket_kind IN ('subject', 'source', 'subject_source')),
	bucket_key_digest bytea NOT NULL CHECK (octet_length(bucket_key_digest) > 0),
	digest_key_version text NOT NULL CHECK (btrim(digest_key_version) <> ''),
	window_started_at timestamp with time zone NOT NULL,
	failure_count bigint NOT NULL CHECK (failure_count >= 0),
	blocked_until timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	PRIMARY KEY (bucket_kind, bucket_key_digest)
);

CREATE INDEX login_attempt_buckets_blocked_until_idx
	ON login_attempt_buckets (blocked_until);

CREATE TABLE recovery_account_reviews (
	recovery_run_id text NOT NULL CHECK (btrim(recovery_run_id) <> ''),
	user_id uuid NOT NULL REFERENCES users (id),
	backup_role text NOT NULL CHECK (backup_role IN ('member', 'admin')),
	backup_status text NOT NULL CHECK (backup_status IN ('active', 'suspended', 'deleted')),
	backup_deleted_at timestamp with time zone,
	decision text NOT NULL CHECK (decision IN ('pending', 'restored', 'kept_deleted', 'denied')),
	evidence_ref text CHECK (evidence_ref IS NULL OR btrim(evidence_ref) <> ''),
	reviewed_by uuid REFERENCES users (id),
	reviewed_at timestamp with time zone,
	credential_reset_at timestamp with time zone,
	PRIMARY KEY (recovery_run_id, user_id),
	CHECK (
		(
			decision = 'pending'
			AND reviewed_by IS NULL
			AND reviewed_at IS NULL
			AND credential_reset_at IS NULL
		)
		OR (
			decision = 'restored'
			AND reviewed_by IS NOT NULL
			AND reviewed_at IS NOT NULL
			AND credential_reset_at IS NOT NULL
		)
		OR (
			decision IN ('kept_deleted', 'denied')
			AND reviewed_by IS NOT NULL
			AND reviewed_at IS NOT NULL
			AND credential_reset_at IS NULL
		)
	)
);

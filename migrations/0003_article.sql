CREATE TABLE deletion_operations (
	id uuid PRIMARY KEY,
	article_id uuid NOT NULL UNIQUE,
	article_version bigint NOT NULL CHECK (article_version >= 1),
	delete_kind text NOT NULL CHECK (delete_kind IN ('ordinary', 'violating', 'orphan')),
	actor_user_id uuid REFERENCES users (id) ON DELETE RESTRICT,
	actor_role text CHECK (actor_role IS NULL OR btrim(actor_role) <> ''),
	orphaned_from_user_id uuid REFERENCES users (id) ON DELETE RESTRICT,
	status text NOT NULL CHECK (
		status IN ('pending_journal', 'journaled', 'finalizing', 'completed')
	),
	tombstone_sequence bigint CHECK (tombstone_sequence > 0),
	journal_key_id text CHECK (journal_key_id IS NULL OR btrim(journal_key_id) <> ''),
	journal_key_version text
		CHECK (journal_key_version IS NULL OR btrim(journal_key_version) <> ''),
	claimed_by text CHECK (claimed_by IS NULL OR btrim(claimed_by) <> ''),
	locked_until timestamp with time zone,
	audit_event_id uuid REFERENCES audit_events (id) ON DELETE RESTRICT,
	started_at timestamp with time zone NOT NULL,
	completed_at timestamp with time zone,
	UNIQUE (id, article_id),
	CHECK ((delete_kind = 'orphan') = (orphaned_from_user_id IS NOT NULL)),
	CHECK (
		(status = 'pending_journal')
		= (
			tombstone_sequence IS NULL
			AND journal_key_id IS NULL
			AND journal_key_version IS NULL
		)
	),
	CHECK (
		(status <> 'pending_journal')
		= (
			tombstone_sequence IS NOT NULL
			AND journal_key_id IS NOT NULL
			AND journal_key_version IS NOT NULL
		)
	),
	CHECK (
		(claimed_by IS NULL) = (locked_until IS NULL)
	),
	CHECK (
		(status = 'finalizing') = (claimed_by IS NOT NULL)
	),
	CHECK (
		(audit_event_id IS NULL) = (completed_at IS NULL)
	),
	CHECK (
		(status = 'completed') = (audit_event_id IS NOT NULL)
	),
	CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX deletion_operations_finalizer_idx
	ON deletion_operations (status, locked_until, started_at, id);

CREATE FUNCTION enforce_deletion_operation_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		RAISE EXCEPTION 'deletion operation cannot be deleted'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.status = 'completed' THEN
		RAISE EXCEPTION 'completed deletion operation is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF NEW.id IS DISTINCT FROM OLD.id
		OR NEW.article_id IS DISTINCT FROM OLD.article_id
		OR NEW.article_version IS DISTINCT FROM OLD.article_version
		OR NEW.delete_kind IS DISTINCT FROM OLD.delete_kind
		OR NEW.actor_user_id IS DISTINCT FROM OLD.actor_user_id
		OR NEW.actor_role IS DISTINCT FROM OLD.actor_role
		OR NEW.orphaned_from_user_id IS DISTINCT FROM OLD.orphaned_from_user_id
		OR NEW.started_at IS DISTINCT FROM OLD.started_at THEN
		RAISE EXCEPTION 'deletion operation identity is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.tombstone_sequence IS NOT NULL
		AND (
			NEW.tombstone_sequence IS DISTINCT FROM OLD.tombstone_sequence
			OR NEW.journal_key_id IS DISTINCT FROM OLD.journal_key_id
			OR NEW.journal_key_version IS DISTINCT FROM OLD.journal_key_version
		) THEN
		RAISE EXCEPTION 'deletion operation tombstone binding is immutable'
			USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;

CREATE TRIGGER deletion_operations_immutable
	BEFORE UPDATE OR DELETE ON deletion_operations
	FOR EACH ROW
	EXECUTE FUNCTION enforce_deletion_operation_immutable();

CREATE TABLE articles (
	id uuid PRIMARY KEY,
	author_id uuid REFERENCES users (id) ON DELETE RESTRICT,
	orphaned_from_user_id uuid REFERENCES users (id) ON DELETE RESTRICT,
	title text NOT NULL CHECK (title ~ '[^[:space:]]'),
	body text NOT NULL CHECK (body ~ '[^[:space:]]'),
	publication_state text NOT NULL CHECK (publication_state IN ('published', 'held')),
	violation_state text NOT NULL CHECK (violation_state IN ('clear', 'violating')),
	violation_category text CHECK (violation_category IN ('violence', 'sexual')),
	deletion_state text NOT NULL CHECK (deletion_state IN ('active', 'deletion_pending')),
	deletion_operation_id uuid,
	version bigint NOT NULL CHECK (version >= 1),
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	CHECK (
		(publication_state = 'held')
		= (author_id IS NULL AND orphaned_from_user_id IS NOT NULL)
	),
	CHECK (orphaned_from_user_id IS NULL OR author_id IS NULL),
	CHECK ((violation_state = 'violating') = (violation_category IS NOT NULL)),
	CHECK (
		(deletion_state = 'deletion_pending')
		= (deletion_operation_id IS NOT NULL)
	),
	FOREIGN KEY (deletion_operation_id, id)
		REFERENCES deletion_operations (id, article_id)
		ON DELETE RESTRICT
);

CREATE INDEX articles_created_at_id_idx
	ON articles (created_at DESC, id DESC);

CREATE INDEX articles_author_id_idx
	ON articles (author_id);

CREATE INDEX articles_orphan_source_idx
	ON articles (orphaned_from_user_id, created_at DESC, id DESC);

CREATE INDEX articles_violation_state_idx
	ON articles (violation_state, created_at DESC, id DESC);

CREATE INDEX articles_deletion_state_idx
	ON articles (deletion_state, created_at DESC, id DESC);

CREATE TABLE transfer_notices (
	id uuid PRIMARY KEY,
	article_ref uuid NOT NULL,
	new_author_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
	former_deleted_user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL
);

CREATE TABLE transfer_notice_recipients (
	notice_id uuid NOT NULL REFERENCES transfer_notices (id) ON DELETE CASCADE,
	recipient_kind text NOT NULL CHECK (
		recipient_kind IN ('new_author', 'former_deleted_author')
	),
	email_snapshot text NOT NULL CHECK (btrim(email_snapshot) <> ''),
	status text NOT NULL CHECK (
		status IN ('pending', 'retry_wait', 'processing', 'accepted', 'failed_terminal')
	),
	claimed_by text CHECK (claimed_by IS NULL OR btrim(claimed_by) <> ''),
	locked_until timestamp with time zone,
	next_attempt_at timestamp with time zone,
	attempt_count bigint NOT NULL CHECK (attempt_count >= 0),
	max_attempts bigint NOT NULL CHECK (max_attempts >= 1),
	last_attempt_at timestamp with time zone,
	accepted_at timestamp with time zone,
	failure_bucket text CHECK (failure_bucket IS NULL OR btrim(failure_bucket) <> ''),
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	PRIMARY KEY (notice_id, recipient_kind),
	CHECK (attempt_count <= max_attempts),
	CHECK (
		status <> 'processing'
		OR (
			claimed_by IS NOT NULL
			AND locked_until IS NOT NULL
			AND last_attempt_at IS NOT NULL
		)
	),
	CHECK (status <> 'retry_wait' OR next_attempt_at IS NOT NULL),
	CHECK ((status = 'accepted') = (accepted_at IS NOT NULL))
);

CREATE INDEX transfer_notice_recipients_claim_idx
	ON transfer_notice_recipients (status, next_attempt_at, locked_until);

CREATE FUNCTION require_transfer_notice_recipients_complete(checked_notice_id uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM transfer_notices WHERE id = checked_notice_id
	) AND NOT (
		SELECT count(*) = 2
			AND count(*) FILTER (WHERE recipient_kind = 'new_author') = 1
			AND count(*) FILTER (
				WHERE recipient_kind = 'former_deleted_author'
			) = 1
		FROM transfer_notice_recipients
		WHERE notice_id = checked_notice_id
	) THEN
		RAISE EXCEPTION 'transfer notice must have exactly two bound recipients'
			USING ERRCODE = '23514';
	END IF;
END;
$$;

CREATE FUNCTION enforce_transfer_notice_recipients_complete()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF TG_TABLE_NAME = 'transfer_notices' THEN
		IF TG_OP <> 'DELETE' THEN
			PERFORM require_transfer_notice_recipients_complete(NEW.id);
		END IF;
		IF TG_OP <> 'INSERT' AND OLD.id IS DISTINCT FROM NEW.id THEN
			PERFORM require_transfer_notice_recipients_complete(OLD.id);
		END IF;
	ELSE
		IF TG_OP <> 'DELETE' THEN
			PERFORM require_transfer_notice_recipients_complete(NEW.notice_id);
		END IF;
		IF TG_OP <> 'INSERT'
			AND (
				TG_OP = 'DELETE'
				OR OLD.notice_id IS DISTINCT FROM NEW.notice_id
			) THEN
			PERFORM require_transfer_notice_recipients_complete(OLD.notice_id);
		END IF;
	END IF;
	RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER transfer_notices_recipients_complete
	AFTER INSERT OR UPDATE ON transfer_notices
	DEFERRABLE INITIALLY DEFERRED
	FOR EACH ROW
	EXECUTE FUNCTION enforce_transfer_notice_recipients_complete();

CREATE CONSTRAINT TRIGGER transfer_notice_recipients_complete
	AFTER INSERT OR UPDATE OR DELETE ON transfer_notice_recipients
	DEFERRABLE INITIALLY DEFERRED
	FOR EACH ROW
	EXECUTE FUNCTION enforce_transfer_notice_recipients_complete();

CREATE TABLE deletion_replay_runs (
	run_id text NOT NULL CHECK (btrim(run_id) <> ''),
	namespace text NOT NULL CHECK (namespace = 'article-deletion/v1'),
	backup_ref text NOT NULL CHECK (btrim(backup_ref) <> ''),
	input_digest text CHECK (input_digest IS NULL OR btrim(input_digest) <> ''),
	current_evidence_ref text
		CHECK (current_evidence_ref IS NULL OR btrim(current_evidence_ref) <> ''),
	status text NOT NULL CHECK (status IN ('running', 'failed', 'ready')),
	last_error_bucket text
		CHECK (last_error_bucket IS NULL OR btrim(last_error_bucket) <> ''),
	started_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	PRIMARY KEY (run_id, namespace),
	UNIQUE (run_id, namespace, backup_ref),
	CHECK (updated_at >= started_at),
	CHECK (
		status <> 'ready'
		OR (input_digest IS NOT NULL AND current_evidence_ref IS NOT NULL)
	),
	CHECK (current_evidence_ref IS NULL OR input_digest IS NOT NULL)
);

CREATE FUNCTION enforce_deletion_replay_run_binding_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		RAISE EXCEPTION 'deletion replay run binding cannot be deleted'
			USING ERRCODE = '55000';
	END IF;
	IF NEW.run_id IS DISTINCT FROM OLD.run_id
		OR NEW.namespace IS DISTINCT FROM OLD.namespace
		OR NEW.backup_ref IS DISTINCT FROM OLD.backup_ref THEN
		RAISE EXCEPTION 'deletion replay run binding is immutable'
			USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;

CREATE TRIGGER deletion_replay_runs_binding_immutable
	BEFORE UPDATE OR DELETE ON deletion_replay_runs
	FOR EACH ROW
	EXECUTE FUNCTION enforce_deletion_replay_run_binding_immutable();

CREATE TABLE deletion_replay_evidence (
	evidence_version text NOT NULL CHECK (evidence_version = 'article-deletion-replay/v1'),
	run_id text NOT NULL,
	backup_ref text NOT NULL CHECK (btrim(backup_ref) <> ''),
	namespace text NOT NULL CHECK (namespace = 'article-deletion/v1'),
	journal_head_sequence bigint NOT NULL CHECK (journal_head_sequence >= 0),
	journal_head_hash text NOT NULL CHECK (btrim(journal_head_hash) <> ''),
	inventory_scope text NOT NULL CHECK (btrim(inventory_scope) <> ''),
	backup_cutoff timestamp with time zone NOT NULL,
	inventory_digest text NOT NULL CHECK (btrim(inventory_digest) <> ''),
	deletion_operations_digest text NOT NULL
		CHECK (btrim(deletion_operations_digest) <> ''),
	audit_events_digest text NOT NULL CHECK (btrim(audit_events_digest) <> ''),
	tombstones_total bigint NOT NULL CHECK (tombstones_total >= 0),
	tombstones_replayed bigint NOT NULL CHECK (tombstones_replayed >= 0),
	already_absent bigint NOT NULL CHECK (already_absent >= 0),
	deletion_operations_reconciled_count bigint NOT NULL
		CHECK (deletion_operations_reconciled_count >= 0),
	audit_events_verified_count bigint NOT NULL
		CHECK (audit_events_verified_count >= 0),
	evidence_digest text NOT NULL CHECK (btrim(evidence_digest) <> ''),
	evidence_ref text PRIMARY KEY CHECK (btrim(evidence_ref) <> ''),
	completed_at timestamp with time zone NOT NULL,
	UNIQUE (evidence_ref, run_id, namespace),
	UNIQUE (run_id, namespace, evidence_version, evidence_digest),
	FOREIGN KEY (run_id, namespace, backup_ref)
		REFERENCES deletion_replay_runs (run_id, namespace, backup_ref)
		ON DELETE RESTRICT,
	CHECK (tombstones_total = tombstones_replayed + already_absent)
);

CREATE INDEX deletion_replay_evidence_run_idx
	ON deletion_replay_evidence (run_id, namespace, completed_at DESC);

CREATE TRIGGER deletion_replay_evidence_append_only
	BEFORE UPDATE OR DELETE ON deletion_replay_evidence
	FOR EACH STATEMENT
	EXECUTE FUNCTION reject_append_only_mutation();

ALTER TABLE deletion_replay_runs
	ADD CONSTRAINT deletion_replay_runs_current_evidence_fk
	FOREIGN KEY (current_evidence_ref, run_id, namespace)
	REFERENCES deletion_replay_evidence (evidence_ref, run_id, namespace)
	ON DELETE RESTRICT;

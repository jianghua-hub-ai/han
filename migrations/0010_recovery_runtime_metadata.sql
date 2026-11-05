CREATE TABLE recovery_installation (
	singleton boolean PRIMARY KEY CHECK (singleton),
	metadata_version text NOT NULL CHECK (metadata_version = 'han-recovery-install/v1'),
	deployment_id text NOT NULL CHECK (btrim(deployment_id) <> ''),
	installation_id text NOT NULL UNIQUE CHECK (btrim(installation_id) <> ''),
	target_digest text NOT NULL CHECK (target_digest ~ '^sha256:[0-9a-f]{64}$'),
	commitment_ref text NOT NULL CHECK (btrim(commitment_ref) <> ''),
	commitment_digest text NOT NULL CHECK (commitment_digest ~ '^sha256:[0-9a-f]{64}$'),
	recovery_journal_id text NOT NULL CHECK (btrim(recovery_journal_id) <> ''),
	article_journal_id text NOT NULL CHECK (btrim(article_journal_id) <> ''),
	recovery_epoch bigint NOT NULL CHECK (recovery_epoch > 0),
	article_epoch bigint NOT NULL CHECK (article_epoch > 0),
	recovery_initial_sequence bigint NOT NULL CHECK (recovery_initial_sequence >= 0),
	article_initial_sequence bigint NOT NULL CHECK (article_initial_sequence >= 0),
	recovery_initial_hash text NOT NULL,
	article_initial_hash text NOT NULL,
	revision bigint NOT NULL CHECK (revision > 0),
	installed_at timestamptz NOT NULL,
	UNIQUE (installation_id, deployment_id, target_digest),
	CHECK ((recovery_initial_sequence = 0 AND recovery_initial_hash = '') OR
		(recovery_initial_sequence > 0 AND recovery_initial_hash ~ '^sha256:[0-9a-f]{64}$')),
	CHECK ((article_initial_sequence = 0 AND article_initial_hash = '') OR
		(article_initial_sequence > 0 AND article_initial_hash ~ '^sha256:[0-9a-f]{64}$'))
);

ALTER TABLE recovery_runs
	ADD CONSTRAINT recovery_runs_run_backup_unique UNIQUE (run_id, backup_ref),
	ADD COLUMN article_publication_ref text CHECK (article_publication_ref IS NULL OR btrim(article_publication_ref) <> ''),
	ADD COLUMN article_publication_digest text CHECK (article_publication_digest IS NULL OR article_publication_digest ~ '^sha256:[0-9a-f]{64}$'),
	ADD COLUMN initial_admin_user_id uuid REFERENCES users(id),
	ADD COLUMN initial_admin_bootstrap_generation bigint CHECK (initial_admin_bootstrap_generation > 0 AND initial_admin_bootstrap_generation < 9223372036854775807),
	ADD COLUMN initial_admin_changed_generation bigint,
	ADD COLUMN initial_admin_change_audit_id uuid REFERENCES audit_events(id),
	ADD COLUMN initial_admin_changed_at timestamptz,
	ADD CONSTRAINT recovery_runs_publication_pair CHECK ((article_publication_ref IS NULL) = (article_publication_digest IS NULL)),
	ADD CONSTRAINT recovery_runs_initial_proof CHECK (
		(initial_admin_user_id IS NULL) = (initial_admin_bootstrap_generation IS NULL)
		AND (initial_admin_changed_generation IS NULL) = (initial_admin_change_audit_id IS NULL)
		AND (initial_admin_changed_generation IS NULL) = (initial_admin_changed_at IS NULL)
		AND (initial_admin_changed_generation IS NULL OR
			(initial_admin_bootstrap_generation IS NOT NULL AND initial_admin_changed_generation = initial_admin_bootstrap_generation + 1))
	);

CREATE TABLE recovery_restore_bindings (
	run_id text PRIMARY KEY,
	backup_ref text NOT NULL,
	installation_id text NOT NULL,
	deployment_id text NOT NULL,
	artifact_digest text NOT NULL CHECK (artifact_digest ~ '^sha256:[0-9a-f]{64}$'),
	target_digest text NOT NULL CHECK (target_digest ~ '^sha256:[0-9a-f]{64}$'),
	receipt_digest text NOT NULL CHECK (receipt_digest ~ '^sha256:[0-9a-f]{64}$'),
	source_approval_digest text NOT NULL CHECK (source_approval_digest ~ '^sha256:[0-9a-f]{64}$'),
	restored_schema_digest text NOT NULL CHECK (restored_schema_digest ~ '^sha256:[0-9a-f]{64}$'),
	artifact_bytes bigint NOT NULL CHECK (artifact_bytes > 0),
	restored_at timestamptz NOT NULL,
	receipt_version text NOT NULL CHECK (receipt_version = 'han-db-restore/v1'),
	FOREIGN KEY (run_id, backup_ref) REFERENCES recovery_runs(run_id, backup_ref),
	FOREIGN KEY (installation_id, deployment_id, target_digest)
		REFERENCES recovery_installation(installation_id, deployment_id, target_digest)
);

CREATE TRIGGER recovery_restore_bindings_immutable
	BEFORE UPDATE OR DELETE ON recovery_restore_bindings
	FOR EACH ROW EXECUTE FUNCTION reject_append_only_mutation();

ALTER TABLE sessions
	ADD COLUMN access_scope text NOT NULL DEFAULT 'ordinary' CHECK (access_scope IN ('ordinary', 'recovery_maintenance')),
	ADD COLUMN recovery_run_id text REFERENCES recovery_runs(run_id),
	ADD COLUMN recovery_installation_id text REFERENCES recovery_installation(installation_id),
	ADD COLUMN recovery_runtime_id text,
	ADD COLUMN recovery_generation bigint CHECK (recovery_generation > 0),
	ADD CONSTRAINT sessions_recovery_scope CHECK (
		(access_scope = 'ordinary' AND recovery_run_id IS NULL AND recovery_installation_id IS NULL
			AND recovery_runtime_id IS NULL AND recovery_generation IS NULL)
		OR (access_scope = 'recovery_maintenance' AND recovery_run_id IS NOT NULL
			AND recovery_installation_id IS NOT NULL AND recovery_runtime_id IS NOT NULL AND btrim(recovery_runtime_id) <> ''
			AND recovery_generation IS NOT NULL)
	);

ALTER TABLE password_reset_operations
	ADD COLUMN recovery_run_id text,
	ADD COLUMN recovery_installation_id text REFERENCES recovery_installation(installation_id),
	ADD COLUMN recovery_evidence_ref text,
	ADD COLUMN recovery_evidence_digest text,
	ADD COLUMN recovery_actor_id uuid REFERENCES users(id),
	ADD COLUMN recovery_target_role text,
	ADD COLUMN recovery_target_status text,
	ADD CONSTRAINT password_reset_recovery_approval CHECK (
		(recovery_run_id IS NULL AND recovery_installation_id IS NULL AND recovery_evidence_ref IS NULL
			AND recovery_evidence_digest IS NULL AND recovery_actor_id IS NULL
			AND recovery_target_role IS NULL AND recovery_target_status IS NULL)
		OR (recovery_run_id IS NOT NULL AND recovery_installation_id IS NOT NULL
			AND recovery_evidence_ref IS NOT NULL AND btrim(recovery_evidence_ref) <> ''
			AND recovery_evidence_digest IS NOT NULL AND recovery_evidence_digest ~ '^sha256:[0-9a-f]{64}$'
			AND recovery_actor_id IS NOT NULL AND recovery_target_role IS NOT NULL AND recovery_target_role IN ('member','admin')
			AND recovery_target_status IS NOT NULL AND recovery_target_status IN ('active','suspended'))
	),
	ADD CONSTRAINT password_reset_recovery_review_fk FOREIGN KEY (recovery_run_id, user_id)
		REFERENCES recovery_account_reviews(recovery_run_id, user_id) DEFERRABLE INITIALLY DEFERRED,
	ADD CONSTRAINT password_reset_recovery_operation_run_user_unique UNIQUE (operation_id, recovery_run_id, user_id);

ALTER TABLE recovery_account_reviews
	ADD COLUMN restoration_operation_id uuid,
	ADD CONSTRAINT recovery_review_operation_user_fk FOREIGN KEY (restoration_operation_id, user_id)
		REFERENCES password_reset_operations(operation_id, user_id) DEFERRABLE INITIALLY DEFERRED,
	ADD CONSTRAINT recovery_review_operation_run_user_fk FOREIGN KEY (restoration_operation_id, recovery_run_id, user_id)
		REFERENCES password_reset_operations(operation_id, recovery_run_id, user_id) DEFERRABLE INITIALLY DEFERRED;

CREATE FUNCTION enforce_recovery_installation_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		RAISE EXCEPTION 'installation cannot be deleted' USING ERRCODE = '55000';
	END IF;
	IF TG_OP = 'INSERT' THEN
		IF NEW.revision <> 1 THEN RAISE EXCEPTION 'installation starts at revision one' USING ERRCODE = '55000'; END IF;
		RETURN NEW;
	END IF;
	IF (to_jsonb(NEW) - 'revision') IS DISTINCT FROM (to_jsonb(OLD) - 'revision') THEN
		RAISE EXCEPTION 'installation binding is immutable' USING ERRCODE = '55000';
	END IF;
	IF NEW.revision = OLD.revision THEN NEW.revision := OLD.revision + 1;
	ELSIF NEW.revision <> OLD.revision + 1 THEN
		RAISE EXCEPTION 'invalid installation revision successor' USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;
CREATE TRIGGER recovery_installation_immutable BEFORE INSERT OR UPDATE OR DELETE ON recovery_installation
	FOR EACH ROW EXECUTE FUNCTION enforce_recovery_installation_immutable();

CREATE FUNCTION advance_recovery_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	UPDATE recovery_installation SET revision = revision + 1 WHERE singleton;
	IF NOT FOUND THEN RAISE EXCEPTION 'recovery installation not bound' USING ERRCODE = '55000'; END IF;
	RETURN NULL;
END;
$$;
CREATE TRIGGER recovery_runs_revision AFTER INSERT OR UPDATE ON recovery_runs
	FOR EACH ROW EXECUTE FUNCTION advance_recovery_revision();
CREATE TRIGGER recovery_restore_revision AFTER INSERT ON recovery_restore_bindings
	FOR EACH ROW EXECUTE FUNCTION advance_recovery_revision();
CREATE TRIGGER recovery_review_revision AFTER INSERT OR UPDATE OR DELETE ON recovery_account_reviews
	FOR EACH ROW EXECUTE FUNCTION advance_recovery_revision();

CREATE FUNCTION advance_recovery_operation_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		IF OLD.recovery_run_id IS NULL THEN RETURN NULL; END IF;
	ELSE
		IF NEW.recovery_run_id IS NULL THEN RETURN NULL; END IF;
	END IF;
	UPDATE recovery_installation SET revision = revision + 1 WHERE singleton;
	IF NOT FOUND THEN RAISE EXCEPTION 'recovery installation not bound' USING ERRCODE = '55000'; END IF;
	RETURN NULL;
END;
$$;
CREATE TRIGGER recovery_operation_revision AFTER INSERT OR UPDATE OR DELETE ON password_reset_operations
	FOR EACH ROW EXECUTE FUNCTION advance_recovery_operation_revision();

CREATE FUNCTION enforce_recovery_operation_approval() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	IF ROW(NEW.recovery_run_id,NEW.recovery_installation_id,NEW.recovery_evidence_ref,NEW.recovery_evidence_digest,
		NEW.recovery_actor_id,NEW.recovery_target_role,NEW.recovery_target_status)
		IS DISTINCT FROM ROW(OLD.recovery_run_id,OLD.recovery_installation_id,OLD.recovery_evidence_ref,OLD.recovery_evidence_digest,
		OLD.recovery_actor_id,OLD.recovery_target_role,OLD.recovery_target_status) THEN
		RAISE EXCEPTION 'recovery approval is immutable' USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;
CREATE TRIGGER recovery_operation_approval BEFORE UPDATE ON password_reset_operations
	FOR EACH ROW EXECUTE FUNCTION enforce_recovery_operation_approval();

CREATE FUNCTION enforce_recovery_review_metadata() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		IF OLD.decision <> 'pending' THEN RAISE EXCEPTION 'terminal recovery review cannot be deleted' USING ERRCODE = '55000'; END IF;
		RETURN OLD;
	END IF;
	IF OLD.decision <> 'pending' AND NEW IS DISTINCT FROM OLD THEN
		RAISE EXCEPTION 'terminal recovery review is immutable' USING ERRCODE = '55000';
	END IF;
	IF ROW(NEW.recovery_run_id,NEW.user_id,NEW.backup_role,NEW.backup_status,NEW.backup_deleted_at)
		IS DISTINCT FROM ROW(OLD.recovery_run_id,OLD.user_id,OLD.backup_role,OLD.backup_status,OLD.backup_deleted_at) THEN
		RAISE EXCEPTION 'recovery review source is immutable' USING ERRCODE = '55000';
	END IF;
	IF OLD.restoration_operation_id IS NOT NULL AND NEW.restoration_operation_id IS DISTINCT FROM OLD.restoration_operation_id THEN
		IF NEW.restoration_operation_id IS NULL OR NOT EXISTS (SELECT 1 FROM password_reset_operations
			WHERE operation_id = OLD.restoration_operation_id AND state IN ('failed','invalidated')) THEN
			RAISE EXCEPTION 'unresolved recovery candidate cannot be replaced' USING ERRCODE = '55000';
		END IF;
	END IF;
	RETURN NEW;
END;
$$;
CREATE TRIGGER recovery_review_metadata BEFORE UPDATE OR DELETE ON recovery_account_reviews
	FOR EACH ROW EXECUTE FUNCTION enforce_recovery_review_metadata();

CREATE FUNCTION enforce_recovery_run_metadata() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE field_name text;
BEGIN
	IF TG_OP = 'UPDATE' THEN
		FOREACH field_name IN ARRAY ARRAY['article_publication_ref','article_publication_digest',
			'initial_admin_user_id','initial_admin_bootstrap_generation','initial_admin_changed_generation',
			'initial_admin_change_audit_id','initial_admin_changed_at','audit_event_id',
			'article_evidence_version','article_evidence_digest','article_evidence_ref','article_evidence_completed_at',
			'article_namespace','article_replay_verified_at','journal_consumed_at','completed_at'] LOOP
			IF (to_jsonb(OLD)->field_name) <> 'null'::jsonb AND
				(to_jsonb(NEW)->field_name) IS DISTINCT FROM (to_jsonb(OLD)->field_name) THEN
				RAISE EXCEPTION 'recovery provenance is immutable' USING ERRCODE = '55000';
			END IF;
		END LOOP;
		IF OLD.journal_stage = 'consumed' AND NEW.journal_stage <> 'consumed' THEN
			RAISE EXCEPTION 'recovery consumption cannot reverse' USING ERRCODE = '55000';
		END IF;
	END IF;
	IF TG_OP = 'INSERT' OR NEW.outcome IS DISTINCT FROM OLD.outcome THEN
		IF NEW.outcome IN ('article_deletion_ready','ready_to_consume','journal_consumed','succeeded') AND NEW.article_publication_ref IS NULL THEN
			RAISE EXCEPTION 'new recovery stage requires publication' USING ERRCODE = '55000';
		END IF;
		IF NEW.outcome IN ('identity_ready','article_deletion_ready','ready_to_consume','journal_consumed','succeeded') AND
			(NEW.initial_admin_changed_generation IS NULL OR EXISTS (SELECT 1 FROM recovery_account_reviews WHERE recovery_run_id=NEW.run_id AND decision='pending')) THEN
			RAISE EXCEPTION 'new recovery stage requires changed trust and complete reviews' USING ERRCODE = '55000';
		END IF;
	END IF;
	RETURN NEW;
END;
$$;
CREATE TRIGGER recovery_runs_metadata BEFORE INSERT OR UPDATE ON recovery_runs
	FOR EACH ROW EXECUTE FUNCTION enforce_recovery_run_metadata();

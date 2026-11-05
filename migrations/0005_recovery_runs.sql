CREATE TABLE recovery_runs (
	run_id text PRIMARY KEY CHECK (btrim(run_id) <> ''),
	backup_ref text NOT NULL CHECK (btrim(backup_ref) <> ''),
	journal_key_id text NOT NULL CHECK (btrim(journal_key_id) <> ''),
	journal_key_version text NOT NULL CHECK (btrim(journal_key_version) <> ''),
	journal_secret_fingerprint text NOT NULL
		CHECK (btrim(journal_secret_fingerprint) <> ''),
	journal_stage text NOT NULL CHECK (journal_stage IN ('begun', 'consumed')),
	journal_consumed_at timestamp with time zone,
	article_evidence_version text,
	article_evidence_digest text,
	article_evidence_ref text,
	article_evidence_completed_at timestamp with time zone,
	article_namespace text,
	article_replay_verified_at timestamp with time zone,
	started_at timestamp with time zone NOT NULL,
	completed_at timestamp with time zone,
	outcome text NOT NULL CHECK (
		outcome IN (
			'running', 'failed', 'identity_ready', 'article_deletion_ready',
			'ready_to_consume', 'journal_consumed', 'succeeded'
		)
	),
	account_count bigint NOT NULL DEFAULT 0 CHECK (account_count >= 0),
	permanent_credential_count bigint NOT NULL DEFAULT 0
		CHECK (permanent_credential_count >= 0),
	session_count bigint NOT NULL DEFAULT 0 CHECK (session_count >= 0),
	temporary_credential_count bigint NOT NULL DEFAULT 0
		CHECK (temporary_credential_count >= 0),
	idempotency_count bigint NOT NULL DEFAULT 0 CHECK (idempotency_count >= 0),
	pending_review_count bigint NOT NULL DEFAULT 0 CHECK (pending_review_count >= 0),
	audit_event_id uuid,
	CHECK ((journal_stage = 'consumed') = (journal_consumed_at IS NOT NULL)),
	CHECK (
		(article_evidence_version IS NULL)
		= (article_evidence_digest IS NULL)
		AND (article_evidence_version IS NULL) = (article_evidence_ref IS NULL)
		AND (article_evidence_version IS NULL) = (article_evidence_completed_at IS NULL)
		AND (article_evidence_version IS NULL) = (article_namespace IS NULL)
		AND (article_evidence_version IS NULL) = (article_replay_verified_at IS NULL)
	),
	CHECK ((outcome = 'succeeded') = (completed_at IS NOT NULL)),
	CHECK (journal_consumed_at IS NULL OR journal_consumed_at >= started_at),
	CHECK (completed_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX recovery_runs_outcome_idx ON recovery_runs (outcome, started_at, run_id);

CREATE FUNCTION enforce_recovery_run_binding_and_transition()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF TG_OP = 'DELETE' THEN
		RAISE EXCEPTION 'recovery run cannot be deleted' USING ERRCODE = '55000';
	END IF;
	IF TG_OP = 'INSERT' THEN
		IF NEW.outcome <> 'running' THEN
			RAISE EXCEPTION 'recovery run must be created running' USING ERRCODE = '55000';
		END IF;
		RETURN NEW;
	END IF;
	IF NEW.run_id IS DISTINCT FROM OLD.run_id
		OR NEW.backup_ref IS DISTINCT FROM OLD.backup_ref
		OR NEW.journal_key_id IS DISTINCT FROM OLD.journal_key_id
		OR NEW.journal_key_version IS DISTINCT FROM OLD.journal_key_version
		OR NEW.journal_secret_fingerprint IS DISTINCT FROM OLD.journal_secret_fingerprint
		OR NEW.started_at IS DISTINCT FROM OLD.started_at THEN
		RAISE EXCEPTION 'recovery run binding is immutable' USING ERRCODE = '55000';
	END IF;
	IF NEW.outcome IS DISTINCT FROM OLD.outcome AND NOT (
		(OLD.outcome = 'running' AND NEW.outcome = 'identity_ready')
		OR (OLD.outcome = 'identity_ready' AND NEW.outcome = 'article_deletion_ready')
		OR (OLD.outcome = 'article_deletion_ready' AND NEW.outcome = 'ready_to_consume')
		OR (OLD.outcome = 'ready_to_consume' AND NEW.outcome = 'journal_consumed')
		OR (OLD.outcome = 'journal_consumed' AND NEW.outcome = 'succeeded')
		OR (OLD.outcome IN (
			'running', 'identity_ready', 'article_deletion_ready',
			'ready_to_consume', 'journal_consumed'
		) AND NEW.outcome = 'failed')
	) THEN
		RAISE EXCEPTION 'invalid recovery run transition' USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;

CREATE TRIGGER recovery_runs_binding_and_transition
	BEFORE INSERT OR UPDATE OR DELETE ON recovery_runs
	FOR EACH ROW
	EXECUTE FUNCTION enforce_recovery_run_binding_and_transition();

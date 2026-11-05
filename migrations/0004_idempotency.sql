CREATE TABLE idempotency_records (
	scope_hash bytea NOT NULL CHECK (octet_length(scope_hash) = 32),
	key_locator bytea NOT NULL CHECK (octet_length(key_locator) = 32),
	key_digest bytea NOT NULL CHECK (octet_length(key_digest) = 32),
	request_digest bytea NOT NULL CHECK (octet_length(request_digest) = 32),
	digest_key_version text NOT NULL CHECK (btrim(digest_key_version) <> ''),
	status text NOT NULL CHECK (
		status IN (
			'processing',
			'succeeded',
			'failed',
			'uncertain',
			'recovery_review_required'
		)
	),
	execution_generation bigint NOT NULL CHECK (execution_generation >= 1),
	result_code text CHECK (result_code IS NULL OR btrim(result_code) <> ''),
	response_snapshot jsonb,
	locked_until timestamp with time zone,
	reviewed_by text CHECK (reviewed_by IS NULL OR btrim(reviewed_by) <> ''),
	reviewed_at timestamp with time zone,
	expires_at timestamp with time zone NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	PRIMARY KEY (scope_hash, key_locator),
	CHECK ((result_code IS NULL) = (response_snapshot IS NULL)),
	CHECK (
		(status IN ('succeeded', 'failed'))
		= (result_code IS NOT NULL)
	),
	CHECK (
		response_snapshot IS NULL
		OR (
			jsonb_typeof(response_snapshot) = 'object'
			AND response_snapshot ? 'outcome'
			AND jsonb_typeof(response_snapshot -> 'outcome') = 'string'
			AND btrim(response_snapshot ->> 'outcome') <> ''
			AND response_snapshot ->> 'outcome' ~ '[^[:space:]]'
			AND (
				NOT response_snapshot ? 'resource_id'
				OR jsonb_typeof(response_snapshot -> 'resource_id') = 'string'
			)
			AND (
				NOT response_snapshot ? 'error_code'
				OR jsonb_typeof(response_snapshot -> 'error_code') = 'string'
			)
			AND response_snapshot - ARRAY[
				'outcome',
				'resource_id',
				'error_code'
			] = '{}'::jsonb
		)
	),
	CHECK ((status = 'processing') = (locked_until IS NOT NULL)),
	CHECK ((reviewed_by IS NULL) = (reviewed_at IS NULL)),
	CHECK (expires_at > created_at),
	CHECK (updated_at >= created_at)
);

CREATE INDEX idempotency_records_expiry_idx
	ON idempotency_records (expires_at);

CREATE INDEX idempotency_records_recovery_idx
	ON idempotency_records (status, updated_at)
	WHERE status IN ('uncertain', 'recovery_review_required');

CREATE FUNCTION enforce_idempotency_record_transition()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF NEW.scope_hash IS DISTINCT FROM OLD.scope_hash
		OR NEW.key_locator IS DISTINCT FROM OLD.key_locator
		OR NEW.key_digest IS DISTINCT FROM OLD.key_digest
		OR NEW.request_digest IS DISTINCT FROM OLD.request_digest
		OR NEW.digest_key_version IS DISTINCT FROM OLD.digest_key_version
		OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
		RAISE EXCEPTION 'idempotency record binding is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF NEW.updated_at < OLD.updated_at THEN
		RAISE EXCEPTION 'idempotency record time cannot move backwards'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.status IN ('succeeded', 'failed') THEN
		RAISE EXCEPTION 'completed idempotency record is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.status = 'processing' THEN
		IF NEW.status NOT IN (
			'succeeded',
			'failed',
			'uncertain',
			'recovery_review_required'
		) OR NEW.execution_generation <> OLD.execution_generation THEN
			RAISE EXCEPTION 'invalid processing idempotency transition'
				USING ERRCODE = '55000';
		END IF;
	ELSIF OLD.status = 'uncertain' THEN
		IF NEW.status <> 'recovery_review_required'
			OR NEW.execution_generation <> OLD.execution_generation THEN
			RAISE EXCEPTION 'invalid uncertain idempotency transition'
				USING ERRCODE = '55000';
		END IF;
	ELSIF OLD.status = 'recovery_review_required' THEN
		IF NOT (
			(
				NEW.status = 'failed'
				AND NEW.execution_generation = OLD.execution_generation
			)
			OR (
				NEW.status = 'processing'
				AND NEW.execution_generation = OLD.execution_generation + 1
			)
		) THEN
			RAISE EXCEPTION 'invalid reviewed idempotency transition'
				USING ERRCODE = '55000';
		END IF;
	END IF;
	RETURN NEW;
END;
$$;

CREATE TRIGGER idempotency_records_state_machine
	BEFORE UPDATE ON idempotency_records
	FOR EACH ROW
	EXECUTE FUNCTION enforce_idempotency_record_transition();

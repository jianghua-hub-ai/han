DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM temporary_credentials
		WHERE purpose = 'password_reset'
			AND state IN ('pending_delivery', 'active', 'expired')
	) THEN
		RAISE EXCEPTION
			'password-reset durable migration cannot explain a legacy current credential'
			USING ERRCODE = '55000';
	END IF;
	IF EXISTS (
		SELECT 1
		FROM idempotency_records
		WHERE status IN ('processing', 'uncertain')
			OR (
				status = 'succeeded'
				AND (
					(
						result_code = 'created'
						AND jsonb_typeof(response_snapshot) = 'object'
						AND NOT response_snapshot ? 'schema_version'
						AND response_snapshot ->> 'outcome' = 'created'
						AND jsonb_typeof(response_snapshot -> 'outcome') = 'string'
						AND jsonb_typeof(response_snapshot -> 'resource_id') = 'string'
						AND btrim(response_snapshot ->> 'resource_id') <> ''
						AND NOT response_snapshot ? 'error_code'
						AND response_snapshot - ARRAY[
							'outcome',
							'resource_id'
						] = '{}'::jsonb
					)
					OR (
						result_code = 'created'
						AND jsonb_typeof(response_snapshot) = 'object'
						AND response_snapshot ->> 'schema_version'
							= 'identity-administrator-create-response/v1'
						AND jsonb_typeof(
							response_snapshot -> 'schema_version'
						) = 'string'
						AND response_snapshot -> 'http_status' = '201'::jsonb
						AND jsonb_typeof(
							response_snapshot -> 'http_status'
						) = 'number'
						AND response_snapshot ->> 'result_code' = 'created'
						AND jsonb_typeof(
							response_snapshot -> 'result_code'
						) = 'string'
						AND jsonb_typeof(response_snapshot -> 'body') = 'object'
						AND response_snapshot - ARRAY[
							'schema_version',
							'http_status',
							'result_code',
							'body'
						] = '{}'::jsonb
						AND jsonb_typeof(
							response_snapshot -> 'body' -> 'id'
						) = 'string'
						AND btrim(response_snapshot -> 'body' ->> 'id') <> ''
						AND jsonb_typeof(
							response_snapshot -> 'body' -> 'username'
						) = 'string'
						AND btrim(response_snapshot -> 'body' ->> 'username') <> ''
						AND response_snapshot -> 'body' ->> 'role' = 'admin'
						AND jsonb_typeof(
							response_snapshot -> 'body' -> 'role'
						) = 'string'
						AND response_snapshot -> 'body' ->> 'status' = 'active'
						AND jsonb_typeof(
							response_snapshot -> 'body' -> 'status'
						) = 'string'
						AND response_snapshot -> 'body'
							-> 'must_change_password' = 'true'::jsonb
						AND response_snapshot -> 'body'
							-> 'must_acknowledge_privacy' = 'true'::jsonb
						AND jsonb_typeof(
							response_snapshot -> 'body'
								-> 'temporary_credential_expires_at'
						) = 'string'
						AND btrim(
							response_snapshot -> 'body'
								->> 'temporary_credential_expires_at'
						) <> ''
						AND response_snapshot -> 'body'
							->> 'delivery_status' = 'accepted'
						AND jsonb_typeof(
							response_snapshot -> 'body' -> 'delivery_status'
						) = 'string'
						AND (response_snapshot -> 'body') - ARRAY[
							'id',
							'username',
							'role',
							'status',
							'must_change_password',
							'must_acknowledge_privacy',
							'temporary_credential_expires_at',
							'delivery_status'
						] = '{}'::jsonb
					)
				) IS NOT TRUE
			)
	) THEN
		RAISE EXCEPTION
			'password-reset durable migration found unclassified or in-flight generic idempotency records'
			USING ERRCODE = '55000';
	END IF;
END;
$$;

ALTER TABLE idempotency_records
	ADD COLUMN password_reset_operation_id uuid,
	ADD COLUMN password_reset_user_id uuid,
	ADD CONSTRAINT idempotency_records_password_reset_marker_check
	CHECK (
		(password_reset_operation_id IS NULL)
		= (password_reset_user_id IS NULL)
	),
	ADD CONSTRAINT idempotency_records_password_reset_identity_unique
	UNIQUE (
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	),
	ADD CONSTRAINT idempotency_records_password_reset_operation_identity_unique
	UNIQUE (
		password_reset_operation_id,
		password_reset_user_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	);

ALTER TABLE users
	ADD COLUMN auth_generation bigint NOT NULL DEFAULT 0
		CHECK (auth_generation >= 0),
	ADD COLUMN permanent_password_generation bigint,
	ADD COLUMN password_reset_execution_generation bigint NOT NULL DEFAULT 0
		CHECK (password_reset_execution_generation >= 0),
	ADD COLUMN password_reset_current_operation_id uuid,
	ADD COLUMN permanent_password_source_operation_id uuid,
	ADD COLUMN permanent_password_source_credential_id uuid,
	ADD COLUMN auth_integrity_hold boolean NOT NULL DEFAULT false,
	ADD COLUMN auth_integrity_hold_operation_id uuid,
	ADD COLUMN auth_integrity_hold_reason text,
	ADD COLUMN auth_integrity_hold_at timestamp with time zone;

UPDATE users
SET permanent_password_generation = 0
WHERE permanent_password_digest IS NOT NULL;

ALTER TABLE users
	ADD CONSTRAINT users_permanent_password_generation_check
	CHECK (
		(permanent_password_digest IS NULL) = (permanent_password_generation IS NULL)
		AND (
			permanent_password_generation IS NULL
			OR (
				permanent_password_generation >= 0
				AND permanent_password_generation <= auth_generation
			)
		)
	),
	ADD CONSTRAINT users_permanent_password_lineage_pair_check
	CHECK (
		(permanent_password_source_operation_id IS NULL)
		= (permanent_password_source_credential_id IS NULL)
	),
	ADD CONSTRAINT users_password_reset_current_operation_check
	CHECK (
		(
			password_reset_execution_generation = 0
			AND password_reset_current_operation_id IS NULL
		)
		OR (
			password_reset_execution_generation >= 1
			AND password_reset_current_operation_id IS NOT NULL
		)
	),
	ADD CONSTRAINT users_auth_integrity_hold_check
	CHECK (
		(
			auth_integrity_hold
			AND auth_integrity_hold_reason IS NOT NULL
			AND btrim(auth_integrity_hold_reason) <> ''
			AND auth_integrity_hold_at IS NOT NULL
		)
		OR (
			NOT auth_integrity_hold
			AND auth_integrity_hold_operation_id IS NULL
			AND auth_integrity_hold_reason IS NULL
			AND auth_integrity_hold_at IS NULL
		)
	);

ALTER TABLE sessions
	ADD COLUMN auth_generation bigint NOT NULL DEFAULT 0
		CHECK (auth_generation >= 0),
	ADD COLUMN lineage_operation_id uuid,
	ADD COLUMN lineage_credential_id uuid,
	ADD CONSTRAINT sessions_lineage_pair_check
	CHECK (
		(lineage_operation_id IS NULL) = (lineage_credential_id IS NULL)
	);

ALTER TABLE temporary_credentials
	ADD COLUMN auth_generation bigint NOT NULL DEFAULT 0
		CHECK (auth_generation >= 0),
	ADD COLUMN source_operation_id uuid,
	ADD CONSTRAINT temporary_credentials_operation_generation_check
	CHECK (
		(
			source_operation_id IS NULL
			AND auth_generation = 0
		)
		OR (
			source_operation_id IS NOT NULL
			AND purpose = 'password_reset'
			AND auth_generation >= 1
		)
	);

DROP INDEX temporary_credentials_current_user_idx;

CREATE UNIQUE INDEX temporary_credentials_settled_current_idx
	ON temporary_credentials (user_id, purpose)
	WHERE state IN ('active', 'expired');

CREATE UNIQUE INDEX temporary_credentials_password_reset_pending_idx
	ON temporary_credentials (user_id, purpose)
	WHERE purpose = 'password_reset' AND state = 'pending_delivery';

CREATE TABLE password_reset_operations (
	operation_id uuid PRIMARY KEY,
	user_id uuid NOT NULL REFERENCES users (id),
	temporary_credential_id uuid NOT NULL,
	scope_hash bytea NOT NULL CHECK (octet_length(scope_hash) = 32),
	key_locator bytea NOT NULL CHECK (octet_length(key_locator) = 32),
	key_digest bytea NOT NULL CHECK (octet_length(key_digest) = 32),
	digest_key_version text NOT NULL CHECK (btrim(digest_key_version) <> ''),
	request_digest bytea NOT NULL CHECK (octet_length(request_digest) = 32),
	execution_generation bigint NOT NULL CHECK (execution_generation >= 1),
	request_kind text NOT NULL CHECK (request_kind IN ('initial_reset', 'resend')),
	predecessor_operation_id uuid,
	replaced_credential_id uuid,
	base_auth_generation bigint NOT NULL CHECK (base_auth_generation >= 0),
	target_auth_generation bigint NOT NULL CHECK (
		target_auth_generation = base_auth_generation + 1
	),
	settlement_state text NOT NULL CHECK (
		settlement_state IN (
			'pending',
			'temporary_active',
			'permanent_successor',
			'invalidated',
			'integrity_hold'
		)
	),
	settled_auth_generation bigint,
	successor_auth_generation bigint,
	settled_at timestamp with time zone,
	delivery_state text NOT NULL CHECK (
		delivery_state IN ('not_started', 'dispatching', 'accepted', 'rejected', 'unknown')
	),
	delivery_attempt_id uuid,
	delivery_attempted_at timestamp with time zone,
	delivery_result_recorded_at timestamp with time zone,
	state text NOT NULL CHECK (
		state IN (
			'prepared',
			'dispatching',
			'commit_unknown',
			'recovery_required',
			'converging',
			'succeeded',
			'invalidated',
			'failed'
		)
	),
	recovery_reason text CHECK (
		recovery_reason IS NULL OR btrim(recovery_reason) <> ''
	),
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	terminal_at timestamp with time zone,
	UNIQUE (operation_id, user_id),
	UNIQUE (operation_id, user_id, temporary_credential_id),
	UNIQUE (operation_id, user_id, target_auth_generation, temporary_credential_id),
	CONSTRAINT password_reset_operations_current_user_unique
		UNIQUE (operation_id, user_id, execution_generation),
	UNIQUE (
		operation_id,
		user_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	),
	UNIQUE (temporary_credential_id),
	UNIQUE (user_id, execution_generation),
	UNIQUE (scope_hash, key_locator, execution_generation),
	FOREIGN KEY (
		operation_id,
		user_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	)
	REFERENCES idempotency_records (
		password_reset_operation_id,
		password_reset_user_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	)
	DEFERRABLE INITIALLY DEFERRED,
	FOREIGN KEY (temporary_credential_id, user_id)
	REFERENCES temporary_credentials (id, user_id)
	DEFERRABLE INITIALLY DEFERRED,
	FOREIGN KEY (predecessor_operation_id, user_id)
	REFERENCES password_reset_operations (operation_id, user_id)
	DEFERRABLE INITIALLY DEFERRED,
	FOREIGN KEY (replaced_credential_id, user_id)
	REFERENCES temporary_credentials (id, user_id)
	DEFERRABLE INITIALLY DEFERRED,
	CHECK (
		(
			request_kind = 'initial_reset'
			AND predecessor_operation_id IS NULL
			AND replaced_credential_id IS NULL
		)
		OR (
			request_kind = 'resend'
			AND predecessor_operation_id IS NOT NULL
			AND replaced_credential_id IS NOT NULL
		)
	),
	CHECK (
		(
			settlement_state = 'pending'
			AND settled_auth_generation IS NULL
			AND successor_auth_generation IS NULL
			AND settled_at IS NULL
		)
		OR (
			settlement_state = 'temporary_active'
			AND settled_auth_generation = target_auth_generation
			AND successor_auth_generation IS NULL
			AND settled_at IS NOT NULL
		)
		OR (
			settlement_state = 'permanent_successor'
			AND settled_auth_generation = target_auth_generation
			AND successor_auth_generation = target_auth_generation + 1
			AND settled_at IS NOT NULL
		)
		OR (
			settlement_state IN ('invalidated', 'integrity_hold')
			AND settled_auth_generation IS NULL
			AND successor_auth_generation IS NULL
			AND settled_at IS NOT NULL
		)
	),
	CHECK (
		(
			delivery_state = 'not_started'
			AND delivery_attempt_id IS NULL
			AND delivery_attempted_at IS NULL
			AND delivery_result_recorded_at IS NULL
		)
		OR (
			delivery_state = 'dispatching'
			AND delivery_attempt_id IS NOT NULL
			AND delivery_attempted_at IS NOT NULL
			AND delivery_result_recorded_at IS NULL
		)
		OR (
			delivery_state IN ('accepted', 'rejected', 'unknown')
			AND delivery_attempt_id IS NOT NULL
			AND delivery_attempted_at IS NOT NULL
			AND delivery_result_recorded_at IS NOT NULL
		)
	),
	CHECK (
		(
			state IN ('succeeded', 'invalidated', 'failed')
			AND terminal_at IS NOT NULL
		)
		OR (
			state NOT IN ('succeeded', 'invalidated', 'failed')
			AND terminal_at IS NULL
		)
	),
	CONSTRAINT password_reset_operations_authority_check
	CHECK (
		(
			state = 'prepared'
			AND delivery_state = 'not_started'
			AND settlement_state = 'pending'
		)
		OR (
			state = 'dispatching'
			AND delivery_state = 'dispatching'
			AND settlement_state = 'pending'
		)
		OR (
			state = 'commit_unknown'
			AND delivery_state IN ('dispatching', 'unknown')
			AND settlement_state = 'pending'
		)
		OR (
			state IN ('recovery_required', 'converging')
			AND delivery_state = 'accepted'
			AND settlement_state = 'pending'
		)
		OR (
			state = 'succeeded'
			AND delivery_state = 'accepted'
			AND settlement_state IN ('temporary_active', 'permanent_successor')
		)
		OR (
			state = 'invalidated'
			AND delivery_state IN ('not_started', 'dispatching', 'unknown')
			AND settlement_state IN ('invalidated', 'integrity_hold')
		)
		OR (
			state = 'failed'
			AND delivery_state = 'rejected'
			AND settlement_state = 'invalidated'
		)
	),
	CHECK (updated_at >= created_at),
	CHECK (terminal_at IS NULL OR terminal_at >= created_at)
);

CREATE UNIQUE INDEX password_reset_operations_unresolved_user_idx
	ON password_reset_operations (user_id)
	WHERE state NOT IN ('succeeded', 'invalidated', 'failed');

CREATE INDEX password_reset_operations_gate_idx
	ON password_reset_operations (user_id, state);

CREATE INDEX password_reset_operations_predecessor_idx
	ON password_reset_operations (predecessor_operation_id);

CREATE INDEX password_reset_operations_replaced_credential_idx
	ON password_reset_operations (user_id, replaced_credential_id);

CREATE TABLE password_reset_recovery_jobs (
	operation_id uuid PRIMARY KEY
		REFERENCES password_reset_operations (operation_id)
		DEFERRABLE INITIALLY DEFERRED,
	status text NOT NULL
		CHECK (status IN ('ready', 'leased', 'retry_wait', 'succeeded', 'dead_letter')),
	attempt_count bigint NOT NULL CHECK (attempt_count >= 0),
	next_attempt_at timestamp with time zone NOT NULL,
	lease_owner text CHECK (lease_owner IS NULL OR btrim(lease_owner) <> ''),
	lease_expires_at timestamp with time zone,
	last_failure_code text CHECK (
		last_failure_code IS NULL OR btrim(last_failure_code) <> ''
	),
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	completed_at timestamp with time zone,
	CHECK (
		(
			status = 'leased'
			AND lease_owner IS NOT NULL
			AND lease_expires_at IS NOT NULL
		)
		OR (
			status <> 'leased'
			AND lease_owner IS NULL
			AND lease_expires_at IS NULL
		)
	),
	CHECK ((status = 'succeeded') = (completed_at IS NOT NULL)),
	CHECK (updated_at >= created_at),
	CHECK (next_attempt_at >= created_at),
	CHECK (lease_expires_at IS NULL OR lease_expires_at > updated_at),
	CHECK (completed_at IS NULL OR completed_at >= created_at)
);

ALTER TABLE password_reset_operations
	ADD CONSTRAINT password_reset_operations_recovery_job_fk
	FOREIGN KEY (operation_id)
	REFERENCES password_reset_recovery_jobs (operation_id)
	DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE idempotency_records
	ADD CONSTRAINT idempotency_records_password_reset_operation_fk
	FOREIGN KEY (
		password_reset_operation_id,
		password_reset_user_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	)
	REFERENCES password_reset_operations (
		operation_id,
		user_id,
		scope_hash,
		key_locator,
		key_digest,
		digest_key_version,
		request_digest,
		execution_generation
	)
	DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE users
	ADD CONSTRAINT users_password_reset_current_operation_fk
	FOREIGN KEY (
		password_reset_current_operation_id,
		id,
		password_reset_execution_generation
	)
	REFERENCES password_reset_operations (
		operation_id,
		user_id,
		execution_generation
	)
	DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX password_reset_recovery_jobs_candidate_idx
	ON password_reset_recovery_jobs (status, next_attempt_at, lease_expires_at);

ALTER TABLE temporary_credentials
	ADD CONSTRAINT temporary_credentials_source_operation_fk
	FOREIGN KEY (source_operation_id, user_id, auth_generation, id)
	REFERENCES password_reset_operations (
		operation_id,
		user_id,
		target_auth_generation,
		temporary_credential_id
	)
	DEFERRABLE INITIALLY DEFERRED;

CREATE UNIQUE INDEX temporary_credentials_source_operation_generation_idx
	ON temporary_credentials (source_operation_id, auth_generation)
	WHERE source_operation_id IS NOT NULL;

ALTER TABLE users
	ADD CONSTRAINT users_permanent_password_source_fk
	FOREIGN KEY (
		permanent_password_source_operation_id,
		id,
		permanent_password_source_credential_id
	)
	REFERENCES password_reset_operations (
		operation_id,
		user_id,
		temporary_credential_id
	)
	DEFERRABLE INITIALLY DEFERRED,
	ADD CONSTRAINT users_auth_integrity_hold_operation_fk
	FOREIGN KEY (auth_integrity_hold_operation_id, id)
	REFERENCES password_reset_operations (operation_id, user_id)
	DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE sessions
	ADD CONSTRAINT sessions_password_reset_lineage_fk
	FOREIGN KEY (lineage_operation_id, user_id, lineage_credential_id)
	REFERENCES password_reset_operations (
		operation_id,
		user_id,
		temporary_credential_id
	)
	DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX sessions_user_auth_generation_idx
	ON sessions (user_id, auth_generation);

CREATE INDEX sessions_password_reset_lineage_idx
	ON sessions (lineage_operation_id, lineage_credential_id);

CREATE FUNCTION enforce_user_auth_generation_monotonic()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF NEW.auth_generation < OLD.auth_generation
		OR NEW.password_reset_execution_generation
			< OLD.password_reset_execution_generation THEN
		RAISE EXCEPTION 'user authentication generation cannot move backwards'
			USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;

CREATE FUNCTION enforce_password_reset_user_counter_complete()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
	operation_count bigint;
BEGIN
	IF NEW.password_reset_execution_generation = 0 THEN
		RETURN NULL;
	END IF;
	SELECT count(*)
	INTO operation_count
	FROM password_reset_operations
	WHERE user_id = NEW.id
		AND execution_generation BETWEEN 1
			AND NEW.password_reset_execution_generation;
	IF operation_count <> NEW.password_reset_execution_generation THEN
		RAISE EXCEPTION 'password reset user counter has an operation gap'
			USING ERRCODE = '55000';
	END IF;
	RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER password_reset_user_counter_complete
	AFTER INSERT OR UPDATE ON users
	DEFERRABLE INITIALLY DEFERRED
	FOR EACH ROW
	EXECUTE FUNCTION enforce_password_reset_user_counter_complete();

CREATE FUNCTION enforce_password_reset_operation_counter_complete()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
	target_user_id uuid;
	user_counter bigint;
BEGIN
	IF TG_OP = 'DELETE' THEN
		target_user_id := OLD.user_id;
	ELSE
		target_user_id := NEW.user_id;
	END IF;
	SELECT password_reset_execution_generation
	INTO user_counter
	FROM users
	WHERE id = target_user_id;
	IF FOUND AND EXISTS (
		SELECT 1
		FROM password_reset_operations
		WHERE user_id = target_user_id
			AND execution_generation > user_counter
	) THEN
		RAISE EXCEPTION 'password reset operation exceeds the user counter'
			USING ERRCODE = '55000';
	END IF;
	RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER password_reset_operation_counter_complete
	AFTER INSERT OR UPDATE OR DELETE ON password_reset_operations
	DEFERRABLE INITIALLY DEFERRED
	FOR EACH ROW
	EXECUTE FUNCTION enforce_password_reset_operation_counter_complete();

CREATE TRIGGER users_auth_generation_monotonic
	BEFORE UPDATE ON users
	FOR EACH ROW
	EXECUTE FUNCTION enforce_user_auth_generation_monotonic();

CREATE FUNCTION enforce_password_reset_operation_binding_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	IF NEW.operation_id IS DISTINCT FROM OLD.operation_id
		OR NEW.user_id IS DISTINCT FROM OLD.user_id
		OR NEW.temporary_credential_id IS DISTINCT FROM OLD.temporary_credential_id
		OR NEW.scope_hash IS DISTINCT FROM OLD.scope_hash
		OR NEW.key_locator IS DISTINCT FROM OLD.key_locator
		OR NEW.key_digest IS DISTINCT FROM OLD.key_digest
		OR NEW.digest_key_version IS DISTINCT FROM OLD.digest_key_version
		OR NEW.request_digest IS DISTINCT FROM OLD.request_digest
		OR NEW.execution_generation IS DISTINCT FROM OLD.execution_generation
		OR NEW.request_kind IS DISTINCT FROM OLD.request_kind
		OR NEW.predecessor_operation_id IS DISTINCT FROM OLD.predecessor_operation_id
		OR NEW.replaced_credential_id IS DISTINCT FROM OLD.replaced_credential_id
		OR NEW.base_auth_generation IS DISTINCT FROM OLD.base_auth_generation
		OR NEW.target_auth_generation IS DISTINCT FROM OLD.target_auth_generation
		OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
		RAISE EXCEPTION 'password reset operation binding is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.delivery_attempt_id IS NOT NULL
		AND (
			NEW.delivery_attempt_id IS DISTINCT FROM OLD.delivery_attempt_id
			OR NEW.delivery_attempted_at IS DISTINCT FROM OLD.delivery_attempted_at
		) THEN
		RAISE EXCEPTION 'password reset delivery attempt is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.delivery_state IS DISTINCT FROM NEW.delivery_state
		AND NOT (
			(OLD.delivery_state = 'not_started' AND NEW.delivery_state = 'dispatching')
			OR (
				OLD.delivery_state = 'dispatching'
				AND NEW.delivery_state IN ('accepted', 'rejected', 'unknown')
			)
			OR (
				OLD.delivery_state = 'unknown'
				AND NEW.delivery_state = 'accepted'
			)
		) THEN
		RAISE EXCEPTION 'password reset delivery cannot move backwards'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.settlement_state IS DISTINCT FROM NEW.settlement_state
		AND NOT (
			OLD.settlement_state = 'pending'
			AND NEW.settlement_state IN (
				'temporary_active',
				'permanent_successor',
				'invalidated',
				'integrity_hold'
			)
			OR (
				OLD.settlement_state = 'temporary_active'
				AND NEW.settlement_state = 'permanent_successor'
			)
		) THEN
		RAISE EXCEPTION 'password reset settlement cannot move backwards'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.state IS DISTINCT FROM NEW.state
		AND NOT (
			(OLD.state = 'prepared' AND NEW.state IN ('dispatching', 'invalidated'))
			OR (
				OLD.state = 'dispatching'
				AND NEW.state IN (
					'commit_unknown',
					'recovery_required',
					'invalidated',
					'failed'
				)
			)
			OR (
				OLD.state = 'commit_unknown'
				AND NEW.state IN ('recovery_required', 'invalidated')
			)
			OR (
				OLD.state = 'recovery_required'
				AND NEW.state = 'converging'
			)
			OR (
				OLD.state = 'converging'
				AND NEW.state IN (
					'recovery_required',
					'succeeded',
					'invalidated',
					'failed'
				)
			)
		) THEN
		RAISE EXCEPTION 'password reset operation state transition is invalid'
			USING ERRCODE = '55000';
	END IF;
	IF OLD.state IN ('succeeded', 'invalidated', 'failed')
		AND (
			NEW.state IS DISTINCT FROM OLD.state
			OR NEW.delivery_state IS DISTINCT FROM OLD.delivery_state
			OR NEW.delivery_attempt_id IS DISTINCT FROM OLD.delivery_attempt_id
			OR NEW.delivery_attempted_at IS DISTINCT FROM OLD.delivery_attempted_at
			OR NEW.delivery_result_recorded_at
				IS DISTINCT FROM OLD.delivery_result_recorded_at
			OR NEW.terminal_at IS DISTINCT FROM OLD.terminal_at
			OR NEW.recovery_reason IS DISTINCT FROM OLD.recovery_reason
			OR (
				(
					NEW.settlement_state IS DISTINCT FROM OLD.settlement_state
					OR NEW.settled_auth_generation
						IS DISTINCT FROM OLD.settled_auth_generation
					OR NEW.successor_auth_generation
						IS DISTINCT FROM OLD.successor_auth_generation
					OR NEW.settled_at IS DISTINCT FROM OLD.settled_at
				)
				AND NOT (
					OLD.state = 'succeeded'
					AND OLD.settlement_state = 'temporary_active'
					AND NEW.settlement_state = 'permanent_successor'
				)
			)
		) THEN
		RAISE EXCEPTION 'password reset terminal evidence is immutable'
			USING ERRCODE = '55000';
	END IF;
	IF NEW.updated_at < OLD.updated_at THEN
		RAISE EXCEPTION 'password reset operation time cannot move backwards'
			USING ERRCODE = '55000';
	END IF;
	RETURN NEW;
END;
$$;

CREATE TRIGGER password_reset_operations_binding_immutable
	BEFORE UPDATE ON password_reset_operations
	FOR EACH ROW
	EXECUTE FUNCTION enforce_password_reset_operation_binding_immutable();

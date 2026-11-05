CREATE TABLE processing_gate_records (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	region_ref text NOT NULL CHECK (btrim(region_ref) <> ''),
	controller_ref text NOT NULL CHECK (btrim(controller_ref) <> ''),
	legal_basis_ref text NOT NULL CHECK (btrim(legal_basis_ref) <> ''),
	smtp_processor_ref text NOT NULL CHECK (btrim(smtp_processor_ref) <> ''),
	effective_at timestamp with time zone NOT NULL,
	recorded_by text NOT NULL CHECK (btrim(recorded_by) <> '')
);

CREATE INDEX processing_gate_records_current_idx
	ON processing_gate_records (effective_at DESC, id DESC);

CREATE TABLE audit_events (
	id uuid PRIMARY KEY,
	occurred_at timestamp with time zone NOT NULL,
	actor_kind text NOT NULL CHECK (actor_kind IN ('anonymous', 'user', 'system')),
	actor_user_id uuid,
	actor_system text,
	actor_role text,
	action text NOT NULL CHECK (btrim(action) <> ''),
	target_type text NOT NULL CHECK (btrim(target_type) <> ''),
	target_id text NOT NULL CHECK (btrim(target_id) <> ''),
	outcome text NOT NULL CHECK (btrim(outcome) <> ''),
	error_code text,
	trace_id text NOT NULL CHECK (btrim(trace_id) <> ''),
	source_fingerprint text,
	metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
	CHECK (
		(actor_kind = 'anonymous' AND actor_user_id IS NULL AND actor_system IS NULL)
		OR (actor_kind = 'user' AND actor_user_id IS NOT NULL AND actor_system IS NULL)
		OR (actor_kind = 'system' AND actor_user_id IS NULL AND btrim(actor_system) <> '')
	)
);

CREATE INDEX audit_events_target_idx
	ON audit_events (target_type, target_id, occurred_at DESC, id);

CREATE INDEX audit_events_trace_idx
	ON audit_events (trace_id, occurred_at DESC, id);

CREATE FUNCTION reject_append_only_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
	RAISE EXCEPTION 'table % is append-only', TG_TABLE_NAME
		USING ERRCODE = '55000';
END;
$$;

CREATE TRIGGER processing_gate_records_append_only
	BEFORE UPDATE OR DELETE ON processing_gate_records
	FOR EACH STATEMENT
	EXECUTE FUNCTION reject_append_only_mutation();

CREATE TRIGGER audit_events_append_only
	BEFORE UPDATE OR DELETE ON audit_events
	FOR EACH STATEMENT
	EXECUTE FUNCTION reject_append_only_mutation();

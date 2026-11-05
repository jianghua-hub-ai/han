DO $$
DECLARE
	legacy_constraint_name text;
	legacy_constraint_count integer;
BEGIN
	SELECT count(*), min(conname)
	INTO legacy_constraint_count, legacy_constraint_name
	FROM pg_constraint
	WHERE conrelid = 'idempotency_records'::regclass
		AND contype = 'c'
		AND pg_get_constraintdef(oid) LIKE '%response_snapshot%'
		AND pg_get_constraintdef(oid) LIKE '%outcome%'
		AND pg_get_constraintdef(oid) NOT LIKE '%schema_version%';

	IF legacy_constraint_count <> 1 THEN
		RAISE EXCEPTION 'expected one legacy response_snapshot constraint';
	END IF;

	EXECUTE format(
		'ALTER TABLE idempotency_records DROP CONSTRAINT %I',
		legacy_constraint_name
	);
END;
$$;

ALTER TABLE idempotency_records
	ADD CONSTRAINT idempotency_records_response_snapshot_v1_check
	CHECK (
		response_snapshot IS NULL
		OR (
			NOT response_snapshot ? 'schema_version'
			AND jsonb_typeof(response_snapshot) = 'object'
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
		OR (
			jsonb_typeof(response_snapshot) = 'object'
			AND (response_snapshot ? 'schema_version')
			AND (response_snapshot ? 'http_status')
			AND (response_snapshot ? 'result_code')
			AND (response_snapshot ? 'body')
			AND response_snapshot ->> 'schema_version'
				= 'identity-administrator-create-response/v1'
			AND jsonb_typeof(response_snapshot -> 'schema_version') = 'string'
			AND jsonb_typeof(response_snapshot -> 'http_status') = 'number'
			AND jsonb_typeof(response_snapshot -> 'result_code') = 'string'
			AND jsonb_typeof(response_snapshot -> 'body') = 'object'
			AND response_snapshot - ARRAY[
				'schema_version',
				'http_status',
				'result_code',
				'body'
			] = '{}'::jsonb
			AND (
				(
					(response_snapshot -> 'body') ? 'id'
					AND (response_snapshot -> 'body') ? 'username'
					AND (response_snapshot -> 'body') ? 'role'
					AND (response_snapshot -> 'body') ? 'status'
					AND (response_snapshot -> 'body') ? 'must_change_password'
					AND (response_snapshot -> 'body') ? 'must_acknowledge_privacy'
					AND (response_snapshot -> 'body') ?
						'temporary_credential_expires_at'
					AND (response_snapshot -> 'body') ? 'delivery_status'
					AND response_snapshot -> 'http_status' = '201'::jsonb
					AND response_snapshot ->> 'result_code' = 'created'
					AND jsonb_typeof(response_snapshot -> 'body' -> 'id') = 'string'
					AND btrim(response_snapshot -> 'body' ->> 'id') <> ''
					AND jsonb_typeof(response_snapshot -> 'body' -> 'username') = 'string'
					AND btrim(response_snapshot -> 'body' ->> 'username') <> ''
					AND response_snapshot -> 'body' ->> 'role' = 'admin'
					AND jsonb_typeof(response_snapshot -> 'body' -> 'role') = 'string'
					AND response_snapshot -> 'body' ->> 'status' = 'active'
					AND jsonb_typeof(response_snapshot -> 'body' -> 'status') = 'string'
					AND response_snapshot -> 'body' -> 'must_change_password' = 'true'::jsonb
					AND response_snapshot -> 'body' -> 'must_acknowledge_privacy' = 'true'::jsonb
					AND jsonb_typeof(
						response_snapshot -> 'body' -> 'temporary_credential_expires_at'
					) = 'string'
					AND btrim(
						response_snapshot -> 'body' ->> 'temporary_credential_expires_at'
					) <> ''
					AND response_snapshot -> 'body' ->> 'delivery_status' = 'accepted'
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
				OR (
					(response_snapshot -> 'body') ? 'code'
					AND (response_snapshot -> 'body') ? 'message'
					AND (response_snapshot -> 'body') ? 'field_errors'
					AND (
						(
							response_snapshot -> 'http_status' = '409'::jsonb
							AND response_snapshot ->> 'result_code'
								= 'identity.identifier_conflict'
							AND response_snapshot -> 'body' ->> 'code'
								= 'identity.identifier_conflict'
							AND response_snapshot -> 'body' ->> 'message'
								= '身份标识已被使用'
						)
						OR (
							response_snapshot -> 'http_status' = '503'::jsonb
							AND response_snapshot ->> 'result_code'
								= 'identity.email_delivery_failed'
							AND response_snapshot -> 'body' ->> 'code'
								= 'identity.email_delivery_failed'
							AND response_snapshot -> 'body' ->> 'message'
								= '邮件暂未受理'
						)
					)
				)
				AND jsonb_typeof(response_snapshot -> 'body' -> 'code') = 'string'
				AND jsonb_typeof(response_snapshot -> 'body' -> 'message') = 'string'
				AND jsonb_typeof(response_snapshot -> 'body' -> 'field_errors') = 'array'
				AND response_snapshot -> 'body' -> 'field_errors' = '[]'::jsonb
				AND (response_snapshot -> 'body') - ARRAY[
					'code',
					'message',
					'field_errors'
				] = '{}'::jsonb
			)
		)
	);

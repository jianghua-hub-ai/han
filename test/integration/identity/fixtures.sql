-- S1.6 identity PostgreSQL transaction/lock/concurrency fixtures.
--
-- The fixtures below are applied by the Go harness on an isolated schema that
-- already carries the full migration chain (0001..0011). They only create
-- synthetic rows; no real user data, credentials or addresses are used.
--
-- The harness substitutes the `:name` tokens with validated literals before
-- execution, so every statement is a complete, separately executable command
-- and no placeholder is left untyped.

UPDATE users
SET role = 'member',
	status = 'active',
	credential_mode = 'permanent',
	permanent_password_digest = :initial_digest,
	permanent_password_generation = 0
WHERE id = :member;

UPDATE users
SET role = 'member',
	status = 'active',
	credential_mode = 'permanent',
	permanent_password_digest = :initial_digest,
	permanent_password_generation = 0
WHERE id = :other_member;

UPDATE users
SET role = 'admin',
	is_initial_admin = true,
	status = 'active',
	credential_mode = 'permanent',
	permanent_password_digest = :initial_digest,
	permanent_password_generation = 0,
	onboarding_completed_at = :now
WHERE id = :initial_admin;

UPDATE users
SET role = 'admin',
	is_initial_admin = false,
	status = 'active',
	credential_mode = 'permanent',
	permanent_password_digest = :initial_digest,
	permanent_password_generation = 0,
	onboarding_completed_at = :now
WHERE id = :other_admin;

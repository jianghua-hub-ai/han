CREATE INDEX users_role_status_created_at_id_idx
	ON users (role, status, created_at DESC, id DESC);

CREATE INDEX users_created_at_id_idx
	ON users (created_at DESC, id DESC);

ALTER TABLE deletion_replay_runs
	ADD COLUMN attempt_generation bigint NOT NULL DEFAULT 0,
	ADD COLUMN attempt_token text;

ALTER TABLE deletion_replay_runs
	ADD CONSTRAINT deletion_replay_runs_attempt_generation_non_negative
	CHECK (attempt_generation >= 0);

ALTER TABLE deletion_replay_runs
	ADD CONSTRAINT deletion_replay_runs_attempt_consistency
	CHECK (
		(attempt_generation = 0 AND attempt_token IS NULL)
		OR (
			attempt_generation > 0
			AND attempt_token IS NOT NULL
			AND btrim(attempt_token) <> ''
		)
	);

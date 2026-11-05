-- Forward-only publication metadata. Existing evidence bytes/semantics stay unchanged.
ALTER TABLE deletion_replay_evidence
    ADD CONSTRAINT deletion_replay_evidence_publication_binding_key
    UNIQUE (run_id, namespace, backup_ref, evidence_ref, evidence_digest);

CREATE TABLE article_replay_publications (
    publication_ref text PRIMARY KEY CHECK (btrim(publication_ref) <> ''),
    publication_version text NOT NULL
        CHECK (publication_version = 'article-replay-publication/v1'),
    source_kind text NOT NULL
        CHECK (source_kind IN ('atomic_current_v1', 'verified_legacy_import_v1')),
    run_id text NOT NULL CHECK (btrim(run_id) <> ''),
    namespace text NOT NULL CHECK (namespace = 'article-deletion/v1'),
    backup_ref text NOT NULL CHECK (btrim(backup_ref) <> ''),
    evidence_ref text NOT NULL CHECK (btrim(evidence_ref) <> ''),
    evidence_digest text NOT NULL CHECK (evidence_digest ~ '^sha256:[0-9a-f]{64}$'),
    attempt_generation bigint NOT NULL CHECK (attempt_generation >= 0),
    attempt_token text,
    journal_id text NOT NULL CHECK (btrim(journal_id) <> ''),
    journal_epoch bigint NOT NULL CHECK (journal_epoch > 0),
    journal_head_sequence bigint NOT NULL CHECK (journal_head_sequence >= 0),
    journal_head_hash text NOT NULL,
    evidence_completed_at timestamptz NOT NULL,
    published_at timestamptz NOT NULL,
    publication_digest text NOT NULL CHECK (publication_digest ~ '^sha256:[0-9a-f]{64}$'),
    source_ref text NOT NULL CHECK (btrim(source_ref) <> ''),
    source_digest text NOT NULL CHECK (source_digest ~ '^sha256:[0-9a-f]{64}$'),
    CONSTRAINT article_replay_publications_evidence_fk
        FOREIGN KEY (run_id, namespace, backup_ref, evidence_ref, evidence_digest)
        REFERENCES deletion_replay_evidence
            (run_id, namespace, backup_ref, evidence_ref, evidence_digest)
        ON DELETE RESTRICT,
    CONSTRAINT article_replay_publications_generation_token_check CHECK (
        (attempt_generation = 0 AND attempt_token IS NULL)
        OR (attempt_generation > 0 AND attempt_token IS NOT NULL
            AND btrim(attempt_token) <> '')
    ),
    CONSTRAINT article_replay_publications_time_check
        CHECK (published_at >= evidence_completed_at),
    CONSTRAINT article_replay_publications_generation_evidence_key
        UNIQUE (run_id, namespace, attempt_generation, evidence_ref, source_kind)
);

CREATE INDEX article_replay_publications_evidence_idx
    ON article_replay_publications (run_id, namespace, evidence_ref);

CREATE TRIGGER article_replay_publications_append_only
    BEFORE UPDATE OR DELETE ON article_replay_publications
    FOR EACH STATEMENT
    EXECUTE FUNCTION reject_append_only_mutation();

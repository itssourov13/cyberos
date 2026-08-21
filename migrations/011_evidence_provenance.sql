-- Phase 8A — durable evidence + provenance
BEGIN;
CREATE TABLE IF NOT EXISTS evidence (
    id              TEXT PRIMARY KEY,
    task_id         TEXT NOT NULL,
    grant_id        TEXT NOT NULL,
    capability_id   TEXT NOT NULL,
    principal_id    TEXT NOT NULL,
    principal_type  TEXT NOT NULL DEFAULT '',
    project_id      TEXT NOT NULL,
    scope_id        TEXT NOT NULL,
    content_type    TEXT NOT NULL DEFAULT '',
    digest          TEXT NOT NULL DEFAULT '',
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    produced_at     TIMESTAMPTZ NOT NULL,
    correlation_id  TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_evidence_task ON evidence(task_id);
CREATE INDEX IF NOT EXISTS idx_evidence_project ON evidence(project_id);
CREATE INDEX IF NOT EXISTS idx_evidence_grant ON evidence(grant_id);

CREATE TABLE IF NOT EXISTS provenance (
    subject_type    TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    principal_id    TEXT NOT NULL,
    principal_type  TEXT NOT NULL DEFAULT '',
    project_id      TEXT NOT NULL,
    scope_id        TEXT NOT NULL,
    task_id         TEXT NOT NULL,
    capability_id   TEXT NOT NULL,
    grant_id        TEXT NOT NULL DEFAULT '',
    executor_id     TEXT NOT NULL DEFAULT '',
    produced_at     TIMESTAMPTZ NOT NULL,
    correlation_id  TEXT NOT NULL DEFAULT '',
    parent_ids      JSONB NOT NULL DEFAULT '[]'::jsonb,
    PRIMARY KEY (subject_type, subject_id)
);
CREATE INDEX IF NOT EXISTS idx_provenance_project ON provenance(project_id);
CREATE INDEX IF NOT EXISTS idx_provenance_task ON provenance(task_id);

INSERT INTO schema_migrations (version) VALUES ('011_evidence_provenance')
ON CONFLICT (version) DO NOTHING;
COMMIT;

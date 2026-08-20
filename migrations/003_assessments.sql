-- Phase 4A — durable assessments
BEGIN;

CREATE TABLE IF NOT EXISTS assessments (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    scope_id      TEXT NOT NULL,
    principal_id  TEXT NOT NULL,
    title         TEXT NOT NULL,
    target        TEXT NOT NULL DEFAULT '',
    state         TEXT NOT NULL,
    steps         JSONB NOT NULL DEFAULT '[]'::jsonb,
    step_results  JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assessments_project ON assessments(project_id);
CREATE INDEX IF NOT EXISTS idx_assessments_scope ON assessments(scope_id);
CREATE INDEX IF NOT EXISTS idx_assessments_state ON assessments(state);

INSERT INTO schema_migrations (version) VALUES ('003_assessments')
ON CONFLICT (version) DO NOTHING;

COMMIT;

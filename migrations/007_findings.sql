-- Phase 6D — durable findings (optional wire-up)
BEGIN;
CREATE TABLE IF NOT EXISTS findings (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL,
    scope_id        TEXT NOT NULL,
    task_id         TEXT NOT NULL,
    assessment_id   TEXT NOT NULL DEFAULT '',
    execution_id    TEXT NOT NULL DEFAULT '',
    principal_id    TEXT NOT NULL DEFAULT '',
    tool_id         TEXT NOT NULL DEFAULT '',
    capability_id   TEXT NOT NULL DEFAULT '',
    severity        TEXT NOT NULL,
    confidence      TEXT NOT NULL DEFAULT 'MEDIUM',
    status          TEXT NOT NULL DEFAULT 'NEW',
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    target          TEXT NOT NULL DEFAULT '',
    location        TEXT NOT NULL DEFAULT '',
    fingerprint     TEXT NOT NULL,
    evidence_ids    JSONB NOT NULL DEFAULT '[]'::jsonb,
    first_seen      TIMESTAMPTZ NOT NULL,
    last_seen       TIMESTAMPTZ NOT NULL,
    produced_at     TIMESTAMPTZ NOT NULL,
    correlation_id  TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_findings_fingerprint ON findings(fingerprint);
CREATE INDEX IF NOT EXISTS idx_findings_project ON findings(project_id);
CREATE INDEX IF NOT EXISTS idx_findings_assessment ON findings(assessment_id);
INSERT INTO schema_migrations (version) VALUES ('007_findings')
ON CONFLICT (version) DO NOTHING;
COMMIT;

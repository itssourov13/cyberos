BEGIN;
-- Durable, attributable history of case status transitions (Phase 24D-1).
CREATE TABLE IF NOT EXISTS case_status_history (
    id          BIGSERIAL PRIMARY KEY,
    case_id     TEXT        NOT NULL REFERENCES security_cases(id) ON DELETE CASCADE,
    tenant_id   TEXT        NOT NULL DEFAULT '',
    project_id  TEXT        NOT NULL,
    from_status TEXT        NOT NULL,
    to_status   TEXT        NOT NULL,
    actor       TEXT        NOT NULL,
    reason      TEXT        NOT NULL DEFAULT '',
    changed_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS case_status_history_case_idx ON case_status_history (case_id, id);
CREATE INDEX IF NOT EXISTS security_cases_project_idx ON security_cases (project_id, updated_at DESC);
INSERT INTO schema_migrations (version) VALUES ('022_case_status_history') ON CONFLICT (version) DO NOTHING;
COMMIT;

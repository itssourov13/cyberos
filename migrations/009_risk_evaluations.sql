BEGIN;
CREATE TABLE IF NOT EXISTS risk_evaluations (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    assessment_id TEXT NOT NULL DEFAULT '',
    finding_id TEXT NOT NULL,
    asset_id TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL,
    confidence TEXT NOT NULL,
    asset_criticality TEXT NOT NULL,
    exposure TEXT NOT NULL,
    score INT NOT NULL,
    priority TEXT NOT NULL,
    factors JSONB NOT NULL DEFAULT '{}'::jsonb,
    model_version TEXT NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_risk_finding ON risk_evaluations(finding_id);
CREATE INDEX IF NOT EXISTS idx_risk_assessment ON risk_evaluations(assessment_id);
INSERT INTO schema_migrations (version) VALUES ('009_risk_evaluations') ON CONFLICT (version) DO NOTHING;
COMMIT;

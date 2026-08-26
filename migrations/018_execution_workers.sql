-- Phase 17C — Worker identity registry (no execution, no Grants, no secrets plaintext)
BEGIN;
CREATE TABLE IF NOT EXISTS execution_workers (
    id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    host_id TEXT NOT NULL,
    status TEXT NOT NULL,
    credential_ref TEXT NOT NULL DEFAULT '',
    cert_fingerprint TEXT NOT NULL DEFAULT '',
    capabilities_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (project_id, id),
    UNIQUE (project_id, host_id)
);
CREATE INDEX IF NOT EXISTS idx_execution_workers_status ON execution_workers (project_id, status);
INSERT INTO schema_migrations (version) VALUES ('018_execution_workers') ON CONFLICT (version) DO NOTHING;
COMMIT;

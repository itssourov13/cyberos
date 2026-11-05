BEGIN;
-- Server-side registry of issued execution grants (Phase 24D). The Broker claims
-- a grant by a single conditional UPDATE, so one-time use, revocation and expiry
-- are enforced atomically across every API/worker process. A grant that is not
-- in this table was not issued by this control plane and cannot execute.
CREATE TABLE IF NOT EXISTS execution_grant_ledger (
    grant_id       TEXT        PRIMARY KEY,
    nonce          TEXT        NOT NULL UNIQUE,
    principal_id   TEXT        NOT NULL,
    project_id     TEXT        NOT NULL,
    scope_id       TEXT        NOT NULL,
    task_id        TEXT        NOT NULL DEFAULT '',
    capability_id  TEXT        NOT NULL,
    correlation_id TEXT        NOT NULL DEFAULT '',
    issued_at      TIMESTAMPTZ NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    state          TEXT        NOT NULL DEFAULT 'ISSUED'
                   CHECK (state IN ('ISSUED','CLAIMED','COMPLETED','FAILED','REVOKED')),
    claimed_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    result_status  TEXT        NOT NULL DEFAULT '',
    revoked_at     TIMESTAMPTZ,
    revoked_by     TEXT        NOT NULL DEFAULT '',
    revoke_reason  TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS execution_grant_ledger_expires_idx ON execution_grant_ledger (expires_at);
CREATE INDEX IF NOT EXISTS execution_grant_ledger_project_idx ON execution_grant_ledger (project_id, issued_at DESC);
INSERT INTO schema_migrations (version) VALUES ('023_execution_grant_ledger') ON CONFLICT (version) DO NOTHING;
COMMIT;

-- CYBEROS Phase 3B — Durable task state extensions
BEGIN;

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS depends_on JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS schedule JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS retry JSONB;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS timeout_ns BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS deadline TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS worker_id TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS lease_until TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_tasks_state_updated ON tasks(state, updated_at);
CREATE INDEX IF NOT EXISTS idx_tasks_lease ON tasks(lease_until) WHERE lease_until IS NOT NULL;

-- Runner checkpoint / recovery journal (append-only style)
CREATE TABLE IF NOT EXISTS task_runner_events (
    id           BIGSERIAL PRIMARY KEY,
    task_id      TEXT NOT NULL REFERENCES tasks(id),
    event_type   TEXT NOT NULL,
    worker_id    TEXT NOT NULL DEFAULT '',
    detail       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_runner_events_task ON task_runner_events(task_id, created_at);

INSERT INTO schema_migrations (version) VALUES ('002_task_durable')
ON CONFLICT (version) DO NOTHING;

COMMIT;

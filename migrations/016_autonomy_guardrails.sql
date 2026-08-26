-- Phase 16C — AutonomyRun LLM/cooldown/no-progress counters (no secrets/grants)
BEGIN;
ALTER TABLE autonomy_runs ADD COLUMN IF NOT EXISTS llm_calls_used INT NOT NULL DEFAULT 0;
ALTER TABLE autonomy_runs ADD COLUMN IF NOT EXISTS no_progress_count INT NOT NULL DEFAULT 0;
ALTER TABLE autonomy_runs ADD COLUMN IF NOT EXISTS last_progress_key TEXT NOT NULL DEFAULT '';
ALTER TABLE autonomy_runs ADD COLUMN IF NOT EXISTS next_allowed_at TIMESTAMPTZ;
INSERT INTO schema_migrations (version) VALUES ('016_autonomy_guardrails') ON CONFLICT (version) DO NOTHING;
COMMIT;

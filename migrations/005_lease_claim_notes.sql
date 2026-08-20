-- Phase 4C — durable atomic claim uses tasks.worker_id + tasks.lease_until (from 002).
-- No schema change required; index idx_tasks_lease already present.
BEGIN;
INSERT INTO schema_migrations (version) VALUES ('005_lease_claim')
ON CONFLICT (version) DO NOTHING;
COMMIT;

# Handoff

**Release `v0.15.0` — Phase 15 frozen.**

Do not start Phase 16 without explicit approval.

Spine: Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor

Events/outbox deliver to EventAutonomyProcessor only (observe→plan→policy eligibility).
Handoff → Orchestrator for execution; no AI/Event/Policy Grant authority.
Live PG: SKIP without DSN.

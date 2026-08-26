# Handoff

**Release `v0.14.0` — Phase 14 frozen.**

Do not start Phase 15 without explicit approval.

Security spine:
Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor

Autonomy is policy-gated eligibility + handoff to Orchestrator.
MayAdvance ≠ Grant. AutonomyRun does not store Grants.
AutonomyRun store: memory + PG migration 014; live PG SKIP without DSN.

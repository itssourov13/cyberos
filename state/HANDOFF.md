# Handoff

**Release `v0.9.0` — Phase 9 frozen.**

Do not start Phase 10 without explicit approval.

Security spine unchanged: Scope → Auth → CP → Grant → Broker → Sandbox → Executor.

Process execution is fail-closed by default. Production: `CYBEROS_ENV=production` + `CYBEROS_DATABASE_URL` → `sdk.NewProduction`.

Live PG integration tests require a running database; SKIP must not be reported as durability verified.

# Handoff

**Release `v0.8.0` — Phase 8 frozen.**

Do not start Phase 9 without explicit approval.

Production: `CYBEROS_ENV=production` + `CYBEROS_DATABASE_URL` → `sdk.NewProduction`; mandatory auth; bootstrap via existing keys or `CYBEROS_BOOTSTRAP_API_KEY`.

Live PG integration tests require a running database; otherwise they SKIP (not a durability pass).

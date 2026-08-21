# Handoff

**Release `v0.7.0` — Phase 7 frozen.**

Do not start Phase 8 or modify the security spine without explicit approval.

Production: `CYBEROS_ENV=production` + `CYBEROS_DATABASE_URL` → `sdk.NewProduction`.  
Development: `sdk.NewInMemory()`.

Live PostgreSQL integration tests require a running database; otherwise they SKIP.

# CYBEROS Project State

**Last Updated:** 2026-08-27

## Current
**Phase 15 FROZEN** at tag **`v0.15.0`**.

## Next
Awaiting explicit approval for Phase 16.


## Frontend (Phase 24C accepted; 24D partial)
Clean-slate React/TS/Vite console in `frontend/`. Live adapter + real sign-in against the Go API are the default; preview data is opt-in (`VITE_CYBEROS_API_MODE=preview`). Embedded in the Go binary via `make web`. Details: `docs/FRONTEND_REBUILD.md`, `docs/PHASE24D.md`.

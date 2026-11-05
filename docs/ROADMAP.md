# CYBEROS Roadmap

Status labels: **DONE** · **PARTIAL** · **DEFERRED** · **UNVERIFIED**

| Phase | Topic | Status |
|-------|-------|--------|
| 1–7 | Foundation → platform freeze | **DONE** |
| 8–9 | Durability, tools, process policy | **DONE** |
| 10–17 | Workflows, autonomy isolation, hardening | **DONE** (see phase docs) |
| 18 | Pack integrity / signature | **DONE** |
| 19 | Grant pack bind, durable trust, audit | **DONE** |
| 20 | ClaimAtomic durable claiming | **DONE** (unit); live multi-node **UNVERIFIED** |
| 21 | TrustAdmin | **DONE** |
| 22 | mTLS, result auth, revoke, regression | **DONE** (unit); live fleet **UNVERIFIED** |
| 23A | Security/architecture audit | **DONE** |
| 23B | CLI operator surfaces | **DONE** |
| 23C | Docs + live integration skip harness | **DONE** (committed HEAD family) |
| 24A | Web UI design specification | **DONE** (design doc) |
| 24B | Frontend Parts 1–3 + D1–D5 (old MOCK UI) | **SUPERSEDED** — old demo UI retired by 24C |
| 24C | Clean-slate frontend (React/TS/Vite) on preview data | **ACCEPTED by owner** |
| 24D | Real `/api/v1` integration, authentication, remaining backend gaps | **PARTIAL** — auth/sessions, grant provenance, isolation, live read adapter, production serving done and verified; see `docs/PHASE24D.md` for remaining gaps |
| 25+ | Marketplace / JARVIS / further product | **DEFERRED** |

## Live verification

PostgreSQL, multi-worker, and mTLS fleet checks **SKIP** unless `DATABASE_URL` / cert env is set. Absence of env must never be reported as production PASS.

## Next recommended

1. Keep docs aligned with code (`docs/AUDIT_FINAL_24B.md`)
2. Explicitly commit or quarantine `frontend/`
3. **24D** only with approval: thin client → real API, no authority in browser

## Phase 24C — Clean-slate frontend (implemented; audit fixes applied; re-verification pending)
New public site, sign-in screen and 24-route operator console on preview data. See `docs/FRONTEND_REBUILD.md`.

## Phase 24D — Backend integration (PARTIAL; details in docs/PHASE24D.md)
Done and verified by Go/TS checks: live `/api/v1` adapter (default; preview opt-in), real sign-in/session/CSRF, signed grants + durable grant ledger (PostgreSQL ledger not yet run against a live DB), server-side ownership/isolation + RBAC checks, case status history, embedded production SPA serving.
Remaining: API-key role persistence, read endpoints (workers, assignments, packs/trust, decisions/grants/evidence/provenance/audit), console control wiring (workers, trust, assessments, reports, autonomy), durable sessions, Vitest, PostgreSQL integration run, browser E2E.

### Post-audit fix pass (applies to 24C)
- API: authenticated principal is now authoritative for evaluate, grants, assessment create and workflow start; `/execute` requires the presenting principal to own the grant.
- Dev server binds `127.0.0.1:8080` by default; non-loopback listen without mandatory auth is refused; `http.Server` timeouts configured.
- Vite upgraded to 8.3.5 (+ `@vitejs/plugin-react` 6.2.1).
- Preview adapter: strict target-in-scope validation; fake `sha256:` identifiers replaced with `sample:` identifiers.
- The open grant-provenance concern raised here was resolved in 24D (signature + issued-grant ledger).

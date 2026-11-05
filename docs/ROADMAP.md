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
- Live `/api/v1` adapter implementing `CyberosApi` (swap one line in `frontend/src/api/index.ts`)
- Real authentication/session; sign-in form activation
- Endpoints: assessments, decisions, grants, executions, provenance graph, autonomy status, worker/trust admin actions, report download
- Serve the built `frontend/dist` from the Go server (replace placeholder `internal/api/v1/static/index.html`, mind `//go:embed`)

### Post-audit fix pass (applies to 24C)
- API: authenticated principal is now authoritative for evaluate, grants, assessment create and workflow start; `/execute` requires the presenting principal to own the grant.
- Dev server binds `127.0.0.1:8080` by default; non-loopback listen without mandatory auth is refused; `http.Server` timeouts configured.
- Vite upgraded to 8.3.5 (+ `@vitejs/plugin-react` 6.2.1).
- Preview adapter: strict target-in-scope validation; fake `sha256:` identifiers replaced with `sample:` identifiers.
- Open concern for 24D / backend phase: the Broker validates a client-supplied grant structurally and by expiry/nonce but there is no server-side registry proving the grant was issued by the Control Plane (see `state/HANDOFF.md`).

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
| 24B | Frontend Parts 1–3 + D1–D5 (MOCK UI) | **PARTIAL** — source under `frontend/`; often **uncommitted**; no real API |
| 24C | Real `/api/v1` integration | **DEFERRED** |
| 25+ | Marketplace / JARVIS / further product | **DEFERRED** |

## Live verification

PostgreSQL, multi-worker, and mTLS fleet checks **SKIP** unless `DATABASE_URL` / cert env is set. Absence of env must never be reported as production PASS.

## Next recommended

1. Keep docs aligned with code (`docs/AUDIT_FINAL_24B.md`)
2. Explicitly commit or quarantine `frontend/`
3. **24C** only with approval: thin client → real API, no authority in browser

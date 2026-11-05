# Handoff

## Current state: Phase 24D (partial) — branch `feat/cyberos-phase-24d-integration`

Phase 24C is **accepted**. 24D delivered: server-side sessions + cookie/CSRF auth, signed grants **plus a durable issued-grant ledger** (Broker rejects forged/unissued/replayed/revoked/expired grants), server-side ownership isolation + RBAC permission checks, case status history, a live `/api/v1` frontend adapter (default; preview opt-in), real sign-in/route guarding, embedded production serving. Detail and exact remaining dependencies: `docs/PHASE24D.md`.

Verified: `go vet ./...` and `go test ./...` clean; frontend `typecheck`, `lint` (0 warnings), `build` pass; earlier manual smoke of the embedded binary (SPA fallback, 404 asset, 401s, cookie flags, `/auth/me`).
NOT verified: any PostgreSQL run (ledger, case history, migrations 022/023; integration tests are written but skipped without a DB), browser E2E, authenticated write flows from the real UI, UI screens beyond type/build checks. `go test -race` is unusable on this kernel (ThreadSanitizer VMA error).

Known gap: API-key roles are not persisted/issuable, so every tenant key is `analyst`; admin-gated actions work only for the operator credential.

Next: API-key roles; read endpoints (workers list, packs/trust, decisions/grants/evidence/audit, assignments); wire worker/trust/assessment/report/autonomy controls; durable sessions; Vitest for the API client. Do not add authority to the browser.

---
## Previous: Phase 24C notes (historical)

## Current state: Phase 24C frontend + post-audit fix pass — implemented, UNVERIFIED

Backend remains the Go modular monolith (spine: Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor). `docs/AUDIT_FINAL_24B.md` is historical; the latest state is the post-24C audit and this fix pass.

### What changed in the fix pass
1. **Principal binding.** `bindPrincipal` (`internal/api/v1/isolation.go`) uses `auth.PrincipalFromContext` as authoritative when auth is on: a different body `principal_id` -> 403, omitted -> filled from context, auth on but no identity -> 401. Applied to `/evaluate`, `/grants`, assessment create, workflow start. `/execute` requires the presenting principal to equal `grant.Principal.ID`. `enforceProject` added to evaluate/grants/execute/assessment create. Dev mode (no auth middleware) is unchanged.
2. **Listen safety.** Default `127.0.0.1:8080`. `config.ValidateListenSecurity` refuses wildcard / non-loopback addresses (IPv4, IPv6, empty host, unknown hostnames) unless auth is mandatory. `http.Server` with ReadHeader 10s / Read 30s / Write 120s / Idle 120s (env-overridable); the SSE handler clears its own write deadline.
3. **Vite 8.3.5** + `@vitejs/plugin-react` 6.2.1 (Node `^20.19 || >=22.12`); `vite.config.ts` alias now uses `import.meta.url`.
4. **TS/lint.** Explicit generics on `DataTable`/`Tabs`/`Segmented`; typed setter wrappers; `key` props on KV tuples; nav-open derived from pathname (no effect-driven setState); `theme-context.ts`, `toast-context.ts`, `spine-stages.ts` split out.
5. **Preview adapter.** Strict target-in-scope matching; `sample:` identifiers instead of fake `sha256:`.

### Tests/build/lint/type-check were NOT executed because the owner reserved verification for a separate phase.

### Next steps
1. Owner: `go vet ./... && go test ./...`; `cd frontend && npm ci && npm run typecheck && npm run lint && npm run build && npm audit`.
2. Existing API tests that authenticate as one principal but send another `principal_id` will now (correctly) get 403; update those tests, not the check.
3. Deployments that set `CYBEROS_API_ADDR` to a wildcard/non-loopback address must also set `CYBEROS_AUTH_REQUIRED=true` or startup is refused.
4. Only after acceptance: Phase 24D.

### Open concerns (not fixed; need a design decision)
- **Grant provenance at `/execute`.** `Broker.Execute` validates the client-supplied `ExecutionGrant` structurally, by status/expiry and by nonce replay, but nothing proves the Control Plane issued it (no issued-grant registry or signature check). The owner check above stops using *someone else's* grant, but an authenticated caller could still forge a grant for themselves. Fix belongs in the Broker/Issuer (signed grants or an issued-grant store) and is deliberately not done here to avoid changing the security architecture.
- Other handlers (`/capabilities` POST, `/tasks`, `/scopes`, `/projects`) were not changed; review authorisation there during the backend phase.
- Fonts are not bundled; Go server does not yet serve `frontend/dist`; sign-in is inert.

### Do not
Add grant/trust/verify/execute methods to `CyberosApi`; start Phase 16+ work; commit/push without request.

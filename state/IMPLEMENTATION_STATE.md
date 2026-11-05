# CYBEROS Implementation State

**Last Updated:** 2026-08-27  
**Release:** **v0.15.0** (Phase 15 frozen)

## Phase 15 — COMPLETE
- [x] 15A Event + Observation contracts
- [x] 15B Event → Analyze → Re-plan → Policy
- [x] 15C Durable outbox + dispatcher
- [x] 15D Provider hardening + external brain API
- [x] 15E Final audit + freeze
- [x] Tag **v0.15.0**

## Prior
- v0.14.0 Durable controlled autonomy
- v0.13.0 Autonomy foundation
- v0.12.0 AI Brain


## Phase 24D — Integration — PARTIAL
- [x] Sessions + login/me/logout, cookie flags, CSRF + Origin, login rate limit (`internal/auth/session.go`, `handlers_auth.go`)
- [x] Grant provenance: HMAC-signed grants (`internal/broker/provenance*.go`) + durable issued-grant ledger (`ledger.go`, `postgres/grant_ledger.go`, migration 023); unit tests verified, PostgreSQL integration test written but NOT run
- [x] Isolation: server-side ownership resolution, uniform 404 on foreign by-id, RBAC permission checks, scope/task ref validation (`tests/security/isolation_24d_test.go`, `session_test.go`, `internal/api/v1`)
- [x] Case status history (migration 022, `GET /cases/{id}/history`) — unit-level only, not run against PostgreSQL
- [x] Embedded SPA serving + tests (`internal/api/v1/ui.go`, `ui_test.go`), `make web`/`release`
- [x] Frontend live adapter, session provider, route guard, real sign-in (typecheck/lint/build pass)
- [ ] API-key roles persisted + issuable (all tenant keys are currently `analyst`)
- [ ] Workers list, packs/trust, decisions/grants/evidence/provenance/audit read endpoints and UI wiring
- [ ] Report download, assessment creation, autonomy/case controls wiring
- [ ] Durable sessions; frontend unit tests; PostgreSQL integration run; browser E2E
Go: `go vet ./...` and `go test ./...` clean (no PostgreSQL server available, so integration-tagged tests did not run). See `docs/PHASE24D.md`.

## Phase 24C — Frontend clean-slate rebuild — ACCEPTED by owner
**Updated:** 2026-11-05
- [x] Old UI retired in 3 locations; Go embed placeholder in place
- [x] Design system, component library, API seam, preview adapter
- [x] Public site + sign-in UI + 24 console routes
- [x] Docs: `docs/FRONTEND_REBUILD.md`, roadmap, build plan, handoff, decisions
- [x] Owner verification and acceptance
- [x] Phase 24D started (see above)

Tests/build/lint/type-check were not executed in 24C itself; they were run in 24D (see above).

## Post-24C audit fix pass — superseded by 24D verification (go vet/test and frontend typecheck/lint/build now pass; `npm audit` not re-run)
- [x] Principal binding (`internal/api/v1/isolation.go`, `server.go`, `handlers_assessment.go`, `handlers_workflow.go`); `/execute` grant-owner check
- [x] Loopback default + listen-security validation + `http.Server` timeouts (`internal/config/config.go`, `cmd/cyberos/main.go`, SSE write-deadline clear in `sse.go`)
- [x] Vite 8.3.5 + plugin-react 6.2.1, lockfile regenerated with `--package-lock-only`
- [x] TypeScript generic-inference fixes; lint fixes (keys, nav effects, export separation)
- [x] Preview adapter: target-in-scope validation; `sample:` identifiers
- [ ] `npm audit` not re-run in 24D

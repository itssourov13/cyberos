# Phase 20 — Production & Distributed Hardening

## Scope
- Multi-node assignment claim via `ClaimAtomic` (conditional UPDATE / memory lock)
- Trust multi-instance semantics (shared store tests; PG adapters from 19B)
- Fail-closed: empty/unavailable trust never yields TRUSTED
- Preserve Phase 17–19 security spine and 19A pack binding

## Non-goals
Marketplace, JARVIS, second Broker/GrantIssuer, Web UI, public trust admin API

## Assignment claim
`AssignmentRepository.ClaimAtomic` — only one successful claim under contention; terminal reclaim denied; worker ownership enforced.

## Trust
PostgreSQL `PackTrustRepo` / `SignerKeyRepo` remain production authority when DSN configured. No silent memory fallback in `NewProduction`.

## Security spine unchanged
Scope → Auth → CP → Decision → Grant → Broker → Host → Sandbox → Executor

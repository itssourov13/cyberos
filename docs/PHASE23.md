# Phase 23 — Operator surfaces

## 23A Audit
Core security spine sound; CLI/API partial; live remote SKIPPED.

## 23B CLI + API operator workflow (COMPLETE)

### CLI commands added
- `cyberos status` — non-secret wiring report
- `cyberos doctor` — degraded if Control/Broker missing; no insecure fallback
- `cyberos trust help` — TrustAdmin SERVICE boundary (no client trust injection)
- `cyberos workers help` / `assignments help` — operational guidance

Existing: evaluate, grant, execute, projects, scopes, tasks, tools, findings, …

### Security
CLI → SDK → CP → Grant → Broker only. No Host/Executor direct calls.
TRUSTED ≠ AUTHORIZED. EffectivePackTrust not client-controlled.

### Web UI readiness
**READY WITH LIMITATIONS** — operator CLI improved; live remote still unverified.

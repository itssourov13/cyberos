# Phase 17 — Execution Host Expansion

## 17A — ExecutionHost + LocalHost (COMPLETE)

### Architecture
```
Scope → Auth → CP → Decision → Grant → Broker → HostRegistry → LocalHost → Sandbox/Executor
```

- **Broker** remains the sole authorization gateway (Grant, expiry, nonce).
- **ExecutionHost** is routing only — never issues Grants or calls CP.
- **LocalHost** wraps existing `contracts.Executor` (Router/Registry/Sandbox path).
- **HostID** is a logical allowlisted id (`local`), not a URL/endpoint.

### Host selection
- Default: `local`
- Explicit `host_id=local` allowed
- Unknown / URL-like host IDs → **fail closed**
- AI cannot invent host endpoints

### NOT in 17A
ContainerHost, RemoteHost, worker identity, mTLS, marketplace, JARVIS

### Security
| Path | Result |
|------|--------|
| Invalid/expired/replay Grant | Host not invoked |
| Unknown host | DENY |
| Event/AI → Host | No direct path |
| Host → GrantIssuer | Forbidden / absent |

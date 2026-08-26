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

## 17B — ContainerHost (COMPLETE)

### Architecture
```
Broker → HostRegistry → ContainerHost → ContainerRuntime → hardened container
```

- HostID `container` (logical; not a URL)
- Default empty HostID remains **local** (17A compatibility)
- Grant validated **before** ContainerHost / runtime

### Security defaults
- Privileged: **forbidden**
- Docker socket mount: **forbidden**
- Host network: **forbidden** (network=none required)
- Arbitrary host mounts: **forbidden**
- Image: **allowlisted** only
- Non-root user (`65534:65534`), no-new-privileges, read-only root
- Memory / PIDs / output / timeout bounded
- CLI runtime uses `exec.CommandContext` argv arrays (no shell)

### Live runtime
Requires operator-configured `docker`/`podman` binary. Unit tests use a **fake** runtime.

## 17C — Worker Identity (COMPLETE)

### Implemented
- Worker identity + lifecycle: PENDING → ACTIVE → SUSPENDED → REVOKED/EXPIRED
- WorkerRegistry (register, activate, suspend, revoke, heartbeat, authenticate)
- Memory + PostgreSQL repositories; migration `018_execution_workers.sql`
- Host↔worker binding (unique per project)
- Credential **ref** + cert fingerprint only (no private keys)
- RemoteHost stub: **Execute always rejects** until 17D
- Operator API: register/get/suspend/revoke/heartbeat under `/api/v1/projects/{project}/workers`

### Security
- Worker ≠ GrantIssuer / CP / Broker
- Heartbeat ≠ authorization
- Cross-project Get/Heartbeat DENY
- URL host IDs rejected
- No remote job dispatch in 17C

## 17D — Remote Execution Lifecycle (COMPLETE)

### Implemented
- Durable `ExecutionAssignment` + `AssignmentService` (create/dispatch/claim/renew/complete/cancel/recover)
- Memory + PostgreSQL repositories; migration `019_execution_assignments.sql`
- `RemoteHost` creates assignment after worker auth; **no Grant persistence**
- Idempotent dispatch by execution_id; result hash conflict detection
- AutoComplete mode for in-process fake worker tests
- Cancel is durable; completed cannot be claimed

### Security
- Grant validated by Broker **before** RemoteHost
- Worker mismatch / revoked / inactive → DENY
- No AI/Event → worker execute path

# CYBEROS Security Model

**Phase 0 Foundation Document** (invariants still apply)

> Code-verified status: [AUDIT_FINAL_24B.md](AUDIT_FINAL_24B.md). TRUSTED ≠ AUTHORIZED. Broker is the sole execution gateway.


## The Non-Bypassable Invariant

Every sensitive operation **MUST** pass through the following chain. No component may create a shortcut.

```
SCOPE
  ↓
PRINCIPAL
  ↓
AUTHENTICATION
  ↓
AUTHORIZATION
  ↓
POLICY
  ↓
PERMISSION
  ↓
RISK
  ↓
APPROVAL
  ↓
EXECUTION GRANT
  ↓
TOOL BROKER
  ↓
EXECUTION
  ↓
EVIDENCE
  ↓
PROVENANCE
  ↓
EVENT
  ↓
AUDIT
```

## Control Plane

The Control Plane is the security authority of CYBEROS.

It evaluates:

1. Scope
2. Principal
3. Authorization
4. Policy
5. Permission
6. Risk
7. Approval
8. Execution Grant

Decisions are structured (not simple booleans) and include:

- decision
- reason
- principal
- project
- scope
- task
- capability
- permission
- risk
- approval state
- policy references
- correlation ID
- timestamp

## Execution Grant

The Execution Grant is the security bridge between authorization and execution.

It is:

- short-lived
- narrowly scoped
- task-specific
- capability-specific
- auditable

Future versions may add cryptographic signing. Phase 1 uses a secure internal representation.

## Tool Broker

The Tool Broker is the **only** execution gateway.

```
Control Plane → Execution Grant → Tool Broker → Executor
```

The Broker validates:

- grant
- principal
- task
- capability
- scope
- expiration
- constraints

**Phase 1 rule:** Only safe mock execution is permitted. No unrestricted shell. No real pentesting tools. No arbitrary command execution.

## Principal Model

Principals are first-class. Future types include:

- HUMAN
- SERVICE
- AGENT
- PLUGIN
- ZOYA
- EXTERNAL_CLIENT
- AUTOMATION

The architecture must never assume “only humans can initiate tasks.”

## Scope

Scope is the root authorization boundary. A task cannot receive authorization outside its scope. Expired scopes must fail authorization.

## Capability vs Authorization

- **Capability Registry** answers: “What is technically available?”
- **Authorization** answers: “Is this principal allowed to use it?”

These remain strictly separate.

## Security Test Requirements (Future)

Critical tests must cover (among others):

- missing / expired scope
- out-of-scope asset
- unauthorized principal / capability
- policy / permission denial
- missing / invalid approval
- expired / malformed / replayed execution grant
- wrong principal / task / capability
- invalid state transitions
- direct Tool Broker misuse
- attempted Control Plane bypass

The invariant must be tested continuously.

## Phase 1D — Identity + Scope Boundary (Implemented)

### Principal (Identity)

- `internal/identity.Service` manages principals.
- `RequireActive` is the gate: principal must exist, validate, and be active.
- Inactive principals are rejected with `ErrForbidden`.

### Scope (Authorization Boundary)

- `internal/application/scope.Service` treats Scope as the mandatory root boundary.
- `RequireActive` fails for missing, inactive, revoked, or time-expired scopes (`ErrScopeInvalid`).
- `EnsureAllowsAsset` rejects assets outside the scope allow-list (`ErrForbidden`).
- `EnsureSameProject` enforces project membership of the scope.

These gates are the foundation for the future Control Plane evaluation chain.

## Phase 1E — Control Plane (Implemented)

The Control Plane (`internal/controlplane.Service`) evaluates:

1. Scope (RequireActive + project membership + asset constraints)
2. Principal (RequireActive)
3. Policy
4. Permission
5. Risk
6. Approval

and returns a structured `contracts.Decision` with:

- outcome (ALLOWED / DENIED)
- reason
- decision ID, timestamp
- principal / project / scope / task / capability context
- policy, permission, risk, approval sub-results
- correlation ID

Default evaluators are intentionally simple and replaceable.  
No ExecutionGrant is issued in Phase 1E.

## Phase 1G — Execution Grant + Tool Broker (Implemented)

### Grant Issuer (`internal/broker.Issuer`)

- Issues short-lived ExecutionGrants **only** after `Decision.Allowed`
- Denied decisions cannot produce grants (`ErrForbidden`)
- Grants carry nonce for replay protection, TTL, context binding

### Tool Broker (`internal/broker.Broker`)

- Sole execution gateway
- Validates: structure, status, expiry, context match, nonce (replay), executor hints
- **NO VALID GRANT ⇒ NO EXECUTION**

### Mock Executor (`internal/execution/mock`)

- Only executor registered in Phase 1G
- Returns synthetic output; no shell, no real tools, no network scanners

## Phase 1K — End-to-End Validation (Complete)

Integration tests under `tests/integration` prove:

1. Full chain: Scope → Principal → Control Plane → Grant → Broker → Mock → Evidence → Provenance → Audit
2. No valid grant ⇒ no execution
3. Denied decisions cannot issue grants
4. Expired scopes block the chain
5. Grant nonce replay is rejected
6. Only the safe mock executor is used
7. CLI, API, and SDK share the same Control Plane services

## Phase 2C — Resource Policies + Sandbox Boundary

### ResourcePolicy (`contracts.ResourcePolicy`)
- MaxDuration, MaxMemoryBytes, MaxOutputBytes, MaxConcurrency
- Hard ceilings; network/filesystem flags default **false**
- Not authorization

### Sandbox (`internal/execution/sandbox`)
- Wraps inner Executor (Phase 2C: mock only)
- Rejects AllowNetwork / AllowFilesystem
- Enforces concurrency, output size, duration
- SDK broker path uses sandbox → mock

## Phase 3D — Secrets Hooks + Secure Configuration

- `secrets.Ref` (`env:NAME`, `file:path`) — references only, never plaintext in task/events/audit
- `Resolver` / `EnvResolver` / `MapResolver` / `Chain`
- `RedactString`, `SanitizeMap`, `AssertNoSecretInMetadata`
- Config holds `SecretRefs`, not resolved values; `RedactedDatabaseURL()` for logs

Secrets hooks do **not** authorize execution.

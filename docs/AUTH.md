# Authentication & Tenant Isolation (Phase 5)

## Authentication boundary

- Package: `internal/auth`
- **Authentication ≠ Authorization**
- Successful auth binds `PrincipalRef` (+ optional `TenantID`) to request context
- Does **not** issue ExecutionGrants or bypass Control Plane

### API keys

- Format: `cos_<prefix><secret>`
- Header: `Authorization: Bearer <key>` or `X-API-Key`
- SSE: `?access_token=<key>`
- Stored as SHA-256 hash; plaintext returned only at issue time
- Supports expiry and revoke

### OIDC

- `OIDCBoundary` is fail-closed until an IdP adapter is configured
- Core does not hard-code a provider

### Public paths (when middleware enabled)

- `/api/v1/health`, `/api/v1/version`
- `/ui/` static foundation

## Tenant / project isolation

- `Tenant` domain + `Project.TenantID`
- `auth.AssertSameTenant` rejects cross-tenant access
- Isolation is **not** execution authorization

## What authentication does NOT do

- Does not authorize tool execution
- Does not replace Scope → Control Plane → Grant
- No valid grant ⇒ no execution (unchanged)

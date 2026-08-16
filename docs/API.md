# CYBEROS API

**Phase 0 Foundation Document**

## Principles

- Versioned (`/api/v1/`, future `/api/v2/`)
- Thin handlers — business logic lives in application / control-plane services
- API-first design so that CLI, Web UI, SDK, Agents, ZOYA, and UAB all consume the same contracts
- Structured error responses and correlation IDs

## Planned Endpoints (Phase 1+)

```
/api/v1/projects
/api/v1/assets
/api/v1/scopes
/api/v1/tasks
/api/v1/capabilities
/api/v1/events
/api/v1/findings
/api/v1/evidence
```

Additional endpoints for approvals, audit, grants, and identity will be introduced as the control plane solidifies.

## OpenAPI

OpenAPI specifications will live under `api/openapi/`.  
JSON/YAML schemas under `api/schemas/`.

## Clients

- Official CLI (`cmd/cyberos`)
- Official SDK (`internal/sdk` → later public surface)
- Future Web UI (thin client only)
- Future Agents / ZOYA / UAB (via SDK or API)

No client may bypass the Control Plane.

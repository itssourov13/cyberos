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

## Phase 1J — Implemented

### Endpoints (`/api/v1/`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Liveness |
| GET | `/api/v1/version` | Build info |
| GET/POST | `/api/v1/capabilities` | List / register (discovery only) |
| POST | `/api/v1/evaluate` | Control Plane evaluation |
| POST | `/api/v1/grants` | Issue grant after evaluation (403 if denied) |
| POST | `/api/v1/execute` | Execute under grant via Tool Broker |

Handlers are thin adapters over `internal/sdk.Client`.

### CLI

```
cyberos version | health
cyberos capabilities list | register <id> <name> <type>
cyberos evaluate --principal ... --scope ... --project ... --capability ...
cyberos grant ...
cyberos execute --grant-json <file>
cyberos serve   # API server
```

### SDK

`internal/sdk.Client` — in-process foundation for automation and future agents/ZOYA/UAB.

## Phase 3A — Expanded surfaces

### New API routes
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/projects` | List projects |
| GET | `/api/v1/scopes?project_id=` | List scopes |
| GET | `/api/v1/tasks?project_id=&scope_id=` | List tasks |
| GET | `/api/v1/tasks/{id}` | Task detail |
| GET | `/api/v1/dashboard` | Summary counts |
| GET | `/ui/` | Web UI foundation |

### CLI
`projects`, `scopes`, `tasks`, `dashboard`

### Web UI
Static thin client under `/ui/` — calls API only; no business logic.

# CYBEROS CLI

Operator surface only. Does **not** authorize execution.

## Commands (post-23B)

| Command | Purpose |
|---------|---------|
| `version` / `health` | Build health |
| `status` | Non-secret wiring (SDK/Control/Broker) |
| `doctor` | Degraded if Control/Broker missing; no insecure fallback |
| `evaluate` | CP evaluate (requires wired Control) |
| `grant` | Issue Grant via Issuer after Decision |
| `execute` | Execute via **Broker** with Grant JSON |
| `projects` / `scopes` / `tasks` | Catalog views |
| `tools` | Pack/tool discovery (not authorization) |
| `trust help` | TrustAdmin is **SERVICE-gated**; CLI does not inject trust |
| `workers` / `assignments` | Operational guidance |
| `findings` / `assets` / `risks` / `cases` / `reports` | Domain views |

## Security

```
CLI → SDK → Auth → CP → Grant → Broker → Host
```

Forbidden: CLI → Host/Executor, client `EffectivePackTrust`, client-minted Grants.

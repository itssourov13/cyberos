# Phase 12 — AI Brain (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **12A** | Brain contract + Plan IR | Done (`c3618ec`) |
| **12B** | Context assembly + sanitization | Done |
| **12C** | Provider adapters | Done |
| **12D** | Plan → workflow | Done |
| **12E** | Freeze | Not started |

## Trust boundary (12B)

```
CYBEROS data → Assembler (isolation) → Redactor → AIContext → future Brain adapter
```

AIContext is data only. No path to Grant/Broker/Executor.
Secret values never enter context; tool/evidence content is `trust: untrusted`.

## 12C Providers
- `openai_compatible` / `local_http` HTTP adapters (configurable BaseURL+model)
- Credentials via `secrets.Ref` only
- Provider output validated as Plan IR (12A); never executes tools

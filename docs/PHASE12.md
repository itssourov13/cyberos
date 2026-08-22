# Phase 12 — AI Brain (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **12A** | Brain contract + Plan IR | Done (`c3618ec`) |
| **12B** | Context assembly + sanitization | Done |
| **12C** | Provider adapters | Not started |
| **12D** | Plan → workflow | Not started |
| **12E** | Freeze | Not started |

## Trust boundary (12B)

```
CYBEROS data → Assembler (isolation) → Redactor → AIContext → future Brain adapter
```

AIContext is data only. No path to Grant/Broker/Executor.
Secret values never enter context; tool/evidence content is `trust: untrusted`.

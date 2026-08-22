# Phase 12 — Provider-Neutral AI Brain (**FROZEN** at `v0.12.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **12A** | Brain contract + Plan IR | Done |
| **12B** | Context assembly + sanitization | Done |
| **12C** | OpenAI-compatible / local HTTP adapters | Done |
| **12D** | Plan applicator + session provenance | Done |
| **12E** | Final audit + freeze | Done |

## Architecture

```
Human/API → Brain (untrusted) → Plan IR → Validate → Accept
  → Applicator → Workflow Definition → [STOP]
Operator Advance → CP → Grant → Broker → Sandbox → Executor
  → Evidence → Findings → AIContext → future re-plan
```

**Sole execution path:** Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor

## Invariants

- AI ≠ authorization / Grant / Broker / Executor
- Plans are capability-first; no shell/paths/binaries
- Apply requires explicit acceptance; does not execute
- Context: project/tenant isolation + secret redaction + untrusted evidence
- Credentials: secret references only

## Limitations (honest)

| Item | Classification |
|------|----------------|
| AI session/plan provenance in-memory | Product/durability |
| Live PG AI tables | Not implemented |
| Autonomous re-plan loop | Phase 13 |
| External JARVIS product | Future |
| Model routing | Future |

## JARVIS/Zoya

External brains authenticate via API/SDK as principals, submit plans, and never gain internal Broker access.

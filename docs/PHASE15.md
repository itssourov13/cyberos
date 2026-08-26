# Phase 15 — Event-Driven Controlled Autonomy (**FROZEN** at `v0.15.0`)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **15A** | Autonomy domain events + observation contracts | Done |
| **15B** | Event → Observe → Analyze → Re-plan → Policy | Done |
| **15C** | Durable dispatcher / outbox | Done |
| **15D** | Provider / external-brain hardening | Done |
| **15E** | Final audit + freeze | Done |

## Architecture

```
Human / External Brain (OpenAI/OpenRouter/Local / future JARVIS)
        │ Auth + API/SDK
        ▼
AI Layer (Context / Brain / ValidatePlan)     ← untrusted
        │
Autonomy (Event → Observe → RePlan → Policy → Handoff eligibility)
        │ Outbox + Dispatcher (delivery only)
        ▼
Workflow Orchestrator
        │
CP → Decision → Fresh Grant → Broker → Sandbox → Executor   ← trusted kernel
        │
Evidence → AutonomyEvent → (next iteration)
```

## Trust model

| Component | Role |
|-----------|------|
| Brain / Event / Plan / Evidence | **Untrusted** |
| Autonomy Policy | **Eligibility only** |
| Handoff | Delegates to Orchestrator (does not mint Grant) |
| Control Plane | **Sole authorization** |
| Broker | **Sole execution gateway** |

## Limitations

1. Live PostgreSQL autonomy/outbox verification **SKIP** without DSN.
2. PG outbox claim uses select+update (not `SKIP LOCKED`).
3. No dedicated max-LLM-calls budget beyond existing autonomy step/runtime caps.
4. JARVIS/Zoya product, remote workers, marketplace — **future**.
5. Event processor reports eligibility; explicit Handoff/API still drives Orchestrator advance.

## Security kernel (unchanged)

```
Scope → Auth → CP → Decision → Grant → Broker → Sandbox → Executor
```

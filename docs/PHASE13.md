# Phase 13 — Controlled Autonomous Cyber Operations (in progress)

| Sub-phase | Focus | Status |
|-----------|--------|--------|
| **13A** | Observe → Analyze | Done |
| **13B** | Re-planning | Done |
| **13C** | Autonomy policy gate | Done |
| **13D** | Continuous assessment | Done |
| **13E** | Freeze | Not started |

## 13A architecture

```
ObservationRequest → AIContext (12B) → Brain.Analyze → AnalysisResult (untrusted) → STOP
```

No Grant, Broker, Orchestrator.Advance, or process execution.

AnalysisResult.Trust is always `untrusted`. Recommended capabilities are IDs only.

## 13B architecture

```
AnalysisResult → RePlanRequest → Brain.Plan → ValidatePlan → Lineage → (optional AcceptApply) Applicator → Workflow Definition → STOP
```

No Orchestrator.Advance, Grant, or Broker. Plan lineage records parent analysis/plan IDs.

## 13C architecture

```
Workflow Definition → AutonomyPolicy.Evaluate → ALLOW|WAIT|DENY|STOP
  → (only if ALLOW + CONTROLLED_AUTONOMOUS) existing Orchestrator.Advance
  → CP → Grant → Broker (unchanged)
```

Policy is an eligibility gate, not authorization. MANUAL/ASSISTED always WAIT.

## 13D architecture

```
AutonomyRun.Start → Observe → Analyze → Plan → Apply → Policy
  → (MayAdvance only) external Orchestrator.Advance → CP → Grant → Broker
RecordStepExecuted accounts budgets. Cancel/Waiting supported.
```

In-memory RunStore (process-local). No Grant issuance in autonomy layer.

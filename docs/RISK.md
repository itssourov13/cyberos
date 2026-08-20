# Risk & Prioritization (Phase 6F)

## Distinction

| Concept | Role |
|---------|------|
| **CYBEROS Risk Score** | Deterministic prioritization 0–100 (`cyberos-risk-v1`) |
| **CVSS** | Future extension — not implemented |
| **Control Plane RiskResult** | Authorization gate (Phase 1) — separate package |
| **Risk prioritization** | Downstream analysis only |

**Risk ≠ authorization.** High/critical priority never grants tool execution.

## Scoring model `cyberos-risk-v1`

Weights: **severity 40%** + **asset criticality 25%** + **exposure 20%** + **confidence 15%**.

Priority bands: INFO (&lt;25), LOW (25–44), MEDIUM (45–69), HIGH (70–84), CRITICAL (≥85).

Factors are stored on every evaluation for explainability.

## Flow

Finding (+ Asset criticality + Exposure) → Evaluate → Priority → Assessment summary

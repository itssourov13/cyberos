# Security Cases & Reports (Phase 6G)

Cases and reports are **management/reporting** only.

They do **not** authorize execution or issue grants.

## Case lifecycle

OPEN → ACKNOWLEDGED → IN_PROGRESS → RESOLVED|ACCEPTED → (REOPENED)

Priority is derived from linked risk evaluations (`riskprio`), not a second scorer.

## Report schema

`cyberos-report-v1` — JSON and Markdown exporters.

Flow: Assessment → Assets/Findings/Risk/Cases → Report → Export

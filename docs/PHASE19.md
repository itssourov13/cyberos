# Phase 19 — Pack Binding & Durable Trust

## 19A — Grant ↔ Pack Identity Binding (COMPLETE)

### Problem
Phase 18 allowed pack trust to influence CP eligibility, but `ExecutionGrant` did not bind the authorized pack content digest. That risked A→B substitution after ALLOW.

### Solution
1. **Grant fields:** `PackContentDigest`, `PackKeyID` (provenance only)
2. **CP:** `RequireTrustedPack` + `TRUSTED` requires non-empty server-supplied `PackContentDigest`
3. **Issuer:** copies digest/key only when `RequireTrustedPack` and trust is `TRUSTED`
4. **Broker:** if Grant is pack-bound, request digest must match **before** Host resolution

### Invariants
- Pack A Grant + Pack B request → **DENY**, Host.Execute not called
- Legacy grants (empty digest) unchanged
- TRUSTED ≠ AUTHORIZED; capability policy still applies
- Client/AI must not invent trust; server derives digest via TrustEvaluator

### NOT in 19A
Durable TrustStore (19B), freeze/tag (19C)

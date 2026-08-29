# Phase 22 — Remote Worker Security

## 22A — Worker Identity & mTLS (COMPLETE)

### Principle
**mTLS authenticates the worker; it does not authorize execution.**  
Grant + Broker remain the sole authorization/execution path.

### Identity model
1. TLS handshake with `RequireAndVerifyClientCert` (server) and verified server cert (client)
2. Peer leaf cert → SHA-256 fingerprint
3. `WorkerRegistry.AuthenticateTLS` binds fingerprint to registered `WorkerID + ProjectID + HostID + ACTIVE`

### Package
`internal/execution/mtls` — ServerTLSConfig, ClientTLSConfig, ExtractPeerIdentity, Bind

### Non-goals (later)
22B result authenticity · 22C CRL/OCSP · Web UI · Marketplace · JARVIS

## 22B — Assignment Binding & Result Authenticity (COMPLETE)

### Assignment
- `GrantID`, `PackContentDigest` on assignment (migration 021)
- `CreateFromGrant` copies Grant identity immutably

### Result authenticity
- `internal/execution/resultauth` — Ed25519, domain `CYBEROS-WORKER-RESULT-V1\n`
- Canonical sorted-key JSON; public keys only in `KeyRegistry`
- `VerifyCompletion` binds assignment/grant/project/worker/host/pack
- `ReplayGuard` on ResultID

Private keys never in DB/events.

## 22C — Revocation, Replay & Fail-Closed (COMPLETE)

- Complete() rejects expired leases and invalid statuses
- Result key `Revoke`/`Disable` fail-closed
- Concurrent ReplayGuard: exactly one accept
- Revoked worker AuthenticateTLS DENY
- Concurrent claim: wrong worker DENY; authorized claims safe

LIVE: UNVERIFIED without DSN/fleet

## 22D — Final Audit (COMPLETE)

| Check | Status |
|-------|--------|
| Unit/regression 18–22C | PASS |
| Race execution/tools/broker/CP | PASS |
| E2E assignment→signed result chain | UNIT |
| Live PG / mTLS / worker / Docker | SKIPPED |
| InsecureSkipVerify | absent (tests assert false) |
| Release | READY WITH DOCUMENTED LIMITATIONS |

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

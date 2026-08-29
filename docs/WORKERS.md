# Remote workers

## mTLS (Phase 22A)

TLS 1.3, `RequireAndVerifyClientCert`, no `InsecureSkipVerify`.
Peer cert → SHA-256 fingerprint → `WorkerRegistry.AuthenticateTLS(project, worker, host, fingerprint)`.

Only **ACTIVE** workers authenticate. DISABLED/REVOKED → DENY.

## Assignment + result (22B/C)

Assignment carries `GrantID` + `PackContentDigest`.
Results: Ed25519 envelope domain `CYBEROS-WORKER-RESULT-V1\n`, `VerifyCompletion`, ReplayGuard, expired lease DENY.

Private result-signing keys are **never** stored in PostgreSQL.

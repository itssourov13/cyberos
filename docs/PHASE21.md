# Phase 21 — Observability & Operator Trust Ops

## Observability
- Audit actions: PACK_TRUSTED, PACK_REVOKED, SIGNER_*
- Event types: trust.pack.*, trust.signer.*
- Metadata carries pack_content_digest / pack_key_id for forensics
- Audit remains non-authoritative

## TrustAdmin
- SERVICE principal only (HUMAN/AGENT/etc denied)
- Mutates TrustStore / SignerKeyRegistry ports
- Emits audit on success
- Does not issue Grants or call Broker

## Non-goals
Marketplace, JARVIS, second Broker, Web UI, worker mTLS

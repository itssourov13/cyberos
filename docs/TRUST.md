# Pack trust

```
Canonical bytes → SHA-256 → Ed25519 → Signer registry → TrustStore
  → TrustEvaluator → CP eligibility → Grant.PackContentDigest → Broker bind
```

- TRUSTED ≠ AUTHORIZED
- REVOKED dominates
- Project-scoped pack trust; global signer keys
- PackKeyID = provenance only
- TrustAdmin = SERVICE principal only

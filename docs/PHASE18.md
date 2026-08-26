# Phase 18 — Pack Supply Chain

## 18A — Pack Identity & Integrity (COMPLETE)

### What 18A proves
Content identity via deterministic canonicalization + SHA-256.

```
PackFile → Validate → CanonicalBytes → sha256:<hex> → PackIdentity
```

### Digest format
`sha256:` + 64 lowercase hex characters.

### PackIdentity
- Name, Version, Digest (Namespace reserved)
- **Digest ≠ signature ≠ trust ≠ authorization**

### Behavior
- `VerifyDigest(pack, expected)`: empty expected = legacy undigested OK; mismatch = **fail closed**
- `Loader.InstallPackFile` rejects declared `content_digest` mismatch
- No automatic digest repair

### Canonicalization
- Sorted object keys, sorted tool IDs / capability lists
- Independent of source JSON whitespace/field order
- No timestamps, paths, or registry state in the digest

### NOT in 18A
Signing, key registry, trust tiers, revocation, marketplace, Grant/Broker changes

### Future
- **18B** — signature + signer verification over the same canonical bytes
- **18C** — trust / revocation policy

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

## 18B — Signing & Verification (COMPLETE)

### Cryptography
- **Ed25519** (`crypto/ed25519`)
- Signed message: `CYBEROS-PACK-SIG-V1\n` || `CanonicalBytes(pack)`
- Signature encoding: **base64.StdEncoding** (64-byte sig)
- `SIGNED_BYTES` exclude `content_digest` and `signature` (no circular hash)

### Key registry
- `SignerKeyRegistry`: Register (no replace), Get, Disable
- Pack **cannot** register its own trust anchor
- DISABLED / UNKNOWN key → verification **FAIL**

### Verification modes
| Mode | Behavior |
|------|----------|
| LEGACY | undigested/unsigned OK; not reported signed |
| DIGEST_ONLY | content_digest required + match |
| SIGNED_REQUIRED | active key + valid signature |

### Semantics
**Signature ≠ trust ≠ authorization.** No Pack → Grant/Broker path.

### NOT in 18B
Trust tiers, full revocation, CP integration, marketplace, key rotation product

## 18C — Trust Policy & Revocation (COMPLETE)

### Trust states
`UNTRUSTED` → `VERIFIED` → `TRUSTED` · any → `REVOKED` (terminal)

### Semantics
| Outcome | Meaning |
|---------|---------|
| VERIFIED | Valid signature + active key; **not** policy-trusted |
| TRUSTED | Explicit digest trust (or optional signer inheritance, **default OFF**) |
| REVOKED | Pack digest or signer revoked — dominates TRUSTED |

### Policy
- Project-scoped `TrustStore` (TrustPack / RevokePack)
- Signer `Revoke` → KEY_REVOKED for all packs
- **Signature alone never yields TRUSTED**
- **Trust ≠ authorization** (no CP/Grant/Broker path)

### Loader mode
`TRUST_REQUIRED` — install only if `TrustDecision.State == TRUSTED`

### NOT in 18C
CP integration (18D), marketplace, automatic trust-from-signer (default)

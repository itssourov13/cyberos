package tools

import (
	"context"

	"cyberos/internal/contracts"
)

// TrustStorePort is the durable/project-scoped pack trust store (Phase 19B).
// Implementations: memory TrustStore, PostgreSQL. Not an authorization authority.
type TrustStorePort interface {
	TrustPack(project contracts.ProjectID, digest string) error
	RevokePack(project contracts.ProjectID, digest, reason, by string) error
	IsTrusted(project contracts.ProjectID, digest string) bool
	IsRevoked(project contracts.ProjectID, digest string) bool
}

// SignerKeyRegistryPort holds public verification keys only (Phase 19B).
// Never stores private keys. Not an authorization authority.
type SignerKeyRegistryPort interface {
	Register(key SignerKey) error
	Get(keyID string) (SignerKey, error)
	Disable(keyID string) error
	Revoke(keyID string) error
}

// Ensure memory types implement ports.
var (
	_ TrustStorePort         = (*TrustStore)(nil)
	_ SignerKeyRegistryPort  = (*SignerKeyRegistry)(nil)
)

// Context-aware variants optional for PG; default methods ignore ctx via wrappers.
type TrustStorePortCtx interface {
	TrustPack(ctx context.Context, project contracts.ProjectID, digest string) error
	RevokePack(ctx context.Context, project contracts.ProjectID, digest, reason, by string) error
	IsTrusted(ctx context.Context, project contracts.ProjectID, digest string) (bool, error)
	IsRevoked(ctx context.Context, project contracts.ProjectID, digest string) (bool, error)
}

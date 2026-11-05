package tools

import (
	"fmt"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
)

// TrustAdmin is the operator-gated trust mutation surface (Phase 21).
// Only SERVICE principals may mutate trust/signer state.
// Does not issue Grants or call Broker.
type TrustAdmin struct {
	Store TrustStorePort
	Keys  SignerKeyRegistryPort
	Audit *audit.Service // optional
}

// AuthorizeOperator rejects non-SERVICE principals (fail closed).
func AuthorizeOperator(p contracts.PrincipalRef) error {
	if p.ID == "" {
		return fmt.Errorf("%w: operator principal required", contracts.ErrForbidden)
	}
	// The bootstrap credential is the platform's human break-glass operator.
	// No other human account may administer signer or pack trust state.
	if p.Type != contracts.PrincipalTypeService &&
		!(p.Type == contracts.PrincipalTypeHuman && p.ID == "bootstrap") {
		return fmt.Errorf("%w: trust admin requires a platform operator principal", contracts.ErrForbidden)
	}
	return nil
}

func (a *TrustAdmin) TrustPack(op contracts.PrincipalRef, project contracts.ProjectID, digest, correlation, reason string) error {
	if err := AuthorizeOperator(op); err != nil {
		return err
	}
	if a == nil || a.Store == nil {
		return fmt.Errorf("%w: trust store not configured", contracts.ErrInvalidArgument)
	}
	if err := a.Store.TrustPack(project, digest); err != nil {
		return err
	}
	if a.Audit != nil {
		if _, err := a.Audit.RecordTrustMutation("PACK_TRUSTED", op, project, digest, "", reason, correlation, nil); err != nil {
			return fmt.Errorf("pack trust changed but audit persistence failed: %w", err)
		}
	}
	return nil
}

func (a *TrustAdmin) RevokePack(op contracts.PrincipalRef, project contracts.ProjectID, digest, reason, correlation string) error {
	if err := AuthorizeOperator(op); err != nil {
		return err
	}
	if a == nil || a.Store == nil {
		return fmt.Errorf("%w: trust store not configured", contracts.ErrInvalidArgument)
	}
	if err := a.Store.RevokePack(project, digest, reason, string(op.ID)); err != nil {
		return err
	}
	if a.Audit != nil {
		if _, err := a.Audit.RecordTrustMutation("PACK_REVOKED", op, project, digest, "", reason, correlation, nil); err != nil {
			return fmt.Errorf("pack revocation committed but audit persistence failed: %w", err)
		}
	}
	return nil
}

func (a *TrustAdmin) RegisterSigner(op contracts.PrincipalRef, key SignerKey, correlation string) error {
	if err := AuthorizeOperator(op); err != nil {
		return err
	}
	if a == nil || a.Keys == nil {
		return fmt.Errorf("%w: signer registry not configured", contracts.ErrInvalidArgument)
	}
	if err := a.Keys.Register(key); err != nil {
		return err
	}
	if a.Audit != nil {
		if _, err := a.Audit.RecordTrustMutation("SIGNER_REGISTERED", op, "", "", key.KeyID, "", correlation, nil); err != nil {
			return fmt.Errorf("signer registered but audit persistence failed: %w", err)
		}
	}
	return nil
}

func (a *TrustAdmin) DisableSigner(op contracts.PrincipalRef, keyID, correlation string) error {
	if err := AuthorizeOperator(op); err != nil {
		return err
	}
	if a == nil || a.Keys == nil {
		return fmt.Errorf("%w: signer registry not configured", contracts.ErrInvalidArgument)
	}
	if err := a.Keys.Disable(keyID); err != nil {
		return err
	}
	if a.Audit != nil {
		if _, err := a.Audit.RecordTrustMutation("SIGNER_DISABLED", op, "", "", keyID, "", correlation, nil); err != nil {
			return fmt.Errorf("signer disabled but audit persistence failed: %w", err)
		}
	}
	return nil
}

func (a *TrustAdmin) RevokeSigner(op contracts.PrincipalRef, keyID, correlation string) error {
	if err := AuthorizeOperator(op); err != nil {
		return err
	}
	if a == nil || a.Keys == nil {
		return fmt.Errorf("%w: signer registry not configured", contracts.ErrInvalidArgument)
	}
	if err := a.Keys.Revoke(keyID); err != nil {
		return err
	}
	if a.Audit != nil {
		if _, err := a.Audit.RecordTrustMutation("SIGNER_REVOKED", op, "", "", keyID, "", correlation, nil); err != nil {
			return fmt.Errorf("signer revoked but audit persistence failed: %w", err)
		}
	}
	return nil
}

func (a *TrustAdmin) IsTrusted(project contracts.ProjectID, digest string) bool {
	if a == nil || a.Store == nil {
		return false
	}
	return a.Store.IsTrusted(project, digest)
}

func (a *TrustAdmin) GetSigner(keyID string) (SignerKey, error) {
	if a == nil || a.Keys == nil {
		return SignerKey{}, fmt.Errorf("%w: signer registry not configured", contracts.ErrInvalidArgument)
	}
	return a.Keys.Get(keyID)
}

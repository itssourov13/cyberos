package contracts

import "fmt"

// Extension contract versions for ToolManifest / Pack surfaces (Phase 11A).
// These version the *contract shape*, not the tool product version.
const (
	// ExtensionContractV1 is the initial stable extension surface.
	ExtensionContractV1 SchemaVersion = "1"
)

// NormalizeExtensionContract returns the effective contract version (default V1).
func NormalizeExtensionContract(v SchemaVersion) SchemaVersion {
	if v == "" {
		return ExtensionContractV1
	}
	return v
}

// ValidateExtensionContract fails closed on unsupported contract versions.
func ValidateExtensionContract(v SchemaVersion) error {
	n := NormalizeExtensionContract(v)
	switch n {
	case ExtensionContractV1:
		return nil
	default:
		return fmt.Errorf("%w: unsupported extension contract version %q (supported: %q)",
			ErrInvalidArgument, v, ExtensionContractV1)
	}
}

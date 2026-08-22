package contracts

// ToolManifest describes a replaceable tool integration layer.
// It maps to one or more capabilities without embedding tool-specific logic in the core.
type ToolManifest struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Version      SchemaVersion  `json:"version"`
	Runtime      PluginRuntime  `json:"runtime"`
	Capabilities []CapabilityID `json:"capabilities,omitempty"`
	// ExecutorID is the registry key used by the Tool Broker / router.
	ExecutorID string `json:"executor_id"`
	// Description is human-readable metadata only.
	Description string `json:"description,omitempty"`
	// ContractVersion versions the manifest extension contract (not the tool product version).
	// Empty defaults to ExtensionContractV1 for backward compatibility.
	ContractVersion SchemaVersion `json:"contract_version,omitempty"`
}

// Validate checks required tool manifest fields.
func (m ToolManifest) Validate() error {
	if err := ValidateNonEmpty("tool.id", m.ID); err != nil {
		return err
	}
	if err := ValidateNonEmpty("tool.name", m.Name); err != nil {
		return err
	}
	if err := ValidateNonEmpty("tool.executor_id", m.ExecutorID); err != nil {
		return err
	}
	if !m.Runtime.Valid() {
		return ErrInvalidArgument
	}
	if err := ValidateNonEmpty("tool.version", string(m.Version)); err != nil {
		return err
	}
	if err := ValidateExtensionContract(m.ContractVersion); err != nil {
		return err
	}
	return nil
}

// ToolExecutor is a language-neutral execution layer behind the Tool Broker.
// Implementations must assume the grant was already validated by the broker,
// but should still refuse work when the request is structurally invalid.
type ToolExecutor interface {
	Manifest() ToolManifest
	Executor
}

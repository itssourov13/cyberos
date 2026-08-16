package contracts

// CapabilityType classifies capabilities at a high level.
type CapabilityType string

const (
	CapabilityTypeRecon    CapabilityType = "RECON"
	CapabilityTypeScan     CapabilityType = "SCAN"
	CapabilityTypeAnalysis CapabilityType = "ANALYSIS"
	CapabilityTypeUtility  CapabilityType = "UTILITY"
	CapabilityTypeCustom   CapabilityType = "CUSTOM"
)

// Valid reports whether the type is known.
func (t CapabilityType) Valid() bool {
	switch t {
	case CapabilityTypeRecon, CapabilityTypeScan, CapabilityTypeAnalysis,
		CapabilityTypeUtility, CapabilityTypeCustom:
		return true
	default:
		return false
	}
}

// CapabilityDescriptor describes a registered capability.
// Discovery of a capability does NOT imply authorization to use it.
type CapabilityDescriptor struct {
	ID          CapabilityID   `json:"id"`
	Name        string         `json:"name"`
	Type        CapabilityType `json:"type"`
	Description string         `json:"description,omitempty"`
	// Version of the capability contract itself (not the tool version).
	Version SchemaVersion `json:"version,omitempty"`
}

// Validate checks required fields.
func (c CapabilityDescriptor) Validate() error {
	if err := ValidateNonEmpty("capability.id", string(c.ID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("capability.name", c.Name); err != nil {
		return err
	}
	if !c.Type.Valid() {
		return ErrInvalidArgument
	}
	return nil
}

// CapabilityRef is a lightweight reference used in grants and decisions.
type CapabilityRef struct {
	ID CapabilityID `json:"id"`
}

// Validate checks required fields.
func (c CapabilityRef) Validate() error {
	return ValidateNonEmpty("capability.id", string(c.ID))
}

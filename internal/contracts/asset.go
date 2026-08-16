package contracts

// AssetType classifies the kind of asset.
type AssetType string

const (
	AssetTypeHost        AssetType = "HOST"
	AssetTypeDomain      AssetType = "DOMAIN"
	AssetTypeURL         AssetType = "URL"
	AssetTypeApplication AssetType = "APPLICATION"
	AssetTypeCloud       AssetType = "CLOUD"
	AssetTypeOther       AssetType = "OTHER"
)

// Valid reports whether the asset type is known.
func (t AssetType) Valid() bool {
	switch t {
	case AssetTypeHost, AssetTypeDomain, AssetTypeURL,
		AssetTypeApplication, AssetTypeCloud, AssetTypeOther:
		return true
	default:
		return false
	}
}

// AssetRef is a lightweight reference to an asset.
type AssetRef struct {
	ID   AssetID   `json:"id"`
	Type AssetType `json:"type,omitempty"`
}

// Validate checks required fields.
func (a AssetRef) Validate() error {
	return ValidateNonEmpty("asset.id", string(a.ID))
}

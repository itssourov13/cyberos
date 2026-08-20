package contracts

// AssetCriticality is independent of finding severity.
type AssetCriticality string

const (
	AssetCriticalityUnknown  AssetCriticality = "UNKNOWN"
	AssetCriticalityLow      AssetCriticality = "LOW"
	AssetCriticalityMedium   AssetCriticality = "MEDIUM"
	AssetCriticalityHigh     AssetCriticality = "HIGH"
	AssetCriticalityCritical AssetCriticality = "CRITICAL"
)

// Valid reports whether criticality is known.
func (c AssetCriticality) Valid() bool {
	switch c {
	case AssetCriticalityUnknown, AssetCriticalityLow, AssetCriticalityMedium,
		AssetCriticalityHigh, AssetCriticalityCritical:
		return true
	default:
		return false
	}
}

// ExposureContext is a simple reachability factor.
type ExposureContext string

const (
	ExposureUnknown        ExposureContext = "UNKNOWN"
	ExposureInternal       ExposureContext = "INTERNAL"
	ExposureExternal       ExposureContext = "EXTERNALLY_REACHABLE"
	ExposureInternetFacing ExposureContext = "INTERNET_FACING"
)

// Valid reports whether exposure is known.
func (e ExposureContext) Valid() bool {
	switch e {
	case ExposureUnknown, ExposureInternal, ExposureExternal, ExposureInternetFacing:
		return true
	default:
		return false
	}
}

// RiskPriority is the prioritization outcome (not authorization).
type RiskPriority string

const (
	RiskPriorityInfo     RiskPriority = "INFO"
	RiskPriorityLow      RiskPriority = "LOW"
	RiskPriorityMedium   RiskPriority = "MEDIUM"
	RiskPriorityHigh     RiskPriority = "HIGH"
	RiskPriorityCritical RiskPriority = "CRITICAL"
)

// Valid reports whether priority is known.
func (p RiskPriority) Valid() bool {
	switch p {
	case RiskPriorityInfo, RiskPriorityLow, RiskPriorityMedium, RiskPriorityHigh, RiskPriorityCritical:
		return true
	default:
		return false
	}
}

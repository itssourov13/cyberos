package contracts

// RiskLevel is an ordered risk classification.
type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "NONE"
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// Valid reports whether the level is known.
func (r RiskLevel) Valid() bool {
	switch r {
	case RiskLevelNone, RiskLevelLow, RiskLevelMedium, RiskLevelHigh, RiskLevelCritical:
		return true
	default:
		return false
	}
}

// RiskResult captures the outcome of a risk evaluation.
type RiskResult struct {
	Level    RiskLevel `json:"level"`
	Accepted bool      `json:"accepted"`
	Reason   string    `json:"reason,omitempty"`
	Score    *float64  `json:"score,omitempty"` // optional numeric score
}

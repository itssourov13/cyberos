package contracts

// ObservationKind classifies a tool-derived observation on an asset.
type ObservationKind string

const (
	ObservationKindPortOpen   ObservationKind = "PORT_OPEN"
	ObservationKindHTTPHeader ObservationKind = "HTTP_HEADER"
	ObservationKindHostAlive  ObservationKind = "HOST_ALIVE"
	ObservationKindGeneric    ObservationKind = "GENERIC"
)

// Valid reports whether the observation kind is known.
func (k ObservationKind) Valid() bool {
	switch k {
	case ObservationKindPortOpen, ObservationKindHTTPHeader, ObservationKindHostAlive, ObservationKindGeneric:
		return true
	default:
		return false
	}
}

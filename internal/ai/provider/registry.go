package provider

import (
	"fmt"

	"cyberos/internal/ai"
	"cyberos/internal/contracts"
	"cyberos/internal/secrets"
)

// Registry constructs Brain implementations from explicit provider config.
type Registry struct {
	Resolver secrets.Resolver
}

// New creates a Brain for the given config. Selection is explicit — never AI-driven.
func (r *Registry) New(cfg Config) (ai.Brain, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	res := r.Resolver
	if res == nil {
		res = secrets.EnvResolver{}
	}
	switch cfg.Type {
	case TypeMock:
		return &ai.MockBrain{}, nil
	case TypeOpenAICompatible:
		return NewOpenAICompat(cfg, res)
	case TypeLocalHTTP:
		return NewLocalHTTP(cfg, res)
	default:
		return nil, fmt.Errorf("%w: unknown provider type %q", contracts.ErrInvalidArgument, cfg.Type)
	}
}

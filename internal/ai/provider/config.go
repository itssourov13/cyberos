// Package provider implements provider-neutral LLM adapters for the Brain contract.
// Inference only — never grants, Broker, or process execution. Phase 12C.
package provider

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/secrets"
)

// Type identifies a provider adapter kind.
type Type string

const (
	TypeOpenAICompatible Type = "openai_compatible"
	TypeLocalHTTP        Type = "local_http"
	TypeMock             Type = "mock"
)

// Config is provider configuration. Credentials are secret references only.
type Config struct {
	ID               string        `json:"id"`
	Type             Type          `json:"type"`
	BaseURL          string        `json:"base_url"`
	Model            string        `json:"model"`
	APIKeyRef        secrets.Ref   `json:"api_key_ref,omitempty"` // e.g. env:OPENAI_API_KEY
	Timeout          time.Duration `json:"timeout,omitempty"`
	MaxResponseBytes int64         `json:"max_response_bytes,omitempty"`
}

// Validate checks config shape without resolving secrets.
func (c Config) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("%w: provider id required", contracts.ErrInvalidArgument)
	}
	switch c.Type {
	case TypeOpenAICompatible, TypeLocalHTTP, TypeMock:
	default:
		return fmt.Errorf("%w: unknown provider type %q", contracts.ErrInvalidArgument, c.Type)
	}
	if c.Type != TypeMock {
		if c.BaseURL == "" {
			return fmt.Errorf("%w: base_url required", contracts.ErrInvalidArgument)
		}
		if c.Model == "" {
			return fmt.Errorf("%w: model required", contracts.ErrInvalidArgument)
		}
	}
	if c.APIKeyRef != "" {
		if err := secrets.ValidateRef(c.APIKeyRef); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 30 * time.Second
	}
	return c.Timeout
}

func (c Config) maxBytes() int64 {
	if c.MaxResponseBytes <= 0 {
		return 1 << 20 // 1 MiB
	}
	return c.MaxResponseBytes
}

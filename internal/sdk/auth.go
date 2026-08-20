package sdk

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// WithAuth attaches an authenticator (does not authorize).
func (c *Client) WithAuth(a auth.Authenticator) *Client {
	c.Auth = a
	return c
}

// Authenticate verifies credentials and returns identity.
func (c *Client) Authenticate(ctx context.Context, credential string) (auth.Identity, error) {
	if c.Auth == nil {
		return auth.Identity{}, fmt.Errorf("%w: authenticator not configured", contracts.ErrInvalidArgument)
	}
	return c.Auth.Authenticate(ctx, credential)
}

// IssueAPIKey issues a key bound to principal (+ optional tenant).
func (c *Client) IssueAPIKey(ctx context.Context, principalID contracts.PrincipalID, tenantID contracts.TenantID, ttl time.Duration) (string, auth.APIKey, error) {
	a, ok := c.Auth.(*auth.APIKeyAuthenticator)
	if !ok || a == nil {
		return "", auth.APIKey{}, fmt.Errorf("%w: API key authenticator not configured", contracts.ErrInvalidArgument)
	}
	return a.Issue(ctx, principalID, tenantID, ttl)
}

// AssertTenant checks isolation for a resource tenant.
func (c *Client) AssertTenant(caller, resource contracts.TenantID) error {
	return auth.AssertSameTenant(caller, resource)
}

package security_test

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
)

func grantReq() contracts.ExecutionRequest {
	now := time.Now().UTC()
	return contracts.ExecutionRequest{Grant: contracts.ExecutionGrant{
		ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "c",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
	}}
}

// Sandbox must not enable host network/fs in Phase 2C.
func TestSecurity_SandboxNoHostAccess(t *testing.T) {
	s := sandbox.New(mock.New())
	p := s.Policy()
	if p.AllowNetwork || p.AllowFilesystem {
		t.Fatal("default sandbox must deny network and filesystem")
	}
	if _, err := sandbox.New(mock.New(), sandbox.WithPolicy(contracts.ResourcePolicy{
		AllowNetwork: true, MaxDuration: time.Second,
	})).Execute(grantReq()); err == nil {
		t.Fatal("network policy must be rejected")
	}
}

// Resource policy is not a substitute for a grant — empty grant still fails at mock/validate.
func TestSecurity_PolicyDoesNotReplaceGrant(t *testing.T) {
	s := sandbox.New(mock.New())
	_, err := s.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("missing grant must still fail")
	}
}

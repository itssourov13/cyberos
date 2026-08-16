package security_test

import (
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/execution/mock"
)

// Lifecycle controller cannot execute without a valid grant.
func TestSecurity_LifecycleRequiresGrant(t *testing.T) {
	ctrl := lifecycle.New(broker.New(mock.New()), lifecycle.WithEvidence(evidence.NewService()))
	_, err := ctrl.Run(lifecycle.RunRequest{})
	if err == nil {
		t.Fatal("must require grant")
	}
}

// Execution records always bind principal/task/grant.
func TestSecurity_LifecycleAttribution(t *testing.T) {
	ctrl := lifecycle.New(broker.New(mock.New()), lifecycle.WithEvidence(evidence.NewService()))
	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID: "g1", Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", CapabilityID: "c1",
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		Nonce: "n-life", CorrelationID: "corr-attr",
	}
	out, err := ctrl.Run(lifecycle.RunRequest{Grant: g, CorrelationID: "corr-attr"})
	if err != nil {
		t.Fatal(err)
	}
	r := out.Record
	if r.Principal.ID != "u1" || r.GrantID != "g1" || r.TaskID != "t1" || r.ScopeID != "s1" {
		t.Fatalf("incomplete attribution: %+v", r)
	}
}

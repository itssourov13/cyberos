package integration_test

import (
	"context"
	"testing"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/audit"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/events"
	"cyberos/internal/identity"
	"cyberos/internal/sdk"
)

// Phase 3J E2E: Assessment → CP → Grant → Broker → Sandbox → Tool → Evidence path.
func TestPhase3J_AssessmentAuthorizedE2E(t *testing.T) {
	ctx := context.Background()
	pr, sr := &memP{}, &memS{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(ctx, p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(ctx, sc)

	client := sdk.NewInMemory()
	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	cp := controlplane.NewService(idSvc, scSvc)
	client.WithIdentity(idSvc).WithScopes(scSvc).WithControlPlane(cp)
	client.WithAudit(audit.NewService(client.Events))
	client.WireAssessment()

	_ = client.RegisterCapability(ctx, "assessment.stub.probe", "Stub Probe", contracts.CapabilityTypeUtility)

	a, err := client.CreateAssessment("asmt-3j", "proj-1", "scope-1", "user-1", "Phase3J", "example.test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.PlanAssessment("asmt-3j", []contracts.AssessmentStep{
		{ID: "s1", CapabilityID: "assessment.stub.probe", ExecutorHint: "stub.security.scanner"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := client.RunAssessment(ctx, "asmt-3j")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != contracts.AssessmentStateSucceeded {
		t.Fatalf("state=%s results=%+v", out.State, out.StepResults)
	}
	if len(out.StepResults) != 1 || !out.StepResults[0].Success {
		t.Fatalf("steps=%+v", out.StepResults)
	}
	if out.StepResults[0].GrantID == "" {
		t.Fatal("expected grant id on step")
	}
	_ = a
}

func TestPhase3J_DeniedControlPlane(t *testing.T) {
	ctx := context.Background()
	pr, sr := &memP{}, &memS{}
	p, _ := principal.NewHuman("user-inactive", "bob")
	p.Active = false
	_ = pr.Create(ctx, p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(ctx, sc)

	client := sdk.NewInMemory()
	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	client.WithIdentity(idSvc).WithScopes(scSvc).
		WithControlPlane(controlplane.NewService(idSvc, scSvc)).
		WireAssessment()
	_ = client.RegisterCapability(ctx, "assessment.stub.probe", "Stub", contracts.CapabilityTypeUtility)
	_, _ = client.CreateAssessment("asmt-deny", "proj-1", "scope-1", "user-inactive", "Deny", "t")
	_, _ = client.PlanAssessment("asmt-deny", []contracts.AssessmentStep{
		{ID: "s1", CapabilityID: "assessment.stub.probe"},
	})
	out, err := client.RunAssessment(ctx, "asmt-deny")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != contracts.AssessmentStateFailed {
		t.Fatalf("state=%s results=%+v", out.State, out.StepResults)
	}
}

func TestPhase3J_ExpiredScopeBlocks(t *testing.T) {
	ctx := context.Background()
	pr, sr := &memP{}, &memS{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(ctx, p)
	sc, _ := domainscope.New("scope-exp", "proj-1", "Expired", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-2 * time.Hour)
	sc.ExpiresAt = &past
	_ = sr.Create(ctx, sc)

	client := sdk.NewInMemory()
	idSvc := identity.NewService(pr)
	scSvc := appscope.NewService(sr)
	client.WithIdentity(idSvc).WithScopes(scSvc).WithControlPlane(controlplane.NewService(idSvc, scSvc)).WireAssessment()
	_ = client.RegisterCapability(ctx, "assessment.stub.probe", "Stub", contracts.CapabilityTypeUtility)
	_, _ = client.CreateAssessment("asmt-exp", "proj-1", "scope-exp", "user-1", "Exp", "t")
	_, _ = client.PlanAssessment("asmt-exp", []contracts.AssessmentStep{
		{ID: "s1", CapabilityID: "assessment.stub.probe"},
	})
	out, err := client.RunAssessment(ctx, "asmt-exp")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != contracts.AssessmentStateFailed {
		t.Fatalf("expected failed on expired scope, got %s", out.State)
	}
}

func TestPhase3J_SSEPublishDoesNotExecute(t *testing.T) {
	hub := events.NewHub()
	ch := hub.Subscribe(1)
	hub.Publish(contracts.EventEnvelope{
		ID: "e1", Type: "assessment.state", SchemaVersion: contracts.EventSchemaV1,
		Timestamp: time.Now().UTC(), CorrelationID: "c",
	})
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
	// Observation only — no broker involved.
}

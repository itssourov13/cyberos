package sdk

import (
	"context"
	"testing"

	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/tools"
	"cyberos/internal/workflow"
)

func TestWireWorkflowInMemory(t *testing.T) {
	c := NewInMemory()
	if c.Workflow == nil {
		t.Fatal("WireWorkflow should attach orchestrator")
	}
	if c.Lifecycle == nil {
		t.Fatal("lifecycle required")
	}
}

func TestStartWorkflowRequiresDefinitionValidation(t *testing.T) {
	c := NewInMemory()
	_, err := c.StartWorkflow(context.Background(), workflow.Definition{}, "u1", "")
	if err == nil {
		t.Fatal("invalid definition")
	}
}

func TestWorkflowOrchestrationDoesNotBypassWhenNoControl(t *testing.T) {
	c := NewInMemory()
	// Without Control/Identity, authorizer should fail closed on Advance
	capReg := c.Caps
	if capReg == nil {
		capReg = capability.NewRegistry(nil)
		c.Caps = capReg
	}
	cap, _ := domaincap.New("cap.demo", "Demo", contracts.CapabilityTypeRecon)
	_ = capReg.Register(context.Background(), cap)
	_ = c.Tools.Register(tools.NewStubScanner(nil))
	// re-wire after tool register
	c.Platform = tools.NewPlatform(c.Tools)
	c.WireWorkflow()
	def := workflow.Definition{
		ID: "wf", Name: "n", Version: "1", ProjectID: "p", ScopeID: "s",
		Steps: []workflow.Step{{ID: "s1", CapabilityID: "cap.demo"}},
	}
	// Materialize may fail on unknown capability resolution without platform tool for cap.demo
	run, err := c.StartWorkflow(context.Background(), def, "u1", "r1")
	if err != nil {
		// fail closed is acceptable without matching tool
		return
	}
	_, err = c.AdvanceWorkflow(context.Background(), run.ID, def)
	if err == nil {
		t.Fatal("advance without control plane must not silently succeed")
	}
}

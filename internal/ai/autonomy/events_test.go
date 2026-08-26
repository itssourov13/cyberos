package autonomy_test

import (
	"testing"
	"time"

	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
)

func validEvent() autonomy.AutonomyEvent {
	return autonomy.AutonomyEvent{
		EventID: "ev-1", SchemaVersion: autonomy.AutonomyEventContractV1,
		Type: autonomy.EventExecutionCompleted, ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		AutonomyRunID: "ar-1", WorkflowRunID: "wr-1", ExecutionID: "ex-1",
		CorrelationID: "corr-1", IdempotencyKey: "idem-exec-ex-1", Timestamp: time.Now().UTC(),
	}
}

func TestValidateAutonomyEventOK(t *testing.T) {
	if err := autonomy.ValidateAutonomyEvent(validEvent()); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownSchemaVersionRejected(t *testing.T) {
	e := validEvent()
	e.SchemaVersion = "99"
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("expected reject")
	}
}

func TestMissingSchemaVersionRejected(t *testing.T) {
	e := validEvent()
	e.SchemaVersion = ""
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("expected reject")
	}
}

func TestInvalidEventTypeRejected(t *testing.T) {
	e := validEvent()
	e.Type = "shell.execute"
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("expected reject")
	}
}

func TestMissingProjectRejected(t *testing.T) {
	e := validEvent()
	e.ProjectID = ""
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("expected reject")
	}
}

func TestMissingIdempotencyRejected(t *testing.T) {
	e := validEvent()
	e.IdempotencyKey = ""
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("expected reject")
	}
}

func TestPayloadSecretKeyRejected(t *testing.T) {
	e := validEvent()
	e.Payload = map[string]any{"api_key": "sk-secret"}
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("secret field must reject")
	}
}

func TestPayloadTooLongRejected(t *testing.T) {
	e := validEvent()
	long := make([]byte, autonomy.MaxAutonomyEventStringLen+10)
	for i := range long {
		long[i] = 'a'
	}
	e.Payload = map[string]any{"note": string(long)}
	if err := autonomy.ValidateAutonomyEvent(e); err == nil {
		t.Fatal("oversized value")
	}
}

func TestToEventObservationPreservesIdentity(t *testing.T) {
	e := validEvent()
	o, err := autonomy.ToEventObservation(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := autonomy.ValidateEventObservation(o); err != nil {
		t.Fatal(err)
	}
	if o.EventID != e.EventID || o.ProjectID != e.ProjectID || o.IdempotencyKey != e.IdempotencyKey {
		t.Fatal(o)
	}
	if o.Source != autonomy.ObservationSourceExecution {
		t.Fatal(o.Source)
	}
	if o.ContractVersion != autonomy.ObservationContractV1 {
		t.Fatal(o.ContractVersion)
	}
}

func TestSameLogicalEventProjectScoped(t *testing.T) {
	a := validEvent()
	b := validEvent()
	b.EventID = "ev-2"
	if !autonomy.SameLogicalEvent(a, b) {
		t.Fatal("same project+key should match")
	}
	b.ProjectID = "proj-b"
	if autonomy.SameLogicalEvent(a, b) {
		t.Fatal("different project must be independent")
	}
}

func TestToEnvelopeCompatible(t *testing.T) {
	e := validEvent()
	env := e.ToEnvelope()
	if err := env.Validate(); err != nil {
		t.Fatal(err)
	}
	if env.Type != contracts.EventType(autonomy.EventExecutionCompleted) {
		t.Fatal(env.Type)
	}
}

func TestPolicyBlockedIsInformationalOnly(t *testing.T) {
	e := validEvent()
	e.Type = autonomy.EventAutonomyPolicyBlocked
	e.IdempotencyKey = "idem-policy-1"
	o, err := autonomy.ToEventObservation(e)
	if err != nil {
		t.Fatal(err)
	}
	if o.Source != autonomy.ObservationSourcePolicy {
		t.Fatal(o.Source)
	}
	// Structural: conversion never returns grant-like fields
	if o.EventID == "" {
		t.Fatal("missing event id")
	}
}

func TestUnknownObservationVersionRejected(t *testing.T) {
	o, err := autonomy.ToEventObservation(validEvent())
	if err != nil {
		t.Fatal(err)
	}
	o.ContractVersion = "99"
	if err := autonomy.ValidateEventObservation(o); err == nil {
		t.Fatal("expected reject")
	}
}

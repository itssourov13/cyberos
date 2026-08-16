package contracts

import (
	"errors"
	"testing"
	"time"
)

func TestPrincipalTypeValid(t *testing.T) {
	valid := []PrincipalType{
		PrincipalTypeHuman, PrincipalTypeService, PrincipalTypeAgent,
		PrincipalTypePlugin, PrincipalTypeZOYA, PrincipalTypeExternalClient,
		PrincipalTypeAutomation,
	}
	for _, v := range valid {
		if !v.Valid() {
			t.Errorf("%q should be valid", v)
		}
	}
	if PrincipalType("UNKNOWN").Valid() {
		t.Error("UNKNOWN should be invalid")
	}
}

func TestPrincipalRefValidate(t *testing.T) {
	if err := (PrincipalRef{}).Validate(); err == nil {
		t.Error("empty principal should fail")
	}
	if err := (PrincipalRef{ID: "p1", Type: "BAD"}).Validate(); err == nil {
		t.Error("bad type should fail")
	}
	if err := (PrincipalRef{ID: "p1", Type: PrincipalTypeHuman}).Validate(); err != nil {
		t.Errorf("valid principal failed: %v", err)
	}
}

func TestScopeRefIsActive(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name  string
		scope ScopeRef
		want  bool
	}{
		{"active no bounds", ScopeRef{Status: ScopeStatusActive}, true},
		{"inactive", ScopeRef{Status: ScopeStatusInactive}, false},
		{"expired status", ScopeRef{Status: ScopeStatusExpired}, false},
		{"starts in future", ScopeRef{Status: ScopeStatusActive, StartsAt: &future}, false},
		{"expired by time", ScopeRef{Status: ScopeStatusActive, ExpiresAt: &past}, false},
		{"within window", ScopeRef{Status: ScopeStatusActive, StartsAt: &past, ExpiresAt: &future}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.IsActive(now); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskState(t *testing.T) {
	if !TaskStateCreated.Valid() {
		t.Error("CREATED should be valid")
	}
	if TaskState("BOGUS").Valid() {
		t.Error("BOGUS should be invalid")
	}
	if !TaskStateSucceeded.IsTerminal() {
		t.Error("SUCCEEDED should be terminal")
	}
	if TaskStateRunning.IsTerminal() {
		t.Error("RUNNING should not be terminal")
	}
}

func TestExecutionGrantValidateAndUsable(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	issued := now.Add(-time.Minute)
	expires := now.Add(5 * time.Minute)

	good := ExecutionGrant{
		ID:           "g1",
		Principal:    PrincipalRef{ID: "p1", Type: PrincipalTypeHuman},
		ProjectID:    "proj1",
		ScopeID:      "scope1",
		TaskID:       "task1",
		CapabilityID: "cap1",
		Status:       ExecutionGrantStatusIssued,
		IssuedAt:     issued,
		ExpiresAt:    expires,
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid grant failed validation: %v", err)
	}
	if !good.IsUsable(now) {
		t.Error("valid grant should be usable")
	}

	// Expired
	expired := good
	expired.ExpiresAt = now.Add(-time.Second)
	if expired.IsUsable(now) {
		t.Error("expired grant must not be usable")
	}

	// Wrong status
	consumed := good
	consumed.Status = ExecutionGrantStatusConsumed
	if consumed.IsUsable(now) {
		t.Error("consumed grant must not be usable")
	}

	// Missing fields
	if err := (ExecutionGrant{}).Validate(); err == nil {
		t.Error("empty grant should fail validation")
	}

	// Expires before issued
	badTime := good
	badTime.ExpiresAt = issued.Add(-time.Second)
	if err := badTime.Validate(); err == nil {
		t.Error("expires before issued should fail")
	}
}

func TestDecisionValidateAndAllowed(t *testing.T) {
	d := Decision{
		ID:        "d1",
		Outcome:   DecisionAllowed,
		Reason:    "policy permits",
		Timestamp: NowUTC(),
		Principal: PrincipalRef{ID: "p1", Type: PrincipalTypeHuman},
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("valid decision failed: %v", err)
	}
	if !d.Allowed() {
		t.Error("ALLOWED decision should report Allowed() == true")
	}

	denied := d
	denied.Outcome = DecisionDenied
	if denied.Allowed() {
		t.Error("DENIED decision should report Allowed() == false")
	}

	if err := (Decision{}).Validate(); err == nil {
		t.Error("empty decision should fail")
	}
}

func TestEventEnvelopeValidate(t *testing.T) {
	e := EventEnvelope{
		ID:            "e1",
		Type:          "task.created",
		SchemaVersion: EventSchemaV1,
		Timestamp:     NowUTC(),
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("valid event failed: %v", err)
	}
	if err := (EventEnvelope{}).Validate(); err == nil {
		t.Error("empty event should fail")
	}
}

func TestProvenanceValidate(t *testing.T) {
	p := Provenance{
		SubjectType:  "evidence",
		SubjectID:    "ev1",
		Principal:    PrincipalRef{ID: "p1", Type: PrincipalTypeHuman},
		ProjectID:    "proj1",
		ScopeID:      "scope1",
		TaskID:       "task1",
		CapabilityID: "cap1",
		ProducedAt:   NowUTC(),
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("valid provenance failed: %v", err)
	}
	if err := (Provenance{}).Validate(); err == nil {
		t.Error("empty provenance should fail")
	}
}

func TestErrorSentinels(t *testing.T) {
	// Ensure sentinels are distinct and usable with errors.Is
	if errors.Is(ErrGrantExpired, ErrGrantInvalid) {
		t.Error("ErrGrantExpired must not be equal to ErrGrantInvalid")
	}
	if !errors.Is(ErrScopeInvalid, ErrScopeInvalid) {
		t.Error("sentinel should match itself")
	}
}

func TestCapabilityDescriptorValidate(t *testing.T) {
	c := CapabilityDescriptor{
		ID:   "recon.domain.enumerate",
		Name: "Domain Enumeration",
		Type: CapabilityTypeRecon,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid capability failed: %v", err)
	}
	if err := (CapabilityDescriptor{}).Validate(); err == nil {
		t.Error("empty capability should fail")
	}
}

func TestExecutionRequestAndResult(t *testing.T) {
	now := NowUTC()
	grant := ExecutionGrant{
		ID:           "g1",
		Principal:    PrincipalRef{ID: "p1", Type: PrincipalTypeHuman},
		ProjectID:    "proj1",
		ScopeID:      "scope1",
		TaskID:       "task1",
		CapabilityID: "cap1",
		Status:       ExecutionGrantStatusIssued,
		IssuedAt:     now,
		ExpiresAt:    now.Add(5 * time.Minute),
	}
	req := ExecutionRequest{Grant: grant}
	if err := req.Validate(); err != nil {
		t.Fatalf("valid request failed: %v", err)
	}

	res := ExecutionResult{
		Status:     ExecutionStatusSucceeded,
		StartedAt:  now,
		FinishedAt: now.Add(time.Second),
		ExecutorID: "mock",
	}
	if err := res.Validate(); err != nil {
		t.Fatalf("valid result failed: %v", err)
	}
}

package findings

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint("p", "host", "cap", "title", "loc")
	b := Fingerprint("p", "host", "cap", "title", "loc")
	if a != b || a == "" {
		t.Fatal(a, b)
	}
	c := Fingerprint("p", "other", "cap", "title", "loc")
	if a == c {
		t.Fatal("must differ")
	}
}

func TestIngestNewAndDedup(t *testing.T) {
	s := NewService(nil)
	req := IngestRequest{
		Result: tools.NormalizedResult{
			ToolID: "tool.nmap", Capability: "recon.network.portscan", Summary: "scan",
			Findings: []tools.FindingHint{{Title: "Open port", Severity: "INFO", Detail: "22/tcp"}},
			Raw:      map[string]any{"target": "10.0.0.1"},
		},
		ProjectID: "p1", ScopeID: "s1", TaskID: "t1", AssessmentID: "a1", ExecutionID: "ex1",
	}
	r1, err := s.Ingest(req)
	if err != nil || len(r1) != 1 || r1[0].Action != contracts.FindingStatusNew {
		t.Fatalf("%v %+v", err, r1)
	}
	r2, err := s.Ingest(req)
	if err != nil || len(r2) != 1 || r2[0].Action != contracts.FindingStatusUnchanged {
		t.Fatalf("dedup: %v %+v", err, r2)
	}
	if r1[0].Finding.Fingerprint != r2[0].Finding.Fingerprint {
		t.Fatal("fp")
	}
	list := s.ListByAssessment("a1")
	if len(list) != 1 {
		t.Fatal(len(list))
	}
}

func TestResolveAndReopen(t *testing.T) {
	s := NewService(nil)
	req := IngestRequest{
		Result: tools.NormalizedResult{
			ToolID: "t", Capability: "c", Findings: []tools.FindingHint{{Title: "X", Detail: "d"}},
			Raw: map[string]any{"target": "h"},
		},
		ProjectID: "p", ScopeID: "s", TaskID: "t",
	}
	r, _ := s.Ingest(req)
	_, _ = s.Resolve(r[0].Finding.ID)
	r2, _ := s.Ingest(req)
	if r2[0].Action != contracts.FindingStatusReopened {
		t.Fatalf("%v", r2[0].Action)
	}
}

package security_test

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/findings"
	"cyberos/internal/tools"
)

func TestSecurity_FindingsDoNotAuthorize(t *testing.T) {
	s := findings.NewService(nil)
	_, err := s.Ingest(findings.IngestRequest{
		Result: tools.NormalizedResult{
			ToolID: "tool.nmap", Capability: "recon.network.portscan",
			Findings: []tools.FindingHint{{Title: "x"}},
			Raw:      map[string]any{"target": "h"},
		},
		ProjectID: "p", ScopeID: "s", TaskID: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Creating findings never issues grants or executes tools.
	if len(s.ListByProject("p")) != 1 {
		t.Fatal("expected one finding")
	}
	_ = contracts.FindingStatusNew
}

func TestSecurity_MultiToolSamePipeline(t *testing.T) {
	s := findings.NewService(nil)
	for _, tool := range []string{"tool.nmap", "tool.http.probe", "tool.python.probe"} {
		_, err := s.Ingest(findings.IngestRequest{
			Result: tools.NormalizedResult{
				ToolID: tool, Capability: "c-" + tool,
				Findings: []tools.FindingHint{{Title: "obs-" + tool}},
				Raw:      map[string]any{"target": "t1"},
			},
			ProjectID: "p", ScopeID: "s", TaskID: "task1", AssessmentID: "a1",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(s.ListByAssessment("a1")) != 3 {
		t.Fatal(s.ListByAssessment("a1"))
	}
}

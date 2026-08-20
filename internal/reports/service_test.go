package reports

import (
	"strings"
	"testing"

	"cyberos/internal/cases"
	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
)

func TestReportJSONAndMarkdown(t *testing.T) {
	svc := NewService()
	cs := cases.NewService()
	c, _ := cs.Create(cases.CreateRequest{ID: "c1", ProjectID: "p1", AssessmentID: "a1", Title: "Case"})
	r, err := svc.Create("r1", BuildInput{
		AssessmentID: "a1", ProjectID: "p1",
		Findings:    []map[string]any{{"id": "f1", "title": "Open port"}},
		Assets:      []map[string]any{{"id": "a", "value": "10.0.0.1"}},
		Cases:       []*cases.Case{c},
		RiskSummary: riskprio.Summary{Total: 1, HighestScore: 90, ByPriority: map[string]int{"CRITICAL": 1}, ModelVersion: riskprio.ModelVersion},
		Timeline:    []string{"assessment started", "finding created"},
	})
	if err != nil || r.SchemaVersion != SchemaVersion {
		t.Fatal(err, r)
	}
	jb, err := svc.Export("r1", contracts.ReportFormatJSON)
	if err != nil || !strings.Contains(string(jb), SchemaVersion) {
		t.Fatal(err, string(jb))
	}
	mb, err := svc.Export("r1", contracts.ReportFormatMarkdown)
	if err != nil || !strings.Contains(string(mb), "Executive Summary") {
		t.Fatal(err, string(mb))
	}
}

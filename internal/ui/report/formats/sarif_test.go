// # internal/ui/report/formats/sarif_test.go
package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateSARIF_EmptyResults(t *testing.T) {
	data, err := GenerateSARIF("", nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateSARIF returned error: %v", err)
	}
	var report sarifReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if report.Schema != sarifSchema {
		t.Errorf("$schema = %q, want %q", report.Schema, sarifSchema)
	}
	if report.Version != sarifVersion {
		t.Errorf("version = %q, want %q", report.Version, sarifVersion)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(report.Runs))
	}
	if len(report.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(report.Runs[0].Results))
	}
}

func TestGenerateSARIF_SingleCycle(t *testing.T) {
	cycles := [][]string{{"a", "b", "a"}}
	data, err := GenerateSARIF("/project", cycles, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report sarifReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	results := report.Runs[0].Results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.RuleID != ruleIDCycle {
		t.Errorf("ruleId = %q, want %q", r.RuleID, ruleIDCycle)
	}
	if r.Level != "error" {
		t.Errorf("level = %q, want error", r.Level)
	}
	if !strings.Contains(r.Message.Text, "Circular dependency") {
		t.Errorf("message text %q does not contain 'Circular dependency'", r.Message.Text)
	}
	if len(results[0].Locations) == 0 {
		t.Error("expected at least one location for cycle result")
	}
}

func TestGenerateSARIF_SecretUsesRelativeURI(t *testing.T) {
	secrets := []parser.Secret{
		{
			Kind:       "aws-access-key-id",
			Severity:   "high",
			Value:      "AKIAIOSFODNN7EXAMPLE",
			Confidence: 0.99,
			Location: parser.Location{
				File:   "/project/internal/config/secrets.go",
				Line:   42,
				Column: 5,
			},
		},
	}
	data, err := GenerateSARIF("/project", nil, nil, secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report sarifReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	results := report.Runs[0].Results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.RuleID != ruleIDSecret {
		t.Errorf("ruleId = %q, want %q", r.RuleID, ruleIDSecret)
	}
	if r.Level != "error" { // high → error
		t.Errorf("level = %q, want error", r.Level)
	}

	if len(r.Locations) == 0 {
		t.Fatal("expected location on secret result")
	}
	uri := r.Locations[0].PhysicalLocation.ArtifactLocation.URI
	if strings.Contains(uri, "/project") {
		t.Errorf("URI %q should be relative, not absolute", uri)
	}
	if uri != "internal/config/secrets.go" {
		t.Errorf("URI = %q, want internal/config/secrets.go", uri)
	}
	if r.Locations[0].PhysicalLocation.ArtifactLocation.URIBaseID != "%SRCROOT%" {
		t.Errorf("uriBaseId should be %%SRCROOT%%")
	}
	region := r.Locations[0].PhysicalLocation.Region
	if region == nil || region.StartLine != 42 {
		t.Errorf("expected region.startLine = 42")
	}
}

func TestGenerateSARIF_ArchViolation(t *testing.T) {
	violations := []graph.ArchitectureViolation{
		{
			RuleName:   "no-internal-from-api",
			FromModule: "api",
			FromLayer:  "api",
			ToModule:   "internal/data",
			ToLayer:    "data",
			File:       "/project/api/handler.go",
			Line:       10,
		},
	}
	data, err := GenerateSARIF("/project", nil, violations, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report sarifReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	results := report.Runs[0].Results
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].RuleID != ruleIDViolation {
		t.Errorf("ruleId = %q, want %q", results[0].RuleID, ruleIDViolation)
	}
}

func TestRelativeURI(t *testing.T) {
	cases := []struct {
		root    string
		path    string
		wantURI string
	}{
		{"/project", "/project/internal/foo.go", "internal/foo.go"},
		{"/project", "/other/bar.go", "../other/bar.go"},
		{"", "/abs/path.go", "/abs/path.go"},
		{"/project", "relative/path.go", "relative/path.go"},
	}
	for _, tc := range cases {
		got := relativeURI(tc.root, tc.path)
		if got != tc.wantURI {
			t.Errorf("relativeURI(%q, %q) = %q, want %q", tc.root, tc.path, got, tc.wantURI)
		}
	}
}

func TestSecretSeverityToLevel(t *testing.T) {
	cases := []struct{ sev, want string }{
		{"critical", "error"},
		{"high", "error"},
		{"medium", "warning"},
		{"low", "note"},
		{"", "note"},
	}
	for _, tc := range cases {
		got := secretSeverityToLevel(tc.sev)
		if got != tc.want {
			t.Errorf("severity %q → level %q, want %q", tc.sev, got, tc.want)
		}
	}
}

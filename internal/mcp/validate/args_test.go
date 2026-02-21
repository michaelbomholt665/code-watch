package validate

import (
	"circular/internal/mcp/contracts"
	"reflect"
	"strings"
	"testing"
)

func TestParseToolArgs_ScanRun(t *testing.T) {
	raw := map[string]any{
		"operation": "scan.run",
		"params": map[string]any{
			"paths": []any{"./a", "./b", "./a"},
		},
	}

	op, input, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationScanRun {
		t.Fatalf("expected operation %s, got %s", contracts.OperationScanRun, op)
	}

	scanInput, ok := input.(contracts.ScanRunInput)
	if !ok {
		t.Fatalf("expected ScanRunInput, got %T", input)
	}
	if len(scanInput.Paths) != 2 {
		t.Fatalf("expected deduped paths, got %v", scanInput.Paths)
	}
}

func TestParseToolArgs_SecretsScan(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationSecretsScan),
		"params": map[string]any{
			"paths": []any{"./a", "./b", "./a"},
		},
	}

	op, input, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationSecretsScan {
		t.Fatalf("expected operation %s, got %s", contracts.OperationSecretsScan, op)
	}

	scanInput, ok := input.(contracts.SecretsScanInput)
	if !ok {
		t.Fatalf("expected SecretsScanInput, got %T", input)
	}
	if len(scanInput.Paths) != 2 {
		t.Fatalf("expected deduped paths, got %v", scanInput.Paths)
	}
}

func TestParseToolArgs_SecretsList_InvalidLimit(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationSecretsList),
		"params": map[string]any{
			"limit": 9001,
		},
	}

	_, _, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseToolArgs_InvalidOperation(t *testing.T) {
	raw := map[string]any{"operation": "nope"}
	_, _, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateToolArgs_ModuleDetails(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationQueryDetails),
		"params":    map[string]any{"module": "mod"},
	}
	input, err := ValidateToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := contracts.QueryModuleDetailsInput{Module: "mod"}
	if !reflect.DeepEqual(input, expected) {
		t.Fatalf("expected %v, got %v", expected, input)
	}
}

func TestParseToolArgs_GraphSyncDiagrams(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationGraphSyncDiag),
		"params": map[string]any{
			"formats": []any{"dot", "mermaid", "bad", "dot"},
		},
	}

	op, input, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationGraphSyncDiag {
		t.Fatalf("expected operation %s, got %s", contracts.OperationGraphSyncDiag, op)
	}
	got, ok := input.(contracts.SystemSyncOutputsInput)
	if !ok {
		t.Fatalf("expected SystemSyncOutputsInput, got %T", input)
	}
	expected := []string{"dot", "mermaid"}
	if !reflect.DeepEqual(got.Formats, expected) {
		t.Fatalf("expected formats %v, got %v", expected, got.Formats)
	}
}

func TestParseToolArgs_LegacySystemSyncOutputsAlias(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationSystemSyncOut),
	}
	op, _, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationGraphSyncDiag {
		t.Fatalf("expected canonical operation %s, got %s", contracts.OperationGraphSyncDiag, op)
	}
}

func TestParseToolArgs_SystemGenerateScript(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationSystemGenScript),
	}

	op, _, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationSystemGenScript {
		t.Fatalf("expected operation %s, got %s", contracts.OperationSystemGenScript, op)
	}
}

func TestParseToolArgs_ReportGenerateMarkdown(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationReportGenMD),
		"params": map[string]any{
			"write_file": true,
			"path":       "docs/reports/analysis.md",
			"verbosity":  "DETAILED",
		},
	}

	op, input, err := ParseToolArgs(contracts.ToolNameCircular, raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationReportGenMD {
		t.Fatalf("expected operation %s, got %s", contracts.OperationReportGenMD, op)
	}
	got, ok := input.(contracts.ReportGenerateMarkdownInput)
	if !ok {
		t.Fatalf("expected ReportGenerateMarkdownInput, got %T", input)
	}
	if !got.WriteFile || got.Path != "docs/reports/analysis.md" || got.Verbosity != "detailed" {
		t.Fatalf("unexpected parsed input: %+v", got)
	}
}

func TestParseToolArgs_PathTraversal(t *testing.T) {
	raw := map[string]any{
		"operation": "scan.run",
		"params": map[string]any{
			"paths": []any{"../../etc/passwd"},
		},
	}

	_, _, err := ParseToolArgs(contracts.ToolNameCircular, raw, "/home/user/project")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "escapes project root") {
		t.Fatalf("expected path traversal error message, got: %v", err)
	}
}

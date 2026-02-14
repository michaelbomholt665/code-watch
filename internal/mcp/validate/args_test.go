package validate

import (
	"circular/internal/mcp/contracts"
	"reflect"
	"testing"
)

func TestParseToolArgs_ScanRun(t *testing.T) {
	raw := map[string]any{
		"operation": "scan.run",
		"params": map[string]any{
			"paths": []any{"./a", "./b", "./a"},
		},
	}

	op, input, err := ParseToolArgs(contracts.ToolNameCircular, raw)
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

func TestParseToolArgs_InvalidOperation(t *testing.T) {
	raw := map[string]any{"operation": "nope"}
	_, _, err := ParseToolArgs(contracts.ToolNameCircular, raw)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateToolArgs_ModuleDetails(t *testing.T) {
	raw := map[string]any{
		"operation": string(contracts.OperationQueryDetails),
		"params":    map[string]any{"module": "mod"},
	}
	input, err := ValidateToolArgs(contracts.ToolNameCircular, raw)
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

	op, input, err := ParseToolArgs(contracts.ToolNameCircular, raw)
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
	op, _, err := ParseToolArgs(contracts.ToolNameCircular, raw)
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

	op, _, err := ParseToolArgs(contracts.ToolNameCircular, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != contracts.OperationSystemGenScript {
		t.Fatalf("expected operation %s, got %s", contracts.OperationSystemGenScript, op)
	}
}

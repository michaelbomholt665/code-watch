package config

import (
	"fmt"
	"os"
	"testing"
)

func TestValidateOutputConflicts(t *testing.T) {
	cfg := &Config{
		Output: Output{
			DOT:      "graph.dot",
			TSV:      "graph.dot", // Conflict
			Paths:    OutputPaths{DiagramsDir: "docs"},
			Report:   ReportOutput{Verbosity: "standard"},
			Formats:  DiagramFormats{},
			Diagrams: DiagramOutput{FlowConfig: FlowDiagramConfig{MaxDepth: 5}},
		},
	}
	errs := Validate(cfg)
	found := false
	for _, err := range errs {
		if err.Error() == `output conflict: output.dot and output.tsv share the same path "graph.dot"` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected output conflict error, got %v", errs)
	}
}

func TestValidateGrammarsPath(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "circular-test")
	defer os.Remove(tmpFile.Name())

	cfg := &Config{
		GrammarsPath: tmpFile.Name(), // File, not directory
	}
	errs := Validate(cfg)
	found := false
	targetError := fmt.Sprintf("grammars_path %q is not a directory", tmpFile.Name())
	for _, err := range errs {
		if err.Error() == targetError {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected grammar path error, got %v", errs)
	}
}

func TestValidateWatchPaths(t *testing.T) {
	cfg := &Config{
		WatchPaths: []string{"/non/existent/path"},
	}
	errs := Validate(cfg)
	found := false
	targetError := "watch_paths[0] \"/non/existent/path\" does not exist"
	for _, err := range errs {
		if err.Error() == targetError {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected watch path error %q, got %v", targetError, errs)
	}
}

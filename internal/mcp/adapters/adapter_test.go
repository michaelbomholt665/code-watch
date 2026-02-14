package adapters

import (
	coreapp "circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/engine/parser"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestResolveDiagramPath_SeparatorAware(t *testing.T) {
	t.Parallel()

	root := filepath.Join("workspace", "repo")
	diagramsDir := filepath.Join(root, "docs", "diagrams")

	if got := resolveDiagramPath("graph.mmd", root, diagramsDir); got != filepath.Join(diagramsDir, "graph.mmd") {
		t.Fatalf("expected filename output under diagrams dir, got %q", got)
	}
	if got := resolveDiagramPath("docs/graph.mmd", root, diagramsDir); got != filepath.Join(root, "docs", "graph.mmd") {
		t.Fatalf("expected slash path output under root, got %q", got)
	}
	if got := resolveDiagramPath(`docs\graph.mmd`, root, diagramsDir); got != filepath.Join(root, `docs\graph.mmd`) {
		t.Fatalf("expected backslash path output under root, got %q", got)
	}
}

func TestCLIAndMCPParity_SummaryAndOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Output: config.Output{
			DOT: "graph.dot",
			TSV: "dependencies.tsv",
		},
		Alerts: config.Alerts{Terminal: false},
	}
	appInstance, err := coreapp.NewWithDependencies(cfg, coreapp.Dependencies{
		CodeParser: stubCodeParser{},
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	appInstance.Graph.AddFile(&parser.File{
		Path:   "a.go",
		Module: "app/a",
		Imports: []parser.Import{
			{Module: "app/b"},
		},
		Secrets: []parser.Secret{
			{
				Kind:     "token",
				Severity: "high",
				Value:    "supersecret",
				Location: parser.Location{File: "a.go", Line: 10, Column: 5},
			},
		},
	})
	appInstance.Graph.AddFile(&parser.File{
		Path:   "b.go",
		Module: "app/b",
		Imports: []parser.Import{
			{Module: "app/a"},
		},
	})

	analysis := appInstance.AnalysisService()
	adapter := NewAdapter(analysis, nil, "default")

	snapshot, err := analysis.SummarySnapshot(context.Background())
	if err != nil {
		t.Fatalf("summary snapshot: %v", err)
	}

	cyclesOut, err := adapter.Cycles(context.Background(), 0)
	if err != nil {
		t.Fatalf("mcp cycles: %v", err)
	}
	if cyclesOut.CycleCount != len(snapshot.Cycles) {
		t.Fatalf("cycle count mismatch: cli=%d mcp=%d", len(snapshot.Cycles), cyclesOut.CycleCount)
	}
	if !reflect.DeepEqual(cyclesOut.Cycles, snapshot.Cycles) {
		t.Fatalf("cycle contract mismatch: cli=%v mcp=%v", snapshot.Cycles, cyclesOut.Cycles)
	}

	secretsOut, err := adapter.ListSecrets(context.Background(), 0)
	if err != nil {
		t.Fatalf("mcp secrets: %v", err)
	}
	if secretsOut.SecretCount != snapshot.SecretCount {
		t.Fatalf("secret count mismatch: cli=%d mcp=%d", snapshot.SecretCount, secretsOut.SecretCount)
	}

	cliOutputs, err := analysis.SyncOutputs(context.Background(), ports.SyncOutputsRequest{Formats: []string{"dot", "tsv"}})
	if err != nil {
		t.Fatalf("cli sync outputs: %v", err)
	}
	mcpOutputs, err := adapter.SyncOutputs(context.Background(), []string{"dot", "tsv"})
	if err != nil {
		t.Fatalf("mcp sync outputs: %v", err)
	}

	sort.Strings(cliOutputs.Written)
	sort.Strings(mcpOutputs)
	if !reflect.DeepEqual(cliOutputs.Written, mcpOutputs) {
		t.Fatalf("output paths mismatch: cli=%v mcp=%v", cliOutputs.Written, mcpOutputs)
	}
}

type stubCodeParser struct{}

func (stubCodeParser) ParseFile(_ string, _ []byte) (*parser.File, error) { return nil, nil }
func (stubCodeParser) IsSupportedPath(_ string) bool                      { return true }
func (stubCodeParser) IsTestFile(_ string) bool                           { return false }
func (stubCodeParser) SupportedExtensions() []string                      { return []string{".go"} }
func (stubCodeParser) SupportedFilenames() []string                       { return nil }
func (stubCodeParser) SupportedTestFileSuffixes() []string                { return nil }

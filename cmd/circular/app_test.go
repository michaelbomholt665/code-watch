// # cmd/circular/app_test.go
package main

import (
	"circular/internal/config"
	"circular/internal/graph"
	"circular/internal/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApp(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "apptest")
	defer os.RemoveAll(tmpDir)

	// Create a dummy Go file
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nimport \"fmt\"\nfunc Main() { fmt.Println(\"hi\") }"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/app"), 0644)

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{tmpDir},
		Output: config.Output{
			DOT: filepath.Join(tmpDir, "graph.dot"),
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
		Alerts: config.Alerts{Terminal: true},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Test InitialScan
	err = app.InitialScan()
	if err != nil {
		t.Fatal(err)
	}

	if len(app.Graph.GetAllFiles()) != 1 {
		t.Errorf("Expected 1 file, got %d", len(app.Graph.GetAllFiles()))
	}

	// Test GenerateOutputs
	err = app.GenerateOutputs(nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cfg.Output.DOT); os.IsNotExist(err) {
		t.Error("DOT file was not generated")
	}
	if _, err := os.Stat(cfg.Output.TSV); os.IsNotExist(err) {
		t.Error("TSV file was not generated")
	}

	// Test HandleChanges
	app.HandleChanges([]string{filepath.Join(tmpDir, "main.go")})
	// Should not crash and should re-process
}

func TestApp_GenerateOutputs_IncludesUnusedImportRows(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "appunused")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/appunused"), 0644)

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{tmpDir},
		Output: config.Output{
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
		Alerts: config.Alerts{Terminal: false},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.InitialScan(); err != nil {
		t.Fatal(err)
	}

	unused := app.AnalyzeUnusedImports()
	if len(unused) == 0 {
		t.Fatal("expected at least one unused import")
	}

	if err := app.GenerateOutputs(nil, unused, app.Graph.ComputeModuleMetrics(), nil, nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cfg.Output.TSV)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, "unused_import") {
		t.Fatalf("expected unused_import rows in TSV output, got: %s", out)
	}
	if !strings.Contains(out, "Type\tFile\tLanguage\tModule\tAlias\tItem\tLine\tColumn\tConfidence") {
		t.Fatalf("expected unused import header in TSV output, got: %s", out)
	}
}

func TestApp_TraceImportChain(t *testing.T) {
	app := &App{Graph: graph.NewGraph()}
	app.Graph.AddFile(&parser.File{Path: "a.go", Module: "A", Imports: []parser.Import{{Module: "B"}}})
	app.Graph.AddFile(&parser.File{Path: "b.go", Module: "B", Imports: []parser.Import{{Module: "C"}}})
	app.Graph.AddFile(&parser.File{Path: "c.go", Module: "C"})

	out, err := app.TraceImportChain("A", "C")
	if err != nil {
		t.Fatalf("expected trace success, got error: %v", err)
	}

	if !strings.Contains(out, "Import chain: A -> C") {
		t.Fatalf("expected trace header, got: %s", out)
	}
	if !strings.Contains(out, "A\n  -> B\n  -> C") {
		t.Fatalf("expected chain body, got: %s", out)
	}
}

func TestApp_TraceImportChain_Errors(t *testing.T) {
	app := &App{Graph: graph.NewGraph()}
	app.Graph.AddFile(&parser.File{Path: "a.go", Module: "A"})
	app.Graph.AddFile(&parser.File{Path: "b.go", Module: "B"})

	tests := []struct {
		name       string
		from       string
		to         string
		errContain string
	}{
		{
			name:       "missing source",
			from:       "missing",
			to:         "A",
			errContain: "source module not found",
		},
		{
			name:       "missing target",
			from:       "A",
			to:         "missing",
			errContain: "target module not found",
		},
		{
			name:       "no path",
			from:       "A",
			to:         "B",
			errContain: "no import chain found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := app.TraceImportChain(tc.from, tc.to)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContain) {
				t.Fatalf("expected error to contain %q, got %q", tc.errContain, err.Error())
			}
		})
	}
}

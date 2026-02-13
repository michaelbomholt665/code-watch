// # cmd/circular/app_test.go
package app

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
	tmpDir, err := os.MkdirTemp("", "apptest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy Go file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nimport \"fmt\"\nfunc Main() { fmt.Println(\"hi\") }"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/app"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{tmpDir},
		Output: config.Output{
			DOT: filepath.Join(tmpDir, "graph.dot"),
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
		Alerts: config.Alerts{Terminal: true},
	}

	app, err := New(cfg)
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
	tmpDir, err := os.MkdirTemp("", "appunused")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/appunused"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{tmpDir},
		Output: config.Output{
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
		Alerts: config.Alerts{Terminal: false},
	}

	app, err := New(cfg)
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

func TestApp_GenerateOutputs_MermaidPlantUMLAndMarkdownInjection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "app-diagrams")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte("package main\nimport \"github.com/acme/b\"\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/acme/a"), 0644); err != nil {
		t.Fatal(err)
	}

	readmePath := filepath.Join(tmpDir, "README.md")
	readmeContent := strings.Join([]string{
		"# Test",
		"<!-- circular:deps-mermaid:start -->",
		"old",
		"<!-- circular:deps-mermaid:end -->",
		"<!-- circular:deps-plantuml:start -->",
		"old",
		"<!-- circular:deps-plantuml:end -->",
		"",
	}, "\n")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{tmpDir},
		Output: config.Output{
			DOT:      filepath.Join(tmpDir, "graph.dot"),
			TSV:      filepath.Join(tmpDir, "dependencies.tsv"),
			Mermaid:  filepath.Join(tmpDir, "graph.mmd"),
			PlantUML: filepath.Join(tmpDir, "graph.puml"),
			UpdateMarkdown: []config.MarkdownInjection{
				{File: readmePath, Marker: "deps-mermaid", Format: "mermaid"},
				{File: readmePath, Marker: "deps-plantuml", Format: "plantuml"},
			},
		},
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.InitialScan(); err != nil {
		t.Fatal(err)
	}

	if err := app.GenerateOutputs(nil, nil, app.Graph.ComputeModuleMetrics(), nil, nil); err != nil {
		t.Fatal(err)
	}

	mmd, err := os.ReadFile(cfg.Output.Mermaid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mmd), "flowchart LR") {
		t.Fatalf("expected mermaid flowchart output, got: %s", string(mmd))
	}

	puml, err := os.ReadFile(cfg.Output.PlantUML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(puml), "@startuml") {
		t.Fatalf("expected plantuml output, got: %s", string(puml))
	}

	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(readme)
	if !strings.Contains(content, "```mermaid") {
		t.Fatalf("expected injected mermaid fenced block, got: %s", content)
	}
	if !strings.Contains(content, "```plantuml") {
		t.Fatalf("expected injected plantuml fenced block, got: %s", content)
	}
}

func TestApp_GenerateOutputs_DiagramPathsUseDetectedRootAndDiagramsDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "app-diagram-paths")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/paths"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{tmpDir},
		Output: config.Output{
			Mermaid:  "graph.mmd",
			PlantUML: "graph.puml",
			Paths: config.OutputPaths{
				DiagramsDir: "docs/diagrams",
			},
		},
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.InitialScan(); err != nil {
		t.Fatal(err)
	}

	if err := app.GenerateOutputs(nil, nil, app.Graph.ComputeModuleMetrics(), nil, nil); err != nil {
		t.Fatal(err)
	}

	mermaidPath := filepath.Join(tmpDir, "docs", "diagrams", "graph.mmd")
	plantumlPath := filepath.Join(tmpDir, "docs", "diagrams", "graph.puml")
	if _, err := os.Stat(mermaidPath); err != nil {
		t.Fatalf("expected mermaid output at %q, err=%v", mermaidPath, err)
	}
	if _, err := os.Stat(plantumlPath); err != nil {
		t.Fatalf("expected plantuml output at %q, err=%v", plantumlPath, err)
	}
}

func TestUniqueScanRoots_DeduplicatesRelativeAndAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	roots := uniqueScanRoots([]string{".", tmpDir, "./"})
	if len(roots) != 1 {
		t.Fatalf("expected 1 unique root, got %d (%v)", len(roots), roots)
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

func TestApp_ProcessFile_PythonWithoutWatchPathReturnsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apppy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pythonFile := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(pythonFile, []byte("import os\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		Output: config.Output{
			DOT: filepath.Join(tmpDir, "graph.dot"),
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = app.ProcessFile(pythonFile)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "python resolver requires at least one watch path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApp_ProcessFile_PythonUsesContainingWatchPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apppy-watch-path")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	watchPathOne := filepath.Join(tmpDir, "watch-one")
	watchPathTwo := filepath.Join(tmpDir, "watch-two")
	pythonDir := filepath.Join(watchPathTwo, "pkg")
	pythonFile := filepath.Join(pythonDir, "main.py")

	if err := os.MkdirAll(pythonDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pythonDir, "__init__.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pythonFile, []byte("import os\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{watchPathOne, watchPathTwo},
		Output: config.Output{
			DOT: filepath.Join(tmpDir, "graph.dot"),
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := app.ProcessFile(pythonFile); err != nil {
		t.Fatal(err)
	}

	gotFile, ok := app.Graph.GetFile(pythonFile)
	if !ok {
		t.Fatalf("expected processed file %q in graph", pythonFile)
	}
	if gotFile.Module != "pkg.main" {
		t.Fatalf("expected module pkg.main, got %q", gotFile.Module)
	}
}

func TestApp_ProcessFile_PythonOutsideWatchPathsReturnsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "apppy-outside-watch-path")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	watchPath := filepath.Join(tmpDir, "watch")
	outsideDir := filepath.Join(tmpDir, "outside")
	pythonFile := filepath.Join(outsideDir, "main.py")

	if err := os.MkdirAll(watchPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pythonFile, []byte("import os\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GrammarsPath: "./grammars",
		WatchPaths:   []string{watchPath},
		Output: config.Output{
			DOT: filepath.Join(tmpDir, "graph.dot"),
			TSV: filepath.Join(tmpDir, "dependencies.tsv"),
		},
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = app.ProcessFile(pythonFile)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not under any configured watch path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApp_ResolveGoModule_CacheRelErrorReturnsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "app-go-mod-rel-error")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "pkg", "file.go")
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &App{
		goModCache: make(map[string]goModuleCacheEntry),
	}
	app.goModCache[filepath.Dir(filePath)] = goModuleCacheEntry{
		Found:      true,
		ModuleRoot: "relative-root",
		ModulePath: "example.com/project",
	}

	moduleName, ok, err := app.resolveGoModule(filePath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ok {
		t.Fatal("expected ok=false on resolve error")
	}
	if moduleName != "" {
		t.Fatalf("expected empty module name on resolve error, got %q", moduleName)
	}
	if !strings.Contains(err.Error(), "resolve module name from cache entry") {
		t.Fatalf("expected wrapped cache context in error, got: %v", err)
	}
}

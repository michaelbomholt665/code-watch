package app

import (
	"circular/internal/core/config"
	"circular/internal/engine/parser"
	"os"
	"path/filepath"
	"testing"
)

type stubCodeParser struct {
	parsedFile *parser.File
}

func (s stubCodeParser) ParseFile(path string, content []byte) (*parser.File, error) {
	file := *s.parsedFile
	file.Path = path
	return &file, nil
}

func (s stubCodeParser) GetLanguage(path string) string {
	return "stub"
}

func (s stubCodeParser) IsSupportedPath(filePath string) bool {
	return filepath.Ext(filePath) == ".stub"
}

func (s stubCodeParser) IsTestFile(path string) bool {
	return false
}

func (s stubCodeParser) SupportedExtensions() []string {
	return []string{".stub"}
}

func (s stubCodeParser) SupportedFilenames() []string {
	return nil
}

func (s stubCodeParser) SupportedTestFileSuffixes() []string {
	return nil
}

func TestNewWithDependencies_RequiresCodeParser(t *testing.T) {
	cfg := &config.Config{}
	_, err := NewWithDependencies(cfg, Dependencies{})
	if err == nil {
		t.Fatal("expected missing code parser dependency error")
	}
}

func TestNewWithDependencies_UsesInjectedCodeParser(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.stub")
	if err := os.WriteFile(filePath, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		WatchPaths: []string{tmpDir},
	}
	app, err := NewWithDependencies(cfg, Dependencies{
		CodeParser: stubCodeParser{
			parsedFile: &parser.File{
				Language: "stub",
				Module:   "example.stub",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	files, err := app.ScanDirectories([]string{tmpDir}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != filePath {
		t.Fatalf("expected only stub file, got %v", files)
	}

	if err := app.ProcessFile(filePath); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.Graph.GetFile(filePath); !ok {
		t.Fatalf("expected file %q in graph", filePath)
	}
}

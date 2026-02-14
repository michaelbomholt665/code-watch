package adapters

import (
	"path/filepath"
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

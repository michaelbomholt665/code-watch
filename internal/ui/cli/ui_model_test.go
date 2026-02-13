package cli

import (
	"circular/internal/data/query"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_FilterAndFocusFlow(t *testing.T) {
	m := initialModel(nil, nil)

	updated, _ := m.Update(updateMsg{
		cycles: [][]string{{"a", "b"}},
		hallucinations: []resolver.UnresolvedReference{
			{File: "main.go"},
		},
		modules: []query.ModuleSummary{
			{Name: "app/a", FileCount: 1, ExportCount: 2, DependencyCount: 1, ReverseDependencyCount: 0},
			{Name: "app/b", FileCount: 2, ExportCount: 1, DependencyCount: 0, ReverseDependencyCount: 1},
		},
		moduleCount: 2,
		fileCount:   3,
	})

	state, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model type, got %T", updated)
	}
	if len(state.issueList.Items()) != 2 {
		t.Fatalf("expected 2 issue items, got %d", len(state.issueList.Items()))
	}
	if len(state.moduleList.Items()) != 2 {
		t.Fatalf("expected 2 module items, got %d", len(state.moduleList.Items()))
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyTab})
	state = updated.(model)
	if state.mode != panelModules {
		t.Fatalf("expected module panel after tab, got %v", state.mode)
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyTab})
	state = updated.(model)
	if state.mode != panelIssues {
		t.Fatalf("expected issues panel after second tab, got %v", state.mode)
	}
}

func TestModel_ModuleDrillDownAndTrendToggle(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "app/a",
		Imports: []parser.Import{
			{Module: "app/b", Location: parser.Location{Line: 7, Column: 3}},
		},
		Definitions: []parser.Definition{
			{Name: "ExportedA", Exported: true},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "app/b",
	})

	m := initialModel(query.NewService(g, nil, "default"), nil)
	updated, _ := m.Update(updateMsg{
		modules: []query.ModuleSummary{
			{Name: "app/a", FileCount: 1, ExportCount: 1, DependencyCount: 1},
			{Name: "app/b", FileCount: 1, ExportCount: 0, DependencyCount: 0, ReverseDependencyCount: 1},
		},
		moduleCount: 2,
		fileCount:   2,
	})
	state := updated.(model)

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyTab})
	state = updated.(model)
	if state.mode != panelModules {
		t.Fatalf("expected module panel, got %v", state.mode)
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEnter})
	state = updated.(model)
	if !state.hasModuleDetails {
		t.Fatal("expected module details to open")
	}
	if len(state.moduleDetails.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(state.moduleDetails.Dependencies))
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	state = updated.(model)
	if !state.showTrend {
		t.Fatal("expected trend overlay toggled on")
	}

	updated, _ = state.Update(tea.KeyMsg{Type: tea.KeyEsc})
	state = updated.(model)
	if state.hasModuleDetails {
		t.Fatal("expected module details to close on esc")
	}
}

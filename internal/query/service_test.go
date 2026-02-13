package query

import (
	"circular/internal/graph"
	"circular/internal/history"
	"circular/internal/parser"
	"context"
	"strings"
	"testing"
	"time"
)

type fakeHistoryStore struct {
	snapshots []history.Snapshot
	err       error
}

func (f fakeHistoryStore) LoadSnapshots(since time.Time) ([]history.Snapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]history.Snapshot, 0, len(f.snapshots))
	for _, snapshot := range f.snapshots {
		if !since.IsZero() && snapshot.Timestamp.Before(since) {
			continue
		}
		out = append(out, snapshot)
	}
	return out, nil
}

func seedGraph() *graph.Graph {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "app/a",
		Imports: []parser.Import{
			{Module: "app/b", Location: parser.Location{Line: 3, Column: 1}},
		},
		Definitions: []parser.Definition{
			{Name: "ExportedA", Exported: true},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "app/b",
		Imports: []parser.Import{
			{Module: "app/c", Location: parser.Location{Line: 4, Column: 1}},
		},
		Definitions: []parser.Definition{
			{Name: "ExportedB", Exported: true},
		},
	})
	g.AddFile(&parser.File{
		Path:   "c.go",
		Module: "app/c",
	})
	return g
}

func TestService_ListModules(t *testing.T) {
	svc := NewService(seedGraph(), nil)
	got, err := svc.ListModules(context.Background(), "app/", 0)
	if err != nil {
		t.Fatalf("list modules: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(got))
	}
	if got[0].Name != "app/a" || got[1].Name != "app/b" || got[2].Name != "app/c" {
		t.Fatalf("unexpected ordering: %+v", got)
	}
}

func TestService_ModuleDetails(t *testing.T) {
	svc := NewService(seedGraph(), nil)
	details, err := svc.ModuleDetails(context.Background(), "app/b")
	if err != nil {
		t.Fatalf("module details: %v", err)
	}
	if len(details.Dependencies) != 1 || details.Dependencies[0].To != "app/c" {
		t.Fatalf("unexpected dependencies: %+v", details.Dependencies)
	}
	if len(details.ReverseDependencies) != 1 || details.ReverseDependencies[0] != "app/a" {
		t.Fatalf("unexpected reverse dependencies: %+v", details.ReverseDependencies)
	}
}

func TestService_DependencyTrace(t *testing.T) {
	svc := NewService(seedGraph(), nil)
	trace, err := svc.DependencyTrace(context.Background(), "app/a", "app/c", 4)
	if err != nil {
		t.Fatalf("dependency trace: %v", err)
	}
	if trace.Depth != 2 {
		t.Fatalf("expected depth=2, got %d", trace.Depth)
	}
	path := strings.Join(trace.Path, " -> ")
	if path != "app/a -> app/b -> app/c" {
		t.Fatalf("unexpected trace path: %s", path)
	}
}

func TestService_TrendSlice(t *testing.T) {
	base := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)
	store := fakeHistoryStore{
		snapshots: []history.Snapshot{
			{Timestamp: base.Add(-48 * time.Hour), ModuleCount: 2},
			{Timestamp: base.Add(-12 * time.Hour), ModuleCount: 3},
			{Timestamp: base, ModuleCount: 4},
		},
	}

	svc := NewService(seedGraph(), store)
	slice, err := svc.TrendSlice(context.Background(), base.Add(-24*time.Hour), 1)
	if err != nil {
		t.Fatalf("trend slice: %v", err)
	}
	if slice.ScanCount != 1 {
		t.Fatalf("expected 1 snapshot after filtering and limit, got %d", slice.ScanCount)
	}
	if slice.Snapshots[0].ModuleCount != 4 {
		t.Fatalf("unexpected snapshot payload: %+v", slice.Snapshots[0])
	}
}

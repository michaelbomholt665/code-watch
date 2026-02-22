package graph

import (
	"circular/internal/engine/parser"
	"path/filepath"
	"testing"
)

func TestSQLiteSymbolStore_SyncLookupAndPrune(t *testing.T) {
	store, err := OpenSQLiteSymbolStore(filepath.Join(t.TempDir(), "symbols.db"), "proj-a")
	if err != nil {
		t.Fatalf("open sqlite symbol store: %v", err)
	}
	defer store.Close()

	g := NewGraph()
	g.AddFile(&parser.File{
		Path:     "svc.py",
		Language: "python",
		Module:   "svc",
		Definitions: []parser.Definition{{
			Name:       "GreeterServicer",
			FullName:   "svc.GreeterServicer",
			Kind:       parser.KindClass,
			Exported:   true,
			Visibility: "public",
			Signature:  "class GreeterServicer",
			TypeHint:   "class",
			Decorators: []string{"grpc.service"},
		}},
	})

	if err := store.SyncFromGraph(g); err != nil {
		t.Fatalf("sync initial graph: %v", err)
	}

	records := store.Lookup("GreeterServicer")
	if len(records) != 1 {
		t.Fatalf("expected one direct lookup match, got %d", len(records))
	}
	if !records[0].IsService {
		t.Fatal("expected service classification")
	}

	serviceMatches := store.LookupService("GreeterClient")
	if len(serviceMatches) == 0 {
		t.Fatal("expected service-key lookup match")
	}

	g.RemoveFile("svc.py")
	if err := store.SyncFromGraph(g); err != nil {
		t.Fatalf("sync pruned graph: %v", err)
	}
	if got := store.Lookup("GreeterServicer"); len(got) != 0 {
		t.Fatalf("expected lookup to be pruned after delete, got %d", len(got))
	}
}

func TestSQLiteSymbolStore_ProjectIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "symbols.db")
	storeA, err := OpenSQLiteSymbolStore(path, "proj-a")
	if err != nil {
		t.Fatalf("open store A: %v", err)
	}
	defer storeA.Close()

	storeB, err := OpenSQLiteSymbolStore(path, "proj-b")
	if err != nil {
		t.Fatalf("open store B: %v", err)
	}
	defer storeB.Close()

	g := NewGraph()
	g.AddFile(&parser.File{
		Path:     "a.go",
		Language: "go",
		Module:   "modA",
		Definitions: []parser.Definition{{
			Name:     "Alpha",
			Kind:     parser.KindFunction,
			Exported: true,
		}},
	})
	if err := storeA.SyncFromGraph(g); err != nil {
		t.Fatalf("sync store A: %v", err)
	}

	if got := storeB.Lookup("Alpha"); len(got) != 0 {
		t.Fatalf("expected project-isolated lookup to be empty for store B, got %d", len(got))
	}
}

func TestSQLiteSymbolStore_UpsertDeleteAndPrune(t *testing.T) {
	store, err := OpenSQLiteSymbolStore(filepath.Join(t.TempDir(), "symbols.db"), "proj-a")
	if err != nil {
		t.Fatalf("open sqlite symbol store: %v", err)
	}
	defer store.Close()

	if err := store.UpsertFile(&parser.File{
		Path:     "one.go",
		Language: "go",
		Module:   "mod/one",
		Definitions: []parser.Definition{
			{Name: "One", FullName: "mod/one.One", Kind: parser.KindFunction, Exported: true},
		},
	}); err != nil {
		t.Fatalf("upsert one.go: %v", err)
	}
	if err := store.UpsertFile(&parser.File{
		Path:     "two.go",
		Language: "go",
		Module:   "mod/two",
		Definitions: []parser.Definition{
			{Name: "Two", FullName: "mod/two.Two", Kind: parser.KindFunction, Exported: true},
		},
	}); err != nil {
		t.Fatalf("upsert two.go: %v", err)
	}

	if got := store.Lookup("One"); len(got) != 1 {
		t.Fatalf("expected one lookup row, got %d", len(got))
	} else if got[0].Branches != 0 || got[0].Parameters != 0 || got[0].Nesting != 0 || got[0].LOC != 0 {
		t.Fatalf("expected default complexity metrics to be zero, got %+v", got[0])
	}
	if got := store.Lookup("Two"); len(got) != 1 {
		t.Fatalf("expected two lookup row, got %d", len(got))
	}

	if err := store.DeleteFile("one.go"); err != nil {
		t.Fatalf("delete one.go: %v", err)
	}
	if got := store.Lookup("One"); len(got) != 0 {
		t.Fatalf("expected deleted one.go symbol rows, got %d", len(got))
	}
	if got := store.Lookup("Two"); len(got) != 1 {
		t.Fatalf("expected two.go rows preserved, got %d", len(got))
	}

	if err := store.PruneToPaths([]string{"one.go"}); err != nil {
		t.Fatalf("prune to one.go: %v", err)
	}
	if got := store.Lookup("Two"); len(got) != 0 {
		t.Fatalf("expected two.go rows pruned, got %d", len(got))
	}
}

func TestSQLiteSymbolStore_LookupPersistsComplexityMetrics(t *testing.T) {
	store, err := OpenSQLiteSymbolStore(filepath.Join(t.TempDir(), "symbols.db"), "proj-a")
	if err != nil {
		t.Fatalf("open sqlite symbol store: %v", err)
	}
	defer store.Close()

	if err := store.UpsertFile(&parser.File{
		Path:     "hot.go",
		Language: "go",
		Module:   "mod/hot",
		Definitions: []parser.Definition{
			{
				Name:           "Hot",
				FullName:       "mod/hot.Hot",
				Kind:           parser.KindFunction,
				Exported:       true,
				BranchCount:    4,
				ParameterCount: 3,
				NestingDepth:   2,
				LOC:            55,
				Location:       parser.Location{File: "hot.go", Line: 12, Column: 1},
			},
		},
	}); err != nil {
		t.Fatalf("upsert hot.go: %v", err)
	}

	got := store.Lookup("Hot")
	if len(got) != 1 {
		t.Fatalf("expected one lookup row, got %d", len(got))
	}
	if got[0].Branches != 4 || got[0].Parameters != 3 || got[0].Nesting != 2 || got[0].LOC != 55 {
		t.Fatalf("expected persisted complexity metrics, got %+v", got[0])
	}
}

func TestSQLiteSymbolStore_PruneToPaths_RemovesFileBlobs(t *testing.T) {
	store, err := OpenSQLiteSymbolStore(filepath.Join(t.TempDir(), "symbols.db"), "proj-a")
	if err != nil {
		t.Fatalf("open sqlite symbol store: %v", err)
	}
	defer store.Close()

	if err := store.UpsertFile(&parser.File{
		Path:     "one.go",
		Language: "go",
		Module:   "mod/one",
		Definitions: []parser.Definition{
			{Name: "One", FullName: "mod/one.One", Kind: parser.KindFunction, Exported: true},
		},
	}); err != nil {
		t.Fatalf("upsert one.go: %v", err)
	}
	if err := store.UpsertFile(&parser.File{
		Path:     "two.go",
		Language: "go",
		Module:   "mod/two",
		Definitions: []parser.Definition{
			{Name: "Two", FullName: "mod/two.Two", Kind: parser.KindFunction, Exported: true},
		},
	}); err != nil {
		t.Fatalf("upsert two.go: %v", err)
	}

	if err := store.PruneToPaths([]string{"one.go"}); err != nil {
		t.Fatalf("prune to one.go: %v", err)
	}

	var count int
	if err := store.DB().QueryRow(`SELECT count(*) FROM file_blobs WHERE project_key = ?`, "proj-a").Scan(&count); err != nil {
		t.Fatalf("count file blobs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 file blob after prune, got %d", count)
	}
}

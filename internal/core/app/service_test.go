package app

import (
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/data/history"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAnalysisServiceRunScan_WithPaths(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.stub")
	if err := os.WriteFile(filePath, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{WatchPaths: []string{tmpDir}}
	app, err := NewWithDependencies(cfg, Dependencies{
		CodeParser: stubCodeParser{
			parsedFile: &parser.File{Language: "stub", Module: "example.stub"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := app.AnalysisService()
	res, err := svc.RunScan(context.Background(), ports.ScanRequest{Paths: []string{tmpDir}})
	if err != nil {
		t.Fatalf("run scan: %v", err)
	}
	if res.FilesScanned != 1 {
		t.Fatalf("expected files_scanned=1, got %d", res.FilesScanned)
	}
	if res.Modules != 1 {
		t.Fatalf("expected modules=1, got %d", res.Modules)
	}
}

func TestAnalysisServiceQueryService_ListModules(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.stub")
	if err := os.WriteFile(filePath, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{WatchPaths: []string{tmpDir}}
	app, err := NewWithDependencies(cfg, Dependencies{
		CodeParser: stubCodeParser{
			parsedFile: &parser.File{Language: "stub", Module: "example.stub"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.ProcessFile(filePath); err != nil {
		t.Fatal(err)
	}

	svc := app.AnalysisService().QueryService(nil, "default")
	rows, err := svc.ListModules(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("list modules: %v", err)
	}
	if len(rows) != 1 || rows[0].Name != "example.stub" {
		t.Fatalf("unexpected modules: %+v", rows)
	}
}

type serviceHistoryStoreStub struct {
	snapshots []history.Snapshot
}

func (h *serviceHistoryStoreStub) SaveSnapshot(projectKey string, snapshot history.Snapshot) error {
	snapshot.ProjectKey = projectKey
	h.snapshots = append(h.snapshots, snapshot)
	return nil
}

func (h *serviceHistoryStoreStub) LoadSnapshots(projectKey string, since time.Time) ([]history.Snapshot, error) {
	out := make([]history.Snapshot, 0, len(h.snapshots))
	for _, snapshot := range h.snapshots {
		if snapshot.ProjectKey != projectKey {
			continue
		}
		if !since.IsZero() && snapshot.Timestamp.Before(since) {
			continue
		}
		out = append(out, snapshot)
	}
	return out, nil
}

func TestAnalysisServiceCaptureHistoryTrend(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.stub")
	if err := os.WriteFile(filePath, []byte("stub"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{WatchPaths: []string{tmpDir}}
	app, err := NewWithDependencies(cfg, Dependencies{
		CodeParser: stubCodeParser{
			parsedFile: &parser.File{Language: "stub", Module: "example.stub"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.ProcessFile(filePath); err != nil {
		t.Fatal(err)
	}

	store := &serviceHistoryStoreStub{}
	result, err := app.AnalysisService().CaptureHistoryTrend(context.Background(), store, ports.HistoryTrendRequest{
		ProjectKey: "default",
		Since:      time.Time{},
		Window:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("capture history trend: %v", err)
	}
	if !result.SnapshotSaved {
		t.Fatal("expected snapshot to be saved")
	}
	if result.Report == nil {
		t.Fatal("expected non-nil trend report")
	}
	if result.Report.ScanCount != 1 {
		t.Fatalf("expected one snapshot in trend report, got %d", result.Report.ScanCount)
	}
}

func TestAnalysisServiceWatchServiceCurrentUpdate(t *testing.T) {
	app := &App{
		Config: &config.Config{},
		Graph:  graph.NewGraph(),
	}
	app.Graph.AddFile(&parser.File{Path: "main.stub", Module: "example.stub"})

	update, err := app.AnalysisService().WatchService().CurrentUpdate(context.Background())
	if err != nil {
		t.Fatalf("current update: %v", err)
	}
	if update.ModuleCount != 1 {
		t.Fatalf("expected module_count=1, got %d", update.ModuleCount)
	}
	if update.FileCount != 1 {
		t.Fatalf("expected file_count=1, got %d", update.FileCount)
	}
}

func TestAnalysisServiceWatchServiceSubscribe(t *testing.T) {
	app := &App{
		Config: &config.Config{},
		Graph:  graph.NewGraph(),
	}
	svc := app.AnalysisService().WatchService()

	got := make(chan ports.WatchUpdate, 1)
	if err := svc.Subscribe(context.Background(), func(update ports.WatchUpdate) {
		got <- update
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	expected := Update{
		Cycles:         [][]string{{"a", "b"}},
		Hallucinations: []resolver.UnresolvedReference{{File: "main.go"}},
		ModuleCount:    2,
		FileCount:      3,
		SecretCount:    1,
	}
	app.emitUpdate(expected)

	select {
	case update := <-got:
		if update.ModuleCount != expected.ModuleCount {
			t.Fatalf("expected module_count=%d, got %d", expected.ModuleCount, update.ModuleCount)
		}
		if len(update.Cycles) != 1 {
			t.Fatalf("expected 1 cycle, got %d", len(update.Cycles))
		}
		if len(update.Hallucinations) != 1 {
			t.Fatalf("expected 1 hallucination, got %d", len(update.Hallucinations))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch update")
	}
}

func TestAnalysisServiceDetectCycles_WithLimit(t *testing.T) {
	app := &App{
		Config: &config.Config{},
		Graph:  graph.NewGraph(),
	}
	app.Graph.AddFile(&parser.File{
		Path:   "a.go",
		Module: "module/a",
		Imports: []parser.Import{
			{Module: "module/b"},
		},
	})
	app.Graph.AddFile(&parser.File{
		Path:   "b.go",
		Module: "module/b",
		Imports: []parser.Import{
			{Module: "module/a"},
		},
	})

	cycles, count, err := app.AnalysisService().DetectCycles(context.Background(), 1)
	if err != nil {
		t.Fatalf("detect cycles: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected cycle count=1, got %d", count)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected limited cycles length=1, got %d", len(cycles))
	}
	if len(cycles[0]) == 0 {
		t.Fatal("expected non-empty cycle")
	}
}

func TestAnalysisServiceListFiles(t *testing.T) {
	app := &App{
		Config: &config.Config{},
		Graph:  graph.NewGraph(),
	}
	app.Graph.AddFile(&parser.File{Path: "main.stub", Module: "example.stub"})

	files, err := app.AnalysisService().ListFiles(context.Background())
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Module != "example.stub" {
		t.Fatalf("expected module example.stub, got %q", files[0].Module)
	}
}

func TestAnalysisServiceSummarySnapshot(t *testing.T) {
	app := &App{
		Config: &config.Config{},
		Graph:  graph.NewGraph(),
	}
	app.Graph.AddFile(&parser.File{
		Path:   "a.go",
		Module: "module/a",
		Imports: []parser.Import{
			{Module: "module/b"},
		},
		Secrets: []parser.Secret{
			{
				Kind:     "api_key",
				Severity: "high",
				Location: parser.Location{File: "a.go", Line: 3, Column: 1},
			},
		},
	})
	app.Graph.AddFile(&parser.File{
		Path:   "b.go",
		Module: "module/b",
		Imports: []parser.Import{
			{Module: "module/a"},
		},
	})

	snapshot, err := app.AnalysisService().SummarySnapshot(context.Background())
	if err != nil {
		t.Fatalf("summary snapshot: %v", err)
	}
	if snapshot.FileCount != 2 {
		t.Fatalf("expected file_count=2, got %d", snapshot.FileCount)
	}
	if snapshot.ModuleCount != 2 {
		t.Fatalf("expected module_count=2, got %d", snapshot.ModuleCount)
	}
	if snapshot.SecretCount != 1 {
		t.Fatalf("expected secret_count=1, got %d", snapshot.SecretCount)
	}
	if len(snapshot.Cycles) != 1 {
		t.Fatalf("expected cycle count=1, got %d", len(snapshot.Cycles))
	}
	if len(snapshot.Metrics) != 2 {
		t.Fatalf("expected metrics for 2 modules, got %d", len(snapshot.Metrics))
	}
}

func TestAnalysisServicePrintSummary(t *testing.T) {
	app := &App{
		Config: &config.Config{},
		Graph:  graph.NewGraph(),
	}
	err := app.AnalysisService().PrintSummary(context.Background(), ports.SummaryPrintRequest{
		Duration: time.Second,
		Snapshot: ports.SummarySnapshot{},
	})
	if err != nil {
		t.Fatalf("print summary: %v", err)
	}
}

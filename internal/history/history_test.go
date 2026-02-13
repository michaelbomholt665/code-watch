package history

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_OpenInitializesSchemaAndSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	base := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	first := Snapshot{
		Timestamp:       base,
		ModuleCount:     5,
		FileCount:       8,
		CycleCount:      1,
		UnresolvedCount: 3,
	}
	dup := Snapshot{
		Timestamp:       base,
		ModuleCount:     8,
		FileCount:       11,
		CycleCount:      2,
		UnresolvedCount: 5,
	}
	second := Snapshot{
		Timestamp:       base.Add(2 * time.Hour),
		ModuleCount:     6,
		FileCount:       9,
		CycleCount:      0,
		UnresolvedCount: 1,
		AvgFanIn:        1.5,
		AvgFanOut:       2.0,
		MaxFanIn:        4,
		MaxFanOut:       5,
	}

	if err := store.SaveSnapshot("project-a", first); err != nil {
		t.Fatalf("save first snapshot: %v", err)
	}
	if err := store.SaveSnapshot("project-a", dup); err != nil {
		t.Fatalf("save duplicate snapshot: %v", err)
	}
	if err := store.SaveSnapshot("project-a", second); err != nil {
		t.Fatalf("save second snapshot: %v", err)
	}

	got, err := store.LoadSnapshots("project-a", base.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("load snapshots: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 snapshot after since filter, got %d", len(got))
	}
	if got[0].ModuleCount != 6 {
		t.Fatalf("expected module_count=6, got %d", got[0].ModuleCount)
	}
	if got[0].AvgFanIn != 1.5 || got[0].MaxFanOut != 5 {
		t.Fatalf("expected fan metrics to roundtrip, got %+v", got[0])
	}

	// Duplicate key should have upserted the first timestamp.
	all, err := store.LoadSnapshots("project-a", time.Time{})
	if err != nil {
		t.Fatalf("load all snapshots: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected deduplicated 2 snapshots, got %d", len(all))
	}
	if all[0].ModuleCount != 8 {
		t.Fatalf("expected upserted module_count=8, got %d", all[0].ModuleCount)
	}
}

func TestStore_OpenRejectsDirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := Open(tmpDir)
	if err == nil {
		t.Fatal("expected open error for directory path")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStore_OpenCorruptDBPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.db")
	if err := os.WriteFile(path, []byte("this is not sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Open(path)
	if err == nil {
		t.Fatal("expected sqlite open error")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "not a database") && !strings.Contains(lower, "schema") {
		t.Fatalf("expected schema/open error, got: %v", err)
	}
}

func TestEnsureSchema_DetectsNewerVersionDrift(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.db.Exec(`INSERT OR REPLACE INTO schema_migrations(version) VALUES (?)`, SchemaVersion+1)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open(driverName, "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = EnsureSchema(db)
	if err == nil {
		t.Fatal("expected drift error")
	}
	if !strings.Contains(err.Error(), "newer than supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildTrendReport(t *testing.T) {
	base := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	snapshots := []Snapshot{
		{Timestamp: base, ModuleCount: 4, FileCount: 5, CycleCount: 2, UnresolvedCount: 4, AvgFanIn: 1, AvgFanOut: 1.2},
		{Timestamp: base.Add(2 * time.Hour), ModuleCount: 6, FileCount: 8, CycleCount: 1, UnresolvedCount: 2, AvgFanIn: 2, AvgFanOut: 2.4},
		{Timestamp: base.Add(25 * time.Hour), ModuleCount: 7, FileCount: 9, CycleCount: 3, UnresolvedCount: 1, AvgFanIn: 2.5, AvgFanOut: 2.1},
	}

	report, err := BuildTrendReport("project-a", snapshots, 24*time.Hour)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.ScanCount != 3 {
		t.Fatalf("expected scan_count=3, got %d", report.ScanCount)
	}
	if report.Points[1].DeltaModules != 2 {
		t.Fatalf("expected delta_modules=2, got %d", report.Points[1].DeltaModules)
	}
	if report.Points[2].DeltaCycles != 2 {
		t.Fatalf("expected delta_cycles=2, got %d", report.Points[2].DeltaCycles)
	}
	if report.Points[1].DeltaAvgFanIn != 1 {
		t.Fatalf("expected delta_avg_fan_in=1, got %v", report.Points[1].DeltaAvgFanIn)
	}
	if report.Points[1].ModuleGrowthPct != 50 {
		t.Fatalf("expected module growth pct=50, got %v", report.Points[1].ModuleGrowthPct)
	}
}

func TestIsCorruptError(t *testing.T) {
	if !IsCorruptError(errors.New("database disk image is malformed")) {
		t.Fatal("expected malformed sqlite message to be treated as corrupt")
	}
}

func TestStore_SaveLoadSnapshots_ProjectIsolation(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	base := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	if err := store.SaveSnapshot("project-a", Snapshot{Timestamp: base, ModuleCount: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSnapshot("project-b", Snapshot{Timestamp: base, ModuleCount: 2}); err != nil {
		t.Fatal(err)
	}

	aRows, err := store.LoadSnapshots("project-a", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(aRows) != 1 || aRows[0].ModuleCount != 1 {
		t.Fatalf("unexpected project-a rows: %+v", aRows)
	}

	bRows, err := store.LoadSnapshots("project-b", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bRows) != 1 || bRows[0].ModuleCount != 2 {
		t.Fatalf("unexpected project-b rows: %+v", bRows)
	}
}

package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAdapter_SaveAndLoadSnapshots(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := NewAdapter(store)
	now := time.Now().UTC().Truncate(time.Second)
	snapshot := Snapshot{
		Timestamp:   now,
		ModuleCount: 3,
		FileCount:   5,
	}
	if err := adapter.SaveSnapshot("project-a", snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	rows, err := adapter.LoadSnapshots("project-a", now.Add(-time.Second))
	if err != nil {
		t.Fatalf("load snapshots: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(rows))
	}
	if rows[0].ModuleCount != 3 || rows[0].FileCount != 5 {
		t.Fatalf("unexpected snapshot: %+v", rows[0])
	}
}

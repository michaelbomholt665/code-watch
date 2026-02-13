package history

import (
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkStore_SaveSnapshot(b *testing.B) {
	store, err := Open(filepath.Join(b.TempDir(), "history.db"))
	if err != nil {
		b.Fatalf("open store: %v", err)
	}
	defer store.Close()

	base := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := Snapshot{
			Timestamp:       base.Add(time.Duration(i) * time.Second),
			ModuleCount:     100 + (i % 7),
			FileCount:       250 + (i % 11),
			CycleCount:      i % 3,
			UnresolvedCount: i % 5,
			AvgFanIn:        1.2,
			AvgFanOut:       1.6,
			MaxFanIn:        9,
			MaxFanOut:       12,
		}
		if err := store.SaveSnapshot(s); err != nil {
			b.Fatalf("save snapshot: %v", err)
		}
	}
}

func BenchmarkStore_LoadSnapshots(b *testing.B) {
	store, err := Open(filepath.Join(b.TempDir(), "history.db"))
	if err != nil {
		b.Fatalf("open store: %v", err)
	}
	defer store.Close()

	base := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 2500; i++ {
		if err := store.SaveSnapshot(Snapshot{
			Timestamp:       base.Add(time.Duration(i) * time.Minute),
			ModuleCount:     30 + i%17,
			FileCount:       90 + i%19,
			CycleCount:      i % 4,
			UnresolvedCount: i % 9,
		}); err != nil {
			b.Fatalf("seed snapshot %d: %v", i, err)
		}
	}

	since := base.Add(24 * time.Hour)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshots, err := store.LoadSnapshots(since)
		if err != nil {
			b.Fatalf("load snapshots: %v", err)
		}
		if len(snapshots) == 0 {
			b.Fatal("expected snapshots")
		}
	}
}

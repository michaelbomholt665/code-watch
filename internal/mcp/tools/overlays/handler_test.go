// # internal/mcp/tools/overlays/handler_test.go
package overlays

import (
	"circular/internal/engine/graph"
	"context"
	"path/filepath"
	"testing"
)

func openTestOverlayStore(t *testing.T) *OverlayStore {
	t.Helper()
	store, err := graph.OpenSQLiteSymbolStore(filepath.Join(t.TempDir(), "symbols.db"), "proj-overlay")
	if err != nil {
		t.Fatalf("open sqlite symbol store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return NewOverlayStore(store.DB(), "proj-overlay")
}

func TestOverlayStore_AddAndCheck(t *testing.T) {
	s := openTestOverlayStore(t)
	ctx := context.Background()

	out, err := s.AddOverlay(ctx, AddOverlayInput{
		Symbol: "os",
		File:   "cli.go",
		Type:   OverlayVetted,
		Reason: "os.Stdout used at line 82",
	})
	if err != nil {
		t.Fatalf("AddOverlay: %v", err)
	}
	if out.ID == 0 {
		t.Fatal("expected non-zero overlay ID")
	}

	overlay, err := s.CheckOverlay(ctx, "os", "cli.go")
	if err != nil {
		t.Fatalf("CheckOverlay: %v", err)
	}
	if overlay == nil {
		t.Fatal("expected overlay to be found, got nil")
	}
	if overlay.Reason != "os.Stdout used at line 82" {
		t.Errorf("reason mismatch: got %q", overlay.Reason)
	}
}

func TestOverlayStore_CheckOverlay_NoMatch(t *testing.T) {
	s := openTestOverlayStore(t)
	ctx := context.Background()

	overlay, err := s.CheckOverlay(ctx, "nonexistent", "any.go")
	if err != nil {
		t.Fatalf("CheckOverlay: %v", err)
	}
	if overlay != nil {
		t.Fatalf("expected nil overlay for unknown symbol, got %+v", overlay)
	}
}

func TestOverlayStore_List(t *testing.T) {
	s := openTestOverlayStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := s.AddOverlay(ctx, AddOverlayInput{
			Symbol: "fmt",
			File:   "main.go",
			Type:   OverlayExclusion,
			Reason: "test",
		}); err != nil {
			t.Fatalf("AddOverlay %d: %v", i, err)
		}
	}

	list, err := s.ListOverlays(ctx, ListOverlaysInput{Symbol: "fmt"})
	if err != nil {
		t.Fatalf("ListOverlays: %v", err)
	}
	if list.Total != 3 {
		t.Errorf("expected 3 overlays, got %d", list.Total)
	}
}

func TestOverlayStore_MarkStale(t *testing.T) {
	s := openTestOverlayStore(t)
	ctx := context.Background()

	if _, err := s.AddOverlay(ctx, AddOverlayInput{
		Symbol:     "log",
		File:       "server.go",
		Type:       OverlayVetted,
		Reason:     "used in init",
		SourceHash: "abc123",
	}); err != nil {
		t.Fatalf("AddOverlay: %v", err)
	}

	// Different hash → overlay should be stale.
	if err := s.MarkStale(ctx, "server.go", "def456"); err != nil {
		t.Fatalf("MarkStale: %v", err)
	}

	// CheckOverlay should return nil because status is RE-VERIFICATION.
	overlay, err := s.CheckOverlay(ctx, "log", "server.go")
	if err != nil {
		t.Fatalf("CheckOverlay after stale: %v", err)
	}
	if overlay != nil {
		t.Fatalf("expected nil after mark-stale, got status=%s", overlay.Status)
	}
}

func TestOverlayStore_EmptySymbol_Error(t *testing.T) {
	s := openTestOverlayStore(t)
	ctx := context.Background()

	_, err := s.AddOverlay(ctx, AddOverlayInput{Type: OverlayVetted})
	if err == nil {
		t.Fatal("expected error for empty symbol, got nil")
	}
}

// # internal/engine/graph/schema.go
package graph

import (
	"database/sql"
	"fmt"
)

// ensureOverlaySchema creates the semantic_overlays table used by Phase IV.
// It is called from migrateSymbolSchema so it is always run when the store opens.
func ensureOverlaySchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS semantic_overlays (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  project_key  TEXT    NOT NULL,
  symbol       TEXT    NOT NULL,
  file_path    TEXT    NOT NULL DEFAULT '',
  overlay_type TEXT    NOT NULL,
  reason       TEXT    NOT NULL DEFAULT '',
  verified_by  TEXT    NOT NULL DEFAULT 'ai',
  source_hash  TEXT    NOT NULL DEFAULT '',
  status       TEXT    NOT NULL DEFAULT 'ACTIVE',
  created_at   INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_overlays_project_symbol
  ON semantic_overlays(project_key, symbol);
CREATE INDEX IF NOT EXISTS idx_overlays_project_file
  ON semantic_overlays(project_key, file_path);
`)
	if err != nil {
		return fmt.Errorf("ensure overlay schema: %w", err)
	}
	return nil
}

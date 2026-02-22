package queue

import (
	"database/sql"
	"fmt"
)

func migrateSpoolSchema(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("spool db is nil")
	}
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS write_spool (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_key TEXT NOT NULL,
  operation TEXT NOT NULL,
  file_path TEXT NOT NULL DEFAULT '',
  payload BLOB NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_write_spool_project_next ON write_spool(project_key, next_attempt_at, id);
`)
	if err != nil {
		return fmt.Errorf("migrate spool schema: %w", err)
	}
	return nil
}

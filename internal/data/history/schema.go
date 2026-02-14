package history

import (
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		sql: `
CREATE TABLE IF NOT EXISTS snapshots (
  project_key TEXT NOT NULL DEFAULT 'default',
  schema_version INTEGER NOT NULL,
  ts_utc TEXT NOT NULL,
  commit_hash TEXT NOT NULL DEFAULT '',
  commit_ts_utc TEXT NOT NULL DEFAULT '',
  module_count INTEGER NOT NULL,
  file_count INTEGER NOT NULL,
  cycle_count INTEGER NOT NULL,
  unresolved_count INTEGER NOT NULL,
  unused_import_count INTEGER NOT NULL,
  violation_count INTEGER NOT NULL,
  hotspot_count INTEGER NOT NULL,
  avg_fan_in REAL NOT NULL DEFAULT 0,
  avg_fan_out REAL NOT NULL DEFAULT 0,
  max_fan_in INTEGER NOT NULL DEFAULT 0,
  max_fan_out INTEGER NOT NULL DEFAULT 0,
  created_at_utc TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (project_key, ts_utc, commit_hash)
);
CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts_utc);
CREATE INDEX IF NOT EXISTS idx_snapshots_commit_hash ON snapshots(commit_hash);
CREATE INDEX IF NOT EXISTS idx_snapshots_project_key ON snapshots(project_key);
`,
	},
	{
		version: 2,
		sql: `
CREATE TABLE IF NOT EXISTS snapshots_v2 (
  project_key TEXT NOT NULL DEFAULT 'default',
  schema_version INTEGER NOT NULL,
  ts_utc TEXT NOT NULL,
  commit_hash TEXT NOT NULL DEFAULT '',
  commit_ts_utc TEXT NOT NULL DEFAULT '',
  module_count INTEGER NOT NULL,
  file_count INTEGER NOT NULL,
  cycle_count INTEGER NOT NULL,
  unresolved_count INTEGER NOT NULL,
  unused_import_count INTEGER NOT NULL,
  violation_count INTEGER NOT NULL,
  hotspot_count INTEGER NOT NULL,
  avg_fan_in REAL NOT NULL DEFAULT 0,
  avg_fan_out REAL NOT NULL DEFAULT 0,
  max_fan_in INTEGER NOT NULL DEFAULT 0,
  max_fan_out INTEGER NOT NULL DEFAULT 0,
  created_at_utc TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (project_key, ts_utc, commit_hash)
);
INSERT OR IGNORE INTO snapshots_v2 (
  project_key, schema_version, ts_utc, commit_hash, commit_ts_utc, module_count, file_count,
  cycle_count, unresolved_count, unused_import_count, violation_count, hotspot_count,
  avg_fan_in, avg_fan_out, max_fan_in, max_fan_out, created_at_utc
)
SELECT
  'default', schema_version, ts_utc, commit_hash, commit_ts_utc, module_count, file_count,
  cycle_count, unresolved_count, unused_import_count, violation_count, hotspot_count,
  avg_fan_in, avg_fan_out, max_fan_in, max_fan_out, created_at_utc
FROM snapshots;
DROP TABLE snapshots;
ALTER TABLE snapshots_v2 RENAME TO snapshots;
CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts_utc);
CREATE INDEX IF NOT EXISTS idx_snapshots_commit_hash ON snapshots(commit_hash);
CREATE INDEX IF NOT EXISTS idx_snapshots_project_key ON snapshots(project_key);
`,
	},
	{
		version: 3,
		sql: `
CREATE TABLE IF NOT EXISTS symbols (
  project_key TEXT NOT NULL,
  symbol_name TEXT NOT NULL,
  canonical_name TEXT NOT NULL,
  full_name TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL,
  language TEXT NOT NULL DEFAULT '',
  file_path TEXT NOT NULL,
  kind INTEGER NOT NULL DEFAULT 0,
  is_exported INTEGER NOT NULL DEFAULT 0,
  visibility TEXT NOT NULL DEFAULT '',
  scope TEXT NOT NULL DEFAULT '',
  signature TEXT NOT NULL DEFAULT '',
  type_hint TEXT NOT NULL DEFAULT '',
  decorators TEXT NOT NULL DEFAULT '[]',
  is_service INTEGER NOT NULL DEFAULT 0,
  service_key TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (project_key, file_path, symbol_name, full_name)
);
CREATE INDEX IF NOT EXISTS idx_symbols_project_canonical ON symbols(project_key, canonical_name);
CREATE INDEX IF NOT EXISTS idx_symbols_project_service_key ON symbols(project_key, service_key);
CREATE INDEX IF NOT EXISTS idx_symbols_project_file ON symbols(project_key, file_path);
`,
	},
}

func EnsureSchema(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at_utc TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);
`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read schema_migrations version: %w", err)
	}
	if current > SchemaVersion {
		return fmt.Errorf("schema version %d is newer than supported version %d", current, SchemaVersion)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.version, err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}

	return nil
}

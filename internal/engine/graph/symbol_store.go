package graph

import (
	"circular/internal/engine/parser"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

type SQLiteSymbolStore struct {
	db         *sql.DB
	projectKey string
}

func OpenSQLiteSymbolStore(path, projectKey string) (*SQLiteSymbolStore, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, fmt.Errorf("symbol store path must not be empty")
	}
	if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
		return nil, fmt.Errorf("symbol store path %q is a directory, expected file", cleanPath)
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create symbol store directory %q: %w", dir, err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(2000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", cleanPath)
	db, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite symbol store %q: %w", cleanPath, err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite symbol store %q: %w", cleanPath, err)
	}

	if err := migrateSymbolSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	key := strings.TrimSpace(projectKey)
	if key == "" {
		key = "default"
	}

	return &SQLiteSymbolStore{db: db, projectKey: key}, nil
}

func (s *SQLiteSymbolStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB returns the underlying *sql.DB so callers (e.g. OverlayStore) can share
// the same connection without opening a second WAL writer.
func (s *SQLiteSymbolStore) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *SQLiteSymbolStore) SyncFromGraph(g *Graph) error {
	if s == nil || s.db == nil || g == nil {
		return nil
	}

	files := g.GetAllFiles()
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin symbol sync tx: %w", err)
	}

	if len(paths) == 0 {
		if _, err := tx.Exec(`DELETE FROM symbols WHERE project_key = ?`, s.projectKey); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("clear symbols for empty graph: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit symbol sync tx: %w", err)
		}
		return nil
	}

	if err := deleteMissingPaths(tx, s.projectKey, paths); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, file := range files {
		if err := upsertFileRows(tx, s.projectKey, file); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit symbol sync tx: %w", err)
	}
	return nil
}

func (s *SQLiteSymbolStore) UpsertFile(file *parser.File) error {
	if s == nil || s.db == nil || file == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin symbol upsert tx: %w", err)
	}
	if err := upsertFileRows(tx, s.projectKey, file); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit symbol upsert tx: %w", err)
	}
	return nil
}

func (s *SQLiteSymbolStore) DeleteFile(path string) error {
	if s == nil || s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin symbol delete tx: %w", err)
	}
	if err := deletePath(tx, s.projectKey, path); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit symbol delete tx: %w", err)
	}
	return nil
}

func (s *SQLiteSymbolStore) PruneToPaths(paths []string) error {
	if s == nil || s.db == nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin symbol prune tx: %w", err)
	}
	if len(paths) == 0 {
		if _, err := tx.Exec(`DELETE FROM symbols WHERE project_key = ?`, s.projectKey); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("clear symbols for empty path set: %w", err)
		}
	} else {
		if err := deleteMissingPaths(tx, s.projectKey, paths); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit symbol prune tx: %w", err)
	}
	return nil
}

func (s *SQLiteSymbolStore) Lookup(symbol string) []SymbolRecord {
	if s == nil || s.db == nil {
		return nil
	}
	key := canonicalSymbol(symbol)
	if key == "" {
		return nil
	}
	return s.lookupRows(`SELECT
  symbol_name,
  full_name,
  module_name,
  language,
  file_path,
  kind,
  is_exported,
  visibility,
  scope,
  signature,
  type_hint,
  decorators,
  is_service,
  COALESCE(usage_tag,''),
  COALESCE(confidence,0.0),
  COALESCE(ancestry,'')
FROM symbols
WHERE project_key = ? AND canonical_name = ?
ORDER BY module_name, file_path, symbol_name`, s.projectKey, key)
}

func (s *SQLiteSymbolStore) LookupService(symbol string) []SymbolRecord {
	if s == nil || s.db == nil {
		return nil
	}
	key := serviceSymbolKey(symbol)
	if key == "" {
		return nil
	}
	return s.lookupRows(`SELECT
  symbol_name,
  full_name,
  module_name,
  language,
  file_path,
  kind,
  is_exported,
  visibility,
  scope,
  signature,
  type_hint,
  decorators,
  is_service,
  COALESCE(usage_tag,''),
  COALESCE(confidence,0.0),
  COALESCE(ancestry,'')
FROM symbols
WHERE project_key = ? AND service_key = ? AND is_service = 1
ORDER BY module_name, file_path, symbol_name`, s.projectKey, key)
}

type symbolRow struct {
	Name          string
	CanonicalName string
	FullName      string
	Module        string
	Language      string
	FilePath      string
	Kind          int
	Exported      bool
	Visibility    string
	Scope         string
	Signature     string
	TypeHint      string
	Decorators    string
	IsService     bool
	ServiceKey    string
	// v4 fields
	UsageTag   string
	Confidence float64
	Ancestry   string
}

func (s *SQLiteSymbolStore) lookupRows(query string, args ...any) []SymbolRecord {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]SymbolRecord, 0)
	for rows.Next() {
		var (
			rec            SymbolRecord
			kind           int
			decoratorsJSON string
		)
		if err := rows.Scan(
			&rec.Name,
			&rec.FullName,
			&rec.Module,
			&rec.Language,
			&rec.File,
			&kind,
			&rec.Exported,
			&rec.Visibility,
			&rec.Scope,
			&rec.Signature,
			&rec.TypeHint,
			&decoratorsJSON,
			&rec.IsService,
			&rec.UsageTag,
			&rec.Confidence,
			&rec.Ancestry,
		); err != nil {
			continue
		}
		rec.Kind = parser.DefinitionKind(kind)
		if decoratorsJSON != "" {
			var decorators []string
			if err := json.Unmarshal([]byte(decoratorsJSON), &decorators); err == nil {
				rec.Decorators = decorators
			}
		}
		out = append(out, rec)
	}
	return out
}

// migrateSymbolSchema creates or migrates the symbols table to schema v4.
// Schema versions:
//
//	v1..v3 = legacy (no confidence/ancestry/usage_tag)
//	v4     = adds usage_tag TEXT, confidence REAL, ancestry TEXT
func migrateSymbolSchema(db *sql.DB) error {
	// Create the base table and indexes if they don't exist.
	_, err := db.Exec(`
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
`)
	if err != nil {
		return fmt.Errorf("ensure symbol base schema: %w", err)
	}

	// Check current schema version.
	var version int
	_ = db.QueryRow(`PRAGMA user_version`).Scan(&version)

	if version < 4 {
		// Add v4 columns — ignore errors for already-existing columns.
		for _, col := range []string{
			`ALTER TABLE symbols ADD COLUMN usage_tag TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE symbols ADD COLUMN confidence REAL NOT NULL DEFAULT 0.0`,
			`ALTER TABLE symbols ADD COLUMN ancestry TEXT NOT NULL DEFAULT ''`,
		} {
			if _, err := db.Exec(col); err != nil {
				// SQLite returns an error if the column already exists; that is safe to ignore.
				if !strings.Contains(err.Error(), "duplicate column") {
					return fmt.Errorf("schema v4 migration (%s): %w", col, err)
				}
			}
		}
		if _, err := db.Exec(`PRAGMA user_version = 4`); err != nil {
			return fmt.Errorf("set schema user_version=4: %w", err)
		}
	}

	// Ensure the semantic_overlays table exists (Phase IV).
	if err := ensureOverlaySchema(db); err != nil {
		return err
	}
	return nil
}

func deleteMissingPaths(tx *sql.Tx, projectKey string, paths []string) error {
	placeholders := make([]string, 0, len(paths))
	args := make([]any, 0, len(paths)+1)
	args = append(args, projectKey)
	for _, p := range paths {
		placeholders = append(placeholders, "?")
		args = append(args, p)
	}
	query := fmt.Sprintf(`DELETE FROM symbols WHERE project_key = ? AND file_path NOT IN (%s)`, strings.Join(placeholders, ","))
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("delete stale symbol rows: %w", err)
	}
	return nil
}

func deleteCurrentPaths(tx *sql.Tx, projectKey string, paths []string) error {
	placeholders := make([]string, 0, len(paths))
	args := make([]any, 0, len(paths)+1)
	args = append(args, projectKey)
	for _, p := range paths {
		placeholders = append(placeholders, "?")
		args = append(args, p)
	}
	query := fmt.Sprintf(`DELETE FROM symbols WHERE project_key = ? AND file_path IN (%s)`, strings.Join(placeholders, ","))
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("delete updated symbol rows: %w", err)
	}
	return nil
}

func deletePath(tx *sql.Tx, projectKey, path string) error {
	if _, err := tx.Exec(`DELETE FROM symbols WHERE project_key = ? AND file_path = ?`, projectKey, path); err != nil {
		return fmt.Errorf("delete symbol rows for path %q: %w", path, err)
	}
	return nil
}

func upsertFileRows(tx *sql.Tx, projectKey string, file *parser.File) error {
	if err := deletePath(tx, projectKey, file.Path); err != nil {
		return err
	}
	rows, err := symbolRowsForFile(file)
	if err != nil {
		return err
	}
	if err := insertRows(tx, projectKey, rows); err != nil {
		return err
	}
	return nil
}

func symbolRowsForFile(file *parser.File) ([]symbolRow, error) {
	rows := make([]symbolRow, 0, len(file.Definitions))
	for i := range file.Definitions {
		def := file.Definitions[i]
		decorators := "[]"
		if len(def.Decorators) > 0 {
			raw, err := json.Marshal(def.Decorators)
			if err != nil {
				return nil, fmt.Errorf("marshal decorators for %q: %w", def.Name, err)
			}
			decorators = string(raw)
		}
		rows = append(rows, symbolRow{
			Name:          def.Name,
			CanonicalName: canonicalSymbol(def.Name),
			FullName:      def.FullName,
			Module:        file.Module,
			Language:      file.Language,
			FilePath:      file.Path,
			Kind:          int(def.Kind),
			Exported:      def.Exported,
			Visibility:    def.Visibility,
			Scope:         def.Scope,
			Signature:     def.Signature,
			TypeHint:      def.TypeHint,
			Decorators:    decorators,
			IsService:     isLikelyServiceDefinition(def),
			ServiceKey:    serviceSymbolKey(def.Name),
		})
	}
	return rows, nil
}

func insertRows(tx *sql.Tx, projectKey string, rows []symbolRow) error {
	stmt, err := tx.Prepare(`
INSERT INTO symbols (
  project_key,
  symbol_name,
  canonical_name,
  full_name,
  module_name,
  language,
  file_path,
  kind,
  is_exported,
  visibility,
  scope,
  signature,
  type_hint,
  decorators,
  is_service,
  service_key,
  usage_tag,
  confidence,
  ancestry
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("prepare symbol insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		if _, err := stmt.Exec(
			projectKey,
			row.Name,
			row.CanonicalName,
			row.FullName,
			row.Module,
			row.Language,
			row.FilePath,
			row.Kind,
			boolToInt(row.Exported),
			row.Visibility,
			row.Scope,
			row.Signature,
			row.TypeHint,
			row.Decorators,
			boolToInt(row.IsService),
			row.ServiceKey,
			row.UsageTag,
			row.Confidence,
			row.Ancestry,
		); err != nil {
			return fmt.Errorf("insert symbol row (%s:%s): %w", row.Module, row.Name, err)
		}
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

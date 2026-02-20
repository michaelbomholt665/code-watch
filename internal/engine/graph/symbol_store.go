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
  COALESCE(ancestry,''),
  COALESCE(line_number,0)
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
  COALESCE(ancestry,''),
  COALESCE(line_number,0)
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
	// v5 fields
	Line int
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
			line           int
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
			&line,
		); err != nil {
			continue
		}
		rec.Kind = parser.DefinitionKind(kind)
		// We don't currently expose Line in SymbolRecord (it's in Location),
		// but we could if needed. For now just scanning it to consume the column.
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

// migrateSymbolSchema creates or migrates the symbols table to schema v5.
// Schema versions:
//
//	v1..v3 = legacy (no confidence/ancestry/usage_tag)
//	v4     = adds usage_tag, confidence, ancestry
//	v5     = adds line_number to PK to support multiple symbols per file
func migrateSymbolSchema(db *sql.DB) error {
	// 1. Check current version
	var version int
	_ = db.QueryRow(`PRAGMA user_version`).Scan(&version)

	// 2. Initial creation (if version 0)
	if version == 0 {
		_, err := db.Exec(`
CREATE TABLE symbols (
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
  usage_tag TEXT NOT NULL DEFAULT '',
  confidence REAL NOT NULL DEFAULT 0.0,
  ancestry TEXT NOT NULL DEFAULT '',
  line_number INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (project_key, file_path, symbol_name, full_name, line_number)
);
CREATE INDEX idx_symbols_project_canonical ON symbols(project_key, canonical_name);
CREATE INDEX idx_symbols_project_service_key ON symbols(project_key, service_key);
CREATE INDEX IF NOT EXISTS idx_symbols_project_file ON symbols(project_key, file_path);
PRAGMA user_version = 5;
`)
		if err != nil {
			return fmt.Errorf("create v5 schema: %w", err)
		}
		return ensureOverlaySchema(db)
	}

	// 3. Migration v0..v3 -> v4
	if version < 4 {
		// (Same as before, add columns if missing)
		for _, col := range []string{
			`ALTER TABLE symbols ADD COLUMN usage_tag TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE symbols ADD COLUMN confidence REAL NOT NULL DEFAULT 0.0`,
			`ALTER TABLE symbols ADD COLUMN ancestry TEXT NOT NULL DEFAULT ''`,
		} {
			if _, err := db.Exec(col); err != nil {
				if !strings.Contains(err.Error(), "duplicate column") {
					return fmt.Errorf("schema v4 migration (%s): %w", col, err)
				}
			}
		}
		if _, err := db.Exec(`PRAGMA user_version = 4`); err != nil {
			return fmt.Errorf("set version=4: %w", err)
		}
		version = 4
	}

	// 4. Migration v4 -> v5 (Recreate table to change PK)
	if version < 5 {
		// Transaction for safety
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Rename old table
		if _, err := tx.Exec(`ALTER TABLE symbols RENAME TO symbols_old`); err != nil {
			return fmt.Errorf("rename old symbols: %w", err)
		}

		// Drop old indexes to free up names
		for _, idx := range []string{"idx_symbols_project_canonical", "idx_symbols_project_service_key", "idx_symbols_project_file"} {
			if _, err := tx.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %s`, idx)); err != nil {
				return fmt.Errorf("drop old index %s: %w", idx, err)
			}
		}

		// Create new table
		_, err = tx.Exec(`
CREATE TABLE symbols (
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
  usage_tag TEXT NOT NULL DEFAULT '',
  confidence REAL NOT NULL DEFAULT 0.0,
  ancestry TEXT NOT NULL DEFAULT '',
  line_number INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (project_key, file_path, symbol_name, full_name, line_number)
);
CREATE INDEX idx_symbols_project_canonical ON symbols(project_key, canonical_name);
CREATE INDEX idx_symbols_project_service_key ON symbols(project_key, service_key);
CREATE INDEX idx_symbols_project_file ON symbols(project_key, file_path);
`)
		if err != nil {
			return fmt.Errorf("create v5 symbols: %w", err)
		}

		// Copy data (defaulting line_number to 0)
		if _, err := tx.Exec(`
INSERT INTO symbols (
  project_key, symbol_name, canonical_name, full_name, module_name, language, file_path,
  kind, is_exported, visibility, scope, signature, type_hint, decorators, is_service, service_key,
  usage_tag, confidence, ancestry, line_number
)
SELECT 
  project_key, symbol_name, canonical_name, full_name, module_name, language, file_path,
  kind, is_exported, visibility, scope, signature, type_hint, decorators, is_service, service_key,
  usage_tag, confidence, ancestry, 0
FROM symbols_old
`); err != nil {
			return fmt.Errorf("copy data to v5: %w", err)
		}

		// Drop old table
		if _, err := tx.Exec(`DROP TABLE symbols_old`); err != nil {
			return fmt.Errorf("drop symbols_old: %w", err)
		}

		// Update version
		if _, err := tx.Exec(`PRAGMA user_version = 5`); err != nil {
			return fmt.Errorf("set version=5: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit v5 migration: %w", err)
		}
	}

	return ensureOverlaySchema(db)
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
	// Use a map to dedup by (Name, Line) to prevent UNIQUE matches on nested AST nodes.
	// Key = "Name:Line". Value = symbolRow.
	// Strategy: Keep Highest Confidence.
	dedup := make(map[string]symbolRow)

	// 1. Process Definitions
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

		// Universal Extractor stores Ancestry in Scope
		usageTag := "SYM_DEF"
		ancestry := ""
		confidence := 1.0
		scope := def.Scope

		if strings.Contains(scope, "->") {
			ancestry = scope
			scope = "global" // Reset scope to default
		}

		key := fmt.Sprintf("%s:%d", def.Name, def.Location.Line)
		row := symbolRow{
			Name:          def.Name,
			CanonicalName: canonicalSymbol(def.Name),
			FullName:      def.FullName,
			Module:        file.Module,
			Language:      file.Language,
			FilePath:      file.Path,
			Kind:          int(def.Kind),
			Exported:      def.Exported,
			Visibility:    def.Visibility,
			Scope:         scope,
			Signature:     def.Signature,
			TypeHint:      def.TypeHint,
			Decorators:    decorators,
			IsService:     isLikelyServiceDefinition(def),
			ServiceKey:    serviceSymbolKey(def.Name),
			// V5 fields
			UsageTag:   usageTag,
			Confidence: confidence,
			Ancestry:   ancestry,
			Line:       def.Location.Line,
		}

		// Definitions always win or merge?
		// Usually definition confidence is 1.0, so it will overwrite refs.
		if existing, ok := dedup[key]; ok {
			if row.Confidence > existing.Confidence {
				dedup[key] = row
			}
		} else {
			dedup[key] = row
		}
	}

	// 2. Process References (mixed into symbols table for Surgical API)
	for i := range file.References {
		ref := file.References[i]

		// Universal Extractor stores "TAG|ANCESTRY" in Context
		usageTag := ""
		ancestry := ""
		confidence := 0.0

		if strings.Contains(ref.Context, "|") {
			parts := strings.SplitN(ref.Context, "|", 2)
			usageTag = parts[0]
			ancestry = parts[1]
			// Restore confidence defaults
			switch usageTag {
			case "REF_CALL":
				confidence = 0.9
			case "REF_TYPE":
				confidence = 0.8
			case "REF_SIDE":
				confidence = 0.7
			case "REF_DYN":
				confidence = 0.4
			default:
				confidence = 0.5
			}
		} else {
			// Legacy reference context (e.g. "service_bridge")
			// We can map this to a tag if we want, or leave empty
			if ref.Context != "" {
				usageTag = "REF_" + strings.ToUpper(ref.Context)
				confidence = 0.6
			}
		}

		// Only insert if we have a tag (Surgical API focus) or if we decide to store all refs?
		// Storing all refs might bloom the table size.
		// The Surgical API needs context for *usages*.
		// Universal Extractor emits everything.
		// Let's store them.

		key := fmt.Sprintf("%s:%d", ref.Name, ref.Location.Line)
		row := symbolRow{
			Name:          ref.Name,
			CanonicalName: canonicalSymbol(ref.Name),
			FullName:      ref.Name, // Refs usually don't have FullName resolved yet
			Module:        file.Module,
			Language:      file.Language,
			FilePath:      file.Path,
			Kind:          0, // Unknown kind for ref
			Exported:      false,
			Visibility:    "",
			Scope:         "",
			Signature:     "",
			TypeHint:      "",
			Decorators:    "[]",
			IsService:     false,
			ServiceKey:    "",
			// V5 fields
			UsageTag:   usageTag,
			Confidence: confidence,
			Ancestry:   ancestry,
			Line:       ref.Location.Line,
		}

		if existing, ok := dedup[key]; ok {
			if row.Confidence > existing.Confidence {
				dedup[key] = row
			}
		} else {
			dedup[key] = row
		}
	}

	rows := make([]symbolRow, 0, len(dedup))
	for _, row := range dedup {
		rows = append(rows, row)
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
  ancestry,
  line_number
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			row.Line,
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

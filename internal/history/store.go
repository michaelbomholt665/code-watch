package history

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	driverName  = "sqlite"
	maxAttempts = 5
)

type Store struct {
	path string
	db   *sql.DB
	mu   sync.Mutex
}

func Open(path string) (*Store, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, fmt.Errorf("history path must not be empty")
	}
	if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
		return nil, fmt.Errorf("history path %q is a directory, expected file", cleanPath)
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create history directory %q: %w", dir, err)
		}
	}

	// busy_timeout + WAL reduce lock conflicts during watch-mode churn.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(2000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", cleanPath)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite history %q: %w", cleanPath, err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite history %q: %w", cleanPath, err)
	}
	if err := EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize sqlite schema %q: %w", cleanPath, err)
	}

	return &Store{path: cleanPath, db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) SaveSnapshot(projectKey string, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		projectKey = "default"
	}

	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now().UTC()
	}
	if snapshot.SchemaVersion == 0 {
		snapshot.SchemaVersion = SchemaVersion
	}
	if snapshot.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported snapshot schema version %d", snapshot.SchemaVersion)
	}

	commitTS := ""
	if !snapshot.CommitTimestamp.IsZero() {
		commitTS = snapshot.CommitTimestamp.UTC().Format(time.RFC3339Nano)
	}

	query := `
INSERT INTO snapshots (
  project_key, schema_version, ts_utc, commit_hash, commit_ts_utc, module_count, file_count,
  cycle_count, unresolved_count, unused_import_count, violation_count, hotspot_count,
  avg_fan_in, avg_fan_out, max_fan_in, max_fan_out
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(project_key, ts_utc, commit_hash) DO UPDATE SET
  schema_version=excluded.schema_version,
  commit_ts_utc=excluded.commit_ts_utc,
  module_count=excluded.module_count,
  file_count=excluded.file_count,
  cycle_count=excluded.cycle_count,
  unresolved_count=excluded.unresolved_count,
  unused_import_count=excluded.unused_import_count,
  violation_count=excluded.violation_count,
  hotspot_count=excluded.hotspot_count,
  avg_fan_in=excluded.avg_fan_in,
  avg_fan_out=excluded.avg_fan_out,
  max_fan_in=excluded.max_fan_in,
  max_fan_out=excluded.max_fan_out
`
	return s.withRetry("save snapshot", func() error {
		_, err := s.db.Exec(
			query,
			projectKey,
			snapshot.SchemaVersion,
			snapshot.Timestamp.UTC().Format(time.RFC3339Nano),
			snapshot.CommitHash,
			commitTS,
			snapshot.ModuleCount,
			snapshot.FileCount,
			snapshot.CycleCount,
			snapshot.UnresolvedCount,
			snapshot.UnusedImportCount,
			snapshot.ViolationCount,
			snapshot.HotspotCount,
			snapshot.AvgFanIn,
			snapshot.AvgFanOut,
			snapshot.MaxFanIn,
			snapshot.MaxFanOut,
		)
		return err
	})
}

func (s *Store) LoadSnapshots(projectKey string, since time.Time) ([]Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		projectKey = "default"
	}

	base := `
SELECT
  project_key, schema_version, ts_utc, commit_hash, commit_ts_utc, module_count, file_count,
  cycle_count, unresolved_count, unused_import_count, violation_count, hotspot_count,
  avg_fan_in, avg_fan_out, max_fan_in, max_fan_out
FROM snapshots
`
	base += " WHERE project_key = ?"
	args := make([]any, 0, 2)
	args = append(args, projectKey)
	if !since.IsZero() {
		base += " AND ts_utc >= ?"
		args = append(args, since.UTC().Format(time.RFC3339Nano))
	}
	base += " ORDER BY ts_utc ASC, commit_hash ASC"

	var rows *sql.Rows
	err := s.withRetry("load snapshots", func() error {
		var qErr error
		rows, qErr = s.db.Query(base, args...)
		return qErr
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make([]Snapshot, 0)
	for rows.Next() {
		var (
			tsRaw       string
			commitTSRaw string
			snapshot    Snapshot
		)
		if err := rows.Scan(
			&snapshot.ProjectKey,
			&snapshot.SchemaVersion,
			&tsRaw,
			&snapshot.CommitHash,
			&commitTSRaw,
			&snapshot.ModuleCount,
			&snapshot.FileCount,
			&snapshot.CycleCount,
			&snapshot.UnresolvedCount,
			&snapshot.UnusedImportCount,
			&snapshot.ViolationCount,
			&snapshot.HotspotCount,
			&snapshot.AvgFanIn,
			&snapshot.AvgFanOut,
			&snapshot.MaxFanIn,
			&snapshot.MaxFanOut,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot row: %w", err)
		}

		ts, err := time.Parse(time.RFC3339Nano, tsRaw)
		if err != nil {
			return nil, fmt.Errorf("parse snapshot timestamp %q: %w", tsRaw, err)
		}
		snapshot.Timestamp = ts.UTC()

		if commitTSRaw != "" {
			commitTS, err := time.Parse(time.RFC3339Nano, commitTSRaw)
			if err != nil {
				return nil, fmt.Errorf("parse commit timestamp %q: %w", commitTSRaw, err)
			}
			snapshot.CommitTimestamp = commitTS.UTC()
		}

		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot rows: %w", err)
	}

	return snapshots, nil
}

func (s *Store) withRetry(op string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if !isLockError(err) || attempt == maxAttempts {
			break
		}
		time.Sleep(time.Duration(attempt*25) * time.Millisecond)
	}
	return fmt.Errorf("%s: %w", op, lastErr)
}

func isLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "busy")
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func IsCorruptError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "malformed") || strings.Contains(msg, "not a database") || errors.Is(err, os.ErrInvalid)
}

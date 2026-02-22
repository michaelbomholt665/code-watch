package queue

import (
	"circular/internal/core/ports"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const sqliteDriverName = "sqlite"

var _ ports.WriteSpoolPort = (*SQLiteSpool)(nil)

type SQLiteSpool struct {
	db         *sql.DB
	projectKey string
}

type spoolPayload struct {
	Version int                `json:"version"`
	Request ports.WriteRequest `json:"request"`
}

func OpenSQLiteSpool(path string, projectKey string) (*SQLiteSpool, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, fmt.Errorf("spool path must not be empty")
	}
	if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
		return nil, fmt.Errorf("spool path %q is a directory", cleanPath)
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create spool directory %q: %w", dir, err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", cleanPath)
	db, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open spool sqlite %q: %w", cleanPath, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping spool sqlite %q: %w", cleanPath, err)
	}
	if err := migrateSpoolSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	key := strings.TrimSpace(projectKey)
	if key == "" {
		key = "default"
	}
	return &SQLiteSpool{db: db, projectKey: key}, nil
}

func (s *SQLiteSpool) Enqueue(req ports.WriteRequest) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("spool not initialized")
	}
	now := time.Now().UTC().UnixMilli()
	payload := spoolPayload{Version: 1, Request: req}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal spool payload: %w", err)
	}
	_, err = s.db.Exec(`
INSERT INTO write_spool (project_key, operation, file_path, payload, attempts, next_attempt_at, created_at, last_error)
VALUES (?, ?, ?, ?, 0, ?, ?, '')
`, s.projectKey, string(req.Operation), req.FilePath, raw, now, now)
	if err != nil {
		return fmt.Errorf("enqueue spool write: %w", err)
	}
	return nil
}

func (s *SQLiteSpool) DequeueBatch(ctx context.Context, maxItems int) ([]ports.SpoolRow, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("spool not initialized")
	}
	if maxItems <= 0 {
		maxItems = 1
	}
	now := time.Now().UTC().UnixMilli()
	rows, err := s.db.QueryContext(ctx, `
SELECT id, payload, attempts
FROM write_spool
WHERE project_key = ? AND next_attempt_at <= ?
ORDER BY id ASC
LIMIT ?
`, s.projectKey, now, maxItems)
	if err != nil {
		return nil, fmt.Errorf("dequeue spool batch: %w", err)
	}
	defer rows.Close()

	out := make([]ports.SpoolRow, 0, maxItems)
	for rows.Next() {
		var (
			id       int64
			raw      []byte
			attempts int
		)
		if err := rows.Scan(&id, &raw, &attempts); err != nil {
			return nil, fmt.Errorf("scan spool row: %w", err)
		}
		var payload spoolPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode spool payload id=%d: %w", id, err)
		}
		out = append(out, ports.SpoolRow{
			ID:       id,
			Request:  payload.Request,
			Attempts: attempts,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spool rows: %w", err)
	}
	return out, nil
}

func (s *SQLiteSpool) Ack(ids []int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("spool not initialized")
	}
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin spool ack tx: %w", err)
	}
	stmt, err := tx.Prepare(`DELETE FROM write_spool WHERE project_key = ? AND id = ?`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare spool ack: %w", err)
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.Exec(s.projectKey, id); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("ack spool row %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit spool ack tx: %w", err)
	}
	return nil
}

func (s *SQLiteSpool) Nack(rows []ports.SpoolRow, nextAttemptAt time.Time, lastErr string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("spool not initialized")
	}
	if len(rows) == 0 {
		return nil
	}
	nextMS := nextAttemptAt.UTC().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin spool nack tx: %w", err)
	}
	stmt, err := tx.Prepare(`
UPDATE write_spool
SET attempts = ?, next_attempt_at = ?, last_error = ?
WHERE project_key = ? AND id = ?
`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare spool nack: %w", err)
	}
	defer stmt.Close()
	for _, row := range rows {
		if _, err := stmt.Exec(row.Attempts+1, nextMS, lastErr, s.projectKey, row.ID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("nack spool row %d: %w", row.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit spool nack tx: %w", err)
	}
	return nil
}

func (s *SQLiteSpool) PendingCount(ctx context.Context) (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("spool not initialized")
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM write_spool WHERE project_key = ?`, s.projectKey).Scan(&count); err != nil {
		return 0, fmt.Errorf("count spool rows: %w", err)
	}
	return count, nil
}

func (s *SQLiteSpool) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

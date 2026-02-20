// # internal/mcp/tools/overlays/handler.go
package overlays

import (
	"circular/internal/engine/graph"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// OverlayType classifies the intent of an AI-created semantic overlay.
type OverlayType string

const (
	// OverlayExclusion suppresses a false-positive warning for a symbol.
	OverlayExclusion OverlayType = "EXCLUSION"
	// OverlayVetted marks a symbol as confirmed-in-use by an AI agent.
	OverlayVetted OverlayType = "VETTED_USAGE"
	// OverlayReAlias records a known alias relationship between symbols.
	OverlayReAlias OverlayType = "RE-ALIAS"
)

// Overlay is a persisted AI-verified annotation for a symbol.
type Overlay struct {
	ID          int64
	ProjectKey  string
	Symbol      string
	FilePath    string
	OverlayType OverlayType
	Reason      string
	VerifiedBy  string
	SourceHash  string
	Status      string
	CreatedAt   time.Time
}

// AddOverlayInput describes the data needed to persist a new overlay.
type AddOverlayInput struct {
	// Symbol is the name of the symbol being annotated.
	Symbol string `json:"symbol"`
	// File is the file that contains the symbol (optional; empty = project-wide).
	File string `json:"file,omitempty"`
	// Type is one of EXCLUSION, VETTED_USAGE, or RE-ALIAS.
	Type OverlayType `json:"type"`
	// Reason is a human-readable explanation written by the AI agent.
	Reason string `json:"reason"`
	// SourceHash is the current SHA-256 of the file (for staleness detection).
	SourceHash string `json:"source_hash,omitempty"`
	// VerifiedBy identifies the agent or user that created this overlay.
	VerifiedBy string `json:"verified_by,omitempty"`
}

// AddOverlayOutput is returned after a successful overlay creation.
type AddOverlayOutput struct {
	ID      int64  `json:"id"`
	Symbol  string `json:"symbol"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ListOverlaysInput filters the overlay query.
type ListOverlaysInput struct {
	// File restricts results to overlays for a specific file path.
	File string `json:"file,omitempty"`
	// Symbol restricts results to overlays for a specific symbol name.
	Symbol string `json:"symbol,omitempty"`
}

// ListOverlaysOutput wraps the overlay result set.
type ListOverlaysOutput struct {
	Overlays []Overlay `json:"overlays"`
	Total    int       `json:"total"`
}

// OverlayStore provides overlay CRUD operations backed by an SQLite database.
type OverlayStore struct {
	db         *sql.DB
	projectKey string
}

// NewOverlayStore wraps an existing SQLite db pointer (shared with the symbol store).
func NewOverlayStore(db *sql.DB, projectKey string) *OverlayStore {
	return &OverlayStore{db: db, projectKey: projectKey}
}

// AddOverlay persists a new overlay entry and returns its ID.
func (s *OverlayStore) AddOverlay(_ context.Context, in AddOverlayInput) (AddOverlayOutput, error) {
	if s.db == nil {
		return AddOverlayOutput{}, fmt.Errorf("overlay store not initialised")
	}
	if in.Symbol == "" {
		return AddOverlayOutput{}, fmt.Errorf("symbol must not be empty")
	}
	if in.VerifiedBy == "" {
		in.VerifiedBy = "ai"
	}
	res, err := s.db.Exec(`
INSERT INTO semantic_overlays
  (project_key, symbol, file_path, overlay_type, reason, verified_by, source_hash, status)
VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')`,
		s.projectKey, in.Symbol, in.File, string(in.Type),
		in.Reason, in.VerifiedBy, in.SourceHash,
	)
	if err != nil {
		return AddOverlayOutput{}, fmt.Errorf("insert overlay: %w", err)
	}
	id, _ := res.LastInsertId()
	return AddOverlayOutput{
		ID:      id,
		Symbol:  in.Symbol,
		Status:  "ACTIVE",
		Message: fmt.Sprintf("overlay %d created: %s %s", id, in.Type, in.Symbol),
	}, nil
}

// ListOverlays returns all active overlays matching the given filter.
func (s *OverlayStore) ListOverlays(_ context.Context, in ListOverlaysInput) (ListOverlaysOutput, error) {
	if s.db == nil {
		return ListOverlaysOutput{}, fmt.Errorf("overlay store not initialised")
	}

	query := `SELECT id, project_key, symbol, file_path, overlay_type, reason,
                     verified_by, source_hash, status, created_at
              FROM semantic_overlays
              WHERE project_key = ? AND status = 'ACTIVE'`
	args := []any{s.projectKey}

	if in.Symbol != "" {
		query += ` AND symbol = ?`
		args = append(args, in.Symbol)
	}
	if in.File != "" {
		query += ` AND file_path = ?`
		args = append(args, in.File)
	}
	query += ` ORDER BY id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return ListOverlaysOutput{}, fmt.Errorf("list overlays: %w", err)
	}
	defer rows.Close()

	overlays := make([]Overlay, 0)
	for rows.Next() {
		var o Overlay
		var ts int64
		var overlayType string
		if err := rows.Scan(&o.ID, &o.ProjectKey, &o.Symbol, &o.FilePath,
			&overlayType, &o.Reason, &o.VerifiedBy, &o.SourceHash, &o.Status, &ts); err != nil {
			return ListOverlaysOutput{}, fmt.Errorf("scan overlay row: %w", err)
		}
		o.OverlayType = OverlayType(overlayType)
		o.CreatedAt = time.Unix(ts, 0).UTC()
		overlays = append(overlays, o)
	}
	return ListOverlaysOutput{Overlays: overlays, Total: len(overlays)}, nil
}

// CheckOverlay returns the first active overlay for symbol+file, or nil.
// It is used by the analysis pipeline to silence false positives before reporting.
func (s *OverlayStore) CheckOverlay(_ context.Context, symbol, filePath string) (*Overlay, error) {
	if s.db == nil {
		return nil, nil
	}
	query := `SELECT id, project_key, symbol, file_path, overlay_type, reason,
                     verified_by, source_hash, status, created_at
              FROM semantic_overlays
              WHERE project_key = ? AND symbol = ? AND status = 'ACTIVE'
                AND (file_path = '' OR file_path = ?)
              ORDER BY id DESC LIMIT 1`
	row := s.db.QueryRow(query, s.projectKey, symbol, filePath)
	var o Overlay
	var ts int64
	var overlayType string
	if err := row.Scan(&o.ID, &o.ProjectKey, &o.Symbol, &o.FilePath,
		&overlayType, &o.Reason, &o.VerifiedBy, &o.SourceHash, &o.Status, &ts); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("check overlay: %w", err)
	}
	o.OverlayType = OverlayType(overlayType)
	o.CreatedAt = time.Unix(ts, 0).UTC()
	return &o, nil
}

// MarkStale updates overlays for a file to RE-VERIFICATION when the source
// hash has changed, indicating the AI-verified state may be outdated.
func (s *OverlayStore) MarkStale(_ context.Context, filePath, newHash string) error {
	if s.db == nil || filePath == "" {
		return nil
	}
	_, err := s.db.Exec(`
UPDATE semantic_overlays
   SET status = 'RE-VERIFICATION'
 WHERE project_key = ? AND file_path = ? AND status = 'ACTIVE'
   AND source_hash != '' AND source_hash != ?`,
		s.projectKey, filePath, newHash,
	)
	if err != nil {
		return fmt.Errorf("mark overlays stale for %q: %w", filePath, err)
	}
	return nil
}

// Ensure OverlayStore can be used via the SymbolLookupTable interface pattern.
var _ graph.SymbolLookupTable = (*graph.UniversalSymbolTable)(nil)

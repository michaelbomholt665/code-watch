package app

import (
	"context"
	"fmt"
	"time"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Components map[string]string `json:"components"`
}

type HealthService struct {
	app *App
}

func NewHealthService(app *App) *HealthService {
	return &HealthService{app: app}
}

func (s *HealthService) Check(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Status:    "up",
		Timestamp: time.Now().UTC(),
		Components: make(map[string]string),
	}

	// Check Graph
	if s.app.Graph == nil {
		status.Status = "degraded"
		status.Components["graph"] = "missing"
	} else {
		status.Components["graph"] = fmt.Sprintf("ok (%d files, %d modules)", s.app.Graph.FileCount(), s.app.Graph.ModuleCount())
	}

	// Check Symbol Store
	if s.app.symbolStore != nil {
		status.Components["symbol_store"] = "ok"
	} else if s.app.Config.DB.Enabled {
		status.Status = "degraded"
		status.Components["symbol_store"] = "missing but enabled in config"
	}

	// Check Parser
	if s.app.codeParser != nil {
		status.Components["parser"] = "ok"
	} else {
		status.Status = "degraded"
		status.Components["parser"] = "missing"
	}

	return status
}

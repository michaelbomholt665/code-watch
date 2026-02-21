package cli

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"circular/internal/core/app"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ObservabilityServer struct {
	addr          string
	healthService *app.HealthService
	server        *http.Server
}

func NewObservabilityServer(addr string, healthService *app.HealthService) *ObservabilityServer {
	return &ObservabilityServer{
		addr:          addr,
		healthService: healthService,
	}
}

func (s *ObservabilityServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status := s.healthService.Check(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if status.Status != "up" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(status)
	})

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	slog.Info("observability server starting", "addr", s.addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("observability server failed", "error", err)
		}
	}()

	return nil
}

func (s *ObservabilityServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

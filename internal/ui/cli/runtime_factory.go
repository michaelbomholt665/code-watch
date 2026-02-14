package cli

import (
	coreapp "circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"fmt"
)

type analysisFactory interface {
	New(cfg *config.Config, includeTests bool) (ports.AnalysisService, error)
}

type coreAnalysisFactory struct{}

func (coreAnalysisFactory) New(cfg *config.Config, includeTests bool) (ports.AnalysisService, error) {
	app, err := coreapp.New(cfg)
	if err != nil {
		return nil, err
	}
	app.IncludeTests = includeTests
	return app.AnalysisService(), nil
}

func initializeAnalysis(cfg *config.Config, includeTests bool, factory analysisFactory) (ports.AnalysisService, error) {
	if factory == nil {
		return nil, fmt.Errorf("analysis factory is required")
	}
	return factory.New(cfg, includeTests)
}

package cli

import (
	"circular/internal/core/ports"
	"circular/internal/data/history"
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func runUI(analysis ports.AnalysisService, historyStore ports.HistoryStore, projectKey string, report *history.TrendReport) error {
	if analysis == nil {
		return fmt.Errorf("analysis service unavailable")
	}
	service := analysis.QueryService(historyStore, projectKey)
	watch := analysis.WatchService()
	if service == nil || watch == nil {
		return fmt.Errorf("ui services unavailable")
	}
	m := initialModel(service, report)
	p := tea.NewProgram(m, tea.WithAltScreen())

	sendUpdate := func(update ports.WatchUpdate) {
		modules, err := service.ListModules(context.Background(), "", 0)
		if err != nil {
			modules = nil
		}
		p.Send(updateMsg{
			cycles:         update.Cycles,
			hallucinations: update.Hallucinations,
			modules:        modules,
			moduleCount:    update.ModuleCount,
			fileCount:      update.FileCount,
		})
	}

	if err := watch.Subscribe(context.Background(), sendUpdate); err != nil {
		return err
	}

	go func() {
		update, err := watch.CurrentUpdate(context.Background())
		if err != nil {
			return
		}
		sendUpdate(update)
	}()

	_, err := p.Run()
	return err
}

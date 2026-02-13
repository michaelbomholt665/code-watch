package cli

import (
	coreapp "circular/internal/core/app"
	"circular/internal/data/history"
	"circular/internal/data/query"
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func runUI(app *coreapp.App, report *history.TrendReport) error {
	service := query.NewService(app.Graph, nil, "default")
	m := initialModel(service, report)
	p := tea.NewProgram(m, tea.WithAltScreen())

	sendUpdate := func(update coreapp.Update) {
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

	app.SetUpdateHandler(func(update coreapp.Update) {
		sendUpdate(update)
	})

	go func() {
		update := app.CurrentUpdate()
		sendUpdate(update)
	}()

	_, err := p.Run()
	return err
}

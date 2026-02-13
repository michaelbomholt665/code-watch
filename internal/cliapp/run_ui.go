package cliapp

import (
	coreapp "circular/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func runUI(app *coreapp.App) error {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	app.SetUpdateHandler(func(update coreapp.Update) {
		p.Send(updateMsg{
			cycles:         update.Cycles,
			hallucinations: update.Hallucinations,
			moduleCount:    update.ModuleCount,
			fileCount:      update.FileCount,
		})
	})

	go func() {
		update := app.CurrentUpdate()
		p.Send(updateMsg{
			cycles:         update.Cycles,
			hallucinations: update.Hallucinations,
			moduleCount:    update.ModuleCount,
			fileCount:      update.FileCount,
		})
	}()

	_, err := p.Run()
	return err
}

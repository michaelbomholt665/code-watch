// # cmd/circular/ui.go
package main

import (
	"circular/internal/resolver"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			Foreground(lipgloss.Color("#3B82F6")).
			Bold(true).
			Render

	docStyle = lipgloss.NewStyle().Margin(1, 2)

	cycleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")).
			Bold(true)

	hallucinationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FBBF24")).
				Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B")).
			Italic(true)
)

type item struct {
	title, desc string
	isCycle     bool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + i.desc }

type model struct {
	list           list.Model
	cycles         [][]string
	hallucinations []resolver.UnresolvedReference
	lastUpdate     time.Time
	moduleCount    int
	fileCount      int
}

type updateMsg struct {
	cycles         [][]string
	hallucinations []resolver.UnresolvedReference
	moduleCount    int
	fileCount      int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
	case updateMsg:
		m.cycles = msg.cycles
		m.hallucinations = msg.hallucinations
		m.moduleCount = msg.moduleCount
		m.fileCount = msg.fileCount
		m.lastUpdate = time.Now()

		items := []list.Item{}
		for _, c := range m.cycles {
			items = append(items, item{
				title:   "Circular Import",
				desc:    strings.Join(c, " -> "),
				isCycle: true,
			})
		}
		for _, h := range m.hallucinations {
			items = append(items, item{
				title:   "Unresolved Reference",
				desc:    fmt.Sprintf("%s in %s:%d", h.Reference.Name, h.File, h.Reference.Location.Line),
				isCycle: false,
			})
		}
		m.list.SetItems(items)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	status := statusStyle.Render(fmt.Sprintf("Last update: %v | %d files | %d modules",
		m.lastUpdate.Format("15:04:05"), m.fileCount, m.moduleCount))

	var summary string
	if len(m.cycles) == 0 && len(m.hallucinations) == 0 {
		summary = successStyle.Render("✅ System Clean")
	} else {
		summary = fmt.Sprintf("⚠️  %s | %s",
			cycleStyle.Render(fmt.Sprintf("%d Cycles", len(m.cycles))),
			hallucinationStyle.Render(fmt.Sprintf("%d Unresolved", len(m.hallucinations))))
	}

	header := fmt.Sprintf("%s\n%s | %s\n", titleStyle("Circular Dependency Monitor"), status, summary)
	return docStyle.Render(header + "\n" + m.list.View())
}

func initialModel() model {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Detected Issues"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return model{
		list:       l,
		lastUpdate: time.Now(),
	}
}

package cliapp

import (
	"circular/internal/history"
	"circular/internal/query"
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
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + i.desc }

type model struct {
	issueList      list.Model
	moduleList     list.Model
	mode           panelMode
	querySvc       *query.Service
	trendReport    *history.TrendReport
	showTrend      bool
	cycles         [][]string
	hallucinations []resolver.UnresolvedReference
	modules        []query.ModuleSummary
	lastUpdate     time.Time
	moduleCount    int
	fileCount      int

	moduleDetails    query.ModuleDetails
	hasModuleDetails bool
	moduleDetailsErr string
	selectedDepIndex int
	sourceJumpStatus string
}

type panelMode int

const (
	panelIssues panelMode = iota
	panelModules
)

type updateMsg struct {
	cycles         [][]string
	hallucinations []resolver.UnresolvedReference
	modules        []query.ModuleSummary
	moduleCount    int
	fileCount      int
}

type sourceJumpResultMsg struct {
	target string
	err    error
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return handleKeyActions(msg, m)
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		width := msg.Width - h
		height := msg.Height - v - 8
		if height < 5 {
			height = 5
		}
		m.issueList.SetSize(width, height)
		m.moduleList.SetSize(width, height)
	case updateMsg:
		m.cycles = msg.cycles
		m.hallucinations = msg.hallucinations
		m.modules = msg.modules
		m.moduleCount = msg.moduleCount
		m.fileCount = msg.fileCount
		m.lastUpdate = time.Now()
		m.moduleDetailsErr = ""

		items := []list.Item{}
		for _, c := range m.cycles {
			items = append(items, item{
				title: "Circular Import",
				desc:  strings.Join(c, " -> "),
			})
		}
		for _, h := range m.hallucinations {
			items = append(items, item{
				title: "Unresolved Reference",
				desc:  fmt.Sprintf("%s in %s:%d", h.Reference.Name, h.File, h.Reference.Location.Line),
			})
		}
		m.issueList.SetItems(items)

		moduleItems := make([]list.Item, 0, len(m.modules))
		for _, module := range m.modules {
			moduleItems = append(moduleItems, item{
				title: module.Name,
				desc: fmt.Sprintf(
					"files=%d exports=%d deps=%d imported_by=%d",
					module.FileCount,
					module.ExportCount,
					module.DependencyCount,
					module.ReverseDependencyCount,
				),
			})
		}
		m.moduleList.SetItems(moduleItems)
		if m.hasModuleDetails {
			m, _ = refreshModuleDetails(m)
		}
	case sourceJumpResultMsg:
		if msg.err != nil {
			m.sourceJumpStatus = statusStyle.Render(fmt.Sprintf("Source jump failed: %v", msg.err))
		} else {
			m.sourceJumpStatus = statusStyle.Render(fmt.Sprintf("Opened source: %s", msg.target))
		}
	}

	var cmd tea.Cmd
	if m.mode == panelIssues {
		m.issueList, cmd = m.issueList.Update(msg)
	} else {
		m.moduleList, cmd = m.moduleList.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	status := statusStyle.Render(fmt.Sprintf("Last update: %v | %d files | %d modules",
		m.lastUpdate.Format("15:04:05"), m.fileCount, m.moduleCount))

	var summary string
	if len(m.cycles) == 0 && len(m.hallucinations) == 0 {
		summary = successStyle.Render("System Clean")
	} else {
		summary = fmt.Sprintf("%s | %s",
			cycleStyle.Render(fmt.Sprintf("%d cycles", len(m.cycles))),
			hallucinationStyle.Render(fmt.Sprintf("%d unresolved", len(m.hallucinations))))
	}

	header := fmt.Sprintf("%s\n%s | %s\n", titleStyle("Circular Dependency Monitor"), status, summary)
	help := renderHelp(m)

	body := m.issueList.View()
	if m.mode == panelModules {
		body = renderModulePanel(m)
	}
	if m.showTrend {
		body += "\n\n" + renderTrendOverlay(m.trendReport)
	}
	if m.sourceJumpStatus != "" {
		body += "\n\n" + m.sourceJumpStatus
	}

	return docStyle.Render(header + "\n" + help + "\n\n" + body)
}

func initialModel(service *query.Service, trendReport *history.TrendReport) model {
	issueList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	issueList.Title = "Detected Issues"
	issueList.SetShowStatusBar(false)
	issueList.SetFilteringEnabled(true)

	moduleList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	moduleList.Title = "Module Explorer"
	moduleList.SetShowStatusBar(false)
	moduleList.SetFilteringEnabled(true)

	return model{
		issueList:        issueList,
		moduleList:       moduleList,
		mode:             panelIssues,
		querySvc:         service,
		trendReport:      trendReport,
		lastUpdate:       time.Now(),
		selectedDepIndex: 0,
	}
}

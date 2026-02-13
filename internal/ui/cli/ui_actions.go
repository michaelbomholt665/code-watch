package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func handleKeyActions(msg tea.KeyMsg, m model) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		if m.mode == panelIssues {
			m.mode = panelModules
		} else {
			m.mode = panelIssues
		}
		return m, nil
	case "t":
		m.showTrend = !m.showTrend
		return m, nil
	}

	if m.mode != panelModules {
		var cmd tea.Cmd
		m.issueList, cmd = m.issueList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter":
		return refreshModuleDetails(m)
	case "esc", "backspace":
		m.hasModuleDetails = false
		m.moduleDetailsErr = ""
		m.selectedDepIndex = 0
		return m, nil
	case "j":
		if m.hasModuleDetails && len(m.moduleDetails.Dependencies) > 0 {
			if m.selectedDepIndex < len(m.moduleDetails.Dependencies)-1 {
				m.selectedDepIndex++
			}
			return m, nil
		}
	case "k":
		if m.hasModuleDetails && len(m.moduleDetails.Dependencies) > 0 {
			if m.selectedDepIndex > 0 {
				m.selectedDepIndex--
			}
			return m, nil
		}
	case "o":
		if !m.hasModuleDetails {
			return m, nil
		}
		target, ok := selectedSourceTarget(m)
		if !ok {
			m.sourceJumpStatus = statusStyle.Render("No source target available.")
			return m, nil
		}
		return m, jumpToSourceCmd(target)
	}

	var cmd tea.Cmd
	m.moduleList, cmd = m.moduleList.Update(msg)
	return m, cmd
}

func refreshModuleDetails(m model) (model, tea.Cmd) {
	if m.querySvc == nil || len(m.modules) == 0 {
		return m, nil
	}
	idx := m.moduleList.Index()
	if idx < 0 || idx >= len(m.modules) {
		idx = 0
	}
	details, err := m.querySvc.ModuleDetails(context.Background(), m.modules[idx].Name)
	if err != nil {
		m.moduleDetailsErr = err.Error()
		m.hasModuleDetails = false
		return m, nil
	}
	m.moduleDetails = details
	m.moduleDetailsErr = ""
	m.hasModuleDetails = true
	m.selectedDepIndex = 0
	return m, nil
}

type sourceTarget struct {
	file string
	line int
}

func selectedSourceTarget(m model) (sourceTarget, bool) {
	if len(m.moduleDetails.Dependencies) > 0 {
		idx := m.selectedDepIndex
		if idx < 0 {
			idx = 0
		}
		if idx >= len(m.moduleDetails.Dependencies) {
			idx = len(m.moduleDetails.Dependencies) - 1
		}
		dep := m.moduleDetails.Dependencies[idx]
		return sourceTarget{file: dep.File, line: dep.Line}, dep.File != ""
	}
	if len(m.moduleDetails.Files) > 0 {
		return sourceTarget{file: m.moduleDetails.Files[0], line: 1}, true
	}
	return sourceTarget{}, false
}

func jumpToSourceCmd(target sourceTarget) tea.Cmd {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = "vi"
	}
	args := []string{target.file}
	if strings.Contains(editor, "vim") || strings.Contains(editor, "nvim") || strings.HasSuffix(editor, "/vi") {
		args = []string{fmt.Sprintf("+%d", target.line), target.file}
	}
	cmd := exec.Command(editor, args...)
	label := fmt.Sprintf("%s:%d", target.file, target.line)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return sourceJumpResultMsg{target: label, err: err}
	})
}

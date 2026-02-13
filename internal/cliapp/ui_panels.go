package cliapp

import (
	"circular/internal/history"
	"fmt"
	"strings"
)

func renderHelp(m model) string {
	keys := "Keys: tab panel | / filter | enter details | esc back | t trend overlay | j/k dependency cursor | o open source | q quit"
	if m.mode == panelIssues {
		keys = "Keys: tab panel | / filter | q quit"
	}
	return statusStyle.Render(keys)
}

func renderModulePanel(m model) string {
	summary := m.moduleList.View()
	details := renderModuleSummary(m)
	if m.hasModuleDetails {
		details = renderModuleDetails(m)
	}
	return summary + "\n\n" + details
}

func renderModuleSummary(m model) string {
	if len(m.modules) == 0 {
		return statusStyle.Render("No modules available.")
	}
	idx := m.moduleList.Index()
	if idx < 0 || idx >= len(m.modules) {
		idx = 0
	}
	selected := m.modules[idx]
	return strings.Join([]string{
		"Selected Module",
		fmt.Sprintf("  Name: %s", selected.Name),
		fmt.Sprintf("  Files: %d", selected.FileCount),
		fmt.Sprintf("  Exports: %d", selected.ExportCount),
		fmt.Sprintf("  Dependencies: %d", selected.DependencyCount),
		fmt.Sprintf("  Imported by: %d", selected.ReverseDependencyCount),
		"  Press enter for dependency/file drill-down.",
	}, "\n")
}

func renderModuleDetails(m model) string {
	if m.moduleDetailsErr != "" {
		return cycleStyle.Render("Module details error: " + m.moduleDetailsErr)
	}
	d := m.moduleDetails
	lines := []string{
		fmt.Sprintf("Module Detail: %s", d.Name),
		fmt.Sprintf("  Files (%d): %s", len(d.Files), strings.Join(d.Files, ", ")),
		fmt.Sprintf("  Exports (%d): %s", len(d.ExportedSymbols), strings.Join(d.ExportedSymbols, ", ")),
		fmt.Sprintf("  Reverse dependencies (%d): %s", len(d.ReverseDependencies), strings.Join(d.ReverseDependencies, ", ")),
		fmt.Sprintf("  Dependencies (%d):", len(d.Dependencies)),
	}
	for i, edge := range d.Dependencies {
		prefix := "   "
		if i == m.selectedDepIndex {
			prefix = " ->"
		}
		lines = append(lines, fmt.Sprintf("%s %s (from %s:%d)", prefix, edge.To, edge.File, edge.Line))
	}
	if len(d.Dependencies) == 0 {
		lines = append(lines, "   none")
	}
	lines = append(lines, "  Press esc to exit details, o to jump to highlighted source.")
	return strings.Join(lines, "\n")
}

func renderTrendOverlay(report *history.TrendReport) string {
	if report == nil || len(report.Points) == 0 {
		return statusStyle.Render("Trend overlay unavailable (enable --history to capture snapshots).")
	}
	last := report.Points[len(report.Points)-1]
	return strings.Join([]string{
		"Trend Overlay",
		fmt.Sprintf("  Window: %s | Scans: %d", report.Window, report.ScanCount),
		fmt.Sprintf("  Module growth: %+d (%.2f%%)", last.DeltaModules, last.ModuleGrowthPct),
		fmt.Sprintf("  Fan-in drift: %+0.2f (avg %.2f, max %d)", last.DeltaAvgFanIn, last.AvgFanIn, last.MaxFanIn),
		fmt.Sprintf("  Fan-out drift: %+0.2f (avg %.2f, max %d)", last.DeltaAvgFanOut, last.AvgFanOut, last.MaxFanOut),
		fmt.Sprintf("  Cycles delta: %+d | Unresolved delta: %+d", last.DeltaCycles, last.DeltaUnresolved),
	}, "\n")
}

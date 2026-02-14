package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/resolver"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MarkdownReportData struct {
	TotalModules int
	TotalFiles   int

	Cycles        [][]string
	Unresolved    []resolver.UnresolvedReference
	UnusedImports []resolver.UnusedImport
	Violations    []graph.ArchitectureViolation
	Hotspots      []graph.ComplexityHotspot
}

type MarkdownReportOptions struct {
	ProjectName         string
	ProjectRoot         string
	Version             string
	GeneratedAt         time.Time
	Verbosity           string
	TableOfContents     bool
	CollapsibleSections bool
	IncludeMermaid      bool
	MermaidDiagram      string
}

type MarkdownGenerator struct{}

func NewMarkdownGenerator() *MarkdownGenerator {
	return &MarkdownGenerator{}
}

func (m *MarkdownGenerator) Generate(data MarkdownReportData, opts MarkdownReportOptions) (string, error) {
	if opts.GeneratedAt.IsZero() {
		opts.GeneratedAt = time.Now().UTC()
	}
	verbosity := normalizeReportVerbosity(opts.Verbosity)

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: Code Analysis Report\n")
	b.WriteString("project: " + nonEmpty(opts.ProjectName, "unknown") + "\n")
	b.WriteString("generated_at: " + opts.GeneratedAt.UTC().Format(time.RFC3339) + "\n")
	b.WriteString("version: " + nonEmpty(opts.Version, "unknown") + "\n")
	b.WriteString("---\n\n")

	b.WriteString("# Analysis Report\n\n")
	if opts.TableOfContents {
		b.WriteString("## Table of Contents\n")
		b.WriteString("- [Executive Summary](#executive-summary)\n")
		b.WriteString("- [Circular Imports](#circular-imports)\n")
		b.WriteString("- [Architecture Violations](#architecture-violations)\n")
		b.WriteString("- [Complexity Hotspots](#complexity-hotspots)\n")
		b.WriteString("- [Unresolved References](#unresolved-references)\n")
		b.WriteString("- [Unused Imports](#unused-imports)\n")
		if opts.IncludeMermaid && strings.TrimSpace(opts.MermaidDiagram) != "" {
			b.WriteString("- [Dependency Diagram](#dependency-diagram)\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Executive Summary\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("| --- | --- |\n")
	b.WriteString(fmt.Sprintf("| Total Modules | %d |\n", data.TotalModules))
	b.WriteString(fmt.Sprintf("| Total Files | %d |\n", data.TotalFiles))
	b.WriteString(fmt.Sprintf("| Circular Imports | %d |\n", len(data.Cycles)))
	b.WriteString(fmt.Sprintf("| Architecture Violations | %d |\n", len(data.Violations)))
	b.WriteString(fmt.Sprintf("| Complexity Hotspots | %d |\n", len(data.Hotspots)))
	b.WriteString(fmt.Sprintf("| Unresolved References | %d |\n", len(data.Unresolved)))
	b.WriteString(fmt.Sprintf("| Unused Imports | %d |\n\n", len(data.UnusedImports)))

	m.writeCycles(&b, data.Cycles, opts.CollapsibleSections)
	m.writeViolations(&b, data.Violations, opts.ProjectRoot, opts.CollapsibleSections)
	m.writeHotspots(&b, data.Hotspots, opts.ProjectRoot, opts.CollapsibleSections, verbosity)
	m.writeUnresolved(&b, data.Unresolved, opts.ProjectRoot, opts.CollapsibleSections)
	m.writeUnusedImports(&b, data.UnusedImports, opts.ProjectRoot, opts.CollapsibleSections, verbosity)

	if opts.IncludeMermaid && strings.TrimSpace(opts.MermaidDiagram) != "" {
		b.WriteString("## Dependency Diagram\n")
		b.WriteString("```mermaid\n")
		b.WriteString(strings.TrimSpace(opts.MermaidDiagram))
		b.WriteString("\n```\n")
	}

	return b.String(), nil
}

func (m *MarkdownGenerator) writeCycles(b *strings.Builder, cycles [][]string, collapsible bool) {
	b.WriteString("## Circular Imports\n")
	if len(cycles) == 0 {
		b.WriteString("No circular imports detected.\n\n")
		return
	}
	rows := make([]string, 0, len(cycles))
	for i, cycle := range cycles {
		nodes := append([]string(nil), cycle...)
		sort.Strings(nodes)
		impact := "🟡 Medium"
		if len(cycle) >= 4 {
			impact = "🔴 High"
		}
		rows = append(rows, fmt.Sprintf("| %d | `%s` | %s | %d |\n", i+1, strings.Join(cycle, " -> "), impact, len(cycle)*10))
	}
	m.writeTableWithCollapse(
		b,
		"Cycle details",
		collapsible,
		len(rows) > 10,
		[]string{"| # | Cycle Path | Impact | Impact Score |\n", "| --- | --- | --- | --- |\n"},
		rows,
	)
}

func (m *MarkdownGenerator) writeViolations(b *strings.Builder, rows []graph.ArchitectureViolation, projectRoot string, collapsible bool) {
	b.WriteString("## Architecture Violations\n")
	if len(rows) == 0 {
		b.WriteString("No architecture violations detected.\n\n")
		return
	}
	rendered := make([]string, 0, len(rows))
	for _, row := range rows {
		rendered = append(rendered, fmt.Sprintf(
			"| `%s` | `%s` | `%s` | `%s` | `%s` | `%s:%d:%d` |\n",
			row.RuleName,
			row.FromLayer,
			row.ToLayer,
			row.FromModule,
			row.ToModule,
			relPath(projectRoot, row.File),
			row.Line,
			row.Column,
		))
	}
	m.writeTableWithCollapse(
		b,
		"Violation details",
		collapsible,
		len(rendered) > 10,
		[]string{"| Rule | From Layer | To Layer | From Module | To Module | Location |\n", "| --- | --- | --- | --- | --- | --- |\n"},
		rendered,
	)
}

func (m *MarkdownGenerator) writeHotspots(b *strings.Builder, hotspots []graph.ComplexityHotspot, projectRoot string, collapsible bool, verbosity string) {
	b.WriteString("## Complexity Hotspots\n")
	if len(hotspots) == 0 {
		b.WriteString("No complexity hotspots detected.\n\n")
		return
	}
	rendered := make([]string, 0, len(hotspots))
	for _, row := range hotspots {
		if verbosity == "summary" {
			rendered = append(rendered, fmt.Sprintf("| `%s` | `%s` | %d |\n", row.Module, row.Definition, row.Score))
			continue
		}
		rendered = append(rendered, fmt.Sprintf(
			"| `%s` | `%s` | `%s` | %d | %d | %d | %d | %d |\n",
			row.Module,
			row.Definition,
			relPath(projectRoot, row.File),
			row.Score,
			row.Branches,
			row.Parameters,
			row.Nesting,
			row.LOC,
		))
	}
	if verbosity == "summary" {
		m.writeTableWithCollapse(
			b,
			"Hotspot details",
			collapsible,
			len(rendered) > 10,
			[]string{"| Module | Definition | Score |\n", "| --- | --- | --- |\n"},
			rendered,
		)
		return
	}
	m.writeTableWithCollapse(
		b,
		"Hotspot details",
		collapsible,
		len(rendered) > 10,
		[]string{"| Module | Definition | File | Score | Branches | Params | Nesting | LOC |\n", "| --- | --- | --- | --- | --- | --- | --- | --- |\n"},
		rendered,
	)
}

func (m *MarkdownGenerator) writeUnresolved(b *strings.Builder, rows []resolver.UnresolvedReference, projectRoot string, collapsible bool) {
	b.WriteString("## Unresolved References\n")
	if len(rows) == 0 {
		b.WriteString("No unresolved references detected.\n\n")
		return
	}
	rendered := make([]string, 0, len(rows))
	for _, row := range rows {
		rendered = append(rendered, fmt.Sprintf(
			"| `%s` | `%s:%d:%d` |\n",
			row.Reference.Name,
			relPath(projectRoot, row.File),
			row.Reference.Location.Line,
			row.Reference.Location.Column,
		))
	}
	m.writeTableWithCollapse(
		b,
		"Unresolved reference details",
		collapsible,
		len(rendered) > 15,
		[]string{"| Reference | Location |\n", "| --- | --- |\n"},
		rendered,
	)
}

func (m *MarkdownGenerator) writeUnusedImports(b *strings.Builder, rows []resolver.UnusedImport, projectRoot string, collapsible bool, verbosity string) {
	b.WriteString("## Unused Imports\n")
	if len(rows) == 0 {
		b.WriteString("No unused imports detected.\n\n")
		return
	}
	rendered := make([]string, 0, len(rows))
	for _, row := range rows {
		location := fmt.Sprintf("%s:%d:%d", relPath(projectRoot, row.File), row.Location.Line, row.Location.Column)
		target := row.Module
		if row.Item != "" {
			target = target + "." + row.Item
		}
		if verbosity == "summary" {
			rendered = append(rendered, fmt.Sprintf("| `%s` | `%s` | `%s` |\n", row.Language, target, location))
			continue
		}
		rendered = append(rendered, fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n", row.Language, row.Module, row.Alias, row.Item, row.Confidence, location))
	}
	if verbosity == "summary" {
		m.writeTableWithCollapse(
			b,
			"Unused import details",
			collapsible,
			len(rendered) > 15,
			[]string{"| Language | Import | Location |\n", "| --- | --- | --- |\n"},
			rendered,
		)
		return
	}
	m.writeTableWithCollapse(
		b,
		"Unused import details",
		collapsible,
		len(rendered) > 15,
		[]string{"| Language | Module | Alias | Item | Confidence | Location |\n", "| --- | --- | --- | --- | --- | --- |\n"},
		rendered,
	)
}

func (m *MarkdownGenerator) writeTableWithCollapse(
	b *strings.Builder,
	summary string,
	collapsible bool,
	collapse bool,
	header []string,
	rows []string,
) {
	if collapsible && collapse {
		b.WriteString("<details>\n")
		b.WriteString("<summary>")
		b.WriteString(summary)
		b.WriteString("</summary>\n\n")
	}
	for _, line := range header {
		b.WriteString(line)
	}
	for _, line := range rows {
		b.WriteString(line)
	}
	b.WriteString("\n")
	if collapsible && collapse {
		b.WriteString("</details>\n\n")
	}
}

func relPath(root, path string) string {
	root = strings.TrimSpace(root)
	path = strings.TrimSpace(path)
	if root == "" || path == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func normalizeReportVerbosity(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "summary":
		return "summary"
	case "detailed":
		return "detailed"
	default:
		return "standard"
	}
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

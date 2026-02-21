package app

import (
	"circular/internal/core/ports"
	"circular/internal/data/query"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (a *App) TraceImportChain(from, to string) (string, error) {
	if _, ok := a.Graph.GetModule(from); !ok {
		return "", fmt.Errorf("source module not found: %s", from)
	}
	if _, ok := a.Graph.GetModule(to); !ok {
		return "", fmt.Errorf("target module not found: %s", to)
	}

	chain, ok := a.Graph.FindImportChain(from, to)
	if !ok {
		return "", fmt.Errorf("no import chain found from %s to %s", from, to)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Import chain: %s -> %s\n\n", from, to))
	for i, module := range chain {
		b.WriteString(module)
		b.WriteString("\n")
		if i < len(chain)-1 {
			b.WriteString("  -> ")
		}
	}

	return strings.TrimRight(b.String(), "\n"), nil
}

func (a *App) AnalyzeImpact(path string) (graph.ImpactReport, error) {
	return a.Graph.AnalyzeImpact(path)
}

func (a *App) ArchitectureViolations() []graph.ArchitectureViolation {
	return a.archEngine.Validate(a.Graph)
}

func (a *App) BuildQueryService(historyStore ports.HistoryStore, projectKey string) *query.Service {
	return query.NewService(a.Graph, historyStore, projectKey)
}

func (a *App) SecretCount() int {
	total := 0
	for _, file := range a.Graph.GetAllFiles() {
		total += len(file.Secrets)
	}
	return total
}

func (a *App) allSecrets(limit int) []parser.Secret {
	all := make([]parser.Secret, 0)
	for _, file := range a.Graph.GetAllFiles() {
		if file == nil || len(file.Secrets) == 0 {
			continue
		}
		all = append(all, file.Secrets...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Location.File != all[j].Location.File {
			return all[i].Location.File < all[j].Location.File
		}
		if all[i].Location.Line != all[j].Location.Line {
			return all[i].Location.Line < all[j].Location.Line
		}
		if all[i].Location.Column != all[j].Location.Column {
			return all[i].Location.Column < all[j].Location.Column
		}
		return all[i].Kind < all[j].Kind
	})
	if limit > 0 && len(all) > limit {
		return append([]parser.Secret(nil), all[:limit]...)
	}
	return all
}

func (a *App) GenerateMarkdownReport(ctx context.Context, req MarkdownReportRequest) (MarkdownReportResult, error) {
	return newPresentationService(a).GenerateMarkdownReport(ctx, req)
}

func (a *App) PrintSummary(
	fileCount, moduleCount int,
	duration time.Duration,
	cycles [][]string,
	hallucinations []resolver.UnresolvedReference,
	unusedImports []resolver.UnusedImport,
	metrics map[string]graph.ModuleMetrics,
	violations []graph.ArchitectureViolation,
	hotspots []graph.ComplexityHotspot,
) {
	newPresentationService(a).PrintSummary(fileCount, moduleCount, duration, cycles, hallucinations, unusedImports, metrics, violations, hotspots)
}

package adapters

import (
	"circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/data/history"
	"circular/internal/data/query"
	"circular/internal/engine/parser"
	"circular/internal/engine/secrets"
	"circular/internal/mcp/contracts"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Adapter struct {
	mu         sync.RWMutex
	app        *app.App
	history    *history.Store
	projectKey string
}

func NewAdapter(app *app.App, historyStore *history.Store, projectKey string) *Adapter {
	return &Adapter{
		app:        app,
		history:    historyStore,
		projectKey: strings.TrimSpace(projectKey),
	}
}

func (a *Adapter) SetProjectKey(projectKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.projectKey = strings.TrimSpace(projectKey)
}

func (a *Adapter) ProjectKey() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.projectKey
}

func (a *Adapter) RunScan(ctx context.Context, in contracts.ScanRunInput) (contracts.ScanRunOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.ScanRunOutput{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	start := time.Now()
	warnings := make([]string, 0)
	filesScanned := 0

	if len(in.Paths) > 0 {
		paths := normalizeScanPaths(in.Paths)
		files, err := a.app.ScanDirectories(paths, a.app.Config.Exclude.Dirs, a.app.Config.Exclude.Files)
		if err != nil {
			return contracts.ScanRunOutput{}, err
		}
		filesScanned = len(files)
		for _, filePath := range files {
			if err := a.app.ProcessFile(filePath); err != nil {
				warnings = append(warnings, fmt.Sprintf("process file %s: %v", filePath, err))
			}
		}
	} else {
		if err := a.app.InitialScan(); err != nil {
			return contracts.ScanRunOutput{}, err
		}
		filesScanned = a.app.Graph.FileCount()
	}

	return contracts.ScanRunOutput{
		FilesScanned: filesScanned,
		Modules:      a.app.Graph.ModuleCount(),
		DurationMs:   int(time.Since(start).Milliseconds()),
		Warnings:     warnings,
	}, nil
}

func (a *Adapter) ScanSecrets(ctx context.Context, in contracts.SecretsScanInput) (contracts.SecretsScanOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.SecretsScanOutput{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	warnings := make([]string, 0)
	filesScanned := 0

	if len(in.Paths) > 0 {
		paths := normalizeScanPaths(in.Paths)
		files, err := a.app.ScanDirectories(paths, a.app.Config.Exclude.Dirs, a.app.Config.Exclude.Files)
		if err != nil {
			return contracts.SecretsScanOutput{}, err
		}
		filesScanned = len(files)
		for _, filePath := range files {
			if err := a.app.ProcessFile(filePath); err != nil {
				warnings = append(warnings, fmt.Sprintf("process file %s: %v", filePath, err))
			}
		}
	} else {
		if err := a.app.InitialScan(); err != nil {
			return contracts.SecretsScanOutput{}, err
		}
		filesScanned = a.app.Graph.FileCount()
	}

	findings, total := secretFindings(a.app.Graph.GetAllFiles(), 0)
	return contracts.SecretsScanOutput{
		FilesScanned: filesScanned,
		SecretCount:  total,
		Findings:     findings,
		Warnings:     warnings,
	}, nil
}

func (a *Adapter) ListSecrets(ctx context.Context, limit int) (contracts.SecretsListOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.SecretsListOutput{}, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	findings, total := secretFindings(a.app.Graph.GetAllFiles(), limit)
	return contracts.SecretsListOutput{
		SecretCount: total,
		Findings:    findings,
	}, nil
}

func (a *Adapter) Cycles(ctx context.Context, limit int) (contracts.GraphCyclesOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.GraphCyclesOutput{}, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	cycles := a.app.Graph.DetectCycles()
	count := len(cycles)
	if limit > 0 && count > limit {
		cycles = cycles[:limit]
	}

	return contracts.GraphCyclesOutput{
		CycleCount: count,
		Cycles:     cycles,
	}, nil
}

func (a *Adapter) ListModules(ctx context.Context, filter string, limit int) (contracts.QueryModulesOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.QueryModulesOutput{}, err
	}

	rows, err := a.queryService().ListModules(ctx, filter, limit)
	if err != nil {
		return contracts.QueryModulesOutput{}, err
	}

	out := make([]contracts.ModuleSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, contracts.ModuleSummary{
			Name:                   row.Name,
			FileCount:              row.FileCount,
			ExportCount:            row.ExportCount,
			DependencyCount:        row.DependencyCount,
			ReverseDependencyCount: row.ReverseDependencyCount,
		})
	}

	return contracts.QueryModulesOutput{Modules: out}, nil
}

func (a *Adapter) ModuleDetails(ctx context.Context, module string) (contracts.QueryModuleDetailsOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.QueryModuleDetailsOutput{}, err
	}

	details, err := a.queryService().ModuleDetails(ctx, module)
	if err != nil {
		return contracts.QueryModuleDetailsOutput{}, err
	}

	deps := make([]contracts.DependencyEdge, 0, len(details.Dependencies))
	for _, edge := range details.Dependencies {
		deps = append(deps, contracts.DependencyEdge{
			From:   edge.From,
			To:     edge.To,
			File:   edge.File,
			Line:   edge.Line,
			Column: edge.Column,
		})
	}

	return contracts.QueryModuleDetailsOutput{
		Module: contracts.ModuleDetails{
			Name:                details.Name,
			Files:               append([]string(nil), details.Files...),
			ExportedSymbols:     append([]string(nil), details.ExportedSymbols...),
			Dependencies:        deps,
			ReverseDependencies: append([]string(nil), details.ReverseDependencies...),
		},
	}, nil
}

func (a *Adapter) Trace(ctx context.Context, from, to string, maxDepth int) (contracts.QueryTraceOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.QueryTraceOutput{}, err
	}

	trace, err := a.queryService().DependencyTrace(ctx, from, to, maxDepth)
	if err != nil {
		return contracts.QueryTraceOutput{}, err
	}

	return contracts.QueryTraceOutput{
		Found: true,
		Path:  append([]string(nil), trace.Path...),
		Depth: trace.Depth,
	}, nil
}

func (a *Adapter) TrendSlice(ctx context.Context, since time.Time, limit int) (contracts.QueryTrendsOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.QueryTrendsOutput{}, err
	}

	slice, err := a.queryService().TrendSlice(ctx, since, limit)
	if err != nil {
		return contracts.QueryTrendsOutput{}, err
	}

	out := contracts.QueryTrendsOutput{
		Since:     slice.Since,
		Until:     slice.Until,
		ScanCount: slice.ScanCount,
		Snapshots: make([]contracts.TrendSnapshot, 0, len(slice.Snapshots)),
	}
	for _, snapshot := range slice.Snapshots {
		out.Snapshots = append(out.Snapshots, contracts.TrendSnapshot{
			Timestamp: snapshot.Timestamp.Format(time.RFC3339),
			Modules:   snapshot.ModuleCount,
			Files:     snapshot.FileCount,
		})
	}

	return out, nil
}

func (a *Adapter) SyncOutputs(ctx context.Context, formats []string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	originalOutput := a.app.Config.Output
	filteredOutput := originalOutput
	formatSet := formatSet(formats)
	if len(formatSet) > 0 {
		if !formatSet["dot"] {
			filteredOutput.DOT = ""
		}
		if !formatSet["tsv"] {
			filteredOutput.TSV = ""
		}
		if !formatSet["mermaid"] {
			filteredOutput.Mermaid = ""
		}
		if !formatSet["plantuml"] {
			filteredOutput.PlantUML = ""
		}
		if !formatSet["markdown"] {
			filteredOutput.Markdown = ""
		}

		filteredUpdate := make([]config.MarkdownInjection, 0, len(filteredOutput.UpdateMarkdown))
		for _, injection := range filteredOutput.UpdateMarkdown {
			if formatSet[strings.ToLower(strings.TrimSpace(injection.Format))] {
				filteredUpdate = append(filteredUpdate, injection)
			}
		}
		filteredOutput.UpdateMarkdown = filteredUpdate
	}

	cfgCopy := *a.app.Config
	cfgCopy.Output = filteredOutput
	writeTargets, err := resolveOutputTargets(&cfgCopy)
	if err != nil {
		return nil, err
	}

	cycles := a.app.Graph.DetectCycles()
	metrics := a.app.Graph.ComputeModuleMetrics()
	hotspots := a.app.Graph.TopComplexity(a.app.Config.Architecture.TopComplexity)
	violations := a.app.ArchitectureViolations()
	unused := a.app.AnalyzeUnusedImports()

	a.app.Config.Output = filteredOutput
	err = a.app.GenerateOutputs(cycles, nil, unused, metrics, violations, hotspots)
	a.app.Config.Output = originalOutput
	if err != nil {
		return nil, err
	}

	written := make([]string, 0, 4)
	if writeTargets.DOT != "" {
		written = append(written, writeTargets.DOT)
	}
	if writeTargets.TSV != "" {
		written = append(written, writeTargets.TSV)
	}
	if writeTargets.Mermaid != "" {
		written = append(written, writeTargets.Mermaid)
	}
	if writeTargets.PlantUML != "" {
		written = append(written, writeTargets.PlantUML)
	}
	if writeTargets.Markdown != "" {
		written = append(written, writeTargets.Markdown)
	}
	for _, injection := range filteredOutput.UpdateMarkdown {
		if strings.TrimSpace(injection.File) != "" {
			written = append(written, injection.File)
		}
	}

	return uniqueStrings(written), nil
}

func (a *Adapter) GenerateMarkdownReport(ctx context.Context, in contracts.ReportGenerateMarkdownInput) (contracts.ReportGenerateMarkdownOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.ReportGenerateMarkdownOutput{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	result, err := a.app.GenerateMarkdownReport(app.MarkdownReportRequest{
		OutputPath: in.Path,
		WriteFile:  in.WriteFile || strings.TrimSpace(in.Path) != "",
		Verbosity:  in.Verbosity,
	})
	if err != nil {
		return contracts.ReportGenerateMarkdownOutput{}, err
	}
	return contracts.ReportGenerateMarkdownOutput{
		Markdown: result.Markdown,
		Path:     result.Path,
		Written:  result.Written,
	}, nil
}

func (a *Adapter) queryService() *query.Service {
	return query.NewService(a.app.Graph, a.history, a.ProjectKey())
}

func normalizeScanPaths(paths []string) []string {
	cleaned := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		abs := trimmed
		if absPath, err := filepath.Abs(trimmed); err == nil {
			abs = absPath
		}
		abs = filepath.Clean(abs)
		if seen[abs] {
			continue
		}
		seen[abs] = true
		cleaned = append(cleaned, abs)
	}
	return cleaned
}

func formatSet(formats []string) map[string]bool {
	if len(formats) == 0 {
		return nil
	}
	out := make(map[string]bool, len(formats))
	for _, format := range formats {
		trimmed := strings.ToLower(strings.TrimSpace(format))
		if trimmed == "" {
			continue
		}
		out[trimmed] = true
	}
	return out
}

type outputTargets struct {
	DOT      string
	TSV      string
	Mermaid  string
	PlantUML string
	Markdown string
}

func resolveOutputTargets(cfg *config.Config) (outputTargets, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return outputTargets{}, err
	}

	paths, err := config.ResolvePaths(cfg, cwd)
	if err != nil {
		return outputTargets{}, fmt.Errorf("resolve output root: %w", err)
	}
	root := paths.OutputRoot

	diagramsDir := strings.TrimSpace(cfg.Output.Paths.DiagramsDir)
	if diagramsDir == "" {
		diagramsDir = "docs/diagrams"
	}
	if !filepath.IsAbs(diagramsDir) {
		diagramsDir = filepath.Join(root, diagramsDir)
	}

	return outputTargets{
		DOT:      resolveOutputPath(strings.TrimSpace(cfg.Output.DOT), root),
		TSV:      resolveOutputPath(strings.TrimSpace(cfg.Output.TSV), root),
		Mermaid:  resolveDiagramPath(strings.TrimSpace(cfg.Output.Mermaid), root, diagramsDir),
		PlantUML: resolveDiagramPath(strings.TrimSpace(cfg.Output.PlantUML), root, diagramsDir),
		Markdown: resolveOutputPath(strings.TrimSpace(cfg.Output.Markdown), root),
	}, nil
}

func resolveOutputPath(path, root string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

func resolveDiagramPath(path, root, diagramsDir string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if strings.Contains(path, string(os.PathSeparator)) || strings.Contains(path, "/") {
		return filepath.Join(root, path)
	}
	return filepath.Join(diagramsDir, path)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func secretFindings(files []*parser.File, limit int) ([]contracts.SecretFinding, int) {
	findings := make([]contracts.SecretFinding, 0)
	for _, file := range files {
		if file == nil || len(file.Secrets) == 0 {
			continue
		}
		for _, finding := range file.Secrets {
			findings = append(findings, contracts.SecretFinding{
				Kind:        finding.Kind,
				Severity:    finding.Severity,
				ValueMasked: secrets.MaskValue(finding.Value),
				Entropy:     finding.Entropy,
				Confidence:  finding.Confidence,
				File:        finding.Location.File,
				Line:        finding.Location.Line,
				Column:      finding.Location.Column,
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Column != findings[j].Column {
			return findings[i].Column < findings[j].Column
		}
		return findings[i].Kind < findings[j].Kind
	})

	total := len(findings)
	if limit > 0 && len(findings) > limit {
		findings = findings[:limit]
	}
	return findings, total
}

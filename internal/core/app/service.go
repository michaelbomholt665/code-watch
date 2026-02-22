package app

import (
	"circular/internal/core/config"
	"circular/internal/core/errors"
	"circular/internal/core/ports"
	"circular/internal/data/history"
	"circular/internal/data/query"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"circular/internal/shared/observability"
	"circular/internal/shared/util"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type analysisService struct {
	app *App
}

var _ ports.AnalysisService = (*analysisService)(nil)

func NewAnalysisService(app *App) ports.AnalysisService {
	return &analysisService{app: app}
}

func (s *analysisService) Unwrap() *App {
	return s.app
}

func (s *analysisService) Close(ctx context.Context) error {
	if s == nil || s.app == nil {
		return nil
	}
	return s.app.Close(ctx)
}

func (a *App) AnalysisService() ports.AnalysisService {
	return NewAnalysisService(a)
}

func (s *analysisService) RunScan(ctx context.Context, req ports.ScanRequest) (ports.ScanResult, error) {
	ctx, span := observability.Tracer.Start(ctx, "analysisService.RunScan", trace.WithAttributes())
	defer span.End()

	if err := ctx.Err(); err != nil {
		return ports.ScanResult{}, err
	}
	if s.app == nil {
		return ports.ScanResult{}, fmt.Errorf("app is required")
	}
	if s.app.Config == nil {
		return ports.ScanResult{}, fmt.Errorf("config is required")
	}

	warnings := make([]string, 0)
	filesScanned := 0

	if len(req.Paths) > 0 {
		paths := normalizeScanPaths(req.Paths)
		files, err := s.app.ScanDirectories(paths, s.app.Config.Exclude.Dirs, s.app.Config.Exclude.Files)
		if err != nil {
			return ports.ScanResult{}, errors.AddContext(err, errors.CtxOperation, "scan_directories")
		}
		filesScanned = len(files)
		for i, filePath := range files {
			if err := s.app.ProcessFile(filePath); err != nil {
				warnings = append(warnings, fmt.Sprintf("process file %s: %v", filePath, err))
			}
			if i%100 == 0 {
				if util.GetHeapAllocMB() > uint64(s.app.Config.Performance.MaxHeapMB) {
					s.app.PruneCache(20)
				}
			}
		}
	} else {
		if err := s.app.InitialScan(ctx); err != nil {
			return ports.ScanResult{}, errors.AddContext(err, errors.CtxOperation, "initial_scan")
		}
		filesScanned = s.app.Graph.FileCount()
	}

	return ports.ScanResult{
		FilesScanned: filesScanned,
		Modules:      s.app.Graph.ModuleCount(),
		Warnings:     warnings,
	}, nil
}

func (s *analysisService) TraceImportChain(ctx context.Context, from, to string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.app == nil {
		return "", fmt.Errorf("app is required")
	}
	chain, err := s.app.TraceImportChain(from, to)
	if err != nil {
		err = errors.AddContext(err, "from", from)
		err = errors.AddContext(err, "to", to)
		return "", err
	}
	return chain, nil
}

func (s *analysisService) AnalyzeImpact(ctx context.Context, path string) (graph.ImpactReport, error) {
	if err := ctx.Err(); err != nil {
		return graph.ImpactReport{}, err
	}
	if s.app == nil {
		return graph.ImpactReport{}, fmt.Errorf("app is required")
	}
	report, err := s.app.AnalyzeImpact(ctx, path)
	if err != nil {
		return graph.ImpactReport{}, errors.AddContext(err, errors.CtxPath, path)
	}
	return report, nil
}

func (s *analysisService) DetectCycles(ctx context.Context, limit int) ([][]string, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if s.app == nil {
		return nil, 0, fmt.Errorf("app is required")
	}
	cycles := s.app.Graph.DetectCycles()
	count := len(cycles)
	if limit > 0 && len(cycles) > limit {
		cycles = cycles[:limit]
	}
	out := make([][]string, 0, len(cycles))
	for _, cycle := range cycles {
		out = append(out, append([]string(nil), cycle...))
	}
	return out, count, nil
}

func (s *analysisService) ListFiles(ctx context.Context) ([]*parser.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.app == nil {
		return nil, fmt.Errorf("app is required")
	}
	return append([]*parser.File(nil), s.app.Graph.GetAllFiles()...), nil
}

func (s *analysisService) QueryService(historyStore ports.HistoryStore, projectKey string) ports.QueryService {
	return query.NewService(s.app.Graph, historyStore, strings.TrimSpace(projectKey))
}

func (s *analysisService) CaptureHistoryTrend(ctx context.Context, historyStore ports.HistoryStore, req ports.HistoryTrendRequest) (ports.HistoryTrendResult, error) {
	if err := ctx.Err(); err != nil {
		return ports.HistoryTrendResult{}, err
	}
	if s.app == nil {
		return ports.HistoryTrendResult{}, fmt.Errorf("app is required")
	}
	if historyStore == nil {
		return ports.HistoryTrendResult{}, fmt.Errorf("history store is required")
	}

	projectKey := strings.TrimSpace(req.ProjectKey)
	if projectKey == "" {
		projectKey = "default"
	}

	window := req.Window
	if window <= 0 {
		window = 24 * time.Hour
	}

	projectRoot := strings.TrimSpace(req.ProjectRoot)
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err == nil {
			projectRoot = cwd
		}
	}

	metrics := s.app.Graph.ComputeModuleMetrics()
	cycles := s.app.Graph.DetectCycles()
	unresolved := s.app.AnalyzeHallucinations(ctx)
	unused := s.app.AnalyzeUnusedImports(ctx)
	violations := make([]graph.ArchitectureViolation, 0)
	if s.app.archEngine != nil {
		violations = s.app.ArchitectureViolations()
	}
	hotspotLimit := 0
	if s.app.Config != nil {
		hotspotLimit = s.app.Config.Architecture.TopComplexity
	}
	hotspots := s.app.Graph.TopComplexity(hotspotLimit)
	avgFanIn, avgFanOut, maxFanIn, maxFanOut := summarizeFanMetrics(metrics)
	commitHash, commitTime := history.ResolveGitMetadata(projectRoot)

	snapshot := history.Snapshot{
		Timestamp:         time.Now().UTC(),
		CommitHash:        commitHash,
		CommitTimestamp:   commitTime,
		ModuleCount:       s.app.Graph.ModuleCount(),
		FileCount:         s.app.Graph.FileCount(),
		CycleCount:        len(cycles),
		UnresolvedCount:   len(unresolved),
		UnusedImportCount: len(unused),
		ViolationCount:    len(violations),
		HotspotCount:      len(hotspots),
		AvgFanIn:          avgFanIn,
		AvgFanOut:         avgFanOut,
		MaxFanIn:          maxFanIn,
		MaxFanOut:         maxFanOut,
	}

	if err := historyStore.SaveSnapshot(projectKey, snapshot); err != nil {
		return ports.HistoryTrendResult{}, fmt.Errorf("save history snapshot: %w", err)
	}

	snapshots, err := historyStore.LoadSnapshots(projectKey, req.Since)
	if err != nil {
		return ports.HistoryTrendResult{}, fmt.Errorf("load history snapshots: %w", err)
	}

	result := ports.HistoryTrendResult{
		SnapshotSaved:       true,
		SnapshotsEvaluated:  len(snapshots),
		LatestModuleCount:   snapshot.ModuleCount,
		LatestCycleCount:    snapshot.CycleCount,
		LatestUnresolvedRef: snapshot.UnresolvedCount,
	}
	if len(snapshots) == 0 {
		return result, nil
	}

	report, err := history.BuildTrendReport(projectKey, snapshots, window)
	if err != nil {
		return ports.HistoryTrendResult{}, fmt.Errorf("build trend report: %w", err)
	}
	result.Report = &report
	return result, nil
}

func (s *analysisService) WatchService() ports.WatchService {
	return &watchService{app: s.app}
}

func (s *analysisService) SummarySnapshot(ctx context.Context) (ports.SummarySnapshot, error) {
	fmt.Println("DEBUG: SummarySnapshot started")
	if err := ctx.Err(); err != nil {
		return ports.SummarySnapshot{}, err
	}
	if s.app == nil {
		return ports.SummarySnapshot{}, fmt.Errorf("app is required")
	}

	cycles := s.app.Graph.DetectCycles()
	outCycles := make([][]string, 0, len(cycles))
	for _, cycle := range cycles {
		outCycles = append(outCycles, append([]string(nil), cycle...))
	}

	metrics := s.app.Graph.ComputeModuleMetrics()
	outMetrics := make(map[string]graph.ModuleMetrics, len(metrics))
	for module, metric := range metrics {
		outMetrics[module] = metric
	}

	violations := make([]graph.ArchitectureViolation, 0)
	if s.app.archEngine != nil {
		violations = s.app.ArchitectureViolations()
	}
	hotspotLimit := 0
	if s.app.Config != nil {
		hotspotLimit = s.app.Config.Architecture.TopComplexity
	}
	hotspots := s.app.Graph.TopComplexity(hotspotLimit)
	hallucinations := s.app.AnalyzeHallucinations(ctx)
	unusedImports := s.app.AnalyzeUnusedImports(ctx)
	ruleViolations, ruleSummary := s.app.ArchitectureRuleViolations()

	return ports.SummarySnapshot{
		FileCount:      s.app.Graph.FileCount(),
		ModuleCount:    s.app.Graph.ModuleCount(),
		SecretCount:    s.app.SecretCount(),
		Cycles:         outCycles,
		Hallucinations: append([]resolver.UnresolvedReference(nil), hallucinations...),
		UnusedImports:  append([]resolver.UnusedImport(nil), unusedImports...),
		Metrics:        outMetrics,
		Violations:     append([]graph.ArchitectureViolation(nil), violations...),
		RuleViolations: append([]ports.ArchitectureRuleViolation(nil), ruleViolations...),
		RuleSummary:    ruleSummary,
		Hotspots:       append([]graph.ComplexityHotspot(nil), hotspots...),
	}, nil
}

func (s *analysisService) PrintSummary(ctx context.Context, req ports.SummaryPrintRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.app == nil {
		return fmt.Errorf("app is required")
	}
	newPresentationService(s.app).PrintSummary(
		req.Snapshot.FileCount,
		req.Snapshot.ModuleCount,
		req.Duration,
		req.Snapshot.Cycles,
		req.Snapshot.Hallucinations,
		req.Snapshot.UnusedImports,
		req.Snapshot.Metrics,
		req.Snapshot.Violations,
		req.Snapshot.RuleViolations,
		req.Snapshot.RuleSummary,
		req.Snapshot.Hotspots,
	)
	return nil
}

func (s *analysisService) SyncOutputs(ctx context.Context, req ports.SyncOutputsRequest) (ports.SyncOutputsResult, error) {
	snapshot, err := s.SummarySnapshot(ctx)
	if err != nil {
		return ports.SyncOutputsResult{}, err
	}
	return s.SyncOutputsWithSnapshot(ctx, req, snapshot)
}

func (s *analysisService) SyncOutputsWithSnapshot(ctx context.Context, req ports.SyncOutputsRequest, snapshot ports.SummarySnapshot) (ports.SyncOutputsResult, error) {
	if err := ctx.Err(); err != nil {
		return ports.SyncOutputsResult{}, err
	}
	if s.app == nil {
		return ports.SyncOutputsResult{}, fmt.Errorf("app is required")
	}
	if s.app.Config == nil {
		return ports.SyncOutputsResult{}, fmt.Errorf("config is required")
	}

	originalOutput := s.app.Config.Output
	filteredOutput := originalOutput
	formatSet := formatSet(req.Formats)
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

	cfgCopy := *s.app.Config
	cfgCopy.Output = filteredOutput
	s.app.Config.Output = cfgCopy.Output
	writeTargets, err := s.app.resolveOutputTargets()
	s.app.Config.Output = originalOutput
	if err != nil {
		return ports.SyncOutputsResult{}, err
	}

	s.app.Config.Output = filteredOutput
	err = s.app.GenerateOutputs(
		ctx,
		snapshot.Cycles,
		snapshot.Hallucinations,
		snapshot.UnusedImports,
		snapshot.Metrics,
		snapshot.Violations,
		snapshot.RuleViolations,
		snapshot.RuleSummary,
		snapshot.Hotspots,
		nil, // AnalyzeProbableBridges will still be called once inside GenerateOutputs for Markdown if needed, but we can't easily avoid it without bloating snapshot.
	)
	s.app.Config.Output = originalOutput
	if err != nil {
		return ports.SyncOutputsResult{}, err
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

	return ports.SyncOutputsResult{Written: uniqueStrings(written)}, nil
}

func (s *analysisService) GenerateMarkdownReport(ctx context.Context, req ports.MarkdownReportRequest) (ports.MarkdownReportResult, error) {
	if err := ctx.Err(); err != nil {
		return ports.MarkdownReportResult{}, err
	}
	if s.app == nil {
		return ports.MarkdownReportResult{}, fmt.Errorf("app is required")
	}

	result, err := s.app.GenerateMarkdownReport(ctx, MarkdownReportRequest{
		OutputPath: req.Path,
		WriteFile:  req.WriteFile || strings.TrimSpace(req.Path) != "",
		Verbosity:  req.Verbosity,
	})
	if err != nil {
		return ports.MarkdownReportResult{}, err
	}

	return ports.MarkdownReportResult{
		Markdown:  result.Markdown,
		Path:      result.Path,
		Written:   result.Written,
		RuleGuide: result.RuleGuide,
	}, nil
}

func (s *analysisService) UpdateConfig(ctx context.Context, cfg *config.Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.app == nil {
		return fmt.Errorf("app is required")
	}
	return s.app.UpdateConfig(ctx, cfg)
}

type watchService struct {
	app *App
}

var _ ports.WatchService = (*watchService)(nil)

func (s *watchService) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.app == nil {
		return fmt.Errorf("app is required")
	}
	return s.app.StartWatcher()
}

func (s *watchService) CurrentUpdate(ctx context.Context) (ports.WatchUpdate, error) {
	if err := ctx.Err(); err != nil {
		return ports.WatchUpdate{}, err
	}
	if s.app == nil {
		return ports.WatchUpdate{}, fmt.Errorf("app is required")
	}
	return toWatchUpdate(s.app.CurrentUpdate(ctx)), nil
}

func (s *watchService) Subscribe(ctx context.Context, handler func(ports.WatchUpdate)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.app == nil {
		return fmt.Errorf("app is required")
	}
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	s.app.SetUpdateHandler(func(update Update) {
		if ctx.Err() != nil {
			return
		}
		handler(toWatchUpdate(update))
	})
	return nil
}

func toWatchUpdate(update Update) ports.WatchUpdate {
	return ports.WatchUpdate{
		Cycles:         append([][]string(nil), update.Cycles...),
		Hallucinations: append([]resolver.UnresolvedReference(nil), update.Hallucinations...),
		ModuleCount:    update.ModuleCount,
		FileCount:      update.FileCount,
		SecretCount:    update.SecretCount,
	}
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

func summarizeFanMetrics(metrics map[string]graph.ModuleMetrics) (avgIn, avgOut float64, maxIn, maxOut int) {
	if len(metrics) == 0 {
		return 0, 0, 0, 0
	}
	var totalIn, totalOut int
	for _, m := range metrics {
		totalIn += m.FanIn
		totalOut += m.FanOut
		if m.FanIn > maxIn {
			maxIn = m.FanIn
		}
		if m.FanOut > maxOut {
			maxOut = m.FanOut
		}
	}
	n := float64(len(metrics))
	return float64(totalIn) / n, float64(totalOut) / n, maxIn, maxOut
}

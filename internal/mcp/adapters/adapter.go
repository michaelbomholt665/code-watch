package adapters

import (
	"circular/internal/core/ports"
	"circular/internal/data/history"
	"circular/internal/engine/parser"
	"circular/internal/engine/secrets"
	"circular/internal/mcp/contracts"
	"circular/internal/shared/util"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Adapter struct {
	mu         sync.RWMutex
	analysis   ports.AnalysisService
	history    *history.Store
	projectKey string
}

func NewAdapter(analysis ports.AnalysisService, historyStore *history.Store, projectKey string) *Adapter {
	return &Adapter{
		analysis:   analysis,
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
	if a.analysis == nil {
		return contracts.ScanRunOutput{}, fmt.Errorf("analysis service unavailable")
	}

	start := time.Now()
	result, err := a.analysis.RunScan(ctx, ports.ScanRequest{Paths: in.Paths})
	if err != nil {
		return contracts.ScanRunOutput{}, err
	}
	warnings := append([]string(nil), result.Warnings...)
	if a.history != nil {
		if _, err := a.analysis.CaptureHistoryTrend(ctx, a.history, ports.HistoryTrendRequest{
			ProjectKey: a.projectKey,
			Window:     24 * time.Hour,
		}); err != nil {
			warnings = append(warnings, fmt.Sprintf("capture history trend: %v", err))
		}
	}

	return contracts.ScanRunOutput{
		FilesScanned: result.FilesScanned,
		Modules:      result.Modules,
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
	if a.analysis == nil {
		return contracts.SecretsScanOutput{}, fmt.Errorf("analysis service unavailable")
	}

	scanResult, err := a.analysis.RunScan(ctx, ports.ScanRequest{Paths: in.Paths})
	if err != nil {
		return contracts.SecretsScanOutput{}, err
	}

	files, err := a.analysis.ListFiles(ctx)
	if err != nil {
		return contracts.SecretsScanOutput{}, err
	}
	findings, total := secretFindings(files, 0)
	return contracts.SecretsScanOutput{
		FilesScanned: scanResult.FilesScanned,
		SecretCount:  total,
		Findings:     findings,
		Warnings:     scanResult.Warnings,
	}, nil
}

func (a *Adapter) ListSecrets(ctx context.Context, limit int) (contracts.SecretsListOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.SecretsListOutput{}, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.analysis == nil {
		return contracts.SecretsListOutput{}, fmt.Errorf("analysis service unavailable")
	}

	files, err := a.analysis.ListFiles(ctx)
	if err != nil {
		return contracts.SecretsListOutput{}, err
	}
	findings, total := secretFindings(files, limit)
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
	if a.analysis == nil {
		return contracts.GraphCyclesOutput{}, fmt.Errorf("analysis service unavailable")
	}

	cycles, count, err := a.analysis.DetectCycles(ctx, limit)
	if err != nil {
		return contracts.GraphCyclesOutput{}, err
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
	svc := a.queryService()
	if svc == nil {
		return contracts.QueryModulesOutput{}, fmt.Errorf("analysis service unavailable")
	}

	rows, err := svc.ListModules(ctx, filter, limit)
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
	svc := a.queryService()
	if svc == nil {
		return contracts.QueryModuleDetailsOutput{}, fmt.Errorf("analysis service unavailable")
	}

	details, err := svc.ModuleDetails(ctx, module)
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
	svc := a.queryService()
	if svc == nil {
		return contracts.QueryTraceOutput{}, fmt.Errorf("analysis service unavailable")
	}

	trace, err := svc.DependencyTrace(ctx, from, to, maxDepth)
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
	svc := a.queryService()
	if svc == nil {
		return contracts.QueryTrendsOutput{}, fmt.Errorf("analysis service unavailable")
	}

	slice, err := svc.TrendSlice(ctx, since, limit)
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
	if a.analysis == nil {
		return nil, fmt.Errorf("analysis service unavailable")
	}
	result, err := a.analysis.SyncOutputs(ctx, ports.SyncOutputsRequest{
		Formats: append([]string(nil), formats...),
	})
	if err != nil {
		return nil, err
	}
	return append([]string(nil), result.Written...), nil
}

func (a *Adapter) GenerateMarkdownReport(ctx context.Context, in contracts.ReportGenerateMarkdownInput) (contracts.ReportGenerateMarkdownOutput, error) {
	if err := ctx.Err(); err != nil {
		return contracts.ReportGenerateMarkdownOutput{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.analysis == nil {
		return contracts.ReportGenerateMarkdownOutput{}, fmt.Errorf("analysis service unavailable")
	}
	result, err := a.analysis.GenerateMarkdownReport(ctx, ports.MarkdownReportRequest{
		Path:      in.Path,
		WriteFile: in.WriteFile,
		Verbosity: in.Verbosity,
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

func (a *Adapter) queryService() ports.QueryService {
	if a.analysis == nil {
		return nil
	}
	return a.analysis.QueryService(a.history, a.ProjectKey())
}

func resolveDiagramPath(path, root, diagramsDir string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if util.ContainsPathSeparator(path) {
		return filepath.Join(root, path)
	}
	return filepath.Join(diagramsDir, path)
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

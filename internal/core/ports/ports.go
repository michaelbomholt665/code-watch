package ports

import (
	"circular/internal/core/config"
	"circular/internal/data/history"
	"circular/internal/data/query"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"context"
	"time"
)

// CodeParser abstracts source parsing and language-file support checks.
type CodeParser interface {
	ParseFile(path string, content []byte) (*parser.File, error)
	GetLanguage(path string) string
	IsSupportedPath(filePath string) bool
	IsTestFile(path string) bool
	SupportedExtensions() []string
	SupportedFilenames() []string
	SupportedTestFileSuffixes() []string
}

// SecretScanner abstracts secret detection during file ingestion.
type SecretScanner interface {
	Detect(filePath string, content []byte) []parser.Secret
}

// LineRange represents a 1-based inclusive line span used for incremental scans.
type LineRange struct {
	Start int
	End   int
}

// IncrementalSecretScanner is an optional extension for hunk/line-range secret scanning.
type IncrementalSecretScanner interface {
	SecretScanner
	DetectInRanges(filePath string, content []byte, ranges []LineRange) []parser.Secret
}

// HistoryStore abstracts snapshot persistence for trend/report workflows.
type HistoryStore interface {
	SaveSnapshot(projectKey string, snapshot history.Snapshot) error
	LoadSnapshots(projectKey string, since time.Time) ([]history.Snapshot, error)
}

type WriteOperation string

const (
	WriteOperationUpsertFile   WriteOperation = "upsert_file"
	WriteOperationDeleteFile   WriteOperation = "delete_file"
	WriteOperationPruneToPaths WriteOperation = "prune_to_paths"
)

type WriteRequest struct {
	ProjectKey string
	Operation  WriteOperation
	File       *parser.File
	FilePath   string
	Paths      []string
}

type EnqueueResult string

const (
	EnqueueAccepted EnqueueResult = "accepted"
	EnqueueDropped  EnqueueResult = "dropped"
	EnqueueSpooled  EnqueueResult = "spooled"
)

type WriteQueuePort interface {
	Enqueue(req WriteRequest) EnqueueResult
	DequeueBatch(ctx context.Context, maxItems int, wait time.Duration) ([]WriteRequest, error)
	Close() error
}

type SpoolRow struct {
	ID       int64
	Request  WriteRequest
	Attempts int
}

type WriteSpoolPort interface {
	Enqueue(req WriteRequest) error
	DequeueBatch(ctx context.Context, maxItems int) ([]SpoolRow, error)
	Ack(ids []int64) error
	Nack(rows []SpoolRow, nextAttemptAt time.Time, lastErr string) error
	PendingCount(ctx context.Context) (int, error)
	Close() error
}

// ScanRequest defines a scan operation request for driving adapters.
type ScanRequest struct {
	Paths []string
}

// ScanResult summarizes a completed scan operation.
type ScanResult struct {
	FilesScanned int
	Modules      int
	Warnings     []string
}

// SyncOutputsRequest defines output synchronization input for driving adapters.
type SyncOutputsRequest struct {
	Formats []string
}

// SyncOutputsResult contains generated output paths.
type SyncOutputsResult struct {
	Written []string
}

// MarkdownReportRequest defines markdown report generation input.
type MarkdownReportRequest struct {
	Path      string
	WriteFile bool
	Verbosity string
}

// MarkdownReportResult contains markdown report generation results.
type MarkdownReportResult struct {
	Markdown  string
	Path      string
	Written   bool
	RuleGuide ArchitectureRuleGuide
}

// SummarySnapshot captures current graph/resolution state for driving adapters.
type SummarySnapshot struct {
	FileCount      int
	ModuleCount    int
	SecretCount    int
	Cycles         [][]string
	Hallucinations []resolver.UnresolvedReference
	UnusedImports  []resolver.UnusedImport
	Metrics        map[string]graph.ModuleMetrics
	Violations     []graph.ArchitectureViolation
	RuleViolations []ArchitectureRuleViolation
	RuleSummary    ArchitectureRuleSummary
	Hotspots       []graph.ComplexityHotspot
}

type ArchitectureRuleSummary struct {
	RuleCount        int
	ModuleCount      int
	ViolationCount   int
	ImportViolations int
	FileViolations   int
}

type ArchitectureRuleKind string

const (
	ArchitectureRuleKindLayer   ArchitectureRuleKind = "layer"
	ArchitectureRuleKindPackage ArchitectureRuleKind = "package"
)

type ArchitectureImportRule struct {
	Allow []string
	Deny  []string
}

type ArchitectureRuleExclude struct {
	Tests bool
	Files []string
}

type ArchitectureRule struct {
	Name     string
	Kind     ArchitectureRuleKind
	Modules  []string
	MaxFiles int
	Imports  ArchitectureImportRule
	Exclude  ArchitectureRuleExclude
}

type ArchitectureRuleViolation struct {
	RuleName string
	RuleKind ArchitectureRuleKind
	Module   string
	Target   string
	Type     string
	Message  string
	File     string
	Line     int
	Column   int
	Limit    int
	Actual   int
}

type ArchitectureRuleGuide struct {
	Summary    ArchitectureRuleSummary
	Rules      []ArchitectureRule
	Violations []ArchitectureRuleViolation
}

// SummaryPrintRequest captures terminal-summary rendering inputs.
type SummaryPrintRequest struct {
	Duration time.Duration
	Snapshot SummarySnapshot
}

// QueryService exposes read-only dependency query operations for driving adapters.
type QueryService interface {
	ListModules(ctx context.Context, filter string, limit int) ([]query.ModuleSummary, error)
	ModuleDetails(ctx context.Context, moduleName string) (query.ModuleDetails, error)
	DependencyTrace(ctx context.Context, from, to string, maxDepth int) (query.TraceResult, error)
	TrendSlice(ctx context.Context, since time.Time, limit int) (query.TrendSlice, error)
}

// WatchUpdate contains state emitted to driving adapters during watch-mode updates.
type WatchUpdate struct {
	Cycles         [][]string
	Hallucinations []resolver.UnresolvedReference
	ModuleCount    int
	FileCount      int
	SecretCount    int
}

// WatchService exposes watch lifecycle and updates for driving adapters.
type WatchService interface {
	Start(ctx context.Context) error
	CurrentUpdate(ctx context.Context) (WatchUpdate, error)
	Subscribe(ctx context.Context, handler func(WatchUpdate)) error
}

// Reloadable is an interface for components that can be updated with a new configuration.
type Reloadable interface {
	Reload(cfg *config.Config) error
}

// AnalysisService defines the first driving-port surface over scan/query use cases.
type AnalysisService interface {
	RunScan(ctx context.Context, req ScanRequest) (ScanResult, error)
	TraceImportChain(ctx context.Context, from, to string) (string, error)
	AnalyzeImpact(ctx context.Context, path string) (graph.ImpactReport, error)
	DetectCycles(ctx context.Context, limit int) ([][]string, int, error)
	ListFiles(ctx context.Context) ([]*parser.File, error)
	QueryService(historyStore HistoryStore, projectKey string) QueryService
	CaptureHistoryTrend(ctx context.Context, historyStore HistoryStore, req HistoryTrendRequest) (HistoryTrendResult, error)
	WatchService() WatchService
	SummarySnapshot(ctx context.Context) (SummarySnapshot, error)
	PrintSummary(ctx context.Context, req SummaryPrintRequest) error
	SyncOutputs(ctx context.Context, req SyncOutputsRequest) (SyncOutputsResult, error)
	SyncOutputsWithSnapshot(ctx context.Context, req SyncOutputsRequest, snapshot SummarySnapshot) (SyncOutputsResult, error)
	GenerateMarkdownReport(ctx context.Context, req MarkdownReportRequest) (MarkdownReportResult, error)
	UpdateConfig(ctx context.Context, cfg *config.Config) error
}

// HistoryTrendRequest captures inputs needed to save a snapshot and compute trends.
type HistoryTrendRequest struct {
	ProjectKey  string
	ProjectRoot string
	Since       time.Time
	Window      time.Duration
}

// HistoryTrendResult contains the optional trend report and saved snapshot metadata.
type HistoryTrendResult struct {
	Report              *history.TrendReport
	SnapshotSaved       bool
	SnapshotsEvaluated  int
	LatestModuleCount   int
	LatestCycleCount    int
	LatestUnresolvedRef int
}

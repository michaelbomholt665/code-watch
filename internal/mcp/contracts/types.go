package contracts

import "encoding/json"

const (
	ToolNameCircular = "circular"
	ContractVersion  = "v1"
)

type OperationID string

const (
	OperationScanRun         OperationID = "scan.run"
	OperationSecretsScan     OperationID = "secrets.scan"
	OperationSecretsList     OperationID = "secrets.list"
	OperationGraphCycles     OperationID = "graph.cycles"
	OperationGraphSyncDiag   OperationID = "graph.sync_diagrams"
	OperationQueryModules    OperationID = "query.modules"
	OperationQueryDetails    OperationID = "query.module_details"
	OperationQueryTrace      OperationID = "query.trace"
	OperationSystemSyncOut   OperationID = "system.sync_outputs"
	OperationSystemSyncCfg   OperationID = "system.sync_config"
	OperationSystemGenCfg    OperationID = "system.generate_config"
	OperationSystemGenScript OperationID = "system.generate_script"
	OperationSystemSelect    OperationID = "system.select_project"
	OperationSystemWatch     OperationID = "system.watch"
	OperationQueryTrends     OperationID = "query.trends"
	OperationReportGenMD     OperationID = "report.generate_markdown"
)

type CircularToolInput struct {
	Operation OperationID     `json:"operation"`
	Params    json.RawMessage `json:"params,omitempty"`
}

type OperationDescriptor struct {
	ID          OperationID    `json:"id"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}

type ScanRunInput struct {
	Paths       []string `json:"paths,omitempty"`
	ConfigPath  string   `json:"config_path,omitempty"`
	ProjectRoot string   `json:"project_root,omitempty"`
}

type ScanRunOutput struct {
	FilesScanned int      `json:"files_scanned"`
	Modules      int      `json:"modules"`
	DurationMs   int      `json:"duration_ms"`
	Warnings     []string `json:"warnings,omitempty"`
}

type SecretFinding struct {
	Kind        string  `json:"kind"`
	Severity    string  `json:"severity"`
	ValueMasked string  `json:"value_masked"`
	Entropy     float64 `json:"entropy"`
	Confidence  float64 `json:"confidence"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Column      int     `json:"column"`
}

type SecretsScanInput struct {
	Paths []string `json:"paths,omitempty"`
}

type SecretsScanOutput struct {
	FilesScanned int             `json:"files_scanned"`
	SecretCount  int             `json:"secret_count"`
	Findings     []SecretFinding `json:"findings,omitempty"`
	Warnings     []string        `json:"warnings,omitempty"`
}

type SecretsListInput struct {
	Limit int `json:"limit,omitempty"`
}

type SecretsListOutput struct {
	SecretCount int             `json:"secret_count"`
	Findings    []SecretFinding `json:"findings,omitempty"`
}

type GraphCyclesInput struct {
	Limit int `json:"limit,omitempty"`
}

type GraphCyclesOutput struct {
	CycleCount int        `json:"cycle_count"`
	Cycles     [][]string `json:"cycles,omitempty"`
}

type QueryModulesInput struct {
	Filter string `json:"filter,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type ModuleSummary struct {
	Name                   string `json:"name"`
	FileCount              int    `json:"file_count"`
	ExportCount            int    `json:"export_count"`
	DependencyCount        int    `json:"dependency_count"`
	ReverseDependencyCount int    `json:"reverse_dependency_count"`
}

type QueryModulesOutput struct {
	Modules []ModuleSummary `json:"modules"`
}

type QueryModuleDetailsInput struct {
	Module string `json:"module"`
}

type DependencyEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type ModuleDetails struct {
	Name                string           `json:"name"`
	Files               []string         `json:"files"`
	ExportedSymbols     []string         `json:"exported_symbols"`
	Dependencies        []DependencyEdge `json:"dependencies"`
	ReverseDependencies []string         `json:"reverse_dependencies"`
}

type QueryModuleDetailsOutput struct {
	Module ModuleDetails `json:"module"`
}

type QueryTraceInput struct {
	From     string `json:"from_module"`
	To       string `json:"to_module"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

type QueryTraceOutput struct {
	Found bool     `json:"found"`
	Path  []string `json:"path,omitempty"`
	Depth int      `json:"depth,omitempty"`
}

type SystemSyncOutputsInput struct {
	Formats []string `json:"formats,omitempty"`
}

type SystemSyncOutputsOutput struct {
	Written []string `json:"written"`
}

type SystemSyncConfigInput struct{}

type SystemSyncConfigOutput struct {
	Synced bool   `json:"synced"`
	Target string `json:"target,omitempty"`
}

type SystemGenerateConfigInput struct{}

type SystemGenerateConfigOutput struct {
	Generated bool   `json:"generated"`
	Target    string `json:"target,omitempty"`
}

type SystemGenerateScriptInput struct{}

type SystemGenerateScriptOutput struct {
	Generated bool   `json:"generated"`
	Target    string `json:"target,omitempty"`
}

type SystemSelectProjectInput struct {
	Name string `json:"name"`
}

type ProjectSummary struct {
	Name        string `json:"name"`
	Root        string `json:"root"`
	DBNamespace string `json:"db_namespace"`
	Key         string `json:"key"`
}

type SystemSelectProjectOutput struct {
	Project ProjectSummary `json:"project"`
}

type SystemWatchInput struct{}

type SystemWatchOutput struct {
	Status          string `json:"status"`
	AlreadyWatching bool   `json:"already_watching,omitempty"`
}

type QueryTrendsInput struct {
	Since string `json:"since,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type TrendSnapshot struct {
	Timestamp string `json:"timestamp"`
	Modules   int    `json:"modules"`
	Files     int    `json:"files"`
}

type QueryTrendsOutput struct {
	Since     string          `json:"since,omitempty"`
	Until     string          `json:"until,omitempty"`
	ScanCount int             `json:"scan_count"`
	Snapshots []TrendSnapshot `json:"snapshots"`
}

type ReportGenerateMarkdownInput struct {
	WriteFile bool   `json:"write_file,omitempty"`
	Path      string `json:"path,omitempty"`
	Verbosity string `json:"verbosity,omitempty"`
}

type ReportGenerateMarkdownOutput struct {
	Markdown string `json:"markdown"`
	Path     string `json:"path,omitempty"`
	Written  bool   `json:"written"`
}

type ToolError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e ToolError) Error() string {
	return e.Message
}

const (
	ErrorInvalidArgument = "invalid_argument"
	ErrorNotFound        = "not_found"
	ErrorInternal        = "internal"
	ErrorUnavailable     = "unavailable"
)

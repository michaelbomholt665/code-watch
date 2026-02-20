package config

import (
	"time"
)

type Config struct {
	Version             int                 `toml:"version"`
	Paths               Paths               `toml:"paths"`
	ConfigFiles         ConfigFiles         `toml:"config"`
	DB                  Database            `toml:"db"`
	Projects            Projects            `toml:"projects"`
	MCP                 MCP                 `toml:"mcp"`
	GrammarsPath        string              `toml:"grammars_path"`
	GrammarVerification GrammarVerification `toml:"grammar_verification"`
	Languages           map[string]Language `toml:"languages"`
	DynamicGrammars     []DynamicGrammar    `toml:"dynamic_grammars"`
	WatchPaths          []string            `toml:"watch_paths"`
	Exclude             Exclude             `toml:"exclude"`
	Watch               Watch               `toml:"watch"`
	Output              Output              `toml:"output"`
	Alerts              Alerts              `toml:"alerts"`
	Architecture        Architecture        `toml:"architecture"`
	Secrets             Secrets             `toml:"secrets"`
	Resolver            ResolverSettings    `toml:"resolver"`
}

type DynamicGrammar struct {
	Name            string   `toml:"name"`
	Library         string   `toml:"library"`
	Extensions      []string `toml:"extensions"`
	Filenames       []string `toml:"filenames"`
	NamespaceNode   string   `toml:"namespace_node"`
	ImportNode      string   `toml:"import_node"`
	DefinitionNodes []string `toml:"definition_nodes"`
}

type Paths struct {
	ProjectRoot string `toml:"project_root"`
	ConfigDir   string `toml:"config_dir"`
	StateDir    string `toml:"state_dir"`
	CacheDir    string `toml:"cache_dir"`
	DatabaseDir string `toml:"database_dir"`
}

type ConfigFiles struct {
	ActiveFile string   `toml:"active_file"`
	Includes   []string `toml:"includes"`
}

type Database struct {
	Enabled     bool          `toml:"enabled"`
	Driver      string        `toml:"driver"`
	Path        string        `toml:"path"`
	BusyTimeout time.Duration `toml:"busy_timeout"`
	ProjectMode string        `toml:"project_mode"`
}

type Projects struct {
	Active       string         `toml:"active"`
	RegistryFile string         `toml:"registry_file"`
	Entries      []ProjectEntry `toml:"entries"`
}

type ProjectEntry struct {
	Name        string `toml:"name"`
	Root        string `toml:"root"`
	DBNamespace string `toml:"db_namespace"`
	ConfigFile  string `toml:"config_file"`
}

type MCP struct {
	Enabled            bool          `toml:"enabled"`
	Mode               string        `toml:"mode"`
	Transport          string        `toml:"transport"`
	Address            string        `toml:"address"`
	ConfigPath         string        `toml:"config_path"`
	OpenAPISpecPath    string        `toml:"openapi_spec_path"`
	OpenAPISpecURL     string        `toml:"openapi_spec_url"`
	ServerName         string        `toml:"server_name"`
	ServerVersion      string        `toml:"server_version"`
	ExposedToolName    string        `toml:"exposed_tool_name"`
	OperationAllowlist []string      `toml:"operation_allowlist"`
	MaxResponseItems   int           `toml:"max_response_items"`
	RequestTimeout     time.Duration `toml:"request_timeout"`
	AllowMutations     bool          `toml:"allow_mutations"`
	AutoManageOutputs  *bool         `toml:"auto_manage_outputs"`
	AutoSyncConfig     *bool         `toml:"auto_sync_config"`
}

type GrammarVerification struct {
	Enabled *bool `toml:"enabled"`
}

type Language struct {
	Enabled    *bool    `toml:"enabled"`
	Extensions []string `toml:"extensions"`
	Filenames  []string `toml:"filenames"`
}

type Exclude struct {
	Dirs    []string `toml:"dirs"`
	Files   []string `toml:"files"`
	Symbols []string `toml:"symbols"` // Prefixes to ignore (e.g., self., ctx.)
	Imports []string `toml:"imports"` // Import paths to ignore for unused check
}

type Watch struct {
	Debounce time.Duration `toml:"debounce"`
}

type Output struct {
	DOT            string              `toml:"dot"`
	TSV            string              `toml:"tsv"`
	Mermaid        string              `toml:"mermaid"`
	PlantUML       string              `toml:"plantuml"`
	Markdown       string              `toml:"markdown"`
	Formats        DiagramFormats      `toml:"formats"`
	Diagrams       DiagramOutput       `toml:"diagrams"`
	Report         ReportOutput        `toml:"report"`
	UpdateMarkdown []MarkdownInjection `toml:"update_markdown"`
	Paths          OutputPaths         `toml:"paths"`
}

type ReportOutput struct {
	Verbosity           string `toml:"verbosity"`
	TableOfContents     *bool  `toml:"table_of_contents"`
	CollapsibleSections *bool  `toml:"collapsible_sections"`
	IncludeMermaid      *bool  `toml:"include_mermaid"`
}

type DiagramFormats struct {
	Mermaid  *bool `toml:"mermaid"`
	PlantUML *bool `toml:"plantuml"`
}

type DiagramOutput struct {
	Architecture bool                   `toml:"architecture"`
	Component    bool                   `toml:"component"`
	Flow         bool                   `toml:"flow"`
	FlowConfig   FlowDiagramConfig      `toml:"flow_config"`
	ComponentCfg ComponentDiagramConfig `toml:"component_config"`
}

type FlowDiagramConfig struct {
	EntryPoints []string `toml:"entry_points"`
	MaxDepth    int      `toml:"max_depth"`
}

type ComponentDiagramConfig struct {
	ShowInternal bool `toml:"show_internal"`
}

type MarkdownInjection struct {
	File   string `toml:"file"`
	Marker string `toml:"marker"`
	Format string `toml:"format"`
}

type OutputPaths struct {
	Root        string `toml:"root"`
	DiagramsDir string `toml:"diagrams_dir"`
}

type Alerts struct {
	Beep     bool `toml:"beep"`
	Terminal bool `toml:"terminal"`
}

type Architecture struct {
	Enabled       bool                `toml:"enabled"`
	TopComplexity int                 `toml:"top_complexity"`
	Layers        []ArchitectureLayer `toml:"layers"`
	Rules         []ArchitectureRule  `toml:"rules"`
}

type ArchitectureLayer struct {
	Name  string   `toml:"name"`
	Paths []string `toml:"paths"`
}

type ArchitectureRule struct {
	Name  string   `toml:"name"`
	From  string   `toml:"from"`
	Allow []string `toml:"allow"`
}

type Secrets struct {
	Enabled          bool                 `toml:"enabled"`
	EntropyThreshold float64              `toml:"entropy_threshold"`
	MinTokenLength   int                  `toml:"min_token_length"`
	Patterns         []SecretPattern      `toml:"patterns"`
	Exclude          SecretExcludePattern `toml:"exclude"`
}

type SecretPattern struct {
	Name     string `toml:"name"`
	Regex    string `toml:"regex"`
	Severity string `toml:"severity"`
}

type SecretExcludePattern struct {
	Dirs  []string `toml:"dirs"`
	Files []string `toml:"files"`
}

type ResolverSettings struct {
	BridgeScoring ResolverBridgeScoring `toml:"bridge_scoring"`
}

type ResolverBridgeScoring struct {
	ConfirmedThreshold int `toml:"confirmed_threshold"`
	ProbableThreshold  int `toml:"probable_threshold"`

	WeightExplicitRuleMatch       int `toml:"weight_explicit_rule_match"`
	WeightBridgeContext           int `toml:"weight_bridge_context"`
	WeightBridgeImportEvidence    int `toml:"weight_bridge_import_evidence"`
	WeightUniqueCrossLangMatch    int `toml:"weight_unique_cross_language_match"`
	WeightAmbiguousCrossLangMatch int `toml:"weight_ambiguous_cross_language_match"`
	WeightLocalOrModuleConflict   int `toml:"weight_local_or_module_conflict"`
	WeightStdlibConflict          int `toml:"weight_stdlib_conflict"`
}

func (g GrammarVerification) IsEnabled() bool {
	if g.Enabled == nil {
		return true
	}
	return *g.Enabled
}

func (m MCP) AutoManageOutputsEnabled() bool {
	if m.AutoManageOutputs == nil {
		return true
	}
	return *m.AutoManageOutputs
}

func (m MCP) AutoSyncConfigEnabled() bool {
	if m.AutoSyncConfig == nil {
		return true
	}
	return *m.AutoSyncConfig
}

func (o Output) MermaidEnabled() bool {
	if o.Formats.Mermaid != nil {
		return *o.Formats.Mermaid
	}
	return true
}

func (o Output) PlantUMLEnabled() bool {
	if o.Formats.PlantUML != nil {
		return *o.Formats.PlantUML
	}
	return false
}

func (r ReportOutput) TableOfContentsEnabled() bool {
	if r.TableOfContents == nil {
		return true
	}
	return *r.TableOfContents
}

func (r ReportOutput) CollapsibleSectionsEnabled() bool {
	if r.CollapsibleSections == nil {
		return true
	}
	return *r.CollapsibleSections
}

func (r ReportOutput) IncludeMermaidEnabled() bool {
	if r.IncludeMermaid == nil {
		return false
	}
	return *r.IncludeMermaid
}

// validateDynamicGrammars moved to validator.go

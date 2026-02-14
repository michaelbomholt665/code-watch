// # internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	content := `
grammars_path = "./grammars"
watch_paths = ["./src"]

[exclude]
dirs = [".git"]
files = ["*.log"]

[watch]
debounce = "1s"

[output]
dot = "graph.dot"
tsv = "deps.tsv"
mermaid = "graph.mmd"
plantuml = "graph.puml"
markdown = "analysis-report.md"

[output.report]
verbosity = "detailed"
table_of_contents = true
collapsible_sections = true
include_mermaid = true

[output.diagrams]
architecture = true
component = true
flow = false

[output.diagrams.flow_config]
entry_points = ["cmd/circular/main.go"]
max_depth = 10

[output.diagrams.component_config]
show_internal = true

[output.paths]
root = "."
diagrams_dir = "docs/diagrams"

[[output.update_markdown]]
file = "README.md"
marker = "deps-mermaid"
format = "mermaid"

[alerts]
beep = true
terminal = true

[secrets]
enabled = true
entropy_threshold = 4.2
min_token_length = 18

[[secrets.patterns]]
name = "custom-test-token"
regex = "CTK_[A-Z0-9]{12}"
severity = "high"

[secrets.exclude]
dirs = [".secrets"]
files = ["*.pem"]
`
	tmpfile, err := os.CreateTemp("", "config*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.GrammarsPath != "./grammars" {
		t.Errorf("Expected GrammarsPath ./grammars, got %s", cfg.GrammarsPath)
	}
	if len(cfg.WatchPaths) != 1 || cfg.WatchPaths[0] != "./src" {
		t.Errorf("Unexpected WatchPaths: %v", cfg.WatchPaths)
	}
	if cfg.Watch.Debounce != time.Second {
		t.Errorf("Expected debounce 1s, got %v", cfg.Watch.Debounce)
	}
	if cfg.Output.DOT != "graph.dot" {
		t.Errorf("Expected DOT graph.dot, got %s", cfg.Output.DOT)
	}
	if cfg.Output.Mermaid != "graph.mmd" {
		t.Errorf("Expected Mermaid graph.mmd, got %s", cfg.Output.Mermaid)
	}
	if cfg.Output.PlantUML != "graph.puml" {
		t.Errorf("Expected PlantUML graph.puml, got %s", cfg.Output.PlantUML)
	}
	if cfg.Output.Markdown != "analysis-report.md" {
		t.Errorf("Expected Markdown analysis-report.md, got %s", cfg.Output.Markdown)
	}
	if cfg.Output.Report.Verbosity != "detailed" {
		t.Fatalf("expected output.report.verbosity=detailed, got %q", cfg.Output.Report.Verbosity)
	}
	if !cfg.Output.Report.TableOfContentsEnabled() {
		t.Fatal("expected output.report.table_of_contents=true")
	}
	if !cfg.Output.Report.CollapsibleSectionsEnabled() {
		t.Fatal("expected output.report.collapsible_sections=true")
	}
	if !cfg.Output.Report.IncludeMermaidEnabled() {
		t.Fatal("expected output.report.include_mermaid=true")
	}
	if cfg.Output.Paths.DiagramsDir != "docs/diagrams" {
		t.Fatalf("Expected diagrams_dir docs/diagrams, got %q", cfg.Output.Paths.DiagramsDir)
	}
	if !cfg.Output.Diagrams.Architecture {
		t.Fatal("expected output.diagrams.architecture=true")
	}
	if !cfg.Output.Diagrams.Component {
		t.Fatal("expected output.diagrams.component=true")
	}
	if cfg.Output.Diagrams.Flow {
		t.Fatal("expected output.diagrams.flow=false")
	}
	if cfg.Output.Diagrams.FlowConfig.MaxDepth != 10 {
		t.Fatalf("expected flow max_depth=10, got %d", cfg.Output.Diagrams.FlowConfig.MaxDepth)
	}
	if len(cfg.Output.Diagrams.FlowConfig.EntryPoints) != 1 || cfg.Output.Diagrams.FlowConfig.EntryPoints[0] != "cmd/circular/main.go" {
		t.Fatalf("unexpected flow entry points: %#v", cfg.Output.Diagrams.FlowConfig.EntryPoints)
	}
	if !cfg.Output.Diagrams.ComponentCfg.ShowInternal {
		t.Fatal("expected component_config.show_internal=true")
	}
	if len(cfg.Output.UpdateMarkdown) != 1 {
		t.Fatalf("Expected 1 markdown update target, got %d", len(cfg.Output.UpdateMarkdown))
	}
	if cfg.Output.UpdateMarkdown[0].Format != "mermaid" {
		t.Fatalf("Expected markdown format mermaid, got %s", cfg.Output.UpdateMarkdown[0].Format)
	}
	if !cfg.Secrets.Enabled {
		t.Fatal("expected secrets.enabled=true")
	}
	if cfg.Secrets.EntropyThreshold != 4.2 {
		t.Fatalf("expected secrets.entropy_threshold=4.2, got %v", cfg.Secrets.EntropyThreshold)
	}
	if cfg.Secrets.MinTokenLength != 18 {
		t.Fatalf("expected secrets.min_token_length=18, got %d", cfg.Secrets.MinTokenLength)
	}
	if len(cfg.Secrets.Patterns) != 1 || cfg.Secrets.Patterns[0].Name != "custom-test-token" {
		t.Fatalf("unexpected secrets.patterns: %#v", cfg.Secrets.Patterns)
	}
	if len(cfg.Secrets.Exclude.Dirs) != 1 || cfg.Secrets.Exclude.Dirs[0] != ".secrets" {
		t.Fatalf("unexpected secrets.exclude.dirs: %#v", cfg.Secrets.Exclude.Dirs)
	}
	if len(cfg.Secrets.Exclude.Files) != 1 || cfg.Secrets.Exclude.Files[0] != "*.pem" {
		t.Fatalf("unexpected secrets.exclude.files: %#v", cfg.Secrets.Exclude.Files)
	}
}

func TestLoadDefaultDebounce(t *testing.T) {
	content := `grammars_path = "./grammars"`
	tmpfile, err := os.CreateTemp("", "config*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(content))
	tmpfile.Close()

	cfg, _ := Load(tmpfile.Name())
	if cfg.Watch.Debounce != 500*time.Millisecond {
		t.Errorf("Expected default debounce 500ms, got %v", cfg.Watch.Debounce)
	}
}

func TestLoadError(t *testing.T) {
	_, err := Load("nonexistent.toml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	tmpfile, _ := os.CreateTemp("", "badconfig*.toml")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte("bad = toml = format"))
	tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Error("Expected error for malformed TOML")
	}
}

func TestLoadArchitectureRules(t *testing.T) {
	content := `
grammars_path = "./grammars"

[architecture]
enabled = true
top_complexity = 7

[[architecture.layers]]
name = "core"
paths = ["internal/core"]

[[architecture.layers]]
name = "api"
paths = ["internal/api"]

[[architecture.rules]]
name = "api-only-to-core"
from = "api"
allow = ["core"]
`

	tmpfile, err := os.CreateTemp("", "config-architecture*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Architecture.Enabled {
		t.Fatal("expected architecture.enabled to be true")
	}
	if cfg.Architecture.TopComplexity != 7 {
		t.Fatalf("expected top_complexity=7, got %d", cfg.Architecture.TopComplexity)
	}
	if len(cfg.Architecture.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(cfg.Architecture.Layers))
	}
	if len(cfg.Architecture.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Architecture.Rules))
	}
}

func TestLoadArchitectureRules_InvalidOverlap(t *testing.T) {
	content := `
grammars_path = "./grammars"

[architecture]
enabled = true

[[architecture.layers]]
name = "core"
paths = ["internal"]

[[architecture.layers]]
name = "api"
paths = ["internal/api"]
`

	tmpfile, err := os.CreateTemp("", "config-architecture-bad*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected overlap validation error")
	}
}

func TestLoadOutputMarkdownValidation(t *testing.T) {
	content := `
grammars_path = "./grammars"

[output]
dot = "graph.dot"

[[output.update_markdown]]
file = "README.md"
marker = ""
format = "mermaid"
`
	tmpfile, err := os.CreateTemp("", "config-output-bad*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected markdown marker validation error")
	}
}

func TestLoadSecretsValidation_InvalidRegex(t *testing.T) {
	content := `
grammars_path = "./grammars"

[secrets]
enabled = true

[[secrets.patterns]]
name = "bad"
regex = "("
`

	tmpfile, err := os.CreateTemp("", "config-secrets-bad*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected invalid secrets pattern regex error")
	}
	if !strings.Contains(err.Error(), "secrets.patterns[0].regex is invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadOutputPathsDefault(t *testing.T) {
	content := `
grammars_path = "./grammars"
`
	tmpfile, err := os.CreateTemp("", "config-output-paths-default*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output.Paths.DiagramsDir != "docs/diagrams" {
		t.Fatalf("expected default diagrams dir docs/diagrams, got %q", cfg.Output.Paths.DiagramsDir)
	}
	if cfg.Output.Report.Verbosity != "standard" {
		t.Fatalf("expected default report verbosity standard, got %q", cfg.Output.Report.Verbosity)
	}
	if !cfg.Output.Report.TableOfContentsEnabled() {
		t.Fatal("expected output.report.table_of_contents default true")
	}
	if !cfg.Output.Report.CollapsibleSectionsEnabled() {
		t.Fatal("expected output.report.collapsible_sections default true")
	}
	if cfg.Output.Report.IncludeMermaidEnabled() {
		t.Fatal("expected output.report.include_mermaid default false")
	}
	if cfg.Output.Mermaid != "graph.mmd" {
		t.Fatalf("expected default mermaid path graph.mmd, got %q", cfg.Output.Mermaid)
	}
	if !cfg.Output.MermaidEnabled() {
		t.Fatal("expected mermaid output enabled by default")
	}
	if cfg.Output.PlantUMLEnabled() {
		t.Fatal("expected plantuml output disabled by default")
	}
	if cfg.Output.Diagrams.FlowConfig.MaxDepth != 8 {
		t.Fatalf("expected default flow max_depth 8, got %d", cfg.Output.Diagrams.FlowConfig.MaxDepth)
	}
}

func TestOutputFormatsExplicitEnablement(t *testing.T) {
	content := `
grammars_path = "./grammars"

[output]
plantuml = "graph.puml"

[output.formats]
mermaid = false
plantuml = true
`
	tmpfile, err := os.CreateTemp("", "config-output-format-validate*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output.MermaidEnabled() {
		t.Fatal("expected mermaid disabled from output.formats")
	}
	if !cfg.Output.PlantUMLEnabled() {
		t.Fatal("expected plantuml enabled from output.formats")
	}
}

func TestLoadOutputDiagramValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		errSub  string
	}{
		{
			name: "invalid max_depth",
			content: `
grammars_path = "./grammars"

[output.diagrams.flow_config]
max_depth = -1
`,
			errSub: "output.diagrams.flow_config.max_depth must be >= 1",
		},
		{
			name: "empty flow entry point",
			content: `
grammars_path = "./grammars"

[output.diagrams.flow_config]
entry_points = ["", "cmd/circular/main.go"]
max_depth = 4
`,
			errSub: "output.diagrams.flow_config.entry_points[0] must not be empty",
		},
		{
			name: "invalid report verbosity",
			content: `
grammars_path = "./grammars"

[output.report]
verbosity = "loud"
`,
			errSub: "output.report.verbosity must be one of: summary, standard, detailed",
		},
		{
			name: "duplicate flow entry points",
			content: `
grammars_path = "./grammars"

[output.diagrams.flow_config]
entry_points = ["cmd/circular/main.go", "cmd/circular/main.go"]
max_depth = 4
`,
			errSub: "duplicate flow entry point",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-output-diagram-validate*.toml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())
			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			_, err = Load(tmpfile.Name())
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoad_VersionedConfig(t *testing.T) {
	content := `
version = 2

[paths]
project_root = "."
config_dir = "data/config"
database_dir = "data/database"

[config]
active_file = "circular.toml"

[db]
enabled = true
driver = "sqlite"
path = "history.db"
busy_timeout = "3s"
project_mode = "multi"

[mcp]
enabled = false
mode = "server"
transport = "http"
address = "127.0.0.1:8765"

grammars_path = "./grammars"
`
	tmpfile, err := os.CreateTemp("", "config-v2*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("load v2 config: %v", err)
	}
	if cfg.Version != 2 {
		t.Fatalf("expected version=2, got %d", cfg.Version)
	}
	if cfg.DB.BusyTimeout != 3*time.Second {
		t.Fatalf("expected busy timeout 3s, got %v", cfg.DB.BusyTimeout)
	}
	if cfg.MCP.Transport != "http" {
		t.Fatalf("expected mcp transport http, got %q", cfg.MCP.Transport)
	}
}

func TestLoad_BackwardCompatibilityV1(t *testing.T) {
	content := `
grammars_path = "./grammars"
watch_paths = ["./src"]
`
	tmpfile, err := os.CreateTemp("", "config-v1*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("load v1 config: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected default version=1, got %d", cfg.Version)
	}
	if cfg.Paths.ConfigDir != "data/config" {
		t.Fatalf("expected default config dir data/config, got %q", cfg.Paths.ConfigDir)
	}
}

func TestLoad_ProjectsValidation(t *testing.T) {
	content := `
grammars_path = "./grammars"

[projects]
active = "default"

[[projects.entries]]
name = "default"
root = "."
db_namespace = "default"

[[projects.entries]]
name = "default"
root = "./other"
`
	tmpfile, err := os.CreateTemp("", "config-projects*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected duplicate project error")
	}
	if !strings.Contains(err.Error(), "duplicate project name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_ProjectNamespaceValidation(t *testing.T) {
	content := `
grammars_path = "./grammars"

[projects]
active = ""

[[projects.entries]]
name = "alpha"
root = "."
db_namespace = "shared"

[[projects.entries]]
name = "beta"
root = "./other"
db_namespace = "shared"
`
	tmpfile, err := os.CreateTemp("", "config-projects-namespace*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected duplicate namespace error")
	}
	if !strings.Contains(err.Error(), "duplicate project db_namespace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_MCPValidation(t *testing.T) {
	content := `
grammars_path = "./grammars"

[mcp]
enabled = true
mode = "embedded"
transport = "http"
address = "127.0.0.1:8765"
`
	tmpfile, err := os.CreateTemp("", "config-mcp*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected mcp compatibility error")
	}
	if !strings.Contains(err.Error(), "mcp transport http is only valid with mcp.mode=server") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_MCPPOCDefaults(t *testing.T) {
	content := `
grammars_path = "./grammars"

[mcp]
enabled = true
operation_allowlist = ["scan_once"]
`
	tmpfile, err := os.CreateTemp("", "config-mcp-defaults*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MCP.ServerName != "circular" {
		t.Fatalf("expected server_name circular, got %q", cfg.MCP.ServerName)
	}
	if cfg.MCP.ServerVersion != "1.0.0" {
		t.Fatalf("expected server_version 1.0.0, got %q", cfg.MCP.ServerVersion)
	}
	if cfg.MCP.MaxResponseItems != 500 {
		t.Fatalf("expected max_response_items 500, got %d", cfg.MCP.MaxResponseItems)
	}
	if cfg.MCP.RequestTimeout != 30*time.Second {
		t.Fatalf("expected request_timeout 30s, got %v", cfg.MCP.RequestTimeout)
	}
	if !cfg.MCP.AutoManageOutputsEnabled() {
		t.Fatal("expected auto_manage_outputs default true")
	}
	if !cfg.MCP.AutoSyncConfigEnabled() {
		t.Fatal("expected auto_sync_config default true")
	}
	if cfg.MCP.ConfigPath != "circular.toml" {
		t.Fatalf("expected mcp.config_path to default to circular.toml, got %q", cfg.MCP.ConfigPath)
	}
}

func TestLoad_MCPPOCValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		errSub  string
	}{
		{
			name: "missing allowlist",
			content: `
grammars_path = "./grammars"

[mcp]
enabled = true
`,
			errSub: "mcp.operation_allowlist must not be empty",
		},
		{
			name: "invalid max_response_items",
			content: `
grammars_path = "./grammars"

[mcp]
enabled = true
operation_allowlist = ["scan_once"]
max_response_items = -1
`,
			errSub: "mcp.max_response_items must be between 1 and 5000",
		},
		{
			name: "invalid request_timeout",
			content: `
grammars_path = "./grammars"

[mcp]
enabled = true
operation_allowlist = ["scan_once"]
request_timeout = "500ms"
`,
			errSub: "mcp.request_timeout must be between 1s and 2m",
		},
		{
			name: "openapi path and url both set",
			content: `
grammars_path = "./grammars"

[mcp]
enabled = true
operation_allowlist = ["scan_once"]
openapi_spec_path = "openapi.yaml"
openapi_spec_url = "https://example.com/openapi.yaml"
`,
			errSub: "mcp.openapi_spec_path cannot be set alongside mcp.openapi_spec_url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-mcp-validate*.toml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())
			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			_, err = Load(tmpfile.Name())
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoad_GrammarVerificationDefaultsEnabled(t *testing.T) {
	content := `
grammars_path = "./grammars"
`
	tmpfile, err := os.CreateTemp("", "config-grammar-verify-default*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.GrammarVerification.IsEnabled() {
		t.Fatal("expected grammar verification default to enabled")
	}
}

func TestLoad_GrammarVerificationCanBeDisabled(t *testing.T) {
	content := `
grammars_path = "./grammars"

[grammar_verification]
enabled = false
`
	tmpfile, err := os.CreateTemp("", "config-grammar-verify-disabled*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GrammarVerification.IsEnabled() {
		t.Fatal("expected grammar verification to be disabled")
	}
}

func TestLoad_LanguagesValidationRejectsEmptyOverrides(t *testing.T) {
	content := `
grammars_path = "./grammars"

[languages.javascript]
extensions = ["", ".js"]
`
	tmpfile, err := os.CreateTemp("", "config-language-validation*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Fatal("expected language validation error")
	}
}

func TestResolveActiveProject(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Projects: Projects{
			Entries: []ProjectEntry{
				{Name: "root", Root: tmpDir, DBNamespace: "root"},
				{Name: "nested", Root: filepath.Join(tmpDir, "pkg", "sub"), DBNamespace: "nested"},
			},
		},
	}

	cwd := filepath.Join(tmpDir, "pkg", "sub")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	project, err := ResolveActiveProject(cfg, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if project.Name != "nested" {
		t.Fatalf("expected nested project match, got %q", project.Name)
	}
}

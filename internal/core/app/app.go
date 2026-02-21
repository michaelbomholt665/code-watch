package app

import (
	"circular/internal/core/app/helpers"
	"circular/internal/core/config"
	"circular/internal/core/errors"
	"circular/internal/core/ports"
	"circular/internal/core/watcher"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	secretengine "circular/internal/engine/secrets"
	"context"
	"sync"

	"github.com/gobwas/glob"
)

type Update struct {
	Cycles         [][]string
	Hallucinations []resolver.UnresolvedReference
	ModuleCount    int
	FileCount      int
	SecretCount    int
}

type MarkdownReportRequest struct {
	OutputPath string
	WriteFile  bool
	Verbosity  string
}

type MarkdownReportResult struct {
	Markdown string
	Path     string
	Written  bool
}

type App struct {
	Config        *config.Config
	configMu      sync.RWMutex
	codeParser    ports.CodeParser
	Graph         *graph.Graph
	secretScanner ports.SecretScanner
	symbolStore   *graph.SQLiteSymbolStore
	archEngine    *graph.LayerRuleEngine
	goModCache    map[string]goModuleCacheEntry
	IncludeTests  bool

	secretExcludeDirs  []glob.Glob
	secretExcludeFiles []glob.Glob

	updateMu sync.RWMutex
	onUpdate func(Update)

	// Cached unresolved references keyed by file path for incremental updates.
	unresolvedByFile map[string][]resolver.UnresolvedReference
	unresolvedMu     sync.RWMutex

	// Cached unused imports keyed by file path for incremental updates.
	unusedByFile map[string][]resolver.UnusedImport
	unusedMu     sync.RWMutex

	fileContents *graph.LRUCache[string, []byte]

	activeWatcher *watcher.Watcher
}

type Dependencies struct {
	CodeParser    ports.CodeParser
	SecretScanner ports.SecretScanner
}

func (a *App) GetConfig() *config.Config {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.Config
}

func (a *App) UpdateConfig(ctx context.Context, cfg *config.Config) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	// Update components that support hot-reload via Reloadable interface
	if r, ok := a.secretScanner.(ports.Reloadable); ok {
		_ = r.Reload(cfg)
	}

	// Manual updates for components not implementing Reloadable or needing specific handling
	if a.Graph != nil {
		a.Graph.UpdateCapacity(cfg.Caches.Files)
	}
	if a.fileContents != nil {
		a.fileContents.SetCapacity(cfg.Caches.FileContents)
	}

	if a.activeWatcher != nil {
		a.activeWatcher.SetDebounce(cfg.Watch.Debounce)
	}

	// Update secret exclude patterns if they changed
	if cfg.Secrets.Enabled {
		secretExcludeDirs, err := helpers.CompileGlobs(cfg.Secrets.Exclude.Dirs, "secret exclude dir")
		if err == nil {
			a.secretExcludeDirs = secretExcludeDirs
		}
		secretExcludeFiles, err := helpers.CompileGlobs(cfg.Secrets.Exclude.Files, "secret exclude file")
		if err == nil {
			a.secretExcludeFiles = secretExcludeFiles
		}
	}

	a.Config = cfg
	return nil
}

func New(cfg *config.Config) (*App, error) {
	registry, err := buildParserRegistry(cfg)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to build parser registry")
	}
	loader, err := parser.NewGrammarLoaderWithRegistry(cfg.GrammarsPath, registry, cfg.GrammarVerification.IsEnabled())
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to initialize grammar loader")
	}

	parserImpl := parser.NewParser(loader)
	if err := parserImpl.RegisterDefaultExtractors(); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to register default extractors")
	}

	return NewWithDependencies(cfg, Dependencies{
		CodeParser: parser.NewAdapter(parserImpl),
	})
}

func NewWithDependencies(cfg *config.Config, deps Dependencies) (*App, error) {
	if cfg == nil {
		return nil, errors.New(errors.CodeValidationError, "config must not be nil")
	}
	if deps.CodeParser == nil {
		return nil, errors.New(errors.CodeValidationError, "code parser dependency must not be nil")
	}

	secretExcludeDirs, err := helpers.CompileGlobs(cfg.Secrets.Exclude.Dirs, "secret exclude dir")
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeValidationError, "failed to compile secret exclude dirs")
	}
	secretExcludeFiles, err := helpers.CompileGlobs(cfg.Secrets.Exclude.Files, "secret exclude file")
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeValidationError, "failed to compile secret exclude files")
	}
	secretScanner := deps.SecretScanner
	if cfg.Secrets.Enabled && secretScanner == nil {
		customPatterns := make([]secretengine.PatternConfig, 0, len(cfg.Secrets.Patterns))
		for _, pattern := range cfg.Secrets.Patterns {
			customPatterns = append(customPatterns, secretengine.PatternConfig{
				Name:     pattern.Name,
				Regex:    pattern.Regex,
				Severity: pattern.Severity,
			})
		}
		secretScanner, err = secretengine.NewAdapter(secretengine.Config{
			EntropyThreshold: cfg.Secrets.EntropyThreshold,
			MinTokenLength:   cfg.Secrets.MinTokenLength,
			Patterns:         customPatterns,
		})
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternal, "failed to initialize secret scanner")
		}
	}

	app := &App{
		Config:             cfg,
		codeParser:         deps.CodeParser,
		Graph:              graph.NewGraphWithCapacity(cfg.Caches.Files),
		secretScanner:      secretScanner,
		archEngine:         graph.NewLayerRuleEngine(helpers.ArchitectureModelFromConfig(cfg.Architecture)),
		goModCache:         make(map[string]goModuleCacheEntry),
		unresolvedByFile:   make(map[string][]resolver.UnresolvedReference),
		unusedByFile:       make(map[string][]resolver.UnusedImport),
		secretExcludeDirs:  secretExcludeDirs,
		secretExcludeFiles: secretExcludeFiles,
		fileContents:       graph.NewLRUCache[string, []byte](cfg.Caches.FileContents),
	}
	if err := app.initSymbolStore(); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to initialize symbol store")
	}
	return app, nil
}

func buildParserRegistry(cfg *config.Config) (map[string]parser.LanguageSpec, error) {
	overrides := make(map[string]parser.LanguageOverride, len(cfg.Languages))
	for lang, languageCfg := range cfg.Languages {
		overrides[lang] = parser.LanguageOverride{
			Enabled:    languageCfg.Enabled,
			Extensions: append([]string(nil), languageCfg.Extensions...),
			Filenames:  append([]string(nil), languageCfg.Filenames...),
		}
	}

	dynamic := make([]parser.LanguageSpec, 0, len(cfg.DynamicGrammars))
	for _, dg := range cfg.DynamicGrammars {
		dynamic = append(dynamic, parser.LanguageSpec{
			Name:        dg.Name,
			Extensions:  dg.Extensions,
			Filenames:   dg.Filenames,
			IsDynamic:   true,
			LibraryPath: dg.Library,
			SymbolName:  "tree_sitter_" + dg.Name,
			DynamicConfig: &parser.DynamicExtractorConfig{
				NamespaceNode:   dg.NamespaceNode,
				ImportNode:      dg.ImportNode,
				DefinitionNodes: dg.DefinitionNodes,
			},
		})
	}

	registry, err := parser.BuildLanguageRegistry(overrides, dynamic)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to build parser registry")
	}
	return registry, nil
}

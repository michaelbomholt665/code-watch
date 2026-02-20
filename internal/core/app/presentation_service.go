package app

import (
	"circular/internal/core/app/helpers"
	"circular/internal/engine/graph"
	"circular/internal/engine/resolver"
	"circular/internal/shared/version"
	"circular/internal/ui/report"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type PresentationService struct {
	app        *App
	archEngine *graph.LayerRuleEngine
}

func newPresentationService(app *App) *PresentationService {
	return &PresentationService{
		app:        app,
		archEngine: graph.NewLayerRuleEngine(helpers.ArchitectureModelFromConfig(app.Config.Architecture)),
	}
}

func (p *PresentationService) GenerateMarkdownReport(req MarkdownReportRequest) (MarkdownReportResult, error) {
	cycles := p.app.Graph.DetectCycles()
	metrics := p.app.Graph.ComputeModuleMetrics()
	hotspots := p.app.Graph.TopComplexity(p.app.Config.Architecture.TopComplexity)
	violations := p.app.ArchitectureViolations()
	unresolved := p.app.AnalyzeHallucinations()
	unused := p.app.AnalyzeUnusedImports()

	root, err := p.app.resolveOutputRoot()
	if err != nil {
		return MarkdownReportResult{}, err
	}

	var mermaidDiagram string
	if p.app.Config.Output.Report.IncludeMermaidEnabled() && p.app.Config.Output.MermaidEnabled() {
		mermaidGen := report.NewMermaidGenerator(p.app.Graph)
		mermaidGen.SetModuleMetrics(metrics)
		mermaidGen.SetComplexityHotspots(hotspots)
		mermaidDiagram, err = mermaidGen.Generate(cycles, violations, helpers.ArchitectureModelFromConfig(p.app.Config.Architecture))
		if err != nil {
			return MarkdownReportResult{}, fmt.Errorf("generate mermaid diagram for markdown report: %w", err)
		}
	}

	verbosity := strings.TrimSpace(req.Verbosity)
	if verbosity == "" {
		verbosity = p.app.Config.Output.Report.Verbosity
	}
	md, err := report.NewMarkdownGenerator().Generate(report.MarkdownReportData{
		TotalModules:  p.app.Graph.ModuleCount(),
		TotalFiles:    p.app.Graph.FileCount(),
		Cycles:        cycles,
		Unresolved:    unresolved,
		UnusedImports: unused,
		Violations:    violations,
		Hotspots:      hotspots,
	}, report.MarkdownReportOptions{
		ProjectName:         filepath.Base(root),
		ProjectRoot:         root,
		Version:             version.Version,
		GeneratedAt:         time.Now().UTC(),
		Verbosity:           verbosity,
		TableOfContents:     p.app.Config.Output.Report.TableOfContentsEnabled(),
		CollapsibleSections: p.app.Config.Output.Report.CollapsibleSectionsEnabled(),
		IncludeMermaid:      p.app.Config.Output.Report.IncludeMermaidEnabled(),
		MermaidDiagram:      mermaidDiagram,
	})
	if err != nil {
		return MarkdownReportResult{}, fmt.Errorf("generate markdown report: %w", err)
	}

	outPath := strings.TrimSpace(req.OutputPath)
	if outPath == "" {
		outPath = strings.TrimSpace(p.app.Config.Output.Markdown)
	}
	if outPath == "" && req.WriteFile {
		outPath = "analysis-report.md"
	}
	if outPath != "" && !filepath.IsAbs(outPath) {
		outPath = filepath.Join(root, outPath)
	}

	result := MarkdownReportResult{Markdown: md, Path: outPath}
	if req.WriteFile || outPath != "" {
		if outPath == "" {
			return MarkdownReportResult{}, fmt.Errorf("output path is required when write_file=true")
		}
		if err := helpers.WriteArtifact(outPath, md); err != nil {
			return MarkdownReportResult{}, fmt.Errorf("write markdown report %q: %w", outPath, err)
		}
		result.Written = true
	}
	return result, nil
}

func (p *PresentationService) PrintSummary(
	fileCount, moduleCount int,
	duration time.Duration,
	cycles [][]string,
	hallucinations []resolver.UnresolvedReference,
	unusedImports []resolver.UnusedImport,
	metrics map[string]graph.ModuleMetrics,
	violations []graph.ArchitectureViolation,
	hotspots []graph.ComplexityHotspot,
) {
	if !p.app.Config.Alerts.Terminal {
		return
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Update: %d files, %d modules in %v\n", fileCount, moduleCount, duration)

	if len(cycles) > 0 {
		fmt.Printf("⚠️  FOUND %d CIRCULAR IMPORTS:\n", len(cycles))
		for _, c := range cycles {
			fmt.Printf("   %s\n", strings.Join(c, " -> "))
		}
	} else {
		fmt.Println("✅ No circular imports found.")
	}

	if len(hallucinations) > 0 {
		fmt.Printf("❓ FOUND %d UNRESOLVED REFERENCES:\n", len(hallucinations))
		for _, h := range hallucinations {
			fmt.Printf("   %s in %s:%d\n", h.Reference.Name, h.File, h.Reference.Location.Line)
		}
	} else {
		fmt.Println("✅ No unresolved references found.")
	}

	if len(unusedImports) > 0 {
		fmt.Printf("🧹 FOUND %d UNUSED IMPORTS:\n", len(unusedImports))
		for _, u := range unusedImports {
			target := u.Module
			if u.Item != "" {
				target = target + "." + u.Item
			}
			fmt.Printf("   %s in %s:%d\n", target, u.File, u.Location.Line)
		}
	} else {
		fmt.Println("✅ No unused imports found.")
	}

	secretCount := p.app.SecretCount()
	if secretCount > 0 {
		fmt.Printf("🔐 FOUND %d POTENTIAL SECRETS\n", secretCount)
		for _, finding := range p.app.allSecrets(5) {
			fmt.Printf("   %s [%s] %s:%d (%s)\n",
				finding.Kind,
				finding.Severity,
				finding.Location.File,
				finding.Location.Line,
				helpers.MaskSecretValue(finding.Value),
			)
		}
	} else {
		fmt.Println("✅ No hardcoded secrets found.")
	}

	if len(metrics) > 0 {
		topDepth := helpers.MetricLeaders(metrics, func(m graph.ModuleMetrics) int { return m.Depth }, 3, 0)
		topFanIn := helpers.MetricLeaders(metrics, func(m graph.ModuleMetrics) int { return m.FanIn }, 3, 1)
		topFanOut := helpers.MetricLeaders(metrics, func(m graph.ModuleMetrics) int { return m.FanOut }, 3, 1)

		fmt.Println("📊 Dependency Metrics:")
		if len(topDepth) > 0 {
			fmt.Printf("   Deepest modules: %s\n", strings.Join(topDepth, ", "))
		}
		if len(topFanIn) > 0 {
			fmt.Printf("   Highest fan-in: %s\n", strings.Join(topFanIn, ", "))
		}
		if len(topFanOut) > 0 {
			fmt.Printf("   Highest fan-out: %s\n", strings.Join(topFanOut, ", "))
		}
	}

	if len(violations) > 0 {
		fmt.Printf("🏛️  FOUND %d ARCHITECTURE VIOLATIONS:\n", len(violations))
		for _, v := range violations {
			fmt.Printf("   %s (%s -> %s) in %s:%d\n", v.RuleName, v.FromLayer, v.ToLayer, v.File, v.Line)
		}
	} else if p.app.Config.Architecture.Enabled {
		fmt.Println("✅ No architecture violations found.")
	}

	if len(hotspots) > 0 {
		fmt.Println("🔥 Top complexity hotspots:")
		for _, h := range hotspots {
			fmt.Printf("   %s.%s score=%d (branches=%d params=%d depth=%d loc=%d)\n", h.Module, h.Definition, h.Score, h.Branches, h.Parameters, h.Nesting, h.LOC)
		}
	}
	fmt.Println(strings.Repeat("-", 40))
}

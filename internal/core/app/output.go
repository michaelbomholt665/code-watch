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

func (a *App) SetUpdateHandler(handler func(Update)) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	a.onUpdate = handler
}

func (a *App) CurrentUpdate() Update {
	return Update{
		Cycles:         a.Graph.DetectCycles(),
		Hallucinations: a.AnalyzeHallucinations(),
		ModuleCount:    a.Graph.ModuleCount(),
		FileCount:      a.Graph.FileCount(),
		SecretCount:    a.SecretCount(),
	}
}

func (a *App) emitUpdate(update Update) {
	a.updateMu.RLock()
	handler := a.onUpdate
	a.updateMu.RUnlock()
	if handler != nil {
		handler(update)
	}
}

func (a *App) GenerateOutputs(
	cycles [][]string,
	unresolved []resolver.UnresolvedReference,
	unusedImports []resolver.UnusedImport,
	metrics map[string]graph.ModuleMetrics,
	violations []graph.ArchitectureViolation,
	hotspots []graph.ComplexityHotspot,
) error {
	archModel := helpers.ArchitectureModelFromConfig(a.Config.Architecture)
	diagramModes, err := helpers.ResolveDiagramModes(a.Config.Output.Diagrams)
	if err != nil {
		return err
	}
	mermaidDiagrams := make(map[helpers.DiagramMode]string, len(diagramModes))
	plantUMLDiagrams := make(map[helpers.DiagramMode]string, len(diagramModes))
	targets, err := a.resolveOutputTargets()
	if err != nil {
		return err
	}

	if targets.DOT != "" {
		dotGen := report.NewDOTGenerator(a.Graph)
		dotGen.SetModuleMetrics(metrics)
		dotGen.SetComplexityHotspots(hotspots)
		dot, err := dotGen.Generate(cycles)
		if err != nil {
			return fmt.Errorf("generate DOT output: %w", err)
		}
		if err := helpers.WriteArtifact(targets.DOT, dot); err != nil {
			return fmt.Errorf("write DOT output %q: %w", targets.DOT, err)
		}
	}

	if targets.TSV != "" {
		tsvGen := report.NewTSVGenerator(a.Graph)
		dependenciesTSV, err := tsvGen.Generate()
		if err != nil {
			return fmt.Errorf("generate TSV output: %w", err)
		}
		tsv := dependenciesTSV

		if len(unusedImports) > 0 {
			unusedTSV, err := tsvGen.GenerateUnusedImports(unusedImports)
			if err != nil {
				return fmt.Errorf("generate unused-import TSV block: %w", err)
			}
			tsv = strings.TrimRight(dependenciesTSV, "\n") + "\n\n" + strings.TrimRight(unusedTSV, "\n") + "\n"
		}
		if len(violations) > 0 {
			violationsTSV, err := tsvGen.GenerateArchitectureViolations(violations)
			if err != nil {
				return fmt.Errorf("generate architecture-violation TSV block: %w", err)
			}
			tsv = strings.TrimRight(tsv, "\n") + "\n\n" + strings.TrimRight(violationsTSV, "\n") + "\n"
		}
		allSecrets := a.allSecrets(0)
		if len(allSecrets) > 0 {
			secretsTSV, err := tsvGen.GenerateSecrets(allSecrets)
			if err != nil {
				return fmt.Errorf("generate secrets TSV block: %w", err)
			}
			tsv = strings.TrimRight(tsv, "\n") + "\n\n" + strings.TrimRight(secretsTSV, "\n") + "\n"
		}

		if err := helpers.WriteArtifact(targets.TSV, tsv); err != nil {
			return fmt.Errorf("write TSV output %q: %w", targets.TSV, err)
		}
	}

	needMermaid := targets.Mermaid != "" && a.Config.Output.MermaidEnabled()
	needPlantUML := targets.PlantUML != "" && a.Config.Output.PlantUMLEnabled()
	if targets.Markdown != "" && a.Config.Output.Report.IncludeMermaidEnabled() && a.Config.Output.MermaidEnabled() {
		needMermaid = true
	}
	for _, injection := range a.Config.Output.UpdateMarkdown {
		switch strings.ToLower(strings.TrimSpace(injection.Format)) {
		case "mermaid":
			if a.Config.Output.MermaidEnabled() {
				needMermaid = true
			}
		case "plantuml":
			if a.Config.Output.PlantUMLEnabled() {
				needPlantUML = true
			}
		}
	}

	if needMermaid {
		mermaidGen := report.NewMermaidGenerator(a.Graph)
		mermaidGen.SetModuleMetrics(metrics)
		mermaidGen.SetComplexityHotspots(hotspots)
		for _, mode := range diagramModes {
			diagram, genErr := helpers.GenerateMermaidByMode(mermaidGen, mode, archModel, cycles, violations, a.Config.Output.Diagrams)
			if genErr != nil {
				return fmt.Errorf("generate Mermaid output (%s): %w", mode.Suffix(), genErr)
			}
			mermaidDiagrams[mode] = diagram
			if targets.Mermaid != "" {
				outPath := helpers.DiagramOutputPath(targets.Mermaid, mode, len(diagramModes) > 1)
				if err := helpers.WriteArtifact(outPath, diagram); err != nil {
					return fmt.Errorf("write Mermaid output %q: %w", outPath, err)
				}
			}
		}
	}

	if needPlantUML {
		plantUMLGen := report.NewPlantUMLGenerator(a.Graph)
		plantUMLGen.SetModuleMetrics(metrics)
		plantUMLGen.SetComplexityHotspots(hotspots)
		for _, mode := range diagramModes {
			diagram, genErr := helpers.GeneratePlantUMLByMode(plantUMLGen, mode, archModel, cycles, violations, a.Config.Output.Diagrams)
			if genErr != nil {
				return fmt.Errorf("generate PlantUML output (%s): %w", mode.Suffix(), genErr)
			}
			plantUMLDiagrams[mode] = diagram
			if targets.PlantUML != "" {
				outPath := helpers.DiagramOutputPath(targets.PlantUML, mode, len(diagramModes) > 1)
				if err := helpers.WriteArtifact(outPath, diagram); err != nil {
					return fmt.Errorf("write PlantUML output %q: %w", outPath, err)
				}
			}
		}
	}

	injectionMode := helpers.PreferredInjectionMode(diagramModes)
	for _, injection := range a.Config.Output.UpdateMarkdown {
		format := strings.ToLower(strings.TrimSpace(injection.Format))
		diagram := ""
		switch format {
		case "mermaid":
			diagram = markdownDiagramBlock("mermaid", mermaidDiagrams[injectionMode])
		case "plantuml":
			diagram = markdownDiagramBlock("plantuml", plantUMLDiagrams[injectionMode])
		default:
			continue
		}

		if err := report.InjectDiagram(injection.File, injection.Marker, diagram); err != nil {
			return fmt.Errorf("inject %s diagram into %q with marker %q: %w", format, injection.File, injection.Marker, err)
		}
	}

	if targets.Markdown != "" {
		if unresolved == nil {
			unresolved = a.AnalyzeHallucinations()
		}
		root, err := a.resolveOutputRoot()
		if err != nil {
			return err
		}
		// Use the same logic as PresentationService for consistency
		md, err := report.NewMarkdownGenerator().Generate(report.MarkdownReportData{
			TotalModules:  a.Graph.ModuleCount(),
			TotalFiles:    a.Graph.FileCount(),
			Cycles:        cycles,
			Unresolved:    unresolved,
			UnusedImports: unusedImports,
			Violations:    violations,
			Hotspots:      hotspots,
		}, report.MarkdownReportOptions{
			ProjectName:         filepath.Base(root),
			ProjectRoot:         root,
			Version:             version.Version,
			GeneratedAt:         time.Now().UTC(),
			Verbosity:           a.Config.Output.Report.Verbosity,
			TableOfContents:     a.Config.Output.Report.TableOfContentsEnabled(),
			CollapsibleSections: a.Config.Output.Report.CollapsibleSectionsEnabled(),
			IncludeMermaid:      a.Config.Output.Report.IncludeMermaidEnabled(),
			MermaidDiagram:      mermaidDiagrams[injectionMode],
		})
		if err != nil {
			return fmt.Errorf("generate Markdown report: %w", err)
		}
		if err := helpers.WriteArtifact(targets.Markdown, md); err != nil {
			return fmt.Errorf("write Markdown report %q: %w", targets.Markdown, err)
		}
	}

	return nil
}

func markdownDiagramBlock(format, diagram string) string {
	trimmed := strings.TrimRight(diagram, "\n")
	return "```" + format + "\n" + trimmed + "\n```"
}

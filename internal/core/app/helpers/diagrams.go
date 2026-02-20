package helpers

import (
	"circular/internal/core/config"
	"circular/internal/engine/graph"
	"circular/internal/ui/report"
	"path/filepath"
	"strings"
)

type DiagramMode int

const (
	DiagramModeDependency DiagramMode = iota
	DiagramModeArchitecture
	DiagramModeComponent
	DiagramModeFlow
)

func ResolveDiagramMode(diagrams config.DiagramOutput) (DiagramMode, error) {
	modes, err := ResolveDiagramModes(diagrams)
	if err != nil {
		return DiagramModeDependency, err
	}
	if len(modes) == 0 {
		return DiagramModeDependency, nil
	}
	return modes[0], nil
}

func ResolveDiagramModes(diagrams config.DiagramOutput) ([]DiagramMode, error) {
	modes := make([]DiagramMode, 0, 4)
	selected := 0
	if diagrams.Architecture {
		selected++
		modes = append(modes, DiagramModeArchitecture)
	}
	if diagrams.Component {
		selected++
		modes = append(modes, DiagramModeComponent)
	}
	if diagrams.Flow {
		selected++
		modes = append(modes, DiagramModeFlow)
	}
	if selected == 0 {
		return []DiagramMode{DiagramModeDependency}, nil
	}
	if selected > 1 {
		return append([]DiagramMode{DiagramModeDependency}, modes...), nil
	}
	return modes, nil
}

func (m DiagramMode) Suffix() string {
	switch m {
	case DiagramModeArchitecture:
		return "architecture"
	case DiagramModeComponent:
		return "component"
	case DiagramModeFlow:
		return "flow"
	default:
		return "dependency"
	}
}

func PreferredInjectionMode(modes []DiagramMode) DiagramMode {
	if len(modes) == 0 {
		return DiagramModeDependency
	}
	for _, mode := range modes {
		if mode == DiagramModeDependency {
			return mode
		}
	}
	return modes[0]
}

func DiagramOutputPath(base string, mode DiagramMode, suffixOutput bool) string {
	if !suffixOutput {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return stem + "-" + mode.Suffix() + ext
}

func GenerateMermaidByMode(
	gen *report.MermaidGenerator,
	mode DiagramMode,
	archModel graph.ArchitectureModel,
	cycles [][]string,
	violations []graph.ArchitectureViolation,
	diagrams config.DiagramOutput,
) (string, error) {
	switch mode {
	case DiagramModeArchitecture:
		return gen.GenerateArchitecture(archModel, violations)
	case DiagramModeComponent:
		return gen.GenerateComponent(archModel, diagrams.ComponentCfg.ShowInternal)
	case DiagramModeFlow:
		return gen.GenerateFlow(diagrams.FlowConfig.EntryPoints, diagrams.FlowConfig.MaxDepth)
	default:
		return gen.Generate(cycles, violations, archModel)
	}
}

func GeneratePlantUMLByMode(
	gen *report.PlantUMLGenerator,
	mode DiagramMode,
	archModel graph.ArchitectureModel,
	cycles [][]string,
	violations []graph.ArchitectureViolation,
	diagrams config.DiagramOutput,
) (string, error) {
	switch mode {
	case DiagramModeArchitecture:
		return gen.GenerateArchitecture(archModel, violations)
	case DiagramModeComponent:
		return gen.GenerateComponent(archModel, diagrams.ComponentCfg.ShowInternal)
	case DiagramModeFlow:
		return gen.GenerateFlow(diagrams.FlowConfig.EntryPoints, diagrams.FlowConfig.MaxDepth)
	default:
		return gen.Generate(cycles, violations, archModel)
	}
}

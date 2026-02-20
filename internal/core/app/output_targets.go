package app

import (
	"circular/internal/core/app/helpers"
	"circular/internal/core/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type outputTargets struct {
	DOT      string
	TSV      string
	Mermaid  string
	PlantUML string
	Markdown string
}

func (a *App) resolveOutputTargets() (outputTargets, error) {
	root, err := a.resolveOutputRoot()
	if err != nil {
		return outputTargets{}, err
	}
	diagramsDir := strings.TrimSpace(a.Config.Output.Paths.DiagramsDir)
	if diagramsDir == "" {
		diagramsDir = "docs/diagrams"
	}
	if !filepath.IsAbs(diagramsDir) {
		diagramsDir = filepath.Join(root, diagramsDir)
	}
	targets := outputTargets{
		DOT:      helpers.ResolveOutputPath(strings.TrimSpace(a.Config.Output.DOT), root),
		TSV:      helpers.ResolveOutputPath(strings.TrimSpace(a.Config.Output.TSV), root),
		Mermaid:  helpers.ResolveDiagramPath(strings.TrimSpace(a.Config.Output.Mermaid), root, diagramsDir),
		PlantUML: helpers.ResolveDiagramPath(strings.TrimSpace(a.Config.Output.PlantUML), root, diagramsDir),
		Markdown: helpers.ResolveOutputPath(strings.TrimSpace(a.Config.Output.Markdown), root),
	}
	return targets, nil
}

func (a *App) resolveOutputRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	paths, err := config.ResolvePaths(a.Config, cwd)
	if err != nil {
		return "", fmt.Errorf("resolve output root: %w", err)
	}
	return paths.OutputRoot, nil
}

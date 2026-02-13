package graph

import (
	"errors"
	"fmt"
	"sort"
)

var ErrImpactTargetNotFound = errors.New("impact target not found")

type ImpactReport struct {
	TargetPath            string
	TargetModule          string
	DirectImporters       []string
	TransitiveImporters   []string
	ExternallyUsedSymbols []string
}

type ImpactTargetError struct {
	Target string
}

func (e *ImpactTargetError) Error() string {
	return fmt.Sprintf("%v: %s", ErrImpactTargetNotFound, e.Target)
}

func (e *ImpactTargetError) Unwrap() error {
	return ErrImpactTargetNotFound
}

func (g *Graph) AnalyzeImpact(path string) (ImpactReport, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	targetFile, ok := g.files[path]
	if !ok {
		if mod, exists := g.modules[path]; exists {
			targetPath := ""
			if len(mod.Files) > 0 {
				files := append([]string(nil), mod.Files...)
				sort.Strings(files)
				targetPath = files[0]
			}
			return g.analyzeImpactForModule(path, targetPath), nil
		}
		return ImpactReport{}, &ImpactTargetError{Target: path}
	}

	return g.analyzeImpactForModule(targetFile.Module, path), nil
}

func (g *Graph) analyzeImpactForModule(targetModule, targetPath string) ImpactReport {
	report := ImpactReport{
		TargetPath:   targetPath,
		TargetModule: targetModule,
	}

	direct := make([]string, 0, len(g.importedBy[targetModule]))
	for importer := range g.importedBy[targetModule] {
		direct = append(direct, importer)
	}
	sort.Strings(direct)
	report.DirectImporters = direct

	directSet := make(map[string]bool, len(direct))
	for _, importer := range direct {
		directSet[importer] = true
	}

	queue := append([]string(nil), direct...)
	seen := make(map[string]bool, len(queue))
	for _, mod := range queue {
		seen[mod] = true
	}

	transitive := make([]string, 0)
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for next := range g.importedBy[curr] {
			if seen[next] {
				continue
			}
			seen[next] = true
			queue = append(queue, next)
			if !directSet[next] {
				transitive = append(transitive, next)
			}
		}
	}
	sort.Strings(transitive)
	report.TransitiveImporters = transitive

	if mod, ok := g.modules[targetModule]; ok {
		symbols := make([]string, 0, len(mod.Exports))
		for symbol := range mod.Exports {
			symbols = append(symbols, symbol)
		}
		sort.Strings(symbols)
		if len(direct) > 0 || len(transitive) > 0 {
			report.ExternallyUsedSymbols = symbols
		}
	}

	return report
}

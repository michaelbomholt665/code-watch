package query

import (
	"circular/internal/graph"
	"circular/internal/history"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type snapshotReader interface {
	LoadSnapshots(projectKey string, since time.Time) ([]history.Snapshot, error)
}

type Service struct {
	graph      *graph.Graph
	history    snapshotReader
	projectKey string
}

func NewService(g *graph.Graph, h snapshotReader, projectKey string) *Service {
	return &Service{
		graph:      g,
		history:    h,
		projectKey: projectKey,
	}
}

func (s *Service) ListModules(ctx context.Context, filter string, limit int) ([]ModuleSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	modules := s.graph.Modules()
	imports := s.graph.GetImports()

	reverseCounts := make(map[string]int)
	for _, edges := range imports {
		for to := range edges {
			reverseCounts[to]++
		}
	}

	filter = strings.ToLower(strings.TrimSpace(filter))
	rows := make([]ModuleSummary, 0, len(modules))
	for name, module := range modules {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		rows = append(rows, ModuleSummary{
			Name:                   name,
			FileCount:              len(module.Files),
			ExportCount:            len(module.Exports),
			DependencyCount:        len(imports[name]),
			ReverseDependencyCount: reverseCounts[name],
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	if limit > 0 && len(rows) > limit {
		return rows[:limit], nil
	}
	return rows, nil
}

func (s *Service) ModuleDetails(ctx context.Context, moduleName string) (ModuleDetails, error) {
	if err := ctx.Err(); err != nil {
		return ModuleDetails{}, err
	}

	module, ok := s.graph.GetModule(moduleName)
	if !ok {
		return ModuleDetails{}, fmt.Errorf("module not found: %s", moduleName)
	}

	imports := s.graph.GetImports()
	dependencies := make([]DependencyEdge, 0, len(imports[moduleName]))
	for _, edge := range imports[moduleName] {
		dependencies = append(dependencies, DependencyEdge{
			From:   edge.From,
			To:     edge.To,
			File:   edge.ImportedBy,
			Line:   edge.Location.Line,
			Column: edge.Location.Column,
		})
	}
	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].To == dependencies[j].To {
			if dependencies[i].File == dependencies[j].File {
				if dependencies[i].Line == dependencies[j].Line {
					return dependencies[i].Column < dependencies[j].Column
				}
				return dependencies[i].Line < dependencies[j].Line
			}
			return dependencies[i].File < dependencies[j].File
		}
		return dependencies[i].To < dependencies[j].To
	})

	reverseSet := make(map[string]bool)
	for from, edges := range imports {
		if from == moduleName {
			continue
		}
		if _, ok := edges[moduleName]; ok {
			reverseSet[from] = true
		}
	}
	reverse := make([]string, 0, len(reverseSet))
	for from := range reverseSet {
		reverse = append(reverse, from)
	}
	sort.Strings(reverse)

	files := append([]string(nil), module.Files...)
	sort.Strings(files)

	symbols := make([]string, 0, len(module.Exports))
	for symbol := range module.Exports {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	return ModuleDetails{
		Name:                moduleName,
		Files:               files,
		ExportedSymbols:     symbols,
		Dependencies:        dependencies,
		ReverseDependencies: reverse,
	}, nil
}

func (s *Service) DependencyTrace(ctx context.Context, from, to string, maxDepth int) (TraceResult, error) {
	if err := ctx.Err(); err != nil {
		return TraceResult{}, err
	}

	path, ok := s.graph.FindImportChain(from, to)
	if !ok {
		return TraceResult{}, fmt.Errorf("no path from %s to %s", from, to)
	}
	depth := len(path) - 1
	if maxDepth > 0 && depth > maxDepth {
		return TraceResult{}, fmt.Errorf("trace depth %d exceeds max_depth %d", depth, maxDepth)
	}

	return TraceResult{
		From:  from,
		To:    to,
		Path:  path,
		Depth: depth,
	}, nil
}

func (s *Service) TrendSlice(ctx context.Context, since time.Time, limit int) (TrendSlice, error) {
	if err := ctx.Err(); err != nil {
		return TrendSlice{}, err
	}
	if s.history == nil {
		return TrendSlice{}, fmt.Errorf("history store unavailable")
	}

	snapshots, err := s.history.LoadSnapshots(s.projectKey, since)
	if err != nil {
		return TrendSlice{}, err
	}

	if limit > 0 && len(snapshots) > limit {
		snapshots = snapshots[len(snapshots)-limit:]
	}

	out := TrendSlice{
		ScanCount: len(snapshots),
		Snapshots: snapshots,
	}
	if len(snapshots) > 0 {
		out.Since = snapshots[0].Timestamp.Format(time.RFC3339)
		out.Until = snapshots[len(snapshots)-1].Timestamp.Format(time.RFC3339)
	}
	return out, nil
}

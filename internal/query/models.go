package query

import "circular/internal/history"

type ModuleSummary struct {
	Name                   string
	FileCount              int
	ExportCount            int
	DependencyCount        int
	ReverseDependencyCount int
}

type ModuleDetails struct {
	Name                string
	Files               []string
	ExportedSymbols     []string
	Dependencies        []DependencyEdge
	ReverseDependencies []string
}

type DependencyEdge struct {
	From   string
	To     string
	File   string
	Line   int
	Column int
}

type TraceResult struct {
	From  string
	To    string
	Path  []string
	Depth int
}

type TrendSlice struct {
	Since     string
	Until     string
	ScanCount int
	Snapshots []history.Snapshot
}

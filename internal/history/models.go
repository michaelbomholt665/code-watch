package history

import "time"

const SchemaVersion = 1

type Snapshot struct {
	SchemaVersion     int       `json:"schema_version"`
	Timestamp         time.Time `json:"timestamp"`
	CommitHash        string    `json:"commit_hash,omitempty"`
	CommitTimestamp   time.Time `json:"commit_timestamp,omitempty"`
	ModuleCount       int       `json:"module_count"`
	FileCount         int       `json:"file_count"`
	CycleCount        int       `json:"cycle_count"`
	UnresolvedCount   int       `json:"unresolved_count"`
	UnusedImportCount int       `json:"unused_import_count"`
	ViolationCount    int       `json:"violation_count"`
	HotspotCount      int       `json:"hotspot_count"`
	AvgFanIn          float64   `json:"avg_fan_in"`
	AvgFanOut         float64   `json:"avg_fan_out"`
	MaxFanIn          int       `json:"max_fan_in"`
	MaxFanOut         int       `json:"max_fan_out"`
}

type TrendPoint struct {
	Timestamp          time.Time `json:"timestamp"`
	CommitHash         string    `json:"commit_hash,omitempty"`
	ModuleCount        int       `json:"module_count"`
	FileCount          int       `json:"file_count"`
	CycleCount         int       `json:"cycle_count"`
	UnresolvedCount    int       `json:"unresolved_count"`
	UnusedImportCount  int       `json:"unused_import_count"`
	ViolationCount     int       `json:"violation_count"`
	HotspotCount       int       `json:"hotspot_count"`
	AvgFanIn           float64   `json:"avg_fan_in"`
	AvgFanOut          float64   `json:"avg_fan_out"`
	MaxFanIn           int       `json:"max_fan_in"`
	MaxFanOut          int       `json:"max_fan_out"`
	DeltaModules       int       `json:"delta_modules"`
	DeltaFiles         int       `json:"delta_files"`
	DeltaCycles        int       `json:"delta_cycles"`
	DeltaUnresolved    int       `json:"delta_unresolved"`
	DeltaUnusedImports int       `json:"delta_unused_imports"`
	DeltaViolations    int       `json:"delta_violations"`
	DeltaAvgFanIn      float64   `json:"delta_avg_fan_in"`
	DeltaAvgFanOut     float64   `json:"delta_avg_fan_out"`
	ModuleGrowthPct    float64   `json:"module_growth_pct"`
	AvgCycles          float64   `json:"avg_cycles"`
	AvgUnresolved      float64   `json:"avg_unresolved"`
	WindowHours        float64   `json:"window_hours"`
}

type TrendReport struct {
	SchemaVersion int          `json:"schema_version"`
	Since         time.Time    `json:"since"`
	Until         time.Time    `json:"until"`
	Window        string       `json:"window"`
	ScanCount     int          `json:"scan_count"`
	Points        []TrendPoint `json:"points"`
}

package history

import (
	"fmt"
	"math"
	"time"
)

func BuildTrendReport(snapshots []Snapshot, window time.Duration) (TrendReport, error) {
	if len(snapshots) == 0 {
		return TrendReport{}, fmt.Errorf("no snapshots available")
	}

	points := make([]TrendPoint, 0, len(snapshots))
	for i, current := range snapshots {
		point := TrendPoint{
			Timestamp:         current.Timestamp,
			CommitHash:        current.CommitHash,
			ModuleCount:       current.ModuleCount,
			FileCount:         current.FileCount,
			CycleCount:        current.CycleCount,
			UnresolvedCount:   current.UnresolvedCount,
			UnusedImportCount: current.UnusedImportCount,
			ViolationCount:    current.ViolationCount,
			HotspotCount:      current.HotspotCount,
			AvgFanIn:          current.AvgFanIn,
			AvgFanOut:         current.AvgFanOut,
			MaxFanIn:          current.MaxFanIn,
			MaxFanOut:         current.MaxFanOut,
		}

		if i > 0 {
			prev := snapshots[i-1]
			point.DeltaModules = current.ModuleCount - prev.ModuleCount
			point.DeltaFiles = current.FileCount - prev.FileCount
			point.DeltaCycles = current.CycleCount - prev.CycleCount
			point.DeltaUnresolved = current.UnresolvedCount - prev.UnresolvedCount
			point.DeltaUnusedImports = current.UnusedImportCount - prev.UnusedImportCount
			point.DeltaViolations = current.ViolationCount - prev.ViolationCount
			point.DeltaAvgFanIn = current.AvgFanIn - prev.AvgFanIn
			point.DeltaAvgFanOut = current.AvgFanOut - prev.AvgFanOut
			if prev.ModuleCount > 0 {
				point.ModuleGrowthPct = (float64(point.DeltaModules) / float64(prev.ModuleCount)) * 100
			}
		}

		avgCycles, avgUnresolved := movingAverages(snapshots, i, window)
		point.AvgCycles = round2(avgCycles)
		point.AvgUnresolved = round2(avgUnresolved)
		point.WindowHours = round2(window.Hours())
		points = append(points, point)
	}

	return TrendReport{
		SchemaVersion: SchemaVersion,
		Since:         snapshots[0].Timestamp,
		Until:         snapshots[len(snapshots)-1].Timestamp,
		Window:        window.String(),
		ScanCount:     len(points),
		Points:        points,
	}, nil
}

func movingAverages(snapshots []Snapshot, index int, window time.Duration) (float64, float64) {
	if window <= 0 {
		return float64(snapshots[index].CycleCount), float64(snapshots[index].UnresolvedCount)
	}

	cutoff := snapshots[index].Timestamp.Add(-window)
	var cyclesTotal int
	var unresolvedTotal int
	count := 0
	for i := index; i >= 0; i-- {
		if snapshots[i].Timestamp.Before(cutoff) {
			break
		}
		cyclesTotal += snapshots[i].CycleCount
		unresolvedTotal += snapshots[i].UnresolvedCount
		count++
	}
	if count == 0 {
		return 0, 0
	}
	return float64(cyclesTotal) / float64(count), float64(unresolvedTotal) / float64(count)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

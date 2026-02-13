package report

import (
	"circular/internal/data/history"
	"encoding/json"
	"fmt"
	"strings"
)

func RenderTrendTSV(report history.TrendReport) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("Timestamp\tCommit\tModules\tFiles\tCycles\tUnresolved\tUnusedImports\tViolations\tHotspots\tAvgFanIn\tAvgFanOut\tMaxFanIn\tMaxFanOut\tDeltaModules\tDeltaFiles\tDeltaCycles\tDeltaUnresolved\tDeltaUnusedImports\tDeltaViolations\tDeltaAvgFanIn\tDeltaAvgFanOut\tModuleGrowthPct\tAvgCycles\tAvgUnresolved\tWindowHours\n")
	for _, point := range report.Points {
		buf.WriteString(fmt.Sprintf(
			"%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.2f\t%.2f\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\n",
			point.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			point.CommitHash,
			point.ModuleCount,
			point.FileCount,
			point.CycleCount,
			point.UnresolvedCount,
			point.UnusedImportCount,
			point.ViolationCount,
			point.HotspotCount,
			point.AvgFanIn,
			point.AvgFanOut,
			point.MaxFanIn,
			point.MaxFanOut,
			point.DeltaModules,
			point.DeltaFiles,
			point.DeltaCycles,
			point.DeltaUnresolved,
			point.DeltaUnusedImports,
			point.DeltaViolations,
			point.DeltaAvgFanIn,
			point.DeltaAvgFanOut,
			point.ModuleGrowthPct,
			point.AvgCycles,
			point.AvgUnresolved,
			point.WindowHours,
		))
	}

	return []byte(buf.String()), nil
}

func RenderTrendJSON(report history.TrendReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

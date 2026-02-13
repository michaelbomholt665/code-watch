package report

import (
	"circular/internal/data/history"
	"strings"
	"testing"
	"time"
)

func TestRenderTrendTSV(t *testing.T) {
	report := history.TrendReport{
		SchemaVersion: 1,
		Since:         time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC),
		Until:         time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC),
		Window:        "24h0m0s",
		ScanCount:     1,
		Points: []history.TrendPoint{
			{
				Timestamp:       time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC),
				CommitHash:      "abc123",
				ModuleCount:     10,
				FileCount:       15,
				CycleCount:      1,
				UnresolvedCount: 2,
				AvgFanIn:        1.2,
				AvgFanOut:       1.7,
				MaxFanIn:        3,
				MaxFanOut:       4,
				AvgCycles:       1,
				AvgUnresolved:   2,
				WindowHours:     24,
			},
		},
	}

	out, err := RenderTrendTSV(report)
	if err != nil {
		t.Fatalf("render tsv: %v", err)
	}

	body := string(out)
	if !strings.Contains(body, "Timestamp\tCommit\tModules") {
		t.Fatalf("missing header in output: %s", body)
	}
	if !strings.Contains(body, "abc123\t10\t15\t1\t2\t0\t0\t0\t1.20\t1.70\t3\t4") {
		t.Fatalf("missing row values in output: %s", body)
	}
}

func TestRenderTrendJSON(t *testing.T) {
	report := history.TrendReport{
		SchemaVersion: 1,
		ScanCount:     2,
	}

	out, err := RenderTrendJSON(report)
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	if !strings.Contains(string(out), "\"scan_count\": 2") {
		t.Fatalf("missing scan_count in json: %s", string(out))
	}
}

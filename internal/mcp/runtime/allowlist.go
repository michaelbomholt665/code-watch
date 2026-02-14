package runtime

import (
	"circular/internal/core/config"
	"circular/internal/mcp/contracts"
	"strings"
)

type OperationAllowlist struct {
	allowAll bool
	allowed  map[contracts.OperationID]bool
}

func BuildOperationAllowlist(cfg *config.Config) OperationAllowlist {
	if cfg == nil {
		return OperationAllowlist{allowAll: true}
	}
	if strings.TrimSpace(cfg.MCP.ExposedToolName) != "" {
		return OperationAllowlist{allowAll: true}
	}

	entries := cfg.MCP.OperationAllowlist
	if len(entries) == 0 {
		return OperationAllowlist{allowAll: true}
	}

	allowed := make(map[contracts.OperationID]bool)
	for _, entry := range entries {
		id := normalizeOperationAlias(entry)
		if id == "" {
			continue
		}
		allowed[id] = true
	}

	return OperationAllowlist{allowed: allowed}
}

func (o OperationAllowlist) Allows(id contracts.OperationID) bool {
	if o.allowAll {
		return true
	}
	return o.allowed[id]
}

func normalizeOperationAlias(raw string) contracts.OperationID {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "scan.run", "scan_once":
		return contracts.OperationScanRun
	case "secrets.scan":
		return contracts.OperationSecretsScan
	case "secrets.list":
		return contracts.OperationSecretsList
	case "graph.cycles", "detect_cycles":
		return contracts.OperationGraphCycles
	case "query.modules":
		return contracts.OperationQueryModules
	case "query.module_details", "query.module-details":
		return contracts.OperationQueryDetails
	case "query.trace", "trace_import_chain":
		return contracts.OperationQueryTrace
	case "system.sync_outputs", "generate_reports", "graph.sync_diagrams":
		return contracts.OperationGraphSyncDiag
	case "system.sync_config":
		return contracts.OperationSystemSyncCfg
	case "system.generate_config":
		return contracts.OperationSystemGenCfg
	case "system.generate_script":
		return contracts.OperationSystemGenScript
	case "system.select_project":
		return contracts.OperationSystemSelect
	case "system.watch":
		return contracts.OperationSystemWatch
	case "query.trends":
		return contracts.OperationQueryTrends
	case "report.generate_markdown":
		return contracts.OperationReportGenMD
	default:
		return ""
	}
}

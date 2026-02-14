package schema

import "circular/internal/mcp/contracts"

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
	Version     string         `json:"version"`
}

func BuildToolDefinitions() []ToolDefinition {
	operations := []string{
		string(contracts.OperationScanRun),
		string(contracts.OperationSecretsScan),
		string(contracts.OperationSecretsList),
		string(contracts.OperationGraphCycles),
		string(contracts.OperationGraphSyncDiag),
		string(contracts.OperationQueryModules),
		string(contracts.OperationQueryDetails),
		string(contracts.OperationQueryTrace),
		string(contracts.OperationSystemSyncCfg),
		string(contracts.OperationSystemGenCfg),
		string(contracts.OperationSystemGenScript),
		string(contracts.OperationSystemSelect),
		string(contracts.OperationSystemWatch),
		string(contracts.OperationQueryTrends),
		string(contracts.OperationReportGenMD),
	}

	return []ToolDefinition{
		{
			Name:        contracts.ToolNameCircular,
			Description: "Single entry tool for code-watch MCP operations.",
			Version:     contracts.ContractVersion,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "Operation identifier (e.g., scan.run).",
						"enum":        operations,
					},
					"params": map[string]any{
						"type":                 "object",
						"additionalProperties": true,
					},
				},
				"required": []string{"operation"},
			},
		},
	}
}

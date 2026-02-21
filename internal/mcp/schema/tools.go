package schema

import "circular/internal/mcp/contracts"

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
	Version     string         `json:"version"`
}

func BuildToolDefinitions() []ToolDefinition {
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
						"description": "Operation identifier.",
						"enum": []string{
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
						},
					},
					"params": map[string]any{
						"type": "object",
						"description": "Operation-specific parameters.",
						"oneOf": []map[string]any{
							{
								"title": "scan.run",
								"properties": map[string]any{
									"paths": map[string]any{
										"type": "array",
										"items": map[string]any{"type": "string"},
									},
								},
							},
							{
								"title": "graph.cycles",
								"properties": map[string]any{
									"limit": map[string]any{"type": "integer"},
								},
							},
							{
								"title": "query.module",
								"properties": map[string]any{
									"module": map[string]any{"type": "string"},
								},
							},
							// Add more as needed, but this shows the intent
						},
					},
				},
				"required": []string{"operation"},
			},
		},
	}
}

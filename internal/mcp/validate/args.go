package validate

import (
	"circular/internal/mcp/contracts"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	maxPathCount    = 64
	maxFormatCount  = 8
	maxFilterLength = 200
	maxLimitValue   = 5000
	maxTraceDepth   = 100
)

func ValidateToolArgs(tool string, raw map[string]any) (any, error) {
	_, input, err := ParseToolArgs(tool, raw)
	return input, err
}

func ParseToolArgs(tool string, raw map[string]any) (contracts.OperationID, any, error) {
	if strings.TrimSpace(tool) == "" {
		return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "tool name is required"}
	}
	if tool != contracts.ToolNameCircular {
		return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: fmt.Sprintf("unsupported tool: %s", tool)}
	}
	if raw == nil {
		raw = map[string]any{}
	}

	operationRaw, ok := raw["operation"].(string)
	if !ok || strings.TrimSpace(operationRaw) == "" {
		return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "operation is required"}
	}
	operation := contracts.OperationID(strings.TrimSpace(operationRaw))
	if operation == contracts.OperationSystemSyncOut {
		operation = contracts.OperationGraphSyncDiag
	}

	params := map[string]any{}
	if rawParams, ok := raw["params"]; ok && rawParams != nil {
		if typed, ok := rawParams.(map[string]any); ok {
			params = typed
		} else {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "params must be an object"}
		}
	}

	switch operation {
	case contracts.OperationScanRun:
		var input contracts.ScanRunInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Paths = normalizeStrings(input.Paths, maxPathCount)
		return operation, input, nil
	case contracts.OperationGraphCycles:
		var input contracts.GraphCyclesInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		if input.Limit < 0 || input.Limit > maxLimitValue {
			return "", nil, invalidLimitError("limit")
		}
		return operation, input, nil
	case contracts.OperationGraphSyncDiag:
		var input contracts.SystemSyncOutputsInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Formats = normalizeFormats(input.Formats)
		if len(input.Formats) > maxFormatCount {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "too many formats requested"}
		}
		return operation, input, nil
	case contracts.OperationQueryModules:
		var input contracts.QueryModulesInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Filter = strings.TrimSpace(input.Filter)
		if len(input.Filter) > maxFilterLength {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "filter is too long"}
		}
		if input.Limit < 0 || input.Limit > maxLimitValue {
			return "", nil, invalidLimitError("limit")
		}
		return operation, input, nil
	case contracts.OperationQueryDetails:
		var input contracts.QueryModuleDetailsInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Module = strings.TrimSpace(input.Module)
		if input.Module == "" {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "module is required"}
		}
		return operation, input, nil
	case contracts.OperationQueryTrace:
		var input contracts.QueryTraceInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.From = strings.TrimSpace(input.From)
		input.To = strings.TrimSpace(input.To)
		if input.From == "" || input.To == "" {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "from_module and to_module are required"}
		}
		if input.MaxDepth < 0 || input.MaxDepth > maxTraceDepth {
			return "", nil, invalidLimitError("max_depth")
		}
		return operation, input, nil
	case contracts.OperationSystemSyncOut:
		var input contracts.SystemSyncOutputsInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Formats = normalizeFormats(input.Formats)
		if len(input.Formats) > maxFormatCount {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "too many formats requested"}
		}
		return operation, input, nil
	case contracts.OperationSystemSyncCfg:
		var input contracts.SystemSyncConfigInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		return operation, input, nil
	case contracts.OperationSystemGenCfg:
		var input contracts.SystemGenerateConfigInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		return operation, input, nil
	case contracts.OperationSystemGenScript:
		var input contracts.SystemGenerateScriptInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		return operation, input, nil
	case contracts.OperationSystemSelect:
		var input contracts.SystemSelectProjectInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Name = strings.TrimSpace(input.Name)
		if input.Name == "" {
			return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "name is required"}
		}
		return operation, input, nil
	case contracts.OperationQueryTrends:
		var input contracts.QueryTrendsInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		input.Since = strings.TrimSpace(input.Since)
		if input.Limit < 0 || input.Limit > maxLimitValue {
			return "", nil, invalidLimitError("limit")
		}
		return operation, input, nil
	case contracts.OperationSystemWatch:
		var input contracts.SystemWatchInput
		if err := decodeParams(params, &input); err != nil {
			return "", nil, err
		}
		return operation, input, nil
	default:
		return "", nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: fmt.Sprintf("unsupported operation: %s", operation)}
	}
}

func decodeParams(params map[string]any, out any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "invalid params encoding"}
	}
	if err := json.Unmarshal(data, out); err != nil {
		return contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "invalid params", Details: map[string]any{"error": err.Error()}}
	}
	return nil
}

func normalizeStrings(values []string, maxCount int) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if seen[trimmed] {
			continue
		}
		if maxCount > 0 && len(out) >= maxCount {
			break
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func normalizeFormats(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.ToLower(strings.TrimSpace(v))
		if trimmed == "" {
			continue
		}
		switch trimmed {
		case "dot", "tsv", "mermaid", "plantuml":
			if !seen[trimmed] {
				seen[trimmed] = true
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func invalidLimitError(field string) error {
	return contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: fmt.Sprintf("%s is out of range", field)}
}

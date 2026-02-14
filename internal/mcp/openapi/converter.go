package openapi

import (
	"circular/internal/mcp/contracts"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func Convert(spec *openapi3.T) ([]contracts.OperationDescriptor, error) {
	if spec == nil {
		return nil, fmt.Errorf("openapi spec is nil")
	}
	if spec.Paths == nil {
		return nil, fmt.Errorf("openapi spec has no paths")
	}

	pathMap := spec.Paths.Map()
	if len(pathMap) == 0 {
		return nil, fmt.Errorf("openapi spec has no operations")
	}

	descriptors := make([]contracts.OperationDescriptor, 0, len(pathMap))
	seen := make(map[contracts.OperationID]bool)
	for path, pathItem := range pathMap {
		if pathItem == nil {
			continue
		}
		for method, operation := range pathItem.Operations() {
			if operation == nil {
				continue
			}
			id := contracts.OperationID(strings.TrimSpace(operation.OperationID))
			if id == "" {
				return nil, fmt.Errorf("operation %s %s is missing operationId", strings.ToUpper(method), path)
			}
			if !isValidOperationID(id) {
				return nil, fmt.Errorf("operationId %q is invalid for %s %s", id, strings.ToUpper(method), path)
			}
			if seen[id] {
				return nil, fmt.Errorf("duplicate operationId %q in openapi spec", id)
			}
			seen[id] = true

			inputSchema, err := requestSchema(operation)
			if err != nil {
				return nil, fmt.Errorf("operation %s (%s %s): %w", id, strings.ToUpper(method), path, err)
			}

			descriptors = append(descriptors, contracts.OperationDescriptor{
				ID:          id,
				Summary:     strings.TrimSpace(operation.Summary),
				Description: strings.TrimSpace(operation.Description),
				InputSchema: inputSchema,
			})
		}
	}

	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].ID < descriptors[j].ID
	})
	if len(descriptors) == 0 {
		return nil, fmt.Errorf("openapi spec produced zero operation descriptors")
	}
	return descriptors, nil
}

func requestSchema(op *openapi3.Operation) (map[string]any, error) {
	if op.RequestBody == nil {
		return defaultObjectSchema(), nil
	}
	if op.RequestBody.Value == nil {
		return nil, fmt.Errorf("requestBody is empty")
	}
	content := op.RequestBody.Value.Content.Get("application/json")
	if content == nil || content.Schema == nil {
		return nil, fmt.Errorf("requestBody must define application/json schema")
	}
	return schemaRefToMap(content.Schema)
}

func schemaRefToMap(ref *openapi3.SchemaRef) (map[string]any, error) {
	if ref == nil {
		return nil, fmt.Errorf("schema is nil")
	}
	if strings.TrimSpace(ref.Ref) != "" {
		return map[string]any{"$ref": ref.Ref}, nil
	}
	if ref.Value == nil {
		return nil, fmt.Errorf("schema value is nil")
	}

	data, err := json.Marshal(ref.Value)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(data, &schemaMap); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}

	if schemaType, ok := schemaMap["type"].(string); ok {
		schemaType = strings.TrimSpace(schemaType)
		if schemaType != "" && schemaType != "object" {
			return nil, fmt.Errorf("unsupported schema type %q (only object schemas are supported)", schemaType)
		}
	}
	if _, ok := schemaMap["type"]; !ok {
		schemaMap["type"] = "object"
	}
	return schemaMap, nil
}

func defaultObjectSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
	}
}

func isValidOperationID(id contracts.OperationID) bool {
	value := string(id)
	if value == "" {
		return false
	}
	partLen := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch >= 'a' && ch <= 'z':
			partLen++
		case ch >= '0' && ch <= '9':
			partLen++
		case ch == '_' && partLen > 0:
			partLen++
		case ch == '.' && partLen > 0:
			partLen = 0
		default:
			return false
		}
	}
	return partLen > 0
}

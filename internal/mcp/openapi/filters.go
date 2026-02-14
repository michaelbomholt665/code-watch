package openapi

import (
	"circular/internal/mcp/contracts"
	"sort"
	"strings"
)

func ApplyAllowlist(ops []contracts.OperationDescriptor, allowlist []string) []contracts.OperationDescriptor {
	if len(ops) == 0 {
		return nil
	}
	if len(allowlist) == 0 {
		out := make([]contracts.OperationDescriptor, len(ops))
		copy(out, ops)
		sortDescriptors(out)
		return out
	}

	allowed := make(map[contracts.OperationID]bool, len(allowlist))
	for _, raw := range allowlist {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		if normalized == "" {
			continue
		}
		allowed[contracts.OperationID(normalized)] = true
	}

	filtered := make([]contracts.OperationDescriptor, 0, len(ops))
	for _, op := range ops {
		if allowed[op.ID] {
			filtered = append(filtered, op)
		}
	}
	sortDescriptors(filtered)
	return filtered
}

func sortDescriptors(ops []contracts.OperationDescriptor) {
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].ID < ops[j].ID
	})
}

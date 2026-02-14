package graph

import (
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"context"
)

func HandleCycles(ctx context.Context, a *adapters.Adapter, in contracts.GraphCyclesInput, maxItems int) (contracts.GraphCyclesOutput, error) {
	limit := in.Limit
	if limit <= 0 || (maxItems > 0 && limit > maxItems) {
		limit = maxItems
	}
	return a.Cycles(ctx, limit)
}

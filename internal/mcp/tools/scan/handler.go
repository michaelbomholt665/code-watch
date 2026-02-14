package scan

import (
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"context"
)

func HandleRun(ctx context.Context, a *adapters.Adapter, in contracts.ScanRunInput) (contracts.ScanRunOutput, error) {
	return a.RunScan(ctx, in)
}

package secrets

import (
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"context"
)

func HandleScan(ctx context.Context, a *adapters.Adapter, in contracts.SecretsScanInput, maxItems int) (contracts.SecretsScanOutput, error) {
	out, err := a.ScanSecrets(ctx, in)
	if err != nil {
		return contracts.SecretsScanOutput{}, err
	}
	if maxItems > 0 && len(out.Findings) > maxItems {
		out.Findings = out.Findings[:maxItems]
	}
	return out, nil
}

func HandleList(ctx context.Context, a *adapters.Adapter, in contracts.SecretsListInput, maxItems int) (contracts.SecretsListOutput, error) {
	limit := in.Limit
	if limit <= 0 || (maxItems > 0 && limit > maxItems) {
		limit = maxItems
	}
	return a.ListSecrets(ctx, limit)
}

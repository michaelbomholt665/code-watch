package query

import (
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"context"
	"fmt"
	"strings"
	"time"
)

func HandleModules(ctx context.Context, a *adapters.Adapter, in contracts.QueryModulesInput, maxItems int) (contracts.QueryModulesOutput, error) {
	limit := normalizeLimit(in.Limit, maxItems)
	return a.ListModules(ctx, in.Filter, limit)
}

func HandleModuleDetails(ctx context.Context, a *adapters.Adapter, in contracts.QueryModuleDetailsInput, maxItems int) (contracts.QueryModuleDetailsOutput, error) {
	out, err := a.ModuleDetails(ctx, in.Module)
	if err != nil {
		return contracts.QueryModuleDetailsOutput{}, err
	}

	if maxItems > 0 {
		out.Module.Files = limitStrings(out.Module.Files, maxItems)
		out.Module.ExportedSymbols = limitStrings(out.Module.ExportedSymbols, maxItems)
		out.Module.ReverseDependencies = limitStrings(out.Module.ReverseDependencies, maxItems)
		out.Module.Dependencies = limitDependencies(out.Module.Dependencies, maxItems)
	}

	return out, nil
}

func HandleTrace(ctx context.Context, a *adapters.Adapter, in contracts.QueryTraceInput) (contracts.QueryTraceOutput, error) {
	return a.Trace(ctx, in.From, in.To, in.MaxDepth)
}

func HandleTrends(ctx context.Context, a *adapters.Adapter, in contracts.QueryTrendsInput, maxItems int) (contracts.QueryTrendsOutput, error) {
	since, err := parseSince(in.Since)
	if err != nil {
		return contracts.QueryTrendsOutput{}, err
	}
	limit := normalizeLimit(in.Limit, maxItems)
	return a.TrendSlice(ctx, since, limit)
}

func parseSince(value string) (time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, nil
	}

	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("since must be RFC3339 or YYYY-MM-DD, got %q", value)
}

func normalizeLimit(value, maxItems int) int {
	if value <= 0 {
		return maxItems
	}
	if maxItems > 0 && value > maxItems {
		return maxItems
	}
	return value
}

func limitStrings(values []string, maxItems int) []string {
	if maxItems <= 0 || len(values) <= maxItems {
		return values
	}
	return values[:maxItems]
}

func limitDependencies(values []contracts.DependencyEdge, maxItems int) []contracts.DependencyEdge {
	if maxItems <= 0 || len(values) <= maxItems {
		return values
	}
	return values[:maxItems]
}

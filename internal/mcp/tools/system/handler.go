package system

import (
	"circular/internal/mcp/contracts"
	"context"
)

type Manager interface {
	SyncOutputs(ctx context.Context, formats []string) ([]string, error)
	SyncConfig(ctx context.Context) (string, error)
	GenerateConfig(ctx context.Context) (contracts.SystemGenerateConfigOutput, error)
	GenerateScript(ctx context.Context) (contracts.SystemGenerateScriptOutput, error)
	SelectProject(ctx context.Context, name string) (contracts.ProjectSummary, error)
	StartWatch(ctx context.Context) (contracts.SystemWatchOutput, error)
}

func HandleSyncOutputs(ctx context.Context, mgr Manager, allowMutations bool, in contracts.SystemSyncOutputsInput) (contracts.SystemSyncOutputsOutput, error) {
	if !allowMutations {
		return contracts.SystemSyncOutputsOutput{}, contracts.ToolError{
			Code:    contracts.ErrorUnavailable,
			Message: "mcp.allow_mutations=false blocks system.sync_outputs",
		}
	}

	written, err := mgr.SyncOutputs(ctx, in.Formats)
	if err != nil {
		return contracts.SystemSyncOutputsOutput{}, err
	}

	return contracts.SystemSyncOutputsOutput{Written: written}, nil
}

func HandleSyncConfig(ctx context.Context, mgr Manager, allowMutations bool) (contracts.SystemSyncConfigOutput, error) {
	if !allowMutations {
		return contracts.SystemSyncConfigOutput{}, contracts.ToolError{
			Code:    contracts.ErrorUnavailable,
			Message: "mcp.allow_mutations=false blocks system.sync_config",
		}
	}

	target, err := mgr.SyncConfig(ctx)
	if err != nil {
		return contracts.SystemSyncConfigOutput{}, err
	}

	return contracts.SystemSyncConfigOutput{
		Synced: target != "",
		Target: target,
	}, nil
}

func HandleGenerateConfig(ctx context.Context, mgr Manager, allowMutations bool) (contracts.SystemGenerateConfigOutput, error) {
	if !allowMutations {
		return contracts.SystemGenerateConfigOutput{}, contracts.ToolError{
			Code:    contracts.ErrorUnavailable,
			Message: "mcp.allow_mutations=false blocks system.generate_config",
		}
	}

	out, err := mgr.GenerateConfig(ctx)
	if err != nil {
		return contracts.SystemGenerateConfigOutput{}, err
	}
	return out, nil
}

func HandleGenerateScript(ctx context.Context, mgr Manager, allowMutations bool) (contracts.SystemGenerateScriptOutput, error) {
	if !allowMutations {
		return contracts.SystemGenerateScriptOutput{}, contracts.ToolError{
			Code:    contracts.ErrorUnavailable,
			Message: "mcp.allow_mutations=false blocks system.generate_script",
		}
	}

	out, err := mgr.GenerateScript(ctx)
	if err != nil {
		return contracts.SystemGenerateScriptOutput{}, err
	}
	return out, nil
}

func HandleSelectProject(ctx context.Context, mgr Manager, allowMutations bool, in contracts.SystemSelectProjectInput) (contracts.SystemSelectProjectOutput, error) {
	if !allowMutations {
		return contracts.SystemSelectProjectOutput{}, contracts.ToolError{
			Code:    contracts.ErrorUnavailable,
			Message: "mcp.allow_mutations=false blocks system.select_project",
		}
	}

	project, err := mgr.SelectProject(ctx, in.Name)
	if err != nil {
		return contracts.SystemSelectProjectOutput{}, err
	}

	return contracts.SystemSelectProjectOutput{Project: project}, nil
}

func HandleWatch(ctx context.Context, mgr Manager, allowMutations bool) (contracts.SystemWatchOutput, error) {
	if !allowMutations {
		return contracts.SystemWatchOutput{}, contracts.ToolError{
			Code:    contracts.ErrorUnavailable,
			Message: "mcp.allow_mutations=false blocks system.watch",
		}
	}

	out, err := mgr.StartWatch(ctx)
	if err != nil {
		return contracts.SystemWatchOutput{}, err
	}
	return out, nil
}

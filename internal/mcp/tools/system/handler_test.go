package system

import (
	"circular/internal/mcp/contracts"
	"context"
	"errors"
	"testing"
)

func TestHandleSyncOutputs(t *testing.T) {
	manager := &fakeManager{
		syncOutputsFn: func(_ context.Context, formats []string) ([]string, error) {
			return append([]string(nil), formats...), nil
		},
	}

	_, err := HandleSyncOutputs(context.Background(), manager, false, contracts.SystemSyncOutputsInput{Formats: []string{"dot"}})
	if err == nil {
		t.Fatal("expected mutation policy error")
	}

	out, err := HandleSyncOutputs(context.Background(), manager, true, contracts.SystemSyncOutputsInput{Formats: []string{"dot", "tsv"}})
	if err != nil {
		t.Fatalf("handle sync outputs: %v", err)
	}
	if len(out.Written) != 2 {
		t.Fatalf("expected 2 written outputs, got %d", len(out.Written))
	}
}

func TestHandleSelectProjectNamespaceIsolation(t *testing.T) {
	manager := &fakeManager{
		selectProjectFn: func(_ context.Context, name string) (contracts.ProjectSummary, error) {
			if name == "" {
				return contracts.ProjectSummary{}, errors.New("name is required")
			}
			return contracts.ProjectSummary{
				Name:        name,
				DBNamespace: "ns-" + name,
				Key:         name,
			}, nil
		},
	}

	out, err := HandleSelectProject(context.Background(), manager, true, contracts.SystemSelectProjectInput{Name: "repo-a"})
	if err != nil {
		t.Fatalf("handle select project: %v", err)
	}
	if out.Project.DBNamespace != "ns-repo-a" {
		t.Fatalf("unexpected project namespace: %+v", out.Project)
	}
}

func TestHandleGenerateConfig(t *testing.T) {
	manager := &fakeManager{
		generateConfigFn: func(_ context.Context) (contracts.SystemGenerateConfigOutput, error) {
			return contracts.SystemGenerateConfigOutput{Generated: true, Target: "circular.toml"}, nil
		},
	}

	_, err := HandleGenerateConfig(context.Background(), manager, false)
	if err == nil {
		t.Fatal("expected mutation policy error")
	}

	out, err := HandleGenerateConfig(context.Background(), manager, true)
	if err != nil {
		t.Fatalf("handle generate config: %v", err)
	}
	if !out.Generated || out.Target != "circular.toml" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestHandleGenerateScript(t *testing.T) {
	manager := &fakeManager{
		generateScriptFn: func(_ context.Context) (contracts.SystemGenerateScriptOutput, error) {
			return contracts.SystemGenerateScriptOutput{Generated: true, Target: "circular-mcp"}, nil
		},
	}

	_, err := HandleGenerateScript(context.Background(), manager, false)
	if err == nil {
		t.Fatal("expected mutation policy error")
	}

	out, err := HandleGenerateScript(context.Background(), manager, true)
	if err != nil {
		t.Fatalf("handle generate script: %v", err)
	}
	if !out.Generated || out.Target != "circular-mcp" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestHandleWatch(t *testing.T) {
	manager := &fakeManager{
		startWatchFn: func(_ context.Context) (contracts.SystemWatchOutput, error) {
			return contracts.SystemWatchOutput{Status: "watching"}, nil
		},
	}

	_, err := HandleWatch(context.Background(), manager, false)
	if err == nil {
		t.Fatal("expected mutation policy error")
	}

	out, err := HandleWatch(context.Background(), manager, true)
	if err != nil {
		t.Fatalf("handle watch: %v", err)
	}
	if out.Status != "watching" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

type fakeManager struct {
	syncOutputsFn    func(ctx context.Context, formats []string) ([]string, error)
	syncConfigFn     func(ctx context.Context) (string, error)
	generateConfigFn func(ctx context.Context) (contracts.SystemGenerateConfigOutput, error)
	generateScriptFn func(ctx context.Context) (contracts.SystemGenerateScriptOutput, error)
	selectProjectFn  func(ctx context.Context, name string) (contracts.ProjectSummary, error)
	startWatchFn     func(ctx context.Context) (contracts.SystemWatchOutput, error)
}

func (f *fakeManager) SyncOutputs(ctx context.Context, formats []string) ([]string, error) {
	if f.syncOutputsFn != nil {
		return f.syncOutputsFn(ctx, formats)
	}
	return nil, nil
}

func (f *fakeManager) SyncConfig(ctx context.Context) (string, error) {
	if f.syncConfigFn != nil {
		return f.syncConfigFn(ctx)
	}
	return "", nil
}

func (f *fakeManager) GenerateConfig(ctx context.Context) (contracts.SystemGenerateConfigOutput, error) {
	if f.generateConfigFn != nil {
		return f.generateConfigFn(ctx)
	}
	return contracts.SystemGenerateConfigOutput{}, nil
}

func (f *fakeManager) GenerateScript(ctx context.Context) (contracts.SystemGenerateScriptOutput, error) {
	if f.generateScriptFn != nil {
		return f.generateScriptFn(ctx)
	}
	return contracts.SystemGenerateScriptOutput{}, nil
}

func (f *fakeManager) SelectProject(ctx context.Context, name string) (contracts.ProjectSummary, error) {
	if f.selectProjectFn != nil {
		return f.selectProjectFn(ctx, name)
	}
	return contracts.ProjectSummary{}, nil
}

func (f *fakeManager) StartWatch(ctx context.Context) (contracts.SystemWatchOutput, error) {
	if f.startWatchFn != nil {
		return f.startWatchFn(ctx)
	}
	return contracts.SystemWatchOutput{}, nil
}

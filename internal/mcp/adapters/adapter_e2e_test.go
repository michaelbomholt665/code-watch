package adapters_test

import (
	"context"
	"os"
	"testing"
	"time"

	"circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"circular/internal/mcp/runtime"
	"circular/internal/mcp/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPEndToEnd(t *testing.T) {
	// 1. Setup environment
	tmpDir := t.TempDir()
	err := os.WriteFile(tmpDir+"/main.go", []byte("package main\nfunc main() {}"), 0644)
	require.NoError(t, err)

	cfg := config.DefaultConfig()
	cfg.Paths.ProjectRoot = tmpDir
	cfg.WatchPaths = []string{tmpDir}
	cfg.Projects = config.Projects{
		Entries: []config.ProjectEntry{
			{Name: "test", Root: tmpDir},
		},
	}

	// 2. Initialize App and Adapter
	appInstance, err := app.New(cfg)
	require.NoError(t, err)

	toolAdapter := adapters.NewAdapter(appInstance.AnalysisService(), nil, "")
	mockTransport := transport.NewMockAdapter()

	// 3. Build Server
	server, err := runtime.New(
		cfg,
		runtime.Dependencies{
			Analysis: appInstance.AnalysisService(),
		},
		nil,           // registry
		mockTransport, // transport
		runtime.ProjectContext{Name: "test", Root: tmpDir},
		"circular",
		runtime.BuildOperationAllowlist(nil),
		toolAdapter,
		nil, // history
	)
	require.NoError(t, err)

	// 4. Start Server in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// 5. Simulate Tool Call: scan.run
	res, err := mockTransport.CallJSON("circular", map[string]any{
		"operation": "scan.run",
		"arguments": map[string]any{},
	})
	require.NoError(t, err)
	assert.NotNil(t, res)

	// Verify result shape
	resMap, ok := res.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, contracts.OperationScanRun, resMap["operation"])
	
	result, ok := resMap["result"].(contracts.ScanRunOutput)
	require.True(t, ok)
	assert.Equal(t, 1, result.FilesScanned)

	// 6. Stop server
	cancel()
	err = <-serverErr
	if err != nil && err != context.Canceled {
		t.Errorf("server exited with error: %v", err)
	}
}

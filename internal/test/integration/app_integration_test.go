package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"circular/internal/core/app"
	"circular/internal/core/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestFiles(t *testing.T, tmpDir string) {
	// Simple Go module
	goMod := `module test-project
go 1.24`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	mainGo := `package main
import "fmt"
import "./pkg1"
func main() {
	fmt.Println(pkg1.Hello())
}`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(tmpDir, "pkg1"), 0755)
	require.NoError(t, err)

	pkg1Go := `package pkg1
func Hello() string {
	return "Hello"
}`
	err = os.WriteFile(filepath.Join(tmpDir, "pkg1/pkg1.go"), []byte(pkg1Go), 0644)
	require.NoError(t, err)
}

func TestFullPipelineIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFiles(t, tmpDir)

	cfg := config.DefaultConfig()
	cfg.Paths.ProjectRoot = tmpDir
	cfg.WatchPaths = []string{tmpDir}
	cfg.Projects = config.Projects{
		Entries: []config.ProjectEntry{
			{Name: "test-project", Root: tmpDir, DBNamespace: "test"},
		},
	}

	appInstance, err := app.New(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = appInstance.InitialScan(ctx)
	require.NoError(t, err)

	// Verify Graph State
	modules := appInstance.Graph.Modules()
	assert.NotEmpty(t, modules)

	// Verify that we can find our modules
	foundMain := false
	foundPkg1 := false
	for _, m := range modules {
		if m.Name == "test-project" {
			foundMain = true
		}
		if m.Name == "test-project/pkg1" {
			foundPkg1 = true
		}
	}
	assert.True(t, foundMain, "Should have found test-project (main) module")
	assert.True(t, foundPkg1, "Should have found test-project/pkg1 module")

	// Verify Resolver - checking for cycles (should be none)
	cycles := appInstance.Graph.DetectCycles()
	assert.Empty(t, cycles)
}

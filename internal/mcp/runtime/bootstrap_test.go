package runtime

import (
	"circular/internal/core/config"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOpenAPIOperations_NoSpec(t *testing.T) {
	cfg := &config.Config{}
	if err := loadOpenAPIOperations(cfg); err != nil {
		t.Fatalf("expected nil error without spec, got %v", err)
	}
}

func TestLoadOpenAPIOperations_WithSpec(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(`
openapi: 3.0.3
info:
  title: code-watch
  version: "1.0"
paths:
  /modules:
    get:
      operationId: query.modules
      responses:
        "200":
          description: ok
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		MCP: config.MCP{
			OpenAPISpecPath:    specPath,
			OperationAllowlist: []string{"query.modules"},
		},
	}
	if err := loadOpenAPIOperations(cfg); err != nil {
		t.Fatalf("load openapi operations: %v", err)
	}
}

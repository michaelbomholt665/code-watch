package openapi

import (
	"circular/internal/mcp/contracts"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestConvert_OpenAPIToOperations(t *testing.T) {
	spec := mustLoadSpecFromData(t, []byte(`
openapi: 3.0.3
info:
  title: code-watch
  version: "1.0"
paths:
  /modules:
    get:
      operationId: query.modules
      summary: List modules
      responses:
        "200":
          description: ok
  /trace:
    post:
      operationId: query.trace
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                from_module:
                  type: string
                to_module:
                  type: string
      responses:
        "200":
          description: ok
`))

	ops, err := Convert(spec)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}
	if ops[0].ID != contracts.OperationQueryModules || ops[1].ID != contracts.OperationQueryTrace {
		t.Fatalf("unexpected operation order: %+v", ops)
	}
	if ops[1].InputSchema["type"] != "object" {
		t.Fatalf("expected object schema, got %+v", ops[1].InputSchema)
	}
}

func TestConvert_InvalidSchema(t *testing.T) {
	spec := mustLoadSpecFromData(t, []byte(`
openapi: 3.0.3
info:
  title: code-watch
  version: "1.0"
paths:
  /trace:
    post:
      operationId: query.trace
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: array
              items:
                type: string
      responses:
        "200":
          description: ok
`))

	_, err := Convert(spec)
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !strings.Contains(err.Error(), "unsupported schema type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvert_MissingOperationID(t *testing.T) {
	spec := mustLoadSpecFromData(t, []byte(`
openapi: 3.0.3
info:
  title: code-watch
  version: "1.0"
paths:
  /modules:
    get:
      summary: List modules
      responses:
        "200":
          description: ok
`))

	_, err := Convert(spec)
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !strings.Contains(err.Error(), "missing operationId") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyAllowlist(t *testing.T) {
	ops := []contracts.OperationDescriptor{
		{ID: contracts.OperationQueryTrace},
		{ID: contracts.OperationScanRun},
		{ID: contracts.OperationQueryModules},
	}

	filtered := ApplyAllowlist(ops, []string{"query.modules", "scan.run"})
	ids := make([]contracts.OperationID, 0, len(filtered))
	for _, op := range filtered {
		ids = append(ids, op.ID)
	}

	expected := []contracts.OperationID{contracts.OperationQueryModules, contracts.OperationScanRun}
	if !reflect.DeepEqual(ids, expected) {
		t.Fatalf("expected ids %v, got %v", expected, ids)
	}
}

func TestLoadSpec_Path(t *testing.T) {
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

	spec, err := LoadSpec(specPath)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if spec == nil {
		t.Fatal("expected loaded spec")
	}
}

func mustLoadSpecFromData(t *testing.T, data []byte) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData(data)
	if err != nil {
		t.Fatalf("load spec from data: %v", err)
	}
	if err := spec.Validate(loader.Context); err != nil {
		t.Fatalf("validate spec: %v", err)
	}
	return spec
}

package secrets

import (
	"circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"context"
	"testing"
)

func TestHandleList(t *testing.T) {
	adapter := testSecretsAdapter()

	out, err := HandleList(context.Background(), adapter, contracts.SecretsListInput{Limit: 10}, 1)
	if err != nil {
		t.Fatalf("handle list: %v", err)
	}
	if out.SecretCount != 2 {
		t.Fatalf("expected secret_count=2, got %d", out.SecretCount)
	}
	if len(out.Findings) != 1 {
		t.Fatalf("expected bounded findings=1, got %d", len(out.Findings))
	}
	if out.Findings[0].ValueMasked != "AKIA...CDEF" {
		t.Fatalf("expected masked finding value, got %q", out.Findings[0].ValueMasked)
	}
}

func TestHandleScan_RespectsMaxItems(t *testing.T) {
	adapter := testSecretsAdapter()

	out, err := HandleScan(context.Background(), adapter, contracts.SecretsScanInput{}, 1)
	if err != nil {
		t.Fatalf("handle scan: %v", err)
	}
	if out.SecretCount != 2 {
		t.Fatalf("expected secret_count=2, got %d", out.SecretCount)
	}
	if len(out.Findings) != 1 {
		t.Fatalf("expected bounded findings=1, got %d", len(out.Findings))
	}
}

func testSecretsAdapter() *adapters.Adapter {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "app/a",
		Secrets: []parser.Secret{
			{
				Kind:     "aws-access-key-id",
				Severity: "high",
				Value:    "AKIA1234567890ABCDEF",
				Location: parser.Location{File: "a.go", Line: 1, Column: 1},
			},
			{
				Kind:     "high-entropy-string",
				Severity: "low",
				Value:    "qwerty1234567890asdf",
				Location: parser.Location{File: "a.go", Line: 2, Column: 1},
			},
		},
	})
	appInstance := &app.App{
		Config: &config.Config{},
		Graph:  g,
	}
	return adapters.NewAdapter(appInstance.AnalysisService(), nil, "default")
}

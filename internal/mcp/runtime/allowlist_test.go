package runtime

import (
	"circular/internal/core/config"
	"circular/internal/mcp/contracts"
	"testing"
)

func TestBuildOperationAllowlist_Aliases(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCP{
			OperationAllowlist: []string{"scan_once", "detect_cycles", "graph.sync_diagrams", "system.generate_config", "system.generate_script", "system.watch", "query.modules"},
		},
	}
	allowlist := BuildOperationAllowlist(cfg)
	if !allowlist.Allows(contracts.OperationScanRun) {
		t.Fatalf("expected scan.run allowed")
	}
	if !allowlist.Allows(contracts.OperationGraphCycles) {
		t.Fatalf("expected graph.cycles allowed")
	}
	if !allowlist.Allows(contracts.OperationQueryModules) {
		t.Fatalf("expected query.modules allowed")
	}
	if !allowlist.Allows(contracts.OperationGraphSyncDiag) {
		t.Fatalf("expected graph.sync_diagrams allowed")
	}
	if !allowlist.Allows(contracts.OperationSystemGenCfg) {
		t.Fatalf("expected system.generate_config allowed")
	}
	if !allowlist.Allows(contracts.OperationSystemGenScript) {
		t.Fatalf("expected system.generate_script allowed")
	}
	if !allowlist.Allows(contracts.OperationSystemWatch) {
		t.Fatalf("expected system.watch allowed")
	}
	if allowlist.Allows(contracts.OperationQueryTrace) {
		t.Fatalf("did not expect query.trace allowed")
	}
}

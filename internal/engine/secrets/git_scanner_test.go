// # internal/engine/secrets/git_scanner_test.go
package secrets

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGitAvailable(t *testing.T) {
	// This is informational; we do not fail the test if git is absent.
	available := IsGitAvailable()
	t.Logf("IsGitAvailable() = %v", available)
}

func TestScanGitHistory_ZeroDepth(t *testing.T) {
	d, err := NewDetector(Config{})
	if err != nil {
		t.Fatal(err)
	}
	findings, err := ScanGitHistory(".", 0, d)
	if err != nil {
		t.Fatalf("depth=0 should not error, got: %v", err)
	}
	if findings != nil {
		t.Errorf("depth=0 should return nil findings, got %d", len(findings))
	}
}

func TestScanGitHistory_NilDetector(t *testing.T) {
	_, err := ScanGitHistory(".", 5, nil)
	if err == nil {
		t.Error("expected error when detector is nil")
	}
}

func TestScanGitHistory_DepthCapIsApplied(t *testing.T) {
	// We cannot easily test the cap without a real repo, but we verify
	// that cap logic doesn't panic when depth > max.
	if !IsGitAvailable() {
		t.Skip("git not available")
	}
	d, err := NewDetector(Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Use the project root itself (must have .git).
	wd, _ := os.Getwd()
	// Walk up to find .git
	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Skip("no git repo found in parent directories")
		}
		root = parent
	}
	// A depth of 2000 should be silently capped at 1000; this just verifies no panic.
	_, err = ScanGitHistory(root, 2000, d)
	if err != nil {
		// Git errors (e.g. empty repo) are acceptable; what matters is no panic.
		t.Logf("ScanGitHistory returned error (acceptable): %v", err)
	}
}

// TestScanGitHistory_FindsDeletedSecret creates a temporary git repo,
// commits a file containing a fake AWS key, removes it in the next commit,
// then verifies that ScanGitHistory detects the key in history.
func TestScanGitHistory_FindsDeletedSecret(t *testing.T) {
	if !IsGitAvailable() {
		t.Skip("git not available")
	}

	dir := t.TempDir()

	// Initialise the repo with a minimal config.
	mustRun(t, dir, "git", "init", "-b", "main")
	mustRun(t, dir, "git", "config", "user.email", "test@example.com")
	mustRun(t, dir, "git", "config", "user.name", "Test")

	// Commit 1: add a file that contains a recognisable GitHub PAT pattern
	// (avoids the 'EXAMPLE' substring which is on the ignore-list).
	secretFile := filepath.Join(dir, "creds.env")
	if err := os.WriteFile(secretFile, []byte("GH_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "add creds")

	// Commit 2: remove the secret file.
	if err := os.Remove(secretFile); err != nil {
		t.Fatal(err)
	}
	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "remove creds")

	// Now scan history – the secret should appear even though it's absent in HEAD.
	d, err := NewDetector(Config{})
	if err != nil {
		t.Fatal(err)
	}
	findings, err := ScanGitHistory(dir, 10, d)
	if err != nil {
		t.Fatalf("ScanGitHistory error: %v", err)
	}
	if len(findings) == 0 {
		t.Error("expected at least one finding in git history, got 0")
	}
	found := false
	for _, f := range findings {
		if strings.Contains(f.Location.File, "git:history:") && f.Kind == "github-pat" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected github-pat finding in history paths, got: %+v", findings)
	}
}

func TestParseHunkNewStart(t *testing.T) {
	cases := []struct {
		header string
		want   int
	}{
		{"@@ -0,0 +1,10 @@", 1},
		{"@@ -5,3 +7,4 @@ func foo() {", 7},
		{"@@ -0,0 +1 @@", 1},
	}
	for _, tc := range cases {
		got := parseHunkNewStart(tc.header)
		if got != tc.want {
			t.Errorf("parseHunkNewStart(%q) = %d, want %d", tc.header, got, tc.want)
		}
	}
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %s %v failed: %v\n%s", name, args, err, out)
	}
}

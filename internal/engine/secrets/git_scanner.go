// # internal/engine/secrets/git_scanner.go
package secrets

import (
	"bufio"
	"bytes"
	"circular/internal/engine/parser"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// DefaultGitHistoryDepth is how many commits to scan when depth is not specified.
	DefaultGitHistoryDepth = 50
	// MaxGitHistoryDepth caps the scan to prevent accidental scans of massive repos.
	MaxGitHistoryDepth = 1000
)

// IsGitAvailable reports whether the `git` binary is accessible via PATH.
func IsGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// ScanGitHistory runs `git log -p` in repoPath and scans every added hunk line
// through detector for secrets. depth controls how many commits to inspect.
// depth 0 returns nil immediately. depth > MaxGitHistoryDepth is silently capped.
//
// Findings are attributed with a synthetic path:
//
//	git:history:<short-commit>:<file>
//
// That path is not a real filesystem path; it signals that the secret was
// removed from the working tree but still exists in version history.
func ScanGitHistory(repoPath string, depth int, detector *Detector) ([]parser.Secret, error) {
	if depth <= 0 {
		return nil, nil
	}
	if depth > MaxGitHistoryDepth {
		depth = MaxGitHistoryDepth
	}
	if detector == nil {
		return nil, fmt.Errorf("secrets: detector must not be nil")
	}

	// Run: git log -p --format=format:%H -n <depth> -- <repoPath>
	// %H emits the full commit hash; an empty format line separates commits.
	args := []string{
		"-C", repoPath,
		"log", "-p",
		"--format=format:COMMIT:%H",
		"--no-color",
		"-n", strconv.Itoa(depth),
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git log failed: %w\n%s", err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return parseGitDiff(out, detector, repoPath)
}

// parseGitDiff extracts added hunk lines from unified diff output produced by
// `git log -p` and passes them through detector for secrets.
// It tracks the current commit hash and file path for attribution.
func parseGitDiff(data []byte, detector *Detector, repoPath string) ([]parser.Secret, error) {
	var all []parser.Secret
	seen := make(map[string]bool)

	var (
		currentCommit string
		currentFile   string
		hunkLines     []string
		hunkStartLine int
		lineCounter   int
	)

	flushHunk := func() {
		if len(hunkLines) == 0 || currentFile == "" {
			return
		}
		syntheticPath := fmt.Sprintf("git:history:%s:%s", shortHash(currentCommit), currentFile)
		content := []byte(strings.Join(hunkLines, "\n"))
		findings := detector.detectWithRanges(syntheticPath, content, nil)
		for _, secret := range findings {
			// Translate line number to the original diff line.
			secret.Location.File = syntheticPath
			key := fmt.Sprintf("%s:%d:%s", syntheticPath, secret.Location.Line+hunkStartLine-1, secret.Value)
			if !seen[key] {
				seen[key] = true
				all = append(all, secret)
			}
		}
		hunkLines = nil
		hunkStartLine = 0
		lineCounter = 0
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10 MB max line
	for scanner.Scan() {
		line := scanner.Text()

		// Commit boundary
		if strings.HasPrefix(line, "COMMIT:") {
			flushHunk()
			currentCommit = strings.TrimPrefix(line, "COMMIT:")
			currentFile = ""
			continue
		}

		// New file being diffed: "diff --git a/path b/path"
		if strings.HasPrefix(line, "diff --git ") {
			flushHunk()
			currentFile = extractDiffPath(line, repoPath)
			hunkStartLine = 0
			lineCounter = 0
			continue
		}

		// Hunk header: "@@ -a,b +c,d @@"
		if strings.HasPrefix(line, "@@ ") {
			flushHunk()
			hunkStartLine = parseHunkNewStart(line)
			lineCounter = 0
			continue
		}

		// Added lines only
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			hunkLines = append(hunkLines, line[1:]) // strip leading "+"
			lineCounter++
		}
	}
	flushHunk()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading git diff output: %w", err)
	}

	return all, nil
}

// extractDiffPath parses the file path from a "diff --git a/PATH b/PATH" line.
// It returns the relative path (b-side) normalized to the OS path separator.
func extractDiffPath(line, _ string) string {
	// Format: diff --git a/<path> b/<path>
	parts := strings.SplitN(line, " b/", 2)
	if len(parts) != 2 {
		return ""
	}
	return filepath.FromSlash(parts[1])
}

// parseHunkNewStart parses the "+c" start line from a hunk header like:
// "@@ -a,b +c,d @@ optional context"
func parseHunkNewStart(header string) int {
	// Find the +c,d part
	plusIdx := strings.Index(header, " +")
	if plusIdx < 0 {
		return 1
	}
	rest := header[plusIdx+2:]
	endIdx := strings.IndexAny(rest, ", @")
	if endIdx < 0 {
		endIdx = len(rest)
	}
	n, err := strconv.Atoi(rest[:endIdx])
	if err != nil {
		return 1
	}
	return n
}

func shortHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

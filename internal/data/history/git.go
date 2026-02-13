package history

import (
	"bytes"
	"os/exec"
	"strings"
	"time"
)

func ResolveGitMetadata(projectRoot string) (string, time.Time) {
	commitHash := runGit(projectRoot, "rev-parse", "--short=12", "HEAD")
	commitTimeRaw := runGit(projectRoot, "show", "-s", "--format=%cI", "HEAD")
	if commitHash == "" || commitTimeRaw == "" {
		return "", time.Time{}
	}

	commitTime, err := time.Parse(time.RFC3339, commitTimeRaw)
	if err != nil {
		return commitHash, time.Time{}
	}
	return commitHash, commitTime.UTC()
}

func runGit(projectRoot string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", projectRoot}, args...)...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

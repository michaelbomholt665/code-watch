package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func InjectDiagram(filePath, marker, diagram string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read markdown file %q: %w", filePath, err)
	}

	next, err := ReplaceBetweenMarkers(string(content), marker, diagram)
	if err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".markdown-inject-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", filePath, err)
	}
	tmpName := tmp.Name()

	writeErr := error(nil)
	if _, err := tmp.WriteString(next); err != nil {
		writeErr = fmt.Errorf("write temp markdown file %q: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil && writeErr == nil {
		writeErr = fmt.Errorf("close temp markdown file %q: %w", tmpName, err)
	}
	if writeErr != nil {
		_ = os.Remove(tmpName)
		return writeErr
	}

	if err := os.Rename(tmpName, filePath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replace markdown file %q: %w", filePath, err)
	}
	return nil
}

func ReplaceBetweenMarkers(content, marker, replacement string) (string, error) {
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return "", fmt.Errorf("markdown marker must not be empty")
	}

	newline := "\n"
	if strings.Contains(content, "\r\n") {
		newline = "\r\n"
	}

	start := fmt.Sprintf("<!-- circular:%s:start -->", marker)
	end := fmt.Sprintf("<!-- circular:%s:end -->", marker)

	startCount := strings.Count(content, start)
	endCount := strings.Count(content, end)
	if startCount != 1 || endCount != 1 {
		return "", fmt.Errorf("markdown marker %q must appear exactly once for start and end", marker)
	}

	startIdx := strings.Index(content, start)
	endIdx := strings.Index(content, end)
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return "", fmt.Errorf("invalid marker order for %q", marker)
	}

	startBlockEnd := startIdx + len(start)
	prefix := content[:startBlockEnd]
	suffix := content[endIdx:]
	cleanReplacement := strings.TrimRight(replacement, "\r\n")

	return prefix + newline + cleanReplacement + newline + suffix, nil
}

package util

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NormalizePatternPath cleans and normalizes paths for matcher/pattern usage.
func NormalizePatternPath(s string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(s)))
	if clean == "." {
		return ""
	}
	return strings.TrimPrefix(clean, "./")
}

// HasPathPrefix returns true when path equals prefix or is contained within prefix.
func HasPathPrefix(path, prefix string) bool {
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}

// SortedStringKeys returns the map's keys in sorted order.
func SortedStringKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// WriteFileWithDirs creates parent directories (0755) and writes the file with perm.
func WriteFileWithDirs(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, perm)
}

// WriteStringWithDirs writes string content with parent directories created.
func WriteStringWithDirs(path, content string, perm fs.FileMode) error {
	return WriteFileWithDirs(path, []byte(content), perm)
}

package helpers

import (
	"circular/internal/shared/util"
	"path/filepath"
)

func ResolveOutputPath(path, root string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

func ResolveDiagramPath(path, root, diagramsDir string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if util.ContainsPathSeparator(path) {
		return filepath.Join(root, path)
	}
	return filepath.Join(diagramsDir, path)
}

func WriteArtifact(path, content string) error {
	return util.WriteStringWithDirs(path, content, 0o644)
}

// # internal/resolver/python_resolver.go
package resolver

import (
	"os"
	"path/filepath"
	"strings"
)

type PythonResolver struct {
	projectRoot string
}

func NewPythonResolver(projectRoot string) *PythonResolver {
	return &PythonResolver{projectRoot: projectRoot}
}

func (r *PythonResolver) GetModuleName(filePath string) string {
	rel, err := filepath.Rel(r.projectRoot, filePath)
	if err != nil {
		return ""
	}

	parts := strings.Split(rel, string(os.PathSeparator))

	// Remove non-package prefixes (dirs without __init__.py)
	packageStart := 0
	for i := 0; i < len(parts)-1; i++ {
		checkPath := filepath.Join(r.projectRoot, filepath.Join(parts[:i+1]...), "__init__.py")
		if _, err := os.Stat(checkPath); os.IsNotExist(err) {
			packageStart = i + 1
		} else {
			break // Found first package
		}
	}

	parts = parts[packageStart:]

	// Remove .py extension
	parts[len(parts)-1] = strings.TrimSuffix(parts[len(parts)-1], ".py")

	// Special case: __init__.py
	if parts[len(parts)-1] == "__init__" {
		parts = parts[:len(parts)-1]
	}

	return strings.Join(parts, ".")
}

func (r *PythonResolver) ResolveImport(fromModule, importStmt string, isRelative bool, relativeLevel int) string {
	if !isRelative {
		return importStmt
	}

	parts := strings.Split(fromModule, ".")
	if relativeLevel >= len(parts) {
		return importStmt
	}

	base := strings.Join(parts[:len(parts)-relativeLevel], ".")
	if importStmt == "" {
		return base
	}
	return base + "." + importStmt
}

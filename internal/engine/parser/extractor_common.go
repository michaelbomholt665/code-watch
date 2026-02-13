package parser

import (
	"strings"
	"unicode"
)

func normalizeRefName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\t", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func trimQuoted(value string) string {
	value = strings.TrimSpace(value)
	return strings.Trim(value, "\"'`")
}

func splitAndTrim(value, sep string) []string {
	parts := strings.Split(value, sep)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func isExportedName(name string) bool {
	if name == "" {
		return false
	}
	first := rune(name[0])
	return unicode.IsUpper(first)
}

func appendUnique(values []string, seen map[string]bool, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	if seen[value] {
		return values
	}
	seen[value] = true
	return append(values, value)
}

func ModuleReferenceBase(language, module string) string {
	if module == "" {
		return ""
	}

	if language == "go" {
		// Special cases for common libraries that don't use their path base
		if strings.HasSuffix(module, "go-tree-sitter") {
			return "sitter"
		}
		if strings.HasSuffix(module, "tree-sitter-go/bindings/go") {
			return "tree_sitter_go"
		}
		if strings.HasSuffix(module, "tree-sitter-python/bindings/go") {
			return "tree_sitter_python"
		}

		// Handle Go module paths
		parts := strings.Split(module, "/")
		return parts[len(parts)-1]
	}

	if language == "python" {
		// Handle Python package paths
		parts := strings.Split(module, ".")
		return parts[len(parts)-1]
	}

	if language == "javascript" || language == "typescript" || language == "tsx" {
		// Handle JS/TS imports
		parts := strings.Split(module, "/")
		base := parts[len(parts)-1]
		// Strip extension if present
		if idx := strings.LastIndex(base, "."); idx != -1 {
			return base[:idx]
		}
		return base
	}

	if language == "java" {
		parts := strings.Split(module, ".")
		return parts[len(parts)-1]
	}

	if language == "rust" {
		parts := strings.Split(module, "::")
		return parts[len(parts)-1]
	}

	return module
}

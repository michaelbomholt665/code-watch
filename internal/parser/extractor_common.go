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

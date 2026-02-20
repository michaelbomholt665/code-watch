package helpers

import (
	"os"
	"path/filepath"
	"strings"
)

func HasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[]{}")
}

func IsPathOverlap(a, b string) bool {
	if a == b {
		return true
	}
	if strings.HasPrefix(a, b+string(os.PathSeparator)) {
		return true
	}
	if strings.HasPrefix(b, a+string(os.PathSeparator)) {
		return true
	}
	return false
}

func WildcardPatternsOverlap(a, b string) bool {
	if a == b || strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}

	aPrefix := wildcardPrefix(a)
	bPrefix := wildcardPrefix(b)
	if aPrefix != "" && bPrefix != "" && (strings.HasPrefix(aPrefix, bPrefix) || strings.HasPrefix(bPrefix, aPrefix)) {
		return true
	}

	aSample := wildcardSample(a)
	if aSample != "" {
		if matched, _ := filepath.Match(b, aSample); matched {
			return true
		}
	}

	bSample := wildcardSample(b)
	if bSample != "" {
		if matched, _ := filepath.Match(a, bSample); matched {
			return true
		}
	}

	return false
}

func wildcardPrefix(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[]{}")
	if idx == -1 {
		return pattern
	}
	return pattern[:idx]
}

func wildcardSample(pattern string) string {
	var sample strings.Builder
	inSet := false
	for _, ch := range pattern {
		switch {
		case ch == '[':
			inSet = true
			sample.WriteRune('x')
		case ch == ']':
			inSet = false
		case inSet:
			continue
		case ch == '*' || ch == '?' || ch == '{' || ch == '}' || ch == ',':
			sample.WriteRune('x')
		default:
			sample.WriteRune(ch)
		}
	}
	return sample.String()
}

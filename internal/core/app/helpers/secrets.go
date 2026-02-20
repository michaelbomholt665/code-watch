package helpers

import (
	"circular/internal/core/ports"
	"circular/internal/engine/parser"
	secretengine "circular/internal/engine/secrets"
	"strings"
)

func MaskSecretValue(value string) string {
	length := len(value)
	if length == 0 {
		return ""
	}
	if length <= 8 {
		return strings.Repeat("*", length)
	}
	return value[:4] + "..." + value[length-4:]
}

func DetectSecrets(
	scanner ports.SecretScanner,
	path string,
	previousContent, content []byte,
	previousSecrets []parser.Secret,
) []parser.Secret {
	if scanner == nil {
		return nil
	}
	incremental, ok := scanner.(ports.IncrementalSecretScanner)
	if !ok || len(previousContent) == 0 {
		return scanner.Detect(path, content)
	}
	prevLines := strings.Count(string(previousContent), "\n")
	currLines := strings.Count(string(content), "\n")
	if prevLines != currLines {
		return scanner.Detect(path, content)
	}
	changed := secretengine.ChangedLineRanges(previousContent, content)
	if len(changed) == 0 {
		return previousSecrets
	}
	ranges := make([]ports.LineRange, 0, len(changed))
	for _, r := range changed {
		ranges = append(ranges, ports.LineRange{Start: r.Start, End: r.End})
	}
	updated := incremental.DetectInRanges(path, content, ranges)
	if len(previousSecrets) == 0 {
		return updated
	}
	merged := make([]parser.Secret, 0, len(previousSecrets)+len(updated))
	for _, finding := range previousSecrets {
		if !LineWithinRanges(finding.Location.Line, changed) {
			merged = append(merged, finding)
		}
	}
	merged = append(merged, updated...)
	return merged
}

func LineWithinRanges(line int, ranges []secretengine.LineRange) bool {
	for _, r := range ranges {
		if line >= r.Start && line <= r.End {
			return true
		}
	}
	return false
}

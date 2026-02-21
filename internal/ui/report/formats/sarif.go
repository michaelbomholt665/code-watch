// # internal/ui/report/formats/sarif.go
package formats

import (
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/shared/version"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// SARIF v2.1.0 schema – see https://schemastore.azurewebsites.net/schemas/json/sarif-2.1.0-rtm.5.json

const (
	sarifSchema  = "https://schemastore.azurewebsites.net/schemas/json/sarif-2.1.0-rtm.5.json"
	sarifVersion = "2.1.0"

	ruleIDCycle         = "CIRC001"
	ruleIDSecret        = "CIRC002"
	ruleIDViolation     = "CIRC003"
	ruleIDArchRuleError = "CIRC004"
)

// sarifReport is the top-level SARIF document.
type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	ShortDescription sarifMessage           `json:"shortDescription"`
	DefaultConfig    sarifRuleDefaultConfig `json:"defaultConfiguration"`
}

type sarifRuleDefaultConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
}

// GenerateSARIF builds a SARIF v2.1.0 document from analysis results.
// All file URIs are made relative to projectRoot; absolute paths are never
// included so that reports are safe to share.
func GenerateSARIF(
	projectRoot string,
	cycles [][]string,
	violations []graph.ArchitectureViolation,
	ruleViolations []ports.ArchitectureRuleViolation,
	secrets []parser.Secret,
) ([]byte, error) {
	rules := buildSARIFRules(cycles, violations, ruleViolations, secrets)
	results := make([]sarifResult, 0)

	// --- Cycles → CIRC001 ---
	for _, cycle := range cycles {
		msg := fmt.Sprintf("Circular dependency: %s", strings.Join(cycle, " → "))
		result := sarifResult{
			RuleID:  ruleIDCycle,
			Level:   "error",
			Message: sarifMessage{Text: msg},
		}
		// Attribute to the first file found in the first module of the cycle.
		// We cannot resolve file from module name here, so we use the module path.
		if len(cycle) > 0 {
			result.Locations = []sarifLocation{moduleLocation(projectRoot, cycle[0])}
		}
		results = append(results, result)
	}

	// --- Architecture violations → CIRC003 ---
	for _, v := range violations {
		msg := fmt.Sprintf("Architecture rule %q violated: %s (%s) → %s (%s)",
			v.RuleName, v.FromModule, v.FromLayer, v.ToModule, v.ToLayer)
		result := sarifResult{
			RuleID:  ruleIDViolation,
			Level:   "warning",
			Message: sarifMessage{Text: msg},
		}
		if v.File != "" {
			uri := relativeURI(projectRoot, v.File)
			loc := sarifLocation{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI:       uri,
						URIBaseID: "%SRCROOT%",
					},
				},
			}
			if v.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{
					StartLine:   v.Line,
					StartColumn: v.Column,
				}
			}
			result.Locations = []sarifLocation{loc}
		}
		results = append(results, result)
	}

	// --- Secrets → CIRC002 ---
	for _, s := range secrets {
		level := secretSeverityToLevel(s.Severity)
		msg := fmt.Sprintf("Potential secret detected: %s (confidence %.0f%%)",
			s.Kind, s.Confidence*100)
		result := sarifResult{
			RuleID:  ruleIDSecret,
			Level:   level,
			Message: sarifMessage{Text: msg},
		}
		if s.Location.File != "" {
			uri := relativeURI(projectRoot, s.Location.File)
			loc := sarifLocation{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI:       uri,
						URIBaseID: "%SRCROOT%",
					},
				},
			}
			if s.Location.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{
					StartLine:   s.Location.Line,
					StartColumn: s.Location.Column,
				}
			}
			result.Locations = []sarifLocation{loc}
		}
		results = append(results, result)
	}

	// --- Architecture rule violations → CIRC004 ---
	for _, v := range ruleViolations {
		msg := fmt.Sprintf("Architecture rule %q violated: %s", v.RuleName, v.Message)
		if v.Type == "file_count" {
			msg = fmt.Sprintf("Architecture rule %q violated: %s (files %d > limit %d)", v.RuleName, v.Module, v.Actual, v.Limit)
		}
		result := sarifResult{
			RuleID:  ruleIDArchRuleError,
			Level:   "warning",
			Message: sarifMessage{Text: msg},
		}
		if v.File != "" {
			uri := relativeURI(projectRoot, v.File)
			loc := sarifLocation{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI:       uri,
						URIBaseID: "%SRCROOT%",
					},
				},
			}
			if v.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{
					StartLine:   v.Line,
					StartColumn: v.Column,
				}
			}
			result.Locations = []sarifLocation{loc}
		}
		results = append(results, result)
	}

	report := sarifReport{
		Schema:  sarifSchema,
		Version: sarifVersion,
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    "circular",
						Version: version.Version,
						Rules:   rules,
					},
				},
				Results: results,
			},
		},
	}

	return json.MarshalIndent(report, "", "  ")
}

// buildSARIFRules returns only the rules that are relevant for the given findings.
func buildSARIFRules(cycles [][]string, violations []graph.ArchitectureViolation, ruleViolations []ports.ArchitectureRuleViolation, secrets []parser.Secret) []sarifRule {
	rules := make([]sarifRule, 0, 3)
	if len(cycles) > 0 {
		rules = append(rules, sarifRule{
			ID:               ruleIDCycle,
			Name:             "CircularDependency",
			ShortDescription: sarifMessage{Text: "Circular import dependency detected between modules."},
			DefaultConfig:    sarifRuleDefaultConfig{Level: "error"},
		})
	}
	if len(secrets) > 0 {
		rules = append(rules, sarifRule{
			ID:               ruleIDSecret,
			Name:             "PotentialSecret",
			ShortDescription: sarifMessage{Text: "A potential secret or high-entropy token was detected."},
			DefaultConfig:    sarifRuleDefaultConfig{Level: "warning"},
		})
	}
	if len(violations) > 0 {
		rules = append(rules, sarifRule{
			ID:               ruleIDViolation,
			Name:             "ArchitectureViolation",
			ShortDescription: sarifMessage{Text: "A module-layer architecture rule was violated."},
			DefaultConfig:    sarifRuleDefaultConfig{Level: "warning"},
		})
	}
	if len(ruleViolations) > 0 {
		rules = append(rules, sarifRule{
			ID:               ruleIDArchRuleError,
			Name:             "ArchitectureRuleViolation",
			ShortDescription: sarifMessage{Text: "A module architecture rule was violated."},
			DefaultConfig:    sarifRuleDefaultConfig{Level: "warning"},
		})
	}
	return rules
}

// relativeURI converts an absolute file path to a forward-slash relative URI
// anchored at projectRoot. If the path is already relative or projectRoot is
// empty, the original path (with forward slashes) is returned.
func relativeURI(projectRoot, filePath string) string {
	if projectRoot != "" && filepath.IsAbs(filePath) {
		rel, err := filepath.Rel(projectRoot, filePath)
		if err == nil {
			filePath = rel
		}
	}
	// SARIF URIs use forward slashes.
	return filepath.ToSlash(filePath)
}

// moduleLocation creates a SARIF location for a module name when no file
// path is available (used for cycle results).
func moduleLocation(projectRoot, moduleName string) sarifLocation {
	// Module name is used as a synthetic URI; not a real file path.
	_ = projectRoot
	return sarifLocation{
		PhysicalLocation: sarifPhysicalLocation{
			ArtifactLocation: sarifArtifactLocation{
				URI:       moduleName,
				URIBaseID: "%SRCROOT%",
			},
		},
	}
}

// secretSeverityToLevel maps circular severity strings to SARIF levels.
func secretSeverityToLevel(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}

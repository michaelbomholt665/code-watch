package app

import (
	"circular/internal/engine/graph"
	"fmt"
	"strings"
)

func FormatImpactReport(report graph.ImpactReport) string {
	var b strings.Builder

	b.WriteString("Impact Analysis\n")
	b.WriteString("==============\n")
	b.WriteString(fmt.Sprintf("Target module: %s\n", report.TargetModule))
	if report.TargetPath != "" {
		b.WriteString(fmt.Sprintf("Target file: %s\n", report.TargetPath))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Direct importers (%d)\n", len(report.DirectImporters)))
	for _, mod := range report.DirectImporters {
		b.WriteString(fmt.Sprintf("- %s\n", mod))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Transitive impact (%d)\n", len(report.TransitiveImporters)))
	for _, mod := range report.TransitiveImporters {
		b.WriteString(fmt.Sprintf("- %s\n", mod))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Externally used symbols (%d)\n", len(report.ExternallyUsedSymbols)))
	for _, sym := range report.ExternallyUsedSymbols {
		b.WriteString(fmt.Sprintf("- %s\n", sym))
	}

	return b.String()
}

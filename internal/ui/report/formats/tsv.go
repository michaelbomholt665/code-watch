// # internal/output/tsv.go
package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"circular/internal/engine/secrets"
	"fmt"
	"strings"
)

type TSVGenerator struct {
	graph *graph.Graph
}

func NewTSVGenerator(g *graph.Graph) *TSVGenerator {
	return &TSVGenerator{graph: g}
}

func (t *TSVGenerator) Generate() (string, error) {
	var buf strings.Builder

	buf.WriteString("From\tTo\tFile\tLine\tColumn\n")

	imports := t.graph.GetImports()
	for from, targets := range imports {
		for to, edge := range targets {
			buf.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\n",
				from, to, edge.ImportedBy, edge.Location.Line, edge.Location.Column))
		}
	}

	return buf.String(), nil
}

func (t *TSVGenerator) GenerateUnusedImports(rows []resolver.UnusedImport) (string, error) {
	var buf strings.Builder

	buf.WriteString("Type\tFile\tLanguage\tModule\tAlias\tItem\tLine\tColumn\tConfidence\n")
	for _, row := range rows {
		buf.WriteString(fmt.Sprintf("unused_import\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			row.File,
			row.Language,
			row.Module,
			row.Alias,
			row.Item,
			row.Location.Line,
			row.Location.Column,
			row.Confidence,
		))
	}

	return buf.String(), nil
}

func (t *TSVGenerator) GenerateArchitectureViolations(rows []graph.ArchitectureViolation) (string, error) {
	var buf strings.Builder

	buf.WriteString("Type\tRule\tFromModule\tFromLayer\tToModule\tToLayer\tFile\tLine\tColumn\n")
	for _, row := range rows {
		buf.WriteString(fmt.Sprintf("architecture_violation\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
			row.RuleName,
			row.FromModule,
			row.FromLayer,
			row.ToModule,
			row.ToLayer,
			row.File,
			row.Line,
			row.Column,
		))
	}

	return buf.String(), nil
}

func (t *TSVGenerator) GenerateProbableBridges(rows []resolver.ProbableBridgeReference) (string, error) {
	var buf strings.Builder

	buf.WriteString("Type\tFile\tReference\tLine\tColumn\tConfidence\tScore\tReasons\n")
	for _, row := range rows {
		buf.WriteString(fmt.Sprintf("probable_bridge\t%s\t%s\t%d\t%d\t%s\t%d\t%s\n",
			row.File,
			row.Reference.Name,
			row.Reference.Location.Line,
			row.Reference.Location.Column,
			row.Confidence,
			row.Score,
			strings.Join(row.Reasons, ","),
		))
	}

	return buf.String(), nil
}

func (t *TSVGenerator) GenerateSecrets(rows []parser.Secret) (string, error) {
	var buf strings.Builder

	buf.WriteString("Type\tKind\tSeverity\tValue\tEntropy\tConfidence\tFile\tLine\tColumn\n")
	for _, row := range rows {
		buf.WriteString(fmt.Sprintf("secret\t%s\t%s\t%s\t%.4f\t%.2f\t%s\t%d\t%d\n",
			row.Kind,
			row.Severity,
			secrets.MaskValue(row.Value),
			row.Entropy,
			row.Confidence,
			row.Location.File,
			row.Location.Line,
			row.Location.Column,
		))
	}

	return buf.String(), nil
}

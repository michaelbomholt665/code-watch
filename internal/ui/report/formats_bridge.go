package report

import (
	"circular/internal/engine/graph"
	"circular/internal/ui/report/formats"
)

type DOTGenerator = formats.DOTGenerator
type TSVGenerator = formats.TSVGenerator
type MermaidGenerator = formats.MermaidGenerator
type PlantUMLGenerator = formats.PlantUMLGenerator

func NewDOTGenerator(g *graph.Graph) *DOTGenerator {
	return formats.NewDOTGenerator(g)
}

func NewTSVGenerator(g *graph.Graph) *TSVGenerator {
	return formats.NewTSVGenerator(g)
}

func NewMermaidGenerator(g *graph.Graph) *MermaidGenerator {
	return formats.NewMermaidGenerator(g)
}

func NewPlantUMLGenerator(g *graph.Graph) *PlantUMLGenerator {
	return formats.NewPlantUMLGenerator(g)
}

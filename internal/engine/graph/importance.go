package graph

// internal/engine/graph/importance.go

import "strings"

// CalculateImportanceScore ranks a module's architectural significance using a
// weighted heuristic from the Advanced Graph Visualization plan:
//
//	Score = (FanIn * 2) + (FanOut * 1) + (Complexity * 0.5) + (IsAPI ? 10 : 0)
//
// Parameters:
//   - fanIn:           number of internal modules that import this module
//   - fanOut:          number of internal modules this module imports
//   - complexityScore: top complexity score in this module (from TopComplexity)
//   - moduleName:      used to auto-detect "API surface" modules
func CalculateImportanceScore(fanIn, fanOut, complexityScore int, moduleName string) float64 {
	score := float64(fanIn*2) + float64(fanOut*1) + float64(complexityScore)*0.5
	if isAPIModule(moduleName) {
		score += 10
	}
	return score
}

// isAPIModule returns true when the module name suggests it is a public API surface.
// It matches common naming conventions used in Go and Python projects.
func isAPIModule(name string) bool {
	lower := strings.ToLower(name)
	keywords := []string{"api", "gateway", "handler", "server", "service"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

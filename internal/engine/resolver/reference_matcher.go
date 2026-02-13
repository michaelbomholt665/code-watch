package resolver

import (
	"strings"
)

// isQualifiedMatch checks if a reference name matches a symbol in an imported module.
func isQualifiedMatch(refName, modBase, symbolName string) bool {
	// Simple match: pkg.Symbol
	if refName == modBase+"."+symbolName {
		return true
	}

	// Handle chained calls: NewPkg().Symbol
	if strings.HasPrefix(refName, modBase+"().") && strings.HasSuffix(refName, "."+symbolName) {
		return true
	}

	// Handle constructor chains: Pkg.New().Symbol
	if strings.HasPrefix(refName, modBase+".") && strings.Contains(refName, "."+symbolName) {
		return true
	}

	return false
}

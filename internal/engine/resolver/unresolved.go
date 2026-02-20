package resolver

import (
	"circular/internal/engine/parser"
	"strings"
)

// resolveQualifiedReference checks if a reference matches an imported or local module symbol.
func (r *Resolver) resolveQualifiedReference(file *parser.File, ref parser.Reference) bool {
	// 1. Check local module (same package/module)
	if r.checkModule(file.Module, ref.Name, true) {
		return true
	}

	// 2. Check imported modules
	for _, imp := range file.Imports {
		modBase := parser.ModuleReferenceBase(file.Language, imp.Module)

		// Check: import "github.com/tree-sitter/go-tree-sitter" -> sitter.Node
		// We need to check if modBase matches the prefix.
		// Some packages have bases that don't match their typical alias (e.g. go-tree-sitter -> sitter)

		isMatch := false
		symbolName := ""

		if imp.Alias != "" {
			if ref.Name == imp.Alias {
				isMatch = true
			} else if strings.HasPrefix(ref.Name, imp.Alias+".") {
				isMatch = true
				symbolName = strings.TrimPrefix(ref.Name, imp.Alias+".")
			}
		}

		if !isMatch {
			if ref.Name == modBase {
				isMatch = true
			} else if strings.HasPrefix(ref.Name, modBase+".") {
				isMatch = true
				symbolName = strings.TrimPrefix(ref.Name, modBase+".")
			}
		}

		if isMatch {
			if symbolName != "" {
				if r.graph.HasDefinitions(imp.Module) {
					if r.checkModule(imp.Module, symbolName, false) {
						return true
					}
				} else {
					// External or stdlib
					return true
				}
			} else {
				return true
			}
		}

		// Handle: from auth import login -> login()
		if len(imp.Items) > 0 {
			for _, item := range imp.Items {
				if ref.Name == item || strings.HasPrefix(ref.Name, item+".") {
					if r.graph.HasDefinitions(imp.Module) {
						if r.checkModule(imp.Module, item, false) {
							return true
						}
					} else {
						return true
					}
				}
			}
		}

		// Handle chained calls: NewGoResolver().ModuleBaseName
		if strings.HasPrefix(ref.Name, modBase+"().") || (imp.Alias != "" && strings.HasPrefix(ref.Name, imp.Alias+"().")) {
			return true
		}
	}

	return false
}

func (r *Resolver) FindUnresolved() []UnresolvedReference {
	var unresolved []UnresolvedReference

	files := r.graph.GetAllFiles()

	for _, file := range files {
		unresolved = append(unresolved, r.findUnresolvedInFile(file)...)
	}

	return unresolved
}

func (r *Resolver) FindProbableBridgeReferences() []ProbableBridgeReference {
	var probable []ProbableBridgeReference

	files := r.graph.GetAllFiles()

	for _, file := range files {
		probable = append(probable, r.findProbableBridgeReferencesInFile(file)...)
	}

	return probable
}

func (r *Resolver) FindUnresolvedForPaths(paths []string) []UnresolvedReference {
	var unresolved []UnresolvedReference
	seen := make(map[string]bool, len(paths))

	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true

		file, ok := r.graph.GetFile(path)
		if !ok {
			continue
		}
		unresolved = append(unresolved, r.findUnresolvedInFile(file)...)
	}

	return unresolved
}

func (r *Resolver) FindProbableBridgeReferencesForPaths(paths []string) []ProbableBridgeReference {
	var probable []ProbableBridgeReference
	seen := make(map[string]bool, len(paths))

	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true

		file, ok := r.graph.GetFile(path)
		if !ok {
			continue
		}
		probable = append(probable, r.findProbableBridgeReferencesInFile(file)...)
	}

	return probable
}

func (r *Resolver) findUnresolvedInFile(file *parser.File) []UnresolvedReference {
	var unresolved []UnresolvedReference
	for _, ref := range file.References {
		result := r.resolveReferenceResult(file, ref)
		if result.status != referenceUnresolved {
			continue
		}
		if isLikelyErrorReference(file, ref) {
			unresolved = append(unresolved, UnresolvedReference{
				Reference: ref,
				File:      file.Path,
			})
		}
	}
	return unresolved
}

func (r *Resolver) findProbableBridgeReferencesInFile(file *parser.File) []ProbableBridgeReference {
	var probable []ProbableBridgeReference
	for _, ref := range file.References {
		result := r.resolveReferenceResult(file, ref)
		if result.status != referenceProbableBridge {
			continue
		}
		probable = append(probable, ProbableBridgeReference{
			Reference:  ref,
			File:       file.Path,
			Score:      result.bridge.score,
			Confidence: result.bridge.confidence,
			Reasons:    append([]string(nil), result.bridge.reasons...),
		})
	}
	return probable
}

func isLikelyErrorReference(file *parser.File, ref parser.Reference) bool {
	// CONFIDENCE GATING:
	// Report qualified references (mod.Symbol) even when no import matches.
	for _, imp := range file.Imports {
		modBase := parser.ModuleReferenceBase(file.Language, imp.Module)
		if strings.HasPrefix(ref.Name, modBase+".") || (imp.Alias != "" && strings.HasPrefix(ref.Name, imp.Alias+".")) {
			return true
		}
	}

	parts := strings.Split(ref.Name, ".")
	return len(parts) > 1 && parts[0] != ""
}

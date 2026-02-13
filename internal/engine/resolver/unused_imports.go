package resolver

import (
	"circular/internal/engine/parser"
	"strings"
)

func (r *Resolver) FindUnusedImports(paths []string) []UnusedImport {
	seen := make(map[string]bool, len(paths))
	unused := make([]UnusedImport, 0)

	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true

		file, ok := r.graph.GetFile(path)
		if !ok {
			continue
		}

		unused = append(unused, r.findUnusedInFile(file)...)
	}

	return unused
}

func (r *Resolver) findUnusedInFile(file *parser.File) []UnusedImport {
	if !supportsUnusedImport(file.Language) {
		return nil
	}

	unused := make([]UnusedImport, 0)
	refHits := make(map[string]int, len(file.References))
	for _, ref := range file.References {
		refHits[ref.Name]++
	}

	for _, imp := range file.Imports {
		// Go side-effect imports are intentionally "unused" in code references.
		if file.Language == "go" && imp.Alias == "_" {
			continue
		}
		// Dot imports blend symbols into local namespace and are hard to verify safely.
		if file.Language == "go" && imp.Alias == "." {
			continue
		}

		name := importReferenceName(file.Language, imp)
		if name == "" {
			continue
		}

		if r.isExcludedImport(imp.Module, name) {
			continue
		}

		if len(imp.Items) > 0 {
			for _, item := range imp.Items {
				if item == "" {
					continue
				}
				// For 'from pkg import sym', we check if 'sym' or 'pkg.sym' is used
				if !hasSymbolUse(refHits, item) && !hasSymbolUse(refHits, name+"."+item) {
					unused = append(unused, UnusedImport{
						File:       file.Path,
						Language:   file.Language,
						Module:     imp.Module,
						Alias:      imp.Alias,
						Item:       item,
						Location:   imp.Location,
						Confidence: "high",
					})
				}
			}
			continue
		}

		// JavaScript/TypeScript module imports can be side-effect-only and do not
		// always bind a symbol that can be tracked by reference matching.
		if isLikelySideEffectOnlyImport(file.Language, imp) {
			continue
		}

		if !hasSymbolUse(refHits, name) {
			unused = append(unused, UnusedImport{
				File:       file.Path,
				Language:   file.Language,
				Module:     imp.Module,
				Alias:      imp.Alias,
				Location:   imp.Location,
				Confidence: "medium",
			})
		}
	}

	return unused
}

func (r *Resolver) isExcludedImport(module, refName string) bool {
	for _, excluded := range r.excludedImports {
		if excluded == module || excluded == refName {
			return true
		}
	}
	return false
}

func hasSymbolUse(refHits map[string]int, symbol string) bool {
	if symbol == "" {
		return true
	}

	// 1. Check for exact match in the reference map
	if refHits[symbol] > 0 {
		return true
	}

	// 2. Check if the symbol appears as a prefix or sub-component in any qualified reference
	prefix := symbol + "."
	prefixCall := symbol + "("
	prefixChained := symbol + "()."

	for ref := range refHits {
		if strings.HasPrefix(ref, prefix) ||
			strings.HasPrefix(ref, prefixCall) ||
			strings.HasPrefix(ref, prefixChained) ||
			strings.Contains(ref, "."+symbol+".") ||
			strings.HasSuffix(ref, "."+symbol) {
			return true
		}
	}

	return false
}

func importReferenceName(language string, imp parser.Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	if imp.Module == "" {
		return ""
	}

	return parser.ModuleReferenceBase(language, imp.Module)
}

func supportsUnusedImport(language string) bool {
	switch language {
	case "go", "python", "javascript", "typescript", "tsx", "java", "rust":
		return true
	default:
		return false
	}
}

func isLikelySideEffectOnlyImport(language string, imp parser.Import) bool {
	if len(imp.Items) > 0 || imp.Alias != "" {
		return false
	}
	switch language {
	case "javascript", "typescript", "tsx":
		return true
	default:
		return false
	}
}

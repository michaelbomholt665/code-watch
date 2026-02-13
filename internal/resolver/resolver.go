// # internal/resolver/resolver.go
package resolver

import (
	"circular/internal/graph"
	"circular/internal/parser"
	"strings"
)

type UnresolvedReference struct {
	Reference parser.Reference
	File      string
}

type UnusedImport struct {
	File       string
	Language   string
	Module     string
	Alias      string
	Item       string
	Location   parser.Location
	Confidence string
}

type Resolver struct {
	graph            *graph.Graph
	stdlibByLanguage map[string]map[string]bool
	ExcludedSymbols  []string
}

func NewResolver(g *graph.Graph, excluded []string) *Resolver {
	return &Resolver{
		graph:            g,
		stdlibByLanguage: getStdlibByLanguage(),
		ExcludedSymbols:  excluded,
	}
}

func (r *Resolver) FindUnresolved() []UnresolvedReference {
	var unresolved []UnresolvedReference

	files := r.graph.GetAllFiles()

	for _, file := range files {
		unresolved = append(unresolved, r.findUnresolvedInFile(file)...)
	}

	return unresolved
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

		if len(imp.Items) > 0 {
			for _, item := range imp.Items {
				if item == "" {
					continue
				}
				if !hasSymbolUse(refHits, item) {
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

		name := importReferenceName(file.Language, imp)
		if name == "" {
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

func importReferenceName(language string, imp parser.Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	if imp.Module == "" {
		return ""
	}

	if language == "go" {
		return NewGoResolver().ModuleBaseName(imp.Module)
	}
	if language == "python" {
		parts := strings.Split(imp.Module, ".")
		return parts[len(parts)-1]
	}
	if language == "javascript" || language == "typescript" || language == "tsx" {
		return NewJavaScriptResolver().ResolveModuleName(imp.Module)
	}
	if language == "java" {
		return NewJavaResolver().ResolveModuleName(imp.Module)
	}
	if language == "rust" {
		return NewRustResolver().ResolveModuleName(imp.Module)
	}

	return ""
}

func hasSymbolUse(refHits map[string]int, symbol string) bool {
	if refHits[symbol] > 0 {
		return true
	}
	prefix := symbol + "."
	for ref := range refHits {
		if strings.HasPrefix(ref, prefix) {
			return true
		}
	}
	return false
}

func (r *Resolver) findUnresolvedInFile(file *parser.File) []UnresolvedReference {
	var unresolved []UnresolvedReference
	for _, ref := range file.References {
		if !r.resolveReference(file, ref) {
			unresolved = append(unresolved, UnresolvedReference{
				Reference: ref,
				File:      file.Path,
			})
		}
	}
	return unresolved
}

func (r *Resolver) resolveReference(file *parser.File, ref parser.Reference) bool {
	// 0. Check local symbols (vars, params, etc)
	if r.isLocalSymbol(file, ref.Name) {
		return true
	}

	// 1. Check local module (same package/module)
	if r.checkModule(file.Module, ref.Name, true) { // Allow unexported
		return true
	}

	// 2. Check imported modules
	for _, imp := range file.Imports {
		if imp.Alias != "" {
			// Handle: import numpy as np -> np.array
			if strings.HasPrefix(ref.Name, imp.Alias+".") {
				symbolName := strings.TrimPrefix(ref.Name, imp.Alias+".")
				if r.checkModule(imp.Module, symbolName, false) {
					return true
				}
			}
			// Handle: import numpy as np -> np
			if ref.Name == imp.Alias {
				return true
			}
		}

		// Handle: from auth import login -> login()
		if len(imp.Items) > 0 {
			for _, item := range imp.Items {
				if ref.Name == item || strings.HasPrefix(ref.Name, item+".") {
					if r.checkModule(imp.Module, item, false) {
						return true
					}
				}
			}
		}

		// Handle: import auth -> auth.login()
		// Determine the base name of the module (e.g. log/slog -> slog)
		modBase := moduleReferenceBase(file.Language, imp.Module)

		if strings.HasPrefix(ref.Name, modBase+".") {
			symbolName := strings.TrimPrefix(ref.Name, modBase+".")
			if r.checkModule(imp.Module, symbolName, false) {
				return true
			}
		}

		// Handle direct module reference
		if ref.Name == modBase || ref.Name == imp.Module {
			return true
		}
	}

	// 3. Check stdlib
	if r.isStdlibSymbol(file.Language, ref.Name) || r.isStdlibCall(file.Language, ref.Name) {
		return true
	}

	// 4. Check builtins
	if file.Language == "python" && pythonBuiltins[ref.Name] {
		return true
	}
	if file.Language == "go" && goBuiltins[ref.Name] {
		return true
	}

	return false
}

func (r *Resolver) isLocalSymbol(file *parser.File, name string) bool {
	// Split by dot to handle p.RegisterExtractor -> check if 'p' is local
	parts := strings.Split(name, ".")
	prefix := parts[0]

	for _, sym := range file.LocalSymbols {
		if sym == prefix {
			return true
		}
	}

	if IsKnownNonModule(name, r.ExcludedSymbols) {
		return true
	}

	// Also handle 'self' (Python) and 'this' (Go - though receivers are explicitly named in Go)
	if file.Language == "python" && prefix == "self" {
		return true
	}

	return false
}

func (r *Resolver) checkModule(moduleName, symbolName string, allowUnexported bool) bool {
	defs, ok := r.graph.GetDefinitions(moduleName)
	if !ok {
		return false
	}

	// Direct match
	if def, ok := defs[symbolName]; ok {
		if allowUnexported || def.Exported {
			return true
		}
	}

	// Nested: Class.method or package.Type
	for fullName, def := range defs {
		if !allowUnexported && !def.Exported {
			continue
		}
		if strings.HasPrefix(fullName, symbolName+".") ||
			strings.HasSuffix(fullName, "."+symbolName) {
			return true
		}
	}

	return false
}

func (r *Resolver) isStdlibCall(language, name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return false
	}
	return r.isStdlibSymbol(language, parts[0])
}

func (r *Resolver) isStdlibSymbol(language, name string) bool {
	langStdlib, ok := r.stdlibByLanguage[language]
	if !ok {
		return false
	}
	return langStdlib[name]
}

func moduleReferenceBase(language, module string) string {
	if module == "" {
		return ""
	}
	if language == "go" {
		return NewGoResolver().ModuleBaseName(module)
	}
	if language == "python" {
		parts := strings.Split(module, ".")
		return parts[len(parts)-1]
	}
	if language == "javascript" || language == "typescript" || language == "tsx" {
		return NewJavaScriptResolver().ResolveModuleName(module)
	}
	if language == "java" {
		return NewJavaResolver().ResolveModuleName(module)
	}
	if language == "rust" {
		return NewRustResolver().ResolveModuleName(module)
	}
	return module
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

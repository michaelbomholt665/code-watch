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

type Resolver struct {
	graph           *graph.Graph
	stdlib          map[string]bool
	ExcludedSymbols []string
}

func NewResolver(g *graph.Graph, excluded []string) *Resolver {
	return &Resolver{
		graph:           g,
		stdlib:          getMergedStdlib(),
		ExcludedSymbols: excluded,
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
		modBase := imp.Module
		if file.Language == "go" {
			if parts := strings.Split(imp.Module, "/"); len(parts) > 0 {
				modBase = parts[len(parts)-1]
			}
		} else if file.Language == "python" {
			if parts := strings.Split(imp.Module, "."); len(parts) > 0 {
				modBase = parts[len(parts)-1]
			}
		}

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
	if r.stdlib[ref.Name] || r.isStdlibCall(ref.Name) {
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

func (r *Resolver) isStdlibCall(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return false
	}
	return r.stdlib[parts[0]]
}

func getMergedStdlib() map[string]bool {
	merged := make(map[string]bool)
	for k := range pythonStdlib {
		merged[k] = true
	}
	for k := range goStdlib {
		merged[k] = true
	}
	return merged
}

package resolver

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"strings"
	"unicode"
)

func (r *Resolver) resolveProbabilisticReference(file *parser.File, ref parser.Reference) bool {
	if r.symbolTable == nil {
		return false
	}

	candidates := r.symbolTable.Lookup(ref.Name)
	if len(candidates) == 0 {
		leaf := referenceLeaf(ref.Name)
		if leaf != "" && leaf != ref.Name {
			candidates = r.symbolTable.Lookup(leaf)
		}
	}

	if ref.Context == parser.RefContextService {
		serviceCandidates := r.symbolTable.LookupService(ref.Name)
		if len(serviceCandidates) == 0 {
			leaf := referenceLeaf(ref.Name)
			if leaf != "" {
				serviceCandidates = r.symbolTable.LookupService(leaf)
			}
		}
		candidates = append(candidates, serviceCandidates...)
	}

	if len(candidates) == 0 {
		return false
	}

	best, second := 0, 0
	for _, candidate := range candidates {
		score := scoreCandidate(file, ref, candidate)
		if score > best {
			second = best
			best = score
		} else if score > second {
			second = score
		}
	}

	threshold := 8
	if ref.Context == parser.RefContextFFI || ref.Context == parser.RefContextProcess {
		threshold = 7
	}
	if ref.Context == parser.RefContextService {
		threshold = 6
	}

	return best >= threshold && best-second >= 2
}

func scoreCandidate(file *parser.File, ref parser.Reference, candidate graph.SymbolRecord) int {
	score := 0

	refCanonical := canonicalResolverSymbol(ref.Name)
	refLeaf := referenceLeaf(ref.Name)
	refLeafCanonical := canonicalResolverSymbol(refLeaf)
	candidateCanonical := canonicalResolverSymbol(candidate.Name)

	if refCanonical != "" && refCanonical == candidateCanonical {
		score += 8
	}
	if refLeafCanonical != "" && refLeafCanonical == candidateCanonical {
		score += 6
	}

	if candidate.FullName != "" {
		fullCanonical := canonicalResolverSymbol(candidate.FullName)
		if refCanonical != "" && refCanonical == fullCanonical {
			score += 6
		}
		if refLeafCanonical != "" && refLeafCanonical == fullCanonical {
			score += 4
		}
	}

	if file.Module != "" && candidate.Module == file.Module {
		score += 5
	}
	if file.Language == candidate.Language {
		score += 2
	}

	if candidate.Exported || candidate.Visibility == "public" {
		score++
	}

	if hasModulePrefixMatch(file, ref.Name, candidate.Module) {
		score += 4
	}

	switch ref.Context {
	case parser.RefContextService:
		if candidate.IsService {
			score += 5
		}
	case parser.RefContextFFI, parser.RefContextProcess:
		if !candidate.IsService {
			score++
		}
	}

	if IsCrossLanguageBridgeReference(file.Language, ref) && file.Language != candidate.Language {
		score += 2
	}

	return score
}

func hasModulePrefixMatch(file *parser.File, refName, candidateModule string) bool {
	if candidateModule == "" {
		return false
	}
	for _, imp := range file.Imports {
		if imp.Module != candidateModule {
			continue
		}
		base := importReferenceName(file.Language, imp)
		if base != "" && strings.HasPrefix(refName, base+".") {
			return true
		}
		if imp.Alias != "" && strings.HasPrefix(refName, imp.Alias+".") {
			return true
		}
	}
	return false
}

func referenceLeaf(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '.' || r == ':' || r == '/' || r == '(' || r == ')' || r == '[' || r == ']'
	})
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}

func canonicalResolverSymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(symbol))
	for _, r := range symbol {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

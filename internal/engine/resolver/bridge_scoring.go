package resolver

import (
	"circular/internal/engine/parser"
	"strings"
)

func defaultBridgeResolutionConfig() BridgeResolutionConfig {
	return BridgeResolutionConfig{
		ConfirmedThreshold: 8,
		ProbableThreshold:  5,
		Weights:            defaultBridgeScoreWeights(),
	}
}

func defaultBridgeScoreWeights() BridgeScoreWeights {
	return BridgeScoreWeights{
		ExplicitRuleMatch:       10,
		BridgeContext:           4,
		BridgeImportEvidence:    3,
		UniqueCrossLangMatch:    2,
		AmbiguousCrossLangMatch: -2,
		LocalOrModuleConflict:   -4,
		StdlibConflict:          -3,
	}
}

func (r *Resolver) assessBridgeReference(file *parser.File, ref parser.Reference) bridgeAssessment {
	if r == nil || file == nil {
		return bridgeAssessment{confidence: "low"}
	}

	cfg := r.bridgeConfig
	if cfg.Weights == (BridgeScoreWeights{}) {
		cfg = defaultBridgeResolutionConfig()
	}
	score := 0
	reasons := make([]string, 0, 8)

	for _, bridge := range r.explicitBridges {
		if !bridge.matchesSource(file.Language, file.Module) {
			continue
		}
		if bridge.matchesReference(ref) {
			score += cfg.Weights.ExplicitRuleMatch
			reasons = append(reasons, "explicit_bridge_match")
			break
		}
	}

	if ref.Context == parser.RefContextFFI || ref.Context == parser.RefContextProcess || ref.Context == parser.RefContextService {
		score += cfg.Weights.BridgeContext
		reasons = append(reasons, "bridge_context")
	}

	if IsCrossLanguageBridgeReference(file.Language, ref) {
		score += 1
		reasons = append(reasons, "bridge_prefix_heuristic")
	}

	if r.hasBridgeImportEvidence(file, ref) {
		score += cfg.Weights.BridgeImportEvidence
		reasons = append(reasons, "bridge_import_evidence")
	}

	candidates := r.crossLanguageCandidateCount(file, ref)
	if candidates == 1 {
		score += cfg.Weights.UniqueCrossLangMatch
		reasons = append(reasons, "unique_cross_language_candidate")
	} else if candidates > 1 {
		score += cfg.Weights.AmbiguousCrossLangMatch
		reasons = append(reasons, "ambiguous_cross_language_candidates")
	}

	if r.hasLocalOrModulePrefixConflict(file, ref) {
		score += cfg.Weights.LocalOrModuleConflict
		reasons = append(reasons, "local_or_module_conflict")
	}

	if r.hasStdlibPrefixConflict(file.Language, ref.Name) {
		score += cfg.Weights.StdlibConflict
		reasons = append(reasons, "stdlib_conflict")
	}

	confidence := "low"
	switch {
	case score >= cfg.ConfirmedThreshold:
		confidence = "high"
	case score >= cfg.ProbableThreshold:
		confidence = "medium"
	}

	return bridgeAssessment{
		score:      score,
		confidence: confidence,
		reasons:    reasons,
	}
}

func (r *Resolver) hasBridgeImportEvidence(file *parser.File, ref parser.Reference) bool {
	name := strings.TrimSpace(ref.Name)
	if name == "" {
		return false
	}
	for _, imp := range file.Imports {
		base := importReferenceName(file.Language, imp)
		if base != "" && strings.HasPrefix(name, base+".") && IsCrossLanguageBridgeImportHint(file.Language, imp.Module, base) {
			return true
		}
		module := strings.TrimSpace(imp.Module)
		if module != "" && IsCrossLanguageBridgeImportHint(file.Language, module, base) {
			if strings.HasPrefix(name, module+".") || strings.Contains(name, module+".") {
				return true
			}
		}
	}
	return false
}

func (r *Resolver) crossLanguageCandidateCount(file *parser.File, ref parser.Reference) int {
	if r == nil || r.symbolTable == nil || file == nil {
		return 0
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

	count := 0
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if candidate.Language == "" || candidate.Language == file.Language {
			continue
		}
		key := candidate.Language + "|" + candidate.Module + "|" + candidate.FullName + "|" + candidate.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		count++
	}
	return count
}

func (r *Resolver) hasLocalOrModulePrefixConflict(file *parser.File, ref parser.Reference) bool {
	prefix := normalizedReferencePrefix(ref.Name)
	if prefix == "" {
		return false
	}

	for _, sym := range file.LocalSymbols {
		if sym == prefix {
			return true
		}
	}

	if file.Module != "" && parser.ModuleReferenceBase(file.Language, file.Module) == prefix {
		return true
	}

	for _, imp := range file.Imports {
		base := importReferenceName(file.Language, imp)
		if base == prefix {
			return true
		}
	}
	return false
}

func (r *Resolver) hasStdlibPrefixConflict(language, name string) bool {
	prefix := normalizedReferencePrefix(name)
	if prefix == "" {
		return false
	}
	return r.isStdlibSymbol(language, prefix)
}

func normalizedReferencePrefix(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return ""
	}
	prefix := parts[0]
	if idx := strings.Index(prefix, "["); idx >= 0 {
		prefix = prefix[:idx]
	}
	return strings.TrimLeft(prefix, "*&")
}

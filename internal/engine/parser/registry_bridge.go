package parser

import "circular/internal/engine/parser/registry"

type LanguageSpec = registry.LanguageSpec
type LanguageOverride = registry.LanguageOverride

func DefaultLanguageRegistry() map[string]LanguageSpec {
	return registry.DefaultLanguageRegistry()
}

func BuildLanguageRegistry(overrides map[string]LanguageOverride) (map[string]LanguageSpec, error) {
	return registry.BuildLanguageRegistry(overrides)
}

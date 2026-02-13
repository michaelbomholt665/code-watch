package parser

import "circular/internal/engine/parser/grammar"

type GrammarManifest = grammar.GrammarManifest
type GrammarArtifact = grammar.GrammarArtifact
type VerificationIssue = grammar.VerificationIssue

func LoadGrammarManifest(path string) (GrammarManifest, error) {
	return grammar.LoadGrammarManifest(path)
}

func VerifyGrammarArtifacts(baseDir string, manifest GrammarManifest) ([]VerificationIssue, error) {
	return grammar.VerifyGrammarArtifacts(baseDir, manifest)
}

func VerifyLanguageRegistryArtifacts(baseDir string, registry map[string]LanguageSpec) ([]VerificationIssue, error) {
	return grammar.VerifyLanguageRegistryArtifacts(baseDir, registry)
}

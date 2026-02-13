// # internal/parser/loader.go
package parser

import (
	"circular/internal/shared/util"
	"fmt"
	"os"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_html "github.com/tree-sitter/tree-sitter-html/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type GrammarLoader struct {
	languages map[string]*sitter.Language
	registry  map[string]LanguageSpec
}

func NewGrammarLoader(grammarsPath string) (*GrammarLoader, error) {
	registry, err := BuildLanguageRegistry(nil)
	if err != nil {
		return nil, err
	}
	return NewGrammarLoaderWithRegistry(grammarsPath, registry, false)
}

func NewGrammarLoaderWithRegistry(grammarsPath string, registry map[string]LanguageSpec, verifyArtifacts bool) (*GrammarLoader, error) {
	if registry == nil {
		var err error
		registry, err = BuildLanguageRegistry(nil)
		if err != nil {
			return nil, err
		}
	}

	if grammarsPath != "" {
		if info, err := os.Stat(grammarsPath); err == nil && !info.IsDir() {
			return nil, fmt.Errorf("grammars path is not a directory: %s", grammarsPath)
		}
	}

	if verifyArtifacts && grammarsPath != "" {
		if info, err := os.Stat(grammarsPath); err == nil && info.IsDir() {
			issues, verifyErr := VerifyLanguageRegistryArtifacts(grammarsPath, registry)
			if verifyErr != nil {
				return nil, verifyErr
			}
			if len(issues) > 0 {
				first := issues[0]
				return nil, fmt.Errorf(
					"grammar verification failed (%d issues): %s (%s: %s)",
					len(issues),
					first.Language,
					first.ArtifactPath,
					first.Reason,
				)
			}
		}
	}

	gl := &GrammarLoader{
		languages: make(map[string]*sitter.Language),
		registry:  cloneLanguageRegistry(registry),
	}

	for _, langID := range util.SortedStringKeys(gl.registry) {
		spec := gl.registry[langID]
		if !spec.Enabled {
			continue
		}
		switch langID {
		case "css":
			gl.languages["css"] = sitter.NewLanguage(tree_sitter_css.Language())
		case "go":
			gl.languages["go"] = sitter.NewLanguage(tree_sitter_go.Language())
		case "gomod", "gosum":
			// Parsed by raw-text extractors; no runtime tree-sitter binding required.
			continue
		case "html":
			gl.languages["html"] = sitter.NewLanguage(tree_sitter_html.Language())
		case "java":
			gl.languages["java"] = sitter.NewLanguage(tree_sitter_java.Language())
		case "javascript":
			gl.languages["javascript"] = sitter.NewLanguage(tree_sitter_javascript.Language())
		case "python":
			gl.languages["python"] = sitter.NewLanguage(tree_sitter_python.Language())
		case "rust":
			gl.languages["rust"] = sitter.NewLanguage(tree_sitter_rust.Language())
		case "tsx":
			gl.languages["tsx"] = sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
		case "typescript":
			gl.languages["typescript"] = sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
		default:
			return nil, fmt.Errorf("language %q is enabled but runtime grammar loading is not implemented", langID)
		}
	}

	return gl, nil
}

func (gl *GrammarLoader) LanguageRegistry() map[string]LanguageSpec {
	return cloneLanguageRegistry(gl.registry)
}

func (gl *GrammarLoader) SupportedExtensions() []string {
	set := make(map[string]bool)
	for _, spec := range gl.registry {
		if !spec.Enabled {
			continue
		}
		for _, ext := range spec.Extensions {
			set[ext] = true
		}
	}
	extensions := make([]string, 0, len(set))
	for ext := range set {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

func (gl *GrammarLoader) SupportedFilenames() []string {
	set := make(map[string]bool)
	for _, spec := range gl.registry {
		if !spec.Enabled {
			continue
		}
		for _, name := range spec.Filenames {
			set[stringsToLower(name)] = true
		}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (gl *GrammarLoader) SupportedTestFileSuffixes() []string {
	set := make(map[string]bool)
	for _, spec := range gl.registry {
		if !spec.Enabled {
			continue
		}
		for _, suffix := range spec.TestFileSuffixes {
			set[suffix] = true
		}
	}
	suffixes := make([]string, 0, len(set))
	for suffix := range set {
		suffixes = append(suffixes, suffix)
	}
	sort.Strings(suffixes)
	return suffixes
}

func stringsToLower(value string) string { return strings.ToLower(value) }

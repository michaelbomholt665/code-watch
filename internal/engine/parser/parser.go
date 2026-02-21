// # internal/parser/parser.go
package parser

import (
	"circular/internal/core/errors"
	"circular/internal/shared/util"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Parser struct {
	loader         *GrammarLoader
	extractors     map[string]Extractor // language -> extractor
	extensions     map[string]string
	filenames      map[string]string
	testFileSuffix []string
}

type Extractor interface {
	Extract(node *sitter.Node, source []byte, filePath string) (*File, error)
}

type RawExtractor interface {
	ExtractRaw(source []byte, filePath string) (*File, error)
}

func NewParser(loader *GrammarLoader) *Parser {
	p := &Parser{
		loader:     loader,
		extractors: make(map[string]Extractor),
		extensions: make(map[string]string),
		filenames:  make(map[string]string),
	}
	for lang, spec := range loader.LanguageRegistry() {
		if !spec.Enabled {
			continue
		}
		for _, ext := range spec.Extensions {
			p.extensions[strings.ToLower(ext)] = lang
		}
		for _, name := range spec.Filenames {
			p.filenames[strings.ToLower(path.Base(name))] = lang
		}
		p.testFileSuffix = append(p.testFileSuffix, spec.TestFileSuffixes...)
	}
	sort.Strings(p.testFileSuffix)
	return p
}

func (p *Parser) RegisterExtractor(lang string, e Extractor) {
	p.extractors[lang] = e
}

func (p *Parser) RegisterDefaultExtractors() error {
	for lang, spec := range p.loader.LanguageRegistry() {
		if !spec.Enabled {
			continue
		}
		extractor, ok := DefaultExtractorForLanguage(lang)
		if !ok {
			if spec.IsDynamic && spec.DynamicConfig != nil {
				p.RegisterExtractor(lang, NewDynamicExtractor(*spec.DynamicConfig))
				continue
			}
			return errors.New(errors.CodeNotSupported, fmt.Sprintf("no default extractor for enabled language: %s", lang))
		}
		p.RegisterExtractor(lang, extractor)
	}
	return nil
}

func (p *Parser) ParseFile(path string, content []byte) (*File, error) {
	lang := p.detectLanguage(path)
	if lang == "" {
		return nil, errors.New(errors.CodeNotSupported, "unsupported language")
	}

	extractor := p.extractors[lang]
	if extractor == nil {
		return nil, errors.New(errors.CodeNotSupported, fmt.Sprintf("no extractor for: %s", lang))
	}

	grammar := p.loader.languages[lang]
	if grammar == nil {
		if rawExtractor, ok := extractor.(RawExtractor); ok {
			return rawExtractor.ExtractRaw(content, path)
		}
		return nil, errors.New(errors.CodeInternal, fmt.Sprintf("grammar not loaded: %s", lang))
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(grammar)

	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, errors.New(errors.CodeInternal, "parse failed")
	}
	defer tree.Close()

	root := tree.RootNode()
	res, err := extractor.Extract(root, content, path)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "extraction failed")
	}
	return res, nil
}

func (p *Parser) detectLanguage(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if lang, ok := p.filenames[base]; ok {
		return lang
	}
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := p.extensions[ext]; ok {
		return lang
	}
	return ""
}

func (p *Parser) IsSupportedPath(filePath string) bool {
	return p.GetLanguage(filePath) != ""
}

func (p *Parser) GetLanguage(path string) string {
	return p.detectLanguage(path)
}

func (p *Parser) IsTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, suffix := range p.testFileSuffix {
		if strings.HasSuffix(base, strings.ToLower(suffix)) {
			return true
		}
	}
	return false
}

func (p *Parser) SupportedExtensions() []string {
	return util.SortedStringKeys(p.extensions)
}

func (p *Parser) SupportedFilenames() []string {
	return util.SortedStringKeys(p.filenames)
}

func (p *Parser) SupportedTestFileSuffixes() []string {
	out := make([]string, len(p.testFileSuffix))
	copy(out, p.testFileSuffix)
	return out
}

package parser

// Adapter bridges Parser to interface-based consumers.
type Adapter struct {
	parser *Parser
}

func NewAdapter(p *Parser) *Adapter {
	return &Adapter{parser: p}
}

func (a *Adapter) ParseFile(path string, content []byte) (*File, error) {
	return a.parser.ParseFile(path, content)
}

func (a *Adapter) GetLanguage(path string) string {
	return a.parser.GetLanguage(path)
}

func (a *Adapter) IsSupportedPath(filePath string) bool {
	return a.parser.IsSupportedPath(filePath)
}

func (a *Adapter) IsTestFile(path string) bool {
	return a.parser.IsTestFile(path)
}

func (a *Adapter) SupportedExtensions() []string {
	return a.parser.SupportedExtensions()
}

func (a *Adapter) SupportedFilenames() []string {
	return a.parser.SupportedFilenames()
}

func (a *Adapter) SupportedTestFileSuffixes() []string {
	return a.parser.SupportedTestFileSuffixes()
}

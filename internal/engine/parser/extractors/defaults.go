package extractors

import "circular/internal/engine/parser"

func Go() parser.Extractor {
	return &parser.GoExtractor{}
}

func Python() parser.Extractor {
	return &parser.PythonExtractor{}
}

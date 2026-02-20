// # internal/engine/parser/extractors/defaults.go
package extractors

import "circular/internal/engine/parser"

// Go returns the universal extractor for Go source files.
// GoExtractor has been retired; all languages now use the universal extractor.
func Go() parser.Extractor {
	return parser.NewUniversalExtractor()
}

// Python returns the universal extractor for Python source files.
// PythonExtractor has been retired; all languages now use the universal extractor.
func Python() parser.Extractor {
	return parser.NewUniversalExtractor()
}

package secrets

import (
	"circular/internal/core/ports"
	"circular/internal/engine/parser"
)

// Adapter bridges Detector to the core SecretScanner port.
type Adapter struct {
	detector *Detector
}

func NewAdapter(cfg Config) (*Adapter, error) {
	detector, err := NewDetector(cfg)
	if err != nil {
		return nil, err
	}
	return &Adapter{detector: detector}, nil
}

func NewAdapterFromDetector(detector *Detector) *Adapter {
	return &Adapter{detector: detector}
}

func (a *Adapter) Detect(filePath string, content []byte) []parser.Secret {
	return a.detector.Detect(filePath, content)
}

func (a *Adapter) DetectInRanges(filePath string, content []byte, ranges []ports.LineRange) []parser.Secret {
	if len(ranges) == 0 {
		return a.detector.Detect(filePath, content)
	}
	lineRanges := make([]LineRange, 0, len(ranges))
	for _, r := range ranges {
		lineRanges = append(lineRanges, LineRange{Start: r.Start, End: r.End})
	}
	return a.detector.DetectInRanges(filePath, content, lineRanges)
}

var _ ports.SecretScanner = (*Adapter)(nil)
var _ ports.IncrementalSecretScanner = (*Adapter)(nil)

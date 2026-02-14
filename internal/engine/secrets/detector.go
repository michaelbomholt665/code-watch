package secrets

import (
	"circular/internal/engine/parser"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type PatternConfig struct {
	Name     string
	Regex    string
	Severity string
}

type Config struct {
	EntropyThreshold float64
	MinTokenLength   int
	Patterns         []PatternConfig
}

type compiledPattern struct {
	name     string
	severity string
	re       *regexp.Regexp
}

type Detector struct {
	entropyThreshold float64
	minTokenLength   int
	patterns         []compiledPattern
	contextVarRE     *regexp.Regexp
	quotedValueRE    *regexp.Regexp
	quotedTokenRE    *regexp.Regexp
}

func NewDetector(cfg Config) (*Detector, error) {
	if cfg.EntropyThreshold <= 0 {
		cfg.EntropyThreshold = 4.0
	}
	if cfg.MinTokenLength <= 0 {
		cfg.MinTokenLength = 20
	}

	builtIn := []PatternConfig{
		{Name: "aws-access-key-id", Severity: "high", Regex: `\bAKIA[0-9A-Z]{16}\b`},
		{Name: "github-pat", Severity: "high", Regex: `\bghp_[A-Za-z0-9]{36}\b`},
		{Name: "github-fine-grained-pat", Severity: "high", Regex: `\bgithub_pat_[A-Za-z0-9_]{82}\b`},
		{Name: "stripe-live-secret", Severity: "high", Regex: `\bsk_live_[A-Za-z0-9]{16,}\b`},
		{Name: "slack-token", Severity: "high", Regex: `\bxox[baprs]-[A-Za-z0-9-]{10,}\b`},
		{Name: "private-key-block", Severity: "critical", Regex: `-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----`},
	}

	patterns, err := compilePatterns(append(builtIn, cfg.Patterns...))
	if err != nil {
		return nil, err
	}

	return &Detector{
		entropyThreshold: cfg.EntropyThreshold,
		minTokenLength:   cfg.MinTokenLength,
		patterns:         patterns,
		contextVarRE:     regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|api[_-]?key|token|auth[_-]?token|access[_-]?key|private[_-]?key|client[_-]?secret)\b`),
		quotedValueRE:    regexp.MustCompile(`"([^"\r\n]{4,})"|'([^'\r\n]{4,})'`),
		quotedTokenRE:    regexp.MustCompile(`"([A-Za-z0-9_\-+=:/.]{12,})"|'([A-Za-z0-9_\-+=:/.]{12,})'`),
	}, nil
}

func (d *Detector) Detect(filePath string, content []byte) []parser.Secret {
	if len(content) == 0 {
		return nil
	}

	text := string(content)
	index := buildLineIndex(content)
	findings := make(map[string]parser.Secret)

	d.detectPatternMatches(filePath, text, index, findings)
	d.detectContextMatches(filePath, text, index, findings)
	d.detectEntropyMatches(filePath, text, index, findings)

	if len(findings) == 0 {
		return nil
	}

	out := make([]parser.Secret, 0, len(findings))
	for _, secret := range findings {
		out = append(out, secret)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Location.File != out[j].Location.File {
			return out[i].Location.File < out[j].Location.File
		}
		if out[i].Location.Line != out[j].Location.Line {
			return out[i].Location.Line < out[j].Location.Line
		}
		if out[i].Location.Column != out[j].Location.Column {
			return out[i].Location.Column < out[j].Location.Column
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func (d *Detector) detectPatternMatches(filePath, text string, index lineIndex, findings map[string]parser.Secret) {
	for _, pattern := range d.patterns {
		locs := pattern.re.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			value := text[loc[0]:loc[1]]
			if shouldIgnoreCandidate(value) {
				continue
			}
			line, col := index.lineCol(loc[0])
			secret := parser.Secret{
				Kind:       pattern.name,
				Severity:   pattern.severity,
				Value:      value,
				Entropy:    shannonEntropy(value),
				Confidence: 0.99,
				Location: parser.Location{
					File:   filePath,
					Line:   line,
					Column: col,
				},
			}
			upsertFinding(findings, secret)
		}
	}
}

func (d *Detector) detectContextMatches(filePath, text string, index lineIndex, findings map[string]parser.Secret) {
	offset := 0
	for _, line := range strings.Split(text, "\n") {
		if !d.contextVarRE.MatchString(line) {
			offset += len(line) + 1
			continue
		}
		for _, match := range d.quotedValueRE.FindAllStringSubmatchIndex(line, -1) {
			valueStart, valueEnd, ok := firstMatchedGroup(match)
			if !ok {
				continue
			}
			candidate := line[valueStart:valueEnd]
			if len(candidate) < d.minTokenLength || shouldIgnoreCandidate(candidate) {
				continue
			}
			entropy := shannonEntropy(candidate)
			if entropy < (d.entropyThreshold * 0.8) {
				continue
			}
			globalStart := offset + valueStart
			ln, col := index.lineCol(globalStart)
			confidence := 0.70
			if entropy >= d.entropyThreshold {
				confidence = 0.85
			}
			secret := parser.Secret{
				Kind:       "sensitive-assignment",
				Severity:   "medium",
				Value:      candidate,
				Entropy:    entropy,
				Confidence: confidence,
				Location: parser.Location{
					File:   filePath,
					Line:   ln,
					Column: col,
				},
			}
			upsertFinding(findings, secret)
		}
		offset += len(line) + 1
	}
}

func (d *Detector) detectEntropyMatches(filePath, text string, index lineIndex, findings map[string]parser.Secret) {
	for _, match := range d.quotedTokenRE.FindAllStringSubmatchIndex(text, -1) {
		valueStart, valueEnd, ok := firstMatchedGroup(match)
		if !ok {
			continue
		}
		candidate := text[valueStart:valueEnd]
		if len(candidate) < d.minTokenLength || shouldIgnoreCandidate(candidate) {
			continue
		}
		if !containsLetterAndDigit(candidate) {
			continue
		}
		entropy := shannonEntropy(candidate)
		if entropy < d.entropyThreshold {
			continue
		}
		line, col := index.lineCol(valueStart)
		secret := parser.Secret{
			Kind:       "high-entropy-string",
			Severity:   "low",
			Value:      candidate,
			Entropy:    entropy,
			Confidence: 0.6,
			Location: parser.Location{
				File:   filePath,
				Line:   line,
				Column: col,
			},
		}
		upsertFinding(findings, secret)
	}
}

func compilePatterns(cfg []PatternConfig) ([]compiledPattern, error) {
	compiled := make([]compiledPattern, 0, len(cfg))
	for _, pattern := range cfg {
		name := strings.TrimSpace(pattern.Name)
		if name == "" {
			return nil, fmt.Errorf("secret pattern name must not be empty")
		}
		expr := strings.TrimSpace(pattern.Regex)
		if expr == "" {
			return nil, fmt.Errorf("secret pattern %q regex must not be empty", name)
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("compile secret pattern %q: %w", name, err)
		}
		severity := strings.ToLower(strings.TrimSpace(pattern.Severity))
		if severity == "" {
			severity = "medium"
		}
		compiled = append(compiled, compiledPattern{name: name, severity: severity, re: re})
	}
	return compiled, nil
}

func upsertFinding(findings map[string]parser.Secret, candidate parser.Secret) {
	key := fmt.Sprintf("%s:%d:%d:%s", candidate.Location.File, candidate.Location.Line, candidate.Location.Column, candidate.Value)
	if existing, ok := findings[key]; ok {
		if existing.Confidence >= candidate.Confidence {
			return
		}
	}
	findings[key] = candidate
}

func containsLetterAndDigit(value string) bool {
	hasLetter := false
	hasDigit := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

func shouldIgnoreCandidate(value string) bool {
	lower := strings.ToLower(value)
	for _, blocked := range []string{"example", "sample", "dummy", "placeholder", "changeme", "notasecret", "test"} {
		if strings.Contains(lower, blocked) {
			return true
		}
	}
	return false
}

func shannonEntropy(value string) float64 {
	if value == "" {
		return 0
	}
	freq := make(map[rune]float64)
	for _, r := range value {
		freq[r]++
	}
	length := float64(len([]rune(value)))
	if length == 0 {
		return 0
	}
	entropy := 0.0
	for _, count := range freq {
		p := count / length
		entropy -= p * math.Log2(p)
	}
	return entropy
}

type lineIndex struct {
	starts []int
}

func buildLineIndex(content []byte) lineIndex {
	starts := []int{0}
	for i, b := range content {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return lineIndex{starts: starts}
}

func (i lineIndex) lineCol(offset int) (int, int) {
	if offset < 0 {
		return 1, 1
	}
	line := sort.Search(len(i.starts), func(idx int) bool { return i.starts[idx] > offset }) - 1
	if line < 0 {
		line = 0
	}
	col := (offset - i.starts[line]) + 1
	if col < 1 {
		col = 1
	}
	return line + 1, col
}

func MaskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func firstMatchedGroup(match []int) (int, int, bool) {
	for i := 2; i+1 < len(match); i += 2 {
		if match[i] >= 0 && match[i+1] >= 0 {
			return match[i], match[i+1], true
		}
	}
	return 0, 0, false
}

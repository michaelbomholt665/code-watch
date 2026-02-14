package resolver

import (
	"circular/internal/engine/parser"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

const bridgeConfigFilename = ".circular-bridge.toml"

type bridgeConfigFile struct {
	Bridges []bridgeConfigEntry `toml:"bridges"`
}

type bridgeConfigEntry struct {
	From       string   `toml:"from"`
	To         string   `toml:"to"`
	Reason     string   `toml:"reason"`
	References []string `toml:"references"`
}

type ExplicitBridge struct {
	FromLanguage string
	FromModule   string
	ToLanguage   string
	ToModule     string
	Reason       string
	References   []string
}

func LoadBridgeConfig(path string) ([]ExplicitBridge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bridge config %q: %w", path, err)
	}

	var cfg bridgeConfigFile
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("decode bridge config %q: %w", path, err)
	}

	out := make([]ExplicitBridge, 0, len(cfg.Bridges))
	for i, entry := range cfg.Bridges {
		bridge, err := toExplicitBridge(entry)
		if err != nil {
			return nil, fmt.Errorf("bridge[%d]: %w", i, err)
		}
		out = append(out, bridge)
	}
	return out, nil
}

func DiscoverBridgeConfigPaths(roots []string) []string {
	seen := make(map[string]bool, len(roots))
	paths := make([]string, 0, len(roots)+1)

	for _, root := range roots {
		candidate := filepath.Join(root, bridgeConfigFilename)
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if seen[abs] {
			continue
		}
		seen[abs] = true
		paths = append(paths, abs)
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(cwd, bridgeConfigFilename)
		abs := filepath.Clean(candidate)
		if !seen[abs] {
			paths = append(paths, abs)
		}
	}

	return paths
}

func (r *Resolver) WithExplicitBridges(bridges []ExplicitBridge) *Resolver {
	if r == nil {
		return nil
	}
	if len(bridges) == 0 {
		r.explicitBridges = nil
		return r
	}
	r.explicitBridges = append([]ExplicitBridge(nil), bridges...)
	return r
}

func (r *Resolver) resolveExplicitBridgeReference(file *parser.File, ref parser.Reference) bool {
	if r == nil || len(r.explicitBridges) == 0 || file == nil {
		return false
	}

	for _, bridge := range r.explicitBridges {
		if !bridge.matchesSource(file.Language, file.Module) {
			continue
		}
		if bridge.matchesReference(ref) {
			return true
		}
	}
	return false
}

func toExplicitBridge(entry bridgeConfigEntry) (ExplicitBridge, error) {
	fromLang, fromModule, err := parseBridgeEndpoint(entry.From)
	if err != nil {
		return ExplicitBridge{}, fmt.Errorf("invalid from endpoint: %w", err)
	}
	toLang, toModule, err := parseBridgeEndpoint(entry.To)
	if err != nil {
		return ExplicitBridge{}, fmt.Errorf("invalid to endpoint: %w", err)
	}

	refs := make([]string, 0, len(entry.References)+1)
	for _, ref := range entry.References {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		refs = append(refs, ref)
	}
	if inferred := inferredBridgeReference(toModule); inferred != "" {
		refs = append(refs, inferred)
	}

	return ExplicitBridge{
		FromLanguage: fromLang,
		FromModule:   fromModule,
		ToLanguage:   toLang,
		ToModule:     toModule,
		Reason:       strings.TrimSpace(entry.Reason),
		References:   dedupeStrings(refs),
	}, nil
}

func parseBridgeEndpoint(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("endpoint must not be empty")
	}

	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("endpoint %q must be in <language>:<module> format", raw)
	}

	lang := strings.ToLower(strings.TrimSpace(parts[0]))
	module := strings.TrimSpace(parts[1])
	if lang == "" {
		return "", "", fmt.Errorf("language must not be empty")
	}
	if module == "" {
		return "", "", fmt.Errorf("module must not be empty")
	}
	return lang, module, nil
}

func (b ExplicitBridge) matchesSource(language, module string) bool {
	if b.FromLanguage != "" && !strings.EqualFold(b.FromLanguage, strings.TrimSpace(language)) {
		return false
	}
	return matchBridgePattern(b.FromModule, module)
}

func (b ExplicitBridge) matchesReference(ref parser.Reference) bool {
	if ref.Context == parser.RefContextFFI || ref.Context == parser.RefContextProcess || ref.Context == parser.RefContextService {
		return true
	}
	name := strings.TrimSpace(ref.Name)
	if name == "" {
		return false
	}

	for _, pattern := range b.References {
		if matchBridgePattern(pattern, name) {
			return true
		}
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}
	}
	return false
}

func matchBridgePattern(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	if pattern == "" || value == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if strings.EqualFold(pattern, value) {
		return true
	}

	regex := "^" + regexp.QuoteMeta(pattern) + "$"
	regex = strings.ReplaceAll(regex, "\\*", ".*")
	ok, err := regexp.MatchString(regex, value)
	if err != nil {
		return false
	}
	return ok
}

func inferredBridgeReference(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return ""
	}
	module = strings.ReplaceAll(module, "::", "/")
	module = strings.ReplaceAll(module, ".", "/")
	parts := strings.Split(module, "/")
	if len(parts) == 0 {
		return ""
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	if last == "" {
		return ""
	}
	return last + ".*"
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

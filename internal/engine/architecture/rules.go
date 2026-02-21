package architecture

import (
	"circular/internal/core/ports"
	"circular/internal/shared/util"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobwas/glob"
)

type RuleSet struct {
	rules []Rule
}

type Rule struct {
	Name         string
	Modules      []compiledPattern
	MaxFiles     int
	ImportAllow  []compiledPattern
	ImportDeny   []compiledPattern
	ExcludeFiles []compiledPattern
	ExcludeTests bool
}

type compiledPattern struct {
	raw        string
	isWildcard bool
	glob       glob.Glob
}

func NewRuleSet(rules []ports.ArchitectureRule) RuleSet {
	out := RuleSet{rules: make([]Rule, 0, len(rules))}
	for _, rule := range rules {
		if rule.Kind != ports.ArchitectureRuleKindPackage {
			continue
		}
		r := Rule{
			Name:         rule.Name,
			MaxFiles:     rule.MaxFiles,
			ExcludeTests: rule.Exclude.Tests,
		}
		r.Modules = compilePatterns(rule.Modules)
		r.ImportAllow = compilePatterns(rule.Imports.Allow)
		r.ImportDeny = compilePatterns(rule.Imports.Deny)
		r.ExcludeFiles = compilePatterns(rule.Exclude.Files)
		out.rules = append(out.rules, r)
	}
	return out
}

func (r RuleSet) Rules() []Rule {
	out := make([]Rule, 0, len(r.rules))
	out = append(out, r.rules...)
	return out
}

func (r Rule) MatchesModule(moduleName string) bool {
	return matchPatterns(r.Modules, moduleName, "")
}

func (r Rule) AllowsImport(target string) bool {
	if matchPatterns(r.ImportDeny, target, "") {
		return false
	}
	if len(r.ImportAllow) == 0 {
		return true
	}
	return matchPatterns(r.ImportAllow, target, "")
}

func (r Rule) ExcludesFile(path string) bool {
	if r.ExcludeTests && isTestFile(path) {
		return true
	}
	if matchPatterns(r.ExcludeFiles, path, filepath.Base(path)) {
		return true
	}
	return false
}

func compilePatterns(raw []string) []compiledPattern {
	if len(raw) == 0 {
		return nil
	}
	out := make([]compiledPattern, 0, len(raw))
	for _, pattern := range raw {
		norm := util.NormalizePatternPath(pattern)
		if norm == "" {
			continue
		}
		cp := compiledPattern{
			raw:        norm,
			isWildcard: strings.ContainsAny(norm, "*?[]{}"),
		}
		if cp.isWildcard {
			if g, err := glob.Compile(norm, '/'); err == nil {
				cp.glob = g
			} else {
				continue
			}
		}
		out = append(out, cp)
	}
	return out
}

func matchPatterns(patterns []compiledPattern, moduleName, samplePath string) bool {
	if len(patterns) == 0 {
		return false
	}
	modName := util.NormalizePatternPath(moduleName)
	path := util.NormalizePatternPath(samplePath)
	for _, p := range patterns {
		if p.isWildcard {
			if p.glob != nil && p.glob.Match(modName) {
				return true
			}
			if p.glob != nil && path != "" && p.glob.Match(path) {
				return true
			}
			continue
		}
		if util.HasPathPrefix(modName, p.raw) {
			return true
		}
		if path != "" && util.HasPathPrefix(path, p.raw) {
			return true
		}
	}
	return false
}

func isTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.py") {
		return true
	}
	if strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") {
		return true
	}
	return false
}

func sortRules(rules []Rule) {
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Name == rules[j].Name {
			return len(rules[i].Modules) < len(rules[j].Modules)
		}
		return rules[i].Name < rules[j].Name
	})
}

package graph

import (
	"circular/internal/shared/util"
	"fmt"
	"sort"
	"strings"

	"github.com/gobwas/glob"
)

type ArchitectureModel struct {
	Enabled bool
	Layers  []ArchitectureLayer
	Rules   []ArchitectureRule
}

type ArchitectureLayer struct {
	Name  string
	Paths []string
}

type ArchitectureRule struct {
	Name  string
	From  string
	Allow []string
}

type ArchitectureViolation struct {
	RuleName   string
	FromModule string
	FromLayer  string
	ToModule   string
	ToLayer    string
	File       string
	Line       int
	Column     int
}

type LayerRuleEngine struct {
	enabled bool
	layers  []layerMatcher
	rules   map[string]ruleSet
}

type layerMatcher struct {
	name     string
	patterns []compiledPattern
}

type compiledPattern struct {
	raw        string
	isWildcard bool
	glob       glob.Glob
}

type ruleSet struct {
	name  string
	allow map[string]bool
}

func NewLayerRuleEngine(model ArchitectureModel) *LayerRuleEngine {
	engine := &LayerRuleEngine{
		enabled: model.Enabled,
		rules:   make(map[string]ruleSet),
	}

	for _, layer := range model.Layers {
		matcher := layerMatcher{name: layer.Name}
		for _, raw := range layer.Paths {
			pattern := util.NormalizePatternPath(raw)
			cp := compiledPattern{
				raw:        pattern,
				isWildcard: strings.ContainsAny(pattern, "*?[]{}"),
			}
			if cp.isWildcard {
				if g, err := glob.Compile(pattern, '/'); err == nil {
					cp.glob = g
				} else {
					continue
				}
			}
			matcher.patterns = append(matcher.patterns, cp)
		}
		engine.layers = append(engine.layers, matcher)
	}

	for _, rule := range model.Rules {
		allow := make(map[string]bool, len(rule.Allow))
		for _, target := range rule.Allow {
			allow[target] = true
		}
		engine.rules[rule.From] = ruleSet{name: rule.Name, allow: allow}
	}

	return engine
}

func (e *LayerRuleEngine) Validate(g *Graph) []ArchitectureViolation {
	if e == nil || !e.enabled {
		return nil
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	moduleLayer := make(map[string]string, len(g.modules))
	for modName, mod := range g.modules {
		moduleLayer[modName] = e.layerFor(modName, mod)
	}

	fromMods := util.SortedStringKeys(g.imports)
	violations := make([]ArchitectureViolation, 0)
	for _, from := range fromMods {
		toMap := g.imports[from]
		if len(toMap) == 0 {
			continue
		}

		fromLayer := moduleLayer[from]
		rule, hasRule := e.rules[fromLayer]
		if !hasRule {
			continue
		}

		toMods := util.SortedStringKeys(toMap)
		for _, to := range toMods {
			toLayer := moduleLayer[to]
			if toLayer == "" {
				continue
			}
			if rule.allow[toLayer] {
				continue
			}

			edge := toMap[to]
			violations = append(violations, ArchitectureViolation{
				RuleName:   rule.name,
				FromModule: from,
				FromLayer:  fromLayer,
				ToModule:   to,
				ToLayer:    toLayer,
				File:       edge.ImportedBy,
				Line:       edge.Location.Line,
				Column:     edge.Location.Column,
			})
		}
	}

	return violations
}

func (e *LayerRuleEngine) layerFor(moduleName string, mod *Module) string {
	type candidate struct {
		layer string
		score int
	}

	candidates := make([]candidate, 0)
	samplePath := ""
	if mod != nil && len(mod.Files) > 0 {
		files := append([]string(nil), mod.Files...)
		sort.Strings(files)
		samplePath = util.NormalizePatternPath(files[0])
	}
	modName := util.NormalizePatternPath(moduleName)

	for _, layer := range e.layers {
		best := 0
		for _, p := range layer.patterns {
			if matchPattern(p, modName, samplePath) {
				if l := len(p.raw); l > best {
					best = l
				}
			}
		}
		if best > 0 {
			candidates = append(candidates, candidate{layer: layer.name, score: best})
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].layer < candidates[j].layer
		}
		return candidates[i].score > candidates[j].score
	})
	return candidates[0].layer
}

func matchPattern(p compiledPattern, moduleName, samplePath string) bool {
	if p.isWildcard {
		if p.glob != nil && p.glob.Match(moduleName) {
			return true
		}
		return p.glob != nil && samplePath != "" && p.glob.Match(samplePath)
	}

	if util.HasPathPrefix(moduleName, p.raw) {
		return true
	}
	return samplePath != "" && util.HasPathPrefix(samplePath, p.raw)
}

func (v ArchitectureViolation) String() string {
	return fmt.Sprintf("%s (%s -> %s): %s imports %s", v.RuleName, v.FromLayer, v.ToLayer, v.FromModule, v.ToModule)
}

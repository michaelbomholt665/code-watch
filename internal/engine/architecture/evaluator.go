package architecture

import (
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"circular/internal/shared/util"
	"sort"
)

type RuleEvaluator struct {
	rules RuleSet
}

type EvaluationResult struct {
	Violations       []ports.ArchitectureRuleViolation
	EvaluatedModules int
}

func NewRuleEvaluator(rules []ports.ArchitectureRule) *RuleEvaluator {
	rs := NewRuleSet(rules)
	ruleList := rs.rules
	sortRules(ruleList)
	rs.rules = ruleList
	return &RuleEvaluator{rules: rs}
}

func (e *RuleEvaluator) Evaluate(g *graph.Graph) EvaluationResult {
	if e == nil || len(e.rules.rules) == 0 || g == nil {
		return EvaluationResult{}
	}

	modules := g.Modules()
	imports := g.GetImports()
	moduleNames := util.SortedStringKeys(modules)
	violations := make([]ports.ArchitectureRuleViolation, 0)
	evaluated := 0

	for _, moduleName := range moduleNames {
		mod := modules[moduleName]
		matched := false
		for _, rule := range e.rules.rules {
			if !rule.MatchesModule(moduleName) {
				continue
			}
			matched = true
			if rule.MaxFiles > 0 {
				count := countFiles(mod, rule)
				if count > rule.MaxFiles {
					violations = append(violations, ports.ArchitectureRuleViolation{
						RuleName: rule.Name,
						RuleKind: ports.ArchitectureRuleKindPackage,
						Module:   moduleName,
						Type:     "file_count",
						Message:  "module exceeds file-count limit",
						Limit:    rule.MaxFiles,
						Actual:   count,
					})
				}
			}
			if len(rule.ImportAllow) > 0 || len(rule.ImportDeny) > 0 {
				targets := imports[moduleName]
				if len(targets) == 0 {
					continue
				}
				targetNames := util.SortedStringKeys(targets)
				for _, target := range targetNames {
					if _, ok := modules[target]; !ok {
						continue
					}
					edge := targets[target]
					allowed := rule.AllowsImport(target)
					if allowed {
						continue
					}
					violations = append(violations, ports.ArchitectureRuleViolation{
						RuleName: rule.Name,
						RuleKind: ports.ArchitectureRuleKindPackage,
						Module:   moduleName,
						Target:   target,
						Type:     "import",
						Message:  "import violates rule policy",
						File:     edge.ImportedBy,
						Line:     edge.Location.Line,
						Column:   edge.Location.Column,
					})
				}
			}
		}
		if matched {
			evaluated++
		}
	}

	return EvaluationResult{
		Violations:       violations,
		EvaluatedModules: evaluated,
	}
}

func countFiles(mod *graph.Module, rule Rule) int {
	if mod == nil || len(mod.Files) == 0 {
		return 0
	}
	files := append([]string(nil), mod.Files...)
	sort.Strings(files)
	count := 0
	for _, path := range files {
		if rule.ExcludesFile(path) {
			continue
		}
		count++
	}
	return count
}

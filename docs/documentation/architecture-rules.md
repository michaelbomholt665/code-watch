# Architecture Rules

Architecture rules let you enforce package/module guardrails beyond layer-to-layer rules. They are evaluated against the module graph and can:
- limit the number of files in a module (after rule-specific excludes)
- constrain which modules a target module may import

Rules are optional. They run alongside layer rules when `[architecture].enabled = true`.

## Configuration

Package rules live under `[[architecture.rules]]` with `kind = "package"`:

```toml
[architecture]
enabled = true

[[architecture.rules]]
name = "api-size"
kind = "package"
modules = ["internal/api"]
max_files = 30
exclude = { tests = true, files = ["index.ts", "__init__.py"] }

[[architecture.rules]]
name = "api-imports"
kind = "package"
modules = ["internal/api"]
imports = { allow = ["internal/core/**"], deny = ["internal/engine/**"] }
```

### Fields
- `name`: unique rule name.
- `kind`: `package` for package/module rules.
- `modules`: list of module patterns to match (prefix or glob).
- `max_files`: file-count limit after excludes. Omit or set `0` to disable.
- `imports.allow`: allow-list of module patterns. When set, any import not matching the list is a violation.
- `imports.deny`: deny-list of module patterns. Deny rules always win.
- `exclude.tests`: exclude test files (`*_test.go`, `_test.py`, `test_*.py`).
- `exclude.files`: exclude file patterns by path or name (globs supported).

## Outputs

When rules are configured:
- CLI summary prints rule counts and violations.
- Markdown report includes **Architecture Rules** and **Architecture Rule Violations** sections.
- TSV output includes `architecture_rule_violation` rows.
- SARIF uses `CIRC004` for architecture rule violations.

## Matching Rules

Module patterns match both module names and representative file paths:
- literal prefixes match module names by path prefix
- wildcard patterns (`*`, `**`, `?`) use glob matching

Import rules only apply to modules that are part of the graph (external deps are ignored).

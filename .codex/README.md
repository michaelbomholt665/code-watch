Task routing wrapper for Codex CLI.

Command:
- `scripts/codex-task [--mode auto|go-dev|review|pm] [--show-route] "<prompt>"`

Modes:
- `go-dev`: profile `build`, skill `$go-code-watch-maintainer`, persona `.codex/persona-go-coder.md`
- `review`: profile `review`, skill `$go-defect-reviewer`, persona `.codex/persona-code-reviewer.md`
- `pm`: profile `deep`, skill `$project-planner`, persona `.codex/persona-project-manager.md`

Auto routing keywords:
- `review` mode: `review`, `audit`, `risk`, `regression`, `defect`
- `pm` mode: `plan`, `roadmap`, `milestone`, `scope`, `project`, `migration`
- otherwise falls back to `go-dev`

Dry run:
- `scripts/codex-task --dry-run "review latest changes"`

Show routing on real execution:
- `scripts/codex-task --show-route "plan migration to architecture rules v2"`

Makefile shortcuts:
- `make task PROMPT="fix resolver edge case"` (auto mode)
- `make task pm PROMPT="plan config migration"` (goal mode)
- `make task-go PROMPT="implement watcher fix"`
- `make task-review PROMPT="review latest changes"`
- `make task-pm PROMPT="plan MCP config rollout"`

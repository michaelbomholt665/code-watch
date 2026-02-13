---
name: changelog-maintainer
description: Maintain the project's root CHANGELOG.md with accurate, user-facing release notes for added, changed, fixed, removed, and documentation updates. Use when implementing features, bug fixes, refactors, docs, or version bumps in this repository and a changelog entry must be created or updated.
---

# Changelog Maintainer

Maintain `CHANGELOG.md` in the repository root as part of normal delivery.

## Workflow

1. Check whether `CHANGELOG.md` exists at repository root.
2. Create it if missing using this structure:
- Title: `# Changelog`
- Intro line: `All notable changes to this project will be documented in this file.`
- Sections ordered newest-first by release heading.
3. Add/update the current release section with date (`YYYY-MM-DD`) and grouped bullets:
- `Added`
- `Changed`
- `Fixed`
- `Removed`
- `Docs`
4. Keep entries concise, user-facing, and behavior-oriented.
5. Include concrete references when useful (CLI flags, output files, package paths).
6. Avoid internal-only noise (temporary refactors, test-only churn) unless user impact exists.
7. Ensure no duplicate bullets across categories.
8. Keep markdown clean and stable (single top-level title, consistent heading levels, no trailing whitespace).

## Entry Rules

- Write in past tense and imperative-neutral style.
- Prefer one-line bullets with scope prefix when helpful (example: `graph:` or `cli:`).
- Mention breaking changes explicitly under `Changed`.
- Mention config/output contract changes explicitly (`circular.toml`, `graph.dot`, `dependencies.tsv`).
- If no user-visible change exists, add no changelog entry unless requested.

## Quality Gate

Before finalizing:

```bash
[ -f CHANGELOG.md ]
rg -n "^# Changelog|^## |^### (Added|Changed|Fixed|Removed|Docs)$" CHANGELOG.md
```

## Do and Don't

Do:
- Keep the newest release section first.
- Keep wording actionable and scannable.
- Keep release dates concrete.

Don't:
- Invent versions or dates.
- Add speculative roadmap items.
- Copy commit messages verbatim when they are not user-facing.

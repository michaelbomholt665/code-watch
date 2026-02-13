---
name: go-defect-reviewer
description: Review Go code for defect risk with emphasis on missing error handling, silent error handling, resolver correctness issues, memory/resource leaks, and other crash or performance hazards. Use when auditing changes, performing code review, or investigating production reliability issues in this repository.
---

# Go Defect Reviewer

Run defect-oriented reviews that prioritize correctness and runtime safety over style.

## Execute This Workflow

1. Define review scope from changed files and owning packages.
2. Run fast validation first, then deeper checks.
3. Produce findings ordered by severity with concrete file references.
4. Include residual risks and missing test coverage.

## Run Verification Commands

Run in this order:

```bash
go test ./...
go test -race ./...
go vet ./...
```

If available, also run:

```bash
staticcheck ./...
```

If a command cannot run, state the blocker and resulting risk.

## Check For Missing Error Handling

- Find ignored return errors (`_ =`, dropped `err`, unchecked function results).
- Verify file/network/process operations always handle failures.
- Verify deferred cleanup calls (`Close`, `Stop`, `Cancel`) return handling where relevant.
- Verify wrapped errors preserve actionable context.

## Check For Silent Error Handling

- Flag branches that swallow errors and continue without clear signal.
- Flag broad `recover` usage that hides panics without escalation.
- Flag logging-only paths that should return or surface failure.
- Flag retries/timeouts that discard terminal failure details.

## Check Resolver Correctness Risks

Focus on `internal/resolver` and parser-to-resolver boundaries:

- Validate alias/import resolution and symbol normalization paths.
- Validate nil/empty input handling and default fallbacks.
- Validate deterministic behavior for maps/slices that affect output.
- Validate unresolved-reference classification avoids false positives/negatives.
- Validate cross-language handling (Go/Python) stays isolated and explicit.

## Check For Memory And Resource Leaks

- Detect goroutines without cancellation, join, or lifetime boundaries.
- Detect `time.Ticker`/`time.Timer` without `Stop`.
- Detect watcher/channel lifecycle leaks (`fsnotify`, never-closed channels, blocked send/recv).
- Detect unbounded caches/maps/slices in long-running watch mode.
- Detect file or parser resources not closed/released.

## Check For Crash And Performance Hazards

- Panic-prone paths (`nil` dereference, unchecked type assertions, unsafe indexing).
- Data races and unsafe shared state mutation.
- Hot-loop allocations, repeated parsing without caching, and N^2 traversals.
- Non-deterministic output that causes flaky tests or unstable reports.
- Missing boundary tests for malformed input, empty projects, and large repos.

## Report Format

Report findings first, ordered by severity:

1. Severity (`critical`, `high`, `medium`, `low`)
2. Location (`path:line`)
3. Failure mode and impact
4. Minimal fix direction
5. Test gap to close regression risk

If no defects are found, explicitly state that and include remaining risk areas not fully verified.

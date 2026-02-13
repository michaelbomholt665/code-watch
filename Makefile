.PHONY: \
	help \
	build \
	run \
	watch \
	once \
	ui \
	trace \
	impact \
	history \
	history-export \
	query-modules \
	query-module \
	query-trace \
	query-trends \
	task \
	task-go \
	task-review \
	task-pm \
	fmt \
	coverage \
	test \
	test-offline \
	clean

CACHE_ROOT := $(CURDIR)/.cache
GO_BUILD_CACHE := $(CACHE_ROOT)/go-build
GO_TMP := $(CACHE_ROOT)/go-tmp
GO_MOD_CACHE := $(CACHE_ROOT)/go-mod
GO_ENV := GOCACHE=$(GO_BUILD_CACHE) GOTMPDIR=$(GO_TMP) GOMODCACHE=$(GO_MOD_CACHE)

BINARY ?= circular
CMD_PKG ?= ./cmd/circular
CONFIG ?= ./circular.toml
PATH_ARG ?=
TRACE_FROM ?=
TRACE_TO ?=
IMPACT ?=
QUERY_FILTER ?=
QUERY_MODULE ?=
QUERY_TRACE ?=
QUERY_LIMIT ?=0
SINCE ?=
HISTORY_WINDOW ?=24h
HISTORY_TSV ?=out/trends.tsv
HISTORY_JSON ?=out/trends.json
PROMPT ?=
MODE ?=
SHOW_ROUTE ?=1

# Supports: make task pm PROMPT="..."
ifneq ($(filter task,$(firstword $(MAKECMDGOALS))),)
TASK_GOAL_MODE := $(word 2,$(MAKECMDGOALS))
ifneq ($(TASK_GOAL_MODE),)
MODE := $(TASK_GOAL_MODE)
$(eval $(TASK_GOAL_MODE):;@:)
endif
endif

PATH_OPT := $(if $(PATH_ARG),$(PATH_ARG),)
SINCE_OPT := $(if $(SINCE),--since $(SINCE),)

help: ## Show available targets and common overrides
	@echo "Usage: make <target> [VAR=value]"
	@echo ""
	@echo "Common vars: CONFIG, PATH_ARG, SINCE, QUERY_LIMIT, TRACE_FROM, TRACE_TO, IMPACT, PROMPT, MODE, SHOW_ROUTE"
	@echo ""
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build CLI binary (BINARY, CMD_PKG)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go build -o $(BINARY) $(CMD_PKG)

run: watch ## Alias for watch mode

watch: ## Run default watch mode
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) $(PATH_OPT)

once: ## Run single scan and exit
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --once $(PATH_OPT)

ui: ## Run watch mode with terminal UI
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --ui $(PATH_OPT)

trace: ## Trace shortest chain (requires TRACE_FROM, TRACE_TO)
	@test -n "$(TRACE_FROM)" || (echo "TRACE_FROM is required"; exit 1)
	@test -n "$(TRACE_TO)" || (echo "TRACE_TO is required"; exit 1)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --trace $(TRACE_FROM) $(TRACE_TO)

impact: ## Analyze impact (requires IMPACT=file-or-module)
	@test -n "$(IMPACT)" || (echo "IMPACT is required"; exit 1)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --impact $(IMPACT) $(PATH_OPT)

history: ## Run one scan with history/trends (optional SINCE, HISTORY_WINDOW)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --once --history --history-window $(HISTORY_WINDOW) $(SINCE_OPT) $(PATH_OPT)

history-export: ## Export history trends to TSV/JSON (optional SINCE)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --once --history --history-window $(HISTORY_WINDOW) --history-tsv $(HISTORY_TSV) --history-json $(HISTORY_JSON) $(SINCE_OPT) $(PATH_OPT)

query-modules: ## Query module list (optional QUERY_FILTER, QUERY_LIMIT)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --query-modules --query-filter "$(QUERY_FILTER)" --query-limit $(QUERY_LIMIT) $(PATH_OPT)

query-module: ## Query one module (requires QUERY_MODULE)
	@test -n "$(QUERY_MODULE)" || (echo "QUERY_MODULE is required"; exit 1)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --query-module $(QUERY_MODULE) --query-limit $(QUERY_LIMIT) $(PATH_OPT)

query-trace: ## Query dependency trace (requires QUERY_TRACE=from:to)
	@test -n "$(QUERY_TRACE)" || (echo "QUERY_TRACE is required (format from:to)"; exit 1)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --query-trace $(QUERY_TRACE) --query-limit $(QUERY_LIMIT) $(PATH_OPT)

query-trends: ## Query historical trends (supports SINCE, QUERY_LIMIT)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go run $(CMD_PKG) --config $(CONFIG) --history --query-trends --query-limit $(QUERY_LIMIT) $(SINCE_OPT) $(PATH_OPT)

task: ## Run codex-task wrapper (MODE=auto|go-dev|review|pm, PROMPT required). Also supports: make task pm PROMPT="..."
	@test -n "$(PROMPT)" || (echo "PROMPT is required"; exit 1)
	@ROUTE_FLAG=""; \
	if [ "$(SHOW_ROUTE)" = "1" ] || [ "$(SHOW_ROUTE)" = "true" ] || [ "$(SHOW_ROUTE)" = "yes" ]; then ROUTE_FLAG="--show-route"; fi; \
	scripts/codex-task --mode $(if $(MODE),$(MODE),auto) $$ROUTE_FLAG "$(PROMPT)"

task-go: ## Shortcut: make task-go PROMPT="..."
	@$(MAKE) task MODE=go-dev PROMPT='$(PROMPT)' SHOW_ROUTE='$(SHOW_ROUTE)'

task-review: ## Shortcut: make task-review PROMPT="..."
	@$(MAKE) task MODE=review PROMPT='$(PROMPT)' SHOW_ROUTE='$(SHOW_ROUTE)'

task-pm: ## Shortcut: make task-pm PROMPT="..."
	@$(MAKE) task MODE=pm PROMPT='$(PROMPT)' SHOW_ROUTE='$(SHOW_ROUTE)'

fmt: ## Run go fmt on all packages
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go fmt ./...

coverage: ## Run tests with coverage profile (coverage.out)
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go test ./... -coverprofile=coverage.out

test:
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) go test ./...

test-offline:
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	$(GO_ENV) GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go test ./...

clean: ## Remove build and cache artifacts
	rm -rf $(CACHE_ROOT) coverage.out $(BINARY)

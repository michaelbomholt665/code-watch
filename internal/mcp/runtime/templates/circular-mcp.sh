#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./circular-mcp <operation|command> [flags]

Commands:
  scan                 -> scan.run
  cycles               -> graph.cycles
  modules              -> query.modules
  module-details       -> query.module_details
  trace                -> query.trace
  trends               -> query.trends
  sync-diagrams        -> graph.sync_diagrams
  sync-config          -> system.sync_config
  generate-config      -> system.generate_config
  generate-script      -> system.generate_script
  select-project       -> system.select_project
  watch                -> system.watch

Flags:
  --config <path>        Config path for `circular` (default: data/config/circular.toml)
  --tool <name>          MCP tool name (default: circular)
  --include-tests        Pass `--include-tests` to the CLI runtime
  --path <value>         Repeatable; sets params.paths
  --format <value>       Repeatable; sets params.formats
  --limit <n>            Sets params.limit
  --filter <text>        Sets params.filter
  --module <name>        Sets params.module
  --from <module>        Sets params.from_module
  --to <module>          Sets params.to_module
  --max-depth <n>        Sets params.max_depth
  --name <project>       Sets params.name
  --since <value>        Sets params.since
  --params-json <json>   Raw params object; overrides flag-built params
EOF
}

json_escape() {
  local s=${1//\\/\\\\}
  s=${s//\"/\\\"}
  s=${s//$'\n'/\\n}
  s=${s//$'\r'/\\r}
  s=${s//$'\t'/\\t}
  printf '%s' "$s"
}

json_string_array() {
  local -n arr_ref=$1
  local out="["
  local first=1
  local item
  for item in "${arr_ref[@]}"; do
    if [[ $first -eq 0 ]]; then
      out+=","
    fi
    first=0
    out+="\"$(json_escape "$item")\""
  done
  out+="]"
  printf '%s' "$out"
}

map_operation() {
  case "$1" in
    scan|scan.run) echo "scan.run" ;;
    cycles|graph.cycles) echo "graph.cycles" ;;
    modules|query.modules) echo "query.modules" ;;
    module-details|query.module_details) echo "query.module_details" ;;
    trace|query.trace) echo "query.trace" ;;
    trends|query.trends) echo "query.trends" ;;
    sync-diagrams|graph.sync_diagrams|sync-outputs|system.sync_outputs) echo "graph.sync_diagrams" ;;
    sync-config|system.sync_config) echo "system.sync_config" ;;
    generate-config|system.generate_config) echo "system.generate_config" ;;
    generate-script|system.generate_script) echo "system.generate_script" ;;
    select-project|system.select_project) echo "system.select_project" ;;
    watch|system.watch) echo "system.watch" ;;
    *)
      echo "unknown operation or command: $1" >&2
      exit 1
      ;;
  esac
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

if [[ "$1" == "-h" || "$1" == "--help" ]]; then
  usage
  exit 0
fi

operation="$(map_operation "$1")"
shift

config_path="data/config/circular.toml"
tool_name="circular"
include_tests="false"
params_json=""
limit=""
filter=""
module=""
from_module=""
to_module=""
max_depth=""
project_name=""
since=""
declare -a paths=()
declare -a formats=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config) config_path="${2:-}"; shift 2 ;;
    --tool) tool_name="${2:-}"; shift 2 ;;
    --include-tests) include_tests="true"; shift ;;
    --path) paths+=("${2:-}"); shift 2 ;;
    --format) formats+=("${2:-}"); shift 2 ;;
    --limit) limit="${2:-}"; shift 2 ;;
    --filter) filter="${2:-}"; shift 2 ;;
    --module) module="${2:-}"; shift 2 ;;
    --from) from_module="${2:-}"; shift 2 ;;
    --to) to_module="${2:-}"; shift 2 ;;
    --max-depth) max_depth="${2:-}"; shift 2 ;;
    --name) project_name="${2:-}"; shift 2 ;;
    --since) since="${2:-}"; shift 2 ;;
    --params-json) params_json="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "unknown flag: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$params_json" ]]; then
  declare -a fields=()
  if [[ ${#paths[@]} -gt 0 ]]; then
    fields+=("\"paths\":$(json_string_array paths)")
  fi
  if [[ ${#formats[@]} -gt 0 ]]; then
    fields+=("\"formats\":$(json_string_array formats)")
  fi
  if [[ -n "$limit" ]]; then
    fields+=("\"limit\":$limit")
  fi
  if [[ -n "$max_depth" ]]; then
    fields+=("\"max_depth\":$max_depth")
  fi
  if [[ -n "$filter" ]]; then
    fields+=("\"filter\":\"$(json_escape "$filter")\"")
  fi
  if [[ -n "$module" ]]; then
    fields+=("\"module\":\"$(json_escape "$module")\"")
  fi
  if [[ -n "$from_module" ]]; then
    fields+=("\"from_module\":\"$(json_escape "$from_module")\"")
  fi
  if [[ -n "$to_module" ]]; then
    fields+=("\"to_module\":\"$(json_escape "$to_module")\"")
  fi
  if [[ -n "$project_name" ]]; then
    fields+=("\"name\":\"$(json_escape "$project_name")\"")
  fi
  if [[ -n "$since" ]]; then
    fields+=("\"since\":\"$(json_escape "$since")\"")
  fi

  params_json="{"
  for i in "${!fields[@]}"; do
    if [[ "$i" -gt 0 ]]; then
      params_json+=","
    fi
    params_json+="${fields[$i]}"
  done
  params_json+="}"
fi

request="{\"id\":\"1\",\"tool\":\"$(json_escape "$tool_name")\",\"args\":{\"operation\":\"$(json_escape "$operation")\",\"params\":$params_json}}"

cmd=()
if command -v circular >/dev/null 2>&1; then
  cmd=(circular --config "$config_path")
else
  cmd=(go run ./cmd/circular --config "$config_path")
fi
if [[ "$include_tests" == "true" ]]; then
  cmd+=(--include-tests)
fi

printf '%s\n' "$request" | "${cmd[@]}"

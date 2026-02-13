# Code Watch Capability Contracts

## Common Request Fields

- `paths: []string`
- `config_path: string` (optional)
- `project_root: string` (optional override)

## Tool Contracts

### scan_once

Input:
- `paths`
- `config_path?`

Output:
- `files_scanned: int`
- `modules: int`
- `duration_ms: int`
- `warnings: []string`

### detect_cycles

Input:
- `paths`
- `config_path?`

Output:
- `cycle_count: int`
- `cycles: [][]string`

### find_unresolved

Input:
- `paths`
- `config_path?`

Output:
- `count: int`
- `items: []{file,module,symbol,line,column}`

### trace_import_chain

Input:
- `from_module`
- `to_module`
- `config_path?`

Output:
- `found: bool`
- `path: []string`

### generate_reports

Input:
- `paths`
- `formats: []string` (`tsv|dot|mermaid|plantuml`)

Output:
- `reports: map[string]string`
- `metadata: {generated_at, module_count, edge_count}`

## Error Contract

Return structured errors:
- `code` (`invalid_argument`, `not_found`, `internal`, `unavailable`)
- `message`
- `details` (optional map)

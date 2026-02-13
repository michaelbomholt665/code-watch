# Output Autogeneration Patterns (Go)

Use one canonical in-memory model and render all output formats from that model.

## Canonical Model Pattern

```go
type AnalysisSnapshot struct {
    Modules map[string]Module
    Edges   []Edge
    Metrics map[string]Metric
}
```

Guidelines:
- Sort module keys and edges before rendering.
- Keep formatters pure (`snapshot -> string/bytes`).
- Keep IO outside formatter functions.

## TSV Generation Pattern

```go
func GenerateTSV(s AnalysisSnapshot) (string, error)
```

Rules:
- Fixed header order.
- Always include trailing newline.
- Escape tabs/newlines in field data.

## DOT Generation Pattern

```go
func GenerateDOT(s AnalysisSnapshot) (string, error)
```

Rules:
- Deterministic node/edge ordering.
- Quote labels and IDs safely.
- Add optional style attributes without breaking defaults.

## Mermaid Generation Pattern

```go
func GenerateMermaid(s AnalysisSnapshot) (string, error)
```

Rules:
- Use `graph TD` by default.
- Sanitize IDs into Mermaid-safe tokens.
- Keep output concise for docs embedding.

## PlantUML Generation Pattern

```go
type Event struct {
    From   string
    To     string
    Label  string
}

func GeneratePlantUMLSequence(events []Event) (string, error)
func GeneratePlantUMLComponent(s AnalysisSnapshot) (string, error)
```

Rules:
- Emit `@startuml` / `@enduml` wrappers.
- Keep actor/component names stable.
- Use explicit ordering for reproducible diffs.

## Test Strategy

- Snapshot-test each generator with fixed fixtures.
- Validate deterministic output across repeated runs.
- Add regression fixtures for escaping and edge-case identifiers.

# Pipeline Template Engine

The pipeline template engine provides template rendering for pipeline step parameters and workflow task parameters. It replaces the previous ad-hoc string-replacement logic with a unified, expressive syntax based on Go's `text/template`.

Source: `pkg/pipeline/template/`

## Design

### Why text/template

The old approach used `strings.ReplaceAll` to swap `{{event.id}}` and `{{steps.archive.url}}` with literal values. This had several limitations:

- No support for conditionals (`if`/`else`), loops (`range`), or transformations.
- No built-in utility functions (`join`, `split`, `default`, `json`, `len`).
- Silent fallback: unresolved placeholders were left verbatim in output, hiding configuration errors.
- Each new field required a new `ReplaceAll` call, making the code rigid.

Go's `text/template` was chosen because:

1. It is a standard library package with zero dependencies.
2. It does not HTML-escape output (no `html/template` needed -- pipelines never generate HTML).
3. It supports branching, iteration, pipelining, and custom functions.
4. Pre-existing YAML pipeline definitions can be migrated transparently via a regex preprocessor.

### Architecture

```
YAML config (step params)
    |
    v
RenderContext.RenderParams() / RenderString()
    |
    +--> template.Engine.Render() / RenderString()
             |
             +--> preprocessTemplate()  -- regex rewrites old syntax
             +--> text/template.Parse() -- compiles with FuncMap
             +--> Execute()             -- renders with TemplateData
    |
    v
rendered map[string]any / string
```

### TemplateData

The data context exposed to templates has three namespaces:

| Dot-path   | Type                       | Source                          |
| ---------- | -------------------------- | ------------------------------- |
| `.Event.*` | `map[string]any`           | `DataEvent` fields + `Data` KV  |
| `.Steps.*` | `map[string]map[string]any` | Previous step results, by name  |
| `.Env.*`   | `map[string]string`        | Environment variables (optional) |

**`.Event` keys** (populated by `RenderContext.templateData()`):

```
id               = EntityID          (alias: entity_id)
event_id         = EventID
event_type       = EventType
source           = Source
capability       = Capability
operation        = Operation
backend          = Backend
app              = App
entity_id        = EntityID
idempotency_key  = IdempotencyKey
uid              = UID
topic            = Topic
+ any key from event.Data KV
```

When `event.Data` contains keys that overlap with top-level DataEvent fields, the Data values take precedence (they are copied first, then top-level fields overwrite them).

## Syntax Reference

### Data Access

Basic dot-access on the data context:

```
{{.Event.url}}          # field url from event
{{.Event.id}}           # entity ID
{{.Steps.step1.url}}    # field url from step "step1"
{{.Env.HOME}}           # environment variable
```

Index notation for keys that contain special characters:

```
{{index .Event "some-key"}}
{{index .Steps "step-1" "result-field"}}
```

### Built-in Functions (12)

| Function                          | Description                                   | Example                                                              |
| --------------------------------- | --------------------------------------------- | -------------------------------------------------------------------- |
| `event field`                     | Read a field from the event                   | `{{event "url"}}`                                                    |
| `step name field`                 | Read a field from a step result               | `{{step "archive" "url"}}`                                           |
| `join elems sep`                  | Join a slice into a string                    | `{{join .Event.tags ","}}`                                           |
| `split str sep`                   | Split a string into a slice                   | `{{index (split .Event.csv ",") 0}}`                                 |
| `contains str substr`             | Check if a substring is present               | `{{if contains .Event.title "ERROR"}}alert{{end}}`                   |
| `default def val`                 | Return `def` if `val` is nil or empty string  | `{{default "guest" .Event.username}}`                                |
| `json val`                        | Marshal a value to JSON                       | `{{json .Event.metadata}}`                                           |
| `len val`                         | Return length of string/slice/map (0 for nil) | `{{len .Event.tags}}`                                                |
| `jsonpath jsonStr path`           | Extract string value from JSON via gjson path | `{{jsonpath (json .Event.data) "items.0.id"}}`                       |
| `jsonpathExists jsonStr path`     | Check if a JSON path exists                   | `{{if jsonpathExists (json .Event.data) "error"}}...{{end}}`         |
| `jsonpathRaw jsonStr path`        | Extract raw value from JSON (interface{})     | `{{json (jsonpathRaw (json .Event.data) "nested")}}`                 |

These functions are registered into `text/template.FuncMap` and are available in any template expression.

### JSON Path Extraction (gjson)

The three `jsonpath*` functions use [gjson](https://github.com/tidwall/gjson) path syntax for extracting data from JSON strings. The path syntax supports:

| Syntax              | Meaning                         | Example path       |
| ------------------- | ------------------------------- | ------------------ |
| `field`             | Top-level field                 | `url`              |
| `parent.child`      | Nested field                    | `data.nested.key`  |
| `array.N`           | Array index (0-based)           | `items.0`          |
| `array.#`           | Array length                    | `items.#`          |
| `array.#.field`     | All array elements' field       | `items.#.name`     |
| `array.#(cond)#`    | Filter array by condition       | `users.#(age>20)#` |

**Basic extraction:**

```
{{jsonpath (json .Event.data) "nested.deep"}}
{{jsonpath (step "api" "result") "data.id"}}
{{jsonpath (step "api" "result") "items.1.name"}}
```

When the source is already a JSON string (e.g., a step result from a capability invocation), use `jsonpath` directly:

```
# Step "api" returned: {"data": {"items": [{"id": "x"}, {"id": "y"}]}}
{{jsonpath (step "api" "result") "data.items.1.id"}}          → "y"
{{jsonpath (step "api" "result") "data.items.#"}}             → "2"
{{jsonpath (step "api" "result") "data.items.#.id"}}          → ["x","y"]
```

**Conditional extraction:**

```
{{if jsonpathExists (json .Event.data) "error"}}
  Error: {{jsonpath (json .Event.data) "error.message"}}
{{end}}
```

**Raw value access:**

`jsonpathRaw` returns the underlying Go value (`interface{}`), useful when you need to chain with other functions or iterate:

```
{{range jsonpathRaw (json .Event.data) "items.#"}}
  {{json .}}    # each item as JSON
{{end}}
```

**Filtered array queries:**

gjson supports array filtering with `#(condition)#` syntax:

```
{{jsonpath (json .Event.data) "users.#(age>28).name"}}
```

Operators supported in filters: `==`, `!=`, `<`, `>`, `<=`, `>=`, `%` (mod), `%~` (regex match).

**Composing with other functions:**

```
{{default "unknown" (jsonpath (json .Event.data) "metadata.source")}}
{{if contains (jsonpath (step "log" "result") "status") "ok"}}pass{{end}}
{{printf "ID: %s" (jsonpath (step "api" "result") "data.id")}}
```

### Conditionals

Use Go template control structures:

```
{{if .Event.url}}has-url{{else}}no-url{{end}}
{{if eq .Event.status "done"}}completed{{else}}pending{{end}}
{{if ne .Event.status "failed"}}ok{{end}}
{{if and .Event.x .Event.y}}both{{end}}
{{if or .Event.a .Event.b}}either{{end}}
{{if not .Event.missing}}absent{{end}}
{{if gt .Event.count 3.0}}high{{end}}
{{with .Event.user}}{{.name}}{{end}}
```

Note on boolean evaluation:
- nil / zero-value / empty string → falsy
- non-empty string / non-zero number → truthy

### Loops

```
{{range .Event.items}}{{.}},{{end}}
{{range $index, $value := .Event.items}}{{$index}}:{{$value}};{{end}}
{{range .Event.items}}x{{else}}empty{{end}}
```

Function return values are valid `range` pipelines:

```
{{range split .Event.csv ","}}{{.}}-{{end}}
{{range split "a,b,c" ","}}[{{.}}]{{end}}
```

### Pipelining and variables

```
{{$v := step "archive" "url"}}{{if $v}}URL: {{$v}}{{end}}

{{$parts := split .Event.csv ","}}{{len $parts}} items: {{join $parts "|"}}
```

### Composing Functions

Functions can be chained with parentheses:

```
{{json (event "metadata")}}
{{contains (step "log" "output") "ERROR"}}
{{default "none" (step "prev" "title")}}
{{jsonpath (json .Event.data) "nested.field"}}
{{jsonpath (step "api" "result") "items.0.id"}}
```

### Formatting

The `printf` built-in is always available:

```
{{printf "id-%s" .Event.id}}
{{printf "%02d" .Event.count}}
```

## Backward Compatibility

Three regex-based preprocessors rewrite old syntax **before** template parsing.

### 1. Event fields: `{{event.x}}` → `{{event "x"}}`

```
Input:  {{event.url}}
Output: {{event "url"}}
```

### 2. Step references: `{{steps.s.field}}` → `{{step "s" "field"}}`

```
Input:  {{steps.archive.url}}
Output: {{step "archive" "url"}}
```

### 3. Legacy workflow references: `{{stepName.id}}` → `{{step "stepName" "id"}}`

```
Input:  {{myStep.id}}  or  {{myStep.result}}
Output: {{step "myStep" "id"}}  or  {{step "myStep" "result"}}
```

Note: this regex only matches `.id` and `.result` suffixes, so `{{foo.bar}}` passes through unchanged.

All three preprocessors run in order: `event.` → `steps.` → `stepName.` (id/result).

### Compatibility table

| Old syntax (still works) | New syntax (recommended)       | Go template equivalent           |
| ------------------------ | ------------------------------- | -------------------------------- |
| `{{event.url}}`          | `{{event "url"}}`              | `{{.Event.url}}`                 |
| `{{event.id}}`           | `{{event "id"}}`               | `{{.Event.id}}`                  |
| `{{steps.archive.url}}`  | `{{step "archive" "url"}}`     | `{{index .Steps.archive "url"}}` |
| `{{step1.id}}`           | `{{step "step1" "id"}}`        | `{{index .Steps.step1 "id"}}`    |
| `{{step1.result}}`       | `{{step "step1" "result"}}`    | `{{index .Steps.step1 "result"}}` |

## YAML Usage Notes

### Quoting in YAML

Go template delimiters `{{` and `}}` can confuse YAML parsers. Follow these rules:

**Single-line templates with conditionals**: wrap the value in quotes.

```yaml
params:
  action: "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}"
```

**Multi-line templates**: use YAML literal block scalar `|` or folded block scalar `>`.

```yaml
params:
  message: |
    {{if .Event.user}}
    User: {{.Event.user}}
    {{else}}
    Anonymous
    {{end}}
```

**Simple field references**: no quoting needed.

```yaml
params:
  entity: "{{event.id}}"      # ok
  link: {{event.url}}         # ok, but quoting is safer
```

### Escaping quotes inside templates

When using string literals inside template conditionals in YAML strings, escape double quotes:

```yaml
# Inside a YAML double-quoted string, escape Go template quotes
action: "{{if eq .Event.status \"done\"}}archive{{else}}skip{{end}}"

# Alternative: use YAML single quotes (no escaping needed, but YAML parser-dependent)
action: '{{if eq .Event.status "done"}}archive{{else}}skip{{end}}'
```

## Error Handling

### Missing fields with `event` / `step` functions

The `event` and `step` functions return `""` (empty string) for missing keys, matching the previous behavior:

```
{{event "nonexistent"}}  → ""  (empty string, no error)
{{step "x" "y"}}         → ""  (empty string, no error)
```

### Missing fields with dot-access

Direct `.Event.field` access on a non-existent key renders the Go zero value:

```
{{.Event.nonexistent}}  → "<no value>"  (Go's nil representation)
```

Pipeline authors should prefer `{{event "field"}}` for safe rendering, or use `{{default "fallback" .Event.field}}` to provide a fallback.

### Invalid template syntax

Syntax errors (unbalanced braces, malformed control structures) return an error from both `RenderString()` and `Render()`:

```
{{if .Event.x}}}        → error: "template parse: unexpected }"
{{if missing end}}      → error: "template parse: unexpected EOF"
```

## Usage in Code

### Pipeline (RenderContext)

```go
import "github.com/flowline-io/flowbot/pkg/pipeline/template"

rc := pipeline.NewRenderContext(event)
rc.RecordStepResult("step1", map[string]any{"id": "abc", "url": "https://x.com"})

params := map[string]any{
    "entity":  "{{event.id}}",
    "ref_url": "{{steps.step1.url}}",
}
rendered, err := rc.RenderParams(params)
// or single string:
result, err := rc.RenderString("id={{event.id}} url={{event.url}}")
```

### Workflow (resolveParams)

The `resolveParams` function uses the same template engine internally:

```go
params := types.KV{"output": "{{step1.result}}"}
results := map[string]string{"step1": "my-output"}
resolved, err := workflow.resolveParams(params, results)
// resolved["output"] → "my-output"
```

Workflow results are mapped: each `result` string is exposed as both `id` and `result` fields:

```
results["step1"] = "output-string"
→ Steps["step1"]["id"]     = "output-string"
→ Steps["step1"]["result"]  = "output-string"
```

### Standalone engine

```go
e := template.New()
data := &template.TemplateData{
    Event: map[string]any{"url": "https://example.com"},
    Steps: map[string]map[string]any{
        "archive": {"id": "a1"},
    },
}

s, err := e.RenderString("{{.Event.url}} -> {{step \"archive\" \"id\"}}", data)
// s = "https://example.com -> a1"

params, err := e.Render(map[string]any{
    "action": "{{if .Event.url}}present{{end}}",
}, data)
// params["action"] = "present"
```

## Testing

```bash
go test ./pkg/pipeline/template/...
```

Test coverage includes: plain text passthrough, event fields, step fields, env fields, conditions (if/else/eq/ne/and/or/not/gt), loops (range/range-index/range-else), nested condition+loop, all 12 built-in functions (including jsonpath, jsonpathExists, jsonpathRaw for gjson-based JSON extraction), old syntax compatibility, error propagation, invalid templates, and nil/empty data.

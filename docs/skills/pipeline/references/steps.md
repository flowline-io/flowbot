# Pipeline steps reference

Load this file when authoring or editing pipeline YAML steps. Teaching examples:
- [examples/cron_cleanup.yaml](../examples/cron_cleanup.yaml)
- [examples/event_notify.yaml](../examples/event_notify.yaml)

For every capability operation param table, start at [capabilities.md](capabilities.md) and open `capabilities/<type>.md` **before** writing that step. Do not invent types missing from the index.

Top-level definition fields and triggers: [schema.md](schema.md).

## Shared step fields

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Unique within the pipeline |
| `capability` | yes | Capability type (e.g. `core`, `karakeep`) |
| `operation` | yes | Operation name on that capability |
| `params` | no | Template-rendered before execution |
| `retry` | no | `max_attempts`, `delay`, `backoff`, `max_delay`, `jitter`, `retry_on` |

## Templates

Pipeline step `params` (string values) are rendered with Go `text/template` before the step runs.
Delimiters are `{{` and `}}`.

### Available variables

| Variable | Populated? | How to read | Source |
|----------|------------|-------------|--------|
| `Event` | yes | `{{event "field"}}` / `{{.Event.field}}` | DataEvent payload or `pipeline run --event` JSON |
| `Steps` | yes | `{{step "step_name" "field"}}` | Outputs of **already completed** steps |
| `Env` | sometimes | `{{.Env.HOME}}` | Process environment on the runner |
| `Input` | no | — | Workflow-only; do not rely on it in pipelines |

### Helper functions

**Closed set only.** Missing helpers fail at run time with `function "…" not defined`.

| Helper | Example |
|--------|---------|
| `event` | `{{event "url"}}` |
| `step` | `{{step "fetch" "count"}}` |
| `jsonpath` | `{{jsonpath (step "api" "result") "data.id"}}` |
| `default` | `{{default "guest" (event "user")}}` |
| `json` / `len` / `join` / `split` / `contains` / `now` | see workflow steps docs for shapes |

YAML tip: when an expression contains quotes, wrap the param value in single quotes:

```yaml
params:
  title: 'Item: {{event "title"}}'
  url: "{{event.url}}"
```

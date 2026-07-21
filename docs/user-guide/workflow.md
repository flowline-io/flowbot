# Workflow Engine

YAML-defined workflow execution with retry, DAG validation, and persistent state.

Source: `pkg/workflow/`

## Overview

Workflow definitions are stored in the database. YAML is an exchange format for
`apply` / `export` only — the server does not execute local file paths. The runner
supports capability, Docker, shell, machine, and mapper actions, Go template params,
and DAG validation.

```
flowbot workflow apply --file wf.yaml
        │
        ▼
ParseYAML → ApplyDefinition (DB) → ReloadTriggers (cron/webhook)
        │
POST /service/workflow/run { "name", "input" }
        │
        ▼
StartRunAsync → ValidateInputs → CreateRun → Execute (goroutine)
        │
        ▼
GET /service/workflow/runs/:name  (inspect status)
```

## YAML Schema

Workflow files are standalone YAML documents:

```yaml
name: save_and_track
describe: "Save a URL as a bookmark, archive it, and create a kanban task"
enabled: true                          # false disables cron/webhook triggers
resumable: true                        # enable state persistence

inputs:                                # declared run inputs (validated on run)
  - name: url
    type: string
    required: true
  - name: title
    type: string
    required: true

pipeline:                              # ordered list of task IDs
  - save_bookmark
  - archive_url
  - create_task

tasks:
  - id: save_bookmark
    action: capability:karakeep.create   # action format: <type>:<details>
    describe: "Save the URL as a bookmark"
    params:                              # template-rendered input
      url: "{{input.url}}"
      title: "{{input.title}}"
    vars:                                # declared variables (reserved)
      - url
    conn:                                # DAG dependency edges (for validation)
      - archive_url
    retry:                               # task-level retry (optional)
      max_attempts: 3
      delay: 2s
      backoff: exponential
      max_delay: 30s
      jitter: true

  - id: archive_url
    action: capability:archive.add
    describe: "Archive the URL in ArchiveBox"
    params:
      url: "{{input.url}}"
    retry:
      max_attempts: 5
      delay: 1s
      backoff: linear
      max_delay: 60s

  - id: create_task
    action: capability:kanboard.create_task
    describe: "Create a follow-up task"
    params:
      title: "Read: {{input.title}}"
      description: "Bookmark reference: {{step "save_bookmark" "result"}}"
      tags:
        - reading
        - bookmark
```

### Top-Level Fields

| Field             | Type      | Required | Description                                    |
| ----------------- | --------- | -------- | ---------------------------------------------- |
| `name`            | string    | Yes      | Unique workflow identifier                     |
| `describe`        | string    | No       | Human-readable description                     |
| `enabled`         | bool      | No       | Default true; false disables cron/webhook      |
| `resumable`       | bool      | No       | Enable checkpoint persistence (default: false) |
| `max_concurrency` | int       | No       | >1 enables parallel DAG execution              |
| `inputs`          | []Input   | No       | Declared run inputs for validation/forms       |
| `triggers`        | []Trigger | No       | Trigger configurations (cron, manual, webhook) |
| `pipeline`        | []string  | Yes      | Ordered list of task IDs to execute            |
| `tasks`           | []Task    | Yes      | Task definitions                               |

### Task Fields

| Field      | Type        | Required | Description                                                                            |
| ---------- | ----------- | -------- | -------------------------------------------------------------------------------------- |
| `id`       | string      | Yes      | Unique task identifier                                                                 |
| `action`   | string      | Yes      | `capability:<type>.<op>`, `docker:<image>`, `shell:<cmd>`, `machine:<name>`, `mapper:` |
| `describe` | string      | No       | Human-readable description                                                             |
| `params`   | KV          | No       | Input parameters (template-rendered)                                                   |
| `vars`     | []string    | No       | Declared variable names (reserved)                                                     |
| `conn`     | []string    | No       | DAG dependency edges (validated for cycles; used for parallel scheduling)              |
| `retry`    | RetryConfig | No       | Retry strategy (see below)                                                             |

## Action Types

| Prefix        | Runtime               | Example                      |
| ------------- | --------------------- | ---------------------------- |
| `capability:` | Capability            | `capability:karakeep.create` |
| `docker:`     | Docker container      | `docker:nginx:latest`        |
| `shell:`      | Shell command         | `shell:echo hello`           |
| `machine:`    | Remote SSH            | `machine:vm1`                |
| `mapper:`     | Inline data transform | `mapper:`                    |
| Free-form     | Shell fallback        | `custom-action`              |

### Mapper Step (`mapper:`)

The mapper step provides a lightweight data transformation node within the workflow. It takes template-rendered parameters and serializes them to a JSON string, making it suitable for converting output formats between steps. Unlike other action types, mapper is handled inline in the workflow runner -- no external runtime or process is involved.

Mapper steps are resolved before the normal task execution path. When a task's action starts with `mapper:`, the runner:

1. Resolves template expressions in the step's `params` against previous step results.
2. Marshals the resolved params to a JSON string.
3. Stores the JSON as the step result for downstream consumption.
4. Skips the engine/runtime dispatch entirely.

In YAML, quote the action because a trailing colon is otherwise invalid: `action: "mapper:"`.

## Retry Strategy

See [Pipeline Retry](pipeline.md#retry-strategy) for the full `retry` field schema. The workflow engine uses the same `types.RetryConfig` converted via `ToBackoffConfig()` and executed with `backoff.Do()`.

Key difference: the workflow engine retries ALL errors (no `retry_on` filtering), since workflow tasks don't typically return `types.Error`.

## Parameter Resolution

Task `params` are rendered through the template engine before execution. The data context for each step includes:

- Results from all previously completed steps (mapped as `{{step "id" "result"}}`)
- Input variables passed to the workflow entry point (`{{input.*}}`)

See [Pipeline Template Engine](pipeline-template.md) for the full template syntax.

## DAG Validation

The `conn` field declares dependency edges between tasks. Before execution, `ValidateDAG()` performs a DFS cycle check. With `max_concurrency > 1`, `conn` also drives parallel scheduling; otherwise execution order follows `pipeline`.

## Invocation

### CLI

```bash
flowbot workflow apply --file docs/examples/workflows/save_and_track.yaml
flowbot workflow run save_and_track --input '{"url":"https://example.com","title":"Example"}'
flowbot workflow runs save_and_track
```

Scopes: `workflow:read` (list/get/export/runs), `workflow:run` (apply/delete/run).

### From HTTP

```
POST /service/workflow/apply
{"yaml": "..."}

POST /service/workflow/run
{"name": "save_and_track", "input": {"url": "...", "title": "..."}}
→ 202 {"status":"ok","data":{"run_id":123}}

GET /service/workflow/list
GET /service/workflow/get/:name
GET /service/workflow/export/:name
DELETE /service/workflow/delete/:name
GET /service/workflow/runs/:name
```

Webhook triggers are served at `/webhook/workflow/{path}` with the same token/HMAC auth fields as pipeline webhooks.

### Web UI

Open **Automate → Workflows** (`/service/web/workflows`):

| Page | Features |
|------|----------|
| List | Name, status, triggers, task count, **Last Run**; Enable/Disable; open Runs |
| Detail | Overview tab: Inputs, Triggers (Enable/Disable), Execution DAG, Run now, Recent runs; YAML tab: exported definition text |
| Runs | Expandable run rows with step Input/Output/Error; polling pauses while a run is expanded |

DAG badge **Parallel DAG** appears only when `max_concurrency > 1`. Conn graphs with `max_concurrency ≤ 1` still show topology but are labeled **Sequential**.

### From Code

```go
runID, err := svc.StartRunAsync(ctx, "save_and_track", "manual", types.KV{
    "url": "https://example.com",
    "title": "Example",
})
```

## Testing

```bash
go test ./pkg/workflow/...   # Unit tests for parsing, DAG, params, runner, service
```

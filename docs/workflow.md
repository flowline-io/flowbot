# Workflow Engine

YAML-defined workflow execution with retry, DAG validation, and persistent state.

Source: `pkg/workflow/`

## Overview

The workflow engine executes ordered sequences of tasks defined in YAML files. Tasks can invoke capabilities, run Docker containers, execute shell commands, or connect to remote machines. The engine supports parameter resolution via Go templates and validates DAG dependencies to prevent cycles.

```
Workflow YAML (parse + validate DAG)
    │
    ▼
Runner.Execute()
    │
    ├── For each task in pipeline order:
    │     ├── resolveParams (template engine)
    │     ├── if action is mapper:
    │     │     └── json.Marshal(params) → store as step result
    │     │         (inline, no external runtime)
    │     ├── else:
    │     │     ├── WorkflowTaskToTask (task conversion)
    │     │     ├── runWithRetry (Runner.Run with backoff)
    │     │     └── Collect result for downstream templates
    │     └── continue
    └── Return success or error
```

## YAML Schema

Workflow files are standalone YAML documents:

```yaml
name: save_and_track
describe: "Save a URL as a bookmark, archive it, and create a kanban task"
resumable: true                        # enable state persistence

pipeline:                              # ordered list of task IDs
  - save_bookmark
  - archive_url
  - create_task

tasks:
  - id: save_bookmark
    action: capability:bookmark.create   # action format: <type>:<details>
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
    action: capability:kanban.create_task
    describe: "Create a follow-up task"
    params:
      title: "Read: {{input.title}}"
      description: "Bookmark reference: {{step "save_bookmark" "result"}}"
      tags:
        - reading
        - bookmark
```

### Top-Level Fields

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `name` | string | Yes | Unique workflow identifier |
| `describe` | string | No | Human-readable description |
| `resumable` | bool | No | Enable checkpoint persistence (default: false) |
| `triggers` | []Trigger | No | Trigger configurations (cron, manual, webhook) |
| `pipeline` | []string | Yes | Ordered list of task IDs to execute |
| `tasks` | []Task | Yes | Task definitions |

### Task Fields

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `id` | string | Yes | Unique task identifier |
| `action` | string | Yes | `capability:<type>.<op>`, `docker:<image>`, `shell:<cmd>`, `machine:<name>`, `mapper:` |
| `describe` | string | No | Human-readable description |
| `params` | KV | No | Input parameters (template-rendered) |
| `vars` | []string | No | Declared variable names (reserved) |
| `conn` | []string | No | DAG dependency edges (validated for cycles, not used for scheduling) |
| `retry` | RetryConfig | No | Retry strategy (see below) |

## Action Types

| Prefix | Runtime | Example |
| ------ | ------- | ------- |
| `capability:` | Capability | `capability:bookmark.create` |
| `docker:` | Docker container | `docker:nginx:latest` |
| `shell:` | Shell command | `shell:echo hello` |
| `machine:` | Remote SSH | `machine:vm1` |
| `mapper:` | Inline data transform | `mapper:` |
| Free-form | Shell fallback | `custom-action` |

### Mapper Step (`mapper:`)

The mapper step provides a lightweight data transformation node within the workflow. It takes template-rendered parameters and serializes them to a JSON string, making it suitable for converting output formats between steps. Unlike other action types, mapper is handled inline in the workflow runner -- no external runtime or process is involved.

Mapper steps are resolved before the normal task execution path. When a task's action starts with `mapper:`, the runner:
1. Resolves template expressions in the step's `params` against previous step results.
2. Marshals the resolved params to a JSON string.
3. Stores the JSON as the step result for downstream consumption.
4. Skips the engine/runtime dispatch entirely.

**Example: field mapping between two capability steps**

```yaml
pipeline:
  - fetch_data
  - transform_output
  - consume_data

tasks:
  - id: fetch_data
    action: capability:api.fetch
    params:
      endpoint: "/users"

  - id: transform_output
    action: mapper:
    params:
      target_url: '{{jsonpath (step "fetch_data" "result") "data.0.link"}}'
      target_title: '{{jsonpath (step "fetch_data" "result") "data.0.name"}}'
      metadata:
        source: api
        priority: high

  - id: consume_data
    action: capability:bookmark.create
    params:
      url: '{{jsonpath (step "transform_output" "result") "target_url"}}'
      title: '{{jsonpath (step "transform_output" "result") "target_title"}}'
```

**Example: conditional field mapping**

```yaml
  - id: conditional_map
    action: mapper:
    params:
      status: "{{if jsonpathExists (step \"api\" \"result\") \"error\"}}failed{{else}}ok{{end}}"
      output: '{{default "{}" (step "api" "result")}}'
```

The mapper's output is a JSON object where each key from `params` becomes a top-level field. Subsequent steps can extract individual fields using `jsonpath` or reference the full result with `{{step "transform_output" "result"}}`.

## Retry Strategy

See [Pipeline Retry](pipeline.md#retry-strategy) for the full `retry` field schema. The workflow engine uses the same `types.RetryConfig` and backoff logic via `BuildBackOff()`.

Key difference: the workflow engine retries ALL errors (no `retry_on` filtering), since workflow tasks don't typically return `types.Error`.

## Parameter Resolution

Task `params` are rendered through the template engine before execution. The data context for each step includes:
- Results from all previously completed steps (mapped as `{{step "id" "result"}}`)
- Input variables passed to the workflow entry point (`{{input.*}}`)

See [Pipeline Template Engine](pipeline-template.md) for the full template syntax.

### Result Handling

After a task succeeds:
- If `task.Result` is non-empty, it is stored in the `results` map keyed by task ID.
- Downstream tasks can reference it: `{{step "save_bookmark" "result"}}`.
- Both the raw result string and the step output JSON are available.

## DAG Validation

The `conn` field declares dependency edges between tasks. Before execution, `ValidateDAG()` performs a DFS cycle check:

```
save_bookmark → archive_url → create_task     (valid)
save_bookmark → archive_url → save_bookmark   (cycle detected)
```

Tasks with unknown dependencies in `conn` are also rejected.

Note: `conn` is currently used only for validation. Execution order is strictly determined by the `pipeline` list, not the DAG topology.

## Persistent State (resumable)

When `resumable: true`, the workflow engine persists execution state via the `WorkflowStore` to MySQL:

### Tables Used

| Table | Purpose |
| ----- | ------- |
| `jobs` | Workflow run: state, workflow_id, timing |
| `steps` | Per-task execution: action, input, output, state, error |
| `workflow` | Workflow definition: name, state, counters |
| `workflow_script` | YAML content: lang, code, version |
| `workflow_trigger` | Trigger config: type, rule |

### State Flow

```
Job: Ready → Start → Running → Succeeded / Canceled / Failed
Step: Created → Ready → Start → Running → Succeeded / Failed / Canceled / Skipped
```

Each step creates a `steps` record before execution and updates it on completion. The `output` field stores the task result for downstream parameter resolution.

## Execution Flow

### Sequential

Tasks execute in the order listed in `pipeline`. If a task fails (after retries are exhausted), the entire workflow stops and returns the error. No subsequent tasks execute.

### Retry Loop

```go
for attempt := 1; ; attempt++ {
    err := r.Run(ctx, task)
    if err == nil { return nil }
    if !retryCfg.RetryEnabled() { return err }
    nextDelay := bo.NextBackOff()
    if nextDelay == Stop { return error }
    // wait with context cancellation check
    time.After(nextDelay)
}
```

### Error Propagation

| Stage | Error | Return |
| ----- | ----- | ------ |
| Task not found in taskMap | `task %s not found in workflow` | Immediate |
| Resolve params failure | `resolve params step %s: %w` | Immediate |
| Mapper marshal failure | `mapper step %s: %w` | Immediate |
| Convert task failure | `convert task %s: %w` | Immediate |
| Run failure (retries exhausted) | `step %s (retries exhausted, attempt %d): %w` | Immediate |
| Run failure (context cancel) | `step %s cancelled: %w` | Immediate |

## Invocation

### From Code

```go
import (
    "github.com/flowline-io/flowbot/pkg/workflow"
    "github.com/flowline-io/flowbot/pkg/types"
)

runner := workflow.NewRunner()
wf, _ := workflow.LoadFile("/path/to/workflow.yaml")
err := runner.Execute(ctx, *wf)
```

### From HTTP

```
POST /service/workflow/run
Content-Type: application/json

{"file": "/path/to/workflow.yaml"}
```

## Testing

```bash
go test ./pkg/workflow/...   # Unit tests for parsing, DAG, params, runner
```

## Recovery

See [Recovery Manager](recovery.md) for restart recovery of incomplete workflow jobs.

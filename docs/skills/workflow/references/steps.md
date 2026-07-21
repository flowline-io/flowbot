# Workflow task steps reference

Load this file when authoring or editing workflow YAML tasks. Teaching examples:
- [examples/echo_mapper.yaml](../examples/echo_mapper.yaml)
- [examples/parallel_example.yaml](../examples/parallel_example.yaml)
- [examples/save_and_track.yaml](../examples/save_and_track.yaml)

## Shared task fields

| Field | Required | Notes |
|-------|----------|-------|
| `id` | yes | Unique within the workflow |
| `action` | yes | See action types below |
| `describe` | no | Human-readable label |
| `params` | no | Template-rendered before execution; declare matching top-level `inputs` when using `{{input.*}}` |
| `conn` | no | Upstream task ids (DAG edges; required for parallel scheduling) |
| `retry` | no | Same shape as pipeline retry (`max_attempts`, `delay`, `backoff`, `max_delay`, `jitter`); workflows retry all errors |

With `max_concurrency > 1`, `conn` drives parallel DAG scheduling. Otherwise order follows `pipeline`.

## Templates

Workflow task `params` (string values) are rendered with Go `text/template` before the step runs.
Delimiters are `{{` and `}}`. Missing keys via helpers return empty string; invalid template syntax errors.

### Available variables

Root context is `TemplateData`: `.Input`, `.Steps`, `.Event`, `.Env`. Workflows only populate the first two.

| Variable | Populated in workflow? | How to read | Source |
|----------|------------------------|-------------|--------|
| `Input` | yes | `{{input "name"}}`, `{{input.name}}`, `{{.Input.name}}` | Run payload from `workflow run --input` / API; keys match declared top-level `inputs[].name` (plus defaults applied by validation) |
| `Steps` | yes | `{{step "task_id" "result"}}`, `{{step "task_id" "id"}}`, `{{.Steps.task_id.result}}` | Outputs of **already completed** tasks only. Workflow stores the same string under both `result` and `id` |
| `Event` | no | `{{event "field"}}` / `{{.Event.field}}` | Empty in workflows (pipeline DataEvent only). Do not rely on it |
| `Env` | no | `{{.Env.HOME}}` | Empty in workflows. Do not rely on it |

`{{input.name}}` is sugar for `{{input "name"}}`.

### Helper functions

Data accessors: `input`, `step`, `event`.

| Helper | Example |
|--------|---------|
| `jsonpath` | `{{jsonpath (step "api" "result") "data.id"}}` |
| `jsonpathExists` | `{{if jsonpathExists (step "api" "result") "error"}}bad{{end}}` |
| `jsonpathRaw` | `{{json (jsonpathRaw (step "api" "result") "items")}}` |
| `default` | `{{default "guest" (input "user")}}` |
| `json` | `{{json (input "meta")}}` |
| `len` | `{{len (input "tags")}}` |
| `join` / `split` | `{{join (split (input "tags") ",") ";")}}` |
| `contains` | `{{if contains (input "title") "ERROR"}}alert{{end}}` |
| `if` / `else` | `{{if (input "url")}}has{{else}}missing{{end}}` |

YAML tip: when an expression contains quotes, wrap the param value in single quotes:

```yaml
params:
  description: 'Bookmark: {{step "save_bookmark" "result"}}'
  url: "{{input.url}}"
```

## Action types

### Capability (`capability:`)

Invoke a Flowbot capability operation

**Action form:** `capability:<type>.<operation>`

**Params:** KV object passed to the capability after template render. Keys depend on the operation — use the matching capability skill (e.g. karakeep, kanboard) for field details; do not invent provider-specific keys.

**Notes:** Example: capability:karakeep.create with params.url. See examples/save_and_track.yaml.

```yaml
  - id: save_bookmark
    action: capability:karakeep.create
    params:
      url: "{{input.url}}"
```

### Docker (`docker:`)

Run a container image via the Docker runtime

**Action form:** `docker:<image>`

**Params:** Optional `cmd` (string or string list) overrides the container command.

**Notes:** Image is taken from the action details (e.g. docker:alpine:3.20).

```yaml
  - id: run_tool
    action: docker:alpine:3.20
    params:
      cmd: ["echo", "hello"]
```

### Shell (`shell:`)

Run a shell command on the workflow runner host

**Action form:** `shell:<command>`

**Params:** Optional `cmd` (string) replaces the command from the action details.

**Notes:** Prefer explicit shell: prefix over free-form actions.

```yaml
  - id: echo_host
    action: shell:echo hello
    params:
      cmd: "echo from params"
```

### Machine (SSH) (`machine:`)

Run on a named remote machine via SSH runtime

**Action form:** `machine:<name>`

**Params:** Typically empty; remote target comes from the machine name in the action.

**Notes:** Requires the machine runtime to be configured on the server.

```yaml
  - id: remote_check
    action: machine:vm1
```

### Mapper (`mapper:`)

Inline data transform: render params and marshal to JSON (no external runtime)

**Action form:** `mapper:`

**Params:** Any KV; values support templates. The rendered object is stored as the step result JSON string.

**Notes:** Quote the action in YAML (`action: "mapper:"`) because a trailing colon is otherwise invalid. See examples/echo_mapper.yaml.

```yaml
  - id: build_payload
    action: "mapper:"
    params:
      message: "{{input.message}}"
      tag: "{{input.tag}}"
```

### Free-form and echo (`free-form / echo`)

Actions without a known prefix fall through to shell-style run; bare echo is a special type name

**Action form:** `<command> or echo`

**Params:** Same optional `cmd` override behavior as shell when treated as a shell run.

**Notes:** Prefer shell:, docker:, capability:, or mapper: in new YAML. A bare echo action parses as type echo with empty details; free-form strings become the run command. Avoid relying on free-form for new workflows.

```yaml
  - id: legacy_echo
    action: echo
```

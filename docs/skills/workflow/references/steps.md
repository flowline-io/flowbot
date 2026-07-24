# Workflow task steps reference

Load this file when authoring or editing workflow YAML tasks. Teaching examples:
- [examples/echo_mapper.yaml](../examples/echo_mapper.yaml)
- [examples/parallel_example.yaml](../examples/parallel_example.yaml)
- [examples/save_and_track.yaml](../examples/save_and_track.yaml)

For every `capability:<type>.<op>` param table, start at [capabilities.md](capabilities.md) and open `capabilities/<type>.md` **before** writing that action. Do not invent types missing from the index.

Top-level definition fields and triggers: [schema.md](schema.md).

## Shared task fields

| Field | Required | Notes |
|-------|----------|-------|
| `id` | yes | Unique within the workflow |
| `action` | yes | See action types below |
| `describe` | no | Human-readable label |
| `params` | no | Template-rendered before execution; declare matching top-level `inputs` when using `{{input.*}}` |
| `conn` | no | Upstream task ids (DAG edges; required for parallel scheduling) |
| `vars` | no | Optional string list (advanced; usually omit) |
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

Invoke a Flowbot capability operation (provider or CapCore)

**Action form:** `capability:<type>.<operation>`

**Inputs:**

- **Action:** `<type>` is the hub capability ID (provider ID, or `core`). `<operation>` is the op name after the last dot (e.g. `karakeep.create`, `core.notify_send`).
- **Params:** template-rendered KV passed to `capability.Invoke`. Per-operation required/optional keys: load [capabilities/<type>.md](capabilities/) via the [capabilities.md](capabilities.md) index.

**Outputs (`{{step "id" "result"}}`):**

JSON string of `InvokeResult`: `{"capability","operation","data","text","page",...}`.
- Domain payload is usually under `data` (object, array, or scalar). Optional `text` is a short human summary. List ops may include `page`.
- CapCore shapes (examples): `notify_send` → `data.sent`; `agent_run` → `data.reply`; `kv_get` → `data` is the stored value; `http_request` → `data.status`/`data.body`; `run_*` → `data.output`/`data.exit_code`. Full per-op notes: [capabilities/core.md](capabilities/core.md).

**Usage:**

- Read a field: `{{jsonpath (step "save" "result") "data.url"}}`
- Check a key exists: `{{if jsonpathExists (step "save" "result") "data.id"}}...{{end}}`
- Pass whole JSON into another param (rare): `{{step "save" "result"}}` (string). Prefer jsonpath for structured data.
- Authoring checklist: load [capabilities/<type>.md](capabilities/) first, copy required params, then chain with jsonpath.
- Never invent capability types or ops; only use the [capabilities.md](capabilities.md) index.
- Pre-CapCore notify/agent/clip types were removed; use `capability:core.*` (see docs/developer-guide/capcore-migration.md).

**Notes:** Do not invent param keys or types; capabilities/<type>.md is generated from OpDef.Input. Workflow wiring only forwards params — mounts/env/timeouts are not set at the workflow layer.

```yaml
  - id: save_bookmark
    action: capability:karakeep.create
    params:
      url: "{{input.url}}"
  - id: notify_saved
    action: capability:core.notify_send
    params:
      template_id: bookmark.saved
      channels: ["slack"]
      payload:
        url: "{{input.url}}"
        bookmark: '{{jsonpath (step "save_bookmark" "result") "data"}}'
```

### Docker (`docker:`)

Run a container image via the Docker runtime on the workflow runner

**Action form:** `docker:<image>`

**Inputs:**

- **Action details:** image reference after the prefix (e.g. `alpine:3.20` in `docker:alpine:3.20`).
- **Params:**
  | Key | Type | Required | Meaning |
  |-----|------|----------|---------|
  | `cmd` | string or string list | no | Container command; overrides the image default CMD |

Workflow does **not** map mounts, env, or timeout from YAML params.

**Outputs (`{{step "id" "result"}}`):**

Plain text **stdout** from the container (not JSON). Trailing newlines are kept as returned by the runtime.

**Usage:**

- Echo into a later step: `message: '{{step "run_tool" "result"}}'`
- Do not use `jsonpath` unless the container itself prints JSON; then `{{jsonpath (step "run_tool" "result") "field"}}` works on that string.

**Notes:** Requires Docker available to the server executor.

```yaml
  - id: run_tool
    action: docker:alpine:3.20
    params:
      cmd: ["echo", "hello"]
  - id: use_stdout
    action: "mapper:"
    params:
      from_docker: '{{step "run_tool" "result"}}'
```

### Shell (`shell:`)

Run a shell command on the workflow runner host

**Action form:** `shell:<command>`

**Inputs:**

- **Action details:** default command after the prefix (e.g. `echo hello`).
- **Params:**
  | Key | Type | Required | Meaning |
  |-----|------|----------|---------|
  | `cmd` | string | no | Replaces the command from the action details when set |

**Outputs (`{{step "id" "result"}}`):**

Plain text **stdout** from the process (not JSON).

**Usage:**

- Prefer an explicit `shell:` prefix over free-form actions.
- Chain stdout: `{{step "echo_host" "result"}}`. Use jsonpath only if stdout is JSON.

**Notes:** Runs on the Flowbot host (or sandbox configuration of the shell runtime), not inside an arbitrary container unless the command itself uses docker.

```yaml
  - id: echo_host
    action: shell:echo hello
    params:
      cmd: "echo from params"
  - id: capture
    action: "mapper:"
    params:
      out: '{{step "echo_host" "result"}}'
```

### Machine (SSH) (`machine:`)

Intended for a named remote machine via SSH runtime

**Action form:** `machine:<name>`

**Inputs:**

- **Action details:** machine name (e.g. `vm1`).
- **Params:** typically empty; remote target is meant to come from the machine name.

**Outputs (`{{step "id" "result"}}`):**

If routed correctly, remote command stdout as text. **Current workflow routing:** `DetermineRuntimeType` does not select the machine engine — `machine:<name>` is treated like a shell command whose run string is the machine name. Prefer `shell:` / `docker:` until SSH routing is fixed.

**Usage:**

- Do not rely on `machine:` in new workflows.
- Use `shell:` with an explicit ssh command, or `docker:`, when remote execution is required.

**Notes:** Documented for completeness; avoid in production YAML until workflow selects runtime.Machine.

```yaml
  - id: remote_check
    action: machine:vm1
```

### Mapper (`mapper:`)

Inline data transform: render params and marshal to JSON (no external runtime)

**Action form:** `mapper:`

**Inputs:**

- **Action:** must be quoted in YAML: `action: "mapper:"` (trailing colon is otherwise invalid YAML).
- **Params:** any KV; string values support templates (`{{input.*}}`, `{{step ...}}`). Non-string values are kept as structured data in the output object.

**Outputs (`{{step "id" "result"}}`):**

JSON **object** string of the fully rendered params, e.g. `{"message":"hi","tag":"demo"}`. There is no `data` wrapper — fields are top-level.

**Usage:**

- Read a field: `{{jsonpath (step "build_payload" "result") "message"}}`
- Use as a pure reshape step between capability/shell steps.
- See examples/echo_mapper.yaml.

**Notes:** Does not call executor runtimes; failures are template or JSON marshal errors only.

```yaml
  - id: build_payload
    action: "mapper:"
    params:
      message: "{{input.message}}"
      tag: "{{input.tag}}"
      source: cli-example
  - id: read_message
    action: "mapper:"
    params:
      echoed: '{{jsonpath (step "build_payload" "result") "message"}}'
```

### Free-form and echo (`free-form / echo`)

Actions without a known prefix fall through to shell-style run; bare echo is a special type name

**Action form:** `<command> or echo`

**Inputs:**

- **Action:** bare `echo` or any string without a known prefix (`capability:`/`docker:`/`shell:`/`machine:`/`mapper:`).
- **Params:** optional `cmd` (string), same override behavior as shell when treated as a shell run.

**Outputs (`{{step "id" "result"}}`):**

Plain text stdout (same as shell).

**Usage:**

- Prefer `shell:`, `docker:`, `capability:`, or `mapper:` in new YAML.
- Avoid free-form for new workflows; keep only for legacy definitions.

**Notes:** A bare echo action parses as type echo with empty details; free-form strings become the run command.

```yaml
  - id: legacy_echo
    action: echo
```

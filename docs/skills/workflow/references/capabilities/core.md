# `core` capability actions

Core runtime primitives: notify, clip, agent, HTTP, sandboxed exec, and KV

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:core.agent_health`

Agent subsystem health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: agent_health_step
    action: capability:core.agent_health
```

## `capability:core.agent_run`

Execute one autonomous agent turn with a prompt (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | `string` | yes | User prompt |
| `uid` | `string` | no | Owner UID |
| `tools` | `[]string` | no | Tool allowlist |
| `skills` | `[]string` | no | Skill allowlist |
| `memory_scope` | `string` | no | Memory scope; defaults to pipeline name |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: agent_run_step
    action: capability:core.agent_run
    params:
      prompt: "..."  # required
      uid: "..."
      tools: ["..."]
      skills: ["..."]
      memory_scope: "..."
```

## `capability:core.clip_create`

Create a markdown clip and return its public URL (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | `string` | yes | Markdown body |
| `created_by` | `string` | no | Optional creator identifier |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: clip_create_step
    action: capability:core.clip_create
    params:
      content: "..."  # required
      created_by: "..."
```

## `capability:core.clip_get`

Get a markdown clip by slug

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `slug` | `string` | yes | Clip slug |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: clip_get_step
    action: capability:core.clip_get
    params:
      slug: "..."  # required
```

## `capability:core.clip_health`

Clip subsystem health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: clip_health_step
    action: capability:core.clip_health
```

## `capability:core.health`

Core runtime health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:core.health
```

## `capability:core.http_request`

Perform an outbound HTTP request (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | Request URL |
| `method` | `string` | no | HTTP method (default GET) |
| `headers` | `map[string]any` | no | Request headers |
| `body` | `string` | no | Request body |
| `timeout_seconds` | `number` | no | Timeout in seconds |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: http_request_step
    action: capability:core.http_request
    params:
      url: "..."  # required
      method: "..."
      headers: {}
      body: "..."
      timeout_seconds: 0
```

## `capability:core.kv_delete`

Delete a persistent KV value (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | `string` | yes | Namespace (core/ prefix applied) |
| `key` | `string` | yes | Key |
| `uid` | `string` | no | Owner UID; defaults to instance scope |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: kv_delete_step
    action: capability:core.kv_delete
    params:
      namespace: "..."  # required
      key: "..."  # required
      uid: "..."
```

## `capability:core.kv_get`

Get a persistent KV value

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | `string` | yes | Namespace (core/ prefix applied) |
| `key` | `string` | yes | Key |
| `uid` | `string` | no | Owner UID; defaults to instance scope |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: kv_get_step
    action: capability:core.kv_get
    params:
      namespace: "..."  # required
      key: "..."  # required
      uid: "..."
```

## `capability:core.kv_set`

Set a persistent KV value (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | `string` | yes | Namespace (core/ prefix applied) |
| `key` | `string` | yes | Key |
| `value` | `any` | yes | JSON-serializable value |
| `uid` | `string` | no | Owner UID; defaults to instance scope |
| `ttl_seconds` | `number` | no | Optional TTL |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: kv_set_step
    action: capability:core.kv_set
    params:
      namespace: "..."  # required
      key: "..."  # required
      value: ...  # required
      uid: "..."
      ttl_seconds: 0
```

## `capability:core.notify_health`

Notify subsystem health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: notify_health_step
    action: capability:core.notify_health
```

## `capability:core.notify_send`

Send a notification using a template (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `template_id` | `string` | yes | Template ID to render |
| `channels` | `[]string` | yes | Channels to send to |
| `payload` | `map[string]any` | no | Template data payload |
| `uid` | `string` | no | Owner UID; pipeline injects Event.UID when omitted |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: notify_send_step
    action: capability:core.notify_send
    params:
      template_id: "..."  # required
      channels: ["..."]  # required
      payload: {}
      uid: "..."
```

## `capability:core.run_code`

Run source code in the configured workspace sandbox (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `language` | `string` | yes | python or shell |
| `code` | `string` | yes | Source code |
| `filename` | `string` | no | Optional filename hint |
| `workdir` | `string` | no | Relative workdir under workspace |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: run_code_step
    action: capability:core.run_code
    params:
      language: "..."  # required
      code: "..."  # required
      filename: "..."
      workdir: "..."
```

## `capability:core.run_terminal`

Run a shell command in the configured workspace sandbox (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | `string` | yes | Shell command |
| `workdir` | `string` | no | Relative workdir under workspace |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: run_terminal_step
    action: capability:core.run_terminal
    params:
      command: "..."  # required
      workdir: "..."
```

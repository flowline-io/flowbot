# `core` capability operations

Core runtime primitives: notify, clip, agent, HTTP, sandboxed exec, and KV

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `agent_health`

Agent subsystem health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: agent_health_step
    capability: core
    operation: agent_health
```

## `agent_run`

Execute one autonomous agent turn with a prompt (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | `string` | yes | User prompt |
| `uid` | `string` | no | Owner UID |
| `tools` | `[]string` | no | Tool allowlist |
| `skills` | `[]string` | no | Skill allowlist |
| `memory_scope` | `string` | no | Memory scope; defaults to pipeline name |

**Usage:**

```yaml
  - name: agent_run_step
    capability: core
    operation: agent_run
    params:
      prompt: "..."  # required
      uid: "..."
      tools: ["..."]
      skills: ["..."]
      memory_scope: "..."
```

## `clip_create`

Create a markdown clip and return its public URL (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | `string` | yes | Markdown body |
| `created_by` | `string` | no | Optional creator identifier |

**Usage:**

```yaml
  - name: clip_create_step
    capability: core
    operation: clip_create
    params:
      content: "..."  # required
      created_by: "..."
```

## `clip_get`

Get a markdown clip by slug

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `slug` | `string` | yes | Clip slug |

**Usage:**

```yaml
  - name: clip_get_step
    capability: core
    operation: clip_get
    params:
      slug: "..."  # required
```

## `clip_health`

Clip subsystem health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: clip_health_step
    capability: core
    operation: clip_health
```

## `health`

Core runtime health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: core
    operation: health
```

## `http_request`

Perform an outbound HTTP request (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | Request URL |
| `method` | `string` | no | HTTP method (default GET) |
| `headers` | `map[string]any` | no | Request headers |
| `body` | `string` | no | Request body |
| `timeout_seconds` | `number` | no | Timeout in seconds |

**Usage:**

```yaml
  - name: http_request_step
    capability: core
    operation: http_request
    params:
      url: "..."  # required
      method: "..."
      headers: {}
      body: "..."
      timeout_seconds: 0
```

## `kv_delete`

Delete a persistent KV value (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | `string` | yes | Namespace (core/ prefix applied) |
| `key` | `string` | yes | Key |
| `uid` | `string` | no | Owner UID; defaults to instance scope |

**Usage:**

```yaml
  - name: kv_delete_step
    capability: core
    operation: kv_delete
    params:
      namespace: "..."  # required
      key: "..."  # required
      uid: "..."
```

## `kv_get`

Get a persistent KV value

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | `string` | yes | Namespace (core/ prefix applied) |
| `key` | `string` | yes | Key |
| `uid` | `string` | no | Owner UID; defaults to instance scope |

**Usage:**

```yaml
  - name: kv_get_step
    capability: core
    operation: kv_get
    params:
      namespace: "..."  # required
      key: "..."  # required
      uid: "..."
```

## `kv_set`

Set a persistent KV value (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | `string` | yes | Namespace (core/ prefix applied) |
| `key` | `string` | yes | Key |
| `value` | `any` | yes | JSON-serializable value |
| `uid` | `string` | no | Owner UID; defaults to instance scope |
| `ttl_seconds` | `number` | no | Optional TTL |

**Usage:**

```yaml
  - name: kv_set_step
    capability: core
    operation: kv_set
    params:
      namespace: "..."  # required
      key: "..."  # required
      value: ...  # required
      uid: "..."
      ttl_seconds: 0
```

## `notify_health`

Notify subsystem health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: notify_health_step
    capability: core
    operation: notify_health
```

## `notify_send`

Send a notification using a template (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `template_id` | `string` | yes | Template ID to render |
| `channels` | `[]string` | yes | Channels to send to |
| `payload` | `map[string]any` | no | Template data payload |
| `uid` | `string` | no | Owner UID; pipeline injects Event.UID when omitted |

**Usage:**

```yaml
  - name: notify_send_step
    capability: core
    operation: notify_send
    params:
      template_id: "..."  # required
      channels: ["..."]  # required
      payload: {}
      uid: "..."
```

## `run_code`

Run source code in the configured workspace sandbox (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `language` | `string` | yes | python or shell |
| `code` | `string` | yes | Source code |
| `filename` | `string` | no | Optional filename hint |
| `workdir` | `string` | no | Relative workdir under workspace |

**Usage:**

```yaml
  - name: run_code_step
    capability: core
    operation: run_code
    params:
      language: "..."  # required
      code: "..."  # required
      filename: "..."
      workdir: "..."
```

## `run_terminal`

Run a shell command in the configured workspace sandbox (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | `string` | yes | Shell command |
| `workdir` | `string` | no | Relative workdir under workspace |

**Usage:**

```yaml
  - name: run_terminal_step
    capability: core
    operation: run_terminal
    params:
      command: "..."  # required
      workdir: "..."
```

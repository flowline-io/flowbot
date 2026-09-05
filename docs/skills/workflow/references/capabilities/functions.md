# `functions` capability actions

Named functions (FaaS): pure transform invoke of published function versions. HTTP token/hmac on POST /service/automate/functions/call only; Pipeline and capability.Invoke do not validate function HTTP secrets.

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:functions.get`

Get published function metadata without secrets

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Function name |
| `version` | `number` | no | Optional published version; latest when omitted |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_step
    action: capability:functions.get
    params:
      name: "..."  # required
      version: 0
```

## `capability:functions.health`

Functions subsystem health

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:functions.health
```

## `capability:functions.invoke`

Invoke a published named function version (platform path; does not check function HTTP token/hmac) (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Function name |
| `version` | `number` | yes | Published version to invoke |
| `event` | `any` | no | Event payload passed to the function |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: invoke_step
    action: capability:functions.invoke
    params:
      name: "..."  # required
      version: 0  # required
      event: ...
```

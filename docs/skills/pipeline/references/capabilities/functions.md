# `functions` capability operations

Named functions (FaaS): pure transform invoke of published function versions. HTTP token/hmac on POST /service/automate/functions/call only; Pipeline and capability.Invoke do not validate function HTTP secrets.

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `get`

Get published function metadata without secrets

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Function name |
| `version` | `number` | no | Optional published version; latest when omitted |

**Usage:**

```yaml
  - name: get_step
    capability: functions
    operation: get
    params:
      name: "..."  # required
      version: 0
```

## `health`

Functions subsystem health

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: functions
    operation: health
```

## `invoke`

Invoke a published named function version (platform path; does not check function HTTP token/hmac) (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Function name |
| `version` | `number` | yes | Published version to invoke |
| `event` | `any` | no | Event payload passed to the function |

**Usage:**

```yaml
  - name: invoke_step
    capability: functions
    operation: invoke
    params:
      name: "..."  # required
      version: 0  # required
      event: ...
```

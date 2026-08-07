# `gateway` capability operations

Local CLI gateway: delegate coarse run_cursor jobs to cmd/gateway workers

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `cancel`

Cancel a gateway job by id (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `job_id` | `string` | yes | Job id |

**Usage:**

```yaml
  - name: cancel_step
    capability: gateway
    operation: cancel
    params:
      job_id: "..."  # required
```

## `health`

Report whether a fresh gateway worker is online

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: gateway
    operation: health
```

## `run`

Create a local CLI job and wait for the terminal result (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | `string` | yes | Prompt for the local Cursor CLI |
| `cwd` | `string` | no | Optional workspace path on the worker machine |
| `uid` | `string` | no | Owner UID for audit |
| `cli` | `string` | no | CLI id; only cursor is supported in v1 |

**Usage:**

```yaml
  - name: run_step
    capability: gateway
    operation: run
    params:
      prompt: "..."  # required
      cwd: "..."
      uid: "..."
      cli: "..."
```

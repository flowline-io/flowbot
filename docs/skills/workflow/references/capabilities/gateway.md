# `gateway` capability actions

Local CLI gateway: delegate coarse run_cursor jobs to cmd/gateway workers

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:gateway.cancel`

Cancel a gateway job by id (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `job_id` | `string` | yes | Job id |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: cancel_step
    action: capability:gateway.cancel
    params:
      job_id: "..."  # required
```

## `capability:gateway.health`

Report whether a fresh gateway worker is online

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:gateway.health
```

## `capability:gateway.run`

Create a local CLI job and wait for the terminal result (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | `string` | yes | Prompt for the local Cursor CLI |
| `cwd` | `string` | no | Optional workspace path on the worker machine |
| `uid` | `string` | no | Owner UID for audit |
| `cli` | `string` | no | CLI id; only cursor is supported in v1 |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: run_step
    action: capability:gateway.run
    params:
      prompt: "..."  # required
      cwd: "..."
      uid: "..."
      cli: "..."
```

# `drone` capability actions

Drone CI deploy

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:drone.deploy`

Trigger a build of the deploy repository for the signed-in caller. Creates or updates data.
Example: {} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `uid` | `string` | no | User id for the deploy notification |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: deploy_step
    action: capability:drone.deploy
    params:
      uid: "..."
```

## `capability:drone.health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:drone.health
```

# `drone` capability operations

Drone CI deploy

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `deploy`

Trigger a build of the deploy repository for the signed-in caller. Creates or updates data.
Example: {} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `uid` | `string` | no | User id for the deploy notification |

**Usage:**

```yaml
  - name: deploy_step
    capability: drone
    operation: deploy
    params:
      uid: "..."
```

## `health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: drone
    operation: health
```

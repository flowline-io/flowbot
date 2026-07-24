# `transmission` capability actions

Download capability for Transmission

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:transmission.add`

Add a torrent by magnet or HTTP(S) URL (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | Magnet link or torrent file URL |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: add_step
    action: capability:transmission.add
    params:
      url: "..."  # required
```

## `capability:transmission.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:transmission.health
```

## `capability:transmission.list`

List torrents

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_step
    action: capability:transmission.list
```

## `capability:transmission.remove`

Remove torrents by ID (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `ids` | `array` | yes | Torrent IDs to remove |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: remove_step
    action: capability:transmission.remove
    params:
      ids: ...  # required
```

## `capability:transmission.stop`

Stop torrents by ID (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `ids` | `array` | yes | Torrent IDs to stop |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: stop_step
    action: capability:transmission.stop
    params:
      ids: ...  # required
```

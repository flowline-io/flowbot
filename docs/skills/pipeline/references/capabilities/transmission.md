# `transmission` capability operations

Download capability for Transmission

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `add`

Add a torrent by magnet or HTTP(S) URL (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | Magnet link or torrent file URL |

**Usage:**

```yaml
  - name: add_step
    capability: transmission
    operation: add
    params:
      url: "..."  # required
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: transmission
    operation: health
```

## `list`

List torrents

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: list_step
    capability: transmission
    operation: list
```

## `remove`

Remove torrents by ID (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `ids` | `[]int64` | yes | Torrent IDs to remove |

**Usage:**

```yaml
  - name: remove_step
    capability: transmission
    operation: remove
    params:
      ids: ...  # required
```

## `stop`

Stop torrents by ID (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `ids` | `[]int64` | yes | Torrent IDs to stop |

**Usage:**

```yaml
  - name: stop_step
    capability: transmission
    operation: stop
    params:
      ids: ...  # required
```

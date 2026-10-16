# `miniflux` capability actions

Reader capability

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:miniflux.create_feed`

Create a feed for the signed-in caller. Creates or updates data.
Example: {"feed_url":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `feed_url` | `string` | yes | Feed URL to subscribe to |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_feed_step
    action: capability:miniflux.create_feed
    params:
      feed_url: "..."  # required
```

## `capability:miniflux.health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:miniflux.health
```

## `capability:miniflux.list_entries`

List entries for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `status` | `string` | no | Entry status filter |
| `feed_id` | `int64` | no | Feed ID filter |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_entries_step
    action: capability:miniflux.list_entries
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      status: "..."
      feed_id: ...
```

## `capability:miniflux.list_feeds`

List feeds for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_feeds_step
    action: capability:miniflux.list_feeds
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

## `capability:miniflux.mark_entry_read`

Mark entry as read for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: mark_entry_read_step
    action: capability:miniflux.mark_entry_read
    params:
      id: ...  # required
```

## `capability:miniflux.mark_entry_unread`

Mark entry as unread for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: mark_entry_unread_step
    action: capability:miniflux.mark_entry_unread
    params:
      id: ...  # required
```

## `capability:miniflux.star_entry`

Star an entry for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: star_entry_step
    action: capability:miniflux.star_entry
    params:
      id: ...  # required
```

## `capability:miniflux.unstar_entry`

Unstar an entry for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: unstar_entry_step
    action: capability:miniflux.unstar_entry
    params:
      id: ...  # required
```

# `miniflux` capability operations

Reader capability

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `create_feed`

Create a feed for the signed-in caller. Creates or updates data.
Example: {"feed_url":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `feed_url` | `string` | yes | Feed URL to subscribe to |

**Usage:**

```yaml
  - name: create_feed_step
    capability: miniflux
    operation: create_feed
    params:
      feed_url: "..."  # required
```

## `health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: miniflux
    operation: health
```

## `list_entries`

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

**Usage:**

```yaml
  - name: list_entries_step
    capability: miniflux
    operation: list_entries
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      status: "..."
      feed_id: ...
```

## `list_feeds`

List feeds for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |

**Usage:**

```yaml
  - name: list_feeds_step
    capability: miniflux
    operation: list_feeds
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

## `mark_entry_read`

Mark entry as read for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Usage:**

```yaml
  - name: mark_entry_read_step
    capability: miniflux
    operation: mark_entry_read
    params:
      id: ...  # required
```

## `mark_entry_unread`

Mark entry as unread for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Usage:**

```yaml
  - name: mark_entry_unread_step
    capability: miniflux
    operation: mark_entry_unread
    params:
      id: ...  # required
```

## `star_entry`

Star an entry for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Usage:**

```yaml
  - name: star_entry_step
    capability: miniflux
    operation: star_entry
    params:
      id: ...  # required
```

## `unstar_entry`

Unstar an entry for the signed-in caller. Creates or updates data.
Example: {"id":1} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int64` | yes | Entry ID |

**Usage:**

```yaml
  - name: unstar_entry_step
    capability: miniflux
    operation: unstar_entry
    params:
      id: ...  # required
```

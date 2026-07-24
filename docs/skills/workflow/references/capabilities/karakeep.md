# `karakeep` capability actions

Bookmark capability

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:karakeep.archive`

Archive a bookmark (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: archive_step
    action: capability:karakeep.archive
    params:
      id: "..."  # required
```

## `capability:karakeep.attach_tags`

Attach tags (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |
| `tags` | `[]string` | yes | Tags to attach |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: attach_tags_step
    action: capability:karakeep.attach_tags
    params:
      id: "..."  # required
      tags: ["..."]  # required
```

## `capability:karakeep.check_url`

Check whether a URL exists

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | URL to check |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: check_url_step
    action: capability:karakeep.check_url
    params:
      url: "..."  # required
```

## `capability:karakeep.create`

Create a bookmark (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | URL to bookmark |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_step
    action: capability:karakeep.create
    params:
      url: "..."  # required
```

## `capability:karakeep.delete`

Delete a bookmark (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_step
    action: capability:karakeep.delete
    params:
      id: "..."  # required
```

## `capability:karakeep.detach_tags`

Detach tags (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |
| `tags` | `[]string` | yes | Tags to detach |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: detach_tags_step
    action: capability:karakeep.detach_tags
    params:
      id: "..."  # required
      tags: ["..."]  # required
```

## `capability:karakeep.get`

Get a bookmark

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_step
    action: capability:karakeep.get
    params:
      id: "..."  # required
```

## `capability:karakeep.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:karakeep.health
```

## `capability:karakeep.list`

List bookmarks

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `archived` | `bool` | no | Filter by archive status |
| `favourited` | `bool` | no | Filter by favourite status |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_step
    action: capability:karakeep.list
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      archived: false
      favourited: false
```

## `capability:karakeep.search`

Search bookmarks

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `q` | `string` | no | Search query |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: search_step
    action: capability:karakeep.search
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      q: "..."
```

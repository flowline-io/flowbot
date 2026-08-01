# `karakeep` capability operations

Bookmark capability

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `archive`

Archive a bookmark (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |

**Usage:**

```yaml
  - name: archive_step
    capability: karakeep
    operation: archive
    params:
      id: "..."  # required
```

## `attach_tags`

Attach tags (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |
| `tags` | `[]string` | yes | Tags to attach |

**Usage:**

```yaml
  - name: attach_tags_step
    capability: karakeep
    operation: attach_tags
    params:
      id: "..."  # required
      tags: ["..."]  # required
```

## `check_url`

Check whether a URL exists

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | URL to check |

**Usage:**

```yaml
  - name: check_url_step
    capability: karakeep
    operation: check_url
    params:
      url: "..."  # required
```

## `create`

Create a bookmark (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | `string` | yes | URL to bookmark |

**Usage:**

```yaml
  - name: create_step
    capability: karakeep
    operation: create
    params:
      url: "..."  # required
```

## `delete`

Delete a bookmark (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |

**Usage:**

```yaml
  - name: delete_step
    capability: karakeep
    operation: delete
    params:
      id: "..."  # required
```

## `detach_tags`

Detach tags (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |
| `tags` | `[]string` | yes | Tags to detach |

**Usage:**

```yaml
  - name: detach_tags_step
    capability: karakeep
    operation: detach_tags
    params:
      id: "..."  # required
      tags: ["..."]  # required
```

## `get`

Get a bookmark

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Bookmark ID |

**Usage:**

```yaml
  - name: get_step
    capability: karakeep
    operation: get
    params:
      id: "..."  # required
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: karakeep
    operation: health
```

## `list`

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

**Usage:**

```yaml
  - name: list_step
    capability: karakeep
    operation: list
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      archived: false
      favourited: false
```

## `search`

Search bookmarks

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `q` | `string` | no | Search query |

**Usage:**

```yaml
  - name: search_step
    capability: karakeep
    operation: search
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      q: "..."
```

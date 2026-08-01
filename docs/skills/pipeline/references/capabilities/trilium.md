# `trilium` capability operations

Note capability for note-taking systems

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `create`

Create a note (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | `string` | yes | Note title |
| `content` | `string` | no | Note content |
| `type` | `string` | no | Note type |
| `parent_note_id` | `string` | no | Parent note ID |

**Usage:**

```yaml
  - name: create_step
    capability: trilium
    operation: create
    params:
      title: "..."  # required
      content: "..."
      type: "..."
      parent_note_id: "..."
```

## `delete`

Delete a note (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |

**Usage:**

```yaml
  - name: delete_step
    capability: trilium
    operation: delete
    params:
      id: "..."  # required
```

## `get`

Get a note

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |

**Usage:**

```yaml
  - name: get_step
    capability: trilium
    operation: get
    params:
      id: "..."  # required
```

## `get_app_info`

Get note server info

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: get_app_info_step
    capability: trilium
    operation: get_app_info
```

## `get_content`

Get note content

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |

**Usage:**

```yaml
  - name: get_content_step
    capability: trilium
    operation: get_content
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
    capability: trilium
    operation: health
```

## `list`

List notes

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `query` | `string` | no | Search query filter |

**Usage:**

```yaml
  - name: list_step
    capability: trilium
    operation: list
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      query: "..."
```

## `search`

Search notes

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | yes | Search query string |

**Usage:**

```yaml
  - name: search_step
    capability: trilium
    operation: search
    params:
      query: "..."  # required
```

## `set_content`

Set note content (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |
| `content` | `string` | yes | Note content |

**Usage:**

```yaml
  - name: set_content_step
    capability: trilium
    operation: set_content
    params:
      id: "..."  # required
      content: "..."  # required
```

## `update`

Update a note (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |
| `title` | `string` | no | New title |
| `content` | `string` | no | New content |

**Usage:**

```yaml
  - name: update_step
    capability: trilium
    operation: update
    params:
      id: "..."  # required
      title: "..."
      content: "..."
```

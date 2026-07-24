# `trilium` capability actions

Note capability for note-taking systems

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:trilium.create`

Create a note (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | `string` | yes | Note title |
| `content` | `string` | no | Note content |
| `type` | `string` | no | Note type |
| `parent_note_id` | `string` | no | Parent note ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_step
    action: capability:trilium.create
    params:
      title: "..."  # required
      content: "..."
      type: "..."
      parent_note_id: "..."
```

## `capability:trilium.delete`

Delete a note (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_step
    action: capability:trilium.delete
    params:
      id: "..."  # required
```

## `capability:trilium.get`

Get a note

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_step
    action: capability:trilium.get
    params:
      id: "..."  # required
```

## `capability:trilium.get_app_info`

Get note server info

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_app_info_step
    action: capability:trilium.get_app_info
```

## `capability:trilium.get_content`

Get note content

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_content_step
    action: capability:trilium.get_content
    params:
      id: "..."  # required
```

## `capability:trilium.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:trilium.health
```

## `capability:trilium.list`

List notes

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `query` | `string` | no | Search query filter |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_step
    action: capability:trilium.list
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      query: "..."
```

## `capability:trilium.search`

Search notes

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | yes | Search query string |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: search_step
    action: capability:trilium.search
    params:
      query: "..."  # required
```

## `capability:trilium.set_content`

Set note content (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |
| `content` | `string` | yes | Note content |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: set_content_step
    action: capability:trilium.set_content
    params:
      id: "..."  # required
      content: "..."  # required
```

## `capability:trilium.update`

Update a note (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Note ID |
| `title` | `string` | no | New title |
| `content` | `string` | no | New content |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_step
    action: capability:trilium.update
    params:
      id: "..."  # required
      title: "..."
      content: "..."
```

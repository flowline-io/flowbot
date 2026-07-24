# `memos` capability actions

Memo capability for short-form note-taking

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:memos.create`

Create a memo (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | `string` | yes | Memo content |
| `visibility` | `string` | no | Visibility setting |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_step
    action: capability:memos.create
    params:
      content: "..."  # required
      visibility: "..."
```

## `capability:memos.delete`

Delete a memo (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Memo name |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_step
    action: capability:memos.delete
    params:
      name: "..."  # required
```

## `capability:memos.get`

Get a memo

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Memo name |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_step
    action: capability:memos.get
    params:
      name: "..."  # required
```

## `capability:memos.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:memos.health
```

## `capability:memos.list`

List memos

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
  - id: list_step
    action: capability:memos.list
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

## `capability:memos.update`

Update a memo (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Memo name |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_step
    action: capability:memos.update
    params:
      name: "..."  # required
```

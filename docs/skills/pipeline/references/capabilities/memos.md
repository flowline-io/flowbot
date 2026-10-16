# `memos` capability operations

Memo capability for short-form note-taking

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `create`

Create a memo for the signed-in caller. Creates or updates data.
Example: {"content":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | `string` | yes | Memo content |
| `visibility` | `string` | no | Visibility setting |

**Usage:**

```yaml
  - name: create_step
    capability: memos
    operation: create
    params:
      content: "..."  # required
      visibility: "..."
```

## `delete`

Delete a memo for the signed-in caller. Deletes or removes data.
Example: {"name":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Memo name |

**Usage:**

```yaml
  - name: delete_step
    capability: memos
    operation: delete
    params:
      name: "..."  # required
```

## `get`

Get a memo for the signed-in caller. Read-only.
Example: {"name":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Memo name |

**Usage:**

```yaml
  - name: get_step
    capability: memos
    operation: get
    params:
      name: "..."  # required
```

## `health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: memos
    operation: health
```

## `list`

List memos for the signed-in caller. Read-only.
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
  - name: list_step
    capability: memos
    operation: list
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

## `update`

Update a memo for the signed-in caller. Creates or updates data.
Example: {"name":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Memo name |

**Usage:**

```yaml
  - name: update_step
    capability: memos
    operation: update
    params:
      name: "..."  # required
```

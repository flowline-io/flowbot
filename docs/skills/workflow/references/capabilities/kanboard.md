# `kanboard` capability actions

Kanban capability

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:kanboard.complete_task`

Complete a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: complete_task_step
    action: capability:kanboard.complete_task
    params:
      id: 0  # required
```

## `capability:kanboard.create_task`

Create a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | `string` | no | Task title |
| `description` | `string` | no | Task description |
| `project_id` | `int` | no | Project ID |
| `column_id` | `int` | no | Column ID |
| `tags` | `[]string` | no | Tags to assign |
| `reference` | `string` | no | Reference URL or text |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_task_step
    action: capability:kanboard.create_task
    params:
      title: "..."
      description: "..."
      project_id: 0
      column_id: 0
      tags: ["..."]
      reference: "..."
```

## `capability:kanboard.delete_task`

Delete a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_task_step
    action: capability:kanboard.delete_task
    params:
      id: 0  # required
```

## `capability:kanboard.get_columns`

Get columns

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | `int` | no | Project ID (defaults to 1) |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_columns_step
    action: capability:kanboard.get_columns
    params:
      project_id: 0
```

## `capability:kanboard.get_task`

Get a task

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_task_step
    action: capability:kanboard.get_task
    params:
      id: 0  # required
```

## `capability:kanboard.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:kanboard.health
```

## `capability:kanboard.list_tasks`

List tasks

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `project_id` | `int` | no | Project ID filter |
| `status` | `string` | no | Task status filter |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_tasks_step
    action: capability:kanboard.list_tasks
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      project_id: 0
      status: "..."
```

## `capability:kanboard.move_task`

Move a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |
| `column_id` | `int` | no | Target column ID |
| `position` | `int` | no | Position in column |
| `swimlane_id` | `int` | no | Target swimlane ID |
| `project_id` | `int` | no | Target project ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: move_task_step
    action: capability:kanboard.move_task
    params:
      id: 0  # required
      column_id: 0
      position: 0
      swimlane_id: 0
      project_id: 0
```

## `capability:kanboard.search_tasks`

Search tasks

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `q` | `string` | no | Search query |
| `project_id` | `int` | no | Project ID filter |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: search_tasks_step
    action: capability:kanboard.search_tasks
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      q: "..."
      project_id: 0
```

## `capability:kanboard.update_task`

Update a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |
| `title` | `string` | no | New title |
| `description` | `string` | no | New description |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_task_step
    action: capability:kanboard.update_task
    params:
      id: 0  # required
      title: "..."
      description: "..."
```

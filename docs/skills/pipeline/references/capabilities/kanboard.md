# `kanboard` capability operations

Kanban capability

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `complete_task`

Complete a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |

**Usage:**

```yaml
  - name: complete_task_step
    capability: kanboard
    operation: complete_task
    params:
      id: 0  # required
```

## `create_task`

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

**Usage:**

```yaml
  - name: create_task_step
    capability: kanboard
    operation: create_task
    params:
      title: "..."
      description: "..."
      project_id: 0
      column_id: 0
      tags: ["..."]
      reference: "..."
```

## `delete_task`

Delete a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |

**Usage:**

```yaml
  - name: delete_task_step
    capability: kanboard
    operation: delete_task
    params:
      id: 0  # required
```

## `get_columns`

Get columns

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | `int` | no | Project ID (defaults to 1) |

**Usage:**

```yaml
  - name: get_columns_step
    capability: kanboard
    operation: get_columns
    params:
      project_id: 0
```

## `get_task`

Get a task

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |

**Usage:**

```yaml
  - name: get_task_step
    capability: kanboard
    operation: get_task
    params:
      id: 0  # required
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: kanboard
    operation: health
```

## `list_tasks`

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

**Usage:**

```yaml
  - name: list_tasks_step
    capability: kanboard
    operation: list_tasks
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      project_id: 0
      status: "..."
```

## `move_task`

Move a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |
| `column_id` | `int` | no | Target column ID |
| `position` | `int` | no | Position in column |
| `swimlane_id` | `int` | no | Target swimlane ID |
| `project_id` | `int` | no | Target project ID |

**Usage:**

```yaml
  - name: move_task_step
    capability: kanboard
    operation: move_task
    params:
      id: 0  # required
      column_id: 0
      position: 0
      swimlane_id: 0
      project_id: 0
```

## `search_tasks`

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

**Usage:**

```yaml
  - name: search_tasks_step
    capability: kanboard
    operation: search_tasks
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      q: "..."
      project_id: 0
```

## `update_task`

Update a task (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `int` | yes | Task ID |
| `title` | `string` | no | New title |
| `description` | `string` | no | New description |

**Usage:**

```yaml
  - name: update_task_step
    capability: kanboard
    operation: update_task
    params:
      id: 0  # required
      title: "..."
      description: "..."
```

# `gitea` capability actions

Forge capability

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:gitea.get_commit_diff`

Get commit diff

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `commit_id` | `string` | yes | Commit hash |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_commit_diff_step
    action: capability:gitea.get_commit_diff
    params:
      owner: "..."  # required
      repo: "..."  # required
      commit_id: "..."  # required
```

## `capability:gitea.get_file_content`

Get file content

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `commit_id` | `string` | yes | Commit hash |
| `file_path` | `string` | yes | File path |
| `line_start` | `int` | no | Starting line number |
| `line_count` | `int` | no | Number of lines to fetch |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_file_content_step
    action: capability:gitea.get_file_content
    params:
      owner: "..."  # required
      repo: "..."  # required
      commit_id: "..."  # required
      file_path: "..."  # required
      line_start: 0
      line_count: 0
```

## `capability:gitea.get_issue`

Get an issue

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `index` | `int64` | yes | Issue index number |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_issue_step
    action: capability:gitea.get_issue
    params:
      owner: "..."  # required
      repo: "..."  # required
      index: ...  # required
```

## `capability:gitea.get_repo`

Get a repository

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_repo_step
    action: capability:gitea.get_repo
    params:
      owner: "..."  # required
      repo: "..."  # required
```

## `capability:gitea.get_user`

Get authenticated user

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_user_step
    action: capability:gitea.get_user
```

## `capability:gitea.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:gitea.health
```

## `capability:gitea.list_issues`

List issues

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `state` | `string` | no | Issue state filter (open/closed) |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_issues_step
    action: capability:gitea.list_issues
    params:
      owner: "..."  # required
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      state: "..."
```

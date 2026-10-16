# `github` capability actions

GitHub capability

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:github.get_commit_diff`

Get commit diff for the signed-in caller. Read-only.
Example: {"owner":"example"}

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
    action: capability:github.get_commit_diff
    params:
      owner: "..."  # required
      repo: "..."  # required
      commit_id: "..."  # required
```

## `capability:github.get_file_content`

Get file content for the signed-in caller. Read-only.
Example: {"owner":"example"}

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
    action: capability:github.get_file_content
    params:
      owner: "..."  # required
      repo: "..."  # required
      commit_id: "..."  # required
      file_path: "..."  # required
      line_start: 0
      line_count: 0
```

## `capability:github.get_issue`

Get an issue for the signed-in caller. Read-only.
Example: {"owner":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `number` | `int64` | yes | Issue number |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_issue_step
    action: capability:github.get_issue
    params:
      owner: "..."  # required
      repo: "..."  # required
      number: ...  # required
```

## `capability:github.get_repo`

Get a repository for the signed-in caller. Read-only.
Example: {"owner":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_repo_step
    action: capability:github.get_repo
    params:
      owner: "..."  # required
      repo: "..."  # required
```

## `capability:github.get_user`

Get authenticated user for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_user_step
    action: capability:github.get_user
```

## `capability:github.get_user_by_login`

Get user by login for the signed-in caller. Read-only.
Example: {"login":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `login` | `string` | yes | GitHub username |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_user_by_login_step
    action: capability:github.get_user_by_login
    params:
      login: "..."  # required
```

## `capability:github.health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:github.health
```

## `capability:github.list_issues`

List issues for the signed-in caller. Read-only.
Example: {"owner":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `state` | `string` | no | Issue state (open/closed/all) |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_issues_step
    action: capability:github.list_issues
    params:
      owner: "..."  # required
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      state: "..."
```

## `capability:github.list_notifications`

List notifications for the signed-in caller. Read-only.
Example: {"limit":20}

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
  - id: list_notifications_step
    action: capability:github.list_notifications
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

## `capability:github.list_releases`

List releases for the signed-in caller. Read-only.
Example: {"owner":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_releases_step
    action: capability:github.list_releases
    params:
      owner: "..."  # required
      repo: "..."  # required
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

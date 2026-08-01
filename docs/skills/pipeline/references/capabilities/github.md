# `github` capability operations

GitHub capability

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `get_commit_diff`

Get commit diff

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `commit_id` | `string` | yes | Commit hash |

**Usage:**

```yaml
  - name: get_commit_diff_step
    capability: github
    operation: get_commit_diff
    params:
      owner: "..."  # required
      repo: "..."  # required
      commit_id: "..."  # required
```

## `get_file_content`

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

**Usage:**

```yaml
  - name: get_file_content_step
    capability: github
    operation: get_file_content
    params:
      owner: "..."  # required
      repo: "..."  # required
      commit_id: "..."  # required
      file_path: "..."  # required
      line_start: 0
      line_count: 0
```

## `get_issue`

Get an issue

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `number` | `int64` | yes | Issue number |

**Usage:**

```yaml
  - name: get_issue_step
    capability: github
    operation: get_issue
    params:
      owner: "..."  # required
      repo: "..."  # required
      number: ...  # required
```

## `get_repo`

Get a repository

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |

**Usage:**

```yaml
  - name: get_repo_step
    capability: github
    operation: get_repo
    params:
      owner: "..."  # required
      repo: "..."  # required
```

## `get_user`

Get authenticated user

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: get_user_step
    capability: github
    operation: get_user
```

## `get_user_by_login`

Get user by login

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `login` | `string` | yes | GitHub username |

**Usage:**

```yaml
  - name: get_user_by_login_step
    capability: github
    operation: get_user_by_login
    params:
      login: "..."  # required
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: github
    operation: health
```

## `list_issues`

List issues

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |
| `state` | `string` | no | Issue state (open/closed/all) |

**Usage:**

```yaml
  - name: list_issues_step
    capability: github
    operation: list_issues
    params:
      owner: "..."  # required
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
      state: "..."
```

## `list_notifications`

List notifications

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |

**Usage:**

```yaml
  - name: list_notifications_step
    capability: github
    operation: list_notifications
    params:
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

## `list_releases`

List releases

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `owner` | `string` | yes | Repository owner |
| `repo` | `string` | yes | Repository name |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |
| `sort_by` | `string` | no | Field to sort by |
| `sort_order` | `string` | no | Sort order (asc/desc) |

**Usage:**

```yaml
  - name: list_releases_step
    capability: github
    operation: list_releases
    params:
      owner: "..."  # required
      repo: "..."  # required
      limit: 0
      cursor: "..."
      sort_by: "..."
      sort_order: "..."
```

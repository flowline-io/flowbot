# `confluence` capability actions

Confluence Cloud spaces and pages

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:confluence.create_page`

Create a page (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `space_key` | `string` | no | Space key |
| `title` | `string` | yes | Page title |
| `content` | `string` | no | Storage-format XHTML content |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_page_step
    action: capability:confluence.create_page
    params:
      space_key: "..."
      title: "..."  # required
      content: "..."
```

## `capability:confluence.delete_page`

Delete a page (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_page_step
    action: capability:confluence.delete_page
    params:
      page_id: "..."  # required
```

## `capability:confluence.get_page`

Get a page

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_page_step
    action: capability:confluence.get_page
    params:
      page_id: "..."  # required
```

## `capability:confluence.get_page_content`

Get page storage content

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_page_content_step
    action: capability:confluence.get_page_content
    params:
      page_id: "..."  # required
```

## `capability:confluence.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:confluence.health
```

## `capability:confluence.list_pages`

List pages in a space

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `space_key` | `string` | no | Space key |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_pages_step
    action: capability:confluence.list_pages
    params:
      space_key: "..."
      limit: 0
      cursor: "..."
```

## `capability:confluence.list_spaces`

List spaces

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_spaces_step
    action: capability:confluence.list_spaces
    params:
      limit: 0
      cursor: "..."
```

## `capability:confluence.search_pages`

Search pages with CQL

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `cql` | `string` | yes | CQL query |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: search_pages_step
    action: capability:confluence.search_pages
    params:
      cql: "..."  # required
      limit: 0
      cursor: "..."
```

## `capability:confluence.update_page`

Update a page (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |
| `title` | `string` | no | New title |
| `content` | `string` | no | Storage-format XHTML content |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_page_step
    action: capability:confluence.update_page
    params:
      page_id: "..."  # required
      title: "..."
      content: "..."
```

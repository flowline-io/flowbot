# `confluence` capability operations

Confluence Cloud spaces and pages

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `create_page`

Create a page (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `space_key` | `string` | no | Space key |
| `title` | `string` | yes | Page title |
| `content` | `string` | no | Storage-format XHTML content |

**Usage:**

```yaml
  - name: create_page_step
    capability: confluence
    operation: create_page
    params:
      space_key: "..."
      title: "..."  # required
      content: "..."
```

## `delete_page`

Delete a page (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |

**Usage:**

```yaml
  - name: delete_page_step
    capability: confluence
    operation: delete_page
    params:
      page_id: "..."  # required
```

## `get_page`

Get a page

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |

**Usage:**

```yaml
  - name: get_page_step
    capability: confluence
    operation: get_page
    params:
      page_id: "..."  # required
```

## `get_page_content`

Get page storage content

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |

**Usage:**

```yaml
  - name: get_page_content_step
    capability: confluence
    operation: get_page_content
    params:
      page_id: "..."  # required
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: confluence
    operation: health
```

## `list_pages`

List pages in a space

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `space_key` | `string` | no | Space key |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Usage:**

```yaml
  - name: list_pages_step
    capability: confluence
    operation: list_pages
    params:
      space_key: "..."
      limit: 0
      cursor: "..."
```

## `list_spaces`

List spaces

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Usage:**

```yaml
  - name: list_spaces_step
    capability: confluence
    operation: list_spaces
    params:
      limit: 0
      cursor: "..."
```

## `search_pages`

Search pages with CQL

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `cql` | `string` | yes | CQL query |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Usage:**

```yaml
  - name: search_pages_step
    capability: confluence
    operation: search_pages
    params:
      cql: "..."  # required
      limit: 0
      cursor: "..."
```

## `update_page`

Update a page (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `page_id` | `string` | yes | Page ID |
| `title` | `string` | no | New title |
| `content` | `string` | no | Storage-format XHTML content |

**Usage:**

```yaml
  - name: update_page_step
    capability: confluence
    operation: update_page
    params:
      page_id: "..."  # required
      title: "..."
      content: "..."
```

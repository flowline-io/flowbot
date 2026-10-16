# `nocodb` capability operations

NocoDB bases, tables, and records

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `create_record`

Create a record for the signed-in caller. Creates or updates data.
Example: {"table_id":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `fields` | `object` | yes | Field values |

**Usage:**

```yaml
  - name: create_record_step
    capability: nocodb
    operation: create_record
    params:
      table_id: "..."  # required
      fields: ...  # required
```

## `delete_record`

Delete a record for the signed-in caller. Deletes or removes data.
Example: {"table_id":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `record_id` | `string` | yes | Record ID |

**Usage:**

```yaml
  - name: delete_record_step
    capability: nocodb
    operation: delete_record
    params:
      table_id: "..."  # required
      record_id: "..."  # required
```

## `get_record`

Get a record by ID for the signed-in caller. Read-only.
Example: {"table_id":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `record_id` | `string` | yes | Record ID |

**Usage:**

```yaml
  - name: get_record_step
    capability: nocodb
    operation: get_record
    params:
      table_id: "..."  # required
      record_id: "..."  # required
```

## `get_table`

Get table metadata for the signed-in caller. Read-only.
Example: {"table_id":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |

**Usage:**

```yaml
  - name: get_table_step
    capability: nocodb
    operation: get_table
    params:
      table_id: "..."  # required
```

## `health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: nocodb
    operation: health
```

## `list_bases`

List bases for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: list_bases_step
    capability: nocodb
    operation: list_bases
```

## `list_records`

List records in a table for the signed-in caller. Read-only.
Example: {"table_id":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `limit` | `number` | no | Max records to return |
| `offset` | `number` | no | Record offset |
| `where` | `string` | no | NocoDB where filter |
| `sort` | `string` | no | Sort expression |
| `fields` | `string` | no | Comma-separated field names |

**Usage:**

```yaml
  - name: list_records_step
    capability: nocodb
    operation: list_records
    params:
      table_id: "..."  # required
      limit: 0
      offset: 0
      where: "..."
      sort: "..."
      fields: "..."
```

## `list_tables`

List tables in a base for the signed-in caller. Read-only.
Example: {"base_id":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `base_id` | `string` | yes | Base ID |

**Usage:**

```yaml
  - name: list_tables_step
    capability: nocodb
    operation: list_tables
    params:
      base_id: "..."  # required
```

## `update_record`

Update a record for the signed-in caller. Creates or updates data.
Example: {"table_id":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `record_id` | `string` | yes | Record ID |
| `fields` | `object` | yes | Field values |

**Usage:**

```yaml
  - name: update_record_step
    capability: nocodb
    operation: update_record
    params:
      table_id: "..."  # required
      record_id: "..."  # required
      fields: ...  # required
```

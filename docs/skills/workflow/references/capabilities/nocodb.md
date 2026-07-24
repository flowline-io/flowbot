# `nocodb` capability actions

NocoDB bases, tables, and records

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:nocodb.create_record`

Create a record (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `fields` | `object` | yes | Field values |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_record_step
    action: capability:nocodb.create_record
    params:
      table_id: "..."  # required
      fields: ...  # required
```

## `capability:nocodb.delete_record`

Delete a record (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `record_id` | `string` | yes | Record ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_record_step
    action: capability:nocodb.delete_record
    params:
      table_id: "..."  # required
      record_id: "..."  # required
```

## `capability:nocodb.get_record`

Get a record by ID

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `record_id` | `string` | yes | Record ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_record_step
    action: capability:nocodb.get_record
    params:
      table_id: "..."  # required
      record_id: "..."  # required
```

## `capability:nocodb.get_table`

Get table metadata

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_table_step
    action: capability:nocodb.get_table
    params:
      table_id: "..."  # required
```

## `capability:nocodb.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:nocodb.health
```

## `capability:nocodb.list_bases`

List bases

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_bases_step
    action: capability:nocodb.list_bases
```

## `capability:nocodb.list_records`

List records in a table

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `limit` | `number` | no | Max records to return |
| `offset` | `number` | no | Record offset |
| `where` | `string` | no | NocoDB where filter |
| `sort` | `string` | no | Sort expression |
| `fields` | `string` | no | Comma-separated field names |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_records_step
    action: capability:nocodb.list_records
    params:
      table_id: "..."  # required
      limit: 0
      offset: 0
      where: "..."
      sort: "..."
      fields: "..."
```

## `capability:nocodb.list_tables`

List tables in a base

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `base_id` | `string` | yes | Base ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_tables_step
    action: capability:nocodb.list_tables
    params:
      base_id: "..."  # required
```

## `capability:nocodb.update_record`

Update a record (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `table_id` | `string` | yes | Table ID |
| `record_id` | `string` | yes | Record ID |
| `fields` | `object` | yes | Field values |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_record_step
    action: capability:nocodb.update_record
    params:
      table_id: "..."  # required
      record_id: "..."  # required
      fields: ...  # required
```

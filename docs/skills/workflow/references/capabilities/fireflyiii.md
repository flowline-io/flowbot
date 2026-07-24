# `fireflyiii` capability actions

Finance capability for Firefly III

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:fireflyiii.about`

Get Firefly III about info

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: about_step
    action: capability:fireflyiii.about
```

## `capability:fireflyiii.create_transaction`

Create a transaction (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | yes | Transaction type (withdrawal, deposit, transfer) |
| `date` | `string` | yes | Transaction date (YYYY-MM-DD) |
| `amount` | `string` | yes | Transaction amount |
| `description` | `string` | yes | Transaction description |
| `source_id` | `string` | no | Source account ID |
| `source_name` | `string` | no | Source account name |
| `destination_id` | `string` | no | Destination account ID |
| `destination_name` | `string` | no | Destination account name |
| `category_name` | `string` | no | Category name |
| `notes` | `string` | no | Notes |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_transaction_step
    action: capability:fireflyiii.create_transaction
    params:
      type: "..."  # required
      date: "..."  # required
      amount: "..."  # required
      description: "..."  # required
      source_id: "..."
      source_name: "..."
      destination_id: "..."
      destination_name: "..."
      category_name: "..."
      notes: "..."
```

## `capability:fireflyiii.current_user`

Get current Firefly III user

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: current_user_step
    action: capability:fireflyiii.current_user
```

## `capability:fireflyiii.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:fireflyiii.health
```

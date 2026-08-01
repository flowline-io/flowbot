# `fireflyiii` capability operations

Finance capability for Firefly III

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `about`

Get Firefly III about info

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: about_step
    capability: fireflyiii
    operation: about
```

## `create_transaction`

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

**Usage:**

```yaml
  - name: create_transaction_step
    capability: fireflyiii
    operation: create_transaction
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

## `current_user`

Get current Firefly III user

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: current_user_step
    capability: fireflyiii
    operation: current_user
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: fireflyiii
    operation: health
```

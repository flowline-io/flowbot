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
| `budget_id` | `string` | no | Budget ID |
| `budget_name` | `string` | no | Budget name |
| `bill_id` | `string` | no | Bill ID |
| `bill_name` | `string` | no | Bill name |
| `notes` | `string` | no | Notes |
| `tags` | `[]string` | no | Tags |

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
      budget_id: "..."
      budget_name: "..."
      bill_id: "..."
      bill_name: "..."
      notes: "..."
      tags: ["..."]
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

## `delete_transaction`

Delete a transaction (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Transaction group ID |

**Usage:**

```yaml
  - name: delete_transaction_step
    capability: fireflyiii
    operation: delete_transaction
    params:
      id: "..."  # required
```

## `get_transaction`

Get a transaction by group ID

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Transaction group ID |

**Usage:**

```yaml
  - name: get_transaction_step
    capability: fireflyiii
    operation: get_transaction
    params:
      id: "..."  # required
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

## `list_accounts`

List accounts

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |
| `type` | `string` | no | Account type filter (asset, expense, revenue, ...) |

**Usage:**

```yaml
  - name: list_accounts_step
    capability: fireflyiii
    operation: list_accounts
    params:
      limit: 0
      cursor: "..."
      type: "..."
```

## `list_bills`

List bills

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: list_bills_step
    capability: fireflyiii
    operation: list_bills
    params:
      limit: 0
      cursor: "..."
```

## `list_budgets`

List budgets

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: list_budgets_step
    capability: fireflyiii
    operation: list_budgets
    params:
      limit: 0
      cursor: "..."
```

## `list_categories`

List categories

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: list_categories_step
    capability: fireflyiii
    operation: list_categories
    params:
      limit: 0
      cursor: "..."
```

## `list_tags`

List tags

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: list_tags_step
    capability: fireflyiii
    operation: list_tags
    params:
      limit: 0
      cursor: "..."
```

## `list_transactions`

List transactions

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |
| `type` | `string` | no | Transaction type filter |
| `start` | `string` | no | Start date (YYYY-MM-DD) |
| `end` | `string` | no | End date (YYYY-MM-DD) |
| `account_id` | `string` | no | Filter by account ID |

**Usage:**

```yaml
  - name: list_transactions_step
    capability: fireflyiii
    operation: list_transactions
    params:
      limit: 0
      cursor: "..."
      type: "..."
      start: "..."
      end: "..."
      account_id: "..."
```

## `update_transaction`

Update a transaction (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Transaction group ID |
| `type` | `string` | no | Transaction type (withdrawal, deposit, transfer) |
| `date` | `string` | no | Transaction date (YYYY-MM-DD) |
| `amount` | `string` | no | Transaction amount |
| `description` | `string` | no | Transaction description |
| `source_id` | `string` | no | Source account ID |
| `source_name` | `string` | no | Source account name |
| `destination_id` | `string` | no | Destination account ID |
| `destination_name` | `string` | no | Destination account name |
| `category_name` | `string` | no | Category name |
| `budget_id` | `string` | no | Budget ID |
| `budget_name` | `string` | no | Budget name |
| `bill_id` | `string` | no | Bill ID |
| `bill_name` | `string` | no | Bill name |
| `notes` | `string` | no | Notes |
| `tags` | `[]string` | no | Tags |

**Usage:**

```yaml
  - name: update_transaction_step
    capability: fireflyiii
    operation: update_transaction
    params:
      id: "..."  # required
      type: "..."
      date: "..."
      amount: "..."
      description: "..."
      source_id: "..."
      source_name: "..."
      destination_id: "..."
      destination_name: "..."
      category_name: "..."
      budget_id: "..."
      budget_name: "..."
      bill_id: "..."
      bill_name: "..."
      notes: "..."
      tags: ["..."]
```

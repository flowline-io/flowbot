# `fireflyiii` capability actions

Finance capability for Firefly III

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:fireflyiii.about`

Get Firefly III about info for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: about_step
    action: capability:fireflyiii.about
```

## `capability:fireflyiii.create_transaction`

Create a transaction for the signed-in caller. Creates or updates data.
Example: {"type":"example"} (**mutation**)

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
      budget_id: "..."
      budget_name: "..."
      bill_id: "..."
      bill_name: "..."
      notes: "..."
      tags: ["..."]
```

## `capability:fireflyiii.current_user`

Get current Firefly III user for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: current_user_step
    action: capability:fireflyiii.current_user
```

## `capability:fireflyiii.delete_transaction`

Delete a transaction for the signed-in caller. Deletes or removes data.
Example: {"id":"example"} (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Transaction group ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_transaction_step
    action: capability:fireflyiii.delete_transaction
    params:
      id: "..."  # required
```

## `capability:fireflyiii.get_transaction`

Get a transaction by group ID for the signed-in caller. Read-only.
Example: {"id":"example"}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Transaction group ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_transaction_step
    action: capability:fireflyiii.get_transaction
    params:
      id: "..."  # required
```

## `capability:fireflyiii.health`

Health check for the signed-in caller. Read-only.
Example: {}

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:fireflyiii.health
```

## `capability:fireflyiii.list_accounts`

List accounts for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |
| `type` | `string` | no | Account type filter (asset, expense, revenue, ...) |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_accounts_step
    action: capability:fireflyiii.list_accounts
    params:
      limit: 0
      cursor: "..."
      type: "..."
```

## `capability:fireflyiii.list_bills`

List bills for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_bills_step
    action: capability:fireflyiii.list_bills
    params:
      limit: 0
      cursor: "..."
```

## `capability:fireflyiii.list_budgets`

List budgets for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_budgets_step
    action: capability:fireflyiii.list_budgets
    params:
      limit: 0
      cursor: "..."
```

## `capability:fireflyiii.list_categories`

List categories for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_categories_step
    action: capability:fireflyiii.list_categories
    params:
      limit: 0
      cursor: "..."
```

## `capability:fireflyiii.list_tags`

List tags for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_tags_step
    action: capability:fireflyiii.list_tags
    params:
      limit: 0
      cursor: "..."
```

## `capability:fireflyiii.list_transactions`

List transactions for the signed-in caller. Read-only.
Example: {"limit":20}

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |
| `type` | `string` | no | Transaction type filter |
| `start` | `string` | no | Start date (YYYY-MM-DD) |
| `end` | `string` | no | End date (YYYY-MM-DD) |
| `account_id` | `string` | no | Filter by account ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_transactions_step
    action: capability:fireflyiii.list_transactions
    params:
      limit: 0
      cursor: "..."
      type: "..."
      start: "..."
      end: "..."
      account_id: "..."
```

## `capability:fireflyiii.update_transaction`

Update a transaction for the signed-in caller. Creates or updates data.
Example: {"id":"example"} (**mutation**)

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

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_transaction_step
    action: capability:fireflyiii.update_transaction
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

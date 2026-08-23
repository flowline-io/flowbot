# `trello` capability actions

Trello cloud boards, lists, and cards

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:trello.create_card`

Create a card (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `list_id` | `string` | yes | Target list ID |
| `name` | `string` | yes | Card title |
| `desc` | `string` | no | Card description |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: create_card_step
    action: capability:trello.create_card
    params:
      list_id: "..."  # required
      name: "..."  # required
      desc: "..."
```

## `capability:trello.delete_card`

Delete a card (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_card_step
    action: capability:trello.delete_card
    params:
      card_id: "..."  # required
```

## `capability:trello.delete_webhook`

Delete a Trello webhook (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `webhook_id` | `string` | yes | Webhook ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: delete_webhook_step
    action: capability:trello.delete_webhook
    params:
      webhook_id: "..."  # required
```

## `capability:trello.get_board`

Get a board

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_board_step
    action: capability:trello.get_board
    params:
      board_id: "..."
```

## `capability:trello.get_card`

Get a card

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_card_step
    action: capability:trello.get_card
    params:
      card_id: "..."  # required
```

## `capability:trello.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:trello.health
```

## `capability:trello.list_boards`

List boards

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_boards_step
    action: capability:trello.list_boards
    params:
      limit: 0
      cursor: "..."
```

## `capability:trello.list_cards`

List cards on a board

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_cards_step
    action: capability:trello.list_cards
    params:
      board_id: "..."
      limit: 0
      cursor: "..."
```

## `capability:trello.list_lists`

List lists on a board

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_lists_step
    action: capability:trello.list_lists
    params:
      board_id: "..."
```

## `capability:trello.move_card`

Move a card to another list (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |
| `list_id` | `string` | yes | Target list ID |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: move_card_step
    action: capability:trello.move_card
    params:
      card_id: "..."  # required
      list_id: "..."  # required
```

## `capability:trello.register_webhook`

Register a Trello board webhook (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |
| `callback_url` | `string` | no | Callback URL |
| `description` | `string` | no | Webhook description |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: register_webhook_step
    action: capability:trello.register_webhook
    params:
      board_id: "..."
      callback_url: "..."
      description: "..."
```

## `capability:trello.search_cards`

Search cards

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | yes | Search query |
| `limit` | `int` | no | Maximum results |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: search_cards_step
    action: capability:trello.search_cards
    params:
      query: "..."  # required
      limit: 0
```

## `capability:trello.update_card`

Update a card (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |
| `name` | `string` | no | New title |
| `desc` | `string` | no | New description |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: update_card_step
    action: capability:trello.update_card
    params:
      card_id: "..."  # required
      name: "..."
      desc: "..."
```

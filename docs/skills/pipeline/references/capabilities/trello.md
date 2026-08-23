# `trello` capability operations

Trello cloud boards, lists, and cards

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `create_card`

Create a card (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `list_id` | `string` | yes | Target list ID |
| `name` | `string` | yes | Card title |
| `desc` | `string` | no | Card description |

**Usage:**

```yaml
  - name: create_card_step
    capability: trello
    operation: create_card
    params:
      list_id: "..."  # required
      name: "..."  # required
      desc: "..."
```

## `delete_card`

Delete a card (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |

**Usage:**

```yaml
  - name: delete_card_step
    capability: trello
    operation: delete_card
    params:
      card_id: "..."  # required
```

## `delete_webhook`

Delete a Trello webhook (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `webhook_id` | `string` | yes | Webhook ID |

**Usage:**

```yaml
  - name: delete_webhook_step
    capability: trello
    operation: delete_webhook
    params:
      webhook_id: "..."  # required
```

## `get_board`

Get a board

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |

**Usage:**

```yaml
  - name: get_board_step
    capability: trello
    operation: get_board
    params:
      board_id: "..."
```

## `get_card`

Get a card

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |

**Usage:**

```yaml
  - name: get_card_step
    capability: trello
    operation: get_card
    params:
      card_id: "..."  # required
```

## `health`

Health check

**Inputs (params):**

_(none)_

**Usage:**

```yaml
  - name: health_step
    capability: trello
    operation: health
```

## `list_boards`

List boards

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Usage:**

```yaml
  - name: list_boards_step
    capability: trello
    operation: list_boards
    params:
      limit: 0
      cursor: "..."
```

## `list_cards`

List cards on a board

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Pagination cursor |

**Usage:**

```yaml
  - name: list_cards_step
    capability: trello
    operation: list_cards
    params:
      board_id: "..."
      limit: 0
      cursor: "..."
```

## `list_lists`

List lists on a board

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |

**Usage:**

```yaml
  - name: list_lists_step
    capability: trello
    operation: list_lists
    params:
      board_id: "..."
```

## `move_card`

Move a card to another list (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |
| `list_id` | `string` | yes | Target list ID |

**Usage:**

```yaml
  - name: move_card_step
    capability: trello
    operation: move_card
    params:
      card_id: "..."  # required
      list_id: "..."  # required
```

## `register_webhook`

Register a Trello board webhook (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `board_id` | `string` | no | Board ID |
| `callback_url` | `string` | no | Callback URL |
| `description` | `string` | no | Webhook description |

**Usage:**

```yaml
  - name: register_webhook_step
    capability: trello
    operation: register_webhook
    params:
      board_id: "..."
      callback_url: "..."
      description: "..."
```

## `search_cards`

Search cards

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | `string` | yes | Search query |
| `limit` | `int` | no | Maximum results |

**Usage:**

```yaml
  - name: search_cards_step
    capability: trello
    operation: search_cards
    params:
      query: "..."  # required
      limit: 0
```

## `update_card`

Update a card (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `card_id` | `string` | yes | Card ID |
| `name` | `string` | no | New title |
| `desc` | `string` | no | New description |

**Usage:**

```yaml
  - name: update_card_step
    capability: trello
    operation: update_card
    params:
      card_id: "..."  # required
      name: "..."
      desc: "..."
```

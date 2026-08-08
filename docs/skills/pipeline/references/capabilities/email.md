# `email` capability operations

Email capability for SMTP send and IMAP read

Part of the pipeline capability catalog. Index: [../capabilities.md](../capabilities.md).

## `get`

Get a message by id

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Opaque message id |

**Usage:**

```yaml
  - name: get_step
    capability: email
    operation: get
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
    capability: email
    operation: health
```

## `list`

List messages

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `mailbox` | `string` | no | Mailbox name (default INBOX) |
| `unseen_only` | `bool` | no | Only unseen messages |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: list_step
    capability: email
    operation: list
    params:
      mailbox: "..."
      unseen_only: false
      limit: 0
      cursor: "..."
```

## `mark_read`

Mark a message as read (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Opaque message id |

**Usage:**

```yaml
  - name: mark_read_step
    capability: email
    operation: mark_read
    params:
      id: "..."  # required
```

## `mark_unread`

Mark a message as unread (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Opaque message id |

**Usage:**

```yaml
  - name: mark_unread_step
    capability: email
    operation: mark_unread
    params:
      id: "..."  # required
```

## `search`

Search messages

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `mailbox` | `string` | no | Mailbox name |
| `from` | `string` | no | From filter |
| `to` | `string` | no | To filter |
| `subject` | `string` | no | Subject filter |
| `since` | `string` | no | RFC3339 lower bound |
| `before` | `string` | no | RFC3339 upper bound |
| `unseen` | `bool` | no | Unseen filter |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Usage:**

```yaml
  - name: search_step
    capability: email
    operation: search
    params:
      mailbox: "..."
      from: "..."
      to: "..."
      subject: "..."
      since: "..."
      before: "..."
      unseen: false
      limit: 0
      cursor: "..."
```

## `send`

Send an email (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `to` | `[]string` | yes | Recipient addresses |
| `cc` | `[]string` | no | Cc addresses |
| `bcc` | `[]string` | no | Bcc addresses |
| `subject` | `string` | yes | Subject line |
| `text` | `string` | no | Plain text body |
| `html` | `string` | no | HTML body |
| `from_name` | `string` | no | Display name for From |

**Usage:**

```yaml
  - name: send_step
    capability: email
    operation: send
    params:
      to: ["..."]  # required
      cc: ["..."]
      bcc: ["..."]
      subject: "..."  # required
      text: "..."
      html: "..."
      from_name: "..."
```

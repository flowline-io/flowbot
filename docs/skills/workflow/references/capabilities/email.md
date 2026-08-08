# `email` capability actions

Email capability for SMTP send and IMAP read

Part of the workflow capability catalog. Result envelope and usage patterns: [../capabilities.md](../capabilities.md).

## `capability:email.get`

Get a message by id

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Opaque message id |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: get_step
    action: capability:email.get
    params:
      id: "..."  # required
```

## `capability:email.health`

Health check

**Inputs (params):**

_(none)_

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: health_step
    action: capability:email.health
```

## `capability:email.list`

List messages

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `mailbox` | `string` | no | Mailbox name (default INBOX) |
| `unseen_only` | `bool` | no | Only unseen messages |
| `limit` | `int` | no | Maximum items per page |
| `cursor` | `string` | no | Opaque pagination cursor |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: list_step
    action: capability:email.list
    params:
      mailbox: "..."
      unseen_only: false
      limit: 0
      cursor: "..."
```

## `capability:email.mark_read`

Mark a message as read (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Opaque message id |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: mark_read_step
    action: capability:email.mark_read
    params:
      id: "..."  # required
```

## `capability:email.mark_unread`

Mark a message as unread (**mutation**)

**Inputs (params):**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Opaque message id |

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: mark_unread_step
    action: capability:email.mark_unread
    params:
      id: "..."  # required
```

## `capability:email.search`

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

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: search_step
    action: capability:email.search
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

## `capability:email.send`

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

**Outputs:** `InvokeResult` JSON (see [../capabilities.md](../capabilities.md)). Read domain fields under `data`; use `text` when present.

**Usage:**

```yaml
  - id: send_step
    action: capability:email.send
    params:
      to: ["..."]  # required
      cc: ["..."]
      bcc: ["..."]
      subject: "..."  # required
      text: "..."
      html: "..."
      from_name: "..."
```

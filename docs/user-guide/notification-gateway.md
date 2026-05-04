# Notification Gateway

Unified notification gateway with template-based message rendering, platform-specific overrides, Redis-backed rate limiting, time-window aggregation, and mute/DND rules to prevent notification fatigue.

Source: `pkg/notify/`, `pkg/notify/template/`, `pkg/notify/rules/`, `pkg/ability/notify/`

## Overview

Homelab monitoring produces high-frequency events: disk alerts, download completions, RSS updates, agent status changes. Passively forwarding every event to Slack, Telegram, or ntfy causes notification fatigue -- users mute channels and miss critical alerts.

The Notification Gateway inserts a processing layer between event producers and notification providers to:

1. **Separate data from presentation** -- templates define message formatting, callers provide structured payloads
2. **Rate-limit repetitive alerts** -- prevent a buggy script from sending 1000 messages in one minute
3. **Aggregate batched events** -- collapse 20 RSS fetch events into a single digest every 15 minutes
4. **Honor DND windows** -- silence all notifications during night hours

```
[Pipeline / Cron / Webhook / Agent]
         │
         ▼
┌─────────────────────────────────────────────┐
│            Notification Gateway             │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │  Rule Engine (pkg/notify/rules/)      │  │
│  │  ┌──────────┐ ┌──────────┐ ┌────────┐ │  │
│  │  │ Mute/DND │ │ Throttle │ │Aggregate│ │  │
│  │  └──────────┘ └──────────┘ └────────┘ │  │
│  └─────────────────┬─────────────────────┘  │
│                    ▼                        │
│  ┌───────────────────────────────────────┐  │
│  │  Template Engine (pkg/notify/template/)│  │
│  │  Sprig functions + per-channel overrides│ │
│  └─────────────────┬─────────────────────┘  │
│                    ▼                        │
│  ┌───────────────────────────────────────┐  │
│  │  Channel Router (pkg/notify/)         │  │
│  │  Existing Notifyer registry           │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
         │
         ▼
[Slack Webhook] [ntfy] [Pushover] [Message Pusher]
```

## Architecture

### Data Flow

```
Caller (module, cron, pipeline)
    │  notify.GatewaySend(ctx, uid, templateID, channels, payload)
    ▼
┌──────────────────────────────────────────────────────────┐
│ GatewaySend                                              │
│  1. Resolve template by ID (template.Engine)             │
│  2. For each channel:                                    │
│     a. Evaluate rules (rules.Engine)                     │
│        - Drop → skip                                     │
│        - Mute → skip                                     │
│        - Throttle → check Redis counter, skip if limited │
│        - Aggregate → push to Redis List, set timer       │
│     b. Render template for channel (template.Engine)     │
│     c. Look up user channel config from store            │
│     d. Send via existing notify.Send()                   │
│  3. Background Worker:                                   │
│     - Scans expired aggregate timers every 60s           │
│     - Flushes buffered items, renders digest template    │
│     - Sends single aggregated message                    │
└──────────────────────────────────────────────────────────┘
```

### Pipeline Integration

The gateway can be invoked from pipeline steps via the `notify` capability:

```yaml
pipelines:
  - name: bookmark-notify
    enabled: true
    trigger:
      event: bookmark.created
    steps:
      - name: send-notification
        capability: notify
        operation: send
        params:
          template_id: "bookmark.created"
          channels:
            - slack
            - ntfy
          payload: "{{ .Event.data }}"
```

This uses the [Pipeline Template Engine](pipeline-template.md) to pass event data through to the notification template.

### Direct Invocation

Non-pipeline code (cron jobs, webhook handlers, agent actions) calls `GatewaySend` directly:

```go
import "github.com/flowline-io/flowbot/pkg/notify"

err := notify.GatewaySend(ctx.Context(), ctx.AsUser, "server.offline", []string{"slack", "ntfy"}, map[string]any{
    "hostname": item.Hostname,
    "hostid":   item.Hostid,
})
```

## Template Engine

The template engine renders notification messages using Go `text/template` with the [Sprig](https://masterminds.github.io/sprig/) function library. Sprig provides 70+ template functions for string manipulation, date formatting, math, and type conversion.

### Template Schema

Templates are defined in `flowbot.yaml` under the `notify.templates` key:

```yaml
notify:
  templates:
    - id: bookmark.created
      name: "New Bookmark Notification"
      description: "Triggered when a bookmark is successfully created"
      default_format: markdown
      default_template: |
        **New Bookmark Saved**
        **URL:** {{ .url | default "N/A" }}
        {{ if .title }}**Title:** {{ .title }}{{ end }}
      overrides:
        - channel: telegram
          format: html
          template: |
            <b>New Bookmark Saved</b>
            <b>URL:</b> <a href="{{ .url }}">{{ .url }}</a>
```

### Field Reference

| Field              | Type   | Required | Description                                           |
| ------------------ | ------ | -------- | ----------------------------------------------------- | --------------- |
| `id`               | string | yes      | Unique template identifier (e.g., `bookmark.created`) |
| `name`             | string | yes      | Human-readable display name                           |
| `description`      | string | no       | Text describing when this template is used            |
| `default_format`   | string | yes      | Output format: `markdown` or `html`                   |
| `default_template` | string | yes      | Sprig template body (YAML `                           | ` block scalar) |
| `overrides`        | array  | no       | Per-channel template overrides                        |

### Override Fields

| Field      | Type   | Required | Description                                |
| ---------- | ------ | -------- | ------------------------------------------ |
| `channel`  | string | yes      | Channel name: `slack`, `telegram`, `email` |
| `format`   | string | yes      | Output format for this channel             |
| `template` | string | yes      | Channel-specific template body             |

### Template Data Context

Template payload data is accessed via `{{ .key }}` dot-notation. The payload is a `map[string]any` passed by the caller:

```
{{ .title }}                    -- string field
{{ .url | default "N/A" }}     -- with fallback
{{ .tags | join ", " }}        -- join a string slice
{{ .count | default 0 }}       -- integer with default
{{ .name | upper }}            -- uppercase transform
{{ shorten .text 80 }}         -- truncate with "..."
{{ if .urgent }}URGENT: {{ end }}{{ .title }}  -- conditional
```

### Available Sprig Functions

All [Sprig string functions](https://masterminds.github.io/sprig/strings.html), [date functions](https://masterminds.github.io/sprig/date.html), [math functions](https://masterminds.github.io/sprig/math.html), and [list functions](https://masterminds.github.io/sprig/lists.html) are available. Commonly used:

| Function              | Description            | Example                                  |
| --------------------- | ---------------------- | ---------------------------------------- |
| `upper str`           | Uppercase              | `{{ .name \| upper }}`                   |
| `lower str`           | Lowercase              | `{{ .category \| lower }}`               |
| `default val default` | Default for nil/empty  | `{{ .title \| default "Untitled" }}`     |
| `join sep elems`      | Join slice into string | `{{ .tags \| join ", " }}`               |
| `date format time`    | Format a time value    | `{{ now \| date "2006-01-02" }}`         |
| `now`                 | Current time           | `{{ now \| date "15:04" }}`              |
| `trunc n str`         | Truncate to length     | `{{ .body \| trunc 100 }}`               |
| `contains str substr` | Check substring        | `{{ if contains .body "ERROR" }}`        |
| `replace old new str` | Replace substring      | `{{ .url \| replace "http:" "https:" }}` |
| `quote str`           | Wrap in double quotes  | `{{ .title \| quote }}`                  |
| `toJson val`          | Marshal to JSON        | `{{ .meta \| toJson }}`                  |
| `indent n str`        | Indent each line       | `{{ .body \| indent 2 }}`                |

### Custom Functions

| Function             | Description                                       |
| -------------------- | ------------------------------------------------- |
| `shorten str maxLen` | Truncate and append `"..."` (min output length 4) |

### Template Rendering Logic

1. Gateway passes `templateID` and `channel` to the engine
2. Engine looks up the channel-specific override first
3. If no override exists for the channel, uses the default template
4. Template receives the payload as `.` and renders via `text/template.Execute()`
5. Output includes `Title` (first line, stripped of markdown formatting), `Body` (full rendered output), and `Format`

## Rule Engine

Rules are evaluated before any notification is sent. They are defined in `flowbot.yaml` under `notify.rules` and processed in priority order (higher priority first).

### Rule Schema

```yaml
notify:
  rules:
    - id: "night_mute"
      action: mute
      match:
        event: "*"
        channel: "*"
      condition: "time.hour >= 23 || time.hour < 8"
      priority: 100
```

### Rule Fields

| Field       | Type   | Required | Description                                 |
| ----------- | ------ | -------- | ------------------------------------------- |
| `id`        | string | yes      | Unique rule identifier                      |
| `action`    | string | yes      | `mute`, `throttle`, `aggregate`, or `drop`  |
| `match`     | object | yes      | Event and channel matching criteria         |
| `condition` | string | no       | Time-based expression for conditional rules |
| `priority`  | int    | yes      | Evaluation order (higher = first)           |
| `params`    | object | no       | Action-specific parameters (see below)      |

### Match Fields

| Field     | Type   | Description                                                                                |
| --------- | ------ | ------------------------------------------------------------------------------------------ |
| `event`   | string | Event type pattern: exact match, `*` for all, `prefix.*` for prefix, `*.suffix` for suffix |
| `channel` | string | Channel pattern: same glob syntax as event match                                           |

### Match Examples

| Pattern            | Matches                                   |
| ------------------ | ----------------------------------------- |
| `*`                | Everything                                |
| `bookmark.created` | Exact event type only                     |
| `infra.*`          | `infra.host.down`, `infra.host.up`, etc.  |
| `*.created`        | `bookmark.created`, `kanban.task.created` |
| `server.*`         | `server.offline`, `server.online`         |

### Rule Actions

#### Mute (DND)

Suppresses all matching notifications when the time condition is met. Useful for night-time silence.

```yaml
- id: "night_mute"
  action: mute
  match:
    event: "*"
    channel: "*"
  condition: "time.hour >= 23 || time.hour < 8"
  priority: 100
```

**Condition syntax**: `time.hour >= N`, `time.hour < N`, `time.hour == N` connected with `||` (OR) and `&&` (AND).

#### Throttle

Limits how many notifications of a specific type are sent within a time window. Uses Redis `INCR` with TTL for atomic counting.

```yaml
- id: "infra_throttle"
  action: throttle
  match:
    event: "infra.*"
    channel: "*"
  priority: 50
  params:
    window: "5m"
    limit: 1
```

**Throttle parameters**:

| Field    | Type   | Required | Description                      |
| -------- | ------ | -------- | -------------------------------- |
| `window` | string | yes      | Time window (Go duration format) |
| `limit`  | int    | yes      | Max messages in window           |

Redis key pattern: `notify:throttle:{ruleID}:{eventType}:{channel}`

#### Aggregate

Buffers individual events into a Redis List and flushes them as a single digest message when the window expires. A background worker scans for expired timers every 60 seconds.

```yaml
- id: "download_batch"
  action: aggregate
  match:
    event: "download.completed"
    channel: "telegram"
  priority: 40
  params:
    window: "15m"
    digest_template_id: "download.digest"
```

**Aggregate parameters**:

| Field                | Type   | Required | Description                     |
| -------------------- | ------ | -------- | ------------------------------- |
| `window`             | string | yes      | Aggregation window              |
| `digest_template_id` | string | no       | Template for the digest message |

Redis key pattern: `notify:agg:{ruleID}:{eventType}:{channel}` (List), `notify:agg:timer:{ruleID}:{eventType}:{channel}` (timer key with TTL)

The digest template receives a `.items` field containing all aggregated payloads:

```
**Digest: {{ len .items }} items in the last 15 minutes**
{{ range .items }}
- {{ .title }} ({{ .size }})
{{ end }}
```

#### Drop

Silently discards matching notifications. Useful for suppressing known noise.

```yaml
- id: "drop_test_events"
  action: drop
  match:
    event: "test.*"
    channel: "*"
  priority: 10
```

### Rule Evaluation Order

Rules are sorted by `priority` descending. The first matching rule wins. If a higher-priority mute rule matches, lower-priority throttle or aggregate rules are never evaluated.

Example with two rules:

```yaml
- id: "night_mute"
  action: mute
  match: { event: "*", channel: "*" }
  condition: "time.hour >= 23 || time.hour < 8"
  priority: 100 # Evaluated first

- id: "infra_throttle"
  action: throttle
  match: { event: "infra.*", channel: "*" }
  params: { window: "5m", limit: 1 }
  priority: 50 # Only evaluated if mute doesn't match
```

At 2 PM: night_mute condition is false, so infra_throttle applies to `infra.host.down` events.
At 1 AM: night_mute condition is true, all notifications are silenced regardless of other rules.

## Configuration

### Minimum Setup

1. Add templates to `flowbot.yaml`:

```yaml
notify:
  templates:
    - id: bookmark.created
      name: "Bookmark Created"
      default_format: markdown
      default_template: |
        **New Bookmark**
        {{ .url }}
```

2. Add optional rules:

```yaml
rules: [] # or add rules as needed
```

3. Configure at least one notification channel per user via the `notify config` chat command:

```
/notify config

name:    slack
template: slack://tokenA/tokenB/tokenC
```

### Full Configuration Reference

See [config/notify.yaml](../config/notify.yaml) for a complete template and rule configuration example.

## Predefined Templates

The following templates are built into the reference configuration. Add them to your `flowbot.yaml` to enable notifications for common events.

| Template ID           | Trigger                      | Key Payload Fields                                           |
| --------------------- | ---------------------------- | ------------------------------------------------------------ |
| `bookmark.created`    | Bookmark created             | `url`, `title`                                               |
| `bookmark.archived`   | Bookmark archived            | `id`, `title`                                                |
| `archive.item.added`  | ArchiveBox item added        | `url`, `title`                                               |
| `kanban.task.created` | Kanban task created          | `title`, `project_id`, `description`                         |
| `reader.news.summary` | Daily RSS news summary       | `body`                                                       |
| `server.offline`      | Server offline detection     | `hostname`, `hostid`                                         |
| `finance.transaction` | Finance webhook received     | `amount`, `currency`, `category`, `payee`, `account`, `date` |
| `github.deployment`   | GitHub deployment triggered  | `user`, `repo`, `build`, `drone_url`                         |
| `agent.status`        | Agent online/offline/message | `hostid`, `hostname`, `status`, `message`                    |
| `cron.output`         | Generic cron job output      | `body`, `cron_job`                                           |

## Usage Patterns

### From Pipeline Steps

```yaml
pipelines:
  - name: notify-on-bookmark
    enabled: true
    trigger:
      event: bookmark.created
    steps:
      - name: send-notify
        capability: notify
        operation: send
        params:
          template_id: "bookmark.created"
          channels: ["slack", "ntfy"]
          payload: "{{ .Event.data }}"
```

### From Cron Jobs

```go
func(ctx types.Context) []types.MsgPayload {
    // ... fetch data ...
    err := notify.GatewaySend(ctx.Context(), ctx.AsUser, "reader.news.summary",
        []string{"slack", "ntfy"}, map[string]any{
            "body":        summaryText,
            "entry_count": count,
        })
    if err != nil {
        flog.Error(err)
    }
    return nil
}
```

### From Webhook Handlers

```go
err := notify.GatewaySend(ctx.Context(), ctx.AsUser, "finance.transaction",
    []string{"slack", "ntfy"}, map[string]any{
        "amount":   payload.Amount,
        "currency": payload.Currency,
        "category": payload.Category,
    })
```

### From Agent Actions

```go
err := notify.GatewaySend(ctx.Context(), uid, "agent.status",
    []string{"slack", "ntfy"}, map[string]any{
        "hostid":   hostid,
        "hostname": hostname,
        "status":   "online",
    })
```

## Integration Points

### Call Sites Migrated from event.SendMessage

The following 13 code locations formerly used `event.SendMessage()` and now route through `notify.GatewaySend()`:

| Module            | File            | Template ID           |
| ----------------- | --------------- | --------------------- |
| bookmark          | `event.go:26`   | `bookmark.archived`   |
| bookmark          | `event.go:46`   | `bookmark.created`    |
| bookmark          | `event.go:66`   | `archive.item.added`  |
| kanban            | `event.go:45`   | `kanban.task.created` |
| reader            | `cron.go:116`   | `reader.news.summary` |
| server            | `cron.go:165`   | `server.offline`      |
| finance           | `webhook.go:76` | `finance.transaction` |
| github            | `utils.go:33`   | `github.deployment`   |
| server (internal) | `func.go:328`   | `agent.status`        |
| server (internal) | `func.go:418`   | `agent.status`        |
| server (internal) | `func.go:447`   | `agent.status`        |
| server (internal) | `func.go:457`   | `agent.status`        |
| cron ruleset      | `cron.go:209`   | `cron.output`         |

Interactive chat messages (command responses, form interactions) continue to use `event.SendMessage()` directly.

## Redis Usage

The rule engine uses Redis for three state-tracking patterns:

| Pattern   | Data Structure    | Key Format                                  | TTL         |
| --------- | ----------------- | ------------------------------------------- | ----------- |
| Throttle  | String (counter)  | `notify:throttle:{rule}:{event}:{channel}`  | Rule window |
| Aggregate | List (buffer)     | `notify:agg:{rule}:{event}:{channel}`       | Manual del  |
| Timers    | String (sentinel) | `notify:agg:timer:{rule}:{event}:{channel}` | Rule window |

Throttle counters use atomic `INCR` with `EXPIRE` on first increment, avoiding TOCTOU race conditions. Aggregate lists are flushed by a background worker that scans for expired timer keys.

## Error Handling

- **Missing template**: returns `types.ErrNotFound`, gateway logs warning and skips
- **Template parse error**: caught at startup during `notifytmpl.Init()`, server fails to start
- **Render failure**: logged per-channel, other channels continue
- **Channel not configured**: logged and skipped, no error returned
- **Redis unavailable**: throttle/aggregate operations fail-open (notifications allowed through)
- **Invalid rule window**: logged and rule is skipped (fails-open)

## Testing

```bash
# Template engine tests (11 test cases)
go test ./pkg/notify/template/...

# Rule engine tests (5 test cases)
go test ./pkg/notify/rules/...

# Existing notify tests (13 test cases)
go test ./pkg/notify/...

# Full suite
go tool task test
```

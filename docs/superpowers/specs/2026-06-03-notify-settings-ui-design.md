# Notify Settings UI Design

**Date**: 2026-06-03  
**Status**: Design Complete

## Overview

Add a web UI page for managing notification channels and routing rules. Currently, notify rules and templates are defined only in `flowbot.yaml`, loaded at startup into in-memory engines. This design migrates rules and channel configs to database-backed storage with a management UI, while templates remain in `flowbot.yaml`.

## Scope

- Channel management: list, create, edit, delete, test connectivity for registered notify providers (Slack, ntfy, Pushover, MessagePusher)
- Notification rule editor: create, edit, delete rules with event/channel pattern matching, actions (throttle, aggregate, mute, drop), time conditions, and action-specific parameters
- Silent windows: implemented as "mute" rules with time condition expressions
- Rate limiting: implemented as "throttle" rules with window + limit parameters
- Aggregation: implemented as "aggregate" rules with window + digest template parameters
- DB as sole source of truth for rules and channels (flowbot.yaml notify rules/channels section deprecated)

## Out of Scope

- Template management (remains in `flowbot.yaml`)
- New Email SMTP provider (separate task)
- Per-user channel configuration (separate task)

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Dedicated Ent schemas for rules and channels | Strong typing, proper indexing, queryable by event/channel/action; cleaner than cramming JSON into the generic KV configs table |
| DB is sole source of truth | Simpler mental model; no merge ambiguity between YAML and DB sources |
| Templates stay in YAML | Templates involve Go template syntax and Sprig functions; a textarea editor adds significant complexity without clear benefit |
| No new providers in scope | Existing providers (Slack, ntfy, Pushover, MessagePusher) are sufficient; Email SMTP is a separate task |
| Single tabbed page with inline CRUD | Matches existing Configs page pattern; HTMX partial swaps for all operations |

## Data Model

### notify_channel

Represents a globally-configured notification destination. The URI stored here is used for connectivity testing. Per-user channel URIs (for actual notification delivery) remain in the `configs` table keyed as `notify:<channel_name>` -- that per-user mechanism is unchanged and out of scope for this task. The channel list in this UI serves as a registry of available channel types that rules can reference.

| Field | Type | Attributes | Description |
|-------|------|------------|-------------|
| `id` | bigint | PK, auto | Internal ID |
| `name` | string | unique, not null | Human-readable label, e.g. "Home Slack" |
| `protocol` | string | not null | Protocol identifier: slack, ntfy, pushover, message-pusher |
| `uri` | string | not null | Full connection URI with tokens filled in |
| `enabled` | bool | default true | Whether the channel is active |
| `created_at` | timestamp | not null | |
| `updated_at` | timestamp | not null | |

Indexes: `(protocol)`, `(enabled)`

### notify_rule

Mirrors the existing `config.NotifyRule` struct with database persistence.

| Field | Type | Attributes | Description |
|-------|------|------------|-------------|
| `id` | bigint | PK, auto | Internal ID |
| `rule_id` | string | unique, not null | Logical ID, e.g. "night_mute" |
| `name` | string | not null | Human-readable name |
| `action` | enum | not null | throttle, aggregate, mute, drop |
| `event_pattern` | string | not null, default "*" | Glob pattern for event matching |
| `channel_pattern` | string | not null, default "*" | Glob pattern for channel matching |
| `condition` | string | nullable | Time expression: `time.hour >= 23 \|\| time.hour < 8` |
| `priority` | int | default 0 | Higher values evaluated first |
| `params` | jsonb | default '{}' | Action-specific: window, limit, digest_tpl_id, delayed_send |
| `enabled` | bool | default true | Whether the rule is active |
| `created_at` | timestamp | not null | |
| `updated_at` | timestamp | not null | |

Indexes: `(priority DESC)`, `(enabled)`

### params JSON structure by action

**throttle:**
```json
{"window": "5m", "limit": 1}
```

**aggregate:**
```json
{"window": "10m", "digest_tpl_id": "daily_digest", "delayed_send": false}
```

**mute / drop:**
```json
{}
```

## Store Layer

New methods on the `Adapter` interface in `internal/store/store.go`:

```
// NotifyChannel
CreateNotifyChannel(ctx, *NotifyChannel) error
GetNotifyChannel(ctx, id int64) (*NotifyChannel, error)
ListNotifyChannels(ctx, ListNotifyChannelOptions) ([]NotifyChannel, error)
UpdateNotifyChannel(ctx, *NotifyChannel) error
DeleteNotifyChannel(ctx, id int64) error

// NotifyRule
CreateNotifyRule(ctx, *NotifyRule) error
GetNotifyRule(ctx, id int64) (*NotifyRule, error)
ListNotifyRules(ctx, ListNotifyRuleOptions) ([]NotifyRule, error)
UpdateNotifyRule(ctx, *NotifyRule) error
DeleteNotifyRule(ctx, id int64) error
```

`ListNotifyRuleOptions` supports sorting by priority and an optional `Enabled *bool` filter (nil=all, true=enabled only, false=disabled only). `ListNotifyChannelOptions` supports filtering by protocol and enabled.

Implementation in `internal/store/postgres/adapter.go` using Ent-generated builders.

## Engine Integration

### Startup (internal/server/notify.go)

`initNotificationGateway` loads rules from DB via `ListNotifyRules` (enabled only) instead of `flowbot.yaml`. Builds `[]config.NotifyRule` slice and passes to `rules.Engine.LoadConfig()`.

Templates continue to load from `flowbot.yaml` via `notifytmpl.Init()`.

### Hot Reload

New method `rules.Engine.Reload(ctx)` that:
1. Calls `store.Database.ListNotifyRules(ctx, enabled=true)` 
2. Builds fresh `[]config.NotifyRule`
3. Calls `LoadConfig()` with the new slice (atomically replaces internal rule list)

Called after every rule create/update/delete in the web handler.

### Connectivity Test

`TestNotifyChannel` resolves the channel URI against the registered provider's `Templates()` patterns via `notify.ParseTemplate()`, then dispatches a test `Message{Title: "Test Notification", Body: "Connectivity test from Flowbot", Priority: Low}` through the provider's `Send()`. Uses the authenticated user's UID from the session. Records result in `notification_records` for audit.

## UI Design

### Page: `pkg/views/pages/notify_settings.templ`

Wraps in `@layout.Base("Notification Settings — Flowbot")` with two tabs using DaisyUI tab component (`role="tablist"`). Each tab's content is lazy-loaded via HTMX on first click (`hx-get` + `hx-trigger="click once"`) to avoid loading both tables on initial page render.

**Channels Tab** -- Table of configured channels:
- Columns: Name, Protocol, URI (truncated), Status (enabled/disabled badge), Actions
- Actions per row: Edit, Delete, Test (shows spinner then success/error toast via HTMX trigger)
- "New Channel" button prepends inline form row

**Rules Tab** -- Table of rules sorted by priority descending:
- Columns: Priority, Name, Action (colored badge), Event Pattern, Channel Pattern, Enabled, Actions
- Actions per row: Edit, Delete, Enable/Disable toggle
- "New Rule" button prepends inline form row

### Rule Form Fields (conditional)

| Action | Fields Shown |
|--------|-------------|
| mute | Base + Condition expression input |
| throttle | Base + Window (duration) + Limit (int) |
| aggregate | Base + Window (duration) + Digest Template ID (dropdown populated from `notifytmpl.GetEngine()` template IDs loaded from `flowbot.yaml`) |
| drop | Base only |

Base fields: Name, Rule ID, Action (select dropdown), Event Pattern, Channel Pattern, Priority, Enabled toggle.

### Channel Form Fields

Name, Protocol (select from registered providers via `notify.List()` -- Slack, ntfy, Pushover, MessagePusher), URI (with placeholder hint showing the provider's template patterns), Enabled toggle.

### Partials

```
pkg/views/partials/
  notify_channels_table.templ   -- full table including thead/tbody, empty state row
  notify_channel_row.templ      -- single <tr> in display mode
  notify_channel_form.templ     -- single <tr> in edit/create mode
  notify_rules_table.templ      -- full table including thead/tbody, empty state row
  notify_rule_row.templ         -- single <tr> in display mode
  notify_rule_form.templ        -- single <tr> in edit/create mode
```

### Navigation

Add "Notify Settings" link to navbar in `pkg/views/layout/base.templ` between Notifications and Relations.

## Routes

All under `/service/web/notify-settings`:

```
GET  /notify-settings                       -> notifySettingsPage (full page render)
GET  /notify-settings/channels/list         -> NotifyChannelsTable partial
GET  /notify-settings/channels/new          -> NotifyChannelForm partial (create mode)
POST /notify-settings/channels              -> createChannel, return NotifyChannelRow
GET  /notify-settings/channels/:id/edit     -> NotifyChannelForm partial (edit mode)
PUT  /notify-settings/channels/:id          -> updateChannel, return NotifyChannelRow
DELETE /notify-settings/channels/:id        -> deleteChannel (empty 200 + OOB cleanup)
POST /notify-settings/channels/:id/test     -> testChannel, return HX-Trigger toast

GET  /notify-settings/rules/list            -> NotifyRulesTable partial
GET  /notify-settings/rules/new             -> NotifyRuleForm partial (create mode)
POST /notify-settings/rules                 -> createRule, return NotifyRuleRow, trigger engine reload
GET  /notify-settings/rules/:id/edit        -> NotifyRuleForm partial (edit mode)
PUT  /notify-settings/rules/:id             -> updateRule, return NotifyRuleRow, trigger engine reload
DELETE /notify-settings/rules/:id           -> deleteRule (empty 200), trigger engine reload
```

### Route Registration

New file `internal/modules/web/notify_webservice.go` defines `notifyWebserviceRules []webservice.Rule`.
Registered in `internal/modules/web/module.go` via `module.Webservice(app, Name, notifyWebserviceRules)`.

## Error Handling

- Form validation errors: returned inline in the same form partial with per-field error styling (red border, error message below input)
- Connectivity test failure: `HX-Trigger` response header with `{"showToast": {"type": "error", "message": "Connection failed: ..."}}`; status badge updated via OOB swap
- Connectivity test success: `HX-Trigger` response header with `{"showToast": {"type": "success", "message": "Connection successful"}}`
- Not found: HTTP 404, wraps `types.ErrNotFound`
- Duplicate rule_id or channel name: HTTP 409
- Internal DB errors: HTTP 500, wraps `types.ErrInternal`
- Engine reload failure after CRUD: logged as error, does not block the HTTP response

## Testing

### Unit Tests (table-driven)

- Store methods: create, get, list, update, delete for both notify_channel and notify_rule (minimum 3 cases each)
- `rules.Engine.Reload()`: loads from store, replaces rules, handles empty store, handles store error
- Channel connectivity test: success, provider error, malformed URI

### BDD Specs (Ginkgo/Gomega)

- Full page renders with both tabs
- CRUD lifecycle: create channel/rule -> appears in table -> edit -> updated in table -> delete -> removed
- Conditional form fields change based on action dropdown selection
- Test connectivity shows success/error toast
- Rule priority sorting in table
- Engine reload after rule mutation
- Empty state rendering when no channels/rules exist
- Form validation errors displayed inline

## Files Changed

| File | Action |
|------|--------|
| `internal/store/ent/schema/notify_channel.go` | Create |
| `internal/store/ent/schema/notify_rule.go` | Create |
| `internal/store/store.go` | Add adapter interface methods |
| `internal/store/postgres/adapter.go` | Implement adapter methods |
| `pkg/notify/rules/engine.go` | Add `Reload(ctx)` method |
| `internal/server/notify.go` | Load rules from DB, wire reload callback |
| `internal/modules/web/notify_webservice.go` | Create |
| `internal/modules/web/module.go` | Register notify webservice rules |
| `pkg/views/pages/notify_settings.templ` | Create |
| `pkg/views/partials/notify_channels_table.templ` | Create |
| `pkg/views/partials/notify_channel_row.templ` | Create |
| `pkg/views/partials/notify_channel_form.templ` | Create |
| `pkg/views/partials/notify_rules_table.templ` | Create |
| `pkg/views/partials/notify_rule_row.templ` | Create |
| `pkg/views/partials/notify_rule_form.templ` | Create |
| `pkg/views/layout/base.templ` | Edit |

## Dependencies

- Existing notify providers (Slack, ntfy, Pushover, MessagePusher) -- no changes
- Existing `notify.ParseTemplate()` and `notify.Send()` for channel testing
- Existing Ent ORM code generation (`go tool task ent`)
- Existing DaisyUI + Tailwind CSS (CDN), HTMX, Alpine.js, Toast system
- Existing `layout.Base` template component
- Existing `config.NotifyRule`, `config.NotifyRuleAction`, `config.NotifyRuleMatch`, `config.NotifyRuleParams` types

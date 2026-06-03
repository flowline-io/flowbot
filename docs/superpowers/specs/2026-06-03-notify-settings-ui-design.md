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

**Security**: URIs contain sensitive tokens (API keys, webhook URLs). The URI is stored encrypted at rest using AES-256-GCM (or the existing secret encryption mechanism if one exists). At display time, tokens are masked: e.g. `slack://hooks.slack.com/services/T******/B******/C******`. In edit mode, the URI field uses `type="password"` with a show/hide toggle, and the stored encrypted value is never sent to the client for display -- the field is blank on edit, requiring re-entry or leave-blank-to-keep.

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
3. Calls `LoadConfig()` with the new slice

Called after every rule create/update/delete in the web handler.

### Concurrency Safety

The `rules.Engine` holds its rule list behind a `sync.RWMutex`. `Evaluate()` acquires a read lock; `LoadConfig()` and `Reload()` acquire a write lock. This ensures zero data races between in-flight rule evaluations and hot reloads. All public methods on `Engine` that read rules (`Evaluate`, `CheckThrottle`, `EnqueueForAggregation`, `SetAggregateTimer`) acquire `RLock`; only `LoadConfig` and `Reload` acquire `Lock`. This is verified via `go test -race`.

### Connectivity Test

`TestNotifyChannel` resolves the channel URI against the registered provider's `Templates()` patterns via `notify.ParseTemplate()`, then dispatches a test `Message{Title: "Test Notification", Body: "Connectivity test from Flowbot", Priority: Low}` through the provider's `Send()`. Uses the authenticated user's UID from the session -- the test message is a 1:1 delivery to the configured channel URI only; it does not broadcast to any user group. Records result in `notification_records` for audit.

## Data Validation

All validation happens in the HTTP handler layer before reaching the store, returning 400 with inline form errors on failure.

### Condition Expression Validation

Before persisting a rule with a non-empty `condition`, the handler calls `rules.ValidateCondition(expr string) error` which parses the expression using the same grammar as `evalCondition` in the rules engine. Invalid syntax (e.g. `time.hour >> 5`) is rejected with a descriptive error message in the form.

### Glob Pattern Validation

`event_pattern` and `channel_pattern` are validated using `filepath.Match("*", pattern)` to ensure they are valid glob syntax. Patterns like `infra.[` (unclosed bracket) are rejected before storage.

### Params Validation by Action

Based on the selected `action`, params are unmarshalled into the corresponding struct and validated:

| Action | Validation |
|--------|-----------|
| `throttle` | `window` must parse as `time.Duration` (e.g. "5m", "60s"); `limit` must be > 0 |
| `aggregate` | `window` must parse as `time.Duration`; `digest_tpl_id` must reference an existing template ID from `notifytmpl.GetEngine()` |
| `mute` / `drop` | params must be `{}` (rejected if non-empty) |

## Edge Cases

### Stale Template References

When an aggregate rule's `digest_tpl_id` references a template that no longer exists in `flowbot.yaml`:
- Engine: at evaluation time, if the template is not found, log a warning and fall back to rendering with an empty template (delivers raw payload as body); does not panic.
- UI: the rules table checks each aggregate rule's `digest_tpl_id` against the currently loaded template engine. Missing references are flagged with a warning badge (`⚠ Unknown template`) in the row, prompting the user to update the rule.

### Engine Reload Failure

If `Reload()` encounters a store error, the existing rule set is preserved (not replaced with empty). The error is logged. The HTTP handler still returns success to the user since their data was persisted -- the rules will take effect on the next successful reload or server restart.

## UI Design

### Page: `pkg/views/pages/notify_settings.templ`

Wraps in `@layout.Base("Notification Settings — Flowbot")` with two tabs using DaisyUI tab component (`role="tablist"`). Each tab's content is lazy-loaded via HTMX on first click (`hx-get` + `hx-trigger="click once"`) to avoid loading both tables on initial page render.

**Channels Tab** -- Table of configured channels:
- Columns: Name, Protocol, URI (masked, e.g. `slack://T******/B******`), Status (enabled/disabled badge), Actions
- Actions per row: Edit, Delete, Test (shows spinner then success/error toast via HTMX trigger)
- "New Channel" button prepends inline form row
- Edit form: URI field is `type="password"`, never pre-filled with stored value (leave blank to keep existing)

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

Name, Protocol (select from registered providers via `notify.List()` -- Slack, ntfy, Pushover, MessagePusher), URI (input `type="password"` with show/hide toggle; placeholder hint showing the provider's URI template patterns; left blank on edit to preserve existing value), Enabled toggle.

### URI Masking

Storage helper functions on the store adapter:

- `MaskURI(protocol, uri string) string` -- produces display-safe masked form, e.g. `slack://T******/B******/C******` by splitting on `/` and masking token segments while preserving the scheme prefix and host. Each provider gets a masking implementation based on its URI template structure.
- `EncryptURI(uri string) ([]byte, error)` -- encrypts before DB write.
- `DecryptURI(cipher []byte) (string, error)` -- decrypts for internal use (connectivity test, actual send). Never exposed to UI.

If no existing encryption mechanism exists in the project, encryption is deferred to a follow-up task; masking-only is applied as the minimum viable security measure.

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
DELETE /notify-settings/channels/:id        -> deleteChannel (empty 200, button uses hx-target="closest tr" hx-swap="delete")
POST /notify-settings/channels/:id/test     -> testChannel, return HX-Trigger toast

GET  /notify-settings/rules/list            -> NotifyRulesTable partial
GET  /notify-settings/rules/new             -> NotifyRuleForm partial (create mode)
POST /notify-settings/rules                 -> createRule, return NotifyRuleRow, trigger engine reload
GET  /notify-settings/rules/:id/edit        -> NotifyRuleForm partial (edit mode)
PUT  /notify-settings/rules/:id             -> updateRule, return NotifyRuleRow, trigger engine reload
DELETE /notify-settings/rules/:id           -> deleteRule (empty 200, button uses hx-target="closest tr" hx-swap="delete")
```

### Route Registration

New file `internal/modules/web/notify_webservice.go` defines `notifyWebserviceRules []webservice.Rule`.
Registered in `internal/modules/web/module.go` via `module.Webservice(app, Name, notifyWebserviceRules)`.

## Error Handling

- Form validation errors: returned inline in the same form partial with per-field error styling (red border, error message below input). Covers: empty required fields, invalid condition expression syntax, malformed glob patterns, invalid params (bad duration format, limit <= 0, unknown digest_tpl_id)
- Connectivity test failure: `HX-Trigger` response header with `{"showToast": {"type": "error", "message": "Connection failed: ..."}}`; status badge updated via OOB swap
- Connectivity test success: `HX-Trigger` response header with `{"showToast": {"type": "success", "message": "Connection successful"}}`
- Not found: HTTP 404, wraps `types.ErrNotFound`
- Duplicate rule_id or channel name: HTTP 409
- Internal DB errors: HTTP 500, wraps `types.ErrInternal`
- Engine reload failure after CRUD: logged as error, does not block the HTTP response; existing rule set preserved in engine

## Testing

### Unit Tests (table-driven)

- Store methods: create, get, list, update, delete for both notify_channel and notify_rule (minimum 3 cases each)
- `rules.Engine.Reload()`: loads from store, replaces rules, handles empty store, handles store error
- Concurrency: `Engine.Evaluate()` and `Engine.Reload()` called concurrently from multiple goroutines; verified with `go test -race` (zero data races)
- Channel connectivity test: success, provider error, malformed URI
- URI masking: slack, ntfy, pushover, message-pusher token patterns all masked correctly
- Condition validation: valid expressions pass, syntax errors rejected
- Params validation: valid throttle/aggregate params pass, invalid window format rejected, limit <= 0 rejected
- Glob validation: valid patterns pass, malformed patterns like `[unclosed` rejected
- Stale template handling: engine gracefully handles missing digest_tpl_id (no panic, log warning)
- Dirty data recovery: startup loads rules, encounters a rule with invalid jsonb params -> logs error, skips that rule, loads remaining rules successfully

### BDD Specs (Ginkgo/Gomega)

- Full page renders with both tabs (lazy-loaded)
- CRUD lifecycle: create channel/rule -> appears in table -> edit -> updated in table -> delete -> row removed
- Conditional form fields change based on action dropdown selection (Alpine.js `x-show`)
- Test connectivity shows success/error toast
- Rule priority sorting in table
- Engine reload after rule mutation (hot reload takes effect)
- Empty state rendering when no channels/rules exist
- Form validation errors displayed inline for all field types
- URI field is type="password" in channel form, masked in table
- Stale template references show warning badge in rules table

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

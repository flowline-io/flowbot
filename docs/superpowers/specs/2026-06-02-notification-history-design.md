# Notification History

**Date:** 2026-06-02  
**Status:** Approved  
**Scope:** Database schema + store layer + notify gateway integration + web UI page

## Problem

Notifications are sent through `GatewaySend()` but there is no record of what was sent, to which channel, and whether it succeeded or failed. When a notification doesn't arrive, operators have no way to inspect delivery history, diagnose failures, or retry.

## Design

### Database Schema

New ent schema at `internal/store/ent/schema/notification_record.go`:

| Column | Type | Notes |
|--------|------|-------|
| `id` | int | PK, auto-increment — used as pagination cursor |
| `uid` | string | User ID, indexed for per-user queries |
| `channel` | string | e.g. `slack`, `ntfy`, `pushover` |
| `template_id` | string | Template ID from config, e.g. `bookmark.created` |
| `summary` | string | Human-readable summary from payload `summary` key |
| `status` | enum | `success`, `failed`, `dropped`, `throttled`, `aggregated`, `muted` |
| `error_msg` | string | Provider error text, empty on success |
| `payload_snapshot` | JSON | Original payload map, stored for retry |
| `created_at` | time | Auto timestamp, for display only |

**Indexes:**
- `(uid, id)` — per-user cursor pagination. Query: `WHERE uid = ? AND id < ? ORDER BY id DESC LIMIT ?`.
- `(uid, created_at)` — rolling window cleanup (find oldest records by time per user).

Auto-migrated via existing `client.Schema.Create()` on startup.

**Cursor rationale:** Use `id` (auto-increment PK) rather than `created_at` as the opaque cursor. Timestamps are not guaranteed unique (e.g., a notification broadcast to 3 channels in the same millisecond), which would cause record skipping or duplicates under the limit+1 pattern. Auto-increment ID guarantees strict monotonic ordering.

**Payload type degradation (JSONB):** When `map[string]any` is serialized to JSON and read back for retry, Go unmarshals all numbers as `float64`. The template engine (`text/template` + Sprig) must handle both `int` and `float64` gracefully in numeric function arguments. Existing Sprig functions already coerce numeric types; the `shorten` custom function will be verified for this.

### Store Layer

New `NotifyStore` in `internal/store/store.go`, following the same pattern as `EventStore`:

```go
type NotifyStore struct {
    client *Client
}

func NewNotifyStore(client *Client) *NotifyStore

type ListNotifyRecordsOptions struct {
    Limit  int    // max 100, default 20
    Cursor string // opaque cursor: ID value as string
}
```

**Methods:**

| Method | Purpose |
|--------|---------|
| `Record(ctx, uid, channel, templateID, summary, status, errorMsg string, payload map[string]any) (int, error)` | Insert a delivery record, returns the new row ID |
| `ListRecords(ctx, uid string, opts ListNotifyRecordsOptions) ([]*gen.NotificationRecord, string, error)` | Per-user cursor-paginated history (newest first). Cursor is the last row's ID. |
| `DeleteOldest(ctx, uid string, keepN int) error` | Remove excess records beyond keepN for a user, ordered by `created_at` ascending |

**Key behaviors:**
- `Record` inserts the row only, returns the ID. Does NOT call `DeleteOldest` synchronously.
- Rolling window enforcement is triggered asynchronously from the caller (see Notify Package Integration below).
- `ListRecords` queries: `WHERE uid = ? AND id < ? ORDER BY id DESC LIMIT ?+1`. Cursor is the last returned row's ID, stringified. Same limit+1 pattern as `EventStore`.
- `DeleteOldest` queries: `WHERE uid = ? ORDER BY created_at ASC` with a subquery or offset to find oldest excess rows beyond keepN, then deletes them.

### Notify Package Integration

`GatewaySend()` in `pkg/notify/notify.go` writes a record after each channel send attempt. Rule outcomes (drop/throttle/aggregate/mute) are also recorded so the history shows suppressed notifications.

```go
func GatewaySend(ctx context.Context, uid types.Uid, templateID string,
    channels []string, payload map[string]any) error {

    summary, _ := payload["summary"].(string)

    for _, channel := range channels {
        result := evaluateAndRenderNotification(...)
        if result.action == "drop" || result.action == "mute" || ... {
            go recordAsync(uid, channel, templateID, summary, string(result.action), "", payload)
            continue
        }

        err := Send(templateURI, message)
        status := "success"
        var errMsg string
        if err != nil {
            status = "failed"
            errMsg = err.Error()
        }

        go recordAsync(uid, channel, templateID, summary, status, errMsg, payload)
    }
}

// recordAsync writes the notification record in a goroutine with a 2s timeout,
// then triggers deferred rolling window cleanup.
func recordAsync(uid, channel, templateID, summary, status, errMsg string, payload map[string]any) {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()

        // Record the notification
        notifyStore.Record(ctx, uid, channel, templateID, summary, status, errMsg, payload)

        // Rolling window cleanup (async, best-effort)
        notifyStore.DeleteOldest(ctx, uid, 200)
    }()
}
```

**Key design decisions:**
- **Async recording:** History writes use a goroutine with a 2s timeout. Notification delivery never blocks on DB I/O. If recording fails (timeout, DB down), it's silently dropped — the notification was still sent.
- **Rolling window deferred:** `DeleteOldest` runs in the same goroutine after `Record`. It's not called on every insert synchronously, and it's a single DELETE query per batch of async calls. Since goroutines serialize on the DB connection pool naturally, the actual cleanup burden is minimal for multi-channel sends.
- **Summary:** Extracted from `payload["summary"]`. If absent, the column is empty.
- **Status values:** `success`, `failed`, `dropped`, `throttled`, `aggregated`, `muted` — covers all rule actions plus send outcomes.

**Retry:**
- The web retry handler reads `payload_snapshot` from the failed record and calls `GatewaySend()` with the same `uid`, `templateID`, and payload.
- The channel argument is wrapped in a slice: `[]string{record.Channel}`. This ensures only the failed channel is retried, not all original recipients.
- Retry runs through the full `GatewaySend()` pipeline including rule evaluation. If the notification was previously dropped by a rate limit, the retry may also be throttled. This is desired behavior — operators should see the actual delivery outcome, not a forced bypass. Documented in the UI tooltip: "Retries go through normal delivery, rate limits still apply."

### Web UI

**Routes** (in `internal/modules/web/notification_webservice.go`):

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| `GET` | `/notifications` | `notificationsPage` | Cookie-based, per-user scope |
| `GET` | `/notifications/list` | `notificationsTable` | Cookie-based |
| `POST` | `/notifications/:id/retry` | `retryNotification` | Cookie-based |

All routes use `authenticateWeb()` (cookie validation), consistent with existing web routes. No CSRF token layer exists in the project; this is a cross-cutting concern not addressed here.

**Templates:**

| File | Purpose |
|------|---------|
| `pkg/views/pages/notifications.templ` | Full page wrapping in `@layout.Base("Notifications")` |
| `pkg/views/partials/notifications_table.templ` | Filterable table with cursor pagination, retry button on failed rows |

**Table columns:**

| Time | Channel | Template | Summary | Status |
|------|---------|----------|---------|--------|
| `created_at` | `channel` | `template_id` | `summary` | Badge + retry button |

**Status badges:**
- Green `success`
- Red `failed` — shows error message on hover tooltip. Retry button visible.
- Gray `dropped`, `throttled`, `aggregated`, `muted` — rule action shown in tooltip

**Retry button** (failed rows only):
- `hx-post` to `/service/web/notifications/:id/retry`
- `hx-disabled-elt="this"` — button disables on click, preventing duplicate retry dispatches from rapid double-click
- On success: the row is replaced via HTMX swap with a new success/failed status badge. The old failed row remains visible in the list.
- Tooltip: "Retries go through normal delivery. Rate limits still apply."

**Pagination:** Cursor-based using `id` with "Load more" button at bottom. Appends rows via HTMX. Limit+1 pattern.

**Empty state:** When no records exist, renders `EmptyState("No notifications yet")`.

**Navigation:** "Notifications" link added to `base.templ` nav, between "Events" and "Relations".

**Access control:** `authenticateWeb()` validates the logged-in user. `ListRecords` filters by `uid` from the auth context, so each user only sees their own history. Retry handler verifies the record belongs to the authenticated user.

### Error Handling

- **Store unavailable:** History write goroutine times out silently after 2s. Notification delivery is not affected. Web page shows `EmptyState("Store not available")`.
- **Retry on non-failed record:** Button only renders for `status=failed` rows. If bypassed, re-sending is harmless (creates a new record).
- **Retry record not found or wrong uid:** Handler returns 404 HTML fragment, HTMX swaps it into the row.
- **Double-click retry:** `hx-disabled-elt="this"` disables the button on first click, preventing duplicate dispatches.
- **Payload number type coercion:** Template engine must handle `float64` from JSON unmarshal gracefully. Verified against existing Sprig functions and `shorten` custom function.

# Hub App Management UI

## Scope

Build an HTML UI for managing homelab apps under `/service/web/hub/*`, integrated into the existing web module. Two pages: a list page with auto-polling status and a detail page with SSE log streaming and lifecycle action buttons.

## New Files

| File | Purpose |
|------|---------|
| `internal/modules/web/hub_webservice.go` | Route rules + HTTP handlers |
| `pkg/views/pages/hub_apps.templ` | List page (full page template) |
| `pkg/views/pages/hub_app_detail.templ` | Detail page (full page template) |
| `pkg/views/partials/hub_apps_table.templ` | List table partial (HTMX refresh target) |

## Modified Files

| File | Change |
|------|--------|
| `internal/modules/web/module.go` | Register `hubWebserviceRules` in `Webservice()` and `Rules()` |
| `internal/store/store.go` | Add `HubStore.ListApps(ctx)` to fetch all apps with their `UpdatedAt` timestamp from DB |
| `pkg/views/layout/base.templ` | Add nav link to `/service/web/hub` |

## Routes

All under `/service/web/hub`. All use cookie-based web auth (`route.WithNotAuth()`), consistent with the existing web module.

| Method | Path | Auth | Handler | Returns |
|--------|------|------|---------|---------|
| GET | `/` | cookie | `hubAppsPage` | Full page (`pages.HubApps(...).Render()`) |
| GET | `/list` | cookie | `hubAppsList` | Table partial (HTMX poll target, `hx-trigger="every 10s"`, `hx-swap="outerHTML"`) |
| GET | `/:name` | cookie | `hubAppDetailPage` | Full page |
| GET | `/:name/status` | cookie | `hubAppStatusPartial` | Status badge partial (for HTMX reset after actions) |
| GET | `/:name/logs/stream` | cookie | `hubAppLogsSSE` | `text/event-stream`; each log line emitted as `data: <line>` |
| POST | `/:name/start` | cookie | `hubAppStartAction` | Status partial (HTMX swap into `#status-area`) |
| POST | `/:name/stop` | cookie | `hubAppStopAction` | Status partial |
| POST | `/:name/restart` | cookie | `hubAppRestartAction` | Status partial |
| POST | `/:name/pull` | cookie | `hubAppPullAction` | Status partial |
| POST | `/:name/update` | cookie | `hubAppUpdateAction` | Status partial |

## Status → Badge Mapping

| homelab.AppStatus | Badge Label | Badge Color |
|-------------------|-------------|-------------|
| `running` | online | `badge-success` (green) |
| `stopped` | offline | `badge-ghost` (gray) |
| `unknown` | error | `badge-error` (red) |
| `partial` | warning | `badge-warning` (yellow) |

## Data Flow

### List page
```
homelab.DefaultRegistry.List() → []App (name, status, capabilities)
         +
HubStore.ListApps(ctx) → map[name]UpdatedAt
         ↓
  combine into view model → hub_apps.templ → table partial
         ↑
  hx-get="/list" every 10s (auto-poll)
```

### Detail page
```
homelab.DefaultRegistry.Get(name) → App | homelab.DefaultRegistry.List() for app lookup
homelabRuntime.Status(ctx, app) → AppStatus (live status)

POST start/stop/restart/pull/update:
  homelabRuntime.Xxx(ctx, app)
  → re-check status via homelabRuntime.Status(ctx, app)
  → return status badge partial to swap into page
```

### SSE Logs
```
hubAppLogsSSE:
  homelabRuntime.Logs(ctx, app, tail=100)
  → for each line: w.Write("data: " + line + "\n\n")
  → send "[DONE]" event on completion

Client: EventSource → onmessage appends to log panel
```

### Last operation time
```
HubStore.ListApps(ctx) queries all rows from the `apps` table,
returning AppRecord{Name, UpdatedAt}. This is the last time the
app's record was updated by the scanner (homelab discovery) or
lifecycle operation.
```

## Templates

### `hub_apps.templ` (page)
- Wraps in `@layout.Base("Apps — Flowbot")`
- Page header with "Apps" title
- Renders `@partials.HubAppsTable(items)` for the table body

### `hub_apps_table.templ` (partial)
- `<div id="hub-apps-table">` with `hx-get="/service/web/hub/list" hx-trigger="every 10s" hx-swap="outerHTML"`
- Table columns: Name, Status (badge), Capabilities (tags), Last Updated
- Empty state row when no apps discovered
- Each row links to `/service/web/hub/<name>`

### `hub_app_detail.templ`
- Wraps in `@layout.Base("<name> — Flowbot")`  
- Back link to `/service/web/hub`
- Status section: app name, status badge (target `#status-area`), health
- Action buttons: Start, Stop, Restart, Pull, Update
  - Each POSTs to `/service/web/hub/:name/<action>`, targets `#status-area`
  - Buttons disabled based on `homelab.Permissions` config
  - Show loading spinner via `.htmx-indicator`
- Log panel: `<pre id="log-panel">` with SSE consumer (vanilla JS, no Alpine.js dependency)
  - JS opens `EventSource` on `/service/web/hub/:name/logs/stream?tail=100`
  - Appends each `data:` event to `<pre>`, auto-scrolls

## Store Changes

Add to `HubStore`:
```go
type AppInfo struct {
    Name      string
    UpdatedAt time.Time
}

func (s *HubStore) ListApps(ctx context.Context) ([]AppInfo, error)
```
Queries all rows from the `apps` table, selects `Name` and `UpdatedAt` only.

## Handler Behavior

- All handlers call `authenticateWeb()` (via `route.WithNotAuth()`) — redirects to `/service/web/login` if not authenticated
- Lifecycle action handlers check scope within the handler body via `route.ScopeHandler(ctx, scope)`:
  | Action | Required Scope |
  |--------|---------------|
  | start | `hub:apps:start` |
  | stop | `hub:apps:stop` |
  | restart | `hub:apps:restart` |
  | pull | `hub:apps:pull` |
  | update | `hub:apps:update` |
- Lifecycle action handlers also check `homelab.Permissions` config to disable/hide buttons server-side
- Errors return appropriate HTTP status codes; for HTMX POSTs, return error HTML fragments rather than redirects
- Hub operations that fail (e.g., runtime not configured, permission denied) return user-readable error messages

## Edge Cases

- No apps discovered: show empty state message in list table
- Runtime not configured: action buttons disabled, show "Runtime not configured" hint
- Permission denied for an action: button hidden/disabled
- App not found in detail route: return 404 page
- Log SSE disconnected: EventSource auto-reconnects; on error, close and show message
- Concurrent actions: no explicit locking; docker compose serializes naturally

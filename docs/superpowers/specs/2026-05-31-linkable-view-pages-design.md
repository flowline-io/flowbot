# Linkable View Pages — Design Spec

**Date**: 2026-05-31
**Status**: Draft

## Overview

Add a shareable, token-based view page system to the web module. Links generated from chat responses or notifications open a server-rendered page displaying pipeline step runs, text, images, markdown, or forms. Designed for extensibility: new content types require one template function + one registry entry.

## Approach

Server-rendered via `templ`, using the existing auth middleware (redirect to login if unauthenticated). Content payloads stored in a `page_data` table keyed by opaque tokens generated via `types.Id()` (shortuuid). A Go registry maps `type` strings to `templ.Component` constructors, keeping all rendering logic server-side and aligned with the project's existing templ+HTMX patterns.

## Data Model

### New table: `page_data`

Defined as an ent schema, generated via `go tool task ent`.

| Column      | Type             | Purpose                                   |
| ----------- | ---------------- | ----------------------------------------- |
| `id`        | int              | Primary key, auto-increment               |
| `token`     | varchar(22) unique | Opaque URL token, `types.Id()`          |
| `type`      | varchar(32)      | Content type, maps to template registry   |
| `title`     | text             | Page `<title>` and heading                |
| `data`      | jsonb            | Content payload, schema varies by `type`  |
| `created_by`| varchar(64)      | UID of creator                            |
| `expires_at`| timestamptz      | Auto-cleanup target, null = never expire  |
| `created_at`| timestamptz      |                                           |

### Content type schemas (`data` column by `type`)

**`text`**:
```json
{ "content": "string content here" }
```

**`markdown`**:
```json
{ "content": "# Heading\n\nMarkdown body" }
```

**`image`**:
```json
{ "url": "https://example.com/image.png", "alt": "description" }
```

**`pipeline_run`**:
```json
{ "pipeline_name": "my-pipeline", "run_id": 123 }
```

**`form`**:
```json
{ "fields": [{ "label": "Name", "value": "Alice" }, { "label": "Status", "value": "OK" }] }
```

## Routes

All under existing web module at `/service/web/`. Auth: `authenticateWeb()` via `route.WithNotAuth()` — redirects to login if no session.

| Method | Path                         | Handler          | Purpose                          |
| ------ | ---------------------------- | ---------------- | -------------------------------- |
| `GET`  | `/service/web/view/{token}`  | `viewPage`       | Render page by token             |
| `POST` | `/service/web/view`          | `createView`     | Save payload, return token + URL |
| `DELETE`| `/service/web/view/{token}` | `deleteView`     | Delete a page                    |

### `viewPage(token)` — GET

1. Look up `token` in `page_data` table
2. If not found or expired: render `partials/view_expired.templ` (200, no redirect)
3. If found: look up `type` in template registry, call `fn(data)` to get `templ.Component`
4. Render `pages/view.templ` wrapping the type-specific component with title and expiry banner

### `createView()` — POST

**Request body**:
```json
{
  "type": "pipeline_run",
  "title": "Pipeline my-pipeline Run #123",
  "data": { "pipeline_name": "my-pipeline", "run_id": 123 },
  "expires_at": "2026-06-07T00:00:00Z"
}
```

**Response**:
```json
{
  "token": "3kF9a2B7x...",
  "url": "/service/web/view/3kF9a2B7x..."
}
```

1. Generate token via `types.Id()`
2. Check uniqueness (retry on collision, though shortuuid collision is negligible)
3. Insert row into `page_data`
4. Return `{token, url}`

### `deleteView(token)` — DELETE

Removes the row from `page_data`. Returns 204 on success, 404 if not found.

## Template Registry

### `internal/modules/web/types.go`

```go
var viewTemplates = map[string]func(types.KV) templ.Component{
    "text":          textView,
    "markdown":      markdownView,
    "image":         imageView,
    "pipeline_run":  pipelineRunView,
    "form":          formView,
}
```

Each function receives `types.KV` (the `data` column) and returns a `templ.Component`. Adding a new type = one function + one map entry. The `data` argument uses `types.KV` (not raw `map[string]any`), consistent with project conventions.

## Templates

### `pkg/views/pages/view.templ`

Full page wrapper extending `layout/base.templ`:
- Renders `<h1>` from `title`
- Delegates body to the type-specific component from the registry
- Shows expiry banner if `expires_at` is set and in the past
- Page `<title>` set to `{title} — Flowbot`

### `pkg/views/partials/view_text.templ`

Renders `{{ .content }}` in a `<pre>` block with `whitespace: pre-wrap`.

### `pkg/views/partials/view_markdown.templ`

Server-side Markdown to sanitized HTML (using existing `bluemonday` dependency). Rendered in a `<div>` with prose styling.

### `pkg/views/partials/view_image.templ`

Renders `<img src="{{ .url }}" alt="{{ .alt }}">` with responsive Tailwind classes.

### `pkg/views/partials/view_pipeline_run.templ`

The handler fetches step runs from DB via `store.PipelineStore.GetStepRunsByRunID(rid)` where `rid` is `types.KV.Int64("run_id")`, then passes the step run list into the template data. The template renders the existing `partials.PipelineStepRunsDetail` partial with the pre-fetched step run list. No store dependency in the template function itself — data is loaded by the handler before rendering.

### `pkg/views/partials/view_form.templ`

Read-only form grid: two-column layout (`label` left, `value` right) for each entry in `types.KV.List("fields")`.

### `pkg/views/partials/view_expired.templ`

Clean "Page not found or expired" message, no redirect. Extends `layout/base.templ`.

## Store Layer

### New methods in `internal/store/store.go`

```go
// PageDataStore manages sharable view page storage.
type PageDataStore struct { ... }

// CreatePageData inserts a new page_data row. Returns the token.
func (s *PageDataStore) CreatePageData(ctx context.Context, token string, pageType string, title string, data types.KV, createdBy string, expiresAt *time.Time) error

// GetPageDataByToken retrieves a page_data row by token.
func (s *PageDataStore) GetPageDataByToken(ctx context.Context, token string) (*ent.PageData, error)

// DeletePageData removes a page_data row by token.
func (s *PageDataStore) DeletePageData(ctx context.Context, token string) error

// DeleteExpiredPageData removes rows where expires_at < now().
func (s *PageDataStore) DeleteExpiredPageData(ctx context.Context) (int64, error)
```

Implemented via ent `PageData` client after running `go tool task ent` to generate the schema.

## Cleanup Cron

Add a daily job in `internal/server/cron.go` (existing cron pattern) that calls `store.PageDataStore.DeleteExpiredPageData()`.

## Caller Integration

### Pipeline

After pipeline run completes, a post-run step or the engine calls `POST /service/web/view` with `{type: "pipeline_run", title: "...", data: {pipeline_name, run_id}}` and embeds the returned URL in the notification payload:

```go
viewResp := createView(ctx, types.KV{
    "type":  "pipeline_run",
    "title": "Pipeline " + name + " Run #" + strconv.FormatInt(runID, 10),
    "data":  types.KV{"pipeline_name": name, "run_id": runID},
})
payload["url"] = viewResp["url"]
notify.GatewaySend(uid, "pipeline_complete", channels, payload)
```

### Chat

Module chat handlers call `createView()` before sending the chat reply, embedding the URL in the response text or as a rich link.

### Notification

No changes needed in `pkg/notify/` — `buildNotifyMessage()` already reads `payload["url"]` and sets `msg.Url`.

## File Layout

```
internal/modules/web/
├── view_webservice.go     # Routes: GET /view/{token}, POST /view, DELETE /view/{token}
├── types.go               # Type registry: map[string]func(types.KV) templ.Component
├── view_webservice_test.go # HTTP handler tests
internal/store/
├── store.go               # + PageDataStore methods
pkg/views/
├── pages/view.templ       # Full page wrapper
├── partials/
│   ├── view_text.templ
│   ├── view_markdown.templ
│   ├── view_image.templ
│   ├── view_pipeline_run.templ
│   ├── view_form.templ
│   └── view_expired.templ
internal/server/
├── cron.go                # + expire cleanup job
```

## Testing

### Unit tests (`internal/modules/web/view_webservice_test.go`)

All tests follow table-driven pattern with `t.Run`. Each table has at least 3 cases.

| Test function         | Cases                                                                    |
| --------------------- | ------------------------------------------------------------------------ |
| `TestViewPage`        | valid token → 200, missing token → 404/expired page, expired token → expired page, unauthenticated → redirect to login |
| `TestCreateView`      | valid payload → 201 + token, missing `type` → 400, missing `data` → 400 |
| `TestDeleteView`      | valid token → 204, missing token → 404, unauthorized → 403              |

### Template tests (`internal/modules/web/types_test.go`)

Each template function tested with valid data and edge cases (empty content, missing fields).

### Store tests (`internal/store/store_test.go`)

`PageDataStore` methods: create, get by token, not found, delete, expire cleanup.

### BDD integration tests (`tests/`)

Ginkgo spec: create a view → GET the view URL → verify rendered content → delete → verify gone. Requires Docker for PostgreSQL.

## Implementation Order

1. `go tool task ent` — add `page_data` schema, generate ent code
2. `internal/store/store.go` — add `PageDataStore` with `CreatePageData`, `GetPageDataByToken`, `DeletePageData`, `DeleteExpiredPageData`
3. `internal/modules/web/types.go` — type registry with initial five template functions
4. `pkg/views/pages/view.templ` — page wrapper
5. `pkg/views/partials/view_*.templ` — five content type templates + expired page
6. `go tool task templ` — generate `*_templ.go` files
7. `internal/modules/web/view_webservice.go` — `viewPage`, `createView`, `deleteView` handlers
8. `internal/server/cron.go` — expire cleanup job
9. Unit tests for store, templates, handlers
10. BDD integration spec
11. `go tool task lint && go tool task test` — verify

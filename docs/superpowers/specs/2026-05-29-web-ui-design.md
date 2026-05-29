# Web UI Stack — Design Spec

**Date**: 2026-05-29
**Status**: Approved

## Overview

Introduce a server-rendered web UI layer using Go Fiber + Templ + HTMX + Alpine.js + Tailwind CSS. The first module is `internal/modules/web`, which provides a CRUD interface for the `configs` database table as a reference implementation.

## Tech Stack

| Layer | Tech | Role |
|-------|------|------|
| HTTP Server | Fiber v3 | Routes, middleware, auth |
| Templates | Templ | Type-safe server-side HTML rendering |
| Interactivity | HTMX | Partial page updates, form submissions |
| UI Styling | Tailwind CSS v4 | Utility-first CSS |
| Lightweight JS | Alpine.js | Toggle/dropdown/transitions (Alpine-only) |

## Directory Structure

```
flowbot/
├── package.json                    # NEW: Tailwind CLI + Alpine, prettier
├── public/                         # NEW: Static assets (Fiber Static)
│   ├── css/
│   │   ├── input.css               # Tailwind v4 CSS entry point
│   │   └── styles.css              # Built output (gitignored)
│   └── js/
│       └── app.js                  # Alpine data, HTMX extensions
├── internal/
│   ├── modules/
│   │   ├── fx.go                   # MODIFIED: add web.Register
│   │   └── web/                    # NEW: Web UI module
│   │       ├── module.go           # moduleHandler, Register(), Init()
│   │       ├── webservice.go       # HTTP handlers (page + HTMX routes)
│   │       └── module_test.go      # Handler tests
│   └── store/
│       └── store.go                # MODIFIED: add ListConfigs()
├── pkg/
│   ├── types/
│   │   └── model/                  # NEW: UI model structs
│   │       └── config.go           # ConfigItem struct
│   └── views/                      # NEW: .templ templates
│       ├── layout/
│       │   └── base.templ          # HTML skeleton
│       ├── pages/
│       │   └── configs.templ       # Config list page
│       └── partials/
│           ├── config_table.templ  # Config table partial
│           ├── config_row.templ    # Single config row
│           └── config_form.templ   # Create/edit inline form
```

## Module Integration

The web module follows the standard module pattern:

- Implements `module.Handler` with `module.Base`
- `Register()` calls `module.Register("web", &handler)`
- `Init()` checks `enabled` flag from JSON config
- `Webservice(app)` calls `module.Webservice(app, "web", webserviceRules)`
- Registered in `internal/modules/fx.go` via `fx.Invoke(web.Register)`
- Routes mounted under `/service/web/*` with standard auth middleware

## Routes

All routes require authentication (no `WithNotAuth()`).

### Page-level routes

| Method | Path | Handler | Returns |
|--------|------|---------|---------|
| `GET` | `/service/web/configs` | `configsPage` | `layout.Base(pages.ConfigsPage(initialItems))` — server-side pre-render, no empty flash |

### HTMX partial routes

| Method | Path | Handler | Returns |
|--------|------|---------|---------|
| `GET` | `/service/web/configs/list` | `listConfigs` | `partials.ConfigTable(result)` |
| `GET` | `/service/web/configs/{id}` | `getConfig` | `partials.ConfigRow(item)` |
| `GET` | `/service/web/configs/new` | `newConfigForm` | `partials.ConfigForm(empty)` |
| `POST` | `/service/web/configs` | `createConfig` | `partials.ConfigRow` (success) or `partials.ConfigForm` (422) |
| `GET` | `/service/web/configs/{id}/edit` | `editConfigForm` | `partials.ConfigForm(item)` |
| `PUT` | `/service/web/configs/{id}` | `updateConfig` | `partials.ConfigRow` (success) or `partials.ConfigForm` (422) |
| `DELETE` | `/service/web/configs/{id}` | `deleteConfig` | Empty 200 (HTMX removes DOM) |

## Data Model

`pkg/types/model/config.go`:

```go
type ConfigItem struct {
    ID        int64
    UID       string
    Topic     string
    Key       string
    Value     types.KV
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## Store Layer

New method in `internal/store/store.go`:

```go
type ListConfigOptions struct {
    Offset int
    Limit  int
    Search string
}

ListConfigs(ctx context.Context, opts ListConfigOptions) ([]ConfigItem, error)
```

Queries all configs across uids/topics with optional search and pagination. Uses the ent `ConfigData` client directly, ordering by `updated_at DESC`.

The existing `ConfigSet`, `ConfigGet`, `ConfigDelete` methods are used for individual CRUD with the uid and topic from the auth context.

**IMPORTANT**: All database query code stays in `store.go` per project rules.

## View Components

### `layout/base.templ`
HTML skeleton with `<head>`, Tailwind CDN (dev), Alpine.js CDN, HTMX CDN, global nav bar. Uses `@templ.Children()` slot for page content.

### `pages/configs.templ`
Full page embedding `layout.Base`. Receives `[]ConfigItem` pre-fetched by the handler. Contains the "New Config" button and renders the config table directly from initial data — no `hx-get` on load. The table remains a valid HTMX target for subsequent partial refreshes (search, pagination, manual reload).

### `partials/config_table.templ`
Renders a `<table>` with headers (ID, UID, Topic, Key, Value preview, Actions). Body is `id="configs-rows"` containing all rows.

### `partials/config_row.templ`
Single `<tr>` with `hx-target="this"`. Displays one `ConfigItem`. "Edit" triggers `hx-get` to swap row with inline form. "Delete" triggers `hx-delete` with confirm dialog.

### `partials/config_form.templ`
Inline form with `uid`, `topic`, `key` inputs and `value` textarea (JSON). Used for both create (`hx-post`) and edit (`hx-put`). On validation errors (422), re-renders same form with error messages.

## HTMX Interaction Flows

### Initial page load
Handler fetches data from store, passes `[]ConfigItem` directly to `pages.ConfigsPage()`. No client-side `hx-get` on load — zero round-trips for first paint.

### Inline create
Click "New Config" → `partials.ConfigForm` inserted at top of table. Submit POST → if 200, returns `partials.ConfigRow` swapped in place of the form. No subsequent full table refresh — single network call, instant result.

### Inline edit
Click "Edit" → row replaced with `partials.ConfigForm` (pre-filled). Submit PUT → if 200, swaps form back to `partials.ConfigRow`. If 422, form remains with errors.

### Delete
Click "Delete" → `hx-confirm` dialog, `hx-delete`, server returns 200 empty → `hx-target` row removed from DOM.

### Manual refresh
A "Refresh" button on the page emits `hx-get="/service/web/configs/list"` targeting the table container. Used for explicit reload, search, or pagination.

## Error Handling

| Scenario | HTTP Status | Response |
|----------|-------------|----------|
| Validation failure | 422 | Re-render `partials.ConfigForm` with error messages |
| Not found (edit/delete) | 404 | `<div class="text-red-500">` with message |
| Store error | 500 | Generic error partial, logged server-side |
| Auth failure | 401 | Handled by `Authorize` middleware |

All errors use `types.Errorf(types.ErrXxx, ...)`. No `panic`.

## Build Tooling

### `package.json`
```json
{
  "devDependencies": {
    "tailwindcss": "^4.0.0",
    "@tailwindcss/cli": "^4.0.0",
    "prettier": "^3.4.0",
    "prettier-plugin-tailwindcss": "^0.6.0"
  }
}
```

### `taskfile.yaml` additions
```yaml
templ:   go tool templ generate
css:     npx @tailwindcss/cli -i ./public/css/input.css -o ./public/css/styles.css
css:min: npx @tailwindcss/cli -i ./public/css/input.css -o ./public/css/styles.css --minify
web:     go tool task templ && go tool task css
```

### `go.mod` additions
```
tool github.com/a-h/templ/cmd/templ
```

## Out of Scope

- Login page — assumed token is set via cookie/header externally
- Production asset bundling — CDN used in dev; production pipeline TBD
- Air config — documented but not in scope for initial setup
- `pkg/views/` AGENTS.md — not in scope for this spec

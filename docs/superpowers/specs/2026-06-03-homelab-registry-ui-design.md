# Homelab Registry UI Design

**Date**: 2026-06-03
**Status**: Design Complete

## Overview

Add a dedicated Homelab Registry UI page with a card-based layout for browsing discovered homelab applications. This is a separate page from the existing `/service/web/hub` (which focuses on lifecycle management: start/stop/restart/logs). The new registry page emphasizes discovery metadata: app cards with icons, URLs, online status, capability types, searchable/filterable, and a detail view with exposed endpoints and version info.

## Scope

- Card-based app list with search and capability-type filter (Alpine.js client-side filtering)
- App detail page: services table, exposed endpoints list, version from image tag, last discovery time
- Manual rescan button that re-runs Scanner + Probe engine + Registry Replace
- Navbar link: "Registry"
- Existing `/service/web/hub` pages remain unchanged

## Out of Scope

- Lifecycle operations (start/stop/restart) — stay on the existing hub page
- Log streaming — stays on the existing hub detail page
- Adding new fields to the `App` struct
- Adding new Ent schemas for homelab registry data
- Replacing the existing hub apps page

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Separate page from existing `/hub` | Existing hub page handles lifecycle operations (start/stop/logs); registry page is discovery-focused. Different routes, different navbar links. |
| Alpine.js client-side filtering | Fewer than 50 apps expected; loading all data once and filtering client-side gives instant UX with no network round trips. Matches existing `event_filters.templ` pattern. |
| Initial-letter avatar for icons | No icon field on `App` struct. Initial-letter circles are simple, deterministic, and require no new data. Matches common UI patterns. |
| First capability endpoint as card URL | `App.Capabilities[0].Endpoint.BaseURL` is the most meaningful address — it's the actual reachable URL discovered by the probe engine. |
| Version from image tag | Parse `ComposeService.Image` (e.g., `gitea/gitea:1.22.3`) to extract the tag as version. First service with a tagged image wins. |
| Last discovery time from store | `SaveHomelabApps()` already persists to `hub_apps` table with `UpdatedAt`. Reuse `loadUpdatedAts()` pattern from existing hub page. |
| Rescan re-runs full flow | POST endpoint calls the same Scanner.Scan() + ProbeAll() + Replace pipeline as server startup. Extracted into a reusable exported function in `internal/server/`. |

## Route Design

All routes under `/service/web/homelab`, registered in `module.go` alongside existing web routes:

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/homelab` | `homelabPage` | Full page with Alpine.js card grid |
| GET | `/homelab/:name` | `homelabDetailPage` | Single app detail |
| POST | `/homelab/rescan` | `homelabRescan` | Trigger scan+probe, HX-Redirect |

Route rules use `route.WithNotAuth()` (cookie-based auth, no scope check), matching existing hub routes.

## Template Design

### New files

| File | Package | Purpose |
|------|---------|---------|
| `pkg/views/pages/homelab.templ` | `pages` | Full page wrapping `@layout.Base("Registry — Flowbot")`, contains Alpine.js `homelabRegistry()` controller |
| `pkg/views/pages/homelab_detail.templ` | `pages` | Server-rendered detail page for a single app |
| `pkg/views/partials/homelab_card.templ` | `partials` | Single card: avatar, name, URL link, status badge, capability badges |
| `pkg/views/partials/homelab_grid.templ` | `partials` | Search bar + filter dropdown + rescan button + card grid + empty state |

### Modified files

| File | Change |
|------|--------|
| `pkg/views/layout/base.templ` | Add "Registry" link to navbar between "Apps" and "Capabilities" |
| `internal/modules/web/module.go` | Register `homelabWebserviceRules` |
| `internal/server/homelab.go` | Extract `RunHomelabScan()` exported function for reuse |

### Page: Registry card list (`homelab.templ`)

Alpine.js controller injected via `@templ.JSONScript("homelab-data", apps)`:

```
x-data="homelabRegistry()" x-init="init($data, $el)"
```

Controller state:
- `apps []App` — full dataset from server (JSONScript seed)
- `search string` — bound to search input via `x-model`
- `filterCapability string` — bound to select dropdown via `x-model`
- `scannedAt string` — last scan timestamp display

Computed:
- `filteredApps` — filters `apps` by name match (case-insensitive) and capability type
- `capabilityTypes` — deduplicated sorted list of all capability strings from `apps`

Methods:
- `rescan()` — sets `scanning = true`, triggers HTMX POST to `/service/web/homelab/rescan`

Layout:
```
┌─────────────────────────────────────────────────┐
│ [Search...]  [All Capabilities ▾]  [Rescan ↻]  │
├─────────────────────────────────────────────────┤
│ ┌─────────┐ ┌─────────┐ ┌─────────┐            │
│ │ Gitea   │ │ Karakeep│ │ Miniflux│  ...       │
│ │ online  │ │ running │ │ online  │            │
│ └─────────┘ └─────────┘ └─────────┘            │
└─────────────────────────────────────────────────┘
```

### Component: App card (`homelab_card.templ`)

```
┌──────────────────────┐
│ [G]  Gitea           │  ← Avatar circle (colored) + Name
│ https://git.local    │  ← First capability endpoint (link)
│ [online] [forge]     │  ← Status badge + Capability badges
└──────────────────────┘
```

Avatar color derived from app name (hash to pick from a fixed palette).

### Page: App detail (`homelab_detail.templ`)

Server-rendered, no Alpine.js:

```
┌─────────────────────────────────────────────────┐
│ ← Back to Registry                              │
│                                                 │
│ [G] Gitea                          [online]     │
│ Path: /data/apps/gitea                          │
│ Compose: docker-compose.yaml                    │
│ Version: 1.22.3                                 │
│ Last Discovered: 2026-06-03 14:30               │
│                                                 │
│ ── Health ──────────────────────────────────────│
│ Health: healthy                                 │
│                                                 │
│ ── Services ────────────────────────────────────│
│ Service │ Image            │ Container│ Ports   │
│ gitea   │ gitea/gitea:1.22 │ gitea    │ 3000:300│
│ db      │ postgres:16      │ gitea-db │         │
│                                                 │
│ ── Exposed Endpoints ────────────────────────── │
│ Capability │ Base URL        │ Auth │ Health    │
│ forge      │ http://git:3000 │ none │ /api/v1/h │
└─────────────────────────────────────────────────┘
```

## Backend: Webservice Handler

New file `internal/modules/web/homelab_webservice.go`:

```go
var homelabWebserviceRules = []webservice.Rule{
    webservice.Get("/homelab", homelabPage, route.WithNotAuth()),
    webservice.Get("/homelab/:name", homelabDetailPage, route.WithNotAuth()),
    webservice.Post("/homelab/rescan", homelabRescan, route.WithNotAuth()),
}
```

### `homelabPage`
- Auth check via `authenticateWeb(c)`
- Get apps from `homelab.DefaultRegistry.List()`
- Load scannedAt from `loadUpdatedAts()`
- Render `pages.HomelabPage(apps, scannedAt)`

### `homelabDetailPage`
- Auth check
- Look up app by `c.Params("name")` from `homelab.DefaultRegistry.Get()`
- 404 if not found
- Resolve live status from `homelab.DefaultRuntime.Status()`
- Extract version from `ComposeService.Image` (parse tag after colon)
- Render `pages.HomelabDetailPage(app, status, version)`

### `homelabRescan`
- Auth check
- Call `server.RunHomelabScan()` (exported function extracted from `initHomelabRegistry`)
- Set `HX-Redirect` to `/service/web/homelab`
- Return 200

## Server: Extract Rescan Logic

In `internal/server/homelab.go`, add exported function:

```go
// RunHomelabScan executes a full scan + probe + registry update cycle.
// Exported for use by the homelab web handler to support manual rescan.
func RunHomelabScan(cfg config.Homelab) error {
    // identical logic to initHomelabRegistry, minus DefaultRuntime setup
}
```

`initHomelabRegistry` calls `RunHomelabScan` internally (DRY).

The config object is available at runtime via the global config. The handler reads `config.App.Homelab` and passes it to `RunHomelabScan`.

## Version Parsing

Utility in `pkg/homelab/`:

```go
// ParseImageVersion extracts the version tag from a Docker image reference.
// "gitea/gitea:1.22.3" → "1.22.3"
// "postgres:16-alpine" → "16-alpine"
// "nginx@sha256:abc123" → "sha256:abc123" (digest references)
// "nginx" → "" (no tag)
func ParseImageVersion(image string) string {
    if idx := strings.LastIndex(image, ":"); idx != -1 {
        return image[idx+1:]
    }
    return ""
}
```

The version for an app is the tag of the first service whose image has a tag. If no services have tagged images, version is left empty on the detail page.

## Avatar Colors

App card avatar color derived by hashing the app name and picking from a fixed palette of 8 DaisyUI-compatible background colors: `bg-primary`, `bg-secondary`, `bg-accent`, `bg-info`, `bg-success`, `bg-warning`, `bg-error`, `bg-neutral`. The first letter of the app name is rendered in white text on the colored circle.

## Testing

### Unit tests (TDD)

| Test file | Coverage |
|-----------|----------|
| `pkg/homelab/version_test.go` | `ParseImageVersion` — happy path, no tag, edge cases |
| `pkg/homelab/registry_test.go` | Existing tests remain; no changes to Registry |
| `internal/modules/web/homelab_webservice_test.go` | Handler table tests: page renders, detail 404, rescan redirect |

### BDD acceptance tests (Ginkgo)

| Spec | Coverage |
|------|----------|
| `tests/specs/homelab_registry_spec_test.go` | Full page loads, card grid renders, search filters cards, capability filter works, detail page shows endpoints, rescan button present |

## Files Changed Summary

| File | Action | Description |
|------|--------|-------------|
| `pkg/views/pages/homelab.templ` | New | Card list page with Alpine.js controller |
| `pkg/views/pages/homelab_detail.templ` | New | Server-rendered app detail page |
| `pkg/views/partials/homelab_card.templ` | New | Single app card component |
| `pkg/views/partials/homelab_grid.templ` | New | Card grid with search, filter, empty state |
| `pkg/views/layout/base.templ` | Modify | Add "Registry" navbar link |
| `internal/modules/web/homelab_webservice.go` | New | Route definitions and handlers |
| `internal/modules/web/module.go` | Modify | Register `homelabWebserviceRules` |
| `internal/server/homelab.go` | Modify | Extract `RunHomelabScan()`, add `ParseImageVersion()` to `pkg/homelab/` |
| `pkg/homelab/version.go` | New | `ParseImageVersion()` utility |

# Hub Capability Browser

## Overview

Add an HTML page to the web UI that displays all registered capabilities as a filterable card grid. Each card shows capability metadata and expands inline to reveal operation details.

Data source: `hub.Default.List() returns []hub.Descriptor`

## Files

### New

| File | Purpose |
|---|---|
| `pkg/views/pages/capabilities.templ` | Full page: filter bar + grid container |
| `pkg/views/partials/capability_grid.templ` | Grid partial for HTMX filter swap |
| `pkg/views/partials/capability_card.templ` | Single card with Alpine.js expand/collapse |

### Modified

| File | Change |
|---|---|
| `internal/modules/web/hub_webservice.go` | Add two route handlers: full page (`GET /capabilities`) and HTMX partial (`GET /capabilities/grid`) |
| `pkg/views/layout/base.templ` | Add nav link: `<a href="/service/web/capabilities" data-testid="nav-capabilities" class="btn btn-ghost btn-sm">Capabilities</a>` after the Apps link |
| `pkg/views/pages/hub_apps.templ` | Add a "Capabilities" link/button near the page title, linking to `/service/web/capabilities` |

## Routes

```
GET  /service/web/capabilities          → hubCapabilitiesPage
GET  /service/web/capabilities/grid      → hubCapabilitiesGrid
```

Both use `authenticateWeb` and `route.WithNotAuth()` (same as existing hub web routes).

## Navigation

- **Top nav**: Add "Capabilities" link in `layout.Base` after the "Apps" link, with `data-testid="nav-capabilities"`
- **Hub Apps page**: Add a "Capabilities" link/button near the page title (`hub_apps.templ`), linking to `/service/web/capabilities`

## Data Flow

```
Browser GET /service/web/capabilities
  → handler calls hub.Default.List() — returns []Descriptor
  → pages.CapabilitiesPage(descriptors, types, providers)
  → renders full page with all cards

User changes Type or Provider filter
  → HTMX GET /service/web/capabilities/grid?type=X&provider=Y
  → handler filters List() in memory by Type and/or Backend
  → returns partials.CapabilityGrid(filtered)
  → HTMX swaps #capability-grid

User clicks card header
  → Alpine.js toggles x-show on card body
  → operation list renders inline (already in DOM, just hidden)
```

## Filters

- Two `<select>` dropdowns: Capability Type and Provider
- "All" as default `<option>` in both
- Provider list is derived at render time from unique `Descriptor.Backend` values
- Type list is derived from unique `Descriptor.Type` values
- Changing either triggers HTMX GET on the grid partial
- Filters combine (Type AND Provider)

## Card Design

### Collapsed

```
Type badge  Backend name    [Healthy? green/yellow dot]
App: xxx
Description text (truncated)
N Operations ▼
```

### Expanded (adds below)

```
Operations list:
  Op name | Description | Input params | Output params | Scopes
```

Each operation shown as a row with param names, types, required flag.

## States

| State | Behavior |
|---|---|
| Normal | Grid renders all cards |
| Empty registry | `partials.EmptyState` with "No capabilities registered" |
| Filter no results | `partials.EmptyState` with "No capabilities match these filters" |
| Status healthy=false | Yellow warning badge on the card |
| No description | Omit the line (no empty placeholder) |
| No operations | Show "No operations" in expanded body |
| No scopes | Omit the scopes column |

## Dependencies

- `hub.Default` (already exists)
- `partials.EmptyState` (already exists, accepts a message string)
- `layout.Base` (already exists)
- Alpine.js (already used in pipeline editor, available globally)
- HTMX (already used throughout the web module)
- DaisyUI v5 + Tailwind CSS v4 (already in use)

## No Changes To

- `pkg/hub/` — Descriptor type already has all needed fields
- `pkg/types/` — no new types
- `internal/store/` — no DB queries
- Existing hub web routes — additive only
- JSON API `GET /hub/capabilities` — unchanged

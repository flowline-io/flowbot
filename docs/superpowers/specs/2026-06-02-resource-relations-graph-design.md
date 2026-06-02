# Resource Relations Graph

## Background

Pipeline execution automatically tracks cross-service resource relationships via `ResourceLink`
records. For example, a `sync-issues` pipeline creates chains like:

```
github.get_issue  →  forge.create_issue      (GitHub → Gitea mirror)
forge.get_issue   →  kanban.create_task       (Issue → kanban task)
kanban.create_task →  notify.send             (Task → notification)
```

These relationships are stored in the `resource_links` table but have no visualization.

## Goal

Add a page `/service/web/relations` that renders an interactive, explorable tree view of
cross-service data lineage using only HTML, HTMX, and Alpine.js — no D3.js, SVG, or Canvas.

## Design

### Data Model

A resource node is uniquely identified by the triple `(app, capability, entity_id)`, all
already present in `ResourceLink`. The display label is formed by concatenating
`{capability} → #{entity_id}` (e.g., `issue#42`).

### API

Single endpoint, two modes:

```
GET /service/web/relations?node=app|cap|eid&pipeline=&since=
GET /service/web/relations?q=keyword
```

**Expand mode** (`?node=`):
Returns the node plus its upstream and downstream edges.

```json
{
  "node": {"app": "github", "capability": "issue", "entity_id": "42", "label": "github → issue#42"},
  "upstream": [
    {"edge_id": "src_evt|tgt_evt", "source": {...}, "target": {...}, "pipeline": "sync-issues", "created_at": "..."}
  ],
  "downstream": [
    {"edge_id": "src_evt|tgt_evt", "source": {...}, "target": {...}, "pipeline": "notify", "created_at": "..."}
  ]
}
```

Optional query params:
- `pipeline=name` — filter edges by pipeline name
- `since=Nd` — only edges created in last N days (7d, 30d, 90d)

**Search mode** (`?q=`):
Returns a flat list of distinct `(app, capability, entity_id)` tuples matching the keyword
against `source_entity_id` or `target_entity_id`. Limited to 20 results. Pagination via
`cursor` + `limit` query params.

### Page Layout

Three zones, pure HTML:

```
┌──────────────────────────────────────────────┐
│  [/service/web/relations]                    │
│  ┌──────────────────────────────────────────┐│
│  │  Search input      [pipeline ▼] [since ▼]││  ← search + filters bar
│  └──────────────────────────────────────────┘│
│  ┌─────────────────┬────────────────────────┐│
│  │  Breadcrumbs     │                        ││
│  │  ┌─────────────┐ │  Detail Panel          ││
│  │  │ Root Node   │ │  (metadata card)       ││
│  │  │ ├ Upstream  │ │                        ││
│  │  │ │  [badge]  │ │  Node: app, capability,││
│  │  │ │  [node]   │ │  entity_id, created_at ││
│  │  │ ├ Downstream│ │                        ││
│  │  │ │  [badge]  │ │  Edge: pipeline,       ││
│  │  │ │  [node]   │ │  source→target, time   ││
│  │  └─────────────┘ │                        ││
│  └─────────────────┴────────────────────────┘│
└──────────────────────────────────────────────┘
```

- **Left column**: Tree panel with accordion cards (DaisyUI `collapse` class).
- **Right column**: Detail panel showing metadata for the selected node or edge.

### Interaction Flow

1. User searches by keyword → HTMX replaces search results dropdown below input.
2. User clicks a search result → HTMX loads tree panel with that node as root.
3. Node card shows its upstream and downstream children (depth 1). Each child is a collapsible card.
4. User clicks a child node → that child becomes the new root (URL updates, HTMX reloads tree).
5. User clicks a node or edge badge → HTMX loads detail in right panel.
6. Breadcrumb row shows navigation path. Each segment is a clickable link back to that node.

All interactions are HTMX partial swaps. No page reloads.

### Templates

New files in `pkg/views/`:

| Template | Purpose |
|----------|---------|
| `relations.templ` | Full page: search bar + tree column + detail column |
| `relation_tree.templ` | Accordion tree — renders root node + upstream/downstream children |
| `relation_node.templ` | Single node card (collapsible `collapse`) |
| `relation_edge.templ` | Badge between parent and child nodes showing pipeline name |
| `relation_detail.templ` | Right panel: metadata for selected node or edge |
| `relation_search_results.templ` | Search result list |

Empty states rendered via existing `empty_state.templ`.

### Filters

| Filter | Source | Behavior |
|--------|--------|----------|
| Pipeline name | `SELECT DISTINCT pipeline_name FROM resource_links` | Filters edges on expand |
| Time range | Last 7d / 30d / 90d / all | `created_at >= now() - interval` |

All filter values are reflected in the URL query string, enabling:
- Browser back/forward navigation
- Shareable links

### Backend Changes

**Store** (`internal/store/store.go`, `ResourceChainStore`):

- `FindNodeRelations(ctx, app, capability, entityID, pipeline, since)` — returns upstream + downstream
  edges for a node. Reuses the existing `FindRelations` pattern with added `pipeline` and `since`
  filters.
- `SearchNodes(ctx, query, limit, cursor)` — returns distinct `(app, capability, entity_id)` tuples
  where entity_id contains the query string. Uses `DISTINCT` over union of source/target columns.

**Ability** (`pkg/ability/`):

Two operations on a resource-relations capability:
- `OpRelationsGet` — calls `FindNodeRelations`
- `OpRelationsSearch` — calls `SearchNodes`

**Web module** (`internal/modules/web/`):

- New file `relations.go` with handler functions
- New webservice rules mounted in `Webservice()`

**Files touched:**

| File | Change |
|------|--------|
| `internal/store/store.go` | +2 methods on `ResourceChainStore` |
| `pkg/ability/` | +1 capability descriptor with 2 operations |
| `internal/modules/web/module.go` | +1 route mount line |
| `internal/modules/web/relations.go` | New: handler functions |
| `pkg/views/relations.templ` | New: page template |
| `pkg/views/relation_tree.templ` | New: tree partial |
| `pkg/views/relation_node.templ` | New: node card partial |
| `pkg/views/relation_edge.templ` | New: edge badge partial |
| `pkg/views/relation_detail.templ` | New: detail panel partial |
| `pkg/views/relation_search_results.templ` | New: search results partial |

No changes to `pkg/pipeline/`, `pkg/module/`, existing routes, or dependencies.

### Testing

**TDD unit tests** (`internal/modules/web/relations_test.go`):

- Page render returns 200 with search bar and empty state
- Search returns matching nodes
- Search with no results returns empty state
- Search with empty query returns 400
- Expand node returns tree with upstream/downstream edges
- Expand non-existent node returns 404
- Pipeline filter returns only matching edges
- Since filter returns only edges within time range
- Node detail returns node metadata card
- Edge detail returns edge metadata card

All tests use the table-driven `for _, tt := range tests { t.Run(tt.name, ...) }` pattern.

**BDD specs** (`internal/modules/web/relations_suite_test.go`):

- Page loads with search input and empty state
- Search finds and displays matching nodes
- Search finds nothing shows empty state
- Expand node shows upstream/downstream edges
- Click child makes it the new root
- Pipeline filter restricts edges
- Time filter restricts edges
- URL state survives page refresh
- Detail panel shows node metadata
- Detail panel shows edge metadata

Seeded with 5-10 `ResourceLink` rows covering at least 3 capabilities and 2 pipelines.

## Out of Scope

- Recursive depth > 1 on a single request (user clicks to recurse one level at a time)
- Capability-type filter dropdown (can add later)
- Graph-level aggregation / matrix view
- Export / screenshot
- Real-time updates (WebSocket/SSE)
- Search result pagination beyond limit=20
- Clicking edge badge to navigate to Pipeline run history page
- Node color coding beyond DaisyUI badge classes

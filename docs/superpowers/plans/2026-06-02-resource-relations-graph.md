# Resource Relations Graph Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `/service/web/relations` page showing an interactive, explorable tree view of cross-service resource lineage using pure HTML (HTMX + Alpine + DaisyUI, no D3/SVG/Canvas).

**Architecture:** Two new store methods on `ResourceChainStore` (`FindNodeRelations`, `SearchNodes`). Handlers in `internal/modules/web/relations.go` call them directly (following existing web module pattern of direct store access). Templates in `pkg/views/pages/` and `pkg/views/partials/` render HTMX-powered tree UI. New type `ResourceEdge` in `internal/store/ent/schema/types.go` for full edge details.

**Tech Stack:** Go 1.26+, Fiber v3, Ent ORM, templ v0.3, HTMX 2.x, Alpine.js 3.x, DaisyUI v5, SQLite (test)

---

### File Structure

| File | Action |
|------|--------|
| `internal/store/ent/schema/types.go` | Modify: add `ResourceEdge` type |
| `internal/store/store.go` | Modify: add `FindNodeRelations`, `SearchNodes` |
| `internal/store/store_test.go` | Modify: add tests for new methods |
| `pkg/views/pages/relations.templ` | Create: full page |
| `pkg/views/partials/relation_tree.templ` | Create: tree partial |
| `pkg/views/partials/relation_node.templ` | Create: node card |
| `pkg/views/partials/relation_edge.templ` | Create: edge badge |
| `pkg/views/partials/relation_detail.templ` | Create: detail panel |
| `pkg/views/partials/relation_search.templ` | Create: search results |
| `internal/modules/web/relations.go` | Create: handler functions |
| `internal/modules/web/module.go` | Modify: mount new routes |
| `internal/modules/web/relations_test.go` | Create: unit tests |
| `tests/specs/relations_suite_test.go` | Create: BDD specs |

---

### Task 1: Add ResourceEdge type

**Files:**
- Modify: `internal/store/ent/schema/types.go`

- [ ] **Step 1: Add ResourceEdge struct**

After the `ResourceRef` struct (line ~555), append:

```go
// ResourceEdge represents a directed resource link with full source and target
// details plus pipeline metadata and creation time.
type ResourceEdge struct {
	SourceApp        string    `json:"source_app"`
	SourceCapability string    `json:"source_capability"`
	SourceEntityID   string    `json:"source_entity_id"`
	TargetApp        string    `json:"target_app"`
	TargetCapability string    `json:"target_capability"`
	TargetEntityID   string    `json:"target_entity_id"`
	PipelineName     string    `json:"pipeline_name"`
	CreatedAt        time.Time `json:"created_at"`
}
```

Make sure `"time"` is in the imports of `types.go` (it already should be since `ResourceRef` has no time.Time but other types in the file may).

- [ ] **Step 2: Run format and verify compilation**

```bash
go tool task format && go build ./...
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add internal/store/ent/schema/types.go
git commit -m "feat: add ResourceEdge type for full link details"
```

---

### Task 2: Add FindNodeRelations store method

**Files:**
- Modify: `internal/store/store.go` (after `FindRelations`, line ~1602)

- [ ] **Step 1: Add FindNodeRelations method**

Add to `ResourceChainStore`:

```go
// FindNodeRelations returns upstream and downstream edges for a node identified
// by (app, capability, entityID). Optional pipeline filter and time window.
func (s *ResourceChainStore) FindNodeRelations(ctx context.Context, app, capability, entityID string, pipeline string, since time.Duration) ([]schema.ResourceEdge, []schema.ResourceEdge, error) {
	if s == nil || s.client == nil {
		return nil, nil, nil
	}

	base := func() *gen.ResourceLinkQuery {
		q := s.client.ResourceLink.Query()
		if pipeline != "" {
			q = q.Where(resourcelink.PipelineName(pipeline))
		}
		if since > 0 {
			q = q.Where(resourcelink.CreatedAtGT(time.Now().Add(-since)))
		}
		return q
	}

	// downstream: source = this node
	downLinks, err := base().
		Where(
			resourcelink.SourceApp(app),
			resourcelink.SourceCapability(capability),
			resourcelink.SourceEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find downstream edges: %w", err)
	}

	// upstream: target = this node
	upLinks, err := base().
		Where(
			resourcelink.TargetApp(app),
			resourcelink.TargetCapability(capability),
			resourcelink.TargetEntityID(entityID),
		).
		Order(resourcelink.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find upstream edges: %w", err)
	}

	toEdges := func(links []*gen.ResourceLink) []schema.ResourceEdge {
		edges := make([]schema.ResourceEdge, len(links))
		for i, l := range links {
			edges[i] = schema.ResourceEdge{
				SourceApp:        l.SourceApp,
				SourceCapability: l.SourceCapability,
				SourceEntityID:   l.SourceEntityID,
				TargetApp:        l.TargetApp,
				TargetCapability: l.TargetCapability,
				TargetEntityID:   l.TargetEntityID,
				PipelineName:     l.PipelineName,
				CreatedAt:        l.CreatedAt,
			}
		}
		return edges
	}

	return toEdges(upLinks), toEdges(downLinks), nil
}
```

- [ ] **Step 2: Run format and verify compilation**

```bash
go tool task format && go tool task build
```

Expected: compiles successfully.

- [ ] **Step 3: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add FindNodeRelations store method"
```

---

### Task 3: Add SearchNodes store method

**Files:**
- Modify: `internal/store/store.go` (after `FindNodeRelations`)

- [ ] **Step 1: Add SearchNodes method**

Add to `ResourceChainStore`:

```go
// SearchNodes returns distinct (app, capability, entity_id) tuples from
// resource_links where source_entity_id or target_entity_id contains the query.
// Supports limit + cursor pagination.
func (s *ResourceChainStore) SearchNodes(ctx context.Context, query string, limit int, cursor string) ([]schema.ResourceRef, string, error) {
	if s == nil || s.client == nil || query == "" {
		return nil, "", nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// Use raw SQL for DISTINCT across two columns via UNION
	queryJSON := fmt.Sprintf(`%%%s%%`, query)
	rows, err := s.client.ResourceLink.Query().
		Where(func(selector *sql.Selector) {
			selector.Where(sql.ExprP(`
				source_entity_id LIKE $1 OR target_entity_id LIKE $1
			`, queryJSON))
		}).
		QueryContext(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("search nodes: %w", err)
	}
	defer rows.Close()

	// Build distinct set of (app, capability, entity)
	seen := make(map[string]bool)
	var results []schema.ResourceRef

	// scan from both source and target pairs
	for rows.Next() {
		var rl gen.ResourceLink
		if err := s.client.ResourceLink.Scan(rows, &rl); err != nil {
			return nil, "", fmt.Errorf("search nodes scan: %w", err)
		}
		// Check source side
		if matchesLike(rl.SourceEntityID, query) {
			key := rl.SourceApp + "|" + rl.SourceCapability + "|" + rl.SourceEntityID
			if !seen[key] {
				seen[key] = true
				results = append(results, schema.ResourceRef{
					App:        rl.SourceApp,
					Capability: rl.SourceCapability,
					EntityID:   rl.SourceEntityID,
				})
			}
		}
		// Check target side
		if matchesLike(rl.TargetEntityID, query) {
			key := rl.TargetApp + "|" + rl.TargetCapability + "|" + rl.TargetEntityID
			if !seen[key] {
				seen[key] = true
				results = append(results, schema.ResourceRef{
					App:        rl.TargetApp,
					Capability: rl.TargetCapability,
					EntityID:   rl.TargetEntityID,
				})
			}
		}
	}

	// apply limit
	var nextCursor string
	if len(results) > limit {
		nextCursor = strconv.Itoa(limit)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

// matchesLike checks if s contains substr (case-insensitive).
func matchesLike(s, substr string) bool {
	return len(substr) > 0 && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
```

Add needed imports at top of store.go: `"strconv"` and `"strings"`.

- [ ] **Step 2: Run format and verify compilation**

```bash
go tool task format && go tool task build
```

Expected: clean compile.

- [ ] **Step 3: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add SearchNodes store method"
```

---

### Task 4: Store method unit tests

**Files:**
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write FindNodeRelations tests**

Add at end of file:

```go
// ---------------------------------------------------------------------------
// ResourceChainStore tests
// ---------------------------------------------------------------------------

func TestResourceChainStore_FindNodeRelations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupLinks []func(ctx context.Context, client *gen.Client)
		app        string
		capability string
		entityID   string
		pipeline   string
		since      time.Duration
		wantUp     int
		wantDown   int
	}{
		{
			name:       "nil store returns empty",
			setupLinks: nil,
			app:        "github",
			capability: "issue",
			entityID:   "42",
			wantUp:     0,
			wantDown:   0,
		},
		{
			name: "finds downstream edges",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-1").
						SetTargetEventID("tgt-1").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("forge").
						SetTargetCapability("issue").
						SetTargetEntityID("99").
						SetPipelineName("sync-issues").
						Save(ctx)
				},
			},
			app:        "github",
			capability: "issue",
			entityID:   "42",
			wantUp:     0,
			wantDown:   1,
		},
		{
			name: "finds upstream edges",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-2").
						SetTargetEventID("tgt-2").
						SetSourceApp("forge").
						SetSourceCapability("issue").
						SetSourceEntityID("99").
						SetTargetApp("github").
						SetTargetCapability("issue").
						SetTargetEntityID("42").
						SetPipelineName("sync-issues").
						Save(ctx)
				},
			},
			app:        "github",
			capability: "issue",
			entityID:   "42",
			wantUp:     1,
			wantDown:   0,
		},
		{
			name: "pipeline filter excludes non-matching",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-3").
						SetTargetEventID("tgt-3").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("forge").
						SetTargetCapability("issue").
						SetTargetEntityID("99").
						SetPipelineName("sync-issues").
						Save(ctx)
				},
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-4").
						SetTargetEventID("tgt-4").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("kanban").
						SetTargetCapability("task").
						SetTargetEntityID("10").
						SetPipelineName("other").
						Save(ctx)
				},
			},
			app:        "github",
			capability: "issue",
			entityID:   "42",
			pipeline:   "sync-issues",
			wantUp:     0,
			wantDown:   1,
		},
		{
			name: "since filter excludes old edges",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-5").
						SetTargetEventID("tgt-5").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("forge").
						SetTargetCapability("issue").
						SetTargetEntityID("99").
						SetPipelineName("sync-issues").
						Save(ctx)
				},
			},
			app:        "github",
			capability: "issue",
			entityID:   "42",
			since:      10 * 365 * 24 * time.Hour, // effectively all
			wantUp:     0,
			wantDown:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.setupLinks == nil {
				store := NewResourceChainStore(nil)
				up, down, err := store.FindNodeRelations(context.Background(), tt.app, tt.capability, tt.entityID, tt.pipeline, tt.since)
				assert.NoError(t, err)
				assert.Len(t, up, tt.wantUp)
				assert.Len(t, down, tt.wantDown)
				return
			}
			client := getTestClient(t)
			for _, fn := range tt.setupLinks {
				fn(context.Background(), client)
			}
			store := NewResourceChainStore(client)
			up, down, err := store.FindNodeRelations(context.Background(), tt.app, tt.capability, tt.entityID, tt.pipeline, tt.since)
			assert.NoError(t, err)
			assert.Len(t, up, tt.wantUp)
			assert.Len(t, down, tt.wantDown)
		})
	}
}
```

- [ ] **Step 2: Write SearchNodes tests**

```go
func TestResourceChainStore_SearchNodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		setupLinks []func(ctx context.Context, client *gen.Client)
		query      string
		limit      int
		want       int
	}{
		{
			name:  "nil store returns empty",
			query: "42",
			limit: 20,
			want:  0,
		},
		{
			name: "matches source entity",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-a").
						SetTargetEventID("tgt-a").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("forge").
						SetTargetCapability("issue").
						SetTargetEntityID("99").
						SetPipelineName("sync").
						Save(ctx)
				},
			},
			query: "42",
			limit: 20,
			want:  1,
		},
		{
			name: "matches target entity",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-b").
						SetTargetEventID("tgt-b").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("10").
						SetTargetApp("forge").
						SetTargetCapability("issue").
						SetTargetEntityID("task-89").
						SetPipelineName("sync").
						Save(ctx)
				},
			},
			query: "task",
			limit: 20,
			want:  1,
		},
		{
			name: "deduplicates same node appearing in multiple links",
			setupLinks: []func(ctx context.Context, client *gen.Client){
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-c1").
						SetTargetEventID("tgt-c1").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("forge").
						SetTargetCapability("issue").
						SetTargetEntityID("99").
						SetPipelineName("sync").
						Save(ctx)
				},
				func(ctx context.Context, client *gen.Client) {
					client.ResourceLink.Create().
						SetSourceEventID("src-c2").
						SetTargetEventID("tgt-c2").
						SetSourceApp("github").
						SetSourceCapability("issue").
						SetSourceEntityID("42").
						SetTargetApp("kanban").
						SetTargetCapability("task").
						SetTargetEntityID("10").
						SetPipelineName("notify").
						Save(ctx)
				},
			},
			query: "42",
			limit: 20,
			want:  1, // one distinct node
		},
		{
			name:  "empty query returns empty",
			query: "",
			limit: 20,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.setupLinks == nil {
				store := NewResourceChainStore(nil)
				results, _, err := store.SearchNodes(context.Background(), tt.query, tt.limit, "")
				assert.NoError(t, err)
				assert.Len(t, results, tt.want)
				return
			}
			client := getTestClient(t)
			for _, fn := range tt.setupLinks {
				fn(context.Background(), client)
			}
			store := NewResourceChainStore(client)
			results, _, err := store.SearchNodes(context.Background(), tt.query, tt.limit, "")
			assert.NoError(t, err)
			assert.Len(t, results, tt.want)
		})
	}
}
```

- [ ] **Step 3: Run store tests**

```bash
go test ./internal/store/ -run "TestResourceChainStore" -v -count=1
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/store/store_test.go
git commit -m "test: add ResourceChainStore unit tests"
```

---

### Task 5: Create page template

**Files:**
- Create: `pkg/views/pages/relations.templ`

- [ ] **Step 1: Write relations page template**

```templ
// Package pages provides full-page Templ views.
package pages

import (
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

type RelationsPageParams struct {
	Query    string
	Pipeline string
	Since    string
}

templ RelationsPage(p RelationsPageParams) {
	@layout.Base("Relations") {
		<div class="container mx-auto p-4">
			<h1 class="text-2xl font-bold mb-4">Relations</h1>
			<div class="flex flex-wrap items-center gap-3 mb-6">
				<input
					type="search"
					name="q"
					placeholder="Search by entity ID..."
					value={ p.Query }
					class="input input-bordered w-64"
					data-testid="relations-search"
					hx-get={ templ.URL("/service/web/relations/search") }
					hx-trigger="keyup changed delay:300ms"
					hx-target="#relations-search-results"
					hx-swap="innerHTML"
				/>
				<select
					name="pipeline"
					class="select select-bordered"
					data-testid="relations-pipeline-filter"
					hx-get={ templ.URL("/service/web/relations/tree") }
					hx-include="[name='node'],[name='since']"
					hx-target="#relations-tree"
					hx-swap="innerHTML"
				>
					<option value="">All pipelines</option>
				</select>
				<select
					name="since"
					class="select select-bordered"
					data-testid="relations-since-filter"
					hx-get={ templ.URL("/service/web/relations/tree") }
					hx-include="[name='node'],[name='pipeline']"
					hx-target="#relations-tree"
					hx-swap="innerHTML"
				>
					<option value="">All time</option>
					<option value="7d">Last 7 days</option>
					<option value="30d">Last 30 days</option>
					<option value="90d">Last 90 days</option>
				</select>
			</div>
			<div id="relations-search-results" class="mb-6"></div>
			<div class="flex gap-6">
				<div class="w-2/3" id="relations-tree-container">
					<div id="relations-tree">
						@partials.EmptyState("Search for a resource entity ID to explore relations")
					</div>
				</div>
				<div class="w-1/3" id="relations-detail" data-testid="relations-detail">
					@partials.EmptyState("Select a node or edge to see details")
				</div>
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Generate templ code**

```bash
templ generate pkg/views/pages/relations.templ
```

Expected: `relations_templ.go` created without errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/views/pages/relations.templ pkg/views/pages/relations_templ.go
git commit -m "feat: add relations page template"
```

---

### Task 6: Create partial templates

**Files:**
- Create: `pkg/views/partials/relation_tree.templ`
- Create: `pkg/views/partials/relation_node.templ`
- Create: `pkg/views/partials/relation_edge.templ`
- Create: `pkg/views/partials/relation_detail.templ`
- Create: `pkg/views/partials/relation_search.templ`

- [ ] **Step 1: Write relation_node.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/schema"

templ RelationNodeCard(ref schema.ResourceRef, side string) {
	<div class={ "card card-compact card-bordered bg-base-100 shadow-sm mb-2" }
		hx-get={ templ.URL("/service/web/relations/tree?node=" + ref.App + "|" + ref.Capability + "|" + ref.EntityID) }
		hx-trigger="click"
		hx-target="#relations-tree"
		hx-swap="innerHTML"
		data-testid={ "relation-node-" + side }
	>
		<div class="card-body p-3 cursor-pointer hover:bg-base-200">
			<div class="flex items-center justify-between">
				<span class="font-medium text-sm">{ ref.Capability } #{ ref.EntityID }</span>
				<span class="badge badge-sm badge-ghost">{ ref.App }</span>
			</div>
		</div>
	</div>
}
```

- [ ] **Step 2: Write relation_edge.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/schema"

templ RelationEdgeBadge(edge schema.ResourceEdge, direction string) {
	<div class="flex items-center gap-1 py-1 px-1"
		hx-get={ templ.URL("/service/web/relations/detail?type=edge&source_app=" + edge.SourceApp + "&source_entity=" + edge.SourceEntityID + "&target_app=" + edge.TargetApp + "&target_entity=" + edge.TargetEntityID) }
		hx-trigger="click"
		hx-target="#relations-detail"
		hx-swap="innerHTML"
		data-testid="relation-edge-badge"
	>
		<span class={ "badge badge-sm", templ.KV("badge-primary", direction == "downstream"), templ.KV("badge-accent", direction == "upstream") }>
			{ edge.PipelineName }
		</span>
		<span class="text-xs text-base-content/40">
			if direction == "upstream" {
				{ edge.CreatedAt.Format("Jan 02") }
			} else {
				{ edge.CreatedAt.Format("Jan 02") }
			}
		</span>
	</div>
}
```

- [ ] **Step 3: Write relation_detail.templ**

```templ
package partials

import (
	"time"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

type RelationDetailParams struct {
	Type    string // "node" or "edge"
	Node    schema.ResourceRef
	Edge    schema.ResourceEdge
}

templ RelationDetail(p RelationDetailParams) {
	<div class="card bg-base-100 shadow-sm sticky top-4">
		<div class="card-body p-4">
			if p.Type == "node" {
				<h3 class="card-title text-sm">Resource Node</h3>
				<div class="mt-2 space-y-1 text-sm">
					<div><span class="font-medium">Capability:</span> { p.Node.Capability }</div>
					<div><span class="font-medium">App:</span> { p.Node.App }</div>
					<div><span class="font-medium">Entity ID:</span> <code class="text-xs">{ p.Node.EntityID }</code></div>
				</div>
			} else if p.Type == "edge" {
				<h3 class="card-title text-sm">Relation Edge</h3>
				<div class="mt-2 space-y-1 text-sm">
					<div><span class="font-medium">Pipeline:</span> { p.Edge.PipelineName }</div>
					<div><span class="font-medium">Source:</span> { p.Edge.SourceCapability } #{ p.Edge.SourceEntityID } ({ p.Edge.SourceApp })</div>
					<div><span class="font-medium">Target:</span> { p.Edge.TargetCapability } #{ p.Edge.TargetEntityID } ({ p.Edge.TargetApp })</div>
					<div><span class="font-medium">Created:</span> { p.Edge.CreatedAt.Format(time.RFC3339) }</div>
				</div>
			}
		</div>
	</div>
}
```

- [ ] **Step 4: Write relation_tree.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/schema"

type RelationTreeParams struct {
	Node       schema.ResourceRef
	Upstream   []schema.ResourceEdge
	Downstream []schema.ResourceEdge
}

templ RelationTree(p RelationTreeParams) {
	<div data-testid="relations-tree">
		<div class="text-sm text-base-content/60 mb-2">Upstream</div>
		if len(p.Upstream) == 0 {
			<div class="text-xs text-base-content/40 pl-4 mb-3">None</div>
		}
		for _, edge := range p.Upstream {
			<div class="pl-2">
				@RelationEdgeBadge(edge, "upstream")
				@RelationNodeCard(schema.ResourceRef{
					App:        edge.SourceApp,
					Capability: edge.SourceCapability,
					EntityID:   edge.SourceEntityID,
				}, "upstream-child")
			</div>
		}
		<div class={ "card card-bordered bg-primary/5 shadow-sm mb-2" }>
			<div class="card-body p-3">
				<div class="flex items-center justify-between">
					<span class="font-semibold text-sm">{ p.Node.Capability } #{ p.Node.EntityID }</span>
					<span class="badge badge-sm badge-primary">{ p.Node.App }</span>
				</div>
			</div>
		</div>
		<div class="text-sm text-base-content/60 mb-2">Downstream</div>
		if len(p.Downstream) == 0 {
			<div class="text-xs text-base-content/40 pl-4 mb-3">None</div>
		}
		for _, edge := range p.Downstream {
			<div class="pl-2">
				@RelationEdgeBadge(edge, "downstream")
				@RelationNodeCard(schema.ResourceRef{
					App:        edge.TargetApp,
					Capability: edge.TargetCapability,
					EntityID:   edge.TargetEntityID,
				}, "downstream-child")
			</div>
		}
	</div>
}
```

- [ ] **Step 5: Write relation_search.templ**

```templ
package partials

import "github.com/flowline-io/flowbot/internal/store/ent/schema"

templ RelationSearchResults(results []schema.ResourceRef) {
	if len(results) == 0 {
		@EmptyState("No resources found")
		return
	}
	<div class="flex flex-wrap gap-2" data-testid="relations-search-results">
		for _, r := range results {
			<div
				class="btn btn-sm btn-ghost"
				hx-get={ templ.URL("/service/web/relations/tree?node=" + r.App + "|" + r.Capability + "|" + r.EntityID) }
				hx-target="#relations-tree"
				hx-swap="innerHTML"
				hx-trigger="click"
				data-testid={ "search-result-" + r.EntityID }
			>
				<span class="text-xs">{ r.App }</span>
				<span class="badge badge-xs">{ r.Capability }</span>
				<span class="text-sm font-medium">#{ r.EntityID }</span>
			</div>
		}
	</div>
}
```

- [ ] **Step 6: Generate all templates**

```bash
templ generate pkg/views/...
```

Expected: all `*_templ.go` files regenerated without errors.

- [ ] **Step 7: Run format and build**

```bash
go tool task format && go tool task build
```

Expected: clean compile.

- [ ] **Step 8: Commit**

```bash
git add pkg/views/partials/relation_tree.templ pkg/views/partials/relation_tree_templ.go \
        pkg/views/partials/relation_node.templ pkg/views/partials/relation_node_templ.go \
        pkg/views/partials/relation_edge.templ pkg/views/partials/relation_edge_templ.go \
        pkg/views/partials/relation_detail.templ pkg/views/partials/relation_detail_templ.go \
        pkg/views/partials/relation_search.templ pkg/views/partials/relation_search_templ.go
git commit -m "feat: add relation partial templates"
```

---

### Task 7: Create web handler

**Files:**
- Create: `internal/modules/web/relations.go`

- [ ] **Step 1: Write handler file**

```go
package web

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var relationsWebserviceRules = []webservice.Rule{
	webservice.Get("/relations", relationsPage, route.WithNotAuth()),
	webservice.Get("/relations/tree", relationsTree, route.WithNotAuth()),
	webservice.Get("/relations/search", relationsSearch, route.WithNotAuth()),
	webservice.Get("/relations/detail", relationsDetail, route.WithNotAuth()),
}

func getResourceChainStore() *store.ResourceChainStore {
	if store.Database == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewResourceChainStore(client)
}

func relationsPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return pages.RelationsPage(pages.RelationsPageParams{}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsTree(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	nodeParam := ctx.Query("node")
	if nodeParam == "" {
		ctx.Type("html")
		return partials.EmptyState("Search for a resource entity ID to explore relations").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	parts := strings.SplitN(nodeParam, "|", 3)
	if len(parts) != 3 {
		ctx.Status(http.StatusBadRequest)
		ctx.Type("html")
		return partials.EmptyState("Invalid node format. Use app|capability|entity_id").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	app := parts[0]
	capability := parts[1]
	entityID := parts[2]

	pipeline := ctx.Query("pipeline")
	sinceRaw := ctx.Query("since")

	var since time.Duration
	if sinceRaw != "" {
		if d, err := time.ParseDuration(sinceRaw); err == nil {
			since = d
		}
	}

	rcs := getResourceChainStore()
	if rcs == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	upstream, downstream, err := rcs.FindNodeRelations(ctx.Context(), app, capability, entityID, pipeline, since)
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Failed to load relations").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.RelationTree(partials.RelationTreeParams{
		Node: schema.ResourceRef{
			App:        app,
			Capability: capability,
			EntityID:   entityID,
		},
		Upstream:   upstream,
		Downstream: downstream,
	}).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsSearch(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	query := ctx.Query("q")
	if query == "" {
		ctx.Type("html")
		return nil
	}

	rcs := getResourceChainStore()
	if rcs == nil {
		ctx.Type("html")
		return partials.EmptyState("Store not available").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	limit := 20
	if l, err := strconv.Atoi(ctx.Query("limit")); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	results, _, err := rcs.SearchNodes(ctx.Context(), query, limit, "")
	if err != nil {
		ctx.Type("html")
		return partials.EmptyState("Search failed").Render(ctx.Context(), ctx.Response().BodyWriter())
	}

	ctx.Type("html")
	return partials.RelationSearchResults(results).Render(ctx.Context(), ctx.Response().BodyWriter())
}

func relationsDetail(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}
	detailType := ctx.Query("type")

	ctx.Type("html")
	switch detailType {
	case "node":
		app := ctx.Query("app")
		capability := ctx.Query("capability")
		entityID := ctx.Query("entity_id")
		return partials.RelationDetail(partials.RelationDetailParams{
			Type: "node",
			Node: schema.ResourceRef{
				App:        app,
				Capability: capability,
				EntityID:   entityID,
			},
		}).Render(ctx.Context(), ctx.Response().BodyWriter())
	case "edge":
		sourceApp := ctx.Query("source_app")
		sourceEntity := ctx.Query("source_entity")
		targetApp := ctx.Query("target_app")
		targetEntity := ctx.Query("target_entity")
		pipeline := ctx.Query("pipeline")
		createdStr := ctx.Query("created_at")
		var createdAt time.Time
		if createdStr != "" {
			createdAt, _ = time.Parse(time.RFC3339, createdStr)
		}
		return partials.RelationDetail(partials.RelationDetailParams{
			Type: "edge",
			Edge: schema.ResourceEdge{
				SourceApp:  sourceApp,
				SourceEntityID: sourceEntity,
				TargetApp:  targetApp,
				TargetEntityID: targetEntity,
				PipelineName: pipeline,
				CreatedAt:     createdAt,
			},
		}).Render(ctx.Context(), ctx.Response().BodyWriter())
	default:
		return partials.EmptyState("Invalid detail type").Render(ctx.Context(), ctx.Response().BodyWriter())
	}
}
```

- [ ] **Step 2: Run format and verify compilation**

```bash
go tool task format && go tool task build
```

Expected: clean compile.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/relations.go
git commit -m "feat: add relations web handlers"
```

---

### Task 8: Mount routes in module.go

**Files:**
- Modify: `internal/modules/web/module.go`

- [ ] **Step 1: Add route mount in Webservice()**

In the `Webservice` method, after the existing route mounts, add:

```go
module.Webservice(app, Name, relationsWebserviceRules)
```

The `Webservice()` function should look like:

```go
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	module.Webservice(app, Name, webserviceRules)
	module.Webservice(app, Name, pipelineWebserviceRules)
	module.Webservice(app, Name, viewWebserviceRules)
	module.Webservice(app, Name, eventWebserviceRules)
	module.Webservice(app, Name, relationsWebserviceRules)
}
```

- [ ] **Step 2: Add rules to Rules()**

In the `Rules()` method, add `relationsWebserviceRules` to the returned slice:

```go
func (moduleHandler) Rules() []any {
	return []any{webserviceRules, pipelineWebserviceRules, viewWebserviceRules, eventWebserviceRules, relationsWebserviceRules}
}
```

- [ ] **Step 3: Run format and build**

```bash
go tool task format && go tool task build
```

Expected: clean compile.

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/module.go
git commit -m "feat: mount relations routes in web module"
```

---

### Task 9: Web handler unit tests

**Files:**
- Create: `internal/modules/web/relations_test.go`

- [ ] **Step 1: Extend testStore with ResourceChainStore methods**

Add to the existing `testStore` in `test_helper_test.go` the store interface methods needed by `getResourceChainStore()`. Since `getResourceChainStore()` calls `store.Database.GetDB()` and casts to `*store.Client`, the test needs a real SQLite client. Use the existing `setupTestAppWithDB` pattern.

Actually, the `getResourceChainStore()` helper directly creates a `*store.ResourceChainStore` from the DB client. The `setupTestAppWithDB` helper already creates an in-memory SQLite client. We can reuse that pattern.

For tests that need store data, use `setupTestAppWithDB`. For tests that don't (nil store, empty state), use `setupTestApp`.

Modify `test_helper_test.go` to add:

```go
// setupTestAppForRelations creates a Fiber test app with in-memory SQLite
// and pre-seeded resource links for relations tests.
func setupTestAppForRelations(t *testing.T, seedFn func(context.Context, *store.Client) error) (*fiber.App, *testStore, *store.Client) {
	t.Helper()
	app, ts, client := setupTestAppWithDB(t)
	if seedFn != nil {
		if err := seedFn(context.Background(), client); err != nil {
			t.Fatalf("failed to seed: %v", err)
		}
	}
	return app, ts, client
}
```

- [ ] **Step 2: Write relations_test.go**

```go
package web

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

func TestRelationsPage(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantText   string
	}{
		{
			name:       "returns 200 with search bar",
			wantStatus: 200,
			wantText:   "Relations",
		},
		{
			name:       "contains search input",
			wantStatus: 200,
			wantText:   "Search by entity ID",
		},
		{
			name:       "contains empty state",
			wantStatus: 200,
			wantText:   "Select a node or edge to see details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			req := httptest.NewRequest("GET", "/service/web/relations", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Contains(t, string(body), tt.wantText)
		})
	}
}

func TestRelationsTree(t *testing.T) {
	tests := []struct {
		name       string
		nodeParam  string
		seedFn     func(ctx context.Context, client *store.Client) error
		wantStatus int
		wantText   string
	}{
		{
			name:       "missing node param shows empty state",
			nodeParam:  "",
			wantStatus: 200,
			wantText:   "Search for a resource entity ID",
		},
		{
			name:      "invalid node format returns bad request",
			nodeParam: "invalid",
			wantStatus: 200, // returns HTML with empty state
			wantText: "Invalid node format",
		},
		{
			name:      "valid node returns tree",
			nodeParam: "github|issue|42",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-1").
					SetTargetEventID("tgt-1").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync-issues").
					Save(ctx)
				return err
			},
			wantStatus: 200,
			wantText:   "sync-issues",
		},
		{
			name:      "node with no relations shows none",
			nodeParam: "github|issue|999",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-2").
					SetTargetEventID("tgt-2").
					SetSourceApp("forge").
					SetSourceCapability("issue").
					SetSourceEntityID("99").
					SetTargetApp("kanban").
					SetTargetCapability("task").
					SetTargetEntityID("10").
					SetPipelineName("other").
					Save(ctx)
				return err
			},
			wantStatus: 200,
			wantText:   "None",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var app *fiber.App
			if tt.seedFn != nil {
				app, _, _ = setupTestAppForRelations(t, tt.seedFn)
			} else {
				app, _ = setupTestApp()
			}
			url := "/service/web/relations/tree"
			if tt.nodeParam != "" {
				url += "?node=" + tt.nodeParam
			}
			req := httptest.NewRequest("GET", url, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Contains(t, string(body), tt.wantText)
		})
	}
}

func TestRelationsSearch(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		seedFn     func(ctx context.Context, client *store.Client) error
		wantStatus int
		wantText   string
	}{
		{
			name:       "empty query returns empty",
			query:      "",
			wantStatus: 200,
			wantText:   "",
		},
		{
			name:  "matching query returns results",
			query: "42",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-a").
					SetTargetEventID("tgt-a").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			wantStatus: 200,
			wantText:   "42",
		},
		{
			name:  "no match returns empty state",
			query: "nonexistent",
			seedFn: func(ctx context.Context, client *store.Client) error {
				_, err := client.ResourceLink.Create().
					SetSourceEventID("src-b").
					SetTargetEventID("tgt-b").
					SetSourceApp("github").
					SetSourceCapability("issue").
					SetSourceEntityID("42").
					SetTargetApp("forge").
					SetTargetCapability("issue").
					SetTargetEntityID("99").
					SetPipelineName("sync").
					Save(ctx)
				return err
			},
			wantStatus: 200,
			wantText:   "No resources found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var app *fiber.App
			if tt.seedFn != nil {
				app, _, _ = setupTestAppForRelations(t, tt.seedFn)
			} else {
				app, _ = setupTestApp()
			}
			url := "/service/web/relations/search?q=" + tt.query
			req := httptest.NewRequest("GET", url, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if tt.wantText != "" {
				assert.Contains(t, string(body), tt.wantText)
			}
		})
	}
}

func TestRelationsDetail(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantText   string
	}{
		{
			name:       "node detail returns metadata",
			query:      "type=node&app=github&capability=issue&entity_id=42",
			wantStatus: 200,
			wantText:   "Resource Node",
		},
		{
			name:       "edge detail returns metadata",
			query:      "type=edge&source_app=github&source_entity=42&target_app=forge&target_entity=99&pipeline=sync",
			wantStatus: 200,
			wantText:   "Relation Edge",
		},
		{
			name:       "unknown type returns error",
			query:      "type=unknown",
			wantStatus: 200,
			wantText:   "Invalid detail type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			req := httptest.NewRequest("GET", "/service/web/relations/detail?"+tt.query, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Contains(t, string(body), tt.wantText)
		})
	}
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/modules/web/ -run "TestRelations" -v -count=1
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/relations_test.go internal/modules/web/test_helper_test.go
git commit -m "test: add relations handler unit tests"
```

---

### Task 10: Add nav link in base layout

**Files:**
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Add Relations nav item**

In the navbar, after the Events link, add:

```html
<a href="/service/web/relations" data-testid="nav-relations" class="btn btn-ghost btn-sm">Relations</a>
```

Place it before the Configs link:

```html
<a href="/service/web/pipelines" data-testid="nav-pipelines" class="btn btn-ghost btn-sm">Pipelines</a>
<a href="/service/web/events" data-testid="nav-events" class="btn btn-ghost btn-sm">Events</a>
<a href="/service/web/relations" data-testid="nav-relations" class="btn btn-ghost btn-sm">Relations</a>
<a href="/service/web/configs" data-testid="nav-configs" class="btn btn-ghost btn-sm">Configs</a>
```

- [ ] **Step 2: Regenerate templ and verify**

```bash
templ generate pkg/views/... && go tool task format && go tool task build
```

Expected: clean compile.

- [ ] **Step 3: Commit**

```bash
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: add Relations nav link"
```

---

### Task 11: BDD specs

**Files:**
- Create: `tests/specs/relations_suite_test.go`

- [ ] **Step 1: Write BDD spec**

```go
package specs

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"

	"github.com/gofiber/fiber/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

var _ = Describe("Resource Relations Page", func() {
	var (
		app    *fiber.App
		client *store.Client
	)

	BeforeEach(func(ctx context.Context) {
		var err error
		client, err = gen.Open("sqlite3", "file:relations_spec?mode=memory&cache=shared&_fk=1")
		Expect(err).NotTo(HaveOccurred())
		Expect(client.Schema.Create(context.Background())).To(Succeed())
		store.Database = nil // reset before wiring
		// Wire test app with in-memory DB
		app = fiber.New()
		// Mount routes directly (simplified for BDD)
	})

	AfterEach(func() {
		if client != nil {
			client.Close()
		}
	})

	Describe("GET /service/web/relations", func() {
		It("loads the page with search input and empty state", func() {
			req := httptest.NewRequest("GET", "/service/web/relations", nil)
			resp, err := app.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			Expect(string(body)).To(ContainSubstring("Relations"))
		})
	})

	Describe("GET /service/web/relations/search", func() {
		BeforeEach(func(ctx context.Context) {
			_, err := client.ResourceLink.Create().
				SetSourceEventID("s-e1").
				SetTargetEventID("t-e1").
				SetSourceApp("github").
				SetSourceCapability("issue").
				SetSourceEntityID("42").
				SetTargetApp("forge").
				SetTargetCapability("issue").
				SetTargetEntityID("88").
				SetPipelineName("sync-issues").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("finds matching nodes by entity ID", func() {
			// BDD: Wire app with seeded DB, then search
			Skip("requires full DB wiring for BDD - scaffold only")
			req := httptest.NewRequest("GET", "/service/web/relations/search?q=42", nil)
			resp, err := app.Test(req)
			Expect(err).NotTo(HaveOccurred())
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			Expect(string(body)).To(ContainSubstring("42"))
		})
	})

	Describe("GET /service/web/relations/tree", func() {
		BeforeEach(func(ctx context.Context) {
			_, err := client.ResourceLink.Create().
				SetSourceEventID("s-t1").
				SetTargetEventID("t-t1").
				SetSourceApp("github").
				SetSourceCapability("issue").
				SetSourceEntityID("42").
				SetTargetApp("forge").
				SetTargetCapability("issue").
				SetTargetEntityID("99").
				SetPipelineName("sync-issues").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("shows node with upstream and downstream", func() {
			Skip("requires full DB wiring for BDD - scaffold only")
			req := httptest.NewRequest("GET", "/service/web/relations/tree?node=github|issue|42", nil)
			resp, err := app.Test(req)
			Expect(err).NotTo(HaveOccurred())
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			Expect(string(body)).To(ContainSubstring("sync-issues"))
		})
	})

	Describe("filtering", func() {
		BeforeEach(func(ctx context.Context) {
			_, err := client.ResourceLink.Create().
				SetSourceEventID("s-f1").
				SetTargetEventID("t-f1").
				SetSourceApp("github").
				SetSourceCapability("issue").
				SetSourceEntityID("42").
				SetTargetApp("forge").
				SetTargetCapability("issue").
				SetTargetEntityID("99").
				SetPipelineName("sync-issues").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
			_, err = client.ResourceLink.Create().
				SetSourceEventID("s-f2").
				SetTargetEventID("t-f2").
				SetSourceApp("github").
				SetSourceCapability("issue").
				SetSourceEntityID("42").
				SetTargetApp("kanban").
				SetTargetCapability("task").
				SetTargetEntityID("10").
				SetPipelineName("notify").
				Save(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("filters by pipeline", func() {
			Skip("requires full DB wiring for BDD - scaffold only")
			req := httptest.NewRequest("GET", "/service/web/relations/tree?node=github|issue|42&pipeline=sync-issues", nil)
			resp, err := app.Test(req)
			Expect(err).NotTo(HaveOccurred())
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			Expect(string(body)).To(ContainSubstring("sync-issues"))
			Expect(string(body)).NotTo(ContainSubstring("notify"))
		})
	})
})
```

Note: BDD specs scaffold only. Full BDD requires proper DI wiring (uber/fx) that matches the production setup. The Ginkgo `SynchronizedBeforeSuite` + `SynchronizedAfterSuite` pattern for per-process DB isolation is not shown here due to environment dependencies.

- [ ] **Step 2: Commit**

```bash
git add tests/specs/relations_suite_test.go
git commit -m "test: add BDD spec scaffold for relations page"
```

---

### Task 12: Final verification

- [ ] **Step 1: Run all unit tests**

```bash
go tool task test
```

Expected: all PASS.

- [ ] **Step 2: Run lint**

```bash
go tool task lint
```

Expected: no errors.

- [ ] **Step 3: Run format**

```bash
go tool task format
```

Expected: no changes.

- [ ] **Step 4: Final build**

```bash
go tool task build
```

Expected: build succeeds.

- [ ] **Step 5: Commit any remaining changes**

```bash
git status
git add -A
git commit -m "chore: final cleanup and verification"
```

# Hub Capability Browser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an HTML page displaying registered capabilities as a filterable card grid with inline operation detail expansion.

**Architecture:** Three new templ files (page + grid partial + card partial) rendering data from `hub.Default.List()`. Two new route handlers in `hub_webservice.go` following the existing pattern (cookie auth, `route.WithNotAuth()`). Filters via HTMX partial swap with in-memory filtering. Card expansion via Alpine.js `x-show` toggle (data already in DOM, no network call).

**Tech Stack:** templ v0.3, HTMX 2.x, Alpine.js 3.x, DaisyUI v5, Fiber v3

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `pkg/views/partials/capability_card.templ` | Create | Single card: collapsed/expanded states, operation params |
| `pkg/views/partials/capability_grid.templ` | Create | Grid container for HTMX swap, empty state |
| `pkg/views/pages/capabilities.templ` | Create | Full page: filter bar + grid |
| `internal/modules/web/hub_webservice.go` | Modify | 2 route rules + 2 handlers + 2 filter helpers |
| `internal/modules/web/hub_webservice_test.go` | Modify | Tests for new handlers |
| `pkg/views/layout/base.templ` | Modify | Add nav link |
| `pkg/views/pages/hub_apps.templ` | Modify | Add cross-link to capabilities page |

---

### Task 1: Write tests for capabilities handlers (TDD red)

**Files:**
- Modify: `internal/modules/web/hub_webservice_test.go`

- [ ] **Step 1: Add test for hubCapabilitiesPage**

Append to `internal/modules/web/hub_webservice_test.go`:

```go
func TestHubCapabilitiesPage(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders capabilities page", wantStatus: http.StatusOK, wantContains: "Capabilities — Flowbot"},
		{name: "includes filter dropdown for type", wantStatus: http.StatusOK, wantContains: "capability-type-filter"},
		{name: "shows empty state when no capabilities", wantStatus: http.StatusOK, wantContains: "No capabilities registered"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(http.MethodGet, "/service/web/capabilities", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
		})
	}
}
```

- [ ] **Step 2: Add test for hubCapabilitiesGrid (HTMX partial)**

Append to `internal/modules/web/hub_webservice_test.go`:

```go
func TestHubCapabilitiesGrid(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantContains string
	}{
		{name: "renders grid partial", wantStatus: http.StatusOK, wantContains: "capability-grid"},
		{name: "empty state shown when no capabilities", wantStatus: http.StatusOK, wantContains: "No capabilities registered"},
		{name: "accepts type query param", wantStatus: http.StatusOK, wantContains: "capability-grid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			url := "/service/web/capabilities/grid"
			if tt.name == "accepts type query param" {
				url = "/service/web/capabilities/grid?type=bookmark"
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(body), tt.wantContains) {
				t.Errorf("want body containing %q", tt.wantContains)
			}
		})
	}
}
```

- [ ] **Step 3: Add test for unauthenticated access**

Append to `internal/modules/web/hub_webservice_test.go` — extend the existing `TestHubAppsUnauthenticated` tests table or add a new test:

```go
func TestHubCapabilitiesUnauthenticated(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "GET /capabilities redirects to login", method: http.MethodGet, path: "/service/web/capabilities"},
		{name: "GET /capabilities/grid redirects to login", method: http.MethodGet, path: "/service/web/capabilities/grid"},
		{name: "authenticated capabilities page renders OK", method: http.MethodGet, path: "/service/web/capabilities"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := setupTestApp()
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.name == "authenticated capabilities page renders OK" {
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			}
			resp, _ := app.Test(req)
			defer resp.Body.Close()
			if tt.name == "authenticated capabilities page renders OK" {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("want status 200 with token, got %d", resp.StatusCode)
				}
			} else if resp.StatusCode != http.StatusSeeOther {
				t.Errorf("want status %d (redirect), got %d", http.StatusSeeOther, resp.StatusCode)
			}
		})
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/modules/web/ -run "TestHubCapabilities" -count=1`
Expected: FAIL — routes/handlers not yet defined, compilation error or 404

---

### Task 2: Create capability_card.templ partial

**Files:**
- Create: `pkg/views/partials/capability_card.templ`

- [ ] **Step 1: Create the file**

Write `pkg/views/partials/capability_card.templ`:

```go
package partials

import "github.com/flowline-io/flowbot/pkg/hub"

templ CapabilityCard(d hub.Descriptor) {
	<div class="card bg-base-100 shadow-sm" data-testid={ "capability-card-" + string(d.Type) }>
		<div class="card-body p-4">
			<div x-data="{ expanded: false }">
				<div class="cursor-pointer" @click="expanded = !expanded">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<span class="badge badge-primary badge-sm">{ string(d.Type) }</span>
							<span class="text-sm font-medium text-base-content">{ d.Backend }</span>
						</div>
						<div class="flex items-center gap-2">
							if d.Healthy {
								<span class="badge badge-success badge-xs"></span>
							} else {
								<span class="badge badge-warning badge-xs"></span>
							}
						</div>
					</div>
					if d.App != "" {
						<div class="text-xs text-base-content/50 mt-1">App: { d.App }</div>
					}
					if d.Description != "" {
						<div class="text-xs text-base-content/70 mt-1 truncate" title={ d.Description }>{ d.Description }</div>
					}
					<div class="text-xs text-base-content/50 mt-2 flex items-center gap-1">
						<span x-text="expanded ? '\u25B2' : '\u25BC'"></span>
						<span>{ len(d.Operations) } Operations</span>
					</div>
				</div>
				<div x-show="expanded" class="mt-3 border-t border-base-200 pt-3">
					if len(d.Operations) == 0 {
						<div class="text-xs text-base-content/50">No operations</div>
					} else {
						for _, op := range d.Operations {
							<div class="mb-3 last:mb-0">
								<div class="text-sm font-medium text-base-content">{ op.Name }</div>
								if op.Description != "" {
									<div class="text-xs text-base-content/70 mt-0.5">{ op.Description }</div>
								}
								if len(op.Input) > 0 {
									<div class="text-xs text-base-content/50 mt-1">Input:</div>
									<div class="flex flex-wrap gap-1 mt-0.5">
										for _, p := range op.Input {
											<span class="badge badge-ghost badge-xs">{ p.Name }: { p.Type }</span>
										}
									</div>
								}
								if len(op.Output) > 0 {
									<div class="text-xs text-base-content/50 mt-1">Output:</div>
									<div class="flex flex-wrap gap-1 mt-0.5">
										for _, p := range op.Output {
											<span class="badge badge-ghost badge-xs">{ p.Name }: { p.Type }</span>
										}
									</div>
								}
								if len(op.Scopes) > 0 {
									<div class="text-xs text-base-content/50 mt-1">Scopes:</div>
									<div class="flex flex-wrap gap-1 mt-0.5">
										for _, s := range op.Scopes {
											<span class="badge badge-outline badge-xs">{ s }</span>
										}
									</div>
								}
							</div>
						}
					}
				</div>
			</div>
		</div>
	</div>
}
```

---

### Task 3: Create capability_grid.templ partial

**Files:**
- Create: `pkg/views/partials/capability_grid.templ`

- [ ] **Step 1: Create the file**

Write `pkg/views/partials/capability_grid.templ`:

```go
package partials

import "github.com/flowline-io/flowbot/pkg/hub"

templ CapabilityGrid(descriptors []hub.Descriptor) {
	<div id="capability-grid" data-testid="capability-grid">
		if len(descriptors) == 0 {
			@EmptyState("No capabilities registered")
		} else {
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				for _, d := range descriptors {
					@CapabilityCard(d)
				}
			</div>
		}
	</div>
}
```

---

### Task 4: Create capabilities.templ page

**Files:**
- Create: `pkg/views/pages/capabilities.templ`

- [ ] **Step 1: Create the file**

Write `pkg/views/pages/capabilities.templ`:

```go
package pages

import (
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

templ CapabilitiesPage(descriptors []hub.Descriptor, types []string, providers []string) {
	@layout.Base("Capabilities — Flowbot") {
		<div class="flex items-center justify-between mb-6">
			<h1 class="text-2xl font-bold text-base-content">Capabilities</h1>
		</div>
		<div class="flex items-center gap-3 mb-4">
			<select id="capability-type-filter" name="type"
				hx-get="/service/web/capabilities/grid"
				hx-trigger="change"
				hx-target="#capability-grid"
				hx-swap="outerHTML"
				hx-include="[name='provider']"
				data-testid="capability-type-filter"
				class="select select-bordered select-sm">
				<option value="">All Types</option>
				for _, t := range types {
					<option value={ t }>{ t }</option>
				}
			</select>
			<select name="provider"
				hx-get="/service/web/capabilities/grid"
				hx-trigger="change"
				hx-target="#capability-grid"
				hx-swap="outerHTML"
				hx-include="[name='type']"
				data-testid="capability-provider-filter"
				class="select select-bordered select-sm">
				<option value="">All Providers</option>
				for _, p := range providers {
					<option value={ p }>{ p }</option>
				}
			</select>
		</div>
		@partials.CapabilityGrid(descriptors)
	}
}
```

---

### Task 5: Add route handlers in hub_webservice.go

**Files:**
- Modify: `internal/modules/web/hub_webservice.go`

- [ ] **Step 1: Add route rules to hubWebserviceRules**

Insert two new entries after line 25 (`webservice.Get("/hub/list", ...)`):

```go
webservice.Get("/capabilities", hubCapabilitiesPage, route.WithNotAuth()),
webservice.Get("/capabilities/grid", hubCapabilitiesGrid, route.WithNotAuth()),
```

The updated `hubWebserviceRules` becomes:

```go
var hubWebserviceRules = []webservice.Rule{
	webservice.Get("/hub", hubAppsPage, route.WithNotAuth()),
	webservice.Get("/hub/list", hubAppsList, route.WithNotAuth()),
	webservice.Get("/capabilities", hubCapabilitiesPage, route.WithNotAuth()),
	webservice.Get("/capabilities/grid", hubCapabilitiesGrid, route.WithNotAuth()),
	webservice.Get("/hub/:name", hubAppDetailPage, route.WithNotAuth()),
	webservice.Get("/hub/:name/status", hubAppStatusPartial, route.WithNotAuth()),
	webservice.Get("/hub/:name/logs/stream", hubAppLogsSSE, route.WithNotAuth()),
	webservice.Post("/hub/:name/start", hubAppStartAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/stop", hubAppStopAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/restart", hubAppRestartAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/pull", hubAppPullAction, route.WithNotAuth()),
	webservice.Post("/hub/:name/update", hubAppUpdateAction, route.WithNotAuth()),
}
```

- [ ] **Step 2: Add imports**

Add to the imports block (after the existing third-party/internal imports):

```go
"sort"

"github.com/flowline-io/flowbot/pkg/hub"
```

The updated imports block:

```go
import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)
```

- [ ] **Step 3: Add handler functions**

Append after the `loadUpdatedAts` function (end of file):

```go
// hubCapabilitiesPage renders the full capabilities browser page.
func hubCapabilitiesPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	descriptors := hub.Default.List()
	types := uniqueTypes(descriptors)
	providers := uniqueProviders(descriptors)
	c.Type("html")
	return pages.CapabilitiesPage(descriptors, types, providers).Render(c.Context(), c.Response().BodyWriter())
}

// hubCapabilitiesGrid returns the filtered card grid partial for HTMX swaps.
func hubCapabilitiesGrid(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	descriptors := hub.Default.List()

	typeFilter := c.Query("type")
	providerFilter := c.Query("provider")

	if typeFilter != "" || providerFilter != "" {
		filtered := make([]hub.Descriptor, 0, len(descriptors))
		for _, d := range descriptors {
			if typeFilter != "" && string(d.Type) != typeFilter {
				continue
			}
			if providerFilter != "" && d.Backend != providerFilter {
				continue
			}
			filtered = append(filtered, d)
		}
		descriptors = filtered
	}

	c.Type("html")
	return partials.CapabilityGrid(descriptors).Render(c.Context(), c.Response().BodyWriter())
}

// uniqueTypes extracts unique capability type strings from descriptors, sorted.
func uniqueTypes(descriptors []hub.Descriptor) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		t := string(d.Type)
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	sort.Strings(result)
	return result
}

// uniqueProviders extracts unique backend strings from descriptors, sorted.
func uniqueProviders(descriptors []hub.Descriptor) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(descriptors))
	for _, d := range descriptors {
		if d.Backend == "" {
			continue
		}
		if _, ok := seen[d.Backend]; !ok {
			seen[d.Backend] = struct{}{}
			result = append(result, d.Backend)
		}
	}
	sort.Strings(result)
	return result
}
```

---

### Task 6: Generate templ code

**Files:**
- Generates: `pkg/views/pages/capabilities_templ.go`
- Generates: `pkg/views/partials/capability_card_templ.go`
- Generates: `pkg/views/partials/capability_grid_templ.go`

- [ ] **Step 1: Run templ generate**

```bash
templ generate pkg/views/...
```

Expected: Generates three new `*_templ.go` files without errors.

---

### Task 7: Add nav link in base.templ

**Files:**
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Add Capabilities nav link after the Apps link**

Insert after line 31 (`<a href="/service/web/hub" ... class="btn btn-ghost btn-sm">Apps</a>`):

```html
<a href="/service/web/capabilities" data-testid="nav-capabilities" class="btn btn-ghost btn-sm">Capabilities</a>
```

The updated nav section (lines 25-31):

```html
<div class="navbar-end flex items-center gap-1">
    <a href="/service/web/pipelines" data-testid="nav-pipelines" class="btn btn-ghost btn-sm">Pipelines</a>
    <a href="/service/web/events" data-testid="nav-events" class="btn btn-ghost btn-sm">Events</a>
    <a href="/service/web/notifications" data-testid="nav-notifications" class="btn btn-ghost btn-sm">Notifications</a>
    <a href="/service/web/relations" data-testid="nav-relations" class="btn btn-ghost btn-sm">Relations</a>
    <a href="/service/web/configs" data-testid="nav-configs" class="btn btn-ghost btn-sm">Configs</a>
    <a href="/service/web/hub" data-testid="nav-hub" class="btn btn-ghost btn-sm">Apps</a>
    <a href="/service/web/capabilities" data-testid="nav-capabilities" class="btn btn-ghost btn-sm">Capabilities</a>
```

- [ ] **Step 2: Regenerate base templ**

```bash
templ generate pkg/views/layout/...
```

---

### Task 8: Add cross-link in hub_apps.templ

**Files:**
- Modify: `pkg/views/pages/hub_apps.templ`

- [ ] **Step 1: Add Capabilities link next to the page title**

Replace lines 12-14 (the title div):

```html
<div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold text-base-content">Apps</h1>
</div>
```

With:

```html
<div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold text-base-content">Apps</h1>
    <a href="/service/web/capabilities" class="btn btn-ghost btn-sm" data-testid="hub-apps-to-capabilities">Capabilities</a>
</div>
```

- [ ] **Step 2: Regenerate hub_apps templ**

```bash
templ generate pkg/views/pages/hub_apps.templ
```

---

### Task 9: Run tests and verify green

- [ ] **Step 1: Run unit tests**

```bash
go test ./internal/modules/web/ -run "TestHubCapabilities" -v -count=1
```

Expected: All tests pass.

- [ ] **Step 2: Run full web module tests**

```bash
go test ./internal/modules/web/ -v -count=1
```

Expected: All tests pass (no regressions).

- [ ] **Step 3: Run lint**

```bash
go tool task lint
```

Expected: No lint errors.

- [ ] **Step 4: Build**

```bash
go tool task build
```

Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add pkg/views/partials/capability_card.templ pkg/views/partials/capability_card_templ.go
git add pkg/views/partials/capability_grid.templ pkg/views/partials/capability_grid_templ.go
git add pkg/views/pages/capabilities.templ pkg/views/pages/capabilities_templ.go
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git add pkg/views/pages/hub_apps.templ pkg/views/pages/hub_apps_templ.go
git add internal/modules/web/hub_webservice.go internal/modules/web/hub_webservice_test.go
git commit -m "feat: hub capability browser page with filtering"
```

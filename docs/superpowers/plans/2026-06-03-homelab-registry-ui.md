# Homelab Registry UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a card-based Homelab Registry page with search, capability-type filter, app detail view, and manual rescan.

**Architecture:** Alpine.js client-side filtering on a card grid rendered server-side by templ. Rescan triggers full Scanner + ProbeEngine + Registry Replace pipeline via HTMX POST. Detail page is server-rendered with services table and endpoint list.

**Tech Stack:** Go, templ, DaisyUI v5, Alpine.js 3.x, HTMX 2.x, Fiber v3, `pkg/homelab/`, `pkg/homelab/probe/`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `pkg/homelab/version.go` | Create | `ParseImageVersion()` and `AppVersion()` utilities |
| `pkg/homelab/version_test.go` | Create | Table-driven tests |
| `internal/server/homelab.go` | Modify | Extract `RunHomelabScan()` exported function |
| `pkg/views/partials/homelab_card.templ` | Create | Single app card component |
| `pkg/views/partials/homelab_grid.templ` | Create | Card grid with search, filter, rescan button, empty state |
| `pkg/views/pages/homelab.templ` | Create | Full page wrapping grid in `@layout.Base()` |
| `pkg/views/pages/homelab_detail.templ` | Create | App detail: info, services table, endpoint list |
| `pkg/views/layout/base.templ` | Modify | Add "Registry" navbar link |
| `public/js/homelab-registry.js` | Create | Alpine.js controller for search/filter |
| `internal/modules/web/homelab_webservice.go` | Create | Route definitions + handler functions |
| `internal/modules/web/homelab_webservice_test.go` | Create | Handler table-driven tests |
| `internal/modules/web/module.go` | Modify | Register `homelabWebserviceRules` |
| `tests/specs/homelab_registry_spec_test.go` | Create | BDD acceptance specs |

---

### Task 1: ParseImageVersion utility (TDD)

**Files:**
- Create: `pkg/homelab/version.go`
- Create: `pkg/homelab/version_test.go`

- [ ] **Step 1: Write the failing tests**

Write `pkg/homelab/version_test.go`:

```go
package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImageVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{
			name:     "image_with_tag",
			image:    "gitea/gitea:1.22.3",
			expected: "1.22.3",
		},
		{
			name:     "image_with_alpine_tag",
			image:    "postgres:16-alpine",
			expected: "16-alpine",
		},
		{
			name:     "image_without_tag",
			image:    "nginx",
			expected: "",
		},
		{
			name:     "image_with_digest",
			image:    "nginx@sha256:abc123",
			expected: "",
		},
		{
			name:     "image_with_registry_and_tag",
			image:    "docker.io/library/redis:7.0",
			expected: "7.0",
		},
		{
			name:     "empty_image",
			image:    "",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseImageVersion(tt.image)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAppVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		services []ComposeService
		expected string
	}{
		{
			name: "first_service_with_tag",
			services: []ComposeService{
				{Name: "app", Image: "gitea/gitea:1.22.3"},
				{Name: "db", Image: "postgres:16"},
			},
			expected: "1.22.3",
		},
		{
			name: "skip_service_without_tag",
			services: []ComposeService{
				{Name: "app", Image: "nginx"},
				{Name: "db", Image: "postgres:16"},
			},
			expected: "16",
		},
		{
			name:     "no_services",
			services: []ComposeService{},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := App{Name: tt.name, Services: tt.services}
			got := AppVersion(app)
			assert.Equal(t, tt.expected, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/homelab/ -run TestParseImageVersion -count=1`
Expected: FAIL (undefined: ParseImageVersion)

- [ ] **Step 3: Write minimal implementation**

Write `pkg/homelab/version.go`:

```go
package homelab

import "strings"

// ParseImageVersion extracts the version tag from a Docker image reference.
// Returns the portion after the last colon if present; returns an empty string
// when there is no colon or the image uses a digest reference (@sha256:...).
func ParseImageVersion(image string) string {
	if strings.Contains(image, "@") {
		return ""
	}
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		return image[idx+1:]
	}
	return ""
}

// AppVersion extracts the application version from the first service
// that has a tagged image. Returns an empty string if no tag is found.
func AppVersion(app App) string {
	for _, svc := range app.Services {
		if v := ParseImageVersion(svc.Image); v != "" {
			return v
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/homelab/ -run "TestParseImageVersion|TestAppVersion" -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/homelab/version.go pkg/homelab/version_test.go
git commit -m "feat: add ParseImageVersion and AppVersion utilities"
```

---

### Task 2: Extract RunHomelabScan in server package

**Files:**
- Modify: `internal/server/homelab.go`

- [ ] **Step 1: Add RunHomelabScan and refactor initHomelabRegistry**

Open `internal/server/homelab.go`. Add the exported function before `initHomelabRegistry`, then refactor `initHomelabRegistry` to call it:

```go
// RunHomelabScan executes a full homelab scan + probe + registry update cycle.
// It walks the configured apps directory, discovers compose files, runs the
// probe engine for endpoint/auth discovery, and replaces the default registry.
// Exported for use by the homelab web handler to support manual rescan.
func RunHomelabScan(cfg config.Homelab) error {
	homeConfig := homelabConfig(cfg)
	if homeConfig.AppsDir == "" && homeConfig.Root == "" {
		return fmt.Errorf("homelab app registry disabled: apps_dir and root are empty")
	}
	apps, err := homelab.NewScanner(homeConfig).Scan()
	if err != nil {
		return fmt.Errorf("scan homelab apps: %w", err)
	}

	if eng := probe.NewEngine(homeConfig.Discovery); eng != nil {
		ctx, cancel := context.WithTimeout(context.Background(), homeConfig.Discovery.ProbeTimeout*2)
		defer cancel()
		probeResults := eng.ProbeAll(ctx, apps)
		if len(probeResults) > 0 {
			apps = mergeProbeResults(apps, probeResults)
		}
	}

	homelab.DefaultRegistry.Replace(apps)
	homelab.DefaultRegistry.SetPermissions(homeConfig.Permissions)
	if store.Database != nil && store.Database.GetDB() != nil {
		if client, ok := store.Database.GetDB().(*store.Client); ok {
			if err := store.NewHubStore(client).SaveHomelabApps(context.Background(), apps); err != nil {
				return fmt.Errorf("persist homelab apps: %w", err)
			}
		}
	}
	flog.Info("homelab app registry rescanned with %d apps", len(apps))
	return nil
}
```

Refactor `initHomelabRegistry` from its current body to:

```go
func initHomelabRegistry(cfg config.Homelab) error {
	homelabRuntime = homelab.NewRuntime(homelabConfig(cfg).Runtime, homelabConfig(cfg).AppsDir)
	homelab.DefaultRuntime = homelabRuntime
	if cfg.AppsDir == "" && cfg.Root == "" {
		flog.Info("homelab app registry disabled: homelab.apps_dir and homelab.root are empty")
		return nil
	}
	return RunHomelabScan(cfg)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/server/`
Expected: success (no errors)

- [ ] **Step 3: Run existing server tests**

Run: `go test ./internal/server/ -count=1`
Expected: all existing tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/server/homelab.go
git commit -m "refactor: extract RunHomelabScan for reuse by web rescan handler"
```

---

### Task 3: Create homelab card partial

**Files:**
- Create: `pkg/views/partials/homelab_card.templ`

- [ ] **Step 1: Write homelab_card.templ**

```templ
package partials

import (
	"sort"
	"strings"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

var avatarColors = []string{
	"bg-primary",
	"bg-secondary",
	"bg-accent",
	"bg-info",
	"bg-success",
	"bg-warning",
	"bg-error",
	"bg-neutral",
}

func avatarColorName(name string) string {
	h := 0
	for _, b := range []byte(name) {
		h = h*31 + int(b)
	}
	return avatarColors[(h%len(avatarColors)+len(avatarColors))%len(avatarColors)]
}

func avatarLetter(name string) string {
	if name == "" {
		return "?"
	}
	return string(name[0])
}

func firstEndpointURL(caps []homelab.AppCapability) string {
	for _, cap := range caps {
		if cap.Endpoint != nil && cap.Endpoint.BaseURL != "" {
			return cap.Endpoint.BaseURL
		}
	}
	return ""
}

func capsText(caps []homelab.AppCapability) string {
	items := make([]string, len(caps))
	for i, c := range caps {
		items[i] = c.Capability
	}
	return strings.Join(items, ",")
}

templ HomelabCard(app homelab.App) {
	<div class="card bg-base-100 shadow-sm hover:shadow-md transition-shadow"
		data-app-name={ app.Name }
		data-app-caps={ capsText(app.Capabilities) }
		data-testid={ "homelab-card-" + app.Name }
		x-show="appMatches($el)"
		x-data>
		<div class="card-body p-4">
			<div class="flex items-center gap-3">
				<div class={ "avatar placeholder " + avatarColorName(app.Name) }>
					<div class="w-10 rounded-full text-white">
						<span class="text-lg font-semibold">{ avatarLetter(app.Name) }</span>
					</div>
				</div>
				<div class="flex-1 min-w-0">
					<h3 class="text-base font-medium text-base-content truncate">
						<a href={ templ.URL("/service/web/homelab/" + app.Name) }
							class="link link-hover"
							data-testid={ "homelab-link-" + app.Name }>
							{ app.Name }
						</a>
					</h3>
					if url := firstEndpointURL(app.Capabilities); url != "" {
						<a href={ url } target="_blank"
							class="text-xs text-primary link truncate block"
							data-testid={ "homelab-url-" + app.Name }>
							{ url }
						</a>
					}
				</div>
			</div>
			<div class="flex items-center gap-2 mt-3">
				@HomelabStatusBadge(app.Status)
				if len(app.Capabilities) > 0 {
					<div class="flex flex-wrap gap-1 ml-auto">
						for _, cap := range app.Capabilities {
							<span class="badge badge-outline badge-xs">{ cap.Capability }</span>
						}
					</div>
				}
			</div>
		</div>
	</div>
}

templ HomelabStatusBadge(status homelab.AppStatus) {
	switch status {
	case "running":
		<span class="badge badge-success badge-sm" data-testid="status-badge">online</span>
	case "stopped":
		<span class="badge badge-ghost badge-sm" data-testid="status-badge">offline</span>
	case "partial":
		<span class="badge badge-warning badge-sm" data-testid="status-badge">warning</span>
	default:
		<span class="badge badge-error badge-sm" data-testid="status-badge">unknown</span>
	}
}
```

- [ ] **Step 2: Generate Go code from templ**

Run: `templ generate pkg/views/partials/homelab_card.templ`
Expected: generates `pkg/views/partials/homelab_card_templ.go`

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/partials/`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add pkg/views/partials/homelab_card.templ pkg/views/partials/homelab_card_templ.go
git commit -m "feat: add homelab card and status badge partials"
```

---

### Task 4: Create homelab grid partial

**Files:**
- Create: `pkg/views/partials/homelab_grid.templ`

- [ ] **Step 1: Write homelab_grid.templ**

```templ
package partials

import (
	"sort"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func uniqueCapTypes(apps []homelab.App) []string {
	seen := make(map[string]struct{})
	for _, app := range apps {
		for _, cap := range app.Capabilities {
			seen[cap.Capability] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for k := range seen {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

templ HomelabGrid(apps []homelab.App, scannedAt string) {
	<div id="homelab-registry" x-data="homelabRegistry()"
		data-testid="homelab-registry">
		<div class="flex items-end gap-2 mb-4 flex-wrap">
			<div class="form-control flex-1 min-w-48">
				<input type="search" class="input input-bordered input-sm"
					placeholder="Search apps..."
					x-model="search"
					data-testid="homelab-search"/>
			</div>
			<select class="select select-bordered select-sm"
				x-model="filterCapability"
				data-testid="homelab-capability-filter">
				<option value="">All Capabilities</option>
				for _, capType := range uniqueCapTypes(apps) {
					<option value={ capType }>{ capType }</option>
				}
			</select>
			<button hx-post="/service/web/homelab/rescan"
				class="btn btn-sm btn-outline"
				data-testid="homelab-rescan">
				<span class="loading loading-spinner loading-xs htmx-indicator mr-1"></span>
				Rescan
			</button>
			if scannedAt != "" {
				<span class="text-xs text-base-content/50"
					data-testid="homelab-scanned-at">Last scan: { scannedAt }</span>
			}
		</div>
		if len(apps) == 0 {
			@EmptyState("No apps discovered. Configure homelab.apps_dir in flowbot.yaml.")
		} else {
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
				data-testid="homelab-card-grid">
				for _, app := range apps {
					@HomelabCard(app)
				}
			</div>
		}
	</div>
}
```

- [ ] **Step 2: Generate Go code**

Run: `templ generate pkg/views/partials/homelab_card.templ pkg/views/partials/homelab_grid.templ`
Expected: generates both `*_templ.go` files

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/views/partials/`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add pkg/views/partials/homelab_grid.templ pkg/views/partials/homelab_grid_templ.go
git add pkg/views/partials/homelab_card.templ pkg/views/partials/homelab_card_templ.go
git commit -m "feat: add homelab grid partial with search, filter, and rescan"
```

---

### Task 5: Create Alpine.js controller

**Files:**
- Create: `public/js/homelab-registry.js`

- [ ] **Step 1: Write homelab-registry.js**

```javascript
Alpine.data('homelabRegistry', () => ({
  search: '',
  filterCapability: '',

  appMatches(el) {
    const name = el.getAttribute('data-app-name') || '';
    const caps = el.getAttribute('data-app-caps') || '';
    const searchMatch = !this.search
      || name.toLowerCase().includes(this.search.toLowerCase());
    const capMatch = !this.filterCapability
      || caps.split(',').includes(this.filterCapability);
    return searchMatch && capMatch;
  },
}));
```

- [ ] **Step 2: Commit**

```bash
git add public/js/homelab-registry.js
git commit -m "feat: add homelab registry Alpine.js controller"
```

---

### Task 6: Create page templates

**Files:**
- Create: `pkg/views/pages/homelab.templ`
- Create: `pkg/views/pages/homelab_detail.templ`

- [ ] **Step 1: Write homelab.templ**

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

templ HomelabPage(apps []homelab.App, scannedAt string) {
	@layout.Base("Registry — Flowbot") {
		<script src="/static/js/homelab-registry.js" defer></script>
		<div class="flex items-center justify-between mb-6">
			<h1 class="text-2xl font-bold text-base-content">Homelab Registry</h1>
		</div>
		@partials.HomelabGrid(apps, scannedAt)
	}
}
```

- [ ] **Step 2: Write homelab_detail.templ**

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

templ HomelabDetailPage(app homelab.App, status homelab.AppStatus, version string, scannedAt string) {
	@layout.Base(app.Name + " — Flowbot") {
		<div class="mb-4">
			<a href="/service/web/homelab" class="btn btn-ghost btn-sm">&larr; Back to Registry</a>
		</div>
		<div class="card bg-base-100 shadow-sm mb-6">
			<div class="card-body">
				<div class="flex items-center gap-3 mb-4">
					<div class="avatar placeholder bg-primary">
						<div class="w-12 rounded-full text-white">
							<span class="text-xl font-semibold">
								if app.Name != "" {
									{ string(app.Name[0]) }
								}
							</span>
						</div>
					</div>
					<div>
						<h1 class="card-title text-2xl">{ app.Name }</h1>
						<div class="flex items-center gap-2 mt-1">
							@HomelabDetailStatusBadge(status)
							<span class="text-sm text-base-content/50">Health: { string(app.Health) }</span>
						</div>
					</div>
				</div>
				<div class="grid grid-cols-2 gap-2 text-sm text-base-content/70 mb-4">
					<div>
						<span class="text-base-content/50">Path:</span>
						<code class="text-xs">{ app.Path }</code>
					</div>
					if app.ComposeFile != "" {
						<div>
							<span class="text-base-content/50">Compose:</span>
							<code class="text-xs">{ app.ComposeFile }</code>
						</div>
					}
					if version != "" {
						<div>
							<span class="text-base-content/50">Version:</span>
							<span class="badge badge-sm">{ version }</span>
						</div>
					}
					if scannedAt != "" {
						<div>
							<span class="text-base-content/50">Last Discovered:</span>
							<span>{ scannedAt }</span>
						</div>
					}
				</div>
			</div>
		</div>
		if len(app.Services) > 0 {
			<div class="card bg-base-100 shadow-sm mb-6">
				<div class="card-body">
					<h2 class="card-title text-lg mb-2">Services</h2>
					<div class="overflow-x-auto">
						<table class="table table-xs">
							<thead>
								<tr>
									<th>Service</th>
									<th>Image</th>
									<th>Container</th>
									<th>Ports</th>
								</tr>
							</thead>
							<tbody>
								for _, svc := range app.Services {
									<tr>
										<td class="font-medium">{ svc.Name }</td>
										<td><code class="text-xs">{ svc.Image }</code></td>
										<td>{ svc.Container }</td>
										<td>
											<div class="flex flex-wrap gap-1">
												for _, p := range svc.Ports {
													<span class="badge badge-ghost badge-xs">
														if p.HostPort != "" {
															{ p.HostPort }:{ p.Container }
														} else {
															{ p.Container }
														}
														if p.Protocol != "" {
															/{ p.Protocol }
														}
													</span>
												}
											</div>
										</td>
									</tr>
								}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		}
		if len(app.Capabilities) > 0 {
			<div class="card bg-base-100 shadow-sm mb-6">
				<div class="card-body">
					<h2 class="card-title text-lg mb-2">Exposed Endpoints</h2>
					<div class="overflow-x-auto">
						<table class="table table-xs">
							<thead>
								<tr>
									<th>Capability</th>
									<th>Backend</th>
									<th>Base URL</th>
									<th>Auth</th>
									<th>Health</th>
								</tr>
							</thead>
							<tbody>
								for _, cap := range app.Capabilities {
									<tr>
										<td><span class="badge badge-primary badge-xs">{ cap.Capability }</span></td>
										<td>{ cap.Backend }</td>
										<td>
											if cap.Endpoint != nil && cap.Endpoint.BaseURL != "" {
												<a href={ cap.Endpoint.BaseURL } target="_blank"
													class="link link-hover text-xs">
													{ cap.Endpoint.BaseURL }
												</a>
											}
										</td>
										<td>
											if cap.Auth != nil {
												<span class="badge badge-ghost badge-xs">{ string(cap.Auth.Type) }</span>
											} else {
												<span class="text-base-content/50 text-xs">-</span>
											}
										</td>
										<td>
											if cap.Endpoint != nil && cap.Endpoint.Health != "" {
												<a href={ cap.Endpoint.Health } target="_blank"
													class="link link-hover text-xs">
													{ cap.Endpoint.Health }
												</a>
											} else {
												<span class="text-base-content/50 text-xs">-</span>
											}
										</td>
									</tr>
								}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		}
	}
}

templ HomelabDetailStatusBadge(status homelab.AppStatus) {
	switch status {
	case "running":
		<span class="badge badge-success" data-testid="status-badge">online</span>
	case "stopped":
		<span class="badge badge-ghost" data-testid="status-badge">offline</span>
	case "partial":
		<span class="badge badge-warning" data-testid="status-badge">warning</span>
	default:
		<span class="badge badge-error" data-testid="status-badge">unknown</span>
	}
}
```

- [ ] **Step 3: Generate Go code**

Run: `templ generate pkg/views/pages/homelab.templ pkg/views/pages/homelab_detail.templ`
Expected: generates both `*_templ.go` files

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/views/pages/`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add pkg/views/pages/homelab.templ pkg/views/pages/homelab_templ.go
git add pkg/views/pages/homelab_detail.templ pkg/views/pages/homelab_detail_templ.go
git commit -m "feat: add homelab registry and detail page templates"
```

---

### Task 7: Add Registry navbar link

**Files:**
- Modify: `pkg/views/layout/base.templ`

- [ ] **Step 1: Add Registry link between Apps and Capabilities**

In `pkg/views/layout/base.templ`, after the "Apps" link (line 35), insert:

```templ
<a href="/service/web/homelab" data-testid="nav-homelab" class="btn btn-ghost btn-sm">Registry</a>
```

So the navbar section becomes:

```
<a href="/service/web/hub" data-testid="nav-hub" class="btn btn-ghost btn-sm">Apps</a>
<a href="/service/web/homelab" data-testid="nav-homelab" class="btn btn-ghost btn-sm">Registry</a>
<a href="/service/web/capabilities" data-testid="nav-capabilities" class="btn btn-ghost btn-sm">Capabilities</a>
```

- [ ] **Step 2: Regenerate Go code**

Run: `templ generate pkg/views/layout/base.templ`
Expected: updates `pkg/views/layout/base_templ.go`

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/layout/`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: add Registry link to navbar"
```

---

### Task 8: Create webservice handler

**Files:**
- Create: `internal/modules/web/homelab_webservice.go`

- [ ] **Step 1: Write homelab_webservice.go**

```go
package web

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

var homelabWebserviceRules = []webservice.Rule{
	webservice.Get("/homelab", homelabRegistryPage, route.WithNotAuth()),
	webservice.Get("/homelab/:name", homelabRegistryDetailPage, route.WithNotAuth()),
	webservice.Post("/homelab/rescan", homelabRegistryRescan, route.WithNotAuth()),
}

// homelabRegistryPage renders the full homelab registry card list page.
func homelabRegistryPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	apps := homelab.DefaultRegistry.List()
	updatedAts := loadUpdatedAts(c.Context())
	scannedAt := latestScannedAt(updatedAts)
	c.Type("html")
	return pages.HomelabPage(apps, scannedAt).Render(c.Context(), c.Response().BodyWriter())
}

// homelabRegistryDetailPage renders the detail page for a single homelab app.
func homelabRegistryDetailPage(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	name := c.Params("name")
	app, ok := homelab.DefaultRegistry.Get(name)
	if !ok {
		return c.Status(http.StatusNotFound).SendString("app not found")
	}
	status, err := homelab.DefaultRuntime.Status(c.Context(), app)
	if err != nil {
		status = app.Status
	}
	version := homelab.AppVersion(app)
	updatedAts := loadUpdatedAts(c.Context())
	scannedAt := ""
	if ts, ok := updatedAts[app.Name]; ok {
		scannedAt = ts
	}
	c.Type("html")
	return pages.HomelabDetailPage(app, status, version, scannedAt).Render(c.Context(), c.Response().BodyWriter())
}

// homelabRegistryRescan triggers a full homelab scan + probe cycle.
func homelabRegistryRescan(c fiber.Ctx) error {
	if err := authenticateWeb(c); err != nil {
		return err
	}
	if err := server.RunHomelabScan(config.App.Homelab); err != nil {
		flog.Warn("homelab rescan failed: %v", err)
		c.Set("HX-Redirect", "/service/web/homelab")
		return c.SendStatus(http.StatusOK)
	}
	c.Set("HX-Redirect", "/service/web/homelab")
	return c.SendStatus(http.StatusOK)
}

// latestScannedAt returns the most recent UpdatedAt timestamp from the apps map.
func latestScannedAt(updatedAts map[string]string) string {
	latest := ""
	for _, ts := range updatedAts {
		if ts > latest {
			latest = ts
		}
	}
	return latest
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/web/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/homelab_webservice.go
git commit -m "feat: add homelab registry webservice handlers"
```

---

### Task 9: Register routes in module.go

**Files:**
- Modify: `internal/modules/web/module.go`

- [ ] **Step 1: Add webservice registration**

In `internal/modules/web/module.go`, find the `Webservice` method. After the existing `module.Webservice(app, Name, hubWebserviceRules)` line, add the new registration. The order should be:

```go
module.Webservice(app, Name, hubWebserviceRules)
module.Webservice(app, Name, homelabWebserviceRules)
```

Also, update the `Rules()` return value to include `homelabWebserviceRules`:

```go
return []any{
    webserviceRules,
    hubWebserviceRules,
    homelabWebserviceRules,
    pipelineWebserviceRules,
    viewWebserviceRules,
    eventWebserviceRules,
    relationsWebserviceRules,
    notificationWebserviceRules,
    notifySettingsWebserviceRules,
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/web/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/module.go
git commit -m "feat: register homelab registry webservice routes"
```

---

### Task 10: Write handler tests (TDD)

**Files:**
- Create: `internal/modules/web/homelab_webservice_test.go`

- [ ] **Step 1: Write tests**

```go
package web

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestHomelabRegistryPage(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{})
	_, h := setupWebTestForE2E(t, app)
	defer h.Cleanup()

	homelab.DefaultRegistry.Replace([]homelab.App{
		{
			Name:   "test-app",
			Path:   "/data/apps/test",
			Status: homelab.AppStatusRunning,
			Capabilities: []homelab.AppCapability{
				{Capability: "bookmark"},
			},
		},
	})

	// Save sync point for parallel safety.
	_ = homelab.DefaultRegistry.List()

	req := h.NewAuthenticatedRequest("GET", "/service/web/homelab", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body := readBody(t, resp)
	assert.Contains(t, body, "Homelab Registry")
	assert.Contains(t, body, "test-app")
	assert.Contains(t, body, "bookmark")
}

func TestHomelabRegistryDetailPage(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{})
	_, h := setupWebTestForE2E(t, app)
	defer h.Cleanup()

	homelab.DefaultRegistry.Replace([]homelab.App{
		{
			Name:   "detail-app",
			Path:   "/data/apps/detail",
			Status: homelab.AppStatusRunning,
			Services: []homelab.ComposeService{
				{Name: "web", Image: "nginx:1.25"},
			},
			Capabilities: []homelab.AppCapability{
				{
					Capability: "forge",
					Endpoint:   &homelab.EndpointInfo{BaseURL: "http://localhost:3000"},
				},
			},
		},
	})

	req := h.NewAuthenticatedRequest("GET", "/service/web/homelab/detail-app", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body := readBody(t, resp)
	assert.Contains(t, body, "detail-app")
	assert.Contains(t, body, "1.25")
	assert.Contains(t, body, "forge")
	assert.Contains(t, body, "http://localhost:3000")
}

func TestHomelabRegistryDetailPageNotFound(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{})
	_, h := setupWebTestForE2E(t, app)
	defer h.Cleanup()

	homelab.DefaultRegistry.Replace(nil)

	req := h.NewAuthenticatedRequest("GET", "/service/web/homelab/nonexistent", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestHomelabRegistryRescan(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{})
	_, h := setupWebTestForE2E(t, app)
	defer h.Cleanup()

	req := h.NewAuthenticatedRequest("POST", "/service/web/homelab/rescan", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	// HX-Redirect header should be present.
	assert.Equal(t, "/service/web/homelab", resp.Header.Get("HX-Redirect"))
}
```

- [ ] **Step 2: Check test helper patterns**

Verify the existing `hub_webservice_test.go` uses similar patterns (setupWebTestForE2E, NewAuthenticatedRequest, readBody). If not found, adapt to the actual test helpers used in the codebase.

Run: `grep -rn "setupWebTestForE2E" internal/modules/web/`
Expected: should find existing usage in other test files

Note: The test helpers may be named differently. After checking actual helper names, adjust the test code accordingly. Common patterns found in existing tests:
- `InitForE2E()` / `MountForE2E()` in `module.go`
- Direct `app.Test(req)` with Fiber test requests
- `homelab.DefaultRegistry.Replace()` to seed test data

- [ ] **Step 3: Run tests**

Run: `go test ./internal/modules/web/ -run TestHomelabRegistry -count=1`
Expected: tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/homelab_webservice_test.go
git commit -m "test: add homelab registry webservice handler tests"
```

---

### Task 11: Write BDD acceptance specs

**Files:**
- Create: `tests/specs/homelab_registry_spec_test.go`

- [ ] **Step 1: Write BDD spec**

```go
//go:build integration

package specs

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

var _ = Describe("Homelab Registry UI", func() {
	BeforeEach(func() {
		homelab.DefaultRegistry.Replace([]homelab.App{
			{
				Name:   "gitea",
				Path:   "/data/apps/gitea",
				Status: homelab.AppStatusRunning,
				Health: homelab.HealthHealthy,
				Services: []homelab.ComposeService{
					{Name: "gitea", Image: "gitea/gitea:1.22.3", Container: "gitea", Ports: []homelab.PortMapping{{HostPort: "3000", Container: "3000", Protocol: "tcp"}}},
					{Name: "db", Image: "postgres:16", Container: "gitea-db"},
				},
				Capabilities: []homelab.AppCapability{
					{Capability: "forge", Backend: "gitea", Endpoint: &homelab.EndpointInfo{BaseURL: "http://git.local:3000", Health: "/api/v1/healthz"}, Auth: &homelab.AuthInfo{Type: homelab.AuthAPIToken}},
				},
			},
			{
				Name:   "karakeep",
				Path:   "/data/apps/karakeep",
				Status: homelab.AppStatusStopped,
				Capabilities: []homelab.AppCapability{
					{Capability: "bookmark"},
				},
			},
		})
	})

	AfterEach(func() {
		homelab.DefaultRegistry.Replace(nil)
	})

	Describe("GET /service/web/homelab", func() {
		It("renders the registry page with app cards", func() {
			req, _ := http.NewRequest("GET", srv.URL+"/service/web/homelab", nil)
			req.Header.Set("Accept", "text/html")
			addTestAuthCookie(req)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			body := readResponseBody(resp)
			Expect(body).To(ContainSubstring("Homelab Registry"))
			Expect(body).To(ContainSubstring("gitea"))
			Expect(body).To(ContainSubstring("karakeep"))
			Expect(body).To(ContainSubstring("forge"))
			Expect(body).To(ContainSubstring("bookmark"))
		})

		It("shows empty state when no apps are registered", func() {
			homelab.DefaultRegistry.Replace(nil)
			req, _ := http.NewRequest("GET", srv.URL+"/service/web/homelab", nil)
			req.Header.Set("Accept", "text/html")
			addTestAuthCookie(req)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			body := readResponseBody(resp)
			Expect(body).To(ContainSubstring("No apps discovered"))
		})
	})

	Describe("GET /service/web/homelab/:name", func() {
		It("renders detail page with services and endpoints", func() {
			req, _ := http.NewRequest("GET", srv.URL+"/service/web/homelab/gitea", nil)
			req.Header.Set("Accept", "text/html")
			addTestAuthCookie(req)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			body := readResponseBody(resp)
			Expect(body).To(ContainSubstring("gitea"))
			Expect(body).To(ContainSubstring("1.22.3"))
			Expect(body).To(ContainSubstring("gitea/gitea"))
			Expect(body).To(ContainSubstring("forge"))
			Expect(body).To(ContainSubstring("http://git.local:3000"))
			Expect(body).To(ContainSubstring("Exposed Endpoints"))
		})

		It("returns 404 for unknown app", func() {
			req, _ := http.NewRequest("GET", srv.URL+"/service/web/homelab/unknown", nil)
			req.Header.Set("Accept", "text/html")
			addTestAuthCookie(req)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("POST /service/web/homelab/rescan", func() {
		It("returns HX-Redirect header", func() {
			req, _ := http.NewRequest("POST", srv.URL+"/service/web/homelab/rescan", nil)
			addTestAuthCookie(req)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get("HX-Redirect")).To(Equal("/service/web/homelab"))
		})
	})
})
```

- [ ] **Step 2: Verify BDD helpers exist**

Check that `srv.URL`, `httpClient`, `addTestAuthCookie`, and `readResponseBody` are defined in the BDD test helper/suite. Run:

```bash
grep -rn "var srv\b\|var httpClient\|func addTestAuthCookie\|func readResponseBody" tests/specs/
```

Expected: finds definitions in suite setup files. If not, match the patterns used in existing spec files like `tests/specs/hub_spec_test.go`.

- [ ] **Step 3: Run BDD tests**

Run: `go tool task test:specs`
Expected: Homelab Registry specs pass

- [ ] **Step 4: Commit**

```bash
git add tests/specs/homelab_registry_spec_test.go
git commit -m "test: add homelab registry BDD acceptance specs"
```

---

### Task 12: Format, lint, final verification

- [ ] **Step 1: Run format**

```bash
go tool task format
```

- [ ] **Step 2: Run lint**

```bash
go tool task lint
```
Expected: no new lint errors

- [ ] **Step 3: Run all unit tests**

```bash
go tool task test
```
Expected: all tests PASS

- [ ] **Step 4: Regenerate all templ files**

```bash
templ generate pkg/views/...
```

- [ ] **Step 5: Full build**

```bash
go tool task build
```
Expected: builds successfully

- [ ] **Step 6: Commit any remaining changes**

```bash
git add -A
git diff --cached --stat
git commit -m "chore: format, lint, and regenerate templ files"
```

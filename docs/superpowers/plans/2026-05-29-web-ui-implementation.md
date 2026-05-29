# Web UI Stack Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce Go Fiber + Templ + HTMX + Alpine.js + Tailwind CSS web UI stack with a configs CRUD reference implementation in `internal/modules/web`.

**Architecture:** Standard module pattern (`module.Handler`), routes mounted under `/service/web/*` with auth. Server-rendered HTML via Templ — page handlers return full pages wrapped in `layout.Base`, HTMX handlers return bare partials. Static assets served from `public/` via Fiber Static.

**Tech Stack:** Fiber v3, Templ, HTMX, Alpine.js, Tailwind CSS v4, Ent ORM

**Reference spec:** `docs/superpowers/specs/2026-05-29-web-ui-design.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `package.json` | npm deps for Tailwind CLI + prettier |
| Create | `public/css/input.css` | Tailwind v4 CSS entry |
| Create | `public/js/app.js` | Alpine data, HTMX extensions |
| Create | `pkg/types/model/config.go` | ConfigItem struct for Templ templates |
| Modify | `internal/store/store.go` | Add `ListConfigs` to Adapter interface + `ListConfigOptions` |
| Modify | `internal/store/postgres/adapter.go` | Implement `ListConfigs` |
| Create | `pkg/views/layout/base.templ` | HTML skeleton with `<head>`, nav, scripts |
| Create | `pkg/views/pages/configs.templ` | Config list page (receives pre-fetched data) |
| Create | `pkg/views/partials/config_table.templ` | Config table partial (HTMX target) |
| Create | `pkg/views/partials/config_row.templ` | Single config row (HTMX target) |
| Create | `pkg/views/partials/config_form.templ` | Inline create/edit form with validation errors |
| Create | `internal/modules/web/module.go` | moduleHandler, Register(), Init(), Webservice() |
| Create | `internal/modules/web/webservice.go` | HTTP handlers (page + 7 HTMX routes) |
| Create | `internal/modules/web/module_test.go` | TDD tests for handlers and module init |
| Modify | `internal/modules/fx.go` | Add `web.Register` to fx.Invoke |
| Modify | `taskfile.yaml` | Add `templ`, `css`, `web` tasks |
| Modify | `go.mod` | Add `github.com/a-h/templ` tool directive |

---

### Task 1: Node.js project config and static assets

**Files:**
- Create: `package.json`
- Create: `public/css/input.css`
- Create: `public/js/app.js`

- [ ] **Step 1: Write `package.json`**

```json
{
  "name": "flowbot-web",
  "private": true,
  "devDependencies": {
    "tailwindcss": "^4.0.0",
    "@tailwindcss/cli": "^4.0.0",
    "prettier": "^3.4.0",
    "prettier-plugin-tailwindcss": "^0.6.0"
  }
}
```

- [ ] **Step 2: Write `public/css/input.css`**

```css
@import "tailwindcss";
```

- [ ] **Step 3: Write `public/js/app.js`**

```js
// Alpine.js shared data store
document.addEventListener("alpine:init", () => {
  Alpine.store("app", {
    open: false,
  });
});
```

- [ ] **Step 4: Install npm dependencies**

```bash
npm install
```

Expected: `node_modules/` created, no errors.

- [ ] **Step 5: Verify Tailwind CLI works**

```bash
npx @tailwindcss/cli -i ./public/css/input.css -o ./public/css/styles.css
```

Expected: `public/css/styles.css` created, non-empty.

- [ ] **Step 6: Commit**

```bash
git add package.json package-lock.json public/
git commit -m "feat: add project scaffolding for Tailwind CSS v4 and static assets"
```

---

### Task 2: Data model — ConfigItem

**Files:**
- Create: `pkg/types/model/config.go`

- [ ] **Step 1: Create `pkg/types/model/config.go`**

```go
// Package model provides shared data types for UI views and transport.
package model

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ConfigItem represents a row from the configs database table.
type ConfigItem struct {
	ID        int64    `json:"id"`
	UID       string   `json:"uid"`
	Topic     string   `json:"topic"`
	Key       string   `json:"key"`
	Value     types.KV `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/types/model/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/types/model/config.go
git commit -m "feat: add ConfigItem model for UI views"
```

---

### Task 3: Store layer — ListConfigs

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Add `ListConfigOptions` and `ListConfigs` to the Adapter interface in `store.go`**

After the existing Config methods (line 253), insert:

```go
	// ListConfigs returns config items across all uids/topics with optional search and pagination.
	ListConfigs(ctx context.Context, opts ListConfigOptions) ([]model.ConfigItem, error)
```

Add the options type before the Adapter interface (near other type definitions):

```go
// ListConfigOptions controls pagination and search for ListConfigs.
type ListConfigOptions struct {
	Offset int
	Limit  int
	Search string
}
```

Add the import for `model` at the top of `store.go` imports:

```go
	"github.com/flowline-io/flowbot/pkg/types/model"
```

- [ ] **Step 2: Implement `ListConfigs` in `postgres/adapter.go`**

After the `ConfigDelete` method (line 787), add:

```go
func (a *adapter) ListConfigs(ctx context.Context, opts store.ListConfigOptions) ([]model.ConfigItem, error) {
	q := a.client.ConfigData.Query()
	if opts.Search != "" {
		q = q.Where(
			configdata.Or(
				configdata.UIDContains(opts.Search),
				configdata.TopicContains(opts.Search),
				configdata.KeyContains(opts.Search),
			),
		)
	}
	limit := opts.Limit
	if limit <= 0 || limit > a.maxResults {
		limit = a.maxResults
	}
	items, err := q.
		Offset(opts.Offset).
		Limit(limit).
		Order(gen.Desc(configdata.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: listconfigs: %w", err)
	}
	result := make([]model.ConfigItem, len(items))
	for i, d := range items {
		result[i] = model.ConfigItem{
			ID:        d.ID,
			UID:       d.UID,
			Topic:     d.Topic,
			Key:       d.Key,
			Value:     types.KV(d.Value),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		}
	}
	return result, nil
}
```

Add the import for `model` at the top of `postgres/adapter.go`:

```go
	"github.com/flowline-io/flowbot/pkg/types/model"
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./internal/store/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/store/store.go internal/store/postgres/adapter.go
git commit -m "feat: add ListConfigs to store for config table full-scan with search and pagination"
```

---

### Task 4: Templ layout

**Files:**
- Create: `pkg/views/layout/base.templ`
- Create: `pkg/views/layout/base_templ.go` (generated)

- [ ] **Step 1: Create `pkg/views/layout/base.templ`**

```templ
// Package layout provides the global HTML skeleton for all pages.
package layout

templ Base(title string) {
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<title>{ title }</title>
		<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
		<script src="https://unpkg.com/htmx.org@2.x.x/dist/htmx.js"></script>
		<link href="/static/css/styles.css" rel="stylesheet"/>
	</head>
	<body class="bg-gray-100 min-h-screen">
		<nav class="bg-white shadow-sm border-b border-gray-200">
			<div class="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
				<a href="/service/web/configs" class="font-semibold text-gray-800">Flowbot</a>
				<div class="flex gap-4 text-sm text-gray-600">
					<a href="/service/web/configs" class="hover:text-gray-900">Configs</a>
				</div>
			</div>
		</nav>
		<main class="max-w-7xl mx-auto px-4 py-8">
			{ children... }
		</main>
	</body>
	</html>
}
```

- [ ] **Step 2: Add `templ` tool to go.mod**

Add to the `tool` block in `go.mod`:

```
	github.com/a-h/templ/cmd/templ
```

- [ ] **Step 3: Run `go mod tidy`**

```bash
go mod tidy
```

- [ ] **Step 4: Generate Templ code**

```bash
go tool templ generate
```

Expected: `pkg/views/layout/base_templ.go` created. No errors.

- [ ] **Step 5: Verify compilation**

```bash
go build ./pkg/views/...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum pkg/views/layout/
git commit -m "feat: add Templ layout base with Tailwind, Alpine.js, HTMX CDN"
```

---

### Task 5: Templ partials — config_row, config_form, config_table

**Files:**
- Create: `pkg/views/partials/config_row.templ`
- Create: `pkg/views/partials/config_form.templ`
- Create: `pkg/views/partials/config_table.templ`

- [ ] **Step 1: Create `pkg/views/partials/config_row.templ`**

```templ
// Package partials provides HTMX-targeted partial views.
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ ConfigRow(item model.ConfigItem) {
	<tr id={ configRowID(item) } hx-target="this" class="border-b border-gray-200 hover:bg-gray-50">
		<td class="px-4 py-3 text-sm text-gray-600">{ templ.Sprintf("%d", item.ID) }</td>
		<td class="px-4 py-3 text-sm text-gray-900 font-mono">{ item.UID }</td>
		<td class="px-4 py-3 text-sm text-gray-900">{ item.Topic }</td>
		<td class="px-4 py-3 text-sm text-gray-900 font-mono">{ item.Key }</td>
		<td class="px-4 py-3 text-sm text-gray-500 max-w-xs truncate">{ valuePreview(item.Value) }</td>
		<td class="px-4 py-3 text-sm text-gray-500">{ item.UpdatedAt.Format("2006-01-02 15:04") }</td>
		<td class="px-4 py-3 text-sm">
			<div class="flex gap-2">
				<button hx-get={ configEditURL(item) }
					class="text-blue-600 hover:text-blue-800 font-medium">
					Edit
				</button>
				<button hx-delete={ configKeyURL(item) }
					hx-confirm="Delete this config?"
					class="text-red-600 hover:text-red-800 font-medium">
					Delete
				</button>
			</div>
		</td>
	</tr>
}
```

- [ ] **Step 2: Create `pkg/views/partials/config_form.templ`**

```templ
package partials

import (
	"github.com/flowline-io/flowbot/pkg/types/model"
)

templ ConfigForm(item model.ConfigItem, isNew bool, errors map[string]string) {
	var actionURL string
	if isNew {
		actionURL = "/service/web/configs"
	} else {
		actionURL = configKeyURL(item)
	}

	<tr id={ configFormID(item, isNew) } hx-target="this">
		<td class="px-4 py-2"></td>
		<td class="px-4 py-2">
			<input type="text" name="uid" value={ item.UID }
				class={ "w-full border rounded px-2 py-1 text-sm " + fieldError(errors, "uid") }
				placeholder="uid"/>
			<div class="text-red-500 text-xs">{ errors["uid"] }</div>
		</td>
		<td class="px-4 py-2">
			<input type="text" name="topic" value={ item.Topic }
				class={ "w-full border rounded px-2 py-1 text-sm " + fieldError(errors, "topic") }
				placeholder="topic"/>
			<div class="text-red-500 text-xs">{ errors["topic"] }</div>
		</td>
		<td class="px-4 py-2">
			<input type="text" name="key" value={ item.Key }
				class={ "w-full border rounded px-2 py-1 text-sm " + fieldError(errors, "key") }
				placeholder="key"/>
			<div class="text-red-500 text-xs">{ errors["key"] }</div>
		</td>
		<td class="px-4 py-2">
			<textarea name="value" rows="2"
				class={ "w-full border rounded px-2 py-1 text-sm font-mono " + fieldError(errors, "value") }
				placeholder='{"key": "value"}'>{ valueJSON(item.Value) }</textarea>
			<div class="text-red-500 text-xs">{ errors["value"] }</div>
		</td>
		<td class="px-4 py-2"></td>
		<td class="px-4 py-2">
			<div class="flex gap-2">
				if isNew {
					<button
						hx-post={ actionURL }
						hx-include="closest tr"
						class="bg-blue-600 text-white px-3 py-1 rounded text-sm hover:bg-blue-700">
						Save
					</button>
				} else {
					<button
						hx-put={ actionURL }
						hx-include="closest tr"
						class="bg-blue-600 text-white px-3 py-1 rounded text-sm hover:bg-blue-700">
						Save
					</button>
				}
				<button hx-get={ cancelURL(item, isNew) }
					class="text-gray-600 hover:text-gray-800 text-sm">
					Cancel
				</button>
			</div>
		</td>
	</tr>
}
```

- [ ] **Step 3: Create `pkg/views/partials/config_table.templ`**

```templ
package partials

import "github.com/flowline-io/flowbot/pkg/types/model"

templ ConfigTable(items []model.ConfigItem) {
	<div id="configs-table" class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
		<table class="w-full">
			<thead class="bg-gray-50 border-b border-gray-200">
			<tr>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">UID</th>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Topic</th>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Key</th>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Value</th>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Updated</th>
				<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
			</tr>
			</thead>
			<tbody id="configs-rows">
			for _, item := range items {
				@ConfigRow(item)
			}
			if len(items) == 0 {
				<tr>
					<td colspan="7" class="px-4 py-6 text-center text-sm text-gray-500">No configs found.</td>
				</tr>
			}
			</tbody>
		</table>
	</div>
}
```

- [ ] **Step 4: Create helper Go file `pkg/views/partials/helpers.go`**

```go
package partials

import (
	"fmt"
	"net/url"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func valuePreview(kv types.KV) string {
	b, err := sonic.Marshal(kv)
	if err != nil {
		return "{}"
	}
	s := string(b)
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}

func fieldError(errors map[string]string, field string) string {
	if _, ok := errors[field]; ok {
		return "border-red-500"
	}
	return "border-gray-300"
}

func valueJSON(kv types.KV) string {
	b, err := sonic.Marshal(kv)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// configKeyURL returns the key-based URL for a config item.
func configKeyURL(item model.ConfigItem) string {
	return fmt.Sprintf("/service/web/configs/%s/%s/%s",
		url.PathEscape(item.UID),
		url.PathEscape(item.Topic),
		url.PathEscape(item.Key),
	)
}

// configEditURL returns the edit URL for a config item.
func configEditURL(item model.ConfigItem) string {
	return configKeyURL(item) + "/edit"
}

// configRowID returns the DOM element ID for a config row.
func configRowID(item model.ConfigItem) string {
	return fmt.Sprintf("config-%s-%s-%s",
		url.PathEscape(item.UID),
		url.PathEscape(item.Topic),
		url.PathEscape(item.Key),
	)
}

// configFormID returns the DOM element ID for a config form row.
func configFormID(item model.ConfigItem, isNew bool) string {
	if isNew {
		return "config-form-new"
	}
	return "config-form-" + configRowID(item)
}

func cancelURL(item model.ConfigItem, isNew bool) string {
	if isNew {
		return "/service/web/configs/list"
	}
	return configKeyURL(item)
}
```

- [ ] **Step 6: Generate Templ and verify compilation**

```bash
go tool templ generate && go build ./pkg/views/...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add pkg/views/partials/
git commit -m "feat: add Templ config partials — row, form, table, helpers"
```

---

### Task 6: Templ page — configs page

**Files:**
- Create: `pkg/views/pages/configs.templ`

- [ ] **Step 1: Create `pkg/views/pages/configs.templ`**

```templ
// Package pages provides full-page Templ views.
package pages

import (
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

templ ConfigsPage(items []model.ConfigItem) {
	@layout.Base("Configs — Flowbot") {
		<div class="flex items-center justify-between mb-6">
			<h1 class="text-2xl font-bold text-gray-900">Configs</h1>
			<div class="flex gap-2">
				<button hx-get="/service/web/configs/list"
					hx-target="#configs-table"
					hx-swap="outerHTML"
					class="text-sm text-gray-600 hover:text-gray-900 border rounded px-3 py-1">
					Refresh
				</button>
				<button hx-get="/service/web/configs/new"
					hx-target="#configs-rows"
					hx-swap="afterbegin"
					class="bg-blue-600 text-white px-4 py-2 rounded text-sm hover:bg-blue-700">
					New Config
				</button>
			</div>
		</div>
		@partials.ConfigTable(items)
	}
}
```

- [ ] **Step 2: Generate Templ and verify compilation**

```bash
go tool templ generate && go build ./pkg/views/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/views/pages/
git commit -m "feat: add Templ configs page with server-side pre-rendered data"
```

---

### Task 7: Web module — module.go with tests

**Files:**
- Create: `internal/modules/web/module.go`
- Create: `internal/modules/web/module_test.go`

- [ ] **Step 1: Write the failing test in `module_test.go`**

```go
package web

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "register should not panic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				Register()
			})
		})
	}
}

func TestInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jsonCfg string
		wantErr bool
	}{
		{
			name:    "enabled true succeeds",
			jsonCfg: `{"enabled": true}`,
			wantErr: false,
		},
		{
			name:    "disabled skips initialization",
			jsonCfg: `{"enabled": false}`,
			wantErr: false,
		},
		{
			name:    "invalid json returns error",
			jsonCfg: `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &moduleHandler{}
			err := h.Init(json.RawMessage(tt.jsonCfg))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Reset handler state for subsequent tests
			handler = moduleHandler{}
			config = configType{}
		})
	}
}

func TestIsReady(t *testing.T) {
	tests := []struct {
		name        string
		initialized bool
		want        bool
	}{
		{
			name:        "ready after init",
			initialized: true,
			want:        true,
		},
		{
			name:        "not ready before init",
			initialized: false,
			want:        false,
		},
		{
			name:        "not ready when disabled",
			initialized: false,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler = moduleHandler{initialized: tt.initialized}
			assert.Equal(t, tt.want, handler.IsReady())
			handler = moduleHandler{}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/modules/web/ -v -run TestRegister
```

Expected: FAIL — `undefined: web.Register` (package doesn't exist yet; create the file first).

Actually, let's create the minimal `module.go` first so the test can compile (red phase).

- [ ] **Step 3: Create minimal `internal/modules/web/module.go` to make tests compile**

```go
// Package web provides a web UI module with server-rendered HTML pages.
package web

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
)

const Name = "web"

var handler moduleHandler
var config configType

func Register() {
	module.Register(Name, &handler)
}

type moduleHandler struct {
	initialized bool
	module.Base
}

type configType struct {
	Enabled bool `json:"enabled"`
}

func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	handler.initialized = true
	return nil
}

func (moduleHandler) IsReady() bool {
	return handler.initialized
}

func (moduleHandler) Bootstrap() error {
	return nil
}

func (moduleHandler) Webservice(app *fiber.App) {
	app.Static("/static", "./public")
	module.Webservice(app, Name, webserviceRules)
}

func (moduleHandler) Rules() []any {
	return []any{webserviceRules}
}
```

- [ ] **Step 4: Create the placeholder webservice rules so module.go compiles**

Create `internal/modules/web/webservice.go` with a minimal shell:

```go
package web

import (
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

var webserviceRules = []webservice.Rule{}
```

- [ ] **Step 5: Run tests — they should pass now**

```bash
go test ./internal/modules/web/ -v
```

Expected: all 3 test functions pass.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/web/module.go internal/modules/web/webservice.go internal/modules/web/module_test.go
git commit -m "feat: add web module with Register, Init, and unit tests"
```

---

### Task 8: Web module — webservice handlers with tests

**Files:**
- Modify: `internal/modules/web/webservice.go`
- Modify: `internal/modules/web/module_test.go` (add handler tests)

- [ ] **Step 1: Write handler tests in `module_test.go` (append after existing tests)**

```go
import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

- [ ] **Step 2: Create `internal/modules/web/test_helper_test.go`**

```go
package web

import (
	"context"
	"database/sql"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// testStore implements store.Adapter by embedding the interface and overriding
// only the methods needed for web handler tests.
type testStore struct {
	store.Adapter
	configs     []model.ConfigItem
	configErr   error
	setConfigFn func(uid types.Uid, topic, key string, value types.KV) error
	getConfigFn func(uid types.Uid, topic, key string) (types.KV, error)
	delConfigFn func(uid types.Uid, topic, key string) error
}

func (s *testStore) ListConfigs(_ context.Context, _ store.ListConfigOptions) ([]model.ConfigItem, error) {
	return s.configs, s.configErr
}

func (s *testStore) ConfigSet(_ context.Context, uid types.Uid, topic, key string, value types.KV) error {
	if s.setConfigFn != nil {
		return s.setConfigFn(uid, topic, key, value)
	}
	return nil
}

func (s *testStore) ConfigGet(_ context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	if s.getConfigFn != nil {
		return s.getConfigFn(uid, topic, key)
	}
	return nil, types.ErrNotFound
}

func (s *testStore) ConfigDelete(_ context.Context, uid types.Uid, topic, key string) error {
	if s.delConfigFn != nil {
		return s.delConfigFn(uid, topic, key)
	}
	return nil
}

// Methods required by the Adapter interface that embedded nil won't satisfy at runtime.
func (s *testStore) IsOpen() bool                           { return false }
func (s *testStore) Open(_ store.StoreType) error           { return nil }
func (s *testStore) Close() error                           { return nil }
func (s *testStore) SetMaxResults(_ int)                    {}
func (s *testStore) CreateDb(_ bool) error                  { return nil }
func (s *testStore) UpgradeDb() error                       { return nil }
func (s *testStore) Version() int                           { return 0 }
func (s *testStore) DB() *sql.DB                            { return nil }
func (s *testStore) SetSessCache(_ store.SessCache)         {}
func (s *testStore) SetUidCache(_ store.UidCache)           {}
func (s *testStore) GetName() string                        { return "test" }
func (s *testStore) IsNewNode() bool                        { return false }
func (s *testStore) MaybeUpgradeDb(_ context.Context) error { return nil }

func setupTestApp() (*fiber.App, *testStore) {
	ts := &testStore{}
	store.Database = ts

	app := fiber.New()
	handler = moduleHandler{initialized: true}
	handler.Webservice(app)
	return app, ts
}

// createTestConfig returns a sample ConfigItem for tests.
func createTestConfig(uid, topic, key string) model.ConfigItem {
	return model.ConfigItem{
		ID:     1,
		UID:    uid,
		Topic:  topic,
		Key:    key,
		Value:  types.KV{"v": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
```

Note: `store.Adapter` is embedded as an interface (nil value). Go allows this — any method NOT overridden on `testStore` will panic at runtime if called. The methods we DO override (`ListConfigs`, `ConfigSet`, `ConfigGet`, `ConfigDelete`) and the required interface methods (`IsOpen`, `Open`, `Close`, etc.) are explicitly defined so they never panic. This avoids writing ~100 stub methods. Requires `time` and `database/sql` imports.

- [ ] **Step 3: Write handler test cases in `module_test.go` (append)**

```go
func TestConfigsPage(t *testing.T) {
	tests := []struct {
		name         string
		storeConfigs []model.ConfigItem
		storeErr     error
		wantStatus   int
		wantContains string
	}{
		{
			name:         "renders page with configs",
			storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1")},
			wantStatus:   http.StatusOK,
			wantContains: "k1",
		},
		{
			name:         "renders page with empty list",
			storeConfigs: []model.ConfigItem{},
			wantStatus:   http.StatusOK,
			wantContains: "Configs",
		},
		{
			name:       "store error returns 500",
			storeErr:   fmt.Errorf("db down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			if tt.storeErr != nil {
				ts.configErr = tt.storeErr
				defer func() { ts.configErr = nil }()
			}
			defer func() { store.Database = nil }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/configs", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantContains != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantContains)
			}
		})
	}
}

func TestListConfigs(t *testing.T) {
	tests := []struct {
		name         string
		storeConfigs []model.ConfigItem
		wantStatus   int
		wantContains string
	}{
		{
			name:         "renders config table",
			storeConfigs: []model.ConfigItem{createTestConfig("u1", "t1", "k1")},
			wantStatus:   http.StatusOK,
			wantContains: "k1",
		},
		{
			name:         "renders empty state",
			storeConfigs: []model.ConfigItem{},
			wantStatus:   http.StatusOK,
			wantContains: "No configs",
		},
		{
			name: "renders multiple rows",
			storeConfigs: []model.ConfigItem{
				createTestConfig("u1", "t1", "k1"),
				createTestConfig("u2", "t2", "k2"),
			},
			wantStatus:   http.StatusOK,
			wantContains: "k2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.configs = tt.storeConfigs
			defer func() { store.Database = nil }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/configs/list", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantContains)
		})
	}
}

func TestDeleteConfig(t *testing.T) {
	tests := []struct {
		name       string
		delErr     error
		wantStatus int
	}{
		{
			name:       "delete returns 200 on success",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete returns 500 on store error",
			delErr:     fmt.Errorf("db down"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "delete non-existent config returns 404",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.delErr != nil {
				ts.delConfigFn = func(uid types.Uid, topic, key string) error {
					return tt.delErr
				}
			}
			defer func() { store.Database = nil }()

			req := httptest.NewRequest(http.MethodDelete,
				"/service/web/configs/u1/t1/k1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name         string
		getFn        func(uid types.Uid, topic, key string) (types.KV, error)
		wantStatus   int
		wantContains string
	}{
		{
			name: "existing config returns row",
			getFn: func(uid types.Uid, topic, key string) (types.KV, error) {
				return types.KV{"v": "foo"}, nil
			},
			wantStatus:   http.StatusOK,
			wantContains: `"v":"foo"`,
		},
		{
			name: "not found returns 404",
			getFn: func(uid types.Uid, topic, key string) (types.KV, error) {
				return nil, types.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "store error returns 500",
			getFn: func(uid types.Uid, topic, key string) (types.KV, error) {
				return nil, fmt.Errorf("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			ts.getConfigFn = tt.getFn
			defer func() { store.Database = nil }()

			req := httptest.NewRequest(http.MethodGet,
				"/service/web/configs/u1/t1/k1", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
```

- [ ] **Step 5: Run tests — they should fail (no handlers yet)**

```bash
go test ./internal/modules/web/ -v -run TestConfigsPage
```

Expected: FAIL — `cannot find route` or `404`.

- [ ] **Step 6: Write the full webservice.go (shown above) and module_test.go (shown above) and run all handler tests**

The full webservice.go has correct imports already. The module_test.go import block needs:

```go
import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)
```

And the test_helper_test.go needs: `"context"`, `"database/sql"`, `"net/http"`, `"time"`, `store`, `gen` (if any), `types`, `model`, `fiber`.

- [ ] **Step 7: Verify compilation**

```bash
go build ./internal/modules/web/...
```

Expected: FAIL — `cannot find route` or `404`.

- [ ] **Step 5: Implement webservice handlers in `webservice.go`**

Replace the placeholder file:

```go
package web

import (
	"context"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var webserviceRules = []webservice.Rule{
	webservice.Get("/configs", configsPage),
	webservice.Get("/configs/list", listConfigs),
	webservice.Get("/configs/new", newConfigForm),
	webservice.Post("/configs", createConfig),
	webservice.Get("/configs/{uid}/{topic}/{key}", getConfig),
	webservice.Get("/configs/{uid}/{topic}/{key}/edit", editConfigForm),
	webservice.Put("/configs/{uid}/{topic}/{key}", updateConfig),
	webservice.Delete("/configs/{uid}/{topic}/{key}", deleteConfig),
}

// requireAuth checks auth context; returns nil if OK, error if unauthorized.
func requireAuth(ctx fiber.Ctx) error {
	if route.GetRequestContext(ctx) == nil {
		return ctx.SendStatus(http.StatusUnauthorized)
	}
	return nil
}

func configsPage(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListConfigs(context.Background(), store.ListConfigOptions{Limit: 100})
	if err != nil {
		return types.Errorf(types.ErrInternal, "list configs: %v", err)
	}
	ctx.Type("html")
	return pages.ConfigsPage(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func listConfigs(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	items, err := store.Database.ListConfigs(context.Background(), store.ListConfigOptions{Limit: 100})
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load configs")
	}
	ctx.Type("html")
	return partials.ConfigTable(items).Render(context.Background(), ctx.Response().BodyWriter())
}

func getConfig(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	value, err := store.Database.ConfigGet(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if types.IsNotFound(err) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Config not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load config")
	}
	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{
		UID:   uid,
		Topic: topic,
		Key:   key,
		Value: value,
	}).Render(context.Background(), ctx.Response().BodyWriter())
}

func newConfigForm(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	ctx.Type("html")
	return partials.ConfigForm(model.ConfigItem{}, true, nil).Render(context.Background(), ctx.Response().BodyWriter())
}

func createConfig(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	uid := ctx.FormValue("uid")
	topic := ctx.FormValue("topic")
	key := ctx.FormValue("key")
	valueRaw := ctx.FormValue("value")

	errors := make(map[string]string)
	if uid == "" {
		errors["uid"] = "UID is required"
	}
	if topic == "" {
		errors["topic"] = "Topic is required"
	}
	if key == "" {
		errors["key"] = "Key is required"
	}
	var value types.KV
	if valueRaw != "" {
		if err := sonic.Unmarshal([]byte(valueRaw), &value); err != nil {
			errors["value"] = "Invalid JSON"
		}
	}
	if len(errors) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.ConfigForm(model.ConfigItem{
			UID:   uid,
			Topic: topic,
			Key:   key,
			Value: value,
		}, true, errors).Render(context.Background(), ctx.Response().BodyWriter())
	}

	err := store.Database.ConfigSet(context.Background(), types.Uid(uid), topic, key, value)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to create config")
	}

	// Return the new row using the composite key — no ID=0 problem.
	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{
		UID:   uid,
		Topic: topic,
		Key:   key,
		Value: value,
	}).Render(context.Background(), ctx.Response().BodyWriter())
}

func editConfigForm(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	value, err := store.Database.ConfigGet(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		if types.IsNotFound(err) {
			ctx.Status(http.StatusNotFound)
			return renderError(ctx, "Config not found")
		}
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to load config")
	}
	ctx.Type("html")
	return partials.ConfigForm(model.ConfigItem{
		UID:   uid,
		Topic: topic,
		Key:   key,
		Value: value,
	}, false, nil).Render(context.Background(), ctx.Response().BodyWriter())
}

func updateConfig(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	urlUID, urlTopic, urlKey, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	formUID := ctx.FormValue("uid")
	formTopic := ctx.FormValue("topic")
	formKey := ctx.FormValue("key")
	valueRaw := ctx.FormValue("value")

	errors := make(map[string]string)
	if formUID == "" {
		errors["uid"] = "UID is required"
	}
	if formTopic == "" {
		errors["topic"] = "Topic is required"
	}
	if formKey == "" {
		errors["key"] = "Key is required"
	}
	var value types.KV
	if valueRaw != "" {
		if err := sonic.Unmarshal([]byte(valueRaw), &value); err != nil {
			errors["value"] = "Invalid JSON"
		}
	}
	if len(errors) > 0 {
		ctx.Status(http.StatusUnprocessableEntity)
		ctx.Type("html")
		return partials.ConfigForm(model.ConfigItem{
			UID:   formUID,
			Topic: formTopic,
			Key:   formKey,
			Value: value,
		}, false, errors).Render(context.Background(), ctx.Response().BodyWriter())
	}

	// If composite key changed, delete old record and insert new.
	if formUID != urlUID || formTopic != urlTopic || formKey != urlKey {
		if err := store.Database.ConfigDelete(context.Background(), types.Uid(urlUID), urlTopic, urlKey); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return renderError(ctx, "Failed to update config")
		}
	}

	err = store.Database.ConfigSet(context.Background(), types.Uid(formUID), formTopic, formKey, value)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to update config")
	}

	ctx.Type("html")
	return partials.ConfigRow(model.ConfigItem{
		UID:   formUID,
		Topic: formTopic,
		Key:   formKey,
		Value: value,
	}).Render(context.Background(), ctx.Response().BodyWriter())
}

func deleteConfig(ctx fiber.Ctx) error {
	if err := requireAuth(ctx); err != nil {
		return err
	}
	uid, topic, key, err := decodeConfigParams(ctx)
	if err != nil {
		return err
	}
	err = store.Database.ConfigDelete(context.Background(), types.Uid(uid), topic, key)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return renderError(ctx, "Failed to delete config")
	}
	return ctx.SendStatus(http.StatusOK)
}

// decodeConfigParams extracts and decodes uid, topic, key from URL path params.
func decodeConfigParams(ctx fiber.Ctx) (uid, topic, key string, err error) {
	uid, e1 := url.PathUnescape(ctx.Params("uid"))
	topic, e2 := url.PathUnescape(ctx.Params("topic"))
	key, e3 := url.PathUnescape(ctx.Params("key"))
	if e1 != nil || e2 != nil || e3 != nil {
		return "", "", "", types.Errorf(types.ErrInvalidArgument, "invalid config params")
	}
	if uid == "" || topic == "" || key == "" {
		return "", "", "", types.Errorf(types.ErrInvalidArgument, "uid, topic, and key are required")
	}
	return uid, topic, key, nil
}

// renderError returns an HTML partial with an error message.
func renderError(ctx fiber.Ctx, msg string) error {
	ctx.Type("html")
	_, err := ctx.Write([]byte(`<div class="text-red-500 text-sm py-2">` + msg + `</div>`))
	return err
}
```

- [ ] **Step 8: Run the handler tests**

```bash
go test ./internal/modules/web/ -v -run "TestConfigsPage|TestListConfigs|TestDeleteConfig|TestGetConfig"
```

Expected: All pass.

- [ ] **Step 9: Verify full compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 10: Commit**

```bash
git add internal/modules/web/
git commit -m "feat: add web module handlers for configs CRUD with TDD tests"
```

---

### Task 9: Module registration

**Files:**
- Modify: `internal/modules/fx.go`

- [ ] **Step 1: Add web.Register to `internal/modules/fx.go`**

```go
package modules

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/modules/example"
	"github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/internal/modules/web"
)

var Modules = fx.Options(
	fx.Invoke(
		example.Register,
		hub.Register,
		web.Register,
	),
)
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/modules/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/fx.go
git commit -m "feat: register web module in fx container"
```

---

### Task 10: Build tooling in taskfile.yaml

**Files:**
- Modify: `taskfile.yaml`

- [ ] **Step 1: Add tasks to `taskfile.yaml`**

After the `ent` task (line 216), add:

```yaml
  # Web UI tasks
  templ:
    desc: Generate Go code from Templ templates
    cmds:
      - go tool templ generate

  css:
    desc: Build Tailwind CSS
    cmds:
      - npx @tailwindcss/cli -i ./public/css/input.css -o ./public/css/styles.css

  css:min:
    desc: Build Tailwind CSS minified for production
    cmds:
      - npx @tailwindcss/cli -i ./public/css/input.css -o ./public/css/styles.css --minify

  web:
    desc: Build web UI (Templ + Tailwind)
    cmds:
      - task: templ
      - task: css
```

- [ ] **Step 2: Commit**

```bash
git add taskfile.yaml
git commit -m "feat: add Templ and Tailwind build tasks to taskfile"
```

---

### Task 11: Full build and verification

**Files:**
- None (verification only)

- [ ] **Step 1: Generate Templ and CSS**

```bash
go tool task web
```

Expected: `*_templ.go` files generated, `public/css/styles.css` built, no errors.

- [ ] **Step 2: Full project build**

```bash
go tool task build
```

Expected: `bin/flowbot` built successfully.

- [ ] **Step 3: Run all unit tests**

```bash
go tool task test
```

Expected: all tests pass (existing + new web module tests).

- [ ] **Step 4: Run lint**

```bash
go tool task lint
```

Expected: no new lint violations.

- [ ] **Step 5: Commit if any auto-formatting changes**

```bash
git add -A
git diff --cached --stat
```

If nothing changed (lint was clean), skip.

- [ ] **Step 6: Final commit**

```bash
git commit --allow-empty -m "feat: complete web UI stack — Templ views, configs CRUD, build tooling"
```

---

## Post-Implementation Notes

- The web module is disabled by default. Enable it in `flowbot.yaml`:
  ```yaml
  modules:
    web:
      enabled: true
  ```
- Auth tokens are passed via the standard `X-AccessToken` header or cookie. Without a valid token, routes return 401.
- Tailwind v4 CDN is used in `base.templ` for development. The `styles.css` build is for production (uses `@import "tailwindcss"` in `input.css`).
- Static files are served at `/static/*` from `public/` directory.
- HTML pages are at `/service/web/configs` when module is enabled and auth passes.

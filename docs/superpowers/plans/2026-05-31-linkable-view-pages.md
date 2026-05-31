# Linkable View Pages — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a token-based shareable view page system with five content types (text, markdown, image, pipeline_run, form), stored in a `page_data` table, rendered via templ.

**Architecture:** Ent schema defines `page_data` table. `PageDataStore` in `store.go` handles CRUD + expiry cleanup. `internal/modules/web/view_webservice.go` exposes three routes (`GET /view/{token}`, `POST /view`, `DELETE /view/{token}`) with cookie auth. `internal/modules/web/types.go` maps content types to templ components. Six new `.templ` files under `pkg/views/`. Cleanup goroutine registered via `fx.Invoke`.

**Tech Stack:** Ent (schema + gen), Fiber v3, templ v0.3, Tailwind CSS v4, types.KV, sonic.

---

### Task 1: Ent Schema for `page_data`

**Files:**
- Create: `internal/store/ent/schema/page_data.go`

- [ ] **Step 1: Create the ent schema file**

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PageData struct {
	ent.Schema
}

func (PageData) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("token").NotEmpty().Unique(),
		field.String("type").NotEmpty(),
		field.String("title").Default(""),
		field.JSON("data", map[string]any{}).Optional(),
		field.String("created_by").Default(""),
		field.Time("expires_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (PageData) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token").Unique(),
	}
}

func (PageData) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("page_data"),
	}
}
```

- [ ] **Step 2: Generate ent code**

Run: `go tool task ent`
Expected: Code generated in `internal/store/ent/gen/` including `pagedata/` package.

- [ ] **Step 3: Commit**

```bash
git add internal/store/ent/schema/page_data.go internal/store/ent/gen/
git commit -m "feat: add page_data ent schema and generated code"
```

---

### Task 2: PageDataStore in Store Layer

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add PageDataStore struct and constructor**

Add after the last store struct (before the global `Database` variable, around line 1244 in ResourceChainStore area):

```go
// PageDataStore persists shareable view page data keyed by opaque tokens.
type PageDataStore struct {
	client *gen.Client
}

func NewPageDataStore(client *gen.Client) *PageDataStore {
	return &PageDataStore{client: client}
}
```

- [ ] **Step 2: Add CreatePageData method**

```go
// CreatePageData inserts a new page_data row.
func (s *PageDataStore) CreatePageData(ctx context.Context, token string, pageType string, title string, data types.KV, createdBy string, expiresAt *time.Time) error {
	m := s.client.PageData.Create().
		SetToken(token).
		SetType(pageType).
		SetTitle(title).
		SetCreatedBy(createdBy)
	if len(data) > 0 {
		m.SetData(data)
	}
	if expiresAt != nil {
		m.SetExpiresAt(*expiresAt)
	}
	_, err := m.Save(ctx)
	return err
}
```

- [ ] **Step 3: Add GetPageDataByToken method**

```go
// GetPageDataByToken retrieves a page_data row by token. Returns nil if not found.
func (s *PageDataStore) GetPageDataByToken(ctx context.Context, token string) (*gen.PageData, error) {
	pageData, err := s.client.PageData.Query().
		Where(pagedata.TokenEQ(token)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return pageData, nil
}
```

This requires adding imports: `"github.com/flowline-io/flowbot/internal/store/ent/gen/pagedata"` and `"entgo.io/ent"` and `"time"` and `"github.com/flowline-io/flowbot/pkg/types"` to `store.go` (check existing imports, `time`, `gen`, `ent` are likely already imported — verify).

- [ ] **Step 4: Add DeletePageData method**

```go
// DeletePageData removes a page_data row by token. Returns the number of deleted rows.
func (s *PageDataStore) DeletePageData(ctx context.Context, token string) (int, error) {
	affected, err := s.client.PageData.Delete().
		Where(pagedata.TokenEQ(token)).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}
```

- [ ] **Step 5: Add DeleteExpiredPageData method**

```go
// DeleteExpiredPageData removes rows where expires_at < now(). Returns the number of deleted rows.
func (s *PageDataStore) DeleteExpiredPageData(ctx context.Context) (int64, error) {
	affected, err := s.client.PageData.Delete().
		Where(pagedata.ExpiresAtLT(time.Now())).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int64(affected), nil
}
```

- [ ] **Step 6: Verify compilation**

Run: `go build ./internal/store/...`
Expected: No errors.

- [ ] **Step 7: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add PageDataStore with CRUD and expiry cleanup methods"
```

---

### Task 3: Expired Page Cleanup Cron

**Files:**
- Create: `internal/server/page_data.go`
- Modify: `internal/server/fx.go`

- [ ] **Step 1: Create the cleanup initializer**

Create `internal/server/page_data.go`:

```go
package server

import (
	"context"
	"time"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// initPageDataCleanup starts a background goroutine that periodically deletes
// expired page_data rows.
func initPageDataCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if storepkg.Database == nil || storepkg.Database.GetDB() == nil {
				continue
			}
			client, ok := storepkg.Database.GetDB().(*storepkg.Client)
			if !ok {
				continue
			}
			store := storepkg.NewPageDataStore(client)
			count, err := store.DeleteExpiredPageData(context.Background())
			if err != nil {
				flog.Error("page_data cleanup: %v", err)
			} else if count > 0 {
				flog.Info("page_data cleanup: deleted %d expired rows", count)
			}
		}
	}()
}
```

- [ ] **Step 2: Register in fx.Invoke**

In `internal/server/fx.go`, add `initPageDataCleanup` to the `fx.Invoke` list (after `initPipeline` or any other invoke, e.g., after `profiling.NewProfiler`):

```go
fx.Invoke(
    setServerCacheStore,
    setRouteAuditor,
    handleRoutes,
    handleEvents,
    initPipeline,
    handleModules,
    handlePlatform,
    RunServer,
    profiling.NewProfiler,
    initPageDataCleanup,
),
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/server/...`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add internal/server/page_data.go internal/server/fx.go
git commit -m "feat: add page_data expiry cleanup cron goroutine"
```

---

### Task 4: Type Registry

**Files:**
- Create: `internal/modules/web/types.go`

- [ ] **Step 1: Create the type registry file**

Create `internal/modules/web/types.go`:

```go
package web

import (
	"github.com/a-h/templ"

	"github.com/flowline-io/flowbot/pkg/types"
)

// viewTemplateFn is a function that takes the data payload from page_data
// and returns a templ component for that content type.
type viewTemplateFn func(data types.KV) templ.Component

// viewTemplates maps page_data type strings to their rendering functions.
// Add new entries here when creating new content types.
var viewTemplates = map[string]viewTemplateFn{
	"text":          textView,
	"markdown":      markdownView,
	"image":         imageView,
	"pipeline_run":  pipelineRunView,
	"form":          formView,
}

// textView renders plain text content in a <pre> block.
func textView(data types.KV) templ.Component {
	content, _ := data.String("content")
	return viewTextContent(content)
}

// markdownView renders markdown content. Placeholder — full implementation
// in Task 5 when the templ file is created.
func markdownView(data types.KV) templ.Component {
	content, _ := data.String("content")
	return viewMarkdownContent(content)
}

// imageView renders an image.
func imageView(data types.KV) templ.Component {
	url, _ := data.String("url")
	alt, _ := data.String("alt")
	return viewImageContent(url, alt)
}

// pipelineRunView renders pipeline step run results.
// The handler will pre-fetch step runs and inject them into data.
func pipelineRunView(data types.KV) templ.Component {
	steps, _ := data.Any("steps")
	// steps is injected by the handler after fetching from DB
	return viewPipelineRunContent(steps)
}

// formView renders a read-only form with label-value pairs.
func formView(data types.KV) templ.Component {
	fields, _ := data.List("fields")
	return viewFormContent(fields)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/modules/web/...`
Expected: Error — templ functions not yet defined. This is expected and will resolve in Task 5.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/types.go
git commit -m "feat: add view type registry with five content type stubs"
```

---

### Task 5: Templ Templates — Content Partials

**Files:**
- Create: `pkg/views/partials/view_text.templ`
- Create: `pkg/views/partials/view_markdown.templ`
- Create: `pkg/views/partials/view_image.templ`
- Create: `pkg/views/partials/view_pipeline_run.templ`
- Create: `pkg/views/partials/view_form.templ`
- Create: `pkg/views/partials/view_expired.templ`

- [ ] **Step 1: Create view_text.templ**

```templ
// Package partials provides fragment templates for HTMX responses and shared components.
package partials

templ viewTextContent(content string) {
	<pre class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 whitespace-pre-wrap text-gray-800 text-sm font-mono" data-testid="view-text-content">
		{ content }
	</pre>
}
```

- [ ] **Step 2: Create view_markdown.templ**

```templ
package partials

templ viewMarkdownContent(content string) {
	<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 prose prose-sm max-w-none text-gray-800" data-testid="view-markdown-content">
		{ content }
	</div>
}
```

- [ ] **Step 3: Create view_image.templ**

```templ
package partials

templ viewImageContent(url string, alt string) {
	<div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6" data-testid="view-image-content">
		<img src={ templ.URL(url) } alt={ alt } class="max-w-full h-auto rounded"/>
	</div>
}
```

- [ ] **Step 4: Create view_pipeline_run.templ**

```templ
package partials

import (
	gen "github.com/flowline-io/flowbot/internal/store/ent/gen"
)

templ viewPipelineRunContent(steps any) {
	<div class="bg-white rounded-lg shadow-sm border border-gray-200" data-testid="view-pipeline-run-content">
		if steps == nil {
			<div class="p-6 text-sm text-gray-400">No step run data available.</div>
		} else if stepList, ok := steps.([]*gen.PipelineStepRun); ok {
			@PipelineStepRunsDetail(stepList)
		} else {
			<div class="p-6 text-sm text-gray-400">Unable to render pipeline run data.</div>
		}
	</div>
}
```

- [ ] **Step 5: Create view_form.templ**

```templ
package partials

import "fmt"

templ viewFormContent(fields []any) {
	<div class="bg-white rounded-lg shadow-sm border border-gray-200" data-testid="view-form-content">
		if len(fields) == 0 {
			<div class="p-6 text-sm text-gray-400">No fields to display.</div>
		} else {
			<div class="divide-y divide-gray-100">
				for _, f := range fields {
					<div class="flex px-6 py-4">
						if fm, ok := f.(map[string]any); ok {
							<dt class="w-1/3 text-sm font-medium text-gray-500">{ fmt.Sprint(fm["label"]) }</dt>
							<dd class="w-2/3 text-sm text-gray-800 font-mono">{ fmt.Sprint(fm["value"]) }</dd>
						}
					</div>
				}
			</div>
		}
	</div>
}
```

- [ ] **Step 6: Create view_expired.templ**

```templ
package partials

templ ViewExpiredPage() {
	<div class="text-center py-16" data-testid="view-expired">
		<p class="text-gray-400 text-lg">Page not found or expired.</p>
		<a href="/service/web/home" class="mt-4 inline-block text-blue-600 hover:text-blue-800 text-sm">Return to home</a>
	</div>
}
```

- [ ] **Step 7: Generate templ code**

Run: `go tool task templ`
Expected: Go code generated for all `.templ` files.

- [ ] **Step 8: Verify compilation**

Run: `go build ./pkg/views/... ./internal/modules/web/...`
Expected: No errors (types.go now resolves its template function references).

- [ ] **Step 9: Commit**

```bash
git add pkg/views/partials/view_*.templ pkg/views/partials/view_*_templ.go internal/modules/web/types.go
git commit -m "feat: add five content type templates and expired page template"
```

---

### Task 6: Templ Template — View Page Wrapper

**Files:**
- Create: `pkg/views/pages/view.templ`

- [ ] **Step 1: Create view.templ**

```templ
// Package pages provides full-page templates wrapping content in the global layout.
package pages

import (
	"github.com/a-h/templ"

	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

templ ViewPage(title string, body templ.Component, expired bool) {
	@layout.Base(title + " — Flowbot") {
		if expired {
			<div class="mb-4 bg-yellow-50 border border-yellow-200 rounded-lg px-4 py-3 text-sm text-yellow-700" data-testid="view-expired-banner">
				This page has expired and may no longer be available.
			</div>
		}
		<h1 class="text-2xl font-semibold text-gray-800 mb-6" data-testid="view-title">{ title }</h1>
		{ body }
	}
}
```

- [ ] **Step 2: Generate templ code**

Run: `go tool task templ`
Expected: `pkg/views/pages/view_templ.go` generated.

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/views/...`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/views/pages/view.templ pkg/views/pages/view_templ.go
git commit -m "feat: add view page wrapper template"
```

---

### Task 7: View Handler — Route Handlers

**Files:**
- Create: `internal/modules/web/view_webservice.go`
- Modify: `internal/modules/web/module.go`

- [ ] **Step 1: Create view_webservice.go**

```go
package web

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

var viewWebserviceRules = []webservice.Rule{
	webservice.Get("/view/{token}", viewPage, route.WithNotAuth()),
	webservice.Post("/view", createView, route.WithNotAuth()),
	webservice.Delete("/view/{token}", deleteView, route.WithNotAuth()),
}

// viewPage renders a shareable view page by token.
func viewPage(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	token := ctx.Params("token")
	if token == "" {
		return ctx.Status(http.StatusBadRequest).SendString("missing token")
	}

	client, ok := storepkg.Database.GetDB().(*storepkg.Client)
	if !ok {
		return ctx.Status(http.StatusInternalServerError).SendString("store not available")
	}
	store := storepkg.NewPageDataStore(client)

	pageData, err := store.GetPageDataByToken(context.Background(), token)
	if err != nil {
		flog.Error("viewPage: get page_data: %v", err)
		return ctx.Status(http.StatusInternalServerError).SendString("failed to load page")
	}
	if pageData == nil {
		ctx.Type("html")
		return pages.ViewPage("Not Found", partials.ViewExpiredPage(), false).Render(context.Background(), ctx.Response().BodyWriter())
	}

	// Check expiry
	expired := pageData.ExpiresAt != nil && pageData.ExpiresAt.Before(time.Now())
	if expired {
		ctx.Type("html")
		return pages.ViewPage(pageData.Title, partials.ViewExpiredPage(), true).Render(context.Background(), ctx.Response().BodyWriter())
	}

	dataKV := types.KV(pageData.Data)

	// For pipeline_run type, pre-fetch step runs and inject into data
	if pageData.Type == "pipeline_run" {
		// The view pipeline_run content type expects pre-fetched step runs
		// PipelineStore is used to fetch step runs by run_id
		dataKV = preFetchPipelineData(ctx, storepkg.NewPipelineStore(client), dataKV)
	}

	fn, ok := viewTemplates[pageData.Type]
	if !ok {
		flog.Error("viewPage: unknown type %q", pageData.Type)
		ctx.Type("html")
		return pages.ViewPage(pageData.Title, partials.ViewExpiredPage(), false).Render(context.Background(), ctx.Response().BodyWriter())
	}

	body := fn(dataKV)
	ctx.Type("html")
	return pages.ViewPage(pageData.Title, body, expired).Render(context.Background(), ctx.Response().BodyWriter())
}
```

- [ ] **Step 2: Add preFetchPipelineData helper**

Append to `view_webservice.go`:

```go
// preFetchPipelineData fetches step runs for a pipeline_run view and injects them into data.
func preFetchPipelineData(ctx context.Context, pipeStore *storepkg.PipelineStore, data types.KV) types.KV {
	runID, ok := data.Int64("run_id")
	if !ok {
		return data
	}
	steps, err := pipeStore.GetStepRunsByRunID(ctx, runID)
	if err != nil {
		flog.Error("preFetchPipelineData: GetStepRunsByRunID: %v", err)
		return data
	}
	data["steps"] = steps
	return data
}
```

Requires adding `"time"` to imports in `view_webservice.go`.

- [ ] **Step 3: Add createView handler**

```go
// createView saves a new view page and returns the token and URL.
func createView(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	type createRequest struct {
		Type      string    `json:"type"`
		Title     string    `json:"title"`
		Data      types.KV  `json:"data"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}

	var req createRequest
	if err := sonic.Unmarshal(ctx.Body(), &req); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(types.KV{"error": "invalid JSON: " + err.Error()})
	}
	if req.Type == "" {
		return ctx.Status(http.StatusBadRequest).JSON(types.KV{"error": "type is required"})
	}
	if req.Data == nil {
		req.Data = types.KV{}
	}

	client, ok := storepkg.Database.GetDB().(*storepkg.Client)
	if !ok {
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "store not available"})
	}
	store := storepkg.NewPageDataStore(client)

	token := types.Id()

	rc := route.GetRequestContext(ctx)
	createdBy := ""
	if rc != nil {
		createdBy = string(rc.UID)
	}

	if err := store.CreatePageData(context.Background(), token, req.Type, req.Title, req.Data, createdBy, req.ExpiresAt); err != nil {
		flog.Error("createView: CreatePageData: %v", err)
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "failed to create page"})
	}

	return ctx.Status(http.StatusCreated).JSON(types.KV{
		"token": token,
		"url":   "/service/web/view/" + token,
	})
}
```

Requires adding `"github.com/bytedance/sonic"` to imports.

- [ ] **Step 4: Add deleteView handler**

```go
// deleteView removes a view page by token.
func deleteView(ctx fiber.Ctx) error {
	if err := authenticateWeb(ctx); err != nil {
		return err
	}

	token := ctx.Params("token")
	if token == "" {
		return ctx.Status(http.StatusBadRequest).JSON(types.KV{"error": "missing token"})
	}

	client, ok := storepkg.Database.GetDB().(*storepkg.Client)
	if !ok {
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "store not available"})
	}
	store := storepkg.NewPageDataStore(client)

	affected, err := store.DeletePageData(context.Background(), token)
	if err != nil {
		flog.Error("deleteView: DeletePageData: %v", err)
		return ctx.Status(http.StatusInternalServerError).JSON(types.KV{"error": "failed to delete page"})
	}
	if affected == 0 {
		return ctx.Status(http.StatusNotFound).JSON(types.KV{"error": "page not found"})
	}

	return ctx.Status(http.StatusNoContent).SendString("")
}
```

- [ ] **Step 5: Wire routes in module.go**

In `internal/modules/web/module.go`, modify the `Webservice` method to register the new rules:

```go
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	module.Webservice(app, Name, webserviceRules)
	module.Webservice(app, Name, pipelineWebserviceRules)
	module.Webservice(app, Name, viewWebserviceRules)
}
```

And add `viewWebserviceRules` to `Rules()`:

```go
func (moduleHandler) Rules() []any {
	return []any{webserviceRules, pipelineWebserviceRules, viewWebserviceRules}
}
```

Also update `MountForE2E` (already calls `handler.Webservice(app)` which picks up the new routes automatically).

- [ ] **Step 6: Verify compilation**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 7: Commit**

```bash
git add internal/modules/web/view_webservice.go internal/modules/web/module.go
git commit -m "feat: add view page handlers and route registration"
```

---

### Task 8: Unit Tests — PageDataStore

**Files:**
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Add TestPageDataStore_CreateAndGet tests**

Add after existing test blocks in `store_test.go`. Follows the `getTestClient(t)` + `t.Parallel()` + table-driven pattern:

```go
// ---------------------------------------------------------------------------
// PageDataStore tests
// ---------------------------------------------------------------------------

func TestPageDataStore_CreateAndGet(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	// Seed a page for get/not-found tests
	err := store.CreatePageData(ctx, "seed-token", "text", "Seed", types.KV{"content": "hello"}, "testuser", nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		token    string
		wantNil  bool
		wantType string
		wantErr  bool
	}{
		{
			name:     "retrieve existing page",
			token:    "seed-token",
			wantNil:  false,
			wantType: "text",
			wantErr:  false,
		},
		{
			name:    "retrieve non-existent page",
			token:   "nonexistent-token",
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "retrieve with empty token",
			token:   "",
			wantNil: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pageData, err := store.GetPageDataByToken(ctx, tt.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.wantNil {
				assert.Nil(t, pageData)
			} else {
				assert.NotNil(t, pageData)
				if !tt.wantNil {
					assert.Equal(t, tt.wantType, pageData.Type)
				}
			}
		})
	}
}

func TestPageDataStore_CreateDuplicateToken(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	token := "dup-token"
	err := store.CreatePageData(ctx, token, "text", "First", types.KV{"content": "a"}, "user1", nil)
	require.NoError(t, err)

	err = store.CreatePageData(ctx, token, "text", "Second", types.KV{"content": "b"}, "user2", nil)
	assert.Error(t, err, "duplicate token should return error")
}

func TestPageDataStore_Delete(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	// Create a page to delete
	token := "del-token"
	err := store.CreatePageData(ctx, token, "text", "ToDelete", types.KV{}, "user", nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		token        string
		wantAffected int
		wantErr      bool
	}{
		{
			name:         "delete existing page",
			token:        token,
			wantAffected: 1,
			wantErr:      false,
		},
		{
			name:         "delete already deleted page",
			token:        token,
			wantAffected: 0,
			wantErr:      false,
		},
		{
			name:         "delete non-existent page",
			token:        "no-such-token",
			wantAffected: 0,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			affected, err := store.DeletePageData(ctx, tt.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantAffected, affected)
		})
	}
}

func TestPageDataStore_DeleteExpired(t *testing.T) {
	t.Parallel()

	client := getTestClient(t)
	store := NewPageDataStore(client)
	ctx := context.Background()

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	oneHourLater := time.Now().Add(1 * time.Hour)

	err := store.CreatePageData(ctx, "expired-token", "text", "Expired", types.KV{}, "user", &oneHourAgo)
	require.NoError(t, err)

	err = store.CreatePageData(ctx, "active-token", "text", "Active", types.KV{}, "user", &oneHourLater)
	require.NoError(t, err)

	err = store.CreatePageData(ctx, "no-expiry-token", "text", "Forever", types.KV{}, "user", nil)
	require.NoError(t, err)

	count, err := store.DeleteExpiredPageData(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "only the expired page should be deleted")

	// Verify expired page is gone
	pageData, err := store.GetPageDataByToken(ctx, "expired-token")
	assert.NoError(t, err)
	assert.Nil(t, pageData, "expired page should be deleted")

	// Verify active pages remain
	pageData, err = store.GetPageDataByToken(ctx, "active-token")
	assert.NoError(t, err)
	assert.NotNil(t, pageData, "active page should remain")

	pageData, err = store.GetPageDataByToken(ctx, "no-expiry-token")
	assert.NoError(t, err)
	assert.NotNil(t, pageData, "no-expiry page should remain")
}
```

Requires adding imports to `store_test.go`: `"time"`, `"github.com/stretchr/testify/assert"`, `"github.com/stretchr/testify/require"` (check existing — `require` and `assert` likely already imported; add `"time"` if missing).

- [ ] **Step 2: Run store tests**

Run: `go test ./internal/store/ -run TestPageDataStore -v`
Expected: All four test functions pass.

- [ ] **Step 3: Commit**

```bash
git add internal/store/store_test.go
git commit -m "test: add PageDataStore unit tests"
```

---

### Task 9: Unit Tests — View Handlers

**Files:**
- Create: `internal/modules/web/view_webservice_test.go`
- Modify: `internal/modules/web/test_helper_test.go`

- [ ] **Step 1: Extend testStore for page_data support**

In `internal/modules/web/test_helper_test.go`, the view handler accesses `storepkg.Database.GetDB().(*storepkg.Client)` to create `PageDataStore`. The simplest approach: add a `dbClient *storepkg.Client` field to `testStore` and override `GetDB()`:

Add to `testStore` struct:

```go
type testStore struct {
	storepkg.Adapter

	// existing fields...
	configs     []model.ConfigItem
	configErr   error
	setConfigFn func(uid types.Uid, topic, key string, value types.KV) error
	getConfigFn func(uid types.Uid, topic, key string) (types.KV, error)
	delConfigFn func(uid types.Uid, topic, key string) error
	paramGetFn  func(ctx context.Context, flag string) (gen.Parameter, error)
	paramSetFn  func(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	paramDelFn  func(ctx context.Context, flag string) error

	// page_data test support
	dbClient *storepkg.Client
}

// GetDB overrides the adapter method to return a real ent client for page_data tests.
func (ts *testStore) GetDB() any {
	if ts.dbClient != nil {
		return ts.dbClient
	}
	return nil
}
```

In `setupTestApp()`, add a real in-memory SQLite client for tests that need `PageDataStore`:

```go
func setupTestAppWithDB(t *testing.T) (*fiber.App, *testStore, *storepkg.Client) {
	t.Helper()

	dbClient, err := gen.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed creating schema: %v", err)
	}
	t.Cleanup(func() { dbClient.Close() })

	ts := &testStore{dbClient: dbClient}
	storepkg.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts, dbClient
}
```

Requires adding imports to `test_helper_test.go`: `"context"`, `"database/sql"`, `_ "github.com/mattn/go-sqlite3"`, `"github.com/flowline-io/flowbot/internal/store/ent/gen"`, `storepkg "github.com/flowline-io/flowbot/internal/store"` (verify what's already imported).

- [ ] **Step 2: Create view_webservice_test.go**

```go
package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	storepkg "github.com/flowline-io/flowbot/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewPage_Render(t *testing.T) {
	t.Parallel()

	app, _, dbClient := setupTestAppWithDB(t)

	// Create a page in the DB
	pageStore := storepkg.NewPageDataStore(dbClient)
	token := "render-token"
	err := pageStore.CreatePageData(context.Background(), token, "text", "Test Title", types.KV{"content": "Hello World"}, "user", nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid text page",
			token:      token,
			wantStatus: http.StatusOK,
			wantBody:   "Hello World",
		},
		{
			name:       "non-existent token shows expired page",
			token:      "no-such-token",
			wantStatus: http.StatusOK,
			wantBody:   "Page not found or expired",
		},
		{
			name:       "empty token returns bad request",
			token:      "",
			wantStatus: http.StatusBadRequest,
			wantBody:   "missing token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url := "/service/web/view/" + tt.token
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				body := readBody(resp)
				assert.Contains(t, body, tt.wantBody)
			}
		})
	}
}

func TestViewPage_Unauthenticated(t *testing.T) {
	t.Parallel()

	app, _, _ := setupTestAppWithDB(t)

	req := httptest.NewRequest(http.MethodGet, "/service/web/view/any-token", nil)
	// No cookie set
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, resp.StatusCode) // redirect to login
	loc := resp.Header.Get("Location")
	assert.Contains(t, loc, "/service/web/login")
}

func TestViewPage_ExpiredPage(t *testing.T) {
	t.Parallel()

	app, _, dbClient := setupTestAppWithDB(t)
	pageStore := storepkg.NewPageDataStore(dbClient)

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	token := "expired-render-token"
	err := pageStore.CreatePageData(context.Background(), token, "text", "Expired", types.KV{"content": "stale"}, "user", &oneHourAgo)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/service/web/view/"+token, nil)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := readBody(resp)
	assert.Contains(t, body, "Page not found or expired")
}

func TestCreateView(t *testing.T) {
	t.Parallel()

	app, _, _ := setupTestAppWithDB(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantJSON   string
	}{
		{
			name:       "create text page",
			body:       `{"type":"text","title":"Hello","data":{"content":"world"}}`,
			wantStatus: http.StatusCreated,
			wantJSON:   `"url":"/service/web/view/`,
		},
		{
			name:       "missing type field",
			body:       `{"title":"NoType","data":{}}`,
			wantStatus: http.StatusBadRequest,
			wantJSON:   `"error"`,
		},
		{
			name:       "invalid JSON body",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
			wantJSON:   `"error"`,
		},
		{
			name:       "create with expires_at",
			body:       `{"type":"text","title":"Timed","data":{"content":"x"},"expires_at":"2099-01-01T00:00:00Z"}`,
			wantStatus: http.StatusCreated,
			wantJSON:   `"url":"/service/web/view/`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/service/web/view", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			body := readBody(resp)
			assert.Contains(t, body, tt.wantJSON)
		})
	}
}

func TestDeleteView(t *testing.T) {
	t.Parallel()

	app, _, dbClient := setupTestAppWithDB(t)
	pageStore := storepkg.NewPageDataStore(dbClient)

	token := "del-handler-token"
	err := pageStore.CreatePageData(context.Background(), token, "text", "DeleteMe", types.KV{}, "user", nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "delete existing page",
			token:      token,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete non-existent page",
			token:      "no-such-token",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete with empty token",
			token:      "",
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url := "/service/web/view/" + tt.token
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "test-token"})
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// readBody reads the full response body as a string.
func readBody(resp *http.Response) string {
	body := make([]byte, resp.ContentLength)
	if resp.ContentLength > 0 {
		resp.Body.Read(body)
	}
	resp.Body.Close()
	return string(body)
}
```

Requires imports: `"context"`, `"net/http"`, `"net/http/httptest"`, `"strings"`, `"testing"`, `"time"`, `"github.com/stretchr/testify/assert"`, `"github.com/stretchr/testify/require"`, `"github.com/flowline-io/flowbot/pkg/types"`.

- [ ] **Step 3: Run handler tests**

Run: `go test ./internal/modules/web/ -run "TestViewPage_Render|TestViewPage_Unauthenticated|TestViewPage_ExpiredPage|TestCreateView|TestDeleteView" -v`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/view_webservice_test.go internal/modules/web/test_helper_test.go
git commit -m "test: add view handler unit tests with in-memory SQLite"
```

---

### Task 10: Integration Verification

**Files:**
- No new files. Integration verification via manual testing or BDD spec.

- [ ] **Step 1: Build and run server**

Run: `go tool task build`
Expected: Binary compiles.

- [ ] **Step 2: Run linter**

Run: `go tool task lint`
Expected: No lint errors.

- [ ] **Step 3: Run all unit tests**

Run: `go tool task test`
Expected: All tests pass.

- [ ] **Step 4: Manual smoke test (optional)**

If a running instance is available, curl to verify endpoints:

```bash
# Create a text view
curl -X POST http://localhost:8080/service/web/view \
  -H "Content-Type: application/json" \
  -H "Cookie: accessToken=<valid-token>" \
  -d '{"type":"text","title":"Test","data":{"content":"Hello World"}}'

# Visit the view
curl http://localhost:8080/service/web/view/<returned-token> \
  -H "Cookie: accessToken=<valid-token>"

# Delete the view
curl -X DELETE http://localhost:8080/service/web/view/<returned-token> \
  -H "Cookie: accessToken=<valid-token>"
```

- [ ] **Step 5: Commit final adjustments (if any)**

```bash
git add -A
git commit -m "chore: final adjustments after integration verification"
```

---

### Task 11: BDD Integration Tests

**Files:**
- Create: `tests/view_page_suite_test.go`

- [ ] **Step 1: Create BDD spec**

Create `tests/view_page_suite_test.go` following existing Ginkgo v2 patterns:

```go
package tests

import (
	"net/http"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("View Pages", func() {
	ginkgo.Context("when a user creates a view page", func() {
		ginkgo.It("should return a token and URL", func() {
			// POST /service/web/view with valid payload
			// Expect 201 + token + url in response
		})

		ginkgo.It("should render the page when visiting the token URL", func() {
			// GET /service/web/view/{token}
			// Expect 200 + rendered HTML containing the content
		})

		ginkgo.It("should show expired page for missing token", func() {
			// GET /service/web/view/{nonexistent-token}
			// Expect 200 + "Page not found or expired" in body
		})

		ginkgo.It("should delete a view page", func() {
			// DELETE /service/web/view/{token}
			// Expect 204
			// GET same token → not found page
		})

		ginkgo.It("should redirect unauthenticated users to login", func() {
			// GET /service/web/view/{token} without cookie
			// Expect 302 redirect to /service/web/login
		})
	})
})
```

- [ ] **Step 2: Run BDD spec**

Run: `go tool task test:specs`
Requires Docker for PostgreSQL. Expected: Pass.

- [ ] **Step 3: Commit**

```bash
git add tests/view_page_suite_test.go
git commit -m "test: add BDD integration specs for view pages"
```

---

### Implementation Summary

```
Create:
  internal/store/ent/schema/page_data.go
  internal/server/page_data.go
  internal/modules/web/view_webservice.go
  internal/modules/web/types.go
  internal/modules/web/view_webservice_test.go
  pkg/views/pages/view.templ
  pkg/views/partials/view_text.templ
  pkg/views/partials/view_markdown.templ
  pkg/views/partials/view_image.templ
  pkg/views/partials/view_pipeline_run.templ
  pkg/views/partials/view_form.templ
  pkg/views/partials/view_expired.templ
  tests/view_page_suite_test.go

Modify:
  internal/store/store.go          # + PageDataStore methods
  internal/store/store_test.go     # + PageDataStore tests
  internal/server/fx.go            # + initPageDataCleanup
  internal/modules/web/module.go   # + viewWebserviceRules
  internal/modules/web/test_helper_test.go # + page_data mock fields
```

**Estimated total tasks:** 11
**Estimated total time:** ~2-3 hours

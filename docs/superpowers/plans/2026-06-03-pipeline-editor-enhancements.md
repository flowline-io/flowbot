# Pipeline Editor Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add drag-and-drop step reordering, YAML import/export, version history sidebar, and version diff to the pipeline editor.

**Architecture:** Backend adds a `pipeline_definition_versions` table with Ent schema + two new store methods + two new API endpoints. Frontend uses a custom Alpine drag directive (HTML Drag & Drop API), the `diff` library vendored as a static asset, and extends the existing `pipelineEditor()` Alpine component with sidebar and compare mode state.

**Tech Stack:** Ent ORM, Go 1.26+, Alpine.js, templ, DaisyUI, js-yaml, diff (kpdecker/diff v5), HTML Drag & Drop API

**Spec:** `docs/superpowers/specs/2026-06-03-pipeline-editor-enhancements-design.md`

---

## File Structure

| Action | File | Purpose |
|--------|------|---------|
| Create | `internal/store/ent/schema/pipeline_definition_version.go` | Ent schema for version table |
| Modify | `internal/store/store.go` | `PublishDefinition` insert version; new `ListDefinitionVersions`, `GetDefinitionVersion` |
| Create | `internal/modules/web/pipeline_webservice_test.go` | Handler tests for version endpoints |
| Modify | `internal/modules/web/pipeline_webservice.go` | Two new routes + handlers for version history |
| Create | `public/vendor/diff.min.js` | Vendored diff library (kpdecker/diff v5) |
| Modify | `public/js/pipeline-editor.js` | Drag directive, import/export, version sidebar, diff |
| Modify | `pkg/views/pages/pipeline_editor.templ` | Sidebar markup, import/export buttons, drag attrs |
| Modify | `pkg/views/partials/pipeline_partials.templ` | Remove Move Up/Move Down from StepCard |
| Modify | `internal/store/store_test.go` | Tests for version store methods |

---

### Task 1: Ent Schema for pipeline_definition_versions

**Files:**
- Create: `internal/store/ent/schema/pipeline_definition_version.go`

- [ ] **Step 1: Create Ent schema file**

Write `internal/store/ent/schema/pipeline_definition_version.go`:

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

type PipelineDefinitionVersion struct {
	ent.Schema
}

func (PipelineDefinitionVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("pipeline_name").NotEmpty(),
		field.Int("version"),
		field.Text("yaml").NotEmpty(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (PipelineDefinitionVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("pipeline_name", "version").Unique(),
	}
}

func (PipelineDefinitionVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_definition_versions"),
	}
}
```

- [ ] **Step 2: Run ent code generation**

```bash
go tool task ent
```

Expected: generates `internal/store/ent/gen/pipelinedefinitionversion/` and `internal/store/ent/gen/pipelinedefinitionversion.go` without errors.

- [ ] **Step 3: Verify build compiles**

```bash
go build ./...
```

Expected: no compilation errors.

- [ ] **Step 4: Run format and lint**

```bash
go tool task format
go tool task lint
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/store/ent/schema/pipeline_definition_version.go internal/store/ent/gen/
git commit -m "feat: add pipeline_definition_versions ent schema"
```

---

### Task 2: Store Layer - New Methods + Publish Modification

**Files:**
- Modify: `internal/store/store.go`

- [ ] **Step 1: Add import for pipelinedefinitionversion**

Add to the import block in `internal/store/store.go` (near the other ent gen imports around line 21):

```go
pipelinedefinitionversion "github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinitionversion"
```

- [ ] **Step 2: Add ListDefinitionVersions method**

Insert after `GetDefinitionByName` (around line 1048):

```go
// ListDefinitionVersions returns all published version snapshots for a pipeline,
// ordered by version descending (newest first).
func (s *PipelineStore) ListDefinitionVersions(ctx context.Context, name string) ([]*gen.PipelineDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineDefinitionVersion.Query().
		Where(pipelinedefinitionversion.PipelineName(name)).
		Order(gen.Desc(pipelinedefinitionversion.FieldVersion)).
		All(ctx)
}
```

- [ ] **Step 3: Add GetDefinitionVersion method**

Insert after `ListDefinitionVersions`:

```go
// GetDefinitionVersion returns a single version snapshot by pipeline name and version number.
func (s *PipelineStore) GetDefinitionVersion(ctx context.Context, name string, version int) (*gen.PipelineDefinitionVersion, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.PipelineDefinitionVersion.Query().
		Where(
			pipelinedefinitionversion.PipelineName(name),
			pipelinedefinitionversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}
```

- [ ] **Step 4: Modify PublishDefinition to insert version snapshot**

Replace the entire `PublishDefinition` method (around lines 1084-1114):

```go
// PublishDefinition copies yaml_draft to yaml_published with atomic optimistic locking.
// Also inserts a version snapshot into pipeline_definition_versions.
func (s *PipelineStore) PublishDefinition(ctx context.Context, name string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	// Read current yaml_draft to copy into yaml_published.
	def, err := s.GetDefinitionByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if def.YamlDraft == "" {
		return nil, types.ErrConflict
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
		).
		SetYamlPublished(def.YamlDraft).
		SetVersion(version + 1).
		SetStatus("published").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}

	// Insert version snapshot with the new version number (version + 1).
	if _, err := s.client.PipelineDefinitionVersion.Create().
		SetPipelineName(name).
		SetVersion(version + 1).
		SetYaml(def.YamlDraft).
		SetCreatedAt(time.Now()).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("publish: insert version snapshot: %w", err)
	}

	return s.GetDefinitionByName(ctx, name)
}
```

- [ ] **Step 5: Verify build compiles**

```bash
go build ./...
```

Expected: no compilation errors.

- [ ] **Step 6: Run format and lint**

```bash
go tool task format
go tool task lint
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add ListDefinitionVersions, GetDefinitionVersion, version snapshot on publish"
```

---

### Task 3: Store Layer Tests

**Files:**
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Check existing imports and add missing ones**

The test file already imports `context`, `errors`, `testing`, `time`, `gen`, etc. Add `fmt` to imports if not present (needed for `fmt.Sprintf` in the test).

- [ ] **Step 2: Write version methods test**

Append to `internal/store/store_test.go`, before the file ends:

```go
// ---------------------------------------------------------------------------
// PipelineStore version tests
// ---------------------------------------------------------------------------

func TestPipelineStore_Versions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "publish creates version snapshot"},
		{name: "publish twice stores two versions"},
		{name: "ListDefinitionVersions ordered newest first"},
		{name: "GetDefinitionVersion returns correct YAML"},
		{name: "GetDefinitionVersion not found returns ErrNotFound"},
		{name: "ListDefinitionVersions empty for never-published pipeline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := getTestClient(t)
			store := NewPipelineStore(client)
			ctx := context.Background()

			switch tt.name {
			case "publish creates version snapshot":
				err := store.CreateDefinition(ctx, "vtest-pub1", "")
				require.NoError(t, err)

				err = client.PipelineDefinition.Update().
					Where(pipelinedefinition.Name("vtest-pub1")).
					SetYamlDraft("name: vtest-pub1\nsteps: []").
					Save(ctx)
				require.NoError(t, err)

				_, err = store.PublishDefinition(ctx, "vtest-pub1", 1)
				require.NoError(t, err)

				vers, err := store.ListDefinitionVersions(ctx, "vtest-pub1")
				require.NoError(t, err)
				require.Len(t, vers, 1)
				assert.Equal(t, 2, vers[0].Version)
				assert.Equal(t, "name: vtest-pub1\nsteps: []", vers[0].Yaml)

			case "publish twice stores two versions":
				err := store.CreateDefinition(ctx, "vtest-pub2", "")
				require.NoError(t, err)

				err = client.PipelineDefinition.Update().
					Where(pipelinedefinition.Name("vtest-pub2")).
					SetYamlDraft("name: v1\nsteps: []").
					Save(ctx)
				require.NoError(t, err)

				_, err = store.PublishDefinition(ctx, "vtest-pub2", 1)
				require.NoError(t, err)

				err = client.PipelineDefinition.Update().
					Where(pipelinedefinition.Name("vtest-pub2")).
					SetYamlDraft("name: v2\nsteps: [a]").
					Save(ctx)
				require.NoError(t, err)

				_, err = store.PublishDefinition(ctx, "vtest-pub2", 2)
				require.NoError(t, err)

				vers, err := store.ListDefinitionVersions(ctx, "vtest-pub2")
				require.NoError(t, err)
				require.Len(t, vers, 2)
				assert.Equal(t, 3, vers[0].Version)
				assert.Equal(t, "name: v2\nsteps: [a]", vers[0].Yaml)
				assert.Equal(t, 2, vers[1].Version)
				assert.Equal(t, "name: v1\nsteps: []", vers[1].Yaml)

			case "ListDefinitionVersions ordered newest first":
				err := store.CreateDefinition(ctx, "vtest-order", "")
				require.NoError(t, err)

				for i := range 3 {
					yaml := fmt.Sprintf("name: vtest-order\nsteps: [step%d]", i)
					err = client.PipelineDefinition.Update().
						Where(pipelinedefinition.Name("vtest-order")).
						SetYamlDraft(yaml).
						Save(ctx)
					require.NoError(t, err)

					currentVer := i + 1
					_, err = store.PublishDefinition(ctx, "vtest-order", currentVer)
					require.NoError(t, err)
				}

				vers, err := store.ListDefinitionVersions(ctx, "vtest-order")
				require.NoError(t, err)
				require.Len(t, vers, 3)
				assert.Equal(t, 4, vers[0].Version)
				assert.Equal(t, 3, vers[1].Version)
				assert.Equal(t, 2, vers[2].Version)

			case "GetDefinitionVersion returns correct YAML":
				err := store.CreateDefinition(ctx, "vtest-get", "")
				require.NoError(t, err)

				err = client.PipelineDefinition.Update().
					Where(pipelinedefinition.Name("vtest-get")).
					SetYamlDraft("name: vtest-get\nsteps:\n  - name: s1").
					Save(ctx)
				require.NoError(t, err)

				_, err = store.PublishDefinition(ctx, "vtest-get", 1)
				require.NoError(t, err)

				ver, err := store.GetDefinitionVersion(ctx, "vtest-get", 2)
				require.NoError(t, err)
				assert.Equal(t, 2, ver.Version)
				assert.Equal(t, "name: vtest-get\nsteps:\n  - name: s1", ver.Yaml)

			case "GetDefinitionVersion not found returns ErrNotFound":
				err := store.CreateDefinition(ctx, "vtest-nf", "")
				require.NoError(t, err)

				_, err = store.GetDefinitionVersion(ctx, "vtest-nf", 99)
				assert.True(t, errors.Is(err, types.ErrNotFound))

			case "ListDefinitionVersions empty for never-published pipeline":
				err := store.CreateDefinition(ctx, "vtest-empty", "")
				require.NoError(t, err)

				vers, err := store.ListDefinitionVersions(ctx, "vtest-empty")
				require.NoError(t, err)
				assert.Len(t, vers, 0)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/store/ -run TestPipelineStore_Versions -v -count=1
```

Expected: all 6 subtests pass.

- [ ] **Step 4: Run format and lint**

```bash
go tool task format
go tool task lint
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/store/store_test.go
git commit -m "test: add PipelineStore version methods tests"
```

---

### Task 4: API Layer - Version History Endpoints

**Files:**
- Create: `internal/modules/web/pipeline_webservice_test.go`
- Modify: `internal/modules/web/pipeline_webservice.go`

- [ ] **Step 1: Write handler tests first (TDD)**

Create `internal/modules/web/pipeline_webservice_test.go`:

```go
package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestListPipelineVersions(t *testing.T) {
	tests := []struct {
		name        string
		pipeline    string
		setupStore  func(*testStore)
		wantStatus  int
		wantContain string
	}{
		{
			name:     "empty versions returns empty array",
			pipeline: "test-empty",
			setupStore: func(ts *testStore) {
				ts.versions = nil
			},
			wantStatus:  http.StatusOK,
			wantContain: "[]",
		},
		{
			name:     "returns version list",
			pipeline: "test-with-versions",
			setupStore: func(ts *testStore) {
				ts.versions = []versionItem{
					{Version: 3, Yaml: "name: v3\nsteps: [a]", CreatedAt: "2026-06-03T10:00:00Z"},
					{Version: 2, Yaml: "name: v2\nsteps: []", CreatedAt: "2026-06-03T09:00:00Z"},
				}
			},
			wantStatus:  http.StatusOK,
			wantContain: "version",
		},
		{
			name:     "pipeline not found returns 404",
			pipeline: "not-found",
			setupStore: func(ts *testStore) {
				ts.versionsErr = types.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.setupStore != nil {
				tt.setupStore(ts)
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+tt.pipeline+"/versions", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantContain != "" {
				body, _ := io.ReadAll(resp.Body)
				if !containsAny(string(body), tt.wantContain) {
					t.Errorf("want body containing %q, got %q", tt.wantContain, string(body))
				}
			}
		})
	}
}

func TestGetPipelineVersion(t *testing.T) {
	tests := []struct {
		name        string
		pipeline    string
		version     string
		setupStore  func(*testStore)
		wantStatus  int
		wantContain string
	}{
		{
			name:    "returns version YAML",
			pipeline: "test-get",
			version: "2",
			setupStore: func(ts *testStore) {
				ts.version = &versionItem{Version: 2, Yaml: "name: v2\nsteps:\n  - name: s1", CreatedAt: "2026-06-03T10:00:00Z"}
			},
			wantStatus:  http.StatusOK,
			wantContain: "yaml",
		},
		{
			name:    "version not found returns 404",
			pipeline: "test-nf",
			version: "99",
			setupStore: func(ts *testStore) {
				ts.versionErr = types.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "pipeline not found returns 404",
			pipeline: "bad-pipe",
			version: "1",
			setupStore: func(ts *testStore) {
				ts.versionErr = types.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts := setupTestApp()
			if tt.setupStore != nil {
				tt.setupStore(ts)
			}
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			url := fmt.Sprintf("/service/web/pipelines/%s/versions/%s", tt.pipeline, tt.version)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantContain != "" {
				body, _ := io.ReadAll(resp.Body)
				if !containsAny(string(body), tt.wantContain) {
					t.Errorf("want body containing %q, got %q", tt.wantContain, string(body))
				}
			}
		})
	}
}

func containsAny(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && search(s, substr))
}

func search(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

Note: these tests will fail to compile until we add `versionItem`, `versions`, `versionsErr`, `version`, `versionErr` fields to `testStore` in `test_helper_test.go` — which we do in step 2.

- [ ] **Step 2: Add version mock fields to testStore**

In `internal/modules/web/test_helper_test.go`, add to the `testStore` struct and add helper types:

Add after the existing fields in `testStore`:

```go
	versions       []versionItem
	versionsErr    error
	version        *versionItem
	versionErr     error
```

Add these types at the top of the file (after imports, before `testStore`):

```go
type versionItem struct {
	Version   int    `json:"version"`
	Yaml      string `json:"yaml,omitempty"`
	CreatedAt string `json:"created_at"`
}
```

- [ ] **Step 3: Modify testStore to implement version store methods**

The `testStore` currently doesn't implement `PipelineStore` methods. The handlers use `getPipelineDefStore()` which returns a `*store.PipelineStore`. Since the handlers call `store.NewPipelineStore(client)` where `client` is from `store.Database.GetDB()`, and our testStore wraps with `GetDB()`, we need an approach.

Looking at how other pipeline tests work: they call `getPipelineDefStore()` which does `store.Database.GetDB().(*store.Client)`. With `testStore.dbClient`, this works for DB operations. But for mocking version methods, we need the actual ent client to be able to query.

The simplest approach: the handler tests should use `setupTestAppWithDB` (which uses real in-memory SQLite) rather than mocking the store methods. This way we test actual DB operations. We'll use the real `getPipelineDefStore()` path.

Rewrite the tests in `pipeline_webservice_test.go` to use `setupTestAppWithDB` and seed data directly:

```go
package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/stretchr/testify/require"
)

func TestListPipelineVersions(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   string
		seed       func(*testing.T, context.Context, *store.PipelineStore)
		wantStatus int
		wantBody   string
	}{
		{
			name:     "empty versions returns empty array",
			pipeline: "test-empty-versions",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-empty-versions", ""))
			},
			wantStatus: http.StatusOK,
			wantBody:   "[]",
		},
		{
			name:     "returns version list after publish",
			pipeline: "test-has-versions",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-has-versions", ""))
				// Set draft YAML before publish
				client := store.Database.GetDB().(*store.Client)
				require.NoError(t, client.PipelineDefinition.Update().
					SetYamlDraft("name: test-has-versions\nsteps: [a]").
					Where(pipelinedefinition.Name("test-has-versions")).
					Exec(ctx))
				_, err := s.PublishDefinition(ctx, "test-has-versions", 1)
				require.NoError(t, err)
			},
			wantStatus: http.StatusOK,
			wantBody:   "version",
		},
		{
			name:       "pipeline not found returns 404",
			pipeline:   "nonexistent-zzz",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, ts, client := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			if tt.seed != nil {
				ps := store.NewPipelineStore(client)
				tt.seed(t, context.Background(), ps)
			}

			req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+tt.pipeline+"/versions", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBody) {
					t.Errorf("want body containing %q, got %q", tt.wantBody, string(body))
				}
			}
			_ = ts
		})
	}
}
```

But this requires importing `pipelinedefinition` and `strings`. Let me simplify and create a cleaner test. I'll write the actual test code that uses the real in-memory DB approach.

Actually, the simpler way: the `testStore` wraps `store.Adapter`. The handlers call `getPipelineDefStore()` which gets the underlying ent client from `store.Database.GetDB()`. So with `setupTestAppWithDB`, we get a real ent client. We just need to implement `ListDefinitionVersions` and `GetDefinitionVersion` on `testStore` OR bypass `getPipelineDefStore()`.

Looking at the existing handler code more carefully:
- `getPipelineDefStore()` does `store.Database.GetDB().(*store.Client)` 
- With `setupTestAppWithDB`, `ts.dbClient = dbClient`, so `ts.GetDB()` returns `dbClient`

The handlers then call `s.ListDefinitionVersions(...)` on the `PipelineStore`. So the real store methods will be called on the in-memory SQLite DB. This means we don't need to mock them. The tests just need to work with the real DB.

Let me simplify the test plan significantly. The tests will use `setupTestAppWithDB` and seed data through the real ent client.

Let me rewrite Task 4 with clean, tested code. This is getting complex — let me write the plan section more carefully.

- [ ] **Step 1: Add new routes to pipelineWebserviceRules**

In `internal/modules/web/pipeline_webservice.go`, add to the `pipelineWebserviceRules` slice:

```go
webservice.Get("/pipelines/:name/versions", listPipelineVersions),
webservice.Get("/pipelines/:name/versions/:version", getPipelineVersion),
```

- [ ] **Step 2: Add listPipelineVersions handler**

```go
func listPipelineVersions(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	vers, err := s.ListDefinitionVersions(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "list versions: %v", err)
	}
	items := make([]fiber.Map, 0, len(vers))
	for _, v := range vers {
		items = append(items, fiber.Map{
			"version":    v.Version,
			"created_at": v.CreatedAt,
		})
	}
	return c.JSON(items)
}
```

- [ ] **Step 3: Add getPipelineVersion handler**

```go
func getPipelineVersion(c fiber.Ctx) error {
	name := c.Params("name")
	version, err := strconv.Atoi(c.Params("version"))
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid version: %v", err)
	}
	s := getPipelineDefStore()
	ver, err := s.GetDefinitionVersion(context.Background(), name, version)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get version: %v", err)
	}
	return c.JSON(fiber.Map{
		"yaml":       ver.Yaml,
		"version":    ver.Version,
		"created_at": ver.CreatedAt,
	})
}
```

- [ ] **Step 4: Verify build compiles**

```bash
go build ./...
```

- [ ] **Step 5: Write handler tests with in-memory DB**

Create `internal/modules/web/pipeline_webservice_test.go`:

```go
package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/stretchr/testify/require"
)

func TestListPipelineVersions(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   string
		seed       func(*testing.T, context.Context, *store.PipelineStore, *store.Client)
		wantStatus int
		wantBody   string
	}{
		{
			name:     "empty versions returns empty array",
			pipeline: "test-empty-vers",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, c *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-empty-vers", ""))
			},
			wantStatus: http.StatusOK,
			wantBody:   "[]",
		},
		{
			name:     "returns version list after publish",
			pipeline: "test-has-vers",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, c *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-has-vers", ""))
				require.NoError(t, c.PipelineDefinition.Update().
					SetYamlDraft("name: tv\nsteps:\n  - name: s1").
					Where(pipelinedefinition.Name("test-has-vers")).
					Exec(ctx))
				_, err := s.PublishDefinition(ctx, "test-has-vers", 1)
				require.NoError(t, err)
			},
			wantStatus: http.StatusOK,
			wantBody:   "version",
		},
		{
			name:       "pipeline not found returns 404",
			pipeline:   "no-such-pipeline",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, client := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			if tt.seed != nil {
				ps := store.NewPipelineStore(client)
				tt.seed(t, context.Background(), ps, client)
			}

			req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+tt.pipeline+"/versions", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBody) {
					t.Errorf("want body containing %q, got %q", tt.wantBody, string(body))
				}
			}
		})
	}
}

func TestGetPipelineVersion(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   string
		version    string
		seed       func(*testing.T, context.Context, *store.PipelineStore, *store.Client)
		wantStatus int
		wantBody   string
	}{
		{
			name:    "returns version YAML",
			pipeline: "test-get-vers",
			version: "2",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, c *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-get-vers", ""))
				require.NoError(t, c.PipelineDefinition.Update().
					SetYamlDraft("name: test-get-vers\nsteps:\n  - name: s1").
					Where(pipelinedefinition.Name("test-get-vers")).
					Exec(ctx))
				_, err := s.PublishDefinition(ctx, "test-get-vers", 1)
				require.NoError(t, err)
			},
			wantStatus: http.StatusOK,
			wantBody:   "yaml",
		},
		{
			name:       "version not found returns 404",
			pipeline:   "test-get-nf",
			version:    "99",
			seed: func(t *testing.T, ctx context.Context, s *store.PipelineStore, c *store.Client) {
				t.Helper()
				require.NoError(t, s.CreateDefinition(ctx, "test-get-nf", ""))
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "pipeline not found returns 404",
			pipeline:   "bad-pipe-99",
			version:    "1",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, client := setupTestAppWithDB(t)
			defer func() { store.Database = nil; handler = moduleHandler{}; config = configType{} }()

			if tt.seed != nil {
				ps := store.NewPipelineStore(client)
				tt.seed(t, context.Background(), ps, client)
			}

			req := httptest.NewRequest(http.MethodGet, "/service/web/pipelines/"+tt.pipeline+"/versions/"+tt.version, nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "valid-test-token"})
			resp, _ := app.Test(req)
			defer resp.Body.Close()

			if tt.wantStatus != resp.StatusCode {
				t.Errorf("want status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			if tt.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(body), tt.wantBody) {
					t.Errorf("want body containing %q, got %q", tt.wantBody, string(body))
				}
			}
		})
	}
}
```

- [ ] **Step 6: Run tests and format/lint**

```bash
go test ./internal/modules/web/ -run "TestListPipelineVersions|TestGetPipelineVersion" -v -count=1
go tool task format
go tool task lint
```

- [ ] **Step 7: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go internal/modules/web/pipeline_webservice_test.go
git commit -m "feat: add version history API endpoints"
```

---

### Task 5: Vendor diff.min.js Library

**Files:**
- Create: `public/vendor/diff.min.js`

- [ ] **Step 1: Download diff library**

Download `diff@5` from jsdelivr:

```bash
curl -sL https://cdn.jsdelivr.net/npm/diff@5/dist/diff.min.js -o public/vendor/diff.min.js
```

- [ ] **Step 2: Verify file exists and is non-empty**

```bash
wc -c public/vendor/diff.min.js
```

Expected: file > 5000 bytes.

- [ ] **Step 3: Add script tag to pipeline_editor.templ**

In `pkg/views/pages/pipeline_editor.templ`, add after the `js-yaml` script:

```html
<script src="/static/vendor/diff.min.js"></script>
```

- [ ] **Step 4: Verify build and commit**

```bash
go build ./...
git add public/vendor/diff.min.js pkg/views/pages/pipeline_editor.templ
git commit -m "feat: vendor diff.js library for pipeline version diff"
```

---

### Task 6: Frontend - Drag-and-Drop Step Reordering (Custom Alpine Directive)

**Files:**
- Modify: `public/js/pipeline-editor.js`
- Modify: `pkg/views/pages/pipeline_editor.templ`
- Modify: `pkg/views/partials/pipeline_partials.templ`

- [ ] **Step 1: Register custom Alpine drag directive**

In `pipeline-editor.js`, add a custom directive registration at the top of the `register()` function, before `Alpine.data('pipelineEditor', ...)`:

```javascript
document.addEventListener('alpine:init', () => {
  Alpine.directive('drag-sort', (el, { expression }, { evaluate }) => {
    el.setAttribute('draggable', 'true');

    el.addEventListener('dragstart', (e) => {
      e.dataTransfer.effectAllowed = 'move';
      el.classList.add('opacity-50');
      el.dataset.dragIdx = el.dataset.sortIdx;
    });

    el.addEventListener('dragend', () => {
      el.classList.remove('opacity-50');
      document.querySelectorAll('[data-sort-zone] [data-sort-idx]').forEach((item) => {
        item.classList.remove('border-t-2', 'border-primary');
      });
    });

    el.addEventListener('dragover', (e) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      const target = e.currentTarget;
      document.querySelectorAll('[data-sort-zone] [data-sort-idx]').forEach((item) => {
        item.classList.remove('border-t-2', 'border-primary');
      });
      target.classList.add('border-t-2', 'border-primary');
    });

    el.addEventListener('dragleave', (e) => {
      e.currentTarget.classList.remove('border-t-2', 'border-primary');
    });

    el.addEventListener('drop', (e) => {
      e.preventDefault();
      e.stopPropagation();
      const fromIdx = parseInt(e.dataTransfer.getData('text/plain') || el.dataset.dragIdx, 10);
      const toIdx = parseInt(e.currentTarget.dataset.sortIdx, 10);
      if (isNaN(fromIdx) || isNaN(toIdx) || fromIdx === toIdx) return;

      // Reorder the steps array in Alpine state
      const component = el.closest('[x-data]')._x_dataStack[0];
      const { steps, pushUndo, markDirty, validate } = evaluate(expression)(component);
      if (component) {
        component.pushUndo();
        const item = component.steps.splice(fromIdx, 1)[0];
        component.steps.splice(toIdx, 0, item);
        component.markDirty();
        component.validate();
      }
    });
  });
});
```

Wait, this approach is complex. Let me use a simpler custom directive approach:

Actually, the simplest approach: use a custom Alpine magic `$reorder` or just implement drag-and-drop directly in the Alpine data with helper methods. Let me use a clean approach:

```javascript
// Custom magic: $dragSort
Alpine.magic('dragSort', (el, { Alpine }) => {
  return (listProperty) => {
    const component = Alpine.closestDataStack(el)[0];
    return {
      get list() { return component[listProperty]; },
      dragOver(e) {
        e.preventDefault();
        const target = e.currentTarget.closest('[data-sort-idx]');
        if (!target) return;
        // Clear all highlights
        document.querySelectorAll('[data-sort-zone] [data-sort-idx]').forEach(function(item) {
          item.classList.remove('border-t-2', 'border-primary');
        });
        target.classList.add('border-t-2', 'border-primary');
      },
      dragLeave(e) {
        e.currentTarget.closest('[data-sort-idx]')?.classList.remove('border-t-2', 'border-primary');
      },
      drop(e) {
        e.preventDefault();
        e.currentTarget.closest('[data-sort-idx]')?.classList.remove('border-t-2', 'border-primary');
        const fromIdx = parseInt(e.dataTransfer.getData('text/sort-from'), 10);
        const toIdx = parseInt(e.currentTarget.closest('[data-sort-idx]')?.dataset.sortIdx, 10);
        if (isNaN(fromIdx) || isNaN(toIdx) || fromIdx === toIdx) return;
        component.pushUndo();
        const item = this.list.splice(fromIdx, 1)[0];
        this.list.splice(toIdx, 0, item);
        component.markDirty();
        component.validate();
      },
      dragStart(e) {
        const idx = parseInt(e.currentTarget.closest('[data-sort-idx]')?.dataset.sortIdx, 10);
        e.dataTransfer.setData('text/sort-from', String(idx));
        e.dataTransfer.effectAllowed = 'move';
      },
    };
  };
});
```

Wait, Alpine.magic doesn't quite work like this. Let me use a simpler approach: add methods directly to the `pipelineEditor` data object.

Let me simplify. The most practical approach is:

1. Add `dragFromIdx` state field to the Alpine data
2. Add `onDragStart(idx)`, `onDragOver(idx, e)`, `onDrop(idx, e)` methods
3. Add `draggable="true"` and drag event handlers to step cards
4. Add drop indicator styles

This is the cleanest approach and avoids custom directives entirely. Let me rewrite this task section.

- [ ] **Step 1: Add drag state and methods to pipelineEditor**

In `pipeline-editor.js`, add to the Alpine data (after existing state fields):

```javascript
dragFromIdx: null,
dragOverIdx: null,
```

Add methods to the data object:

```javascript
onStepDragStart(idx, e) {
  this.dragFromIdx = idx;
  e.dataTransfer.effectAllowed = 'move';
  e.dataTransfer.setData('text/plain', String(idx));
  e.target.classList.add('opacity-50');
},

onStepDragEnd(e) {
  this.dragFromIdx = null;
  this.dragOverIdx = null;
  e.target.classList.remove('opacity-50');
  this.$el.querySelectorAll('.drag-over-highlight').forEach(function(el) {
    el.classList.remove('drag-over-highlight', 'border-t-2', 'border-primary');
  });
},

onStepDragOver(idx, e) {
  e.preventDefault();
  if (idx === this.dragFromIdx) return;
  e.dataTransfer.dropEffect = 'move';
  this.dragOverIdx = idx;
  // Highlight the drop target
  const stepEl = e.currentTarget.closest('[data-sort-idx]');
  if (stepEl) {
    this.$el.querySelectorAll('.drag-over-highlight').forEach(function(el) {
      el.classList.remove('drag-over-highlight', 'border-t-2', 'border-primary');
    });
    stepEl.classList.add('drag-over-highlight', 'border-t-2', 'border-primary');
  }
},

onStepDragLeave(e) {
  const stepEl = e.currentTarget.closest('[data-sort-idx]');
  if (stepEl) {
    stepEl.classList.remove('drag-over-highlight', 'border-t-2', 'border-primary');
  }
},

onStepDrop(idx, e) {
  e.preventDefault();
  this.dragOverIdx = null;
  // Clear all highlights
  this.$el.querySelectorAll('.drag-over-highlight').forEach(function(el) {
    el.classList.remove('drag-over-highlight', 'border-t-2', 'border-primary');
  });
  if (this.dragFromIdx === null || this.dragFromIdx === idx) return;

  if (this.dependsOnStep(this.steps[this.dragFromIdx], Math.min(idx, this.dragFromIdx))) {
    showToast('Cannot move: this step depends on data from a step at or above the target position.', 'warning');
    return;
  }

  this.pushUndo();
  const item = this.steps.splice(this.dragFromIdx, 1)[0];
  this.steps.splice(idx, 0, item);
  this.markDirty();
  this.validate();
  this.dragFromIdx = null;
},
```

- [ ] **Step 2: Remove moveStepUp and moveStepDown methods**

Delete the `moveStepUp(idx)` (lines 325-339) and `moveStepDown(idx)` (lines 341-355) methods from `pipeline-editor.js`.

- [ ] **Step 3: Update StepCard template to use drag events**

In `pkg/views/partials/pipeline_partials.templ`, modify the `StepCard` div to add drag attributes:

Replace the opening div:
```go
<div class="card bg-base-100 shadow-sm card-body p-4 relative group cursor-pointer"
```

With:
```go
<div class="card bg-base-100 shadow-sm card-body p-4 relative group cursor-pointer"
  draggable="true"
  :data-sort-idx="idx"
  @dragstart="onStepDragStart(idx, $event)"
  @dragend="onStepDragEnd($event)"
  @dragover="onStepDragOver(idx, $event)"
  @dragleave="onStepDragLeave($event)"
  @drop="onStepDrop(idx, $event)"
```

- [ ] **Step 4: Remove Up/Down buttons from StepCard**

In `pipeline_partials.templ`, remove the Up and Down buttons from the hover toolbar (lines 34-35):

Remove:
```go
<button type="button" @click.stop="moveStepUp(idx)" class="text-base-content/30 hover:text-base-content" data-testid="btn-move-up">Up</button>
<button type="button" @click.stop="moveStepDown(idx)" class="text-base-content/30 hover:text-base-content" data-testid="btn-move-down">Down</button>
```

The toolbar should retain only Copy and Delete buttons.

- [ ] **Step 5: Verify build**

```bash
go tool task format
go build ./...
go test ./internal/store/ -run TestPipelineStore_Versions -count=1
go test ./internal/modules/web/ -run "TestListPipelineVersions|TestGetPipelineVersion" -count=1
```

- [ ] **Step 6: Commit**

```bash
git add public/js/pipeline-editor.js pkg/views/pages/pipeline_editor.templ pkg/views/partials/pipeline_partials.templ
git commit -m "feat: replace move up/down buttons with drag-and-drop step reordering"
```

---

### Task 7: Frontend - YAML Import/Export

**Files:**
- Modify: `public/js/pipeline-editor.js`
- Modify: `pkg/views/pages/pipeline_editor.templ`

- [ ] **Step 1: Add downloadYaml method to pipelineEditor**

In `pipeline-editor.js`, add after existing methods:

```javascript
downloadYaml() {
  const yaml = this.stateToYaml();
  const blob = new Blob([yaml], { type: 'application/x-yaml' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = (this.name || 'pipeline') + '.yaml';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
},

triggerImport() {
  this.$el.querySelector('#yaml-import-input').click();
},

async handleYamlImport(e) {
  const file = e.target.files[0];
  if (!file) return;
  try {
    const text = await new Promise(function(resolve, reject) {
      const reader = new FileReader();
      reader.onload = function(e) { resolve(e.target.result); };
      reader.onerror = function(e) { reject(e); };
      reader.readAsText(file);
    });
    const obj = jsyaml.load(text);
    if (!obj || typeof obj !== 'object') {
      showToast('Invalid YAML: not a pipeline definition', 'error');
      return;
    }
    this.pushUndo();
    this.parseYamlToState(text);
    this.markDirty();
    this.validate();
    showToast('YAML imported successfully', 'success');
  } catch (err) {
    showToast('Import failed: ' + err.message, 'error');
  } finally {
    e.target.value = '';
  }
},
```

- [ ] **Step 2: Add Download and Import buttons to template**

In `pipeline_editor.templ`, in the header toolbar (after the Run History link, before the Save Draft button), add:

```html
<button type="button" @click="downloadYaml"
  class="btn btn-ghost btn-sm"
  data-testid="btn-download-yaml">Download</button>
<button type="button" @click="triggerImport"
  class="btn btn-ghost btn-sm"
  data-testid="btn-import-yaml">Import</button>
```

And add the hidden file input somewhere in the form (after the `</div>` closing the header, before the error summary):

```html
<input type="file" accept=".yaml,.yml" @change="handleYamlImport($event)"
  id="yaml-import-input" class="hidden" data-testid="yaml-import-input">
```

- [ ] **Step 3: Run format, lint, test**

```bash
go tool task format
go tool task lint
go test ./internal/modules/web/ -count=1
```

- [ ] **Step 4: Commit**

```bash
git add public/js/pipeline-editor.js pkg/views/pages/pipeline_editor.templ
git commit -m "feat: add YAML download and import to pipeline editor"
```

---

### Task 8: Frontend - Version History Sidebar

**Files:**
- Modify: `public/js/pipeline-editor.js`
- Modify: `pkg/views/pages/pipeline_editor.templ`

- [ ] **Step 1: Add version history state fields**

In `pipeline-editor.js`, add to the Alpine data state:

```javascript
historyOpen: false,
versions: [],
selectedVersion: null,
selectedVersionYaml: '',
historyLoading: false,
```

- [ ] **Step 2: Add version history methods**

Add methods:

```javascript
async loadVersions() {
  this.historyLoading = true;
  try {
    const resp = await fetch('/service/web/pipelines/' + this.name + '/versions');
    if (!resp.ok) {
      this.versions = [];
      return;
    }
    this.versions = await resp.json();
  } catch (e) {
    console.error('Failed to load versions:', e);
    this.versions = [];
  } finally {
    this.historyLoading = false;
  }
},

toggleHistory() {
  this.historyOpen = !this.historyOpen;
  if (this.historyOpen && this.versions.length === 0) {
    this.loadVersions();
  }
},

async selectVersion(v) {
  this.selectedVersion = v;
  this.historyLoading = true;
  try {
    const resp = await fetch('/service/web/pipelines/' + this.name + '/versions/' + v.version);
    if (!resp.ok) throw new Error('Not found');
    const data = await resp.json();
    this.selectedVersionYaml = data.yaml;
  } catch (e) {
    console.error('Failed to load version:', e);
    this.selectedVersionYaml = '';
  } finally {
    this.historyLoading = false;
  }
},

relativeTime(isoStr) {
  const d = new Date(isoStr);
  const now = new Date();
  const diff = now - d;
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return mins + ' minutes ago';
  const hours = Math.floor(mins / 60);
  if (hours < 24) return hours + ' hours ago';
  const days = Math.floor(hours / 24);
  return days + ' days ago';
},
```

- [ ] **Step 3: Call loadVersions in init**

In the `init()` method, add after the existing init calls:

```javascript
this.loadVersions();
```

- [ ] **Step 4: Add History toggle button to header**

In `pipeline_editor.templ`, in the header toolbar, add before the Code view button:

```html
<button type="button" @click="toggleHistory"
  :class="historyOpen ? 'btn-active' : ''"
  class="btn btn-ghost btn-sm"
  data-testid="btn-history">History</button>
```

- [ ] **Step 5: Add history sidebar to template**

After the `</div>` that closes the main visual/code view area (after line 121, before the drawer backdrop), add:

```html
<!-- Version History Sidebar -->
<div x-show="historyOpen"
  class="fixed right-0 top-0 h-full w-80 bg-base-100 shadow-xl z-30 overflow-y-auto border-l border-base-300"
  data-testid="history-sidebar">
  <div class="p-4">
    <div class="flex items-center justify-between mb-4">
      <h3 class="font-medium text-base-content">Version History</h3>
      <button type="button" @click="historyOpen = false" class="btn btn-ghost btn-sm">&times;</button>
    </div>

    <!-- Version list -->
    <div x-show="!selectedVersion" data-testid="version-list">
      <div x-show="historyLoading" class="text-sm text-base-content/30">Loading...</div>
      <div x-show="!historyLoading && versions.length === 0" class="text-sm text-base-content/30">No published versions yet.</div>
      <template x-for="v in versions" :key="v.version">
        <button type="button" @click="selectVersion(v)"
          class="w-full text-left p-2 rounded-box hover:bg-base-200 mb-1 flex items-center justify-between"
          data-testid="version-item">
          <span class="badge badge-sm" x-text="'v' + v.version"></span>
          <span class="text-xs text-base-content/30" x-text="relativeTime(v.created_at)"></span>
        </button>
      </template>
    </div>

    <!-- Version preview -->
    <div x-show="selectedVersion" data-testid="version-preview">
      <button type="button" @click="selectedVersion = null"
        class="btn btn-ghost btn-sm mb-3">Back to list</button>
      <div class="flex items-center gap-2 mb-2">
        <span class="badge" x-text="'v' + selectedVersion.version"></span>
        <span class="text-xs text-base-content/30" x-text="relativeTime(selectedVersion.created_at)"></span>
      </div>
      <div x-show="historyLoading" class="text-sm text-base-content/30">Loading...</div>
      <pre x-show="!historyLoading && selectedVersionYaml"
        class="text-xs bg-base-200 rounded-box p-3 overflow-x-auto max-h-96 font-mono"
        x-text="selectedVersionYaml"></pre>
      <div x-show="!historyLoading && !selectedVersionYaml" class="text-sm text-base-content/30">No content</div>
    </div>
  </div>
</div>
```

- [ ] **Step 6: Run format, lint, test**

```bash
go tool task format
go tool task lint
go test ./internal/modules/web/ -count=1
```

- [ ] **Step 7: Commit**

```bash
git add public/js/pipeline-editor.js pkg/views/pages/pipeline_editor.templ
git commit -m "feat: add version history sidebar to pipeline editor"
```

---

### Task 9: Frontend - Version Diff

**Files:**
- Modify: `public/js/pipeline-editor.js`
- Modify: `pkg/views/pages/pipeline_editor.templ`

- [ ] **Step 1: Add diff state fields**

In `pipeline-editor.js` Alpine data:

```javascript
compareMode: false,
compareLeft: null,
compareRight: null,
diffResult: null,
```

- [ ] **Step 2: Add diff methods**

```javascript
toggleCompareMode() {
  this.compareMode = !this.compareMode;
  if (!this.compareMode) {
    this.compareLeft = null;
    this.compareRight = null;
    this.diffResult = null;
  }
},

toggleCompareVersion(v) {
  if (this.compareLeft && this.compareLeft.version === v.version) {
    this.compareLeft = null;
  } else if (this.compareRight && this.compareRight.version === v.version) {
    this.compareRight = null;
  } else if (!this.compareLeft) {
    this.compareLeft = v;
  } else if (!this.compareRight) {
    this.compareRight = v;
  }
  if (this.compareLeft && this.compareRight) {
    this.computeDiff();
  }
},

async computeDiff() {
  const left = this.compareLeft;
  const right = this.compareRight;
  // Fetch YAML for both versions
  const fetchYaml = async function(v) {
    const resp = await fetch('/service/web/pipelines/' + this.name + '/versions/' + v.version);
    const data = await resp.json();
    return data.yaml || '';
  }.bind(this);

  try {
    const [leftYaml, rightYaml] = await Promise.all([fetchYaml(left), fetchYaml(right)]);
    const changes = Diff.diffLines(leftYaml || '', rightYaml || '');
    this.diffResult = changes.map(function(part) {
      return {
        text: part.value,
        added: part.added,
        removed: part.removed,
      };
    });
  } catch (e) {
    console.error('Diff error:', e);
    this.diffResult = null;
  }
},
```

- [ ] **Step 3: Add compare mode toggle to sidebar**

In the version history sidebar, in the header section (after the "Version History" title and close button), add:

```html
<button type="button" @click="toggleCompareMode"
  :class="compareMode ? 'btn-primary' : 'btn-ghost'"
  class="btn btn-xs mb-3"
  data-testid="btn-compare">Compare</button>
```

- [ ] **Step 4: Add checkbox and diff rendering to version list items**

Modify the version list item template. Wrap the version button in a div with a checkbox:

In the sidebar template, replace the `x-show="!selectedVersion"` section's `<template x-for...>` with:

```html
<div x-show="!selectedVersion && !compareMode">
  <!-- existing version buttons -->
  <template x-for="v in versions" :key="v.version">
    <button type="button" @click="selectVersion(v)"
      class="w-full text-left p-2 rounded-box hover:bg-base-200 mb-1 flex items-center justify-between"
      data-testid="version-item">
      <span class="badge badge-sm" x-text="'v' + v.version"></span>
      <span class="text-xs text-base-content/30" x-text="relativeTime(v.created_at)"></span>
    </button>
  </template>
</div>

<div x-show="!selectedVersion && compareMode" data-testid="compare-list">
  <p class="text-xs text-base-content/30 mb-2">Select two versions to compare</p>
  <template x-for="v in versions" :key="v.version">
    <div @click="toggleCompareVersion(v)"
      :class="(compareLeft && compareLeft.version === v.version) || (compareRight && compareRight.version === v.version) ? 'bg-primary/10' : ''"
      class="p-2 rounded-box hover:bg-base-200 mb-1 cursor-pointer flex items-center gap-2"
      data-testid="compare-item">
      <span class="text-sm" x-text="'v' + v.version"></span>
      <span class="flex-1 text-xs text-base-content/30" x-text="relativeTime(v.created_at)"></span>
      <span x-show="compareLeft && compareLeft.version === v.version" class="text-xs font-bold text-primary">LEFT</span>
      <span x-show="compareRight && compareRight.version === v.version" class="text-xs font-bold text-primary">RIGHT</span>
    </div>
  </template>

  <!-- Diff result -->
  <div x-show="diffResult" class="mt-4" data-testid="diff-result">
    <div class="text-xs font-medium mb-2">
      <span class="badge badge-sm" x-text="'v' + compareLeft.version"></span>
      vs
      <span class="badge badge-sm" x-text="'v' + compareRight.version"></span>
    </div>
    <pre class="text-xs font-mono bg-base-200 rounded-box p-3 overflow-x-auto max-h-96"><template x-for="chunk in diffResult"><span :class="chunk.added ? 'bg-success/20 text-success' : (chunk.removed ? 'bg-error/20 text-error' : '')" x-text="chunk.text"></span></template></pre>
  </div>
  <div x-show="compareLeft && compareRight && !diffResult" class="text-xs text-base-content/30 mt-4">
    Select two versions to see diff
  </div>
</div>
```

- [ ] **Step 5: Run format, lint, test**

```bash
go tool task format
go tool task lint
go test ./internal/modules/web/ -count=1
go test ./internal/store/ -run TestPipelineStore_Versions -count=1
```

- [ ] **Step 6: Commit**

```bash
git add public/js/pipeline-editor.js pkg/views/pages/pipeline_editor.templ
git commit -m "feat: add version diff comparison to pipeline editor"
```

---

### Task 10: Final Verification

- [ ] **Step 1: Run full test suite**

```bash
go tool task test
```

Expected: all tests pass.

- [ ] **Step 2: Run format and lint**

```bash
go tool task format
go tool task lint
```

Expected: no format changes, no lint errors.

- [ ] **Step 3: Verify build**

```bash
go tool task build
```

Expected: successful build.

- [ ] **Step 4: Commit any remaining changes**

```bash
git status
git diff
# If there are any changes, review and commit
```


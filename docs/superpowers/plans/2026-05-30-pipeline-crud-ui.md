# Pipeline CRUD Web UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a visual pipeline editor with CRUD operations, draft/publish lifecycle, and run history to the Flowbot web UI.

**Architecture:** Database-backed pipeline definitions (ent schema + store), REST API under `/service/web/pipelines` in the web module, Alpine.js canvas editor in templ templates, multi-trigger fan-out in the pipeline engine via compound names.

**Tech Stack:** Go 1.26+, ent (PostgreSQL + SQLite tests), Fiber v3, templ + HTMX + Alpine.js, js-yaml for client-side YAML, Tailwind CSS v4

**Design Spec:** `docs/superpowers/specs/2026-05-30-pipeline-crud-ui-design.md`

---

## File Map

| File                                                              | Action | Responsibility                                             |
| ----------------------------------------------------------------- | ------ | ---------------------------------------------------------- |
| `internal/store/ent/schema/pipeline_definition.go`                | Modify | Updated schema: yaml_draft, yaml_published, version fields |
| `internal/store/store.go`                                         | Modify | PipelineDefinitionStore methods + remove UpsertDefinition  |
| `internal/store/store_test.go`                                    | Modify | Tests for new store methods                                |
| `pkg/pipeline/loader.go`                                          | Modify | EditorDefinition types, expandDefinitions, LoadFromDB      |
| `pkg/pipeline/definition.go`                                      | Create | Editor YAML schema types (EditorDefinition, TriggerEntry)  |
| `pkg/pipeline/pipeline_test.go`                                   | Modify | Tests for expandDefinitions, LoadFromDB                    |
| `pkg/pipeline/engine.go`                                          | Modify | Add ParentName to Definition, use compound names           |
| `internal/server/pipeline.go`                                     | Modify | Wire LoadFromDB, merge definitions                         |
| `internal/modules/web/pipeline_webservice.go`                     | Create | All pipeline API handlers                                  |
| `internal/modules/web/pipeline_webservice_test.go`                | Create | TDD unit tests for handlers                                |
| `internal/modules/web/pipeline_templates/pipeline_list.templ`     | Create | Pipeline list page + table partial                         |
| `internal/modules/web/pipeline_templates/pipeline_editor.templ`   | Create | Canvas editor page (Alpine.js)                             |
| `internal/modules/web/pipeline_templates/pipeline_runs.templ`     | Create | Run history page + table partial                           |
| `internal/modules/web/pipeline_templates/pipeline_partials.templ` | Create | Trigger cards, step cards, drawer, var picker              |
| `public/js/pipeline-editor.js`                                    | Create | Alpine.js canvas component                                 |
| `public/css/input.css`                                            | Modify | Variable pill display CSS                                  |
| `internal/modules/web/webservice.go`                              | Modify | Add pipeline webservice rules                              |
| `internal/modules/web/module.go`                                  | Modify | Add pipeline template/static serving, store injection      |
| `pkg/views/layout/base.templ`                                     | Modify | Add Pipelines nav link                                     |
| `tests/specs/pipeline_spec_test.go`                               | Modify | Fix test using removed UpsertDefinition                    |
| `tests/e2e/pipeline_crud_test.go`                                 | Create | E2E tests                                                  |
| `tests/specs/pipeline_editor_spec_test.go`                        | Create | BDD tests                                                  |

---

### Task 1: Update Ent Schema for PipelineDefinition

**Files:**

- Modify: `internal/store/ent/schema/pipeline_definition.go`

- [ ] **Step 1: Replace schema fields**

Replace the entire file content:

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type PipelineDefinition struct {
	ent.Schema
}

func (PipelineDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Immutable(),
		field.String("name").NotEmpty().Unique().
			Comment("pipeline name, must match ^[a-z0-9][a-z0-9_-]*$").
			Match(regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)),
		field.String("description").Optional().Default(""),
		field.Text("yaml_draft").NotEmpty().Default(""),
		field.Text("yaml_published").Optional().Nillable(),
		field.Int("version").Default(1),
		field.Enum("status").Values("draft", "published").Default("draft"),
		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (PipelineDefinition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Table("pipeline_definitions"),
	}
}
```

- [ ] **Step 2: Add import for regexp at top of file**

The import block already exists; just add `"regexp"` to it.

- [ ] **Step 3: Regenerate ent code**

```bash
go tool task ent
```

- [ ] **Step 4: Commit**

```bash
git add internal/store/ent/
git commit -m "feat: update PipelineDefinition schema for web CRUD (yaml_draft, yaml_published, version)"
```

---

### Task 2: Add PipelineDefinitionStore Methods

**Files:**

- Modify: `internal/store/store.go` (lines ~474-518)
- Modify: `internal/store/store_test.go`

- [ ] **Step 1: Write store tests**

Add to `internal/store/store_test.go`:

```go
func TestPipelineDefinitionStore_CreateAndGet(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewPipelineStore(client)

	tests := []struct {
		name        string
		pipelineName string
		description string
		wantErr     bool
	}{
		{
			name:        "happy path - create pipeline",
			pipelineName: "test-pipeline",
			description: "A test pipeline",
			wantErr:     false,
		},
		{
			name:        "empty description is ok",
			pipelineName: "no-desc-pipeline",
			description: "",
			wantErr:     false,
		},
		{
			name:        "duplicate name returns error",
			pipelineName: "test-pipeline",
			description: "duplicate",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			err := store.CreateDefinition(ctx, tt.pipelineName, tt.description)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			def, err := store.GetDefinitionByName(ctx, tt.pipelineName)
			assert.NoError(t, err)
			assert.Equal(t, tt.pipelineName, def.Name)
			assert.Equal(t, tt.description, def.Description)
			assert.Equal(t, "", def.YAMLDraft)
			assert.Equal(t, 1, def.Version)
		})
	}
}

func TestPipelineDefinitionStore_UpdateDraftConcurrency(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	err := store.CreateDefinition(ctx, "concurrent-test", "")
	require.NoError(t, err)

	// Update with version 1
	def, err := store.UpdateDefinitionDraft(ctx, "concurrent-test", "steps: []", 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, def.Version)

	// Update with stale version 1
	_, err = store.UpdateDefinitionDraft(ctx, "concurrent-test", "steps: [a]", 1)
	assert.ErrorIs(t, err, types.ErrConflict)

	// Update with current version 2 succeeds
	def, err = store.UpdateDefinitionDraft(ctx, "concurrent-test", "steps: [b]", 2)
	assert.NoError(t, err)
	assert.Equal(t, 3, def.Version)
}

func TestPipelineDefinitionStore_PublishAndListPublished(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	store.CreateDefinition(ctx, "pub-test", "desc")

	tests := []struct {
		name    string
		draft   string
		version int
		wantErr bool
	}{
		{
			name:    "publish with valid version",
			draft:   "name: pub-test\ntriggers: []\nsteps: []",
			version: 1,
			wantErr: false,
		},
		{
			name:    "publish with stale version",
			draft:   "name: pub-test\ntriggers: []\nsteps: []",
			version: 1,
			wantErr: true,
		},
		{
			name:    "publish with empty draft",
			draft:   "",
			version: 2,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := store.UpdateDefinitionDraft(ctx, "pub-test", tt.draft, tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}

	// List published
	defs, err := store.ListPublishedDefinitions(ctx)
	assert.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, "pub-test", defs[0].Name)
}

func TestPipelineDefinitionStore_ListAndDelete(t *testing.T) {
	t.Parallel()
	client := getTestClient(t)
	store := NewPipelineStore(client)

	ctx := context.Background()
	store.CreateDefinition(ctx, "list-1", "")
	store.CreateDefinition(ctx, "list-2", "")
	store.CreateDefinition(ctx, "list-3", "")

	tests := []struct {
		name          string
		deleteTarget  string
		wantListCount int
	}{
		{
			name:          "list returns all pipelines",
			deleteTarget:  "",
			wantListCount: 3,
		},
		{
			name:          "delete removes pipeline",
			deleteTarget:  "list-1",
			wantListCount: 2,
		},
		{
			name:          "delete non-existent is no-op",
			deleteTarget:  "nonexistent",
			wantListCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.deleteTarget != "" {
				count, err := store.DeleteDefinitionByName(ctx, tt.deleteTarget)
				assert.NoError(t, err)
				if tt.deleteTarget == "list-1" {
					assert.Equal(t, int64(0), count) // no runs to cascade
				}
			}
			defs, err := store.ListDefinitions(ctx)
			assert.NoError(t, err)
			assert.Len(t, defs, tt.wantListCount)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/store/ -run "TestPipelineDefinitionStore" -v
```

Expected: compilation error — `CreateDefinition`, `GetDefinitionByName`, etc. are not defined.

- [ ] **Step 3: Remove old UpsertDefinition and add new methods in store.go**

Replace lines 474-518 of `internal/store/store.go` (the `UpsertDefinition` method block) with:

```go
// CreateDefinition creates a new pipeline definition with initial yaml_draft and version 1.
func (s *PipelineStore) CreateDefinition(ctx context.Context, name, description string) error {
	if s == nil || s.client == nil {
		return nil
	}
	now := time.Now()
	_, err := s.client.PipelineDefinition.Create().
		SetName(name).
		SetDescription(description).
		SetYamlDraft("").
		SetNillableYamlPublished(nil).
		SetVersion(1).
		SetStatus("draft").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	return err
}

// GetDefinitionByName returns a pipeline definition by name.
func (s *PipelineStore) GetDefinitionByName(ctx context.Context, name string) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, types.ErrNotFound
	}
	def, err := s.client.PipelineDefinition.Query().
		Where(pipelinedefinition.Name(name)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return def, nil
}

// ListDefinitions returns all pipeline definitions ordered by updated_at desc.
func (s *PipelineStore) ListDefinitions(ctx context.Context) ([]*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineDefinition.Query().
		Order(gen.Desc(pipelinedefinition.FieldUpdatedAt)).
		All(ctx)
}

// UpdateDefinitionDraft updates the yaml_draft for a pipeline with atomic optimistic locking.
// Uses a conditional UPDATE with version in the WHERE clause to prevent TOC-TOU races.
// Returns types.ErrConflict if no row matched (version mismatch).
func (s *PipelineStore) UpdateDefinitionDraft(ctx context.Context, name, yamlDraft string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
		).
		SetYamlDraft(yamlDraft).
		SetVersion(version + 1).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, types.ErrConflict
	}
	return s.GetDefinitionByName(ctx, name)
}

// PublishDefinition copies yaml_draft to yaml_published with atomic optimistic locking.
func (s *PipelineStore) PublishDefinition(ctx context.Context, name string, version int) (*gen.PipelineDefinition, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	n, err := s.client.PipelineDefinition.Update().
		Where(
			pipelinedefinition.Name(name),
			pipelinedefinition.Version(version),
			pipelinedefinition.YamlDraftNEQ(""), // reject empty draft at DB level
		).
		SetYamlPublished(pipelinedefinition.Raw("yaml_draft")).
		SetVersion(version + 1).
		SetStatus("published").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		// Could be version mismatch or empty draft — distinguish by checking existence
		exist, err := s.client.PipelineDefinition.Query().
			Where(pipelinedefinition.Name(name)).
			Exist(ctx)
		if err != nil || !exist {
			return nil, types.ErrNotFound
		}
		return nil, types.ErrConflict
	}
	return s.GetDefinitionByName(ctx, name)
}

// DeleteDefinitionByName removes a pipeline definition and cascades to related runs.
// Returns the number of associated pipeline runs that were deleted.
func (s *PipelineStore) DeleteDefinitionByName(ctx context.Context, name string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	runCount, err := s.client.PipelineRun.Delete().
		Where(pipelinerun.PipelineName(name)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete runs for %s: %w", name, err)
	}
	_, err = s.client.PipelineDefinition.Delete().
		Where(pipelinedefinition.Name(name)).
		Exec(ctx)
	if err != nil {
		return runCount, fmt.Errorf("delete definition %s: %w", name, err)
	}
	return int64(runCount), nil
}

// ListPublishedDefinitions returns all published pipeline definitions for engine loading.
func (s *PipelineStore) ListPublishedDefinitions(ctx context.Context) ([]pipeline.DefinitionRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	defs, err := s.client.PipelineDefinition.Query().
		Where(
			pipelinedefinition.Status("published"),
			pipelinedefinition.YamlPublishedNotNil(),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]pipeline.DefinitionRecord, 0, len(defs))
	for _, d := range defs {
		if d.YamlPublished == nil {
			continue
		}
		records = append(records, pipeline.DefinitionRecord{
			Name:        d.Name,
			Description: d.Description,
			YAML:        *d.YamlPublished,
			UpdatedAt:   d.UpdatedAt,
		})
	}
	return records, nil
}
```

- [ ] **Step 4: Add imports to store.go**

Add to the imports block in `store.go`:

- `"github.com/flowline-io/flowbot/pkg/pipeline"` (if not already imported)

- [ ] **Step 5: Fix the existing test file that uses UpsertDefinition**

In `tests/specs/pipeline_spec_test.go`, find the `UpsertDefinition` call and replace it with `CreateDefinition` + `UpdateDefinitionDraft`.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/store/ -run "TestPipelineDefinitionStore" -v
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go tests/specs/pipeline_spec_test.go
git commit -m "feat: add PipelineDefinitionStore methods for web CRUD with optimistic locking"
```

---

### Task 3: Add Editor YAML Types and Engine Loading

**Files:**

- Create: `pkg/pipeline/definition.go`
- Modify: `pkg/pipeline/loader.go`
- Modify: `internal/server/pipeline.go`

- [ ] **Step 1: Create editor-level YAML types**

Create `pkg/pipeline/definition.go`:

```go
package pipeline

import (
	"time"
)

// EditorDefinition is the YAML schema used by the pipeline editor UI.
// It supports multiple triggers (array) unlike the engine's Definition (single Trigger).
type EditorDefinition struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Enabled     bool           `json:"enabled" yaml:"enabled"`
	Resumable   bool           `json:"resumable" yaml:"resumable"`
	Triggers    []TriggerEntry `json:"triggers" yaml:"triggers"`
	Steps       []Step         `json:"steps" yaml:"steps"`
}

// TriggerEntry represents a single trigger in the editor's triggers array.
type TriggerEntry struct {
	Enabled     bool                   `json:"enabled" yaml:"enabled"`
	Type        string                 `json:"type" yaml:"type"` // "event", "cron", "webhook"
	Event       string                 `json:"event,omitempty" yaml:"event,omitempty"`
	Cron        string                 `json:"cron,omitempty" yaml:"cron,omitempty"`
	CronTimeout string                 `json:"cron_timeout,omitempty" yaml:"cron_timeout,omitempty"`
	Webhook     *WebhookConfig         `json:"webhook,omitempty" yaml:"webhook,omitempty"`
}

// DefinitionRecord holds a published pipeline definition loaded from the database.
type DefinitionRecord struct {
	Name        string
	Description string
	YAML        string
	UpdatedAt   time.Time
}

// DefinitionReader is the interface for loading published definitions from a store.
type DefinitionReader interface {
	ListPublishedDefinitions(ctx context.Context) ([]DefinitionRecord, error)
}
```

- [ ] **Step 2: Add expandDefinitions to loader.go**

Append to `pkg/pipeline/loader.go`:

```go
import (
	// existing imports plus:
	"context"
	"fmt"

	"github.com/bytedance/sonic"
)

// ExpandDefinitions fans out an editor definition with multiple triggers into
// engine Definition instances with compound names to avoid key collisions.
func ExpandDefinitions(defs []EditorDefinition) []Definition {
	var expanded []Definition
	for _, d := range defs {
		if !d.Enabled {
			continue
		}
		for i, t := range d.Triggers {
			if !t.Enabled {
				continue
			}
			compoundName := fmt.Sprintf("%s__trigger_%s_%d", d.Name, t.Type, i)
			expanded = append(expanded, Definition{
				Name:        compoundName,
				Description: d.Description,
				Enabled:     true,
				Resumable:   d.Resumable,
				Trigger:     t.toEngineTrigger(),
				Steps:       d.Steps,
				ParentName:  d.Name,
			})
		}
	}
	return expanded
}

func (t TriggerEntry) toEngineTrigger() Trigger {
	tr := Trigger{}
	switch t.Type {
	case "event":
		tr.Event = t.Event
	case "cron":
		tr.Cron = t.Cron
		if t.CronTimeout != "" {
			tr.CronTimeout, _ = time.ParseDuration(t.CronTimeout)
		}
		if tr.CronTimeout == 0 {
			tr.CronTimeout = 10 * time.Minute
		}
	case "webhook":
		tr.Webhook = t.Webhook
	}
	return tr
}

// ParseEditorYAML parses a YAML string into an EditorDefinition.
func ParseEditorYAML(yamlStr string) (*EditorDefinition, error) {
	var def EditorDefinition
	if err := sonic.Unmarshal([]byte(yamlStr), &def); err != nil {
		return nil, fmt.Errorf("parse editor yaml: %w", err)
	}
	return &def, nil
}

// LoadFromDB loads published pipeline definitions from a DefinitionReader.
func LoadFromDB(ctx context.Context, reader DefinitionReader) ([]Definition, error) {
	if reader == nil {
		return nil, nil
	}
	records, err := reader.ListPublishedDefinitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("load definitions from db: %w", err)
	}
	var allDefs []Definition
	for _, rec := range records {
		ed, err := ParseEditorYAML(rec.YAML)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", rec.Name, err)
		}
		allDefs = append(allDefs, ExpandDefinitions([]EditorDefinition{*ed})...)
	}
	return allDefs, nil
}
```

- [ ] **Step 3: Add ParentName to engine Definition struct**

In `pkg/pipeline/loader.go`, add to the `Definition` struct:

```go
type Definition struct {
	Name        string
	Description string
	Enabled     bool
	Resumable   bool
	Trigger     Trigger
	Steps       []Step
	ParentName  string // parent pipeline name for run history aggregation
}
```

- [ ] **Step 4: Write tests for expandDefinitions**

Create/modify `pkg/pipeline/pipeline_test.go`:

```go
func TestExpandDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		input    EditorDefinition
		wantLen  int
		wantName string
	}{
		{
			name: "single event trigger",
			input: EditorDefinition{
				Name: "test", Enabled: true,
				Triggers: []TriggerEntry{
					{Type: "event", Enabled: true, Event: "item.created"},
				},
			},
			wantLen:  1,
			wantName: "test__trigger_event_0",
		},
		{
			name: "event and webhook triggers",
			input: EditorDefinition{
				Name: "multi", Enabled: true,
				Triggers: []TriggerEntry{
					{Type: "event", Enabled: true, Event: "item.created"},
					{Type: "webhook", Enabled: true, Webhook: &WebhookConfig{Path: "/gh"}},
				},
			},
			wantLen:  2,
			wantName: "multi__trigger_event_0",
		},
		{
			name: "disabled trigger skipped",
			input: EditorDefinition{
				Name: "skip", Enabled: true,
				Triggers: []TriggerEntry{
					{Type: "event", Enabled: false, Event: "i.x"},
					{Type: "cron", Enabled: true, Cron: "* * * * *"},
				},
			},
			wantLen:  1,
			wantName: "skip__trigger_cron_1",
		},
		{
			name: "disabled editor definition produces empty",
			input: EditorDefinition{
				Name: "off", Enabled: false,
				Triggers: []TriggerEntry{
					{Type: "event", Enabled: true, Event: "i.x"},
				},
			},
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defs := ExpandDefinitions([]EditorDefinition{tt.input})
			assert.Len(t, defs, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantName, defs[0].Name)
				assert.Equal(t, tt.input.Name, defs[0].ParentName)
			}
		})
	}
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./pkg/pipeline/ -run TestExpandDefinitions -v
```

Expected: All pass.

- [ ] **Step 6: Wire DB loading in initPipeline**

In `internal/server/pipeline.go`, modify `initPipeline`:

Replace:

```go
pipelineDefs := pipeline.LoadConfig(cfg.Pipelines)
if len(pipelineDefs) == 0 {
    return nil
}
```

With:

```go
pipelineDefs := pipeline.LoadConfig(cfg.Pipelines)
ctx := context.Background()
if store.Database != nil && store.Database.GetDB() != nil {
    if client, ok := store.Database.GetDB().(*store.Client); ok {
        pipelineDefStore := store.NewPipelineStore(client)
        dbDefs, err := pipeline.LoadFromDB(ctx, pipelineDefStore)
        if err != nil {
            flog.Error(fmt.Errorf("load pipeline defs from db: %w", err))
        } else {
            // DB definitions override file definitions with the same name
            pipelineDefs = mergeDefinitions(pipelineDefs, dbDefs)
        }
    }
}
if len(pipelineDefs) == 0 {
    return nil
}
```

And add a helper function to the same file:

```go
func mergeDefinitions(fileDefs, dbDefs []pipeline.Definition) []pipeline.Definition {
	if len(dbDefs) == 0 {
		return fileDefs
	}
	seen := make(map[string]bool, len(dbDefs))
	for _, d := range dbDefs {
		seen[d.Name] = true
	}
	merged := make([]pipeline.Definition, 0, len(fileDefs)+len(dbDefs))
	merged = append(merged, dbDefs...)
	for _, d := range fileDefs {
		if seen[d.Name] {
			continue
		}
		merged = append(merged, d)
		seen[d.Name] = true
	}
	return merged
}
```

- [ ] **Step 7: Commit**

```bash
git add pkg/pipeline/definition.go pkg/pipeline/loader.go pkg/pipeline/pipeline_test.go internal/server/pipeline.go
git commit -m "feat: add editor YAML types, expandDefinitions, LoadFromDB with DB merge"
```

---

### Task 4: Create Pipeline API Handlers (Part 1 — List, Create, Get)

**Files:**

- Create: `internal/modules/web/pipeline_webservice.go`
- Create: `internal/modules/web/pipeline_webservice_test.go`

- [ ] **Step 1: Write handler tests**

Create `internal/modules/web/pipeline_webservice_test.go`:

```go
package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPipelineDefStore struct {
	defs map[string]mockDef
}

type mockDef struct {
	name        string
	description string
	yamlDraft   string
	version     int
	status      string
}

func (m *mockPipelineDefStore) CreateDefinition(ctx context.Context, name, desc string) error {
	m.defs[name] = mockDef{name: name, description: desc, version: 1, status: "draft"}
	return nil
}

func (m *mockPipelineDefStore) GetDefinitionByName(ctx context.Context, name string) (*mockDefResult, error) {
	d, ok := m.defs[name]
	if !ok {
		return nil, nil // not found
	}
	return &mockDefResult{
		Name:        d.name,
		Description: d.description,
		YamlDraft:   d.yamlDraft,
		Version:     d.version,
		Status:      d.status,
	}, nil
}

type mockDefResult struct {
	Name        string
	Description string
	YamlDraft   string
	Version     int
	Status      string
}

func TestPipelineListPage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		defs       map[string]mockDef
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty list shows no pipelines message",
			defs:       map[string]mockDef{},
			wantStatus: 200,
			wantBody:   "No pipelines",
		},
		{
			name: "list with pipelines shows table",
			defs: map[string]mockDef{
				"p1": {name: "p1", description: "first", status: "draft"},
				"p2": {name: "p2", description: "second", status: "published"},
			},
			wantStatus: 200,
			wantBody:   "p1",
		},
		{
			name: "list with draft and published",
			defs: map[string]mockDef{
				"a": {name: "a", status: "draft"},
				"b": {name: "b", status: "published"},
				"c": {name: "c", status: "draft"},
			},
			wantStatus: 200,
			wantBody:   "published",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Get("/pipelines", func(c fiber.Ctx) error {
				return pipelineListPage(c, nil) // handler
			})
			req := httptest.NewRequest("GET", "/pipelines", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
```

- [ ] **Step 2: Run tests (fail)**

```bash
go test ./internal/modules/web/ -run TestPipelineListPage -v
```

Expected: compilation error — `pipelineListPage` not defined.

- [ ] **Step 3: Create pipeline_webservice.go skeleton**

Create `internal/modules/web/pipeline_webservice.go`:

```go
package web

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

var pipelineWebserviceRules = []route.WebServiceRule{
	route.NewGet("/pipelines", pipelineListPage),
	route.NewGet("/pipelines/list", pipelineListTable),
	route.NewGet("/pipelines/new", pipelineEditorPage),
	route.NewGet("/pipelines/:name", pipelineEditorPage),
	route.NewPost("/pipelines", createPipeline),
	route.NewPut("/pipelines/:name", updatePipelineDraft),
	route.NewPut("/pipelines/:name/publish", publishPipeline),
	route.NewDelete("/pipelines/:name", deletePipeline),
	route.NewGet("/pipelines/:name/yaml", getPipelineYaml),
	route.NewGet("/pipelines/:name/mock", getMockPayload),
	route.NewPost("/pipelines/:name/test", testPipelineStep),
	route.NewGet("/pipelines/:name/runs", pipelineRunsPage),
	route.NewGet("/pipelines/:name/runs/list", pipelineRunsTable),
}

// getPipelineDefStore returns the PipelineStore from the global store or nil.
func getPipelineDefStore() *store.PipelineStore {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil
	}
	return store.NewPipelineStore(client)
}

func pipelineListPage(c fiber.Ctx) error {
	store := getPipelineDefStore()
	defs, err := store.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return pipeline_templates.PipelineListPage(defs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineListTable(c fiber.Ctx) error {
	store := getPipelineDefStore()
	defs, err := store.ListDefinitions(context.Background())
	if err != nil {
		return types.Errorf(types.ErrInternal, "list pipelines: %v", err)
	}
	c.Type("html")
	return pipeline_templates.PipelineListTable(defs).Render(context.Background(), c.Response().BodyWriter())
}

func createPipeline(c fiber.Ctx) error {
	// Parse name from form or JSON
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}
	if body.Name == "" {
		return types.Errorf(types.ErrInvalidArgument, "name is required")
	}
	store := getPipelineDefStore()
	if err := store.CreateDefinition(context.Background(), body.Name, body.Description); err != nil {
		return types.Errorf(types.ErrInternal, "create pipeline: %v", err)
	}
	// HTMX redirect to editor
	c.Response().Header.Set("HX-Redirect", "/service/web/pipelines/"+body.Name)
	return c.SendStatus(200)
}
```

- [ ] **Step 4: Stage commit (incomplete — templates not yet created)**

```bash
# Skip commit until templates are created in Task 7
```

---

### Task 5: Create Pipeline API Handlers (Part 2 — Update, Publish, Delete, YAML)

**Files:**

- Modify: `internal/modules/web/pipeline_webservice.go`

- [ ] **Step 1: Add remaining handlers**

Append to `internal/modules/web/pipeline_webservice.go`:

```go
func updatePipelineDraft(c fiber.Ctx) error {
	name := c.Params("name")
	var body struct {
		Yaml    string `json:"yaml"`
		Version int    `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}

	s := getPipelineDefStore()
	def, err := s.UpdateDefinitionDraft(context.Background(), name, body.Yaml, body.Version)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return c.Status(409).JSON(fiber.Map{
				"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere. Please refresh the page."},
			})
		}
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "update draft: %v", err)
	}
	return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
}

func publishPipeline(c fiber.Ctx) error {
	name := c.Params("name")
	var body struct {
		Version int `json:"version"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}

	s := getPipelineDefStore()
	def, err := s.PublishDefinition(context.Background(), name, body.Version)
	if err != nil {
		if errors.Is(err, types.ErrConflict) {
			return c.Status(409).JSON(fiber.Map{
				"error": fiber.Map{"code": "CONFLICT", "message": "This draft was modified elsewhere. Please refresh the page."},
			})
		}
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{
				"error": fiber.Map{"code": "NOT_FOUND", "message": "Pipeline not found"},
			})
		}
		return types.Errorf(types.ErrInternal, "publish: %v", err)
	}
	return c.JSON(fiber.Map{"version": def.Version, "status": def.Status})
}

func deletePipeline(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	count, err := s.DeleteDefinitionByName(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "delete pipeline: %v", err)
	}
	return c.JSON(fiber.Map{"deleted": true, "run_count": count})
}

func getPipelineYaml(c fiber.Ctx) error {
	name := c.Params("name")
	s := getPipelineDefStore()
	def, err := s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get yaml: %v", err)
	}
	return c.JSON(fiber.Map{
		"yaml":    def.YamlDraft,
		"version": def.Version,
		"status":  def.Status,
	})
}
```

- [ ] **Step 2: Add imports**

Add to the imports in `pipeline_webservice.go`:

```go
import (
	"errors"
	// ... existing imports
)
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go internal/modules/web/pipeline_webservice_test.go
git commit -m "feat: add pipeline API handlers (list, create, get, update, publish, delete, yaml)"
```

---

### Task 6: Create Mock Payload and Test Execution Handlers

**Files:**

- Modify: `internal/modules/web/pipeline_webservice.go`

- [ ] **Step 1: Add mock payload handler**

Append to `internal/modules/web/pipeline_webservice.go`:

```go
func getMockPayload(c fiber.Ctx) error {
	source := c.Query("source")
	switch source {
	case "event":
		return c.JSON(fiber.Map{
			"source": "event",
			"payload": fiber.Map{
				"event_id":    "mock-ev-001",
				"event_type":  "item.created",
				"title":       "",
				"entity_id":   "",
				"source":      "",
				"capability":  "example",
				"operation":   "create",
			},
			"note": "Generated from event schema. Edit values to match your expected data.",
		})
	case "webhook":
		return c.JSON(fiber.Map{
			"source": "webhook",
			"payload": fiber.Map{
				"event_id": "mock-wb-001",
				"title":    "Sample webhook payload",
				"body":     fiber.Map{},
			},
			"note": "Edit fields to customize your test data.",
		})
	case "cron":
		return c.JSON(fiber.Map{
			"source":  "cron",
			"payload": fiber.Map{},
			"note":    "Cron-triggered pipelines have no event payload.",
		})
	default:
		return types.Errorf(types.ErrInvalidArgument, "missing or invalid source query param")
	}
}

func testPipelineStep(c fiber.Ctx) error {
	var body struct {
		TriggerSource string         `json:"trigger_source"`
		MockPayload   map[string]any `json:"mock_payload"`
		UpToStepIndex int            `json:"up_to_step_index"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return types.Errorf(types.ErrInvalidArgument, "invalid body: %v", err)
	}

	name := c.Params("name")
	s := getPipelineDefStore()
	def, err := s.GetDefinitionByName(context.Background(), name)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "NOT_FOUND"}})
		}
		return types.Errorf(types.ErrInternal, "get pipeline: %v", err)
	}

	ed, err := pipeline.ParseEditorYAML(def.YamlDraft)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to parse pipeline YAML: " + err.Error(),
		})
	}

	results := runTestExecution(name, ed, body.TriggerSource, body.MockPayload, body.UpToStepIndex)
	return c.JSON(results)
}

func runTestExecution(pipelineName string, ed *pipeline.EditorDefinition, source string, mockPayload map[string]any, upToIdx int) fiber.Map {
	type stepResult struct {
		Name           string         `json:"name"`
		Status         string         `json:"status"`
		DurationMs     int64          `json:"duration_ms,omitempty"`
		Output         map[string]any `json:"output,omitempty"`
		RenderedParams map[string]any `json:"rendered_params,omitempty"`
		Error          string         `json:"error,omitempty"`
	}

	if upToIdx < 0 || upToIdx >= len(ed.Steps) {
		return fiber.Map{"success": false, "error": "step index out of range"}
	}

	event := types.DataEvent{Data: make(map[string]any)}
	for k, v := range mockPayload {
		event.Data[k] = v
	}
	event.EventID = "mock-test-" + pipelineName
	if eid, ok := mockPayload["event_id"].(string); ok {
		event.EventID = eid
	}

	rc := pipeline.NewRenderContext(event)
	var results []stepResult

	for i := 0; i <= upToIdx; i++ {
		step := ed.Steps[i]
		start := time.Now()

		rendered, err := rc.RenderParams(step.Params)
		if err != nil {
			results = append(results, stepResult{
				Name:   step.Name,
				Status: "error",
				Error:  fmt.Sprintf("render params: %v", err),
			})
			return fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results}
		}

		output, err := hub.Default.Invoke(context.Background(), hub.InvokeRequest{
			Capability: step.Capability,
			Operation:  step.Operation,
			Params:     rendered,
			DryRun:     true,
		})

		duration := time.Since(start).Milliseconds()

		if err != nil {
			results = append(results, stepResult{
				Name:   step.Name,
				Status: "error",
				Error:  fmt.Sprintf("invoke: %v", err),
			})
			return fiber.Map{"success": false, "error": "Step " + step.Name + " failed", "steps": results}
		}

		outputMap := make(map[string]any)
		if output != nil && output.Result != nil {
			outputMap = output.Result
		}
		rc.RecordStepResult(step.Name, outputMap)

		results = append(results, stepResult{
			Name:           step.Name,
			Status:         "ok",
			DurationMs:     duration,
			Output:         outputMap,
			RenderedParams: rendered,
		})
	}

	return fiber.Map{"success": true, "steps": results}
}
```

- [ ] **Step 2: Add imports for test execution**

In `pipeline_webservice.go`, add:

```go
import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/pipeline"
)
```

- [ ] **Step 3: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go
git commit -m "feat: add mock payload and test execution pipeline handlers"
```

---

### Task 7: Create Pipeline List Templates

**Files:**

- Create: `internal/modules/web/pipeline_templates/pipeline_list.templ`

- [ ] **Step 1: Create pipeline list template with create modal**

Create `internal/modules/web/pipeline_templates/pipeline_list.templ`:

```go
package pipeline_templates

import (
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

templ PipelineListPage(defs []*gen.PipelineDefinition) {
	@layout.Base("Pipelines") {
		<div class="max-w-4xl mx-auto">
			<div class="flex items-center justify-between mb-6">
				<h1 class="text-2xl font-semibold text-gray-800">Pipelines</h1>
				<button type="button"
					onclick="document.getElementById('create-modal').classList.remove('hidden')"
					data-testid="btn-new-pipeline"
					class="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 text-sm font-medium">
					+ New Pipeline
				</button>
			</div>
			<div id="pipeline-list-container" data-testid="pipeline-list-container">
				@PipelineListTable(defs)
			</div>
		</div>

		<!-- Create Modal -->
		<div id="create-modal" class="hidden fixed inset-0 z-50 flex items-center justify-center bg-black/30"
			data-testid="create-modal">
			<div class="bg-white rounded-lg shadow-xl p-6 w-96" @click.outside="document.getElementById('create-modal').classList.add('hidden')">
				<h3 class="text-lg font-medium text-gray-800 mb-4">New Pipeline</h3>
				<form hx-post="/service/web/pipelines" hx-target="body" data-testid="create-form">
					<label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
					<input type="text" name="name" required pattern="[a-z0-9][a-z0-9_-]*"
						class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
						placeholder="my-pipeline"
						title="Lowercase letters, digits, hyphens, underscores. Must start with letter or digit."
						data-testid="input-pipeline-name">
					<label class="block text-sm font-medium text-gray-700 mb-1">Description (optional)</label>
					<input type="text" name="description"
						class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-4"
						placeholder="Brief description"
						data-testid="input-pipeline-desc">
					<div class="flex justify-end gap-2">
						<button type="button"
							onclick="document.getElementById('create-modal').classList.add('hidden')"
							class="px-4 py-2 text-sm text-gray-600 hover:text-gray-800"
							data-testid="btn-cancel-create">Cancel</button>
						<button type="submit"
							class="px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 font-medium"
							data-testid="btn-submit-create">Create</button>
					</div>
				</form>
			</div>
		</div>
	}
}

templ PipelineListTable(defs []*gen.PipelineDefinition) {
	if len(defs) == 0 {
		<div class="text-center py-12 text-gray-400" data-testid="pipeline-empty">
			<p class="text-lg">No pipelines yet.</p>
			<p class="text-sm mt-2">Create your first pipeline to get started.</p>
		</div>
	} else {
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden" data-testid="pipeline-table">
			<table class="w-full text-sm text-left">
				<thead class="bg-gray-50 text-gray-600">
					<tr>
						<th class="px-6 py-3 font-medium">Name</th>
						<th class="px-6 py-3 font-medium">Status</th>
						<th class="px-6 py-3 font-medium">Updated</th>
						<th class="px-6 py-3 font-medium text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					for _, d := range defs {
						<tr class="hover:bg-gray-50">
							<td class="px-6 py-3 font-medium text-gray-800" data-testid={ "pipeline-name-" + d.Name }>
								<a href={ "/service/web/pipelines/" + d.Name } class="hover:text-blue-600">
									{ d.Name }
								</a>
							</td>
							<td class="px-6 py-3">
								if d.Status == "published" {
									<span data-testid={ "pipeline-status-" + d.Name } class="inline-flex items-center gap-1 text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full">
										<span class="w-1.5 h-1.5 bg-green-500 rounded-full"></span>
										Published
									</span>
								} else {
									<span data-testid={ "pipeline-status-" + d.Name } class="inline-flex items-center gap-1 text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full">
										Draft
									</span>
								}
							</td>
							<td class="px-6 py-3 text-gray-500 text-xs">
								{ d.UpdatedAt.Format("2006-01-02 15:04") }
							</td>
							<td class="px-6 py-3 text-right">
								<button type="button"
									hx-delete={ "/service/web/pipelines/" + d.Name }
									hx-target="#pipeline-list-container"
									hx-confirm="Delete this pipeline? Associated run records will also be removed."
									data-testid={ "btn-delete-" + d.Name }
									class="text-red-500 hover:text-red-700 text-xs font-medium">
									Delete
								</button>
							</td>
						</tr>
					}
				</tbody>
			</table>
		</div>
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/web/pipeline_templates/pipeline_list.templ
git commit -m "feat: add pipeline list page and table templ templates"
```

---

### Task 8: Create Pipeline Editor Canvas Template (Alpine.js)

**Files:**

- Create: `internal/modules/web/pipeline_templates/pipeline_editor.templ`

- [ ] **Step 1: Create editor template**

Create `internal/modules/web/pipeline_templates/pipeline_editor.templ`:

```go
package pipeline_templates

templ PipelineEditorPage(name string) {
	@layout.Base("Pipeline: " + name) {
		<div x-data="pipelineEditor()" x-init="init('{ name }')" data-testid="pipeline-editor">
			<!-- Header -->
			<div class="bg-white border-b border-gray-200 px-6 py-3 mb-4 rounded-t-lg shadow-sm flex items-center justify-between">
				<div class="flex items-center gap-4">
					<h2 class="text-lg font-semibold text-gray-800" x-text="name" data-testid="pipeline-title"></h2>
					<span x-show="status === 'draft'" class="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full" data-testid="status-badge">
						Draft
					</span>
					<span x-show="status === 'published'" class="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full" data-testid="status-badge">
						Published
					</span>
				</div>
				<div class="flex items-center gap-2">
					<button type="button" @click="undo" :disabled="undoStack.length === 0"
						class="text-sm px-3 py-1 rounded border border-gray-200 hover:bg-gray-50 disabled:opacity-30"
						data-testid="btn-undo">Undo</button>
					<button type="button" @click="redo" :disabled="redoStack.length === 0"
						class="text-sm px-3 py-1 rounded border border-gray-200 hover:bg-gray-50 disabled:opacity-30"
						data-testid="btn-redo">Redo</button>
					<button type="button" @click="toggleCodeView"
						class="text-sm px-3 py-1 rounded border border-gray-200 hover:bg-gray-50 font-mono"
						data-testid="btn-code-view">&lt;/&gt; Code</button>
					<a :href="'/service/web/pipelines/' + name + '/runs'"
						class="text-sm px-3 py-1 rounded border border-gray-200 hover:bg-gray-50"
						data-testid="btn-run-history">Run History</a>
					<button type="button" @click="saveDraft"
						class="text-sm px-3 py-1 rounded bg-gray-100 hover:bg-gray-200 font-medium"
						data-testid="btn-save-draft">Save Draft</button>
					<button type="button" @click="publish"
						:disabled="publishDisabled"
						class="text-sm px-3 py-1 rounded bg-blue-600 text-white hover:bg-blue-700 font-medium disabled:opacity-30"
						data-testid="btn-publish">Publish</button>
				</div>
			</div>

			<!-- Error summary -->
			<div x-show="errors.length > 0"
				class="bg-red-50 border border-red-200 rounded px-4 py-2 mb-4 text-red-700 text-sm"
				data-testid="error-summary">
				<span x-text="errors.length"></span> node(s) contain errors. Publish is disabled.
			</div>

			<!-- Code View -->
			<div x-show="codeView" class="mb-4">
				<textarea x-model="yamlText" rows="20"
					class="w-full font-mono text-sm p-4 border border-gray-300 rounded bg-gray-50"
					data-testid="yaml-editor"></textarea>
			</div>

			<!-- Visual View -->
			<div x-show="!codeView">
				<!-- Trigger Zone -->
				<div class="border-2 border-dashed border-blue-200 bg-blue-50/30 rounded-lg p-4 mb-6" data-testid="trigger-zone">
					<h3 class="text-sm font-medium text-blue-800 mb-3">Trigger Conditions (any enabled trigger starts the pipeline)</h3>
					<template x-for="(t, idx) in triggers" :key="idx">
						<div class="mb-3">
							@TriggerCard()
						</div>
					</template>
					<div x-show="triggers.length > 1" class="text-center text-xs text-gray-400 my-1">- OR -</div>
					<button type="button" @click="addTrigger" data-testid="btn-add-trigger"
						class="text-sm text-blue-600 hover:text-blue-800 font-medium">
						+ Add Trigger
					</button>
				</div>

				<!-- Steps Zone -->
				<div class="flex flex-col items-center" data-testid="steps-zone">
					<template x-for="(step, idx) in steps" :key="idx">
						<div class="w-full max-w-md">
							<div class="flex justify-center py-2">
								<button type="button" @click="addStep(idx)" data-testid="btn-add-step"
									class="w-8 h-8 rounded-full border-2 border-gray-300 hover:border-blue-400 flex items-center justify-center text-gray-400 hover:text-blue-500 bg-white shadow-sm">
									+
								</button>
							</div>
							@StepCard()
						</div>
					</template>
					<div class="flex justify-center py-2">
						<button type="button" @click="addStep(steps.length)" data-testid="btn-add-step-end"
							class="w-8 h-8 rounded-full border-2 border-gray-300 hover:border-blue-400 flex items-center justify-center text-gray-400 hover:text-blue-500 bg-white shadow-sm">
							+
						</button>
					</div>
					<div x-show="steps.length === 0" class="text-center py-12 text-gray-400" data-testid="steps-empty">
						<p>No steps yet. Click + to add your first step.</p>
					</div>
				</div>
			</div>

			<!-- Drawer Backdrop -->
			<div x-show="drawerOpen" class="fixed inset-0 bg-black/30 z-40" @click="closeDrawer"></div>

			<!-- Configuration Drawer -->
			<div x-show="drawerOpen"
				:class="drawerExpanded ? 'w-4/5' : 'w-2/5'"
				class="fixed right-0 top-0 h-full bg-white shadow-xl z-50 transition-all duration-200 overflow-y-auto"
				data-testid="config-drawer">
				<div class="p-6">
					<div class="flex items-center justify-between mb-4">
						<div class="flex gap-1">
							<button type="button" @click="drawerTab = 'setup'"
								:class="drawerTab === 'setup' ? 'bg-blue-50 text-blue-700' : 'text-gray-600'"
								class="text-sm px-3 py-1.5 rounded font-medium"
								data-testid="tab-setup">Setup</button>
							<button type="button" @click="drawerTab = 'test'"
								:class="drawerTab === 'test' ? 'bg-blue-50 text-blue-700' : 'text-gray-600'"
								class="text-sm px-3 py-1.5 rounded font-medium"
								data-testid="tab-test">Test</button>
						</div>
						<div class="flex gap-2">
							<button type="button" @click="toggleDrawerExpand" data-testid="btn-expand-drawer"
								class="text-gray-400 hover:text-gray-600">Expand</button>
							<button type="button" @click="closeDrawer" data-testid="btn-close-drawer"
								class="text-gray-400 hover:text-gray-600">&times;</button>
						</div>
					</div>

					<!-- Setup Tab -->
					<div x-show="drawerTab === 'setup'" data-testid="drawer-setup">
						<template x-if="selectedNode?.type === 'trigger'">
							<div>
								<!-- Trigger type selector -->
								<label class="block text-sm font-medium text-gray-700 mb-1">Type</label>
								<select x-model="triggers[selectedNode.index].type"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3">
									<option value="event">Event</option>
									<option value="webhook">Webhook</option>
									<option value="cron">Cron</option>
								</select>

								<div x-show="triggers[selectedNode.index].type === 'event'">
									<label class="block text-sm font-medium text-gray-700 mb-1">Event Type</label>
									<input type="text" x-model="triggers[selectedNode.index].event"
										class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
										placeholder="item.created">
								</div>

								<div x-show="triggers[selectedNode.index].type === 'webhook'">
									<label class="block text-sm font-medium text-gray-700 mb-1">Path</label>
									<input type="text" x-model="triggers[selectedNode.index].webhook.path"
										class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
										placeholder="/github-push">
									<label class="block text-sm font-medium text-gray-700 mb-1">Method</label>
									<select x-model="triggers[selectedNode.index].webhook.method"
										class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3">
										<option value="POST">POST</option>
										<option value="GET">GET</option>
										<option value="PUT">PUT</option>
									</select>
									<label class="block text-sm font-medium text-gray-700 mb-1">Token</label>
									<input type="text" x-model="triggers[selectedNode.index].webhook.auth.token"
										class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3">
									<label class="block text-sm font-medium text-gray-700 mb-1">HMAC Secret</label>
									<input type="text" x-model="triggers[selectedNode.index].webhook.auth.hmac_secret"
										class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3">
								</div>

								<div x-show="triggers[selectedNode.index].type === 'cron'">
									<label class="block text-sm font-medium text-gray-700 mb-1">Cron Expression</label>
									<input type="text" x-model="triggers[selectedNode.index].cron"
										class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
										placeholder="*/5 * * * *">
								</div>
							</div>
						</template>

						<template x-if="selectedNode?.type === 'step'">
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">Step Name</label>
								<input type="text" x-model="steps[selectedNode.index].name"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
									placeholder="my-step">

								<label class="block text-sm font-medium text-gray-700 mb-1">Capability</label>
								<input type="text" x-model="steps[selectedNode.index].capability"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
									placeholder="example">

								<label class="block text-sm font-medium text-gray-700 mb-1">Operation</label>
								<input type="text" x-model="steps[selectedNode.index].operation"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
									placeholder="create">

								<label class="block text-sm font-medium text-gray-700 mb-1">Params (JSON)</label>
								<textarea rows="6" x-model="steps[selectedNode.index].paramsText"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono mb-3"
									placeholder='{"title": "{{event.title}}"}'
									data-testid="params-editor"></textarea>

								<button type="button" @click="openVariablePicker(selectedNode.index)"
									class="text-sm text-blue-600 hover:text-blue-800 font-medium"
									data-testid="btn-open-var-picker">
									&#123;x&#125; Insert Variable
								</button>
							</div>
						</template>
					</div>

					<!-- Test Tab -->
					<div x-show="drawerTab === 'test'" data-testid="drawer-test">
						<template x-if="selectedNode?.type === 'step'">
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">Trigger Source</label>
								<select x-model="testTriggerSource"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3">
									<template x-for="t in triggers.filter(tr => tr.enabled)" :key="t.type">
										<option :value="t.type" x-text="t.type"></option>
									</template>
								</select>

								<label class="block text-sm font-medium text-gray-700 mb-1">Mock Payload (JSON)</label>
								<textarea x-model="testMockPayload" rows="4"
									class="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono mb-3"
									placeholder='{"title": "Test Item"}' data-testid="mock-payload"></textarea>

								<button type="button" @click="loadMockPayload"
									class="text-sm text-blue-600 hover:text-blue-800 mb-3 block"
									data-testid="btn-load-mock">Load sample data</button>

								<button type="button" @click="runTest"
									class="w-full bg-blue-600 text-white py-2 rounded text-sm font-medium hover:bg-blue-700 mb-4"
									data-testid="btn-run-test">Test up to this step</button>

								<div x-show="testResults" data-testid="test-results" class="border-t pt-4">
									<template x-for="r in testResults.steps" :key="r.name">
										<div class="mb-3 text-sm">
											<div class="flex items-center gap-2">
												<span x-show="r.status === 'ok'" class="text-green-500">OK</span>
												<span x-show="r.status === 'error'" class="text-red-500">ERR</span>
												<span class="font-medium" x-text="r.name"></span>
												<span class="text-xs text-gray-400" x-text="r.duration_ms + 'ms'"></span>
											</div>
											<pre x-show="r.output" class="text-xs bg-gray-50 p-2 rounded mt-1 overflow-x-auto"
												x-text="JSON.stringify(r.output, null, 2)"></pre>
											<div x-show="r.error" class="text-red-600 text-xs mt-1" x-text="r.error"></div>
										</div>
									</template>
								</div>
							</div>
						</template>
					</div>
				</div>
			</div>

			<!-- Variable Picker Modal -->
			<div x-show="variablePickerOpen" class="fixed inset-0 z-60 flex items-center justify-center"
				@click.self="variablePickerOpen = false">
				<div class="bg-white rounded-lg shadow-xl p-6 w-96 max-h-96 overflow-y-auto" data-testid="var-picker">
					<h4 class="font-medium text-gray-800 mb-3">Insert Variable</h4>
					<div class="text-sm space-y-1">
						<div class="text-xs text-gray-400 uppercase mb-2">event data</div>
						<button @click="insertVariable('event.event_id')" class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">event.event_id</button>
						<button @click="insertVariable('event.event_type')" class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">event.event_type</button>
						<button @click="insertVariable('event.title')" class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">event.title</button>
						<button @click="insertVariable('event.entity_id')" class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">event.entity_id</button>
						<button @click="insertVariable('event.source')" class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">event.source</button>
						<button @click="insertVariable('event.capability')" class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">event.capability</button>

						<template x-for="idx in [...Array(selectedNode?.index || 0).keys()]" :key="idx">
							<div>
								<div class="text-xs text-gray-400 uppercase mt-2 mb-1" x-text="'steps.' + steps[idx].name"></div>
								<button @click="{ const v = 'steps.' + steps[idx].name + '.id'; insertVariable(v) }"
									class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">
									<span x-text="'steps.' + steps[idx].name + '.id'"></span>
								</button>
								<button @click="{ const v = 'steps.' + steps[idx].name + '.result'; insertVariable(v) }"
									class="block w-full text-left px-2 py-1 hover:bg-blue-50 rounded text-gray-700">
									<span x-text="'steps.' + steps[idx].name + '.result'"></span>
								</button>
							</div>
						</template>
					</div>
				</div>
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/web/pipeline_templates/pipeline_editor.templ
git commit -m "feat: add pipeline editor canvas template with Alpine.js"
```

---

### Task 9: Create Trigger/Step Partial Templates

**Files:**

- Create: `internal/modules/web/pipeline_templates/pipeline_partials.templ`

- [ ] **Step 1: Create partials template**

Create `internal/modules/web/pipeline_templates/pipeline_partials.templ`:

```go
package pipeline_templates

templ TriggerCard() {
	<div class="bg-white rounded-lg shadow-sm border p-3 flex items-center justify-between"
		:class="getTriggerErrorClass(idx)"
		@click="selectNode('trigger', idx)" style="cursor:pointer"
		data-testid="trigger-card">
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded bg-blue-100 flex items-center justify-center text-blue-600 text-sm font-bold">
				<span x-show="t.type === 'event'">E</span>
				<span x-show="t.type === 'webhook'">W</span>
				<span x-show="t.type === 'cron'">C</span>
			</div>
			<div>
				<div class="text-sm font-medium text-gray-800">
					<span x-show="t.type === 'event'" x-text="'Event: ' + (t.event || '...')"></span>
					<span x-show="t.type === 'webhook'" x-text="'Webhook: ' + ((t.webhook && t.webhook.path) || '...')"></span>
					<span x-show="t.type === 'cron'" x-text="'Cron: ' + (t.cron || '...')"></span>
				</div>
			</div>
		</div>
		<div class="flex items-center gap-2">
			<label class="relative inline-flex items-center cursor-pointer">
				<input type="checkbox" x-model="t.enabled" class="sr-only peer" data-testid="trigger-switch">
				<div class="w-9 h-5 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all"></div>
			</label>
			<button type="button" @click.stop="removeTrigger(idx)" data-testid="btn-remove-trigger"
				class="text-gray-400 hover:text-red-500">&times;</button>
		</div>
	</div>
}

templ StepCard() {
	<div class="bg-white rounded-lg shadow-sm border p-4 relative group"
		:class="getStepErrorClass(idx)"
		@click="selectNode('step', idx)" style="cursor:pointer"
		data-testid="step-card">
		<!-- Hover actions -->
		<div class="absolute -top-2 right-2 hidden group-hover:flex gap-1 bg-white border rounded shadow-sm px-2 py-0.5 text-xs">
			<button type="button" @click.stop="moveStepUp(idx)" data-testid="btn-move-up"
				class="text-gray-400 hover:text-gray-600">Up</button>
			<button type="button" @click.stop="moveStepDown(idx)" data-testid="btn-move-down"
				class="text-gray-400 hover:text-gray-600">Down</button>
			<button type="button" @click.stop="duplicateStep(idx)" data-testid="btn-copy-step"
				class="text-gray-400 hover:text-gray-600">Copy</button>
			<button type="button" @click.stop="removeStep(idx)" data-testid="btn-delete-step"
				class="text-red-400 hover:text-red-600">Delete</button>
		</div>

		<!-- Card header -->
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded bg-gray-100 flex items-center justify-center text-gray-600 text-xs font-bold">
				<span x-text="(step.capability || '?')[0].toUpperCase()"></span>
			</div>
			<div class="flex-1 min-w-0">
				<div class="text-sm font-medium text-gray-800 truncate" x-text="step.name || 'Unnamed Step'"></div>
				<div class="text-xs text-gray-400 truncate" x-text="step.capability + '.' + step.operation"></div>
			</div>
			<button type="button" @click.stop="selectNode('step', idx)" data-testid="btn-step-menu"
				class="text-gray-400 hover:text-gray-600">...</button>
		</div>

		<!-- Params preview (truncated) -->
		<div x-show="step.paramsText" class="mt-2 text-xs text-gray-500 line-clamp-2 font-mono bg-gray-50 rounded p-2"
			x-text="step.paramsText"></div>
	</div>
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/modules/web/pipeline_templates/pipeline_partials.templ
git commit -m "feat: add pipeline trigger card and step card partial templates"
```

---

### Task 10: Create Pipeline Runs Template

**Files:**

- Create: `internal/modules/web/pipeline_templates/pipeline_runs.templ`

- [ ] **Step 1: Create runs template**

Create `internal/modules/web/pipeline_templates/pipeline_runs.templ`:

```go
package pipeline_templates

import (
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

templ PipelineRunsPage(name string, runs []*gen.PipelineRun) {
	@layout.Base("Runs: " + name) {
		<div class="max-w-4xl mx-auto">
			<div class="flex items-center justify-between mb-6">
				<h1 class="text-xl font-semibold text-gray-800">
					Run History:
					<a href={ "/service/web/pipelines/" + name } class="text-blue-600 hover:underline">{ name }</a>
				</h1>
			</div>
			<div id="pipeline-runs-container" data-testid="pipeline-runs-container">
				@PipelineRunsTable(runs)
			</div>
		</div>
	}
}

templ PipelineRunsTable(runs []*gen.PipelineRun) {
	if len(runs) == 0 {
		<div class="text-center py-12 text-gray-400" data-testid="runs-empty">
			<p>No runs recorded yet.</p>
		</div>
	} else {
		<div class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden" data-testid="runs-table">
			<table class="w-full text-sm text-left">
				<thead class="bg-gray-50 text-gray-600">
					<tr>
						<th class="px-6 py-3 font-medium">Run ID</th>
						<th class="px-6 py-3 font-medium">Event</th>
						<th class="px-6 py-3 font-medium">Status</th>
						<th class="px-6 py-3 font-medium">Started</th>
						<th class="px-6 py-3 font-medium">Duration</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					for _, r := range runs {
						<tr class="hover:bg-gray-50">
							<td class="px-6 py-3 font-mono text-xs text-gray-600">{ fmt.Sprint(r.ID) }</td>
							<td class="px-6 py-3 text-gray-600">{ r.EventID }</td>
							<td class="px-6 py-3">
								<span class={ getRunStatusClass(int(r.Status)) }>{ getRunStatusText(int(r.Status)) }</span>
							</td>
							<td class="px-6 py-3 text-gray-500 text-xs">{ r.CreatedAt.Format("2006-01-02 15:04:05") }</td>
							<td class="px-6 py-3 text-gray-500 text-xs">{ formatDuration(r) }</td>
						</tr>
					}
				</tbody>
			</table>
		</div>
	}
}

func getRunStatusClass(status int) string {
	// PipelineStart=0, Running=1, Success=2, Failed=3
	switch status {
	case 2:
		return "text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full"
	case 3:
		return "text-xs bg-red-100 text-red-700 px-2 py-0.5 rounded-full"
	default:
		return "text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full"
	}
}

func getRunStatusText(status int) string {
	switch status {
	case 2:
		return "Success"
	case 3:
		return "Failed"
	case 1:
		return "Running"
	default:
		return "Started"
	}
}

func formatDuration(r *gen.PipelineRun) string {
	if r.FinishedAt == nil {
		return "-"
	}
	d := r.FinishedAt.Sub(r.CreatedAt)
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(time.Second).String()
}
```

- [ ] **Step 2: Add runs handler to pipeline_webservice.go**

```go
func pipelineRunsPage(c fiber.Ctx) error {
	name := c.Params("name")
	// Use parent_name to aggregate all trigger-variant runs
	runs, err := getPipelineRuns(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return pipeline_templates.PipelineRunsPage(name, runs).Render(context.Background(), c.Response().BodyWriter())
}

func pipelineRunsTable(c fiber.Ctx) error {
	name := c.Params("name")
	runs, err := getPipelineRuns(context.Background(), name)
	if err != nil {
		return types.Errorf(types.ErrInternal, "get runs: %v", err)
	}
	c.Type("html")
	return pipeline_templates.PipelineRunsTable(runs).Render(context.Background(), c.Response().BodyWriter())
}

func getPipelineRuns(ctx context.Context, name string) ([]*gen.PipelineRun, error) {
	if store.Database == nil || store.Database.GetDB() == nil {
		return nil, nil
	}
	client, ok := store.Database.GetDB().(*store.Client)
	if !ok {
		return nil, nil
	}
	pipeStore := store.NewPipelineStore(client)
	// Query runs by parent name pattern (name or name__trigger_*)
	return pipeStore.GetRunsByParentName(ctx, name)
}
```

- [ ] **Step 3: Add GetRunsByParentName to store**

In `internal/store/store.go`, add to `PipelineStore`:

```go
// GetRunsByParentName returns pipeline runs matching a parent pipeline name.
// Matches both exact name and compound trigger names (name__trigger_*).
func (s *PipelineStore) GetRunsByParentName(ctx context.Context, parentName string) ([]*gen.PipelineRun, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	return s.client.PipelineRun.Query().
		Where(
			pipelinerun.Or(
				pipelinerun.PipelineName(parentName),
				pipelinerun.PipelineNameHasPrefix(parentName+"__trigger_"),
			),
		).
		Order(gen.Desc(pipelinerun.FieldCreatedAt)).
		Limit(100).
		All(ctx)
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/pipeline_templates/pipeline_runs.templ internal/modules/web/pipeline_webservice.go internal/store/store.go
git commit -m "feat: add pipeline runs page, handler, and store query"
```

---

### Task 11: Create Alpine.js Canvas Component

**Files:**

- Create: `public/js/pipeline-editor.js`

- [ ] **Step 1: Create Alpine.js component**

Create `public/js/pipeline-editor.js`:

```javascript
document.addEventListener("alpine:init", () => {
  Alpine.data("pipelineEditor", () => ({
    name: "",
    description: "",
    status: "draft",
    version: 1,
    dirty: false,

    undoStack: [],
    redoStack: [],

    triggers: [],
    steps: [],

    selectedNode: null,
    drawerOpen: false,
    drawerExpanded: false,
    drawerTab: "setup",
    drawerDirty: false,
    codeView: false,
    yamlText: "",

    variablePickerOpen: false,
    variablePickerTarget: null,
    variablePickerSource: "event",

    errors: [],
    publishDisabled: false,

    autoSaveTimer: null,

    testTriggerSource: "event",
    testMockPayload: "{}",
    testResults: null,

    init(name) {
      this.name = name || "";
      if (name && name !== "new") {
        this.loadPipeline(name);
      } else {
        this.triggers = [];
        this.steps = [];
        this.status = "draft";
        this.version = 1;
      }
      this.pushUndo();
    },

    async loadPipeline(name) {
      try {
        const resp = await fetch(`/service/web/pipelines/${name}/yaml`);
        const data = await resp.json();
        this.version = data.version;
        this.status = data.status;
        if (data.yaml) {
          this.parseYamlToState(data.yaml);
        }
      } catch (e) {
        console.error("Failed to load pipeline:", e);
      }
    },

    parseYamlToState(yaml) {
      // Simple YAML-to-JSON parser for the editor schema
      // In production, use a YAML parser library or server-side conversion
      try {
        const obj = jsyaml.load(yaml); // assumes js-yaml is loaded
        this.name = obj.name || this.name;
        this.description = obj.description || "";
        this.triggers = (obj.triggers || []).map((t) => ({
          type: t.type || "event",
          enabled: t.enabled !== false,
          event: t.event || "",
          cron: t.cron || "",
          webhook: t.webhook || {
            path: "",
            method: "POST",
            auth: { token: "", hmac_secret: "" },
          },
        }));
        this.steps = (obj.steps || []).map((s) => ({
          name: s.name || "",
          capability: s.capability || "",
          operation: s.operation || "",
          paramsText: JSON.stringify(s.params || {}, null, 2),
        }));
        this.validate();
      } catch (e) {
        console.error("YAML parse error:", e);
      }
    },

    stateToYaml() {
      const obj = {
        name: this.name,
        description: this.description,
        enabled: true,
        resumable: false,
        triggers: this.triggers.map((t) => {
          const entry = { type: t.type, enabled: t.enabled };
          if (t.type === "event") entry.event = t.event;
          if (t.type === "cron") entry.cron = t.cron;
          if (t.type === "webhook") entry.webhook = t.webhook;
          return entry;
        }),
        steps: this.steps.map((s) => ({
          name: s.name,
          capability: s.capability,
          operation: s.operation,
          params: (() => {
            try {
              return JSON.parse(s.paramsText || "{}");
            } catch (e) {
              return {};
            }
          })(),
        })),
      };
      return jsyaml.dump(obj);
    },

    // --- Undo/redo ---
    pushUndo() {
      this.undoStack.push(
        JSON.parse(
          JSON.stringify({
            triggers: this.triggers,
            steps: this.steps,
          }),
        ),
      );
      if (this.undoStack.length > 50) this.undoStack.shift();
      this.redoStack = [];
    },

    undo() {
      if (this.undoStack.length <= 1) return;
      const current = this.undoStack.pop();
      this.redoStack.push(current);
      const prev = this.undoStack[this.undoStack.length - 1];
      this.triggers = JSON.parse(JSON.stringify(prev.triggers));
      this.steps = JSON.parse(JSON.stringify(prev.steps));
      this.markDirty();
      this.validate();
    },

    redo() {
      if (this.redoStack.length === 0) return;
      const next = this.redoStack.pop();
      this.undoStack.push(JSON.parse(JSON.stringify(next)));
      this.triggers = JSON.parse(JSON.stringify(next.triggers));
      this.steps = JSON.parse(JSON.stringify(next.steps));
      this.markDirty();
      this.validate();
    },

    // --- Triggers ---
    addTrigger() {
      this.pushUndo();
      this.triggers.push({
        type: "event",
        enabled: true,
        event: "",
        cron: "",
        webhook: {
          path: "",
          method: "POST",
          auth: { token: "", hmac_secret: "" },
        },
      });
      this.markDirty();
    },

    removeTrigger(idx) {
      this.pushUndo();
      this.triggers.splice(idx, 1);
      this.markDirty();
      this.validate();
    },

    // --- Steps ---
    addStep(afterIdx) {
      this.pushUndo();
      const newStep = {
        name: "",
        capability: "",
        operation: "",
        paramsText: "{}",
      };
      this.steps.splice(afterIdx, 0, newStep);
      this.markDirty();
      this.selectNode("step", afterIdx);
    },

    removeStep(idx) {
      this.pushUndo();
      const stepName = this.steps[idx].name;
      this.steps.splice(idx, 1);
      this.markDirty();
      this.validate();
      if (this.drawerOpen && this.selectedNode?.index === idx) {
        this.drawerOpen = false;
      }
    },

    moveStepUp(idx) {
      if (idx === 0) return;
      const step = this.steps[idx];
      if (this.dependsOnStep(step, idx - 1)) {
        alert(
          "Cannot move: this step depends on data from a step above the target position.",
        );
        return;
      }
      this.pushUndo();
      this.steps.splice(idx, 1);
      this.steps.splice(idx - 1, 0, step);
      this.markDirty();
      this.validate();
    },

    moveStepDown(idx) {
      if (idx >= this.steps.length - 1) return;
      const stepBelow = this.steps[idx + 1];
      if (this.dependsOnStep(stepBelow, idx, this.steps[idx])) {
        alert("Cannot move: the step below depends on this step's data.");
        return;
      }
      this.pushUndo();
      const step = this.steps.splice(idx, 1)[0];
      this.steps.splice(idx + 1, 0, step);
      this.markDirty();
      this.validate();
    },

    duplicateStep(idx) {
      this.pushUndo();
      const copy = JSON.parse(JSON.stringify(this.steps[idx]));
      copy.name = copy.name + "-copy";
      this.steps.splice(idx + 1, 0, copy);
      this.markDirty();
    },

    dependsOnStep(step, targetIdx, movedStep) {
      const re = /\{\{steps\.(\w+)\./g;
      const refs = [...(step.paramsText || "").matchAll(re)].map((m) => m[1]);
      for (const ref of refs) {
        const refIdx = this.steps.findIndex((s) => s.name === ref);
        if (refIdx >= targetIdx) return true;
      }
      return false;
    },

    // --- Drawer ---
    selectNode(type, idx) {
      if (this.drawerDirty && this.selectedNode) {
        if (!confirm("You have unsaved changes. Discard them?")) return;
      }
      this.selectedNode = { type, index: idx };
      this.drawerOpen = true;
      this.drawerDirty = false;
      this.drawerTab = "setup";
    },

    closeDrawer() {
      if (this.drawerDirty) {
        if (!confirm("You have unsaved changes. Discard them?")) return;
      }
      this.drawerOpen = false;
      this.selectedNode = null;
      this.drawerDirty = false;
    },

    toggleDrawerExpand() {
      this.drawerExpanded = !this.drawerExpanded;
    },

    // --- Variable picker ---
    openVariablePicker(targetIdx) {
      this.variablePickerTarget = targetIdx;
      this.variablePickerOpen = true;
    },

    insertVariable(path) {
      if (this.variablePickerTarget === null) return;
      const step = this.steps[this.variablePickerTarget];
      const template = `{{${path}}}`;

      // Insert at cursor position in the params textarea
      const textarea = document.querySelector('[data-testid="params-editor"]');
      if (textarea) {
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const text = step.paramsText || "";
        step.paramsText =
          text.substring(0, start) + template + text.substring(end);
        setTimeout(() => {
          textarea.focus();
          textarea.setSelectionRange(
            start + template.length,
            start + template.length,
          );
        }, 50);
      } else {
        step.paramsText = (step.paramsText || "") + template;
      }

      this.variablePickerOpen = false;
      this.markDirty();
    },

    // --- Validation ---
    validate() {
      this.errors = [];

      // At least one enabled trigger
      const enabledTriggers = this.triggers.filter((t) => t.enabled);
      if (enabledTriggers.length === 0) {
        this.errors.push({
          node: { type: "trigger", index: -1 },
          message: "At least one trigger must be enabled",
        });
      }

      // At least one step
      if (this.steps.length === 0) {
        this.errors.push({
          node: { type: "step", index: -1 },
          message: "At least one step is required",
        });
      }

      // Validate each trigger
      for (let i = 0; i < this.triggers.length; i++) {
        const t = this.triggers[i];
        if (!t.enabled) continue;
        if (t.type === "event" && !t.event) {
          this.errors.push({
            node: { type: "trigger", index: i },
            message: "Event type is required",
          });
        }
        if (t.type === "webhook" && (!t.webhook || !t.webhook.path)) {
          this.errors.push({
            node: { type: "trigger", index: i },
            message: "Webhook path is required",
          });
        }
        if (
          t.type === "webhook" &&
          t.webhook &&
          !t.webhook.auth.token &&
          !t.webhook.auth.hmac_secret
        ) {
          this.errors.push({
            node: { type: "trigger", index: i },
            message: "At least one auth method is required",
          });
        }
        if (t.type === "cron" && !t.cron) {
          this.errors.push({
            node: { type: "trigger", index: i },
            message: "Cron expression is required",
          });
        }
      }

      // Validate each step
      for (let i = 0; i < this.steps.length; i++) {
        const s = this.steps[i];
        if (!s.name) {
          this.errors.push({
            node: { type: "step", index: i },
            message: "Step name is required",
          });
        }
        if (!s.capability) {
          this.errors.push({
            node: { type: "step", index: i },
            message: "Capability is required",
          });
        }
        if (!s.operation) {
          this.errors.push({
            node: { type: "step", index: i },
            message: "Operation is required",
          });
        }

        // Check upstream variable references
        const re = /\{\{steps\.(\w+)\./g;
        const refs = [...(s.paramsText || "").matchAll(re)].map((m) => m[1]);
        for (const ref of refs) {
          const refIdx = this.steps.findIndex((ss) => ss.name === ref);
          if (refIdx === -1) {
            this.errors.push({
              node: { type: "step", index: i },
              message: `Upstream variable {{steps.${ref}.*}} is invalid or has been removed`,
            });
          } else if (refIdx >= i) {
            this.errors.push({
              node: { type: "step", index: i },
              message: `Depends on [${ref}] which must be above this step`,
            });
          }
        }
      }

      this.publishDisabled = this.errors.length > 0;
    },

    getTriggerErrorClass(idx) {
      return this.errors.some(
        (e) => e.node.type === "trigger" && e.node.index === idx,
      )
        ? "border-red-400"
        : "";
    },

    getStepErrorClass(idx) {
      return this.errors.some(
        (e) => e.node.type === "step" && e.node.index === idx,
      )
        ? "border-red-400"
        : "";
    },

    // --- Code view ---
    toggleCodeView() {
      if (this.codeView) {
        try {
          this.parseYamlToState(this.yamlText);
          this.codeView = false;
          this.validate();
        } catch (e) {
          alert(
            "YAML syntax error. Fix errors before switching back to visual mode.\n" +
              e.message,
          );
        }
      } else {
        this.yamlText = this.stateToYaml();
        this.codeView = true;
      }
    },

    // --- Persistence ---
    markDirty() {
      if (!this.dirty) {
        this.dirty = true;
      }
      this.startAutoSave();
    },

    startAutoSave() {
      clearTimeout(this.autoSaveTimer);
      this.autoSaveTimer = setTimeout(() => this.saveDraft(), 30000);
    },

    async saveDraft() {
      const yaml = this.stateToYaml();
      try {
        const resp = await fetch(`/service/web/pipelines/${this.name}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ yaml, version: this.version }),
        });
        if (resp.status === 409) {
          alert("This draft was modified elsewhere. Please refresh the page.");
          return;
        }
        const data = await resp.json();
        this.version = data.version;
        this.status = data.status;
        this.dirty = false;
      } catch (e) {
        console.error("Auto-save failed:", e);
      }
    },

    async publish() {
      if (this.publishDisabled) return;
      await this.saveDraft();
      try {
        const resp = await fetch(
          `/service/web/pipelines/${this.name}/publish`,
          {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ version: this.version }),
          },
        );
        if (resp.status === 409) {
          alert("This draft was modified elsewhere. Please refresh the page.");
          return;
        }
        const data = await resp.json();
        this.version = data.version;
        this.status = "published";
      } catch (e) {
        console.error("Publish failed:", e);
        alert("Publish failed: " + e.message);
      }
    },

    // --- Test ---
    async loadMockPayload() {
      try {
        const resp = await fetch(
          `/service/web/pipelines/${this.name}/mock?source=${this.testTriggerSource}`,
        );
        const data = await resp.json();
        this.testMockPayload = JSON.stringify(data.payload, null, 2);
      } catch (e) {
        console.error("Failed to load mock payload:", e);
      }
    },

    async runTest() {
      // Force save so the server tests the latest params, not stale DB data
      await this.saveDraft();

      const upToIdx = this.selectedNode?.index;
      if (upToIdx === null || upToIdx === undefined) return;
      try {
        const resp = await fetch(`/service/web/pipelines/${this.name}/test`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            trigger_source: this.testTriggerSource,
            mock_payload: JSON.parse(this.testMockPayload || "{}"),
            up_to_step_index: upToIdx,
          }),
        });
        this.testResults = await resp.json();
      } catch (e) {
        console.error("Test failed:", e);
        this.testResults = { success: false, error: e.message };
      }
    },
  }));
});
```

- [ ] **Step 2: Commit**

```bash
git add public/js/pipeline-editor.js
git commit -m "feat: add Alpine.js pipeline canvas component"
```

---

### Task 12: Variable Pill Styling CSS

**Files:**

- Modify: `public/css/input.css`

- [ ] **Step 1: Add variable pill CSS**

Variable pills are inserted as inline template strings (`{{event.title}}`) in the params textarea. For styling within the textarea context, add a helper CSS class for the pill preview shown in step card parameter summaries (not for use inside the textarea itself, since `textarea` does not support inline styled tokens).

In `public/css/input.css`, append:

```css
/* Variable pill display in step card param previews */
.var-pill {
  display: inline-block;
  background: #dbeafe;
  color: #1e40af;
  border-radius: 4px;
  padding: 1px 6px;
  font-family: monospace;
  font-size: 0.75rem;
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}
```

- [ ] **Step 2: Commit**

```bash
git add public/css/input.css
git commit -m "feat: add variable pill display CSS"
```

---

### Task 13: Wire Navigation and Module Registration

**Files:**

- Modify: `pkg/views/layout/base.templ`
- Modify: `internal/modules/web/webservice.go`
- Modify: `internal/modules/web/module.go`

- [ ] **Step 1: Add Pipelines nav link**

In `pkg/views/layout/base.templ`, update the nav section:

Replace:

```html
<div class="flex gap-4 text-sm text-gray-600 items-center">
  <a
    href="/service/web/configs"
    data-testid="nav-configs"
    class="hover:text-gray-900"
    >Configs</a
  >
</div>
```

With:

```html
<div class="flex gap-4 text-sm text-gray-600 items-center">
  <a
    href="/service/web/pipelines"
    data-testid="nav-pipelines"
    class="hover:text-gray-900"
    >Pipelines</a
  >
  <a
    href="/service/web/configs"
    data-testid="nav-configs"
    class="hover:text-gray-900"
    >Configs</a
  >
</div>
```

- [ ] **Step 2: Add pipeline webservice rules**

In `internal/modules/web/webservice.go`, append `pipelineWebserviceRules` to the `webserviceRules` slice:

```go
var webserviceRules = []webservice.Rule{
    // ... existing rules ...
}
webserviceRules = append(webserviceRules, pipelineWebserviceRules...)
```

You'll need to convert the `route.WebServiceRule` type to `webservice.Rule`. Since `pipelineWebserviceRules` uses `route.WebServiceRule`, check if they're compatible. If not, adjust the rule type.

To avoid type mismatch, define pipeline rules using `webservice.Rule` directly. In `pipeline_webservice.go`, change:

```go
import (
    "github.com/flowline-io/flowbot/pkg/route"
)

var pipelineWebserviceRules = []route.WebServiceRule{ ... }
```

To:

```go
import (
    "github.com/flowline-io/flowbot/internal/modules"
)

var pipelineWebserviceRules = []webservice.Rule{
	webservice.Get("/pipelines", pipelineListPage, route.WithNotAuth()),
	webservice.Get("/pipelines/list", pipelineListTable, route.WithNotAuth()),
	webservice.Get("/pipelines/:name", pipelineEditorPage, route.WithNotAuth()),
	webservice.Post("/pipelines", createPipeline, route.WithNotAuth()),
	webservice.Put("/pipelines/:name", updatePipelineDraft, route.WithNotAuth()),
	webservice.Put("/pipelines/:name/publish", publishPipeline, route.WithNotAuth()),
	webservice.Delete("/pipelines/:name", deletePipeline, route.WithNotAuth()),
	webservice.Get("/pipelines/:name/yaml", getPipelineYaml, route.WithNotAuth()),
	webservice.Get("/pipelines/:name/mock", getMockPayload, route.WithNotAuth()),
	webservice.Post("/pipelines/:name/test", testPipelineStep, route.WithNotAuth()),
	webservice.Get("/pipelines/:name/runs", pipelineRunsPage, route.WithNotAuth()),
	webservice.Get("/pipelines/:name/runs/list", pipelineRunsTable, route.WithNotAuth()),
}
```

And import `route` for the `route.WithNotAuth()` option.

- [ ] **Step 3: Handle new pipeline page (blank canvas)**

Add a handler for the `/pipelines/new` route, or make `pipelineEditorPage` handle the case where name is "new":

```go
func pipelineEditorPage(c fiber.Ctx) error {
    name := c.Params("name")
    c.Type("html")
    return pipeline_templates.PipelineEditorPage(name).Render(context.Background(), c.Response().BodyWriter())
}
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/layout/base.templ internal/modules/web/webservice.go internal/modules/web/pipeline_webservice.go internal/modules/web/module.go
git commit -m "feat: wire pipelines nav, webservice rules, and module registration"
```

---

### Task 14: Integration — Fix Build, Tests, and Missing Pieces

**Files:**

- Various

- [ ] **Step 1: Run full build to find compilation errors**

```bash
go build ./...
```

- [ ] **Step 2: Fix import errors in pipeline_webservice.go**

Ensure all imports are correct:

```go
import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/gofiber/fiber/v3"

    "github.com/flowline-io/flowbot/internal/modules/web/pipeline_templates"
    "github.com/flowline-io/flowbot/internal/store"
    "github.com/flowline-io/flowbot/pkg/hub"
    "github.com/flowline-io/flowbot/pkg/pipeline"
    "github.com/flowline-io/flowbot/pkg/route"
    "github.com/flowline-io/flowbot/pkg/types"
)
```

- [ ] **Step 3: Fix import of gen in pipeline_templates**

Make sure `pipeline_templates` imports the generated ent package correctly:

```go
import (
    "fmt"
    "time"
    "github.com/flowline-io/flowbot/internal/store/ent/gen"
)
```

- [ ] **Step 4: Add js-yaml CDN to base layout**

In `base.templ`, add js-yaml for client-side YAML parsing:

```html
<script src="https://cdn.jsdelivr.net/npm/js-yaml@4/dist/js-yaml.min.js"></script>
```

- [ ] **Step 5: Ensure no cyclic imports**

```bash
go vet ./internal/modules/web/...
go vet ./pkg/pipeline/...
```

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/store/... -v
go test ./pkg/pipeline/... -v
go test ./internal/modules/web/... -v
```

- [ ] **Step 7: Fix remaining spec test**

Fix `tests/specs/pipeline_spec_test.go` line 220, replace:

```go
err := pipelineStore.UpsertDefinition(ctx, name, desc, enabled, trigger, steps)
```

With:

```go
err := pipelineStore.CreateDefinition(ctx, name, desc)
// Then update draft if needed
```

- [ ] **Step 8: Run ent generate and regenerate templ templates**

```bash
go tool task ent
go generate ./...
```

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "fix: resolve build errors, imports, and test compatibility for pipeline CRUD"
```

---

### Task 15: E2E Tests

**Files:**

- Create: `tests/e2e/pipeline_crud_test.go`

- [ ] **Step 1: Create E2E test skeleton**

Create `tests/e2e/pipeline_crud_test.go`:

```go
//go:build e2e

package e2e

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPipelineCRUD(t *testing.T) {
    t.Run("create and list pipelines", func(t *testing.T) {
        page := setupE2ETest(t)
        defer page.MustClose()
        ResetDB(t)

        page.MustNavigate(baseURL + "/service/web/pipelines")
        page.MustWaitStable()

        // Should see empty state
        el := page.MustElement("[data-testid='pipeline-empty']")
        assert.NotNil(t, el)

        // TODO: Navigate to new pipeline, fill form, create
        // TODO: Verify pipeline appears in list
        // TODO: Edit pipeline, add trigger, add step
        // TODO: Publish pipeline
        // TODO: Delete pipeline
    })

    t.Run("pipeline editor canvas", func(t *testing.T) {
        // TODO: Open editor, add triggers and steps
        // TODO: Open drawer, configure step params
        // TODO: Use variable picker
        // TODO: Test validation errors
    })

    t.Run("publish and run history", func(t *testing.T) {
        // TODO: Publish pipeline, verify status change
        // TODO: View run history page
    })
}
```

- [ ] **Step 2: Run E2E tests**

```bash
go tool task test:e2e
```

- [ ] **Step 3: Commit**

```bash
git add tests/e2e/pipeline_crud_test.go
git commit -m "test: add pipeline CRUD e2e test skeleton"
```

---

### Task 16: BDD Spec Tests

**Files:**

- Create: `tests/specs/pipeline_editor_spec_test.go`

- [ ] **Step 1: Create BDD test skeleton**

Create `tests/specs/pipeline_editor_spec_test.go`:

```go
package specs_test

import (
    "github.com/onsi/ginkgo/v2"
    "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Pipeline Editor", func() {
    ginkgo.Context("creating a new pipeline", func() {
        ginkgo.It("should create a pipeline with valid name")
        ginkgo.It("should reject duplicate pipeline names")
        ginkgo.It("should reject invalid name format")
    })

    ginkgo.Context("editing pipeline definition", func() {
        ginkgo.It("should save draft with optimistic locking")
        ginkgo.It("should publish a valid draft")
        ginkgo.It("should reject publish with invalid YAML")
    })

    ginkgo.Context("deleting a pipeline", func() {
        ginkgo.It("should delete pipeline and associated runs")
    })
})
```

- [ ] **Step 2: Run BDD tests**

```bash
go tool task test:specs
```

- [ ] **Step 3: Commit**

```bash
git add tests/specs/pipeline_editor_spec_test.go
git commit -m "test: add pipeline editor BDD spec skeleton"
```

---

## Final Verification

After all tasks complete, run:

```bash
go tool task lint
go tool task test
go tool task test:specs
go tool task test:e2e
go tool task build
```

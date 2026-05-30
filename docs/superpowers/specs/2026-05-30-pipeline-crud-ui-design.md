# Pipeline CRUD Web UI Design

**Date**: 2026-05-30
**Status**: Draft

## Overview

Add a visual pipeline editor to the web UI with CRUD operations, draft/publish lifecycle,
and a step-based canvas interface. Users create, edit, test, and publish pipeline definitions
through a browser UI instead of editing YAML files directly.

## Decisions

| Decision | Choice |
|----------|--------|
| Frontend approach | templ + HTMX + Alpine.js, fixed vertical sequence (no free-form drag) |
| Storage | Database-backed (`pipeline_definitions` table) with draft + published columns |
| Module location | Extend existing `internal/modules/web/` |
| Scope (v1) | Definitions CRUD + read-only run history |
| Trigger schema | Multiple triggers array with OR logic |
| All UI text | English only |

## 1. Database Schema

### pipeline_definitions table

```sql
CREATE TABLE pipeline_definitions (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    description    TEXT DEFAULT '',
    yaml_draft     TEXT NOT NULL DEFAULT '',
    yaml_published TEXT DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'draft',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- `yaml_draft`: current working copy, auto-saved every 30s
- `yaml_published`: set when user publishes, copied from draft
- `status`: 'draft' when no published version exists or draft differs from published; 'published' when draft matches published

### Store interface (in `internal/store/store.go`)

```go
type PipelineDefinitionStore struct { client *gen.Client }

func (s *PipelineDefinitionStore) Create(ctx, name, description string) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) GetByName(ctx, name string) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) List(ctx) ([]*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) UpdateDraft(ctx, name, yamlDraft string) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) Publish(ctx, name string) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) Delete(ctx, name string) error
func (s *PipelineDefinitionStore) ListPublished(ctx) ([]pipeline.DefinitionRecord, error)
```

All methods follow the existing DAO pattern: nil-safe, ent-generated client calls,
errors wrapped with `%w`. No query code outside `store.go`.

## 2. YAML Schema (Editor Canonical Format)

```yaml
name: example-pipeline
description: Create items from events
enabled: true
resumable: false
triggers:
  - type: event
    enabled: true
    event: item.created
  - type: webhook
    enabled: true
    path: /github-push
    method: POST
    auth:
      token: "xxx"
      hmac_secret: "xxx"
      hmac_header: "X-Hub-Signature-256"
      token_header: "X-Webhook-Token"
    payload: raw
  - type: cron
    enabled: false
    cron: "*/5 * * * *"
    cron_timeout: "10m"
steps:
  - name: create-item
    capability: example
    operation: create
    params:
      title: '{{default "Untitled" event.title}}'
      tags:
        event_id: "{{event.event_id}}"
    retry:
      max_attempts: 3
      delay: "1s"
      max_delay: "30s"
      backoff: "exponential"
      jitter: 0.1
  - name: notify
    capability: notify
    operation: send
    params:
      message: 'Created {{steps.create-item.id}}'
```

### Template syntax

- `{{event.field}}` — event data field
- `{{steps.StepName.field}}` — previous step output
- `{{webhook.payload.field}}` — webhook request body (via jsonpath)
- `{{input.field}}` — pipeline input parameter
- `{{default "fallback" value}}` — fallback when value is empty
- `{{jsonpath event.data "$.items[0].name"}}` — nested JSON extraction

This matches the existing `pkg/pipeline/template/engine.go` syntax. The `||` operator
described in the original spec is not supported; use `{{default ...}}` instead.

## 3. API Routes

All routes under `/service/web/pipelines` within the web module.

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| `GET` | `/pipelines` | `pipelineListPage` | Pipeline list page |
| `GET` | `/pipelines/list` | `pipelineListTable` | HTMX partial: list rows |
| `GET` | `/pipelines/new` | `pipelineEditorPage` | New pipeline canvas |
| `GET` | `/pipelines/:name` | `pipelineEditorPage` | Edit pipeline canvas (loads draft) |
| `POST` | `/pipelines` | `createPipeline` | Create pipeline, redirect to editor |
| `PUT` | `/pipelines/:name` | `updatePipelineDraft` | Auto-save draft YAML |
| `PUT` | `/pipelines/:name/publish` | `publishPipeline` | Publish: copy draft to published |
| `DELETE` | `/pipelines/:name` | `deletePipeline` | Delete pipeline definition |
| `GET` | `/pipelines/:name/yaml` | `getPipelineYaml` | Return current draft YAML |
| `POST` | `/pipelines/:name/test` | `testPipelineStep` | Test: execute up to given step |
| `GET` | `/pipelines/:name/runs` | `pipelineRunsPage` | Run history page |
| `GET` | `/pipelines/:name/runs/list` | `pipelineRunsTable` | HTMX partial: run rows |

### Test endpoint details

`POST /service/web/pipelines/:name/test`

Request:
```json
{
  "trigger_source": "event",
  "mock_payload": {
    "event_id": "mock-001",
    "title": "Test Item"
  },
  "up_to_step_index": 1
}
```

Response (success):
```json
{
  "success": true,
  "steps": [
    {
      "name": "create-item",
      "status": "ok",
      "duration_ms": 45,
      "output": { "id": "abc123" },
      "rendered_params": {
        "title": "Test Item",
        "tags": { "event_id": "mock-001" }
      }
    }
  ]
}
```

Response (error):
```json
{
  "success": false,
  "error": "Step create-item failed",
  "steps": [ ... ]
}
```

Test execution uses `ability.Invoke()` but does not persist run records, audit events,
or publish events. Timeout: 30 seconds.

### Error response format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "pipeline validation failed",
    "details": [
      { "path": "steps[0].params.title", "message": "Upstream variable {{steps.foo.bar}} does not exist" }
    ]
  }
}
```

HTTP status codes: 400 (validation), 404 (not found), 409 (name conflict), 422 (publish validation), 500 (internal).

## 4. Frontend Architecture

### File layout (within `internal/modules/web/`)

```
internal/modules/web/
├── module.go                    # existing, add pipeline route registration
├── webservice.go                # existing, add pipeline webservice rules
├── pipeline_webservice.go       # NEW: all pipeline HTTP handlers
├── pipeline_templates/          # NEW: templ templates
│   ├── pipeline_list.templ      #   list page + table partial
│   ├── pipeline_editor.templ    #   canvas page (Alpine.js component)
│   ├── pipeline_runs.templ      #   run history page
│   └── pipeline_partials.templ  #   trigger cards, step cards, drawer partials
public/
├── js/
│   └── pipeline-editor.js       # NEW: Alpine.js canvas component
```

### Page structure

```
┌──────────────────────────────────────────────────────────┐
│ Header: [Pipeline Name] [Draft ▾] [Undo] [Redo]        │
│         [</> Code] [Run History] [Save Draft] [Publish]│
├──────────────────────────────────────────────────────────┤
│ Trigger Zone (dashed border, subtle background)        │
│ ┌──────────────────────────┐  [switch: on]             │
│ │ Event: item.created      │                           │
│ └──────────────────────────┘                           │
│               — OR —                                    │
│ ┌──────────────────────────┐  [switch: on]             │
│ │ Webhook: /github-push    │                           │
│ └──────────────────────────┘                           │
│         [+ Add Trigger]                                │
├──────────────────────────────────────────────────────────┤
│                    │                                     │
│               [ + ]  (add step button between steps)    │
│                    │                                     │
│ ┌──────────────────────────────┐                        │
│ │ [icon] create-item      [...]│ (hover: ↑ ↓ copy del) │
│ │ Title: {{default "Unt...    │                        │
│ │ Tags: event_id={{event...   │                        │
│ └──────────────────────────────┘                        │
│                    │                                     │
│               [ + ]                                     │
│                    │                                     │
│ ┌──────────────────────────────┐                        │
│ │ [icon] notify           [...]│                        │
│ │ Message: Created {{step...  │                        │
│ └──────────────────────────────┘                        │
│                              ┌──────────────────────┐   │
│                              │ Config Drawer        │   │
│                              │ [⤢ Expand]          │   │
│                              │ [Setup] [Test]      │   │
│                              │                      │   │
│                              │ Step parameter form  │   │
│                              │ [ {x} Variable ]     │   │
│                              └──────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### Alpine.js component state

```js
{
  // Pipeline metadata
  name: 'example-pipeline',
  description: '',
  status: 'draft',
  dirty: false,
  
  // Undo/redo
  undoStack: [],
  redoStack: [],
  
  // Data
  triggers: [],
  steps: [],
  
  // UI state
  selectedNode: null,       // { type: 'trigger'|'step', index: N }
  drawerOpen: false,
  drawerExpanded: false,
  drawerTab: 'setup',       // 'setup' | 'test'
  drawerDirty: false,       // params modified, prompt on close
  codeView: false,
  yamlText: '',
  
  // Variable picker
  variablePickerOpen: false,
  variablePickerTarget: null,
  variablePickerSource: 'event',
  
  // Validation
  errors: [],
  publishDisabled: false,
  
  // Auto-save
  autoSaveTimer: null,
}
```

### Key interactions

**Variable pill**: Rendered inside `contenteditable` divs as `<span contenteditable="false" class="var-pill">{{event.title}}</span>`. Backspace deletes the entire pill atomically. Max width 150px with ellipsis; hover shows full tooltip.

**Cascade error check**: After any add/remove/move, scan all steps. For each `{{steps.X.Y}}` reference, verify step X exists at a lower index. Flag affected cards with red border and affected fields with red error text.

**Empty variable warning**: For each `{{event.*}}` reference, check if any trigger provides that field. If a disabled trigger exists that would provide it, show yellow warning icon with tooltip: "This variable may be empty under some trigger sources. Set a fallback value."

**Reordering safety**: Before moving step up, check if that step references any step between target and current position. Reject with toast: "Cannot move: this step depends on data from [Step X] which is above the target position."

**Code view**: Toggle serializes Alpine state to YAML. On switch back, parse YAML, validate structure, load into state. Block switch if YAML parse fails: "YAML syntax error. Fix errors before switching back to visual mode."

**Drawer dirty-state**: When params modified and backdrop clicked or Escape pressed, confirm: "You have unsaved changes. Discard them?"

**Auto-save**: Debounced `PUT /pipelines/:name` every 30 seconds. Shows "Saved" indicator briefly, then "Draft" with timestamp.

**Undo/redo**: Push full state snapshot to undo stack on each mutation (add/remove/move/edit). Ctrl+Z / Ctrl+Y triggers restore. Stack limited to 50 entries.

## 5. Validation

### Client-side (runs on every state change)

| Check | Trigger | UI effect |
|-------|---------|-----------|
| At least 1 enabled trigger | Publish button | Disabled: "At least one trigger must be enabled" |
| At least 1 step | Publish button | Disabled: "At least one step is required" |
| `{{steps.X.Y}}` references deleted step X | Card + field | Red border, "Upstream variable {{steps.X.Y}} is invalid or has been removed" |
| `{{steps.X.Y}}` references step X at higher index | Card + field | Red border, "Depends on [Step X] which must be above this step" |
| `{{event.field}}` used, no enabled trigger provides it | Pill inline | Yellow warning icon, "May be empty under active triggers" |
| Step capability/operation not set | Card + field | Red border, "Capability and operation are required" |
| Step name is empty | Card | Red border, "Step name is required" |
| Cron expression invalid | Trigger card | Red border, "Invalid cron expression" |
| Webhook path empty | Trigger card | Red border, "Webhook path is required" |
| Webhook auth: both token and hmac empty | Trigger card | Red border, "At least one auth method is required" |

### Server-side

On `PUT /pipelines/:name` and `PUT /pipelines/:name/publish`:
- Parse and validate YAML structure
- On publish: validate cron expressions via `go-cron` parser
- On publish: check for duplicate pipeline names
- On publish: verify all referenced capabilities exist via `hub.Registry`

## 6. Engine Integration

### Definition loading (dual path)

```go
// pkg/pipeline/loader.go — new interface and loader
type DefinitionReader interface {
    ListPublished(ctx context.Context) ([]DefinitionRecord, error)
}

type DefinitionRecord struct {
    Name        string
    Description string
    YAML        string
    UpdatedAt   time.Time
}

func LoadFromDB(ctx context.Context, store DefinitionReader) ([]Definition, error)
```

### Startup merge (in `internal/server/pipeline.go`)

```go
func initPipeline() {
    fileDefs := pipeline.LoadConfig(config.App.Pipelines)
    dbDefs, _ := pipeline.LoadFromDB(ctx, pipelineDefStore)
    defs := mergeDefinitions(fileDefs, dbDefs) // DB overrides files on name conflict
    engine := pipeline.NewEngine(defs, ...)
}
```

### Multi-trigger expansion

The editor YAML stores triggers as an array. Engine expands each enabled trigger into its own `Definition` with the same steps. A pipeline with 2 enabled triggers produces 2 engine definitions, each with an independent `Trigger` struct.

Runtime matching:
- `DataEvent.EventType == "item.created"` matches the event-triggered definition
- Webhook `POST /webhook/gh` matches the webhook-triggered definition
- Cron `*/5 * * * *` matches the cron-triggered definition

## 7. Editor vs Engine Types

The editor works with its own YAML schema (multiple `triggers` array, editor-level struct).
The engine's `Definition` type retains a single `Trigger` field — unchanged.

Translation happens in `pkg/pipeline/loader.go`:
- Editor YAML is parsed into an editor-level `EditorDefinition` struct (with `[]TriggerEntry`)
- `expandDefinitions()` fans out each enabled trigger entry into a separate engine `Definition`
- A pipeline with 1 Event + 1 Webhook trigger becomes 2 engine `Definition` instances, same name and steps

## 8. Navigation

Add "Pipelines" link to `pkg/views/layout/base.templ` nav bar, next to the existing "Configs" link:

```html
<a href="/service/web/pipelines" data-testid="nav-pipelines" class="hover:text-gray-900">Pipelines</a>
```

Position: between "Flowbot" brand link and "Configs". The nav becomes: Flowbot | Pipelines | Configs | Logout.

## 9. Conventions

- All text in English (labels, errors, tooltips, confirmations)
- Page transitions via HTMX (list ↔ editor, list ↔ runs)
- Canvas interactivity via Alpine.js (triggers, steps, drawer, pills, undo/redo)
- `data-testid` attributes on all interactive elements
- Tailwind CSS for styling; only `.var-pill` needs custom atomic-block CSS
- `templ` for server-rendered HTML (list page, runs page, empty states)
- Store methods in `store.go` only, all queries via ent
- Route handlers in `pipeline_webservice.go`, auth via `route.WithAuthRequired()`
- Test data via `internal/modules/web/pipeline_webservice_test.go` (TDD table-driven)

## 10. Test Coverage

### Unit tests (TDD)
- Store: Create, GetByName, List, UpdateDraft, Publish, Delete, ListPublished
- Handlers: create, update, publish, delete, test execution, YAML get
- Validation: client-side rules via Alpine.js test harness, server-side rules

### BDD tests (Ginkgo v2)
- Full pipeline lifecycle: create → edit → add triggers/steps → test → publish
- Draft auto-save and recovery
- Error cascade on step deletion
- Publish rejection with invalid config

### E2E tests (Go-rod)
- List page: create, delete pipelines
- Editor: add triggers, add steps, configure params, variable picker
- Publish workflow
- Error states (red cards, disabled publish button)
- Code view toggle and YAML validation

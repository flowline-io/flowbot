# Pipeline CRUD Web UI Design

**Date**: 2026-05-30
**Status**: Draft

## Overview

Add a visual pipeline editor to the web UI with CRUD operations, draft/publish lifecycle,
and a step-based canvas interface. Users create, edit, test, and publish pipeline definitions
through a browser UI instead of editing YAML files directly.

## Decisions

| Decision           | Choice                                                                        |
| ------------------ | ----------------------------------------------------------------------------- |
| Frontend approach  | templ + HTMX + Alpine.js, fixed vertical sequence (no free-form drag)         |
| Storage            | Database-backed (`pipeline_definitions` table) with draft + published columns |
| Module location    | Extend existing `internal/modules/web/`                                       |
| Scope (v1)         | Definitions CRUD + read-only run history                                      |
| Trigger schema     | Multiple triggers array with OR logic                                         |
| Branching / Router | Out of scope for V1 (strict linear steps only)                                |
| All UI text        | English only                                                                  |

## 1. Database Schema

### pipeline_definitions table

```sql
CREATE TABLE pipeline_definitions (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    description    TEXT DEFAULT '',
    yaml_draft     TEXT NOT NULL DEFAULT '',
    yaml_published TEXT DEFAULT '',
    version        INT NOT NULL DEFAULT 1,
    status         TEXT NOT NULL DEFAULT 'draft',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT name_format CHECK (name ~ '^[a-z0-9][a-z0-9_-]*$')
);
```

- `name`: lowercase alphanumeric, must start with letter or digit, allows `_` and `-`. Validated both at DB level and API level.
- `version`: optimistic locking counter, incremented on every `UpdateDraft` and `Publish`. Included in PUT request body; server rejects with 409 if version mismatch.
- `yaml_draft`: current working copy, auto-saved every 30s
- `yaml_published`: set when user publishes, copied from draft
- `status`: 'draft' when no published version exists or draft differs from published; 'published' when draft matches published

### Store interface (in `internal/store/store.go`)

```go
type PipelineDefinitionStore struct { client *gen.Client }

func (s *PipelineDefinitionStore) Create(ctx, name, description string) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) GetByName(ctx, name string) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) List(ctx) ([]*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) UpdateDraft(ctx, name, yamlDraft string, version int) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) Publish(ctx, name string, version int) (*gen.PipelineDefinition, error)
func (s *PipelineDefinitionStore) DeleteByName(ctx, name string) (int64, error) // returns count of deleted runs
func (s *PipelineDefinitionStore) ListPublished(ctx) ([]pipeline.DefinitionRecord, error)
```

`UpdateDraft` and `Publish` accept a `version` parameter and use a conditional UPDATE (`WHERE version = $version`). If the version does not match (row not found), return `types.ErrConflict`. The `version` is auto-incremented on success.

`DeleteByName` removes the definition and all associated pipeline runs (ON DELETE CASCADE or explicit cleanup). Returns the count of deleted runs so the UI can show: "Delete this pipeline? 1,234 associated run records will also be removed."

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
      message: "Created {{steps.create-item.id}}"
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

| Method   | Path                         | Handler               | Description                                    |
| -------- | ---------------------------- | --------------------- | ---------------------------------------------- |
| `GET`    | `/pipelines`                 | `pipelineListPage`    | Pipeline list page                             |
| `GET`    | `/pipelines/list`            | `pipelineListTable`   | HTMX partial: list rows                        |
| `GET`    | `/pipelines/new`             | `pipelineEditorPage`  | New pipeline canvas                            |
| `GET`    | `/pipelines/:name`           | `pipelineEditorPage`  | Edit pipeline canvas (loads draft)             |
| `POST`   | `/pipelines`                 | `createPipeline`      | Create pipeline, redirect to editor            |
| `PUT`    | `/pipelines/:name`           | `updatePipelineDraft` | Auto-save draft YAML (body: `{yaml, version}`) |
| `PUT`    | `/pipelines/:name/publish`   | `publishPipeline`     | Publish (body: `{version}`)                    |
| `DELETE` | `/pipelines/:name`           | `deletePipeline`      | Delete pipeline definition + cascade runs      |
| `GET`    | `/pipelines/:name/yaml`      | `getPipelineYaml`     | Return current draft YAML                      |
| `GET`    | `/pipelines/:name/mock`      | `getMockPayload`      | Get sample payload for test (see below)        |
| `POST`   | `/pipelines/:name/test`      | `testPipelineStep`    | Test: execute up to given step                 |
| `GET`    | `/pipelines/:name/runs`      | `pipelineRunsPage`    | Run history page                               |
| `GET`    | `/pipelines/:name/runs/list` | `pipelineRunsTable`   | HTMX partial: run rows                         |

### Mock payload endpoint

`GET /service/web/pipelines/:name/mock?source=webhook`

Returns a sample payload for the selected trigger source so the test drawer can pre-fill the mock JSON instead of showing a blank textarea.

Query params:

- `source` (required): `event`, `webhook`, or `cron`

Response (webhook source):

```json
{
  "source": "webhook",
  "payload": {
    "event_id": "mock-wb-001",
    "title": "Sample webhook payload",
    "body": {}
  },
  "note": "Edit fields above to customize your test data."
}
```

Response (event source with `type` param):

```json
{
  "source": "event",
  "payload": {
    "event_id": "mock-ev-001",
    "event_type": "item.created",
    "title": "",
    "entity_id": "",
    "source": "",
    "capability": "example",
    "operation": "create"
  },
  "note": "Generated from event schema. Fill in values to match your expected data."
}
```

Response (cron source):

```json
{
  "source": "cron",
  "payload": {},
  "note": "Cron-triggered pipelines have no event payload. Steps can only reference {{steps.*}} and {{input.*}}."
}
```

The mock endpoint generates payloads based on:

- **Webhook**: Returns the structure with common webhook fields; if a recent webhook run exists, returns its actual payload as a template.
- **Event**: Looks up the event type in the hub registry and generates a JSON skeleton with all known fields from the `DataEvent` struct.
- **Cron**: Returns an empty payload (cron has no event data).

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

Test execution uses `capability.Invoke()` but does not persist run records, audit events,
or publish events. Timeout: 30 seconds.

### Error response format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "pipeline validation failed",
    "details": [
      {
        "path": "steps[0].params.title",
        "message": "Upstream variable {{steps.foo.bar}} does not exist"
      }
    ]
  }
}
```

HTTP status codes: 400 (validation), 404 (not found), 409 (name conflict or optimistic lock version mismatch), 422 (publish validation), 500 (internal).

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
│   ├── pipeline-editor.js       # NEW: Alpine.js canvas component
│   └── pipeline-pill-editor.js  # NEW: Mention.js / tag-pill input component
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
  version: 1,
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

**Variable pill**: Input fields that accept template variables use a lightweight mention/tag library instead of raw `contenteditable`. Using `contenteditable` with Alpine.js causes caret position loss and DOM thrashing on re-render. The recommended approach:

- Use a library like **Tribute.js** (no dependencies, ~10KB) or a minimal custom `<input-tag>` web component that renders pills as non-editable inline tokens
- Alpine.js manages the data model (the raw template string with `{{...}}` markers) and the input field's value
- The pill library handles rendering: tokens appear as styled capsules; backspace deletes the entire capsule atomically; typing `{` triggers the variable picker (or clicking the `{x}` button)
- Alpine receives the final string value (e.g., `{{default 'Untitled' event.title}}`), never manipulates the pill DOM directly

The pill library is loaded as a separate JS bundle in `public/js/pipeline-pill-editor.js`. The Alpine component binds to it via `x-init` and a custom directive.

**Variable pill styling**: Max width 150px with ellipsis on pill text; hover shows full tooltip. Display format: `{{event.title}}` rendered as blue capsule.

**Cascade error check**: After any add/remove/move, scan all steps. For each `{{steps.X.Y}}` reference, verify step X exists at a lower index. Flag affected cards with red border and affected fields with red error text.

**Empty variable warning**: For each `{{event.*}}` reference, check if any trigger provides that field. If a disabled trigger exists that would provide it, show yellow warning icon with tooltip: "This variable may be empty under some trigger sources. Set a fallback value."

**Reordering safety**: Before moving step up, check if that step references any step between target and current position. Reject with toast: "Cannot move: this step depends on data from [Step X] which is above the target position."

**Code view**: Toggle serializes Alpine state to YAML. On switch back, parse YAML, validate structure, load into state. Block switch if YAML parse fails: "YAML syntax error. Fix errors before switching back to visual mode."

**Drawer dirty-state**: When params modified and backdrop clicked or Escape pressed, confirm: "You have unsaved changes. Discard them?"

**Auto-save**: Debounced `PUT /pipelines/:name` every 30 seconds, sending `{yaml, version}` in body. On success, update local `version` to the returned value. On 409 response, show: "This draft was modified elsewhere. Please refresh the page."

**Undo/redo**: Push full state snapshot to undo stack on each mutation (add/remove/move/edit). Ctrl+Z / Ctrl+Y triggers restore. Stack limited to 50 entries.

## 5. Validation

### Client-side (runs on every state change)

| Check                                                  | Trigger        | UI effect                                                                    |
| ------------------------------------------------------ | -------------- | ---------------------------------------------------------------------------- |
| At least 1 enabled trigger                             | Publish button | Disabled: "At least one trigger must be enabled"                             |
| At least 1 step                                        | Publish button | Disabled: "At least one step is required"                                    |
| `{{steps.X.Y}}` references deleted step X              | Card + field   | Red border, "Upstream variable {{steps.X.Y}} is invalid or has been removed" |
| `{{steps.X.Y}}` references step X at higher index      | Card + field   | Red border, "Depends on [Step X] which must be above this step"              |
| `{{event.field}}` used, no enabled trigger provides it | Pill inline    | Yellow warning icon, "May be empty under active triggers"                    |
| Step capability/operation not set                      | Card + field   | Red border, "Capability and operation are required"                          |
| Step name is empty                                     | Card           | Red border, "Step name is required"                                          |
| Cron expression invalid                                | Trigger card   | Red border, "Invalid cron expression"                                        |
| Webhook path empty                                     | Trigger card   | Red border, "Webhook path is required"                                       |
| Webhook auth: both token and hmac empty                | Trigger card   | Red border, "At least one auth method is required"                           |

### Server-side

On `POST /pipelines`:

- Validate `name` matches `^[a-z0-9][a-z0-9_-]*$` (lowercase, starts with alnum, alnum + `_-` only)
- Return 400 if name is invalid: "Pipeline name must start with a letter or digit and contain only lowercase letters, digits, hyphens, and underscores."

On `PUT /pipelines/:name` and `PUT /pipelines/:name/publish`:

- Compare `version` with current DB version; return 409 if mismatch: "This draft was modified elsewhere. Please refresh the page."
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

The editor YAML stores triggers as an array. The engine's internal registry keys definitions by name, which would collide if two fan-out definitions share the same parent name. To avoid this, `expandDefinitions()` generates compound names by appending a trigger-type suffix:

```go
func expandDefinitions(defs []EditorDefinition) []Definition {
    var expanded []Definition
    for _, d := range defs {
        for i, t := range d.Triggers {
            if !t.Enabled {
                continue
            }
            compoundName := fmt.Sprintf("%s__trigger_%s_%d", d.Name, t.Type, i)
            expanded = append(expanded, Definition{
                Name:        compoundName,
                Description: d.Description,
                Enabled:     d.Enabled,
                Resumable:   d.Resumable,
                Trigger:     t.ToEngineTrigger(),
                Steps:       d.Steps,
                ParentName:  d.Name, // for run history mapping
            })
        }
    }
    return expanded
}
```

Runtime matching uses the compound names internally. When recording pipeline runs (in `PipelineStore`), the run record includes both the compound engine name and the original `parent_name` field. Run history queries match by `parent_name` to aggregate all trigger variants under the user-facing pipeline name.

Example: `example-pipeline` with Event + Webhook triggers becomes:

- `example-pipeline__trigger_event_0` (engine name)
- `example-pipeline__trigger_webhook_1` (engine name)
- Both store `parent_name: "example-pipeline"` in run records

## 7. Editor vs Engine Types

The editor works with its own YAML schema (multiple `triggers` array, editor-level struct).
The engine's `Definition` type retains a single `Trigger` field — unchanged, but gains a `ParentName` field for run history aggregation.

Translation happens in `pkg/pipeline/loader.go`:

- Editor YAML is parsed into an editor-level `EditorDefinition` struct (with `[]TriggerEntry`)
- `expandDefinitions()` fans out each enabled trigger entry into a separate engine `Definition` with compound names (see section 6)
- A pipeline with 1 Event + 1 Webhook trigger becomes 2 engine `Definition` instances
- Run history queries use `parent_name` to aggregate results back to the user-facing pipeline name

## 8. Navigation

Add "Pipelines" link to `pkg/views/layout/base.templ` nav bar, next to the existing "Configs" link:

```html
<a
  href="/service/web/pipelines"
  data-testid="nav-pipelines"
  class="hover:text-gray-900"
  >Pipelines</a
>
```

Position: between "Flowbot" brand link and "Configs". The nav becomes: Flowbot | Pipelines | Configs | Logout.

## 9. Conventions

- All text in English (labels, errors, tooltips, confirmations)
- Page transitions via HTMX (list ↔ editor, list ↔ runs)
- Canvas interactivity via Alpine.js (triggers, steps, drawer, pills, undo/redo)
- `data-testid` attributes on all interactive elements
- Tailwind CSS for general styling; pill rendering handled by the tag-input library
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

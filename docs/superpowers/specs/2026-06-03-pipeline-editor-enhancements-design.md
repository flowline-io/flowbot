# Pipeline Editor Enhancements

2026-06-03 | Status: Approved

## Overview

Four UX improvements to the pipeline editor (Alpine.js SPA): drag-and-drop step reordering, YAML file import/export buttons, a publish-version history sidebar with snapshot viewing, and a two-version YAML diff comparison view.

## Motivation

The current editor requires clicking Move Up/Move Down buttons to reorder steps. There is no way to export/import pipeline definitions as files. Published changes overwrite the previous version with no history to review or compare. These are standard expectations for a pipeline editor and are blocking adoption.

## Summary

Four enhancements to the pipeline editor:

1. Drag-and-drop step reordering (Alpine Sort plugin)
2. YAML file import/export (client-side)
3. Version history sidebar with snapshot on publish
4. Version diff (client-side js-diff, two-version comparison)

Conditional branch visualization is explicitly out of scope.

## Decision Log

| Decision | Rationale |
|----------|-----------|
| Drag: Alpine Sort plugin | Consistent with existing Alpine.js architecture, CDN delivery |
| Diff: client-side js-diff | No extra API, instant rendering after both YAMLs are loaded |
| Version: publish-only snapshots | Draft saves are frequent and low-value; publish is the meaningful checkpoint |
| Import/export: client-side only | js-yaml already loaded; no new backend endpoints needed |
| No conditional branches | Engine does not support it; requires separate design |

---

## 1. Database Schema

### New table: `pipeline_definition_versions`

| Column | Type | Constraints |
|--------|------|-------------|
| id | int64 | PK, auto-increment |
| pipeline_name | string | NOT NULL, FK to pipeline_definitions.name |
| version | int | NOT NULL |
| yaml | text | NOT NULL |
| created_at | timestamp | NOT NULL, default now |

- Unique index on `(pipeline_name, version)`
- Ent schema file: `internal/store/ent/schema/pipeline_definition_version.go`
- Ent auto-migration handles table creation

### Modified logic: `PublishDefinition`

After the existing `UPDATE pipeline_definitions SET yaml_published=..., version=version+1`, insert a row into `pipeline_definition_versions`:

```sql
INSERT INTO pipeline_definition_versions (pipeline_name, version, yaml, created_at)
VALUES ($1, $2, $3, $4)
```

Where `$2` is the new version number (post-increment), `$3` is the published YAML content.

No changes to the `pipeline_definitions` table itself.

---

## 2. Store Layer

### New methods on `PipelineStore`

```go
// ListDefinitionVersions returns all published version snapshots for a pipeline,
// ordered by version descending (newest first).
func (s *PipelineStore) ListDefinitionVersions(ctx context.Context, name string) ([]*gen.PipelineDefinitionVersion, error)

// GetDefinitionVersion returns a single version snapshot by pipeline name and version number.
func (s *PipelineStore) GetDefinitionVersion(ctx context.Context, name string, version int) (*gen.PipelineDefinitionVersion, error)
```

- `ListDefinitionVersions` returns all columns; frontend uses `version` and `created_at` for the list display
- `GetDefinitionVersion` returns the full row including `yaml` for diff and preview
- Both methods follow the existing nil-guard pattern (`if s == nil || s.client == nil`)

### Modified method: `PublishDefinition`

- Keep existing logic: read `yaml_draft`, validate, optimistic UPDATE to set `yaml_published`, bump version, set status
- After successful UPDATE, insert the snapshot into `pipeline_definition_versions`
- If the INSERT fails, return the error — treat it as part of the publish transaction (a failed version snapshot means the publish is incomplete and should be retried)
- Use the version number AFTER the update (i.e., what `PublishDefinition` already sets on the definition row)

Note: Ent does not support cross-table transactions natively in the generated API. The INSERT after UPDATE is an eventual-consistency approach. If the INSERT fails, the user will see a publish error and retry. A subsequent successful publish will create the version correctly.

---

## 3. API Layer

### New routes (added to `pipelineWebserviceRules`)

| Method | Path | Handler | Request | Response |
|--------|------|---------|---------|----------|
| GET | `/pipelines/:name/versions` | `listPipelineVersions` | — | `[{version: int, created_at: string}]` |
| GET | `/pipelines/:name/versions/:version` | `getPipelineVersion` | — | `{yaml: string, version: int, created_at: string}` |

- Both return JSON (`c.JSON(...)`)
- Both call `getPipelineDefStore()` then `ListDefinitionVersions` / `GetDefinitionVersion`
- 404 if pipeline or version not found
- Standard error wrapping patterns from existing handlers

### No new routes for import/export

- Download: client-side Blob + anchor click
- Upload: client-side FileReader + existing `PUT /pipelines/:name`

---

## 4. Frontend

### 4.1 Drag-and-Drop Step Reordering

**Library**: Alpine Sort plugin via CDN (`@alpinejs/sort`)

**Changes to `pipeline-editor.js`**:
- Remove `moveStepUp()`, `moveStepDown()` methods
- On drag end: call `pushUndo()` + `markDirty()` + `validate()`

**Changes to `pipeline_editor.templ`**:
- Steps zone `x-for` gains `x-sort` directive and unique item key
- Remove the Up/Down arrow buttons from `StepCard` partial

**Changes to `pipeline_partials.templ`** (`StepCard`):
- Remove the Move Up and Move Down buttons (the `<span>` buttons with arrow icons)

**Alpine Sort** data attribute approach:
- Add `x-sort:item` on each step card element
- Add `x-sort:group="steps"` on the container
- On sort end, Alpine Sort reorders the data array automatically

### 4.2 YAML Import/Export

**Export (Download)**:
- Header toolbar gains a "Download" button
- Click handler: `stateToYaml()` → `new Blob([yaml], {type: 'application/x-yaml'})` → `URL.createObjectURL(blob)` → programmatic `<a download>` click → `URL.revokeObjectURL()`
- File name: `{this.name}.yaml`

**Import (Upload)**:
- Hidden `<input type="file" accept=".yaml,.yml">` in the template
- Header toolbar gains an "Import" button that triggers `input.click()`
- `onchange` handler: `FileReader.readAsText()` → `js-yaml.load()` validation → `parseYamlToState()` → `markDirty()` + `pushUndo()`
- If validation fails, show toast error, do not overwrite state
- After successful import, auto-save is triggered by `markDirty()`

### 4.3 Version History Sidebar

**Layout**: Collapsible sidebar on the right side of the editor, toggled by a "History" button in the header.

**Data flow**:
1. Page init calls `fetch /pipelines/:name/versions`
2. Stores list in Alpine state: `versions: [{version, created_at}, ...]`
3. Sidebar renders a scrollable list

**Version list item display**:
- `v{version}` badge
- Relative timestamp (e.g., "2 hours ago", "3 days ago") — computed in JS
- Click to load version content

**Version preview**:
- Clicking a version calls `fetch /pipelines/:name/versions/:version`
- YAML content displayed in a read-only `<pre>` or `<textarea readonly>` in the sidebar
- Current editor state is NOT modified; this is view-only
- A "Restore" button could be added: copies the version YAML into `yamlText` / `parseYamlToState()` → `markDirty()` + `pushUndo()`

**Sidebar state**:
- `historyOpen: false` — collapsed by default
- `versions: []` — version list
- `selectedVersion: null` — currently viewed version
- `selectedVersionYaml: ''` — YAML content of selected version

### 4.4 Version Diff

**Library**: `diff` (npm package `diff`, served via CDN or vendored)

**Mode**: Toggled by a "Compare" button in the version history sidebar.

**UI flow**:
1. Click "Compare" in sidebar → enters compare mode
2. Each version row gains a checkbox
3. Select exactly two versions → diff is computed and rendered below the list
4. Click "Exit Compare" to go back to normal view mode

**Diff rendering**:
- Use `Diff.diffLines(oldYaml, newYaml)` from the `diff` library
- Render each diff part with CSS classes:
  - `diff-added` (green background) for added lines
  - `diff-removed` (red background) for removed lines
  - No highlight for unchanged lines
- Display as a monospace `<pre>` block with inline spans

**State additions**:
- `compareMode: false`
- `compareLeft: null`, `compareRight: null` — selected version numbers
- `diffResult: []` — diff chunks

### 4.5 Static Vendor Dependencies

New vendored libraries to add under `public/vendor/` (follows existing pattern for alpine.min.js, js-yaml.min.js, chart.js.min.js):

| File | Source | Loaded in |
|------|--------|-----------|
| `alpine-sort.min.js` | Alpine Sort plugin (custom directive or plugin CDN) | `base.templ` (global, like alpine.min.js) |
| `diff.min.js` | `diff` library (kpdecker/diff) | `pipeline_editor.templ` (pipeline-specific, like js-yaml) |

Script tags:
```html
<!-- base.templ -->
<script src="/static/vendor/alpine-sort.min.js" defer></script>

<!-- pipeline_editor.templ -->
<script src="/static/vendor/diff.min.js"></script>
```

If the Alpine Sort plugin is not available as a standalone CDN file, implement a minimal custom `x-sort` Alpine directive in `pipeline-editor.js` using the native HTML Drag and Drop API, avoiding an extra dependency.

---

## 5. Implementation Order

1. **Database + Store**: New schema, ent generate, new store methods, modify `PublishDefinition`
2. **API**: Two new version history endpoints
3. **Frontend - Drag & Drop**: Alpine Sort plugin, remove move buttons, remove move helpers
4. **Frontend - Import/Export**: Download/Import buttons, FileReader, Blob
5. **Frontend - Version History**: Sidebar with version list and preview
6. **Frontend - Version Diff**: Compare mode with js-diff library

---

## 6. Testing

### Store tests (TDD, table-driven)

- `TestPipelineStore_Versions`: create definition, publish → verify version row exists; publish again → verify both versions exist with correct YAML content and version numbers
- `TestPipelineStore_ListDefinitionVersions`: multiple publishes → list returns correct count and order (descending)
- `TestPipelineStore_GetDefinitionVersion`: fetch specific version → verify YAML content matches
- Edge cases: no versions (never published), version not found, publish after delete

### Handler tests

- `TestListPipelineVersions`: happy path, empty list, pipeline not found (404)
- `TestGetPipelineVersion`: valid version, version not found (404), pipeline not found (404)

### Frontend tests (via BDD specs if applicable)

- Drag-and-drop reorder triggers undo push and auto-save
- Import valid YAML updates state; import invalid YAML shows error toast
- Export creates downloadable .yaml file
- Version history sidebar loads and displays versions
- Diff mode renders line differences correctly

---

## 7. Out of Scope

- Conditional branch visualization (requires engine-level changes)
- Version rollback/restore (optional "Restore" button noted but not required)
- Draft version snapshots (publish-only)
- Server-side diff endpoint
- Drag-and-drop for triggers (steps only)
- Multi-step undo across drag operations (single undo per drag)

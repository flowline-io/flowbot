# Pipeline Node Setup: Params Linkage

**Date**: 2026-05-31
**Status**: Approved

## Problem

In the pipeline editor's node setup drawer, selecting a capability populates the operation dropdown (linkage works). Selecting an operation does nothing to the params textarea. Users must manually type JSON -- they have no guidance on what parameters each operation expects.

## Solution

When the user changes capability or operation, the params textarea auto-populates with a JSON template derived from the operation's `Input` ParamDefs, using type-hint values. Auto-population only occurs when the current params text is "default-like" (empty, `{}`, or matching a known template), preserving any manual edits.

## Design

### Backend: Populate Input ParamDefs

Add `Input []hub.ParamDef` to every operation in all capability descriptors (bookmark, forge, kanban, reader, github, note, memo). Notify already has them.

ParamDef type values: `"string"`, `"int"`, `"int64"`, `"bool"`, `"[]string"`, `"map[string]any"`.

Pagination operations include `limit`, `cursor`, `sort_by`, `sort_order` as optional params.

### Frontend: Template Generation (pipeline-editor.js)

New methods on the Alpine.js `pipelineEditor` component:

| Method | Purpose |
|--------|---------|
| `getOperation(capType, opName)` | Find the full operation object from cached capabilities |
| `getDefaultParams(capType, opName)` | Build a JSON template string from operation Input ParamDefs |
| `typeDefaultValue(type)` | Map type strings to JSON value hints (`"<string>"` -> `"\"<string>\""`) |
| `isParamsDefault(paramsText)` | True if empty, `"{}"`, or matches a known generated template |
| `onCapabilityChange()` | Called by capability select `@change` -- sets operation + conditionally sets paramsText |
| `onOperationChange()` | Called by operation select `@change` -- conditionally sets paramsText |

### Frontend: Template Changes (pipeline_editor.templ)

Capability `<select>` @change:
- Was: inline expression setting operation to first available
- Now: calls `onCapabilityChange()`

Operation `<select>` @change:
- Was: `drawerDirty = true`
- Now: calls `onOperationChange()`

### Edge Cases

| Scenario | Behavior |
|----------|----------|
| Capability changes, params was default | Params update to new operation's template |
| Capability changes, params has manual edits | Params preserved as-is |
| Only operation changes, params was default | Params update |
| Operation has no Input params | Params set to `{}` |
| Step just added (paramsText = "{}") | Treated as default |
| Loading existing pipeline from YAML | Params from YAML preserved |

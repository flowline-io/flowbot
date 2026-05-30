# Pipeline Step — Capability/Operation Cascading Selects

**Date**: 2026-05-30
**Status**: Draft

## Overview

Replace the free-text Capability and Operation text inputs in the pipeline step
configuration drawer with cascading `<select>` dropdowns. Capability options load
from a new pipeline web service endpoint; Operation options filter dynamically
based on the selected capability.

## Decisions

| Decision                       | Choice                                                                             |
| ------------------------------ | ---------------------------------------------------------------------------------- |
| Capability data source         | `GET /service/web/pipelines/capabilities`, calls `hub.Default.List()` server-side  |
| Frontend pattern               | Alpine.js: `fetch()` on init, `getOperationsFor()` method for computed options     |
| Operation on capability change | Reset to first available operation                                                 |
| Auth                           | Inherits pipeline page auth (web-login token with `admin:*`); no scope check added |

## Backend: getCapabilities endpoint

### Route: `GET /service/web/pipelines/capabilities`

Handler calls `hub.Default.List()` and returns descriptors wrapped in
`protocol.NewSuccessResponse(...)`.

### Import addition to `pipeline_webservice.go`

Add `"github.com/flowline-io/flowbot/pkg/hub"` to imports.

### Route rule addition to `pipelineWebserviceRules`

```go
webservice.Get("/pipelines/capabilities", getCapabilities),
```

Note: must be declared BEFORE `/pipelines/:name` to prevent `capabilities`
from matching the `:name` param.

### Handler

```go
func getCapabilities(ctx fiber.Ctx) error {
    return ctx.JSON(protocol.NewSuccessResponse(hub.Default.List()))
}
```

## Frontend: Alpine.js changes

### `pipeline-editor.js` additions

In `pipelineEditor` data function, add:

```javascript
capabilities: [],        // populated on init by fetchCapabilities()

async fetchCapabilities() {
  try {
    const resp = await fetch('/service/web/pipelines/capabilities');
    const json = await resp.json();
    this.capabilities = json.data || [];
  } catch (e) { console.error('Failed to load capabilities:', e); }
},

getOperationsFor(capType) {
  const cap = this.capabilities.find(c => c.type === capType);
  return cap ? (cap.operations || []) : [];
},
```

Call `await this.fetchCapabilities();` in `init()` after loading pipeline data.

### `pipeline_editor.templ` replacements

```html
<select
  x-model="steps[selectedNode.index].capability"
  @change="steps[selectedNode.index].operation = getOperationsFor(steps[selectedNode.index].capability)[0]?.name || ''; drawerDirty = true"
  class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
  data-testid="step-capability-select"
>
  <option value="" disabled>Select capability...</option>
  <template x-for="cap in capabilities" :key="cap.type">
    <option
      :value="cap.type"
      x-text="cap.type"
      :title="cap.description"
    ></option>
  </template>
</select>

<select
  x-model="steps[selectedNode.index].operation"
  @change="drawerDirty = true"
  class="w-full border border-gray-300 rounded px-3 py-2 text-sm mb-3"
  data-testid="step-operation-select"
>
  <option value="" disabled>Select operation...</option>
  <template
    x-for="op in getOperationsFor(steps[selectedNode.index].capability)"
    :key="op.name"
  >
    <option :value="op.name" x-text="op.name" :title="op.description"></option>
  </template>
</select>
```

## Behavior Matrix

| Scenario                 | Capability dropdown                | Operation dropdown                    |
| ------------------------ | ---------------------------------- | ------------------------------------- |
| New step, drawer opens   | "Select capability..." placeholder | "Select operation..." placeholder     |
| Select "bookmark"        | "bookmark" selected                | Auto-selects first bookmark operation |
| Change to "note"         | "note" selected                    | Resets to first note operation        |
| Load existing pipeline   | Pre-selected from YAML             | Pre-selected from YAML                |
| Capability with zero ops | Selected                           | Placeholder, no options               |

## Not Changed

- `parseYamlToState()` / `stateToYaml()` — serialization unchanged
- `addStep()` — defaults `{ capability: '', operation: '' }`
- `validate()` — same non-empty checks
- Step card display — `capability.operation` text unchanged

## Edge Cases

1. **YAML with unknown capability**: Value preserved in model; dropdown shows selected
   value without matching option. User can keep or change it.
2. **Capabilities list empty**: Only placeholder shown. Validation error persists.
3. **Page opened while hub has no registrations**: Dropdown empty; no crash.

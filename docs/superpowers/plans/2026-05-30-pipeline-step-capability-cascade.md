# Pipeline Step — Capability/Operation Cascading Selects Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace free-text capability/operation inputs in the pipeline step editor with cascading `<select>` dropdowns driven by a new web service endpoint.

**Architecture:** New `GET /service/web/pipelines/capabilities` endpoint proxies `hub.Default.List()` server-side. Alpine.js fetches on init and drives capability->operation cascading via `@change`. Three files changed: one Go handler, one JS component, one templ template.

**Tech Stack:** Go (Fiber v3, hub registry), Alpine.js 3.x, templ, js-yaml

---

### File Map

| Action | File                                                                           | Purpose                               |
| ------ | ------------------------------------------------------------------------------ | ------------------------------------- |
| Create | `docs/superpowers/specs/2026-05-30-pipeline-step-capability-cascade-design.md` | Design spec                           |
| Modify | `internal/modules/web/pipeline_webservice.go`                                  | Add `getCapabilities` handler + route |
| Modify | `public/js/pipeline-editor.js`                                                 | Add capabilities fetch + ops lookup   |
| Modify | `internal/modules/web/pipeline_templates/pipeline_editor.templ`                | Replace inputs with selects           |
| Modify | `tests/specs/pipeline_crud_api_spec_test.go`                                   | Add BDD test for new endpoint         |
| Create | `docs/superpowers/plans/2026-05-30-pipeline-step-capability-cascade.md`        | This plan                             |

---

### Task 1: Write spec and plan docs

- [x] Write spec to `docs/superpowers/specs/2026-05-30-pipeline-step-capability-cascade-design.md`
- [x] Write this plan to disk

---

### Task 2: Add `getCapabilities` endpoint (TDD)

**Files:**

- Modify: `internal/modules/web/pipeline_webservice.go`
- Modify: `tests/specs/pipeline_crud_api_spec_test.go`

- [ ] **Step 1: Add BDD test case to `tests/specs/pipeline_crud_api_spec_test.go`**

In `mountPipelineRoutes`, add after line 333 (after `GET /:name/yaml` route):

```go
// GET /service/web/pipelines/capabilities
app.Get("/service/web/pipelines/capabilities", func(c fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "success",
        "data": []fiber.Map{
            {
                "type":        "bookmark",
                "backend":     "native",
                "description": "bookmark management",
                "operations": []fiber.Map{
                    {"name": "list", "description": "list bookmarks"},
                    {"name": "create", "description": "create bookmark"},
                    {"name": "get", "description": "get bookmark"},
                },
            },
        },
    })
})
```

Add a new `Describe` block before `Describe("Pipeline Editor API - CRUD")` (line 31):

```go
Describe("Pipeline Step Capability Selects", func() {
    It("should return capabilities list with operations", func() {
        resp := executeRequest("GET", "/service/web/pipelines/capabilities", nil, map[string]string{
            "Authorization": "Bearer " + testToken,
        })
        Expect(resp.StatusCode).To(Equal(http.StatusOK))

        var body map[string]interface{}
        Expect(sonic.Unmarshal(resp.Body(), &body)).To(Succeed())
        Expect(body["status"]).To(Equal("success"))
        data, ok := body["data"].([]interface{})
        Expect(ok).To(BeTrue())
        Expect(len(data)).To(BeNumerically(">", 0))
        firstCap := data[0].(map[string]interface{})
        Expect(firstCap["type"]).To(Equal("bookmark"))
        operations, ok := firstCap["operations"].([]interface{})
        Expect(ok).To(BeTrue())
        Expect(len(operations)).To(BeNumerically(">", 0))
    })
})
```

- [ ] **Step 2: Run BDD test to verify it fails**

```bash
go tool task test:specs -- --focus="Pipeline Step Capability Selects"
```

Expected: FAIL — route not registered.

- [ ] **Step 3: Add route + handler to `pipeline_webservice.go`**

Add import:

```go
"github.com/flowline-io/flowbot/pkg/hub"
```

Add route BEFORE `/pipelines/:name` rules:

```go
webservice.Get("/pipelines/capabilities", getCapabilities),
```

Add handler:

```go
// getCapabilities returns all registered capabilities with their operations
// for the pipeline editor capability/operation select dropdowns.
func getCapabilities(ctx fiber.Ctx) error {
    return ctx.JSON(protocol.NewSuccessResponse(hub.Default.List()))
}
```

- [ ] **Step 4: Run BDD test to verify it passes**

```bash
go tool task test:specs -- --focus="Pipeline Step Capability Selects"
```

Expected: PASS.

- [ ] **Step 5: Run all BDD tests**

```bash
go tool task test:specs
```

Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/web/pipeline_webservice.go tests/specs/pipeline_crud_api_spec_test.go
git commit -m "feat: add GET /pipelines/capabilities endpoint for step editor selects"
```

---

### Task 3: Add Alpine.js capability fetching and cascading

**File:** Modify `public/js/pipeline-editor.js`

- [ ] **Step 1: Add `capabilities` data property**

After line 12 (`testResults: null,`), add:

```javascript
capabilities: [],
```

- [ ] **Step 2: Add `fetchCapabilities` method after `loadPipeline` (after line 30)**

```javascript
async fetchCapabilities() {
  try {
    const resp = await fetch('/service/web/pipelines/capabilities');
    const json = await resp.json();
    this.capabilities = json.data || [];
  } catch (e) { console.error('Failed to load capabilities:', e); }
},
```

- [ ] **Step 3: Add `getOperationsFor` method after `fetchCapabilities`**

```javascript
getOperationsFor(capType) {
  const cap = this.capabilities.find(c => c.type === capType);
  return cap ? (cap.operations || []) : [];
},
```

- [ ] **Step 4: Call `fetchCapabilities` from `init`**

```javascript
init() {
  const el = this.$el;
  const name = el.dataset.pipelineName || '';
  this.name = name;
  if (name) this.loadPipeline(name);
  this.fetchCapabilities();
  this.pushUndo();
},
```

- [ ] **Step 5: Commit**

```bash
git add public/js/pipeline-editor.js
git commit -m "feat: add capability fetch and operation lookup to pipeline editor"
```

---

### Task 4: Replace templ inputs with selects + regenerate

**Files:**

- Modify: `internal/modules/web/pipeline_templates/pipeline_editor.templ`

- [ ] **Step 1: Replace capability `<input>` (lines 180-183) with `<select>`**

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
```

- [ ] **Step 2: Replace operation `<input>` (lines 184-187) with `<select>`**

```html
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

- [ ] **Step 3: Regenerate templ Go code**

```bash
go tool task templ
```

Expected: No errors.

- [ ] **Step 4: Verify build compiles**

```bash
go tool task build
```

Expected: Compiles successfully.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/web/pipeline_templates/
git commit -m "feat: replace step capability/operation inputs with cascading selects"
```

---

### Task 5: Run lint and all tests

- [ ] **Step 1: Run lint**

```bash
go tool task lint
```

Expected: No new violations.

- [ ] **Step 2: Run unit tests**

```bash
go tool task test
```

Expected: All pass.

- [ ] **Step 3: Run BDD spec tests**

```bash
go tool task test:specs
```

Expected: All pass.

---

### Task 6: Update E2E test for data-testid changes

- [ ] **Step 1: Check for old testid references**

```bash
grep -r 'step-capability-input\|step-operation-input' tests/e2e/ || echo "no matches"
```

If no matches found, skip. Otherwise update to `step-capability-select` / `step-operation-select`.

---

### Task 7: Final commit

```bash
git add docs/superpowers/
git commit -m "docs: add spec and plan for capability cascade selects"
```

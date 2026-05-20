# Pipeline + Workflow Memory Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce per-execution allocations in pipeline and workflow engines via template memoization, sonic buffer pooling, and executor lifecycle cleanup.

**Architecture:** Three independent changes across 3 files (~37 lines total). Template cache uses sync.Map to memoize parsed text/template instances. Sonic pooling replaces bare sonic calls with a frozen Config{PoolAlloc:true} instance. Runner.Close() calls existing executor.Engine.Close() via defer.

**Tech Stack:** Go 1.26+, sync.Map, sonic v2 (`Config.PoolAlloc`), existing `executor.Engine.Close()`

**Spec:** `docs/superpowers/specs/2026-05-18-pipeline-workflow-memory-optimization-design.md`

---

### Task 1: Add template memoization cache

**Files:**

- Modify: `pkg/pipeline/template/engine.go:1-202`
- Test: `pkg/pipeline/template/engine_test.go` (add test at end)

- [ ] **Step 1: Add cache and sync import**

Add `"sync"` to the import block (after `"strings"`).

Add `cache sync.Map` field to `Engine`:

```go
type Engine struct {
	cache sync.Map // string → *txtpl.Template
}
```

---

- [ ] **Step 2: Modify RenderString to use cache**

Replace lines 125-160 (the `RenderString` method body) with:

```go
func (e *Engine) RenderString(tmpl string, data *TemplateData) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	tmpl = preprocessTemplate(tmpl)

	var t *txtpl.Template
	if cached, ok := e.cache.Load(tmpl); ok {
		t = cached.(*txtpl.Template)
	} else {
		var err error
		t, err = txtpl.New("render").Funcs(funcMap(data)).Parse(tmpl)
		if err != nil {
			return "", fmt.Errorf("template parse: %w", err)
		}
		e.cache.Store(tmpl, t)
	}

	tplData := map[string]any{}
	if data != nil {
		if data.Event != nil {
			tplData["Event"] = data.Event
		}
		if data.Steps != nil {
			tplData["Steps"] = data.Steps
		}
		if data.Env != nil {
			tplData["Env"] = data.Env
		}
		if data.Input != nil {
			tplData["Input"] = data.Input
		}
	}

	var buf strings.Builder
	err := t.Execute(&buf, tplData)
	if err != nil {
		return "", fmt.Errorf("template execute: %w", err)
	}

	return buf.String(), nil
}
```

---

- [ ] **Step 3: Run existing template tests**

```bash
go test ./pkg/pipeline/template/... -v -count=1
```

Expected: All existing tests pass.

---

- [ ] **Step 4: Add cache correctness test**

Add to end of `pkg/pipeline/template/engine_test.go`:

```go
func TestRenderString_CacheConsistency(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		template string
		data     *TemplateData
	}{
		{
			name:     "event-field-cached",
			template: `{{event "id"}}`,
			data:     &TemplateData{Event: map[string]any{"id": "42"}},
		},
		{
			name:     "step-field-cached",
			template: `{{step "s1" "result"}}`,
			data:     &TemplateData{Steps: map[string]map[string]any{"s1": {"result": "done"}}},
		},
		{
			name:     "input-field-cached",
			template: `{{input "key"}}`,
			data:     &TemplateData{Input: map[string]any{"key": "val"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New()
			first, err := e.RenderString(tt.template, tt.data)
			require.NoError(t, err)
			second, err := e.RenderString(tt.template, tt.data)
			require.NoError(t, err)
			assert.Equal(t, first, second, "cached and uncached renders must produce identical output")
		})
	}
}
```

---

- [ ] **Step 5: Run tests including new cache test**

```bash
go test ./pkg/pipeline/template/... -v -count=1
```

Expected: All tests pass, including `TestRenderString_CacheConsistency`.

---

- [ ] **Step 6: Commit**

```bash
git add pkg/pipeline/template/engine.go pkg/pipeline/template/engine_test.go
git commit -m "perf(template): add sync.Map cache for parsed text/template instances"
```

---

### Task 2: Enable sonic internal buffer pooling

**Files:**

- Modify: `pkg/pipeline/template/engine.go:15-16` (add variable, replace call)
- Modify: `pkg/pipeline/engine.go:10` (add variable, replace calls)
- Modify: `pkg/workflow/workflow.go:10` (add variable, replace calls)

- [ ] **Step 1: Add pooledSonic to `pkg/pipeline/template/engine.go`**

After `var reStepLegacy` block (line 31), add:

```go
var pooledSonic = sonic.Config{
	PoolAlloc: true,
}.Froze()
```

Replace line 96: `sonic.Marshal(v)` → `pooledSonic.Marshal(v)`

```go
"json": func(v any) (string, error) {
	b, err := pooledSonic.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
},
```

---

- [ ] **Step 2: Add pooledSonic to `pkg/pipeline/engine.go`**

After the import block, before `CheckpointData`:

```go
var pooledSonic = sonic.Config{
	PoolAlloc: true,
}.Froze()
```

Replace line 330: `sonic.Marshal(res.Data)` → `pooledSonic.Marshal(res.Data)`
Replace line 335: `sonic.Unmarshal(dataJSON, &stepResult)` → `pooledSonic.Unmarshal(dataJSON, &stepResult)`

---

- [ ] **Step 3: Add pooledSonic to `pkg/workflow/workflow.go`**

After the import block, before `ActionInfo`:

```go
var pooledSonic = sonic.Config{
	PoolAlloc: true,
}.Froze()
```

Replace all bare `sonic.Marshal` calls with `pooledSonic.Marshal`:

- Line 81: `sonic.Marshal(params)` → `pooledSonic.Marshal(params)`
- Line 201: `sonic.Marshal(input)` → `pooledSonic.Marshal(input)`
- Line 261: `sonic.Marshal(map[string]any(params))` → `pooledSonic.Marshal(map[string]any(params))`
- Line 307: `sonic.Marshal(map[string]any{"result": task.Result})` → `pooledSonic.Marshal(map[string]any{"result": task.Result})`
- Line 407: `sonic.Marshal(map[string]any(params))` → `pooledSonic.Marshal(map[string]any(params))`
- Line 449: `sonic.Marshal(map[string]any{"result": task.Result})` → `pooledSonic.Marshal(map[string]any{"result": task.Result})`

---

- [ ] **Step 4: Run all affected package tests**

```bash
go test ./pkg/pipeline/... ./pkg/workflow/... -v -count=1
```

Expected: All tests pass.

---

- [ ] **Step 5: Commit**

```bash
git add pkg/pipeline/template/engine.go pkg/pipeline/engine.go pkg/workflow/workflow.go
git commit -m "perf: enable sonic internal buffer pooling across pipeline and workflow"
```

---

### Task 3: Add Runner.Close() and executor lifecycle cleanup

**Files:**

- Modify: `pkg/workflow/workflow.go:170-326` (add method, add defer calls)

- [ ] **Step 1: Add Close() method to Runner**

After the `NewRunnerWithStore` function (line 169), before the `Run` method:

```go
// Close releases all executor engine resources (Docker clients, SSH connections, capability runtimes).
func (r *Runner) Close() error {
	for _, eng := range r.engines {
		if cerr := eng.Close(); cerr != nil {
			flog.Error(fmt.Errorf("[workflow] close engine: %w", cerr))
		}
	}
	return nil
}
```

---

- [ ] **Step 2: Add defer r.Close() in Execute**

In `Execute`, after the `taskMap` variable declaration (line 181-184), add as the third line of the function body:

```go
func (r *Runner) Execute(ctx context.Context, wf types.WorkflowMetadata, input types.KV, file string) error {
	taskMap := make(map[string]types.WorkflowTask)
	for _, wt := range wf.Tasks {
		taskMap[wt.ID] = wt
	}

	defer r.Close()

	results := make(map[string]string)
```

---

- [ ] **Step 3: Add defer r.Close() in ResumeWorkflow**

In `ResumeWorkflow`, after the opening brace, as the first statement:

```go
func (r *Runner) ResumeWorkflow(runID int64) error {
	defer r.Close()

	if r.store == nil {
```

---

- [ ] **Step 4: Run workflow tests**

```bash
go test ./pkg/workflow/... -v -count=1
```

Expected: All tests pass. (Test `TestNewRunner_CloseIsIdempotent` can be added in a follow-up; `executor.Engine.Close()` returns nil when runtime is nil.)

---

- [ ] **Step 5: Run lint**

```bash
go tool task lint
```

Expected: No new lint errors.

---

- [ ] **Step 6: Commit**

```bash
git add pkg/workflow/workflow.go
git commit -m "fix(workflow): add Runner.Close() to release executor engine resources"
```

---

### Task 4: Final verification

- [ ] **Step 1: Run all unit tests**

```bash
go test ./pkg/pipeline/... ./pkg/workflow/... -v -count=1
```

Expected: All tests pass.

---

- [ ] **Step 2: Run build**

```bash
go tool task build
```

Expected: Successful build.

---

- [ ] **Step 3: Run lint**

```bash
go tool task lint
```

Expected: No lint errors.

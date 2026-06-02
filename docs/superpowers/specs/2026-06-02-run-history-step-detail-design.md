# Run History Step Detail Enhancement

**Date:** 2026-06-02  
**Status:** Approved  
**Scope:** Frontend-only — `pkg/views/partials/pipeline_runs.templ`

## Problem

The Run History page (`/pipelines/:name/runs`) currently shows step-level input/output data only as hover tooltips on tiny P/R badges. Users cannot easily read or inspect the data flowing through pipeline steps.

## Design

### Template: `PipelineStepRunsDetail`

Rewrite the step runs detail table to show inline expandable Input/Output sections.

**Before:** A flat 8-column table with "Params" and "Result" columns containing P/R badges with hover-triggered JSON dropdowns.

**After:** A 6-column table (Step, Capability, Operation, Status, Attempt, Duration). Each step row is clickable and expands a detail row below containing two `<details>` blocks for Input and Output.

### HTML Structure (per step row)

```
<tr class="cursor-pointer hover" onclick="toggle detail row + rotate chevron">
  <td>▶</td>
  <td>step name</td>
  <td>capability</td>
  <td>operation</td>
  <td>status badge</td>
  <td>attempt</td>
  <td>duration</td>
</tr>
<tr class="step-detail-row hidden">
  <td colspan="7">
    <details open>
      <summary>Input</summary>
      <pre class="bg-base-200 rounded p-2 text-xs font-mono
                  overflow-x-auto max-h-60">
        { sprintJSON(s.Params) }
      </pre>
    </details>
    <details open>
      <summary>Output</summary>
      <pre class="bg-base-200 rounded p-2 text-xs font-mono
                  overflow-x-auto max-h-60">
        { sprintJSON(s.Result) }
      </pre>
    </details>
  </td>
</tr>
```

### Expand/Collapse Behavior

- Pure inline `onclick` on the step summary `<tr>` — toggles `hidden` on the next `<tr>` and rotates the chevron
- No HTMX requests (all data embedded at render time)
- The parent-run expand/collapse (HTMX-based) is unchanged and independent
- `<details open>` defaults Input/Output sections to expanded

### Empty State

- When `Params` is empty: Input `<details>` shows `(empty)` text, no `<pre>` block
- When `Result` is empty: Output `<details>` shows `(empty)` text, no `<pre>` block
- When both are empty: step row is not clickable (no chevron, no `cursor-pointer`)

### Shared Partial

`PipelineStepRunsDetail` is also used by the shareable view page (`view_pipeline_run.templ` → `ViewPipelineRunContent`). The same inline behavior applies there — no separate rendering needed.

## Data Source

No backend changes. All data already exists:

- `PipelineStepRun.Params` (map[string]any, JSON) — rendered input parameters
- `PipelineStepRun.Result` (map[string]any, JSON) — step output data
- `sprintJSON()` helper already exists in `pkg/views/partials/pipeline_runs.templ`

## Files Changed

| File | Change |
|------|--------|
| `pkg/views/partials/pipeline_runs.templ` | Rewrite `PipelineStepRunsDetail` template |
| `pkg/views/partials/pipeline_runs_templ.go` | Regenerated from `.templ` (no manual edit) |

## Out of Scope

- Variable tracing (showing template expression sources like `{{steps.foo.bar}}`). Deferred to a future iteration.
- Changes to the parent-run expand/collapse behavior.

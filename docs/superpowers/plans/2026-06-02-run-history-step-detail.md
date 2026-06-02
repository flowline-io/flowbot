# Run History Step Detail Enhancement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the P/R badge hover-tooltip pattern in `PipelineStepRunsDetail` with inline expandable Input/Output JSON sections.

**Architecture:** Single-file frontend change. Each step row gets an `onclick` handler that toggles a hidden detail `<tr>` below it containing two `<details>` blocks (Input, Output). No backend changes, no new DB columns, no HTMX requests — all data is already embedded in the template at render time.

**Tech Stack:** templ v0.3, Tailwind/DaisyUI utility classes

---

### Task 1: Rewrite PipelineStepRunsDetail template

**Files:**
- Modify: `pkg/views/partials/pipeline_runs.templ:64-128`
- (Regen creates): `pkg/views/partials/pipeline_runs_templ.go`

- [ ] **Step 1: Replace the `PipelineStepRunsDetail` template with inline expandable rows**

In `pkg/views/partials/pipeline_runs.templ`, replace the existing `PipelineStepRunsDetail` function (lines 64-128) with:

```templ
templ PipelineStepRunsDetail(steps []*gen.PipelineStepRun) {
	if len(steps) == 0 {
		<div class="text-sm text-base-content/30 py-3 text-center">No step runs recorded for this run.</div>
	} else {
		<div class="px-4 py-3" data-testid="step-runs-detail">
			<div class="text-xs font-medium text-base-content/30 uppercase tracking-wider mb-2">Step Runs</div>
			<table class="table table-xs">
				<thead>
					<tr>
						<th class="w-8"></th>
						<th>Step</th>
						<th>Capability</th>
						<th>Operation</th>
						<th>Status</th>
						<th>Attempt</th>
						<th>Duration</th>
					</tr>
				</thead>
				<tbody>
					for _, s := range steps {
						if len(s.Params) == 0 && len(s.Result) == 0 {
							<tr data-testid={ "step-row-" + s.StepName }>
								<td></td>
								<td class="font-medium text-base-content">{ s.StepName }</td>
								<td class="text-base-content/50">{ s.Capability }</td>
								<td class="text-base-content/50">{ s.Operation }</td>
								<td>
									<span class={ runsStatusClass(int(s.Status)) }>{ runsStatusText(int(s.Status)) }</span>
								</td>
								<td class="text-base-content/50">{ fmt.Sprint(s.Attempt) }</td>
								<td class="text-base-content/50">{ stepRunsDuration(s) }</td>
							</tr>
						} else {
							<tr class="cursor-pointer hover"
								data-testid={ "step-row-" + s.StepName }
								onclick="var c=this.querySelector('.step-chevron');c.classList.toggle('rotate-90');this.nextElementSibling.classList.toggle('hidden')">
								<td class="text-base-content/30">
									<span class="step-chevron inline-block transition-transform duration-200">&#9654;</span>
								</td>
								<td class="font-medium text-base-content">{ s.StepName }</td>
								<td class="text-base-content/50">{ s.Capability }</td>
								<td class="text-base-content/50">{ s.Operation }</td>
								<td>
									<span class={ runsStatusClass(int(s.Status)) }>{ runsStatusText(int(s.Status)) }</span>
								</td>
								<td class="text-base-content/50">{ fmt.Sprint(s.Attempt) }</td>
								<td class="text-base-content/50">{ stepRunsDuration(s) }</td>
							</tr>
							<tr class="step-detail-row hidden">
								<td colspan="7">
									<div class="space-y-1 py-1">
										if len(s.Params) > 0 {
											<details class="mb-1" open>
												<summary class="cursor-pointer text-xs font-mono text-base-content/60 hover:text-base-content select-none mb-1">Input</summary>
												<pre class="bg-base-200 rounded-box p-2 text-xs font-mono overflow-x-auto max-h-60 whitespace-pre">{ sprintJSON(s.Params) }</pre>
											</details>
										} else {
											<div class="text-xs text-base-content/20 italic py-1">Input: (empty)</div>
										}
										if len(s.Result) > 0 {
											<details open>
												<summary class="cursor-pointer text-xs font-mono text-base-content/60 hover:text-base-content select-none mb-1">Output</summary>
												<pre class="bg-base-200 rounded-box p-2 text-xs font-mono overflow-x-auto max-h-60 whitespace-pre">{ sprintJSON(s.Result) }</pre>
											</details>
										} else {
											<div class="text-xs text-base-content/20 italic py-1">Output: (empty)</div>
										}
									</div>
								</td>
							</tr>
						}
					}
				</tbody>
			</table>
		</div>
	}
}
```

- [ ] **Step 2: Regenerate the templ Go code**

```bash
go tool templ generate pkg/views/partials/pipeline_runs.templ
```

- [ ] **Step 3: Run format and lint**

```bash
go tool task format
go tool task lint
```

- [ ] **Step 4: Build to verify compilation**

```bash
go tool task build
```

- [ ] **Step 5: Commit**

```bash
git add pkg/views/partials/pipeline_runs.templ pkg/views/partials/pipeline_runs_templ.go docs/superpowers/specs/2026-06-02-run-history-step-detail-design.md docs/superpowers/plans/2026-06-02-run-history-step-detail.md
git commit -m "feat: replace step run P/R hover tooltips with inline expandable Input/Output sections"
```

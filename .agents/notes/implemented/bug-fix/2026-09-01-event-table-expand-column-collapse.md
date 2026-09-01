# Agent Note: Event table expand keeps other columns visible

Status: implemented

## Problem

Clicking an expandable table row loads JSON or prompt text into a `colspan` cell. Long unwrapped `pre` lines set the table's min-content width. Combined with `overflow-x-auto` and sticky first-column pinning — which also matched the expanded cell because it is the row's only `td` — other columns scrolled out of view or were covered.

## Decision

Constrain every expanded cell so payload HTML does not participate in column sizing: class `flowbot-table-expand-cell` (`max-width: 0` on the `td`, `width: 0; min-width: 100%` on the child). Wrap `.flowbot-json` with `pre-wrap` / `overflow-wrap: anywhere`. Pin only non-`colspan` first cells.

Surfaces: data events, webhook logs, pipeline runs (outer row and step JSON), workflow runs (outer row and step JSON), subagent task detail.

## Alternatives considered

- **`table-layout: fixed` on all `.flowbot-table-pin` tables.** Rejected: equal-width columns truncate entity IDs and source chips on tables that do not expand.
- **Move payload out of the table (drawer / modal).** Rejected: operators expect inline expand; pipeline run steps already keep JSON in the row.

## Consequences

- Event, webhook, pipeline, and workflow expand fragments stay a single root element so `td > *` can constrain them.
- Pipeline / workflow step JSON still uses `.run-json-preview`; expand cells also use `flowbot-table-expand-cell`.
- Notify-rule and session-entry list expands stay inside a named cell and already wrap; they do not hide sibling columns.
- The unused session-entry payload HTMX fragment is not an expand-row surface.
- Empty-state `colspan` cells have no wide `pre` and stay unconstrained.

## Verification

- `TestDataEventsTable_expandTarget`, `TestWebhookLogsTable_expandConstrainsCellWithoutScroll`, `TestPipelineRunsTable_expandConstrainsCellWithoutScroll`, `TestWorkflowRunsTable_expandConstrainsCellWithoutScroll`, and `TestAgentSubagentTaskDetail_constrainsWidePrompt` assert `flowbot-table-expand-cell`.
- `TestCommittedCSSIncludesNavbarIconUtilities` includes the expand-cell and non-colspan pin selectors in `custom.css`.
- `go test ./pkg/views/partials ./ -run 'Test(DataEventsTable_expandTarget|WebhookLogsTable_expandConstrainsCellWithoutScroll|PipelineRunsTable_expandConstrainsCellWithoutScroll|WorkflowRunsTable_expandConstrainsCellWithoutScroll|AgentSubagentTaskDetail_constrainsWidePrompt|EventPayloadDetail|CommittedCSSIncludesNavbarIconUtilities)'`.

# Agent Note: Table row expand does not scroll the viewport

Status: implemented

## Problem

Clicking a table row to load inline details used `hx-swap="innerHTML show:top"`. HTMX then scrolled the swapped cell to the top of the viewport, which felt like an anchor jump and moved the operator away from the row they clicked.

## Decision

Expand swaps use `innerHTML show:none`. The row stays where it is; details open underneath. Surfaces: data events, webhook logs, pipeline runs, workflow runs, subagent task details.

## Alternatives considered

- **`show:nearest`.** Rejected: still moves the page when the expanded block is taller than the remaining viewport.
- **`data-preserve-scroll` on the table wrapper.** Rejected: that marker is for polling replacements of a whole panel, not a one-shot expand.

## Consequences

- Polling panels keep `show:none` plus `data-preserve-scroll`; expand rows do not need the marker.
- Operators who want the expanded JSON in view scroll manually.

## Verification

- Expand tests assert `hx-swap="innerHTML show:none"` and reject `show:top`.
- `go test ./pkg/views/partials -run 'Test(DataEventsTable_expandTarget|WebhookLogsTable_expandConstrainsCellWithoutScroll|PipelineRunsTable_expandConstrainsCellWithoutScroll|WorkflowRunsTable_expandConstrainsCellWithoutScroll|AgentSubagentTaskRow_expandDoesNotScroll)'`.

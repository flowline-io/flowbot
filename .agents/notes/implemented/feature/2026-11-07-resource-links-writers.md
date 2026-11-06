# Agent Note: resource_links writers (workflow, MCP, failure)

Status: implemented

## Problem

After [resource_links generalization](2026-11-07-resource-links-generalization.md), only pipeline success edges wrote rows. Workflow runs, MCP mutation external-input edges, and failed steps left no `resource_links` rows, so failure outcomes and MCP writes could not appear in live `flow.graph` aggregation ([lineage-flows](../architecture/2026-10-30-lineage-flows-in-analytics.md)).

## Decision

Writers cover four surfaces under the M4.5 grilling lock:

- Pipeline and workflow capability steps write `status=ok` edges when `ResourceMeta` has both `EventID` and `EntityID`. Workflow edges use `run_kind=workflow` and synthetic `source_event_id` `wf:{run_id}`.
- Failed capability steps write `status=failed` edges with synthetic `target_event_id` `fail:{run_id}:{step}` (no `data_events` row; empty `target_entity_id`). Same step failures UPSERT to one row; a later success adds a separate ok edge.
- MCP capability mutations write `run_kind=mcp` external-input edges after a successful invoke. Correlation is minted before invoke (existing mutation path). `source.app` is `agent:{tokenID}` (D11; tokens have no display name). `source.entity_id` is the request id; `target_capability` is the tool name.
- Create-class `ResourceMeta` includes `EventID` for addressable creates that still ship (including `drone.deploy`, `karakeep.create`, `miniflux.create_feed`). The five removed uncalled providers ([note](../simplification/2026-10-21-remove-uncalled-providers.md)) are out of scope. GitHub/Gitea have no create mutations.

`duration_ms`, `uid`, and `flow_edge_daily` remain deferred with the lineage-flows note.

## Alternatives considered

- `source.app = agent:<token_name>` from the original plan — rejected in grilling: access-token params have no display name; row id is the stable node key.
- Skip failure edges until `lineage.trace` lands — rejected. Acceptance requires workflow failures to be visible as failed edges once writers exist.
- Emit MCP edges for every mutation without `ResourceMeta` — rejected. Edges need a target entity; reads and non-creating writes stay silent.

## Consequences

- Pipeline/workflow/MCP invoke paths share `RecordResourceLink` (workflow adapter delegates to `PipelineStore`).
- Operators can correlate MCP writes with audit rows via `correlation_id` on the edge.
- Create ops that still omit `EventID` on `ResourceMeta` still do not produce edges.

## Verification

- `go test` covers pipeline failed-edge write, workflow failed-edge write, and `TestRecordMCPExternalInputLink_Queryable` (MCP write → edge with `run_kind=mcp` and correlation).
- Create handlers for `drone.deploy`, `karakeep.create`, and `miniflux.create_feed` return `ResourceMeta.EventID`.

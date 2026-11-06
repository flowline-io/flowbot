# Agent Note: resource_links generalization

Status: implemented

## Problem

`resource_links` only stored source/target event/app/capability/entity plus pipeline fields. `flow.graph` live aggregation (locked in [2026-10-30-lineage-flows-in-analytics.md](../architecture/2026-10-30-lineage-flows-in-analytics.md)) needs a stable run surface (`run_kind`), outcome (`status`), and `correlation_id`, plus an index on `(source_app, target_app, created_at)`. Pre-generalization rows would otherwise lack usable defaults after an additive Ent migrate.

## Decision

`resource_links` gains `run_kind` (`pipeline|workflow|function|mcp|manual`, default `pipeline`), `status` (`ok|failed`, default `ok`), and `correlation_id` (default empty). Indexes include `(source_app, target_app, created_at)` for flow aggregation and `correlation_id` for trace grouping. `pipeline_run_id` / `pipeline_name` stay as compatibility columns.

Ent `Schema.Create` applies column defaults on upgrade. `store.Migrate` then runs `ResourceChainStore.BackfillResourceLinkDefaults`, which copies a non-empty source `data_events.correlation_id` onto links that still have an empty `correlation_id`. Pipeline writes set `run_kind=pipeline`, `status=ok`, and the trigger event's correlation id through `RecordResourceLink`.

This change does not add `flow_edge_daily`, `duration_ms`, `uid`, `run_id`, or `step_name`; later T45 tasks add writers and queries under the lineage-flows note.

## Alternatives considered

- Full original T45.2 field set (`run_id`, `step_name`, `duration_ms`, `uid`) in the same migrate — deferred. Schema growth without writers or readers would widen blast radius before workflow/MCP edge coverage lands.
- Manual SQL migration files — rejected. Store migrations stay Ent `Schema.Create` on startup ([store AGENTS](../../../internal/store/AGENTS.md)).
- Leave old rows without correlation backfill — rejected. Source events already carry `correlation_id` after T45.1; copying it keeps live aggregation and future trace useful for historical pipeline edges.

## Consequences

- New and upgraded databases expose the three columns and both indexes.
- Empty `run_kind` / `status` on insert normalize to pipeline / ok in `RecordResourceLink`.
- Workflow, MCP, function, manual, and failed-edge writers remain follow-on work.

## Verification

- `go test` covers `TestPipelineStore_RecordResourceLink` (defaults + correlation write/read) and `TestResourceChainStore_BackfillResourceLinkDefaults` (correlation copy from source event).
- Generated migrate schema lists `run_kind`, `status`, `correlation_id`, `resourcelink_source_app_target_app_created_at`, and `resourcelink_correlation_id`.

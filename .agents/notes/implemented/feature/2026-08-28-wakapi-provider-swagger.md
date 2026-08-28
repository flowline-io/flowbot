# Agent Note: Wakapi provider aligned with OpenAPI

Status: implemented

## Problem

The initial Wakapi provider only exposed summary and project list. Summary used native `/api/summary`, which does not return `total_seconds`, so devops `wakapi_summary` always reported zero. The Swagger spec documents richer WakaTime-compatible read endpoints that homelab operators expect for coding-stats dashboards.

## Decision

Expand `pkg/providers/wakapi` against the Wakapi OpenAPI surface:

- Native summary with filter params (`GetSummary`)
- WakaTime-compatible stats with `total_seconds` (`GetStats`)
- All-time totals, heartbeats, health, and project search
- Basic auth: `Authorization: Basic base64(api_key)` per Wakapi docs

Wire read paths through the existing `devops` aggregator:

- `wakapi_summary` now calls `GetStats` and returns human-readable totals
- New ops: `wakapi_health`, `wakapi_all_time`
- Wakapi included in aggregate `HealthCheck` when configured

## Alternatives considered

- Standalone `wakapi` CapType — rejected; Wakapi fits the devops aggregator alongside Beszel/Grafana.
- Keep native `/api/summary` for `wakapi_summary` — rejected; missing `total_seconds` breaks the primary use case.

## Consequences

- Provider client signatures changed (`ListProjects` accepts optional query; summary uses `SummaryParams` / `StatsParams`).
- CLI adds `flowbot devops wakapi health` and `all-time`; summary supports `--project`.

## Verification

```bash
go test ./pkg/providers/wakapi/... ./pkg/capability/devops/... ./cmd/cli/command/... ./internal/modules/hub/...
go tool task lint
```

Configure `providers.wakapi.endpoint` and `providers.wakapi.api_key`, then:

```bash
flowbot devops wakapi health
flowbot devops wakapi summary --interval week
flowbot devops wakapi all-time
flowbot devops wakapi projects --query flowbot
```

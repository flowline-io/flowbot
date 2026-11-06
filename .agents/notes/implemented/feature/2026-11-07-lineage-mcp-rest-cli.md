# Agent Note: lineage / flow / events MCP + REST + CLI

Status: implemented

## Problem

`lineage.trace`, `lineage.search`, `flow.graph`, and `events.timeline` were already named in the MCP lineage rate-limit bucket and in scope routing (`lineage` / `flows` / `events` → `lineage:read`), but nothing advertised or mounted them. Analytics already owned live `FlowGraph` / `Trace` ([flow-graph live aggregation](2026-11-07-flow-graph-live-aggregation.md)); callers still had no MCP tools, REST, or CLI.

## Decision

The analytics module mounts `GET /service/lineage/trace`, `/service/lineage/search`, `/service/flows/graph`, and `/service/events/timeline`. MCP `extraTools` advertise the same four names; handlers call `analytics.ActiveService()` (not the store). CLI `flowbot lineage|flow|events` uses the JSON envelope and the Go client.

D10 for `events.timeline`: rows appear under `lineage:read` (non-admins filtered by uid). Payload `data` is omitted unless `include_payload=true` and the caller holds `service:{capability|app}:read` (or `admin:*`); otherwise the row stays with `payload_omitted: true`. A request that sets both `include_payload` and `app` without that app's read scope is `permission_denied` for the whole call.

API-token REST calls charge the shared MCP `all` + `lineage` Redis buckets via `analytics.SetLineageCharge` (wired from `internal/server/mcp`). `kind=full` web sessions skip those buckets.

## Alternatives considered

- Revive `internal/modules/lineage` for REST — rejected; [lineage-flows-in-analytics](../architecture/2026-10-30-lineage-flows-in-analytics.md) locks analytics ownership.
- MCP handlers calling `ResourceChainStore` directly — rejected; same note requires MCP → analytics service.
- Always return event payloads under `lineage:read` — rejected; D10 keeps payload behind service read scopes.

## Consequences

- Snapshot `internal/server/mcp/testdata/tools.snapshot.json` lists the four tools.
- [2026-10-17-mcp-rate-limit-and-audit-list.md](2026-10-17-mcp-rate-limit-and-audit-list.md) Consequences no longer say lineage REST is unmounted.
- Static edges remain accepted-and-ignored on `flow.graph` until a later slice.

## Verification

- `go test ./internal/server/mcp/ -run 'TestToolSnapshot|TestContract|TestAdvertised'`
- `go test ./internal/modules/analytics/ ./pkg/validate/ ./cmd/cli/command/ -count=1`
- `flowbot lineage --help` / `flowbot flow graph --help` / `flowbot events timeline --help`

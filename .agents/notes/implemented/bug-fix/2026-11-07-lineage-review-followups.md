# Agent Note: Lineage review follow-ups

Status: implemented

## Problem

A two-axis review of the M4.5 lineage/flows slice found hard standards gaps and incorrect Relations Graph / D10 timeline behavior: `internal/modules/AGENTS.md` still denied analytics HTTP; Flows tooltips hard-coded English; Relations Graph forced `via.kind=pipeline` and `count_ok=1`; `events.timeline` omitted `payload_omitted` when `include_payload` was false; MCP writers built `*gen.ResourceLink` directly; and lineage REST had no owning BDD.

## Decision

- Document analytics-mounted `/service/lineage|flows|events` in `internal/modules/AGENTS.md`.
- Set `payload_omitted: true` whenever payload bodies are not attached (default and insufficient-scope paths).
- Carry `run_kind` / `status` on `ResourceEdge` from `FindNodeRelations`, and aggregate Relations Graph edges with correct `via.kind` and ok/failed counts.
- Route MCP resource-link writes through `model.ResourceLink` + `PipelineRunStoreAdapter`.
- Return a typed `types.Error` (invalid argument / field `seed`) from Trace validation instead of string matching.
- Localize flow-graph tooltip labels via `client.flow.*` + `flowbotI18n`; drop the unused JS tree render path.
- Add analytics `InitForE2E` / `MountForE2E` and owning BDD `tests/specs/lineage_api_spec_test.go`.

## Alternatives considered

- Leave Relations Graph as a simplified app-only sketch until a dedicated neighborhood aggregator — rejected; the Flows chart already encodes failed/via semantics, so a neighborhood `FlowGraph` must match.
- Keep MCP on `*gen.ResourceLink` because `PipelineStore.RecordResourceLink` still takes gen — rejected; adapters already own the model→gen map and keep server code off generated types.

## Consequences

- Relations Graph edge clicks navigate by real `run_kind`; failed neighborhood edges render red.
- Operators and MCP clients always see `payload_omitted` when `data` is absent on timeline rows.
- BDD (Docker) owns the lineage/flows/events HTTP surface; unit tests cover timeline defaults and Relations aggregation.

## Verification

- `go test ./internal/modules/analytics/ ./internal/modules/web/ ./internal/server/mcp/ ./internal/store/ -count=1` (package-scoped)
- `go tool task lint`
- BDD: `tests/specs/lineage_api_spec_test.go` (requires Docker; skip without it)

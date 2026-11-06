# Agent Note: Analytics Flows partition and Relations graph

Status: implemented

## Problem

Live `flow.graph` and lineage Trace already ship on the analytics domain service, MCP, REST, and CLI, but operators still had no Web topology. The earlier M4.5 plan named a standalone `/service/web/flows` page; [lineage-flows-in-analytics](../architecture/2026-10-30-lineage-flows-in-analytics.md) locked Flows as the third Analytics partition beside Personal and Ops. Relations stayed HTMX-tree-only with no graph view.

## Decision

Analytics gains a Flows nav section and `/service/web/analytics/flows` (panel + JSON). The page loads a vendored ECharts custom build (`public/vendor/echarts.custom.min.js`: graph, sankey, tree, tooltip, legend, dataZoom) and `public/js/flow-graph.js`. Topology is the default; Sankey is a toggle. Failed edges (`count_failed > 0`) render red; static edges would be dashed when present. Edge clicks navigate to pipeline/workflow/function run lists when `via` resolves. `analytics.Service.FlowGraphUI` caps the chart at 20 nodes / 50 edges (failed and high-traffic first) so the hand-check size stays interactive within about 1s.

`/service/web/relations` adds a Graph tab beside the existing HTMX tree (no-JS fallback). `GET /relations/graph` returns a neighborhood `FlowGraph` for the selected node; `relations-view.js` switches panes and fetches JSON.

Sankey ships in this change. Cuttable A″ (drop Sankey, keep topology only) remains available if a later checkpoint forces a cut; T2.6 client signing was still unsigned at implement time and did not trigger that cut.

## Alternatives considered

- Standalone `/service/web/flows` outside Analytics — rejected; locked by the lineage-flows architecture note.
- Chart.js only — rejected; Chart.js has no graph/Sankey; grilling chose a custom ECharts build.
- Omit Sankey now — rejected while the default task text is “do Sankey”; cut only under the A″ checkpoint rule.

## Consequences

- Flows reuses Analytics range presets (24h / 7d / 30d).
- Product docs at [docs/analytics/README.md](../../../docs/analytics/README.md) describe the Flows partition.
- MCP/REST `flow.graph` stay uncapped; only the Web UI path uses `FlowGraphUI`.

## Verification

- `go test ./pkg/types/ -run CapFlowGraph`, `go test ./internal/modules/web/ -run 'AnalyticsFlows|RelationsPage|RelationsGraph'`, and `go tool task lint` pass.
- `public/vendor/README.md` records the ECharts custom build version and contents.
- Analytics shell HTML includes `data-testid="analytics-nav-flows"`; Relations HTML includes `data-testid="relations-tab-graph"`.

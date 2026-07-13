# Upgrade: Capability ↔ Provider 1:1

## What changed

- Capability type names now equal provider IDs (`karakeep`, `miniflux`, `kanboard`, …).
- Package path: `pkg/ability` → `pkg/capability` with flat provider packages.
- `Descriptor.Backend` / `DataEvent.Backend` removed from the Go API.
- REST routes: `/service/bookmark` → `/service/karakeep` (and similarly for others).
- Auth scopes: `service:bookmark:read` → `service:karakeep:read`.
- Prometheus metrics renamed `ability_*` → `capability_*`; label values use provider IDs.
- Apply SQL drops `backend` columns on `capability_bindings` and `data_events`.

## Mapping

| Old CapType | New CapType | REST prefix |
|------------|-------------|-------------|
| bookmark | karakeep | `/service/karakeep` |
| reader | miniflux | `/service/miniflux` |
| kanban | kanboard | `/service/kanboard` |
| note | trilium | `/service/trilium` |
| memo | memos | `/service/memos` |
| forge | gitea | `/service/gitea` |
| github | github | `/service/github` |

## Steps

1. Stop Flowbot.
2. Backup PostgreSQL.
3. Apply [`2026-07-capability-provider-1to1.sql`](2026-07-capability-provider-1to1.sql).
4. Update pipeline/workflow YAML: `capability:` fields to provider IDs. Leave `event:` as domain names.
5. Update Homelab compose labels: `flowbot.capability=<providerID>`. Remove `flowbot.backend` (ignored with a warning this release).
6. Re-issue API tokens if they used legacy `service:bookmark:*` scopes when convenient; runtime aliases still map those strings to provider scopes this release.
7. Start Flowbot and verify `/hub/capabilities` and health checks.

## Invoke example

```go
capability.Invoke(ctx, hub.CapKarakeep, karakeep.OpList, map[string]any{"limit": 20})
// Pipeline task: capability:karakeep.list
// Event match: bookmark.created
```

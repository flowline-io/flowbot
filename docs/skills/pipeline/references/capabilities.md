# Pipeline capability operations reference

Load a capability file below when authoring pipeline steps with `capability` + `operation`.
Param tables are generated from each capability's `OpDef.Input` (same source as hub registration).

**Rules for agents:**

1. Only use capability types listed in the table below.
2. Before emitting any operation, load `capabilities/<type>.md` and copy required params from that file.
3. Do not invent ops, param keys, or types.
4. Pipeline YAML uses fields `capability` and `operation` (not workflow `action: capability:<type>.<op>`).

## Result envelope

Capability steps expose an `InvokeResult`-shaped payload readable via `{{step "name" "…"}}` / `jsonpath`.

## Capabilities

| Capability | Reference |
|------------|-----------|
| `core` | [capabilities/core.md](capabilities/core.md) — Core runtime primitives: notify, clip, agent, HTTP, sandboxed exec, and KV |
| `devops` | [capabilities/devops.md](capabilities/devops.md) — DevOps aggregator for beszel, uptimekuma, traefik, grafana, wakapi, and dozzle |
| `fireflyiii` | [capabilities/fireflyiii.md](capabilities/fireflyiii.md) — Finance capability for Firefly III |
| `gateway` | [capabilities/gateway.md](capabilities/gateway.md) — Local CLI gateway: delegate coarse run_cursor jobs to cmd/gateway workers |
| `gitea` | [capabilities/gitea.md](capabilities/gitea.md) — Forge capability |
| `github` | [capabilities/github.md](capabilities/github.md) — GitHub capability |
| `kanboard` | [capabilities/kanboard.md](capabilities/kanboard.md) — Kanban capability |
| `karakeep` | [capabilities/karakeep.md](capabilities/karakeep.md) — Bookmark capability |
| `memos` | [capabilities/memos.md](capabilities/memos.md) — Memo capability for short-form note-taking |
| `miniflux` | [capabilities/miniflux.md](capabilities/miniflux.md) — Reader capability |
| `nocodb` | [capabilities/nocodb.md](capabilities/nocodb.md) — NocoDB bases, tables, and records |
| `transmission` | [capabilities/transmission.md](capabilities/transmission.md) — Download capability for Transmission |
| `trilium` | [capabilities/trilium.md](capabilities/trilium.md) — Note capability for note-taking systems |

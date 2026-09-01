# Workflow capability actions reference

Load a capability file below when authoring `capability:<type>.<operation>` tasks.
Param tables are generated from each capability's `OpDef.Input` (same source as hub registration).

**Rules for agents:**

1. Only use capability types listed in the table below.
2. Before emitting any `capability:<type>.*` action, load `capabilities/<type>.md` and copy required params from that file.
3. Do not invent ops, param keys, or types (for example there is no `capability:archive.*`).

## Result envelope

Every capability step stores a JSON `InvokeResult` string as `{{step "id" "result"}}`:

| Field | Meaning |
|-------|---------|
| `capability` | Capability type (e.g. `karakeep`, `core`) |
| `operation` | Operation name |
| `data` | Domain payload (object, array, or scalar); omit when empty |
| `text` | Optional short human summary |
| `page` | Optional pagination for list ops |

**Usage patterns:**

- Field: `{{jsonpath (step "save" "result") "data.url"}}`
- Exists: `{{if jsonpathExists (step "save" "result") "data.id"}}...{{end}}`
- Whole JSON (rare): `{{step "save" "result"}}`

Mutation ops change remote/system state; prefer read ops when exploring.

## Common `data` paths

| Op pattern | Typical `jsonpath` |
|------------|----------------------|
| `karakeep.create` / `get` | `data.id`, `data.url`, `data.title` |
| `karakeep.check_url` | `data.exists`, `data.id` |
| `karakeep.list` / `search` | `data` is an array; `page` for cursor |
| `kanboard.create_task` / `get_task` | `data.id`, `data.title`, `data.project_id` |
| `core.notify_send` | `data.sent` |
| `core.agent_run` | `data.reply` |
| `core.clip_create` | `data` includes public URL fields |
| `core.http_request` | `data.status`, `data.body` |
| `core.run_terminal` / `run_code` | `data.output`, `data.exit_code` |
| `core.kv_get` | `data` is the stored value |

## Capabilities

| Capability | Reference |
|------------|-----------|
| `confluence` | [capabilities/confluence.md](capabilities/confluence.md) — Confluence Cloud spaces and pages |
| `core` | [capabilities/core.md](capabilities/core.md) — Core runtime primitives: notify, clip, agent, HTTP, sandboxed exec, and KV |
| `devops` | [capabilities/devops.md](capabilities/devops.md) — DevOps aggregator for beszel, uptimekuma, traefik, grafana, wakapi, dozzle, netalertx, and scanopy |
| `email` | [capabilities/email.md](capabilities/email.md) — Email capability for SMTP send and IMAP read |
| `fireflyiii` | [capabilities/fireflyiii.md](capabilities/fireflyiii.md) — Finance capability for Firefly III |
| `functions` | [capabilities/functions.md](capabilities/functions.md) — Named functions (FaaS): pure transform invoke of published function versions. HTTP token/hmac on POST /service/functions/call only; Pipeline and capability.Invoke do not validate function HTTP secrets. |
| `gateway` | [capabilities/gateway.md](capabilities/gateway.md) — Local CLI gateway: delegate coarse run_cursor jobs to cmd/gateway workers |
| `gitea` | [capabilities/gitea.md](capabilities/gitea.md) — Forge capability |
| `github` | [capabilities/github.md](capabilities/github.md) — GitHub capability |
| `kanboard` | [capabilities/kanboard.md](capabilities/kanboard.md) — Kanban capability |
| `karakeep` | [capabilities/karakeep.md](capabilities/karakeep.md) — Bookmark capability |
| `memos` | [capabilities/memos.md](capabilities/memos.md) — Memo capability for short-form note-taking |
| `miniflux` | [capabilities/miniflux.md](capabilities/miniflux.md) — Reader capability |
| `nocodb` | [capabilities/nocodb.md](capabilities/nocodb.md) — NocoDB bases, tables, and records |
| `transmission` | [capabilities/transmission.md](capabilities/transmission.md) — Download capability for Transmission |
| `trello` | [capabilities/trello.md](capabilities/trello.md) — Trello cloud boards, lists, and cards |
| `trilium` | [capabilities/trilium.md](capabilities/trilium.md) — Note capability for note-taking systems |

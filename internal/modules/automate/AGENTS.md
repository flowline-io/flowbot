# Automate Module

JSON APIs for named functions, pipelines, and workflows. Web UI for the same domains lives under [`internal/modules/web`](../web/) (`/service/web/...`). Repo-wide standing orders: root [AGENTS.md](../../../AGENTS.md). Parent: [../AGENTS.md](../AGENTS.md). Merge: [note](../../../.agents/notes/implemented/simplification/2026-09-05-merge-automate-modules.md).

## Entry points

- Module config key: `modules.automate` (`enabled` defaults true when omitted).
- Routes (mounted from one `Webservice()`):
  - `/service/automate/functions/*` — `function:read` on GET, `function:run` on POST/PUT/PATCH, `function:delete` on DELETE
  - `/service/automate/pipeline/*` — `pipeline:read` on GET, `pipeline:run` on POST/PUT/PATCH, `pipeline:delete` on DELETE
  - `/service/automate/workflow/*` — `workflow:read` on GET, `workflow:run` on POST/PUT/PATCH, `workflow:delete` on DELETE
- E2E: `InitForE2E` / `MountForE2E`.

## Boundaries

- Handlers call `pkg/functions`, `pkg/pipeline`, `pkg/workflow` active services only — no store SQL, no provider clients.
- Do not mount these JSON APIs from the web module.

## Testing

```bash
go test ./internal/modules/automate/...
```

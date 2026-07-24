# Chatagent (Product Orchestration)

Binds `pkg/agent` to REST (`/chatagent/*`), Web (`/service/web/agents/*`), platform sinks, store, and scheduled tasks. Must not become a second agent engine.

## Boundaries

- **Allowed**: `pkg/agent/*`, `internal/store/*`, views via handlers.
- **Forbidden**: `internal/store` inside `pkg/agent`; store types in engine APIs.

## Entry points

- REST: `internal/server/chatagent_http*.go` — primary `POST .../messages` (SSE); observer `GET .../events` (subset via `IsObserverStreamEvent` in `protocol.go`: confirm, confirm_resolved, canceled, mode_change, run_complete)
- Web SSE: `internal/modules/web/chatagent_web_stream.go`
- Platform DM: `chatagent_handler.go`
- Shared `*Service`: `server.ChatAgentService()` (`chatagent_bootstrap.go`) → `BindSharedService` (`service_state.go`) + `web.SetChatAgentService`
- Scripts: `pkg/views/partials/chatagent_scripts.templ` (`FlowbotChatAgent` only)

Hot-path files: `service.go` (Run), `service_state.go`, `run_io.go` (`withRunIO`), `hooks.go`, `harness_pool.go`, `api_stream.go` / `api_run.go`, `protocol.go`, `confirm.go`, `session_events.go`. Non-interactive: `pipeline_run.go` (not Run phases), `ephemeral_run.go`, `scheduled_run.go`.

## Run pipeline

`StreamAPIRun` → `RunAPI` → `Service.Run`: prepare → lock → harness → hooks/permission/confirm (wired at harness build; ask via `withRunIO`) → stream → deliver → cleanup (abort path). Inject publisher/confirm via `withRunIO`; do not look up from session maps in hooks.

Protocol changes: update `docs/agent/chatagent-feature-checklist.md` + tests first. Keep HTTP handlers thin.

## Testing

```bash
go test ./internal/server/chatagent -count=1
```

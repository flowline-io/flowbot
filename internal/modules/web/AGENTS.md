# Web UI Module

Server-rendered HTML under `/service/web/*` (HTMX + Alpine). Templates live in `pkg/views/`; handlers in this package. UI visual / HTMX / Alpine details: [`.cursor/rules/web-ui.mdc`](../../../.cursor/rules/web-ui.mdc). Repo-wide standing orders: root [AGENTS.md](../../../AGENTS.md). Parent: [../AGENTS.md](../AGENTS.md).

## Boundaries

- **Allowed**: store call sites per [store AGENTS.md](../../store/AGENTS.md); `capability.Invoke`; shared `*chatagent.Service` via `SetChatAgentService` (`chatagent_service.go`); already-wired `pkg/agent` packages listed in [pkg/agent/AGENTS.md](../../../pkg/agent/AGENTS.md) (`permission`; tests: `model` / `msg` / `session`).
- **Forbidden**: provider clients; SQL or Ent queries in handlers; ent types in templates (map to `pkg/types/model` in this package first); view templates under `internal/modules/`.

## Entry points

- Routes: `*_webservice.go` rule slices → aggregated in `rules.go` (`allWebserviceRules`) → `module.go` registers each group.
- Auth: default route auth (cookie `accessToken` + scopes). Only login/setup/logout/`csrf-token` use `route.WithNotAuth()`. Handlers still call `authenticateWeb()` (KindFull + login redirect). CSRF is Fiber `csrf` middleware (cookie `csrf_` or `__Host-csrf_`, header `X-Csrf-Token` / form `csrf_token`); helpers in `public/js/app.js`.
- Chatagent SSE: `chatagent_web_stream.go`. Shared service: `chatagent_service.go` (installed by `server.ChatAgentService`). Write rules: [server AGENTS.md](../../server/AGENTS.md).
- Scripts order of truth: `pkg/views/partials/chatagent_scripts.templ`
  - Composer: `util → slash → chat`
  - Approval: `util → approval → chat`
  - Thread: `util → sse → markdown → codeblocks → context → approval → todos → trajectory → thread → slash → chat → clip-copy`
- Namespace: `window.FlowbotChatAgent` only — no monolithic chatagent JS.

## Non-obvious rules

- New routes: add `*_webservice.go` + append to `allWebserviceRules` in `rules.go`.
- Dangerous UI actions (delete, revoke, logout, stop, disable, …): confirmation modal required — [web-ui.mdc § Dangerous actions](../../../.cursor/rules/web-ui.mdc).
- Set `c.Type("html")` before HTML; HTMX endpoints must not return JSON by mistake.
- Complex JS in `public/js/`; vendored deps only (no CDN).
- Markdown → `utils.MarkdownToSafeHTML` before `templ.Raw`.
- E2E helpers: `InitForE2E()` / `MountForE2E()`; CSRF helpers `AttachCSRFForTest` / `addWebAuth`.

## Testing

Which layer: [docs/testing/README.md](../../../docs/testing/README.md). Owning BDD for `/service/web` pages: `tests/specs/*_page_spec_test.go` (`agents`, `agent_sessions`, `agent_scheduled_tasks`, `notifications`, `event`, `home_token_usage`) and `life_spec_test.go`.

```bash
go test ./internal/modules/web/...
```

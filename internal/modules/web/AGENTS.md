# Web UI Module

Server-rendered HTML under `/service/web/*` (HTMX + Alpine). Templates live in `pkg/views/`; handlers in this package. UI visual / HTMX / Alpine details: [`.cursor/rules/web-ui.mdc`](../../../.cursor/rules/web-ui.mdc).

## Boundaries

- **Allowed**: `store.Database`, `capability.Invoke`, shared `*chatagent.Service` via `SetChatAgentService` (`chatagent_service.go`).
- **Forbidden**: `pkg/providers/*`; DB queries in handlers; ent types in templates; view templates under `internal/modules/`.

## Entry points

- Routes: `*_webservice.go` rule slices → aggregated in `rules.go` (`allWebserviceRules`) → `module.go` registers each group.
- Auth: cookie `authenticateWeb()`; most routes use `route.WithNotAuth()` then authenticate in-handler. CSRF double-submit (`csrfToken` / `X-CSRF-Token`); helpers in `public/js/app.js`.
- Chatagent SSE: `chatagent_web_stream.go`. Shared service: `chatagent_service.go` (installed by `server.ChatAgentService`).
- Scripts order of truth: `pkg/views/partials/chatagent_scripts.templ`
  - Composer: `util → chat`
  - Approval: `util → approval → chat`
  - Thread: `util → sse → markdown → codeblocks → context → approval → todos → thread → chat → clip-copy`
- Namespace: `window.FlowbotChatAgent` only — no monolithic chatagent JS.

## Non-obvious rules

- New routes: add `*_webservice.go` + append to `allWebserviceRules` in `rules.go`.
- Set `c.Type("html")` before HTML; HTMX endpoints must not return JSON by mistake.
- Complex JS in `public/js/`; vendored deps only (no CDN).
- Markdown → `utils.MarkdownToSafeHTML` before `templ.Raw`.
- E2E: `InitForE2E()` / `MountForE2E()`; CSRF helpers `AttachCSRFForTest` / `addWebAuth`.

## Testing

```bash
go test ./internal/modules/web/...
```

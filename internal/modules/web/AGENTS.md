# Web UI Module

Server-rendered HTML under `/service/web/*` (HTMX + Alpine). Templates live in `pkg/views/`; handlers in this package. UI visual / HTMX / Alpine details: [`.cursor/rules/web-ui.mdc`](../../../.cursor/rules/web-ui.mdc). Repo-wide standing orders: root [AGENTS.md](../../../AGENTS.md). Parent: [../AGENTS.md](../AGENTS.md).

## Boundaries

- **Allowed**: store call sites per [store AGENTS.md](../../store/AGENTS.md); `capability.Invoke`; shared `*chatagent.Service` via `SetChatAgentService` (`chatagent_service.go`); already-wired `pkg/agent` packages listed in [pkg/agent/AGENTS.md](../../../pkg/agent/AGENTS.md) (`permission`; tests: `model` / `msg` / `session`).
- **Forbidden**: provider clients; SQL or Ent queries in handlers; ent types in templates (map to `pkg/types/model` in this package first); view templates under `internal/modules/`.

## Entry points

- Routes: `*_webservice.go` rule slices → aggregated in `rules.go` (`allWebserviceRules`) → `module.go` registers each group.
- Auth: default route auth (cookie `accessToken` + scopes). Only login/setup/logout/`csrf-token` use `route.WithNotAuth()`. Handlers still call `authenticateWeb()` (KindFull + login redirect). CSRF is Fiber `csrf` middleware (cookie `csrf_` or `__Host-csrf_`, header `X-Csrf-Token` / form `csrf_token`); helpers in `public/js/app.js`. [TLS proxy Origin](../../../.agents/notes/implemented/bug-fix/2026-08-23-csrf-login-403-tls-proxy.md).
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
- E2E helpers: `InitForE2E()` / `MountForE2E()`; CSRF helpers `AttachCSRFForTest` / `addWebAuth`; locale helper `AttachLocaleForTest` (defaults `flowbot-lang=en` via `addWebAuth`).

## i18n

- **Stack**: [`pkg/i18n`](../../../pkg/i18n/) (go-i18n/v2 + embed TOML). Templ/Go: `i18n.T(ctx, messageID)`; template data: `i18n.TData(ctx, id, map[string]any{...})`. Client: `#flowbot-i18n` JSON in layouts + `flowbotI18n()` in [`public/js/app.js`](../../../public/js/app.js).
- **Locale**: cookie `flowbot-lang` (`en` | `zh`) → localizer tags `en` / `zh-Hans`. **Do not** reuse `flowbot.language` (ChatAgent reply language only).
- **Default**: missing/invalid cookie → `en`.
- **Middleware order** (`module.go`): `localeMiddleware` → CSRF on `/service/web`; `localeMiddleware` on `/c` for public clip pages.
- **Switch**: `POST /service/web/locale` (unauthenticated OK); navbar + auth layouts use `partials.LangSwitcher` (HTMX + server-rendered active state; not Alpine).
- **Templ**: pass request `context.Context` into page/layout components that call `i18n.T`; `Render` must use `ctx.Context()` / `c.Context()` from Fiber — never `context.Background()` for HTML.
- **Page titles**: `partials.PageHeader(ctx, titleKey, subtitleKey)` for static keys; `PageHeaderLiteral(ctx, title, subtitle)` for dynamic entity names; mix at call site (`PageHeaderLiteral(ctx, name, i18n.T(ctx, "page.workflow.subtitle"))`).
- **Handler toasts**: `webMsg(c, id)`, `webMsgData`, `toastErrorKey`, `setShowToastKey`, `renderFormErrorKey`, `renderErrorKey` in `utils.go` — not raw English literals.
- **Document titles**: `pages.DocTitleFlowbot` / `DocTitlePage` / `DocTitleNamed` / `DocTitleLiteral` / `DocTitleLife` in `layout.Base(ctx, title)`.
- **Client IDs**: after adding `client.*` keys to TOML, run `python scripts/update_client_ids.py` (also runs after `go tool task templ`).
- **Catalogs**: `pkg/i18n/locales/{en,zh}.toml` (filename is the go-i18n language tag). A leaf ID that is also a prefix of another ID in the same file must use a quoted table (`["error.validation"]`).
- **JS**: user-visible strings via `flowbotI18n(key, enFallback)`; add keys to the catalogs and run `scripts/update_client_ids.py` to refresh `clientMessageIDs`; no parallel translation tables in `public/js/`.
- **ClientJSON**: marshal via `pkg/i18n.ClientMessages`; embed with `@templ.JSONScript("flowbot-i18n", i18n.ClientMessages(ctx))` in `partials.I18nScript` — inline `<script>` bodies do not interpolate Go expressions.
- **Message IDs**: dotted keys — `nav.*`, `page.*`, `toast.*`, `confirm.*`, `life.*`, `clip.*`, `client.*`, `auth.*`, `common.*`, `error.*`; ship en/zh pairs.

## Testing

Which layer: [docs/testing/README.md](../../../docs/testing/README.md). Owning BDD for `/service/web` pages: `tests/specs/*_page_spec_test.go` (`agents`, `agent_sessions`, `agent_scheduled_tasks`, `notifications`, `event`, `home_token_usage`) and `life_spec_test.go`.

```bash
go test ./internal/modules/web/...
```

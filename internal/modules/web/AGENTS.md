# Web UI Module

Server-rendered HTML pages with HTMX + Alpine.js interactivity.

## Structure

```text
internal/modules/web/
├── module.go                     # moduleHandler, Register(), Init(), Webservice(), Rules(), E2E helpers
├── rules.go                      # Aggregates all *WebserviceRules for route registration
├── auth.go                       # AuthConfig, cookie auth middleware, login rate limiter wiring
├── utils.go                      # Shared helpers (renderError, getUID)
├── home_webservice.go            # Home page route
├── login_webservice.go           # Login/logout routes and handlers
├── config_webservice.go          # Configs CRUD routes and handlers
├── healthz_webservice.go         # Health dashboard route and metrics collection
├── pipeline_webservice.go        # Pipeline-specific routes (CRUD, editor, run history, test)
├── view_webservice.go            # Shareable view page routes (create, view, delete)
├── view_types.go                 # View rendering types (viewTemplateFn, viewTemplates)
├── event_webservice.go           # Data events list and detail routes
├── homelab_webservice.go         # Homelab registry browser routes
├── hub_webservice.go             # Hub app management routes
├── notification_webservice.go    # Notification list routes
├── notify_settings_webservice.go # Notify channel/rule CRUD routes
├── token_webservice.go           # API token management routes
├── agent_sessions_webservice.go # Agent sessions routes (list, detail, run history)
├── agent_skills_webservice.go   # Agent skills routes (list, enable/disable)
├── agents_webservice.go         # Agents listing / chat UI routes
├── agent_memory_webservice.go   # Agent memory modal routes
├── agent_scheduled_tasks_webservice.go # Agent scheduled tasks routes
├── agent_subagents_webservice.go       # Agent subagents routes
├── chatagent_permissions_webservice.go # Chatagent permission policy routes
├── chatagent_web_stream.go      # Chatagent web SSE/stream helpers
├── relations_webservice.go       # Resource relations graph routes
├── ratelimit.go                  # Login rate limiter
├── module_test.go                # Module lifecycle unit tests
├── auth_test.go                  # Auth middleware tests
├── login_webservice_test.go      # Login/logout/rate limit tests
├── config_webservice_test.go     # Configs CRUD tests
├── healthz_webservice_test.go    # Health dashboard tests
├── notify_settings_webservice_test.go # Notify settings validation and auth tests
├── rules_test.go                 # Route group registration tests
├── agent_sessions_webservice_test.go # Agent sessions tests
├── agent_skills_webservice_test.go   # Agent skills tests
├── test_helper_test.go           # E2E test helpers
└── *_test.go                     # Co-located tests per webservice file

> **Legacy layout:** Older docs may reference `webservice.go`. It was split into `home_webservice.go`, `login_webservice.go`, `config_webservice.go`, `healthz_webservice.go`, plus shared `auth.go` and `rules.go`. Register new routes in the matching `*_webservice.go` file and append its rule slice to `allWebserviceRules` in `rules.go`.

pkg/views/
├── layout/
│   └── base.templ                # Global HTML skeleton (htmx, alpinejs, daisyui, tailwind, chart.js)
├── pages/
│   ├── capabilities.templ        # Capability listing page
│   ├── configs.templ             # ConfigsPage
│   ├── events.templ              # Data events list page
│   ├── healthz.templ             # Health status page
│   ├── home.templ                # HomePage
│   ├── homelab.templ             # Homelab registry page
│   ├── homelab_detail.templ      # Homelab app detail page
│   ├── hub_app_detail.templ      # Hub app detail page
│   ├── hub_apps.templ            # Hub apps list page
│   ├── login.templ               # LoginPage, LoginForm
│   ├── notifications.templ       # Notifications page
│   ├── agents.templ              # Agents listing / chat entry
│   ├── agent_sessions.templ      # Agent sessions list page
│   ├── agent_session_detail.templ # Agent session detail page
│   ├── agent_skills.templ        # Agent skills page
│   ├── agent_subagents.templ     # Agent subagents page
│   ├── agent_scheduled_tasks.templ # Scheduled tasks list
│   ├── agent_scheduled_task_detail.templ # Scheduled task detail
│   ├── chatagent_permissions.templ # Chatagent permissions page
│   ├── notify_settings.templ     # Notify channels/rules settings page
│   ├── pipeline_editor.templ     # PipelineEditorPage (SPA: Alpine.js)
│   ├── pipeline_list.templ       # PipelineListPage
│   ├── pipeline_run_live.templ   # Live pipeline run view
│   ├── pipeline_runs.templ       # PipelineRunsPage
│   ├── relations.templ           # Resource relations graph page
│   ├── tokens.templ              # API tokens page
│   └── view.templ                # ViewPage (shareable content)
└── partials/
    ├── helpers.go                # Shared Go helper functions
    ├── notify_settings_helpers.go
    ├── token_helpers.go
    ├── agent_session_helpers.go
    ├── agent_skill_helpers.go
    ├── capability_card.templ     # Capability card component
    ├── capability_grid.templ     # Capability grid layout
    ├── config_form.templ         # ConfigForm
    ├── config_row.templ          # ConfigRow
    ├── config_table.templ        # ConfigTable
    ├── confirm_modal.templ       # Global confirmation modal
    ├── data_events_table.templ   # Data events list table
    ├── empty_state.templ         # Empty state placeholder
    ├── event_filters.templ       # Event timeline filter controls
    ├── event_pagination.templ    # Event pagination controls
    ├── event_payload.templ       # Event payload detail
    ├── healthz_status.templ      # Health check status display
    ├── homelab_card.templ        # Homelab app card
    ├── homelab_grid.templ        # Homelab registry grid
    ├── hub_apps_table.templ      # Hub apps table
    ├── notifications_table.templ # Notifications table
    ├── agent_session_entries.templ    # Agent session entries list
    ├── agent_session_entry_payload.templ # Agent session entry payload detail
    ├── agent_session_entry_row.templ   # Agent session entry row
    ├── agent_session_row.templ         # Agent session row
    ├── agent_session_state_badge.templ # Agent session state badge
    ├── agent_session_table.templ       # Agent sessions table
    ├── agent_skill_form.templ          # Agent skill form
    ├── agent_skill_row.templ           # Agent skill row
    ├── agent_skill_table.templ         # Agent skills table
    ├── agent_skill_file_form.templ     # Agent skill file form
    ├── agent_skill_file_row.templ      # Agent skill file row
    ├── agent_skill_file_table.templ    # Agent skill files table
    ├── agent_memory_modal.templ        # Agent memory modal
    ├── agent_resource_preview.templ    # Agent resource preview
    ├── agent_subagent_form.templ       # Subagent form
    ├── agent_subagent_row.templ        # Subagent row
    ├── agent_subagent_table.templ      # Subagents table
    ├── agent_subagent_task_row.templ   # Subagent task row
    ├── agent_subagent_task_table.templ # Subagent tasks table
    ├── agent_scheduled_task_*.templ    # Scheduled task forms/tables/runs
    ├── chatagent_*.templ               # Chatagent thread/message/tool/approval/permissions UI
    ├── chatagent_helpers.go            # Chatagent template helpers
    ├── chatagent_types.go              # Chatagent view types
    ├── chatagent_markdown_sanitize.go  # Markdown sanitization for chatagent
    ├── chatagent_permissions_helpers.go
    ├── notify_channel_form.templ # Notify channel form
    ├── notify_channel_row.templ  # Notify channel row
    ├── notify_channels_table.templ # Notify channels table
    ├── notify_rule_form.templ    # Notify rule form
    ├── notify_rule_row.templ     # Notify rule row
    ├── notify_rules_table.templ  # Notify rules table
    ├── pipeline_list.templ       # PipelineListTable
    ├── pipeline_partials.templ   # TriggerCard, StepCard
    ├── pipeline_runs.templ       # PipelineRunsTable, PipelineStepRunsDetail
    ├── pipeline_stats.templ      # Pipeline stats dashboard
    ├── relation_detail.templ     # Relation details
    ├── relation_edge.templ       # Relation edge component
    ├── relation_node.templ       # Relation node component
    ├── relation_search.templ     # Relation search input
    ├── relation_tree.templ       # Relation tree view
    ├── token_form.templ          # Token form
    ├── token_row.templ           # Token row
    ├── token_table.templ         # Token table
    ├── token_usage.templ         # Token usage display
    ├── view_expired.templ        # Expired page placeholder
    ├── view_form.templ           # Read-only form partial
    ├── view_image.templ          # Image content partial
    ├── view_markdown.templ       # Markdown content partial
    ├── view_pipeline_run.templ   # Pipeline run content partial
    ├── view_text.templ           # Plain text content partial
    ├── webhook_logs_table.templ  # Webhook log entries table
    └── webhook_payload.templ     # Webhook payload detail
```

## Architecture

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Templates | [templ](https://templ.guide) v0.3 | Server-side HTML rendering, type-safe Go templates |
| Interactivity | [HTMX 2.x](https://htmx.org) | Partial page updates, form submissions, click-to-load |
| SPA components | [Alpine.js 3.x](https://alpinejs.dev) | Pipeline editor (visual/code modes, undo/redo, drawer), theme toggle |
| CSS | [DaisyUI v5](https://daisyui.com) + Tailwind utilities | Committed `public/css/app.css` + Alpine CSP (`alpine.csp.min.js`); no `node_modules` |
| Charts | [Chart.js](https://www.chartjs.org) | Pipeline stats and data visualizations |
| YAML handling | [js-yaml](https://github.com/nodeca/js-yaml) | YAML-to-JSON conversion in pipeline editor |
| Diff viewing | [diff](https://github.com/kpdecker/jsdiff) | Pipeline definition diff display |
| Static embedding | `embed.FS` (project-root `webassets.go`) | All CSS/JS/vendor served from binary (package `webassets`), no runtime filesystem dependency |

## Frontend Dependencies

All JavaScript and CSS dependencies are vendored under `public/` and served via `/static/*`. No project-local `node_modules` and no CDN at runtime.

| File | Purpose |
|------|---------|
| `public/css/app.css` | Committed Tailwind utilities + DaisyUI themes (static; not built via npm in-repo) |
| `public/css/custom.css` | Ad-hoc Flowbot tokens / shell styles |
| `public/vendor/alpine.csp.min.js` | Alpine.js CSP build (no `unsafe-eval`) |
| `public/vendor/htmx.min.js` | HTMX 2.x |
| `public/vendor/chart.js.min.js` | Chart.js |
| `public/vendor/js-yaml.min.js` | YAML parser (pipeline editor) |
| `public/vendor/diff.min.js` | Text diff library (pipeline diff) |
| `public/vendor/daisyui.css` / `themes.css` | Optional reference copies (`scripts/vendor.sh`) |
| `public/js/app.js` | Application bootstrap |
| `public/js/confirm.js` | Global confirmation dialog |
| `public/js/pipeline-editor.js` | Pipeline editor (Alpine.js component) |
| `public/js/pipeline-stats.js` | Pipeline stats charts |
| `public/js/pipeline-run-live.js` | Live pipeline run viewer |
| `public/js/event-filters.js` | Event timeline filter controls |
| `public/js/homelab-registry.js` | Homelab registry interactions |
| `public/js/chatagent-*.js` | Chatagent chat UI (namespaced, split modules) |
| `public/js/chatagent-permissions.js` | Chatagent permissions UI |
| `public/js/token-usage.js` | Token usage charts |
| `public/vendor/katex/` | KaTeX math rendering (chatagent markdown) |
| `public/css/chatagent-markdown.css` | Chatagent markdown styles |
| `public/css/styles.css` | Additional shared styles |

## Template Conventions

- **Pages** (`pkg/views/pages/`): Full-page templates wrapping content in `@layout.Base(title)`. Package `pages`.
- **Partials** (`pkg/views/partials/`): Fragment templates rendered standalone or as HTMX responses. Package `partials`. May contain shared Go helper functions.
- **Layout** (`pkg/views/layout/`): Global HTML skeleton with `<nav>`, local vendor script tags, CSS links. Package `layout`.
- Pages import partials: `import "github.com/flowline-io/flowbot/pkg/views/partials"` and call `@partials.Xxx()`.
- Do not put multi-line inline CSS; use Tailwind utility classes or DaisyUI component classes.
- Test IDs use `data-testid="kebab-case"` on interactive elements.
- Generated `*_templ.go` files are regenerated via `go tool templ generate` (or `go tool task` targets that invoke it). Never edit generated files.
- Always regenerate after changing `.templ` files.

## Chatagent JavaScript guardrails

The chatagent UI scripts are split into multiple files under `public/js/chatagent-*.js` and share a single global namespace:

- `window.FlowbotChatAgent = window.FlowbotChatAgent || {}`

Rules:

- Do not re-introduce a monolithic `public/js/chatagent-chat.js` that contains all logic.
- Keep cross-file APIs on `FlowbotChatAgent` only (no implicit globals).
- Pages that use chatagent must load the scripts **in dependency order** via `pkg/views/partials/chatagent_scripts.templ`:
  - `ChatAgentComposerScripts` — `util → chat` (agents home composer)
  - `ChatAgentApprovalScripts` — `util → approval → chat` (session detail approval panel)
  - `ChatAgentThreadScripts` — full stack `util → sse → markdown → context → approval → thread → chat` (chat thread page)

## Route Conventions

- All web routes are prefixed: `/service/web/*`
- Routes defined in package-level `var ...Rules = []webservice.Rule{...}` and aggregated in `rules.go` (`allWebserviceRules`)
- `module.go` loops `allWebserviceRules` and calls `module.Webservice(app, Name, rules)` for each group.
- General web routes use `route.WithNotAuth()` so Fiber authorization middleware skips scope checks; handlers still call `authenticateWeb()` for cookie-based login. Pipeline/API routes that need scopes omit `WithNotAuth` and use default scope-based auth.
- Standard verbs:
  - `GET /resource` → full page (calls `pages.XxxPage().Render(...)`)
  - `GET /resource/list` → table fragment for HTMX refresh (calls `partials.XxxTable().Render(...)`)
  - `GET /resource/new` → form fragment for HTMX injection (calls `partials.XxxForm().Render(...)`)
  - `POST /resource` → create, returns redirect or inline error
  - `PUT /resource/:id` → update, returns JSON or HTML fragment
  - `DELETE /resource/:id` → delete, returns refreshed table fragment
- Set `c.Type("html")` before rendering HTML responses.
- JSON API endpoints return `c.JSON(fiber.Map{...})`.

## HTMX Patterns

- **Full page redirect**: `ctx.Set("HX-Redirect", url)` + `return ctx.SendStatus(200)`.
- **Partial table refresh**: `hx-get="/service/web/.../list"` + `hx-target="#table-container"` + `hx-swap="outerHTML"`.
- **Inline form injection**: `hx-get="/service/web/.../new"` + `hx-target="#rows-container"` + `hx-swap="afterbegin"`.
- **OOB cleanup**: Return HTML fragments with `hx-swap-oob="delete"` from handler bodies (used for removing stale empty-state rows and form injection placeholders).
- **Form errors**: Use `renderFormError(ctx, "#form-error", msg)` (sets `HX-Retarget` + `HX-Reswap: innerHTML`). Forms must include an empty `<div id="form-error" data-testid="form-error-slot"></div>`.
- **Action-only errors**: Use `toastError(ctx, msg)` / `setShowToast` when there is no form slot (e.g. delete, test connection). Do not blindly retarget list/table swaps.
- **Click-to-expand**: `hx-get="..." + hx-trigger="click" + hx-target="next tr..." + hx-swap="innerHTML show:top"`. Use inline `onclick` with `return false` to toggle collapse.
- **Delete confirmation**: Prefer `data-confirm*` + `confirm.js` over bare `hx-confirm` when destructive.
- HTMX loaded from local vendor: `/static/vendor/htmx.min.js`.

## Error & loading conventions

- User-visible failures must use toast (`showToast` / `HX-Trigger showToast`) or inline `FormError` — never `console.error` alone.
- Intentionally silent failures (CSRF warm-up, cancel races) must be marked `intentionally silent` in a short comment.
- Dual-channel client rule in `public/js/app.js`: HTML/`HX-Retarget` error bodies swap inline (no toast); network/`sendError`/`timeout` and non-HTML errors toast; `401` redirects to login with `next`.
- Do not default every `renderError` to Retarget — that breaks DELETE/table refresh targets. Prefer explicit `renderFormError` or `toastError`.
- Loading: use `partials.PanelSkeleton` (delayed soft spinner) inside `hx-trigger="load"` containers; button spinners keep `.htmx-indicator`. No global progress bar and no large DaisyUI gray skeleton slabs for panel loads.

## Alpine.js Usage (Pipeline Editor)

- Defined in `public/js/pipeline-editor.js` as `pipelineEditor()`.
- Mounted via `x-data="pipelineEditor()" x-init="init()"` in `pipeline_editor.templ`.
- State is Alpine.js `x-data` only; no separate JavaScript framework.
- Capabilities loaded at init via `GET /pipelines/capabilities`.
- YAML-to-visual synced via `onYamlChange()` (code → visual) and `toYaml()` (visual → code).
- Undo/redo stack in Alpine state, persisted to server via `PUT /pipelines/:name`.
- Trigger cards and step cards are templ partials rendered with Alpine directives (`:class`, `@click`, `x-text`).
- Alpine.js loaded from local vendor: `/static/vendor/alpine.csp.min.js`.
- **CSP-safe expressions only** in templates: no `?.`, `??`, object/array literals, or spread (`...`) inside `x-text` / `x-for` / `@click` attributes. Put that logic in `pipeline-editor.js` helpers instead.

## Visual design (ops console)

Product goal: a refined **homelab ops console**, not a marketing landing page and not a generic DaisyUI admin scaffold. Brand seeds: favicon, `BrandMark`, teal primary. Do not copy the marketing site (`docs/website/` abyss/cyan) into the app UI.

### Direction

| Axis | Rule |
|------|------|
| Color | Teal is the only brand accent. Semantic colors only for status (`success` / `warning` / `error`). **Never** hardcode violet / indigo / purple for brand, focus rings, session Active, or Chart.js / context palettes. |
| Surfaces | Prefer border partitions or a single surface. Use `.flowbot-surface` for tables and panels. Reserve DaisyUI `card` + heavy shadow for interactive containers only. Hierarchy comes from border, spacing, and type weight — not decorative gradients, noise textures, or glassmorphism. |
| Radius | Prefer square-ish radii (`--flowbot-radius` ≈ `0.375rem`, `--flowbot-radius-box` ≈ `0.5rem`). Status uses `.flowbot-chip` (small radius). Avoid `rounded-full` pills for badges/metadata (toggles OK; letter avatars use `rounded-md`). |
| Typography | Self-hosted **IBM Plex Sans** in `public/fonts/` (woff2 latin subsets, weights 400 / 500 / 600) via `custom.css` `@font-face`. Body 400/500, titles 600 + `tracking-tight`. No Google Fonts CDN. Avoid all-caps `uppercase tracking-wide` metric labels. |
| Shadow | At most one light shadow (`--flowbot-shadow`). No dual-color glow rings. |
| Theme | Polish **light** and **dark** only. Advanced DaisyUI novelty themes may remain in the picker but are not the product visual language. Overlays (e.g. Context Usage popover) must use `base-*` / content tokens — never hardcode a dark-only panel. |

### Tokens and files

- Shell tokens live under `.flowbot-shell` in `public/css/custom.css` (`--flowbot-brand`, radius, shadow, focus ring, `--chatagent-context-free`). Do not invent a parallel surface palette that fights DaisyUI `base-100/200/300`.
- Override DaisyUI `light` / `dark` **primary**, **secondary**, and **accent** to teal family so defaults are not pink/purple.
- Chart / context JS palettes: teal + neutral greys + semantic colors only (`public/js/token-usage.js`, `pipeline-stats.js`, `chatagent-context.js`).

### Components

- **Page header**: use `partials.PageHeader(title, subtitle)` — `text-2xl font-semibold tracking-tight` + optional subtitle; actions as children. Do not invent per-page `font-bold` / `card-title` page titles.
- **Chips vs badges**: status → `.flowbot-chip` (+ `flowbot-chip-success|error|warning|muted|primary`). Metadata (scopes, capability names, ports) → plain or `font-mono` text joined with ` · `, truncated when needed — **no** flex-wrap badge walls. Do not use DaisyUI `badge-success|error|warning|ghost` for run/health/status.
- **Empty states**: simple copy + optional primary CTA via `EmptyState` / `EmptyStateCTA`; no centered large card shell.
- **Join / range tabs**: keep segmented controls; square corners via `.flowbot-shell .join` rules (not pill `rounded-full`).
- **Navbar**: border-bottom emphasis over drop shadow; brand mark + “Flowbot” with `font-semibold tracking-tight`; dropdown menus use `border border-base-300 shadow-sm`.
- **Agents composer**: square-ish send button (not circular ChatGPT-style). Active session styles use teal, not purple.
- **Avatars**: rounded square letter marks aligned with favicon tile radius language.
- **Loading**: `partials.PanelSkeleton` (delayed soft spinner) for HTMX panel loads — not large gray DaisyUI skeleton slabs.

### Visual Do / Don’t

| Do | Don’t |
|----|--------|
| `.flowbot-surface` for tables/panels | `card bg-base-100 shadow-sm` list shells |
| `.flowbot-chip-*` for status | `badge-success` / `badge-error` status |
| `PageHeader` for page titles | Mixed `font-bold` / `card-title` / `text-xl` titles |
| Teal brand + semantic status colors | Purple / indigo brand accents |
| Clip primary aligned to `--flowbot-brand` | Clip blue `#2563eb` fork |
| light / dark as product themes | Treating Advanced DaisyUI themes as product UI |

### Anti-patterns

- Full-bleed marketing heroes, illustration empty states, multi-layer shadows, purple-to-indigo gradients
- Every block wrapped in `card bg-base-100 shadow-sm`
- Hardcoded hex for theme surfaces/text (use `var(--color-*)` / DaisyUI tokens)
- Mixing `docs/website/` visual language into `/service/web`

## CSS / DaisyUI

- Framework: [DaisyUI v5](https://daisyui.com) utilities via committed `public/css/app.css`
- Delivery: static files under `public/` (embedded); **no** in-repo npm/`node_modules` CSS build
- Alpine: CSP build at `/static/vendor/alpine.csp.min.js` (refresh via `scripts/vendor.sh`)
- Custom CSS: `public/css/custom.css` for tokens, chips, surfaces
- Do not load `tailwind-browser.min.js` in layouts
## Static Assets

- Directory: `public/` (embedded via `//go:embed all:public` in `webassets.go`).
- JavaScript: `public/js/` — Alpine/page scripts (`pipeline-editor.js`, `app.js`, `confirm.js`, `pipeline-stats.js`, `event-filters.js`, `homelab-registry.js`, `pipeline-run-live.js`, `chatagent-chat.js`, `chatagent-permissions.js`, `token-usage.js`).
- Vendor libraries: `public/vendor/` — third-party JS/CSS vendored locally (daisyui, tailwind, htmx, alpinejs, chart.js, js-yaml, diff, katex).
- Served via: `app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))`.
- All script dependencies are local — no external CDN requests in production.

## Authentication

- Cookie-based: `accessToken` HttpOnly + SameSite=Lax cookie; `Secure` via `modules.web.auth.cookie_secure` (default true).
- CSRF (double-submit): non-HttpOnly `csrfToken` cookie issued on safe `/service/web` requests; mutations with a session cookie (or `POST /login`) must send matching `X-CSRF-Token` header or `csrf_token` form field. HTMX/fetch helpers in `public/js/app.js` attach the header automatically.
- API tokens: `X-AccessToken` or `Authorization: Bearer` (query/form `accessToken` is not accepted).
- `authenticateWeb()` / `isAuthenticated()` read the cookie, look up the token in store, populate `route.RequestContext`.
- Routes typically combine `route.WithNotAuth()` (skip scope middleware) with an explicit `authenticateWeb(ctx)` call in the handler; unauthenticated users redirect to `/service/web/login`.
- Login accepts credentials from `flowbot.yaml` → `modules.web.auth` (`password` plaintext for dev, or `password_hash` bcrypt for production). Init rejects empty/weak defaults and short plaintext passwords.
- Token stored hashed via `store.Database.ParameterSet(auth.HashToken(raw), ...)`, expires in 24h; logout deletes the hash entry.
- Unit/BDD helpers: `AttachCSRFForTest(req)` / `addWebAuth(req)` for state-changing requests.

## Store Access

- Web handlers access store via the `store.Database` singleton.
- Never import ent schema/types directly in templates — pass structs as template args.
- Never write DB queries in handlers — all queries live in the `internal/store` package (`store.go` + `postgres/adapter.go`).

## Testing

- Unit tests: `*_test.go` co-located, table-driven with `require`/`assert`.
- E2E helpers: `InitForE2E()` and `MountForE2E()` in `module.go` for integration test setup.
- Test IDs: Use `data-testid="..."` on all interactive elements in templates.
- Mock store where possible; use real SQLite in-memory for store-level tests.

## Anti-Patterns

- Never put view templates under `internal/modules/` — use `pkg/views/`.
- Never mix page and partial templates in the same `.templ` file — split into `pages/` and `partials/`.
- Never use `encoding/json` Marshal/Unmarshal — use `github.com/bytedance/sonic`. `json.RawMessage` type from stdlib is allowed.
- Never return JSON from an endpoint that HTMX expects as HTML — set `c.Type("html")`.
- Never inline complex JavaScript in templates — put it in `public/js/`.
- Never skip `data-testid` on interactive elements.
- Never use `<script>` tags in partial templates — scripts belong in `base.templ` or `public/js/`.
- Never pass unsanitized HTML to `templ.Raw` — markdown must use `utils.MarkdownToSafeHTML` (or escaped plain text) first.
- Never hardcode URLs in templates — use `templ.URL()` for dynamic paths.
- Never call provider clients directly from web handlers — use `capability.Invoke`.
- Never render error pages as full HTML for HTMX requests — return error fragments or set `HX-Retarget`.
- Never reference CDN URLs for frontend dependencies — all deps are vendored in `public/vendor/`.

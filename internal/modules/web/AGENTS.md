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
├── relations_webservice.go       # Resource relations graph routes
├── ratelimit.go                  # Login rate limiter
├── module_test.go                # Module lifecycle unit tests
├── auth_test.go                  # Auth middleware tests
├── login_webservice_test.go      # Login/logout/rate limit tests
├── config_webservice_test.go     # Configs CRUD tests
├── healthz_webservice_test.go    # Health dashboard tests
├── notify_settings_webservice_test.go # Notify settings validation and auth tests
├── rules_test.go                 # Route group registration tests
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
| CSS | [DaisyUI v5](https://daisyui.com) | Component CSS (built on Tailwind CSS v4) |
| Charts | [Chart.js](https://www.chartjs.org) | Pipeline stats and data visualizations |
| YAML handling | [js-yaml](https://github.com/nodeca/js-yaml) | YAML-to-JSON conversion in pipeline editor |
| Diff viewing | [diff](https://github.com/kpdecker/jsdiff) | Pipeline definition diff display |
| Static embedding | `embed.FS` (project-root `webassets.go`) | All CSS/JS/vendor served from binary (package `webassets`), no runtime filesystem dependency |

## Frontend Dependencies

All JavaScript and CSS dependencies are vendored locally in `public/vendor/` and served via `/static/vendor/*` paths. No CDN references in production.

| File | Purpose |
|------|---------|
| `public/vendor/daisyui.css` | DaisyUI v5 component styles |
| `public/vendor/themes.css` | DaisyUI theme definitions |
| `public/vendor/tailwind-browser.min.js` | Tailwind CSS v4 (browser runtime) |
| `public/vendor/htmx.min.js` | HTMX 2.x |
| `public/vendor/alpine.min.js` | Alpine.js 3.x |
| `public/vendor/chart.js.min.js` | Chart.js |
| `public/vendor/js-yaml.min.js` | YAML parser (pipeline editor) |
| `public/vendor/diff.min.js` | Text diff library (pipeline diff) |
| `public/css/custom.css` | Ad-hoc custom styles |
| `public/js/app.js` | Application bootstrap |
| `public/js/confirm.js` | Global confirmation dialog |
| `public/js/pipeline-editor.js` | Pipeline editor (Alpine.js component) |
| `public/js/pipeline-stats.js` | Pipeline stats charts |
| `public/js/pipeline-run-live.js` | Live pipeline run viewer |
| `public/js/event-filters.js` | Event timeline filter controls |
| `public/js/homelab-registry.js` | Homelab registry interactions |

## Template Conventions

- **Pages** (`pkg/views/pages/`): Full-page templates wrapping content in `@layout.Base(title)`. Package `pages`.
- **Partials** (`pkg/views/partials/`): Fragment templates rendered standalone or as HTMX responses. Package `partials`. May contain shared Go helper functions.
- **Layout** (`pkg/views/layout/`): Global HTML skeleton with `<nav>`, local vendor script tags, CSS links. Package `layout`.
- Pages import partials: `import "github.com/flowline-io/flowbot/pkg/views/partials"` and call `@partials.Xxx()`.
- Do not put multi-line inline CSS; use Tailwind utility classes or DaisyUI component classes.
- Test IDs use `data-testid="kebab-case"` on interactive elements.
- Generated `*_templ.go` files are regenerated via `templ generate pkg/views/...`. Never edit generated files.
- Always regenerate after changing `.templ` files.

## Route Conventions

- All web routes are prefixed: `/service/web/*`
- Routes defined in package-level `var ...Rules = []webservice.Rule{...}`
- Filed under `module.Webservice(app, Name, ...Rules)` in `module.go`.
- General web routes (home, login, configs) use `route.WithNotAuth()` which validates cookie-based tokens only (no scope check). Pipeline routes use default scope-based authentication.
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
- **Form errors**: Return error HTML fragment and set `HX-Retarget` + `HX-Reswap` headers to position error message before the form.
- **Click-to-expand**: `hx-get="..." + hx-trigger="click" + hx-target="next tr..." + hx-swap="innerHTML show:top"`. Use inline `onclick` with `return false` to toggle collapse.
- **Delete confirmation**: `hx-confirm="Are you sure?"` on button.
- HTMX loaded from local vendor: `/static/vendor/htmx.min.js`.

## Alpine.js Usage (Pipeline Editor)

- Defined in `public/js/pipeline-editor.js` as `pipelineEditor()`.
- Mounted via `x-data="pipelineEditor()" x-init="init()"` in `pipeline_editor.templ`.
- State is Alpine.js `x-data` only; no separate JavaScript framework.
- Capabilities loaded at init via `GET /pipelines/capabilities`.
- YAML-to-visual synced via `onYamlChange()` (code → visual) and `toYaml()` (visual → code).
- Undo/redo stack in Alpine state, persisted to server via `PUT /pipelines/:name`.
- Trigger cards and step cards are templ partials rendered with Alpine directives (`:class`, `@click`, `x-text`).
- Alpine.js loaded from local vendor: `/static/vendor/alpine.min.js`.

## CSS / DaisyUI

- Framework: [DaisyUI v5](https://daisyui.com) (built on Tailwind CSS v4)
- Delivery: Local vendor files in `public/vendor/`, embedded via `webassets.go`, served at `/static/vendor/*`
- No CDN references; no local build step required
- Theme: `data-theme="light"` on `<html>`, with runtime theme switcher (Alpine.js, persisted to localStorage)
- Custom CSS: `public/css/custom.css` for ad-hoc styles (e.g. `.var-pill`), served via embedded `webassets.FS`
- Component classes: Use `btn`, `card`, `badge`, `table`, `navbar`, `alert`, `input`, `select`, `textarea`, `modal`, `dropdown`, `toast`, etc.
- Color tokens: `base-100/200/300` (surfaces), `base-content` (text), `primary` (actions), `error/success/warning` (states)

## Static Assets

- Directory: `public/` (embedded via `//go:embed all:public` in `webassets.go`).
- JavaScript: `public/js/` — Alpine.js components (`pipeline-editor.js`), utility scripts (`app.js`, `confirm.js`), charts (`pipeline-stats.js`), page-specific interactivity (`event-filters.js`, `homelab-registry.js`, `pipeline-run-live.js`).
- Vendor libraries: `public/vendor/` — third-party JS/CSS vendored locally (daisyui, tailwind, htmx, alpinejs, chart.js, js-yaml, diff).
- Served via: `app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))`.
- All script dependencies are local — no external CDN requests in production.

## Authentication

- Cookie-based: `accessToken` HTTP-only cookie.
- Middleware: `authenticateWeb()` reads cookie, looks up token in store, populates `route.RequestContext`.
- Routes use `route.WithNotAuth()` which calls `authenticateWeb()` and redirects to `/service/web/login` on failure.
- Login accepts username/password from `flowbot.yaml` → `modules.web.auth`.
- Token stored via `store.Database.ParameterSet()`, expires in 24h.

## Store Access

- Web handlers access store via the `store.Database` singleton.
- Never import ent schema/types directly in templates — pass structs as template args.
- Never write DB queries in handlers — all queries live in `internal/store/store.go`.

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
- Never hardcode URLs in templates — use `templ.URL()` for dynamic paths.
- Never call provider clients directly from web handlers — use `ability.Invoke`.
- Never render error pages as full HTML for HTMX requests — return error fragments or set `HX-Retarget`.
- Never reference CDN URLs for frontend dependencies — all deps are vendored in `public/vendor/`.

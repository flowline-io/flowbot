# Web UI Module

Server-rendered HTML pages with HTMX + Alpine.js interactivity.

## Structure

```text
internal/modules/web/
├── module.go               # moduleHandler, Register(), Init(), Webservice(), Rules()
├── webservice.go           # General routes (home, login, configs CRUD), auth middleware
├── pipeline_webservice.go  # Pipeline-specific routes (CRUD, editor, run history, test)
├── view_webservice.go      # Shareable view page routes (create, view, delete)
├── types.go                # View rendering types (viewTemplateFn, viewTemplates)
├── ratelimit.go            # Login rate limiter
├── module_test.go          # Unit tests
└── test_helper_test.go     # E2E test helpers

pkg/views/
├── layout/
│   └── base.templ          # Global HTML skeleton (htmx, alpinejs, js-yaml, daisyui)
├── pages/
│   ├── home.templ          # HomePage
│   ├── login.templ         # LoginPage, LoginForm
│   ├── configs.templ       # ConfigsPage
│   ├── view.templ          # ViewPage (shareable content)
│   ├── pipeline_list.templ     # PipelineListPage
│   ├── pipeline_editor.templ   # PipelineEditorPage (SPA: Alpine.js)
│   └── pipeline_runs.templ     # PipelineRunsPage
└── partials/
    ├── helpers.go               # Shared Go helper functions
    ├── config_form.templ        # ConfigForm
    ├── config_row.templ         # ConfigRow
    ├── config_table.templ       # ConfigTable
    ├── pipeline_list.templ      # PipelineListTable
    ├── pipeline_partials.templ  # TriggerCard, StepCard
    ├── pipeline_runs.templ      # PipelineRunsTable, PipelineStepRunsDetail
    ├── view_expired.templ       # Expired page placeholder
    ├── view_form.templ          # Read-only form partial
    ├── view_image.templ         # Image content partial
    ├── view_markdown.templ      # Markdown content partial
    ├── view_pipeline_run.templ  # Pipeline run content partial
    └── view_text.templ          # Plain text content partial
```

## Architecture

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Templates | [templ](https://templ.guide) v0.3 | Server-side HTML rendering, type-safe Go templates |
| Interactivity | [HTMX 2.x](https://htmx.org) | Partial page updates, form submissions, click-to-load |
| SPA components | [Alpine.js 3.x](https://alpinejs.dev) | Pipeline editor (visual/code modes, undo/redo, drawer) |
| CSS | [DaisyUI v5](https://daisyui.com) | Component CSS via CDN (built on Tailwind CSS) |
| YAML handling | [js-yaml](https://github.com/nodeca/js-yaml) | YAML ↔ JSON conversion in pipeline editor |
| Static embedding | `embed.FS` (project-root `webassets.go`) | CSS/JS served from binary (package `webassets`), no runtime filesystem dependency |

## Template Conventions

- **Pages** (`pkg/views/pages/`): Full-page templates wrapping content in `@layout.Base(title)`. Package `pages`.
- **Partials** (`pkg/views/partials/`): Fragment templates rendered standalone or as HTMX responses. Package `partials`. May contain shared Go helper functions.
- **Layout** (`pkg/views/layout/`): Global HTML skeleton with `<nav>`, CDN script tags, CSS link. Package `layout`.
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
- CDN: `https://unpkg.com/htmx.org@2.x.x/dist/htmx.min.js` loaded in `base.templ`.

## Alpine.js Usage (Pipeline Editor)

- Defined in `public/js/pipeline-editor.js` as `pipelineEditor()`.
- Mounted via `x-data="pipelineEditor()" x-init="init()"` in `pipeline_editor.templ`.
- State is Alpine.js `x-data` only; no separate JavaScript framework.
- Capabilities loaded at init via `GET /pipelines/capabilities`.
- YAML ↔ visual synced via `onYamlChange()` (code → visual) and `toYaml()` (visual → code).
- Undo/redo stack in Alpine state, persisted to server via `PUT /pipelines/:name`.
- Trigger cards and step cards are templ partials rendered with Alpine directives (`:class`, `@click`, `x-text`).
- CDN: `https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js` loaded in `base.templ`.

## CSS / DaisyUI

- Framework: [DaisyUI v5](https://daisyui.com) (built on Tailwind CSS v4)
- Delivery: CDN (`daisyui@5` + `@tailwindcss/browser@4` + `daisyui@5/themes.css`), no local build step
- Theme: `data-theme="light"` on `<html>`, configurable via themes.css CDN
- Custom CSS: `public/css/custom.css` for ad-hoc styles (e.g. `.var-pill`), served via embedded `webassets.FS`
- Component classes: Use `btn`, `card`, `badge`, `table`, `navbar`, `alert`, `input`, `select`, `textarea`, `modal`, etc.
- Color tokens: `base-100/200/300` (surfaces), `base-content` (text), `primary` (actions), `error/success/warning` (states)

## Static Assets

- Directory: `public/` (embedded via `//go:embed all:public` in `webassets.go`).
- JavaScript: `public/js/` — Alpine.js components (`pipeline-editor.js`), utility scripts (`app.js`).
- Served via: `app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))`.
- CDN scripts (htmx, alpinejs, js-yaml) loaded from external URLs in `base.templ`.

## Authentication

- Cookie-based: `accessToken` HTTP-only cookie.
- Middleware: `authenticateWeb()` reads cookie, looks up token in store, populates `route.RequestContext`.
- Routes use `route.WithNotAuth()` which calls `authenticateWeb()` and redirects to `/service/web/login` on failure.
- Login accepts username/password from `flowbot.yaml` → `modules.web.auth`.
- Token stored via `store.Database.ParameterSet()`, expires in 24h.

## Store Access

- Web handlers access store via helper `getPipelineDefStore()` or directly via `store.Database`.
- Never import ent schema/types directly in templates — pass `*gen.Xxx` structs as template args.
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

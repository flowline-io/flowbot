# Frontend Guide

Go-based Progressive Web App UI layer using go-app/v10.

## Structure

```
page/
├── layout.go             # HTML layout template (embed CSS)
├── styles.css            # Global styles (embed via //go:embed)
├── component/            # Reusable UI components
│   ├── admin_views.go   # Admin panel views
│   ├── admin_assets.go  # Admin asset loading
│   ├── form.go          # Form component
│   ├── html.go          # HTML rendering helpers
│   ├── table.go         # Table component
│   └── assets/          # Static assets (embed)
├── form/
│   └── builder.go       # Dynamic form builder from rules
├── library/
│   └── library.go       # UI library registry
└── uikit/               # 40+ UI components (flat files, go-app Compo each)
    ├── form.go, button.go, modal.go, table.go, card.go, navbar.go
    ├── accordion.go, alert.go, badge.go, carousel.go, dropdown.go
    ├── grid.go, flex.go, icon.go, image.go, pagination.go, notification.go
    └── ...              # datepicker.go, countdown.go, divider.go, etc.
```

## Architecture

Uses [go-app/v10](https://github.com/maxence-charriere/go-app) — Go WASM framework. Components implement `app.Compo` with `Render() app.UI`. CSS embedded via `//go:embed styles.css`. Served via Fiber routes: `/p/:id`, `/page/:id/:flag`.

```go
type Button struct {
    app.Compo
    Text    string
    OnClick app.EventHandler
}
func (c *Button) Render() app.UI { return app.Button().Body(app.Text(c.Text)) }
```

**Form Builder** (`form/builder.go`): Dynamic forms driven by `FormRule` definitions. Validates server-side.


## Anti-Patterns

- **Never** use raw HTML strings — use go-app components
- **Never** bypass the layout template
- **Never** add client-side-only state — server-renderable if possible
- **Always** embed static assets via `//go:embed`
- **Always** test UI components via `internal/bots/dev/`

## Commands

```bash
go test ./pkg/page/...       # Test page components
```

# Frontend Guide

Go-based UI using go-app/v10.

## Structure

```
page/
├── layout.go            # HTML layout (embed CSS)
├── styles.css           # Global styles
├── component/           # Reusable components (admin, form, table, html)
├── form/builder.go      # Dynamic form builder from FormRule
├── library/library.go   # UI library registry
└── uikit/               # 40+ UI components (button, modal, table, card, etc.)
```

## Architecture

Components implement `app.Compo` with `Render() app.UI`. CSS embedded via `//go:embed`. Served via Fiber: `/p/:id`, `/page/:id/:flag`.

## Rules

- Never use raw HTML strings — use go-app components
- Never bypass the layout template
- Always embed static assets via `//go:embed`
- Always test UI components via `internal/bots/dev/`

## Commands

```bash
go test ./pkg/page/...
```

# Admin PWA Development Guide

WebAssembly-based admin frontend using go-app/v10, DaisyUI, and Tailwind CSS.

**Generated:** 2026-03-11
**Go Version:** 1.26

## Quick Reference

| Task      | Command                 | Notes                |
| --------- | ----------------------- | -------------------- |
| Build PWA | `task build:app`        | Wasm + static server |
| Run dev   | `task air`              | Live reload          |
| Run tests | `go test ./cmd/app/...` | Unit tests           |
| Format    | `task format`           | go fmt + prettier    |

## Architecture

```
cmd/app/
├── main.go              # Entry point, route definitions
├── api/
│   └── client.go        # HTTP client for backend API
├── components/          # Reusable UI components
│   ├── layout.go        # Page layout wrapper
│   ├── navbar.go        # Navigation bar
│   ├── breadcrumb.go    # Breadcrumb navigation
│   ├── toast.go         # Toast notifications
│   ├── loading.go       # Loading overlay
│   ├── confirm_dialog.go # Confirmation modal
│   ├── form.go          # Form field components
│   ├── text.go          # Text utilities (highlighting)
│   └── keyboard.go      # Keyboard shortcuts
├── pages/               # Page components
│   ├── login.go         # OAuth login page
│   ├── dashboard.go     # Home dashboard
│   ├── users.go         # User management
│   ├── containers.go    # Container management
│   ├── workflows.go     # Workflow management
│   ├── bots.go          # Bot modules page
│   ├── logs.go          # Log viewer
│   └── settings.go      # System settings
├── state/
│   └── store.go         # Global state (auth, theme)
├── utils/
│   ├── debounce.go      # Debounce utility
│   └── pagination.go    # Pagination state helper
└── config/
    └── config.go        # Environment configuration
```

## Key Patterns

### Page Component

```go
type MyPage struct {
    app.Compo

    // State fields
    data    []MyData
    loading bool
    search  string
}

func (p *MyPage) OnNav(ctx app.Context) {
    if !state.IsAuthenticated(ctx) {
        ctx.Navigate("/admin/login")
        return
    }
    p.loadData(ctx)
}

func (p *MyPage) Render() app.UI {
    return components.WithLayout(
        // Page content...
    )
}
```

### API Call Pattern

```go
func (p *MyPage) loadData(ctx app.Context) {
    p.loading = true
    token := state.Token(ctx)

    ctx.Async(func() {
        resp, err := api.ListData(token, p.page, p.pageSize)
        ctx.Dispatch(func(ctx app.Context) {
            p.loading = false
            if err != nil {
                components.ShowToast(ctx, "Failed: "+err.Error(), "error")
                return
            }
            p.data = resp.Items
        })
    })
}
```

### Using Components

```go
// Layout wrapper
components.WithLayout(
    app.H1().Text("Page Title"),
    app.Div().Class("card").Body(...),
)

// Toast notification
components.ShowToast(ctx, "Success!", "success")

// Confirmation dialog
components.ConfirmDialog{
    Show:         p.showConfirm,
    Title:        "Delete Item",
    Message:      "Are you sure?",
    ConfirmLabel: "Delete",
    ConfirmClass: "btn-error",
    OnConfirm:    p.doDelete,
    OnCancel:     p.hideConfirm,
}

// Loading overlay
components.ShowLoading(ctx, "Saving...")
components.HideLoading(ctx)

// Text highlighting
components.HighlightText(text, searchQuery)
```

## State Management

### Authentication

```go
// Check auth status
if !state.IsAuthenticated(ctx) {
    ctx.Navigate("/admin/login")
    return
}

// Get/Set token
token := state.Token(ctx)
state.SetToken(ctx, newToken)
state.ClearToken(ctx)
```

### Theme

```go
// Get current theme
isDark := state.IsDarkMode(ctx)
theme := state.Theme(ctx)

// Toggle theme
state.ToggleTheme(ctx)
state.SetTheme(ctx, "dark")
```

## Utilities

### Debouncer

```go
debounce := utils.NewDebouncer(500 * time.Millisecond)
debounce.Call(ctx, func() {
    // Executed after 500ms of no new calls
})
```

### Pagination

```go
pagination := utils.NewPagination()
pagination.SetPageSize(20)
pagination.GoTo(2)
pagination.Next()
pagination.Prev()
offset := pagination.Offset()
visiblePages := pagination.VisiblePages()
```

## Styling

Uses DaisyUI component classes with Tailwind CSS utilities:

```go
// Buttons
app.Button().Class("btn btn-primary btn-sm")
app.Button().Class("btn btn-ghost btn-circle")

// Badges
app.Span().Class("badge badge-success badge-sm")

// Cards
app.Div().Class("card bg-base-100 shadow-md")

// Form inputs
app.Input().Class("input input-bordered w-full")

// Tables
app.Table().Class("table table-zebra w-full")

// Loading spinner
app.Span().Class("loading loading-spinner loading-lg text-primary")
```

## Routes

| Path                | Component  | Description          |
| ------------------- | ---------- | -------------------- |
| `/admin`            | Dashboard  | Home page            |
| `/admin/login`      | Login      | OAuth authentication |
| `/admin/users`      | Users      | User management      |
| `/admin/containers` | Containers | Container management |
| `/admin/workflows`  | Workflows  | Workflow management  |
| `/admin/bots`       | Bots       | Bot modules          |
| `/admin/logs`       | Logs       | Log viewer           |
| `/admin/settings`   | Settings   | System settings      |

## Testing

```bash
# Run all app tests
go test ./cmd/app/... -v

# Run specific package tests
go test ./cmd/app/components/... -v
go test ./cmd/app/utils/... -v
go test ./cmd/app/api/... -v
```

## Build & Deploy

```bash
# Build PWA
task build:app

# Output
bin/app-server    # Static file server
bin/app.wasm      # WebAssembly binary
```

## Code Style

- Follow `go fmt` formatting
- Use DaisyUI CSS classes for styling
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Match existing patterns in the codebase
- Keep components small and focused

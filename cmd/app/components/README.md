# Component Library

Reusable UI components for the Flowbot Admin PWA.

## Layout Components

### WithLayout

Wraps page content with navbar, footer, toast notifications, and loading overlay.

```go
func (p *MyPage) Render() app.UI {
    return components.WithLayout(
        app.H1().Text("Page Title"),
        app.Div().Class("card").Body(
            // Content...
        ),
    )
}
```

### WithMinimalLayout

Layout without navbar (for login page, etc).

```go
func (p *LoginPage) Render() app.UI {
    return components.WithMinimalLayout(
        app.Div().Class("card").Body(
            // Login form...
        ),
    )
}
```

## Navigation Components

### Navbar

Top navigation bar with logo, nav links, theme toggle, and user dropdown.

```go
&Navbar{}
```

Features:
- Logo and site name
- Desktop navigation links
- Mobile hamburger menu
- Theme toggle button
- Notification indicator
- User avatar dropdown with logout

### Breadcrumb

Automatic breadcrumb navigation based on current URL path.

```go
&Breadcrumb{}
```

Renders breadcrumbs for paths like `/admin/users` → Home / Users

## Feedback Components

### Toast

Display toast notifications.

```go
// Show toast
components.ShowToast(ctx, "Operation successful", "success")
components.ShowToast(ctx, "Something went wrong", "error")
components.ShowToast(ctx, "Please check your input", "warning")
components.ShowToast(ctx, "Here's some info", "info")
```

Types: `success`, `error`, `warning`, `info`

### LoadingOverlay

Full-screen loading overlay.

```go
// Show loading
components.ShowLoading(ctx, "Saving...")

// Hide loading
components.HideLoading(ctx)
```

### ConfirmDialog

Modal confirmation dialog.

```go
type MyPage struct {
    app.Compo
    showConfirm   bool
    confirmTitle  string
    confirmMsg    string
    confirmAction func()
}

func (p *MyPage) Render() app.UI {
    return components.WithLayout(
        // Page content...
        
        &components.ConfirmDialog{
            Show:         p.showConfirm,
            Title:        p.confirmTitle,
            Message:      p.confirmMsg,
            ConfirmLabel: "Delete",
            ConfirmClass: "btn-error",
            OnConfirm: func() {
                if p.confirmAction != nil {
                    p.confirmAction()
                }
                p.showConfirm = false
            },
            OnCancel: func() {
                p.showConfirm = false
            },
        },
    )
}
```

## Form Components

### FormField

Styled form field with label.

```go
func (s *Settings) formField(label, inputType, value, placeholder string, onChange app.EventHandler) app.UI {
    return app.Div().Class("form-control mb-4").Body(
        app.Label().Class("label").Body(
            app.Span().Class("label-text font-medium").Text(label),
        ),
        app.Input().
            Type(inputType).
            Class("input input-bordered w-full").
            Value(value).
            Placeholder(placeholder).
            OnChange(onChange),
    )
}
```

### FormValidator

Helper for form validation.

```go
validator := &components.FormValidator{}

validator.Required("name", name, "Name is required")
validator.Email("email", email, "Invalid email format")
validator.MinLength("password", password, 8, "Password must be at least 8 characters")

if validator.HasErrors() {
    for field, msg := range validator.Errors() {
        components.ShowToast(ctx, msg, "warning")
    }
    return
}
```

## Text Components

### HighlightText

Highlight search matches in text.

```go
// Highlight all occurrences of query in text
components.HighlightText("Hello World", "world")
// Renders: Hello <mark class="...">World</mark>

// Conditional highlighting
components.HighlightTextIf(text, query, shouldHighlight)
```

Features:
- Case-insensitive matching
- Preserves original case in output
- Uses DaisyUI-compatible styling

## Keyboard Components

### KeyboardHandler

Global keyboard shortcut handler.

```go
&KeyboardHandler{}
```

Built-in shortcuts:
- `/` - Focus search (dispatches "focus-search" action)
- `?` - Show help toast

To respond to keyboard events in your page:

```go
func (p *MyPage) OnMount(ctx app.Context) {
    ctx.Handle("focus-search", func(ctx app.Context, a app.Action) {
        // Focus your search input
    })
}
```

## Utility Types

### Debouncer

Debounce function calls.

```go
type MyPage struct {
    debounce *utils.Debouncer
}

func (p *MyPage) handleInput(ctx app.Context, e app.Event) {
    if p.debounce == nil {
        p.debounce = utils.NewDebouncer(500 * time.Millisecond)
    }
    p.debounce.Call(ctx, func() {
        // This runs 500ms after the last input
        p.doSearch(ctx)
    })
}
```

### Pagination

Pagination state management.

```go
type MyPage struct {
    pagination *utils.Pagination
}

func (p *MyPage) OnNav(ctx app.Context) {
    p.pagination = utils.NewPagination()
    p.pagination.SetPageSize(20)
}

func (p *MyPage) loadData(ctx app.Context) {
    offset := p.pagination.Offset()
    // Use offset in API call...
}

func (p *MyPage) handleNextPage(ctx app.Context, e app.Event) {
    if p.pagination.Next() {
        p.loadData(ctx)
    }
}

func (p *MyPage) renderPagination() app.UI {
    if !p.pagination.HasPages() {
        return app.Div()
    }
    
    buttons := make([]app.UI, 0)
    for _, page := range p.pagination.VisiblePages() {
        // Render page button...
    }
    return app.Div().Class("join").Body(buttons...)
}
```

## Styling Reference

### DaisyUI Classes

| Component | Classes |
|-----------|---------|
| Button | `btn`, `btn-primary`, `btn-secondary`, `btn-ghost`, `btn-sm`, `btn-circle` |
| Badge | `badge`, `badge-success`, `badge-warning`, `badge-error`, `badge-info` |
| Card | `card`, `bg-base-100`, `shadow-md` |
| Input | `input`, `input-bordered`, `input-sm` |
| Select | `select`, `select-bordered`, `select-sm` |
| Table | `table`, `table-zebra` |
| Loading | `loading`, `loading-spinner`, `loading-lg`, `loading-sm` |
| Modal | `modal`, `modal-open`, `modal-box`, `modal-action` |

### Tailwind Utilities

| Purpose | Classes |
|---------|---------|
| Flex | `flex`, `flex-col`, `gap-2`, `gap-4`, `items-center`, `justify-between` |
| Spacing | `p-4`, `px-6`, `py-2`, `mb-4`, `mt-2` |
| Text | `text-sm`, `text-lg`, `font-medium`, `font-bold` |
| Colors | `text-base-content`, `bg-base-100`, `text-primary`, `text-error` |
| Responsive | `sm:px-6`, `md:flex`, `lg:grid-cols-3` |

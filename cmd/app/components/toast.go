package components

import (
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// toastDuration is the auto-dismiss duration for toast notifications.
const toastDuration = 3 * time.Second

// toastMsg represents a single toast message.
type toastMsg struct {
	id      int64
	message string
	typ     string // success, error, warning, info
}

// Toast is the global toast notification component.
// It uses go-app's Action mechanism for cross-component communication:
//   - Register action: ctx.Handle("show-toast", handler)
//   - Trigger toast:   ctx.NewAction("show-toast", app.T("message", "..."), app.T("type", "success"))
type Toast struct {
	app.Compo

	messages []toastMsg
	nextID   int64
}

// ShowToast is a convenience function to trigger a global toast notification
// from any component. typ can be "success", "error", "warning", or "info".
func ShowToast(ctx app.Context, message, typ string) {
	ctx.NewAction("show-toast", app.T("message", message), app.T("type", typ))
}

// OnMount registers the "show-toast" action handler when the component mounts.
func (t *Toast) OnMount(ctx app.Context) {
	ctx.Handle("show-toast", t.handleShowToast)
}

// handleShowToast processes the "show-toast" action.
func (t *Toast) handleShowToast(ctx app.Context, a app.Action) {
	msg := a.Tags.Get("message")
	typ := a.Tags.Get("type")
	if msg == "" {
		return
	}
	if typ == "" {
		typ = "info"
	}

	t.nextID++
	id := t.nextID

	// Use Dispatch to update UI state
	ctx.Dispatch(func(ctx app.Context) {
		t.messages = append(t.messages, toastMsg{
			id:      id,
			message: msg,
			typ:     typ,
		})
	})

	// Auto-remove the message after the configured duration
	ctx.Async(func() {
		time.Sleep(toastDuration)
		ctx.Dispatch(func(ctx app.Context) {
			t.removeMessage(id)
		})
	})
}

// removeMessage removes the message with the given ID.
func (t *Toast) removeMessage(id int64) {
	filtered := make([]toastMsg, 0, len(t.messages))
	for _, m := range t.messages {
		if m.id != id {
			filtered = append(filtered, m)
		}
	}
	t.messages = filtered
}

// Render renders the toast container.
func (t *Toast) Render() app.UI {
	if len(t.messages) == 0 {
		return app.Div()
	}

	items := make([]app.UI, 0, len(t.messages))
	for _, m := range t.messages {
		items = append(items, t.renderItem(m))
	}

	return app.Div().Class("toast toast-top toast-end z-[9999]").Body(
		items...,
	)
}

// renderItem renders a single toast message.
func (t *Toast) renderItem(m toastMsg) app.UI {
	alertClass := "alert"
	switch m.typ {
	case "success":
		alertClass += " alert-success"
	case "error":
		alertClass += " alert-error"
	case "warning":
		alertClass += " alert-warning"
	default:
		alertClass += " alert-info"
	}

	return app.Div().Class(alertClass).Body(
		app.Span().Text(m.message),
	)
}

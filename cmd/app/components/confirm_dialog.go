package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

// ConfirmDialog is a reusable confirmation dialog component.
// Usage: Create with title/message, show on user action, handle confirm/cancel callbacks.
type ConfirmDialog struct {
	app.Compo

	// Dialog state
	Show    bool
	Title   string
	Message string

	// Button labels (optional, defaults provided)
	ConfirmLabel string
	CancelLabel  string

	// Styling
	ConfirmClass string // e.g., "btn-error", "btn-warning"

	// Callbacks
	OnConfirm func()
	OnCancel  func()
}

// Render renders the confirmation dialog.
func (d *ConfirmDialog) Render() app.UI {
	if !d.Show {
		return app.Div()
	}

	confirmLabel := d.ConfirmLabel
	if confirmLabel == "" {
		confirmLabel = "Confirm"
	}

	cancelLabel := d.CancelLabel
	if cancelLabel == "" {
		cancelLabel = "Cancel"
	}

	confirmClass := d.ConfirmClass
	if confirmClass == "" {
		confirmClass = "btn-primary"
	}

	return app.Div().Class("modal modal-open").Body(
		app.Div().Class("modal-box max-w-md").Body(
			// Icon
			app.Div().Class("flex items-center gap-3 mb-4").Body(
				app.Div().Class("flex-shrink-0 w-10 h-10 rounded-full bg-error/10 flex items-center justify-center").Body(
					app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-error" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>`),
				),
				app.H3().Class("font-bold text-lg").Text(d.Title),
			),

			// Message
			app.P().Class("text-base-content/70 mb-6").Text(d.Message),

			// Action buttons
			app.Div().Class("modal-action").Body(
				app.Button().
					Class("btn btn-ghost").
					OnClick(d.handleCancel).
					Text(cancelLabel),
				app.Button().
					Class("btn "+confirmClass+" gap-2").
					OnClick(d.handleConfirm).
					Body(
						app.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>`),
						app.Text(confirmLabel),
					),
			),
		),
		// Click backdrop to cancel
		app.Div().Class("modal-backdrop bg-black/40").OnClick(d.handleCancel),
	)
}

// handleConfirm handles the confirm button click.
func (d *ConfirmDialog) handleConfirm(ctx app.Context, e app.Event) {
	if d.OnConfirm != nil {
		d.OnConfirm()
	}
}

// handleCancel handles the cancel button or backdrop click.
func (d *ConfirmDialog) handleCancel(ctx app.Context, e app.Event) {
	if d.OnCancel != nil {
		d.OnCancel()
	}
}

// Open opens the dialog.
func (d *ConfirmDialog) Open() {
	d.Show = true
}

// Close closes the dialog.
func (d *ConfirmDialog) Close() {
	d.Show = false
}

// NewConfirmDialog creates a new ConfirmDialog with common defaults for delete operations.
func NewConfirmDialog(title, message string, onConfirm func()) *ConfirmDialog {
	return &ConfirmDialog{
		Title:        title,
		Message:      message,
		ConfirmLabel: "Delete",
		CancelLabel:  "Cancel",
		ConfirmClass: "btn-error",
		OnConfirm:    onConfirm,
	}
}

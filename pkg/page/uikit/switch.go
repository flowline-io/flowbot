package uikit

import "github.com/maxence-charriere/go-app/v10/pkg/app"

const (
	SwitchClass      = "uk-switch"
	SwitchTrackClass = "uk-switch-track"
	SwitchThumbClass = "uk-switch-thumb"
)

// Switch creates a basic switch component
func Switch(id string, name string, checked bool) app.HTMLLabel {
	input := app.Input().
		ID(id).
		Name(name).
		Type("checkbox").
		Class(SwitchClass)

	if checked {
		input = input.Checked(true)
	}

	return app.Label().Class("uk-switch-container").Body(
		input,
		app.Span().Class(SwitchTrackClass),
		app.Span().Class(SwitchThumbClass),
	)
}

// SwitchWithLabel creates a switch component with a label
func SwitchWithLabel(id string, name string, label string, checked bool) app.HTMLDiv {
	return app.Div().Class("uk-margin uk-grid-small uk-child-width-auto uk-grid").Body(
		app.Label().Body(
			Switch(id, name, checked),
			app.Span().Class("uk-margin-small-left").Text(label),
		),
	)
}

// SwitchWithFormLabel creates a switch component with a form label
func SwitchWithFormLabel(id string, name string, label string, checked bool) app.HTMLDiv {
	return app.Div().Class("uk-margin").Body(
		app.Label().Class("uk-form-label").For(id).Text(label),
		app.Div().Class("uk-form-controls").Body(
			Switch(id, name, checked),
		),
	)
}

// SwitchGroup creates a group of switches
func SwitchGroup(label string, switches []struct {
	ID      string
	Name    string
	Label   string
	Checked bool
}) app.HTMLDiv {
	var items []app.UI

	for _, s := range switches {
		items = append(items, SwitchWithLabel(s.ID, s.Name, s.Label, s.Checked))
	}

	return app.Div().Class("uk-margin").Body(
		app.Label().Class("uk-form-label").Text(label),
		app.Div().Class("uk-form-controls").Body(items...),
	)
}

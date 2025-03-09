package uikit

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

const (
	DatepickerClass = "uk-datepicker"
)

// Datepicker creates a basic date picker
func Datepicker(id string, name string, placeholder string) app.HTMLDiv {
	return app.Div().Class("uk-inline").Body(
		app.Span().Class("uk-form-icon").Attr("uk-icon", "icon: calendar"),
		app.Input().
			ID(id).
			Name(name).
			Class("uk-input").
			Type("text").
			Placeholder(placeholder).
			Attr("uk-datepicker", ""),
	)
}

// DatepickerWithOptions creates a date picker with options
func DatepickerWithOptions(id string, name string, placeholder string, format string, minDate string, maxDate string) app.HTMLDiv {
	options := ""

	if format != "" {
		options += fmt.Sprintf("format: '%s'; ", format)
	}

	if minDate != "" {
		options += fmt.Sprintf("minDate: '%s'; ", minDate)
	}

	if maxDate != "" {
		options += fmt.Sprintf("maxDate: '%s'; ", maxDate)
	}

	return app.Div().Class("uk-inline").Body(
		app.Span().Class("uk-form-icon").Attr("uk-icon", "icon: calendar"),
		app.Input().
			ID(id).
			Name(name).
			Class("uk-input").
			Type("text").
			Placeholder(placeholder).
			Attr("uk-datepicker", options),
	)
}

// DatepickerRange creates a date range picker
func DatepickerRange(startID string, startName string, startPlaceholder string, endID string, endName string, endPlaceholder string) app.HTMLDiv {
	return app.Div().Class("uk-grid-small").Attr("uk-grid", "").Body(
		app.Div().Class("uk-width-1-2").Body(
			app.Div().Class("uk-inline").Body(
				app.Span().Class("uk-form-icon").Attr("uk-icon", "icon: calendar"),
				app.Input().
					ID(startID).
					Name(startName).
					Class("uk-input").
					Type("text").
					Placeholder(startPlaceholder).
					Attr("uk-datepicker", ""),
			),
		),
		app.Div().Class("uk-width-1-2").Body(
			app.Div().Class("uk-inline").Body(
				app.Span().Class("uk-form-icon").Attr("uk-icon", "icon: calendar"),
				app.Input().
					ID(endID).
					Name(endName).
					Class("uk-input").
					Type("text").
					Placeholder(endPlaceholder).
					Attr("uk-datepicker", ""),
			),
		),
	)
}

// DatepickerWithLabel creates a date picker with a label
func DatepickerWithLabel(id string, name string, label string, placeholder string) app.HTMLDiv {
	return app.Div().Class("uk-margin").Body(
		app.Label().Class("uk-form-label").For(id).Text(label),
		app.Div().Class("uk-form-controls").Body(
			Datepicker(id, name, placeholder),
		),
	)
}

// DatepickerRangeWithLabel creates a date range picker with a label
func DatepickerRangeWithLabel(startID string, startName string, endID string, endName string, label string, startPlaceholder string, endPlaceholder string) app.HTMLDiv {
	return app.Div().Class("uk-margin").Body(
		app.Label().Class("uk-form-label").Text(label),
		app.Div().Class("uk-form-controls").Body(
			DatepickerRange(startID, startName, startPlaceholder, endID, endName, endPlaceholder),
		),
	)
}

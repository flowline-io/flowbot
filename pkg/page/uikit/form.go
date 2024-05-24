package uikit

import "github.com/maxence-charriere/go-app/v9/pkg/app"

const (
	FormStackedClass    = "uk-form-stacked"
	FormHorizontalClass = "uk-form-horizontal"
)

func Form(elems ...app.UI) app.HTMLForm {
	return app.Form().Body(elems...)
}

func FormLabel(text string, forId string) app.HTMLLabel {
	return app.Label().Class("uk-form-label").For(forId).Text(text)
}

func FormControls(elems ...app.UI) app.HTMLDiv {
	return app.Div().Class("uk-form-controls").Body(elems...)
}

func FormCustom(elems ...app.UI) app.HTMLDiv {
	return app.Div().Attr("uk-form-custom", "").Body(elems...)
}

func Fieldset(elems ...app.UI) app.HTMLFieldSet {
	return app.FieldSet().Class("uk-fieldset").Body(elems...)
}

func Input() app.HTMLInput {
	return app.Input().Class("uk-input")
}

func Select(elems ...app.UI) app.HTMLSelect {
	return app.Select().Class("uk-select").Body(elems...)
}

func Option(text string) app.HTMLOption {
	return app.Option().Text(text)
}

func Textarea(elems ...app.UI) app.HTMLTextarea {
	return app.Textarea().Class("uk-textarea").Body(elems...)
}

func Radio() app.HTMLInput {
	return app.Input().Class("uk-radio").Type("radio")
}

func Checkbox() app.HTMLInput {
	return app.Input().Class("uk-checkbox").Type("checkbox")
}

func Range() app.HTMLInput {
	return app.Input().Class("uk-range").Type("range")
}

func InputHidden(key, value string) app.HTMLInput {
	return app.Input().Hidden(true).Type("text").Name(key).Value(value)
}

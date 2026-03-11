package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

type FormField struct {
	app.Compo

	Label       string
	Type        string
	Value       string
	Placeholder string
	Error       string
	Required    bool
	OnChange    func(value string)
}

func (f *FormField) Render() app.UI {
	inputClass := "input input-bordered w-full"
	if f.Error != "" {
		inputClass += " input-error"
	}

	return app.Div().Class("form-control mb-4").Body(
		app.Label().Class("label").Body(
			app.Span().Class("label-text font-medium").Text(f.Label),
			app.If(f.Required, func() app.UI {
				return app.Span().Class("text-error ml-1").Text("*")
			}),
		),
		app.Input().
			Type(f.Type).
			Class(inputClass).
			Value(f.Value).
			Placeholder(f.Placeholder).
			OnChange(func(ctx app.Context, e app.Event) {
				if f.OnChange != nil {
					f.OnChange(ctx.JSSrc().Get("value").String())
				}
			}),
		app.If(f.Error != "", func() app.UI {
			return app.Label().Class("label").Body(
				app.Span().Class("label-text-alt text-error text-xs").Text(f.Error),
			)
		}),
	)
}

type FormValidator struct {
	errors map[string]string
}

func NewFormValidator() *FormValidator {
	return &FormValidator{
		errors: make(map[string]string),
	}
}

func (v *FormValidator) ValidateRequired(field, value, message string) bool {
	if value == "" {
		if message == "" {
			message = field + " is required"
		}
		v.errors[field] = message
		return false
	}
	delete(v.errors, field)
	return true
}

func (v *FormValidator) ValidateMinLength(field, value string, minLen int, message string) bool {
	if len(value) < minLen {
		if message == "" {
			message = field + " must be at least " + string(rune('0'+minLen)) + " characters"
		}
		v.errors[field] = message
		return false
	}
	delete(v.errors, field)
	return true
}

func (v *FormValidator) ValidateMaxLength(field, value string, maxLen int, message string) bool {
	if len(value) > maxLen {
		if message == "" {
			message = field + " must be at most " + string(rune('0'+maxLen)) + " characters"
		}
		v.errors[field] = message
		return false
	}
	delete(v.errors, field)
	return true
}

func (v *FormValidator) GetError(field string) string {
	return v.errors[field]
}

func (v *FormValidator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *FormValidator) Clear() {
	v.errors = make(map[string]string)
}

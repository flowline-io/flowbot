package form

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/sysatom/flowbot/internal/page/uikit"
	"github.com/sysatom/flowbot/internal/types"
	"strings"
)

type Builder struct {
	Field  []types.FormField
	Button []app.UI
	Method string
	Action string
	Data   types.KV
}

func NewBuilder(field []types.FormField) *Builder {
	return &Builder{Field: field}
}

func (b Builder) Save() error {
	return nil
}

func (b Builder) UI() (app.UI, error) {
	var elems []app.UI

	// default field
	//fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-csrf-token").Value(types.Id()))
	//fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-form_id").Value(c.Page.PageID))
	//fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-uid").Value(c.Page.UID))
	//fields = append(fields, app.Input().Hidden(true).Type("text").Name("x-topic").Value(c.Page.Topic))

	// Fields
	for _, field := range b.Field {
		field.Value = fixInt64Value(field.ValueType, field.Value)
		switch field.Type {
		case types.FormFieldHidden:
			field.Value = fixInt64Value(field.ValueType, field.Value)
			elems = append(elems, uikit.Input().Hidden(true).Type("text").Name(field.Key).Value(field.Value))
		case types.FormFieldText, types.FormFieldPassword, types.FormFieldNumber, types.FormFieldColor,
			types.FormFieldFile, types.FormFieldMonth, types.FormFieldDate, types.FormFieldTime, types.FormFieldEmail,
			types.FormFieldUrl, types.FormFieldRange:
			// input
			elems = append(elems, uikit.Margin(
				uikit.FormLabel(field.Label, field.Key),
				uikit.FormControls(
					uikit.Input().
						Type(string(field.Type)).
						Name(field.Key).
						Placeholder(field.Placeholder).
						Value(field.Value),
				),
			))
		case types.FormFieldRadio, types.FormFieldCheckbox:
			var options []app.UI
			for _, option := range field.Option {
				options = append(options, app.Label().Body(
					uikit.Input().Class(fmt.Sprintf("uk-%s", field.Type)).
						Type(string(field.Type)).
						Name(field.Key).
						Checked(option == field.Value).
						Value(option),
					uikit.Text(option)),
				)
			}
			elems = append(elems, uikit.Margin(
				uikit.FormLabel(field.Label, field.Key),
				uikit.FormControls(options...),
			))
		case types.FormFieldTextarea:
			// textarea
			elems = append(elems, uikit.Margin(
				uikit.FormLabel(field.Label, field.Key),
				uikit.FormControls(
					uikit.Textarea().
						Name(field.Key).
						Placeholder(field.Placeholder).
						Text(field.Value),
				),
			))
		case types.FormFieldSelect:
			// select
			var options []app.UI
			for _, option := range field.Option {
				options = append(options, uikit.Option(option).Selected(option == field.Value).Value(option))
			}
			elems = append(elems, uikit.Margin(
				uikit.FormLabel(field.Label, field.Key),
				uikit.FormControls(
					uikit.Select().
						Name(field.Key).Body(options...),
				),
			))
		}
	}
	// button
	elems = append(elems, b.Button...)

	return uikit.Form().Method(b.Method).Action(b.Action).
		Body(elems...), nil
}

func (b Builder) Validate() error {
	rules := make(map[string]interface{}, len(b.Field))
	for _, field := range b.Field {
		if field.Rule != "" {
			rules[field.Key] = field.Rule
		}
	}

	validate := validator.New()
	errs := validate.ValidateMap(b.Data, rules)
	for key, val := range errs {
		if err, ok := val.(error); ok {
			errStr := strings.ReplaceAll(err.Error(), "''", key)
			return errors.New(errStr)
		}
	}

	return nil
}

func fixInt64Value(t types.FormFieldValueType, v interface{}) interface{} {
	if t == types.FormFieldValueInt64 {
		switch v := v.(type) {
		case float64:
			return int64(v)
		}
	}
	return v
}

package dev

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
)

const (
	devFormID = "dev_form"
)

var formRules = []form.Rule{
	{
		Id:    devFormID,
		Title: "Current Value: 1, add/reduce ?",
		Field: []types.FormField{
			{
				Key:         "text",
				Type:        types.FormFieldText,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Text",
				Placeholder: "Input text",
				Rule:        "required",
			},
			{
				Key:         "password",
				Type:        types.FormFieldPassword,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Password",
				Placeholder: "Input password",
				Rule:        "required",
			},
			{
				Key:         "number",
				Type:        types.FormFieldNumber,
				ValueType:   types.FormFieldValueInt64,
				Value:       "",
				Label:       "Number",
				Placeholder: "Input number",
				Rule:        "gte=0,lte=130",
			},
			{
				Key:         "bool",
				Type:        types.FormFieldRadio,
				ValueType:   types.FormFieldValueBool,
				Value:       "",
				Label:       "Bool",
				Placeholder: "Switch",
				Option:      []string{"true", "false"},
			},
			{
				Key:         "multi",
				Type:        types.FormFieldCheckbox,
				ValueType:   types.FormFieldValueStringSlice,
				Value:       "",
				Label:       "Multiple",
				Placeholder: "Select multiple",
				Option:      []string{"a", "b", "c"},
			},
			{
				Key:         "textarea",
				Type:        types.FormFieldTextarea,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Textarea",
				Placeholder: "Input textarea",
				Rule:        "required",
			},
			{
				Key:         "select",
				Type:        types.FormFieldSelect,
				ValueType:   types.FormFieldValueFloat64,
				Value:       "",
				Label:       "Select",
				Placeholder: "Select float",
				Option:      []string{"1.01", "2.02", "3.03"},
			},
			{
				Key:         "range",
				Type:        types.FormFieldRange,
				ValueType:   types.FormFieldValueInt64,
				Value:       "",
				Label:       "Range",
				Placeholder: "range value",
			},
		},
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
}

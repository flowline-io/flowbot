package notify

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
)

const (
	createNotifyFormID = "create_notify"
)

var formRules = []form.Rule{
	{
		Id:         createNotifyFormID,
		Title:      "Create one task",
		IsLongTerm: true,
		Field: []types.FormField{
			{
				Key:         "name",
				Type:        types.FormFieldText,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Name",
				Placeholder: "Input notify name",
				Rule:        "required",
			},
			{
				Key:         "template",
				Type:        types.FormFieldTextarea,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Template",
				Placeholder: "Input notify template",
				Rule:        "required",
			},
		},
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			inputName, _ := values.String("name")
			inputTemplate, _ := values.String("template")

			err := store.Database.ConfigSet(ctx.AsUser, "", fmt.Sprintf("notify:%s", inputName), types.KV{
				"value": inputTemplate,
			})
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}

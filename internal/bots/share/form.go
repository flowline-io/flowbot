package share

import (
	"fmt"
	"github.com/sysatom/flowbot/internal/ruleset/form"
	"github.com/sysatom/flowbot/internal/types"
)

const (
	inputFormID = "input_form"
)

var formRules = []form.Rule{
	{
		Id:         inputFormID,
		IsLongTerm: true,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			return types.TextMsg{Text: fmt.Sprintf("%s", values["content"])}
		},
	},
}

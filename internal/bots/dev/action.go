package dev

import (
	"fmt"
	"github.com/sysatom/flowbot/internal/ruleset/action"
	"github.com/sysatom/flowbot/internal/types"
)

const (
	devActionID = "dev_action"
)

var actionRules = []action.Rule{
	{
		Id:     devActionID,
		Title:  "Operate ... ?",
		Option: []string{"do1", "do2"},
		Handler: map[string]func(ctx types.Context) types.MsgPayload{
			"do1": func(ctx types.Context) types.MsgPayload {
				return types.TextMsg{Text: fmt.Sprintf("do 1 something, action [%s: %d]", ctx.ActionRuleId, ctx.SeqId)}
			},
			"do2": func(ctx types.Context) types.MsgPayload {
				return types.TextMsg{Text: fmt.Sprintf("do 2 something, action [%s: %d]", ctx.ActionRuleId, ctx.SeqId)}
			},
		},
	},
}

package dev

import (
	"github.com/flowline-io/flowbot/internal/ruleset/event"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/template"
)

var eventRules = []event.Rule{
	{
		Event: types.GroupEventJoin,
		Handler: func(ctx types.Context, head types.KV, content interface{}) types.MsgPayload {
			txt, err := template.Parse(ctx, "Welcome $username")
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error user"}
			}
			return types.TextMsg{Text: txt}
		},
	},
	{
		Event: types.GroupEventExit,
		Handler: func(ctx types.Context, head types.KV, content interface{}) types.MsgPayload {
			txt, err := template.Parse(ctx, "Bye $username")
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "error user"}
			}
			return types.TextMsg{Text: txt}
		},
	},
	{
		Event: types.GroupEventReceive,
		Handler: func(ctx types.Context, head types.KV, content interface{}) types.MsgPayload {
			return types.TextMsg{Text: "receive something"}
		},
	},
}

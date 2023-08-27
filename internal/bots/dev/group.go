package dev

import (
	"github.com/sysatom/flowbot/internal/ruleset/event"
	"github.com/sysatom/flowbot/internal/types"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/template"
)

var eventRules = []event.Rule{
	{
		Event: types.GroupEventJoin,
		Handler: func(ctx types.Context, head types.KV, content interface{}) types.MsgPayload {
			txt, err := template.Parse(ctx, "Welcome $username")
			if err != nil {
				logs.Err.Println(err)
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
				logs.Err.Println(err)
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

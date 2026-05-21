// Package example implements the example module demonstrating all module entry points.
package example

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{Define: "id", Help: `Generate random id`, Handler: func(_ types.Context, _ []*parser.Token) types.MsgPayload {
		return types.TextMsg{Text: types.Id()}
	}},
	{Define: "form test", Help: `[example] form`, Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
		return module.FormMsg(ctx, exampleFormID)
	}},
	{Define: "page test", Help: `[example] example page`, Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
		url, err := module.PageURL(ctx, examplePageId, nil, 24*time.Hour)
		if err != nil {
			return types.TextMsg{Text: "error"}
		}
		return types.LinkMsg{Url: url}
	}},
	{Define: "event test", Help: `[example] event example`, Handler: func(ctx types.Context, _ []*parser.Token) types.MsgPayload {
		err := event.BotEventFire(ctx, types.ExampleBotEventID, types.KV{"k1": "v1"})
		if err != nil {
			return types.TextMsg{Text: err.Error()}
		}
		return types.TextMsg{Text: "ok"}
	}},
}

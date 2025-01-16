package kanban

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "kanban",
		Help:   `Example kanban command`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			err := event.BotEventFire(ctx.Context(), types.TaskCreateBotEventID, types.BotEvent{
				Uid:   ctx.AsUser.String(),
				Topic: ctx.Topic,
				Param: types.KV{
					"title":      "Test task",
					"project_id": 1,
					"reference":  fmt.Sprintf("%s:%s", "bot", types.Id()),
				},
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.EmptyMsg{}
		},
	},
}

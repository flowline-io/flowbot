package kanban

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: "kanban",
		Help:   `Example kanban command`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(kanboard.ID, kanboard.EndpointKey)
			username, _ := providers.GetConfig(kanboard.ID, kanboard.UsernameKey)
			password, _ := providers.GetConfig(kanboard.ID, kanboard.PasswordKey)
			client, err := kanboard.NewKanboard(endpoint.String(), username.String(), password.String())
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			taskId, err := client.CreateTask(ctx.Context(), &kanboard.Task{
				Title:     "Flowbot task test",
				ProjectID: 1,
				Reference: types.Id(),
			})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("%s/task/%d", config.App.Search.UrlBaseMap[kanboard.ID], taskId)}
		},
	},
}

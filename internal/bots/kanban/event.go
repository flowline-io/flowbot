package kanban

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.TaskCreateBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			endpoint, _ := providers.GetConfig(kanboard.ID, kanboard.EndpointKey)
			username, _ := providers.GetConfig(kanboard.ID, kanboard.UsernameKey)
			password, _ := providers.GetConfig(kanboard.ID, kanboard.PasswordKey)
			client, err := kanboard.NewKanboard(endpoint.String(), username.String(), password.String())
			if err != nil {
				return fmt.Errorf("failed to new client %w", err)
			}

			title, _ := param.String("title")
			projectId, _ := param.Int64("project_id")
			priority, _ := param.Int64("priority")
			reference, _ := param.String("reference")
			description, _ := param.String("description")
			tags, _ := param.List("tags")

			taskId, err := client.CreateTask(ctx.Context(), &kanboard.Task{
				Title:       title,
				ProjectID:   int(projectId),
				Priority:    int(priority),
				Reference:   reference,
				Description: description,
				Tags:        tags,
			})
			if err != nil {
				return fmt.Errorf("failed to create task %w", err)
			}

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("%s/task/%d", config.App.Search.UrlBaseMap[kanboard.ID], taskId),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
}

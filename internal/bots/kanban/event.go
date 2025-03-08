package kanban

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/config"
	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.TaskCreateBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			client, err := kanboard.GetClient()
			if err != nil {
				return fmt.Errorf("failed to new client %w", err)
			}

			title, _ := param.String("title")
			projectId, _ := param.Int64("project_id")
			priority, _ := param.Int64("priority")
			reference, _ := param.String("reference")
			description, _ := param.String("description")
			tags, _ := param.List("tags")

			if title == "" {
				return fmt.Errorf("title is empty")
			}
			if projectId == 0 {
				return fmt.Errorf("project_id is empty")
			}

			task := &kanboard.Task{
				Title:       title,
				ProjectID:   int(projectId),
				Priority:    int(priority),
				Reference:   reference,
				Description: description,
				Tags:        tags,
			}
			taskId, err := client.CreateTask(ctx.Context(), task)
			if err != nil {
				return fmt.Errorf("failed to create task %#v, error %w", task, err)
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

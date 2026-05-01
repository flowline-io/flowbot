package kanban

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgEvent "github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.TaskCreateBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			title, _ := param.String("title")
			projectID, _ := param.Int64("project_id")
			priority, _ := param.Int64("priority")
			reference, _ := param.String("reference")
			description, _ := param.String("description")
			tags, _ := param.List("tags")

			if title == "" {
				return fmt.Errorf("title is empty")
			}
			if projectID == 0 {
				return fmt.Errorf("project_id is empty")
			}

			res, err := ability.Invoke(ctx.Context(), hub.CapKanban, "create_task", map[string]any{
				"title":       title,
				"project_id":  int(projectID),
				"description": description,
				"reference":   reference,
				"tags":        tags,
			})
			if err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			_ = priority

			err = pkgEvent.SendMessage(ctx, types.TextMsg{
				Text: fmt.Sprintf("Task created: [%s](%s/task/%v)\n\n*Title:* %s\n*Project ID:* %d",
					title,
					config.App.Search.UrlBaseMap["kanboard"],
					res.Text,
					title,
					projectID,
				),
			})
			if err != nil {
				return fmt.Errorf("failed to send message %w", err)
			}

			return nil
		},
	},
}

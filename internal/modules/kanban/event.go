package kanban

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/notify"
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

			res, err := ability.Invoke(ctx.Context(), hub.CapKanban, ability.OpKanbanCreateTask, map[string]any{
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

			err = notify.GatewaySend(ctx.Context(), ctx.AsUser, "kanban.task.created", []string{"slack", "ntfy"}, map[string]any{
				"title":       title,
				"task_id":     res.Text,
				"project_id":  projectID,
				"description": description,
				"reference":   reference,
			})
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			return nil
		},
	},
}

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/urfave/cli/v3"
)

func KanbanCommand() *cli.Command {
	return &cli.Command{
		Name:        "kanban",
		Usage:       "Work with kanban boards",
		Description: "Manage kanban boards via Flowbot server",
		Commands: []*cli.Command{
			kanbanListCommand(),
			kanbanSearchCommand(),
			kanbanGetCommand(),
			kanbanCreateCommand(),
			kanbanUpdateCommand(),
			kanbanDeleteCommand(),
			kanbanMoveCommand(),
			kanbanCardCommand(),
			kanbanColumnCommand(),
			kanbanMetadataCommand(),
			kanbanTagCommand(),
			kanbanSubtaskCommand(),
		},
	}
}

func kanbanListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all kanban tasks",
		Description: "Display kanban tasks from Flowbot server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID",
				Value:   1,
			},
			&cli.StringFlag{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Status filter (active, inactive, all)",
				Value:   "active",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId := int(cmd.Int("project"))
			status := cmd.String("status")

			var tasks []kanboard.Task
			if status == "all" {
				tasks, err = c.Kanban.ListAll(ctx, projectId)
			} else {
				statusId := kanboard.Active
				if status == "inactive" {
					statusId = kanboard.Inactive
				}
				tasks, err = c.Kanban.List(ctx, projectId, statusId)
			}
			if err != nil {
				return fmt.Errorf("list kanban tasks: %w", err)
			}

			if len(tasks) == 0 {
				_, _ = fmt.Println("No kanban tasks found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(tasks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban tasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "COLUMN", "STATUS")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for _, t := range tasks {
					id := strconv.Itoa(t.ID)
					title := t.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					column := t.ColumnTitle
					if len(column) > 13 {
						column = column[:10] + "..."
					}
					statusStr := "active"
					if t.IsActive == 0 {
						statusStr = "closed"
					}
					_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", id, title, column, statusStr)
				}
			}

			return nil
		},
	}
}

func kanbanGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get a kanban task by ID",
		ArgsUsage:   "<id>",
		Description: "Display details of a specific kanban task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			task, err := c.Kanban.Get(ctx, id)
			if err != nil {
				return fmt.Errorf("get kanban task: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(task, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban task: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %d\n", task.ID)
				_, _ = fmt.Printf("Title:       %s\n", task.Title)
				_, _ = fmt.Printf("Description: %s\n", task.Description)
				_, _ = fmt.Printf("Project:     %s\n", task.ProjectName)
				_, _ = fmt.Printf("Column:      %s\n", task.ColumnTitle)
				_, _ = fmt.Printf("Priority:    %d\n", task.Priority)
				_, _ = fmt.Printf("Status:      %s\n", map[int]string{0: "inactive", 1: "active"}[task.IsActive])
				_, _ = fmt.Printf("Created:     %s\n", formatTimestamp(task.DateCreation))
				_, _ = fmt.Printf("Updated:     %s\n", formatTimestamp(task.DateModification))
			}

			return nil
		},
	}
}

func kanbanCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new kanban task",
		Description: "Add a new task to the kanban board",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"t"},
				Usage:    "Task title",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Task description",
			},
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID",
				Value:   1,
			},
			&cli.IntFlag{
				Name:    "column",
				Aliases: []string{"c"},
				Usage:   "Column ID",
				Value:   0,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanCreateRequest{
				Title:       cmd.String("title"),
				Description: cmd.String("description"),
				ProjectID:   int(cmd.Int("project")),
				ColumnID:    int(cmd.Int("column")),
			}

			result, err := c.Kanban.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("create kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task created: ID=%d\n", result.ID)
			return nil
		},
	}
}

func kanbanUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Usage:       "Update a kanban task",
		ArgsUsage:   "<id>",
		Description: "Modify an existing kanban task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Usage:   "New title",
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "New description",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanUpdateRequest{}
			if title := cmd.String("title"); title != "" {
				req.Title = title
			}
			if desc := cmd.String("description"); desc != "" {
				req.Description = desc
			}

			_, err = c.Kanban.Update(ctx, id, req)
			if err != nil {
				return fmt.Errorf("update kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task updated: %d\n", id)
			return nil
		},
	}
}

func kanbanDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Close a kanban task",
		ArgsUsage:   "<id>",
		Description: "Close a task by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Close task %d? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			_, err = c.Kanban.Close(ctx, id)
			if err != nil {
				return fmt.Errorf("close kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task closed: %d\n", id)
			return nil
		},
	}
}

func kanbanMoveCommand() *cli.Command {
	return &cli.Command{
		Name:        "move",
		Usage:       "Move a kanban task to another column",
		ArgsUsage:   "<id>",
		Description: "Move a task to a different column",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "column",
				Aliases:  []string{"c"},
				Usage:    "Destination column ID",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "position",
				Aliases: []string{"p"},
				Usage:   "Position in column (0 = first)",
				Value:   0,
			},
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"r"},
				Usage:   "Project ID",
				Value:   1,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanMoveRequest{
				ColumnID:  int(cmd.Int("column")),
				Position:  int(cmd.Int("position")),
				ProjectID: int(cmd.Int("project")),
			}

			_, err = c.Kanban.Move(ctx, id, req)
			if err != nil {
				return fmt.Errorf("move kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task moved: %d -> column %d\n", id, req.ColumnID)
			return nil
		},
	}
}

func kanbanSearchCommand() *cli.Command {
	return &cli.Command{
		Name:        "search",
		Usage:       "Search kanban tasks",
		ArgsUsage:   "<query>",
		Description: "Search tasks using kanboard search syntax",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID",
				Value:   1,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("search query is required")
			}
			query := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId := int(cmd.Int("project"))
			tasks, err := c.Kanban.Search(ctx, projectId, query)
			if err != nil {
				return fmt.Errorf("search kanban tasks: %w", err)
			}

			if len(tasks) == 0 {
				_, _ = fmt.Println("No tasks found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(tasks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal tasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "COLUMN", "STATUS")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for _, t := range tasks {
					id := strconv.Itoa(t.ID)
					title := t.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					column := t.ColumnTitle
					if len(column) > 13 {
						column = column[:10] + "..."
					}
					statusStr := "active"
					if t.IsActive == 0 {
						statusStr = "closed"
					}
					_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", id, title, column, statusStr)
				}
			}

			return nil
		},
	}
}

func kanbanMetadataCommand() *cli.Command {
	return &cli.Command{
		Name:        "metadata",
		Usage:       "Manage task metadata",
		Description: "Get, set, or delete task metadata",
		Commands: []*cli.Command{
			kanbanMetadataGetCommand(),
			kanbanMetadataSetCommand(),
			kanbanMetadataDeleteCommand(),
		},
	}
}

func kanbanMetadataGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get task metadata",
		ArgsUsage:   "<task_id> [name]",
		Description: "Get all metadata or a specific metadata value by name",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (json, value)",
				Value:   "json",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			if cmd.NArg() > 1 {
				name := cmd.Args().Get(1)
				value, err := c.Kanban.GetMetadataByName(ctx, taskId, name)
				if err != nil {
					return fmt.Errorf("get metadata: %w", err)
				}
				_, _ = fmt.Println(value)
			} else {
				metadata, err := c.Kanban.GetMetadata(ctx, taskId)
				if err != nil {
					return fmt.Errorf("get metadata: %w", err)
				}
				data, err := json.MarshalIndent(metadata, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal metadata: %w", err)
				}
				_, _ = fmt.Println(string(data))
			}

			return nil
		},
	}
}

func kanbanMetadataSetCommand() *cli.Command {
	return &cli.Command{
		Name:        "set",
		Usage:       "Set task metadata",
		ArgsUsage:   "<task_id> <name=value>...",
		Description: "Set one or more metadata values for a task",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("task ID and at least one name=value pair are required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			values := make(kanboard.TaskMetadata)
			for i := 1; i < cmd.NArg(); i++ {
				arg := cmd.Args().Get(i)
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format for %s, expected name=value", arg)
				}
				values[parts[0]] = parts[1]
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Kanban.SaveMetadata(ctx, taskId, values)
			if err != nil {
				return fmt.Errorf("save metadata: %w", err)
			}

			if result.Success {
				_, _ = fmt.Println("Metadata saved successfully")
			} else {
				_, _ = fmt.Println("Failed to save metadata")
			}

			return nil
		},
	}
}

func kanbanMetadataDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete task metadata",
		ArgsUsage:   "<task_id> <name>",
		Description: "Delete a metadata entry from a task",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("task ID and metadata name are required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			name := cmd.Args().Get(1)

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Delete metadata '%s' from task %d? [y/N]: ", name, taskId)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Kanban.RemoveMetadata(ctx, taskId, name)
			if err != nil {
				return fmt.Errorf("remove metadata: %w", err)
			}

			if result.Success {
				_, _ = fmt.Printf("Metadata '%s' deleted from task %d\n", name, taskId)
			} else {
				_, _ = fmt.Println("Failed to delete metadata")
			}

			return nil
		},
	}
}

func formatTimestamp(ts int) string {
	if ts == 0 {
		return "N/A"
	}
	t := time.Unix(int64(ts), 0)
	return t.Format(time.RFC3339)
}

func kanbanSubtaskCommand() *cli.Command {
	return &cli.Command{
		Name:        "subtask",
		Usage:       "Manage kanban subtasks",
		Description: "Create, update, list and delete subtasks for kanban tasks",
		Commands: []*cli.Command{
			kanbanSubtaskListCommand(),
			kanbanSubtaskGetCommand(),
			kanbanSubtaskCreateCommand(),
			kanbanSubtaskUpdateCommand(),
			kanbanSubtaskDeleteCommand(),
		},
	}
}

func kanbanSubtaskListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List subtasks for a task",
		ArgsUsage:   "<task_id>",
		Description: "Display all subtasks for a given task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			subtasks, err := c.Kanban.ListSubtasks(ctx, taskId)
			if err != nil {
				return fmt.Errorf("list subtasks: %w", err)
			}

			if len(subtasks) == 0 {
				_, _ = fmt.Println("No subtasks found for this task")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(subtasks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal subtasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-10s %-12s %-12s\n", "ID", "TITLE", "STATUS", "ESTIMATED", "SPENT")
				_, _ = fmt.Println(strings.Repeat("-", 80))
				for _, s := range subtasks {
					title := s.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					status := s.StatusName
					if status == "" {
						status = "Todo"
					}
					_, _ = fmt.Printf("%-8s %-30s %-10s %-12s %-12s\n", s.ID, title, status, s.TimeEstimated, s.TimeSpent)
				}
			}

			return nil
		},
	}
}

func kanbanSubtaskGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get a subtask by ID",
		ArgsUsage:   "<task_id> <subtask_id>",
		Description: "Display details of a specific subtask",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskIdStr := cmd.Args().Get(1)
			subtaskId, err := strconv.Atoi(subtaskIdStr)
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			subtask, err := c.Kanban.GetSubtask(ctx, taskId, subtaskId)
			if err != nil {
				return fmt.Errorf("get subtask: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(subtask, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal subtask: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %s\n", subtask.ID)
				_, _ = fmt.Printf("Title:       %s\n", subtask.Title)
				_, _ = fmt.Printf("Task ID:     %s\n", subtask.TaskID)
				_, _ = fmt.Printf("Status:      %s\n", subtask.StatusName)
				_, _ = fmt.Printf("Time Estimated: %v\n", subtask.TimeEstimated)
				_, _ = fmt.Printf("Time Spent:  %v\n", subtask.TimeSpent)
				if subtask.Username != "" {
					_, _ = fmt.Printf("Assignee:    %s\n", subtask.Username)
				}
			}

			return nil
		},
	}
}

func kanbanSubtaskCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new subtask",
		ArgsUsage:   "<task_id>",
		Description: "Add a subtask to a kanban task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"t"},
				Usage:    "Subtask title",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "user",
				Aliases: []string{"u"},
				Usage:   "User ID to assign",
				Value:   0,
			},
			&cli.IntFlag{
				Name:    "time-estimated",
				Aliases: []string{"e"},
				Usage:   "Estimated time (minutes)",
				Value:   0,
			},
			&cli.IntFlag{
				Name:    "time-spent",
				Aliases: []string{"s"},
				Usage:   "Time spent (minutes)",
				Value:   0,
			},
			&cli.IntFlag{
				Name:    "status",
				Aliases: []string{"S"},
				Usage:   "Status (0=Todo, 1=In progress, 2=Done)",
				Value:   0,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanCreateSubtaskRequest{
				Title:         cmd.String("title"),
				UserID:        int(cmd.Int("user")),
				TimeEstimated: int(cmd.Int("time-estimated")),
				TimeSpent:     int(cmd.Int("time-spent")),
				Status:        int(cmd.Int("status")),
			}

			result, err := c.Kanban.CreateSubtask(ctx, taskId, req)
			if err != nil {
				return fmt.Errorf("create subtask: %w", err)
			}

			_, _ = fmt.Printf("Subtask created: ID=%d\n", result.ID)
			return nil
		},
	}
}

func kanbanSubtaskUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Usage:       "Update a subtask",
		ArgsUsage:   "<task_id> <subtask_id>",
		Description: "Modify an existing subtask",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Usage:   "New title",
			},
			&cli.IntFlag{
				Name:    "user",
				Aliases: []string{"u"},
				Usage:   "User ID to assign (-1 to unassign)",
				Value:   -1,
			},
			&cli.IntFlag{
				Name:    "time-estimated",
				Aliases: []string{"e"},
				Usage:   "Estimated time (minutes, -1 to clear)",
				Value:   -1,
			},
			&cli.IntFlag{
				Name:    "time-spent",
				Aliases: []string{"s"},
				Usage:   "Time spent (minutes, -1 to clear)",
				Value:   -1,
			},
			&cli.IntFlag{
				Name:    "status",
				Aliases: []string{"S"},
				Usage:   "Status (0=Todo, 1=In progress, 2=Done, -1 to leave unchanged)",
				Value:   -1,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskIdStr := cmd.Args().Get(1)
			subtaskId, err := strconv.Atoi(subtaskIdStr)
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanUpdateSubtaskRequest{}
			if title := cmd.String("title"); title != "" {
				req.Title = title
			}
			if user := int(cmd.Int("user")); user >= 0 {
				req.UserID = user
			}
			if te := int(cmd.Int("time-estimated")); te >= 0 {
				req.TimeEstimated = te
			}
			if ts := int(cmd.Int("time-spent")); ts >= 0 {
				req.TimeSpent = ts
			}
			if st := int(cmd.Int("status")); st >= 0 {
				req.Status = st
			}

			result, err := c.Kanban.UpdateSubtask(ctx, taskId, subtaskId, req)
			if err != nil {
				return fmt.Errorf("update subtask: %w", err)
			}

			if result.Success {
				_, _ = fmt.Printf("Subtask updated: %d\n", subtaskId)
			} else {
				_, _ = fmt.Println("Failed to update subtask")
			}
			return nil
		},
	}
}

func kanbanSubtaskDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a subtask",
		ArgsUsage:   "<task_id> <subtask_id>",
		Description: "Remove a subtask by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskIdStr := cmd.Args().Get(1)
			subtaskId, err := strconv.Atoi(subtaskIdStr)
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Delete subtask %d? [y/N]: ", subtaskId)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Kanban.RemoveSubtask(ctx, taskId, subtaskId)
			if err != nil {
				return fmt.Errorf("delete subtask: %w", err)
			}

			if result.Success {
				_, _ = fmt.Printf("Subtask deleted: %d\n", subtaskId)
			} else {
				_, _ = fmt.Println("Failed to delete subtask")
			}
			return nil
		},
	}
}

func kanbanTagCommand() *cli.Command {
	return &cli.Command{
		Name:        "tag",
		Usage:       "Manage kanban tags",
		Description: "Create, update, list and manage kanban tags",
		Commands: []*cli.Command{
			kanbanTagListCommand(),
			kanbanTagCreateCommand(),
			kanbanTagUpdateCommand(),
			kanbanTagDeleteCommand(),
			kanbanTagTaskCommand(),
		},
	}
}

func kanbanTagListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all tags",
		Description: "Display kanban tags",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID (if specified, list tags for this project)",
				Value:   0,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId := int(cmd.Int("project"))
			var tags []client.KanbanTag
			if projectId > 0 {
				tags, err = c.Kanban.ListTagsByProject(ctx, projectId)
			} else {
				tags, err = c.Kanban.ListTags(ctx)
			}
			if err != nil {
				return fmt.Errorf("list tags: %w", err)
			}

			if len(tags) == 0 {
				_, _ = fmt.Println("No tags found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(tags, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal tags: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-10s\n", "ID", "NAME", "PROJECT")
				_, _ = fmt.Println(strings.Repeat("-", 50))
				for _, t := range tags {
					name := t.Name
					if len(name) > 28 {
						name = name[:25] + "..."
					}
					_, _ = fmt.Printf("%-8s %-30s %-10s\n", t.ID, name, t.ProjectID)
				}
			}

			return nil
		},
	}
}

func kanbanTagCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new tag",
		Description: "Add a new tag to the kanban board",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Tag name",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID",
				Value:   1,
			},
			&cli.StringFlag{
				Name:    "color",
				Aliases: []string{"c"},
				Usage:   "Color ID",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanCreateTagRequest{
				ProjectID: int(cmd.Int("project")),
				Name:      cmd.String("name"),
				ColorID:   cmd.String("color"),
			}

			result, err := c.Kanban.CreateTag(ctx, req)
			if err != nil {
				return fmt.Errorf("create tag: %w", err)
			}

			_, _ = fmt.Printf("Tag created: ID=%d\n", result.ID)
			return nil
		},
	}
}

func kanbanTagUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Usage:       "Update a tag",
		ArgsUsage:   "<id>",
		Description: "Modify an existing tag",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "New tag name",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "color",
				Aliases: []string{"c"},
				Usage:   "Color ID",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("tag ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid tag ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanUpdateTagRequest{
				Name:    cmd.String("name"),
				ColorID: cmd.String("color"),
			}

			_, err = c.Kanban.UpdateTag(ctx, id, req)
			if err != nil {
				return fmt.Errorf("update tag: %w", err)
			}

			_, _ = fmt.Printf("Tag updated: %d\n", id)
			return nil
		},
	}
}

func kanbanTagDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a tag",
		ArgsUsage:   "<id>",
		Description: "Remove a tag by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("tag ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return fmt.Errorf("invalid tag ID: %w", err)
			}

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Delete tag %d? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			_, err = c.Kanban.RemoveTag(ctx, id)
			if err != nil {
				return fmt.Errorf("delete tag: %w", err)
			}

			_, _ = fmt.Printf("Tag deleted: %d\n", id)
			return nil
		},
	}
}

func kanbanTagTaskCommand() *cli.Command {
	return &cli.Command{
		Name:        "task",
		Usage:       "Manage task tags",
		Description: "Get or set tags for a task",
		Commands: []*cli.Command{
			kanbanTagTaskGetCommand(),
			kanbanTagTaskSetCommand(),
		},
	}
}

func kanbanTagTaskGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get tags for a task",
		ArgsUsage:   "<task_id>",
		Description: "Display tags assigned to a task",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (json, table)",
				Value:   "table",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			tags, err := c.Kanban.GetTaskTags(ctx, taskId)
			if err != nil {
				return fmt.Errorf("get task tags: %w", err)
			}

			if len(tags) == 0 {
				_, _ = fmt.Println("No tags assigned to this task")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(tags, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal tags: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s\n", "ID", "NAME")
				_, _ = fmt.Println(strings.Repeat("-", 40))
				for id, name := range tags {
					_, _ = fmt.Printf("%-8s %-30s\n", id, name)
				}
			}

			return nil
		},
	}
}

func kanbanTagTaskSetCommand() *cli.Command {
	return &cli.Command{
		Name:        "set",
		Usage:       "Set tags for a task",
		ArgsUsage:   "<task_id>",
		Description: "Assign tags to a task",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "Project ID",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:     "tags",
				Aliases:  []string{"t"},
				Usage:    "Tag names (can be specified multiple times)",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskIdStr := cmd.Args().Get(0)
			taskId, err := strconv.Atoi(taskIdStr)
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanSetTaskTagsRequest{
				ProjectID: int(cmd.Int("project")),
				Tags:      cmd.StringSlice("tags"),
			}

			result, err := c.Kanban.SetTaskTags(ctx, taskId, req)
			if err != nil {
				return fmt.Errorf("set task tags: %w", err)
			}

			if result.Success {
				_, _ = fmt.Println("Task tags updated successfully")
			} else {
				_, _ = fmt.Println("Failed to update task tags")
			}

			return nil
		},
	}
}

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/store"
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
			kanbanGetCommand(),
			kanbanCreateCommand(),
			kanbanUpdateCommand(),
			kanbanDeleteCommand(),
			kanbanMoveCommand(),
			kanbanCardCommand(),
			kanbanColumnCommand(),
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
			c, err := newKanbanClient(cmd)
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

			c, err := newKanbanClient(cmd)
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
			c, err := newKanbanClient(cmd)
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

			c, err := newKanbanClient(cmd)
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

			c, err := newKanbanClient(cmd)
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

			c, err := newKanbanClient(cmd)
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

func newKanbanClient(cmd *cli.Command) (*client.Client, error) {
	profile := cmd.String("profile")

	serverURL := cmd.String("server-url")
	if serverURL == "" {
		return nil, fmt.Errorf("server URL is required (use --server-url or FLOWBOT_SERVER_URL)")
	}

	token, err := store.LoadToken(profile)
	if err != nil {
		return nil, fmt.Errorf("load token: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("not logged in (use 'flowbot login' first)")
	}

	return client.NewClient(serverURL, token), nil
}

func formatTimestamp(ts int) string {
	if ts == 0 {
		return "N/A"
	}
	t := time.Unix(int64(ts), 0)
	return t.Format(time.RFC3339)
}

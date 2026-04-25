package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/internal/store"
	"github.com/flowline-io/flowbot/cmd/cli/pkg/client"
	"github.com/urfave/cli/v3"
)

// KanbanCommand returns the kanban parent command
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

			statusId := 1
			if status == "inactive" {
				statusId = 0
			}

			var result []kanbanTask
			path := fmt.Sprintf("/service/kanban?project_id=%d&status_id=%d", projectId, statusId)
			if status == "all" {
				path = fmt.Sprintf("/service/kanban?project_id=%d", projectId)
			}

			if err := c.Get(path, &result); err != nil {
				return fmt.Errorf("list kanban tasks: %w", err)
			}

			if len(result) == 0 {
				_, _ = fmt.Println("No kanban tasks found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban tasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "COLUMN", "STATUS")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for _, t := range result {
					id := strconv.Itoa(t.ID)
					title := t.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					column := t.ColumnTitle
					if len(column) > 13 {
						column = column[:10] + "..."
					}
					status := "active"
					if t.IsActive == 0 {
						status = "closed"
					}
					_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", id, title, column, status)
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
			id := cmd.Args().Get(0)

			c, err := newKanbanClient(cmd)
			if err != nil {
				return err
			}

			var result kanbanTask
			if err := c.Get("/service/kanban/"+id, &result); err != nil {
				return fmt.Errorf("get kanban task: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban task: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %d\n", result.ID)
				_, _ = fmt.Printf("Title:       %s\n", result.Title)
				_, _ = fmt.Printf("Description: %s\n", result.Description)
				_, _ = fmt.Printf("Project:     %s\n", result.ProjectName)
				_, _ = fmt.Printf("Column:      %s\n", result.ColumnTitle)
				_, _ = fmt.Printf("Priority:    %d\n", result.Priority)
				_, _ = fmt.Printf("Status:      %s\n", map[int]string{0: "inactive", 1: "active"}[result.IsActive])
				_, _ = fmt.Printf("Created:     %s\n", formatTimestamp(result.DateCreation))
				_, _ = fmt.Printf("Updated:     %s\n", formatTimestamp(result.DateModification))
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

			body := map[string]any{
				"title":       cmd.String("title"),
				"description": cmd.String("description"),
				"project_id":  int(cmd.Int("project")),
			}
			if colId := int(cmd.Int("column")); colId > 0 {
				body["column_id"] = colId
			}

			var result kanbanCreateResult
			if err := c.Post("/service/kanban", body, &result); err != nil {
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
			id := cmd.Args().Get(0)

			c, err := newKanbanClient(cmd)
			if err != nil {
				return err
			}

			body := map[string]string{}
			if title := cmd.String("title"); title != "" {
				body["title"] = title
			}
			if desc := cmd.String("description"); desc != "" {
				body["description"] = desc
			}

			var result kanbanUpdateResult
			if err := c.Patch("/service/kanban/"+id, body, &result); err != nil {
				return fmt.Errorf("update kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task updated: %s\n", id)
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
			id := cmd.Args().Get(0)

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Close task %s? [y/N]: ", id)
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

			var result kanbanDeleteResult
			if err := c.Delete("/service/kanban/"+id, nil, &result); err != nil {
				return fmt.Errorf("close kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task closed: %s\n", id)
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
			id := cmd.Args().Get(0)

			c, err := newKanbanClient(cmd)
			if err != nil {
				return err
			}

			body := map[string]any{
				"column_id":  int(cmd.Int("column")),
				"position":   int(cmd.Int("position")),
				"project_id": int(cmd.Int("project")),
			}

			var result kanbanMoveResult
			if err := c.Post("/service/kanban/"+id+"/move", body, &result); err != nil {
				return fmt.Errorf("move kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task moved: %s -> column %d\n", id, int(cmd.Int("column")))
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

// Response types for kanban webservice responses

type kanbanTask struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	ProjectName      string `json:"project_name"`
	ColumnTitle      string `json:"column_title"`
	Priority         int    `json:"priority"`
	IsActive         int    `json:"is_active"`
	DateCreation     int    `json:"date_creation"`
	DateModification int    `json:"date_modification"`
}

type kanbanCreateResult struct {
	ID int64 `json:"id"`
}

type kanbanUpdateResult struct {
	Success bool `json:"success"`
}

type kanbanDeleteResult struct {
	Success bool `json:"success"`
}

type kanbanMoveResult struct {
	Success bool `json:"success"`
}

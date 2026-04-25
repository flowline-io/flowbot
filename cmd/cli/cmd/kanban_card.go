package cmd

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/cmd/cli/internal/store"
	"github.com/flowline-io/flowbot/cmd/cli/pkg/client"
	"github.com/urfave/cli/v3"
)

func kanbanCardCommand() *cli.Command {
	return &cli.Command{
		Name:        "card",
		Usage:       "Work with kanban cards (alias for task operations)",
		Description: "Manage cards within kanban boards via server API",
		Commands: []*cli.Command{
			kanbanCardAddCommand(),
			kanbanCardMoveCommand(),
			kanbanCardDeleteCommand(),
		},
	}
}

func kanbanCardAddCommand() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Usage:       "Add a card to a kanban board",
		Description: "Create a new task in the specified column",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"t"},
				Usage:    "Card title",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Card description",
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
			c, err := newKanbanCardClient(cmd)
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
				return fmt.Errorf("create card: %w", err)
			}

			_, _ = fmt.Printf("Card created: ID=%d\n", result.ID)
			return nil
		},
	}
}

func kanbanCardMoveCommand() *cli.Command {
	return &cli.Command{
		Name:        "move",
		Usage:       "Move a card to another column",
		Description: "Move a task to a different column",
		ArgsUsage:   "<card-id>",
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
				return fmt.Errorf("card ID is required")
			}
			cardID := cmd.Args().Get(0)

			c, err := newKanbanCardClient(cmd)
			if err != nil {
				return err
			}

			body := map[string]any{
				"column_id":  int(cmd.Int("column")),
				"position":   int(cmd.Int("position")),
				"project_id": int(cmd.Int("project")),
			}

			var result kanbanMoveResult
			if err := c.Post("/service/kanban/"+cardID+"/move", body, &result); err != nil {
				return fmt.Errorf("move card: %w", err)
			}

			_, _ = fmt.Printf("Card moved: %s -> column %d\n", cardID, int(cmd.Int("column")))
			return nil
		},
	}
}

func kanbanCardDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a card from a kanban board",
		Description: "Close a task by ID",
		ArgsUsage:   "<card-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("card ID is required")
			}
			cardID := cmd.Args().Get(0)

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Close card %s? [y/N]: ", cardID)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := newKanbanCardClient(cmd)
			if err != nil {
				return err
			}

			var result kanbanDeleteResult
			if err := c.Delete("/service/kanban/"+cardID, nil, &result); err != nil {
				return fmt.Errorf("close card: %w", err)
			}

			_, _ = fmt.Printf("Card closed: %s\n", cardID)
			return nil
		},
	}
}

func kanbanColumnCommand() *cli.Command {
	return &cli.Command{
		Name:        "column",
		Usage:       "Work with kanban columns",
		Description: "Manage columns within kanban boards",
		Commands: []*cli.Command{
			kanbanColumnListCommand(),
		},
	}
}

func kanbanColumnListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List columns in a project",
		Description: "Display all columns in the specified project",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project ID",
				Value:   1,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := newKanbanCardClient(cmd)
			if err != nil {
				return err
			}

			projectId := int(cmd.Int("project"))

			var result []kanbanColumn
			if err := c.Get(fmt.Sprintf("/service/kanban/columns?project_id=%d", projectId), &result); err != nil {
				return fmt.Errorf("list columns: %w", err)
			}

			if len(result) == 0 {
				_, _ = fmt.Println("No columns found")
				return nil
			}

			_, _ = fmt.Printf("%-8s %-20s %-8s\n", "ID", "TITLE", "POSITION")
			for _, col := range result {
				_, _ = fmt.Printf("%-8d %-20s %-8d\n", col.ID, col.Title, col.Position)
			}

			return nil
		},
	}
}

func newKanbanCardClient(cmd *cli.Command) (*client.Client, error) {
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

// Response types for kanban card commands

type kanbanColumn struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Position int    `json:"position"`
}

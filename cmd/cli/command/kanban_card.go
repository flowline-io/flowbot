package command

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/pkg/client"
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

			req := client.KanbanCreateRequest{
				Title:       cmd.String("title"),
				Description: cmd.String("description"),
				ProjectID:   int(cmd.Int("project")),
				ColumnID:    int(cmd.Int("column")),
			}

			result, err := c.Kanban.Create(ctx, req)
			if err != nil {
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
			id, err := strconv.Atoi(cardID)
			if err != nil {
				return fmt.Errorf("invalid card ID: %w", err)
			}

			c, err := newKanbanCardClient(cmd)
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
				return fmt.Errorf("move card: %w", err)
			}

			_, _ = fmt.Printf("Card moved: %d -> column %d\n", id, req.ColumnID)
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
			id, err := strconv.Atoi(cardID)
			if err != nil {
				return fmt.Errorf("invalid card ID: %w", err)
			}

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Close card %d? [y/N]: ", id)
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

			_, err = c.Kanban.Close(ctx, id)
			if err != nil {
				return fmt.Errorf("close card: %w", err)
			}

			_, _ = fmt.Printf("Card closed: %d\n", id)
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

			columns, err := c.Kanban.ListColumns(ctx, projectId)
			if err != nil {
				return fmt.Errorf("list columns: %w", err)
			}

			if len(columns) == 0 {
				_, _ = fmt.Println("No columns found")
				return nil
			}

			_, _ = fmt.Printf("%-8s %-20s\n", "ID", "TITLE")
			for _, col := range columns {
				_, _ = fmt.Printf("%-8d %-20s\n", col.ID, col.Title)
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

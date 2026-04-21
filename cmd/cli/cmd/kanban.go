package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/internal/model"
	"github.com/flowline-io/flowbot/cmd/cli/internal/store"
	"github.com/flowline-io/flowbot/cmd/cli/pkg/utils"
	"github.com/urfave/cli/v3"
)

// KanbanCommand returns the kanban parent command
func KanbanCommand() *cli.Command {
	return &cli.Command{
		Name:        "kanban",
		Usage:       "Work with kanban boards",
		Description: "Manage kanban boards locally",
		Commands: []*cli.Command{
			kanbanCreateCommand(),
			kanbanListCommand(),
			kanbanGetCommand(),
			kanbanUpdateCommand(),
			kanbanDeleteCommand(),
			kanbanCardCommand(),
			kanbanColumnCommand(),
		},
	}
}

func kanbanCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new kanban board",
		Description: "Add a new kanban board to local storage",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"t"},
				Usage:    "Board title",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Board description",
			},
			&cli.StringSliceFlag{
				Name:    "columns",
				Aliases: []string{"c"},
				Usage:   "Initial columns (default: Todo, In Progress, Done)",
				Value:   []string{"Todo", "In Progress", "Done"},
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			columns := make([]model.Column, 0)
			for i, name := range cmd.StringSlice("columns") {
				columns = append(columns, model.Column{
					ID:    utils.GenerateID(),
					Name:  name,
					Order: i,
					Cards: []model.Card{},
				})
			}

			kanban := model.Kanban{
				ID:          utils.GenerateID(),
				Title:       cmd.String("title"),
				Description: cmd.String("description"),
				Columns:     columns,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			s.Kanbans = append(s.Kanbans, kanban)

			if err := store.SaveKanbans(s, profile); err != nil {
				return err
			}

			_, _ = fmt.Printf("Kanban board created: %s (%s)\n", kanban.Title, kanban.ID)
			return nil
		},
	}
}

func kanbanListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all kanban boards",
		Description: "Display all saved kanban boards",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			if len(s.Kanbans) == 0 {
				_, _ = fmt.Println("No kanban boards found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(s.Kanbans, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanbans: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-12s %-30s %-8s\n", "ID", "TITLE", "COLUMNS")
				_, _ = fmt.Println(string(make([]byte, 52)))
				for _, k := range s.Kanbans {
					title := k.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					_, _ = fmt.Printf("%-12s %-30s %d\n", k.ID, title, len(k.Columns))
				}
			}

			return nil
		},
	}
}

func kanbanGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get a kanban board by ID",
		ArgsUsage:   "<id>",
		Description: "Display details of a specific kanban board",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			if cmd.NArg() == 0 {
				return fmt.Errorf("kanban ID is required")
			}
			id := cmd.Args().Get(0)

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			kanban := findKanbanByID(s.Kanbans, id)
			if kanban == nil {
				return fmt.Errorf("kanban board not found: %s", id)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(kanban, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %s\n", kanban.ID)
				_, _ = fmt.Printf("Title:       %s\n", kanban.Title)
				_, _ = fmt.Printf("Description: %s\n", kanban.Description)
				_, _ = fmt.Printf("Created:     %s\n", kanban.CreatedAt.Format(time.RFC3339))
				_, _ = fmt.Printf("Updated:     %s\n", kanban.UpdatedAt.Format(time.RFC3339))
				_, _ = fmt.Printf("\nColumns (%d):\n", len(kanban.Columns))
				for _, col := range kanban.Columns {
					_, _ = fmt.Printf("  [%s] %s (%d cards)\n", col.ID, col.Name, len(col.Cards))
				}
			}

			return nil
		},
	}
}

func kanbanUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Usage:       "Update a kanban board",
		ArgsUsage:   "<id>",
		Description: "Modify an existing kanban board",
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
		Action: func(_ context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			if cmd.NArg() == 0 {
				return fmt.Errorf("kanban ID is required")
			}
			id := cmd.Args().Get(0)

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			found := false
			for i := range s.Kanbans {
				if s.Kanbans[i].ID == id {
					if cmd.String("title") != "" {
						s.Kanbans[i].Title = cmd.String("title")
					}
					if cmd.String("description") != "" {
						s.Kanbans[i].Description = cmd.String("description")
					}
					s.Kanbans[i].UpdatedAt = time.Now()
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("kanban board not found: %s", id)
			}

			if err := store.SaveKanbans(s, profile); err != nil {
				return err
			}

			_, _ = fmt.Printf("Kanban board updated: %s\n", id)
			return nil
		},
	}
}

func kanbanDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a kanban board",
		ArgsUsage:   "<id>",
		Description: "Remove a kanban board by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			if cmd.NArg() == 0 {
				return fmt.Errorf("kanban ID is required")
			}
			id := cmd.Args().Get(0)

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Delete kanban board %s? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			found := false
			for i := range s.Kanbans {
				if s.Kanbans[i].ID == id {
					s.Kanbans = append(s.Kanbans[:i], s.Kanbans[i+1:]...)
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("kanban board not found: %s", id)
			}

			if err := store.SaveKanbans(s, profile); err != nil {
				return err
			}

			_, _ = fmt.Printf("Kanban board deleted: %s\n", id)
			return nil
		},
	}
}

func findKanbanByID(kanbans []model.Kanban, id string) *model.Kanban {
	for i := range kanbans {
		if kanbans[i].ID == id {
			return &kanbans[i]
		}
	}
	return nil
}

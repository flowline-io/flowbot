package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/internal/model"
	"github.com/flowline-io/flowbot/cmd/cli/internal/store"
	"github.com/flowline-io/flowbot/cmd/cli/pkg/utils"
	"github.com/urfave/cli/v3"
)

func kanbanCardCommand() *cli.Command {
	return &cli.Command{
		Name:        "card",
		Usage:       "Work with kanban cards",
		Description: "Manage cards within kanban boards",
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
		Description: "Create a new card in the specified column",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "board",
				Aliases:  []string{"b"},
				Usage:    "Kanban board ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "column",
				Aliases:  []string{"c"},
				Usage:    "Column ID",
				Required: true,
			},
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			boardID := cmd.String("board")
			columnID := cmd.String("column")

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			found := false
			for i := range s.Kanbans {
				if s.Kanbans[i].ID == boardID {
					for j := range s.Kanbans[i].Columns {
						if s.Kanbans[i].Columns[j].ID == columnID {
							card := model.Card{
								ID:          utils.GenerateID(),
								Title:       cmd.String("title"),
								Description: cmd.String("description"),
								Order:       len(s.Kanbans[i].Columns[j].Cards),
								CreatedAt:   time.Now(),
								UpdatedAt:   time.Now(),
							}
							s.Kanbans[i].Columns[j].Cards = append(s.Kanbans[i].Columns[j].Cards, card)
							s.Kanbans[i].UpdatedAt = time.Now()
							found = true
							_, _ = fmt.Printf("Card added: %s (%s)\n", card.Title, card.ID)
							break
						}
					}
				}
			}

			if !found {
				return fmt.Errorf("board or column not found")
			}

			return store.SaveKanbans(s, profile)
		},
	}
}

func kanbanCardMoveCommand() *cli.Command {
	return &cli.Command{
		Name:        "move",
		Usage:       "Move a card between columns",
		Description: "Move a card from one column to another",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "board",
				Aliases:  []string{"b"},
				Usage:    "Kanban board ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "card",
				Usage:    "Card ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "to-column",
				Usage:    "Destination column ID",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			boardID := cmd.String("board")
			cardID := cmd.String("card")
			toColumnID := cmd.String("to-column")

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			var card *model.Card
			var fromColumnIdx, toColumnIdx = -1, -1
			boardIdx := -1

			for i := range s.Kanbans {
				if s.Kanbans[i].ID == boardID {
					boardIdx = i
					for j := range s.Kanbans[i].Columns {
						if s.Kanbans[i].Columns[j].ID == toColumnID {
							toColumnIdx = j
						}
						for k := range s.Kanbans[i].Columns[j].Cards {
							if s.Kanbans[i].Columns[j].Cards[k].ID == cardID {
								card = &s.Kanbans[i].Columns[j].Cards[k]
								fromColumnIdx = j
							}
						}
					}
				}
			}

			if boardIdx == -1 {
				return fmt.Errorf("board not found: %s", boardID)
			}
			if card == nil {
				return fmt.Errorf("card not found: %s", cardID)
			}
			if toColumnIdx == -1 {
				return fmt.Errorf("destination column not found: %s", toColumnID)
			}
			if fromColumnIdx == -1 {
				return fmt.Errorf("card not found in any column")
			}

			cards := s.Kanbans[boardIdx].Columns[fromColumnIdx].Cards
			for i, c := range cards {
				if c.ID == cardID {
					s.Kanbans[boardIdx].Columns[fromColumnIdx].Cards = append(cards[:i], cards[i+1:]...)
					break
				}
			}

			card.UpdatedAt = time.Now()
			s.Kanbans[boardIdx].Columns[toColumnIdx].Cards = append(s.Kanbans[boardIdx].Columns[toColumnIdx].Cards, *card)
			s.Kanbans[boardIdx].UpdatedAt = time.Now()

			_, _ = fmt.Printf("Card moved: %s -> %s\n", cardID, toColumnID)
			return store.SaveKanbans(s, profile)
		},
	}
}

func kanbanCardDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a card from a kanban board",
		Description: "Remove a card by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "board",
				Aliases:  []string{"b"},
				Usage:    "Kanban board ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "card",
				Usage:    "Card ID",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			boardID := cmd.String("board")
			cardID := cmd.String("card")

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			found := false
			for i := range s.Kanbans {
				if s.Kanbans[i].ID == boardID {
					for j := range s.Kanbans[i].Columns {
						cards := s.Kanbans[i].Columns[j].Cards
						for k, c := range cards {
							if c.ID == cardID {
								s.Kanbans[i].Columns[j].Cards = append(cards[:k], cards[k+1:]...)
								found = true
								break
							}
						}
					}
				}
			}

			if !found {
				return fmt.Errorf("card not found: %s", cardID)
			}

			_, _ = fmt.Printf("Card deleted: %s\n", cardID)
			return store.SaveKanbans(s, profile)
		},
	}
}

func kanbanColumnCommand() *cli.Command {
	return &cli.Command{
		Name:        "column",
		Usage:       "Work with kanban columns",
		Description: "Manage columns within kanban boards",
		Commands: []*cli.Command{
			kanbanColumnAddCommand(),
			kanbanColumnDeleteCommand(),
		},
	}
}

func kanbanColumnAddCommand() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Usage:       "Add a column to a kanban board",
		Description: "Create a new column in the specified board",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "board",
				Aliases:  []string{"b"},
				Usage:    "Kanban board ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Column name",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			boardID := cmd.String("board")

			s, err := store.LoadKanbans(profile)
			if err != nil {
				return err
			}

			found := false
			for i := range s.Kanbans {
				if s.Kanbans[i].ID == boardID {
					column := model.Column{
						ID:    utils.GenerateID(),
						Name:  cmd.String("name"),
						Order: len(s.Kanbans[i].Columns),
						Cards: []model.Card{},
					}
					s.Kanbans[i].Columns = append(s.Kanbans[i].Columns, column)
					s.Kanbans[i].UpdatedAt = time.Now()
					found = true
					_, _ = fmt.Printf("Column added: %s (%s)\n", column.Name, column.ID)
					break
				}
			}

			if !found {
				return fmt.Errorf("board not found: %s", boardID)
			}

			return store.SaveKanbans(s, profile)
		},
	}
}

func kanbanColumnDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a column from a kanban board",
		Description: "Remove a column by ID (cards will be deleted)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "board",
				Aliases:  []string{"b"},
				Usage:    "Kanban board ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "column",
				Aliases:  []string{"c"},
				Usage:    "Column ID",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			boardID := cmd.String("board")
			columnID := cmd.String("column")

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Delete column %s? All cards in this column will be deleted. [y/N]: ", columnID)
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
				if s.Kanbans[i].ID == boardID {
					for j, c := range s.Kanbans[i].Columns {
						if c.ID == columnID {
							s.Kanbans[i].Columns = append(s.Kanbans[i].Columns[:j], s.Kanbans[i].Columns[j+1:]...)
							found = true
							break
						}
					}
					if found {
						s.Kanbans[i].UpdatedAt = time.Now()
						break
					}
				}
			}

			if !found {
				return fmt.Errorf("column not found: %s", columnID)
			}

			_, _ = fmt.Printf("Column deleted: %s\n", columnID)
			return store.SaveKanbans(s, profile)
		},
	}
}

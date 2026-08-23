// Package command implements CLI command definitions.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// TrelloCommand returns the root command for Trello boards and cards.
func TrelloCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trello",
		Short: "Work with Trello boards and cards",
		Long:  "Manage Trello boards, lists, and cards via Flowbot server",
	}
	cmd.AddCommand(
		trelloBoardListCommand(),
		trelloBoardGetCommand(),
		trelloListListCommand(),
		trelloCardListCommand(),
		trelloCardGetCommand(),
		trelloCardSearchCommand(),
		trelloCardCreateCommand(),
		trelloCardUpdateCommand(),
		trelloCardMoveCommand(),
		trelloCardDeleteCommand(),
		trelloWebhookRegisterCommand(),
		trelloWebhookDeleteCommand(),
		trelloHealthCommand(),
	)
	return cmd
}

func trelloBoardListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board list",
		Short: "List Trello boards",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			result, err := c.Trello.ListBoards(cmd.Context(), limit, cursor)
			if err != nil {
				return fmt.Errorf("list boards: %w", err)
			}
			if len(result.Data) == 0 {
				return PrintEmptyList(cmd, "No Trello boards found")
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum items per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

func trelloBoardGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "board get [board_id]",
		Short: "Get a Trello board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			board, err := c.Trello.GetBoard(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get board: %w", err)
			}
			return PrintJSON(board)
		},
	}
}

func trelloListListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list list [board_id]",
		Short: "List lists on a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			lists, err := c.Trello.ListLists(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("list lists: %w", err)
			}
			if len(lists) == 0 {
				return PrintEmptyList(cmd, "No lists found")
			}
			return PrintJSON(lists)
		},
	}
}

func trelloCardListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card list [board_id]",
		Short: "List cards on a board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			result, err := c.Trello.ListCards(cmd.Context(), args[0], limit, cursor)
			if err != nil {
				return fmt.Errorf("list cards: %w", err)
			}
			if len(result.Data) == 0 {
				return PrintEmptyList(cmd, "No cards found")
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum items per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

func trelloCardGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "card get [card_id]",
		Short: "Get a Trello card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			card, err := c.Trello.GetCard(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get card: %w", err)
			}
			return PrintJSON(card)
		},
	}
}

func trelloCardSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card search",
		Short: "Search Trello cards",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			query, _ := cmd.Flags().GetString("query")
			if query == "" {
				return fmt.Errorf("query is required")
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cards, err := c.Trello.SearchCards(cmd.Context(), query, limit)
			if err != nil {
				return fmt.Errorf("search cards: %w", err)
			}
			if len(cards) == 0 {
				return PrintEmptyList(cmd, "No cards matched")
			}
			return PrintJSON(cards)
		},
	}
	cmd.Flags().StringP("query", "q", "", "Search query")
	cmd.Flags().Int("limit", 0, "Maximum results")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func trelloCardCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card create",
		Short: "Create a Trello card",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			listID, _ := cmd.Flags().GetString("list-id")
			name, _ := cmd.Flags().GetString("name")
			desc, _ := cmd.Flags().GetString("desc")
			card, err := c.Trello.CreateCard(cmd.Context(), &client.CreateCardRequest{
				ListID: listID,
				Name:   name,
				Desc:   desc,
			})
			if err != nil {
				return fmt.Errorf("create card: %w", err)
			}
			return PrintJSON(card)
		},
	}
	cmd.Flags().String("list-id", "", "Target list ID")
	cmd.Flags().StringP("name", "n", "", "Card title")
	cmd.Flags().StringP("desc", "d", "", "Card description")
	_ = cmd.MarkFlagRequired("list-id")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func trelloCardUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card update [card_id]",
		Short: "Update a Trello card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			desc, _ := cmd.Flags().GetString("desc")
			card, err := c.Trello.UpdateCard(cmd.Context(), args[0], &client.UpdateCardRequest{
				Name: name,
				Desc: desc,
			})
			if err != nil {
				return fmt.Errorf("update card: %w", err)
			}
			return PrintJSON(card)
		},
	}
	cmd.Flags().StringP("name", "n", "", "New title")
	cmd.Flags().StringP("desc", "d", "", "New description")
	return cmd
}

func trelloCardMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card move [card_id]",
		Short: "Move a card to another list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			listID, _ := cmd.Flags().GetString("list-id")
			card, err := c.Trello.MoveCard(cmd.Context(), args[0], listID)
			if err != nil {
				return fmt.Errorf("move card: %w", err)
			}
			return PrintJSON(card)
		},
	}
	cmd.Flags().String("list-id", "", "Target list ID")
	_ = cmd.MarkFlagRequired("list-id")
	return cmd
}

func trelloCardDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card delete [card_id]",
		Short: "Delete a Trello card",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Trello.DeleteCard(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete card: %w", err)
			}
			_, _ = fmt.Printf("card deleted: %s\n", args[0])
			return nil
		},
	}
	return cmd
}

func trelloWebhookRegisterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook register",
		Short: "Register a Trello board webhook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			boardID, _ := cmd.Flags().GetString("board-id")
			callbackURL, _ := cmd.Flags().GetString("callback-url")
			description, _ := cmd.Flags().GetString("description")
			webhook, err := c.Trello.RegisterWebhook(cmd.Context(), &client.RegisterWebhookRequest{
				BoardID:     boardID,
				CallbackURL: callbackURL,
				Description: description,
			})
			if err != nil {
				return fmt.Errorf("register webhook: %w", err)
			}
			return PrintJSON(webhook)
		},
	}
	cmd.Flags().String("board-id", "", "Board ID")
	cmd.Flags().String("callback-url", "", "Callback URL")
	cmd.Flags().String("description", "", "Webhook description")
	return cmd
}

func trelloWebhookDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "webhook delete [webhook_id]",
		Short: "Delete a Trello webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Trello.DeleteWebhook(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete webhook: %w", err)
			}
			_, _ = fmt.Println("webhook deleted")
			return nil
		},
	}
}

func trelloHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check Trello connectivity",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ok, err := c.Trello.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("health check: %w", err)
			}
			if !ok {
				_, _ = fmt.Println("trello unhealthy")
				return nil
			}
			_, _ = fmt.Println("trello healthy")
			return nil
		},
	}
}

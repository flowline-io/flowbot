// Package command implements CLI command definitions.
package command

import (
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

func MemoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memo",
		Short: "Work with memos",
		Long:  "Manage memos via Flowbot server",
	}
	cmd.AddCommand(
		memoCreateCommand(),
		memoListCommand(),
		memoGetCommand(),
		memoUpdateCommand(),
		memoDeleteCommand(),
		memoHealthCommand(),
	)
	return cmd
}

func memoCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new memo",
		Long:  "Add a new memo to the Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			content, _ := cmd.Flags().GetString("content")
			visibility, _ := cmd.Flags().GetString("visibility")

			memo, err := c.Memo.Create(cmd.Context(), content, visibility)
			if err != nil {
				return fmt.Errorf("create memo: %w", err)
			}

			_, _ = fmt.Printf("Memo created: %s\n", memo.Name)
			_, _ = fmt.Printf("Content: %s\n", memo.Content)
			if memo.Visibility != "" {
				_, _ = fmt.Printf("Visibility: %s\n", memo.Visibility)
			}
			return nil
		},
	}
	cmd.Flags().StringP("content", "c", "", "Memo content")
	_ = cmd.MarkFlagRequired("content")
	cmd.Flags().StringP("visibility", "v", "", "Memo visibility (PRIVATE, PROTECTED, PUBLIC)")
	return cmd
}

func memoListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all memos",
		Long:  "Display memos from the Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			query := &client.ListMemosQuery{
				Limit: limit,
			}

			result, err := c.Memo.List(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("list memos: %w", err)
			}

			if len(result.Items) == 0 {
				_, _ = fmt.Println("No memos found")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(result.Items, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal memos: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				for _, m := range result.Items {
					pinned := ""
					if m.Pinned {
						pinned = " [pinned]"
					}
					snippet := m.Snippet
					if snippet == "" {
						snippet = truncate(m.Content, 80)
					}
					_, _ = fmt.Printf("[%s]%s %s\n", m.Name, pinned, snippet)
					if !m.CreateTime.IsZero() {
						_, _ = fmt.Printf("  Created: %s\n", m.CreateTime.Format(time.RFC3339))
					}
					if m.Visibility != "" {
						_, _ = fmt.Printf("  Visibility: %s\n", m.Visibility)
					}
					_, _ = fmt.Println()
				}
				if result.Page.NextCursor != "" {
					_, _ = fmt.Printf("--- Next cursor: %s\n", result.Page.NextCursor)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of memos")
	return cmd
}

func memoGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a memo by resource name",
		Long:  "Display details of a specific memo",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("memo name is required (e.g., memos/123)")
			}
			name := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			memo, err := c.Memo.Get(cmd.Context(), name)
			if err != nil {
				return fmt.Errorf("get memo: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(memo, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal memo: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("Name:       %s\n", memo.Name)
				_, _ = fmt.Printf("Content:    %s\n", memo.Content)
				_, _ = fmt.Printf("State:      %s\n", memo.State)
				_, _ = fmt.Printf("Visibility: %s\n", memo.Visibility)
				_, _ = fmt.Printf("Pinned:     %v\n", memo.Pinned)
				_, _ = fmt.Printf("Creator:    %s\n", memo.Creator)
				if !memo.CreateTime.IsZero() {
					_, _ = fmt.Printf("Created:    %s\n", memo.CreateTime.Format(time.RFC3339))
				}
				if !memo.UpdateTime.IsZero() {
					_, _ = fmt.Printf("Updated:    %s\n", memo.UpdateTime.Format(time.RFC3339))
				}
				if len(memo.Tags) > 0 {
					_, _ = fmt.Printf("Tags:       %v\n", memo.Tags)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func memoUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a memo",
		Long:  "Update content, visibility, or pinned status of a memo",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("memo name is required (e.g., memos/123)")
			}
			name := args[0]

			req := &client.UpdateMemoRequest{}
			hasUpdate := false

			if cmd.Flags().Changed("content") {
				content, _ := cmd.Flags().GetString("content")
				req.Content = content
				hasUpdate = true
			}
			if cmd.Flags().Changed("visibility") {
				visibility, _ := cmd.Flags().GetString("visibility")
				req.Visibility = visibility
				hasUpdate = true
			}
			if cmd.Flags().Changed("pinned") {
				pinned, _ := cmd.Flags().GetBool("pinned")
				req.Pinned = &pinned
				hasUpdate = true
			}

			if !hasUpdate {
				return fmt.Errorf("at least one of --content, --visibility, or --pinned must be provided")
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			memo, err := c.Memo.Update(cmd.Context(), name, req)
			if err != nil {
				return fmt.Errorf("update memo: %w", err)
			}

			if memo.Pinned {
				_, _ = fmt.Printf("Memo updated [pinned]: %s\n", memo.Name)
			} else {
				_, _ = fmt.Printf("Memo updated: %s\n", memo.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringP("content", "c", "", "New content")
	cmd.Flags().StringP("visibility", "v", "", "New visibility (PRIVATE, PROTECTED, PUBLIC)")
	cmd.Flags().BoolP("pinned", "p", false, "Set pinned status")
	return cmd
}

func memoDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a memo",
		Long:  "Delete a memo by its resource name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("memo name is required (e.g., memos/123)")
			}
			name := args[0]

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Delete memo %s? [y/N]: ", name)
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

			if err := c.Memo.Delete(cmd.Context(), name); err != nil {
				return fmt.Errorf("delete memo: %w", err)
			}

			_, _ = fmt.Printf("Memo deleted: %s\n", name)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func memoHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check memo backend health",
		Long:  "Check whether the memo backend is reachable",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			healthy, err := c.Memo.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("check health: %w", err)
			}

			if healthy {
				_, _ = fmt.Println("Memo backend is healthy")
			} else {
				_, _ = fmt.Println("Memo backend is NOT healthy")
			}
			return nil
		},
	}
	return cmd
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Package command implements CLI command definitions.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// TriliumCommand returns the root command for trilium notes.
func TriliumCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trilium",
		Short: "Work with trilium notes",
		Long:  "Manage trilium notes via Flowbot server",
	}
	cmd.AddCommand(
		triliumCreateCommand(),
		triliumListCommand(),
		triliumGetCommand(),
		triliumUpdateCommand(),
		triliumDeleteCommand(),
		triliumSearchCommand(),
		triliumContentCommand(),
		triliumHealthCommand(),
	)
	return cmd
}

func triliumCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new note",
		Long:  "Add a new note to the trilium backend",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			title, _ := cmd.Flags().GetString("title")
			content, _ := cmd.Flags().GetString("content")
			typ, _ := cmd.Flags().GetString("type")
			parent, _ := cmd.Flags().GetString("parent")

			note, err := c.Trilium.Create(cmd.Context(), &client.CreateNoteRequest{
				Title:        title,
				Content:      content,
				Type:         typ,
				ParentNoteID: parent,
			})
			if err != nil {
				return fmt.Errorf("create note: %w", err)
			}

			_, _ = fmt.Printf("Note created: %s\n", note.ID)
			_, _ = fmt.Printf("Title: %s\n", note.Title)
			if note.Type != "" {
				_, _ = fmt.Printf("Type: %s\n", note.Type)
			}
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "Note title")
	_ = cmd.MarkFlagRequired("title")
	cmd.Flags().StringP("content", "c", "", "Note content")
	cmd.Flags().String("type", "", "Note type (default: text)")
	cmd.Flags().StringP("parent", "p", "", "Parent note ID")
	return cmd
}

func triliumListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notes",
		Long:  "Display notes from the trilium backend",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			query, _ := cmd.Flags().GetString("query")
			result, err := c.Trilium.List(cmd.Context(), &client.ListNotesQuery{
				Limit: limit,
				Query: query,
			})
			if err != nil {
				return fmt.Errorf("list notes: %w", err)
			}

			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No notes found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result.Items)
			}
			for _, n := range result.Items {
				_, _ = fmt.Printf("[%s] %s", n.ID, n.Title)
				if n.Type != "" {
					_, _ = fmt.Printf(" (%s)", n.Type)
				}
				_, _ = fmt.Println()
				if n.DateModified != "" {
					_, _ = fmt.Printf("  Modified: %s\n", n.DateModified)
				}
				_, _ = fmt.Println()
			}
			if result.Page.NextCursor != "" {
				_, _ = fmt.Printf("--- Next cursor: %s\n", result.Page.NextCursor)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of notes")
	cmd.Flags().StringP("query", "q", "", "Optional search filter")
	return cmd
}

func triliumGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a note by ID",
		Long:  "Display details of a specific note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("note id is required")
			}
			id := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			note, err := c.Trilium.Get(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get note: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(note)
			}
			_, _ = fmt.Printf("ID:       %s\n", note.ID)
			_, _ = fmt.Printf("Title:    %s\n", note.Title)
			_, _ = fmt.Printf("Type:     %s\n", note.Type)
			_, _ = fmt.Printf("Protected: %v\n", note.IsProtected)
			if note.DateCreated != "" {
				_, _ = fmt.Printf("Created:  %s\n", note.DateCreated)
			}
			if note.DateModified != "" {
				_, _ = fmt.Printf("Modified: %s\n", note.DateModified)
			}
			if len(note.ParentNoteIDs) > 0 {
				_, _ = fmt.Printf("Parents:  %v\n", note.ParentNoteIDs)
			}
			if note.Content != "" {
				_, _ = fmt.Printf("Content:  %s\n", note.Content)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func triliumUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a note",
		Long:  "Update title and/or content of a note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("note id is required")
			}
			id := args[0]

			req := &client.UpdateNoteRequest{}
			hasUpdate := false
			if cmd.Flags().Changed("title") {
				title, _ := cmd.Flags().GetString("title")
				req.Title = title
				hasUpdate = true
			}
			if cmd.Flags().Changed("content") {
				content, _ := cmd.Flags().GetString("content")
				req.Content = content
				hasUpdate = true
			}
			if !hasUpdate {
				return fmt.Errorf("at least one of --title or --content must be provided")
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			note, err := c.Trilium.Update(cmd.Context(), id, req)
			if err != nil {
				return fmt.Errorf("update note: %w", err)
			}

			_, _ = fmt.Printf("Note updated: %s\n", note.ID)
			_, _ = fmt.Printf("Title: %s\n", note.Title)
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "New title")
	cmd.Flags().StringP("content", "c", "", "New content")
	return cmd
}

func triliumDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a note",
		Long:  "Delete a note by its ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("note id is required")
			}
			id := args[0]

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Delete note %s? [y/N]: ", id)
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

			if err := c.Trilium.Delete(cmd.Context(), id); err != nil {
				return fmt.Errorf("delete note: %w", err)
			}

			_, _ = fmt.Printf("Note deleted: %s\n", id)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func triliumSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search notes",
		Long:  "Full-text search across trilium notes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			q, _ := cmd.Flags().GetString("query")
			result, err := c.Trilium.Search(cmd.Context(), &client.SearchNotesQuery{Q: q})
			if err != nil {
				return fmt.Errorf("search notes: %w", err)
			}

			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No notes found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result.Items)
			}
			for _, n := range result.Items {
				_, _ = fmt.Printf("[%s] %s\n", n.ID, n.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringP("query", "q", "", "Search query")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func triliumContentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Work with note content",
		Long:  "Get or set the full content of a note",
	}
	cmd.AddCommand(
		triliumContentGetCommand(),
		triliumContentSetCommand(),
	)
	return cmd
}

func triliumContentGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get note content",
		Long:  "Display the full content of a note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("note id is required")
			}
			id := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			content, err := c.Trilium.GetContent(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get note content: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(map[string]string{"id": id, "content": content})
			}
			_, _ = fmt.Print(content)
			if len(content) > 0 && content[len(content)-1] != '\n' {
				_, _ = fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func triliumContentSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <id>",
		Short: "Set note content",
		Long:  "Replace the full content of a note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("note id is required")
			}
			id := args[0]
			content, _ := cmd.Flags().GetString("content")

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			if err := c.Trilium.SetContent(cmd.Context(), id, content); err != nil {
				return fmt.Errorf("set note content: %w", err)
			}

			_, _ = fmt.Printf("Note content updated: %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringP("content", "c", "", "New note content")
	_ = cmd.MarkFlagRequired("content")
	return cmd
}

func triliumHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check trilium backend health",
		Long:  "Check whether the trilium backend is reachable",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			info, err := c.Trilium.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("check health: %w", err)
			}

			_, _ = fmt.Println("Trilium backend is healthy")
			if info.Title != "" {
				_, _ = fmt.Printf("Info: %s\n", info.Title)
			}
			if info.ID != "" {
				_, _ = fmt.Printf("Instance: %s\n", info.ID)
			}
			return nil
		},
	}
	return cmd
}

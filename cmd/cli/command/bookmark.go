// Package command implements CLI command definitions.
package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/client"
)

func BookmarkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bookmark",
		Aliases: []string{"karakeep"},
		Short:   "Work with bookmarks",
		Long:    "Manage bookmarks via Flowbot server (alias: karakeep)",
	}
	cmd.AddCommand(
		bookmarkCreateCommand(),
		bookmarkListCommand(),
		bookmarkGetCommand(),
		bookmarkArchiveCommand(),
		bookmarkDeleteCommand(),
		bookmarkCheckUrlCommand(),
		bookmarkSearchCommand(),
	)
	return cmd
}

func bookmarkCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new bookmark",
		Long:  "Add a new bookmark to the Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			urlStr, _ := cmd.Flags().GetString("url")
			bookmark, err := c.Bookmark.Create(cmd.Context(), urlStr)
			if err != nil {
				return fmt.Errorf("create bookmark: %w", err)
			}

			_, _ = fmt.Printf("Bookmark created: %s (%s)\n", bookmark.Title, bookmark.ID)
			return nil
		},
	}
	cmd.Flags().StringP("url", "u", "", "Bookmark URL")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func bookmarkListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all bookmarks",
		Long:  "Display bookmarks from the Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			query := &client.ListBookmarksQuery{
				Limit:  limit,
				Cursor: cursor,
			}

			result, err := c.Bookmark.List(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("list bookmarks: %w", err)
			}

			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No bookmarks found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result.Items)
			}
			printBookmarkItems(result.Items)
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of bookmarks")
	cmd.Flags().StringP("cursor", "c", "", "Pagination cursor")
	return cmd
}

func bookmarkGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a bookmark by ID",
		Long:  "Display details of a specific bookmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			bookmark, err := c.Bookmark.Get(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get bookmark: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(bookmark)
			}
			_, _ = fmt.Printf("ID:          %s\n", bookmark.ID)
			_, _ = fmt.Printf("Title:       %s\n", bookmark.Title)
			_, _ = fmt.Printf("URL:         %s\n", bookmark.URL)
			_, _ = fmt.Printf("Description: %s\n", bookmark.Summary)
			_, _ = fmt.Printf("Tags:        %v\n", bookmark.Tags)
			_, _ = fmt.Printf("Archived:    %v\n", bookmark.Archived)
			if !bookmark.CreatedAt.IsZero() {
				_, _ = fmt.Printf("Created:     %s\n", bookmark.CreatedAt.Format(time.RFC3339))
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func bookmarkArchiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <id>",
		Short: "Toggle archive status of a bookmark",
		Long:  "Archive or unarchive a bookmark by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			bookmark, err := c.Bookmark.Get(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get bookmark: %w", err)
			}

			currentStatus := "unarchived"
			newStatus := "archived"
			if bookmark.Archived {
				currentStatus = "archived"
				newStatus = "unarchived"
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Bookmark is currently %s. Change to %s? [y/N]: ", currentStatus, newStatus)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			result, err := c.Bookmark.Archive(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("toggle archive status: %w", err)
			}

			status := "archived"
			if !result.Archived {
				status = "unarchived"
			}
			_, _ = fmt.Printf("Bookmark %s: %s\n", status, id)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func bookmarkDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete (archive) a bookmark",
		Long:  "Archive a bookmark by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := args[0]

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Archive bookmark %s? [y/N]: ", id)
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

			_, err = c.Bookmark.Archive(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("delete bookmark: %w", err)
			}

			_, _ = fmt.Printf("Bookmark archived: %s\n", id)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func bookmarkCheckUrlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-url",
		Short: "Check if a URL is already bookmarked",
		Long:  "Check if a URL exists in the bookmark collection",
		RunE: func(cmd *cobra.Command, _ []string) error {
			urlStr, _ := cmd.Flags().GetString("url")

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Bookmark.CheckUrl(cmd.Context(), urlStr)
			if err != nil {
				return fmt.Errorf("check URL: %w", err)
			}

			if result.Exists && result.ID != "" {
				_, _ = fmt.Printf("URL is bookmarked: %s (ID: %s)\n", urlStr, result.ID)
			} else {
				_, _ = fmt.Printf("URL is not bookmarked: %s\n", urlStr)
			}
			return nil
		},
	}
	cmd.Flags().StringP("url", "u", "", "URL to check")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func bookmarkSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search bookmarks",
		Long:  "Full-text search across all bookmarks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			q, _ := cmd.Flags().GetString("query")
			sortOrder, _ := cmd.Flags().GetString("sort-order")
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			includeContent, _ := cmd.Flags().GetBool("include-content")

			query := &client.SearchBookmarksQuery{
				Q:              q,
				SortOrder:      sortOrder,
				Limit:          limit,
				Cursor:         cursor,
				IncludeContent: includeContent,
			}

			result, err := c.Bookmark.Search(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("search bookmarks: %w", err)
			}

			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No bookmarks found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result.Items)
			}
			_, _ = fmt.Printf("Found %d bookmark(s):\n\n", len(result.Items))
			printBookmarkItems(result.Items)
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	cmd.Flags().StringP("query", "q", "", "Search query")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().StringP("sort-order", "s", "relevance", "Sort order (asc, desc, relevance)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of results")
	cmd.Flags().StringP("cursor", "c", "", "Pagination cursor")
	cmd.Flags().BoolP("include-content", "i", false, "Include full content in results")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func printBookmarkItems(items []*capability.Bookmark) {
	for _, b := range items {
		if b == nil {
			continue
		}
		var status []string
		if b.Archived {
			status = append(status, "archived")
		}
		if b.Favourited {
			status = append(status, "favourited")
		}

		statusStr := "-"
		if len(status) > 0 {
			statusStr = strings.Join(status, ", ")
		}

		title := b.Title
		if title == "" {
			title = b.URL
		}
		_, _ = fmt.Printf("[%s] %s\n", b.ID, title)
		_, _ = fmt.Printf("  URL:    %s\n", b.URL)
		_, _ = fmt.Printf("  Status: %s\n", statusStr)
		_, _ = fmt.Println()
	}
}

func printNextCursor(cursor string) {
	if cursor != "" {
		_, _ = fmt.Printf("--- Next cursor: %s\n", cursor)
	}
}

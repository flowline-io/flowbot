package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/client"
)

func ReaderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reader",
		Short: "Work with RSS feeds",
		Long:  "Manage RSS feeds via Flowbot server",
	}
	cmd.AddCommand(
		readerFeedListCommand(),
		readerFeedGetCommand(),
		readerFeedCreateCommand(),
		readerEntryListCommand(),
		readerEntryUpdateCommand(),
		readerFeedEntriesCommand(),
	)
	return cmd
}

func readerFeedListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all feeds",
		Long:  "Display all RSS feeds from Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			feeds, err := c.Reader.ListFeeds(cmd.Context())
			if err != nil {
				return fmt.Errorf("list feeds: %w", err)
			}

			if len(feeds) == 0 {
				return PrintEmptyList(cmd, "No feeds found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(feeds)
			}
			_, _ = fmt.Printf("%-8s %-30s %-40s %-15s\n", "ID", "TITLE", "FEED URL", "CATEGORY")
			_, _ = fmt.Println(strings.Repeat("-", 95))
			for _, f := range feeds {
				if f == nil {
					continue
				}
				title := f.Title
				if len(title) > 28 {
					title = title[:25] + "..."
				}
				feedURL := f.FeedURL
				if len(feedURL) > 38 {
					feedURL = feedURL[:35] + "..."
				}
				category := f.Category
				if category == "" {
					category = "-"
				}
				_, _ = fmt.Printf("%-8d %-30s %-40s %-15s\n", f.ID, title, feedURL, category)
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func readerFeedGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a feed by ID",
		Long:  "Display details of a specific RSS feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("feed ID is required")
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			feed, err := c.Reader.GetFeed(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get feed: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(feed)
			}
			_, _ = fmt.Printf("ID:          %d\n", feed.ID)
			_, _ = fmt.Printf("Title:       %s\n", feed.Title)
			_, _ = fmt.Printf("Feed URL:    %s\n", feed.FeedURL)
			_, _ = fmt.Printf("Site URL:    %s\n", feed.SiteURL)
			_, _ = fmt.Printf("Category:    %s\n", feed.Category)

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func readerFeedCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new feed",
		Long:  "Add a new RSS feed to the Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			urlStr, _ := cmd.Flags().GetString("url")
			category, _ := cmd.Flags().GetInt64("category")

			req := &client.CreateFeedRequest{
				FeedURL:    urlStr,
				CategoryID: category,
			}

			result, err := c.Reader.CreateFeed(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create feed: %w", err)
			}

			_, _ = fmt.Printf("Feed created: ID=%d\n", result.ID)
			if result.Title != "" {
				_, _ = fmt.Printf("Title: %s\n", result.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringP("url", "u", "", "Feed URL")
	_ = cmd.MarkFlagRequired("url")
	cmd.Flags().Int64P("category", "c", 0, "Category ID")
	return cmd
}

func readerEntryListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entries",
		Short: "List entries",
		Long:  "Display RSS entries from Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetInt("limit")
			feedID, _ := cmd.Flags().GetInt64("feed")

			query := &client.ListEntriesQuery{
				Status: status,
				Limit:  limit,
				FeedID: feedID,
			}

			entries, err := c.Reader.ListEntries(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("list entries: %w", err)
			}

			return printReaderEntries(cmd, entries)
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().StringP("status", "s", "", "Status filter (read, unread, removed)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of entries")
	cmd.Flags().Int64P("feed", "f", 0, "Filter by feed ID")
	return cmd
}

func readerEntryUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-entries",
		Short: "Update entries status",
		Long:  "Update the status of multiple entries",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			ids, _ := cmd.Flags().GetInt64Slice("ids")
			status, _ := cmd.Flags().GetString("status")

			req := &client.UpdateEntriesRequest{
				EntryIDs: ids,
				Status:   status,
			}

			_, err = c.Reader.UpdateEntriesStatus(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("update entries status: %w", err)
			}

			_, _ = fmt.Printf("Updated %d entries to status: %s\n", len(req.EntryIDs), req.Status)
			return nil
		},
	}
	cmd.Flags().Int64SliceP("ids", "i", nil, "Entry IDs to update")
	_ = cmd.MarkFlagRequired("ids")
	cmd.Flags().StringP("status", "s", "", "New status (read, unread, removed)")
	_ = cmd.MarkFlagRequired("status")
	return cmd
}

func readerFeedEntriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feed-entries <feed-id>",
		Short: "Get entries for a feed",
		Long:  "Display RSS entries for a specific feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("feed ID is required")
			}
			feedID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetInt("limit")

			query := &client.GetFeedEntriesQuery{
				Status: status,
				Limit:  limit,
			}

			entries, err := c.Reader.GetFeedEntries(cmd.Context(), feedID, query)
			if err != nil {
				return fmt.Errorf("get feed entries: %w", err)
			}

			return printReaderEntries(cmd, entries)
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().StringP("status", "s", "", "Status filter (read, unread, removed)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of entries")
	return cmd
}

func printReaderEntries(cmd *cobra.Command, entries []*capability.Entry) error {
	if len(entries) == 0 {
		return PrintEmptyList(cmd, "No entries found")
	}

	if IsJSON(cmd) {
		return PrintJSON(entries)
	}

	_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "STATUS", "STARRED")
	_, _ = fmt.Println(strings.Repeat("-", 65))
	for _, e := range entries {
		if e == nil {
			continue
		}
		title := e.Title
		if len(title) > 28 {
			title = title[:25] + "..."
		}
		starred := "no"
		if e.Starred {
			starred = "yes"
		}
		_, _ = fmt.Printf("%-8d %-30s %-15s %-10s\n", e.ID, title, e.Status, starred)
	}
	return nil
}

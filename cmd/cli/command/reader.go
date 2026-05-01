package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/urfave/cli/v3"
)

func ReaderCommand() *cli.Command {
	return &cli.Command{
		Name:        "reader",
		Usage:       "Work with RSS feeds",
		Description: "Manage RSS feeds via Flowbot server",
		Commands: []*cli.Command{
			readerFeedListCommand(),
			readerFeedGetCommand(),
			readerFeedCreateCommand(),
			readerFeedUpdateCommand(),
			readerFeedRefreshCommand(),
			readerEntryListCommand(),
			readerEntryUpdateCommand(),
			readerFeedEntriesCommand(),
		},
	}
}

func readerFeedListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all feeds",
		Description: "Display all RSS feeds from Flowbot server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			feeds, err := c.Reader.ListFeeds(ctx)
			if err != nil {
				return fmt.Errorf("list feeds: %w", err)
			}

			if len(feeds) == 0 {
				_, _ = fmt.Println("No feeds found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(feeds, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal feeds: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-40s %-10s\n", "ID", "TITLE", "FEED URL", "STATUS")
				_, _ = fmt.Println(strings.Repeat("-", 90))
				for _, f := range feeds {
					title := f.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					feedURL := f.FeedURL
					if len(feedURL) > 38 {
						feedURL = feedURL[:35] + "..."
					}
					status := "active"
					if f.Disabled {
						status = "disabled"
					}
					_, _ = fmt.Printf("%-8d %-30s %-40s %-10s\n", f.ID, title, feedURL, status)
				}
			}

			return nil
		},
	}
}

func readerFeedGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get a feed by ID",
		ArgsUsage:   "<id>",
		Description: "Display details of a specific RSS feed",
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
				return fmt.Errorf("feed ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			feed, err := c.Reader.GetFeed(ctx, id)
			if err != nil {
				return fmt.Errorf("get feed: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(feed, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal feed: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %d\n", feed.ID)
				_, _ = fmt.Printf("Title:       %s\n", feed.Title)
				_, _ = fmt.Printf("Feed URL:    %s\n", feed.FeedURL)
				_, _ = fmt.Printf("Site URL:    %s\n", feed.SiteURL)
				_, _ = fmt.Printf("Disabled:    %v\n", feed.Disabled)
				_, _ = fmt.Printf("Checked At:  %v\n", feed.CheckedAt)
			}

			return nil
		},
	}
}

func readerFeedCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new feed",
		Description: "Add a new RSS feed to the Flowbot server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Feed URL",
				Required: true,
			},
			&cli.Int64Flag{
				Name:    "category",
				Aliases: []string{"c"},
				Usage:   "Category ID",
				Value:   0,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := &client.CreateFeedRequest{
				FeedURL:    cmd.String("url"),
				CategoryID: cmd.Int64("category"),
			}

			result, err := c.Reader.CreateFeed(ctx, req)
			if err != nil {
				return fmt.Errorf("create feed: %w", err)
			}

			_, _ = fmt.Printf("Feed created: ID=%d\n", result.ID)
			return nil
		},
	}
}

func readerFeedUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Usage:       "Update a feed",
		ArgsUsage:   "<id>",
		Description: "Modify an existing RSS feed",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Usage:   "New title",
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "New feed URL",
			},
			&cli.BoolFlag{
				Name:  "disable",
				Usage: "Disable the feed",
			},
			&cli.BoolFlag{
				Name:  "enable",
				Usage: "Enable the feed",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("feed ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := &client.UpdateFeedRequest{}
			if title := cmd.String("title"); title != "" {
				req.Title = title
			}
			if url := cmd.String("url"); url != "" {
				req.FeedURL = url
			}
			if cmd.Bool("disable") {
				disabled := true
				req.Disabled = &disabled
			}
			if cmd.Bool("enable") {
				disabled := false
				req.Disabled = &disabled
			}

			feed, err := c.Reader.UpdateFeed(ctx, id, req)
			if err != nil {
				return fmt.Errorf("update feed: %w", err)
			}

			_, _ = fmt.Printf("Feed updated: %s\n", feed.Title)
			return nil
		},
	}
}

func readerFeedRefreshCommand() *cli.Command {
	return &cli.Command{
		Name:        "refresh",
		Usage:       "Refresh a feed",
		ArgsUsage:   "<id>",
		Description: "Trigger a refresh of a specific RSS feed",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("feed ID is required")
			}
			idStr := cmd.Args().Get(0)
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			_, err = c.Reader.RefreshFeed(ctx, id)
			if err != nil {
				return fmt.Errorf("refresh feed: %w", err)
			}

			_, _ = fmt.Printf("Feed refreshed: ID=%d\n", id)
			return nil
		},
	}
}

func readerEntryListCommand() *cli.Command {
	return &cli.Command{
		Name:        "entries",
		Usage:       "List entries",
		Description: "Display RSS entries from Flowbot server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.StringFlag{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Status filter (read, unread, removed)",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Maximum number of entries",
				Value:   20,
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "Pagination offset",
				Value: 0,
			},
			&cli.BoolFlag{
				Name:  "starred",
				Usage: "Starred entries only",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			query := &client.ListEntriesQuery{
				Status:  cmd.String("status"),
				Limit:   int(cmd.Int("limit")),
				Offset:  int(cmd.Int("offset")),
				Starred: cmd.Bool("starred"),
			}

			result, err := c.Reader.ListEntries(ctx, query)
			if err != nil {
				return fmt.Errorf("list entries: %w", err)
			}

			if len(result.Entries) == 0 {
				_, _ = fmt.Println("No entries found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result.Entries, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal entries: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("Total: %d entries\n\n", result.Total)
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "STATUS", "STARRED")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for _, e := range result.Entries {
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
			}

			return nil
		},
	}
}

func readerEntryUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update-entries",
		Usage:       "Update entries status",
		Description: "Update the status of multiple entries",
		Flags: []cli.Flag{
			&cli.Int64SliceFlag{
				Name:     "ids",
				Aliases:  []string{"i"},
				Usage:    "Entry IDs to update",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "status",
				Aliases:  []string{"s"},
				Usage:    "New status (read, unread, removed)",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := &client.UpdateEntriesRequest{
				EntryIDs: cmd.Int64Slice("ids"),
				Status:   cmd.String("status"),
			}

			_, err = c.Reader.UpdateEntriesStatus(ctx, req)
			if err != nil {
				return fmt.Errorf("update entries status: %w", err)
			}

			_, _ = fmt.Printf("Updated %d entries to status: %s\n", len(req.EntryIDs), req.Status)
			return nil
		},
	}
}

func readerFeedEntriesCommand() *cli.Command {
	return &cli.Command{
		Name:        "feed-entries",
		Usage:       "Get entries for a feed",
		ArgsUsage:   "<feed-id>",
		Description: "Display RSS entries for a specific feed",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.StringFlag{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "Status filter (read, unread, removed)",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Maximum number of entries",
				Value:   20,
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "Pagination offset",
				Value: 0,
			},
			&cli.BoolFlag{
				Name:  "starred",
				Usage: "Starred entries only",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("feed ID is required")
			}
			feedIDStr := cmd.Args().Get(0)
			feedID, err := strconv.ParseInt(feedIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid feed ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			query := &client.GetFeedEntriesQuery{
				Status:  cmd.String("status"),
				Limit:   int(cmd.Int("limit")),
				Offset:  int(cmd.Int("offset")),
				Starred: cmd.Bool("starred"),
			}

			result, err := c.Reader.GetFeedEntries(ctx, feedID, query)
			if err != nil {
				return fmt.Errorf("get feed entries: %w", err)
			}

			if len(result.Entries) == 0 {
				_, _ = fmt.Println("No entries found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result.Entries, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal entries: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("Total: %d entries for feed %d\n\n", result.Total, feedID)
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "STATUS", "STARRED")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for _, e := range result.Entries {
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
			}

			return nil
		},
	}
}

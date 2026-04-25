package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/urfave/cli/v3"
)

func BookmarkCommand() *cli.Command {
	return &cli.Command{
		Name:        "bookmark",
		Usage:       "Work with bookmarks",
		Description: "Manage bookmarks via Flowbot server",
		Commands: []*cli.Command{
			bookmarkCreateCommand(),
			bookmarkListCommand(),
			bookmarkGetCommand(),
			bookmarkDeleteCommand(),
		},
	}
}

func bookmarkCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new bookmark",
		Description: "Add a new bookmark to the Flowbot server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Bookmark URL",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			urlStr := cmd.String("url")
			bookmark, err := c.Bookmark.Create(ctx, urlStr)
			if err != nil {
				return fmt.Errorf("create bookmark: %w", err)
			}

			title := ""
			if bookmark.Title != nil {
				title = *bookmark.Title
			}
			_, _ = fmt.Printf("Bookmark created: %s (%s)\n", title, bookmark.Id)
			return nil
		},
	}
}

func bookmarkListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all bookmarks",
		Description: "Display bookmarks from the Flowbot server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "Maximum number of bookmarks",
				Value:   20,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			query := &client.ListBookmarksQuery{
				Limit: int(cmd.Int("limit")),
			}

			result, err := c.Bookmark.List(ctx, query)
			if err != nil {
				return fmt.Errorf("list bookmarks: %w", err)
			}

			if len(result.Bookmarks) == 0 {
				_, _ = fmt.Println("No bookmarks found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result.Bookmarks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal bookmarks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-12s %-30s %-50s\n", "ID", "TITLE", "URL")
				_, _ = fmt.Println(strings.Repeat("-", 94))
				for _, b := range result.Bookmarks {
					id := b.Id
					if len(id) > 10 {
						id = id[:8] + ".."
					}
					title := b.GetTitle()
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					url := ""
					if b.Content.Url != "" {
						url = b.Content.Url
					}
					if len(url) > 48 {
						url = url[:45] + "..."
					}
					_, _ = fmt.Printf("%-12s %-30s %-50s\n", id, title, url)
				}
			}

			return nil
		},
	}
}

func bookmarkGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get a bookmark by ID",
		ArgsUsage:   "<id>",
		Description: "Display details of a specific bookmark",
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
				return fmt.Errorf("bookmark ID is required")
			}
			id := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			bookmark, err := c.Bookmark.Get(ctx, id)
			if err != nil {
				return fmt.Errorf("get bookmark: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(bookmark, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal bookmark: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				createdAt := bookmark.CreatedAt
				if t, err := time.Parse(time.RFC3339, bookmark.CreatedAt); err == nil {
					createdAt = t.Format(time.RFC3339)
				}
				title := ""
				if bookmark.Title != nil {
					title = *bookmark.Title
				}
				description := ""
				if bookmark.Summary != nil {
					description = *bookmark.Summary
				}
				_, _ = fmt.Printf("ID:          %s\n", bookmark.Id)
				_, _ = fmt.Printf("Title:       %s\n", title)
				_, _ = fmt.Printf("URL:         %s\n", bookmark.Content.Url)
				_, _ = fmt.Printf("Description: %s\n", description)
				tagNames := make([]string, 0, len(bookmark.Tags))
				for _, tag := range bookmark.Tags {
					tagNames = append(tagNames, tag.Name)
				}
				_, _ = fmt.Printf("Tags:        %v\n", tagNames)
				_, _ = fmt.Printf("Archived:    %v\n", bookmark.Archived)
				_, _ = fmt.Printf("Created:     %s\n", createdAt)
			}

			return nil
		},
	}
}

func bookmarkDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete (archive) a bookmark",
		ArgsUsage:   "<id>",
		Description: "Archive a bookmark by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := cmd.Args().Get(0)

			if !cmd.Bool("yes") {
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

			_, err = c.Bookmark.Archive(ctx, id)
			if err != nil {
				return fmt.Errorf("delete bookmark: %w", err)
			}

			_, _ = fmt.Printf("Bookmark archived: %s\n", id)
			return nil
		},
	}
}

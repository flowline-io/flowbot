package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/cmd/cli/internal/store"
	"github.com/flowline-io/flowbot/cmd/cli/pkg/client"
	"github.com/urfave/cli/v3"
)

// BookmarkCommand returns the bookmark parent command.
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
			c, err := newBookmarkClient(cmd)
			if err != nil {
				return err
			}

			urlStr := cmd.String("url")
			body := map[string]string{"url": urlStr}

			var result bookmarkItem
			if err := c.Post("/service/bookmark", body, &result); err != nil {
				return fmt.Errorf("create bookmark: %w", err)
			}

			_, _ = fmt.Printf("Bookmark created: %s (%s)\n", result.Title, result.ID)
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
			c, err := newBookmarkClient(cmd)
			if err != nil {
				return err
			}

			limit := int(cmd.Int("limit"))

			var result bookmarkListResult
			path := fmt.Sprintf("/service/bookmark?limit=%d", limit)
			if err := c.Get(path, &result); err != nil {
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
					id := b.ID
					if len(id) > 10 {
						id = id[:8] + ".."
					}
					title := b.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					url := b.URL
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

			c, err := newBookmarkClient(cmd)
			if err != nil {
				return err
			}

			var result bookmarkItem
			if err := c.Get("/service/bookmark/"+id, &result); err != nil {
				return fmt.Errorf("get bookmark: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal bookmark: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				createdAt := result.CreatedAt
				if t, err := time.Parse(time.RFC3339, result.CreatedAt); err == nil {
					createdAt = t.Format(time.RFC3339)
				}
				_, _ = fmt.Printf("ID:          %s\n", result.ID)
				_, _ = fmt.Printf("Title:       %s\n", result.Title)
				_, _ = fmt.Printf("URL:         %s\n", result.URL)
				_, _ = fmt.Printf("Description: %s\n", result.Description)
				_, _ = fmt.Printf("Tags:        %v\n", result.Tags)
				_, _ = fmt.Printf("Archived:    %v\n", result.Archived)
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

			c, err := newBookmarkClient(cmd)
			if err != nil {
				return err
			}

			body := map[string]bool{"archived": true}
			var result map[string]any
			if err := c.Patch("/service/bookmark/"+id, body, &result); err != nil {
				return fmt.Errorf("delete bookmark: %w", err)
			}

			_, _ = fmt.Printf("Bookmark archived: %s\n", id)
			return nil
		},
	}
}

func newBookmarkClient(cmd *cli.Command) (*client.Client, error) {
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

// Response types for bookmark webservice responses.

type bookmarkListResult struct {
	Bookmarks  []bookmarkItem `json:"bookmarks"`
	NextCursor string         `json:"nextCursor"`
}

type bookmarkItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	CreatedAt   string   `json:"createdAt"`
	Archived    bool     `json:"archived"`
}

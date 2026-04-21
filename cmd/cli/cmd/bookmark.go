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

// BookmarkCommand returns the bookmark parent command
func BookmarkCommand() *cli.Command {
	return &cli.Command{
		Name:        "bookmark",
		Usage:       "Work with bookmarks",
		Description: "Manage bookmarks locally",
		Commands: []*cli.Command{
			bookmarkCreateCommand(),
			bookmarkListCommand(),
			bookmarkGetCommand(),
			bookmarkUpdateCommand(),
			bookmarkDeleteCommand(),
		},
	}
}

func bookmarkCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Usage:       "Create a new bookmark",
		Description: "Add a new bookmark to local storage",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "title",
				Aliases:  []string{"t"},
				Usage:    "Bookmark title",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Bookmark URL",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Bookmark description",
			},
			&cli.StringSliceFlag{
				Name:  "tags",
				Usage: "Tags for the bookmark",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			title := cmd.String("title")
			urlStr := cmd.String("url")
			description := cmd.String("description")

			if err := store.ValidateBookmark(title, urlStr, description); err != nil {
				return err
			}

			s, err := store.LoadBookmarks(profile)
			if err != nil {
				return err
			}

			bookmark := model.Bookmark{
				ID:          utils.GenerateID(),
				Title:       title,
				URL:         urlStr,
				Description: description,
				Tags:        cmd.StringSlice("tags"),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			s.Bookmarks = append(s.Bookmarks, bookmark)

			if err := store.SaveBookmarks(s, profile); err != nil {
				return err
			}

			_, _ = fmt.Printf("Bookmark created: %s (%s)\n", bookmark.Title, bookmark.ID)
			return nil
		},
	}
}

func bookmarkListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all bookmarks",
		Description: "Display all saved bookmarks",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.StringSliceFlag{
				Name:  "tag",
				Usage: "Filter by tag",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")

			s, err := store.LoadBookmarks(profile)
			if err != nil {
				return err
			}

			filters := cmd.StringSlice("tag")
			bookmarks := filterBookmarksByTags(s.Bookmarks, filters)

			if len(bookmarks) == 0 {
				_, _ = fmt.Println("No bookmarks found")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(bookmarks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal bookmarks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-12s %-30s %-50s\n", "ID", "TITLE", "URL")
				_, _ = fmt.Println(string(make([]byte, 94)))
				for _, b := range bookmarks {
					title := b.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					url := b.URL
					if len(url) > 48 {
						url = url[:45] + "..."
					}
					_, _ = fmt.Printf("%-12s %-30s %-50s\n", b.ID, title, url)
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
			profile := cmd.String("profile")
			if cmd.NArg() == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := cmd.Args().Get(0)

			s, err := store.LoadBookmarks(profile)
			if err != nil {
				return err
			}

			bookmark := findBookmarkByID(s.Bookmarks, id)
			if bookmark == nil {
				return fmt.Errorf("bookmark not found: %s", id)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(bookmark, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal bookmark: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %s\n", bookmark.ID)
				_, _ = fmt.Printf("Title:       %s\n", bookmark.Title)
				_, _ = fmt.Printf("URL:         %s\n", bookmark.URL)
				_, _ = fmt.Printf("Description: %s\n", bookmark.Description)
				_, _ = fmt.Printf("Tags:        %v\n", bookmark.Tags)
				_, _ = fmt.Printf("Created:     %s\n", bookmark.CreatedAt.Format(time.RFC3339))
				_, _ = fmt.Printf("Updated:     %s\n", bookmark.UpdatedAt.Format(time.RFC3339))
			}

			return nil
		},
	}
}

func bookmarkUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Usage:       "Update a bookmark",
		ArgsUsage:   "<id>",
		Description: "Modify an existing bookmark",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Usage:   "New title",
			},
			&cli.StringFlag{
				Name:    "url",
				Aliases: []string{"u"},
				Usage:   "New URL",
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "New description",
			},
			&cli.StringSliceFlag{
				Name:  "tags",
				Usage: "New tags (replaces existing)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			if cmd.NArg() == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := cmd.Args().Get(0)

			s, err := store.LoadBookmarks(profile)
			if err != nil {
				return err
			}

			found := false
			for i, b := range s.Bookmarks {
				if b.ID == id {
					newTitle := b.Title
					newURL := b.URL
					newDesc := b.Description

					if cmd.String("title") != "" {
						newTitle = cmd.String("title")
					}
					if cmd.String("url") != "" {
						newURL = cmd.String("url")
					}
					if cmd.String("description") != "" {
						newDesc = cmd.String("description")
					}

					if err := store.ValidateBookmark(newTitle, newURL, newDesc); err != nil {
						return err
					}

					s.Bookmarks[i].Title = newTitle
					s.Bookmarks[i].URL = newURL
					s.Bookmarks[i].Description = newDesc
					if len(cmd.StringSlice("tags")) > 0 {
						s.Bookmarks[i].Tags = cmd.StringSlice("tags")
					}
					s.Bookmarks[i].UpdatedAt = time.Now()
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("bookmark not found: %s", id)
			}

			if err := store.SaveBookmarks(s, profile); err != nil {
				return err
			}

			_, _ = fmt.Printf("Bookmark updated: %s\n", id)
			return nil
		},
	}
}

func bookmarkDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:        "delete",
		Usage:       "Delete a bookmark",
		ArgsUsage:   "<id>",
		Description: "Remove a bookmark by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profile := cmd.String("profile")
			if cmd.NArg() == 0 {
				return fmt.Errorf("bookmark ID is required")
			}
			id := cmd.Args().Get(0)

			if !cmd.Bool("yes") {
				_, _ = fmt.Printf("Delete bookmark %s? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			s, err := store.LoadBookmarks(profile)
			if err != nil {
				return err
			}

			found := false
			for i, b := range s.Bookmarks {
				if b.ID == id {
					s.Bookmarks = append(s.Bookmarks[:i], s.Bookmarks[i+1:]...)
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("bookmark not found: %s", id)
			}

			if err := store.SaveBookmarks(s, profile); err != nil {
				return err
			}

			_, _ = fmt.Printf("Bookmark deleted: %s\n", id)
			return nil
		},
	}
}

func filterBookmarksByTags(bookmarks []model.Bookmark, tags []string) []model.Bookmark {
	if len(tags) == 0 {
		return bookmarks
	}

	var filtered []model.Bookmark
	for _, b := range bookmarks {
		if hasAllTags(b.Tags, tags) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

func hasAllTags(bookmarkTags, filterTags []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range bookmarkTags {
		tagSet[t] = true
	}
	for _, t := range filterTags {
		if !tagSet[t] {
			return false
		}
	}
	return true
}

func findBookmarkByID(bookmarks []model.Bookmark, id string) *model.Bookmark {
	for i := range bookmarks {
		if bookmarks[i].ID == id {
			return &bookmarks[i]
		}
	}
	return nil
}

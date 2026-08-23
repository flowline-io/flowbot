// Package command implements CLI command definitions.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// ConfluenceCommand returns the root command for Confluence pages.
func ConfluenceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confluence",
		Short: "Work with Confluence spaces and pages",
		Long:  "Manage Confluence Cloud spaces and pages via Flowbot server",
	}
	cmd.AddCommand(
		confluenceSpaceListCommand(),
		confluencePageListCommand(),
		confluencePageGetCommand(),
		confluencePageContentCommand(),
		confluencePageSearchCommand(),
		confluencePageCreateCommand(),
		confluencePageUpdateCommand(),
		confluencePageDeleteCommand(),
		confluenceHealthCommand(),
	)
	return cmd
}

func confluenceSpaceListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "space list",
		Short: "List Confluence spaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			result, err := c.Confluence.ListSpaces(cmd.Context(), limit, cursor)
			if err != nil {
				return fmt.Errorf("list spaces: %w", err)
			}
			if len(result.Data) == 0 {
				return PrintEmptyList(cmd, "No spaces found")
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum items per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

func confluencePageListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page list [space_key]",
		Short: "List pages in a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			result, err := c.Confluence.ListPages(cmd.Context(), args[0], limit, cursor)
			if err != nil {
				return fmt.Errorf("list pages: %w", err)
			}
			if len(result.Data) == 0 {
				return PrintEmptyList(cmd, "No pages found")
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().Int("limit", 0, "Maximum items per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

func confluencePageGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "page get [page_id]",
		Short: "Get a Confluence page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			page, err := c.Confluence.GetPage(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get page: %w", err)
			}
			return PrintJSON(page)
		},
	}
}

func confluencePageContentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "page content [page_id]",
		Short: "Get page storage content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			content, err := c.Confluence.GetPageContent(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get page content: %w", err)
			}
			_, _ = fmt.Println(content)
			return nil
		},
	}
}

func confluencePageSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page search",
		Short: "Search pages with CQL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			cql, _ := cmd.Flags().GetString("cql")
			if cql == "" {
				return fmt.Errorf("cql is required")
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			result, err := c.Confluence.SearchPages(cmd.Context(), cql, limit, cursor)
			if err != nil {
				return fmt.Errorf("search pages: %w", err)
			}
			if len(result.Data) == 0 {
				return PrintEmptyList(cmd, "No pages matched")
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().String("cql", "", "CQL query")
	cmd.Flags().Int("limit", 0, "Maximum items per page")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	_ = cmd.MarkFlagRequired("cql")
	return cmd
}

func confluencePageCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page create",
		Short: "Create a Confluence page",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			spaceKey, _ := cmd.Flags().GetString("space-key")
			title, _ := cmd.Flags().GetString("title")
			content, _ := cmd.Flags().GetString("content")
			page, err := c.Confluence.CreatePage(cmd.Context(), &client.CreatePageRequest{
				SpaceKey: spaceKey,
				Title:    title,
				Content:  content,
			})
			if err != nil {
				return fmt.Errorf("create page: %w", err)
			}
			return PrintJSON(page)
		},
	}
	cmd.Flags().String("space-key", "", "Space key")
	cmd.Flags().StringP("title", "t", "", "Page title")
	cmd.Flags().StringP("content", "c", "", "Storage-format XHTML content")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func confluencePageUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page update [page_id]",
		Short: "Update a Confluence page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			title, _ := cmd.Flags().GetString("title")
			content, _ := cmd.Flags().GetString("content")
			page, err := c.Confluence.UpdatePage(cmd.Context(), args[0], &client.UpdatePageRequest{
				Title:   title,
				Content: content,
			})
			if err != nil {
				return fmt.Errorf("update page: %w", err)
			}
			return PrintJSON(page)
		},
	}
	cmd.Flags().StringP("title", "t", "", "New title")
	cmd.Flags().StringP("content", "c", "", "Storage-format XHTML content")
	return cmd
}

func confluencePageDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "page delete [page_id]",
		Short: "Delete a Confluence page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Confluence.DeletePage(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete page: %w", err)
			}
			_, _ = fmt.Printf("page deleted: %s\n", args[0])
			return nil
		},
	}
}

func confluenceHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check Confluence connectivity",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ok, err := c.Confluence.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("health check: %w", err)
			}
			if !ok {
				_, _ = fmt.Println("confluence unhealthy")
				return nil
			}
			_, _ = fmt.Println("confluence healthy")
			return nil
		},
	}
}

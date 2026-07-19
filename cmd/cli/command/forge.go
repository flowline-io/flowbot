package command

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

func ForgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "forge",
		Aliases: []string{"gitea"},
		Short:   "Work with software forges",
		Long:    "Access forge resources via Flowbot server (alias: gitea)",
	}
	cmd.AddCommand(
		forgeUserCommand(),
		forgeRepoCommand(),
		forgeIssuesCommand(),
		forgeIssueCommand(),
		forgeDiffCommand(),
		forgeFileCommand(),
	)
	return cmd
}

func forgeUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Get authenticated forge user",
		Long:  "Display the authenticated user from the configured forge",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			user, err := c.Forge.GetUser(cmd.Context())
			if err != nil {
				return fmt.Errorf("get forge user: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(user)
			}
			_, _ = fmt.Printf("ID:        %d\n", user.ID)
			_, _ = fmt.Printf("Username:  %s\n", user.UserName)
			_, _ = fmt.Printf("Email:     %s\n", user.Email)
			_, _ = fmt.Printf("Avatar:    %s\n", user.AvatarURL)

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func forgeRepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo <owner> <repo>",
		Short: "Get a repository",
		Long:  "Display repository details from the forge",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			repo, err := c.Forge.GetRepo(cmd.Context(), args[0], args[1])
			if err != nil {
				return fmt.Errorf("get forge repo: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(repo)
			}
			_, _ = fmt.Printf("ID:          %d\n", repo.ID)
			_, _ = fmt.Printf("Name:        %s\n", repo.Name)
			_, _ = fmt.Printf("Full Name:   %s\n", repo.FullName)
			_, _ = fmt.Printf("Description: %s\n", repo.Description)
			_, _ = fmt.Printf("Private:     %v\n", repo.Private)
			_, _ = fmt.Printf("HTML URL:    %s\n", repo.HTMLURL)
			_, _ = fmt.Printf("Clone URL:   %s\n", repo.CloneURL)
			_, _ = fmt.Printf("Owner:       %s\n", repo.Owner)

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func forgeIssuesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues <owner>",
		Short: "List issues",
		Long:  "List issues for an owner from the forge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			state, _ := cmd.Flags().GetString("state")
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")

			query := &client.ListIssuesQuery{
				State:  state,
				Limit:  limit,
				Cursor: cursor,
			}

			issues, err := c.Forge.ListIssues(cmd.Context(), args[0], query)
			if err != nil {
				return fmt.Errorf("list forge issues: %w", err)
			}

			if len(issues) == 0 {
				return PrintEmptyList(cmd, "No issues found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(issues)
			}
			_, _ = fmt.Printf("%-6s %-30s %-10s %-10s\n", "NUMBER", "TITLE", "STATE", "AUTHOR")
			_, _ = fmt.Println("---------------------------------------------------------------")
			for _, iss := range issues {
				title := iss.Title
				if len(title) > 28 {
					title = title[:25] + "..."
				}
				_, _ = fmt.Printf("%-6d %-30s %-10s %-10s\n", iss.Index, title, iss.State, iss.Author)
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().StringP("state", "s", "", "Issue state filter (open, closed)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of issues")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

func forgeIssueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <owner> <repo> <index>",
		Short: "Get an issue",
		Long:  "Display a single issue from the forge by owner, repo, and index",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			index, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue index: %w", err)
			}

			issue, err := c.Forge.GetIssue(cmd.Context(), args[0], args[1], index)
			if err != nil {
				return fmt.Errorf("get forge issue: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(issue)
			}
			_, _ = fmt.Printf("ID:       %d\n", issue.ID)
			_, _ = fmt.Printf("Number:   #%d\n", issue.Index)
			_, _ = fmt.Printf("Title:    %s\n", issue.Title)
			_, _ = fmt.Printf("State:    %s\n", issue.State)
			_, _ = fmt.Printf("Author:   %s\n", issue.Author)
			_, _ = fmt.Printf("URL:      %s\n", issue.HTMLURL)
			_, _ = fmt.Printf("Body:\n%s\n", issue.Body)

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func forgeDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <owner> <repo> <commit-id>",
		Short: "Get commit diff",
		Long:  "Display the diff for a specific commit from the forge",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			diff, err := c.Forge.GetCommitDiff(cmd.Context(), args[0], args[1], args[2])
			if err != nil {
				return fmt.Errorf("get forge commit diff: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(diff)
			}
			_, _ = fmt.Printf("Commit:  %s\n", diff.CommitID)
			_, _ = fmt.Printf("Message: %s\n", diff.CommitMessage)
			_, _ = fmt.Printf("Files:   %v\n", diff.Files)
			_, _ = fmt.Printf("\n%s\n", diff.DiffContent)

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func forgeFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <owner> <repo> <commit-id> <file-path>",
		Short: "Get file content",
		Long:  "Display the content of a file at a specific commit from the forge",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			lineStart, _ := cmd.Flags().GetInt("line-start")
			lineCount, _ := cmd.Flags().GetInt("line-count")

			query := &client.FileContentQuery{}
			if lineStart > 0 {
				query.LineStart = lineStart
			}
			if lineCount > 0 {
				query.LineCount = lineCount
			}

			content, err := c.Forge.GetFileContent(cmd.Context(), args[0], args[1], args[2], args[3], query)
			if err != nil {
				return fmt.Errorf("get forge file content: %w", err)
			}

			_, _ = fmt.Println(content)
			return nil
		},
	}
	cmd.Flags().Int("line-start", 0, "Starting line number")
	cmd.Flags().Int("line-count", 0, "Number of lines to return")
	return cmd
}

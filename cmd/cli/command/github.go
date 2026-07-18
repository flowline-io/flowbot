package command

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

func GithubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Work with GitHub",
		Long:  "Access GitHub resources via Flowbot server",
	}
	cmd.AddCommand(
		githubUserCommand(),
		githubUserByLoginCommand(),
		githubRepoCommand(),
		githubIssuesCommand(),
		githubIssueCommand(),
		githubDiffCommand(),
		githubFileCommand(),
		githubNotificationsCommand(),
		githubReleasesCommand(),
	)
	return cmd
}

func githubUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Get authenticated GitHub user",
		Long:  "Display the authenticated GitHub user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			user, err := c.Github.GetUser(cmd.Context())
			if err != nil {
				return fmt.Errorf("get github user: %w", err)
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

func githubUserByLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user-by-login <login>",
		Short: "Get GitHub user by login",
		Long:  "Display a GitHub user by their login name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			user, err := c.Github.GetUserByLogin(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get github user by login: %w", err)
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

func githubRepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo <owner> <repo>",
		Short: "Get a GitHub repository",
		Long:  "Display repository details from GitHub",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			repo, err := c.Github.GetRepo(cmd.Context(), args[0], args[1])
			if err != nil {
				return fmt.Errorf("get github repo: %w", err)
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

func githubIssuesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues <owner>",
		Short: "List GitHub issues",
		Long:  "List issues for an owner from GitHub",
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

			issues, err := c.Github.ListIssues(cmd.Context(), args[0], query)
			if err != nil {
				return fmt.Errorf("list github issues: %w", err)
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

func githubIssueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <owner> <repo> <number>",
		Short: "Get a GitHub issue",
		Long:  "Display a single GitHub issue by owner, repo, and issue number",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			number, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue number: %w", err)
			}

			issue, err := c.Github.GetIssue(cmd.Context(), args[0], args[1], number)
			if err != nil {
				return fmt.Errorf("get github issue: %w", err)
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

func githubDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <owner> <repo> <commit-id>",
		Short: "Get GitHub commit diff",
		Long:  "Display the diff for a specific GitHub commit",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			diff, err := c.Github.GetCommitDiff(cmd.Context(), args[0], args[1], args[2])
			if err != nil {
				return fmt.Errorf("get github commit diff: %w", err)
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

func githubFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <owner> <repo> <commit-id> <file-path>",
		Short: "Get GitHub file content",
		Long:  "Display the content of a file at a specific GitHub commit",
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

			content, err := c.Github.GetFileContent(cmd.Context(), args[0], args[1], args[2], args[3], query)
			if err != nil {
				return fmt.Errorf("get github file content: %w", err)
			}

			_, _ = fmt.Println(content)
			return nil
		},
	}
	cmd.Flags().Int("line-start", 0, "Starting line number")
	cmd.Flags().Int("line-count", 0, "Number of lines to return")
	return cmd
}

func githubNotificationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications",
		Short: "List GitHub notifications",
		Long:  "Display the authenticated user's GitHub notifications",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")

			query := &client.ListNotificationsQuery{
				Limit:  limit,
				Cursor: cursor,
			}

			notifications, err := c.Github.ListNotifications(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("list github notifications: %w", err)
			}

			if len(notifications) == 0 {
				return PrintEmptyList(cmd, "No notifications found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(notifications)
			}
			_, _ = fmt.Printf("%-12s %-15s %-8s %-30s %-20s\n", "ID", "REASON", "UNREAD", "SUBJECT", "REPO")
			_, _ = fmt.Println("---------------------------------------------------------------")
			for _, n := range notifications {
				subject := n.Subject
				if len(subject) > 28 {
					subject = subject[:25] + "..."
				}
				repoName := n.RepoName
				if len(repoName) > 18 {
					repoName = repoName[:15] + "..."
				}
				unread := "no"
				if n.Unread {
					unread = "yes"
				}
				_, _ = fmt.Printf("%-12s %-15s %-8s %-30s %-20s\n", n.ID, n.Reason, unread, subject, repoName)
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of notifications")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

func githubReleasesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "releases <owner> <repo>",
		Short: "List GitHub releases",
		Long:  "Display releases for a GitHub repository",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")

			query := &client.ListNotificationsQuery{
				Limit:  limit,
				Cursor: cursor,
			}

			releases, err := c.Github.ListReleases(cmd.Context(), args[0], args[1], query)
			if err != nil {
				return fmt.Errorf("list github releases: %w", err)
			}

			if len(releases) == 0 {
				return PrintEmptyList(cmd, "No releases found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(releases)
			}
			_, _ = fmt.Printf("%-6s %-20s %-25s %-10s %-12s\n", "ID", "TAG", "NAME", "DRAFT", "PRERELEASE")
			_, _ = fmt.Println("-----------------------------------------------------------------------------------")
			for _, r := range releases {
				name := r.Name
				if len(name) > 23 {
					name = name[:20] + "..."
				}
				draft := "no"
				if r.Draft {
					draft = "yes"
				}
				pre := "no"
				if r.Prerelease {
					pre = "yes"
				}
				_, _ = fmt.Printf("%-6d %-20s %-25s %-10s %-12s\n", r.ID, r.TagName, name, draft, pre)
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of releases")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	return cmd
}

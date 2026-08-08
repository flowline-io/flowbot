package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// EmailCommand returns the root command for email.
func EmailCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "email",
		Short: "Work with email",
		Long:  "Send and read email via Flowbot SMTP/IMAP capability",
	}
	cmd.AddCommand(
		emailSendCommand(),
		emailListCommand(),
		emailGetCommand(),
		emailSearchCommand(),
		emailMarkReadCommand(),
		emailMarkUnreadCommand(),
		emailHealthCommand(),
	)
	return cmd
}

func emailSendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an email",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			to, _ := cmd.Flags().GetStringSlice("to")
			cc, _ := cmd.Flags().GetStringSlice("cc")
			bcc, _ := cmd.Flags().GetStringSlice("bcc")
			subject, _ := cmd.Flags().GetString("subject")
			text, _ := cmd.Flags().GetString("text")
			html, _ := cmd.Flags().GetString("html")
			fromName, _ := cmd.Flags().GetString("from-name")
			if err := c.Email.Send(cmd.Context(), &client.SendEmailRequest{
				To: to, Cc: cc, Bcc: bcc, Subject: subject, Text: text, HTML: html, FromName: fromName,
			}); err != nil {
				return fmt.Errorf("send email: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Email sent")
			return nil
		},
	}
	cmd.Flags().StringSlice("to", nil, "Recipient addresses")
	cmd.Flags().StringSlice("cc", nil, "Cc addresses")
	cmd.Flags().StringSlice("bcc", nil, "Bcc addresses")
	cmd.Flags().String("subject", "", "Subject")
	cmd.Flags().String("text", "", "Plain text body")
	cmd.Flags().String("html", "", "HTML body")
	cmd.Flags().String("from-name", "", "From display name")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("subject")
	return cmd
}

func emailListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			mailbox, _ := cmd.Flags().GetString("mailbox")
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			var unseen *bool
			if cmd.Flags().Changed("unseen-only") {
				v, _ := cmd.Flags().GetBool("unseen-only")
				unseen = &v
			}
			items, err := c.Email.ListMessages(cmd.Context(), mailbox, unseen, limit, cursor)
			if err != nil {
				return fmt.Errorf("list email: %w", err)
			}
			if items == nil || len(items.Items) == 0 {
				return PrintEmptyList(cmd, "No messages found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(items)
			}
			for _, item := range items.Items {
				from := ""
				if len(item.From) > 0 {
					from = item.From[0]
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", item.ID, from, item.Subject, item.Date.Format("2006-01-02"))
			}
			if items.Page.NextCursor != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "next_cursor\t%s\n", items.Page.NextCursor)
			}
			return nil
		},
	}
	cmd.Flags().String("mailbox", "", "Mailbox name")
	cmd.Flags().Bool("unseen-only", false, "Only unseen messages")
	cmd.Flags().Int("limit", 20, "Page size")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func emailGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a message by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			msg, err := c.Email.GetMessage(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get email: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(msg)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nSubject: %s\nFrom: %s\nTo: %s\n",
				msg.ID, msg.Subject, strings.Join(msg.From, ", "), strings.Join(msg.To, ", "))
			if msg.Text != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", msg.Text)
			} else if msg.HTML != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", msg.HTML)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func emailSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search messages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			req := &client.EmailSearchRequest{}
			req.Mailbox, _ = cmd.Flags().GetString("mailbox")
			req.From, _ = cmd.Flags().GetString("from")
			req.To, _ = cmd.Flags().GetString("to")
			req.Subject, _ = cmd.Flags().GetString("subject")
			req.Since, _ = cmd.Flags().GetString("since")
			req.Before, _ = cmd.Flags().GetString("before")
			req.Limit, _ = cmd.Flags().GetInt("limit")
			req.Cursor, _ = cmd.Flags().GetString("cursor")
			if cmd.Flags().Changed("unseen") {
				v, _ := cmd.Flags().GetBool("unseen")
				req.Unseen = &v
			}
			items, err := c.Email.SearchMessages(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("search email: %w", err)
			}
			if items == nil || len(items.Items) == 0 {
				return PrintEmptyList(cmd, "No messages found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(items)
			}
			for _, item := range items.Items {
				from := ""
				if len(item.From) > 0 {
					from = item.From[0]
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", item.ID, from, item.Subject)
			}
			if items.Page.NextCursor != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "next_cursor\t%s\n", items.Page.NextCursor)
			}
			return nil
		},
	}
	cmd.Flags().String("mailbox", "", "Mailbox name")
	cmd.Flags().String("from", "", "From filter")
	cmd.Flags().String("to", "", "To filter")
	cmd.Flags().String("subject", "", "Subject filter")
	cmd.Flags().String("since", "", "RFC3339 since")
	cmd.Flags().String("before", "", "RFC3339 before")
	cmd.Flags().Bool("unseen", false, "Unseen only")
	cmd.Flags().Int("limit", 20, "Page size")
	cmd.Flags().String("cursor", "", "Pagination cursor")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func emailMarkReadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark-read <id>",
		Short: "Mark a message as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Email.MarkRead(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("mark read: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Marked read")
			return nil
		},
	}
	return cmd
}

func emailMarkUnreadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark-unread <id>",
		Short: "Mark a message as unread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Email.MarkUnread(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("mark unread: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Marked unread")
			return nil
		},
	}
	return cmd
}

func emailHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check email backend health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ok, err := c.Email.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("email health: %w", err)
			}
			if ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "healthy")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "unhealthy")
			}
			return nil
		},
	}
	return cmd
}

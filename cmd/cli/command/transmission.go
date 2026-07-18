// Package command implements CLI command definitions.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// TransmissionCommand returns the root command for Transmission downloads.
func TransmissionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transmission",
		Short: "Work with Transmission downloads",
		Long:  "Manage Transmission torrents via Flowbot server",
	}
	cmd.AddCommand(
		transmissionAddCommand(),
		transmissionListCommand(),
		transmissionStopCommand(),
		transmissionRemoveCommand(),
		transmissionHealthCommand(),
	)
	return cmd
}

func transmissionAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a torrent",
		Long:  "Add a torrent by magnet link or HTTP(S) .torrent URL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			url, _ := cmd.Flags().GetString("url")
			torrent, err := c.Transmission.AddTorrent(cmd.Context(), &client.AddTorrentRequest{URL: url})
			if err != nil {
				return fmt.Errorf("add torrent: %w", err)
			}

			_, _ = fmt.Printf("Torrent added: %d\n", torrent.ID)
			if torrent.Name != "" {
				_, _ = fmt.Printf("Name: %s\n", torrent.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringP("url", "u", "", "Magnet link or torrent file URL")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func transmissionListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List torrents",
		Long:  "Display torrents from Transmission",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			items, err := c.Transmission.ListTorrents(cmd.Context())
			if err != nil {
				return fmt.Errorf("list torrents: %w", err)
			}

			if len(items) == 0 {
				return PrintEmptyList(cmd, "No torrents found")
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(items)
			}
			for _, item := range items {
				_, _ = fmt.Printf("%d\t%s\t%s\t%.0f%%\n", item.ID, item.Status, item.Name, item.PercentDone*100)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func transmissionStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop torrents",
		Long:  "Stop one or more torrents by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			ids, _ := cmd.Flags().GetInt64Slice("ids")
			if err := c.Transmission.StopTorrents(cmd.Context(), ids); err != nil {
				return fmt.Errorf("stop torrents: %w", err)
			}
			_, _ = fmt.Printf("Stopped %d torrent(s)\n", len(ids))
			return nil
		},
	}
	cmd.Flags().Int64Slice("ids", nil, "Torrent IDs to stop")
	_ = cmd.MarkFlagRequired("ids")
	return cmd
}

func transmissionRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove torrents",
		Long:  "Remove one or more torrents by ID (downloaded data is kept)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			ids, _ := cmd.Flags().GetInt64Slice("ids")
			if err := c.Transmission.RemoveTorrents(cmd.Context(), ids); err != nil {
				return fmt.Errorf("remove torrents: %w", err)
			}
			_, _ = fmt.Printf("Removed %d torrent(s)\n", len(ids))
			return nil
		},
	}
	cmd.Flags().Int64Slice("ids", nil, "Torrent IDs to remove")
	_ = cmd.MarkFlagRequired("ids")
	return cmd
}

func transmissionHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check Transmission backend health",
		Long:  "Check whether the Transmission backend is reachable",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			healthy, err := c.Transmission.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("check health: %w", err)
			}

			if healthy {
				_, _ = fmt.Println("Transmission backend is healthy")
			} else {
				_, _ = fmt.Println("Transmission backend is NOT healthy")
			}
			return nil
		},
	}
	return cmd
}

package command

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

func HubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Flowbot hub (homelab apps, capabilities, health)",
		Long:  "Hub management plane for homelab app registry, capability registry, and health checks.",
	}
	cmd.AddCommand(
		hubAppsCommand(),
		hubCapabilitiesCommand(),
		hubHealthCommand(),
	)
	return cmd
}

func hubAppsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Manage homelab apps",
		Long:  "List, inspect, and manage homelab applications registered in the hub.",
	}
	cmd.AddCommand(
		hubAppsListCommand(),
		hubAppsStatusCommand(),
		hubAppsLogsCommand(),
		hubAppsRestartCommand(),
	)
	return cmd
}

func hubAppsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all homelab apps",
		Long:  "Display all registered homelab applications with their status.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			apps, err := c.Hub.ListApps(cmd.Context())
			if err != nil {
				return fmt.Errorf("list apps: %w", err)
			}

			if len(apps) == 0 {
				_, _ = fmt.Println("No apps registered")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(apps, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal apps: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-24s %-12s %-12s\n", "NAME", "STATUS", "HEALTH")
				_, _ = fmt.Printf("%s\n", strings.Repeat("-", 50))
				for _, a := range apps {
					_, _ = fmt.Printf("%-24s %-12s %-12s\n", a.Name, a.Status, a.Health)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func hubAppsStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Get app status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("app name is required")
			}
			name := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			status, err := c.Hub.GetAppStatus(cmd.Context(), name)
			if err != nil {
				return fmt.Errorf("get app status: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(status, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal status: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("App:    %s\n", status.Name)
				_, _ = fmt.Printf("Status: %s\n", status.Status)
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func hubAppsLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Get app logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("app name is required")
			}
			name := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			tail, _ := cmd.Flags().GetInt("tail")
			logs, err := c.Hub.GetAppLogs(cmd.Context(), name, tail)
			if err != nil {
				return fmt.Errorf("get app logs: %w", err)
			}

			for _, line := range logs.Logs {
				_, _ = fmt.Println(line)
			}

			return nil
		},
	}
	cmd.Flags().IntP("tail", "n", 100, "Number of lines to tail")
	return cmd
}

func hubAppsRestartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a homelab app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("app name is required")
			}
			name := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Hub.RestartApp(cmd.Context(), name)
			if err != nil {
				return fmt.Errorf("restart app: %w", err)
			}

			_, _ = fmt.Printf("App %s: %v\n", name, result)
			return nil
		},
	}
	return cmd
}

func hubCapabilitiesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "List capabilities",
		Long:  "Display all registered capability descriptors with their backends.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			caps, err := c.Hub.ListCapabilities(cmd.Context())
			if err != nil {
				return fmt.Errorf("list capabilities: %w", err)
			}

			if len(caps) == 0 {
				_, _ = fmt.Println("No capabilities registered")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(caps, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal capabilities: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-18s %-14s %-14s %s\n", "CAPABILITY", "BACKEND", "APP", "HEALTHY")
				_, _ = fmt.Printf("%s\n", strings.Repeat("-", 64))
				for _, c := range caps {
					healthy := "no"
					if c.Healthy {
						healthy = "yes"
					}
					_, _ = fmt.Printf("%-18s %-14s %-14s %s\n", c.Type, c.Backend, c.App, healthy)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func hubHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check hub health",
		Long:  "Display overall hub health including capability and app statuses.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			health, err := c.Hub.GetHealth(cmd.Context())
			if err != nil {
				return fmt.Errorf("check health: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(health, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal health: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("Hub Status: %s\n", health.Status)
				if health.Timestamp != "" {
					_, _ = fmt.Printf("Timestamp: %s\n", health.Timestamp)
				}
				_, _ = fmt.Println()

				if len(health.Details) > 0 {
					_, _ = fmt.Printf("Capabilities:\n")
					for _, d := range health.Details {
						_, _ = fmt.Printf("  %-18s (backend: %-14s app: %s) [%s]\n",
							d.Capability, d.Backend, d.App, d.Status)
					}
					_, _ = fmt.Println()
				}

				if len(health.AppStatuses) > 0 {
					_, _ = fmt.Printf("Apps:\n")
					for _, a := range health.AppStatuses {
						_, _ = fmt.Printf("  %-24s status: %-12s health: %s\n",
							a.Name, a.Status, a.Health)
					}
				}
			}

			if health.Status != "healthy" {
				return fmt.Errorf("hub status is %s", health.Status)
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

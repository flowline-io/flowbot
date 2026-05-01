package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/urfave/cli/v3"
)

func HubCommand() *cli.Command {
	return &cli.Command{
		Name:        "hub",
		Usage:       "Manage the Flowbot hub (homelab apps, capabilities, health)",
		Description: "Hub management plane for homelab app registry, capability registry, and health checks.",
		Commands: []*cli.Command{
			hubAppsCommand(),
			hubCapabilitiesCommand(),
			hubHealthCommand(),
		},
	}
}

func hubAppsCommand() *cli.Command {
	return &cli.Command{
		Name:        "apps",
		Usage:       "Manage homelab apps",
		Description: "List, inspect, and manage homelab applications registered in the hub.",
		Commands: []*cli.Command{
			hubAppsListCommand(),
			hubAppsStatusCommand(),
			hubAppsLogsCommand(),
			hubAppsRestartCommand(),
		},
	}
}

func hubAppsListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all homelab apps",
		Description: "Display all registered homelab applications with their status.",
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

			apps, err := c.Hub.ListApps(ctx)
			if err != nil {
				return fmt.Errorf("list apps: %w", err)
			}

			if len(apps) == 0 {
				_, _ = fmt.Println("No apps registered")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(apps, "", "  ")
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
}

func hubAppsStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Get app status",
		ArgsUsage: "<name>",
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
				return fmt.Errorf("app name is required")
			}
			name := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			status, err := c.Hub.GetAppStatus(ctx, name)
			if err != nil {
				return fmt.Errorf("get app status: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(status, "", "  ")
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
}

func hubAppsLogsCommand() *cli.Command {
	return &cli.Command{
		Name:      "logs",
		Usage:     "Get app logs",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "tail",
				Aliases: []string{"n"},
				Usage:   "Number of lines to tail",
				Value:   100,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("app name is required")
			}
			name := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			logs, err := c.Hub.GetAppLogs(ctx, name, int(cmd.Int("tail")))
			if err != nil {
				return fmt.Errorf("get app logs: %w", err)
			}

			for _, line := range logs.Logs {
				_, _ = fmt.Println(line)
			}

			return nil
		},
	}
}

func hubAppsRestartCommand() *cli.Command {
	return &cli.Command{
		Name:      "restart",
		Usage:     "Restart a homelab app",
		ArgsUsage: "<name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("app name is required")
			}
			name := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Hub.RestartApp(ctx, name)
			if err != nil {
				return fmt.Errorf("restart app: %w", err)
			}

			_, _ = fmt.Printf("App %s: %v\n", name, result)
			return nil
		},
	}
}

func hubCapabilitiesCommand() *cli.Command {
	return &cli.Command{
		Name:        "capabilities",
		Usage:       "List capabilities",
		Description: "Display all registered capability descriptors with their backends.",
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

			caps, err := c.Hub.ListCapabilities(ctx)
			if err != nil {
				return fmt.Errorf("list capabilities: %w", err)
			}

			if len(caps) == 0 {
				_, _ = fmt.Println("No capabilities registered")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(caps, "", "  ")
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
}

func hubHealthCommand() *cli.Command {
	return &cli.Command{
		Name:        "health",
		Usage:       "Check hub health",
		Description: "Display overall hub health including capability and app statuses.",
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

			health, err := c.Hub.GetHealth(ctx)
			if err != nil {
				return fmt.Errorf("check health: %w", err)
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(health, "", "  ")
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
}

package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/urfave/cli/v3"
)

// ConfigCommand returns the config parent command
func ConfigCommand() *cli.Command {
	return &cli.Command{
		Name:        "config",
		Usage:       "Manage configuration for flowbot",
		Description: "View and modify flowbot configuration",
		Commands: []*cli.Command{
			configGetCommand(),
			configSetCommand(),
			configListCommand(),
		},
	}
}

func configGetCommand() *cli.Command {
	return &cli.Command{
		Name:        "get",
		Usage:       "Get a configuration value",
		ArgsUsage:   "<key>",
		Description: "Retrieve a specific configuration setting",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("configuration key is required")
			}
			key := cmd.Args().Get(0)

			switch key {
			case "server-url":
				val := cmd.String("server-url")
				if val == "" {
					val = "not set"
				}
				_, _ = fmt.Println(val)
			case "profile":
				val := cmd.String("profile")
				if val == "" {
					val = "default"
				}
				_, _ = fmt.Println(val)
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}
			return nil
		},
	}
}

func configSetCommand() *cli.Command {
	return &cli.Command{
		Name:        "set",
		Usage:       "Set a configuration value",
		ArgsUsage:   "<key> <value>",
		Description: "Modify a configuration setting (stored in environment or config file)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("both key and value are required")
			}
			key := cmd.Args().Get(0)
			value := cmd.Args().Get(1)

			_, _ = fmt.Printf("Configuration '%s' set to '%s'\n", key, value)
			_, _ = fmt.Println("Note: Set environment variable FLOWBOT_" + toEnvKey(key) + "=" + value)
			return nil
		},
	}
}

func configListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all configuration values",
		Description: "Display all current configuration settings",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, _ = fmt.Println("Current Configuration:")
			_, _ = fmt.Println("----------------------")

			serverURL := cmd.String("server-url")
			if serverURL == "" {
				serverURL = "not set (use --server-url or FLOWBOT_SERVER_URL)"
			}
			_, _ = fmt.Printf("server-url: %s\n", serverURL)

			profile := cmd.String("profile")
			if profile == "" {
				profile = "default"
			}
			_, _ = fmt.Printf("profile: %s\n", profile)

			token, _ := store.LoadToken(cmd.String("profile"))
			if token == "" {
				_, _ = fmt.Println("token: not logged in")
			} else {
				_, _ = fmt.Println("token: [stored]")
			}

			return nil
		},
	}
}

func toEnvKey(key string) string {
	return strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
}

// VersionCommand returns the version command
func VersionCommand(version string) *cli.Command {
	return &cli.Command{
		Name:        "version",
		Usage:       "Print version information",
		Description: "Display the version of flowbot CLI",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, _ = fmt.Printf("flowbot version %s\n", version)
			return nil
		},
	}
}

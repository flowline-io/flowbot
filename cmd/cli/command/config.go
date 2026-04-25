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

			profile := cmd.String("profile")

			switch key {
			case "server-url":
				val := cmd.String("server-url")
				if val == "" {
					stored, err := store.LoadServerURL(profile)
					if err != nil {
						return fmt.Errorf("load server URL: %w", err)
					}
					if stored != "" {
						val = stored
					} else {
						val = "not set"
					}
				}
				_, _ = fmt.Println(val)
			case "debug":
				val := cmd.Bool("debug")
				if !val {
					stored, err := store.LoadDebug(profile)
					if err != nil {
						return fmt.Errorf("load debug: %w", err)
					}
					val = stored
				}
				_, _ = fmt.Println(formatBool(val))
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

			profile := cmd.String("profile")

			switch key {
			case "server-url":
				if err := store.SaveServerURL(value, profile); err != nil {
					return fmt.Errorf("save server URL: %w", err)
				}
				_, _ = fmt.Printf("Configuration '%s' set to '%s'\n", key, value)
			case "debug":
				enabled := value == "on" || value == "true" || value == "1"
				if err := store.SaveDebug(enabled, profile); err != nil {
					return fmt.Errorf("save debug: %w", err)
				}
				_, _ = fmt.Printf("Configuration '%s' set to '%s'\n", key, formatBool(enabled))
			default:
				_, _ = fmt.Printf("Configuration '%s' set to '%s'\n", key, value)
				_, _ = fmt.Println("Note: Set environment variable FLOWBOT_" + toEnvKey(key) + "=" + value)
			}
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

			profile := cmd.String("profile")

			serverURL := cmd.String("server-url")
			if serverURL == "" {
				stored, err := store.LoadServerURL(profile)
				if err == nil && stored != "" {
					serverURL = stored
				} else {
					serverURL = "not set (use 'config set server-url <url>' or FLOWBOT_SERVER_URL)"
				}
			}
			_, _ = fmt.Printf("server-url: %s\n", serverURL)

			displayProfile := profile
			if displayProfile == "" {
				displayProfile = "default"
			}
			_, _ = fmt.Printf("profile: %s\n", displayProfile)

			token, _ := store.LoadToken(profile)
			if token == "" {
				_, _ = fmt.Println("token: not logged in")
			} else {
				_, _ = fmt.Println("token: [stored]")
			}

			debug, _ := store.LoadDebug(profile)
			_, _ = fmt.Printf("debug: %s\n", formatBool(debug))

			return nil
		},
	}
}

func toEnvKey(key string) string {
	return strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
}

func formatBool(b bool) string {
	if b {
		return "on"
	}
	return "off"
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

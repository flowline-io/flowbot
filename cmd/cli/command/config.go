package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/store"
)

// ConfigCommand returns the config parent command
func ConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration for flowbot",
		Long:  "View and modify flowbot configuration",
	}
	cmd.AddCommand(
		configGetCommand(),
		configSetCommand(),
		configListCommand(),
	)
	return cmd
}

func configGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  "Retrieve a specific configuration setting",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("configuration key is required")
			}
			key := args[0]

			profile, _ := cmd.Flags().GetString("profile")

			switch key {
			case "server-url":
				val, _ := cmd.Flags().GetString("server-url")
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
				val, _ := cmd.Flags().GetBool("debug")
				if !val {
					stored, err := store.LoadDebug(profile)
					if err != nil {
						return fmt.Errorf("load debug: %w", err)
					}
					val = stored
				}
				_, _ = fmt.Println(formatBool(val))
			case "profile":
				val, _ := cmd.Flags().GetString("profile")
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
	return cmd
}

func configSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Modify a configuration setting (stored in environment or config file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("both key and value are required")
			}
			key := args[0]
			value := args[1]

			profile, _ := cmd.Flags().GetString("profile")

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
	return cmd
}

func configListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long:  "Display all current configuration settings",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Println("Current Configuration:")
			_, _ = fmt.Println("----------------------")

			profile, _ := cmd.Flags().GetString("profile")

			serverURL, _ := cmd.Flags().GetString("server-url")
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
	return cmd
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
func VersionCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display the version of flowbot CLI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Printf("flowbot version %s\n", version)
			return nil
		},
	}
	return cmd
}

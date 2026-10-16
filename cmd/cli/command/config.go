package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"errors"
	"github.com/flowline-io/flowbot/cmd/cli/store"
	"github.com/flowline-io/flowbot/cmd/cli/utils"
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
				return errors.New("configuration key is required")
			}
			key := args[0]

			profile, _ := cmd.Flags().GetString("profile")

			var shown string
			var value any
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
				shown = val
				value = val
			case "debug":
				val, _ := cmd.Flags().GetBool("debug")
				if !val {
					stored, err := store.LoadDebug(profile)
					if err != nil {
						return fmt.Errorf("load debug: %w", err)
					}
					val = stored
				}
				shown = formatBool(val)
				value = val
			case "profile":
				val, _ := cmd.Flags().GetString("profile")
				if val == "" {
					val = "default"
				}
				shown = val
				value = val
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}
			if IsJSON(cmd) {
				return render(cmd, map[string]any{"key": key, "value": value})
			}
			return writef(cmd, "%s\n", shown)
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
				return errors.New("both key and value are required")
			}
			key := args[0]
			value := args[1]

			profile, _ := cmd.Flags().GetString("profile")

			var stored any
			note := ""
			switch key {
			case "server-url":
				if err := store.SaveServerURL(value, profile); err != nil {
					return fmt.Errorf("save server URL: %w", err)
				}
				stored = value
			case "debug":
				enabled := value == "on" || value == "true" || value == "1"
				if err := store.SaveDebug(enabled, profile); err != nil {
					return fmt.Errorf("save debug: %w", err)
				}
				stored = enabled
				value = formatBool(enabled)
			default:
				stored = value
				note = "Note: Set environment variable FLOWBOT_" + toEnvKey(key) + "=" + value
			}
			if IsJSON(cmd) {
				return render(cmd, map[string]any{"key": key, "value": stored})
			}
			if err := writef(cmd, "Configuration '%s' set to '%s'\n", key, value); err != nil {
				return err
			}
			if note != "" {
				return writef(cmd, "%s\n", note)
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

			displayProfile := profile
			if displayProfile == "" {
				displayProfile = "default"
			}

			tokenLabel := "[stored]"
			token, err := store.LoadToken(profile)
			if err != nil {
				tokenLabel = fmt.Sprintf("error loading (%v)", err)
			} else if token == "" {
				tokenLabel = "not logged in"
			}

			debugLabel := ""
			var debugValue any
			debug, err := store.LoadDebug(profile)
			if err != nil {
				debugLabel = fmt.Sprintf("error loading (%v)", err)
				debugValue = debugLabel
			} else {
				debugLabel = formatBool(debug)
				debugValue = debug
			}

			if IsJSON(cmd) {
				return render(cmd, map[string]any{
					"server-url": serverURL,
					"profile":    displayProfile,
					"token":      tokenLabel,
					"debug":      debugValue,
				})
			}

			if err := writef(cmd, "Current Configuration:\n----------------------\n"); err != nil {
				return err
			}
			if err := writef(cmd, "server-url: %s\n", serverURL); err != nil {
				return err
			}
			if err := writef(cmd, "profile: %s\n", displayProfile); err != nil {
				return err
			}
			if err := writef(cmd, "token: %s\n", tokenLabel); err != nil {
				return err
			}
			return writef(cmd, "debug: %s\n", debugLabel)
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
func VersionCommand(clientVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display the client and server versions of flowbot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			data := map[string]any{"client": clientVersion}
			cl, err := utils.NewPublicClient(cmd)
			if err != nil {
				data["server_error"] = err.Error()
			} else if info, vErr := cl.Version(cmd.Context()); vErr != nil {
				data["server_error"] = vErr.Error()
			} else {
				data["server"] = info.Version
			}
			if IsJSON(cmd) {
				return render(cmd, data)
			}
			if err := writef(cmd, "Client version: %s\n", clientVersion); err != nil {
				return err
			}
			if msg, ok := data["server_error"].(string); ok {
				return writef(cmd, "Server version: unavailable (%s)\n", msg)
			}
			return writef(cmd, "Server version: %s\n", data["server"])
		},
	}
	return cmd
}

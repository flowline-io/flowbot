// Package main is the entry point for the Flowbot CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/command"
	"github.com/flowline-io/flowbot/version"
)

const (
	appName  = "flowbot"
	appUsage = "Work seamlessly with Flowbot from the command line"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           appName,
		Short:         appUsage,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().String("profile", "", "Configuration profile name (e.g. dev)")
	rootCmd.PersistentFlags().String("server-url", "", "Flowbot server URL")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode (prints HTTP request/response logs)")

	rootCmd.AddCommand(
		command.LoginCommand(),
		command.HubCommand(),
		command.PipelineCommand(),
		command.WorkflowCommand(),
		command.BookmarkCommand(),
		command.KanbanCommand(),
		command.ReaderCommand(),
		command.ForgeCommand(),
		command.GithubCommand(),
		command.MemoCommand(),
		command.TriliumCommand(),
		command.FireflyiiiCommand(),
		command.TransmissionCommand(),
		command.NocodbCommand(),
		command.ConfigCommand(),
		command.VersionCommand(version.Buildtags),
	)

	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		if command.IsJSON(cmd) {
			command.PrintJSONError(err)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

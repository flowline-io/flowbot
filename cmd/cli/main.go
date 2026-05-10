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
		Use:          appName,
		Short:        appUsage,
		SilenceUsage: true,
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
		command.ConfigCommand(),
		command.VersionCommand(version.Buildtags),
	)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

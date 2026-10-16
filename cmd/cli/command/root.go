package command

import (
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	appName  = "flowbot"
	appUsage = "Work seamlessly with Flowbot from the command line"
)

// NewRoot builds the flowbot CLI root command and its persistent flags.
func NewRoot(buildtags string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               appName,
		Short:             appUsage,
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: utils.ValidateOutput,
	}
	rootCmd.PersistentFlags().String("profile", "", "Configuration profile name (e.g. dev)")
	rootCmd.PersistentFlags().String("server-url", "", "Flowbot server URL")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode (prints HTTP request/response logs)")
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format (table, json)")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().Bool("no-input", false, "Fail instead of prompting; implied when stdin is not a terminal")
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return types.Errorf(types.ErrInvalidArgument, "%s", err.Error())
	})

	rootCmd.AddCommand(
		LoginCommand(),
		HubCommand(),
		PipelineCommand(),
		BlueprintCommand(),
		FunctionCommand(),
		WorkflowCommand(),
		BookmarkCommand(),
		KanbanCommand(),
		ReaderCommand(),
		ForgeCommand(),
		GithubCommand(),
		MemoCommand(),
		TriliumCommand(),
		TrelloCommand(),
		ConfluenceCommand(),
		FireflyiiiCommand(),
		TransmissionCommand(),
		EmailCommand(),
		NocodbCommand(),
		DevopsCommand(),
		ConfigCommand(),
		MCPCommand(),
		VersionCommand(buildtags),
	)
	return rootCmd
}

// Execute runs the CLI and returns the command that ran and any error.
func Execute(buildtags string) (*cobra.Command, error) {
	return NewRoot(buildtags).ExecuteC()
}

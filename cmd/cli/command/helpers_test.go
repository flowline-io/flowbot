package command

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func subcommandNames(cmd *cobra.Command) []string {
	names := make([]string, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	return names
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

func checkAllLeavesHaveRunE(t *testing.T, cmd *cobra.Command, path string) {
	t.Helper()
	if !cmd.HasSubCommands() {
		if cmd.Name() == "" {
			return
		}
		require.NotNilf(t, cmd.RunE, "leaf command %q has no RunE", path)
		return
	}
	for _, sub := range cmd.Commands() {
		subPath := path + " " + sub.Name()
		checkAllLeavesHaveRunE(t, sub, subPath)
	}
}

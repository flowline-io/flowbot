package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipelineCommand(t *testing.T) {
	cmd := PipelineCommand()

	require.Equal(t, "pipeline", cmd.Use)
	require.True(t, cmd.HasSubCommands())

	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "list")
	require.Contains(t, subNames, "run")
}

func TestPipelineListCommand(t *testing.T) {
	cmd := PipelineCommand()
	listCmd := findSubcommand(cmd, "list")
	require.NotNil(t, listCmd)
	require.NotNil(t, listCmd.RunE)

	output := listCmd.Flags().Lookup("output")
	require.NotNil(t, output)
}

func TestPipelineRunCommand(t *testing.T) {
	cmd := PipelineCommand()
	runCmd := findSubcommand(cmd, "run")
	require.NotNil(t, runCmd)
	require.Contains(t, runCmd.Use, "run")
	require.NotNil(t, runCmd.RunE)
}

package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowCommand(t *testing.T) {
	cmd := WorkflowCommand()

	require.Equal(t, "workflow", cmd.Use)
	require.True(t, cmd.HasSubCommands())

	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "run")
}

func TestWorkflowRunCommand(t *testing.T) {
	cmd := WorkflowCommand()
	runCmd := findSubcommand(cmd, "run")
	require.NotNil(t, runCmd)
	require.Contains(t, runCmd.Use, "run")
	require.NotNil(t, runCmd.RunE)
}

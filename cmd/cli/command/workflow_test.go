package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "workflow command has correct use and subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WorkflowCommand()

			require.Equal(t, "workflow", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "run")
		})
	}
}

func TestWorkflowRunCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "workflow run command has correct use and RunE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := WorkflowCommand()
			runCmd := findSubcommand(cmd, "run")
			require.NotNil(t, runCmd)
			require.Contains(t, runCmd.Use, "run")
			require.NotNil(t, runCmd.RunE)
		})
	}
}

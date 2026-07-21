package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wantUse string
	}{
		{name: "workflow command has correct use", wantUse: "workflow"},
		{name: "workflow command has apply subcommand", wantUse: "workflow"},
		{name: "workflow command has run subcommand", wantUse: "workflow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := WorkflowCommand()

			require.Equal(t, tt.wantUse, cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "apply")
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "export")
			require.Contains(t, subNames, "delete")
			require.Contains(t, subNames, "run")
			require.Contains(t, subNames, "runs")
		})
	}
}

func TestWorkflowSubcommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		sub      string
		wantFlag string
	}{
		{name: "apply requires file flag", sub: "apply", wantFlag: "file"},
		{name: "list has output flag", sub: "list", wantFlag: "output"},
		{name: "run has input flag", sub: "run", wantFlag: "input"},
		{name: "runs has output flag", sub: "runs", wantFlag: "output"},
		{name: "export has output flag", sub: "export", wantFlag: "output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := WorkflowCommand()
			sub := findSubcommand(cmd, tt.sub)
			require.NotNil(t, sub)
			require.NotNil(t, sub.RunE)
			flag := sub.Flags().Lookup(tt.wantFlag)
			require.NotNil(t, flag)
		})
	}
}

func TestWorkflowRunCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "workflow run command has correct use and RunE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := WorkflowCommand()
			runCmd := findSubcommand(cmd, "run")
			require.NotNil(t, runCmd)
			require.Contains(t, runCmd.Use, "run")
			require.NotNil(t, runCmd.RunE)
		})
	}
}

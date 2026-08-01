package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipelineCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "pipeline command has correct use and subcommands"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := PipelineCommand()
			require.NotNil(t, cmd)
			require.Equal(t, "pipeline", cmd.Use)
			require.NotNil(t, cmd.Commands())
			names := map[string]bool{}
			for _, c := range cmd.Commands() {
				names[c.Name()] = true
			}
			for _, want := range []string{"apply", "list", "get", "export", "delete", "run", "runs"} {
				require.True(t, names[want], "missing subcommand %s", want)
			}
		})
	}
}

func TestPipelineListCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "pipeline list command has correct flags"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := PipelineCommand()
			listCmd, _, err := cmd.Find([]string{"list"})
			require.NoError(t, err)
			require.NotNil(t, listCmd.Flags().Lookup("output"))
		})
	}
}

func TestPipelineRunCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "pipeline run command has correct use and RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := PipelineCommand()
			runCmd, _, err := cmd.Find([]string{"run"})
			require.NoError(t, err)
			require.Equal(t, "run <name>", runCmd.Use)
			require.NotNil(t, runCmd.RunE)
			require.NotNil(t, runCmd.Flags().Lookup("event"))
		})
	}
}

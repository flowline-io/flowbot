package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "memo command has expected subcommands"},
		{name: "memo subcommands are wired"},
		{name: "memo create requires RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := MemoCommand()
			require.Equal(t, "memo", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "create")
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "update")
			require.Contains(t, subNames, "delete")
			require.Contains(t, subNames, "health")

			createCmd := findSubcommand(cmd, "create")
			require.NotNil(t, createCmd)
			require.NotNil(t, createCmd.RunE)
		})
	}
}

func TestMemoCreateRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		flagName string
	}{
		{name: "create has content flag", flagName: "content"},
		{name: "create has visibility flag", flagName: "visibility"},
		{name: "create has content shorthand", flagName: "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := findSubcommand(MemoCommand(), "create")
			require.NotNil(t, cmd)
			if tt.flagName == "c" {
				content := cmd.Flags().Lookup("content")
				require.NotNil(t, content)
				require.Equal(t, "c", content.Shorthand)
				return
			}
			require.NotNil(t, cmd.Flags().Lookup(tt.flagName))
		})
	}
}

func TestMemoListCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "list has limit flag"},
		{name: "list has output flag"},
		{name: "list has RunE handler"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := findSubcommand(MemoCommand(), "list")
			require.NotNil(t, cmd)
			require.NotNil(t, cmd.RunE)
			require.NotNil(t, cmd.Flags().Lookup("limit"))
			require.NotNil(t, cmd.Flags().Lookup("output"))
		})
	}
}

package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriliumCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "trilium command has expected subcommands"},
		{name: "trilium subcommands are wired"},
		{name: "trilium create requires RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := TriliumCommand()
			require.Equal(t, "trilium", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "create")
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "update")
			require.Contains(t, subNames, "delete")
			require.Contains(t, subNames, "search")
			require.Contains(t, subNames, "content")
			require.Contains(t, subNames, "health")

			createCmd := findSubcommand(cmd, "create")
			require.NotNil(t, createCmd)
			require.NotNil(t, createCmd.RunE)
		})
	}
}

func TestTriliumCreateRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		flagName string
	}{
		{name: "create has title flag", flagName: "title"},
		{name: "create has content flag", flagName: "content"},
		{name: "create has parent flag", flagName: "parent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := findSubcommand(TriliumCommand(), "create")
			require.NotNil(t, cmd)
			require.NotNil(t, cmd.Flags().Lookup(tt.flagName))
		})
	}
}

func TestTriliumListAndContentCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "list has limit and query flags"},
		{name: "content has get and set subcommands"},
		{name: "search requires query flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := TriliumCommand()
			listCmd := findSubcommand(root, "list")
			require.NotNil(t, listCmd)
			require.NotNil(t, listCmd.RunE)
			require.NotNil(t, listCmd.Flags().Lookup("limit"))
			require.NotNil(t, listCmd.Flags().Lookup("query"))

			contentCmd := findSubcommand(root, "content")
			require.NotNil(t, contentCmd)
			contentSubs := subcommandNames(contentCmd)
			require.Contains(t, contentSubs, "get")
			require.Contains(t, contentSubs, "set")

			searchCmd := findSubcommand(root, "search")
			require.NotNil(t, searchCmd)
			require.NotNil(t, searchCmd.Flags().Lookup("query"))
		})
	}
}

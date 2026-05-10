package command

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestReaderCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "reader command has correct use and subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ReaderCommand()

			require.Equal(t, "reader", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "create")
			require.Contains(t, subNames, "update")
			require.Contains(t, subNames, "refresh")
			require.Contains(t, subNames, "entries")
			require.Contains(t, subNames, "update-entries")
			require.Contains(t, subNames, "feed-entries")
		})
	}
}

func TestReaderCreateRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "reader create has required --url flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ReaderCommand()
			createCmd := findSubcommand(cmd, "create")
			require.NotNil(t, createCmd)

			url := createCmd.Flags().Lookup("url")
			require.NotNil(t, url)
			ann := url.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
			require.Contains(t, ann, "true")
		})
	}
}

func TestReaderUpdateEntriesCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "reader update-entries has required flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := ReaderCommand()
			updateCmd := findSubcommand(cmd, "update-entries")
			require.NotNil(t, updateCmd)
			require.NotNil(t, updateCmd.RunE)

			ids := updateCmd.Flags().Lookup("ids")
			require.NotNil(t, ids)
			ann := ids.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)

			status := updateCmd.Flags().Lookup("status")
			require.NotNil(t, status)
			ann = status.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
		})
	}
}

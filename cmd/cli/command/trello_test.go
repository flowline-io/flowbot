package command

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestTrelloCommand(t *testing.T) {
	t.Parallel()
	cmd := TrelloCommand()
	require.Equal(t, "trello", cmd.Use)
	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "board")
	require.Contains(t, subNames, "list")
	require.Contains(t, subNames, "card")
	require.Contains(t, subNames, "webhook")
	require.Contains(t, subNames, "health")
}

func TestTrelloDescDoesNotClaimDebugShorthand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		usePrefix string
		args      []string
	}{
		{name: "card create", usePrefix: "card create", args: []string{"trello", "card", "create", "--help"}},
		{name: "card update", usePrefix: "card update", args: []string{"trello", "card", "update", "--help"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := &cobra.Command{Use: "flowbot"}
			root.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
			root.AddCommand(TrelloCommand())

			trello := findSubcommand(root, "trello")
			require.NotNil(t, trello)
			var cmd *cobra.Command
			for _, sub := range trello.Commands() {
				if strings.HasPrefix(sub.Use, tt.usePrefix) {
					cmd = sub
					break
				}
			}
			require.NotNil(t, cmd)
			desc := cmd.Flags().Lookup("desc")
			require.NotNil(t, desc)
			require.Empty(t, desc.Shorthand)

			root.SetArgs(tt.args)
			require.NoError(t, root.Execute())
		})
	}
}

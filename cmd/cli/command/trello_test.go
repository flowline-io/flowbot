package command

import (
	"testing"

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

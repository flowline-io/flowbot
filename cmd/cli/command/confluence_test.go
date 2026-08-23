package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfluenceCommand(t *testing.T) {
	t.Parallel()
	cmd := ConfluenceCommand()
	require.Equal(t, "confluence", cmd.Use)
	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "space")
	require.Contains(t, subNames, "page")
	require.Contains(t, subNames, "health")
}

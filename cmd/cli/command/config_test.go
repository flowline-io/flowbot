package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigCommand(t *testing.T) {
	cmd := ConfigCommand()

	require.Equal(t, "config", cmd.Use)
	require.True(t, cmd.HasSubCommands())

	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "get")
	require.Contains(t, subNames, "set")
	require.Contains(t, subNames, "list")
}

func TestConfigGetCommand(t *testing.T) {
	cmd := ConfigCommand()
	getCmd := findSubcommand(cmd, "get")
	require.NotNil(t, getCmd)
	require.Contains(t, getCmd.Use, "get")
	require.NotNil(t, getCmd.RunE)
}

func TestConfigSetCommand(t *testing.T) {
	cmd := ConfigCommand()
	setCmd := findSubcommand(cmd, "set")
	require.NotNil(t, setCmd)
	require.Contains(t, setCmd.Use, "set")
	require.NotNil(t, setCmd.RunE)
}

func TestConfigListCommand(t *testing.T) {
	cmd := ConfigCommand()
	listCmd := findSubcommand(cmd, "list")
	require.NotNil(t, listCmd)
	require.Equal(t, "list", listCmd.Use)
	require.NotNil(t, listCmd.RunE)
}

func TestVersionCommand(t *testing.T) {
	cmd := VersionCommand("1.0.0")

	require.Equal(t, "version", cmd.Use)
	require.NotNil(t, cmd.RunE)
	require.Contains(t, cmd.Short, "Print version")
}

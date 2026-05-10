package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHubCommand(t *testing.T) {
	cmd := HubCommand()

	require.Equal(t, "hub", cmd.Use)
	require.True(t, cmd.HasSubCommands())

	subNames := subcommandNames(cmd)
	require.Contains(t, subNames, "apps")
	require.Contains(t, subNames, "capabilities")
	require.Contains(t, subNames, "health")
}

func TestHubAppsCommand(t *testing.T) {
	hubCmd := HubCommand()
	appsCmd := findSubcommand(hubCmd, "apps")
	require.NotNil(t, appsCmd)
	require.True(t, appsCmd.HasSubCommands())

	subNames := subcommandNames(appsCmd)
	require.Contains(t, subNames, "list")
	require.Contains(t, subNames, "status")
	require.Contains(t, subNames, "logs")
	require.Contains(t, subNames, "restart")
}

func TestHubAppsListCommand(t *testing.T) {
	hubCmd := HubCommand()
	appsCmd := findSubcommand(hubCmd, "apps")
	listCmd := findSubcommand(appsCmd, "list")
	require.NotNil(t, listCmd)
	require.NotNil(t, listCmd.RunE)

	output := listCmd.Flags().Lookup("output")
	require.NotNil(t, output)
	require.Equal(t, "output", output.Name)
	require.Equal(t, "o", output.Shorthand)
	defVal, _ := listCmd.Flags().GetString("output")
	require.Equal(t, "table", defVal)
}

func TestHubAppsStatusCommand(t *testing.T) {
	hubCmd := HubCommand()
	appsCmd := findSubcommand(hubCmd, "apps")
	statusCmd := findSubcommand(appsCmd, "status")
	require.NotNil(t, statusCmd)
	require.Contains(t, statusCmd.Use, "status")
	require.NotNil(t, statusCmd.RunE)
}

func TestHubAppsLogsCommand(t *testing.T) {
	hubCmd := HubCommand()
	appsCmd := findSubcommand(hubCmd, "apps")
	logsCmd := findSubcommand(appsCmd, "logs")
	require.NotNil(t, logsCmd)
	require.NotNil(t, logsCmd.RunE)

	tail := logsCmd.Flags().Lookup("tail")
	require.NotNil(t, tail)
	require.Equal(t, "tail", tail.Name)
	require.Equal(t, "n", tail.Shorthand)
	tailVal, _ := logsCmd.Flags().GetInt("tail")
	require.Equal(t, 100, tailVal)
}

func TestHubHealthCommand(t *testing.T) {
	hubCmd := HubCommand()
	healthCmd := findSubcommand(hubCmd, "health")
	require.NotNil(t, healthCmd)
	require.NotNil(t, healthCmd.RunE)

	output := healthCmd.Flags().Lookup("output")
	require.NotNil(t, output)
}

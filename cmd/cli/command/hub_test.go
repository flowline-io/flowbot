package command

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestHubCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "hub command has correct use and subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := HubCommand()

			require.Equal(t, "hub", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "apps")
			require.Contains(t, subNames, "capabilities")
			require.Contains(t, subNames, "health")
		})
	}
}

func TestHubAppsCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "hub apps has correct subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hubCmd := HubCommand()
			appsCmd := findSubcommand(hubCmd, "apps")
			require.NotNil(t, appsCmd)
			require.True(t, appsCmd.HasSubCommands())

			subNames := subcommandNames(appsCmd)
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "status")
			require.Contains(t, subNames, "logs")
			require.Contains(t, subNames, "restart")
		})
	}
}

func TestHubAppsListCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "hub apps list has correct flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hubCmd := HubCommand()
			appsCmd := findSubcommand(hubCmd, "apps")
			listCmd := findSubcommand(appsCmd, "list")
			require.NotNil(t, listCmd)
			require.NotNil(t, listCmd.RunE)

			root := NewRoot("test")
			list := findLeaf(t, root, "hub", "apps", "list")
			output := lookupFlag(list, "output")
			require.NotNil(t, output)
			require.Equal(t, "output", output.Name)
			require.Equal(t, "o", output.Shorthand)
			require.Equal(t, "table", output.DefValue)
		})
	}
}

func TestHubAppsStatusCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "hub apps status has correct use and RunE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hubCmd := HubCommand()
			appsCmd := findSubcommand(hubCmd, "apps")
			statusCmd := findSubcommand(appsCmd, "status")
			require.NotNil(t, statusCmd)
			require.Contains(t, statusCmd.Use, "status")
			require.NotNil(t, statusCmd.RunE)
		})
	}
}

func TestHubAppsLogsCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "hub apps logs has correct flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
		})
	}
}

func TestHubHealthCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "hub health has correct flags and RunE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := NewRoot("test")
			healthCmd := findLeaf(t, root, "hub", "health")
			require.NotNil(t, healthCmd.RunE)
			require.NotNil(t, lookupFlag(healthCmd, "output"))
		})
	}
}

func findLeaf(t *testing.T, cmd *cobra.Command, names ...string) *cobra.Command {
	t.Helper()
	cur := cmd
	for _, name := range names {
		cur = findSubcommand(cur, name)
		require.NotNilf(t, cur, "missing %s", name)
	}
	return cur
}

func lookupFlag(cmd *cobra.Command, name string) *pflag.Flag {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.PersistentFlags().Lookup(name); f != nil {
			return f
		}
		if f := c.Flags().Lookup(name); f != nil {
			return f
		}
	}
	return nil
}

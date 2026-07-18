package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDevopsCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "devops command has expected subcommands"},
		{name: "devops provider groups are wired"},
		{name: "devops beszel has nested commands"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := DevopsCommand()
			require.Equal(t, "devops", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "status")
			require.Contains(t, subNames, "beszel")
			require.Contains(t, subNames, "uptimekuma")
			require.Contains(t, subNames, "traefik")
			require.Contains(t, subNames, "grafana")
			require.Contains(t, subNames, "wakapi")
			require.Contains(t, subNames, "dozzle")
			require.Contains(t, subNames, "netalertx")

			beszel := findSubcommand(cmd, "beszel")
			require.NotNil(t, beszel)
			require.Contains(t, subcommandNames(beszel), "systems")
			require.Contains(t, subcommandNames(beszel), "get")

			grafana := findSubcommand(cmd, "grafana")
			require.NotNil(t, grafana)
			require.Contains(t, subcommandNames(grafana), "query")
		})
	}
}

func TestDevopsRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		parent   string
		child    string
		flagName string
	}{
		{name: "beszel get requires id", parent: "beszel", child: "get", flagName: "id"},
		{name: "grafana query requires backend", parent: "grafana", child: "query", flagName: "backend"},
		{name: "grafana query requires expr", parent: "grafana", child: "query", flagName: "expr"},
		{name: "netalertx search requires query", parent: "netalertx", child: "search", flagName: "query"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := DevopsCommand()
			cmd := findSubcommand(root, tt.parent)
			require.NotNil(t, cmd)
			if tt.child != "" {
				cmd = findSubcommand(cmd, tt.child)
				require.NotNil(t, cmd)
			}
			require.NotNil(t, cmd.Flags().Lookup(tt.flagName))
		})
	}
}

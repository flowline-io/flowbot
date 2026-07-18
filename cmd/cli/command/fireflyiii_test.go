package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFireflyiiiCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "fireflyiii command has expected subcommands"},
		{name: "fireflyiii subcommands are wired"},
		{name: "fireflyiii create requires RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := FireflyiiiCommand()
			require.Equal(t, "fireflyiii", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "create")
			require.Contains(t, subNames, "about")
			require.Contains(t, subNames, "user")
			require.Contains(t, subNames, "health")

			createCmd := findSubcommand(cmd, "create")
			require.NotNil(t, createCmd)
			require.NotNil(t, createCmd.RunE)
		})
	}
}

func TestFireflyiiiCreateRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		flagName string
	}{
		{name: "create has type flag", flagName: "type"},
		{name: "create has date flag", flagName: "date"},
		{name: "create has amount flag", flagName: "amount"},
		{name: "create has description flag", flagName: "description"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := findSubcommand(FireflyiiiCommand(), "create")
			require.NotNil(t, cmd)
			require.NotNil(t, cmd.Flags().Lookup(tt.flagName))
		})
	}
}

func TestFireflyiiiAboutAndHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "about has output flag"},
		{name: "user has output flag"},
		{name: "health has RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := FireflyiiiCommand()
			aboutCmd := findSubcommand(root, "about")
			require.NotNil(t, aboutCmd)
			require.NotNil(t, aboutCmd.RunE)
			require.NotNil(t, aboutCmd.Flags().Lookup("output"))

			userCmd := findSubcommand(root, "user")
			require.NotNil(t, userCmd)
			require.NotNil(t, userCmd.Flags().Lookup("output"))

			healthCmd := findSubcommand(root, "health")
			require.NotNil(t, healthCmd)
			require.NotNil(t, healthCmd.RunE)
		})
	}
}

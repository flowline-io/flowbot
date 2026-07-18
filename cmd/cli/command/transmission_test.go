package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransmissionCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "transmission command has expected subcommands"},
		{name: "transmission subcommands are wired"},
		{name: "transmission add requires RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := TransmissionCommand()
			require.Equal(t, "transmission", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "add")
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "stop")
			require.Contains(t, subNames, "remove")
			require.Contains(t, subNames, "health")

			addCmd := findSubcommand(cmd, "add")
			require.NotNil(t, addCmd)
			require.NotNil(t, addCmd.RunE)
		})
	}
}

func TestTransmissionAddRequiredFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		flagName string
	}{
		{name: "add has url flag", flagName: "url"},
		{name: "stop has ids flag", flagName: "ids"},
		{name: "remove has ids flag", flagName: "ids"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := TransmissionCommand()
			var cmdName string
			switch tt.flagName {
			case "url":
				cmdName = "add"
			case "ids":
				if tt.name == "stop has ids flag" {
					cmdName = "stop"
				} else {
					cmdName = "remove"
				}
			}
			cmd := findSubcommand(root, cmdName)
			require.NotNil(t, cmd)
			require.NotNil(t, cmd.Flags().Lookup(tt.flagName))
		})
	}
}

func TestTransmissionListAndHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "list has output flag"},
		{name: "health has RunE"},
		{name: "stop has RunE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := TransmissionCommand()
			listCmd := findSubcommand(root, "list")
			require.NotNil(t, listCmd)
			require.NotNil(t, listCmd.RunE)
			require.NotNil(t, listCmd.Flags().Lookup("output"))

			healthCmd := findSubcommand(root, "health")
			require.NotNil(t, healthCmd)
			require.NotNil(t, healthCmd.RunE)

			stopCmd := findSubcommand(root, "stop")
			require.NotNil(t, stopCmd)
			require.NotNil(t, stopCmd.RunE)
		})
	}
}

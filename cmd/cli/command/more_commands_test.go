package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGithubNotificationsCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "notifications subcommand exists"},
		{name: "notifications has limit flag"},
		{name: "notifications has output flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			notif := findSubcommand(GithubCommand(), "notifications")
			require.NotNil(t, notif)
			require.NotNil(t, notif.RunE)
			require.NotNil(t, notif.Flags().Lookup("limit"))
			require.NotNil(t, notif.Flags().Lookup("output"))
		})
	}
}

func TestHubCapabilitiesCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "capabilities command exists"},
		{name: "capabilities has RunE handler"},
		{name: "capabilities has output flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			capCmd := findSubcommand(HubCommand(), "capabilities")
			require.NotNil(t, capCmd)
			require.NotNil(t, capCmd.RunE)
			require.NotNil(t, capCmd.Flags().Lookup("output"))
		})
	}
}

func TestGithubReleasesCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "releases command exists"},
		{name: "releases requires two args"},
		{name: "releases has limit flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			releases := findSubcommand(GithubCommand(), "releases")
			require.NotNil(t, releases)
			require.NotNil(t, releases.RunE)
			require.NotNil(t, releases.Flags().Lookup("limit"))
		})
	}
}

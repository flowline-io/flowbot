package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "chat root command has correct use, short, and version"},
		{name: "chat root command exposes profile flag"},
		{name: "chat root command exposes server-url flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := NewCommand()

			switch tt.name {
			case "chat root command has correct use, short, and version":
				require.Equal(t, "flowbot-chat", cmd.Use)
				require.Equal(t, "Chat with the Flowbot Chat Agent in your terminal", cmd.Short)
				require.NotEmpty(t, cmd.Version, "version must be set so --version prints it")
			case "chat root command exposes profile flag":
				profile := cmd.PersistentFlags().Lookup("profile")
				require.NotNil(t, profile)
				require.Equal(t, "string", profile.Value.Type())
			case "chat root command exposes server-url flag":
				serverURL := cmd.PersistentFlags().Lookup("server-url")
				require.NotNil(t, serverURL)
				require.Equal(t, "string", serverURL.Value.Type())
			}
		})
	}
}

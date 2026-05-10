package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "login command has correct use, description, and flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := LoginCommand()

			require.Equal(t, "login", cmd.Use)
			require.Contains(t, cmd.Short, "Save access token")
			require.NotEmpty(t, cmd.Long)
			require.NotNil(t, cmd.RunE)

			token := cmd.Flags().Lookup("token")
			require.NotNil(t, token)
			require.Equal(t, "token", token.Name)
			require.Equal(t, "t", token.Shorthand)
		})
	}
}

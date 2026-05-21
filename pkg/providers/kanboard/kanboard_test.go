package kanboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKanboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		username string
		password string
		wantErr  bool
	}{
		{
			name:     "creates client with valid endpoint",
			endpoint: "http://localhost:8080/jsonrpc.php",
			username: "admin",
			password: "secret",
			wantErr:  false,
		},
		{
			name:     "creates client with https endpoint",
			endpoint: "https://kanboard.example.com/jsonrpc.php",
			username: "user",
			password: "pass",
			wantErr:  false,
		},
		{
			name:     "creates client with empty credentials",
			endpoint: "http://localhost:8080/jsonrpc.php",
			username: "",
			password: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := NewKanboard(tt.endpoint, tt.username, tt.password)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, v, "NewKanboard returned nil struct")
			assert.NotNil(t, v.channel, "NewKanboard channel should not be nil")
			assert.NotNil(t, v.c, "NewKanboard rpc client should not be nil")
		})
	}
}

package transmission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransmission(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "creates client with valid http endpoint",
			endpoint: "http://localhost:9091/transmission/rpc",
			wantErr:  false,
		},
		{
			name:     "creates client with valid https endpoint",
			endpoint: "https://transmission.example.com/transmission/rpc",
			wantErr:  false,
		},
		{
			name:     "fails with invalid endpoint",
			endpoint: "://invalid-url",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := NewTransmission(tt.endpoint)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, v, "NewTransmission returned nil struct")
			assert.NotNil(t, v.c, "NewTransmission rpc client should not be nil")
		})
	}
}

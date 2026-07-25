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
		wantNil  bool
		wantErr  bool
	}{
		{
			name:     "creates client with valid http endpoint",
			endpoint: "http://localhost:9091/transmission/rpc",
		},
		{
			name:     "creates client with valid https endpoint",
			endpoint: "https://transmission.example.com/transmission/rpc",
		},
		{
			name:     "empty endpoint returns nil",
			endpoint: "",
			wantNil:  true,
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
			if tt.wantNil {
				assert.Nil(t, v)
				return
			}
			require.NotNil(t, v, "NewTransmission returned nil struct")
			assert.NotNil(t, v.c, "NewTransmission rpc client should not be nil")
		})
	}
}

func TestIsValidRedirect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "https ok", raw: "https://tracker.example.com/file.torrent", want: true},
		{name: "http ok", raw: "http://cdn.example.com/a.torrent", want: true},
		{name: "loopback http ok for homelab", raw: "http://127.0.0.1:8080/x.torrent", want: true},
		{name: "magnet rejected here", raw: "magnet:?xt=urn:btih:abc", want: false},
		{name: "file scheme blocked", raw: "file:///etc/passwd", want: false},
		{name: "empty blocked", raw: "", want: false},
		{name: "no host blocked", raw: "https:///nohost", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isValidRedirect(tt.raw))
		})
	}
}

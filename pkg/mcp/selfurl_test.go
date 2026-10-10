package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSelfMCP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rawURL  string
		listen  string
		apiPath string
		want    bool
	}{
		{name: "loopback mcp", rawURL: "http://127.0.0.1:6060/mcp", listen: "127.0.0.1:6060", want: true},
		{name: "localhost", rawURL: "http://localhost:6060/mcp", listen: "127.0.0.1:6060", want: true},
		{name: "docker internal", rawURL: "http://host.docker.internal:6060/mcp", listen: "0.0.0.0:6060", want: true},
		{name: "wrong path", rawURL: "http://127.0.0.1:6060/hub/health", listen: "127.0.0.1:6060", want: false},
		{name: "wrong port", rawURL: "http://127.0.0.1:7070/mcp", listen: "127.0.0.1:6060", want: false},
		{name: "other host wildcard listen", rawURL: "http://example.com:6060/mcp", listen: ":6060", want: false},
		{name: "api path prefix", rawURL: "http://127.0.0.1:6060/v1/mcp", listen: "127.0.0.1:6060", apiPath: "/v1", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsSelfMCP(tt.rawURL, tt.listen, tt.apiPath))
		})
	}
}

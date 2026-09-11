package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestDeduplicateCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []homelab.AppCapability
		want []string
	}{
		{
			name: "keeps unique capabilities",
			in: []homelab.AppCapability{
				{Capability: "gitea"},
				{Capability: "memos"},
			},
			want: []string{"gitea", "memos"},
		},
		{
			name: "drops duplicate capability ids",
			in: []homelab.AppCapability{
				{Capability: "gitea"},
				{Capability: "gitea"},
				{Capability: "memos"},
			},
			want: []string{"gitea", "memos"},
		},
		{
			name: "empty input",
			in:   nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deduplicateCapabilities(tt.in)
			var ids []string
			for _, c := range got {
				ids = append(ids, c.Capability)
			}
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestAuthTypeLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		auth *homelab.AuthInfo
		want string
	}{
		{name: "nil auth is unknown", auth: nil, want: "unknown"},
		{name: "oauth2 auth", auth: &homelab.AuthInfo{Type: homelab.AuthOAuth2}, want: "oauth2"},
		{name: "api token auth", auth: &homelab.AuthInfo{Type: homelab.AuthAPIToken}, want: "api_token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, authTypeLabel(tt.auth))
		})
	}
}

func TestResolveHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		port homelab.PortMapping
		want string
	}{
		{
			name: "explicit host",
			port: homelab.PortMapping{
				Protocol: "tcp",
				Host:     "192.168.1.10",
				HostPort: "3000",
			},
			want: "192.168.1.10",
		},
		{
			name: "missing host falls back to localhost when host port set",
			port: homelab.PortMapping{
				Protocol: "tcp",
				HostPort: "8080",
			},
			want: "localhost",
		},
		{
			name: "no host and no host port yields empty host",
			port: homelab.PortMapping{
				Protocol:  "tcp",
				Container: "80",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveHost(tt.port))
		})
	}
}

func TestResolveTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		app  homelab.App
		want []string
	}{
		{
			name: "tcp published ports produce http and https targets",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "tcp", Host: "127.0.0.1", HostPort: "3000"},
				},
			},
			want: []string{"http://127.0.0.1:3000", "https://127.0.0.1:3000"},
		},
		{
			name: "udp ports are skipped",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "udp", Host: "127.0.0.1", HostPort: "53"},
				},
			},
			want: nil,
		},
		{
			name: "ports without published host port are skipped",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "tcp", Container: "80"},
				},
			},
			want: nil,
		},
		{
			name: "localhost used when host omitted but host port present",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "tcp", HostPort: "8080", Container: "80"},
				},
			},
			want: []string{"http://localhost:8080", "https://localhost:8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &Engine{}
			assert.Equal(t, tt.want, e.resolveTargets(tt.app))
		})
	}
}

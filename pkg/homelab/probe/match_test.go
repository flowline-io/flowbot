package probe

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

func TestMatchHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		headers http.Header
		key     string
		pattern string
		want    bool
	}{
		{
			name:    "header contains pattern",
			headers: http.Header{"Server": []string{"nginx/1.25"}},
			key:     "Server",
			pattern: "nginx",
			want:    true,
		},
		{
			name:    "empty pattern matches present header",
			headers: http.Header{"X-App": []string{"flowbot"}},
			key:     "X-App",
			pattern: "",
			want:    true,
		},
		{
			name:    "missing header does not match",
			headers: http.Header{},
			key:     "Server",
			pattern: "nginx",
			want:    false,
		},
		{
			name:    "pattern mismatch",
			headers: http.Header{"Server": []string{"apache"}},
			key:     "Server",
			pattern: "nginx",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, matchHeader(tt.headers, tt.key, tt.pattern))
		})
	}
}

func TestMatchTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		pattern string
		want    bool
	}{
		{
			name:    "title contains pattern",
			body:    "<html><head><title>Gitea</title></head></html>",
			pattern: "Gitea",
			want:    true,
		},
		{
			name:    "empty pattern does not match",
			body:    "<title>Gitea</title>",
			pattern: "",
			want:    false,
		},
		{
			name:    "missing title does not match",
			body:    "<html><body>no title</body></html>",
			pattern: "Gitea",
			want:    false,
		},
		{
			name:    "title mismatch",
			body:    "<title>GitLab</title>",
			pattern: "Gitea",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, matchTitle([]byte(tt.body), tt.pattern))
		})
	}
}

func TestMatchBodyKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		key  string
		want bool
	}{
		{
			name: "json key present",
			body: `{"version":"1.0","status":"ok"}`,
			key:  "version",
			want: true,
		},
		{
			name: "empty key does not match",
			body: `{"version":"1.0"}`,
			key:  "",
			want: false,
		},
		{
			name: "missing key does not match",
			body: `{"status":"ok"}`,
			key:  "version",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, matchBodyKey([]byte(tt.body), tt.key))
		})
	}
}

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

func TestResolveHostAndHostPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		strategy     string
		port         homelab.PortMapping
		wantHost     string
		wantHostPort string
	}{
		{
			name:     "explicit host and host port",
			strategy: "host",
			port: homelab.PortMapping{
				Protocol:  "tcp",
				Host:      "192.168.1.10",
				HostPort:  "3000",
				Container: "80",
			},
			wantHost:     "192.168.1.10",
			wantHostPort: "3000",
		},
		{
			name:     "missing host falls back to localhost when host port set",
			strategy: "host",
			port: homelab.PortMapping{
				Protocol: "tcp",
				HostPort: "8080",
			},
			wantHost:     "localhost",
			wantHostPort: "8080",
		},
		{
			name:     "container strategy prefers container port",
			strategy: "container",
			port: homelab.PortMapping{
				Protocol:  "tcp",
				Host:      "127.0.0.1",
				HostPort:  "3000",
				Container: "80",
			},
			wantHost:     "127.0.0.1",
			wantHostPort: "80",
		},
		{
			name:     "both strategy prefers host port",
			strategy: "both",
			port: homelab.PortMapping{
				Protocol:  "tcp",
				Host:      "127.0.0.1",
				HostPort:  "3000",
				Container: "80",
			},
			wantHost:     "127.0.0.1",
			wantHostPort: "3000",
		},
		{
			name:     "no host and no host port yields empty host",
			strategy: "host",
			port: homelab.PortMapping{
				Protocol:  "tcp",
				Container: "80",
			},
			wantHost:     "",
			wantHostPort: "80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &Engine{config: homelab.DiscoveryConfig{ProbePortStrategy: tt.strategy}}
			assert.Equal(t, tt.wantHost, e.resolveHost(tt.port))
			assert.Equal(t, tt.wantHostPort, e.resolveHostPort(tt.port))
		})
	}
}

func TestResolveTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy string
		app      homelab.App
		want     []string
	}{
		{
			name:     "tcp ports produce http and https targets",
			strategy: "host",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "tcp", Host: "127.0.0.1", HostPort: "3000"},
				},
			},
			want: []string{"http://127.0.0.1:3000", "https://127.0.0.1:3000"},
		},
		{
			name:     "udp ports are skipped",
			strategy: "host",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "udp", Host: "127.0.0.1", HostPort: "53"},
				},
			},
			want: nil,
		},
		{
			name:     "ports without host info are skipped",
			strategy: "host",
			app: homelab.App{
				Ports: []homelab.PortMapping{
					{Protocol: "tcp", Container: "80"},
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := &Engine{config: homelab.DiscoveryConfig{ProbePortStrategy: tt.strategy}}
			assert.Equal(t, tt.want, e.resolveTargets(tt.app))
		})
	}
}

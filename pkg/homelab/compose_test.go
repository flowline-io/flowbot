package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCompose_ValidDocument(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		data             []byte
		expectedCount    int
		expectedNetworks []string
		expectedLabels   map[string]string
	}{
		{
			name: "valid compose document with all fields",
			data: []byte(`
services:
  web:
    image: archivebox/archivebox:latest
    container_name: archivebox
    ports:
      - "8080:8000/tcp"
    labels:
      flowbot.capability: archive
      flowbot.env: production
networks:
  proxy: {}
  backend: {}
`),
			expectedCount:    1,
			expectedNetworks: []string{"proxy", "backend"},
			expectedLabels:   map[string]string{"flowbot.capability": "archive", "flowbot.env": "production"},
		},
		{
			name: "service with multiple ports",
			data: []byte(`
services:
  web:
    image: nginx:latest
    ports:
      - "80:80/tcp"
      - "443:443/tcp"
`),
			expectedCount:    1,
			expectedNetworks: nil,
			expectedLabels:   map[string]string{},
		},
		{
			name: "service with no networks or labels",
			data: []byte(`
services:
  db:
    image: postgres:15
    container_name: postgres
`),
			expectedCount:    1,
			expectedNetworks: nil,
			expectedLabels:   map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			services, networks, _, labels, err := ParseCompose(tt.data)
			require.NoError(t, err)
			require.Len(t, services, tt.expectedCount)
			assert.ElementsMatch(t, tt.expectedNetworks, networks)
			assert.Equal(t, tt.expectedLabels, labels)
		})
	}
}

func TestParseCompose_InvalidYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
	}{
		{name: "invalid YAML content", data: []byte(`services: {{{invalid`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, _, err := ParseCompose(tt.data)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parse compose")
		})
	}
}

func TestParseCompose_EmptyDocument(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty document", data: []byte(``)},
		{name: "whitespace only document", data: []byte("\n\n   \n")},
		{name: "comments only document", data: []byte("# this is a comment\n# another comment\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			services, networks, ports, labels, err := ParseCompose(tt.data)
			require.NoError(t, err)
			assert.Empty(t, services)
			assert.Empty(t, networks)
			assert.Empty(t, ports)
			assert.Empty(t, labels)
		})
	}
}

func TestParseCompose_NoServices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		data             []byte
		expectedNetworks []string
	}{
		{
			name:             "networks only, no services",
			data:             []byte("networks:\n  proxy: {}\n"),
			expectedNetworks: []string{"proxy"},
		},
		{
			name:             "networks and volumes, no services",
			data:             []byte("networks:\n  proxy: {}\nvolumes:\n  data: {}\n"),
			expectedNetworks: []string{"proxy"},
		},
		{
			name:             "empty services map",
			data:             []byte("services: {}\nnetworks:\n  backend: {}\n"),
			expectedNetworks: []string{"backend"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			services, networks, ports, labels, err := ParseCompose(tt.data)
			require.NoError(t, err)
			assert.Empty(t, services)
			assert.ElementsMatch(t, tt.expectedNetworks, networks)
			assert.Empty(t, ports)
			assert.Empty(t, labels)
		})
	}
}

func TestParseCompose_MultipleServices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		data         []byte
		expectedLen  int
		expectedImgs map[string]string
	}{
		{
			name: "two services",
			data: []byte(`
services:
  web:
    image: nginx:latest
    ports:
      - "80:80/tcp"
  db:
    image: postgres:15
    container_name: postgres
`),
			expectedLen:  2,
			expectedImgs: map[string]string{"web": "nginx:latest", "db": "postgres:15"},
		},
		{
			name: "three services with different images",
			data: []byte(`
services:
  frontend:
    image: alpine:3.19
  api:
    image: golang:1.22
  cache:
    image: redis:7
`),
			expectedLen:  3,
			expectedImgs: map[string]string{"frontend": "alpine:3.19", "api": "golang:1.22", "cache": "redis:7"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			services, _, _, _, err := ParseCompose(tt.data)
			require.NoError(t, err)
			require.Len(t, services, tt.expectedLen)
			for _, s := range services {
				expectedImg, ok := tt.expectedImgs[s.Name]
				require.True(t, ok, "unexpected service %s", s.Name)
				assert.Equal(t, expectedImg, s.Image)
			}
		})
	}
}

func TestParseCompose_PortMapFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		data              []byte
		expectedCount     int
		expectedHost      string
		expectedHostPort  string
		expectedContainer string
		expectedProtocol  string
	}{
		{
			name:              "port map format",
			data:              []byte("services:\n  web:\n    image: test:latest\n    ports:\n      - published: 8080\n        target: 3000\n        protocol: udp\n"),
			expectedCount:     1,
			expectedHostPort:  "8080",
			expectedContainer: "3000",
			expectedProtocol:  "udp",
		},
		{
			name:              "port map with host_ip",
			data:              []byte("services:\n  web:\n    image: test:latest\n    ports:\n      - host_ip: 127.0.0.1\n        published: 9090\n        target: 4000\n        protocol: tcp\n"),
			expectedCount:     1,
			expectedHost:      "127.0.0.1",
			expectedHostPort:  "9090",
			expectedContainer: "4000",
			expectedProtocol:  "tcp",
		},
		{
			name:              "port map without protocol defaults to tcp",
			data:              []byte("services:\n  web:\n    image: test:latest\n    ports:\n      - published: 3000\n        target: 3000\n"),
			expectedCount:     1,
			expectedHostPort:  "3000",
			expectedContainer: "3000",
			expectedProtocol:  "tcp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, ports, _, err := ParseCompose(tt.data)
			require.NoError(t, err)
			require.Len(t, ports, tt.expectedCount)
			assert.Equal(t, tt.expectedHost, ports[0].Host)
			assert.Equal(t, tt.expectedHostPort, ports[0].HostPort)
			assert.Equal(t, tt.expectedContainer, ports[0].Container)
			assert.Equal(t, tt.expectedProtocol, ports[0].Protocol)
		})
	}
}

func TestParseCompose_PortStringFormats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		yaml      string
		hostPort  string
		container string
		host      string
		protocol  string
	}{
		{
			name:      "container only",
			yaml:      `services: { app: { image: x, ports: ["3000"] } }`,
			container: "3000",
			protocol:  "tcp",
		},
		{
			name:      "host:container",
			yaml:      `services: { app: { image: x, ports: ["8080:3000"] } }`,
			hostPort:  "8080",
			container: "3000",
			protocol:  "tcp",
		},
		{
			name:      "host:hostport:container with protocol",
			yaml:      `services: { app: { image: x, ports: ["127.0.0.1:8080:3000/udp"] } }`,
			host:      "127.0.0.1",
			hostPort:  "8080",
			container: "3000",
			protocol:  "udp",
		},
		{
			name:      "default protocol when omitted",
			yaml:      `services: { app: { image: x, ports: ["8080:3000"] } }`,
			hostPort:  "8080",
			container: "3000",
			protocol:  "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, ports, _, err := ParseCompose([]byte(tt.yaml))
			require.NoError(t, err)
			require.Len(t, ports, 1)
			assert.Equal(t, tt.hostPort, ports[0].HostPort)
			assert.Equal(t, tt.container, ports[0].Container)
			assert.Equal(t, tt.host, ports[0].Host)
			assert.Equal(t, tt.protocol, ports[0].Protocol)
		})
	}
}

func TestParseCompose_LabelsListFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		data           []byte
		expectedKeys   []string
		expectedValues map[string]string
	}{
		{
			name: "labels in list format",
			data: []byte(`
services:
  web:
    image: traefik:latest
    labels:
      - "traefik.enable=true"
      - "homepage.group=Server"
      - "homepage.name=Traefik"
      - keyonly
`),
			expectedKeys: []string{"traefik.enable", "homepage.group", "homepage.name", "keyonly"},
			expectedValues: map[string]string{
				"traefik.enable": "true",
				"homepage.group": "Server",
				"homepage.name":  "Traefik",
				"keyonly":        "",
			},
		},
		{
			name: "list labels with empty entry",
			data: []byte(`
services:
  web:
    image: test:latest
    labels:
      - ""
      - "key=val"
`),
			expectedKeys:   []string{"", "key"},
			expectedValues: map[string]string{"": "", "key": "val"},
		},
		{
			name: "list labels with equals in value",
			data: []byte(`
services:
  web:
    image: test:latest
    labels:
      - "key=val=with=equals"
      - "normal=value"
`),
			expectedKeys:   []string{"key", "normal"},
			expectedValues: map[string]string{"key": "val=with=equals", "normal": "value"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, labels, err := ParseCompose(tt.data)
			require.NoError(t, err)
			for _, key := range tt.expectedKeys {
				assert.Equal(t, tt.expectedValues[key], labels[key])
			}
		})
	}
}

func TestNormalizeLabels_MapFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		raw      map[string]any
		expected map[string]string
	}{
		{
			name:     "map format labels",
			raw:      map[string]any{"env": "prod", "tier": 1},
			expected: map[string]string{"env": "prod", "tier": "1"},
		},
		{
			name:     "empty map",
			raw:      map[string]any{},
			expected: map[string]string{},
		},
		{
			name:     "map with boolean value",
			raw:      map[string]any{"enabled": true, "debug": false},
			expected: map[string]string{"enabled": "true", "debug": "false"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := normalizeLabels(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeLabels_ListFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		raw      []any
		expected map[string]string
	}{
		{
			name:     "list format labels",
			raw:      []any{"key1=val1", "key2=val2 with spaces", "keyonly"},
			expected: map[string]string{"key1": "val1", "key2": "val2 with spaces", "keyonly": ""},
		},
		{
			name:     "empty list",
			raw:      []any{},
			expected: map[string]string{},
		},
		{
			name:     "list with whitespace around equals",
			raw:      []any{" key = val ", " k2=v2"},
			expected: map[string]string{"key": "val", "k2": "v2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := normalizeLabels(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeLabels_Nil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		raw      any
		expected map[string]string
	}{
		{name: "nil input returns empty map", raw: nil, expected: map[string]string{}},
		{name: "empty map as any", raw: map[string]any{}, expected: map[string]string{}},
		{name: "empty list as any", raw: []any{}, expected: map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := normalizeLabels(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCompose_LabelsAcrossServices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		data           []byte
		expectedLabels map[string]string
	}{
		{
			name: "labels merged across services",
			data: []byte(`
services:
  web:
    image: nginx:latest
    labels:
      env: prod
  worker:
    image: worker:latest
    labels:
      env: prod
      tier: backend
`),
			expectedLabels: map[string]string{"env": "prod", "tier": "backend"},
		},
		{
			name: "conflicting label, last service wins",
			data: []byte(`
services:
  frontend:
    image: alpine:latest
    labels:
      env: staging
  backend:
    image: golang:latest
    labels:
      env: production
`),
			expectedLabels: map[string]string{"env": "production"},
		},
		{
			name: "one service with labels, another without",
			data: []byte(`
services:
  db:
    image: postgres:15
  app:
    image: app:latest
    labels:
      env: prod
      tier: frontend
`),
			expectedLabels: map[string]string{"env": "prod", "tier": "frontend"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, labels, err := ParseCompose(tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLabels, labels)
		})
	}
}

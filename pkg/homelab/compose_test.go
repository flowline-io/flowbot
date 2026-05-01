package homelab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCompose_ValidDocument(t *testing.T) {
	data := []byte(`
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
`)
	services, networks, ports, labels, err := ParseCompose(data)
	require.NoError(t, err)
	require.Len(t, services, 1)

	assert.Equal(t, "web", services[0].Name)
	assert.Equal(t, "archivebox/archivebox:latest", services[0].Image)
	assert.Equal(t, "archivebox", services[0].Container)
	require.Len(t, services[0].Ports, 1)
	assert.Equal(t, "8080", services[0].Ports[0].HostPort)
	assert.Equal(t, "8000", services[0].Ports[0].Container)
	assert.Equal(t, "tcp", services[0].Ports[0].Protocol)

	require.Len(t, ports, 1)
	assert.Equal(t, "8080", ports[0].HostPort)

	assert.ElementsMatch(t, []string{"proxy", "backend"}, networks)

	assert.Equal(t, "archive", labels["flowbot.capability"])
	assert.Equal(t, "production", labels["flowbot.env"])
}

func TestParseCompose_InvalidYAML(t *testing.T) {
	data := []byte(`services: {{{invalid`)
	_, _, _, _, err := ParseCompose(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse compose")
}

func TestParseCompose_EmptyDocument(t *testing.T) {
	data := []byte(``)
	services, networks, ports, labels, err := ParseCompose(data)
	require.NoError(t, err)
	assert.Empty(t, services)
	assert.Empty(t, networks)
	assert.Empty(t, ports)
	assert.Empty(t, labels)
}

func TestParseCompose_NoServices(t *testing.T) {
	data := []byte(`
networks:
  proxy: {}
`)
	services, networks, ports, labels, err := ParseCompose(data)
	require.NoError(t, err)
	assert.Empty(t, services)
	assert.Equal(t, []string{"proxy"}, networks)
	assert.Empty(t, ports)
	assert.Empty(t, labels)
	_ = networks
	_ = labels
}

func TestParseCompose_MultipleServices(t *testing.T) {
	data := []byte(`
services:
  web:
    image: nginx:latest
    ports:
      - "80:80/tcp"
  db:
    image: postgres:15
    container_name: postgres
`)
	services, _, ports, _, err := ParseCompose(data)
	require.NoError(t, err)
	require.Len(t, services, 2)

	svcMap := make(map[string]ComposeService)
	for _, s := range services {
		svcMap[s.Name] = s
	}

	assert.Equal(t, "nginx:latest", svcMap["web"].Image)
	assert.Equal(t, "postgres:15", svcMap["db"].Image)
	assert.Equal(t, "postgres", svcMap["db"].Container)
	assert.Empty(t, svcMap["web"].Container)

	require.Len(t, ports, 1)
	assert.Equal(t, "80", ports[0].HostPort)
}

func TestParseCompose_PortMapFormat(t *testing.T) {
	data := []byte(`
services:
  web:
    image: test:latest
    ports:
      - published: 8080
        target: 3000
        protocol: udp
`)
	_, _, ports, _, err := ParseCompose(data)
	require.NoError(t, err)
	require.Len(t, ports, 1)
	assert.Equal(t, "8080", ports[0].HostPort)
	assert.Equal(t, "3000", ports[0].Container)
	assert.Equal(t, "udp", ports[0].Protocol)
}

func TestParseCompose_PortStringFormats(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		hostPort string
		container string
		host     string
		protocol string
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, ports, _, err := ParseCompose([]byte(tc.yaml))
			require.NoError(t, err)
			require.Len(t, ports, 1)
			assert.Equal(t, tc.hostPort, ports[0].HostPort)
			assert.Equal(t, tc.container, ports[0].Container)
			assert.Equal(t, tc.host, ports[0].Host)
			assert.Equal(t, tc.protocol, ports[0].Protocol)
		})
	}
}

func TestParseCompose_LabelsAcrossServices(t *testing.T) {
	data := []byte(`
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
`)
	_, _, _, labels, err := ParseCompose(data)
	require.NoError(t, err)
	assert.Equal(t, "prod", labels["env"])
	assert.Equal(t, "backend", labels["tier"])
}

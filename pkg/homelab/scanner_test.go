package homelab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerDiscoversComposeApps(t *testing.T) {
	root := t.TempDir()
	appsDir := filepath.Join(root, "apps")
	require.NoError(t, os.MkdirAll(filepath.Join(appsDir, "archivebox"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, "archivebox", "docker-compose.yaml"), []byte(`
services:
  web:
    image: archivebox/archivebox:latest
    container_name: archivebox
    ports:
      - "8080:8000/tcp"
    labels:
      flowbot.capability: archive
networks:
  proxy: {}
`), 0o644))

	apps, err := NewScanner(Config{AppsDir: appsDir}).Scan()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, "archivebox", apps[0].Name)
	require.Equal(t, "archivebox/archivebox:latest", apps[0].Services[0].Image)
	require.Equal(t, "8080", apps[0].Ports[0].HostPort)
	require.Equal(t, "8000", apps[0].Ports[0].Container)
	require.Equal(t, "archive", apps[0].Labels["flowbot.capability"])
	require.Len(t, apps[0].Capabilities, 1)
	assert.Equal(t, CapArchive, apps[0].Capabilities[0].Capability)
	assert.Equal(t, "archive", apps[0].Capabilities[0].Backend)
}

func TestScannerAppliesAllowlist(t *testing.T) {
	root := t.TempDir()
	appsDir := filepath.Join(root, "apps")
	writeComposeApp(t, appsDir, "archivebox")
	writeComposeApp(t, appsDir, "karakeep")

	apps, err := NewScanner(Config{AppsDir: appsDir, Allowlist: []string{"karakeep"}}).Scan()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, "karakeep", apps[0].Name)
}

func TestScannerRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	appsDir := filepath.Join(root, "apps")
	escapeDir := filepath.Join(root, "outside")
	require.NoError(t, os.MkdirAll(appsDir, 0o755))
	writeComposeApp(t, root, "outside")

	link := filepath.Join(appsDir, "escape")
	if err := os.Symlink(escapeDir, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	_, err := NewScanner(Config{AppsDir: appsDir}).Scan()
	require.Error(t, err)
}

func writeComposeApp(t *testing.T, appsDir string, name string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(appsDir, name), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, name, "docker-compose.yaml"), []byte(`services: { app: { image: example/app:latest } }`), 0o644))
}

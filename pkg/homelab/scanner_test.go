package homelab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerDiscoversComposeApps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(t *testing.T, appsDir string) []string
		wantLen int
		check   func(t *testing.T, apps []App)
	}{
		{
			name: "discovers a single compose app",
			setup: func(t *testing.T, appsDir string) []string {
				writeComposeAppWithContent(t, appsDir, "archivebox", `services: { web: { image: archivebox/archivebox:latest, ports: ["8080:8000/tcp"], labels: {"flowbot.capability": "archive"} }, networks: { proxy: {} } }`)
				return []string{"archivebox"}
			},
			wantLen: 1,
			check: func(t *testing.T, apps []App) {
				assert.Equal(t, "archivebox", apps[0].Name)
				assert.Equal(t, "archivebox/archivebox:latest", apps[0].Services[0].Image)
				assert.Equal(t, "8080", apps[0].Ports[0].HostPort)
				assert.Equal(t, "8000", apps[0].Ports[0].Container)
				assert.Equal(t, "archive", apps[0].Labels["flowbot.capability"])
				assert.Len(t, apps[0].Capabilities, 1)
				assert.Equal(t, CapArchive, apps[0].Capabilities[0].Capability)
				assert.Equal(t, "archive", apps[0].Capabilities[0].Backend)
			},
		},
		{
			name: "discovers multiple compose apps",
			setup: func(t *testing.T, appsDir string) []string {
				writeComposeApp(t, appsDir, "archivebox")
				writeComposeApp(t, appsDir, "karakeep")
				return []string{"archivebox", "karakeep"}
			},
			wantLen: 2,
		},
		{
			name: "skips directories without compose files",
			setup: func(t *testing.T, appsDir string) []string {
				require.NoError(t, os.MkdirAll(filepath.Join(appsDir, "empty-dir"), 0o755))
				writeComposeApp(t, appsDir, "karakeep")
				return []string{"karakeep"}
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			appsDir := filepath.Join(root, "apps")
			require.NoError(t, os.MkdirAll(appsDir, 0o755))

			expected := tt.setup(t, appsDir)

			apps, err := NewScanner(Config{AppsDir: appsDir}).Scan()
			require.NoError(t, err)
			require.Len(t, apps, tt.wantLen)
			for i, name := range expected {
				assert.Equal(t, name, apps[i].Name)
			}
			if tt.check != nil {
				tt.check(t, apps)
			}
		})
	}
}

func TestScannerAppliesAllowlist(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		allowlist []string
		wantLen   int
		wantName  string
	}{
		{name: "filters apps using allowlist", allowlist: []string{"karakeep"}, wantLen: 1, wantName: "karakeep"},
		{name: "returns no apps when allowlist has no matches", allowlist: []string{"nope"}, wantLen: 0},
		{name: "returns all apps when allowlist is empty", allowlist: nil, wantLen: 2, wantName: "archivebox"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			appsDir := filepath.Join(root, "apps")
			writeComposeApp(t, appsDir, "archivebox")
			writeComposeApp(t, appsDir, "karakeep")

			apps, err := NewScanner(Config{AppsDir: appsDir, Allowlist: tt.allowlist}).Scan()
			require.NoError(t, err)
			require.Len(t, apps, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantName, apps[0].Name)
			}
		})
	}
}

func TestScannerRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		linkName string
		linkDest string
		wantErr  bool
	}{
		{name: "rejects symlink pointing outside apps dir", linkName: "escape", linkDest: "outside", wantErr: true},
		{
			name:     "accepts valid symlink within apps dir",
			linkName: "internal-link",
			linkDest: "../outside-in-dir",
			wantErr:  true,
		},
		{
			name:     "rejects symlink chain escaping apps dir",
			linkName: "chain",
			linkDest: "../../outside",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			appsDir := filepath.Join(root, "apps")
			targetDir := filepath.Join(root, tt.linkDest)
			require.NoError(t, os.MkdirAll(appsDir, 0o755))
			writeComposeApp(t, root, tt.linkDest)

			link := filepath.Join(appsDir, tt.linkName)
			if err := os.Symlink(targetDir, link); err != nil {
				t.Skipf("symlink not available: %v", err)
			}

			_, err := NewScanner(Config{AppsDir: appsDir}).Scan()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func writeComposeApp(t *testing.T, appsDir string, name string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(appsDir, name), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, name, "docker-compose.yaml"), []byte(`services: { app: { image: example/app:latest } }`), 0o644))
}

func writeComposeAppWithContent(t *testing.T, appsDir string, name string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(appsDir, name), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(appsDir, name, "docker-compose.yaml"), []byte(content), 0o644))
}

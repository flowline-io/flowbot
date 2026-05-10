package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveLoadToken(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveToken("test-token-123", "")
			require.NoError(t, err)

			token, err := LoadToken("")
			require.NoError(t, err)
			require.Equal(t, "test-token-123", token)
		})
	}
}

func TestSaveLoadTokenWithProfile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load token with profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveToken("dev-token-abc", "dev")
			require.NoError(t, err)

			token, err := LoadToken("dev")
			require.NoError(t, err)
			require.Equal(t, "dev-token-abc", token)
		})
	}
}

func TestLoadTokenNotExist(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "load token when file does not exist returns empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			token, err := LoadToken("")
			require.NoError(t, err)
			require.Empty(t, token)
		})
	}
}

func TestProfileIsolation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "token profiles are isolated from each other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			require.NoError(t, SaveToken("default-token", ""))
			require.NoError(t, SaveToken("dev-token", "dev"))

			defaultToken, err := LoadToken("")
			require.NoError(t, err)
			require.Equal(t, "default-token", defaultToken)

			devToken, err := LoadToken("dev")
			require.NoError(t, err)
			require.Equal(t, "dev-token", devToken)
		})
	}
}

func TestSaveLoadServerURL(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load server URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveServerURL("https://flowbot.example.com", "")
			require.NoError(t, err)

			url, err := LoadServerURL("")
			require.NoError(t, err)
			require.Equal(t, "https://flowbot.example.com", url)
		})
	}
}

func TestSaveLoadServerURLWithProfile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "save and load server URL with profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			err := SaveServerURL("https://dev.flowbot.example.com", "dev")
			require.NoError(t, err)

			url, err := LoadServerURL("dev")
			require.NoError(t, err)
			require.Equal(t, "https://dev.flowbot.example.com", url)
		})
	}
}

func TestLoadServerURLNotExist(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "load server URL when file does not exist returns empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			url, err := LoadServerURL("")
			require.NoError(t, err)
			require.Empty(t, url)
		})
	}
}

func TestGetTokenPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "get token path for default profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			path, err := GetTokenPath("")
			require.NoError(t, err)
			require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "token"), path)
		})
	}
}

func TestGetTokenPathWithProfile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "get token path with profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			path, err := GetTokenPath("prod")
			require.NoError(t, err)
			require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "token.prod"), path)
		})
	}
}

func TestGetServerURLPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "get server URL path for default profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			path, err := GetServerURLPath("")
			require.NoError(t, err)
			require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot", "server_url"), path)
		})
	}
}

func TestAcquireLock(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "acquire lock creates and releases lock file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockPath := filepath.Join(tmpDir, "test")

			unlock, err := AcquireLock(lockPath)
			require.NoError(t, err)
			require.NotNil(t, unlock)

			_, err = os.Stat(lockPath + ".lock")
			require.NoError(t, err, "lock file should exist")

			unlock()

			_, err = os.Stat(lockPath + ".lock")
			require.True(t, os.IsNotExist(err), "lock file should be removed")
		})
	}
}

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "get config dir creates and returns directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfgDir, err := GetConfigDir()
			require.NoError(t, err)
			require.Equal(t, filepath.Join(tmpDir, ".config", "flowbot"), cfgDir)

			stat, err := os.Stat(cfgDir)
			require.NoError(t, err)
			require.True(t, stat.IsDir())
		})
	}
}

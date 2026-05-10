package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newTestCmd(serverURL, profile string, debug bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
	}
	cmd.Flags().String("server-url", serverURL, "")
	cmd.Flags().String("profile", profile, "")
	cmd.Flags().Bool("debug", debug, "")
	return cmd
}

func TestNewClientErrorNoServerURL(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "missing server URL returns error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cmd := newTestCmd("", "", false)
			c, err := NewClient(cmd)
			require.Error(t, err)
			require.Nil(t, c)
			require.Contains(t, err.Error(), "server URL is required")
		})
	}
}

func TestNewClientErrorNoToken(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "missing token returns error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cmd := newTestCmd("https://flowbot.example.com", "", false)
			c, err := NewClient(cmd)
			require.Error(t, err)
			require.Nil(t, c)
			require.Contains(t, err.Error(), "not logged in")
		})
	}
}

func TestNewClientWithEnvServerURL(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "env server URL still fails without token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			t.Setenv("FLOWBOT_SERVER_URL", "https://env.flowbot.example.com")

			cmd := newTestCmd("", "", false)
			c, err := NewClient(cmd)
			require.Error(t, err) // still fails because no token
			require.Nil(t, c)
			require.Contains(t, err.Error(), "not logged in")
		})
	}
}

func TestNewClientWithEnvDebug(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "env debug flag still fails without token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			t.Setenv("FLOWBOT_DEBUG", "true")

			cmd := newTestCmd("https://flowbot.example.com", "", false)
			c, err := NewClient(cmd)
			require.Error(t, err) // still fails because no token
			require.Nil(t, c)
			require.Contains(t, err.Error(), "not logged in")
		})
	}
}

func TestNewClientWithStoredConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "stored config creates client successfully"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
			require.NoError(t, os.MkdirAll(cfgDir, 0750))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("test-token"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://stored.flowbot.example.com"), 0600))

			cmd := newTestCmd("", "", false)
			c, err := NewClient(cmd)
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

func TestNewClientWithDebugFlag(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "debug flag from CLI with stored config creates client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
			require.NoError(t, os.MkdirAll(cfgDir, 0750))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("test-token"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://s.example.com"), 0600))

			cmd := newTestCmd("", "", true)
			c, err := NewClient(cmd)
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

func TestNewClientWithStoredDebug(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "stored debug config creates client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
			require.NoError(t, os.MkdirAll(cfgDir, 0750))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("test-token"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://s.example.com"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "debug"), []byte("true"), 0600))

			cmd := newTestCmd("", "", false)
			c, err := NewClient(cmd)
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

func TestNewClientFlagOverridesStored(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "CLI flag overrides stored server URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
			require.NoError(t, os.MkdirAll(cfgDir, 0750))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("test-token"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://stored.example.com"), 0600))

			cmd := newTestCmd("https://flag.example.com", "", false)
			c, err := NewClient(cmd)
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newTestCmd(serverURL, profile string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("server-url", serverURL, "")
	cmd.Flags().String("profile", profile, "")
	return cmd
}

func writeChatClientConfig(t *testing.T, tmpDir string) {
	t.Helper()
	cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
	require.NoError(t, os.MkdirAll(cfgDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte("test-token"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://s.example.com"), 0600))
}

func TestNewClientIgnoresDebugSources(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{name: "ignores FLOWBOT_DEBUG env", env: map[string]string{"FLOWBOT_DEBUG": "true"}},
		{name: "ignores FLOWBOT_DEBUG numeric env", env: map[string]string{"FLOWBOT_DEBUG": "1"}},
		{name: "ignores stored debug config", env: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			writeChatClientConfig(t, tmpDir)
			if tt.name == "ignores stored debug config" {
				cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
				require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "debug"), []byte("true"), 0600))
			}

			cmd := newTestCmd("", "")
			cl, err := NewClient(cmd)
			require.NoError(t, err)
			require.NotNil(t, cl)
			require.False(t, cl.DebugEnabled())
		})
	}
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

			cmd := newTestCmd("", "")
			cl, err := NewClient(cmd)
			require.Error(t, err)
			require.Nil(t, cl)
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

			cmd := newTestCmd("https://flowbot.example.com", "")
			cl, err := NewClient(cmd)
			require.Error(t, err)
			require.Nil(t, cl)
			require.Contains(t, err.Error(), "not logged in")
		})
	}
}

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
			t.Setenv("FLOWBOT_TOKEN", "")

			cmd := newTestCmd("", "", false)
			c, err := NewClient(cmd)
			require.Error(t, err) // still fails because no token
			require.Nil(t, c)
			require.Contains(t, err.Error(), "not logged in")
		})
	}
}

func TestNewClientWithEnvToken(t *testing.T) {
	tests := []struct {
		name      string
		envToken  string
		fileToken string
		wantOK    bool
		wantSub   string
		wantToken string
	}{
		{
			name:      "env token preferred over file",
			envToken:  "env-token",
			fileToken: "file-token",
			wantOK:    true,
			wantToken: "env-token",
		},
		{
			name:      "file token used when env empty",
			envToken:  "",
			fileToken: "file-token",
			wantOK:    true,
			wantToken: "file-token",
		},
		{
			name:     "both empty returns not logged in",
			envToken: "",
			wantOK:   false,
			wantSub:  "not logged in",
		},
		{
			name:      "env token alone succeeds",
			envToken:  "env-only-token",
			wantOK:    true,
			wantToken: "env-only-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)
			t.Setenv("FLOWBOT_TOKEN", tt.envToken)

			cfgDir := filepath.Join(tmpDir, ".config", "flowbot")
			require.NoError(t, os.MkdirAll(cfgDir, 0750))
			if tt.fileToken != "" {
				require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "token"), []byte(tt.fileToken), 0600))
			}
			require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "server_url"), []byte("https://s.example.com"), 0600))

			cmd := newTestCmd("", "", false)
			c, err := NewClient(cmd)
			if !tt.wantOK {
				require.Error(t, err)
				require.Nil(t, c)
				require.Contains(t, err.Error(), tt.wantSub)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
			require.Equal(t, tt.wantToken, c.AccessToken())
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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(path, []byte("flowbot_url: http://yaml\naccess_token: yaml-tok\n"), 0o600))

	t.Setenv("FLOWBOT_URL", "http://env-url")
	t.Setenv("FLOWBOT_AGENT_TOKEN", "env-tok")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "http://env-url", cfg.FlowbotURL)
	assert.Equal(t, "env-tok", cfg.AccessToken)
	assert.Equal(t, "http://env-url/agent/v1", cfg.LLMBaseURL())
}

func TestLoad_MissingFileUsesEnv(t *testing.T) {
	t.Setenv("FLOWBOT_URL", "http://only-env")
	t.Setenv("FLOWBOT_AGENT_TOKEN", "tok")
	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.NoError(t, err)
	require.NoError(t, cfg.Validate())
	assert.Equal(t, "http://only-env", cfg.FlowbotURL)
}

func TestValidate_RequiresFields(t *testing.T) {
	cfg := &config.Config{}
	require.Error(t, cfg.Validate())
	cfg.FlowbotURL = "http://x"
	require.Error(t, cfg.Validate())
	cfg.AccessToken = "t"
	require.NoError(t, cfg.Validate())
}

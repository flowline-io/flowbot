// Package config loads cmd/gateway sidecar configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/goccy/go-yaml"
)

// Config is the local cmd/gateway sidecar configuration.
type Config struct {
	FlowbotURL         string        `yaml:"flowbot_url"`
	AccessToken        string        `yaml:"access_token"`
	WorkerID           string        `yaml:"worker_id"`
	ClaimInterval      time.Duration `yaml:"claim_interval"`
	HeartbeatInterval  time.Duration `yaml:"heartbeat_interval"`
	DefaultWorkspace   string        `yaml:"default_workspace"`
	WorkspaceAllowlist []string      `yaml:"workspace_allowlist"`
	MaxConcurrent      int           `yaml:"max_concurrent"`
	JobTimeout         time.Duration `yaml:"job_timeout"`
	CursorBinary       string        `yaml:"cursor_binary"`
	CursorAPIKey       string        `yaml:"cursor_api_key"`
	AgentAccessToken   string        `yaml:"agent_access_token"`
	Listen             string        `yaml:"listen"`
}

// Load reads gateway YAML from path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.WorkerID == "" {
		host, err := os.Hostname()
		if err != nil || host == "" {
			host = "worker"
		}
		c.WorkerID = host + "-" + types.Id()[:8]
	}
	if c.ClaimInterval <= 0 {
		c.ClaimInterval = 2 * time.Second
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 20 * time.Second
	}
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 1
	}
	if c.JobTimeout <= 0 {
		c.JobTimeout = 30 * time.Minute
	}
	if c.CursorBinary == "" {
		c.CursorBinary = "agent"
	}
	c.applyEnvOverrides()
}

func (c *Config) applyEnvOverrides() {
	if c.CursorAPIKey == "" {
		c.CursorAPIKey = os.Getenv("CURSOR_API_KEY")
	}
	if c.AgentAccessToken == "" {
		c.AgentAccessToken = os.Getenv("FLOWBOT_AGENT_TOKEN")
	}
	if tok := os.Getenv("FLOWBOT_TOKEN"); tok != "" && c.AccessToken == "" {
		c.AccessToken = tok
	}
	if u := os.Getenv("FLOWBOT_SERVER_URL"); u != "" && c.FlowbotURL == "" {
		c.FlowbotURL = u
	}
	if u := os.Getenv("FLOWBOT_URL"); u != "" && c.FlowbotURL == "" {
		c.FlowbotURL = u
	}
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.FlowbotURL) == "" {
		return fmt.Errorf("flowbot_url is required")
	}
	if strings.TrimSpace(c.AccessToken) == "" {
		return fmt.Errorf("access_token is required")
	}
	if strings.TrimSpace(c.DefaultWorkspace) == "" {
		return fmt.Errorf("default_workspace is required")
	}
	if len(c.WorkspaceAllowlist) == 0 {
		c.WorkspaceAllowlist = []string{c.DefaultWorkspace}
	}
	return nil
}

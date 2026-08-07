// Package config loads cmd/agent headless CLI configuration.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// Config is the local flowbot-agent configuration.
type Config struct {
	FlowbotURL  string `yaml:"flowbot_url"`
	AccessToken string `yaml:"access_token"`
}

// Load reads agent YAML from path. Missing file is allowed when env supplies values.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else if err := yaml.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	}
	cfg.applyEnv()
	return cfg, nil
}

func (c *Config) applyEnv() {
	if u := firstNonEmpty(os.Getenv("FLOWBOT_URL"), os.Getenv("FLOWBOT_SERVER_URL")); u != "" {
		c.FlowbotURL = u
	}
	if tok := firstNonEmpty(os.Getenv("FLOWBOT_AGENT_TOKEN"), os.Getenv("FLOWBOT_TOKEN")); tok != "" {
		c.AccessToken = tok
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.FlowbotURL) == "" {
		return fmt.Errorf("flowbot_url is required (config or FLOWBOT_URL)")
	}
	if strings.TrimSpace(c.AccessToken) == "" {
		return fmt.Errorf("access_token is required (config or FLOWBOT_AGENT_TOKEN)")
	}
	return nil
}

// LLMBaseURL returns the OpenAI-compatible base URL for the agent LLM proxy.
func (c *Config) LLMBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(c.FlowbotURL), "/") + "/agent/v1"
}

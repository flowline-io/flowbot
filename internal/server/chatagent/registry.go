package chatagent

import (
	"os"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
)

const agentName = "chat"

// NewRegistry registers coding tools for the assistant.
func NewRegistry(ws coding.Workspace) (*tool.Registry, error) {
	registry := tool.NewRegistry()
	if err := coding.RegisterAll(registry, ws); err != nil {
		return nil, err
	}
	registry.SetActive(coding.ActiveToolNames())
	return registry, nil
}

// WorkspaceFromConfig resolves workspace settings from application config.
func WorkspaceFromConfig() coding.Workspace {
	cfg := config.App.ChatAgent
	root := cfg.Workspace
	if root == "" {
		cwd, err := os.Getwd()
		if err == nil {
			root = cwd
		}
	}
	timeout := cfg.ShellTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxOutput := cfg.MaxToolOutput
	if maxOutput <= 0 {
		maxOutput = 8192
	}
	return coding.Workspace{
		Root:      root,
		Timeout:   timeout,
		MaxOutput: maxOutput,
	}
}

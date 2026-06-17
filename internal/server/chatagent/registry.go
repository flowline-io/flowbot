package chatagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
)

const agentName = "chat"

// NewRegistry registers assistant tools including DB-backed skills support.
// When taskDeps is non-nil, the subagent delegation task tool is registered and activated.
func NewRegistry(ws coding.Workspace, taskDeps *TaskToolDeps) (*tool.Registry, error) {
	registry := tool.NewRegistry()
	if err := coding.RegisterAll(registry, ws, nil); err != nil {
		return nil, err
	}
	if err := registry.Register(ReadSkillTool{}); err != nil {
		return nil, err
	}
	if taskDeps != nil {
		if err := registry.Register(NewTaskTool(ws, *taskDeps)); err != nil {
			return nil, err
		}
	}
	registry.SetActive(ActiveToolNames())
	return registry, nil
}

// ActiveToolNames returns the default active tool names for the chat assistant.
func ActiveToolNames() []string {
	names := coding.ActiveToolNames()
	return append(names, "read_skill", taskToolName)
}

// WorkspaceFromConfig resolves workspace settings from application config.
func WorkspaceFromConfig() (coding.Workspace, error) {
	cfg := config.App.ChatAgent
	root := strings.TrimSpace(cfg.Workspace)
	if root == "" {
		return coding.Workspace{}, fmt.Errorf("chat_agent.workspace is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return coding.Workspace{}, fmt.Errorf("chat_agent.workspace: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return coding.Workspace{}, fmt.Errorf("chat_agent.workspace: %w", err)
	}
	if !info.IsDir() {
		return coding.Workspace{}, fmt.Errorf("chat_agent.workspace is not a directory")
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
		Root:      abs,
		Timeout:   timeout,
		MaxOutput: maxOutput,
	}, nil
}

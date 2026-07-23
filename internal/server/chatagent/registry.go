package chatagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/clip"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	agentnotify "github.com/flowline-io/flowbot/pkg/agent/notify"
	"github.com/flowline-io/flowbot/pkg/agent/sandbox"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

const agentName = "chat"

// NewRegistry registers assistant tools including DB-backed skills support.
// When taskDeps is non-nil, the delegate_subagent tool is registered and activated.
// When scheduleDeps is non-nil, scheduled task tools are registered and activated.
func NewRegistry(ws coding.Workspace, taskDeps *TaskToolDeps, scheduleDeps *ScheduleToolDeps) (*tool.Registry, error) {
	registry := tool.NewRegistry()
	if err := coding.RegisterAll(registry, ws, executionEnvForWorkspace(ws)); err != nil {
		return nil, err
	}
	if err := clip.Register(registry, config.App.Flowbot.URL); err != nil {
		return nil, err
	}
	uid := registryUID(taskDeps, scheduleDeps)
	if err := agentnotify.Register(registry, uid); err != nil {
		return nil, err
	}
	if err := registry.Register(ReadSkillTool{}); err != nil {
		return nil, err
	}
	if err := registry.Register(SearchKnowledgeTool{}); err != nil {
		return nil, err
	}
	if err := registry.Register(GetKnowledgeTool{}); err != nil {
		return nil, err
	}
	if taskDeps != nil {
		if err := registry.Register(NewTaskTool(ws, *taskDeps)); err != nil {
			return nil, err
		}
	}
	if scheduleDeps != nil {
		if err := NewScheduleTools(*scheduleDeps).Register(registry); err != nil {
			return nil, err
		}
	}
	if sessionID := registrySessionID(taskDeps, scheduleDeps); sessionID != "" {
		if err := NewTodoTools(TodoToolDeps{SessionID: sessionID}).Register(registry); err != nil {
			return nil, err
		}
	}
	memTool, err := NewUpdateMemoryTool()
	if err != nil {
		return nil, err
	}
	if err := registry.Register(memTool); err != nil {
		return nil, err
	}
	registry.SetActive(ActiveToolNames())
	return registry, nil
}

// NewSubagentRegistry registers coding tools and an optional allowlisted read_skill tool for subagent runs.
func NewSubagentRegistry(ws coding.Workspace, skillAllowlist []string) (*tool.Registry, error) {
	registry := tool.NewRegistry()
	if err := coding.RegisterAll(registry, ws, executionEnvForWorkspace(ws)); err != nil {
		return nil, err
	}
	skillTool := ReadSkillTool{}
	if len(skillAllowlist) > 0 {
		skillTool = NewReadSkillTool(skillAllowlist)
	}
	if err := registry.Register(skillTool); err != nil {
		return nil, err
	}
	if err := registry.Register(SearchKnowledgeTool{}); err != nil {
		return nil, err
	}
	if err := registry.Register(GetKnowledgeTool{}); err != nil {
		return nil, err
	}
	memTool, err := NewUpdateMemoryTool()
	if err != nil {
		return nil, err
	}
	if err := registry.Register(memTool); err != nil {
		return nil, err
	}
	return registry, nil
}

// ActiveToolNames returns the default active tool names for the chat assistant.
func ActiveToolNames() []string {
	names := coding.ActiveToolNames()
	names = append(names, clip.ActiveToolNames()...)
	names = append(names, agentnotify.ActiveToolNames()...)
	names = append(names, "read_skill", delegateSubagentToolName)
	names = append(names, KnowledgeToolNames()...)
	names = append(names, scheduleToolNames()...)
	names = append(names, todoToolNames()...)
	names = append(names, updateMemoryToolName)
	return names
}

func registryUID(taskDeps *TaskToolDeps, scheduleDeps *ScheduleToolDeps) types.Uid {
	if scheduleDeps != nil && scheduleDeps.UID != "" {
		return scheduleDeps.UID
	}
	if taskDeps != nil {
		return taskDeps.UID
	}
	return types.Uid("")
}

func registrySessionID(taskDeps *TaskToolDeps, scheduleDeps *ScheduleToolDeps) string {
	if scheduleDeps != nil && scheduleDeps.SessionID != "" {
		return scheduleDeps.SessionID
	}
	if taskDeps != nil && taskDeps.SessionID != "" {
		return taskDeps.SessionID
	}
	return ""
}

// BaseToolNamesForRun returns the active tool set for one run.
// Autonomous runs omit update_memory unless it appears in explicitTools.
func BaseToolNamesForRun(kind RunKind, explicitTools []string) []string {
	if len(explicitTools) > 0 {
		return append([]string(nil), explicitTools...)
	}
	names := ActiveToolNames()
	if IsAutonomousRunKind(kind) {
		return omitToolName(names, updateMemoryToolName)
	}
	return names
}

func omitToolName(names []string, drop string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if name != drop {
			out = append(out, name)
		}
	}
	return out
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
		Root:                 abs,
		Timeout:              timeout,
		MaxOutput:            maxOutput,
		WebSearchSearxURL:    strings.TrimSpace(cfg.WebSearch.SearxURL),
		WebSearchBraveAPIKey: strings.TrimSpace(cfg.WebSearch.BraveAPIKey),
	}, nil
}

func executionEnvForWorkspace(ws coding.Workspace) env.ExecutionEnv {
	cfg := config.App.ChatAgent.Sandbox
	if !cfg.Enabled {
		return nil
	}
	return sandbox.New(sandbox.ConfigFromChatAgent(cfg, ws.Root), env.Default(), nil)
}

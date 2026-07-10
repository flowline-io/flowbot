package chatagent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
)

// registerPathSensors adds PostToolUse workspace path verification for coding tools.
func registerPathSensors(reg *hooks.Registry) {
	hooks.OnToolResult(reg, func(_ context.Context, event hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
		switch event.ToolCall.Name {
		case "write_file", "read_file", "run_code":
		default:
			return nil, nil
		}
		if event.Result.IsError {
			return nil, nil
		}
		workspaceRoot := strings.TrimSpace(config.App.ChatAgent.Workspace)
		if workspaceRoot == "" {
			return nil, nil
		}
		pathArg := pathArgFromTool(event.ToolCall.Name, event.Args)
		if pathArg == "" {
			return nil, nil
		}
		ws := coding.Workspace{Root: workspaceRoot}
		if resolved := ws.ResolvePath(pathArg); resolved.IsOk() {
			absRoot, err := filepath.Abs(workspaceRoot)
			if err == nil {
				if !strings.HasPrefix(strings.ToLower(resolved.Value()), strings.ToLower(absRoot)) {
					return pathSensorError(event.ToolCall.Name, pathArg), nil
				}
			}
			return nil, nil
		}
		return pathSensorError(event.ToolCall.Name, pathArg), nil
	})
}

func pathSensorError(toolName, pathArg string) *hooks.ToolResultResult {
	text := tool.FormatToolError(
		"path_escape",
		fmt.Sprintf("tool %s path %q is outside the workspace", toolName, pathArg),
		"use a relative path inside the configured chat_agent.workspace",
	)
	isErr := true
	return &hooks.ToolResultResult{
		Parts:   []msg.ContentPart{msg.TextPart{Text: text}},
		IsError: &isErr,
	}
}

func pathArgFromTool(name string, args map[string]any) string {
	if args == nil {
		return ""
	}
	switch name {
	case "write_file", "read_file":
		return strings.TrimSpace(fmt.Sprint(args["path"]))
	case "run_code":
		filename := strings.TrimSpace(fmt.Sprint(args["filename"]))
		if filename == "" {
			language := strings.ToLower(strings.TrimSpace(fmt.Sprint(args["language"])))
			filename = defaultRunCodeFilename(language)
		}
		return filepath.Join(".flowbot-run", filename)
	default:
		return ""
	}
}

func defaultRunCodeFilename(language string) string {
	switch language {
	case "python", "py":
		return "script.py"
	case "shell", "sh", "bash":
		return "script.sh"
	default:
		return "snippet.txt"
	}
}

// registerLintSensor observes write_file of Go sources without rewriting tool results.
func registerLintSensor(reg *hooks.Registry) {
	hooks.OnToolResult(reg, func(_ context.Context, event hooks.ToolResultEvent) (*hooks.ToolResultResult, error) {
		if !config.App.ChatAgent.Sensors.LintOnWrite {
			return nil, nil
		}
		if event.ToolCall.Name != "write_file" || event.Result.IsError {
			return nil, nil
		}
		pathArg := pathArgFromTool(event.ToolCall.Name, event.Args)
		if pathArg == "" || !strings.EqualFold(filepath.Ext(pathArg), ".go") {
			return nil, nil
		}
		flog.Info("[chat-agent] lint sensor observed write_file path=%s", pathArg)
		metrics.Agent().IncSensorLint("observed")
		return nil, nil
	})
}

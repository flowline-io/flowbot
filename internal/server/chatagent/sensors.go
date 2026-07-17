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
		case "write_file", "read_file", "run_code", "list_dir", "glob_files", "grep_files", "apply_patch":
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
		paths := pathArgsFromTool(event.ToolCall.Name, event.Args)
		if len(paths) == 0 {
			return nil, nil
		}
		ws := coding.Workspace{Root: workspaceRoot}
		for _, pathArg := range paths {
			if resolved := ws.ResolvePath(pathArg); resolved.IsOk() {
				continue
			}
			return pathSensorError(event.ToolCall.Name, pathArg), nil
		}
		return nil, nil
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

func pathArgsFromTool(name string, args map[string]any) []string {
	if args == nil {
		return nil
	}
	switch name {
	case "write_file", "read_file", "list_dir", "glob_files", "grep_files":
		path := normalizeSensorPath(fmt.Sprint(args["path"]))
		if path == "" {
			switch name {
			case "list_dir", "glob_files", "grep_files":
				return []string{"."}
			default:
				return nil
			}
		}
		return []string{path}
	case "apply_patch":
		return coding.PatchFilePaths(fmt.Sprint(args["patch"]))
	case "run_code":
		filename := normalizeSensorPath(fmt.Sprint(args["filename"]))
		if filename == "" {
			language := strings.ToLower(strings.TrimSpace(fmt.Sprint(args["language"])))
			filename = defaultRunCodeFilename(language)
		}
		return []string{filepath.Join(".flowbot-run", filename)}
	default:
		return nil
	}
}

func normalizeSensorPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "<nil>" {
		return ""
	}
	if after, ok := strings.CutPrefix(path, "file://"); ok {
		path = after
	}
	return path
}

// pathArgFromTool returns the primary path argument for sensors that need a single path.
func pathArgFromTool(name string, args map[string]any) string {
	paths := pathArgsFromTool(name, args)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
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

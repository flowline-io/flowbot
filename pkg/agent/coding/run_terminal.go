package coding

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// RunTerminalTool executes shell commands inside the workspace.
type RunTerminalTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (RunTerminalTool) Name() string { return "run_terminal" }

// Description explains the tool to the model.
func (RunTerminalTool) Description() string {
	return "Runs a shell command in the workspace directory and returns combined stdout and stderr"
}

// Parameters returns the JSON schema for tool arguments.
func (RunTerminalTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs the requested command.
func (t RunTerminalTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
	command := strings.TrimSpace(fmt.Sprint(args["command"]))
	if command == "" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "command is required", "provide a non-empty shell command"), nil
	}
	if onUpdate != nil {
		_ = onUpdate("running command...")
	}

	rootResult := t.Workspace.absRoot()
	if !rootResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(rootResult.ErrorValue())), nil
	}

	timeout := t.Workspace.Timeout
	if timeout <= 0 {
		timeout = defaultShellTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	execResult := t.executionEnv().Exec(runCtx, env.ExecOptions{
		Command: command,
		Dir:     rootResult.Value(),
		Timeout: runCtx,
	})
	if !execResult.IsOk() {
		return toolError(id, t.Name(), env.FormatExecutionError(execResult.ErrorValue())), nil
	}

	capture := execResult.Value()
	output := t.Workspace.TruncateOutput(env.FormatExecOutput(capture, capture.ExitCode != 0, nil))
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: output}},
		IsError:    capture.ExitCode != 0,
	}, nil
}

func (t RunTerminalTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

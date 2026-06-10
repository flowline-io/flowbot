package coding

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// RunTerminalTool executes shell commands inside the workspace.
type RunTerminalTool struct {
	Workspace Workspace
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
		return toolError(id, t.Name(), "command is required"), nil
	}
	if onUpdate != nil {
		_ = onUpdate("running command...")
	}

	root, err := t.Workspace.absRoot()
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}

	timeout := t.Workspace.Timeout
	if timeout <= 0 {
		timeout = defaultShellTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(runCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(runCtx, "sh", "-c", command)
	}
	cmd.Dir = root

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	output := t.Workspace.TruncateOutput(buf.String())
	if err != nil {
		output = fmt.Sprintf("exit error: %v\n%s", err, output)
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: strings.TrimSpace(output)}},
		IsError:    err != nil,
	}, nil
}

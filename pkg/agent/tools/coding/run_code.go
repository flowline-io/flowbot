package coding

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
)

// RunCodeTool executes source code by writing a temporary file and invoking an interpreter.
type RunCodeTool struct {
	Workspace Workspace
	Env       env.ExecutionEnv
}

// Name returns the tool identifier.
func (RunCodeTool) Name() string { return "run_code" }

// Description explains the tool to the model.
func (RunCodeTool) Description() string {
	return "Executes Python or shell code in the workspace using a language-specific interpreter"
}

// Parameters returns the JSON schema for tool arguments.
func (RunCodeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"language": map[string]any{
				"type":        "string",
				"description": "Language identifier: python or shell (aliases: py, sh, bash)",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Source code to execute",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional filename hint such as script.py, script.sh",
			},
		},
		"required": []string{"language", "code"},
	}
}

// Execute runs the provided code snippet.
func (t RunCodeTool) Execute(ctx context.Context, id string, args map[string]any, onUpdate tool.UpdateHandler) (msg.ToolResultMessage, error) {
	language := strings.ToLower(strings.TrimSpace(fmt.Sprint(args["language"])))
	code := fmt.Sprint(args["code"])
	filename := strings.TrimSpace(fmt.Sprint(args["filename"]))
	if language == "" || strings.TrimSpace(code) == "" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "language and code are required", "provide language (python|shell) and non-empty code"), nil
	}
	if len(code) > MaxRunCodeBytes {
		return tool.ErrorResult(id, t.Name(), "invalid_args", fmt.Sprintf("code exceeds %d bytes", MaxRunCodeBytes), "reduce the code size"), nil
	}
	if onUpdate != nil {
		_ = onUpdate("executing code...")
	}

	rootResult := t.Workspace.absRoot()
	if !rootResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(rootResult.ErrorValue())), nil
	}

	timeout := t.Workspace.Timeout
	if timeout <= 0 {
		timeout = DefaultShellTimeout
	}
	res, err := pkgexec.RunCode(ctx, pkgexec.Config{
		Workspace: rootResult.Value(),
		Env:       t.executionEnv(),
		Timeout:   timeout,
		MaxOutput: t.Workspace.MaxOutput,
	}, language, code, filename, "", nil)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: res.Output}},
		IsError:    res.ExitCode != 0,
	}, nil
}

func (t RunCodeTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

func toolError(id, name, text string) msg.ToolResultMessage {
	return tool.ErrorResult(id, name, "tool_error", text, "fix the arguments or path and retry within the workspace")
}

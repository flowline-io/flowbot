package coding

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
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

	if filename == "" {
		filename = defaultFilename(language)
	}
	resolvedResult := t.Workspace.ResolvePath(filepath.Join(".flowbot-run", filename))
	if !resolvedResult.IsOk() {
		return toolError(id, t.Name(), env.FormatFileError(resolvedResult.ErrorValue())), nil
	}
	resolved := resolvedResult.Value()
	execEnv := t.executionEnv()

	if mkdirResult := execEnv.MkdirAll(ctx, filepath.Dir(resolved), 0o755); !mkdirResult.IsOk() {
		return toolError(id, t.Name(), fmt.Sprintf("mkdir: %s", env.FormatFileError(mkdirResult.ErrorValue()))), nil
	}
	if writeResult := execEnv.WriteFile(ctx, resolved, []byte(code), 0o644); !writeResult.IsOk() {
		return toolError(id, t.Name(), fmt.Sprintf("write code file: %s", env.FormatFileError(writeResult.ErrorValue()))), nil
	}
	defer func() {
		_ = execEnv.Remove(context.Background(), resolved)
	}()

	cmdArgs, err := interpreterCommand(language, resolved)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}

	timeout := t.Workspace.Timeout
	if timeout <= 0 {
		timeout = DefaultShellTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	execResult := execEnv.Exec(runCtx, env.ExecOptions{
		Argv:    cmdArgs,
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

func (t RunCodeTool) executionEnv() env.ExecutionEnv {
	if t.Env != nil {
		return t.Env
	}
	return env.Default()
}

func defaultFilename(language string) string {
	switch language {
	case "python", "py":
		return "script.py"
	case "shell", "sh", "bash":
		return "script.sh"
	default:
		return "snippet.txt"
	}
}

func interpreterCommand(language, filePath string) ([]string, error) {
	switch language {
	case "python", "py":
		return []string{"python", filePath}, nil
	case "shell", "sh", "bash":
		return []string{"sh", filePath}, nil
	default:
		return nil, fmt.Errorf("unsupported language %q", language)
	}
}

func toolError(id, name, text string) msg.ToolResultMessage {
	return tool.ErrorResult(id, name, "tool_error", text, "fix the arguments or path and retry within the workspace")
}

package coding

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

const defaultShellTimeout = 60 * time.Second

// RunCodeTool executes source code by writing a temporary file and invoking an interpreter.
type RunCodeTool struct {
	Workspace Workspace
}

// Name returns the tool identifier.
func (RunCodeTool) Name() string { return "run_code" }

// Description explains the tool to the model.
func (RunCodeTool) Description() string {
	return "Executes source code in the workspace using a language-specific interpreter"
}

// Parameters returns the JSON schema for tool arguments.
func (RunCodeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"language": map[string]any{
				"type":        "string",
				"description": "Language identifier: go, python, javascript, shell",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Source code to execute",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional filename hint such as main.go or script.py",
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
		return toolError(id, t.Name(), "language and code are required"), nil
	}
	if onUpdate != nil {
		_ = onUpdate("executing code...")
	}

	root, err := t.Workspace.absRoot()
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}

	if filename == "" {
		filename = defaultFilename(language)
	}
	resolved, err := t.Workspace.ResolvePath(filepath.Join(".flowbot-run", filename))
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return toolError(id, t.Name(), fmt.Sprintf("mkdir: %v", err)), nil
	}
	if err := os.WriteFile(resolved, []byte(code), 0o644); err != nil {
		return toolError(id, t.Name(), fmt.Sprintf("write code file: %v", err)), nil
	}
	defer func() {
		_ = os.Remove(resolved)
	}()

	cmdArgs, err := interpreterCommand(language, resolved)
	if err != nil {
		return toolError(id, t.Name(), err.Error()), nil
	}

	timeout := t.Workspace.Timeout
	if timeout <= 0 {
		timeout = defaultShellTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = root
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	output := t.Workspace.TruncateOutput(buf.String())
	if runErr != nil {
		output = fmt.Sprintf("exit error: %v\n%s", runErr, output)
	}

	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: strings.TrimSpace(output)}},
		IsError:    runErr != nil,
	}, nil
}

func defaultFilename(language string) string {
	switch language {
	case "go", "golang":
		return "main.go"
	case "python", "py":
		return "script.py"
	case "javascript", "js", "node":
		return "script.js"
	case "shell", "sh", "bash":
		return "script.sh"
	default:
		return "snippet.txt"
	}
}

func interpreterCommand(language, filePath string) ([]string, error) {
	switch language {
	case "go", "golang":
		return []string{"go", "run", filePath}, nil
	case "python", "py":
		return []string{"python", filePath}, nil
	case "javascript", "js", "node":
		return []string{"node", filePath}, nil
	case "shell", "sh", "bash":
		return []string{"sh", filePath}, nil
	default:
		return nil, fmt.Errorf("unsupported language %q", language)
	}
}

func toolError(id, name, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
		IsError:    true,
	}
}

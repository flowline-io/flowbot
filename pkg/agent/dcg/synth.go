package dcg

import (
	"fmt"
	"strconv"
	"strings"
)

// ReasonBlocked is the default denial message when dcg omits a reason.
const ReasonBlocked = "command blocked by dcg"

// SynthCommand builds a shell-shaped command string for dcg from run_code inputs.
// Language aliases match pkg/agent/coding run_code interpreters.
func SynthCommand(language, code string) (string, error) {
	lang := strings.ToLower(strings.TrimSpace(language))
	body := strings.TrimSpace(code)
	if lang == "" {
		return "", fmt.Errorf("dcg: language is required")
	}
	if body == "" {
		return "", fmt.Errorf("dcg: code is required")
	}
	quoted := strconv.Quote(body)
	switch lang {
	case "python", "py":
		return "python -c " + quoted, nil
	case "shell", "sh", "bash":
		return "sh -c " + quoted, nil
	default:
		return "", fmt.Errorf("dcg: unsupported language %q", language)
	}
}

// CommandForTool extracts the command string to check for a tool call.
// ok is false when the tool is not guarded by dcg (caller should skip).
func CommandForTool(tool string, args map[string]any) (command string, ok bool, err error) {
	switch tool {
	case "run_terminal":
		cmd, err := requiredStringArg(args, "command")
		if err != nil {
			return "", true, err
		}
		return cmd, true, nil
	case "run_code":
		language, err := requiredStringArg(args, "language")
		if err != nil {
			return "", true, err
		}
		code, err := requiredStringArg(args, "code")
		if err != nil {
			return "", true, err
		}
		synth, err := SynthCommand(language, code)
		if err != nil {
			return "", true, err
		}
		return synth, true, nil
	default:
		return "", false, nil
	}
}

func requiredStringArg(args map[string]any, key string) (string, error) {
	raw, exists := args[key]
	if !exists || raw == nil {
		return "", fmt.Errorf("dcg: %s is required", key)
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return "", fmt.Errorf("dcg: %s is required", key)
	}
	return value, nil
}

// TruncateCommandForLog shortens command strings for log lines.
func TruncateCommandForLog(command string) string {
	const maxLen = 200
	if len(command) <= maxLen {
		return command
	}
	return command[:maxLen] + "..."
}

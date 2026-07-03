package permission

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"
)

// Tool names used by the chat agent coding toolkit.
const (
	ToolRunTerminal         = "run_terminal"
	ToolReadFile            = "read_file"
	ToolWriteFile           = "write_file"
	ToolWebSearch           = "web_search"
	ToolRunCode             = "run_code"
	ToolReadSkill           = "read_skill"
	ToolTask                = "task"
	ToolScheduleTask        = "schedule_task"
	ToolUpdateScheduledTask = "update_scheduled_task"
	ToolListScheduledTasks  = "list_scheduled_tasks"
	ToolCancelScheduledTask = "cancel_scheduled_task"
)

// PermissionKeyForTool maps a tool name to its OpenCode permission key.
func PermissionKeyForTool(tool string) string {
	switch tool {
	case ToolRunTerminal, ToolRunCode:
		return "bash"
	case ToolReadFile:
		return "read"
	case ToolWriteFile:
		return "edit"
	case ToolWebSearch:
		return "websearch"
	case ToolReadSkill:
		return "skill"
	case ToolTask:
		return KeyDelegate
	case ToolScheduleTask, ToolUpdateScheduledTask, ToolCancelScheduledTask:
		return KeySchedule
	case ToolListScheduledTasks:
		return KeyScheduleRead
	default:
		return KeyWildcard
	}
}

// Request is the input for permission evaluation.
type Request struct {
	Tool          string
	Args          map[string]any
	WorkspaceRoot string
	ExternalPath  bool
}

// ExtractedInputs holds match strings derived from a tool call.
type ExtractedInputs struct {
	PermissionKey string
	Primary       string
	Bash          ParseBashCommand
	ExternalPaths []string
}

// ExtractInputs derives permission match inputs from a tool call.
func ExtractInputs(req Request) ExtractedInputs {
	key := PermissionKeyForTool(req.Tool)
	out := ExtractedInputs{PermissionKey: key}
	switch req.Tool {
	case ToolRunTerminal:
		cmd := strings.TrimSpace(fmt.Sprint(req.Args["command"]))
		out.Bash = AnalyzeBashCommand(cmd)
		if out.Bash.Complex {
			out.Primary = cmd
		} else if cmd != "" {
			out.Primary = cmd
		} else {
			out.Primary = out.Bash.Prefix
		}
		out.ExternalPaths = extractPathsFromCommand(cmd, req.WorkspaceRoot)
	case ToolRunCode:
		lang := strings.TrimSpace(fmt.Sprint(req.Args["language"]))
		out.Primary = "run " + lang
		out.Bash = ParseBashCommand{Prefix: out.Primary}
	case ToolReadFile, ToolWriteFile:
		out.Primary = strings.TrimSpace(fmt.Sprint(req.Args["path"]))
	case ToolWebSearch:
		out.Primary = strings.TrimSpace(fmt.Sprint(req.Args["query"]))
	case ToolReadSkill:
		out.Primary = strings.TrimSpace(fmt.Sprint(req.Args["name"]))
	case ToolTask:
		out.Primary = strings.TrimSpace(fmt.Sprint(req.Args["subagent_type"]))
	case ToolScheduleTask:
		out.Primary = strings.TrimSpace(fmt.Sprint(req.Args["name"]))
	case ToolUpdateScheduledTask, ToolCancelScheduledTask:
		out.Primary = strings.TrimSpace(fmt.Sprint(req.Args["task_id"]))
	case ToolListScheduledTasks:
		out.Primary = "*"
	default:
		out.Primary = req.Tool
	}
	if req.ExternalPath && out.Primary != "" {
		out.ExternalPaths = append(out.ExternalPaths, out.Primary)
	}
	return out
}

func extractPathsFromCommand(command, workspaceRoot string) []string {
	if strings.TrimSpace(command) == "" {
		return nil
	}
	segment := command
	if hasShellChain(command) {
		segment = strings.TrimSpace(splitFirstChain(command))
	}
	words, err := shellwords.Parse(segment)
	if err != nil {
		return nil
	}
	words = stripEnvAssignments(words)
	var paths []string
	for _, word := range words {
		if strings.Contains(word, "..") {
			paths = append(paths, word)
			continue
		}
		if filepath.IsAbs(word) || strings.HasPrefix(word, "/") {
			if workspaceRoot != "" && !isUnderRoot(workspaceRoot, filepath.Clean(word)) {
				paths = append(paths, word)
			} else if workspaceRoot == "" && (filepath.IsAbs(word) || strings.HasPrefix(word, "/")) {
				paths = append(paths, word)
			}
			continue
		}
		if strings.HasPrefix(word, "~/") || strings.HasPrefix(word, "$HOME/") {
			paths = append(paths, word)
		}
	}
	return paths
}

func isUnderRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if target == root {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

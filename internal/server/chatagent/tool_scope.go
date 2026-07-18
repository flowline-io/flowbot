package chatagent

import (
	"regexp"
	"slices"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/clip"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
)

// Tool group name constants for dynamic activation.
const (
	ToolGroupCore     = "core"
	ToolGroupFS       = "fs"
	ToolGroupShell    = "shell"
	ToolGroupSearch   = "search"
	ToolGroupSchedule = "schedule"
	ToolGroupSubagent = "subagent"
	ToolGroupMemory   = "memory"
)

var scheduleIntentPattern = regexp.MustCompile(`(?i)\b(schedule|cron|scheduled\s+task|remind\s+me|every\s+day|recurring)\b`)

// ToolScopeInput configures applyToolScope.
type ToolScopeInput struct {
	Mode      string
	Kind      RunKind
	UserText  string
	AllActive []string
}

// ApplyToolScope selects the active tool set for one Prompt.
// Plan mode uses the read-only set. Normal mode excludes schedule tools unless
// the run is a cron scheduled task or the user message matches schedule intent.
func ApplyToolScope(in ToolScopeInput) []string {
	if in.Mode == ModePlan {
		return ReadOnlyToolNames()
	}
	all := in.AllActive
	if len(all) == 0 {
		all = ActiveToolNames()
	}
	includeSchedule := in.Kind == RunKindScheduled || scheduleIntentPattern.MatchString(in.UserText)
	if includeSchedule {
		return append([]string(nil), all...)
	}
	out := make([]string, 0, len(all))
	schedule := scheduleToolNameSet()
	for _, name := range all {
		if schedule[name] {
			continue
		}
		out = append(out, name)
	}
	return out
}

func scheduleToolNameSet() map[string]bool {
	set := make(map[string]bool, len(scheduleToolNames()))
	for _, name := range scheduleToolNames() {
		set[name] = true
	}
	return set
}

// ToolGroupOf returns the group for a known tool name.
func ToolGroupOf(name string) string {
	switch name {
	case "read_file", "write_file", "list_dir", "apply_patch":
		return ToolGroupFS
	case "run_terminal", "run_code":
		return ToolGroupShell
	case "web_search", "web_fetch", "glob_files", "grep_files":
		return ToolGroupSearch
	case clip.CreateToolName, clip.GetToolName:
		return ToolGroupCore
	case taskToolName:
		return ToolGroupSubagent
	case scheduleToolName, updateScheduleToolName, listScheduleToolName, cancelScheduleToolName:
		return ToolGroupSchedule
	case "read_skill":
		return ToolGroupCore
	case updateMemoryToolName:
		return ToolGroupMemory
	default:
		if strings.HasPrefix(name, "schedule") {
			return ToolGroupSchedule
		}
		if slices.Contains(coding.ActiveToolNames(), name) {
			return ToolGroupCore
		}
		return ToolGroupCore
	}
}

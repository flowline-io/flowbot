package approval

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
)

// FlaggedResult describes whether a tool call needs aux review.
type FlaggedResult struct {
	Flagged bool
	Reason  string
}

var (
	inlineScriptPattern = regexp.MustCompile(`(?i)(\s|^)(-c|--command)\s|python\s+-c\b|node\s+-e\b|ruby\s+-e\b|perl\s+-e\b`)
	destructivePattern  = regexp.MustCompile(`(?i)\brm\s+(-[a-zA-Z]*r|\s)*|del\s+/[fs]|format\s+|mkfs\b|dd\s+if=|shred\b`)
	privilegePattern    = regexp.MustCompile(`(?i)\bsudo\b|\bdoas\b|\brunas\b|\bchmod\s+[0-7]{3,4}\b|\bchown\b`)
	forcePushPattern    = regexp.MustCompile(`(?i)git\s+push\b.*--force\b|git\s+push\b.*\s-f\b|--force-with-lease`)
	networkSysPattern   = regexp.MustCompile(`(?i)\biptables\b|\bufw\b|\bfirewall-cmd\b|\bsysctl\b|\bnmcli\b|\bip\s+link\b|\bifconfig\b`)
	sensitiveNamePattern = regexp.MustCompile(`(?i)(^|[/\\])(\.env($|\.)|\.env\.[^/\\]+|id_rsa|id_ed25519|.*\.(pem|key|p12|pfx|jks)|credentials\.json|secret|secrets)`)
)

// IsReadonlyTool reports whether the tool is treated as side-effect free for auto mode.
func IsReadonlyTool(tool string) bool {
	switch tool {
	case permission.ToolReadFile, permission.ToolListDir, permission.ToolGlobFiles, permission.ToolGrepFiles,
		permission.ToolSearchKnowledge, permission.ToolGetKnowledge,
		permission.ToolListScheduledTasks,
		permission.ToolMemoryGet, permission.ToolMemoryList, permission.ToolSearchSessionSummaries,
		permission.ToolListTodos,
		permission.ToolReadSkill:
		return true
	default:
		return false
	}
}

// EvaluateFlagged returns whether a side-effect tool call should enter aux review.
func EvaluateFlagged(req permission.Request) FlaggedResult {
	if IsReadonlyTool(req.Tool) {
		return FlaggedResult{}
	}
	inputs := permission.ExtractInputs(req)

	if req.ExternalPath || len(inputs.ExternalPaths) > 0 {
		return FlaggedResult{Flagged: true, Reason: "workspace-external path access"}
	}

	switch req.Tool {
	case permission.ToolWebSearch, permission.ToolWebFetch:
		return FlaggedResult{Flagged: true, Reason: "network side-effect tool"}
	case permission.ToolDelegateSubagent, permission.ToolScheduleTask,
		permission.ToolUpdateScheduledTask, permission.ToolCancelScheduledTask,
		permission.ToolMemorySet, permission.ToolMemoryDelete:
		return FlaggedResult{Flagged: true, Reason: "privileged side-effect tool"}
	case permission.ToolRunTerminal, permission.ToolRunCode:
		return flagCommand(req, inputs)
	case permission.ToolWriteFile, permission.ToolApplyPatch:
		return flagFileWrite(inputs.Primary)
	default:
		// Unknown side-effect tools are always reviewed.
		return FlaggedResult{Flagged: true, Reason: "unclassified side-effect tool"}
	}
}

func flagCommand(req permission.Request, inputs permission.ExtractedInputs) FlaggedResult {
	cmd := strings.TrimSpace(fmt.Sprint(req.Args["command"]))
	if req.Tool == permission.ToolRunCode {
		code := strings.TrimSpace(fmt.Sprint(req.Args["code"]))
		if code == "" {
			code = strings.TrimSpace(fmt.Sprint(req.Args["source"]))
		}
		cmd = code
		if cmd == "" {
			return FlaggedResult{Flagged: true, Reason: "inline code execution"}
		}
	}
	if inputs.Bash.HasChain || inputs.Bash.Complex {
		return FlaggedResult{Flagged: true, Reason: "complex or chained shell command"}
	}
	if inlineScriptPattern.MatchString(cmd) {
		return FlaggedResult{Flagged: true, Reason: "inline script execution"}
	}
	if destructivePattern.MatchString(cmd) {
		return FlaggedResult{Flagged: true, Reason: "destructive file or disk operation"}
	}
	if privilegePattern.MatchString(cmd) {
		return FlaggedResult{Flagged: true, Reason: "privilege escalation or ownership change"}
	}
	if forcePushPattern.MatchString(cmd) {
		return FlaggedResult{Flagged: true, Reason: "force push"}
	}
	if networkSysPattern.MatchString(cmd) {
		return FlaggedResult{Flagged: true, Reason: "network or system configuration change"}
	}
	if sensitiveNamePattern.MatchString(cmd) {
		return FlaggedResult{Flagged: true, Reason: "command references sensitive path"}
	}
	return FlaggedResult{}
}

func flagFileWrite(path string) FlaggedResult {
	path = strings.TrimSpace(path)
	if path == "" {
		return FlaggedResult{Flagged: true, Reason: "write without path"}
	}
	base := filepath.Base(path)
	if sensitiveNamePattern.MatchString(path) || sensitiveNamePattern.MatchString(base) {
		return FlaggedResult{Flagged: true, Reason: "sensitive file path"}
	}
	return FlaggedResult{}
}

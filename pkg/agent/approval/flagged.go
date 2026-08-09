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

type flagRule struct {
	reason  string
	pattern *regexp.Regexp
}

var (
	// Interpreter-anchored inline execution (avoid bare -c / --command).
	inlineScriptPattern = regexp.MustCompile(`(?i)\b(bash|sh|zsh|ksh|fish)\s+-c\b|\b(python3?|pypy3?)\s+-c\b|\b(node|bun)\s+-e\b|\b(ruby|perl|lua)\s+-e\b|\bphp\s+-r\b|\b(pwsh|powershell)\s+(-[cC]|-Command|-EncodedCommand)\b|\bcmd(\.exe)?\s+/c\b|\bdeno\s+eval\b`)

	// Recursive/force deletes, disk wipe, destructive git reset/clean.
	destructivePattern = regexp.MustCompile(`(?i)\brm\s+(-[a-zA-Z]*[rRfF]|--recursive|--force)\b|\bdel\s+/[fs]\b|\brmdir\s+/s\b|\bRemove-Item\b.*(-(Recurse|Force)\b)|\bformat\s+[A-Za-z]:|\bmkfs(\.[a-z0-9]+)?\b|\bdd\s+.*\b(if|of)=(/dev/|[A-Za-z]:)|\bshred\b|\btruncate\s+-s\s*0\b|\bfind\b.*\s-delete\b|\bwipefs\b|\bunlink\b|\bgit\s+reset\s+--hard\b|\bgit\s+clean\s+-[a-zA-Z]*f`)

	privilegePattern = regexp.MustCompile(`(?i)\b(sudo|sudoedit|doas|pkexec|runas)\b|\bsu\b(\s|$)|chmod\s+([0-7]{3,4}|[ugoa]*[-+=][rwxXstugo]+)\b|\bchown\b|\bchgrp\b`)

	forcePushPattern = regexp.MustCompile(`(?i)\bgit\s+push\b.*(\s--force(-with-lease|-if-includes)?(=|\b)|\s-f\b|\s\+[^\s]+)`)

	networkSysPattern = regexp.MustCompile(`(?i)\b(iptables|ip6tables|nft|ufw|firewall-cmd|sysctl|nmcli|ifconfig|netsh)\b|\bip\s+(link|route|addr|neigh)\b|\broute\s+(add|delete|del|change)\b`)

	networkCLIPattern = regexp.MustCompile(`(?i)\b(curl|wget|Invoke-WebRequest|Invoke-RestMethod)\b`)

	packageRiskPattern = regexp.MustCompile(`(?i)\b(npm|pnpm|yarn)\s+publish\b|\bpip\s+install\b[^\n]*--break-system`)

	// Path-shaped sensitive names (file write / path args).
	sensitivePathPattern = regexp.MustCompile(`(?i)(^|[/\\])(\.env($|\.[^/\\]+)|(id_rsa|id_ed25519)(\.[^/\\]+)?$|[^/\\]+\.(pem|key|p12|pfx|jks)$|credentials\.json$|\.aws[/\\]credentials$|\.npmrc$|\.netrc$|kubeconfig$|service-account\.json$|(secret|secrets)($|[/\\]))`)

	// Command-string sensitive refs (allow whitespace / quote boundaries before tokens).
	sensitiveCommandPattern = regexp.MustCompile(`(?i)(^|[\s="'])(\.env($|[\s."']|\.[^\s"']+)|(id_rsa|id_ed25519)(\.[^\s"']*)?|[^\s"']+\.(pem|key|p12|pfx|jks)\b|credentials\.json|\.aws[/\\]credentials|\.npmrc\b|\.netrc\b|kubeconfig\b|service-account\.json|(secret|secrets)($|[\s/\\]))`)

	dangerousCodePattern = regexp.MustCompile(`(?i)\bos\.(remove|rmdir|system|popen)\b|\bsubprocess\.|\bshutil\.(rmtree|move)\b|\beval\s*\(|\bexec\s*\(|\b__import__\s*\(|\bchild_process\b|\bfs\.(unlink|rmSync|rmdir|writeFileSync)\b|\bRuntime\.getRuntime\(\)\.exec\b|\bSocket\b|\brequests\.(put|post|delete)\b|\burllib\.request\b`)

	commandRules = []flagRule{
		{reason: "inline script execution", pattern: inlineScriptPattern},
		{reason: "destructive file or disk operation", pattern: destructivePattern},
		{reason: "privilege escalation or ownership change", pattern: privilegePattern},
		{reason: "force push", pattern: forcePushPattern},
		{reason: "network or system configuration change", pattern: networkSysPattern},
		{reason: "network side-effect command", pattern: networkCLIPattern},
		{reason: "package publish or break-system install", pattern: packageRiskPattern},
		{reason: "command references sensitive path", pattern: sensitiveCommandPattern},
	}
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
	cmd := argString(req.Args, "command")
	if req.Tool == permission.ToolRunCode {
		cmd = argString(req.Args, "code")
		if cmd == "" {
			cmd = argString(req.Args, "source")
		}
		if cmd == "" {
			return FlaggedResult{Flagged: true, Reason: "inline code execution"}
		}
		if dangerousCodePattern.MatchString(cmd) {
			return FlaggedResult{Flagged: true, Reason: "dangerous code API"}
		}
	}
	if inputs.Bash.HasChain || inputs.Bash.Complex {
		return FlaggedResult{Flagged: true, Reason: "complex or chained shell command"}
	}
	for _, rule := range commandRules {
		if rule.pattern.MatchString(cmd) {
			return FlaggedResult{Flagged: true, Reason: rule.reason}
		}
	}
	return FlaggedResult{}
}

func flagFileWrite(path string) FlaggedResult {
	path = strings.TrimSpace(path)
	if path == "" {
		return FlaggedResult{Flagged: true, Reason: "write without path"}
	}
	base := filepath.Base(path)
	if sensitivePathPattern.MatchString(path) || sensitivePathPattern.MatchString(base) {
		return FlaggedResult{Flagged: true, Reason: "sensitive file path"}
	}
	return FlaggedResult{}
}

func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
}

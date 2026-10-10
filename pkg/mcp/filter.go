package mcp

import "strings"

var denied = map[string]struct{}{
	"core.run_terminal": {},
	"core.run_code":     {},
	"core.http_request": {},
	"core.agent_run":    {},
	"gateway.run":       {},
	"gateway.cancel":    {},
}

// Denied reports whether capability.operation is never exposed on /mcp.
func Denied(capability, operation string) bool {
	key := strings.TrimSpace(capability) + "." + strings.TrimSpace(operation)
	_, ok := denied[key]
	return ok
}

// Match reports whether toolName survives include/exclude. An empty include
// list means all tools. Exclude always wins. A filter item matching the group
// (capability or MCP server name) or the full tool name is a hit. Callers
// that expose /mcp must also apply Denied; outbound MCP clients must not.
func Match(include, exclude []string, group, toolName string) bool {
	if matchesList(exclude, group, toolName) {
		return false
	}
	if len(nonzero(include)) == 0 {
		return true
	}
	return matchesList(include, group, toolName)
}

func matchesList(list []string, group, toolName string) bool {
	for _, item := range nonzero(list) {
		if item == toolName || item == group {
			return true
		}
	}
	return false
}

func nonzero(list []string) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

package validate

import "strings"

// IsListTool reports whether group.op is a paged list.
// Capability operations match when the name contains "list".
// Pipeline, workflow, and function list, plus hub apps, are paged too.
func IsListTool(group, op string) bool {
	switch group {
	case "pipeline", "workflow", "function":
		return op == "list"
	case "hub":
		return op == "apps"
	default:
		return strings.Contains(op, "list")
	}
}

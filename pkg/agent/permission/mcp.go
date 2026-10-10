package permission

import (
	"sync"
)

type mcpToolMeta struct {
	readOnly    bool
	destructive bool
}

var mcpTools sync.Map // map[string]mcpToolMeta

// SetMCPToolMeta records annotations for an outbound MCP tool name.
func SetMCPToolMeta(name string, readOnly, destructive bool) {
	if name == "" {
		return
	}
	mcpTools.Store(name, mcpToolMeta{readOnly: readOnly, destructive: destructive})
}

// ClearMCPToolMeta removes one registered MCP tool name.
func ClearMCPToolMeta(name string) {
	mcpTools.Delete(name)
}

// MCPToolReadOnly reports whether the MCP tool was advertised as read-only.
func MCPToolReadOnly(name string) bool {
	v, ok := mcpTools.Load(name)
	if !ok {
		return false
	}
	meta, ok := v.(mcpToolMeta)
	if !ok {
		return false
	}
	return meta.readOnly
}

// MCPToolDestructive reports whether the MCP tool was advertised as destructive.
func MCPToolDestructive(name string) bool {
	v, ok := mcpTools.Load(name)
	if !ok {
		return false
	}
	meta, ok := v.(mcpToolMeta)
	if !ok {
		return false
	}
	return meta.destructive
}

func mcpDefaultRuleSet() RuleSet {
	return RuleSet{
		Patterns: []PatternRule{
			{Pattern: "read", Action: ActionAllow},
			{Pattern: "write", Action: ActionAsk},
		},
		Default: ActionAsk,
	}
}

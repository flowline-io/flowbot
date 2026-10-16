package auth

import "strings"

// MCPReadScopes is the MCP wizard read-only preset.
// It is the hub read scopes used by MCP, every service:*:read scope, and the automate read scopes.
func MCPReadScopes() []string {
	out := []string{
		ScopeHubAppsRead,
		ScopeHubCapabilitiesRead,
		ScopeHubHealthRead,
	}
	for _, info := range AllScopes() {
		if strings.HasPrefix(info.Value, "service:") && strings.HasSuffix(info.Value, ":read") {
			out = append(out, info.Value)
		}
	}
	out = append(out,
		ScopePipelineRead,
		ScopeFunctionRead,
		ScopeWorkflowRead,
	)
	return out
}

// MCPRunScopes is the read preset plus pipeline, function, and workflow run.
// Those three run scopes also authorize apply and delete on the automate HTTP API.
func MCPRunScopes() []string {
	return append(MCPReadScopes(), ScopePipelineRun, ScopeFunctionRun, ScopeWorkflowRun)
}

// MCPWizardScopes returns scopes the MCP wizard may grant. admin:* is omitted.
func MCPWizardScopes() []ScopeInfo {
	all := AllScopes()
	out := make([]ScopeInfo, 0, len(all))
	for _, info := range all {
		if info.Value == ScopeAdmin {
			continue
		}
		out = append(out, info)
	}
	return out
}

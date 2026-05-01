package auth

const (
	ScopeAdmin = "admin:*"

	ScopeHubAppsRead          = "hub:apps:read"
	ScopeHubAppsStatus        = "hub:apps:status"
	ScopeHubAppsLogs          = "hub:apps:logs"
	ScopeHubAppsStart         = "hub:apps:start"
	ScopeHubAppsStop          = "hub:apps:stop"
	ScopeHubAppsRestart       = "hub:apps:restart"
	ScopeHubAppsPull          = "hub:apps:pull"
	ScopeHubAppsUpdate        = "hub:apps:update"
	ScopeHubCapabilitiesRead  = "hub:capabilities:read"
	ScopeHubHealthRead        = "hub:health:read"
	ScopeServiceBookmarkRead  = "service:bookmark:read"
	ScopeServiceBookmarkWrite = "service:bookmark:write"
	ScopeServiceArchiveRead   = "service:archive:read"
	ScopeServiceArchiveWrite  = "service:archive:write"
	ScopeServiceReaderRead    = "service:reader:read"
	ScopeServiceReaderWrite   = "service:reader:write"
	ScopeServiceKanbanRead    = "service:kanban:read"
	ScopeServiceKanbanWrite   = "service:kanban:write"
	ScopeServiceInfraRead     = "service:infra:read"
	ScopeServiceShellRead     = "service:shell-history:read"
	ScopePipelineRead         = "pipeline:read"
	ScopePipelineRun          = "pipeline:run"
	ScopeWorkflowRun          = "workflow:run"
)

func HasScope(scopes []string, required string) bool {
	if required == "" {
		return true
	}
	for _, item := range scopes {
		if item == required || item == ScopeAdmin {
			return true
		}
	}
	return false
}

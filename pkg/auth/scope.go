package auth

// ScopeInfo describes an available scope for token creation.
type ScopeInfo struct {
	Value       string
	Description string
}

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
	ScopeServiceExampleRead   = "service:example:read"
	ScopeServiceExampleWrite  = "service:example:write"
	ScopePipelineRead         = "pipeline:read"
	ScopePipelineRun          = "pipeline:run"
	ScopeWorkflowRun          = "workflow:run"
)

// AllScopes returns all scopes available for CLI token creation.
func AllScopes() []ScopeInfo {
	return []ScopeInfo{
		{Value: ScopeAdmin, Description: "full access"},
		{Value: ScopeHubAppsRead, Description: "read apps"},
		{Value: ScopeHubAppsStatus, Description: "app status"},
		{Value: ScopeHubAppsLogs, Description: "app logs"},
		{Value: ScopeHubAppsStart, Description: "start apps"},
		{Value: ScopeHubAppsStop, Description: "stop apps"},
		{Value: ScopeHubAppsRestart, Description: "restart apps"},
		{Value: ScopeHubAppsPull, Description: "pull apps"},
		{Value: ScopeHubAppsUpdate, Description: "update apps"},
		{Value: ScopeHubCapabilitiesRead, Description: "read capabilities"},
		{Value: ScopeHubHealthRead, Description: "read health"},
		{Value: ScopeServiceBookmarkRead, Description: "read bookmarks"},
		{Value: ScopeServiceBookmarkWrite, Description: "write bookmarks"},
		{Value: ScopeServiceArchiveRead, Description: "read archives"},
		{Value: ScopeServiceArchiveWrite, Description: "write archives"},
		{Value: ScopeServiceReaderRead, Description: "read feeds"},
		{Value: ScopeServiceReaderWrite, Description: "write feeds"},
		{Value: ScopeServiceKanbanRead, Description: "read kanban"},
		{Value: ScopeServiceKanbanWrite, Description: "write kanban"},
		{Value: ScopeServiceInfraRead, Description: "read infra"},
		{Value: ScopeServiceShellRead, Description: "read shell history"},
		{Value: ScopeServiceExampleRead, Description: "read example"},
		{Value: ScopeServiceExampleWrite, Description: "write example"},
		{Value: ScopePipelineRead, Description: "read pipelines"},
		{Value: ScopePipelineRun, Description: "run pipelines"},
		{Value: ScopeWorkflowRun, Description: "run workflows"},
	}
}

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

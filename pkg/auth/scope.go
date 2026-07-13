package auth

// ScopeInfo describes an available scope for token creation.
type ScopeInfo struct {
	Value       string
	Description string
}

const (
	ScopeAdmin = "admin:*"

	ScopeHubAppsRead         = "hub:apps:read"
	ScopeHubAppsStatus       = "hub:apps:status"
	ScopeHubAppsLogs         = "hub:apps:logs"
	ScopeHubAppsStart        = "hub:apps:start"
	ScopeHubAppsStop         = "hub:apps:stop"
	ScopeHubAppsRestart      = "hub:apps:restart"
	ScopeHubAppsPull         = "hub:apps:pull"
	ScopeHubAppsUpdate       = "hub:apps:update"
	ScopeHubCapabilitiesRead = "hub:capabilities:read"
	ScopeHubHealthRead       = "hub:health:read"

	ScopeServiceKarakeepRead  = "service:karakeep:read"
	ScopeServiceKarakeepWrite = "service:karakeep:write"
	ScopeServiceMinifluxRead  = "service:miniflux:read"
	ScopeServiceMinifluxWrite = "service:miniflux:write"
	ScopeServiceKanboardRead  = "service:kanboard:read"
	ScopeServiceKanboardWrite = "service:kanboard:write"
	ScopeServiceTriliumRead   = "service:trilium:read"
	ScopeServiceTriliumWrite  = "service:trilium:write"
	ScopeServiceMemosRead     = "service:memos:read"
	ScopeServiceMemosWrite    = "service:memos:write"
	ScopeServiceGiteaRead     = "service:gitea:read"
	ScopeServiceGiteaWrite    = "service:gitea:write"
	ScopeServiceGithubRead    = "service:github:read"
	ScopeServiceGithubWrite   = "service:github:write"
	ScopeServiceExampleRead   = "service:example:read"
	ScopeServiceExampleWrite  = "service:example:write"

	// Legacy Go constant aliases (same string as provider scopes). Prefer provider-scoped constants.
	ScopeServiceBookmarkRead  = ScopeServiceKarakeepRead
	ScopeServiceBookmarkWrite = ScopeServiceKarakeepWrite
	ScopeServiceReaderRead    = ScopeServiceMinifluxRead
	ScopeServiceReaderWrite   = ScopeServiceMinifluxWrite
	ScopeServiceKanbanRead    = ScopeServiceKanboardRead
	ScopeServiceKanbanWrite   = ScopeServiceKanboardWrite
	ScopeServiceNoteRead      = ScopeServiceTriliumRead
	ScopeServiceNoteWrite     = ScopeServiceTriliumWrite
	ScopeServiceMemoRead      = ScopeServiceMemosRead
	ScopeServiceMemoWrite     = ScopeServiceMemosWrite
	ScopeServiceForgeRead     = ScopeServiceGiteaRead
	ScopeServiceForgeWrite    = ScopeServiceGiteaWrite
	ScopeServiceArchiveRead   = "service:archive:read"
	ScopeServiceArchiveWrite  = "service:archive:write"
	ScopeServiceInfraRead     = "service:infra:read"
	ScopeServiceShellRead     = "service:shell-history:read"

	ScopePipelineRead  = "pipeline:read"
	ScopePipelineRun   = "pipeline:run"
	ScopeWorkflowRun   = "workflow:run"
	ScopeChatAgentChat = "chatagent:chat"
)

// legacyScopeStrings maps deprecated token scope strings to canonical provider scopes.
// Kept for this release so existing tokens with domain CapType names still authorize.
var legacyScopeStrings = map[string]string{
	"service:bookmark:read":  ScopeServiceKarakeepRead,
	"service:bookmark:write": ScopeServiceKarakeepWrite,
	"service:reader:read":    ScopeServiceMinifluxRead,
	"service:reader:write":   ScopeServiceMinifluxWrite,
	"service:kanban:read":    ScopeServiceKanboardRead,
	"service:kanban:write":   ScopeServiceKanboardWrite,
	"service:note:read":      ScopeServiceTriliumRead,
	"service:note:write":     ScopeServiceTriliumWrite,
	"service:memo:read":      ScopeServiceMemosRead,
	"service:memo:write":     ScopeServiceMemosWrite,
	"service:forge:read":     ScopeServiceGiteaRead,
	"service:forge:write":    ScopeServiceGiteaWrite,
}

// canonicalScope returns the provider-scoped form of a scope string.
func canonicalScope(scope string) string {
	if mapped, ok := legacyScopeStrings[scope]; ok {
		return mapped
	}
	return scope
}

// HasScope reports whether scopes includes the required scope or admin:*.
// Legacy domain scope strings (e.g. service:bookmark:read) match their provider equivalents.
func HasScope(scopes []string, required string) bool {
	if required == "" {
		return true
	}
	want := canonicalScope(required)
	for _, item := range scopes {
		if item == ScopeAdmin {
			return true
		}
		if canonicalScope(item) == want {
			return true
		}
	}
	return false
}

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
		{Value: ScopeServiceKarakeepRead, Description: "read karakeep"},
		{Value: ScopeServiceKarakeepWrite, Description: "write karakeep"},
		{Value: ScopeServiceMinifluxRead, Description: "read miniflux"},
		{Value: ScopeServiceMinifluxWrite, Description: "write miniflux"},
		{Value: ScopeServiceKanboardRead, Description: "read kanboard"},
		{Value: ScopeServiceKanboardWrite, Description: "write kanboard"},
		{Value: ScopeServiceTriliumRead, Description: "read trilium"},
		{Value: ScopeServiceTriliumWrite, Description: "write trilium"},
		{Value: ScopeServiceMemosRead, Description: "read memos"},
		{Value: ScopeServiceMemosWrite, Description: "write memos"},
		{Value: ScopeServiceGiteaRead, Description: "read gitea"},
		{Value: ScopeServiceGiteaWrite, Description: "write gitea"},
		{Value: ScopeServiceGithubRead, Description: "read github"},
		{Value: ScopeServiceGithubWrite, Description: "write github"},
		{Value: ScopeServiceExampleRead, Description: "read example"},
		{Value: ScopeServiceExampleWrite, Description: "write example"},
		{Value: ScopePipelineRead, Description: "read pipelines"},
		{Value: ScopePipelineRun, Description: "run pipelines"},
		{Value: ScopeWorkflowRun, Description: "run workflows"},
		{Value: ScopeChatAgentChat, Description: "chat agent"},
	}
}

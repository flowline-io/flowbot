package web

import "github.com/flowline-io/flowbot/internal/server/chatagent"

// webChatAgentService is the process-shared chatagent Service for Web hot-path APIs.
// Set via SetChatAgentService during server bootstrap.
var webChatAgentService *chatagent.Service

// SetChatAgentService installs the shared chatagent Service used by Web handlers.
func SetChatAgentService(s *chatagent.Service) {
	webChatAgentService = s
}

// ensureChatAgentService installs a Service when E2E/unit helpers mount Web
// without going through server.ChatAgentService bootstrap.
func ensureChatAgentService() {
	if webChatAgentService == nil {
		svc := chatagent.NewService()
		SetChatAgentService(svc)
		chatagent.BindSharedService(svc)
	}
}

// chatAgentService returns the shared Service installed by SetChatAgentService.
func chatAgentService() *chatagent.Service {
	if webChatAgentService == nil {
		panic("web: chatagent service unset; call SetChatAgentService (via server.ChatAgentService)")
	}
	return webChatAgentService
}

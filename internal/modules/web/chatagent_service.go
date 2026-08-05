package web

import (
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/types"
)

// webChatAgentService is the process-shared chatagent Service for Web hot-path APIs.
// Set via SetChatAgentService during server bootstrap.
var webChatAgentService *chatagent.Service

// errChatAgentServiceUnset is returned when handlers run before SetChatAgentService.
var errChatAgentServiceUnset = types.Errorf(types.ErrInternal, "chatagent service unset")

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
// Returns an error instead of panicking when unset (request handlers must not panic).
func chatAgentService() (*chatagent.Service, error) {
	if webChatAgentService == nil {
		return nil, errChatAgentServiceUnset
	}
	return webChatAgentService, nil
}

// pendingApprovalSessionCount returns pending approval count, or 0 if the service is unset.
func pendingApprovalSessionCount() int {
	svc, err := chatAgentService()
	if err != nil {
		return 0
	}
	return svc.CountPendingApprovalSessions()
}

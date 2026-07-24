package server

import (
	"sync"

	"github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

var (
	chatAgentSVCMu sync.Mutex
	chatAgentSVC   *chatagent.Service
)

// ChatAgentService returns the process-wide chatagent Service, creating and
// binding it once on first use.
func ChatAgentService() *chatagent.Service {
	chatAgentSVCMu.Lock()
	defer chatAgentSVCMu.Unlock()
	if chatAgentSVC == nil {
		installChatAgentServiceLocked(chatagent.NewService())
	}
	return chatAgentSVC
}

// installChatAgentService installs svc as the process-wide chatagent Service for
// REST, platform handler, pipeline/scheduled, and Web entry points.
func installChatAgentService(svc *chatagent.Service) {
	if svc == nil {
		return
	}
	chatAgentSVCMu.Lock()
	defer chatAgentSVCMu.Unlock()
	installChatAgentServiceLocked(svc)
}

func installChatAgentServiceLocked(svc *chatagent.Service) {
	chatAgentSVC = svc
	chatAgentService = svc
	chatagent.BindSharedService(svc)
	web.SetChatAgentService(svc)
}
